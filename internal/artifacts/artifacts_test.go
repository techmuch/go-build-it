package artifacts

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func write(t *testing.T, root, rel, body string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func kinds(ps []Problem) map[string]int {
	out := map[string]int{}
	for _, p := range ps {
		out[p.Kind]++
	}
	return out
}

func TestLabelsAndEntries(t *testing.T) {
	root := t.TempDir()
	write(t, root, "docs/mockups/index.html", "<html></html>")
	write(t, root, "docs/mockups/img/a.png", "png")
	write(t, root, "docs/notes.md", "# notes")
	write(t, root, "docs/req.md", "# r")
	m, _ := Load(root)

	if _, err := m.SetLabels(root, "docs/mockups", []string{"mockup"}); err != nil {
		t.Fatal(err)
	}
	if got := m.LabelsFor("docs/mockups/img/a.png"); len(got) != 1 || got[0] != "mockup" || m.From("docs/mockups/img/a.png") != "docs/mockups/" {
		t.Fatalf("folder label not inherited: %v from %q", got, m.From("docs/mockups/img/a.png"))
	}
	for _, bad := range []struct{ path, label string }{{"docs/nope.md", "research"}, {"README.md", "support"}, {"../x", "support"}, {"docs/notes.md", "guess"}} {
		if _, err := m.SetLabels(root, bad.path, []string{bad.label}); err == nil {
			t.Errorf("SetLabels(%s, %s) should fail", bad.path, bad.label)
		}
	}
	if _, err := m.SetEntry(root, "requirements", "docs/req.md"); err == nil || !strings.Contains(err.Error(), "HTML") {
		t.Fatalf("a Markdown entry point should be refused: %v", err)
	}
	if _, err := m.SetEntry(root, "mockups", "docs/mockups/index.html"); err != nil {
		t.Fatal(err)
	}
	if _, err := m.SetEntry(root, "roadmap", "docs/mockups/index.html"); err == nil {
		t.Fatal("unknown role accepted")
	}
	// Round trip through TOML.
	write(t, root, ManifestPath, m.Encode())
	m2, err := Load(root)
	if err != nil || m2.Entries["mockups"] != "docs/mockups/index.html" || len(m2.Labels["docs/mockups/"]) != 1 {
		t.Fatalf("round trip: %+v %v", m2, err)
	}
	files := Scan(root, m2)
	un := Unlabelled(files)
	if len(un) != 2 || un[0].Path != "docs/notes.md" || un[1].Path != "docs/req.md" {
		t.Fatalf("unlabelled: %+v", un)
	}
	for _, f := range files {
		if f.Path == ManifestPath && !f.Kept {
			t.Fatal("gbi.toml needs no label")
		}
		if f.Path == "docs/mockups/index.html" && f.Entry != "mockups" {
			t.Fatalf("entry not marked: %+v", f)
		}
	}
	// Removing every label removes the key.
	if _, err := m2.SetLabels(root, "docs/mockups/", nil); err != nil || len(m2.Labels) != 0 {
		t.Fatalf("remove: %v %v", m2.Labels, err)
	}
}

func project(t *testing.T) (string, Manifest) {
	root := t.TempDir()
	m, _ := Load(root)
	written, _, err := Init(root, "trailmap", &m)
	if err != nil || len(written) < 6 {
		t.Fatalf("init: %v %v", written, err)
	}
	write(t, root, "docs/requirements.md", "# R\n\n- **FR-07**: The map SHALL work **offline**.\n- **FR-08**: Download per trail.\n* **NFR-01**: Fast.\n")
	write(t, root, "docs/mockups/index.html", `<a href="phone.html">Phone</a> <img src="img/missing.png">
<section data-view="v03" data-route="/trail/:id" data-req="FR-07 FR-08"><h2>v03 · Trail <i>map</i></h2></section>
<section data-view="v04" data-req="FR-31" data-title="Recording"></section>`)
	write(t, root, "docs/mockups/phone.html", `<div data-view="v05" data-req="FR-08"><h3>Downloads</h3></div>`)
	write(t, root, "docs/screenshots/v03.png", "shot-1")
	write(t, root, "docs/screenshots/v05-iphone.png", "shot-5")
	return root, m
}

func TestInitNeverOverwrites(t *testing.T) {
	root := t.TempDir()
	write(t, root, "docs/requirements.md", "mine")
	m, _ := Load(root)
	written, kept, err := Init(root, "p", &m)
	if err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(filepath.Join(root, "docs/requirements.md")); string(b) != "mine" || len(kept) != 1 || kept[0] != "docs/requirements.md" {
		t.Fatalf("overwrote: %q kept %v", b, kept)
	}
	for _, w := range written {
		if strings.Contains(w, "requirements.md") {
			t.Fatal("wrote requirements.md")
		}
	}
	for _, r := range Roles {
		if m.Entries[r] == "" {
			t.Fatalf("no %s entry after init: %+v", r, m.Entries)
		}
	}
	if b, _ := os.ReadFile(filepath.Join(root, "docs/requirements.html")); !strings.Contains(string(b), `<gbi-md src="requirements.md">`) || !strings.Contains(string(b), "p · Requirements") {
		t.Fatalf("template: %s", b)
	}
}

func TestTraceCompareAndCheck(t *testing.T) {
	root, m := project(t)
	files := Scan(root, m)
	reqs := Requirements(root, m, files)
	if len(reqs) != 3 || reqs[0].ID != "FR-07" || reqs[0].Text != "The map SHALL work offline." || reqs[2].ID != "NFR-01" {
		t.Fatalf("requirements: %+v", reqs)
	}
	views := Views(root, files)
	if len(views) != 3 || views[0].ID != "v03" || views[0].Title != "v03 · Trail map" || views[0].Route != "/trail/:id" || len(views[0].Reqs) != 2 || views[1].Title != "Recording" {
		t.Fatalf("views: %+v", views)
	}
	shots := Screenshots(files, views)
	if shots["v03"] != "docs/screenshots/v03.png" || shots["v05"] != "docs/screenshots/v05-iphone.png" || shots["v04"] != "" {
		t.Fatalf("screenshots: %v", shots)
	}
	tasks := []Task{{ID: "T-1", Title: "Trail map v03", Status: "done", Reqs: []string{"FR-07"}}, {ID: "T-2", Title: "Downloads for FR-08", Status: "queued"}}
	tr, err := BuildTrace(root, m, files, tasks)
	if err != nil {
		t.Fatal(err)
	}
	if r := tr.Rows[0]; r.ID != "FR-07" || len(r.Tasks) != 1 || r.Done != 1 || len(r.Views) != 1 || len(r.Gaps) != 1 || r.Gaps[0] != "v03 not reviewed" {
		t.Fatalf("FR-07 row: %+v", r)
	}
	if r := tr.Rows[2]; r.ID != "NFR-01" || len(r.Gaps) != 1 || r.Gaps[0] != "no task" {
		t.Fatalf("NFR-01 row: %+v", r)
	}
	if len(tr.ViewGaps) != 1 || !strings.Contains(tr.ViewGaps[0], "FR-31") {
		t.Fatalf("view gaps: %v", tr.ViewGaps)
	}
	if got := TaskViews(tasks[0], views); len(got) != 1 || got[0] != "v03" {
		t.Fatalf("task views: %v", got)
	}

	// Compare: accepted, then the screenshot changes → drifted.
	dec := map[string]Decision{"v03": {Decision: "accepted", Screenshot: shots["v03"], SHA256: FileSHA(root, shots["v03"]), By: "david"}, "v05": {Decision: "task", Screenshot: shots["v05"], SHA256: FileSHA(root, shots["v05"]), Task: "T-9"}}
	write(t, root, ComparePath, EncodeDecisions(dec))
	d2, _ := LoadDecisions(root)
	st := map[string]string{}
	for _, v := range CompareViews(root, views, shots, d2) {
		st[v.ID] = v.State
	}
	if st["v03"] != StateAccepted || st["v04"] != StateMissing || st["v05"] != StateTask {
		t.Fatalf("states: %v", st)
	}
	write(t, root, "docs/screenshots/v03.png", "shot-2")
	for _, v := range CompareViews(root, views, shots, d2) {
		if v.ID == "v03" && v.State != StateDrifted {
			t.Fatalf("v03 should have drifted: %s", v.State)
		}
	}

	// Check: the broken image, the unknown requirement, the unlabelled file, the stale label.
	write(t, root, "docs/research/x.md", "x")
	m.Labels["docs/gone.md"] = []string{"research"}
	k := kinds(Check(root, m))
	if k["link.broken"] != 2 || k["id.unknown"] != 1 || k["file.unlabelled"] != 1 || k["label.stale"] != 1 || k["entry.missing"] != 0 {
		// link.broken: img/missing.png and tracker.json (no snapshot yet)
		t.Fatalf("check: %v\n%+v", k, Check(root, m))
	}
}
