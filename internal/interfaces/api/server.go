// Package api provides the HTTP and WebSocket interface for external control
// and monitoring of the dynamic_background application.
//
// This file (server.go) defines the Server — a REST API + WebSocket server
// that allows external tools to read/modify configuration, manage widgets,
// and receive real-time updates without restarting the process.
//
// Why it exists:
//   Without the API server, users would need to edit the YAML config file and
//   restart the process to change widgets or background settings. The API
//   enables:
//   - Hot-reload: change widgets/background without restarting
//   - Monitoring: check health, system info, current widget state
//   - Integration: other tools can programmatically control the background
//   - Live updates: WebSocket pushes changes to connected clients
//
// How it connects:
//   - Created in main_cgo.go if cfg.API.Enabled is true
//   - Receives references to Config, WidgetManager, Scheduler, Orchestrator
//   - Routes are registered in registerRoutes() using gorilla/mux
//   - handleUpdateConfig() validates incoming JSON, creates new domain.Widgets
//     via NewWidget(), updates shared cfg, and triggers re-render via
//     orchestrator.UpdateBackgroundConfig()
//   - Broadcast() sends JSON messages to all connected WebSocket clients
//   - WebSocket clients receive "config_change" events when config is updated
//
// API Endpoints (all under /api/v1/):
//   GET  /health          - Health check {"status": "ok"}
//   GET  /config          - Current configuration
//   PUT  /config          - Hot-reload configuration
//   GET  /widgets         - List all widgets
//   POST /widgets         - Add a new widget
//   PUT  /widgets/{id}    - Update widget (TODO)
//   DELETE /widgets/{id}  - Remove widget (TODO)
//   GET  /system          - System info (uptime, widget count)
//   GET  /ws              - WebSocket upgrade for live updates
//
// Key concept: The API server shares the same *domain.Config pointer as the
// rest of the application. When handleUpdateConfig() replaces cfg.Widgets,
// the WidgetManager and Scheduler immediately see the new widgets. A mutex
// protects concurrent access to the config.
package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"gittea.kittel.dev/marco/dynamic_background/internal/application"
	"gittea.kittel.dev/marco/dynamic_background/internal/domain"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

// Server is the HTTP/WebSocket API server.
//
// WHY: Externer Zugriff auf Config, Widgets, System-Infos.
//      Hot-Reload ohne Neustart; Monitoring; Integration mit anderen Tools.
//
// WHAT: REST API für Config/Widgets/System, WebSocket für Echtzeit-Updates.
// IMPACT: Ohne API Server müsste User Config-Datei bearbeiten und Prozess neustarten.
type Server struct {
	cfg           *domain.Config
	widgetManager *application.WidgetManager
	scheduler     *application.Scheduler
	orchestrator  *application.Orchestrator
	router        *mux.Router
	upgrader      websocket.Upgrader
	clients       map[*websocket.Conn]bool
	mu            sync.RWMutex
}

// NewServer creates a new API server instance.
//
// WHY: Factory für saubere Initialisierung mit allen Dependencies.
// WHAT: Erstellt Router, registriert Routes.
// IMPACT: Ohne Factory müsste Caller Router manuell konfigurieren.
func NewServer(cfg *domain.Config, wm *application.WidgetManager, sched *application.Scheduler, orch *application.Orchestrator) *Server {
	s := &Server{
		cfg:           cfg,
		widgetManager: wm,
		scheduler:     sched,
		orchestrator:  orch,
		router:        mux.NewRouter(),
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins for local use
			},
		},
		clients: make(map[*websocket.Conn]bool),
	}

	s.registerRoutes()

	return s
}

// registerRoutes sets up all API endpoints.
func (s *Server) registerRoutes() {
	// Health check
	s.router.HandleFunc("/health", s.handleHealth).Methods("GET")

	// API v1
	api := s.router.PathPrefix("/api/v1").Subrouter()
	api.HandleFunc("/config", s.handleGetConfig).Methods("GET")
	api.HandleFunc("/config", s.handleUpdateConfig).Methods("PUT")
	api.HandleFunc("/widgets", s.handleGetWidgets).Methods("GET")
	api.HandleFunc("/widgets", s.handleAddWidget).Methods("POST")
	api.HandleFunc("/widgets/{id}", s.handleUpdateWidget).Methods("PUT")
	api.HandleFunc("/widgets/{id}", s.handleRemoveWidget).Methods("DELETE")
	api.HandleFunc("/system", s.handleGetSystemInfo).Methods("GET")
	api.HandleFunc("/ws", s.handleWebSocket).Methods("GET")
}

// Start begins listening on the configured port.
//
// WHY: Startet den HTTP Server für API-Zugriff.
// WHAT: http.ListenAndServe auf konfiguriertem Port.
// IMPACT: Ohne Start() ist API nicht erreichbar.
func (s *Server) Start() error {
	addr := fmt.Sprintf(":%d", s.cfg.API.Port)
	fmt.Printf("API server starting on %s\n", addr)
	return http.ListenAndServe(addr, s.router)
}

// handleHealth returns server health status.
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "ok",
	})
}

// handleGetConfig returns the current configuration.
func (s *Server) handleGetConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s.cfg)
}

// handleUpdateConfig updates the configuration.
//
// WHY: Hot-Reload der Config ohne Neustart; User kann Widgets/Background/API ändern.
// WHAT: Parst JSON mit String-Intervallen, konvertiert zu Domain-Typen, validiert, applied, broadcastet.
// IMPACT: Ohne Validierung könnten invalide Configs das System brechen; ohne Broadcast wissen Clients nicht von Änderungen.
func (s *Server) handleUpdateConfig(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Background domain.BackgroundConfig `json:"background"`
		Widgets    []struct {
			Type     string            `json:"type"`
			Position domain.Position   `json:"position"`
			Size     domain.Size       `json:"size"`
			Style    domain.Style      `json:"style"`
			Interval string            `json:"interval"`
		} `json:"widgets"`
		API domain.APIConfig `json:"api"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var widgets []*domain.Widget
	for _, wReq := range req.Widgets {
		interval, err := time.ParseDuration(wReq.Interval)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "invalid interval: " + err.Error(),
			})
			return
		}

		widget, err := domain.NewWidget(
			domain.WidgetType(wReq.Type),
			wReq.Position,
			wReq.Size,
			wReq.Style,
			interval,
		)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{
				"error": err.Error(),
			})
			return
		}

		widgets = append(widgets, widget)
	}

	newCfg, err := domain.NewConfig(domain.Config{
		Widgets:    widgets,
		Background: req.Background,
		API:        req.API,
	})
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": err.Error(),
		})
		return
	}

	s.mu.Lock()
	s.cfg.Widgets = newCfg.Widgets
	s.cfg.Background = newCfg.Background
	s.cfg.API = newCfg.API
	s.mu.Unlock()

	// Update renderer and trigger immediate re-render
	if s.orchestrator != nil {
		s.orchestrator.UpdateBackgroundConfig(newCfg.Background, context.Background())
	}

	s.Broadcast(map[string]interface{}{
		"type": "config_change",
		"data": map[string]interface{}{
			"widgets":   len(newCfg.Widgets),
			"timestamp": time.Now().Format(time.RFC3339),
		},
	})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "config updated",
	})
}

// handleGetWidgets returns all widgets.
func (s *Server) handleGetWidgets(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s.cfg.Widgets)
}

// handleAddWidget adds a new widget to the configuration.
func (s *Server) handleAddWidget(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Type       string        `json:"type"`
		Position   domain.Position `json:"position"`
		Size       domain.Size     `json:"size"`
		Style      domain.Style    `json:"style"`
		Interval   string        `json:"interval"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	interval, err := time.ParseDuration(req.Interval)
	if err != nil {
		http.Error(w, "invalid interval: "+err.Error(), http.StatusBadRequest)
		return
	}

	widget, err := domain.NewWidget(
		domain.WidgetType(req.Type),
		req.Position,
		req.Size,
		req.Style,
		interval,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	s.cfg.AddWidget(widget)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(widget)
}

// handleUpdateWidget updates an existing widget.
func (s *Server) handleUpdateWidget(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	_ = vars["id"]

	// TODO: Implement widget update by ID
	w.WriteHeader(http.StatusNotImplemented)
}

// handleRemoveWidget removes a widget by ID.
func (s *Server) handleRemoveWidget(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	_ = vars["id"]

	// TODO: Implement widget removal by ID
	w.WriteHeader(http.StatusNotImplemented)
}

// handleGetSystemInfo returns live system information.
func (s *Server) handleGetSystemInfo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"timestamp": time.Now().Format(time.RFC3339),
		"widgets":   len(s.cfg.Widgets),
		"uptime":    time.Since(startTime).String(),
	})
}

// handleWebSocket upgrades connection to WebSocket for real-time updates.
func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	s.mu.Lock()
	s.clients[conn] = true
	s.mu.Unlock()

	// Send initial state
	s.sendToClient(conn, map[string]interface{}{
		"type": "connected",
		"data": map[string]interface{}{
			"timestamp": time.Now().Format(time.RFC3339),
		},
	})

	// Keep connection alive and handle messages
	go s.handleClientMessages(conn)
}

// handleClientMessages reads messages from a client.
func (s *Server) handleClientMessages(conn *websocket.Conn) {
	defer func() {
		s.mu.Lock()
		delete(s.clients, conn)
		s.mu.Unlock()
		conn.Close()
	}()

	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			break
		}
	}
}

// Broadcast sends a message to all connected WebSocket clients.
//
// WHY: Echtzeit-Updates an alle Clients wenn sich Widgets ändern.
// WHAT: Iteriert über alle Connections, sendet JSON-Message.
// IMPACT: Ohne Broadcast müssten Clients pollen; ineffizient und verzögert.
func (s *Server) Broadcast(message map[string]interface{}) {
	data, err := json.Marshal(message)
	if err != nil {
		return
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	for conn := range s.clients {
		s.sendToClient(conn, message)
		_ = data // Use data variable
	}
}

// sendToClient sends a JSON message to a single client.
func (s *Server) sendToClient(conn *websocket.Conn, message interface{}) {
	data, err := json.Marshal(message)
	if err != nil {
		return
	}

	conn.WriteMessage(websocket.TextMessage, data)
}

var startTime = time.Now()
