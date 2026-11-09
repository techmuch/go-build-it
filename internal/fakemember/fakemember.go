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
//
// The doctor's prompt ("Reply with the single word READY.") is answered with READY.
// --start-blocked shows a sign-in screen and reports "blocked" at start, like a harness
// that has never been signed in (doctor tests, FR-07b).
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

var (
	askRe    = regexp.MustCompile(`(\.gbi-ask-[A-Za-z0-9-]+\.md)`)
	answerRe = regexp.MustCompile(`(\.gbi-answer-[A-Za-z0-9-]+\.md)`)
	docRe    = regexp.MustCompile(`(?s)<<<DOC\n(.*?)\nDOC>>>`)
)

// ask answers a one-shot helper request (artifact or persona edits, Lens): it reads the
// brief and writes the answer file the brief names, like a real harness would.
func (a *agent) ask(file string) {
	b, err := os.ReadFile(file)
	brief := string(b)
	m := answerRe.FindStringSubmatch(brief)
	if err != nil || m == nil {
		a.say("I could not read "+file, 500)
		a.report("idle", "")
		return
	}
	answer := "OK"
	switch {
	case docRe.MatchString(brief):
		req := ""
		if i := strings.Index(brief, "Request: "); i >= 0 {
			req = strings.SplitN(brief[i+9:], "\n", 2)[0]
		}
		doc := strings.TrimRight(docRe.FindStringSubmatch(brief)[1], "\n") + "\n- " + strings.TrimSpace(req)
		answer = "I added your request as a new line at the end.\n<<<DOC\n" + doc + "\nDOC>>>"
	case strings.Contains(brief, `"checks"`):
		answer = `{"summary":"Fake Lens: the change is small and tested.","recommendation":"approve","checks":[]}`
	case strings.Contains(strings.ToLower(brief), "release notes"):
		answer = "## Highlights\n- Fake release notes."
	}
	a.assistant([]map[string]any{{"type": "tool_use", "name": "Write", "input": map[string]string{"file_path": m[1]}}}, 1500)
	_ = os.WriteFile(m[1], []byte(answer+"\n"), 0o644)
	a.say("DONE", 200)
	a.report("idle", "")
}

// Source is the herdr report source for the fake agent.
const Source = "gbi-fake"

// TranscriptDir is where fake transcripts go: $GBI_FAKE_TRANSCRIPTS, else <GBI_HOME>/fake-transcripts.
func TranscriptDir() string {
	if d := os.Getenv("GBI_FAKE_TRANSCRIPTS"); d != "" {
		return d
	}
	if d := os.Getenv("GBI_HOME"); d != "" {
		return filepath.Join(d, "fake-transcripts")
	}
	h, _ := os.UserHomeDir()
	return filepath.Join(h, ".go-build-it", "fake-transcripts")
}

const handoffBrief = "<<<HANDOFF\nObjective and definition of done: finish the long change.\nCurrent state: half done.\nDecisions made and why: kept it simple.\nOpen problems, risks and dead ends to avoid: none.\nSpecific next steps: finish the second half.\nPointers: FAKE_WORK.md\nHANDOFF>>>"

type agent struct {
	handedOff bool
	h         *herdr.Client
	pane      string
	session   string
	tx        *os.File
	out       io.Writer
	seq       int64
	used      int
}

// Run starts the agent. --session-id <id> starts a conversation with that ID (go-build-it
// chooses it, as it does for Claude Code); --resume <id> continues one.
func Run(args []string, in io.Reader, out io.Writer) error {
	a := &agent{h: herdr.FromEnv(), pane: os.Getenv("HERDR_PANE_ID"), out: out}
	startBlocked := false
	for i := 0; i < len(args); i++ {
		if args[i] == "--start-blocked" {
			startBlocked = true
			continue
		}
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
	if startBlocked {
		fmt.Fprintln(out, "fake agent · not signed in\nSelect login method:\n❯ 1. Fake account\n  2. Fake console")
		a.report("blocked", "sign in")
		_, _ = io.Copy(io.Discard, in)
		return nil
	}
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
		if line == "/cancel" {
			continue // turns are synchronous; nothing to interrupt
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
	if strings.Contains(prompt, "Reply with the single word READY") {
		a.say("READY", 100)
		a.report("idle", "")
		return
	}
	if m := askRe.FindStringSubmatch(prompt); m != nil {
		a.ask(m[1])
		return
	}
	if strings.Contains(prompt, "<<<HANDOFF") || strings.Contains(prompt, "handoff briefing") {
		if !a.handedOff {
			a.say(handoffBrief, 1000)
			a.handedOff = true
		}
		a.report("idle", "")
		return
	}
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
		taskID := os.Getenv("GBI_TASK")
		if m := regexp.MustCompile(`working on task (T-\d+)`).FindStringSubmatch(task); m != nil {
			taskID = m[1]
		}
		body, _ := json.Marshal(map[string]any{"task": taskID, "title": "Fake agent needs a decision", "why": "testing blocking questions", "blocking": true,
			"options": []map[string]any{{"key": "A", "title": "Option A", "recommended": true}, {"key": "B", "title": "Option B"}}})
		req, _ := http.NewRequest(http.MethodPost, os.Getenv("GBI_URL")+"/api/questions", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		persona := os.Getenv("GBI_PERSONA")
		if persona == "" {
			persona = memberFromCwd() // like `gbi` itself, identify by the member worktree
		}
		req.Header.Set("X-GBI-Persona", persona)
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
	case strings.Contains(task, "FAKE:handoff") && !strings.Contains(task, "Handoff briefing from"):
		// Fill the context, then hand off in the same turn (the request may also arrive as a prompt).
		a.say("Working through a long change.", 170000)
		time.Sleep(300 * time.Millisecond)
		a.say(handoffBrief, 2000)
		a.handedOff = true
		a.report("idle", "")
		return
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

// memberFromCwd recognises …/worktrees/<project>/members/<slug> and returns <slug>@<project>.
func memberFromCwd() string {
	wd, err := os.Getwd()
	if err != nil {
		return ""
	}
	parts := strings.Split(filepath.ToSlash(filepath.Clean(wd)), "/")
	for i := len(parts) - 1; i >= 3; i-- {
		if parts[i-1] == "members" && parts[i-3] == "worktrees" {
			return parts[i] + "@" + parts[i-2]
		}
	}
	return ""
}
