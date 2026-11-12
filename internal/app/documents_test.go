package app

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/techmuch/go-build-it/docs"
)

// gbi runs the real CLI against the test server.
func (e *env) gbi(args ...string) (string, error) {
	cmd := exec.Command(gbiBin, args...)
	cmd.Env = append(os.Environ(), "GBI_URL="+e.srv.URL)
	cmd.Dir = e.t.TempDir()
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func (e *env) gbiJSON(v any, args ...string) {
	e.t.Helper()
	out, err := e.gbi(append(args, "--json")...)
	if err != nil {
		e.t.Fatalf("gbi %v: %v\n%s", args, err, out)
	}
	if err := json.Unmarshal([]byte(out), v); err != nil {
		e.t.Fatalf("gbi %v: %v\n%s", args, err, out)
	}
}

// P20: the standard set, explicit labels, entry points, Compare, traceability and the
// tracker snapshot, all through the CLI, each committed by Courier.
func TestProjectDocuments(t *testing.T) {
	e := setup(t)
	var init struct{ Written, Kept []string }
	e.gbiJSON(&init, "artifacts", "init", "demo")
	if len(init.Kept) != 1 || init.Kept[0] != "docs/requirements.md" || len(init.Written) < 6 {
		t.Fatalf("init: %+v", init)
	}
	if b, _ := os.ReadFile(filepath.Join(e.repo, "docs/requirements.md")); !strings.Contains(string(b), "FR-01") {
		t.Fatal("init overwrote requirements.md")
	}
	if out := git(t, e.repo, "status", "--porcelain"); out != "" {
		t.Fatalf("init left uncommitted files:\n%s", out)
	}

	// A file added outside gbi is unlabelled until someone labels it; no guessing.
	os.MkdirAll(filepath.Join(e.repo, "docs/research"), 0o755)
	os.WriteFile(filepath.Join(e.repo, "docs/research/notes.md"), []byte("# notes\n"), 0o644)
	var files struct {
		Entries    map[string]string
		Unlabelled []struct{ Path string }
	}
	e.gbiJSON(&files, "artifacts", "files", "demo")
	if len(files.Unlabelled) != 1 || files.Unlabelled[0].Path != "docs/research/notes.md" || files.Entries["mockups"] != "docs/mockups/index.html" {
		t.Fatalf("files: %+v", files)
	}
	if out, err := e.gbi("artifacts", "add", "demo", "docs/research/notes.md"); err == nil || !strings.Contains(out, "label is required") {
		t.Fatalf("add without a label should fail: %v %s", err, out)
	}
	if out, err := e.gbi("artifacts", "add", "demo", "docs/research/notes.md", "--label", "guess"); err == nil || !strings.Contains(out, "unknown label") {
		t.Fatalf("unknown label should fail: %v %s", err, out)
	}
	if out, err := e.gbi("artifacts", "add", "demo", "docs/research/notes.md", "--label", "research"); err != nil {
		t.Fatalf("add: %v %s", err, out)
	}
	if !strings.Contains(git(t, e.repo, "ls-files", "docs/research/notes.md"), "notes.md") {
		t.Fatal("the added file was not committed")
	}
	if out, err := e.gbi("artifacts", "entry", "demo", "plan", "docs/plan.md"); err == nil || !strings.Contains(out, "HTML") {
		t.Fatalf("a Markdown entry point should be refused: %v %s", err, out)
	}
	var check struct {
		Problems []struct{ Kind, Path string }
	}
	e.gbiJSON(&check, "artifacts", "check", "demo")
	if len(check.Problems) != 0 {
		t.Fatalf("check after init and labelling: %+v", check.Problems)
	}

	// Compare: the template's view v01 (FR-01) gets a screenshot.
	shot := filepath.Join(e.repo, "docs/screenshots/v01.png")
	os.WriteFile(shot, []byte("png-1"), 0o644)
	var cmp struct {
		Views []struct {
			ID, State, Screenshot string
			Reqs                  []string
		}
		ToReview int
	}
	e.gbiJSON(&cmp, "compare", "list", "demo")
	if len(cmp.Views) != 1 || cmp.Views[0].State != "not-reviewed" || cmp.Views[0].Screenshot != "docs/screenshots/v01.png" || cmp.ToReview != 1 {
		t.Fatalf("compare: %+v", cmp)
	}
	if out, err := e.gbi("compare", "accept", "demo", "v01", "--note", "matches"); err != nil {
		t.Fatalf("accept: %v %s", err, out)
	}
	e.gbiJSON(&cmp, "compare", "list", "demo")
	if cmp.Views[0].State != "accepted" {
		t.Fatalf("after accept: %+v", cmp.Views)
	}
	os.WriteFile(shot, []byte("png-2"), 0o644)
	e.gbiJSON(&cmp, "compare", "list", "demo")
	if cmp.Views[0].State != "drifted" {
		t.Fatalf("a changed screenshot should drift: %+v", cmp.Views)
	}
	var task struct {
		Decision struct{ Task string }
	}
	e.gbiJSON(&task, "compare", "task", "demo", "v01", "--note", "Make the button full width")
	tk, err := e.a.Queue.Get(e.ctx, task.Decision.Task)
	if err != nil || len(tk.Reqs) != 1 || tk.Reqs[0] != "FR-01" || !strings.Contains(tk.Description, "Make the button full width") {
		t.Fatalf("filed task: %+v %v", tk, err)
	}

	// Traceability: FR-01 → view v01 → the filed task.
	var tr struct {
		Rows []struct {
			ID    string
			Views []string
			Tasks []string
		}
	}
	e.gbiJSON(&tr, "trace", "list", "demo")
	if len(tr.Rows) != 1 || tr.Rows[0].ID != "FR-01" || len(tr.Rows[0].Views) != 1 || len(tr.Rows[0].Tasks) != 1 || tr.Rows[0].Tasks[0] != tk.ID {
		t.Fatalf("trace: %+v", tr)
	}

	// Tracker snapshot.
	if out, err := e.gbi("artifacts", "snapshot", "demo"); err != nil {
		t.Fatalf("snapshot: %v %s", err, out)
	}
	b, _ := os.ReadFile(filepath.Join(e.repo, "docs/tracker/tracker.json"))
	if !strings.Contains(string(b), tk.ID) {
		t.Fatalf("snapshot missing the task:\n%s", b)
	}
	for _, typ := range []string{"artifact.init", "artifact.labeled", "artifact.compared", "artifact.snapshot"} {
		if evs, _ := e.a.Hist.List(e.ctx, historyQueryType(typ)); len(evs) == 0 {
			t.Errorf("no %s event", typ)
		}
	}

	// FR-151: a project's own role gets a tab, and gbi keeps the order when it rewrites the file.
	os.WriteFile(filepath.Join(e.repo, "docs/vision.html"), []byte("<html><body>Vision</body></html>"), 0o644)
	if out, err := e.gbi("artifacts", "entry", "demo", "vision", "docs/vision.html"); err != nil {
		t.Fatalf("vision entry: %v %s", err, out)
	}
	var tabs struct{ Tabs []struct{ ID, Title string } }
	e.gbiJSON(&tabs, "artifacts", "files", "demo")
	var order []string
	for _, x := range tabs.Tabs {
		order = append(order, x.ID)
	}
	if strings.Join(order, ",") != "requirements,mockups,plan,tracker,vision,compare,files" {
		t.Fatalf("tabs: %v", order)
	}
	// Vision first, no mockups: edit the file by hand, as the owner would.
	man, _ := os.ReadFile(filepath.Join(e.repo, "docs/gbi.toml"))
	edited := strings.Replace(strings.Replace(string(man), "vision       = \"docs/vision.html\"\n", "", 1), "[entries]\n", "[entries]\nvision = \"docs/vision.html\"\n", 1)
	edited = strings.Replace(edited, "mockups      = \"docs/mockups/index.html\"\n", "", 1)
	os.WriteFile(filepath.Join(e.repo, "docs/gbi.toml"), []byte(edited), 0o644)
	e.gbiJSON(&tabs, "artifacts", "files", "demo")
	order = nil
	for _, x := range tabs.Tabs {
		order = append(order, x.ID)
	}
	if strings.Join(order, ",") != "vision,requirements,plan,tracker,files" {
		t.Fatalf("tabs after the hand edit: %v\n%s", order, edited)
	}
	if out, err := e.gbi("artifacts", "label", "demo", "docs/research/", "--label", "research"); err != nil {
		t.Fatalf("label: %v %s", err, out)
	}
	if b, _ := os.ReadFile(filepath.Join(e.repo, "docs/gbi.toml")); !strings.Contains(string(b), "[entries]\nvision ") || strings.Contains(string(b), "mockups      =") {
		t.Fatalf("gbi lost the owner's layout when it rewrote the file:\n%s", b)
	}

	// The viewer's raw files: sandboxed, fetchable by the page, HTML gets gbi-nav.js.
	mux := http.NewServeMux()
	mux.HandleFunc("GET /raw/{id}/{path...}", e.a.RawHandler)
	mux.Handle("GET /gbi-assets/", e.a.AssetsHandler())
	srv := httptest.NewServer(mux)
	defer srv.Close()
	get := func(p string) (*http.Response, string) {
		resp, err := http.Get(srv.URL + p)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		b, _ := io.ReadAll(resp.Body)
		return resp, string(b)
	}
	resp, body := get("/raw/demo/docs/requirements.html")
	if !strings.Contains(resp.Header.Get("Content-Security-Policy"), "sandbox") || resp.Header.Get("Access-Control-Allow-Origin") != "*" || !strings.Contains(body, "/gbi-assets/gbi-nav.js") || !strings.Contains(body, `<gbi-md src="requirements.md">`) {
		t.Fatalf("raw html: %v\n%s", resp.Header, body)
	}
	if _, body := get("/raw/demo/docs/research/notes.md?gbi=md"); !strings.Contains(body, `<gbi-md src="notes.md">`) {
		t.Fatalf("rendered markdown page: %s", body)
	}
	if resp, body := get("/gbi-assets/gbi-md.js"); resp.StatusCode != 200 || !strings.Contains(body, "customElements.define('gbi-md'") {
		t.Fatalf("gbi-md.js: %d", resp.StatusCode)
	}
}

// A new installation starts with the starter archetypes and the starter standards; an
// installation that already has standards is not reseeded.
func TestNewInstallationDefaults(t *testing.T) {
	e := setup(t)
	if _, err := e.a.DB.Exec(`DELETE FROM standards`); err != nil { // setup saves one; start empty
		t.Fatal(err)
	}
	if n := e.a.SeedStandards(e.ctx); n != 14 {
		t.Fatalf("seeded %d standards, want 13 starters + Project documents", n)
	}
	stds, err := e.a.Standards.List(e.ctx)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]string{}
	for _, s := range stds {
		got[s.ID] = s.Title
	}
	for _, id := range append([]string{"project-documents"}, docs.DefaultStandardIDs...) {
		if got[id] == "" || got[id] == id {
			t.Errorf("standard %s missing or untitled: %q", id, got[id])
		}
	}
	if got["backend"] != "" {
		t.Error("an older standard that is not in the starter set was seeded")
	}
	if n := e.a.SeedStandards(e.ctx); n != 0 {
		t.Fatalf("an existing installation was reseeded: %d", n)
	}
	as, _ := e.a.Team.Archetypes(e.ctx)
	if len(as) != 6 {
		t.Fatalf("archetypes: %d", len(as))
	}
	for _, ar := range as {
		if !strings.Contains(ar.Instructions, "## Never") {
			t.Errorf("%s does not carry the starter instructions", ar.ID)
		}
	}
}
