package update

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"
)

// Unreleased is the heading of work not yet released.
const Unreleased = "Unreleased"

// Section is one group of changes in an entry ("Added", "Fixed", …).
type Section struct {
	Title string   `json:"title"`
	Items []string `json:"items"`
}

// Entry is one version in the change log (FR-06a).
type Entry struct {
	Version  string    `json:"version"`
	Date     string    `json:"date,omitempty"`
	Sections []Section `json:"sections"`
	Body     string    `json:"body"` // the entry's Markdown, without its heading
}

var (
	headingRe = regexp.MustCompile(`^## \[([^\]]+)\](?:\s*-\s*(\d{4}-\d{2}-\d{2}))?\s*$`)
	calverRe  = regexp.MustCompile(`^\d{4}\.\d{2}\.\d{2}\.\d+$`)
)

// ParseChangelog reads a Keep a Changelog file: "## [VERSION] - DATE" headings,
// "### Group" subheadings and "- item" lines (continuation lines are joined).
func ParseChangelog(b []byte) []Entry {
	var out []Entry
	var cur *Entry
	var sec *Section
	var body []string
	flush := func() {
		if cur != nil {
			cur.Body = strings.TrimSpace(strings.Join(body, "\n"))
			out = append(out, *cur)
		}
	}
	sc := bufio.NewScanner(bytes.NewReader(b))
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for sc.Scan() {
		line := strings.TrimRight(sc.Text(), " \t")
		if m := headingRe.FindStringSubmatch(line); m != nil {
			flush()
			cur, sec, body = &Entry{Version: m[1], Date: m[2], Sections: []Section{}}, nil, nil
			continue
		}
		if cur == nil {
			continue
		}
		body = append(body, line)
		switch {
		case strings.HasPrefix(line, "### "):
			cur.Sections = append(cur.Sections, Section{Title: strings.TrimSpace(line[4:]), Items: []string{}})
			sec = &cur.Sections[len(cur.Sections)-1]
		case strings.HasPrefix(line, "- ") && sec != nil:
			sec.Items = append(sec.Items, strings.TrimSpace(line[2:]))
		case strings.HasPrefix(line, "  ") && sec != nil && len(sec.Items) > 0 && strings.TrimSpace(line) != "":
			sec.Items[len(sec.Items)-1] += " " + strings.TrimSpace(line)
		}
	}
	flush()
	return out
}

// CheckChangelog reports problems that would make the change log wrong in the app:
// versions that are not CalVer, out of order, duplicated, undated, or without changes.
func CheckChangelog(entries []Entry) []string {
	var errs []string
	seen := map[string]bool{}
	prev := ""
	for i, e := range entries {
		if seen[e.Version] {
			errs = append(errs, "duplicate entry "+e.Version)
		}
		seen[e.Version] = true
		if e.Version == Unreleased {
			if i != 0 {
				errs = append(errs, "Unreleased must be the first entry")
			}
			continue
		}
		if !calverRe.MatchString(e.Version) {
			errs = append(errs, fmt.Sprintf("%s is not a YYYY.MM.DD.N version", e.Version))
			continue
		}
		if e.Date == "" {
			errs = append(errs, e.Version+" has no date")
		} else if _, err := time.Parse("2006-01-02", e.Date); err != nil {
			errs = append(errs, e.Version+" has a bad date "+e.Date)
		}
		if len(e.Sections) == 0 {
			errs = append(errs, e.Version+" lists no changes")
		}
		if prev != "" && !Newer(prev, e.Version) {
			errs = append(errs, fmt.Sprintf("%s is listed after %s but is not older", e.Version, prev))
		}
		prev = e.Version
	}
	return errs
}

// Notes returns the Markdown body of one version's entry, for its GitHub release.
func Notes(entries []Entry, version string) (string, error) {
	for _, e := range entries {
		if e.Version == version {
			if strings.TrimSpace(e.Body) == "" {
				return "", fmt.Errorf("the change log entry for %s is empty", version)
			}
			return e.Body, nil
		}
	}
	return "", fmt.Errorf("CHANGELOG.md has no entry for %s; run `gbi dev changelog release %s` first", version, version)
}

// ReleaseChangelog turns the Unreleased entry of the change log file at path into a dated
// entry for version and opens a new, empty Unreleased entry above it.
func ReleaseChangelog(path, version string, today time.Time) error {
	if !calverRe.MatchString(version) {
		return fmt.Errorf("%s is not a YYYY.MM.DD.N version", version)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	entries := ParseChangelog(b)
	if len(entries) == 0 || entries[0].Version != Unreleased {
		return fmt.Errorf("%s has no Unreleased entry", path)
	}
	if len(entries[0].Sections) == 0 {
		return fmt.Errorf("the Unreleased entry lists no changes")
	}
	for _, e := range entries {
		if e.Version == version {
			return fmt.Errorf("%s is already in the change log", version)
		}
	}
	if len(entries) > 1 && !Newer(version, entries[1].Version) {
		return fmt.Errorf("%s is not newer than %s", version, entries[1].Version)
	}
	s := strings.Replace(string(b), "## [Unreleased]", "## [Unreleased]\n\n## ["+version+"] - "+today.Format("2006-01-02"), 1)
	return os.WriteFile(path, []byte(s), 0o644)
}
