package platform

import (
	"testing"

	"github.com/GuilhermeVozniak/tiles-spliter/desktop/internal/engine"
)

type recorder struct{ frames []engine.Rect }

func (r *recorder) SetFrame(f engine.Rect) error { r.frames = append(r.frames, f); return nil }

func TestAnimateDisabledSetsOnce(t *testing.T) {
	r := &recorder{}
	AnimateFrame(r, engine.Rect{}, engine.Rect{X: 100, W: 50, H: 50}, false, func(int) {})
	if len(r.frames) != 1 || r.frames[0].X != 100 {
		t.Fatalf("want single direct set, got %+v", r.frames)
	}
}

func TestAnimateEnabledEndsExactlyAtTarget(t *testing.T) {
	r := &recorder{}
	to := engine.Rect{X: 100, Y: 200, W: 300, H: 400}
	AnimateFrame(r, engine.Rect{}, to, true, func(int) {})
	if len(r.frames) != animSteps {
		t.Fatalf("want %d steps, got %d", animSteps, len(r.frames))
	}
	if !r.frames[len(r.frames)-1].Eq(to, 0.001) {
		t.Fatalf("final frame must equal target, got %+v", r.frames[len(r.frames)-1])
	}
	if r.frames[0].X <= 0 {
		t.Fatal("first step must move toward target")
	}
}
