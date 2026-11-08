package sessions

// Attachable (headed) sessions — experimental (FR-67, Q7, ADR-008).
//
// The harness runs interactively inside a tmux session named gbi-<session id>.
// go-build-it mirrors the pane into the terminal view, sends input with
// `tmux send-keys`, and anyone can attach with `tmux attach -t gbi-<id>` to take
// control. Usage is not reported in this mode. The session ends when the harness
// exits or someone marks it finished.

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/techmuch/go-build-it/internal/access"
	"github.com/techmuch/go-build-it/internal/history"
	"github.com/techmuch/go-build-it/internal/queue"
	"github.com/techmuch/go-build-it/internal/store"
)

// interactive commands per harness (no print/headless flags).
var interactive = map[string][]string{
	"claude":      {"claude", "--permission-mode", "acceptEdits"},
	"antigravity": {"agy"},
	"opencode":    {"opencode"},
	"pi":          {"pi"},
	"fake":        {"gbi", "dev", "fake-tui"}, // tests: binary comes from the fake harness config
}

// TmuxName is the tmux session for a go-build-it session.
func TmuxName(id string) string { return "gbi-" + id }

// AttachCommand is what a human runs to take control.
func AttachCommand(id string) string { return "tmux attach -t " + TmuxName(id) }

func tmux(args ...string) (string, error) {
	out, err := exec.Command("tmux", args...).CombinedOutput()
	return strings.TrimRight(string(out), "\n"), err
}

// AttachableAvailable reports whether headed sessions can run here.
func (m *Manager) AttachableAvailable(harnessName string) bool {
	if !m.cfg.Experimental.AttachableSessions {
		return false
	}
	if _, ok := interactive[harnessName]; !ok {
		return false
	}
	_, err := exec.LookPath("tmux")
	return err == nil
}

// startAttached runs a task in a tmux session. Called by Start when attachable mode applies.
func (m *Manager) startAttached(ctx context.Context, s Session, t queue.Task, prompt, binary string, env []string) (Session, error) {
	cmd := append([]string{}, interactive[s.Harness]...)
	if binary != "" {
		cmd[0] = binary
	}
	// The prompt goes into a file so newlines don't submit early in the TUI.
	taskFile := filepath.Join(s.Workdir, ".gbi-task.md")
	if err := os.WriteFile(taskFile, []byte(prompt), 0o644); err != nil {
		return s, err
	}
	shell := strings.Join(append(envPrefix(env), quoteAll(cmd)...), " ")
	if out, err := tmux("new-session", "-d", "-s", TmuxName(s.ID), "-x", "160", "-y", "48", "-c", s.Workdir, shell); err != nil {
		return s, fmt.Errorf("tmux: %v: %s", err, out)
	}
	s.Status = "running"
	s.ExitReason = "attachable: " + AttachCommand(s.ID)
	if err := m.write(ctx, s); err != nil {
		return s, err
	}
	if _, err := m.queue.SetStatus(ctx, t.ID, queue.Running, "attachable session "+s.ID+" started"); err != nil {
		return s, err
	}
	m.mu.Lock()
	m.attached[s.ID] = make(chan string, 1)
	m.mu.Unlock()
	pctx := access.Persona(context.Background(), s.MemberID)
	m.hist.Must(pctx, history.Event{ProjectID: s.ProjectID, Type: "session.started", Subject: s.ID, Summary: fmt.Sprintf("%s started %s in an attachable session", s.MemberID, t.ID),
		Why: "experimental attachable sessions (FR-67)", Data: history.Data(map[string]any{"attach": AttachCommand(s.ID), "task": t.ID})})
	go func() {
		time.Sleep(2 * time.Second) // let the TUI start
		_, _ = tmux("send-keys", "-t", TmuxName(s.ID), "-l", "Read .gbi-task.md in this directory and do the task it describes. Say DONE when finished.")
		_, _ = tmux("send-keys", "-t", TmuxName(s.ID), "Enter")
	}()
	go m.mirror(pctx, s, t)
	return s, nil
}

// mirror copies new pane lines into session output until the tmux session ends or is finished.
func (m *Manager) mirror(ctx context.Context, s Session, t queue.Task) {
	var prev []string
	seq := 0
	m.mu.Lock()
	done := m.attached[s.ID]
	m.mu.Unlock()
	reason := "harness exited"
	tick := time.NewTicker(time.Second)
	defer tick.Stop()
loop:
	for {
		select {
		case r := <-done:
			reason = r
			break loop
		case <-tick.C:
		}
		out, err := tmux("capture-pane", "-p", "-t", TmuxName(s.ID), "-S", "-300")
		if err != nil {
			break // session gone: the harness exited
		}
		lines := strings.Split(out, "\n")
		start := overlap(prev, lines)
		for _, l := range lines[start:] {
			seq++
			_, _ = m.db.ExecContext(ctx, `INSERT OR REPLACE INTO session_output(session_id,seq,ts,kind,text) VALUES(?,?,?,?,?)`, s.ID, 100000+seq, store.Now(), "raw", l)
			m.hist.Publish(history.Event{Type: "session.output", Subject: s.ID, ProjectID: s.ProjectID, Summary: l, Data: history.Data(map[string]any{"seq": 100000 + seq, "kind": "raw"})})
		}
		prev = lines
	}
	_, _ = tmux("kill-session", "-t", TmuxName(s.ID))
	_ = os.Remove(filepath.Join(s.Workdir, ".gbi-task.md"))
	m.mu.Lock()
	delete(m.attached, s.ID)
	m.mu.Unlock()
	s.EndedAt = store.Now()
	if reason == "cancelled" {
		s.Status, s.ExitReason = "cancelled", "cancelled"
		_, _ = m.queue.SetStatus(ctx, t.ID, queue.Queued, "attachable session cancelled")
		_ = m.write(ctx, s)
		return
	}
	s.Status, s.ExitReason = "done", reason
	_ = m.write(ctx, s)
	cur, _ := m.queue.Get(ctx, t.ID)
	if m.Hooks.Completed != nil {
		if err := m.Hooks.Completed(ctx, s, cur); err != nil {
			_, _ = m.queue.SetStatus(ctx, t.ID, queue.Failed, "courier: "+err.Error())
			return
		}
	}
	if cur2, _ := m.queue.Get(ctx, t.ID); cur2.Status == queue.Running {
		_, _ = m.queue.SetStatus(ctx, t.ID, queue.Review, "attachable session finished")
	}
	m.hist.Must(ctx, history.Event{ProjectID: s.ProjectID, Type: "session.completed", Subject: s.ID, Summary: fmt.Sprintf("%s finished %s (attachable)", s.MemberID, t.ID), Why: reason})
}

// overlap finds where new lines begin, given the previous capture.
func overlap(prev, cur []string) int {
	if len(prev) == 0 {
		return 0
	}
	last := prev[len(prev)-1]
	for i := len(cur) - 1; i >= 0; i-- {
		if cur[i] == last && i+1 <= len(cur) {
			return i + 1
		}
	}
	return 0
}

// Finish ends an attachable session; Courier then commits and opens the review.
func (m *Manager) Finish(ctx context.Context, id string) error {
	m.mu.Lock()
	ch, ok := m.attached[id]
	m.mu.Unlock()
	if !ok {
		return errors.New("only attachable sessions are finished by hand")
	}
	ch <- "finished by " + access.FromContext(ctx).Name
	return nil
}

func sendAttached(id, text string) error {
	if _, err := tmux("send-keys", "-t", TmuxName(id), "-l", text); err != nil {
		return err
	}
	_, err := tmux("send-keys", "-t", TmuxName(id), "Enter")
	return err
}

func envPrefix(env []string) []string {
	out := []string{"env"}
	for _, e := range env {
		out = append(out, shellQuote(e))
	}
	return out
}

func quoteAll(a []string) []string {
	out := make([]string, len(a))
	for i, s := range a {
		out[i] = shellQuote(s)
	}
	return out
}

func shellQuote(s string) string { return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'" }
