package collector

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBatteryCollector_Collect(t *testing.T) {
	// Create mock battery directory
	tmpDir := t.TempDir()
	batteryDir := filepath.Join(tmpDir, "BAT0")

	if err := os.MkdirAll(batteryDir, 0755); err != nil {
		t.Fatalf("failed to create mock battery dir: %v", err)
	}

	// Write mock battery data
	mockData := map[string]string{
		"status":      "Discharging",
		"capacity":    "75",
		"voltage_now": "12000000",
		"current_now": "1500000",
	}

	for file, content := range mockData {
		if err := os.WriteFile(filepath.Join(batteryDir, file), []byte(content), 0644); err != nil {
			t.Fatalf("failed to write mock file %s: %v", file, err)
		}
	}

	c := NewBatteryCollectorWithPath(batteryDir)

	ctx := context.Background()
	data, err := c.Collect(ctx)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(data.Value, "75") {
		t.Errorf("expected value containing '75', got %q", data.Value)
	}

	if !strings.Contains(data.Value, "Discharging") {
		t.Errorf("expected value containing 'Discharging', got %q", data.Value)
	}

	if data.NumericValue != 75.0 {
		t.Errorf("expected numeric value 75.0, got %.1f", data.NumericValue)
	}
}

func TestBatteryCollector_Collect_Charging(t *testing.T) {
	tmpDir := t.TempDir()
	batteryDir := filepath.Join(tmpDir, "BAT0")

	if err := os.MkdirAll(batteryDir, 0755); err != nil {
		t.Fatalf("failed to create mock battery dir: %v", err)
	}

	os.WriteFile(filepath.Join(batteryDir, "status"), []byte("Charging"), 0644)
	os.WriteFile(filepath.Join(batteryDir, "capacity"), []byte("50"), 0644)

	c := NewBatteryCollectorWithPath(batteryDir)

	ctx := context.Background()
	data, err := c.Collect(ctx)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(data.Value, "Charging") {
		t.Errorf("expected value containing 'Charging', got %q", data.Value)
	}
}

func TestBatteryCollector_Collect_Full(t *testing.T) {
	tmpDir := t.TempDir()
	batteryDir := filepath.Join(tmpDir, "BAT0")

	if err := os.MkdirAll(batteryDir, 0755); err != nil {
		t.Fatalf("failed to create mock battery dir: %v", err)
	}

	os.WriteFile(filepath.Join(batteryDir, "status"), []byte("Full"), 0644)
	os.WriteFile(filepath.Join(batteryDir, "capacity"), []byte("100"), 0644)

	c := NewBatteryCollectorWithPath(batteryDir)

	ctx := context.Background()
	data, err := c.Collect(ctx)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(data.Value, "Full") {
		t.Errorf("expected value containing 'Full', got %q", data.Value)
	}

	if data.NumericValue != 100.0 {
		t.Errorf("expected numeric value 100.0, got %.1f", data.NumericValue)
	}
}

func TestBatteryCollector_Collect_MissingDirectory(t *testing.T) {
	c := NewBatteryCollectorWithPath("/nonexistent/battery/path")

	ctx := context.Background()
	_, err := c.Collect(ctx)

	if err == nil {
		t.Error("expected error for missing battery directory")
	}
}

func TestBatteryCollector_Collect_MissingCapacityFile(t *testing.T) {
	tmpDir := t.TempDir()
	batteryDir := filepath.Join(tmpDir, "BAT0")

	if err := os.MkdirAll(batteryDir, 0755); err != nil {
		t.Fatalf("failed to create mock battery dir: %v", err)
	}

	// Only write status, not capacity
	os.WriteFile(filepath.Join(batteryDir, "status"), []byte("Discharging"), 0644)

	c := NewBatteryCollectorWithPath(batteryDir)

	ctx := context.Background()
	_, err := c.Collect(ctx)

	if err == nil {
		t.Error("expected error for missing capacity file")
	}
}

func TestBatteryCollector_Collect_InvalidCapacity(t *testing.T) {
	tmpDir := t.TempDir()
	batteryDir := filepath.Join(tmpDir, "BAT0")

	if err := os.MkdirAll(batteryDir, 0755); err != nil {
		t.Fatalf("failed to create mock battery dir: %v", err)
	}

	os.WriteFile(filepath.Join(batteryDir, "status"), []byte("Discharging"), 0644)
	os.WriteFile(filepath.Join(batteryDir, "capacity"), []byte("not-a-number"), 0644)

	c := NewBatteryCollectorWithPath(batteryDir)

	ctx := context.Background()
	_, err := c.Collect(ctx)

	if err == nil {
		t.Error("expected error for invalid capacity value")
	}
}
