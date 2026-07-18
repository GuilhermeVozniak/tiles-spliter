//go:build darwin

package platform

/*
#include "drag_darwin.h"
*/
import "C"

import (
	"errors"
	"sync"
)

type DragEventKind int

const (
	DragDown DragEventKind = iota
	DragMoved
	DragUp
)

var (
	dragMu      sync.Mutex
	dragHandler func(kind DragEventKind, x, y float64, modThirds bool)
)

//export goDragEvent
func goDragEvent(kind C.int, x, y C.double, modThirds C.int) {
	dragMu.Lock()
	h := dragHandler
	dragMu.Unlock()
	if h != nil {
		h(DragEventKind(kind), float64(x), float64(y), modThirds != 0)
	}
}

// StartDragTap installs a listen-only session event tap on the main run loop.
// Requires Accessibility permission. The handler runs on the tap thread — keep
// it fast (the SnapController is lock-cheap and does no AX calls on DragMoved).
func StartDragTap(handler func(kind DragEventKind, x, y float64, modThirds bool)) error {
	dragMu.Lock()
	dragHandler = handler
	dragMu.Unlock()
	if !bool(C.ts_tap_start()) {
		return errors.New("platform: CGEventTapCreate failed (missing Accessibility permission?)")
	}
	return nil
}

func StopDragTap() {
	C.ts_tap_stop()
	dragMu.Lock()
	dragHandler = nil
	dragMu.Unlock()
}
