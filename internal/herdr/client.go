// Package herdr is go-build-it's client for herdr, the only agent backend (ADR-009).
//
// go-build-it talks to one named herdr session (default "gbi") over herdr's local
// socket API: newline-delimited JSON, one request per connection (a Unix socket, or
// a named pipe on Windows). It never uses the user's default herdr session.
package herdr

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"
)

// DefaultSession is the herdr session go-build-it owns.
const DefaultSession = "gbi"

// APIError is an error response from herdr.
type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *APIError) Error() string { return "herdr " + e.Code + ": " + e.Message }

// IsCode reports whether err is a herdr API error with the given code.
func IsCode(err error, code string) bool {
	var a *APIError
	return errors.As(err, &a) && a.Code == code
}

// ErrNotRunning means the gbi session's socket does not answer.
var ErrNotRunning = errors.New("the gbi herdr session is not running")

// Client talks to one herdr session.
type Client struct {
	Bin     string   // herdr binary
	Session string   // session name
	Socket  string   // socket path (Windows: the name under \\.\pipe\)
	Env     []string // environment for herdr CLI calls (nil: inherit)
	// Dial opens a connection; tests replace it.
	Dial func(ctx context.Context, socket string) (io.ReadWriteCloser, error)
	seq  atomic.Int64
}

// New returns a client for session (DefaultSession when empty), resolving the socket
// path from herdr itself so it is right on every platform.
func New(bin, session string) *Client { return NewWithEnv(bin, session, nil) }

// NewWithEnv is New with an explicit environment for herdr CLI calls (e.g. an isolated XDG_CONFIG_HOME).
func NewWithEnv(bin, session string, env []string) *Client {
	if session == "" {
		session = DefaultSession
	}
	if bin == "" {
		bin = "herdr"
	}
	c := &Client{Bin: bin, Session: session, Dial: dial, Env: env}
	c.Socket = c.resolveSocket()
	return c
}

// resolveSocket asks `herdr session list --json`; if the session is unknown yet, it
// derives the path from the default session's directory: <dir>/sessions/<name>/herdr.sock.
func (c *Client) resolveSocket() string {
	cmd := exec.Command(c.Bin, "session", "list", "--json")
	cmd.Env = c.Env
	if out, err := cmd.Output(); err == nil {
		var v any
		if json.Unmarshal(out, &v) == nil {
			var def string
			for _, s := range sessionsOf(v) {
				if s.Name == c.Session && s.Socket != "" {
					return s.Socket
				}
				if s.Name == "default" {
					def = s.Directory
				}
			}
			if def != "" {
				return filepath.Join(def, "sessions", c.Session, "herdr.sock")
			}
		}
	}
	home, _ := os.UserHomeDir()
	base := filepath.Join(home, ".config", "herdr")
	if x := envValue(c.Env, "XDG_CONFIG_HOME"); x != "" {
		base = filepath.Join(x, "herdr")
	}
	return filepath.Join(base, "sessions", c.Session, "herdr.sock")
}

type sessionRow struct {
	Name      string `json:"name"`
	Directory string `json:"directory"`
	Socket    string `json:"socket"`
}

func sessionsOf(v any) []sessionRow {
	var rows []sessionRow
	var walk func(any)
	walk = func(x any) {
		switch t := x.(type) {
		case []any:
			for _, e := range t {
				walk(e)
			}
		case map[string]any:
			if n, ok := t["name"].(string); ok {
				r := sessionRow{Name: n}
				r.Directory, _ = t["directory"].(string)
				if r.Directory == "" {
					r.Directory, _ = t["dir"].(string)
				}
				if r.Directory == "" {
					r.Directory, _ = t["session_dir"].(string)
				}
				r.Socket, _ = t["socket"].(string)
				if r.Socket == "" {
					r.Socket, _ = t["socket_path"].(string)
				}
				rows = append(rows, r)
				return
			}
			for _, e := range t {
				walk(e)
			}
		}
	}
	walk(v)
	return rows
}

type request struct {
	ID     string `json:"id"`
	Method string `json:"method"`
	Params any    `json:"params"`
}

type response struct {
	ID     string          `json:"id"`
	Result json.RawMessage `json:"result"`
	Error  *APIError       `json:"error"`
}

// Call sends one request and decodes the result into out (which may be nil).
func (c *Client) Call(ctx context.Context, method string, params, out any) error {
	if params == nil {
		params = struct{}{}
	}
	conn, err := c.Dial(ctx, c.Socket)
	if err != nil {
		return fmt.Errorf("%w (%s): %v", ErrNotRunning, c.Socket, err)
	}
	defer conn.Close()
	stop := context.AfterFunc(ctx, func() { conn.Close() })
	defer stop()
	id := fmt.Sprintf("gbi-%d", c.seq.Add(1))
	b, _ := json.Marshal(request{ID: id, Method: method, Params: params})
	if _, err := conn.Write(append(b, '\n')); err != nil {
		return fmt.Errorf("herdr %s: write: %w", method, err)
	}
	r := bufio.NewReaderSize(conn, 1<<20)
	line, err := r.ReadBytes('\n')
	if err != nil && len(line) == 0 {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return fmt.Errorf("herdr %s: read: %w", method, err)
	}
	var resp response
	if err := json.Unmarshal(line, &resp); err != nil {
		return fmt.Errorf("herdr %s: bad response: %w", method, err)
	}
	if resp.Error != nil {
		return resp.Error
	}
	if out != nil && len(resp.Result) > 0 {
		if err := json.Unmarshal(resp.Result, out); err != nil {
			return fmt.Errorf("herdr %s: decode result: %w", method, err)
		}
	}
	return nil
}

// Event is one pushed event: Event is its kind (e.g. "pane.agent_status_changed"),
// Data the raw payload with its "type" discriminator.
type Event struct {
	Event string          `json:"event"`
	Data  json.RawMessage `json:"data"`
}

// StatusChange is the payload of pane.agent_status_changed.
type StatusChange struct {
	PaneID       string      `json:"pane_id"`
	WorkspaceID  string      `json:"workspace_id"`
	Agent        string      `json:"agent"`
	AgentStatus  AgentStatus `json:"agent_status"`
	DisplayAgent string      `json:"display_agent"`
	Title        string      `json:"title"`
}

// PaneRef is the payload of pane.exited and pane.closed.
type PaneRef struct {
	PaneID      string `json:"pane_id"`
	WorkspaceID string `json:"workspace_id"`
}

// Sub is one subscription. pane.agent_status_changed and pane.scroll_changed need PaneID;
// lifecycle events (pane.created, pane.exited, workspace.closed, …) do not.
type Sub struct {
	Type   string
	PaneID string
}

// ErrEventsLost means the subscriber fell behind herdr's retained history; resubscribe
// and reconcile from session.snapshot.
var ErrEventsLost = errors.New("herdr events lost")

// Subscribe opens a long-lived subscription. Events arrive on the returned channel,
// which is closed when ctx ends or the connection drops; the final error (nil on
// ctx cancel, ErrEventsLost on overrun) is then sent on errc.
func (c *Client) Subscribe(ctx context.Context, specs ...Sub) (<-chan Event, <-chan error, error) {
	subs := make([]map[string]any, 0, len(specs))
	for _, sp := range specs {
		m := map[string]any{"type": sp.Type}
		if sp.PaneID != "" {
			m["pane_id"] = sp.PaneID
		}
		subs = append(subs, m)
	}
	conn, err := c.Dial(ctx, c.Socket)
	if err != nil {
		return nil, nil, fmt.Errorf("%w: %v", ErrNotRunning, err)
	}
	id := fmt.Sprintf("gbi-sub-%d", c.seq.Add(1))
	b, _ := json.Marshal(request{ID: id, Method: MethodEventsSubscribe, Params: map[string]any{"subscriptions": subs}})
	if _, err := conn.Write(append(b, '\n')); err != nil {
		conn.Close()
		return nil, nil, err
	}
	r := bufio.NewReaderSize(conn, 1<<20)
	first, err := r.ReadBytes('\n')
	if err != nil {
		conn.Close()
		return nil, nil, err
	}
	var ack response
	if err := json.Unmarshal(first, &ack); err != nil || ack.Error != nil {
		conn.Close()
		if ack.Error != nil {
			return nil, nil, ack.Error
		}
		return nil, nil, fmt.Errorf("herdr subscribe: %v", err)
	}
	ch, errc := make(chan Event, 256), make(chan error, 1)
	stop := context.AfterFunc(ctx, func() { conn.Close() })
	go func() {
		defer stop()
		defer conn.Close()
		defer close(ch)
		for {
			line, err := r.ReadBytes('\n')
			if len(line) > 0 {
				var resp response
				if json.Unmarshal(line, &resp) == nil && resp.Error != nil {
					if resp.Error.Code == "events_lost" {
						errc <- ErrEventsLost
					} else {
						errc <- resp.Error
					}
					return
				}
				var e Event
				if json.Unmarshal(line, &e) == nil && e.Event != "" {
					e.Event = normalizeEvent(e.Event)
					select {
					case ch <- e:
					case <-ctx.Done():
						errc <- nil
						return
					}
				}
			}
			if err != nil {
				if ctx.Err() != nil {
					errc <- nil
				} else {
					errc <- err
				}
				return
			}
		}
	}()
	return ch, errc, nil
}

// ---- typed helpers ----

// Pong is the ping result.
type Pong struct {
	Version      string              `json:"version"`
	Protocol     int64               `json:"protocol"`
	Capabilities *ServerCapabilities `json:"capabilities"`
}

// Ping checks the session answers.
func (c *Client) Ping(ctx context.Context) (Pong, error) {
	var p Pong
	err := c.Call(ctx, MethodPing, struct{}{}, &p)
	return p, err
}

// Snapshot returns the full current state.
func (c *Client) Snapshot(ctx context.Context) (SessionSnapshot, error) {
	var r struct {
		Snapshot SessionSnapshot `json:"snapshot"`
	}
	err := c.Call(ctx, MethodSessionSnapshot, struct{}{}, &r)
	return r.Snapshot, err
}

// Created is the result of workspace.create and worktree.open.
type Created struct {
	Workspace   WorkspaceInfo `json:"workspace"`
	Tab         TabInfo       `json:"tab"`
	RootPane    PaneInfo      `json:"root_pane"`
	AlreadyOpen bool          `json:"already_open"`
}

// CreateWorkspace opens a workspace at cwd.
func (c *Client) CreateWorkspace(ctx context.Context, cwd, label string) (Created, error) {
	var r Created
	err := c.Call(ctx, MethodWorkspaceCreate, WorkspaceCreateParams{Cwd: &cwd, Label: &label}, &r)
	return r, err
}

// OpenWorktree opens an existing checkout as a workspace grouped under its repository's workspace.
func (c *Client) OpenWorktree(ctx context.Context, workspaceID, path, label string) (Created, error) {
	var r Created
	p := WorktreeOpenParams{Path: &path, Label: &label}
	if workspaceID != "" {
		p.WorkspaceID = &workspaceID
	}
	err := c.Call(ctx, MethodWorktreeOpen, p, &r)
	return r, err
}

func agentOf(raw struct {
	Agent AgentInfo `json:"agent"`
}) AgentInfo {
	return raw.Agent
}

// StartAgent starts a supported agent kind in an idle shell pane.
func (c *Client) StartAgent(ctx context.Context, name, kind, paneID string, args []string, timeout time.Duration) (AgentInfo, error) {
	var r struct {
		Agent AgentInfo `json:"agent"`
	}
	ms := timeout.Milliseconds()
	p := AgentStartParams{Name: name, Kind: kind, PaneID: paneID, Args: args}
	if ms > 0 {
		p.TimeoutMs = &ms
	}
	err := c.Call(ctx, MethodAgentStart, p, &r)
	return agentOf(r), err
}

// Prompt submits text; with wait, it returns once the agent settles in one of until.
func (c *Client) Prompt(ctx context.Context, target, text string, wait bool, until []AgentStatus, timeout time.Duration) (AgentInfo, error) {
	var r struct {
		Agent AgentInfo `json:"agent"`
	}
	p := AgentPromptParams{Target: target, Text: text}
	if wait {
		ms := timeout.Milliseconds()
		w := AgentPromptWaitOptions{Until: until}
		if ms > 0 {
			w.TimeoutMs = &ms
		}
		p.Wait = &w
	}
	err := c.Call(ctx, MethodAgentPrompt, p, &r)
	return agentOf(r), err
}

// Wait blocks until the agent is in one of until.
func (c *Client) Wait(ctx context.Context, target string, until []AgentStatus, timeout time.Duration) (AgentInfo, error) {
	var r struct {
		Agent AgentInfo `json:"agent"`
	}
	ms := timeout.Milliseconds()
	p := AgentWaitParams{Target: target, Until: until}
	if ms > 0 {
		p.TimeoutMs = &ms
	}
	err := c.Call(ctx, MethodAgentWait, p, &r)
	return agentOf(r), err
}

// Agent returns one agent by name or pane ID.
func (c *Client) Agent(ctx context.Context, target string) (AgentInfo, error) {
	var r struct {
		Agent AgentInfo `json:"agent"`
	}
	err := c.Call(ctx, MethodAgentGet, AgentTarget{Target: target}, &r)
	return agentOf(r), err
}

// SendKeys sends key combos (e.g. "esc", "ctrl+c") to an agent.
func (c *Client) SendKeys(ctx context.Context, target string, keys ...string) error {
	return c.Call(ctx, MethodAgentSendKeys, AgentSendKeysParams{Target: target, Keys: keys}, nil)
}

// Pane returns one pane.
func (c *Client) Pane(ctx context.Context, paneID string) (PaneInfo, error) {
	var r struct {
		Pane PaneInfo `json:"pane"`
	}
	err := c.Call(ctx, MethodPaneGet, map[string]string{"pane_id": paneID}, &r)
	return r.Pane, err
}

// Read returns pane text (source: visible, recent, recent_unwrapped, detection).
func (c *Client) Read(ctx context.Context, paneID, source string, lines int) (string, error) {
	var r struct {
		Read PaneReadResult `json:"read"`
	}
	p := map[string]any{"pane_id": paneID, "source": source}
	if lines > 0 {
		p["lines"] = lines
	}
	err := c.Call(ctx, MethodPaneRead, p, &r)
	return r.Read.Text, err
}

// Metadata is what go-build-it shows on a member's pane (docs/specs/herdr-layout.md).
type Metadata struct {
	DisplayAgent string
	Title        string
	Tokens       map[string]string
	ClearTokens  []string // keys to remove (token updates are patches)
	StateLabels  map[string]string
}

// ReportMetadata labels a pane. Source is always "gbi".
func (c *Client) ReportMetadata(ctx context.Context, paneID string, m Metadata) error {
	p := PaneReportMetadataParams{PaneID: paneID, Source: "gbi", StateLabels: m.StateLabels}
	if m.DisplayAgent != "" {
		p.DisplayAgent = &m.DisplayAgent
	}
	if m.Title != "" {
		p.Title = &m.Title
	}
	if len(m.Tokens) > 0 || len(m.ClearTokens) > 0 {
		p.Tokens = map[string]*string{}
		for k, v := range m.Tokens {
			v := v
			p.Tokens[k] = &v
		}
		for _, k := range m.ClearTokens {
			p.Tokens[k] = nil
		}
	}
	return c.Call(ctx, MethodPaneReportMetadata, p, nil)
}

// ReportWorkspace sets workspace tokens (e.g. gbi = "2 working · 1 needs you").
func (c *Client) ReportWorkspace(ctx context.Context, workspaceID string, tokens map[string]string) error {
	t := map[string]*string{}
	for k, v := range tokens {
		v := v
		t[k] = &v
	}
	return c.Call(ctx, MethodWorkspaceReportMetadata, WorkspaceReportMetadataParams{WorkspaceID: workspaceID, Source: "gbi", Tokens: t}, nil)
}

// Notify shows a herdr notification (sound: none, done, request).
func (c *Client) Notify(ctx context.Context, title, body, sound string) error {
	p := NotificationShowParams{Title: title}
	if body != "" {
		p.Body = &body
	}
	if sound != "" {
		p.Sound = NotificationShowSound(sound)
	}
	return c.Call(ctx, MethodNotificationShow, p, nil)
}

// Integrations lists herdr's agent integrations and their state.
func (c *Client) Integrations(ctx context.Context) ([]IntegrationInfo, error) {
	var r struct {
		Integrations []IntegrationInfo `json:"integrations"`
	}
	err := c.Call(ctx, MethodIntegrationList, struct{}{}, &r)
	return r.Integrations, err
}

// InstallIntegration installs or updates herdr's integration for target (e.g. "claude").
func (c *Client) InstallIntegration(ctx context.Context, target string) error {
	return c.Call(ctx, MethodIntegrationInstall, IntegrationInstallParams{Target: IntegrationTarget(target)}, nil)
}

// SchemaMethods returns the methods the installed herdr binary's schema declares.
func (c *Client) SchemaMethods(ctx context.Context) (map[string]bool, error) {
	cmd := exec.CommandContext(ctx, c.Bin, "api", "schema", "--json")
	cmd.Env = c.Env
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	var s struct {
		Schemas struct {
			Request struct {
				OneOf []struct {
					Properties struct {
						Method struct {
							Const string `json:"const"`
						} `json:"method"`
					} `json:"properties"`
				} `json:"oneOf"`
			} `json:"request"`
		} `json:"schemas"`
	}
	if err := json.Unmarshal(out, &s); err != nil {
		return nil, err
	}
	m := map[string]bool{}
	for _, o := range s.Schemas.Request.OneOf {
		m[o.Properties.Method.Const] = true
	}
	return m, nil
}

// MissingMethods lists RequiredMethods the installed herdr does not declare.
func (c *Client) MissingMethods(ctx context.Context) ([]string, error) {
	have, err := c.SchemaMethods(ctx)
	if err != nil {
		return nil, err
	}
	var miss []string
	for _, m := range RequiredMethods {
		if !have[m] {
			miss = append(miss, m)
		}
	}
	return miss, nil
}

// SafeName turns a project and member into a herdr agent name: [a-z][a-z0-9_-]{0,31}.
func SafeName(parts ...string) string {
	var b strings.Builder
	for i, p := range parts {
		if i > 0 {
			b.WriteByte('-')
		}
		for _, r := range strings.ToLower(p) {
			switch {
			case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '_', r == '-':
				b.WriteRune(r)
			default:
				b.WriteByte('-')
			}
		}
	}
	s := strings.Trim(b.String(), "-")
	if s == "" || s[0] < 'a' || s[0] > 'z' {
		s = "a" + s
	}
	if len(s) > 32 {
		s = strings.TrimRight(s[:32], "-")
	}
	return s
}

// FromEnv returns a client for the herdr that owns the current pane (HERDR_SOCKET_PATH),
// or nil outside herdr. Agents use it to report their own state.
func FromEnv() *Client {
	sock := os.Getenv("HERDR_SOCKET_PATH")
	if os.Getenv("HERDR_ENV") != "1" || sock == "" {
		return nil
	}
	bin := os.Getenv("HERDR_BIN_PATH")
	if bin == "" {
		bin = "herdr"
	}
	return &Client{Bin: bin, Socket: sock, Dial: dial}
}

// ReportAgent reports a custom agent's state (idle, working, blocked) on a pane.
// sessionID and resume are optional (resume: argv that reopens the session after a herdr restart).
func (c *Client) ReportAgent(ctx context.Context, paneID, source, agent, state, message, sessionID string, resume []string, seq int64) error {
	p := map[string]any{"pane_id": paneID, "source": source, "agent": agent, "state": state, "seq": seq}
	if message != "" {
		p["message"] = message
	}
	if sessionID != "" {
		p["agent_session_id"] = sessionID
	}
	if len(resume) > 0 {
		p["resume_argv"] = resume
	}
	return c.Call(ctx, MethodPaneReportAgent, p, nil)
}

// ReleaseAgent clears a custom agent from its pane.
func (c *Client) ReleaseAgent(ctx context.Context, paneID, source, agent string, seq int64) error {
	return c.Call(ctx, MethodPaneReleaseAgent, map[string]any{"pane_id": paneID, "source": source, "agent": agent, "seq": seq}, nil)
}

// Rename gives the agent in a pane a name; agent.prompt only targets named agents.
func (c *Client) Rename(ctx context.Context, target, name string) error {
	return c.Call(ctx, MethodAgentRename, AgentRenameParams{Target: target, Name: &name}, nil)
}

// ErrPromptStalled means the agent never changed state after a prompt was sent.
var ErrPromptStalled = errors.New("the agent did not start working after the prompt")

// Submit sends a prompt to the agent in paneID and returns the agent's state-change
// sequence from just before sending, for WaitSettled. Known herdr agent kinds use
// agent.prompt (bracketed paste aware); custom agents (reported with pane.report_agent,
// such as the fake member) get the text typed into the pane followed by Enter.
func (c *Client) Submit(ctx context.Context, paneID, name string, known bool, text string) (int64, error) {
	a, err := c.Agent(ctx, paneID)
	if err != nil {
		return 0, err
	}
	if known {
		target := name
		if target == "" {
			target = paneID
		}
		if _, err := c.Prompt(ctx, target, text, false, nil, 0); err != nil {
			return 0, err
		}
		return a.StateChangeSeq, nil
	}
	return a.StateChangeSeq, c.Call(ctx, MethodPaneSendText, map[string]any{"pane_id": paneID, "text": text + "\n"}, nil)
}

// WaitSettled waits until the agent's state has changed since seq and is idle, done or
// blocked. It returns ErrPromptStalled when nothing changes within stall.
func (c *Client) WaitSettled(ctx context.Context, paneID string, seq int64, stall time.Duration) (AgentInfo, error) {
	if stall <= 0 {
		stall = 15 * time.Second
	}
	start, sawChange := time.Now(), false
	for {
		a, err := c.Agent(ctx, paneID)
		if err == nil {
			if a.StateChangeSeq > seq {
				sawChange = true
				switch a.AgentStatus {
				case AgentStatusIdle, AgentStatusDone, AgentStatusBlocked:
					return a, nil
				}
			}
		} else if !IsCode(err, "agent_not_found") && !IsCode(err, "not_found") {
			return a, err
		}
		if !sawChange && time.Since(start) > stall {
			return a, ErrPromptStalled
		}
		select {
		case <-ctx.Done():
			return a, ctx.Err()
		case <-time.After(200 * time.Millisecond):
		}
	}
}

// normalizeEvent maps lifecycle event kinds ("pane_closed") to the dotted form used for
// subscriptions and status events ("pane.closed").
func normalizeEvent(k string) string {
	if strings.Contains(k, ".") {
		return k
	}
	return strings.Replace(k, "_", ".", 1)
}

func envValue(env []string, key string) string {
	if env == nil {
		return os.Getenv(key)
	}
	for i := len(env) - 1; i >= 0; i-- {
		if strings.HasPrefix(env[i], key+"=") {
			return env[i][len(key)+1:]
		}
	}
	return ""
}

// NewTab opens a tab (with one pane) in a workspace at cwd and returns the pane ID.
func (c *Client) NewTab(ctx context.Context, workspaceID, cwd, label string, env map[string]string) (string, error) {
	var r struct {
		Tab      TabInfo  `json:"tab"`
		RootPane PaneInfo `json:"root_pane"`
	}
	p := map[string]any{"workspace_id": workspaceID, "cwd": cwd, "label": label, "focus": false}
	if len(env) > 0 {
		p["env"] = env
	}
	if err := c.Call(ctx, MethodTabCreate, p, &r); err != nil {
		return "", err
	}
	return r.RootPane.PaneID, nil
}
