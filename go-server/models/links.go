package models

// Link represents a detected link
type Link struct {
	ID         int    `json:"id"`
	ChatroomID string `json:"chatroom_id"`
	MessageID  string `json:"message_id"`
	URL        string `json:"url"`
	Title      string `json:"title,omitempty"`
	Date       string `json:"date"`
	CreatedAt  string `json:"created_at,omitempty"`
}

// LinksResponse represents the links API response
type LinksResponse struct {
	OK    bool   `json:"ok"`
	Links []Link `json:"links"`
}

// LinkAnalyzeRequest represents a link analysis request
type LinkAnalyzeRequest struct {
	ChatroomID string `json:"chatroom_id"`
	Date       string `json:"date"`
}

// LinkAnalyzeResponse represents a link analysis response
type LinkAnalyzeResponse struct {
	OK    bool   `json:"ok"`
	Links []Link `json:"links"`
}
