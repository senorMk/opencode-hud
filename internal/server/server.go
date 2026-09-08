// Package server is the localhost HTTP backend for the HUD window.
//
// The Wails webview loads these pages over 127.0.0.1, which keeps the
// frontend free of JS-bridge bindings and makes every endpoint testable
// with curl. --serve runs this server standalone for browser previews.
package server

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/senorMk/opencode-hud/internal/store"
	"github.com/senorMk/opencode-hud/internal/watcher"
)

//go:embed frontend
var frontend embed.FS

// Server serves the HUD UI and JSON API on 127.0.0.1.
type Server struct {
	dbPath  string
	w       *watcher.Watcher
	latest  atomic.Pointer[watcher.Snapshot]
	http    *http.Server
	ln      net.Listener
	assets  http.Handler
	baseURL string
}

// New creates a Server. Call Run to bind and block.
func New(dbPath string, interval time.Duration) *Server {
	sub, err := fs.Sub(frontend, "frontend")
	if err != nil {
		panic(fmt.Sprintf("frontend assets: %v", err))
	}
	s := &Server{
		dbPath: dbPath,
		w:      watcher.New(dbPath, interval, 10),
		assets: http.FileServer(http.FS(sub)),
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/snapshot", s.handleSnapshot)
	mux.HandleFunc("/api/history", s.handleHistory)
	mux.HandleFunc("/api/models", s.handleModels)
	mux.HandleFunc("/api/health", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, map[string]bool{"ok": true})
	})
	mux.Handle("/", s.assets)
	s.http = &http.Server{Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	return s
}

// Bind binds 127.0.0.1:0 and starts the watcher. Serve blocks serving.
func (s *Server) Bind() (string, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", err
	}
	s.ln = ln
	s.baseURL = "http://" + ln.Addr().String()
	s.w.Start()
	go func() {
		for snap := range s.w.C() {
			snap := snap
			s.latest.Store(&snap)
		}
	}()
	return s.baseURL, nil
}

// Serve blocks serving the bound listener. Call Bind first.
func (s *Server) Serve() error {
	if s.ln == nil {
		return fmt.Errorf("server: Serve called before Bind")
	}
	return s.http.Serve(s.ln)
}

// Run binds and serves (convenience for --serve).
func (s *Server) Run() error {
	if _, err := s.Bind(); err != nil {
		return err
	}
	return s.Serve()
}

// URL returns the base URL (valid after Run has bound the listener).
func (s *Server) URL() string { return s.baseURL }

// Close stops the HTTP server and the watcher.
func (s *Server) Close() {
	_ = s.http.Close()
	s.w.Stop()
}

func (s *Server) current() (watcher.Snapshot, error) {
	if snap := s.latest.Load(); snap != nil {
		return *snap, nil
	}
	snap, err := watcher.LoadSnapshot(s.dbPath, 10)
	if err != nil {
		return watcher.Snapshot{}, err
	}
	s.latest.Store(&snap)
	return snap, nil
}

func (s *Server) handleSnapshot(w http.ResponseWriter, _ *http.Request) {
	snap, err := s.current()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, snap)
}

func (s *Server) handleHistory(w http.ResponseWriter, r *http.Request) {
	days := queryDays(r, 30)
	db, err := store.Open(s.dbPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer db.Close()
	rows, err := store.History(db, days)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if rows == nil {
		rows = []store.DayStat{}
	}
	writeJSON(w, rows)
}

func (s *Server) handleModels(w http.ResponseWriter, r *http.Request) {
	days := queryDays(r, 30)
	db, err := store.Open(s.dbPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer db.Close()
	rows, err := store.ModelsByDate(db, days)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if rows == nil {
		rows = []store.ModelDayStat{}
	}
	writeJSON(w, rows)
}

func queryDays(r *http.Request, def int) int {
	n, err := strconv.Atoi(r.URL.Query().Get("days"))
	if err != nil || n < 1 || n > 365 {
		return def
	}
	return n
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}
