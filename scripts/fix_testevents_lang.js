/**
 * fix_testevents_lang.js — PR 9a test fix
 *
 * Purpose: Add i18n import and `i18n.SetLang("en")` call at the start
 * of TestEvents in cmd_test.go. Without this, TestEvents inherits the
 * Chinese language setting from a preceding test (global state leak),
 * causing assertion failures.
 *
 * Input:  internal/cmd/cmd_test.go
 * Output: Same file with i18n import + SetLang("en") added.
 *
 * Usage:  node scripts/fix_testevents_lang.js
 *
 * Status: One-shot script. Used in PR 9a.
 */
const fs = require('fs');

const filepath = 'internal/cmd/cmd_test.go';
let content = fs.readFileSync(filepath, 'utf8');

// Normalize CRLF → LF for matching
const crlf = content.includes('\r\n');
if (crlf) content = content.replace(/\r\n/g, '\n');

// Add i18n import
const hasI18nImport = content.includes('"github.com/AgentPal/AgentCRM/internal/i18n"');
if (!hasI18nImport) {
  content = content.replace(
    '"github.com/AgentPal/AgentCRM/internal/model"',
    '"github.com/AgentPal/AgentCRM/internal/i18n"\n\t\t"github.com/AgentPal/AgentCRM/internal/model"'
  );
}

// Add i18n.SetLang("en") at start of TestEvents
content = content.replace(
  'func TestEvents(t *testing.T) {\n\tdir := t.TempDir()',
  'func TestEvents(t *testing.T) {\n\ti18n.SetLang("en")\n\tdir := t.TempDir()'
);

// Restore CRLF if needed
if (crlf) content = content.replace(/\n/g, '\r\n');

fs.writeFileSync(filepath, content, 'utf8');
console.log('Fixed TestEvents lang + import');
