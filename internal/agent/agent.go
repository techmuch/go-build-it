// Package agent makes one-shot calls to a persona for conversational work: the artifact
// editor chat (FR-45a), the persona coach (FR-13), standards editing (FR-51) and Lens's
// assessment narrative (FR-112). Each call is a short-lived conversation in herdr
// (sessions.Manager.Ask, FR-60): there is no headless harness path.
package agent

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/techmuch/go-build-it/internal/config"
	"github.com/techmuch/go-build-it/internal/team"
)

// Asker runs one turn as a member and returns the answer.
type Asker func(ctx context.Context, m team.Member, prompt string) (string, error)

// Caller runs prompt-only agent turns.
type Caller struct {
	cfg  *config.Config
	team *team.Service
	ask  Asker
}

// New returns a Caller that asks through ask.
func New(cfg *config.Config, t *team.Service, ask Asker) *Caller { return &Caller{cfg: cfg, team: t, ask: ask} }

// Reply is an agent's answer, optionally with a revised document.
type Reply struct {
	Persona  string `json:"persona"`
	Message  string `json:"message"`
	Document string `json:"document,omitempty"`
	Hunks    []Hunk `json:"hunks,omitempty"`
}

// Ask runs one turn as the given member (or archetype) and returns its text.
func (c *Caller) Ask(ctx context.Context, memberID, prompt string) (string, error) {
	m, err := c.team.Member(ctx, memberID)
	if err != nil {
		return "", err
	}
	if m.Kind != "ai" {
		return "", fmt.Errorf("%s is not an AI persona", m.Name)
	}
	if m.Harness == "courier" {
		return "", errors.New("Courier is built in and does not chat; pick another persona")
	}
	if c.ask == nil {
		return "", errors.New("agent calls are not connected")
	}
	return c.ask(ctx, m, prompt)
}

var docRe = regexp.MustCompile(`(?s)<<<DOC\s*\n(.*?)\n?DOC>>>`)

// Edit asks a persona to change a document. The agent answers with a short message and
// the full revised document between <<<DOC and DOC>>> markers; Edit returns the hunks.
func (c *Caller) Edit(ctx context.Context, memberID, kind, path, doc, request string, history []string) (Reply, error) {
	var b strings.Builder
	fmt.Fprintf(&b, "You are helping a human edit a %s (%s).\n\n", kind, path)
	if len(history) > 0 {
		b.WriteString("Conversation so far:\n")
		for _, h := range history {
			b.WriteString(h + "\n")
		}
		b.WriteString("\n")
	}
	fmt.Fprintf(&b, "Current document:\n<<<DOC\n%s\nDOC>>>\n\nRequest: %s\n\n", doc, request)
	b.WriteString("Reply with one or two sentences explaining what you changed, then the COMPLETE revised document between a line '<<<DOC' and a line 'DOC>>>'. If no change is needed, explain why and omit the document. Do not use tools.")
	out, err := c.Ask(ctx, memberID, b.String())
	if err != nil {
		return Reply{}, err
	}
	r := Reply{Persona: memberID}
	if m := docRe.FindStringSubmatch(out); m != nil {
		r.Document = m[1]
		r.Message = strings.TrimSpace(docRe.ReplaceAllString(out, ""))
		r.Hunks = Diff(doc, r.Document)
	} else {
		r.Message = out
	}
	return r, nil
}

// Hunk is one changed region of a line diff.
type Hunk struct {
	Start   int      `json:"start"`   // first line index in the old text
	Removed []string `json:"removed"` // old lines
	Added   []string `json:"added"`   // new lines
	Context string   `json:"context"` // previous unchanged line, for display
}

// Diff returns line hunks turning a into b (LCS-based).
func Diff(a, b string) []Hunk {
	x, y := strings.Split(a, "\n"), strings.Split(b, "\n")
	n, m := len(x), len(y)
	lcs := make([][]int, n+1)
	for i := range lcs {
		lcs[i] = make([]int, m+1)
	}
	for i := n - 1; i >= 0; i-- {
		for j := m - 1; j >= 0; j-- {
			if x[i] == y[j] {
				lcs[i][j] = lcs[i+1][j+1] + 1
			} else if lcs[i+1][j] >= lcs[i][j+1] {
				lcs[i][j] = lcs[i+1][j]
			} else {
				lcs[i][j] = lcs[i][j+1]
			}
		}
	}
	var hunks []Hunk
	var cur *Hunk
	flush := func() {
		if cur != nil {
			hunks = append(hunks, *cur)
			cur = nil
		}
	}
	i, j := 0, 0
	for i < n || j < m {
		switch {
		case i < n && j < m && x[i] == y[j]:
			flush()
			i, j = i+1, j+1
		case j < m && (i == n || lcs[i][j+1] >= lcs[i+1][j]):
			if cur == nil {
				cur = &Hunk{Start: i, Removed: []string{}, Added: []string{}, Context: ctxLine(x, i)}
			}
			cur.Added = append(cur.Added, y[j])
			j++
		default:
			if cur == nil {
				cur = &Hunk{Start: i, Removed: []string{}, Added: []string{}, Context: ctxLine(x, i)}
			}
			cur.Removed = append(cur.Removed, x[i])
			i++
		}
	}
	flush()
	return hunks
}

func ctxLine(x []string, i int) string {
	if i > 0 {
		return x[i-1]
	}
	return ""
}

// Apply applies the accepted hunks (by index) to a, returning the new text.
func Apply(a string, hunks []Hunk, accepted []int) string {
	ok := map[int]bool{}
	for _, k := range accepted {
		ok[k] = true
	}
	x := strings.Split(a, "\n")
	var out []string
	pos := 0
	for k, h := range hunks {
		out = append(out, x[pos:h.Start]...)
		pos = h.Start
		if ok[k] {
			out = append(out, h.Added...)
		} else {
			out = append(out, x[h.Start:h.Start+len(h.Removed)]...)
		}
		pos += len(h.Removed)
	}
	out = append(out, x[pos:]...)
	return strings.Join(out, "\n")
}

func firstNonEmpty(a ...string) string {
	for _, s := range a {
		if s != "" {
			return s
		}
	}
	return ""
}
