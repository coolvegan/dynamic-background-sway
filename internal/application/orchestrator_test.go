package application

import (
	"context"
	"testing"
	"time"

	"gittea.kittel.dev/marco/dynamic_background/internal/domain"
	"gittea.kittel.dev/marco/dynamic_background/internal/infrastructure/collector"
	"gittea.kittel.dev/marco/dynamic_background/internal/infrastructure/renderer"
)

func TestNewOrchestrator(t *testing.T) {
	cfg, err := domain.NewConfig(domain.Config{
		Widgets: []*domain.Widget{},
		Background: domain.BackgroundConfig{
			Type: domain.BackgroundTypeSolid,
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	orch := NewOrchestrator(cfg, nil, nil)

	if orch == nil {
		t.Fatal("expected Orchestrator instance")
	}
}

func TestOrchestrator_StartStop(t *testing.T) {
	collectors := map[domain.WidgetType]domain.Collector{
		domain.WidgetTypeClock: collector.NewClockCollectorWithTimeSource(func() time.Time {
			return time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
		}),
	}

	w, _ := domain.NewWidget(domain.WidgetTypeClock, domain.Position{0, 0}, domain.Size{100, 50}, domain.Style{}, 100*time.Millisecond)

	cfg, err := domain.NewConfig(domain.Config{
		Widgets: []*domain.Widget{w},
		Background: domain.BackgroundConfig{
			Type: domain.BackgroundTypeSolid,
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	r := &renderer.MockRenderer{}
	orch := NewOrchestrator(cfg, collectors, r)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err = orch.Start(ctx)
	if err != nil {
		t.Fatalf("unexpected error starting orchestrator: %v", err)
	}

	// Give it time to run at least one cycle
	time.Sleep(150 * time.Millisecond)

	orch.Stop()

	// Renderer should have been called
	if !r.RenderCalled {
		t.Error("expected renderer to be called")
	}
}

func TestOrchestrator_RenderLoop(t *testing.T) {
	collectors := map[domain.WidgetType]domain.Collector{
		domain.WidgetTypeClock: collector.NewClockCollectorWithTimeSource(func() time.Time {
			return time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
		}),
	}

	w, _ := domain.NewWidget(domain.WidgetTypeClock, domain.Position{0, 0}, domain.Size{100, 50}, domain.Style{}, 50*time.Millisecond)

	cfg, err := domain.NewConfig(domain.Config{
		Widgets: []*domain.Widget{w},
		Background: domain.BackgroundConfig{
			Type: domain.BackgroundTypeSolid,
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	r := &renderer.MockRenderer{}
	orch := NewOrchestrator(cfg, collectors, r)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_ = orch.Start(ctx)

	// Wait for multiple cycles
	time.Sleep(200 * time.Millisecond)

	orch.Stop()

	// Renderer should have been called multiple times
	if !r.RenderCalled {
		t.Error("expected renderer to be called at least once")
	}
}

func TestOrchestrator_CollectorFactory(t *testing.T) {
	collectors := map[domain.WidgetType]domain.Collector{
		domain.WidgetTypeCPU:    collector.NewCPUCollector(),
		domain.WidgetTypeMemory: collector.NewMemoryCollector(),
		domain.WidgetTypeDisk:   collector.NewDiskCollector(),
		domain.WidgetTypeClock:  collector.NewClockCollector(),
		domain.WidgetTypeNetwork: collector.NewNetworkCollector("lo"),
		domain.WidgetTypeBattery: collector.NewBatteryCollector(),
		domain.WidgetTypeCustom:  collector.NewCustomCollector("echo test"),
	}

	// Verify all collectors implement the interface
	for wt, c := range collectors {
		if c == nil {
			t.Errorf("collector for %s is nil", wt)
		}
	}
}
