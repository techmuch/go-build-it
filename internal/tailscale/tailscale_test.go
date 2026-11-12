package tailscale

import (
	"context"
	"errors"
	"strings"
	"testing"
)

const dns = "box.tail1.ts.net"

// fake is a machine with a Serve configuration; calls records the tailscale commands run.
type fake struct {
	serve  string
	answer string // what https://<url>/api/system returns
	calls  []string
}

func (f *fake) env() Env {
	return Env{
		Look: func() bool { return true },
		Run: func(_ context.Context, args ...string) (string, error) {
			f.calls = append(f.calls, strings.Join(args, " "))
			switch strings.Join(args, " ") {
			case "status --json":
				return `{"BackendState":"Running","Self":{"DNSName":"` + dns + `."}}`, nil
			case "serve status --json":
				return f.serve, nil
			}
			return "", nil
		},
		Fetch: func(_ context.Context, u string) (int, []byte, error) {
			if f.answer == "" {
				return 0, nil, errors.New("connection refused")
			}
			return 200, []byte(f.answer), nil
		},
		Process: func(port int) string { return map[int]string{7420: "gbi", 3000: "node", 8080: "Python"}[port] },
	}
}

const others = `"3100":{"HTTP":true},"8080":{"TCPForward":"127.0.0.1:8080"}`

func serve(tcp, web, funnel string) string {
	return `{"TCP":{` + tcp + `},"Web":{` + web + `},"AllowFunnel":{` + funnel + `}}`
}

// FR-20a: the rule is found by its exact target, the round trip proves it is this
// go-build-it, other services are listed with their programs, and a QR code is made.
func TestServedAndReachable(t *testing.T) {
	f := &fake{answer: `{"version":"2026.11.12.1","dataDir":"/home/d/.go-build-it"}`,
		serve: serve(`"443":{"HTTPS":true},`+others, `"`+dns+`:443":{"Handlers":{"/":{"Proxy":"http://127.0.0.1:7420"}}}`, "")}
	s := f.env().Detect(context.Background(), "127.0.0.1:7420", "/home/d/.go-build-it")
	if !s.Serving || !s.Reachable || !s.Private || s.ServeURL != "https://"+dns {
		t.Fatalf("status: %+v", s)
	}
	if len(s.Rules) != 2 || !s.Rules[0].Mine || s.Rules[0].Process != "gbi" || s.Rules[1].Kind != "tcp" || s.Rules[1].Process != "Python" {
		t.Fatalf("rules: %+v", s.Rules)
	}
	for _, c := range s.Checks {
		if c.Status != "ok" {
			t.Errorf("check %s: %s %s", c.ID, c.Status, c.Detail)
		}
	}
	if len(s.QR) < 21 || len(s.QR[0]) != len(s.QR) || !strings.Contains(Terminal(s.QR), "█") {
		t.Fatalf("qr: %d rows", len(s.QR))
	}
}

// A rule that mentions 7420 somewhere else is not go-build-it (the old check was a text search).
func TestPortInTextIsNotARule(t *testing.T) {
	f := &fake{serve: serve(`"17420":{"HTTPS":true}`, `"`+dns+`:17420":{"Handlers":{"/":{"Proxy":"http://127.0.0.1:17420"}}}`, "")}
	if s := f.env().Detect(context.Background(), "127.0.0.1:7420", ""); s.Serving {
		t.Fatalf("17420 was taken for go-build-it: %+v", s.Rules)
	}
}

// Another copy of go-build-it, or another program, answering is not reachable; Funnel is a failure.
func TestWrongAnswerAndFunnel(t *testing.T) {
	f := &fake{answer: `{"version":"x","dataDir":"/other/.go-build-it"}`,
		serve: serve(`"443":{"HTTPS":true}`, `"`+dns+`:443":{"Handlers":{"/":{"Proxy":"http://localhost:7420"}}}`, `"`+dns+`:443":true`)}
	s := f.env().Detect(context.Background(), "127.0.0.1:7420", "/home/d/.go-build-it")
	if !s.Serving || s.Reachable || s.Private || len(s.QR) != 0 {
		t.Fatalf("status: %+v", s)
	}
	got := map[string]Check{}
	for _, c := range s.Checks {
		got[c.ID] = c
	}
	if got["reachable"].Status != "fail" || !strings.Contains(got["reachable"].Detail, "different go-build-it") {
		t.Errorf("reachable: %+v", got["reachable"])
	}
	if got["private"].Status != "fail" || got["private"].Fix != "tailscale funnel --https=443 off" {
		t.Errorf("private: %+v", got["private"])
	}
}

// FR-20b: with 443 free, setup serves there without asking.
func TestSetupOnFreeDefault(t *testing.T) {
	f := &fake{serve: serve(others, "", "")}
	s := f.env().Detect(context.Background(), "127.0.0.1:7420", "")
	if s.Proposal == nil || s.Proposal.Port != 443 || s.Proposal.NeedsConfirm {
		t.Fatalf("proposal: %+v", s.Proposal)
	}
	if _, err := f.env().Setup(context.Background(), "127.0.0.1:7420", "", 0, false); err != nil {
		t.Fatal(err)
	}
	if !contains(f.calls, "serve --bg --https=443 http://127.0.0.1:7420") {
		t.Fatalf("calls: %v", f.calls)
	}
}

// FR-20b: when another service owns 443, go-build-it never replaces it: it proposes its own
// port, names the owner, and serves there only after confirmation.
func TestSetupNeverTakesAnotherServicesAddress(t *testing.T) {
	f := &fake{serve: serve(`"443":{"HTTPS":true},"7420":{"HTTPS":true}`,
		`"`+dns+`:443":{"Handlers":{"/":{"Proxy":"http://127.0.0.1:3000"}}},"`+dns+`:7420":{"Handlers":{"/":{"Proxy":"http://127.0.0.1:9999"}}}`, "")}
	s := f.env().Detect(context.Background(), "0.0.0.0:7420", "")
	if len(s.Conflicts) != 1 || s.Conflicts[0].Process != "node" {
		t.Fatalf("conflicts: %+v", s.Conflicts)
	}
	p := s.Proposal
	if p == nil || !p.NeedsConfirm || p.Port != 8443 || !strings.Contains(p.Reason, "node (http://127.0.0.1:3000)") {
		t.Fatalf("proposal: %+v", p)
	}
	if _, err := f.env().Setup(context.Background(), "0.0.0.0:7420", "", 0, false); err == nil || !strings.Contains(err.Error(), "confirm") {
		t.Fatalf("setup without confirmation: %v", err)
	}
	if _, err := f.env().Setup(context.Background(), "0.0.0.0:7420", "", 443, true); err == nil {
		t.Fatal("asking for the taken port must fail")
	}
	for _, c := range f.calls {
		if strings.HasPrefix(c, "serve --bg") {
			t.Fatalf("served without confirmation: %v", f.calls)
		}
	}
	if _, err := f.env().Setup(context.Background(), "0.0.0.0:7420", "", 0, true); err != nil {
		t.Fatal(err)
	}
	if !contains(f.calls, "serve --bg --https=8443 http://127.0.0.1:7420") {
		t.Fatalf("calls: %v", f.calls)
	}
}

// Stop removes go-build-it's own rule and nothing else.
func TestStopRemovesOnlyItsOwnRule(t *testing.T) {
	f := &fake{serve: serve(`"443":{"HTTPS":true},"8443":{"HTTPS":true},`+others,
		`"`+dns+`:443":{"Handlers":{"/":{"Proxy":"http://127.0.0.1:3000"}}},"`+dns+`:8443":{"Handlers":{"/":{"Proxy":"http://127.0.0.1:7420"}}}`, "")}
	if _, err := f.env().Stop(context.Background(), "127.0.0.1:7420", ""); err != nil {
		t.Fatal(err)
	}
	var offs []string
	for _, c := range f.calls {
		if strings.HasSuffix(c, " off") {
			offs = append(offs, c)
		}
	}
	if len(offs) != 1 || offs[0] != "serve --https=8443 off" {
		t.Fatalf("off calls: %v", offs)
	}
}

func TestNotInstalled(t *testing.T) {
	e := (&fake{}).env()
	e.Look = func() bool { return false }
	s := e.Detect(context.Background(), "127.0.0.1:7420", "")
	if s.Installed || s.Checks[0].Status != "fail" || !strings.Contains(s.Guidance, "tailscale.com/download") {
		t.Fatalf("%+v", s)
	}
}

func contains(xs []string, s string) bool {
	for _, x := range xs {
		if x == s {
			return true
		}
	}
	return false
}
