//go:build cgo

package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

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
	flag.Parse()

	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	collectors := createCollectors()

	s := wayland.NewLayerSurface()
	if err := s.Connect(context.Background()); err != nil {
		return fmt.Errorf("connecting to wayland: %w", err)
	}

	outputs := s.Outputs()
	if len(outputs) == 0 {
		return fmt.Errorf("no wayland outputs found")
	}
	output := outputs[0]

	if err := s.CreateSurface(output); err != nil {
		return fmt.Errorf("creating surface: %w", err)
	}

	var r renderer.Renderer
	switch cfg.Renderer.Type {
	case domain.RendererTypeEGL:
		egl := renderer.NewEGLRenderer(s, cfg.Background)
		if err := egl.Init(); err != nil {
			return fmt.Errorf("initializing EGL renderer: %w", err)
		}
		r = egl
		fmt.Println("using EGL renderer (hardware-accelerated)")
	default:
		r = renderer.NewWaylandRenderer(s, cfg.Background)
		fmt.Println("using Wayland SHM renderer (software)")
	}
	orch := application.NewOrchestrator(cfg, collectors, r, s)

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

	fmt.Println("dynamic_background running (native wayland layer). Press Ctrl+C to stop.")

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	fmt.Println("\nshutting down...")
	if cr, ok := r.(interface{ Cleanup() }); ok {
		cr.Cleanup()
	}
	orch.Stop()
	return nil
}

func createCollectors() map[domain.WidgetType]domain.Collector {
	return map[domain.WidgetType]domain.Collector{
		domain.WidgetTypeCPU:     collector.NewCPUCollector(),
		domain.WidgetTypeMemory:  collector.NewMemoryCollector(),
		domain.WidgetTypeDisk:    collector.NewDiskCollector(),
		domain.WidgetTypeNetwork: collector.NewNetworkCollector(""),
		domain.WidgetTypeBattery: collector.NewBatteryCollector(),
		domain.WidgetTypeClock:   collector.NewClockCollector(),
		domain.WidgetTypeCustom:  collector.NewCustomCollector(""),
	}
}
