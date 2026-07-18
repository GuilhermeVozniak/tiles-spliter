package engine

import "testing"

func zone(t *testing.T, x, y float64) Zone {
	t.Helper()
	z, ok := ZoneAt(Point{x, y}, disp, 10)
	if !ok {
		t.Fatalf("expected a zone at %v,%v", x, y)
	}
	return z
}

func TestZoneRows(t *testing.T) {
	// Top edge row (y within 10 of top): 25/50/25 split of width 1440 → 360/720/360.
	if zone(t, 100, 5) != ZoneTopLeft || zone(t, 720, 5) != ZoneTop || zone(t, 1400, 5) != ZoneTopRight {
		t.Fatal("top row")
	}
	if zone(t, 100, 897) != ZoneBottomLeft || zone(t, 720, 897) != ZoneBottom || zone(t, 1400, 897) != ZoneBottomRight {
		t.Fatal("bottom row")
	}
	// Side edges: 25/50/25 split of height 900 → 225/450/225.
	if zone(t, 5, 100) != ZoneTopLeft || zone(t, 5, 450) != ZoneLeft || zone(t, 5, 800) != ZoneBottomLeft {
		t.Fatal("left edge")
	}
	if zone(t, 1435, 450) != ZoneRight {
		t.Fatal("right edge")
	}
}

func TestZoneMissesInterior(t *testing.T) {
	if _, ok := ZoneAt(Point{720, 450}, disp, 10); ok {
		t.Fatal("screen middle is no zone")
	}
	if _, ok := ZoneAt(Point{720, 12}, disp, 10); ok {
		t.Fatal("just past thickness is no zone")
	}
}

func TestSnapActionMapping(t *testing.T) {
	s := DefaultSettings().Snap
	if SnapAction(ZoneLeft, s, false) != ActionHalfLeft {
		t.Fatal("left → half-left")
	}
	if SnapAction(ZoneBottom, s, false) != ActionNone {
		t.Fatal("disabled zone → none")
	}
	if SnapAction(ZoneLeft, s, true) != ActionSnapThirdLeft || SnapAction(ZoneRight, s, true) != ActionSnapThirdRight {
		t.Fatal("modifier → thirds on side edges")
	}
	if SnapAction(ZoneTopLeft, s, true) != ActionUpperLeft {
		t.Fatal("modifier does not affect corners")
	}
	s.Zones[ZoneLeft] = ActionNone
	if SnapAction(ZoneLeft, s, true) != ActionNone {
		t.Fatal("modifier must not override an explicitly disabled zone")
	}
	if SnapAction(ZoneRight, s, true) != ActionSnapThirdRight {
		t.Fatal("other enabled edges keep modifier thirds")
	}
}
