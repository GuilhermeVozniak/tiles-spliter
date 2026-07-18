package app

import (
	"log/slog"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"time"

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

// Swap atomically replaces the settings under a single write-lock: it reads
// the previous value, installs next, and saves — no concurrent Swap can
// interleave between the read and the write. Returns the previous value so
// callers can diff old against next.
func (st *SettingsStore) Swap(next engine.Settings) (old engine.Settings, err error) {
	st.mu.Lock()
	defer st.mu.Unlock()
	old = st.s
	st.s = next
	// Save under the lock so concurrent swaps can't interleave writes to the
	// shared settings.json.tmp file.
	return old, next.Save(st.path)
}

func (st *SettingsStore) Set(s engine.Settings) error {
	_, err := st.Swap(s)
	return err
}

// SettingsService is bound to the frontend via Wails.
type SettingsService struct {
	store          *SettingsStore
	dispatcher     *Dispatcher
	app            *application.App
	prefs          *application.WebviewWindow
	prefsFactory   func() *application.WebviewWindow // recreates prefs if destroyed
	tray           *application.SystemTray
	applyHotkeysFn func(engine.Settings) // seam: tests count hotkey re-registration

	engineMu sync.Mutex
	engineOn bool

	// hotkeysMu serializes the entire unregister→install→register sequence in
	// applyHotkeys so concurrent StartEngine/Update callers can't interleave
	// and leave a partial hotkey set registered.
	hotkeysMu sync.Mutex
	// hotkeysSuspended is set while the Hotkeys tab is recording a new combo:
	// with global hotkeys unregistered, Carbon no longer intercepts the
	// keydown, so an already-bound combo can be captured in the webview.
	// Guarded by hotkeysMu; checked inside applyHotkeys so every caller of
	// applyHotkeysFn (StartEngine, applySideEffects, ResumeHotkeys) honors it.
	hotkeysSuspended bool
	// suspendTimer is the watchdog started by SuspendHotkeys: if the webview
	// dies mid-recording (crash, force-quit) ResumeHotkeys is never called by
	// the frontend and hotkeys would stay suspended forever. Guarded by
	// hotkeysMu.
	suspendTimer *time.Timer
	// afterFunc is a seam over time.AfterFunc so tests can control the
	// watchdog without sleeping 60s; defaults to time.AfterFunc.
	afterFunc func(d time.Duration, f func()) *time.Timer
	// updateMu serializes Swap+applySideEffects in Update (and the
	// Get→mutate→Update sequence in RestoreDefaultHotkeys) so concurrent
	// updates always see consecutive (old, next) pairs — side effects can't
	// be applied against a stale old snapshot, and a read-modify-write can't
	// be interleaved by another Update.
	updateMu sync.Mutex
}

func NewSettingsService(store *SettingsStore, d *Dispatcher) *SettingsService {
	s := &SettingsService{store: store, dispatcher: d}
	s.applyHotkeysFn = s.applyHotkeys
	return s
}

func (s *SettingsService) SetApp(a *application.App)                   { s.app = a }
func (s *SettingsService) SetPrefsWindow(w *application.WebviewWindow) { s.prefs = w }
func (s *SettingsService) SetTray(t *application.SystemTray)           { s.tray = t }

// SetPrefsFactory wires a constructor used to (re)create the prefs window if
// it is missing or was destroyed. Wails v3 alpha offers no hide-on-close
// window option and no IsDestroyed probe (checked against
// WebviewWindowOptions / WebviewWindow docs), so main.go cancels the close
// event and hides instead — this factory is the defensive fallback.
func (s *SettingsService) SetPrefsFactory(f func() *application.WebviewWindow) { s.prefsFactory = f }

func (s *SettingsService) ShowPreferences() {
	platform.ActivatePrefs()
	w := s.prefs
	if w == nil && s.prefsFactory != nil {
		w = s.prefsFactory()
		s.prefs = w
	}
	if w == nil {
		slog.Warn("ShowPreferences: no prefs window wired")
		return
	}
	if !showWindow(w) && s.prefsFactory != nil {
		// Window was destroyed under us (normally WindowClosing cancels the
		// close and hides, so this is a last resort): recreate once.
		w = s.prefsFactory()
		s.prefs = w
		showWindow(w)
	}
}

// showWindow shows+focuses w, recovering if the native window is gone — the
// Wails API has no destroyed-check, so recover() is the only probe available.
func showWindow(w *application.WebviewWindow) (ok bool) {
	defer func() {
		if r := recover(); r != nil {
			slog.Warn("prefs window Show/Focus panicked (window destroyed?)", "recover", r)
			ok = false
		}
	}()
	w.Show()
	w.Focus()
	return true
}

// StartEngine registers hotkeys first, then the drag tap. Idempotent and safe
// for concurrent callers (frontend AX polling can race the launch path).
// engineOn latches only on full success: if StartDragTap fails, hotkeys stay
// registered but engineOn remains false so the next AXTrusted poll retries.
// A retry re-runs applyHotkeys, which is idempotent under hotkeysMu — it
// starts by unregistering everything — so partial-hotkeys-then-retry is safe.
func (s *SettingsService) StartEngine() {
	s.engineMu.Lock()
	defer s.engineMu.Unlock()
	if s.engineOn {
		return
	}
	s.applyHotkeysFn(s.store.Get())
	if err := platform.StartDragTap(func(kind platform.DragEventKind, x, y float64, mod bool) {
		s.dispatcher.OnDragEvent(int(kind), x, y, mod)
	}); err != nil {
		slog.Error("engine: drag tap failed to start; will retry on next AX poll", "err", err)
		return
	}
	s.engineOn = true
}

func (s *SettingsService) engineRunning() bool {
	s.engineMu.Lock()
	defer s.engineMu.Unlock()
	return s.engineOn
}

// MenuOrder is the canonical action order shared by the tray menu and hotkey
// registration: hotkey id == index into MenuOrder(); undo uses undoHotkeyID.
func MenuOrder() []engine.Action {
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
	s.hotkeysMu.Lock()
	defer s.hotkeysMu.Unlock()
	platform.UnregisterAllHotkeys()
	if s.hotkeysSuspended || !cfg.Hotkeys.Enabled {
		return
	}
	platform.InstallHotkeyHandler(func(id uint32) {
		if id == undoHotkeyID {
			s.dispatcher.Undo()
			return
		}
		actions := MenuOrder()
		if int(id) < len(actions) {
			s.dispatcher.Perform(actions[id])
		}
	})
	// Registration failures (e.g. combo already taken by another app) must not
	// be silent: log each one. Surfacing them in the UI is a later ticket.
	for i, a := range MenuOrder() {
		if hk, ok := cfg.Hotkeys.Bindings[a]; ok {
			if err := platform.RegisterHotkey(uint32(i), hk); err != nil {
				slog.Warn("hotkey registration failed", "action", a, "err", err)
			}
		}
	}
	// Undo binding (⌥⌘Y by default) is stored under a pseudo-action key "undo".
	if hk, ok := cfg.Hotkeys.Bindings[engine.Action("undo")]; ok {
		if err := platform.RegisterHotkey(undoHotkeyID, hk); err != nil {
			slog.Warn("hotkey registration failed", "action", "undo", "err", err)
		}
	}
}

// SuspendHotkeys unregisters all global hotkeys and marks registration
// suspended, so StartEngine/applySideEffects skip re-registering until
// ResumeHotkeys is called. Used by the Hotkeys tab while recording a new
// combo: Carbon otherwise intercepts the keydown before it reaches the
// webview, making already-bound combos impossible to capture. Safe to call
// even if the engine isn't running yet.
func (s *SettingsService) SuspendHotkeys() {
	s.hotkeysMu.Lock()
	defer s.hotkeysMu.Unlock()
	platform.UnregisterAllHotkeys()
	s.hotkeysSuspended = true
	if s.afterFunc == nil {
		s.afterFunc = time.AfterFunc
	}
	// A fresh Suspend (e.g. the user starts recording a second binding before
	// the first watchdog would have fired) resets the 60s window rather than
	// stacking timers.
	if s.suspendTimer != nil {
		s.suspendTimer.Stop()
	}
	s.suspendTimer = s.afterFunc(60*time.Second, s.ResumeHotkeys)
}

// ResumeHotkeys clears the suspend flag and, if the engine is running,
// re-registers hotkeys from the current settings. Also stops the watchdog
// timer started by SuspendHotkeys, if any.
func (s *SettingsService) ResumeHotkeys() {
	s.hotkeysMu.Lock()
	s.hotkeysSuspended = false
	if s.suspendTimer != nil {
		s.suspendTimer.Stop()
		s.suspendTimer = nil
	}
	s.hotkeysMu.Unlock()
	if s.engineRunning() {
		s.applyHotkeysFn(s.store.Get())
	}
}

// --- Frontend-bound API ---

func (s *SettingsService) Get() engine.Settings { return s.store.Get() }

func (s *SettingsService) Update(next engine.Settings) error {
	s.updateMu.Lock()
	defer s.updateMu.Unlock()
	return s.updateLocked(next)
}

// updateLocked performs Clamp+Swap+side-effects. Callers must hold updateMu
// — it exists so RestoreDefaultHotkeys can hold updateMu across its whole
// Get→mutate→update sequence without re-entrantly deadlocking on Update.
func (s *SettingsService) updateLocked(next engine.Settings) error {
	next = engine.Clamp(next)
	old, err := s.store.Swap(next)
	if err != nil {
		return err
	}
	s.applySideEffects(old, next)
	if s.app != nil {
		// API drift: event emission lives on the Event manager, not App.EmitEvent.
		s.app.Event.Emit("settings:changed", next)
	}
	return nil
}

// RestoreDefaultHotkeys holds updateMu across the whole Get→mutate→update
// sequence so a concurrent Update can't read the pre-restore settings,
// interleave, and clobber the restore (or vice versa).
func (s *SettingsService) RestoreDefaultHotkeys() engine.Settings {
	s.updateMu.Lock()
	defer s.updateMu.Unlock()
	cur := s.store.Get()
	cur.Hotkeys = engine.DefaultSettings().Hotkeys
	_ = s.updateLocked(cur)
	return cur
}

// AXTrusted reports whether Accessibility permission is granted. The
// onboarding UI polls this after sending the user to System Settings, so a
// freshly granted permission also starts the engine — no relaunch needed.
func (s *SettingsService) AXTrusted() bool {
	ok := platform.AXTrusted(false)
	if ok {
		s.StartEngine()
	}
	return ok
}
func (s *SettingsService) RequestAXPermission() bool {
	ok := platform.AXTrusted(true)
	if ok {
		s.StartEngine()
	}
	return ok
}

// ReconcileStartup re-applies settings whose state lives outside the process
// (login item registration, tray visibility), so a settings file edited while
// the app wasn't running — or drift in the login-item database — is corrected
// at launch. Called from ApplicationDidFinishLaunching regardless of AX
// permission.
func (s *SettingsService) ReconcileStartup() {
	cur := s.store.Get()
	if err := platform.SetLoginItem(cur.General.LaunchAtLogin); err != nil {
		slog.Warn("startup: login item reconcile failed", "err", err)
	}
	if s.tray != nil {
		if cur.General.ShowMenuBarIcon {
			s.tray.Show()
		} else {
			s.tray.Hide()
		}
	}
}

func (s *SettingsService) applySideEffects(old, next engine.Settings) {
	// Only re-register hotkeys when the hotkey config actually changed —
	// otherwise every settings tick (e.g. a padding slider) churns Carbon
	// registrations.
	if s.engineRunning() && !reflect.DeepEqual(old.Hotkeys, next.Hotkeys) {
		s.applyHotkeysFn(next)
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
