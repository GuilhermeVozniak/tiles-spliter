package engine

import "testing"

func third(i int) Rect { return ThirdColumn(i, disp, 0) }

func TestThirdColumns(t *testing.T) {
	eq(t, third(0), Rect{0, 25, 480, 815})
	eq(t, third(1), Rect{480, 25, 480, 815})
	eq(t, third(2), Rect{960, 25, 480, 815})
}

func TestNextThirdFromUnpositionedStartsLeft(t *testing.T) {
	r, ok := ThirdFrame(ActionNextThird, Rect{100, 100, 500, 400}, disp, 0)
	got := must(t, r, ok)
	eq(t, got, third(0))
}

func TestNextThirdCyclesAndWraps(t *testing.T) {
	r1, ok1 := ThirdFrame(ActionNextThird, third(0), disp, 0)
	eq(t, must(t, r1, ok1), third(1))
	r2, ok2 := ThirdFrame(ActionNextThird, third(1), disp, 0)
	eq(t, must(t, r2, ok2), third(2))
	r3, ok3 := ThirdFrame(ActionNextThird, third(2), disp, 0)
	eq(t, must(t, r3, ok3), third(0))
}

func TestPrevThirdCyclesAndWraps(t *testing.T) {
	r0, ok0 := ThirdFrame(ActionPrevThird, Rect{100, 100, 500, 400}, disp, 0)
	eq(t, must(t, r0, ok0), third(2))
	r1, ok1 := ThirdFrame(ActionPrevThird, third(0), disp, 0)
	eq(t, must(t, r1, ok1), third(2))
	r2, ok2 := ThirdFrame(ActionPrevThird, third(2), disp, 0)
	eq(t, must(t, r2, ok2), third(1))
}

func TestThirdFrameIgnoresOtherActions(t *testing.T) {
	if _, ok := ThirdFrame(ActionCenter, Rect{}, disp, 0); ok {
		t.Fatal("only thirds actions")
	}
}
