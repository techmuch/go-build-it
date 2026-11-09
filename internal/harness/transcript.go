package harness

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"time"
)

// Event kinds read from a transcript (FR-66 structured view).
const (
	KUser      = "user"
	KText      = "text"
	KTool      = "tool"
	KSystem    = "system"
	KCompacted = "compacted"
	KLimit     = "limit"
	KResult    = "result"
	KError     = "error"
)

// Usage is context-window usage (FR-71).
type Usage struct {
	Used     int  `json:"used"`
	Window   int  `json:"window"`
	Measured bool `json:"measured"`
}

// Fraction is Used/Window.
func (u Usage) Fraction() float64 {
	if u.Window <= 0 {
		return 0
	}
	return float64(u.Used) / float64(u.Window)
}

// RateLimit is a subscription limit with its reset time (FR-65).
type RateLimit struct {
	ResetsAt time.Time `json:"resetsAt"`
	Kind     string    `json:"kind"`
}

// Event is one structured transcript event.
type Event struct {
	Kind    string     `json:"kind"`
	Text    string     `json:"text,omitempty"`
	Usage   *Usage     `json:"usage,omitempty"`
	Limit   *RateLimit `json:"limit,omitempty"`
	IsError bool       `json:"isError,omitempty"`
	Model   string     `json:"model,omitempty"`
}

// Reader reads a transcript incrementally: each call returns the events appended since the last.
type Reader interface {
	Read(path string) ([]Event, error)
}

type tail struct{ off int64 }

// lines returns the complete lines appended to path since the last call.
func (t *tail) lines(path string) ([][]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	if fi, err := f.Stat(); err == nil && fi.Size() < t.off {
		t.off = 0 // rewritten
	}
	if _, err := f.Seek(t.off, io.SeekStart); err != nil {
		return nil, err
	}
	r := bufio.NewReaderSize(f, 1<<20)
	var out [][]byte
	for {
		line, err := r.ReadBytes('\n')
		if err != nil {
			break // an incomplete last line is read next time
		}
		t.off += int64(len(line))
		if l := strings.TrimSpace(string(line)); l != "" {
			out = append(out, []byte(l))
		}
	}
	return out, nil
}

var limitRe = regexp.MustCompile(`(?i)(usage limit|rate limit|limit reached|quota exceeded|too many requests|resets? (at )?\d)`)
var resetRe = regexp.MustCompile(`(?i)resets?\s+(?:at\s+)?(\d{1,2})(?::(\d{2}))?\s*(am|pm)?`)

// LooksRateLimited reports whether text reads like a subscription limit message.
func LooksRateLimited(s string) bool { return limitRe.MatchString(s) }

// guessReset reads "resets 3pm" style text; it falls back to one hour from now.
func guessReset(s string, now time.Time) time.Time {
	if m := resetRe.FindStringSubmatch(s); m != nil {
		var h, min int
		fmt.Sscanf(m[1], "%d", &h)
		if m[2] != "" {
			fmt.Sscanf(m[2], "%d", &min)
		}
		if strings.EqualFold(m[3], "pm") && h < 12 {
			h += 12
		}
		t := time.Date(now.Year(), now.Month(), now.Day(), h, min, 0, 0, now.Location())
		if !t.After(now) {
			t = t.Add(24 * time.Hour)
		}
		return t
	}
	return now.Add(time.Hour)
}

func summarizeInput(in json.RawMessage) string {
	var m map[string]any
	if json.Unmarshal(in, &m) != nil {
		return ""
	}
	for _, k := range []string{"file_path", "path", "command", "pattern", "url", "description", "file"} {
		if v, ok := m[k].(string); ok && v != "" {
			if len(v) > 120 {
				v = v[:120] + "…"
			}
			return v
		}
	}
	return ""
}

// ---- Claude Code (and the fake member, which writes the same format) ----

type claudeReader struct{ tail }

func (r *claudeReader) Read(path string) ([]Event, error) {
	ls, err := r.lines(path)
	if err != nil {
		return nil, err
	}
	var out []Event
	for _, l := range ls {
		var row struct {
			Type      string `json:"type"`
			Subtype   string `json:"subtype"`
			IsError   bool   `json:"is_error"`
			Result    string `json:"result"`
			APIError  bool   `json:"isApiErrorMessage"`
			RateLimit *struct {
				ResetsAt int64  `json:"resetsAt"`
				Type     string `json:"rateLimitType"`
			} `json:"rate_limit"`
			Message struct {
				Role    string          `json:"role"`
				Model   string          `json:"model"`
				Content json.RawMessage `json:"content"`
				Usage   *struct {
					In    int `json:"input_tokens"`
					Cache int `json:"cache_creation_input_tokens"`
					Read  int `json:"cache_read_input_tokens"`
				} `json:"usage"`
			} `json:"message"`
		}
		if json.Unmarshal(l, &row) != nil {
			continue
		}
		switch row.Type {
		case "user":
			var s string
			if json.Unmarshal(row.Message.Content, &s) == nil && s != "" {
				out = append(out, Event{Kind: KUser, Text: s})
			}
		case "assistant":
			var items []struct {
				Type  string          `json:"type"`
				Text  string          `json:"text"`
				Name  string          `json:"name"`
				Input json.RawMessage `json:"input"`
			}
			_ = json.Unmarshal(row.Message.Content, &items)
			for _, it := range items {
				switch it.Type {
				case "text":
					if row.APIError && LooksRateLimited(it.Text) {
						out = append(out, Event{Kind: KLimit, Text: it.Text, Limit: &RateLimit{ResetsAt: guessReset(it.Text, time.Now()), Kind: "message"}})
					} else {
						out = append(out, Event{Kind: KText, Text: it.Text})
					}
				case "tool_use":
					out = append(out, Event{Kind: KTool, Text: strings.TrimSpace(it.Name + " " + summarizeInput(it.Input))})
				}
			}
			if u := row.Message.Usage; u != nil {
				out = append(out, Event{Kind: KSystem, Model: row.Message.Model, Usage: &Usage{Used: u.In + u.Cache + u.Read, Measured: true}})
			}
		case "system":
			switch row.Subtype {
			case "compact_boundary":
				out = append(out, Event{Kind: KCompacted, Text: "context compacted"})
			case "rate_limit":
				if row.RateLimit != nil {
					out = append(out, Event{Kind: KLimit, Text: "subscription limit", Limit: &RateLimit{ResetsAt: time.Unix(row.RateLimit.ResetsAt, 0), Kind: row.RateLimit.Type}})
				}
			case "result":
				out = append(out, Event{Kind: KResult, Text: row.Result, IsError: row.IsError})
			}
		}
	}
	return out, nil
}

// ---- Antigravity: transcript_full.jsonl steps ----

type agyReader struct{ tail }

func (r *agyReader) Read(path string) ([]Event, error) {
	ls, err := r.lines(path)
	if err != nil {
		return nil, err
	}
	var out []Event
	for _, l := range ls {
		var row struct {
			Type      string `json:"type"`
			Source    string `json:"source"`
			Content   string `json:"content"`
			In        int    `json:"input_tokens"`
			Read      int    `json:"cache_read_tokens"`
			ToolCalls []struct {
				Name string `json:"name"`
			} `json:"tool_calls"`
		}
		if json.Unmarshal(l, &row) != nil {
			continue
		}
		switch {
		case row.Type == "USER_INPUT":
			out = append(out, Event{Kind: KUser, Text: strings.TrimSpace(strings.NewReplacer("<USER_REQUEST>", "", "</USER_REQUEST>", "").Replace(row.Content))})
		case row.Type == "PLANNER_RESPONSE":
			if row.Content != "" {
				out = append(out, Event{Kind: KText, Text: row.Content})
			}
			for _, tc := range row.ToolCalls {
				out = append(out, Event{Kind: KTool, Text: tc.Name})
			}
			if row.In+row.Read > 0 {
				out = append(out, Event{Kind: KSystem, Usage: &Usage{Used: row.In + row.Read, Measured: true}})
			}
		case row.Source == "MODEL" && row.Content != "" && LooksRateLimited(row.Content) && strings.Contains(strings.ToLower(row.Content), "quota"):
			out = append(out, Event{Kind: KLimit, Text: row.Content, Limit: &RateLimit{ResetsAt: guessReset(row.Content, time.Now()), Kind: "quota"}})
		}
	}
	return out, nil
}

// ---- Pi: session .jsonl with message entries ----

type piReader struct{ tail }

func (r *piReader) Read(path string) ([]Event, error) {
	ls, err := r.lines(path)
	if err != nil {
		return nil, err
	}
	var out []Event
	for _, l := range ls {
		var row struct {
			Type    string `json:"type"`
			Message struct {
				Role         string          `json:"role"`
				Model        string          `json:"model"`
				Content      json.RawMessage `json:"content"`
				ErrorMessage string          `json:"errorMessage"`
				Usage        *struct {
					Input      int `json:"input"`
					CacheRead  int `json:"cacheRead"`
					CacheWrite int `json:"cacheWrite"`
				} `json:"usage"`
			} `json:"message"`
		}
		if json.Unmarshal(l, &row) != nil || row.Type != "message" {
			continue
		}
		var items []struct {
			Type string `json:"type"`
			Text string `json:"text"`
			Name string `json:"name"`
		}
		var s string
		if json.Unmarshal(row.Message.Content, &s) == nil && s != "" {
			items = append(items, struct {
				Type string `json:"type"`
				Text string `json:"text"`
				Name string `json:"name"`
			}{Type: "text", Text: s})
		} else {
			_ = json.Unmarshal(row.Message.Content, &items)
		}
		for _, it := range items {
			switch {
			case it.Type == "text" && row.Message.Role == "user":
				out = append(out, Event{Kind: KUser, Text: it.Text})
			case it.Type == "text" && row.Message.Role == "assistant":
				out = append(out, Event{Kind: KText, Text: it.Text})
			case it.Type == "toolCall" || it.Type == "tool_use":
				out = append(out, Event{Kind: KTool, Text: it.Name})
			}
		}
		if row.Message.ErrorMessage != "" {
			if LooksRateLimited(row.Message.ErrorMessage) {
				out = append(out, Event{Kind: KLimit, Text: row.Message.ErrorMessage, Limit: &RateLimit{ResetsAt: guessReset(row.Message.ErrorMessage, time.Now()), Kind: "error"}})
			} else {
				out = append(out, Event{Kind: KError, Text: row.Message.ErrorMessage})
			}
		}
		if u := row.Message.Usage; u != nil && row.Message.Role == "assistant" {
			out = append(out, Event{Kind: KSystem, Model: row.Message.Model, Usage: &Usage{Used: u.Input + u.CacheRead + u.CacheWrite, Measured: true}})
		}
	}
	return out, nil
}
