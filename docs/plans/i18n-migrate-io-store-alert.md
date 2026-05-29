# Plan: i18n Migration — io.go + store.go + events.go + alert/engine.go (PR 9a)

## 1. Scope Confirmation — Key Count by File

### cmd/io.go — keys needed

| Area | Total | Already exist | Net new |
|---|---|---|---|
| `cmd.io.*` | 3 | 0 | 3 |
| `flag.io.*` | 3 | 1 (`flag.io.format`) | 2 |
| `error.io.*` | 2 | 1 (`error.io.file.required`) | 1 |
| `output.io.*` | 12 | 1 (`output.io.export.done`) | 11 |
| Total | **20** | **3** | **17** |

### store/store.go (Reindex) — keys needed

| Area | Total | Already exist | Net new |
|---|---|---|---|
| `output.reindex.*` | 3 | 0 | 3 (skip reuses `output.io.skip_file`, write_fail new, done new) |
| Total | **3** | **0** | **2 net new + 1 reuse** |

### store/events.go (Propose/Commit errors) — keys needed

| Area | Total | Already exist | Net new |
|---|---|---|---|
| `error.memory.scope.format` | 1 | 1 | 0 |
| `error.memory.invalid_action` | 1 | 0 | 1 |
| Total | **2** | **1** | **1** |

### alert/engine.go (built-in rule suggestions) — keys needed

| Area | Total | Already exist | Net new |
|---|---|---|---|
| `alert.rule.*.title` | 4 | 4 | 0 |
| `alert.rule.*.suggestion` | 4 | 2 | 2 |
| Total | **8** | **6** | **2** |

### Grand total: ~21 net new translation keys

- io.go: 17 new
- store.go: 2 new (1 reused from io)
- events.go: 1 new
- alert/engine.go: 2 new

---

## 2. PR Boundary

### In scope (PR 9a)

- **`internal/cmd/io.go`**: 28 Chinese strings → i18n.T().
  - Short/Long descriptions, flag descriptions, output messages, stderr warnings, 1 error.
- **`internal/store/store.go`**: 7 Chinese strings in Reindex() → i18n.T().
  - `fmt.Printf`/`fmt.Fprintf` warning messages + "重建完成" summary line.
- **`internal/store/events.go`**: 2 Chinese `fmt.Errorf` in Propose()/Commit() → i18n.T().
  - Scope format error (key exists: `error.memory.scope.format`).
  - Invalid action error (new key: `error.memory.invalid_action`).
- **`internal/alert/engine.go`**: 8 Chinese strings in `builtinRules()` → i18n.T().
  - 4 rule titles, 4 suggestions. 6 keys exist, 2 net new.

### Deferred (PR 9b)

- **`internal/model/config.go`**: `PreservePatterns` default value (`[]string{"决策*", "偏好*", "*性格*"}`).
  - Product decision about English-friendly default value, not a translation.
  - Key `config.memory.preserve_patterns` already exists in JSON.

### Excluded (already English)

- io.go line 42: `"unsupported format: %s (supported: json, csv)"` — already English.
- io.go line 76: `"unsupported source: %s (supported: vcard, csv)"` — already English.

---

## 3. Cross-Package Considerations

This is the first PR to import `internal/i18n` outside `internal/cmd/`:
- `internal/store/store.go` → imports `internal/i18n`
- `internal/store/events.go` → imports `internal/i18n`
- `internal/alert/engine.go` → imports `internal/i18n`

**Safety verified**: `internal/i18n` imports only stdlib (`embed`, `encoding/json`, `fmt`, `os`, `strings`, `sync`). Zero circular dependency risk. Confirmed by reading `i18n.go` source.

---

## 4. Translation Key Draft

### 4.1 io.go — new cmd.* keys

| Key | en | zh |
|---|---|---|
| `cmd.io.export.short` | Export all data to JSON or CSV | 导出所有数据到 JSON 或 CSV |
| `cmd.io.import.short` | Import data (vcard/csv) | 导入数据（vcard/csv） |
| `cmd.io.import.long` | Import contact data from external sources.\n\nSources:\n  vcard  Import vCard (.vcf) files\n  csv    Import CSV files (columns: name,email,company,title,phone,tags,source) | 从外部源导入联系人数据。\n\n来源:\n  vcard  导入 vCard (.vcf) 文件\n  csv    导入 CSV 文件（列: name,email,company,title,phone,tags,source） |

### 4.2 io.go — new flag.* keys

| Key | en | zh |
|---|---|---|
| `flag.io.format` | export format: json\|csv | (exists: "导出格式: json\|csv") |
| `flag.io.out` | output directory | 输出目录 |
| `flag.io.source` | import source: vcard\|csv | 导入源: vcard\|csv |
| `flag.io.file` | import file path | 导入文件路径 |

### 4.3 io.go — new error.* keys

| Key | en | zh |
|---|---|---|
| `error.io.file.required` | (exists) | (exists) |
| `error.io.csv.header_required` | CSV file requires header and data rows | CSV 文件需要表头和数据行 |

### 4.4 io.go — new output.* keys

| Key | en | zh |
|---|---|---|
| `output.io.skip_file` |   ⚠ Skipped %s: %v |   ⚠ 跳过 %s: %v |
| `output.io.import_fail` |   ⚠ Import failed %s: %v |   ⚠ 导入失败 %s: %v |
| `output.io.write_fail` |   ⚠ Failed to write file for %s: %v |   ⚠ 写入文件失败 %s: %v |
| `output.io.export_contacts` | Exported %d contacts | 已导出 %d 个联系人 |
| `output.io.export_deals` | Exported %d deals | 已导出 %d 个商机 |
| `output.io.export_activities` | Exported %d activities | 已导出 %d 条活动 |
| `output.io.export_events` | Exported %d events | 已导出 %d 条事件 |
| `output.io.export_alerts` | Exported %d alerts | 已导出 %d 条提醒 |
| `output.io.import_done` | Imported %d/%d contacts | 已导入 %d/%d 个联系人 |
| `output.io.import_done_simple` | Imported %d contacts | 已导入 %d 个联系人 |
| `output.io.export.done` | (exists: "Export complete: %s") | (exists: "导出完成: %s") |

### 4.5 store.go — new output.* keys

| Key | en | zh |
|---|---|---|
| `output.reindex.write_fail` |   ⚠ Failed to write %s: %v |   ⚠ 写入 %s 失败: %v |
| `output.reindex.done` | Reindex complete: %d contacts, %d deals | 重建完成: %d 联系人, %d 商机 |

### 4.6 events.go — new error.* keys

| Key | en | zh |
|---|---|---|
| `error.memory.scope.format` | (exists: "invalid scope format, expected contact:\<id\>") | (exists) |
| `error.memory.invalid_action` | invalid action: %s (supported: supersede, keep-both, reject) | 无效动作: %s（支持: supersede, keep-both, reject） |

### 4.7 alert/engine.go — missing suggestion keys

| Key | en | zh |
|---|---|---|
| `alert.rule.vip_silence.title` | (exists) | (exists) |
| `alert.rule.vip_silence.suggestion` | Send a greeting or schedule a follow-up | 发送问候或安排跟进 |
| `alert.rule.stale_memory.title` | (exists) | (exists) |
| `alert.rule.stale_memory.suggestion` | Run `memory decay-scan` to clean up expired entries | 运行 memory decay-scan 清理过期条目 |

---

## 5. Existing Key Reuse

Keys that already exist and will be referenced (no new entry needed):

| Existing key | Used in |
|---|---|
| `flag.io.format` | `exportCmd` —format flag desc |
| `output.io.export.done` | exportJSON + exportCSV completion |
| `error.io.file.required` | importCmd —file validation |
| `error.memory.scope.format` | store/events.go Propose() |
| `alert.rule.stale_deal.title` + `.suggestion` | alert/engine.go |
| `alert.rule.closing_deadline.title` + `.suggestion` | alert/engine.go |
| `alert.rule.vip_silence.title` | alert/engine.go |
| `alert.rule.stale_memory.title` | alert/engine.go |

---

## 6. Special Attention Items

### 6.1 Stderr vs stdout in io.go

Some warning messages go to `os.Stderr`, others to `os.Stdout`. Preserve output stream choice after migration:

- `os.Stderr`: skip warnings (line 101, 121), import failures (341-345, 424-428)
- `os.Stdout`: export counts, done messages, import counts

No change in behavior — just replace format strings.

### 6.2 store.go Reindex output format

The reindex warnings use `fmt.Printf` to stdout (not stderr). This is intentional for `agentcrm reindex` CLI UX. Keep stdout after migration.

### 6.3 Layout-embedded characters in keys

Several keys embed leading spaces or ⚠ characters:
```
output.io.skip_file:  "  ⚠ Skipped %s: %v" / "  ⚠ 跳过 %s: %v"
output.io.write_fail: "  ⚠ Failed to write %s: %v" / "  ⚠ 写入 %s 失败: %v"
output.reindex.write_fail: (same pattern)
```

This follows the precedent from PR 8 (`output.alert.scan.suggestion` embeds `"  Suggestion: %s"`).

### 6.4 No sentinel errors needed

Unlike PR 8 (memory.go parameter validation), the errors in this PR are either:
- Single-use `fmt.Errorf` with i18n.T() directly (io.go, events.go)
- Not errors at all (Printf output in store.go, rule data in engine.go)

No new sentinel errors or userFacingError extensions needed.

---

## 7. Test Impact Assessment

### Existing i18n test coverage (cmd_i18n_test.go)

Already covers: contact, deal, activity, alert, events, memory list — all passing.
Does NOT cover: io.go (export/import), store.go (reindex), alert/engine.go rule strings.

### Test assertions that will change

**`cmd_more_test.go` lines ~95, 136**: Assertions checking `"已导入"` in Chinese output from import commands.
→ These will break when io.go switches to i18n.T() with English default.
→ Fix: change to check English `"Imported"` instead.

### No new i18n tests needed

The alert/events/memory i18n tests already exist in cmd_i18n_test.go. The alert/engine.go rule strings are tested implicitly through alert list output (already covered in `TestI18N_AlertEnglish/Chinese`).

No new test functions needed for this PR.

---

## 8. Commit Splitting Plan

### Commit 1 — `feat(i18n): add ~21 translation keys for io/store/alert`

**Modified:**
- `internal/i18n/messages_en.json` — add 21 new English keys
- `internal/i18n/messages_zh.json` — add 21 new Chinese translations
- `docs/i18n-design.md` — verify glossary is current (no new terms needed)

**Verify:**
- `TestJSONIntegrity` passes (no duplicate/missing keys)
- `TestNoForbiddenTerms` passes (no "customer"/"opportunity" violations)

### Commit 2 — `refactor(cmd): migrate io.go to i18n.T()`

**Modified:**
- `internal/cmd/io.go`
  - Add import `"github.com/AgentPal/AgentCRM/internal/i18n"`
  - Replace `Short`/`Long` descriptions → `i18n.T()`
  - Replace flag description strings → `i18n.T()`
  - Replace `fmt.Errorf` Chinese → `fmt.Errorf(i18n.T(...))` (2 locations)
  - Replace all `fmt.Printf`/`fmt.Fprintf` output → `i18n.T()` (all export/import messages)

### Commit 3 — `refactor(store): migrate store.go + events.go to i18n.T()`

**Modified:**
- `internal/store/store.go`
  - Add import `"github.com/AgentPal/AgentCRM/internal/i18n"`
  - Replace 7 Chinese `fmt.Printf` strings in Reindex() → `i18n.T()`
- `internal/store/events.go`
  - Add import `"github.com/AgentPal/AgentCRM/internal/i18n"`
  - Replace 2 Chinese `fmt.Errorf` in Propose()/Commit() → `fmt.Errorf(i18n.T(...))`

### Commit 4 — `refactor(alert): migrate alert/engine.go to i18n.T() + fix test assertions`

**Modified:**
- `internal/alert/engine.go`
  - Add import `"github.com/AgentPal/AgentCRM/internal/i18n"`
  - Replace 8 Chinese strings in `builtinRules()` → `i18n.T()` (4 titles + 4 suggestions)
  - 6 keys already exist, 2 new keys added in commit 1
- `internal/cmd/cmd_more_test.go`
  - Fix `"已导入"` assertions → `"Imported"` (lines ~95, 136)

---

## 9. Implementation Checklist

- [ ] Commit 1: Add 21 new keys to messages_en.json + messages_zh.json
- [ ] Commit 1: Verify TestJSONIntegrity + TestNoForbiddenTerms pass
- [ ] Commit 2: Import i18n in io.go, replace all Chinese strings
- [ ] Commit 2: Verify `go vet ./internal/cmd/...` passes
- [ ] Commit 3: Import i18n in store/store.go + store/events.go
- [ ] Commit 3: Replace Chinese strings in both files
- [ ] Commit 3: Verify `go vet ./internal/store/...` passes
- [ ] Commit 4: Import i18n in alert/engine.go, replace rule strings
- [ ] Commit 4: Fix `"已导入"` test assertions in cmd_more_test.go
- [ ] Verify: `go vet ./...` passes (all packages)
- [ ] Verify: full test suite passes (Chinese + English scenarios)
- [ ] Push branch + create PR
