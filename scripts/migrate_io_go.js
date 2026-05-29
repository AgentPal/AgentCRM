/**
 * migrate_io_go.js — PR 9a migration helper
 *
 * Purpose: Replace all Chinese strings in internal/cmd/io.go with i18n.T()
 * calls. Covers: Short/Long descriptions, flag descriptions, fmt.Errorf
 * error messages, fmt.Printf/Fprintf output messages (export counts,
 * import results, skip/warning messages, done messages).
 *
 * Input:  internal/cmd/io.go
 * Output: Same file with all Chinese → i18n.T() replacements.
 *
 * Usage:  node scripts/migrate_io_go.js
 *
 * Status: One-shot script, used in PR 9a commit 2. Largest migration
 *         in the PR 9a series (~28 replacements across 3 output streams).
 *         Reusable pattern for future cmd-layer i18n migration.
 */
const fs = require('fs');

const filepath = 'internal/cmd/io.go';
let content = fs.readFileSync(filepath, 'utf8');

// Normalize CRLF → LF for matching consistency
const crlf = content.includes('\r\n');
if (crlf) content = content.replace(/\r\n/g, '\n');

// 1. Add i18n import (after last existing internal import)
const hasI18nImport = content.includes('"github.com/AgentPal/AgentCRM/internal/i18n"');
if (!hasI18nImport) {
  content = content.replace(
    '"github.com/AgentPal/AgentCRM/internal/model"',
    '"github.com/AgentPal/AgentCRM/internal/i18n"\n\t\t"github.com/AgentPal/AgentCRM/internal/model"'
  );
}

// 2. Replace export Short: "导出所有数据到 JSON 或 CSV"
content = content.replace(
  'Short: "导出所有数据到 JSON 或 CSV"',
  'Short: i18n.T("cmd.io.export.short")'
);

// 3. Replace import Short
content = content.replace(
  'Short: "导入数据（vcard/csv）"',
  'Short: i18n.T("cmd.io.import.short")'
);

// 4. Replace import Long (backtick block)
content = content.replace(
  'Long: `从外部源导入联系人数据。\n\n\t来源:\n\t  vcard  导入 vCard (.vcf) 文件\n\t  csv    导入 CSV 文件（列: name,email,company,title,phone,tags,source）`,',
  'Long: i18n.T("cmd.io.import.long"),'
);

// 5. Replace flag descriptions
content = content.replace(
  '"导出格式: json|csv"',
  'i18n.T("flag.io.format")'
);
content = content.replace(
  '"输出目录"',
  'i18n.T("flag.io.out")'
);
content = content.replace(
  '"导入源: vcard|csv"',
  'i18n.T("flag.io.source")'
);
content = content.replace(
  '"导入文件路径"',
  'i18n.T("flag.io.file")'
);

// 6. Replace fmt.Errorf("--file 是必需的")
content = content.replace(
  'return fmt.Errorf("--file 是必需的")',
  'return fmt.Errorf(i18n.T("error.io.file.required"))'
);

// 7. Replace fmt.Errorf("CSV 文件需要表头和数据行")
content = content.replace(
  'return fmt.Errorf("CSV 文件需要表头和数据行")',
  'return fmt.Errorf(i18n.T("error.io.csv.header_required"))'
);

// 8. Replace export output messages (process each line)
// 8a. "  ⚠ 跳过 %s: %v\n" → i18n.T("output.io.skip_file")
content = content.replace(
  'fmt.Fprintf(os.Stderr, "  ⚠ 跳过 %s: %v\\n", slug, err)',
  'fmt.Fprintf(os.Stderr, i18n.T("output.io.skip_file")+"\\n", slug, err)'
);
content = content.replace(
  'fmt.Fprintf(os.Stderr, "  ⚠ 跳过 %s: %v\\n", f, err)',
  'fmt.Fprintf(os.Stderr, i18n.T("output.io.skip_file")+"\\n", f, err)'
);

// 8b. Export count messages
content = content.replace(
  'fmt.Printf("已导出 %d 个联系人\\n", len(contacts))',
  'fmt.Printf(i18n.T("output.io.export_contacts")+"\\n", len(contacts))'
);
content = content.replace(
  'fmt.Printf("已导出 %d 个商机\\n", len(deals))',
  'fmt.Printf(i18n.T("output.io.export_deals")+"\\n", len(deals))'
);
content = content.replace(
  'fmt.Printf("已导出 %d 条活动\\n", len(activities))',
  'fmt.Printf(i18n.T("output.io.export_activities")+"\\n", len(activities))'
);
content = content.replace(
  'fmt.Printf("已导出 %d 条事件\\n", len(events))',
  'fmt.Printf(i18n.T("output.io.export_events")+"\\n", len(events))'
);

// This one: "已导出 %d 条提醒\n" (only printed if len(alerts) > 0)
content = content.replace(
  'fmt.Printf("已导出 %d 条提醒\\n", len(alerts))',
  'fmt.Printf(i18n.T("output.io.export_alerts")+"\\n", len(alerts))'
);

// 8c. Export done: "导出完成: %s\n" → appears twice (JSON + CSV)
content = content.replace(
  'fmt.Printf("导出完成: %s\\n", outDir)',
  'fmt.Printf(i18n.T("output.io.export.done")+"\\n", outDir)'
);

// 8d. CSV export messages (same as JSON but different variable names)
content = content.replace(
  'fmt.Printf("已导出 %d 个联系人\\n", len(slugs))',
  'fmt.Printf(i18n.T("output.io.export_contacts")+"\\n", len(slugs))'
);
content = content.replace(
  'fmt.Printf("已导出 %d 个商机\\n", len(dealFiles))',
  'fmt.Printf(i18n.T("output.io.export_deals")+"\\n", len(dealFiles))'
);

// 8e. Import failure messages
content = content.replace(
  'fmt.Fprintf(os.Stderr, "  ⚠ 导入失败 %s: %v\\n", c.Name, err)',
  'fmt.Fprintf(os.Stderr, i18n.T("output.io.import_fail")+"\\n", c.Name, err)'
);

content = content.replace(
  'fmt.Fprintf(os.Stderr, "  ⚠ 写入文件失败 %s: %v\\n", c.Name, err)',
  'fmt.Fprintf(os.Stderr, i18n.T("output.io.write_fail")+"\\n", c.Name, err)'
);

// 8f. Import done messages
content = content.replace(
  'fmt.Printf("已导入 %d/%d 个联系人\\n", imported, len(contacts))',
  'fmt.Printf(i18n.T("output.io.import_done")+"\\n", imported, len(contacts))'
);

content = content.replace(
  'fmt.Printf("已导入 %d 个联系人\\n", imported)',
  'fmt.Printf(i18n.T("output.io.import_done_simple")+"\\n", imported)'
);

// 9. Fix trailing comma issue for importCSV's last fmt.Printf
// (no issue — the original already works correctly)

// Restore CRLF if original had it
if (crlf) content = content.replace(/\n/g, '\r\n');

fs.writeFileSync(filepath, content, 'utf8');
console.log('Migrated io.go successfully');
