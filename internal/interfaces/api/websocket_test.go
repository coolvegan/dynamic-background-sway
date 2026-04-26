package api

import (
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"gittea.kittel.dev/marco/dynamic_background/internal/application"
	"gittea.kittel.dev/marco/dynamic_background/internal/domain"
	"gittea.kittel.dev/marco/dynamic_background/internal/infrastructure/collector"

	"github.com/gorilla/websocket"
)

func setupWebSocketTest(t *testing.T) (*Server, *httptest.Server) {
	collectors := map[domain.WidgetType]domain.Collector{
		domain.WidgetTypeClock: collector.NewClockCollectorWithTimeSource(func() time.Time {
			return time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
		}),
	}

	w, _ := domain.NewWidget(domain.WidgetTypeClock, domain.Position{0, 0}, domain.Size{100, 50}, domain.Style{}, time.Second)

	cfg, err := domain.NewConfig(domain.Config{
		Widgets: []*domain.Widget{w},
		Background: domain.BackgroundConfig{
			Type: domain.BackgroundTypeSolid,
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

func TestWebSocket_Connection(t *testing.T) {
	_, testSrv := setupWebSocketTest(t)
	defer testSrv.Close()

	// Convert HTTP URL to WebSocket URL
	wsURL := "ws" + strings.TrimPrefix(testSrv.URL, "http") + "/api/v1/ws"

	dialer := websocket.Dialer{}
	conn, _, err := dialer.Dial(wsURL, nil)

	if err != nil {
		t.Fatalf("failed to connect to WebSocket: %v", err)
	}
	defer conn.Close()

	// Connection established successfully
}

func TestWebSocket_ReceivesUpdates(t *testing.T) {
	_, testSrv := setupWebSocketTest(t)
	defer testSrv.Close()

	wsURL := "ws" + strings.TrimPrefix(testSrv.URL, "http") + "/api/v1/ws"

	dialer := websocket.Dialer{}
	conn, _, err := dialer.Dial(wsURL, nil)

	if err != nil {
		t.Fatalf("failed to connect to WebSocket: %v", err)
	}
	defer conn.Close()

	// Read message with timeout
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))

	_, message, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("failed to read message: %v", err)
	}

	// Should receive a JSON message
	if len(message) == 0 {
		t.Error("expected non-empty message")
	}
}

func TestWebSocket_Broadcast(t *testing.T) {
	srv, testSrv := setupWebSocketTest(t)
	defer testSrv.Close()

	wsURL := "ws" + strings.TrimPrefix(testSrv.URL, "http") + "/api/v1/ws"

	// Connect two clients
	dialer := websocket.Dialer{}
	conn1, _, err := dialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("failed to connect client 1: %v", err)
	}
	defer conn1.Close()

	conn2, _, err := dialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("failed to connect client 2: %v", err)
	}
	defer conn2.Close()

	// Give connections time to register
	time.Sleep(50 * time.Millisecond)

	// Broadcast a message
	srv.Broadcast(map[string]interface{}{
		"type": "test",
		"data": "hello",
	})

	// Both clients should receive
	conn1.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, msg1, err := conn1.ReadMessage()
	if err != nil {
		t.Fatalf("client 1 failed to read: %v", err)
	}

	conn2.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, msg2, err := conn2.ReadMessage()
	if err != nil {
		t.Fatalf("client 2 failed to read: %v", err)
	}

	if string(msg1) != string(msg2) {
		t.Error("expected both clients to receive same message")
	}
}
