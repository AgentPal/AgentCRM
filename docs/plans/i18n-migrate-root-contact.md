# Plan: i18n Migration — root.go + contact.go (PR 6)

## 1. Scope Confirmation

### root.go — 10 keys

| Area | Keys | Details |
|---|---|---|
| `cmd.root.*` | 5 | short, long, version, init, reindex |
| `flag.*` | 3 | data_dir, format, actor |
| `output.root.*` | 2 | version.text, init.success |
| Total | **10** | |

### contact.go — 45 keys

| Area | Keys | Details |
|---|---|---|
| `cmd.contact.*` | 9 | short, upsert.short, get.short, search.short, list.short, update.short, history.short, merge.short, merge.long |
| `flag.contact.*` | 16 | name, email, company, title, phone, source, tag, as_of, limit, tag_filter, updated_since, has_field, set, reason, field, merge_reason |
| `error.contact.*` | 3 | name.required, set.required, field.required |
| `output.contact.*` | 17 | created, updated, upsert.success, upsert.company, upsert.title, get.as_of, search.none, search.recent, list.none, list.count, update.same, update.changed, update.set, history.none, history.to, merge.success, merge.reason |
| Total | **45** | |

### Grand total: 55 keys

- Already in PR 4 sample set: ~10 keys (cmd.contact.short, cmd.contact.upsert.short, output.contact.created, output.contact.updated, output.contact.list.none, output.root.init.success, error.contact.name.required, error.contact.field.required, flag.contact.name, flag.contact.email)
- **Net new keys needed: ~45**
- The estimate in §12 (~70) was for root + contact + deal — deal.go is not in this PR.

---

## 2. Sentinel Error Extraction Plan

### Current fmt.Errorf calls in contact.go (Chinese ones only)

| Line | Current | Proposed sentinel | Package |
|---|---|---|---|
| 31 | `"--name 是必需的"` | `ErrContactNameRequired` | sentinel |
| 269 | `"至少需要一个 --set 参数"` | `ErrContactSetRequired` | sentinel |
| 320 | `"--field 是必需的"` | `ErrContactFieldRequired` | sentinel |

Other error returns in contact.go are already English (wrapping store errors) — these stay as-is.

### Decision: Where to put sentinels

**Option A — Top of each command file:**
```go
// internal/cmd/contact.go
var ErrContactNameRequired = errors.New("--name is required")
```

**Option B — Centralized internal/cmd/errors.go:**
```go
// internal/cmd/errors.go
var (
    ErrContactNameRequired = errors.New("--name is required")
    ErrContactSetRequired  = errors.New("at least one --set is required")
)
```

**Recommendation: Option B (centralized errors.go)**

Rationale:
- All cmd package sentinel errors in one place, easy to audit
- The `userFacingError()` mapping function will live in this file too
- Future PRs (deal.go, activity.go, etc.) will add their own sentinels — keeping them in one file prevents scatter
- Follows Go standard library pattern (`io.EOF`, `net.ErrClosed`)

### Sentinel naming convention

`Err<Resource><Issue>` — e.g., `ErrContactNameRequired`, `ErrDealTitleRequired`

---

## 3. userFacingError() Introduction

### Design

New file: `internal/cmd/error_handler.go`

```go
package cmd

import (
    "errors"
    "github.com/AgentPal/AgentCRM/internal/i18n"
)

// userFacingError translates sentinel errors to the user's language.
// Unrecognized errors pass through unchanged.
func userFacingError(err error) string {
    switch {
    case errors.Is(err, ErrContactNameRequired):
        return i18n.T("error.contact.name.required")
    case errors.Is(err, ErrContactSetRequired):
        return i18n.T("error.contact.set.required")
    case errors.Is(err, ErrContactFieldRequired):
        return i18n.T("error.contact.field.required")
    default:
        return err.Error()
    }
}
```

### Integration point

The current `Execute()` function in root.go line 33:
```go
func Execute() error {
    return rootCmd.Execute()
}
```

After migration:
```go
func Execute() error {
    err := rootCmd.Execute()
    if err != nil {
        fmt.Fprintf(os.Stderr, "%s\n", userFacingError(err))
        os.Exit(1)
    }
    return nil
}
```

Wait — this changes `Execute()` to print to stderr and exit instead of returning the error. Let me check how `Execute()` is called currently.

Actually, looking at root.go, `Execute()` is the main entry point called from `cmd/agentcrm/main.go`. The `RunE` handlers currently return errors and cobra prints them. But cobra prints the raw error text, not translated. The cleanest approach:

**Option 1 — Wrap in Execute():** Have Execute() print the translated error and exit. This matches how most CLI tools work.

**Option 2 — Intercept at main.go:** Have main.go call userFacingError before printing.

Let me check what cmd/agentcrm/main.go looks like.

Actually, cobra's default behavior is: if RunE returns an error, cobra prints it and exits with code 1. The translation needs to happen before cobra prints it. 

The cleanest approach: override the SilenceErrors and SilenceUsage on rootCmd, then handle translation in Execute().

### Plan for error_handler.go

I'll structure it with:
1. Sentinel error variables
2. userFacingError() translation function
3. Integration in Execute() via SilenceErrors + custom error output

---

## 4. JSON Key Translation Draft

### 4.1 Already exist from PR 4 (reuse, no change needed)

```json
"cmd.contact.short": "Manage contacts" / "管理联系人"
"cmd.contact.upsert.short": "Create or update contact (deduped by email)" / "创建或更新联系人（按 email 去重）"
"output.contact.created": "Created contact: %s (%s)" / "已创建联系人: %s (%s)"
"output.contact.updated": "Updated contact: %s (%s)" / "已更新联系人: %s (%s)"
"output.contact.list.none": "No contacts" / "无联系人"
"output.root.init.success": "AgentCRM initialized: %s" / "AgentCRM 已初始化: %s"
"error.contact.name.required": "--name is required" / "--name 是必需的"
"error.contact.field.required": "--field is required" / "--field 是必需的"
"flag.contact.name": "contact name (required)" / "联系人姓名（必需）"
"flag.contact.email": "email" / "邮箱"
"flag.contact.company": "company" / "公司"
```

### 4.2 New keys needed

en translations must follow §11 Style Guide (lowercase, no period, imperative, direct).

New keys needed:

#### cmd.* (net new: 9)

```json
"cmd.root.short": "Local customer memory system — shared brain for your AI agents",
"cmd.root.long": "AgentCRM is a local-first, file-based, zero-process customer relationship memory system. Distributed as a CLI tool, it gives every AI agent an immediate persistent, readable, backup-friendly customer memory. All data lives in ~/AgentCRM/. Files are the source of truth — open them in any editor.",
"cmd.root.version": "Show version information",
"cmd.root.init": "Initialize AgentCRM data directory",
"cmd.root.reindex": "Rebuild SQLite index from files",

"cmd.contact.get.short": "Show contact details",
"cmd.contact.search.short": "Search contacts",
"cmd.contact.list.short": "List contacts",
"cmd.contact.update.short": "Update contact fields (history auto-archived)",
"cmd.contact.history.short": "View contact field history",
"cmd.contact.merge.short": "Merge two contacts",
"cmd.contact.merge.long": "Merge merge-id into keeper-id. The merge-id file is renamed to .merged-into-<keeper-id>.md.",
```

#### flag.* (net new: 16, minus 3 existing = 13)

```json
// Already exist:
"flag.contact.name": "contact name (required)",
"flag.contact.email": "email",
"flag.contact.company": "company",

// New:
"flag.data_dir": "data directory (default ~/AgentCRM, overridden by AGENTCRM_HOME)",
"flag.format": "output format: text or json",
"flag.actor": "actor name (default AGENTCRM_ACTOR env or unknown)",

"flag.contact.title": "job title",
"flag.contact.phone": "phone",
"flag.contact.source": "source",
"flag.contact.tag": "tags",
"flag.contact.as_of": "view data as of this time",
"flag.contact.limit": "max results",
"flag.contact.tag_filter": "filter by tag",
"flag.contact.updated_since": "filter by last updated",
"flag.contact.has_field": "filter by field existence (birthday)",
"flag.contact.set": "set field (field=value)",
"flag.contact.reason": "change reason",
"flag.contact.field": "field name",
"flag.contact.merge_reason": "merge reason",
```

#### error.* (net new: 1)

```json
"error.contact.set.required": "at least one --set is required",
```

#### output.* (net new: 17, minus 4 existing = 13)

```json
// Already exist:
"output.contact.created": "Created contact: %s (%s)",
"output.contact.updated": "Updated contact: %s (%s)",
"output.contact.list.none": "No contacts",
"output.root.init.success": "AgentCRM initialized: %s",

// New:
"output.root.version.text": "Local customer memory system — files are the truth",

"output.contact.upsert.success": "%s contact: %s (%s)",
"output.contact.upsert.company": "  Company: %s",
"output.contact.upsert.title": "  Title: %s",
"output.contact.get.as_of": "[as of: %s]",
"output.contact.search.none": "No matching contacts",
"output.contact.search.recent": " [last: %s]",
"output.contact.list.count": "\n%d contacts total\n",
"output.contact.update.same": "%s is already set to that value",
"output.contact.update.changed": "Updated %s: %s → %s",
"output.contact.update.set": "Set %s = %s",
"output.contact.history.none": "No history",
"output.contact.history.to": "present",
"output.contact.merge.success": "Merged: %s (%s) ← %s (%s)",
"output.contact.merge.reason": "Reason: %s",
```

#### zh translations for new keys

```json
"cmd.root.short": "本地客户记忆系统 - 给你的 AI Agent 一个共享的客户大脑",
"cmd.root.long": "AgentCRM 是一个本地化、文件存储、零服务进程的客户领域记忆系统。它以 CLI 工具的形式分发，让 AI Agent 拥有一个长期、可读、可备份的客户记忆。所有数据存储在 ~/AgentCRM/ 目录，文件即真相，可用任何编辑器打开。",
"cmd.root.version": "显示版本信息",
"cmd.root.init": "初始化 AgentCRM 数据目录",
"cmd.root.reindex": "从文件重建 SQLite 索引",
"cmd.contact.get.short": "获取联系人详情",
"cmd.contact.search.short": "搜索联系人",
"cmd.contact.list.short": "列出联系人",
"cmd.contact.update.short": "更新联系人字段（自动归档旧值到 _history）",
"cmd.contact.history.short": "查看联系人字段历史",
"cmd.contact.merge.short": "合并两个联系人",
"cmd.contact.merge.long": "将 merge-id 合并到 keeper-id。merge-id 的文件会被重命名为 .merged-into-<keeper-id>.md。",

"flag.data_dir": "数据目录（默认 ~/AgentCRM，可被 AGENTCRM_HOME 覆盖）",
"flag.format": "输出格式: text 或 json",
"flag.actor": "执行者名称（默认 AGENTCRM_ACTOR 环境变量或 unknown）",
"flag.contact.title": "职位",
"flag.contact.phone": "电话",
"flag.contact.source": "来源",
"flag.contact.tag": "标签",
"flag.contact.as_of": "查看指定时间点的数据",
"flag.contact.limit": "返回数量上限",
"flag.contact.tag_filter": "按标签过滤",
"flag.contact.updated_since": "按更新日期过滤",
"flag.contact.has_field": "按存在字段过滤 (birthday)",
"flag.contact.set": "设置字段 (field=value)",
"flag.contact.reason": "变更原因",
"flag.contact.field": "字段名",
"flag.contact.merge_reason": "合并原因",

"error.contact.set.required": "至少需要一个 --set 参数",

"output.root.version.text": "本地客户记忆系统 - 文件即真相",
"output.contact.upsert.success": "%s 联系人: %s (%s)",
"output.contact.upsert.company": "  公司: %s",
"output.contact.upsert.title": "  职位: %s",
"output.contact.get.as_of": "[时间点: %s]",
"output.contact.search.none": "未找到匹配的联系人",
"output.contact.search.recent": " [最近: %s]",
"output.contact.list.count": "\n共 %d 个联系人\n",
"output.contact.update.same": "%s 已经是该值",
"output.contact.update.changed": "已更新 %s: %s → %s",
"output.contact.update.set": "已设置 %s = %s",
"output.contact.history.none": "无历史记录",
"output.contact.history.to": "至今",
"output.contact.merge.success": "已合并: %s (%s) ← %s (%s)",
"output.contact.merge.reason": "原因: %s",
```

### 4.3 Keys with layout issues (see §9 of design)

These embed leading spaces/colons — marked for future cleanup but accepted for v1.0:

```json
"output.contact.upsert.company": "  Company: %s",
"output.contact.upsert.title": "  Title: %s",
"output.contact.get.as_of": "[as of: %s]",
"output.contact.search.recent": " [last: %s]",
"output.contact.list.count": "\n%d contacts total\n",
"output.contact.merge.success": "Merged: %s (%s) ← %s (%s)",
```

---

## 5. Test Impact Assessment

### Assertions that check Chinese output

| File | Line | Current | Will change to | Action |
|---|---|---|---|---|
| `cmd_test.go` | 47 | `strings.Contains(out, "已初始化")` | `strings.Contains(out, "AgentCRM initialized")` | Update assertion |
| `cmd_test.go` | 201 | `strings.Contains(out, "已合并")` | `strings.Contains(out, "Merged:")` | Update assertion |

### Assertions NOT affected (scope = root.go + contact.go only)

`cmd_test.go:554` checks `"已推进"` — that's in events.go, not in this PR.
`cmd_test.go:95,136` check `"已导入"` — that's in io.go, not in this PR.
`cmd_more_test.go:265` checks `"已重置"` — that's in events.go, not in this PR.

### Fixture data (names like "张三")

These stay Chinese — they're test data, not UI strings. No change needed.

### Test implications of i18n.T() in cmd package

Test `executeCommand()` captures stdout. After migration, output text will be English (since MustInit defaults to "en"). The test assertions need to match English:
- "AgentCRM initialized: <dir>" instead of "AgentCRM 已初始化: <dir>"
- "Merged: ..." instead of "已合并: ..."

### New test needed

Consider adding one i18n integration test that sets `AGENTCRM_LANG=zh` and verifies Chinese output for a root command + a contact command. This confirms the full pipeline works end-to-end. This can be in `cmd_test.go` or a new `cmd_i18n_test.go`.

---

## 6. Commit Splitting Plan

### Commit 1 — `feat(cmd): add sentinel errors and userFacingError handler`

- New file: `internal/cmd/errors.go`
  - Sentinel error variables extracted from contact.go
- New file: `internal/cmd/error_handler.go`
  - `userFacingError()` function with switch on sentinels
- Modified: `internal/cmd/root.go`
  - Update `Execute()` to use `userFacingError()` for error display
  - Add `SilenceErrors: true` to rootCmd
- Modified: `internal/cmd/contact.go`
  - Replace `fmt.Errorf("--name 是必需的")` with `ErrContactNameRequired`
  - Replace `fmt.Errorf("至少需要一个 --set 参数")` with `ErrContactSetRequired`
  - Replace `fmt.Errorf("--field 是必需的")` with `ErrContactFieldRequired`

### Commit 2 — `feat(i18n): add ~45 translation keys for root + contact commands`

- Modified: `internal/i18n/messages_en.json`
  - Add 45 new keys with English translations
- Modified: `internal/i18n/messages_zh.json`
  - Add 45 new keys with Chinese translations

### Commit 3 — `refactor(cmd): migrate root.go + contact.go to i18n.T()`

- Modified: `internal/cmd/root.go`
  - Replace all Short/Long/flag descriptions with i18n.T()
  - Replace fmt.Printf output messages with i18n.T()
- Modified: `internal/cmd/contact.go`
  - Replace all Short/Long/flag descriptions with i18n.T()
  - Replace fmt.Printf output messages with i18n.T()

### Commit 4 — `test(cmd): fix test assertions for English output`

- Modified: `internal/cmd/cmd_test.go`
  - Line 47: `"已初始化"` → `"AgentCRM initialized"`
  - Line 201: `"已合并"` → `"Merged:"`

### Why this split

- **Commit 1** (sentinels + error_handler) — no translation needed, pure refactoring. Can be reviewed for correctness independently. No behavior change.
- **Commit 2** (JSON keys) — pure data addition. Testable via `TestJSONIntegrity` + `TestNoForbiddenTerms`.
- **Commit 3** (T() migration) — the actual behavior change. All output now goes through i18n. Each file's changes are atomic.
- **Commit 4** (tests) — update assertions to match new English output. Separate from logic changes so reviewers can focus on one thing at a time.

---

## 7. Implementation Checklist

- [ ] Create `internal/cmd/errors.go` with sentinel errors
- [ ] Create `internal/cmd/error_handler.go` with `userFacingError()`
- [ ] Update contact.go RunE to use sentinels instead of fmt.Errorf
- [ ] Update root.go Execute() to use SilenceErrors + userFacingError
- [ ] Verify: `go build ./...` compiles
- [ ] Add 45 new keys to messages_en.json + messages_zh.json
- [ ] Verify: `TestJSONIntegrity` passes
- [ ] Verify: `TestNoForbiddenTerms` passes
- [ ] Migrate all Short/Long/flag descriptions in root.go → i18n.T()
- [ ] Migrate all fmt.Printf output in root.go → i18n.T()
- [ ] Migrate all Short/Long/flag descriptions in contact.go → i18n.T()
- [ ] Migrate all fmt.Printf output in contact.go → i18n.T()
- [ ] Update cmd_test.go assertions
- [ ] Run `go test ./internal/i18n/...` — all pass
- [ ] Run `go test ./internal/cmd/...` — all pass
- [ ] Run `go test ./...` — no regressions
- [ ] Push + create PR
