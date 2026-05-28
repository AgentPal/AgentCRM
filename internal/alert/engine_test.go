package alert

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/AgentPal/AgentCRM/internal/model"
	"github.com/AgentPal/AgentCRM/internal/store"
)

func setupTestStore(t *testing.T) (*store.Store, string) {
	t.Helper()
	dir, err := os.MkdirTemp("", "agentcrm-alert-test-*")
	if err != nil {
		t.Fatalf("temp dir: %v", err)
	}
	s, err := store.Open(dir)
	if err != nil {
		os.RemoveAll(dir)
		t.Fatalf("store.Open: %v", err)
	}
	if err := s.FS.Init(); err != nil {
		s.Close()
		os.RemoveAll(dir)
		t.Fatalf("FS.Init: %v", err)
	}
	return s, dir
}

func cleanupStore(s *store.Store, dir string) {
	s.Close()
	os.RemoveAll(dir)
}

func TestScan_StaleDeal(t *testing.T) {
	s, dir := setupTestStore(t)
	defer cleanupStore(s, dir)

	now := store.Now()
	oldTime := time.Now().UTC().Add(-20 * 24 * time.Hour).Format(time.RFC3339)

	dealOld := model.NewDeal()
	dealOld.ID = "deal-stale-1"
	dealOld.Slug = "old-deal"
	dealOld.Title = "旧商机"
	dealOld.Stage = "lead"
	dealOld.UpdatedAt = oldTime
	dealOld.CreatedAt = oldTime
	if err := s.Deals.Create(dealOld); err != nil {
		t.Fatalf("create deal: %v", err)
	}

	dealFresh := model.NewDeal()
	dealFresh.ID = "deal-fresh-1"
	dealFresh.Slug = "fresh-deal"
	dealFresh.Title = "新商机"
	dealFresh.Stage = "lead"
	dealFresh.UpdatedAt = now
	dealFresh.CreatedAt = now
	if err := s.Deals.Create(dealFresh); err != nil {
		t.Fatalf("create deal: %v", err)
	}

	eng := NewEngine(s, dir)
	alerts, err := eng.Scan()
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}

	var staleAlerts []model.Alert
	for _, a := range alerts {
		if a.RuleName == "stale_deal" {
			staleAlerts = append(staleAlerts, a)
		}
	}
	if len(staleAlerts) != 1 {
		t.Fatalf("expected 1 stale_deal alert, got %d", len(staleAlerts))
	}
	if staleAlerts[0].DealID != "deal-stale-1" {
		t.Errorf("expected deal-stale-1, got %q", staleAlerts[0].DealID)
	}
	if staleAlerts[0].Title == "" {
		t.Error("expected non-empty title")
	}
	if staleAlerts[0].Suggestion == "" {
		t.Error("expected non-empty suggestion")
	}
}

func TestScan_StaleDeal_StageFilter(t *testing.T) {
	s, dir := setupTestStore(t)
	defer cleanupStore(s, dir)

	oldTime := time.Now().UTC().Add(-20 * 24 * time.Hour).Format(time.RFC3339)

	dealWon := model.NewDeal()
	dealWon.ID = "deal-won-1"
	dealWon.Slug = "won-deal"
	dealWon.Title = "已赢单"
	dealWon.Stage = "won"
	dealWon.UpdatedAt = oldTime
	dealWon.CreatedAt = oldTime
	if err := s.Deals.Create(dealWon); err != nil {
		t.Fatalf("create deal: %v", err)
	}

	eng := NewEngine(s, dir)
	alerts, err := eng.Scan()
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	for _, a := range alerts {
		if a.RuleName == "stale_deal" {
			t.Errorf("unexpected stale_deal alert for won deal: %s", a.DealID)
		}
	}
}

func TestScan_ClosingDeadline(t *testing.T) {
	s, dir := setupTestStore(t)
	defer cleanupStore(s, dir)

	now := store.Now()
	closeIn3Days := time.Now().UTC().Add(3 * 24 * time.Hour).Format("2006-01-02")

	dealClosing := model.NewDeal()
	dealClosing.ID = "deal-closing-1"
	dealClosing.Slug = "closing-deal"
	dealClosing.Title = "即将截止"
	dealClosing.Stage = "negotiation"
	dealClosing.ExpectedCloseAt = closeIn3Days
	dealClosing.UpdatedAt = now
	dealClosing.CreatedAt = now
	if err := s.Deals.Create(dealClosing); err != nil {
		t.Fatalf("create deal: %v", err)
	}

	dealNoClose := model.NewDeal()
	dealNoClose.ID = "deal-noclose-1"
	dealNoClose.Slug = "no-close-deal"
	dealNoClose.Title = "无截止"
	dealNoClose.Stage = "lead"
	dealNoClose.UpdatedAt = now
	dealNoClose.CreatedAt = now
	if err := s.Deals.Create(dealNoClose); err != nil {
		t.Fatalf("create deal: %v", err)
	}

	eng := NewEngine(s, dir)
	alerts, err := eng.Scan()
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}

	var deadlineAlerts []model.Alert
	for _, a := range alerts {
		if a.RuleName == "closing_deadline" {
			deadlineAlerts = append(deadlineAlerts, a)
		}
	}
	if len(deadlineAlerts) != 1 {
		t.Fatalf("expected 1 closing_deadline alert, got %d", len(deadlineAlerts))
	}
	if deadlineAlerts[0].DealID != "deal-closing-1" {
		t.Errorf("expected deal-closing-1, got %q", deadlineAlerts[0].DealID)
	}
}

func TestScan_ClosingDeadline_StageFilter(t *testing.T) {
	s, dir := setupTestStore(t)
	defer cleanupStore(s, dir)

	now := store.Now()
	closeIn3Days := time.Now().UTC().Add(3 * 24 * time.Hour).Format("2006-01-02")

	dealWon := model.NewDeal()
	dealWon.ID = "deal-won-close"
	dealWon.Slug = "won-close-deal"
	dealWon.Title = "已赢单"
	dealWon.Stage = "won"
	dealWon.ExpectedCloseAt = closeIn3Days
	dealWon.UpdatedAt = now
	dealWon.CreatedAt = now
	if err := s.Deals.Create(dealWon); err != nil {
		t.Fatalf("create deal: %v", err)
	}

	eng := NewEngine(s, dir)
	alerts, err := eng.Scan()
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	for _, a := range alerts {
		if a.RuleName == "closing_deadline" {
			t.Errorf("unexpected closing_deadline for won deal")
		}
	}
}

func TestScan_VIPSilence(t *testing.T) {
	s, dir := setupTestStore(t)
	defer cleanupStore(s, dir)

	oldTime := time.Now().UTC().Add(-60 * 24 * time.Hour).Format(time.RFC3339)
	recentTime := time.Now().UTC().Add(-7 * 24 * time.Hour).Format(time.RFC3339)

	cVip := model.NewContact()
	cVip.ID = model.ContactID()
	cVip.Name = "VIP-Customer-A"
	cVip.Emails = []string{"vip@example.com"}
	cVip.Tags = []string{"vip", "key-account"}
	cVip.LastActivityAt = oldTime
	cVip.CreatedAt = oldTime
	cVip.UpdatedAt = oldTime
	if _, err := s.Contacts.Upsert(cVip); err != nil {
		t.Fatalf("upsert contact: %v", err)
	}

	cVipRecent := model.NewContact()
	cVipRecent.ID = model.ContactID()
	cVipRecent.Name = "VIP-Customer-B"
	cVipRecent.Emails = []string{"vip-recent@example.com"}
	cVipRecent.Tags = []string{"vip"}
	cVipRecent.LastActivityAt = recentTime
	cVipRecent.CreatedAt = recentTime
	cVipRecent.UpdatedAt = recentTime
	if _, err := s.Contacts.Upsert(cVipRecent); err != nil {
		t.Fatalf("upsert contact: %v", err)
	}

	cNormal := model.NewContact()
	cNormal.ID = model.ContactID()
	cNormal.Name = "Normal-Customer-C"
	cNormal.Emails = []string{"normal@example.com"}
	cNormal.Tags = []string{"normal"}
	cNormal.LastActivityAt = oldTime
	cNormal.CreatedAt = oldTime
	cNormal.UpdatedAt = oldTime
	if _, err := s.Contacts.Upsert(cNormal); err != nil {
		t.Fatalf("upsert contact: %v", err)
	}

	eng := NewEngine(s, dir)
	alerts, err := eng.Scan()
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}

	var vipAlerts []model.Alert
	for _, a := range alerts {
		if a.RuleName == "vip_silence" {
			vipAlerts = append(vipAlerts, a)
		}
	}
	if len(vipAlerts) != 1 {
		t.Fatalf("expected 1 vip_silence alert, got %d", len(vipAlerts))
	}
	if vipAlerts[0].ContactName != "VIP-Customer-A" {
		t.Errorf("expected 'VIP-Customer-A', got %q", vipAlerts[0].ContactName)
	}
}

func TestScan_StaleMemory(t *testing.T) {
	s, dir := setupTestStore(t)
	defer cleanupStore(s, dir)

	memo := &model.Memo{ID: model.MemoID(), ScopeType: "contact", ScopeID: "c1", Text: "旧记忆"}
	if err := s.Memos.Insert(memo); err != nil {
		t.Fatalf("insert memo: %v", err)
	}
	if err := s.Memos.MarkExpired(memo.ID); err != nil {
		t.Fatalf("mark expired: %v", err)
	}

	eng := NewEngine(s, dir)
	alerts, err := eng.Scan()
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}

	var memoryAlerts []model.Alert
	for _, a := range alerts {
		if a.RuleName == "stale_memory" {
			memoryAlerts = append(memoryAlerts, a)
		}
	}
	if len(memoryAlerts) != 1 {
		t.Fatalf("expected 1 stale_memory alert, got %d", len(memoryAlerts))
	}
	if memoryAlerts[0].Title == "" {
		t.Error("expected non-empty title")
	}
}

func TestScan_StaleMemory_NoExpired(t *testing.T) {
	s, dir := setupTestStore(t)
	defer cleanupStore(s, dir)

	memo := &model.Memo{ID: model.MemoID(), ScopeType: "contact", ScopeID: "c1", Text: "新鲜记忆"}
	if err := s.Memos.Insert(memo); err != nil {
		t.Fatalf("insert memo: %v", err)
	}

	eng := NewEngine(s, dir)
	alerts, err := eng.Scan()
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	for _, a := range alerts {
		if a.RuleName == "stale_memory" {
			t.Errorf("unexpected stale_memory alert when no expired memos exist")
		}
	}
}

func TestScan_Dedup(t *testing.T) {
	s, dir := setupTestStore(t)
	defer cleanupStore(s, dir)

	oldTime := time.Now().UTC().Add(-20 * 24 * time.Hour).Format(time.RFC3339)
	now := store.Now()

	deal := model.NewDeal()
	deal.ID = "deal-dedup-1"
	deal.Slug = "dedup-deal"
	deal.Title = "重复检测"
	deal.Stage = "lead"
	deal.UpdatedAt = oldTime
	deal.CreatedAt = oldTime
	if err := s.Deals.Create(deal); err != nil {
		t.Fatalf("create deal: %v", err)
	}

	existingAlerts := []model.Alert{{
		ID:        "alert-exist-1",
		RuleName:  "stale_deal",
		Status:    model.AlertStatusPending,
		DealID:    "deal-dedup-1",
		Title:     "商机长时间未跟进",
		Priority:  2,
		CreatedAt: now,
	}}
	if err := s.FS.WritePendingAlerts(existingAlerts); err != nil {
		t.Fatalf("WritePendingAlerts: %v", err)
	}

	eng := NewEngine(s, dir)
	alerts, err := eng.Scan()
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}

	var staleAlerts []model.Alert
	for _, a := range alerts {
		if a.RuleName == "stale_deal" {
			staleAlerts = append(staleAlerts, a)
		}
	}
	if len(staleAlerts) != 0 {
		t.Errorf("expected 0 new stale_deal alerts (dedup), got %d", len(staleAlerts))
	}
}

func TestScan_NoRulesNoAlerts(t *testing.T) {
	s, dir := setupTestStore(t)
	defer cleanupStore(s, dir)

	now := store.Now()

	deal := model.NewDeal()
	deal.ID = "deal-fresh-test"
	deal.Slug = "fresh-test-deal"
	deal.Title = "新鲜商机"
	deal.Stage = "lead"
	deal.UpdatedAt = now
	deal.CreatedAt = now
	if err := s.Deals.Create(deal); err != nil {
		t.Fatalf("create deal: %v", err)
	}

	c := model.NewContact()
	c.ID = model.ContactID()
	c.Name = "普通客户丁"
	c.Emails = []string{"normal@example.com"}
	c.Tags = []string{"normal"}
	c.LastActivityAt = now
	c.CreatedAt = now
	c.UpdatedAt = now
	if _, err := s.Contacts.Upsert(c); err != nil {
		t.Fatalf("upsert contact: %v", err)
	}

	eng := NewEngine(s, dir)
	alerts, err := eng.Scan()
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(alerts) != 0 {
		t.Errorf("expected 0 alerts with all fresh data, got %d", len(alerts))
	}
}

func TestBuiltinRules(t *testing.T) {
	rules := BuiltinRules()
	if len(rules) != 4 {
		t.Fatalf("expected 4 built-in rules, got %d", len(rules))
	}
	names := make(map[string]bool)
	for _, r := range rules {
		names[r.Name] = true
	}
	expected := []string{"stale_deal", "closing_deadline", "vip_silence", "stale_memory"}
	for _, name := range expected {
		if !names[name] {
			t.Errorf("missing built-in rule: %s", name)
		}
	}
}

func TestScan_MultipleRules(t *testing.T) {
	s, dir := setupTestStore(t)
	defer cleanupStore(s, dir)

	oldTime := time.Now().UTC().Add(-20 * 24 * time.Hour).Format(time.RFC3339)
	closeIn3Days := time.Now().UTC().Add(3 * 24 * time.Hour).Format("2006-01-02")
	now := store.Now()

	dealStale := model.NewDeal()
	dealStale.ID = "deal-multi-1"
	dealStale.Slug = "multi-stale"
	dealStale.Title = "旧 deal"
	dealStale.Stage = "lead"
	dealStale.UpdatedAt = oldTime
	dealStale.CreatedAt = oldTime
	if err := s.Deals.Create(dealStale); err != nil {
		t.Fatalf("create deal: %v", err)
	}

	dealClosing := model.NewDeal()
	dealClosing.ID = "deal-multi-2"
	dealClosing.Slug = "multi-closing"
	dealClosing.Title = "即将截止"
	dealClosing.Stage = "negotiation"
	dealClosing.ExpectedCloseAt = closeIn3Days
	dealClosing.UpdatedAt = now
	dealClosing.CreatedAt = now
	if err := s.Deals.Create(dealClosing); err != nil {
		t.Fatalf("create deal: %v", err)
	}

	eng := NewEngine(s, dir)
	alerts, err := eng.Scan()
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}

	ruleNames := make(map[string]bool)
	for _, a := range alerts {
		ruleNames[a.RuleName] = true
	}
	if !ruleNames["stale_deal"] {
		t.Error("expected stale_deal alert")
	}
	if !ruleNames["closing_deadline"] {
		t.Error("expected closing_deadline alert")
	}
}

func TestScan_UserRulesNotLoadedWithoutFile(t *testing.T) {
	s, dir := setupTestStore(t)
	defer cleanupStore(s, dir)

	eng := NewEngine(s, dir)
	_, err := eng.Scan()
	if err != nil {
		t.Fatalf("Scan should not error without user.yaml: %v", err)
	}
}

func TestScan_AlertHasRequiredFields(t *testing.T) {
	s, dir := setupTestStore(t)
	defer cleanupStore(s, dir)

	oldTime := time.Now().UTC().Add(-20 * 24 * time.Hour).Format(time.RFC3339)

	deal := model.NewDeal()
	deal.ID = "deal-fields-1"
	deal.Slug = "fields-deal"
	deal.Title = "字段测试"
	deal.Stage = "lead"
	deal.UpdatedAt = oldTime
	deal.CreatedAt = oldTime
	if err := s.Deals.Create(deal); err != nil {
		t.Fatalf("create deal: %v", err)
	}

	eng := NewEngine(s, dir)
	alerts, err := eng.Scan()
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}

	for _, a := range alerts {
		if a.RuleName == "stale_deal" {
			if a.ID == "" {
				t.Error("expected non-empty ID")
			}
			if !strings.HasPrefix(a.ID, "alr_") {
				t.Errorf("expected 'alr_' prefix, got %q", a.ID)
			}
			if a.Status != model.AlertStatusPending {
				t.Errorf("expected pending status, got %q", a.Status)
			}
			if a.CreatedAt == "" {
				t.Error("expected non-empty CreatedAt")
			}
		}
	}
}
