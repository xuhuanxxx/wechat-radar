package handlers

import (
	"github.com/xuhuanxxx/wechat-radar/apps/data-service/api"
	"net/http"
	"strconv"
)

// Groups returns all groups with tags
func (h *Handlers) Groups(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	rows, err := h.db.Query(`
		SELECT g.chatroom_id, g.name, g.member_count, 
			(SELECT GROUP_CONCAT(tag) FROM group_tags WHERE chatroom_id = g.chatroom_id) as tags
		FROM groups g
		ORDER BY g.name
	`)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()

	groups := []api.GroupWithTags{}
	for rows.Next() {
		var g api.GroupWithTags
		var tagsStr string
		if err := rows.Scan(&g.ChatroomID, &g.Name, &g.MemberCount, &tagsStr); err != nil {
			continue
		}
		if tagsStr != "" {
			g.Tags = splitTags(tagsStr)
		} else {
			g.Tags = []string{}
		}
		groups = append(groups, g)
	}

	writeJSON(w, http.StatusOK, api.GroupsResponse{
		OK:     true,
		Groups: groups,
	})
}

// GroupDetail returns a single group detail
func (h *Handlers) GroupDetail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	chatroomID := getPathParam(r, "/api/groups/")
	if chatroomID == "" {
		writeError(w, http.StatusBadRequest, "chatroom_id required")
		return
	}

	date := r.URL.Query().Get("date")
	limitStr := r.URL.Query().Get("limit")
	limit := 50
	if v, err := strconv.Atoi(limitStr); err == nil && v > 0 {
		limit = v
		if limit > 200 {
			limit = 200
		}
	}

	var name string
	var memberCount int
	err := h.db.QueryRow("SELECT name, member_count FROM groups WHERE chatroom_id = ?", chatroomID).Scan(&name, &memberCount)
	if err != nil {
		writeError(w, http.StatusNotFound, "Group not found")
		return
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

	// Build response
	resp := api.GroupDetailResponse{
		OK:          true,
		ChatroomID:  chatroomID,
		Name:        name,
		MemberCount: memberCount,
		Tags:        tags,
	}

	// Get recent messages
	msgQuery := "SELECT chatroom_id, local_id, sender, content, timestamp, type, date FROM messages WHERE chatroom_id = ?"
	msgArgs := []interface{}{chatroomID}
	if date != "" {
		msgQuery += " AND date = ?"
		msgArgs = append(msgArgs, date)
	}
	msgQuery += " ORDER BY timestamp DESC LIMIT ?"
	msgArgs = append(msgArgs, limit)

	msgRows, err := h.db.Query(msgQuery, msgArgs...)
	if err == nil {
		defer msgRows.Close()
		recent := []api.Message{}
		for msgRows.Next() {
			var m api.Message
			if err := msgRows.Scan(&m.ChatroomID, &m.LocalID, &m.Sender, &m.Content, &m.Timestamp, &m.Type, &m.Date); err != nil {
				continue
			}
			recent = append(recent, m)
		}
		resp.Recent = recent
	}

	// Get daily history
	histRows, err := h.db.Query(
		"SELECT date, COUNT(*) as total, COUNT(DISTINCT sender) as unique_senders FROM messages WHERE chatroom_id = ? GROUP BY date ORDER BY date DESC LIMIT 30",
		chatroomID,
	)
	if err == nil {
		defer histRows.Close()
		history := []api.DailyEntry{}
		for histRows.Next() {
			var e api.DailyEntry
			if err := histRows.Scan(&e.Date, &e.Total, &e.UniqueSenders); err != nil {
				continue
			}
			e.MessageCount = e.Total
			history = append(history, e)
		}
		resp.DailyHistory = history
	}

	writeJSON(w, http.StatusOK, resp)
}

// GroupTags handles group tag operations
func (h *Handlers) GroupTags(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.listGroupTags(w, r)
	case http.MethodPost:
		h.addGroupTag(w, r)
	case http.MethodDelete:
		h.removeGroupTag(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

func (h *Handlers) listGroupTags(w http.ResponseWriter, r *http.Request) {
	chatroomID := r.URL.Query().Get("chatroom_id")
	if chatroomID == "" {
		writeError(w, http.StatusBadRequest, "chatroom_id required")
		return
	}

	rows, err := h.db.Query("SELECT tag FROM group_tags WHERE chatroom_id = ?", chatroomID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()

	tags := []string{}
	for rows.Next() {
		var tag string
		if err := rows.Scan(&tag); err != nil {
			continue
		}
		tags = append(tags, tag)
	}

	writeJSON(w, http.StatusOK, api.GroupTagsResponse{
		OK:   true,
		Tags: tags,
	})
}

func (h *Handlers) addGroupTag(w http.ResponseWriter, r *http.Request) {
	var req api.GroupTagRequest
	if err := parseJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	if req.ChatroomID == "" || req.Tag == "" {
		writeError(w, http.StatusBadRequest, "chatroom_id and tag required")
		return
	}

	_, err := h.db.Exec(
		"INSERT OR IGNORE INTO group_tags (chatroom_id, tag) VALUES (?, ?)",
		req.ChatroomID, req.Tag,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, api.GroupTagResponse{
		OK:  true,
		Tag: req.Tag,
	})
}

func (h *Handlers) removeGroupTag(w http.ResponseWriter, r *http.Request) {
	chatroomID := r.URL.Query().Get("chatroom_id")
	tag := r.URL.Query().Get("tag")

	if chatroomID == "" || tag == "" {
		writeError(w, http.StatusBadRequest, "chatroom_id and tag required")
		return
	}

	_, err := h.db.Exec(
		"DELETE FROM group_tags WHERE chatroom_id = ? AND tag = ?",
		chatroomID, tag,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"ok": true,
	})
}

func splitTags(s string) []string {
	if s == "" {
		return []string{}
	}
	parts := []string{}
	for _, p := range split(s, ",") {
		p = trimSpace(p)
		if p != "" {
			parts = append(parts, p)
		}
	}
	return parts
}

func split(s, sep string) []string {
	return splitString(s, sep)
}

func splitString(s, sep string) []string {
	result := []string{}
	start := 0
	for i := 0; i < len(s); i++ {
		if i+len(sep) <= len(s) && s[i:i+len(sep)] == sep {
			result = append(result, s[start:i])
			start = i + len(sep)
			i += len(sep) - 1
		}
	}
	result = append(result, s[start:])
	return result
}

func trimSpace(s string) string {
	start := 0
	for start < len(s) && (s[start] == ' ' || s[start] == '\t' || s[start] == '\n' || s[start] == '\r') {
		start++
	}
	end := len(s)
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\n' || s[end-1] == '\r') {
		end--
	}
	return s[start:end]
}
