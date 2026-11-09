package update

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Step statuses.
const (
	Pending = "pending"
	Running = "running"
	Done    = "done"
	Failed  = "failed"
	Skipped = "skipped"
)

// Job statuses. "restarting" means the new binary is installed and go-build-it is
// restarting; the restarted process finishes the job (ADR-011).
const (
	JobRunning    = "running"
	JobRestarting = "restarting"
	JobDone       = "done"
	JobFailed     = "failed"
)

// Step IDs, in order (FR-06d).
const (
	StepFind      = "find"
	StepDownload  = "download"
	StepSignature = "signature"
	StepChecksum  = "checksum"
	StepTest      = "test"
	StepInstall   = "install"
	StepRestore   = "restore" // rollback jobs
	StepRestart   = "restart"
	StepReconnect = "reconnect"
)

// Step is one named step of an update, shown as it runs.
type Step struct {
	ID        string     `json:"id"`
	Title     string     `json:"title"`
	Status    string     `json:"status"`
	Detail    string     `json:"detail,omitempty"`
	Error     string     `json:"error,omitempty"`
	StartedAt *time.Time `json:"startedAt,omitempty"`
	EndedAt   *time.Time `json:"endedAt,omitempty"`
	Bytes     int64      `json:"bytes,omitempty"` // download progress
	Total     int64      `json:"total,omitempty"`
}

// Job is one update (or rollback) and its steps. It is saved after every change so the
// restarted go-build-it can show and finish it.
type Job struct {
	ID          string     `json:"id"`
	Kind        string     `json:"kind"`    // update | rollback
	Trigger     string     `json:"trigger"` // manual | auto
	Actor       string     `json:"actor,omitempty"`
	From        string     `json:"from"`
	To          string     `json:"to"`
	Status      string     `json:"status"`
	Error       string     `json:"error,omitempty"`
	StartedAt   time.Time  `json:"startedAt"`
	EndedAt     *time.Time `json:"endedAt,omitempty"`
	Steps       []Step     `json:"steps"`
	CanRetry    bool       `json:"canRetry"`    // failed before anything changed
	CanRollback bool       `json:"canRollback"` // failed after the install step
}

// Step returns the step with id, or nil.
func (j *Job) Step(id string) *Step {
	for i := range j.Steps {
		if j.Steps[i].ID == id {
			return &j.Steps[i]
		}
	}
	return nil
}

func (j Job) clone() Job {
	j.Steps = append([]Step(nil), j.Steps...)
	return j
}

// Jobs runs updates one at a time and remembers the last one.
type Jobs struct {
	U    *Updater
	Path string // job file, e.g. ~/.go-build-it/update/job.json
	// Restart replaces the running process with the installed binary. It is set by
	// `gbi serve`; when it is nil the restart step asks the user to restart by hand.
	Restart func()
	// OnEvent is told about job milestones: started, completed, failed, rolledback.
	OnEvent func(kind string, j Job)
	// RestartDelay lets the UI see the restart step running before the process goes away.
	RestartDelay time.Duration

	mu       sync.Mutex
	job      *Job
	lastSave time.Time
}

// NewJobs returns a job runner that saves to path and loads the last job from it.
func NewJobs(u *Updater, path string) *Jobs {
	j := &Jobs{U: u, Path: path, RestartDelay: 1200 * time.Millisecond}
	if b, err := os.ReadFile(path); err == nil {
		var job Job
		if json.Unmarshal(b, &job) == nil && job.ID != "" {
			j.job = &job
		}
	}
	return j
}

// Current returns the running or last job (ok is false when there has never been one).
func (j *Jobs) Current() (Job, bool) {
	j.mu.Lock()
	defer j.mu.Unlock()
	if j.job == nil {
		return Job{}, false
	}
	return j.job.clone(), true
}

// Busy reports whether a job is in progress.
func (j *Jobs) Busy() bool {
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.job != nil && (j.job.Status == JobRunning || j.job.Status == JobRestarting)
}

// ErrBusy is returned when an update is already in progress.
var ErrBusy = errors.New("an update is already in progress")

func updateSteps(asset string) []Step {
	return []Step{
		{ID: StepFind, Title: "Find the release", Status: Pending},
		{ID: StepDownload, Title: "Download", Status: Pending, Detail: asset},
		{ID: StepSignature, Title: "Verify the signature", Status: Pending, Detail: "Ed25519 over SHA256SUMS"},
		{ID: StepChecksum, Title: "Verify the checksum", Status: Pending, Detail: "SHA-256 of " + asset},
		{ID: StepTest, Title: "Test the new binary", Status: Pending},
		{ID: StepInstall, Title: "Install", Status: Pending},
		{ID: StepRestart, Title: "Restart go-build-it", Status: Pending, Detail: "the page reconnects by itself"},
		{ID: StepReconnect, Title: "Reconnect", Status: Pending, Detail: "gbi herdr session and running members"},
	}
}

// Start begins an update to the latest release in the background (FR-06d).
func (j *Jobs) Start(trigger, actor string) (Job, error) {
	return j.begin(Job{Kind: "update", Trigger: trigger, Actor: actor, From: j.U.Current, Steps: updateSteps(AssetName())}, j.runUpdate)
}

// StartRollback restores the previous binary and restarts, shown with the same steps view.
func (j *Jobs) StartRollback(actor string) (Job, error) {
	prev := j.U.previousVersion()
	if _, err := os.Stat(j.U.Exe + ".prev"); err != nil {
		return Job{}, errors.New("no previous version to roll back to")
	}
	steps := []Step{
		{ID: StepRestore, Title: "Restore " + firstNonEmpty(prev, "the previous version"), Status: Pending, Detail: "keeps " + j.U.Current + " so you can update again"},
		{ID: StepRestart, Title: "Restart go-build-it", Status: Pending, Detail: "the page reconnects by itself"},
		{ID: StepReconnect, Title: "Reconnect", Status: Pending, Detail: "gbi herdr session and running members"},
	}
	return j.begin(Job{Kind: "rollback", Trigger: "manual", Actor: actor, From: j.U.Current, To: prev, Steps: steps}, j.runRollback)
}

func (j *Jobs) begin(job Job, run func()) (Job, error) {
	j.mu.Lock()
	if j.job != nil && (j.job.Status == JobRunning || j.job.Status == JobRestarting) {
		j.mu.Unlock()
		return Job{}, ErrBusy
	}
	job.ID = fmt.Sprintf("U-%d", time.Now().UnixMilli())
	job.Status = JobRunning
	job.StartedAt = time.Now().UTC()
	j.job = &job
	j.saveLocked(true)
	out := job.clone()
	j.mu.Unlock()
	j.event("started", out)
	go run()
	return out, nil
}

func (j *Jobs) event(kind string, job Job) {
	if j.OnEvent != nil {
		j.OnEvent(kind, job)
	}
}

// change applies fn to the job under the lock and saves it (throttled unless force).
func (j *Jobs) change(force bool, fn func(*Job)) {
	j.mu.Lock()
	defer j.mu.Unlock()
	fn(j.job)
	j.saveLocked(force)
}

func (j *Jobs) saveLocked(force bool) {
	if j.Path == "" || j.job == nil {
		return
	}
	if !force && time.Since(j.lastSave) < 250*time.Millisecond {
		return
	}
	j.lastSave = time.Now()
	b, _ := json.MarshalIndent(j.job, "", "  ")
	_ = os.MkdirAll(filepath.Dir(j.Path), 0o755)
	tmp := j.Path + ".tmp"
	if os.WriteFile(tmp, b, 0o644) == nil {
		_ = os.Rename(tmp, j.Path)
	}
}

func now() *time.Time { t := time.Now().UTC(); return &t }

func (j *Jobs) begins(id, detail string) {
	j.change(true, func(job *Job) {
		s := job.Step(id)
		s.Status, s.StartedAt = Running, now()
		if detail != "" {
			s.Detail = detail
		}
	})
}

func (j *Jobs) finishes(id, detail string) {
	j.change(true, func(job *Job) {
		s := job.Step(id)
		s.Status, s.EndedAt = Done, now()
		if s.StartedAt == nil {
			s.StartedAt = s.EndedAt
		}
		if detail != "" {
			s.Detail = detail
		}
	})
}

// fails marks step id failed and ends the job. Steps before the install change nothing,
// so the user can try again; later failures offer a rollback (FR-06d).
func (j *Jobs) fails(id string, err error) {
	var out Job
	j.change(true, func(job *Job) {
		s := job.Step(id)
		s.Status, s.EndedAt, s.Error = Failed, now(), err.Error()
		if s.StartedAt == nil {
			s.StartedAt = s.EndedAt
		}
		for i := range job.Steps {
			if job.Steps[i].Status == Pending {
				job.Steps[i].Status = Skipped
			}
		}
		job.Status, job.Error, job.EndedAt = JobFailed, err.Error(), now()
		installed := false
		for _, st := range job.Steps {
			if (st.ID == StepInstall || st.ID == StepRestore) && st.Status == Done {
				installed = true
			}
		}
		job.CanRetry = !installed && job.Kind == "update"
		job.CanRollback = installed && job.Kind == "update"
		out = job.clone()
	})
	j.event("failed", out)
}

func (j *Jobs) runUpdate() {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
	defer cancel()
	u := j.U

	j.begins(StepFind, "")
	rel, err := u.Check(ctx)
	if err != nil {
		j.fails(StepFind, err)
		return
	}
	if !rel.Newer {
		j.fails(StepFind, fmt.Errorf("already up to date: %s is the latest release", u.Current))
		return
	}
	if rel.URL == "" {
		j.fails(StepFind, fmt.Errorf("release %s has no %s download", rel.Tag, rel.AssetName))
		return
	}
	j.change(true, func(job *Job) {
		job.To = rel.Tag
		job.Step(StepTest).Detail = "runs it and expects " + rel.Tag
		job.Step(StepInstall).Detail = "keeps " + u.Current + " for rollback"
	})
	j.finishes(StepFind, rel.Tag+" · "+rel.AssetName)

	j.begins(StepDownload, "")
	asset, err := u.download(ctx, rel.URL, func(done, total int64) {
		j.change(false, func(job *Job) {
			s := job.Step(StepDownload)
			s.Bytes, s.Total = done, total
			s.Detail = progressText(done, total)
		})
	})
	if err != nil {
		j.fails(StepDownload, err)
		return
	}
	j.change(true, func(job *Job) { s := job.Step(StepDownload); s.Bytes, s.Total = int64(len(asset)), int64(len(asset)) })
	j.finishes(StepDownload, megabytes(int64(len(asset))))

	j.begins(StepSignature, "")
	base := rel.URL[:strings.LastIndex(rel.URL, "/")+1]
	sums, err := u.download(ctx, base+"SHA256SUMS", nil)
	if err != nil {
		j.fails(StepSignature, fmt.Errorf("download SHA256SUMS: %w", err))
		return
	}
	sig, err := u.download(ctx, base+"SHA256SUMS.sig", nil)
	if err != nil {
		j.fails(StepSignature, fmt.Errorf("download SHA256SUMS.sig: %w", err))
		return
	}
	if err := VerifySignature(sums, sig, u.Key); err != nil {
		j.fails(StepSignature, err)
		return
	}
	j.finishes(StepSignature, "signed by the go-build-it release key ✓")

	j.begins(StepChecksum, "")
	if err := VerifyChecksum(asset, rel.AssetName, sums); err != nil {
		j.fails(StepChecksum, err)
		return
	}
	j.finishes(StepChecksum, "SHA-256 matches")

	j.begins(StepTest, "")
	next := u.Exe + ".new"
	if err := os.WriteFile(next, asset, 0o755); err != nil {
		j.fails(StepTest, err)
		return
	}
	got, err := runVersion(ctx, next)
	if err != nil {
		_ = os.Remove(next)
		j.fails(StepTest, fmt.Errorf("the new binary did not run: %w", err))
		return
	}
	if !SameVersion(got, rel.Tag) {
		_ = os.Remove(next)
		j.fails(StepTest, fmt.Errorf("the new binary reports %q, expected %s", got, rel.Tag))
		return
	}
	j.finishes(StepTest, "printed "+got)

	j.begins(StepInstall, "")
	if err := u.install(next); err != nil {
		_ = os.Remove(next)
		j.fails(StepInstall, err)
		return
	}
	j.finishes(StepInstall, u.Current+" kept for rollback")
	j.restart()
}

func (j *Jobs) runRollback() {
	j.begins(StepRestore, "")
	v, err := j.U.Rollback()
	if err != nil {
		j.fails(StepRestore, err)
		return
	}
	j.change(true, func(job *Job) { job.To = v })
	j.finishes(StepRestore, "restored "+firstNonEmpty(v, "the previous version")+"; "+j.U.Current+" kept")
	j.restart()
}

// restart marks the job as restarting and hands over to the new process.
func (j *Jobs) restart() {
	var out Job
	j.change(true, func(job *Job) {
		s := job.Step(StepRestart)
		s.Status, s.StartedAt = Running, now()
		job.Status = JobRestarting
		out = job.clone()
	})
	if j.Restart == nil {
		j.fails(StepRestart, errors.New("installed; restart go-build-it to finish (it is not running as a server)"))
		return
	}
	j.event("restarting", out)
	time.Sleep(j.RestartDelay)
	j.Restart()
}

// Resume is called once by a starting go-build-it. If the saved job was restarting into
// this version, it completes the restart step, runs reconnect, and marks the job done.
// A job that was running when the process stopped is marked failed (ADR-011).
func (j *Jobs) Resume(ctx context.Context, version string, reconnect func(context.Context) (string, error)) {
	j.mu.Lock()
	job := j.job
	j.mu.Unlock()
	if job == nil {
		return
	}
	switch job.Status {
	case JobRunning:
		for _, s := range job.Steps {
			if s.Status == Running {
				j.fails(s.ID, errors.New("interrupted: go-build-it stopped during this step"))
				return
			}
		}
		j.fails(job.Steps[0].ID, errors.New("interrupted: go-build-it stopped during the update"))
	case JobRestarting:
		if job.To != "" && !SameVersion(version, job.To) {
			j.fails(StepRestart, fmt.Errorf("go-build-it restarted but is running %s, not %s", version, job.To))
			return
		}
		var took string
		j.change(true, func(job *Job) {
			if s := job.Step(StepRestart); s.StartedAt != nil {
				took = fmt.Sprintf("back in %.1fs", time.Since(*s.StartedAt).Seconds())
			}
		})
		j.finishes(StepRestart, took)
		j.begins(StepReconnect, "")
		rctx, cancel := context.WithTimeout(ctx, 60*time.Second)
		detail, err := reconnect(rctx)
		cancel()
		if err != nil {
			j.fails(StepReconnect, err)
			return
		}
		j.finishes(StepReconnect, detail)
		var out Job
		j.change(true, func(job *Job) {
			job.Status, job.EndedAt = JobDone, now()
			out = job.clone()
		})
		kind := "completed"
		if out.Kind == "rollback" {
			kind = "rolledback"
		}
		j.event(kind, out)
	}
}

func runVersion(ctx context.Context, bin string) (string, error) {
	tctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	out, err := exec.CommandContext(tctx, bin, "version").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func megabytes(n int64) string { return fmt.Sprintf("%.1f MB", float64(n)/1e6) }

func progressText(done, total int64) string {
	if total <= 0 {
		return megabytes(done)
	}
	return fmt.Sprintf("%.1f of %.1f MB · %d%%", float64(done)/1e6, float64(total)/1e6, done*100/total)
}

func firstNonEmpty(s ...string) string {
	for _, x := range s {
		if x != "" {
			return x
		}
	}
	return ""
}
