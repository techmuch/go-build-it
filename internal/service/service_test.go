package service

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestServicePathKeepsInstallerPath(t *testing.T) {
	sep := string(os.PathListSeparator)
	home := filepath.Join(string(filepath.Separator)+"home", "me")
	cur := strings.Join([]string{filepath.Join(home, "bin"), "/usr/bin"}, sep)
	got := filepath.SplitList(ServicePath(cur, home))
	if got[0] != filepath.Join(home, "bin") || got[1] != "/usr/bin" {
		t.Fatalf("installer PATH not kept first: %v", got)
	}
	has := map[string]int{}
	for _, d := range got {
		has[d]++
	}
	if has[filepath.Join(home, ".local", "bin")] != 1 {
		t.Fatalf("~/.local/bin missing: %v", got)
	}
	if has["/usr/bin"] != 1 {
		t.Fatalf("duplicates not removed: %v", got)
	}
}

func TestBuildUsesServicePath(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("PATH only written for launchd and systemd")
	}
	t.Setenv("PATH", "/custom/tools"+string(os.PathListSeparator)+"/usr/bin")
	p, err := Build("/x/gbi", "/x/home")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(p.Content, "/custom/tools") || !strings.Contains(p.Content, filepath.Join(".local", "bin")) {
		t.Fatalf("service file lacks PATH:\n%s", p.Content)
	}
}
