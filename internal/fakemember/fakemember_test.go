package fakemember

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/techmuch/go-build-it/internal/herdr"
)

// The test binary doubles as the fake agent when GBI_RUN_FAKE_MEMBER=1.
func TestMain(m *testing.M) {
	if os.Getenv("GBI_RUN_FAKE_MEMBER") == "1" {
		if err := Run(os.Args[1:], os.Stdin, os.Stdout); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		os.Exit(0)
	}
	os.Exit(m.Run())
}

// StartInPane launches the fake member in a herdr pane and waits until herdr sees it idle.
func startInPane(t *testing.T, c *herdr.Client, pane, tx string) {
	t.Helper()
	exe, _ := os.Executable()
	cmd := fmt.Sprintf("GBI_RUN_FAKE_MEMBER=1 GBI_FAKE_TRANSCRIPTS=%q %q --session-id fake-test1\n", tx, exe)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := c.Call(ctx, herdr.MethodPaneSendText, map[string]any{"pane_id": pane, "text": cmd}, nil); err != nil {
		t.Fatal(err)
	}
	for ctx.Err() == nil {
		if a, err := c.Agent(ctx, pane); err == nil && a.Agent != nil && *a.Agent == "fake" && a.AgentStatus == herdr.AgentStatusIdle {
			return
		}
		time.Sleep(200 * time.Millisecond)
	}
	out, _ := c.Read(context.Background(), pane, "visible", 0)
	t.Fatalf("fake member never reported idle; screen:\n%s", out)
}

func TestFakeMemberInHerdr(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("pane shell syntax differs on Windows (P15-04)")
	}
	s := herdr.TestSession(t)
	c := s.C
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	dir, tx := t.TempDir(), t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".gbi-task.md"), []byte("# Task: Add work log\n\nDo it.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cr, err := c.CreateWorkspace(ctx, dir, "demo")
	if err != nil {
		t.Fatal(err)
	}
	pane := cr.RootPane.PaneID
	startInPane(t, c, pane, tx)
	// herdr keeps session IDs only from its official integrations, so go-build-it chooses
	// the conversation ID at launch (--session-id) rather than reading it back.
	if a, err := c.Agent(ctx, pane); err != nil || a.Agent == nil || *a.Agent != "fake" {
		t.Fatalf("agent: %+v %v", a, err)
	}
	prompt := "Read .gbi-task.md in this folder and do what it says."
	seq, err := c.Submit(ctx, pane, "", false, prompt)
	if err != nil {
		t.Fatal(err)
	}
	done, err := c.WaitSettled(ctx, pane, seq, 10*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if done.AgentStatus == herdr.AgentStatusBlocked {
		t.Fatalf("unexpected blocked: %+v", done)
	}
	b, err := os.ReadFile(filepath.Join(dir, "FAKE_WORK.md"))
	if err != nil || !strings.Contains(string(b), "Add work log") {
		t.Fatalf("work not done: %q %v", b, err)
	}
	tr, err := os.ReadFile(filepath.Join(tx, "fake-test1.jsonl"))
	if err != nil || !strings.Contains(string(tr), `"cache_read_input_tokens"`) || !strings.Contains(string(tr), `"Edit"`) {
		t.Fatalf("transcript: %v\n%s", err, tr)
	}

	// FAKE:block reports blocked until the human answers in the terminal.
	if err := os.WriteFile(filepath.Join(dir, ".gbi-task.md"), []byte("# Task: Guarded\nFAKE:block\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	seq, err = c.Submit(ctx, pane, "", false, prompt)
	if err != nil {
		t.Fatal(err)
	}
	got, err := c.WaitSettled(ctx, pane, seq, 10*time.Second)
	if err != nil || got.AgentStatus != herdr.AgentStatusBlocked {
		t.Fatalf("want blocked, got %+v %v", got, err)
	}
	seq = got.StateChangeSeq
	if err := c.Call(ctx, herdr.MethodPaneSendText, map[string]any{"pane_id": cr.RootPane.PaneID, "text": "y\n"}, nil); err != nil {
		t.Fatal(err)
	}
	if fin, err := c.WaitSettled(ctx, pane, seq, 10*time.Second); err != nil || fin.AgentStatus == herdr.AgentStatusBlocked {
		t.Fatalf("after answer: %+v %v", fin, err)
	}
}
