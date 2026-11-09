// Package app wires go-build-it's services together and runs background loops.
package app

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/techmuch/go-build-it/internal/doctor"
	"github.com/techmuch/go-build-it/internal/learn"
	"log/slog"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/techmuch/go-build-it/internal/access"
	"github.com/techmuch/go-build-it/internal/agent"
	"github.com/techmuch/go-build-it/internal/config"
	"github.com/techmuch/go-build-it/internal/courier"
	"github.com/techmuch/go-build-it/internal/gitx"
	"github.com/techmuch/go-build-it/internal/herdr"
	"github.com/techmuch/go-build-it/internal/history"
	"github.com/techmuch/go-build-it/internal/options"
	"github.com/techmuch/go-build-it/internal/pins"
	"github.com/techmuch/go-build-it/internal/projects"
	"github.com/techmuch/go-build-it/internal/proposals"
	"github.com/techmuch/go-build-it/internal/questions"
	"github.com/techmuch/go-build-it/internal/queue"
	"github.com/techmuch/go-build-it/internal/rehearsals"
	"github.com/techmuch/go-build-it/internal/reviews"
	"github.com/techmuch/go-build-it/internal/sessions"
	"github.com/techmuch/go-build-it/internal/standards"
	"github.com/techmuch/go-build-it/internal/store"
	"github.com/techmuch/go-build-it/internal/team"
	"github.com/techmuch/go-build-it/internal/update"
)

// App holds every service.
type App struct {
	Version    string
	Cfg        *config.Config
	DB         *sql.DB
	Log        *slog.Logger
	Hist       *history.Recorder
	Git        *gitx.Git
	Forge      *gitx.ForgeCLI
	Projects   *projects.Service
	Team       *team.Service
	Standards  *standards.Service
	Queue      *queue.Service
	Sessions   *sessions.Manager
	Courier    *courier.Courier
	Questions  *questions.Service
	Reviews    *reviews.Service
	Rehearsals *rehearsals.Service
	Options    *options.Service
	Proposals  *proposals.Service
	Pins       *pins.Service
	Agent      *agent.Caller
	Updater    *update.Updater
	Updates    *update.Jobs // update and rollback jobs with their steps (FR-06d)
	Herdr      *herdr.Supervisor

	herdrReady  chan struct{} // closed once the first herdr Ensure has returned
	Doctor      *doctor.Doctor
	Learn       learn.Progress
	lessonsOnce sync.Once
	lessonList  []learn.Lesson
	practiceMu  sync.Mutex
	doctorMu    sync.Mutex
	lastCheck   atomic.Pointer[update.Release]
}

// New opens the database and connects all services.
func New(cfg *config.Config, version string, log *slog.Logger) (*App, error) {
	db, err := store.Open(cfg.DBPath())
	if err != nil {
		return nil, err
	}
	return Wire(cfg, db, version, log), nil
}

// Wire connects services around an open database (tests pass an in-memory DB).
func Wire(cfg *config.Config, db *sql.DB, version string, log *slog.Logger) *App {
	a := &App{Version: version, Cfg: cfg, DB: db, Log: log}
	a.Hist = history.New(db)
	a.Herdr = herdr.NewSupervisor(cfg.Herdr.Binary, cfg.Herdr.Session, cfg.Herdr.ConfigHome, log)
	a.Git = gitx.New()
	a.Forge = gitx.NewForge()
	a.Projects = projects.New(db, a.Hist, a.Git, a.Forge)
	a.Team = team.New(db, a.Hist)
	a.Standards = standards.New(db, a.Hist)
	a.Queue = queue.New(db, a.Hist)
	a.Sessions = sessions.New(db, a.Hist, cfg, a.Projects, a.Queue, a.Team, a.Standards, a.Git, log)
	a.Courier = courier.New(a.Git, a.Forge, a.Hist, a.Projects, a.Queue, a.Standards, a.Sessions.TaskDir)
	a.Questions = questions.New(db, a.Hist, a.Queue)
	a.Reviews = reviews.New(db, a.Hist, a.Queue, a.Projects, a.Standards)
	a.Rehearsals = rehearsals.New(db, a.Hist, a.Queue, a.Projects)
	a.Options = options.New(db, a.Hist)
	a.Proposals = proposals.New(db, a.Hist)
	a.Pins = pins.New(db, a.Hist)
	a.Agent = agent.New(cfg, a.Team, a.Sessions.Ask)
	a.Sessions.SetHerdr(a.Herdr.C)
	a.Learn = learn.Progress{DB: db}
	a.Doctor = &doctor.Doctor{Cfg: cfg, Sup: a.Herdr, Exe: a.Sessions.GbiBin, Version: version}
	a.Updater = update.New(cfg.Updates.Repo, version)
	if cfg.Updates.API != "" {
		a.Updater.API = strings.TrimRight(cfg.Updates.API, "/")
	}
	jobFile := ""
	if cfg.Dir() != "" {
		jobFile = filepath.Join(cfg.Dir(), "update", "job.json")
	}
	a.Updates = update.NewJobs(a.Updater, jobFile)
	a.Updates.OnEvent = a.updateEvent
	a.herdrReady = make(chan struct{})

	// Review gates hold back work in later milestones until approved (FR-115).
	a.Queue.GateOpen = a.gateOpen
	// Session done → Courier commits, pushes, opens PR → Lens opens the review (FR-92, FR-112).
	a.Sessions.Hooks.Completed = a.Courier.Completed
	a.Courier.AfterPush = func(ctx context.Context, t queue.Task) error {
		_, err := a.Reviews.Open(ctx, t.ID)
		return err
	}
	a.Reviews.Hooks = reviews.Hooks{Merge: a.Courier.Merge, Lens: a.lens, Feedback: a.feedbackFor, Workdir: a.Sessions.WorktreePath, Assign: a.assignReviewer}
	a.Questions.Hooks = questions.Hooks{
		StopSession:   a.Sessions.StopForQuestion,
		SendToSession: a.Sessions.Send,
		ProposeValue:  a.proposeValue,
	}
	a.Options.Hooks = options.Hooks{Values: a.valueTexts, ProposeValue: a.proposeValue, FollowUp: a.optionFollowUp}
	a.Rehearsals.IsOwner = cfg.IsOwner
	a.Proposals.OnAccept = a.proposalTask
	a.Sessions.Hooks.Handoff = a.handoffRequest
	a.Courier.Notes = a.releaseNotes
	return a
}

func (a *App) gateOpen(ctx context.Context, milestoneID string) (bool, string) {
	m, err := a.Projects.Milestone(ctx, milestoneID)
	if err != nil {
		return true, ""
	}
	ms, err := a.Projects.Milestones(ctx, m.ProjectID)
	if err != nil {
		return true, ""
	}
	for _, prev := range ms {
		if prev.Ord < m.Ord && prev.GateName != "" && prev.GateStatus != "approved" {
			return false, fmt.Sprintf("waits for gate %q (%s)", prev.GateName, prev.Name)
		}
	}
	return true, ""
}

func (a *App) proposeValue(ctx context.Context, projectID, text, source string) error {
	_, err := a.Projects.AddValue(ctx, projectID, text, source, "proposed")
	return err
}

func (a *App) valueTexts(ctx context.Context, projectID string) []string {
	vals, _ := a.Projects.Values(ctx, projectID, false)
	out := []string{}
	for _, v := range vals {
		if v.Status == "accepted" {
			out = append(out, v.Text)
		}
	}
	return out
}

// assignReviewer picks the human reviewer with the fewest pending reviews (P9-08), or "" for the owner.
func (a *App) assignReviewer(ctx context.Context, t queue.Task) string {
	p, _ := a.Projects.Get(ctx, t.ProjectID)
	ms, _ := a.Team.Members(ctx, p.ID)
	best, bestN := "", 1<<30
	for _, m := range ms {
		if m.Kind != "human" || m.Role != "reviewer" {
			continue
		}
		var n int
		_ = a.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM reviews WHERE status='pending' AND assignee=?`, m.ID).Scan(&n)
		if n < bestN {
			best, bestN = m.ID, n
		}
	}
	return best
}

// reviewer returns an AI reviewer: a project member when one exists, else the reviewer archetype.
func (a *App) reviewer(ctx context.Context) string {
	return a.defaultPersona(ctx, "", "reviewer")
}

func (a *App) lens(ctx context.Context, prompt string) (string, error) {
	id := a.reviewer(ctx)
	if id == "" {
		return "", fmt.Errorf("no AI persona holds the reviewer role")
	}
	return a.Agent.Ask(ctx, id, prompt)
}

// feedbackFor gathers rehearsal findings, earlier review comments and answers for a task (FR-112).
func (a *App) feedbackFor(ctx context.Context, t queue.Task) []reviews.Check {
	out := []reviews.Check{}
	if fs, err := a.Rehearsals.FindingsForTask(ctx, t.ProjectID, t.ID); err == nil {
		for _, f := range fs {
			if f.ChangeRequest == t.ID || f.ChangeRequest == "" {
				out = append(out, reviews.Check{Item: f.Text, Status: "unassessed", Author: f.Author, Evidence: "rehearsal " + f.RehearsalID})
			}
		}
	}
	if rs, err := a.Reviews.List(ctx, t.ProjectID, "changes"); err == nil {
		for _, r := range rs {
			if r.TaskID == t.ID {
				out = append(out, reviews.Check{Item: r.Comment, Status: "unassessed", Author: r.Reviewer, Evidence: "review " + r.ID})
			}
		}
	}
	return out
}

func (a *App) optionFollowUp(ctx context.Context, s options.Set) error {
	var parts []string
	for _, p := range s.Choice {
		if p.Aspect != "" {
			parts = append(parts, p.Option+": "+p.Aspect)
		} else {
			parts = append(parts, "option "+p.Option)
		}
	}
	_, err := a.Queue.Add(ctx, queue.Task{ProjectID: s.ProjectID, Role: "architect", Priority: 2, Title: "Combine choice from " + s.ID + ": " + s.Title,
		Description: fmt.Sprintf("The human chose %s in %s.\nReason: %s\n\nProduce the combined design (mockup or ADR) and update docs accordingly.", strings.Join(parts, ", "), s.ID, s.Reason)})
	return err
}

func (a *App) proposalTask(ctx context.Context, p proposals.Proposal) (string, error) {
	ps, _ := a.Projects.List(ctx)
	target := ""
	for _, pr := range ps {
		if pr.ID == "go-build-it" {
			target = pr.ID
		}
	}
	if target == "" && len(ps) > 0 {
		target = ps[0].ID
	}
	if target == "" {
		return "", fmt.Errorf("no project to hold the improvement task")
	}
	t, err := a.Queue.Add(ctx, queue.Task{ProjectID: target, Role: "developer", Title: "Improvement " + p.ID + ": " + p.Title, Description: p.Body + "\n\nEvidence: " + p.Evidence})
	return t.ID, err
}

// handoffRequest asks the session for a curated briefing (experimental smart handoffs, FR-72–FR-74).
func (a *App) handoffRequest(ctx context.Context, s sessions.Session, t queue.Task) string {
	a.Hist.Must(ctx, history.Event{ProjectID: s.ProjectID, Type: "handoff.requested", Subject: s.ID, Summary: fmt.Sprintf("Handoff requested for %s at %.0f%%", t.ID, s.Fraction*100),
		Why: "experimental smart handoffs enabled; context reached the threshold"})
	return HandoffPrompt
}

// releaseNotes asks the PM persona to turn commit subjects into release notes.
func (a *App) releaseNotes(ctx context.Context, projectID, tag string, subjects []string) (string, error) {
	pm := a.defaultPersona(ctx, projectID, "pm")
	if pm == "" {
		return "", fmt.Errorf("no PM persona")
	}
	return a.Agent.Ask(ctx, pm, fmt.Sprintf("Write short release notes in Markdown for %s %s, grouped under Added / Changed / Fixed, from these commit subjects. Start with the heading '## %s'. No preamble.\n\n- %s", projectID, tag, tag, strings.Join(subjects, "\n- ")))
}

// HandoffPrompt asks an agent for a curated briefing (FR-73).
const HandoffPrompt = `Your context window is nearly full. Stop the current work at a safe point and write a handoff briefing for the next person, as if handing over responsibility. Put it between the lines <<<HANDOFF and HANDOFF>>> with these headings:
Objective and definition of done / Current state and what is completed / Decisions made and why / Open problems, risks and dead ends to avoid / Specific next steps / Pointers (files and artifacts, as references).
Keep it under 800 words. Then end your turn.`

// Start runs background loops: recovery, push guarantee, auto-update checks.
func (a *App) Start(ctx context.Context) {
	// The agent backend: adopt or start the gbi herdr session, then keep it up (FR-07, FR-08).
	go func() {
		if _, err := a.Herdr.Ensure(ctx); err != nil {
			a.Log.Warn("herdr session not available; agents cannot run until it is", "err", err)
		}
		close(a.herdrReady)
		a.Herdr.Run(ctx)
	}()
	a.Sessions.Recover(ctx)
	// Finish an update that restarted into this version: restart and reconnect steps (ADR-011).
	go a.Updates.Resume(ctx, a.Version, a.reconnect)
	if a.Cfg.AutoDispatch {
		a.Sessions.SetAuto(true)
	}
	go func() {
		t := time.NewTicker(5 * time.Minute)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				if n, _ := a.Courier.EnsurePushed(context.Background()); n > 0 {
					a.Log.Info("pushed unpushed task branches", "count", n)
				}
			}
		}
	}()
	go a.updateLoop(ctx)
}

func (a *App) updateLoop(ctx context.Context) {
	if a.Version == "dev" {
		return // development builds are updated by hand from the change log
	}
	t := time.NewTimer(time.Minute)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
		t.Reset(6 * time.Hour)
		rel, err := a.checkUpdates(ctx)
		if err != nil || !rel.Newer || !a.Cfg.Updates.AutoInstall || a.Updates.Busy() {
			continue
		}
		if a.Sessions.LiveCount() > 0 {
			continue // the automatic updater never restarts in the middle of someone's work (ADR-011)
		}
		if _, err := a.Updates.Start("auto", "go-build-it"); err != nil {
			a.Log.Warn("automatic update did not start", "err", err)
		}
	}
}

// checkUpdates checks for a newer release and remembers the result for the sidebar.
func (a *App) checkUpdates(ctx context.Context) (update.Release, error) {
	rel, err := a.Updater.Check(ctx)
	if err == nil {
		a.lastCheck.Store(&rel)
	}
	return rel, err
}

// reconnect is the last update step, run by the restarted go-build-it: wait for the gbi
// herdr session and report the members that kept working (FR-06d, FR-08).
func (a *App) reconnect(ctx context.Context) (string, error) {
	select {
	case <-a.herdrReady:
	case <-ctx.Done():
		return "", errors.New("timed out waiting for the herdr session")
	}
	pctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if _, err := a.Herdr.C.Ping(pctx); err != nil {
		return "", fmt.Errorf("herdr session %s is not answering: %w", a.Herdr.C.Session, err)
	}
	n := a.Sessions.LiveCount()
	return fmt.Sprintf("herdr session %s · %d %s still working", a.Herdr.C.Session, n, map[bool]string{true: "member", false: "members"}[n == 1]), nil
}

// updateEvent records update milestones in the history (FR-06e, NFR-10).
func (a *App) updateEvent(kind string, j update.Job) {
	ctx := access.Persona(context.Background(), firstNonEmpty(j.Actor, "go-build-it"))
	data := history.Data(map[string]any{"job": j.ID, "kind": j.Kind, "trigger": j.Trigger, "from": j.From, "to": j.To, "steps": j.Steps})
	to := firstNonEmpty(j.To, "the latest release")
	switch kind {
	case "started":
		if j.Kind == "rollback" {
			a.Hist.Must(ctx, history.Event{Type: "update.rollback.started", Subject: j.ID, Summary: "Rolling back from " + j.From + " to " + to, Why: j.Trigger, Data: data})
		} else {
			a.Hist.Must(ctx, history.Event{Type: "update.started", Subject: j.ID, Summary: "Updating from " + j.From, Why: j.Trigger, Data: data})
		}
	case "restarting":
		a.Hist.Must(ctx, history.Event{Type: "update.installed", Subject: j.ID, Summary: "Installed " + to + "; restarting", Why: j.Trigger, Data: data})
	case "completed":
		a.Hist.Must(ctx, history.Event{Type: "update.completed", Subject: j.ID, Summary: "Updated from " + j.From + " to " + to, Why: j.Trigger, Data: data})
	case "rolledback":
		a.Hist.Must(ctx, history.Event{Type: "update.rollback", Subject: j.ID, Summary: "Rolled back from " + j.From + " to " + to, Why: j.Trigger, Data: data})
	case "failed":
		a.Hist.Must(ctx, history.Event{Type: "update.failed", Subject: j.ID, Summary: "Update to " + to + " failed", Why: j.Error, Data: data})
	}
}
