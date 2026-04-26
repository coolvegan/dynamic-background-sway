package renderer

import (
	"context"
	"image"
)

// Renderer defines the contract for drawing widgets to the screen.
//
// WHY: Rendering-Implementierung (Cairo, OpenGL, etc.) soll austauschbar sein.
//      Interface erlaubt Mocking für Tests und verschiedene Backends.
//
// WHAT: Render() zeichnet dirty Widgets, Clear() löscht Screen.
// IMPACT: Ohne Interface wäre Code an spezifische Rendering-Bibliothek gebunden;
//       kein Testing möglich; kein Wechsel zwischen Cairo/OpenGL.
type Renderer interface {
	// Render draws dirty widgets from the render context.
	Render(ctx context.Context, rc *RenderContext) error

	// Clear clears the entire screen.
	Clear() error
}

// MockRenderer is a test double for the Renderer interface.
//
// WHY: Ermöglicht Testing von Application-Logik ohne echtes Rendering.
// WHAT: Trackt ob Render/Clear aufgerufen wurden, kann Error simulieren.
// IMPACT: Ohne MockRenderer wären Tests von Rendering-Logik nicht möglich.
type MockRenderer struct {
	RenderCalled bool
	ClearCalled  bool
	RenderErr    error
	ClearErr     error
}

// MockBounds is a default screen size for testing.
var MockBounds = image.Rect(0, 0, 1920, 1080)

func (m *MockRenderer) Render(ctx context.Context, rc *RenderContext) error {
	m.RenderCalled = true
	return m.RenderErr
}

func (m *MockRenderer) Clear() error {
	m.ClearCalled = true
	return m.ClearErr
}

// Ensure MockRenderer implements Renderer at compile time.
var _ Renderer = (*MockRenderer)(nil)
