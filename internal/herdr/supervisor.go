package herdr

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// SupportedRange is the herdr version range go-build-it is tested against (NFR-32).
// MinVersion is inclusive; a herdr newer than TestedMax still runs but gets a warning.
const (
	MinVersion = "0.9.2" // resume commands and agent_session reports
	TestedMax  = "0.9.5"
)

// Supervisor keeps go-build-it's herdr session running (FR-07). It adopts a session
// that is already up, starts one detached when it is not, and restarts it if it
// stops answering. Stopping go-build-it never stops the session (FR-08).
type Supervisor struct {
	C          *Client
	ConfigHome string // optional XDG_CONFIG_HOME for an isolated herdr
	Log        *slog.Logger
	Interval   time.Duration

	mu       sync.Mutex
	status   Status
	restarts int
}

// Status is what the doctor and settings show.
type Status struct {
	Running   bool      `json:"running"`
	Version   string    `json:"version"`
	Protocol  int64     `json:"protocol"`
	Session   string    `json:"session"`
	Socket    string    `json:"socket"`
	Adopted   bool      `json:"adopted"`
	Restarts  int       `json:"restarts"`
	CheckedAt time.Time `json:"checkedAt"`
	Error     string    `json:"error,omitempty"`
}

// NewSupervisor builds a supervisor for bin/session, isolated under configHome when set.
func NewSupervisor(bin, session, configHome string, log *slog.Logger) *Supervisor {
	if log == nil {
		log = slog.Default()
	}
	s := &Supervisor{ConfigHome: configHome, Log: log, Interval: 5 * time.Second}
	s.C = NewWithEnv(bin, session, s.env()) // the client resolves sockets with the same herdr
	return s
}

// Status returns the last known state.
func (s *Supervisor) Status() Status {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.status
}

// Ensure makes sure the session answers, starting it if needed, and returns its status.
func (s *Supervisor) Ensure(ctx context.Context) (Status, error) {
	pctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	p, err := s.C.Ping(pctx)
	cancel()
	if err == nil {
		return s.set(true, p, true, nil), nil
	}
	if err := s.spawn(); err != nil {
		st := s.set(false, Pong{}, false, err)
		return st, err
	}
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return s.Status(), ctx.Err()
		case <-time.After(200 * time.Millisecond):
		}
		c2, cancel := context.WithTimeout(ctx, time.Second)
		p, err = s.C.Ping(c2)
		cancel()
		if err == nil {
			s.C.Socket = s.C.resolveSocket()
			return s.set(true, p, false, nil), nil
		}
	}
	err = fmt.Errorf("started herdr --session %s server but it did not answer on %s: %w", s.C.Session, s.C.Socket, err)
	return s.set(false, Pong{}, false, err), err
}

// Run checks the session every Interval and restarts it if it stops answering.
func (s *Supervisor) Run(ctx context.Context) {
	t := time.NewTicker(s.Interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
		pctx, cancel := context.WithTimeout(ctx, 2*time.Second)
		p, err := s.C.Ping(pctx)
		cancel()
		if err == nil {
			s.set(true, p, s.Status().Adopted, nil)
			continue
		}
		if ctx.Err() != nil {
			return
		}
		s.Log.Warn("herdr session stopped answering; restarting", "session", s.C.Session, "err", err)
		s.mu.Lock()
		s.restarts++
		s.mu.Unlock()
		if _, err := s.Ensure(ctx); err != nil {
			s.Log.Error("could not restart herdr session", "err", err)
		}
	}
}

// Stop stops the session (explicit user action only).
func (s *Supervisor) Stop(ctx context.Context) error {
	out, err := s.command(ctx, "session", "stop", s.C.Session).CombinedOutput()
	if err != nil {
		return fmt.Errorf("herdr session stop %s: %v: %s", s.C.Session, err, strings.TrimSpace(string(out)))
	}
	s.set(false, Pong{}, false, nil)
	return nil
}

func (s *Supervisor) command(ctx context.Context, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, s.C.Bin, args...)
	cmd.Env = s.env()
	return cmd
}

func (s *Supervisor) env() []string {
	env := os.Environ()
	// The session is chosen explicitly; never let a stray variable point herdr elsewhere.
	out := env[:0]
	for _, e := range env {
		if strings.HasPrefix(e, "HERDR_SESSION=") || strings.HasPrefix(e, "HERDR_SOCKET_PATH=") {
			continue
		}
		out = append(out, e)
	}
	if s.ConfigHome != "" {
		out = append(out, "XDG_CONFIG_HOME="+s.ConfigHome)
	}
	return out
}

// spawn starts `herdr --session <name> server` detached, so it outlives go-build-it.
func (s *Supervisor) spawn() error {
	if _, err := exec.LookPath(s.C.Bin); err != nil {
		return fmt.Errorf("herdr is not installed (%s): %w", s.C.Bin, err)
	}
	cmd := exec.Command(s.C.Bin, "--session", s.C.Session, "server")
	cmd.Env = s.env()
	detach(cmd)
	if err := cmd.Start(); err != nil {
		return err
	}
	s.Log.Info("started herdr session", "session", s.C.Session, "pid", cmd.Process.Pid)
	go func() { _ = cmd.Wait() }() // reap if it exits while we run
	return nil
}

func (s *Supervisor) set(running bool, p Pong, adopted bool, err error) Status {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.status = Status{Running: running, Version: p.Version, Protocol: p.Protocol, Session: s.C.Session, Socket: s.C.Socket,
		Adopted: adopted, Restarts: s.restarts, CheckedAt: time.Now()}
	if err != nil {
		s.status.Error = err.Error()
	}
	return s.status
}

// CompareVersions compares dotted versions numerically: -1, 0 or 1.
func CompareVersions(a, b string) int {
	pa, pb := strings.Split(strings.TrimPrefix(a, "v"), "."), strings.Split(strings.TrimPrefix(b, "v"), ".")
	for i := 0; i < len(pa) || i < len(pb); i++ {
		var x, y int
		if i < len(pa) {
			fmt.Sscanf(pa[i], "%d", &x)
		}
		if i < len(pb) {
			fmt.Sscanf(pb[i], "%d", &y)
		}
		if x != y {
			if x < y {
				return -1
			}
			return 1
		}
	}
	return 0
}

// ErrTooOld means herdr is older than MinVersion.
var ErrTooOld = errors.New("herdr is older than the supported minimum " + MinVersion)
