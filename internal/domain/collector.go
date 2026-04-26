package domain

import (
	"context"
	"fmt"
)

// CollectorData holds the result of a data collection operation.
// WHY: Uniform data structure allows widgets to render any collector output.
// WHAT: Container for string value, numeric value, and optional error.
// IMPACT: Without CollectorData, each widget type would need its own data structure,
// making the rendering layer complex and tightly coupled to specific collectors.
type CollectorData struct {
	Value        string
	NumericValue float64
	Error        error
}

// Collector defines the contract for fetching system data.
// WHY: Decouples data sources from widgets; enables mocking and testing.
// WHAT: Single method interface that returns data for a given context.
// IMPACT: Without this interface, widgets would directly read from /proc or /sys,
// making testing impossible and preventing custom data sources.
type Collector interface {
	Collect(ctx context.Context) (CollectorData, error)
}

// CollectorError represents a collection failure with context.
type CollectorError struct {
	Err string
}

func (e *CollectorError) Error() string {
	return e.Err
}

// MockCollector is a test double for the Collector interface.
// WHY: Enables unit testing of widgets and application logic without real system calls.
// WHAT: Returns predefined data or error when Collect is called.
// IMPACT: Without MockCollector, tests would depend on actual system state,
// making them flaky and non-deterministic.
type MockCollector struct {
	Data CollectorData
	Err  error
}

func (m *MockCollector) Collect(ctx context.Context) (CollectorData, error) {
	if m.Err != nil {
		return CollectorData{}, m.Err
	}
	return m.Data, nil
}

// Ensure MockCollector implements Collector at compile time.
var _ Collector = (*MockCollector)(nil)

// CollectorFunc is an adapter to use ordinary functions as Collectors.
// WHY: Allows quick inline collectors without creating structs.
// WHAT: Function type that implements the Collector interface.
// IMPACT: Without CollectorFunc, every simple collector would need a full struct definition.
type CollectorFunc func(ctx context.Context) (CollectorData, error)

func (f CollectorFunc) Collect(ctx context.Context) (CollectorData, error) {
	return f(ctx)
}

var _ Collector = (CollectorFunc)(nil)

// NewStaticCollector creates a collector that always returns the same data.
// WHY: Useful for testing and for widgets that display constant data.
// WHAT: Returns a CollectorFunc that ignores context and returns fixed data.
// IMPACT: Without this, testing static rendering would require more complex mocks.
func NewStaticCollector(data CollectorData) Collector {
	return CollectorFunc(func(ctx context.Context) (CollectorData, error) {
		return data, nil
	})
}

// NewErrorCollector creates a collector that always returns an error.
// WHY: Tests error handling paths in widget rendering.
// WHAT: Returns a CollectorFunc that always fails with the given error.
// IMPACT: Without this, error scenarios would be harder to test deterministically.
func NewErrorCollector(err error) Collector {
	return CollectorFunc(func(ctx context.Context) (CollectorData, error) {
		return CollectorData{}, err
	})
}

// FormatCollectorData creates a human-readable string from collector data.
func FormatCollectorData(data CollectorData) string {
	if data.Error != nil {
		return fmt.Sprintf("error: %v", data.Error)
	}
	if data.Value != "" {
		return data.Value
	}
	return fmt.Sprintf("%.1f", data.NumericValue)
}
