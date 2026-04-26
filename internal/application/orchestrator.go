// Package application contains the business logic that wires domain entities
// with infrastructure implementations.
//
// This file (orchestrator.go) defines the Orchestrator — the top-level component
// that coordinates all other parts of the system. It is the "conductor" that
// starts the scheduler (for periodic data collection), runs the render loop
// (for displaying widgets on screen), and manages the lifecycle of the
// Wayland surface.
//
// Why it exists:
//
//	Without the Orchestrator, every component would need to be manually started
//	and synchronized. The Orchestrator ensures:
//	- Initial widget data is collected before first render
//	- Scheduler and render loop run concurrently
//	- Clean shutdown stops all goroutines and disconnects Wayland
//	- Render hook support for external integrations (e.g., swaybg updater)
//
// How it connects:
//   - Created in main_cgo.go with WidgetManager, Scheduler, Renderer, Surface
//   - Start() is called after Wayland connection is established
//   - renderLoop() runs on 100ms ticker, calling renderFrame() each tick
//   - renderFrame() gets widgets from WidgetManager, renders via Renderer,
//     commits via Surface, then marks widgets clean
//   - API server calls UpdateBackgroundConfig() for hot-reload
//   - Stop() is called on SIGINT/SIGTERM for clean shutdown
//
// Key concept: The render loop is time-driven (100ms ticker), not event-driven.
// This is intentional — rendering is cheap and a fixed interval avoids complex
// synchronization between collector goroutines and the render goroutine.
package application

import (
	"context"
	"fmt"
	"time"

	"gittea.kittel.dev/marco/dynamic_background/internal/domain"
	"gittea.kittel.dev/marco/dynamic_background/internal/infrastructure/renderer"
	"gittea.kittel.dev/marco/dynamic_background/internal/infrastructure/wayland"
)

// Orchestrator wires together all application components.
//
// WHY: Alle Komponenten (WidgetManager, Scheduler, Renderer, Surface) müssen zusammenarbeiten.
//
//	Orchestrator ist der "Dirigent" der das gesamte System koordiniert.
//
// WHAT: Startet Scheduler für Updates, Render-Loop für Anzeige,
//
//	verbindet WidgetManager mit Renderer und Wayland Surface.
//
// IMPACT: Ohne Orchestrator müssten alle Komponenten manuell gestartet werden;
//
//	keine Koordination zwischen Updates und Rendering; Race Conditions möglich.
type Orchestrator struct {
	widgetManager  *WidgetManager
	scheduler      *Scheduler
	renderer       renderer.Renderer
	surface        wayland.Surface
	renderInterval time.Duration
	renderHook     func()
}

// NewOrchestrator creates a new Orchestrator instance.
//
// WHY: Factory für saubere Initialisierung aller Komponenten.
// WHAT: Erstellt WidgetManager, Scheduler, speichert Renderer und Surface.
// IMPACT: Ohne Factory müsste Caller alle Komponenten manuell erstellen.
func NewOrchestrator(cfg *domain.Config, collectors map[domain.WidgetType]domain.Collector, r renderer.Renderer, s wayland.Surface) *Orchestrator {
	wm := NewWidgetManager(cfg, collectors)
	sched := NewScheduler(wm)

	return &Orchestrator{
		widgetManager:  wm,
		scheduler:      sched,
		renderer:       r,
		surface:        s,
		renderInterval: 1 * time.Second,
	}
}

// Start begins the orchestrator: scheduler for updates, render loop for display.
//
// WHY: Startet das gesamte System; Updates + Rendering laufen parallel.
// WHAT: Connectet Surface, startet Scheduler, dann Render-Loop in separatem Goroutine.
// IMPACT: Ohne Start() läuft nichts; Background bleibt leer.
func (o *Orchestrator) Start(ctx context.Context) error {
	o.widgetManager.UpdateAll(ctx)

	if err := o.scheduler.Start(ctx); err != nil {
		return fmt.Errorf("starting scheduler: %w", err)
	}

	o.renderFrame(ctx)

	go o.renderLoop(ctx)

	return nil
}

// Stop gracefully shuts down the orchestrator.
//
// WHY: Sauberes Beenden aller Goroutines; verhindert Leaks.
// WHAT: Stoppt Scheduler, disconnectet Surface, Render-Loop beendet sich via Context.
// IMPACT: Ohne Stop() würden Goroutines weiterlaufen → Memory-Leak.
func (o *Orchestrator) Stop() {
	o.scheduler.Stop()
	o.surface.Disconnect()
}

// SetRenderHook sets a callback invoked after each render.
func (o *Orchestrator) SetRenderHook(fn func()) {
	o.renderHook = fn
}

// UpdateBackgroundConfig updates the renderer's background config and triggers an immediate re-render.
func (o *Orchestrator) UpdateBackgroundConfig(cfg domain.BackgroundConfig, ctx context.Context) {
	if wr, ok := o.renderer.(interface{ SetConfig(domain.BackgroundConfig) }); ok {
		wr.SetConfig(cfg)
	}
	o.renderFrame(ctx)
}

// renderLoop periodically checks for dirty widgets and renders them.
//
// WHY: Renderer muss regelmäßig prüfen ob es was zu zeichnen gibt.
//
//	Nur rendern wenn Widgets dirty sind → CPU sparen.
//
// WHAT: Sleep → Dirty check → Nur rendern wenn nötig → Commit → Repeat.
// IMPACT: Ohne Render-Loop würden Widgets updated aber nie angezeigt.
func (o *Orchestrator) renderLoop(ctx context.Context) {
	ticker := time.NewTicker(o.renderInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			widgets := o.widgetManager.GetAllWidgets()
			hasDirty := false
			for _, w := range widgets {
				if w.IsDirty() {
					hasDirty = true
					break
				}
			}
			if hasDirty {
				o.renderFrame(ctx)
			}
		}
	}
}

// renderFrame renders one frame with all widgets.
func (o *Orchestrator) renderFrame(ctx context.Context) {
	allWidgets := o.widgetManager.GetAllWidgets()

	rc := renderer.NewRenderContext(o.surface.Bounds())
	rc.SetWidgets(allWidgets)

	if err := o.renderer.Render(ctx, rc); err != nil {
		return
	}

	// Skip Commit() for renderers that present their own frames (e.g. EGL)
	if _, ok := o.renderer.(interface{ PresentsOwnFrames() bool }); !ok {
		if err := o.surface.Commit(); err != nil {
			return
		}
	}

	if o.renderHook != nil {
		o.renderHook()
	}

	for _, w := range allWidgets {
		w.MarkClean()
	}
}
