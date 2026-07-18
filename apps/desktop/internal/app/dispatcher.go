// Package app glues the pure engine to the native platform layer.
package app

import (
	"log/slog"
	"time"

	"github.com/GuilhermeVozniak/tiles-spliter/desktop/internal/engine"
)

type AppWindow interface {
	WinID() uint32
	Frame() (engine.Rect, error)
	SetFrame(engine.Rect) error
	Release()
}

type Platform interface {
	FocusedWindow() (AppWindow, error)
	WindowAt(engine.Point) (AppWindow, error)
	Displays() []engine.Display
	ShowOverlay(engine.Rect)
	HideOverlay()
}

type Dispatcher struct {
	plat        Platform
	getSettings func() engine.Settings
	undo        *engine.UndoStack
	snap        *engine.SnapController
	animate     func(w AppWindow, from, to engine.Rect)

	dragWin AppWindow // window picked up at mouse-down, nil otherwise
}

// snapPlatformAdapter adds Now() to the overlay pair for the SnapController.
type snapPlatformAdapter struct{ p Platform }

func (a snapPlatformAdapter) ShowOverlay(r engine.Rect) { a.p.ShowOverlay(r) }
func (a snapPlatformAdapter) HideOverlay()              { a.p.HideOverlay() }
func (a snapPlatformAdapter) Now() int64                { return time.Now().UnixMilli() }

func NewDispatcher(p Platform, getSettings func() engine.Settings) *Dispatcher {
	d := &Dispatcher{
		plat:        p,
		getSettings: getSettings,
		undo:        engine.NewUndoStack(),
	}
	d.snap = engine.NewSnapController(snapPlatformAdapter{p}, getSettings, p.Displays)
	d.animate = func(w AppWindow, from, to engine.Rect) {
		enabled := getSettings().General.EnableAnimations
		animateFrame(w, from, to, enabled)
	}
	return d
}

// Snap returns the underlying SnapController.
func (d *Dispatcher) Snap() *engine.SnapController { return d.snap }

// Perform runs one of the 17 actions on the currently focused window.
func (d *Dispatcher) Perform(a engine.Action) {
	w, err := d.plat.FocusedWindow()
	if err != nil {
		slog.Warn("no focused window", "action", a, "err", err)
		return
	}
	defer w.Release()
	cur, err := w.Frame()
	if err != nil {
		return
	}
	displays := d.plat.Displays()
	if len(displays) == 0 {
		return
	}
	s := d.getSettings()
	di := engine.DisplayOf(cur, displays)
	disp := displays[di]
	pad := s.General.WindowPadding

	var target engine.Rect
	var ok bool
	switch a {
	case engine.ActionNextThird, engine.ActionPrevThird:
		target, ok = engine.ThirdFrame(a, cur, disp, pad)
	case engine.ActionNextDisplay, engine.ActionPrevDisplay:
		to := engine.AdjacentDisplay(di, displays, a == engine.ActionNextDisplay)
		if to != di {
			target, ok = engine.MapToDisplay(cur, disp, displays[to]), true
		}
	default:
		target, ok = engine.FrameFor(a, cur, disp, pad, s.General.PadFullscreen)
	}
	if !ok {
		return
	}
	d.undo.Push(w.WinID(), cur)
	d.animate(w, cur, target)
}

func (d *Dispatcher) Undo() {
	w, err := d.plat.FocusedWindow()
	if err != nil {
		return
	}
	defer w.Release()
	prev, ok := d.undo.Pop(w.WinID())
	if !ok {
		return
	}
	cur, err := w.Frame()
	if err != nil {
		return
	}
	d.animate(w, cur, prev)
}

// OnDragEvent receives raw tap events (kind: 0=down 1=moved 2=up).
func (d *Dispatcher) OnDragEvent(kind int, x, y float64, modThirds bool) {
	p := engine.Point{X: x, Y: y}
	switch kind {
	case 0: // down: resolve the window once
		if d.dragWin != nil {
			d.dragWin.Release()
			d.dragWin = nil
		}
		w, err := d.plat.WindowAt(p)
		if err != nil {
			return
		}
		frame, err := w.Frame()
		if err != nil {
			w.Release()
			return
		}
		d.dragWin = w
		if restore, ok := d.snap.DragStart(engine.DragWindow{ID: w.WinID(), Frame: frame}); ok {
			// Restore previous size under the cursor (keep grab point proportional).
			_ = w.SetFrame(engine.Rect{X: x - restore.W/2, Y: frame.Y, W: restore.W, H: restore.H})
		}
	case 1:
		if d.dragWin != nil {
			d.snap.DragMove(p, modThirds)
		}
	case 2:
		if d.dragWin == nil {
			return
		}
		if _, frame, apply := d.snap.DragEnd(p, modThirds); apply {
			if cur, err := d.dragWin.Frame(); err == nil {
				d.undo.Push(d.dragWin.WinID(), cur)
				d.animate(d.dragWin, cur, frame)
			}
		}
		d.dragWin.Release()
		d.dragWin = nil
	}
}
