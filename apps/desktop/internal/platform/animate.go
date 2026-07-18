package platform

import "github.com/GuilhermeVozniak/tiles-spliter/desktop/internal/engine"

// FrameSetter abstracts *Window for testing.
type FrameSetter interface{ SetFrame(engine.Rect) error }

const animSteps = 8

// AnimateFrame moves a window from → to. With animation enabled it eases out
// over animSteps steps (~150ms via sleep(18)); otherwise one direct set.
func AnimateFrame(w FrameSetter, from, to engine.Rect, enabled bool, sleep func(ms int)) {
	if !enabled {
		_ = w.SetFrame(to)
		return
	}
	for i := 1; i <= animSteps; i++ {
		t := float64(i) / animSteps
		t = 1 - (1-t)*(1-t) // ease-out
		_ = w.SetFrame(engine.Rect{
			X: from.X + (to.X-from.X)*t,
			Y: from.Y + (to.Y-from.Y)*t,
			W: from.W + (to.W-from.W)*t,
			H: from.H + (to.H-from.H)*t,
		})
		if i < animSteps {
			sleep(18)
		}
	}
}
