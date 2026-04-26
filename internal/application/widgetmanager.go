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
//
// WHY: Renderer braucht nur dirty Widgets → Dirty-Rect-Optimierung → weniger CPU.
// WHAT: Filtert Widgets nach Dirty-Flag.
// IMPACT: Ohne diese Methode müsste der Renderer alle Widgets prüfen → ineffizient.
func (wm *WidgetManager) GetDirtyWidgets() []*domain.Widget {
	var dirty []*domain.Widget

	for _, w := range wm.cfg.Widgets {
		if w.Dirty {
			dirty = append(dirty, w)
		}
	}

	return dirty
}

// RegisterCollector adds a collector for a specific widget type.
//
// WHY: Ermöglicht das Hinzufügen von Collectoren nach der Erstellung.
// WHAT: Speichert Collector in der Map unter dem Widget-Typ.
// IMPACT: Ohne diese Methode müssten alle Collector vor der Erstellung bekannt sein.
func (wm *WidgetManager) RegisterCollector(wt domain.WidgetType, c domain.Collector) {
	wm.collectors[wt] = c
}
