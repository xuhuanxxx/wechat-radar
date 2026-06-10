package handlers

import (
	"encoding/json"
	"net/http"
	"os/exec"

	"go-server/models"
)

// LarkChats returns Lark chats
func (h *Handlers) LarkChats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	cmd := exec.Command("lark", "im", "chat", "list", "--format", "json")
	output, err := cmd.Output()
	if err != nil {
		writeJSON(w, http.StatusOK, models.LarkChatsResponse{
			OK:     false,
			Chats:  []models.LarkChatItem{},
			Filter: models.LarkChatFilter{},
		})
		return
	}

	var resp models.LarkChatListResponse
	if err := json.Unmarshal(output, &resp); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to parse lark response")
		return
	}

	cfg, _ := h.config.Load()
	filter := cfg.LarkChatFilter

	chats := []models.LarkChatItem{}
	var allChats []models.LarkChat
	if resp.Data != nil {
		allChats = append(allChats, resp.Data.Items...)
		allChats = append(allChats, resp.Data.Chats...)
	}
	allChats = append(allChats, resp.Chats...)

	for _, chat := range allChats {
		filtered := !h.syncEngine.ShouldSyncChat(chat, cfg)
		chats = append(chats, models.LarkChatItem{
			ID:          chat.ChatID,
			Name:        chat.Name,
			MemberCount: chat.MemberCount,
			Filtered:    filtered,
		})
	}

	writeJSON(w, http.StatusOK, models.LarkChatsResponse{
		OK:     true,
		Chats:  chats,
		Filter: filter,
	})
}

// LarkMessages returns messages from a Lark chat
func (h *Handlers) LarkMessages(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	chatID := r.URL.Query().Get("chat_id")
	if chatID == "" {
		writeError(w, http.StatusBadRequest, "chat_id required")
		return
	}

	cmd := exec.Command("lark", "im", "message", "list", "--chat-id", chatID, "--format", "json")
	output, err := cmd.Output()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to fetch messages")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(output)
}
