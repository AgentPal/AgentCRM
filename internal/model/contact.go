package model

import "time"

// Contact 表示一个联系人。
// 存储为 ~/.agentcrm/contacts/<slug>.md，frontmatter 是机器读的，body 是人和 LLM 可读的。
type Contact struct {
	ID    string `yaml:"id" json:"id"`
	Slug  string `yaml:"-" json:"slug"`
	Name  string `yaml:"name" json:"name"`
	Emails []string `yaml:"emails" json:"emails"`
	Phones []string `yaml:"phones,omitempty" json:"phones,omitempty"`
	Social ContactSocial `yaml:"social,omitempty" json:"social,omitempty"`

	// 当前值
	Company string   `yaml:"company,omitempty" json:"company,omitempty"`
	Title   string   `yaml:"title,omitempty" json:"title,omitempty"`
	Tags    []string `yaml:"tags,omitempty" json:"tags,omitempty"`

	// 历史归档（CLI 自动维护）
	CompanyHistory []FieldHistory `yaml:"company_history,omitempty" json:"company_history,omitempty"`
	TitleHistory   []FieldHistory `yaml:"title_history,omitempty" json:"title_history,omitempty"`

	Source      string `yaml:"source,omitempty" json:"source,omitempty"`
	Birthday    string `yaml:"birthday,omitempty" json:"birthday,omitempty"` // YYYY-MM-DD
	HasBirthday bool   `yaml:"-" json:"-"` // 计算字段，仅在 SQLite 层使用
	CreatedAt   string `yaml:"created_at" json:"created_at"`
	UpdatedAt   string `yaml:"updated_at" json:"updated_at"`
	LastActivityAt string `yaml:"last_activity_at,omitempty" json:"last_activity_at,omitempty"`

	Body string `yaml:"-" json:"body,omitempty"` // frontmatter 之后的正文
}

type ContactSocial struct {
	Twitter  string `yaml:"twitter,omitempty" json:"twitter,omitempty"`
	LinkedIn string `yaml:"linkedin,omitempty" json:"linkedin,omitempty"`
	WeChat   string `yaml:"wechat,omitempty" json:"wechat,omitempty"`
	GitHub   string `yaml:"github,omitempty" json:"github,omitempty"`
}

// FieldHistory 记录一个字段的历史值。
type FieldHistory struct {
	Value  string `yaml:"value" json:"value"`
	From   string `yaml:"from" json:"from"`    // 起始日期
	To     string `yaml:"to" json:"to"`        // 截止日期，~ 表示当前
	SetBy  string `yaml:"set_by,omitempty" json:"set_by,omitempty"`
	Reason string `yaml:"reason,omitempty" json:"reason,omitempty"`
}

// IsCurrent 判断此历史条目是否为当前值。
func (h FieldHistory) IsCurrent() bool {
	return h.To == "" || h.To == "~"
}

// ActiveUntil 返回有效截止日期。当前值返回 "9999-12-31" 便于比较。
func (h FieldHistory) ActiveUntil() string {
	if h.IsCurrent() {
		return "9999-12-31"
	}
	return h.To
}

// NewFieldHistory 创建一个新的历史条目。
func NewFieldHistory(value, from, setBy, reason string) FieldHistory {
	return FieldHistory{
		Value:  value,
		From:   from,
		To:     "~",
		SetBy:  setBy,
		Reason: reason,
	}
}

// ContactSearchResult 用于 CLI 输出的精简版。
type ContactSearchResult struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Company        string `json:"company,omitempty"`
	Title          string `json:"title,omitempty"`
	Email          string `json:"email,omitempty"`
	LastActivityAt string `json:"last_activity_at,omitempty"`
	Tags           string `json:"tags,omitempty"` // 逗号分隔
}

// AsOf 将联系人字段重建到指定时间点（利用 history 字段）。
func (c *Contact) AsOf(date string) *Contact {
	// Return copy to avoid mutating original
	clone := *c

	resolveHistory := func(history []FieldHistory, currentVal string) string {
		for _, h := range history {
			if h.From <= date && date <= h.ActiveUntil() {
				return h.Value
			}
		}
		return currentVal
	}

	clone.Company = resolveHistory(c.CompanyHistory, c.Company)
	clone.Title = resolveHistory(c.TitleHistory, c.Title)
	return &clone
}

func NewContact() *Contact {
	now := time.Now().UTC().Format(time.RFC3339)
	return &Contact{
		CreatedAt: now,
		UpdatedAt: now,
	}
}

const (
	StageLead         = "lead"
	StageQualified    = "qualified"
	StageProposal     = "proposal"
	StageNegotiation  = "negotiation"
	StageWon          = "won"
	StageLost         = "lost"
)

var ValidStages = []string{StageLead, StageQualified, StageProposal, StageNegotiation, StageWon, StageLost}
