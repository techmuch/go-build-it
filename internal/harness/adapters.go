package harness

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

func init() {
	Register(Claude{})
	Register(Antigravity{})
	Register(OpenCode{})
	Register(Fake{})
}

// ---------------------------------------------------------------- Claude Code

// Claude drives Claude Code: `claude -p --output-format stream-json --input-format stream-json --verbose`.
type Claude struct{}

func (Claude) Name() string          { return "claude" }
func (Claude) DefaultBinary() string { return "claude" }

func (Claude) Command(s Spec) (string, []string) {
	bin := s.Binary
	if bin == "" {
		bin = "claude"
	}
	args := []string{"-p", "--output-format", "stream-json", "--input-format", "stream-json", "--verbose", "--permission-mode", "acceptEdits"}
	if s.Model != "" {
		args = append(args, "--model", s.Model)
	}
	if s.SystemPrompt != "" {
		args = append(args, "--append-system-prompt", s.SystemPrompt)
	}
	return bin, append(args, s.Flags...)
}

func (Claude) InputLine(text string) ([]byte, bool) {
	return jsonLine(map[string]any{"type": "user", "message": map[string]any{"role": "user", "content": []map[string]string{{"type": "text", "text": text}}}}), true
}

func (Claude) Detection() Detection {
	return Detection{Name: "claude", Headless: "stream-json", UsageReported: "measured", Verified: true, SupportsInput: true}
}

func (Claude) NewParser(s Spec) Parser {
	return &claudeParser{window: EstimateWindow(s.Model, s.Window)}
}

type claudeParser struct {
	window   int
	measured bool
	comp     compactionDetector
}

type claudeLine struct {
	Type      string `json:"type"`
	Subtype   string `json:"subtype"`
	SessionID string `json:"session_id"`
	Model     string `json:"model"`
	IsError   bool   `json:"is_error"`
	Result    string `json:"result"`
	Message   struct {
		Content []struct {
			Type  string          `json:"type"`
			Text  string          `json:"text"`
			Name  string          `json:"name"`
			Input json.RawMessage `json:"input"`
		} `json:"content"`
		Usage *struct {
			Input         int `json:"input_tokens"`
			CacheCreation int `json:"cache_creation_input_tokens"`
			CacheRead     int `json:"cache_read_input_tokens"`
			Output        int `json:"output_tokens"`
		} `json:"usage"`
	} `json:"message"`
	RateLimit *struct {
		Status        string `json:"status"`
		ResetsAt      int64  `json:"resetsAt"`
		RateLimitType string `json:"rateLimitType"`
	} `json:"rate_limit_info"`
	ModelUsage map[string]struct {
		ContextWindow int `json:"contextWindow"`
	} `json:"modelUsage"`
}

func (p *claudeParser) usage(used int) *Usage {
	return &Usage{Used: used, Window: p.window, Measured: p.measured}
}

func (p *claudeParser) Parse(line []byte) []Event {
	var l claudeLine
	if json.Unmarshal(line, &l) != nil {
		return nil
	}
	var out []Event
	switch l.Type {
	case "system":
		switch l.Subtype {
		case "init":
			if l.Model != "" {
				p.window = EstimateWindow(l.Model, p.window)
			}
			out = append(out, Event{Kind: KSystem, Text: "session started · model " + l.Model, SessionID: l.SessionID})
		case "compact_boundary":
			out = append(out, Event{Kind: KCompacted, Text: "Claude Code compacted the conversation"})
		}
	case "assistant":
		for _, c := range l.Message.Content {
			switch c.Type {
			case "text":
				if strings.TrimSpace(c.Text) != "" {
					out = append(out, Event{Kind: KText, Text: c.Text})
				}
			case "tool_use":
				out = append(out, Event{Kind: KTool, Text: c.Name + " " + trunc(string(c.Input), 160)})
			}
		}
		if u := l.Message.Usage; u != nil {
			used := u.Input + u.CacheCreation + u.CacheRead + u.Output
			if p.comp.check(used) {
				out = append(out, Event{Kind: KCompacted, Text: "context dropped sharply: compaction detected"})
			}
			out = append(out, Event{Kind: KUsage, Usage: p.usage(used)})
		}
	case "user":
		for _, c := range l.Message.Content {
			if c.Type == "tool_result" {
				out = append(out, Event{Kind: KTool, Text: "tool result"})
			}
		}
	case "rate_limit_event":
		if r := l.RateLimit; r != nil && r.Status != "allowed" && r.Status != "" {
			out = append(out, Event{Kind: KLimit, Limit: &RateLimit{Limited: true, ResetsAt: time.Unix(r.ResetsAt, 0), Kind: r.RateLimitType}})
		}
	case "result":
		for _, mu := range l.ModelUsage {
			if mu.ContextWindow > 0 {
				p.window, p.measured = mu.ContextWindow, true
			}
		}
		e := Event{Kind: KResult, Text: l.Result, SessionID: l.SessionID, IsError: l.IsError}
		if l.IsError && LooksRateLimited(l.Result) {
			out = append(out, Event{Kind: KLimit, Limit: &RateLimit{Limited: true, ResetsAt: time.Now().Add(15 * time.Minute), Kind: "result"}})
		}
		out = append(out, e)
	}
	// Claude reports usage exactly; the window becomes measured once the result arrives.
	if !p.measured {
		for i := range out {
			if out[i].Usage != nil {
				out[i].Usage.Measured = false
			}
		}
	}
	return out
}

// ---------------------------------------------------------------- Antigravity

// Antigravity drives the `agy` CLI in print mode with stream-json output.
type Antigravity struct{}

func (Antigravity) Name() string          { return "antigravity" }
func (Antigravity) DefaultBinary() string { return "agy" }

func (Antigravity) Command(s Spec) (string, []string) {
	bin := s.Binary
	if bin == "" {
		bin = "agy"
	}
	prompt := s.Prompt
	if s.SystemPrompt != "" {
		prompt = s.SystemPrompt + "\n\n" + prompt
	}
	args := []string{"-p", prompt, "--output-format", "stream-json", "--mode", "accept-edits"}
	if s.Model != "" {
		args = append(args, "--model", s.Model)
	}
	return bin, append(args, s.Flags...)
}

func (Antigravity) InputLine(string) ([]byte, bool) { return nil, false }

func (Antigravity) Detection() Detection {
	return Detection{Name: "antigravity", Headless: "stream-json (agy -p)", UsageReported: "estimated", Verified: true}
}

func (Antigravity) NewParser(s Spec) Parser {
	return &agyParser{window: EstimateWindow(firstNonEmpty(s.Model, "gemini"), s.Window)}
}

type agyParser struct {
	window int
	total  int
	comp   compactionDetector
}

type agyUsage struct {
	Input     int `json:"input_tokens"`
	Output    int `json:"output_tokens"`
	Thinking  int `json:"thinking_tokens"`
	CacheRead int `json:"cache_read_tokens"`
	Total     int `json:"total_tokens"`
}

func (p *agyParser) Parse(line []byte) []Event {
	var l struct {
		Event          string `json:"event"`
		ConversationID string `json:"conversation_id"`
		StepUpdate     struct {
			State     string    `json:"state"`
			StepType  string    `json:"step_type"`
			TextDelta string    `json:"text_delta"`
			Usage     *agyUsage `json:"usage"`
			ToolName  string    `json:"tool_name"`
		} `json:"step_update"`
		Result struct {
			Status   string    `json:"status"`
			Response string    `json:"response"`
			Error    string    `json:"error"`
			Usage    *agyUsage `json:"usage"`
		} `json:"result"`
	}
	if json.Unmarshal(line, &l) != nil {
		return nil
	}
	var out []Event
	switch l.Event {
	case "init":
		out = append(out, Event{Kind: KSystem, Text: "antigravity session started", SessionID: l.ConversationID})
	case "step_update":
		su := l.StepUpdate
		if su.TextDelta != "" && su.StepType == "agent_response" {
			out = append(out, Event{Kind: KText, Text: su.TextDelta})
		}
		if su.ToolName != "" {
			out = append(out, Event{Kind: KTool, Text: su.ToolName})
		}
		if su.Usage != nil {
			used := su.Usage.Input + su.Usage.Output + su.Usage.CacheRead
			if p.comp.check(used) {
				out = append(out, Event{Kind: KCompacted, Text: "context dropped sharply: compaction detected"})
			}
			p.total = used
			out = append(out, Event{Kind: KUsage, Usage: &Usage{Used: used, Window: p.window}})
		}
	case "result":
		r := l.Result
		isErr := r.Status != "" && r.Status != "SUCCESS"
		if isErr && LooksRateLimited(r.Error+r.Response) {
			out = append(out, Event{Kind: KLimit, Limit: &RateLimit{Limited: true, ResetsAt: time.Now().Add(15 * time.Minute), Kind: "result"}})
		}
		out = append(out, Event{Kind: KResult, Text: firstNonEmpty(r.Response, r.Error), IsError: isErr})
	}
	return out
}

// ---------------------------------------------------------------- OpenCode

// OpenCode drives `opencode run --format json`. Unverified on the build machine (ADR-002).
type OpenCode struct{}

func (OpenCode) Name() string          { return "opencode" }
func (OpenCode) DefaultBinary() string { return "opencode" }

func (OpenCode) Command(s Spec) (string, []string) {
	bin := s.Binary
	if bin == "" {
		bin = "opencode"
	}
	prompt := s.Prompt
	if s.SystemPrompt != "" {
		prompt = s.SystemPrompt + "\n\n" + prompt
	}
	args := []string{"run", "--format", "json"}
	if s.Model != "" {
		args = append(args, "--model", s.Model)
	}
	return bin, append(append(args, s.Flags...), prompt)
}

func (OpenCode) InputLine(string) ([]byte, bool) { return nil, false }

func (OpenCode) Detection() Detection {
	return Detection{Name: "opencode", Headless: "run --format json", UsageReported: "estimated", Verified: false}
}

func (OpenCode) NewParser(s Spec) Parser {
	return &openCodeParser{window: EstimateWindow(s.Model, s.Window)}
}

// openCodeParser sums token counts per message and estimates the window (Q-34 answer A).
type openCodeParser struct {
	window int
	used   int
	comp   compactionDetector
}

func (p *openCodeParser) Parse(line []byte) []Event {
	var m map[string]any
	if json.Unmarshal(line, &m) != nil {
		return nil
	}
	var out []Event
	typ, _ := m["type"].(string)
	part, _ := m["part"].(map[string]any)
	switch {
	case typ == "text" && part != nil:
		if t, _ := part["text"].(string); t != "" {
			out = append(out, Event{Kind: KText, Text: t})
		}
	case typ == "tool_use" || typ == "tool":
		name := ""
		if part != nil {
			name, _ = part["tool"].(string)
		}
		out = append(out, Event{Kind: KTool, Text: firstNonEmpty(name, "tool")})
	case typ == "error":
		msg := fmt.Sprint(m["error"])
		if LooksRateLimited(msg) {
			out = append(out, Event{Kind: KLimit, Limit: &RateLimit{Limited: true, ResetsAt: time.Now().Add(15 * time.Minute), Kind: "error"}})
		}
		out = append(out, Event{Kind: KError, Text: msg, IsError: true})
	}
	if tok := findTokens(m); tok > 0 {
		if p.comp.check(tok) {
			out = append(out, Event{Kind: KCompacted, Text: "context dropped sharply: compaction detected"})
		}
		p.used = tok
		out = append(out, Event{Kind: KUsage, Usage: &Usage{Used: tok, Window: p.window}})
	}
	if typ == "step_finish" || typ == "finish" || typ == "done" {
		out = append(out, Event{Kind: KResult})
	}
	return out
}

// findTokens looks for a "tokens" object anywhere and sums input+output+cache.read.
func findTokens(v any) int {
	switch x := v.(type) {
	case map[string]any:
		if t, ok := x["tokens"].(map[string]any); ok {
			n := num(t["input"]) + num(t["output"])
			if c, ok := t["cache"].(map[string]any); ok {
				n += num(c["read"])
			}
			return n
		}
		for _, vv := range x {
			if n := findTokens(vv); n > 0 {
				return n
			}
		}
	case []any:
		for _, vv := range x {
			if n := findTokens(vv); n > 0 {
				return n
			}
		}
	}
	return 0
}

func num(v any) int { f, _ := v.(float64); return int(f) }

// ---------------------------------------------------------------- Fake

// Fake is a scripted harness for tests and demos: it runs `gbi dev fake-agent`, which
// emits Claude-style stream-json and edits files in the worktree. No subscription used.
type Fake struct{}

func (Fake) Name() string          { return "fake" }
func (Fake) DefaultBinary() string { exe, _ := os.Executable(); return exe }

func (f Fake) Command(s Spec) (string, []string) {
	bin := s.Binary
	if bin == "" {
		bin = f.DefaultBinary()
	}
	return bin, append([]string{"dev", "fake-agent", "--prompt", s.Prompt}, s.Flags...)
}
func (Fake) InputLine(string) ([]byte, bool) { return nil, false }
func (Fake) Detection() Detection {
	return Detection{Name: "fake", Headless: "stream-json (scripted)", UsageReported: "measured", Verified: true}
}
func (Fake) NewParser(s Spec) Parser { return &claudeParser{window: 200_000, measured: true} }

func firstNonEmpty(a ...string) string {
	for _, s := range a {
		if s != "" {
			return s
		}
	}
	return ""
}

// ---------------------------------------------------------------- Pi

// Pi drives the pi coding agent: `pi -p --mode json --no-session <prompt>` (P9-04).
// The event format was recorded from a real run (testdata/pi-*.jsonl); a successful
// turn has not been observed on the build machine, so it is marked partly verified.
type Pi struct{}

func init() { Register(Pi{}) }

func (Pi) Name() string          { return "pi" }
func (Pi) DefaultBinary() string { return "pi" }

func (Pi) Command(s Spec) (string, []string) {
	bin := s.Binary
	if bin == "" {
		bin = "pi"
	}
	args := []string{"-p", "--mode", "json", "--no-session"}
	if s.Model != "" {
		args = append(args, "--model", s.Model)
	}
	if s.SystemPrompt != "" {
		args = append(args, "--append-system-prompt", s.SystemPrompt)
	}
	return bin, append(append(args, s.Flags...), s.Prompt)
}

func (Pi) InputLine(string) ([]byte, bool) { return nil, false }

func (Pi) Detection() Detection {
	return Detection{Name: "pi", Headless: "-p --mode json", UsageReported: "estimated", Verified: false}
}

func (Pi) NewParser(s Spec) Parser { return &piParser{window: EstimateWindow(s.Model, s.Window)} }

type piParser struct {
	window  int
	comp    compactionDetector
	lastErr string
}

func (p *piParser) Parse(line []byte) []Event {
	var l struct {
		Type         string `json:"type"`
		ID           string `json:"id"`
		ErrorMessage string `json:"errorMessage"`
		Message      struct {
			Role    string `json:"role"`
			Model   string `json:"model"`
			Content []struct {
				Type string `json:"type"`
				Text string `json:"text"`
				Name string `json:"name"`
			} `json:"content"`
			Usage *struct {
				Input       int `json:"input"`
				Output      int `json:"output"`
				CacheRead   int `json:"cacheRead"`
				CacheWrite  int `json:"cacheWrite"`
				TotalTokens int `json:"totalTokens"`
			} `json:"usage"`
			StopReason   string `json:"stopReason"`
			ErrorMessage string `json:"errorMessage"`
		} `json:"message"`
	}
	if json.Unmarshal(line, &l) != nil {
		return nil
	}
	var out []Event
	switch l.Type {
	case "session":
		out = append(out, Event{Kind: KSystem, Text: "pi session started", SessionID: l.ID})
	case "message_end":
		if l.Message.Role != "assistant" {
			return nil
		}
		if l.Message.Model != "" {
			p.window = EstimateWindow(l.Message.Model, p.window)
		}
		for _, c := range l.Message.Content {
			switch c.Type {
			case "text":
				if strings.TrimSpace(c.Text) != "" {
					out = append(out, Event{Kind: KText, Text: c.Text})
				}
			case "toolCall", "tool_use":
				out = append(out, Event{Kind: KTool, Text: firstNonEmpty(c.Name, "tool")})
			}
		}
		if l.Message.ErrorMessage != "" {
			p.lastErr = l.Message.ErrorMessage
		}
		if u := l.Message.Usage; u != nil {
			used := u.Input + u.Output + u.CacheRead + u.CacheWrite
			if used > 0 {
				if p.comp.check(used) {
					out = append(out, Event{Kind: KCompacted, Text: "context dropped sharply: compaction detected"})
				}
				out = append(out, Event{Kind: KUsage, Usage: &Usage{Used: used, Window: p.window}})
			}
		}
	case "auto_retry_start":
		p.lastErr = l.ErrorMessage
		out = append(out, Event{Kind: KSystem, Text: "pi retrying: " + l.ErrorMessage})
		if LooksRateLimited(l.ErrorMessage) {
			out = append(out, Event{Kind: KLimit, Limit: &RateLimit{Limited: true, ResetsAt: time.Now().Add(15 * time.Minute), Kind: "error"}})
		}
	case "agent_end":
		// pi emits agent_end before each retry; the final one is the last line of output.
		out = append(out, Event{Kind: KSystem, Text: "pi turn ended"})
	case "auto_retry_end", "error":
		if l.ErrorMessage != "" {
			out = append(out, Event{Kind: KError, Text: l.ErrorMessage, IsError: true})
		}
	}
	return out
}
