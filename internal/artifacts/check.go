package artifacts

import (
	"embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
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
	// Missing entry points: only for tabs the file asks for, or, with no tabs and no
	// entries at all, once for the whole project. A role left out is a choice.
	if len(m.Tabs) > 0 {
		for _, t := range m.Tabs {
			if t != TabCompare && t != TabFiles && m.Entries[t] == "" {
				out = append(out, Problem{Kind: "entry.missing", Detail: fmt.Sprintf("tabs lists %s but [entries] has no %s entry point: gbi artifacts entry <project> %s docs/<file>.html", t, t, t)})
			}
		}
	} else if len(m.EntryRoles()) == 0 {
		out = append(out, Problem{Kind: "entry.missing", Detail: "no entry points yet: gbi artifacts init <project>, or gbi artifacts entry <project> <role> docs/<file>.html"})
	}
	for _, r := range m.EntryRoles() {
		e := m.Entries[r]
		switch {
		case !ValidRole(r):
			out = append(out, Problem{Kind: "entry.invalid", Path: e, Detail: fmt.Sprintf("%q is not a valid role name (short lowercase; compare and files are built in)", r)})
		case !exists(root, e):
			out = append(out, Problem{Kind: "entry.invalid", Path: e, Detail: fmt.Sprintf("the %s entry point %s does not exist", r, e)})
		case Kind(e) != "html":
			out = append(out, Problem{Kind: "entry.invalid", Path: e, Detail: fmt.Sprintf("the %s entry point %s is not an HTML file", r, e)})
		}
	}
	for _, t := range m.Tabs {
		if t != TabCompare && t != TabFiles && !ValidRole(t) {
			out = append(out, Problem{Kind: "tab.unknown", Detail: fmt.Sprintf("tabs lists %q, which is not a valid tab", t)})
		}
	}
	for l := range labelSet(m) {
		if !ValidLabel(l) {
			out = append(out, Problem{Kind: "label.unknown", Detail: fmt.Sprintf("unknown label %q (labels: %s)", l, strings.Join(Labels, ", "))})
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
	files := map[string][]string{ // role → template files it brings
		"requirements": {"docs/requirements.html", "docs/requirements.md"},
		"mockups":      {"docs/mockups/index.html", "docs/screenshots/README.md"},
		"plan":         {"docs/plan.html", "docs/plan.md"},
		"tracker":      {"docs/tracker/index.html"},
	}
	entry := map[string]string{"requirements": "docs/requirements.html", "mockups": "docs/mockups/index.html", "plan": "docs/plan.html", "tracker": "docs/tracker/index.html"}
	want := map[string]bool{}
	for _, r := range m.WantedRoles() {
		want[r] = true
	}
	// Links in the templates to entry points the project does not use are removed.
	var drop []*regexp.Regexp
	for _, r := range Roles {
		if !want[r] {
			drop = append(drop, regexp.MustCompile(`<a[^>]*href="(\.\./)?`+regexp.QuoteMeta(strings.TrimPrefix(entry[r], "docs/"))+`"[^>]*>[^<]*</a>`))
		}
	}
	for _, r := range Roles {
		if !want[r] {
			continue
		}
		for _, rel := range files[r] {
			full := filepath.Join(root, filepath.FromSlash(rel))
			if _, err := os.Stat(full); err == nil {
				kept = append(kept, rel)
				continue
			}
			b, err := templates.ReadFile("templates/" + rel)
			if err != nil {
				return nil, nil, err
			}
			body := strings.ReplaceAll(string(b), "{{project}}", project)
			for _, re := range drop {
				body = re.ReplaceAllString(body, "")
			}
			if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
				return nil, nil, err
			}
			if err := os.WriteFile(full, []byte(body), 0o644); err != nil {
				return nil, nil, err
			}
			written = append(written, rel)
		}
		if m.Entries[r] == "" && exists(root, entry[r]) {
			if _, err := m.SetEntry(root, r, entry[r]); err != nil {
				return nil, nil, err
			}
		}
	}
	for rel, labels := range map[string][]string{"docs/requirements.md": {"requirements"}, "docs/plan.md": {"plan"}, "docs/mockups/": {"mockup"}, "docs/screenshots/": {"screenshot"}, "docs/tracker/": {"tracker"}} {
		if _, has := m.Labels[rel]; !has && exists(root, strings.TrimSuffix(rel, "/")) {
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

func labelSet(m Manifest) map[string]bool {
	out := map[string]bool{}
	for _, ls := range m.Labels {
		for _, l := range ls {
			out[l] = true
		}
	}
	return out
}
