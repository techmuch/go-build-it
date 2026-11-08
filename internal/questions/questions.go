// Package questions lets team members ask humans questions (FR-100–FR-105).
package questions

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/techmuch/go-build-it/internal/access"
	"github.com/techmuch/go-build-it/internal/history"
	"github.com/techmuch/go-build-it/internal/queue"
	"github.com/techmuch/go-build-it/internal/store"
)

// Option is a structured answer choice (FR-103).
type Option struct {
	Key         string `json:"key"`
	Title       string `json:"title"`
	Detail      string `json:"detail"`
	Recommended bool   `json:"recommended"`
}

// Attachment is a file or link carried by a question (FR-102, FR-102a).
type Attachment struct {
	Name string `json:"name"`
	Kind string `json:"kind"` // html | image | diff | json | log | link | text
	Path string `json:"path,omitempty"`
	URL  string `json:"url,omitempty"`
	Body string `json:"body,omitempty"`
}

// Question is a request for human input.
type Question struct {
	ID             string       `json:"id"`
	ProjectID      string       `json:"projectId"`
	TaskID         string       `json:"taskId"`
	SessionID      string       `json:"sessionId"`
	Asker          string       `json:"asker"`
	Title          string       `json:"title"`
	Why            string       `json:"why"`
	Reqs           []string     `json:"reqs"`
	Blocking       bool         `json:"blocking"`
	Assumption     string       `json:"assumption"`
	Recommendation string       `json:"recommendation"`
	Options        []Option     `json:"options"`
	Attachments    []Attachment `json:"attachments"`
	Status         string       `json:"status"`
	Answer         string       `json:"answer"`
	Choice         string       `json:"choice"`
	AnsweredBy     string       `json:"answeredBy"`
	AnsweredAt     string       `json:"answeredAt"`
	CreatedAt      string       `json:"createdAt"`
}

// Hooks connect answers to sessions and values.
type Hooks struct {
	StopSession   func(ctx context.Context, taskID string)
	SendToSession func(ctx context.Context, sessionID, text string) error
	ProposeValue  func(ctx context.Context, projectID, text, source string) error
}

// Service manages questions.
type Service struct {
	db    *sql.DB
	hist  *history.Recorder
	queue *queue.Service
	Hooks Hooks
}

// New returns a question service.
func New(db *sql.DB, h *history.Recorder, q *queue.Service) *Service {
	return &Service{db: db, hist: h, queue: q}
}

const cols = `id,project_id,task_id,session_id,asker,title,why,reqs,blocking,assumption,recommendation,options,attachments,status,answer,choice,answered_by,answered_at,created_at`

func scan(sc interface{ Scan(...any) error }) (Question, error) {
	var q Question
	var reqs, opts, atts string
	err := sc.Scan(&q.ID, &q.ProjectID, &q.TaskID, &q.SessionID, &q.Asker, &q.Title, &q.Why, &reqs, &q.Blocking, &q.Assumption, &q.Recommendation,
		&opts, &atts, &q.Status, &q.Answer, &q.Choice, &q.AnsweredBy, &q.AnsweredAt, &q.CreatedAt)
	_ = json.Unmarshal([]byte(reqs), &q.Reqs)
	_ = json.Unmarshal([]byte(opts), &q.Options)
	_ = json.Unmarshal([]byte(atts), &q.Attachments)
	if q.Reqs == nil {
		q.Reqs = []string{}
	}
	if q.Options == nil {
		q.Options = []Option{}
	}
	if q.Attachments == nil {
		q.Attachments = []Attachment{}
	}
	return q, err
}

// Ask records a question. A blocking question pauses its task (FR-101).
func (s *Service) Ask(ctx context.Context, q Question) (Question, error) {
	if strings.TrimSpace(q.Title) == "" {
		return q, errors.New("a question needs a title")
	}
	if q.TaskID != "" && q.ProjectID == "" {
		if t, err := s.queue.Get(ctx, q.TaskID); err == nil {
			q.ProjectID = t.ProjectID
		}
	}
	if q.ProjectID == "" {
		return q, errors.New("a question needs a project or task")
	}
	if !q.Blocking && q.Assumption == "" {
		return q, errors.New("a non-blocking question must state the assumption work continues on")
	}
	id, err := store.NextID(s.db, "Q")
	if err != nil {
		return q, err
	}
	q.ID, q.Status, q.CreatedAt = id, "open", store.Now()
	if q.Asker == "" {
		q.Asker = access.FromContext(ctx).Name
	}
	reqs, _ := json.Marshal(nz(q.Reqs))
	opts, _ := json.Marshal(nzo(q.Options))
	atts, _ := json.Marshal(nza(q.Attachments))
	_, err = s.db.ExecContext(ctx, `INSERT INTO questions(`+cols+`) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		q.ID, q.ProjectID, q.TaskID, q.SessionID, q.Asker, q.Title, q.Why, string(reqs), q.Blocking, q.Assumption, q.Recommendation, string(opts), string(atts),
		q.Status, "", "", "", "", q.CreatedAt)
	if err != nil {
		return q, err
	}
	kind := "non-blocking"
	if q.Blocking {
		kind = "blocking"
	}
	s.hist.Must(ctx, history.Event{ProjectID: q.ProjectID, Type: "question.asked", Subject: q.ID, Summary: fmt.Sprintf("%s asked (%s): %s", q.Asker, kind, q.Title), Why: q.Why,
		Data: history.Data(map[string]any{"task": q.TaskID, "blocking": q.Blocking})})
	if q.Blocking && q.TaskID != "" {
		if _, err := s.queue.SetStatus(ctx, q.TaskID, queue.Blocked, "waiting on "+q.ID); err == nil && s.Hooks.StopSession != nil {
			go s.Hooks.StopSession(context.WithoutCancel(ctx), q.TaskID)
		}
	}
	return q, nil
}

func nz(v []string) []string {
	if v == nil {
		return []string{}
	}
	return v
}
func nzo(v []Option) []Option {
	if v == nil {
		return []Option{}
	}
	return v
}
func nza(v []Attachment) []Attachment {
	if v == nil {
		return []Attachment{}
	}
	return v
}

// Get returns one question.
func (s *Service) Get(ctx context.Context, id string) (Question, error) {
	q, err := scan(s.db.QueryRowContext(ctx, `SELECT `+cols+` FROM questions WHERE id=?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return q, fmt.Errorf("question %s not found", id)
	}
	return q, err
}

// List returns questions: open ones with blocking first (FR-104), or by status.
func (s *Service) List(ctx context.Context, projectID, status string) ([]Question, error) {
	q, args := `SELECT `+cols+` FROM questions WHERE 1=1`, []any{}
	if projectID != "" {
		q, args = q+` AND project_id=?`, append(args, projectID)
	}
	if status == "" {
		status = "open"
	}
	if status != "all" {
		q, args = q+` AND status=?`, append(args, status)
	}
	rows, err := s.db.QueryContext(ctx, q+` ORDER BY status='open' DESC, blocking DESC, created_at ASC`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Question{}
	for rows.Next() {
		qq, err := scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, qq)
	}
	return out, rows.Err()
}

// Answer records an answer and sends it back to the asking work (FR-105).
func (s *Service) Answer(ctx context.Context, id, answer, choice string, proposeValue string) (Question, error) {
	q, err := s.Get(ctx, id)
	if err != nil {
		return q, err
	}
	if q.Status != "open" {
		return q, fmt.Errorf("%s is already %s", id, q.Status)
	}
	if strings.TrimSpace(answer) == "" && choice == "" {
		return q, errors.New("give an answer or pick an option")
	}
	a := access.FromContext(ctx)
	q.Status, q.Answer, q.Choice, q.AnsweredBy, q.AnsweredAt = "answered", answer, choice, a.Name, store.Now()
	if _, err := s.db.ExecContext(ctx, `UPDATE questions SET status=?, answer=?, choice=?, answered_by=?, answered_at=? WHERE id=?`,
		q.Status, q.Answer, q.Choice, q.AnsweredBy, q.AnsweredAt, id); err != nil {
		return q, err
	}
	text := answerText(q)
	s.hist.Must(ctx, history.Event{ProjectID: q.ProjectID, Type: "question.answered", Subject: id, Summary: fmt.Sprintf("%s answered %s", a.Name, id), Why: text,
		Data: history.Data(map[string]any{"task": q.TaskID, "choice": choice})})
	if q.TaskID != "" {
		if t, err := s.queue.Get(ctx, q.TaskID); err == nil {
			t.ContextNote = strings.TrimSpace(t.ContextNote + fmt.Sprintf("\n\nQ %s: %s\nAnswer from %s: %s", q.ID, q.Title, a.Name, text))
			_, _ = s.queue.Update(ctx, t)
			sent := false
			if q.SessionID != "" && s.Hooks.SendToSession != nil {
				sent = s.Hooks.SendToSession(ctx, q.SessionID, "Answer to "+q.ID+": "+text) == nil
			}
			if !sent && t.Status == queue.Blocked {
				_, _ = s.queue.SetStatus(ctx, t.ID, queue.Queued, "answer received for "+q.ID)
			}
		}
	}
	if proposeValue != "" && s.Hooks.ProposeValue != nil {
		_ = s.Hooks.ProposeValue(ctx, q.ProjectID, proposeValue, "answer to "+q.ID)
	}
	return q, nil
}

func answerText(q Question) string {
	parts := []string{}
	if q.Choice != "" {
		for _, o := range q.Options {
			if o.Key == q.Choice {
				parts = append(parts, "chose "+o.Key+" ("+o.Title+")")
			}
		}
		if len(parts) == 0 {
			parts = append(parts, "chose "+q.Choice)
		}
	}
	if q.Answer != "" {
		parts = append(parts, q.Answer)
	}
	return strings.Join(parts, ". ")
}

// Skip closes a question without an answer; a blocked task resumes on its assumption.
func (s *Service) Skip(ctx context.Context, id, why string) (Question, error) {
	q, err := s.Get(ctx, id)
	if err != nil {
		return q, err
	}
	q.Status = "skipped"
	if _, err := s.db.ExecContext(ctx, `UPDATE questions SET status='skipped', answered_by=?, answered_at=? WHERE id=?`, access.FromContext(ctx).Name, store.Now(), id); err != nil {
		return q, err
	}
	s.hist.Must(ctx, history.Event{ProjectID: q.ProjectID, Type: "question.skipped", Subject: id, Summary: "Skipped " + id, Why: why})
	if q.TaskID != "" {
		if t, err := s.queue.Get(ctx, q.TaskID); err == nil && t.Status == queue.Blocked {
			_, _ = s.queue.SetStatus(ctx, t.ID, queue.Queued, "question skipped; continue with your recommendation")
		}
	}
	return q, nil
}

// Confirm accepts a non-blocking question's assumption.
func (s *Service) Confirm(ctx context.Context, id string) (Question, error) {
	q, err := s.Get(ctx, id)
	if err != nil {
		return q, err
	}
	if q.Blocking {
		return q, errors.New("only non-blocking assumptions can be confirmed")
	}
	return s.Answer(ctx, id, "Assumption confirmed: "+q.Assumption, "", "")
}
