package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadConfig(t *testing.T) {
	content := `
background:
  type: gradient
  colors:
    - "#1a1a2e"
    - "#16213e"

widgets:
  - type: clock
    position:
      x: 20
      y: 20
    size:
      width: 200
      height: 50
    interval: 1s
    style:
      font: "Monospace 12"
      color: "#ffffff"

  - type: cpu
    position:
      x: 20
      y: 80
    size:
      width: 200
      height: 50
    interval: 2s

api:
  enabled: true
  port: 8080
  websocket: true
`

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	err := os.WriteFile(configPath, []byte(content), 0644)
	if err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Background.Type != "gradient" {
		t.Errorf("expected background type 'gradient', got %q", cfg.Background.Type)
	}

	if len(cfg.Background.Colors) != 2 {
		t.Errorf("expected 2 background colors, got %d", len(cfg.Background.Colors))
	}

	if len(cfg.Widgets) != 2 {
		t.Fatalf("expected 2 widgets, got %d", len(cfg.Widgets))
	}

	if cfg.Widgets[0].Type != "clock" {
		t.Errorf("expected first widget type 'clock', got %q", cfg.Widgets[0].Type)
	}

	if cfg.Widgets[0].Interval != time.Second {
		t.Errorf("expected clock interval 1s, got %v", cfg.Widgets[0].Interval)
	}

	if cfg.Widgets[1].Type != "cpu" {
		t.Errorf("expected second widget type 'cpu', got %q", cfg.Widgets[1].Type)
	}

	if cfg.Widgets[1].Interval != 2*time.Second {
		t.Errorf("expected cpu interval 2s, got %v", cfg.Widgets[1].Interval)
	}

	if !cfg.API.Enabled {
		t.Error("expected API to be enabled")
	}

	if cfg.API.Port != 8080 {
		t.Errorf("expected API port 8080, got %d", cfg.API.Port)
	}

	if !cfg.API.WebSocket {
		t.Error("expected WebSocket to be enabled")
	}
}

func TestLoadConfig_FileNotFound(t *testing.T) {
	_, err := LoadConfig("/nonexistent/config.yaml")

	if err == nil {
		t.Error("expected error for non-existent file")
	}
}

func TestLoadConfig_InvalidYAML(t *testing.T) {
	content := `
background:
  type: [invalid yaml
  colors:
    - "#1a1a2e"
`

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	err := os.WriteFile(configPath, []byte(content), 0644)
	if err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	_, err = LoadConfig(configPath)

	if err == nil {
		t.Error("expected error for invalid YAML")
	}
}

func TestLoadConfig_Defaults(t *testing.T) {
	content := `
background:
  type: solid

widgets: []
`

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	err := os.WriteFile(configPath, []byte(content), 0644)
	if err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// API should be disabled by default
	if cfg.API.Enabled {
		t.Error("expected API to be disabled by default")
	}

	// Default port should be 0 (not configured)
	if cfg.API.Port != 0 {
		t.Errorf("expected default API port 0, got %d", cfg.API.Port)
	}
}

func TestLoadConfig_CustomWidget(t *testing.T) {
	content := `
background:
  type: solid

widgets:
  - type: custom
    position:
      x: 0
      y: 0
    size:
      width: 100
      height: 50
    interval: 30s
    command: "uptime -p"
`

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	err := os.WriteFile(configPath, []byte(content), 0644)
	if err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(cfg.Widgets) != 1 {
		t.Fatalf("expected 1 widget, got %d", len(cfg.Widgets))
	}

	if cfg.Widgets[0].Type != "custom" {
		t.Errorf("expected widget type 'custom', got %q", cfg.Widgets[0].Type)
	}

	if cfg.Widgets[0].CustomCommand != "uptime -p" {
		t.Errorf("expected custom command 'uptime -p', got %q", cfg.Widgets[0].CustomCommand)
	}

	if cfg.Widgets[0].Interval != 30*time.Second {
		t.Errorf("expected interval 30s, got %v", cfg.Widgets[0].Interval)
	}
}

func TestLoadConfig_ImageBackground(t *testing.T) {
	content := `
background:
  type: image
  image_path: "/path/to/image.png"

widgets: []
`

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	err := os.WriteFile(configPath, []byte(content), 0644)
	if err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Background.Type != "image" {
		t.Errorf("expected background type 'image', got %q", cfg.Background.Type)
	}

	if cfg.Background.ImagePath != "/path/to/image.png" {
		t.Errorf("expected image path '/path/to/image.png', got %q", cfg.Background.ImagePath)
	}
}
