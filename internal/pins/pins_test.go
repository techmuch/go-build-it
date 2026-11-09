package pins

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/techmuch/go-build-it/internal/access"
	"github.com/techmuch/go-build-it/internal/history"
	"github.com/techmuch/go-build-it/internal/store"
)

func TestPinsAddListRemove(t *testing.T) {
	db, err := store.Open(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	s := New(db, history.New(db))
	ctx := access.WithActor(context.Background(), access.Actor{Name: "david", Kind: "human", Origin: access.Local})

	for _, bad := range []Pin{
		{Item: "a.html", X: 1, Y: 1, Text: "x"},
		{Subject: "Q-1", Item: "a.html", X: 1, Y: 1, Text: "  "},
		{Subject: "Q-1", Item: "a.html", X: 101, Y: 1, Text: "x"},
	} {
		if _, err := s.Add(ctx, bad); err == nil {
			t.Fatalf("expected error for %+v", bad)
		}
	}
	p, err := s.Add(ctx, Pin{Subject: "Q-1", Item: "/raw/p/a.html", ProjectID: "p", X: 25.5, Y: 40, Text: " Move this up "})
	if err != nil {
		t.Fatal(err)
	}
	if p.ID == 0 || p.Author != "david" || p.Text != "Move this up" {
		t.Fatalf("pin %+v", p)
	}
	if _, err := s.Add(ctx, Pin{Subject: "Q-1", Item: "/raw/p/b.json", X: 1, Y: 1, Text: "other item"}); err != nil {
		t.Fatal(err)
	}
	got, _ := s.List(ctx, "Q-1", "/raw/p/a.html")
	if len(got) != 1 || got[0].X != 25.5 {
		t.Fatalf("list item = %+v", got)
	}
	if all, _ := s.List(ctx, "Q-1", ""); len(all) != 2 {
		t.Fatalf("list subject = %d pins", len(all))
	}
	if err := s.Remove(ctx, p.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.Remove(ctx, p.ID); err == nil {
		t.Fatal("removing twice should fail")
	}
	if got, _ := s.List(ctx, "Q-1", "/raw/p/a.html"); len(got) != 0 {
		t.Fatalf("after remove = %+v", got)
	}
	var n int
	_ = db.QueryRow(`SELECT COUNT(*) FROM events WHERE type LIKE 'pin.%'`).Scan(&n)
	if n != 3 {
		t.Fatalf("history events = %d, want 3", n)
	}
}
