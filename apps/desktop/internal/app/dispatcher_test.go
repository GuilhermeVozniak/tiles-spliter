package app

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/GuilhermeVozniak/tiles-spliter/desktop/internal/engine"
)

var disp = engine.Display{ID: 1, Frame: engine.Rect{X: 0, Y: 0, W: 1440, H: 900}, Visible: engine.Rect{X: 0, Y: 25, W: 1440, H: 815}}

type fakeWin struct {
	id       uint32
	frame    engine.Rect
	frameErr error
	sets     []engine.Rect
}

func (w *fakeWin) WinID() uint32 { return w.id }
func (w *fakeWin) Frame() (engine.Rect, error) {
	if w.frameErr != nil {
		return engine.Rect{}, w.frameErr
	}
	return w.frame, nil
}
func (w *fakeWin) SetFrame(r engine.Rect) error { w.frame = r; w.sets = append(w.sets, r); return nil }
func (w *fakeWin) Release()                     {}

type fakePlatform struct {
	win           *fakeWin
	overlays      int
	windowAtCalls int
}

func (p *fakePlatform) FocusedWindow() (AppWindow, error) { return p.win, nil }
func (p *fakePlatform) WindowAt(engine.Point) (AppWindow, error) {
	p.windowAtCalls++
	return p.win, nil
}
func (p *fakePlatform) Displays() []engine.Display { return []engine.Display{disp} }
func (p *fakePlatform) ShowOverlay(engine.Rect)    { p.overlays++ }
func (p *fakePlatform) HideOverlay()               {}

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
	d.OnDragEvent(1, 5, 450, false)   // >4pt movement: drag starts, into left edge zone
	if p.overlays == 0 {
		t.Fatal("overlay shown while hovering zone")
	}
	d.OnDragEvent(2, 5, 450, false) // drop
	want, _ := engine.FrameFor(engine.ActionHalfLeft, engine.Rect{}, disp, 0, false)
	if !w.frame.Eq(want, 0.01) {
		t.Fatalf("dropped window snapped, got %+v", w.frame)
	}
}

func TestClickWithoutMovementDoesNothing(t *testing.T) {
	w := &fakeWin{id: 1, frame: engine.Rect{X: 300, Y: 300, W: 500, H: 400}}
	d, p := newTestDispatcher(w)
	d.OnDragEvent(0, 320, 310, false)
	d.OnDragEvent(2, 320, 310, false)
	if p.windowAtCalls != 0 {
		t.Fatal("a click must not resolve a window via AX")
	}
	if len(w.sets) != 0 || p.overlays != 0 {
		t.Fatal("a click must not move a window or show overlays")
	}
}

func TestSmallMovementIsStillAClick(t *testing.T) {
	w := &fakeWin{id: 1, frame: engine.Rect{X: 300, Y: 300, W: 500, H: 400}}
	d, p := newTestDispatcher(w)
	d.OnDragEvent(0, 320, 310, false)
	d.OnDragEvent(1, 323, 310, false) // 3pt ≤ 4pt threshold
	d.OnDragEvent(2, 323, 310, false)
	if p.windowAtCalls != 0 || len(w.sets) != 0 {
		t.Fatal("movement within the click slop must not start a drag")
	}
}

func TestMovementBeyondThresholdStartsDragOnce(t *testing.T) {
	w := &fakeWin{id: 1, frame: engine.Rect{X: 300, Y: 300, W: 500, H: 400}}
	d, p := newTestDispatcher(w)
	d.OnDragEvent(0, 320, 310, false)
	if p.windowAtCalls != 0 {
		t.Fatal("mouse-down alone must not resolve the window")
	}
	d.OnDragEvent(1, 326, 310, false) // 6pt > threshold
	if p.windowAtCalls != 1 {
		t.Fatalf("crossing the threshold resolves the window once, got %d calls", p.windowAtCalls)
	}
	d.OnDragEvent(1, 400, 310, false)
	if p.windowAtCalls != 1 {
		t.Fatal("the window is resolved only once per drag")
	}
	d.OnDragEvent(2, 400, 310, false)
}

func TestRestoreAppliesOnDragStartNotMouseDown(t *testing.T) {
	w := &fakeWin{id: 9, frame: engine.Rect{X: 300, Y: 300, W: 500, H: 400}}
	d, _ := newTestDispatcher(w)
	// Snap the window first so a restore record exists.
	d.OnDragEvent(0, 320, 310, false)
	d.OnDragEvent(1, 5, 450, false)
	d.OnDragEvent(2, 5, 450, false)
	base := len(w.sets)
	// Mouse-down plus movement within the slop: no restore yet.
	d.OnDragEvent(0, 100, 450, false)
	d.OnDragEvent(1, 102, 450, false)
	if len(w.sets) != base {
		t.Fatal("restore must not apply before the drag threshold")
	}
	// Crossing the threshold starts the drag and applies the restore size.
	d.OnDragEvent(1, 110, 450, false)
	if len(w.sets) != base+1 {
		t.Fatalf("restore applies once the drag is genuine, sets=%d base=%d", len(w.sets), base)
	}
	last := w.sets[len(w.sets)-1]
	if last.W != 500 || last.H != 400 {
		t.Fatalf("restore uses the pre-snap size, got %+v", last)
	}
	d.OnDragEvent(2, 110, 450, false)
}

// Two Perform calls racing (macOS hotkey auto-repeat while the previous fire
// is still mid-animation) must coalesce: the second is dropped, not queued,
// so exactly one action applies and exactly one undo entry is recorded.
func TestPerformConcurrentCallsAreCoalesced(t *testing.T) {
	orig := engine.Rect{X: 100, Y: 100, W: 400, H: 300}
	w := &fakeWin{id: 1, frame: orig}
	d, _ := newTestDispatcher(w)

	// Channel-gated fake: the first caller to acquire actionMu blocks here
	// until the test releases it, giving a concurrent second call a window
	// to either overlap (bug) or bounce off TryLock (fix).
	gate := make(chan struct{})
	entered := make(chan struct{}, 2)
	d.animate = func(win AppWindow, from, to engine.Rect) {
		entered <- struct{}{}
		<-gate
		_ = win.SetFrame(to)
	}

	var wg sync.WaitGroup
	wg.Add(2)
	go func() { defer wg.Done(); d.Perform(engine.ActionFullscreen) }()
	go func() { defer wg.Done(); d.Perform(engine.ActionFullscreen) }()

	<-entered // exactly one goroutine should get past TryLock into animate
	select {
	case <-entered:
		t.Fatal("a second Perform entered animate concurrently — actions were not serialized")
	case <-time.After(75 * time.Millisecond):
		// expected: the second call bounced off TryLock and returned already
	}
	close(gate)
	wg.Wait()

	if len(w.sets) != 1 {
		t.Fatalf("want exactly one applied action, got %d", len(w.sets))
	}
	if _, ok := d.undo.Pop(w.id); !ok {
		t.Fatal("want one undo entry")
	}
	if _, ok := d.undo.Pop(w.id); ok {
		t.Fatal("want exactly one undo entry, found a second")
	}
}

func TestUndoKeepsRecordOnFrameError(t *testing.T) {
	orig := engine.Rect{X: 100, Y: 100, W: 400, H: 300}
	w := &fakeWin{id: 1, frame: orig}
	d, _ := newTestDispatcher(w)
	d.Perform(engine.ActionFullscreen)
	w.frameErr = errors.New("ax timeout")
	d.Undo() // must not consume the record
	w.frameErr = nil
	d.Undo()
	if !w.frame.Eq(orig, 0.01) {
		t.Fatalf("undo record must survive a Frame error; got %+v", w.frame)
	}
}
