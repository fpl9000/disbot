// Package parley is the platform-independent conversation engine for a
// Discord bot through which Claude takes part in a channel the way a person
// would: it listens to everyone, speaks when it is addressed (and, optionally,
// when it has something worth adding), waits out bursts of messages so that
// several people typing at once get one considered answer, and keeps a
// separate, speaker-labelled transcript for every channel, thread and DM.
//
// The package deliberately has no dependency on Discord, on the network, or on
// the wall clock:
//
//   - Discord is reached only through the small Chat interface (transport.go).
//     A real implementation will wrap discordgo; tests use FakeChat.
//   - Claude is reached only through the Brain interface (brain.go). A real
//     implementation will call the Claude Messages API; tests use BrainFunc.
//   - Time is reached only through the Clock interface (clock.go). Tests use
//     FakeClock, which makes debounce and rate-limit behaviour deterministic.
//
// This is the prototype that accompanies the design document "Claude in
// Discord: design for a conversational bot". It does not change the existing
// disbot program in any way: disbot is package main in the repository root and
// does not import this package.
//
// The pieces, in the order a message flows through them:
//
//   - policy.go decides what kind of message this is (ignore, record only,
//     a direct address to Claude, an ambient message, or an owner command).
//   - history.go keeps the bounded per-conversation transcript and renders it
//     with stable speaker tags, escaping anything that tries to forge a line.
//   - engine.go debounces, enforces one generation in flight per
//     conversation, regenerates once if the conversation moved on while Claude
//     was thinking, and applies the rate and budget limits from ratelimit.go.
//   - outgoing.go scrubs mass mentions and splits long replies into pieces
//     that fit Discord's per-message length limit.
//   - prompt.go builds the text that a real Brain sends to Claude.
package parley
