// Package renderer provides implementations of the Renderer interface for
// drawing widgets to different targets (Wayland SHM, PNG files).
//
// This file (draw.go) contains the low-level drawing primitives used by both
// WaylandRenderer and ImageRenderer. It handles background rendering (solid,
// gradient, image), widget rendering (background rect + text), text rendering
// with font loading, and hex color parsing.
//
// Why it exists:
//   The drawing logic is shared between the Wayland renderer (real-time display)
//   and the Image renderer (PNG output for testing/non-CGO builds). By keeping
//   draw functions separate from the Renderer implementations, both can reuse
//   the same drawing code. This file is pure Go with no CGO dependencies.
//
// How it connects:
//   - drawBackground() is called by WaylandRenderer.Render() and ImageRenderer.Render()
//     to fill the canvas with solid color, gradient, or background image
//   - drawWidget() is called per widget to draw the widget's background rect
//     (if styled) and text (label + value)
//   - drawText() uses golang.org/x/image/font for text rendering with
//     configurable font face (falls back to basicfont.Face7x13)
//   - parseHexColor() converts "#RGB" or "#RRGGBB" strings to color.RGBA
//   - Font loading delegates to font.go (loadFont, parseFontString)
//
// Key concept: Drawing is done pixel-by-pixel for backgrounds (simple but
// slow for large screens). Widget text uses the Go image/font library.
// There is no GPU acceleration — everything is software-rendered to a
// software image buffer.
package renderer

import (
	"fmt"
	"image"
	"image/color"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"strconv"

	"gittea.kittel.dev/marco/dynamic_background/internal/domain"

	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

// drawBackground fills the image with the configured background.
func drawBackground(img *image.RGBA, config domain.BackgroundConfig) error {
	switch config.Type {
	case domain.BackgroundTypeSolid:
		if len(config.Colors) > 0 {
			c, err := parseHexColor(config.Colors[0])
			if err != nil {
				return err
			}
			drawSolidBackground(img, c)
		}
	case domain.BackgroundTypeGradient:
		if len(config.Colors) >= 2 {
			top, err := parseHexColor(config.Colors[0])
			if err != nil {
				return err
			}
			bottom, err := parseHexColor(config.Colors[1])
			if err != nil {
				return err
			}
			drawGradientBackground(img, top, bottom)
		}
	case domain.BackgroundTypeImage:
		if config.ImagePath != "" {
			return drawImageBackground(img, config.ImagePath)
		}
	}
	return nil
}

func drawSolidBackground(img *image.RGBA, c color.RGBA) {
	for y := img.Rect.Min.Y; y < img.Rect.Max.Y; y++ {
		for x := img.Rect.Min.X; x < img.Rect.Max.X; x++ {
			img.Set(x, y, c)
		}
	}
}

func drawGradientBackground(img *image.RGBA, top, bottom color.RGBA) {
	height := img.Rect.Max.Y - img.Rect.Min.Y
	for y := img.Rect.Min.Y; y < img.Rect.Max.Y; y++ {
		t := float64(y-img.Rect.Min.Y) / float64(height)
		r := uint8(float64(top.R)*(1-t) + float64(bottom.R)*t)
		g := uint8(float64(top.G)*(1-t) + float64(bottom.G)*t)
		b := uint8(float64(top.B)*(1-t) + float64(bottom.B)*t)
		c := color.RGBA{R: r, G: g, B: b, A: 255}
		for x := img.Rect.Min.X; x < img.Rect.Max.X; x++ {
			img.Set(x, y, c)
		}
	}
}

func drawImageBackground(img *image.RGBA, path string) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("opening background image: %w", err)
	}
	defer file.Close()
	bgImg, _, err := image.Decode(file)
	if err != nil {
		return fmt.Errorf("decoding background image: %w", err)
	}

	// Scale image to fill the entire canvas
	bounds := bgImg.Bounds()
	srcW := bounds.Dx()
	srcH := bounds.Dy()
	dstW := img.Rect.Dx()
	dstH := img.Rect.Dy()

	for dy := 0; dy < dstH; dy++ {
		for dx := 0; dx < dstW; dx++ {
			sx := (dx * srcW) / dstW
			sy := (dy * srcH) / dstH
			c := bgImg.At(bounds.Min.X+sx, bounds.Min.Y+sy)
			r, g, b, _ := c.RGBA()
			img.Set(img.Rect.Min.X+dx, img.Rect.Min.Y+dy, color.RGBA{
				R: uint8(r >> 8),
				G: uint8(g >> 8),
				B: uint8(b >> 8),
				A: 255,
			})
		}
	}
	return nil
}

func drawWidget(img *image.RGBA, w *domain.Widget) {
	if w.Style.Background != "" {
		c, err := parseHexColor(w.Style.Background)
		if err == nil {
			bounds := w.Bounds()
			rect := image.Rect(bounds.X, bounds.Y, bounds.X+bounds.Width, bounds.Y+bounds.Height)
			for y := rect.Min.Y; y < rect.Max.Y; y++ {
				for x := rect.Min.X; x < rect.Max.X; x++ {
					img.Set(x, y, c)
				}
			}
		}
	}

	if w.Value != "" {
		c := color.RGBA{R: 255, G: 255, B: 255, A: 255}
		if w.Style.Color != "" {
			if parsed, err := parseHexColor(w.Style.Color); err == nil {
				c = parsed
			}
		}

		fontName := ""
		fontSize := 12
		if w.Style.Font != "" {
			fontName, fontSize = parseFontString(w.Style.Font)
		}

		label := string(w.Type) + ": "
		lineHeight := fontSize + 4
		drawText(img, label, w.Position.X+8, w.Position.Y+lineHeight, c, fontName, fontSize)
		drawText(img, w.Value, w.Position.X+8, w.Position.Y+lineHeight*2, c, fontName, fontSize)
	}
}

func drawText(img *image.RGBA, text string, x, y int, c color.RGBA, fontName string, fontSize int) {
	point := fixed.Point26_6{X: fixed.I(x), Y: fixed.I(y)}

	var face font.Face
	if fontName != "" && fontSize > 0 {
		var err error
		face, err = loadFont(fontName, fontSize)
		if err != nil {
			face = basicfont.Face7x13
		}
	} else {
		face = basicfont.Face7x13
	}

	d := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(c),
		Face: face,
		Dot:  point,
	}
	d.DrawString(text)
}

func parseHexColor(hex string) (color.RGBA, error) {
	hex = hex[1:]
	var r, g, b uint8
	if len(hex) == 3 {
		rv, _ := strconv.ParseUint(hex[0:1]+hex[0:1], 16, 8)
		gv, _ := strconv.ParseUint(hex[1:2]+hex[1:2], 16, 8)
		bv, _ := strconv.ParseUint(hex[2:3]+hex[2:3], 16, 8)
		r, g, b = uint8(rv), uint8(gv), uint8(bv)
	} else if len(hex) == 6 {
		rv, err := strconv.ParseUint(hex[0:2], 16, 8)
		if err != nil {
			return color.RGBA{}, err
		}
		gv, err := strconv.ParseUint(hex[2:4], 16, 8)
		if err != nil {
			return color.RGBA{}, err
		}
		bv, err := strconv.ParseUint(hex[4:6], 16, 8)
		if err != nil {
			return color.RGBA{}, err
		}
		r, g, b = uint8(rv), uint8(gv), uint8(bv)
	} else {
		return color.RGBA{}, fmt.Errorf("invalid hex color length: %d", len(hex))
	}
	return color.RGBA{R: r, G: g, B: b, A: 255}, nil
}
