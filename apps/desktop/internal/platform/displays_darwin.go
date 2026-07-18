//go:build darwin

package platform

/*
#include "displays_darwin.h"
*/
import "C"

import "github.com/GuilhermeVozniak/tiles-spliter/desktop/internal/engine"

// Displays returns all screens in top-left-origin coordinates.
func Displays() []engine.Display {
	var buf [16]C.TSDisplay
	n := int(C.ts_displays(&buf[0], 16))
	out := make([]engine.Display, 0, n)
	for i := 0; i < n; i++ {
		d := buf[i]
		out = append(out, engine.Display{
			ID:      uint32(d.id),
			Frame:   engine.Rect{X: float64(d.fx), Y: float64(d.fy), W: float64(d.fw), H: float64(d.fh)},
			Visible: engine.Rect{X: float64(d.vx), Y: float64(d.vy), W: float64(d.vw), H: float64(d.vh)},
		})
	}
	return out
}
