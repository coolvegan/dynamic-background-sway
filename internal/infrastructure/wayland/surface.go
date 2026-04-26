package wayland

import (
	"context"
	"errors"
	"image"
	"sync"
)

// SurfaceState tracks the lifecycle of a Wayland surface.
type SurfaceState int

const (
	SurfaceStateInitialized SurfaceState = iota
	SurfaceStateRunning
	SurfaceStateStopped
)

// Output represents a physical display connected to the system.
type Output struct {
	Name   string
	Width  int
	Height int
	Scale  int
}

// Surface defines the contract for Wayland background surface management.
//
// WHY: Abstrahiert Wayland-spezifische Operationen; ermöglicht austauschbare
//      Implementationen (echtes Wayland, Mock, X11-Fallback).
//
// WHAT: Display-Verbindung, Surface-Lifecycle, Buffer-Management, Frame-Callbacks.
// IMPACT: Ohne Interface wäre Code fest an libwayland gebunden; nicht testbar;
//       kein Wechsel zu anderem Backend möglich.
type Surface interface {
	// Connect establishes connection to the Wayland compositor.
	Connect(ctx context.Context) error

	// Disconnect closes the Wayland connection gracefully.
	Disconnect() error

	// Outputs returns all connected displays.
	Outputs() []Output

	// CreateSurface creates a background surface for the given output.
	CreateSurface(output Output) error

	// Buffer returns the shared memory buffer for rendering.
	Buffer() []byte

	// Bounds returns the dimensions of the surface.
	Bounds() image.Rectangle

	// Commit sends the buffer to the compositor for display.
	Commit() error

	// State returns the current surface state.
	State() SurfaceState

	// SetFrameCallback registers a callback invoked after each frame.
	SetFrameCallback(fn func())
}

// Config holds settings for Wayland surface creation.
type Config struct {
	Layer            string // "background", "bottom", "top", "overlay"
	KeyboardInteract bool   // Whether surface receives keyboard input
	ExclusiveZone    int    // Reserved screen area
}

// DefaultConfig returns a Config suitable for background use.
func DefaultConfig() Config {
	return Config{
		Layer:            "background",
		KeyboardInteract: false,
		ExclusiveZone:    0,
	}
}

// Validate checks if the configuration is valid.
func (c Config) Validate() error {
	switch c.Layer {
	case "background", "bottom", "top", "overlay":
		// Valid layer
	default:
		return errors.New("invalid layer: must be background, bottom, top, or overlay")
	}
	return nil
}

// MockSurface is a test double for the Surface interface.
//
// WHY: Ermöglicht Testing von Orchestrator und Renderer ohne echtes Wayland.
// WHAT: Simuliert Surface-Lifecycle, speichert Buffer im RAM, trackt Aufrufe.
// IMPACT: Ohne MockSurface wären Integrationstests nur mit laufendem Wayland möglich.
type MockSurface struct {
	mu            sync.RWMutex
	state         SurfaceState
	outputs       []Output
	bounds        image.Rectangle
	buffer        []byte
	frameCallback func()

	ConnectCalled    bool
	DisconnectCalled bool
	CommitCalled     int
	SurfacesCreated  int
}

// NewMockSurface creates a MockSurface with default 1080p dimensions.
func NewMockSurface() *MockSurface {
	bounds := image.Rect(0, 0, 1920, 1080)
	return &MockSurface{
		state:  SurfaceStateInitialized,
		bounds: bounds,
		buffer: make([]byte, bounds.Dx()*bounds.Dy()*4), // 4 bytes per pixel (RGBA)
		outputs: []Output{
			{Name: "mock-output-1", Width: 1920, Height: 1080, Scale: 1},
		},
	}
}

func (m *MockSurface) Connect(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ConnectCalled = true
	m.state = SurfaceStateRunning
	return nil
}

func (m *MockSurface) Disconnect() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.DisconnectCalled = true
	m.state = SurfaceStateStopped
	return nil
}

func (m *MockSurface) Outputs() []Output {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.outputs
}

func (m *MockSurface) CreateSurface(output Output) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.SurfacesCreated++
	m.bounds = image.Rect(0, 0, output.Width, output.Height)
	m.buffer = make([]byte, output.Width*output.Height*4)
	return nil
}

func (m *MockSurface) Buffer() []byte {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.buffer
}

func (m *MockSurface) Bounds() image.Rectangle {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.bounds
}

func (m *MockSurface) Commit() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.CommitCalled++
	return nil
}

func (m *MockSurface) State() SurfaceState {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.state
}

func (m *MockSurface) SetFrameCallback(fn func()) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.frameCallback = fn
}

// TriggerFrameCallback simulates a compositor frame completion.
func (m *MockSurface) TriggerFrameCallback() {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.frameCallback != nil {
		m.frameCallback()
	}
}

// Ensure MockSurface implements Surface at compile time.
var _ Surface = (*MockSurface)(nil)
