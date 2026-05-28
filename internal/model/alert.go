package model

// Alert 表示一条提醒，由规则引擎生成。
type Alert struct {
	ID        string `json:"id"`
	RuleName  string `json:"rule_name"`
	Title     string `json:"title"`
	Suggestion string `json:"suggestion,omitempty"`

	ContactID string `json:"contact_id,omitempty"`
	ContactName string `json:"contact_name,omitempty"`
	DealID    string `json:"deal_id,omitempty"`
	DealTitle string `json:"deal_title,omitempty"`

	Priority  int    `json:"priority"` // 0=P0, 1=P1, 2=P2, 3=P3
	Status    string `json:"status"`   // pending | dismissed | snoozed
	SnoozedUntil string `json:"snoozed_until,omitempty"`

	CreatedAt string `json:"created_at"`
	DismissedAt string `json:"dismissed_at,omitempty"`
	DismissReason string `json:"dismiss_reason,omitempty"`
	DismissPermanent bool `json:"dismiss_permanent,omitempty"`
}

const (
	AlertStatusPending  = "pending"
	AlertStatusDismissed = "dismissed"
	AlertStatusSnoozed  = "snoozed"
)

// Rule 定义一条提醒规则。
type Rule struct {
	Name string `yaml:"name"`
	When RuleCondition `yaml:"when"`
	Then RuleAction `yaml:"then"`
}

type RuleCondition struct {
	Object             string `yaml:"object"` // deal | contact
	StageIn            []string `yaml:"stage_in,omitempty"`
	StageNotIn         []string `yaml:"stage_not_in,omitempty"`
	LastActivityAgeDays string `yaml:"last_activity_age_days,omitempty"`
	ExpectedCloseInDays string `yaml:"expected_close_in_days,omitempty"`
	TagIncludes        []string `yaml:"tag_includes,omitempty"`
	HasExpiredMemos    bool     `yaml:"has_expired_memos,omitempty"`
}

type RuleAction struct {
	Alert RuleAlert `yaml:"alert"`
}

type RuleAlert struct {
	Title      string `yaml:"title"`
	Suggestion string `yaml:"suggestion,omitempty"`
}

// RulesConfig 是规则文件的顶层结构。
type RulesConfig struct {
	Rules []Rule `yaml:"rules"`
}
