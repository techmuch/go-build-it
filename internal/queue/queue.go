// Package queue is the persistent per-project task queue (FR-80–FR-82).
package queue

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/techmuch/go-build-it/internal/history"
	"github.com/techmuch/go-build-it/internal/store"
)

// Statuses.
const (
	Queued  = "queued"
	Running = "running"
	Blocked = "blocked" // waiting on a blocking question or gate
	Review  = "review"
	Done    = "done"
	Failed  = "failed"
	Paused  = "paused"
)

// Task is a unit of work.
type Task struct {
	ID          string   `json:"id"`
	ProjectID   string   `json:"projectId"`
	MilestoneID string   `json:"milestoneId"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Role        string   `json:"role"`
	Assignee    string   `json:"assignee"`
	Priority    int      `json:"priority"`
	Status      string   `json:"status"`
	Deps        []string `json:"deps"`
	Reqs        []string `json:"reqs"`
	Retries     int      `json:"retries"`
	MaxRetries  int      `json:"maxRetries"`
	Compactions int      `json:"compactions"`
	Handoffs    int      `json:"handoffs"`
	Branch      string   `json:"branch"`
	PRURL       string   `json:"prUrl"`
	ContextNote string   `json:"contextNote"`
	CreatedAt   string   `json:"createdAt"`
	UpdatedAt   string   `json:"updatedAt"`
	// Derived
	Ready     bool   `json:"ready"`
	BlockedBy string `json:"blockedBy,omitempty"`
}

// Service manages tasks.
type Service struct {
	db   *sql.DB
	hist *history.Recorder
	// GateOpen reports whether a milestone's predecessors' gates allow work (FR-115). Optional.
	GateOpen func(ctx context.Context, milestoneID string) (bool, string)
}

// New returns a queue service.
func New(db *sql.DB, h *history.Recorder) *Service { return &Service{db: db, hist: h} }

const cols = `id,project_id,milestone_id,title,description,role,assignee,priority,status,deps,reqs,retries,max_retries,compactions,handoffs,branch,pr_url,context_note,created_at,updated_at`

func scan(sc interface{ Scan(...any) error }) (Task, error) {
	var t Task
	var deps, reqs string
	err := sc.Scan(&t.ID, &t.ProjectID, &t.MilestoneID, &t.Title, &t.Description, &t.Role, &t.Assignee, &t.Priority, &t.Status, &deps, &reqs,
		&t.Retries, &t.MaxRetries, &t.Compactions, &t.Handoffs, &t.Branch, &t.PRURL, &t.ContextNote, &t.CreatedAt, &t.UpdatedAt)
	_ = json.Unmarshal([]byte(deps), &t.Deps)
	_ = json.Unmarshal([]byte(reqs), &t.Reqs)
	if t.Deps == nil {
		t.Deps = []string{}
	}
	if t.Reqs == nil {
		t.Reqs = []string{}
	}
	return t, err
}

// Add queues a new task.
func (s *Service) Add(ctx context.Context, t Task) (Task, error) {
	if t.ProjectID == "" || strings.TrimSpace(t.Title) == "" {
		return t, errors.New("project and title are required")
	}
	id, err := store.NextID(s.db, "T")
	if err != nil {
		return t, err
	}
	t.ID = id
	if t.Role == "" {
		t.Role = "developer"
	}
	if t.Priority == 0 {
		t.Priority = 2
	}
	if t.MaxRetries == 0 {
		t.MaxRetries = 2
	}
	t.Status = Queued
	if t.Deps == nil {
		t.Deps = []string{}
	}
	if t.Reqs == nil {
		t.Reqs = []string{}
	}
	for _, d := range t.Deps {
		if _, err := s.Get(ctx, d); err != nil {
			return t, fmt.Errorf("dependency %s: %w", d, err)
		}
	}
	t.CreatedAt, t.UpdatedAt = store.Now(), store.Now()
	if err := s.write(ctx, t); err != nil {
		return t, err
	}
	s.hist.Must(ctx, history.Event{ProjectID: t.ProjectID, Type: "task.created", Subject: t.ID, Summary: fmt.Sprintf("%s %s", t.ID, t.Title), Why: "role " + t.Role})
	return t, nil
}

func (s *Service) write(ctx context.Context, t Task) error {
	deps, _ := json.Marshal(t.Deps)
	reqs, _ := json.Marshal(t.Reqs)
	_, err := s.db.ExecContext(ctx, `INSERT INTO tasks(`+cols+`) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
		ON CONFLICT(id) DO UPDATE SET milestone_id=excluded.milestone_id, title=excluded.title, description=excluded.description, role=excluded.role,
		assignee=excluded.assignee, priority=excluded.priority, status=excluded.status, deps=excluded.deps, reqs=excluded.reqs, retries=excluded.retries,
		max_retries=excluded.max_retries, compactions=excluded.compactions, handoffs=excluded.handoffs, branch=excluded.branch, pr_url=excluded.pr_url,
		context_note=excluded.context_note, updated_at=excluded.updated_at`,
		t.ID, t.ProjectID, t.MilestoneID, t.Title, t.Description, t.Role, t.Assignee, t.Priority, t.Status, string(deps), string(reqs),
		t.Retries, t.MaxRetries, t.Compactions, t.Handoffs, t.Branch, t.PRURL, t.ContextNote, t.CreatedAt, t.UpdatedAt)
	return err
}

// Get returns one task.
func (s *Service) Get(ctx context.Context, id string) (Task, error) {
	t, err := scan(s.db.QueryRowContext(ctx, `SELECT `+cols+` FROM tasks WHERE id=?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return t, fmt.Errorf("task %s not found", id)
	}
	return t, err
}

// Filter for List.
type Filter struct {
	ProjectID   string
	Status      string
	Role        string
	MilestoneID string
}

// List returns tasks ordered by status, priority and age, with readiness computed.
func (s *Service) List(ctx context.Context, f Filter) ([]Task, error) {
	var where []string
	var args []any
	if f.ProjectID != "" {
		where, args = append(where, "project_id=?"), append(args, f.ProjectID)
	}
	if f.Status != "" {
		where, args = append(where, "status=?"), append(args, f.Status)
	}
	if f.Role != "" {
		where, args = append(where, "role=?"), append(args, f.Role)
	}
	if f.MilestoneID != "" {
		where, args = append(where, "milestone_id=?"), append(args, f.MilestoneID)
	}
	q := `SELECT ` + cols + ` FROM tasks`
	if len(where) > 0 {
		q += " WHERE " + strings.Join(where, " AND ")
	}
	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	out := []Task{}
	for rows.Next() {
		t, err := scan(rows)
		if err != nil {
			rows.Close()
			return nil, err
		}
		out = append(out, t)
	}
	rows.Close()
	status := map[string]string{}
	allRows, err := s.db.QueryContext(ctx, `SELECT id,status FROM tasks`)
	if err == nil {
		for allRows.Next() {
			var id, st string
			_ = allRows.Scan(&id, &st)
			status[id] = st
		}
		allRows.Close()
	}
	for i := range out {
		s.derive(ctx, &out[i], status)
	}
	sort.SliceStable(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if rank(a) != rank(b) {
			return rank(a) < rank(b)
		}
		if a.Priority != b.Priority {
			return a.Priority < b.Priority
		}
		return a.CreatedAt < b.CreatedAt
	})
	return out, nil
}

func rank(t Task) int {
	switch t.Status {
	case Running:
		return 0
	case Queued:
		if t.Ready {
			return 1
		}
		return 2
	case Blocked, Paused:
		return 3
	case Review:
		return 4
	case Failed:
		return 5
	}
	return 6
}

func (s *Service) derive(ctx context.Context, t *Task, status map[string]string) {
	if t.Status != Queued {
		return
	}
	for _, d := range t.Deps {
		if status[d] != Done {
			t.BlockedBy = "waits on " + d
			return
		}
	}
	if s.GateOpen != nil && t.MilestoneID != "" {
		if ok, why := s.GateOpen(ctx, t.MilestoneID); !ok {
			t.BlockedBy = why
			return
		}
	}
	t.Ready = true
}

// Next returns the highest-priority ready task for a project and role set, or nil.
func (s *Service) Next(ctx context.Context, projectID string) (*Task, error) {
	ts, err := s.List(ctx, Filter{ProjectID: projectID, Status: Queued})
	if err != nil {
		return nil, err
	}
	for _, t := range ts {
		if t.Ready {
			t := t
			return &t, nil
		}
	}
	return nil, nil
}

// SetStatus changes status and records why.
func (s *Service) SetStatus(ctx context.Context, id, status, why string) (Task, error) {
	t, err := s.Get(ctx, id)
	if err != nil {
		return t, err
	}
	if t.Status == status {
		return t, nil
	}
	old := t.Status
	t.Status, t.UpdatedAt = status, store.Now()
	if err := s.write(ctx, t); err != nil {
		return t, err
	}
	s.hist.Must(ctx, history.Event{ProjectID: t.ProjectID, Type: "task." + status, Subject: id, Summary: fmt.Sprintf("%s %s → %s", id, old, status), Why: why})
	return t, nil
}

// Update edits task fields (title, description, priority, role, assignee, deps, milestone, context note).
func (s *Service) Update(ctx context.Context, t Task) (Task, error) {
	cur, err := s.Get(ctx, t.ID)
	if err != nil {
		return cur, err
	}
	if t.Title != "" {
		cur.Title = t.Title
	}
	if t.Description != "" {
		cur.Description = t.Description
	}
	if t.Priority != 0 {
		cur.Priority = t.Priority
	}
	if t.Role != "" {
		cur.Role = t.Role
	}
	if t.Assignee != "" {
		cur.Assignee = t.Assignee
	}
	if t.Deps != nil {
		cur.Deps = t.Deps
	}
	if t.Reqs != nil {
		cur.Reqs = t.Reqs
	}
	if t.MilestoneID != "" {
		cur.MilestoneID = t.MilestoneID
	}
	if t.ContextNote != "" {
		cur.ContextNote = t.ContextNote
	}
	if t.Branch != "" {
		cur.Branch = t.Branch
	}
	if t.PRURL != "" {
		cur.PRURL = t.PRURL
	}
	cur.UpdatedAt = store.Now()
	if err := s.write(ctx, cur); err != nil {
		return cur, err
	}
	s.hist.Must(ctx, history.Event{ProjectID: cur.ProjectID, Type: "task.updated", Subject: cur.ID, Summary: "Updated " + cur.ID})
	return cur, nil
}

// Bump increments a counter field: "retries", "compactions" or "handoffs".
func (s *Service) Bump(ctx context.Context, id, field string) (Task, error) {
	switch field {
	case "retries", "compactions", "handoffs":
	default:
		return Task{}, fmt.Errorf("unknown counter %q", field)
	}
	if _, err := s.db.ExecContext(ctx, `UPDATE tasks SET `+field+`=`+field+`+1, updated_at=? WHERE id=?`, store.Now(), id); err != nil {
		return Task{}, err
	}
	return s.Get(ctx, id)
}

// Fail marks a session failure: requeue while retries remain, else failed.
func (s *Service) Fail(ctx context.Context, id, why string) (Task, error) {
	t, err := s.Bump(ctx, id, "retries")
	if err != nil {
		return t, err
	}
	if t.Retries <= t.MaxRetries {
		return s.SetStatus(ctx, id, Queued, fmt.Sprintf("retry %d/%d: %s", t.Retries, t.MaxRetries, why))
	}
	return s.SetStatus(ctx, id, Failed, why)
}

// RecoverInterrupted requeues tasks left running by a crash or restart (FR-82).
func (s *Service) RecoverInterrupted(ctx context.Context) (int, error) {
	ts, err := s.List(ctx, Filter{Status: Running})
	if err != nil {
		return 0, err
	}
	for _, t := range ts {
		if _, err := s.SetStatus(ctx, t.ID, Queued, "go-build-it restarted while the session was running; resuming"); err != nil {
			return 0, err
		}
	}
	return len(ts), nil
}

// SplitCandidates returns tasks that needed repeated compactions or handoffs (FR-75).
func (s *Service) SplitCandidates(ctx context.Context, projectID string, threshold int) ([]Task, error) {
	ts, err := s.List(ctx, Filter{ProjectID: projectID})
	if err != nil {
		return nil, err
	}
	var out []Task
	for _, t := range ts {
		if t.Status != Done && t.Compactions+t.Handoffs >= threshold {
			out = append(out, t)
		}
	}
	return out, nil
}
