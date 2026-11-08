package gitx

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"
)

// TestLiveGitHub exercises the real forge calls Courier uses (P4-02, P4-06) against a GitHub
// repo. It only runs with GBI_LIVE=1 GBI_LIVE_REPO=owner/name GBI_LIVE_BRANCH=branch, from
// a checkout of that repo, because it opens a PR and enables Pages.
func TestLiveGitHub(t *testing.T) {
	if os.Getenv("GBI_LIVE") != "1" {
		t.Skip("set GBI_LIVE=1 to run against GitHub")
	}
	repo, branch := os.Getenv("GBI_LIVE_REPO"), os.Getenv("GBI_LIVE_BRANCH")
	dir, _ := os.Getwd()
	dir += "/../.."
	ctx := context.Background()
	f := NewForge()
	if inst, authed, acct := f.AuthStatus(ctx, "github"); !inst || !authed {
		t.Fatalf("gh not ready (account %q)", acct)
	}
	body := "Opened by go-build-it's Courier (live forge test, P4-02).\n\n🤖 Generated with [Claude Code](https://claude.com/claude-code)"
	url, err := f.CreatePR(ctx, "github", dir, branch, "main", "Build go-build-it through the phased plan", body)
	if err != nil && !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("CreatePR: %v", err)
	}
	t.Logf("PR: %s", strings.TrimSpace(url))
	var info RepoInfo
	for i := 0; i < 30; i++ { // wait for the CI run to appear
		info, err = f.Info(ctx, "github", repo, dir)
		if err != nil {
			t.Fatalf("Info: %v", err)
		}
		if info.CI != "none" {
			break
		}
		time.Sleep(5 * time.Second)
	}
	t.Logf("open PRs %d, issues %d, CI %s at %s", info.OpenPRs, info.OpenIssues, info.CI, info.CIAt)
	if info.OpenPRs < 1 {
		t.Fatal("expected the PR to be counted")
	}
	pages, err := f.EnsurePages(ctx, repo, dir)
	t.Logf("Pages: %q err=%v", pages, err)
	if err != nil && strings.Contains(err.Error(), "plan does not support") {
		t.Log("Pages is not available for this repo on the account's plan (private repo on a free plan); the API call itself worked")
		return
	}
	if err != nil {
		t.Fatalf("EnsurePages: %v", err)
	}
}
