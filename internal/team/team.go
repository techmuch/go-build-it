// Package team manages roles, global persona archetypes and per-project team members
// (FR-10–FR-13, FR-12a–FR-12e, FR-25, FR-26; ADR-010).
//
// An archetype is a reusable definition (role, base instructions, default harness and
// model, role standards). Every change makes a new revision. A project's AI team member
// is made from an archetype revision plus project tuning; its effective definition is
// what sessions use. Humans are global people who hold a role in a project.
package team

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/techmuch/go-build-it/internal/access"
	"github.com/techmuch/go-build-it/internal/history"
	"github.com/techmuch/go-build-it/internal/store"
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

// StarterArchetypes ship with go-build-it, one per built-in role (FR-12).
var StarterArchetypes = []Archetype{
	{ID: "pm", Name: "Pilot", Role: "pm", Harness: "claude", Instructions: "## How you work\n- Keep the roadmap and the queue honest: small tasks, clear done-when, explicit dependencies.\n- Split any task that needed more than one handoff or compaction.\n- Ask the human only what blocks progress; give your recommendation with every question.\n"},
	{ID: "architect", Name: "Atlas", Role: "architect", Harness: "claude", Instructions: "## How you work\n- Present significant decisions as an option set of 2–3 approaches with tradeoffs.\n- Record decisions as ADRs; keep designs within the global standards.\n- Prefer simple, boring technology; explain any new dependency.\n"},
	{ID: "developer", Name: "Forge", Role: "developer", Harness: "claude", Instructions: "## How you work\n- One task at a time; tests first.\n- Ask a blocking question if the task is ambiguous.\n- No remote git commands; Courier owns git.\n\n## Hand back when\n- tests pass and done-when is met, or\n- you are stuck after 3 attempts on one problem.\n"},
	{ID: "qa", Name: "Probe", Role: "qa", Harness: "claude", Instructions: "## How you work\n- Write end-to-end tests for every feature in scope, with screenshots.\n- Host dress rehearsals step by step and record findings per step.\n"},
	{ID: "reviewer", Name: "Lens", Role: "reviewer", Harness: "claude", Instructions: "## How you work\n- Assess finished work against requirements, standards, values and earlier feedback.\n- Cite evidence for every verdict. Start every review fresh; do not defend earlier work.\n"},
	{ID: "git", Name: "Courier", Role: "git", Harness: "courier", Instructions: "Built in: go-build-it runs git and forge operations directly (FR-93) and calls an agent only for judgment (merge conflicts, release notes).\n"},
}

// StarterTeam lists the archetypes a newly imported project gets as members.
var StarterTeam = []string{"pm", "architect", "developer", "reviewer"}

// Role is a responsibility in a team.
type Role struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Builtin     bool   `json:"builtin"`
}

// Archetype is a global persona template (FR-12).
type Archetype struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Role         string   `json:"role"`
	Instructions string   `json:"instructions"`
	Harness      string   `json:"harness"`
	Model        string   `json:"model"`
	Standards    []string `json:"standards"`
	Rev          int      `json:"rev"`
	Builtin      bool     `json:"builtin"`
	UpdatedAt    string   `json:"updatedAt"`
	UsedBy       int      `json:"usedBy"`
}

// Member is a project team member. For AI members, Harness, Model, Instructions and
// Standards hold the effective values (archetype revision + tuning); the Tuning,
// HarnessOverride, ModelOverride and ReplaceInstructions fields hold the project's own changes.
type Member struct {
	ID        string `json:"id"` // <slug>@<project> for project members
	ProjectID string `json:"projectId"`
	Name      string `json:"name"`
	Kind      string `json:"kind"` // ai | human
	Role      string `json:"role"`
	Login     string `json:"login"`

	ArchetypeID         string   `json:"archetypeId"`
	ArchetypeRev        int      `json:"archetypeRev"`
	ArchetypeLatest     int      `json:"archetypeLatest"`
	Tuning              string   `json:"tuning"`
	ReplaceInstructions bool     `json:"replaceInstructions"`
	HarnessOverride     string   `json:"harnessOverride"`
	ModelOverride       string   `json:"modelOverride"`
	ExtraStandards      []string `json:"extraStandards"`

	Harness      string   `json:"harness"`
	Model        string   `json:"model"`
	Instructions string   `json:"instructions"`
	Standards    []string `json:"standards"`

	Worktree            string `json:"worktree"`
	PaneID              string `json:"paneId"`
	ConversationID      string `json:"conversationId"`
	ConversationTasks   int    `json:"conversationTasks"`
	ConversationStarted string `json:"conversationStarted"`
	Rev                 int    `json:"rev"`
}

// Tuned reports whether a member differs from its archetype.
func (m Member) Tuned() bool {
	return strings.TrimSpace(m.Tuning) != "" || m.HarnessOverride != "" || m.ModelOverride != "" || len(m.ExtraStandards) > 0
}

// Service reads and writes the team model.
type Service struct {
	db   *sql.DB
	hist *history.Recorder
}

// New returns a team service and makes sure built-in roles and starter archetypes exist.
func New(db *sql.DB, h *history.Recorder) *Service {
	for _, r := range BuiltinRoles {
		_, _ = db.Exec(`INSERT OR IGNORE INTO roles(id,name,description,builtin) VALUES(?,?,?,1)`, r.ID, r.Name, r.Description)
	}
	s := &Service{db: db, hist: h}
	for _, a := range StarterArchetypes {
		var n int
		_ = db.QueryRow(`SELECT COUNT(*) FROM archetypes WHERE id=?`, a.ID).Scan(&n)
		if n == 0 {
			a.Builtin = true
			_ = s.insertArchetype(context.Background(), a, 1, "go-build-it", "starter archetype")
		}
	}
	return s
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
	id = slug(id)
	if id == "" || name == "" {
		return Role{}, errors.New("id and name are required")
	}
	if _, err := s.db.ExecContext(ctx, `INSERT INTO roles(id,name,description,builtin) VALUES(?,?,?,0)`, id, name, desc); err != nil {
		return Role{}, err
	}
	s.hist.Must(ctx, history.Event{Type: "role.added", Subject: id, Summary: "Added role " + name})
	return Role{ID: id, Name: name, Description: desc}, nil
}

// ---- archetypes ----

const archCols = `id,name,role,instructions,harness,model,standards,rev,builtin,updated_at`

func scanArch(sc interface{ Scan(...any) error }) (Archetype, error) {
	var a Archetype
	var st string
	err := sc.Scan(&a.ID, &a.Name, &a.Role, &a.Instructions, &a.Harness, &a.Model, &st, &a.Rev, &a.Builtin, &a.UpdatedAt)
	a.Standards = decodeList(st)
	return a, err
}

// Archetypes lists archetypes with how many members use each.
func (s *Service) Archetypes(ctx context.Context) ([]Archetype, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT `+archCols+` FROM archetypes ORDER BY builtin DESC, name`)
	if err != nil {
		return nil, err
	}
	var out []Archetype
	for rows.Next() {
		a, err := scanArch(rows)
		if err != nil {
			rows.Close()
			return nil, err
		}
		out = append(out, a)
	}
	rows.Close()
	order := map[string]int{}
	for i, r := range BuiltinRoles {
		order[r.ID] = i
	}
	sort.SliceStable(out, func(i, j int) bool {
		oi, iok := order[out[i].Role]
		oj, jok := order[out[j].Role]
		if iok != jok {
			return iok
		}
		return oi < oj
	})
	for i := range out {
		_ = s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM members WHERE archetype_id=?`, out[i].ID).Scan(&out[i].UsedBy)
	}
	if out == nil {
		out = []Archetype{}
	}
	return out, nil
}

// Archetype returns one archetype.
func (s *Service) Archetype(ctx context.Context, id string) (Archetype, error) {
	a, err := scanArch(s.db.QueryRowContext(ctx, `SELECT `+archCols+` FROM archetypes WHERE id=?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return a, fmt.Errorf("archetype %q not found", id)
	}
	if err == nil {
		_ = s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM members WHERE archetype_id=?`, id).Scan(&a.UsedBy)
	}
	return a, err
}

// ArchetypeAt returns the archetype as it was at a revision.
func (s *Service) ArchetypeAt(ctx context.Context, id string, rev int) (Archetype, error) {
	var a Archetype
	var st string
	err := s.db.QueryRowContext(ctx, `SELECT archetype_id,rev,name,role,instructions,harness,model,standards FROM archetype_revisions WHERE archetype_id=? AND rev=?`, id, rev).
		Scan(&a.ID, &a.Rev, &a.Name, &a.Role, &a.Instructions, &a.Harness, &a.Model, &st)
	if errors.Is(err, sql.ErrNoRows) {
		return s.Archetype(ctx, id)
	}
	a.Standards = decodeList(st)
	return a, err
}

func (s *Service) insertArchetype(ctx context.Context, a Archetype, rev int, author, note string) error {
	if a.Standards == nil {
		a.Standards = []string{}
	}
	st, _ := json.Marshal(a.Standards)
	now := store.Now()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `INSERT INTO archetypes(`+archCols+`) VALUES(?,?,?,?,?,?,?,?,?,?)
		ON CONFLICT(id) DO UPDATE SET name=excluded.name, role=excluded.role, instructions=excluded.instructions, harness=excluded.harness,
		model=excluded.model, standards=excluded.standards, rev=excluded.rev, updated_at=excluded.updated_at`,
		a.ID, a.Name, a.Role, a.Instructions, a.Harness, a.Model, string(st), rev, a.Builtin, now); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO archetype_revisions(archetype_id,rev,name,role,instructions,harness,model,standards,author,note,created_at) VALUES(?,?,?,?,?,?,?,?,?,?,?)`,
		a.ID, rev, a.Name, a.Role, a.Instructions, a.Harness, a.Model, string(st), author, note, now); err != nil {
		return err
	}
	return tx.Commit()
}

// SaveArchetype creates an archetype or records a new revision of it.
func (s *Service) SaveArchetype(ctx context.Context, a Archetype) (Archetype, error) {
	a.ID = slug(firstNonEmpty(a.ID, a.Role, a.Name))
	if a.ID == "" || a.Name == "" || a.Role == "" {
		return a, errors.New("id, name and role are required")
	}
	if a.Harness == "" {
		return a, errors.New("an archetype needs a default harness")
	}
	rev := 1
	if cur, err := s.Archetype(ctx, a.ID); err == nil {
		rev = cur.Rev + 1
		a.Builtin = cur.Builtin
	}
	if err := s.insertArchetype(ctx, a, rev, access.FromContext(ctx).Name, ""); err != nil {
		return a, err
	}
	s.hist.Must(ctx, history.Event{Type: "archetype.saved", Subject: a.ID, Summary: fmt.Sprintf("Saved archetype %s (%s) rev %d", a.Name, a.Role, rev),
		Data: history.Data(map[string]any{"rev": rev, "instructions": a.Instructions})})
	return s.Archetype(ctx, a.ID)
}

// RemoveArchetype deletes an archetype no member uses.
func (s *Service) RemoveArchetype(ctx context.Context, id string) error {
	a, err := s.Archetype(ctx, id)
	if err != nil {
		return err
	}
	if a.UsedBy > 0 {
		return fmt.Errorf("%d members are made from %s; remove or re-base them first", a.UsedBy, a.Name)
	}
	if _, err := s.db.ExecContext(ctx, `DELETE FROM archetypes WHERE id=?`, id); err != nil {
		return err
	}
	_, _ = s.db.ExecContext(ctx, `DELETE FROM archetype_revisions WHERE archetype_id=?`, id)
	s.hist.Must(ctx, history.Event{Type: "archetype.removed", Subject: id, Summary: "Removed archetype " + a.Name})
	return nil
}

// Change is one affected member in an archetype change review (FR-12b).
type Change struct {
	Member        Member   `json:"member"`
	FromRev       int      `json:"fromRev"`
	ToRev         int      `json:"toRev"`
	Changed       []string `json:"changed"`   // archetype fields that changed between the revisions
	Overrides     []string `json:"overrides"` // of those, fields the member has overridden
	CanApplyClean bool     `json:"canApplyClean"`
}

// Changes lists members made from an archetype that are behind its latest revision.
func (s *Service) Changes(ctx context.Context, id string) ([]Change, error) {
	cur, err := s.Archetype(ctx, id)
	if err != nil {
		return nil, err
	}
	ms, err := s.query(ctx, `WHERE archetype_id=? AND archetype_rev<?`, id, cur.Rev)
	if err != nil {
		return nil, err
	}
	out := []Change{}
	for _, m := range ms {
		old, _ := s.ArchetypeAt(ctx, id, m.ArchetypeRev)
		c := Change{Member: m, FromRev: m.ArchetypeRev, ToRev: cur.Rev, Changed: []string{}, Overrides: []string{}}
		diff := map[string]bool{
			"instructions": strings.TrimSpace(old.Instructions) != strings.TrimSpace(cur.Instructions),
			"harness":      old.Harness != cur.Harness,
			"model":        old.Model != cur.Model,
			"standards":    strings.Join(old.Standards, "\n") != strings.Join(cur.Standards, "\n"),
			"name":         old.Name != cur.Name,
		}
		over := map[string]bool{"instructions": m.ReplaceInstructions, "harness": m.HarnessOverride != "", "model": m.ModelOverride != ""}
		for _, f := range []string{"instructions", "harness", "model", "standards", "name"} {
			if diff[f] {
				c.Changed = append(c.Changed, f)
				if over[f] {
					c.Overrides = append(c.Overrides, f)
				}
			}
		}
		c.CanApplyClean = len(c.Overrides) == 0
		out = append(out, c)
	}
	return out, nil
}

// Apply moves one member to its archetype's latest revision, keeping its tuning.
func (s *Service) Apply(ctx context.Context, memberID string) (Member, error) {
	m, err := s.Member(ctx, memberID)
	if err != nil {
		return m, err
	}
	if m.ArchetypeID == "" {
		return m, fmt.Errorf("%s is not made from an archetype", m.Name)
	}
	a, err := s.Archetype(ctx, m.ArchetypeID)
	if err != nil {
		return m, err
	}
	if _, err := s.db.ExecContext(ctx, `UPDATE members SET archetype_rev=? WHERE id=?`, a.Rev, m.ID); err != nil {
		return m, err
	}
	s.hist.Must(ctx, history.Event{ProjectID: m.ProjectID, Type: "member.archetype_applied", Subject: m.ID,
		Summary: fmt.Sprintf("Applied %s rev %d to %s (tuning kept)", a.Name, a.Rev, m.Name)})
	return s.Member(ctx, m.ID)
}

// ApplyClean applies the latest revision to every member that has not overridden a changed field (Q10).
func (s *Service) ApplyClean(ctx context.Context, archetypeID string) ([]Member, error) {
	cs, err := s.Changes(ctx, archetypeID)
	if err != nil {
		return nil, err
	}
	out := []Member{}
	for _, c := range cs {
		if !c.CanApplyClean {
			continue
		}
		m, err := s.Apply(ctx, c.Member.ID)
		if err != nil {
			return out, err
		}
		out = append(out, m)
	}
	return out, nil
}

// Promote moves lines of a member's tuning into its archetype as a new revision (FR-12c).
// The promoted lines are removed from the member's tuning and the member moves to the new revision.
func (s *Service) Promote(ctx context.Context, memberID, text string) (Archetype, Member, error) {
	m, err := s.Member(ctx, memberID)
	if err != nil {
		return Archetype{}, m, err
	}
	text = strings.TrimSpace(text)
	if text == "" || m.ArchetypeID == "" {
		return Archetype{}, m, errors.New("choose tuning text to promote from a member made from an archetype")
	}
	a, err := s.Archetype(ctx, m.ArchetypeID)
	if err != nil {
		return a, m, err
	}
	a.Instructions = strings.TrimRight(a.Instructions, "\n") + "\n" + text + "\n"
	rev := a.Rev + 1
	if err := s.insertArchetype(ctx, a, rev, access.FromContext(ctx).Name, "promoted from "+m.ID); err != nil {
		return a, m, err
	}
	tuning := m.Tuning
	for _, line := range strings.Split(text, "\n") {
		if l := strings.TrimSpace(line); l != "" {
			tuning = strings.Replace(tuning, line, "", 1)
		}
	}
	tuning = collapseBlank(tuning)
	if _, err := s.db.ExecContext(ctx, `UPDATE members SET tuning=?, archetype_rev=? WHERE id=?`, tuning, rev, m.ID); err != nil {
		return a, m, err
	}
	s.hist.Must(ctx, history.Event{ProjectID: m.ProjectID, Type: "archetype.promoted", Subject: a.ID,
		Summary: fmt.Sprintf("Promoted tuning from %s into %s rev %d", m.Name, a.Name, rev), Why: text})
	a, _ = s.Archetype(ctx, a.ID)
	m, _ = s.Member(ctx, m.ID)
	return a, m, nil
}

// ---- members ----

const memCols = `id,project_id,name,kind,role,login,archetype_id,archetype_rev,tuning,replace_instructions,harness,model,standards,worktree,pane_id,conversation_id,conversation_tasks,conversation_started,rev`

func (s *Service) query(ctx context.Context, where string, args ...any) ([]Member, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT `+memCols+` FROM members `+where+` ORDER BY project_id, kind, name`, args...)
	if err != nil {
		return nil, err
	}
	var raw []Member
	for rows.Next() {
		var m Member
		var st string
		if err := rows.Scan(&m.ID, &m.ProjectID, &m.Name, &m.Kind, &m.Role, &m.Login, &m.ArchetypeID, &m.ArchetypeRev, &m.Tuning, &m.ReplaceInstructions,
			&m.HarnessOverride, &m.ModelOverride, &st, &m.Worktree, &m.PaneID, &m.ConversationID, &m.ConversationTasks, &m.ConversationStarted, &m.Rev); err != nil {
			rows.Close()
			return nil, err
		}
		m.ExtraStandards = decodeList(st)
		raw = append(raw, m)
	}
	rows.Close()
	out := make([]Member, 0, len(raw))
	for _, m := range raw {
		out = append(out, s.effective(ctx, m))
	}
	return out, rows.Err()
}

// effective fills Harness, Model, Instructions and Standards from the archetype revision plus tuning.
func (s *Service) effective(ctx context.Context, m Member) Member {
	m.Harness, m.Model, m.Standards = m.HarnessOverride, m.ModelOverride, append([]string{}, m.ExtraStandards...)
	if m.Kind != "ai" || m.ArchetypeID == "" {
		m.Instructions = m.Tuning
		return m
	}
	a, err := s.ArchetypeAt(ctx, m.ArchetypeID, m.ArchetypeRev)
	if err != nil {
		m.Instructions = m.Tuning
		return m
	}
	if cur, err := s.Archetype(ctx, m.ArchetypeID); err == nil {
		m.ArchetypeLatest = cur.Rev
	}
	m.Harness = firstNonEmpty(m.HarnessOverride, a.Harness)
	m.Model = firstNonEmpty(m.ModelOverride, a.Model)
	m.Standards = append(append([]string{}, a.Standards...), m.ExtraStandards...)
	switch {
	case m.ReplaceInstructions:
		m.Instructions = m.Tuning
	case strings.TrimSpace(m.Tuning) != "":
		m.Instructions = strings.TrimRight(a.Instructions, "\n") + "\n\n## For this project\n" + strings.TrimSpace(m.Tuning) + "\n"
	default:
		m.Instructions = a.Instructions
	}
	return m
}

// Members lists a project's members; projectID "" lists every member.
func (s *Service) Members(ctx context.Context, projectID string) ([]Member, error) {
	if projectID == "" {
		return s.query(ctx, ``)
	}
	return s.query(ctx, `WHERE project_id=?`, projectID)
}

// Member returns one member. An archetype ID resolves to a virtual member made from the
// archetype (used for global work such as coaching standards, where no project is involved).
func (s *Service) Member(ctx context.Context, id string) (Member, error) {
	ms, err := s.query(ctx, `WHERE id=?`, id)
	if err != nil {
		return Member{}, err
	}
	if len(ms) == 1 {
		return ms[0], nil
	}
	if a, err := s.Archetype(ctx, id); err == nil {
		return Member{ID: a.ID, Name: a.Name, Kind: "ai", Role: a.Role, ArchetypeID: a.ID, ArchetypeRev: a.Rev, ArchetypeLatest: a.Rev,
			Harness: a.Harness, Model: a.Model, Instructions: a.Instructions, Standards: a.Standards, Rev: a.Rev}, nil
	}
	return Member{}, fmt.Errorf("member %q not found", id)
}

// ByRole returns a project's AI members holding a role ("" project: any project).
func (s *Service) ByRole(ctx context.Context, role, projectID string) ([]Member, error) {
	all, err := s.Members(ctx, projectID)
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

// MemberID builds a project member ID: <slug>@<project>.
func MemberID(name, projectID string) string { return slug(name) + "@" + projectID }

// AddFromArchetype creates an AI member in a project from an archetype (FR-12a).
func (s *Service) AddFromArchetype(ctx context.Context, projectID, archetypeID, name string) (Member, error) {
	a, err := s.Archetype(ctx, archetypeID)
	if err != nil {
		return Member{}, err
	}
	if projectID == "" {
		return Member{}, errors.New("a member belongs to a project")
	}
	name = firstNonEmpty(strings.TrimSpace(name), a.Name)
	id := MemberID(name, projectID)
	if _, err := s.Member(ctx, id); err == nil {
		return Member{}, fmt.Errorf("%s already has a member called %s", projectID, name)
	}
	if _, err := s.db.ExecContext(ctx, `INSERT INTO members(id,project_id,name,kind,role,archetype_id,archetype_rev,standards,rev) VALUES(?,?,?,?,?,?,?,?,1)`,
		id, projectID, name, "ai", a.Role, a.ID, a.Rev, "[]"); err != nil {
		return Member{}, err
	}
	s.hist.Must(ctx, history.Event{ProjectID: projectID, Type: "member.added", Subject: id, Summary: fmt.Sprintf("Added %s (%s) to %s from the %s archetype", name, a.Role, projectID, a.Name)})
	return s.Member(ctx, id)
}

// AddHuman gives a global person a role in a project (FR-12e).
func (s *Service) AddHuman(ctx context.Context, projectID, name, login, role string) (Member, error) {
	if projectID == "" || name == "" || role == "" {
		return Member{}, errors.New("project, name and role are required")
	}
	id := MemberID(name, projectID)
	if _, err := s.db.ExecContext(ctx, `INSERT INTO members(id,project_id,name,kind,role,login,standards,rev) VALUES(?,?,?,?,?,?,?,1)
		ON CONFLICT(id) DO UPDATE SET role=excluded.role, login=excluded.login`, id, projectID, name, "human", role, login, "[]"); err != nil {
		return Member{}, err
	}
	s.hist.Must(ctx, history.Event{ProjectID: projectID, Type: "member.added", Subject: id, Summary: fmt.Sprintf("%s is %s on %s", name, role, projectID)})
	return s.Member(ctx, id)
}

// Tuning is a member update: project-specific changes (FR-12a).
type Tuning struct {
	Name                *string   `json:"name"`
	Role                *string   `json:"role"`
	Tuning              *string   `json:"tuning"`
	ReplaceInstructions *bool     `json:"replaceInstructions"`
	Harness             *string   `json:"harness"` // "" clears the override
	Model               *string   `json:"model"`
	Standards           *[]string `json:"standards"`
	Login               *string   `json:"login"`
}

// Tune updates a member's project-specific settings.
func (s *Service) Tune(ctx context.Context, id string, t Tuning) (Member, error) {
	m, err := s.Member(ctx, id)
	if err != nil {
		return m, err
	}
	if m.ProjectID == "" {
		return m, fmt.Errorf("%s is an archetype; edit it in the archetype library", id)
	}
	set := func(p *string, dst *string) {
		if p != nil {
			*dst = strings.TrimSpace(*p)
		}
	}
	set(t.Name, &m.Name)
	set(t.Role, &m.Role)
	set(t.Harness, &m.HarnessOverride)
	set(t.Model, &m.ModelOverride)
	set(t.Login, &m.Login)
	if t.Tuning != nil {
		m.Tuning = *t.Tuning
	}
	if t.ReplaceInstructions != nil {
		m.ReplaceInstructions = *t.ReplaceInstructions
	}
	if t.Standards != nil {
		m.ExtraStandards = *t.Standards
	}
	if m.Name == "" || m.Role == "" {
		return m, errors.New("name and role are required")
	}
	st, _ := json.Marshal(nonNil(m.ExtraStandards))
	if _, err := s.db.ExecContext(ctx, `UPDATE members SET name=?, role=?, harness=?, model=?, tuning=?, replace_instructions=?, standards=?, login=?, rev=rev+1 WHERE id=?`,
		m.Name, m.Role, m.HarnessOverride, m.ModelOverride, m.Tuning, m.ReplaceInstructions, string(st), m.Login, m.ID); err != nil {
		return m, err
	}
	s.hist.Must(ctx, history.Event{ProjectID: m.ProjectID, Type: "member.tuned", Subject: m.ID, Summary: "Tuned " + m.Name + " for " + m.ProjectID,
		Data: history.Data(map[string]any{"tuning": m.Tuning, "harness": m.HarnessOverride, "model": m.ModelOverride})})
	return s.Member(ctx, m.ID)
}

// SetRuntime records where a member runs: its worktree, herdr pane and conversation (FR-12d).
func (s *Service) SetRuntime(ctx context.Context, id, worktree, paneID, conversationID string, tasks int, started string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE members SET worktree=?, pane_id=?, conversation_id=?, conversation_tasks=?, conversation_started=? WHERE id=?`,
		worktree, paneID, conversationID, tasks, started, id)
	return err
}

// Remove deletes a member.
func (s *Service) Remove(ctx context.Context, id string) error {
	m, err := s.Member(ctx, id)
	if err != nil {
		return err
	}
	if m.ProjectID == "" && m.ArchetypeID == m.ID {
		return fmt.Errorf("%s is an archetype; remove it from the archetype library", id)
	}
	if _, err := s.db.ExecContext(ctx, `DELETE FROM members WHERE id=?`, id); err != nil {
		return err
	}
	s.hist.Must(ctx, history.Event{ProjectID: m.ProjectID, Type: "member.removed", Subject: id, Summary: "Removed " + m.Name + " from " + m.ProjectID})
	return nil
}

// EnsureStarterTeam gives a project with no AI members the starter team (P12 gate).
func (s *Service) EnsureStarterTeam(ctx context.Context, projectID string) ([]Member, error) {
	ms, err := s.Members(ctx, projectID)
	if err != nil {
		return nil, err
	}
	for _, m := range ms {
		if m.Kind == "ai" {
			return ms, nil
		}
	}
	for _, id := range StarterTeam {
		if _, err := s.AddFromArchetype(ctx, projectID, id, ""); err != nil {
			return nil, err
		}
	}
	return s.Members(ctx, projectID)
}

// Persona renders a member's effective definition as Markdown (edited through the artifact editor, FR-13).
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

// ---- helpers ----

var nonSlug = regexp.MustCompile(`[^a-z0-9-]+`)

func slug(s string) string {
	return strings.Trim(nonSlug.ReplaceAllString(strings.ToLower(strings.TrimSpace(s)), "-"), "-")
}

func decodeList(s string) []string {
	var out []string
	_ = json.Unmarshal([]byte(s), &out)
	return nonNil(out)
}

func nonNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

func firstNonEmpty(a ...string) string {
	for _, s := range a {
		if s != "" {
			return s
		}
	}
	return ""
}

func collapseBlank(s string) string {
	lines := strings.Split(s, "\n")
	var out []string
	for _, l := range lines {
		if strings.TrimSpace(l) == "" && (len(out) == 0 || strings.TrimSpace(out[len(out)-1]) == "") {
			continue
		}
		out = append(out, l)
	}
	return strings.TrimSpace(strings.Join(out, "\n"))
}
