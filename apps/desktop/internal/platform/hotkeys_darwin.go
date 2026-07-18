//go:build darwin

package platform

/*
#include "hotkeys_darwin.h"
*/
import "C"

import (
	"fmt"
	"sync"
	"unsafe"

	"github.com/GuilhermeVozniak/tiles-spliter/desktop/internal/engine"
)

var (
	hotkeyMu   sync.Mutex
	hotkeyRefs []unsafe.Pointer
	hotkeyFire func(id uint32)
)

//export goHotkeyFired
func goHotkeyFired(id C.uint) {
	hotkeyMu.Lock()
	fire := hotkeyFire
	hotkeyMu.Unlock()
	if fire != nil {
		go fire(uint32(id))
	}
}

// InstallHotkeyHandler installs the Carbon event handler before any
// RegisterHotkey. Safe to call from any goroutine — the C side funnels the
// Carbon calls to the main thread — and idempotent (install happens once).
func InstallHotkeyHandler(fire func(id uint32)) {
	hotkeyMu.Lock()
	hotkeyFire = fire
	hotkeyMu.Unlock()
	C.ts_hk_install()
}

func RegisterHotkey(id uint32, hk engine.Hotkey) error {
	ref := C.ts_hk_register(C.uint(id), C.uint(hk.KeyCode), C.uint(hk.Modifiers))
	if ref == nil {
		return fmt.Errorf("platform: hotkey %d (key %d mods %d) rejected — already taken?", id, hk.KeyCode, hk.Modifiers)
	}
	hotkeyMu.Lock()
	hotkeyRefs = append(hotkeyRefs, ref)
	hotkeyMu.Unlock()
	return nil
}

// UnregisterAllHotkeys drops every registration. hotkeyMu must NOT be held
// across the C calls: they dispatch_sync to the main queue, and the main
// thread's hotkey handler (goHotkeyFired) takes hotkeyMu — holding it here
// while waiting on the main queue could deadlock.
func UnregisterAllHotkeys() {
	hotkeyMu.Lock()
	refs := hotkeyRefs
	hotkeyRefs = nil
	hotkeyMu.Unlock()
	for _, ref := range refs {
		C.ts_hk_unregister(ref)
	}
}

// RunLoop blocks running the current thread's CFRunLoop (probe/testing only).
func RunLoop() { C.CFRunLoopRun() }
