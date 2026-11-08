// Package tailscale detects Tailscale and configures Tailscale Serve (FR-20).
package tailscale

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// Status describes the local Tailscale state.
type Status struct {
	Installed bool   `json:"installed"`
	LoggedIn  bool   `json:"loggedIn"`
	DNSName   string `json:"dnsName"`
	ServeURL  string `json:"serveUrl"`
	Serving   bool   `json:"serving"`
	Guidance  string `json:"guidance"`
}

func run(ctx context.Context, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "tailscale", args...).CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

// Detect reports Tailscale status and whether Serve proxies to listen (e.g. 127.0.0.1:7420).
func Detect(ctx context.Context, listen string) Status {
	var s Status
	if _, err := exec.LookPath("tailscale"); err != nil {
		s.Guidance = "Tailscale is not installed. Install it from https://tailscale.com/download, log in, then run setup again."
		return s
	}
	s.Installed = true
	out, err := run(ctx, "status", "--json")
	if err != nil {
		s.Guidance = "Tailscale is installed but not running. Start it and log in."
		return s
	}
	var st struct {
		BackendState string
		Self         struct{ DNSName string }
	}
	_ = json.Unmarshal([]byte(out), &st)
	s.LoggedIn = st.BackendState == "Running"
	s.DNSName = strings.TrimSuffix(st.Self.DNSName, ".")
	if !s.LoggedIn {
		s.Guidance = "Log in to Tailscale (`tailscale up`), then run setup again."
		return s
	}
	if out, err := run(ctx, "serve", "status", "--json"); err == nil && strings.Contains(out, listen[strings.LastIndex(listen, ":")+1:]) {
		s.Serving = true
		s.ServeURL = "https://" + s.DNSName
	}
	if !s.Serving {
		s.Guidance = "Tailscale is ready. Run setup to serve go-build-it at https://" + s.DNSName
	}
	return s
}

// Setup runs `tailscale serve --bg <listen>` (FR-20).
func Setup(ctx context.Context, listen string) (Status, error) {
	s := Detect(ctx, listen)
	if !s.LoggedIn {
		return s, fmt.Errorf("%s", s.Guidance)
	}
	if out, err := run(ctx, "serve", "--bg", "http://"+listen); err != nil {
		return s, fmt.Errorf("tailscale serve: %v: %s", err, out)
	}
	return Detect(ctx, listen), nil
}
