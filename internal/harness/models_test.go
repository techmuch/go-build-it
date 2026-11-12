package harness

import (
	"context"
	"strings"
	"testing"
)

// FR-63a: each harness's list output becomes models; real output captured from agy 1.5.6 and pi 0.85.1.
func TestParseModelLists(t *testing.T) {
	agy := parseAgy("Fetching available models...\ngemini-4-argon-high\tGemini 4 Argon (High)\nclaude-opus-5-5-medium\tClaude Opus 5.5 (Medium)\n")
	if len(agy) != 2 || agy[1].ID != "claude-opus-5-5-medium" || agy[1].Name != "Claude Opus 5.5 (Medium)" {
		t.Fatalf("agy: %+v", agy)
	}
	pi := parsePi("provider    model                              context  max-out  thinking  images\nopenrouter  ~anthropic/claude-opus-latest      1M       128K     yes       yes\nopenrouter  ~deepseek/deepseek-pro-latest      1.0M     393.2K   yes       no\n")
	if len(pi) != 2 || pi[0].ID != "openrouter/~anthropic/claude-opus-latest" || pi[1].Detail != "context 1.0M · max output 393.2K · thinking" {
		t.Fatalf("pi: %+v", pi)
	}
	oc := parseLines("anthropic/claude-sonnet-5-5\nopenai/gpt-astra\n\nsome warning text\n")
	if len(oc) != 2 || oc[1].ID != "openai/gpt-astra" {
		t.Fatalf("opencode: %+v", oc)
	}
}

func TestModelsPerHarness(t *testing.T) {
	c, _ := Get("claude")
	ms, src, err := c.Models(context.Background(), "")
	if err != nil || len(ms) < 4 || ms[0].ID != "opus" || !strings.Contains(src, "built in") {
		t.Fatalf("claude: %v %q %v", ms, src, err)
	}
	o, _ := Get("opencode")
	if _, _, err := o.Models(context.Background(), "/nonexistent/opencode"); err == nil || !strings.Contains(err.Error(), "not installed") {
		t.Fatalf("a missing binary should say so: %v", err)
	}
	for _, n := range append(Names(), "fake") {
		p, _ := Get(n)
		if p.models == nil {
			t.Errorf("%s has no model list", n)
		}
	}
}
