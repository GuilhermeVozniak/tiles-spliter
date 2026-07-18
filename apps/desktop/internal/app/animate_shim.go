package app

import (
	"time"

	"github.com/GuilhermeVozniak/tiles-spliter/desktop/internal/engine"
	"github.com/GuilhermeVozniak/tiles-spliter/desktop/internal/platform"
)

func animateFrame(w AppWindow, from, to engine.Rect, enabled bool) {
	platform.AnimateFrame(w, from, to, enabled, func(ms int) { time.Sleep(time.Duration(ms) * time.Millisecond) })
}
