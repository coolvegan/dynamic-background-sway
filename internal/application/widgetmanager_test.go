package application

import (
	"context"
	"testing"
	"time"

	"gittea.kittel.dev/marco/dynamic_background/internal/domain"
)

func TestNewWidgetManager(t *testing.T) {
	collectors := map[domain.WidgetType]domain.Collector{
		domain.WidgetTypeCPU: domain.NewStaticCollector(domain.CollectorData{Value: "50%"}),
	}

	cfg, err := domain.NewConfig(domain.Config{
		Widgets: []*domain.Widget{},
		Background: domain.BackgroundConfig{
			Type: domain.BackgroundTypeSolid,
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	wm := NewWidgetManager(cfg, collectors)

	if wm == nil {
		t.Fatal("expected WidgetManager instance")
	}
}

func TestWidgetManager_UpdateWidget(t *testing.T) {
	collector := domain.NewStaticCollector(domain.CollectorData{Value: "42%"})
	collectors := map[domain.WidgetType]domain.Collector{
		domain.WidgetTypeCPU: collector,
	}

	w, err := domain.NewWidget(domain.WidgetTypeCPU, domain.Position{0, 0}, domain.Size{100, 50}, domain.Style{}, time.Second)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cfg, err := domain.NewConfig(domain.Config{
		Widgets: []*domain.Widget{w},
		Background: domain.BackgroundConfig{
			Type: domain.BackgroundTypeSolid,
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	wm := NewWidgetManager(cfg, collectors)

	ctx := context.Background()
	err = wm.UpdateWidget(ctx, w)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if w.Value != "42%" {
		t.Errorf("expected widget value '42%%', got %q", w.Value)
	}
	if !w.Dirty {
		t.Error("expected widget to be dirty after update")
	}
}

func TestWidgetManager_UpdateWidget_NoCollector(t *testing.T) {
	w, err := domain.NewWidget(domain.WidgetTypeCPU, domain.Position{0, 0}, domain.Size{100, 50}, domain.Style{}, time.Second)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cfg, err := domain.NewConfig(domain.Config{
		Widgets: []*domain.Widget{w},
		Background: domain.BackgroundConfig{
			Type: domain.BackgroundTypeSolid,
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// No collectors registered
	wm := NewWidgetManager(cfg, nil)

	ctx := context.Background()
	err = wm.UpdateWidget(ctx, w)

	if err == nil {
		t.Fatal("expected error when no collector is registered")
	}
}

func TestWidgetManager_UpdateAll(t *testing.T) {
	collectors := map[domain.WidgetType]domain.Collector{
		domain.WidgetTypeCPU:    domain.NewStaticCollector(domain.CollectorData{Value: "50%"}),
		domain.WidgetTypeMemory: domain.NewStaticCollector(domain.CollectorData{Value: "8GB"}),
	}

	w1, _ := domain.NewWidget(domain.WidgetTypeCPU, domain.Position{0, 0}, domain.Size{100, 50}, domain.Style{}, time.Second)
	w2, _ := domain.NewWidget(domain.WidgetTypeMemory, domain.Position{100, 0}, domain.Size{100, 50}, domain.Style{}, time.Second)

	cfg, err := domain.NewConfig(domain.Config{
		Widgets: []*domain.Widget{w1, w2},
		Background: domain.BackgroundConfig{
			Type: domain.BackgroundTypeSolid,
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	wm := NewWidgetManager(cfg, collectors)

	ctx := context.Background()
	updated := wm.UpdateAll(ctx)

	if updated != 2 {
		t.Errorf("expected 2 widgets updated, got %d", updated)
	}
	if w1.Value != "50%" {
		t.Errorf("expected CPU value '50%%', got %q", w1.Value)
	}
	if w2.Value != "8GB" {
		t.Errorf("expected Memory value '8GB', got %q", w2.Value)
	}
}

func TestWidgetManager_UpdateAll_SomeFail(t *testing.T) {
	collectors := map[domain.WidgetType]domain.Collector{
		domain.WidgetTypeCPU:    domain.NewStaticCollector(domain.CollectorData{Value: "50%"}),
		domain.WidgetTypeMemory: domain.NewErrorCollector(&domain.CollectorError{Err: "failed"}),
	}

	w1, _ := domain.NewWidget(domain.WidgetTypeCPU, domain.Position{0, 0}, domain.Size{100, 50}, domain.Style{}, time.Second)
	w2, _ := domain.NewWidget(domain.WidgetTypeMemory, domain.Position{100, 0}, domain.Size{100, 50}, domain.Style{}, time.Second)

	cfg, err := domain.NewConfig(domain.Config{
		Widgets: []*domain.Widget{w1, w2},
		Background: domain.BackgroundConfig{
			Type: domain.BackgroundTypeSolid,
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	wm := NewWidgetManager(cfg, collectors)

	ctx := context.Background()
	updated := wm.UpdateAll(ctx)

	// CPU should succeed, Memory should fail but still count as "attempted"
	if updated != 2 {
		t.Errorf("expected 2 widgets attempted, got %d", updated)
	}
	if w1.Value != "50%" {
		t.Errorf("expected CPU value '50%%', got %q", w1.Value)
	}
	// Memory widget should have error value
	if w1.Value != "50%" {
		t.Errorf("expected CPU to still work, got %q", w1.Value)
	}
}

func TestWidgetManager_GetDirtyWidgets(t *testing.T) {
	collectors := map[domain.WidgetType]domain.Collector{
		domain.WidgetTypeCPU: domain.NewStaticCollector(domain.CollectorData{Value: "50%"}),
	}

	w1, _ := domain.NewWidget(domain.WidgetTypeCPU, domain.Position{0, 0}, domain.Size{100, 50}, domain.Style{}, time.Second)
	w2, _ := domain.NewWidget(domain.WidgetTypeCPU, domain.Position{100, 0}, domain.Size{100, 50}, domain.Style{}, time.Second)

	cfg, err := domain.NewConfig(domain.Config{
		Widgets: []*domain.Widget{w1, w2},
		Background: domain.BackgroundConfig{
			Type: domain.BackgroundTypeSolid,
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	wm := NewWidgetManager(cfg, collectors)

	ctx := context.Background()
	wm.UpdateAll(ctx)

	dirty := wm.GetDirtyWidgets()

	// Both widgets should be dirty after update
	if len(dirty) != 2 {
		t.Errorf("expected 2 dirty widgets, got %d", len(dirty))
	}
}

func TestWidgetManager_GetDirtyWidgets_AfterMarkClean(t *testing.T) {
	collectors := map[domain.WidgetType]domain.Collector{
		domain.WidgetTypeCPU: domain.NewStaticCollector(domain.CollectorData{Value: "50%"}),
	}

	w1, _ := domain.NewWidget(domain.WidgetTypeCPU, domain.Position{0, 0}, domain.Size{100, 50}, domain.Style{}, time.Second)
	w2, _ := domain.NewWidget(domain.WidgetTypeCPU, domain.Position{100, 0}, domain.Size{100, 50}, domain.Style{}, time.Second)

	cfg, err := domain.NewConfig(domain.Config{
		Widgets: []*domain.Widget{w1, w2},
		Background: domain.BackgroundConfig{
			Type: domain.BackgroundTypeSolid,
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	wm := NewWidgetManager(cfg, collectors)

	ctx := context.Background()
	wm.UpdateAll(ctx)

	// Mark one widget as clean (simulating it was rendered)
	w1.MarkClean()

	dirty := wm.GetDirtyWidgets()

	if len(dirty) != 1 {
		t.Errorf("expected 1 dirty widget after marking one clean, got %d", len(dirty))
	}
	if dirty[0] != w2 {
		t.Error("expected w2 to be the dirty one")
	}
}
