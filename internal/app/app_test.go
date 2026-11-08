package app

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/techmuch/go-build-it/internal/access"
	"github.com/techmuch/go-build-it/internal/api"
	"github.com/techmuch/go-build-it/internal/config"
	"github.com/techmuch/go-build-it/internal/projects"
	"github.com/techmuch/go-build-it/internal/queue"
	"github.com/techmuch/go-build-it/internal/store"
	"github.com/techmuch/go-build-it/internal/team"
)

var gbiBin string

func TestMain(m *testing.M) {
	dir, _ := os.MkdirTemp("", "gbi-bin-")
	gbiBin = filepath.Join(dir, "gbi")
	cmd := exec.Command("go", "build", "-o", gbiBin, "../../cmd/gbi")
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		panic("build gbi: " + err.Error())
	}
	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}

type env struct {
	t    *testing.T
	a    *App
	repo string
	srv  *httptest.Server
	ctx  context.Context // acts as the human owner
}

func git(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

func setup(t *testing.T) *env {
	t.Helper()
	home := t.TempDir()
	cfg, err := config.LoadFrom(home)
	if err != nil {
		t.Fatal(err)
	}
	cfg.DefaultUser = "owner"
	cfg.Harnesses["fake"] = config.Harness{Enabled: true, Binary: gbiBin}
	db, err := store.Open(filepath.Join(home, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	a := Wire(cfg, db, "test", slog.New(slog.NewTextHandler(io.Discard, nil)))
	a.Reviews.RunTests = false
	mux := http.NewServeMux()
	api.Mount(mux, a.Routes())
	srv := httptest.NewServer(access.Middleware("owner", func() access.ExternalPolicy { return access.ExternalPolicy{} }, mux))
	t.Cleanup(srv.Close)
	cfg.Listen = strings.TrimPrefix(srv.URL, "http://")

	repo := filepath.Join(t.TempDir(), "demo")
	os.MkdirAll(filepath.Join(repo, "docs"), 0o755)
	git(t, repo, "init", "-q", "-b", "main")
	os.WriteFile(filepath.Join(repo, "README.md"), []byte("# demo\n"), 0o644)
	os.WriteFile(filepath.Join(repo, "docs", "requirements.md"), []byte("# Req\n\n- **FR-01**: The demo SHALL work.\n"), 0o644)
	git(t, repo, "add", "-A")
	git(t, repo, "-c", "user.name=t", "-c", "user.email=t@t", "commit", "-q", "-m", "init")

	ctx := access.WithActor(context.Background(), access.Actor{Name: "owner", Kind: "human", Origin: access.Local})
	if _, err := a.Projects.Import(ctx, repo, "demo", "A demo project", "Implementation"); err != nil {
		t.Fatal(err)
	}
	for _, m := range []team.Member{
		{ID: "forge", Name: "Forge", Kind: "ai", Role: "developer", Harness: "fake", Instructions: "Implement tasks."},
		{ID: "lens", Name: "Lens", Kind: "ai", Role: "reviewer", Harness: "fake", Instructions: "Review."},
		{ID: "atlas", Name: "Atlas", Kind: "ai", Role: "architect", Harness: "fake"},
	} {
		if _, err := a.Team.Save(ctx, m); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := a.Standards.Save(ctx, "git-workflow", "Git workflow", "Branch per task, PR, Lens review, human approves, squash-merge, tag YYYY.MM.DD.N, publish Pages.", "test"); err != nil {
		t.Fatal(err)
	}
	return &env{t: t, a: a, repo: repo, srv: srv, ctx: ctx}
}

func (e *env) waitStatus(id, want string) queue.Task {
	e.t.Helper()
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		tk, _ := e.a.Queue.Get(context.Background(), id)
		if tk.Status == want {
			return tk
		}
		time.Sleep(100 * time.Millisecond)
	}
	tk, _ := e.a.Queue.Get(context.Background(), id)
	ss, _ := e.a.Sessions.List(context.Background(), "", id, false)
	e.t.Fatalf("task %s status %q, want %q; sessions %+v", id, tk.Status, want, ss)
	return tk
}

func TestFullWorkflow(t *testing.T) {
	e := setup(t)
	ctx := e.ctx
	tk, err := e.a.Queue.Add(ctx, queue.Task{ProjectID: "demo", Title: "Add work log", Reqs: []string{"FR-01"}})
	if err != nil {
		t.Fatal(err)
	}
	prompt := e.a.Sessions.Prepare(ctx, tk, mustProject(t, e.a, "demo"))
	if !strings.Contains(prompt, "FR-01: The demo SHALL work.") || !strings.Contains(prompt, "A demo project") {
		t.Fatalf("context preparation missing requirement or purpose:\n%s", prompt)
	}
	s, err := e.a.Sessions.Start(ctx, tk.ID, "")
	if err != nil {
		t.Fatal(err)
	}
	if s.MemberID != "forge" {
		t.Fatalf("picked %s, want forge", s.MemberID)
	}
	tk = e.waitStatus(tk.ID, queue.Review)
	if !strings.HasPrefix(tk.Branch, "t-1-") {
		t.Fatalf("branch %q", tk.Branch)
	}
	// Instruction files were generated in the worktree but not committed (FR-53).
	wt := e.a.Sessions.WorktreePath("demo", tk.ID)
	stds, _ := e.a.Standards.Effective(ctx, "demo")
	if !strings.Contains(standardsRender("demo", stds), "Git workflow") {
		t.Fatal("instruction file content missing the git workflow standard")
	}
	if _, err := os.Stat(filepath.Join(wt, "AGENTS.md")); err == nil {
		t.Fatal("generated AGENTS.md should be cleaned before commit")
	}
	if strings.Contains(git(t, wt, "show", "--stat", "HEAD"), "AGENTS.md") {
		t.Fatal("instruction files must not be committed")
	}
	sess, _ := e.a.Sessions.Get(ctx, s.ID)
	if sess.Status != "done" || sess.Used == 0 || !sess.Measured {
		t.Fatalf("session %+v", sess)
	}
	out, _ := e.a.Sessions.Output(ctx, s.ID, 0, "terminal")
	if len(out) == 0 {
		t.Fatal("no terminal output recorded (FR-66)")
	}
	rs, _ := e.a.Reviews.List(ctx, "demo", "pending")
	if len(rs) != 1 {
		t.Fatalf("reviews: %+v", rs)
	}
	if len(rs[0].Assessment.Requirements) != 1 || rs[0].Assessment.Summary == "" {
		t.Fatalf("assessment: %+v", rs[0].Assessment)
	}
	// Verdict by a persona is refused.
	if _, err := e.a.Reviews.Verdict(access.Persona(ctx, "Lens"), rs[0].ID, "approve", ""); err == nil {
		t.Fatal("persona verdict should fail")
	}
	if _, err := e.a.Reviews.Verdict(ctx, rs[0].ID, "approve", "looks good"); err != nil {
		t.Fatal(err)
	}
	e.waitStatus(tk.ID, queue.Done)
	if b, err := os.ReadFile(filepath.Join(e.repo, "FAKE_WORK.md")); err != nil || !strings.Contains(string(b), "Add work log") {
		t.Fatalf("merged work missing: %v", err)
	}
	if _, err := os.Stat(wt); err == nil {
		t.Fatal("worktree should be removed after merge")
	}
	// Release: CalVer tag (FR-05).
	rel, err := e.a.Courier.Release(ctx, "demo", false)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(rel.Tag, time.Now().Format("2006.01.02")+".") || !strings.Contains(rel.Notes, "T-1") {
		t.Fatalf("release %+v", rel)
	}
	// History has the whole story (NFR-10).
	evs, _ := e.a.Hist.List(ctx, historyQuery("demo"))
	types := map[string]bool{}
	for _, ev := range evs {
		types[ev.Type] = true
	}
	for _, want := range []string{"task.created", "session.started", "git.commit", "review.opened", "review.approved", "git.merge", "release.created"} {
		if !types[want] {
			t.Errorf("history missing %s", want)
		}
	}
}

func TestBlockingQuestionAndAnswer(t *testing.T) {
	e := setup(t)
	tk, _ := e.a.Queue.Add(e.ctx, queue.Task{ProjectID: "demo", Title: "Decide something", Description: "FAKE:ask"})
	if _, err := e.a.Sessions.Start(e.ctx, tk.ID, "forge"); err != nil {
		t.Fatal(err)
	}
	e.waitStatus(tk.ID, queue.Blocked)
	qs, _ := e.a.Questions.List(e.ctx, "demo", "open")
	if len(qs) != 1 || !qs[0].Blocking || qs[0].Asker != "forge" || qs[0].TaskID != tk.ID {
		t.Fatalf("questions %+v", qs)
	}
	inbox := e.a.inboxItems(e.ctx, "")
	if len(inbox) == 0 || !inbox[0].Blocking {
		t.Fatalf("inbox %+v", inbox)
	}
	if _, err := e.a.Questions.Answer(e.ctx, qs[0].ID, "Go with A", "A", "Prefer simple options"); err != nil {
		t.Fatal(err)
	}
	got := e.waitStatus(tk.ID, queue.Queued)
	if !strings.Contains(got.ContextNote, "chose A") {
		t.Fatalf("answer not routed into context: %q", got.ContextNote)
	}
	vals, _ := e.a.Projects.Values(e.ctx, "demo", false)
	if len(vals) != 1 || vals[0].Status != "proposed" {
		t.Fatalf("value proposal: %+v", vals)
	}
}

func TestRateLimitPauseAndResume(t *testing.T) {
	e := setup(t)
	tk, _ := e.a.Queue.Add(e.ctx, queue.Task{ProjectID: "demo", Title: "Hit a limit", Description: "FAKE:limit"})
	s, err := e.a.Sessions.Start(e.ctx, tk.ID, "forge")
	if err != nil {
		t.Fatal(err)
	}
	e.waitStatus(tk.ID, queue.Paused)
	sess, _ := e.a.Sessions.Get(e.ctx, s.ID)
	if sess.Status != "rate_limited" || sess.ResumeAt == "" {
		t.Fatalf("session %+v", sess)
	}
	time.Sleep(3 * time.Second)
	// Make sure the dispatcher does not immediately rerun it (the description still says FAKE:limit).
	e.a.Cfg.MaxConcurrent = 0
	e.a.Sessions.Tick(e.ctx)
	got, _ := e.a.Queue.Get(e.ctx, tk.ID)
	if got.Status != queue.Queued && got.Status != queue.Running {
		evs, _ := e.a.Hist.List(e.ctx, historyQuery("demo"))
		for _, ev := range evs {
			t.Log(ev.Type, ev.Summary, ev.Why)
		}
		sess, _ := e.a.Sessions.Get(e.ctx, s.ID)
		t.Logf("session %+v now %s", sess, time.Now().UTC().Format(time.RFC3339))
		t.Fatalf("not resumed: %s", got.Status)
	}
}

func TestCompactionLogged(t *testing.T) {
	e := setup(t)
	tk, _ := e.a.Queue.Add(e.ctx, queue.Task{ProjectID: "demo", Title: "Long change", Description: "FAKE:compact"})
	if _, err := e.a.Sessions.Start(e.ctx, tk.ID, "forge"); err != nil {
		t.Fatal(err)
	}
	got := e.waitStatus(tk.ID, queue.Review)
	if got.Compactions < 1 {
		t.Fatalf("compactions = %d", got.Compactions)
	}
	evs, _ := e.a.Hist.List(e.ctx, historyQueryType("session.compacted"))
	if len(evs) == 0 {
		t.Fatal("compaction not in history (FR-72)")
	}
	cands, _ := e.a.Queue.SplitCandidates(e.ctx, "demo", 1)
	if len(cands) != 1 {
		t.Fatalf("split candidates %+v", cands)
	}
}

func TestGateBlocksLaterMilestone(t *testing.T) {
	e := setup(t)
	m1, _ := e.a.Projects.SaveMilestone(e.ctx, projects.Milestone{ProjectID: "demo", Name: "M1", Ord: 1, GateName: "Design review", GateKind: "tabletop"})
	m2, _ := e.a.Projects.SaveMilestone(e.ctx, projects.Milestone{ProjectID: "demo", Name: "M2", Ord: 2})
	tk, _ := e.a.Queue.Add(e.ctx, queue.Task{ProjectID: "demo", Title: "Later work", MilestoneID: m2.ID})
	next, _ := e.a.Queue.Next(e.ctx, "demo")
	if next != nil {
		t.Fatalf("task should wait for the gate, got %s", next.ID)
	}
	ts, _ := e.a.Queue.List(e.ctx, queue.Filter{ProjectID: "demo"})
	if !strings.Contains(ts[0].BlockedBy, "Design review") {
		t.Fatalf("blockedBy %q", ts[0].BlockedBy)
	}
	g, err := e.a.Rehearsals.Create(e.ctx, rehearsalOf("demo", m1.ID))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := e.a.Rehearsals.Decide(e.ctx, g.ID, "approve", ""); err == nil {
		t.Fatal("approving with unreviewed steps should fail")
	}
	maria := access.WithActor(context.Background(), access.Actor{Name: "maria", Kind: "human", Origin: access.Tailscale})
	if _, err := e.a.Rehearsals.AddFinding(maria, g.ID, 0, "Label is unclear"); err != nil {
		t.Fatal(err)
	}
	if _, err := e.a.Rehearsals.Decide(maria, g.ID, "approve", ""); err == nil {
		t.Fatal("only the owner may approve (ADR-006)")
	}
	if _, err := e.a.Rehearsals.Decide(e.ctx, g.ID, "approve", "ok"); err != nil {
		t.Fatal(err)
	}
	next, _ = e.a.Queue.Next(e.ctx, "demo")
	if next == nil || next.ID != tk.ID {
		t.Fatalf("gate approved but task not ready: %+v", next)
	}
}

func TestArtifactEditAndChat(t *testing.T) {
	e := setup(t)
	p := mustProject(t, e.a, "demo")
	reply, err := e.a.Agent.Edit(e.ctx, "atlas", "project artifact", "docs/requirements.md", "# Req\n\n- **FR-01**: The demo SHALL work.", "Add FR-02 for offline mode", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(reply.Hunks) != 1 || !strings.Contains(strings.Join(reply.Hunks[0].Added, "\n"), "offline") {
		t.Fatalf("hunks %+v", reply.Hunks)
	}
	hash, err := e.a.Courier.CommitArtifact(e.ctx, p, "docs/requirements.md", reply.Document, "docs: add FR-02", "owner")
	if err != nil || hash == "" {
		t.Fatalf("commit artifact: %v", err)
	}
	if revs := e.a.Git.FileHistory(e.ctx, p.Path, "docs/requirements.md", 10); len(revs) != 2 {
		t.Fatalf("revisions %+v", revs)
	}
}

func TestRoutesParity(t *testing.T) {
	e := setup(t)
	names, clis := map[string]bool{}, map[string]bool{}
	for _, r := range e.a.Routes() {
		if r.CLI == "" || r.Name == "" || r.Summary == "" || r.Handler == nil {
			t.Errorf("route %s %s is missing CLI, name, summary or handler (FR-30)", r.Method, r.Path)
		}
		if names[r.Name] || clis[r.CLI] {
			t.Errorf("duplicate route %s / %q", r.Name, r.CLI)
		}
		names[r.Name], clis[r.CLI] = true, true
	}
	// External requests are refused (ADR-005).
	req := httptest.NewRequest("GET", "/api/system", nil)
	req.RemoteAddr = "192.168.1.20:5000"
	rec := httptest.NewRecorder()
	access.Middleware("owner", func() access.ExternalPolicy { return access.ExternalPolicy{} }, http.NotFoundHandler()).ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("external request got %d", rec.Code)
	}
	// With external access allowed, a token is required (P9-07).
	ext := func() access.ExternalPolicy { return access.ExternalPolicy{Allow: true, Token: "s3cret"} }
	ok := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) })
	for _, c := range []struct {
		auth string
		want int
	}{{"", 401}, {"Bearer wrong", 401}, {"Bearer s3cret", 204}} {
		req := httptest.NewRequest("GET", "/api/system", nil)
		req.RemoteAddr = "192.168.1.20:5000"
		if c.auth != "" {
			req.Header.Set("Authorization", c.auth)
		}
		rec := httptest.NewRecorder()
		access.Middleware("owner", ext, ok).ServeHTTP(rec, req)
		if rec.Code != c.want {
			t.Errorf("auth %q: got %d want %d", c.auth, rec.Code, c.want)
		}
	}
	// Allowed but no token configured: still refused.
	rec = httptest.NewRecorder()
	access.Middleware("owner", func() access.ExternalPolicy { return access.ExternalPolicy{Allow: true} }, ok).ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("allowed without a token got %d", rec.Code)
	}
}

func mustProject(t *testing.T, a *App, id string) projects.Project {
	p, err := a.Projects.Get(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestMergeConflictGoesBackToDeveloper(t *testing.T) {
	e := setup(t)
	t1, _ := e.a.Queue.Add(e.ctx, queue.Task{ProjectID: "demo", Title: "First change"})
	t2, _ := e.a.Queue.Add(e.ctx, queue.Task{ProjectID: "demo", Title: "Second change"})
	for _, tk := range []queue.Task{t1, t2} {
		if _, err := e.a.Sessions.Start(e.ctx, tk.ID, "forge"); err != nil {
			t.Fatal(err)
		}
		e.waitStatus(tk.ID, queue.Review)
	}
	if err := e.a.Courier.Merge(e.ctx, t1.ID); err != nil {
		t.Fatal(err)
	}
	// Both branches created FAKE_WORK.md from the same base: the second merge conflicts.
	err := e.a.Courier.Merge(e.ctx, t2.ID)
	if err == nil || !strings.Contains(err.Error(), "requeued") {
		t.Fatalf("expected a conflict that requeues the task, got %v", err)
	}
	got, _ := e.a.Queue.Get(e.ctx, t2.ID)
	if got.Status != queue.Queued || !strings.Contains(got.ContextNote, "conflicts in FAKE_WORK.md") {
		t.Fatalf("task %s %q", got.Status, got.ContextNote)
	}
	if st := git(t, e.repo, "status", "--porcelain"); st != "" {
		t.Fatalf("main checkout left dirty after aborted merge:\n%s", st)
	}
}

func TestCourierPushesToRemote(t *testing.T) {
	e := setup(t)
	bare := filepath.Join(t.TempDir(), "origin.git")
	git(t, filepath.Dir(bare), "init", "-q", "--bare", bare)
	git(t, e.repo, "remote", "add", "origin", bare)
	git(t, e.repo, "push", "-q", "-u", "origin", "main")
	// Re-import so the project records its remote.
	if err := e.a.Projects.Remove(e.ctx, "demo"); err != nil {
		t.Fatal(err)
	}
	if _, err := e.a.Projects.Import(e.ctx, e.repo, "demo", "A demo project", "Implementation"); err != nil {
		t.Fatal(err)
	}
	tk, _ := e.a.Queue.Add(e.ctx, queue.Task{ProjectID: "demo", Title: "Pushed work"})
	if _, err := e.a.Sessions.Start(e.ctx, tk.ID, "forge"); err != nil {
		t.Fatal(err)
	}
	tk = e.waitStatus(tk.ID, queue.Review)
	if out := git(t, bare, "branch", "--list", tk.Branch); !strings.Contains(out, tk.Branch) {
		t.Fatalf("branch %s not on the remote (FR-94)", tk.Branch)
	}
	p := mustProject(t, e.a, "demo")
	if u := e.a.Git.FindUnpushed(e.ctx, p.Path); u.Commits != 0 {
		t.Fatalf("unpushed commits remain: %+v", u)
	}
}

func TestSmartHandoffExperimental(t *testing.T) {
	e := setup(t)
	e.a.Cfg.Experimental.SmartHandoffs = true
	e.a.Cfg.Experimental.SmartHandoffProjects = []string{"demo"}
	e.a.Cfg.Experimental.HandoffThreshold = 0.75
	tk, _ := e.a.Queue.Add(e.ctx, queue.Task{ProjectID: "demo", Title: "Very long change", Description: "FAKE:handoff"})
	s, err := e.a.Sessions.Start(e.ctx, tk.ID, "forge")
	if err != nil {
		t.Fatal(err)
	}
	got := e.waitStatus(tk.ID, queue.Queued)
	if got.Handoffs != 1 || !strings.Contains(got.ContextNote, "Specific next steps: finish the second half.") {
		t.Fatalf("handoffs=%d note=%q", got.Handoffs, got.ContextNote)
	}
	sess, _ := e.a.Sessions.Get(e.ctx, s.ID)
	if sess.ExitReason != "handed off" || !strings.Contains(sess.Handoff, "Objective") {
		t.Fatalf("session %+v", sess)
	}
	// The next session starts from the briefing (FR-74) and finishes normally.
	if _, err := e.a.Sessions.Start(e.ctx, tk.ID, "forge"); err != nil {
		t.Fatal(err)
	}
	e.waitStatus(tk.ID, queue.Review)
	evs, _ := e.a.Hist.List(e.ctx, historyQueryType("handoff."))
	if len(evs) < 2 { // requested + written
		t.Fatalf("handoff events: %d", len(evs))
	}
	// Disabled for other projects: compaction stays the default.
	if e.a.Cfg.HandoffsEnabledFor("other") {
		t.Fatal("handoffs should be limited to listed projects")
	}
}

func TestAttachableSessionWithTmux(t *testing.T) {
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skip("tmux not installed")
	}
	e := setup(t)
	e.a.Cfg.Experimental.AttachableSessions = true
	tk, _ := e.a.Queue.Add(e.ctx, queue.Task{ProjectID: "demo", Title: "Headed work"})
	s, err := e.a.Sessions.Start(e.ctx, tk.ID, "forge")
	if err != nil {
		t.Fatal(err)
	}
	got, _ := e.a.Sessions.Get(e.ctx, s.ID)
	if !got.Live || !strings.Contains(got.Attach, "tmux attach -t gbi-") {
		t.Fatalf("session %+v", got)
	}
	e.waitStatus(tk.ID, queue.Review)
	out, _ := e.a.Sessions.Output(e.ctx, s.ID, 0, "terminal")
	joined := ""
	for _, l := range out {
		joined += l.Text + "\n"
	}
	if !strings.Contains(joined, "fake agent ready") {
		t.Fatalf("pane not mirrored:\n%s", joined)
	}
	wt := e.a.Sessions.WorktreePath("demo", tk.ID)
	if !strings.Contains(git(t, wt, "log", "-1", "--stat"), "FAKE_WORK.md") {
		t.Fatal("work from the attachable session was not committed")
	}
}

func TestReviewAssignmentAndOwners(t *testing.T) {
	e := setup(t)
	if _, err := e.a.Team.SaveTeam(e.ctx, "core", "Core"); err != nil {
		t.Fatal(err)
	}
	for _, m := range []team.Member{{ID: "maria", Name: "maria", Kind: "human", Role: "reviewer", TeamID: "core", Login: "maria@github"}, {ID: "sam", Name: "sam", Kind: "human", Role: "reviewer", TeamID: "core"}} {
		if _, err := e.a.Team.Save(e.ctx, m); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := e.a.Projects.Update(e.ctx, "demo", "", "", "", "", "core"); err != nil {
		t.Fatal(err)
	}
	a1, _ := e.a.Queue.Add(e.ctx, queue.Task{ProjectID: "demo", Title: "one"})
	a2, _ := e.a.Queue.Add(e.ctx, queue.Task{ProjectID: "demo", Title: "two"})
	r1, _ := e.a.Reviews.Open(e.ctx, a1.ID)
	r2, _ := e.a.Reviews.Open(e.ctx, a2.ID)
	if r1.Assignee == "" || r2.Assignee == "" || r1.Assignee == r2.Assignee {
		t.Fatalf("round-robin assignment: %q %q", r1.Assignee, r2.Assignee)
	}
	if !e.a.Cfg.IsOwner("owner", "") || e.a.Cfg.IsOwner("maria", "maria@github") {
		t.Fatal("default owner check")
	}
	e.a.Cfg.Owners = []string{"owner", "maria@github"}
	if !e.a.Cfg.IsOwner("maria", "maria@github") {
		t.Fatal("second owner by Tailscale login")
	}
}
