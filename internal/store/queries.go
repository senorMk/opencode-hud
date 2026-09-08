package store

import (
	"database/sql"
	"encoding/json"
	"strings"
	"time"
)

// dayStartUnix returns local-midnight epoch seconds (days-1) days ago,
// i.e. the inclusive lower bound for a "last N days" window.
func dayStartUnix(days int) int64 {
	now := time.Now()
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	return start.AddDate(0, 0, -(days - 1)).Unix()
}

// Session is one row of the session table with the project name joined in.
type Session struct {
	ID           string  `json:"id"`
	Title        string  `json:"title"`
	DisplayTitle string  `json:"displayTitle"`
	Project      string  `json:"project"`
	Directory    string  `json:"directory"`
	Agent        string  `json:"agent"`
	Model        string  `json:"model"` // "provider/model"
	Cost         float64 `json:"cost"`
	Input        int64   `json:"input"`
	Output       int64   `json:"output"`
	Reasoning    int64   `json:"reasoning"`
	CacheRead    int64   `json:"cacheRead"`
	CacheWrite   int64   `json:"cacheWrite"`
	CreatedMs    int64   `json:"createdMs"`
	UpdatedMs    int64   `json:"updatedMs"`
}

// modelRef mirrors the JSON stored in session.model.
type modelRef struct {
	ProviderID string `json:"providerID"`
	ID         string `json:"id"`
}

// NormaliseModel turns the session.model column (JSON like
// {"providerID":"opencode","id":"muse-spark-..."} or a plain string)
// into "provider/model".
func NormaliseModel(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if strings.HasPrefix(raw, "{") {
		var m modelRef
		if err := json.Unmarshal([]byte(raw), &m); err == nil && m.ID != "" {
			if m.ProviderID != "" {
				return m.ProviderID + "/" + m.ID
			}
			return m.ID
		}
	}
	return raw
}

const sessionColumns = `s.id, s.title, COALESCE(p.name, ''),
  s.directory, COALESCE(s.agent, ''), COALESCE(s.model, ''),
  s.cost, s.tokens_input, s.tokens_output, s.tokens_reasoning,
  s.tokens_cache_read, s.tokens_cache_write,
  s.time_created, s.time_updated`

func scanSession(row interface{ Scan(...any) error }) (Session, error) {
	var s Session
	var rawModel string
	err := row.Scan(&s.ID, &s.Title, &s.Project, &s.Directory, &s.Agent,
		&rawModel, &s.Cost, &s.Input, &s.Output, &s.Reasoning,
		&s.CacheRead, &s.CacheWrite, &s.CreatedMs, &s.UpdatedMs)
	if err != nil {
		return Session{}, err
	}
	s.Model = NormaliseModel(rawModel)
	s.DisplayTitle = DisplayTitle(s.Title)
	return s, nil
}

// DisplayTitle renders opencode's auto-generated titles readably.
// opencode names untitled sessions "New session - 2026-09-08T13:39:38.688Z";
// that becomes "New session · Sep 8, 1:39 PM" (local time).
func DisplayTitle(title string) string {
	const prefix = "New session - "
	if rest, ok := strings.CutPrefix(title, prefix); ok {
		if ts, err := time.Parse(time.RFC3339, rest); err == nil {
			return "New session · " + ts.Local().Format("Jan 2, 3:04 PM")
		}
	}
	if title == "" {
		return "Untitled session"
	}
	return title
}

// DaySessions lists sessions updated on a local calendar day (YYYY-MM-DD).
func DaySessions(db *sql.DB, day string) ([]Session, error) {
	rows, err := db.Query(`SELECT `+sessionColumns+`
		FROM session s LEFT JOIN project p ON p.id = s.project_id
		WHERE date(s.time_updated/1000, 'unixepoch', 'localtime') = ?
		ORDER BY s.time_updated DESC`, day)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Session
	for rows.Next() {
		s, err := scanSession(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// RecentSessions lists the most recently updated sessions across all time.
func RecentSessions(db *sql.DB, limit int) ([]Session, error) {
	rows, err := db.Query(`SELECT `+sessionColumns+`
		FROM session s LEFT JOIN project p ON p.id = s.project_id
		ORDER BY s.time_updated DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Session
	for rows.Next() {
		s, err := scanSession(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// DayStat is one per-day rollup row.
type DayStat struct {
	Day       string  `json:"day"`
	Sessions  int64   `json:"sessions"`
	Messages  int64   `json:"messages"`
	Cost      float64 `json:"cost"`
	Input     int64   `json:"input"`
	Output    int64   `json:"output"`
	Reasoning int64   `json:"reasoning"`
	CacheRead int64   `json:"cacheRead"`
}

// History returns per-day session-grain rollups for the last N days.
// Message counts are merged in from a second query because SQLite cannot
// reference a SELECT alias inside a correlated subquery.
func History(db *sql.DB, days int) ([]DayStat, error) {
	cutoff := dayStartUnix(days)
	rows, err := db.Query(`
		SELECT date(s.time_updated/1000, 'unixepoch', 'localtime') AS d,
		  COUNT(*),
		  COALESCE(SUM(s.cost),0),
		  COALESCE(SUM(s.tokens_input),0), COALESCE(SUM(s.tokens_output),0),
		  COALESCE(SUM(s.tokens_reasoning),0), COALESCE(SUM(s.tokens_cache_read),0)
		FROM session s
		WHERE s.time_updated/1000 >= ?
		GROUP BY d ORDER BY d DESC`, cutoff)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	byDay := map[string]int{}
	var out []DayStat
	for rows.Next() {
		var st DayStat
		if err := rows.Scan(&st.Day, &st.Sessions, &st.Cost,
			&st.Input, &st.Output, &st.Reasoning, &st.CacheRead); err != nil {
			return nil, err
		}
		byDay[st.Day] = len(out)
		out = append(out, st)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	mrows, err := db.Query(`
		SELECT date(m.time_created/1000, 'unixepoch', 'localtime') AS d, COUNT(*)
		FROM message m
		WHERE m.time_created/1000 >= ?
		GROUP BY d`, cutoff)
	if err != nil {
		return nil, err
	}
	defer mrows.Close()
	for mrows.Next() {
		var day string
		var n int64
		if err := mrows.Scan(&day, &n); err != nil {
			return nil, err
		}
		if st, ok := byDay[day]; ok {
			out[st].Messages = n
		}
	}
	return out, mrows.Err()
}

// ModelDayStat is one per-model-per-day rollup at message grain.
type ModelDayStat struct {
	Day       string  `json:"day"`
	Model     string  `json:"model"`
	Messages  int64   `json:"messages"`
	Cost      float64 `json:"cost"`
	Input     int64   `json:"input"`
	Output    int64   `json:"output"`
	Reasoning int64   `json:"reasoning"`
	CacheRead int64   `json:"cacheRead"`
}

// ModelsByDate returns per-model-per-day rollups from assistant messages
// (message grain carries the true per-model attribution).
func ModelsByDate(db *sql.DB, days int) ([]ModelDayStat, error) {
	rows, err := db.Query(`
		SELECT date(json_extract(m.data,'$.time.created')/1000,'unixepoch','localtime') AS d,
		  json_extract(m.data,'$.providerID') || '/' || json_extract(m.data,'$.modelID') AS model,
		  COUNT(*),
		  COALESCE(SUM(json_extract(m.data,'$.cost')),0),
		  COALESCE(SUM(json_extract(m.data,'$.tokens.input')),0),
		  COALESCE(SUM(json_extract(m.data,'$.tokens.output')),0),
		  COALESCE(SUM(json_extract(m.data,'$.tokens.reasoning')),0),
		  COALESCE(SUM(json_extract(m.data,'$.tokens.cache.read')),0)
		FROM message m
		WHERE json_extract(m.data,'$.role') = 'assistant'
		  AND json_extract(m.data,'$.time.created')/1000 >= ?
		GROUP BY d, model ORDER BY d DESC, model ASC`, dayStartUnix(days))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ModelDayStat
	for rows.Next() {
		var st ModelDayStat
		if err := rows.Scan(&st.Day, &st.Model, &st.Messages, &st.Cost,
			&st.Input, &st.Output, &st.Reasoning, &st.CacheRead); err != nil {
			return nil, err
		}
		out = append(out, st)
	}
	return out, rows.Err()
}
