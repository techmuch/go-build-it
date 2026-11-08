// Command gbi is go-build-it: the server (`gbi serve`) and a CLI with the same
// operations as the web UI (FR-30, FR-32).
package main

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"strings"
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
	a.Start(ctx)
	srv := &http.Server{Addr: cfg.Listen, Handler: Handler(a), ReadHeaderTimeout: 10 * time.Second}
	host, _, _ := net.SplitHostPort(cfg.Listen)
	if ip := net.ParseIP(host); ip == nil || !ip.IsLoopback() {
		if !cfg.AllowExternal {
			log.Warn("listening on a non-loopback address; external requests will be refused (ADR-005)", "listen", cfg.Listen)
		}
	}
	errc := make(chan error, 1)
	go func() { errc <- srv.ListenAndServe() }()
	log.Info("go-build-it running", "version", version, "url", "http://"+cfg.Listen, "data", cfg.Dir())
	select {
	case <-ctx.Done():
		sctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return srv.Shutdown(sctx)
	case err := <-errc:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
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
		return errors.New("usage: gbi dev gen-client|fake-agent|sign|openapi")
	}
	switch args[0] {
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
