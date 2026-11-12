package app

import (
	"strings"
	"testing"
)

// FR-63a: the model picker's list comes from the harness, through the CLI like every UI action.
func TestHarnessModels(t *testing.T) {
	e := setup(t)
	var l struct {
		Models    []struct{ ID, Name string }
		Source    string
		FetchedAt string
		Error     string
	}
	e.gbiJSON(&l, "harnesses", "models", "fake")
	if len(l.Models) != 2 || l.Models[0].ID != "fake-fast" || l.FetchedAt == "" || l.Error != "" {
		t.Fatalf("fake: %+v", l)
	}
	first := l.FetchedAt
	e.gbiJSON(&l, "harnesses", "models", "fake")
	if l.FetchedAt != first {
		t.Fatal("the second request should come from the cache")
	}
	e.gbiJSON(&l, "harnesses", "models", "claude")
	if len(l.Models) < 4 || !strings.Contains(l.Source, "built in") {
		t.Fatalf("claude: %+v", l)
	}
	if out, err := e.gbi("harnesses", "models", "nope"); err == nil || !strings.Contains(out, "unknown harness") {
		t.Fatalf("unknown harness: %v %s", err, out)
	}
}
