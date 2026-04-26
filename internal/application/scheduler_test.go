package application

import (
	"context"
	"sync"
	"testing"
	"time"

	"gittea.kittel.dev/marco/dynamic_background/internal/domain"
)

func TestNewScheduler(t *testing.T) {
	cfg, err := domain.NewConfig(domain.Config{
		Widgets: []*domain.Widget{},
		Background: domain.BackgroundConfig{
			Type: domain.BackgroundTypeSolid,
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	wm := NewWidgetManager(cfg, nil)
	sched := NewScheduler(wm)

	if sched == nil {
		t.Fatal("expected Scheduler instance")
	}
}

func TestScheduler_StartStop(t *testing.T) {
	collectors := map[domain.WidgetType]domain.Collector{
		domain.WidgetTypeCPU: domain.NewStaticCollector(domain.CollectorData{Value: "50%"}),
	}

	w, _ := domain.NewWidget(domain.WidgetTypeCPU, domain.Position{0, 0}, domain.Size{100, 50}, domain.Style{}, 100*time.Millisecond)

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
	sched := NewScheduler(wm)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err = sched.Start(ctx)
	if err != nil {
		t.Fatalf("unexpected error starting scheduler: %v", err)
	}

	// Give it time to run at least once
	time.Sleep(150 * time.Millisecond)

	// Stop should not panic or hang
	sched.Stop()
}

func TestScheduler_UpdatesWidgets(t *testing.T) {
	collector := domain.NewStaticCollector(domain.CollectorData{Value: "50%"})
	collectors := map[domain.WidgetType]domain.Collector{
		domain.WidgetTypeCPU: collector,
	}

	w, _ := domain.NewWidget(domain.WidgetTypeCPU, domain.Position{0, 0}, domain.Size{100, 50}, domain.Style{}, 50*time.Millisecond)

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
	sched := NewScheduler(wm)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_ = sched.Start(ctx)

	// Wait for at least one update cycle
	time.Sleep(100 * time.Millisecond)

	sched.Stop()

	if w.Value != "50%" {
		t.Errorf("expected widget value '50%%', got %q", w.Value)
	}
}

func TestScheduler_MultipleIntervals(t *testing.T) {
	fastCollector := domain.NewStaticCollector(domain.CollectorData{Value: "fast"})
	slowCollector := domain.NewStaticCollector(domain.CollectorData{Value: "slow"})

	collectors := map[domain.WidgetType]domain.Collector{
		domain.WidgetTypeCPU:    fastCollector,
		domain.WidgetTypeMemory: slowCollector,
	}

	fastWidget, _ := domain.NewWidget(domain.WidgetTypeCPU, domain.Position{0, 0}, domain.Size{100, 50}, domain.Style{}, 50*time.Millisecond)
	slowWidget, _ := domain.NewWidget(domain.WidgetTypeMemory, domain.Position{100, 0}, domain.Size{100, 50}, domain.Style{}, 200*time.Millisecond)

	cfg, err := domain.NewConfig(domain.Config{
		Widgets: []*domain.Widget{fastWidget, slowWidget},
		Background: domain.BackgroundConfig{
			Type: domain.BackgroundTypeSolid,
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	wm := NewWidgetManager(cfg, collectors)
	sched := NewScheduler(wm)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_ = sched.Start(ctx)

	// Wait for fast widget to update but slow widget may not have updated yet
	time.Sleep(75 * time.Millisecond)

	sched.Stop()

	// Fast widget should have updated
	if fastWidget.Value != "fast" {
		t.Errorf("expected fast widget value 'fast', got %q", fastWidget.Value)
	}
}

func TestScheduler_ConcurrentSafety(t *testing.T) {
	collectors := map[domain.WidgetType]domain.Collector{
		domain.WidgetTypeCPU: domain.NewStaticCollector(domain.CollectorData{Value: "50%"}),
	}

	w, _ := domain.NewWidget(domain.WidgetTypeCPU, domain.Position{0, 0}, domain.Size{100, 50}, domain.Style{}, 10*time.Millisecond)

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
	sched := NewScheduler(wm)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_ = sched.Start(ctx)

	// Let it run for a bit
	time.Sleep(100 * time.Millisecond)

	// Stop should be safe to call multiple times
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sched.Stop()
		}()
	}
	wg.Wait()
}

func TestScheduler_ContextCancellation(t *testing.T) {
	collectors := map[domain.WidgetType]domain.Collector{
		domain.WidgetTypeCPU: domain.NewStaticCollector(domain.CollectorData{Value: "50%"}),
	}

	w, _ := domain.NewWidget(domain.WidgetTypeCPU, domain.Position{0, 0}, domain.Size{100, 50}, domain.Style{}, 50*time.Millisecond)

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
	sched := NewScheduler(wm)

	ctx, cancel := context.WithCancel(context.Background())

	_ = sched.Start(ctx)

	// Cancel immediately
	cancel()

	// Give it time to process cancellation
	time.Sleep(100 * time.Millisecond)

	// Should not panic or hang
	sched.Stop()
}
