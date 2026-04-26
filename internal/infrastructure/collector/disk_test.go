package collector

import (
	"context"
	"testing"
)

func TestDiskCollector_Collect(t *testing.T) {
	// Test with real filesystem - use /tmp which should exist
	c := NewDiskCollectorWithPath("/tmp")

	ctx := context.Background()
	data, err := c.Collect(ctx)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if data.Value == "" {
		t.Error("expected non-empty disk usage string")
	}

	if data.NumericValue < 0 || data.NumericValue > 100 {
		t.Errorf("expected percentage between 0-100, got %.1f", data.NumericValue)
	}
}

func TestDiskCollector_Collect_NonExistentPath(t *testing.T) {
	c := NewDiskCollectorWithPath("/nonexistent/path/that/does/not/exist")

	ctx := context.Background()
	_, err := c.Collect(ctx)

	if err == nil {
		t.Error("expected error for non-existent path")
	}
}

func TestDiskCollector_Collect_DefaultPath(t *testing.T) {
	c := NewDiskCollector()

	ctx := context.Background()
	data, err := c.Collect(ctx)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if data.Value == "" {
		t.Error("expected non-empty disk usage string")
	}
}

func TestCalculateDiskPercent(t *testing.T) {
	tests := []struct {
		name  string
		total uint64
		free  uint64
		want  float64
	}{
		{
			name:  "50 percent used",
			total: 1000,
			free:  500,
			want:  50.0,
		},
		{
			name:  "0 percent used",
			total: 1000,
			free:  1000,
			want:  0.0,
		},
		{
			name:  "100 percent used",
			total: 1000,
			free:  0,
			want:  100.0,
		},
		{
			name:  "zero total (avoid division by zero)",
			total: 0,
			free:  0,
			want:  0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calculateDiskPercent(tt.total, tt.free)

			if got != tt.want {
				t.Errorf("expected %.1f%%, got %.1f%%", tt.want, got)
			}
		})
	}
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		name     string
		bytes    uint64
		expected string
	}{
		{
			name:     "bytes",
			bytes:    512,
			expected: "512 B",
		},
		{
			name:     "kilobytes",
			bytes:    1024,
			expected: "1.0 KB",
		},
		{
			name:     "megabytes",
			bytes:    1024 * 1024,
			expected: "1.0 MB",
		},
		{
			name:     "gigabytes",
			bytes:    1024 * 1024 * 1024,
			expected: "1.0 GB",
		},
		{
			name:     "terabytes",
			bytes:    1024 * 1024 * 1024 * 1024,
			expected: "1.0 TB",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatBytes(tt.bytes)
			if got != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, got)
			}
		})
	}
}
