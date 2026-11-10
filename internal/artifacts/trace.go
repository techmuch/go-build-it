package artifacts

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"html"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// IDs (FR-146). Requirements are FR-nn / NFR-nn; mockup views are marked data-view.
var (
	reqMD     = regexp.MustCompile(`(?m)^\s*(?:[-*+]\s+)?\*\*((?:FR|NFR)-\d+[a-z]?)\*\*:?\s*(.*)$`)
	reqHTMLID = regexp.MustCompile(`(?s)<[a-zA-Z][^>]*\bid="((?:FR|NFR)-\d+[a-z]?)"[^>]*>(.{0,400}?)</`)
	reqHTMLB  = regexp.MustCompile(`<(?:b|strong)>((?:FR|NFR)-\d+[a-z]?)</(?:b|strong)>:?\s*([^<]{0,300})`)
	reqRef    = regexp.MustCompile(`\b((?:FR|NFR)-\d+[a-z]?)\b`)
	viewTag   = regexp.MustCompile(`<[a-zA-Z][^>]*\bdata-view="([^"]+)"[^>]*>`)
	attrRe    = regexp.MustCompile(`\b(data-req|data-route|data-title)="([^"]*)"`)
	headingRe = regexp.MustCompile(`(?s)<h[1-6][^>]*>(.*?)</h[1-6]>`)
	tagRe     = regexp.MustCompile(`<[^>]+>`)
	gbiMDRe   = regexp.MustCompile(`<gbi-md[^>]*\bsrc="([^"]+)"`)
	linkRe    = regexp.MustCompile(`\s(?:href|src)="([^"]+)"`)
	openTagRe = regexp.MustCompile(`<[a-zA-Z][^<>]*>`)
	viewRefRe = regexp.MustCompile(`\bv\d+[a-z]?\b`)
	spacesRe  = regexp.MustCompile(`\s+`)
)

// Requirement is one requirement found in the requirements documents.
type Requirement struct {
	ID    string `json:"id"`
	Text  string `json:"text"`
	Where string `json:"where"`
}

// View is one mockup view.
type View struct {
	ID    string   `json:"id"`
	Title string   `json:"title"`
	Route string   `json:"route,omitempty"`
	Reqs  []string `json:"reqs"`
	File  string   `json:"file"`
}

// Task is the part of a queue task traceability needs.
type Task struct {
	ID          string
	Title       string
	Description string
	Status      string
	Reqs        []string
}

func plain(s string) string {
	s = html.UnescapeString(tagRe.ReplaceAllString(s, " "))
	return strings.TrimSpace(spacesRe.ReplaceAllString(s, " "))
}

func read(root, rel string) string {
	b, _ := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
	return string(b)
}

// requirementFiles are the requirements entry, the Markdown it includes, and every file
// labelled requirements.
func requirementFiles(root string, m Manifest, files []File) []string {
	seen := map[string]bool{}
	var out []string
	add := func(p string) {
		if p != "" && !seen[p] {
			seen[p] = true
			out = append(out, p)
		}
	}
	if e := m.Entries["requirements"]; e != "" {
		add(e)
		for _, mm := range gbiMDRe.FindAllStringSubmatch(read(root, e), -1) {
			if r, ok := resolve(e, mm[1]); ok {
				add(r)
			}
		}
	}
	for _, f := range files {
		if contains(f.Labels, "requirements") && (f.Kind == "markdown" || f.Kind == "html") {
			add(f.Path)
		}
	}
	return out
}

// Requirements parses the requirement IDs (first definition wins).
func Requirements(root string, m Manifest, files []File) []Requirement {
	var out []Requirement
	seen := map[string]bool{}
	for _, f := range requirementFiles(root, m, files) {
		s := read(root, f)
		var found [][]string
		if Kind(f) == "markdown" {
			found = reqMD.FindAllStringSubmatch(s, -1)
		} else {
			found = append(reqHTMLID.FindAllStringSubmatch(s, -1), reqHTMLB.FindAllStringSubmatch(s, -1)...)
		}
		for _, mm := range found {
			if seen[mm[1]] {
				continue
			}
			seen[mm[1]] = true
			t := plain(strings.ReplaceAll(mm[2], "**", ""))
			if len(t) > 160 {
				t = t[:157] + "…"
			}
			out = append(out, Requirement{ID: mm[1], Text: t, Where: f})
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return idLess(out[i].ID, out[j].ID) })
	return out
}

// Views parses data-view marks in HTML files labelled mockup.
func Views(root string, files []File) []View {
	var out []View
	for _, f := range files {
		if f.Kind != "html" || !contains(f.Labels, "mockup") {
			continue
		}
		s := read(root, f.Path)
		for _, loc := range viewTag.FindAllStringSubmatchIndex(s, -1) {
			tag := s[loc[0]:loc[1]]
			v := View{ID: s[loc[2]:loc[3]], File: f.Path, Reqs: []string{}}
			for _, a := range attrRe.FindAllStringSubmatch(tag, -1) {
				switch a[1] {
				case "data-req":
					v.Reqs = append(v.Reqs, reqRef.FindAllString(a[2], -1)...)
				case "data-route":
					v.Route = a[2]
				case "data-title":
					v.Title = html.UnescapeString(a[2])
				}
			}
			if v.Title == "" {
				end := loc[1] + 3000
				if end > len(s) {
					end = len(s)
				}
				if h := headingRe.FindStringSubmatch(s[loc[1]:end]); h != nil {
					v.Title = plain(h[1])
				}
			}
			out = append(out, v)
		}
	}
	return out
}

// Screenshot is an image labelled screenshot that belongs to a view (named after it).
func Screenshots(files []File, views []View) map[string]string {
	out := map[string]string{}
	for _, f := range files {
		if f.Kind != "image" || !contains(f.Labels, "screenshot") {
			continue
		}
		base := strings.TrimSuffix(path.Base(f.Path), path.Ext(f.Path))
		for _, v := range views {
			if base == v.ID || strings.HasPrefix(base, v.ID+"-") || strings.HasPrefix(base, v.ID+"_") || strings.HasPrefix(base, v.ID+".") {
				if _, ok := out[v.ID]; !ok || base == v.ID {
					out[v.ID] = f.Path
				}
			}
		}
	}
	return out
}

// Decision is one Compare decision, stored in docs/compare.json (FR-147).
type Decision struct {
	Decision   string `json:"decision"` // accepted | task
	Screenshot string `json:"screenshot"`
	SHA256     string `json:"sha256"`
	Note       string `json:"note,omitempty"`
	Task       string `json:"task,omitempty"`
	By         string `json:"by"`
	At         string `json:"at"`
}

// LoadDecisions reads docs/compare.json.
func LoadDecisions(root string) (map[string]Decision, error) {
	out := map[string]Decision{}
	b, err := os.ReadFile(filepath.Join(root, ComparePath))
	if errors.Is(err, fs.ErrNotExist) {
		return out, nil
	}
	if err != nil {
		return out, err
	}
	return out, json.Unmarshal(b, &out)
}

// EncodeDecisions renders compare.json.
func EncodeDecisions(d map[string]Decision) string {
	b, _ := json.MarshalIndent(d, "", "  ")
	return string(b) + "\n"
}

// FileSHA returns the SHA-256 of a file under root.
func FileSHA(root, rel string) string {
	b, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil {
		return ""
	}
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

// Compare states.
const (
	StateMissing  = "no-screenshot"
	StateNew      = "not-reviewed"
	StateAccepted = "accepted"
	StateDrifted  = "drifted"
	StateTask     = "task"
)

// ViewState is one row of Compare.
type ViewState struct {
	View
	Screenshot string    `json:"screenshot,omitempty"`
	State      string    `json:"state"`
	Decision   *Decision `json:"decision,omitempty"`
}

// CompareViews pairs each view with its screenshot and decision.
func CompareViews(root string, views []View, shots map[string]string, dec map[string]Decision) []ViewState {
	var out []ViewState
	for _, v := range views {
		vs := ViewState{View: v, Screenshot: shots[v.ID]}
		d, has := dec[v.ID]
		if has {
			dd := d
			vs.Decision = &dd
		}
		switch {
		case vs.Screenshot == "":
			vs.State = StateMissing
		case !has:
			vs.State = StateNew
		case d.SHA256 != FileSHA(root, vs.Screenshot):
			vs.State = StateDrifted
		case d.Decision == "task":
			vs.State = StateTask
		default:
			vs.State = StateAccepted
		}
		out = append(out, vs)
	}
	return out
}

// TraceRow is one requirement's chain (FR-146).
type TraceRow struct {
	Requirement
	Views   []string `json:"views"`
	Tasks   []string `json:"tasks"`
	Done    int      `json:"done"`
	Screens []string `json:"screens"` // view:state
	Gaps    []string `json:"gaps"`
}

// Trace is the traceability report.
type Trace struct {
	Rows     []TraceRow  `json:"rows"`
	Views    []ViewState `json:"views"`
	ViewGaps []string    `json:"viewGaps"` // views naming unknown requirements
	GapCount int         `json:"gaps"`
}

// BuildTrace links requirements, views, tasks and screenshots.
func BuildTrace(root string, m Manifest, files []File, tasks []Task) (Trace, error) {
	reqs := Requirements(root, m, files)
	views := Views(root, files)
	dec, err := LoadDecisions(root)
	if err != nil {
		return Trace{}, err
	}
	states := CompareViews(root, views, Screenshots(files, views), dec)
	known := map[string]bool{}
	for _, r := range reqs {
		known[r.ID] = true
	}
	viewIDs := map[string]bool{}
	for _, v := range views {
		viewIDs[v.ID] = true
	}
	tr := Trace{Views: states, Rows: []TraceRow{}, ViewGaps: []string{}}
	for _, r := range reqs {
		row := TraceRow{Requirement: r, Views: []string{}, Tasks: []string{}, Screens: []string{}, Gaps: []string{}}
		for _, vs := range states {
			if contains(vs.Reqs, r.ID) {
				row.Views = append(row.Views, vs.ID)
				row.Screens = append(row.Screens, vs.ID+":"+vs.State)
				if vs.State == StateDrifted || vs.State == StateNew {
					row.Gaps = append(row.Gaps, vs.ID+" "+strings.ReplaceAll(vs.State, "-", " "))
				}
			}
		}
		for _, t := range tasks {
			if taskNames(t, r.ID) {
				row.Tasks = append(row.Tasks, t.ID)
				if t.Status == "done" {
					row.Done++
				}
			}
		}
		if len(row.Tasks) == 0 {
			row.Gaps = append(row.Gaps, "no task")
		}
		tr.GapCount += len(row.Gaps)
		tr.Rows = append(tr.Rows, row)
	}
	if len(reqs) > 0 {
		for _, v := range views {
			for _, id := range v.Reqs {
				if !known[id] {
					tr.ViewGaps = append(tr.ViewGaps, v.ID+" names "+id+", which is not a requirement")
				}
			}
		}
	}
	tr.GapCount += len(tr.ViewGaps)
	return tr, nil
}

func taskNames(t Task, id string) bool {
	if contains(t.Reqs, id) {
		return true
	}
	for _, m := range reqRef.FindAllString(t.Title+" "+t.Description, -1) {
		if m == id {
			return true
		}
	}
	return false
}

// TaskViews returns the view IDs a task names in its text.
func TaskViews(t Task, views []View) []string {
	known := map[string]bool{}
	for _, v := range views {
		known[v.ID] = true
	}
	var out []string
	for _, m := range viewRefRe.FindAllString(t.Title+" "+t.Description, -1) {
		if known[m] && !contains(out, m) {
			out = append(out, m)
		}
	}
	return out
}

// idLess orders FR-2 before FR-10, and FR before NFR.
func idLess(a, b string) bool {
	pa, na, sa := splitID(a)
	pb, nb, sb := splitID(b)
	if pa != pb {
		return pa < pb
	}
	if na != nb {
		return na < nb
	}
	return sa < sb
}

func splitID(s string) (string, int, string) {
	p, rest, _ := strings.Cut(s, "-")
	n := 0
	i := 0
	for i < len(rest) && rest[i] >= '0' && rest[i] <= '9' {
		n = n*10 + int(rest[i]-'0')
		i++
	}
	return p, n, rest[i:]
}

// resolve resolves a link in the document at from (both repository-relative). It reports
// false for external, absolute or anchor-only links.
func resolve(from, link string) (string, bool) {
	l := strings.TrimSpace(link)
	if l == "" || strings.HasPrefix(l, "#") || strings.HasPrefix(l, "/") || strings.Contains(l, "://") || strings.HasPrefix(l, "//") {
		return "", false
	}
	for _, p := range []string{"mailto:", "data:", "javascript:", "tel:", "{", "$"} {
		if strings.HasPrefix(strings.ToLower(l), p) {
			return "", false
		}
	}
	if i := strings.IndexAny(l, "?#"); i >= 0 {
		l = l[:i]
	}
	if l == "" {
		return "", false
	}
	r := path.Clean(path.Join(path.Dir(from), l))
	if strings.HasPrefix(r, "..") {
		return "", false
	}
	return r, true
}
