package handlers

import (
	"net/http"
	"regexp"

	"github.com/xuhuanxxx/wechat-radar/apps/data-service/models"
)

var urlRegex = regexp.MustCompile(`https?://[^\s<>"{}|\\^\[\]]+`)

// Links returns detected links
func (h *Handlers) Links(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	chatroomID := r.URL.Query().Get("chatroom_id")
	date := r.URL.Query().Get("date")

	query := "SELECT id, chatroom_id, message_id, url, title, date FROM message_links WHERE 1=1"
	args := []interface{}{}

	if chatroomID != "" {
		query += " AND chatroom_id = ?"
		args = append(args, chatroomID)
	}
	if date != "" {
		query += " AND date = ?"
		args = append(args, date)
	}
	query += " ORDER BY date DESC LIMIT 200"

	rows, err := h.db.Query(query, args...)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()

	links := []models.Link{}
	for rows.Next() {
		var l models.Link
		if err := rows.Scan(&l.ID, &l.ChatroomID, &l.MessageID, &l.URL, &l.Title, &l.Date); err != nil {
			continue
		}
		links = append(links, l)
	}

	writeJSON(w, http.StatusOK, models.LinksResponse{
		OK:    true,
		Links: links,
	})
}

// AnalyzeLinks extracts links from messages
func (h *Handlers) AnalyzeLinks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req models.LinkAnalyzeRequest
	if err := parseJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	if req.ChatroomID == "" || req.Date == "" {
		writeError(w, http.StatusBadRequest, "chatroom_id and date required")
		return
	}

	links, err := h.extractLinks(req.ChatroomID, req.Date)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, models.LinkAnalyzeResponse{
		OK:    true,
		Links: links,
	})
}

func (h *Handlers) extractLinks(chatroomID, date string) ([]models.Link, error) {
	rows, err := h.db.Query(
		"SELECT local_id, content FROM messages WHERE chatroom_id = ? AND date = ?",
		chatroomID, date,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	links := []models.Link{}
	for rows.Next() {
		var msgID, content string
		if err := rows.Scan(&msgID, &content); err != nil {
			continue
		}
		urls := urlRegex.FindAllString(content, -1)
		for _, url := range urls {
			links = append(links, models.Link{
				ChatroomID: chatroomID,
				MessageID:  msgID,
				URL:        url,
				Date:       date,
			})
		}
	}

	return links, nil
}
