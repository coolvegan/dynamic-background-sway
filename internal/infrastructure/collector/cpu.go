package collector

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"sync"

	"gittea.kittel.dev/marco/dynamic_background/internal/domain"
)

// CPUStats holds the raw jiffies values from /proc/stat.
//
// WHY: Linux trackt CPU-Zeit in "jiffies" (meist 1/100 Sekunde).
//      Diese Struktur speichert die einzelnen Kategorien.
//
// WHAT: Alle Felder aus der "cpu"-Zeile in /proc/stat.
// IMPACT: Ohne diese Struktur können wir keine CPU-Auslastung berechnen.
type CPUStats struct {
	User    uint64
	Nice    uint64
	System  uint64
	Idle    uint64
	IOWait  uint64
	IRQ     uint64
	SoftIRQ uint64
	Steal   uint64
}

// Total returns the sum of all jiffies.
func (s CPUStats) Total() uint64 {
	return s.User + s.Nice + s.System + s.Idle + s.IOWait + s.IRQ + s.SoftIRQ + s.Steal
}

// IdleTotal returns idle + iowait (time CPU was not doing work).
func (s CPUStats) IdleTotal() uint64 {
	return s.Idle + s.IOWait
}

// CPUCollector collects CPU usage from /proc/stat.
//
// WHY: CPU-Auslastung ist eine der wichtigsten Systeminformationen.
//      /proc/stat ist der Standard-Linux-Weg diese Daten zu holen.
//
// WHAT: Liest /proc/stat, berechnet Delta zwischen zwei Messungen.
//       Erster Aufruf speichert Snapshot, zweiter berechnet %.
//
// IMPACT: Ohne CPUCollector gibt es kein CPU-Widget; eine der Kern-Features fehlt.
type CPUCollector struct {
	mu      sync.Mutex
	prev    CPUStats
	hasPrev bool
	reader  io.Reader
	opener  func() (io.Reader, error)
}

// NewCPUCollector creates a CPU collector that reads from /proc/stat.
//
// WHY: Factory für den Normalfall (echtes System).
// WHAT: Öffnet /proc/stat bei jedem Collect-Aufruf neu.
// IMPACT: Ohne diese Factory müsste man /proc/stat manuell öffnen.
func NewCPUCollector() *CPUCollector {
	return &CPUCollector{
		opener: func() (io.Reader, error) {
			return os.Open("/proc/stat")
		},
	}
}

// NewCPUCollectorWithReader creates a CPU collector with a custom reader.
//
// WHY: Ermöglicht Testing mit fake /proc/stat Daten.
// WHAT: Speichert Reader statt /proc/stat zu öffnen.
// IMPACT: Ohne diese Factory wäre CPUCollector nicht testbar.
func NewCPUCollectorWithReader(reader io.Reader) *CPUCollector {
	return &CPUCollector{reader: reader}
}

// Collect reads CPU stats and calculates usage percentage.
//
// WHY: Holt aktuelle CPU-Daten und berechnet Auslastung.
// WHAT: Liest /proc/stat, berechnet Delta zum letzten Aufruf, returniert %.
// IMPACT: Ohne Collect() gibt es keine CPU-Daten für das Widget.
func (c *CPUCollector) Collect(ctx context.Context) (domain.CollectorData, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	current, err := c.readCPUStats()
	if err != nil {
		return domain.CollectorData{}, fmt.Errorf("failed to read CPU stats: %w", err)
	}

	if !c.hasPrev {
		c.prev = current
		c.hasPrev = true
		return domain.CollectorData{
			Value: "N/A (first reading)",
		}, nil
	}

	percent := calculateCPUPercent(c.prev, current)
	c.prev = current

	return domain.CollectorData{
		Value:        fmt.Sprintf("%.1f%%", percent),
		NumericValue: percent,
	}, nil
}

// readCPUStats reads and parses the CPU line from /proc/stat.
func (c *CPUCollector) readCPUStats() (CPUStats, error) {
	if c.reader != nil {
		return parseCPUStats(c.reader)
	}

	if c.opener != nil {
		reader, err := c.opener()
		if err != nil {
			return CPUStats{}, fmt.Errorf("opening /proc/stat: %w", err)
		}
		if closer, ok := reader.(io.Closer); ok {
			defer closer.Close()
		}
		return parseCPUStats(reader)
	}

	return CPUStats{}, fmt.Errorf("no reader or opener configured")
}

// parseCPUStats parses the "cpu" line from /proc/stat.
//
// WHY: /proc/stat hat ein spezifisches Format das geparst werden muss.
// WHAT: Liest erste Zeile die mit "cpu " beginnt, parst die Zahlen.
// IMPACT: Ohne Parser können wir die Rohdaten nicht verstehen.
func parseCPUStats(reader io.Reader) (CPUStats, error) {
	scanner := bufio.NewScanner(reader)

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "cpu ") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 5 {
			return CPUStats{}, fmt.Errorf("invalid cpu line: too few fields")
		}

		// Skip "cpu" prefix
		fields = fields[1:]

		stats := CPUStats{}

		// Parse required fields
		required := []*uint64{
			&stats.User, &stats.Nice, &stats.System, &stats.Idle,
			&stats.IOWait, &stats.IRQ, &stats.SoftIRQ, &stats.Steal,
		}

		for i, ptr := range required {
			if i >= len(fields) {
				break
			}
			val, err := strconv.ParseUint(fields[i], 10, 64)
			if err != nil {
				return CPUStats{}, fmt.Errorf("parsing field %d: %w", i, err)
			}
			*ptr = val
		}

		return stats, nil
	}

	return CPUStats{}, fmt.Errorf("no cpu line found in /proc/stat")
}

// calculateCPUPercent calculates CPU usage percentage between two snapshots.
//
// WHY: CPU% = (Total Delta - Idle Delta) / Total Delta * 100
// WHAT: Berechnet die Differenz zwischen zwei CPUStats und wandelt in % um.
// IMPACT: Ohne diese Berechnung haben wir nur Roh-Jiffies, keine verständliche %.
func calculateCPUPercent(prev, curr CPUStats) float64 {
	totalDelta := curr.Total() - prev.Total()
	if totalDelta == 0 {
		return 0
	}

	idleDelta := curr.IdleTotal() - prev.IdleTotal()
	activeDelta := totalDelta - idleDelta

	return float64(activeDelta) / float64(totalDelta) * 100
}
