package engine

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultsMatchSpec(t *testing.T) {
	s := DefaultSettings()
	if !s.General.LaunchAtLogin || !s.General.ShowMenuBarIcon || !s.General.EnableAnimations {
		t.Fatal("general defaults on")
	}
	if s.General.WindowPadding != 0 || s.General.PadFullscreen {
		t.Fatal("padding defaults")
	}
	if s.Snap.ZoneThickness != 10 || s.Snap.ActivationDelayMs != 0 || !s.Snap.RestorePreviousSize {
		t.Fatal("snap defaults")
	}
	if s.Snap.Zones[ZoneLeft] != ActionHalfLeft || s.Snap.Zones[ZoneTop] != ActionFullscreen || s.Snap.Zones[ZoneBottom] != ActionNone {
		t.Fatal("zone defaults")
	}
	if s.Hotkeys.Bindings[ActionCenter] != (Hotkey{KeyCode: 8, Modifiers: 256 + 2048}) {
		t.Fatal("center hotkey ⌥⌘C")
	}
	if len(s.Hotkeys.Bindings) != 18 {
		t.Fatalf("18 default bindings, got %d", len(s.Hotkeys.Bindings))
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "settings.json")
	s := DefaultSettings()
	s.General.WindowPadding = 12
	s.Snap.Zones[ZoneBottom] = ActionHalfBottom
	if err := s.Save(path); err != nil {
		t.Fatal(err)
	}
	got := LoadSettings(path)
	if got.General.WindowPadding != 12 || got.Snap.Zones[ZoneBottom] != ActionHalfBottom {
		t.Fatalf("round trip lost data: %+v", got)
	}
}

func TestLoadMissingReturnsDefaults(t *testing.T) {
	got := LoadSettings(filepath.Join(t.TempDir(), "nope.json"))
	if got.Snap.ZoneThickness != 10 {
		t.Fatal("want defaults")
	}
}

func TestLoadCorruptBacksUpAndDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	os.WriteFile(path, []byte("{not json"), 0o644)
	got := LoadSettings(path)
	if got.Snap.ZoneThickness != 10 {
		t.Fatal("want defaults on corrupt file")
	}
	if _, err := os.Stat(path + ".bak"); err != nil {
		t.Fatal("corrupt file should be backed up")
	}
}
