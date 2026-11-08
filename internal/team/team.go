// Package team manages teams, roles, AI personas and human members (FR-10–FR-13, FR-25, FR-26).
package team

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/techmuch/go-build-it/internal/history"
)

// Built-in roles (FR-25).
var BuiltinRoles = []Role{
	{ID: "pm", Name: "Project Manager", Description: "Plans the roadmap, sizes and splits tasks, keeps the queue healthy.", Builtin: true},
	{ID: "architect", Name: "Architect", Description: "Owns design decisions, option sets and ADRs.", Builtin: true},
	{ID: "developer", Name: "Developer", Description: "Implements one task at a time, tests first.", Builtin: true},
	{ID: "qa", Name: "QA", Description: "Writes e2e tests and hosts dress rehearsals.", Builtin: true},
	{ID: "reviewer", Name: "Reviewer", Description: "Assesses completed work against standards, requirements and feedback.", Builtin: true},
	{ID: "git", Name: "Git / Release Manager", Description: "Runs every git and forge operation; releases and publishes the website.", Builtin: true},
}

// Role is a responsibility in a team.
type Role struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Builtin     bool   `json:"builtin"`
}

// Team groups members and is assigned to projects.
type Team struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Member is an AI persona or a human.
type Member struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Kind         string   `json:"kind"` // ai | human
	Role         string   `json:"role"`
	TeamID       string   `json:"teamId"`
	Harness      string   `json:"harness"`
	Model        string   `json:"model"`
	Instructions string   `json:"instructions"`
	Login        string   `json:"login"`
	Standards    []string `json:"standards"`
	Rev          int      `json:"rev"`
}

// Service manages the team.
type Service struct {
	db   *sql.DB
	hist *history.Recorder
}

// New returns a team service and makes sure built-in roles exist.
func New(db *sql.DB, h *history.Recorder) *Service {
	for _, r := range BuiltinRoles {
		_, _ = db.Exec(`INSERT OR IGNORE INTO roles(id,name,description,builtin) VALUES(?,?,?,1)`, r.ID, r.Name, r.Description)
	}
	return &Service{db: db, hist: h}
}

// Roles lists roles.
func (s *Service) Roles(ctx context.Context) ([]Role, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id,name,description,builtin FROM roles ORDER BY builtin DESC, name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Role{}
	for rows.Next() {
		var r Role
		if err := rows.Scan(&r.ID, &r.Name, &r.Description, &r.Builtin); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// AddRole defines a custom role (FR-26).
func (s *Service) AddRole(ctx context.Context, id, name, desc string) (Role, error) {
	r := Role{ID: id, Name: name, Description: desc}
	if id == "" || name == "" {
		return r, errors.New("role id and name are required")
	}
	if _, err := s.db.ExecContext(ctx, `INSERT INTO roles(id,name,description,builtin) VALUES(?,?,?,0)`, id, name, desc); err != nil {
		return r, err
	}
	s.hist.Must(ctx, history.Event{Type: "role.added", Subject: id, Summary: "Added custom role " + name})
	return r, nil
}

// Teams lists teams.
func (s *Service) Teams(ctx context.Context) ([]Team, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id,name FROM teams ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Team{}
	for rows.Next() {
		var t Team
		if err := rows.Scan(&t.ID, &t.Name); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// SaveTeam creates or renames a team.
func (s *Service) SaveTeam(ctx context.Context, id, name string) (Team, error) {
	if id == "" || name == "" {
		return Team{}, errors.New("team id and name are required")
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO teams(id,name) VALUES(?,?) ON CONFLICT(id) DO UPDATE SET name=excluded.name`, id, name)
	if err == nil {
		s.hist.Must(ctx, history.Event{Type: "team.saved", Subject: id, Summary: "Saved team " + name})
	}
	return Team{ID: id, Name: name}, err
}

// RemoveTeam deletes a team; its members become unassigned.
func (s *Service) RemoveTeam(ctx context.Context, id string) error {
	if _, err := s.db.ExecContext(ctx, `DELETE FROM teams WHERE id=?`, id); err != nil {
		return err
	}
	_, _ = s.db.ExecContext(ctx, `UPDATE members SET team_id='' WHERE team_id=?`, id)
	s.hist.Must(ctx, history.Event{Type: "team.removed", Subject: id, Summary: "Removed team " + id})
	return nil
}

const cols = `id,name,kind,role,team_id,harness,model,instructions,login,standards,rev`

func scan(sc interface{ Scan(...any) error }) (Member, error) {
	var m Member
	var st string
	err := sc.Scan(&m.ID, &m.Name, &m.Kind, &m.Role, &m.TeamID, &m.Harness, &m.Model, &m.Instructions, &m.Login, &st, &m.Rev)
	_ = json.Unmarshal([]byte(st), &m.Standards)
	if m.Standards == nil {
		m.Standards = []string{}
	}
	return m, err
}

// Members lists members, optionally for one team.
func (s *Service) Members(ctx context.Context, teamID string) ([]Member, error) {
	q, args := `SELECT `+cols+` FROM members`, []any{}
	if teamID != "" {
		q += ` WHERE team_id=?`
		args = append(args, teamID)
	}
	rows, err := s.db.QueryContext(ctx, q+` ORDER BY kind, name`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Member{}
	for rows.Next() {
		m, err := scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// Member returns one member.
func (s *Service) Member(ctx context.Context, id string) (Member, error) {
	m, err := scan(s.db.QueryRowContext(ctx, `SELECT `+cols+` FROM members WHERE id=?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return m, fmt.Errorf("member %q not found", id)
	}
	return m, err
}

// ByRole returns AI personas holding a role, optionally restricted to a team.
func (s *Service) ByRole(ctx context.Context, role, teamID string) ([]Member, error) {
	all, err := s.Members(ctx, teamID)
	if err != nil {
		return nil, err
	}
	var out []Member
	for _, m := range all {
		if m.Role == role && m.Kind == "ai" {
			out = append(out, m)
		}
	}
	return out, nil
}

// Save creates or updates a member. Each update of an existing persona bumps its revision.
func (s *Service) Save(ctx context.Context, m Member) (Member, error) {
	m.ID = strings.ToLower(strings.TrimSpace(m.ID))
	if m.ID == "" {
		m.ID = strings.ToLower(strings.ReplaceAll(m.Name, " ", "-"))
	}
	if m.Name == "" || m.Role == "" {
		return m, errors.New("name and role are required")
	}
	if m.Kind != "ai" && m.Kind != "human" {
		return m, errors.New("kind must be ai or human")
	}
	if m.Kind == "ai" && m.Harness == "" {
		return m, errors.New("an AI persona needs a harness")
	}
	if m.Standards == nil {
		m.Standards = []string{}
	}
	existing, err := s.Member(ctx, m.ID)
	isNew := err != nil
	m.Rev = 1
	if !isNew {
		m.Rev = existing.Rev + 1
	}
	st, _ := json.Marshal(m.Standards)
	_, err = s.db.ExecContext(ctx, `INSERT INTO members(`+cols+`) VALUES(?,?,?,?,?,?,?,?,?,?,?)
		ON CONFLICT(id) DO UPDATE SET name=excluded.name, kind=excluded.kind, role=excluded.role, team_id=excluded.team_id,
		harness=excluded.harness, model=excluded.model, instructions=excluded.instructions, login=excluded.login, standards=excluded.standards, rev=excluded.rev`,
		m.ID, m.Name, m.Kind, m.Role, m.TeamID, m.Harness, m.Model, m.Instructions, m.Login, string(st), m.Rev)
	if err != nil {
		return m, err
	}
	verb := "Updated"
	if isNew {
		verb = "Added"
	}
	s.hist.Must(ctx, history.Event{Type: "member.saved", Subject: m.ID, Summary: fmt.Sprintf("%s %s %s (%s) rev %d", verb, m.Kind, m.Name, m.Role, m.Rev),
		Data: history.Data(map[string]any{"instructions": m.Instructions, "rev": m.Rev})})
	return m, nil
}

// Remove deletes a member.
func (s *Service) Remove(ctx context.Context, id string) error {
	if _, err := s.Member(ctx, id); err != nil {
		return err
	}
	_, err := s.db.ExecContext(ctx, `DELETE FROM members WHERE id=?`, id)
	if err == nil {
		s.hist.Must(ctx, history.Event{Type: "member.removed", Subject: id, Summary: "Removed member " + id})
	}
	return err
}

// Persona renders a persona definition as Markdown (edited through the artifact editor, FR-13).
func Persona(m Member) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# %s · %s\n\n", m.Name, m.Role)
	b.WriteString(strings.TrimSpace(m.Instructions))
	b.WriteString("\n")
	if len(m.Standards) > 0 {
		b.WriteString("\n## Role standards (added to global)\n")
		for _, s := range m.Standards {
			fmt.Fprintf(&b, "- %s\n", s)
		}
	}
	return b.String()
}
