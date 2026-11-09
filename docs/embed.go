// Package docs embeds the user documentation shown inside the app (FR-130, FR-131).
package docs

import (
	"embed"
	"io/fs"
	"path"
	"sort"
	"strings"
)

//go:embed user
var content embed.FS

//go:embed learn agent-guide.md
var learnContent embed.FS

// LearnFS holds the lessons (docs/learn/*.md, FR-134).
func LearnFS() fs.FS { sub, _ := fs.Sub(learnContent, "learn"); return sub }

// AgentGuide is the guide for AI agents driving go-build-it (`gbi agent guide`, FR-137).
func AgentGuide() string { b, _ := learnContent.ReadFile("agent-guide.md"); return string(b) }

// Page is one documentation page.
type Page struct {
	Slug    string `json:"slug"`
	Title   string `json:"title"`
	Section string `json:"section"`
	Order   string `json:"order"`
	Body    string `json:"body,omitempty"`
	Reqs    string `json:"reqs,omitempty"`
}

// FS exposes the embedded docs (screenshots are served from user/screenshots).
func FS() fs.FS { sub, _ := fs.Sub(content, "user"); return sub }

// Pages lists all pages, ordered by file name. Front matter lines at the top:
// "section: …" and "reqs: …".
func Pages() []Page {
	var out []Page
	entries, _ := fs.ReadDir(content, "user")
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		p := parse(e.Name())
		p.Body = ""
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Order < out[j].Order })
	return out
}

func parse(name string) Page {
	b, _ := content.ReadFile(path.Join("user", name))
	order := strings.TrimSuffix(name, ".md")
	slug := order
	if i := strings.Index(order, "-"); i > 0 && i <= 3 {
		slug = order[i+1:]
	}
	p := Page{Slug: slug, Order: order, Title: slug}
	lines := strings.Split(string(b), "\n")
	var body []string
	for i, l := range lines {
		switch {
		case i < 4 && strings.HasPrefix(l, "section: "):
			p.Section = strings.TrimPrefix(l, "section: ")
		case i < 4 && strings.HasPrefix(l, "reqs: "):
			p.Reqs = strings.TrimPrefix(l, "reqs: ")
		case strings.HasPrefix(l, "# ") && p.Title == slug:
			p.Title = strings.TrimPrefix(l, "# ")
			body = append(body, l)
		default:
			body = append(body, l)
		}
	}
	p.Body = strings.TrimSpace(strings.Join(body, "\n"))
	return p
}

// Get returns a page by slug.
func Get(slug string) (Page, bool) {
	entries, _ := fs.ReadDir(content, "user")
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".md") {
			if p := parse(e.Name()); p.Slug == slug {
				return p, true
			}
		}
	}
	return Page{}, false
}

// Hit is a search result.
type Hit struct {
	Page    Page   `json:"page"`
	Score   int    `json:"score"`
	Snippet string `json:"snippet"`
}

// Search scores pages by term matches in title and body.
func Search(q string) []Hit {
	terms := strings.Fields(strings.ToLower(q))
	var hits []Hit
	if len(terms) == 0 {
		return []Hit{}
	}
	for _, meta := range Pages() {
		p, _ := Get(meta.Slug)
		title, body := strings.ToLower(p.Title), strings.ToLower(p.Body)
		score := 0
		snippet := ""
		for _, t := range terms {
			if strings.Contains(title, t) {
				score += 10
			}
			n := strings.Count(body, t)
			score += n
			if n > 0 && snippet == "" {
				i := strings.Index(body, t)
				start, end := max(0, i-60), min(len(p.Body), i+100)
				snippet = strings.TrimSpace(strings.ReplaceAll(p.Body[start:end], "\n", " "))
			}
		}
		if score > 0 {
			meta.Body = ""
			hits = append(hits, Hit{Page: meta, Score: score, Snippet: snippet})
		}
	}
	sort.Slice(hits, func(i, j int) bool { return hits[i].Score > hits[j].Score })
	if hits == nil {
		hits = []Hit{}
	}
	return hits
}
