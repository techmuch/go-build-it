package reviews

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"
)

// A project with an npm test script has its tests run and counted for the review, for
// Vitest/Jest summaries and for Node's own test runner.
func TestNpmTestsCounted(t *testing.T) {
	if _, err := exec.LookPath("npm"); err != nil {
		t.Skip("npm is not installed")
	}
	// The fake test command is a Node script, so it runs the same under sh and cmd.exe.
	for name, out := range map[string]string{
		"node":   `ℹ tests 3\nℹ pass 2\nℹ fail 1`,
		"vitest": ` Tests  1 failed | 2 passed (3)`,
	} {
		dir := t.TempDir()
		js := "process.stdout.write(" + strconv.Quote(out+"\n") + "); process.exit(1);\n"
		_ = os.WriteFile(filepath.Join(dir, "t.js"), []byte(js), 0o644)
		pkg := `{"name":"x","private":true,"scripts":{"test":"node t.js"}}`
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
