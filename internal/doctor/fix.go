package doctor

import (
	"context"
	"fmt"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/techmuch/go-build-it/internal/service"
)

// Apply runs one automatic fix (FR-07c). Actions:
//
//	start-session                 start the gbi herdr session
//	install-integration:<target>  install or update herdr's integration for a harness
//	disable-harness:<name>        turn off a harness that is not installed
//	set-binary:<name>=<path>      use a harness's full path (not on the panes' PATH)
//	service-reinstall             rewrite the login service with the current PATH
func (d *Doctor) Apply(ctx context.Context, action string) error {
	name, arg, _ := strings.Cut(action, ":")
	switch name {
	case "start-session":
		_, err := d.Sup.Ensure(ctx)
		return err
	case "install-integration":
		for _, p := range d.probes("") {
			if p.integration == arg {
				return d.Sup.C.InstallIntegration(ctx, arg)
			}
		}
		return fmt.Errorf("unknown integration %q", arg)
	case "disable-harness":
		hc, ok := d.Cfg.Harnesses[arg]
		if !ok {
			return fmt.Errorf("unknown harness %q", arg)
		}
		hc.Enabled = false
		d.Cfg.Harnesses[arg] = hc
		return d.Cfg.Save()
	case "set-binary":
		h, path, ok := strings.Cut(arg, "=")
		hc, known := d.Cfg.Harnesses[h]
		if !ok || !known || path == "" {
			return fmt.Errorf("bad fix %q", action)
		}
		if _, err := lookPath(path); err != nil {
			return err
		}
		hc.Binary = path
		d.Cfg.Harnesses[h] = hc
		return d.Cfg.Save()
	case "service-reinstall":
		p, err := d.servicePlan()
		if err != nil {
			return err
		}
		return service.Install(p)
	}
	return fmt.Errorf("unknown fix %q", action)
}

// FixAll applies every automatic fix in rep and returns what it did.
func (d *Doctor) FixAll(ctx context.Context, rep Report) (applied []string, errs []error) {
	seen := map[string]bool{}
	for _, r := range rep.Results {
		if r.Fix == nil || r.Fix.Kind != "auto" || (r.Status != Fail && r.Status != Warn) || seen[r.Fix.Action] {
			continue
		}
		seen[r.Fix.Action] = true
		if err := d.Apply(ctx, r.Fix.Action); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", r.Fix.Action, err))
			continue
		}
		applied = append(applied, r.Fix.Action)
	}
	return applied, errs
}

// Markdown renders rep for pasting into an AI chat or an issue (FR-07d). The
// configuration is included with secrets removed.
func (d *Doctor) Markdown(rep Report) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# go-build-it doctor report\n\n")
	fmt.Fprintf(&b, "- go-build-it: %s\n- platform: %s/%s\n- herdr: %s\n- ran: %s (%d ms)\n- result: **%s** (exit %d)\n\n", rep.Version, rep.OS, rep.Arch, orNone(rep.Herdr), rep.Started.Format("2006-01-02 15:04:05 MST"), rep.Ms, rep.Summary, rep.Exit)
	b.WriteString("## Tiers\n\n")
	for _, t := range rep.Tiers {
		if t.Ran {
			fmt.Fprintf(&b, "- %s: %d/%d ok\n", t.Tier, t.OK, t.Total)
		} else {
			fmt.Fprintf(&b, "- %s: %s\n", t.Tier, t.Note)
		}
	}
	b.WriteString("\n## Results\n\n")
	for _, r := range rep.Results {
		fmt.Fprintf(&b, "### %s %s (`%s`)\n\n%s\n\n", icon(r.Status), r.Title, r.ID, r.Detail)
		if r.Evidence != nil && (r.Evidence.Command != "" || r.Evidence.Output != "") {
			b.WriteString("```\n")
			if r.Evidence.Command != "" {
				fmt.Fprintf(&b, "$ %s\n", r.Evidence.Command)
			}
			if r.Evidence.Exit != nil {
				fmt.Fprintf(&b, "(exit %d)\n", *r.Evidence.Exit)
			}
			if r.Evidence.Output != "" {
				b.WriteString(r.Evidence.Output + "\n")
			}
			b.WriteString("```\n\n")
		}
		if r.Fix != nil {
			who := "Needs a human"
			if r.Fix.Kind == "auto" {
				who = "Automatic fix (`" + r.Fix.Action + "`, run `gbi doctor --fix`)"
			}
			fmt.Fprintf(&b, "**%s:** %s\n\n", who, r.Fix.Instructions)
		}
	}
	cfg := *d.Cfg
	cfg.ExternalToken = ""
	var cb strings.Builder
	_ = toml.NewEncoder(&cb).Encode(cfg)
	fmt.Fprintf(&b, "## Configuration (secrets removed)\n\n```toml\n%s```\n", cb.String())
	return b.String()
}

func orNone(s string) string {
	if s == "" {
		return "not found"
	}
	return s
}

func icon(s Status) string {
	return map[Status]string{OK: "✅", Warn: "⚠️", Fail: "✖", Skipped: "–"}[s]
}
