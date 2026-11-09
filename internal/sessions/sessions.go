// Package sessions runs AI team members in herdr and dispatches work from the queue
// (FR-60–FR-69, FR-70–FR-76, FR-81, FR-82; ADR-009, ADR-010).
//
// Every member is a long-running harness in its own herdr pane, working in its own
// long-lived git worktree. A session is the span of one member working on one task
// inside its conversation: go-build-it writes .gbi-task.md, submits a short prompt,
// follows herdr's agent state and the harness transcript, and hands the result to
// Courier when the member settles. There is no other way to run a harness.
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
	"github.com/techmuch/go-build-it/internal/herdr"
	"github.com/techmuch/go-build-it/internal/history"
	"github.com/techmuch/go-build-it/internal/projects"
	"github.com/techmuch/go-build-it/internal/queue"
	"github.com/techmuch/go-build-it/internal/standards"
	"github.com/techmuch/go-build-it/internal/store"
	"github.com/techmuch/go-build-it/internal/team"
)

// Session is one member working on one task.
type Session struct {
	ID             string  `json:"id"`
	TaskID         string  `json:"taskId"`
	ProjectID      string  `json:"projectId"`
	MemberID       string  `json:"memberId"`
	Harness        string  `json:"harness"`
	Model          string  `json:"model"`
	Status         string  `json:"status"` // starting | running | blocked | rate_limited | done | failed | cancelled
	StartedAt      string  `json:"startedAt"`
	EndedAt        string  `json:"endedAt"`
	Used           int     `json:"used"`
	Window         int     `json:"window"`
	Measured       bool    `json:"measured"`
	Compactions    int     `json:"compactions"`
	HarnessSession string  `json:"harnessSession"` // the conversation ID
	Workdir        string  `json:"workdir"`
	ResumeAt       string  `json:"resumeAt"`
	ExitReason     string  `json:"exitReason"`
	Handoff        string  `json:"handoff"`
	PaneID         string  `json:"paneId"`
	Fraction       float64 `json:"fraction"`
	Live           bool    `json:"live"`
	HerdrState     string  `json:"herdrState,omitempty"`
}

// Line is one line of structured session output read from the transcript (FR-66).
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
	// Handoff returns the briefing request sent when smart handoffs are enabled (experimental).
	Handoff func(ctx context.Context, s Session, t queue.Task) string
}

// Manager runs members in herdr.
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

	// H talks to the gbi herdr session; nil until SetHerdr (agents cannot run without it).
	H *herdr.Client
	// GbiBin is the go-build-it executable (the fake harness runs `gbi dev fake-member`).
	GbiBin string
	// Poll is how often a running session is checked (tests shorten it).
	Poll time.Duration

	mu         sync.Mutex
	running    map[string]*run
	workspaces map[string]string // project ID → herdr workspace ID
	launching  map[string]*sync.Mutex
	auto       bool
	stop       chan struct{}
}

// New returns a session manager.
func New(db *sql.DB, h *history.Recorder, cfg *config.Config, p *projects.Service, q *queue.Service, t *team.Service, st *standards.Service, g *gitx.Git, log *slog.Logger) *Manager {
	exe, _ := os.Executable()
	return &Manager{db: db, hist: h, cfg: cfg, projects: p, queue: q, team: t, standards: st, git: g, log: log, GbiBin: exe, Poll: 500 * time.Millisecond,
		running: map[string]*run{}, workspaces: map[string]string{}, launching: map[string]*sync.Mutex{}}
}

// SetHerdr connects the manager to the gbi herdr session.
func (m *Manager) SetHerdr(c *herdr.Client) { m.mu.Lock(); m.H = c; m.mu.Unlock() }

func (m *Manager) herdr() (*herdr.Client, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.H == nil {
		return nil, errors.New("the gbi herdr session is not available; see Settings → herdr setup")
	}
	return m.H, nil
}

const cols = `id,task_id,project_id,member_id,harness,model,status,started_at,ended_at,used,window,measured,compactions,harness_session,workdir,resume_at,exit_reason,handoff,pane_id`

func scan(sc interface{ Scan(...any) error }) (Session, error) {
	var s Session
	err := sc.Scan(&s.ID, &s.TaskID, &s.ProjectID, &s.MemberID, &s.Harness, &s.Model, &s.Status, &s.StartedAt, &s.EndedAt, &s.Used, &s.Window, &s.Measured,
		&s.Compactions, &s.HarnessSession, &s.Workdir, &s.ResumeAt, &s.ExitReason, &s.Handoff, &s.PaneID)
	if s.Window > 0 {
		s.Fraction = float64(s.Used) / float64(s.Window)
	}
	return s, err
}

func (m *Manager) write(ctx context.Context, s Session) error {
	_, err := m.db.ExecContext(ctx, `INSERT INTO sessions(`+cols+`) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
		ON CONFLICT(id) DO UPDATE SET status=excluded.status, ended_at=excluded.ended_at, used=excluded.used, window=excluded.window, measured=excluded.measured,
		compactions=excluded.compactions, harness_session=excluded.harness_session, resume_at=excluded.resume_at, exit_reason=excluded.exit_reason,
		handoff=excluded.handoff, pane_id=excluded.pane_id, workdir=excluded.workdir`,
		s.ID, s.TaskID, s.ProjectID, s.MemberID, s.Harness, s.Model, s.Status, s.StartedAt, s.EndedAt, s.Used, s.Window, s.Measured, s.Compactions,
		s.HarnessSession, s.Workdir, s.ResumeAt, s.ExitReason, s.Handoff, s.PaneID)
	return err
}

// Get returns one session.
func (m *Manager) Get(ctx context.Context, id string) (Session, error) {
	s, err := scan(m.db.QueryRowContext(ctx, `SELECT `+cols+` FROM sessions WHERE id=?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return s, fmt.Errorf("session %q not found", id)
	}
	m.mu.Lock()
	if r, ok := m.running[id]; ok {
		s.Live, s.HerdrState = true, r.state
	}
	m.mu.Unlock()
	return s, err
}

// List returns sessions, newest first.
func (m *Manager) List(ctx context.Context, projectID, taskID string, live bool) ([]Session, error) {
	q, args := `SELECT `+cols+` FROM sessions WHERE 1=1`, []any{}
	if projectID != "" {
		q, args = q+` AND project_id=?`, append(args, projectID)
	}
	if taskID != "" {
		q, args = q+` AND task_id=?`, append(args, taskID)
	}
	if live {
		q += ` AND status IN ('starting','running','blocked','rate_limited','paused')`
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
		if r, ok := m.running[s.ID]; ok {
			s.Live, s.HerdrState = true, r.state
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// Output returns structured output after seq (FR-66). The terminal view streams from herdr.
func (m *Manager) Output(ctx context.Context, id string, after int, _ string) ([]Line, error) {
	rows, err := m.db.QueryContext(ctx, `SELECT seq,ts,kind,text FROM session_output WHERE session_id=? AND seq>? ORDER BY seq LIMIT 2000`, id, after)
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
	branch := t.Branch
	if branch == "" {
		branch = BranchFor(t)
	}
	b.WriteString("## How to finish\n- You are on branch `" + branch + "` in your own worktree. Make the change, with tests, here. Do not push or switch branches; Courier handles git.\n- If something is ambiguous, ask with `gbi questions ask --task " + t.ID + " --blocking --title \"…\" --why \"…\"` and stop.\n- End with a short summary of what you did and how you tested it.\n")
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

// BranchFor returns the task branch name (ADR-003).
func BranchFor(t queue.Task) string {
	slug := projects.Slugify(t.Title)
	if len(slug) > 40 {
		slug = strings.Trim(slug[:40], "-")
	}
	return "t-" + strings.ToLower(strings.TrimPrefix(t.ID, "T-")) + "-" + slug
}

// TaskDir returns where a task's work lives: the worktree of the member that last worked
// on it, or the project checkout (Courier pushes and merges by branch name from any
// checkout of the repository).
func (m *Manager) TaskDir(projectID, taskID string) string {
	var dir string
	_ = m.db.QueryRow(`SELECT workdir FROM sessions WHERE task_id=? AND workdir!='' ORDER BY started_at DESC LIMIT 1`, taskID).Scan(&dir)
	if dir != "" {
		if _, err := os.Stat(dir); err == nil {
			return dir
		}
	}
	if p, err := m.projects.Get(context.Background(), projectID); err == nil {
		return p.Path
	}
	return ""
}

// WorktreePath is kept for callers that name a task's directory (Courier).
func (m *Manager) WorktreePath(projectID, taskID string) string { return m.TaskDir(projectID, taskID) }

// TaskFile is the per-task brief in a member's worktree; it is never committed.
const TaskFile = ".gbi-task.md"

// Start dispatches a task to a member. It returns once the session is recorded; the
// member's pane, harness and prompt are handled in the background.
func (m *Manager) Start(ctx context.Context, taskID, memberID string) (Session, error) {
	h, err := m.herdr()
	if err != nil {
		return Session{}, err
	}
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
	prof, ok := harness.Get(mem.Harness)
	if !ok {
		return Session{}, fmt.Errorf("unknown harness %q for %s", mem.Harness, mem.Name)
	}
	hc := m.cfg.Harnesses[mem.Harness]
	if mem.Harness != "fake" && !hc.Enabled {
		return Session{}, fmt.Errorf("harness %s is disabled in settings", mem.Harness)
	}
	branch := t.Branch
	if branch == "" {
		branch = BranchFor(t)
	}
	if t.Branch != branch {
		t, _ = m.queue.Update(ctx, queue.Task{ID: t.ID, Branch: branch})
		t.Branch = branch
	}
	wt, err := m.prepareWorktree(ctx, p, mem, branch)
	if err != nil {
		return Session{}, err
	}
	// Standards and persona reach the agent through instruction files (FR-53); the task
	// itself goes in .gbi-task.md, read on request (long prompts are rejected by herdr on macOS).
	stds, _ := m.standards.Effective(ctx, p.ID)
	if _, err := standards.WriteFiles(wt, standards.Render(p.ID, stds, team.Persona(mem))); err != nil {
		return Session{}, fmt.Errorf("write instruction files: %w", err)
	}
	prompt := m.Prepare(ctx, t, p)
	if mem.ConversationID == "" {
		prompt = m.withNotes(ctx, mem, prompt)
	}
	if err := os.WriteFile(filepath.Join(wt, TaskFile), []byte(prompt), 0o644); err != nil {
		return Session{}, err
	}
	id, err := store.NextID(m.db, "S")
	if err != nil {
		return Session{}, err
	}
	model := firstNonEmpty(mem.Model, hc.Model)
	window := harness.EstimateWindow(model, firstPositive(hc.Window, prof.Window))
	s := Session{ID: id, TaskID: t.ID, ProjectID: p.ID, MemberID: mem.ID, Harness: mem.Harness, Model: model, Status: "starting", StartedAt: store.Now(),
		Workdir: wt, Window: window, Measured: prof.UsageMeasured, HarnessSession: mem.ConversationID, PaneID: mem.PaneID}
	if err := m.write(ctx, s); err != nil {
		return s, err
	}
	if _, err := m.queue.SetStatus(ctx, t.ID, queue.Running, fmt.Sprintf("session %s started by %s on %s", id, mem.Name, mem.Harness)); err != nil {
		m.log.Warn("set running", "err", err)
	}
	pctx := access.Persona(context.Background(), mem.Name)
	m.hist.Must(pctx, history.Event{ProjectID: p.ID, Type: "session.started", Subject: id, Summary: fmt.Sprintf("%s started %s (%s, herdr)", mem.Name, t.ID, mem.Harness),
		Why: "dispatched from queue", Data: history.Data(map[string]any{"task": t.ID, "harness": mem.Harness, "workdir": wt, "branch": branch})})
	rctx, cancel := context.WithCancel(pctx)
	r := &run{s: s, t: t, p: p, mem: mem, prof: prof, cancel: cancel, h: h, state: "starting"}
	m.mu.Lock()
	m.running[id] = r
	m.mu.Unlock()
	go m.drive(rctx, r)
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
		if mem.ProjectID == "" {
			return mem, fmt.Errorf("%s is an archetype; add it to %s's team first", mem.Name, p.ID)
		}
		if mem.ProjectID != p.ID {
			return mem, fmt.Errorf("%s is on %s's team, not %s's (team members are per project)", mem.Name, mem.ProjectID, p.ID)
		}
		if m.memberBusy(mem.ID) {
			return mem, fmt.Errorf("%s is busy with another task (one task at a time)", mem.Name)
		}
		return mem, nil
	}
	cands, err := m.team.ByRole(ctx, t.Role, p.ID)
	if err != nil {
		return team.Member{}, err
	}
	for _, c := range cands {
		if c.Harness != "courier" && !m.memberBusy(c.ID) {
			return c, nil
		}
	}
	if len(cands) > 0 {
		return team.Member{}, fmt.Errorf("every %s member on %s is busy", t.Role, p.ID)
	}
	return team.Member{}, fmt.Errorf("%s has no AI member with the %s role; add one in the project's Team tab", p.ID, t.Role)
}

func (m *Manager) memberBusy(memberID string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, r := range m.running {
		if r.mem.ID == memberID {
			return true
		}
	}
	return false
}

// withNotes adds the member's notes from finished tasks when a new conversation starts (FR-76).
func (m *Manager) withNotes(ctx context.Context, mem team.Member, prompt string) string {
	rows, err := m.db.QueryContext(ctx, `SELECT task_id, note FROM member_notes WHERE member_id=? ORDER BY id DESC LIMIT 8`, mem.ID)
	if err != nil {
		return prompt
	}
	defer rows.Close()
	var notes []string
	for rows.Next() {
		var tid, n string
		if rows.Scan(&tid, &n) == nil {
			notes = append(notes, fmt.Sprintf("- %s: %s", tid, n))
		}
	}
	if len(notes) == 0 {
		return prompt
	}
	return prompt + "\n## Your notes from earlier tasks on this project (newest first)\n" + strings.Join(notes, "\n") + "\n"
}

// AddNote records something a member learned (FR-76).
func (m *Manager) AddNote(ctx context.Context, memberID, taskID, note string) {
	note = strings.TrimSpace(note)
	if note == "" {
		return
	}
	if len(note) > 600 {
		note = note[:600] + "…"
	}
	_, _ = m.db.ExecContext(ctx, `INSERT INTO member_notes(member_id,task_id,note,created_at) VALUES(?,?,?,?)`, memberID, taskID, note, store.Now())
}

// Send passes human input to a member mid-task (FR-61).
func (m *Manager) Send(ctx context.Context, id, text string) error {
	m.mu.Lock()
	r, ok := m.running[id]
	m.mu.Unlock()
	if !ok {
		return fmt.Errorf("session %s is not running", id)
	}
	if _, err := r.h.Submit(ctx, r.paneID(), r.agentName(), r.prof.Kind != "", text); err != nil {
		return err
	}
	m.hist.Must(ctx, history.Event{ProjectID: r.s.ProjectID, Type: "session.input", Subject: id, Summary: "Input sent to " + r.mem.Name + " (" + id + ")", Why: trunc(text, 200)})
	return nil
}

// Cancel interrupts a member's current task.
func (m *Manager) Cancel(ctx context.Context, id string) error {
	if err := m.stopRun(ctx, id, "cancelled"); err != nil {
		return err
	}
	s, _ := m.Get(ctx, id)
	m.hist.Must(ctx, history.Event{ProjectID: s.ProjectID, Type: "session.cancelled", Subject: id, Summary: "Cancelled " + id})
	return nil
}

// StopForQuestion notes that a task's session stopped for a blocking question; the
// member was told to stop and goes idle, and the session ends then.
func (m *Manager) StopForQuestion(ctx context.Context, taskID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, r := range m.running {
		if r.t.ID == taskID {
			r.reason = "blocked"
		}
	}
}

func (m *Manager) stopRun(ctx context.Context, id, reason string) error {
	m.mu.Lock()
	r, ok := m.running[id]
	if ok {
		r.reason = reason
	}
	m.mu.Unlock()
	if !ok {
		return fmt.Errorf("session %s is not running", id)
	}
	// Interrupt the agent's turn (Esc for known kinds; the fake member reads /cancel).
	if r.prof.Kind != "" {
		_ = r.h.SendKeys(ctx, r.agentTarget(), "esc")
	} else if p := r.paneID(); p != "" {
		_ = r.h.Call(ctx, herdr.MethodPaneSendText, map[string]any{"pane_id": p, "text": "/cancel\n"}, nil)
	}
	r.cancel()
	return nil
}

// LiveCount returns the number of members working on a task.
func (m *Manager) LiveCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.running)
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

func firstPositive(a ...int) int {
	for _, n := range a {
		if n > 0 {
			return n
		}
	}
	return 0
}

func trunc(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
