package app

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/techmuch/go-build-it/internal/api"
	"github.com/techmuch/go-build-it/internal/artifacts"
	"github.com/techmuch/go-build-it/internal/history"
	"github.com/techmuch/go-build-it/internal/projects"
	"github.com/techmuch/go-build-it/internal/queue"
)

// Project documents (FR-140–FR-150, ADR-014): entry points, labels, the check, the
// standard set, traceability, Compare and the tracker snapshot. Every change is a commit
// by Courier and an artifact.* history event.
func (a *App) documentRoutes() []api.Route {
	return []api.Route{
		r("GET", "/api/projects/{id}/documents", "artifacts.files", "artifacts files", "Every file under docs/ with its labels, the entry points and the unlabelled files", a.docFiles, f("label", "only files with this label")),
		r("POST", "/api/projects/{id}/documents/add/{path...}", "artifacts.add", "artifacts add", "Label a file or folder under docs/ (a label is required) and commit it", a.docAdd, f("label", "labels, comma-separated: "+strings.Join(artifacts.Labels, ", "))),
		r("POST", "/api/projects/{id}/documents/label/{path...}", "artifacts.label", "artifacts label", "Set a file's or folder's labels (--remove takes the given labels away)", a.docLabel, f("label", "labels, comma-separated"), fb("remove", "remove these labels instead of setting them")),
		r("POST", "/api/projects/{id}/documents/entry/{role}/{path...}", "artifacts.entry", "artifacts entry", "Make an HTML file the entry point for requirements, mockups, plan or tracker", a.docEntry),
		r("GET", "/api/projects/{id}/documents/check", "artifacts.check", "artifacts check", "Missing entry points, unlabelled files, stale labels, broken links and unknown IDs", a.docCheck),
		r("POST", "/api/projects/{id}/documents/init", "artifacts.init", "artifacts init", "Create the standard documents from templates (never overwrites)", a.docInit),
		r("POST", "/api/projects/{id}/documents/snapshot", "artifacts.snapshot", "artifacts snapshot", "Write docs/tracker/tracker.json from the project's milestones and tasks", a.docSnapshot),
		r("GET", "/api/projects/{id}/trace", "trace.list", "trace list", "Requirements with their mockup views, tasks and real screens, and the gaps", a.traceList, fb("gaps", "only rows with gaps")),
		r("GET", "/api/projects/{id}/compare", "compare.list", "compare list", "Each mockup view with its screenshot and Compare state", a.compareList),
		r("POST", "/api/projects/{id}/compare/{view}/accept", "compare.accept", "compare accept", "Accept the real screen for a mockup view", a.compareAccept, f("note", "why it is fine")),
		r("POST", "/api/projects/{id}/compare/{view}/task", "compare.task", "compare task", "File a task to make the real screen match a mockup view", a.compareTask, f("note", "what to change"), f("role", "role for the task (default developer)")),
	}
}

func (a *App) docProject(ctx context.Context, c *api.Ctx) (projects.Project, artifacts.Manifest, error) {
	p, err := a.Projects.Get(ctx, c.Param("id"))
	if err != nil {
		return p, artifacts.Manifest{}, notFound(err)
	}
	m, err := artifacts.Load(p.Path)
	if err != nil {
		return p, m, bad(err)
	}
	return p, m, nil
}

// commitDocs writes files (path → content; nil content means the file is already on disk)
// and commits them through Courier as one revision each.
func (a *App) commitDocs(ctx context.Context, p projects.Project, actor, msg string, files map[string]*string) error {
	for rel, content := range files {
		if content == nil && a.unchanged(ctx, p, rel) {
			continue // already committed as it is
		}
		if content != nil {
			if b, err := os.ReadFile(filepath.Join(p.Path, filepath.FromSlash(rel))); err == nil && string(b) == *content && a.unchanged(ctx, p, rel) {
				continue
			}
		}
		var body string
		if content != nil {
			body = *content
		} else {
			b, err := os.ReadFile(filepath.Join(p.Path, filepath.FromSlash(rel)))
			if err != nil {
				return err
			}
			body = string(b)
		}
		if _, err := a.Courier.CommitArtifact(ctx, p, rel, body, msg+"\n\nBy "+actor+" in go-build-it", actor); err != nil && !strings.Contains(err.Error(), "no changes") {
			return err
		}
	}
	return nil
}

func ptr(s string) *string { return &s }

// unchanged reports whether rel is tracked by git and has no changes.
func (a *App) unchanged(ctx context.Context, p projects.Project, rel string) bool {
	if _, err := a.Git.Run(ctx, p.Path, "git", "ls-files", "--error-unmatch", "--", rel); err != nil {
		return false
	}
	out, err := a.Git.Run(ctx, p.Path, "git", "status", "--porcelain", "--", rel)
	return err == nil && strings.TrimSpace(out) == ""
}

func splitLabels(s string) []string {
	var out []string
	for _, l := range strings.FieldsFunc(s, func(r rune) bool { return r == ',' || r == ' ' }) {
		out = append(out, strings.ToLower(l))
	}
	return out
}

func (a *App) docFiles(c *api.Ctx) (any, error) {
	p, m, err := a.docProject(c.R.Context(), c)
	if err != nil {
		return nil, err
	}
	files := artifacts.Scan(p.Path, m)
	if l := c.Query("label"); l != "" {
		var keep []artifacts.File
		for _, f := range files {
			for _, x := range f.Labels {
				if x == l {
					keep = append(keep, f)
					break
				}
			}
		}
		files = keep
	}
	if files == nil {
		files = []artifacts.File{}
	}
	un := artifacts.Unlabelled(artifacts.Scan(p.Path, m))
	if un == nil {
		un = []artifacts.File{}
	}
	return map[string]any{"entries": m.Entries, "labels": m.Labels, "files": files, "unlabelled": un, "roles": artifacts.Roles, "allLabels": artifacts.Labels, "manifest": artifacts.ManifestPath}, nil
}

func (a *App) labelDoc(c *api.Ctx, add bool) (any, error) {
	ctx := context.WithoutCancel(c.R.Context()) // finish the commit even if the client goes away
	var b struct {
		Label  string `json:"label"`
		Remove bool   `json:"remove"`
	}
	if err := c.Bind(&b); err != nil {
		return nil, err
	}
	labels := splitLabels(b.Label)
	if len(labels) == 0 {
		return nil, api.BadRequest("a label is required (--label %s)", strings.Join(artifacts.Labels, "|"))
	}
	a.docMu.Lock()
	defer a.docMu.Unlock()
	p, m, err := a.docProject(ctx, c)
	if err != nil {
		return nil, err
	}
	rel := c.Param("path")
	if b.Remove {
		cur := m.Labels[rel]
		if cur == nil {
			if cl, err := artifacts.Clean(rel); err == nil {
				cur = m.Labels[cl]
				if cur == nil {
					cur = m.Labels[cl+"/"]
				}
			}
		}
		var keep []string
		for _, x := range cur {
			drop := false
			for _, l := range labels {
				drop = drop || l == x
			}
			if !drop {
				keep = append(keep, x)
			}
		}
		labels = keep
	} else if add {
		if cl, err := artifacts.Clean(rel); err == nil {
			labels = append(labels, m.Labels[cl]...)
		}
	}
	key, err := m.SetLabels(p.Path, rel, labels)
	if err != nil {
		return nil, bad(err)
	}
	files := map[string]*string{artifacts.ManifestPath: ptr(m.Encode())}
	if add && !strings.HasSuffix(key, "/") {
		files[key] = nil // commit the file itself too, if it is new or changed
	}
	what := strings.Join(m.Labels[key], ", ")
	if what == "" {
		what = "no labels"
	}
	if err := a.commitDocs(ctx, p, c.Actor.Name, fmt.Sprintf("docs: label %s (%s)", key, what), files); err != nil {
		return nil, bad(err)
	}
	a.Hist.Must(ctx, history.Event{ProjectID: p.ID, Type: "artifact.labeled", Subject: key, Summary: fmt.Sprintf("Labelled %s: %s", key, what), Data: history.Data(map[string]any{"labels": m.Labels[key]})})
	return map[string]any{"path": key, "labels": m.Labels[key]}, nil
}

func (a *App) docAdd(c *api.Ctx) (any, error)   { return a.labelDoc(c, true) }
func (a *App) docLabel(c *api.Ctx) (any, error) { return a.labelDoc(c, false) }

func (a *App) docEntry(c *api.Ctx) (any, error) {
	ctx := context.WithoutCancel(c.R.Context()) // finish the commit even if the client goes away
	a.docMu.Lock()
	defer a.docMu.Unlock()
	p, m, err := a.docProject(ctx, c)
	if err != nil {
		return nil, err
	}
	role := c.Param("role")
	rel, err := m.SetEntry(p.Path, role, c.Param("path"))
	if err != nil {
		return nil, bad(err)
	}
	if err := a.commitDocs(ctx, p, c.Actor.Name, fmt.Sprintf("docs: %s entry point is %s", role, rel), map[string]*string{artifacts.ManifestPath: ptr(m.Encode())}); err != nil {
		return nil, bad(err)
	}
	a.Hist.Must(ctx, history.Event{ProjectID: p.ID, Type: "artifact.entry", Subject: rel, Summary: fmt.Sprintf("The %s entry point is now %s", role, rel), Data: history.Data(map[string]any{"role": role})})
	return map[string]any{"role": role, "path": rel, "entries": m.Entries}, nil
}

func (a *App) docCheck(c *api.Ctx) (any, error) {
	p, m, err := a.docProject(c.R.Context(), c)
	if err != nil {
		return nil, err
	}
	ps := artifacts.Check(p.Path, m)
	return map[string]any{"ok": len(ps) == 0, "problems": ps}, nil
}

func (a *App) docInit(c *api.Ctx) (any, error) {
	ctx := context.WithoutCancel(c.R.Context()) // finish the commit even if the client goes away
	a.docMu.Lock()
	defer a.docMu.Unlock()
	p, m, err := a.docProject(ctx, c)
	if err != nil {
		return nil, err
	}
	written, kept, err := artifacts.Init(p.Path, p.ID, &m)
	if err != nil {
		return nil, bad(err)
	}
	files := map[string]*string{artifacts.ManifestPath: ptr(m.Encode())}
	for _, w := range written {
		files[w] = nil
	}
	snap := a.snapshot(ctx, p)
	files[artifacts.TrackerPath] = ptr(artifacts.EncodeSnapshot(snap))
	if err := a.commitDocs(ctx, p, c.Actor.Name, "docs: create the standard project documents", files); err != nil {
		return nil, bad(err)
	}
	a.Hist.Must(ctx, history.Event{ProjectID: p.ID, Type: "artifact.init", Subject: p.ID, Summary: fmt.Sprintf("Created the standard documents (%d files; %d kept as they were)", len(written), len(kept)),
		Data: history.Data(map[string]any{"written": written, "kept": kept})})
	if written == nil {
		written = []string{}
	}
	if kept == nil {
		kept = []string{}
	}
	return map[string]any{"written": written, "kept": kept, "entries": m.Entries}, nil
}

func (a *App) snapshot(ctx context.Context, p projects.Project) artifacts.Snapshot {
	s := artifacts.Snapshot{Project: p.ID, Generated: time.Now().UTC().Format(time.RFC3339), Milestones: []map[string]any{}, Tasks: []map[string]any{}}
	ms, _ := a.Projects.Milestones(ctx, p.ID)
	for _, m := range ms {
		s.Milestones = append(s.Milestones, map[string]any{"id": m.ID, "name": m.Name, "status": m.Status, "gate": m.GateName, "gateStatus": m.GateStatus, "target": m.Target})
	}
	ts, _ := a.Queue.List(ctx, queue.Filter{ProjectID: p.ID})
	for _, t := range ts {
		reqs := t.Reqs
		if reqs == nil {
			reqs = []string{}
		}
		s.Tasks = append(s.Tasks, map[string]any{"id": t.ID, "title": t.Title, "status": t.Status, "role": t.Role, "assignee": t.Assignee, "milestone": t.MilestoneID, "reqs": reqs})
		s.Total++
		if t.Status == queue.Done {
			s.Done++
		}
	}
	return s
}

func (a *App) docSnapshot(c *api.Ctx) (any, error) {
	ctx := context.WithoutCancel(c.R.Context()) // finish the commit even if the client goes away
	a.docMu.Lock()
	defer a.docMu.Unlock()
	p, _, err := a.docProject(ctx, c)
	if err != nil {
		return nil, err
	}
	s := a.snapshot(ctx, p)
	if err := a.commitDocs(ctx, p, c.Actor.Name, fmt.Sprintf("docs: tracker snapshot (%d/%d tasks done)", s.Done, s.Total), map[string]*string{artifacts.TrackerPath: ptr(artifacts.EncodeSnapshot(s))}); err != nil {
		return nil, bad(err)
	}
	a.Hist.Must(ctx, history.Event{ProjectID: p.ID, Type: "artifact.snapshot", Subject: artifacts.TrackerPath, Summary: fmt.Sprintf("Wrote the tracker snapshot: %d/%d tasks done", s.Done, s.Total)})
	return map[string]any{"path": artifacts.TrackerPath, "done": s.Done, "total": s.Total, "milestones": len(s.Milestones)}, nil
}

func (a *App) traceData(ctx context.Context, p projects.Project, m artifacts.Manifest) (artifacts.Trace, []artifacts.File, error) {
	files := artifacts.Scan(p.Path, m)
	ts, _ := a.Queue.List(ctx, queue.Filter{ProjectID: p.ID})
	var tasks []artifacts.Task
	for _, t := range ts {
		tasks = append(tasks, artifacts.Task{ID: t.ID, Title: t.Title, Description: t.Description, Status: t.Status, Reqs: t.Reqs})
	}
	tr, err := artifacts.BuildTrace(p.Path, m, files, tasks)
	return tr, files, err
}

func (a *App) traceList(c *api.Ctx) (any, error) {
	ctx := c.R.Context()
	p, m, err := a.docProject(ctx, c)
	if err != nil {
		return nil, err
	}
	tr, _, err := a.traceData(ctx, p, m)
	if err != nil {
		return nil, bad(err)
	}
	if c.Query("gaps") == "true" || c.Query("gaps") == "1" {
		var keep []artifacts.TraceRow
		for _, r := range tr.Rows {
			if len(r.Gaps) > 0 {
				keep = append(keep, r)
			}
		}
		if keep == nil {
			keep = []artifacts.TraceRow{}
		}
		tr.Rows = keep
	}
	return tr, nil
}

func (a *App) compareList(c *api.Ctx) (any, error) {
	ctx := c.R.Context()
	p, m, err := a.docProject(ctx, c)
	if err != nil {
		return nil, err
	}
	tr, _, err := a.traceData(ctx, p, m)
	if err != nil {
		return nil, bad(err)
	}
	counts := map[string]int{}
	for _, v := range tr.Views {
		counts[v.State]++
	}
	views := tr.Views
	if views == nil {
		views = []artifacts.ViewState{}
	}
	return map[string]any{"views": views, "counts": counts, "toReview": counts[artifacts.StateNew] + counts[artifacts.StateDrifted]}, nil
}

var viewIDRe = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)

func (a *App) decide(c *api.Ctx, decision string) (any, error) {
	ctx := context.WithoutCancel(c.R.Context()) // finish the commit even if the client goes away
	var b struct{ Note, Role string }
	if err := c.Bind(&b); err != nil {
		return nil, err
	}
	a.docMu.Lock()
	defer a.docMu.Unlock()
	p, m, err := a.docProject(ctx, c)
	if err != nil {
		return nil, err
	}
	id := c.Param("view")
	if !viewIDRe.MatchString(id) {
		return nil, api.BadRequest("invalid view id %q", id)
	}
	tr, _, err := a.traceData(ctx, p, m)
	if err != nil {
		return nil, bad(err)
	}
	var vs *artifacts.ViewState
	for i := range tr.Views {
		if tr.Views[i].ID == id {
			vs = &tr.Views[i]
		}
	}
	if vs == nil {
		return nil, api.NotFound("no mockup view %s (views are data-view marks in files labelled mockup)", id)
	}
	if vs.Screenshot == "" {
		return nil, api.BadRequest("view %s has no screenshot yet: add docs/…/%s.png and label it screenshot", id, id)
	}
	dec, err := artifacts.LoadDecisions(p.Path)
	if err != nil {
		return nil, bad(err)
	}
	d := artifacts.Decision{Decision: decision, Screenshot: vs.Screenshot, SHA256: artifacts.FileSHA(p.Path, vs.Screenshot), Note: strings.TrimSpace(b.Note), By: c.Actor.Name, At: time.Now().UTC().Format(time.RFC3339)}
	summary := fmt.Sprintf("Accepted the real screen for %s", id)
	if decision == "task" {
		if d.Note == "" {
			return nil, api.BadRequest("say what to change (--note)")
		}
		role := b.Role
		if role == "" {
			role = "developer"
		}
		title := "Match " + id
		if strings.HasPrefix(vs.Title, id) {
			title = "Match " + vs.Title
		} else if vs.Title != "" {
			title += " · " + vs.Title
		}
		title += " to its mockup"
		t, err := a.Queue.Add(ctx, queue.Task{ProjectID: p.ID, Title: title, Role: role, Reqs: vs.Reqs,
			Description: fmt.Sprintf("%s\n\nMockup view %s: %s#%s\nReal screen: %s\nCompare it in go-build-it: Artifacts → Compare → %s", d.Note, id, vs.File, id, vs.Screenshot, id)})
		if err != nil {
			return nil, bad(err)
		}
		d.Task = t.ID
		summary = fmt.Sprintf("Filed %s so %s matches its mockup", t.ID, id)
	}
	dec[id] = d
	if err := a.commitDocs(ctx, p, c.Actor.Name, "docs: compare "+id+": "+decision, map[string]*string{artifacts.ComparePath: ptr(artifacts.EncodeDecisions(dec))}); err != nil {
		return nil, bad(err)
	}
	a.Hist.Must(ctx, history.Event{ProjectID: p.ID, Type: "artifact.compared", Subject: id, Summary: summary, Why: d.Note, Data: history.Data(map[string]any{"decision": decision, "screenshot": d.Screenshot, "task": d.Task})})
	return map[string]any{"view": id, "decision": d}, nil
}

func (a *App) compareAccept(c *api.Ctx) (any, error) { return a.decide(c, "accepted") }
func (a *App) compareTask(c *api.Ctx) (any, error)   { return a.decide(c, "task") }

// RawHandler serves a project's files for the document viewer (FR-144). Every response is
// sandboxed (an opaque origin with no access to go-build-it's API), and may be fetched by
// the page itself (CORS). HTML gets gbi-nav.js, so the viewer can follow it. A Markdown
// file asked for with ?gbi=md is rendered in a small page. ?rev= serves a git revision.
func (a *App) RawHandler(w http.ResponseWriter, r *http.Request) {
	p, err := a.Projects.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	rel := filepath.Clean(r.PathValue("path"))
	if strings.HasPrefix(rel, "..") || filepath.IsAbs(rel) {
		http.Error(w, "bad path", http.StatusBadRequest)
		return
	}
	slash := filepath.ToSlash(rel)
	w.Header().Set("Content-Security-Policy", "sandbox allow-scripts allow-forms allow-popups allow-popups-to-escape-sandbox")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Cache-Control", "no-cache")
	kind := artifacts.Kind(slash)
	if kind == "markdown" && r.URL.Query().Get("gbi") == "md" {
		name := path.Base(slash)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprintf(w, `<!doctype html><html><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>%s</title>
<style>body{margin:0;font:15px/1.6 -apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif;color:#1c2329}main{max-width:900px;padding:12px 24px 40px}code{background:#f1f3f5;border-radius:4px;padding:0 4px}</style>
<script src="/gbi-assets/gbi-md.js" defer></script><script src="/gbi-assets/gbi-nav.js"></script></head>
<body><main><gbi-md src="%s"></gbi-md></main></body></html>`, htmlEscape(name), htmlEscape(name))
		return
	}
	var body []byte
	if rev := r.URL.Query().Get("rev"); rev != "" {
		s, err := a.Git.Show(r.Context(), p.Path, rev, slash)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		body = []byte(s)
	} else if kind == "html" {
		body, err = os.ReadFile(filepath.Join(p.Path, rel))
		if err != nil {
			http.NotFound(w, r)
			return
		}
	}
	if body == nil {
		http.ServeFile(w, r, filepath.Join(p.Path, rel))
		return
	}
	if kind == "html" {
		body = injectNav(body)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
	} else if ct := mimeFor(slash); ct != "" {
		w.Header().Set("Content-Type", ct)
	}
	_, _ = w.Write(body)
}

// AssetsHandler serves /gbi-assets/ (gbi-md.js, gbi-nav.js) to sandboxed pages.
func (a *App) AssetsHandler() http.Handler {
	files := http.StripPrefix("/gbi-assets/", http.FileServerFS(artifacts.Assets()))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Cache-Control", "no-cache")
		files.ServeHTTP(w, r)
	})
}

func injectNav(b []byte) []byte {
	const tag = `<script src="/gbi-assets/gbi-nav.js"></script>`
	s := string(b)
	if i := strings.LastIndex(strings.ToLower(s), "</body>"); i >= 0 {
		return []byte(s[:i] + tag + s[i:])
	}
	return []byte(s + tag)
}

func htmlEscape(s string) string {
	return strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;").Replace(s)
}

func mimeFor(p string) string {
	switch artifacts.Kind(p) {
	case "markdown":
		return "text/markdown; charset=utf-8"
	case "json":
		return "application/json"
	}
	switch strings.ToLower(path.Ext(p)) {
	case ".png":
		return "image/png"
	case ".svg":
		return "image/svg+xml"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".css":
		return "text/css; charset=utf-8"
	case ".js":
		return "text/javascript; charset=utf-8"
	}
	return ""
}
