// Package api is the HTTP API every client uses (FR-31). Routes are declared once with
// their CLI words, so the CLI, the OpenAPI spec and the TypeScript client are generated
// from the same table and stay in parity (FR-30, FR-32).
package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strings"

	"github.com/techmuch/go-build-it/internal/access"
)

// Route declares one API operation.
type Route struct {
	Method  string                  `json:"method"`
	Path    string                  `json:"path"`    // Go 1.22 pattern path, e.g. /api/projects/{id}
	Name    string                  `json:"name"`    // dotted name, e.g. projects.get
	CLI     string                  `json:"cli"`     // CLI words, e.g. "projects get"; path params become positional args
	Summary string                  `json:"summary"` // one line
	Flags   []Flag                  `json:"flags,omitempty"`
	Handler func(*Ctx) (any, error) `json:"-"`
}

// Flag documents a query or body field for the CLI and OpenAPI.
type Flag struct {
	Name string `json:"name"`
	Help string `json:"help"`
	Type string `json:"type,omitempty"` // string (default) | int | bool | json
}

// Ctx is passed to handlers.
type Ctx struct {
	W     http.ResponseWriter
	R     *http.Request
	Actor access.Actor
}

// Param returns a path parameter.
func (c *Ctx) Param(name string) string { return c.R.PathValue(name) }

// Query returns a query parameter.
func (c *Ctx) Query(name string) string { return c.R.URL.Query().Get(name) }

// Bind decodes the JSON body into v.
func (c *Ctx) Bind(v any) error {
	if c.R.Body == nil {
		return nil
	}
	dec := json.NewDecoder(c.R.Body)
	if err := dec.Decode(v); err != nil && !errors.Is(err, io.EOF) {
		return BadRequest("invalid JSON body: %v", err)
	}
	return nil
}

// Error is an API error with an HTTP status.
type Error struct {
	Status int    `json:"-"`
	Msg    string `json:"error"`
}

func (e *Error) Error() string { return e.Msg }

// BadRequest returns a 400 error.
func BadRequest(f string, a ...any) error { return &Error{http.StatusBadRequest, fmt.Sprintf(f, a...)} }

// NotFound returns a 404 error.
func NotFound(f string, a ...any) error { return &Error{http.StatusNotFound, fmt.Sprintf(f, a...)} }

// Conflict returns a 409 error.
func Conflict(f string, a ...any) error { return &Error{http.StatusConflict, fmt.Sprintf(f, a...)} }

// Forbidden returns a 403 error.
func Forbidden(f string, a ...any) error { return &Error{http.StatusForbidden, fmt.Sprintf(f, a...)} }

// Raw marks a handler that wrote the response itself.
type Raw struct{}

// Mount registers routes on mux.
func Mount(mux *http.ServeMux, routes []Route) {
	for _, rt := range routes {
		rt := rt
		mux.HandleFunc(rt.Method+" "+rt.Path, func(w http.ResponseWriter, r *http.Request) {
			c := &Ctx{W: w, R: r, Actor: access.FromContext(r.Context())}
			v, err := rt.Handler(c)
			if err != nil {
				WriteError(w, err)
				return
			}
			if _, ok := v.(Raw); ok {
				return
			}
			WriteJSON(w, http.StatusOK, v)
		})
	}
}

// WriteJSON writes v as JSON.
func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(v)
}

// WriteError writes err as a JSON error.
func WriteError(w http.ResponseWriter, err error) {
	var e *Error
	if errors.As(err, &e) {
		WriteJSON(w, e.Status, e)
		return
	}
	WriteJSON(w, http.StatusInternalServerError, &Error{Msg: err.Error()})
}

var paramRe = regexp.MustCompile(`\{([a-zA-Z0-9_]+)(\.\.\.)?\}`)

// PathParams returns the parameter names in a route path, in order.
func PathParams(path string) []string {
	var out []string
	for _, m := range paramRe.FindAllStringSubmatch(path, -1) {
		out = append(out, m[1])
	}
	return out
}

// OpenAPI builds an OpenAPI 3.1 document from routes.
func OpenAPI(routes []Route, version string) map[string]any {
	paths := map[string]map[string]any{}
	for _, rt := range routes {
		p := paths[rt.Path]
		if p == nil {
			p = map[string]any{}
			paths[rt.Path] = p
		}
		var params []map[string]any
		for _, n := range PathParams(rt.Path) {
			params = append(params, map[string]any{"name": n, "in": "path", "required": true, "schema": map[string]string{"type": "string"}})
		}
		op := map[string]any{"operationId": rt.Name, "summary": rt.Summary, "x-cli": "gbi " + rt.CLI,
			"responses": map[string]any{"200": map[string]any{"description": "OK"}}}
		if rt.Method == http.MethodGet {
			for _, f := range rt.Flags {
				params = append(params, map[string]any{"name": f.Name, "in": "query", "description": f.Help, "schema": map[string]string{"type": typeOr(f.Type)}})
			}
		} else if len(rt.Flags) > 0 {
			props := map[string]any{}
			for _, f := range rt.Flags {
				props[f.Name] = map[string]any{"description": f.Help, "type": typeOr(f.Type)}
			}
			op["requestBody"] = map[string]any{"content": map[string]any{"application/json": map[string]any{"schema": map[string]any{"type": "object", "properties": props}}}}
		}
		if params != nil {
			op["parameters"] = params
		}
		p[strings.ToLower(rt.Method)] = op
	}
	return map[string]any{"openapi": "3.1.0", "info": map[string]any{"title": "go-build-it API", "version": version}, "paths": paths}
}

func typeOr(t string) string {
	switch t {
	case "int":
		return "integer"
	case "bool":
		return "boolean"
	case "json":
		return "object"
	}
	return "string"
}

// TSClient generates the TypeScript client source (web/src/api.gen.ts).
func TSClient(routes []Route) string {
	sorted := append([]Route(nil), routes...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Name < sorted[j].Name })
	var b strings.Builder
	b.WriteString("// Code generated by `gbi dev gen-client`. DO NOT EDIT.\n")
	b.WriteString("import { call } from './http';\n\n")
	groups := map[string][]Route{}
	var order []string
	for _, rt := range sorted {
		g, _, _ := strings.Cut(rt.Name, ".")
		if _, ok := groups[g]; !ok {
			order = append(order, g)
		}
		groups[g] = append(groups[g], rt)
	}
	b.WriteString("export const api = {\n")
	for _, g := range order {
		fmt.Fprintf(&b, "  %s: {\n", g)
		for _, rt := range groups[g] {
			_, fn, _ := strings.Cut(rt.Name, ".")
			fn = camel(fn)
			params := PathParams(rt.Path)
			args := make([]string, 0, len(params)+1)
			for _, p := range params {
				args = append(args, p+": string")
			}
			path := "`" + paramRe.ReplaceAllStringFunc(rt.Path, func(m string) string {
				sub := paramRe.FindStringSubmatch(m)
				if sub[2] != "" {
					return "${encodeURI(" + sub[1] + ")}" // rest-of-path params keep their slashes
				}
				return "${encodeURIComponent(" + sub[1] + ")}"
			}) + "`"
			if rt.Method == http.MethodGet {
				args = append(args, "query?: Record<string, string | number | boolean | undefined>")
				fmt.Fprintf(&b, "    /** %s */\n    %s: (%s) => call<any>('GET', %s, undefined, query),\n", rt.Summary, fn, strings.Join(args, ", "), path)
			} else {
				args = append(args, "body?: unknown")
				fmt.Fprintf(&b, "    /** %s */\n    %s: (%s) => call<any>('%s', %s, body),\n", rt.Summary, fn, strings.Join(args, ", "), rt.Method, path)
			}
		}
		b.WriteString("  },\n")
	}
	b.WriteString("};\n")
	return b.String()
}

func camel(s string) string {
	parts := strings.FieldsFunc(s, func(r rune) bool { return r == '-' || r == '_' || r == '.' })
	for i := 1; i < len(parts); i++ {
		parts[i] = strings.ToUpper(parts[i][:1]) + parts[i][1:]
	}
	return strings.Join(parts, "")
}
