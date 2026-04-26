package collector

import (
	"context"
	"testing"
	"time"
)

func TestClockCollector_Collect(t *testing.T) {
	tests := []struct {
		name   string
		format string
	}{
		{
			name:   "default format",
			format: "",
		},
		{
			name:   "custom format HH:MM",
			format: "15:04",
		},
		{
			name:   "custom format with date",
			format: "2006-01-02 15:04:05",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var c *ClockCollector
			if tt.format == "" {
				c = NewClockCollector()
			} else {
				c = NewClockCollectorWithFormat(tt.format)
			}

			ctx := context.Background()
			data, err := c.Collect(ctx)

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if data.Value == "" {
				t.Error("expected non-empty time string")
			}
		})
	}
}

func TestClockCollector_Collect_WithFixedTime(t *testing.T) {
	fixedTime := time.Date(2024, 1, 15, 14, 30, 45, 0, time.UTC)
	c := NewClockCollectorWithTimeSource(func() time.Time { return fixedTime })

	ctx := context.Background()
	data, err := c.Collect(ctx)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "2024-01-15 14:30:45"
	if data.Value != expected {
		t.Errorf("expected %q, got %q", expected, data.Value)
	}
}

func TestClockCollector_Collect_CustomFormatWithFixedTime(t *testing.T) {
	fixedTime := time.Date(2024, 1, 15, 14, 30, 45, 0, time.UTC)
	c := &ClockCollector{
		format:    "15:04",
		timeSource: func() time.Time { return fixedTime },
	}

	ctx := context.Background()
	data, err := c.Collect(ctx)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "14:30"
	if data.Value != expected {
		t.Errorf("expected %q, got %q", expected, data.Value)
	}
}

func TestClockCollector_Collect_ErrorHandling(t *testing.T) {
	c := &ClockCollector{
		format:     "15:04",
		timeSource: func() time.Time { return time.Time{} },
	}

	ctx := context.Background()
	data, err := c.Collect(ctx)

	// Zero time should still produce output (not an error)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if data.Value == "" {
		t.Error("expected some output even for zero time")
	}
}
