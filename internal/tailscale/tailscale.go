// Package tailscale detects Tailscale and configures Tailscale Serve for go-build-it
// (FR-20, FR-20a–FR-20d, ADR-015). It reads the machine's whole Serve configuration, so it
// can prove that go-build-it is the one being served, show what else this machine serves,
// and never take an address another service already uses.
package tailscale

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"

	"rsc.io/qr"
)

// Check is one line of the remote access checklist (FR-20a).
type Check struct {
	ID     string `json:"id"`     // installed | logged_in | rule | reachable | private | conflicts
	Title  string `json:"title"`  // what is checked
	Status string `json:"status"` // ok | warn | fail | skip
	Detail string `json:"detail"`
	Fix    string `json:"fix,omitempty"` // what to do, as a command when there is one
}

// Rule is one entry of the machine's Serve configuration.
type Rule struct {
	URL     string `json:"url"`     // what the tailnet sees, e.g. https://host.ts.net:8443/
	Port    int    `json:"port"`    // tailnet port
	Path    string `json:"path"`    // mount path ("/" for the root; "" for TCP forwards)
	Kind    string `json:"kind"`    // https | http | tcp
	Target  string `json:"target"`  // where it goes, e.g. http://127.0.0.1:7420
	Funnel  bool   `json:"funnel"`  // also on the public internet
	Mine    bool   `json:"mine"`    // goes to this go-build-it
	Process string `json:"process"` // the local program listening on the target, when known
}

// Proposal is how Setup would serve go-build-it (FR-20b).
type Proposal struct {
	Port         int    `json:"port"`
	URL          string `json:"url"`
	Command      string `json:"command"`
	NeedsConfirm bool   `json:"needsConfirm"` // the default address is taken, so a person agrees first
	Reason       string `json:"reason"`
}

// Status describes Tailscale and how go-build-it is served on the tailnet.
type Status struct {
	Installed bool      `json:"installed"`
	LoggedIn  bool      `json:"loggedIn"`
	DNSName   string    `json:"dnsName"`
	ServeURL  string    `json:"serveUrl"`  // go-build-it's tailnet URL, when a rule serves it
	Serving   bool      `json:"serving"`   // a Serve rule points at go-build-it
	Reachable bool      `json:"reachable"` // the URL answered, and it was this go-build-it
	Private   bool      `json:"private"`   // not on the public internet (no Funnel)
	Guidance  string    `json:"guidance"`
	Checks    []Check   `json:"checks"`
	Rules     []Rule    `json:"rules"`     // everything this machine serves
	Conflicts []Rule    `json:"conflicts"` // other services on the address go-build-it would use
	Proposal  *Proposal `json:"proposal,omitempty"`
	QR        []string  `json:"qr,omitempty"` // the ServeURL as QR modules, one string of 0/1 per row
}

// Env is what Detect needs from the outside world; tests replace it.
type Env struct {
	Run     func(ctx context.Context, args ...string) (string, error) // runs the tailscale CLI
	Fetch   func(ctx context.Context, url string) (int, []byte, error)
	Process func(port int) string // the program listening on 127.0.0.1:port
	Look    func() bool           // the tailscale CLI is installed
}

// System is the real machine.
var System = Env{Run: run, Fetch: fetch, Process: process, Look: func() bool { _, err := exec.LookPath("tailscale"); return err == nil }}

func run(ctx context.Context, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "tailscale", args...).CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

func fetch(ctx context.Context, u string) (int, []byte, error) {
	ctx, cancel := context.WithTimeout(ctx, 6*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return 0, nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	var b strings.Builder
	buf := make([]byte, 4096)
	for b.Len() < 1<<16 {
		n, err := resp.Body.Read(buf)
		b.Write(buf[:n])
		if err != nil {
			break
		}
	}
	return resp.StatusCode, []byte(b.String()), nil
}

func process(port int) string {
	out, err := exec.Command("lsof", "-nP", "-iTCP:"+strconv.Itoa(port), "-sTCP:LISTEN", "-Fc").Output()
	if err != nil {
		return ""
	}
	for _, l := range strings.Split(string(out), "\n") {
		if strings.HasPrefix(l, "c") {
			return strings.TrimPrefix(l, "c")
		}
	}
	return ""
}

// serveConfig is the part of `tailscale serve status --json` go-build-it reads.
type serveConfig struct {
	TCP map[string]struct {
		HTTPS      bool
		HTTP       bool
		TCPForward string
	}
	Web map[string]struct {
		Handlers map[string]struct {
			Proxy string
			Path  string
			Text  string
		}
	}
	AllowFunnel map[string]bool
}

// Detect reports Tailscale, everything Serve publishes on this machine, and whether
// go-build-it (listening on listen, with data in dataDir) is reachable on the tailnet.
func Detect(ctx context.Context, listen, dataDir string) Status {
	return System.Detect(ctx, listen, dataDir)
}

// Detect is Detect with this environment.
func (e Env) Detect(ctx context.Context, listen, dataDir string) Status {
	s := Status{Checks: []Check{}, Rules: []Rule{}, Conflicts: []Rule{}}
	add := func(id, title, st, detail, fix string) {
		s.Checks = append(s.Checks, Check{ID: id, Title: title, Status: st, Detail: detail, Fix: fix})
	}
	if !e.Look() {
		s.Guidance = "Tailscale is not installed. Install it from https://tailscale.com/download, log in, then come back."
		add("installed", "Tailscale is installed", "fail", "the tailscale command was not found", "Install Tailscale from https://tailscale.com/download")
		return s
	}
	s.Installed = true
	add("installed", "Tailscale is installed", "ok", "", "")
	out, err := e.Run(ctx, "status", "--json")
	var st struct {
		BackendState string
		Self         struct{ DNSName string }
	}
	if err == nil {
		_ = json.Unmarshal([]byte(out), &st)
	}
	s.LoggedIn = st.BackendState == "Running"
	s.DNSName = strings.TrimSuffix(st.Self.DNSName, ".")
	if !s.LoggedIn {
		s.Guidance = "Start Tailscale and log in (`tailscale up`), then come back."
		add("logged_in", "Tailscale is running and logged in", "fail", firstNonEmpty(st.BackendState, "not running"), "tailscale up")
		return s
	}
	add("logged_in", "Tailscale is running and logged in", "ok", s.DNSName, "")

	cfg := serveConfig{}
	if out, err := e.Run(ctx, "serve", "status", "--json"); err == nil && strings.HasPrefix(strings.TrimSpace(out), "{") {
		_ = json.Unmarshal([]byte(out), &cfg)
	}
	s.Rules = e.rules(cfg, s.DNSName, listen)
	var mine *Rule
	for i := range s.Rules {
		if s.Rules[i].Mine && (mine == nil || s.Rules[i].Port == 443) {
			mine = &s.Rules[i]
		}
	}
	if mine == nil {
		s.Proposal = propose(cfg, s.Rules, s.DNSName, listen)
		s.Conflicts = conflictsAt(s.Rules, 443)
		detail := "nothing on this tailnet address serves go-build-it yet"
		add("rule", "A Serve rule points at go-build-it", "warn", detail, s.Proposal.Command)
		add("reachable", "go-build-it answers on the tailnet", "skip", "no rule yet", "")
		add("private", "Not on the public internet (Funnel off)", "skip", "no rule yet", "")
		if len(s.Conflicts) > 0 {
			add("conflicts", "No other service on go-build-it's address", "warn", s.Proposal.Reason, "")
			s.Guidance = s.Proposal.Reason + " Serve go-build-it on " + s.Proposal.URL + " instead."
		} else {
			add("conflicts", "No other service on go-build-it's address", "ok", "", "")
			s.Guidance = "Tailscale is ready. Set up Serve to reach go-build-it at " + s.Proposal.URL
		}
		return s
	}
	s.Serving, s.ServeURL = true, strings.TrimSuffix(mine.URL, "/")
	add("rule", "A Serve rule points at go-build-it", "ok", mine.URL+" → "+mine.Target, "")

	code, body, ferr := e.Fetch(ctx, s.ServeURL+"/api/system")
	var info struct{ DataDir, Version string }
	_ = json.Unmarshal(body, &info)
	switch {
	case ferr != nil:
		add("reachable", "go-build-it answers on the tailnet", "fail", "no answer from "+s.ServeURL+": "+ferr.Error(), "Check that go-build-it is running (gbi service install), then refresh")
	case code != http.StatusOK || info.DataDir == "":
		add("reachable", "go-build-it answers on the tailnet", "fail", fmt.Sprintf("%s answered %d, not go-build-it", s.ServeURL, code), "Another program answers on this address; set Serve up again")
	case dataDir != "" && info.DataDir != dataDir:
		add("reachable", "go-build-it answers on the tailnet", "fail", "a different go-build-it answered (data in "+info.DataDir+")", "Stop the other copy, or serve this one on its own port")
	default:
		s.Reachable = true
		add("reachable", "go-build-it answers on the tailnet", "ok", "this go-build-it answered (version "+info.Version+")", "")
	}
	s.Private = !mine.Funnel
	if mine.Funnel {
		add("private", "Not on the public internet (Funnel off)", "fail", "Funnel is on: anyone on the internet can reach go-build-it",
			fmt.Sprintf("tailscale funnel --https=%d off", mine.Port))
	} else {
		add("private", "Not on the public internet (Funnel off)", "ok", "tailnet only", "")
	}
	s.Conflicts = []Rule{}
	for _, r := range conflictsAt(s.Rules, mine.Port) {
		if r.Path == mine.Path {
			s.Conflicts = append(s.Conflicts, r)
		}
	}
	others := 0
	for _, r := range s.Rules {
		if !r.Mine {
			others++
		}
	}
	add("conflicts", "No other service on go-build-it's address", "ok", fmt.Sprintf("%d other services served from this machine, none on %s", others, s.ServeURL), "")
	if s.Reachable && s.Private {
		s.QR = QR(s.ServeURL)
	} else {
		s.Guidance = "go-build-it has a Serve rule but it is not working yet: see the checks."
	}
	return s
}

// rules flattens a Serve configuration.
func (e Env) rules(cfg serveConfig, dns, listen string) []Rule {
	out := []Rule{}
	for hp, w := range cfg.Web {
		host, ps, _ := net.SplitHostPort(hp)
		port, _ := strconv.Atoi(ps)
		kind := "https"
		if t, ok := cfg.TCP[ps]; ok && t.HTTP && !t.HTTPS {
			kind = "http"
		}
		for path, h := range w.Handlers {
			target := h.Proxy
			if target == "" {
				target = firstNonEmpty(h.Path, "text")
			}
			r := Rule{URL: webURL(kind, host, port, path), Port: port, Path: path, Kind: kind, Target: target, Funnel: cfg.AllowFunnel[hp]}
			r.Mine = path == "/" && sameLocal(target, listen)
			out = append(out, r)
		}
	}
	for ps, t := range cfg.TCP {
		if t.TCPForward == "" {
			continue
		}
		port, _ := strconv.Atoi(ps)
		out = append(out, Rule{URL: fmt.Sprintf("tcp://%s:%d", dns, port), Port: port, Kind: "tcp", Target: t.TCPForward,
			Funnel: cfg.AllowFunnel[net.JoinHostPort(dns, ps)]})
	}
	for i := range out {
		if p := localPort(out[i].Target); p > 0 && e.Process != nil {
			out[i].Process = e.Process(p)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Port != out[j].Port {
			return out[i].Port < out[j].Port
		}
		return out[i].Path < out[j].Path
	})
	return out
}

func webURL(kind, host string, port int, path string) string {
	u := kind + "://" + host
	if !(kind == "https" && port == 443) && !(kind == "http" && port == 80) {
		u += ":" + strconv.Itoa(port)
	}
	return u + path
}

// conflictsAt lists other services on a tailnet port's root, or a TCP forward on it.
func conflictsAt(rs []Rule, port int) []Rule {
	out := []Rule{}
	for _, r := range rs {
		if !r.Mine && r.Port == port && (r.Path == "/" || r.Kind == "tcp") {
			out = append(out, r)
		}
	}
	return out
}

// propose picks the address for go-build-it: the root of 443 when it is free, else its
// own HTTPS port (the local port when free, else the first free one from 8443).
func propose(cfg serveConfig, rs []Rule, dns, listen string) *Proposal {
	target := "http://" + localTarget(listen)
	if len(conflictsAt(rs, 443)) == 0 {
		return &Proposal{Port: 443, URL: "https://" + dns, Command: fmt.Sprintf("tailscale serve --bg --https=443 %s", target)}
	}
	used := map[int]bool{}
	for ps := range cfg.TCP {
		p, _ := strconv.Atoi(ps)
		used[p] = true
	}
	port := localPort(target)
	if port <= 0 || used[port] {
		for port = 8443; used[port]; port++ {
		}
	}
	c := conflictsAt(rs, 443)[0]
	who := c.Target
	if c.Process != "" {
		who = c.Process + " (" + c.Target + ")"
	}
	return &Proposal{Port: port, URL: fmt.Sprintf("https://%s:%d", dns, port), NeedsConfirm: true,
		Command: fmt.Sprintf("tailscale serve --bg --https=%d %s", port, target),
		Reason:  fmt.Sprintf("https://%s is already used by %s; go-build-it will not replace it.", dns, who)}
}

// Setup serves go-build-it on the tailnet (FR-20b). port 0 takes the proposal. When the
// proposal is not the default address, confirm must be true: a person agreed to it.
func Setup(ctx context.Context, listen, dataDir string, port int, confirm bool) (Status, error) {
	return System.Setup(ctx, listen, dataDir, port, confirm)
}

// Setup is Setup with this environment.
func (e Env) Setup(ctx context.Context, listen, dataDir string, port int, confirm bool) (Status, error) {
	s := e.Detect(ctx, listen, dataDir)
	if !s.LoggedIn {
		return s, errors.New(s.Guidance)
	}
	if s.Serving {
		return s, nil
	}
	p := s.Proposal
	if port != 0 && port != p.Port {
		for _, r := range s.Rules {
			if r.Port == port && (r.Path == "/" || r.Kind == "tcp") {
				return s, fmt.Errorf("port %d is already used by %s; choose another", port, firstNonEmpty(r.Process, r.Target))
			}
		}
		p = &Proposal{Port: port, NeedsConfirm: true, Command: fmt.Sprintf("tailscale serve --bg --https=%d http://%s", port, localTarget(listen))}
	}
	if p.NeedsConfirm && !confirm {
		return s, fmt.Errorf("%s To serve go-build-it at its own port, confirm: %s", firstNonEmpty(p.Reason, "This is not the default address."), p.Command)
	}
	if out, err := e.Run(ctx, "serve", "--bg", fmt.Sprintf("--https=%d", p.Port), "http://"+localTarget(listen)); err != nil {
		return s, fmt.Errorf("tailscale serve: %v: %s", err, out)
	}
	return e.Detect(ctx, listen, dataDir), nil
}

// Stop removes go-build-it's own Serve rules and nothing else (FR-20b).
func Stop(ctx context.Context, listen, dataDir string) (Status, error) {
	return System.Stop(ctx, listen, dataDir)
}

// Stop is Stop with this environment.
func (e Env) Stop(ctx context.Context, listen, dataDir string) (Status, error) {
	s := e.Detect(ctx, listen, dataDir)
	for _, r := range s.Rules {
		if !r.Mine {
			continue
		}
		flag := fmt.Sprintf("--%s=%d", r.Kind, r.Port)
		if out, err := e.Run(ctx, "serve", flag, "off"); err != nil {
			return s, fmt.Errorf("tailscale serve %s off: %v: %s", flag, err, out)
		}
	}
	return e.Detect(ctx, listen, dataDir), nil
}

// sameLocal reports whether a proxy target is go-build-it's listen address on this machine.
func sameLocal(target, listen string) bool {
	tp := localPort(target)
	_, lps, err := net.SplitHostPort(listen)
	if err != nil || tp <= 0 {
		return false
	}
	lp, _ := strconv.Atoi(lps)
	return tp == lp
}

// localPort returns the port of a target on this machine (127.0.0.1, localhost, ::1), or 0.
func localPort(target string) int {
	if !strings.Contains(target, "://") {
		target = "tcp://" + target
	}
	u, err := url.Parse(target)
	if err != nil {
		return 0
	}
	switch u.Hostname() {
	case "127.0.0.1", "localhost", "::1", "0.0.0.0", "":
	default:
		return 0
	}
	p, _ := strconv.Atoi(u.Port())
	return p
}

// localTarget is the loopback address Serve should proxy to for listen.
func localTarget(listen string) string {
	host, port, err := net.SplitHostPort(listen)
	if err != nil {
		return listen
	}
	if host == "" || host == "0.0.0.0" || host == "::" {
		host = "127.0.0.1"
	}
	return net.JoinHostPort(host, port)
}

// QR encodes text as QR modules, one string of 0/1 per row, with a 4-module quiet zone.
func QR(text string) []string {
	c, err := qr.Encode(text, qr.M)
	if err != nil {
		return nil
	}
	const quiet = 4
	n := c.Size + 2*quiet
	out := make([]string, n)
	for y := 0; y < n; y++ {
		var b strings.Builder
		for x := 0; x < n; x++ {
			if x >= quiet && y >= quiet && x < n-quiet && y < n-quiet && c.Black(x-quiet, y-quiet) {
				b.WriteByte('1')
			} else {
				b.WriteByte('0')
			}
		}
		out[y] = b.String()
	}
	return out
}

// Terminal draws QR modules with half blocks, two rows per line, for `gbi tailscale qr`.
// Blocks are the light modules: terminals draw text light on dark, so the code reads
// dark-on-light to a phone camera.
func Terminal(rows []string) string {
	var b strings.Builder
	for y := 0; y < len(rows); y += 2 {
		for x := 0; x < len(rows[y]); x++ {
			top := rows[y][x] == '0'
			bot := y+1 >= len(rows) || rows[y+1][x] == '0'
			switch {
			case top && bot:
				b.WriteString("█")
			case top:
				b.WriteString("▀")
			case bot:
				b.WriteString("▄")
			default:
				b.WriteString(" ")
			}
		}
		b.WriteString("\n")
	}
	return b.String()
}

func firstNonEmpty(s ...string) string {
	for _, x := range s {
		if x != "" {
			return x
		}
	}
	return ""
}
