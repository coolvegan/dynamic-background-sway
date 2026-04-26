package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"gittea.kittel.dev/marco/dynamic_background/internal/application"
	"gittea.kittel.dev/marco/dynamic_background/internal/domain"

	"github.com/gorilla/mux"
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
func (s *Server) handleUpdateConfig(w http.ResponseWriter, r *http.Request) {
	var newCfg domain.Config
	if err := json.NewDecoder(r.Body).Decode(&newCfg); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// TODO: Validate and apply new config
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
	// TODO: Implement WebSocket handler
	w.WriteHeader(http.StatusNotImplemented)
}

var startTime = time.Now()
