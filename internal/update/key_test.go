package update

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The release public key is printed in the README and baked into both install scripts
// (FR-02b, ADR-012). Rotating it means changing all four together.
func TestPublicKeyEverywhere(t *testing.T) {
	for _, f := range []string{"README.md", "scripts/install.sh", "scripts/install.ps1"} {
		b, err := os.ReadFile(filepath.Join("..", "..", f))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(b), PublicKey) {
			t.Errorf("%s does not carry the release public key %s", f, PublicKey)
		}
	}
}
