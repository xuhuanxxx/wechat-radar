package models

import "time"

// StatsResponse represents the stats API response
type StatsResponse struct {
	OK             bool              `json:"ok"`
	Range          string            `json:"range"`
	Window         TimeWindow        `json:"window"`
	Cards          CardsData         `json:"cards"`
	Trend          TrendData         `json:"trend"`
	ActiveGroups   []ActiveGroup     `json:"active_groups"`
	Categories     []CategoryStat    `json:"categories"`
	Intelligence   interface{}       `json:"intelligence"`
	SidebarCounts  SidebarCounts     `json:"sidebar_counts"`
}

// TimeWindow represents a time range
type TimeWindow struct {
	Since         string    `json:"since"`
	Until         string    `json:"until"`
	Days          int       `json:"days"`
	StartDate     string    `json:"start_date"`
	EndDate       string    `json:"end_date"`
	StartDateTime time.Time `json:"-"`
	EndDateTime   time.Time `json:"-"`
}

// CardsData represents the dashboard cards data
type CardsData struct {
	ActiveGroups  int `json:"active_groups"`
	TotalGroups   int `json:"total_groups"`
	TotalMessages int `json:"total_messages"`
	UniqueSenders int `json:"unique_senders"`
	Mentions      int `json:"mentions"`
	SilentGroups  int `json:"silent_groups"`
	AvgPerGroup   int `json:"avg_per_group"`
}

// TrendData represents the trend chart data
type TrendData struct {
	Data   []TrendPoint `json:"data"`
	Labels []string     `json:"labels"`
	Values []int        `json:"values"`
	Peak   TrendPoint   `json:"peak"`
	Avg    float64      `json:"avg"`
	Total  int          `json:"total"`
}

// TrendPoint represents a single trend data point
type TrendPoint struct {
	Date  string `json:"date"`
	Count int    `json:"count"`
}

// ActiveGroup represents an active group in stats
type ActiveGroup struct {
	ChatroomID  string        `json:"chatroom_id"`
	Name        string        `json:"name"`
	Summary     string        `json:"summary"`
	Total       int           `json:"total"`
	MessageCount int          `json:"message_count"`
	SenderCount  int          `json:"sender_count"`
	TopSenders  []SenderCount `json:"top_senders"`
}

// CategoryStat represents a category statistic
type CategoryStat struct {
	ID           int    `json:"id"`
	Name         string `json:"name"`
	Category     string `json:"category"`
	Color        string `json:"color"`
	Emoji        string `json:"emoji"`
	GroupCount   int    `json:"group_count"`
	MessageCount int    `json:"message_count"`
	Count        int    `json:"count"`
}

// SidebarCounts represents sidebar count data
type SidebarCounts struct {
	All       int `json:"all"`
	Favorites int `json:"favorites"`
	Unsorted  int `json:"unsorted"`
}

// SessionsResponse represents the sessions API response (matches web frontend)
type SessionsResponse struct {
	OK         bool           `json:"ok"`
	Total      int            `json:"total"`
	Groups     []SessionGroup `json:"groups"`
	Categories []CategoryInfo `json:"categories"`
}

// SessionGroup represents a group in sessions response (matches web frontend Group type)
type SessionGroup struct {
	ChatroomID string `json:"chatroom_id"`
	Name       string `json:"name"`
	Summary    string `json:"summary"`
	Time       string `json:"time"`
	Timestamp  int64  `json:"timestamp"`
	Unread     int    `json:"unread"`
	IsFavorite bool   `json:"is_favorite"`
	GroupIDs   []int  `json:"group_ids"`
}

// CategoryInfo represents category information
type CategoryInfo struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color"`
	Emoji string `json:"emoji"`
}
