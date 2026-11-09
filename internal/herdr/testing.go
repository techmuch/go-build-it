package herdr

import (
	"context"
	"os"
	"os/exec"
	"testing"
	"time"
)

// TestSession starts an isolated herdr (its own XDG_CONFIG_HOME, so the user's herdr
// is never touched) and returns its supervisor. It skips the test when herdr is not
// installed, unless GBI_REQUIRE_HERDR=1 (CI), where it fails instead.
func TestSession(t testing.TB) *Supervisor {
	t.Helper()
	bin, err := exec.LookPath("herdr")
	if err != nil {
		if os.Getenv("GBI_REQUIRE_HERDR") == "1" {
			t.Fatal("herdr is required (GBI_REQUIRE_HERDR=1) but not on PATH")
		}
		t.Skip("herdr not installed")
	}
	// Unix socket paths are limited to ~104 bytes, so keep the directory short.
	home, err := os.MkdirTemp("", "gh")
	if err != nil {
		t.Fatal(err)
	}
	s := NewSupervisor(bin, "gbi", home, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if _, err := s.Ensure(ctx); err != nil {
		t.Fatalf("start isolated herdr: %v", err)
	}
	t.Cleanup(func() {
		c, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = s.Stop(c)
		os.RemoveAll(home)
	})
	return s
}
