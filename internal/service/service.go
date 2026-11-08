// Package service installs go-build-it as a background service: launchd on macOS,
// a systemd user unit on Linux, a logon scheduled task on Windows (FR-02, NFR-01).
package service

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

const label = "dev.gobuildit.gbi"

// Plan describes what Install will do, so it can be shown before running.
type Plan struct {
	OS      string   `json:"os"`
	File    string   `json:"file"`
	Content string   `json:"content"`
	Start   []string `json:"start"`
	Stop    []string `json:"stop"`
}

// Build returns the install plan for this OS.
func Build(exe, home string) (Plan, error) {
	userHome, _ := os.UserHomeDir()
	path := ServicePath(os.Getenv("PATH"), userHome)
	switch runtime.GOOS {
	case "darwin":
		file := filepath.Join(userHome, "Library", "LaunchAgents", label+".plist")
		content := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict>
  <key>Label</key><string>%s</string>
  <key>ProgramArguments</key><array><string>%s</string><string>serve</string></array>
  <key>EnvironmentVariables</key><dict><key>GBI_HOME</key><string>%s</string><key>PATH</key><string>%s</string></dict>
  <key>RunAtLoad</key><true/>
  <key>KeepAlive</key><true/>
  <key>StandardOutPath</key><string>%s</string>
  <key>StandardErrorPath</key><string>%s</string>
</dict></plist>
`, label, exe, home, xmlEscape(path), filepath.Join(home, "gbi.log"), filepath.Join(home, "gbi.log"))
		uid := fmt.Sprint(os.Getuid())
		return Plan{OS: "darwin", File: file, Content: content,
			Start: []string{"launchctl", "bootstrap", "gui/" + uid, file},
			Stop:  []string{"launchctl", "bootout", "gui/" + uid + "/" + label}}, nil
	case "linux":
		file := filepath.Join(userHome, ".config", "systemd", "user", "gbi.service")
		content := fmt.Sprintf("[Unit]\nDescription=go-build-it\n\n[Service]\nExecStart=%s serve\nEnvironment=GBI_HOME=%s\nEnvironment=\"PATH=%s\"\nRestart=on-failure\n\n[Install]\nWantedBy=default.target\n", exe, home, strings.ReplaceAll(path, "%", "%%"))
		return Plan{OS: "linux", File: file, Content: content,
			Start: []string{"systemctl", "--user", "enable", "--now", "gbi.service"},
			Stop:  []string{"systemctl", "--user", "disable", "--now", "gbi.service"}}, nil
	case "windows":
		return Plan{OS: "windows", File: "", Content: "",
			Start: []string{"schtasks", "/Create", "/F", "/SC", "ONLOGON", "/TN", "go-build-it", "/TR", `"` + exe + `" serve`},
			Stop:  []string{"schtasks", "/Delete", "/F", "/TN", "go-build-it"}}, nil
	}
	return Plan{}, fmt.Errorf("background service not supported on %s", runtime.GOOS)
}

// ServicePath is the PATH a login service runs with. launchd and systemd
// start services with a minimal PATH, so harness CLIs installed in places
// like ~/.local/bin would not be found. It keeps the installer's own PATH,
// then makes sure the usual user and system bin directories are present.
func ServicePath(current, userHome string) string {
	var out []string
	seen := map[string]bool{}
	add := func(d string) {
		if d != "" && !seen[d] {
			seen[d] = true
			out = append(out, d)
		}
	}
	for _, d := range filepath.SplitList(current) {
		add(d)
	}
	extra := []string{"/usr/local/bin", "/usr/bin", "/bin"}
	if runtime.GOOS == "darwin" {
		extra = append([]string{"/opt/homebrew/bin"}, extra...)
	}
	if userHome != "" {
		extra = append([]string{filepath.Join(userHome, ".local", "bin"), filepath.Join(userHome, "go", "bin")}, extra...)
	}
	for _, d := range extra {
		add(d)
	}
	return strings.Join(out, string(os.PathListSeparator))
}

func xmlEscape(s string) string {
	return strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;").Replace(s)
}

// Install writes the unit file and starts the service.
func Install(p Plan) error {
	if p.File != "" {
		if err := os.MkdirAll(filepath.Dir(p.File), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(p.File, []byte(p.Content), 0o644); err != nil {
			return err
		}
	}
	if p.OS == "linux" {
		_ = exec.Command("systemctl", "--user", "daemon-reload").Run()
	}
	return runCmd(p.Start)
}

// Uninstall stops the service and removes the unit file.
func Uninstall(p Plan) error {
	err := runCmd(p.Stop)
	if p.File != "" {
		_ = os.Remove(p.File)
	}
	return err
}

func runCmd(c []string) error {
	out, err := exec.Command(c[0], c[1:]...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %v: %s", strings.Join(c, " "), err, strings.TrimSpace(string(out)))
	}
	return nil
}
