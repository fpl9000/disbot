package parley

import (
	"strings"
	"testing"
)

func TestParseModelOutput(t *testing.T) {
	if r := ParseModelOutput("  <silent/>\n", "Claude"); !r.Silent {
		t.Fatal("silent token not recognised")
	}
	if r := ParseModelOutput("", "Claude"); !r.Silent {
		t.Fatal("empty output should be silent")
	}
	if r := ParseModelOutput("[Claude]: Sure, Sam!", "Claude"); r.Silent || r.Text != "Sure, Sam!" {
		t.Fatalf("self label not stripped: %+v", r)
	}
}

func TestRenderUserTurn(t *testing.T) {
	req := Request{
		Transcript: "[Sam #abcd]: hey Claude",
		SelfName:   "Claude",
		Direct:     true,
		Triggers:   []Message{{AuthorID: "sam", AuthorName: "Sam", Content: "hey Claude"}},
	}
	out := RenderUserTurn(req)
	for _, want := range []string{"<transcript>", "[Sam #abcd]: hey Claude", "addressed you", SilentToken} {
		if !strings.Contains(out, want) {
			t.Errorf("user turn lacks %q:\n%s", want, out)
		}
	}
	req.Direct = false
	if !strings.Contains(RenderUserTurn(req), "Nobody addressed you") {
		t.Error("ambient turn not described as ambient")
	}
}

func TestDefaultSystemPromptStatesIdentity(t *testing.T) {
	p := DefaultSystemPrompt("Claude", "Fran", "Meets Thursdays.")
	for _, want := range []string{"AI model made by Anthropic", "Never claim or imply to be human", "Meets Thursdays.", SilentToken} {
		if !strings.Contains(p, want) {
			t.Errorf("system prompt lacks %q", want)
		}
	}
}
