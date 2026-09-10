package service

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/senorMk/opencode-hud/internal/store"
)

// ActiveWindow is how recently a session must have been updated
// to count as "active" in the menu bar.
const ActiveWindow = 5 * time.Minute

// MenuStatus is the glanceable state rendered into the menu bar.
type MenuStatus struct {
	Label   string `json:"label"`   // e.g. "$4.20 today · 2 active"
	Tooltip string `json:"tooltip"` // multi-line detail for menu/tooltip
	Active  int    `json:"active"`
}

// LoadToday loads the full "today" snapshot: sessions, message count,
// totals and the derived menu status.
func LoadToday(db *sql.DB, now time.Time, recent []store.Session) (DayReport, MenuStatus, error) {
	rep, err := Today(db, now)
	if err != nil {
		return DayReport{}, MenuStatus{}, err
	}
	return rep, Summarise(rep, recent, now), nil
}

// Summarise derives the menu-bar status from a day report plus the
// most recently updated sessions (for active detection).
func Summarise(rep DayReport, recent []store.Session, now time.Time) MenuStatus {
	var active []store.Session
	cutoff := now.Add(-ActiveWindow).UnixMilli()
	for _, s := range recent {
		if s.UpdatedMs >= cutoff {
			active = append(active, s)
		}
	}
	label := fmt.Sprintf("$%.2f today", rep.Totals.Cost)
	if len(active) > 0 {
		label += fmt.Sprintf(" · %d active", len(active))
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Today: $%.2f · %d sessions · %d messages\n", rep.Totals.Cost, rep.Totals.Sessions, rep.Totals.Messages)
	fmt.Fprintf(&b, "Tokens: %s in / %s out / %s cache",
		compact(rep.Totals.Input), compact(rep.Totals.Output), compact(rep.Totals.CacheRead))
	for _, s := range active {
		fmt.Fprintf(&b, "\n● %s (%s) — %s in / %s out / %s cache",
			s.DisplayTitle, s.Model, compact(s.Input), compact(s.Output), compact(s.CacheRead))
	}
	return MenuStatus{Label: label, Tooltip: b.String(), Active: len(active)}
}

// compact renders large token counts like 60.2M / 85K.
func compact(n int64) string {
	switch f := float64(n); {
	case f >= 1e9:
		return fmt.Sprintf("%.1fB", f/1e9)
	case f >= 1e6:
		return fmt.Sprintf("%.1fM", f/1e6)
	case f >= 1e3:
		return fmt.Sprintf("%.1fK", f/1e3)
	default:
		return fmt.Sprintf("%d", n)
	}
}
