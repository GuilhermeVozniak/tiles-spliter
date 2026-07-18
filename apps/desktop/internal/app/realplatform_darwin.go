//go:build darwin

package app

import (
	"github.com/GuilhermeVozniak/tiles-spliter/desktop/internal/engine"
	"github.com/GuilhermeVozniak/tiles-spliter/desktop/internal/platform"
)

// RealPlatform adapts the cgo platform package to the Platform interface.
type RealPlatform struct{}

func (RealPlatform) FocusedWindow() (AppWindow, error) {
	w, err := platform.FocusedWindow()
	return realWindow{w}, err
}
func (RealPlatform) WindowAt(p engine.Point) (AppWindow, error) {
	w, err := platform.WindowAt(p)
	return realWindow{w}, err
}
func (RealPlatform) Displays() []engine.Display { return platform.Displays() }
func (RealPlatform) ShowOverlay(r engine.Rect)  { platform.ShowOverlay(r) }
func (RealPlatform) HideOverlay()               { platform.HideOverlay() }

type realWindow struct{ w *platform.Window }

func (r realWindow) WinID() uint32 {
	if r.w == nil {
		return 0
	}
	return r.w.ID
}
func (r realWindow) Frame() (engine.Rect, error)  { return r.w.Frame() }
func (r realWindow) SetFrame(f engine.Rect) error { return r.w.SetFrame(f) }
func (r realWindow) Release() {
	if r.w != nil {
		r.w.Release()
	}
}
