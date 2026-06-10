package sync

import (
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"sync"
	"time"

	"go-server/db"
	"go-server/models"
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
	Running   bool                  `json:"running"`
	StartedAt time.Time             `json:"started_at,omitempty"`
	CompletedAt time.Time           `json:"completed_at,omitempty"`
	Progress  map[string]int        `json:"progress"`
	Results   map[string]models.SyncResult `json:"results,omitempty"`
	Error     string                `json:"error,omitempty"`
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
	cmd := exec.Command("lark", "--version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("lark-cli not found: %w", err)
	}
	return nil
}

// CheckLarkAuth verifies lark-cli authentication
func (e *Engine) CheckLarkAuth() error {
	cmd := exec.Command("lark", "doctor", "--json")
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

func (e *Engine) fetchChats() ([]models.LarkChat, error) {
	cmd := exec.Command("lark", "im", "chat", "list", "--format", "json")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("lark im chat list failed: %w", err)
	}

	var resp models.LarkChatListResponse
	if err := json.Unmarshal(output, &resp); err != nil {
		return nil, fmt.Errorf("parse chat list: %w", err)
	}

	if !resp.OK && resp.Error != nil {
		return nil, fmt.Errorf("lark API error: %s", resp.Error.Message)
	}

	var chats []models.LarkChat
	if resp.Data != nil {
		chats = append(chats, resp.Data.Items...)
		chats = append(chats, resp.Data.Chats...)
	}
	chats = append(chats, resp.Chats...)

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

func (e *Engine) doSyncChat(chatID string, daysBack int) (models.SyncResult, error) {
	result := models.SyncResult{}

	// Calculate start time
	startTime := time.Now().AddDate(0, 0, -daysBack).Unix()

	// Fetch messages from lark-cli
	cmd := exec.Command("lark", "im", "message", "list", "--chat-id", chatID, "--format", "json")
	output, err := cmd.Output()
	if err != nil {
		return result, fmt.Errorf("fetch messages: %w", err)
	}

	var resp models.LarkMessagesResponse
	if err := json.Unmarshal(output, &resp); err != nil {
		return result, fmt.Errorf("parse messages: %w", err)
	}

	if !resp.OK && resp.Error != nil {
		return result, fmt.Errorf("lark API error: %s", resp.Error.Message)
	}

	var messages []models.LarkMessage
	if resp.Data != nil {
		messages = append(messages, resp.Data.Items...)
		messages = append(messages, resp.Data.Messages...)
	}
	messages = append(messages, resp.Messages...)

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
	// Try to get chat info from lark
	cmd := exec.Command("lark", "im", "chat", "get", "--chat-id", chatID, "--format", "json")
	output, err := cmd.Output()
	if err != nil {
		return
	}

	var chat models.LarkChat
	if err := json.Unmarshal(output, &chat); err != nil {
		return
	}

	_, _ = e.db.Exec(`
		INSERT INTO groups (chatroom_id, name, member_count)
		VALUES (?, ?, ?)
		ON CONFLICT(chatroom_id) DO UPDATE SET
			name = excluded.name,
			member_count = excluded.member_count
	`, chatID, chat.Name, chat.MemberCount)
}

func parseMessageTime(msg models.LarkMessage) int64 {
	if msg.CreateTime != "" {
		if ts, err := parseInt(msg.CreateTime); err == nil {
			return ts / 1000 // Lark uses milliseconds
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
