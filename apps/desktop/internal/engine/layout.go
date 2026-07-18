package engine

type Action string

const (
	ActionCenter          Action = "center"
	ActionFullscreen      Action = "fullscreen"
	ActionHalfLeft        Action = "half-left"
	ActionHalfRight       Action = "half-right"
	ActionHalfTop         Action = "half-top"
	ActionHalfBottom      Action = "half-bottom"
	ActionUpperLeft       Action = "upper-left"
	ActionUpperRight      Action = "upper-right"
	ActionLowerLeft       Action = "lower-left"
	ActionLowerRight      Action = "lower-right"
	ActionNextThird       Action = "next-third"
	ActionPrevThird       Action = "prev-third"
	ActionTwoThirdsLeft   Action = "two-thirds-left"
	ActionTwoThirdsRight  Action = "two-thirds-right"
	ActionTwoThirdsCenter Action = "two-thirds-center"
	ActionNextDisplay     Action = "next-display"
	ActionPrevDisplay     Action = "prev-display"
	ActionNone            Action = "none"
)

const frac = 1e-9

// cell returns the sub-rect of v at fractional origin (fx,fy) with fractional
// size (fw,fh), padded: outer edges (touching v's edge) inset by pad, inner
// shared edges inset by pad/2 so adjacent cells end up pad apart.
func cell(v Rect, fx, fy, fw, fh, pad float64) Rect {
	r := Rect{v.X + fx*v.W, v.Y + fy*v.H, fw * v.W, fh * v.H}
	left, top, right, bottom := pad/2, pad/2, pad/2, pad/2
	if fx <= frac {
		left = pad
	}
	if fy <= frac {
		top = pad
	}
	if fx+fw >= 1-frac {
		right = pad
	}
	if fy+fh >= 1-frac {
		bottom = pad
	}
	return Rect{r.X + left, r.Y + top, r.W - left - right, r.H - top - bottom}
}

// FrameFor computes the target frame for a stateless action. Thirds cycling
// (needs current position) and display moves (need the display list) are
// handled by ThirdFrame / MapToDisplay; for those it returns ok=false.
func FrameFor(a Action, win Rect, d Display, pad float64, padFullscreen bool) (Rect, bool) {
	v := d.Visible
	switch a {
	case ActionHalfLeft:
		return cell(v, 0, 0, 0.5, 1, pad), true
	case ActionHalfRight:
		return cell(v, 0.5, 0, 0.5, 1, pad), true
	case ActionHalfTop:
		return cell(v, 0, 0, 1, 0.5, pad), true
	case ActionHalfBottom:
		return cell(v, 0, 0.5, 1, 0.5, pad), true
	case ActionUpperLeft:
		return cell(v, 0, 0, 0.5, 0.5, pad), true
	case ActionUpperRight:
		return cell(v, 0.5, 0, 0.5, 0.5, pad), true
	case ActionLowerLeft:
		return cell(v, 0, 0.5, 0.5, 0.5, pad), true
	case ActionLowerRight:
		return cell(v, 0.5, 0.5, 0.5, 0.5, pad), true
	case ActionTwoThirdsLeft:
		return cell(v, 0, 0, 2.0/3, 1, pad), true
	case ActionTwoThirdsRight:
		return cell(v, 1.0/3, 0, 2.0/3, 1, pad), true
	case ActionTwoThirdsCenter:
		return cell(v, 1.0/6, 0, 2.0/3, 1, pad), true
	case ActionFullscreen:
		if padFullscreen {
			return cell(v, 0, 0, 1, 1, pad), true
		}
		return v, true
	case ActionCenter:
		w, h := min(win.W, v.W), min(win.H, v.H)
		return Rect{v.X + (v.W-w)/2, v.Y + (v.H-h)/2, w, h}, true
	}
	return Rect{}, false
}
