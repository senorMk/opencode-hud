package service

import (
	"database/sql"
	"time"

	"github.com/parentalk/opencode-hud/internal/store"
)

// Totals aggregates cost + tokens over any set of rows.
type Totals struct {
	Cost      float64 `json:"cost"`
	Input     int64   `json:"input"`
	Output    int64   `json:"output"`
	Reasoning int64   `json:"reasoning"`
	CacheRead int64   `json:"cacheRead"`
	Messages  int64   `json:"messages"`
	Sessions  int64   `json:"sessions"`
}

// DayReport is the "today" view: sessions plus rolled-up totals.
type DayReport struct {
	Day      string          `json:"day"`
	Totals   Totals          `json:"totals"`
	Sessions []store.Session `json:"sessions"`
}

// Today loads all sessions updated on the local calendar day.
func Today(db *sql.DB, now time.Time) (DayReport, error) {
	day := now.Format("2006-01-02")
	sessions, err := store.DaySessions(db, day)
	if err != nil {
		return DayReport{}, err
	}
	rep := DayReport{Day: day, Sessions: sessions}
	for _, s := range sessions {
		rep.Totals.Cost += s.Cost
		rep.Totals.Input += s.Input
		rep.Totals.Output += s.Output
		rep.Totals.Reasoning += s.Reasoning
		rep.Totals.CacheRead += s.CacheRead
		rep.Totals.Sessions++
	}
	return rep, nil
}
