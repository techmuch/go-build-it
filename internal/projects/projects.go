// Package projects manages projects, their lifecycle, roadmap, values and problem
// definition (FR-40–FR-46, FR-115, FR-123).
package projects

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/techmuch/go-build-it/internal/gitx"
	"github.com/techmuch/go-build-it/internal/history"
	"github.com/techmuch/go-build-it/internal/store"
)

// Stages is the project lifecycle (FR-43).
var Stages = []string{"Discovery", "Requirements", "Design", "Implementation", "Review", "Released", "Maintenance", "Paused", "Archived"}

// Project is a managed software project.
type Project struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Path       string `json:"path"`
	Purpose    string `json:"purpose"`
	Stage      string `json:"stage"`
	Forge      string `json:"forge"`
	Remote     string `json:"remote"`
	WebsiteURL string `json:"websiteUrl"`
	TeamID     string `json:"teamId"`
	CreatedAt  string `json:"createdAt"`
	UpdatedAt  string `json:"updatedAt"`
}

// Slug returns owner/name for the remote.
func (p Project) Slug() string { return gitx.RepoSlug(p.Remote) }

// Live is git/forge data shown on catalog cards (FR-41).
type Live struct {
	Release   *gitx.Release `json:"release"`
	Unpushed  gitx.Unpushed `json:"unpushed"`
	Repo      gitx.RepoInfo `json:"repo"`
	Branch    string        `json:"branch"`
	FetchedAt string        `json:"fetchedAt"`
	Errors    []string      `json:"errors,omitempty"`
}

// Value is a project guiding principle with its provenance (FR-123).
type Value struct {
	ID        int64  `json:"id"`
	ProjectID string `json:"projectId"`
	Text      string `json:"text"`
	Source    string `json:"source"`
	Status    string `json:"status"`
	CreatedAt string `json:"createdAt"`
}

// Problem is one version of the living problem definition (FR-123).
type Problem struct {
	Version   int    `json:"version"`
	Text      string `json:"text"`
	Source    string `json:"source"`
	CreatedAt string `json:"createdAt"`
}

// Milestone is a roadmap item with an optional review gate (FR-45b, FR-115).
type Milestone struct {
	ID           string   `json:"id"`
	ProjectID    string   `json:"projectId"`
	Ord          int      `json:"ord"`
	Name         string   `json:"name"`
	Scope        string   `json:"scope"`
	Status       string   `json:"status"`
	GateName     string   `json:"gateName"`
	GateKind     string   `json:"gateKind"`
	GateStatus   string   `json:"gateStatus"`
	GateCriteria []string `json:"gateCriteria"`
	Target       string   `json:"target"`
}

// Service manages projects.
type Service struct {
	db    *sql.DB
	hist  *history.Recorder
	git   *gitx.Git
	forge *gitx.ForgeCLI

	mu    sync.Mutex
	cache map[string]Live
}

// New returns a project service.
func New(db *sql.DB, h *history.Recorder, g *gitx.Git, f *gitx.ForgeCLI) *Service {
	return &Service{db: db, hist: h, git: g, forge: f, cache: map[string]Live{}}
}

var idRe = regexp.MustCompile(`[^a-z0-9-]+`)

// Slugify makes a project id.
func Slugify(s string) string {
	return strings.Trim(idRe.ReplaceAllString(strings.ToLower(s), "-"), "-")
}

// Import registers an existing git repository as a project.
func (s *Service) Import(ctx context.Context, path, id, purpose, stage string) (Project, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return Project{}, err
	}
	if st, err := os.Stat(abs); err != nil || !st.IsDir() {
		return Project{}, fmt.Errorf("%s is not a directory", abs)
	}
	if !s.git.IsRepo(ctx, abs) {
		return Project{}, fmt.Errorf("%s is not a git repository", abs)
	}
	if id == "" {
		id = Slugify(gitx.RepoName(abs))
	}
	if stage == "" {
		stage = "Discovery"
	}
	if !validStage(stage) {
		return Project{}, fmt.Errorf("unknown stage %q", stage)
	}
	remote := s.git.Remote(ctx, abs)
	now := store.Now()
	p := Project{ID: id, Name: id, Path: abs, Purpose: purpose, Stage: stage, Forge: gitx.Forge(remote), Remote: remote, CreatedAt: now, UpdatedAt: now}
	_, err = s.db.ExecContext(ctx, `INSERT INTO projects(id,name,path,purpose,stage,forge,remote,website_url,team_id,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?,?,?,?)`,
		p.ID, p.Name, p.Path, p.Purpose, p.Stage, p.Forge, p.Remote, "", "", now, now)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			return Project{}, fmt.Errorf("project %q already exists", id)
		}
		return Project{}, err
	}
	s.hist.Must(ctx, history.Event{ProjectID: id, Type: "project.imported", Subject: id, Summary: "Imported project " + id, Why: "registered " + abs})
	return p, nil
}

func validStage(st string) bool {
	for _, x := range Stages {
		if x == st {
			return true
		}
	}
	return false
}

const cols = `id,name,path,purpose,stage,forge,remote,website_url,team_id,created_at,updated_at`

func scan(sc interface{ Scan(...any) error }) (Project, error) {
	var p Project
	err := sc.Scan(&p.ID, &p.Name, &p.Path, &p.Purpose, &p.Stage, &p.Forge, &p.Remote, &p.WebsiteURL, &p.TeamID, &p.CreatedAt, &p.UpdatedAt)
	return p, err
}

// ErrNotFound is returned for unknown projects.
var ErrNotFound = errors.New("project not found")

// Get returns one project.
func (s *Service) Get(ctx context.Context, id string) (Project, error) {
	p, err := scan(s.db.QueryRowContext(ctx, `SELECT `+cols+` FROM projects WHERE id=?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return p, ErrNotFound
	}
	return p, err
}

// List returns all projects.
func (s *Service) List(ctx context.Context) ([]Project, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT `+cols+` FROM projects ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Project{}
	for rows.Next() {
		p, err := scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// Update changes editable fields. Empty strings leave fields unchanged.
func (s *Service) Update(ctx context.Context, id string, name, purpose, stage, website, team string) (Project, error) {
	p, err := s.Get(ctx, id)
	if err != nil {
		return p, err
	}
	var changes []string
	set := func(dst *string, v, label string) {
		if v != "" && v != *dst {
			changes = append(changes, fmt.Sprintf("%s: %q → %q", label, *dst, v))
			*dst = v
		}
	}
	if stage != "" && !validStage(stage) {
		return p, fmt.Errorf("unknown stage %q", stage)
	}
	set(&p.Name, name, "name")
	set(&p.Purpose, purpose, "purpose")
	set(&p.Stage, stage, "stage")
	set(&p.WebsiteURL, website, "website")
	set(&p.TeamID, team, "team")
	if len(changes) == 0 {
		return p, nil
	}
	p.UpdatedAt = store.Now()
	_, err = s.db.ExecContext(ctx, `UPDATE projects SET name=?,purpose=?,stage=?,website_url=?,team_id=?,updated_at=? WHERE id=?`,
		p.Name, p.Purpose, p.Stage, p.WebsiteURL, p.TeamID, p.UpdatedAt, id)
	if err == nil {
		s.hist.Must(ctx, history.Event{ProjectID: id, Type: "project.updated", Subject: id, Summary: "Updated " + id, Why: strings.Join(changes, "; ")})
	}
	return p, err
}

// Remove unregisters a project (files are untouched).
func (s *Service) Remove(ctx context.Context, id string) error {
	if _, err := s.Get(ctx, id); err != nil {
		return err
	}
	_, err := s.db.ExecContext(ctx, `DELETE FROM projects WHERE id=?`, id)
	if err == nil {
		s.hist.Must(ctx, history.Event{ProjectID: id, Type: "project.removed", Subject: id, Summary: "Removed project " + id})
	}
	return err
}

// Live returns cached git/forge data, refreshing when older than maxAge.
func (s *Service) Live(ctx context.Context, p Project, maxAge time.Duration) Live {
	s.mu.Lock()
	l, ok := s.cache[p.ID]
	s.mu.Unlock()
	if ok {
		if t, err := time.Parse(time.RFC3339, l.FetchedAt); err == nil && time.Since(t) < maxAge {
			return l
		}
	}
	return s.Refresh(ctx, p)
}

// Refresh reloads git and forge data for a project.
func (s *Service) Refresh(ctx context.Context, p Project) Live {
	l := Live{FetchedAt: store.Now()}
	l.Release = s.git.LatestRelease(ctx, p.Path)
	l.Unpushed = s.git.FindUnpushed(ctx, p.Path)
	l.Branch = s.git.CurrentBranch(ctx, p.Path)
	if p.Forge == "github" && s.forge != nil {
		ri, err := s.forge.Info(ctx, p.Forge, p.Slug(), p.Path)
		if err != nil {
			l.Errors = append(l.Errors, err.Error())
		}
		l.Repo = ri
		if ri.WebsiteURL != "" && p.WebsiteURL == "" {
			_, _ = s.db.ExecContext(ctx, `UPDATE projects SET website_url=? WHERE id=?`, ri.WebsiteURL, p.ID)
		}
		if l.Release != nil {
			l.Repo.ReleaseURL = ri.RepoURL + "/releases/tag/" + l.Release.Tag
		}
	}
	s.mu.Lock()
	s.cache[p.ID] = l
	s.mu.Unlock()
	return l
}

// Values ---------------------------------------------------------------------

// Values lists a project's values (all statuses except dismissed unless all=true).
func (s *Service) Values(ctx context.Context, projectID string, all bool) ([]Value, error) {
	q := `SELECT id,project_id,text,source,status,created_at FROM project_values WHERE project_id=?`
	if !all {
		q += ` AND status != 'dismissed'`
	}
	rows, err := s.db.QueryContext(ctx, q+` ORDER BY id`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Value{}
	for rows.Next() {
		var v Value
		if err := rows.Scan(&v.ID, &v.ProjectID, &v.Text, &v.Source, &v.Status, &v.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

// AddValue records a value; status is "accepted" or "proposed".
func (s *Service) AddValue(ctx context.Context, projectID, text, source, status string) (Value, error) {
	if status == "" {
		status = "accepted"
	}
	if status != "accepted" && status != "proposed" {
		return Value{}, fmt.Errorf("status must be accepted or proposed")
	}
	v := Value{ProjectID: projectID, Text: text, Source: source, Status: status, CreatedAt: store.Now()}
	res, err := s.db.ExecContext(ctx, `INSERT INTO project_values(project_id,text,source,status,created_at) VALUES(?,?,?,?,?)`, projectID, text, source, status, v.CreatedAt)
	if err != nil {
		return v, err
	}
	v.ID, _ = res.LastInsertId()
	verb := "Added value"
	if status == "proposed" {
		verb = "Proposed value"
	}
	s.hist.Must(ctx, history.Event{ProjectID: projectID, Type: "value." + status, Subject: fmt.Sprint(v.ID), Summary: fmt.Sprintf("%s %q", verb, text), Why: source})
	return v, nil
}

// SetValueStatus accepts or dismisses a value.
func (s *Service) SetValueStatus(ctx context.Context, id int64, status, text string) (Value, error) {
	if status != "accepted" && status != "dismissed" && status != "proposed" {
		return Value{}, fmt.Errorf("invalid status %q", status)
	}
	var v Value
	err := s.db.QueryRowContext(ctx, `SELECT id,project_id,text,source,status,created_at FROM project_values WHERE id=?`, id).Scan(&v.ID, &v.ProjectID, &v.Text, &v.Source, &v.Status, &v.CreatedAt)
	if err != nil {
		return v, fmt.Errorf("value %d: %w", id, err)
	}
	if text != "" {
		v.Text = text
	}
	v.Status = status
	if _, err := s.db.ExecContext(ctx, `UPDATE project_values SET status=?, text=? WHERE id=?`, status, v.Text, id); err != nil {
		return v, err
	}
	s.hist.Must(ctx, history.Event{ProjectID: v.ProjectID, Type: "value." + status, Subject: fmt.Sprint(id), Summary: fmt.Sprintf("Value %q %s", v.Text, status)})
	return v, nil
}

// Problem returns the latest problem definition and its version number.
func (s *Service) Problem(ctx context.Context, projectID string) (Problem, error) {
	var p Problem
	err := s.db.QueryRowContext(ctx, `SELECT (SELECT COUNT(*) FROM problem_defs WHERE project_id=?), text, source, created_at FROM problem_defs WHERE project_id=? ORDER BY id DESC LIMIT 1`, projectID, projectID).
		Scan(&p.Version, &p.Text, &p.Source, &p.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return Problem{}, nil
	}
	return p, err
}

// RefineProblem stores a new version of the problem definition.
func (s *Service) RefineProblem(ctx context.Context, projectID, text, source string) (Problem, error) {
	if _, err := s.db.ExecContext(ctx, `INSERT INTO problem_defs(project_id,text,source,created_at) VALUES(?,?,?,?)`, projectID, text, source, store.Now()); err != nil {
		return Problem{}, err
	}
	p, err := s.Problem(ctx, projectID)
	s.hist.Must(ctx, history.Event{ProjectID: projectID, Type: "problem.refined", Subject: fmt.Sprintf("v%d", p.Version), Summary: fmt.Sprintf("Problem definition v%d", p.Version), Why: source})
	return p, err
}

// Milestones -----------------------------------------------------------------

const mcols = `id,project_id,ord,name,scope,status,gate_name,gate_kind,gate_status,gate_criteria,target`

func scanM(sc interface{ Scan(...any) error }) (Milestone, error) {
	var m Milestone
	var crit string
	err := sc.Scan(&m.ID, &m.ProjectID, &m.Ord, &m.Name, &m.Scope, &m.Status, &m.GateName, &m.GateKind, &m.GateStatus, &crit, &m.Target)
	_ = json.Unmarshal([]byte(crit), &m.GateCriteria)
	if m.GateCriteria == nil {
		m.GateCriteria = []string{}
	}
	return m, err
}

// Milestones returns a project's roadmap in order.
func (s *Service) Milestones(ctx context.Context, projectID string) ([]Milestone, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT `+mcols+` FROM milestones WHERE project_id=? ORDER BY ord, id`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Milestone{}
	for rows.Next() {
		m, err := scanM(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// Milestone returns one roadmap item.
func (s *Service) Milestone(ctx context.Context, id string) (Milestone, error) {
	m, err := scanM(s.db.QueryRowContext(ctx, `SELECT `+mcols+` FROM milestones WHERE id=?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return m, fmt.Errorf("milestone %s not found", id)
	}
	return m, err
}

// SaveMilestone creates or updates a milestone. A new one gets id <project>-M<n>.
func (s *Service) SaveMilestone(ctx context.Context, m Milestone) (Milestone, error) {
	if m.Status == "" {
		m.Status = "planned"
	}
	if m.GateStatus == "" {
		m.GateStatus = "pending"
	}
	if m.GateCriteria == nil {
		m.GateCriteria = []string{}
	}
	crit, _ := json.Marshal(m.GateCriteria)
	isNew := m.ID == ""
	if isNew {
		var n int
		_ = s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM milestones WHERE project_id=?`, m.ProjectID).Scan(&n)
		m.ID = fmt.Sprintf("%s-M%d", m.ProjectID, n)
		if m.Ord == 0 {
			m.Ord = n
		}
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO milestones(`+mcols+`) VALUES(?,?,?,?,?,?,?,?,?,?,?)
		ON CONFLICT(id) DO UPDATE SET ord=excluded.ord, name=excluded.name, scope=excluded.scope, status=excluded.status,
		gate_name=excluded.gate_name, gate_kind=excluded.gate_kind, gate_status=excluded.gate_status, gate_criteria=excluded.gate_criteria, target=excluded.target`,
		m.ID, m.ProjectID, m.Ord, m.Name, m.Scope, m.Status, m.GateName, m.GateKind, m.GateStatus, string(crit), m.Target)
	if err != nil {
		return m, err
	}
	verb := "Updated"
	if isNew {
		verb = "Added"
	}
	s.hist.Must(ctx, history.Event{ProjectID: m.ProjectID, Type: "milestone.saved", Subject: m.ID, Summary: fmt.Sprintf("%s milestone %s", verb, m.Name)})
	return m, nil
}

// SetGate records a gate decision (FR-118). Only "approved" opens the gate.
func (s *Service) SetGate(ctx context.Context, id, status, note string) (Milestone, error) {
	m, err := s.Milestone(ctx, id)
	if err != nil {
		return m, err
	}
	if status != "approved" && status != "changes" && status != "pending" {
		return m, fmt.Errorf("gate status must be approved, changes or pending")
	}
	m.GateStatus = status
	if status == "approved" {
		m.Status = "done"
	}
	if _, err := s.SaveMilestone(ctx, m); err != nil {
		return m, err
	}
	s.hist.Must(ctx, history.Event{ProjectID: m.ProjectID, Type: "gate." + status, Subject: m.ID, Summary: fmt.Sprintf("Gate %q %s", m.GateName, status), Why: note})
	return m, nil
}
