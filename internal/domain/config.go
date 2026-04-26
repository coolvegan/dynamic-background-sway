// Package domain defines the core business entities for dynamic_background.
//
// This file (config.go) defines the Config domain entity — the root configuration
// structure that describes the entire application state: which widgets to display,
// how the background should look, and whether the API server is enabled.
//
// Why it exists:
//   Config is the single source of truth for what the application should do.
//   It is loaded from YAML at startup, can be hot-reloaded via the API, and
//   is referenced by every layer (WidgetManager reads widgets, Renderer reads
//   background config, API server reads port settings).
//
// How it connects:
//   - Loaded by config.LoadConfig() from YAML into domain.Config
//   - Passed to NewOrchestrator() which creates WidgetManager from cfg.Widgets
//   - Background config passed to WaylandRenderer for drawBackground()
//   - API config controls whether Server.Start() listens on a port
//   - NewConfig() validates everything before the app starts
//
// Key concept: Config is mutable at runtime. The API can replace cfg.Widgets
// and cfg.Background while the app is running. The orchestrator picks up
// changes on the next render cycle.
package domain

import (
	"errors"
	"fmt"
)

// BackgroundType defines the rendering mode for the background.
type BackgroundType string

const (
	BackgroundTypeSolid     BackgroundType = "solid"
	BackgroundTypeGradient  BackgroundType = "gradient"
	BackgroundTypeImage     BackgroundType = "image"
	BackgroundTypeAnimated  BackgroundType = "animated"
)

// IsValid checks if the background type is supported.
func (bt BackgroundType) IsValid() bool {
	switch bt {
	case BackgroundTypeSolid, BackgroundTypeGradient, BackgroundTypeImage, BackgroundTypeAnimated:
		return true
	}
	return false
}

// BackgroundConfig holds settings for the background rendering.
type BackgroundConfig struct {
	Type      BackgroundType
	Colors    []string
	ImagePath string
}

// APIConfig holds settings for the HTTP/WebSocket API server.
type APIConfig struct {
	Enabled   bool
	Port      int
	WebSocket bool
}

// Config is the root domain entity for application configuration.
// WHY: Centralizes all settings; validates consistency before application starts.
// WHAT: Contains widgets, background settings, and API configuration.
// IMPACT: Without Config, there is no way to define what the background should display
// or how it should behave; the application would have no instructions.
type Config struct {
	Widgets    []*Widget
	Background BackgroundConfig
	API        APIConfig
}

// NewConfig creates a validated Config instance.
// WHY: Ensures configuration is consistent before the application uses it.
// WHAT: Validates widgets, background type, colors, and API port.
// IMPACT: Without validation, invalid configs could cause runtime panics or silent failures.
func NewConfig(cfg Config) (*Config, error) {
	if cfg.Widgets == nil {
		cfg.Widgets = []*Widget{}
	}
	if cfg.Background.Type == "" {
		return nil, errors.New("background type must not be empty")
	}
	if !cfg.Background.Type.IsValid() {
		return nil, fmt.Errorf("invalid background type: %s", cfg.Background.Type)
	}
	if cfg.Background.Type == BackgroundTypeGradient && len(cfg.Background.Colors) < 2 {
		return nil, errors.New("gradient background requires at least 2 colors")
	}
	if cfg.API.Enabled {
		if cfg.API.Port < 1 || cfg.API.Port > 65535 {
			return nil, errors.New("API port must be between 1 and 65535")
		}
	}

	return &Config{
		Widgets:    cfg.Widgets,
		Background: cfg.Background,
		API:        cfg.API,
	}, nil
}

// AddWidget appends a widget to the configuration.
func (c *Config) AddWidget(w *Widget) {
	c.Widgets = append(c.Widgets, w)
}

// RemoveWidget removes a widget by index.
func (c *Config) RemoveWidget(index int) {
	if index < 0 || index >= len(c.Widgets) {
		return
	}
	c.Widgets = append(c.Widgets[:index], c.Widgets[index+1:]...)
}

// FindWidgetByType returns the first widget matching the given type.
func (c *Config) FindWidgetByType(wt WidgetType) *Widget {
	for _, w := range c.Widgets {
		if w.Type == wt {
			return w
		}
	}
	return nil
}
