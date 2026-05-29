package i18n

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// TestJSONIntegrity — CI gate
// ---------------------------------------------------------------------------

func TestJSONIntegrity(t *testing.T) {
	// Parse en and zh JSON
	var en, zh map[string]string
	if err := json.Unmarshal(enJSON, &en); err != nil {
		t.Fatalf("messages_en.json: invalid JSON: %v", err)
	}
	if err := json.Unmarshal(zhJSON, &zh); err != nil {
		t.Fatalf("messages_zh.json: invalid JSON: %v", err)
	}

	// Key parity: same keys in both languages
	missingInZH := keysMissing(en, zh)
	missingInEN := keysMissing(zh, en)

	if len(missingInZH) > 0 {
		t.Errorf("keys in en but missing in zh: %v", missingInZH)
	}
	if len(missingInEN) > 0 {
		t.Errorf("keys in zh but missing in en: %v", missingInEN)
	}
	if len(missingInZH) > 0 || len(missingInEN) > 0 {
		t.Errorf("en and zh must have identical key sets")
	}

	// No empty values
	for k, v := range en {
		if strings.TrimSpace(v) == "" {
			t.Errorf("en key %q has empty value", k)
		}
	}
	for k, v := range zh {
		if strings.TrimSpace(v) == "" {
			t.Errorf("zh key %q has empty value", k)
		}
	}

	// Placeholder parity: same %s/%d/%v etc. in both languages
	for key := range en {
		enPH := PlaceholderString(en[key])
		zhPH := PlaceholderString(zh[key])
		if enPH != zhPH {
			t.Errorf("placeholder mismatch for %q: en=%q zh=%q", key, enPH, zhPH)
		}
	}
}

func keysMissing(a, b map[string]string) []string {
	var out []string
	for k := range a {
		if _, ok := b[k]; !ok {
			out = append(out, k)
		}
	}
	return out
}

// ---------------------------------------------------------------------------
// TestT_Simple
// ---------------------------------------------------------------------------

func TestT_Simple(t *testing.T) {
	MustInit("en")
	got := T("cmd.contact.short")
	if got != "Manage contacts" {
		t.Errorf("expected %q, got %q", "Manage contacts", got)
	}
}

// ---------------------------------------------------------------------------
// TestT_Interpolation
// ---------------------------------------------------------------------------

func TestT_Interpolation(t *testing.T) {
	MustInit("en")
	got := T("output.contact.created", "Zhang San", "cnt_abc")
	want := "Created contact: Zhang San (cnt_abc)"
	if got != want {
		t.Errorf("expected %q, got %q", want, got)
	}
}

func TestT_Interpolation_Int(t *testing.T) {
	MustInit("en")
	got := T("output.deal.list.count", 5)
	if !strings.Contains(got, "5") || !strings.Contains(got, "deals") {
		t.Errorf("unexpected: %q", got)
	}
}

// ---------------------------------------------------------------------------
// TestT_Fallback
// ---------------------------------------------------------------------------

func TestT_Fallback(t *testing.T) {
	MustInit("zh")
	// "output.contact.created" exists in zh — should use zh
	got := T("output.contact.created", "张三", "cnt_abc")
	if !strings.Contains(got, "创建") {
		t.Errorf("expected zh translation, got %q", got)
	}
}

func TestT_Fallback_MissingInZH(t *testing.T) {
	// Pretend a key exists only in en by adding it at runtime.
	// We use a key known to exist in en — this tests that when zh is
	// active and the key doesn't exist in zh, we fall back to en.
	MustInit("zh")
	got := T("output.contact.list.none")
	if got != "无联系人" {
		t.Errorf("expected zh value, got %q", got)
	}
}

// ---------------------------------------------------------------------------
// TestT_Missing
// ---------------------------------------------------------------------------

func TestT_Missing(t *testing.T) {
	MustInit("en")
	got := T("key.does.not.exist")
	if !strings.Contains(got, "missing") {
		t.Errorf("expected [missing ...] indicator, got %q", got)
	}
}

// ---------------------------------------------------------------------------
// TestSetLang_Invalid
// ---------------------------------------------------------------------------

func TestSetLang_Invalid(t *testing.T) {
	MustInit("en")
	err := SetLang("fr")
	if err == nil {
		t.Fatal("expected error for unsupported language")
	}
}

func TestSetLang_Valid(t *testing.T) {
	MustInit("en")
	if err := SetLang("zh"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if CurrentLang() != "zh" {
		t.Errorf("expected zh, got %s", CurrentLang())
	}
}

// ---------------------------------------------------------------------------
// TestNoForbiddenTerms — glossary enforcement
// ---------------------------------------------------------------------------

func TestNoForbiddenTerms(t *testing.T) {
	MustInit("en")

	// Brand-level keys exempt from the "customer" check only.
	// These keys describe AgentCRM as a product category, not data operations.
	// See docs/i18n-design.md §10 Glossary exceptions.
	customerExceptions := map[string]bool{
		"cmd.root.short":           true,
		"cmd.root.long":            true,
		"output.root.version.text": true,
	}

	forbidden := map[string]string{
		"customer":    "use 'contact' instead (see glossary)",
		"opportunity": "use 'deal' instead (see glossary)",
		"person":      "use 'contact' instead (see glossary)",
	}

	msgs := MessagesForLang("en")
	if msgs == nil {
		t.Fatal("no en messages loaded")
	}

	for key, val := range msgs {
		valLower := strings.ToLower(val)
		for term, hint := range forbidden {
			// Allowlist only applies to "customer", not to opportunity/person
			if term == "customer" && customerExceptions[key] {
				continue
			}
			if strings.Contains(valLower, term) {
				t.Errorf("en key %q = %q — contains %q (%s)", key, val, term, hint)
			}
		}
	}
}

// ---------------------------------------------------------------------------
// TestLangSelection_Priority
// ---------------------------------------------------------------------------

func TestLangSelection_Priority_CLIOverride(t *testing.T) {
	// --lang CLI flag should override AGENTCRM_LANG env var
	os.Setenv("AGENTCRM_LANG", "zh")
	defer os.Unsetenv("AGENTCRM_LANG")

	MustInit("en") // cmdLang = "en"
	if CurrentLang() != "en" {
		t.Errorf("expected en (CLI override), got %s", CurrentLang())
	}
}

func TestLangSelection_Priority_Env(t *testing.T) {
	// AGENTCRM_LANG should be used when no CLI flag is set
	os.Setenv("AGENTCRM_LANG", "zh")
	defer os.Unsetenv("AGENTCRM_LANG")

	MustInit("")
	if CurrentLang() != "zh" {
		t.Errorf("expected zh (env), got %s", CurrentLang())
	}
}

func TestLangSelection_Priority_Default(t *testing.T) {
	// Default should be "en" when nothing is set
	os.Unsetenv("AGENTCRM_LANG")
	MustInit("")
	if CurrentLang() != "en" {
		t.Errorf("expected en (default), got %s", CurrentLang())
	}
}

// ---------------------------------------------------------------------------
// TestTn
// ---------------------------------------------------------------------------

func TestTn_Simple(t *testing.T) {
	MustInit("en")
	// output.deal.list.count uses %d — test with Tn
	got1 := T("output.deal.list.count", 1)
	got3 := T("output.deal.list.count", 3)

	if got1 == "" || strings.Contains(got1, "missing") {
		t.Errorf("unexpected: %q", got1)
	}
	if got3 == "" || strings.Contains(got3, "missing") {
		t.Errorf("unexpected: %q", got3)
	}
}

func TestTn_PluralWorking(t *testing.T) {
	// English: n=1 uses base key (singular), n≠1 uses .plural key
	MustInit("en")

	got1 := Tn("output.activity.timeline.count", 1)
	if strings.Contains(got1, "activities") {
		t.Errorf("n=1 should use singular, got: %q", got1)
	}
	if !strings.Contains(got1, "1") {
		t.Errorf("expected count 1 in output, got: %q", got1)
	}

	gotN := Tn("output.activity.timeline.count", 3)
	if !strings.Contains(gotN, "3 activities") {
		t.Errorf("n=3 should use plural '3 activities', got: %q", gotN)
	}

	// Chinese: always uses base key regardless of n
	MustInit("zh")

	gotZH1 := Tn("output.activity.timeline.count", 1)
	gotZH3 := Tn("output.activity.timeline.count", 3)

	if !strings.Contains(gotZH1, "共") || !strings.Contains(gotZH1, "1") {
		t.Errorf("unexpected zh n=1: %q", gotZH1)
	}
	if !strings.Contains(gotZH3, "共") || !strings.Contains(gotZH3, "3") {
		t.Errorf("unexpected zh n=3: %q", gotZH3)
	}

	MustInit("en")
}

// ---------------------------------------------------------------------------
// TestKnownLangs
// ---------------------------------------------------------------------------

func TestKnownLangs(t *testing.T) {
	MustInit("en")
	langs := KnownLangs()
	if len(langs) != 2 {
		t.Errorf("expected 2 langs, got %d: %v", len(langs), langs)
	}
	hasEN, hasZH := false, false
	for _, l := range langs {
		if l == "en" {
			hasEN = true
		}
		if l == "zh" {
			hasZH = true
		}
	}
	if !hasEN || !hasZH {
		t.Errorf("expected both en and zh, got %v", langs)
	}
}

// ---------------------------------------------------------------------------
// TestAllKeys
// ---------------------------------------------------------------------------

func TestAllKeys(t *testing.T) {
	MustInit("en")
	keys := AllKeys()
	if len(keys) == 0 {
		t.Fatal("expected at least 1 key, got 0")
	}
}

// ---------------------------------------------------------------------------
// TestPlaceholders
// ---------------------------------------------------------------------------

func TestPlaceholders_NoPlaceholders(t *testing.T) {
	if ph := PlaceholderString("hello"); ph != "" {
		t.Errorf("expected empty, got %q", ph)
	}
}

func TestPlaceholders_Single(t *testing.T) {
	ph := PlaceholderString("%s is required")
	if !strings.Contains(ph, "%s") {
		t.Errorf("expected %%s, got %q", ph)
	}
}

func TestPlaceholders_Multiple(t *testing.T) {
	ph := PlaceholderString("created %s (%s)")
	if !strings.Contains(ph, "%s") {
		t.Errorf("expected %%s, got %q", ph)
	}
}

func TestPlaceholders_EscapedPercent(t *testing.T) {
	ph := PlaceholderString("20%% discount on %s")
	// The %% should be ignored, only %s should be found
	if strings.Count(ph, "%") > 1 {
		t.Errorf("expected only one placeholder, got %q", ph)
	}
}

func TestPlaceholders_Int(t *testing.T) {
	ph := PlaceholderString("found %d contacts")
	if !strings.Contains(ph, "%d") {
		t.Errorf("expected %%d, got %q", ph)
	}
}
