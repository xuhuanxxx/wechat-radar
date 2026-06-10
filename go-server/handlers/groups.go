package handlers

import (
	"net/http"

	"go-server/models"
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

	groups := []models.GroupWithTags{}
	for rows.Next() {
		var g models.GroupWithTags
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

	writeJSON(w, http.StatusOK, models.GroupsResponse{
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

	writeJSON(w, http.StatusOK, models.GroupDetailResponse{
		OK:          true,
		ChatroomID:  chatroomID,
		Name:        name,
		MemberCount: memberCount,
		Tags:        tags,
	})
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

	writeJSON(w, http.StatusOK, models.GroupTagsResponse{
		OK:   true,
		Tags: tags,
	})
}

func (h *Handlers) addGroupTag(w http.ResponseWriter, r *http.Request) {
	var req models.GroupTagRequest
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

	writeJSON(w, http.StatusOK, models.GroupTagResponse{
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
