package parley

import "context"

// Request is what the engine asks the Brain to respond to.
type Request struct {
	// Conv identifies the conversation, e.g. for per-conversation logging or
	// prompt-cache keys.
	Conv ConvKey

	// SystemPrompt is the persona and house rules. A real Brain sends it as
	// the system prompt and marks it for prompt caching, because it is the
	// same for every turn.
	SystemPrompt string

	// Transcript is the rendered, speaker-labelled recent history of the
	// conversation (see History.Render), oldest line first.
	Transcript string

	// SelfName is how Claude's own lines are labelled in Transcript.
	SelfName string

	// Triggers are the messages that caused this turn, oldest first. There
	// may be several when a burst was debounced into one turn, possibly from
	// different people. The reply is sent as a Discord reply to the last one.
	Triggers []Message

	// Direct is true when at least one trigger addressed Claude directly.
	// When false the turn is purely ambient and staying silent is the
	// expected, usual outcome.
	Direct bool
}

// Reply is the Brain's answer.
type Reply struct {
	// Text is the reply to post. It is ignored when Silent is true.
	Text string

	// Silent means Claude decided that saying nothing is the right response
	// (for example to a "thanks" at the end of an exchange, or to an ambient
	// message it has nothing useful to add to).
	Silent bool

	// CostUSD is the estimated cost of producing this reply, computed by the
	// Brain from the token usage the API reports and the model's prices. The
	// engine adds it to the daily budget whether or not anything was sent.
	CostUSD float64
}

// Brain produces Claude's side of a turn. The production implementation will
// call the Claude Messages API (via the official Go SDK) with the system
// prompt cached, adaptive thinking at low effort, and the web search tool, and
// will use RenderUserTurn and ParseModelOutput from prompt.go.
type Brain interface {
	Respond(ctx context.Context, req Request) (Reply, error)
}

// BrainFunc adapts an ordinary function to the Brain interface. Tests use it
// to script Claude's answers and to inject events mid-generation.
type BrainFunc func(ctx context.Context, req Request) (Reply, error)

// Respond calls f.
func (f BrainFunc) Respond(ctx context.Context, req Request) (Reply, error) { return f(ctx, req) }
