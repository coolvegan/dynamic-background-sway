//go:build cgo

// Package renderer provides implementations of the Renderer interface for
// drawing widgets to different targets (Wayland SHM, PNG files).
//
// This file (wayland.go) defines WaylandRenderer — the renderer that draws
// directly into a Wayland shared memory (SHM) buffer. It is the primary
// renderer used when running on a real Wayland compositor with CGO enabled.
//
// Why it exists:
//   Wayland compositors require surfaces to be backed by SHM buffers (or
//   EGL/GLES). This renderer creates an in-memory RGBA image, draws the
//   background and widgets onto it, then converts and copies the pixels
//   into the Wayland SHM buffer for the compositor to display.
//
// How it connects:
//   - Created in main_cgo.go: renderer.NewWaylandRenderer(surface, cfg.Background)
//   - Passed to Orchestrator which calls Render(ctx, renderContext) each frame
//   - Render() creates image.RGBA, calls drawBackground() and drawWidget()
//     from draw.go, then calls blitToSHM() to copy to the Wayland buffer
//   - blitToSHM() converts RGBA → ARGB8888 (byte swap: R↔B) because Wayland
//     uses little-endian XRGB8888 format
//   - After Render() returns, Orchestrator calls Surface.Commit() to display
//
// Key concept: The Go-side renders to a single RGBA buffer sized to the
// largest monitor. The C-side (layer.go) copies this buffer to each monitor's
// individual SHM buffer during Commit(). See ARCHITECTURE.md for details.
package renderer

import (
	"context"
	"fmt"
	"image"
	"sync"

	"gittea.kittel.dev/marco/dynamic_background/internal/domain"
	"gittea.kittel.dev/marco/dynamic_background/internal/infrastructure/wayland"
)

// WaylandRenderer renders directly into a Wayland SHM buffer.
type WaylandRenderer struct {
	surface wayland.Surface
	config  domain.BackgroundConfig
	width   int
	height  int
	mu      sync.RWMutex
}

func NewWaylandRenderer(s wayland.Surface, config domain.BackgroundConfig) *WaylandRenderer {
	bounds := s.Bounds()
	return &WaylandRenderer{
		surface: s,
		config:  config,
		width:   bounds.Dx(),
		height:  bounds.Dy(),
	}
}

func (r *WaylandRenderer) SetConfig(cfg domain.BackgroundConfig) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.config = cfg
}

func (r *WaylandRenderer) Render(ctx context.Context, rc *RenderContext) error {
	img := image.NewRGBA(image.Rect(0, 0, r.width, r.height))

	r.mu.RLock()
	cfg := r.config
	r.mu.RUnlock()

	if err := drawBackground(img, cfg); err != nil {
		return fmt.Errorf("drawing background: %w", err)
	}

	for _, w := range rc.Widgets() {
		drawWidget(img, w)
	}

	r.blitToSHM(img)
	return nil
}

func (r *WaylandRenderer) Clear() error {
	img := image.NewRGBA(image.Rect(0, 0, r.width, r.height))

	r.mu.RLock()
	cfg := r.config
	r.mu.RUnlock()

	if err := drawBackground(img, cfg); err != nil {
		return fmt.Errorf("drawing background: %w", err)
	}
	r.blitToSHM(img)
	return nil
}

// Compile-time check: WaylandRenderer implements Renderer
var _ Renderer = (*WaylandRenderer)(nil)

// blitToSHM draws directly into the SHM buffer in XRGB8888 format.
// No intermediate RGBA buffer, no copy.
func (r *WaylandRenderer) blitToSHM(src *image.RGBA) {
	shmBuf := r.surface.Buffer()
	if len(shmBuf) == 0 {
		return
	}

	srcPix := src.Pix
	// Convert RGBA→XRGB while copying
	for i := 0; i < len(srcPix) && i+3 < len(shmBuf); i += 4 {
		shmBuf[i] = srcPix[i+2]     // B
		shmBuf[i+1] = srcPix[i+1]   // G
		shmBuf[i+2] = srcPix[i]     // R
		shmBuf[i+3] = 0xff          // A (unused in XRGB)
	}
}
