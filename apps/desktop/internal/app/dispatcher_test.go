package app

import (
	"testing"

	"github.com/GuilhermeVozniak/tiles-spliter/desktop/internal/engine"
)

var disp = engine.Display{ID: 1, Frame: engine.Rect{X: 0, Y: 0, W: 1440, H: 900}, Visible: engine.Rect{X: 0, Y: 25, W: 1440, H: 815}}

type fakeWin struct {
	id    uint32
	frame engine.Rect
	sets  []engine.Rect
}

func (w *fakeWin) WinID() uint32                { return w.id }
func (w *fakeWin) Frame() (engine.Rect, error)  { return w.frame, nil }
func (w *fakeWin) SetFrame(r engine.Rect) error { w.frame = r; w.sets = append(w.sets, r); return nil }
func (w *fakeWin) Release()                     {}

type fakePlatform struct {
	win      *fakeWin
	overlays int
}

func (p *fakePlatform) FocusedWindow() (AppWindow, error)        { return p.win, nil }
func (p *fakePlatform) WindowAt(engine.Point) (AppWindow, error) { return p.win, nil }
func (p *fakePlatform) Displays() []engine.Display               { return []engine.Display{disp} }
func (p *fakePlatform) ShowOverlay(engine.Rect)                  { p.overlays++ }
func (p *fakePlatform) HideOverlay()                             {}

func newTestDispatcher(w *fakeWin) (*Dispatcher, *fakePlatform) {
	p := &fakePlatform{win: w}
	s := engine.DefaultSettings()
	s.General.EnableAnimations = false
	d := NewDispatcher(p, func() engine.Settings { return s })
	d.runAsync = func(f func()) { f() } // deterministic drops in tests
	return d, p
}

func TestPerformHalfLeftMovesWindow(t *testing.T) {
	w := &fakeWin{id: 1, frame: engine.Rect{X: 100, Y: 100, W: 400, H: 300}}
	d, _ := newTestDispatcher(w)
	d.Perform(engine.ActionHalfLeft)
	want, _ := engine.FrameFor(engine.ActionHalfLeft, engine.Rect{}, disp, 0, false)
	if !w.frame.Eq(want, 0.01) {
		t.Fatalf("got %+v want %+v", w.frame, want)
	}
}

func TestPerformThenUndoRestores(t *testing.T) {
	orig := engine.Rect{X: 100, Y: 100, W: 400, H: 300}
	w := &fakeWin{id: 1, frame: orig}
	d, _ := newTestDispatcher(w)
	d.Perform(engine.ActionFullscreen)
	d.Undo()
	if !w.frame.Eq(orig, 0.01) {
		t.Fatalf("undo should restore %+v, got %+v", orig, w.frame)
	}
}

func TestNextThirdCyclesViaDispatcher(t *testing.T) {
	w := &fakeWin{id: 1, frame: engine.Rect{X: 100, Y: 100, W: 400, H: 300}}
	d, _ := newTestDispatcher(w)
	d.Perform(engine.ActionNextThird)
	if !w.frame.Eq(engine.ThirdColumn(0, disp, 0), 0.01) {
		t.Fatal("first next-third → left column")
	}
	d.Perform(engine.ActionNextThird)
	if !w.frame.Eq(engine.ThirdColumn(1, disp, 0), 0.01) {
		t.Fatal("second next-third → middle column")
	}
}

func TestDragSnapEndToEnd(t *testing.T) {
	w := &fakeWin{id: 9, frame: engine.Rect{X: 300, Y: 300, W: 500, H: 400}}
	d, p := newTestDispatcher(w)
	d.OnDragEvent(0, 320, 310, false) // mouse-down on the window
	d.OnDragEvent(1, 5, 450, false)   // drag into left edge zone
	if p.overlays == 0 {
		t.Fatal("overlay shown while hovering zone")
	}
	d.OnDragEvent(2, 5, 450, false) // drop
	want, _ := engine.FrameFor(engine.ActionHalfLeft, engine.Rect{}, disp, 0, false)
	if !w.frame.Eq(want, 0.01) {
		t.Fatalf("dropped window snapped, got %+v", w.frame)
	}
}
