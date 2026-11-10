package doctor

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/techmuch/go-build-it/internal/config"
	"github.com/techmuch/go-build-it/internal/fakemember"
	"github.com/techmuch/go-build-it/internal/herdr"
	"github.com/techmuch/go-build-it/internal/service"
)

// The test binary doubles as the scripted harness: herdr panes run `<exe> dev fake-member …`.
func TestMain(m *testing.M) {
	if len(os.Args) > 2 && os.Args[1] == "dev" && os.Args[2] == "fake-member" {
		if err := fakemember.Run(os.Args[3:], os.Stdin, os.Stdout); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		os.Exit(0)
	}
	os.Exit(m.Run())
}

func testConfig(t *testing.T, flags ...string) *config.Config {
	t.Helper()
	cfg, err := config.LoadFrom(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	exe, _ := os.Executable()
	cfg.Harnesses = map[string]config.Harness{"fake": {Enabled: true, Binary: exe, Flags: flags}}
	cfg.ExternalToken = "s3cret-token"
	return cfg
}

func noService() (service.Plan, error) { return service.Plan{}, nil }

func find(t *testing.T, rep Report, id string) Result {
	t.Helper()
	for _, r := range rep.Results {
		if r.ID == id {
			return r
		}
	}
	b, _ := json.MarshalIndent(rep, "", " ")
	t.Fatalf("no result %s in\n%s", id, b)
	return Result{}
}

func tierRun(rep Report, t Tier) TierRun {
	for _, x := range rep.Tiers {
		if x.Tier == t {
			return x
		}
	}
	return TierRun{}
}

func TestNoHerdrNeedsAHuman(t *testing.T) {
	cfg := testConfig(t)
	cfg.Harnesses["claude"] = config.Harness{Enabled: true, Binary: "gbi-test-no-such-harness"}
	d := &Doctor{Cfg: cfg, Sup: herdr.NewSupervisor(filepath.Join(t.TempDir(), "herdr-missing"), "gbi", "", nil), Version: "dev", Service: noService}
	rep := d.Run(context.Background(), Options{})
	r := find(t, rep, "herdr.installed")
	if r.Status != Fail || r.Fix == nil || r.Fix.Kind != "human" || !strings.Contains(r.Fix.Instructions, "Install herdr") {
		t.Fatalf("herdr.installed = %+v", r)
	}
	if rep.Exit != 2 {
		t.Fatalf("exit = %d, want 2", rep.Exit)
	}
	if lt := tierRun(rep, Live); lt.Ran || !strings.Contains(lt.Note, "static tier failed") {
		t.Fatalf("live tier = %+v", lt)
	}
	// A harness that is not installed is a warning with an automatic fix: disable it.
	c := find(t, rep, "harness.installed")
	for _, x := range rep.Results {
		if x.ID == "harness.installed" && x.Harness == "claude" {
			c = x
		}
	}
	if c.Status != Warn || c.Fix == nil || c.Fix.Action != "disable-harness:claude" {
		t.Fatalf("claude harness.installed = %+v", c)
	}
	applied, errs := d.FixAll(context.Background(), rep)
	if len(errs) > 0 || len(applied) != 1 {
		t.Fatalf("FixAll = %v %v", applied, errs)
	}
	again, err := config.LoadFrom(cfg.Dir())
	if err != nil || again.Harnesses["claude"].Enabled {
		t.Fatalf("claude still enabled after fix: %v", err)
	}
	for _, x := range rep.Results {
		if x.Docs != DocsURL(x.ID) {
			t.Fatalf("result %s has docs %q", x.ID, x.Docs)
		}
	}
	md := d.Markdown(rep)
	if strings.Contains(md, "s3cret-token") || !strings.Contains(md, "herdr.installed") || !strings.Contains(md, "Needs a human") {
		t.Fatalf("report:\n%s", md)
	}
}

func TestServicePATH(t *testing.T) {
	plist := `<key>EnvironmentVariables</key><dict><key>GBI_HOME</key><string>/x</string><key>PATH</key><string>/a&amp;b:/usr/bin</string></dict>`
	if p, ok := servicePATH(plist); !ok || p != "/a&b:/usr/bin" {
		t.Fatalf("plist PATH = %q %v", p, ok)
	}
	unit := "[Service]\nExecStart=/x serve\nEnvironment=\"PATH=/opt/100%%/bin:/usr/bin\"\n"
	if p, ok := servicePATH(unit); !ok || p != "/opt/100%/bin:/usr/bin" {
		t.Fatalf("unit PATH = %q %v", p, ok)
	}
	if runtime.GOOS == "windows" {
		return
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "tool"), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, ok := findIn("/nope"+string(os.PathListSeparator)+dir, "tool"); !ok {
		t.Fatal("tool not found in its folder")
	}
	if _, ok := findIn("/usr/bin", "tool"); ok {
		t.Fatal("tool found where it is not")
	}
}

// The service file's PATH decides harness.on_service_path; a stale one has an automatic fix.
func TestHarnessNotOnServicePath(t *testing.T) {
	cfg := testConfig(t)
	svc := filepath.Join(t.TempDir(), "gbi.plist")
	if err := os.WriteFile(svc, []byte(`<key>PATH</key><string>/usr/bin</string>`), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg.Harnesses["fake"] = config.Harness{Enabled: true, Binary: "sh"} // relative: looked up in the service PATH
	d := &Doctor{Cfg: cfg, Sup: herdr.NewSupervisor(filepath.Join(t.TempDir(), "herdr-missing"), "gbi", "", nil), Service: func() (service.Plan, error) { return service.Plan{File: svc}, nil }}
	rep := d.Run(context.Background(), Options{Through: Static})
	if runtime.GOOS == "windows" {
		return
	}
	r := find(t, rep, "harness.on_service_path")
	if r.Status != OK { // /usr/bin/sh or /bin/sh: on most systems /usr/bin has sh
		if _, err := os.Stat("/usr/bin/sh"); err == nil {
			t.Fatalf("sh not found on /usr/bin: %+v", r)
		}
	}
	cfg.Harnesses["fake"] = config.Harness{Enabled: true, Binary: "gbi-doctor-test-only-here"}
	bin := t.TempDir()
	if err := os.WriteFile(filepath.Join(bin, "gbi-doctor-test-only-here"), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	rep = d.Run(context.Background(), Options{Through: Static})
	r = find(t, rep, "harness.on_service_path")
	if r.Status != Warn || r.Fix == nil || r.Fix.Kind != "auto" || r.Fix.Action != "service-reinstall" {
		t.Fatalf("on_service_path = %+v", r)
	}
	if rep.Exit != 2 { // herdr is missing too (human), so 2
		t.Fatalf("exit = %d", rep.Exit)
	}
}

// Every tier against a real, isolated herdr with the scripted harness.
func TestEveryTierWithTheScriptedHarness(t *testing.T) {
	cfg := testConfig(t)
	s := herdr.TestSession(t)
	exe, _ := os.Executable()
	d := &Doctor{Cfg: cfg, Sup: s, Exe: exe, Version: "dev", Service: noService, StartTimeout: 30 * time.Second}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	rep := d.Run(ctx, Options{Prompt: true})
	b, _ := json.MarshalIndent(rep, "", " ")
	if rep.Exit != 0 {
		t.Fatalf("exit %d:\n%s", rep.Exit, b)
	}
	for _, id := range []string{"herdr.installed", "herdr.api", "session.running", "pane.roundtrip", "harness.on_pane_path", "harness.starts", "harness.answers"} {
		if r := find(t, rep, id); r.Status != OK {
			t.Fatalf("%s = %+v", id, r)
		}
	}
	for _, tr := range rep.Tiers {
		if !tr.Ran {
			t.Fatalf("tier %s did not run: %s", tr.Tier, tr.Note)
		}
	}
	// The throwaway workspace is gone afterwards.
	snap, err := s.C.Snapshot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, w := range snap.Workspaces {
		if w.Label == WorkspaceLabel {
			t.Fatalf("doctor workspace left open")
		}
	}
}

// A harness that stops at a sign-in screen fails the harness tier with screen evidence and a human fix.
func TestBlockedHarnessNeedsAHuman(t *testing.T) {
	cfg := testConfig(t, "--start-blocked")
	s := herdr.TestSession(t)
	exe, _ := os.Executable()
	d := &Doctor{Cfg: cfg, Sup: s, Exe: exe, Version: "dev", Service: noService, StartTimeout: 20 * time.Second}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	rep := d.Run(ctx, Options{Through: HarnessTier, Prompt: true})
	r := find(t, rep, "harness.starts")
	if r.Status != Fail || r.Fix == nil || r.Fix.Kind != "human" || r.Evidence == nil || !strings.Contains(r.Evidence.Output, "Select login method") {
		b, _ := json.MarshalIndent(r, "", " ")
		t.Fatalf("harness.starts:\n%s", b)
	}
	if rep.Exit != 2 {
		t.Fatalf("exit = %d", rep.Exit)
	}
	if pt := tierRun(rep, Prompt); pt.Ran {
		t.Fatal("prompt tier ran after the harness tier failed")
	}
}

// A folder trust screen in go-build-it's own folder is answered, so the harness starts (FR-07f).
// With [agents] trust_folders off it is a human fix that names the screen.
func TestTrustScreenAnswered(t *testing.T) {
	for _, on := range []bool{true, false} {
		cfg := testConfig(t, "--start-trust")
		cfg.Agents.TrustFolders = on
		s := herdr.TestSession(t)
		exe, _ := os.Executable()
		d := &Doctor{Cfg: cfg, Sup: s, Exe: exe, Version: "dev", Service: noService, StartTimeout: 20 * time.Second}
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		rep := d.Run(ctx, Options{Through: HarnessTier})
		cancel()
		r := find(t, rep, "harness.starts")
		b, _ := json.MarshalIndent(r, "", " ")
		if on && (r.Status != OK || !strings.Contains(r.Detail, "answered its folder trust prompt")) {
			t.Fatalf("trust on:\n%s", b)
		}
		if !on && (r.Status != Fail || r.Fix == nil || r.Fix.Kind != "human" || !strings.Contains(r.Detail, "trust its folder") || !strings.Contains(r.Fix.Instructions, "trust_folders")) {
			t.Fatalf("trust off:\n%s", b)
		}
	}
}

// A harness the doctor's PATH has but herdr panes do not: fail with a set-binary fix, which works.
func TestHarnessMissingInsidePanes(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("posix shell")
	}
	cfg := testConfig(t)
	s := herdr.TestSession(t) // started with the original PATH
	exe, _ := os.Executable()
	bin := t.TempDir()
	if err := os.Symlink(exe, filepath.Join(bin, "gbi-doctor-fake")); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	cfg.Harnesses["fake"] = config.Harness{Enabled: true, Binary: "gbi-doctor-fake"}
	d := &Doctor{Cfg: cfg, Sup: s, Exe: exe, Version: "dev", Service: noService}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	rep := d.Run(ctx, Options{})
	r := find(t, rep, "harness.on_pane_path")
	if r.Status != Fail || r.Fix == nil || r.Fix.Kind != "auto" || !strings.HasPrefix(r.Fix.Action, "set-binary:fake=") {
		b, _ := json.MarshalIndent(rep, "", " ")
		t.Fatalf("on_pane_path:\n%s", b)
	}
	if rep.Exit != 1 {
		t.Fatalf("exit = %d, want 1 (automatic fix)", rep.Exit)
	}
	if applied, errs := d.FixAll(ctx, rep); len(errs) > 0 || len(applied) == 0 {
		t.Fatalf("FixAll %v %v", applied, errs)
	}
	rep = d.Run(ctx, Options{})
	if r := find(t, rep, "harness.on_pane_path"); r.Status != OK {
		t.Fatalf("after fix: %+v", r)
	}
}

func TestEveryCheckHasATroubleshootingSection(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("..", "..", "docs", "user", "20-troubleshooting.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range Checks {
		if !strings.Contains(string(b), "### `"+c.ID+"`") {
			t.Errorf("docs/user/20-troubleshooting.md has no section ### `%s`", c.ID)
		}
	}
}
