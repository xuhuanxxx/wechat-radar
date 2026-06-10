package handlers

import (
	"github.com/xuhuanxxx/wechat-radar/apps/data-service/api"
	"fmt"
	"net/http"
)

// Search searches messages
func (h *Handlers) Search(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	query := r.URL.Query().Get("q")
	if query == "" {
		writeJSON(w, http.StatusOK, api.SearchResponse{
			OK:      true,
			Results: []api.SearchResult{},
			Count:   0,
		})
		return
	}

	chatroomID := r.URL.Query().Get("chatroom_id")
	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := parseInt(l); err == nil && parsed > 0 {
			limit = int(parsed)
		}
	}

	sqlQuery := `
		SELECT m.chatroom_id, COALESCE(g.name, m.chatroom_id), m.local_id, m.sender, m.content, m.date, m.timestamp
		FROM messages m
		LEFT JOIN groups g ON m.chatroom_id = g.chatroom_id
		WHERE m.content LIKE ?
	`
	args := []interface{}{"%" + query + "%"}

	if chatroomID != "" {
		sqlQuery += " AND m.chatroom_id = ?"
		args = append(args, chatroomID)
	}
	sqlQuery += " ORDER BY m.timestamp DESC LIMIT ?"
	args = append(args, limit)

	rows, err := h.db.Query(sqlQuery, args...)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()

	results := []api.SearchResult{}
	for rows.Next() {
		var res api.SearchResult
		if err := rows.Scan(&res.ChatroomID, &res.ChatName, &res.MessageID, &res.Sender, &res.Content, &res.Date, &res.Timestamp); err != nil {
			continue
		}
		results = append(results, res)
	}

	writeJSON(w, http.StatusOK, api.SearchResponse{
		OK:      true,
		Results: results,
		Count:   len(results),
	})
}

func parseInt(s string) (int64, error) {
	var result int64
	for _, c := range s {
		if c < '0' || c > '9' {
			break
		}
		result = result*10 + int64(c-'0')
	}
	if result == 0 {
		return 0, fmt.Errorf("no digits found")
	}
	return result, nil
}
