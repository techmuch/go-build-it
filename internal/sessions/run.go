package sessions

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/techmuch/go-build-it/internal/access"
	"github.com/techmuch/go-build-it/internal/harness"
	"github.com/techmuch/go-build-it/internal/herdr"
	"github.com/techmuch/go-build-it/internal/history"
	"github.com/techmuch/go-build-it/internal/projects"
	"github.com/techmuch/go-build-it/internal/queue"
	"github.com/techmuch/go-build-it/internal/store"
	"github.com/techmuch/go-build-it/internal/team"
)

// run is one live session.
type run struct {
	s      Session
	t      queue.Task
	p      projects.Project
	mem    team.Member
	prof   harness.Profile
	h      *herdr.Client
	cancel context.CancelFunc

	pane   string
	state  string // starting | working | blocked | idle (herdr view)
	reason string // cancelled | blocked (question)
	seq    int
	reader harness.Reader
	path   string
}

func (r *run) paneID() string    { return r.pane }
func (r *run) agentName() string { return agentName(r.mem) }

func (r *run) agentTarget() string {
	if r.pane != "" {
		return r.pane
	}
	return agentName(r.mem)
}

// Prompt is what every task dispatch sends; the task itself is in TaskFile.
const Prompt = "Read " + TaskFile + " in this folder and do what it says."

func (m *Manager) setState(r *run, state string) {
	m.mu.Lock()
	changed := r.state != state
	r.state = state
	m.mu.Unlock()
	if changed {
		m.hist.Publish(history.Event{Type: "session.state", Subject: r.s.ID, ProjectID: r.s.ProjectID, Summary: state,
			Data: history.Data(map[string]any{"member": r.mem.ID, "state": state, "pane": r.pane})})
		m.RefreshProjectSummary(context.Background(), r.s.ProjectID)
	}
}

// drive launches or resumes the member, submits the task and follows it to the end.
func (m *Manager) drive(ctx context.Context, r *run) {
	defer func() {
		m.mu.Lock()
		delete(m.running, r.s.ID)
		m.mu.Unlock()
		m.RefreshProjectSummary(context.Background(), r.s.ProjectID)
	}()
	l, err := m.ensureAgent(ctx, r.h, r.p, r.mem, r.s.Workdir)
	if err != nil {
		m.fail(ctx, r, "could not start "+r.mem.Name+" in herdr: "+err.Error())
		return
	}
	r.pane, r.s.PaneID, r.s.HarnessSession = l.pane, l.pane, l.conv
	m.label(ctx, r.h, r.mem, r.pane, r.t.ID+" "+r.t.Title, 0, r.prof.UsageMeasured, r.mem.ConversationTasks)
	if l.blocked {
		if !m.waitUnblocked(ctx, r, l.message) {
			m.finishCancelled(ctx, r)
			return
		}
	}
	if !m.waitReady(ctx, r) {
		m.finishCancelled(ctx, r)
		return
	}
	r.s.Status = "running"
	_ = m.write(ctx, r.s)
	m.setState(r, "working")
	m.openTranscript(r)
	var seq int64
	for i := 0; ; i++ {
		seq, err = r.h.Submit(ctx, r.pane, agentName(r.mem), r.prof.Kind != "", Prompt)
		if err == nil || !herdr.IsCode(err, "agent_not_ready") || i >= 20 {
			break
		}
		time.Sleep(500 * time.Millisecond) // herdr marks a just-started agent ready a moment later
	}
	if err != nil {
		m.fail(ctx, r, "could not send the task to "+r.mem.Name+": "+err.Error())
		return
	}
	m.follow(ctx, r, seq)
}

// waitUnblocked records a blocked member as needing the human and waits until herdr no
// longer reports it blocked (FR-68). go-build-it answers only folder-trust screens in its
// own folders (FR-07f); every other dialog is the person's.
func (m *Manager) waitUnblocked(ctx context.Context, r *run, msg string) bool {
	r.s.Status = "blocked"
	r.s.ExitReason = firstNonEmpty(msg, r.mem.Name+" needs you in its terminal")
	_ = m.write(ctx, r.s)
	m.setState(r, "blocked")
	m.hist.Must(ctx, history.Event{ProjectID: r.s.ProjectID, Type: "session.blocked", Subject: r.s.ID, Summary: r.mem.Name + " needs you before starting " + r.t.ID,
		Why: r.s.ExitReason, Data: history.Data(map[string]any{"member": r.mem.ID, "pane": r.pane, "task": r.t.ID})})
	for {
		select {
		case <-ctx.Done():
			return false
		case <-time.After(m.Poll):
		}
		a, err := r.h.Agent(ctx, r.pane)
		if err == nil && a.AgentStatus != herdr.AgentStatusBlocked && a.AgentStatus != herdr.AgentStatusUnknown {
			r.s.ExitReason = ""
			m.hist.Must(ctx, history.Event{ProjectID: r.s.ProjectID, Type: "session.unblocked", Subject: r.s.ID, Summary: r.mem.Name + " is ready again (" + r.t.ID + ")"})
			if r.s.HarnessSession == "" && a.AgentSession != nil {
				r.s.HarnessSession = a.AgentSession.Value
			}
			return true
		}
	}
}

// waitReady waits until herdr reports the member ready for a prompt; a blocked member
// becomes a needs-you item first (FR-68).
func (m *Manager) waitReady(ctx context.Context, r *run) bool {
	deadline := time.Now().Add(60 * time.Second)
	for time.Now().Before(deadline) {
		a, err := r.h.Agent(ctx, r.pane)
		if err == nil {
			if a.AgentStatus == herdr.AgentStatusBlocked {
				if m.answerTrust(ctx, r.h, r.s.ProjectID, r.mem.Name, r.prof.Label, r.pane) {
					continue
				}
				if !m.waitUnblocked(ctx, r, r.mem.Name+" is waiting for you in its terminal") {
					return false
				}
				continue
			}
			ready := r.prof.Kind == "" || a.InteractiveReady || !a.LaunchPending
			if ready && (a.AgentStatus == herdr.AgentStatusIdle || a.AgentStatus == herdr.AgentStatusDone) {
				return true
			}
		}
		select {
		case <-ctx.Done():
			return false
		case <-time.After(m.Poll):
		}
	}
	return true // let Submit report what is wrong
}

func (m *Manager) openTranscript(r *run) {
	r.reader = r.prof.NewReader()
	r.path = r.prof.TranscriptPath(r.s.Workdir, r.s.HarnessSession)
	if r.reader != nil && r.path != "" {
		_, _ = r.reader.Read(r.path) // skip earlier tasks in the same conversation
	}
}

// follow polls herdr state and the transcript until the member settles after seq.
func (m *Manager) follow(ctx context.Context, r *run, seq int64) {
	var limited *harness.RateLimit
	var resultErr bool
	var resultText, lastText string
	sawChange, lastSave := false, time.Now()
	submitted := time.Now()
	handoffAsked := false
	for {
		select {
		case <-ctx.Done():
			m.finishCancelled(ctx, r)
			return
		case <-time.After(m.Poll):
		}
		// Transcript: structured view, usage, compactions, limits.
		if r.reader == nil || r.path == "" {
			if r.s.HarnessSession == "" {
				if a, err := r.h.Agent(ctx, r.pane); err == nil && a.AgentSession != nil {
					r.s.HarnessSession = a.AgentSession.Value
				} else if id := r.prof.FindNewConversation(submitted.Add(-time.Minute)); id != "" {
					r.s.HarnessSession = id
				}
			}
			if r.s.HarnessSession != "" && r.reader == nil {
				m.openTranscript(r)
			} else if r.reader != nil && r.path == "" {
				r.path = r.prof.TranscriptPath(r.s.Workdir, r.s.HarnessSession)
			}
		}
		if r.reader != nil && r.path != "" {
			evs, _ := r.reader.Read(r.path)
			for _, e := range evs {
				switch e.Kind {
				case harness.KUser:
					m.appendLine(ctx, r, "input", e.Text)
				case harness.KText:
					m.appendLine(ctx, r, "text", e.Text)
					lastText = e.Text
					if hb := handoffRe.FindStringSubmatch(e.Text); hb != nil && r.s.Handoff == "" {
						r.s.Handoff = strings.TrimSpace(hb[1])
					}
				case harness.KTool:
					m.appendLine(ctx, r, "tool", e.Text)
				case harness.KError:
					m.appendLine(ctx, r, "error", e.Text)
				case harness.KCompacted:
					r.s.Compactions++
					m.appendLine(ctx, r, "system", "⟲ "+e.Text)
					_, _ = m.queue.Bump(ctx, r.t.ID, "compactions")
					m.hist.Must(ctx, history.Event{ProjectID: r.s.ProjectID, Type: "session.compacted", Subject: r.s.ID,
						Summary: fmt.Sprintf("Context compacted in %s (%s)", r.s.ID, r.t.ID), Why: "harness auto-compaction (default)",
						Data: history.Data(map[string]any{"task": r.t.ID, "usedBefore": r.s.Used})})
				case harness.KLimit:
					limited = e.Limit
					m.appendLine(ctx, r, "system", "subscription limit until "+e.Limit.ResetsAt.Local().Format("15:04"))
				case harness.KResult:
					resultErr, resultText = e.IsError, e.Text
					m.appendLine(ctx, r, "result", firstNonEmpty(e.Text, "turn finished"))
				}
				if e.Usage != nil {
					r.s.Used, r.s.Measured = e.Usage.Used, e.Usage.Measured
					if r.s.Window > 0 {
						r.s.Fraction = float64(r.s.Used) / float64(r.s.Window)
					}
					// Experimental smart handoff: ask for a briefing before the harness compacts (FR-72).
					if !handoffAsked && m.cfg.HandoffsEnabledFor(r.s.ProjectID) && r.s.Fraction >= m.cfg.Experimental.HandoffThreshold && m.Hooks.Handoff != nil {
						handoffAsked = true
						if msg := m.Hooks.Handoff(ctx, r.s, r.t); msg != "" {
							_, _ = r.h.Submit(ctx, r.pane, agentName(r.mem), r.prof.Kind != "", msg)
						}
					}
				}
			}
			if time.Since(lastSave) > time.Second {
				_ = m.write(ctx, r.s)
				lastSave = time.Now()
				m.hist.Publish(history.Event{Type: "session.usage", Subject: r.s.ID, ProjectID: r.s.ProjectID, Data: history.Data(r.s)})
				m.label(ctx, r.h, r.mem, r.pane, r.t.ID+" "+r.t.Title, r.s.Fraction, r.s.Measured, r.mem.ConversationTasks)
			}
		}
		// herdr state is the source of truth (FR-68).
		a, err := r.h.Agent(ctx, r.pane)
		if err != nil {
			if herdr.IsCode(err, "agent_not_found") || herdr.IsCode(err, "not_found") || herdr.IsCode(err, "pane_not_found") {
				m.fail(ctx, r, r.mem.Name+"'s harness exited")
				return
			}
			continue
		}
		if a.StateChangeSeq > seq {
			sawChange = true
		}
		switch a.AgentStatus {
		case herdr.AgentStatusBlocked:
			if r.state != "blocked" && m.answerTrust(ctx, r.h, r.s.ProjectID, r.mem.Name, r.prof.Label, r.pane) {
				continue
			}
			if r.state != "blocked" {
				r.s.Status, r.s.ExitReason = "blocked", r.mem.Name+" is waiting for you in its terminal"
				_ = m.write(ctx, r.s)
				m.setState(r, "blocked")
				m.hist.Must(ctx, history.Event{ProjectID: r.s.ProjectID, Type: "session.blocked", Subject: r.s.ID, Summary: r.mem.Name + " needs you (" + r.t.ID + ")",
					Why: "herdr reports the agent blocked", Data: history.Data(map[string]any{"member": r.mem.ID, "pane": r.pane, "task": r.t.ID})})
			}
			continue
		case herdr.AgentStatusWorking:
			if r.state == "blocked" {
				m.hist.Must(ctx, history.Event{ProjectID: r.s.ProjectID, Type: "session.unblocked", Subject: r.s.ID, Summary: r.mem.Name + " is working again (" + r.t.ID + ")"})
			}
			if r.state != "working" {
				r.s.Status, r.s.ExitReason = "running", ""
				_ = m.write(ctx, r.s)
				m.setState(r, "working")
			}
			continue
		}
		if !sawChange {
			if time.Since(submitted) > 45*time.Second {
				m.fail(ctx, r, r.mem.Name+" did not start working after the prompt")
				return
			}
			continue
		}
		// Settled (idle or done) after the prompt. Read the transcript once more.
		if r.reader != nil && r.path != "" {
			if evs, _ := r.reader.Read(r.path); len(evs) > 0 {
				for _, e := range evs {
					if e.Kind == harness.KText {
						m.appendLine(ctx, r, "text", e.Text)
						lastText = e.Text
					}
					if e.Kind == harness.KResult {
						resultErr, resultText = e.IsError, e.Text
					}
					if e.Kind == harness.KLimit {
						limited = e.Limit
					}
					if e.Usage != nil {
						r.s.Used = e.Usage.Used
					}
				}
			}
		}
		m.finish(ctx, r, limited, resultErr, firstNonEmpty(resultText, lastText))
		return
	}
}

func (m *Manager) appendLine(ctx context.Context, r *run, kind, text string) {
	r.seq++
	ts := store.Now()
	if r.seq == 1 {
		_ = m.db.QueryRowContext(ctx, `SELECT COALESCE(MAX(seq),0)+1 FROM session_output WHERE session_id=?`, r.s.ID).Scan(&r.seq)
	}
	_, _ = m.db.ExecContext(ctx, `INSERT INTO session_output(session_id,seq,ts,kind,text) VALUES(?,?,?,?,?)`, r.s.ID, r.seq, ts, kind, text)
	m.hist.Publish(history.Event{Type: "session.output", Subject: r.s.ID, ProjectID: r.s.ProjectID, TS: ts, Summary: text,
		Data: history.Data(map[string]any{"seq": r.seq, "kind": kind})})
}

// release frees the member before the task's final status is written, so the member can
// take its next task the moment this one reaches review.
func (m *Manager) release(r *run) {
	m.mu.Lock()
	delete(m.running, r.s.ID)
	m.mu.Unlock()
}

func (m *Manager) fail(ctx context.Context, r *run, why string) {
	ctx = context.WithoutCancel(ctx)
	m.release(r)
	r.s.Status, r.s.EndedAt, r.s.ExitReason = "failed", store.Now(), why
	_ = m.write(ctx, r.s) // session first: readers never see a failed task with a running session
	_, _ = m.queue.Fail(ctx, r.t.ID, "session failed: "+trunc(why, 200))
	m.hist.Must(ctx, history.Event{ProjectID: r.s.ProjectID, Type: "session.failed", Subject: r.s.ID, Summary: fmt.Sprintf("%s failed (%s)", r.s.ID, r.t.ID), Why: trunc(why, 300)})
	m.label(ctx, r.h, r.mem, r.pane, "", 0, true, r.mem.ConversationTasks)
}

func (m *Manager) finishCancelled(ctx context.Context, r *run) {
	ctx = context.WithoutCancel(ctx)
	m.release(r)
	m.mu.Lock()
	reason := r.reason
	m.mu.Unlock()
	r.s.EndedAt = store.Now()
	if reason == "blocked" {
		r.s.Status, r.s.ExitReason = "done", "stopped for a blocking question"
		_ = m.write(ctx, r.s)
		return
	}
	r.s.Status, r.s.ExitReason = "cancelled", "cancelled by "+access.FromContext(ctx).Name
	_ = m.write(ctx, r.s)
	_, _ = m.queue.SetStatus(ctx, r.t.ID, queue.Queued, "session cancelled")
	m.label(ctx, r.h, r.mem, r.pane, "", 0, true, r.mem.ConversationTasks)
}

// finish handles a settled member: rate limit, failure, a blocking question, a smart
// handoff, or completion (Courier commits, pushes and opens the review).
func (m *Manager) finish(ctx context.Context, r *run, limited *harness.RateLimit, resultErr bool, resultText string) {
	ctx = context.WithoutCancel(ctx)
	// The member is released just before each final task status: after Courier has finished
	// with its worktree, before the task can be seen as done.
	r.s.EndedAt = store.Now()
	t := r.t
	mem, _ := m.team.Member(ctx, r.mem.ID)
	tasks := mem.ConversationTasks + 1
	_ = m.team.SetRuntime(ctx, mem.ID, r.s.Workdir, r.pane, r.s.HarnessSession, tasks, firstNonEmpty(mem.ConversationStarted, store.Now()))
	defer m.label(ctx, r.h, mem, r.pane, "", r.s.Fraction, r.s.Measured, tasks)
	m.mu.Lock()
	reason := r.reason
	m.mu.Unlock()
	switch {
	case limited != nil:
		// FR-65: pause and resume automatically when the limit resets.
		r.s.Status, r.s.ResumeAt, r.s.ExitReason = "rate_limited", limited.ResetsAt.UTC().Format(time.RFC3339), "subscription limit ("+limited.Kind+")"
		_ = m.write(ctx, r.s)
		m.release(r)
		_, _ = m.queue.SetStatus(ctx, t.ID, queue.Paused, "rate limited until "+limited.ResetsAt.Local().Format("15:04"))
		m.hist.Must(ctx, history.Event{ProjectID: r.s.ProjectID, Type: "session.rate_limited", Subject: r.s.ID, Summary: fmt.Sprintf("%s paused: %s limit", t.ID, r.s.Harness),
			Why: "resumes at " + limited.ResetsAt.Local().Format("15:04"), Data: history.Data(map[string]any{"task": t.ID, "resumeAt": r.s.ResumeAt})})
	case resultErr:
		m.fail(ctx, r, firstNonEmpty(resultText, "the harness reported an error"))
	case reason == "blocked":
		r.s.Status, r.s.ExitReason = "done", "stopped for a blocking question"
		_ = m.write(ctx, r.s)
		m.release(r)
	case r.s.Handoff != "":
		// Smart handoff (experimental): the member moves to a new conversation that starts from the briefing.
		r.s.Status, r.s.ExitReason = "done", "handed off"
		_ = m.write(ctx, r.s)
		cur, _ := m.queue.Get(ctx, t.ID)
		cur.ContextNote = strings.TrimSpace(cur.ContextNote + "\n\nHandoff briefing from " + r.s.ID + ":\n" + r.s.Handoff)
		_, _ = m.queue.Update(ctx, cur)
		_, _ = m.queue.Bump(ctx, t.ID, "handoffs")
		m.rotate(ctx, mem, r, "handed off at "+fmt.Sprintf("%.0f%%", r.s.Fraction*100))
		m.release(r)
		_, _ = m.queue.SetStatus(ctx, t.ID, queue.Queued, "handed off at "+fmt.Sprintf("%.0f%%", r.s.Fraction*100)+"; a fresh conversation continues from the briefing")
		m.hist.Must(ctx, history.Event{ProjectID: r.s.ProjectID, Type: "handoff.written", Subject: r.s.ID, Summary: fmt.Sprintf("Handoff written for %s (%s)", t.ID, r.s.ID),
			Why: "experimental smart handoffs; context reached the threshold", Data: history.Data(map[string]any{"task": t.ID, "briefing": r.s.Handoff, "usedBefore": r.s.Used, "missingSections": MissingHandoffSections(r.s.Handoff)})})
	default:
		r.s.Status, r.s.ExitReason = "done", "completed"
		cur, _ := m.queue.Get(ctx, t.ID)
		if cur.Status == queue.Blocked {
			r.s.ExitReason = "stopped for a blocking question"
			_ = m.write(ctx, r.s)
			m.release(r)
			return
		}
		_ = os.Remove(filepath.Join(r.s.Workdir, TaskFile))
		m.AddNote(ctx, mem.ID, t.ID, t.Title+": "+resultText)
		// Q8: keep the conversation across tasks until context reaches the threshold, then rotate.
		// Done before the task reaches review, so a next task never finds the old process.
		if th := m.rotationThreshold(); r.s.Fraction >= th {
			m.rotate(ctx, mem, r, fmt.Sprintf("context at %.0f%% (threshold %.0f%%)", r.s.Fraction*100, th*100))
			tasks = 0
		}
		if m.Hooks.Completed != nil {
			if err := m.Hooks.Completed(ctx, r.s, cur); err != nil {
				m.hist.Must(ctx, history.Event{ProjectID: r.s.ProjectID, Type: "courier.error", Subject: t.ID, Summary: "Courier could not finish " + t.ID, Why: err.Error()})
				_ = m.write(ctx, r.s)
				m.release(r)
				_, _ = m.queue.SetStatus(ctx, t.ID, queue.Failed, "courier: "+err.Error())
				return
			}
		}
		_ = m.write(ctx, r.s)
		m.release(r)
		if cur2, _ := m.queue.Get(ctx, t.ID); cur2.Status == queue.Running {
			_, _ = m.queue.SetStatus(ctx, t.ID, queue.Review, "session completed: "+trunc(resultText, 160))
		}
		m.hist.Must(ctx, history.Event{ProjectID: r.s.ProjectID, Type: "session.completed", Subject: r.s.ID, Summary: fmt.Sprintf("%s finished %s", mem.Name, t.ID), Why: trunc(resultText, 300)})
	}
	m.hist.Publish(history.Event{Type: "session.usage", Subject: r.s.ID, ProjectID: r.s.ProjectID, Data: history.Data(r.s)})
}

func (m *Manager) rotationThreshold() float64 {
	if th := m.cfg.Experimental.HandoffThreshold; th > 0 {
		return th
	}
	return 0.75
}

// rotate ends a member's conversation: the next task starts a new one from FR-70's
// prepared context plus the member's notes (FR-76, Q8). The old harness process exits.
func (m *Manager) rotate(ctx context.Context, mem team.Member, r *run, why string) {
	old := mem.ConversationID
	_ = m.team.SetRuntime(ctx, mem.ID, r.s.Workdir, r.pane, "", 0, "")
	if r.prof.Kind != "" {
		_ = r.h.SendKeys(ctx, agentName(mem), "ctrl+c")
		_ = r.h.SendKeys(ctx, agentName(mem), "ctrl+c")
	} else if r.pane != "" {
		_ = r.h.Call(ctx, herdr.MethodPaneSendText, map[string]any{"pane_id": r.pane, "text": "/exit\n"}, nil)
	}
	waitGone(ctx, r.h, r.pane, 15*time.Second) // the next task must launch a new process, not find the old one
	m.hist.Must(ctx, history.Event{ProjectID: r.s.ProjectID, Type: "member.rotated", Subject: mem.ID, Summary: mem.Name + " starts a new conversation next task",
		Why: why, Data: history.Data(map[string]any{"previousConversation": old, "used": r.s.Used})})
}

// Recover reattaches to members that kept working while go-build-it restarted (FR-08,
// FR-82): herdr owns the processes, so live sessions are followed again rather than
// failed. Sessions whose pane is gone are requeued.
func (m *Manager) Recover(ctx context.Context) {
	rows, err := m.db.QueryContext(ctx, `SELECT `+cols+` FROM sessions WHERE status IN ('starting','running','blocked')`)
	if err != nil {
		return
	}
	var live []Session
	for rows.Next() {
		if s, err := scan(rows); err == nil {
			live = append(live, s)
		}
	}
	rows.Close()
	h, herr := m.herdr()
	requeued := 0
	for _, s := range live {
		t, terr := m.queue.Get(ctx, s.TaskID)
		p, perr := m.projects.Get(ctx, s.ProjectID)
		mem, merr := m.team.Member(ctx, s.MemberID)
		prof, pok := harness.Get(s.Harness)
		var a herdr.AgentInfo
		var aerr error = errors.New("no herdr")
		if herr == nil && s.PaneID != "" {
			a, aerr = h.Agent(ctx, s.PaneID)
		}
		if terr != nil || perr != nil || merr != nil || !pok || aerr != nil || a.Agent == nil {
			_, _ = m.db.ExecContext(ctx, `UPDATE sessions SET status='failed', ended_at=?, exit_reason='interrupted: the member was not running after a restart' WHERE id=?`, store.Now(), s.ID)
			continue
		}
		rctx, cancel := context.WithCancel(access.Persona(context.Background(), mem.Name))
		r := &run{s: s, t: t, p: p, mem: mem, prof: prof, h: h, cancel: cancel, pane: s.PaneID, state: map[bool]string{true: "blocked", false: "working"}[s.Status == "blocked"]}
		m.mu.Lock()
		m.running[s.ID] = r
		m.mu.Unlock()
		m.openTranscript(r)
		go func(r *run, seq int64) {
			defer func() {
				m.mu.Lock()
				delete(m.running, r.s.ID)
				m.mu.Unlock()
			}()
			m.follow(rctx, r, seq-1) // any settled state now counts as the end of the task
		}(r, a.StateChangeSeq)
		m.hist.Must(ctx, history.Event{ProjectID: s.ProjectID, Type: "session.reattached", Subject: s.ID, Summary: "Reattached to " + mem.Name + " after a restart (" + t.ID + ")",
			Why: "herdr kept the member running (FR-08)"})
	}
	// Running tasks without a reattached member go back to the queue.
	attached := map[string]bool{}
	m.mu.Lock()
	for _, r := range m.running {
		attached[r.t.ID] = true
	}
	m.mu.Unlock()
	if ts, err := m.queue.List(ctx, queue.Filter{Status: queue.Running}); err == nil {
		for _, t := range ts {
			if !attached[t.ID] {
				_, _ = m.queue.SetStatus(ctx, t.ID, queue.Queued, "go-build-it restarted and the member was not running; resuming")
				requeued++
			}
		}
	}
	if requeued > 0 {
		m.hist.Must(ctx, history.Event{Type: "system.recovered", Summary: fmt.Sprintf("Requeued %d interrupted task(s)", requeued), Why: "go-build-it restarted"})
	}
}

// RestartMember stops a member's harness in herdr. With fresh, the next task starts a new
// conversation; otherwise it resumes the same one (FR-61, FR-69). A member working on a
// task must be cancelled first.
func (m *Manager) RestartMember(ctx context.Context, memberID string, fresh bool) error {
	if m.memberBusy(memberID) {
		return fmt.Errorf("%s is working on a task; cancel it first", memberID)
	}
	h, err := m.herdr()
	if err != nil {
		return err
	}
	mem, err := m.team.Member(ctx, memberID)
	if err != nil {
		return err
	}
	prof, _ := harness.Get(mem.Harness)
	if mem.PaneID != "" {
		if prof.Kind != "" {
			_ = h.SendKeys(ctx, mem.PaneID, "ctrl+c")
			_ = h.SendKeys(ctx, mem.PaneID, "ctrl+c")
		} else {
			_ = h.Call(ctx, herdr.MethodPaneSendText, map[string]any{"pane_id": mem.PaneID, "text": "/exit\n"}, nil)
		}
	}
	conv, tasks, started := mem.ConversationID, mem.ConversationTasks, mem.ConversationStarted
	if fresh {
		conv, tasks, started = "", 0, ""
	}
	if mem.PaneID != "" {
		waitGone(ctx, h, mem.PaneID, 15*time.Second)
	}
	if err := m.team.SetRuntime(ctx, mem.ID, mem.Worktree, mem.PaneID, conv, tasks, started); err != nil {
		return err
	}
	m.hist.Must(ctx, history.Event{ProjectID: mem.ProjectID, Type: "member.restarted", Subject: mem.ID,
		Summary: fmt.Sprintf("Restarted %s (%s conversation next task)", mem.Name, map[bool]string{true: "new", false: "same"}[fresh])})
	return nil
}

// waitGone waits until no agent occupies the pane (its harness exited and herdr released it).
func waitGone(ctx context.Context, h *herdr.Client, pane string, d time.Duration) {
	if pane == "" {
		return
	}
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		a, err := h.Agent(ctx, pane)
		if err != nil || a.Agent == nil || *a.Agent == "" {
			return
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(200 * time.Millisecond):
		}
	}
}
