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
	"github.com/techmuch/go-build-it/internal/history"
	"github.com/techmuch/go-build-it/internal/team"
)

var askSeq atomic.Int64

// HelpersWorkspace is the herdr workspace that hosts one-shot helper conversations.
const HelpersWorkspace = "gbi-helpers"

// Ask runs one turn as a persona in a fresh, short-lived herdr conversation and returns
// its answer (artifact and persona edits, Lens assessments, release notes). The prompt
// goes in a file and the answer comes back in a file, so nothing is read off the screen.
// Each helper harness works in a fixed folder (its trust prompt is answered, FR-07f),
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
		"\n\n---\nWrite your complete answer to the file " + filepath.Base(out) + " in this folder (create it), then reply DONE. Do not create or change any other file.\n"
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
			if !herdr.IsCode(err, "agent_not_ready") {
				return "", err
			}
			if !m.answerTrust(ctx, h, mem.ProjectID, mem.Name, prof.Label, sp) {
				return "", fmt.Errorf("%s needs you once in herdr (workspace %s): answer its first-run prompt for %s, then try again", prof.Label, HelpersWorkspace, dir)
			}
		}
	} else {
		bin := firstNonEmpty(hc.Binary, m.GbiBin)
		cmd := ShellCommand(map[string]string{"GBI_FAKE_TRANSCRIPTS": harness.FakeTranscriptDir()}, bin, args)
		if err := h.Call(ctx, herdr.MethodPaneSendText, map[string]any{"pane_id": sp, "text": cmd}, nil); err != nil {
			return "", err
		}
		if err := waitAgent(ctx, h, sp, prof, 30*time.Second); err != nil {
			return "", err
		}
	}
	var seq int64
	trusted := false
	// A harness that is started by typing its command reports a trust screen only as
	// "blocked": answer it before the question is typed into it.
	if ag, e := h.Agent(ctx, sp); e == nil && ag.AgentStatus == herdr.AgentStatusBlocked {
		trusted = true
		if !m.answerTrust(ctx, h, mem.ProjectID, mem.Name, prof.Label, sp) {
			return "", fmt.Errorf("%s is waiting for you in herdr (workspace %s); answer it there, then try again", mem.Name, HelpersWorkspace)
		}
		m.waitNotBlocked(ctx, h, sp)
	}
	for i := 0; ; i++ {
		seq, err = h.Submit(ctx, sp, name, prof.Kind != "", "Read "+filepath.Base(in)+" in this folder and do what it says.") // see prompt below
		// The folder-trust screen can appear after the harness reports it has started: answer
		// it once (FR-07f) and send the question again.
		if err != nil && herdr.IsCode(err, "agent_blocked") && !trusted {
			trusted = true
			if m.answerTrust(ctx, h, mem.ProjectID, mem.Name, prof.Label, sp) {
				m.waitNotBlocked(ctx, h, sp)
				continue
			}
		}
		if err == nil || !herdr.IsCode(err, "agent_not_ready") || i >= 20 {
			break
		}
		time.Sleep(500 * time.Millisecond) // a just-answered harness is marked ready a moment later
	}
	if err != nil {
		return "", err
	}
	// A harness that has only just started can drop the prompt and stay idle. Send it again,
	// twice at most, before giving up (the same rule as for members).
	question := "Read " + filepath.Base(in) + " in this folder and do what it says."
	a, err := h.WaitSettled(ctx, sp, seq, 30*time.Second)
	for retry := 0; errors.Is(err, herdr.ErrPromptStalled) && retry < 2; retry++ {
		if _, statErr := os.Stat(out); statErr == nil {
			break // it answered without the state change being seen
		}
		if seq, err = h.Submit(ctx, sp, name, prof.Kind != "", question); err != nil {
			break
		}
		m.hist.Must(ctx, history.Event{ProjectID: mem.ProjectID, Type: "helper.resubmitted", Subject: mem.ID,
			Summary: "Asked " + mem.Name + " again", Why: mem.Name + " was idle and had not read the question", Data: history.Data(map[string]any{"attempt": retry + 2})})
		a, err = h.WaitSettled(ctx, sp, seq, 30*time.Second)
	}
	if err != nil && !errors.Is(err, herdr.ErrPromptStalled) {
		return "", fmt.Errorf("%s did not answer: %w", mem.Name, err)
	}
	if _, statErr := os.Stat(out); err != nil && statErr != nil {
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

// waitNotBlocked waits up to ten seconds for a harness to leave a screen that was just
// answered, so the next text is not read as part of the answer.
func (m *Manager) waitNotBlocked(ctx context.Context, h *herdr.Client, pane string) {
	for j := 0; j < 40; j++ {
		if ag, err := h.Agent(ctx, pane); err == nil && ag.AgentStatus != herdr.AgentStatusBlocked {
			return
		}
		time.Sleep(250 * time.Millisecond)
	}
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
