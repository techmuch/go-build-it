package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	gobuildit "github.com/techmuch/go-build-it"
	"github.com/techmuch/go-build-it/docs"
	"github.com/techmuch/go-build-it/internal/access"
	"github.com/techmuch/go-build-it/internal/agent"
	"github.com/techmuch/go-build-it/internal/api"
	"github.com/techmuch/go-build-it/internal/doctor"
	"github.com/techmuch/go-build-it/internal/harness"
	"github.com/techmuch/go-build-it/internal/herdr"
	"github.com/techmuch/go-build-it/internal/history"
	"github.com/techmuch/go-build-it/internal/metrics"
	"github.com/techmuch/go-build-it/internal/options"
	"github.com/techmuch/go-build-it/internal/pins"
	"github.com/techmuch/go-build-it/internal/projects"
	"github.com/techmuch/go-build-it/internal/questions"
	"github.com/techmuch/go-build-it/internal/queue"
	"github.com/techmuch/go-build-it/internal/rehearsals"
	"github.com/techmuch/go-build-it/internal/standards"
	"github.com/techmuch/go-build-it/internal/tailscale"
	"github.com/techmuch/go-build-it/internal/team"
	"github.com/techmuch/go-build-it/internal/update"
)

func r(method, path, name, cli, summary string, h func(*api.Ctx) (any, error), flags ...api.Flag) api.Route {
	return api.Route{Method: method, Path: path, Name: name, CLI: cli, Summary: summary, Handler: h, Flags: flags}
}

func f(name, help string) api.Flag  { return api.Flag{Name: name, Help: help} }
func fj(name, help string) api.Flag { return api.Flag{Name: name, Help: help, Type: "json"} }
func fb(name, help string) api.Flag { return api.Flag{Name: name, Help: help, Type: "bool"} }
func fi(name, help string) api.Flag { return api.Flag{Name: name, Help: help, Type: "int"} }
func notFound(err error) error      { return api.NotFound("%v", err) }
func bad(err error) error           { return api.BadRequest("%v", err) }
func atoi(s string, def int) int {
	if n, err := strconv.Atoi(s); err == nil {
		return n
	}
	return def
}

// Routes returns every API route. The CLI and MCP server are generated from this list.
func (a *App) Routes() []api.Route {
	var rs []api.Route
	add := func(x ...api.Route) { rs = append(rs, x...) }

	// System ---------------------------------------------------------------
	add(
		r("GET", "/api/system", "system.info", "system info", "Version, identity, live counts", a.systemInfo),
		r("GET", "/api/routes", "system.routes", "system routes", "Every API route with its CLI command", func(c *api.Ctx) (any, error) { return a.routeList(), nil }),
		r("GET", "/api/openapi.json", "system.openapi", "system openapi", "OpenAPI document", func(c *api.Ctx) (any, error) { return api.OpenAPI(a.routeList(), a.Version), nil }),
		r("GET", "/api/events/stream", "events.watch", "events watch", "Live event stream (Server-Sent Events)", a.stream, f("project", "only this project")),
		r("GET", "/api/settings", "settings.get", "settings get", "Current settings", func(c *api.Ctx) (any, error) { return a.Cfg, nil }),
		r("PATCH", "/api/settings", "settings.update", "settings update", "Change settings (partial JSON)", a.updateSettings, fj("data", "partial settings JSON")),
		r("GET", "/api/harnesses", "harnesses.list", "harnesses list", "Detected harnesses and capabilities", a.harnesses),
		r("GET", "/api/tailscale", "tailscale.status", "tailscale status", "Tailscale and Serve status", func(c *api.Ctx) (any, error) { return tailscale.Detect(c.R.Context(), a.Cfg.Listen), nil }),
		r("POST", "/api/tailscale/setup", "tailscale.setup", "tailscale setup", "Serve go-build-it on your tailnet", func(c *api.Ctx) (any, error) { return tailscale.Setup(c.R.Context(), a.Cfg.Listen) }),
		r("GET", "/api/dispatcher", "dispatcher.status", "dispatcher status", "Dispatcher state", a.dispatcherStatus),
		r("POST", "/api/dispatcher", "dispatcher.set", "dispatcher set", "Turn automatic dispatching on or off", a.dispatcherSet, fb("auto", "true to dispatch automatically")),
		r("POST", "/api/dispatcher/tick", "dispatcher.tick", "dispatcher tick", "Dispatch ready tasks now", func(c *api.Ctx) (any, error) { a.Sessions.Tick(c.R.Context()); return a.dispatcherStatus(c) }),
		r("GET", "/api/inbox", "inbox.list", "inbox list", "Everything waiting on a human: questions, reviews, gates, option sets", a.inbox),
		r("GET", "/api/search", "search.all", "search", "Search projects, tasks, docs and history", a.search, f("q", "search text")),
	)

	// Projects -------------------------------------------------------------
	add(
		r("GET", "/api/projects", "projects.list", "projects list", "Catalog: every project with card data", a.projectList, fb("refresh", "refresh git/forge data")),
		r("POST", "/api/projects", "projects.import", "projects import", "Import an existing git repository", a.projectImport, f("path", "repository path"), f("id", "project id"), f("purpose", "one-line purpose"), f("stage", "lifecycle stage")),
		r("GET", "/api/projects/{id}", "projects.get", "projects get", "Project overview", a.projectGet),
		r("PATCH", "/api/projects/{id}", "projects.update", "projects update", "Change name, purpose, stage or website", a.projectUpdate, f("name", ""), f("purpose", ""), f("stage", ""), f("websiteUrl", "")),
		r("DELETE", "/api/projects/{id}", "projects.remove", "projects remove", "Unregister a project (files untouched)", func(c *api.Ctx) (any, error) {
			return map[string]bool{"ok": true}, a.Projects.Remove(c.R.Context(), c.Param("id"))
		}),
		r("POST", "/api/projects/{id}/refresh", "projects.refresh", "projects refresh", "Refresh git and forge data", a.projectRefresh),
		r("GET", "/api/projects/{id}/values", "values.list", "values list", "Project values", func(c *api.Ctx) (any, error) {
			return a.Projects.Values(c.R.Context(), c.Param("id"), c.Query("all") == "true")
		}, fb("all", "include dismissed")),
		r("POST", "/api/projects/{id}/values", "values.add", "values add", "Add or propose a value", a.valueAdd, f("text", ""), f("source", "where it came from"), f("status", "accepted|proposed")),
		r("PATCH", "/api/values/{vid}", "values.set", "values set", "Accept, edit or dismiss a value", a.valueSet, f("status", "accepted|dismissed"), f("text", "")),
		r("GET", "/api/projects/{id}/problem", "problem.get", "problem get", "Latest problem definition", func(c *api.Ctx) (any, error) { return a.Projects.Problem(c.R.Context(), c.Param("id")) }),
		r("POST", "/api/projects/{id}/problem", "problem.refine", "problem refine", "Record a new problem definition version", a.problemRefine, f("text", ""), f("source", "")),
		r("GET", "/api/projects/{id}/milestones", "milestones.list", "milestones list", "Roadmap", func(c *api.Ctx) (any, error) { return a.Projects.Milestones(c.R.Context(), c.Param("id")) }),
		r("POST", "/api/projects/{id}/milestones", "milestones.save", "milestones save", "Add or update a milestone", a.milestoneSave, f("id", "existing id to update"), f("name", ""), f("scope", ""), f("status", ""), f("gateName", ""), f("gateKind", ""), fj("gateCriteria", "list"), fi("ord", "order"), f("target", "")),
		r("GET", "/api/milestones/{mid}", "milestones.get", "milestones get", "Roadmap item detail", a.milestoneGet),
		r("POST", "/api/milestones/{mid}/gate", "gates.set", "gates set", "Record a gate decision", a.gateSet, f("status", "approved|changes|pending"), f("note", "")),
		r("GET", "/api/projects/{id}/artifacts", "artifacts.list", "artifacts list", "Project documents and mockups", a.artifactList),
		r("GET", "/api/projects/{id}/artifacts/{path...}", "artifacts.get", "artifacts get", "Artifact content and revisions", a.artifactGet, f("rev", "git revision")),
		r("PUT", "/api/projects/{id}/artifacts/{path...}", "artifacts.save", "artifacts save", "Save an artifact as a new revision (Courier commits it)", a.artifactSave, f("content", ""), f("message", "commit message")),
		r("POST", "/api/projects/{id}/chat/artifact", "artifacts.chat", "artifacts chat", "Ask a persona to change an artifact", a.artifactChat, f("path", ""), f("persona", ""), f("message", ""), f("content", "current text"), fj("history", "earlier messages")),
		r("POST", "/api/diff/apply", "diff.apply", "diff apply", "Apply accepted hunks to a document", a.diffApply, f("original", ""), fj("hunks", ""), fj("accepted", "hunk indexes")),
		r("GET", "/api/projects/{id}/instructions", "standards.preview", "standards preview", "Generated agent instruction file for a project", a.instructionsPreview, f("persona", "persona id")),
		r("GET", "/api/projects/{id}/workflow", "git.workflow", "git workflow", "Effective git workflow", func(c *api.Ctx) (any, error) { return a.Courier.Workflow(c.R.Context(), c.Param("id")), nil }),
		r("POST", "/api/projects/{id}/release", "git.release", "git release", "Tag a CalVer release, create notes, publish Pages", a.release, fb("dryRun", "only show what would happen")),
		r("GET", "/api/projects/{id}/split-candidates", "tasks.split-candidates", "tasks split-candidates", "Tasks that needed repeated compactions or handoffs", func(c *api.Ctx) (any, error) {
			return a.Queue.SplitCandidates(c.R.Context(), c.Param("id"), atoi(c.Query("threshold"), 2))
		}, fi("threshold", "default 2")),
	)

	// Team & standards -----------------------------------------------------
	add(
		r("GET", "/api/roles", "roles.list", "roles list", "Roles", func(c *api.Ctx) (any, error) { return a.Team.Roles(c.R.Context()) }),
		r("POST", "/api/roles", "roles.add", "roles add", "Define a custom role", a.roleAdd, f("id", ""), f("name", ""), f("description", "")),
		r("GET", "/api/archetypes", "archetypes.list", "archetypes list", "Global persona archetypes (FR-12)", func(c *api.Ctx) (any, error) { return a.Team.Archetypes(c.R.Context()) }),
		r("GET", "/api/archetypes/{aid}", "archetypes.get", "archetypes get", "An archetype with its definition and pending member changes", a.archetypeGet),
		r("POST", "/api/archetypes", "archetypes.save", "archetypes save", "Create an archetype or save a new revision", a.archetypeSave, f("id", ""), f("name", ""), f("role", ""), f("harness", ""), f("model", ""), f("instructions", ""), fj("standards", "list")),
		r("DELETE", "/api/archetypes/{aid}", "archetypes.remove", "archetypes remove", "Remove an unused archetype", func(c *api.Ctx) (any, error) {
			if err := a.Team.RemoveArchetype(c.R.Context(), c.Param("aid")); err != nil {
				return nil, bad(err)
			}
			return map[string]bool{"ok": true}, nil
		}),
		r("GET", "/api/archetypes/{aid}/changes", "archetypes.changes", "archetypes changes", "Members behind the archetype's latest revision (FR-12b)", func(c *api.Ctx) (any, error) { return a.Team.Changes(c.R.Context(), c.Param("aid")) }),
		r("POST", "/api/archetypes/{aid}/apply", "archetypes.apply", "archetypes apply", "Apply the latest revision to one member, or to every member without overrides", a.archetypeApply, f("member", "member id (omit with --all)"), fb("all", "every member that has not overridden the changed fields")),
		r("POST", "/api/members/{mid}/restart", "members.restart", "members restart", "Stop the member's harness in herdr; --fresh starts a new conversation next task", func(c *api.Ctx) (any, error) {
			var b struct{ Fresh bool }
			_ = c.Bind(&b)
			if err := a.Sessions.RestartMember(c.R.Context(), c.Param("mid"), b.Fresh); err != nil {
				return nil, bad(err)
			}
			return map[string]bool{"ok": true}, nil
		}, fb("fresh", "start a new conversation next task")),
		r("GET", "/api/members/{mid}/terminal", "members.terminal", "members terminal", "Live terminal of the member's herdr pane (WebSocket; mode=observe|control)", a.memberTerminal, f("mode", "observe|control"), fi("cols", ""), fi("rows", "")),
		r("POST", "/api/members/{mid}/focus", "members.focus", "members focus", "Focus the member's pane in the gbi herdr session", a.memberFocus),
		r("POST", "/api/members/{mid}/promote", "members.promote", "members promote", "Promote tuning text into the member's archetype (FR-12c)", a.memberPromote, f("text", "tuning lines to promote")),
		r("GET", "/api/projects/{id}/members", "members.list", "members list", "A project's team (FR-11)", func(c *api.Ctx) (any, error) { return a.Team.Members(c.R.Context(), c.Param("id")) }),
		r("POST", "/api/projects/{id}/members", "members.add", "members add", "Add an AI member from an archetype, or a human with a role", a.memberAdd, f("archetype", "archetype id (AI member)"), f("name", "display name"), f("kind", "ai|human"), f("login", "human: Tailscale login"), f("role", "human: role")),
		r("POST", "/api/projects/{id}/members/starter", "members.starter", "members starter", "Give a project with no AI members the starter team", func(c *api.Ctx) (any, error) { return a.Team.EnsureStarterTeam(c.R.Context(), c.Param("id")) }),
		r("GET", "/api/members", "members.all", "members all", "Every member across projects", func(c *api.Ctx) (any, error) { return a.Team.Members(c.R.Context(), "") }),
		r("GET", "/api/members/{mid}", "members.get", "members get", "A member with its effective definition and archetype", a.memberGet),
		r("PATCH", "/api/members/{mid}", "members.tune", "members tune", "Tune a member for its project", a.memberTune, f("name", ""), f("role", ""), f("tuning", "project-specific instructions"), fb("replaceInstructions", "use tuning instead of the archetype instructions"), f("harness", "override; empty clears"), f("model", "override; empty clears"), fj("standards", "extra standards"), f("login", "")),
		r("DELETE", "/api/members/{mid}", "members.remove", "members remove", "Remove a member", func(c *api.Ctx) (any, error) {
			return map[string]bool{"ok": true}, a.Team.Remove(c.R.Context(), c.Param("mid"))
		}),
		r("POST", "/api/members/{mid}/chat", "members.chat", "members chat", "Persona coach: develop a member's tuning or an archetype with an agent", a.memberChat, f("coach", "persona doing the coaching"), f("message", ""), f("content", "current definition"), fj("history", "")),
		r("GET", "/api/standards", "standards.list", "standards list", "Global standards", func(c *api.Ctx) (any, error) { return a.Standards.List(c.R.Context()) }),
		r("GET", "/api/standards/{sid}", "standards.get", "standards get", "One standard", func(c *api.Ctx) (any, error) {
			s, err := a.Standards.Get(c.R.Context(), c.Param("sid"))
			if err != nil {
				return nil, notFound(err)
			}
			return s, nil
		}),
		r("PUT", "/api/standards/{sid}", "standards.save", "standards save", "Create or update a standard", a.standardSave, f("title", ""), f("body", "Markdown"), f("why", "")),
		r("DELETE", "/api/standards/{sid}", "standards.remove", "standards remove", "Remove a standard", func(c *api.Ctx) (any, error) {
			return map[string]bool{"ok": true}, a.Standards.Remove(c.R.Context(), c.Param("sid"))
		}),
		r("POST", "/api/standards/{sid}/overrides", "standards.override", "standards override", "Override a standard for a project", a.standardOverride, f("project", ""), f("body", ""), f("reason", "")),
		r("POST", "/api/standards/{sid}/chat", "standards.chat", "standards chat", "Ask a persona to change a standard", a.standardChat, f("persona", ""), f("message", ""), f("content", ""), fj("history", "")),
	)

	// Work -----------------------------------------------------------------
	add(
		r("GET", "/api/tasks", "tasks.list", "tasks list", "Task queue", a.taskList, f("project", ""), f("status", ""), f("role", ""), f("milestone", "")),
		r("POST", "/api/tasks", "tasks.add", "tasks add", "Queue a task", a.taskAdd, f("project", ""), f("title", ""), f("description", ""), f("role", ""), f("assignee", ""), fi("priority", "1 highest"), fj("deps", "task ids"), fj("reqs", "requirement ids"), f("milestone", "")),
		r("GET", "/api/tasks/{tid}", "tasks.get", "tasks get", "Task detail", a.taskGet),
		r("PATCH", "/api/tasks/{tid}", "tasks.update", "tasks update", "Edit a task", a.taskUpdate, f("title", ""), f("description", ""), f("role", ""), f("assignee", ""), fi("priority", ""), fj("deps", ""), fj("reqs", ""), f("milestone", ""), f("note", "append to context note")),
		r("POST", "/api/tasks/{tid}/status", "tasks.status", "tasks status", "Change task status", a.taskStatus, f("status", ""), f("why", "")),
		r("POST", "/api/tasks/{tid}/run", "tasks.run", "tasks run", "Start a session for a task now", a.taskRun, f("member", "persona id")),
		r("POST", "/api/tasks/{tid}/merge", "git.merge", "git merge", "Merge an approved task (Courier)", func(c *api.Ctx) (any, error) {
			return map[string]bool{"ok": true}, a.Courier.Merge(c.R.Context(), c.Param("tid"))
		}),
		r("POST", "/api/git/ensure-pushed", "git.ensure-pushed", "git ensure-pushed", "Push any unpushed task branches", func(c *api.Ctx) (any, error) {
			n, err := a.Courier.EnsurePushed(c.R.Context())
			return map[string]int{"pushed": n}, err
		}),
		r("GET", "/api/sessions", "sessions.list", "sessions list", "Sessions", func(c *api.Ctx) (any, error) {
			return a.Sessions.List(c.R.Context(), c.Query("project"), c.Query("task"), c.Query("live") == "true")
		}, f("project", ""), f("task", ""), fb("live", "only live sessions (starting, running, blocked, rate-limited)")),
		r("GET", "/api/sessions/{sid}", "sessions.get", "sessions get", "Session detail", func(c *api.Ctx) (any, error) {
			s, err := a.Sessions.Get(c.R.Context(), c.Param("sid"))
			if err != nil {
				return nil, notFound(err)
			}
			return s, nil
		}),
		r("GET", "/api/sessions/{sid}/output", "sessions.output", "sessions output", "Structured session output read from the transcript (the terminal streams from herdr)", func(c *api.Ctx) (any, error) {
			return a.Sessions.Output(c.R.Context(), c.Param("sid"), atoi(c.Query("after"), 0), c.Query("view"))
		}, fi("after", "sequence number"), f("view", "structured|terminal")),
		r("POST", "/api/sessions/{sid}/input", "sessions.send", "sessions send", "Send input to a live session", a.sessionSend, f("text", "")),
		r("POST", "/api/sessions/{sid}/cancel", "sessions.cancel", "sessions cancel", "Cancel a live session", func(c *api.Ctx) (any, error) {
			return map[string]bool{"ok": true}, a.Sessions.Cancel(c.R.Context(), c.Param("sid"))
		}),
	)

	// Human interaction ----------------------------------------------------
	add(
		r("GET", "/api/questions", "questions.list", "questions list", "Questions (open, blocking first)", func(c *api.Ctx) (any, error) {
			return a.Questions.List(c.R.Context(), c.Query("project"), c.Query("status"))
		}, f("project", ""), f("status", "open|answered|all")),
		r("GET", "/api/questions/{qid}", "questions.get", "questions get", "One question", func(c *api.Ctx) (any, error) {
			q, err := a.Questions.Get(c.R.Context(), c.Param("qid"))
			if err != nil {
				return nil, notFound(err)
			}
			return q, nil
		}),
		r("POST", "/api/questions", "questions.ask", "questions ask", "Ask the humans a question", a.questionAsk, f("project", ""), f("task", ""), f("session", ""), f("title", ""), f("why", ""), fb("blocking", ""), f("assumption", "required if not blocking"), f("recommendation", ""), fj("options", "[{key,title,detail,recommended}]"), fj("attachments", "[{name,kind,path,url,body}]"), fj("reqs", "")),
		r("POST", "/api/questions/{qid}/answer", "questions.answer", "questions answer", "Answer a question", a.questionAnswer, f("answer", ""), f("choice", "option key"), f("proposeValue", "value text to propose")),
		r("POST", "/api/questions/{qid}/skip", "questions.skip", "questions skip", "Skip a question", func(c *api.Ctx) (any, error) {
			var b struct{ Why string }
			_ = c.Bind(&b)
			return a.Questions.Skip(c.R.Context(), c.Param("qid"), b.Why)
		}, f("why", "")),
		r("POST", "/api/questions/{qid}/confirm", "questions.confirm", "questions confirm", "Confirm a non-blocking assumption", func(c *api.Ctx) (any, error) { return a.Questions.Confirm(c.R.Context(), c.Param("qid")) }),
		r("GET", "/api/reviews", "reviews.list", "reviews list", "Reviews (pending first)", func(c *api.Ctx) (any, error) {
			return a.Reviews.List(c.R.Context(), c.Query("project"), c.Query("status"))
		}, f("project", ""), f("status", "")),
		r("GET", "/api/reviews/{rid}", "reviews.get", "reviews get", "Review with assessment", func(c *api.Ctx) (any, error) {
			rv, err := a.Reviews.Get(c.R.Context(), c.Param("rid"))
			if err != nil {
				return nil, notFound(err)
			}
			return rv, nil
		}),
		r("POST", "/api/reviews", "reviews.open", "reviews open", "Open a review for a task", func(c *api.Ctx) (any, error) {
			var b struct{ Task string }
			if err := c.Bind(&b); err != nil {
				return nil, err
			}
			return a.Reviews.Open(c.R.Context(), b.Task)
		}, f("task", "")),
		r("POST", "/api/reviews/{rid}/reassess", "reviews.reassess", "reviews reassess", "Rerun the assessment", func(c *api.Ctx) (any, error) { return a.Reviews.Reassess(c.R.Context(), c.Param("rid")) }),
		r("POST", "/api/reviews/{rid}/verdict", "reviews.verdict", "reviews verdict", "Approve, request changes or reject", a.reviewVerdict, f("verdict", "approve|changes|reject"), f("comment", "")),
		r("GET", "/api/rehearsals", "rehearsals.list", "rehearsals list", "Dress rehearsals", func(c *api.Ctx) (any, error) { return a.Rehearsals.List(c.R.Context(), c.Query("project")) }, f("project", "")),
		r("GET", "/api/rehearsals/{gid}", "rehearsals.get", "rehearsals get", "Rehearsal with findings", func(c *api.Ctx) (any, error) {
			g, err := a.Rehearsals.Get(c.R.Context(), c.Param("gid"))
			if err != nil {
				return nil, notFound(err)
			}
			return g, nil
		}),
		r("POST", "/api/rehearsals", "rehearsals.create", "rehearsals create", "Start a dress rehearsal", a.rehearsalCreate, f("project", ""), f("milestone", ""), f("title", ""), f("kind", "tabletop|uat"), fj("steps", "[{title,scenario,reqs,screen}]")),
		r("POST", "/api/rehearsals/{gid}/findings", "rehearsals.finding", "rehearsals finding", "Add feedback on a step", a.rehearsalFinding, fi("step", "0-based"), f("text", "")),
		r("POST", "/api/rehearsals/{gid}/advance", "rehearsals.advance", "rehearsals advance", "Mark the current step and move", a.rehearsalAdvance, fi("step", "target step"), f("status", "ok|notes")),
		r("POST", "/api/rehearsals/{gid}/decide", "rehearsals.decide", "rehearsals decide", "Approve the gate or request changes", a.rehearsalDecide, f("decision", "approve|changes"), f("note", "")),
		r("POST", "/api/findings/{fid}/change-request", "findings.change-request", "findings change-request", "Turn a finding into a task", a.changeRequest, fb("now", "do it now (priority 1)")),
		r("GET", "/api/options", "options.list", "options list", "Option sets", func(c *api.Ctx) (any, error) { return a.Options.List(c.R.Context(), c.Query("project")) }, f("project", "")),
		r("GET", "/api/options/{oid}", "options.get", "options get", "One option set", func(c *api.Ctx) (any, error) {
			o, err := a.Options.Get(c.R.Context(), c.Param("oid"))
			if err != nil {
				return nil, notFound(err)
			}
			return o, nil
		}),
		r("POST", "/api/options", "options.create", "options create", "Present an option set", a.optionsCreate, f("project", ""), f("title", ""), fj("options", "[{key,title,summary,pros,cons,mockup,aspects}]"), fj("valueLens", "")),
		r("POST", "/api/options/{oid}/choose", "options.choose", "options choose", "Choose options or aspects, with a reason", a.optionsChoose, fj("picks", "[{option,aspect}]"), f("reason", ""), f("value", "value to propose")),
	)

	// History, insights, docs, updates ---------------------------------------
	add(
		r("GET", "/api/history", "history.list", "history list", "Searchable event log", a.historyList, f("project", ""), f("type", "prefix"), f("actor", ""), f("q", "text"), f("subject", ""), f("since", "RFC3339"), fi("limit", ""), fi("before", "event id")),
		r("GET", "/api/history/{eid}", "history.get", "history get", "Event detail with related events", a.historyGet),
		r("GET", "/api/metrics", "metrics.get", "metrics get", "Performance metrics", func(c *api.Ctx) (any, error) {
			return metrics.Compute(c.R.Context(), a.DB, c.Query("project"), atoi(c.Query("days"), 7))
		}, f("project", ""), fi("days", "default 7")),
		r("GET", "/api/metrics/compare", "metrics.compare", "metrics compare", "Smart handoffs vs compaction (graduation evidence)", func(c *api.Ctx) (any, error) { return metrics.Compare(c.R.Context(), a.DB, atoi(c.Query("days"), 30)) }, fi("days", "default 30")),
		r("GET", "/api/herdr", "herdr.status", "herdr status", "The supervised gbi herdr session (FR-07)", func(c *api.Ctx) (any, error) { return a.Herdr.Status(), nil }),
		r("GET", "/api/doctor", "doctor.run", "herdr doctor", "Run the doctor through the server: static, live, harness and (opt-in) prompt tiers (FR-07a–FR-07d). `gbi doctor` runs it without the server.", a.doctorRun, f("through", "last tier: static, live (default) or harness"), f("harness", "only this harness"), fb("prompt", "also send one short prompt per harness: uses tokens")),
		r("POST", "/api/doctor/fix", "doctor.fix", "herdr fix", "Apply one automatic doctor fix, then run the static and live tiers again", a.doctorFix, f("action", "fix action from a doctor result")),
		r("GET", "/api/doctor/report", "doctor.report", "herdr report", "The doctor's Markdown report (secrets removed) to paste into an AI chat or an issue", a.doctorReport),
		r("GET", "/api/pins", "pins.list", "pins list", "Comments pinned to spots on attachments and mockups", func(c *api.Ctx) (any, error) {
			return a.Pins.List(c.R.Context(), c.Query("subject"), c.Query("item"))
		}, f("subject", "what the viewer was opened for, e.g. Q-34"), f("item", "attachment URL or name (empty: all items)")),
		r("POST", "/api/pins", "pins.add", "pins add", "Pin a comment to a spot on an attachment or mockup", a.pinAdd, f("subject", ""), f("item", ""), f("projectId", ""), fj("x", "percent of width"), fj("y", "percent of height"), f("text", "")),
		r("DELETE", "/api/pins/{pid}", "pins.remove", "pins remove", "Remove a pinned comment", func(c *api.Ctx) (any, error) {
			id, _ := strconv.ParseInt(c.Param("pid"), 10, 64)
			if err := a.Pins.Remove(c.R.Context(), id); err != nil {
				return nil, notFound(err)
			}
			return map[string]bool{"ok": true}, nil
		}),
		r("GET", "/api/proposals", "proposals.list", "proposals list", "Improvement proposals", func(c *api.Ctx) (any, error) { return a.Proposals.List(c.R.Context()) }),
		r("POST", "/api/proposals", "proposals.create", "proposals create", "Propose an improvement (with evidence)", a.proposalCreate, f("title", ""), f("body", ""), f("evidence", "")),
		r("POST", "/api/proposals/{pid}/decide", "proposals.decide", "proposals decide", "Accept or reject a proposal", a.proposalDecide, f("decision", "accepted|rejected"), f("note", "")),
		r("GET", "/api/docs", "docs.list", "docs list", "Documentation pages", func(c *api.Ctx) (any, error) { return docs.Pages(), nil }),
		r("GET", "/api/docs/search", "docs.search", "docs search", "Search the docs", func(c *api.Ctx) (any, error) { return docs.Search(c.Query("q")), nil }, f("q", "")),
		r("GET", "/api/docs/{slug}", "docs.get", "docs get", "One documentation page", func(c *api.Ctx) (any, error) {
			p, ok := docs.Get(c.Param("slug"))
			if !ok {
				return nil, api.NotFound("no page %q", c.Param("slug"))
			}
			return p, nil
		}),
		r("GET", "/api/changelog", "changelog.list", "changelog", "go-build-it's change log, newest first, with the installed version", func(c *api.Ctx) (any, error) {
			return map[string]any{"current": a.Version, "entries": update.ParseChangelog(gobuildit.Changelog)}, nil
		}),
		r("GET", "/api/updates", "updates.check", "updates check", "Check for a newer release and list every newer release's notes", func(c *api.Ctx) (any, error) {
			rel, err := a.checkUpdates(c.R.Context())
			if err != nil {
				return map[string]any{"current": a.Version, "error": err.Error(), "previous": rel.Previous, "checkedAt": rel.CheckedAt, "releases": []any{}}, nil
			}
			return rel, nil
		}),
		r("POST", "/api/updates/install", "updates.install", "updates install", "Update to the latest release now; follow it with `updates job`", a.updateInstall),
		r("GET", "/api/updates/job", "updates.job", "updates job", "The running or last update, with its steps", func(c *api.Ctx) (any, error) {
			j, ok := a.Updates.Current()
			if !ok {
				return map[string]any{"job": nil}, nil
			}
			return map[string]any{"job": j}, nil
		}),
		r("POST", "/api/updates/rollback", "updates.rollback", "updates rollback", "Restore the previous version and restart", func(c *api.Ctx) (any, error) {
			j, err := a.Updates.StartRollback(c.Actor.Name)
			if errors.Is(err, update.ErrBusy) {
				return nil, api.Conflict("%s", err.Error())
			}
			if err != nil {
				return nil, bad(err)
			}
			return map[string]any{"job": j}, nil
		}),
	)
	return rs
}

func (a *App) routeList() []api.Route { return a.Routes() }

// System handlers ----------------------------------------------------------

func (a *App) systemInfo(c *api.Ctx) (any, error) {
	ctx := c.R.Context()
	live, _ := a.Sessions.List(ctx, "", "", true)
	return map[string]any{
		"version": a.Version, "actor": c.Actor, "listen": a.Cfg.Listen, "dataDir": a.Cfg.Dir(),
		"liveSessions": len(live), "autoDispatch": a.Sessions.Auto(), "owner": a.Cfg.DefaultUser,
		"experimental": a.Cfg.Experimental, "isOwner": a.Cfg.IsOwner(c.Actor.Name, c.Actor.Login), "member": a.memberFor(c.R.Context(), c.Actor),
		"update": a.updateSummary(),
	}, nil
}

// updateSummary is what the sidebar shows next to the version (FR-06b): the last check's
// result and whether an update is running. It never calls GitHub itself.
func (a *App) updateSummary() map[string]any {
	out := map[string]any{"updating": a.Updates.Busy()}
	if r := a.lastCheck.Load(); r != nil {
		out["latest"], out["newer"], out["checkedAt"] = r.Tag, r.Newer && !update.SameVersion(r.Tag, a.Version), r.CheckedAt
	}
	return out
}

// memberFor maps a Tailscale login or local name to a team member (P9-08).
func (a *App) memberFor(ctx context.Context, actor access.Actor) string {
	ms, _ := a.Team.Members(ctx, "")
	for _, m := range ms {
		if m.Kind == "human" && ((actor.Login != "" && m.Login == actor.Login) || m.ID == actor.Name) {
			return m.ID
		}
	}
	return ""
}

func (a *App) updateSettings(c *api.Ctx) (any, error) {
	var raw map[string]json.RawMessage
	if err := c.Bind(&raw); err != nil {
		return nil, err
	}
	if d, ok := raw["data"]; ok && len(raw) == 1 {
		_ = json.Unmarshal(d, &raw)
	}
	cur, _ := json.Marshal(a.Cfg)
	var merged map[string]json.RawMessage
	_ = json.Unmarshal(cur, &merged)
	for k, v := range raw {
		if k == "listen" || k == "defaultUser" {
			continue // changed only in config.toml (needs restart / affects identity)
		}
		merged[k] = v
	}
	b, _ := json.Marshal(merged)
	if err := json.Unmarshal(b, a.Cfg); err != nil {
		return nil, api.BadRequest("invalid settings: %v", err)
	}
	if err := a.Cfg.Save(); err != nil {
		return nil, err
	}
	keys := make([]string, 0, len(raw))
	for k := range raw {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	a.Hist.Must(c.R.Context(), history.Event{Type: "settings.updated", Summary: "Settings changed: " + strings.Join(keys, ", "), Data: history.Data(raw)})
	if _, ok := raw["autoDispatch"]; ok {
		a.Sessions.SetAuto(a.Cfg.AutoDispatch)
	}
	return a.Cfg, nil
}

func (a *App) harnesses(c *api.Ctx) (any, error) {
	type row struct {
		harness.Detection
		Enabled bool   `json:"enabled"`
		Model   string `json:"model"`
	}
	var out []row
	for _, n := range harness.Names() {
		ad, _ := harness.Get(n)
		hc := a.Cfg.Harnesses[n]
		if n == "fake" {
			continue
		}
		out = append(out, row{Detection: harness.Detect(c.R.Context(), ad, hc.Binary), Enabled: hc.Enabled, Model: hc.Model})
	}
	return out, nil
}

func (a *App) dispatcherStatus(c *api.Ctx) (any, error) {
	return map[string]any{"auto": a.Sessions.Auto(), "live": a.Sessions.LiveCount(), "maxConcurrent": a.Cfg.MaxConcurrent}, nil
}

func (a *App) dispatcherSet(c *api.Ctx) (any, error) {
	var b struct{ Auto bool }
	if err := c.Bind(&b); err != nil {
		return nil, err
	}
	a.Cfg.AutoDispatch = b.Auto
	_ = a.Cfg.Save()
	a.Sessions.SetAuto(b.Auto)
	a.Hist.Must(c.R.Context(), history.Event{Type: "dispatcher.set", Summary: fmt.Sprintf("Automatic dispatch %s", map[bool]string{true: "on", false: "off"}[b.Auto])})
	return a.dispatcherStatus(c)
}

// stream sends live events as Server-Sent Events (FR-36).
func (a *App) stream(c *api.Ctx) (any, error) {
	w := c.W
	fl, ok := w.(http.Flusher)
	if !ok {
		return nil, errors.New("streaming unsupported")
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, ": connected\n\n")
	fl.Flush()
	ch, cancel := a.Hist.Subscribe()
	defer cancel()
	project := c.Query("project")
	ping := time.NewTicker(25 * time.Second)
	defer ping.Stop()
	for {
		select {
		case <-c.R.Context().Done():
			return api.Raw{}, nil
		case <-ping.C:
			fmt.Fprint(w, ": ping\n\n")
			fl.Flush()
		case e := <-ch:
			if project != "" && e.ProjectID != project {
				continue
			}
			b, _ := json.Marshal(e)
			fmt.Fprintf(w, "data: %s\n\n", b)
			fl.Flush()
		}
	}
}

// InboxItem is something waiting on a human (FR-104, unified "needs you").
type InboxItem struct {
	Kind      string `json:"kind"` // question | review | gate | options | value
	ID        string `json:"id"`
	ProjectID string `json:"projectId"`
	Title     string `json:"title"`
	Detail    string `json:"detail"`
	Blocking  bool   `json:"blocking"`
	From      string `json:"from"`
	CreatedAt string `json:"createdAt"`
	Assignee  string `json:"assignee,omitempty"`
}

func (a *App) inboxItems(ctx context.Context, project string) []InboxItem {
	out := []InboxItem{}
	if qs, err := a.Questions.List(ctx, project, "open"); err == nil {
		for _, q := range qs {
			d := q.Why
			if !q.Blocking {
				d = "Assumed: " + q.Assumption
			}
			out = append(out, InboxItem{Kind: "question", ID: q.ID, ProjectID: q.ProjectID, Title: q.Title, Detail: d, Blocking: q.Blocking, From: q.Asker, CreatedAt: q.CreatedAt})
		}
	}
	// Members herdr reports blocked need the human in their terminal (FR-68).
	if ss, err := a.Sessions.List(ctx, project, "", true); err == nil {
		for _, se := range ss {
			if se.Status != "blocked" {
				continue
			}
			name := se.MemberID
			if m, err := a.Team.Member(ctx, se.MemberID); err == nil {
				name = m.Name
			}
			out = append(out, InboxItem{Kind: "member", ID: se.ID, ProjectID: se.ProjectID, Title: name + " is waiting for you (" + se.TaskID + ")",
				Detail: firstNonEmptyStr(se.ExitReason, "Open the live terminal to answer"), Blocking: true, From: se.MemberID, CreatedAt: se.StartedAt})
		}
	}
	if rs, err := a.Reviews.List(ctx, project, "pending"); err == nil {
		for _, rv := range rs {
			d := rv.Assessment.Summary
			if rv.Assignee != "" {
				d = "Assigned to " + rv.Assignee + ". " + d
			}
			out = append(out, InboxItem{Kind: "review", ID: rv.ID, ProjectID: rv.ProjectID, Title: "Review " + rv.TaskID + " " + rv.TaskTitle, Detail: d, From: "Lens", CreatedAt: rv.CreatedAt, Assignee: rv.Assignee})
		}
	}
	if gs, err := a.Rehearsals.List(ctx, project); err == nil {
		for _, g := range gs {
			if g.Status == "open" {
				out = append(out, InboxItem{Kind: "gate", ID: g.ID, ProjectID: g.ProjectID, Title: "Gate " + g.ID + " · " + g.Title, Detail: fmt.Sprintf("%s rehearsal, %d steps", g.Kind, len(g.Steps)), From: g.Host, CreatedAt: g.CreatedAt})
			}
		}
	}
	if os, err := a.Options.List(ctx, project); err == nil {
		for _, o := range os {
			if o.Status == "open" {
				out = append(out, InboxItem{Kind: "options", ID: o.ID, ProjectID: o.ProjectID, Title: o.ID + " · " + o.Title, Detail: fmt.Sprintf("%d options", len(o.Options)), From: o.GeneratedBy, CreatedAt: o.CreatedAt})
			}
		}
	}
	ps, _ := a.Projects.List(ctx)
	for _, p := range ps {
		if project != "" && p.ID != project {
			continue
		}
		vals, _ := a.Projects.Values(ctx, p.ID, false)
		for _, v := range vals {
			if v.Status == "proposed" {
				out = append(out, InboxItem{Kind: "value", ID: strconv.FormatInt(v.ID, 10), ProjectID: p.ID, Title: "Proposed value: " + v.Text, Detail: v.Source, CreatedAt: v.CreatedAt})
			}
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Blocking != out[j].Blocking {
			return out[i].Blocking
		}
		return out[i].CreatedAt < out[j].CreatedAt
	})
	return out
}

func (a *App) inbox(c *api.Ctx) (any, error) {
	return a.inboxItems(c.R.Context(), c.Query("project")), nil
}

func (a *App) search(c *api.Ctx) (any, error) {
	ctx := c.R.Context()
	q := strings.ToLower(strings.TrimSpace(c.Query("q")))
	type hit struct {
		Kind  string `json:"kind"`
		ID    string `json:"id"`
		Title string `json:"title"`
		Link  string `json:"link"`
		CLI   string `json:"cli"`
	}
	out := []hit{}
	if q == "" {
		return out, nil
	}
	ps, _ := a.Projects.List(ctx)
	for _, p := range ps {
		if strings.Contains(strings.ToLower(p.ID+" "+p.Purpose), q) {
			out = append(out, hit{"project", p.ID, p.Name, "/projects/" + p.ID, "gbi projects get " + p.ID})
		}
	}
	ts, _ := a.Queue.List(ctx, queue.Filter{})
	for _, t := range ts {
		if strings.Contains(strings.ToLower(t.ID+" "+t.Title), q) && len(out) < 40 {
			out = append(out, hit{"task", t.ID, t.ID + " " + t.Title, "/queue?task=" + t.ID, "gbi tasks get " + t.ID})
		}
	}
	for _, h := range docs.Search(q) {
		out = append(out, hit{"doc", h.Page.Slug, h.Page.Title, "/docs/" + h.Page.Slug, "gbi docs get " + h.Page.Slug})
	}
	for _, rt := range a.routeList() {
		if strings.Contains(strings.ToLower(rt.Summary+" "+rt.CLI), q) && len(out) < 60 {
			out = append(out, hit{"command", rt.Name, rt.Summary, "", "gbi " + rt.CLI})
		}
	}
	return out, nil
}

// Project handlers -----------------------------------------------------------

// Card is a catalog entry (FR-41).
type Card struct {
	projects.Project
	Live        projects.Live `json:"live"`
	NeedsYou    int           `json:"needsYou"`
	Blocking    int           `json:"blocking"`
	Running     int           `json:"running"`
	Queued      int           `json:"queued"`
	Values      []string      `json:"values"`
	StatusLine  string        `json:"statusLine"`
	Milestone   string        `json:"milestone"`
	TeamMembers []string      `json:"teamMembers"`
	Team        []CardMember  `json:"team"`
}

func (a *App) card(ctx context.Context, p projects.Project, refresh bool) Card {
	maxAge := 10 * time.Minute
	if refresh {
		maxAge = 0
	}
	c := Card{Project: p, Live: a.Projects.Live(ctx, p, maxAge)}
	items := a.inboxItems(ctx, p.ID)
	c.NeedsYou = len(items)
	for _, it := range items {
		if it.Blocking {
			c.Blocking++
		}
	}
	ts, _ := a.Queue.List(ctx, queue.Filter{ProjectID: p.ID})
	for _, t := range ts {
		switch t.Status {
		case queue.Running:
			c.Running++
		case queue.Queued:
			c.Queued++
		}
	}
	c.Values = a.valueTexts(ctx, p.ID)
	ms, _ := a.Projects.Milestones(ctx, p.ID)
	for _, m := range ms {
		if m.Status != "done" {
			c.Milestone = m.Name
			break
		}
	}
	c.StatusLine = statusLine(c)
	state := map[string]string{}
	if ss, err := a.Sessions.List(ctx, p.ID, "", true); err == nil {
		for _, se := range ss {
			st := "working"
			if se.Status == "blocked" {
				st = "blocked"
			} else if se.Status == "rate_limited" {
				st = "rate-limited"
			}
			state[se.MemberID] = st
		}
	}
	mem, _ := a.Team.Members(ctx, p.ID)
	for _, m := range mem {
		c.TeamMembers = append(c.TeamMembers, m.Name)
		st := state[m.ID]
		if st == "" && m.Kind == "ai" && m.PaneID != "" {
			st = "idle"
		}
		c.Team = append(c.Team, CardMember{ID: m.ID, Name: m.Name, Role: m.Role, Kind: m.Kind, Tuned: m.Tuned(), State: st})
	}
	return c
}

// CardMember is a team member as shown on a project card (FR-41, FR-12a).
type CardMember struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Role  string `json:"role"`
	Kind  string `json:"kind"`
	Tuned bool   `json:"tuned"`
	State string `json:"state,omitempty"` // herdr state when the member runs (P13)
}

func statusLine(c Card) string {
	switch {
	case c.Blocking > 0:
		return fmt.Sprintf("%d blocking question(s) are pausing work", c.Blocking)
	case c.Running > 0:
		return fmt.Sprintf("%d session(s) working", c.Running)
	case c.NeedsYou > 0:
		return fmt.Sprintf("%d item(s) waiting on you", c.NeedsYou)
	case c.Queued > 0:
		return fmt.Sprintf("%d task(s) queued", c.Queued)
	}
	return "Idle"
}

func (a *App) projectList(c *api.Ctx) (any, error) {
	ctx := c.R.Context()
	ps, err := a.Projects.List(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Card, len(ps))
	done := make(chan struct{}, len(ps))
	for i, p := range ps {
		go func(i int, p projects.Project) {
			out[i] = a.card(ctx, p, c.Query("refresh") == "true")
			done <- struct{}{}
		}(i, p)
	}
	for range ps {
		<-done
	}
	sort.SliceStable(out, func(i, j int) bool {
		if (out[i].NeedsYou > 0 || out[i].Live.Unpushed.Any()) != (out[j].NeedsYou > 0 || out[j].Live.Unpushed.Any()) {
			return out[i].NeedsYou > 0 || out[i].Live.Unpushed.Any()
		}
		return out[i].Name < out[j].Name
	})
	return out, nil
}

func (a *App) projectImport(c *api.Ctx) (any, error) {
	var b struct{ Path, ID, Purpose, Stage string }
	if err := c.Bind(&b); err != nil {
		return nil, err
	}
	p, err := a.Projects.Import(c.R.Context(), b.Path, b.ID, b.Purpose, b.Stage)
	if err != nil {
		return nil, bad(err)
	}
	if _, err := a.Team.EnsureStarterTeam(c.R.Context(), p.ID); err != nil {
		a.Log.Warn("could not create the starter team", "project", p.ID, "err", err)
	}
	return p, nil
}

func (a *App) projectGet(c *api.Ctx) (any, error) {
	ctx := c.R.Context()
	p, err := a.Projects.Get(ctx, c.Param("id"))
	if err != nil {
		return nil, notFound(err)
	}
	card := a.card(ctx, p, false)
	vals, _ := a.Projects.Values(ctx, p.ID, false)
	prob, _ := a.Projects.Problem(ctx, p.ID)
	ms, _ := a.Projects.Milestones(ctx, p.ID)
	type mprog struct {
		projects.Milestone
		Done  int `json:"done"`
		Total int `json:"total"`
	}
	var mp []mprog
	for _, m := range ms {
		ts, _ := a.Queue.List(ctx, queue.Filter{MilestoneID: m.ID})
		x := mprog{Milestone: m, Total: len(ts)}
		for _, t := range ts {
			if t.Status == queue.Done {
				x.Done++
			}
		}
		mp = append(mp, x)
	}
	live, _ := a.Sessions.List(ctx, p.ID, "", true)
	return map[string]any{"project": card, "values": vals, "problem": prob, "milestones": mp, "sessions": live,
		"needsYou": a.inboxItems(ctx, p.ID), "workflow": a.Courier.Workflow(ctx, p.ID)}, nil
}

func (a *App) projectUpdate(c *api.Ctx) (any, error) {
	var b struct{ Name, Purpose, Stage, WebsiteURL string }
	if err := c.Bind(&b); err != nil {
		return nil, err
	}
	p, err := a.Projects.Update(c.R.Context(), c.Param("id"), b.Name, b.Purpose, b.Stage, b.WebsiteURL, "")
	if errors.Is(err, projects.ErrNotFound) {
		return nil, notFound(err)
	}
	if err != nil {
		return nil, bad(err)
	}
	return p, nil
}

func (a *App) projectRefresh(c *api.Ctx) (any, error) {
	p, err := a.Projects.Get(c.R.Context(), c.Param("id"))
	if err != nil {
		return nil, notFound(err)
	}
	return a.Projects.Refresh(c.R.Context(), p), nil
}

func (a *App) doctorOptions(c *api.Ctx) doctor.Options {
	return doctor.Options{Through: doctor.Tier(c.Query("through")), Harness: c.Query("harness"), Prompt: c.Query("prompt") == "true"}
}

// doctorRun runs the doctor; one run at a time (it opens panes in the gbi session).
func (a *App) doctorRun(c *api.Ctx) (any, error) {
	a.doctorMu.Lock()
	defer a.doctorMu.Unlock()
	rep := a.Doctor.Run(c.R.Context(), a.doctorOptions(c))
	return map[string]any{"report": rep, "status": a.Herdr.Status(), "installHint": herdr.InstallHint()}, nil
}

func (a *App) doctorFix(c *api.Ctx) (any, error) {
	var b struct{ Action string }
	if err := c.Bind(&b); err != nil {
		return nil, err
	}
	a.doctorMu.Lock()
	err := a.Doctor.Apply(c.R.Context(), b.Action)
	a.doctorMu.Unlock()
	if err != nil {
		return nil, bad(err)
	}
	a.Hist.Must(c.R.Context(), history.Event{Type: "doctor.fix", Subject: b.Action, Summary: "Applied doctor fix " + b.Action})
	return a.doctorRun(c)
}

func (a *App) doctorReport(c *api.Ctx) (any, error) {
	a.doctorMu.Lock()
	defer a.doctorMu.Unlock()
	rep := a.Doctor.Run(c.R.Context(), a.doctorOptions(c))
	return map[string]any{"markdown": a.Doctor.Markdown(rep), "exit": rep.Exit}, nil
}

func (a *App) pinAdd(c *api.Ctx) (any, error) {
	var p pins.Pin
	if err := c.Bind(&p); err != nil {
		return nil, err
	}
	out, err := a.Pins.Add(c.R.Context(), p)
	if err != nil {
		return nil, bad(err)
	}
	return out, nil
}

func (a *App) valueAdd(c *api.Ctx) (any, error) {
	var b struct{ Text, Source, Status string }
	if err := c.Bind(&b); err != nil {
		return nil, err
	}
	if strings.TrimSpace(b.Text) == "" {
		return nil, api.BadRequest("text is required")
	}
	v, err := a.Projects.AddValue(c.R.Context(), c.Param("id"), b.Text, b.Source, b.Status)
	if err != nil {
		return nil, bad(err)
	}
	return v, nil
}

func (a *App) valueSet(c *api.Ctx) (any, error) {
	var b struct{ Status, Text string }
	if err := c.Bind(&b); err != nil {
		return nil, err
	}
	id, _ := strconv.ParseInt(c.Param("vid"), 10, 64)
	v, err := a.Projects.SetValueStatus(c.R.Context(), id, b.Status, b.Text)
	if err != nil {
		return nil, bad(err)
	}
	return v, nil
}

func (a *App) problemRefine(c *api.Ctx) (any, error) {
	var b struct{ Text, Source string }
	if err := c.Bind(&b); err != nil {
		return nil, err
	}
	if strings.TrimSpace(b.Text) == "" {
		return nil, api.BadRequest("text is required")
	}
	return a.Projects.RefineProblem(c.R.Context(), c.Param("id"), b.Text, b.Source)
}

func (a *App) milestoneSave(c *api.Ctx) (any, error) {
	var m projects.Milestone
	if err := c.Bind(&m); err != nil {
		return nil, err
	}
	m.ProjectID = c.Param("id")
	if m.ID != "" {
		cur, err := a.Projects.Milestone(c.R.Context(), m.ID)
		if err != nil {
			return nil, notFound(err)
		}
		if m.Name == "" {
			m.Name = cur.Name
		}
		if m.Scope == "" {
			m.Scope = cur.Scope
		}
		if m.GateName == "" {
			m.GateName, m.GateKind = cur.GateName, cur.GateKind
		}
		if m.GateStatus == "" {
			m.GateStatus = cur.GateStatus
		}
		if m.GateCriteria == nil {
			m.GateCriteria = cur.GateCriteria
		}
		if m.Ord == 0 {
			m.Ord = cur.Ord
		}
		if m.Status == "" {
			m.Status = cur.Status
		}
	}
	if m.Name == "" {
		return nil, api.BadRequest("name is required")
	}
	return a.Projects.SaveMilestone(c.R.Context(), m)
}

func (a *App) milestoneGet(c *api.Ctx) (any, error) {
	ctx := c.R.Context()
	m, err := a.Projects.Milestone(ctx, c.Param("mid"))
	if err != nil {
		return nil, notFound(err)
	}
	ts, _ := a.Queue.List(ctx, queue.Filter{MilestoneID: m.ID})
	reqs := map[string]bool{}
	for _, t := range ts {
		for _, r := range t.Reqs {
			reqs[r] = true
		}
	}
	var rl []string
	for r := range reqs {
		rl = append(rl, r)
	}
	sort.Strings(rl)
	hist, _ := a.Hist.List(ctx, history.Query{Subject: m.ID, Limit: 50})
	for _, t := range ts {
		h, _ := a.Hist.List(ctx, history.Query{Subject: t.ID, Limit: 10})
		hist = append(hist, h...)
	}
	sort.Slice(hist, func(i, j int) bool { return hist[i].ID > hist[j].ID })
	if len(hist) > 40 {
		hist = hist[:40]
	}
	gs, _ := a.Rehearsals.List(ctx, m.ProjectID)
	var rehearsal any
	for _, g := range gs {
		if g.MilestoneID == m.ID {
			rehearsal = g
			break
		}
	}
	open, why := a.gateOpen(ctx, m.ID)
	return map[string]any{"milestone": m, "tasks": ts, "requirements": rl, "history": hist, "rehearsal": rehearsal, "workAllowed": open, "blockedBy": why}, nil
}

func (a *App) gateSet(c *api.Ctx) (any, error) {
	var b struct{ Status, Note string }
	if err := c.Bind(&b); err != nil {
		return nil, err
	}
	if b.Status == "approved" && (c.Actor.Kind != "human" || !a.Cfg.IsOwner(c.Actor.Name, c.Actor.Login)) {
		return nil, api.Forbidden("only an owner can approve a gate (ADR-006)")
	}
	m, err := a.Projects.SetGate(c.R.Context(), c.Param("mid"), b.Status, b.Note)
	if err != nil {
		return nil, bad(err)
	}
	return m, nil
}

// Artifact is a document or mockup in the project repo.
type Artifact struct {
	Path     string `json:"path"`
	Kind     string `json:"kind"` // markdown | html | image | other
	Group    string `json:"group"`
	Size     int64  `json:"size"`
	Modified string `json:"modified"`
}

func artifactKind(p string) string {
	switch strings.ToLower(filepath.Ext(p)) {
	case ".md":
		return "markdown"
	case ".html", ".htm":
		return "html"
	case ".png", ".jpg", ".jpeg", ".gif", ".svg", ".webp":
		return "image"
	case ".json":
		return "json"
	}
	return "other"
}

func (a *App) artifactList(c *api.Ctx) (any, error) {
	p, err := a.Projects.Get(c.R.Context(), c.Param("id"))
	if err != nil {
		return nil, notFound(err)
	}
	out := []Artifact{}
	root := filepath.Join(p.Path, "docs")
	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() && (d.Name() == "node_modules" || strings.HasPrefix(d.Name(), ".")) {
			return filepath.SkipDir
		}
		if d.IsDir() || len(out) > 500 {
			return nil
		}
		rel, _ := filepath.Rel(p.Path, path)
		info, _ := d.Info()
		group := "Documents"
		switch {
		case strings.Contains(rel, "mockups"):
			group = "Mockups"
		case strings.Contains(rel, "decisions") || strings.Contains(rel, "adr"):
			group = "Decisions"
		case strings.Contains(rel, "standards"):
			group = "Standards"
		case strings.Contains(rel, "screenshots"):
			group = "Screenshots"
		}
		kind := artifactKind(rel)
		if kind == "other" || (kind == "json" && group != "Documents") {
			return nil
		}
		out = append(out, Artifact{Path: rel, Kind: kind, Group: group, Size: info.Size(), Modified: info.ModTime().UTC().Format(time.RFC3339)})
		return nil
	})
	if f, err := os.Stat(filepath.Join(p.Path, "README.md")); err == nil {
		out = append([]Artifact{{Path: "README.md", Kind: "markdown", Group: "Documents", Size: f.Size(), Modified: f.ModTime().UTC().Format(time.RFC3339)}}, out...)
	}
	return out, nil
}

func safeRel(p string) (string, error) {
	clean := filepath.Clean(p)
	if clean == "." || strings.HasPrefix(clean, "..") || filepath.IsAbs(clean) {
		return "", api.BadRequest("invalid path %q", p)
	}
	return clean, nil
}

func (a *App) artifactGet(c *api.Ctx) (any, error) {
	ctx := c.R.Context()
	p, err := a.Projects.Get(ctx, c.Param("id"))
	if err != nil {
		return nil, notFound(err)
	}
	rel, err := safeRel(c.Param("path"))
	if err != nil {
		return nil, err
	}
	var content string
	if rev := c.Query("rev"); rev != "" {
		content, err = a.Git.Show(ctx, p.Path, rev, rel)
	} else {
		var b []byte
		b, err = os.ReadFile(filepath.Join(p.Path, rel))
		content = string(b)
	}
	if err != nil {
		return nil, api.NotFound("%s: %v", rel, err)
	}
	if artifactKind(rel) == "image" {
		content = ""
	}
	revs := a.Git.FileHistory(ctx, p.Path, rel, 30)
	return map[string]any{"path": rel, "kind": artifactKind(rel), "content": content, "revisions": revs, "rev": len(revs), "rawUrl": "/raw/" + p.ID + "/" + rel}, nil
}

func (a *App) artifactSave(c *api.Ctx) (any, error) {
	ctx := c.R.Context()
	p, err := a.Projects.Get(ctx, c.Param("id"))
	if err != nil {
		return nil, notFound(err)
	}
	rel, err := safeRel(c.Param("path"))
	if err != nil {
		return nil, err
	}
	var b struct{ Content, Message string }
	if err := c.Bind(&b); err != nil {
		return nil, err
	}
	revs := a.Git.FileHistory(ctx, p.Path, rel, 1000)
	msg := b.Message
	if msg == "" {
		msg = fmt.Sprintf("docs: update %s (rev %d)", rel, len(revs)+1)
	}
	hash, err := a.Courier.CommitArtifact(ctx, p, rel, b.Content, msg+"\n\nEdited by "+c.Actor.Name+" in go-build-it", c.Actor.Name)
	if err != nil {
		return nil, bad(err)
	}
	return map[string]any{"path": rel, "hash": hash, "rev": len(revs) + 1}, nil
}

type chatReq struct {
	Path, Persona, Message, Content string
	Coach                           string
	History                         []string
}

// defaultPersona picks a member of the project holding one of roles, else that role's archetype.
func (a *App) defaultPersona(ctx context.Context, projectID string, roles ...string) string {
	for _, role := range roles {
		if projectID != "" {
			if ms, _ := a.Team.ByRole(ctx, role, projectID); len(ms) > 0 {
				return ms[0].ID
			}
		}
	}
	for _, role := range roles {
		if as, _ := a.Team.Archetypes(ctx); len(as) > 0 {
			for _, ar := range as {
				if ar.Role == role && ar.Harness != "courier" {
					return ar.ID
				}
			}
		}
	}
	return ""
}

func (a *App) chatEdit(ctx context.Context, persona, kind, path, content, message string, hist []string, roles ...string) (any, error) {
	if persona == "" {
		project := ""
		if kind == "project artifact" {
			project = strings.SplitN(path, "/", 2)[0]
		}
		persona = a.defaultPersona(ctx, project, roles...)
	}
	if persona == "" {
		return nil, api.BadRequest("add an AI persona to the team first")
	}
	if strings.TrimSpace(message) == "" {
		return nil, api.BadRequest("message is required")
	}
	reply, err := a.Agent.Edit(ctx, persona, kind, path, content, message, hist)
	if err != nil {
		return nil, err
	}
	a.Hist.Must(ctx, history.Event{Type: "chat." + strings.ReplaceAll(kind, " ", "_"), Subject: path, Summary: fmt.Sprintf("%s asked %s about %s", access.FromContext(ctx).Name, persona, path), Why: message,
		Data: history.Data(map[string]any{"reply": reply.Message, "hunks": len(reply.Hunks)})})
	return reply, nil
}

func (a *App) artifactChat(c *api.Ctx) (any, error) {
	var b chatReq
	if err := c.Bind(&b); err != nil {
		return nil, err
	}
	return a.chatEdit(c.R.Context(), b.Persona, "project artifact", c.Param("id")+"/"+b.Path, b.Content, b.Message, b.History, "architect", "pm")
}

func (a *App) diffApply(c *api.Ctx) (any, error) {
	var b struct {
		Original string
		Hunks    []agent.Hunk
		Accepted []int
	}
	if err := c.Bind(&b); err != nil {
		return nil, err
	}
	return map[string]string{"content": agent.Apply(b.Original, b.Hunks, b.Accepted)}, nil
}

func (a *App) instructionsPreview(c *api.Ctx) (any, error) {
	ctx := c.R.Context()
	stds, err := a.Standards.Effective(ctx, c.Param("id"))
	if err != nil {
		return nil, err
	}
	persona := ""
	if id := c.Query("persona"); id != "" {
		if m, err := a.Team.Member(ctx, id); err == nil {
			persona = team.Persona(m)
		}
	}
	return map[string]any{"files": standards.InstructionFiles, "content": standards.Render(c.Param("id"), stds, persona)}, nil
}

func (a *App) release(c *api.Ctx) (any, error) {
	var b struct{ DryRun bool }
	_ = c.Bind(&b)
	if !b.DryRun && (c.Actor.Kind != "human" || !a.Cfg.IsOwner(c.Actor.Name, c.Actor.Login)) {
		return nil, api.Forbidden("only an owner can start a release (ADR-006)")
	}
	return a.Courier.Release(c.R.Context(), c.Param("id"), b.DryRun)
}

// Team handlers ------------------------------------------------------------

func (a *App) roleAdd(c *api.Ctx) (any, error) {
	var b struct{ ID, Name, Description string }
	if err := c.Bind(&b); err != nil {
		return nil, err
	}
	rl, err := a.Team.AddRole(c.R.Context(), b.ID, b.Name, b.Description)
	if err != nil {
		return nil, bad(err)
	}
	return rl, nil
}

func (a *App) memberGet(c *api.Ctx) (any, error) {
	ctx := c.R.Context()
	m, err := a.Team.Member(ctx, c.Param("mid"))
	if err != nil {
		return nil, notFound(err)
	}
	live, _ := a.Sessions.List(ctx, "", "", true)
	busy := 0
	for _, s := range live {
		if s.MemberID == m.ID {
			busy++
		}
	}
	out := map[string]any{"member": m, "persona": team.Persona(m), "liveSessions": busy, "tuned": m.Tuned()}
	if m.ArchetypeID != "" {
		base, _ := a.Team.ArchetypeAt(ctx, m.ArchetypeID, m.ArchetypeRev)
		out["archetype"] = base
		if m.ArchetypeLatest > m.ArchetypeRev {
			latest, _ := a.Team.Archetype(ctx, m.ArchetypeID)
			out["latest"] = latest
		}
	}
	return out, nil
}

func (a *App) memberAdd(c *api.Ctx) (any, error) {
	var b struct{ Archetype, Name, Kind, Login, Role string }
	if err := c.Bind(&b); err != nil {
		return nil, err
	}
	var m team.Member
	var err error
	if b.Kind == "human" {
		m, err = a.Team.AddHuman(c.R.Context(), c.Param("id"), b.Name, b.Login, b.Role)
	} else {
		m, err = a.Team.AddFromArchetype(c.R.Context(), c.Param("id"), b.Archetype, b.Name)
	}
	if err != nil {
		return nil, bad(err)
	}
	return m, nil
}

func (a *App) memberTune(c *api.Ctx) (any, error) {
	var t team.Tuning
	if err := c.Bind(&t); err != nil {
		return nil, err
	}
	m, err := a.Team.Tune(c.R.Context(), c.Param("mid"), t)
	if err != nil {
		return nil, bad(err)
	}
	return m, nil
}

func (a *App) memberPromote(c *api.Ctx) (any, error) {
	var b struct{ Text string }
	if err := c.Bind(&b); err != nil {
		return nil, err
	}
	ar, m, err := a.Team.Promote(c.R.Context(), c.Param("mid"), b.Text)
	if err != nil {
		return nil, bad(err)
	}
	return map[string]any{"archetype": ar, "member": m}, nil
}

func (a *App) archetypeGet(c *api.Ctx) (any, error) {
	ar, err := a.Team.Archetype(c.R.Context(), c.Param("aid"))
	if err != nil {
		return nil, notFound(err)
	}
	ch, _ := a.Team.Changes(c.R.Context(), ar.ID)
	ms, _ := a.Team.Members(c.R.Context(), "")
	var used []team.Member
	for _, m := range ms {
		if m.ArchetypeID == ar.ID {
			used = append(used, m)
		}
	}
	return map[string]any{"archetype": ar, "changes": ch, "members": used,
		"persona": team.Persona(team.Member{Name: ar.Name, Role: ar.Role, Instructions: ar.Instructions, Standards: ar.Standards})}, nil
}

func (a *App) archetypeSave(c *api.Ctx) (any, error) {
	var ar team.Archetype
	if err := c.Bind(&ar); err != nil {
		return nil, err
	}
	out, err := a.Team.SaveArchetype(c.R.Context(), ar)
	if err != nil {
		return nil, bad(err)
	}
	ch, _ := a.Team.Changes(c.R.Context(), out.ID)
	return map[string]any{"archetype": out, "changes": ch}, nil
}

func (a *App) archetypeApply(c *api.Ctx) (any, error) {
	var b struct {
		Member string
		All    bool
	}
	if err := c.Bind(&b); err != nil {
		return nil, err
	}
	if b.All {
		ms, err := a.Team.ApplyClean(c.R.Context(), c.Param("aid"))
		if err != nil {
			return nil, bad(err)
		}
		return map[string]any{"applied": ms}, nil
	}
	m, err := a.Team.Apply(c.R.Context(), b.Member)
	if err != nil {
		return nil, bad(err)
	}
	return map[string]any{"applied": []team.Member{m}}, nil
}

func (a *App) memberChat(c *api.Ctx) (any, error) {
	var b chatReq
	if err := c.Bind(&b); err != nil {
		return nil, err
	}
	m, err := a.Team.Member(c.R.Context(), c.Param("mid"))
	if err != nil {
		return nil, notFound(err)
	}
	content := b.Content
	if content == "" {
		content = team.Persona(m)
	}
	path := "archetypes/" + m.ID + ".md"
	if m.ProjectID != "" {
		path = m.ProjectID + "/team/" + m.ID + ".md"
	}
	return a.chatEdit(c.R.Context(), b.Coach, "persona definition", path, content, b.Message, b.History, "architect", "pm")
}

func (a *App) standardSave(c *api.Ctx) (any, error) {
	var b struct{ Title, Body, Why string }
	if err := c.Bind(&b); err != nil {
		return nil, err
	}
	s, err := a.Standards.Save(c.R.Context(), c.Param("sid"), b.Title, b.Body, b.Why)
	if err != nil {
		return nil, bad(err)
	}
	return s, nil
}

func (a *App) standardOverride(c *api.Ctx) (any, error) {
	var b struct{ Project, Body, Reason string }
	if err := c.Bind(&b); err != nil {
		return nil, err
	}
	o, err := a.Standards.SetOverride(c.R.Context(), c.Param("sid"), b.Project, b.Body, b.Reason)
	if err != nil {
		return nil, bad(err)
	}
	return o, nil
}

func (a *App) standardChat(c *api.Ctx) (any, error) {
	var b chatReq
	if err := c.Bind(&b); err != nil {
		return nil, err
	}
	content := b.Content
	if content == "" {
		if s, err := a.Standards.Get(c.R.Context(), c.Param("sid")); err == nil {
			content = s.Body
		}
	}
	return a.chatEdit(c.R.Context(), b.Persona, "global standard", "standards/"+c.Param("sid")+".md", content, b.Message, b.History, "reviewer", "architect")
}

// Work handlers ----------------------------------------------------------------

func (a *App) taskList(c *api.Ctx) (any, error) {
	return a.Queue.List(c.R.Context(), queue.Filter{ProjectID: c.Query("project"), Status: c.Query("status"), Role: c.Query("role"), MilestoneID: c.Query("milestone")})
}

type taskBody struct {
	Project, Title, Description, Role, Assignee, Milestone, Note string
	Priority                                                     int
	Deps, Reqs                                                   []string
}

func (a *App) taskAdd(c *api.Ctx) (any, error) {
	var b taskBody
	if err := c.Bind(&b); err != nil {
		return nil, err
	}
	if _, err := a.Projects.Get(c.R.Context(), b.Project); err != nil {
		return nil, api.BadRequest("unknown project %q", b.Project)
	}
	t, err := a.Queue.Add(c.R.Context(), queue.Task{ProjectID: b.Project, Title: b.Title, Description: b.Description, Role: b.Role, Assignee: b.Assignee,
		Priority: b.Priority, Deps: b.Deps, Reqs: b.Reqs, MilestoneID: b.Milestone})
	if err != nil {
		return nil, bad(err)
	}
	return t, nil
}

func (a *App) taskGet(c *api.Ctx) (any, error) {
	ctx := c.R.Context()
	t, err := a.Queue.Get(ctx, c.Param("tid"))
	if err != nil {
		return nil, notFound(err)
	}
	ss, _ := a.Sessions.List(ctx, "", t.ID, false)
	hist, _ := a.Hist.List(ctx, history.Query{Subject: t.ID, Limit: 50})
	return map[string]any{"task": t, "sessions": ss, "history": hist}, nil
}

func (a *App) taskUpdate(c *api.Ctx) (any, error) {
	var b taskBody
	if err := c.Bind(&b); err != nil {
		return nil, err
	}
	t := queue.Task{ID: c.Param("tid"), Title: b.Title, Description: b.Description, Role: b.Role, Assignee: b.Assignee, Priority: b.Priority, Deps: b.Deps, Reqs: b.Reqs, MilestoneID: b.Milestone}
	if b.Note != "" {
		cur, err := a.Queue.Get(c.R.Context(), t.ID)
		if err != nil {
			return nil, notFound(err)
		}
		t.ContextNote = strings.TrimSpace(cur.ContextNote + "\n\nNote from " + c.Actor.Name + ": " + b.Note)
	}
	out, err := a.Queue.Update(c.R.Context(), t)
	if err != nil {
		return nil, bad(err)
	}
	return out, nil
}

func (a *App) taskStatus(c *api.Ctx) (any, error) {
	var b struct{ Status, Why string }
	if err := c.Bind(&b); err != nil {
		return nil, err
	}
	switch b.Status {
	case queue.Queued, queue.Paused, queue.Done, queue.Failed, queue.Blocked, queue.Review:
	default:
		return nil, api.BadRequest("status must be queued, paused, blocked, review, done or failed")
	}
	t, err := a.Queue.SetStatus(c.R.Context(), c.Param("tid"), b.Status, firstNonEmpty(b.Why, "set by "+c.Actor.Name))
	if err != nil {
		return nil, notFound(err)
	}
	return t, nil
}

func (a *App) taskRun(c *api.Ctx) (any, error) {
	var b struct{ Member string }
	_ = c.Bind(&b)
	s, err := a.Sessions.Start(c.R.Context(), c.Param("tid"), b.Member)
	if err != nil {
		return nil, bad(err)
	}
	return s, nil
}

func (a *App) sessionSend(c *api.Ctx) (any, error) {
	var b struct{ Text string }
	if err := c.Bind(&b); err != nil {
		return nil, err
	}
	if err := a.Sessions.Send(c.R.Context(), c.Param("sid"), b.Text); err != nil {
		return nil, bad(err)
	}
	return map[string]bool{"ok": true}, nil
}

// Human interaction handlers ----------------------------------------------------

func (a *App) questionAsk(c *api.Ctx) (any, error) {
	var b struct {
		Project, Task, Session, Title, Why, Assumption, Recommendation string
		Blocking                                                       bool
		Options                                                        []questions.Option
		Attachments                                                    []questions.Attachment
		Reqs                                                           []string
	}
	if err := c.Bind(&b); err != nil {
		return nil, err
	}
	if b.Session == "" {
		b.Session = c.R.Header.Get("X-GBI-Session")
	}
	if b.Task == "" {
		b.Task = c.R.Header.Get("X-GBI-Task")
	}
	q, err := a.Questions.Ask(c.R.Context(), questions.Question{ProjectID: b.Project, TaskID: b.Task, SessionID: b.Session, Title: b.Title, Why: b.Why,
		Blocking: b.Blocking, Assumption: b.Assumption, Recommendation: b.Recommendation, Options: b.Options, Attachments: b.Attachments, Reqs: b.Reqs})
	if err != nil {
		return nil, bad(err)
	}
	return q, nil
}

func (a *App) questionAnswer(c *api.Ctx) (any, error) {
	var b struct{ Answer, Choice, ProposeValue string }
	if err := c.Bind(&b); err != nil {
		return nil, err
	}
	if c.Actor.Kind != "human" {
		return nil, api.Forbidden("questions are answered by humans")
	}
	q, err := a.Questions.Answer(c.R.Context(), c.Param("qid"), b.Answer, b.Choice, b.ProposeValue)
	if err != nil {
		return nil, bad(err)
	}
	return q, nil
}

func (a *App) reviewVerdict(c *api.Ctx) (any, error) {
	var b struct{ Verdict, Comment string }
	if err := c.Bind(&b); err != nil {
		return nil, err
	}
	rv, err := a.Reviews.Verdict(c.R.Context(), c.Param("rid"), b.Verdict, b.Comment)
	if err != nil {
		return nil, bad(err)
	}
	return rv, nil
}

func (a *App) rehearsalCreate(c *api.Ctx) (any, error) {
	var b struct {
		Project, Milestone, Title, Kind string
		Steps                           []rehearsals.Step
	}
	if err := c.Bind(&b); err != nil {
		return nil, err
	}
	g, err := a.Rehearsals.Create(c.R.Context(), rehearsals.Rehearsal{ProjectID: b.Project, MilestoneID: b.Milestone, Title: b.Title, Kind: b.Kind, Steps: b.Steps})
	if err != nil {
		return nil, bad(err)
	}
	return g, nil
}

func (a *App) rehearsalFinding(c *api.Ctx) (any, error) {
	var b struct {
		Step int
		Text string
	}
	if err := c.Bind(&b); err != nil {
		return nil, err
	}
	fd, err := a.Rehearsals.AddFinding(c.R.Context(), c.Param("gid"), b.Step, b.Text)
	if err != nil {
		return nil, bad(err)
	}
	return fd, nil
}

func (a *App) rehearsalAdvance(c *api.Ctx) (any, error) {
	var b struct {
		Step   int
		Status string
	}
	if err := c.Bind(&b); err != nil {
		return nil, err
	}
	g, err := a.Rehearsals.Advance(c.R.Context(), c.Param("gid"), b.Step, b.Status)
	if err != nil {
		return nil, bad(err)
	}
	return g, nil
}

func (a *App) rehearsalDecide(c *api.Ctx) (any, error) {
	var b struct{ Decision, Note string }
	if err := c.Bind(&b); err != nil {
		return nil, err
	}
	g, err := a.Rehearsals.Decide(c.R.Context(), c.Param("gid"), b.Decision, b.Note)
	if err != nil {
		return nil, bad(err)
	}
	return g, nil
}

func (a *App) changeRequest(c *api.Ctx) (any, error) {
	var b struct{ Now bool }
	_ = c.Bind(&b)
	id, _ := strconv.ParseInt(c.Param("fid"), 10, 64)
	t, err := a.Rehearsals.ChangeRequest(c.R.Context(), id, b.Now)
	if err != nil {
		return nil, bad(err)
	}
	if b.Now {
		go func() { _, _ = a.Sessions.Start(context.WithoutCancel(c.R.Context()), t.ID, "") }()
	}
	return t, nil
}

func (a *App) optionsCreate(c *api.Ctx) (any, error) {
	var b struct {
		Project, Title string
		Options        []options.Option
		ValueLens      []string
	}
	if err := c.Bind(&b); err != nil {
		return nil, err
	}
	s, err := a.Options.Create(c.R.Context(), options.Set{ProjectID: b.Project, Title: b.Title, Options: b.Options, ValueLens: b.ValueLens})
	if err != nil {
		return nil, bad(err)
	}
	return s, nil
}

func (a *App) optionsChoose(c *api.Ctx) (any, error) {
	var b struct {
		Picks         []options.Pick
		Reason, Value string
	}
	if err := c.Bind(&b); err != nil {
		return nil, err
	}
	if c.Actor.Kind != "human" {
		return nil, api.Forbidden("option sets are decided by humans")
	}
	s, err := a.Options.Choose(c.R.Context(), c.Param("oid"), b.Picks, b.Reason, b.Value)
	if err != nil {
		return nil, bad(err)
	}
	return s, nil
}

// History handlers ---------------------------------------------------------------

func (a *App) historyList(c *api.Ctx) (any, error) {
	return a.Hist.List(c.R.Context(), history.Query{ProjectID: c.Query("project"), Type: c.Query("type"), Actor: c.Query("actor"), Text: c.Query("q"),
		Subject: c.Query("subject"), Since: c.Query("since"), Limit: atoi(c.Query("limit"), 200), BeforeID: int64(atoi(c.Query("before"), 0))})
}

// historyGet returns an event with its related records, so each type gets a detail view (NFR-14).
func (a *App) historyGet(c *api.Ctx) (any, error) {
	ctx := c.R.Context()
	id, _ := strconv.ParseInt(c.Param("eid"), 10, 64)
	e, err := a.Hist.Get(ctx, id)
	if err != nil {
		return nil, api.NotFound("event %s not found", c.Param("eid"))
	}
	related, _ := a.Hist.List(ctx, history.Query{Subject: e.Subject, Limit: 30})
	out := map[string]any{"event": e, "related": related}
	kind, _, _ := strings.Cut(e.Type, ".")
	switch kind {
	case "session":
		if s, err := a.Sessions.Get(ctx, e.Subject); err == nil {
			out["session"] = s
			out["output"], _ = a.Sessions.Output(ctx, s.ID, 0, "structured")
		}
	case "question":
		out["question"], _ = a.Questions.Get(ctx, e.Subject)
	case "review":
		out["review"], _ = a.Reviews.Get(ctx, e.Subject)
	case "rehearsal":
		out["rehearsal"], _ = a.Rehearsals.Get(ctx, e.Subject)
	case "options":
		out["optionSet"], _ = a.Options.Get(ctx, e.Subject)
	case "task":
		out["task"], _ = a.Queue.Get(ctx, e.Subject)
	}
	if e.ProjectID != "" {
		// Events that happened as a result: same project, within 10 minutes after.
		if t, err := time.Parse(time.RFC3339, e.TS); err == nil {
			after, _ := a.Hist.List(ctx, history.Query{ProjectID: e.ProjectID, Since: e.TS, Limit: 50})
			var res []history.Event
			for _, x := range after {
				if xt, _ := time.Parse(time.RFC3339, x.TS); x.ID > e.ID && xt.Sub(t) < 10*time.Minute {
					res = append(res, x)
				}
			}
			out["consequences"] = res
		}
	}
	return out, nil
}

func (a *App) proposalCreate(c *api.Ctx) (any, error) {
	var b struct{ Title, Body, Evidence string }
	if err := c.Bind(&b); err != nil {
		return nil, err
	}
	p, err := a.Proposals.Create(c.R.Context(), b.Title, b.Body, b.Evidence)
	if err != nil {
		return nil, bad(err)
	}
	return p, nil
}

func (a *App) proposalDecide(c *api.Ctx) (any, error) {
	var b struct{ Decision, Note string }
	if err := c.Bind(&b); err != nil {
		return nil, err
	}
	if c.Actor.Kind != "human" {
		return nil, api.Forbidden("proposals are decided by humans")
	}
	p, task, err := a.Proposals.Decide(c.R.Context(), c.Param("pid"), b.Decision, b.Note)
	if err != nil {
		return nil, bad(err)
	}
	return map[string]any{"proposal": p, "task": task}, nil
}

// updateInstall starts an update job (FR-06d). herdr keeps members running while
// go-build-it restarts, so a manual update does not wait for them (ADR-011).
func (a *App) updateInstall(c *api.Ctx) (any, error) {
	j, err := a.Updates.Start("manual", c.Actor.Name)
	if errors.Is(err, update.ErrBusy) {
		return nil, api.Conflict("%s", err.Error())
	}
	if err != nil {
		return nil, bad(err)
	}
	return map[string]any{"job": j}, nil
}

func firstNonEmpty(a ...string) string {
	for _, s := range a {
		if s != "" {
			return s
		}
	}
	return ""
}

func firstNonEmptyStr(a ...string) string {
	for _, s := range a {
		if s != "" {
			return s
		}
	}
	return ""
}
