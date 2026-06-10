package models

// NewMessagesResponse represents the new messages API response
type NewMessagesResponse struct {
	OK       bool      `json:"ok"`
	Messages []Message `json:"messages"`
	Count    int       `json:"count"`
}

// RescanResponse represents the rescan API response
type RescanResponse struct {
	OK      bool   `json:"ok"`
	Message string `json:"message,omitempty"`
}

// WXImageResponse represents the wx image proxy response
type WXImageResponse struct {
	OK      bool   `json:"ok"`
	URL     string `json:"url,omitempty"`
	Data    string `json:"data,omitempty"`
	Message string `json:"message,omitempty"`
}
