// Package config loads and saves go-build-it's settings under ~/.go-build-it (FR-03).
package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/user"
	"path/filepath"
	"runtime"

	"github.com/BurntSushi/toml"
)

// Harness holds per-harness settings (FR-63).
type Harness struct {
	Enabled bool     `toml:"enabled" json:"enabled"`
	Binary  string   `toml:"binary" json:"binary"`
	Model   string   `toml:"model" json:"model"`
	Flags   []string `toml:"flags" json:"flags"`
	// Window overrides the context window used for estimates when the harness does not report it.
	Window int `toml:"window" json:"window"`
}

// Experimental feature flags (rev 2: handoffs and attachable sessions are experimental).
type Experimental struct {
	SmartHandoffs        bool     `toml:"smart_handoffs" json:"smartHandoffs"`
	SmartHandoffProjects []string `toml:"smart_handoff_projects" json:"smartHandoffProjects"`
	HandoffThreshold     float64  `toml:"handoff_threshold" json:"handoffThreshold"`
	AttachableSessions   bool     `toml:"attachable_sessions" json:"attachableSessions"`
}

// Updates configures auto-update (FR-06, ADR-004).
type Updates struct {
	Channel     string `toml:"channel" json:"channel"`
	Repo        string `toml:"repo" json:"repo"`
	AutoInstall bool   `toml:"auto_install" json:"autoInstall"`
}

// Config is the full application configuration.
type Config struct {
	Listen        string             `toml:"listen" json:"listen"`
	DefaultUser   string             `toml:"default_user" json:"defaultUser"`
	Owners        []string           `toml:"owners" json:"owners"` // who may approve gates and releases (ADR-006, P9-08); empty = default_user
	AllowExternal bool               `toml:"allow_external" json:"allowExternal"`
	ExternalToken string             `toml:"external_token" json:"-"` // never sent to clients (P9-07)
	MaxConcurrent int                `toml:"max_concurrent" json:"maxConcurrent"`
	AutoDispatch  bool               `toml:"auto_dispatch" json:"autoDispatch"`
	ProjectsRoot  string             `toml:"projects_root" json:"projectsRoot"`
	Harnesses     map[string]Harness `toml:"harnesses" json:"harnesses"`
	Experimental  Experimental       `toml:"experimental" json:"experimental"`
	Updates       Updates            `toml:"updates" json:"updates"`
	dir           string
}

// Dir returns the state directory: ~/.go-build-it (or %USERPROFILE%\.go-build-it on Windows).
// GBI_HOME overrides it, which tests and development use.
func Dir() (string, error) {
	if d := os.Getenv("GBI_HOME"); d != "" {
		return d, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("find home directory: %w", err)
	}
	return filepath.Join(home, ".go-build-it"), nil
}

// Default returns the default configuration.
func Default() Config {
	name := "owner"
	if u, err := user.Current(); err == nil && u.Username != "" {
		name = u.Username
	}
	home, _ := os.UserHomeDir()
	return Config{
		Listen:        "127.0.0.1:7420",
		DefaultUser:   name,
		MaxConcurrent: 3,
		ProjectsRoot:  filepath.Join(home, "projects"),
		Harnesses: map[string]Harness{
			"claude":      {Enabled: true, Binary: "claude"},
			"opencode":    {Enabled: true, Binary: "opencode", Window: 200000},
			"antigravity": {Enabled: true, Binary: "agy", Window: 1000000},
			"pi":          {Enabled: true, Binary: "pi"},
		},
		Experimental: Experimental{HandoffThreshold: 0.75},
		Updates:      Updates{Channel: "stable", Repo: "techmuch/go-build-it", AutoInstall: true},
	}
}

// Path returns the config file path in dir.
func Path(dir string) string { return filepath.Join(dir, "config.toml") }

// Load reads config.toml from the state directory, creating the directory and
// a default file when missing.
func Load() (*Config, error) {
	dir, err := Dir()
	if err != nil {
		return nil, err
	}
	return LoadFrom(dir)
}

// LoadFrom reads config from a specific state directory.
func LoadFrom(dir string) (*Config, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create state dir: %w", err)
	}
	c := Default()
	c.dir = dir
	b, err := os.ReadFile(Path(dir))
	if errors.Is(err, fs.ErrNotExist) {
		return &c, c.Save()
	}
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	if err := toml.Unmarshal(b, &c); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	c.dir = dir
	for k, v := range Default().Harnesses {
		if _, ok := c.Harnesses[k]; !ok {
			if c.Harnesses == nil {
				c.Harnesses = map[string]Harness{}
			}
			c.Harnesses[k] = v
		}
	}
	return &c, nil
}

// Dir returns the directory this config was loaded from.
func (c *Config) Dir() string { return c.dir }

// DBPath is the SQLite database location.
func (c *Config) DBPath() string { return filepath.Join(c.dir, "gbi.db") }

// Save writes the config back to disk.
func (c *Config) Save() error {
	f, err := os.Create(Path(c.dir))
	if err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	defer f.Close()
	fmt.Fprintf(f, "# go-build-it configuration (%s/%s)\n", runtime.GOOS, runtime.GOARCH)
	return toml.NewEncoder(f).Encode(c)
}

// IsOwner reports whether a name or Tailscale login is an owner.
func (c *Config) IsOwner(name, login string) bool {
	owners := c.Owners
	if len(owners) == 0 {
		owners = []string{c.DefaultUser}
	}
	for _, o := range owners {
		if o != "" && (o == name || o == login) {
			return true
		}
	}
	return false
}

// HandoffsEnabledFor reports whether experimental smart handoffs apply to a project.
func (c *Config) HandoffsEnabledFor(project string) bool {
	if !c.Experimental.SmartHandoffs {
		return false
	}
	if len(c.Experimental.SmartHandoffProjects) == 0 {
		return true
	}
	for _, p := range c.Experimental.SmartHandoffProjects {
		if p == project {
			return true
		}
	}
	return false
}
