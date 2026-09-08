// Package tray hosts the macOS menu-bar extra: a Wails v3 SystemTray
// whose label shows live usage, fed by the watcher, plus the HUD window.
package tray

import (
	"time"

	"github.com/senorMk/opencode-hud/internal/server"
	"github.com/senorMk/opencode-hud/internal/watcher"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// Run starts the menu-bar daemon and blocks until Quit.
// dbPath is the opencode database; interval is the poll period.
func Run(dbPath string, interval time.Duration) error {
	srv := server.New(dbPath, interval)
	url, err := srv.Bind()
	if err != nil {
		return err
	}
	defer srv.Close()
	go func() { _ = srv.Serve() }()

	app := application.New(application.Options{
		Name: "opencode-hud",
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: false,
		},
	})

	window := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:  "opencode-hud",
		URL:    url + "/",
		Width:  400,
		Height: 600,
		Hidden: true,
	})

	systray := app.SystemTray.New()
	systray.SetLabel("opencode…")
	systray.AttachWindow(window)
	systray.WindowOffset(10)

	w := watcher.New(dbPath, interval, 10)

	menu := app.NewMenu()
	statusItem := menu.Add("Starting…")
	statusItem.SetEnabled(false)
	menu.AddSeparator()
	menu.Add("Open HUD").OnClick(func(_ *application.Context) {
		window.Show()
		window.Focus()
	})
	menu.AddSeparator()
	menu.Add("Refresh now").OnClick(func(_ *application.Context) {
		w.Refresh()
	})
	menu.AddSeparator()
	menu.Add("Quit").OnClick(func(_ *application.Context) {
		app.Quit()
	})
	systray.SetMenu(menu)

	w.Start()
	defer w.Stop()

	go func() {
		for snap := range w.C() {
			systray.SetLabel(snap.Status.Label)
			systray.SetTooltip(snap.Status.Tooltip)
		}
	}()

	app.Run()
	return nil
}
