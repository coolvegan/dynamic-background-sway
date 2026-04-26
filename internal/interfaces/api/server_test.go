package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"gittea.kittel.dev/marco/dynamic_background/internal/application"
	"gittea.kittel.dev/marco/dynamic_background/internal/domain"
	"gittea.kittel.dev/marco/dynamic_background/internal/infrastructure/collector"
)

func setupTestServer(t *testing.T) (*Server, *httptest.Server) {
	collectors := map[domain.WidgetType]domain.Collector{
		domain.WidgetTypeClock: collector.NewClockCollectorWithTimeSource(func() time.Time {
			return time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
		}),
		domain.WidgetTypeCPU: collector.NewCPUCollector(),
	}

	w, _ := domain.NewWidget(domain.WidgetTypeClock, domain.Position{0, 0}, domain.Size{100, 50}, domain.Style{}, time.Second)

	cfg, err := domain.NewConfig(domain.Config{
		Widgets: []*domain.Widget{w},
		Background: domain.BackgroundConfig{
			Type: domain.BackgroundTypeSolid,
		},
		API: domain.APIConfig{
			Enabled: true,
			Port:    8080,
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	wm := application.NewWidgetManager(cfg, collectors)
	sched := application.NewScheduler(wm)

	srv := NewServer(cfg, wm, sched, nil)

	testSrv := httptest.NewServer(srv.router)

	return srv, testSrv
}

func TestServer_GetConfig(t *testing.T) {
	_, testSrv := setupTestServer(t)
	defer testSrv.Close()

	resp, err := http.Get(testSrv.URL + "/api/v1/config")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	var cfg domain.Config
	if err := json.NewDecoder(resp.Body).Decode(&cfg); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(cfg.Widgets) != 1 {
		t.Errorf("expected 1 widget, got %d", len(cfg.Widgets))
	}
}

func TestServer_GetWidgets(t *testing.T) {
	_, testSrv := setupTestServer(t)
	defer testSrv.Close()

	resp, err := http.Get(testSrv.URL + "/api/v1/widgets")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	var widgets []domain.Widget
	if err := json.NewDecoder(resp.Body).Decode(&widgets); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(widgets) != 1 {
		t.Errorf("expected 1 widget, got %d", len(widgets))
	}
}

func TestServer_GetSystemInfo(t *testing.T) {
	_, testSrv := setupTestServer(t)
	defer testSrv.Close()

	resp, err := http.Get(testSrv.URL + "/api/v1/system")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	var info map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Should have timestamp
	if _, ok := info["timestamp"]; !ok {
		t.Error("expected timestamp in system info")
	}
}

func TestServer_GetHealth(t *testing.T) {
	_, testSrv := setupTestServer(t)
	defer testSrv.Close()

	resp, err := http.Get(testSrv.URL + "/health")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	var health map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if health["status"] != "ok" {
		t.Errorf("expected status 'ok', got %q", health["status"])
	}
}

func TestServer_AddWidget(t *testing.T) {
	_, testSrv := setupTestServer(t)
	defer testSrv.Close()

	newWidget := map[string]interface{}{
		"type": "cpu",
		"position": map[string]int{
			"x": 100,
			"y": 100,
		},
		"size": map[string]int{
			"width":  150,
			"height": 60,
		},
		"interval": "2s",
	}

	body, _ := json.Marshal(newWidget)

	resp, err := http.Post(testSrv.URL+"/api/v1/widgets", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("expected status 201, got %d", resp.StatusCode)
	}
}

func TestServer_AddWidget_InvalidType(t *testing.T) {
	_, testSrv := setupTestServer(t)
	defer testSrv.Close()

	newWidget := map[string]interface{}{
		"type": "invalid_type",
		"position": map[string]int{
			"x": 0,
			"y": 0,
		},
		"size": map[string]int{
			"width":  100,
			"height": 50,
		},
		"interval": "1s",
	}

	body, _ := json.Marshal(newWidget)

	resp, err := http.Post(testSrv.URL+"/api/v1/widgets", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", resp.StatusCode)
	}
}
