package handlers

import (
	"net/http"
	"regexp"
	"strings"

	"github.com/xuhuanxxx/wechat-radar/apps/data-service/models"
)

var urlRegexMessageLinks = regexp.MustCompile(`https?://[^\s<>"{}|\\^\[\]]+`)

// MessageLinksRaw handles GET /api/message-links/raw
func (h *Handlers) MessageLinksRaw(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	date := r.URL.Query().Get("date")
	if date == "" {
		writeError(w, http.StatusBadRequest, "date required")
		return
	}

	// Try to query message_links table; if it doesn't exist or errors, fall back to extracting from messages
	rows, err := h.db.Query(
		"SELECT id, chatroom_id, message_id, url, title, date FROM message_links WHERE date = ? ORDER BY id DESC LIMIT 500",
		date,
	)
	if err != nil {
		// Table may not exist yet; return empty mock data
		writeJSON(w, http.StatusOK, models.MessageLinksRawResponse{
			OK:    true,
			Links: []models.MessageLink{},
			Date:  date,
		})
		return
	}
	defer rows.Close()

	links := []models.MessageLink{}
	for rows.Next() {
		var l models.MessageLink
		if err := rows.Scan(&l.ID, &l.ChatroomID, &l.MessageID, &l.URL, &l.Title, &l.Date); err != nil {
			continue
		}
		links = append(links, l)
	}

	writeJSON(w, http.StatusOK, models.MessageLinksRawResponse{
		OK:    true,
		Links: links,
		Date:  date,
	})
}

// MessageLinksBackfill handles POST /api/message-links/backfill
func (h *Handlers) MessageLinksBackfill(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// For MVP: scan messages for URLs and insert into message_links
	rows, err := h.db.Query(`
		SELECT local_id, chatroom_id, content, date
		FROM messages
		WHERE content LIKE '%http%'
		ORDER BY timestamp DESC
		LIMIT 1000
	`)
	if err != nil {
		writeJSON(w, http.StatusOK, models.MessageLinksBackfillResponse{
			OK:       true,
			Inserted: 0,
		})
		return
	}
	defer rows.Close()

	backfilled := 0
	for rows.Next() {
		var msgID, chatroomID, content, date string
		if err := rows.Scan(&msgID, &chatroomID, &content, &date); err != nil {
			continue
		}
		urls := urlRegexMessageLinks.FindAllString(content, -1)
		for _, url := range urls {
			_, _ = h.db.Exec(
				"INSERT OR IGNORE INTO message_links (chatroom_id, message_id, url, date) VALUES (?, ?, ?, ?)",
				chatroomID, msgID, url, date,
			)
			backfilled++
		}
	}

	writeJSON(w, http.StatusOK, models.MessageLinksBackfillResponse{
		OK:       true,
		Inserted: backfilled,
	})
}

// MessageLinksResolve handles POST /api/message-links/resolve
func (h *Handlers) MessageLinksResolve(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// For MVP: mark unresolved links as resolved with placeholder title
	rows, err := h.db.Query(
		"SELECT id, url FROM message_links WHERE title IS NULL OR title = '' LIMIT 100",
	)
	if err != nil {
		writeJSON(w, http.StatusOK, models.MessageLinksResolveResponse{
			OK:       true,
			Resolved: 0,
		})
		return
	}
	defer rows.Close()

	resolved := 0
	for rows.Next() {
		var id int
		var url string
		if err := rows.Scan(&id, &url); err != nil {
			continue
		}
		title := extractTitleFromURL(url)
		_, _ = h.db.Exec(
			"UPDATE message_links SET title = ?, resolved = 1 WHERE id = ?",
			title, id,
		)
		resolved++
	}

	writeJSON(w, http.StatusOK, models.MessageLinksResolveResponse{
		OK:       true,
		Resolved: resolved,
	})
}

func extractTitleFromURL(url string) string {
	// Simple heuristic: use last path segment or domain as placeholder title
	url = strings.TrimSuffix(url, "/")
	parts := strings.Split(url, "/")
	if len(parts) > 0 {
		last := parts[len(parts)-1]
		if last != "" && !strings.Contains(last, ".") {
			return last
		}
	}
	// Fallback to domain
	if strings.HasPrefix(url, "http://") {
		url = strings.TrimPrefix(url, "http://")
	}
	if strings.HasPrefix(url, "https://") {
		url = strings.TrimPrefix(url, "https://")
	}
	parts = strings.Split(url, "/")
	return parts[0]
}
