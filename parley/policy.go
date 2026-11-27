package parley

import (
	"regexp"
	"strings"
)

// Decision is what the engine should do with an incoming message.
type Decision int

const (
	// Ignore: not stored and not answered (other bots, channels outside the
	// allowlist, DMs from people who may not DM the bot).
	Ignore Decision = iota

	// RecordOnly: added to the transcript so Claude has context later, but
	// not answered. Claude's own messages and ordinary chatter in
	// non-ambient channels land here.
	RecordOnly

	// Direct: the message addresses Claude (a DM, an @mention, a reply to one
	// of Claude's messages, or Claude's name used as a word). Answered,
	// subject to rate limits.
	Direct

	// Ambient: ordinary chatter in a channel where ambient mode is on. A
	// candidate for an uninvited contribution, subject to a cooldown and to
	// Claude deciding it has something worth saying.
	Ambient

	// OwnerCommand: an administrative command from the owner (status, pause,
	// resume, forget). Handled by the engine, never sent to Claude.
	OwnerCommand
)

// String returns the decision's name, for logs and test failures.
func (d Decision) String() string {
	switch d {
	case Ignore:
		return "Ignore"
	case RecordOnly:
		return "RecordOnly"
	case Direct:
		return "Direct"
	case Ambient:
		return "Ambient"
	case OwnerCommand:
		return "OwnerCommand"
	}
	return "Decision(?)"
}

// Policy holds the static rules about who and where Claude listens and
// answers. All identities are user IDs, never display names, because display
// names can be copied by anyone.
type Policy struct {
	// BotUserID is the bot's own user ID; its messages are recorded, never
	// answered.
	BotUserID string

	// BotName is the name people use to address Claude in text (e.g.
	// "Claude"). Using it as a whole word counts as a direct address.
	BotName string

	// OwnerID is the user ID of the person who runs the bot. Only this user
	// may issue owner commands, and this user may always DM the bot.
	OwnerID string

	// AllowedGuilds restricts which servers the bot listens in. Empty means
	// any server, which is safe only because AllowedChannels is also checked.
	AllowedGuilds map[string]bool

	// AllowedChannels lists the channel IDs where Claude listens. A thread is
	// allowed when its parent channel is. Empty means no guild channels.
	AllowedChannels map[string]bool

	// DMAllowed lists users other than the owner who may DM the bot.
	DMAllowed map[string]bool

	// AllowedBots lists bot or webhook IDs that are treated like people.
	// Every other bot is ignored, which prevents two bots answering each
	// other forever.
	AllowedBots map[string]bool

	// AmbientChannels lists channels (or parent channels of threads) where
	// Claude may join in without being addressed. Empty means nowhere, the
	// recommended default.
	AmbientChannels map[string]bool

	// CommandPrefix starts an owner command, e.g. "!parley".
	CommandPrefix string

	// nameRE caches the compiled whole-word matcher for BotName.
	nameRE    *regexp.Regexp
	nameRESrc string
}

// Classify decides what to do with a message. The order of checks matters:
// identity first (self, other bots), then location (allowlists), then intent
// (commands, direct address, ambient).
func (p *Policy) Classify(m Message) Decision {
	// Claude's own messages, echoed back by the Gateway: remember, never answer.
	if p.BotUserID != "" && m.AuthorID == p.BotUserID {
		return RecordOnly
	}
	// Other bots and webhooks are ignored unless explicitly allowed.
	if m.AuthorIsBot && !p.AllowedBots[m.AuthorID] {
		return Ignore
	}

	// Where did it come from?
	if m.IsDM {
		if m.AuthorID != p.OwnerID && !p.DMAllowed[m.AuthorID] {
			return Ignore
		}
	} else {
		if len(p.AllowedGuilds) > 0 && !p.AllowedGuilds[m.GuildID] {
			return Ignore
		}
		if !p.AllowedChannels[m.ChannelID] && !(m.ParentID != "" && p.AllowedChannels[m.ParentID]) {
			return Ignore
		}
	}

	// Owner commands. The same text from anyone else is just conversation.
	if p.isCommand(m.Content) {
		if m.AuthorID == p.OwnerID {
			return OwnerCommand
		}
		return RecordOnly
	}

	// Direct address.
	if m.IsDM || m.MentionsBot || m.RepliesToBot || p.NamesBot(m.Content) {
		return Direct
	}

	// Ambient chatter, only where ambient mode is switched on.
	if p.AmbientChannels[m.ChannelID] || (m.ParentID != "" && p.AmbientChannels[m.ParentID]) {
		return Ambient
	}
	return RecordOnly
}

// isCommand reports whether text starts with the command prefix as a word.
func (p *Policy) isCommand(text string) bool {
	if p.CommandPrefix == "" {
		return false
	}
	t := strings.TrimSpace(text)
	if !strings.HasPrefix(t, p.CommandPrefix) {
		return false
	}
	rest := t[len(p.CommandPrefix):]
	return rest == "" || rest[0] == ' ' || rest[0] == '\t'
}

// NamesBot reports whether text uses BotName as a whole word, ignoring case:
// "hey Claude, ..." and "what do you think, claude?" match, "Claudette" does
// not.
func (p *Policy) NamesBot(text string) bool {
	if p.BotName == "" {
		return false
	}
	if p.nameRE == nil || p.nameRESrc != p.BotName {
		p.nameRE = regexp.MustCompile(`(?i)(^|[^\pL\pN_])` + regexp.QuoteMeta(p.BotName) + `($|[^\pL\pN_])`)
		p.nameRESrc = p.BotName
	}
	return p.nameRE.MatchString(text)
}

// CommandArgs returns the words after the command prefix, e.g. ["status"]
// for "!parley status".
func (p *Policy) CommandArgs(text string) []string {
	t := strings.TrimSpace(text)
	t = strings.TrimPrefix(t, p.CommandPrefix)
	return strings.Fields(t)
}
