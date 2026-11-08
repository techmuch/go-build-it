// Package history records every action with who, what, when and why (NFR-10), and
// fans events out to live subscribers (FR-36).
package history

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/techmuch/go-build-it/internal/access"
	"github.com/techmuch/go-build-it/internal/store"
)

// Event is one history record.
type Event struct {
	ID        int64           `json:"id"`
	TS        string          `json:"ts"`
	Actor     string          `json:"actor"`
	ActorKind string          `json:"actorKind"`
	Origin    string          `json:"origin"`
	ProjectID string          `json:"projectId"`
	Type      string          `json:"type"`
	Subject   string          `json:"subject"`
	Summary   string          `json:"summary"`
	Why       string          `json:"why"`
	Data      json.RawMessage `json:"data"`
}

// Recorder writes events and publishes them.
type Recorder struct {
	db   *sql.DB
	mu   sync.Mutex
	subs map[chan Event]struct{}
}

// New returns a Recorder backed by db.
func New(db *sql.DB) *Recorder { return &Recorder{db: db, subs: map[chan Event]struct{}{}} }

// Record stores an event attributed to the actor in ctx and publishes it.
func (r *Recorder) Record(ctx context.Context, e Event) (Event, error) {
	a := access.FromContext(ctx)
	if e.Actor == "" {
		e.Actor, e.ActorKind, e.Origin = a.Name, a.Kind, string(a.Origin)
	}
	if e.ActorKind == "" {
		e.ActorKind = "system"
	}
	if e.Origin == "" {
		e.Origin = string(access.System)
	}
	e.TS = store.Now()
	if len(e.Data) == 0 {
		e.Data = json.RawMessage("{}")
	}
	res, err := r.db.ExecContext(ctx, `INSERT INTO events(ts, actor, actor_kind, origin, project_id, type, subject, summary, why, data)
		VALUES(?,?,?,?,?,?,?,?,?,?)`, e.TS, e.Actor, e.ActorKind, e.Origin, e.ProjectID, e.Type, e.Subject, e.Summary, e.Why, string(e.Data))
	if err != nil {
		return e, fmt.Errorf("record event: %w", err)
	}
	e.ID, _ = res.LastInsertId()
	r.Publish(e)
	return e, nil
}

// Must records an event and ignores errors (history failures must not break the action; they are logged by callers).
func (r *Recorder) Must(ctx context.Context, e Event) Event { ev, _ := r.Record(ctx, e); return ev }

// Data marshals v for Event.Data.
func Data(v any) json.RawMessage { b, _ := json.Marshal(v); return b }

// Publish sends an event to live subscribers without storing it (e.g. session output ticks).
func (r *Recorder) Publish(e Event) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for ch := range r.subs {
		select {
		case ch <- e:
		default: // slow subscriber: drop
		}
	}
}

// Subscribe returns a channel of live events and a cancel func.
func (r *Recorder) Subscribe() (<-chan Event, func()) {
	ch := make(chan Event, 256)
	r.mu.Lock()
	r.subs[ch] = struct{}{}
	r.mu.Unlock()
	return ch, func() {
		r.mu.Lock()
		delete(r.subs, ch)
		r.mu.Unlock()
	}
}

// Query filters for List.
type Query struct {
	ProjectID string
	Type      string
	Actor     string
	Text      string
	Subject   string
	Since     string
	Limit     int
	BeforeID  int64
}

// List returns events, newest first.
func (r *Recorder) List(ctx context.Context, q Query) ([]Event, error) {
	var where []string
	var args []any
	add := func(cond string, v any) { where = append(where, cond); args = append(args, v) }
	if q.ProjectID != "" {
		add("project_id = ?", q.ProjectID)
	}
	if q.Type != "" {
		add("type LIKE ?", q.Type+"%")
	}
	if q.Actor != "" {
		add("actor = ?", q.Actor)
	}
	if q.Subject != "" {
		add("subject = ?", q.Subject)
	}
	if q.Since != "" {
		add("ts >= ?", q.Since)
	}
	if q.BeforeID > 0 {
		add("id < ?", q.BeforeID)
	}
	if q.Text != "" {
		add("(summary LIKE ? OR why LIKE ? OR subject LIKE ?)", "%"+q.Text+"%")
		args = append(args, "%"+q.Text+"%", "%"+q.Text+"%")
	}
	if q.Limit <= 0 || q.Limit > 1000 {
		q.Limit = 200
	}
	s := `SELECT id, ts, actor, actor_kind, origin, project_id, type, subject, summary, why, data FROM events`
	if len(where) > 0 {
		s += " WHERE " + strings.Join(where, " AND ")
	}
	s += " ORDER BY id DESC LIMIT ?"
	args = append(args, q.Limit)
	rows, err := r.db.QueryContext(ctx, s, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Event{}
	for rows.Next() {
		var e Event
		var data string
		if err := rows.Scan(&e.ID, &e.TS, &e.Actor, &e.ActorKind, &e.Origin, &e.ProjectID, &e.Type, &e.Subject, &e.Summary, &e.Why, &data); err != nil {
			return nil, err
		}
		e.Data = json.RawMessage(data)
		out = append(out, e)
	}
	return out, rows.Err()
}

// Get returns one event by ID.
func (r *Recorder) Get(ctx context.Context, id int64) (Event, error) {
	var e Event
	var data string
	err := r.db.QueryRowContext(ctx, `SELECT id, ts, actor, actor_kind, origin, project_id, type, subject, summary, why, data FROM events WHERE id=?`, id).
		Scan(&e.ID, &e.TS, &e.Actor, &e.ActorKind, &e.Origin, &e.ProjectID, &e.Type, &e.Subject, &e.Summary, &e.Why, &data)
	e.Data = json.RawMessage(data)
	return e, err
}
