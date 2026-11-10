package sessions

import (
	"context"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/techmuch/go-build-it/internal/config"
	"github.com/techmuch/go-build-it/internal/herdr"
	"github.com/techmuch/go-build-it/internal/history"
)

// Folder trust (FR-07f, ADR-013). Interactive harnesses ask once per folder whether to
// trust it. go-build-it creates those folders itself (member worktrees, helpers, the
// practice project), so with [agents] trust_folders on (the default) it answers yes for
// them, by pressing the keys a person would. Any other screen that blocks a harness
// (sign-in, permission dialogs) is still left to the person.

var (
	trustQuestion = regexp.MustCompile(`(?i)(do you trust|one you trust|trust this folder\?|trust the (files|contents|authors))`)
	trustYes      = regexp.MustCompile(`(?i)^(\d+[.)]\s*)?yes,?( i trust| proceed\b| trust\b)`)
)

// cursorMarks are the characters harness menus put before the selected option.
const cursorMarks = "❯>›▸▶→*"

// TrustKeys recognises a folder-trust screen and returns the herdr key names that select
// "Yes, I trust this folder" and confirm it. It returns false for any other screen, and
// when it cannot tell which option is selected.
func TrustKeys(screen string) ([]string, bool) {
	if !trustQuestion.MatchString(screen) {
		return nil, false
	}
	var opts []string // the non-blank lines, trimmed
	for _, l := range strings.Split(screen, "\n") {
		if t := strings.TrimSpace(l); t != "" {
			opts = append(opts, t)
		}
	}
	yes, cur := -1, -1
	for i, t := range opts {
		body := strings.TrimSpace(strings.TrimLeft(t, cursorMarks+" "))
		if trustYes.MatchString(body) {
			yes = i
		}
	}
	if yes < 0 {
		return nil, false
	}
	// The selected option is the marked line closest to "Yes" (menus are a few lines long).
	for d := 0; d <= 4 && cur < 0; d++ {
		for _, i := range []int{yes - d, yes + d} {
			if i >= 0 && i < len(opts) && cur < 0 && strings.ContainsAny(firstRune(opts[i]), cursorMarks) && strings.TrimSpace(strings.TrimLeft(opts[i], cursorMarks+" ")) != "" {
				cur = i
			}
		}
	}
	if cur < 0 {
		return nil, false
	}
	var keys []string
	for i := cur; i < yes; i++ {
		keys = append(keys, "down")
	}
	for i := cur; i > yes; i-- {
		keys = append(keys, "up")
	}
	return append(keys, "enter"), true
}

func firstRune(s string) string {
	for _, r := range s {
		return string(r)
	}
	return ""
}

// TrustAllowed reports whether go-build-it may answer a trust screen for dir: the setting
// is on and dir is inside go-build-it's own directory.
func TrustAllowed(cfg *config.Config, dir string) bool {
	if cfg == nil || !cfg.Agents.TrustFolders || dir == "" {
		return false
	}
	root, err := filepath.Abs(cfg.Dir())
	if err != nil {
		return false
	}
	d, err := filepath.Abs(dir)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(realPath(root), realPath(d))
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// realPath resolves symlinks in p's longest existing prefix (macOS /var is /private/var).
func realPath(p string) string {
	rest := ""
	for q := p; ; q = filepath.Dir(q) {
		if r, err := filepath.EvalSymlinks(q); err == nil {
			return filepath.Join(r, rest)
		}
		if filepath.Dir(q) == q {
			return p
		}
		rest = filepath.Join(filepath.Base(q), rest)
	}
}

// AnswerTrust answers a folder-trust screen in pane when TrustAllowed for the pane's
// folder (as herdr reports it). It waits briefly for the screen to draw, presses the keys,
// and waits until herdr no longer reports the agent blocked. It returns the folder and
// whether it answered.
func AnswerTrust(ctx context.Context, h *herdr.Client, cfg *config.Config, pane string) (string, bool) {
	if h == nil || cfg == nil || !cfg.Agents.TrustFolders {
		return "", false
	}
	info, err := h.Pane(ctx, pane)
	if err != nil {
		return "", false
	}
	var dir string
	for _, d := range []*string{info.Cwd, info.ForegroundCwd} {
		if d == nil || *d == "" {
			continue
		}
		if !TrustAllowed(cfg, *d) {
			return *d, false
		}
		dir = *d
	}
	if dir == "" {
		return "", false
	}
	return dir, answerTrust(ctx, h, pane)
}

func answerTrust(ctx context.Context, h *herdr.Client, pane string) bool {
	answered := false
	for try := 0; try < 3; try++ { // a harness may ask again (e.g. a second folder prompt)
		var keys []string
		ok := false
		for i := 0; i < 10 && !ok; i++ { // the screen can take a moment to draw
			screen, err := h.Read(ctx, pane, "visible", 60)
			if err == nil {
				keys, ok = TrustKeys(screen)
			}
			if !ok && !sleepCtx(ctx, 300*time.Millisecond) {
				return answered
			}
		}
		if !ok {
			return answered
		}
		if err := pressKeys(ctx, h, pane, keys); err != nil {
			return answered
		}
		answered = true
		for i := 0; i < 50; i++ { // up to 10 s for the harness to move on
			if !sleepCtx(ctx, 200*time.Millisecond) {
				return answered
			}
			a, err := h.Agent(ctx, pane)
			if err == nil && a.AgentStatus != herdr.AgentStatusBlocked && a.AgentStatus != herdr.AgentStatusUnknown {
				return answered
			}
		}
	}
	return answered
}

// keyBytes are what a terminal sends for the keys TrustKeys uses.
var keyBytes = map[string]string{"down": "\x1b[B", "up": "\x1b[A", "enter": "\r"}

// pressKeys sends keys to the agent in pane. herdr's agent.send_keys needs an agent herdr
// started by name; for any other agent the key codes are typed into the pane.
func pressKeys(ctx context.Context, h *herdr.Client, pane string, keys []string) error {
	if err := h.SendKeys(ctx, pane, keys...); err == nil || !herdr.IsCode(err, "agent_not_ready") && !herdr.IsCode(err, "agent_not_found") {
		return err
	}
	for _, k := range keys {
		if err := h.Call(ctx, herdr.MethodPaneSendText, map[string]any{"pane_id": pane, "text": keyBytes[k]}, nil); err != nil {
			return err
		}
		if !sleepCtx(ctx, 100*time.Millisecond) {
			return ctx.Err()
		}
	}
	return nil
}

func sleepCtx(ctx context.Context, d time.Duration) bool {
	select {
	case <-ctx.Done():
		return false
	case <-time.After(d):
		return true
	}
}

// answerTrust answers a trust screen in pane and records it in the history (FR-07f).
func (m *Manager) answerTrust(ctx context.Context, h *herdr.Client, projectID, who, label, pane string) bool {
	dir, ok := AnswerTrust(ctx, h, m.cfg, pane)
	if !ok {
		return false
	}
	m.hist.Must(ctx, history.Event{ProjectID: projectID, Type: "agent.trusted", Subject: who,
		Summary: "Answered yes to " + label + "'s folder trust prompt for " + who,
		Why:     "go-build-it created this folder, and [agents] trust_folders is on",
		Data:    history.Data(map[string]any{"folder": dir, "pane": pane, "harness": label})})
	return true
}
