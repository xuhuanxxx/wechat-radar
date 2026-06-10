package db

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/xuhuanxxx/wechat-radar/apps/data-service/models"
)

// ConfigManager manages the config file
type ConfigManager struct {
	path string
	mu   sync.RWMutex
}

// NewConfigManager creates a new config manager
func NewConfigManager(path string) *ConfigManager {
	return &ConfigManager{path: path}
}

// Load loads the config from file
func (cm *ConfigManager) Load() (*models.Config, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	data, err := os.ReadFile(cm.path)
	if err != nil {
		if os.IsNotExist(err) {
			return &models.Config{
				MyNicknames:      []string{},
				DefaultRange:     "7d",
				Port:             8787,
				AutoSyncInterval: 0,
				LarkChatFilter: models.LarkChatFilter{
					Mode:      "all",
					Allowlist: []string{},
					Blocklist: []string{},
				},
			}, nil
		}
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg models.Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	// Set defaults
	if cfg.DefaultRange == "" {
		cfg.DefaultRange = "7d"
	}
	if cfg.Port == 0 {
		cfg.Port = 8787
	}

	return &cfg, nil
}

// Save saves the config to file
func (cm *ConfigManager) Save(cfg *models.Config) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	if err := os.WriteFile(cm.path, data, 0644); err != nil {
		return fmt.Errorf("write config: %w", err)
	}

	return nil
}

// IsConfigured checks if the app has been set up
func (cm *ConfigManager) IsConfigured() bool {
	cfg, err := cm.Load()
	if err != nil {
		return false
	}
	return len(cfg.MyNicknames) > 0
}

// DataDir returns the directory containing the config file
func (cm *ConfigManager) DataDir() string {
	// Extract directory from path
	for i := len(cm.path) - 1; i >= 0; i-- {
		if cm.path[i] == '/' || cm.path[i] == '\\' {
			return cm.path[:i]
		}
	}
	return "."
}
