/**
 * fix_i18n_test_cleanup2.js — PR 9a test fix (v2, final)
 *
 * Purpose: Add `defer i18n.SetLang("en")` after EVERY `i18n.SetLang("zh")`
 * in cmd_i18n_test.go. v1 (fix_i18n_test_cleanup.js) only fixed one
 * occurrence; this version uses a global regex to catch all.
 *
 * Input:  internal/cmd/cmd_i18n_test.go
 * Output: Same file with all Chinese tests properly restoring English.
 *
 * Usage:  node scripts/fix_i18n_test_cleanup2.js
 *
 * Status: One-shot script. Used in PR 9a to fix test order dependency.
 *         Pattern (global regex with CJK-aware matching) is reusable for
 *         similar bulk test cleanup tasks.
 */
const fs = require('fs');

const filepath = 'internal/cmd/cmd_i18n_test.go';
let content = fs.readFileSync(filepath, 'utf8');

// Normalize CRLF
const crlf = content.includes('\r\n');
if (crlf) content = content.replace(/\r\n/g, '\n');

// Replace EVERY occurrence of:
// i18n.SetLang("zh")\n\n\tdir := t.TempDir()
// with:
// i18n.SetLang("zh")\n\tdefer i18n.SetLang("en")\n\n\tdir := t.TempDir()
// But do it globally
content = content.replace(
  /i18n\.SetLang\("zh"\)\n\n\tdir := t\.TempDir\(\)/g,
  'i18n.SetLang("zh")\n\tdefer i18n.SetLang("en")\n\n\tdir := t.TempDir()'
);

if (crlf) content = content.replace(/\n/g, '\r\n');

fs.writeFileSync(filepath, content, 'utf8');
console.log('Fixed ALL Chinese tests with defer en restore');

// Verify
const zhCount = (content.match(/i18n\.SetLang\("zh"\)/g) || []).length;
const deferCount = (content.match(/defer i18n\.SetLang\("en"\)/g) || []).length;
console.log(`Found ${zhCount} SetLang("zh") and ${deferCount} defer en restore`);
