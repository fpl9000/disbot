# parley: conversation engine prototype

This directory is a **prototype**, not a working bot. It holds the Discord-free heart of a proposed second program in this repository, through which Claude would take part in a Discord channel the way a person does: listening to everyone, answering when addressed, waiting out bursts so that several people typing at once get one considered answer, and keeping a separate, speaker-labelled transcript for each channel, thread and DM. The full design, including the parts not built yet, is in the design document linked from the pull request that added this directory.

Nothing here changes `disbot`. disbot is `package main` in the repository root and does not import this package; building it produces the same program as before.

## What is here

| File | What it does |
|---|---|
| `policy.go` | Decides, per message, whether to ignore it, just remember it, treat it as a direct address to Claude, treat it as ambient chatter, or run an owner command. Identity is always by user ID, never by display name. |
| `history.go` | The bounded per-conversation transcript. Renders lines as `[Name #tag]: text`, where the tag comes from the user ID, and escapes multi-line messages so nobody can forge a line from someone else or from Claude. |
| `engine.go` | Debounce (quiet period plus a maximum wait), one generation in flight per conversation, regenerate-once when the conversation moves on mid-generation, per-conversation and per-user rate limits, daily reply and spending caps, owner commands. |
| `outgoing.go` | Defuses `@everyone`, `@here` and role mentions, and splits long replies into pieces under Discord's 2,000-character limit, closing and reopening code blocks across pieces. |
| `prompt.go` | A draft system prompt, the user-turn text a real brain would send to the Claude API, and parsing of Claude's "stay silent" answer. |
| `clock.go`, `transport.go`, `brain.go` | The three seams: time, the chat platform, and Claude. Each has a fake for tests. |
| `../cmd/parley-sim` | A terminal simulator: play several people in a pretend channel and see when Claude would answer. Uses a stand-in brain, so it costs nothing and needs no network. |

## Trying it

```
go test ./parley/
go run ./cmd/parley-sim
```

In the simulator, type lines such as `sam: hey Claude, any ideas for dinner?` or `alex: @claude me too`. The person called `fran` is the owner and may type `!parley status`, `!parley pause`, `!parley resume` or `!parley forget`. Type `/transcript` to see what Claude would see and `/quit` to leave.

## Not built yet

The Discord adapter (discordgo, Gateway intents, sending with restricted `allowed_mentions`, 429 handling) and the real brain (the Claude Messages API through the official Go SDK). Both need network access that was not available when this was written, and both wait for the owner's answers to the open questions in the design document.
