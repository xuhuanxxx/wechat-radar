package models

// Group represents a chat group
type Group struct {
	ChatroomID  string `json:"chatroom_id"`
	Name        string `json:"name"`
	MemberCount int    `json:"member_count,omitempty"`
	CreatedAt   string `json:"created_at,omitempty"`
}

// GroupTag represents a group tag
type GroupTag struct {
	ID         int    `json:"id"`
	ChatroomID string `json:"chatroom_id"`
	Tag        string `json:"tag"`
	CreatedAt  string `json:"created_at,omitempty"`
}

// GroupWithTags represents a group with its tags
type GroupWithTags struct {
	Group
	Tags []string `json:"tags"`
}

// GroupsResponse represents the groups API response
type GroupsResponse struct {
	OK     bool            `json:"ok"`
	Groups []GroupWithTags `json:"groups"`
}

// GroupDetailResponse represents a single group detail response
type GroupDetailResponse struct {
	OK           bool         `json:"ok"`
	ChatroomID   string       `json:"chatroom_id"`
	Name         string       `json:"name"`
	MemberCount  int          `json:"member_count,omitempty"`
	Tags         []string     `json:"tags"`
	Stats        *GroupStats  `json:"stats,omitempty"`
	Recent       []Message    `json:"recent,omitempty"`
	DailyHistory []DailyEntry `json:"daily_history,omitempty"`
}

// GroupTagRequest represents a group tag request
type GroupTagRequest struct {
	ChatroomID string `json:"chatroom_id"`
	Tag        string `json:"tag"`
}

// GroupTagResponse represents a group tag response
type GroupTagResponse struct {
	OK  bool `json:"ok"`
	Tag string `json:"tag"`
}

// GroupTagsResponse represents the group tags API response
type GroupTagsResponse struct {
	OK   bool     `json:"ok"`
	Tags []string `json:"tags"`
}
