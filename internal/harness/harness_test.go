package harness

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func kinds(es []Event) string {
	var k []string
	for _, e := range es {
		k = append(k, e.Kind)
	}
	return strings.Join(k, ",")
}

func lastUsage(es []Event) int {
	n := 0
	for _, e := range es {
		if e.Usage != nil {
			n = e.Usage.Used
		}
	}
	return n
}

func TestClaudeReader(t *testing.T) {
	p, _ := Get("claude")
	es, err := p.NewReader().Read(filepath.Join("testdata", "claude.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	if got := kinds(es); got != "user,tool,system,tool,system,compacted,limit,text,system" {
		t.Fatalf("kinds %s", got)
	}
	if lastUsage(es) != 2+247+42296 {
		t.Fatalf("usage %d", lastUsage(es))
	}
	if es[1].Text != "Read .gbi-task.md" || es[6].Limit == nil || es[6].Limit.ResetsAt.Hour() != 15 {
		t.Fatalf("events %+v", es)
	}
}

func TestReaderIsIncremental(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "t.jsonl")
	src, _ := os.ReadFile(filepath.Join("testdata", "claude.jsonl"))
	lines := strings.SplitAfter(string(src), "\n")
	os.WriteFile(f, []byte(lines[0]+lines[1]+lines[2][:20]), 0o644) // last line incomplete
	r := (&claudeReader{})
	es, _ := r.Read(f)
	if kinds(es) != "user,tool,system" {
		t.Fatalf("first read %s", kinds(es))
	}
	os.WriteFile(f, []byte(lines[0]+lines[1]+lines[2]), 0o644)
	es, _ = r.Read(f)
	if kinds(es) != "tool,system" {
		t.Fatalf("second read %s", kinds(es))
	}
	if es, _ = r.Read(f); len(es) != 0 {
		t.Fatalf("no new lines, got %d", len(es))
	}
}

func TestAgyAndPiReaders(t *testing.T) {
	a, _ := Get("antigravity")
	es, err := a.NewReader().Read(filepath.Join("testdata", "agy.jsonl"))
	if err != nil || kinds(es) != "user,text,tool,system" || lastUsage(es) != 10822 || es[0].Text != "Read .gbi-task.md" {
		t.Fatalf("agy %s %d %v %+v", kinds(es), lastUsage(es), err, es)
	}
	p, _ := Get("pi")
	es, err = p.NewReader().Read(filepath.Join("testdata", "pi.jsonl"))
	if err != nil || kinds(es) != "user,text,tool,system,limit" || lastUsage(es) != 3721 {
		t.Fatalf("pi %s %d %v", kinds(es), lastUsage(es), err)
	}
}

func TestLaunchArgsAndPaths(t *testing.T) {
	c, _ := Get("claude")
	if got := strings.Join(c.Args("opus", nil, "abc", false), " "); got != "--permission-mode acceptEdits --model opus --session-id abc" {
		t.Fatalf("claude new %q", got)
	}
	if got := strings.Join(c.Args("", nil, "abc", true), " "); got != "--permission-mode acceptEdits --resume abc" {
		t.Fatalf("claude resume %q", got)
	}
	if p := c.TranscriptPath("/tmp/x.y_z", "abc"); !strings.HasSuffix(p, filepath.Join("projects", "-tmp-x-y-z", "abc.jsonl")) && !strings.Contains(p, "-private-tmp-x-y-z") {
		t.Fatalf("claude path %s", p)
	}
	pi, _ := Get("pi")
	if got := strings.Join(pi.Args("", nil, "id1", false), " "); got != "--session-id id1" {
		t.Fatalf("pi %q", got)
	}
	agy, _ := Get("antigravity")
	if got := strings.Join(agy.Args("", nil, "conv", true), " "); got != "--mode accept-edits --conversation conv" {
		t.Fatalf("agy %q", got)
	}
	if id := c.NewConversationID(); len(id) != 36 || agy.NewConversationID() != "" {
		t.Fatalf("ids %q", id)
	}
	if len(Names()) != 4 {
		t.Fatalf("names %v", Names())
	}
}
