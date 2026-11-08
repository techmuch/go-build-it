// Package standards manages global standards, project overrides, and their delivery
// to agent sessions through generated instruction files (FR-50–FR-53).
package standards

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/techmuch/go-build-it/internal/history"
	"github.com/techmuch/go-build-it/internal/store"
)

// Standard is one global standard, authored in Markdown (FR-51).
type Standard struct {
	ID        string     `json:"id"`
	Title     string     `json:"title"`
	Body      string     `json:"body"`
	Rev       int        `json:"rev"`
	UpdatedAt string     `json:"updatedAt"`
	Overrides []Override `json:"overrides"`
}

// Override is a project's change to a standard, with its reason (FR-52).
type Override struct {
	StandardID string `json:"standardId"`
	ProjectID  string `json:"projectId"`
	Body       string `json:"body"`
	Reason     string `json:"reason"`
	CreatedAt  string `json:"createdAt"`
}

// Service manages standards.
type Service struct {
	db   *sql.DB
	hist *history.Recorder
}

// New returns a standards service.
func New(db *sql.DB, h *history.Recorder) *Service { return &Service{db: db, hist: h} }

// List returns all standards with their overrides.
func (s *Service) List(ctx context.Context) ([]Standard, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id,title,body,rev,updated_at FROM standards ORDER BY title`)
	if err != nil {
		return nil, err
	}
	out := []Standard{}
	for rows.Next() {
		var st Standard
		if err := rows.Scan(&st.ID, &st.Title, &st.Body, &st.Rev, &st.UpdatedAt); err != nil {
			rows.Close()
			return nil, err
		}
		out = append(out, st)
	}
	rows.Close()
	ovs, err := s.overrides(ctx, "")
	if err != nil {
		return nil, err
	}
	for i := range out {
		out[i].Overrides = []Override{}
		for _, o := range ovs {
			if o.StandardID == out[i].ID {
				out[i].Overrides = append(out[i].Overrides, o)
			}
		}
	}
	return out, nil
}

// Get returns one standard.
func (s *Service) Get(ctx context.Context, id string) (Standard, error) {
	all, err := s.List(ctx)
	if err != nil {
		return Standard{}, err
	}
	for _, st := range all {
		if st.ID == id {
			return st, nil
		}
	}
	return Standard{}, fmt.Errorf("standard %q not found", id)
}

// Save creates or updates a standard; updates bump the revision.
func (s *Service) Save(ctx context.Context, id, title, body, why string) (Standard, error) {
	if id == "" || strings.TrimSpace(body) == "" {
		return Standard{}, errors.New("id and body are required")
	}
	if title == "" {
		title = id
	}
	now := store.Now()
	_, err := s.db.ExecContext(ctx, `INSERT INTO standards(id,title,body,rev,updated_at) VALUES(?,?,?,1,?)
		ON CONFLICT(id) DO UPDATE SET title=excluded.title, body=excluded.body, rev=standards.rev+1, updated_at=excluded.updated_at`, id, title, body, now)
	if err != nil {
		return Standard{}, err
	}
	st, err := s.Get(ctx, id)
	if err == nil {
		s.hist.Must(ctx, history.Event{Type: "standard.saved", Subject: id, Summary: fmt.Sprintf("Standard %q rev %d", title, st.Rev), Why: why,
			Data: history.Data(map[string]any{"body": body, "rev": st.Rev})})
	}
	return st, err
}

// Remove deletes a standard and its overrides.
func (s *Service) Remove(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM standards WHERE id=?`, id)
	_, _ = s.db.ExecContext(ctx, `DELETE FROM standard_overrides WHERE standard_id=?`, id)
	if err == nil {
		s.hist.Must(ctx, history.Event{Type: "standard.removed", Subject: id, Summary: "Removed standard " + id})
	}
	return err
}

// SetOverride records a project override; a reason is mandatory (FR-52).
func (s *Service) SetOverride(ctx context.Context, standardID, projectID, body, reason string) (Override, error) {
	if strings.TrimSpace(reason) == "" {
		return Override{}, errors.New("an override needs a reason")
	}
	if _, err := s.Get(ctx, standardID); err != nil {
		return Override{}, err
	}
	o := Override{StandardID: standardID, ProjectID: projectID, Body: body, Reason: reason, CreatedAt: store.Now()}
	_, err := s.db.ExecContext(ctx, `INSERT INTO standard_overrides(standard_id,project_id,body,reason,created_at) VALUES(?,?,?,?,?)
		ON CONFLICT(standard_id,project_id) DO UPDATE SET body=excluded.body, reason=excluded.reason, created_at=excluded.created_at`,
		standardID, projectID, body, reason, o.CreatedAt)
	if err == nil {
		s.hist.Must(ctx, history.Event{ProjectID: projectID, Type: "standard.override", Subject: standardID, Summary: fmt.Sprintf("%s overrides %s", projectID, standardID), Why: reason})
	}
	return o, err
}

// RemoveOverride deletes a project override.
func (s *Service) RemoveOverride(ctx context.Context, standardID, projectID string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM standard_overrides WHERE standard_id=? AND project_id=?`, standardID, projectID)
	return err
}

func (s *Service) overrides(ctx context.Context, projectID string) ([]Override, error) {
	q, args := `SELECT standard_id,project_id,body,reason,created_at FROM standard_overrides`, []any{}
	if projectID != "" {
		q += ` WHERE project_id=?`
		args = append(args, projectID)
	}
	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Override
	for rows.Next() {
		var o Override
		if err := rows.Scan(&o.StandardID, &o.ProjectID, &o.Body, &o.Reason, &o.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, o)
	}
	return out, rows.Err()
}

// Effective returns the standards that apply to a project, with overrides applied.
func (s *Service) Effective(ctx context.Context, projectID string) ([]Standard, error) {
	all, err := s.List(ctx)
	if err != nil {
		return nil, err
	}
	for i := range all {
		for _, o := range all[i].Overrides {
			if o.ProjectID == projectID {
				all[i].Body = o.Body + "\n\n> Project override: " + o.Reason
			}
		}
	}
	return all, nil
}

// Render produces the instruction file content for a project and optional persona text (FR-53).
func Render(project string, stds []Standard, personaMD string) string {
	var b strings.Builder
	b.WriteString("<!-- Generated by go-build-it. Edit standards in go-build-it, not here. -->\n")
	fmt.Fprintf(&b, "# Standards for %s\n\n", project)
	sort.Slice(stds, func(i, j int) bool { return stds[i].Title < stds[j].Title })
	for _, st := range stds {
		body := strings.TrimSpace(st.Body)
		if !strings.HasPrefix(body, "#") {
			fmt.Fprintf(&b, "## %s\n\n", st.Title)
		} else {
			body = "#" + body // demote one level
		}
		b.WriteString(body)
		b.WriteString("\n\n")
	}
	if personaMD != "" {
		b.WriteString("---\n\n")
		b.WriteString(strings.TrimSpace(personaMD))
		b.WriteString("\n")
	}
	b.WriteString("\n## Working with go-build-it\n\n- Use `gbi questions ask` to ask the human a question (add `--blocking` when you cannot continue).\n- Never run git push or open PRs: the Git / Release Manager does that.\n")
	return b.String()
}

// InstructionFiles are the per-harness file names that carry standards (FR-53).
var InstructionFiles = []string{"AGENTS.md", "CLAUDE.md", "GEMINI.md"}

// WriteFiles writes the instruction files into dir (a session worktree).
func WriteFiles(dir, content string) ([]string, error) {
	var written []string
	for _, f := range InstructionFiles {
		p := filepath.Join(dir, f)
		if b, err := os.ReadFile(p); err == nil && !strings.Contains(string(b), "Generated by go-build-it") {
			// Keep a project's own file; append our block below it.
			content2 := string(b) + "\n\n" + content
			if err := os.WriteFile(p, []byte(content2), 0o644); err != nil {
				return written, err
			}
		} else if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			return written, err
		}
		written = append(written, f)
	}
	return written, nil
}

// SeedFromDir loads *.md files from dir as standards when none exist yet.
func (s *Service) SeedFromDir(ctx context.Context, dir string) (int, error) {
	var n int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM standards`).Scan(&n); err != nil || n > 0 {
		return 0, err
	}
	files, _ := filepath.Glob(filepath.Join(dir, "*.md"))
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		id := strings.TrimSuffix(filepath.Base(f), ".md")
		title := id
		if first, _, ok := strings.Cut(string(b), "\n"); ok && strings.HasPrefix(first, "# ") {
			title = strings.TrimPrefix(first, "# ")
		}
		if _, err := s.Save(ctx, id, title, string(b), "seeded from "+dir); err == nil {
			n++
		}
	}
	return n, nil
}
