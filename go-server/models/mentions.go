package models

// Mention represents a mention in a message
type Mention struct {
	ID         int    `json:"id"`
	ChatroomID string `json:"chatroom_id"`
	MessageID  string `json:"message_id"`
	Mentioned  string `json:"mentioned"`
	Mentioner  string `json:"mentioner"`
	Date       string `json:"date"`
	CreatedAt  string `json:"created_at,omitempty"`
}

// MentionsResponse represents the mentions API response
type MentionsResponse struct {
	OK       bool      `json:"ok"`
	Mentions []Mention `json:"mentions"`
}

// MentionStats represents mention statistics
type MentionStats struct {
	OK           bool              `json:"ok"`
	Mentioned    map[string]int    `json:"mentioned"`
	MentionedBy  map[string][]string `json:"mentioned_by,omitempty"`
}
