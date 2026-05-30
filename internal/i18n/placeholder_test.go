package i18n

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

// countPlaceholders counts the number of format specifiers (%s, %d, %v, %f etc.)
// in a format string, ignoring escaped percent signs (%%).
func countPlaceholders(s string) int {
	var count int
	for i := 0; i < len(s); i++ {
		if s[i] == '%' && i+1 < len(s) {
			next := s[i+1]
			if next == '%' {
				i++ // skip escaped %
				continue
			}
			// Count any printf verb
			if strings.ContainsRune("sdvfxXqtcUbeoEgGw", rune(next)) {
				count++
			}
		}
	}
	return count
}

// TestPlaceholderConsistency verifies that every i18n key has the same number
// of printf placeholders (%s, %d, %v, %f) in both English and Chinese translations.
// This prevents panics when switching languages at runtime with format strings
// that have mismatched argument counts.
//
// Note: TestJSONIntegrity already checks that placeholder types match (via
// PlaceholderString with dedup), but does not catch count differences.
// This test catches cases like en="%s %s" vs zh="%s" which would pass the
// type check but cause runtime argument mismatch.
func TestPlaceholderConsistency(t *testing.T) {
	var en, zh map[string]string
	if err := json.Unmarshal(enJSON, &en); err != nil {
		t.Fatalf("messages_en.json: invalid JSON: %v", err)
	}
	if err := json.Unmarshal(zhJSON, &zh); err != nil {
		t.Fatalf("messages_zh.json: invalid JSON: %v", err)
	}

	var failures []string

	for key, enVal := range en {
		zhVal, ok := zh[key]
		if !ok {
			failures = append(failures, fmt.Sprintf("key %q exists in en but missing in zh", key))
			continue
		}
		enCount := countPlaceholders(enVal)
		zhCount := countPlaceholders(zhVal)
		if enCount != zhCount {
			failures = append(failures, fmt.Sprintf(
				"key %q: en has %d placeholder(s) (%q), zh has %d placeholder(s) (%q)",
				key, enCount, enVal, zhCount, zhVal,
			))
		}
	}

	for key := range zh {
		if _, ok := en[key]; !ok {
			failures = append(failures, fmt.Sprintf("key %q exists in zh but missing in en", key))
		}
	}

	for _, f := range failures {
		t.Error(f)
	}
}
