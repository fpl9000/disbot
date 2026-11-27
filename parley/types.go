package parley

import "time"

// ConvKey identifies one conversation: a guild text channel, a thread, or a
// DM channel. Each conversation has its own transcript and its own turn state.
//
// Discord gives every thread its own channel ID (the thread "is" a channel
// whose parent is the text channel it was started in), so keying on the
// channel ID automatically gives each thread a transcript separate from its
// parent channel.
type ConvKey string

// Message is the engine's platform-neutral view of one incoming chat message.
// The Discord adapter fills it in from a discordgo MessageCreate event; tests
// and the simulator construct it directly.
type Message struct {
	// ID is the platform's message ID. It is used to de-duplicate (the bot's
	// own messages come back over the Gateway after it sends them), to apply
	// edits and deletions, and as the target of a reply.
	ID string

	// GuildID is empty for DMs.
	GuildID string

	// ChannelID is the channel the message was posted in. For a message in a
	// thread this is the thread's ID.
	ChannelID string

	// ParentID is the parent text channel's ID when ChannelID is a thread, and
	// empty otherwise. The allowlist is checked against both, so allowing a
	// channel also allows threads started inside it.
	ParentID string

	// IsDM is true for direct messages to the bot.
	IsDM bool

	// AuthorID is the author's stable user ID. Display names can change and
	// collide; IDs cannot, so every decision about who someone is uses this.
	AuthorID string

	// AuthorName is the author's display name at the time of the message. It
	// is used only to label transcript lines for Claude's benefit.
	AuthorName string

	// AuthorIsBot is true for bot accounts and webhooks.
	AuthorIsBot bool

	// Content is the message text. Without Discord's privileged Message
	// Content intent this would be empty for most guild messages, which is
	// why the design requires that intent.
	Content string

	// MentionsBot is true when the message @mentions the bot's user.
	MentionsBot bool

	// RepliesToBot is true when the message is a Discord "reply" to one of
	// the bot's own messages.
	RepliesToBot bool

	// Time is when the message was posted.
	Time time.Time
}

// Key returns the conversation this message belongs to.
func (m Message) Key() ConvKey {
	if m.IsDM {
		return ConvKey("dm:" + m.ChannelID)
	}
	return ConvKey("ch:" + m.ChannelID)
}
