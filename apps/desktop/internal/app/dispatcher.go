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
	runAsync    func(func()) // seam: tests run the drop handler synchronously

	dragWin  AppWindow    // window resolved once a genuine drag starts, nil otherwise
	dragDown bool         // mouse is down and we are watching for drag movement
	downPt   engine.Point // mouse-down location
}

// dragThresholdSq gates drag-start: the cursor must move more than 4pt from
// the mouse-down point before the gesture counts as a drag rather than a click.
const dragThresholdSq = 16.0

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
	d.runAsync = func(f func()) { go f() }
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
	// Read the frame before popping so a transient AX failure does not
	// silently consume the undo record.
	cur, err := w.Frame()
	if err != nil {
		return
	}
	prev, ok := d.undo.Pop(w.WinID())
	if !ok {
		return
	}
	d.animate(w, cur, prev)
}

// OnDragEvent receives raw tap events (kind: 0=down 1=moved 2=up).
func (d *Dispatcher) OnDragEvent(kind int, x, y float64, modThirds bool) {
	p := engine.Point{X: x, Y: y}
	switch kind {
	case 0: // down: record the point only — a click must not touch AX or snap
		if d.dragWin != nil {
			d.dragWin.Release()
			d.dragWin = nil
		}
		d.dragDown = false
		// WindowAt is a blocking AX round-trip on the main run loop — skip
		// drag tracking entirely when drag-snapping is disabled.
		if !d.getSettings().Snap.Enabled {
			return
		}
		d.dragDown = true
		d.downPt = p
	case 1:
		if d.dragWin == nil {
			if !d.dragDown {
				return
			}
			dx, dy := x-d.downPt.X, y-d.downPt.Y
			if dx*dx+dy*dy <= dragThresholdSq {
				return // still within click slop; not a drag yet
			}
			// Genuine drag: resolve the window once, at the mouse-down point.
			d.dragDown = false // one attempt per gesture, even on AX failure
			w, err := d.plat.WindowAt(d.downPt)
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
		}
		d.snap.DragMove(p, modThirds)
	case 2:
		d.dragDown = false
		if d.dragWin == nil {
			return
		}
		w := d.dragWin
		d.dragWin = nil // cleared synchronously; the goroutine owns the ref now
		if _, frame, apply := d.snap.DragEnd(p, modThirds); apply {
			if cur, err := w.Frame(); err == nil {
				// The drop animation is ~130ms of AX calls + sleeps; it must not
				// run inside the event-tap callback on the main run loop.
				d.runAsync(func() {
					d.undo.Push(w.WinID(), cur)
					d.animate(w, cur, frame)
					w.Release()
				})
				return
			}
		}
		w.Release()
	}
}
