package harness

import (
	"context"
	"errors"
	"os/exec"
	"strings"
	"time"
)

// Model is one model a harness can run (FR-63a).
type Model struct {
	ID     string `json:"id"`               // what goes after --model
	Name   string `json:"name,omitempty"`   // friendly name, when the harness gives one
	Detail string `json:"detail,omitempty"` // context window, effort, capabilities
}

// Models lists the models a harness offers. Harnesses with a list command are asked
// (`agy models`, `pi --list-models`, `opencode models`); Claude Code has none, so its
// aliases and current model IDs ship with go-build-it. The second result names where
// the list came from. Like `--version`, this is a short query, not an agent session.
func (p Profile) Models(ctx context.Context, bin string) ([]Model, string, error) {
	if p.models == nil {
		return []Model{}, "", errors.New(p.Label + " cannot list its models; type a model name")
	}
	return p.models(ctx, firstNonEmptyStr(bin, p.Binary))
}

// listCommand runs a harness's list command and parses its output.
func listCommand(args []string, parse func(string) []Model) func(context.Context, string) ([]Model, string, error) {
	return func(ctx context.Context, bin string) ([]Model, string, error) {
		src := strings.Join(append([]string{bin}, args...), " ")
		if _, err := exec.LookPath(bin); err != nil {
			return []Model{}, src, errors.New(bin + " is not installed")
		}
		ctx, cancel := context.WithTimeout(ctx, 45*time.Second)
		defer cancel()
		out, err := exec.CommandContext(ctx, bin, args...).Output()
		ms := parse(string(out))
		if len(ms) == 0 {
			if err == nil {
				err = errors.New("the list was empty")
			}
			return []Model{}, src, errors.New(src + ": " + err.Error())
		}
		return ms, src, nil
	}
}

// fixed returns a built-in list.
func fixed(src string, ms []Model) func(context.Context, string) ([]Model, string, error) {
	return func(context.Context, string) ([]Model, string, error) { return ms, src, nil }
}

// claudeModels: Claude Code accepts an alias for the latest model of a family, or a full ID.
var claudeModels = []Model{
	{ID: "opus", Name: "Opus (latest)", Detail: "alias for the latest Opus"},
	{ID: "sonnet", Name: "Sonnet (latest)", Detail: "alias for the latest Sonnet"},
	{ID: "haiku", Name: "Haiku (latest)", Detail: "alias for the latest Haiku"},
	{ID: "fable", Name: "Fable (latest)", Detail: "alias for the latest Fable"},
	{ID: "claude-opus-5-5", Name: "Claude Opus 5.5"},
	{ID: "claude-sonnet-5-5", Name: "Claude Sonnet 5.5"},
	{ID: "claude-haiku-5-5", Name: "Claude Haiku 5.5"},
	{ID: "claude-fable-5-2", Name: "Claude Fable 5.2"},
}

// parseAgy reads `agy models`: "id<TAB>Name" lines after a "Fetching…" line.
func parseAgy(out string) []Model {
	ms := []Model{}
	for _, l := range strings.Split(out, "\n") {
		id, name, ok := strings.Cut(strings.TrimSpace(l), "\t")
		if !ok || id == "" || strings.ContainsAny(id, " ") {
			continue
		}
		ms = append(ms, Model{ID: id, Name: strings.TrimSpace(name)})
	}
	return ms
}

// parsePi reads `pi --list-models`: a table "provider model context max-out thinking images".
func parsePi(out string) []Model {
	ms := []Model{}
	for _, l := range strings.Split(out, "\n") {
		f := strings.Fields(l)
		if len(f) < 2 || f[0] == "provider" {
			continue
		}
		m := Model{ID: f[0] + "/" + f[1], Name: f[1]}
		var d []string
		if len(f) > 2 {
			d = append(d, "context "+f[2])
		}
		if len(f) > 3 {
			d = append(d, "max output "+f[3])
		}
		if len(f) > 4 && f[4] == "yes" {
			d = append(d, "thinking")
		}
		if len(f) > 5 && f[5] == "yes" {
			d = append(d, "images")
		}
		m.Detail = strings.Join(d, " · ")
		ms = append(ms, m)
	}
	return ms
}

// parseLines reads one "provider/model" per line (`opencode models`).
func parseLines(out string) []Model {
	ms := []Model{}
	for _, l := range strings.Split(out, "\n") {
		l = strings.TrimSpace(l)
		if l == "" || strings.ContainsAny(l, " \t") || !strings.Contains(l, "/") {
			continue
		}
		ms = append(ms, Model{ID: l})
	}
	return ms
}

func firstNonEmptyStr(s ...string) string {
	for _, x := range s {
		if x != "" {
			return x
		}
	}
	return ""
}
