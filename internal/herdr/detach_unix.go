//go:build !windows

package herdr

import (
	"os/exec"
	"syscall"
)

// detach puts the server in its own session so it survives go-build-it exiting.
func detach(cmd *exec.Cmd) { cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true} }
