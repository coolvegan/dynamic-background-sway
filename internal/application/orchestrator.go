package application

import (
	"context"
	"fmt"
	"time"

	"gittea.kittel.dev/marco/dynamic_background/internal/domain"
	"gittea.kittel.dev/marco/dynamic_background/internal/infrastructure/renderer"
)

// Orchestrator wires together all application components.
//
// WHY: Alle Komponenten (WidgetManager, Scheduler, Renderer) müssen zusammenarbeiten.
//      Orchestrator ist der "Dirigent" der das gesamte System koordiniert.
//
// WHAT: Startet Scheduler für Updates, Render-Loop für Anzeige,
//       verbindet WidgetManager mit Renderer.
//
// IMPACT: Ohne Orchestrator müssten alle Komponenten manuell gestartet werden;
//       keine Koordination zwischen Updates und Rendering; Race Conditions möglich.
type Orchestrator struct {
	widgetManager *WidgetManager
	scheduler     *Scheduler
	renderer      renderer.Renderer
	renderInterval time.Duration
}

// NewOrchestrator creates a new Orchestrator instance.
//
// WHY: Factory für saubere Initialisierung aller Komponenten.
// WHAT: Erstellt WidgetManager, Scheduler, speichert Renderer.
// IMPACT: Ohne Factory müsste Caller alle Komponenten manuell erstellen.
func NewOrchestrator(cfg *domain.Config, collectors map[domain.WidgetType]domain.Collector, r renderer.Renderer) *Orchestrator {
	wm := NewWidgetManager(cfg, collectors)
	sched := NewScheduler(wm)

	return &Orchestrator{
		widgetManager:  wm,
		scheduler:      sched,
		renderer:       r,
		renderInterval: 100 * time.Millisecond,
	}
}

// Start begins the orchestrator: scheduler for updates, render loop for display.
//
// WHY: Startet das gesamte System; Updates + Rendering laufen parallel.
// WHAT: Startet Scheduler, dann Render-Loop in separatem Goroutine.
// IMPACT: Ohne Start() läuft nichts; Background bleibt leer.
func (o *Orchestrator) Start(ctx context.Context) error {
	// Initial update of all widgets
	o.widgetManager.UpdateAll(ctx)

	// Start scheduler for periodic updates
	if err := o.scheduler.Start(ctx); err != nil {
		return fmt.Errorf("starting scheduler: %w", err)
	}

	// Start render loop
	go o.renderLoop(ctx)

	return nil
}

// Stop gracefully shuts down the orchestrator.
//
// WHY: Sauberes Beenden aller Goroutines; verhindert Leaks.
// WHAT: Stoppt Scheduler, Render-Loop beendet sich via Context.
// IMPACT: Ohne Stop() würden Goroutines weiterlaufen → Memory-Leak.
func (o *Orchestrator) Stop() {
	o.scheduler.Stop()
}

// renderLoop periodically checks for dirty widgets and renders them.
//
// WHY: Renderer muss regelmäßig prüfen ob es was zu zeichnen gibt.
//      Fester Intervall statt event-driven weil Rendering cheap ist.
//
// WHAT: Sleep → Dirty Widgets holen → Render → Clean markieren → Repeat.
// IMPACT: Ohne Render-Loop würden Widgets updated aber nie angezeigt.
func (o *Orchestrator) renderLoop(ctx context.Context) {
	ticker := time.NewTicker(o.renderInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			o.renderFrame(ctx)
		}
	}
}

// renderFrame renders one frame if there are dirty widgets.
func (o *Orchestrator) renderFrame(ctx context.Context) {
	dirty := o.widgetManager.GetDirtyWidgets()
	if len(dirty) == 0 {
		return
	}

	rc := renderer.NewRenderContext(renderer.MockBounds)
	rc.SetDirtyWidgets(dirty)

	if err := o.renderer.Render(ctx, rc); err != nil {
		// Log error but continue running
		return
	}

	// Mark rendered widgets as clean
	for _, w := range dirty {
		w.MarkClean()
	}
}
