package parley

import (
	"regexp"
	"strings"
)

// DiscordMaxMessageChars is Discord's per-message length limit for bots
// (2,000 characters). It is a long-standing, widely documented limit, but it
// was not re-checked against Discord's documentation while writing this.
const DiscordMaxMessageChars = 2000

// roleMentionRE matches Discord's raw role-mention syntax, <@&123...>.
var roleMentionRE = regexp.MustCompile(`<@&\d+>`)

// ScrubMentions defuses mass and role mentions in text Claude produced, so a
// reply can never notify a whole server even if someone talks Claude into
// writing "@everyone". The adapter also sends with allowed_mentions
// restricted, so this is a second, independent layer: it changes what people
// see as well as what Discord does.
func ScrubMentions(text string) string {
	// A zero-width space after "@" stops Discord treating it as a mention
	// while leaving it readable.
	text = strings.ReplaceAll(text, "@everyone", "@​everyone")
	text = strings.ReplaceAll(text, "@here", "@​here")
	return roleMentionRE.ReplaceAllString(text, "@role")
}

// SplitMessage breaks text into pieces of at most max characters each, so a
// long reply can be posted as several Discord messages. It prefers to cut, in
// order: at a blank line outside a code block, at a line break outside a code
// block, at any line break, at a space, and only as a last resort in the
// middle of a word.
//
// When a cut has to fall inside a fenced code block (```), the piece is
// closed with ``` and the next piece reopens the block with the same language
// tag, so both pieces render as code.
//
// maxPieces caps the number of pieces (<= 0 means no cap). An answer that
// needs more is cut short with a visible note rather than flooding the
// channel.
func SplitMessage(text string, max, maxPieces int) []string {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}
	if max < 64 {
		max = 64 // below this, fence bookkeeping would not fit
	}

	var pieces []string
	runes := []rune(text)
	reopen := "" // fence opener to prepend to the next piece, e.g. "```go"

	for len(runes) > 0 {
		prefix := ""
		if reopen != "" {
			prefix = reopen + "\n"
		}
		budget := max - len([]rune(prefix))
		if len(runes) <= budget {
			pieces = append(pieces, prefix+string(runes))
			break
		}

		// Leave room for a closing "\n```" in case the cut lands in a code block.
		cut := chooseCut(runes, budget-4, reopen != "")
		piece := prefix + strings.TrimRight(string(runes[:cut]), " \n")
		rest := runes[cut:]

		// Drop the separator we cut at, but keep leading spaces on a line,
		// which may be meaningful indentation in code.
		for len(rest) > 0 && rest[0] == '\n' {
			rest = rest[1:]
		}
		if len(rest) > 0 && rest[0] == ' ' && cut > 0 && runes[cut-1] != '\n' {
			rest = rest[1:]
		}

		if open, lang := fenceState(piece); open {
			piece += "\n```"
			reopen = "```" + lang
		} else {
			reopen = ""
		}
		pieces = append(pieces, piece)
		runes = rest
	}

	if maxPieces > 0 && len(pieces) > maxPieces {
		pieces = pieces[:maxPieces]
		pieces[maxPieces-1] = cutShort(pieces[maxPieces-1], max)
	}
	return pieces
}

// cutShort ends a piece with a note that the reply was truncated, keeping the
// piece within max characters and its code fences balanced.
func cutShort(piece string, max int) string {
	const note = "\n…(reply cut short)"
	r := []rune(strings.TrimSuffix(piece, "\n```"))
	keep := max - len([]rune(note)) - 4
	if len(r) > keep {
		r = r[:keep]
	}
	out := strings.TrimRight(string(r), " \n")
	if open, _ := fenceState(out); open {
		out += "\n```"
	}
	return out + note
}

// chooseCut returns how many runes of r to put in the next piece, at most
// limit. inFence says whether r starts inside a code block (because the
// previous piece ended inside one).
func chooseCut(r []rune, limit int, inFence bool) int {
	if limit >= len(r) {
		return len(r)
	}
	bestPara, bestLine, bestAnyLine, bestSpace := -1, -1, -1, -1
	fence := inFence
	for i := 0; i < limit; i++ {
		// A line that starts with ``` toggles the code-block state.
		if i == 0 || r[i-1] == '\n' {
			j := i
			for j < len(r) && (r[j] == ' ' || r[j] == '\t') {
				j++
			}
			if j+3 <= len(r) && string(r[j:j+3]) == "```" {
				fence = !fence
			}
		}
		if i == 0 {
			continue // a cut at 0 would make an empty piece
		}
		switch r[i] {
		case '\n':
			bestAnyLine = i
			if !fence {
				bestLine = i
				if r[i-1] == '\n' {
					bestPara = i - 1
				}
			}
		case ' ':
			bestSpace = i
		}
	}
	for _, c := range []int{bestPara, bestLine, bestAnyLine, bestSpace} {
		if c > 0 {
			return c
		}
	}
	return limit
}

// fenceState reports whether s ends inside an unclosed ``` code block and, if
// so, the language tag written after the opening fence (e.g. "go").
func fenceState(s string) (open bool, lang string) {
	for _, line := range strings.Split(s, "\n") {
		t := strings.TrimLeft(line, " \t")
		if strings.HasPrefix(t, "```") {
			if open {
				open, lang = false, ""
			} else {
				open, lang = true, strings.TrimSpace(t[3:])
			}
		}
	}
	return open, lang
}
