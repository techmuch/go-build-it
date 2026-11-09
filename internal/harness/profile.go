// Package harness describes each agent harness go-build-it runs inside herdr (FR-61,
// ADR-009): its herdr agent kind, how to launch and resume a conversation, where its
// transcript lives and how to read events, context usage and limits from it.
//
// There is no process handling here: herdr owns every harness process.
package harness

import (
	"context"
	"crypto/rand"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// Profile describes one harness.
type Profile struct {
	Name        string // go-build-it name: claude, antigravity, pi, opencode, fake
	Label       string // shown in the UI
	Kind        string // herdr agent kind; "" for custom agents (the fake member)
	Binary      string // default executable
	Integration string // herdr integration target ("" when none)
	// ChoosesID: go-build-it picks the conversation ID at launch (claude, pi, fake).
	// Otherwise the ID is learned after launch (herdr agent_session, or FindNewConversation).
	ChoosesID bool
	// Window is the default context window when the transcript does not say.
	Window int
	// UsageMeasured: the transcript carries real token counts.
	UsageMeasured bool
	// Verified against a real run in herdr (ADR-009 spike, P13).
	Verified bool

	launch     func(model string, flags []string, conv string, resume bool) []string
	transcript func(cwd, conv string) string
	newReader  func() Reader
	findNewer  func(since time.Time) string
}

// Args returns the harness arguments (after the executable) to start a new conversation
// (resume=false, conv = the chosen ID or "") or to resume conv.
func (p Profile) Args(model string, flags []string, conv string, resume bool) []string {
	return p.launch(model, flags, conv, resume)
}

// TranscriptPath returns the transcript file for a conversation, or "" if unknown.
func (p Profile) TranscriptPath(cwd, conv string) string {
	if p.transcript == nil || conv == "" {
		return ""
	}
	return p.transcript(cwd, conv)
}

// NewReader returns a transcript reader, or nil when the harness has none.
func (p Profile) NewReader() Reader {
	if p.newReader == nil {
		return nil
	}
	return p.newReader()
}

// FindNewConversation looks for a conversation started after since (harnesses that do
// not let go-build-it choose the ID).
func (p Profile) FindNewConversation(since time.Time) string {
	if p.findNewer == nil {
		return ""
	}
	return p.findNewer(since)
}

// NewConversationID returns a fresh conversation ID for harnesses that accept one.
func (p Profile) NewConversationID() string {
	if !p.ChoosesID {
		return ""
	}
	if p.Name == "fake" {
		return "fake-" + uuid()[:13]
	}
	return uuid()
}

var profiles = map[string]Profile{}

func register(p Profile) { profiles[p.Name] = p }

// Get returns a profile by name.
func Get(name string) (Profile, bool) { p, ok := profiles[name]; return p, ok }

// Names lists the real harnesses (not the fake), sorted.
func Names() []string {
	var out []string
	for n := range profiles {
		if n != "fake" {
			out = append(out, n)
		}
	}
	sort.Strings(out)
	return out
}

func home() string { h, _ := os.UserHomeDir(); return h }

var nonAlnum = regexp.MustCompile(`[^A-Za-z0-9]`)

func withModel(args []string, flag, model string) []string {
	if model != "" {
		args = append(args, flag, model)
	}
	return args
}

func init() {
	register(Profile{Name: "claude", Label: "Claude Code", Kind: "claude", Binary: "claude", Integration: "claude", ChoosesID: true, Window: 200_000, UsageMeasured: true, Verified: true,
		launch: func(model string, flags []string, conv string, resume bool) []string {
			args := []string{"--permission-mode", "acceptEdits"}
			args = withModel(args, "--model", model)
			if resume {
				args = append(args, "--resume", conv)
			} else if conv != "" {
				args = append(args, "--session-id", conv)
			}
			return append(args, flags...)
		},
		// ~/.claude/projects/<cwd with every non-alphanumeric character as "-">/<id>.jsonl (ADR-009 spike).
		transcript: func(cwd, conv string) string {
			base := os.Getenv("CLAUDE_CONFIG_DIR")
			if base == "" {
				base = filepath.Join(home(), ".claude")
			}
			if real, err := filepath.EvalSymlinks(cwd); err == nil {
				cwd = real
			}
			return filepath.Join(base, "projects", nonAlnum.ReplaceAllString(cwd, "-"), conv+".jsonl")
		},
		newReader: func() Reader { return &claudeReader{} },
	})
	register(Profile{Name: "antigravity", Label: "Antigravity", Kind: "agy", Binary: "agy", Integration: "antigravity_cli", Window: 1_000_000, UsageMeasured: true,
		launch: func(model string, flags []string, conv string, resume bool) []string {
			args := []string{"--mode", "accept-edits"}
			args = withModel(args, "--model", model)
			if resume && conv != "" {
				args = append(args, "--conversation", conv)
			}
			return append(args, flags...)
		},
		// ~/.gemini/antigravity-cli/brain/<id>/.system_generated/logs/transcript_full.jsonl (Ovis server/agy-store.ts).
		transcript: func(_, conv string) string {
			return filepath.Join(agyRoot(), "brain", conv, ".system_generated", "logs", "transcript_full.jsonl")
		},
		newReader: func() Reader { return &agyReader{} },
		findNewer: func(since time.Time) string { return newestDir(filepath.Join(agyRoot(), "brain"), since) },
	})
	register(Profile{Name: "pi", Label: "Pi", Kind: "pi", Binary: "pi", Integration: "pi", ChoosesID: true, Window: 200_000, UsageMeasured: true,
		launch: func(model string, flags []string, conv string, resume bool) []string {
			args := withModel(nil, "--model", model)
			if resume {
				args = append(args, "--session", conv)
			} else if conv != "" {
				args = append(args, "--session-id", conv)
			}
			return append(args, flags...)
		},
		// <agent dir>/sessions/<cwd>/<timestamp>_<id>.jsonl (Ovis server/pi.ts).
		transcript: func(_, conv string) string {
			dir := os.Getenv("PI_CODING_AGENT_SESSION_DIR")
			if dir == "" {
				agent := os.Getenv("PI_CODING_AGENT_DIR")
				if agent == "" {
					agent = filepath.Join(home(), ".pi", "agent")
				}
				dir = filepath.Join(agent, "sessions")
			}
			m, _ := filepath.Glob(filepath.Join(dir, "*", "*_"+conv+".jsonl"))
			if len(m) > 0 {
				return m[0]
			}
			return ""
		},
		newReader: func() Reader { return &piReader{} },
	})
	register(Profile{Name: "opencode", Label: "OpenCode", Kind: "opencode", Binary: "opencode", Integration: "opencode", Window: 200_000,
		launch: func(model string, flags []string, conv string, resume bool) []string {
			args := withModel(nil, "--model", model)
			if resume && conv != "" {
				args = append(args, "--session", conv)
			}
			return append(args, flags...)
		},
		// OpenCode keeps sessions in a SQLite database, not a transcript file: usage is unavailable.
	})
	register(Profile{Name: "fake", Label: "Fake (scripted)", Binary: "", ChoosesID: true, Window: 200_000, UsageMeasured: true, Verified: true,
		launch: func(model string, flags []string, conv string, resume bool) []string {
			if resume {
				return append([]string{"dev", "fake-member", "--resume", conv}, flags...)
			}
			return append([]string{"dev", "fake-member", "--session-id", conv}, flags...)
		},
		transcript: func(_, conv string) string { return filepath.Join(FakeTranscriptDir(), conv+".jsonl") },
		newReader:  func() Reader { return &claudeReader{} },
	})
}

// FakeTranscriptDir is where the fake member writes (same rule as internal/fakemember).
func FakeTranscriptDir() string {
	if d := os.Getenv("GBI_FAKE_TRANSCRIPTS"); d != "" {
		return d
	}
	if d := os.Getenv("GBI_HOME"); d != "" {
		return filepath.Join(d, "fake-transcripts")
	}
	return filepath.Join(home(), ".go-build-it", "fake-transcripts")
}

func agyRoot() string {
	if d := os.Getenv("ANTIGRAVITY_APP_DATA_DIR"); d != "" {
		return d
	}
	return filepath.Join(home(), ".gemini", "antigravity-cli")
}

func newestDir(dir string, since time.Time) string {
	es, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}
	best, bestT := "", since
	for _, e := range es {
		if !e.IsDir() {
			continue
		}
		if fi, err := e.Info(); err == nil && fi.ModTime().After(bestT) {
			best, bestT = e.Name(), fi.ModTime()
		}
	}
	return best
}

// Detection is what Settings shows for a harness (FR-63).
type Detection struct {
	Name          string `json:"name"`
	Label         string `json:"label"`
	Binary        string `json:"binary"`
	Path          string `json:"path"`
	Version       string `json:"version"`
	Installed     bool   `json:"installed"`
	Kind          string `json:"kind"`          // herdr agent kind
	Integration   string `json:"integration"`   // herdr integration target
	UsageReported string `json:"usageReported"` // measured | unavailable
	Verified      bool   `json:"verified"`
}

// Detect finds a harness binary and its version.
func Detect(ctx context.Context, p Profile, binary string) Detection {
	if binary == "" {
		binary = p.Binary
	}
	d := Detection{Name: p.Name, Label: p.Label, Binary: binary, Kind: p.Kind, Integration: p.Integration, Verified: p.Verified, UsageReported: "unavailable"}
	if p.UsageMeasured {
		d.UsageReported = "measured"
	}
	path, err := exec.LookPath(binary)
	if err != nil {
		return d
	}
	d.Path, d.Installed = path, true
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if out, err := exec.CommandContext(ctx, path, "--version").CombinedOutput(); err == nil {
		d.Version = strings.TrimSpace(strings.SplitN(string(out), "\n", 2)[0])
	}
	return d
}

// EstimateWindow picks a context window for a model when the transcript does not report one.
func EstimateWindow(model string, def int) int {
	m := strings.ToLower(model)
	switch {
	case strings.Contains(m, "gemini"):
		return 1_000_000
	case strings.Contains(m, "[1m]"), strings.Contains(m, "1m"):
		return 1_000_000
	case def > 0:
		return def
	}
	return 200_000
}

func uuid() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
