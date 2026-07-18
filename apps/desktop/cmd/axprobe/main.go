// axprobe is a manual smoke-test harness for the cgo platform layer.
// Usage: go run ./cmd/axprobe [displays|focused|snapleft]
package main

import (
	"fmt"
	"os"

	"github.com/GuilhermeVozniak/tiles-spliter/desktop/internal/engine"
	"github.com/GuilhermeVozniak/tiles-spliter/desktop/internal/platform"
)

func main() {
	cmd := "displays"
	if len(os.Args) > 1 {
		cmd = os.Args[1]
	}
	fmt.Println("AX trusted:", platform.AXTrusted(true))
	switch cmd {
	case "displays":
		for i, d := range platform.Displays() {
			fmt.Printf("display %d: id=%d frame=%+v visible=%+v\n", i, d.ID, d.Frame, d.Visible)
		}
	case "focused":
		w, err := platform.FocusedWindow()
		if err != nil {
			fmt.Println("error:", err)
			return
		}
		defer w.Release()
		f, _ := w.Frame()
		fmt.Printf("focused window id=%d frame=%+v\n", w.ID, f)
	case "snapleft":
		w, err := platform.FocusedWindow()
		if err != nil {
			fmt.Println("error:", err)
			return
		}
		defer w.Release()
		f, _ := w.Frame()
		d := platform.Displays()[engine.DisplayOf(f, platform.Displays())]
		target, _ := engine.FrameFor(engine.ActionHalfLeft, f, d, 0, false)
		fmt.Println("setting frame:", target, "err:", w.SetFrame(target))
	}
}
