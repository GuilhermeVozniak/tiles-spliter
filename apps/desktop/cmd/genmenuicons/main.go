// Command genmenuicons renders the tray-menu tile glyphs: an 18x18pt (36x36px
// @2x) rounded-rect "screen" outline with the action's target region filled.
// Pure Go stdlib; output is deterministic so the PNGs are committed and CI can
// verify `go run ./cmd/genmenuicons` reproduces them byte-for-byte.
//
// NSMenuItem images are rendered as-is (Wails' setMenuItemBitmap does not mark
// the NSImage as a template), so template black+alpha would stay black on dark
// menus. Instead the glyphs use mid-gray #7f7f7f with antialiased alpha, which
// reads on both light and dark menu appearances. A pHYs chunk (144 DPI) is
// injected so NSImage sizes the 36px bitmap at 18pt.
package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
)

const (
	size = 36  // output canvas in px (@2x for 18pt)
	ss   = 4   // supersampling factor per axis (16 samples/px)
	gray = 127 // #7f7f7f
)

// Screen outline geometry (in 36px canvas units).
const (
	ox0, oy0, ox1, oy1 = 3.0, 8.0, 33.0, 28.0 // outer rounded rect
	outerR             = 5.0
	stroke             = 3.0 // 1.5pt at @2x
)

// Interior fill area: inside the stroke with a 1px gap.
const (
	ix0, iy0 = ox0 + stroke + 1, oy0 + stroke + 1 // 7, 12
	ix1, iy1 = ox1 - stroke - 1, oy1 - stroke - 1 // 29, 24
	iw, ih   = ix1 - ix0, iy1 - iy0               // 22, 12
)

type rect struct{ x0, y0, x1, y1 float64 }

func (r rect) contains(x, y float64) bool {
	return x >= r.x0 && x < r.x1 && y >= r.y0 && y < r.y1
}

type tri struct{ ax, ay, bx, by, cx, cy float64 }

func (t tri) contains(x, y float64) bool {
	d1 := (x-t.bx)*(t.ay-t.by) - (t.ax-t.bx)*(y-t.by)
	d2 := (x-t.cx)*(t.by-t.cy) - (t.bx-t.cx)*(y-t.cy)
	d3 := (x-t.ax)*(t.cy-t.ay) - (t.cx-t.ax)*(y-t.ay)
	hasNeg := d1 < 0 || d2 < 0 || d3 < 0
	hasPos := d1 > 0 || d2 > 0 || d3 > 0
	return !(hasNeg && hasPos)
}

type spec struct {
	rects []rect
	tris  []tri
}

// inRoundedRect reports whether (x,y) lies inside the rounded rect with the
// given bounds and corner radius.
func inRoundedRect(x, y, x0, y0, x1, y1, r float64) bool {
	if x < x0 || x >= x1 || y < y0 || y >= y1 {
		return false
	}
	cx, cy := (x0+x1)/2, (y0+y1)/2
	hw, hh := (x1-x0)/2, (y1-y0)/2
	dx := abs(x-cx) - (hw - r)
	dy := abs(y-cy) - (hh - r)
	if dx <= 0 || dy <= 0 {
		return true
	}
	return dx*dx+dy*dy <= r*r
}

func abs(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}

// inOutline is the screen frame: outer rounded rect minus its inset.
func inOutline(x, y float64) bool {
	if !inRoundedRect(x, y, ox0, oy0, ox1, oy1, outerR) {
		return false
	}
	return !inRoundedRect(x, y, ox0+stroke, oy0+stroke, ox1-stroke, oy1-stroke, outerR-stroke)
}

func specs() map[string]spec {
	const (
		midX = ix0 + iw/2 // 18
		midY = iy0 + ih/2 // 18
		t1   = ix0 + iw/3 // ~14.33 first third boundary
		t2   = ix0 + 2*iw/3
		gap  = 0.5 // half-gap between quadrant fills
	)
	midCol := rect{t1, iy0, t2, iy1}
	return map[string]spec{
		"center":      {rects: []rect{{ix0 + iw/4, iy0 + ih/4, ix1 - iw/4, iy1 - ih/4}}},
		"fullscreen":  {rects: []rect{{ix0, iy0, ix1, iy1}}},
		"half-left":   {rects: []rect{{ix0, iy0, midX, iy1}}},
		"half-right":  {rects: []rect{{midX, iy0, ix1, iy1}}},
		"half-top":    {rects: []rect{{ix0, iy0, ix1, midY}}},
		"half-bottom": {rects: []rect{{ix0, midY, ix1, iy1}}},
		"upper-left":  {rects: []rect{{ix0, iy0, midX - gap, midY - gap}}},
		"upper-right": {rects: []rect{{midX + gap, iy0, ix1, midY - gap}}},
		"lower-left":  {rects: []rect{{ix0, midY + gap, midX - gap, iy1}}},
		"lower-right": {rects: []rect{{midX + gap, midY + gap, ix1, iy1}}},
		// Thirds: middle column filled; a small chevron triangle points at the
		// cycle direction so next/prev read differently at a glance.
		"next-third":        {rects: []rect{midCol}, tris: []tri{{t2 + 2, iy0 + 2.5, t2 + 2, iy1 - 2.5, ix1 - 1, midY}}},
		"prev-third":        {rects: []rect{midCol}, tris: []tri{{t1 - 2, iy0 + 2.5, t1 - 2, iy1 - 2.5, ix0 + 1, midY}}},
		"two-thirds-left":   {rects: []rect{{ix0, iy0, t2, iy1}}},
		"two-thirds-right":  {rects: []rect{{t1, iy0, ix1, iy1}}},
		"two-thirds-center": {rects: []rect{{ix0 + iw/6, iy0, ix1 - iw/6, iy1}}},
		// Displays: filled arrow triangle inside the screen.
		"next-display": {tris: []tri{{midX - 4.5, iy0 + 1, midX - 4.5, iy1 - 1, midX + 6, midY}}},
		"prev-display": {tris: []tri{{midX + 4.5, iy0 + 1, midX + 4.5, iy1 - 1, midX - 6, midY}}},
	}
}

func render(sp spec) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, size, size))
	for py := 0; py < size; py++ {
		for px := 0; px < size; px++ {
			covered := 0
			for sy := 0; sy < ss; sy++ {
				for sx := 0; sx < ss; sx++ {
					x := float64(px) + (float64(sx)+0.5)/ss
					y := float64(py) + (float64(sy)+0.5)/ss
					if ink(sp, x, y) {
						covered++
					}
				}
			}
			if covered > 0 {
				a := uint8(covered * 255 / (ss * ss))
				img.SetNRGBA(px, py, color.NRGBA{gray, gray, gray, a})
			}
		}
	}
	return img
}

func ink(sp spec, x, y float64) bool {
	if inOutline(x, y) {
		return true
	}
	for _, r := range sp.rects {
		if r.contains(x, y) {
			return true
		}
	}
	for _, t := range sp.tris {
		if t.contains(x, y) {
			return true
		}
	}
	return false
}

// withDPI injects a pHYs chunk (5669 px/m ≈ 144 DPI) right after IHDR so
// NSImage reports the 36px bitmap as 18pt. Go's png encoder always emits the
// 8-byte signature followed by a 25-byte IHDR chunk, so the insertion offset
// is fixed.
func withDPI(b []byte) []byte {
	const ihdrEnd = 8 + 25
	body := make([]byte, 0, 13)
	body = append(body, 'p', 'H', 'Y', 's')
	body = binary.BigEndian.AppendUint32(body, 5669) // px per metre, X
	body = binary.BigEndian.AppendUint32(body, 5669) // px per metre, Y
	body = append(body, 1)                           // unit: metre
	out := make([]byte, 0, len(b)+len(body)+8)
	out = append(out, b[:ihdrEnd]...)
	out = binary.BigEndian.AppendUint32(out, 9) // data length (type excluded)
	out = append(out, body...)
	out = binary.BigEndian.AppendUint32(out, crc32.ChecksumIEEE(body))
	return append(out, b[ihdrEnd:]...)
}

func main() {
	outDir := filepath.Join("build", "menu-icons")
	if len(os.Args) > 1 {
		outDir = os.Args[1]
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	for name, sp := range specs() {
		var buf bytes.Buffer
		if err := png.Encode(&buf, render(sp)); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		path := filepath.Join(outDir, name+".png")
		if err := os.WriteFile(path, withDPI(buf.Bytes()), 0o644); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
	fmt.Printf("wrote %d icons to %s\n", len(specs()), outDir)
}
