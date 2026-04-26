package renderer

import (
	"context"
	"testing"

	"gittea.kittel.dev/marco/dynamic_background/internal/domain"
)

func TestMockRenderer_Render(t *testing.T) {
	r := &MockRenderer{}

	ctx := context.Background()
	rc := NewRenderContext(MockBounds)

	err := r.Render(ctx, rc)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !r.RenderCalled {
		t.Error("expected Render to be called")
	}
}

func TestMockRenderer_Render_Error(t *testing.T) {
	expectedErr := &domain.CollectorError{Err: "render failed"}
	r := &MockRenderer{RenderErr: expectedErr}

	ctx := context.Background()
	rc := NewRenderContext(MockBounds)

	err := r.Render(ctx, rc)

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if err.Error() != "render failed" {
		t.Errorf("expected error 'render failed', got %q", err.Error())
	}
}

func TestMockRenderer_Clear(t *testing.T) {
	r := &MockRenderer{}

	err := r.Clear()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !r.ClearCalled {
		t.Error("expected Clear to be called")
	}
}
