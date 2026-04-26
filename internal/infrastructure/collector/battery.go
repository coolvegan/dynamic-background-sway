package collector

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"gittea.kittel.dev/marco/dynamic_background/internal/domain"
)

// BatteryCollector collects battery status from /sys/class/power_supply/.
//
// WHY: Laptop-User brauchen Batterie-Info; wann muss Gerät an Strom?
//      /sys/class/power_supply/BAT0/ ist Standard-Linux-Pfad.
//
// WHAT: Liest status (Charging/Discharging/Full), capacity (%), voltage.
//       Im Gegensatz zu CPU/Memory ist Battery sehr stabil – 10s Refresh reicht.
//
// IMPACT: Ohne BatteryCollector sieht Laptop-User keine Batterie-Info;
//       können leeren Akku nicht vorhersagen.
type BatteryCollector struct {
	batteryPath string
}

// NewBatteryCollector creates a battery collector with default path.
//
// WHY: Factory für Normalfall (erste verfügbare Batterie).
// WHAT: Sucht /sys/class/power_supply/BAT0, fallback BAT1, etc.
// IMPACT: Ohne Factory müsste User Batterie-Pfad manuell kennen.
func NewBatteryCollector() *BatteryCollector {
	// Try common battery paths
	paths := []string{
		"/sys/class/power_supply/BAT0",
		"/sys/class/power_supply/BAT1",
		"/sys/class/power_supply/battery",
	}

	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			return &BatteryCollector{batteryPath: path}
		}
	}

	// Return default even if not found - will error on Collect
	return &BatteryCollector{batteryPath: "/sys/class/power_supply/BAT0"}
}

// NewBatteryCollectorWithPath creates a battery collector with custom path.
//
// WHY: User möchte spezifische Batterie tracken (BAT1, etc.).
// WHAT: Verwendet angegebenen Pfad statt Auto-Detection.
// IMPACT: Ohne diese Factory wäre nur Auto-Detection möglich.
func NewBatteryCollectorWithPath(path string) *BatteryCollector {
	return &BatteryCollector{batteryPath: path}
}

// Collect reads battery status and returns formatted string.
//
// WHY: Holt aktuelle Batterie-Daten für Anzeige im Widget.
// WHAT: Liest status, capacity, voltage aus sysfs.
// IMPACT: Ohne Collect() gibt es keine Batterie-Daten für das Widget.
func (c *BatteryCollector) Collect(ctx context.Context) (domain.CollectorData, error) {
	status, err := c.readFile("status")
	if err != nil {
		return domain.CollectorData{}, fmt.Errorf("reading status: %w", err)
	}

	capacityStr, err := c.readFile("capacity")
	if err != nil {
		return domain.CollectorData{}, fmt.Errorf("reading capacity: %w", err)
	}

	capacity, err := strconv.ParseFloat(strings.TrimSpace(capacityStr), 64)
	if err != nil {
		return domain.CollectorData{}, fmt.Errorf("parsing capacity: %w", err)
	}

	// Voltage is optional
	voltage := ""
	if v, err := c.readFile("voltage_now"); err == nil {
		voltage = formatVoltage(v)
	}

	var value string
	if voltage != "" {
		value = fmt.Sprintf("%s %.0f%% (%s)", status, capacity, voltage)
	} else {
		value = fmt.Sprintf("%s %.0f%%", status, capacity)
	}

	return domain.CollectorData{
		Value:        value,
		NumericValue: capacity,
	}, nil
}

// readFile reads a file from the battery directory.
func (c *BatteryCollector) readFile(name string) (string, error) {
	path := filepath.Join(c.batteryPath, name)
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// formatVoltage converts microvolts to human-readable string.
func formatVoltage(voltageStr string) string {
	voltageStr = strings.TrimSpace(voltageStr)
	voltage, err := strconv.ParseFloat(voltageStr, 64)
	if err != nil {
		return ""
	}

	// Voltage is typically in microvolts (μV)
	volts := voltage / 1000000.0
	return fmt.Sprintf("%.2fV", volts)
}
