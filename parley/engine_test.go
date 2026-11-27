package parley

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"
)

// harness wires an Engine to a FakeClock, a FakeChat and a scripted brain,
// and records every request the brain receives.
type harness struct {
	t     *testing.T
	clock *FakeClock
	chat  *FakeChat
	eng   *Engine

	mu       sync.Mutex
	requests []Request
	callTime []time.Time

	// script decides each reply; call counts from 1.
	script func(h *harness, req Request, call int) (Reply, error)
}

// newHarness builds a harness. mod may adjust the default config.
func newHarness(t *testing.T, script func(h *harness, req Request, call int) (Reply, error), mod func(*Config)) *harness {
	t.Helper()
	h := &harness{t: t, clock: NewFakeClock(t0), chat: &FakeChat{}, script: script}
	cfg := DefaultConfig()
	cfg.Policy = testPolicy()
	cfg.SystemPrompt = "test persona"
	cfg.Logf = func(string, ...any) {} // keep test output quiet
	if mod != nil {
		mod(&cfg)
	}
	brain := BrainFunc(func(_ context.Context, req Request) (Reply, error) {
		h.mu.Lock()
		h.requests = append(h.requests, req)
		h.callTime = append(h.callTime, h.clock.Now())
		n := len(h.requests)
		h.mu.Unlock()
		return h.script(h, req, n)
	})
	h.eng = New(cfg, h.chat, brain, h.clock)
	return h
}

// reply is a script that always answers with fixed text.
func reply(text string) func(*harness, Request, int) (Reply, error) {
	return func(*harness, Request, int) (Reply, error) { return Reply{Text: text, CostUSD: 0.01}, nil }
}

// msg builds a guild message in #general.
func msg(id, author, content string, mention bool) Message {
	return Message{ID: id, GuildID: "g1", ChannelID: "general", AuthorID: author, AuthorName: author, Content: content, MentionsBot: mention}
}

// calls returns a copy of the brain requests so far.
func (h *harness) calls() []Request {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]Request(nil), h.requests...)
}

func TestDirectAddressWaitsForQuietPeriod(t *testing.T) {
	h := newHarness(t, reply("Hi Sam!"), nil)
	h.eng.HandleMessage(msg("m1", "sam", "hey", true))

	h.clock.Advance(2900 * time.Millisecond)
	if n := len(h.chat.Sent()); n != 0 {
		t.Fatalf("answered before the quiet period ended (%d sent)", n)
	}
	h.clock.Advance(200 * time.Millisecond)
	sent := h.chat.Sent()
	if len(sent) != 1 || sent[0].Text != "Hi Sam!" || sent[0].ReplyTo != "m1" || sent[0].ChannelID != "general" {
		t.Fatalf("unexpected sends: %+v", sent)
	}
	if h.chat.TypingCount("general") != 1 {
		t.Fatal("typing indicator not shown")
	}
}

// Three messages in quick succession, from two people, get one answer that
// sees all of them.
func TestBurstIsAnsweredOnce(t *testing.T) {
	h := newHarness(t, reply("Sam, Alex: here you go."), nil)
	h.eng.HandleMessage(msg("m1", "sam", "Claude, what's the weather?", false))
	h.clock.Advance(time.Second)
	h.eng.HandleMessage(msg("m2", "sam", "in Blackstone", false))
	h.clock.Advance(time.Second)
	h.eng.HandleMessage(msg("m3", "alex", "and in Boston?", true))
	h.clock.Advance(10 * time.Second)

	calls := h.calls()
	if len(calls) != 1 {
		t.Fatalf("brain called %d times, want 1", len(calls))
	}
	if got := len(calls[0].Triggers); got != 2 {
		t.Errorf("turn covers %d triggers, want 2 (m1 and m3)", got)
	}
	for _, s := range []string{"what's the weather", "in Blackstone", "and in Boston"} {
		if !strings.Contains(calls[0].Transcript, s) {
			t.Errorf("transcript lacks %q", s)
		}
	}
	if sent := h.chat.Sent(); len(sent) != 1 || sent[0].ReplyTo != "m3" {
		t.Fatalf("unexpected sends: %+v", sent)
	}
}

// Continuous chatter cannot postpone the answer beyond MaxWait.
func TestMaxWaitCapsDebounce(t *testing.T) {
	h := newHarness(t, reply("ok"), nil)
	h.eng.HandleMessage(msg("m0", "sam", "question for you", true))
	for i := 1; i <= 10; i++ {
		h.clock.Advance(2 * time.Second)
		h.eng.HandleMessage(msg(fmt.Sprintf("c%d", i), "alex", "chatter", false))
	}
	h.clock.Advance(10 * time.Second)

	if len(h.callTime) == 0 {
		t.Fatal("never answered")
	}
	if got := h.callTime[0].Sub(t0); got != 12*time.Second {
		t.Fatalf("first answer after %v, want 12s", got)
	}
}

func TestAmbientIsOffByDefault(t *testing.T) {
	h := newHarness(t, reply("unwanted"), nil)
	h.eng.HandleMessage(msg("m1", "sam", "anyone seen my keys?", false))
	h.clock.Advance(time.Minute)
	if len(h.calls()) != 0 || len(h.chat.Sent()) != 0 {
		t.Fatal("answered unaddressed chatter in a non-ambient channel")
	}
	if !strings.Contains(h.eng.Transcript("ch:general"), "my keys") {
		t.Fatal("chatter was not recorded for context")
	}
}

// In an ambient channel Claude may speak, may choose silence, and observes a
// cooldown after it has spoken.
func TestAmbientCooldownAndSilence(t *testing.T) {
	h := newHarness(t, func(_ *harness, req Request, call int) (Reply, error) {
		if req.Direct {
			t.Error("ambient turn marked as direct")
		}
		if call == 2 {
			return Reply{Silent: true}, nil
		}
		return Reply{Text: "Fun fact!"}, nil
	}, nil)
	lounge := func(id string) Message {
		m := msg(id, "sam", "chatting", false)
		m.ChannelID = "lounge"
		return m
	}

	h.eng.HandleMessage(lounge("a1"))
	h.clock.Advance(5 * time.Second)
	if len(h.chat.Sent()) != 1 {
		t.Fatal("first ambient turn did not post")
	}

	h.eng.HandleMessage(lounge("a2")) // within the cooldown
	h.clock.Advance(time.Minute)
	if len(h.calls()) != 1 {
		t.Fatal("ambient turn taken during the cooldown")
	}

	h.clock.Advance(10 * time.Minute)
	h.eng.HandleMessage(lounge("a3")) // after the cooldown; brain chooses silence
	h.clock.Advance(5 * time.Second)
	if len(h.calls()) != 2 || len(h.chat.Sent()) != 1 {
		t.Fatalf("calls=%d sent=%d, want 2 and 1", len(h.calls()), len(h.chat.Sent()))
	}
	if s := h.eng.Stats(); s.Silent != 1 {
		t.Fatalf("silent turns = %d", s.Silent)
	}
}

// If someone speaks while Claude is generating, the draft is discarded and
// the turn regenerated once with the newer transcript; the newcomer's
// question is covered by that single reply.
func TestStaleTurnRegeneratesOnce(t *testing.T) {
	h := newHarness(t, func(h *harness, req Request, call int) (Reply, error) {
		if call == 1 {
			// Alex asks something while Claude is still thinking.
			h.eng.HandleMessage(msg("m2", "alex", "Claude, also: is it raining?", false))
			return Reply{Text: "draft"}, nil
		}
		return Reply{Text: "final"}, nil
	}, nil)
	h.eng.HandleMessage(msg("m1", "sam", "Claude, how's the weather?", false))
	h.clock.Advance(4 * time.Second)

	calls := h.calls()
	if len(calls) != 2 {
		t.Fatalf("brain called %d times, want 2", len(calls))
	}
	if !strings.Contains(calls[1].Transcript, "is it raining") || len(calls[1].Triggers) != 2 {
		t.Fatalf("regeneration did not include the new message: %+v", calls[1])
	}
	sent := h.chat.Sent()
	if len(sent) != 1 || sent[0].Text != "final" || sent[0].ReplyTo != "m2" {
		t.Fatalf("unexpected sends: %+v", sent)
	}

	// Alex's question was answered by that reply; no extra turn follows.
	h.clock.Advance(time.Minute)
	if len(h.calls()) != 2 {
		t.Fatalf("an extra turn ran: %d calls", len(h.calls()))
	}
	if s := h.eng.Stats(); s.Regenerated != 1 {
		t.Fatalf("regenerated = %d", s.Regenerated)
	}
}

// Messages that keep arriving cannot cause more than one regeneration.
func TestRegenerationIsBounded(t *testing.T) {
	h := newHarness(t, func(h *harness, req Request, call int) (Reply, error) {
		if call <= 2 {
			h.eng.HandleMessage(msg(fmt.Sprintf("x%d", call), "alex", "more chatter", false))
		}
		return Reply{Text: fmt.Sprintf("reply %d", call)}, nil
	}, nil)
	h.eng.HandleMessage(msg("m1", "sam", "hi", true))
	h.clock.Advance(4 * time.Second)
	if n := len(h.calls()); n != 2 {
		t.Fatalf("brain called %d times, want 2", n)
	}
	if sent := h.chat.Sent(); len(sent) != 1 || sent[0].Text != "reply 2" {
		t.Fatalf("unexpected sends: %+v", sent)
	}
}

func TestPerUserLimit(t *testing.T) {
	h := newHarness(t, reply("ok"), func(c *Config) { c.UserAddressesPerWindow = 2 })
	for i := 1; i <= 3; i++ {
		h.eng.HandleMessage(msg(fmt.Sprintf("m%d", i), "sam", "hey", true))
		h.clock.Advance(5 * time.Second)
	}
	if n := len(h.chat.Sent()); n != 2 {
		t.Fatalf("sent %d replies to one person, want 2", n)
	}
	// Someone else can still get an answer.
	h.eng.HandleMessage(msg("m4", "alex", "hey", true))
	h.clock.Advance(5 * time.Second)
	if n := len(h.chat.Sent()); n != 3 {
		t.Fatalf("other user not answered (sent=%d)", n)
	}
	if s := h.eng.Stats(); s.Limited != 1 {
		t.Fatalf("limited = %d", s.Limited)
	}
}

func TestPerConversationLimit(t *testing.T) {
	h := newHarness(t, reply("ok"), func(c *Config) { c.ConvRepliesPerMinute = 1 })
	h.eng.HandleMessage(msg("m1", "sam", "hey", true))
	h.clock.Advance(5 * time.Second)
	h.eng.HandleMessage(msg("m2", "alex", "hey", true))
	h.clock.Advance(5 * time.Second)
	if n := len(h.chat.Sent()); n != 1 {
		t.Fatalf("sent %d, want 1", n)
	}
	h.clock.Advance(time.Minute)
	h.eng.HandleMessage(msg("m3", "alex", "hey again", true))
	h.clock.Advance(5 * time.Second)
	if n := len(h.chat.Sent()); n != 2 {
		t.Fatalf("limit did not recover after a minute (sent=%d)", n)
	}
}

func TestDailySpendingCap(t *testing.T) {
	h := newHarness(t, func(*harness, Request, int) (Reply, error) {
		return Reply{Text: "ok", CostUSD: 0.03}, nil
	}, func(c *Config) { c.DailyMaxUSD = 0.05 })
	for i := 1; i <= 3; i++ {
		h.eng.HandleMessage(msg(fmt.Sprintf("m%d", i), fmt.Sprintf("user%d", i), "hey", true))
		h.clock.Advance(15 * time.Second)
	}
	if n := len(h.calls()); n != 2 {
		t.Fatalf("brain called %d times, want 2 (third over budget)", n)
	}
}

func TestOwnerCommands(t *testing.T) {
	h := newHarness(t, reply("ok"), nil)

	// Someone else cannot pause Claude.
	h.eng.HandleMessage(msg("c1", "sam", "!parley pause", false))
	h.eng.HandleMessage(msg("m1", "sam", "hey", true))
	h.clock.Advance(5 * time.Second)
	if n := len(h.calls()); n != 1 {
		t.Fatalf("non-owner command had an effect (calls=%d)", n)
	}

	// The owner can, and gets an immediate acknowledgement.
	h.eng.HandleMessage(msg("c2", "fran", "!parley pause", false))
	sent := h.chat.Sent()
	if last := sent[len(sent)-1]; !strings.HasPrefix(last.Text, "Paused") || last.ReplyTo != "c2" {
		t.Fatalf("pause not acknowledged: %+v", last)
	}
	h.eng.HandleMessage(msg("m2", "sam", "hey", true))
	h.clock.Advance(5 * time.Second)
	if n := len(h.calls()); n != 1 {
		t.Fatal("answered while paused")
	}

	h.eng.HandleMessage(msg("c3", "fran", "!parley resume", false))
	h.eng.HandleMessage(msg("m3", "sam", "hey", true))
	h.clock.Advance(5 * time.Second)
	if n := len(h.calls()); n != 2 {
		t.Fatal("not answering after resume")
	}

	h.eng.HandleMessage(msg("c4", "fran", "!parley status", false))
	sent = h.chat.Sent()
	if last := sent[len(sent)-1]; !strings.HasPrefix(last.Text, "Status: listening") {
		t.Fatalf("status: %q", last.Text)
	}

	h.eng.HandleMessage(msg("c5", "fran", "!parley forget", false))
	if tr := h.eng.Transcript("ch:general"); tr != "" {
		t.Fatalf("forget left a transcript: %q", tr)
	}
}

func TestOtherBotsAreIgnored(t *testing.T) {
	h := newHarness(t, reply("ok"), nil)
	m := msg("b1", "spambot", "hey Claude", true)
	m.AuthorIsBot = true
	h.eng.HandleMessage(m)
	h.clock.Advance(time.Minute)
	if len(h.calls()) != 0 || h.eng.Transcript("ch:general") != "" {
		t.Fatal("another bot's message was answered or stored")
	}
}

// A thread is its own conversation with its own transcript.
func TestThreadsAreSeparateConversations(t *testing.T) {
	h := newHarness(t, reply("ok"), nil)
	inThread := msg("t1m", "sam", "thread question", true)
	inThread.ChannelID, inThread.ParentID = "thread1", "general"
	h.eng.HandleMessage(inThread)
	h.eng.HandleMessage(msg("g1m", "alex", "channel question", true))
	h.clock.Advance(5 * time.Second)

	calls := h.calls()
	if len(calls) != 2 {
		t.Fatalf("brain called %d times, want 2", len(calls))
	}
	for _, c := range calls {
		if strings.Contains(c.Transcript, "thread question") && strings.Contains(c.Transcript, "channel question") {
			t.Fatal("thread and channel transcripts were mixed")
		}
	}
	channels := map[string]bool{}
	for _, s := range h.chat.Sent() {
		channels[s.ChannelID] = true
	}
	if !channels["thread1"] || !channels["general"] {
		t.Fatalf("replies went to %v", channels)
	}
}

// The Gateway echoes Claude's own messages back; they must not appear twice.
func TestOwnEchoIsNotDuplicated(t *testing.T) {
	h := newHarness(t, reply("Hello!"), nil)
	h.eng.HandleMessage(msg("m1", "sam", "hi", true))
	h.clock.Advance(5 * time.Second)
	echo := msg("sent-1", "bot", "Hello!", false)
	h.eng.HandleMessage(echo)
	if n := strings.Count(h.eng.Transcript("ch:general"), "[Claude]: Hello!"); n != 1 {
		t.Fatalf("own line appears %d times", n)
	}
}

func TestLongReplyIsSplitAndScrubbed(t *testing.T) {
	long := "@everyone listen up.\n\n" + strings.Repeat("All work and no play. ", 230) // ~5,000 chars
	h := newHarness(t, reply(long), nil)
	h.eng.HandleMessage(msg("m1", "sam", "tell me a story", true))
	h.clock.Advance(5 * time.Second)

	sent := h.chat.Sent()
	if len(sent) < 3 {
		t.Fatalf("expected at least 3 pieces, got %d", len(sent))
	}
	for i, s := range sent {
		if len([]rune(s.Text)) > DiscordMaxMessageChars {
			t.Errorf("piece %d too long", i)
		}
		if strings.Contains(s.Text, "@everyone") {
			t.Errorf("piece %d pings everyone", i)
		}
		if (i == 0) != (s.ReplyTo == "m1") {
			t.Errorf("piece %d replyTo=%q", i, s.ReplyTo)
		}
	}
}

// A failed Brain call must not leave the conversation stuck "in flight".
func TestBrainErrorDoesNotWedge(t *testing.T) {
	h := newHarness(t, func(_ *harness, _ Request, call int) (Reply, error) {
		if call == 1 {
			return Reply{}, errors.New("overloaded")
		}
		return Reply{Text: "back now"}, nil
	}, nil)
	h.eng.HandleMessage(msg("m1", "sam", "hi", true))
	h.clock.Advance(5 * time.Second)
	h.eng.HandleMessage(msg("m2", "sam", "hello?", true))
	h.clock.Advance(5 * time.Second)
	if sent := h.chat.Sent(); len(sent) != 1 || sent[0].Text != "back now" {
		t.Fatalf("unexpected sends: %+v", sent)
	}
	if s := h.eng.Stats(); s.Errors != 1 {
		t.Fatalf("errors = %d", s.Errors)
	}
}

func TestEditsAndDeletesReachTheTranscript(t *testing.T) {
	h := newHarness(t, reply("ok"), nil)
	h.eng.HandleMessage(msg("m1", "sam", "teh plan", false))
	h.eng.HandleMessage(msg("m2", "sam", "my password is hunter2", false))
	h.eng.HandleEdit(msg("m1", "sam", "the plan", false))
	h.eng.HandleDelete("ch:general", "m2")
	tr := h.eng.Transcript("ch:general")
	if !strings.Contains(tr, "the plan") || strings.Contains(tr, "teh") || strings.Contains(tr, "hunter2") {
		t.Fatalf("transcript = %q", tr)
	}
}
