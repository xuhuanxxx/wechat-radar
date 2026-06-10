package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"go-server/db"
	"go-server/models"
	"go-server/sync"
)

// Handlers holds all HTTP handlers
type Handlers struct {
	db         *db.DB
	config     *db.ConfigManager
	syncEngine *sync.Engine
}

// New creates a new handlers instance
func New(database *db.DB, config *db.ConfigManager, syncEngine *sync.Engine) *Handlers {
	return &Handlers{
		db:         database,
		config:     config,
		syncEngine: syncEngine,
	}
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]interface{}{
		"ok":    false,
		"error": message,
	})
}

func parseJSON(r *http.Request, v interface{}) error {
	return json.NewDecoder(r.Body).Decode(v)
}

func getQueryParam(r *http.Request, key, defaultValue string) string {
	if v := r.URL.Query().Get(key); v != "" {
		return v
	}
	return defaultValue
}

func getPathParam(r *http.Request, prefix string) string {
	path := strings.TrimPrefix(r.URL.Path, prefix)
	path = strings.Trim(path, "/")
	return path
}

// requireSetup middleware check
func (h *Handlers) requireSetup(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !h.config.IsConfigured() {
			writeJSON(w, http.StatusServiceUnavailable, models.SetupStatus{
				OK:         false,
				Configured: false,
				Error:      "Setup required",
			})
			return
		}
		next(w, r)
	}
}

// Health returns health status
func (h *Handlers) Health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, models.HealthResponse{
		OK:      true,
		Version: "0.1.0",
		Service: "lark-radar-data",
	})
}

// Doctor runs diagnostic checks
func (h *Handlers) Doctor(w http.ResponseWriter, r *http.Request) {
	checks := []models.DoctorCheck{}

	// DB check
	if err := h.db.Ping(); err != nil {
		checks = append(checks, models.DoctorCheck{
			Name:    "database",
			Status:  "error",
			Message: err.Error(),
		})
	} else {
		checks = append(checks, models.DoctorCheck{
			Name:   "database",
			Status: "ok",
		})
	}

	// Config check
	if _, err := h.config.Load(); err != nil {
		checks = append(checks, models.DoctorCheck{
			Name:    "config",
			Status:  "error",
			Message: err.Error(),
		})
	} else {
		checks = append(checks, models.DoctorCheck{
			Name:   "config",
			Status: "ok",
		})
	}

	// Lark CLI check
	if err := h.syncEngine.CheckLarkCLI(); err != nil {
		checks = append(checks, models.DoctorCheck{
			Name:    "lark-cli",
			Status:  "error",
			Message: err.Error(),
		})
	} else {
		checks = append(checks, models.DoctorCheck{
			Name:   "lark-cli",
			Status: "ok",
		})
	}

	allOK := true
	for _, c := range checks {
		if c.Status != "ok" {
			allOK = false
			break
		}
	}

	writeJSON(w, http.StatusOK, models.DoctorResponse{
		OK:     allOK,
		Checks: checks,
	})
}

// SetupStatus returns setup status
func (h *Handlers) SetupStatus(w http.ResponseWriter, r *http.Request) {
	cfg, err := h.config.Load()
	if err != nil {
		writeJSON(w, http.StatusOK, models.SetupStatus{
			OK:         false,
			Configured: false,
			Error:      err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, models.SetupStatus{
		OK:         true,
		Configured: len(cfg.MyNicknames) > 0,
	})
}

// Setup handles initial setup
func (h *Handlers) Setup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req models.SetupRequest
	if err := parseJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	if len(req.MyNicknames) == 0 {
		writeError(w, http.StatusBadRequest, "myNicknames required")
		return
	}

	cfg := &models.Config{
		MyNicknames:      req.MyNicknames,
		DefaultRange:     req.DefaultRange,
		Port:             req.Port,
		AutoSyncInterval: req.AutoSyncInterval,
		LarkChatFilter:   req.LarkChatFilter,
	}

	if cfg.DefaultRange == "" {
		cfg.DefaultRange = "7d"
	}
	if cfg.Port == 0 {
		cfg.Port = 8787
	}

	if err := h.config.Save(cfg); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, models.SetupResponse{
		OK:      true,
		Message: "Setup complete",
	})
}

// GetConfig returns current config
func (h *Handlers) GetConfig(w http.ResponseWriter, r *http.Request) {
	cfg, err := h.config.Load()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, models.ConfigResponse{
		OK:     true,
		Config: *cfg,
	})
}
