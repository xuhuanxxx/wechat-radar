package sync

import (
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"sync"
	"time"

	"github.com/xuhuanxxx/wechat-radar/apps/data-service/db"
	"github.com/xuhuanxxx/wechat-radar/apps/data-service/models"
)

// Engine handles Lark data synchronization
type Engine struct {
	db     *db.DB
	config *db.ConfigManager
	mu     sync.RWMutex
	status *SyncStatus
}

// SyncStatus tracks current sync state
type SyncStatus struct {
	Running     bool                      `json:"running"`
	StartedAt   time.Time                 `json:"started_at,omitempty"`
	CompletedAt time.Time                 `json:"completed_at,omitempty"`
	Progress    map[string]int            `json:"progress"`
	Results     map[string]models.SyncResult `json:"results,omitempty"`
	Error       string                    `json:"error,omitempty"`
}

// NewEngine creates a new sync engine
func NewEngine(database *db.DB, config *db.ConfigManager) *Engine {
	return &Engine{
		db:     database,
		config: config,
		status: &SyncStatus{
			Progress: make(map[string]int),
			Results:  make(map[string]models.SyncResult),
		},
	}
}

// CheckLarkCLI verifies lark-cli is available
func (e *Engine) CheckLarkCLI() error {
	cmd := exec.Command("lark-cli", "--version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("lark-cli not found: %w", err)
	}
	return nil
}

// CheckLarkAuth verifies lark-cli authentication
func (e *Engine) CheckLarkAuth() error {
	cmd := exec.Command("lark-cli", "doctor")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("lark doctor failed: %w", err)
	}

	var result struct {
		OK     bool `json:"ok"`
		Checks []struct {
			Name    string `json:"name"`
			Status  string `json:"status"`
			Message string `json:"message,omitempty"`
		} `json:"checks,omitempty"`
	}
	if err := json.Unmarshal(output, &result); err != nil {
		return fmt.Errorf("parse doctor output: %w", err)
	}

	if !result.OK {
		return fmt.Errorf("lark doctor: not ok")
	}

	for _, c := range result.Checks {
		if c.Name == "user_identity" && c.Status != "pass" {
			return fmt.Errorf("lark not authenticated: %s", c.Message)
		}
	}

	return nil
}

// GetStatus returns current sync status
func (e *Engine) GetStatus() *SyncStatus {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.status
}

// SyncChat syncs a single chat by ID
func (e *Engine) SyncChat(chatID string, daysBack int) (models.SyncResult, error) {
	e.mu.Lock()
	if e.status.Running {
		e.mu.Unlock()
		return models.SyncResult{}, fmt.Errorf("sync already in progress")
	}
	e.status.Running = true
	e.status.StartedAt = time.Now()
	e.status.CompletedAt = time.Time{}
	e.status.Progress = make(map[string]int)
	e.status.Results = make(map[string]models.SyncResult)
	e.status.Error = ""
	e.mu.Unlock()

	defer func() {
		e.mu.Lock()
		e.status.Running = false
		e.status.CompletedAt = time.Now()
		e.mu.Unlock()
	}()

	result, err := e.doSyncChat(chatID, daysBack)
	if err != nil {
		e.mu.Lock()
		e.status.Error = err.Error()
		e.mu.Unlock()
	}

	e.mu.Lock()
	e.status.Progress[chatID] = 100
	e.status.Results[chatID] = result
	e.mu.Unlock()

	return result, err
}

// SyncAll syncs all chats for the given days back
func (e *Engine) SyncAll(daysBack int) (map[string]models.SyncResult, error) {
	e.mu.Lock()
	if e.status.Running {
		e.mu.Unlock()
		return nil, fmt.Errorf("sync already in progress")
	}
	e.status.Running = true
	e.status.StartedAt = time.Now()
	e.status.CompletedAt = time.Time{}
	e.status.Progress = make(map[string]int)
	e.status.Results = make(map[string]models.SyncResult)
	e.status.Error = ""
	e.mu.Unlock()

	defer func() {
		e.mu.Lock()
		e.status.Running = false
		e.status.CompletedAt = time.Now()
		e.mu.Unlock()
	}()

	// Get chats from lark-cli
	chats, err := e.fetchChats()
	if err != nil {
		e.mu.Lock()
		e.status.Error = err.Error()
		e.mu.Unlock()
		return nil, err
	}

	results := make(map[string]models.SyncResult)
	var mu sync.Mutex
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 3) // Max 3 concurrent

	for _, chat := range chats {
		wg.Add(1)
		go func(c models.LarkChat) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			e.mu.Lock()
			e.status.Progress[c.ChatID] = 0
			e.mu.Unlock()

			result, err := e.doSyncChat(c.ChatID, daysBack)

			mu.Lock()
			if err != nil {
				result.Error = err.Error()
			}
			results[c.ChatID] = result
			mu.Unlock()

			e.mu.Lock()
			e.status.Progress[c.ChatID] = 100
			e.status.Results[c.ChatID] = result
			e.mu.Unlock()
		}(chat)
	}

	wg.Wait()
	return results, nil
}

// NewChatListResponse represents the new lark-cli +chat-list output
type NewChatListResponse struct {
	OK       bool                `json:"ok"`
	Identity string              `json:"identity,omitempty"`
	Data     *NewChatListData    `json:"data,omitempty"`
	Error    *models.LarkError   `json:"error,omitempty"`
}

type NewChatListData struct {
	Chats   []models.LarkChat `json:"chats,omitempty"`
	HasMore bool              `json:"has_more,omitempty"`
}

func (e *Engine) fetchChats() ([]models.LarkChat, error) {
	cmd := exec.Command("lark-cli", "im", "+chat-list", "--json")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("lark im +chat-list failed: %w", err)
	}

	var resp NewChatListResponse
	if err := json.Unmarshal(output, &resp); err != nil {
		return nil, fmt.Errorf("parse chat list: %w", err)
	}

	if !resp.OK && resp.Error != nil {
		return nil, fmt.Errorf("lark API error: %s", resp.Error.Message)
	}

	var chats []models.LarkChat
	if resp.Data != nil {
		chats = resp.Data.Chats
	}

	// Apply filter
	cfg, _ := e.config.Load()
	filtered := []models.LarkChat{}
	for _, chat := range chats {
		if e.shouldSyncChat(chat, cfg) {
			filtered = append(filtered, chat)
		}
	}

	return filtered, nil
}

func (e *Engine) shouldSyncChat(chat models.LarkChat, cfg *models.Config) bool {
	if cfg == nil {
		return true
	}
	filter := cfg.LarkChatFilter
	if filter.Mode == "blocklist" {
		for _, id := range filter.Blocklist {
			if id == chat.ChatID {
				return false
			}
		}
		return true
	}
	if filter.Mode == "allowlist" {
		for _, id := range filter.Allowlist {
			if id == chat.ChatID {
				return true
			}
		}
		return false
	}
	return true
}

// FetchChats returns the list of chats available for sync
func (e *Engine) FetchChats() ([]models.LarkChat, error) {
	return e.fetchChats()
}

// NewMessagesResponse represents the new lark-cli +chat-messages-list output
type NewMessagesResponse struct {
	OK       bool                 `json:"ok"`
	Identity string               `json:"identity,omitempty"`
	Data     *NewMessagesData     `json:"data,omitempty"`
	Error    *models.LarkError    `json:"error,omitempty"`
}

type NewMessagesData struct {
	Messages  []models.LarkMessage `json:"messages,omitempty"`
	HasMore   bool                 `json:"has_more,omitempty"`
	PageToken string               `json:"page_token,omitempty"`
}

func (e *Engine) doSyncChat(chatID string, daysBack int) (models.SyncResult, error) {
	result := models.SyncResult{}

	// Calculate start time
	startTime := time.Now().AddDate(0, 0, -daysBack).Unix()

	// Fetch messages from lark-cli using new shortcut format
	cmd := exec.Command("lark-cli", "im", "+chat-messages-list", "--chat-id", chatID, "--json")
	output, err := cmd.Output()
	if err != nil {
		return result, fmt.Errorf("fetch messages: %w", err)
	}

	var resp NewMessagesResponse
	if err := json.Unmarshal(output, &resp); err != nil {
		return result, fmt.Errorf("parse messages: %w", err)
	}

	if !resp.OK && resp.Error != nil {
		return result, fmt.Errorf("lark API error: %s", resp.Error.Message)
	}

	var messages []models.LarkMessage
	if resp.Data != nil {
		messages = resp.Data.Messages
	}

	// Filter by time and insert
	tx, err := e.db.Begin()
	if err != nil {
		return result, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT OR IGNORE INTO messages (chatroom_id, local_id, sender, content, timestamp, type, date)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return result, fmt.Errorf("prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, msg := range messages {
		ts := parseMessageTime(msg)
		if ts < startTime {
			continue
		}

		sender := getSenderName(msg)
		content := getMessageContent(msg)
		date := time.Unix(ts, 0).Format("2006-01-02")
		msgType := getMessageType(msg)

		res, err := stmt.Exec(chatID, msg.MessageID, sender, content, ts, msgType, date)
		if err != nil {
			log.Printf("Insert error for %s: %v", msg.MessageID, err)
			continue
		}

		rowsAffected, _ := res.RowsAffected()
		if rowsAffected > 0 {
			result.Inserted++
		} else {
			result.Skipped++
		}
	}

	if err := tx.Commit(); err != nil {
		return result, fmt.Errorf("commit transaction: %w", err)
	}

	// Update sync state
	_, err = e.db.Exec(`
		INSERT INTO sync_state (chatroom_id, last_sync_at, message_count)
		VALUES (?, datetime('now'), ?)
		ON CONFLICT(chatroom_id) DO UPDATE SET
			last_sync_at = excluded.last_sync_at,
			message_count = message_count + excluded.message_count
	`, chatID, result.Inserted)
	if err != nil {
		log.Printf("Update sync state error: %v", err)
	}

	// Update group info
	e.updateGroupInfo(chatID)

	return result, nil
}

func (e *Engine) updateGroupInfo(chatID string) {
	// Try to get chat info from lark using generic API
	cmd := exec.Command("lark-cli", "api", "GET", "/open-apis/im/v1/chats/"+chatID, "--json")
	output, err := cmd.Output()
	if err != nil {
		return
	}

	var resp struct {
		Code int `json:"code"`
		Data struct {
			Avatar      string `json:"avatar"`
			ChatID      string `json:"chat_id"`
			ChatMode    string `json:"chat_mode"`
			Description string `json:"description"`
			Name        string `json:"name"`
			OwnerID     string `json:"owner_id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(output, &resp); err != nil {
		return
	}

	if resp.Data.ChatID == "" {
		return
	}

	_, _ = e.db.Exec(`
		INSERT INTO groups (chatroom_id, name, member_count)
		VALUES (?, ?, ?)
		ON CONFLICT(chatroom_id) DO UPDATE SET
			name = excluded.name,
			member_count = excluded.member_count
	`, chatID, resp.Data.Name, 0)
}

func parseMessageTime(msg models.LarkMessage) int64 {
	// New lark-cli returns create_time as string like "2026-06-10 11:56"
	if msg.CreateTime != "" {
		// Try parsing as ISO/RFC3339 or custom format
		formats := []string{
			time.RFC3339,
			"2006-01-02 15:04",
			"2006-01-02 15:04:05",
		}
		for _, f := range formats {
			if t, err := time.Parse(f, msg.CreateTime); err == nil {
				return t.Unix()
			}
		}
		// Try as timestamp (milliseconds)
		if ts, err := parseInt(msg.CreateTime); err == nil && ts > 1000000000000 {
			return ts / 1000
		}
	}
	if msg.CreateTimeAlt != "" {
		if ts, err := parseInt(msg.CreateTimeAlt); err == nil {
			return ts / 1000
		}
	}
	return time.Now().Unix()
}

func parseInt(s string) (int64, error) {
	var result int64
	for _, c := range s {
		if c < '0' || c > '9' {
			break
		}
		result = result*10 + int64(c-'0')
	}
	if result == 0 {
		return 0, fmt.Errorf("no digits found")
	}
	return result, nil
}

func getSenderName(msg models.LarkMessage) string {
	if msg.Sender != nil {
		if msg.Sender.Name != "" {
			return msg.Sender.Name
		}
		if msg.Sender.SenderName != "" {
			return msg.Sender.SenderName
		}
		if msg.Sender.ID != "" {
			return msg.Sender.ID
		}
	}
	return "unknown"
}

func getMessageContent(msg models.LarkMessage) string {
	if msg.Content != "" {
		return msg.Content
	}
	if msg.Body != nil && msg.Body.Content != "" {
		return msg.Body.Content
	}
	return ""
}

func getMessageType(msg models.LarkMessage) string {
	if msg.MsgType != "" {
		return msg.MsgType
	}
	if msg.MsgTypeAlt != "" {
		return msg.MsgTypeAlt
	}
	return "text"
}
