// Package tray hosts the macOS menu-bar extra: a Wails v3 SystemTray
// whose label shows live usage, fed by the watcher.
package tray

import (
	"time"

	"github.com/senorMk/opencode-hud/internal/watcher"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// Run starts the menu-bar daemon and blocks until Quit.
// dbPath is the opencode database; interval is the poll period.
func Run(dbPath string, interval time.Duration) error {
	app := application.New(application.Options{
		Name: "opencode-hud",
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: false,
		},
	})

	systray := app.SystemTray.New()
	systray.SetLabel("opencode…")

	w := watcher.New(dbPath, interval, 10)

	menu := app.NewMenu()
	statusItem := menu.Add("Starting…")
	statusItem.SetEnabled(false)
	menu.AddSeparator()
	openItem := menu.Add("Open HUD")
	openItem.SetEnabled(false) // enabled in M3 when the popover window lands
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
