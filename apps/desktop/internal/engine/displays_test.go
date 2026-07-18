package engine

import "testing"

var dispB = Display{ID: 2, Frame: Rect{1440, 0, 1920, 1080}, Visible: Rect{1440, 25, 1920, 1055}}
var two = []Display{disp, dispB}

func TestDisplayOf(t *testing.T) {
	if DisplayOf(Rect{100, 100, 400, 300}, two) != 0 {
		t.Fatal("want display 0")
	}
	if DisplayOf(Rect{2000, 100, 400, 300}, two) != 1 {
		t.Fatal("want display 1")
	}
	if DisplayOf(Rect{-5000, -5000, 10, 10}, two) != 0 {
		t.Fatal("off-screen falls back to 0")
	}
}

func TestAdjacentDisplayWraps(t *testing.T) {
	if AdjacentDisplay(1, two, true) != 0 {
		t.Fatal("next from last wraps to first")
	}
	if AdjacentDisplay(0, two, false) != 1 {
		t.Fatal("prev from first wraps to last")
	}
}

func TestMapToDisplayProportional(t *testing.T) {
	// Left half of A maps to left half of B.
	win := Rect{0, 25, 720, 815}
	eq(t, MapToDisplay(win, disp, dispB), Rect{1440, 25, 960, 1055})
}

func TestMapToDisplayKeepsRelativeOffset(t *testing.T) {
	// Centered quarter-size window stays centered.
	win := Rect{360, 25 + 203.75, 720, 407.5}
	got := MapToDisplay(win, disp, dispB)
	eq(t, got, Rect{1440 + 480, 25 + 263.75, 960, 527.5})
}
