// Command parley-sim is a terminal simulator for the parley conversation
// engine. It lets you play several people in one pretend Discord channel and
// watch when and how "Claude" would answer, without Discord and without
// calling the Claude API: the brain here is a stand-in that only describes
// what it was asked, so the simulator costs nothing to run.
//
// Usage:
//
//	go run ./cmd/parley-sim
//
// Then type lines of the form "name: message", for example:
//
//	sam: anyone around?
//	sam: Claude, what's a good name for a cat?
//	alex: @claude and for a dog?
//
// "@claude" or the word "Claude" addresses the bot. The person named "fran"
// is the owner and may use "!parley status|pause|resume|forget". Type
// "/transcript" to see what Claude would see, and "/quit" to exit.
package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"disbot/parley"
)

func main() {
	policy := &parley.Policy{
		BotUserID:       "bot",
		BotName:         "Claude",
		OwnerID:         "user-fran",
		AllowedChannels: map[string]bool{"sim": true},
		CommandPrefix:   "!parley",
	}

	cfg := parley.DefaultConfig()
	cfg.Policy = policy
	cfg.SystemPrompt = parley.DefaultSystemPrompt("Claude", "Fran", "")
	cfg.QuietPeriod = 2 * time.Second // a little shorter than production, for a snappier demo
	cfg.Logf = func(format string, args ...any) { fmt.Printf("  (log) "+format+"\n", args...) }

	chat := &parley.FakeChat{OnSend: func(m parley.SentMessage) {
		fmt.Printf("\n[Claude] %s\n> ", m.Text)
	}}

	// The stand-in brain: it reports what a real brain would have been asked.
	brain := parley.BrainFunc(func(_ context.Context, req parley.Request) (parley.Reply, error) {
		time.Sleep(700 * time.Millisecond) // pretend to think
		var who []string
		for _, t := range req.Triggers {
			who = append(who, t.AuthorName)
		}
		lines := strings.Count(req.Transcript, "\n") + 1
		kind := "addressed by"
		if !req.Direct {
			kind = "ambient, prompted by"
		}
		return parley.Reply{Text: fmt.Sprintf("(stand-in reply) I'd answer %s, %s, having read %d transcript lines.",
			strings.Join(who, " and "), kind, lines)}, nil
	})

	eng := parley.New(cfg, chat, brain, nil)

	fmt.Println(`parley simulator. Type "name: message" (e.g. "sam: hey Claude"), "/transcript", or "/quit".`)
	var nextID atomic.Int64
	in := bufio.NewScanner(os.Stdin)
	fmt.Print("> ")
	for in.Scan() {
		line := strings.TrimSpace(in.Text())
		switch {
		case line == "":
		case line == "/quit":
			return
		case line == "/transcript":
			fmt.Println(eng.Transcript("ch:sim"))
		default:
			name, text, ok := strings.Cut(line, ":")
			if !ok {
				fmt.Println(`Please type "name: message".`)
				break
			}
			name, text = strings.TrimSpace(name), strings.TrimSpace(text)
			mention := strings.Contains(strings.ToLower(text), "@claude")
			eng.HandleMessage(parley.Message{
				ID:          fmt.Sprintf("m%d", nextID.Add(1)),
				ChannelID:   "sim",
				AuthorID:    "user-" + strings.ToLower(name),
				AuthorName:  name,
				Content:     text,
				MentionsBot: mention,
			})
		}
		fmt.Print("> ")
	}
}
