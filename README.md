# opencode-hud

A native macOS usage HUD for [OpenCode](https://opencode.ai) — live session usage and historical analytics in your menu bar, no matter where you run opencode.

## Motivation

OpenCode runs everywhere: multiple terminals, the web UI (`opencode web`), IDE extensions, background agents. But existing usage HUDs are **terminal-bound plugins**:

- [Two-Weeks-Team/opencode-status-hud](https://github.com/Two-Weeks-Team/opencode-status-hud) — renders a HUD line inside one TUI session.
- [Alaye-Dong/opencode-hud](https://github.com/Alaye-Dong/opencode-hud) — end-of-conversation toasts in one TUI session.
- [ndom91/opencode-hud](https://github.com/ndom91/opencode-hud) — a persistent TUI widget, still inside one terminal.
- [Ainsley0917/opencode-token-monitor](https://github.com/Ainsley0917/opencode-token-monitor) — terminal analytics and budgets.

Each of these only sees the session in front of you. If you run three terminals plus the web UI, you have no single place that answers: **"What have I spent today, on which models, across all sessions?"**

`opencode-hud` takes a different approach: instead of living inside a terminal, it reads the one thing all opencode clients already share — the local SQLite database at `~/.local/share/opencode/opencode.db` — and surfaces it as a **native macOS menu bar extra with a popover dashboard**:

- **Always visible** — glanceable `$ today` / active-session indicator in the menu bar.
- **Cross-surface** — TUI, web, IDE, background runs all write to the same DB, so all are counted.
- **Historical** — per-day totals, per-model breakdowns, per-model-by-date matrix, CSV export.
- **Local-first, read-only** — the app opens the database `mode=ro, query_only`, never writes to it. Costs come straight from the stored `cost` fields (same numbers as `opencode stats`).

## Stack

- **Language:** Go (pure-Go SQLite via `modernc.org/sqlite` — no CGO).
- **UI:** [Wails v3](https://v3.wails.io) — Go backend + webview frontend; menu bar extra (`Systray`) with attached popover window, plus a full window for history views.
- **Data:** read-only polling of `opencode.db` (WAL-aware, 1–2s ticker + `fsnotify` wake on `opencode.db-wal`).

## Status

M3 — menu-bar daemon + HUD window (this repo). Browser preview:

```sh
go run ./cmd/opencode-hud --serve   # prints http://127.0.0.1:PORT — open it
go run ./cmd/opencode-hud --tray    # macOS menu-bar extra + attached window
```

## Quick start (M1)

```sh
go run ./cmd/opencode-hud --dump-today
go run ./cmd/opencode-hud --history --days 30
go run ./cmd/opencode-hud --models-by-date --days 7
```

Compare against the reference implementation:

```sh
opencode stats --days 7 --models
```

## Project layout

```
cmd/opencode-hud/   # Wails entry (later); M1: CLI dump commands
internal/store/     # read-only SQLite open + row types
internal/service/   # aggregation: live, history, per-model-by-date
frontend/           # TS + charts popover UI (M3+)
build/              # .app bundle, Info.plist, icons (M5)
scripts/            # dev, package, launchd install (M5)
```

## License

MIT.
