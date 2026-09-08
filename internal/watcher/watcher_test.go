package watcher

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

const testSchema = `
CREATE TABLE project (id TEXT PRIMARY KEY, worktree TEXT NOT NULL, name TEXT, time_created INTEGER NOT NULL, time_updated INTEGER NOT NULL, sandboxes TEXT NOT NULL);
CREATE TABLE session (id TEXT PRIMARY KEY, project_id TEXT NOT NULL, directory TEXT NOT NULL, slug TEXT NOT NULL, title TEXT NOT NULL, version TEXT NOT NULL, cost REAL DEFAULT 0 NOT NULL, tokens_input INTEGER DEFAULT 0 NOT NULL, tokens_output INTEGER DEFAULT 0 NOT NULL, tokens_reasoning INTEGER DEFAULT 0 NOT NULL, tokens_cache_read INTEGER DEFAULT 0 NOT NULL, tokens_cache_write INTEGER DEFAULT 0 NOT NULL, agent TEXT, model TEXT, time_created INTEGER NOT NULL, time_updated INTEGER NOT NULL);
CREATE TABLE message (id TEXT PRIMARY KEY, session_id TEXT NOT NULL, time_created INTEGER NOT NULL, time_updated INTEGER NOT NULL, data TEXT NOT NULL);
`

func seedTestDB(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.db")
	db, err := sql.Open("sqlite", "file:"+path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(testSchema); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UnixMilli()
	if _, err := db.Exec(`INSERT INTO project (id, worktree, time_created, time_updated, sandboxes) VALUES ('prj_1', '/tmp/x', ?, ?, '[]')`, now, now); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO session (id, project_id, directory, slug, title, version, cost, tokens_input, agent, model, time_created, time_updated) VALUES ('ses_1', 'prj_1', '/tmp/x', 's', 'first', 'v', 1.5, 100, 'build', '{"providerID":"p","id":"m"}', ?, ?)`, now, now); err != nil {
		t.Fatal(err)
	}
	return path
}

func execOn(t *testing.T, path, q string, args ...any) {
	t.Helper()
	db, err := sql.Open("sqlite", "file:"+path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(q, args...); err != nil {
		t.Fatal(err)
	}
}

func TestWatcherEmitsInitialSnapshot(t *testing.T) {
	path := seedTestDB(t)
	w := New(path, 20*time.Millisecond, 10)
	w.Start()
	defer w.Stop()

	select {
	case snap := <-w.C():
		if snap.Today.Totals.Cost != 1.5 {
			t.Fatalf("cost = %.2f, want 1.50", snap.Today.Totals.Cost)
		}
		if snap.Status.Label != "$1.50 today · 1 active" {
			t.Fatalf("label = %q", snap.Status.Label)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("no initial snapshot")
	}
}

func TestWatcherEmitsOnChangeAndStaysQuietOtherwise(t *testing.T) {
	path := seedTestDB(t)
	w := New(path, 20*time.Millisecond, 10)
	w.Start()
	defer w.Stop()

	select {
	case <-w.C():
	case <-time.After(2 * time.Second):
		t.Fatal("no initial snapshot")
	}

	// Quiet period: no snapshots expected.
	select {
	case snap := <-w.C():
		t.Fatalf("unexpected snapshot while idle: %+v", snap.Status)
	case <-time.After(150 * time.Millisecond):
	}

	// Mutate the DB: fingerprint must move and a new snapshot arrive.
	now := time.Now().UnixMilli()
	execOn(t, path, `UPDATE session SET cost = 4.25, time_updated = ? WHERE id = 'ses_1'`, now)

	select {
	case snap := <-w.C():
		if snap.Today.Totals.Cost != 4.25 {
			t.Fatalf("cost = %.2f, want 4.25", snap.Today.Totals.Cost)
		}
		if snap.Status.Label != "$4.25 today · 1 active" {
			t.Fatalf("label = %q", snap.Status.Label)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("no snapshot after change")
	}
}

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}
