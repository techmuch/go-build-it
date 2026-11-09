package herdr

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"sort"
	"sync"
	"time"
)

// Watcher keeps an up-to-date view of agent state for the panes go-build-it cares
// about (FR-68, NFR-11). It bootstraps from session.snapshot, subscribes to lifecycle
// events plus pane.agent_status_changed for every watched pane, and reconciles from a
// fresh snapshot after events_lost or a dropped connection (herdr's event history is
// not durable).
type Watcher struct {
	C        *Client
	Log      *slog.Logger
	OnChange func(AgentInfo)     // called for every observed change of a watched pane
	OnGone   func(paneID string) // a watched pane exited or closed

	mu      sync.Mutex
	panes   map[string]bool
	state   map[string]AgentInfo
	restart chan struct{}
	resyncs int
}

// NewWatcher returns a watcher; call Run in a goroutine.
func NewWatcher(c *Client, log *slog.Logger) *Watcher {
	if log == nil {
		log = slog.Default()
	}
	return &Watcher{C: c, Log: log, panes: map[string]bool{}, state: map[string]AgentInfo{}, restart: make(chan struct{}, 1)}
}

// Watch adds panes; the subscription is renewed to include them.
func (w *Watcher) Watch(panes ...string) {
	w.mu.Lock()
	changed := false
	for _, p := range panes {
		if p != "" && !w.panes[p] {
			w.panes[p], changed = true, true
		}
	}
	w.mu.Unlock()
	if changed {
		w.kick()
	}
}

// Unwatch removes panes.
func (w *Watcher) Unwatch(panes ...string) {
	w.mu.Lock()
	for _, p := range panes {
		delete(w.panes, p)
		delete(w.state, p)
	}
	w.mu.Unlock()
	w.kick()
}

func (w *Watcher) kick() {
	select {
	case w.restart <- struct{}{}:
	default:
	}
}

// State returns the last known info for a pane.
func (w *Watcher) State(paneID string) (AgentInfo, bool) {
	w.mu.Lock()
	defer w.mu.Unlock()
	a, ok := w.state[paneID]
	return a, ok
}

// Resyncs counts snapshot reconciliations (for tests and diagnostics).
func (w *Watcher) Resyncs() int {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.resyncs
}

// Run subscribes until ctx ends, reconnecting with backoff.
func (w *Watcher) Run(ctx context.Context) {
	backoff := 250 * time.Millisecond
	for ctx.Err() == nil {
		err := w.once(ctx)
		if ctx.Err() != nil {
			return
		}
		if errors.Is(err, errRestart) {
			backoff = 250 * time.Millisecond
			continue
		}
		if err != nil && !errors.Is(err, ErrEventsLost) {
			w.Log.Debug("herdr watcher reconnecting", "err", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
		}
		if backoff < 5*time.Second {
			backoff *= 2
		}
	}
}

var errRestart = errors.New("restart subscription")

func (w *Watcher) once(ctx context.Context) error {
	w.mu.Lock()
	panes := make([]string, 0, len(w.panes))
	for p := range w.panes {
		panes = append(panes, p)
	}
	w.mu.Unlock()
	sort.Strings(panes)
	subs := []Sub{{Type: "pane.exited"}, {Type: "pane.closed"}, {Type: "pane.created"}, {Type: "workspace.closed"}}
	// Subscribe first, then reconcile from a snapshot, so nothing between them is missed.
	live := map[string]bool{}
	if snap, err := w.C.Snapshot(ctx); err == nil {
		for _, a := range snap.Agents {
			live[a.PaneID] = true
		}
		for _, p := range snap.Panes {
			live[p.PaneID] = true
		}
	} else {
		return err
	}
	for _, p := range panes {
		if live[p] {
			subs = append(subs, Sub{Type: "pane.agent_status_changed", PaneID: p})
		}
	}
	sctx, cancel := context.WithCancel(ctx)
	defer cancel()
	evs, errc, err := w.C.Subscribe(sctx, subs...)
	if err != nil {
		return err
	}
	w.reconcile(ctx, panes, live)
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-w.restart:
			return errRestart
		case e, ok := <-evs:
			if !ok {
				return <-errc
			}
			w.handle(ctx, e)
		}
	}
}

func (w *Watcher) reconcile(ctx context.Context, panes []string, live map[string]bool) {
	w.mu.Lock()
	w.resyncs++
	w.mu.Unlock()
	for _, p := range panes {
		if !live[p] {
			w.gone(p)
			continue
		}
		if a, err := w.C.Agent(ctx, p); err == nil {
			w.update(a)
		}
	}
}

func (w *Watcher) handle(ctx context.Context, e Event) {
	switch e.Event {
	case "pane.agent_status_changed":
		var sc StatusChange
		if json.Unmarshal(e.Data, &sc) == nil {
			// The event carries the status; read the full record (seq, session, title).
			if a, err := w.C.Agent(ctx, sc.PaneID); err == nil {
				w.update(a)
			} else {
				w.mu.Lock()
				a := w.state[sc.PaneID]
				a.PaneID, a.AgentStatus = sc.PaneID, sc.AgentStatus
				w.mu.Unlock()
				w.update(a)
			}
		}
	case "pane.exited", "pane.closed":
		var r PaneRef
		if json.Unmarshal(e.Data, &r) == nil {
			w.gone(r.PaneID)
		}
	case "workspace.closed":
		w.kick() // reconcile everything
	}
}

func (w *Watcher) update(a AgentInfo) {
	w.mu.Lock()
	if !w.panes[a.PaneID] {
		w.mu.Unlock()
		return
	}
	prev, had := w.state[a.PaneID]
	w.state[a.PaneID] = a
	w.mu.Unlock()
	if (!had || prev.AgentStatus != a.AgentStatus || prev.StateChangeSeq != a.StateChangeSeq) && w.OnChange != nil {
		w.OnChange(a)
	}
}

func (w *Watcher) gone(p string) {
	w.mu.Lock()
	watched := w.panes[p]
	delete(w.state, p)
	w.mu.Unlock()
	if watched && w.OnGone != nil {
		w.OnGone(p)
	}
}
