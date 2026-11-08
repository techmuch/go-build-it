// Package gitx wraps the git, gh and glab command lines. These are the deterministic
// operations Courier runs without an LLM (FR-92, FR-93).
package gitx

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Runner runs external commands; tests replace it.
type Runner func(ctx context.Context, dir, name string, args ...string) (string, error)

// Exec is the default Runner.
func Exec(ctx context.Context, dir, name string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb
	err := cmd.Run()
	if err != nil {
		return strings.TrimSpace(out.String()), fmt.Errorf("%s %s: %w: %s", name, strings.Join(args, " "), err, strings.TrimSpace(errb.String()))
	}
	return strings.TrimSpace(out.String()), nil
}

// Git is a git client bound to a runner.
type Git struct{ Run Runner }

// New returns a Git using the real command line.
func New() *Git { return &Git{Run: Exec} }

func (g *Git) git(ctx context.Context, dir string, args ...string) (string, error) {
	return g.Run(ctx, dir, "git", args...)
}

// IsRepo reports whether dir is a git work tree.
func (g *Git) IsRepo(ctx context.Context, dir string) bool {
	out, err := g.git(ctx, dir, "rev-parse", "--is-inside-work-tree")
	return err == nil && out == "true"
}

// Remote returns the origin URL ("" if none).
func (g *Git) Remote(ctx context.Context, dir string) string {
	out, _ := g.git(ctx, dir, "remote", "get-url", "origin")
	return out
}

// DefaultBranch returns the default branch name (main/master/…).
func (g *Git) DefaultBranch(ctx context.Context, dir string) string {
	if out, err := g.git(ctx, dir, "symbolic-ref", "--short", "refs/remotes/origin/HEAD"); err == nil {
		return strings.TrimPrefix(out, "origin/")
	}
	if out, err := g.git(ctx, dir, "branch", "--show-current"); err == nil && out != "" {
		return out
	}
	return "main"
}

// Forge identifies the hosting forge from a remote URL (FR-90).
func Forge(remote string) string {
	switch {
	case strings.Contains(remote, "github.com"):
		return "github"
	case strings.Contains(remote, "gitlab"):
		return "gitlab"
	case remote == "":
		return ""
	}
	return "other"
}

var slugRe = regexp.MustCompile(`[:/]([^/:]+/[^/]+?)(\.git)?$`)

// RepoSlug returns owner/name from a remote URL.
func RepoSlug(remote string) string {
	m := slugRe.FindStringSubmatch(remote)
	if m == nil {
		return ""
	}
	return m[1]
}

// Unpushed describes work that is not on the remote (FR-94).
type Unpushed struct {
	Commits     int      `json:"commits"`     // commits on local branches not on any remote
	Branches    []string `json:"branches"`    // local branches with unpushed commits
	Uncommitted int      `json:"uncommitted"` // changed files in the main work tree
}

// Any reports whether there is anything unpushed.
func (u Unpushed) Any() bool { return u.Commits > 0 || u.Uncommitted > 0 }

// FindUnpushed counts unpushed commits across local branches and uncommitted changes.
func (g *Git) FindUnpushed(ctx context.Context, dir string) Unpushed {
	var u Unpushed
	if out, err := g.git(ctx, dir, "status", "--porcelain"); err == nil && out != "" {
		u.Uncommitted = len(strings.Split(out, "\n"))
	}
	out, err := g.git(ctx, dir, "for-each-ref", "--format=%(refname:short)", "refs/heads")
	if err != nil {
		return u
	}
	for _, br := range strings.Fields(out) {
		n, err := g.git(ctx, dir, "rev-list", "--count", br, "--not", "--remotes")
		if err != nil {
			continue
		}
		if c, _ := strconv.Atoi(n); c > 0 {
			u.Commits += c
			u.Branches = append(u.Branches, br)
		}
	}
	return u
}

// calverRe matches CalVer release tags, YYYY.MM.DD.N (FR-05). Month and day may or may not be
// zero-padded (2026.11.08.1 and v2026.11.8.1 are both releases).
var calverRe = regexp.MustCompile(`^v?(\d{4})\.(\d{1,2})\.(\d{1,2})\.(\d+)$`)

// semverRe matches stable SemVer tags (no pre-release or build suffix).
var semverRe = regexp.MustCompile(`^v?(\d+)\.(\d+)\.(\d+)$`)

// Release is a tagged release.
type Release struct {
	Tag     string `json:"tag"`
	Date    string `json:"date"`
	Commits int    `json:"commitsSince"`
}

// LatestTag picks the newest stable release tag: the highest CalVer tag, or
// the highest SemVer tag when the repository has no CalVer tags.
func LatestTag(tags []string) string {
	best := func(re *regexp.Regexp) string {
		var top string
		for _, t := range tags {
			if re.MatchString(t) && (top == "" || versionLess(re, top, t)) {
				top = t
			}
		}
		return top
	}
	if t := best(calverRe); t != "" {
		return t
	}
	return best(semverRe)
}

// LatestRelease finds the newest stable release tag and counts commits since it.
func (g *Git) LatestRelease(ctx context.Context, dir string) *Release {
	tag := LatestTag(g.Tags(ctx, dir))
	if tag == "" {
		return nil
	}
	return g.ReleaseAt(ctx, dir, tag)
}

// ReleaseAt describes a known release tag: its date and the commits since it.
// The tag may be missing locally (not fetched yet); then only Tag is set.
func (g *Git) ReleaseAt(ctx context.Context, dir, tag string) *Release {
	r := &Release{Tag: tag}
	if d, err := g.git(ctx, dir, "log", "-1", "--format=%cs", tag); err == nil {
		r.Date = d
	}
	if n, err := g.git(ctx, dir, "rev-list", "--count", tag+"..HEAD"); err == nil {
		r.Commits, _ = strconv.Atoi(n)
	}
	return r
}

// versionLess compares two tags matched by re, numerically part by part.
func versionLess(re *regexp.Regexp, a, b string) bool {
	pa, pb := re.FindStringSubmatch(a), re.FindStringSubmatch(b)
	for i := 1; i < len(pa); i++ {
		x, _ := strconv.Atoi(pa[i])
		y, _ := strconv.Atoi(pb[i])
		if x != y {
			return x < y
		}
	}
	return false
}

// NextCalVer returns the next CalVer for today given existing tags (FR-05).
// Existing tags count for today whether or not they zero-pad month and day.
func NextCalVer(today time.Time, existing []string) string {
	n := 0
	for _, t := range existing {
		m := calverRe.FindStringSubmatch(t)
		if m == nil {
			continue
		}
		y, _ := strconv.Atoi(m[1])
		mo, _ := strconv.Atoi(m[2])
		d, _ := strconv.Atoi(m[3])
		if y == today.Year() && mo == int(today.Month()) && d == today.Day() {
			if v, _ := strconv.Atoi(m[4]); v > n {
				n = v
			}
		}
	}
	return fmt.Sprintf("%s.%d", today.Format("2006.01.02"), n+1)
}

// Tags lists all tags.
func (g *Git) Tags(ctx context.Context, dir string) []string {
	out, _ := g.git(ctx, dir, "tag", "--list")
	return strings.Fields(out)
}

// Revision is one commit touching a file.
type Revision struct {
	Hash    string `json:"hash"`
	Author  string `json:"author"`
	Date    string `json:"date"`
	Subject string `json:"subject"`
}

// FileHistory lists commits that touched path (artifact revisions, FR-46).
func (g *Git) FileHistory(ctx context.Context, dir, path string, limit int) []Revision {
	out, err := g.git(ctx, dir, "log", fmt.Sprintf("-%d", limit), "--format=%h\x1f%an\x1f%cs\x1f%s", "--", path)
	if err != nil || out == "" {
		return []Revision{}
	}
	var revs []Revision
	for _, l := range strings.Split(out, "\n") {
		f := strings.Split(l, "\x1f")
		if len(f) == 4 {
			revs = append(revs, Revision{f[0], f[1], f[2], f[3]})
		}
	}
	return revs
}

// Show returns a file's content at a revision.
func (g *Git) Show(ctx context.Context, dir, rev, path string) (string, error) {
	return g.git(ctx, dir, "show", rev+":"+path)
}

// CurrentBranch returns the checked-out branch.
func (g *Git) CurrentBranch(ctx context.Context, dir string) string {
	out, _ := g.git(ctx, dir, "branch", "--show-current")
	return out
}

// AddWorktree creates a worktree at path on a new branch from base (FR-64).
func (g *Git) AddWorktree(ctx context.Context, repo, path, branch, base string) error {
	if _, err := g.git(ctx, repo, "worktree", "add", "-B", branch, path, base); err != nil {
		return err
	}
	return nil
}

// RemoveWorktree removes a worktree.
func (g *Git) RemoveWorktree(ctx context.Context, repo, path string) error {
	_, err := g.git(ctx, repo, "worktree", "remove", "--force", path)
	return err
}

// CommitAll stages everything and commits; returns false if there was nothing to commit.
func (g *Git) CommitAll(ctx context.Context, dir, msg, author string) (bool, error) {
	if _, err := g.git(ctx, dir, "add", "-A"); err != nil {
		return false, err
	}
	if out, _ := g.git(ctx, dir, "status", "--porcelain"); out == "" {
		return false, nil
	}
	args := []string{"commit", "-m", msg}
	if author != "" {
		args = append([]string{"-c", "user.name=" + author, "-c", "user.email=" + strings.ToLower(author) + "@go-build-it.local"}, args...)
	}
	_, err := g.git(ctx, dir, args...)
	return err == nil, err
}

// Push pushes a branch to origin and sets upstream.
func (g *Git) Push(ctx context.Context, dir, branch string) error {
	_, err := g.git(ctx, dir, "push", "-u", "origin", branch)
	return err
}

// Tag creates an annotated tag and pushes it.
func (g *Git) Tag(ctx context.Context, dir, tag, msg string, push bool) error {
	if _, err := g.git(ctx, dir, "tag", "-a", tag, "-m", msg); err != nil {
		return err
	}
	if push {
		_, err := g.git(ctx, dir, "push", "origin", tag)
		return err
	}
	return nil
}

// Log returns commit subjects in a range (for release notes).
func (g *Git) Log(ctx context.Context, dir, rng string) []string {
	out, err := g.git(ctx, dir, "log", "--format=%s", rng)
	if err != nil || out == "" {
		return nil
	}
	return strings.Split(out, "\n")
}

// Merge merges branch into the current branch; returns conflicted files on failure.
func (g *Git) Merge(ctx context.Context, dir, branch string, squash bool) ([]string, error) {
	args := []string{"merge", "--no-edit", branch}
	if squash {
		args = []string{"merge", "--squash", branch}
	}
	if _, err := g.git(ctx, dir, args...); err != nil {
		out, _ := g.git(ctx, dir, "diff", "--name-only", "--diff-filter=U")
		return strings.Fields(out), err
	}
	return nil, nil
}

// Forge CLI ------------------------------------------------------------------

// ForgeCLI runs gh or glab.
type ForgeCLI struct{ Run Runner }

// NewForge returns a ForgeCLI using the real command line.
func NewForge() *ForgeCLI { return &ForgeCLI{Run: Exec} }

func cliFor(forge string) string {
	if forge == "gitlab" {
		return "glab"
	}
	return "gh"
}

// AuthStatus reports whether the forge CLI is installed and authenticated (FR-90).
func (f *ForgeCLI) AuthStatus(ctx context.Context, forge string) (installed, authed bool, account string) {
	bin := cliFor(forge)
	if _, err := exec.LookPath(bin); err != nil {
		return false, false, ""
	}
	out, err := f.Run(ctx, "", bin, "auth", "status")
	if err != nil {
		return true, false, ""
	}
	if m := regexp.MustCompile(`(?i)account (\S+)|as (\S+)`).FindStringSubmatch(out); m != nil {
		account = m[1] + m[2]
	}
	return true, true, account
}

// RepoInfo is forge metadata for the catalog card (FR-41).
type RepoInfo struct {
	OpenPRs     int    `json:"openPrs"`
	OpenIssues  int    `json:"openIssues"`
	CI          string `json:"ci"` // passing | failing | pending | none
	CIAt        string `json:"ciAt"`
	WebsiteURL  string `json:"websiteUrl"`
	ReleaseURL  string `json:"releaseUrl"`
	LatestTag   string `json:"latestTag"` // the forge's "Latest" release, if any
	RepoURL     string `json:"repoUrl"`
	Description string `json:"description"`
}

// Info fetches PR/issue counts, CI status and Pages URL via gh (GitHub only for now).
func (f *ForgeCLI) Info(ctx context.Context, forge, slug, dir string) (RepoInfo, error) {
	var ri RepoInfo
	if forge != "github" || slug == "" {
		return ri, errors.New("forge info only available for GitHub")
	}
	ri.RepoURL = "https://github.com/" + slug
	if out, err := f.Run(ctx, dir, "gh", "repo", "view", slug, "--json", "description,homepageUrl"); err == nil {
		var v struct{ Description, HomepageUrl string }
		_ = json.Unmarshal([]byte(out), &v)
		ri.Description, ri.WebsiteURL = v.Description, v.HomepageUrl
	}
	if out, err := f.Run(ctx, dir, "gh", "api", "repos/"+slug+"/pages", "--jq", ".html_url"); err == nil && out != "" {
		ri.WebsiteURL = out
	}
	// gh's "latest" is the newest published, non-draft, non-prerelease release.
	if out, err := f.Run(ctx, dir, "gh", "release", "view", "-R", slug, "--json", "tagName", "--jq", ".tagName"); err == nil {
		ri.LatestTag = strings.TrimSpace(out)
	}
	if out, err := f.Run(ctx, dir, "gh", "pr", "list", "-R", slug, "--state", "open", "--json", "number", "--jq", "length"); err == nil {
		ri.OpenPRs, _ = strconv.Atoi(out)
	}
	if out, err := f.Run(ctx, dir, "gh", "issue", "list", "-R", slug, "--state", "open", "--json", "number", "--jq", "length"); err == nil {
		ri.OpenIssues, _ = strconv.Atoi(out)
	}
	ri.CI = "none"
	if out, err := f.Run(ctx, dir, "gh", "run", "list", "-R", slug, "--limit", "1", "--json", "conclusion,status,updatedAt"); err == nil {
		var runs []struct{ Conclusion, Status, UpdatedAt string }
		if json.Unmarshal([]byte(out), &runs) == nil && len(runs) > 0 {
			ri.CIAt = runs[0].UpdatedAt
			switch {
			case runs[0].Status != "completed":
				ri.CI = "pending"
			case runs[0].Conclusion == "success":
				ri.CI = "passing"
			default:
				ri.CI = "failing"
			}
		}
	}
	return ri, nil
}

// CreatePR opens a pull/merge request and returns its URL.
func (f *ForgeCLI) CreatePR(ctx context.Context, forge, dir, branch, base, title, body string) (string, error) {
	if forge == "gitlab" {
		return f.Run(ctx, dir, "glab", "mr", "create", "--source-branch", branch, "--target-branch", base, "--title", title, "--description", body, "--yes")
	}
	return f.Run(ctx, dir, "gh", "pr", "create", "--head", branch, "--base", base, "--title", title, "--body", body)
}

// MergePR squash-merges a PR/MR.
func (f *ForgeCLI) MergePR(ctx context.Context, forge, dir, pr string) error {
	var err error
	if forge == "gitlab" {
		_, err = f.Run(ctx, dir, "glab", "mr", "merge", pr, "--squash", "--remove-source-branch", "--yes")
	} else {
		_, err = f.Run(ctx, dir, "gh", "pr", "merge", pr, "--squash", "--delete-branch")
	}
	return err
}

// CreateRelease creates a forge release for a tag with notes.
func (f *ForgeCLI) CreateRelease(ctx context.Context, forge, dir, tag, notes string) (string, error) {
	if forge == "gitlab" {
		return f.Run(ctx, dir, "glab", "release", "create", tag, "--notes", notes)
	}
	return f.Run(ctx, dir, "gh", "release", "create", tag, "--title", tag, "--notes", notes)
}

// EnsurePages enables GitHub Pages from the gh-pages branch or docs/ folder and returns the URL (FR-44).
func (f *ForgeCLI) EnsurePages(ctx context.Context, slug, dir string) (string, error) {
	if out, err := f.Run(ctx, dir, "gh", "api", "repos/"+slug+"/pages", "--jq", ".html_url"); err == nil && out != "" {
		return out, nil
	}
	_, err := f.Run(ctx, dir, "gh", "api", "-X", "POST", "repos/"+slug+"/pages", "-f", "source[branch]=main", "-f", "source[path]=/docs")
	if err != nil {
		return "", err
	}
	owner, name, _ := strings.Cut(slug, "/")
	return fmt.Sprintf("https://%s.github.io/%s/", owner, name), nil
}

// RepoName returns the base name of a path, used as a default project id.
func RepoName(path string) string { return filepath.Base(filepath.Clean(path)) }
