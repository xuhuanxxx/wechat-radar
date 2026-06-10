package handlers

import (
	"github.com/xuhuanxxx/wechat-radar/apps/data-service/api"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/xuhuanxxx/wechat-radar/apps/data-service/db"
	"github.com/xuhuanxxx/wechat-radar/apps/data-service/sync"
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
			writeJSON(w, http.StatusServiceUnavailable, api.SetupStatus{
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
	writeJSON(w, http.StatusOK, api.HealthResponse{
		OK:      true,
		Version: "0.1.0",
		Service: "lark-radar-data",
	})
}

// Doctor runs diagnostic checks
func (h *Handlers) Doctor(w http.ResponseWriter, r *http.Request) {
	checks := []api.DoctorCheck{}

	// DB check
	if err := h.db.Ping(); err != nil {
		checks = append(checks, api.DoctorCheck{
			Name:    "database",
			Status:  "error",
			Message: err.Error(),
		})
	} else {
		checks = append(checks, api.DoctorCheck{
			Name:   "database",
			Status: "ok",
		})
	}

	// Config check
	if _, err := h.config.Load(); err != nil {
		checks = append(checks, api.DoctorCheck{
			Name:    "config",
			Status:  "error",
			Message: err.Error(),
		})
	} else {
		checks = append(checks, api.DoctorCheck{
			Name:   "config",
			Status: "ok",
		})
	}

	// Lark CLI check
	if err := h.syncEngine.CheckLarkCLI(); err != nil {
		checks = append(checks, api.DoctorCheck{
			Name:    "lark-cli",
			Status:  "error",
			Message: err.Error(),
		})
	} else {
		checks = append(checks, api.DoctorCheck{
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

	writeJSON(w, http.StatusOK, api.DoctorResponse{
		OK:     allOK,
		Checks: checks,
	})
}

// Setup handles /api/setup (GET returns status, POST saves config)
func (h *Handlers) Setup(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.setupGet(w, r)
	case http.MethodPost:
		h.setupPost(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

func (h *Handlers) setupGet(w http.ResponseWriter, r *http.Request) {
	cfg, err := h.config.Load()
	if err != nil {
		writeJSON(w, http.StatusOK, api.SetupStatus{
			OK:         false,
			DataDir:    h.config.DataDir(),
			Configured: false,
			Config:     api.Config{},
			Checks:     h.runChecks(),
			Error:      err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, api.SetupStatus{
		OK:         true,
		DataDir:    h.config.DataDir(),
		Configured: len(cfg.MyNicknames) > 0,
		Config:     *cfg,
		Checks:     h.runChecks(),
	})
}

// runChecks runs environment checks for the setup page
func (h *Handlers) runChecks() api.SetupChecks {
	checks := api.SetupChecks{}

	// Check lark-cli availability
	if err := h.syncEngine.CheckLarkCLI(); err != nil {
		checks.LarkError = err.Error()
	} else {
		checks.LarkInstalled = true
		// Check authentication
		if h.syncEngine.CheckLarkAuth() == nil {
			checks.LarkAuthenticated = true
		}
	}

	return checks
}

func (h *Handlers) setupPost(w http.ResponseWriter, r *http.Request) {
	var req api.SetupRequest
	if err := parseJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	if len(req.MyNicknames) == 0 {
		writeError(w, http.StatusBadRequest, "myNicknames required")
		return
	}

	cfg := &api.Config{
		MyNicknames:      req.MyNicknames,
		DefaultRange:     req.DefaultRange,
		Port:             req.Port,
		AutoSyncInterval: req.AutoSyncInterval,
		LarkChatFilter:   req.LarkChatFilter,
		DemoMode:         req.DemoMode,
		PrivacyConfirmed: req.PrivacyConfirmed,
		DefaultSyncDays:  req.DefaultSyncDays,
		Source:           req.Source,
		LarkCliPath:      req.LarkCliPath,
		OpenApiKey:       req.OpenApiKey,
		SetupCompleted:   true,
	}

	if cfg.DefaultRange == "" {
		cfg.DefaultRange = "7d"
	}
	if cfg.Port == 0 {
		cfg.Port = 8787
	}
	if cfg.DefaultSyncDays == 0 {
		cfg.DefaultSyncDays = 7
	}

	if err := h.config.Save(cfg); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, api.SetupResponse{
		OK:         true,
		Configured: true,
		Message:    "Setup complete",
	})
}

// GetConfig returns current config
func (h *Handlers) GetConfig(w http.ResponseWriter, r *http.Request) {
	cfg, err := h.config.Load()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, api.ConfigResponse{
		OK:     true,
		Config: *cfg,
	})
}
