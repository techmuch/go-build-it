//go:build windows

package update

import (
	"os"
	"os/exec"
	"syscall"
)

// Exec starts exe detached with the same arguments and exits; Windows cannot replace a
// running process. The new process retries binding the port until this one is gone.
func Exec(exe string, args []string) error {
	cmd := exec.Command(exe, args...)
	cmd.Stdout, cmd.Stderr, cmd.Env = os.Stdout, os.Stderr, os.Environ()
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: 0x00000008 | syscall.CREATE_NEW_PROCESS_GROUP} // DETACHED_PROCESS
	if err := cmd.Start(); err != nil {
		return err
	}
	os.Exit(0)
	return nil
}
