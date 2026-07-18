package engine

import "sync"

// SnapPlatform is what the controller needs from the native layer.
type SnapPlatform interface {
	ShowOverlay(frame Rect)
	HideOverlay()
	Now() int64 // milliseconds
}

type DragWindow struct {
	ID    uint32
	Frame Rect
}

type snapRecord struct{ snapped, original Rect }

// SnapController is a pure state machine driving snap-to-edges. The platform
// layer feeds it DragStart/DragMove/DragEnd from the event tap; it decides
// when to show the overlay and what frame to apply. No goroutines, no timers.
type SnapController struct {
	mu          sync.Mutex
	plat        SnapPlatform
	getSettings func() Settings
	displays    func() []Display

	dragging   bool
	win        DragWindow
	zone       Zone
	zoneSince  int64
	armed      bool
	armedFrame Rect
	history    map[uint32]snapRecord
}

func NewSnapController(p SnapPlatform, getSettings func() Settings, displays func() []Display) *SnapController {
	return &SnapController{plat: p, getSettings: getSettings, displays: displays, history: map[uint32]snapRecord{}}
}

func (c *SnapController) DragStart(w DragWindow) (Rect, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	s := c.getSettings()
	if !s.Snap.Enabled {
		return Rect{}, false
	}
	c.dragging, c.win, c.zone, c.armed = true, w, "", false
	if rec, ok := c.history[w.ID]; ok {
		delete(c.history, w.ID)
		if s.Snap.RestorePreviousSize && w.Frame.Eq(rec.snapped, 2.0) {
			return Rect{W: rec.original.W, H: rec.original.H}, true
		}
	}
	return Rect{}, false
}

// resolve computes the target frame for the zone under the cursor, or ok=false.
func (c *SnapController) resolve(cursor Point, modThirds bool) (Rect, bool) {
	s := c.getSettings()
	for _, d := range c.displays() {
		zone, ok := ZoneAt(cursor, d, s.Snap.ZoneThickness)
		if !ok {
			continue
		}
		if zone != c.zone {
			c.zone, c.zoneSince = zone, c.plat.Now()
		}
		if c.plat.Now()-c.zoneSince < int64(s.Snap.ActivationDelayMs) {
			return Rect{}, false
		}
		switch a := SnapAction(zone, s.Snap, modThirds); a {
		case ActionNone:
			return Rect{}, false
		case ActionSnapThirdLeft:
			return ThirdColumn(0, d, s.General.WindowPadding), true
		case ActionSnapThirdRight:
			return ThirdColumn(2, d, s.General.WindowPadding), true
		default:
			if f, ok := FrameFor(a, c.win.Frame, d, s.General.WindowPadding, s.General.PadFullscreen); ok {
				return f, true
			}
			return Rect{}, false
		}
	}
	c.zone = ""
	return Rect{}, false
}

func (c *SnapController) DragMove(cursor Point, modThirds bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.dragging {
		return
	}
	frame, ok := c.resolve(cursor, modThirds)
	switch {
	case ok && (!c.armed || !frame.Eq(c.armedFrame, 0.5)):
		c.armed, c.armedFrame = true, frame
		c.plat.ShowOverlay(frame)
	case !ok && c.armed:
		c.armed = false
		c.plat.HideOverlay()
	}
}

func (c *SnapController) DragEnd(cursor Point, modThirds bool) (uint32, Rect, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.dragging {
		return 0, Rect{}, false
	}
	c.dragging = false
	if c.armed {
		c.armed = false
		c.plat.HideOverlay()
	}
	frame, ok := c.resolve(cursor, modThirds)
	if !ok {
		return 0, Rect{}, false
	}
	c.history[c.win.ID] = snapRecord{snapped: frame, original: c.win.Frame}
	return c.win.ID, frame, true
}
