package engine

import "testing"

// 1440x900 display, 25pt menu bar, 60pt Dock at bottom.
var disp = Display{ID: 1, Frame: Rect{0, 0, 1440, 900}, Visible: Rect{0, 25, 1440, 815}}

func eq(t *testing.T, got, want Rect) {
	t.Helper()
	if !got.Eq(want, 0.01) {
		t.Fatalf("got %+v want %+v", got, want)
	}
}

func must(t *testing.T, r Rect, ok bool) Rect {
	t.Helper()
	if !ok {
		t.Fatal("expected handled action")
	}
	return r
}

func TestHalvesNoPadding(t *testing.T) {
	w := Rect{100, 100, 400, 300}
	r, ok := FrameFor(ActionHalfLeft, w, disp, 0, false)
	eq(t, must(t, r, ok), Rect{0, 25, 720, 815})
	r, ok = FrameFor(ActionHalfRight, w, disp, 0, false)
	eq(t, must(t, r, ok), Rect{720, 25, 720, 815})
	r, ok = FrameFor(ActionHalfTop, w, disp, 0, false)
	eq(t, must(t, r, ok), Rect{0, 25, 1440, 407.5})
	r, ok = FrameFor(ActionHalfBottom, w, disp, 0, false)
	eq(t, must(t, r, ok), Rect{0, 432.5, 1440, 407.5})
}

func TestQuarters(t *testing.T) {
	w := Rect{0, 0, 10, 10}
	r, ok := FrameFor(ActionUpperLeft, w, disp, 0, false)
	eq(t, must(t, r, ok), Rect{0, 25, 720, 407.5})
	r, ok = FrameFor(ActionUpperRight, w, disp, 0, false)
	eq(t, must(t, r, ok), Rect{720, 25, 720, 407.5})
	r, ok = FrameFor(ActionLowerLeft, w, disp, 0, false)
	eq(t, must(t, r, ok), Rect{0, 432.5, 720, 407.5})
	r, ok = FrameFor(ActionLowerRight, w, disp, 0, false)
	eq(t, must(t, r, ok), Rect{720, 432.5, 720, 407.5})
}

func TestPaddingOuterFullInnerHalf(t *testing.T) {
	// pad=10: outer edges inset 10, shared inner edge each side inset 5.
	r, ok := FrameFor(ActionHalfLeft, Rect{}, disp, 10, false)
	got := must(t, r, ok)
	eq(t, got, Rect{10, 35, 720 - 10 - 5, 815 - 20})
}

func TestFullscreenPadding(t *testing.T) {
	r, ok := FrameFor(ActionFullscreen, Rect{}, disp, 10, false)
	eq(t, must(t, r, ok), disp.Visible)
	r, ok = FrameFor(ActionFullscreen, Rect{}, disp, 10, true)
	eq(t, must(t, r, ok), Rect{10, 35, 1420, 795})
}

func TestCenterKeepsSizeAndClamps(t *testing.T) {
	r, ok := FrameFor(ActionCenter, Rect{0, 0, 400, 300}, disp, 0, false)
	eq(t, must(t, r, ok), Rect{520, 282.5, 400, 300})
	// Oversized window is clamped to the visible frame.
	r, ok = FrameFor(ActionCenter, Rect{0, 0, 2000, 1000}, disp, 0, false)
	eq(t, must(t, r, ok), disp.Visible)
}

func TestTwoThirds(t *testing.T) {
	r, ok := FrameFor(ActionTwoThirdsLeft, Rect{}, disp, 0, false)
	eq(t, must(t, r, ok), Rect{0, 25, 960, 815})
	r, ok = FrameFor(ActionTwoThirdsRight, Rect{}, disp, 0, false)
	eq(t, must(t, r, ok), Rect{480, 25, 960, 815})
	r, ok = FrameFor(ActionTwoThirdsCenter, Rect{}, disp, 0, false)
	eq(t, must(t, r, ok), Rect{240, 25, 960, 815})
}

func TestUnhandledActionsReturnFalse(t *testing.T) {
	if _, ok := FrameFor(ActionNextThird, Rect{}, disp, 0, false); ok {
		t.Fatal("thirds are handled elsewhere")
	}
	if _, ok := FrameFor(ActionNextDisplay, Rect{}, disp, 0, false); ok {
		t.Fatal("display moves are handled elsewhere")
	}
}
