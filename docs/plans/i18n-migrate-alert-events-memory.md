# Plan: i18n Migration — alert.go + events.go + memory.go (PR 8)

## 1. Scope Confirmation

### alert.go — keys needed

| Area | Total | Already exist | Net new |
|---|---|---|---|
| `cmd.alert.*` | 9 | 1 (`cmd.alert.short`) | 8 |
| `cmd.alert.rule.*` | 4 | 0 | 4 |
| `flag.alert.*` | 5 | 1 (`flag.alert.days`) | 4 |
| `error.alert.*` | 2 | 0 | 2 |
| `output.alert.*` | 14 | 2 (`output.alert.scan.count`, `output.alert.scan.none`) | 12 |
| Total | **34** | **4** | **30** |

### events.go — keys needed

| Area | Total | Already exist | Net new |
|---|---|---|---|
| `cmd.events.*` | 6 | 1 (`cmd.events.short`) | 5 |
| `flag.events.*` | 6 | 1 (`flag.events.as`) | 5 |
| `error.events.*` | 0 | 0 | 0 |
| `output.events.*` | 6 | 1 (`output.events.poll.none`) | 5 |
| Total | **18** | **3** | **15** |

### memory.go — keys needed

| Area | Total | Already exist | Net new |
|---|---|---|---|
| `cmd.memory.*` | 8 | 1 (`cmd.memory.short`) | 7 |
| `flag.memory.*` | 12 | 1 (`flag.memory.scope`) + 1 reusable (`flag.deal.as_of`) | 10 |
| `error.memory.*` | 5 | 1 (`error.memory.scope.format`) | 4 |
| `output.memory.*` | 11 | 1 (`output.memory.written`) | 10 |
| Total | **36** | **4** | **31** |

### Grand total: 88 keys, 76 net new

- alert.go: 30 net new
- events.go: 15 net new
- memory.go: 31 net new
- Design doc §12 estimate: ~110. The difference is that ~10-12 keys already exist, plus more precise counting vs the design doc's rough estimate.

### PR boundary confirmation

All 76 keys are cmd-layer only (CLI labels/output/errors).

- `alert/engine.go` built-in rule titles/suggestions are **excluded** (for PR 9)
- `events/` internal store code is **excluded**
- `memory/` internal store code is **excluded**

No overlap with engine-layer content.

---

## 2. Sentinel Error Extraction Plan

### New sentinels (memory.go only — ~4)

| Line | Current | Proposed sentinel | Key |
|---|---|---|---|
| 28 | `"--scope 和 --text 是必需的"` | `ErrMemoryScopeTextRequired` | `error.memory.scope_text.required` |
| 71 | `"--scope 是必需的"` | `ErrMemoryScopeRequired` | `error.memory.scope.required` |
| 205 | `"--scope 和 --statement 是必需的"` | `ErrMemoryScopeStatementRequired` | `error.memory.scope_statement.required` |
| 248 | `"--proposal-id 和 --action 是必需的"` | `ErrMemoryProposalActionRequired` | `error.memory.proposal_action.required` |

### Direct i18n.T errors (no sentinel)

| File:Line | Current | Key |
|---|---|---|
| `alert.go:164,205` | `"提醒未找到: %s"` | `error.alert.not_found` |
| `alert.go:280` | `"规则文件为空"` | `error.alert.rule.empty` |
| `memory.go:33` | `"无效 scope 格式，应如 contact:<id>"` | `error.memory.scope.format` (key exists) |

These use `fmt.Errorf(i18n.T("key"), args...)` directly — userFacingError default passes through unchanged.

### Append to `internal/cmd/errors.go`

```go
// Memory (PR 8)
ErrMemoryScopeTextRequired      = errors.New("--scope and --text are required")
ErrMemoryScopeRequired          = errors.New("--scope is required")
ErrMemoryScopeStatementRequired = errors.New("--scope and --statement are required")
ErrMemoryProposalActionRequired = errors.New("--proposal-id and --action are required")
```

---

## 3. userFacingError() Extension

Add to `error_handler.go` switch:

```go
// Memory (PR 8)
case errors.Is(err, ErrMemoryScopeTextRequired):
    return i18n.T("error.memory.scope_text.required")
case errors.Is(err, ErrMemoryScopeRequired):
    return i18n.T("error.memory.scope.required")
case errors.Is(err, ErrMemoryScopeStatementRequired):
    return i18n.T("error.memory.scope_statement.required")
case errors.Is(err, ErrMemoryProposalActionRequired):
    return i18n.T("error.memory.proposal_action.required")
```

No changes to `Execute()` or `rootCmd`.

---

## 4. JSON Key Translation Draft

### 4.1 alert.go — new cmd.* keys

| Key | en | zh |
|---|---|---|
| `cmd.alert.scan.short` | Scan and generate alerts | 扫描并生成提醒 |
| `cmd.alert.list.short` | List pending alerts | 列出当前提醒 |
| `cmd.alert.dismiss.short` | Dismiss an alert | 忽略一条提醒 |
| `cmd.alert.snooze.short` | Snooze an alert | 推迟提醒 |
| `cmd.alert.rule.short` | Manage alert rules | 管理提醒规则 |
| `cmd.alert.rule.list.short` | List rules | 列出规则 |
| `cmd.alert.rule.add.short` | Add a rule | 添加规则 |
| `cmd.alert.rule.disable.short` | Disable a rule | 禁用规则 |

### 4.2 alert.go — new flag.* keys

| Key | en | zh |
|---|---|---|
| `flag.alert.since_last_scan` | only data since last scan | 仅上次扫描后的新数据 |
| `flag.alert.permanent` | dismiss permanently | 永久忽略 |
| `flag.alert.reason` | dismiss reason | 忽略原因 |
| `flag.alert.from_yaml` | YAML file path | YAML 文件路径 |

### 4.3 alert.go — new error.* keys

| Key | en | zh |
|---|---|---|
| `error.alert.not_found` | alert not found: %s | 提醒未找到: %s |
| `error.alert.rule.empty` | rule file is empty | 规则文件为空 |

### 4.4 alert.go — new output.* keys

| Key | en | zh |
|---|---|---|
| `output.alert.scan.active` | %d pending alerts already exist — review them with `alert list` | 已有 %d 条待处理提醒——先用 `alert list` 查看 |
| `output.alert.scan.suggestion` |   Suggestion: %s |   建议: %s |
| `output.alert.scan.contact` |   Contact: %s |   联系人: %s |
| `output.alert.scan.deal` |   Deal: %s |   商机: %s |
| `output.alert.list.pending` | No pending alerts | 无待处理提醒 |
| `output.alert.list.count` | \n%d pending alerts total\n | \n共 %d 条待处理提醒\n |
| `output.alert.dismiss.ok` | Alert dismissed | 已忽略提醒 |
| `output.alert.snooze.ok` | Alert snoozed %d days | 已推迟提醒 %d 天 |
| `output.alert.rule.builtin` | Built-in rules: | 内置规则: |
| `output.alert.rule.custom` | \nCustom rules: | \n用户规则: |
| `output.alert.rule.added` | %d rule added | 已添加 %d 条规则 |
| `output.alert.rule.added.plural` | %d rules added | 已添加 %d 条规则 |
| `output.alert.rule.disabled` | Disabled rule: %s | 已禁用规则: %s |

### 4.5 events.go — new cmd.* keys

| Key | en | zh |
|---|---|---|
| `cmd.events.poll.short` | Poll for new events | 拉取新事件 |
| `cmd.events.ack.short` | Advance event cursor | 推进事件 cursor |
| `cmd.events.watch.short` | Watch events (blocking poll) | 持续监听新事件（阻塞式轮询） |
| `cmd.events.subscribers.short` | Manage subscribers | 管理订阅者 |
| `cmd.events.subscribers.list.short` | List subscribers | 列出所有订阅者 |
| `cmd.events.subscribers.reset.short` | Reset subscriber cursor | 重置订阅者 cursor |

### 4.6 events.go — new flag.* keys

| Key | en | zh |
|---|---|---|
| `flag.events.filter` | event filter | 事件过滤器 |
| `flag.events.limit` | max results | 返回数量上限 |
| `flag.events.include_self` | include own events | 包含自己产生的事件 |
| `flag.events.up_to_seq` | advance to this seq | 推进到此 seq |
| `flag.events.interval` | poll interval (seconds) | 轮询间隔（秒） |

### 4.7 events.go — new output.* keys

| Key | en | zh |
|---|---|---|
| `output.events.poll.count` | \n%d events total (cursor: %d)\n | \n共 %d 个事件 (cursor: %d)\n |
| `output.events.ack.ok` | Advanced %s cursor to #%d | 已推进 %s cursor 到 #%d |
| `output.events.watch.start` | Watching events as %s · interval %ds · cursor %d | 监听事件 · actor: %s · 间隔: %ds · cursor %d |
| `output.events.watch.stop` | Press Ctrl+C to stop | 按 Ctrl+C 停止 |
| `output.events.subscribers.none` | No subscribers | 无订阅者 |
| `output.events.subscribers.reset.ok` | Reset %s cursor | 已重置 %s cursor |

### 4.8 memory.go — new cmd.* keys

| Key | en | zh |
|---|---|---|
| `cmd.memory.write.short` | Write a memory | 写入记忆 |
| `cmd.memory.list.short` | List memories | 列出记忆 |
| `cmd.memory.recall.short` | Recall memories | 检索记忆 |
| `cmd.memory.forget.short` | Delete a memory | 删除记忆 |
| `cmd.memory.propose.short` | Propose a memory (with conflict detection) | 提议新的记忆（带矛盾检测） |
| `cmd.memory.commit.short` | Commit/resolve a proposal | 提交/裁决提议 |
| `cmd.memory.decay.short` | Scan and expire old memories | 扫描并标记过期记忆 |

### 4.9 memory.go — new flag.* keys

| Key | en | zh |
|---|---|---|
| `flag.memory.text` | memory text | 记忆内容 |
| `flag.memory.decay` | decay policy | 衰减策略 |
| `flag.memory.valid_from` | valid from time | 生效时间 |
| `flag.memory.top_k` | max results | 返回数量上限 |
| `flag.memory.statement` | statement text | 陈述内容 |
| `flag.memory.source_snippet` | source snippet | 来源原文片段 |
| `flag.memory.confidence` | confidence (0-1) | 置信度 (0-1) |
| `flag.memory.proposal_id` | proposal ID | 提议 ID |
| `flag.memory.action` | action: supersede\|keep-both\|reject | 动作: supersede\|keep-both\|reject |
| `flag.memory.dry_run` | preview only, no changes | 仅预览不执行 |

`flag.deal.as_of` already exists and is reused for memory recall's `--as-of`.

### 4.10 memory.go — new error.* keys

| Key | en | zh |
|---|---|---|
| `error.memory.scope_text.required` | --scope and --text are required | --scope 和 --text 是必需的 |
| `error.memory.scope.required` | --scope is required | --scope 是必需的 |
| `error.memory.scope_statement.required` | --scope and --statement are required | --scope 和 --statement 是必需的 |
| `error.memory.proposal_action.required` | --proposal-id and --action are required | --proposal-id 和 --action 是必需的 |

### 4.11 memory.go — new output.* keys

| Key | en | zh |
|---|---|---|
| `output.memory.list.none` | No memories | 无记忆 |
| `output.memory.list.expired` |  [expired] |  [过期] |
| `output.memory.recall.none` | No matching memories | 未找到匹配的记忆 |
| `output.memory.forget.ok` | Memory deleted | 已删除记忆 |
| `output.memory.propose.recorded` | Proposal recorded: %s (status: %s) | 提议已记录: %s (status: %s) |
| `output.memory.propose.conflicts` |   %d conflict(s) detected: |   检测到 %d 条冲突: |
| `output.memory.propose.suggested_action` |   Suggested action: %s |   建议动作: %s |
| `output.memory.commit.done` | Processed proposal %s (action: %s) | 已处理提议 %s (action: %s) |
| `output.memory.decay.preview` | %d memories would expire (dry-run, not marked) | 预计有 %d 条记忆会过期（仅预览，未实际标记） |
| `output.memory.decay.done` | %d memories expired | 已标记 %d 条过期记忆 |

---

## 5. Special Attention: memory propose/commit/decay Multi-Step Flow

These keys appear in sequence during Agent interaction (矛盾消解流程). Each key's position in the user-visible dialog is shown below.

### Flow 1: memory propose (no conflict)

```
Agent calls: agentcrm memory propose --scope contact:abc --statement "Prefers email"
→  output.memory.propose.recorded   ← "Proposal recorded: prop_001 (status: accepted)"
   (no conflict block → flow ends here, no further output)
```

### Flow 2: memory propose (with conflict)

```
Agent calls: agentcrm memory propose --scope contact:abc --statement "Opposes cold calls"
→  output.memory.propose.recorded   ← "Proposal recorded: prop_002 (status: conflict)"
→  output.memory.propose.conflicts  ← "  2 conflict(s) detected:"
→  [lists conflicting memos]
→  output.memory.propose.suggested_action  ← "  Suggested action: supersede"
```

### Flow 3: memory commit (resolve a conflict)

```
Agent calls: agentcrm memory commit --proposal-id prop_002 --action supersede
→  output.memory.commit.done        ← "Processed proposal prop_002 (action: supersede)"
```

### Flow 4: memory decay-scan

```
Agent calls: agentcrm memory decay-scan --dry-run
→  output.memory.decay.preview      ← "3 memories would expire (preview mode, not yet marked)"

Agent calls: agentcrm memory decay-scan  (without --dry-run)
→  output.memory.decay.done          ← "3 memories expired"
```

### Key definitions with full context

#### `output.memory.propose.recorded`
- **Context**: Always first output of `memory propose`. Shows the proposal was created.
- **Preceded by**: (nothing, first output)
- **Followed by**: `output.memory.propose.conflicts` (if status==conflict) or end
- **en**: `"Proposal recorded: %s (status: %s)"`
- **zh**: `"提议已记录: %s (status: %s)"`

#### `output.memory.propose.conflicts`
- **Context**: Only shown when propose detects conflicts (status==conflict). Lists how many.
- **Preceded by**: `output.memory.propose.recorded`
- **Followed by**: conflict list items (not translated) then `output.memory.propose.suggested_action`
- **en**: `"  %d conflict(s) detected:"`
- **zh**: `"  检测到 %d 条冲突:"`

#### `output.memory.propose.suggested_action`
- **Context**: Last line of conflict output. Shows the recommended action.
- **Preceded by**: `output.memory.propose.conflicts` + conflict list
- **Followed by**: (nothing, end of propose output)
- **en**: `"  Suggested action: %s"`
- **zh**: `"  建议动作: %s"`

#### `output.memory.commit.done`
- **Context**: Output of `memory commit` after successful resolution.
- **Preceded by**: (nothing, first and only output)
- **Followed by**: (nothing)
- **en**: `"Processed proposal %s (action: %s)"`
- **zh**: `"已处理提议 %s (action: %s)"`

#### `output.memory.decay.preview`
- **Context**: Output of `memory decay-scan --dry-run`.
- **Preceded by**: (nothing, first and only output)
- **Followed by**: (nothing)
- **en**: `"%d memories would expire (preview mode, not yet marked)"`
- **zh**: `"发现 %d 条将要过期的记忆（预览模式，未实际标记）"`

#### `output.memory.decay.done`
- **Context**: Output of `memory decay-scan` (without --dry-run).
- **Preceded by**: (nothing, first and only output)
- **Followed by**: (nothing)
- **en**: `"%d memories expired"`
- **zh**: `"已标记 %d 条过期记忆"`

---

## 6. Special Attention: events.go Subscriber Terminology

### Glossary addition (§10)

Add to glossary:

> `subscriber` → `订阅者` (do NOT use `user`/`customer`/`agent`)
>
> In the events system, a "subscriber" is an Agent (Claude/OpenClaw/etc.) that polls for events. The term describes a machine actor, not a human user.

### Affected keys

| Key | en | zh | Rationale |
|---|---|---|---|
| `cmd.events.subscribers.short` | Manage subscribers | 管理订阅者 | 订阅者 = subscribing Agent |
| `cmd.events.subscribers.list.short` | List subscribers | 列出所有订阅者 | same |
| `cmd.events.subscribers.reset.short` | Reset subscriber cursor | 重置订阅者 cursor | same |
| `output.events.subscribers.none` | No subscribers | 无订阅者 | same |
| `output.events.subscribers.reset.ok` | Reset %s cursor | 已重置 %s cursor | Agent name in %s |

The `%s cursor` parts keep English "cursor" in zh too — it's a protocol-level term.

---

## 7. Special Attention: PR Boundary vs alert/engine.go

Confirmed: all 30 alert.go keys are cmd-layer only.

The built-in alert rules (in `alert/engine.go`) have hardcoded title/suggestion strings like:
- `"Deal approaching close date"` / `"Reach out to check progress"`
- `"Deal has not been updated recently"`
- `"Memory entries need updating"`
- `"VIP contact has not been contacted recently"`

These are **not** included in PR 8. They will be migrated in PR 9 when the alert engine itself is i18n'd.

---

## 8. Glossary Compliance Check

- "customer": NOT used in any new key. No exception needed.
- "opportunity": NOT used. Pass.
- "person": NOT used. Pass.
- "subscriber": Used in events.go. Added to glossary as `订阅者`. Pass.

---

## 9. Layout-Embedded Keys

These embed leading spaces/newlines (accepted for v1.0, see §9 of design doc):

```
output.alert.scan.suggestion: "  Suggestion: %s" / "  建议: %s"
output.alert.scan.contact: "  Contact: %s" / "  联系人: %s"
output.alert.scan.deal: "  Deal: %s" / "  商机: %s"
output.alert.list.count: "\n%d pending alerts total\n" / "\n共 %d 条待处理提醒\n"
output.alert.rule.custom: "\nCustom rules:" / "\n用户规则:"
output.memory.list.expired: " [expired]" / " [过期]"
output.memory.propose.conflicts: "  %d conflict(s) detected:" / "  检测到 %d 条冲突:"
output.memory.propose.suggested_action: "  Suggested action: %s" / "  建议动作: %s"
```

All follow the same pattern as PR 6/7.

---

## 10. Test Impact Assessment

### Current Chinese assertions in cmd tests for these files

| File:Line | Current | Type | Action |
|---|---|---|---|
| `cmd_test.go:554` | `"已推进"` | events.go ack output | **WILL CHANGE** — verifies translated output. |
| `cmd_more_test.go:95,136` | `"已导入"` | io.go import output | **OUT OF SCOPE** (PR 9) |
| `cmd_more_test.go:265` | `"已重置"` | events.go subscriber reset output | **WILL CHANGE** — verifies translated output. |

### Assertions that need updating in this PR

**`cmd_test.go` around line 554:**
```go
// events ack — might assert on output text
strings.Contains(out, "已推进")  // → must change to check en output
```

**`cmd_more_test.go` around line 265:**
```go
// events subscriber reset
strings.Contains(out, "已重置")  // → must change to check en output
```

I need to read these exact lines to confirm. These are the only test changes needed.

### cmd_i18n_test.go extension

Extend with alert + events + memory scenarios:

- `TestI18N_AlertEnglish` / `TestI18N_AlertChinese` — `alert scan`, `alert list`, `alert rule list`
- `TestI18N_EventsEnglish` / `TestI18N_EventsChinese` — `events poll`, `events sub list`
- `TestI18N_MemoryEnglish` / `TestI18N_MemoryChinese` — `memory list --scope contact:id`

---

## 11. Commit Splitting Plan

### Commit 1 — `feat(cmd): add sentinels for memory (4), extend userFacingError`

- Modified: `internal/cmd/errors.go`
  - Add 4 memory sentinel errors
- Modified: `internal/cmd/error_handler.go`
  - Add 4 switch cases for memory sentinels
- Modified: `internal/cmd/memory.go`
  - Replace 4 Chinese fmt.Errorf → sentinel returns
- Modified: `internal/cmd/alert.go`
  - Replace 2 direct fmt.Errorf → i18n.T (no sentinel)
- **No translation logic. Pure refactoring.**

### Commit 2 — `feat(i18n): add ~76 translation keys for alert/events/memory`

- Modified: `internal/i18n/messages_en.json`
  - Add ~76 new keys with English translations
- Modified: `internal/i18n/messages_zh.json`
  - Add ~76 new keys with Chinese translations
- Verify: `TestJSONIntegrity` passes
- Verify: `TestNoForbiddenTerms` passes
- Add glossary entry for `subscriber → 订阅者` in `docs/i18n-design.md` §10

### Commit 3 — `refactor(cmd): migrate alert.go + events.go + memory.go to i18n.T()`

- Modified: `internal/cmd/alert.go`
- Modified: `internal/cmd/events.go`
- Modified: `internal/cmd/memory.go`
- Import `"github.com/AgentPal/AgentCRM/internal/i18n"` in all three
- Replace all Short descriptions, flag descriptions, output messages

### Commit 4 — `test(fix): update test assertions + add i18n integration tests`

- Modified: `cmd_test.go` (fix "已推进" assertion at ~line 554)
- Modified: `cmd_more_test.go` (fix "已重置" assertion at ~line 265)
- Modified: `internal/cmd/cmd_i18n_test.go`
  - Add alert/events/memory scenarios

---

## 12. Implementation Checklist

- [ ] Add 4 memory sentinel errors to `internal/cmd/errors.go`
- [ ] Extend `userFacingError()` in `error_handler.go`
- [ ] Replace alert.go fmt.Errorf → direct i18n.T (2 locations)
- [ ] Replace memory.go fmt.Errorf → sentinel returns (4 locations)
- [ ] Add ~76 new keys to messages_en.json + messages_zh.json
- [ ] Add glossary entry for subscriber in docs/i18n-design.md
- [ ] Verify: `TestJSONIntegrity` passes
- [ ] Verify: `TestNoForbiddenTerms` passes
- [ ] Migrate all Short/flag descriptions in alert.go → i18n.T()
- [ ] Migrate all output messages in alert.go → i18n.T()
- [ ] Migrate all Short/flag descriptions in events.go → i18n.T()
- [ ] Migrate all output messages in events.go → i18n.T()
- [ ] Migrate all Short/flag descriptions in memory.go → i18n.T()
- [ ] Migrate all output messages in memory.go → i18n.T()
- [ ] Fix test assertions for events ack + subscriber reset
- [ ] Extend cmd_i18n_test.go for alert/events/memory
- [ ] Push + create PR
