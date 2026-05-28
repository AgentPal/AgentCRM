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
| `output.contact.*` | 17 | created, updated, upsert.created, upsert.updated, upsert.company, upsert.title, get.as_of, search.none, search.recent, list.none, list.count, update.same, update.changed, update.set, history.none, history.to, merge.success, merge.reason |
| Total | **46** | |

### Grand total: 56 keys

- Already in PR 4 sample set: ~10 keys (cmd.contact.short, cmd.contact.upsert.short, output.contact.created, output.contact.updated, output.contact.list.none, output.root.init.success, error.contact.name.required, error.contact.field.required, flag.contact.name, flag.contact.email)
- **Net new keys needed: ~46**
- The estimate in §12 (~70) was for root + contact + deal — deal.go is not in this PR.

**Note:** `output.contact.upsert.created` and `output.contact.upsert.updated` replace the original single `output.contact.upsert.success` key (split per review — see §4.2 for rationale).

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
    "fmt"
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

### Integration: Updated Execute() in root.go

**Current state:**
- `rootCmd` has no `SilenceErrors` or `SilenceUsage` set (defaults to false)
- `Execute()` simply calls `rootCmd.Execute()` and returns any error
- `cmd/agentcrm/main.go` receives the error and prints `"Error: %v\n"` to stderr, then exits 1

**Problem:** cobra's default behavior prints errors itself (to stderr) AND returns them. This creates double output when main.go also prints. With i18n, the "Error:" English prefix from main.go would mingle with translated messages.

**After migration:**

`root.go` — updated rootCmd and Execute():
```go
var rootCmd = &cobra.Command{
    Use:   "agentcrm",
    Short: i18n.T("cmd.root.short"),
    Long:  i18n.T("cmd.root.long"),
    SilenceErrors: true,   // cobra won't print errors — we handle them
    SilenceUsage:  true,   // don't print usage on every error (avoids noise)
    PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
        return nil
    },
}

// Execute is the main entry point. It runs the root command and handles error
// display with i18n translation. Errors are printed to stderr; output to stdout.
func Execute() {
    if err := rootCmd.Execute(); err != nil {
        fmt.Fprintln(os.Stderr, userFacingError(err))
        os.Exit(1)
    }
}
```

**Key decisions:**
1. **SilenceErrors: true** — cobra won't print errors, our `Execute()` owns the error path
2. **SilenceUsage: true** — prevents cobra from printing full usage on errors (e.g., wrong args). User can still run `--help` explicitly. Reduces noise.
3. **Error output → stderr** — `userFacingError()` output goes to `os.Stderr` via `fmt.Fprintln`
4. **Normal output → stdout** — all `output.*` keys printed via existing `fmt.Printf`/`fmt.Println` calls. No change to stdout behavior.
5. **Exit code: 1 for all errors** — simplified. Graded exit codes (validation=2 vs system=1) deferred to post-v1.0 if needed.
6. **Execute() signature changes from `func Execute() error` to `func Execute()`** — no return value, it always exits or succeeds. The error is handled internally.

`cmd/agentcrm/main.go` — simplified:
```go
func main() {
    cmd.Execute()
}
```

### stderr vs stdout boundary

| Category | Output stream | Handling |
|---|---|---|
| `output.*` keys (success messages) | stdout | `fmt.Printf`/`fmt.Println` — no change |
| Error messages (sentinel errors) | stderr | `userFacingError()` via `fmt.Fprintln(os.Stderr, ...)` |
| JSON output (`format == "json"`) | stdout | `fmt.Println(string(out))` — no change |
| cobra usage/help | stderr | cobra's default (not silenced) |

### Exit code strategy

All errors → exit code 1. Rationale:
- CLI is consumed by AI agents, not shell scripts. Exit code differentiation adds complexity with little benefit.
- If graded exit codes become needed (e.g., validation vs system errors), introduce `type ExitCode int` sentinel interface post-v1.0.
- Current main.go already exits 1 on all errors — behavior preserved.

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
"flag.contact.tag": "tag (repeatable)",        // StringSlice on upsert
"flag.contact.as_of": "view data as of this time",
"flag.contact.limit": "max results",
"flag.contact.tag_filter": "filter by tag",     // String on list
"flag.contact.updated_since": "filter by last updated",
"flag.contact.has_field": "filter by field existence (birthday)",
"flag.contact.set": "set field (field=value)",
"flag.contact.reason": "change reason",
"flag.contact.field": "field name",
"flag.contact.merge_reason": "merge reason",
```

Note: `flag.contact.tag` is used by `contactUpsertCmd` with `StringSlice` (repeatable). The separate `flag.contact.tag_filter` covers `contactListCmd`'s single-value `--tag` flag.

#### error.* (net new: 1)

```json
"error.contact.set.required": "at least one --set is required",
```

#### output.* (net new: 17, minus 4 existing = 13, plus split = 14)

`output.contact.upsert.success` is split into two keys (see §4.3 rationale):

```json
// Already exist:
"output.contact.created": "Created contact: %s (%s)",
"output.contact.updated": "Updated contact: %s (%s)",
"output.contact.list.none": "No contacts",
"output.root.init.success": "AgentCRM initialized: %s",

// New:
"output.root.version.text": "Local-first customer memory · No servers, no lock-in",

"output.contact.upsert.created": "Created contact: %s (%s)",
"output.contact.upsert.updated": "Updated contact: %s (%s)",
"output.contact.upsert.company": "  Company: %s",
"output.contact.upsert.title": "  Title: %s",
"output.contact.get.as_of": "[as of: %s]",
"output.contact.search.none": "No matching contacts",
"output.contact.search.recent": " [last: %s]",
"output.contact.list.count": "\n%d contacts total\n",
"output.contact.update.same": "%s: no change",
"output.contact.update.changed": "%s: %s → %s",
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
"flag.contact.tag": "标签（可重复）",
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

"output.root.version.text": "本地优先的客户记忆 · 无服务、无锁定",
"output.contact.upsert.created": "已创建联系人: %s (%s)",
"output.contact.upsert.updated": "已更新联系人: %s (%s)",
"output.contact.upsert.company": "  公司: %s",
"output.contact.upsert.title": "  职位: %s",
"output.contact.get.as_of": "[时间点: %s]",
"output.contact.search.none": "未找到匹配的联系人",
"output.contact.search.recent": " [最近: %s]",
"output.contact.list.count": "\n共 %d 个联系人\n",
"output.contact.update.same": "%s: 无变化",
"output.contact.update.changed": "%s: %s → %s",
"output.contact.update.set": "已设置 %s = %s",
"output.contact.history.none": "无历史记录",
"output.contact.history.to": "至今",
"output.contact.merge.success": "已合并: %s (%s) ← %s (%s)",
"output.contact.merge.reason": "原因: %s",
```

### 4.3 Key splits and changes (per review)

**Split — output.contact.upsert.success → created/updated:**
- Rationale: the original `"%s contact: %s (%s)"` used a verb placeholder (`%s`) with `"Created"`/`"Updated"` passed from code. This is unsafe because:
  - English sentence structure differs per verb
  - The format string can't be properly translated (verb position varies by language)
- Solution: two separate keys, code dispatches based on `created` boolean

**Change — output.contact.update.changed:**
- Before: `"Updated %s: %s → %s"` — "Updated company" is awkward English
- After: `"%s: %s → %s"` — the arrow conveys change, verb omitted
- zh matching: `"%s: %s → %s"` — same arrow format, consistent across languages

**Change — output.contact.update.same:**
- Before: `"%s is already set to that value"` — too administrative
- After: `"%s: no change"` / zh: `"%s: 无变化"` — concise

**Change — output.root.version.text:**
- Before: `"...files are the truth"` — literal translation, awkward in English
- After: `"Local-first customer memory · No servers, no lock-in"` — matches README tagline, brand-consistent
- zh: `"本地优先的客户记忆 · 无服务、无锁定"`

### 4.4 Keys with layout issues (see §9 of design)

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

### Full Chinese character scan of cmd test files

Ran `grep -n "[一-鿿]" internal/cmd/*_test.go` across both test files:

#### Assertions that check Chinese output (WILL CHANGE in this PR)

| File | Line | Current | Will change to |
|---|---|---|---|
| `cmd_test.go` | 47 | `strings.Contains(out, "已初始化")` | `strings.Contains(out, "AgentCRM initialized")` |
| `cmd_test.go` | 201 | `strings.Contains(out, "已合并")` | `strings.Contains(out, "Merged:")` |

**Total: 2 assertions across the entire cmd test suite.**

#### Assertions NOT affected (NOT in this PR scope)

| File | Line | Current | Located in | Action |
|---|---|---|---|---|
| `cmd_test.go` | 554 | `"已推进"` | events.go (deal stage change) | PR 8 |
| `cmd_more_test.go` | 95 | `"已导入"` | io.go (import) | PR 9 |
| `cmd_more_test.go` | 136 | `"已导入"` | io.go (import CSV) | PR 9 |
| `cmd_more_test.go` | 265 | `"已重置"` | events.go (subscribers reset) | PR 8 |

#### Chinese characters that stay as-is (test data, not UI strings)

All of these are fixture data (names, titles, reasons, CSV content) or code comments:

| File | Line(s) | Content | Reason |
|---|---|---|---|
| `cmd_test.go` | 82, 119, 120 | "张三" | Test contact name |
| `cmd_test.go` | 147 | "公司更名" | --reason arg value (data) |
| `cmd_test.go` | 183 | "李四" | Test contact name |
| `cmd_test.go` | 197 | "重复联系人" | --reason arg value (data) |
| `cmd_test.go` | 217 | "王五" | Test contact name |
| `cmd_test.go` | 257, 258 | "企业版订阅" | Test deal title |
| `cmd_test.go` | 339 | "测试联系人" | Test contact name |
| `cmd_test.go` | 439 | "记忆测试" | Test memory name |
| `cmd_test.go` | 520 | "事件测试" | Test event name |
| `cmd_test.go` | 571 | "导出测试" | Test export name |
| `cmd_test.go` | 606 | "导出测试" | Test export name |
| `cmd_test.go` | 632 | "测试" | Test contact name |
| `cmd_more_test.go` | 23 | "CSV导出测试" | Test name |
| `cmd_more_test.go` | 60 | "CSV导出测试" | Test name |
| `cmd_more_test.go` | 77 | "FN:张三" | vCard fixture |
| `cmd_more_test.go` | 124 | "李四", "王五" | CSV fixture data |
| `cmd_more_test.go` | 203 | "导出测试含商机" | Test name |
| `cmd_more_test.go` | 214 | "测试商机" | Test deal title |
| `cmd_more_test.go` | 241 | "测试商机" | Test deal title |

**Conclusion: Only 2 assertions in the entire cmd test package are affected by this PR.** All other Chinese characters are fixture data (names, titles, CSV content, vCard data, --reason arguments) or belong to commands outside this PR's scope.

### Test implications of i18n.T() in cmd package

Test `executeCommand()` captures stdout. After migration, output text will be English (since MustInit defaults to "en"):
- "AgentCRM initialized: <dir>" instead of "AgentCRM 已初始化: <dir>"
- "Merged: ..." instead of "已合并: ..."

### New test file: cmd_i18n_test.go (HARD REQUIREMENT for this PR)

Add two integration tests to verify the i18n pipeline end-to-end through the CLI:

```go
package cmd

import (
    "strings"
    "testing"

    "github.com/AgentPal/AgentCRM/internal/i18n"
)

func TestI18N_ChineseOutput(t *testing.T) {
    i18n.SetLang("zh")

    // Test root command (init) produces Chinese output
    out, err := executeCommand(initCmd, "--data-dir", t.TempDir())
    if err != nil {
        t.Fatal(err)
    }
    if !strings.Contains(out, "已初始化") {
        t.Errorf("expected Chinese init output, got: %s", out)
    }

    // Test contact command (list) produces Chinese output
    out, err = executeCommand(contactListCmd, "--data-dir", t.TempDir())
    if err != nil {
        t.Fatal(err)
    }
    if !strings.Contains(out, "无联系人") {
        t.Errorf("expected Chinese contact list output, got: %s", out)
    }
}

func TestI18N_EnglishDefault(t *testing.T) {
    // Default lang is "en" (no SetLang call)
    // TestNoForbiddenTerms already covers glossary compliance

    out, err := executeCommand(initCmd, "--data-dir", t.TempDir())
    if err != nil {
        t.Fatal(err)
    }
    if !strings.Contains(out, "AgentCRM initialized") {
        t.Errorf("expected English init output, got: %s", out)
    }

    out, err = executeCommand(contactListCmd, "--data-dir", t.TempDir())
    if err != nil {
        t.Fatal(err)
    }
    if !strings.Contains(out, "No contacts") {
        t.Errorf("expected English contact list output, got: %s", out)
    }
}
```

Note: These tests share the `executeCommand()` helper from `cmd_test.go` (same package). They don't duplicate TestNoForbiddenTerms coverage — `TestI18N_EnglishDefault` just verifies the default is English.

---

## 6. Commit Splitting Plan

### Commit 1 — `feat(cmd): add sentinel errors and userFacingError handler`

- New file: `internal/cmd/errors.go`
  - Sentinel error variables extracted from contact.go:
    - `ErrContactNameRequired`
    - `ErrContactSetRequired`
    - `ErrContactFieldRequired`
- New file: `internal/cmd/error_handler.go`
  - `userFacingError()` function with switch on sentinels
  - Calls `i18n.T()` for known sentinels, falls back to `err.Error()` for unknown
- Modified: `internal/cmd/root.go`
  - Update `Execute()`: change signature to `func Execute()`, add `SilenceErrors: true` and `SilenceUsage: true` to rootCmd, print translated error to stderr, exit 1
  - Import `"fmt"` and `"os"` (already imported)
- Modified: `internal/cmd/contact.go`
  - Replace `fmt.Errorf("--name 是必需的")` with `ErrContactNameRequired`
  - Replace `fmt.Errorf("至少需要一个 --set 参数")` with `ErrContactSetRequired`
  - Replace `fmt.Errorf("--field 是必需的")` with `ErrContactFieldRequired`
- Modified: `cmd/agentcrm/main.go`
  - Simplify: `Execute()` no longer returns error, remove `if err` check
  - New: `func main() { cmd.Execute() }`
- Verify: `go build ./...` compiles

**No translation logic in this commit. Pure refactoring. No behavior change (errors still Chinese).**

### Commit 2 — `feat(i18n): add ~46 translation keys for root + contact commands`

- Modified: `internal/i18n/messages_en.json`
  - Add ~46 new keys with English translations
- Modified: `internal/i18n/messages_zh.json`
  - Add ~46 new keys with Chinese translations
- Verify: `TestJSONIntegrity` passes (en/zh key parity)
- Verify: `TestNoForbiddenTerms` passes (glossary compliance)

**Pure data addition. No code changes to cmd/ package.**

### Commit 3 — `refactor(cmd): migrate root.go + contact.go to i18n.T()`

- Modified: `internal/cmd/root.go`
  - Replace Short/Long descriptions: `i18n.T("cmd.root.short")`, `i18n.T("cmd.root.long")`
  - Replace flag descriptions: `i18n.T("flag.data_dir")`, `i18n.T("flag.format")`, `i18n.T("flag.actor")`
  - Replace output messages in Run/RunE handlers:
    - versionCmd: `fmt.Printf("AgentCRM %s\n", appVersion)` stays (version number not i18n), `fmt.Println("本地客户记忆系统...")` → `i18n.T("output.root.version.text")`
    - initCmd: `fmt.Printf(...)` → `i18n.T("output.root.init.success", s.ConfigDir())`
    - reindexCmd: no output messages, stays as-is
  - Import `"github.com/AgentPal/AgentCRM/internal/i18n"`
- Modified: `internal/cmd/contact.go`
  - Replace Short/Long descriptions: `i18n.T("cmd.contact.*")`
  - Replace flag descriptions: `i18n.T("flag.contact.*")`
  - Replace output messages in RunE handlers:
    - `contactUpsertCmd`: split upsert.success into upsert.created/upsert.updated based on `created` boolean
    - `contactGetCmd`: `fmt.Printf("[时间点: %s]\n")` → `i18n.T("output.contact.get.as_of", ...)`
    - `contactSearchCmd`: `fmt.Println("未找到匹配的联系人")` → `i18n.T("output.contact.search.none")`, `fmt.Printf(" [最近: %s]")` → `i18n.T("output.contact.search.recent", ...)`
    - `contactListCmd`: `fmt.Println("无联系人")` → `i18n.T("output.contact.list.none")`, `fmt.Printf("\n共 %d 个联系人\n")` → `i18n.T("output.contact.list.count", len(results))`
    - `contactUpdateCmd`: `fmt.Printf("%s 已经是该值")` → `i18n.T("output.contact.update.same", field)`, `fmt.Printf("已更新 %s: %s → %s")` → `i18n.T("output.contact.update.changed", field, oldVal, value)`, `fmt.Printf("已设置 %s = %s")` → `i18n.T("output.contact.update.set", field, value)`
    - `contactHistoryCmd`: `fmt.Println("无历史记录")` → `i18n.T("output.contact.history.none")`, `to = "至今"` → `to = i18n.T("output.contact.history.to")`
    - `contactMergeCmd`: `fmt.Printf("已合并: ...")` → `i18n.T("output.contact.merge.success", ...)`, `fmt.Printf("原因: %s\n")` → `i18n.T("output.contact.merge.reason", reason)`
  - Import `"github.com/AgentPal/AgentCRM/internal/i18n"`
- Verify: `go build ./...` compiles

**The behavior-change commit. All output now flows through i18n.**

### Commit 4 — `test(cmd): fix test assertions + add i18n integration tests`

- Modified: `internal/cmd/cmd_test.go`
  - Line 47: `"已初始化"` → `"AgentCRM initialized"`
  - Line 201: `"已合并"` → `"Merged:"`
- New file: `internal/cmd/cmd_i18n_test.go`
  - `TestI18N_ChineseOutput`: SetLang("zh"), run init + contact list, verify Chinese output
  - `TestI18N_EnglishDefault`: default en, run init + contact list, verify English output
- Run: `go test ./internal/cmd/...` — all pass
- Run: `go test ./...` — no regressions

### Why this split

- **Commit 1** (sentinels + error_handler + Execute change) — no translation needed, pure Go refactoring. Can be reviewed for correctness independently. No behavior change.
- **Commit 2** (JSON keys) — pure data addition. Testable via `TestJSONIntegrity` + `TestNoForbiddenTerms`.
- **Commit 3** (T() migration) — the actual behavior change. All output flows through i18n. Each file's changes are atomic.
- **Commit 4** (tests) — update assertions for English output + add i18n integration tests. Separate from logic changes so reviewers can focus.

---

## 7. Implementation Checklist

- [ ] Create `internal/cmd/errors.go` with sentinel errors
- [ ] Create `internal/cmd/error_handler.go` with `userFacingError()`
- [ ] Update contact.go RunE to use sentinels instead of fmt.Errorf
- [ ] Update root.go Execute() to use SilenceErrors + SilenceUsage + userFacingError
- [ ] Update cmd/agentcrm/main.go for new Execute() signature
- [ ] Verify: `go build ./...` compiles
- [ ] Add ~46 new keys to messages_en.json + messages_zh.json
- [ ] Verify: `TestJSONIntegrity` passes
- [ ] Verify: `TestNoForbiddenTerms` passes
- [ ] Migrate all Short/Long/flag descriptions in root.go → i18n.T()
- [ ] Migrate all fmt.Printf output in root.go → i18n.T()
- [ ] Migrate all Short/Long/flag descriptions in contact.go → i18n.T()
- [ ] Migrate all fmt.Printf output in contact.go → i18n.T()
- [ ] Update cmd_test.go assertions (2 lines)
- [ ] Create cmd_i18n_test.go with TestI18N_ChineseOutput + TestI18N_EnglishDefault
- [ ] Run `go test ./internal/i18n/...` — all pass
- [ ] Run `go test ./internal/cmd/...` — all pass
- [ ] Run `go test ./...` — no regressions
- [ ] Push + create PR
