// Package learn holds the lessons and their progress (FR-134–FR-136, ADR-012).
//
// Lessons are Markdown files in docs/learn, embedded in the binary:
//
//	title: Running work
//	minutes: 10
//	summary: The queue, sessions, and watching a member work in herdr.
//	# Running work
//	What the lesson is about.
//	> **Pro tip:** a tip for the whole lesson.
//	## Step: Add a task
//	check: event task.created
//	check: visit /projects/{practice}/queue
//	show: /projects/{practice}/queue [data-tour=add-task]
//	What to do, in Markdown.
//
// A step's checks say how go-build-it knows it is done: "event TYPE" is a history event of
// that type (TYPE may end in *) in the practice project after the lesson started ("event
// TYPE any" accepts any project); "visit PATH" is the user opening that view (a prefix). A
// step without checks is ticked off by the user. {practice} is the practice project's id.
package learn

import (
	"bufio"
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strconv"
	"strings"

	"github.com/techmuch/go-build-it/internal/store"
)

// PracticeID is the practice project's id.
const PracticeID = "practice"

// Check is one way a step gets done.
type Check struct {
	Kind string `json:"kind"` // event | visit
	Arg  string `json:"arg"`  // event type (may end in *) or a path prefix
	Any  bool   `json:"any,omitempty"`
}

// Show is where "Show me" goes and what it highlights.
type Show struct {
	Path     string `json:"path"`
	Selector string `json:"selector,omitempty"`
}

// Step is one numbered step.
type Step struct {
	N      int      `json:"n"`
	Title  string   `json:"title"`
	Body   string   `json:"body"`
	Checks []Check  `json:"checks"`
	Show   *Show    `json:"show,omitempty"`
	Tips   []string `json:"tips,omitempty"`
	// Filled in by Status.
	Done bool   `json:"done"`
	How  string `json:"how,omitempty"` // event | visit | tick
}

// Lesson is one lesson.
type Lesson struct {
	Slug    string   `json:"slug"`
	Order   string   `json:"order"`
	Number  int      `json:"number"`
	Title   string   `json:"title"`
	Minutes int      `json:"minutes"`
	Summary string   `json:"summary"`
	Desktop bool     `json:"desktopOnly,omitempty"`
	Intro   string   `json:"intro"`
	Tips    []string `json:"tips"`
	Steps   []Step   `json:"steps"`
	Related []string `json:"related,omitempty"` // docs slugs
	// Filled in by Status.
	Started string `json:"started,omitempty"`
	Done    int    `json:"done"`
	Total   int    `json:"total"`
}

// Parse reads every lesson in fsys (NN-slug.md), ordered by file name.
func Parse(fsys fs.FS) ([]Lesson, error) {
	names, err := fs.Glob(fsys, "*.md")
	if err != nil {
		return nil, err
	}
	sort.Strings(names)
	var out []Lesson
	for _, n := range names {
		b, err := fs.ReadFile(fsys, n)
		if err != nil {
			return nil, err
		}
		l, err := parseOne(strings.TrimSuffix(path.Base(n), ".md"), string(b))
		if err != nil {
			return nil, fmt.Errorf("%s: %w", n, err)
		}
		out = append(out, l)
	}
	return out, nil
}

func parseOne(order, src string) (Lesson, error) {
	l := Lesson{Order: order, Slug: order}
	if i := strings.Index(order, "-"); i > 0 && i <= 3 {
		l.Slug = order[i+1:]
		l.Number, _ = strconv.Atoi(order[:i])
	}
	var cur *Step
	var body []string
	var tip []string
	flushTip := func() {
		if len(tip) == 0 {
			return
		}
		t := strings.TrimSpace(strings.Join(tip, "\n"))
		tip = nil
		if cur != nil {
			cur.Tips = append(cur.Tips, t)
		} else {
			l.Tips = append(l.Tips, t)
		}
	}
	flushBody := func() {
		flushTip()
		text := strings.TrimSpace(strings.Join(body, "\n"))
		body = nil
		if cur != nil {
			cur.Body = text
			l.Steps = append(l.Steps, *cur)
		} else {
			l.Intro = text
		}
	}
	head := true
	sc := bufio.NewScanner(strings.NewReader(src))
	sc.Buffer(make([]byte, 0, 64<<10), 1<<20)
	for sc.Scan() {
		line := sc.Text()
		if head {
			if k, v, ok := strings.Cut(line, ": "); ok && !strings.HasPrefix(line, "#") && !strings.Contains(k, " ") {
				switch k {
				case "title":
					l.Title = v
				case "minutes":
					l.Minutes, _ = strconv.Atoi(v)
				case "summary":
					l.Summary = v
				case "desktop":
					l.Desktop = v == "only"
				case "related":
					for _, r := range strings.Split(v, ",") {
						l.Related = append(l.Related, strings.TrimSpace(r))
					}
				}
				continue
			}
			head = false
		}
		switch {
		case strings.HasPrefix(line, "# ") && l.Intro == "" && cur == nil && len(body) == 0:
			if l.Title == "" {
				l.Title = strings.TrimPrefix(line, "# ")
			}
		case strings.HasPrefix(line, "## Step: "):
			flushBody()
			cur = &Step{N: len(l.Steps) + 1, Title: strings.TrimPrefix(line, "## Step: ")}
		case cur != nil && strings.HasPrefix(line, "check: "):
			f := strings.Fields(strings.TrimPrefix(line, "check: "))
			if len(f) < 2 || (f[0] != "event" && f[0] != "visit") {
				return l, fmt.Errorf("step %d: bad check %q", cur.N, line)
			}
			cur.Checks = append(cur.Checks, Check{Kind: f[0], Arg: f[1], Any: len(f) > 2 && f[2] == "any"})
		case cur != nil && strings.HasPrefix(line, "show: "):
			v := strings.TrimSpace(strings.TrimPrefix(line, "show: "))
			s := Show{Path: v}
			if i := strings.Index(v, " "); i > 0 {
				s.Path, s.Selector = v[:i], strings.TrimSpace(v[i+1:])
			}
			cur.Show = &s
		case strings.HasPrefix(line, "> **Pro tip:**"):
			flushTip()
			tip = append(tip, strings.TrimSpace(strings.TrimPrefix(line, "> **Pro tip:**")))
		case len(tip) > 0 && strings.HasPrefix(line, "> "):
			tip = append(tip, strings.TrimPrefix(line, "> "))
		default:
			flushTip()
			body = append(body, line)
		}
	}
	flushBody()
	if l.Title == "" || len(l.Steps) == 0 {
		return l, fmt.Errorf("a lesson needs a title and at least one step")
	}
	if len(l.Tips) == 0 && !hasStepTips(l) {
		return l, fmt.Errorf("a lesson needs at least one Pro tip")
	}
	l.Total = len(l.Steps)
	return l, sc.Err()
}

func hasStepTips(l Lesson) bool {
	for _, s := range l.Steps {
		if len(s.Tips) > 0 {
			return true
		}
	}
	return false
}

// Progress stores lesson starts and ticked or visited steps per user.
type Progress struct{ DB *sql.DB }

// Start (re)starts a lesson for user now: earlier ticks are cleared.
func (p Progress) Start(ctx context.Context, user, lesson string) error {
	tx, err := p.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `DELETE FROM learn_steps WHERE user=? AND lesson=?`, user, lesson); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO learn_started(user, lesson, started_at) VALUES(?,?,?)
		ON CONFLICT(user, lesson) DO UPDATE SET started_at=excluded.started_at`, user, lesson, store.Now()); err != nil {
		return err
	}
	return tx.Commit()
}

// Mark records a step as done by the user (how: tick or visit).
func (p Progress) Mark(ctx context.Context, user, lesson string, step int, how string) error {
	_, err := p.DB.ExecContext(ctx, `INSERT INTO learn_steps(user, lesson, step, how, at) VALUES(?,?,?,?,?)
		ON CONFLICT(user, lesson, step) DO NOTHING`, user, lesson, step, how, store.Now())
	return err
}

// Visit marks visit-checked steps of the user's started lessons whose path matches.
func (p Progress) Visit(ctx context.Context, user, visited string, lessons []Lesson) (int, error) {
	started, err := p.starts(ctx, user)
	if err != nil {
		return 0, err
	}
	n := 0
	for _, l := range lessons {
		if started[l.Slug] == "" {
			continue
		}
		for _, s := range l.Steps {
			for _, c := range s.Checks {
				if c.Kind == "visit" && visitMatches(c.Arg, visited) {
					if err := p.Mark(ctx, user, l.Slug, s.N, "visit"); err != nil {
						return n, err
					}
					n++
				}
			}
		}
	}
	return n, nil
}

func visitMatches(want, got string) bool {
	want = strings.ReplaceAll(want, "{practice}", PracticeID)
	got = strings.SplitN(got, "?", 2)[0]
	return got == want || strings.HasPrefix(got, strings.TrimSuffix(want, "/")+"/")
}

func (p Progress) starts(ctx context.Context, user string) (map[string]string, error) {
	rows, err := p.DB.QueryContext(ctx, `SELECT lesson, started_at FROM learn_started WHERE user=?`, user)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]string{}
	for rows.Next() {
		var l, at string
		if err := rows.Scan(&l, &at); err != nil {
			return nil, err
		}
		out[l] = at
	}
	return out, rows.Err()
}

// Status fills in Started, Done and each step's Done for user. Lessons not started have
// nothing done.
func (p Progress) Status(ctx context.Context, user string, ls []Lesson) ([]Lesson, error) {
	started, err := p.starts(ctx, user)
	if err != nil {
		return nil, err
	}
	out := make([]Lesson, len(ls))
	for i, l := range ls {
		l.Steps = append([]Step(nil), l.Steps...)
		l.Started, l.Done = started[l.Slug], 0
		if l.Started != "" {
			marks := map[int]string{}
			rows, err := p.DB.QueryContext(ctx, `SELECT step, how FROM learn_steps WHERE user=? AND lesson=?`, user, l.Slug)
			if err != nil {
				return nil, err
			}
			for rows.Next() {
				var n int
				var how string
				if err := rows.Scan(&n, &how); err == nil {
					marks[n] = how
				}
			}
			rows.Close()
			for j := range l.Steps {
				s := &l.Steps[j]
				if how := marks[s.N]; how != "" {
					s.Done, s.How = true, how
				}
				for _, c := range s.Checks {
					if s.Done || c.Kind != "event" {
						continue
					}
					ok, err := p.eventSince(ctx, c, l.Started)
					if err != nil {
						return nil, err
					}
					if ok {
						s.Done, s.How = true, "event"
					}
				}
				if s.Done {
					l.Done++
				}
			}
		}
		out[i] = l
	}
	return out, nil
}

func (p Progress) eventSince(ctx context.Context, c Check, since string) (bool, error) {
	q := `SELECT 1 FROM events WHERE ts >= ?`
	args := []any{since}
	if strings.HasSuffix(c.Arg, "*") {
		q += ` AND type LIKE ?`
		args = append(args, strings.TrimSuffix(c.Arg, "*")+"%")
	} else {
		q += ` AND type = ?`
		args = append(args, c.Arg)
	}
	if !c.Any {
		q += ` AND project_id = ?`
		args = append(args, PracticeID)
	}
	var x int
	err := p.DB.QueryRowContext(ctx, q+` LIMIT 1`, args...).Scan(&x)
	if err == sql.ErrNoRows {
		return false, nil
	}
	return err == nil, err
}
