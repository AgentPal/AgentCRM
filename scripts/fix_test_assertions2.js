/**
 * fix_test_assertions2.js — PR 9a test fix (v2, final)
 *
 * Purpose: Change ALL occurrences of "已导入" → "Imported" in
 * cmd_more_test.go assertions using a global regex.
 * v1 (fix_test_assertions.js) only caught the first match.
 *
 * Input:  internal/cmd/cmd_more_test.go
 * Output: Same file with all import assertion strings in English.
 *
 * Usage:  node scripts/fix_test_assertions2.js
 *
 * Status: One-shot script. Used in PR 9a.
 */
const fs = require('fs');

const filepath = 'internal/cmd/cmd_more_test.go';
let content = fs.readFileSync(filepath, 'utf8');

const crlf = content.includes('\r\n');
if (crlf) content = content.replace(/\r\n/g, '\n');

content = content.replace(
  /strings\.Contains\(out, "已导入"\)/g,
  'strings.Contains(out, "Imported")'
);

if (crlf) content = content.replace(/\n/g, '\r\n');

fs.writeFileSync(filepath, content, 'utf8');
console.log('Fixed ALL "已导入" assertions in cmd_more_test.go');

const remaining = (content.match(/已导入/g) || []).length;
console.log(`Remaining "已导入" occurrences: ${remaining}`);
