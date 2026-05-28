package model

import "time"

// Event 表示一次数据变化事件，用于多 Agent 协作。
type Event struct {
	Seq     int64  `json:"seq"`
	TS      string `json:"ts"`
	Actor   string `json:"actor"`
	Type    string `json:"type"`
	Payload string `json:"payload"` // JSON string
}

// NewEvent 创建新事件。
func NewEvent(seq int64, actor, eventType string, payload string) *Event {
	return &Event{
		Seq:     seq,
		TS:      time.Now().UTC().Format(time.RFC3339),
		Actor:   actor,
		Type:    eventType,
		Payload: payload,
	}
}

// 事件类型常量（v1.0 固定清单）。
const (
	EventContactCreated     = "contact.created"
	EventContactUpdated     = "contact.updated"
	EventContactFieldUpdated = "contact.field_updated"
	EventContactTagged      = "contact.tagged"
	EventContactMerged      = "contact.merged"
	EventContactDeleted     = "contact.deleted"

	EventDealCreated      = "deal.created"
	EventDealUpdated      = "deal.updated"
	EventDealStageChanged = "deal.stage_changed"
	EventDealAmountChanged = "deal.amount_changed"
	EventDealClosedWon    = "deal.closed_won"
	EventDealClosedLost   = "deal.closed_lost"

	EventActivityLogged = "activity.logged"

	EventMemoryWritten    = "memory.written"
	EventMemorySuperseded = "memory.superseded"
	EventMemoryDecayed    = "memory.decayed"
	EventMemoryForgotten  = "memory.forgotten"

	EventAlertTriggered  = "alert.triggered"
	EventAlertDismissed  = "alert.dismissed"
)

// SubscriberCursor 记录订阅者的消费位置。
type SubscriberCursor struct {
	Actor     string `json:"actor"`
	LastSeq   int64  `json:"last_seq"`
	LastCheck string `json:"last_check_at"`
	Filter    string `json:"filter_default,omitempty"`
}
