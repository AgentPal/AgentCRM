/**
 * migrate_events_store_go.js — PR 9a migration helper
 *
 * Purpose: Replace Chinese fmt.Errorf strings in internal/store/events.go
 * with i18n.T() calls. Two errors migrated:
 *   - Propose() scope format error → i18n.T("error.memory.scope.format")
 *   - Commit() invalid action error → i18n.T("error.memory.invalid_action")
 *
 * Input:  internal/store/events.go
 * Output: Same file with Chinese → i18n.T() replacements.
 *
 * Usage:  node scripts/migrate_events_store_go.js
 *
 * Status: One-shot script, used in PR 9a commit 3.
 *         Reusable pattern for future store-layer i18n migration.
 */
const fs = require('fs');

const filepath = 'internal/store/events.go';
let content = fs.readFileSync(filepath, 'utf8');

const crlf = content.includes('\r\n');
if (crlf) content = content.replace(/\r\n/g, '\n');

// 1. Add i18n import
const hasI18n = content.includes('"github.com/AgentPal/AgentCRM/internal/i18n"');
if (!hasI18n) {
  content = content.replace(
    '"github.com/AgentPal/AgentCRM/internal/model"',
    '"github.com/AgentPal/AgentCRM/internal/i18n"\n\t"github.com/AgentPal/AgentCRM/internal/model"'
  );
}

// 2. Replace Propose scope format error (line 290)
content = content.replace(
  'fmt.Errorf("无效 scope 格式，应如 contact:<id>")',
  'fmt.Errorf(i18n.T("error.memory.scope.format"))'
);

// 3. Replace Commit invalid action error (line 385)
content = content.replace(
  'fmt.Errorf("无效动作: %s (支持: supersede, keep-both, reject)", action)',
  'fmt.Errorf(i18n.T("error.memory.invalid_action"), action)'
);

if (crlf) content = content.replace(/\n/g, '\r\n');

fs.writeFileSync(filepath, content, 'utf8');
console.log('Migrated store/events.go successfully');
