package engine

import "testing"

type fakePlat struct {
	now      int64
	shown    []Rect
	hidden   int
}

func (f *fakePlat) ShowOverlay(r Rect) { f.shown = append(f.shown, r) }
func (f *fakePlat) HideOverlay()       { f.hidden++ }
func (f *fakePlat) Now() int64         { return f.now }

func newCtl(f *fakePlat, mut func(*Settings)) *SnapController {
	s := DefaultSettings()
	if mut != nil {
		mut(&s)
	}
	return NewSnapController(f, func() Settings { return s }, func() []Display { return []Display{disp} })
}

func TestSnapLeftEdgeShowsOverlayAndApplies(t *testing.T) {
	f := &fakePlat{}
	c := newCtl(f, nil)
	c.DragStart(DragWindow{ID: 7, Frame: Rect{100, 100, 400, 300}})
	c.DragMove(Point{5, 450}, false)
	if len(f.shown) != 1 {
		t.Fatal("overlay shown on zone enter")
	}
	half, _ := FrameFor(ActionHalfLeft, Rect{}, disp, 0, false)
	eq(t, f.shown[0], half)
	win, frame, apply := c.DragEnd(Point{5, 450}, false)
	if !apply || win != 7 {
		t.Fatal("apply on drop in zone")
	}
	eq(t, frame, half)
	if f.hidden == 0 {
		t.Fatal("overlay hidden after drop")
	}
}

func TestActivationDelayArmsLate(t *testing.T) {
	f := &fakePlat{now: 1000}
	c := newCtl(f, func(s *Settings) { s.Snap.ActivationDelayMs = 200 })
	c.DragStart(DragWindow{ID: 1, Frame: Rect{}})
	c.DragMove(Point{5, 450}, false)
	if len(f.shown) != 0 {
		t.Fatal("not armed before delay")
	}
	if _, _, apply := c.DragEnd(Point{5, 450}, false); apply {
		t.Fatal("drop before delay does nothing")
	}
	c.DragStart(DragWindow{ID: 1, Frame: Rect{}})
	c.DragMove(Point{5, 450}, false)
	f.now = 1300
	c.DragMove(Point{6, 450}, false)
	if len(f.shown) == 0 {
		t.Fatal("armed after delay elapsed")
	}
}

func TestLeavingZoneHidesOverlay(t *testing.T) {
	f := &fakePlat{}
	c := newCtl(f, nil)
	c.DragStart(DragWindow{ID: 1, Frame: Rect{}})
	c.DragMove(Point{5, 450}, false)
	c.DragMove(Point{700, 450}, false)
	if f.hidden == 0 {
		t.Fatal("overlay hides when leaving zone")
	}
	if _, _, apply := c.DragEnd(Point{700, 450}, false); apply {
		t.Fatal("no apply outside zone")
	}
}

func TestModifierThirdsOnSideEdge(t *testing.T) {
	f := &fakePlat{}
	c := newCtl(f, nil)
	c.DragStart(DragWindow{ID: 1, Frame: Rect{}})
	c.DragMove(Point{5, 450}, true)
	_, frame, apply := c.DragEnd(Point{5, 450}, true)
	if !apply {
		t.Fatal("apply")
	}
	eq(t, frame, ThirdColumn(0, disp, 0))
}

func TestDisabledZoneDoesNothing(t *testing.T) {
	f := &fakePlat{}
	c := newCtl(f, nil)
	c.DragStart(DragWindow{ID: 1, Frame: Rect{}})
	c.DragMove(Point{720, 897}, false) // bottom zone: default "none"
	if len(f.shown) != 0 {
		t.Fatal("disabled zone shows no overlay")
	}
}

func TestRestorePreviousSize(t *testing.T) {
	f := &fakePlat{}
	c := newCtl(f, nil)
	orig := Rect{100, 100, 400, 300}
	c.DragStart(DragWindow{ID: 7, Frame: orig})
	c.DragMove(Point{5, 450}, false)
	_, snapped, _ := c.DragEnd(Point{5, 450}, false)
	// Next drag of the same window while still snapped → restore original size.
	restore, ok := c.DragStart(DragWindow{ID: 7, Frame: snapped})
	if !ok || restore.W != orig.W || restore.H != orig.H {
		t.Fatalf("want restore to original size, got %+v %v", restore, ok)
	}
	// A window moved since snapping does not restore.
	if _, ok := c.DragStart(DragWindow{ID: 7, Frame: orig}); ok {
		t.Fatal("frame changed since snap → no restore")
	}
}

func TestSnapDisabledGloballyIgnoresEverything(t *testing.T) {
	f := &fakePlat{}
	c := newCtl(f, func(s *Settings) { s.Snap.Enabled = false })
	c.DragStart(DragWindow{ID: 1, Frame: Rect{}})
	c.DragMove(Point{5, 450}, false)
	if len(f.shown) != 0 {
		t.Fatal("disabled snap shows nothing")
	}
}
