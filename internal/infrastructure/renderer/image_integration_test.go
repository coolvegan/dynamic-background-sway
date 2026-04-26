package renderer

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"gittea.kittel.dev/marco/dynamic_background/internal/domain"
)

func TestImageRenderer_Render(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "output.png")

	config := domain.BackgroundConfig{
		Type:   domain.BackgroundTypeSolid,
		Colors: []string{"#1a1a2e"},
	}

	r := NewImageRenderer(100, 100, outputPath, config)

	ctx := context.Background()
	rc := NewRenderContext(MockBounds)

	w, _ := domain.NewWidget(domain.WidgetTypeClock, domain.Position{10, 10}, domain.Size{80, 40}, domain.Style{
		Color:      "#ffffff",
		Background: "rgba(0,0,0,0.5)",
	}, 1)
	w.Value = "12:00:00"

	rc.SetDirtyWidgets([]*domain.Widget{w})

	err := r.Render(ctx, rc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Error("expected output file to be created")
	}
}

func TestImageRenderer_Clear(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "output.png")

	config := domain.BackgroundConfig{
		Type:   domain.BackgroundTypeSolid,
		Colors: []string{"#ff0000"},
	}

	r := NewImageRenderer(100, 100, outputPath, config)

	err := r.Clear()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Error("expected output file to be created")
	}
}

func TestImageRenderer_GradientBackground(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "output.png")

	config := domain.BackgroundConfig{
		Type:   domain.BackgroundTypeGradient,
		Colors: []string{"#ff0000", "#0000ff"},
	}

	r := NewImageRenderer(100, 100, outputPath, config)

	err := r.Clear()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Error("expected output file to be created")
	}
}

func TestImageRenderer_MultipleWidgets(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "output.png")

	config := domain.BackgroundConfig{
		Type:   domain.BackgroundTypeSolid,
		Colors: []string{"#1a1a2e"},
	}

	r := NewImageRenderer(200, 100, outputPath, config)

	ctx := context.Background()
	rc := NewRenderContext(MockBounds)

	w1, _ := domain.NewWidget(domain.WidgetTypeCPU, domain.Position{10, 10}, domain.Size{90, 40}, domain.Style{Color: "#ffffff"}, 1)
	w1.Value = "50%"

	w2, _ := domain.NewWidget(domain.WidgetTypeMemory, domain.Position{110, 10}, domain.Size{90, 40}, domain.Style{Color: "#ffffff"}, 1)
	w2.Value = "8GB"

	rc.SetDirtyWidgets([]*domain.Widget{w1, w2})

	err := r.Render(ctx, rc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Error("expected output file to be created")
	}
}
