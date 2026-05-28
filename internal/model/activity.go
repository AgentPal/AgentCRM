package model

import "time"

// Activity 表示一次互动事件（邮件、通话、会议、聊天、笔记、社交等）。
type Activity struct {
	ID        string `json:"id"`
	Timestamp string `json:"ts"`
	Actor     string `json:"actor"`
	Type      string `json:"type"` // email | call | meeting | chat | note | social
	Direction string `json:"direction"` // in | out
	Channel   string `json:"channel,omitempty"` // gmail | twitter | wechat | phone | ...

	ContactIDs []string `json:"contact_ids,omitempty"`
	DealIDs    []string `json:"deal_ids,omitempty"`

	Subject string `json:"subject,omitempty"`
	Summary string `json:"summary"`
	Body    string `json:"body,omitempty"`

	DedupeKey string `json:"dedupe_key"`
	SourceURL string `json:"source_url,omitempty"`
}

// NewActivity 创建一个新的 Activity 实例。
func NewActivity() *Activity {
	return &Activity{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
}

const (
	DirectionIn  = "in"
	DirectionOut = "out"
)

const (
	TypeEmail   = "email"
	TypeCall    = "call"
	TypeMeeting = "meeting"
	TypeChat    = "chat"
	TypeNote    = "note"
	TypeSocial  = "social"
)

var ValidActivityTypes = []string{TypeEmail, TypeCall, TypeMeeting, TypeChat, TypeNote, TypeSocial}
var ValidDirections = []string{DirectionIn, DirectionOut}
