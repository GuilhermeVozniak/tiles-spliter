package app

import (
	"path/filepath"
	"testing"

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
