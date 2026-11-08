package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/techmuch/go-build-it/internal/api"
)

// fakeAgent is a scripted harness (`gbi dev fake-agent`) that speaks Claude-style
// stream-json. It edits a file in the working directory so the full workflow
// (session → commit → push → review) can be exercised without an LLM.
//
// Prompt directives: "FAKE:ask" asks a blocking question, "FAKE:limit" reports a rate
// limit, "FAKE:fail" fails, "FAKE:compact" simulates a compaction.
func fakeAgent(args []string) error {
	prompt, chat := "", false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--prompt":
			if i+1 < len(args) {
				prompt = args[i+1]
				i++
			}
		case "--chat":
			chat = true
		}
	}
	if prompt == "" {
		b, _ := io.ReadAll(os.Stdin)
		prompt = string(b)
	}
	out := json.NewEncoder(os.Stdout)
	emit := func(v any) { _ = out.Encode(v); time.Sleep(30 * time.Millisecond) }
	usage := func(n int) map[string]any { return map[string]any{"input_tokens": n, "output_tokens": 40} }
	text := func(t string, n int) {
		emit(map[string]any{"type": "assistant", "message": map[string]any{"content": []map[string]any{{"type": "text", "text": t}}, "usage": usage(n)}})
	}
	result := func(t string, isErr bool) {
		emit(map[string]any{"type": "result", "subtype": "success", "is_error": isErr, "result": t, "session_id": "fake-session",
			"modelUsage": map[string]any{"fake": map[string]any{"contextWindow": 200000}}})
	}
	emit(map[string]any{"type": "system", "subtype": "init", "session_id": "fake-session", "model": "fake-model"})

	if chat {
		// Echo a small, visible revision for the editor/coach flows.
		if m := regexp.MustCompile(`(?s)<<<DOC\n(.*?)\nDOC>>>`).FindStringSubmatch(prompt); m != nil {
			req := ""
			if i := strings.Index(prompt, "Request: "); i >= 0 {
				req = strings.SplitN(prompt[i+9:], "\n", 2)[0]
			}
			doc := strings.TrimRight(m[1], "\n") + "\n- " + strings.TrimSpace(req)
			text("I added your request as a new line at the end.\n<<<DOC\n"+doc+"\nDOC>>>", 3000)
			result("I added your request as a new line at the end.\n<<<DOC\n"+doc+"\nDOC>>>", false)
			return nil
		}
		if strings.Contains(prompt, `"checks"`) {
			r := `{"summary":"Fake Lens: the change is small and tested.","recommendation":"approve","checks":[]}`
			text(r, 2000)
			result(r, false)
			return nil
		}
		text("OK", 1000)
		result("OK", false)
		return nil
	}

	text("Reading the task and the instruction files.", 12000)
	emit(map[string]any{"type": "assistant", "message": map[string]any{"content": []map[string]any{{"type": "tool_use", "name": "Edit", "input": map[string]string{"file": "FAKE_WORK.md"}}}, "usage": usage(30000)}})
	switch {
	case strings.Contains(prompt, "FAKE:limit"):
		emit(map[string]any{"type": "rate_limit_event", "rate_limit_info": map[string]any{"status": "rejected", "resetsAt": time.Now().Add(2 * time.Second).Unix(), "rateLimitType": "five_hour"}})
		result("rate limit reached", true)
		return nil
	case strings.Contains(prompt, "FAKE:fail"):
		result("could not complete the task", true)
		return nil
	case strings.Contains(prompt, "FAKE:ask"):
		task := os.Getenv("GBI_TASK")
		body, _ := json.Marshal(map[string]any{"task": task, "title": "Fake agent needs a decision", "why": "testing blocking questions", "blocking": true,
			"options": []map[string]any{{"key": "A", "title": "Option A", "recommended": true}, {"key": "B", "title": "Option B"}}})
		req, _ := http.NewRequest(http.MethodPost, os.Getenv("GBI_URL")+"/api/questions", bytes.NewReader(body))
		req.Header.Set("X-GBI-Persona", os.Getenv("GBI_PERSONA"))
		req.Header.Set("X-GBI-Session", os.Getenv("GBI_SESSION"))
		if resp, err := http.DefaultClient.Do(req); err == nil {
			resp.Body.Close()
		}
		text("I asked a blocking question and will stop here.", 32000)
		time.Sleep(2 * time.Second)
		result("waiting on a question", false)
		return nil
	case strings.Contains(prompt, "FAKE:handoff") && !strings.Contains(prompt, "Handoff briefing from"):
		text("Working through a long change.", 170000)
		time.Sleep(300 * time.Millisecond)
		brief := "<<<HANDOFF\nObjective and definition of done: finish the long change.\nCurrent state: half done.\nDecisions made and why: kept it simple.\nOpen problems, risks and dead ends to avoid: none.\nSpecific next steps: finish the second half.\nPointers: FAKE_WORK.md\nHANDOFF>>>"
		text(brief, 172000)
		result("handed off", false)
		return nil
	case strings.Contains(prompt, "FAKE:compact"):
		text("Working through a long change.", 160000)
		emit(map[string]any{"type": "system", "subtype": "compact_boundary"})
		text("Continuing after compaction.", 40000)
	}
	title := "task"
	if m := regexp.MustCompile(`# Task: (.*)`).FindStringSubmatch(prompt); m != nil {
		title = m[1]
	}
	f, err := os.OpenFile("FAKE_WORK.md", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	fmt.Fprintf(f, "- %s: done by fake agent at %s\n", title, time.Now().UTC().Format(time.RFC3339))
	f.Close()
	text("Updated FAKE_WORK.md and checked it.", 42000)
	result("Implemented "+title+". Tested by reading the file back.", false)
	return nil
}

// fakeTUI is a scripted interactive agent for attachable-session tests: it reads typed
// lines, does the work when told to read .gbi-task.md, and exits.
func fakeTUI() error {
	fmt.Println("fake agent ready. Type a request.")
	sc := bufio.NewScanner(os.Stdin)
	for sc.Scan() {
		line := sc.Text()
		fmt.Println("> " + line)
		if strings.Contains(line, ".gbi-task.md") {
			b, _ := os.ReadFile(".gbi-task.md")
			title := "task"
			if m := regexp.MustCompile(`# Task: (.*)`).FindStringSubmatch(string(b)); m != nil {
				title = m[1]
			}
			f, err := os.OpenFile("FAKE_WORK.md", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
			if err != nil {
				return err
			}
			fmt.Fprintf(f, "- %s: done in an attachable session\n", title)
			f.Close()
			fmt.Println("Did the work. DONE")
			time.Sleep(1500 * time.Millisecond)
			return nil
		}
	}
	return sc.Err()
}

// mcpServe runs a Model Context Protocol server over stdio (FR-33). Each API route is a
// tool; calls are forwarded to the running go-build-it server.
func mcpServe() error {
	rs := routes()
	tools := map[string]api.Route{}
	var list []map[string]any
	for _, r := range rs {
		if r.Name == "events.watch" {
			continue
		}
		name := strings.NewReplacer(".", "_", "-", "_").Replace(r.Name)
		tools[name] = r
		props := map[string]any{}
		var req []string
		for _, p := range api.PathParams(r.Path) {
			props[p] = map[string]any{"type": "string"}
			req = append(req, p)
		}
		for _, f := range r.Flags {
			t := map[string]string{"int": "integer", "bool": "boolean", "json": "object"}[f.Type]
			if t == "" {
				t = "string"
			}
			if f.Type == "json" {
				props[f.Name] = map[string]any{"description": f.Help}
			} else {
				props[f.Name] = map[string]any{"type": t, "description": f.Help}
			}
		}
		schema := map[string]any{"type": "object", "properties": props}
		if len(req) > 0 {
			schema["required"] = req
		}
		list = append(list, map[string]any{"name": name, "description": r.Summary + " (gbi " + r.CLI + ")", "inputSchema": schema})
	}
	in := bufio.NewScanner(os.Stdin)
	in.Buffer(make([]byte, 0, 1<<20), 16<<20)
	out := json.NewEncoder(os.Stdout)
	for in.Scan() {
		var msg struct {
			ID     json.RawMessage `json:"id"`
			Method string          `json:"method"`
			Params json.RawMessage `json:"params"`
		}
		if json.Unmarshal(in.Bytes(), &msg) != nil {
			continue
		}
		reply := func(result any) {
			_ = out.Encode(map[string]any{"jsonrpc": "2.0", "id": msg.ID, "result": result})
		}
		fail := func(code int, m string) {
			_ = out.Encode(map[string]any{"jsonrpc": "2.0", "id": msg.ID, "error": map[string]any{"code": code, "message": m}})
		}
		switch msg.Method {
		case "initialize":
			reply(map[string]any{"protocolVersion": "2025-06-18", "capabilities": map[string]any{"tools": map[string]any{}},
				"serverInfo": map[string]any{"name": "go-build-it", "version": version}})
		case "notifications/initialized", "notifications/cancelled":
			// no response to notifications
		case "ping":
			reply(map[string]any{})
		case "tools/list":
			reply(map[string]any{"tools": list})
		case "tools/call":
			var p struct {
				Name      string         `json:"name"`
				Arguments map[string]any `json:"arguments"`
			}
			_ = json.Unmarshal(msg.Params, &p)
			r, ok := tools[p.Name]
			if !ok {
				fail(-32602, "unknown tool "+p.Name)
				continue
			}
			text, isErr := mcpCall(r, p.Arguments)
			reply(map[string]any{"content": []map[string]any{{"type": "text", "text": text}}, "isError": isErr})
		default:
			if len(msg.ID) > 0 {
				fail(-32601, "method not found: "+msg.Method)
			}
		}
	}
	return in.Err()
}

func mcpCall(r api.Route, args map[string]any) (string, bool) {
	p := r.Path
	for _, name := range api.PathParams(r.Path) {
		v := fmt.Sprint(args[name])
		p = strings.Replace(strings.Replace(p, "{"+name+"...}", v, 1), "{"+name+"}", v, 1)
		delete(args, name)
	}
	var body io.Reader
	if r.Method == http.MethodGet || r.Method == http.MethodDelete {
		var q []string
		for k, v := range args {
			q = append(q, k+"="+fmt.Sprint(v))
		}
		if len(q) > 0 {
			p += "?" + strings.Join(q, "&")
		}
	} else {
		b, _ := json.Marshal(args)
		body = bytes.NewReader(b)
	}
	req, _ := http.NewRequest(r.Method, baseURL()+p, body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-GBI-Persona", "mcp-client")
	resp, err := (&http.Client{Timeout: 5 * time.Minute}).Do(req)
	if err != nil {
		return "go-build-it is not reachable: " + err.Error(), true
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return string(b), resp.StatusCode >= 400
}
