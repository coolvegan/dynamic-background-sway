package domain

import (
	"testing"
	"time"
)

func TestNewWidget(t *testing.T) {
	tests := []struct {
		name        string
		widgetType  WidgetType
		position    Position
		size        Size
		style       Style
		interval    time.Duration
		expectError bool
	}{
		{
			name:        "valid clock widget",
			widgetType:  WidgetTypeClock,
			position:    Position{X: 10, Y: 20},
			size:        Size{Width: 100, Height: 50},
			style:       Style{Font: "Monospace 12", Color: "#ffffff"},
			interval:    time.Second,
			expectError: false,
		},
		{
			name:        "zero size fails",
			widgetType:  WidgetTypeClock,
			position:    Position{X: 0, Y: 0},
			size:        Size{Width: 0, Height: 0},
			style:       Style{Font: "Monospace 12", Color: "#ffffff"},
			interval:    time.Second,
			expectError: true,
		},
		{
			name:        "negative interval fails",
			widgetType:  WidgetTypeCPU,
			position:    Position{X: 10, Y: 20},
			size:        Size{Width: 100, Height: 50},
			style:       Style{Font: "Monospace 12", Color: "#ffffff"},
			interval:    -1 * time.Second,
			expectError: true,
		},
		{
			name:        "unknown widget type fails",
			widgetType:  WidgetType("unknown"),
			position:    Position{X: 10, Y: 20},
			size:        Size{Width: 100, Height: 50},
			style:       Style{Font: "Monospace 12", Color: "#ffffff"},
			interval:    time.Second,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w, err := NewWidget(tt.widgetType, tt.position, tt.size, tt.style, tt.interval)

			if tt.expectError {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if w.Type != tt.widgetType {
				t.Errorf("expected type %s, got %s", tt.widgetType, w.Type)
			}
			if w.Position != tt.position {
				t.Errorf("expected position %+v, got %+v", tt.position, w.Position)
			}
			if w.Size != tt.size {
				t.Errorf("expected size %+v, got %+v", tt.size, w.Size)
			}
			if w.Interval != tt.interval {
				t.Errorf("expected interval %v, got %v", tt.interval, w.Interval)
			}
			if !w.Dirty {
				t.Error("expected new widget to be dirty")
			}
		})
	}
}

func TestWidget_MarkClean(t *testing.T) {
	w, err := NewWidget(WidgetTypeClock, Position{X: 0, Y: 0}, Size{Width: 100, Height: 50}, Style{}, time.Second)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	w.MarkClean()

	if w.Dirty {
		t.Error("expected widget to be clean after MarkClean()")
	}
}

func TestWidget_MarkDirty(t *testing.T) {
	w, err := NewWidget(WidgetTypeClock, Position{X: 0, Y: 0}, Size{Width: 100, Height: 50}, Style{}, time.Second)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	w.MarkClean()
	w.MarkDirty()

	if !w.Dirty {
		t.Error("expected widget to be dirty after MarkDirty()")
	}
}

func TestWidget_Bounds(t *testing.T) {
	w, err := NewWidget(WidgetTypeClock, Position{X: 10, Y: 20}, Size{Width: 100, Height: 50}, Style{}, time.Second)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	bounds := w.Bounds()

	if bounds.X != 10 {
		t.Errorf("expected bounds X=10, got %d", bounds.X)
	}
	if bounds.Y != 20 {
		t.Errorf("expected bounds Y=20, got %d", bounds.Y)
	}
	if bounds.Width != 100 {
		t.Errorf("expected bounds Width=100, got %d", bounds.Width)
	}
	if bounds.Height != 50 {
		t.Errorf("expected bounds Height=50, got %d", bounds.Height)
	}
}

func TestValidWidgetTypes(t *testing.T) {
	validTypes := []WidgetType{
		WidgetTypeCPU,
		WidgetTypeMemory,
		WidgetTypeDisk,
		WidgetTypeNetwork,
		WidgetTypeBattery,
		WidgetTypeClock,
		WidgetTypeUptime,
		WidgetTypeTemperature,
		WidgetTypeCustom,
	}

	for _, wt := range validTypes {
		if !wt.IsValid() {
			t.Errorf("expected %s to be valid", wt)
		}
	}

	if WidgetType("invalid").IsValid() {
		t.Error("expected 'invalid' to be not valid")
	}
}
