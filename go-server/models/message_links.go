package models

// MessageLink represents a raw or resolved message link
type MessageLink struct {
	ID         int    `json:"id"`
	ChatroomID string `json:"chatroom_id"`
	MessageID  string `json:"message_id"`
	URL        string `json:"url"`
	Title      string `json:"title,omitempty"`
	Summary    string `json:"summary,omitempty"`
	Date       string `json:"date"`
	Resolved   bool   `json:"resolved"`
	CreatedAt  string `json:"created_at,omitempty"`
}

// MessageLinksRawResponse represents the raw links API response
type MessageLinksRawResponse struct {
	OK    bool          `json:"ok"`
	Links []MessageLink `json:"links"`
	Date  string        `json:"date,omitempty"`
}

// MessageLinksBackfillResponse represents the backfill response
type MessageLinksBackfillResponse struct {
	OK       bool `json:"ok"`
	Inserted int  `json:"inserted"`
}

// MessageLinksResolveResponse represents the resolve response
type MessageLinksResolveResponse struct {
	OK       bool `json:"ok"`
	Resolved int  `json:"resolved"`
}
