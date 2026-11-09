//go:build !windows

package herdr

import (
	"context"
	"io"
	"net"
)

func dial(ctx context.Context, socket string) (io.ReadWriteCloser, error) {
	var d net.Dialer
	return d.DialContext(ctx, "unix", socket)
}
