package platform

import "github.com/GuilhermeVozniak/tiles-spliter/desktop/internal/engine"

// FrameSetter abstracts *Window for testing.
type FrameSetter interface{ SetFrame(engine.Rect) error }

// FastFrameSetter is an optional fast path: a single size+position set (2 AX
// calls) instead of the robust triple set. AnimateFrame type-asserts for it
// and uses it for intermediate steps only — the final step always goes
// through SetFrame so end placement keeps the min-size-constraint handling.
type FastFrameSetter interface{ SetFrameFast(engine.Rect) error }

const animSteps = 8

// AnimateFrame moves a window from → to. With animation enabled it eases out
// over animSteps steps (~150ms via sleep(18)); otherwise one direct set.
func AnimateFrame(w FrameSetter, from, to engine.Rect, enabled bool, sleep func(ms int)) {
	if !enabled {
		_ = w.SetFrame(to)
		return
	}
	fast, hasFast := w.(FastFrameSetter)
	for i := 1; i <= animSteps; i++ {
		t := float64(i) / animSteps
		t = 1 - (1-t)*(1-t) // ease-out
		r := engine.Rect{
			X: from.X + (to.X-from.X)*t,
			Y: from.Y + (to.Y-from.Y)*t,
			W: from.W + (to.W-from.W)*t,
			H: from.H + (to.H-from.H)*t,
		}
		if hasFast && i < animSteps {
			_ = fast.SetFrameFast(r)
		} else {
			_ = w.SetFrame(r) // final step keeps the robust 3-set path
		}
		if i < animSteps {
			sleep(18)
		}
	}
}
