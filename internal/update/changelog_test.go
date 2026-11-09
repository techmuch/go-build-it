package update

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	gobuildit "github.com/techmuch/go-build-it"
)

// The embedded CHANGELOG.md must parse and be in order, or the app would show it wrong (FR-06a).
func TestEmbeddedChangelogIsValid(t *testing.T) {
	entries := ParseChangelog(gobuildit.Changelog)
	if len(entries) == 0 {
		t.Fatal("CHANGELOG.md has no entries")
	}
	if errs := CheckChangelog(entries); len(errs) > 0 {
		t.Fatalf("CHANGELOG.md: %s", strings.Join(errs, "; "))
	}
}

const sample = `# Changelog

## [Unreleased]

### Added

- New thing
  that wraps

## [2026.11.09.2] - 2026-11-09

### Fixed

- A bug

## [2026.11.09.1] - 2026-11-09

### Added

- First
`

func TestParseChangelog(t *testing.T) {
	e := ParseChangelog([]byte(sample))
	if len(e) != 3 || e[0].Version != Unreleased || e[1].Version != "2026.11.09.2" || e[1].Date != "2026-11-09" {
		t.Fatalf("entries: %+v", e)
	}
	if got := e[0].Sections[0].Items[0]; got != "New thing that wraps" {
		t.Fatalf("continuation not joined: %q", got)
	}
	if errs := CheckChangelog(e); len(errs) > 0 {
		t.Fatal(errs)
	}
	n, err := Notes(e, "2026.11.09.2")
	if err != nil || !strings.Contains(n, "### Fixed") || strings.Contains(n, "2026.11.09.1") {
		t.Fatalf("notes %q %v", n, err)
	}
	if _, err := Notes(e, "2027.01.01.1"); err == nil {
		t.Fatal("missing entry should fail")
	}
}

func TestCheckChangelogFindsProblems(t *testing.T) {
	bad := ParseChangelog([]byte("## [2026.11.09.1] - 2026-11-09\n### Added\n- a\n## [2026.11.10.1] - 2026-11-10\n### Added\n- b\n## [v2]\n### Added\n- c\n## [Unreleased]\n"))
	errs := strings.Join(CheckChangelog(bad), "|")
	for _, want := range []string{"not older", "not a YYYY.MM.DD.N", "Unreleased must be the first"} {
		if !strings.Contains(errs, want) {
			t.Errorf("want %q in %s", want, errs)
		}
	}
}

func TestReleaseDatesTheUnreleasedEntry(t *testing.T) {
	p := filepath.Join(t.TempDir(), "CHANGELOG.md")
	os.WriteFile(p, []byte(sample), 0o644)
	if err := ReleaseChangelog(p, "2026.11.09.2", time.Now()); err == nil {
		t.Fatal("existing version should fail")
	}
	if err := ReleaseChangelog(p, "2026.11.10.1", time.Date(2026, 11, 10, 0, 0, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(p)
	e := ParseChangelog(b)
	if e[0].Version != Unreleased || len(e[0].Sections) != 0 || e[1].Version != "2026.11.10.1" || e[1].Date != "2026-11-10" || e[1].Sections[0].Items[0] != "New thing that wraps" {
		t.Fatalf("after release: %+v", e[:2])
	}
	if errs := CheckChangelog(e); len(errs) > 0 {
		t.Fatal(errs)
	}
	if err := ReleaseChangelog(p, "2026.11.10.2", time.Now()); err == nil || !strings.Contains(err.Error(), "no changes") {
		t.Fatalf("empty Unreleased should fail: %v", err)
	}
}
