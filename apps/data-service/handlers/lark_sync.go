package handlers

import (
	"github.com/xuhuanxxx/wechat-radar/apps/data-service/api"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// LarkSync handles POST /api/lark/sync with optional SSE streaming
func (h *Handlers) LarkSync(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req api.SyncRequest
	if err := parseJSON(r, &req); err != nil {
		req = api.SyncRequest{}
	}

	if req.DaysBack == 0 {
		req.DaysBack = 7
	}

	// Check if streaming is requested via query param or body
	streamParam := r.URL.Query().Get("stream")
	isStream := req.Stream || streamParam == "true" || streamParam == "1"

	if isStream {
		h.handleLarkSyncStream(w, r, req)
		return
	}

	// Non-streaming: sync all or single chat
	var results map[string]api.SyncResult
	var err error

	if req.ChatID != "" {
		results = make(map[string]api.SyncResult)
		result, syncErr := h.syncEngine.SyncChat(req.ChatID, req.DaysBack)
		if syncErr != nil {
			err = syncErr
		} else {
			results[req.ChatID] = result
		}
	} else {
		results, err = h.syncEngine.SyncAll(req.DaysBack)
	}

	if err != nil {
		writeJSON(w, http.StatusOK, api.SyncResponse{
			OK:     false,
			Error:  err.Error(),
			Synced: results,
		})
		return
	}

	writeJSON(w, http.StatusOK, api.SyncResponse{
		OK:     true,
		Synced: results,
	})
}

func (h *Handlers) handleLarkSyncStream(w http.ResponseWriter, r *http.Request, req api.SyncRequest) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "Streaming not supported")
		return
	}

	sendEvent := func(event api.SyncProgressEvent) {
		data, _ := json.Marshal(event)
		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()
	}

	// Send start event
	sendEvent(api.SyncProgressEvent{
		Type:  "start",
		Phase: "fetching_chats",
	})

	// If single chat, just sync it
	if req.ChatID != "" {
		sendEvent(api.SyncProgressEvent{
			Type:   "progress",
			ChatID: req.ChatID,
			Phase:  "syncing",
			Count:  0,
		})

		result, err := h.syncEngine.SyncChat(req.ChatID, req.DaysBack)
		if err != nil {
			sendEvent(api.SyncProgressEvent{
				Type:  "error",
				Error: err.Error(),
			})
			return
		}

		sendEvent(api.SyncProgressEvent{
			Type:   "progress",
			ChatID: req.ChatID,
			Phase:  "done",
			Count:  result.Inserted,
		})

		sendEvent(api.SyncProgressEvent{
			Type:   "finished",
			OK:     true,
			Synced: map[string]api.SyncResult{req.ChatID: result},
		})
		return
	}

	// Sync all with progress updates via status polling
	go func() {
		_, _ = h.syncEngine.SyncAll(req.DaysBack)
	}()

	// Poll status and send progress
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			status := h.syncEngine.GetStatus()
			if !status.Running && !status.CompletedAt.IsZero() {
				// Send final progress for all chats
				for chatID, result := range status.Results {
					sendEvent(api.SyncProgressEvent{
						Type:   "progress",
						ChatID: chatID,
						Phase:  "done",
						Count:  result.Inserted,
					})
				}
				sendEvent(api.SyncProgressEvent{
					Type:   "finished",
					OK:     status.Error == "",
					Synced: status.Results,
				})
				return
			}
			for chatID, progress := range status.Progress {
				sendEvent(api.SyncProgressEvent{
					Type:   "progress",
					ChatID: chatID,
					Phase:  "syncing",
					Count:  progress,
				})
			}
		}
	}
}
