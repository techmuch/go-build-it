// Command gbi is go-build-it: the server (`gbi serve`) and a CLI with the same
// operations as the web UI (FR-30, FR-32).
package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/techmuch/go-build-it/internal/fakemember"
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/techmuch/go-build-it/docs"
	"github.com/techmuch/go-build-it/internal/access"
	"github.com/techmuch/go-build-it/internal/api"
	"github.com/techmuch/go-build-it/internal/app"
	"github.com/techmuch/go-build-it/internal/config"
	"github.com/techmuch/go-build-it/internal/service"
	"github.com/techmuch/go-build-it/internal/store"
	"github.com/techmuch/go-build-it/internal/update"
	"github.com/techmuch/go-build-it/web"
)

// version is stamped at build time: -ldflags "-X main.version=YYYY.MM.DD.N" (FR-05).
var version = "dev"

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "gbi:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 || args[0] == "help" || args[0] == "--help" || args[0] == "-h" {
		return usage(args)
	}
	switch args[0] {
	case "serve":
		return serve(args[1:])
	case "version", "--version":
		fmt.Println(version)
		return nil
	case "service":
		return serviceCmd(args[1:])
	case "mcp":
		return mcpServe()
	case "dev":
		return devCmd(args[1:])
	}
	return cliCall(args)
}

func newLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
}

func serve(args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	for i := 0; i < len(args); i++ {
		if args[i] == "--listen" && i+1 < len(args) {
			cfg.Listen = args[i+1]
			i++
		} else if strings.HasPrefix(args[i], "--listen=") {
			cfg.Listen = strings.TrimPrefix(args[i], "--listen=")
		}
	}
	log := newLogger()
	a, err := app.New(cfg, version, log)
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if n, _ := a.Standards.SeedFromDir(ctx, filepath.Join(cfg.Dir(), "standards")); n > 0 {
		log.Info("seeded standards", "count", n)
	}
	// After an update installs, the job asks to restart: shut down cleanly, then replace
	// this process with the new binary (FR-06d, ADR-011).
	restart := make(chan struct{})
	var restartOnce sync.Once
	a.Updates.Restart = func() { restartOnce.Do(func() { close(restart) }) }
	a.Start(ctx)
	// Request contexts derive from base, so a restart can end live streams (SSE, terminals)
	// at once instead of waiting for them.
	base, endStreams := context.WithCancel(context.Background())
	defer endStreams()
	srv := &http.Server{Addr: cfg.Listen, Handler: Handler(a), ReadHeaderTimeout: 10 * time.Second, BaseContext: func(net.Listener) context.Context { return base }}
	host, _, _ := net.SplitHostPort(cfg.Listen)
	if ip := net.ParseIP(host); ip == nil || !ip.IsLoopback() {
		if !cfg.AllowExternal {
			log.Warn("listening on a non-loopback address; external requests will be refused (ADR-005)", "listen", cfg.Listen)
		}
	}
	// The process this one replaced may still hold the port for a moment (Windows restarts).
	ln, err := listenRetry(cfg.Listen, 15*time.Second)
	if err != nil {
		return err
	}
	errc := make(chan error, 1)
	go func() { errc <- srv.Serve(ln) }()
	log.Info("go-build-it running", "version", version, "url", "http://"+cfg.Listen, "data", cfg.Dir())
	select {
	case <-ctx.Done():
		sctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return srv.Shutdown(sctx)
	case <-restart:
		log.Info("restarting into the updated binary", "exe", a.Updater.Exe)
		endStreams()
		sctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		if srv.Shutdown(sctx) != nil {
			_ = srv.Close() // live streams (SSE, terminals) do not end by themselves
		}
		cancel()
		stop()
		_ = a.DB.Close()
		if err := update.Exec(a.Updater.Exe, os.Args[1:]); err != nil {
			return fmt.Errorf("restart into %s: %w", a.Updater.Exe, err)
		}
		return nil
	case err := <-errc:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}

// listenRetry binds addr, retrying while it is in use for up to wait.
func listenRetry(addr string, wait time.Duration) (net.Listener, error) {
	deadline := time.Now().Add(wait)
	for {
		ln, err := net.Listen("tcp", addr)
		if err == nil || time.Now().After(deadline) {
			return ln, err
		}
		time.Sleep(250 * time.Millisecond)
	}
}

// Handler builds the HTTP handler: API, raw artifacts, docs assets and the embedded UI.
func Handler(a *app.App) http.Handler {
	mux := http.NewServeMux()
	api.Mount(mux, a.Routes())
	mux.HandleFunc("GET /raw/{id}/{path...}", func(w http.ResponseWriter, r *http.Request) {
		p, err := a.Projects.Get(r.Context(), r.PathValue("id"))
		if err != nil {
			http.NotFound(w, r)
			return
		}
		rel := filepath.Clean(r.PathValue("path"))
		if strings.HasPrefix(rel, "..") || filepath.IsAbs(rel) {
			http.Error(w, "bad path", http.StatusBadRequest)
			return
		}
		// Mockups run sandboxed: scripts allowed, no access to the app origin (FR-102a).
		w.Header().Set("Content-Security-Policy", "sandbox allow-scripts allow-forms allow-popups")
		http.ServeFile(w, r, filepath.Join(p.Path, rel))
	})
	mux.Handle("GET /docs-assets/", http.StripPrefix("/docs-assets/", http.FileServerFS(docs.FS())))
	mux.Handle("GET /", spa(web.Dist()))
	return access.Middleware(a.Cfg.DefaultUser, func() access.ExternalPolicy {
		return access.ExternalPolicy{Allow: a.Cfg.AllowExternal, Token: a.Cfg.ExternalToken}
	}, mux)
}

// spa serves the built frontend, falling back to index.html for client routes.
func spa(dist fs.FS) http.Handler {
	files := http.FileServerFS(dist)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		if p == "" {
			p = "index.html"
		}
		if _, err := fs.Stat(dist, p); err != nil {
			idx, err := fs.ReadFile(dist, "index.html")
			if err != nil {
				http.Error(w, "frontend not built: run `make web`", http.StatusNotFound)
				return
			}
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write(idx)
			return
		}
		if strings.HasPrefix(p, "assets/") {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		}
		files.ServeHTTP(w, r)
	})
}

func serviceCmd(args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	exe, _ := os.Executable()
	plan, err := service.Build(exe, cfg.Dir())
	if err != nil {
		return err
	}
	sub := "plan"
	if len(args) > 0 {
		sub = args[0]
	}
	switch sub {
	case "plan":
		fmt.Printf("OS: %s\nFile: %s\nStart: %s\nStop: %s\n\n%s", plan.OS, plan.File, strings.Join(plan.Start, " "), strings.Join(plan.Stop, " "), plan.Content)
		return nil
	case "install":
		if err := service.Install(plan); err != nil {
			return err
		}
		fmt.Println("go-build-it will run in the background at login:", "http://"+cfg.Listen)
		return nil
	case "uninstall":
		return service.Uninstall(plan)
	}
	return fmt.Errorf("usage: gbi service plan|install|uninstall")
}

func devCmd(args []string) error {
	if len(args) == 0 {
		return errors.New("usage: gbi dev gen-client|fake-agent|fake-member|sign|openapi")
	}
	switch args[0] {
	case "fake-member":
		return fakemember.Run(args[1:], os.Stdin, os.Stdout)
	case "gen-client":
		out := "web/src/api.gen.ts"
		if len(args) > 1 {
			out = args[1]
		}
		a := app.Wire(&config.Config{}, mustMemDB(), version, newLogger())
		return os.WriteFile(out, []byte(api.TSClient(a.Routes())), 0o644)
	case "fake-agent":
		return fakeAgent(args[1:])
	case "fake-tui":
		return fakeTUI()
	case "changelog":
		// gbi dev changelog release VERSION · notes VERSION · check (FR-06a, ADR-011)
		path := "CHANGELOG.md"
		if len(args) < 2 {
			return errors.New("usage: gbi dev changelog release|notes VERSION, or gbi dev changelog check")
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("run this from the go-build-it repository: %w", err)
		}
		entries := update.ParseChangelog(b)
		switch args[1] {
		case "check":
			if errs := update.CheckChangelog(entries); len(errs) > 0 {
				return errors.New(strings.Join(errs, "; "))
			}
			fmt.Printf("CHANGELOG.md ok: %d entries\n", len(entries))
			return nil
		case "notes", "release":
			if len(args) < 3 {
				return fmt.Errorf("usage: gbi dev changelog %s VERSION", args[1])
			}
			if args[1] == "release" {
				if err := update.ReleaseChangelog(path, args[2], time.Now()); err != nil {
					return err
				}
				fmt.Printf("CHANGELOG.md: Unreleased is now %s. Commit it, then: VERSION=%s make release\n", args[2], args[2])
				return nil
			}
			n, err := update.Notes(entries, args[2])
			if err != nil {
				return err
			}
			fmt.Println(n)
			return nil
		}
		return fmt.Errorf("unknown changelog command %q", args[1])
	case "sign":
		key := os.Getenv("GBI_SIGNING_KEY")
		if key == "" {
			home, _ := os.UserHomeDir()
			b, err := os.ReadFile(filepath.Join(home, ".go-build-it", "release-signing.key"))
			if err != nil {
				return fmt.Errorf("no signing key: %w", err)
			}
			key = string(b)
		}
		files := args[1:]
		if len(files) == 0 {
			return errors.New("usage: gbi dev sign <files...>")
		}
		sums, sig, err := update.Sign(files, key)
		if err != nil {
			return err
		}
		dir := filepath.Dir(files[0])
		if err := os.WriteFile(filepath.Join(dir, "SHA256SUMS"), sums, 0o644); err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(dir, "SHA256SUMS.sig"), sig, 0o644)
	}
	return fmt.Errorf("unknown dev command %q", args[0])
}

func mustMemDB() *sqlDB {
	db, err := store.Open(":memory:")
	if err != nil {
		panic(err)
	}
	return db
}
