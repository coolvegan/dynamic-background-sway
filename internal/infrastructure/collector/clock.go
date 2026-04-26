package collector

import (
	"context"
	"time"

	"gittea.kittel.dev/marco/dynamic_background/internal/domain"
)

// ClockCollector collects the current system time.
//
// WHY: Uhrzeit ist das sichtbarste Widget; User braucht ständigen Zeit-Check.
//      Im Gegensatz zu CPU braucht Clock kein Delta – jeder Aufruf ist unabhängig.
//
// WHAT: Holt aktuelle Zeit, formatiert sie als String.
//       Unterstützt custom Formate und testbare Time-Source.
//
// IMPACT: Ohne ClockCollector gibt es keine Uhr; User sieht keine Zeit im Background.
type ClockCollector struct {
	format     string
	timeSource func() time.Time
}

// NewClockCollector creates a clock collector with default format.
//
// WHY: Factory für den Normalfall (Systemzeit mit Standard-Format).
// WHAT: Verwendet "2006-01-02 15:04:05" als Default-Format.
// IMPACT: Ohne Factory müsste Caller Format und Time-Source manuell setzen.
func NewClockCollector() *ClockCollector {
	return &ClockCollector{
		format:     "2006-01-02 15:04:05",
		timeSource: time.Now,
	}
}

// NewClockCollectorWithFormat creates a clock collector with custom format.
//
// WHY: User möchte vielleicht nur "15:04" oder "Mon 02.01." sehen.
// WHAT: Verwendet Go-Zeitformat (2006-01-02 15:04:05 Referenzzeit).
// IMPACT: Ohne diese Factory wäre nur ein festes Format möglich.
func NewClockCollectorWithFormat(format string) *ClockCollector {
	return &ClockCollector{
		format:     format,
		timeSource: time.Now,
	}
}

// NewClockCollectorWithTimeSource creates a clock collector with custom time source.
//
// WHY: Ermöglicht Testing mit fixer Zeit; deterministische Tests.
// WHAT: Ersetzt time.Now durch eigene Funktion.
// IMPACT: Ohne diese Factory wären Tests von Zeit-abhängiger Logik nicht deterministisch.
func NewClockCollectorWithTimeSource(timeSource func() time.Time) *ClockCollector {
	return &ClockCollector{
		format:     "2006-01-02 15:04:05",
		timeSource: timeSource,
	}
}

// Collect returns the current time formatted as a string.
//
// WHY: Holt aktuelle Zeit für Anzeige im Widget.
// WHAT: Ruft timeSource auf, formatiert mit configured format.
// IMPACT: Ohne Collect() gibt es keine Zeit-Daten für das Clock-Widget.
func (c *ClockCollector) Collect(ctx context.Context) (domain.CollectorData, error) {
	now := c.timeSource()
	return domain.CollectorData{
		Value: now.Format(c.format),
	}, nil
}
