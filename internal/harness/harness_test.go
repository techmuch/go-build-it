package harness

import (
	"bufio"
	"os"
	"strings"
	"testing"
)

func parseFile(t *testing.T, p Parser, path string) []Event {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	var out []Event
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 1<<20), 16<<20)
	for sc.Scan() {
		out = append(out, p.Parse(sc.Bytes())...)
	}
	return out
}

func TestClaudeFixture(t *testing.T) {
	evs := parseFile(t, Claude{}.NewParser(Spec{}), "testdata/claude-ok.jsonl")
	var text, result bool
	var last *Usage
	for _, e := range evs {
		switch e.Kind {
		case KText:
			text = text || e.Text == "OK"
		case KResult:
			result = true
		case KUsage:
			last = e.Usage
		}
	}
	if !text || !result || last == nil {
		t.Fatalf("text=%v result=%v usage=%v", text, result, last)
	}
	if last.Used != 2+10903+9075+4 {
		t.Errorf("used = %d", last.Used)
	}
}

func TestClaudeRateLimitAndCompaction(t *testing.T) {
	p := Claude{}.NewParser(Spec{})
	evs := p.Parse([]byte(`{"type":"rate_limit_event","rate_limit_info":{"status":"rejected","resetsAt":1794120000,"rateLimitType":"five_hour"}}`))
	if len(evs) != 1 || evs[0].Kind != KLimit || evs[0].Limit.ResetsAt.Unix() != 1794120000 {
		t.Fatalf("limit events: %+v", evs)
	}
	if evs := p.Parse([]byte(`{"type":"rate_limit_event","rate_limit_info":{"status":"allowed"}}`)); len(evs) != 0 {
		t.Fatalf("allowed should be silent: %+v", evs)
	}
	big := `{"type":"assistant","message":{"content":[],"usage":{"input_tokens":150000,"output_tokens":10}}}`
	small := `{"type":"assistant","message":{"content":[],"usage":{"input_tokens":30000,"output_tokens":10}}}`
	p.Parse([]byte(big))
	found := false
	for _, e := range p.Parse([]byte(small)) {
		found = found || e.Kind == KCompacted
	}
	if !found {
		t.Fatal("expected compaction to be detected")
	}
	if evs := p.Parse([]byte(`{"type":"system","subtype":"compact_boundary"}`)); len(evs) != 1 || evs[0].Kind != KCompacted {
		t.Fatalf("compact_boundary: %+v", evs)
	}
}

func TestAntigravityFixture(t *testing.T) {
	evs := parseFile(t, Antigravity{}.NewParser(Spec{}), "testdata/antigravity-ok.jsonl")
	var usage *Usage
	var result bool
	for _, e := range evs {
		if e.Usage != nil {
			usage = e.Usage
		}
		result = result || e.Kind == KResult
	}
	if usage == nil || usage.Measured || usage.Window != 1_000_000 || usage.Used != 9717+23 {
		t.Fatalf("usage = %+v", usage)
	}
	if !result {
		t.Fatal("no result")
	}
}

func TestOpenCodeParser(t *testing.T) {
	p := OpenCode{}.NewParser(Spec{Model: "anthropic/claude-sonnet-4"})
	evs := p.Parse([]byte(`{"type":"text","part":{"text":"hello"}}`))
	if len(evs) != 1 || evs[0].Text != "hello" {
		t.Fatalf("%+v", evs)
	}
	evs = p.Parse([]byte(`{"type":"step_finish","part":{"tokens":{"input":1000,"output":50,"cache":{"read":200}}}}`))
	var u *Usage
	var res bool
	for _, e := range evs {
		if e.Usage != nil {
			u = e.Usage
		}
		res = res || e.Kind == KResult
	}
	if u == nil || u.Used != 1250 || u.Measured || !res {
		t.Fatalf("usage %+v result %v", u, res)
	}
	evs = p.Parse([]byte(`{"type":"error","error":"429 Too Many Requests"}`))
	if evs[0].Kind != KLimit {
		t.Fatalf("expected limit: %+v", evs)
	}
}

func TestEstimateWindow(t *testing.T) {
	if EstimateWindow("gemini-3-pro", 0) != 1_000_000 || EstimateWindow("", 123) != 123 {
		t.Fatal("window table")
	}
}

func TestPiParser(t *testing.T) {
	evs := parseFile(t, Pi{}.NewParser(Spec{}), "testdata/pi-connection-error.jsonl")
	var retry bool
	for _, e := range evs {
		retry = retry || (e.Kind == KSystem && strings.Contains(e.Text, "Connection error"))
	}
	if !retry {
		t.Fatalf("expected retry notice from the recorded run: %+v", evs)
	}
	p := Pi{}.NewParser(Spec{Model: "gemini-3-pro"})
	evs = p.Parse([]byte(`{"type":"message_end","message":{"role":"assistant","content":[{"type":"text","text":"OK"}],"usage":{"input":1200,"output":10,"cacheRead":300,"cacheWrite":0,"totalTokens":1510}}}`))
	if len(evs) != 2 || evs[0].Text != "OK" || evs[1].Usage.Used != 1510 || evs[1].Usage.Window != 1_000_000 {
		t.Fatalf("%+v", evs)
	}
}
