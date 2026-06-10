package handlers

import (
	"net/http"

	"go-server/models"
)

// Favorites returns all favorites
func (h *Handlers) Favorites(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	chatroomID := r.URL.Query().Get("chatroom_id")

	query := "SELECT id, chatroom_id, message_id, sender, content, date FROM favorites WHERE 1=1"
	args := []interface{}{}

	if chatroomID != "" {
		query += " AND chatroom_id = ?"
		args = append(args, chatroomID)
	}
	query += " ORDER BY created_at DESC"

	rows, err := h.db.Query(query, args...)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()

	favorites := []models.Favorite{}
	for rows.Next() {
		var f models.Favorite
		if err := rows.Scan(&f.ID, &f.ChatroomID, &f.MessageID, &f.Sender, &f.Content, &f.Date); err != nil {
			continue
		}
		favorites = append(favorites, f)
	}

	writeJSON(w, http.StatusOK, models.FavoritesResponse{
		OK:        true,
		Favorites: favorites,
	})
}

// ToggleFavorite toggles a message as favorite
func (h *Handlers) ToggleFavorite(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req models.FavoriteToggleRequest
	if err := parseJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	if req.ChatroomID == "" || req.MessageID == "" {
		writeError(w, http.StatusBadRequest, "chatroom_id and message_id required")
		return
	}

	// Check if already favorited
	var existingID int
	err := h.db.QueryRow(
		"SELECT id FROM favorites WHERE chatroom_id = ? AND message_id = ?",
		req.ChatroomID, req.MessageID,
	).Scan(&existingID)

	if err == nil {
		// Remove favorite
		_, err = h.db.Exec(
			"DELETE FROM favorites WHERE chatroom_id = ? AND message_id = ?",
			req.ChatroomID, req.MessageID,
		)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, models.FavoriteToggleResponse{
			OK:        true,
			Favorited: false,
		})
		return
	}

	// Get message content
	var sender, content, date string
	err = h.db.QueryRow(
		"SELECT sender, content, date FROM messages WHERE chatroom_id = ? AND local_id = ?",
		req.ChatroomID, req.MessageID,
	).Scan(&sender, &content, &date)
	if err != nil {
		writeError(w, http.StatusNotFound, "Message not found")
		return
	}

	// Add favorite
	_, err = h.db.Exec(
		"INSERT INTO favorites (chatroom_id, message_id, sender, content, date) VALUES (?, ?, ?, ?, ?)",
		req.ChatroomID, req.MessageID, sender, content, date,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, models.FavoriteToggleResponse{
		OK:        true,
		Favorited: true,
	})
}
