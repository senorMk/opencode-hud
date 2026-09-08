// Package watcher polls the opencode database on a ticker and emits a
// Snapshot whenever the database fingerprint moves. It is UI-independent:
// the Wails tray (or --tray-print) consumes the channel.
package watcher

import (
	"database/sql"
	"time"

	"github.com/senorMk/opencode-hud/internal/service"
	"github.com/senorMk/opencode-hud/internal/store"
)

// Snapshot is one full refresh of HUD state.
type Snapshot struct {
	At     time.Time          `json:"at"`
	Today  service.DayReport  `json:"today"`
	Recent []store.Session    `json:"recent"`
	Status service.MenuStatus `json:"status"`
}

// Watcher polls dbPath every interval and publishes snapshots.
type Watcher struct {
	dbPath   string
	interval time.Duration
	recentN  int

	ch      chan Snapshot
	refresh chan struct{}
	stop    chan struct{}
	done    chan struct{}
}

// New creates a Watcher. interval is the poll period (2s is a good
// default); recentN bounds the recent-sessions list used for "active"
// detection.
func New(dbPath string, interval time.Duration, recentN int) *Watcher {
	if recentN <= 0 {
		recentN = 10
	}
	return &Watcher{
		dbPath:   dbPath,
		interval: interval,
		recentN:  recentN,
		ch:       make(chan Snapshot, 1),
		refresh:  make(chan struct{}, 1),
		stop:     make(chan struct{}),
		done:     make(chan struct{}),
	}
}

// C returns the snapshot channel (buffered, latest-wins: slow consumers
// may skip intermediate snapshots but never block the poll loop).
func (w *Watcher) C() <-chan Snapshot { return w.ch }

// Start begins polling in a new goroutine, emitting an initial snapshot.
func (w *Watcher) Start() {
	go w.loop()
}

// Stop blocks until the poll loop exits.
func (w *Watcher) Stop() {
	close(w.stop)
	<-w.done
}

// Refresh requests an immediate poll (non-blocking).
func (w *Watcher) Refresh() {
	select {
	case w.refresh <- struct{}{}:
	default:
	}
}

func (w *Watcher) loop() {
	defer close(w.done)
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()
	var last store.Fingerprint
	first := true
	for {
		if w.poll(&last, first) {
			first = false
		}
		select {
		case <-w.stop:
			return
		case <-w.refresh:
		case <-ticker.C:
		}
	}
}

// poll loads and publishes a snapshot when the fingerprint moved
// (or always, on the first pass). It reports whether a snapshot
// was published.
func (w *Watcher) poll(last *store.Fingerprint, first bool) bool {
	db, err := store.Open(w.dbPath)
	if err != nil {
		return false
	}
	defer db.Close()

	fp, err := store.FingerprintOf(db)
	if err != nil {
		return false
	}
	if !first && fp == *last {
		return false
	}
	snap, err := w.snapshot(db)
	if err != nil {
		return false
	}
	*last = fp
	select {
	case w.ch <- snap:
	default:
		// Slow consumer: replace the stale buffered snapshot so the
		// next receive always sees the latest state.
		select {
		case <-w.ch:
		default:
		}
		select {
		case w.ch <- snap:
		default:
		}
	}
	return true
}

func (w *Watcher) snapshot(db *sql.DB) (Snapshot, error) {
	return loadSnapshot(db, w.recentN, time.Now())
}

// LoadSnapshot loads a one-off snapshot outside the poll loop
// (used to serve the first HTTP request before the watcher ticks).
func LoadSnapshot(dbPath string, recentN int) (Snapshot, error) {
	db, err := store.Open(dbPath)
	if err != nil {
		return Snapshot{}, err
	}
	defer db.Close()
	return loadSnapshot(db, recentN, time.Now())
}

func loadSnapshot(db *sql.DB, recentN int, now time.Time) (Snapshot, error) {
	recent, err := store.RecentSessions(db, recentN)
	if err != nil {
		return Snapshot{}, err
	}
	if recent == nil {
		recent = []store.Session{}
	}
	rep, status, err := service.LoadToday(db, now, recent)
	if err != nil {
		return Snapshot{}, err
	}
	if rep.Sessions == nil {
		rep.Sessions = []store.Session{}
	}
	return Snapshot{At: now, Today: rep, Recent: recent, Status: status}, nil
}
