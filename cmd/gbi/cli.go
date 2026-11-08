package main

import (
	"bufio"
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/techmuch/go-build-it/internal/api"
	"github.com/techmuch/go-build-it/internal/app"
	"github.com/techmuch/go-build-it/internal/config"
)

type sqlDB = sql.DB

// routes returns the route table without starting services (for CLI parsing and MCP).
func routes() []api.Route {
	a := app.Wire(&config.Config{}, mustMemDB(), version, newLogger())
	return a.Routes()
}

func baseURL() string {
	if u := os.Getenv("GBI_URL"); u != "" {
		return strings.TrimRight(u, "/")
	}
	if cfg, err := config.Load(); err == nil {
		return "http://" + cfg.Listen
	}
	return "http://127.0.0.1:7420"
}

func usage(args []string) error {
	rs := routes()
	if len(args) > 1 {
		return commandHelp(rs, args[1:])
	}
	groups := map[string][]api.Route{}
	var names []string
	for _, r := range rs {
		g := strings.Fields(r.CLI)[0]
		if _, ok := groups[g]; !ok {
			names = append(names, g)
		}
		groups[g] = append(groups[g], r)
	}
	sort.Strings(names)
	fmt.Println(`gbi — go-build-it

Usage:
  gbi serve [--listen 127.0.0.1:7420]   run the server and web UI
  gbi <group> <command> [args] [--flag value] [--json]
  gbi help <group> [command]            show commands and flags
  gbi service plan|install|uninstall    run in the background at login
  gbi mcp                               MCP server over stdio for external agents
  gbi version

Every web UI action has a CLI command (FR-30). Add --json for machine-readable output.

Groups:`)
	w := tabwriter.NewWriter(os.Stdout, 2, 4, 2, ' ', 0)
	for _, g := range names {
		var cmds []string
		for _, r := range groups[g] {
			f := strings.Fields(r.CLI)
			if len(f) > 1 {
				cmds = append(cmds, f[1])
			} else {
				cmds = append(cmds, "(default)")
			}
		}
		fmt.Fprintf(w, "  %s\t%s\n", g, strings.Join(cmds, ", "))
	}
	return w.Flush()
}

func commandHelp(rs []api.Route, words []string) error {
	prefix := strings.Join(words, " ")
	found := false
	for _, r := range rs {
		if r.CLI == prefix || strings.HasPrefix(r.CLI, prefix+" ") || strings.Fields(r.CLI)[0] == prefix {
			found = true
			args := ""
			for _, p := range api.PathParams(r.Path) {
				args += " <" + p + ">"
			}
			fmt.Printf("gbi %s%s\n    %s  [%s %s]\n", r.CLI, args, r.Summary, r.Method, r.Path)
			for _, f := range r.Flags {
				t := f.Type
				if t == "" {
					t = "string"
				}
				fmt.Printf("    --%-16s %-6s %s\n", f.Name, t, f.Help)
			}
		}
	}
	if !found {
		return fmt.Errorf("no command %q", prefix)
	}
	return nil
}

// match finds the route whose CLI words are the longest prefix of args.
func match(rs []api.Route, args []string) (api.Route, []string, bool) {
	var best api.Route
	bestLen := 0
	for _, r := range rs {
		w := strings.Fields(r.CLI)
		if len(w) <= len(args) && len(w) > bestLen {
			ok := true
			for i := range w {
				if w[i] != args[i] {
					ok = false
					break
				}
			}
			if ok {
				best, bestLen = r, len(w)
			}
		}
	}
	return best, args[min(bestLen, len(args)):], bestLen > 0
}

func cliCall(args []string) error {
	rs := routes()
	rt, rest, ok := match(rs, args)
	if !ok {
		return fmt.Errorf("unknown command %q (try `gbi help`)", strings.Join(args, " "))
	}
	var positional []string
	flags := map[string]string{}
	jsonOut := false
	for i := 0; i < len(rest); i++ {
		a := rest[i]
		if a == "--json" {
			jsonOut = true
			continue
		}
		if strings.HasPrefix(a, "--") {
			name, val, has := strings.Cut(strings.TrimPrefix(a, "--"), "=")
			if !has {
				if i+1 < len(rest) && !strings.HasPrefix(rest[i+1], "--") && !isBool(rt, name) {
					val = rest[i+1]
					i++
				} else {
					val = "true"
				}
			}
			flags[name] = val
			continue
		}
		positional = append(positional, a)
	}
	p := rt.Path
	params := api.PathParams(rt.Path)
	for i, name := range params {
		if i >= len(positional) {
			return fmt.Errorf("missing <%s>: gbi %s%s", name, rt.CLI, " <"+strings.Join(params, "> <")+">")
		}
		val := positional[i]
		if strings.Contains(p, "{"+name+"...}") {
			val = strings.Join(positional[i:], "/")
			p = strings.Replace(p, "{"+name+"...}", val, 1)
		} else {
			p = strings.Replace(p, "{"+name+"}", url.PathEscape(val), 1)
		}
	}
	if rt.Name == "events.watch" {
		return watch(p, flags)
	}
	var body io.Reader
	if rt.Method == http.MethodGet || rt.Method == http.MethodDelete {
		q := url.Values{}
		for k, v := range flags {
			q.Set(k, v)
		}
		if len(q) > 0 {
			p += "?" + q.Encode()
		}
	} else {
		obj := map[string]any{}
		if d, ok := flags["data"]; ok {
			if strings.HasPrefix(d, "@") {
				b, err := os.ReadFile(d[1:])
				if err != nil {
					return err
				}
				d = string(b)
			}
			if err := json.Unmarshal([]byte(d), &obj); err != nil {
				return fmt.Errorf("--data: %w", err)
			}
			delete(flags, "data")
		}
		for k, v := range flags {
			obj[k] = typed(rt, k, v)
		}
		b, _ := json.Marshal(obj)
		body = bytes.NewReader(b)
	}
	req, _ := http.NewRequest(rt.Method, baseURL()+p, body)
	req.Header.Set("Content-Type", "application/json")
	for env, h := range map[string]string{"GBI_PERSONA": "X-GBI-Persona", "GBI_TASK": "X-GBI-Task", "GBI_SESSION": "X-GBI-Session"} {
		if v := os.Getenv(env); v != "" {
			req.Header.Set(h, v)
		}
	}
	resp, err := (&http.Client{Timeout: 10 * time.Minute}).Do(req)
	if err != nil {
		return fmt.Errorf("cannot reach go-build-it at %s (is `gbi serve` running?): %w", baseURL(), err)
	}
	defer resp.Body.Close()
	out, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		var e api.Error
		if json.Unmarshal(out, &e) == nil && e.Msg != "" {
			return errors.New(e.Msg)
		}
		return fmt.Errorf("%s: %s", resp.Status, strings.TrimSpace(string(out)))
	}
	if jsonOut {
		_, err := os.Stdout.Write(out)
		return err
	}
	return pretty(out)
}

func isBool(rt api.Route, name string) bool {
	for _, f := range rt.Flags {
		if f.Name == name {
			return f.Type == "bool"
		}
	}
	return false
}

func typed(rt api.Route, name, v string) any {
	for _, f := range rt.Flags {
		if f.Name != name {
			continue
		}
		switch f.Type {
		case "bool":
			return v == "true" || v == "1" || v == "yes"
		case "int":
			n, _ := strconv.Atoi(v)
			return n
		case "json":
			var x any
			if json.Unmarshal([]byte(v), &x) == nil {
				return x
			}
			if strings.Contains(v, ",") || !strings.HasPrefix(v, "{") {
				return strings.Split(v, ",") // comma list shorthand
			}
		}
	}
	return v
}

// pretty prints JSON as a table (arrays of objects) or key: value lines.
func pretty(b []byte) error {
	var v any
	if json.Unmarshal(b, &v) != nil {
		_, err := os.Stdout.Write(b)
		return err
	}
	switch x := v.(type) {
	case []any:
		if len(x) == 0 {
			fmt.Println("(none)")
			return nil
		}
		obj, ok := x[0].(map[string]any)
		if !ok {
			for _, e := range x {
				fmt.Println(e)
			}
			return nil
		}
		cols := pickCols(obj)
		w := tabwriter.NewWriter(os.Stdout, 2, 4, 2, ' ', 0)
		fmt.Fprintln(w, strings.ToUpper(strings.Join(cols, "\t")))
		for _, e := range x {
			m, _ := e.(map[string]any)
			var cells []string
			for _, c := range cols {
				cells = append(cells, cell(m[c]))
			}
			fmt.Fprintln(w, strings.Join(cells, "\t"))
		}
		return w.Flush()
	case map[string]any:
		keys := make([]string, 0, len(x))
		for k := range x {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		w := tabwriter.NewWriter(os.Stdout, 2, 4, 2, ' ', 0)
		for _, k := range keys {
			fmt.Fprintf(w, "%s:\t%s\n", k, cell(x[k]))
		}
		return w.Flush()
	}
	fmt.Println(v)
	return nil
}

func pickCols(m map[string]any) []string {
	pref := []string{"id", "kind", "name", "title", "status", "stage", "role", "projectId", "priority", "blocking", "summary", "actor", "ts", "cli", "statusLine"}
	var cols []string
	for _, p := range pref {
		if v, ok := m[p]; ok && scalar(v) {
			cols = append(cols, p)
		}
		if len(cols) >= 6 {
			break
		}
	}
	if len(cols) == 0 {
		for k, v := range m {
			if scalar(v) && len(cols) < 6 {
				cols = append(cols, k)
			}
		}
		sort.Strings(cols)
	}
	return cols
}

func scalar(v any) bool {
	switch v.(type) {
	case string, float64, bool, nil:
		return true
	}
	return false
}

func cell(v any) string {
	switch x := v.(type) {
	case nil:
		return ""
	case string:
		s := strings.ReplaceAll(x, "\n", " ")
		if len(s) > 70 {
			s = s[:70] + "…"
		}
		return s
	case float64:
		return strconv.FormatFloat(x, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(x)
	}
	b, _ := json.Marshal(v)
	s := string(b)
	if len(s) > 70 {
		s = s[:70] + "…"
	}
	return s
}

func watch(p string, flags map[string]string) error {
	q := url.Values{}
	for k, v := range flags {
		q.Set(k, v)
	}
	u := baseURL() + p
	if len(q) > 0 {
		u += "?" + q.Encode()
	}
	resp, err := http.Get(u)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	sc := bufio.NewScanner(resp.Body)
	sc.Buffer(make([]byte, 0, 1<<20), 8<<20)
	for sc.Scan() {
		if d, ok := strings.CutPrefix(sc.Text(), "data: "); ok {
			var e struct{ TS, Type, Actor, Summary string }
			if json.Unmarshal([]byte(d), &e) == nil {
				fmt.Printf("%s  %-22s %-10s %s\n", e.TS, e.Type, e.Actor, e.Summary)
			}
		}
	}
	return sc.Err()
}
