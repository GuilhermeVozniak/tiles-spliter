package engine

import "sort"

// DisplayOf returns the index of the display whose full frame contains the
// window's midpoint, falling back to index 0 as a last resort.
func DisplayOf(win Rect, displays []Display) int {
	mid := win.Mid()
	for i, d := range displays {
		if d.Frame.Contains(mid) {
			return i
		}
	}
	return 0
}

// AdjacentDisplay returns the next/previous display index in left-to-right
// order of Frame.X, wrapping around.
func AdjacentDisplay(cur int, displays []Display, next bool) int {
	if len(displays) < 2 {
		return cur
	}
	order := make([]int, len(displays))
	for i := range order {
		order[i] = i
	}
	sort.Slice(order, func(a, b int) bool {
		return displays[order[a]].Frame.X < displays[order[b]].Frame.X
	})
	pos := 0
	for i, idx := range order {
		if idx == cur {
			pos = i
			break
		}
	}
	if next {
		return order[(pos+1)%len(order)]
	}
	return order[(pos-1+len(order))%len(order)]
}

// MapToDisplay proportionally re-maps a window from one display's visible
// frame into another's, preserving relative position and relative size.
func MapToDisplay(win Rect, from, to Display) Rect {
	f, t := from.Visible, to.Visible
	sx, sy := t.W/f.W, t.H/f.H
	return Rect{
		X: t.X + (win.X-f.X)*sx,
		Y: t.Y + (win.Y-f.Y)*sy,
		W: win.W * sx,
		H: win.H * sy,
	}
}
