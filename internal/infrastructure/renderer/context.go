package renderer

import (
	"image"

	"gittea.kittel.dev/marco/dynamic_background/internal/domain"
)

// RenderContext holds the state for a single render operation.
//
// WHY: Renderer braucht Kontext über Screen-Größe, dirty Rects, etc.
//      Diese Struktur kapselt alle Render-Parameter.
//
// WHAT: Screen bounds, dirty widgets, dirty rects union.
// IMPACT: Ohne RenderContext müsste jeder Renderer-Call alle Parameter einzeln übergeben.
type RenderContext struct {
	Bounds      image.Rectangle
	dirtyWidgets []*domain.Widget
	dirtyRects   []image.Rectangle
}

// NewRenderContext creates a new render context for the given screen bounds.
//
// WHY: Factory für saubere Initialisierung.
// WHAT: Speichert Bounds, initialisiert leere dirty-Listen.
// IMPACT: Ohne Factory müsste Caller Interna kennen.
func NewRenderContext(bounds image.Rectangle) *RenderContext {
	return &RenderContext{
		Bounds: bounds,
	}
}

// SetDirtyWidgets sets the widgets that need re-rendering.
func (rc *RenderContext) SetDirtyWidgets(widgets []*domain.Widget) {
	rc.dirtyWidgets = nil
	rc.dirtyRects = nil

	for _, w := range widgets {
		if w.Dirty {
			rc.dirtyWidgets = append(rc.dirtyWidgets, w)
			bounds := w.Bounds()
			rect := image.Rect(
				bounds.X,
				bounds.Y,
				bounds.X+bounds.Width,
				bounds.Y+bounds.Height,
			)
			rc.dirtyRects = append(rc.dirtyRects, rect)
		}
	}
}

// SetWidgets sets all widgets for rendering (regardless of dirty state).
func (rc *RenderContext) SetWidgets(widgets []*domain.Widget) {
	rc.dirtyWidgets = widgets
	rc.dirtyRects = nil

	for _, w := range widgets {
		bounds := w.Bounds()
		rect := image.Rect(
			bounds.X,
			bounds.Y,
			bounds.X+bounds.Width,
			bounds.Y+bounds.Height,
		)
		rc.dirtyRects = append(rc.dirtyRects, rect)
	}
}

// DirtyRects returns the rectangles that need re-rendering.
//
// WHY: Renderer (Cairo/OpenGL) braucht Rects für partial updates.
// WHAT: Returniert berechnete dirty Rects.
// IMPACT: Ohne diese Methode könnte Renderer keine Dirty-Rect-Optimierung machen.
func (rc *RenderContext) DirtyRects() []image.Rectangle {
	return rc.dirtyRects
}

// HasDirtyWidgets checks if there are any widgets to render.
//
// WHY: Schneller Check ob Rendering nötig ist.
// WHAT: Returniert true wenn dirty Widgets vorhanden.
// IMPACT: Ohne diese Methode müsste Renderer immer prüfen ob was zu tun ist.
func (rc *RenderContext) HasDirtyWidgets() bool {
	return len(rc.dirtyWidgets) > 0
}

// DirtyWidgets returns the widgets that need re-rendering.
func (rc *RenderContext) DirtyWidgets() []*domain.Widget {
	return rc.dirtyWidgets
}

// Widgets returns all widgets in the render context.
func (rc *RenderContext) Widgets() []*domain.Widget {
	return rc.dirtyWidgets
}
