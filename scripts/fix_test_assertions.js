/**
 * fix_test_assertions.js — PR 9a test fix (v1, superseded)
 *
 * Purpose: Change "已导入" → "Imported" in cmd_more_test.go assertions
 * after io.go's import output switched from Chinese to i18n.T() English default.
 *
 * This is v1 of the fix. It only replaced the first occurrence.
 *
 * Input:  internal/cmd/cmd_more_test.go
 * Output: Same file with one assertion fixed.
 *
 * Usage:  node scripts/fix_test_assertions.js
 *
 * Status: Superseded by fix_test_assertions2.js which uses a global regex
 *         to replace ALL occurrences. Retained as reference.
 */
const fs = require('fs');

const filepath = 'internal/cmd/cmd_more_test.go';
let content = fs.readFileSync(filepath, 'utf8');

content = content.replace(
  'strings.Contains(out, "已导入")',
  'strings.Contains(out, "Imported")'
);

fs.writeFileSync(filepath, content, 'utf8');
console.log('Fixed test assertions');
