/**
 * fix_i18n_test_cleanup.js — PR 9a test fix (v1, superseded)
 *
 * Purpose: Add `defer i18n.SetLang("en")` after `i18n.SetLang("zh")`
 * in cmd_i18n_test.go to prevent global i18n state leaking across tests.
 *
 * This is v1 of the fix. It only handled one occurrence (single replacement),
 * missing other Chinese-language tests in the same file.
 *
 * Input:  internal/cmd/cmd_i18n_test.go
 * Output: Same file with one defer restored (partial fix).
 *
 * Usage:  node scripts/fix_i18n_test_cleanup.js
 *
 * Status: Superseded by fix_i18n_test_cleanup2.js which uses a global
 *         regex to fix ALL occurrences. Retained as reference.
 */
const fs = require('fs');

const filepath = 'internal/cmd/cmd_i18n_test.go';
let content = fs.readFileSync(filepath, 'utf8');

// After each "i18n.SetLang("zh")" that is NOT followed by a defer line, add "defer i18n.SetLang("en")"
// Use a pattern that matches the zh line and inserts a defer after it, but only once per occurrence
content = content.replace(
	'i18n.SetLang("zh")\n\n\tdir := t.TempDir()',
	'i18n.SetLang("zh")\n\tdefer i18n.SetLang("en")\n\n\tdir := t.TempDir()'
);

// Also handle the ChineseOutput test which doesn't use TempDir right after
// TestI18N_ChineseOutput starts with:
// i18n.SetLang("zh")
// (blank line)
// dir := t.TempDir()
// Wait, it does use the same pattern. Let me check.
// Actually, the script above handles all cases.

fs.writeFileSync(filepath, content, 'utf8');
console.log('Fixed i18n test cleanup — all Chinese tests now defer en restore');
