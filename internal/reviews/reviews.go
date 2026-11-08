// Package reviews assesses completed work and records human verdicts (FR-54, FR-110–FR-112).
package reviews

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/techmuch/go-build-it/internal/access"
	"github.com/techmuch/go-build-it/internal/history"
	"github.com/techmuch/go-build-it/internal/projects"
	"github.com/techmuch/go-build-it/internal/queue"
	"github.com/techmuch/go-build-it/internal/standards"
	"github.com/techmuch/go-build-it/internal/store"
)

// Check is one assessed item.
type Check struct {
	Item     string `json:"item"`
	Status   string `json:"status"` // met | partly | not | unassessed
	Evidence string `json:"evidence"`
	Author   string `json:"author,omitempty"`
}

// MockupPair compares an approved mockup with a screenshot of the build (FR-112).
type MockupPair struct {
	Name   string `json:"name"`
	Mockup string `json:"mockup"` // path relative to the project/worktree
	Built  string `json:"built"`
	Note   string `json:"note"`
}

// Tests summarizes the test run.
type Tests struct {
	Ran        bool   `json:"ran"`
	Command    string `json:"command"`
	UnitPassed int    `json:"unitPassed"`
	UnitFailed int    `json:"unitFailed"`
	E2EPassed  int    `json:"e2ePassed"`
	E2EFailed  int    `json:"e2eFailed"`
	Coverage   string `json:"coverage"`
	Output     string `json:"output"`
}

// Assessment is the review content (FR-112).
type Assessment struct {
	Mockups        []MockupPair `json:"mockups"`
	Tests          Tests        `json:"tests"`
	Standards      []Check      `json:"standards"`
	Feedback       []Check      `json:"feedback"`
	Requirements   []Check      `json:"requirements"`
	DiffStat       string       `json:"diffStat"`
	Summary        string       `json:"summary"`
	Recommendation string       `json:"recommendation"`
	GeneratedAt    string       `json:"generatedAt"`
	PRURL          string       `json:"prUrl"`
}

// Review is a pending or decided review.
type Review struct {
	ID         string     `json:"id"`
	ProjectID  string     `json:"projectId"`
	TaskID     string     `json:"taskId"`
	Status     string     `json:"status"`
	Assessment Assessment `json:"assessment"`
	Verdict    string     `json:"verdict"`
	Comment    string     `json:"comment"`
	Reviewer   string     `json:"reviewer"`
	CreatedAt  string     `json:"createdAt"`
	DecidedAt  string     `json:"decidedAt"`
	TaskTitle  string     `json:"taskTitle"`
	Assignee   string     `json:"assignee"` // human reviewer assigned (P9-08)
}

// Hooks connect reviews to Courier, Lens and rehearsal feedback.
type Hooks struct {
	Merge    func(ctx context.Context, taskID string) error
	Lens     func(ctx context.Context, prompt string) (string, error)
	Feedback func(ctx context.Context, t queue.Task) []Check
	Workdir  func(projectID, taskID string) string
	Assign   func(ctx context.Context, t queue.Task) string // picks a human reviewer (P9-08)
}

// Service manages reviews.
type Service struct {
	db        *sql.DB
	hist      *history.Recorder
	queue     *queue.Service
	projects  *projects.Service
	standards *standards.Service
	Hooks     Hooks
	// RunTests can be disabled in tests.
	RunTests bool
}

// New returns a review service.
func New(db *sql.DB, h *history.Recorder, q *queue.Service, p *projects.Service, st *standards.Service) *Service {
	return &Service{db: db, hist: h, queue: q, projects: p, standards: st, RunTests: true}
}

const cols = `id,project_id,task_id,status,assessment,verdict,comment,reviewer,created_at,decided_at,assignee`

func scan(sc interface{ Scan(...any) error }) (Review, error) {
	var r Review
	var a string
	err := sc.Scan(&r.ID, &r.ProjectID, &r.TaskID, &r.Status, &a, &r.Verdict, &r.Comment, &r.Reviewer, &r.CreatedAt, &r.DecidedAt, &r.Assignee)
	_ = json.Unmarshal([]byte(a), &r.Assessment)
	return r, err
}

// Open creates a pending review for a task and runs the assessment.
func (s *Service) Open(ctx context.Context, taskID string) (Review, error) {
	t, err := s.queue.Get(ctx, taskID)
	if err != nil {
		return Review{}, err
	}
	id, err := store.NextID(s.db, "R")
	if err != nil {
		return Review{}, err
	}
	r := Review{ID: id, ProjectID: t.ProjectID, TaskID: t.ID, Status: "pending", CreatedAt: store.Now()}
	if s.Hooks.Assign != nil {
		r.Assignee = s.Hooks.Assign(ctx, t)
	}
	r.Assessment = s.Assess(ctx, t)
	if err := s.write(ctx, r); err != nil {
		return r, err
	}
	if t.Status != queue.Review {
		_, _ = s.queue.SetStatus(ctx, t.ID, queue.Review, "review "+id+" opened")
	}
	s.hist.Must(access.Persona(ctx, "Lens"), history.Event{ProjectID: t.ProjectID, Type: "review.opened", Subject: id,
		Summary: fmt.Sprintf("Review %s for %s: %s", id, t.ID, firstNonEmpty(r.Assessment.Recommendation, "ready")), Why: r.Assessment.Summary})
	r.TaskTitle = t.Title
	return r, nil
}

func (s *Service) write(ctx context.Context, r Review) error {
	a, _ := json.Marshal(r.Assessment)
	_, err := s.db.ExecContext(ctx, `INSERT INTO reviews(`+cols+`) VALUES(?,?,?,?,?,?,?,?,?,?,?)
		ON CONFLICT(id) DO UPDATE SET status=excluded.status, assessment=excluded.assessment, verdict=excluded.verdict, comment=excluded.comment,
		reviewer=excluded.reviewer, decided_at=excluded.decided_at, assignee=excluded.assignee`,
		r.ID, r.ProjectID, r.TaskID, r.Status, string(a), r.Verdict, r.Comment, r.Reviewer, r.CreatedAt, r.DecidedAt, r.Assignee)
	return err
}

// Get returns one review.
func (s *Service) Get(ctx context.Context, id string) (Review, error) {
	r, err := scan(s.db.QueryRowContext(ctx, `SELECT `+cols+` FROM reviews WHERE id=?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return r, fmt.Errorf("review %s not found", id)
	}
	if t, err := s.queue.Get(ctx, r.TaskID); err == nil {
		r.TaskTitle = t.Title
	}
	return r, err
}

// List returns reviews, pending first.
func (s *Service) List(ctx context.Context, projectID, status string) ([]Review, error) {
	q, args := `SELECT `+cols+` FROM reviews WHERE 1=1`, []any{}
	if projectID != "" {
		q, args = q+` AND project_id=?`, append(args, projectID)
	}
	if status != "" && status != "all" {
		q, args = q+` AND status=?`, append(args, status)
	}
	rows, err := s.db.QueryContext(ctx, q+` ORDER BY status='pending' DESC, created_at DESC`, args...)
	if err != nil {
		return nil, err
	}
	out := []Review{}
	for rows.Next() {
		r, err := scan(rows)
		if err != nil {
			rows.Close()
			return nil, err
		}
		out = append(out, r)
	}
	rows.Close()
	for i := range out {
		if t, err := s.queue.Get(ctx, out[i].TaskID); err == nil {
			out[i].TaskTitle = t.Title
		}
	}
	return out, nil
}

// Reassess reruns the assessment for a pending review.
func (s *Service) Reassess(ctx context.Context, id string) (Review, error) {
	r, err := s.Get(ctx, id)
	if err != nil {
		return r, err
	}
	t, err := s.queue.Get(ctx, r.TaskID)
	if err != nil {
		return r, err
	}
	r.Assessment = s.Assess(ctx, t)
	return r, s.write(ctx, r)
}

// Verdict records the human decision (FR-110). Approve merges via Courier; changes requeues the
// task with the feedback as context (FR-111); reject fails the task.
func (s *Service) Verdict(ctx context.Context, id, verdict, comment string) (Review, error) {
	r, err := s.Get(ctx, id)
	if err != nil {
		return r, err
	}
	if r.Status != "pending" {
		return r, fmt.Errorf("review %s is already %s", id, r.Status)
	}
	a := access.FromContext(ctx)
	if a.Kind != "human" {
		return r, errors.New("only a human can give the verdict")
	}
	t, err := s.queue.Get(ctx, r.TaskID)
	if err != nil {
		return r, err
	}
	switch verdict {
	case "approve":
		r.Status = "approved"
	case "changes":
		r.Status = "changes"
		if strings.TrimSpace(comment) == "" {
			return r, errors.New("say what should change")
		}
	case "reject":
		r.Status = "rejected"
	default:
		return r, errors.New("verdict must be approve, changes or reject")
	}
	r.Verdict, r.Comment, r.Reviewer, r.DecidedAt = verdict, comment, a.Name, store.Now()
	if err := s.write(ctx, r); err != nil {
		return r, err
	}
	s.hist.Must(ctx, history.Event{ProjectID: r.ProjectID, Type: "review." + r.Status, Subject: id, Summary: fmt.Sprintf("%s: %s %s", a.Name, verdict, t.ID), Why: comment,
		Data: history.Data(map[string]any{"task": t.ID, "assessment": r.Assessment.Recommendation})})
	switch verdict {
	case "approve":
		if s.Hooks.Merge != nil {
			if err := s.Hooks.Merge(ctx, t.ID); err != nil {
				return r, fmt.Errorf("approved, but Courier could not merge: %w", err)
			}
		} else {
			_, _ = s.queue.SetStatus(ctx, t.ID, queue.Done, "approved in "+id)
		}
	case "changes":
		t.ContextNote = strings.TrimSpace(t.ContextNote + fmt.Sprintf("\n\nReview %s by %s requested changes: %s", id, a.Name, comment))
		_, _ = s.queue.Update(ctx, t)
		_, _ = s.queue.SetStatus(ctx, t.ID, queue.Queued, "changes requested in "+id)
	case "reject":
		_, _ = s.queue.SetStatus(ctx, t.ID, queue.Failed, "rejected in "+id+": "+comment)
	}
	return r, nil
}

// Assess builds the assessment for a task (FR-112).
func (s *Service) Assess(ctx context.Context, t queue.Task) Assessment {
	a := Assessment{GeneratedAt: store.Now(), PRURL: t.PRURL, Mockups: []MockupPair{}, Standards: []Check{}, Feedback: []Check{}, Requirements: []Check{}}
	p, err := s.projects.Get(ctx, t.ProjectID)
	if err != nil {
		a.Summary = err.Error()
		return a
	}
	dir := p.Path
	if s.Hooks.Workdir != nil {
		if wd := s.Hooks.Workdir(p.ID, t.ID); exists(wd) {
			dir = wd
		}
	}
	a.Mockups = PairMockups(dir)
	if s.RunTests {
		a.Tests = RunTests(ctx, dir)
	}
	if t.Branch != "" {
		if out, err := exec.CommandContext(ctx, "git", "-C", dir, "diff", "--stat", "HEAD~1..HEAD").Output(); err == nil {
			a.DiffStat = strings.TrimSpace(string(out))
		}
	}
	reqText := requirementText(filepath.Join(dir, "docs", "requirements.md"))
	for _, r := range t.Reqs {
		a.Requirements = append(a.Requirements, Check{Item: strings.TrimSpace(r + " " + reqText[r]), Status: "unassessed"})
	}
	if stds, err := s.standards.Effective(ctx, p.ID); err == nil {
		for _, st := range stds {
			a.Standards = append(a.Standards, Check{Item: st.Title, Status: "unassessed"})
		}
	}
	if s.Hooks.Feedback != nil {
		a.Feedback = s.Hooks.Feedback(ctx, t)
	}
	if a.Tests.Ran && (a.Tests.UnitFailed > 0 || a.Tests.E2EFailed > 0) {
		a.Recommendation = "changes"
	}
	if s.Hooks.Lens != nil {
		s.lens(ctx, t, &a)
	}
	if a.Recommendation == "" {
		a.Recommendation = "approve"
		for _, group := range [][]Check{a.Requirements, a.Standards, a.Feedback} {
			for _, c := range group {
				if c.Status == "not" || c.Status == "partly" {
					a.Recommendation = "changes"
				}
			}
		}
	}
	return a
}

// lens asks the Reviewer persona to fill in statuses as JSON.
func (s *Service) lens(ctx context.Context, t queue.Task, a *Assessment) {
	type item struct {
		Group string `json:"group"`
		Index int    `json:"index"`
		Item  string `json:"item"`
	}
	var items []item
	for i, c := range a.Requirements {
		items = append(items, item{"requirements", i, c.Item})
	}
	for i, c := range a.Standards {
		items = append(items, item{"standards", i, c.Item})
	}
	for i, c := range a.Feedback {
		items = append(items, item{"feedback", i, c.Item})
	}
	ib, _ := json.Marshal(items)
	prompt := fmt.Sprintf(`Assess task %s "%s" for review.
Diff summary:
%s
Tests: %+v

Items to assess (JSON): %s

Answer ONLY with JSON: {"summary": "two sentences", "recommendation": "approve|changes", "checks": [{"group":"…","index":0,"status":"met|partly|not","evidence":"…"}]}`,
		t.ID, t.Title, trunc(a.DiffStat, 3000), a.Tests, ib)
	out, err := s.Hooks.Lens(ctx, prompt)
	if err != nil {
		a.Summary = "Lens could not assess: " + err.Error()
		return
	}
	var res struct {
		Summary        string `json:"summary"`
		Recommendation string `json:"recommendation"`
		Checks         []struct {
			Group    string `json:"group"`
			Index    int    `json:"index"`
			Status   string `json:"status"`
			Evidence string `json:"evidence"`
		} `json:"checks"`
	}
	if m := regexp.MustCompile(`(?s)\{.*\}`).FindString(out); m == "" || json.Unmarshal([]byte(m), &res) != nil {
		a.Summary = trunc(out, 600)
		return
	}
	a.Summary = res.Summary
	if res.Recommendation == "approve" || res.Recommendation == "changes" {
		if a.Recommendation != "changes" {
			a.Recommendation = res.Recommendation
		}
	}
	for _, c := range res.Checks {
		var group []Check
		switch c.Group {
		case "requirements":
			group = a.Requirements
		case "standards":
			group = a.Standards
		case "feedback":
			group = a.Feedback
		}
		if c.Index >= 0 && c.Index < len(group) {
			group[c.Index].Status, group[c.Index].Evidence = c.Status, c.Evidence
		}
	}
}

// PairMockups pairs approved mockups with e2e screenshots by name (FR-112). Mockups live under
// docs/mockups; screenshots under e2e-screenshots or web/e2e-screenshots.
func PairMockups(dir string) []MockupPair {
	shots := map[string]string{}
	for _, d := range []string{"e2e-screenshots", "web/e2e-screenshots", "docs/user/screenshots"} {
		files, _ := filepath.Glob(filepath.Join(dir, d, "*.png"))
		for _, f := range files {
			rel, _ := filepath.Rel(dir, f)
			shots[stem(f)] = rel
		}
	}
	var out []MockupPair
	seen := map[string]bool{}
	_ = filepath.Walk(filepath.Join(dir, "docs", "mockups"), func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".html") {
			return nil
		}
		rel, _ := filepath.Rel(dir, path)
		st := stem(path)
		if b, ok := shots[st]; ok {
			out = append(out, MockupPair{Name: st, Mockup: rel, Built: b})
			seen[st] = true
		}
		return nil
	})
	for st, b := range shots {
		if !seen[st] && len(out) < 12 {
			out = append(out, MockupPair{Name: st, Built: b, Note: "no matching mockup"})
		}
	}
	if out == nil {
		out = []MockupPair{}
	}
	return out
}

func stem(p string) string {
	b := strings.TrimSuffix(filepath.Base(p), filepath.Ext(p))
	b = strings.TrimSuffix(strings.TrimSuffix(b, "-desktop"), "-phone")
	return strings.ToLower(b)
}

// RunTests runs the project's tests and counts results (Go: `go test -json`; Playwright JSON report if present).
func RunTests(ctx context.Context, dir string) Tests {
	var t Tests
	ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()
	if exists(filepath.Join(dir, "go.mod")) {
		t.Ran, t.Command = true, "go test ./... -json -cover"
		cmd := exec.CommandContext(ctx, "go", "test", "./...", "-json", "-cover")
		cmd.Dir = dir
		out, _ := cmd.Output()
		sc := bufio.NewScanner(strings.NewReader(string(out)))
		sc.Buffer(make([]byte, 0, 1<<20), 16<<20)
		covRe := regexp.MustCompile(`coverage: ([0-9.]+)% of statements`)
		var covs []string
		for sc.Scan() {
			var ev struct{ Action, Test, Output string }
			if json.Unmarshal(sc.Bytes(), &ev) != nil {
				continue
			}
			if ev.Test != "" && ev.Action == "pass" {
				t.UnitPassed++
			}
			if ev.Test != "" && ev.Action == "fail" {
				t.UnitFailed++
				t.Output += ev.Test + " failed\n"
			}
			if m := covRe.FindStringSubmatch(ev.Output); m != nil {
				covs = append(covs, m[1])
			}
		}
		if len(covs) > 0 {
			t.Coverage = covs[len(covs)-1] + "% (last package)"
		}
	}
	for _, rp := range []string{"web/test-results/results.json", "test-results/results.json"} {
		b, err := os.ReadFile(filepath.Join(dir, rp))
		if err != nil {
			continue
		}
		var rep struct {
			Stats struct {
				Expected   int `json:"expected"`
				Unexpected int `json:"unexpected"`
			} `json:"stats"`
		}
		if json.Unmarshal(b, &rep) == nil {
			t.Ran = true
			t.E2EPassed, t.E2EFailed = rep.Stats.Expected, rep.Stats.Unexpected
			t.Command = strings.TrimSpace(t.Command + " + playwright report " + rp)
		}
	}
	return t
}

var reqLineRe = regexp.MustCompile(`^- \*\*((?:FR|NFR)-\d+[a-z]?)\*\*: (.*)`)

func requirementText(path string) map[string]string {
	out := map[string]string{}
	b, err := os.ReadFile(path)
	if err != nil {
		return out
	}
	for _, l := range strings.Split(string(b), "\n") {
		if m := reqLineRe.FindStringSubmatch(l); m != nil {
			out[m[1]] = trunc(strings.ReplaceAll(m[2], "**", ""), 140)
		}
	}
	return out
}

func exists(p string) bool { _, err := os.Stat(p); return err == nil }

func trunc(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) > n {
		return s[:n] + "…"
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
