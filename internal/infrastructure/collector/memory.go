package collector

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"gittea.kittel.dev/marco/dynamic_background/internal/domain"
)

// MemInfo holds memory statistics from /proc/meminfo.
//
// WHY: Linux stellt RAM-Information in /proc/meminfo bereit.
//      Diese Struktur extrahiert die relevanten Felder.
//
// WHAT: Total, Free, Available, Buffers, Cached in kB.
// IMPACT: Ohne diese Struktur können wir keine RAM-Auslastung berechnen.
type MemInfo struct {
	Total     uint64
	Free      uint64
	Available uint64
	Buffers   uint64
	Cached    uint64
}

// MemoryCollector collects memory usage from /proc/meminfo.
//
// WHY: RAM-Nutzung ist eine der wichtigsten Systeminformationen.
//      /proc/meminfo ist der Standard-Linux-Weg diese Daten zu holen.
//
// WHAT: Liest /proc/meminfo, berechnet (Total - Available) / Total.
//       Im Gegensatz zu CPU braucht Memory KEIN Delta – jeder Aufruf ist absolut.
//
// IMPACT: Ohne MemoryCollector gibt es kein RAM-Widget; wichtige Info fehlt.
type MemoryCollector struct {
	reader io.Reader
}

// NewMemoryCollector creates a memory collector that reads from /proc/meminfo.
//
// WHY: Factory für den Normalfall (echtes System).
// WHAT: Öffnet /proc/meminfo bei jedem Collect-Aufruf.
// IMPACT: Ohne Factory müsste Caller /proc/meminfo manuell öffnen.
func NewMemoryCollector() *MemoryCollector {
	return &MemoryCollector{}
}

// NewMemoryCollectorWithReader creates a memory collector with a custom reader.
//
// WHY: Ermöglicht Testing mit fake /proc/meminfo Daten.
// WHAT: Verwendet Reader statt /proc/meminfo zu öffnen.
// IMPACT: Ohne diese Factory wäre MemoryCollector nicht testbar.
func NewMemoryCollectorWithReader(reader io.Reader) *MemoryCollector {
	return &MemoryCollector{reader: reader}
}

// Collect reads memory stats and calculates usage percentage.
//
// WHY: Holt aktuelle RAM-Daten und berechnet Auslastung.
// WHAT: Liest /proc/meminfo, berechnet (Total - Available) / Total * 100.
// IMPACT: Ohne Collect() gibt es keine RAM-Daten für das Widget.
func (c *MemoryCollector) Collect(ctx context.Context) (domain.CollectorData, error) {
	var info MemInfo
	var err error

	if c.reader != nil {
		info, err = parseMemInfo(c.reader)
	} else {
		file, err := os.Open("/proc/meminfo")
		if err != nil {
			return domain.CollectorData{}, fmt.Errorf("opening /proc/meminfo: %w", err)
		}
		defer file.Close()

		info, err = parseMemInfo(file)
	}

	if err != nil {
		return domain.CollectorData{}, fmt.Errorf("parsing memory info: %w", err)
	}

	percent := calculateMemoryPercent(info)
	usedMB := (info.Total - info.Available) / 1024
	totalMB := info.Total / 1024

	return domain.CollectorData{
		Value:        fmt.Sprintf("%.1f%% (%d MB / %d MB)", percent, usedMB, totalMB),
		NumericValue: percent,
	}, nil
}

// parseMemInfo parses the key-value pairs from /proc/meminfo.
//
// WHY: /proc/meminfo hat "Key: Value kB" Format das geparst werden muss.
// WHAT: Liest relevante Keys (MemTotal, MemFree, MemAvailable, Buffers, Cached).
// IMPACT: Ohne Parser können wir die Rohdaten nicht verstehen.
func parseMemInfo(reader io.Reader) (MemInfo, error) {
	scanner := bufio.NewScanner(reader)
	info := MemInfo{}

	found := make(map[string]bool)

	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Fields(line)
		if len(parts) < 3 {
			continue
		}

		key := strings.TrimSuffix(parts[0], ":")
		value, err := strconv.ParseUint(parts[1], 10, 64)
		if err != nil {
			continue
		}

		switch key {
		case "MemTotal":
			info.Total = value
			found["MemTotal"] = true
		case "MemFree":
			info.Free = value
		case "MemAvailable":
			info.Available = value
		case "Buffers":
			info.Buffers = value
		case "Cached":
			info.Cached = value
		}
	}

	if !found["MemTotal"] {
		return MemInfo{}, fmt.Errorf("MemTotal not found in /proc/meminfo")
	}

	return info, nil
}

// calculateMemoryPercent calculates RAM usage percentage.
//
// WHY: RAM% = (Total - Available) / Total * 100
//      "Available" ist besser als "Free" da es Buffers/Cache berücksichtigt.
//
// WHAT: Berechnet Prozent aus Total und Available.
// IMPACT: Ohne Berechnung haben wir nur kB-Werte, keine verständliche %.
func calculateMemoryPercent(info MemInfo) float64 {
	if info.Total == 0 {
		return 0
	}

	used := info.Total - info.Available
	return float64(used) / float64(info.Total) * 100
}
