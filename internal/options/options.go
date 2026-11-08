// Package options manages option sets: 2–3 alternatives the human chooses among or
// combines, recorded with reasons so values can be refined (FR-120–FR-124).
package options

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/techmuch/go-build-it/internal/access"
	"github.com/techmuch/go-build-it/internal/history"
	"github.com/techmuch/go-build-it/internal/store"
)

// Option is one alternative (FR-121).
type Option struct {
	Key     string   `json:"key"`
	Title   string   `json:"title"`
	Summary string   `json:"summary"`
	Pros    []string `json:"pros"`
	Cons    []string `json:"cons"`
	Mockup  string   `json:"mockup"` // path to an HTML mockup or image in the project repo
	Aspects []string `json:"aspects"`
}

// Pick is an aspect chosen from an option (FR-122).
type Pick struct {
	Option string `json:"option"`
	Aspect string `json:"aspect"`
}

// Set is an option set.
type Set struct {
	ID          string   `json:"id"`
	ProjectID   string   `json:"projectId"`
	Title       string   `json:"title"`
	Status      string   `json:"status"`
	GeneratedBy string   `json:"generatedBy"`
	ValueLens   []string `json:"valueLens"` // values the options were generated against (FR-124)
	Options     []Option `json:"options"`
	Choice      []Pick   `json:"choice"`
	Reason      string   `json:"reason"`
	CreatedAt   string   `json:"createdAt"`
	DecidedAt   string   `json:"decidedAt"`
}

// Hooks connect choices to values and follow-up work.
type Hooks struct {
	Values       func(ctx context.Context, projectID string) []string
	ProposeValue func(ctx context.Context, projectID, text, source string) error
	FollowUp     func(ctx context.Context, s Set) error
}

// Service manages option sets.
type Service struct {
	db    *sql.DB
	hist  *history.Recorder
	Hooks Hooks
}

// New returns an option set service.
func New(db *sql.DB, h *history.Recorder) *Service { return &Service{db: db, hist: h} }

const cols = `id,project_id,title,status,generated_by,value_lens,options,choice,reason,created_at,decided_at`

func scan(sc interface{ Scan(...any) error }) (Set, error) {
	var s Set
	var vl, op, ch string
	err := sc.Scan(&s.ID, &s.ProjectID, &s.Title, &s.Status, &s.GeneratedBy, &vl, &op, &ch, &s.Reason, &s.CreatedAt, &s.DecidedAt)
	_ = json.Unmarshal([]byte(vl), &s.ValueLens)
	_ = json.Unmarshal([]byte(op), &s.Options)
	_ = json.Unmarshal([]byte(ch), &s.Choice)
	if s.ValueLens == nil {
		s.ValueLens = []string{}
	}
	if s.Choice == nil {
		s.Choice = []Pick{}
	}
	return s, err
}

func (svc *Service) write(ctx context.Context, s Set) error {
	vl, _ := json.Marshal(s.ValueLens)
	op, _ := json.Marshal(s.Options)
	ch, _ := json.Marshal(s.Choice)
	_, err := svc.db.ExecContext(ctx, `INSERT INTO option_sets(`+cols+`) VALUES(?,?,?,?,?,?,?,?,?,?,?)
		ON CONFLICT(id) DO UPDATE SET status=excluded.status, options=excluded.options, choice=excluded.choice, reason=excluded.reason, decided_at=excluded.decided_at`,
		s.ID, s.ProjectID, s.Title, s.Status, s.GeneratedBy, string(vl), string(op), string(ch), s.Reason, s.CreatedAt, s.DecidedAt)
	return err
}

// Create records a new option set; it must have 2 or 3 options (FR-120).
func (svc *Service) Create(ctx context.Context, s Set) (Set, error) {
	if s.ProjectID == "" || s.Title == "" {
		return s, errors.New("project and title are required")
	}
	if len(s.Options) < 2 || len(s.Options) > 3 {
		return s, errors.New("an option set has 2 or 3 options")
	}
	for i := range s.Options {
		if s.Options[i].Key == "" {
			s.Options[i].Key = string(rune('A' + i))
		}
		if s.Options[i].Summary == "" {
			return s, fmt.Errorf("option %s needs a summary", s.Options[i].Key)
		}
	}
	id, err := store.NextID(svc.db, "OS")
	if err != nil {
		return s, err
	}
	s.ID, s.Status, s.CreatedAt = id, "open", store.Now()
	if s.GeneratedBy == "" {
		s.GeneratedBy = access.FromContext(ctx).Name
	}
	if len(s.ValueLens) == 0 && svc.Hooks.Values != nil {
		s.ValueLens = svc.Hooks.Values(ctx, s.ProjectID) // FR-124
	}
	s.Choice = []Pick{}
	if err := svc.write(ctx, s); err != nil {
		return s, err
	}
	svc.hist.Must(ctx, history.Event{ProjectID: s.ProjectID, Type: "options.created", Subject: id, Summary: fmt.Sprintf("Option set %s: %s (%d options)", id, s.Title, len(s.Options)),
		Why: "values: " + strings.Join(s.ValueLens, ", ")})
	return s, nil
}

// Get returns one option set.
func (svc *Service) Get(ctx context.Context, id string) (Set, error) {
	s, err := scan(svc.db.QueryRowContext(ctx, `SELECT `+cols+` FROM option_sets WHERE id=?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return s, fmt.Errorf("option set %s not found", id)
	}
	return s, err
}

// List returns option sets, open first.
func (svc *Service) List(ctx context.Context, projectID string) ([]Set, error) {
	q, args := `SELECT `+cols+` FROM option_sets`, []any{}
	if projectID != "" {
		q, args = q+` WHERE project_id=?`, append(args, projectID)
	}
	rows, err := svc.db.QueryContext(ctx, q+` ORDER BY status='open' DESC, created_at DESC`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Set{}
	for rows.Next() {
		s, err := scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// Choose records the human's choice: whole options or individual aspects, with a reason (FR-122).
func (svc *Service) Choose(ctx context.Context, id string, picks []Pick, reason, value string) (Set, error) {
	s, err := svc.Get(ctx, id)
	if err != nil {
		return s, err
	}
	if s.Status != "open" {
		return s, fmt.Errorf("%s is already decided", id)
	}
	if len(picks) == 0 {
		return s, errors.New("pick at least one option or aspect")
	}
	if strings.TrimSpace(reason) == "" {
		return s, errors.New("say why: the reason refines the project's values")
	}
	keys := map[string]Option{}
	for _, o := range s.Options {
		keys[o.Key] = o
	}
	for _, p := range picks {
		if _, ok := keys[p.Option]; !ok {
			return s, fmt.Errorf("unknown option %q", p.Option)
		}
	}
	s.Status, s.Choice, s.Reason, s.DecidedAt = "decided", picks, reason, store.Now()
	if err := svc.write(ctx, s); err != nil {
		return s, err
	}
	var desc []string
	for _, p := range picks {
		if p.Aspect == "" {
			desc = append(desc, p.Option)
		} else {
			desc = append(desc, p.Option+": "+p.Aspect)
		}
	}
	a := access.FromContext(ctx)
	svc.hist.Must(ctx, history.Event{ProjectID: s.ProjectID, Type: "options.chosen", Subject: id, Summary: fmt.Sprintf("%s chose %s in %s", a.Name, strings.Join(desc, " + "), id), Why: reason,
		Data: history.Data(map[string]any{"options": s.Options, "choice": picks, "reason": reason, "valueLens": s.ValueLens})})
	if value != "" && svc.Hooks.ProposeValue != nil {
		_ = svc.Hooks.ProposeValue(ctx, s.ProjectID, value, "choice in "+id)
	}
	if svc.Hooks.FollowUp != nil {
		_ = svc.Hooks.FollowUp(ctx, s)
	}
	return s, nil
}
