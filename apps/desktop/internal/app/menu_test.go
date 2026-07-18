package app

import (
	"testing"

	"github.com/GuilhermeVozniak/tiles-spliter/desktop/internal/engine"
)

// Hotkey ids are indexes into MenuOrder(), so the sectioned menu must flatten
// to exactly MenuOrder — same actions, same order.
func TestMenuSectionsFlattenToMenuOrder(t *testing.T) {
	var flat []engine.Action
	for _, section := range MenuSections() {
		if len(section) == 0 {
			t.Fatal("empty menu section")
		}
		flat = append(flat, section...)
	}
	order := MenuOrder()
	if len(flat) != len(order) {
		t.Fatalf("flatten(MenuSections) has %d actions, MenuOrder has %d", len(flat), len(order))
	}
	for i := range order {
		if flat[i] != order[i] {
			t.Fatalf("index %d: sections give %q, MenuOrder gives %q", i, flat[i], order[i])
		}
	}
}

func TestAcceleratorString(t *testing.T) {
	cases := []struct {
		name string
		hk   engine.Hotkey
		want string
	}{
		{"center default ⌥⌘C", engine.Hotkey{KeyCode: 8, Modifiers: 256 + 2048}, "option+cmd+c"},
		{"half-left ⌥⌘←", engine.Hotkey{KeyCode: 123, Modifiers: 256 + 2048}, "option+cmd+left"},
		{"next-third ⌃⌥→", engine.Hotkey{KeyCode: 124, Modifiers: 4096 + 2048}, "ctrl+option+right"},
		{"next-display ⌃⌥⌘→", engine.Hotkey{KeyCode: 124, Modifiers: 256 + 2048 + 4096}, "ctrl+option+cmd+right"},
		{"shifted digit", engine.Hotkey{KeyCode: 18, Modifiers: 512 + 256}, "shift+cmd+1"},
		{"space up-arrowless", engine.Hotkey{KeyCode: 49, Modifiers: 256}, "cmd+space"},
		{"bare key, no modifiers", engine.Hotkey{KeyCode: 8}, "c"},
		{"unmapped key code", engine.Hotkey{KeyCode: 50, Modifiers: 256}, ""},
		{"garbage key code", engine.Hotkey{KeyCode: 127, Modifiers: 256}, ""},
	}
	for _, tc := range cases {
		if got := AcceleratorString(tc.hk); got != tc.want {
			t.Errorf("%s: got %q want %q", tc.name, got, tc.want)
		}
	}
}

// Every default binding for the 17 menu actions plus undo must produce a
// non-empty accelerator hint — defaults may never silently lose their hints.
func TestAllDefaultBindingsHaveAcceleratorStrings(t *testing.T) {
	s := engine.DefaultSettings()
	for a, hk := range s.Hotkeys.Bindings {
		if AcceleratorString(hk) == "" {
			t.Errorf("default binding for %q has no accelerator string (%+v)", a, hk)
		}
	}
}
