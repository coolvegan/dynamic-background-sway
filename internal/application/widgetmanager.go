// Package application contains the business logic that wires domain entities
// with infrastructure implementations.
//
// This file (widgetmanager.go) defines the WidgetManager — the component that
// bridges widgets (what to display) with collectors (how to get data). It
// maintains a map of WidgetType → Collector and is responsible for fetching
// data and updating widget values.
//
// Why it exists:
//   Without WidgetManager, widgets would need to know about collectors directly,
//   violating clean architecture (domain would depend on infrastructure). The
//   WidgetManager keeps the domain layer pure and testable:
//   - Widgets only know their type, position, size, style
//   - Collectors only know how to fetch system data
//   - WidgetManager is the glue that connects them
//
// How it connects:
//   - Created by Orchestrator with Config and collector map from main_cgo.go
//   - Scheduler calls UpdateWidget() on each timer tick per widget
//   - UpdateWidget() looks up collector by widget.Type, calls Collect(ctx)
//   - Result is formatted into widget.Value and widget.Data, then MarkDirty()
//   - Orchestrator's render loop calls GetAllWidgets() to get current state
//   - API server can call RegisterCollector() to add new collectors at runtime
//
// Key concept: WidgetManager does NOT own the widgets — it reads them from
// cfg.Widgets (which is shared with the API server). When the API replaces
// cfg.Widgets, the WidgetManager automatically sees the new widgets.
package application

import (
	"context"
	"fmt"

	"gittea.kittel.dev/marco/dynamic_background/internal/domain"
)

// WidgetManager orchestrates the relationship between widgets and collectors.
//
// WHY: Widgets wissen WAS sie anzeigen, Collectors wissen WIE sie Daten holen.
//      WidgetManager verbindet beides – er ist der "Dirigent" im Orchester.
//
// WHAT: Mappt Widget-Typen zu Collectoren, aktualisiert Widgets mit Daten,
//       trackt welche Widgets neu gerendert werden müssen (dirty).
//
// IMPACT: Ohne WidgetManager müssten Widgets selbst Daten sammeln → Domain Layer
//       hätte Dependencies auf Infrastructure → nicht mehr testbar.
type WidgetManager struct {
	cfg        *domain.Config
	collectors map[domain.WidgetType]domain.Collector
}

// NewWidgetManager creates a new WidgetManager instance.
// WHY: Factory-Funktion kapselt die Initialisierung; erlaubt spätere Erweiterung.
// WHAT: Speichert Config und Collector-Map für spätere Verwendung.
// IMPACT: Ohne Factory müsste jeder Caller die Interna von WidgetManager kennen.
func NewWidgetManager(cfg *domain.Config, collectors map[domain.WidgetType]domain.Collector) *WidgetManager {
	if collectors == nil {
		collectors = make(map[domain.WidgetType]domain.Collector)
	}

	return &WidgetManager{
		cfg:        cfg,
		collectors: collectors,
	}
}

// UpdateWidget fetches data from the collector and updates the widget.
//
// WHY: Einzelnes Widget aktualisieren – nützlich für gezielte Updates.
// WHAT: Findet Collector für Widget-Typ, sammelt Daten, setzt Widget.Value und markiert dirty.
// IMPACT: Ohne diese Methode gäbe es keinen Weg, ein einzelnes Widget zu aktualisieren.
func (wm *WidgetManager) UpdateWidget(ctx context.Context, w *domain.Widget) error {
	collector, exists := wm.collectors[w.Type]
	if !exists {
		return fmt.Errorf("no collector registered for widget type %s", w.Type)
	}

	data, err := collector.Collect(ctx)
	if err != nil {
		w.Value = fmt.Sprintf("error: %v", err)
		w.Data = &data
		w.MarkDirty()
		return err
	}

	w.Value = domain.FormatCollectorData(data)
	w.Data = &data
	w.MarkDirty()

	return nil
}

// UpdateAll updates all widgets in the configuration.
//
// WHY: Alle Widgets auf einmal aktualisieren – nützlich für Initialisierung und Full-Refresh.
// WHAT: Iteriert über alle Widgets, ruft UpdateWidget auf, zählt erfolgreiche Updates.
// IMPACT: Ohne diese Methode müsste jeder Caller selbst über Widgets iterieren.
func (wm *WidgetManager) UpdateAll(ctx context.Context) int {
	updated := 0

	for _, w := range wm.cfg.Widgets {
		_ = wm.UpdateWidget(ctx, w)
		updated++
	}

	return updated
}

// GetDirtyWidgets returns all widgets that need re-rendering.
func (wm *WidgetManager) GetDirtyWidgets() []*domain.Widget {
	var dirty []*domain.Widget

	for _, w := range wm.cfg.Widgets {
		if w.Dirty {
			dirty = append(dirty, w)
		}
	}

	return dirty
}

// GetAllWidgets returns all widgets regardless of dirty state.
func (wm *WidgetManager) GetAllWidgets() []*domain.Widget {
	return wm.cfg.Widgets
}

// RegisterCollector adds a collector for a specific widget type.
//
// WHY: Ermöglicht das Hinzufügen von Collectoren nach der Erstellung.
// WHAT: Speichert Collector in der Map unter dem Widget-Typ.
// IMPACT: Ohne diese Methode müssten alle Collector vor der Erstellung bekannt sein.
func (wm *WidgetManager) RegisterCollector(wt domain.WidgetType, c domain.Collector) {
	wm.collectors[wt] = c
}
