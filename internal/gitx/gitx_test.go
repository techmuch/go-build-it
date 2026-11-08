package gitx

import (
	"testing"
	"time"
)

func TestCalVerAndForge(t *testing.T) {
	d := time.Date(2026, 11, 8, 0, 0, 0, 0, time.UTC)
	if v := NextCalVer(d, []string{"2026.11.07.3", "2026.11.08.1", "2026.11.08.2", "v1.0"}); v != "2026.11.08.3" {
		t.Fatalf("next = %s", v)
	}
	if v := NextCalVer(d, nil); v != "2026.11.08.1" {
		t.Fatalf("first = %s", v)
	}
	if Forge("git@github.com:techmuch/go-build-it.git") != "github" || Forge("https://gitlab.com/a/b") != "gitlab" || Forge("") != "" {
		t.Fatal("forge detection")
	}
	if s := RepoSlug("git@github.com:techmuch/go-build-it.git"); s != "techmuch/go-build-it" {
		t.Fatalf("slug %q", s)
	}
	if s := RepoSlug("https://github.com/techmuch/go-build-it"); s != "techmuch/go-build-it" {
		t.Fatalf("slug %q", s)
	}
}
