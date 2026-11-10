// Package artifacts keeps a project's documents organised the same way in every project
// (FR-140–FR-150, ADR-014): HTML entry points for requirements, mockups, plan and tracker,
// explicit labels for every file under docs/, and the IDs that link them.
//
// Everything is stored in the repository in docs/gbi.toml. Nothing is inferred: a file
// has a label only because someone gave it one, and an entry point only because someone
// chose it.
package artifacts

import (
	"bytes"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
)

// ManifestPath is where entry points and labels live, relative to the repository root.
const ManifestPath = "docs/gbi.toml"

// ComparePath holds Compare decisions; TrackerPath the tracker snapshot.
const (
	ComparePath = "docs/compare.json"
	TrackerPath = "docs/tracker/tracker.json"
)

// Roles are the sub-tabs that open an entry point, in order.
var Roles = []string{"requirements", "mockups", "plan", "tracker"}

// Labels is the fixed list of labels a file may have.
var Labels = []string{"requirements", "mockup", "plan", "tracker", "screenshot", "decision", "research", "asset", "support"}

// roleLabel is the label an entry point's file gets when it is chosen.
var roleLabel = map[string]string{"requirements": "requirements", "mockups": "mockup", "plan": "plan", "tracker": "tracker"}

// Kept are files go-build-it itself writes; they never need a label.
var Kept = []string{ManifestPath, ComparePath}

// Manifest is docs/gbi.toml.
type Manifest struct {
	Entries map[string]string   `toml:"entries" json:"entries"`
	Labels  map[string][]string `toml:"labels" json:"labels"` // path (a folder ends in /) → labels
}

// Load reads docs/gbi.toml from the repository at root. A missing file is an empty manifest.
func Load(root string) (Manifest, error) {
	m := Manifest{Entries: map[string]string{}, Labels: map[string][]string{}}
	b, err := os.ReadFile(filepath.Join(root, ManifestPath))
	if errors.Is(err, fs.ErrNotExist) {
		return m, nil
	}
	if err != nil {
		return m, err
	}
	if err := toml.Unmarshal(b, &m); err != nil {
		return m, fmt.Errorf("%s: %w", ManifestPath, err)
	}
	if m.Entries == nil {
		m.Entries = map[string]string{}
	}
	if m.Labels == nil {
		m.Labels = map[string][]string{}
	}
	return m, nil
}

// Encode renders the manifest as TOML with a short header.
func (m Manifest) Encode() string {
	var b bytes.Buffer
	b.WriteString("# go-build-it project documents (FR-140–FR-150). Change with `gbi artifacts entry|add|label`.\n\n")
	b.WriteString("[entries]\n")
	for _, r := range Roles {
		if p := m.Entries[r]; p != "" {
			fmt.Fprintf(&b, "%-12s = %q\n", r, p)
		}
	}
	b.WriteString("\n[labels]\n")
	keys := make([]string, 0, len(m.Labels))
	for k := range m.Labels {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		q := make([]string, len(m.Labels[k]))
		for i, l := range m.Labels[k] {
			q[i] = fmt.Sprintf("%q", l)
		}
		fmt.Fprintf(&b, "%q = [%s]\n", k, strings.Join(q, ", "))
	}
	return b.String()
}

// Clean checks rel is a path under docs/ (slash-separated, no ..) and normalises it.
// A trailing slash marks a folder and is kept.
func Clean(rel string) (string, error) {
	rel = strings.ReplaceAll(strings.TrimSpace(rel), "\\", "/")
	dir := strings.HasSuffix(rel, "/")
	c := path.Clean(rel)
	if c == "." || strings.HasPrefix(c, "..") || path.IsAbs(c) {
		return "", fmt.Errorf("invalid path %q", rel)
	}
	if c != "docs" && !strings.HasPrefix(c, "docs/") {
		return "", fmt.Errorf("%s is not under docs/", rel)
	}
	if dir && c != "docs" {
		c += "/"
	}
	if c == "docs" {
		c = "docs/"
	}
	return c, nil
}

// ValidLabel reports whether l is in Labels.
func ValidLabel(l string) bool {
	for _, x := range Labels {
		if x == l {
			return true
		}
	}
	return false
}

// SetLabels gives the file or folder rel exactly labels (an empty list removes it).
// The file or folder must exist under root.
func (m *Manifest) SetLabels(root, rel string, labels []string) (string, error) {
	c, err := Clean(rel)
	if err != nil {
		return "", err
	}
	st, err := os.Stat(filepath.Join(root, filepath.FromSlash(strings.TrimSuffix(c, "/"))))
	if err != nil {
		return "", fmt.Errorf("%s does not exist", c)
	}
	if st.IsDir() && !strings.HasSuffix(c, "/") {
		c += "/"
	}
	if !st.IsDir() && strings.HasSuffix(c, "/") {
		return "", fmt.Errorf("%s is a file, not a folder", strings.TrimSuffix(c, "/"))
	}
	var out []string
	seen := map[string]bool{}
	for _, l := range labels {
		l = strings.ToLower(strings.TrimSpace(l))
		if l == "" || seen[l] {
			continue
		}
		if !ValidLabel(l) {
			return "", fmt.Errorf("unknown label %q (labels: %s)", l, strings.Join(Labels, ", "))
		}
		seen[l] = true
		out = append(out, l)
	}
	sort.Strings(out)
	if len(out) == 0 {
		delete(m.Labels, c)
	} else {
		m.Labels[c] = out
	}
	return c, nil
}

// SetEntry makes rel the entry point for role. It must be an existing .html file under
// docs/. The file also gets the role's label, so it is listed with its kind.
func (m *Manifest) SetEntry(root, role, rel string) (string, error) {
	if _, ok := roleLabel[role]; !ok {
		return "", fmt.Errorf("unknown role %q (roles: %s)", role, strings.Join(Roles, ", "))
	}
	c, err := Clean(rel)
	if err != nil {
		return "", err
	}
	if ext := strings.ToLower(path.Ext(c)); ext != ".html" && ext != ".htm" {
		return "", fmt.Errorf("an entry point must be an HTML file, not %s", c)
	}
	if st, err := os.Stat(filepath.Join(root, filepath.FromSlash(c))); err != nil || st.IsDir() {
		return "", fmt.Errorf("%s does not exist", c)
	}
	m.Entries[role] = c
	if l := roleLabel[role]; !contains(m.LabelsFor(c), l) {
		m.Labels[c] = sortedUnique(append(append([]string{}, m.Labels[c]...), l))
	}
	return c, nil
}

// LabelsFor returns the labels of the file rel: its own, plus those of every folder above it.
func (m Manifest) LabelsFor(rel string) []string {
	var out []string
	out = append(out, m.Labels[rel]...)
	for d := path.Dir(rel); d != "." && d != "/"; d = path.Dir(d) {
		out = append(out, m.Labels[d+"/"]...)
	}
	return sortedUnique(out)
}

// From returns the folder rel's labels come from, when they are not its own.
func (m Manifest) From(rel string) string {
	if len(m.Labels[rel]) > 0 {
		return ""
	}
	for d := path.Dir(rel); d != "." && d != "/"; d = path.Dir(d) {
		if len(m.Labels[d+"/"]) > 0 {
			return d + "/"
		}
	}
	return ""
}

// EntryRole returns the role whose entry point is rel, or "".
func (m Manifest) EntryRole(rel string) string {
	for _, r := range Roles {
		if m.Entries[r] == rel {
			return r
		}
	}
	return ""
}

// File is one file under docs/.
type File struct {
	Path     string   `json:"path"`
	Kind     string   `json:"kind"` // html | markdown | image | json | other
	Labels   []string `json:"labels"`
	From     string   `json:"from,omitempty"`  // the folder the labels come from
	Entry    string   `json:"entry,omitempty"` // the role this file is the entry point of
	Kept     bool     `json:"kept,omitempty"`  // written by go-build-it; needs no label
	Size     int64    `json:"size"`
	Modified string   `json:"modified"`
}

// Kind classifies a path by extension.
func Kind(p string) string {
	switch strings.ToLower(path.Ext(p)) {
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

// Scan lists every file under root/docs with its labels, skipping hidden folders and node_modules.
func Scan(root string, m Manifest) []File {
	var out []File
	base := filepath.Join(root, "docs")
	_ = filepath.WalkDir(base, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		name := d.Name()
		if d.IsDir() {
			if p != base && (strings.HasPrefix(name, ".") || name == "node_modules") {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasPrefix(name, ".") || len(out) >= 5000 {
			return nil
		}
		r, _ := filepath.Rel(root, p)
		rel := filepath.ToSlash(r)
		info, _ := d.Info()
		f := File{Path: rel, Kind: Kind(rel), Labels: m.LabelsFor(rel), From: m.From(rel), Entry: m.EntryRole(rel), Kept: contains(Kept, rel) || rel == TrackerPath}
		if info != nil {
			f.Size, f.Modified = info.Size(), info.ModTime().UTC().Format(time.RFC3339)
		}
		if f.Labels == nil {
			f.Labels = []string{}
		}
		out = append(out, f)
		return nil
	})
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out
}

// Unlabelled returns the files that need a label.
func Unlabelled(files []File) []File {
	var out []File
	for _, f := range files {
		if len(f.Labels) == 0 && !f.Kept {
			out = append(out, f)
		}
	}
	return out
}

func contains(xs []string, x string) bool {
	for _, y := range xs {
		if y == x {
			return true
		}
	}
	return false
}

func sortedUnique(xs []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, x := range xs {
		if !seen[x] {
			seen[x] = true
			out = append(out, x)
		}
	}
	sort.Strings(out)
	return out
}

//go:embed assets
var assets embed.FS

// Assets are the scripts served at /gbi-assets/ (gbi-md.js, gbi-nav.js).
func Assets() fs.FS {
	sub, _ := fs.Sub(assets, "assets")
	return sub
}
