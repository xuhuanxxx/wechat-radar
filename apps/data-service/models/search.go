package models

// SearchRequest represents a search request
type SearchRequest struct {
	Query      string `json:"q"`
	ChatroomID string `json:"chatroom_id,omitempty"`
	Limit      int    `json:"limit,omitempty"`
}

// SearchResult represents a single search result
type SearchResult struct {
	ChatroomID string `json:"chatroom_id"`
	ChatName   string `json:"chat_name,omitempty"`
	MessageID  string `json:"message_id"`
	Sender     string `json:"sender"`
	Content    string `json:"content"`
	Date       string `json:"date"`
	Timestamp  int64  `json:"timestamp"`
}

// SearchResponse represents the search API response
type SearchResponse struct {
	OK      bool           `json:"ok"`
	Results []SearchResult `json:"results"`
	Count   int            `json:"count"`
}
