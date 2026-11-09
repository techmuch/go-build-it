package main

import "testing"

func TestMemberFromPath(t *testing.T) {
	for in, want := range map[string]string{
		"/Users/d/.go-build-it/worktrees/inboxql/members/forge":         "forge@inboxql",
		"/Users/d/.go-build-it/worktrees/inboxql/members/forge/web/src": "forge@inboxql",
		"/Users/d/projects/inboxql":                                     "",
	} {
		if got := memberFromPath(in); got != want {
			t.Errorf("memberFromPath(%q) = %q, want %q", in, got, want)
		}
	}
}
