package app

import (
	"context"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/techmuch/go-build-it/internal/herdr"
	"github.com/techmuch/go-build-it/internal/queue"
	"github.com/techmuch/go-build-it/internal/sessions"
)

// waitSession polls until a session reaches a status.
func (e *env) waitSession(id, want string) sessions.Session {
	e.t.Helper()
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		if s, err := e.a.Sessions.Get(context.Background(), id); err == nil && s.Status == want {
			return s
		}
		time.Sleep(100 * time.Millisecond)
	}
	s, _ := e.a.Sessions.Get(context.Background(), id)
	e.t.Fatalf("session %s status %q, want %q (%s)", id, s.Status, want, s.ExitReason)
	return s
}

// P13-07: a member keeps one conversation across tasks and rotates past the threshold (Q8).
func TestMemberConversationContinuityAndRotation(t *testing.T) {
	e := setup(t)
	t1, _ := e.a.Queue.Add(e.ctx, queue.Task{ProjectID: "demo", Title: "First task"})
	s1, err := e.a.Sessions.Start(e.ctx, t1.ID, "forge@demo")
	if err != nil {
		t.Fatal(err)
	}
	e.waitStatus(t1.ID, queue.Review)
	m1, _ := e.a.Team.Member(e.ctx, "forge@demo")
	if m1.ConversationID == "" || m1.ConversationTasks != 1 || m1.PaneID == "" {
		t.Fatalf("after task 1: %+v", m1)
	}
	// Labels in herdr: name and role, idle after the task (FR-67). The idle label is set just
	// after the task reaches review, so wait for it (it raced on slow CI runners).
	labelled := func() (herdr.PaneInfo, error, bool) {
		p, err := e.a.Herdr.C.Pane(e.ctx, m1.PaneID)
		return p, err, err == nil && p.DisplayAgent != nil && *p.DisplayAgent == "Forge (developer)" && p.Title != nil && *p.Title == "idle"
	}
	for deadline := time.Now().Add(10 * time.Second); time.Now().Before(deadline); time.Sleep(100 * time.Millisecond) {
		if _, _, ok := labelled(); ok {
			break
		}
	}
	if p, err, ok := labelled(); !ok {
		t.Fatalf("pane labels: %+v %v", p, err)
	}
	t2, _ := e.a.Queue.Add(e.ctx, queue.Task{ProjectID: "demo", Title: "Second task, big", Description: "FAKE:big"})
	s2, err := e.a.Sessions.Start(e.ctx, t2.ID, "forge@demo")
	if err != nil {
		t.Fatal(err)
	}
	e.waitStatus(t2.ID, queue.Review)
	ss2, _ := e.a.Sessions.Get(e.ctx, s2.ID)
	if ss2.HarnessSession != m1.ConversationID || ss2.PaneID != m1.PaneID {
		t.Fatalf("task 2 should continue the same conversation and pane: %s/%s vs %s/%s", ss2.HarnessSession, ss2.PaneID, m1.ConversationID, m1.PaneID)
	}
	if ss2.Fraction < 0.75 {
		t.Fatalf("FAKE:big should pass the threshold: %.2f", ss2.Fraction)
	}
	m2, _ := e.a.Team.Member(e.ctx, "forge@demo")
	if m2.ConversationID != "" {
		t.Fatalf("conversation should rotate past 75%%: %+v", m2)
	}
	evs, _ := e.a.Hist.List(e.ctx, historyQueryType("member.rotated"))
	if len(evs) != 1 {
		t.Fatalf("rotation events %d", len(evs))
	}
	// The next task starts a new conversation that carries the member's notes (FR-76).
	t3, _ := e.a.Queue.Add(e.ctx, queue.Task{ProjectID: "demo", Title: "Third task"})
	s3, err := e.a.Sessions.Start(e.ctx, t3.ID, "forge@demo")
	if err != nil {
		t.Fatal(err)
	}
	e.waitStatus(t3.ID, queue.Review)
	ss3, _ := e.a.Sessions.Get(e.ctx, s3.ID)
	if ss3.HarnessSession == "" || ss3.HarnessSession == m1.ConversationID {
		t.Fatalf("task 3 should start a new conversation: %q", ss3.HarnessSession)
	}
	out, _ := e.a.Sessions.Output(e.ctx, s3.ID, 0, "")
	joined := ""
	for _, l := range out {
		joined += l.Text + "\n"
	}
	_ = s1
	if !strings.Contains(joined, "Read .gbi-task.md") {
		t.Fatalf("structured output missing the prompt:\n%s", joined)
	}
}

// P13-05 / P13-10: a blocked member becomes a needs-you item; answering in the terminal resumes it.
func TestBlockedMemberNeedsYou(t *testing.T) {
	e := setup(t)
	tk, _ := e.a.Queue.Add(e.ctx, queue.Task{ProjectID: "demo", Title: "Guarded change", Description: "FAKE:block"})
	s, err := e.a.Sessions.Start(e.ctx, tk.ID, "forge@demo")
	if err != nil {
		t.Fatal(err)
	}
	blocked := e.waitSession(s.ID, "blocked")
	inbox := e.a.inboxItems(e.ctx, "demo")
	if len(inbox) == 0 || inbox[0].Kind != "member" || !inbox[0].Blocking || inbox[0].ID != s.ID {
		t.Fatalf("inbox %+v", inbox)
	}
	card := e.a.card(e.ctx, mustProject(t, e.a, "demo"), false)
	found := false
	for _, m := range card.Team {
		found = found || (m.ID == "forge@demo" && m.State == "blocked")
	}
	if !found {
		t.Fatalf("card team state %+v", card.Team)
	}
	// The human answers in the terminal (here: typed into the pane).
	if err := e.a.Herdr.C.Call(e.ctx, herdr.MethodPaneSendText, map[string]any{"pane_id": blocked.PaneID, "text": "y\n"}, nil); err != nil {
		t.Fatal(err)
	}
	e.waitStatus(tk.ID, queue.Review)
	evs, _ := e.a.Hist.List(e.ctx, historyQueryType("session.blocked"))
	if len(evs) == 0 {
		t.Fatal("blocked state not in history")
	}
}

// FR-07f: a member whose harness starts at a folder trust screen in its worktree is
// answered by go-build-it and runs the task; the answer is in the history.
func TestMemberTrustScreenAnswered(t *testing.T) {
	e := setup(t)
	hc := e.a.Cfg.Harnesses["fake"]
	hc.Flags = []string{"--start-trust"}
	e.a.Cfg.Harnesses["fake"] = hc
	tk, _ := e.a.Queue.Add(e.ctx, queue.Task{ProjectID: "demo", Title: "First run in a new worktree"})
	if _, err := e.a.Sessions.Start(e.ctx, tk.ID, "forge@demo"); err != nil {
		t.Fatal(err)
	}
	e.waitStatus(tk.ID, queue.Review)
	evs, _ := e.a.Hist.List(e.ctx, historyQueryType("agent.trusted"))
	if len(evs) != 1 || evs[0].Subject != "Forge" || evs[0].ProjectID != "demo" {
		t.Fatalf("agent.trusted events: %+v", evs)
	}
	if blocked, _ := e.a.Hist.List(e.ctx, historyQueryType("session.blocked")); len(blocked) != 0 {
		t.Fatalf("the trust screen should not make the member blocked: %+v", blocked)
	}
}

// P13-08: a member keeps working while go-build-it restarts; the new process reattaches.
func TestReattachAfterRestart(t *testing.T) {
	e := setup(t)
	tk, _ := e.a.Queue.Add(e.ctx, queue.Task{ProjectID: "demo", Title: "Survives a restart", Description: "FAKE:block"})
	s, err := e.a.Sessions.Start(e.ctx, tk.ID, "forge@demo")
	if err != nil {
		t.Fatal(err)
	}
	blocked := e.waitSession(s.ID, "blocked")
	// "Restart": a second app on the same database and herdr session; the old one stops following.
	b := Wire(e.a.Cfg, e.a.DB, "test", slog.New(slog.NewTextHandler(io.Discard, nil)))
	b.Reviews.RunTests = false
	b.Sessions.Poll = 100 * time.Millisecond
	if _, err := b.Herdr.Ensure(e.ctx); err != nil || !b.Herdr.Status().Adopted {
		t.Fatalf("second app should adopt the running herdr session: %+v %v", b.Herdr.Status(), err)
	}
	e.a.Sessions.Cancel(context.Background(), s.ID) //nolint: the old process "exits"; its follower stops
	// Cancel marked the task queued; put the state back as if the old process had simply died.
	_, _ = e.a.DB.Exec(`UPDATE sessions SET status='blocked', ended_at='' WHERE id=?`, s.ID)
	_, _ = e.a.Queue.SetStatus(e.ctx, tk.ID, queue.Running, "test: simulate a crash")
	b.Sessions.Recover(e.ctx)
	if b.Sessions.LiveCount() != 1 {
		got, _ := b.Sessions.Get(e.ctx, s.ID)
		t.Fatalf("not reattached: live=%d session=%+v", b.Sessions.LiveCount(), got)
	}
	if err := b.Herdr.C.Call(e.ctx, herdr.MethodPaneSendText, map[string]any{"pane_id": blocked.PaneID, "text": "y\n"}, nil); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		if got, _ := b.Queue.Get(e.ctx, tk.ID); got.Status == queue.Review {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	got, _ := b.Queue.Get(e.ctx, tk.ID)
	ss, _ := b.Sessions.Get(e.ctx, s.ID)
	t.Fatalf("after reattach: task %s session %+v", got.Status, ss)
}
