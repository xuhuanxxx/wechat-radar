package models

// Message represents a chat message
type Message struct {
	ChatroomID string `json:"chatroom_id"`
	LocalID    string `json:"local_id"`
	Sender     string `json:"sender"`
	SenderName string `json:"sender_name,omitempty"`
	Content    string `json:"content"`
	Time       string `json:"time"`
	Timestamp  int64  `json:"timestamp"`
	Type       string `json:"type"`
	Date       string `json:"date"`
	Source     string `json:"source"`
	Raw        string `json:"raw,omitempty"`
}

// Session represents a chat session
type Session struct {
	Chat        string `json:"chat"`
	ChatroomID  string `json:"chatroom_id"`
	Name        string `json:"name"`
	ChatType    string `json:"chat_type"`
	IsGroup     bool   `json:"is_group"`
	LastMsgType string `json:"last_msg_type"`
	LastSender  string `json:"last_sender"`
	Summary     string `json:"summary"`
	Time        string `json:"time"`
	Timestamp   int64  `json:"timestamp"`
	Unread      int    `json:"unread"`
	Username    string `json:"username"`
	MessageCount int   `json:"message_count"`
	UniqueSenders int  `json:"unique_senders"`
	LastActive   int64 `json:"last_active"`
}

// DailyStats represents daily message statistics
type DailyStats struct {
	ChatroomID string            `json:"chatroom_id"`
	Date       string            `json:"date"`
	Total      int               `json:"total"`
	TopSenders []SenderCount     `json:"top_senders"`
	ByHour     []HourCount       `json:"by_hour"`
}

// SenderCount represents a sender's message count
type SenderCount struct {
	Sender string `json:"sender"`
	Count  int    `json:"count"`
}

// HourCount represents message count by hour
type HourCount struct {
	Hour  int `json:"hour"`
	Count int `json:"count"`
}

// TypeCount represents message count by type
type TypeCount struct {
	Type  string `json:"type"`
	Count int    `json:"count"`
}

// GroupDetail represents a group's detail response
type GroupDetail struct {
	OK           bool          `json:"ok"`
	ChatroomID   string        `json:"chatroom_id"`
	Date         string        `json:"date"`
	Stats        *GroupStats   `json:"stats"`
	Recent       []Message     `json:"recent"`
	DailyHistory []DailyEntry  `json:"daily_history"`
	SyncState    *SyncState    `json:"sync_state"`
	SyncedDates  []string      `json:"synced_dates"`
}

// GroupStats represents statistics for a group on a specific date
type GroupStats struct {
	Chat          string        `json:"chat"`
	Total         int           `json:"total"`
	MessageCount  int           `json:"message_count"`
	UniqueSenders int           `json:"unique_senders"`
	ByHour        []HourCount   `json:"by_hour"`
	ByType        []TypeCount   `json:"by_type"`
	TopSenders    []SenderCount `json:"top_senders"`
}

// DailyEntry represents a daily history entry
type DailyEntry struct {
	Date          string `json:"date"`
	Total         int    `json:"total"`
	MessageCount  int    `json:"message_count"`
	UniqueSenders int    `json:"unique_senders"`
}

// SyncState represents the sync state for a chatroom
type SyncState struct {
	ChatroomID     string `json:"chatroom_id"`
	Source         string `json:"source"`
	LastSyncedAt   int64  `json:"last_synced_at"`
	TotalMessages  int    `json:"total_messages"`
	Status         string `json:"status"`
}
