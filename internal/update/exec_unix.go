//go:build !windows

package update

import (
	"os"
	"syscall"
)

// Exec replaces the running process with exe, keeping its PID, arguments and environment,
// so launchd and systemd see no exit (ADR-011).
func Exec(exe string, args []string) error {
	return syscall.Exec(exe, append([]string{exe}, args...), os.Environ())
}
