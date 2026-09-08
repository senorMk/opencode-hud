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
	"os/signal"
	"syscall"
	"time"

	"github.com/senorMk/opencode-hud/internal/server"
	"github.com/senorMk/opencode-hud/internal/service"
	"github.com/senorMk/opencode-hud/internal/store"
	"github.com/senorMk/opencode-hud/internal/tray"
	"github.com/senorMk/opencode-hud/internal/watcher"
)

func main() {
	dbPath := flag.String("db", "", "opencode database path (default: ~/.local/share/opencode/opencode.db)")
	dumpToday := flag.Bool("dump-today", false, "dump sessions updated today with totals")
	history := flag.Bool("history", false, "dump per-day history rollup")
	modelsByDate := flag.Bool("models-by-date", false, "dump per-model-per-day rollup")
	days := flag.Int("days", 7, "days of history for --history / --models-by-date")
	recent := flag.Int("recent", 0, "list N most recently updated sessions")
	trayMode := flag.Bool("tray", false, "run the macOS menu-bar daemon")
	trayPrint := flag.Bool("tray-print", false, "headless watcher: print menu-bar label on each snapshot until interrupted")
	serve := flag.Bool("serve", false, "run the localhost HUD server (browser preview) and print its URL")
	interval := flag.Duration("interval", 2*time.Second, "poll interval for --tray / --tray-print / --serve")
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
	case *trayMode:
		if err := tray.Run(path, *interval); err != nil {
			fatal(err)
		}
	case *serve:
		srv := server.New(path, *interval)
		url, err := srv.Bind()
		if err != nil {
			fatal(err)
		}
		defer srv.Close()
		fmt.Println(url)
		go func() { _ = srv.Serve() }()
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
		<-sig
	case *trayPrint:
		w := watcher.New(path, *interval, 10)
		w.Start()
		defer w.Stop()
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
		for {
			select {
			case snap := <-w.C():
				fmt.Printf("%s  %s\n", snap.At.Format("15:04:05"), snap.Status.Label)
			case <-sig:
				return
			}
		}
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
		fmt.Fprintln(os.Stderr, "usage: opencode-hud --dump-today | --history [--days N] | --models-by-date [--days N] | --recent N | --tray | --tray-print")
		os.Exit(2)
	}
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "opencode-hud:", err)
	os.Exit(1)
}
