// Package proposals holds improvement proposals agents make from history analysis (NFR-13).
// Accepted proposals become tasks on the go-build-it project and go through normal review.
package proposals

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/techmuch/go-build-it/internal/access"
	"github.com/techmuch/go-build-it/internal/history"
	"github.com/techmuch/go-build-it/internal/store"
)

// Proposal is a suggested change to prompts, personas, templates or code.
type Proposal struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Body      string `json:"body"`
	Evidence  string `json:"evidence"`
	Author    string `json:"author"`
	Status    string `json:"status"`
	CreatedAt string `json:"createdAt"`
}

// Service manages proposals.
type Service struct {
	db       *sql.DB
	hist     *history.Recorder
	OnAccept func(ctx context.Context, p Proposal) (string, error) // returns the task created
}

// New returns a proposal service.
func New(db *sql.DB, h *history.Recorder) *Service { return &Service{db: db, hist: h} }

// Create records a proposal; evidence is required (NFR-13 bases proposals on data).
func (s *Service) Create(ctx context.Context, title, body, evidence string) (Proposal, error) {
	if title == "" || body == "" || evidence == "" {
		return Proposal{}, errors.New("title, body and evidence are required")
	}
	id, err := store.NextID(s.db, "IP")
	if err != nil {
		return Proposal{}, err
	}
	p := Proposal{ID: id, Title: title, Body: body, Evidence: evidence, Author: access.FromContext(ctx).Name, Status: "open", CreatedAt: store.Now()}
	if _, err := s.db.ExecContext(ctx, `INSERT INTO proposals(id,title,body,evidence,author,status,created_at) VALUES(?,?,?,?,?,?,?)`,
		p.ID, p.Title, p.Body, p.Evidence, p.Author, p.Status, p.CreatedAt); err != nil {
		return p, err
	}
	s.hist.Must(ctx, history.Event{Type: "proposal.created", Subject: id, Summary: fmt.Sprintf("%s proposed %s: %s", p.Author, id, title), Why: evidence})
	return p, nil
}

// List returns proposals, open first.
func (s *Service) List(ctx context.Context) ([]Proposal, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id,title,body,evidence,author,status,created_at FROM proposals ORDER BY status='open' DESC, created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Proposal{}
	for rows.Next() {
		var p Proposal
		if err := rows.Scan(&p.ID, &p.Title, &p.Body, &p.Evidence, &p.Author, &p.Status, &p.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// Decide accepts or rejects a proposal. Accepting creates a task that goes through normal review.
func (s *Service) Decide(ctx context.Context, id, decision, note string) (Proposal, string, error) {
	var p Proposal
	err := s.db.QueryRowContext(ctx, `SELECT id,title,body,evidence,author,status,created_at FROM proposals WHERE id=?`, id).
		Scan(&p.ID, &p.Title, &p.Body, &p.Evidence, &p.Author, &p.Status, &p.CreatedAt)
	if err != nil {
		return p, "", fmt.Errorf("proposal %s: %w", id, err)
	}
	if decision != "accepted" && decision != "rejected" {
		return p, "", errors.New("decision must be accepted or rejected")
	}
	p.Status = decision
	if _, err := s.db.ExecContext(ctx, `UPDATE proposals SET status=? WHERE id=?`, decision, id); err != nil {
		return p, "", err
	}
	task := ""
	if decision == "accepted" && s.OnAccept != nil {
		task, err = s.OnAccept(ctx, p)
		if err != nil {
			return p, "", err
		}
	}
	s.hist.Must(ctx, history.Event{Type: "proposal." + decision, Subject: id, Summary: fmt.Sprintf("%s %s", id, decision), Why: note, Data: history.Data(map[string]string{"task": task})})
	return p, task, nil
}
