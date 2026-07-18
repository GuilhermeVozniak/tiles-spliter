//go:build darwin

// Package platform is the cgo/Objective-C bridge to macOS. It contains no
// logic beyond translation — all decisions live in internal/engine.
package platform

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa -framework ApplicationServices -framework Carbon
#include "ax_darwin.h"
*/
import "C"

import (
	"errors"

	"github.com/GuilhermeVozniak/tiles-spliter/desktop/internal/engine"
)

var ErrNoWindow = errors.New("platform: no accessible window")

// AXTrusted reports (and with prompt=true requests) Accessibility permission.
func AXTrusted(prompt bool) bool { return bool(C.ts_ax_trusted(C.bool(prompt))) }

type Window struct {
	ID  uint32
	ref C.AXUIElementRef
}

func wrap(ref C.AXUIElementRef) (*Window, error) {
	if ref == 0 {
		return nil, ErrNoWindow
	}
	return &Window{ID: uint32(C.ts_window_id(ref)), ref: ref}, nil
}

func FocusedWindow() (*Window, error) { return wrap(C.ts_focused_window()) }

func WindowAt(p engine.Point) (*Window, error) {
	return wrap(C.ts_window_at(C.double(p.X), C.double(p.Y)))
}

func (w *Window) Frame() (engine.Rect, error) {
	f := C.ts_window_frame(w.ref)
	if f.ok == 0 {
		return engine.Rect{}, ErrNoWindow
	}
	return engine.Rect{X: float64(f.x), Y: float64(f.y), W: float64(f.w), H: float64(f.h)}, nil
}

func (w *Window) SetFrame(r engine.Rect) error {
	if !bool(C.ts_set_window_frame(w.ref, C.double(r.X), C.double(r.Y), C.double(r.W), C.double(r.H))) {
		return errors.New("platform: window rejected frame (not resizable?)")
	}
	return nil
}

// SetFrameFast is the cheap two-call variant of SetFrame (no final size
// re-set). Use for intermediate animation steps only; see FastFrameSetter.
func (w *Window) SetFrameFast(r engine.Rect) error {
	if !bool(C.ts_set_window_frame_fast(w.ref, C.double(r.X), C.double(r.Y), C.double(r.W), C.double(r.H))) {
		return errors.New("platform: window rejected frame (not resizable?)")
	}
	return nil
}

func (w *Window) Release() { C.ts_release(w.ref); w.ref = 0 }
