// Package update checks for, verifies, installs and rolls back releases (FR-06, ADR-004).
package update

import (
	"bufio"
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// PublicKey verifies SHA256SUMS.sig. The private key lives outside the repo in
// ~/.go-build-it/release-signing.key and is used by `gbi dev sign`.
const PublicKey = "eJ3eKk81oKaKZlNaLjafw1j1GO+Uueyt51jNMCchih4="

// Release is the latest release on the channel.
type Release struct {
	Tag       string `json:"tag"`
	Current   string `json:"current"`
	Newer     bool   `json:"newer"`
	AssetName string `json:"assetName"`
	URL       string `json:"url"`
	Notes     string `json:"notes"`
	Previous  string `json:"previous"` // version available for rollback
}

// Updater talks to GitHub Releases.
type Updater struct {
	Repo    string
	Current string
	Exe     string
	Client  *http.Client
	API     string // override for tests
}

// New returns an updater for the running executable.
func New(repo, current string) *Updater {
	exe, _ := os.Executable()
	return &Updater{Repo: repo, Current: current, Exe: exe, Client: &http.Client{Timeout: 60 * time.Second}, API: "https://api.github.com"}
}

// AssetName is the release asset for this platform.
func AssetName() string {
	name := fmt.Sprintf("gbi_%s_%s", runtime.GOOS, runtime.GOARCH)
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	return name
}

type ghRelease struct {
	TagName string `json:"tag_name"`
	Body    string `json:"body"`
	Assets  []struct {
		Name string `json:"name"`
		URL  string `json:"browser_download_url"`
	} `json:"assets"`
}

func (u *Updater) latest(ctx context.Context) (ghRelease, error) {
	var r ghRelease
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, u.API+"/repos/"+u.Repo+"/releases/latest", nil)
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := u.Client.Do(req)
	if err != nil {
		return r, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return r, errors.New("no releases published yet")
	}
	if resp.StatusCode != http.StatusOK {
		return r, fmt.Errorf("releases API: %s", resp.Status)
	}
	return r, json.NewDecoder(resp.Body).Decode(&r)
}

// Check returns the latest release and whether it is newer than the running version.
func (u *Updater) Check(ctx context.Context) (Release, error) {
	rel := Release{Current: u.Current, AssetName: AssetName(), Previous: u.previousVersion()}
	r, err := u.latest(ctx)
	if err != nil {
		return rel, err
	}
	rel.Tag, rel.Notes = r.TagName, r.Body
	rel.Newer = u.Current == "dev" || Newer(r.TagName, u.Current)
	for _, a := range r.Assets {
		if a.Name == rel.AssetName {
			rel.URL = a.URL
		}
	}
	return rel, nil
}

// Newer compares CalVer strings YYYY.MM.DD.N.
func Newer(a, b string) bool {
	pa, pb := strings.Split(strings.TrimPrefix(a, "v"), "."), strings.Split(strings.TrimPrefix(b, "v"), ".")
	for i := 0; i < len(pa) && i < len(pb); i++ {
		x, _ := strconv.Atoi(pa[i])
		y, _ := strconv.Atoi(pb[i]) // decimal: "08" must not be read as octal
		if x != y {
			return x > y
		}
	}
	return len(pa) > len(pb)
}

func (u *Updater) download(ctx context.Context, url string) ([]byte, error) {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	resp, err := u.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download %s: %s", url, resp.Status)
	}
	return io.ReadAll(io.LimitReader(resp.Body, 512<<20))
}

// Verify checks the asset against SHA256SUMS and SHA256SUMS against its Ed25519 signature.
func Verify(asset []byte, name string, sums, sig []byte, pubB64 string) error {
	pub, err := base64.StdEncoding.DecodeString(pubB64)
	if err != nil || len(pub) != ed25519.PublicKeySize {
		return errors.New("bad public key")
	}
	rawSig, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(sig)))
	if err != nil {
		return fmt.Errorf("decode signature: %w", err)
	}
	if !ed25519.Verify(ed25519.PublicKey(pub), sums, rawSig) {
		return errors.New("signature verification failed")
	}
	h := sha256.Sum256(asset)
	want := ""
	sc := bufio.NewScanner(strings.NewReader(string(sums)))
	for sc.Scan() {
		f := strings.Fields(sc.Text())
		if len(f) == 2 && strings.TrimPrefix(f[1], "*") == name {
			want = f[0]
		}
	}
	if want == "" {
		return fmt.Errorf("%s is not listed in SHA256SUMS", name)
	}
	if want != hex.EncodeToString(h[:]) {
		return errors.New("checksum mismatch")
	}
	return nil
}

// Sign produces SHA256SUMS content and its signature for the given files (used by `gbi dev sign`).
func Sign(files []string, privB64 string) (sums, sig []byte, err error) {
	priv, err := base64.StdEncoding.DecodeString(strings.TrimSpace(privB64))
	if err != nil || len(priv) != ed25519.PrivateKeySize {
		return nil, nil, errors.New("bad private key")
	}
	var b strings.Builder
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			return nil, nil, err
		}
		h := sha256.Sum256(data)
		fmt.Fprintf(&b, "%s  %s\n", hex.EncodeToString(h[:]), filepath.Base(f))
	}
	sums = []byte(b.String())
	sig = []byte(base64.StdEncoding.EncodeToString(ed25519.Sign(ed25519.PrivateKey(priv), sums)))
	return sums, sig, nil
}

// Install downloads, verifies and swaps in the new binary, keeping the old one for rollback.
// Callers must only call it when no sessions are running (FR-06).
func (u *Updater) Install(ctx context.Context) (Release, error) {
	rel, err := u.Check(ctx)
	if err != nil {
		return rel, err
	}
	if !rel.Newer {
		return rel, errors.New("already up to date")
	}
	if rel.URL == "" {
		return rel, fmt.Errorf("release %s has no asset %s", rel.Tag, rel.AssetName)
	}
	base := rel.URL[:strings.LastIndex(rel.URL, "/")+1]
	asset, err := u.download(ctx, rel.URL)
	if err != nil {
		return rel, err
	}
	sums, err := u.download(ctx, base+"SHA256SUMS")
	if err != nil {
		return rel, err
	}
	sig, err := u.download(ctx, base+"SHA256SUMS.sig")
	if err != nil {
		return rel, err
	}
	if err := Verify(asset, rel.AssetName, sums, sig, PublicKey); err != nil {
		return rel, err
	}
	return rel, u.swap(asset, rel.Tag)
}

func (u *Updater) swap(asset []byte, tag string) error {
	next, prev := u.Exe+".new", u.Exe+".prev"
	if err := os.WriteFile(next, asset, 0o755); err != nil {
		return err
	}
	_ = os.Remove(prev)
	if err := os.Rename(u.Exe, prev); err != nil {
		return err
	}
	if err := os.Rename(next, u.Exe); err != nil {
		_ = os.Rename(prev, u.Exe)
		return err
	}
	_ = os.WriteFile(prev+".version", []byte(u.Current), 0o644)
	return nil
}

func (u *Updater) previousVersion() string {
	b, err := os.ReadFile(u.Exe + ".prev.version")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

// Rollback restores the previous binary.
func (u *Updater) Rollback() (string, error) {
	prev := u.Exe + ".prev"
	if _, err := os.Stat(prev); err != nil {
		return "", errors.New("no previous version to roll back to")
	}
	v := u.previousVersion()
	tmp := u.Exe + ".rollback"
	if err := os.Rename(u.Exe, tmp); err != nil {
		return "", err
	}
	if err := os.Rename(prev, u.Exe); err != nil {
		_ = os.Rename(tmp, u.Exe)
		return "", err
	}
	_ = os.Rename(tmp, prev)
	_ = os.WriteFile(prev+".version", []byte(u.Current), 0o644)
	return v, nil
}
