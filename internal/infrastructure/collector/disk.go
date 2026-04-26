package collector

import (
	"context"
	"fmt"
	"math"
	"syscall"

	"gittea.kittel.dev/marco/dynamic_background/internal/domain"
)

// DiskCollector collects disk usage for a specific mount point.
//
// WHY: Festplatten-Speicher ist wichtige Systeminfo; User will wissen wann Platte voll.
//      Im Gegensatz zu CPU/Memory ist Disk sehr stabil – 30s Refresh reicht.
//
// WHAT: Nutzt syscall.Statfs für Filesystem-Stats, berechnet (Total - Free) / Total.
//       Konfigurierbar pro Mount-Point (/ /home etc.).
//
// IMPACT: Ohne DiskCollector gibt es keine Speicher-Info; User verpasst volle Platte.
type DiskCollector struct {
	path string
}

// NewDiskCollector creates a disk collector for the root path.
//
// WHY: Factory für Normalfall (root /).
// WHAT: Verwendet "/" als Default-Pfad.
// IMPACT: Ohne Factory müsste Caller Pfad manuell setzen.
func NewDiskCollector() *DiskCollector {
	return &DiskCollector{path: "/"}
}

// NewDiskCollectorWithPath creates a disk collector for a specific path.
//
// WHY: User möchte vielleicht /home oder /data separat tracken.
// WHAT: Verwendet angegebenen Pfad für Statfs-Aufruf.
// IMPACT: Ohne diese Factory wäre nur root / trackbar.
func NewDiskCollectorWithPath(path string) *DiskCollector {
	return &DiskCollector{path: path}
}

// Collect reads disk stats and calculates usage percentage.
//
// WHY: Holt aktuelle Disk-Daten und berechnet Auslastung.
// WHAT: syscall.Statfs → Total/Free Blocks → Berechnet %.
// IMPACT: Ohne Collect() gibt es keine Disk-Daten für das Widget.
func (c *DiskCollector) Collect(ctx context.Context) (domain.CollectorData, error) {
	var stat syscall.Statfs_t

	err := syscall.Statfs(c.path, &stat)
	if err != nil {
		return domain.CollectorData{}, fmt.Errorf("statfs %s: %w", c.path, err)
	}

	// Blocks are typically 4096 bytes
	blockSize := uint64(stat.Bsize)
	totalBytes := stat.Blocks * blockSize
	freeBytes := stat.Bavail * blockSize // Bavail = free for non-root

	percent := calculateDiskPercent(totalBytes, freeBytes)
	usedBytes := totalBytes - freeBytes

	return domain.CollectorData{
		Value:        fmt.Sprintf("%.1f%% (%s / %s)", percent, formatBytes(usedBytes), formatBytes(totalBytes)),
		NumericValue: percent,
	}, nil
}

// calculateDiskPercent calculates disk usage percentage.
//
// WHY: Disk% = (Total - Free) / Total * 100
// WHAT: Berechnet Prozent aus Total und Free Bytes.
// IMPACT: Ohne Berechnung haben wir nur Byte-Werte, keine verständliche %.
func calculateDiskPercent(total, free uint64) float64 {
	if total == 0 {
		return 0
	}

	used := total - free
	return float64(used) / float64(total) * 100
}

// formatBytes converts bytes to human-readable string.
//
// WHY: 1073741824 Bytes ist nicht lesbar; "1.0 GB" schon.
// WHAT: Wandelt Bytes in B/KB/MB/GB/TB um.
// IMPACT: Ohne Formatierung wären Disk-Werte schwer verständlich.
func formatBytes(bytes uint64) string {
	if bytes < 1024 {
		return fmt.Sprintf("%d B", bytes)
	}

	units := []string{"KB", "MB", "GB", "TB"}
	value := float64(bytes)

	for _, unit := range units {
		value /= 1024.0
		if value < 1024.0 || unit == "TB" {
			return fmt.Sprintf("%.1f %s", value, unit)
		}
	}

	// Fallback for very large values
	return fmt.Sprintf("%.1f TB", value*math.Pow(1024, 4))
}
