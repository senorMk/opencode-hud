// Command genicon generates the opencode-hud icon set with only the Go
// standard library (no font/tooling dependencies), so it stays reproducible:
//
//	go run ./scripts/genicon
//
// Output:
//
//	internal/tray/icons/tray-template.png — monochrome template mask for the
//	    menu bar (macOS renders it for light/dark automatically).
//	internal/tray/icons/appicon-512.png — colour icon for the app/window,
//	    reused by the .app bundle in M5.
package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
)

func main() {
	out := filepath.Join("internal", "tray", "icons")
	if err := os.MkdirAll(out, 0o755); err != nil {
		fatal(err)
	}

	// Menu-bar template: white ascending bars on transparent.
	tray := image.NewRGBA(image.Rect(0, 0, 44, 44))
	drawBars(tray, barSpec{white: color.White, bg: nil, radius: 3})
	if err := writePNG(filepath.Join(out, "tray-template.png"), tray); err != nil {
		fatal(err)
	}

	// Colour app icon: dark rounded tile + green ascending bars.
	app := image.NewRGBA(image.Rect(0, 0, 512, 512))
	drawTile(app, 112, color.RGBA{0x1e, 0x1e, 0x1e, 0xff})
	drawBars(app, barSpec{white: color.RGBA{0x30, 0xd1, 0x58, 0xff}, bg: nil, radius: 28})
	if err := writePNG(filepath.Join(out, "appicon-512.png"), app); err != nil {
		fatal(err)
	}
	fmt.Println("icons written to", out)
}

type barSpec struct {
	white  color.Color
	bg     color.Color // nil = transparent
	radius int
}

// drawBars paints three ascending rounded bars centred on the canvas.
func drawBars(img *image.RGBA, spec barSpec) {
	b := img.Bounds()
	n := 3
	gap := b.Dx() / 22
	if gap < 2 {
		gap = 2
	}
	totalGaps := gap * (n - 1)
	barW := (b.Dx()*2/3 - totalGaps) / n
	heights := []float64{0.38, 0.62, 0.88}
	base := b.Dy() * 9 / 10
	startX := (b.Dx() - (barW*n + totalGaps)) / 2
	for i, h := range heights {
		bh := int(float64(b.Dy()) * h)
		x0 := startX + i*(barW+gap)
		r := image.Rect(x0, base-bh, x0+barW, base)
		fillRounded(img, r, spec.radius, spec.white)
	}
}

// drawTile paints a full-bleed rounded square.
func drawTile(img *image.RGBA, radius int, c color.Color) {
	fillRounded(img, img.Bounds(), radius, c)
}

// fillRounded fills r with c, honouring corner radius (pixels outside the
// rounded shape are left untouched).
func fillRounded(img *image.RGBA, r image.Rectangle, radius int, c color.Color) {
	if radius < 0 {
		radius = 0
	}
	fr := float64(radius)
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			dx := 0.0
			dy := 0.0
			if x < r.Min.X+radius {
				dx = float64(r.Min.X+radius-x) - 0.5
			} else if x >= r.Max.X-radius {
				dx = float64(x-(r.Max.X-radius)) + 0.5
			}
			if y < r.Min.Y+radius {
				dy = float64(r.Min.Y+radius-y) - 0.5
			} else if y >= r.Max.Y-radius {
				dy = float64(y-(r.Max.Y-radius)) + 0.5
			}
			if dx > 0 && dy > 0 && math.Hypot(dx, dy) > fr {
				continue
			}
			img.Set(x, y, c)
		}
	}
}

func writePNG(path string, img image.Image) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "genicon:", err)
	os.Exit(1)
}
