package app

import (
	"os"
	"path/filepath"
	"sync"

	"github.com/GuilhermeVozniak/tiles-spliter/desktop/internal/engine"
	"github.com/GuilhermeVozniak/tiles-spliter/desktop/internal/platform"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// SettingsStore holds the live settings with thread-safe access.
type SettingsStore struct {
	mu   sync.RWMutex
	s    engine.Settings
	path string
}

func NewSettingsStore() *SettingsStore {
	home, _ := os.UserHomeDir()
	path := filepath.Join(home, "Library", "Application Support", "tiles-spliter", "settings.json")
	return &SettingsStore{s: engine.LoadSettings(path), path: path}
}

func (st *SettingsStore) Get() engine.Settings {
	st.mu.RLock()
	defer st.mu.RUnlock()
	return st.s
}

func (st *SettingsStore) Set(s engine.Settings) error {
	st.mu.Lock()
	st.s = s
	st.mu.Unlock()
	return s.Save(st.path)
}

// SettingsService is bound to the frontend via Wails.
type SettingsService struct {
	store      *SettingsStore
	dispatcher *Dispatcher
	app        *application.App
	prefs      *application.WebviewWindow
	tray       *application.SystemTray
	engineOn   bool
}

func NewSettingsService(store *SettingsStore, d *Dispatcher) *SettingsService {
	return &SettingsService{store: store, dispatcher: d}
}

func (s *SettingsService) SetApp(a *application.App)                   { s.app = a }
func (s *SettingsService) SetPrefsWindow(w *application.WebviewWindow) { s.prefs = w }
func (s *SettingsService) SetTray(t *application.SystemTray)           { s.tray = t }

func (s *SettingsService) ShowPreferences() {
	platform.ActivatePrefs()
	s.prefs.Show()
	s.prefs.Focus()
}

// StartEngine registers hotkeys and the drag tap. Idempotent.
func (s *SettingsService) StartEngine() {
	if s.engineOn {
		return
	}
	s.engineOn = true
	s.applyHotkeys(s.store.Get())
	_ = platform.StartDragTap(func(kind platform.DragEventKind, x, y float64, mod bool) {
		s.dispatcher.OnDragEvent(int(kind), x, y, mod)
	})
}

// Hotkey id == index into menuOrder(); undo uses undoHotkeyID.
func menuOrder() []engine.Action {
	return []engine.Action{
		engine.ActionCenter, engine.ActionFullscreen,
		engine.ActionHalfLeft, engine.ActionHalfRight, engine.ActionHalfTop, engine.ActionHalfBottom,
		engine.ActionUpperLeft, engine.ActionUpperRight, engine.ActionLowerLeft, engine.ActionLowerRight,
		engine.ActionNextThird, engine.ActionPrevThird,
		engine.ActionTwoThirdsLeft, engine.ActionTwoThirdsRight, engine.ActionTwoThirdsCenter,
		engine.ActionNextDisplay, engine.ActionPrevDisplay,
	}
}

const undoHotkeyID = 1000

func (s *SettingsService) applyHotkeys(cfg engine.Settings) {
	platform.UnregisterAllHotkeys()
	if !cfg.Hotkeys.Enabled {
		return
	}
	platform.InstallHotkeyHandler(func(id uint32) {
		if id == undoHotkeyID {
			s.dispatcher.Undo()
			return
		}
		actions := menuOrder()
		if int(id) < len(actions) {
			s.dispatcher.Perform(actions[id])
		}
	})
	for i, a := range menuOrder() {
		if hk, ok := cfg.Hotkeys.Bindings[a]; ok {
			_ = platform.RegisterHotkey(uint32(i), hk)
		}
	}
	// Undo binding (⌥⌘Y by default) is stored under a pseudo-action key "undo".
	if hk, ok := cfg.Hotkeys.Bindings[engine.Action("undo")]; ok {
		_ = platform.RegisterHotkey(undoHotkeyID, hk)
	}
}

// --- Frontend-bound API ---

func (s *SettingsService) Get() engine.Settings { return s.store.Get() }

func (s *SettingsService) Update(next engine.Settings) error {
	old := s.store.Get()
	if err := s.store.Set(next); err != nil {
		return err
	}
	s.applySideEffects(old, next)
	if s.app != nil {
		// API drift: event emission lives on the Event manager, not App.EmitEvent.
		s.app.Event.Emit("settings:changed", next)
	}
	return nil
}

func (s *SettingsService) RestoreDefaultHotkeys() engine.Settings {
	cur := s.store.Get()
	cur.Hotkeys = engine.DefaultSettings().Hotkeys
	_ = s.Update(cur)
	return cur
}

func (s *SettingsService) AXTrusted() bool { return platform.AXTrusted(false) }
func (s *SettingsService) RequestAXPermission() bool {
	ok := platform.AXTrusted(true)
	if ok {
		s.StartEngine()
	}
	return ok
}

func (s *SettingsService) applySideEffects(old, next engine.Settings) {
	if s.engineOn {
		s.applyHotkeys(next)
	}
	if old.General.LaunchAtLogin != next.General.LaunchAtLogin {
		_ = platform.SetLoginItem(next.General.LaunchAtLogin)
	}
	if s.tray != nil && old.General.ShowMenuBarIcon != next.General.ShowMenuBarIcon {
		if next.General.ShowMenuBarIcon {
			s.tray.Show()
		} else {
			s.tray.Hide()
		}
	}
}
