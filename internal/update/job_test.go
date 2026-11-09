package update

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

// fakeRelease serves a GitHub-like releases API with signed assets (ADR-011 test seam).
type fakeRelease struct {
	srv      *httptest.Server
	pub      string
	assets   map[string][]byte
	releases []map[string]any
}

func script(version string) []byte {
	return []byte("#!/bin/sh\necho " + version + "\n")
}

func newFakeRelease(t *testing.T, tags []string, binVersion string) *fakeRelease {
	t.Helper()
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	f := &fakeRelease{pub: base64.StdEncoding.EncodeToString(pub), assets: map[string][]byte{}}
	dir := t.TempDir()
	asset := filepath.Join(dir, AssetName())
	os.WriteFile(asset, script(binVersion), 0o755)
	sums, sig, err := Sign([]string{asset}, base64.StdEncoding.EncodeToString(priv))
	if err != nil {
		t.Fatal(err)
	}
	f.assets[AssetName()], f.assets["SHA256SUMS"], f.assets["SHA256SUMS.sig"] = script(binVersion), sums, sig
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/o/r/releases", func(w http.ResponseWriter, r *http.Request) { json.NewEncoder(w).Encode(f.releases) })
	mux.HandleFunc("/dl/", func(w http.ResponseWriter, r *http.Request) {
		b, ok := f.assets[strings.TrimPrefix(r.URL.Path, "/dl/")]
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Length", fmt.Sprint(len(b)))
		w.Write(b)
	})
	f.srv = httptest.NewServer(mux)
	t.Cleanup(f.srv.Close)
	for _, tag := range tags {
		f.releases = append(f.releases, map[string]any{"tag_name": tag, "body": "### Added\n\n- things in " + tag, "published_at": "2026-12-01T10:00:00Z",
			"assets": []map[string]string{{"name": AssetName(), "browser_download_url": f.srv.URL + "/dl/" + AssetName()}}})
	}
	f.releases = append(f.releases, map[string]any{"tag_name": "2099.01.01.1", "prerelease": true, "body": "beta"}, map[string]any{"tag_name": "2099.01.02.1", "draft": true})
	return f
}

type harness struct {
	exe     string
	path    string
	u       *Updater
	jobs    *Jobs
	mu      sync.Mutex
	events  []string
	restart chan struct{}
}

func newHarness(t *testing.T, f *fakeRelease, current string) *harness {
	if runtime.GOOS == "windows" {
		t.Skip("uses shell-script binaries")
	}
	dir := t.TempDir()
	h := &harness{exe: filepath.Join(dir, "gbi"), path: filepath.Join(dir, "update", "job.json"), restart: make(chan struct{}, 1)}
	os.WriteFile(h.exe, script(current), 0o755)
	h.u = &Updater{Repo: "o/r", Current: current, Exe: h.exe, Key: f.pub, Client: http.DefaultClient, API: f.srv.URL}
	h.jobs = h.newJobs(current)
	return h
}

// newJobs simulates a (re)started go-build-it reading the saved job.
func (h *harness) newJobs(version string) *Jobs {
	u := *h.u
	u.Current = version
	j := NewJobs(&u, h.path)
	j.RestartDelay = 0
	j.Restart = func() { h.restart <- struct{}{} }
	j.OnEvent = func(kind string, _ Job) { h.mu.Lock(); h.events = append(h.events, kind); h.mu.Unlock() }
	return j
}

func wait(t *testing.T, j *Jobs, status ...string) Job {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if job, ok := j.Current(); ok {
			for _, s := range status {
				if job.Status == s {
					return job
				}
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	job, _ := j.Current()
	t.Fatalf("job never reached %v: %+v", status, job)
	return job
}

func TestCheckListsEveryNewerRelease(t *testing.T) {
	f := newFakeRelease(t, []string{"2026.12.01.1", "2026.12.03.1", "2026.11.20.1", "2026.11.01.1"}, "2026.12.03.1")
	u := &Updater{Repo: "o/r", Current: "2026.11.15.1", Exe: "x", Key: f.pub, Client: http.DefaultClient, API: f.srv.URL}
	rel, err := u.Check(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	var tags []string
	for _, r := range rel.Releases {
		tags = append(tags, r.Tag)
	}
	if rel.Tag != "2026.12.03.1" || !rel.Newer || strings.Join(tags, ",") != "2026.12.03.1,2026.12.01.1,2026.11.20.1" || rel.CheckedAt.IsZero() {
		t.Fatalf("check: tag %s newer %v releases %v", rel.Tag, rel.Newer, tags)
	}
	if !strings.Contains(rel.Releases[0].Notes, "things in 2026.12.03.1") || rel.Releases[0].Date != "2026-12-01" {
		t.Fatalf("notes: %+v", rel.Releases[0])
	}
	u.Current = "2026.12.03.1"
	if rel, _ := u.Check(context.Background()); rel.Newer || len(rel.Releases) != 0 {
		t.Fatalf("up to date: %+v", rel)
	}
}

func TestUpdateRunsEveryStepAcrossTheRestart(t *testing.T) {
	f := newFakeRelease(t, []string{"2026.12.01.1"}, "2026.12.01.1")
	h := newHarness(t, f, "2026.11.07.1")
	if _, err := h.jobs.Start("manual", "david"); err != nil {
		t.Fatal(err)
	}
	if _, err := h.jobs.Start("manual", "david"); !errors.Is(err, ErrBusy) {
		t.Fatalf("second start: %v", err)
	}
	<-h.restart
	job := wait(t, h.jobs, JobRestarting)
	for _, s := range job.Steps[:6] {
		if s.Status != Done || s.StartedAt == nil || s.EndedAt == nil {
			t.Fatalf("step %s: %+v", s.ID, s)
		}
	}
	if job.Step(StepRestart).Status != Running || job.To != "2026.12.01.1" || job.Step(StepDownload).Total == 0 {
		t.Fatalf("restarting job: %+v", job)
	}
	if b, _ := os.ReadFile(h.exe); !strings.Contains(string(b), "2026.12.01.1") {
		t.Fatal("new binary not installed")
	}
	if b, _ := os.ReadFile(h.exe + ".prev.version"); string(b) != "2026.11.07.1" {
		t.Fatalf("previous version not kept: %q", b)
	}
	if _, err := os.Stat(h.exe + ".new"); err == nil {
		t.Fatal(".new left behind")
	}

	// The restarted process (new version) finishes the job from the saved file.
	next := h.newJobs("2026.12.01.1")
	if !next.Busy() {
		t.Fatal("saved job not loaded as in progress")
	}
	next.Resume(context.Background(), "2026.12.01.1", func(context.Context) (string, error) { return "herdr session gbi · 2 members working", nil })
	job, _ = next.Current()
	if job.Status != JobDone || job.Step(StepRestart).Status != Done || job.Step(StepReconnect).Detail != "herdr session gbi · 2 members working" || job.EndedAt == nil {
		t.Fatalf("finished job: %+v", job)
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if strings.Join(h.events, ",") != "started,restarting,completed" {
		t.Fatalf("events: %v", h.events)
	}
}

func TestFailuresBeforeInstallChangeNothing(t *testing.T) {
	cases := map[string]struct {
		step   string
		mutate func(f *fakeRelease, h *harness)
		want   string
	}{
		"tampered download": {StepChecksum, func(f *fakeRelease, h *harness) { f.assets[AssetName()] = script("evil") }, "checksum mismatch"},
		"other key":         {StepSignature, func(f *fakeRelease, h *harness) { h.jobs.U.Key = PublicKey }, "signature verification failed"},
		"wrong version":     {StepTest, nil, "reports \"2026.11.30.1\""},
		"missing sums":      {StepSignature, func(f *fakeRelease, h *harness) { delete(f.assets, "SHA256SUMS") }, "SHA256SUMS"},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			bin := "2026.12.01.1"
			if name == "wrong version" {
				bin = "2026.11.30.1"
			}
			f := newFakeRelease(t, []string{"2026.12.01.1"}, bin)
			h := newHarness(t, f, "2026.11.07.1")
			if c.mutate != nil {
				c.mutate(f, h)
			}
			h.jobs.Start("manual", "david")
			job := wait(t, h.jobs, JobFailed)
			s := job.Step(c.step)
			if s.Status != Failed || !strings.Contains(s.Error, c.want) || !job.CanRetry || job.CanRollback {
				t.Fatalf("job: %+v", job)
			}
			if job.Step(StepInstall).Status != Skipped {
				t.Fatalf("install should be skipped: %+v", job.Step(StepInstall))
			}
			if b, _ := os.ReadFile(h.exe); !strings.Contains(string(b), "2026.11.07.1") {
				t.Fatal("running binary was changed")
			}
			if _, err := os.Stat(h.exe + ".new"); err == nil {
				t.Fatal(".new left behind")
			}
			// Try again starts a fresh job.
			if _, err := h.jobs.Start("manual", "david"); err != nil {
				t.Fatalf("retry: %v", err)
			}
			wait(t, h.jobs, JobFailed, JobRestarting)
		})
	}
}

func TestUpToDateFailsAtFind(t *testing.T) {
	f := newFakeRelease(t, []string{"2026.12.01.1"}, "2026.12.01.1")
	h := newHarness(t, f, "2026.12.01.1")
	h.jobs.Start("manual", "")
	job := wait(t, h.jobs, JobFailed)
	if job.Step(StepFind).Status != Failed || !strings.Contains(job.Error, "already up to date") {
		t.Fatalf("%+v", job)
	}
}

func TestResumeWithTheWrongVersionOffersRollback(t *testing.T) {
	f := newFakeRelease(t, []string{"2026.12.01.1"}, "2026.12.01.1")
	h := newHarness(t, f, "2026.11.07.1")
	h.jobs.Start("manual", "")
	<-h.restart
	next := h.newJobs("2026.11.07.1")
	next.Resume(context.Background(), "2026.11.07.1", func(context.Context) (string, error) { return "", nil })
	job, _ := next.Current()
	if job.Status != JobFailed || job.Step(StepRestart).Status != Failed || !job.CanRollback || job.CanRetry {
		t.Fatalf("%+v", job)
	}
}

func TestInterruptedJobIsMarkedFailed(t *testing.T) {
	f := newFakeRelease(t, []string{"2026.12.01.1"}, "2026.12.01.1")
	h := newHarness(t, f, "2026.11.07.1")
	now := time.Now()
	saved := Job{ID: "U-1", Kind: "update", Status: JobRunning, From: "2026.11.07.1", Steps: updateSteps(AssetName())}
	saved.Steps[1].Status, saved.Steps[1].StartedAt = Running, &now
	b, _ := json.Marshal(saved)
	os.MkdirAll(filepath.Dir(h.path), 0o755)
	os.WriteFile(h.path, b, 0o644)
	j := h.newJobs("2026.11.07.1")
	j.Resume(context.Background(), "2026.11.07.1", nil)
	job, _ := j.Current()
	if job.Status != JobFailed || job.Step(StepDownload).Status != Failed || !strings.Contains(job.Error, "interrupted") || !job.CanRetry {
		t.Fatalf("%+v", job)
	}
}

func TestRollbackJob(t *testing.T) {
	f := newFakeRelease(t, []string{"2026.12.01.1"}, "2026.12.01.1")
	h := newHarness(t, f, "2026.11.07.1")
	if _, err := h.jobs.StartRollback(""); err == nil {
		t.Fatal("rollback without a previous version should fail")
	}
	h.jobs.Start("manual", "")
	<-h.restart
	next := h.newJobs("2026.12.01.1")
	next.Resume(context.Background(), "2026.12.01.1", func(context.Context) (string, error) { return "ok", nil })
	if _, err := next.StartRollback("david"); err != nil {
		t.Fatal(err)
	}
	<-h.restart
	if b, _ := os.ReadFile(h.exe); !strings.Contains(string(b), "2026.11.07.1") {
		t.Fatal("previous binary not restored")
	}
	after := h.newJobs("2026.11.07.1")
	after.Resume(context.Background(), "2026.11.07.1", func(context.Context) (string, error) { return "ok", nil })
	job, _ := after.Current()
	if job.Kind != "rollback" || job.Status != JobDone || job.To != "2026.11.07.1" {
		t.Fatalf("%+v", job)
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.events[len(h.events)-1] != "rolledback" {
		t.Fatalf("events %v", h.events)
	}
}

func TestNoRestarterAsksForAManualRestart(t *testing.T) {
	f := newFakeRelease(t, []string{"2026.12.01.1"}, "2026.12.01.1")
	h := newHarness(t, f, "2026.11.07.1")
	h.jobs.Restart = nil
	h.jobs.Start("manual", "")
	job := wait(t, h.jobs, JobFailed)
	if job.Step(StepRestart).Status != Failed || !strings.Contains(job.Error, "restart go-build-it to finish") || !job.CanRollback {
		t.Fatalf("%+v", job)
	}
}
