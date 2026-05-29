/**
 * fix_events_pr8_bug.js — PR 8 regression fix (v1, superseded)
 *
 * Purpose: Fix events.go compile error caused by PR 8 migration script
 * missing Chinese fullwidth parentheses (U+FF08). The script left
 * "（多 Agent 协作）" and "（阻塞式轮询）" outside i18n.T() calls,
 * producing broken Go syntax.
 *
 * This is v1 of the fix. It used naive string replacement and worked
 * correctly, but read the full file content rather than only the affected
 * section — overly broad but harmless.
 *
 * Input:  internal/cmd/events.go
 * Output: Same file with dangling Chinese parentheticals removed.
 *
 * Usage:  node scripts/fix_events_pr8_bug.js
 *
 * Status: One-shot script, applied during PR 9a. Superseded by
 *         fix_events_pr8_bug2.js which fixed a double-comma artifact.
 *         Retained as reference — documents the CJK fullwidth character
 *         trap that future migration scripts must handle.
 *
 * Lesson: When matching Go source containing CJK text in regex/string
 *         patterns, always account for fullwidth characters (U+FF00–U+FFEF)
 *         which look like ASCII counterparts but have different code points.
 */
const fs = require('fs');

const filepath = 'internal/cmd/events.go';
let content = fs.readFileSync(filepath, 'utf8');

// Line 19: i18n.T("cmd.events.short")（多 Agent 协作）" → i18n.T("cmd.events.short"),
content = content.replace(
  'i18n.T("cmd.events.short")（多 Agent 协作）"',
  'i18n.T("cmd.events.short"),'
);

// Line 110: i18n.T("cmd.events.watch.short")（阻塞式轮询）" → i18n.T("cmd.events.watch.short"),
content = content.replace(
  'i18n.T("cmd.events.watch.short")（阻塞式轮询）"',
  'i18n.T("cmd.events.watch.short"),'
);

fs.writeFileSync(filepath, content, 'utf8');
console.log('Fixed events.go PR 8 migration bugs');
