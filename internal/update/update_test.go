package update

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"
)

func TestSignVerifyAndTamper(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	dir := t.TempDir()
	f := filepath.Join(dir, AssetName())
	os.WriteFile(f, []byte("binary"), 0o644)
	sums, sig, err := Sign([]string{f}, base64.StdEncoding.EncodeToString(priv))
	if err != nil {
		t.Fatal(err)
	}
	pk := base64.StdEncoding.EncodeToString(pub)
	if err := Verify([]byte("binary"), AssetName(), sums, sig, pk); err != nil {
		t.Fatalf("valid release rejected: %v", err)
	}
	if err := Verify([]byte("evil"), AssetName(), sums, sig, pk); err == nil {
		t.Fatal("tampered binary accepted")
	}
	bad := append([]byte(nil), sums...)
	bad[0] ^= 1
	if err := Verify([]byte("binary"), AssetName(), bad, sig, pk); err == nil {
		t.Fatal("tampered SHA256SUMS accepted")
	}
	if err := Verify([]byte("binary"), AssetName(), sums, sig, PublicKey); err == nil {
		t.Fatal("signature from another key accepted")
	}
}

func TestNewer(t *testing.T) {
	if !Newer("2026.11.08.2", "2026.11.08.1") || Newer("2026.11.07.9", "2026.11.08.1") || !Newer("2026.12.01.1", "2026.11.30.4") {
		t.Fatal("CalVer comparison")
	}
}

func TestSwapAndRollback(t *testing.T) {
	dir := t.TempDir()
	exe := filepath.Join(dir, "gbi")
	os.WriteFile(exe, []byte("old"), 0o755)
	u := &Updater{Exe: exe, Current: "2026.11.01.1"}
	if err := u.swap([]byte("new"), "2026.11.08.1"); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(exe); string(b) != "new" {
		t.Fatal("swap failed")
	}
	v, err := u.Rollback()
	if err != nil || v != "2026.11.01.1" {
		t.Fatalf("rollback %q %v", v, err)
	}
	if b, _ := os.ReadFile(exe); string(b) != "old" {
		t.Fatal("rollback did not restore old binary")
	}
}
