package domain

import (
	"context"
	"testing"
)

func TestCollectorData(t *testing.T) {
	tests := []struct {
		name    string
		data    CollectorData
		wantErr bool
	}{
		{
			name:    "valid string data",
			data:    CollectorData{Value: "42%"},
			wantErr: false,
		},
		{
			name:    "valid numeric data",
			data:    CollectorData{NumericValue: 42.5},
			wantErr: false,
		},
		{
			name:    "empty data is valid",
			data:    CollectorData{},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.data.Value == "" && tt.data.NumericValue == 0 && tt.data.Error == nil {
				return
			}
		})
	}
}

func TestMockCollector(t *testing.T) {
	expected := CollectorData{Value: "test", NumericValue: 1.0}
	mc := &MockCollector{Data: expected}

	ctx := context.Background()
	result, err := mc.Collect(ctx)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result.Value != expected.Value {
		t.Errorf("expected value %q, got %q", expected.Value, result.Value)
	}
	if result.NumericValue != expected.NumericValue {
		t.Errorf("expected numeric value %v, got %v", expected.NumericValue, result.NumericValue)
	}
}

func TestMockCollector_Error(t *testing.T) {
	expectedErr := &CollectorError{Err: "simulated failure"}
	mc := &MockCollector{Err: expectedErr}

	ctx := context.Background()
	_, err := mc.Collect(ctx)

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "simulated failure" {
		t.Errorf("expected error message 'simulated failure', got %q", err.Error())
	}
}
