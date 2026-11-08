// Package harness drives external agent CLIs headlessly through one adapter
// interface (FR-60, FR-61). See ADR-002 for what each harness reports.
package harness

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"
)

// Usage is context-window usage (FR-71). Measured is false when Used or Window is estimated.
type Usage struct {
	Used     int  `json:"used"`
	Window   int  `json:"window"`
	Measured bool `json:"measured"`
}

// Fraction returns Used/Window (0 when unknown).
func (u Usage) Fraction() float64 {
	if u.Window <= 0 {
		return 0
	}
	return float64(u.Used) / float64(u.Window)
}

// RateLimit reports a subscription limit (FR-65).
type RateLimit struct {
	Limited  bool      `json:"limited"`
	ResetsAt time.Time `json:"resetsAt"`
	Kind     string    `json:"kind"`
}

// Event kinds.
const (
	KText      = "text"      // assistant text
	KTool      = "tool"      // tool call or result summary
	KSystem    = "system"    // harness system info
	KUsage     = "usage"     // usage update only
	KLimit     = "limit"     // rate limit info
	KCompacted = "compacted" // the harness compacted its context (FR-72)
	KResult    = "result"    // turn finished
	KError     = "error"
	KRaw       = "raw" // unparsed line (terminal view, FR-66)
)

// Event is one normalized harness event.
type Event struct {
	Kind      string     `json:"kind"`
	Text      string     `json:"text,omitempty"`
	Usage     *Usage     `json:"usage,omitempty"`
	Limit     *RateLimit `json:"limit,omitempty"`
	SessionID string     `json:"sessionId,omitempty"`
	IsError   bool       `json:"isError,omitempty"`
}

// Spec describes a session to start.
type Spec struct {
	Dir          string
	Prompt       string
	SystemPrompt string
	Model        string
	Binary       string
	Flags        []string
	Window       int // context window to assume when the harness doesn't report it
	Env          []string
}

// Parser turns output lines into events. Parsers are stateful (they track usage).
type Parser interface {
	Parse(line []byte) []Event
}

// Detection reports whether a harness is installed and what it supports (FR-63).
type Detection struct {
	Name          string `json:"name"`
	Binary        string `json:"binary"`
	Path          string `json:"path"`
	Version       string `json:"version"`
	Installed     bool   `json:"installed"`
	Headless      string `json:"headless"`      // e.g. "stream-json"
	UsageReported string `json:"usageReported"` // measured | estimated | unknown
	Verified      bool   `json:"verified"`      // checked against a real run (ADR-002)
	SupportsInput bool   `json:"supportsInput"`
}

// Adapter is implemented by every harness.
type Adapter interface {
	Name() string
	DefaultBinary() string
	Command(s Spec) (bin string, args []string)
	NewParser(s Spec) Parser
	// InputLine formats a mid-session message for stdin, or returns false if unsupported.
	InputLine(text string) ([]byte, bool)
	Detection() Detection
}

// Registry of adapters by name.
var adapters = map[string]Adapter{}

// Register adds an adapter.
func Register(a Adapter) { adapters[a.Name()] = a }

// Get returns an adapter.
func Get(name string) (Adapter, bool) { a, ok := adapters[name]; return a, ok }

// Names lists registered adapters.
func Names() []string {
	var out []string
	for _, n := range []string{"claude", "opencode", "antigravity", "pi", "fake"} {
		if _, ok := adapters[n]; ok {
			out = append(out, n)
		}
	}
	for n := range adapters {
		known := false
		for _, k := range out {
			known = known || k == n
		}
		if !known {
			out = append(out, n)
		}
	}
	return out
}

// Detect checks whether the adapter's binary is installed.
func Detect(ctx context.Context, a Adapter, binary string) Detection {
	d := a.Detection()
	if binary == "" {
		binary = a.DefaultBinary()
	}
	d.Binary = binary
	p, err := exec.LookPath(binary)
	if err != nil {
		return d
	}
	d.Path, d.Installed = p, true
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if out, err := exec.CommandContext(ctx, p, "--version").CombinedOutput(); err == nil {
		d.Version = strings.TrimSpace(strings.SplitN(string(out), "\n", 2)[0])
	}
	return d
}

// Window table for estimates when a harness doesn't report the window size.
var modelWindows = []struct {
	re     *regexp.Regexp
	window int
}{
	{regexp.MustCompile(`(?i)gemini`), 1_000_000},
	{regexp.MustCompile(`(?i)claude.*(opus|sonnet)-[45]`), 1_000_000},
	{regexp.MustCompile(`(?i)claude`), 200_000},
	{regexp.MustCompile(`(?i)gpt-5`), 400_000},
	{regexp.MustCompile(`(?i)gpt`), 128_000},
}

// EstimateWindow returns the assumed window for a model, falling back to def.
func EstimateWindow(model string, def int) int {
	for _, m := range modelWindows {
		if model != "" && m.re.MatchString(model) {
			return m.window
		}
	}
	if def > 0 {
		return def
	}
	return 200_000
}

var limitRe = regexp.MustCompile(`(?i)(rate.?limit|usage limit|quota|resource.?exhausted|\b429\b|too many requests)`)

// LooksRateLimited reports whether error text indicates a subscription limit.
func LooksRateLimited(s string) bool { return limitRe.MatchString(s) }

// compactionDetector flags a large drop in used tokens as a compaction (FR-72).
type compactionDetector struct{ last int }

func (c *compactionDetector) check(used int) bool {
	defer func() { c.last = used }()
	return c.last > 20_000 && used < c.last*6/10
}

// Process is a running harness session.
type Process struct {
	Events <-chan Event
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	mu     sync.Mutex
	done   chan struct{}
	err    error
	a      Adapter
	closed bool
}

// Start launches the harness and streams normalized events. Raw lines are also
// emitted as KRaw events so the terminal view can show them (FR-66).
func Start(ctx context.Context, a Adapter, s Spec) (*Process, error) {
	bin, args := a.Command(s)
	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Dir = s.Dir
	cmd.Env = append(os.Environ(), s.Env...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}
	var stdin io.WriteCloser
	first, inputOK := a.InputLine(s.Prompt)
	if inputOK {
		if stdin, err = cmd.StdinPipe(); err != nil {
			return nil, err
		}
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start %s: %w", bin, err)
	}
	ch := make(chan Event, 512)
	p := &Process{Events: ch, cmd: cmd, stdin: stdin, done: make(chan struct{}), a: a}
	if inputOK {
		if _, err := stdin.Write(first); err != nil {
			return nil, fmt.Errorf("write prompt: %w", err)
		}
	}
	parser := a.NewParser(s)
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		sc := bufio.NewScanner(stdout)
		sc.Buffer(make([]byte, 0, 1<<20), 16<<20)
		for sc.Scan() {
			line := append([]byte(nil), sc.Bytes()...)
			ch <- Event{Kind: KRaw, Text: string(line)}
			for _, e := range parser.Parse(line) {
				ch <- e
				if e.Kind == KResult && inputOK {
					p.CloseInput() // one turn per session unless more input is queued
				}
			}
		}
	}()
	go func() {
		defer wg.Done()
		sc := bufio.NewScanner(stderr)
		for sc.Scan() {
			t := sc.Text()
			ch <- Event{Kind: KRaw, Text: t}
			if LooksRateLimited(t) {
				ch <- Event{Kind: KLimit, Limit: &RateLimit{Limited: true, ResetsAt: time.Now().Add(15 * time.Minute), Kind: "stderr"}, Text: t}
			}
		}
	}()
	go func() {
		wg.Wait()
		p.err = cmd.Wait()
		close(ch)
		close(p.done)
	}()
	return p, nil
}

// Send writes a mid-session message (FR-61 "send input"). Returns false if the harness
// doesn't accept input or the turn already ended.
func (p *Process) Send(text string) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.stdin == nil || p.closed {
		return false
	}
	line, ok := p.a.InputLine(text)
	if !ok {
		return false
	}
	_, err := p.stdin.Write(line)
	return err == nil
}

// CloseInput ends the session's input stream so the harness exits after its turn.
func (p *Process) CloseInput() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.stdin != nil && !p.closed {
		p.closed = true
		_ = p.stdin.Close()
	}
}

// Cancel stops the harness.
func (p *Process) Cancel() {
	if p.cmd.Process != nil {
		_ = p.cmd.Process.Signal(os.Interrupt)
		go func() {
			select {
			case <-p.done:
			case <-time.After(5 * time.Second):
				_ = p.cmd.Process.Kill()
			}
		}()
	}
}

// Wait blocks until the process exits and returns its error.
func (p *Process) Wait() error { <-p.done; return p.err }

// helpers ----------------------------------------------------------------------

func jsonLine(v any) []byte { b, _ := json.Marshal(v); return append(b, '\n') }

func trunc(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) > n {
		return s[:n] + "…"
	}
	return s
}
