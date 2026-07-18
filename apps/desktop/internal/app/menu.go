package app

import (
	"strings"

	"github.com/GuilhermeVozniak/tiles-spliter/desktop/internal/engine"
)

// MenuSections groups the layout actions into the tray-menu sections used by
// the original Tiles app, in display order. Flattened it MUST equal
// MenuOrder() exactly — hotkey ids are indexes into MenuOrder, so the grouping
// may never reorder actions (asserted by TestMenuSectionsFlattenToMenuOrder).
func MenuSections() [][]engine.Action {
	return [][]engine.Action{
		{engine.ActionCenter, engine.ActionFullscreen},
		{engine.ActionHalfLeft, engine.ActionHalfRight, engine.ActionHalfTop, engine.ActionHalfBottom},
		{engine.ActionUpperLeft, engine.ActionUpperRight, engine.ActionLowerLeft, engine.ActionLowerRight},
		{engine.ActionNextThird, engine.ActionPrevThird, engine.ActionTwoThirdsLeft, engine.ActionTwoThirdsRight, engine.ActionTwoThirdsCenter},
		{engine.ActionNextDisplay, engine.ActionPrevDisplay},
	}
}

// Carbon modifier masks as stored in engine.Hotkey.Modifiers (cmdKey,
// shiftKey, optionKey, controlKey from Carbon's Events.h). They mirror the
// unexported mod* constants in the engine package and MOD in packages/shared.
const (
	carbonCmd   = 256
	carbonShift = 512
	carbonOpt   = 2048
	carbonCtrl  = 4096
)

// carbonKeyNames maps Carbon virtual key codes to Wails accelerator key names
// (the lower-cased names parseAccelerator accepts; arrows are "left"/"right"/
// "up"/"down"). It mirrors the KEY_NAMES set in packages/shared: letters,
// digits, space and arrows — the keys the hotkey recorder can produce.
var carbonKeyNames = map[uint32]string{
	0: "a", 1: "s", 2: "d", 3: "f", 4: "h", 5: "g", 6: "z", 7: "x",
	8: "c", 9: "v", 11: "b", 12: "q", 13: "w", 14: "e", 15: "r", 16: "y",
	17: "t", 31: "o", 32: "u", 34: "i", 35: "p", 37: "l", 38: "j", 40: "k",
	45: "n", 46: "m",
	18: "1", 19: "2", 20: "3", 21: "4", 22: "6", 23: "5",
	25: "9", 26: "7", 28: "8", 29: "0",
	49:  "space",
	123: "left", 124: "right", 125: "down", 126: "up",
}

// AcceleratorString converts a Carbon hotkey to the Wails accelerator syntax
// ("ctrl+option+cmd+right") used by MenuItem.SetAccelerator. On macOS Wails
// renders it as the native glyph hint (⌃⌥⌘→). Returns "" for key codes with
// no name — callers should then show no accelerator at all.
func AcceleratorString(hk engine.Hotkey) string {
	key, ok := carbonKeyNames[hk.KeyCode]
	if !ok {
		return ""
	}
	parts := make([]string, 0, 5)
	if hk.Modifiers&carbonCtrl != 0 {
		parts = append(parts, "ctrl")
	}
	if hk.Modifiers&carbonOpt != 0 {
		parts = append(parts, "option")
	}
	if hk.Modifiers&carbonShift != 0 {
		parts = append(parts, "shift")
	}
	if hk.Modifiers&carbonCmd != 0 {
		parts = append(parts, "cmd")
	}
	parts = append(parts, key)
	return strings.Join(parts, "+")
}
