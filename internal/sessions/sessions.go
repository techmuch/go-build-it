// Package sessions runs harness sessions for tasks and dispatches work from the queue
// (FR-60, FR-64, FR-65, FR-70–FR-72, FR-81, FR-82).
package sessions

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/techmuch/go-build-it/internal/access"
	"github.com/techmuch/go-build-it/internal/config"
	"github.com/techmuch/go-build-it/internal/gitx"
	"github.com/techmuch/go-build-it/internal/harness"
	"github.com/techmuch/go-build-it/internal/history"
	"github.com/techmuch/go-build-it/internal/projects"
	"github.com/techmuch/go-build-it/internal/queue"
	"github.com/techmuch/go-build-it/internal/standards"
	"github.com/techmuch/go-build-it/internal/store"
	"github.com/techmuch/go-build-it/internal/team"
)

// Session is one harness run on a task.
type Session struct {
	ID             string  `json:"id"`
	TaskID         string  `json:"taskId"`
	ProjectID      string  `json:"projectId"`
	MemberID       string  `json:"memberId"`
	Harness        string  `json:"harness"`
	Model          string  `json:"model"`
	Status         string  `json:"status"`
	StartedAt      string  `json:"startedAt"`
	EndedAt        string  `json:"endedAt"`
	Used           int     `json:"used"`
	Window         int     `json:"window"`
	Measured       bool    `json:"measured"`
	Compactions    int     `json:"compactions"`
	HarnessSession string  `json:"harnessSession"`
	Workdir        string  `json:"workdir"`
	ResumeAt       string  `json:"resumeAt"`
	ExitReason     string  `json:"exitReason"`
	Handoff        string  `json:"handoff"`
	Fraction       float64 `json:"fraction"`
	Live           bool    `json:"live"`
	Attach         string  `json:"attach,omitempty"` // tmux attach command for attachable sessions
}

// Line is one line of session output (structured or raw, FR-66).
type Line struct {
	Seq  int    `json:"seq"`
	TS   string `json:"ts"`
	Kind string `json:"kind"`
	Text string `json:"text"`
}

// Hooks let other services react to session outcomes without import cycles.
type Hooks struct {
	// Completed runs when a session ends successfully, before the task moves to review (Courier commits/pushes here).
	Completed func(ctx context.Context, s Session, t queue.Task) error
	// Handoff asks for a briefing when smart handoffs are enabled (experimental, P9).
	Handoff func(ctx context.Context, s Session, t queue.Task) string
}

// Manager runs sessions.
type Manager struct {
	db        *sql.DB
	hist      *history.Recorder
	cfg       *config.Config
	projects  *projects.Service
	queue     *queue.Service
	team      *team.Service
	standards *standards.Service
	git       *gitx.Git
	Hooks     Hooks
	log       *slog.Logger

	mu       sync.Mutex
	active   map[string]*active
	attached map[string]chan string // attachable (tmux) sessions, experimental
	auto     bool
	stop     chan struct{}
}

type active struct {
	proc   *harness.Process
	cancel context.CancelFunc
	seq    int
	reason string
}

// New returns a session manager.
func New(db *sql.DB, h *history.Recorder, cfg *config.Config, p *projects.Service, q *queue.Service, t *team.Service, st *standards.Service, g *gitx.Git, log *slog.Logger) *Manager {
	return &Manager{db: db, hist: h, cfg: cfg, projects: p, queue: q, team: t, standards: st, git: g, log: log, active: map[string]*active{}, attached: map[string]chan string{}}
}

const cols = `id,task_id,project_id,member_id,harness,model,status,started_at,ended_at,used,window,measured,compactions,harness_session,workdir,resume_at,exit_reason,handoff`

func scan(sc interface{ Scan(...any) error }) (Session, error) {
	var s Session
	err := sc.Scan(&s.ID, &s.TaskID, &s.ProjectID, &s.MemberID, &s.Harness, &s.Model, &s.Status, &s.StartedAt, &s.EndedAt, &s.Used, &s.Window, &s.Measured,
		&s.Compactions, &s.HarnessSession, &s.Workdir, &s.ResumeAt, &s.ExitReason, &s.Handoff)
	if s.Window > 0 {
		s.Fraction = float64(s.Used) / float64(s.Window)
	}
	return s, err
}

func (m *Manager) write(ctx context.Context, s Session) error {
	_, err := m.db.ExecContext(ctx, `INSERT INTO sessions(`+cols+`) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
		ON CONFLICT(id) DO UPDATE SET status=excluded.status, ended_at=excluded.ended_at, used=excluded.used, window=excluded.window, measured=excluded.measured,
		compactions=excluded.compactions, harness_session=excluded.harness_session, resume_at=excluded.resume_at, exit_reason=excluded.exit_reason, handoff=excluded.handoff`,
		s.ID, s.TaskID, s.ProjectID, s.MemberID, s.Harness, s.Model, s.Status, s.StartedAt, s.EndedAt, s.Used, s.Window, s.Measured, s.Compactions,
		s.HarnessSession, s.Workdir, s.ResumeAt, s.ExitReason, s.Handoff)
	return err
}

// Get returns one session.
func (m *Manager) Get(ctx context.Context, id string) (Session, error) {
	s, err := scan(m.db.QueryRowContext(ctx, `SELECT `+cols+` FROM sessions WHERE id=?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return s, fmt.Errorf("session %s not found", id)
	}
	m.mu.Lock()
	_, s.Live = m.active[id]
	if _, ok := m.attached[id]; ok {
		s.Live, s.Attach = true, AttachCommand(id)
	}
	m.mu.Unlock()
	return s, err
}

// List returns sessions; live=true limits to running/paused ones.
func (m *Manager) List(ctx context.Context, projectID, taskID string, live bool) ([]Session, error) {
	q, args := `SELECT `+cols+` FROM sessions WHERE 1=1`, []any{}
	if projectID != "" {
		q, args = q+` AND project_id=?`, append(args, projectID)
	}
	if taskID != "" {
		q, args = q+` AND task_id=?`, append(args, taskID)
	}
	if live {
		q += ` AND status IN ('starting','running','rate_limited','paused')`
	}
	rows, err := m.db.QueryContext(ctx, q+` ORDER BY started_at DESC LIMIT 200`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Session{}
	m.mu.Lock()
	defer m.mu.Unlock()
	for rows.Next() {
		s, err := scan(rows)
		if err != nil {
			return nil, err
		}
		_, s.Live = m.active[s.ID]
		if _, ok := m.attached[s.ID]; ok {
			s.Live, s.Attach = true, AttachCommand(s.ID)
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// Output returns session output after seq; kind "raw" for the terminal view, "structured" for the rest.
func (m *Manager) Output(ctx context.Context, id string, after int, view string) ([]Line, error) {
	q := `SELECT seq,ts,kind,text FROM session_output WHERE session_id=? AND seq>?`
	switch view {
	case "terminal":
		q += ` AND kind='raw'`
	case "structured", "":
		q += ` AND kind!='raw'`
	}
	rows, err := m.db.QueryContext(ctx, q+` ORDER BY seq LIMIT 2000`, id, after)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Line{}
	for rows.Next() {
		var l Line
		if err := rows.Scan(&l.Seq, &l.TS, &l.Kind, &l.Text); err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

// Prepare builds the starting context for a task (FR-70): the task, its requirements,
// values, problem definition and incoming notes — not the whole project history.
func (m *Manager) Prepare(ctx context.Context, t queue.Task, p projects.Project) string {
	var b strings.Builder
	fmt.Fprintf(&b, "You are working on task %s for project %s.\n\n# Task: %s\n\n", t.ID, p.ID, t.Title)
	if t.Description != "" {
		b.WriteString(strings.TrimSpace(t.Description) + "\n\n")
	}
	if p.Purpose != "" {
		fmt.Fprintf(&b, "## Project purpose\n%s\n\n", p.Purpose)
	}
	if pr, _ := m.projects.Problem(ctx, p.ID); pr.Text != "" {
		fmt.Fprintf(&b, "## Problem definition (v%d)\n%s\n\n", pr.Version, pr.Text)
	}
	if vals, _ := m.projects.Values(ctx, p.ID, false); len(vals) > 0 {
		b.WriteString("## Project values\n")
		for _, v := range vals {
			if v.Status == "accepted" {
				fmt.Fprintf(&b, "- %s\n", v.Text)
			}
		}
		b.WriteString("\n")
	}
	if len(t.Reqs) > 0 {
		b.WriteString("## Requirements in scope\n")
		reqText := RequirementText(filepath.Join(p.Path, "docs", "requirements.md"), t.Reqs)
		for _, r := range t.Reqs {
			if txt := reqText[r]; txt != "" {
				fmt.Fprintf(&b, "- %s: %s\n", r, txt)
			} else {
				fmt.Fprintf(&b, "- %s\n", r)
			}
		}
		b.WriteString("\n")
	}
	if t.ContextNote != "" {
		fmt.Fprintf(&b, "## Notes from earlier work, answers and feedback\n%s\n\n", strings.TrimSpace(t.ContextNote))
	}
	b.WriteString("## How to finish\n- Make the change, with tests, in this working directory. Do not push; Courier handles git.\n- If something is ambiguous, ask with `gbi questions ask --task " + t.ID + " --blocking --title \"…\" --why \"…\"` and stop.\n- End with a short summary of what you did and how you tested it.\n")
	return b.String()
}

// HandoffSections are the parts a briefing must contain (FR-73).
var HandoffSections = []string{"objective", "current state", "decisions", "dead ends", "next steps", "pointers"}

// MissingHandoffSections returns FR-73 sections a briefing lacks (a completeness check).
func MissingHandoffSections(b string) []string {
	l := strings.ToLower(b)
	missing := []string{}
	for _, sec := range HandoffSections {
		if !strings.Contains(l, sec) {
			missing = append(missing, sec)
		}
	}
	return missing
}

var handoffRe = regexp.MustCompile(`(?s)<<<HANDOFF\s*\n(.*?)\n?HANDOFF>>>`)

var reqLineRe = regexp.MustCompile(`^- \*\*((?:FR|NFR)-\d+[a-z]?)\*\*: (.*)`)

// RequirementText extracts requirement sentences from a requirements.md.
func RequirementText(path string, ids []string) map[string]string {
	out := map[string]string{}
	b, err := os.ReadFile(path)
	if err != nil {
		return out
	}
	want := map[string]bool{}
	for _, id := range ids {
		want[id] = true
	}
	for _, l := range strings.Split(string(b), "\n") {
		if mm := reqLineRe.FindStringSubmatch(l); mm != nil && want[mm[1]] {
			out[mm[1]] = strings.ReplaceAll(mm[2], "**", "")
		}
	}
	return out
}

// WorktreePath is where a task's isolated workspace lives (FR-64).
func (m *Manager) WorktreePath(projectID, taskID string) string {
	return filepath.Join(m.cfg.Dir(), "worktrees", projectID, taskID)
}

// BranchFor returns the task branch name (ADR-003).
func BranchFor(t queue.Task) string {
	slug := projects.Slugify(t.Title)
	if len(slug) > 40 {
		slug = strings.Trim(slug[:40], "-")
	}
	return "t-" + strings.ToLower(strings.TrimPrefix(t.ID, "T-")) + "-" + slug
}

// Start runs a task with a persona. It returns once the harness has started.
func (m *Manager) Start(ctx context.Context, taskID, memberID string) (Session, error) {
	t, err := m.queue.Get(ctx, taskID)
	if err != nil {
		return Session{}, err
	}
	if t.Status == queue.Running {
		return Session{}, fmt.Errorf("%s is already running", taskID)
	}
	p, err := m.projects.Get(ctx, t.ProjectID)
	if err != nil {
		return Session{}, err
	}
	mem, err := m.pickMember(ctx, t, p, memberID)
	if err != nil {
		return Session{}, err
	}
	a, ok := harness.Get(mem.Harness)
	if !ok {
		return Session{}, fmt.Errorf("unknown harness %q for %s", mem.Harness, mem.Name)
	}
	hc := m.cfg.Harnesses[mem.Harness]
	if mem.Harness != "fake" && !hc.Enabled {
		return Session{}, fmt.Errorf("harness %s is disabled in settings", mem.Harness)
	}

	// Isolated workspace on the task branch (FR-64).
	wt := m.WorktreePath(p.ID, t.ID)
	branch := t.Branch
	if branch == "" {
		branch = BranchFor(t)
	}
	if _, err := os.Stat(wt); err != nil {
		if err := os.MkdirAll(filepath.Dir(wt), 0o755); err != nil {
			return Session{}, err
		}
		base := m.git.DefaultBranch(ctx, p.Path)
		if err := m.git.AddWorktree(ctx, p.Path, wt, branch, base); err != nil {
			return Session{}, fmt.Errorf("create worktree: %w", err)
		}
	}
	if t.Branch != branch {
		t, _ = m.queue.Update(ctx, queue.Task{ID: t.ID, Branch: branch})
	}

	// Standards and persona reach the agent through instruction files (FR-53).
	stds, _ := m.standards.Effective(ctx, p.ID)
	if _, err := standards.WriteFiles(wt, standards.Render(p.ID, stds, team.Persona(mem))); err != nil {
		return Session{}, fmt.Errorf("write instruction files: %w", err)
	}

	id, err := store.NextID(m.db, "S")
	if err != nil {
		return Session{}, err
	}
	model := firstNonEmpty(mem.Model, hc.Model)
	s := Session{ID: id, TaskID: t.ID, ProjectID: p.ID, MemberID: mem.ID, Harness: mem.Harness, Model: model, Status: "running", StartedAt: store.Now(), Workdir: wt}
	spec := harness.Spec{Dir: wt, Prompt: m.Prepare(ctx, t, p), Model: model, Binary: hc.Binary, Flags: hc.Flags, Window: hc.Window,
		Env: []string{"GBI_PERSONA=" + mem.ID, "GBI_TASK=" + t.ID, "GBI_SESSION=" + id, "GBI_URL=http://" + m.cfg.Listen}}
	if err := m.write(ctx, s); err != nil {
		return s, err
	}
	if m.AttachableAvailable(mem.Harness) {
		return m.startAttached(ctx, s, t, spec.Prompt, hc.Binary, spec.Env)
	}
	runCtx, cancel := context.WithCancel(context.Background())
	proc, err := harness.Start(runCtx, a, spec)
	if err != nil {
		cancel()
		s.Status, s.EndedAt, s.ExitReason = "failed", store.Now(), err.Error()
		_ = m.write(ctx, s)
		_, _ = m.queue.Fail(ctx, t.ID, err.Error())
		return s, err
	}
	act := &active{proc: proc, cancel: cancel}
	m.mu.Lock()
	m.active[id] = act
	m.mu.Unlock()
	if _, err := m.queue.SetStatus(ctx, t.ID, queue.Running, fmt.Sprintf("session %s started by %s on %s", id, mem.Name, mem.Harness)); err != nil {
		m.log.Warn("set running", "err", err)
	}
	pctx := access.Persona(context.Background(), mem.Name)
	m.hist.Must(pctx, history.Event{ProjectID: p.ID, Type: "session.started", Subject: id, Summary: fmt.Sprintf("%s started %s (%s)", mem.Name, t.ID, mem.Harness),
		Why: "dispatched from queue", Data: history.Data(map[string]any{"task": t.ID, "harness": mem.Harness, "workdir": wt, "branch": branch})})
	go m.consume(pctx, s, t, act)
	return s, nil
}

func (m *Manager) pickMember(ctx context.Context, t queue.Task, p projects.Project, memberID string) (team.Member, error) {
	if memberID == "" {
		memberID = t.Assignee
	}
	if memberID != "" {
		mem, err := m.team.Member(ctx, memberID)
		if err != nil {
			return mem, err
		}
		if mem.Kind != "ai" {
			return mem, fmt.Errorf("%s is a human; assign an AI persona to run the task", mem.Name)
		}
		if mem.ProjectID != "" && mem.ProjectID != p.ID {
			return mem, fmt.Errorf("%s is on %s's team, not %s's (team members are per project)", mem.Name, mem.ProjectID, p.ID)
		}
		return mem, nil
	}
	cands, err := m.team.ByRole(ctx, t.Role, p.ID)
	if err != nil {
		return team.Member{}, err
	}
	busy := map[string]bool{}
	live, _ := m.List(ctx, "", "", true)
	for _, s := range live {
		busy[s.MemberID] = true
	}
	for _, c := range cands {
		if !busy[c.ID] {
			return c, nil
		}
	}
	if len(cands) > 0 {
		return team.Member{}, fmt.Errorf("every %s persona is busy", t.Role)
	}
	return team.Member{}, fmt.Errorf("%s has no AI member with the %s role; add one in the project's Team tab", p.ID, t.Role)
}

func (m *Manager) appendLine(ctx context.Context, s *Session, act *active, kind, text string) {
	act.seq++
	ts := store.Now()
	_, _ = m.db.ExecContext(ctx, `INSERT INTO session_output(session_id,seq,ts,kind,text) VALUES(?,?,?,?,?)`, s.ID, act.seq, ts, kind, text)
	m.hist.Publish(history.Event{Type: "session.output", Subject: s.ID, ProjectID: s.ProjectID, TS: ts, Summary: text,
		Data: history.Data(map[string]any{"seq": act.seq, "kind": kind})})
}

func (m *Manager) consume(ctx context.Context, s Session, t queue.Task, act *active) {
	var resultErr bool
	var resultText string
	var limited *harness.RateLimit
	lastSave := time.Now()
	handoffAsked := false
	var textBuf strings.Builder
	for e := range act.proc.Events {
		switch e.Kind {
		case harness.KRaw:
			m.appendLine(ctx, &s, act, "raw", e.Text)
			continue
		case harness.KText:
			m.appendLine(ctx, &s, act, "text", e.Text)
			if m.cfg.HandoffsEnabledFor(s.ProjectID) {
				textBuf.WriteString(e.Text)
				if hb := handoffRe.FindStringSubmatch(textBuf.String()); hb != nil && s.Handoff == "" {
					s.Handoff = strings.TrimSpace(hb[1])
				}
			}
		case harness.KTool:
			m.appendLine(ctx, &s, act, "tool", e.Text)
		case harness.KSystem:
			m.appendLine(ctx, &s, act, "system", e.Text)
			if e.SessionID != "" {
				s.HarnessSession = e.SessionID
			}
		case harness.KError:
			m.appendLine(ctx, &s, act, "error", e.Text)
		case harness.KCompacted:
			s.Compactions++
			m.appendLine(ctx, &s, act, "system", "⟲ "+e.Text)
			_, _ = m.queue.Bump(ctx, t.ID, "compactions")
			m.hist.Must(ctx, history.Event{ProjectID: s.ProjectID, Type: "session.compacted", Subject: s.ID,
				Summary: fmt.Sprintf("Context compacted in %s (%s)", s.ID, t.ID), Why: "harness auto-compaction (MVP default)",
				Data: history.Data(map[string]any{"task": t.ID, "usedBefore": s.Used})})
		case harness.KLimit:
			// Keep a precise reset time from the harness over a guessed one from error text.
			precise := e.Limit.Kind != "result" && e.Limit.Kind != "stderr"
			if limited == nil || precise {
				limited = e.Limit
			}
			m.appendLine(ctx, &s, act, "system", "rate limited until "+e.Limit.ResetsAt.Local().Format("15:04"))
		case harness.KResult:
			resultErr, resultText = e.IsError, e.Text
			if e.SessionID != "" {
				s.HarnessSession = e.SessionID
			}
			m.appendLine(ctx, &s, act, "result", firstNonEmpty(e.Text, "turn finished"))
		}
		if e.Usage != nil {
			s.Used, s.Window, s.Measured = e.Usage.Used, e.Usage.Window, e.Usage.Measured
			// Experimental smart handoff (P9): ask for a briefing at the threshold.
			if !handoffAsked && m.cfg.HandoffsEnabledFor(s.ProjectID) && e.Usage.Fraction() >= m.cfg.Experimental.HandoffThreshold && m.Hooks.Handoff != nil {
				handoffAsked = true
				if msg := m.Hooks.Handoff(ctx, s, t); msg != "" && !act.proc.Send(msg) {
					m.appendLine(ctx, &s, act, "system", s.Harness+" does not accept input mid-session; it will compact instead of handing off")
				}
			}
			if time.Since(lastSave) > time.Second {
				_ = m.write(ctx, s)
				lastSave = time.Now()
				m.hist.Publish(history.Event{Type: "session.usage", Subject: s.ID, ProjectID: s.ProjectID, Data: history.Data(s)})
			}
		}
	}
	waitErr := act.proc.Wait()
	m.mu.Lock()
	delete(m.active, s.ID)
	m.mu.Unlock()
	act.cancel()
	s.EndedAt = store.Now()

	switch {
	case act.reason == "cancelled":
		s.Status, s.ExitReason = "cancelled", "cancelled by "+access.FromContext(ctx).Name
		_, _ = m.queue.SetStatus(ctx, t.ID, queue.Queued, "session cancelled")
	case limited != nil:
		// FR-65: pause and resume automatically when the limit resets.
		s.Status, s.ResumeAt, s.ExitReason = "rate_limited", limited.ResetsAt.UTC().Format(time.RFC3339), "subscription limit ("+limited.Kind+")"
		_ = m.write(ctx, s) // record the session first so readers never see a paused task with a running session
		_, _ = m.queue.SetStatus(ctx, t.ID, queue.Paused, "rate limited until "+limited.ResetsAt.Local().Format("15:04"))
		m.hist.Must(ctx, history.Event{ProjectID: s.ProjectID, Type: "session.rate_limited", Subject: s.ID, Summary: fmt.Sprintf("%s paused: %s limit", t.ID, s.Harness),
			Why: "resumes at " + limited.ResetsAt.Local().Format("15:04"), Data: history.Data(map[string]any{"task": t.ID, "resumeAt": s.ResumeAt})})
	case s.Handoff != "" && waitErr == nil && !resultErr:
		// Experimental smart handoff (FR-72–FR-74): the next session starts from the briefing.
		s.Status, s.ExitReason = "done", "handed off"
		cur, _ := m.queue.Get(ctx, t.ID)
		cur.ContextNote = strings.TrimSpace(cur.ContextNote + "\n\nHandoff briefing from " + s.ID + ":\n" + s.Handoff)
		_, _ = m.queue.Update(ctx, cur)
		_, _ = m.queue.Bump(ctx, t.ID, "handoffs")
		_ = m.write(ctx, s) // session first: readers never see a re-queued task with a running session
		_, _ = m.queue.SetStatus(ctx, t.ID, queue.Queued, "handed off at "+fmt.Sprintf("%.0f%%", s.Fraction*100)+"; a fresh session continues from the briefing")
		m.hist.Must(ctx, history.Event{ProjectID: s.ProjectID, Type: "handoff.written", Subject: s.ID, Summary: fmt.Sprintf("Handoff written for %s (%s)", t.ID, s.ID),
			Why: "experimental smart handoffs; context reached the threshold", Data: history.Data(map[string]any{"task": t.ID, "briefing": s.Handoff, "usedBefore": s.Used, "missingSections": MissingHandoffSections(s.Handoff)})})
	case act.reason == "blocked":
		s.Status, s.ExitReason = "done", "stopped for a blocking question"
	case waitErr != nil || resultErr:
		s.Status, s.ExitReason = "failed", firstNonEmpty(resultText, errString(waitErr))
		_ = m.write(ctx, s)
		_, _ = m.queue.Fail(ctx, t.ID, "session failed: "+trunc(s.ExitReason, 200))
		m.hist.Must(ctx, history.Event{ProjectID: s.ProjectID, Type: "session.failed", Subject: s.ID, Summary: fmt.Sprintf("%s failed (%s)", s.ID, t.ID), Why: trunc(s.ExitReason, 300)})
	default:
		s.Status, s.ExitReason = "done", "completed"
		cur, _ := m.queue.Get(ctx, t.ID)
		if cur.Status == queue.Blocked {
			s.ExitReason = "stopped for a blocking question"
			break
		}
		if m.Hooks.Completed != nil {
			if err := m.Hooks.Completed(ctx, s, cur); err != nil {
				m.hist.Must(ctx, history.Event{ProjectID: s.ProjectID, Type: "courier.error", Subject: t.ID, Summary: "Courier could not finish " + t.ID, Why: err.Error()})
				_ = m.write(ctx, s)
				_, _ = m.queue.SetStatus(ctx, t.ID, queue.Failed, "courier: "+err.Error())
				break
			}
		}
		_ = m.write(ctx, s) // session first: readers never see a task in review with a running session
		if cur2, _ := m.queue.Get(ctx, t.ID); cur2.Status == queue.Running {
			_, _ = m.queue.SetStatus(ctx, t.ID, queue.Review, "session completed: "+trunc(resultText, 160))
		}
		m.hist.Must(ctx, history.Event{ProjectID: s.ProjectID, Type: "session.completed", Subject: s.ID, Summary: fmt.Sprintf("%s finished %s", s.MemberID, t.ID), Why: trunc(resultText, 300)})
	}
	_ = m.write(ctx, s)
	m.hist.Publish(history.Event{Type: "session.usage", Subject: s.ID, ProjectID: s.ProjectID, Data: history.Data(s)})
}

// Send passes human input to a live session (FR-61).
func (m *Manager) Send(ctx context.Context, id, text string) error {
	m.mu.Lock()
	act, ok := m.active[id]
	_, att := m.attached[id]
	m.mu.Unlock()
	if att {
		if err := sendAttached(id, text); err != nil {
			return err
		}
		s, _ := m.Get(ctx, id)
		m.hist.Must(ctx, history.Event{ProjectID: s.ProjectID, Type: "session.input", Subject: id, Summary: "Input typed into attachable session " + id, Why: trunc(text, 200)})
		return nil
	}
	if !ok {
		return fmt.Errorf("session %s is not running", id)
	}
	if !act.proc.Send(text) {
		return errors.New("this harness does not accept input mid-session; add it as a task note instead")
	}
	s, _ := m.Get(ctx, id)
	m.appendLine(ctx, &s, act, "input", text)
	m.hist.Must(ctx, history.Event{ProjectID: s.ProjectID, Type: "session.input", Subject: id, Summary: "Input sent to " + id, Why: trunc(text, 200)})
	return nil
}

// Cancel stops a live session.
func (m *Manager) Cancel(ctx context.Context, id string) error {
	return m.stopActive(ctx, id, "cancelled")
}

// StopForQuestion ends a session whose task is now blocked on a question.
func (m *Manager) StopForQuestion(ctx context.Context, taskID string) {
	live, _ := m.List(ctx, "", taskID, true)
	for _, s := range live {
		_ = m.stopActive(ctx, s.ID, "blocked")
	}
}

func (m *Manager) stopActive(ctx context.Context, id, reason string) error {
	m.mu.Lock()
	if ch, ok := m.attached[id]; ok {
		m.mu.Unlock()
		ch <- reason
		return nil
	}
	act, ok := m.active[id]
	if ok {
		act.reason = reason
	}
	m.mu.Unlock()
	if !ok {
		return fmt.Errorf("session %s is not running", id)
	}
	act.proc.Cancel()
	if reason == "cancelled" {
		s, _ := m.Get(ctx, id)
		m.hist.Must(ctx, history.Event{ProjectID: s.ProjectID, Type: "session.cancelled", Subject: id, Summary: "Cancelled " + id})
	}
	return nil
}

// LiveCount returns the number of running harness processes.
func (m *Manager) LiveCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.active) + len(m.attached)
}

// Recover marks sessions orphaned by a restart and requeues their tasks (FR-82).
func (m *Manager) Recover(ctx context.Context) {
	rows, err := m.db.QueryContext(ctx, `SELECT id FROM sessions WHERE status IN ('starting','running')`)
	if err != nil {
		return
	}
	var ids []string
	for rows.Next() {
		var id string
		_ = rows.Scan(&id)
		ids = append(ids, id)
	}
	rows.Close()
	for _, id := range ids {
		_, _ = m.db.ExecContext(ctx, `UPDATE sessions SET status='failed', ended_at=?, exit_reason='interrupted by restart' WHERE id=?`, store.Now(), id)
	}
	if n, _ := m.queue.RecoverInterrupted(ctx); n > 0 {
		m.hist.Must(ctx, history.Event{Type: "system.recovered", Summary: fmt.Sprintf("Requeued %d interrupted task(s)", n), Why: "go-build-it restarted"})
	}
}

// Dispatcher ----------------------------------------------------------------

// SetAuto turns automatic dispatching on or off.
func (m *Manager) SetAuto(on bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if on == m.auto {
		return
	}
	m.auto = on
	if on {
		m.stop = make(chan struct{})
		go m.loop(m.stop)
	} else if m.stop != nil {
		close(m.stop)
	}
}

// Auto reports whether automatic dispatching is on.
func (m *Manager) Auto() bool { m.mu.Lock(); defer m.mu.Unlock(); return m.auto }

func (m *Manager) loop(stop chan struct{}) {
	tick := time.NewTicker(5 * time.Second)
	defer tick.Stop()
	for {
		select {
		case <-stop:
			return
		case <-tick.C:
			m.Tick(context.Background())
		}
	}
}

// Tick resumes rate-limited work whose limit has reset and starts ready tasks up to the concurrency limit.
func (m *Manager) Tick(ctx context.Context) {
	ctx = access.Persona(ctx, "dispatcher")
	now := time.Now().UTC().Format(time.RFC3339)
	rows, err := m.db.QueryContext(ctx, `SELECT id, task_id FROM sessions WHERE status='rate_limited' AND resume_at != '' AND resume_at <= ?`, now)
	if err == nil {
		var resume [][2]string
		for rows.Next() {
			var sid, tid string
			_ = rows.Scan(&sid, &tid)
			resume = append(resume, [2]string{sid, tid})
		}
		rows.Close()
		for _, r := range resume {
			_, _ = m.db.ExecContext(ctx, `UPDATE sessions SET status='done', exit_reason='limit reset; task resumed' WHERE id=?`, r[0])
			if t, err := m.queue.Get(ctx, r[1]); err == nil && t.Status == queue.Paused {
				t.ContextNote = strings.TrimSpace(t.ContextNote + "\n\nThe previous session was paused by a subscription limit. Continue from the current state of the working directory.")
				_, _ = m.queue.Update(ctx, t)
				_, _ = m.queue.SetStatus(ctx, r[1], queue.Queued, "subscription limit reset: resuming")
			}
		}
	}
	max := m.cfg.MaxConcurrent
	if max <= 0 {
		max = 1
	}
	ps, err := m.projects.List(ctx)
	if err != nil {
		return
	}
	for _, p := range ps {
		if m.LiveCount() >= max {
			return
		}
		if p.Stage == "Paused" || p.Stage == "Archived" {
			continue
		}
		t, err := m.queue.Next(ctx, p.ID)
		if err != nil || t == nil {
			continue
		}
		if _, err := m.Start(ctx, t.ID, ""); err != nil {
			m.log.Debug("dispatch", "task", t.ID, "err", err)
		}
	}
}

func firstNonEmpty(a ...string) string {
	for _, s := range a {
		if s != "" {
			return s
		}
	}
	return ""
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func trunc(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) > n {
		return s[:n] + "…"
	}
	return s
}
