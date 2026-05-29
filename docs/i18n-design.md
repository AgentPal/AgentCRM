# i18n Design — AgentCRM

## Design Decision

**Self-built minimal i18n, no third-party library.**

Rationale:
- Only 2 languages (en, zh) needed for v1.0
- No plural rules, no gender, no RTL
- Avoids adding 1-2MB to binary size (go-i18n + CLDR data)
- Aligns with CLAUDE.md core principle: no heavy dependencies
- Our needs are simple enough that JSON key-value + fmt-style interpolation suffice

---

## 1. Package Structure

```
internal/i18n/
├── i18n.go            # Core API: T(), SetLang(), Init(), embedded JSON
├── messages_en.json   # English strings (default)
├── messages_zh.json   # Chinese strings
└── i18n_test.go       # Unit + integrity tests
```

### Build-time embedding

```go
//go:embed messages_en.json
var enJSON []byte

//go:embed messages_zh.json
var zhJSON []byte
```

Zero external files at runtime. The binary is self-contained.

---

## 2. Core API

### T() — translate a key with optional fmt-style interpolation

```go
package i18n

// T returns the translated string for key in the current language.
// If key is missing, returns "[missing: <key>]" (visible in dev).
// Supports fmt-style formatting via variadic args.
func T(key string, args ...interface{}) string
```

Usage:

```go
i18n.T("cmd.contact.short")                          → "Manage Contacts"
i18n.T("output.contact.created", name)               → "Created contact: Zhang San"
i18n.T("output.contact.merged", kName, kID, mName, mID)
```

### Tn() — translate with number-aware plural (English only)

```go
func Tn(key string, n int, args ...interface{}) string
```

For the rare case where English needs singular/plural distinction. The JSON
stores two keys (`key` and `key.plural`), and `Tn()` selects based on `n`.

### SetLang() — switch language at runtime

```go
func SetLang(lang string) error
```

Accepts `"en"` or `"zh"`. Returns error on unknown code.

### Language selection priority (highest to lowest)

| Priority | Source | Example |
|---|---|---|
| 1 | `--lang` CLI flag | `agentcrm --lang zh contact list` |
| 2 | `AGENTCRM_LANG` env var | `export AGENTCRM_LANG=zh` |
| 3 | `config.json` `lang` field | `{ "lang": "zh" }` |
| 4 | Default | `"en"` |

> **Implementation note (v1.0.0-alpha):** Priority layer 3 (config.json) is
> deferred. Config is loaded after store init, which happens after MustInit.
> Root command handler should call `SetLang(cfg.Lang)` once config is available,
> when no higher-priority source (--flag, env) was set.

Init flow:

```go
func Init(lang string) {
    // 1. If lang param is non-empty, use it (called from root.go with --lang value)
    // 2. Else check AGENTCRM_LANG
    // 3. Else check config.json (loaded after FS init) — deferred
    // 4. Default "en"
}
```

### Fallback behavior

- If a key is missing in the selected language → fall back to `en`
- If a key is missing in all languages → return `[missing: <key>]`
- No panic during translation. Loading errors are surfaced as compile-time
  failures via TestJSONIntegrity (see §8), not as runtime panics.

---

## 3. Key Naming Convention

### Format

```
<area>.<group>[.<subgroup>].<descriptor>
```

Four levels max. Dots separate hierarchy. Lowercase. No spaces. Use underscores
for multi-word descriptors.

### Area prefixes

| Area | Scope | File origin |
|---|---|---|
| `cmd` | Cobra command `Short`, `Long` descriptions | `internal/cmd/*.go` |
| `flag` | Flag descriptions in command registration | `internal/cmd/*.go` |
| `output` | `fmt.Printf` user-facing result messages | `internal/cmd/*.go`, `internal/store/store.go` |
| `error` | `fmt.Errorf` parameter validation + store errors | `internal/cmd/*.go`, `internal/store/events.go` |
| `alert` | Built-in rule titles and suggestions | `internal/alert/engine.go` |
| `config` | Configuration default values | `internal/model/config.go` |

### Descriptor types

| Suffix | Purpose | Example |
|---|---|---|
| `.short` | Cobra Short description | `cmd.contact.short` |
| `.long` | Cobra Long description | `cmd.root.long` |
| `.required` | Missing required param | `error.contact.name.required` |
| `.success` | Success result | `output.deal.created` |
| `.none` | Empty result ("no X found") | `output.contact.list.none` |
| `.count` | Result count | `output.contact.list.count` |
| `.title` | Alert rule title | `alert.rule.stale_deal.title` |
| `.suggestion` | Alert rule suggestion | `alert.rule.stale_deal.suggestion` |

---

## 4. Complete Key List (~370 entries)

### 4.1 `cmd.*` — command descriptions (93 entries)

```json
{
  "cmd.root.short": "本地客户记忆系统 - 给你的 AI Agent 一个共享的客户大脑",
  "cmd.root.long": "AgentCRM 是一个本地化、文件存储、零服务进程的客户领域记忆系统。它以 CLI 工具的形式分发，让 AI Agent 拥有一个长期、可读、可备份的客户记忆。所有数据存储在 ~/AgentCRM/ 目录，文件即真相，可用任何编辑器打开。",
  "cmd.root.version": "显示版本信息",
  "cmd.root.init": "初始化 AgentCRM 数据目录",
  "cmd.root.reindex": "从文件重建 SQLite 索引",
  "cmd.contact.short": "管理联系人",
  "cmd.contact.upsert.short": "创建或更新联系人（按 email 去重）",
  "cmd.contact.get.short": "获取联系人详情",
  "cmd.contact.search.short": "搜索联系人",
  "cmd.contact.list.short": "列出联系人",
  "cmd.contact.update.short": "更新联系人字段（自动归档旧值到 _history）",
  "cmd.contact.history.short": "查看联系人字段历史",
  "cmd.contact.merge.short": "合并两个联系人",
  "cmd.contact.merge.long": "将 merge-id 合并到 keeper-id。merge-id 的文件会被重命名为 .merged-into-<keeper-id>.md。",
  "cmd.deal.short": "管理商机",
  "cmd.deal.create.short": "创建商机",
  "cmd.deal.update.short": "更新商机字段（阶段/金额/其他）",
  "cmd.deal.list.short": "列出商机",
  "cmd.deal.get.short": "获取商机详情",
  "cmd.deal.history.short": "查看商机字段历史",
  "cmd.deal.summarize.short": "生成商机自然语言总结",
  "cmd.activity.short": "管理活动记录",
  "cmd.activity.log.short": "记录一次互动",
  "cmd.activity.list.short": "列出活动",
  "cmd.activity.timeline.short": "查看联系人的时间线（含活动 + 商机变动）",
  "cmd.memory.short": "管理长期记忆",
  "cmd.memory.write.short": "写入记忆",
  "cmd.memory.list.short": "列出记忆",
  "cmd.memory.recall.short": "检索记忆",
  "cmd.memory.forget.short": "删除记忆",
  "cmd.memory.propose.short": "提议新的记忆（带矛盾检测）",
  "cmd.memory.commit.short": "提交/裁决提议",
  "cmd.memory.decay_scan.short": "扫描并标记过期记忆",
  "cmd.events.short": "管理事件订阅（多 Agent 协作）",
  "cmd.events.poll.short": "拉取新事件",
  "cmd.events.ack.short": "推进事件 cursor",
  "cmd.events.watch.short": "持续监听新事件（阻塞式轮询）",
  "cmd.events.subscribers.short": "管理订阅者",
  "cmd.events.subscribers.list.short": "列出所有订阅者",
  "cmd.events.subscribers.reset.short": "重置订阅者 cursor",
  "cmd.alert.short": "管理提醒",
  "cmd.alert.scan.short": "扫描并生成提醒",
  "cmd.alert.list.short": "列出当前提醒",
  "cmd.alert.dismiss.short": "忽略一条提醒",
  "cmd.alert.snooze.short": "推迟提醒",
  "cmd.alert.rule.short": "管理提醒规则",
  "cmd.alert.rule.list.short": "列出规则",
  "cmd.alert.rule.add.short": "添加规则",
  "cmd.alert.rule.disable.short": "禁用规则",
  "cmd.io.export.short": "导出所有数据到 JSON 或 CSV",
  "cmd.io.import.short": "导入数据（vcard/csv）",
  "cmd.io.import.long": "从外部源导入联系人数据。\n来源:\n  vcard  导入 vCard (.vcf) 文件\n  csv    导入 CSV 文件（列: name,email,company,title,phone,tags,source）"
}
```

### 4.2 `flag.*` — flag descriptions (56 entries)

```json
{
  "flag.data_dir": "数据目录（默认 ~/AgentCRM，可被 AGENTCRM_HOME 覆盖）",
  "flag.format": "输出格式: text 或 json",
  "flag.actor": "执行者名称（默认 AGENTCRM_ACTOR 环境变量或 unknown）",
  "flag.contact.name": "联系人姓名（必需）",
  "flag.contact.email": "邮箱",
  "flag.contact.company": "公司",
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
  "flag.deal.title": "商机名称（必需）",
  "flag.deal.contact": "关联联系人 ID",
  "flag.deal.amount": "金额",
  "flag.deal.currency": "币种",
  "flag.deal.stage": "起始阶段",
  "flag.deal.set": "设置字段 (field=value)",
  "flag.deal.reason": "变更原因",
  "flag.deal.stage_filter": "按阶段过滤",
  "flag.deal.stage_not_in": "排除的阶段（逗号分隔）",
  "flag.deal.owner": "按负责人过滤",
  "flag.deal.as_of": "查看指定时间点的数据",
  "flag.deal.field": "字段名",
  "flag.activity.contact": "联系人 ID（必需）",
  "flag.activity.deal": "商机 ID",
  "flag.activity.type": "类型: email|call|meeting|chat|note|social",
  "flag.activity.direction": "方向: in|out",
  "flag.activity.channel": "渠道: gmail|twitter|wechat|phone|...",
  "flag.activity.summary": "一句话摘要（必需）",
  "flag.activity.dedupe_key": "去重键（必需）",
  "flag.activity.body_file": "正文文件路径",
  "flag.activity.since": "起始时间",
  "flag.activity.type_filter": "活动类型",
  "flag.activity.search": "搜索摘要和正文",
  "flag.activity.limit": "返回数量上限",
  "flag.activity.detail": "详细程度: brief|standard|full",
  "flag.alert.since_last_scan": "仅上次扫描后的新数据",
  "flag.alert.permanent": "永久忽略",
  "flag.alert.reason": "忽略原因",
  "flag.alert.days": "推迟天数",
  "flag.alert.from_yaml": "YAML 文件路径",
  "flag.memory.scope": "作用域 (contact:<id> 或 deal:<id>)",
  "flag.memory.text": "记忆内容",
  "flag.memory.decay": "衰减策略",
  "flag.memory.valid_from": "生效时间",
  "flag.memory.top_k": "返回数量上限",
  "flag.memory.as_of": "查看指定时间点",
  "flag.memory.statement": "陈述内容",
  "flag.memory.source_snippet": "来源原文片段",
  "flag.memory.confidence": "置信度 (0-1)",
  "flag.memory.proposal_id": "提议 ID",
  "flag.memory.action": "动作: supersede|keep-both|reject",
  "flag.memory.dry_run": "仅预览不执行",
  "flag.events.as": "订阅者名称",
  "flag.events.filter": "事件过滤器",
  "flag.events.limit": "返回数量上限",
  "flag.events.include_self": "包含自己产生的事件",
  "flag.events.up_to_seq": "推进到此 seq",
  "flag.events.interval": "轮询间隔（秒）",
  "flag.io.format": "导出格式: json|csv",
  "flag.io.out": "输出目录",
  "flag.io.source": "导入源: vcard|csv",
  "flag.io.file": "导入文件路径"
}
```

### 4.3 `output.*` — user-facing result messages (90 entries)

```json
{
  "output.root.init.success": "AgentCRM 已初始化: %s",
  "output.root.version.text": "本地客户记忆系统 - 文件即真相",
  "output.root.reindex.complete": "重建完成: %d 联系人, %d 商机",
  "output.contact.created": "已创建",
  "output.contact.updated": "已更新",
  "output.contact.upsert.success": "%s 联系人: %s (%s)",
  "output.contact.upsert.company": "  公司: %s",
  "output.contact.upsert.title": "  职位: %s",
  "output.contact.get.as_of": "[时间点: %s]",
  "output.contact.search.none": "未找到匹配的联系人",
  "output.contact.search.recent": " [最近: %s]",
  "output.contact.list.none": "无联系人",
  "output.contact.list.count": "\n共 %d 个联系人\n",
  "output.contact.update.same": "%s 已经是该值",
  "output.contact.update.changed": "已更新 %s: %s → %s",
  "output.contact.update.set": "已设置 %s = %s",
  "output.contact.history.none": "无历史记录",
  "output.contact.history.to": "至今",
  "output.contact.merge.success": "已合并: %s (%s) ← %s (%s)",
  "output.contact.merge.reason": "原因: %s",
  "output.deal.created": "已创建商机: %s (%s)",
  "output.deal.created.amount": "  金额: %d %s",
  "output.deal.created.stage": "  阶段: %s",
  "output.deal.stage.current": "阶段已经是 %s",
  "output.deal.stage.changed": "阶段: %s → %s",
  "output.deal.amount.warning": "⚠ 金额变动超过 20%%（%.0f%%），请确认",
  "output.deal.amount.changed": "金额: %d → %s",
  "output.deal.field.updated": "%s 已更新",
  "output.deal.list.none": "无商机",
  "output.deal.list.count": "\n共 %d 个商机\n",
  "output.deal.get.title": "名称: %s",
  "output.deal.get.stage": "阶段: %s",
  "output.deal.get.amount": "金额: %d %s",
  "output.deal.get.close_date": "预计成交: %s",
  "output.deal.get.contacts": "联系人: %s",
  "output.deal.get.stage_history": "阶段历程:",
  "output.deal.history.stage_none": "无阶段历史",
  "output.deal.history.amount_none": "无金额历史",
  "output.deal.history.to": "至今",
  "output.deal.summarize.title": "商机: %s",
  "output.deal.summarize.stage": "当前阶段: %s",
  "output.deal.summarize.stage_days": " (已停留 %d 天)",
  "output.deal.summarize.amount": "金额: %d %s",
  "output.deal.summarize.created_days": "创建: %d 天前\n",
  "output.deal.summarize.close_date": "预计成交: %s",
  "output.deal.summarize.owner": "负责人: %s",
  "output.deal.summarize.history": "阶段历程: %s",
  "output.activity.dup": "活动已存在（重复）: %s",
  "output.activity.logged": "已记录活动: %s",
  "output.activity.type": "  类型: %s",
  "output.activity.summary": "  摘要: %s",
  "output.activity.list.none": "无活动记录",
  "output.activity.list.count": "\n共 %d 条记录\n",
  "output.activity.timeline.none": "无活动记录",
  "output.activity.timeline.deal_section": "\n--- 商机变动 ---",
  "output.activity.timeline.count": "\n共 %d 条活动",
  "output.activity.timeline.deal_count": ", %d 条商机变动",
  "output.memory.written": "已写入记忆: %s",
  "output.memory.list.none": "无记忆",
  "output.memory.list.expired": " [过期]",
  "output.memory.recall.none": "未找到匹配的记忆",
  "output.memory.forgotten": "已删除记忆",
  "output.memory.proposed": "提议已记录: %s (status: %s)",
  "output.memory.conflicts": "  检测到 %d 条冲突:",
  "output.memory.suggested_action": "  建议动作: %s",
  "output.memory.committed": "已处理提议 %s (action: %s)",
  "output.memory.decay.dry_run": "发现 %d 条将要过期的记忆（预览模式，未实际标记）",
  "output.memory.decay.executed": "已标记 %d 条过期记忆",
  "output.events.poll.none": "无新事件",
  "output.events.poll.count": "\n共 %d 个事件 (cursor: %d)",
  "output.events.ack.success": "已推进 %s cursor 到 #%d",
  "output.events.watch.start": "开始监听事件 (actor: %s, interval: %ds, cursor: %d)",
  "output.events.watch.hint": "按 Ctrl+C 停止",
  "output.events.subscribers.none": "无订阅者",
  "output.events.subscribers.reset": "已重置 %s cursor",
  "output.alert.scan.skipped": "存在 %d 条未处理提醒，跳过扫描",
  "output.alert.scan.none": "未发现新问题",
  "output.alert.scan.item": "  建议: %s",
  "output.alert.scan.contact": "  联系人: %s",
  "output.alert.scan.deal": "  商机: %s",
  "output.alert.scan.count": "\n发现 %d 个新提醒\n",
  "output.alert.list.none": "无待处理提醒",
  "output.alert.list.count": "\n共 %d 条待处理提醒\n",
  "output.alert.dismissed": "已忽略提醒",
  "output.alert.snoozed": "已推迟提醒 %d 天",
  "output.alert.rule.builtin": "内置规则:",
  "output.alert.rule.user": "\n用户规则:",
  "output.alert.rule.added": "已添加 %d 条规则",
  "output.alert.rule.disabled": "已禁用规则: %s",
  "output.io.export.skip_contact": "  ⚠ 跳过 %s: %v",
  "output.io.export.contacts": "已导出 %d 个联系人",
  "output.io.export.deals": "已导出 %d 个商机",
  "output.io.export.activities": "已导出 %d 条活动",
  "output.io.export.events": "已导出 %d 条事件",
  "output.io.export.alerts": "已导出 %d 条提醒",
  "output.io.export.done": "导出完成: %s",
  "output.io.import.warn_contact": "  ⚠ 导入失败 %s: %v",
  "output.io.import.warn_file": "  ⚠ 写入文件失败 %s: %v",
  "output.io.import.done": "已导入 %d/%d 个联系人",
  "output.io.import.done_simple": "已导入 %d 个联系人",
  "output.store.reindex.skip": "  ⚠ 跳过 %s: %v",
  "output.store.reindex.contact_fail": "  ⚠ 写入联系人 %s 失败: %v",
  "output.store.reindex.skip_activity": "  ⚠ 跳过活动文件 %s: %v",
  "output.store.reindex.activity_fail": "  ⚠ 写入活动 %s 失败: %v",
  "output.store.reindex.skip_deal": "  ⚠ 跳过商机 %s: %v",
  "output.store.reindex.deal_fail": "  ⚠ 写入商机 %s 失败: %v"
}
```

### 4.4 `error.*` — error messages (22 entries)

```json
{
  "error.contact.name.required": "--name 是必需的",
  "error.contact.set.required": "至少需要一个 --set 参数",
  "error.contact.field.required": "--field 是必需的",
  "error.deal.title.required": "--title 是必需的",
  "error.deal.set.required": "至少需要一个 --set 参数",
  "error.deal.set.format": "无效的 --set 格式: %s",
  "error.deal.stage.irreversible": "无法从 %s 转换到 %s: won/lost 不可逆",
  "error.deal.field.unsupported": "不支持的字段: %s (支持: stage, amount)",
  "error.activity.contact.required": "--contact 是必需的",
  "error.activity.summary.required": "--summary 是必需的",
  "error.activity.dedupe_key.required": "--dedupe-key 是必需的",
  "error.memory.scope_text.required": "--scope 和 --text 是必需的",
  "error.memory.scope.format": "无效 scope 格式，应如 contact:<id>",
  "error.memory.scope.required": "--scope 是必需的",
  "error.memory.proposal_scope.required": "--scope 和 --statement 是必需的",
  "error.memory.proposal_id.required": "--proposal-id 和 --action 是必需的",
  "error.alert.dismiss.not_found": "提醒未找到: %s",
  "error.alert.snooze.not_found": "提醒未找到: %s",
  "error.alert.rule.empty": "规则文件为空",
  "error.io.file.required": "--file 是必需的",
  "error.io.csv.invalid": "CSV 文件需要表头和数据行",
  "error.store.scope.format": "无效 scope 格式，应如 contact:<id>",
  "error.store.proposal.action": "无效动作: %s (支持: supersede, keep-both, reject)"
}
```

### 4.5 `alert.*` — built-in rule content (8 entries)

```json
{
  "alert.rule.stale_deal.title": "商机长时间未跟进",
  "alert.rule.stale_deal.suggestion": "联系客户了解进展，推进商机阶段",
  "alert.rule.closing_deadline.title": "商机即将到截止日期",
  "alert.rule.closing_deadline.suggestion": "确认成交状态或更新预计日期",
  "alert.rule.vip_silence.title": "VIP 联系人长期未联系",
  "alert.rule.vip_silence.suggestion": "发送问候或安排跟进",
  "alert.rule.stale_memory.title": "有记忆条目已过期需要更新",
  "alert.rule.stale_memory.suggestion": "运行 memory decay-scan 清理过期条目"
}
```

### 4.6 `config.*` — default configuration values (1 entry)

```json
{
  "config.memory.preserve_patterns": "决策*, 偏好*, *性格*"
}
```

---

## 5. JSON Message File Format

**Flat key-value format.** No nested objects, no build-time flatten step.

### messages_en.json (flat)

```json
{
  "cmd.root.short": "Local customer memory system — shared brain for your AI agents",
  "cmd.root.long": "AgentCRM is a local-first, file-based, zero-process customer relationship memory system. ...",
  "cmd.root.version": "Show version information",
  "cmd.root.init": "Initialize AgentCRM data directory",
  "cmd.root.reindex": "Rebuild SQLite index from files",
  "output.contact.created": "Created contact: %s (%s)"
}
```

### messages_zh.json (same keys, Chinese values)

```json
{
  "cmd.root.short": "本地客户记忆系统 - 给你的 AI Agent 一个共享的客户大脑",
  "cmd.root.long": "AgentCRM 是一个本地化、文件存储、零服务进程的客户领域记忆系统。...",
  "cmd.root.version": "显示版本信息",
  "cmd.root.init": "初始化 AgentCRM 数据目录",
  "cmd.root.reindex": "从文件重建 SQLite 索引",
  "output.contact.created": "已创建联系人: %s (%s)"
}
```

### Key integrity constraint

en and zh JSON **must have identical key sets** — every key in en must exist in
zh and vice versa. Enforced by `TestJSONIntegrity` (see §8).

JSON is chosen over YAML/TOML because: (a) no parsing library needed, (b)
`//go:embed` works directly, (c) IDE autocomplete.

---

## 6. Embedding + Loading Strategy

```go
//go:embed messages_en.json
var enJSON []byte

//go:embed messages_zh.json
var zhJSON []byte

var messages map[string]map[string]string  // lang → key → value

// MustInit loads embedded JSON into memory. Panics only if JSON is
// syntactically invalid — but this is caught by TestJSONIntegrity at
// test time, so it should never reach production.
func MustInit() {
    messages = make(map[string]map[string]string)
    for lang, data := range map[string][]byte{"en": enJSON, "zh": zhJSON} {
        var m map[string]string
        if err := json.Unmarshal(data, &m); err != nil {
            panic("i18n: corrupt embedded json: " + err.Error())
        }
        messages[lang] = m
    }
}
```

The JSON is embedded as flat key-value maps at compile time. No runtime file
I/O. No directory scanning. The map is populated once during package init.

---

## 7. Error Handling Integration

Two distinct layers with different i18n strategies:

### Layer 1: Internal sentinel errors (always English)

Inside Go code, use `errors.New` + `%w` for error chains. These are **never
shown directly to end users**. They travel through call stacks via standard Go
error wrapping.

```go
// Define sentinel errors in the relevant package
var ErrFieldRequired = errors.New("--name is required")

// Use %w for wrapping
if name == "" {
    return fmt.Errorf("%w", ErrFieldRequired)
}

// Wrapping with context
if _, err := strconv.Atoi(amount); err != nil {
    return fmt.Errorf("parse --amount: %w", err)
}
```

### Layer 2: CLI error translation (i18n boundary)

At the CLI boundary (cobra `RunE` handlers or a middleware wrapper), sentinel
errors are mapped to i18n keys for user-facing output:

```go
// In root.go or a centralized error handler
func userFacingError(err error) string {
    if errors.Is(err, store.ErrFieldRequired) {
        return i18n.T("error.contact.name.required")
    }
    // Unrecognized errors → show as-is (likely English, acceptable for edge cases)
    return err.Error()
}
```

This two-layer approach preserves:
- `errors.Is()` / `errors.As()` for programmatic checks
- English error messages in logs and stack traces (universal developer language)
- Translated messages only at the user-facing boundary
- No `%w` wrapping through translated strings (which would break error chains)

Migration rule: **sentinel errors stay English; user-facing translations
enter only at the cobra RunE boundary.** The current code mixes both in
`fmt.Errorf` — the i18n migration must first extract sentinels, then add
translation at the boundary.

---

## 8. Testing Strategy

### Unit tests

```go
func TestT_Simple(t *testing.T) {
    SetLang("en")
    got := T("cmd.contact.short")
    if got == "" || strings.Contains(got, "missing") {
        t.Errorf("unexpected: %s", got)
    }
}

func TestT_Interpolation(t *testing.T) {
    SetLang("en")
    got := T("output.contact.created", "Zhang San")
    want := "Created contact: Zhang San"
    if got != want {
        t.Errorf("got %q, want %q", got, want)
    }
}

func TestT_Fallback(t *testing.T) {
    SetLang("zh")
    got := T("nonexistent.key")
    if !strings.Contains(got, "missing") {
        t.Errorf("expected missing key indicator, got %s", got)
    }
}

func TestSetLang_Invalid(t *testing.T) {
    err := SetLang("fr")
    if err == nil {
        t.Error("expected error for unsupported language")
    }
}
```

### Integrity tests (CI gate)

```go
// TestJSONIntegrity validates both JSON files at test time.
// This catches: syntax errors, missing keys in either language,
// empty values, duplicate keys.
func TestJSONIntegrity(t *testing.T) {
    // 1. Parse both files
    // 2. Compare key sets — must be identical
    // 3. Reject empty values
    // 4. Check for %s/%d placeholder consistency
}
```

This test must pass in CI before any i18n change can merge. If a developer
adds a key to `messages_en.json` but forgets `messages_zh.json`, the test
fails.

### Term compliance test (optional CI gate)

```go
// TestNoForbiddenTerms scans message values for terms that violate
// the glossary. This catches drift before review.
func TestNoForbiddenTerms(t *testing.T) {
    forbidden := map[string]string{
        "customer": "use 'contact' instead (see glossary)",
        "opportunity": "use 'deal' instead (see glossary)",
    }
    for lang, msgs := range messages {
        for key, val := range msgs {
            for term, msg := range forbidden {
                if strings.Contains(val, term) {
                    t.Errorf("[%s] %s = %q — %s", lang, key, val, msg)
                }
            }
        }
    }
}
```

---

## 9. Known Issues: Layout-Embedded Keys

Some `output.*` keys embed layout characters (leading spaces, colons, newlines):

```json
{
  "output.contact.upsert.company": "  公司: %s",
  "output.contact.upsert.title": "  职位: %s",
  "output.activity.type": "  类型: %s"
}
```

This violates the i18n principle that translations should contain only
semantic content, not layout. The root cause is that the original code
interleaved layout and content in `fmt.Printf`.

**Impact:** When translating to English, the layout characters stay correct,
but the translation must remember to preserve leading spaces and colons:

```json
{
  "output.contact.upsert.company": "  Company: %s"
}
```

**Planned cleanup (post-v1.0):** Extract layout from translations:
- Code becomes: `fmt.Printf("  %s: %s\n", T("output.field.company"), value)`
- Translation becomes: `"output.field.company": "Company"`

For the initial v1.0.0-alpha release, keeping layout in the translation is
acceptable. The affected keys are marked with a `⚠ layout-embedded` comment
in the JSON source for future cleanup.

---

## 10. Glossary

| Chinese | English | Forbidden alternatives |
|---|---|---|
| 联系人 | Contact | person, customer, lead |
| 商机 | Deal | opportunity |
| 活动/互动 | Activity | — |
| 记忆 | Memory / note (user-facing) | — |
| 提议 | Proposal / propose (verb) | — |
| 衰减 | Decay | — |
| 订阅者 | Subscriber | — |
| 提醒 | Alert | notification |
| 规则 | Rule | — |
| 币种 | Currency | — |
| 阶段 | Stage | phase, step |
| 标签 | Tag | label, category |
| 去重键 | Dedupe-key | — |

### Glossary exceptions

Brand-level keys where "customer" refers to the product category, not data:

- `cmd.root.short` / `cmd.root.long` — product tagline and positioning
- `output.root.version.text` — brand tagline in version output

These keys describe AgentCRM as a product category (Customer Relationship
Memory), not as a data model. The forbidden word "customer" in these
specific keys is intentional brand language, not term drift.

All other keys (cmd.contact.*, cmd.deal.*, flag.*, output.contact.*, etc.)
remain subject to the standard glossary check — they describe data
operations and must use the canonical term "contact".

This exception list is enforced by TestNoForbiddenTerms via the
customerExceptions map.

---

## 11. English Style Guide

### CLI messages

- **Lowercase first letter** for error messages (Go convention)
- **No trailing period** on short messages
- **Imperative voice** for required flag descriptions: `"--email is required"` not `"the email is a required parameter"`
- **Direct phrasing**: `"specify --email"` not `"please input email"`
- **Flag references use full flag name**: `"--email"` not `"the email flag"`
- **Consistent label/colons**: `"Company: %s"` with a single space before the colon

### Good examples

```
cmd.contact.short             = "Manage contacts"
cmd.contact.upsert.short      = "Create or update contact (deduped by email)"
flag.contact.name             = "contact name (required)"
flag.contact.email            = "email"
error.contact.name.required   = "--name is required"
error.contact.set.required    = "at least one --set is required"
output.contact.created        = "Created contact: %s (%s)"
output.contact.updated        = "Updated contact: %s (%s)"
output.contact.list.none      = "No contacts"
output.contact.list.count     = "\n%d contacts total\n"
```

### Tone

- Professional but not corporate. Think `git` or `jq` CLI output.
- Error messages own the failure: `"invalid scope format"` not `"you entered an invalid scope"`
- Success messages confirm concisely: `"Created contact: Zhang San (cnt_abc123)"`

---

## 12. Migration Strategy

### Per-file batches (5 PRs)

| PR | Files | Key count | Scope |
|---|---|---|---|
| 1 | `internal/i18n/*` | ~10 | Core i18n package + JSON files |
| 2 | `internal/cmd/root.go` + `contact.go` | ~70 | Root + Contact commands |
| 3 | `internal/cmd/deal.go` + `activity.go` | ~90 | Deal + Activity commands |
| 4 | `internal/cmd/alert.go` + `events.go` + `memory.go` | ~110 | Alert + Events + Memory |
| 5 | `internal/cmd/io.go` + `store/store.go` + `store/events.go` + `alert/engine.go` + `model/config.go` | ~90 | IO + Store + Alert engine |

### Migration pattern per file

```go
// Before
func init() {
    contactCmd.Use = "contact"
    contactCmd.Short = "管理联系人"
}

// After
func init() {
    contactCmd.Use = "contact"
    contactCmd.Short = T("cmd.contact.short")
}
```

### Error migration pattern

```go
// Before (current)
return fmt.Errorf("--name 是必需的")

// Step 1: extract sentinel
var ErrNameRequired = errors.New("--name is required")
// ...in handler:
if name == "" {
    return ErrNameRequired
}

// Step 2: translate at boundary (in root.go RunE wrapper)
err = cmd.Execute()
if err != nil {
    fmt.Fprintf(os.Stderr, "%s\n", userFacingError(err))
    os.Exit(1)
}
```

Sentinel errors are defined in the package where they originate (e.g.
`internal/store/errors.go`, `internal/cmd/errors.go`).

### Test migration

Test files (`_test.go`) contain Chinese in fixture data (names like "张三",
assertion messages like "期望 张三, 得到 %s"). These fall into two categories:

1. **Fixture names** — Test data names can stay Chinese; they're data, not UI
2. **Assertion messages** (`t.Errorf("期望 %s, 得到 %s", ...)`) — should be
   migrated to English, but this is low priority since only devs read them

---

## 13. Exclusions (v1.0 scope)

- Skill package (`skills/`) — Agent-facing content, always in the agent's
  language. No i18n needed.
- CLI output field names (e.g. `--format json` field names like `"name"`,
  `"company"`) — these are API contract, not user-facing text. Always English.
- `README.md` / `README.zh.md` — already bilingual.
- Go comments — English only is standard practice.
