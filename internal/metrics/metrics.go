// Package metrics computes performance figures agents can analyze (NFR-12).
package metrics

import (
	"context"
	"database/sql"
	"sort"
	"time"
)

// Metrics are computed over a window of days.
type Metrics struct {
	Days                int     `json:"days"`
	TasksDone           int     `json:"tasksDone"`
	CompactionsPerTask  float64 `json:"compactionsPerTask"`
	HandoffsPerTask     float64 `json:"handoffsPerTask"`
	ReworkRate          float64 `json:"reworkRate"`        // reviews with changes / decided reviews
	MedianAnswerHours   float64 `json:"medianAnswerHours"` // blocking questions
	MedianTaskMinutes   float64 `json:"medianTaskMinutes"` // created → done
	QuestionsAsked      int     `json:"questionsAsked"`
	RateLimitPauses     int     `json:"rateLimitPauses"`
	SessionsRun         int     `json:"sessionsRun"`
	SessionsFailed      int     `json:"sessionsFailed"`
	EstimatedVsMeasured string  `json:"estimatedVsMeasured"`
}

// Compute returns metrics for the last days (optionally one project).
func Compute(ctx context.Context, db *sql.DB, projectID string, days int) (Metrics, error) {
	if days <= 0 {
		days = 7
	}
	m := Metrics{Days: days}
	since := time.Now().UTC().AddDate(0, 0, -days).Format(time.RFC3339)
	pf, args := "", []any{since}
	if projectID != "" {
		pf, args = " AND project_id=?", append(args, projectID)
	}
	one := func(q string, dst any) { _ = db.QueryRowContext(ctx, q, args...).Scan(dst) }
	one(`SELECT COUNT(*) FROM tasks WHERE status='done' AND updated_at>=?`+pf, &m.TasksDone)
	var comp, hand sql.NullFloat64
	_ = db.QueryRowContext(ctx, `SELECT AVG(compactions), AVG(handoffs) FROM tasks WHERE status='done' AND updated_at>=?`+pf, args...).Scan(&comp, &hand)
	m.CompactionsPerTask, m.HandoffsPerTask = comp.Float64, hand.Float64
	var decided, changes int
	one(`SELECT COUNT(*) FROM reviews WHERE status!='pending' AND decided_at>=?`+pf, &decided)
	one(`SELECT COUNT(*) FROM reviews WHERE status='changes' AND decided_at>=?`+pf, &changes)
	if decided > 0 {
		m.ReworkRate = float64(changes) / float64(decided)
	}
	one(`SELECT COUNT(*) FROM questions WHERE created_at>=?`+pf, &m.QuestionsAsked)
	one(`SELECT COUNT(*) FROM events WHERE type='session.rate_limited' AND ts>=?`+pf, &m.RateLimitPauses)
	one(`SELECT COUNT(*) FROM sessions WHERE started_at>=?`+pf, &m.SessionsRun)
	one(`SELECT COUNT(*) FROM sessions WHERE status='failed' AND started_at>=?`+pf, &m.SessionsFailed)
	m.MedianAnswerHours = median(durations(ctx, db, `SELECT created_at, answered_at FROM questions WHERE blocking=1 AND status='answered' AND created_at>=?`+pf, args), time.Hour)
	m.MedianTaskMinutes = median(durations(ctx, db, `SELECT created_at, updated_at FROM tasks WHERE status='done' AND updated_at>=?`+pf, args), time.Minute)
	var est, meas int
	one(`SELECT COUNT(*) FROM sessions WHERE measured=0 AND started_at>=?`+pf, &est)
	one(`SELECT COUNT(*) FROM sessions WHERE measured=1 AND started_at>=?`+pf, &meas)
	m.EstimatedVsMeasured = itoa(est) + " estimated / " + itoa(meas) + " measured"
	return m, nil
}

func durations(ctx context.Context, db *sql.DB, q string, args []any) []time.Duration {
	rows, err := db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []time.Duration
	for rows.Next() {
		var a, b string
		if rows.Scan(&a, &b) != nil {
			continue
		}
		ta, e1 := time.Parse(time.RFC3339, a)
		tb, e2 := time.Parse(time.RFC3339, b)
		if e1 == nil && e2 == nil && tb.After(ta) {
			out = append(out, tb.Sub(ta))
		}
	}
	return out
}

func median(d []time.Duration, unit time.Duration) float64 {
	if len(d) == 0 {
		return 0
	}
	sort.Slice(d, func(i, j int) bool { return d[i] < d[j] })
	return float64(d[len(d)/2]) / float64(unit)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	s := ""
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	return s
}

// Group summarizes done tasks that share a context strategy.
type Group struct {
	Name       string  `json:"name"`
	Tasks      int     `json:"tasks"`
	ReworkRate float64 `json:"reworkRate"` // reviews requesting changes per task
	AvgRetries float64 `json:"avgRetries"`
}

// Comparison contrasts smart handoffs with compaction (P9-02, Q3).
type Comparison struct {
	Days     int     `json:"days"`
	Groups   []Group `json:"groups"`
	Ready    bool    `json:"ready"`
	Criteria string  `json:"criteria"`
	Verdict  string  `json:"verdict"`
}

// Compare groups done tasks by handoffs vs compactions vs neither.
func Compare(ctx context.Context, db *sql.DB, days int) (Comparison, error) {
	if days <= 0 {
		days = 30
	}
	since := time.Now().UTC().AddDate(0, 0, -days).Format(time.RFC3339)
	c := Comparison{Days: days, Criteria: "Graduate when at least 10 handed-off tasks exist and their rework rate is no worse than compaction-only tasks (ADR-007)."}
	defs := []struct{ name, where string }{
		{"Handed off", "t.handoffs > 0"},
		{"Compacted only", "t.handoffs = 0 AND t.compactions > 0"},
		{"Fit in one session", "t.handoffs = 0 AND t.compactions = 0"},
	}
	for _, d := range defs {
		g := Group{Name: d.name}
		var retries sql.NullFloat64
		var changes int
		_ = db.QueryRowContext(ctx, `SELECT COUNT(*), AVG(t.retries) FROM tasks t WHERE t.status='done' AND t.updated_at>=? AND `+d.where, since).Scan(&g.Tasks, &retries)
		_ = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM reviews r JOIN tasks t ON t.id=r.task_id WHERE r.status='changes' AND t.status='done' AND t.updated_at>=? AND `+d.where, since).Scan(&changes)
		g.AvgRetries = retries.Float64
		if g.Tasks > 0 {
			g.ReworkRate = float64(changes) / float64(g.Tasks)
		}
		c.Groups = append(c.Groups, g)
	}
	h, comp := c.Groups[0], c.Groups[1]
	switch {
	case h.Tasks < 10:
		c.Verdict = "Not enough handed-off tasks yet to judge."
	case comp.Tasks == 0 || h.ReworkRate <= comp.ReworkRate:
		c.Ready = true
		c.Verdict = "Handoffs perform at least as well as compaction: ready for a graduation review."
	default:
		c.Verdict = "Handed-off tasks need more rework than compacted ones: keep handoffs experimental."
	}
	return c, nil
}
