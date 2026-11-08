package api

import (
	"strings"
	"testing"
)

func TestOpenAPIAndClient(t *testing.T) {
	rs := []Route{
		{Method: "GET", Path: "/api/projects/{id}", Name: "projects.get", CLI: "projects get", Summary: "Get"},
		{Method: "PUT", Path: "/api/projects/{id}/artifacts/{path...}", Name: "artifacts.save", CLI: "artifacts save", Summary: "Save", Flags: []Flag{{Name: "content"}}},
	}
	doc := OpenAPI(rs, "test")
	paths := doc["paths"].(map[string]map[string]any)
	if _, ok := paths["/api/projects/{id}"]["get"]; !ok {
		t.Fatal("missing get op")
	}
	ts := TSClient(rs)
	if !strings.Contains(ts, "encodeURI(path)") || !strings.Contains(ts, "encodeURIComponent(id)") {
		t.Fatalf("client:\n%s", ts)
	}
	if got := PathParams("/a/{x}/b/{y...}"); len(got) != 2 || got[1] != "y" {
		t.Fatalf("params %v", got)
	}
}
