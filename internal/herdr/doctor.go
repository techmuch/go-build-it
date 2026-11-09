package herdr

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"sort"
	"strings"
	"time"
)

// Check is one doctor result (FR-07). Fix, when set, is an action the user can run
// from the UI or `gbi herdr fix <action>`.
type Check struct {
	Name   string `json:"name"`
	OK     bool   `json:"ok"`
	Warn   bool   `json:"warn,omitempty"` // works, but should be fixed
	Detail string `json:"detail"`
	Hint   string `json:"hint,omitempty"`
	Fix    string `json:"fix,omitempty"`
}

// IntegrationFor maps a go-build-it harness to herdr's integration target.
var IntegrationFor = map[string]string{"claude": "claude", "antigravity": "antigravity_cli", "opencode": "opencode", "pi": "pi"}

// InstallHint says how to install herdr on this platform.
func InstallHint() string {
	switch runtime.GOOS {
	case "darwin":
		return "brew install herdr"
	case "windows":
		return `powershell -ExecutionPolicy Bypass -c "irm https://herdr.dev/install.ps1 | iex"   (Windows support is beta)`
	default:
		return "curl -fsSL https://herdr.dev/install.sh | sh"
	}
}

// Doctor runs every check. harnesses are the enabled go-build-it harness names;
// serviceInstalled reports whether the login service is installed.
func (s *Supervisor) Doctor(ctx context.Context, harnesses []string, serviceInstalled bool) []Check {
	var out []Check
	path, err := exec.LookPath(s.C.Bin)
	if err != nil {
		return append(out, Check{Name: "herdr installed", Detail: "herdr was not found on PATH (" + s.C.Bin + ")", Hint: InstallHint()})
	}
	verOut, _ := exec.CommandContext(ctx, path, "--version").Output()
	ver := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(string(verOut)), "herdr"))
	c := Check{Name: "herdr installed", OK: true, Detail: path + " · " + ver}
	switch {
	case ver == "":
		c.Warn, c.Detail = true, path+" · version unknown"
	case CompareVersions(ver, MinVersion) < 0:
		c.OK, c.Hint = false, fmt.Sprintf("herdr %s is older than %s; update it (herdr update, or your package manager)", ver, MinVersion)
	case CompareVersions(ver, TestedMax) > 0:
		c.Warn, c.Hint = true, fmt.Sprintf("newer than the version go-build-it was tested with (%s); it should work, and the weekly upstream review will confirm", TestedMax)
	default:
		c.Detail += fmt.Sprintf(" (supported ≥ %s)", MinVersion)
	}
	out = append(out, c)

	st := s.Status()
	if st.CheckedAt.IsZero() || time.Since(st.CheckedAt) > 10*time.Second {
		pctx, cancel := context.WithTimeout(ctx, 2*time.Second)
		p, err := s.C.Ping(pctx)
		cancel()
		st = s.set(err == nil, p, st.Adopted, err)
	}
	sc := Check{Name: "gbi session running", OK: st.Running, Detail: fmt.Sprintf("herdr --session %s · %s", s.C.Session, s.C.Socket)}
	if st.Running {
		sc.Detail += " · supervised"
		if st.Restarts > 0 {
			sc.Detail += fmt.Sprintf(" · restarted %d×", st.Restarts)
		}
	} else {
		sc.Hint, sc.Fix = "go-build-it starts it automatically; start it now", "start-session"
		if st.Error != "" {
			sc.Detail += " · " + st.Error
		}
	}
	out = append(out, sc)

	if miss, err := s.C.MissingMethods(ctx); err != nil {
		out = append(out, Check{Name: "API compatible", Warn: true, OK: true, Detail: "could not read herdr api schema: " + err.Error()})
	} else if len(miss) > 0 {
		out = append(out, Check{Name: "API compatible", Detail: "missing methods: " + strings.Join(miss, ", "), Hint: "update herdr"})
	} else {
		out = append(out, Check{Name: "API compatible", OK: true, Detail: fmt.Sprintf("all %d required methods present", len(RequiredMethods))})
	}

	if st.Running {
		ints, err := s.C.Integrations(ctx)
		byTarget := map[string]IntegrationInfo{}
		for _, i := range ints {
			byTarget[string(i.Target)] = i
		}
		sort.Strings(harnesses)
		for _, h := range harnesses {
			target, ok := IntegrationFor[h]
			if !ok {
				continue
			}
			name := fmt.Sprintf("%s integration", h)
			if err != nil {
				out = append(out, Check{Name: name, Warn: true, OK: true, Detail: "could not list integrations: " + err.Error()})
				continue
			}
			i, found := byTarget[target]
			switch {
			case !found:
				out = append(out, Check{Name: name, Warn: true, OK: true, Detail: "herdr has no integration for " + target})
			case i.State == IntegrationStateCurrent:
				out = append(out, Check{Name: name, OK: true, Detail: "current"})
			case i.State == IntegrationStateOutdated:
				out = append(out, Check{Name: name, OK: true, Warn: true, Detail: "outdated", Hint: "needed so the conversation ID survives herdr restarts", Fix: "install-integration:" + target})
			default:
				out = append(out, Check{Name: name, OK: true, Warn: true, Detail: "not installed", Hint: "needed so the conversation ID survives herdr restarts", Fix: "install-integration:" + target})
			}
		}
	}
	svc := Check{Name: "Login service", OK: serviceInstalled, Detail: "starts go-build-it and its herdr session at login"}
	if !serviceInstalled {
		svc.Warn, svc.OK, svc.Hint = true, true, "gbi service install"
	}
	out = append(out, svc)
	return out
}

// Fix runs a doctor action.
func (s *Supervisor) Fix(ctx context.Context, action string) error {
	switch {
	case action == "start-session":
		_, err := s.Ensure(ctx)
		return err
	case strings.HasPrefix(action, "install-integration:"):
		target := strings.TrimPrefix(action, "install-integration:")
		for _, t := range IntegrationFor {
			if t == target {
				return s.C.InstallIntegration(ctx, target)
			}
		}
		return fmt.Errorf("unknown integration %q", target)
	}
	return fmt.Errorf("unknown fix %q", action)
}
