package sync

import "go-server/models"

// ShouldSyncChat checks if a chat should be synced based on filter config
func (e *Engine) ShouldSyncChat(chat models.LarkChat, cfg *models.Config) bool {
	if cfg == nil {
		return true
	}
	filter := cfg.LarkChatFilter
	if filter.Mode == "blocklist" {
		for _, id := range filter.Blocklist {
			if id == chat.ChatID {
				return false
			}
		}
		return true
	}
	if filter.Mode == "allowlist" {
		for _, id := range filter.Allowlist {
			if id == chat.ChatID {
				return true
			}
		}
		return false
	}
	return true
}
