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

// A partial settings file must only override the fields it actually contains:
// missing sections keep their defaults, and an explicit false sticks.
func TestLoadPartialFileKeepsDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	os.WriteFile(path, []byte(`{"snap":{"enabled":false}}`), 0o644)
	got := LoadSettings(path)
	if got.Snap.Enabled {
		t.Fatal("explicit snap.enabled=false must stick")
	}
	if !got.General.LaunchAtLogin || !got.General.ShowMenuBarIcon {
		t.Fatal("missing general section must keep defaults")
	}
	if !got.Hotkeys.Enabled || len(got.Hotkeys.Bindings) != 18 {
		t.Fatal("missing hotkeys section must keep default bindings")
	}
	if got.Snap.ZoneThickness != 10 {
		t.Fatal("missing snap fields must keep defaults")
	}
}

func TestClampAbsurdValues(t *testing.T) {
	s := DefaultSettings()
	s.General.WindowPadding = 10000
	s.Snap.ZoneThickness = -5
	s.Snap.ActivationDelayMs = 99999
	s.Hotkeys.Bindings[ActionCenter] = Hotkey{KeyCode: 999, Modifiers: 256}
	got := Clamp(s)
	if got.General.WindowPadding != 100 {
		t.Fatalf("padding clamped to 100, got %v", got.General.WindowPadding)
	}
	if got.Snap.ZoneThickness != 1 {
		t.Fatalf("thickness clamped to 1, got %v", got.Snap.ZoneThickness)
	}
	if got.Snap.ActivationDelayMs != 5000 {
		t.Fatalf("delay clamped to 5000, got %v", got.Snap.ActivationDelayMs)
	}
	if _, ok := got.Hotkeys.Bindings[ActionCenter]; ok {
		t.Fatal("binding with keyCode>127 must be dropped")
	}
	if _, ok := got.Hotkeys.Bindings[ActionFullscreen]; !ok {
		t.Fatal("valid bindings must be kept")
	}
}

func TestClampUnknownZoneAction(t *testing.T) {
	s := DefaultSettings()
	s.Snap.Zones[ZoneTop] = Action("explode")
	got := Clamp(s)
	if got.Snap.Zones[ZoneTop] != ActionNone {
		t.Fatalf("unknown zone action must become none, got %q", got.Snap.Zones[ZoneTop])
	}
	if got.Snap.Zones[ZoneLeft] != ActionHalfLeft {
		t.Fatal("known zone actions must be kept")
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
