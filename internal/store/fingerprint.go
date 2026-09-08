package store

import "database/sql"

// Fingerprint is a cheap change-detection signature over the database.
// The watcher polls this on a ticker and only loads a full snapshot
// when the fingerprint moves.
type Fingerprint struct {
	Sessions     int64
	SessionStamp int64 // MAX(session.time_updated), ms
	Messages     int64
}

// FingerprintOf reads the current fingerprint. It opens no transaction
// and takes no locks beyond the statement itself.
func FingerprintOf(db *sql.DB) (Fingerprint, error) {
	var fp Fingerprint
	err := db.QueryRow(`
		SELECT (SELECT COUNT(*) FROM session),
		       COALESCE((SELECT MAX(time_updated) FROM session), 0),
		       (SELECT COUNT(*) FROM message)`).Scan(
		&fp.Sessions, &fp.SessionStamp, &fp.Messages)
	return fp, err
}

// CountMessagesSince counts messages created at or after sinceMs (ms epoch).
func CountMessagesSince(db *sql.DB, sinceMs int64) (int64, error) {
	var n int64
	err := db.QueryRow(`SELECT COUNT(*) FROM message WHERE time_created >= ?`, sinceMs).Scan(&n)
	return n, err
}
