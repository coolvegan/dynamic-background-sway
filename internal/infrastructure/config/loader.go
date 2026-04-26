package config

import (
	"fmt"
	"os"
	"time"

	"gittea.kittel.dev/marco/dynamic_background/internal/domain"
	"gopkg.in/yaml.v3"
)

// yamlConfig is the YAML representation of the configuration.
//
// WHY: YAML ist menschenlesbar; User schreibt Config-Dateien nicht Go-Code.
//      Diese Struktur mappt YAML auf Go-Types.
//
// WHAT: Spiegelung der domain.Config mit YAML-Tags.
// IMPACT: Ohne diese Struktur können wir YAML nicht parsen.
type yamlConfig struct {
	Background yamlBackground `yaml:"background"`
	Widgets    []yamlWidget   `yaml:"widgets"`
	Renderer   yamlRenderer   `yaml:"renderer"`
	API        yamlAPI        `yaml:"api"`
}

type yamlBackground struct {
	Type      string   `yaml:"type"`
	Colors    []string `yaml:"colors"`
	ImagePath string   `yaml:"image_path"`
}

type yamlRenderer struct {
	Type string `yaml:"type"`
}

type yamlWidget struct {
	Type        string        `yaml:"type"`
	Position    yamlPosition  `yaml:"position"`
	Size        yamlSize      `yaml:"size"`
	Style       yamlStyle     `yaml:"style"`
	Interval    string        `yaml:"interval"`
	CustomCommand string      `yaml:"command"`
}

type yamlPosition struct {
	X int `yaml:"x"`
	Y int `yaml:"y"`
}

type yamlSize struct {
	Width  int `yaml:"width"`
	Height int `yaml:"height"`
}

type yamlStyle struct {
	Font       string `yaml:"font"`
	Color      string `yaml:"color"`
	Background string `yaml:"background"`
}

type yamlAPI struct {
	Enabled   bool `yaml:"enabled"`
	Port      int  `yaml:"port"`
	WebSocket bool `yaml:"websocket"`
}

// LoadConfig reads and parses a YAML configuration file.
//
// WHY: User braucht Config-Datei statt Code-Änderung für neue Widgets.
//      YAML ist Standard für Konfiguration in Go-Projekten.
//
// WHAT: Liest Datei, parst YAML, validiert, erstellt domain.Config.
// IMPACT: Ohne LoadConfig müsste User Go-Code ändern für neue Widgets;
//       kein Hot-Reload möglich; schlechte Developer Experience.
func LoadConfig(path string) (*domain.Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	var yamlCfg yamlConfig
	err = yaml.Unmarshal(data, &yamlCfg)
	if err != nil {
		return nil, fmt.Errorf("parsing YAML: %w", err)
	}

	return toDomainConfig(&yamlCfg)
}

// toDomainConfig converts YAML config to domain config.
//
// WHY: Trennung zwischen YAML-Representation und Domain-Entity.
//      YAML kann sich ändern ohne Domain zu beeinflussen.
//
// WHAT: Mappt YAML-Strukturen auf domain-Types, parst Duration-Strings.
// IMPACT: Ohne Konversion wäre Domain an YAML-Format gebunden.
func toDomainConfig(yamlCfg *yamlConfig) (*domain.Config, error) {
	widgets := make([]*domain.Widget, 0, len(yamlCfg.Widgets))

	for _, yw := range yamlCfg.Widgets {
		interval, err := time.ParseDuration(yw.Interval)
		if err != nil {
			return nil, fmt.Errorf("parsing interval for widget %s: %w", yw.Type, err)
		}

		w, err := domain.NewWidget(
			domain.WidgetType(yw.Type),
			domain.Position{X: yw.Position.X, Y: yw.Position.Y},
			domain.Size{Width: yw.Size.Width, Height: yw.Size.Height},
			domain.Style{
				Font:       yw.Style.Font,
				Color:      yw.Style.Color,
				Background: yw.Style.Background,
			},
			interval,
		)
		if err != nil {
			return nil, fmt.Errorf("creating widget %s: %w", yw.Type, err)
		}

		w.CustomCommand = yw.CustomCommand
		widgets = append(widgets, w)
	}

	cfg, err := domain.NewConfig(domain.Config{
		Widgets: widgets,
		Background: domain.BackgroundConfig{
			Type:      domain.BackgroundType(yamlCfg.Background.Type),
			Colors:    yamlCfg.Background.Colors,
			ImagePath: yamlCfg.Background.ImagePath,
		},
		Renderer: domain.RendererConfig{
			Type: domain.RendererType(yamlCfg.Renderer.Type),
		},
		API: domain.APIConfig{
			Enabled:   yamlCfg.API.Enabled,
			Port:      yamlCfg.API.Port,
			WebSocket: yamlCfg.API.WebSocket,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("validating config: %w", err)
	}

	return cfg, nil
}
