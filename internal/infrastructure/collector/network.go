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
	"time"

	"gittea.kittel.dev/marco/dynamic_background/internal/domain"
)

// detectDefaultInterface finds the first non-loopback interface from /proc/net/dev.
func detectDefaultInterface() string {
	f, err := os.Open("/proc/net/dev")
	if err != nil {
		return "eth0"
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "lo:") {
			continue
		}
		if idx := strings.Index(line, ":"); idx > 0 {
			return strings.TrimSpace(line[:idx])
		}
	}
	return "eth0"
}

// NetDevStats holds network statistics for a single interface.
//
// WHY: Linux trackt Netzwerk-Traffic in /proc/net/dev.
//      Diese Struktur speichert Rx/Tx Bytes und Packets.
//
// WHAT: Receive und Transmit Bytes/Packets für ein Interface.
// IMPACT: Ohne diese Struktur können wir keine Netzwerk-Geschwindigkeit berechnen.
type NetDevStats struct {
	RxBytes   uint64
	RxPackets uint64
	TxBytes   uint64
	TxPackets uint64
}

// NetworkCollector collects network traffic speed from /proc/net/dev.
//
// WHY: Netzwerk-Traffic ist wichtige Systeminfo; User will Up/Down Speed sehen.
//      Wie CPU braucht Network DELTA zwischen zwei Messungen für Bytes/s.
//
// WHAT: Liest /proc/net/dev, berechnet Delta für spezifisches Interface.
//       Erster Aufruf speichert Snapshot, zweiter berechnet Speed.
//
// IMPACT: Ohne NetworkCollector gibt es keine Netzwerk-Info; Traffic wird nicht angezeigt.
type NetworkCollector struct {
	mu      sync.Mutex
	iface   string
	prev    NetDevStats
	prevTime time.Time
	hasPrev bool
	reader  io.Reader
	opener  func() (io.Reader, error)
}

// NewNetworkCollector creates a network collector for a specific interface.
// If iface is empty, auto-detects the first non-loopback interface.
//
// WHY: Factory für Normalfall (eth0 oder erstes verfügbares Interface).
// WHAT: Verwendet /proc/net/dev als Datenquelle.
// IMPACT: Ohne Factory müsste Caller /proc/net/dev manuell öffnen.
func NewNetworkCollector(iface string) *NetworkCollector {
	if iface == "" {
		iface = detectDefaultInterface()
	}
	return &NetworkCollector{
		iface: iface,
		opener: func() (io.Reader, error) {
			return os.Open("/proc/net/dev")
		},
	}
}

// NewNetworkCollectorWithReader creates a network collector with a custom reader.
//
// WHY: Ermöglicht Testing mit fake /proc/net/dev Daten.
// WHAT: Verwendet Reader statt /proc/net/dev zu öffnen.
// IMPACT: Ohne diese Factory wäre NetworkCollector nicht testbar.
func NewNetworkCollectorWithReader(iface string, reader io.Reader) *NetworkCollector {
	return &NetworkCollector{
		iface:  iface,
		reader: reader,
	}
}

// Collect reads network stats and calculates speed (bytes/s).
//
// WHY: Holt aktuelle Network-Daten und berechnet Speed.
// WHAT: Liest /proc/net/dev, berechnet Delta für Interface → Bytes/s Up/Down.
// IMPACT: Ohne Collect() gibt es keine Network-Daten für das Widget.
func (c *NetworkCollector) Collect(ctx context.Context) (domain.CollectorData, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	current, err := c.readNetDev()
	if err != nil {
		return domain.CollectorData{}, fmt.Errorf("reading /proc/net/dev: %w", err)
	}

	now := time.Now()

	if !c.hasPrev {
		c.prev = current
		c.prevTime = now
		c.hasPrev = true
		return domain.CollectorData{
			Value: "N/A (first reading)",
		}, nil
	}

	elapsed := now.Sub(c.prevTime)
	rxSpeed, txSpeed := calculateNetworkSpeed(c.prev, current, elapsed)
	c.prev = current
	c.prevTime = now

	return domain.CollectorData{
		Value:        fmt.Sprintf("↓ %s/s  ↑ %s/s", formatBytes(uint64(rxSpeed)), formatBytes(uint64(txSpeed))),
		NumericValue: rxSpeed + txSpeed,
	}, nil
}

// readNetDev reads and parses network stats for the configured interface.
func (c *NetworkCollector) readNetDev() (NetDevStats, error) {
	if c.reader != nil {
		stats, err := parseNetDev(c.reader)
		if err != nil {
			return NetDevStats{}, err
		}
		s, exists := stats[c.iface]
		if !exists {
			return NetDevStats{}, fmt.Errorf("interface %s not found", c.iface)
		}
		return s, nil
	}

	if c.opener != nil {
		reader, err := c.opener()
		if err != nil {
			return NetDevStats{}, fmt.Errorf("opening /proc/net/dev: %w", err)
		}
		if closer, ok := reader.(io.Closer); ok {
			defer closer.Close()
		}

		stats, err := parseNetDev(reader)
		if err != nil {
			return NetDevStats{}, err
		}
		s, exists := stats[c.iface]
		if !exists {
			return NetDevStats{}, fmt.Errorf("interface %s not found", c.iface)
		}
		return s, nil
	}

	return NetDevStats{}, fmt.Errorf("no reader or opener configured")
}

// parseNetDev parses /proc/net/dev and returns stats for all interfaces.
//
// WHY: /proc/net/dev hat komplexes Format mit Headern das geparst werden muss.
// WHAT: Skippt Header-Zeilen, parst "interface: rx... tx..." Format.
// IMPACT: Ohne Parser können wir die Rohdaten nicht verstehen.
func parseNetDev(reader io.Reader) (map[string]NetDevStats, error) {
	scanner := bufio.NewScanner(reader)
	stats := make(map[string]NetDevStats)

	lineCount := 0

	for scanner.Scan() {
		line := scanner.Text()
		lineCount++

		// Skip first two header lines
		if lineCount <= 2 {
			continue
		}

		// Format: "  eth0: bytes packets ..."
		if !strings.Contains(line, ":") {
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		iface := strings.TrimSpace(parts[0])
		fields := strings.Fields(parts[1])

		if len(fields) < 16 {
			continue
		}

		rxBytes, _ := strconv.ParseUint(fields[0], 10, 64)
		rxPackets, _ := strconv.ParseUint(fields[1], 10, 64)
		txBytes, _ := strconv.ParseUint(fields[8], 10, 64)
		txPackets, _ := strconv.ParseUint(fields[9], 10, 64)

		stats[iface] = NetDevStats{
			RxBytes:   rxBytes,
			RxPackets: rxPackets,
			TxBytes:   txBytes,
			TxPackets: txPackets,
		}
	}

	if len(stats) == 0 {
		return nil, fmt.Errorf("no interfaces found in /proc/net/dev")
	}

	return stats, nil
}

// calculateNetworkSpeed calculates network speed in bytes/s.
//
// WHY: Speed = Delta(Bytes) / Delta(Time)
// WHAT: Berechnet Rx und Tx Speed aus zwei Snapshots.
// IMPACT: Ohne Berechnung haben wir nur Byte-Zähler, keine Speed-Angabe.
func calculateNetworkSpeed(prev, curr NetDevStats, elapsed time.Duration) (rxSpeed, txSpeed float64) {
	if elapsed <= 0 {
		return 0, 0
	}

	seconds := elapsed.Seconds()
	rxSpeed = float64(curr.RxBytes-prev.RxBytes) / seconds
	txSpeed = float64(curr.TxBytes-prev.TxBytes) / seconds

	return rxSpeed, txSpeed
}
