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
