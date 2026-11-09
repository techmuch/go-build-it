//go:build windows

package herdr

import (
	"context"
	"io"
	"os"
	"strings"
)

// herdr listens on \\.\pipe\<socket path> on Windows (src/platform/windows.rs).
func dial(ctx context.Context, socket string) (io.ReadWriteCloser, error) {
	name := socket
	if !strings.HasPrefix(name, `\\.\pipe\`) {
		name = `\\.\pipe\` + socket
	}
	return os.OpenFile(name, os.O_RDWR, 0)
}
