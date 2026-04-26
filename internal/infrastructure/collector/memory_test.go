package collector

import (
	"context"
	"strings"
	"testing"

	"gittea.kittel.dev/marco/dynamic_background/internal/domain"
)

func TestParseMemInfo(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    MemInfo
		wantErr bool
	}{
		{
			name: "valid meminfo",
			input: `MemTotal:       16384000 kB
MemFree:         2048000 kB
MemAvailable:    4096000 kB
Buffers:          512000 kB
Cached:          3072000 kB`,
			want: MemInfo{
				Total:     16384000,
				Free:      2048000,
				Available: 4096000,
				Buffers:   512000,
				Cached:    3072000,
			},
			wantErr: false,
		},
		{
			name:    "empty input",
			input:   "",
			wantErr: true,
		},
		{
			name: "missing MemTotal",
			input: `MemFree:         2048000 kB
MemAvailable:    4096000 kB`,
			wantErr: true,
		},
		{
			name: "invalid number",
			input: `MemTotal:       notanumber kB
MemFree:         2048000 kB
MemAvailable:    4096000 kB`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseMemInfo(strings.NewReader(tt.input))

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got != tt.want {
				t.Errorf("expected %+v, got %+v", tt.want, got)
			}
		})
	}
}

func TestCalculateMemoryPercent(t *testing.T) {
	tests := []struct {
		name string
		info MemInfo
		want float64
	}{
		{
			name: "75 percent used",
			info: MemInfo{Total: 1000, Available: 250},
			want: 75.0,
		},
		{
			name: "0 percent used",
			info: MemInfo{Total: 1000, Available: 1000},
			want: 0.0,
		},
		{
			name: "100 percent used",
			info: MemInfo{Total: 1000, Available: 0},
			want: 100.0,
		},
		{
			name: "zero total (avoid division by zero)",
			info: MemInfo{Total: 0, Available: 0},
			want: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calculateMemoryPercent(tt.info)

			if got != tt.want {
				t.Errorf("expected %.1f%%, got %.1f%%", tt.want, got)
			}
		})
	}
}

func TestMemoryCollector_Collect(t *testing.T) {
	input := `MemTotal:       16384000 kB
MemFree:         2048000 kB
MemAvailable:    4096000 kB
Buffers:          512000 kB
Cached:          3072000 kB`

	c := NewMemoryCollectorWithReader(strings.NewReader(input))

	ctx := context.Background()
	data, err := c.Collect(ctx)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// (16384000 - 4096000) / 16384000 * 100 = 75%
	if !strings.Contains(data.Value, "75") {
		t.Errorf("expected value containing '75', got %q", data.Value)
	}

	if data.NumericValue != 75.0 {
		t.Errorf("expected numeric value 75.0, got %.1f", data.NumericValue)
	}
}

func TestMemoryCollector_Collect_ReadError(t *testing.T) {
	c := NewMemoryCollectorWithReader(&errorReader{})

	ctx := context.Background()
	_, err := c.Collect(ctx)

	if err == nil {
		t.Error("expected error from failing reader")
	}
}

// errorReader always returns an error on Read
type errorReader struct{}

func (e *errorReader) Read(p []byte) (n int, err error) {
	return 0, &domain.CollectorError{Err: "read error"}
}
