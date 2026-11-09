// Package pins stores comments pinned to a spot on an attachment or mockup in the viewer
// (FR-110: comments attach to a specific artifact, screen or diff location; FR-102a).
package pins

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/techmuch/go-build-it/internal/access"
	"github.com/techmuch/go-build-it/internal/history"
	"github.com/techmuch/go-build-it/internal/store"
)

// Pin is one comment at a position on an item, in percent of the item's size.
type Pin struct {
	ID        int64   `json:"id"`
	Subject   string  `json:"subject"`
	Item      string  `json:"item"`
	ProjectID string  `json:"projectId"`
	X         float64 `json:"x"`
	Y         float64 `json:"y"`
	Text      string  `json:"text"`
	Author    string  `json:"author"`
	CreatedAt string  `json:"createdAt"`
}

// Service reads and writes pins.
type Service struct {
	db   *sql.DB
	hist *history.Recorder
}

// New returns a pin service.
func New(db *sql.DB, h *history.Recorder) *Service { return &Service{db: db, hist: h} }

// List returns the pins for an item (or every item of the subject when item is empty), oldest first.
func (s *Service) List(ctx context.Context, subject, item string) ([]Pin, error) {
	q := `SELECT id,subject,item,project_id,x,y,text,author,created_at FROM pins WHERE subject=?`
	args := []any{subject}
	if item != "" {
		q += ` AND item=?`
		args = append(args, item)
	}
	rows, err := s.db.QueryContext(ctx, q+` ORDER BY id`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Pin{}
	for rows.Next() {
		var p Pin
		if err := rows.Scan(&p.ID, &p.Subject, &p.Item, &p.ProjectID, &p.X, &p.Y, &p.Text, &p.Author, &p.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// Add stores a pin attributed to the actor in ctx.
func (s *Service) Add(ctx context.Context, p Pin) (Pin, error) {
	p.Text = strings.TrimSpace(p.Text)
	switch {
	case p.Subject == "" || p.Item == "":
		return p, errors.New("subject and item are required")
	case p.Text == "":
		return p, errors.New("text is required")
	case p.X < 0 || p.X > 100 || p.Y < 0 || p.Y > 100:
		return p, errors.New("x and y are percentages from 0 to 100")
	}
	p.Author = access.FromContext(ctx).Name
	p.CreatedAt = store.Now()
	res, err := s.db.ExecContext(ctx, `INSERT INTO pins(subject,item,project_id,x,y,text,author,created_at) VALUES(?,?,?,?,?,?,?,?)`,
		p.Subject, p.Item, p.ProjectID, p.X, p.Y, p.Text, p.Author, p.CreatedAt)
	if err != nil {
		return p, err
	}
	p.ID, _ = res.LastInsertId()
	s.hist.Must(ctx, history.Event{ProjectID: p.ProjectID, Type: "pin.added", Subject: p.Subject,
		Summary: fmt.Sprintf("Pinned a comment on %s: %q", itemName(p.Item), p.Text), Data: history.Data(p)})
	return p, nil
}

// Remove deletes a pin.
func (s *Service) Remove(ctx context.Context, id int64) error {
	var p Pin
	err := s.db.QueryRowContext(ctx, `SELECT subject,item,project_id,text FROM pins WHERE id=?`, id).Scan(&p.Subject, &p.Item, &p.ProjectID, &p.Text)
	if errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("pin %d not found", id)
	} else if err != nil {
		return err
	}
	if _, err := s.db.ExecContext(ctx, `DELETE FROM pins WHERE id=?`, id); err != nil {
		return err
	}
	s.hist.Must(ctx, history.Event{ProjectID: p.ProjectID, Type: "pin.removed", Subject: p.Subject,
		Summary: fmt.Sprintf("Removed a pinned comment on %s: %q", itemName(p.Item), p.Text)})
	return nil
}

func itemName(item string) string {
	if i := strings.LastIndex(item, "/"); i >= 0 && i < len(item)-1 {
		return item[i+1:]
	}
	return item
}
