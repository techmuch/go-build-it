package sessions

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"github.com/techmuch/go-build-it/internal/harness"
	"github.com/techmuch/go-build-it/internal/herdr"
	"github.com/techmuch/go-build-it/internal/team"
)

var askSeq atomic.Int64

// HelpersWorkspace is the herdr workspace that hosts one-shot helper conversations.
const HelpersWorkspace = "gbi-helpers"

// Ask runs one turn as a persona in a fresh, short-lived herdr conversation and returns
// its answer (artifact and persona edits, Lens assessments, release notes). The prompt
// goes in a file and the answer comes back in a file, so nothing is read off the screen.
// Each helper harness works in a fixed folder (one first-run trust prompt per harness),
// and its pane is closed afterwards. FR-60: herdr is the only way a harness runs.
func (m *Manager) Ask(ctx context.Context, mem team.Member, prompt string) (string, error) {
	h, err := m.herdr()
	if err != nil {
		return "", err
	}
	prof, ok := harness.Get(mem.Harness)
	if !ok {
		return "", fmt.Errorf("unknown harness %q", mem.Harness)
	}
	dir := filepath.Join(m.cfg.Dir(), "helpers", mem.Harness)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	n := askSeq.Add(1)
	stamp := fmt.Sprintf("%d-%d", time.Now().UnixNano()%1e9, n)
	in, out := filepath.Join(dir, ".gbi-ask-"+stamp+".md"), filepath.Join(dir, ".gbi-answer-"+stamp+".md")
	defer os.Remove(in)
	defer os.Remove(out)
	brief := "You are " + mem.Name + ", the " + mem.Role + " on this team.\n" + strings.TrimSpace(mem.Instructions) + "\n\n---\n\n" + prompt +
		"\n\n---\nWrite your complete answer to the file " + filepath.Base(out) + " in this folder (create it), then reply DONE. Do not use other tools.\n"
	if err := os.WriteFile(in, []byte(brief), 0o644); err != nil {
		return "", err
	}
	ctx, cancel := context.WithTimeout(ctx, 6*time.Minute)
	defer cancel()
	ws, err := m.helpersWorkspace(ctx, h)
	if err != nil {
		return "", err
	}
	sp, err := h.NewTab(ctx, ws, dir, "ask "+fmt.Sprint(n), nil)
	if err != nil {
		return "", err
	}
	defer func() { _ = h.Call(context.Background(), herdr.MethodPaneClose, map[string]any{"pane_id": sp}, nil) }()
	hc := m.cfg.Harnesses[mem.Harness]
	args := prof.Args(firstNonEmpty(mem.Model, hc.Model), hc.Flags, prof.NewConversationID(), false)
	name := herdr.SafeName("ask", fmt.Sprint(n))
	if prof.Kind != "" {
		if _, err := h.StartAgent(ctx, name, prof.Kind, sp, args, 60*time.Second); err != nil {
			if herdr.IsCode(err, "agent_not_ready") {
				return "", fmt.Errorf("%s needs you once in herdr (workspace %s): answer its first-run prompt for %s, then try again", prof.Label, HelpersWorkspace, dir)
			}
			return "", err
		}
	} else {
		bin := firstNonEmpty(hc.Binary, m.GbiBin)
		cmd := fmt.Sprintf("GBI_FAKE_TRANSCRIPTS=%s %s %s\n", shq(harness.FakeTranscriptDir()), shq(bin), strings.Join(shqAll(args), " "))
		if err := h.Call(ctx, herdr.MethodPaneSendText, map[string]any{"pane_id": sp, "text": cmd}, nil); err != nil {
			return "", err
		}
		if err := waitAgent(ctx, h, sp, prof, 30*time.Second); err != nil {
			return "", err
		}
	}
	seq, err := h.Submit(ctx, sp, name, prof.Kind != "", "Read "+filepath.Base(in)+" in this folder and do what it says.")
	if err != nil {
		return "", err
	}
	a, err := h.WaitSettled(ctx, sp, seq, 45*time.Second)
	if err != nil {
		return "", fmt.Errorf("%s did not answer: %w", mem.Name, err)
	}
	if a.AgentStatus == herdr.AgentStatusBlocked {
		return "", fmt.Errorf("%s is waiting for a permission in herdr (workspace %s); answer it there", mem.Name, HelpersWorkspace)
	}
	b, err := os.ReadFile(out)
	if err != nil {
		return "", errors.New(mem.Name + " finished without writing an answer")
	}
	return strings.TrimSpace(string(b)), nil
}

func (m *Manager) helpersWorkspace(ctx context.Context, h *herdr.Client) (string, error) {
	m.mu.Lock()
	id := m.workspaces[HelpersWorkspace]
	m.mu.Unlock()
	if snap, err := h.Snapshot(ctx); err == nil {
		for _, w := range snap.Workspaces {
			if (id != "" && w.WorkspaceID == id) || (id == "" && w.Label == HelpersWorkspace) {
				m.mu.Lock()
				m.workspaces[HelpersWorkspace] = w.WorkspaceID
				m.mu.Unlock()
				return w.WorkspaceID, nil
			}
		}
	}
	cr, err := h.CreateWorkspace(ctx, filepath.Join(m.cfg.Dir(), "helpers"), HelpersWorkspace)
	if err != nil {
		return "", err
	}
	m.mu.Lock()
	m.workspaces[HelpersWorkspace] = cr.Workspace.WorkspaceID
	m.mu.Unlock()
	return cr.Workspace.WorkspaceID, nil
}
