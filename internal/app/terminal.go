package app

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/coder/websocket"

	"github.com/techmuch/go-build-it/internal/api"
	"github.com/techmuch/go-build-it/internal/herdr"
	"github.com/techmuch/go-build-it/internal/history"
)

// TerminalBudget is how many frame bytes may be in flight to a browser before the bridge
// stops forwarding (the browser acknowledges each frame once xterm.js has drawn it). After
// it catches up, the stream restarts to get a full repaint. Same idea as Ovis (P14-01).
const TerminalBudget = 1 << 20

type termMsg struct {
	T    string `json:"t"`
	Seq  int64  `json:"seq,omitempty"`
	B    string `json:"b,omitempty"` // base64 input bytes
	Cols int    `json:"cols,omitempty"`
	Rows int    `json:"rows,omitempty"`
}

// memberTerminal streams a member's herdr pane over a WebSocket (FR-66). mode=observe
// is read-only; mode=control takes control (herdr --takeover) for typing and resizing.
func (a *App) memberTerminal(c *api.Ctx) (any, error) {
	m, err := a.Team.Member(c.R.Context(), c.Param("mid"))
	if err != nil {
		return nil, notFound(err)
	}
	if m.PaneID == "" {
		return nil, api.BadRequest("%s has not started in herdr yet", m.Name)
	}
	control := c.Query("mode") == "control"
	cols, _ := strconv.Atoi(c.Query("cols"))
	rows, _ := strconv.Atoi(c.Query("rows"))
	ws, err := websocket.Accept(c.W, c.R, &websocket.AcceptOptions{OriginPatterns: []string{"*"}})
	if err != nil {
		return api.Raw{}, nil
	}
	defer ws.CloseNow()
	ws.SetReadLimit(1 << 20)
	ctx, cancel := context.WithCancel(c.R.Context())
	defer cancel()
	who := c.Actor.Name
	if control {
		a.Hist.Must(c.R.Context(), history.Event{ProjectID: m.ProjectID, Type: "terminal.control", Subject: m.ID, Summary: who + " took control of " + m.Name,
			Data: history.Data(map[string]any{"pane": m.PaneID})})
		start := time.Now()
		defer func() {
			a.Hist.Must(context.WithoutCancel(c.R.Context()), history.Event{ProjectID: m.ProjectID, Type: "terminal.released", Subject: m.ID,
				Summary: fmt.Sprintf("%s handed %s back after %s", who, m.Name, time.Since(start).Round(time.Second))})
		}()
	}
	var mu sync.Mutex
	var term *herdr.Terminal
	inflight, dropping := 0, false
	sizes := map[int64]int{}
	write := func(v any) error {
		b, _ := json.Marshal(v)
		wctx, wcancel := context.WithTimeout(ctx, 10*time.Second)
		defer wcancel()
		return ws.Write(wctx, websocket.MessageText, b)
	}
	open := func() (*herdr.Terminal, error) {
		t, err := a.Herdr.OpenTerminal(ctx, m.PaneID, control, cols, rows)
		if err != nil {
			return nil, err
		}
		mu.Lock()
		term, inflight, dropping = t, 0, false
		sizes = map[int64]int{}
		mu.Unlock()
		return t, nil
	}
	t, err := open()
	if err != nil {
		_ = write(map[string]any{"t": "error", "message": err.Error()})
		return api.Raw{}, nil
	}
	_ = write(map[string]any{"t": "mode", "mode": map[bool]string{true: "control", false: "observe"}[control], "pane": m.PaneID, "member": m.Name})
	restart := make(chan struct{}, 1)
	// Browser → herdr: acks, input, resize.
	go func() {
		defer cancel()
		for {
			_, data, err := ws.Read(ctx)
			if err != nil {
				return
			}
			var msg termMsg
			if json.Unmarshal(data, &msg) != nil {
				continue
			}
			mu.Lock()
			cur := term
			switch msg.T {
			case "ack":
				inflight -= sizes[msg.Seq]
				delete(sizes, msg.Seq)
				if inflight < 0 {
					inflight = 0
				}
				if dropping && inflight < TerminalBudget/4 {
					select {
					case restart <- struct{}{}:
					default:
					}
				}
			}
			mu.Unlock()
			switch msg.T {
			case "input":
				if b, err := base64.StdEncoding.DecodeString(msg.B); err == nil {
					_ = cur.Input(b)
				}
			case "resize":
				if msg.Cols > 0 && msg.Rows > 0 {
					cols, rows = msg.Cols, msg.Rows
					_ = cur.Resize(msg.Cols, msg.Rows)
				}
			}
		}
	}()
	// herdr → browser: frames within the budget.
	for {
		select {
		case <-ctx.Done():
			t.Close()
			return api.Raw{}, nil
		case <-restart:
			t.Close()
			if t, err = open(); err != nil {
				_ = write(map[string]any{"t": "error", "message": err.Error()})
				return api.Raw{}, nil
			}
		case f, ok := <-t.Frames:
			if !ok {
				_ = write(map[string]any{"t": "closed", "reason": "the herdr stream ended"})
				return api.Raw{}, nil
			}
			if f.Type == "terminal.closed" {
				_ = write(map[string]any{"t": "closed", "reason": f.Reason})
				t.Close()
				return api.Raw{}, nil
			}
			mu.Lock()
			if inflight+len(f.Bytes) > TerminalBudget && !f.Full {
				dropping = true
				mu.Unlock()
				continue
			}
			inflight += len(f.Bytes)
			sizes[f.Seq] = len(f.Bytes)
			mu.Unlock()
			if err := write(map[string]any{"t": "frame", "seq": f.Seq, "full": f.Full, "w": f.Width, "h": f.Height, "b": f.Bytes}); err != nil {
				if !errors.Is(err, context.Canceled) {
					a.Log.Debug("terminal write", "err", err)
				}
				t.Close()
				return api.Raw{}, nil
			}
		}
	}
}

// memberFocus focuses a member's pane in herdr, so `herdr --session gbi` opens on it (P14-06).
func (a *App) memberFocus(c *api.Ctx) (any, error) {
	m, err := a.Team.Member(c.R.Context(), c.Param("mid"))
	if err != nil {
		return nil, notFound(err)
	}
	if m.PaneID == "" {
		return nil, api.BadRequest("%s has not started in herdr yet", m.Name)
	}
	if err := a.Herdr.C.Call(c.R.Context(), herdr.MethodPaneFocus, map[string]any{"pane_id": m.PaneID}, nil); err != nil {
		return nil, bad(err)
	}
	return map[string]any{"command": "herdr --session " + a.Herdr.C.Session, "pane": m.PaneID}, nil
}
