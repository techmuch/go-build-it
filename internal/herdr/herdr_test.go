package herdr

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSafeNameAndVersions(t *testing.T) {
	for in, want := range map[string]string{"InboxQL|Forge": "inboxql-forge", "go-build-it|atlas": "go-build-it-atlas", "9lives|x": "a9lives-x", "a very long project name here|member": "a-very-long-project-name-here-me"} {
		p := strings.Split(in, "|")
		if got := SafeName(p...); got != want {
			t.Errorf("SafeName(%v) = %q, want %q", p, got, want)
		}
	}
	if CompareVersions("0.9.5", "0.9.2") != 1 || CompareVersions("0.10.0", "0.9.9") != 1 || CompareVersions("v0.9.2", "0.9.2") != 0 || CompareVersions("0.8", "0.9.2") != -1 {
		t.Fatal("version compare")
	}
}

func TestGeneratedTypesMatchSchema(t *testing.T) {
	_, err := os.Stat(filepath.Join("schema", "herdr-api.schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(RequiredMethods) < 20 || SchemaProtocol == 0 {
		t.Fatal("generated constants missing")
	}
}

func TestLiveSession(t *testing.T) {
	dir := t.TempDir() // before the session, so cleanup stops herdr before removing it (Windows)
	s := TestSession(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	c := s.C
	st := s.Status()
	if !st.Running || st.Version == "" || !strings.Contains(c.Socket, filepath.Join("sessions", "gbi")) {
		t.Fatalf("status %+v socket %s", st, c.Socket)
	}
	if CompareVersions(st.Version, MinVersion) < 0 {
		t.Fatalf("herdr %s older than %s", st.Version, MinVersion)
	}
	// A second Ensure adopts the running session instead of starting another.
	if st2, err := s.Ensure(ctx); err != nil || !st2.Adopted {
		t.Fatalf("adopt: %+v %v", st2, err)
	}
	if miss, err := c.MissingMethods(ctx); err != nil || len(miss) > 0 {
		t.Fatalf("installed herdr lacks %v (%v)", miss, err)
	}
	cr, err := c.CreateWorkspace(ctx, dir, "demo")
	if err != nil {
		t.Fatal(err)
	}
	if cr.RootPane.PaneID == "" || cr.Workspace.Label != "demo" {
		t.Fatalf("created %+v", cr)
	}
	evs, errc, err := c.Subscribe(ctx, Sub{Type: "pane.agent_status_changed", PaneID: cr.RootPane.PaneID}, Sub{Type: "pane.exited"})
	if err != nil {
		t.Fatal(err)
	}
	if err := c.ReportMetadata(ctx, cr.RootPane.PaneID, Metadata{DisplayAgent: "Forge (developer)", Title: "T-1 demo", Tokens: map[string]string{"task": "T-1", "ctx": "12%"}}); err != nil {
		t.Fatal(err)
	}
	p, err := c.Pane(ctx, cr.RootPane.PaneID)
	if err != nil {
		t.Fatal(err)
	}
	if p.Title == nil || *p.Title != "T-1 demo" || p.Tokens["task"] != "T-1" {
		t.Fatalf("metadata not applied: %+v", p)
	}
	if err := c.ReportWorkspace(ctx, cr.Workspace.WorkspaceID, map[string]string{"gbi": "1 working"}); err != nil {
		t.Fatal(err)
	}
	// A custom agent reported on the pane produces a status event.
	if err := c.Call(ctx, MethodPaneReportAgent, map[string]any{"pane_id": cr.RootPane.PaneID, "source": "gbi-test", "agent": "fake", "state": "working"}, nil); err != nil {
		t.Fatal(err)
	}
	deadline := time.After(10 * time.Second)
	for got := false; !got; {
		select {
		case e := <-evs:
			got = e.Event == "pane.agent_status_changed" && strings.Contains(string(e.Data), `"working"`)
		case err := <-errc:
			t.Fatalf("subscription ended: %v", err)
		case <-deadline:
			t.Fatal("no working status event")
		}
	}
	snap, err := c.Snapshot(ctx)
	if err != nil || len(snap.Workspaces) == 0 {
		t.Fatalf("snapshot %v %+v", err, snap)
	}
	if _, err := c.Read(ctx, cr.RootPane.PaneID, "visible", 0); err != nil {
		t.Fatal(err)
	}
}

func TestWatcher(t *testing.T) {
	dir := t.TempDir()
	s := TestSession(t)
	c := s.C
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	cr, err := c.CreateWorkspace(ctx, dir, "w")
	if err != nil {
		t.Fatal(err)
	}
	pane := cr.RootPane.PaneID
	changes, gone := make(chan AgentInfo, 16), make(chan string, 4)
	w := NewWatcher(c, nil)
	w.OnChange = func(a AgentInfo) { changes <- a }
	w.OnGone = func(p string) { gone <- p }
	w.Watch(pane)
	go w.Run(ctx)
	waitFor := func(status AgentStatus) {
		t.Helper()
		for {
			select {
			case a := <-changes:
				if a.AgentStatus == status {
					return
				}
			case <-time.After(10 * time.Second):
				t.Fatalf("no %s change", status)
			}
		}
	}
	// Give the subscription a moment, then report states.
	time.Sleep(300 * time.Millisecond)
	for i, st := range []string{"working", "blocked", "idle"} {
		if err := c.ReportAgent(ctx, pane, "gbi-test", "fake", st, "", "", nil, int64(i+1)); err != nil {
			t.Fatal(err)
		}
		waitFor(AgentStatus(st))
	}
	if a, ok := w.State(pane); !ok || a.AgentStatus != AgentStatusIdle {
		t.Fatalf("state %+v %v", a, ok)
	}
	if err := c.Call(ctx, MethodPaneClose, map[string]any{"pane_id": pane}, nil); err != nil {
		t.Fatal(err)
	}
	select {
	case p := <-gone:
		if p != pane {
			t.Fatalf("gone %s", p)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("no gone event")
	}
	if w.Resyncs() == 0 {
		t.Fatal("never reconciled from a snapshot")
	}
}

// The CI pin is the one place a herdr version is installed; it must be the tested maximum.
func TestCIPinMatchesTestedMax(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("..", "..", ".github", "workflows", "ci.yml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), `HERDR_VERSION: "`+TestedMax+`"`) {
		t.Fatalf("ci.yml HERDR_VERSION does not equal herdr.TestedMax %s", TestedMax)
	}
}
