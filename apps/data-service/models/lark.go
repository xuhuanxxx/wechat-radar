// Package models holds Go-only types that don't appear on the wire
// (and therefore don't belong in the OpenAPI contract). The wire-facing
// types live in package api, generated from packages/api-contract/openapi.yaml.
//
// What lives here today: shapes that parse `lark-cli` output. These vary
// between lark-cli versions (camelCase vs snake_case fields), so we keep
// them in Go where struct tags can absorb the variance without polluting
// the public contract.
package models

// LarkChat is the shape emitted by `lark-cli im +chat-list`.
type LarkChat struct {
	ChatID      string `json:"chat_id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	ChatType    string `json:"chat_type,omitempty"`
	ChatMode    string `json:"chat_mode,omitempty"`
	UserCount   string `json:"user_count,omitempty"`
	MemberCount int    `json:"memberCount,omitempty"`
	ChatIDAlt   string `json:"chatId,omitempty"`
}

// LarkSender is the sender block on a parsed lark-cli message.
type LarkSender struct {
	ID         string `json:"id"`
	IDType     string `json:"id_type,omitempty"`
	Name       string `json:"name,omitempty"`
	SenderID   string `json:"sender_id,omitempty"`
	SenderName string `json:"sender_name,omitempty"`
}

// LarkMessageBody mirrors the body sub-object on a parsed lark-cli message.
type LarkMessageBody struct {
	Content string `json:"content,omitempty"`
}

// LarkMention is a single @-mention entry on a parsed lark-cli message.
type LarkMention struct {
	Key       string `json:"key"`
	ID        string `json:"id"`
	Name      string `json:"name"`
	TenantKey string `json:"tenant_key,omitempty"`
}

// LarkMessage is the shape emitted by `lark-cli im +chat-messages-list`.
// Note both snake_case and camelCase variants of common fields — lark-cli
// has been inconsistent across versions and we let the JSON decoder pick
// whichever is present.
type LarkMessage struct {
	MessageID     string           `json:"message_id"`
	ChatID        string           `json:"chat_id,omitempty"`
	Sender        *LarkSender      `json:"sender,omitempty"`
	CreateTime    string           `json:"create_time,omitempty"`
	UpdateTime    string           `json:"update_time,omitempty"`
	MsgType       string           `json:"msg_type,omitempty"`
	Content       string           `json:"content,omitempty"`
	Body          *LarkMessageBody `json:"body,omitempty"`
	Mentions      []LarkMention    `json:"mentions,omitempty"`
	ParentID      string           `json:"parent_id,omitempty"`
	ThreadID      string           `json:"thread_id,omitempty"`
	MessageIDAlt  string           `json:"messageId,omitempty"`
	MsgTypeAlt    string           `json:"msgType,omitempty"`
	CreateTimeAlt string           `json:"createTime,omitempty"`
}

// LarkError is the error block lark-cli emits when a command fails.
type LarkError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}
