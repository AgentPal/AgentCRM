/**
 * migrate_store_go.js — PR 9a migration helper
 *
 * Purpose: Replace Chinese Reindex() output strings in
 * internal/store/store.go with i18n.T() calls. Migrates:
 *   - 3 skip_file warnings (contact, activity, deal)
 *   - 3 write_fail warnings (contact, activity, deal)
 *   - 1 reindex done summary line
 *
 * All keys reused from the io.go set (output.io.skip_file,
 * output.reindex.write_fail, output.reindex.done).
 *
 * Input:  internal/store/store.go
 * Output: Same file with Chinese → i18n.T() replacements.
 *
 * Usage:  node scripts/migrate_store_go.js
 *
 * Status: One-shot script, used in PR 9a commit 3.
 *         Reusable pattern for future store-layer i18n migration.
 */
const fs = require('fs');

const filepath = 'internal/store/store.go';
let content = fs.readFileSync(filepath, 'utf8');

// Normalize CRLF
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

// 2. Replace skip_file warnings (3 occurrences)
content = content.replace(
  'fmt.Printf("  ⚠ 跳过 %s: %v\\n", slug, err)',
  'fmt.Printf(i18n.T("output.io.skip_file")+"\\n", slug, err)'
);
content = content.replace(
  'fmt.Printf("  ⚠ 跳过活动文件 %s: %v\\n", file, err)',
  'fmt.Printf(i18n.T("output.io.skip_file")+"\\n", file, err)'
);
// deal skip: note the tab before ⚠
content = content.replace(
  'fmt.Printf("  ⚠ 跳过商机 %s: %v\\n", slug, err)',
  'fmt.Printf(i18n.T("output.io.skip_file")+"\\n", slug, err)'
);

// 3. Replace write_fail warnings (3 occurrences - contact, activity, deal)
// Contact write fail (line 138): "  ⚠ 写入联系人 %s 失败: %v"
content = content.replace(
  'fmt.Printf("  ⚠ 写入联系人 %s 失败: %v\\n", slug, err)',
  'fmt.Printf(i18n.T("output.reindex.write_fail")+"\\n", slug, err)'
);
// Activity write fail (line 171): "  ⚠ 写入活动 %s 失败: %v"
content = content.replace(
  'fmt.Printf("  ⚠ 写入活动 %s 失败: %v\\n", a.ID, err)',
  'fmt.Printf(i18n.T("output.reindex.write_fail")+"\\n", a.ID, err)'
);
// Deal write fail (line 218): "  ⚠ 写入商机 %s 失败: %v"
content = content.replace(
  'fmt.Printf("  ⚠ 写入商机 %s 失败: %v\\n", slug, err)',
  'fmt.Printf(i18n.T("output.reindex.write_fail")+"\\n", slug, err)'
);

// 4. Replace reindex done message
content = content.replace(
  'fmt.Printf("重建完成: %d 联系人, %d 商机\\n", len(contactSlugs), len(dealFiles))',
  'fmt.Printf(i18n.T("output.reindex.done")+"\\n", len(contactSlugs), len(dealFiles))'
);

// Restore CRLF
if (crlf) content = content.replace(/\n/g, '\r\n');

fs.writeFileSync(filepath, content, 'utf8');
console.log('Migrated store.go successfully');
