package renderer

import (
	"context"
	"fmt"
	"image"
	"image/png"
	"os"

	"gittea.kittel.dev/marco/dynamic_background/internal/domain"
)

// ImageRenderer renders widgets to a PNG image.
type ImageRenderer struct {
	width      int
	height     int
	outputPath string
	config     domain.BackgroundConfig
}

func NewImageRenderer(width, height int, outputPath string, config domain.BackgroundConfig) *ImageRenderer {
	return &ImageRenderer{
		width:      width,
		height:     height,
		outputPath: outputPath,
		config:     config,
	}
}

func (r *ImageRenderer) Render(ctx context.Context, rc *RenderContext) error {
	img := image.NewRGBA(image.Rect(0, 0, r.width, r.height))

	if err := drawBackground(img, r.config); err != nil {
		return fmt.Errorf("drawing background: %w", err)
	}

	for _, w := range rc.Widgets() {
		drawWidget(img, w)
	}

	if r.outputPath != "" {
		if err := saveImage(img, r.outputPath); err != nil {
			return fmt.Errorf("saving image: %w", err)
		}
	}

	return nil
}

func (r *ImageRenderer) Clear() error {
	img := image.NewRGBA(image.Rect(0, 0, r.width, r.height))
	if err := drawBackground(img, r.config); err != nil {
		return fmt.Errorf("drawing background: %w", err)
	}
	return saveImage(img, r.outputPath)
}

func saveImage(img image.Image, path string) error {
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("creating file: %w", err)
	}
	defer file.Close()
	if err := png.Encode(file, img); err != nil {
		return fmt.Errorf("encoding PNG: %w", err)
	}
	return nil
}
