// Package application contains the business logic that wires domain entities
// with infrastructure implementations.
//
// This file (scheduler.go) defines the Scheduler — the component that triggers
// periodic widget updates based on each widget's individual interval. It runs
// one goroutine per widget, each with its own time.Ticker.
//
// Why it exists:
//   Different widgets need different update frequencies. CPU usage changes
//   every second, but disk usage barely changes. A single fixed interval for
//   all widgets would either waste CPU (updating disk every second) or show
//   stale data (updating CPU every 30 seconds). The Scheduler solves this by
//   giving each widget its own timer.
//
// How it connects:
//   - Created by Orchestrator with a reference to WidgetManager
//   - Start() spawns one goroutine per widget, each running runWidgetTimer()
//   - runWidgetTimer() does an initial update, then ticks at widget.Interval
//   - Each tick calls WidgetManager.UpdateWidget() which fetches data and
//     marks the widget dirty
//   - Stop() cancels the context, which stops all goroutines via wg.Wait()
//   - Orchestrator calls Stop() during shutdown (SIGINT/SIGTERM)
//
// Key concept: The Scheduler does NOT render anything. It only updates widget
// data and sets the dirty flag. The render loop (in Orchestrator) is separate
// and runs at a fixed 100ms interval regardless of collector intervals.
package application

import (
	"context"
	"sync"
	"time"

	"gittea.kittel.dev/marco/dynamic_background/internal/domain"
)

// Scheduler triggers widget updates based on their individual intervals.
//
// WHY: Verschiedene Widgets brauchen verschiedene Update-Frequenzen.
//      CPU braucht 1s, Disk nur 30s. Fester Takt für alle = Ressourcenverschwendung.
//
// WHAT: Startet pro Widget einen Goroutine mit eigenem Timer.
//       Bei Timer-Ablauf → WidgetManager.UpdateWidget → Widget wird dirty.
//
// IMPACT: Ohne Scheduler müssten Widgets manuell getriggert werden → kein automatisches
//       Update → Background zeigt veraltete Daten ODER alles wird jede Sekunde aktualisiert → CPU-Last.
type Scheduler struct {
	widgetManager *WidgetManager
	mu            sync.Mutex
	cancel        context.CancelFunc
	running       bool
}

// NewScheduler creates a new Scheduler instance.
// WHY: Factory-Funktion für saubere Initialisierung.
// WHAT: Speichert WidgetManager-Referenz für spätere Updates.
// IMPACT: Ohne Factory müsste Caller Scheduler-Interna kennen.
func NewScheduler(wm *WidgetManager) *Scheduler {
	return &Scheduler{
		widgetManager: wm,
	}
}

// Start begins the scheduling loop for all widgets.
//
// WHY: Startet die periodischen Updates; jeder Widget bekommt seinen eigenen Timer.
// WHAT: Erzeugt Child-Context, startet pro Widget einen Goroutine mit Ticker.
// IMPACT: Ohne Start() werden Widgets nicht automatisch aktualisiert.
func (s *Scheduler) Start(ctx context.Context) error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return nil
	}

	childCtx, cancel := context.WithCancel(ctx)
	s.cancel = cancel
	s.running = true
	s.mu.Unlock()

	go s.run(childCtx)

	return nil
}

// Stop gracefully shuts down the scheduler.
//
// WHY: Sauberes Beenden aller Timer-Goroutines; verhindert Goroutine-Leaks.
// WHAT: Ruft Cancel-Funktion auf, wartet auf Beendigung.
// IMPACT: Ohne Stop() würden Goroutines weiterlaufen → Memory-Leak.
func (s *Scheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return
	}

	if s.cancel != nil {
		s.cancel()
	}
	s.running = false
}

// run is the main loop that manages per-widget timers.
func (s *Scheduler) run(ctx context.Context) {
	var wg sync.WaitGroup

	for _, w := range s.widgetManager.cfg.Widgets {
		wg.Add(1)
		go func(widget *domain.Widget) {
			defer wg.Done()
			s.runWidgetTimer(ctx, widget)
		}(w)
	}

	// Wait for all widget timers to finish (on context cancellation)
	wg.Wait()
}

// runWidgetTimer runs a ticker loop for a single widget.
func (s *Scheduler) runWidgetTimer(ctx context.Context, w *domain.Widget) {
	// Initial update
	_ = s.widgetManager.UpdateWidget(ctx, w)

	ticker := time.NewTicker(w.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_ = s.widgetManager.UpdateWidget(ctx, w)
		}
	}
}
