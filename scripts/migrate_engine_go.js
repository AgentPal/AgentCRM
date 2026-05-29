/**
 * migrate_engine_go.js — PR 9a migration helper
 *
 * Purpose: Replace Chinese builtin rule strings in internal/alert/engine.go
 * with i18n.T() calls. Migrates 4 rule titles + 4 suggestions.
 * 6 of 8 keys already existed; 2 new keys (vip_silence.suggestion,
 * stale_memory.suggestion) were added in commit 1 of PR 9a.
 *
 * Input:  internal/alert/engine.go
 * Output: Same file with Chinese → i18n.T() replacements.
 *
 * Usage:  node scripts/migrate_engine_go.js
 *
 * Status: One-shot script, used in PR 9a commit 4.
 *         Pattern (i18n.T() replacement in Go source) is reusable
 *         for future package migrations.
 */
const fs = require('fs');

const filepath = 'internal/alert/engine.go';
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

// 2. Replace stale_deal title + suggestion
content = content.replace(
  'Title:      "商机长时间未跟进"',
  'Title:      i18n.T("alert.rule.stale_deal.title")'
);
content = content.replace(
  'Suggestion: "联系客户了解进展，推进商机阶段"',
  'Suggestion: i18n.T("alert.rule.stale_deal.suggestion")'
);

// 3. Replace closing_deadline title + suggestion
content = content.replace(
  'Title:      "商机即将到截止日期"',
  'Title:      i18n.T("alert.rule.closing_deadline.title")'
);
content = content.replace(
  'Suggestion: "确认成交状态或更新预计日期"',
  'Suggestion: i18n.T("alert.rule.closing_deadline.suggestion")'
);

// 4. Replace vip_silence title + suggestion
content = content.replace(
  'Title:      "VIP 联系人长期未联系"',
  'Title:      i18n.T("alert.rule.vip_silence.title")'
);
content = content.replace(
  'Suggestion: "主动联系或安排跟进"',
  'Suggestion: i18n.T("alert.rule.vip_silence.suggestion")'
);

// 5. Replace stale_memory title + suggestion
content = content.replace(
  'Title:      "有记忆条目已过期需要更新"',
  'Title:      i18n.T("alert.rule.stale_memory.title")'
);
content = content.replace(
  'Suggestion: "运行 memory decay-scan 清理过期条目"',
  'Suggestion: i18n.T("alert.rule.stale_memory.suggestion")'
);

if (crlf) content = content.replace(/\n/g, '\r\n');

fs.writeFileSync(filepath, content, 'utf8');
console.log('Migrated alert/engine.go successfully');
