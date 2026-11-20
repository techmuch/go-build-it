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
	"strconv"
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
	Mockups      []MockupPair `json:"mockups"`
	Tests        Tests        `json:"tests"`
	Standards    []Check      `json:"standards"`
	Feedback     []Check      `json:"feedback"`
	Requirements []Check      `json:"requirements"`
	DiffStat     string       `json:"diffStat"`
	// Gaps lists the evidence that is missing or against the change. Approving with gaps
	// needs a stated reason (FR-112).
	Gaps []string `json:"gaps"`
	// Base is the commit the change is compared with: where its branch left the main line.
	Base            string `json:"base,omitempty"`
	diff, dir, task string
	Summary         string `json:"summary"`
	Recommendation  string `json:"recommendation"`
	GeneratedAt     string `json:"generatedAt"`
	PRURL           string `json:"prUrl"`
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
	Merge func(ctx context.Context, taskID string) error
	// Lens asks the project's reviewer member to assess a change.
	Lens func(ctx context.Context, projectID, prompt string) (string, error)
	// Resume starts a task again after changes were requested.
	Resume   func(ctx context.Context, taskID string)
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
		if len(r.Assessment.Gaps) > 0 && strings.TrimSpace(comment) == "" {
			return r, errors.New("the evidence has gaps (" + strings.Join(r.Assessment.Gaps, "; ") + "): say why you approve anyway")
		}
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
		// The person asked for changes: start the follow-up now, without a second click.
		if _, err := s.queue.SetStatus(ctx, t.ID, queue.Queued, "changes requested in "+id); err == nil && s.Hooks.Resume != nil {
			go s.Hooks.Resume(context.WithoutCancel(ctx), t.ID)
		}
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
		// The whole change: everything on the task's branch since it left the main line, not
		// only its last commit.
		a.Base = "HEAD~1"
		for _, main := range []string{"main", "master"} {
			if out, err := exec.CommandContext(ctx, "git", "-C", dir, "merge-base", "HEAD", main).Output(); err == nil && strings.TrimSpace(string(out)) != "" {
				a.Base = strings.TrimSpace(string(out))
				break
			}
		}
		if out, err := exec.CommandContext(ctx, "git", "-C", dir, "diff", "--stat", a.Base+"..HEAD").Output(); err == nil {
			a.DiffStat = strings.TrimSpace(string(out))
		}
		if out, err := exec.CommandContext(ctx, "git", "-C", dir, "diff", a.Base+"..HEAD").Output(); err == nil {
			a.diff = string(out)
		}
	}
	a.dir, a.task = dir, strings.TrimSpace(t.Description)
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
	a.Gaps = a.gaps()
	return a
}

// gaps names what an approver would be approving without.
func (a Assessment) gaps() []string {
	out := []string{}
	count := func(xs []Check, statuses ...string) int {
		n := 0
		for _, c := range xs {
			for _, st := range statuses {
				if c.Status == st {
					n++
				}
			}
		}
		return n
	}
	switch {
	case !a.Tests.Ran:
		out = append(out, "no tests ran")
	case a.Tests.UnitFailed+a.Tests.E2EFailed > 0:
		out = append(out, fmt.Sprintf("%d tests failed", a.Tests.UnitFailed+a.Tests.E2EFailed))
	}
	if n := count(a.Requirements, "not", "partly"); n > 0 {
		out = append(out, fmt.Sprintf("%d requirements not fully met", n))
	}
	if n := count(a.Requirements, "unassessed"); n > 0 {
		out = append(out, fmt.Sprintf("%d requirements not assessed", n))
	}
	if n := count(a.Standards, "not", "partly"); n > 0 {
		out = append(out, fmt.Sprintf("%d standards not fully met", n))
	}
	if n := count(a.Feedback, "not", "partly", "unassessed"); n > 0 {
		out = append(out, fmt.Sprintf("%d feedback items not addressed", n))
	}
	for _, m := range a.Mockups {
		if m.Mockup != "" && m.Built == "" {
			out = append(out, "a mockup has no screenshot of the built screen")
			break
		}
	}
	return out
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
	tests := "not run: go-build-it found no test command for this project"
	if a.Tests.Ran {
		tests = fmt.Sprintf("%s: %d passed, %d failed; end-to-end %d passed, %d failed. %s", a.Tests.Command, a.Tests.UnitPassed, a.Tests.UnitFailed, a.Tests.E2EPassed, a.Tests.E2EFailed, trunc(a.Tests.Output, 1500))
	}
	prompt := fmt.Sprintf(`Assess task %s "%s" for review.

What the task asked for:
%s

The change is in the folder %s (a git worktree, compared with commit %s). You may read any file there and run read-only commands in it (git diff, git log, the project's tests). Do not change anything in it.

Files changed:
%s

The change:
%s

Tests go-build-it ran: %s

Items to assess (JSON): %s

Judge each item from what you read, and quote the file or line that shows it. Use "na" for an item that does not apply to this change (for example a backend standard on a documents-only change); never mark such an item "met".

Answer ONLY with JSON: {"summary": "two sentences", "recommendation": "approve|changes", "checks": [{"group":"…","index":0,"status":"met|partly|not|na","evidence":"…"}]}`,
		t.ID, t.Title, trunc(firstLine(a.task, "(no description)"), 4000), a.dir, trunc(a.Base, 12), trunc(a.DiffStat, 3000), trunc(a.diff, 60000), tests, ib)
	out, err := s.Hooks.Lens(ctx, t.ProjectID, prompt)
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
			switch c.Status {
			case "met", "partly", "not", "na":
				group[c.Index].Status, group[c.Index].Evidence = c.Status, c.Evidence
			}
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
	if !t.Ran {
		npmTests(ctx, dir, &t)
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

// firstLine returns s, or alt when s is empty.
func firstLine(s, alt string) string {
	if strings.TrimSpace(s) == "" {
		return alt
	}
	return s
}

var (
	npmPassedRe = regexp.MustCompile(`(?m)^\s*Tests\s+.*?(\d+) passed`)
	npmFailedRe = regexp.MustCompile(`(?m)^\s*Tests\s+.*?(\d+) failed`)
	nodePassRe  = regexp.MustCompile(`(?m)^\s*(?:ℹ|#)\s*pass\s+(\d+)`)
	nodeFailRe  = regexp.MustCompile(`(?m)^\s*(?:ℹ|#)\s*fail\s+(\d+)`)
	ansiRe      = regexp.MustCompile(`\x1b\[[0-9;]*m`)
)

// npmTests runs "npm test" for a project whose package.json has a test script (Vitest and
// Jest print "Tests  3 failed | 12 passed"). Without counts, the exit code decides.
func npmTests(ctx context.Context, dir string, t *Tests) {
	b, err := os.ReadFile(filepath.Join(dir, "package.json"))
	if err != nil {
		return
	}
	var pkg struct {
		Scripts map[string]string `json:"scripts"`
	}
	if json.Unmarshal(b, &pkg) != nil || strings.TrimSpace(pkg.Scripts["test"]) == "" {
		return
	}
	run := func(args ...string) (string, error) {
		cmd := exec.CommandContext(ctx, "npm", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "CI=1", "NO_COLOR=1")
		out, err := cmd.CombinedOutput()
		return ansiRe.ReplaceAllString(string(out), ""), err
	}
	if !exists(filepath.Join(dir, "node_modules")) {
		if out, err := run("install", "--no-audit", "--no-fund"); err != nil {
			t.Ran, t.Command, t.UnitFailed, t.Output = true, "npm install", 1, "npm install failed:\n"+tail(out, 1500)
			return
		}
	}
	out, err := run("test")
	t.Ran, t.Command = true, "npm test"
	if m := npmPassedRe.FindStringSubmatch(out); m != nil {
		t.UnitPassed, _ = strconv.Atoi(m[1])
	}
	if m := npmFailedRe.FindStringSubmatch(out); m != nil {
		t.UnitFailed, _ = strconv.Atoi(m[1])
	}
	// Node's own test runner prints "ℹ pass 24" and "ℹ fail 0" (or "# pass 24" as TAP).
	if t.UnitPassed+t.UnitFailed == 0 {
		if m := nodePassRe.FindStringSubmatch(out); m != nil {
			t.UnitPassed, _ = strconv.Atoi(m[1])
		}
		if m := nodeFailRe.FindStringSubmatch(out); m != nil {
			t.UnitFailed, _ = strconv.Atoi(m[1])
		}
	}
	if err != nil {
		if t.UnitFailed == 0 {
			t.UnitFailed = 1
		}
		t.Output = tail(out, 2000)
	}
}

// tail returns the last n bytes of s.
func tail(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return "…" + s[len(s)-n:]
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
