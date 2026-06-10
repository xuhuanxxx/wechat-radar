package models

// Topic represents a detected topic
type Topic struct {
	ID         int     `json:"id"`
	ChatroomID string  `json:"chatroom_id"`
	Date       string  `json:"date"`
	Topic      string  `json:"topic"`
	Category   string  `json:"category"`
	Confidence float64 `json:"confidence"`
	MessageIDs string  `json:"message_ids,omitempty"`
	CreatedAt  string  `json:"created_at,omitempty"`
}

// TopicCategory represents a topic category summary
type TopicCategory struct {
	Category string  `json:"category"`
	Count    int     `json:"count"`
	Topics   []Topic `json:"topics,omitempty"`
}

// TopicsResponse represents the topics API response
type TopicsResponse struct {
	OK         bool            `json:"ok"`
	ChatroomID string          `json:"chatroom_id,omitempty"`
	Date       string          `json:"date,omitempty"`
	Topics     []Topic         `json:"topics"`
	Categories []TopicCategory `json:"categories,omitempty"`
}

// TopicAnalyzeRequest represents a topic analysis request
type TopicAnalyzeRequest struct {
	ChatroomID string `json:"chatroom_id"`
	Date       string `json:"date"`
}

// TopicAnalyzeResponse represents a topic analysis response
type TopicAnalyzeResponse struct {
	OK     bool    `json:"ok"`
	Topics []Topic `json:"topics"`
}

// TopicDetailResponse represents a single topic detail response
type TopicDetailResponse struct {
	OK    bool  `json:"ok"`
	Topic Topic `json:"topic"`
}

// TopicLinksResponse represents links extracted from topics
type TopicLinksResponse struct {
	OK    bool   `json:"ok"`
	Date  string `json:"date,omitempty"`
	Links []Link `json:"links"`
	Count int    `json:"count"`
}
