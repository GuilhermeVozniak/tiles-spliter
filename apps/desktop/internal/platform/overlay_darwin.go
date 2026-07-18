//go:build darwin

package platform

/*
#include "overlay_darwin.h"
*/
import "C"

import "github.com/GuilhermeVozniak/tiles-spliter/desktop/internal/engine"

// ShowOverlay shows (or moves) the translucent snap preview. Safe from any thread.
func ShowOverlay(r engine.Rect) {
	C.ts_overlay_show(C.double(r.X), C.double(r.Y), C.double(r.W), C.double(r.H))
}

func HideOverlay() { C.ts_overlay_hide() }
