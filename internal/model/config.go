package model

import "encoding/json"

// Config 是 ~/AgentCRM/config.json 的结构。
type Config struct {
	Version string `json:"version"`
	DataDir string `json:"data_dir"`
	User    UserConfig `json:"user"`
	Memory  MemoryConfig `json:"memory"`
	Search  SearchConfig `json:"search"`
	Alerts  AlertConfig `json:"alerts"`
	Privacy PrivacyConfig `json:"privacy"`
}

type UserConfig struct {
	Name    string `json:"name"`
	PrimaryEmail string `json:"primary_email"`
	Timezone string `json:"timezone"`
}

type MemoryConfig struct {
	DefaultDecay      string   `json:"default_decay"`
	ProposeConfidence float64  `json:"propose_confidence,omitempty"`
}

type SearchConfig struct {
	Strategies []string `json:"strategies"`
	Embedding  EmbeddingConfig `json:"embedding"`
}

type EmbeddingConfig struct {
	Enabled bool   `json:"enabled"`
	Model   string `json:"model,omitempty"`
}

type AlertConfig struct {
	StaleDealDays int    `json:"stale_deal_days"`
	Channel       string `json:"channel"`
}

type PrivacyConfig struct {
	RedactInAudit []string `json:"redact_in_audit"`
}

// DefaultConfig 返回默认配置。
func DefaultConfig(dataDir string) *Config {
	return &Config{
		Version: "1.0.0",
		DataDir: dataDir,
		User: UserConfig{
			Name:    "",
			PrimaryEmail: "",
			Timezone: "Asia/Shanghai",
		},
		Memory: MemoryConfig{
			DefaultDecay:      "180d",
			ProposeConfidence: 0.7,
		},
		Search: SearchConfig{
			Strategies: []string{"fts", "entity", "temporal"},
			Embedding: EmbeddingConfig{
				Enabled: false,
				Model:   "all-MiniLM-L6-v2",
			},
		},
		Alerts: AlertConfig{
			StaleDealDays: 14,
			Channel:       "stdout",
		},
		Privacy: PrivacyConfig{
			RedactInAudit: []string{"phones"},
		},
	}
}

func (c *Config) ToJSON() (string, error) {
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}
