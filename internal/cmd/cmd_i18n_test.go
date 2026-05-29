package cmd

import (
	"os"
	"strings"
	"testing"

	"github.com/AgentPal/AgentCRM/internal/i18n"
)

// TestMain initializes i18n before all cmd package tests.
func TestMain(m *testing.M) {
	i18n.MustInit("")
	os.Exit(m.Run())
}

func TestI18N_ChineseOutput(t *testing.T) {
	// MustInit is called by TestMain or init, so we need to switch to zh
	i18n.SetLang("zh")

	dir := t.TempDir()
	out, err := executeCommand(t, "--data-dir", dir, "init")
	if err != nil {
		t.Fatalf("init failed: %v\noutput: %s", err, out)
	}
	if !strings.Contains(out, "已初始化") {
		t.Errorf("expected Chinese init output, got: %s", out)
	}

	out, err = executeCommand(t, "--data-dir", dir, "contact", "list")
	if err != nil {
		t.Fatalf("contact list failed: %v\noutput: %s", err, out)
	}
	if !strings.Contains(out, "无联系人") {
		t.Errorf("expected Chinese contact list output, got: %s", out)
	}
}

func TestI18N_EnglishDefault(t *testing.T) {
	// Default lang is "en" (reset in case previous test changed it)
	i18n.SetLang("en")

	dir := t.TempDir()
	out, err := executeCommand(t, "--data-dir", dir, "init")
	if err != nil {
		t.Fatalf("init failed: %v\noutput: %s", err, out)
	}
	if !strings.Contains(out, "AgentCRM initialized") {
		t.Errorf("expected English init output, got: %s", out)
	}

	out, err = executeCommand(t, "--data-dir", dir, "contact", "list")
	if err != nil {
		t.Fatalf("contact list failed: %v\noutput: %s", err, out)
	}
	if !strings.Contains(out, "No contacts") {
		t.Errorf("expected English contact list output, got: %s", out)
	}
}

func TestI18N_DealEnglish(t *testing.T) {
	i18n.SetLang("en")

	dir := t.TempDir()
	_, err := executeCommand(t, "--data-dir", dir, "init")
	if err != nil {
		t.Fatalf("init failed: %v", err)
	}

	out, err := executeCommand(t, "--data-dir", dir, "deal", "list")
	if err != nil {
		t.Fatalf("deal list failed: %v\noutput: %s", err, out)
	}
	if !strings.Contains(out, "No deals") {
		t.Errorf("expected English deal list output, got: %s", out)
	}
}

func TestI18N_DealChinese(t *testing.T) {
	i18n.SetLang("zh")

	dir := t.TempDir()
	_, err := executeCommand(t, "--data-dir", dir, "init")
	if err != nil {
		t.Fatalf("init failed: %v", err)
	}

	out, err := executeCommand(t, "--data-dir", dir, "deal", "list")
	if err != nil {
		t.Fatalf("deal list failed: %v\noutput: %s", err, out)
	}
	if !strings.Contains(out, "无商机") {
		t.Errorf("expected Chinese deal list output, got: %s", out)
	}
}

func TestI18N_ActivityEnglish(t *testing.T) {
	i18n.SetLang("en")

	dir := t.TempDir()
	_, err := executeCommand(t, "--data-dir", dir, "init")
	if err != nil {
		t.Fatalf("init failed: %v", err)
	}

	out, err := executeCommand(t, "--data-dir", dir, "activity", "list")
	if err != nil {
		t.Fatalf("activity list failed: %v\noutput: %s", err, out)
	}
	if !strings.Contains(out, "No activities") {
		t.Errorf("expected English activity list output, got: %s", out)
	}
}

func TestI18N_ActivityChinese(t *testing.T) {
	i18n.SetLang("zh")

	dir := t.TempDir()
	_, err := executeCommand(t, "--data-dir", dir, "init")
	if err != nil {
		t.Fatalf("init failed: %v", err)
	}

	out, err := executeCommand(t, "--data-dir", dir, "activity", "list")
	if err != nil {
		t.Fatalf("activity list failed: %v\noutput: %s", err, out)
	}
	if !strings.Contains(out, "无活动记录") {
		t.Errorf("expected Chinese activity list output, got: %s", out)
	}
}

func TestI18N_AlertEnglish(t *testing.T) {
	i18n.SetLang("en")

	dir := t.TempDir()
	_, err := executeCommand(t, "--data-dir", dir, "init")
	if err != nil {
		t.Fatalf("init failed: %v", err)
	}

	out, err := executeCommand(t, "--data-dir", dir, "alert", "list")
	if err != nil {
		t.Fatalf("alert list failed: %v\noutput: %s", err, out)
	}
	if !strings.Contains(out, "No pending alerts") {
		t.Errorf("expected English alert list output, got: %s", out)
	}
}

func TestI18N_AlertChinese(t *testing.T) {
	i18n.SetLang("zh")

	dir := t.TempDir()
	_, err := executeCommand(t, "--data-dir", dir, "init")
	if err != nil {
		t.Fatalf("init failed: %v", err)
	}

	out, err := executeCommand(t, "--data-dir", dir, "alert", "list")
	if err != nil {
		t.Fatalf("alert list failed: %v\noutput: %s", err, out)
	}
	if !strings.Contains(out, "无待处理提醒") {
		t.Errorf("expected Chinese alert list output, got: %s", out)
	}
}

func TestI18N_EventsEnglish(t *testing.T) {
	i18n.SetLang("en")

	dir := t.TempDir()
	_, err := executeCommand(t, "--data-dir", dir, "init")
	if err != nil {
		t.Fatalf("init failed: %v", err)
	}

	out, err := executeCommand(t, "--data-dir", dir, "events", "subscribers", "list")
	if err != nil {
		t.Fatalf("events subscribers list failed: %v\noutput: %s", err, out)
	}
	if !strings.Contains(out, "No subscribers") {
		t.Errorf("expected English events subscribers list output, got: %s", out)
	}
}

func TestI18N_EventsChinese(t *testing.T) {
	i18n.SetLang("zh")

	dir := t.TempDir()
	_, err := executeCommand(t, "--data-dir", dir, "init")
	if err != nil {
		t.Fatalf("init failed: %v", err)
	}

	out, err := executeCommand(t, "--data-dir", dir, "events", "subscribers", "list")
	if err != nil {
		t.Fatalf("events subscribers list failed: %v\noutput: %s", err, out)
	}
	if !strings.Contains(out, "无订阅者") {
		t.Errorf("expected Chinese events subscribers list output, got: %s", out)
	}
}

func TestI18N_MemoryEnglish(t *testing.T) {
	i18n.SetLang("en")

	dir := t.TempDir()
	_, err := executeCommand(t, "--data-dir", dir, "init")
	if err != nil {
		t.Fatalf("init failed: %v", err)
	}

	out, err := executeCommand(t, "--data-dir", dir, "memory", "list", "--scope", "contact:test")
	if err != nil {
		t.Fatalf("memory list failed: %v\noutput: %s", err, out)
	}
	if !strings.Contains(out, "No memories") {
		t.Errorf("expected English memory list output, got: %s", out)
	}
}

func TestI18N_MemoryChinese(t *testing.T) {
	i18n.SetLang("zh")

	dir := t.TempDir()
	_, err := executeCommand(t, "--data-dir", dir, "init")
	if err != nil {
		t.Fatalf("init failed: %v", err)
	}

	out, err := executeCommand(t, "--data-dir", dir, "memory", "list", "--scope", "contact:test")
	if err != nil {
		t.Fatalf("memory list failed: %v\noutput: %s", err, out)
	}
	if !strings.Contains(out, "无记忆") {
		t.Errorf("expected Chinese memory list output, got: %s", out)
	}
}

