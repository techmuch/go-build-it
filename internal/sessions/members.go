package sessions

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/techmuch/go-build-it/internal/harness"
	"github.com/techmuch/go-build-it/internal/herdr"
	"github.com/techmuch/go-build-it/internal/projects"
	"github.com/techmuch/go-build-it/internal/store"
	"github.com/techmuch/go-build-it/internal/team"
)

// MemberWorktree is a member's long-lived worktree (FR-64, FR-12d).
func (m *Manager) MemberWorktree(projectID, memberID string) string {
	slug := strings.SplitN(memberID, "@", 2)[0]
	return filepath.Join(m.cfg.Dir(), "worktrees", projectID, "members", slug)
}

// IsMemberWorktree reports whether dir is a member worktree (never removed after a merge).
func IsMemberWorktree(dir string) bool { return filepath.Base(filepath.Dir(dir)) == "members" }

// prepareWorktree makes sure the member's worktree exists and has the task branch checked
// out: an existing branch (a requeued task) is checked out as is, a new one is created from
// the default branch. Leftover changes from earlier work are stashed, never lost.
func (m *Manager) prepareWorktree(ctx context.Context, p projects.Project, mem team.Member, branch string) (string, error) {
	wt := m.MemberWorktree(p.ID, mem.ID)
	base := m.git.DefaultBranch(ctx, p.Path)
	if _, err := os.Stat(filepath.Join(wt, ".git")); err != nil {
		if err := os.MkdirAll(filepath.Dir(wt), 0o755); err != nil {
			return "", err
		}
		home := "gbi/" + strings.SplitN(mem.ID, "@", 2)[0]
		if _, err := m.git.Run(ctx, p.Path, "git", "worktree", "add", "-B", home, wt, base); err != nil {
			return "", fmt.Errorf("create member worktree: %w", err)
		}
	}
	if out, _ := m.git.Run(ctx, wt, "git", "status", "--porcelain"); strings.TrimSpace(filterGenerated(out)) != "" {
		_, _ = m.git.Run(ctx, wt, "git", "stash", "push", "-u", "-m", "gbi: left over before "+branch)
	}
	if _, err := m.git.Run(ctx, p.Path, "git", "rev-parse", "--verify", "--quiet", "refs/heads/"+branch); err == nil {
		if _, err := m.git.Run(ctx, wt, "git", "checkout", branch); err != nil {
			return "", fmt.Errorf("check out %s in %s's worktree: %w", branch, mem.Name, err)
		}
	} else if _, err := m.git.Run(ctx, wt, "git", "checkout", "-b", branch, base); err != nil {
		return "", fmt.Errorf("create %s in %s's worktree: %w", branch, mem.Name, err)
	}
	return wt, nil
}

// filterGenerated drops go-build-it's own files from `git status --porcelain` output.
func filterGenerated(status string) string {
	var keep []string
	for _, l := range strings.Split(status, "\n") {
		f := strings.TrimSpace(l)
		if len(f) > 3 {
			f = strings.TrimSpace(f[2:])
		}
		switch f {
		case "", TaskFile, "AGENTS.md", "CLAUDE.md", "GEMINI.md", ".gbi-handoff.md":
			continue
		}
		keep = append(keep, l)
	}
	return strings.Join(keep, "\n")
}

func (m *Manager) memberLock(memberID string) *sync.Mutex {
	m.mu.Lock()
	defer m.mu.Unlock()
	l, ok := m.launching[memberID]
	if !ok {
		l = &sync.Mutex{}
		m.launching[memberID] = l
	}
	return l
}

// projectWorkspace finds or creates the herdr workspace for a project (one per project, FR-67).
func (m *Manager) projectWorkspace(ctx context.Context, h *herdr.Client, p projects.Project) (string, error) {
	m.mu.Lock()
	id := m.workspaces[p.ID]
	m.mu.Unlock()
	snap, err := h.Snapshot(ctx)
	if err != nil {
		return "", err
	}
	for _, w := range snap.Workspaces {
		if (id != "" && w.WorkspaceID == id) || (id == "" && w.Label == p.ID && w.Worktree == nil) {
			m.mu.Lock()
			m.workspaces[p.ID] = w.WorkspaceID
			m.mu.Unlock()
			return w.WorkspaceID, nil
		}
	}
	cr, err := h.CreateWorkspace(ctx, p.Path, p.ID)
	if err != nil {
		return "", err
	}
	m.mu.Lock()
	m.workspaces[p.ID] = cr.Workspace.WorkspaceID
	m.mu.Unlock()
	_ = h.ReportMetadata(ctx, cr.RootPane.PaneID, herdr.Metadata{DisplayAgent: p.ID + " (project)", Title: p.Purpose})
	return cr.Workspace.WorkspaceID, nil
}

// ensurePane returns the member's pane, opening its worktree grouped under the project.
func (m *Manager) ensurePane(ctx context.Context, h *herdr.Client, p projects.Project, mem team.Member, wt string) (string, error) {
	if mem.PaneID != "" {
		if pi, err := h.Pane(ctx, mem.PaneID); err == nil && pi.Cwd != nil && samePath(*pi.Cwd, wt) {
			return mem.PaneID, nil
		}
	}
	// After a herdr restart pane IDs change: find the member's pane by its directory.
	if snap, err := h.Snapshot(ctx); err == nil {
		for _, pi := range snap.Panes {
			if pi.Cwd != nil && samePath(*pi.Cwd, wt) {
				return pi.PaneID, nil
			}
		}
	}
	ws, err := m.projectWorkspace(ctx, h, p)
	if err != nil {
		return "", err
	}
	cr, err := h.OpenWorktree(ctx, ws, wt, mem.Name)
	if err != nil {
		// Older herdr or a non-git layout: fall back to a plain workspace in the worktree.
		cr2, err2 := h.CreateWorkspace(ctx, wt, mem.Name)
		if err2 != nil {
			return "", fmt.Errorf("open %s's worktree in herdr: %v (%v)", mem.Name, err, err2)
		}
		cr = cr2
	}
	if cr.RootPane.PaneID == "" {
		return "", errors.New("herdr returned no pane for the member worktree")
	}
	return cr.RootPane.PaneID, nil
}

func samePath(a, b string) bool {
	ra, ea := filepath.EvalSymlinks(a)
	rb, eb := filepath.EvalSymlinks(b)
	if ea == nil && eb == nil {
		return ra == rb
	}
	return filepath.Clean(a) == filepath.Clean(b)
}

// agentName is the herdr agent name for a member: [a-z][a-z0-9_-]{0,31}.
func agentName(mem team.Member) string {
	parts := strings.SplitN(mem.ID, "@", 2)
	if len(parts) == 2 {
		return herdr.SafeName(parts[1], parts[0])
	}
	return herdr.SafeName(mem.ID)
}

// launched describes the agent running in a member's pane.
type launched struct {
	pane    string
	conv    string
	blocked bool   // the harness is waiting for the human (e.g. first-run folder trust)
	message string // what it is waiting for
	started bool   // a new process was started by this call
}

// ensureAgent makes sure the member's harness runs in its pane, resuming its conversation
// or starting a new one whose ID go-build-it chooses (ADR-009 findings).
func (m *Manager) ensureAgent(ctx context.Context, h *herdr.Client, p projects.Project, mem team.Member, wt string) (launched, error) {
	lock := m.memberLock(mem.ID)
	lock.Lock()
	defer lock.Unlock()
	prof, ok := harness.Get(mem.Harness)
	if !ok {
		return launched{}, fmt.Errorf("unknown harness %q", mem.Harness)
	}
	pane, err := m.ensurePane(ctx, h, p, mem, wt)
	if err != nil {
		return launched{}, err
	}
	out := launched{pane: pane, conv: mem.ConversationID}
	if a, err := h.Agent(ctx, pane); err == nil && a.Agent != nil && agentMatches(*a.Agent, prof) {
		if a.AgentStatus == herdr.AgentStatusBlocked {
			out.blocked = true
		}
		if out.conv == "" && a.AgentSession != nil {
			out.conv = a.AgentSession.Value
		}
		m.saveRuntime(ctx, mem, wt, pane, out.conv)
		return out, nil
	}
	resume := mem.ConversationID != ""
	conv := mem.ConversationID
	if !resume {
		conv = prof.NewConversationID()
	}
	hc := m.cfg.Harnesses[mem.Harness]
	args := prof.Args(firstNonEmpty(mem.Model, hc.Model), hc.Flags, conv, resume)
	launchedAt := time.Now()
	if prof.Kind != "" {
		var err error
		// A freshly opened pane may still be starting its shell (agent_pane_busy): retry briefly.
		for i := 0; i < 30; i++ {
			_, err = h.StartAgent(ctx, agentName(mem), prof.Kind, pane, args, 60*time.Second)
			if !herdr.IsCode(err, "agent_pane_busy") {
				break
			}
			select {
			case <-ctx.Done():
				return out, ctx.Err()
			case <-time.After(500 * time.Millisecond):
			}
		}
		if err == nil {
			// Some harnesses show a first-run prompt only after they are detected (Antigravity's folder trust).
			time.Sleep(1500 * time.Millisecond)
			if a, aerr := h.Agent(ctx, pane); aerr == nil && a.AgentStatus == herdr.AgentStatusBlocked {
				out.blocked, out.message = true, prof.Label+" is waiting for you (for example a first-run trust prompt)"
			}
		}
		switch {
		case err == nil:
		case herdr.IsCode(err, "agent_not_ready"):
			out.blocked, out.message = true, "the harness is waiting for you (for example a first-run trust prompt)"
		default:
			return out, fmt.Errorf("start %s in herdr: %w", prof.Label, err)
		}
	} else {
		bin := firstNonEmpty(hc.Binary, m.GbiBin)
		cmd := ShellCommand(map[string]string{"GBI_FAKE_TRANSCRIPTS": harness.FakeTranscriptDir(), "GBI_URL": "http://" + m.cfg.Listen}, bin, args)
		if err := h.Call(ctx, herdr.MethodPaneSendText, map[string]any{"pane_id": pane, "text": cmd}, nil); err != nil {
			return out, err
		}
		if err := waitAgent(ctx, h, pane, prof, 30*time.Second); err != nil {
			return out, err
		}
	}
	out.started = true
	if conv == "" { // harnesses that pick their own ID: ask herdr, then the harness's store
		if a, err := h.Agent(ctx, pane); err == nil && a.AgentSession != nil {
			conv = a.AgentSession.Value
		} else {
			conv = prof.FindNewConversation(launchedAt.Add(-2 * time.Second))
		}
	}
	out.conv = conv
	started := mem.ConversationStarted
	if !resume {
		started = store.Now()
	}
	_ = m.team.SetRuntime(ctx, mem.ID, wt, pane, conv, map[bool]int{true: mem.ConversationTasks, false: 0}[resume], started)
	return out, nil
}

func (m *Manager) saveRuntime(ctx context.Context, mem team.Member, wt, pane, conv string) {
	if mem.Worktree != wt || mem.PaneID != pane || mem.ConversationID != conv {
		_ = m.team.SetRuntime(ctx, mem.ID, wt, pane, conv, mem.ConversationTasks, firstNonEmpty(mem.ConversationStarted, store.Now()))
	}
}

func agentMatches(agent string, prof harness.Profile) bool {
	switch {
	case prof.Kind == "":
		return agent == "fake"
	case prof.Kind == "agy":
		return agent == "agy" || agent == "antigravity" || strings.HasPrefix(agent, "agy")
	default:
		return agent == prof.Kind
	}
}

func waitAgent(ctx context.Context, h *herdr.Client, pane string, prof harness.Profile, d time.Duration) error {
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		if a, err := h.Agent(ctx, pane); err == nil && a.Agent != nil && agentMatches(*a.Agent, prof) && a.AgentStatus != herdr.AgentStatusUnknown {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(200 * time.Millisecond):
		}
	}
	screen, _ := h.Read(context.Background(), pane, "visible", 20)
	return fmt.Errorf("%s did not start in herdr within %s; screen:\n%s", prof.Label, d, screen)
}

func shq(s string) string { return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'" }

// ShellCommand is a line typed into a pane's shell that runs bin with args and env:
// POSIX shells on macOS and Linux, PowerShell on Windows (herdr's default there).
func ShellCommand(env map[string]string, bin string, args []string) string {
	keys := make([]string, 0, len(env))
	for k := range env {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	if runtime.GOOS == "windows" {
		ps := func(s string) string { return "'" + strings.ReplaceAll(s, "'", "''") + "'" }
		var b strings.Builder
		for _, k := range keys {
			fmt.Fprintf(&b, "$env:%s=%s; ", k, ps(env[k]))
		}
		b.WriteString("& " + ps(bin))
		for _, a := range args {
			b.WriteString(" " + ps(a))
		}
		return b.String() + "\r\n"
	}
	var b strings.Builder
	for _, k := range keys {
		fmt.Fprintf(&b, "%s=%s ", k, shq(env[k]))
	}
	b.WriteString(shq(bin))
	for _, a := range args {
		b.WriteString(" " + shq(a))
	}
	return b.String() + "\n"
}

func shqAll(a []string) []string {
	out := make([]string, len(a))
	for i, s := range a {
		out[i] = shq(s)
	}
	return out
}

// Label shows a member's name, role, task and context on its herdr pane (FR-67).
func (m *Manager) label(ctx context.Context, h *herdr.Client, mem team.Member, pane, task string, frac float64, measured bool, convTasks int) {
	if pane == "" {
		return
	}
	title := "idle"
	tokens := map[string]string{"conv": fmt.Sprint(convTasks)}
	var clear []string
	if task != "" {
		title, tokens["task"] = task, strings.SplitN(task, " ", 2)[0]
	} else {
		clear = append(clear, "task")
	}
	if frac > 0 {
		ctxs := fmt.Sprintf("%.0f%%", frac*100)
		if !measured {
			ctxs = "~" + ctxs
		}
		tokens["ctx"] = ctxs
	}
	_ = h.ReportMetadata(ctx, pane, herdr.Metadata{DisplayAgent: fmt.Sprintf("%s (%s)", mem.Name, mem.Role), Title: title, Tokens: tokens, ClearTokens: clear,
		StateLabels: map[string]string{"done": "review ready"}})
}

// RefreshProjectSummary sets the project workspace's summary token (FR-67).
func (m *Manager) RefreshProjectSummary(ctx context.Context, projectID string) {
	h, err := m.herdr()
	if err != nil {
		return
	}
	m.mu.Lock()
	ws := m.workspaces[projectID]
	working, blocked := 0, 0
	for _, r := range m.running {
		if r.s.ProjectID != projectID {
			continue
		}
		if r.state == "blocked" {
			blocked++
		} else {
			working++
		}
	}
	m.mu.Unlock()
	if ws == "" {
		return
	}
	var parts []string
	if working > 0 {
		parts = append(parts, fmt.Sprintf("%d working", working))
	}
	if blocked > 0 {
		parts = append(parts, fmt.Sprintf("%d needs you", blocked))
	}
	if len(parts) == 0 {
		parts = []string{"idle"}
	}
	_ = h.ReportWorkspace(ctx, ws, map[string]string{"gbi": strings.Join(parts, " · ")})
}
