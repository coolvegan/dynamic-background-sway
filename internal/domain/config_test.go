package domain

import (
	"testing"
	"time"
)

func TestNewConfig(t *testing.T) {
	widgets := []*Widget{
		mustNewWidget(WidgetTypeClock, Position{0, 0}, Size{100, 50}, time.Second),
	}

	cfg, err := NewConfig(Config{
		Widgets: widgets,
		Background: BackgroundConfig{
			Type:   "gradient",
			Colors: []string{"#1a1a2e", "#16213e"},
		},
		API: APIConfig{
			Enabled: true,
			Port:    8080,
		},
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(cfg.Widgets) != 1 {
		t.Errorf("expected 1 widget, got %d", len(cfg.Widgets))
	}
	if cfg.Background.Type != "gradient" {
		t.Errorf("expected background type 'gradient', got %q", cfg.Background.Type)
	}
	if !cfg.API.Enabled {
		t.Error("expected API to be enabled")
	}
}

func TestConfig_Validation(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr string
	}{
		{
			name: "nil widgets",
			cfg: Config{
				Widgets: nil,
				Background: BackgroundConfig{
					Type: BackgroundTypeSolid,
				},
			},
			wantErr: "",
		},
		{
			name: "empty background type",
			cfg: Config{
				Widgets: []*Widget{},
				Background: BackgroundConfig{
					Type: "",
				},
			},
			wantErr: "background type must not be empty",
		},
		{
			name: "invalid background type",
			cfg: Config{
				Widgets: []*Widget{},
				Background: BackgroundConfig{
					Type: "invalid",
				},
			},
			wantErr: "invalid background type: invalid",
		},
		{
			name: "negative port",
			cfg: Config{
				Widgets: []*Widget{},
				Background: BackgroundConfig{
					Type: "solid",
				},
				API: APIConfig{
					Enabled: true,
					Port:    -1,
				},
			},
			wantErr: "API port must be between 1 and 65535",
		},
		{
			name: "port too high",
			cfg: Config{
				Widgets: []*Widget{},
				Background: BackgroundConfig{
					Type: "solid",
				},
				API: APIConfig{
					Enabled: true,
					Port:    70000,
				},
			},
			wantErr: "API port must be between 1 and 65535",
		},
		{
			name: "gradient without colors",
			cfg: Config{
				Widgets: []*Widget{},
				Background: BackgroundConfig{
					Type:   "gradient",
					Colors: []string{},
				},
			},
			wantErr: "gradient background requires at least 2 colors",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewConfig(tt.cfg)
			if tt.wantErr == "" {
				if err != nil {
					t.Errorf("expected no error, got %q", err.Error())
				}
				return
			}
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if err.Error() != tt.wantErr {
				t.Errorf("expected error %q, got %q", tt.wantErr, err.Error())
			}
		})
	}
}

func TestConfig_AddWidget(t *testing.T) {
	cfg, err := NewConfig(Config{
		Widgets: []*Widget{},
		Background: BackgroundConfig{
			Type: "solid",
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	w := mustNewWidget(WidgetTypeCPU, Position{0, 0}, Size{100, 50}, time.Second)
	cfg.AddWidget(w)

	if len(cfg.Widgets) != 1 {
		t.Errorf("expected 1 widget after add, got %d", len(cfg.Widgets))
	}
}

func TestConfig_RemoveWidget(t *testing.T) {
	w1 := mustNewWidget(WidgetTypeCPU, Position{0, 0}, Size{100, 50}, time.Second)
	w2 := mustNewWidget(WidgetTypeClock, Position{100, 0}, Size{100, 50}, time.Second)

	cfg, err := NewConfig(Config{
		Widgets: []*Widget{w1, w2},
		Background: BackgroundConfig{
			Type: "solid",
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cfg.RemoveWidget(0)

	if len(cfg.Widgets) != 1 {
		t.Errorf("expected 1 widget after remove, got %d", len(cfg.Widgets))
	}
	if cfg.Widgets[0] != w2 {
		t.Error("expected second widget to remain")
	}
}

func TestConfig_FindWidgetByType(t *testing.T) {
	w1 := mustNewWidget(WidgetTypeCPU, Position{0, 0}, Size{100, 50}, time.Second)
	w2 := mustNewWidget(WidgetTypeClock, Position{100, 0}, Size{100, 50}, time.Second)

	cfg, err := NewConfig(Config{
		Widgets: []*Widget{w1, w2},
		Background: BackgroundConfig{
			Type: "solid",
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found := cfg.FindWidgetByType(WidgetTypeClock)
	if found == nil {
		t.Fatal("expected to find clock widget")
	}
	if found.Type != WidgetTypeClock {
		t.Errorf("expected type clock, got %s", found.Type)
	}

	notFound := cfg.FindWidgetByType(WidgetTypeBattery)
	if notFound != nil {
		t.Error("expected nil for non-existent widget type")
	}
}

func mustNewWidget(wt WidgetType, pos Position, size Size, interval time.Duration) *Widget {
	w, err := NewWidget(wt, pos, size, Style{}, interval)
	if err != nil {
		panic(err)
	}
	return w
}
