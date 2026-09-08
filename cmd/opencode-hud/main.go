// Command opencode-hud is the native macOS usage HUD for OpenCode.
//
// M1: CLI dump commands over the read-only data layer, used to prove
// parity with `opencode stats` before any UI is built.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/parentalk/opencode-hud/internal/service"
	"github.com/parentalk/opencode-hud/internal/store"
)

func main() {
	dbPath := flag.String("db", "", "opencode database path (default: ~/.local/share/opencode/opencode.db)")
	dumpToday := flag.Bool("dump-today", false, "dump sessions updated today with totals")
	history := flag.Bool("history", false, "dump per-day history rollup")
	modelsByDate := flag.Bool("models-by-date", false, "dump per-model-per-day rollup")
	days := flag.Int("days", 7, "days of history for --history / --models-by-date")
	recent := flag.Int("recent", 0, "list N most recently updated sessions")
	flag.Parse()

	path := *dbPath
	if path == "" {
		var err error
		path, err = store.DefaultPath()
		if err != nil {
			fatal(err)
		}
	}
	db, err := store.Open(path)
	if err != nil {
		fatal(err)
	}
	defer db.Close()

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")

	switch {
	case *dumpToday:
		rep, err := service.Today(db, time.Now())
		if err != nil {
			fatal(err)
		}
		if err := enc.Encode(rep); err != nil {
			fatal(err)
		}
	case *history:
		rows, err := store.History(db, *days)
		if err != nil {
			fatal(err)
		}
		if rows == nil {
			rows = []store.DayStat{}
		}
		if err := enc.Encode(rows); err != nil {
			fatal(err)
		}
	case *modelsByDate:
		rows, err := store.ModelsByDate(db, *days)
		if err != nil {
			fatal(err)
		}
		if rows == nil {
			rows = []store.ModelDayStat{}
		}
		if err := enc.Encode(rows); err != nil {
			fatal(err)
		}
	case *recent > 0:
		rows, err := store.RecentSessions(db, *recent)
		if err != nil {
			fatal(err)
		}
		if rows == nil {
			rows = []store.Session{}
		}
		if err := enc.Encode(rows); err != nil {
			fatal(err)
		}
	default:
		fmt.Fprintln(os.Stderr, "usage: opencode-hud --dump-today | --history [--days N] | --models-by-date [--days N] | --recent N")
		os.Exit(2)
	}
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "opencode-hud:", err)
	os.Exit(1)
}
