package engine

// ThirdColumn returns column i (0..2) of the display's visible frame.
func ThirdColumn(i int, d Display, pad float64) Rect {
	return cell(d.Visible, float64(i)/3, 0, 1.0/3, 1, pad)
}

// currentThird reports which third the window currently occupies, or -1.
// A window "occupies" a third when its frame matches within a tolerance —
// tolerant of padding differences by comparing against every padding=pad column.
func currentThird(win Rect, d Display, pad float64) int {
	for i := 0; i < 3; i++ {
		if win.Eq(ThirdColumn(i, d, pad), 2.0) {
			return i
		}
	}
	return -1
}

// ThirdFrame handles ActionNextThird / ActionPrevThird: cycle the window
// left → middle → right (wrapping); a window not currently on a third starts
// at the left (Next) or right (Prev) column, matching Tiles.
func ThirdFrame(a Action, win Rect, d Display, pad float64) (Rect, bool) {
	cur := currentThird(win, d, pad)
	switch a {
	case ActionNextThird:
		return ThirdColumn((cur+1)%3, d, pad), true // cur==-1 → 0
	case ActionPrevThird:
		if cur <= 0 {
			return ThirdColumn(2, d, pad), true
		}
		return ThirdColumn(cur-1, d, pad), true
	}
	return Rect{}, false
}
