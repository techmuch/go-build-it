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
	"sort"
	"strconv"
	"strings"
	"time"
)

// PublicKey verifies SHA256SUMS.sig. The private key lives outside the repo in
// ~/.go-build-it/release-signing.key and is used by `gbi dev sign`. It is a variable only
// so a test build can trust a throwaway key (-ldflags "-X …/update.PublicKey=…", ADR-011);
// nothing changes it at run time.
var PublicKey = "eJ3eKk81oKaKZlNaLjafw1j1GO+Uueyt51jNMCchih4="

// Release is the latest release on the channel.
type Release struct {
	Tag       string      `json:"tag"`
	Current   string      `json:"current"`
	Newer     bool        `json:"newer"`
	AssetName string      `json:"assetName"`
	URL       string      `json:"url"`
	Notes     string      `json:"notes"`
	Previous  string      `json:"previous"`  // version available for rollback
	Releases  []NewerNote `json:"releases"`  // every release newer than Current, newest first (FR-06c)
	CheckedAt time.Time   `json:"checkedAt"` // when this check ran
}

// NewerNote is one release newer than the running version, with its change log entry.
type NewerNote struct {
	Tag   string `json:"tag"`
	Date  string `json:"date"`
	Notes string `json:"notes"`
	URL   string `json:"url"`
}

// Updater talks to GitHub Releases.
type Updater struct {
	Repo    string
	Current string
	Exe     string
	Key     string // public key that release signatures must verify against
	Client  *http.Client
	API     string // override for tests
}

// New returns an updater for the running executable.
func New(repo, current string) *Updater {
	exe, _ := os.Executable()
	if real, err := filepath.EvalSymlinks(exe); err == nil {
		exe = real // replace the binary, not a symlink to it
	}
	return &Updater{Repo: repo, Current: current, Exe: exe, Key: PublicKey, Client: &http.Client{Timeout: 10 * time.Minute}, API: "https://api.github.com"}
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
	TagName     string    `json:"tag_name"`
	Body        string    `json:"body"`
	Draft       bool      `json:"draft"`
	Prerelease  bool      `json:"prerelease"`
	PublishedAt time.Time `json:"published_at"`
	HTMLURL     string    `json:"html_url"`
	Assets      []struct {
		Name string `json:"name"`
		URL  string `json:"browser_download_url"`
	} `json:"assets"`
}

// releases lists the stable, published releases, newest version first.
func (u *Updater) releases(ctx context.Context) ([]ghRelease, error) {
	cctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(cctx, http.MethodGet, u.API+"/repos/"+u.Repo+"/releases?per_page=50", nil)
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := u.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("could not reach GitHub: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, errors.New("no releases published yet")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("releases API: %s", resp.Status)
	}
	var all []ghRelease
	if err := json.NewDecoder(resp.Body).Decode(&all); err != nil {
		return nil, err
	}
	var out []ghRelease
	for _, r := range all {
		if !r.Draft && !r.Prerelease && r.TagName != "" {
			out = append(out, r)
		}
	}
	if len(out) == 0 {
		return nil, errors.New("no releases published yet")
	}
	sort.SliceStable(out, func(i, j int) bool { return Newer(out[i].TagName, out[j].TagName) })
	return out, nil
}

// Check returns the latest release, whether it is newer than the running version, and
// every release in between with its notes (FR-06c).
func (u *Updater) Check(ctx context.Context) (Release, error) {
	rel := Release{Current: u.Current, AssetName: AssetName(), Previous: u.previousVersion(), Releases: []NewerNote{}, CheckedAt: time.Now().UTC()}
	all, err := u.releases(ctx)
	if err != nil {
		return rel, err
	}
	r := all[0]
	rel.Tag, rel.Notes = r.TagName, r.Body
	rel.Newer = u.Current == "dev" || Newer(r.TagName, u.Current)
	for _, a := range r.Assets {
		if a.Name == rel.AssetName {
			rel.URL = a.URL
		}
	}
	for _, x := range all {
		if u.Current != "dev" && !Newer(x.TagName, u.Current) {
			break
		}
		date := ""
		if !x.PublishedAt.IsZero() {
			date = x.PublishedAt.Format("2006-01-02")
		}
		rel.Releases = append(rel.Releases, NewerNote{Tag: x.TagName, Date: date, Notes: x.Body, URL: x.HTMLURL})
		if u.Current == "dev" && len(rel.Releases) == 1 {
			break // a development build is told about the latest release only
		}
	}
	return rel, nil
}

// SameVersion reports whether two version strings name the same release ("v" prefix ignored).
func SameVersion(a, b string) bool { return strings.TrimPrefix(a, "v") == strings.TrimPrefix(b, "v") }

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

// download fetches url. progress, when set, is called with bytes read and the total
// (-1 when the server does not say).
func (u *Updater) download(ctx context.Context, url string, progress func(done, total int64)) ([]byte, error) {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	resp, err := u.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download %s: %s", url, resp.Status)
	}
	var r io.Reader = io.LimitReader(resp.Body, 512<<20)
	if progress != nil {
		r = &countingReader{r: r, total: resp.ContentLength, fn: progress}
	}
	return io.ReadAll(r)
}

type countingReader struct {
	r     io.Reader
	n     int64
	total int64
	fn    func(done, total int64)
}

func (c *countingReader) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	c.n += int64(n)
	c.fn(c.n, c.total)
	return n, err
}

// Verify checks the asset against SHA256SUMS and SHA256SUMS against its Ed25519 signature.
func Verify(asset []byte, name string, sums, sig []byte, pubB64 string) error {
	if err := VerifySignature(sums, sig, pubB64); err != nil {
		return err
	}
	return VerifyChecksum(asset, name, sums)
}

// VerifySignature checks SHA256SUMS against its Ed25519 signature.
func VerifySignature(sums, sig []byte, pubB64 string) error {
	pub, err := base64.StdEncoding.DecodeString(pubB64)
	if err != nil || len(pub) != ed25519.PublicKeySize {
		return errors.New("bad public key")
	}
	rawSig, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(sig)))
	if err != nil {
		return fmt.Errorf("decode signature: %w", err)
	}
	if !ed25519.Verify(ed25519.PublicKey(pub), sums, rawSig) {
		return errors.New("signature verification failed: SHA256SUMS was not signed by the go-build-it release key")
	}
	return nil
}

// VerifyChecksum checks the asset's SHA-256 against its line in SHA256SUMS.
func VerifyChecksum(asset []byte, name string, sums []byte) error {
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
		return errors.New("checksum mismatch: the download does not match SHA256SUMS")
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

// install moves the tested binary at next into place, keeping the running one as .prev
// with its version, for rollback (FR-06).
func (u *Updater) install(next string) error {
	prev := u.Exe + ".prev"
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
