package service

import (
	"database/sql"
	"time"

	"github.com/senorMk/opencode-hud/internal/store"
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

// DayReport is the per-day view: sessions plus rolled-up totals.
type DayReport struct {
	Day      string          `json:"day"`
	Totals   Totals          `json:"totals"`
	Sessions []store.Session `json:"sessions"`
}

// Today loads the report for the local calendar day containing now.
func Today(db *sql.DB, now time.Time) (DayReport, error) {
	return ForDay(db, now.Format("2006-01-02"))
}

// ForDay loads the report for an arbitrary local calendar day (YYYY-MM-DD).
func ForDay(db *sql.DB, day string) (DayReport, error) {
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
	msgs, err := store.CountMessagesOnDay(db, day)
	if err != nil {
		return DayReport{}, err
	}
	rep.Totals.Messages = msgs
	return rep, nil
}
