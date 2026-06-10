package models

// AIClassifyRequest represents an AI classification request
type AIClassifyRequest struct {
	ChatroomIDs []string `json:"chatroom_ids"`
	Date        string   `json:"date,omitempty"`
}

// AIClassifyResult represents a single classification result
type AIClassifyResult struct {
	ChatroomID string `json:"chatroom_id"`
	Name       string `json:"name,omitempty"`
	Category   string `json:"category"`
	Confidence float64 `json:"confidence"`
	Reason     string `json:"reason,omitempty"`
}

// AIClassifyResponse represents the AI classify API response
type AIClassifyResponse struct {
	OK          bool               `json:"ok"`
	Results     []AIClassifyResult `json:"results"`
	Classified  int                `json:"classified"`
}
