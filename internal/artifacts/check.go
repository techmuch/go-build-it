package artifacts

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Problem is one finding of Check (FR-150).
type Problem struct {
	Kind   string `json:"kind"` // entry.missing | entry.invalid | label.stale | file.unlabelled | link.broken | id.unknown | view.duplicate
	Path   string `json:"path,omitempty"`
	Detail string `json:"detail"`
}

// Check reports what needs attention in a project's documents.
func Check(root string, m Manifest) []Problem {
	out := []Problem{}
	for _, r := range Roles {
		e := m.Entries[r]
		switch {
		case e == "":
			out = append(out, Problem{Kind: "entry.missing", Detail: fmt.Sprintf("no %s entry point: gbi artifacts entry <project> %s docs/<file>.html", r, r)})
		case !exists(root, e):
			out = append(out, Problem{Kind: "entry.invalid", Path: e, Detail: fmt.Sprintf("the %s entry point %s does not exist", r, e)})
		}
	}
	keys := make([]string, 0, len(m.Labels))
	for k := range m.Labels {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		if !exists(root, strings.TrimSuffix(k, "/")) {
			out = append(out, Problem{Kind: "label.stale", Path: k, Detail: k + " is labelled but does not exist"})
		}
	}
	files := Scan(root, m)
	for _, f := range Unlabelled(files) {
		out = append(out, Problem{Kind: "file.unlabelled", Path: f.Path, Detail: f.Path + " has no label: gbi artifacts label <project> " + f.Path + " --label …"})
	}
	for _, f := range files {
		if f.Kind != "html" || len(f.Labels) == 0 {
			continue
		}
		s := read(root, f.Path)
		seen := map[string]bool{}
		var links [][]string
		for _, tag := range openTagRe.FindAllString(s, -1) { // only real tags, not example text
			links = append(links, linkRe.FindAllStringSubmatch(tag, -1)...)
		}
		for _, mm := range links {
			r, ok := resolve(f.Path, mm[1])
			if !ok || seen[r] {
				continue
			}
			seen[r] = true
			if !exists(root, r) {
				out = append(out, Problem{Kind: "link.broken", Path: f.Path, Detail: fmt.Sprintf("%s links to %s, which is missing", f.Path, r)})
			}
		}
	}
	reqs := Requirements(root, m, files)
	known := map[string]bool{}
	for _, r := range reqs {
		known[r.ID] = true
	}
	views := Views(root, files)
	ids := map[string]string{}
	for _, v := range views {
		if prev, dup := ids[v.ID]; dup {
			out = append(out, Problem{Kind: "view.duplicate", Path: v.File, Detail: fmt.Sprintf("view %s is marked in %s and %s", v.ID, prev, v.File)})
		}
		ids[v.ID] = v.File
		if len(reqs) == 0 {
			continue
		}
		for _, id := range v.Reqs {
			if !known[id] {
				out = append(out, Problem{Kind: "id.unknown", Path: v.File, Detail: fmt.Sprintf("view %s names %s, which is not in the requirements", v.ID, id)})
			}
		}
	}
	return out
}

func exists(root, rel string) bool {
	_, err := os.Stat(filepath.Join(root, filepath.FromSlash(rel)))
	return err == nil
}

//go:embed templates
var templates embed.FS

// Init writes the standard set into root from templates (FR-149). Existing files are
// never overwritten. Entry points and labels are added to the manifest only for roles
// that have none yet. It returns the files it wrote and the ones it kept.
func Init(root, project string, m *Manifest) (written, kept []string, err error) {
	err = fs.WalkDir(templates, "templates", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel := strings.TrimPrefix(p, "templates/")
		full := filepath.Join(root, filepath.FromSlash(rel))
		if _, err := os.Stat(full); err == nil {
			kept = append(kept, rel)
			return nil
		}
		b, err := templates.ReadFile(p)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(full, []byte(strings.ReplaceAll(string(b), "{{project}}", project)), 0o644); err != nil {
			return err
		}
		written = append(written, rel)
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	for role, rel := range map[string]string{"requirements": "docs/requirements.html", "mockups": "docs/mockups/index.html", "plan": "docs/plan.html", "tracker": "docs/tracker/index.html"} {
		if m.Entries[role] == "" && exists(root, rel) {
			if _, err := m.SetEntry(root, role, rel); err != nil {
				return nil, nil, err
			}
		}
	}
	for rel, labels := range map[string][]string{"docs/requirements.md": {"requirements"}, "docs/plan.md": {"plan"}, "docs/mockups/": {"mockup"}, "docs/screenshots/": {"screenshot"}, "docs/tracker/": {"tracker"}} {
		key := rel
		if _, has := m.Labels[key]; !has && exists(root, strings.TrimSuffix(rel, "/")) {
			if _, err := m.SetLabels(root, rel, labels); err != nil {
				return nil, nil, err
			}
		}
	}
	sort.Strings(written)
	sort.Strings(kept)
	return written, kept, nil
}

// Snapshot is docs/tracker/tracker.json (FR-148).
type Snapshot struct {
	Project    string           `json:"project"`
	Generated  string           `json:"generated"`
	Milestones []map[string]any `json:"milestones"`
	Tasks      []map[string]any `json:"tasks"`
	Done       int              `json:"done"`
	Total      int              `json:"total"`
}

// EncodeSnapshot renders the snapshot.
func EncodeSnapshot(s Snapshot) string {
	b, _ := json.MarshalIndent(s, "", "  ")
	return string(b) + "\n"
}
