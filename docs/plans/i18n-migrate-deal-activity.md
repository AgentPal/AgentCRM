# Plan: i18n Migration — deal.go + activity.go (PR 7)

## 1. Scope Confirmation

### deal.go — keys needed

| Area | Total | Already exist | Net new |
|---|---|---|---|
| `cmd.deal.*` | 7 | 1 (`cmd.deal.short`) | 6 |
| `flag.deal.*` | 11 | 2 (`flag.deal.title`, `flag.deal.amount`) | 9 |
| `error.deal.*` | 5 | 0 | 5 |
| `output.deal.*` | 21 | 1 (`output.deal.list.count`) | 20 |
| Total | **44** | **4** | **40** |

### activity.go — keys needed

| Area | Total | Already exist | Net new |
|---|---|---|---|
| `cmd.activity.*` | 4 | 1 (`cmd.activity.short`) | 3 |
| `flag.activity.*` | 13 | 2 (`flag.activity.direction`, `flag.activity.type`) | 11 |
| `error.activity.*` | 3 | 1 (`error.activity.contact.required`) | 2 |
| `output.activity.*` | 9 | 1 (`output.activity.logged`) | 8 |
| Total | **29** | **5** | **24** |

### Grand total: 73 keys, 64 net new

The original design doc estimated ~90 for this scope. The difference is that ~10 keys were already shipped in PR 4/PR 6, and the design doc was a rough estimate.

---

## 2. Sentinel Error Extraction Plan

### deal.go — Chinese fmt.Errorf calls

| Line | Current | Proposed sentinel | Key |
|---|---|---|---|
| 29 | `"--title 是必需的"` | `ErrDealTitleRequired` | `error.deal.title.required` |
| 96 | `"至少需要一个 --set 参数"` | `ErrDealSetRequired` | `error.deal.set.required` |
| 113 | `"无效的 --set 格式: %s"` | `ErrDealSetFormat` | `error.deal.set.format` |
| 120 | `"无法从 %s 转换到 %s: won/lost 不可逆"` | `ErrDealStageIrreversible` | `error.deal.stage.irreversible` |
| 297 | `"--field 是必需的"` | `ErrDealFieldRequired` | `error.deal.field.required` |
| 358 | `"不支持的字段: %s (支持: stage, amount)"` | `ErrDealFieldUnsupported` | `error.deal.field.unsupported` |

Note: `split2` validation at deal.go:111 (`"无效的 --set 格式: %s"`) is a dynamic error **not** extracted as sentinel in PR 6's contact.go either — the contact equivalent stayed as raw `fmt.Errorf`. Following the same pattern: extract it here for i18n but as a direct `fmt.Errorf(i18n.T("error.deal.set.format"), set)` in the RunE handler (no sentinel needed for single-use format errors).

### activity.go — Chinese fmt.Errorf calls

| Line | Current | Proposed sentinel | Key |
|---|---|---|---|
| 32 | `"--contact 是必需的"` | `ErrActivityContactRequired` | `error.activity.contact.required` (already exists in JSON) |
| 35 | `"--summary 是必需的"` | `ErrActivitySummaryRequired` | `error.activity.summary.required` |
| 38 | `"--dedupe-key 是必需的"` | `ErrActivityDedupeKeyRequired` | `error.activity.dedupe_key.required` |

### Decision: Append to existing `internal/cmd/errors.go`

```go
var (
    // Contact (PR 6)
    ErrContactNameRequired  = errors.New("--name is required")
    ErrContactSetRequired   = errors.New("at least one --set is required")
    ErrContactFieldRequired = errors.New("--field is required")

    // Deal (PR 7)
    ErrDealTitleRequired      = errors.New("--title is required")
    ErrDealSetRequired        = errors.New("at least one --set is required")
    ErrDealStageIrreversible  = errors.New("cannot transition from %s to %s: won/lost is irreversible")
    ErrDealFieldRequired      = errors.New("--field is required")
    ErrDealFieldUnsupported   = errors.New("unsupported field: %s (supported: stage, amount)")

    // Activity (PR 7)
    ErrActivityContactRequired  = errors.New("--contact is required")
    ErrActivitySummaryRequired  = errors.New("--summary is required")
    ErrActivityDedupeKeyRequired = errors.New("--dedupe-key is required")
)
```

---

## 3. userFacingError() Extension

Add new cases to `error_handler.go` switch:

```go
// Deal errors
case errors.Is(err, ErrDealTitleRequired):
    return i18n.T("error.deal.title.required")
case errors.Is(err, ErrDealSetRequired):
    return i18n.T("error.deal.set.required")
case errors.Is(err, ErrDealStageIrreversible):
    return i18n.T("error.deal.stage.irreversible")
case errors.Is(err, ErrDealFieldRequired):
    return i18n.T("error.deal.field.required")
case errors.Is(err, ErrDealFieldUnsupported):
    return i18n.T("error.deal.field.unsupported")

// Activity errors
case errors.Is(err, ErrActivityContactRequired):
    return i18n.T("error.activity.contact.required")
case errors.Is(err, ErrActivitySummaryRequired):
    return i18n.T("error.activity.summary.required")
case errors.Is(err, ErrActivityDedupeKeyRequired):
    return i18n.T("error.activity.dedupe_key.required")
```

`ErrDealSetFormat` is NOT a sentinel — it's used directly: `fmt.Errorf(i18n.T("error.deal.set.format"), set)` in the RunE handler.

No changes to `Execute()` or `rootCmd` — the error handling infrastructure is already in place from PR 6.

---

## 4. JSON Key Translation Draft

### 4.1 deal.go — new cmd.* keys

| Key | en | zh |
|---|---|---|
| `cmd.deal.create.short` | Create deal | 创建商机 |
| `cmd.deal.update.short` | Update deal fields (stage/amount/other) | 更新商机字段（阶段/金额/其他） |
| `cmd.deal.list.short` | List deals | 列出商机 |
| `cmd.deal.get.short` | Show deal details | 获取商机详情 |
| `cmd.deal.history.short` | View deal field history | 查看商机字段历史 |
| `cmd.deal.summarize.short` | Generate deal summary | 生成商机自然语言总结 |

### 4.2 deal.go — new flag.* keys

| Key | en | zh |
|---|---|---|
| `flag.deal.contact` | contact ID | 关联联系人 ID |
| `flag.deal.currency` | currency | 币种 |
| `flag.deal.stage` | starting stage | 起始阶段 |
| `flag.deal.set` | set field (field=value) | 设置字段 (field=value) |
| `flag.deal.reason` | change reason | 变更原因 |
| `flag.deal.stage_filter` | filter by stage | 按阶段过滤 |
| `flag.deal.stage_not_in` | exclude stages (comma separated) | 排除的阶段（逗号分隔） |
| `flag.deal.owner` | filter by owner | 按负责人过滤 |
| `flag.deal.as_of` | view data as of this time | 查看指定时间点的数据 |
| `flag.deal.field` | field name | 字段名 |

### 4.3 deal.go — new error.* keys

| Key | en | zh |
|---|---|---|
| `error.deal.title.required` | --title is required | --title 是必需的 |
| `error.deal.set.required` | at least one --set is required | 至少需要一个 --set 参数 |
| `error.deal.set.format` | invalid --set format: %s (expected field=value) | 无效的 --set 格式: %s |
| `error.deal.stage.irreversible` | cannot transition from %s to %s: won/lost is irreversible | 无法从 %s 转换到 %s: won/lost 不可逆 |
| `error.deal.field.required` | --field is required | --field 是必需的 |
| `error.deal.field.unsupported` | unsupported field: %s (supported: stage, amount) | 不支持的字段: %s（支持: stage, amount） |

### 4.4 deal.go — new output.* keys

| Key | en | zh |
|---|---|---|
| `output.deal.created` | Created deal: %s (%s) | 已创建商机: %s (%s) |
| `output.deal.created.amount` |   Amount: %d %s |   金额: %d %s |
| `output.deal.created.stage` |   Stage: %s |   阶段: %s |
| `output.deal.stage.current` | Stage already %s | 阶段已经是 %s |
| `output.deal.stage.changed` | Stage: %s → %s | 阶段: %s → %s |
| `output.deal.amount.warning` | ⚠ Amount changed over 20%% (%.0f%%), confirm | ⚠ 金额变动超过 20%%（%.0f%%），请确认 |
| `output.deal.amount.changed` | Amount: %d → %s | 金额: %d → %s |
| `output.deal.field.updated` | %s updated | %s 已更新 |
| `output.deal.list.none` | No deals | 无商机 |
| `output.deal.get.as_of` | [as of: %s] | [时间点: %s] |
| `output.deal.get.title` | Title: %s | 名称: %s |
| `output.deal.get.stage` | Stage: %s | 阶段: %s |
| `output.deal.get.amount` | Amount: %d %s | 金额: %d %s |
| `output.deal.get.close_date` | Close date: %s | 预计成交: %s |
| `output.deal.get.contacts` | Contacts: %s | 联系人: %s |
| `output.deal.get.stage_history` | Stage history: | 阶段历程: |
| `output.deal.history.stage_none` | No stage history | 无阶段历史 |
| `output.deal.history.amount_none` | No amount history | 无金额历史 |
| `output.deal.history.to` | present | 至今 |
| `output.deal.summarize.title` | Deal: %s | 商机: %s |
| `output.deal.summarize.stage` | Current stage: %s | 当前阶段: %s |
| `output.deal.summarize.stage_days` |  (%d days) | （已停留 %d 天） |
| `output.deal.summarize.amount` | Amount: %d %s | 金额: %d %s |
| `output.deal.summarize.created_days` | Created: %d days ago | 创建: %d 天前 |
| `output.deal.summarize.close_date` | Close date: %s | 预计成交: %s |
| `output.deal.summarize.owner` | Owner: %s | 负责人: %s |
| `output.deal.summarize.history` | Stage history: %s | 阶段历程: %s |

### 4.5 activity.go — new cmd.* keys

| Key | en | zh |
|---|---|---|
| `cmd.activity.log.short` | Log an interaction | 记录一次互动 |
| `cmd.activity.list.short` | List activities | 列出活动 |
| `cmd.activity.timeline.short` | View contact timeline (activities + deal changes) | 查看联系人的时间线（含活动 + 商机变动） |

### 4.6 activity.go — new flag.* keys

| Key | en | zh |
|---|---|---|
| `flag.activity.contact` | contact ID (required) | 联系人 ID（必需） |
| `flag.activity.deal` | deal ID | 商机 ID |
| `flag.activity.channel` | channel: gmail\|twitter\|wechat\|phone\|... | 渠道: gmail\|twitter\|wechat\|phone\|... |
| `flag.activity.summary` | one-line summary (required) | 一句话摘要（必需） |
| `flag.activity.dedupe_key` | dedupe key (required) | 去重键（必需） |
| `flag.activity.body_file` | body file path | 正文文件路径 |
| `flag.activity.since` | starting time | 起始时间 |
| `flag.activity.type_filter` | activity type | 活动类型 |
| `flag.activity.search` | search summary and body | 搜索摘要和正文 |
| `flag.activity.limit` | max results | 返回数量上限 |
| `flag.activity.detail` | detail level: brief\|standard\|full | 详细程度: brief\|standard\|full |

### 4.7 activity.go — new error.* keys

| Key | en | zh |
|---|---|---|
| `error.activity.summary.required` | --summary is required | --summary 是必需的 |
| `error.activity.dedupe_key.required` | --dedupe-key is required | --dedupe-key 是必需的 |

### 4.8 activity.go — new output.* keys

| Key | en | zh |
|---|---|---|
| `output.activity.dup` | Activity already exists (duplicate): %s | 活动已存在（重复）: %s |
| `output.activity.type` |   Type: %s |   类型: %s |
| `output.activity.summary` |   Summary: %s |   摘要: %s |
| `output.activity.list.none` | No activities | 无活动记录 |
| `output.activity.list.count` | \n%d records total\n | \n共 %d 条记录\n |
| `output.activity.timeline.none` | No activities | 无活动记录 |
| `output.activity.timeline.deal_section` | \n--- Deal changes --- | \n--- 商机变动 --- |
| `output.activity.timeline.count` | \n%d activities total | \n共 %d 条活动 |
| `output.activity.timeline.deal_count` | , %d deal changes | , %d 条商机变动 |

### 4.9 Glossary compliance check

- "opportunity": NOT used in any key. Glossary check passes.
- "customer": NOT used in any key. No exception needed.
- "person": NOT used in any key. Glossary check passes.

No glossary exceptions required for PR 7.

### 4.10 Layout-embedded keys (see §9 of design)

These embed leading spaces/newlines in translations (accepted for v1.0):

```
output.deal.created.amount: "  Amount: %d %s" / "  金额: %d %s"
output.deal.created.stage: "  Stage: %s" / "  阶段: %s"
output.activity.type: "  Type: %s" / "  类型: %s"
output.activity.summary: "  Summary: %s" / "  摘要: %s"
output.deal.summarize.stage_days: " (%d days)" / "（已停留 %d 天）"
output.deal.summarize.created_days: "Created: %d days ago" / "创建: %d 天前"
output.deal.list.count: "\n%d deals total\n" / "\n共 %d 个商机\n"
output.activity.timeline.deal_section: "\n--- Deal changes ---" / "\n--- 商机变动 ---"
output.activity.list.count: "\n%d records total\n" / "\n共 %d 条记录\n"
output.activity.timeline.count: "\n%d activities total" / "\n共 %d 条活动"
output.activity.timeline.deal_count: ", %d deal changes" / ", %d 条商机变动"
```

Known and consistent with existing PR 6 keys.

---

## 5. Test Impact Assessment

### Full Chinese character scan of cmd test files

#### Deal-related assertions

| File:Line | Current | Type | Action |
|---|---|---|---|
| `cmd_test.go:275` | `strings.Contains(out, "qualified → proposal")` | Asserts stage names in update output | **NO CHANGE** — only checks stage name, not the label |
| `cmd_test.go:323` | `strings.Contains(out, "企业版订阅")` | Asserts deal title in summarize output | **NO CHANGE** — fixture data (title), not translated message |
| `cmd_test.go:257-258` | `d.Title != "企业版订阅"` | Fixture data assertion | **NO CHANGE** |

All other deal test assertions use `--format json` — no i18n impact.

#### Activity-related assertions

| File:Line | Current | Type | Action |
|---|---|---|---|
| `cmd_test.go:405` | `strings.Contains(out, "产品介绍")` | Asserts summary text in timeline output | **NO CHANGE** — fixture data (summary), not translated message |

All other activity test assertions use `--format json` — no i18n impact.

#### Assertions NOT in scope (other PRs)

| File:Line | Current | Located in | PR |
|---|---|---|---|
| `cmd_test.go:554` | `"已推进"` | events.go (ack) | 8 |
| `cmd_more_test.go:95,136` | `"已导入"` | io.go (import) | 9 |
| `cmd_more_test.go:265` | `"已重置"` | events.go (subscribers) | 8 |

#### Chinese characters stay as-is (fixture data)

All deal/activity fixture data in tests (titles like "企业版订阅", "测试商机"; reasons like "已发送方案", "增加了用户数"; summaries like "发送了产品介绍", "电话沟通需求"; names like "测试联系人", "记忆测试") are fixture data, not UI strings. Unchanged.

**Conclusion: Zero test assertions need to change in this PR.**

This is different from PR 6 (which had 2 assertion changes) because:
- All deal tests that check output use `--format json` (no i18n impact)
- The one text-mode assertion (`qualified → proposal`) checks variable content, not labels
- Activity text-mode output checks (`产品介绍`) uses fixture data

### cmd_i18n_test.go extension

The existing `TestI18N_ChineseOutput` / `TestI18N_EnglishDefault` verify the i18n pipeline for `init` + `contact list`. Adding deal + activity would verify key existence but not add pipeline test coverage. **Optional** — only add if needed for confidence. Suggest adding deal create + activity list to `TestI18N_ChineseOutput` for completeness.

---

## 6. Commit Splitting Plan

### Commit 1 — `feat(cmd): add sentinels for deal + activity, extend userFacingError`

- Modified: `internal/cmd/errors.go`
  - Add sentinel error variables for deal (ErrDealTitleRequired, ErrDealSetRequired, ErrDealStageIrreversible, ErrDealFieldRequired, ErrDealFieldUnsupported) and activity (ErrActivityContactRequired, ErrActivitySummaryRequired, ErrActivityDedupeKeyRequired)
- Modified: `internal/cmd/error_handler.go`
  - Add switch cases for all new sentinels
- Modified: `internal/cmd/deal.go`
  - Replace Chinese fmt.Errorf with sentinel returns (6 locations)
- Modified: `internal/cmd/activity.go`
  - Replace Chinese fmt.Errorf with sentinel returns (3 locations)
- Verify: `go build ./...` compiles

**No translation logic in this commit. Pure refactoring.**

### Commit 2 — `feat(i18n): add ~64 translation keys for deal + activity`

- Modified: `internal/i18n/messages_en.json`
  - Add ~64 new keys with English translations
- Modified: `internal/i18n/messages_zh.json`
  - Add ~64 new keys with Chinese translations
- Verify: `TestJSONIntegrity` passes (en/zh key parity)
- Verify: `TestNoForbiddenTerms` passes (glossary compliance)

### Commit 3 — `refactor(cmd): migrate deal.go + activity.go to i18n.T()`

- Modified: `internal/cmd/deal.go`
  - Replace Short descriptions: `i18n.T("cmd.deal.*")`
  - Replace flag descriptions: `i18n.T("flag.deal.*")`
  - Replace output messages in RunE handlers
- Modified: `internal/cmd/activity.go`
  - Replace Short descriptions: `i18n.T("cmd.activity.*")`
  - Replace flag descriptions: `i18n.T("flag.activity.*")`
  - Replace output messages in RunE handlers
- Import `"github.com/AgentPal/AgentCRM/internal/i18n"` in both files
- Verify: `go build ./...` compiles

### Commit 4 — `test(cmd): add deal + activity to i18n integration tests`

- Modified: `internal/cmd/cmd_i18n_test.go`
  - Extend `TestI18N_ChineseOutput`: add deal create + activity log in Chinese mode
  - Extend `TestI18N_EnglishDefault`: add deal create + activity log in English mode
- Run: `go test ./internal/cmd/...` — all pass
- Run: `go test ./...` — no regressions

---

## 7. Implementation Checklist

- [ ] Add sentinel errors to `internal/cmd/errors.go` (deal + activity)
- [ ] Extend `userFacingError()` in `error_handler.go`
- [ ] Replace deal.go Chinese fmt.Errorf → sentinel returns (6 locations)
- [ ] Replace activity.go Chinese fmt.Errorf → sentinel returns (3 locations)
- [ ] Verify: `go build ./...` compiles
- [ ] Add ~64 new keys to messages_en.json + messages_zh.json
- [ ] Verify: `TestJSONIntegrity` passes
- [ ] Verify: `TestNoForbiddenTerms` passes
- [ ] Migrate all Short/flag descriptions in deal.go → i18n.T()
- [ ] Migrate all fmt.Printf output in deal.go → i18n.T()
- [ ] Migrate all Short/flag descriptions in activity.go → i18n.T()
- [ ] Migrate all fmt.Printf output in activity.go → i18n.T()
- [ ] Extend cmd_i18n_test.go for deal + activity
- [ ] Run `go test ./internal/i18n/...` — all pass
- [ ] Run `go test ./internal/cmd/...` — all pass
- [ ] Run `go test ./...` — no regressions
- [ ] Push + create PR
