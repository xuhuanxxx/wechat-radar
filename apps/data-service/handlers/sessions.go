package handlers

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/xuhuanxxx/wechat-radar/apps/data-service/models"
)

// Sessions returns all chatroom sessions
func (h *Handlers) Sessions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	query := `
		SELECT 
			m.chatroom_id,
			COALESCE(g.name, m.chatroom_id) as name,
			COUNT(*) as message_count,
			MAX(m.timestamp) as last_active,
			COUNT(DISTINCT m.sender) as unique_senders
		FROM messages m
		LEFT JOIN groups g ON m.chatroom_id = g.chatroom_id
		GROUP BY m.chatroom_id
		ORDER BY last_active DESC
	`

	rows, err := h.db.Query(query)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()

	groups := []models.SessionGroup{}
	for rows.Next() {
		var chatroomID, name string
		var messageCount int
		var lastActive sql.NullInt64
		var uniqueSenders int
		if err := rows.Scan(&chatroomID, &name, &messageCount, &lastActive, &uniqueSenders); err != nil {
			continue
		}
		
		var ts int64
		if lastActive.Valid {
			ts = lastActive.Int64
		}
		
		groups = append(groups, models.SessionGroup{
			ChatroomID: chatroomID,
			Name:       name,
			Summary:    fmt.Sprintf("%d 条消息 · %d 人参与", messageCount, uniqueSenders),
			Time:       formatTime(ts),
			Timestamp:  ts,
			Unread:     0,
			IsFavorite: false,
			GroupIDs:   []int{},
		})
	}

	writeJSON(w, http.StatusOK, models.SessionsResponse{
		OK:         true,
		Total:      len(groups),
		Groups:     groups,
		Categories: []models.CategoryInfo{},
	})
}

func formatTime(ts int64) string {
	if ts == 0 {
		return ""
	}
	t := time.Unix(ts, 0)
	now := time.Now()
	if t.Year() == now.Year() && t.YearDay() == now.YearDay() {
		return t.Format("15:04")
	}
	if t.Year() == now.Year() {
		return t.Format("01-02")
	}
	return t.Format("2006-01-02")
}

// SessionDetail returns details for a specific chatroom
func (h *Handlers) SessionDetail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	chatroomID := getPathParam(r, "/api/sessions/")
	if chatroomID == "" {
		writeError(w, http.StatusBadRequest, "chatroom_id required")
		return
	}

	// Get group info
	var name string
	var memberCount int
	h.db.QueryRow("SELECT name, member_count FROM groups WHERE chatroom_id = ?", chatroomID).Scan(&name, &memberCount)
	if name == "" {
		name = chatroomID
	}

	// Get tags
	tagRows, err := h.db.Query("SELECT tag FROM group_tags WHERE chatroom_id = ?", chatroomID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer tagRows.Close()

	tags := []string{}
	for tagRows.Next() {
		var tag string
		if err := tagRows.Scan(&tag); err != nil {
			continue
		}
		tags = append(tags, tag)
	}

	// Get stats
	var messageCount, uniqueSenders int
	h.db.QueryRow(
		"SELECT COUNT(*), COUNT(DISTINCT sender) FROM messages WHERE chatroom_id = ?",
		chatroomID,
	).Scan(&messageCount, &uniqueSenders)

	// Get recent messages
	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	msgRows, err := h.db.Query(`
		SELECT chatroom_id, local_id, sender, content, timestamp, type, date
		FROM messages
		WHERE chatroom_id = ?
		ORDER BY timestamp DESC
		LIMIT ?`,
		chatroomID, limit,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer msgRows.Close()

	recent := []models.Message{}
	for msgRows.Next() {
		var m models.Message
		if err := msgRows.Scan(&m.ChatroomID, &m.LocalID, &m.Sender, &m.Content, &m.Timestamp, &m.Type, &m.Date); err != nil {
			continue
		}
		recent = append(recent, m)
	}

	// Get daily history
	dailyRows, err := h.db.Query(`
		SELECT date, message_count, unique_senders
		FROM daily_stats
		WHERE chatroom_id = ?
		ORDER BY date DESC
		LIMIT 30`,
		chatroomID,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer dailyRows.Close()

	dailyHistory := []models.DailyEntry{}
	for dailyRows.Next() {
		var d models.DailyEntry
		if err := dailyRows.Scan(&d.Date, &d.MessageCount, &d.UniqueSenders); err != nil {
			continue
		}
		dailyHistory = append(dailyHistory, d)
	}

	writeJSON(w, http.StatusOK, models.GroupDetail{
		OK:           true,
		ChatroomID:   chatroomID,
		Date:         "",
		Stats: &models.GroupStats{
			MessageCount:  messageCount,
			UniqueSenders: uniqueSenders,
		},
		Recent:       recent,
		DailyHistory: dailyHistory,
	})
}
