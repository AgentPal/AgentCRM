package model

import (
	"strings"
	"testing"
	"time"
)

func TestNewID(t *testing.T) {
	id := NewID("test")
	if !strings.HasPrefix(id, "test_") {
		t.Errorf("expected prefix 'test_', got %q", id)
	}
	if len(id) < 10 {
		t.Errorf("id too short: %q", id)
	}
}

func TestNewID_Unique(t *testing.T) {
	a := NewID("xxx")
	b := NewID("xxx")
	if a == b {
		t.Errorf("expected unique IDs, got %q and %q", a, b)
	}
}

func TestContactID(t *testing.T) {
	id := ContactID()
	if !strings.HasPrefix(id, PrefixContact+"_") {
		t.Errorf("expected prefix %s_, got %q", PrefixContact, id)
	}
}

func TestDealID(t *testing.T) {
	id := DealID()
	if !strings.HasPrefix(id, PrefixDeal+"_") {
		t.Errorf("expected prefix %s_, got %q", PrefixDeal, id)
	}
}

func TestActivityID(t *testing.T) {
	id := ActivityID()
	if !strings.HasPrefix(id, PrefixActivity+"_") {
		t.Errorf("expected prefix %s_, got %q", PrefixActivity, id)
	}
}

func TestMemoID(t *testing.T) {
	id := MemoID()
	if !strings.HasPrefix(id, PrefixMemo+"_") {
		t.Errorf("expected prefix %s_, got %q", PrefixMemo, id)
	}
}

func TestSlugify_Simple(t *testing.T) {
	slug := Slugify("John Smith")
	if slug != "john-smith" {
		t.Errorf("expected 'john-smith', got %q", slug)
	}
}

func TestSlugify_Mixed(t *testing.T) {
	slug := Slugify("ABC 科技")
	if slug != "abc" {
		t.Errorf("expected 'abc', got %q", slug)
	}
}

func TestSlugify_ChineseOnly(t *testing.T) {
	slug := Slugify("张三")
	if slug != "" {
		t.Errorf("expected empty for pure CJK, got %q", slug)
	}
}

func TestSlugify_SpecialChars(t *testing.T) {
	slug := Slugify("Hello_World! @Company")
	if slug != "helloworld-company" {
		t.Errorf("expected 'helloworld-company', got %q", slug)
	}
}

func TestNewContact(t *testing.T) {
	c := NewContact()
	if c.ID != "" {
		t.Errorf("expected empty ID, got %q", c.ID)
	}
	if c.CreatedAt == "" {
		t.Error("expected non-empty CreatedAt")
	}
	if c.UpdatedAt == "" {
		t.Error("expected non-empty UpdatedAt")
	}
	if c.CreatedAt != c.UpdatedAt {
		t.Error("CreatedAt should equal UpdatedAt on creation")
	}
}

func TestNewDeal(t *testing.T) {
	d := NewDeal()
	if d.Stage != StageLead {
		t.Errorf("expected stage %q, got %q", StageLead, d.Stage)
	}
	if d.Currency != "CNY" {
		t.Errorf("expected currency 'CNY', got %q", d.Currency)
	}
	if d.CreatedAt == "" {
		t.Error("expected non-empty CreatedAt")
	}
}

func TestStageTransitionAllowed(t *testing.T) {
	tests := []struct {
		from string
		to   string
		want bool
	}{
		{StageLead, StageQualified, true},
		{StageQualified, StageProposal, true},
		{StageProposal, StageNegotiation, true},
		{StageNegotiation, StageWon, true},
		{StageNegotiation, StageLost, true},
		{StageWon, StageLead, false},
		{StageWon, StageLost, false},
		{StageLost, StageLead, false},
		{StageLost, StageWon, false},
		{StageQualified, StageLead, true}, // 允许后退
	}
	for _, tt := range tests {
		got := StageTransitionAllowed(tt.from, tt.to)
		if got != tt.want {
			t.Errorf("StageTransitionAllowed(%q, %q) = %v, want %v", tt.from, tt.to, got, tt.want)
		}
	}
}

func TestFieldHistory_IsCurrent(t *testing.T) {
	h := FieldHistory{Value: "旧公司", To: "~"}
	if !h.IsCurrent() {
		t.Error("expected current when To is '~'")
	}
	h.To = ""
	if !h.IsCurrent() {
		t.Error("expected current when To is empty")
	}
	h.To = "2026-01-01"
	if h.IsCurrent() {
		t.Error("expected not current when To is a date")
	}
}

func TestFieldHistory_ActiveUntil(t *testing.T) {
	h := FieldHistory{Value: "旧公司", To: "~"}
	if h.ActiveUntil() != "9999-12-31" {
		t.Errorf("expected '9999-12-31', got %q", h.ActiveUntil())
	}
	h.To = "2026-06-01"
	if h.ActiveUntil() != "2026-06-01" {
		t.Errorf("expected '2026-06-01', got %q", h.ActiveUntil())
	}
}

func TestMemoIsExpired_NotExpired(t *testing.T) {
	m := &Memo{
		ValidFrom: time.Now().UTC().Format(time.RFC3339),
		Decay:     "180d",
	}
	if m.IsExpired("") {
		t.Error("memo created now should not be expired")
	}
}

func TestMemoIsExpired_WithExplicit(t *testing.T) {
	m := &Memo{
		Expired: true,
	}
	if !m.IsExpired("") {
		t.Error("memo with Expired=true should be expired")
	}
}

func TestMemoIsExpired_EmptyDecay(t *testing.T) {
	m := &Memo{
		ValidFrom: "2020-01-01T00:00:00Z",
	}
	if m.IsExpired("") {
		t.Error("memo with empty decay should not be expired")
	}
}

func TestMemoIsExpired_Never(t *testing.T) {
	m := &Memo{
		ValidFrom: "2020-01-01T00:00:00Z",
		Decay:     "never",
	}
	if m.IsExpired("") {
		t.Error("memo with decay=never should not be expired")
	}
}

func TestMemoIsExpired_PastDecay(t *testing.T) {
	m := &Memo{
		ValidFrom: "2020-01-01T00:00:00Z",
		Decay:     "30d",
	}
	if !m.IsExpired("") {
		t.Error("memo from 2020 with 30d decay should be expired")
	}
}

func TestMemoIsExpired_AsOf(t *testing.T) {
	m := &Memo{
		ValidFrom: "2026-06-01T00:00:00Z",
		Decay:     "30d",
	}
	if m.IsExpired("2026-06-15") {
		t.Error("memo valid from June 1 with 30d decay should not be expired by June 15")
	}
	if !m.IsExpired("2026-07-15") {
		t.Error("memo valid from June 1 with 30d decay should be expired by July 15")
	}
}

func TestParseDecayDuration(t *testing.T) {
	tests := []struct {
		input string
		want  time.Duration
	}{
		{"never", 0},
		{"30d", 30 * 24 * time.Hour},
		{"90d", 90 * 24 * time.Hour},
		{"180d", 180 * 24 * time.Hour},
		{"365d", 365 * 24 * time.Hour},
		{"1y", 365 * 24 * time.Hour},
	}
	for _, tt := range tests {
		got, err := ParseDecayDuration(tt.input)
		if err != nil {
			t.Errorf("ParseDecayDuration(%q) unexpected error: %v", tt.input, err)
		}
		if got != tt.want {
			t.Errorf("ParseDecayDuration(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestNewFieldHistory(t *testing.T) {
	h := NewFieldHistory("some-value", "2026-01-01", "bot1", "reason text")
	if h.Value != "some-value" {
		t.Errorf("expected 'some-value', got %q", h.Value)
	}
	if h.From != "2026-01-01" {
		t.Errorf("expected from '2026-01-01', got %q", h.From)
	}
	if h.To != "~" {
		t.Errorf("expected To '~', got %q", h.To)
	}
	if h.SetBy != "bot1" {
		t.Errorf("expected SetBy 'bot1', got %q", h.SetBy)
	}
	if h.Reason != "reason text" {
		t.Errorf("expected Reason 'reason text', got %q", h.Reason)
	}
}

func TestAmountHistory_IsCurrent(t *testing.T) {
	h := AmountHistory{Value: 1000, To: "~"}
	if !h.IsCurrent() {
		t.Error("expected current when To is '~'")
	}
	h.To = "2026-03-01"
	if h.IsCurrent() {
		t.Error("expected not current when To is a date")
	}
}

func TestNewActivity(t *testing.T) {
	a := NewActivity()
	if a.ID != "" {
		t.Errorf("expected empty ID, got %q", a.ID)
	}
	if a.Timestamp == "" {
		t.Error("expected non-empty Timestamp")
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig("/tmp/test")
	if cfg.Version != "1.0.0" {
		t.Errorf("expected version '1.0.0', got %q", cfg.Version)
	}
	if cfg.User.Timezone != "Asia/Shanghai" {
		t.Errorf("expected timezone 'Asia/Shanghai', got %q", cfg.User.Timezone)
	}
	if cfg.Memory.DefaultDecay != "180d" {
		t.Errorf("expected default decay '180d', got %q", cfg.Memory.DefaultDecay)
	}
	if len(cfg.Search.Strategies) != 3 {
		t.Errorf("expected 3 search strategies, got %d", len(cfg.Search.Strategies))
	}
}

func TestValidStages(t *testing.T) {
	if len(ValidStages) != 6 {
		t.Errorf("expected 6 valid stages, got %d", len(ValidStages))
	}
	expected := []string{StageLead, StageQualified, StageProposal, StageNegotiation, StageWon, StageLost}
	for i, s := range expected {
		if ValidStages[i] != s {
			t.Errorf("ValidStages[%d] = %q, want %q", i, ValidStages[i], s)
		}
	}
}

func TestEventConstants(t *testing.T) {
	if EventContactCreated != "contact.created" {
		t.Errorf("unexpected EventContactCreated: %q", EventContactCreated)
	}
	if EventDealClosedWon != "deal.closed_won" {
		t.Errorf("unexpected EventDealClosedWon: %q", EventDealClosedWon)
	}
	if EventActivityLogged != "activity.logged" {
		t.Errorf("unexpected EventActivityLogged: %q", EventActivityLogged)
	}
	if EventMemoryDecayed != "memory.decayed" {
		t.Errorf("unexpected EventMemoryDecayed: %q", EventMemoryDecayed)
	}
}

func TestConfigToJSON(t *testing.T) {
	cfg := DefaultConfig("/tmp/test")
	json, err := cfg.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON error: %v", err)
	}
	if len(json) == 0 {
		t.Error("expected non-empty JSON")
	}
}
