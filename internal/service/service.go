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

// RangeReport is the arbitrary-range view (e.g. a calendar month):
// sessions plus rolled-up totals over [From, To].
type RangeReport struct {
	From     string          `json:"from"`
	To       string          `json:"to"`
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
	rep.Totals = SumSessions(sessions)
	msgs, err := store.CountMessagesOnDay(db, day)
	if err != nil {
		return DayReport{}, err
	}
	rep.Totals.Messages = msgs
	return rep, nil
}

// ForRange loads the report for an inclusive local calendar range
// [from, to] (both YYYY-MM-DD). Callers must validate the format and
// that from <= to.
func ForRange(db *sql.DB, from, to string) (RangeReport, error) {
	sessions, err := store.RangeSessions(db, from, to)
	if err != nil {
		return RangeReport{}, err
	}
	rep := RangeReport{From: from, To: to, Sessions: sessions}
	rep.Totals = SumSessions(sessions)
	msgs, err := store.CountMessagesInRange(db, from, to)
	if err != nil {
		return RangeReport{}, err
	}
	rep.Totals.Messages = msgs
	return rep, nil
}

// SumSessions rolls up cost + tokens over a set of sessions.
func SumSessions(sessions []store.Session) Totals {
	var t Totals
	for _, s := range sessions {
		t.Cost += s.Cost
		t.Input += s.Input
		t.Output += s.Output
		t.Reasoning += s.Reasoning
		t.CacheRead += s.CacheRead
		t.Sessions++
	}
	return t
}
