package gitx

import (
	"context"
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

func TestLatestTag(t *testing.T) {
	cases := []struct {
		name string
		tags []string
		want string
	}{
		{"unpadded calver", []string{"v2026.9.20.4", "v2026.11.2.1", "v2026.10.15.1"}, "v2026.11.2.1"},
		{"padded calver", []string{"2026.11.07.3", "2026.11.08.1"}, "2026.11.08.1"},
		{"mixed padding", []string{"2026.11.08.1", "v2026.11.8.2"}, "v2026.11.8.2"},
		{"semver only", []string{"0.9.0", "0.14.4", "0.14.10", "v0.2.0"}, "0.14.10"},
		{"calver beats semver", []string{"0.15.0", "v2026.11.1.1", "v2026.11.1.0"}, "v2026.11.1.1"},
		{"pre-releases ignored", []string{"1.0.0", "1.1.0-rc1", "v2.0.0-beta"}, "1.0.0"},
		{"none", []string{"v1.0", "nightly"}, ""},
	}
	for _, c := range cases {
		if got := LatestTag(c.tags); got != c.want {
			t.Errorf("%s: LatestTag(%v) = %q, want %q", c.name, c.tags, got, c.want)
		}
	}
}

func TestNextCalVerCountsUnpaddedTags(t *testing.T) {
	d := time.Date(2026, 11, 8, 0, 0, 0, 0, time.UTC)
	if v := NextCalVer(d, []string{"v2026.11.8.1", "v2026.11.8.2"}); v != "2026.11.08.3" {
		t.Fatalf("next = %s", v)
	}
}

func TestLatestReleaseUsesGitTags(t *testing.T) {
	g := &Git{Run: func(_ context.Context, _, _ string, args ...string) (string, error) {
		switch args[0] {
		case "tag":
			return "0.14.4\n0.15.0\nv2026.11.1.0\nv2026.11.1.1", nil
		case "log":
			return "2026-11-01", nil
		case "rev-list":
			return "7", nil
		}
		return "", nil
	}}
	r := g.LatestRelease(context.Background(), "/repo")
	if r == nil || r.Tag != "v2026.11.1.1" || r.Date != "2026-11-01" || r.Commits != 7 {
		t.Fatalf("release = %+v", r)
	}
}
