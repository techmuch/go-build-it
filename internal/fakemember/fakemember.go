// Package fakemember is a scripted interactive agent for tests and demos (`gbi dev
// fake-member`). It runs in a herdr pane like a real harness: it reports its own
// state to herdr (idle, working, blocked), takes prompts typed into its terminal,
// writes a Claude Code-format transcript, and edits files in its working directory.
// No AI subscription is used (P11-06).
//
// Prompt directives (in the prompt or in .gbi-task.md):
//
//	FAKE:ask      ask a blocking question through the go-build-it API, then go idle
//	FAKE:block    report "blocked" (like a permission dialog) until a line "y" arrives
//	FAKE:limit    write a rate-limit record and go idle
//	FAKE:fail     finish with an error result
//	FAKE:compact  write a compaction boundary
//	FAKE:big      use ~150k tokens of context (for rotation tests)
package fakemember

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/techmuch/go-build-it/internal/herdr"
)

// Source is the herdr report source for the fake agent.
const Source = "gbi-fake"

// TranscriptDir is where fake transcripts go: $GBI_FAKE_TRANSCRIPTS, else <home>/.gbi-fake.
func TranscriptDir() string {
	if d := os.Getenv("GBI_FAKE_TRANSCRIPTS"); d != "" {
		return d
	}
	h, _ := os.UserHomeDir()
	return filepath.Join(h, ".gbi-fake")
}

type agent struct {
	h       *herdr.Client
	pane    string
	session string
	tx      *os.File
	out     io.Writer
	seq     int64
	used    int
}

// Run starts the agent. --session-id <id> starts a conversation with that ID (go-build-it
// chooses it, as it does for Claude Code); --resume <id> continues one.
func Run(args []string, in io.Reader, out io.Writer) error {
	a := &agent{h: herdr.FromEnv(), pane: os.Getenv("HERDR_PANE_ID"), out: out}
	for i := 0; i < len(args); i++ {
		if (args[i] == "--resume" || args[i] == "--session-id") && i+1 < len(args) {
			a.session = args[i+1]
			i++
		}
	}
	if a.session == "" {
		b := make([]byte, 8)
		_, _ = rand.Read(b)
		a.session = "fake-" + hex.EncodeToString(b)
	}
	if err := os.MkdirAll(TranscriptDir(), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(filepath.Join(TranscriptDir(), a.session+".jsonl"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	a.tx = f
	defer f.Close()
	fmt.Fprintf(out, "fake agent · session %s · ready\n", a.session)
	a.report("idle", "")
	sc := bufio.NewScanner(in)
	sc.Buffer(make([]byte, 0, 64<<10), 4<<20)
	for sc.Scan() {
		line := strings.TrimSpace(strings.Trim(sc.Text(), "\x1b[200~\x1b[201~"))
		if line == "" {
			continue
		}
		if line == "/exit" {
			break
		}
		a.turn(line, sc)
	}
	if a.h != nil {
		a.seq++
		_ = a.h.ReleaseAgent(context.Background(), a.pane, Source, "fake", a.seqNext())
	}
	return sc.Err()
}

func (a *agent) seqNext() int64 { a.seq = time.Now().UnixNano(); return a.seq }

func (a *agent) report(state, msg string) {
	if a.h == nil || a.pane == "" {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	exe := "gbi"
	_ = a.h.ReportAgent(ctx, a.pane, Source, "fake", state, msg, a.session, []string{exe, "dev", "fake-member", "--resume", a.session}, a.seqNext())
}

func (a *agent) write(v map[string]any) {
	v["sessionId"] = a.session
	v["timestamp"] = time.Now().UTC().Format(time.RFC3339Nano)
	b, _ := json.Marshal(v)
	_, _ = a.tx.Write(append(b, '\n'))
}

func (a *agent) assistant(content []map[string]any, add int) {
	a.used += add
	a.write(map[string]any{"type": "assistant", "message": map[string]any{"role": "assistant", "model": "fake-model", "content": content,
		"usage": map[string]any{"input_tokens": 2, "cache_creation_input_tokens": add, "cache_read_input_tokens": a.used - add, "output_tokens": 40}}})
	time.Sleep(40 * time.Millisecond)
}

func (a *agent) say(t string, add int) {
	fmt.Fprintln(a.out, "● "+t)
	a.assistant([]map[string]any{{"type": "text", "text": t}}, add)
}

func (a *agent) turn(prompt string, sc *bufio.Scanner) {
	a.report("working", "")
	fmt.Fprintln(a.out, "❯ "+prompt)
	a.write(map[string]any{"type": "user", "message": map[string]any{"role": "user", "content": prompt}})
	task := prompt
	if strings.Contains(prompt, ".gbi-task.md") {
		if b, err := os.ReadFile(".gbi-task.md"); err == nil {
			task = string(b)
			a.assistant([]map[string]any{{"type": "tool_use", "name": "Read", "input": map[string]string{"file_path": ".gbi-task.md"}}}, 4000)
		}
	}
	if a.used == 0 {
		a.used = 8000
	}
	switch {
	case strings.Contains(task, "FAKE:block"):
		fmt.Fprintln(a.out, "Allow edit to FAKE_WORK.md? (y/n)")
		a.report("blocked", "permission: edit FAKE_WORK.md")
		for sc.Scan() {
			if strings.HasPrefix(strings.TrimSpace(sc.Text()), "y") {
				break
			}
		}
		a.report("working", "")
	case strings.Contains(task, "FAKE:limit"):
		reset := time.Now().Add(2 * time.Second).Unix()
		a.write(map[string]any{"type": "system", "subtype": "rate_limit", "rate_limit": map[string]any{"status": "rejected", "resetsAt": reset, "rateLimitType": "five_hour"}})
		a.say("I hit a usage limit.", 1000)
		a.report("idle", "")
		return
	case strings.Contains(task, "FAKE:fail"):
		a.say("I could not complete the task.", 2000)
		a.write(map[string]any{"type": "system", "subtype": "result", "is_error": true, "result": "could not complete the task"})
		a.report("idle", "")
		return
	case strings.Contains(task, "FAKE:ask"):
		body, _ := json.Marshal(map[string]any{"task": os.Getenv("GBI_TASK"), "title": "Fake agent needs a decision", "why": "testing blocking questions", "blocking": true,
			"options": []map[string]any{{"key": "A", "title": "Option A", "recommended": true}, {"key": "B", "title": "Option B"}}})
		if m := regexp.MustCompile(`GBI_TASK=(\S+)`).FindStringSubmatch(task); m != nil {
			body, _ = json.Marshal(map[string]any{"task": m[1], "title": "Fake agent needs a decision", "why": "testing blocking questions", "blocking": true,
				"options": []map[string]any{{"key": "A", "title": "Option A", "recommended": true}, {"key": "B", "title": "Option B"}}})
		}
		req, _ := http.NewRequest(http.MethodPost, os.Getenv("GBI_URL")+"/api/questions", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-GBI-Persona", os.Getenv("GBI_PERSONA"))
		if resp, err := http.DefaultClient.Do(req); err == nil {
			resp.Body.Close()
		}
		a.say("I asked a blocking question and will stop here.", 3000)
		a.report("idle", "")
		return
	case strings.Contains(task, "FAKE:compact"):
		a.say("Working through a long change.", 150000)
		a.write(map[string]any{"type": "system", "subtype": "compact_boundary"})
		a.used = 30000
	case strings.Contains(task, "FAKE:big"):
		a.say("Working through a long change.", 140000)
	}
	title := "task"
	if m := regexp.MustCompile(`# Task: (.*)`).FindStringSubmatch(task); m != nil {
		title = strings.TrimSpace(m[1])
	}
	a.assistant([]map[string]any{{"type": "tool_use", "name": "Edit", "input": map[string]string{"file_path": "FAKE_WORK.md"}}}, 2000)
	f, err := os.OpenFile("FAKE_WORK.md", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err == nil {
		fmt.Fprintf(f, "- %s: done by fake member at %s\n", title, time.Now().UTC().Format(time.RFC3339))
		f.Close()
	}
	a.say("Implemented "+title+". Tested by reading the file back.", 1500)
	a.write(map[string]any{"type": "system", "subtype": "result", "is_error": false, "result": "Implemented " + title})
	a.report("idle", "")
}
