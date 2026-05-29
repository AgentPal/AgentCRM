/**
 * add_i18n_keys_pr9a.js — PR 9a migration helper
 *
 * Purpose: Add ~22 new i18n translation keys to messages_en.json and
 * messages_zh.json for io.go + store.go + events.go + alert/engine.go.
 * Sorts keys alphabetically and preserves 2-space indent.
 *
 * Input:  internal/i18n/messages_en.json, internal/i18n/messages_zh.json
 * Output: Same files with new keys inserted.
 *
 * Usage:  node scripts/add_i18n_keys_pr9a.js
 *
 * Status: One-shot script, used in PR 9a commit 1.
 *         Reusable pattern for future i18n key batch additions.
 */
const fs = require('fs');

const enKeys = {
  "alert.rule.stale_memory.suggestion": "Run `memory decay-scan` to clean up expired entries",
  "alert.rule.vip_silence.suggestion": "Send a greeting or schedule a follow-up",
  "cmd.io.export.short": "Export all data to JSON or CSV",
  "cmd.io.import.long": "Import contact data from external sources.\n\nSources:\n  vcard  Import vCard (.vcf) files\n  csv    Import CSV files (columns: name,email,company,title,phone,tags,source)",
  "cmd.io.import.short": "Import data (vcard/csv)",
  "error.io.csv.header_required": "CSV file requires header and data rows",
  "error.memory.invalid_action": "invalid action: %s (supported: supersede, keep-both, reject)",
  "flag.io.file": "import file path",
  "flag.io.out": "output directory",
  "flag.io.source": "import source: vcard|csv",
  "output.io.export_activities": "Exported %d activities",
  "output.io.export_alerts": "Exported %d alerts",
  "output.io.export_contacts": "Exported %d contacts",
  "output.io.export_deals": "Exported %d deals",
  "output.io.export_events": "Exported %d events",
  "output.io.import_done": "Imported %d of %d contacts",
  "output.io.import_done_simple": "Imported %d contacts",
  "output.io.import_fail": "  ⚠ Import failed %s: %v",
  "output.io.skip_file": "  ⚠ Skipped %s: %v",
  "output.io.write_fail": "  ⚠ Failed to write file for %s: %v",
  "output.reindex.done": "Reindex complete: %d contacts, %d deals",
  "output.reindex.write_fail": "  ⚠ Failed to write %s: %v",
};

const zhKeys = {
  "alert.rule.stale_memory.suggestion": "运行 memory decay-scan 清理过期条目",
  "alert.rule.vip_silence.suggestion": "主动联系或安排跟进",
  "cmd.io.export.short": "导出所有数据到 JSON 或 CSV",
  "cmd.io.import.long": "从外部源导入联系人数据。\n\n来源:\n  vcard  导入 vCard (.vcf) 文件\n  csv    导入 CSV 文件（列: name,email,company,title,phone,tags,source）",
  "cmd.io.import.short": "导入数据（vcard/csv）",
  "error.io.csv.header_required": "CSV 文件需要表头和数据行",
  "error.memory.invalid_action": "无效动作: %s（支持: supersede, keep-both, reject）",
  "flag.io.file": "导入文件路径",
  "flag.io.out": "输出目录",
  "flag.io.source": "导入源: vcard|csv",
  "output.io.export_activities": "已导出 %d 条活动",
  "output.io.export_alerts": "已导出 %d 条提醒",
  "output.io.export_contacts": "已导出 %d 个联系人",
  "output.io.export_deals": "已导出 %d 个商机",
  "output.io.export_events": "已导出 %d 条事件",
  "output.io.import_done": "已导入 %d / %d 个联系人",
  "output.io.import_done_simple": "已导入 %d 个联系人",
  "output.io.import_fail": "  ⚠ 导入失败 %s: %v",
  "output.io.skip_file": "  ⚠ 跳过 %s: %v",
  "output.io.write_fail": "  ⚠ 写入文件失败 %s: %v",
  "output.reindex.done": "重建完成: %d 联系人, %d 商机",
  "output.reindex.write_fail": "  ⚠ 写入 %s 失败: %v",
};

function addKeys(filepath, newKeys) {
  const raw = fs.readFileSync(filepath, 'utf8');
  const data = JSON.parse(raw);

  // add new keys (overwrite if exists for safety)
  for (const [k, v] of Object.entries(newKeys)) {
    data[k] = v;
  }

  // sort keys alphabetically
  const sorted = {};
  for (const k of Object.keys(data).sort()) {
    sorted[k] = data[k];
  }

  // write with consistent formatting (2-space indent, trailing newline)
  const output = JSON.stringify(sorted, null, 2) + '\n';
  fs.writeFileSync(filepath, output, 'utf8');
  console.log(`Updated ${filepath} — ${Object.keys(newKeys).length} keys added, ${Object.keys(data).length} total`);
}

addKeys('internal/i18n/messages_en.json', enKeys);
addKeys('internal/i18n/messages_zh.json', zhKeys);
