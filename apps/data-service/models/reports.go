package models

// ReportRequest represents a report generation request
type ReportRequest struct {
	ChatroomID string `json:"chatroom_id"`
	Date       string `json:"date,omitempty"`
	Range      string `json:"range,omitempty"`
}

// ReportResponse represents a report generation response
type ReportResponse struct {
	OK      bool   `json:"ok"`
	Report  string `json:"report"`
	Error   string `json:"error,omitempty"`
}

// IntelligenceBrief represents the dashboard intelligence brief
type IntelligenceBrief struct {
	OK           bool   `json:"ok"`
	Summary      string `json:"summary"`
	Highlights   []string `json:"highlights,omitempty"`
	Risks        []string `json:"risks,omitempty"`
	Opportunities []string `json:"opportunities,omitempty"`
}
