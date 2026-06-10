package handlers

import (
	"net/http"

	"github.com/xuhuanxxx/wechat-radar/apps/data-service/models"
)

// Mentions returns mentions
func (h *Handlers) Mentions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	chatroomID := r.URL.Query().Get("chatroom_id")
	mentioned := r.URL.Query().Get("mentioned")

	query := "SELECT id, chatroom_id, message_id, mentioned, mentioner, date FROM mentions WHERE 1=1"
	args := []interface{}{}

	if chatroomID != "" {
		query += " AND chatroom_id = ?"
		args = append(args, chatroomID)
	}
	if mentioned != "" {
		query += " AND mentioned = ?"
		args = append(args, mentioned)
	}
	query += " ORDER BY date DESC LIMIT 100"

	rows, err := h.db.Query(query, args...)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()

	mentions := []models.Mention{}
	for rows.Next() {
		var m models.Mention
		if err := rows.Scan(&m.ID, &m.ChatroomID, &m.MessageID, &m.Mentioned, &m.Mentioner, &m.Date); err != nil {
			continue
		}
		mentions = append(mentions, m)
	}

	writeJSON(w, http.StatusOK, models.MentionsResponse{
		OK:       true,
		Mentions: mentions,
	})
}

// MentionStats returns mention statistics
func (h *Handlers) MentionStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	chatroomID := r.URL.Query().Get("chatroom_id")

	query := "SELECT mentioned, mentioner FROM mentions WHERE 1=1"
	args := []interface{}{}

	if chatroomID != "" {
		query += " AND chatroom_id = ?"
		args = append(args, chatroomID)
	}

	rows, err := h.db.Query(query, args...)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()

	mentioned := make(map[string]int)
	mentionedBy := make(map[string][]string)

	for rows.Next() {
		var m, by string
		if err := rows.Scan(&m, &by); err != nil {
			continue
		}
		mentioned[m]++
		if !contains(mentionedBy[m], by) {
			mentionedBy[m] = append(mentionedBy[m], by)
		}
	}

	writeJSON(w, http.StatusOK, models.MentionStats{
		OK:          true,
		Mentioned:   mentioned,
		MentionedBy: mentionedBy,
	})
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
