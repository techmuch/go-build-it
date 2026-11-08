// Package rehearsals runs dress rehearsals for review gates (FR-115–FR-118, FR-116a, FR-117a).
package rehearsals

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/techmuch/go-build-it/internal/access"
	"github.com/techmuch/go-build-it/internal/history"
	"github.com/techmuch/go-build-it/internal/projects"
	"github.com/techmuch/go-build-it/internal/queue"
	"github.com/techmuch/go-build-it/internal/store"
)

// Step is one walkthrough step.
type Step struct {
	Title    string   `json:"title"`
	Scenario string   `json:"scenario"`
	Reqs     []string `json:"reqs"`
	Screen   string   `json:"screen"` // mockup path (tabletop) or app URL (UAT)
	Status   string   `json:"status"` // pending | ok | notes
}

// Finding is feedback on a step from any reviewer, human or AI (FR-117, FR-117a).
type Finding struct {
	ID            int64  `json:"id"`
	RehearsalID   string `json:"rehearsalId"`
	Step          int    `json:"step"`
	Author        string `json:"author"`
	AuthorKind    string `json:"authorKind"`
	Text          string `json:"text"`
	ChangeRequest string `json:"changeRequest"` // task ID created from it
	Status        string `json:"status"`
	CreatedAt     string `json:"createdAt"`
}

// Rehearsal is a guided walkthrough.
type Rehearsal struct {
	ID          string    `json:"id"`
	ProjectID   string    `json:"projectId"`
	MilestoneID string    `json:"milestoneId"`
	Title       string    `json:"title"`
	Kind        string    `json:"kind"`
	Status      string    `json:"status"`
	Steps       []Step    `json:"steps"`
	Current     int       `json:"current"`
	Host        string    `json:"host"`
	CreatedAt   string    `json:"createdAt"`
	DecidedAt   string    `json:"decidedAt"`
	Findings    []Finding `json:"findings"`
	Reviewers   []string  `json:"reviewers"`
}

// Service manages rehearsals.
type Service struct {
	db       *sql.DB
	hist     *history.Recorder
	queue    *queue.Service
	projects *projects.Service
	IsOwner  func(name, login string) bool
}

// New returns a rehearsal service.
func New(db *sql.DB, h *history.Recorder, q *queue.Service, p *projects.Service) *Service {
	return &Service{db: db, hist: h, queue: q, projects: p}
}

const cols = `id,project_id,milestone_id,title,kind,status,steps,current,host,created_at,decided_at`

func scan(sc interface{ Scan(...any) error }) (Rehearsal, error) {
	var r Rehearsal
	var steps string
	err := sc.Scan(&r.ID, &r.ProjectID, &r.MilestoneID, &r.Title, &r.Kind, &r.Status, &steps, &r.Current, &r.Host, &r.CreatedAt, &r.DecidedAt)
	_ = json.Unmarshal([]byte(steps), &r.Steps)
	if r.Steps == nil {
		r.Steps = []Step{}
	}
	return r, err
}

func (s *Service) write(ctx context.Context, r Rehearsal) error {
	steps, _ := json.Marshal(r.Steps)
	_, err := s.db.ExecContext(ctx, `INSERT INTO rehearsals(`+cols+`) VALUES(?,?,?,?,?,?,?,?,?,?,?)
		ON CONFLICT(id) DO UPDATE SET title=excluded.title, status=excluded.status, steps=excluded.steps, current=excluded.current, decided_at=excluded.decided_at`,
		r.ID, r.ProjectID, r.MilestoneID, r.Title, r.Kind, r.Status, string(steps), r.Current, r.Host, r.CreatedAt, r.DecidedAt)
	return err
}

// Create starts a rehearsal for a gate (FR-116).
func (s *Service) Create(ctx context.Context, r Rehearsal) (Rehearsal, error) {
	if r.ProjectID == "" || r.Title == "" || len(r.Steps) == 0 {
		return r, errors.New("project, title and at least one step are required")
	}
	if r.Kind != "uat" {
		r.Kind = "tabletop"
	}
	id, err := store.NextID(s.db, "G")
	if err != nil {
		return r, err
	}
	r.ID, r.Status, r.CreatedAt = id, "open", store.Now()
	if r.Host == "" {
		r.Host = access.FromContext(ctx).Name
	}
	for i := range r.Steps {
		if r.Steps[i].Status == "" {
			r.Steps[i].Status = "pending"
		}
		if r.Steps[i].Reqs == nil {
			r.Steps[i].Reqs = []string{}
		}
	}
	if err := s.write(ctx, r); err != nil {
		return r, err
	}
	s.hist.Must(ctx, history.Event{ProjectID: r.ProjectID, Type: "rehearsal.created", Subject: id, Summary: fmt.Sprintf("Rehearsal %s: %s (%d steps)", id, r.Title, len(r.Steps)), Why: r.Kind})
	return s.Get(ctx, id)
}

// Get returns a rehearsal with findings.
func (s *Service) Get(ctx context.Context, id string) (Rehearsal, error) {
	r, err := scan(s.db.QueryRowContext(ctx, `SELECT `+cols+` FROM rehearsals WHERE id=?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return r, fmt.Errorf("rehearsal %s not found", id)
	}
	if err != nil {
		return r, err
	}
	r.Findings, err = s.findings(ctx, `rehearsal_id=?`, id)
	seen := map[string]bool{}
	r.Reviewers = []string{}
	for _, f := range r.Findings {
		if !seen[f.Author] {
			seen[f.Author] = true
			r.Reviewers = append(r.Reviewers, f.Author)
		}
	}
	return r, err
}

func (s *Service) findings(ctx context.Context, where string, args ...any) ([]Finding, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id,rehearsal_id,step,author,author_kind,text,change_request,status,created_at FROM findings WHERE `+where+` ORDER BY id`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Finding{}
	for rows.Next() {
		var f Finding
		if err := rows.Scan(&f.ID, &f.RehearsalID, &f.Step, &f.Author, &f.AuthorKind, &f.Text, &f.ChangeRequest, &f.Status, &f.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

// List returns rehearsals.
func (s *Service) List(ctx context.Context, projectID string) ([]Rehearsal, error) {
	q, args := `SELECT `+cols+` FROM rehearsals`, []any{}
	if projectID != "" {
		q, args = q+` WHERE project_id=?`, append(args, projectID)
	}
	rows, err := s.db.QueryContext(ctx, q+` ORDER BY status='open' DESC, created_at DESC`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Rehearsal{}
	for rows.Next() {
		r, err := scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// AddFinding records feedback on a step from the current actor (FR-117a).
func (s *Service) AddFinding(ctx context.Context, id string, step int, text string) (Finding, error) {
	r, err := s.Get(ctx, id)
	if err != nil {
		return Finding{}, err
	}
	if step < 0 || step >= len(r.Steps) {
		return Finding{}, fmt.Errorf("step %d out of range", step)
	}
	if strings.TrimSpace(text) == "" {
		return Finding{}, errors.New("finding text is required")
	}
	a := access.FromContext(ctx)
	f := Finding{RehearsalID: id, Step: step, Author: a.Name, AuthorKind: a.Kind, Text: text, Status: "open", CreatedAt: store.Now()}
	res, err := s.db.ExecContext(ctx, `INSERT INTO findings(rehearsal_id,step,author,author_kind,text,change_request,status,created_at) VALUES(?,?,?,?,?,?,?,?)`,
		f.RehearsalID, f.Step, f.Author, f.AuthorKind, f.Text, "", f.Status, f.CreatedAt)
	if err != nil {
		return f, err
	}
	f.ID, _ = res.LastInsertId()
	r.Steps[step].Status = "notes"
	_ = s.write(ctx, r)
	s.hist.Must(ctx, history.Event{ProjectID: r.ProjectID, Type: "rehearsal.finding", Subject: id, Summary: fmt.Sprintf("%s on step %d (%s): %s", a.Name, step+1, r.Steps[step].Title, trunc(text, 120)),
		Data: history.Data(map[string]any{"step": step, "finding": f.ID})})
	return f, nil
}

// ChangeRequest turns a finding into a task: now=true makes it priority 1 so it runs during the rehearsal.
func (s *Service) ChangeRequest(ctx context.Context, findingID int64, now bool) (queue.Task, error) {
	fs, err := s.findings(ctx, `id=?`, findingID)
	if err != nil || len(fs) == 0 {
		return queue.Task{}, fmt.Errorf("finding %d not found", findingID)
	}
	f := fs[0]
	if f.ChangeRequest != "" {
		return s.queue.Get(ctx, f.ChangeRequest)
	}
	r, err := s.Get(ctx, f.RehearsalID)
	if err != nil {
		return queue.Task{}, err
	}
	st := r.Steps[f.Step]
	prio := 3
	if now {
		prio = 1
	}
	t, err := s.queue.Add(ctx, queue.Task{ProjectID: r.ProjectID, Title: "Rehearsal change: " + trunc(f.Text, 60), Role: "developer", Priority: prio, Reqs: st.Reqs,
		Description: fmt.Sprintf("From rehearsal %s step %d (%s), reported by %s:\n\n%s\n\nScenario: %s", r.ID, f.Step+1, st.Title, f.Author, f.Text, st.Scenario)})
	if err != nil {
		return t, err
	}
	_, _ = s.db.ExecContext(ctx, `UPDATE findings SET change_request=? WHERE id=?`, t.ID, findingID)
	s.hist.Must(ctx, history.Event{ProjectID: r.ProjectID, Type: "rehearsal.change_request", Subject: t.ID, Summary: fmt.Sprintf("Change request %s from %s", t.ID, r.ID),
		Why: map[bool]string{true: "do it now", false: "queued for after the rehearsal"}[now]})
	return t, nil
}

// Advance marks the current step and moves to another step.
func (s *Service) Advance(ctx context.Context, id string, step int, status string) (Rehearsal, error) {
	r, err := s.Get(ctx, id)
	if err != nil {
		return r, err
	}
	if r.Current < len(r.Steps) && status != "" {
		if r.Steps[r.Current].Status == "pending" || status == "ok" {
			r.Steps[r.Current].Status = status
		}
	}
	if step < 0 || step >= len(r.Steps) {
		return r, fmt.Errorf("step %d out of range", step)
	}
	r.Current = step
	if err := s.write(ctx, r); err != nil {
		return r, err
	}
	s.hist.Publish(history.Event{ProjectID: r.ProjectID, Type: "rehearsal.advanced", Subject: id, Summary: fmt.Sprintf("Step %d", step+1)})
	return r, nil
}

// Decide records the gate decision. Only the owner may approve, and only after every step is reviewed (FR-118, ADR-006).
func (s *Service) Decide(ctx context.Context, id, decision, note string) (Rehearsal, error) {
	r, err := s.Get(ctx, id)
	if err != nil {
		return r, err
	}
	a := access.FromContext(ctx)
	if a.Kind != "human" {
		return r, errors.New("a human must decide the gate")
	}
	if decision == "approve" {
		if s.IsOwner != nil && !s.IsOwner(a.Name, a.Login) {
			return r, fmt.Errorf("only an owner can approve a gate (ADR-006); %s is a reviewer", a.Name)
		}
		for i, st := range r.Steps {
			if st.Status == "pending" {
				return r, fmt.Errorf("step %d (%s) has not been reviewed", i+1, st.Title)
			}
		}
		r.Status = "approved"
	} else if decision == "changes" {
		r.Status = "changes"
	} else {
		return r, errors.New("decision must be approve or changes")
	}
	r.DecidedAt = store.Now()
	if err := s.write(ctx, r); err != nil {
		return r, err
	}
	if r.MilestoneID != "" {
		gs := map[string]string{"approved": "approved", "changes": "changes"}[r.Status]
		if _, err := s.projects.SetGate(ctx, r.MilestoneID, gs, note); err != nil {
			return r, err
		}
	}
	s.hist.Must(ctx, history.Event{ProjectID: r.ProjectID, Type: "rehearsal." + r.Status, Subject: id, Summary: fmt.Sprintf("%s %s gate %s", a.Name, r.Status, r.Title), Why: note})
	return r, nil
}

// FindingsForTask returns rehearsal feedback relevant to a task: change requests it implements and open
// findings in the same project (used by review assessments, FR-112).
func (s *Service) FindingsForTask(ctx context.Context, projectID, taskID string) ([]Finding, error) {
	return s.findings(ctx, `change_request=? OR rehearsal_id IN (SELECT id FROM rehearsals WHERE project_id=? AND status='open')`, taskID, projectID)
}

func trunc(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) > n {
		return s[:n] + "…"
	}
	return s
}
