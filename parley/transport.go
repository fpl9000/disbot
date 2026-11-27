package parley

import (
	"context"
	"fmt"
	"sync"
)

// Chat is everything the engine needs from the chat platform. The production
// implementation will be a thin wrapper around discordgo that:
//
//   - sends with allowed_mentions restricted so a reply can never ping
//     @everyone, @here or a role, whatever text Claude produced;
//   - serialises sends per channel and honours HTTP 429 retry_after values;
//   - refreshes the typing indicator while a generation is running.
//
// Keeping this interface tiny is what lets the whole engine be tested without
// Discord.
type Chat interface {
	// Send posts text to a channel. If replyTo is non-empty, the message is
	// sent as a Discord reply to that message ID. It returns the new
	// message's ID.
	Send(ctx context.Context, channelID, text, replyTo string) (string, error)

	// Typing shows the "Claude is typing" indicator in a channel.
	Typing(ctx context.Context, channelID string) error
}

// SentMessage records one call to FakeChat.Send.
type SentMessage struct {
	ID        string
	ChannelID string
	Text      string
	ReplyTo   string
}

// FakeChat is an in-memory Chat for tests and the simulator. It records what
// would have been sent instead of sending it.
type FakeChat struct {
	mu     sync.Mutex
	sent   []SentMessage
	typing map[string]int
	nextID int

	// OnSend, if set, is called after each message is recorded. The
	// simulator uses it to print replies as they are "sent".
	OnSend func(SentMessage)
}

// Send records the message and returns a synthetic ID ("sent-1", "sent-2", ...).
func (f *FakeChat) Send(_ context.Context, channelID, text, replyTo string) (string, error) {
	f.mu.Lock()
	f.nextID++
	msg := SentMessage{ID: fmt.Sprintf("sent-%d", f.nextID), ChannelID: channelID, Text: text, ReplyTo: replyTo}
	f.sent = append(f.sent, msg)
	hook := f.OnSend
	f.mu.Unlock()
	if hook != nil {
		hook(msg)
	}
	return msg.ID, nil
}

// Typing counts typing indications per channel.
func (f *FakeChat) Typing(_ context.Context, channelID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.typing == nil {
		f.typing = make(map[string]int)
	}
	f.typing[channelID]++
	return nil
}

// Sent returns a copy of everything sent so far, in order.
func (f *FakeChat) Sent() []SentMessage {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]SentMessage(nil), f.sent...)
}

// TypingCount returns how many typing indications a channel has received.
func (f *FakeChat) TypingCount(channelID string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.typing[channelID]
}
