package collector

import (
	"context"
	"io"
	"strings"
	"testing"
	"time"
)

func TestParseNetDev(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    map[string]NetDevStats
		wantErr bool
	}{
		{
			name: "valid net/dev",
			input: `Inter-|   Receive                                                |  Transmit
 face |bytes    packets errs drop fifo frame compressed multicast|bytes    packets errs drop fifo colls carrier compressed
    lo: 1000 10 0 0 0 0 0 0 2000 20 0 0 0 0 0 0
  eth0: 500000 500 0 0 0 0 0 0 300000 300 0 0 0 0 0 0`,
			want: map[string]NetDevStats{
				"lo": {
					RxBytes: 1000, RxPackets: 10,
					TxBytes: 2000, TxPackets: 20,
				},
				"eth0": {
					RxBytes: 500000, RxPackets: 500,
					TxBytes: 300000, TxPackets: 300,
				},
			},
			wantErr: false,
		},
		{
			name:    "empty input",
			input:   "",
			wantErr: true,
		},
		{
			name: "skip header lines",
			input: `Inter-|   Receive                                                |  Transmit
 face |bytes    packets errs drop fifo frame compressed multicast|bytes    packets errs drop fifo colls carrier compressed`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseNetDev(strings.NewReader(tt.input))

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			for iface, wantStats := range tt.want {
				gotStats, exists := got[iface]
				if !exists {
					t.Errorf("missing interface %s", iface)
					continue
				}
				if gotStats != wantStats {
					t.Errorf("interface %s: expected %+v, got %+v", iface, wantStats, gotStats)
				}
			}
		})
	}
}

func TestCalculateNetworkSpeed(t *testing.T) {
	prev := NetDevStats{RxBytes: 1000, TxBytes: 2000}
	curr := NetDevStats{RxBytes: 3000, TxBytes: 5000}
	elapsed := 2 * time.Second

	rxSpeed, txSpeed := calculateNetworkSpeed(prev, curr, elapsed)

	// Rx: (3000 - 1000) / 2 = 1000 bytes/s
	// Tx: (5000 - 2000) / 2 = 1500 bytes/s
	if rxSpeed != 1000 {
		t.Errorf("expected rx speed 1000, got %.1f", rxSpeed)
	}
	if txSpeed != 1500 {
		t.Errorf("expected tx speed 1500, got %.1f", txSpeed)
	}
}

func TestCalculateNetworkSpeed_ZeroElapsed(t *testing.T) {
	prev := NetDevStats{RxBytes: 1000, TxBytes: 2000}
	curr := NetDevStats{RxBytes: 3000, TxBytes: 5000}

	rxSpeed, txSpeed := calculateNetworkSpeed(prev, curr, 0)

	if rxSpeed != 0 || txSpeed != 0 {
		t.Errorf("expected zero speed for zero elapsed time, got rx=%.1f tx=%.1f", rxSpeed, txSpeed)
	}
}

func TestNetworkCollector_Collect(t *testing.T) {
	first := `Inter-|   Receive                                                |  Transmit
 face |bytes    packets errs drop fifo frame compressed multicast|bytes    packets errs drop fifo colls carrier compressed
  eth0: 1000000 1000 0 0 0 0 0 0 500000 500 0 0 0 0 0 0`

	second := `Inter-|   Receive                                                |  Transmit
 face |bytes    packets errs drop fifo frame compressed multicast|bytes    packets errs drop fifo colls carrier compressed
  eth0: 2000000 2000 0 0 0 0 0 0 1000000 1000 0 0 0 0 0 0`

	callCount := 0
	opener := func() (io.Reader, error) {
		callCount++
		if callCount == 1 {
			return strings.NewReader(first), nil
		}
		return strings.NewReader(second), nil
	}

	c := &NetworkCollector{
		iface:  "eth0",
		opener: opener,
	}

	ctx := context.Background()

	// First call - stores snapshot
	_, _ = c.Collect(ctx)

	// Small delay to ensure different timestamp
	time.Sleep(10 * time.Millisecond)

	// Second call - calculates speed
	data, err := c.Collect(ctx)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if data.Value == "" {
		t.Error("expected non-empty network speed string")
	}
}

func TestNetworkCollector_Collect_InterfaceNotFound(t *testing.T) {
	input := `Inter-|   Receive                                                |  Transmit
 face |bytes    packets errs drop fifo frame compressed multicast|bytes    packets errs drop fifo colls carrier compressed
  eth0: 1000000 1000 0 0 0 0 0 0 500000 500 0 0 0 0 0 0`

	c := NewNetworkCollectorWithReader("nonexistent", strings.NewReader(input))

	ctx := context.Background()
	_, err := c.Collect(ctx)

	if err == nil {
		t.Error("expected error for non-existent interface")
	}
}
