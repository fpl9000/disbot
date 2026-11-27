package parley

import (
	"strings"
	"testing"
)

func TestScrubMentions(t *testing.T) {
	got := ScrubMentions("ping @everyone and @here and <@&123456>, but <@42> is fine")
	if strings.Contains(got, "@everyone") || strings.Contains(got, "@here") || strings.Contains(got, "<@&") {
		t.Fatalf("mass mention survived: %q", got)
	}
	if !strings.Contains(got, "<@42>") {
		t.Fatalf("user mention was altered: %q", got)
	}
}

func TestSplitShortMessageIsOnePiece(t *testing.T) {
	if got := SplitMessage("  hello  ", 2000, 4); len(got) != 1 || got[0] != "hello" {
		t.Fatalf("got %q", got)
	}
	if got := SplitMessage("   ", 2000, 4); got != nil {
		t.Fatalf("blank text should give no pieces, got %q", got)
	}
}

// Every piece fits, nothing is lost, and cuts prefer paragraph breaks.
func TestSplitPrefersParagraphs(t *testing.T) {
	para := strings.Repeat("word ", 30) // 150 chars
	text := strings.TrimSpace(strings.Repeat(para+"\n\n", 6))
	pieces := SplitMessage(text, 400, 0)
	if len(pieces) < 2 {
		t.Fatalf("expected several pieces, got %d", len(pieces))
	}
	for i, p := range pieces {
		if n := len([]rune(p)); n > 400 {
			t.Errorf("piece %d has %d chars", i, n)
		}
		if strings.HasPrefix(p, "\n") || strings.HasSuffix(p, "\n") {
			t.Errorf("piece %d has stray newlines", i)
		}
	}
	joined := strings.Join(pieces, "\n\n")
	if strings.Join(strings.Fields(joined), " ") != strings.Join(strings.Fields(text), " ") {
		t.Fatal("text was lost or reordered")
	}
}

// A code block cut across pieces is closed and reopened with its language.
func TestSplitReopensCodeFence(t *testing.T) {
	var b strings.Builder
	b.WriteString("Here is the code:\n```go\n")
	for i := 0; i < 40; i++ {
		b.WriteString("fmt.Println(\"line number something\")\n")
	}
	b.WriteString("```\nDone.")
	pieces := SplitMessage(b.String(), 500, 0)
	if len(pieces) < 2 {
		t.Fatalf("expected several pieces, got %d", len(pieces))
	}
	for i, p := range pieces {
		if n := len([]rune(p)); n > 500 {
			t.Errorf("piece %d has %d chars", i, n)
		}
		if open, _ := fenceState(p); open {
			t.Errorf("piece %d leaves a code block open:\n%s", i, p)
		}
		if i > 0 && i < len(pieces)-1 && !strings.HasPrefix(p, "```go\n") {
			t.Errorf("middle piece %d does not reopen the go block:\n%s", i, p)
		}
	}
}

// With a piece cap, the last piece says the reply was cut short and still fits.
func TestSplitCapsPieces(t *testing.T) {
	text := strings.Repeat("sentence after sentence. ", 400) // ~10,000 chars
	pieces := SplitMessage(text, 2000, 2)
	if len(pieces) != 2 {
		t.Fatalf("want 2 pieces, got %d", len(pieces))
	}
	last := pieces[1]
	if !strings.HasSuffix(last, "(reply cut short)") {
		t.Fatalf("no truncation note: ...%q", last[len(last)-40:])
	}
	if n := len([]rune(last)); n > 2000 {
		t.Fatalf("last piece has %d chars", n)
	}
}

// A single unbroken word longer than the limit is still split.
func TestSplitHardCut(t *testing.T) {
	pieces := SplitMessage(strings.Repeat("x", 250), 100, 0)
	total := 0
	for _, p := range pieces {
		if len(p) > 100 {
			t.Fatalf("piece too long: %d", len(p))
		}
		total += len(p)
	}
	if total != 250 {
		t.Fatalf("lost characters: %d", total)
	}
}
