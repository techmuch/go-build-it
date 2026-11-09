package learn

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/techmuch/go-build-it/docs"
	"github.com/techmuch/go-build-it/internal/access"
	"github.com/techmuch/go-build-it/internal/history"
	"github.com/techmuch/go-build-it/internal/store"
)

func TestEmbeddedLessons(t *testing.T) {
	ls, err := Parse(docs.LearnFS())
	if err != nil {
		t.Fatal(err)
	}
	if len(ls) != 11 {
		t.Fatalf("%d lessons, want 11 (0–10)", len(ls))
	}
	// Every event a lesson waits for must be one go-build-it records. A renamed event
	// should fail here, not leave a lesson that never completes (ADR-012).
	known := recordedEventTypes(t)
	for i, l := range ls {
		if l.Number != i || l.Title == "" || l.Minutes < 3 || l.Minutes > 12 || l.Summary == "" {
			t.Errorf("%s: number/title/minutes/summary: %+v", l.Order, l)
		}
		tips := len(l.Tips)
		for _, s := range l.Steps {
			tips += len(s.Tips)
			for _, c := range s.Checks {
				if c.Kind != "event" {
					continue
				}
				arg := strings.TrimSuffix(c.Arg, "*")
				ok := false
				for k := range known {
					if k == c.Arg || (strings.HasSuffix(c.Arg, "*") && strings.HasPrefix(k, arg)) {
						ok = true
					}
				}
				if !ok {
					t.Errorf("%s step %d waits for event %q, which nothing records", l.Order, s.N, c.Arg)
				}
			}
		}
		if tips < 1 || tips > 3 {
			t.Errorf("%s has %d Pro tips, want 1–3", l.Order, tips)
		}
	}
}

// recordedEventTypes collects the event types written in Go source (literal or prefix + status).
func recordedEventTypes(t *testing.T) map[string]bool {
	t.Helper()
	known := map[string]bool{}
	re := regexp.MustCompile(`Type: +"([a-z_.]+)"`)
	_ = filepath.Walk(filepath.Join("..", ".."), func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() && (info.Name() == "node_modules" || info.Name() == "web") || !strings.HasSuffix(p, ".go") || strings.HasSuffix(p, "_test.go") {
			return nil
		}
		b, _ := os.ReadFile(p)
		for _, m := range re.FindAllStringSubmatch(string(b), -1) {
			known[m[1]] = true
		}
		return nil
	})
	// Dynamic types: "<prefix>." + status.
	for _, x := range []string{"review.approved", "value.accepted", "value.proposed", "gate.approved", "terminal.control"} {
		if known[strings.SplitN(x, ".", 2)[0]+"."] || known[x] {
			known[x] = true
		}
	}
	if len(known) < 20 {
		t.Fatalf("found only %d event types", len(known))
	}
	return known
}

func TestParse(t *testing.T) {
	fsys := fstest.MapFS{"03-x.md": {Data: []byte(`title: X
minutes: 5
summary: S
# X
Intro.
> **Pro tip:** lesson tip
## Step: One
check: event task.created
check: visit /projects/{practice}/queue
show: /projects/{practice}/queue [data-tour=add-task]
Do it.
> **Pro tip:** step tip
> second line
## Step: Two
Tick me.
`)}}
	ls, err := Parse(fsys)
	if err != nil {
		t.Fatal(err)
	}
	l := ls[0]
	if l.Slug != "x" || l.Number != 3 || l.Intro != "Intro." || len(l.Tips) != 1 || len(l.Steps) != 2 {
		t.Fatalf("%+v", l)
	}
	s := l.Steps[0]
	if len(s.Checks) != 2 || s.Show == nil || s.Show.Selector != "[data-tour=add-task]" || s.Body != "Do it." || len(s.Tips) != 1 || s.Tips[0] != "step tip\nsecond line" {
		t.Fatalf("step %+v", s)
	}
	if _, err := Parse(fstest.MapFS{"01-y.md": {Data: []byte("title: Y\n# Y\n## Step: A\n")}}); err == nil {
		t.Fatal("a lesson without a Pro tip parsed")
	}
}

func TestProgressFromHistory(t *testing.T) {
	db, err := store.Open(filepath.Join(t.TempDir(), "gbi.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() }) // Windows cannot remove an open database file
	ctx := access.WithActor(context.Background(), access.Actor{Name: "david", Kind: "human", Origin: access.Local})
	h := history.New(db)
	ls, err := Parse(fstest.MapFS{"04-w.md": {Data: []byte(`title: W
minutes: 5
summary: S
# W
> **Pro tip:** t
## Step: Add a task
check: event task.created
## Step: Open the queue
check: visit /projects/{practice}/queue
## Step: Any project
check: event project.imported any
## Step: Tick
`)}})
	if err != nil {
		t.Fatal(err)
	}
	p := Progress{DB: db}
	// Before the lesson starts nothing counts, even matching events.
	h.Must(ctx, history.Event{ProjectID: PracticeID, Type: "task.created", Summary: "old"})
	st, _ := p.Status(ctx, "david", ls)
	if st[0].Done != 0 || st[0].Started != "" {
		t.Fatalf("not started: %+v", st[0])
	}
	if err := p.Start(ctx, "david", "w"); err != nil {
		t.Fatal(err)
	}
	// Events are compared by second; make the earlier one clearly older.
	if _, err := db.Exec(`UPDATE events SET ts='2000-01-01T00:00:00Z'`); err != nil {
		t.Fatal(err)
	}
	st, _ = p.Status(ctx, "david", ls)
	if st[0].Done != 0 {
		t.Fatalf("old event counted: %+v", st[0].Steps)
	}
	h.Must(ctx, history.Event{ProjectID: "elsewhere", Type: "task.created", Summary: "another project"})
	h.Must(ctx, history.Event{ProjectID: "mine", Type: "project.imported", Summary: "real project"})
	st, _ = p.Status(ctx, "david", ls)
	if st[0].Steps[0].Done || !st[0].Steps[2].Done {
		t.Fatalf("project scoping: %+v", st[0].Steps)
	}
	h.Must(ctx, history.Event{ProjectID: PracticeID, Type: "task.created", Summary: "new"})
	if n, _ := p.Visit(ctx, "david", "/projects/practice/queue?task=T-1", ls); n != 1 {
		t.Fatalf("visit marked %d", n)
	}
	if n, _ := p.Visit(ctx, "maria", "/projects/practice/queue", ls); n != 0 {
		t.Fatal("another user's visit counted: they have not started the lesson")
	}
	_ = p.Mark(ctx, "david", "w", 4, "tick")
	st, _ = p.Status(ctx, "david", ls)
	if st[0].Done != 4 || st[0].Steps[0].How != "event" || st[0].Steps[1].How != "visit" || st[0].Steps[3].How != "tick" {
		t.Fatalf("done %d: %+v", st[0].Done, st[0].Steps)
	}
	// Restarting clears ticks and visits.
	_ = p.Start(ctx, "david", "w")
	_, _ = db.Exec(`UPDATE events SET ts='2000-01-01T00:00:00Z'`)
	st, _ = p.Status(ctx, "david", ls)
	if st[0].Done != 0 {
		t.Fatalf("restart kept progress: %+v", st[0].Steps)
	}
}
