package herdr

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"sync"
)

// Frame is one terminal.frame from `herdr terminal session observe|control`:
// base64 ANSI bytes, a sequence number, and whether it is a full repaint.
type Frame struct {
	Type     string `json:"type"`
	Seq      int64  `json:"seq"`
	Full     bool   `json:"full"`
	Width    int    `json:"width"`
	Height   int    `json:"height"`
	Encoding string `json:"encoding"`
	Bytes    string `json:"bytes"`
	Reason   string `json:"reason,omitempty"`
}

// Terminal is a live stream of a pane (FR-66). Observe streams are read-only and
// several can run at once; a control stream also takes input, resizes and mouse events,
// and only one controller owns a pane at a time.
type Terminal struct {
	Control bool
	Frames  <-chan Frame
	cmd     *exec.Cmd
	stdin   io.WriteCloser
	mu      sync.Mutex
	done    chan struct{}
	err     error
}

// OpenTerminal starts `herdr terminal session observe|control <pane>` against the
// session. Control streams use --takeover: the human asked to take control.
func (s *Supervisor) OpenTerminal(ctx context.Context, pane string, control bool, cols, rows int) (*Terminal, error) {
	mode := "observe"
	if control {
		mode = "control"
	}
	// The target comes before the options (`herdr terminal session control <target> [--takeover] …`).
	args := []string{"--session", s.C.Session, "terminal", "session", mode, pane}
	if control {
		args = append(args, "--takeover")
	}
	if cols > 0 && rows > 0 {
		args = append(args, "--cols", strconv.Itoa(cols), "--rows", strconv.Itoa(rows))
	}
	cmd := exec.CommandContext(ctx, s.C.Bin, args...)
	cmd.Env = s.env()
	out, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	t := &Terminal{Control: control, cmd: cmd, done: make(chan struct{})}
	if control {
		if t.stdin, err = cmd.StdinPipe(); err != nil {
			return nil, err
		}
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("herdr terminal session %s: %w", mode, err)
	}
	frames := make(chan Frame, 64)
	t.Frames = frames
	go func() {
		defer close(t.done)
		defer close(frames)
		r := bufio.NewReaderSize(out, 4<<20)
		for {
			line, err := r.ReadBytes('\n')
			if len(line) > 0 {
				var f Frame
				if json.Unmarshal(line, &f) == nil {
					select {
					case frames <- f:
					case <-ctx.Done():
						return
					}
					if f.Type == "terminal.closed" {
						return
					}
				}
			}
			if err != nil {
				t.err = err
				return
			}
		}
	}()
	go func() { <-t.done; _ = cmd.Wait() }()
	return t, nil
}

func (t *Terminal) send(v any) error {
	if !t.Control || t.stdin == nil {
		return errors.New("this terminal stream is read-only; take control first")
	}
	b, _ := json.Marshal(v)
	t.mu.Lock()
	defer t.mu.Unlock()
	_, err := t.stdin.Write(append(b, '\n'))
	return err
}

// Input types raw bytes into the pane.
func (t *Terminal) Input(b []byte) error {
	return t.send(map[string]any{"type": "terminal.input", "bytes": base64.StdEncoding.EncodeToString(b)})
}

// Resize changes the pane size (control streams only; observers never resize).
func (t *Terminal) Resize(cols, rows int) error {
	return t.send(map[string]any{"type": "terminal.resize", "cols": cols, "rows": rows})
}

// Release hands control back.
func (t *Terminal) Release() error { return t.send(map[string]any{"type": "terminal.release"}) }

// Close ends the stream.
func (t *Terminal) Close() {
	if t.Control {
		_ = t.Release()
		if t.stdin != nil {
			_ = t.stdin.Close()
		}
	}
	if t.cmd.Process != nil {
		_ = t.cmd.Process.Kill()
	}
	<-t.done
}
