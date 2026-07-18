package app

import (
	"testing"

	"github.com/GuilhermeVozniak/tiles-spliter/desktop/internal/engine"
)

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
