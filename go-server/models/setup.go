package models

// SetupStatus represents the setup status
type SetupStatus struct {
	OK         bool   `json:"ok"`
	Configured bool   `json:"configured"`
	Error      string `json:"error,omitempty"`
}

// SetupRequest represents a setup request
type SetupRequest struct {
	MyNicknames      []string       `json:"myNicknames"`
	DefaultRange     string         `json:"defaultRange"`
	Port             int            `json:"port,omitempty"`
	AutoSyncInterval int            `json:"autoSyncInterval,omitempty"`
	LarkChatFilter   LarkChatFilter `json:"larkChatFilter,omitempty"`
}

// SetupResponse represents a setup response
type SetupResponse struct {
	OK      bool   `json:"ok"`
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
}

// LarkChatFilter represents the Lark chat filter configuration
type LarkChatFilter struct {
	Mode      string   `json:"mode"`
	Allowlist []string `json:"allowlist,omitempty"`
	Blocklist []string `json:"blocklist,omitempty"`
}

// Config represents the application configuration
type Config struct {
	MyNicknames      []string       `json:"myNicknames"`
	DefaultRange     string         `json:"defaultRange"`
	Port             int            `json:"port"`
	AutoSyncInterval int            `json:"autoSyncInterval"`
	LarkChatFilter   LarkChatFilter `json:"larkChatFilter,omitempty"`
}

// ConfigResponse represents the config API response
type ConfigResponse struct {
	OK     bool   `json:"ok"`
	Config Config `json:"config"`
}
