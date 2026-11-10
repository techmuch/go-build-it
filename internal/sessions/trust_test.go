package sessions

import (
	"path/filepath"
	"reflect"
	"testing"

	"github.com/techmuch/go-build-it/internal/config"
)

// Screens captured from the real harnesses (Claude Code 2.1.321, Antigravity CLI 1.5.3).
const claudeTrust = `➜  claude claude --permission-mode acceptEdits

───────────────────────────────────────────────
 Accessing workspace:

 /Users/me/.go-build-it/helpers/claude

 Quick safety check: Is this a project you created or one you trust? (Like your own code, a well-known open source
 project, or work from your team). If not, take a moment to review what's in this folder first.

 Claude Code'll be able to read, edit, and execute files here.

 Security guide

 ❯ No, exit
   Yes, I trust this folder

 Enter to confirm · Esc to cancel
`

const agyTrust = `
/Users/me/.go-build-it/helpers/antigravity

Do you trust the contents of this project?

Antigravity CLI requires permission to read, edit, and execute files here.

> Yes, I trust this folder
  No, exit

  ↑/↓ Navigate · enter Confirm
                                                    Gemini 3.8 Flash · medium
`

const claudeOldTrust = `Do you trust the files in this folder?

/tmp/x

❯ 1. Yes, proceed
  2. No, exit
`

func TestTrustKeys(t *testing.T) {
	for name, tc := range map[string]struct {
		screen string
		keys   []string
		ok     bool
	}{
		"claude":        {claudeTrust, []string{"down", "enter"}, true},
		"antigravity":   {agyTrust, []string{"enter"}, true},
		"claude older":  {claudeOldTrust, []string{"enter"}, true},
		"yes below two": {"Do you trust this folder?\n❯ No, exit\n  Not now\n  Yes, I trust this folder\n", []string{"down", "down", "enter"}, true},
		"cursor below":  {"Do you trust this folder?\n  Yes, I trust this folder\n❯ No, exit\n", []string{"up", "enter"}, true},
		"sign in":       {"fake agent · not signed in\nSelect login method:\n❯ 1. Fake account\n  2. Fake console", nil, false},
		"permission":    {"Do you want to make this edit to main.go?\n❯ 1. Yes\n  2. No, and tell Claude what to do", nil, false},
		"no cursor":     {"Do you trust the contents of this project?\nYes, I trust this folder\nNo, exit", nil, false},
		"idle prompt":   {"? for shortcuts\n> ", nil, false},
	} {
		keys, ok := TrustKeys(tc.screen)
		if ok != tc.ok || !reflect.DeepEqual(keys, tc.keys) {
			t.Errorf("%s: got %v %v, want %v %v", name, keys, ok, tc.keys, tc.ok)
		}
	}
}

func TestTrustAllowed(t *testing.T) {
	dir := t.TempDir()
	cfg, err := config.LoadFrom(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.Agents.TrustFolders {
		t.Fatal("trust_folders should default to on")
	}
	in := filepath.Join(dir, "worktrees", "demo", "members", "forge")
	if !TrustAllowed(cfg, in) || !TrustAllowed(cfg, filepath.Join(dir, "helpers", "claude")) {
		t.Fatal("folders inside go-build-it's directory should be trusted")
	}
	for _, out := range []string{t.TempDir(), filepath.Dir(dir), dir + "-other", ""} {
		if TrustAllowed(cfg, out) {
			t.Errorf("%q is outside go-build-it's directory", out)
		}
	}
	cfg.Agents.TrustFolders = false
	if TrustAllowed(cfg, in) {
		t.Fatal("the setting turns it off")
	}
}
