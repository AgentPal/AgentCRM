package model

import "time"

// Deal 表示一个商机。
// 存储为 ~/AgentCRM/deals/<slug>.md。
type Deal struct {
	ID    string   `yaml:"id" json:"id"`
	Slug  string   `yaml:"-" json:"slug"`
	Title string   `yaml:"title" json:"title"`
	Stage string   `yaml:"stage" json:"stage"`
	ContactIDs []string `yaml:"contact_ids" json:"contact_ids"`

	Amount   int    `yaml:"amount,omitempty" json:"amount,omitempty"`
	Currency string `yaml:"currency,omitempty" json:"currency,omitempty"`

	ExpectedCloseAt string `yaml:"expected_close_at,omitempty" json:"expected_close_at,omitempty"`
	Owner           string `yaml:"owner,omitempty" json:"owner,omitempty"`

	CreatedAt string `yaml:"created_at" json:"created_at"`
	UpdatedAt string `yaml:"updated_at" json:"updated_at"`

	// 历史归档
	StageHistory  []StageHistory  `yaml:"stage_history,omitempty" json:"stage_history,omitempty"`
	AmountHistory []AmountHistory `yaml:"amount_history,omitempty" json:"amount_history,omitempty"`

	Body string `yaml:"-" json:"body,omitempty"`
}

type StageHistory struct {
	Stage     string `yaml:"stage" json:"stage"`
	EnteredAt string `yaml:"entered_at" json:"entered_at"`
	By        string `yaml:"by,omitempty" json:"by,omitempty"`
	Reason    string `yaml:"reason,omitempty" json:"reason,omitempty"`
}

type AmountHistory struct {
	Value  int    `yaml:"value" json:"value"`
	From   string `yaml:"from" json:"from"`
	To     string `yaml:"to" json:"to"`
	Reason string `yaml:"reason,omitempty" json:"reason,omitempty"`
}

func (h AmountHistory) IsCurrent() bool {
	return h.To == "" || h.To == "~"
}

// AsOf 将商机字段重建到指定时间点（利用 history 字段）。
func (d *Deal) AsOf(date string) *Deal {
	clone := *d

	// Stage history: find the stage active at date
	for _, h := range d.StageHistory {
		if h.EnteredAt <= date {
			clone.Stage = h.Stage
		}
	}

	// Amount history: find entry where from <= date <= to
	for _, h := range d.AmountHistory {
		until := h.To
		if until == "" || until == "~" {
			until = "9999-12-31"
		}
		if h.From <= date && date <= until {
			clone.Amount = h.Value
		}
	}

	return &clone
}

func NewDeal() *Deal {
	now := time.Now().UTC().Format(time.RFC3339)
	return &Deal{
		Stage:     StageLead,
		Currency:  "CNY",
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// StageTransitionAllowed 检查阶段跃迁是否合法。
// won/lost 不可逆；其他阶段可前进可后退。
func StageTransitionAllowed(from, to string) bool {
	if from == StageWon || from == StageLost {
		return false
	}
	return true
}

// DealSearchResult CLI 输出用精简版。
type DealSearchResult struct {
	ID              string `json:"id"`
	Title           string `json:"title"`
	Stage           string `json:"stage"`
	Amount          int    `json:"amount"`
	Currency        string `json:"currency"`
	ContactName     string `json:"contact_name,omitempty"`
	ExpectedCloseAt string `json:"expected_close_at,omitempty"`
	UpdatedAt       string `json:"updated_at"`
}
