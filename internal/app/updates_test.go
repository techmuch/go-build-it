package app

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/techmuch/go-build-it/internal/api"
	"github.com/techmuch/go-build-it/internal/config"
	"github.com/techmuch/go-build-it/internal/history"
	"github.com/techmuch/go-build-it/internal/store"
)

// The change log and update routes (FR-06a–FR-06e) work without herdr: a failed update
// is shown with its steps and recorded in the history.
func TestChangelogAndUpdateRoutes(t *testing.T) {
	home := t.TempDir()
	cfg, err := config.LoadFrom(home)
	if err != nil {
		t.Fatal(err)
	}
	gh := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`[{"tag_name":"2026.12.01.1","body":"### Added\n\n- shiny","published_at":"2026-12-01T00:00:00Z","assets":[]}]`))
	}))
	defer gh.Close()
	cfg.Updates.API = gh.URL
	db, err := store.Open(filepath.Join(home, "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	a := Wire(cfg, db, "2026.11.07.1", slog.New(slog.NewTextHandler(io.Discard, nil)))
	mux := http.NewServeMux()
	api.Mount(mux, a.Routes())
	call := func(method, path string) map[string]any {
		t.Helper()
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, httptest.NewRequest(method, path, strings.NewReader("{}")))
		var out map[string]any
		json.Unmarshal(rec.Body.Bytes(), &out)
		if rec.Code != 200 {
			t.Fatalf("%s %s: %d %s", method, path, rec.Code, rec.Body)
		}
		return out
	}

	cl := call("GET", "/api/changelog")
	entries := cl["entries"].([]any)
	if cl["current"] != "2026.11.07.1" || len(entries) == 0 || entries[0].(map[string]any)["version"] != "Unreleased" {
		t.Fatalf("changelog: %v", cl)
	}
	if j := call("GET", "/api/updates/job"); j["job"] != nil {
		t.Fatalf("no job yet: %v", j)
	}
	chk := call("GET", "/api/updates")
	rels := chk["releases"].([]any)
	if chk["newer"] != true || len(rels) != 1 || !strings.Contains(rels[0].(map[string]any)["notes"].(string), "shiny") {
		t.Fatalf("check: %v", chk)
	}
	sys := call("GET", "/api/system")["update"].(map[string]any)
	if sys["newer"] != true || sys["latest"] != "2026.12.01.1" {
		t.Fatalf("system update summary: %v", sys)
	}

	// The release has no download for this platform: the job fails at the first step.
	call("POST", "/api/updates/install")
	var job map[string]any
	for i := 0; i < 100; i++ {
		job, _ = call("GET", "/api/updates/job")["job"].(map[string]any)
		if job != nil && job["status"] == "failed" {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	steps := job["steps"].([]any)
	if job["status"] != "failed" || job["canRetry"] != true || steps[0].(map[string]any)["status"] != "failed" || len(steps) != 8 {
		t.Fatalf("job: %v", job)
	}
	for _, typ := range []string{"update.started", "update.failed"} {
		evs, _ := a.Hist.List(t.Context(), history.Query{Type: typ})
		if len(evs) != 1 {
			t.Fatalf("%s events: %d", typ, len(evs))
		}
	}
	// The job is saved, so a restarted go-build-it shows it.
	if b := Wire(cfg, db, "2026.11.07.1", a.Log); func() bool { j, ok := b.Updates.Current(); return !ok || j.Status != "failed" }() {
		t.Fatal("job not reloaded from disk")
	}
}
