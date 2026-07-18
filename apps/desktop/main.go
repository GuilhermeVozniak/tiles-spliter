package main

import (
	"embed"
	"log/slog"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"

	"github.com/GuilhermeVozniak/tiles-spliter/desktop/internal/app"
	"github.com/GuilhermeVozniak/tiles-spliter/desktop/internal/engine"
	"github.com/GuilhermeVozniak/tiles-spliter/desktop/internal/platform"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed build/tray-icon.png
var trayIconBytes []byte

var actionLabels = map[engine.Action]string{
	engine.ActionCenter: "Center", engine.ActionFullscreen: "Fullscreen",
	engine.ActionHalfLeft: "Half Left", engine.ActionHalfRight: "Half Right",
	engine.ActionHalfTop: "Half Top", engine.ActionHalfBottom: "Half Bottom",
	engine.ActionUpperLeft: "Upper Left", engine.ActionUpperRight: "Upper Right",
	engine.ActionLowerLeft: "Lower Left", engine.ActionLowerRight: "Lower Right",
	engine.ActionNextThird: "Next Third", engine.ActionPrevThird: "Previous Third",
	engine.ActionTwoThirdsLeft: "Two Thirds Left", engine.ActionTwoThirdsRight: "Two Thirds Right",
	engine.ActionTwoThirdsCenter: "Two Thirds Center",
	engine.ActionNextDisplay:     "Next Display", engine.ActionPrevDisplay: "Previous Display",
}

func main() {
	store := app.NewSettingsStore() // loads from Application Support
	dispatcher := app.NewDispatcher(app.RealPlatform{}, store.Get)
	svc := app.NewSettingsService(store, dispatcher)

	wailsApp := application.New(application.Options{
		Name: "Tiles Spliter",
		Services: []application.Service{
			application.NewService(svc),
		},
		Assets: application.AssetOptions{Handler: application.AssetFileServerFS(assets)},
		Mac: application.MacOptions{
			ActivationPolicy: application.ActivationPolicyAccessory,
		},
		SingleInstance: &application.SingleInstanceOptions{
			UniqueID: "com.guilhermevozniak.tilespliter",
			OnSecondInstanceLaunch: func(data application.SecondInstanceData) {
				svc.ShowPreferences()
			},
		},
	})
	svc.SetApp(wailsApp)

	// --- Prefs window (hidden until requested) ---
	// API drift: Wails v3 alpha exposes window creation via the Window manager,
	// not a top-level App.NewWebviewWindowWithOptions method. It also has no
	// hide-on-close option in WebviewWindowOptions and no IsDestroyed probe on
	// WebviewWindow, so: closing the prefs window must not destroy it (the app
	// keeps running as a menu-bar accessory) — intercept the close, hide the
	// window instead, and drop the Dock icon that ActivatePrefs added. The
	// factory lets ShowPreferences recreate the window defensively if it was
	// ever destroyed anyway.
	makePrefs := func() *application.WebviewWindow {
		prefs := wailsApp.Window.NewWithOptions(application.WebviewWindowOptions{
			Title:  "Tiles Spliter",
			Width:  640,
			Height: 520,
			Hidden: true,
		})
		prefs.OnWindowEvent(events.Common.WindowClosing, func(e *application.WindowEvent) {
			e.Cancel()
			prefs.Hide()
			platform.HideFromDock()
		})
		return prefs
	}
	svc.SetPrefsFactory(makePrefs)
	svc.SetPrefsWindow(makePrefs())

	// --- Tray ---
	// API drift: system tray creation lives on the SystemTray manager.
	tray := wailsApp.SystemTray.New()
	menu := wailsApp.NewMenu()
	menu.Add("Preferences…").SetAccelerator("CmdOrCtrl+,").OnClick(func(*application.Context) { svc.ShowPreferences() })
	menu.AddSeparator()
	for _, a := range app.MenuOrder() {
		action := a
		menu.Add(actionLabels[action]).OnClick(func(*application.Context) { dispatcher.Perform(action) })
	}
	menu.AddSeparator()
	menu.Add("Undo").OnClick(func(*application.Context) { dispatcher.Undo() })
	menu.AddSeparator()
	menu.Add("About Tiles Spliter").OnClick(func(*application.Context) { svc.ShowPreferences() })
	menu.Add("Quit Tiles Spliter").SetAccelerator("CmdOrCtrl+Q").OnClick(func(*application.Context) { wailsApp.Quit() })
	tray.SetMenu(menu)
	tray.SetTemplateIcon(trayIconBytes)
	svc.SetTray(tray)

	// --- Engine wiring (only once AX permission exists) ---
	// API drift: application-lifecycle events are subscribed via the Event
	// manager, and the event name is a typed constant from pkg/events rather
	// than a bare string.
	wailsApp.Event.OnApplicationEvent(events.Mac.ApplicationDidFinishLaunching, func(*application.ApplicationEvent) {
		// Reconcile out-of-process state (login item, tray visibility) with the
		// loaded settings regardless of AX permission.
		svc.ReconcileStartup()
		if platform.AXTrusted(false) {
			svc.StartEngine()
		} else {
			svc.ShowPreferences() // onboarding view will ask for permission
		}
	})

	if err := wailsApp.Run(); err != nil {
		slog.Error("app exited", "err", err)
	}
}
