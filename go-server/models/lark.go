package models

// LarkChat represents a Lark chatroom
type LarkChat struct {
	ChatID      string `json:"chat_id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	ChatType    string `json:"chat_type,omitempty"`
	ChatMode    string `json:"chat_mode,omitempty"`
	UserCount   string `json:"user_count,omitempty"`
	MemberCount int    `json:"memberCount,omitempty"`
	ChatIdAlt   string `json:"chatId,omitempty"`
}

// LarkSender represents a message sender
type LarkSender struct {
	ID         string `json:"id"`
	IDType     string `json:"id_type,omitempty"`
	Name       string `json:"name,omitempty"`
	SenderID   string `json:"sender_id,omitempty"`
	SenderName string `json:"sender_name,omitempty"`
}

// LarkMessage represents a Lark message
type LarkMessage struct {
	MessageID   string `json:"message_id"`
	ChatID      string `json:"chat_id,omitempty"`
	Sender      *LarkSender `json:"sender,omitempty"`
	CreateTime  string `json:"create_time,omitempty"`
	UpdateTime  string `json:"update_time,omitempty"`
	MsgType     string `json:"msg_type,omitempty"`
	Content     string `json:"content,omitempty"`
	Body        *LarkMessageBody `json:"body,omitempty"`
	Mentions    []LarkMention `json:"mentions,omitempty"`
	ParentID    string `json:"parent_id,omitempty"`
	ThreadID    string `json:"thread_id,omitempty"`
	MessageIdAlt string `json:"messageId,omitempty"`
	MsgTypeAlt  string `json:"msgType,omitempty"`
	CreateTimeAlt string `json:"createTime,omitempty"`
}

// LarkMessageBody represents the message body
type LarkMessageBody struct {
	Content string `json:"content,omitempty"`
}

// LarkMention represents a mention in a message
type LarkMention struct {
	Key         string `json:"key"`
	ID          string `json:"id"`
	Name        string `json:"name"`
	TenantKey   string `json:"tenant_key,omitempty"`
}

// LarkChatListResponse represents the chat list API response
type LarkChatListResponse struct {
	OK    bool `json:"ok"`
	Data  *LarkChatListData `json:"data,omitempty"`
	Chats []LarkChat `json:"chats,omitempty"`
	Error *LarkError `json:"error,omitempty"`
}

// LarkChatListData represents the chat list data
type LarkChatListData struct {
	Items     []LarkChat `json:"items,omitempty"`
	Chats     []LarkChat `json:"chats,omitempty"`
	HasMore   bool       `json:"has_more,omitempty"`
	PageToken string     `json:"page_token,omitempty"`
}

// LarkMessagesResponse represents the messages API response
type LarkMessagesResponse struct {
	OK       bool `json:"ok"`
	Data     *LarkMessagesData `json:"data,omitempty"`
	Messages []LarkMessage `json:"messages,omitempty"`
	Error    *LarkError `json:"error,omitempty"`
}

// LarkMessagesData represents the messages data
type LarkMessagesData struct {
	Items     []LarkMessage `json:"items,omitempty"`
	Messages  []LarkMessage `json:"messages,omitempty"`
	HasMore   bool          `json:"has_more,omitempty"`
	PageToken string        `json:"page_token,omitempty"`
}

// LarkError represents a Lark API error
type LarkError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

// LarkDoctorResponse represents the doctor command response
type LarkDoctorResponse struct {
	OK     bool              `json:"ok,omitempty"`
	Checks []LarkDoctorCheck `json:"checks,omitempty"`
}

// LarkDoctorCheck represents a single doctor check
type LarkDoctorCheck struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

// LarkChatsResponse represents the enriched chats API response
type LarkChatsResponse struct {
	OK     bool             `json:"ok"`
	Chats  []LarkChatItem   `json:"chats"`
	Filter LarkChatFilter   `json:"filter"`
}

// LarkChatItem represents an enriched chat item
type LarkChatItem struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	MemberCount  int    `json:"member_count"`
	Filtered     bool   `json:"filtered"`
}

// SyncRequest represents a sync request
type SyncRequest struct {
	ChatID   string `json:"chat_id,omitempty"`
	DaysBack int    `json:"days_back,omitempty"`
	Stream   bool   `json:"stream,omitempty"`
}

// SyncResult represents a single chat sync result
type SyncResult struct {
	Inserted int    `json:"inserted"`
	Skipped  int    `json:"skipped"`
	Error    string `json:"error"`
}

// SyncResponse represents the sync API response
type SyncResponse struct {
	OK     bool                  `json:"ok"`
	Error  string                `json:"error,omitempty"`
	Synced map[string]SyncResult `json:"synced"`
}

// SyncProgressEvent represents a sync progress SSE event
type SyncProgressEvent struct {
	Type    string `json:"type"`
	ChatID  string `json:"chatId,omitempty"`
	Phase   string `json:"phase,omitempty"`
	Count   int    `json:"count,omitempty"`
	OK      bool   `json:"ok,omitempty"`
	Synced  map[string]SyncResult `json:"synced,omitempty"`
	Error   string `json:"error,omitempty"`
}
