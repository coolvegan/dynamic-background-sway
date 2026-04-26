//go:build !cgo

package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"gittea.kittel.dev/marco/dynamic_background/internal/application"
	"gittea.kittel.dev/marco/dynamic_background/internal/domain"
	"gittea.kittel.dev/marco/dynamic_background/internal/infrastructure/collector"
	"gittea.kittel.dev/marco/dynamic_background/internal/infrastructure/config"
	"gittea.kittel.dev/marco/dynamic_background/internal/infrastructure/renderer"
	"gittea.kittel.dev/marco/dynamic_background/internal/infrastructure/wayland"
	"gittea.kittel.dev/marco/dynamic_background/internal/interfaces/api"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	configPath := flag.String("config", "config.yaml", "path to config file")
	outputPath := flag.String("output", "/tmp/dynamic_background.png", "path for rendered image")
	flag.Parse()

	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	collectors := createCollectors()

	s := wayland.NewMockSurface()
	output := wayland.Output{Name: "mock", Width: 1920, Height: 1080, Scale: 1}
	if err := s.CreateSurface(output); err != nil {
		return fmt.Errorf("creating surface: %w", err)
	}

	r := renderer.NewImageRenderer(s.Bounds().Dx(), s.Bounds().Dy(), *outputPath, cfg.Background)
	orch := application.NewOrchestrator(cfg, collectors, r, s)

	bgUpdater := NewBgUpdater("*", *outputPath, "fill", 2*time.Second)
	orch.SetRenderHook(func() { bgUpdater.Update() })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := orch.Start(ctx); err != nil {
		return fmt.Errorf("starting orchestrator: %w", err)
	}

	var apiServer *api.Server
	if cfg.API.Enabled {
		apiServer = api.NewServer(cfg, nil, nil, orch)
		go func() {
			if err := apiServer.Start(); err != nil {
				fmt.Fprintf(os.Stderr, "API server error: %v\n", err)
			}
		}()
	}

	fmt.Println("dynamic_background running (PNG mode). Press Ctrl+C to stop.")

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	fmt.Println("\nshutting down...")
	bgUpdater.Flush()
	orch.Stop()
	return nil
}

func createCollectors() map[domain.WidgetType]domain.Collector {
	return map[domain.WidgetType]domain.Collector{
		domain.WidgetTypeCPU:     collector.NewCPUCollector(),
		domain.WidgetTypeMemory:  collector.NewMemoryCollector(),
		domain.WidgetTypeDisk:    collector.NewDiskCollector(),
		domain.WidgetTypeNetwork: collector.NewNetworkCollector("eth0"),
		domain.WidgetTypeBattery: collector.NewBatteryCollector(),
		domain.WidgetTypeClock:   collector.NewClockCollector(),
		domain.WidgetTypeCustom:  collector.NewCustomCollector(""),
	}
}
