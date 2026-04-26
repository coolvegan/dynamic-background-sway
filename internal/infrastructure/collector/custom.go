package collector

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"gittea.kittel.dev/marco/dynamic_background/internal/domain"
)

// CustomCollector executes a shell command and returns its output.
//
// WHY: User möchte beliebige Daten anzeigen (uptime, weather API, etc.).
//      Custom Scripts ermöglichen maximale Flexibilität ohne Code-Änderung.
//
// WHAT: Führt Command via exec.Command aus, capture stdout, returniert als String.
//       Mit Timeout-Schutz damit hängende Commands den Background nicht blockieren.
//
// IMPACT: Ohne CustomCollector wäre System auf Built-in Widgets beschränkt;
//       keine Erweiterbarkeit durch User; weniger nützlich.
type CustomCollector struct {
	command string
	timeout time.Duration
}

// NewCustomCollector creates a custom collector with default timeout.
//
// WHY: Factory für Normalfall (5s Timeout).
// WHAT: Speichert Command, setzt Default-Timeout.
// IMPACT: Ohne Factory müsste Caller Timeout manuell setzen.
func NewCustomCollector(command string) *CustomCollector {
	return &CustomCollector{
		command: command,
		timeout: 5 * time.Second,
	}
}

// NewCustomCollectorWithTimeout creates a custom collector with custom timeout.
//
// WHY: Manche Commands brauchen länger (API-Calls), andere sollen schnell failen.
// WHAT: Setzt benutzerdefiniertes Timeout.
// IMPACT: Ohne diese Factory wäre Timeout fest auf 5s.
func NewCustomCollectorWithTimeout(command string, timeout time.Duration) *CustomCollector {
	return &CustomCollector{
		command: command,
		timeout: timeout,
	}
}

// Collect executes the command and returns its output.
//
// WHY: Holt Daten von externem Command für Anzeige im Widget.
// WHAT: exec.Command mit Timeout, capture stdout, trim whitespace.
// IMPACT: Ohne Collect() gibt es keine Custom-Script-Daten für das Widget.
func (c *CustomCollector) Collect(ctx context.Context) (domain.CollectorData, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "sh", "-c", c.command)
	output, err := cmd.Output()

	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return domain.CollectorData{}, fmt.Errorf("command timed out after %v: %w", c.timeout, err)
		}
		return domain.CollectorData{}, fmt.Errorf("executing command: %w", err)
	}

	return domain.CollectorData{
		Value: strings.TrimSpace(string(output)),
	}, nil
}
