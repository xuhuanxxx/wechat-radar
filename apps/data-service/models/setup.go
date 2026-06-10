package models

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
	// Extended fields for web frontend compatibility
	DemoMode         bool   `json:"demoMode"`
	PrivacyConfirmed bool   `json:"privacyConfirmed"`
	DefaultSyncDays  int    `json:"defaultSyncDays"`
	Source           string `json:"source"`
	LarkCliPath      string `json:"larkCliPath"`
	OpenApiKey       string `json:"openApiKey"`
	SetupCompleted   bool   `json:"setupCompleted"`
}

// SetupStatus represents the setup status (GET /api/setup)
type SetupStatus struct {
	OK         bool           `json:"ok"`
	DataDir    string         `json:"dataDir"`
	Configured bool           `json:"configured"`
	Config     Config         `json:"config"`
	Checks     SetupChecks    `json:"checks"`
	Error      string         `json:"error,omitempty"`
}

// SetupChecks represents environment checks
type SetupChecks struct {
	LarkInstalled      bool   `json:"larkInstalled"`
	LarkAuthenticated  bool   `json:"larkAuthenticated"`
	LarkError          string `json:"larkError,omitempty"`
}

// SetupRequest represents a setup request (POST /api/setup)
type SetupRequest struct {
	MyNicknames      []string       `json:"myNicknames"`
	DefaultRange     string         `json:"defaultRange"`
	Port             int            `json:"port,omitempty"`
	AutoSyncInterval int            `json:"autoSyncInterval,omitempty"`
	LarkChatFilter   LarkChatFilter `json:"larkChatFilter,omitempty"`
	// Extended fields for web frontend compatibility
	DemoMode         bool   `json:"demoMode"`
	PrivacyConfirmed bool   `json:"privacyConfirmed"`
	DefaultSyncDays  int    `json:"defaultSyncDays,omitempty"`
	Source           string `json:"source"`
	LarkCliPath      string `json:"larkCliPath,omitempty"`
	OpenApiKey       string `json:"openApiKey,omitempty"`
}

// SetupResponse represents a setup response (POST /api/setup)
type SetupResponse struct {
	OK         bool   `json:"ok"`
	Configured bool   `json:"configured,omitempty"`
	Message    string `json:"message,omitempty"`
	Error      string `json:"error,omitempty"`
}

// ConfigResponse represents the config API response
type ConfigResponse struct {
	OK     bool   `json:"ok"`
	Config Config `json:"config"`
}
