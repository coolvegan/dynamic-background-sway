package renderer

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"strconv"

	"gittea.kittel.dev/marco/dynamic_background/internal/domain"
)

// ImageRenderer renders widgets to a PNG image.
//
// WHY: Einfachster Renderer ohne externe Dependencies (kein Cairo/Wayland).
//      Funktioniert sofort, testbar, kann mit swaymsg gesetzt werden.
//
// WHAT: Generiert PNG mit Hintergrund (Solid/Gradient) und Widgets als Text.
//       Speichert Datei, optional ruft swaymsg auf.
//
// IMPACT: Ohne ImageRenderer gäbe es keine visuelle Ausgabe; Background wäre leer.
type ImageRenderer struct {
	width     int
	height    int
	outputPath string
	config    domain.BackgroundConfig
}

// NewImageRenderer creates a new image renderer.
//
// WHY: Factory für saubere Initialisierung.
// WHAT: Speichert Dimensionen, Output-Pfad, Background-Config.
// IMPACT: Ohne Factory müsste Caller Interna kennen.
func NewImageRenderer(width, height int, outputPath string, config domain.BackgroundConfig) *ImageRenderer {
	return &ImageRenderer{
		width:     width,
		height:    height,
		outputPath: outputPath,
		config:    config,
	}
}

// Render draws dirty widgets to the image and saves it.
//
// WHY: Generiert das finale Bild mit allen Widgets.
// WHAT: Zeichnet Hintergrund, dann Widgets, speichert als PNG.
// IMPACT: Ohne Render() gibt es kein Bild; Background bleibt leer.
func (r *ImageRenderer) Render(ctx context.Context, rc *RenderContext) error {
	// Create new image
	img := image.NewRGBA(image.Rect(0, 0, r.width, r.height))

	// Draw background
	if err := r.drawBackground(img); err != nil {
		return fmt.Errorf("drawing background: %w", err)
	}

	// Draw dirty widgets
	for _, w := range rc.DirtyWidgets() {
		r.drawWidget(img, w)
	}

	// Save to file
	if err := r.saveImage(img); err != nil {
		return fmt.Errorf("saving image: %w", err)
	}

	return nil
}

// Clear creates a blank image with just the background.
//
// WHY: Löscht alle Widgets vom Screen.
// WHAT: Erstellt neues Bild nur mit Hintergrund.
// IMPACT: Ohne Clear() gäbe es keinen Weg den Screen zu resetten.
func (r *ImageRenderer) Clear() error {
	img := image.NewRGBA(image.Rect(0, 0, r.width, r.height))

	if err := r.drawBackground(img); err != nil {
		return fmt.Errorf("drawing background: %w", err)
	}

	return r.saveImage(img)
}

// drawBackground fills the image with the configured background.
func (r *ImageRenderer) drawBackground(img *image.RGBA) error {
	switch r.config.Type {
	case domain.BackgroundTypeSolid:
		if len(r.config.Colors) > 0 {
			c, err := parseHexColor(r.config.Colors[0])
			if err != nil {
				return err
			}
			drawSolidBackground(img, c)
		}
	case domain.BackgroundTypeGradient:
		if len(r.config.Colors) >= 2 {
			top, err := parseHexColor(r.config.Colors[0])
			if err != nil {
				return err
			}
			bottom, err := parseHexColor(r.config.Colors[1])
			if err != nil {
				return err
			}
			drawGradientBackground(img, top, bottom)
		}
	case domain.BackgroundTypeImage:
		if r.config.ImagePath != "" {
			return r.drawImageBackground(img)
		}
	}

	return nil
}

// drawSolidBackground fills the image with a single color.
func drawSolidBackground(img *image.RGBA, c color.RGBA) {
	for y := img.Rect.Min.Y; y < img.Rect.Max.Y; y++ {
		for x := img.Rect.Min.X; x < img.Rect.Max.X; x++ {
			img.Set(x, y, c)
		}
	}
}

// drawGradientBackground fills the image with a vertical gradient.
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

// drawImageBackground loads and draws an image as background.
func (r *ImageRenderer) drawImageBackground(img *image.RGBA) error {
	file, err := os.Open(r.config.ImagePath)
	if err != nil {
		return fmt.Errorf("opening background image: %w", err)
	}
	defer file.Close()

	bgImg, _, err := image.Decode(file)
	if err != nil {
		return fmt.Errorf("decoding background image: %w", err)
	}

	draw.Draw(img, img.Rect, bgImg, image.Point{}, draw.Src)
	return nil
}

// drawWidget renders a single widget to the image.
func (r *ImageRenderer) drawWidget(img *image.RGBA, w *domain.Widget) {
	// Draw widget background
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

	// Draw widget text
	if w.Value != "" {
		// Simple pixel-based text rendering (placeholder)
		// In production, use a proper font library like freetype
		r.drawSimpleText(img, w)
	}
}

// drawSimpleText renders text as simple pixel blocks (placeholder).
//
// WHY: Echte Font-Rendering benötigt externe Libraries (freetype).
//      Dies ist ein Placeholder der Text-Position und Größe zeigt.
//
// TODO: Ersetzen durch freetype-go für echte Font-Unterstützung.
func (r *ImageRenderer) drawSimpleText(img *image.RGBA, w *domain.Widget) {
	bounds := w.Bounds()
	c := color.RGBA{R: 255, G: 255, B: 255, A: 255}

	if w.Style.Color != "" {
		if parsed, err := parseHexColor(w.Style.Color); err == nil {
			c = parsed
		}
	}

	// Draw a simple rectangle to represent text area
	// This is a placeholder - real implementation would use freetype
	for y := bounds.Y + 5; y < bounds.Y+bounds.Height-5; y++ {
		for x := bounds.X + 5; x < bounds.X+bounds.Width-5; x++ {
			if (x+y)%4 < 2 {
				img.Set(x, y, c)
			}
		}
	}
}

// saveImage encodes and saves the image to disk.
func (r *ImageRenderer) saveImage(img image.Image) error {
	file, err := os.Create(r.outputPath)
	if err != nil {
		return fmt.Errorf("creating file: %w", err)
	}
	defer file.Close()

	if err := png.Encode(file, img); err != nil {
		return fmt.Errorf("encoding PNG: %w", err)
	}

	return nil
}

// parseHexColor parses a hex color string (#RRGGBB or #RGB).
func parseHexColor(hex string) (color.RGBA, error) {
	hex = hex[1:] // Remove #

	var r, g, b uint8

	if len(hex) == 3 {
		// Short form: #RGB
		rv, err := strconv.ParseUint(hex[0:1]+hex[0:1], 16, 8)
		if err != nil {
			return color.RGBA{}, err
		}
		gv, err := strconv.ParseUint(hex[1:2]+hex[1:2], 16, 8)
		if err != nil {
			return color.RGBA{}, err
		}
		bv, err := strconv.ParseUint(hex[2:3]+hex[2:3], 16, 8)
		if err != nil {
			return color.RGBA{}, err
		}
		r = uint8(rv)
		g = uint8(gv)
		b = uint8(bv)
	} else if len(hex) == 6 {
		// Long form: #RRGGBB
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
		r = uint8(rv)
		g = uint8(gv)
		b = uint8(bv)
	} else {
		return color.RGBA{}, fmt.Errorf("invalid hex color length: %d", len(hex))
	}

	return color.RGBA{R: r, G: g, B: b, A: 255}, nil
}
