package app

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/GuilhermeVozniak/tiles-spliter/desktop/internal/engine"
)

// Swap must return the pre-swap value and leave the store holding next.
func TestSwapReturnsOldAtomically(t *testing.T) {
	st := &SettingsStore{s: engine.DefaultSettings(), path: filepath.Join(t.TempDir(), "settings.json")}
	next := engine.DefaultSettings()
	next.General.WindowPadding = 42
	old, err := st.Swap(next)
	if err != nil {
		t.Fatal(err)
	}
	if old.General.WindowPadding != 0 {
		t.Fatalf("old must be the pre-swap value, got %+v", old.General)
	}
	if st.Get().General.WindowPadding != 42 {
		t.Fatal("store must hold next after Swap")
	}
}

// MenuOrder is the shared tray/hotkey ordering: exactly the 17 layout actions.
func TestMenuOrderHas17Actions(t *testing.T) {
	if got := len(MenuOrder()); got != 17 {
		t.Fatalf("want 17 actions, got %d", got)
	}
}

// A settings update that doesn't touch hotkey config must not re-register
// hotkeys (Carbon churn on every slider tick), while a real hotkey change must.
func TestApplySideEffectsHotkeyChurn(t *testing.T) {
	s := &SettingsService{engineOn: true}
	calls := 0
	s.applyHotkeysFn = func(engine.Settings) { calls++ }

	old := engine.DefaultSettings()
	next := engine.DefaultSettings()
	next.General.WindowPadding = 12 // non-hotkey change (slider tick)
	s.applySideEffects(old, next)
	if calls != 0 {
		t.Fatalf("unchanged hotkeys were re-registered %d times", calls)
	}

	next.Hotkeys.Enabled = false
	s.applySideEffects(old, next)
	if calls != 1 {
		t.Fatalf("changed hotkeys should re-register exactly once, got %d", calls)
	}
}

// The tray-menu refresh callback must fire exactly when the hotkey config
// changes — never on unrelated settings ticks — and must not require the
// engine to be running (the menu exists regardless of AX permission).
func TestApplySideEffectsMenuRefresh(t *testing.T) {
	s := &SettingsService{}
	s.applyHotkeysFn = func(engine.Settings) {}
	refreshes := 0
	s.SetMenuRefresh(func() { refreshes++ })

	old := engine.DefaultSettings()
	next := engine.DefaultSettings()
	next.General.WindowPadding = 12 // non-hotkey change
	s.applySideEffects(old, next)
	if refreshes != 0 {
		t.Fatalf("menu refreshed %d times on a non-hotkey change", refreshes)
	}

	next.Hotkeys.Bindings[engine.ActionCenter] = engine.Hotkey{KeyCode: 3, Modifiers: 256}
	s.applySideEffects(old, next)
	if refreshes != 1 {
		t.Fatalf("hotkey remap with engine off must refresh the menu once, got %d", refreshes)
	}
}

// Engine off: even a hotkey change must not register anything yet.
func TestApplySideEffectsEngineOff(t *testing.T) {
	s := &SettingsService{}
	calls := 0
	s.applyHotkeysFn = func(engine.Settings) { calls++ }

	old := engine.DefaultSettings()
	next := engine.DefaultSettings()
	next.Hotkeys.Enabled = false
	s.applySideEffects(old, next)
	if calls != 0 {
		t.Fatalf("engine off: applyHotkeys called %d times", calls)
	}
}

// SuspendHotkeys marks registration suspended; ResumeHotkeys clears it and,
// with the engine running, re-applies hotkeys exactly once via the
// applyHotkeysFn seam (so this doesn't touch real Carbon registration).
func TestSuspendResumeHotkeys(t *testing.T) {
	s := &SettingsService{
		engineOn: true,
		store:    &SettingsStore{s: engine.DefaultSettings(), path: filepath.Join(t.TempDir(), "settings.json")},
	}
	calls := 0
	s.applyHotkeysFn = func(engine.Settings) { calls++ }

	s.SuspendHotkeys()
	if !s.hotkeysSuspended {
		t.Fatal("SuspendHotkeys must set the suspended flag")
	}
	if calls != 0 {
		t.Fatalf("SuspendHotkeys must not re-apply hotkeys, got %d calls", calls)
	}

	s.ResumeHotkeys()
	if s.hotkeysSuspended {
		t.Fatal("ResumeHotkeys must clear the suspended flag")
	}
	if calls != 1 {
		t.Fatalf("ResumeHotkeys with engine running must re-apply hotkeys once, got %d", calls)
	}
}

// With the engine not yet running, ResumeHotkeys must clear the flag but not
// trigger registration — StartEngine will apply hotkeys itself on launch.
func TestResumeHotkeysNoopWhenEngineOff(t *testing.T) {
	s := &SettingsService{
		store: &SettingsStore{s: engine.DefaultSettings(), path: filepath.Join(t.TempDir(), "settings.json")},
	}
	calls := 0
	s.applyHotkeysFn = func(engine.Settings) { calls++ }

	s.SuspendHotkeys()
	s.ResumeHotkeys()
	if s.hotkeysSuspended {
		t.Fatal("ResumeHotkeys must clear the suspended flag even when engine is off")
	}
	if calls != 0 {
		t.Fatalf("ResumeHotkeys with engine off must not re-apply hotkeys, got %d", calls)
	}
}

// SuspendHotkeys must arm a 60s watchdog; if the frontend never calls
// ResumeHotkeys (e.g. the webview died mid-recording), the watchdog itself
// must resume hotkeys so they don't stay suspended forever.
func TestSuspendHotkeysWatchdogResumesOnTimeout(t *testing.T) {
	s := &SettingsService{
		store: &SettingsStore{s: engine.DefaultSettings(), path: filepath.Join(t.TempDir(), "settings.json")},
	}
	var scheduledAfter time.Duration
	var fired func()
	s.afterFunc = func(d time.Duration, f func()) *time.Timer {
		scheduledAfter = d
		fired = f
		return time.AfterFunc(time.Hour, func() {}) // never actually fires in the test
	}

	s.SuspendHotkeys()
	if scheduledAfter != 60*time.Second {
		t.Fatalf("want a 60s watchdog, got %v", scheduledAfter)
	}
	if s.suspendTimer == nil {
		t.Fatal("SuspendHotkeys must store the watchdog timer")
	}
	if fired == nil {
		t.Fatal("SuspendHotkeys must schedule the watchdog callback")
	}

	// Simulate the 60s timeout firing (rather than the frontend calling
	// ResumeHotkeys) — it must resume hotkeys itself.
	fired()
	if s.hotkeysSuspended {
		t.Fatal("watchdog firing must clear hotkeysSuspended")
	}
}

// A fresh SuspendHotkeys call (e.g. recording a second binding) must reset
// the watchdog rather than stack timers, and ResumeHotkeys must stop it so a
// stale watchdog can't fire after a normal resume.
func TestResumeHotkeysStopsWatchdog(t *testing.T) {
	s := &SettingsService{
		store: &SettingsStore{s: engine.DefaultSettings(), path: filepath.Join(t.TempDir(), "settings.json")},
	}
	calls := 0
	s.afterFunc = func(d time.Duration, f func()) *time.Timer {
		calls++
		return time.AfterFunc(time.Hour, func() {})
	}

	s.SuspendHotkeys()
	first := s.suspendTimer
	s.SuspendHotkeys() // fresh suspend: must reset, not stack
	if calls != 2 {
		t.Fatalf("want afterFunc scheduled twice, got %d", calls)
	}
	if s.suspendTimer == first {
		t.Fatal("a fresh Suspend must install a new watchdog timer")
	}

	s.ResumeHotkeys()
	if s.suspendTimer != nil {
		t.Fatal("ResumeHotkeys must clear the watchdog timer")
	}
}

// applyHotkeys itself (not just the seam) must honor the suspended flag: it
// unregisters (idempotent/no-op with nothing registered) but must not
// install the Carbon handler or attempt registration while suspended.
func TestApplyHotkeysSkipsWhenSuspended(t *testing.T) {
	s := NewSettingsService(&SettingsStore{s: engine.DefaultSettings(), path: filepath.Join(t.TempDir(), "settings.json")}, nil)
	s.hotkeysSuspended = true
	// Must return promptly without panicking or blocking on Carbon calls.
	s.applyHotkeys(engine.DefaultSettings())
}
