package renderer

import (
	"image"
	"testing"

	"gittea.kittel.dev/marco/dynamic_background/internal/domain"
)

func TestNewRenderContext(t *testing.T) {
	bounds := image.Rect(0, 0, 1920, 1080)
	rc := NewRenderContext(bounds)

	if rc == nil {
		t.Fatal("expected RenderContext instance")
	}

	if rc.Bounds != bounds {
		t.Errorf("expected bounds %+v, got %+v", bounds, rc.Bounds)
	}
}

func TestRenderContext_DirtyRects(t *testing.T) {
	bounds := image.Rect(0, 0, 1920, 1080)
	rc := NewRenderContext(bounds)

	// Add some dirty widgets
	w1, _ := domain.NewWidget(domain.WidgetTypeCPU, domain.Position{10, 20}, domain.Size{100, 50}, domain.Style{}, 1)
	w2, _ := domain.NewWidget(domain.WidgetTypeClock, domain.Position{200, 300}, domain.Size{150, 60}, domain.Style{}, 1)

	widgets := []*domain.Widget{w1, w2}

	rc.SetDirtyWidgets(widgets)

	dirty := rc.DirtyRects()

	if len(dirty) != 2 {
		t.Errorf("expected 2 dirty rects, got %d", len(dirty))
	}
}

func TestRenderContext_DirtyRects_Union(t *testing.T) {
	bounds := image.Rect(0, 0, 1920, 1080)
	rc := NewRenderContext(bounds)

	// Two overlapping widgets
	w1, _ := domain.NewWidget(domain.WidgetTypeCPU, domain.Position{10, 10}, domain.Size{100, 50}, domain.Style{}, 1)
	w2, _ := domain.NewWidget(domain.WidgetTypeCPU, domain.Position{50, 20}, domain.Size{100, 50}, domain.Style{}, 1)

	// Mark only w1 as dirty
	w1.MarkDirty()
	w2.MarkClean()

	widgets := []*domain.Widget{w1, w2}

	rc.SetDirtyWidgets(widgets)

	dirty := rc.DirtyRects()

	// Only w1 should be dirty
	if len(dirty) != 1 {
		t.Errorf("expected 1 dirty rect, got %d", len(dirty))
	}
}

func TestRenderContext_HasDirtyWidgets(t *testing.T) {
	bounds := image.Rect(0, 0, 1920, 1080)
	rc := NewRenderContext(bounds)

	// No dirty widgets
	if rc.HasDirtyWidgets() {
		t.Error("expected no dirty widgets initially")
	}

	// Add a dirty widget
	w, _ := domain.NewWidget(domain.WidgetTypeCPU, domain.Position{0, 0}, domain.Size{100, 50}, domain.Style{}, 1)
	w.MarkDirty()

	rc.SetDirtyWidgets([]*domain.Widget{w})

	if !rc.HasDirtyWidgets() {
		t.Error("expected dirty widgets after adding one")
	}
}
