package reviews

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// A project with an npm test script has its tests run and counted for the review, for
// Vitest/Jest summaries and for Node's own test runner.
func TestNpmTestsCounted(t *testing.T) {
	if _, err := exec.LookPath("npm"); err != nil {
		t.Skip("npm is not installed")
	}
	for name, script := range map[string]string{
		"node":   `echo 'ℹ tests 3'; echo 'ℹ pass 2'; echo 'ℹ fail 1'; exit 1`,
		"vitest": `echo ' Tests  1 failed | 2 passed (3)'; exit 1`,
	} {
		dir := t.TempDir()
		pkg := `{"name":"x","private":true,"scripts":{"test":"` + script + `"}}`
		_ = os.WriteFile(filepath.Join(dir, "package.json"), []byte(pkg), 0o644)
		_ = os.Mkdir(filepath.Join(dir, "node_modules"), 0o755)
		got := RunTests(context.Background(), dir)
		if !got.Ran || got.Command != "npm test" || got.UnitPassed != 2 || got.UnitFailed != 1 || got.Output == "" {
			t.Fatalf("%s: %+v", name, got)
		}
	}
	if got := RunTests(context.Background(), t.TempDir()); got.Ran {
		t.Fatalf("no test command should mean not run: %+v", got)
	}
}
