// Package access classifies how each request reaches go-build-it and who made it (FR-15–FR-19).
package access

import (
	"context"
	"crypto/subtle"
	"net"
	"net/http"
	"strings"
)

// Origin is how a request reached the app.
type Origin string

const (
	Local     Origin = "local"     // loopback, no Tailscale identity (FR-16)
	Tailscale Origin = "tailscale" // loopback via Tailscale Serve with identity headers (FR-17)
	External  Origin = "external"  // any other interface (FR-18)
	System    Origin = "system"    // internal actions, not from a request
)

// Actor identifies who performed an action. Personas and the system use Kind "persona" / "system".
type Actor struct {
	Name   string `json:"name"`
	Kind   string `json:"kind"` // human | persona | system
	Origin Origin `json:"origin"`
	Login  string `json:"login,omitempty"`
}

// Classify determines the origin and actor for r. Local processes can set the Tailscale
// header; that is acceptable because local requests are trusted anyway (FR-16).
func Classify(r *http.Request, defaultUser string) Actor {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	ip := net.ParseIP(strings.Trim(host, "[]"))
	if ip != nil && ip.IsLoopback() {
		if login := r.Header.Get("Tailscale-User-Login"); login != "" {
			name := r.Header.Get("Tailscale-User-Name")
			if name == "" {
				name = login
			}
			return Actor{Name: name, Kind: "human", Origin: Tailscale, Login: login}
		}
		if p := r.Header.Get("X-GBI-Persona"); p != "" {
			// CLI calls made by agent sessions identify their persona (FR-30).
			return Actor{Name: p, Kind: "persona", Origin: Local}
		}
		return Actor{Name: defaultUser, Kind: "human", Origin: Local}
	}
	return Actor{Name: "unknown", Kind: "human", Origin: External}
}

type ctxKey struct{}

// WithActor stores the actor in ctx.
func WithActor(ctx context.Context, a Actor) context.Context {
	return context.WithValue(ctx, ctxKey{}, a)
}

// FromContext returns the actor stored in ctx, or the system actor.
func FromContext(ctx context.Context) Actor {
	if a, ok := ctx.Value(ctxKey{}).(Actor); ok {
		return a
	}
	return Actor{Name: "go-build-it", Kind: "system", Origin: System}
}

// Persona returns a context acting as the named persona (used by the dispatcher and Courier).
func Persona(ctx context.Context, name string) context.Context {
	return WithActor(ctx, Actor{Name: name, Kind: "persona", Origin: System})
}

// External describes how external requests are handled (FR-18, ADR-005, P9-07).
type ExternalPolicy struct {
	Allow bool
	Token string // required when Allow is true; empty means external access stays refused
}

// Middleware classifies every request. External requests are refused unless external access
// is allowed AND the request presents the token: an `Authorization: Bearer` header, a
// `gbi_token` cookie, or `?token=` (which sets the cookie).
func Middleware(defaultUser string, external func() ExternalPolicy, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		a := Classify(r, defaultUser)
		if a.Origin == External {
			ext := external()
			if !ext.Allow || ext.Token == "" {
				http.Error(w, "external access is disabled: use Tailscale Serve (see Settings → Access)", http.StatusForbidden)
				return
			}
			got := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
			if c, err := r.Cookie("gbi_token"); err == nil && got == "" {
				got = c.Value
			}
			if q := r.URL.Query().Get("token"); q != "" {
				got = q
			}
			if subtle.ConstantTimeCompare([]byte(got), []byte(ext.Token)) != 1 {
				w.Header().Set("WWW-Authenticate", `Bearer realm="go-build-it"`)
				http.Error(w, "a token is required for external access", http.StatusUnauthorized)
				return
			}
			if r.URL.Query().Get("token") != "" {
				http.SetCookie(w, &http.Cookie{Name: "gbi_token", Value: got, Path: "/", HttpOnly: true, SameSite: http.SameSiteStrictMode, Secure: r.TLS != nil})
			}
			a = Actor{Name: "external", Kind: "human", Origin: External}
		}
		next.ServeHTTP(w, r.WithContext(WithActor(r.Context(), a)))
	})
}
