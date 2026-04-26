package domain

import (
	"errors"
	"time"
)

// WidgetType defines the kind of data a widget displays.
type WidgetType string

const (
	WidgetTypeCPU         WidgetType = "cpu"
	WidgetTypeMemory      WidgetType = "memory"
	WidgetTypeDisk        WidgetType = "disk"
	WidgetTypeNetwork     WidgetType = "network"
	WidgetTypeBattery     WidgetType = "battery"
	WidgetTypeClock       WidgetType = "clock"
	WidgetTypeUptime      WidgetType = "uptime"
	WidgetTypeTemperature WidgetType = "temperature"
	WidgetTypeCustom      WidgetType = "custom"
)

// IsValid checks if the widget type is a recognized built-in type.
func (wt WidgetType) IsValid() bool {
	switch wt {
	case WidgetTypeCPU, WidgetTypeMemory, WidgetTypeDisk, WidgetTypeNetwork,
		WidgetTypeBattery, WidgetTypeClock, WidgetTypeUptime,
		WidgetTypeTemperature, WidgetTypeCustom:
		return true
	}
	return false
}

// Position represents the top-left coordinate of a widget on the screen.
type Position struct {
	X int
	Y int
}

// Size represents the dimensions of a widget.
type Size struct {
	Width  int
	Height int
}

// Bounds returns the rectangular area occupied by the widget.
type Bounds struct {
	X      int
	Y      int
	Width  int
	Height int
}

// Style holds visual presentation settings for a widget.
type Style struct {
	Font       string
	Color      string
	Background string
}

// Widget represents a single displayable unit on the background.
// It tracks its own dirty state to enable partial rendering.
type Widget struct {
	Type     WidgetType
	Position Position
	Size     Size
	Style    Style
	Interval time.Duration
	Dirty    bool

	// CustomCommand is used when Type is WidgetTypeCustom.
	CustomCommand string
}

// NewWidget creates a validated Widget instance.
// WHY: Widgets are the core domain entity; they must be valid before rendering.
// WHAT: Factory function that enforces business rules (valid type, non-zero size, positive interval).
// IMPACT: Without validation, invalid widgets could cause rendering crashes or invisible elements.
func NewWidget(widgetType WidgetType, position Position, size Size, style Style, interval time.Duration) (*Widget, error) {
	if !widgetType.IsValid() {
		return nil, errors.New("invalid widget type")
	}
	if size.Width <= 0 || size.Height <= 0 {
		return nil, errors.New("widget size must be positive")
	}
	if interval <= 0 {
		return nil, errors.New("widget interval must be positive")
	}

	return &Widget{
		Type:     widgetType,
		Position: position,
		Size:     size,
		Style:    style,
		Interval: interval,
		Dirty:    true,
	}, nil
}

// MarkClean resets the dirty flag after the widget has been rendered.
func (w *Widget) MarkClean() {
	w.Dirty = false
}

// MarkDirty sets the dirty flag to trigger a re-render on the next cycle.
func (w *Widget) MarkDirty() {
	w.Dirty = true
}

// Bounds returns the rectangular area this widget occupies.
func (w *Widget) Bounds() Bounds {
	return Bounds{
		X:      w.Position.X,
		Y:      w.Position.Y,
		Width:  w.Size.Width,
		Height: w.Size.Height,
	}
}
