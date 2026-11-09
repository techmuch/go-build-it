package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/techmuch/go-build-it/internal/config"
	"github.com/techmuch/go-build-it/internal/doctor"
	"github.com/techmuch/go-build-it/internal/herdr"
)

// exitError carries a process exit code without printing anything more.
type exitError int

func (e exitError) Error() string { return fmt.Sprintf("exit %d", int(e)) }

// doctorCmd runs the doctor in this process, without the server (FR-07a, FR-07d).
func doctorCmd(args []string) error {
	fs := flag.NewFlagSet("doctor", flag.ContinueOnError)
	tier := fs.String("tier", "live", "last tier to run: static, live or harness")
	only := fs.String("harness", "", "only this harness (claude, antigravity, pi, opencode)")
	prompt := fs.Bool("prompt", false, "also send one short prompt to each harness (uses a few tokens)")
	fix := fs.Bool("fix", false, "apply every automatic fix, then check again")
	asJSON := fs.Bool("json", false, "print the results as JSON")
	report := fs.String("report", "", "write a Markdown report to this file (- for stdout)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	tierSet := false
	fs.Visit(func(f *flag.Flag) { tierSet = tierSet || f.Name == "tier" })
	if *only != "" && !tierSet {
		*tier = "harness" // --harness claude means: start it
	}
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	exe, _ := os.Executable()
	d := &doctor.Doctor{Cfg: cfg, Sup: herdr.NewSupervisor(cfg.Herdr.Binary, cfg.Herdr.Session, cfg.Herdr.ConfigHome, quietLogger()), Exe: exe, Version: version}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()
	opt := doctor.Options{Through: doctor.Tier(*tier), Harness: *only, Prompt: *prompt}

	rep := d.Run(ctx, opt)
	var applied []string
	if *fix {
		var errs []error
		applied, errs = d.FixAll(ctx, rep)
		for _, e := range errs {
			fmt.Fprintln(os.Stderr, "fix failed:", e)
		}
		if len(applied) > 0 {
			rep = d.Run(ctx, opt)
		}
		rep.Applied = applied
	}
	switch {
	case *report != "":
		md := d.Markdown(rep)
		if *report == "-" {
			fmt.Print(md)
		} else if err := os.WriteFile(*report, []byte(md), 0o644); err != nil {
			return err
		} else if !*asJSON {
			fmt.Fprintf(os.Stderr, "wrote %s\n", *report)
		}
		if *asJSON {
			_ = writeJSON(os.Stdout, rep)
		}
	case *asJSON:
		if err := writeJSON(os.Stdout, rep); err != nil {
			return err
		}
	default:
		printReport(os.Stdout, rep)
	}
	if rep.Exit != 0 {
		return exitError(rep.Exit)
	}
	return nil
}

func writeJSON(w io.Writer, v any) error {
	e := json.NewEncoder(w)
	e.SetIndent("", "  ")
	return e.Encode(v)
}

func quietLogger() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

func printReport(w io.Writer, rep doctor.Report) {
	for _, a := range rep.Applied {
		fmt.Fprintf(w, "fixed: %s\n", a)
	}
	var tiers []string
	for _, t := range rep.Tiers {
		switch {
		case t.Ran && t.OK == t.Total:
			tiers = append(tiers, fmt.Sprintf("✅ %s %d/%d", t.Tier, t.OK, t.Total))
		case t.Ran:
			tiers = append(tiers, fmt.Sprintf("✖ %s %d/%d", t.Tier, t.OK, t.Total))
		default:
			tiers = append(tiers, fmt.Sprintf("– %s (%s)", t.Tier, t.Note))
		}
	}
	fmt.Fprintf(w, "go-build-it doctor · %s\n%s\n\n", rep.Version, strings.Join(tiers, " · "))
	icon := map[doctor.Status]string{doctor.OK: "✅", doctor.Warn: "⚠️ ", doctor.Fail: "✖ ", doctor.Skipped: "– "}
	for _, r := range rep.Results {
		fmt.Fprintf(w, "%s %s: %s\n", icon[r.Status], r.Title, r.Detail)
		if r.Status != doctor.Fail && r.Status != doctor.Warn {
			continue
		}
		fmt.Fprintf(w, "   id: %s · help: gbi docs get troubleshooting\n", r.ID)
		if r.Evidence != nil {
			if r.Evidence.Command != "" {
				fmt.Fprintf(w, "   $ %s\n", r.Evidence.Command)
			}
			for _, l := range strings.Split(strings.TrimSpace(r.Evidence.Output), "\n") {
				if l != "" && r.Status == doctor.Fail {
					fmt.Fprintf(w, "   │ %s\n", l)
				}
			}
		}
		if r.Fix != nil {
			who := "needs you"
			if r.Fix.Kind == "auto" {
				who = "automatic (gbi doctor --fix)"
			}
			fmt.Fprintf(w, "   fix, %s: %s\n", who, r.Fix.Instructions)
		}
	}
	fmt.Fprintf(w, "\n%s (%.1fs)\n", rep.Summary, float64(rep.Ms)/1000)
}
