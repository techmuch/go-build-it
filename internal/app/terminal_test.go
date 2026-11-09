package app

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"

	"github.com/techmuch/go-build-it/internal/queue"
)

func readFrames(t *testing.T, ctx context.Context, ws *websocket.Conn, want string) {
	t.Helper()
	var screen strings.Builder
	for {
		_, data, err := ws.Read(ctx)
		if err != nil {
			t.Fatalf("waiting for %q: %v\nscreen so far:\n%s", want, err, screen.String())
		}
		var m map[string]any
		_ = json.Unmarshal(data, &m)
		if m["t"] == "frame" {
			b, _ := base64.StdEncoding.DecodeString(m["b"].(string))
			screen.Write(b)
			ack, _ := json.Marshal(map[string]any{"t": "ack", "seq": m["seq"]})
			_ = ws.Write(ctx, websocket.MessageText, ack)
			if strings.Contains(screen.String(), want) {
				return
			}
		}
		if m["t"] == "error" || m["t"] == "closed" {
			t.Fatalf("stream ended: %v", m)
		}
	}
}

// P14-01: watch a member's pane live, take control, type, and the task finishes.
func TestMemberTerminalStream(t *testing.T) {
	e := setup(t)
	tk, _ := e.a.Queue.Add(e.ctx, queue.Task{ProjectID: "demo", Title: "Guarded change", Description: "FAKE:block"})
	s, err := e.a.Sessions.Start(e.ctx, tk.ID, "forge@demo")
	if err != nil {
		t.Fatal(err)
	}
	e.waitSession(s.ID, "blocked")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	base := "ws" + strings.TrimPrefix(e.srv.URL, "http") + "/api/members/forge@demo/terminal?cols=100&rows=30"
	obs, _, err := websocket.Dial(ctx, base+"&mode=observe", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer obs.CloseNow()
	readFrames(t, ctx, obs, "Allow edit")
	ctl, _, err := websocket.Dial(ctx, base+"&mode=control", nil)
	if err != nil {
		t.Fatal(err)
	}
	readFrames(t, ctx, ctl, "Allow edit")
	in, _ := json.Marshal(map[string]any{"t": "input", "b": base64.StdEncoding.EncodeToString([]byte("y\r"))})
	if err := ctl.Write(ctx, websocket.MessageText, in); err != nil {
		t.Fatal(err)
	}
	e.waitStatus(tk.ID, queue.Review)
	ctl.Close(websocket.StatusNormalClosure, "")
	time.Sleep(300 * time.Millisecond)
	evs, _ := e.a.Hist.List(e.ctx, historyQueryType("terminal."))
	if len(evs) < 2 { // control + released
		t.Fatalf("takeover not recorded: %d events", len(evs))
	}
}
