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

// InstallHotkeyHandler installs the Carbon event handler. Call once, from the
// main thread, before any RegisterHotkey.
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

func UnregisterAllHotkeys() {
	hotkeyMu.Lock()
	defer hotkeyMu.Unlock()
	for _, ref := range hotkeyRefs {
		C.ts_hk_unregister(ref)
	}
	hotkeyRefs = nil
}

// RunLoop blocks running the current thread's CFRunLoop (probe/testing only).
func RunLoop() { C.CFRunLoopRun() }
