/**
 * fix_events_pr8_bug2.js — PR 8 regression fix (v2, final)
 *
 * Purpose: Fix double-comma artifact introduced by fix_events_pr8_bug.js v1.
 * The v1 script correctly removed the dangling Chinese parentheticals but
 * left trailing double commas (`,,`) in events.go. This script cleans those.
 *
 * Input:  internal/cmd/events.go
 * Output: Same file with `,,` → `,` fix applied.
 *
 * Usage:  node scripts/fix_events_pr8_bug2.js
 *
 * Status: One-shot script, applied after fix_events_pr8_bug.js during PR 9a.
 *         Kept as reference alongside v1 to document the full recovery
 *         from the PR 8 CJK fullwidth character regression.
 */
const fs = require('fs');

const filepath = 'internal/cmd/events.go';
let content = fs.readFileSync(filepath, 'utf8');

content = content.replace(
  'i18n.T("cmd.events.short"),,',
  'i18n.T("cmd.events.short"),'
);
content = content.replace(
  'i18n.T("cmd.events.watch.short"),,',
  'i18n.T("cmd.events.watch.short"),'
);

fs.writeFileSync(filepath, content, 'utf8');
console.log('Fixed double commas');
