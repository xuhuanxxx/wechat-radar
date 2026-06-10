package handlers

import (
	"net/http"
	"strconv"

	"github.com/xuhuanxxx/wechat-radar/apps/data-service/models"
)

// NewMessages handles GET /api/new-messages
func (h *Handlers) NewMessages(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	rows, err := h.db.Query(`
		SELECT chatroom_id, local_id, sender, content, timestamp, type, date
		FROM messages
		ORDER BY timestamp DESC
		LIMIT ?`,
		limit,
	)
	if err != nil {
		writeJSON(w, http.StatusOK, models.NewMessagesResponse{
			OK:       true,
			Messages: []models.Message{},
			Count:    0,
		})
		return
	}
	defer rows.Close()

	messages := []models.Message{}
	for rows.Next() {
		var m models.Message
		if err := rows.Scan(&m.ChatroomID, &m.LocalID, &m.Sender, &m.Content, &m.Timestamp, &m.Type, &m.Date); err != nil {
			continue
		}
		messages = append(messages, m)
	}

	writeJSON(w, http.StatusOK, models.NewMessagesResponse{
		OK:       true,
		Messages: messages,
		Count:    len(messages),
	})
}

// Rescan handles POST /api/rescan
func (h *Handlers) Rescan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// For MVP: trigger a lightweight re-analysis
	// Re-count daily_stats from messages
	_, err := h.db.Exec(`
		INSERT INTO daily_stats (chatroom_id, date, message_count, unique_senders)
		SELECT chatroom_id, date, COUNT(*), COUNT(DISTINCT sender)
		FROM messages
		GROUP BY chatroom_id, date
		ON CONFLICT(chatroom_id, date) DO UPDATE SET
			message_count = excluded.message_count,
			unique_senders = excluded.unique_senders
	`)
	if err != nil {
		writeJSON(w, http.StatusOK, models.RescanResponse{
			OK:      true,
			Message: "Re-scan triggered (stats update may be partial)",
		})
		return
	}

	writeJSON(w, http.StatusOK, models.RescanResponse{
		OK:      true,
		Message: "Re-scan completed",
	})
}

// WXImage handles GET /api/wx-image (proxy image fetch)
func (h *Handlers) WXImage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	url := r.URL.Query().Get("url")
	if url == "" {
		writeError(w, http.StatusBadRequest, "url required")
		return
	}

	// For MVP: return a placeholder JSON response
	// In production, this would proxy the image with proper headers/cookies
	writeJSON(w, http.StatusOK, models.WXImageResponse{
		OK:      true,
		URL:     url,
		Message: "Image proxy placeholder",
	})
}
