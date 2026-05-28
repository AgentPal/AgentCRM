package alert

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/AgentPal/AgentCRM/internal/model"
	"github.com/AgentPal/AgentCRM/internal/store"
	"gopkg.in/yaml.v3"
)

// Engine 评估规则并生成提醒。
type Engine struct {
	store     *store.Store
	configDir string
}

// NewEngine 创建规则引擎。
func NewEngine(s *store.Store, configDir string) *Engine {
	return &Engine{store: s, configDir: configDir}
}

// Scan 执行所有规则扫描，返回新生成的提醒。
func (e *Engine) Scan() ([]model.Alert, error) {
	rules, err := e.loadRules()
	if err != nil {
		return nil, fmt.Errorf("load rules: %w", err)
	}

	existing, _ := e.store.FS.ReadPendingAlerts()
	existingMap := make(map[string]bool)
	for _, a := range existing {
		if a.Status == model.AlertStatusPending {
			existingMap[a.RuleName+":"+a.ContactID+":"+a.DealID] = true
		}
	}

	var alerts []model.Alert
	for _, rule := range rules {
		results, err := e.evaluate(rule, existingMap)
		if err != nil {
			continue
		}
		alerts = append(alerts, results...)
	}
	return alerts, nil
}

func (e *Engine) loadRules() ([]model.Rule, error) {
	rules := builtinRules()

	userRules, err := loadYAMLRules(filepath.Join(e.configDir, "rules", "user.yaml"))
	if err == nil {
		rules = append(rules, userRules...)
	}
	// also load default.yaml if exists
	defaultRules, err := loadYAMLRules(filepath.Join(e.configDir, "rules", "default.yaml"))
	if err == nil {
		rules = append(rules, defaultRules...)
	}

	return rules, nil
}

// BuiltinRules 返回内置规则列表（包外可访问）。
func BuiltinRules() []model.Rule {
	return builtinRules()
}

func builtinRules() []model.Rule {
	return []model.Rule{
		{
			Name: "stale_deal",
			When: model.RuleCondition{
				Object:             "deal",
				StageIn:            []string{"lead", "qualified"},
				LastActivityAgeDays: "14",
			},
			Then: model.RuleAction{
				Alert: model.RuleAlert{
					Title:      "商机长时间未跟进",
					Suggestion: "联系客户了解进展，推进商机阶段",
				},
			},
		},
		{
			Name: "closing_deadline",
			When: model.RuleCondition{
				Object:             "deal",
				ExpectedCloseInDays: "7",
				StageNotIn:         []string{"won", "lost"},
			},
			Then: model.RuleAction{
				Alert: model.RuleAlert{
					Title:      "商机即将到截止日期",
					Suggestion: "确认成交状态或更新预计日期",
				},
			},
		},
		{
			Name: "vip_silence",
			When: model.RuleCondition{
				Object:             "contact",
				TagIncludes:        []string{"vip"},
				LastActivityAgeDays: "30",
			},
			Then: model.RuleAction{
				Alert: model.RuleAlert{
					Title:      "VIP 联系人长期未联系",
					Suggestion: "发送问候或安排跟进",
				},
			},
		},
		{
			Name: "stale_memory",
			When: model.RuleCondition{
				Object:          "memory",
				HasExpiredMemos: true,
			},
			Then: model.RuleAction{
				Alert: model.RuleAlert{
					Title:      "有记忆条目已过期需要更新",
					Suggestion: "运行 memory decay-scan 清理过期条目",
				},
			},
		},
	}
}

func loadYAMLRules(path string) ([]model.Rule, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg model.RulesConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse rules: %w", err)
	}
	return cfg.Rules, nil
}

func (e *Engine) evaluate(rule model.Rule, existing map[string]bool) ([]model.Alert, error) {
	switch rule.When.Object {
	case "deal":
		return e.evaluateDealRule(rule, existing)
	case "contact":
		return e.evaluateContactRule(rule, existing)
	case "memory":
		return e.evaluateMemoryRule(rule, existing)
	}
	return nil, nil
}

func (e *Engine) evaluateDealRule(rule model.Rule, existing map[string]bool) ([]model.Alert, error) {
	deals, err := e.store.Deals.List("", "", "")
	if err != nil {
		return nil, err
	}

	ageDays, _ := strconv.Atoi(rule.When.LastActivityAgeDays)
	closeDays, _ := strconv.Atoi(rule.When.ExpectedCloseInDays)
	now := time.Now()

	var alerts []model.Alert
	for _, d := range deals {
		deal, err := e.store.Deals.GetByID(d.ID)
		if err != nil {
			continue
		}

		// Check stage_in / stage_not_in
		if len(rule.When.StageIn) > 0 {
			if !contains(rule.When.StageIn, deal.Stage) {
				continue
			}
		}
		if len(rule.When.StageNotIn) > 0 {
			if contains(rule.When.StageNotIn, deal.Stage) {
				continue
			}
		}

		// Check last_activity_age_days
		if ageDays > 0 {
			updatedAt := deal.UpdatedAt
			if updatedAt == "" {
				continue
			}
			t, err := time.Parse(time.RFC3339, updatedAt)
			if err != nil {
				continue
			}
			if now.Sub(t).Hours()/24 < float64(ageDays) {
				continue // not stale enough
			}
		}

		// Check expected_close_in_days
		if closeDays > 0 {
			if deal.ExpectedCloseAt == "" {
				continue
			}
			t, err := time.Parse(time.RFC3339, deal.ExpectedCloseAt)
			if err != nil {
				t, err = time.Parse("2006-01-02", deal.ExpectedCloseAt)
				if err != nil {
					continue
				}
			}
			remaining := t.Sub(now).Hours() / 24
			if remaining > float64(closeDays) || remaining < 0 {
				continue // not within window or already past
			}
		}

		key := rule.Name + "::" + d.ID
		if existing[key] {
			continue
		}

		alerts = append(alerts, model.Alert{
			ID:        model.NewID(model.PrefixAlert),
			RuleName:  rule.Name,
			Title:     rule.Then.Alert.Title,
			Suggestion: rule.Then.Alert.Suggestion,
			DealID:    d.ID,
			DealTitle: d.Title,
			Priority:  2,
			Status:    model.AlertStatusPending,
			CreatedAt: store.Now(),
		})
	}
	return alerts, nil
}

func (e *Engine) evaluateContactRule(rule model.Rule, existing map[string]bool) ([]model.Alert, error) {
	contacts, err := e.store.Contacts.List("", "", "", 0)
	if err != nil {
		return nil, err
	}

	ageDays, _ := strconv.Atoi(rule.When.LastActivityAgeDays)
	now := time.Now()

	var alerts []model.Alert
	for _, c := range contacts {
		// Check tag_includes
		if len(rule.When.TagIncludes) > 0 {
			hasTag := false
			for _, tag := range strings.Split(c.Tags, ", ") {
				for _, required := range rule.When.TagIncludes {
					if strings.TrimSpace(tag) == required {
						hasTag = true
						break
					}
				}
				if hasTag {
					break
				}
			}
			if !hasTag {
				continue
			}
		}

		// Check last_activity_age_days
		if ageDays > 0 {
			lastAct := c.LastActivityAt
			if lastAct == "" {
				// no activity ever → use created_at
				contact, err := e.store.Contacts.GetByID(c.ID)
				if err != nil {
					continue
				}
				lastAct = contact.CreatedAt
			}
			t, err := time.Parse(time.RFC3339, lastAct)
			if err != nil {
				continue
			}
			if now.Sub(t).Hours()/24 < float64(ageDays) {
				continue
			}
		}

		key := rule.Name + ":" + c.ID + ":"
		if existing[key] {
			continue
		}

		alerts = append(alerts, model.Alert{
			ID:          model.NewID(model.PrefixAlert),
			RuleName:    rule.Name,
			Title:       rule.Then.Alert.Title,
			Suggestion:  rule.Then.Alert.Suggestion,
			ContactID:   c.ID,
			ContactName: c.Name,
			Priority:    1,
			Status:      model.AlertStatusPending,
			CreatedAt:   store.Now(),
		})
	}
	return alerts, nil
}

func (e *Engine) evaluateMemoryRule(rule model.Rule, existing map[string]bool) ([]model.Alert, error) {
	if !rule.When.HasExpiredMemos {
		return nil, nil
	}

	memos, err := e.store.Memos.ListByScope("", "")
	if err != nil {
		return nil, err
	}

	hasExpired := false
	for _, m := range memos {
		if m.Expired {
			hasExpired = true
			break
		}
	}

	if !hasExpired {
		return nil, nil
	}

	key := rule.Name + "::"
	if existing[key] {
		return nil, nil
	}

	return []model.Alert{{
		ID:         model.NewID(model.PrefixAlert),
		RuleName:   rule.Name,
		Title:      rule.Then.Alert.Title,
		Suggestion: rule.Then.Alert.Suggestion,
		Priority:   3,
		Status:     model.AlertStatusPending,
		CreatedAt:  store.Now(),
	}}, nil
}

func contains(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}
