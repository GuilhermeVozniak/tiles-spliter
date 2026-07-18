// Command gendmgbg renders the drag-to-Applications DMG installer background:
// a dark zinc-950 canvas (matching the app's own dark UI, see AboutTab.tsx)
// with a subtle mid-gray arrow pointing from the app icon slot to the
// Applications slot. Pure Go stdlib, no fonts — same approach as
// cmd/genmenuicons (supersampled coverage antialiasing, deterministic
// output committed to the repo).
//
// Writes bg@1x.png and bg@2x.png to build/dmg/ (or the directory given as
// the first argument). The retina-aware bg.tiff consumed by appdmg.json is
// then produced from those two PNGs with:
//
//	tiffutil -cathidpicheck build/dmg/bg@1x.png build/dmg/bg@2x.png -out build/dmg/bg.tiff
//
// (tiffutil has no Go equivalent worth vendoring for two files; this is a
// manual/CI step, same as drag-zone's bg.html → bg.tiff note.)
package main

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
)

const (
	// Window size in points (1x), matching apps/desktop/build/dmg/appdmg.json's
	// window.size and the house pattern shared with app-cleaner/drag-zone.
	winW, winH = 660.0, 400.0

	ss = 4 // supersampling factor per axis for the arrow's antialiasing

	// Icon slot centers, matching appdmg.json's contents[].{x,y}: the app on
	// the left, the Applications symlink on the right, both at icon-size 128.
	appIconX, applicationsIconX, iconY = 165.0, 495.0, 195.0

	// Arrow geometry (1x points), centered between the two icon slots.
	arrowCX, arrowCY   = (appIconX + applicationsIconX) / 2, iconY
	arrowHalfW         = 70.0
	arrowHeadLen       = 24.0
	arrowThickness     = 5.0
	arrowHeadHalfWidth = 14.0

	// Arrow opacity: fully-covered pixels blend to this fraction of the way
	// from background to arrowColor, keeping the glyph subtle on the dark bg.
	arrowOpacity = 0.62
)

// bgTop/bgBottom form a subtle light vertical gradient, matching the classic
// bright Finder installer look (and drag-zone's light DMG background).
var (
	bgTop     = color.NRGBA{R: 249, G: 249, B: 251, A: 255} // ~#f9f9fb, near-white
	bgBottom  = color.NRGBA{R: 231, G: 231, B: 237, A: 255} // ~#e7e7ed, light gray
	arrowGray = color.NRGBA{R: 110, G: 110, B: 115, A: 255} // ~#6e6e73, reads clearly on light bg
)

type pt struct{ x, y float64 }

func triContains(a, b, c, p pt) bool {
	d1 := (p.x-b.x)*(a.y-b.y) - (a.x-b.x)*(p.y-b.y)
	d2 := (p.x-c.x)*(b.y-c.y) - (b.x-c.x)*(p.y-c.y)
	d3 := (p.x-a.x)*(c.y-a.y) - (c.x-a.x)*(p.y-a.y)
	hasNeg := d1 < 0 || d2 < 0 || d3 < 0
	hasPos := d1 > 0 || d2 > 0 || d3 > 0
	return !(hasNeg && hasPos)
}

// arrowCoverage reports whether the 1x point (x,y) lies on the arrow glyph:
// a horizontal shaft rectangle plus a triangular head.
func arrowCoverage(x, y float64) bool {
	shaftX0 := arrowCX - arrowHalfW
	shaftX1 := arrowCX + arrowHalfW - arrowHeadLen
	if x >= shaftX0 && x <= shaftX1 && y >= arrowCY-arrowThickness/2 && y <= arrowCY+arrowThickness/2 {
		return true
	}
	tipX := arrowCX + arrowHalfW
	a := pt{x: shaftX1, y: arrowCY - arrowHeadHalfWidth}
	b := pt{x: tipX, y: arrowCY}
	c := pt{x: shaftX1, y: arrowCY + arrowHeadHalfWidth}
	return triContains(a, b, c, pt{x: x, y: y})
}

func lerp8(a, b uint8, t float64) uint8 {
	return uint8(float64(a) + (float64(b)-float64(a))*t)
}

// render draws the background at the given integer scale (1 or 2).
func render(scale int) *image.NRGBA {
	w, h := int(winW)*scale, int(winH)*scale
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for py := 0; py < h; py++ {
		t := float64(py) / float64(h-1)
		base := color.NRGBA{
			R: lerp8(bgTop.R, bgBottom.R, t),
			G: lerp8(bgTop.G, bgBottom.G, t),
			B: lerp8(bgTop.B, bgBottom.B, t),
			A: 255,
		}
		for px := 0; px < w; px++ {
			covered := 0
			for sy := 0; sy < ss; sy++ {
				for sx := 0; sx < ss; sx++ {
					x1 := (float64(px) + (float64(sx)+0.5)/ss) / float64(scale)
					y1 := (float64(py) + (float64(sy)+0.5)/ss) / float64(scale)
					if arrowCoverage(x1, y1) {
						covered++
					}
				}
			}
			c := base
			if covered > 0 {
				a := arrowOpacity * float64(covered) / float64(ss*ss)
				c.R = lerp8(base.R, arrowGray.R, a)
				c.G = lerp8(base.G, arrowGray.G, a)
				c.B = lerp8(base.B, arrowGray.B, a)
			}
			img.SetNRGBA(px, py, c)
		}
	}
	return img
}

func writePNG(path string, img *image.NRGBA) error {
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return err
	}
	return os.WriteFile(path, buf.Bytes(), 0o644)
}

func main() {
	outDir := filepath.Join("build", "dmg")
	if len(os.Args) > 1 {
		outDir = os.Args[1]
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	for _, spec := range []struct {
		scale int
		name  string
	}{{1, "bg@1x.png"}, {2, "bg@2x.png"}} {
		if err := writePNG(filepath.Join(outDir, spec.name), render(spec.scale)); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
	fmt.Printf("wrote bg@1x.png + bg@2x.png to %s\n", outDir)
}
