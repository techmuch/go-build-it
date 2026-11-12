package app

import (
	"context"
	"sync"
	"time"

	"github.com/techmuch/go-build-it/internal/api"
	"github.com/techmuch/go-build-it/internal/harness"
	"github.com/techmuch/go-build-it/internal/store"
)

// Model lists for the model picker (FR-63a). Asking a harness can take seconds and needs
// the network, so lists are cached per harness and refreshed in the background once a day,
// or at once on request. A failed refresh keeps the last good list.

// ModelList is one harness's models and where they came from.
type ModelList struct {
	Harness   string          `json:"harness"`
	Models    []harness.Model `json:"models"`
	Source    string          `json:"source"`
	FetchedAt string          `json:"fetchedAt"`
	Error     string          `json:"error,omitempty"`
	Default   string          `json:"default"` // the harness's configured default model ("" = its own)
}

type modelCache struct {
	mu      sync.Mutex
	lists   map[string]ModelList
	at      map[string]time.Time
	running map[string]bool
}

const modelsMaxAge = 24 * time.Hour

func (a *App) harnessModels(c *api.Ctx) (any, error) {
	name := c.Param("name")
	if _, ok := harness.Get(name); !ok {
		return nil, api.NotFound("unknown harness %q", name)
	}
	refresh := c.R.URL.Query().Get("refresh") == "true"
	l := a.models.get(name)
	switch {
	case refresh || l.FetchedAt == "":
		l = a.fetchModels(c.R.Context(), name)
	case a.models.stale(name):
		go a.fetchModels(context.Background(), name)
	}
	l.Default = a.Cfg.Harnesses[name].Model
	return l, nil
}

// fetchModels asks the harness for its models and caches the result.
func (a *App) fetchModels(ctx context.Context, name string) ModelList {
	if !a.models.start(name) {
		return a.models.get(name) // a refresh is already running
	}
	defer a.models.done(name)
	p, _ := harness.Get(name)
	ms, src, err := p.Models(ctx, a.Cfg.Harnesses[name].Binary)
	l := ModelList{Harness: name, Models: ms, Source: src, FetchedAt: store.Now()}
	if err != nil {
		l.Error = err.Error()
		if prev := a.models.get(name); len(prev.Models) > 0 {
			prev.Error = "refresh failed, showing the list from " + prev.FetchedAt + ": " + err.Error()
			return prev
		}
	}
	a.models.put(name, l)
	return l
}

func (m *modelCache) get(name string) ModelList {
	m.mu.Lock()
	defer m.mu.Unlock()
	if l, ok := m.lists[name]; ok {
		return l
	}
	return ModelList{Harness: name, Models: []harness.Model{}}
}

func (m *modelCache) put(name string, l ModelList) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.lists == nil {
		m.lists, m.at = map[string]ModelList{}, map[string]time.Time{}
	}
	m.lists[name], m.at[name] = l, time.Now()
}

func (m *modelCache) stale(name string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return time.Since(m.at[name]) > modelsMaxAge
}

func (m *modelCache) start(name string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.running == nil {
		m.running = map[string]bool{}
	}
	if m.running[name] {
		return false
	}
	m.running[name] = true
	return true
}

func (m *modelCache) done(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.running, name)
}
