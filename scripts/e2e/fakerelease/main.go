// Command fakerelease is a stand-in for GitHub Releases, used by scripts/update_e2e.sh to
// test a real update and restart without publishing anything (ADR-011).
//
//	fakerelease keygen                         prints "PUBLIC PRIVATE" (base64 Ed25519)
//	fakerelease serve ADDR DIR TAG [OLDER...]  serves DIR's files as the assets of TAG
//
// Besides the API listing (/repos/…/releases) it serves GitHub's plain download paths,
// /OWNER/REPO/releases/latest/download/NAME and /OWNER/REPO/releases/download/TAG/NAME,
// which scripts/install.sh uses (scripts/install_e2e.sh, ADR-012).
package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("usage: fakerelease keygen | serve ADDR DIR TAG [OLDER...]")
	}
	switch os.Args[1] {
	case "keygen":
		pub, priv, _ := ed25519.GenerateKey(rand.Reader)
		fmt.Println(base64.StdEncoding.EncodeToString(pub), base64.StdEncoding.EncodeToString(priv))
	case "serve":
		serve(os.Args[2], os.Args[3], os.Args[4], os.Args[5:])
	default:
		log.Fatalf("unknown command %q", os.Args[1])
	}
}

func serve(addr, dir, tag string, older []string) {
	base := "http://" + addr
	entries, err := os.ReadDir(dir)
	if err != nil {
		log.Fatal(err)
	}
	var assets []map[string]string
	for _, e := range entries {
		assets = append(assets, map[string]string{"name": e.Name(), "browser_download_url": base + "/dl/" + e.Name()})
	}
	notes := func(t string) string {
		return fmt.Sprintf("### Added\n\n- Everything new in %s (fake release for the update e2e)\n\n### Fixed\n\n- A bug fixed in %s", t, t)
	}
	releases := []map[string]any{{"tag_name": tag, "body": notes(tag), "published_at": time.Now().UTC().Format(time.RFC3339), "html_url": base + "/r/" + tag, "assets": assets}}
	for _, t := range older {
		releases = append(releases, map[string]any{"tag_name": t, "body": notes(t), "published_at": time.Now().Add(-72 * time.Hour).UTC().Format(time.RFC3339), "html_url": base + "/r/" + t, "assets": []any{}})
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/", func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/releases") {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(releases)
	})
	download := func(w http.ResponseWriter, r *http.Request, name string) {
		b, err := os.ReadFile(filepath.Join(dir, filepath.Base(name)))
		if err != nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Length", fmt.Sprint(len(b)))
		_, _ = w.Write(b)
	}
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		p := r.URL.Path
		if !strings.HasPrefix(p, "/techmuch/go-build-it/releases/") {
			http.NotFound(w, r)
			return
		}
		if i := strings.Index(p, "/releases/latest/download/"); i >= 0 {
			download(w, r, p[i+len("/releases/latest/download/"):])
			return
		}
		if i := strings.Index(p, "/releases/download/"+tag+"/"); i >= 0 {
			download(w, r, p[i+len("/releases/download/"+tag+"/"):])
			return
		}
		http.NotFound(w, r)
	})
	mux.HandleFunc("/dl/", func(w http.ResponseWriter, r *http.Request) {
		// Serve slowly enough that the download step is seen running with progress.
		name := filepath.Base(r.URL.Path)
		b, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Length", fmt.Sprint(len(b)))
		chunk := len(b)/20 + 1
		for i := 0; i < len(b); i += chunk {
			end := min(i+chunk, len(b))
			if _, err := w.Write(b[i:end]); err != nil {
				return
			}
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
			if len(b) > 1<<20 {
				time.Sleep(60 * time.Millisecond)
			}
		}
	})
	log.Printf("fake releases for %s on %s", tag, base)
	log.Fatal(http.ListenAndServe(addr, mux))
}
