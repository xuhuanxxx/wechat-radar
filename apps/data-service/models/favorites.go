package models

// Favorite represents a saved favorite
type Favorite struct {
	ID          int    `json:"id"`
	ChatroomID  string `json:"chatroom_id"`
	MessageID   string `json:"message_id"`
	Sender      string `json:"sender"`
	Content     string `json:"content"`
	Date        string `json:"date"`
	CreatedAt   string `json:"created_at,omitempty"`
}

// FavoriteToggleRequest represents a favorite toggle request
type FavoriteToggleRequest struct {
	ChatroomID string `json:"chatroom_id"`
	MessageID  string `json:"message_id"`
}

// FavoriteToggleResponse represents a favorite toggle response
type FavoriteToggleResponse struct {
	OK        bool `json:"ok"`
	Favorited bool `json:"favorited"`
}

// FavoritesResponse represents the favorites API response
type FavoritesResponse struct {
	OK        bool       `json:"ok"`
	Favorites []Favorite `json:"favorites"`
}
