package handlers

import (
	"net/http"

	"github.com/xuhuanxxx/wechat-radar/apps/data-service/models"
)

// Sync triggers a sync operation
func (h *Handlers) Sync(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req models.SyncRequest
	if err := parseJSON(r, &req); err != nil {
		// Empty body is ok, use defaults
		req = models.SyncRequest{}
	}

	if req.DaysBack == 0 {
		req.DaysBack = 7
	}

	// Run sync synchronously for now
	results, err := h.syncEngine.SyncAll(req.DaysBack)
	if err != nil {
		writeJSON(w, http.StatusOK, models.SyncResponse{
			OK:     false,
			Error:  err.Error(),
			Synced: results,
		})
		return
	}

	writeJSON(w, http.StatusOK, models.SyncResponse{
		OK:     true,
		Synced: results,
	})
}

// SyncProgress returns sync progress (WebSocket upgrade or SSE)
func (h *Handlers) SyncProgress(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// For now, return current sync status
	status := h.syncEngine.GetStatus()
	writeJSON(w, http.StatusOK, status)
}
