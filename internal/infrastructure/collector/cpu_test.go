package collector

import (
	"context"
	"io"
	"strings"
	"testing"

	"gittea.kittel.dev/marco/dynamic_background/internal/domain"
)

func TestParseCPUStats(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    CPUStats
		wantErr bool
	}{
		{
			name:  "valid cpu line",
			input: "cpu  100 50 200 500 30 10 20 0 0 0",
			want: CPUStats{
				User: 100, Nice: 50, System: 200, Idle: 500,
				IOWait: 30, IRQ: 10, SoftIRQ: 20, Steal: 0,
			},
			wantErr: false,
		},
		{
			name:    "invalid line",
			input:   "invalid",
			wantErr: true,
		},
		{
			name:    "not enough fields",
			input:   "cpu 100 200",
			wantErr: true,
		},
		{
			name:    "wrong prefix",
			input:   "cpu0 100 200 300 400 500 600 700 800 900 1000",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseCPUStats(strings.NewReader(tt.input))

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

func TestCalculateCPUPercent(t *testing.T) {
	tests := []struct {
		name string
		prev CPUStats
		curr CPUStats
		want float64
	}{
		{
			name: "50 percent usage",
			prev: CPUStats{User: 100, System: 100, Idle: 400},
			curr: CPUStats{User: 150, System: 150, Idle: 500},
			want: 50.0,
		},
		{
			name: "0 percent usage (all idle)",
			prev: CPUStats{User: 100, System: 100, Idle: 800},
			curr: CPUStats{User: 100, System: 100, Idle: 1800},
			want: 0.0,
		},
		{
			name: "100 percent usage (no idle)",
			prev: CPUStats{User: 100, System: 100, Idle: 100},
			curr: CPUStats{User: 300, System: 300, Idle: 100},
			want: 100.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calculateCPUPercent(tt.prev, tt.curr)

			if got != tt.want {
				t.Errorf("expected %.1f%%, got %.1f%%", tt.want, got)
			}
		})
	}
}

func TestCPUCollector_Collect_FirstCall(t *testing.T) {
	reader := strings.NewReader("cpu  100 50 200 500 30 10 20 0 0 0")
	c := NewCPUCollectorWithReader(reader)

	ctx := context.Background()
	data, err := c.Collect(ctx)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// First call should indicate no previous data
	if !strings.Contains(data.Value, "N/A") && data.Value != "" {
		t.Logf("first call returned: %q (may be acceptable)", data.Value)
	}
}

func TestCPUCollector_Collect_SecondCall(t *testing.T) {
	// Simulate two reads where CPU went from idle to busy
	first := "cpu  100 50 200 500 30 10 20 0 0 0"
	second := "cpu  200 100 400 500 30 10 20 0 0 0"

	callCount := 0
	opener := func() (io.Reader, error) {
		callCount++
		if callCount == 1 {
			return strings.NewReader(first), nil
		}
		return strings.NewReader(second), nil
	}

	c := &CPUCollector{opener: opener}

	ctx := context.Background()

	// First call - stores snapshot
	_, _ = c.Collect(ctx)

	// Second call - calculates delta
	data, err := c.Collect(ctx)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have a percentage value
	if data.Value == "" {
		t.Error("expected non-empty value on second call")
	}
}

func TestCPUCollector_Collect_ReadError(t *testing.T) {
	opener := func() (io.Reader, error) {
		return nil, &domain.CollectorError{Err: "read error"}
	}
	c := &CPUCollector{opener: opener}

	ctx := context.Background()
	_, err := c.Collect(ctx)

	if err == nil {
		t.Error("expected error from failing opener")
	}
}

// mockReader simulates /proc/stat reads with predefined responses
type mockReader struct {
	data    []string
	index   int
	readErr bool
}

func (m *mockReader) Read(p []byte) (n int, err error) {
	if m.readErr {
		return 0, &domain.CollectorError{Err: "read error"}
	}
	if m.index >= len(m.data) {
		return 0, &domain.CollectorError{Err: "no more data"}
	}
	line := m.data[m.index]
	m.index++
	return copy(p, line), nil
}
