package parley

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"
)

// Config holds the engine's tunable behaviour. DefaultConfig gives the values
// proposed in the design document; every one is a starting point, not a
// measured optimum.
type Config struct {
	// Policy decides who and where Claude listens and answers. Required.
	Policy *Policy

	// SelfName labels Claude's own lines in transcripts (e.g. "Claude").
	SelfName string

	// SystemPrompt is passed through to the Brain on every turn.
	SystemPrompt string

	// QuietPeriod is how long a conversation must be quiet before Claude
	// answers. Every new message restarts the wait...
	QuietPeriod time.Duration

	// ...but never beyond MaxWait after the first message that wanted an
	// answer, so a busy channel cannot postpone Claude indefinitely.
	MaxWait time.Duration

	// Transcript budgets per conversation (see History).
	HistoryChars   int
	HistoryEntries int
	MaxEntryChars  int

	// Outgoing message splitting (see SplitMessage).
	MaxPieceChars int
	MaxPieces     int

	// ConvRepliesPerMinute caps Claude's replies in any one conversation.
	ConvRepliesPerMinute int

	// UserAddressesPerWindow caps how many direct addresses from one person
	// are answered within UserWindow, so one person cannot monopolise Claude.
	UserAddressesPerWindow int
	UserWindow             time.Duration

	// AmbientCooldown is the minimum time since Claude last spoke in a
	// conversation before an ambient (unaddressed) message may prompt a turn.
	AmbientCooldown time.Duration

	// Daily caps across all conversations.
	DailyMaxReplies int
	DailyMaxUSD     float64
	Location        *time.Location // time zone for "daily"; nil means UTC

	// Logf receives one line per notable event. Message content is never
	// logged. nil means log.Printf.
	Logf func(format string, args ...any)
}

// DefaultConfig returns the defaults proposed in the design document. The
// caller must still set Policy (and usually SystemPrompt).
func DefaultConfig() Config {
	return Config{
		SelfName:               "Claude",
		QuietPeriod:            3 * time.Second,
		MaxWait:                12 * time.Second,
		HistoryChars:           24000,
		HistoryEntries:         60,
		MaxEntryChars:          2000,
		MaxPieceChars:          DiscordMaxMessageChars,
		MaxPieces:              4,
		ConvRepliesPerMinute:   6,
		UserAddressesPerWindow: 10,
		UserWindow:             10 * time.Minute,
		AmbientCooldown:        10 * time.Minute,
		DailyMaxReplies:        300,
		DailyMaxUSD:            2.00,
	}
}

// Stats are simple counters for the owner's status command and for tests.
type Stats struct {
	Seen        int // messages classified as anything but Ignore
	Turns       int // Brain calls (including regenerations)
	Replies     int // turns that posted something
	Silent      int // turns where Claude chose to say nothing
	Regenerated int // turns redone because the conversation moved on
	Limited     int // addresses or turns withheld by a rate limit or budget
	Errors      int // Brain or Chat failures
}

// pending is one message waiting for Claude's response.
type pending struct {
	msg    Message
	direct bool
}

// conversation is the engine's state for one channel, thread or DM.
type conversation struct {
	key       ConvKey
	channelID string
	history   *History

	// Debounce state.
	timer        Timer     // pending quiet-period timer, or nil
	timerGen     int       // incremented on every re-arm; stale callbacks compare it
	firstPending time.Time // when the current wait started (for MaxWait)

	// Messages that want a response and have not been answered yet.
	triggers []pending

	// Generation state: at most one Brain call per conversation at a time.
	inFlight bool
	stale    bool // set when a message arrives during a generation

	lastReply time.Time // when Claude last posted here
}

// snapshot is the input to one generation, captured under the lock.
type snapshot struct {
	req Request
	n   int // how many triggers this turn answers
}

// Engine is the conversation engine. It is safe for concurrent use: the
// Discord adapter may call HandleMessage from any goroutine, and timer
// callbacks arrive on their own goroutines under RealClock.
type Engine struct {
	cfg   Config
	chat  Chat
	brain Brain
	clock Clock

	mu          sync.Mutex
	convs       map[ConvKey]*conversation
	convLimiter *WindowLimiter
	userLimiter *WindowLimiter
	daily       DailyBudget
	paused      bool
	stats       Stats
}

// New creates an Engine. clock may be nil, meaning RealClock.
func New(cfg Config, chat Chat, brain Brain, clock Clock) *Engine {
	if cfg.Policy == nil {
		panic("parley: Config.Policy is required")
	}
	if clock == nil {
		clock = RealClock{}
	}
	if cfg.Logf == nil {
		cfg.Logf = log.Printf
	}
	if cfg.SelfName == "" {
		cfg.SelfName = "Claude"
	}
	return &Engine{
		cfg:         cfg,
		chat:        chat,
		brain:       brain,
		clock:       clock,
		convs:       make(map[ConvKey]*conversation),
		convLimiter: NewWindowLimiter(cfg.ConvRepliesPerMinute, time.Minute),
		userLimiter: NewWindowLimiter(cfg.UserAddressesPerWindow, cfg.UserWindow),
		daily:       DailyBudget{MaxReplies: cfg.DailyMaxReplies, MaxUSD: cfg.DailyMaxUSD, Location: cfg.Location},
	}
}

// Stats returns a copy of the engine's counters.
func (e *Engine) Stats() Stats {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.stats
}

// Transcript returns the rendered transcript of a conversation (for tests,
// debugging and the simulator).
func (e *Engine) Transcript(key ConvKey) string {
	e.mu.Lock()
	defer e.mu.Unlock()
	if c := e.convs[key]; c != nil {
		return c.history.Render(e.cfg.SelfName)
	}
	return ""
}

// conv returns the conversation for a message, creating it if needed. The
// caller holds e.mu.
func (e *Engine) conv(m Message) *conversation {
	key := m.Key()
	c := e.convs[key]
	if c == nil {
		c = &conversation{
			key:       key,
			channelID: m.ChannelID,
			history:   NewHistory(e.cfg.HistoryChars, e.cfg.HistoryEntries, e.cfg.MaxEntryChars),
		}
		e.convs[key] = c
	}
	return c
}

// HandleMessage is the entry point for every new message the adapter sees.
func (e *Engine) HandleMessage(m Message) {
	e.mu.Lock()
	d := e.cfg.Policy.Classify(m)
	if d == Ignore {
		e.mu.Unlock()
		return
	}
	e.stats.Seen++
	now := e.clock.Now()
	if m.Time.IsZero() {
		m.Time = now
	}
	c := e.conv(m)
	c.history.Add(Entry{
		ID:         m.ID,
		AuthorID:   m.AuthorID,
		AuthorName: m.AuthorName,
		IsSelf:     m.AuthorID == e.cfg.Policy.BotUserID,
		Content:    m.Content,
		Time:       m.Time,
	})

	if d == OwnerCommand {
		answer := e.runCommandLocked(c, m)
		e.mu.Unlock()
		if _, err := e.chat.Send(context.Background(), m.ChannelID, answer, m.ID); err != nil {
			e.logf("command reply failed in %s: %v", c.key, err)
		}
		return
	}

	// Claude's own echoed messages need nothing more.
	if m.AuthorID == e.cfg.Policy.BotUserID {
		e.mu.Unlock()
		return
	}

	// Anything said while Claude is thinking makes the draft potentially
	// out of date.
	if c.inFlight {
		c.stale = true
	}

	wantsTurn := false
	switch d {
	case Direct:
		if e.userLimiter.Take(m.AuthorID, now) {
			c.triggers = append(c.triggers, pending{msg: m, direct: true})
			wantsTurn = true
		} else {
			e.stats.Limited++
			e.logf("per-user limit: not answering a message in %s", c.key)
		}
	case Ambient:
		if c.lastReply.IsZero() || now.Sub(c.lastReply) >= e.cfg.AmbientCooldown {
			c.triggers = append(c.triggers, pending{msg: m, direct: false})
			wantsTurn = true
		}
	}

	// A message that wants an answer starts (or extends) the quiet-period
	// wait; so does any message arriving while a wait is already running,
	// because the person may not have finished their thought.
	if wantsTurn || c.timer != nil {
		e.armLocked(c)
	}
	e.mu.Unlock()
}

// HandleEdit applies an edited message's new text to the transcript.
func (e *Engine) HandleEdit(m Message) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if c := e.convs[m.Key()]; c != nil {
		c.history.Edit(m.ID, m.Content)
	}
}

// HandleDelete removes a deleted message from the transcript.
func (e *Engine) HandleDelete(key ConvKey, messageID string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if c := e.convs[key]; c != nil {
		c.history.Delete(messageID)
	}
}

// armLocked (re)starts the quiet-period timer for c. The caller holds e.mu.
func (e *Engine) armLocked(c *conversation) {
	now := e.clock.Now()
	if c.timer == nil {
		c.firstPending = now
	} else {
		c.timer.Stop()
	}
	wait := e.cfg.QuietPeriod
	if deadline := c.firstPending.Add(e.cfg.MaxWait); now.Add(wait).After(deadline) {
		wait = deadline.Sub(now)
		if wait < 0 {
			wait = 0
		}
	}
	c.timerGen++
	gen, key := c.timerGen, c.key
	c.timer = e.clock.AfterFunc(wait, func() { e.onTimer(key, gen) })
}

// onTimer runs when a conversation's quiet period ends.
func (e *Engine) onTimer(key ConvKey, gen int) {
	e.mu.Lock()
	c := e.convs[key]
	if c == nil || gen != c.timerGen {
		// Superseded by a later re-arm (Stop raced with firing).
		e.mu.Unlock()
		return
	}
	c.timer = nil
	if c.inFlight {
		// A generation is running; it will notice and regenerate, or re-arm
		// afterwards for whatever it did not cover.
		c.stale = true
		e.mu.Unlock()
		return
	}
	snap, ok := e.startTurnLocked(c)
	e.mu.Unlock()
	if ok {
		e.runTurn(c, snap)
	}
}

// startTurnLocked applies the turn-level limits and, if the turn may proceed,
// marks the conversation in flight and captures its input. The caller holds
// e.mu.
func (e *Engine) startTurnLocked(c *conversation) (snapshot, bool) {
	if len(c.triggers) == 0 {
		return snapshot{}, false
	}
	now := e.clock.Now()
	reason := ""
	if e.paused {
		reason = "paused"
	} else if out, why := e.daily.Exhausted(now); out {
		reason = why
	} else if !e.convLimiter.Allow(string(c.key), now) {
		reason = "per-conversation limit"
	}
	if reason != "" {
		// Go quiet rather than post an error; the owner can ask why.
		e.stats.Limited++
		e.logf("%s: not answering %d message(s) in %s", reason, len(c.triggers), c.key)
		c.triggers = nil
		return snapshot{}, false
	}
	c.inFlight = true
	c.stale = false
	return e.snapshotLocked(c), true
}

// snapshotLocked captures the Brain request for c's current state. The
// caller holds e.mu.
func (e *Engine) snapshotLocked(c *conversation) snapshot {
	req := Request{
		Conv:         c.key,
		SystemPrompt: e.cfg.SystemPrompt,
		Transcript:   c.history.Render(e.cfg.SelfName),
		SelfName:     e.cfg.SelfName,
	}
	for _, p := range c.triggers {
		req.Triggers = append(req.Triggers, p.msg)
		if p.direct {
			req.Direct = true
		}
	}
	return snapshot{req: req, n: len(c.triggers)}
}

// runTurn performs one turn: typing indicator, Brain call (regenerating once
// if the conversation moved on meanwhile), then sending. It is called
// without e.mu held.
func (e *Engine) runTurn(c *conversation, snap snapshot) {
	ctx := context.Background()
	if err := e.chat.Typing(ctx, c.channelID); err != nil {
		e.logf("typing indicator failed in %s: %v", c.key, err)
	}

	for attempt := 0; ; attempt++ {
		reply, err := e.brain.Respond(ctx, snap.req)

		e.mu.Lock()
		now := e.clock.Now()
		e.stats.Turns++
		e.daily.AddCost(now, reply.CostUSD)

		if err != nil {
			e.stats.Errors++
			e.logf("brain error in %s: %v", c.key, err)
			e.finishLocked(c, snap.n)
			e.mu.Unlock()
			return
		}

		// If people kept talking while Claude was thinking, the draft may
		// answer a conversation that has moved on. Redo it once with the
		// newer transcript; never more, so a busy channel cannot trap the
		// engine in a loop.
		if c.stale && attempt == 0 {
			c.stale = false
			e.stats.Regenerated++
			snap = e.snapshotLocked(c)
			e.mu.Unlock()
			continue
		}

		if reply.Silent || strings.TrimSpace(reply.Text) == "" {
			e.stats.Silent++
			e.finishLocked(c, snap.n)
			e.mu.Unlock()
			return
		}
		pieces := SplitMessage(ScrubMentions(reply.Text), e.cfg.MaxPieceChars, e.cfg.MaxPieces)
		e.mu.Unlock()

		// Send outside the lock: sending can be slow and must not block
		// incoming messages. The first piece is a Discord reply to the last
		// message that prompted the turn.
		replyTo := ""
		if len(snap.req.Triggers) > 0 {
			replyTo = snap.req.Triggers[len(snap.req.Triggers)-1].ID
		}
		var sentIDs []string
		var sentText []string
		for i, p := range pieces {
			to := ""
			if i == 0 {
				to = replyTo
			}
			id, err := e.chat.Send(ctx, c.channelID, p, to)
			if err != nil {
				e.mu.Lock()
				e.stats.Errors++
				e.mu.Unlock()
				e.logf("send failed in %s: %v", c.key, err)
				break
			}
			sentIDs = append(sentIDs, id)
			sentText = append(sentText, p)
		}

		e.mu.Lock()
		now = e.clock.Now()
		for i := range sentIDs {
			c.history.Add(Entry{ID: sentIDs[i], AuthorID: e.cfg.Policy.BotUserID, IsSelf: true, Content: sentText[i], Time: now})
		}
		if len(sentIDs) > 0 {
			c.lastReply = now
			e.convLimiter.Record(string(c.key), now)
			e.daily.AddReply(now)
			e.stats.Replies++
		}
		e.finishLocked(c, snap.n)
		e.mu.Unlock()
		return
	}
}

// finishLocked ends a turn: it drops the triggers the turn covered, clears
// the in-flight state, and schedules another turn if messages that want an
// answer arrived after the turn's snapshot. The caller holds e.mu.
func (e *Engine) finishLocked(c *conversation, answered int) {
	if answered > len(c.triggers) {
		answered = len(c.triggers) // "forget" may have cleared them meanwhile
	}
	c.triggers = c.triggers[answered:]
	c.inFlight = false
	c.stale = false
	if len(c.triggers) > 0 && c.timer == nil {
		e.armLocked(c)
	}
}

// runCommandLocked executes an owner command and returns the text to post.
// The caller holds e.mu.
func (e *Engine) runCommandLocked(c *conversation, m Message) string {
	args := e.cfg.Policy.CommandArgs(m.Content)
	sub := ""
	if len(args) > 0 {
		sub = strings.ToLower(args[0])
	}
	switch sub {
	case "status":
		replies, usd := e.daily.Today(e.clock.Now())
		state := "listening"
		if e.paused {
			state = "paused"
		}
		return fmt.Sprintf("Status: %s. Today: %d replies, about $%.2f spent. Since start: %d messages seen, %d replies, %d silent turns, %d withheld by limits, %d errors.",
			state, replies, usd, e.stats.Seen, e.stats.Replies, e.stats.Silent, e.stats.Limited, e.stats.Errors)
	case "pause":
		e.paused = true
		return "Paused. I'll keep listening but won't reply until you resume me."
	case "resume":
		e.paused = false
		return "Resumed."
	case "forget":
		c.history.Clear()
		c.triggers = nil
		return "I've forgotten this conversation's history."
	default:
		return "Owner commands: status, pause, resume, forget."
	}
}

// logf writes a log line via the configured logger.
func (e *Engine) logf(format string, args ...any) {
	e.cfg.Logf(format, args...)
}
