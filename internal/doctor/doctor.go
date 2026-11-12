package doctor

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/techmuch/go-build-it/internal/config"
	"github.com/techmuch/go-build-it/internal/harness"
	"github.com/techmuch/go-build-it/internal/herdr"
	"github.com/techmuch/go-build-it/internal/service"
	"github.com/techmuch/go-build-it/internal/sessions"
	"github.com/techmuch/go-build-it/internal/tailscale"
)

// Tier is a group of checks. Each tier runs only when the tiers before it had no failure.
type Tier string

const (
	Static      Tier = "static"  // look only: herdr, its API, harness binaries, the login service
	Live        Tier = "live"    // start the gbi session, open a pane, run a command in it
	HarnessTier Tier = "harness" // start each harness in herdr; no prompt
	Prompt      Tier = "prompt"  // one short prompt per harness; uses tokens, opt-in only (Q11)
)

// Tiers in the order they run.
var Tiers = []Tier{Static, Live, HarnessTier, Prompt}

// Status of a result.
type Status string

const (
	OK      Status = "ok"
	Warn    Status = "warn"
	Fail    Status = "fail"
	Skipped Status = "skipped"
)

// Evidence is what the doctor saw.
type Evidence struct {
	Command string `json:"command,omitempty"`
	Exit    *int   `json:"exit,omitempty"`
	Output  string `json:"output,omitempty"`
}

// Fix says how to resolve a result. Kind "auto" can be applied by the doctor
// (`gbi doctor --fix`, or the UI); kind "human" must be done by a person, and
// Instructions are written so an AI agent can pass them on word for word.
type Fix struct {
	Kind         string `json:"kind"`
	Action       string `json:"action,omitempty"`
	Instructions string `json:"instructions"`
}

// Result is one check's outcome (FR-07c).
type Result struct {
	ID       string    `json:"id"`
	Tier     Tier      `json:"tier"`
	Status   Status    `json:"status"`
	Title    string    `json:"title"`
	Detail   string    `json:"detail"`
	Harness  string    `json:"harness,omitempty"`
	Evidence *Evidence `json:"evidence,omitempty"`
	Fix      *Fix      `json:"fix,omitempty"`
	Docs     string    `json:"docs"`
	Ms       int64     `json:"ms"`
}

// TierRun summarises one tier.
type TierRun struct {
	Tier  Tier   `json:"tier"`
	Ran   bool   `json:"ran"`
	OK    int    `json:"ok"`
	Total int    `json:"total"`
	Note  string `json:"note,omitempty"`
}

// Report is a whole doctor run.
type Report struct {
	Version string    `json:"version"`
	OS      string    `json:"os"`
	Arch    string    `json:"arch"`
	Herdr   string    `json:"herdr,omitempty"`
	Started time.Time `json:"started"`
	Ms      int64     `json:"ms"`
	Tiers   []TierRun `json:"tiers"`
	Results []Result  `json:"results"`
	// Exit is the CLI exit code: 0 nothing failed, 1 every failure has an automatic fix,
	// 2 a failure needs a human (FR-07d).
	Exit    int      `json:"exit"`
	Summary string   `json:"summary"`
	Applied []string `json:"applied,omitempty"` // fixes applied by --fix before this run
}

// Options choose what runs.
type Options struct {
	Through Tier   `json:"through"` // last tier to run: static, live (default) or harness
	Harness string `json:"harness"` // only this harness ("" = every enabled one)
	Prompt  bool   `json:"prompt"`  // also run the prompt tier (uses tokens)
}

// Doctor runs the checks.
type Doctor struct {
	Cfg     *config.Config
	Sup     *herdr.Supervisor
	Exe     string // the gbi binary: the scripted harness and the login service run it
	Version string
	// Service returns the login service plan; nil uses service.Build. Tests replace it.
	Service func() (service.Plan, error)
	// Tailscale is the machine the remote access check reads; nil uses the real one. Tests replace it.
	Tailscale *tailscale.Env
	// StartTimeout bounds how long a harness may take to become ready (default 45s).
	StartTimeout time.Duration
}

// WorkspaceLabel is the herdr workspace that holds the doctor's throwaway panes.
const WorkspaceLabel = "gbi-doctor"

// PromptText is the prompt tier's prompt.
const PromptText = "Reply with the single word READY."

type probe struct {
	name, label, bin, kind, integration string
	prof                                harness.Profile
	pane                                string
	ok                                  bool // still eligible for the next tier
}

// Run runs the tiers chosen in o.
func (d *Doctor) Run(ctx context.Context, o Options) Report {
	start := time.Now()
	rep := Report{Version: d.Version, OS: runtime.GOOS, Arch: runtime.GOARCH, Started: start}
	if o.Through == "" {
		o.Through = Live
	}
	if o.Prompt {
		o.Through = HarnessTier
	}
	want := map[Tier]bool{Static: true}
	for _, t := range []Tier{Live, HarnessTier} {
		want[t] = want[t] || tierIndex(o.Through) >= tierIndex(t)
	}
	want[Prompt] = o.Prompt

	probes := d.probes(o.Harness)
	var ws string
	defer func() {
		if ws != "" {
			c, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			_ = d.Sup.C.Call(c, herdr.MethodWorkspaceClose, map[string]any{"workspace_id": ws}, nil)
			cancel()
		}
	}()
	blocked := ""
	for _, t := range Tiers {
		tr := TierRun{Tier: t}
		switch {
		case !want[t]:
			tr.Note = "not run"
			if t == Prompt {
				tr.Note = "not run: uses tokens, run with --prompt"
			}
		case blocked != "":
			tr.Note = "skipped: " + blocked
		default:
			tr.Ran = true
			before := len(rep.Results)
			switch t {
			case Static:
				rep.Results = append(rep.Results, d.static(ctx, &rep, probes)...)
			case Live:
				var rs []Result
				ws, rs = d.live(ctx, probes)
				rep.Results = append(rep.Results, rs...)
			case HarnessTier:
				rep.Results = append(rep.Results, d.harnessTier(ctx, ws, probes, o.Prompt)...)
			case Prompt:
				rep.Results = append(rep.Results, d.promptTier(ctx, probes)...)
			}
			for _, r := range rep.Results[before:] {
				tr.Total++
				if r.Status == OK || r.Status == Warn {
					tr.OK++
				}
				if r.Status == Fail && blocked == "" {
					blocked = "the " + string(t) + " tier failed"
				}
			}
		}
		rep.Tiers = append(rep.Tiers, tr)
	}
	for i := range rep.Results {
		rep.Results[i].Docs = DocsURL(rep.Results[i].ID)
	}
	rep.Ms = time.Since(start).Milliseconds()
	rep.Exit, rep.Summary = exitCode(rep.Results)
	return rep
}

func tierIndex(t Tier) int {
	for i, x := range Tiers {
		if x == t {
			return i
		}
	}
	return 1
}

func exitCode(rs []Result) (int, string) {
	fails, human := 0, 0
	warns := 0
	for _, r := range rs {
		switch r.Status {
		case Fail:
			fails++
			if r.Fix == nil || r.Fix.Kind != "auto" {
				human++
			}
		case Warn:
			warns++
		}
	}
	switch {
	case fails == 0 && warns == 0:
		return 0, "all good"
	case fails == 0:
		return 0, fmt.Sprintf("working, %d to improve", warns)
	case human == 0:
		return 1, fmt.Sprintf("%d to fix, all automatic: run gbi doctor --fix", fails)
	default:
		return 2, fmt.Sprintf("%d to fix, %d needs a human", fails, human)
	}
}

// probes lists the enabled harnesses (the scripted one only when enabled in config).
func (d *Doctor) probes(only string) []*probe {
	var names []string
	for name, hc := range d.Cfg.Harnesses {
		if hc.Enabled && (only == "" || only == name) {
			names = append(names, name)
		}
	}
	if only != "" && len(names) == 0 {
		names = []string{only} // asked for by name: check it even when disabled
	}
	sort.Strings(names)
	var out []*probe
	for _, n := range names {
		prof, ok := harness.Get(n)
		if !ok {
			continue
		}
		hc := d.Cfg.Harnesses[n]
		bin := firstNonEmpty(hc.Binary, prof.Binary)
		if n == "fake" {
			bin = firstNonEmpty(hc.Binary, d.Exe)
		}
		out = append(out, &probe{name: n, label: prof.Label, bin: bin, kind: prof.Kind, integration: prof.Integration, prof: prof})
	}
	return out
}

func res(id string, st Status, detail string, p *probe) Result {
	c, _ := CheckByID(id)
	r := Result{ID: id, Tier: c.Tier, Status: st, Title: c.Title, Detail: detail}
	if p != nil {
		r.Harness = p.name
		r.Title = fmt.Sprintf(c.Title, p.label)
	}
	return r
}

func human(s string) *Fix        { return &Fix{Kind: "human", Instructions: s} }
func auto(action, s string) *Fix { return &Fix{Kind: "auto", Action: action, Instructions: s} }
func intp(i int) *int            { return &i }
func since(t time.Time) int64    { return time.Since(t).Milliseconds() }
func firstNonEmpty(s ...string) string {
	for _, x := range s {
		if x != "" {
			return x
		}
	}
	return ""
}

// ---- static ----

func (d *Doctor) static(ctx context.Context, rep *Report, probes []*probe) []Result {
	var out []Result
	t0 := time.Now()
	bin, err := exec.LookPath(d.Sup.C.Bin)
	if err != nil {
		r := res("herdr.installed", Fail, "herdr was not found on PATH ("+d.Sup.C.Bin+")", nil)
		r.Evidence = &Evidence{Command: "which " + d.Sup.C.Bin, Exit: intp(1)}
		r.Fix = human("Install herdr: " + herdr.InstallHint() + ". Then run gbi doctor again.")
		out = append(out, r)
	} else {
		r := res("herdr.installed", OK, bin, nil)
		r.Ms = since(t0)
		out = append(out, r)

		t0 = time.Now()
		vctx, cancel := context.WithTimeout(ctx, 10*time.Second)
		vout, verr := exec.CommandContext(vctx, bin, "--version").CombinedOutput()
		cancel()
		ver := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(string(vout)), "herdr"))
		rep.Herdr = ver
		ev := &Evidence{Command: bin + " --version", Exit: intp(exitOf(verr)), Output: strings.TrimSpace(string(vout))}
		switch {
		case verr != nil || ver == "":
			r = res("herdr.version", Warn, "could not read herdr's version", nil)
			r.Fix = human("Run `herdr --version` in a terminal; if it fails, reinstall herdr (" + herdr.InstallHint() + ").")
		case herdr.CompareVersions(ver, herdr.MinVersion) < 0:
			r = res("herdr.version", Fail, fmt.Sprintf("herdr %s is older than %s", ver, herdr.MinVersion), nil)
			r.Fix = human(fmt.Sprintf("Update herdr to %s or newer (`herdr update`, or your package manager), then run gbi doctor again.", herdr.MinVersion))
		case herdr.CompareVersions(ver, herdr.TestedMax) > 0:
			r = res("herdr.version", Warn, fmt.Sprintf("herdr %s is newer than the version go-build-it was tested with (%s); it should work", ver, herdr.TestedMax), nil)
		default:
			r = res("herdr.version", OK, fmt.Sprintf("%s (supported ≥ %s)", ver, herdr.MinVersion), nil)
		}
		r.Evidence, r.Ms = ev, since(t0)
		out = append(out, r)

		t0 = time.Now()
		sctx, cancel := context.WithTimeout(ctx, 10*time.Second)
		miss, merr := d.Sup.C.MissingMethods(sctx)
		cancel()
		switch {
		case merr != nil:
			r = res("herdr.api", Warn, "could not read herdr's API schema: "+merr.Error(), nil)
		case len(miss) > 0:
			r = res("herdr.api", Fail, "missing methods: "+strings.Join(miss, ", "), nil)
			r.Fix = human("Update herdr (`herdr update`, or your package manager); this herdr lacks methods go-build-it needs.")
		default:
			r = res("herdr.api", OK, fmt.Sprintf("all %d required methods present", len(herdr.RequiredMethods)), nil)
		}
		r.Evidence, r.Ms = &Evidence{Command: bin + " api schema --json"}, since(t0)
		out = append(out, r)
	}

	installed := 0
	for _, p := range probes {
		t0 = time.Now()
		path, err := lookPath(p.bin)
		var r Result
		if err != nil {
			r = res("harness.installed", Warn, p.bin+" was not found", p)
			r.Evidence = &Evidence{Command: "which " + p.bin, Exit: intp(1)}
			r.Fix = auto("disable-harness:"+p.name, fmt.Sprintf("Install %s and run gbi doctor again, or disable it (gbi doctor --fix disables it) if you do not use it.", p.label))
		} else {
			installed++
			p.ok = true
			r = res("harness.installed", OK, path, p)
		}
		r.Ms = since(t0)
		out = append(out, r)
	}
	if installed == 0 {
		r := res("harness.any", Fail, "no enabled agent harness was found on this machine", nil)
		r.Fix = human("Install an agent harness (Claude Code, Antigravity, Pi or OpenCode) by following its own install instructions, sign in to it once in a terminal, then run gbi doctor again.")
		out = append(out, r)
	}

	t0 = time.Now()
	plan, perr := d.servicePlan()
	svcInstalled := false
	var svc Result
	switch {
	case perr != nil:
		svc = res("service.installed", Warn, "could not plan the login service: "+perr.Error(), nil)
	case plan.File == "":
		svc = res("service.installed", Skipped, "not checked: no service file on "+runtime.GOOS, nil)
	default:
		if _, err := os.Stat(plan.File); err == nil {
			svcInstalled = true
			svc = res("service.installed", OK, plan.File, nil)
		} else {
			svc = res("service.installed", Warn, "not installed: go-build-it and its herdr session will not start at login", nil)
			svc.Fix = human("Run `gbi service install` so go-build-it and its herdr session start at login.")
		}
	}
	svc.Ms = since(t0)
	out = append(out, svc)
	if svcInstalled {
		b, _ := os.ReadFile(plan.File)
		if spath, ok := servicePATH(string(b)); ok {
			for _, p := range probes {
				if !p.ok {
					continue
				}
				r := res("harness.on_service_path", OK, "", p)
				r.Evidence = &Evidence{Command: "PATH from " + plan.File, Output: spath}
				if found, ok := findIn(spath, p.bin); ok {
					r.Detail = found
				} else {
					// A warning: herdr panes may still find it through the login shell; the live tier decides.
					r.Status, r.Detail = Warn, p.bin+" is not on the PATH the login service starts go-build-it with"
					r.Fix = auto("service-reinstall", "Reinstall the login service from a shell where "+p.bin+" works: run `gbi service install` (gbi doctor --fix does this).")
				}
				out = append(out, r)
			}
		}
	}
	out = append(out, d.tailscaleServe(ctx))
	return out
}

// tailscaleServe checks remote access (FR-20d). Not using Tailscale is fine: it is skipped.
func (d *Doctor) tailscaleServe(ctx context.Context) Result {
	t0 := time.Now()
	env := tailscale.System
	if d.Tailscale != nil {
		env = *d.Tailscale
	}
	s := env.Detect(ctx, d.Cfg.Listen, d.Cfg.Dir())
	var r Result
	switch {
	case !s.LoggedIn || !s.Serving:
		r = res("tailscale.serve", Skipped, "not used: "+firstNonEmpty(s.Guidance, "go-build-it is not served on a tailnet"), nil)
	case !s.Private:
		r = res("tailscale.serve", Fail, "Funnel is on for "+s.ServeURL+": anyone on the internet can reach go-build-it", nil)
		r.Fix = human("Turn Funnel off for go-build-it: run `" + checkFix(s, "private") + "`. go-build-it trusts Tailscale identity and must stay on your tailnet.")
	case !s.Reachable:
		r = res("tailscale.serve", Warn, checkDetail(s, "reachable"), nil)
		r.Fix = human("Open Settings → Remote access, or run `gbi tailscale status`, and follow the failing check. If go-build-it is not running, start it with `gbi service install`.")
	default:
		r = res("tailscale.serve", OK, s.ServeURL+" answers, tailnet only", nil)
	}
	r.Evidence = &Evidence{Command: "tailscale serve status --json"}
	for _, c := range s.Checks {
		r.Evidence.Output += fmt.Sprintf("%s: %s %s\n", c.ID, c.Status, c.Detail)
	}
	r.Ms = since(t0)
	return r
}

func checkDetail(s tailscale.Status, id string) string {
	for _, c := range s.Checks {
		if c.ID == id {
			return c.Detail
		}
	}
	return ""
}

func checkFix(s tailscale.Status, id string) string {
	for _, c := range s.Checks {
		if c.ID == id {
			return c.Fix
		}
	}
	return ""
}

func (d *Doctor) servicePlan() (service.Plan, error) {
	if d.Service != nil {
		return d.Service()
	}
	return service.Build(d.Exe, d.Cfg.Dir())
}

var (
	plistPath   = regexp.MustCompile(`<key>PATH</key>\s*<string>([^<]*)</string>`)
	systemdPath = regexp.MustCompile(`Environment="PATH=([^"]*)"`)
)

// servicePATH extracts the PATH a launchd plist or systemd unit sets.
func servicePATH(content string) (string, bool) {
	if m := plistPath.FindStringSubmatch(content); m != nil {
		return strings.NewReplacer("&amp;", "&", "&lt;", "<", "&gt;", ">").Replace(m[1]), true
	}
	if m := systemdPath.FindStringSubmatch(content); m != nil {
		return strings.ReplaceAll(m[1], "%%", "%"), true
	}
	return "", false
}

func lookPath(bin string) (string, error) {
	if filepath.IsAbs(bin) {
		if st, err := os.Stat(bin); err != nil || st.IsDir() {
			return "", fmt.Errorf("%s not found", bin)
		}
		return bin, nil
	}
	return exec.LookPath(bin)
}

func findIn(pathList, bin string) (string, bool) {
	if filepath.IsAbs(bin) {
		p, err := lookPath(bin)
		return p, err == nil
	}
	exts := []string{""}
	if runtime.GOOS == "windows" {
		exts = []string{".exe", ".cmd", ".bat", ""}
	}
	for _, dir := range filepath.SplitList(pathList) {
		for _, e := range exts {
			f := filepath.Join(dir, bin+e)
			if st, err := os.Stat(f); err == nil && !st.IsDir() && (runtime.GOOS == "windows" || st.Mode()&0o111 != 0) {
				return f, true
			}
		}
	}
	return "", false
}

func exitOf(err error) int {
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		return ee.ExitCode()
	}
	if err != nil {
		return -1
	}
	return 0
}

// ---- live ----

func token() string {
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func (d *Doctor) live(ctx context.Context, probes []*probe) (string, []Result) {
	var out []Result
	t0 := time.Now()
	ectx, cancel := context.WithTimeout(ctx, 20*time.Second)
	st, err := d.Sup.Ensure(ectx)
	cancel()
	sr := res("session.running", OK, fmt.Sprintf("herdr --session %s · %s", d.Sup.C.Session, d.Sup.C.Socket), nil)
	sr.Evidence = &Evidence{Command: fmt.Sprintf("herdr --session %s server", d.Sup.C.Session)}
	if err != nil {
		sr.Status, sr.Detail = Fail, "the gbi herdr session did not start: "+err.Error()
		sr.Evidence.Output = err.Error()
		sr.Fix = human(fmt.Sprintf("Run `herdr --session %s server` in a terminal to see why herdr does not start, fix what it reports, then run gbi doctor again.", d.Sup.C.Session))
		sr.Ms = since(t0)
		return "", append(out, sr)
	}
	if st.Adopted {
		sr.Detail += " · already running"
	} else {
		sr.Detail += " · started now"
	}
	sr.Ms = since(t0)
	out = append(out, sr)

	// Integrations (warn only: members still run without them).
	if ints, err := d.Sup.C.Integrations(ctx); err == nil {
		by := map[string]herdr.IntegrationInfo{}
		for _, i := range ints {
			by[string(i.Target)] = i
		}
		for _, p := range probes {
			if p.integration == "" || !p.ok {
				continue
			}
			i, found := by[p.integration]
			r := res("integration.current", OK, "current", p)
			switch {
			case !found:
				r.Status, r.Detail = Warn, "herdr has no integration for "+p.integration
			case i.State == herdr.IntegrationStateCurrent:
			default:
				r.Status, r.Detail = Warn, string(i.State)
				r.Fix = auto("install-integration:"+p.integration, "Install herdr's "+p.label+" integration (gbi doctor --fix does this) so a member's conversation survives herdr restarts.")
			}
			r.Evidence = &Evidence{Command: "herdr integration status"}
			out = append(out, r)
		}
	}

	// A throwaway workspace: run a command and read it back.
	t0 = time.Now()
	ws, pane, err := d.workspace(ctx)
	rt := res("pane.roundtrip", OK, "", nil)
	if err != nil {
		rt.Status, rt.Detail = Fail, "could not open a pane: "+err.Error()
		rt.Evidence = &Evidence{Command: "workspace.create " + WorkspaceLabel, Output: err.Error()}
		rt.Fix = human("Open `herdr --session gbi` in a terminal and check that it can open a new tab; update herdr if it cannot.")
		rt.Ms = since(t0)
		return ws, append(out, rt)
	}
	tok := token()
	cmd := echoCommand("gbi-doctor-ok", tok)
	text, ok := d.runInPane(ctx, pane, cmd, regexp.MustCompile(`(?m)^gbi-doctor-ok-`+tok+`\s*$`), 15*time.Second)
	rt.Evidence = &Evidence{Command: strings.TrimSpace(cmd)}
	if !ok {
		rt.Status, rt.Detail = Fail, "the command's output never came back from the pane"
		rt.Evidence.Output = tail(text, 20)
		rt.Fix = human("Open `herdr --session gbi`, workspace gbi-doctor, and look at the pane: the shell may be waiting for input or failing to start. Fix your shell's startup files, then run gbi doctor again.")
		rt.Ms = since(t0)
		return ws, append(out, rt)
	}
	rt.Detail = fmt.Sprintf("workspace %s · ran a command and read its output back", WorkspaceLabel)
	rt.Ms = since(t0)
	out = append(out, rt)

	// Harness binaries as a herdr pane sees them.
	var eligible []*probe
	for _, p := range probes {
		if p.ok {
			eligible = append(eligible, p)
		}
	}
	if len(eligible) == 0 {
		return ws, out
	}
	t0 = time.Now()
	tok = token()
	wcmd := whichCommand(eligible, tok)
	text, _ = d.runInPane(ctx, pane, wcmd, regexp.MustCompile(`(?m)^gbi-which-done-`+tok+`\s*$`), 15*time.Second)
	found := map[string]string{}
	for _, m := range regexp.MustCompile(`(?m)^gbi-which-([a-z0-9]+)=(.+?)\s*$`).FindAllStringSubmatch(text, -1) {
		found[m[1]] = m[2]
	}
	for _, p := range eligible {
		r := res("harness.on_pane_path", OK, "", p)
		r.Evidence = &Evidence{Command: strings.TrimSpace(wcmd), Output: "gbi-which-" + p.name + "=" + firstNonEmpty(found[p.name], "(no answer)")}
		v := found[p.name]
		switch {
		case v != "" && v != "missing":
			r.Detail = v
		default:
			p.ok = false
			r.Status, r.Detail = Fail, p.bin+" is not on the PATH inside herdr panes"
			if abs, err := lookPath(p.bin); err == nil && filepath.IsAbs(abs) {
				r.Fix = auto("set-binary:"+p.name+"="+abs, fmt.Sprintf("Use the full path for %s: set harnesses.%s.binary = %q in config.toml (gbi doctor --fix does this), or add %s to the PATH your login shell sets.", p.label, p.name, abs, filepath.Dir(abs)))
			} else {
				r.Fix = human(fmt.Sprintf("Make %s available to herdr panes: add its folder to the PATH your login shell sets (for example in ~/.zprofile or ~/.profile), or set harnesses.%s.binary to its full path in config.toml. Then run gbi doctor again.", p.bin, p.name))
			}
		}
		r.Ms = since(t0)
		out = append(out, r)
	}
	return ws, out
}

// workspace finds or creates the doctor workspace and returns it with its root pane.
func (d *Doctor) workspace(ctx context.Context) (string, string, error) {
	dir := filepath.Join(d.Cfg.Dir(), "doctor")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", "", err
	}
	if snap, err := d.Sup.C.Snapshot(ctx); err == nil {
		for _, w := range snap.Workspaces {
			if w.Label == WorkspaceLabel {
				c, cancel := context.WithTimeout(ctx, 5*time.Second)
				_ = d.Sup.C.Call(c, herdr.MethodWorkspaceClose, map[string]any{"workspace_id": w.WorkspaceID}, nil)
				cancel()
			}
		}
	}
	cr, err := d.Sup.C.CreateWorkspace(ctx, dir, WorkspaceLabel)
	if err != nil {
		return "", "", err
	}
	return cr.Workspace.WorkspaceID, cr.RootPane.PaneID, nil
}

// runInPane types cmd and reads the pane until done matches or the time is up.
func (d *Doctor) runInPane(ctx context.Context, pane, cmd string, done *regexp.Regexp, limit time.Duration) (string, bool) {
	if err := d.Sup.C.Call(ctx, herdr.MethodPaneSendText, map[string]any{"pane_id": pane, "text": cmd}, nil); err != nil {
		return err.Error(), false
	}
	deadline := time.Now().Add(limit)
	text := ""
	for time.Now().Before(deadline) {
		if t, err := d.Sup.C.Read(ctx, pane, "recent_unwrapped", 200); err == nil {
			text = t
			if done.MatchString(t) {
				return t, true
			}
		}
		select {
		case <-ctx.Done():
			return text, false
		case <-time.After(150 * time.Millisecond):
		}
	}
	return text, false
}

// echoCommand prints <prefix>-<tok> without that text appearing in the typed line.
func echoCommand(prefix, tok string) string {
	if runtime.GOOS == "windows" {
		return fmt.Sprintf("Write-Output ('%s' + '-%s')\r\n", prefix, tok)
	}
	return fmt.Sprintf("printf '%%s-%%s\\n' %s %s\n", prefix, tok)
}

func whichCommand(ps []*probe, tok string) string {
	if runtime.GOOS == "windows" {
		var b strings.Builder
		for _, p := range ps {
			fmt.Fprintf(&b, "$c = Get-Command '%s' -ErrorAction SilentlyContinue; if ($c) { 'gbi-which-%s' + '=' + $c.Source } else { 'gbi-which-%s' + '=missing' }; ", strings.ReplaceAll(p.bin, "'", "''"), p.name, p.name)
		}
		return b.String() + fmt.Sprintf("'gbi-which-done' + '-%s'\r\n", tok)
	}
	var b strings.Builder
	for _, p := range ps {
		q := "'" + strings.ReplaceAll(p.bin, "'", `'\''`) + "'"
		fmt.Fprintf(&b, "printf '%%s=%%s\\n' gbi-which-%s \"$(command -v %s || echo missing)\"; ", p.name, q)
	}
	return b.String() + fmt.Sprintf("printf '%%s-%%s\\n' gbi-which-done %s\n", tok)
}

func tail(s string, n int) string {
	lines := strings.Split(strings.TrimRight(s, "\n "), "\n")
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return strings.Join(lines, "\n")
}

// ---- harness ----

func (d *Doctor) harnessTier(ctx context.Context, ws string, probes []*probe, keep bool) []Result {
	var out []Result
	limit := d.StartTimeout
	if limit == 0 {
		limit = 45 * time.Second
	}
	for _, p := range probes {
		if !p.ok {
			continue
		}
		t0 := time.Now()
		r := res("harness.starts", OK, "", p)
		dir := filepath.Join(d.Cfg.Dir(), "helpers", p.name)
		_ = os.MkdirAll(dir, 0o755)
		pane, err := d.Sup.C.NewTab(ctx, ws, dir, "doctor "+p.name, nil)
		if err != nil {
			r.Status, r.Detail = Fail, "could not open a pane: "+err.Error()
			r.Fix = human("Open `herdr --session gbi` and check that it can open a new tab; update herdr if it cannot.")
			p.ok = false
			out = append(out, r)
			continue
		}
		p.pane = pane
		hc := d.Cfg.Harnesses[p.name]
		args := p.prof.Args(hc.Model, hc.Flags, p.prof.NewConversationID(), false)
		name := herdr.SafeName("doctor", p.name)
		var startErr error
		if p.kind != "" {
			r.Evidence = &Evidence{Command: fmt.Sprintf("herdr agent start %s --kind %s -- %s", name, p.kind, strings.Join(args, " "))}
			for i := 0; i < 30; i++ {
				_, startErr = d.Sup.C.StartAgent(ctx, name, p.kind, pane, args, limit)
				if !herdr.IsCode(startErr, "agent_pane_busy") {
					break
				}
				time.Sleep(500 * time.Millisecond)
			}
		} else {
			cmd := sessions.ShellCommand(map[string]string{"GBI_FAKE_TRANSCRIPTS": harness.FakeTranscriptDir()}, p.bin, args)
			r.Evidence = &Evidence{Command: strings.TrimSpace(cmd)}
			startErr = d.Sup.C.Call(ctx, herdr.MethodPaneSendText, map[string]any{"pane_id": pane, "text": cmd}, nil)
		}
		state, waitErr := d.waitReady(ctx, pane, limit)
		trusted := ""
		if herdr.IsCode(startErr, "agent_not_ready") || state == herdr.AgentStatusBlocked {
			// go-build-it answers a folder-trust screen in its own folders (FR-07f), as it does for members.
			if tdir, ok := sessions.AnswerTrust(ctx, d.Sup.C, d.Cfg, pane); ok {
				trusted = tdir
				startErr = nil
				state, waitErr = d.waitReady(ctx, pane, limit)
			}
		}
		screen, _ := d.Sup.C.Read(context.Background(), pane, "visible", 20)
		_, trustScreen := sessions.TrustKeys(screen)
		switch {
		case (herdr.IsCode(startErr, "agent_not_ready") || state == herdr.AgentStatusBlocked) && trustScreen:
			r.Status, r.Detail = Fail, p.label+" is asking you to trust its folder, and go-build-it is set not to answer it"
			r.Fix = human(fmt.Sprintf("Turn on **Settings → Agent harnesses → Answer folder trust prompts** (`[agents] trust_folders = true`), or run `%s` once in a terminal in %s and choose \"Yes, I trust this folder\". Then run `gbi doctor --harness %s`.", p.bin, dir, p.name))
		case herdr.IsCode(startErr, "agent_not_ready") || state == herdr.AgentStatusBlocked:
			r.Status, r.Detail = Fail, p.label+" started but stopped at a screen that needs you (herdr state blocked)"
			r.Fix = human(fmt.Sprintf("Run `%s` once in a terminal in %s and finish what it asks (for example signing in), then run `gbi doctor --harness %s`. An AI agent cannot do this for you.", p.bin, dir, p.name))
		case startErr != nil:
			r.Status, r.Detail = Fail, "herdr could not start "+p.label+": "+startErr.Error()
			r.Fix = human(fmt.Sprintf("Run `%s` in a terminal to check that it starts, then run `gbi doctor --harness %s`.", p.bin, p.name))
		case waitErr != nil:
			r.Status, r.Detail = Fail, p.label+" "+waitErr.Error()
			r.Fix = human(fmt.Sprintf("Open `herdr --session gbi`, workspace %s, to see %s's screen; run `%s` in a terminal to check that it starts, then run `gbi doctor --harness %s`.", WorkspaceLabel, p.label, p.bin, p.name))
		default:
			r.Detail = fmt.Sprintf("herdr reports it %s", state)
			if trusted != "" {
				r.Detail += "; go-build-it answered its folder trust prompt for " + trusted
			}
		}
		r.Evidence.Exit = intp(map[bool]int{true: 0, false: 1}[r.Status == OK])
		if r.Status != OK {
			r.Evidence.Output = tail(screen, 20)
			p.ok = false
		}
		r.Ms = since(t0)
		out = append(out, r)
		if !keep || !p.ok {
			_ = d.Sup.C.Call(context.Background(), herdr.MethodPaneClose, map[string]any{"pane_id": pane}, nil)
			p.pane = ""
		}
	}
	return out
}

// waitReady waits until herdr reports an agent in the pane as idle or done (ready) or blocked.
func (d *Doctor) waitReady(ctx context.Context, pane string, limit time.Duration) (herdr.AgentStatus, error) {
	deadline := time.Now().Add(limit)
	blockedSince := time.Time{}
	for time.Now().Before(deadline) {
		a, err := d.Sup.C.Agent(ctx, pane)
		if err != nil {
			if _, perr := d.Sup.C.Pane(ctx, pane); perr != nil && ctx.Err() == nil {
				return "", errors.New("exited: its pane closed")
			}
		}
		if err == nil && a.Agent != nil {
			switch a.AgentStatus {
			case herdr.AgentStatusIdle, herdr.AgentStatusDone:
				return a.AgentStatus, nil
			case herdr.AgentStatusBlocked:
				// Give a harness a moment: some flash a blocked state while starting.
				if blockedSince.IsZero() {
					blockedSince = time.Now()
				} else if time.Since(blockedSince) > 1500*time.Millisecond {
					return a.AgentStatus, nil
				}
			default:
				blockedSince = time.Time{}
			}
		}
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(200 * time.Millisecond):
		}
	}
	return "", fmt.Errorf("did not become ready within %s", limit)
}

// ---- prompt ----

var readyLine = regexp.MustCompile(`(?m)^.*\bREADY\b.*$`)

func (d *Doctor) promptTier(ctx context.Context, probes []*probe) []Result {
	var out []Result
	for _, p := range probes {
		if !p.ok || p.pane == "" {
			continue
		}
		t0 := time.Now()
		r := res("harness.answers", OK, "", p)
		r.Evidence = &Evidence{Command: fmt.Sprintf("prompt %q", PromptText)}
		name := herdr.SafeName("doctor", p.name)
		seq, err := d.Sup.C.Submit(ctx, p.pane, name, p.kind != "", PromptText)
		var a herdr.AgentInfo
		if err == nil {
			a, err = d.Sup.C.WaitSettled(ctx, p.pane, seq, 90*time.Second)
		}
		text, _ := d.Sup.C.Read(context.Background(), p.pane, "recent_unwrapped", 80)
		answered := false
		for _, l := range readyLine.FindAllString(text, -1) {
			if !strings.Contains(l, "single word") {
				answered = true
			}
		}
		switch {
		case err != nil:
			r.Status, r.Detail = Fail, "no answer: "+err.Error()
		case a.AgentStatus == herdr.AgentStatusBlocked:
			r.Status, r.Detail = Fail, p.label+" stopped at a screen that needs you instead of answering"
		case !answered:
			r.Status, r.Detail = Fail, p.label+" settled without answering READY"
		default:
			r.Detail = "answered READY"
		}
		if r.Status != OK {
			r.Evidence.Output = tail(text, 20)
			r.Fix = human(fmt.Sprintf("Run `%s` in a terminal and send it a short message: check that you are signed in and have quota left. Then run `gbi doctor --harness %s --prompt`.", p.bin, p.name))
		}
		r.Ms = since(t0)
		out = append(out, r)
		_ = d.Sup.C.Call(context.Background(), herdr.MethodPaneClose, map[string]any{"pane_id": p.pane}, nil)
		p.pane = ""
	}
	return out
}
