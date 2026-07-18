// Package engine contains all window-management logic as pure Go.
// Coordinates are top-left-origin global points (y grows downward), matching the AX API.
package engine

import "math"

type Point struct{ X, Y float64 }

type Rect struct{ X, Y, W, H float64 }

func (r Rect) Contains(p Point) bool {
	return p.X >= r.X && p.X < r.X+r.W && p.Y >= r.Y && p.Y < r.Y+r.H
}

func (r Rect) Mid() Point { return Point{r.X + r.W/2, r.Y + r.H/2} }

func (r Rect) Eq(o Rect, eps float64) bool {
	return math.Abs(r.X-o.X) < eps && math.Abs(r.Y-o.Y) < eps &&
		math.Abs(r.W-o.W) < eps && math.Abs(r.H-o.H) < eps
}

// Display describes one screen. Visible excludes the menu bar and Dock.
type Display struct {
	ID             uint32
	Frame, Visible Rect
}
