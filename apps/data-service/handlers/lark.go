package handlers

import (
	"encoding/json"
	"net/http"
	"os/exec"

	"github.com/xuhuanxxx/wechat-radar/apps/data-service/models"
)

// LarkChats returns Lark chats using new lark-cli shortcut format
func (h *Handlers) LarkChats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	cmd := exec.Command("lark-cli", "im", "+chat-list", "--json")
	output, err := cmd.Output()
	if err != nil {
		writeJSON(w, http.StatusOK, models.LarkChatsResponse{
			OK:     false,
			Chats:  []models.LarkChatItem{},
			Filter: models.LarkChatFilter{},
		})
		return
	}

	var resp struct {
		OK       bool `json:"ok"`
		Identity string `json:"identity,omitempty"`
		Data     *struct {
			Chats   []models.LarkChat `json:"chats,omitempty"`
			HasMore bool              `json:"has_more,omitempty"`
		} `json:"data,omitempty"`
		Error *models.LarkError `json:"error,omitempty"`
	}
	if err := json.Unmarshal(output, &resp); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to parse lark response")
		return
	}

	if !resp.OK && resp.Error != nil {
		writeJSON(w, http.StatusOK, models.LarkChatsResponse{
			OK:     false,
			Chats:  []models.LarkChatItem{},
			Filter: models.LarkChatFilter{},
		})
		return
	}

	cfg, _ := h.config.Load()
	filter := cfg.LarkChatFilter

	chats := []models.LarkChatItem{}
	var allChats []models.LarkChat
	if resp.Data != nil {
		allChats = resp.Data.Chats
	}

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

// LarkMessages returns messages from a Lark chat using new shortcut format
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

	cmd := exec.Command("lark-cli", "im", "+chat-messages-list", "--chat-id", chatID, "--json")
	output, err := cmd.Output()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to fetch messages")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(output)
}
