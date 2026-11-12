package app

import (
	"fmt"

	"github.com/techmuch/go-build-it/internal/api"
	"github.com/techmuch/go-build-it/internal/history"
	"github.com/techmuch/go-build-it/internal/tailscale"
)

// Remote access over Tailscale Serve (FR-20, FR-20a–FR-20d, ADR-015).

func (a *App) ownerOnly(c *api.Ctx, what string) error {
	if c.Actor.Kind != "human" || !a.Cfg.IsOwner(c.Actor.Name, c.Actor.Login) {
		return api.Forbidden("only the owner can %s", what)
	}
	return nil
}

func (a *App) tailscaleSetup(c *api.Ctx) (any, error) {
	if err := a.ownerOnly(c, "change how go-build-it is served"); err != nil {
		return nil, err
	}
	var b struct {
		Port    int
		Confirm bool
	}
	if err := c.Bind(&b); err != nil {
		return nil, err
	}
	return a.tailscaleRecord(c, "tailscale.served", "Serving go-build-it on the tailnet")(tailscale.Setup(c.R.Context(), a.Cfg.Listen, a.Cfg.Dir(), b.Port, b.Confirm))
}

func (a *App) tailscaleStop(c *api.Ctx) (any, error) {
	if err := a.ownerOnly(c, "change how go-build-it is served"); err != nil {
		return nil, err
	}
	return a.tailscaleRecord(c, "tailscale.stopped", "Stopped serving go-build-it on the tailnet")(tailscale.Stop(c.R.Context(), a.Cfg.Listen, a.Cfg.Dir()))
}

// tailscaleRecord turns a Setup or Stop result into a response and records what changed.
func (a *App) tailscaleRecord(c *api.Ctx, typ, summary string) func(tailscale.Status, error) (any, error) {
	return func(s tailscale.Status, err error) (any, error) {
		if err != nil {
			return nil, bad(err)
		}
		if typ == "tailscale.served" && s.ServeURL != "" {
			summary += " at " + s.ServeURL
		}
		a.Hist.Must(c.R.Context(), history.Event{Type: typ, Summary: summary, Why: "Settings → Remote access",
			Data: history.Data(map[string]any{"url": s.ServeURL, "reachable": s.Reachable, "private": s.Private})})
		return s, nil
	}
}

// tailscaleQR returns the tailnet URL drawn as a QR code for a terminal.
func (a *App) tailscaleQR(c *api.Ctx) (any, error) {
	s := tailscale.Detect(c.R.Context(), a.Cfg.Listen, a.Cfg.Dir())
	if len(s.QR) == 0 {
		return nil, api.BadRequest("no working tailnet address yet: %s", firstNonEmpty(s.Guidance, "see gbi tailscale status"))
	}
	return fmt.Sprintf("%s\nOpen %s on a phone signed in to the same tailnet.", tailscale.Terminal(s.QR), s.PhoneURL), nil
}
