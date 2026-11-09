package sessions

import (
	"runtime"
	"strings"
	"testing"
)

func TestShellCommand(t *testing.T) {
	got := ShellCommand(map[string]string{"B": "it's", "A": "/x y"}, "/bin/gbi", []string{"dev", "fake-member"})
	if runtime.GOOS == "windows" {
		if got != "$env:A='/x y'; $env:B='it''s'; & '/bin/gbi' 'dev' 'fake-member'\r\n" {
			t.Fatalf("%q", got)
		}
		return
	}
	if got != `A='/x y' B='it'\''s' '/bin/gbi' 'dev' 'fake-member'`+"\n" {
		t.Fatalf("%q", got)
	}
	if !strings.HasSuffix(got, "\n") {
		t.Fatal("must end with Enter")
	}
}
