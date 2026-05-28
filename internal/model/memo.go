package model

import "time"

// Memo 表示一条记忆条目——Agent 蒸馏出的长期信息。
// 存储在 memory/contacts/<slug>.md 或 memory/global.md 中。
type Memo struct {
	ID        string `json:"id"`
	ScopeType string `json:"scope_type"` // contact | deal | tenant
	ScopeID   string `json:"scope_id"`
	Text      string `json:"text"`

	ValidFrom string `json:"valid_from,omitempty"`
	ValidTo   string `json:"valid_to,omitempty"`
	Decay     string `json:"decay,omitempty"` // never | 30d | 90d | 180d | ...

	Expired   bool   `json:"expired,omitempty"`
	ExpiredAt string `json:"expired_at,omitempty"`

	Supersedes string `json:"supersedes,omitempty"` // 被此 memo 取代的旧 memo ID
	Source     string `json:"source,omitempty"`
	Actor      string `json:"actor,omitempty"`

	CreatedAt string `json:"created_at,omitempty"`
	UpdatedAt string `json:"updated_at,omitempty"`
}

// IsExpired 检查 memo 是否已过期。
// 过期判断：有 expired_at 且已到；或 valid_from + decay 已超期。
func (m *Memo) IsExpired(asOf string) bool {
	if m.Expired {
		return true
	}
	if m.Decay == "" || m.Decay == "never" {
		return false
	}
	if m.ValidFrom == "" {
		return false
	}

	from, err := time.Parse(time.RFC3339, m.ValidFrom)
	if err != nil {
		// 尝试日期格式
		from, err = time.Parse("2006-01-02", m.ValidFrom)
		if err != nil {
			return false
		}
	}

	dur, err := ParseDecayDuration(m.Decay)
	if err != nil {
		return false
	}

	expiry := from.Add(dur)

	var asOfTime time.Time
	if asOf == "" {
		asOfTime = time.Now().UTC()
	} else {
		asOfTime, _ = time.Parse(time.RFC3339, asOf)
		if asOfTime.IsZero() {
			asOfTime, _ = time.Parse("2006-01-02", asOf)
		}
	}

	return asOfTime.After(expiry)
}

// ParseDecayDuration 解析 decay 字符串为 time.Duration。
func ParseDecayDuration(s string) (time.Duration, error) {
	switch s {
	case "never":
		return 0, nil
	case "30d":
		return 30 * 24 * time.Hour, nil
	case "60d":
		return 60 * 24 * time.Hour, nil
	case "90d":
		return 90 * 24 * time.Hour, nil
	case "180d":
		return 180 * 24 * time.Hour, nil
	case "365d", "1y":
		return 365 * 24 * time.Hour, nil
	}

	// 尝试直接解析 Xd 格式
	d, err := time.ParseDuration(s)
	if err == nil {
		return d, nil
	}

	return 180 * 24 * time.Hour, nil // 默认 180d
}

// Proposal 表示一条待裁决的记忆建议。
type Proposal struct {
	ID          string        `json:"id"`
	Timestamp   string        `json:"ts"`
	Actor       string        `json:"actor"`
	Scope       string        `json:"scope"`
	Statement   string        `json:"statement"`
	SourceSnippet string      `json:"source_snippet,omitempty"`
	Confidence  float64       `json:"confidence"`
	Status      string        `json:"status"` // pending | conflict | clean
	ConflictWith []ConflictItem `json:"conflict_with,omitempty"`
	SuggestedAction string   `json:"suggested_action,omitempty"` // supersede | keep-both | reject
}

type ConflictItem struct {
	MemoID    string `json:"memo_id"`
	Text      string `json:"text"`
	ValidFrom string `json:"valid_from"`
}
