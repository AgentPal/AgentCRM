package model

import (
	"strings"
	"testing"
	"time"
)

func TestNewEvent(t *testing.T) {
	e := NewEvent(1, "tester", EventContactCreated, `{"id":"cnt_01"}`)
	if e.Seq != 1 {
		t.Errorf("expected seq 1, got %d", e.Seq)
	}
	if e.Actor != "tester" {
		t.Errorf("expected actor tester, got %s", e.Actor)
	}
	if e.Type != EventContactCreated {
		t.Errorf("expected type %s, got %s", EventContactCreated, e.Type)
	}
	if e.Payload != `{"id":"cnt_01"}` {
		t.Errorf("unexpected payload: %s", e.Payload)
	}
	if e.TS == "" {
		t.Error("expected timestamp")
	}
	// Verify it's a valid RFC3339
	_, err := time.Parse(time.RFC3339, e.TS)
	if err != nil {
		t.Errorf("invalid timestamp: %v", err)
	}
}

func TestContact_AsOf(t *testing.T) {
	c := NewContact()
	c.Company = "Current Corp"
	c.Title = "CEO"
	c.CompanyHistory = []FieldHistory{
		{Value: "Old Corp", From: "2020-01-01", To: "2023-12-31"},
		{Value: "New Corp", From: "2024-01-01", To: "~"},
	}
	c.TitleHistory = []FieldHistory{
		{Value: "Engineer", From: "2019-06-01", To: "2022-05-31"},
		{Value: "Manager", From: "2022-06-01", To: "~"},
	}

	// Before any history
	early := c.AsOf("2019-01-01")
	if early.Company != "Current Corp" {
		t.Errorf("expected current value before history, got %s", early.Company)
	}
	if early.Title != "CEO" {
		t.Errorf("expected current value before history, got %s", early.Title)
	}

	// During Old Corp period
	middle := c.AsOf("2022-06-15")
	if middle.Company != "Old Corp" {
		t.Errorf("expected Old Corp, got %s", middle.Company)
	}
	if middle.Title != "Manager" {
		t.Errorf("expected Manager, got %s", middle.Title)
	}

	// After transition
	later := c.AsOf("2024-06-15")
	if later.Company != "New Corp" {
		t.Errorf("expected New Corp, got %s", later.Company)
	}
	if later.Title != "Manager" {
		t.Errorf("expected Manager, got %s", later.Title)
	}

	// Original should be unchanged
	if c.Company != "Current Corp" {
		t.Errorf("original mutated: %s", c.Company)
	}
}

func TestDeal_AsOf(t *testing.T) {
	d := NewDeal()
	d.Stage = "negotiation"
	d.Amount = 30000
	d.StageHistory = []StageHistory{
		{Stage: "lead", EnteredAt: "2024-01-01"},
		{Stage: "qualified", EnteredAt: "2024-02-01"},
		{Stage: "proposal", EnteredAt: "2024-03-01"},
	}
	d.AmountHistory = []AmountHistory{
		{Value: 10000, From: "2024-01-01", To: "2024-01-31"},
		{Value: 20000, From: "2024-02-01", To: "2024-02-28"},
	}

	// Early - lead stage
	early := d.AsOf("2024-01-15")
	if early.Stage != "lead" {
		t.Errorf("expected lead, got %s", early.Stage)
	}
	if early.Amount != 10000 {
		t.Errorf("expected 10000, got %d", early.Amount)
	}

	// After all history entries but amount coverage has ended
	middle := d.AsOf("2024-03-15")
	// Stage: all entries match (entered_at <= 2024-03-15), last one wins
	if middle.Stage != "proposal" {
		t.Errorf("expected proposal, got %s", middle.Stage)
	}
	// Amount: neither history entry covers 2024-03-15, falls to current
	if middle.Amount != 30000 {
		t.Errorf("expected current amount 30000, got %d", middle.Amount)
	}

	// Far future - stage is last matching history entry, amount falls to current
	later := d.AsOf("2024-06-15")
	// Stage: all entries still match (entered_at <= date), last one wins
	if later.Stage != "proposal" {
		t.Errorf("expected proposal (last history match), got %s", later.Stage)
	}
	// Amount: no history covers 2024-06-15, uses current
	if later.Amount != 30000 {
		t.Errorf("expected current amount 30000, got %d", later.Amount)
	}

	// Original should be unchanged
	if d.Stage != "negotiation" {
		t.Errorf("original mutated: %s", d.Stage)
	}
}

func TestDeal_AsOf_EmptyHistory(t *testing.T) {
	d := NewDeal()
	d.Stage = "qualified"
	d.Amount = 50000

	result := d.AsOf("2026-01-01")
	if result.Stage != "qualified" {
		t.Errorf("expected qualified, got %s", result.Stage)
	}
	if result.Amount != 50000 {
		t.Errorf("expected 50000, got %d", result.Amount)
	}
}

func TestParseDecayDuration_Errors(t *testing.T) {
	// Unknown string should fall back to 180d
	dur, err := ParseDecayDuration("invalid")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if dur != 180*24*time.Hour {
		t.Errorf("expected 180d fallback, got %v", dur)
	}

	// Custom duration that time.ParseDuration can handle
	dur, err = ParseDecayDuration("72h")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if dur != 72*time.Hour {
		t.Errorf("expected 72h, got %v", dur)
	}

	// Empty string fallback
	dur, err = ParseDecayDuration("")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if dur != 180*24*time.Hour {
		t.Errorf("expected 180d fallback for empty, got %v", dur)
	}
}

func TestMemoIsExpired_AsOfPast(t *testing.T) {
	m := &Memo{
		ValidFrom: "2020-01-01T00:00:00Z",
		Decay:     "30d",
	}
	if !m.IsExpired("2025-01-01") {
		t.Error("should be expired by 2025")
	}
}

func TestMemoIsExpired_DateOnlyValidFrom(t *testing.T) {
	m := &Memo{
		ValidFrom: "2020-01-01",
		Decay:     "30d",
	}
	if !m.IsExpired("2025-01-01") {
		t.Error("should be expired with date-only valid_from")
	}
}

func TestMemoIsExpired_InvalidValidFrom(t *testing.T) {
	m := &Memo{
		ValidFrom: "not-a-date",
		Decay:     "30d",
	}
	if m.IsExpired("") {
		t.Error("memo with invalid valid_from should not be expired")
	}
}

func TestMemoIsExpired_EmptyValidFrom(t *testing.T) {
	m := &Memo{
		ValidFrom: "",
		Decay:     "30d",
	}
	if m.IsExpired("") {
		t.Error("memo with empty valid_from should not be expired")
	}
}

func TestMemoIsExpired_AsOfZeroTime(t *testing.T) {
	m := &Memo{
		ValidFrom: "2020-01-01T00:00:00Z",
		Decay:     "30d",
	}
	// When asOf is unparseable, IsExpired uses zero time (0001-01-01)
	// which is before any expiry, so returns false
	if m.IsExpired("not-a-date") {
		t.Error("memo with invalid asOf should fall through to zero time (not expired)")
	}
}

func TestNewContact_Defaults(t *testing.T) {
	c := NewContact()
	if c.ID != "" {
		t.Errorf("expected empty ID, got %s", c.ID)
	}
	if c.CreatedAt == "" {
		t.Error("expected non-empty CreatedAt")
	}
	if c.UpdatedAt != c.CreatedAt {
		t.Error("UpdatedAt should equal CreatedAt")
	}
	if c.Emails != nil {
		t.Errorf("expected nil emails, got %v", c.Emails)
	}
}

func TestNewDeal_Defaults(t *testing.T) {
	d := NewDeal()
	if d.Stage != StageLead {
		t.Errorf("expected lead, got %s", d.Stage)
	}
	if d.Currency != "CNY" {
		t.Errorf("expected CNY, got %s", d.Currency)
	}
	if len(d.ContactIDs) != 0 {
		t.Errorf("expected empty ContactIDs, got %v", d.ContactIDs)
	}
}

func TestConfigToJSON_Error(t *testing.T) {
	// Create config with problematic values
	cfg := DefaultConfig("/tmp/test")
	json, err := cfg.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON error: %v", err)
	}
	if !strings.Contains(json, "\"version\"") {
		t.Error("expected version field in JSON")
	}
	if !strings.Contains(json, "\"timezone\"") {
		t.Error("expected timezone in JSON")
	}
}

func TestAmountHistory_IsCurrent_Empty(t *testing.T) {
	h := AmountHistory{Value: 5000, To: ""}
	if !h.IsCurrent() {
		t.Error("expected current when To is empty")
	}
}

func TestStageTransition_WonLost(t *testing.T) {
	if StageTransitionAllowed(StageWon, StageWon) {
		t.Error("won -> won should be disallowed")
	}
	if StageTransitionAllowed(StageLost, StageLost) {
		t.Error("lost -> lost should be disallowed")
	}
}
