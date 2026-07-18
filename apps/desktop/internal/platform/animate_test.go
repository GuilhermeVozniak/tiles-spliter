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

type fastRecorder struct {
	recorder
	fastFrames []engine.Rect
}

func (r *fastRecorder) SetFrameFast(f engine.Rect) error {
	r.fastFrames = append(r.fastFrames, f)
	return nil
}

// A setter offering the fast path gets it for every intermediate step, while
// the final step still uses the robust SetFrame and lands exactly on target.
func TestAnimateUsesFastPathForIntermediateSteps(t *testing.T) {
	r := &fastRecorder{}
	to := engine.Rect{X: 100, Y: 200, W: 300, H: 400}
	AnimateFrame(r, engine.Rect{}, to, true, func(int) {})
	if len(r.fastFrames) != animSteps-1 {
		t.Fatalf("want %d fast steps, got %d", animSteps-1, len(r.fastFrames))
	}
	if len(r.frames) != 1 || !r.frames[0].Eq(to, 0.001) {
		t.Fatalf("final step must be a robust SetFrame at target, got %+v", r.frames)
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
