package engine

// Special snap-only pseudo-actions produced when ⌥/⌘ is held while dragging
// to a plain side edge; the snap controller resolves them to third columns.
const (
	ActionSnapThirdLeft  Action = "snap-third-left"
	ActionSnapThirdRight Action = "snap-third-right"
)

// ZoneAt reports which active-edge zone contains p on display d. Corner zones
// take the outer 25% of each edge, the plain edge zone the middle 50%.
func ZoneAt(p Point, d Display, thickness float64) (Zone, bool) {
	f := d.Frame
	if !f.Contains(p) {
		return "", false
	}
	hx := (p.X - f.X) / f.W
	hy := (p.Y - f.Y) / f.H
	switch {
	case p.Y <= f.Y+thickness: // top row
		switch {
		case hx < 0.25:
			return ZoneTopLeft, true
		case hx > 0.75:
			return ZoneTopRight, true
		default:
			return ZoneTop, true
		}
	case p.Y >= f.Y+f.H-thickness: // bottom row
		switch {
		case hx < 0.25:
			return ZoneBottomLeft, true
		case hx > 0.75:
			return ZoneBottomRight, true
		default:
			return ZoneBottom, true
		}
	case p.X <= f.X+thickness: // left edge
		switch {
		case hy < 0.25:
			return ZoneTopLeft, true
		case hy > 0.75:
			return ZoneBottomLeft, true
		default:
			return ZoneLeft, true
		}
	case p.X >= f.X+f.W-thickness: // right edge
		switch {
		case hy < 0.25:
			return ZoneTopRight, true
		case hy > 0.75:
			return ZoneBottomRight, true
		default:
			return ZoneRight, true
		}
	}
	return "", false
}

// SnapAction maps a zone to its configured action. With ⌥/⌘ held (modThirds),
// plain left/right edges snap to thirds instead of their configured action.
func SnapAction(zone Zone, s SnapSettings, modThirds bool) Action {
	if modThirds {
		switch zone {
		case ZoneLeft:
			return ActionSnapThirdLeft
		case ZoneRight:
			return ActionSnapThirdRight
		}
	}
	a, ok := s.Zones[zone]
	if !ok {
		return ActionNone
	}
	return a
}
