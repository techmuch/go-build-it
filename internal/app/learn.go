package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/techmuch/go-build-it/docs"
	"github.com/techmuch/go-build-it/internal/access"
	"github.com/techmuch/go-build-it/internal/api"
	"github.com/techmuch/go-build-it/internal/herdr"
	"github.com/techmuch/go-build-it/internal/history"
	"github.com/techmuch/go-build-it/internal/learn"
	"github.com/techmuch/go-build-it/internal/projects"
	"github.com/techmuch/go-build-it/internal/queue"
	"github.com/techmuch/go-build-it/internal/team"
)

// Learn: lessons, progress and the practice project (FR-134–FR-137, ADR-012).

func (a *App) learnRoutes() []api.Route {
	return []api.Route{
		r("GET", "/api/learn", "learn.list", "learn list", "Lessons with your progress, and the practice project (FR-134)", a.learnList),
		r("GET", "/api/learn/{slug}", "learn.get", "learn get", "One lesson with its steps and your progress", a.learnGet),
		r("POST", "/api/learn/{slug}/start", "learn.start", "learn start", "Start (or restart) a lesson: steps check themselves off from now on", a.learnStart),
		r("POST", "/api/learn/{slug}/done", "learn.done", "learn done", "Tick off a step yourself", a.learnDone, fi("step", "step number")),
		r("POST", "/api/learn/visit", "learn.visit", "learn visit", "Record that you opened a view (checks off visit steps)", a.learnVisit, f("path", "the view's path")),
		r("POST", "/api/learn/practice", "learn.practice", "learn practice", "Create the practice project (scripted harness, no subscription), or reset or remove it (FR-136)", a.learnPractice, fb("reset", "recreate it from scratch"), fb("remove", "remove it and its files")),
		r("GET", "/api/agent/guide", "agent.guide", "agent guide", "The guide for AI agents driving gbi; save it as a harness skill (FR-137). Also works without the server.", func(c *api.Ctx) (any, error) {
			return map[string]string{"markdown": docs.AgentGuide()}, nil
		}),
	}
}

func (a *App) lessons() []learn.Lesson {
	a.lessonsOnce.Do(func() {
		ls, err := learn.Parse(docs.LearnFS())
		if err != nil {
			a.Log.Error("lessons do not parse", "err", err)
		}
		a.lessonList = ls
	})
	return a.lessonList
}

func user(ctx context.Context) string {
	if n := access.FromContext(ctx).Name; n != "" {
		return n
	}
	return "owner"
}

type practiceInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Path string `json:"path"`
}

func (a *App) practiceProject(ctx context.Context) *practiceInfo {
	p, err := a.Projects.Get(ctx, learn.PracticeID)
	if err != nil {
		return nil
	}
	return &practiceInfo{ID: p.ID, Name: p.Name, Path: p.Path}
}

func (a *App) learnList(c *api.Ctx) (any, error) {
	ls, err := a.Learn.Status(c.R.Context(), user(c.R.Context()), a.lessons())
	if err != nil {
		return nil, err
	}
	return map[string]any{"lessons": ls, "practice": a.practiceProject(c.R.Context())}, nil
}

func (a *App) lesson(c *api.Ctx) (learn.Lesson, error) {
	ls, err := a.Learn.Status(c.R.Context(), user(c.R.Context()), a.lessons())
	if err != nil {
		return learn.Lesson{}, err
	}
	slug := c.Param("slug")
	for _, l := range ls {
		if l.Slug == slug || fmt.Sprint(l.Number) == slug {
			return l, nil
		}
	}
	return learn.Lesson{}, notFound(fmt.Errorf("no lesson %q", slug))
}

func (a *App) learnGet(c *api.Ctx) (any, error) {
	l, err := a.lesson(c)
	if err != nil {
		return nil, err
	}
	return map[string]any{"lesson": l, "practice": a.practiceProject(c.R.Context())}, nil
}

func (a *App) learnStart(c *api.Ctx) (any, error) {
	l, err := a.lesson(c)
	if err != nil {
		return nil, err
	}
	if err := a.Learn.Start(c.R.Context(), user(c.R.Context()), l.Slug); err != nil {
		return nil, err
	}
	return a.learnGet(c)
}

func (a *App) learnDone(c *api.Ctx) (any, error) {
	var b struct{ Step int }
	if err := c.Bind(&b); err != nil {
		return nil, err
	}
	l, err := a.lesson(c)
	if err != nil {
		return nil, err
	}
	if b.Step < 1 || b.Step > len(l.Steps) {
		return nil, bad(fmt.Errorf("lesson %s has steps 1–%d", l.Slug, len(l.Steps)))
	}
	if l.Started == "" {
		if err := a.Learn.Start(c.R.Context(), user(c.R.Context()), l.Slug); err != nil {
			return nil, err
		}
	}
	if err := a.Learn.Mark(c.R.Context(), user(c.R.Context()), l.Slug, b.Step, "tick"); err != nil {
		return nil, err
	}
	return a.learnGet(c)
}

func (a *App) learnVisit(c *api.Ctx) (any, error) {
	var b struct{ Path string }
	if err := c.Bind(&b); err != nil {
		return nil, err
	}
	n, err := a.Learn.Visit(c.R.Context(), user(c.R.Context()), b.Path, a.lessons())
	if err != nil {
		return nil, err
	}
	if n > 0 {
		a.Hist.Publish(history.Event{Type: "learn.progress", Summary: "lesson step done"})
	}
	return map[string]int{"marked": n}, nil
}

func (a *App) learnPractice(c *api.Ctx) (any, error) {
	var b struct{ Reset, Remove bool }
	if err := c.Bind(&b); err != nil {
		return nil, err
	}
	ctx := c.R.Context()
	a.practiceMu.Lock()
	defer a.practiceMu.Unlock()
	if b.Reset || b.Remove {
		if err := a.removePractice(ctx); err != nil {
			return nil, err
		}
		if b.Remove {
			a.Hist.Must(ctx, history.Event{Type: "learn.practice", Subject: learn.PracticeID, Summary: "Removed the practice project"})
			return map[string]any{"practice": nil}, nil
		}
	}
	if p := a.practiceProject(ctx); p != nil {
		return map[string]any{"practice": p}, nil
	}
	if err := a.createPractice(ctx); err != nil {
		return nil, err
	}
	verb := "Created"
	if b.Reset {
		verb = "Reset"
	}
	a.Hist.Must(ctx, history.Event{Type: "learn.practice", Subject: learn.PracticeID, Summary: verb + " the practice project (scripted harness, no subscription used)"})
	return map[string]any{"practice": a.practiceProject(ctx)}, nil
}

func (a *App) practiceRoot() string { return filepath.Join(a.Cfg.Dir(), "practice") }

const practicePurpose = "Practice project: a small bird-sighting log, used by the Learn lessons. Its AI members use the scripted harness, so nothing here uses an AI subscription."

// createPractice seeds a small repository and imports it as the practice project, with
// the starter team switched to the scripted harness and the current user as PM.
func (a *App) createPractice(ctx context.Context) error {
	repo := filepath.Join(a.practiceRoot(), "birdwatch")
	files := map[string]string{
		"README.md":                       "# birdwatch\n\nA small log of bird sightings: what, where, when. This is go-build-it's **practice project**; the Learn lessons use it, and its AI members are scripted, so nothing here uses an AI subscription.\n",
		"docs/requirements.md":            "# birdwatch — Requirements\n\n## 1. Sightings\n- **FR-01**: The user SHALL record a sighting with species, place, time and an optional note.\n- **FR-02**: The user SHALL see their sightings newest first.\n- **FR-03**: The species field SHOULD suggest names from a species list.\n\n## 2. Offline\n- **FR-04**: Recording a sighting SHALL work without a network connection.\n",
		"docs/mockups/sighting-form.html": `<!doctype html><html><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>Sighting form</title><style>body{margin:0;font:14px -apple-system,sans-serif;background:#f4f7f2}header{background:#2f5d3a;color:#fff;padding:14px 18px;font-weight:700}main{padding:18px}label{display:block;margin:10px 0 4px;font-weight:600}input,textarea{width:100%;padding:8px;border:1px solid #c9d3c4;border-radius:8px}button{margin-top:14px;background:#2f5d3a;color:#fff;border:0;border-radius:8px;padding:10px 16px}</style></head><body><header>New sighting</header><main><label>Species</label><input value="Eurasian wren"><label>Place</label><input value="Riverside path"><label>Note</label><textarea>Singing from the hedge</textarea><button>Save sighting</button></main></body></html>`,
	}
	for name, body := range files {
		f := filepath.Join(repo, name)
		if err := os.MkdirAll(filepath.Dir(f), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(f, []byte(body), 0o644); err != nil {
			return err
		}
	}
	git := func(args ...string) error {
		cmd := exec.CommandContext(ctx, "git", append([]string{"-c", "user.name=go-build-it", "-c", "user.email=practice@go-build-it.local", "-c", "commit.gpgsign=false"}, args...)...)
		cmd.Dir = repo
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("git %s: %v: %s", strings.Join(args, " "), err, strings.TrimSpace(string(out)))
		}
		return nil
	}
	for _, args := range [][]string{{"init", "-q", "-b", "main"}, {"add", "-A"}, {"commit", "-q", "-m", "Practice project: birdwatch"}} {
		if err := git(args...); err != nil {
			return err
		}
	}
	p, err := a.Projects.Import(ctx, repo, learn.PracticeID, practicePurpose, "Design")
	if err != nil {
		return err
	}
	if p, err = a.Projects.Update(ctx, p.ID, "birdwatch", "", "", "", ""); err != nil {
		return err
	}
	ms, err := a.Team.EnsureStarterTeam(ctx, p.ID)
	if err != nil {
		return err
	}
	fake := "fake"
	for _, m := range ms {
		if m.Kind == "ai" {
			if _, err := a.Team.Tune(ctx, m.ID, team.Tuning{Harness: &fake}); err != nil {
				return err
			}
		}
	}
	if _, err := a.Team.AddHuman(ctx, p.ID, user(ctx), "", "pm"); err != nil {
		a.Log.Warn("practice: could not add you as PM", "err", err)
	}
	m, err := a.Projects.SaveMilestone(ctx, projects.Milestone{ProjectID: p.ID, Name: "M1 · Sighting form", Scope: "Record a sighting (FR-01) and list sightings (FR-02).", Status: "active", GateName: "Sighting form review", GateKind: "review", GateCriteria: []string{"A sighting can be saved", "The list shows it first"}})
	if err != nil {
		return err
	}
	_, err = a.Queue.Add(ctx, queue.Task{ProjectID: p.ID, MilestoneID: m.ID, Title: "Sighting form", Description: "Build the sighting form from docs/mockups/sighting-form.html.", Role: "developer", Priority: 2, Reqs: []string{"FR-01"}})
	return err
}

// removePractice stops the practice project's work and removes it and its files.
func (a *App) removePractice(ctx context.Context) error {
	if _, err := a.Projects.Get(ctx, learn.PracticeID); err == nil {
		if live, err := a.Sessions.List(ctx, learn.PracticeID, "", true); err == nil {
			for _, s := range live {
				_ = a.Sessions.Cancel(ctx, s.ID)
			}
		}
		a.closePracticePanes(ctx)
		if ms, err := a.Team.Members(ctx, learn.PracticeID); err == nil {
			for _, m := range ms {
				_ = a.Team.Remove(ctx, m.ID)
			}
		}
		if err := a.purgeProjectRows(ctx, learn.PracticeID); err != nil {
			return err
		}
		if err := a.Projects.Remove(ctx, learn.PracticeID); err != nil {
			return err
		}
	}
	_ = os.RemoveAll(filepath.Join(a.Cfg.Dir(), "worktrees", learn.PracticeID))
	return os.RemoveAll(a.practiceRoot())
}

func (a *App) closePracticePanes(ctx context.Context) {
	cctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	snap, err := a.Herdr.C.Snapshot(cctx)
	if err != nil {
		return
	}
	roots := []string{a.practiceRoot(), filepath.Join(a.Cfg.Dir(), "worktrees", learn.PracticeID)}
	for _, w := range snap.Workspaces {
		hit := w.Label == learn.PracticeID
		if w.Worktree != nil {
			for _, r := range roots {
				hit = hit || strings.HasPrefix(w.Worktree.CheckoutPath, r)
			}
		}
		if hit {
			_ = a.Herdr.C.Call(cctx, herdr.MethodWorkspaceClose, map[string]any{"workspace_id": w.WorkspaceID}, nil)
		}
	}
}

// purgeProjectRows deletes a project's rows from every table with a project_id column
// (except the history, which stays an audit trail). Tables are retried until foreign keys
// allow it.
func (a *App) purgeProjectRows(ctx context.Context, id string) error {
	rows, err := a.DB.QueryContext(ctx, `SELECT m.name FROM sqlite_master m JOIN pragma_table_info(m.name) c
		WHERE m.type='table' AND c.name='project_id' AND m.name NOT IN ('events', 'projects')`)
	if err != nil {
		return err
	}
	var tables []string
	for rows.Next() {
		var t string
		if err := rows.Scan(&t); err == nil {
			tables = append(tables, t)
		}
	}
	rows.Close()
	for pass := 0; pass < len(tables)+1 && len(tables) > 0; pass++ {
		var left []string
		var last error
		for _, t := range tables {
			if _, err := a.DB.ExecContext(ctx, `DELETE FROM "`+t+`" WHERE project_id=?`, id); err != nil {
				left, last = append(left, t), err
			}
		}
		tables = left
		if len(tables) == 0 {
			return nil
		}
		if pass == len(tables) {
			return errors.Join(fmt.Errorf("could not clear %v", tables), last)
		}
	}
	return nil
}

// SeedStandards gives a new installation its standards: *.md files in ~/.go-build-it/standards
// when that folder has any, otherwise the starter set shipped in docs/standards. An
// installation that already has standards is left alone. Project documents (FR-150) is
// always ensured. It returns how many standards it created.
func (a *App) SeedStandards(ctx context.Context) int {
	n := 0
	if a.Standards.Empty(ctx) {
		n, _ = a.Standards.SeedFromDir(ctx, filepath.Join(a.Cfg.Dir(), "standards"))
		if n == 0 {
			for _, d := range docs.DefaultStandards() {
				if _, err := a.Standards.Save(ctx, d.ID, d.Title, d.Body, "starter standard shipped with go-build-it"); err == nil {
					n++
				}
			}
		}
	}
	if ok, _ := a.Standards.EnsureBuiltin(ctx, "project-documents", "Project documents", docs.ProjectDocumentsStandard()); ok {
		n++
	}
	return n
}
