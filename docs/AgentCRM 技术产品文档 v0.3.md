好。v0.3 的核心变化是：产品定位从"轻量 CRM"升级到"客户领域记忆系统"，数据模型加入分层和 temporal，CLI 加入 propose/commit、decay、history、多 Agent 协作（事件订阅、actor），并保留 v0.2 的本地/零服务/文件即真相原则。

我直接写出来。

---

# AgentCRM 技术产品文档 v0.3

**状态**：开发立项文档（v1.0 完整版设计） **版本**：v0.3（合并 v0.2 + 多 Agent 协作 + 记忆系统借鉴） **受众**：开发团队、社区贡献者 **日期**：2026-05-28

---

## 0. TL;DR

**AgentCRM 是一个本地化、文件存储、零服务进程的客户领域记忆系统**。它以 Skill 包 + CLI 工具的形式分发，让 Claude Code、Claude Desktop、OpenClaw、Codex、Hermes 等任何支持 Skill 的 Agent，在装上之后立刻拥有一个长期、可读、可备份的客户记忆。

它不是 CRM 软件，不是 SaaS 服务。它是你 home 目录下的一个文件夹 (`~/.agentcrm/`)，加上一个 Skill 包和一个静态编译的 CLI 二进制。

**v0.3 相对于 v0.2 的本质升级**：从"轻量 CRM"重新定位为"领域特化的记忆系统"。这意味着我们从 Mem0、Supermemory、Zep、Hindsight 等通用记忆系统借鉴了五项关键能力：记忆分层、temporal validity、propose/commit 矛盾消解、多策略 hybrid retrieval、selective forgetting。同时加入了完整的多 Agent 协作机制（事件订阅 + actor 标记）。

**v1.0 是完整可用版本，不是 MVP**。开源免费，许可证 MIT 或 Apache 2.0。未来 Pro 版本作为加法（托管同步、后台 ingestion、向量增强等），永不作为对开源版本的功能阉割。

---

## 1. 这是什么

### 1.1 一句话定位（升级版）

> **AgentCRM 是 Mem0 / Supermemory 在客户关系领域的特化版本。它给你的每一个 AI Agent 一个共享的、本地的、文件即真相的客户大脑。**

### 1.2 心智模型：领域特化的记忆系统

通用记忆系统（Mem0、Supermemory、Zep、Hindsight）解决的是"Agent 跨会话记住任何事"。它们是通用 SDK，给开发者用，存任何对象，目标是建一个普适的记忆基础设施。

AgentCRM 解决的是同一类问题，但范围聚焦在客户关系这一个垂直领域。这种聚焦带来三个收益：

1. **领域模型可以做深**：通用记忆系统不知道什么是"商机阶段"、什么是"客户偏好"、什么是"上次跟进"。AgentCRM 知道
2. **CLI 接口可以做窄做好**：通用记忆系统提供 generic add/search/get，AgentCRM 提供 `deal stage`、`timeline`、`alert scan` 这种业务原语
3. **Skill 智慧可以做厚**：通用记忆系统不能告诉 Agent "丢单时该按什么流程操作"，AgentCRM 的 Skill 可以

它和通用记忆系统不竞争。用户的 Agent 可以同时挂 Mem0（通用记忆）和 AgentCRM（客户记忆），各管一摊。

### 1.3 七条铁律（决定每个设计选择）

1. **AgentCRM 是 Agent 的工具，不是人的应用**。任何 PRD 写到"用户打开 X 页面"——重新审视
2. **文件就是真相，能用任何编辑器打开**。SQLite 只做索引，永远可重建
3. **没有服务进程**。CLI 按需运行，跑完就退。零 daemon
4. **多 Agent 是常态，不是例外**。每个 CLI 调用必须知道是谁在调
5. **记忆有生命周期**。每条信息都要回答：什么时候有效？什么时候过期？
6. **CLI 提供能力，Skill 提供智慧**。CLI 不替 Agent 做推理决策，但提供检测矛盾、对比时间、按 actor 过滤等能力
7. **可观测大于可配置**。先让用户看清 Agent 做了什么，再考虑能让他们配什么

### 1.4 和市面其他产品的位置图

```
            通用 ←──────────────────────────→ 领域特化
   ↑
   云  Mem0 Cloud, Supermemory       Coffee, Attio, HubSpot
端                                   (传统 CRM + AI 插件)
形
态  Mem0 OSS, Zep OSS              ★ AgentCRM v1.0 ★
   Hindsight, Letta, Memori        (本地、文件、Skill)
   本
   地
   ↓
```

我们占的位置：**本地端 + 领域特化**。这个象限目前没有商业化产品。

---

## 2. 为什么这样设计

### 2.1 为什么本地纯文件（保留 v0.2 论点）

四条理由：

- **OCP 用户已经在文件里工作**：他们的笔记在 Obsidian/Markdown，代码在本地仓库，Agent 在本机跑。给他们云端 CRM 是逆流
- **开源产品要让人立刻能跑**：注册 + OAuth + 配 Postgres 的开源 CRM，文档读完用户已经走了
- **数据完全归用户是真正的护城河**：随时可以 `rm -rf ~/.agentcrm/`，随时可以打开 markdown 看内容。这种"反锁定"恰恰是 OCP 群体愿意长期用的理由
- **分发渠道天然存在**：GitHub、Claude Skill Marketplace、OpenClaw `awesome-skills`——这些天然为开源 Skill 设计

### 2.2 为什么定位升级为"记忆系统"

通用记忆系统在 2026 趟过的坑，我们直接借鉴，跳过两年弯路：

- **Mem0/Zep 的 temporal benchmark 数据**告诉我们：单一时间戳的记忆系统在时间推理上得分 49%，带 validity window 的得 63.8%，差 15 分。CRM 几乎所有问题都是时间问题（"张三现在的 title 是什么"、"上次报价多少"），temporal 必须做
- **Hindsight 的多策略融合**告诉我们：单一向量检索已经过时，semantic + BM25 + entity + temporal 融合得 91.4%。我们不需要上向量，光是 FTS + entity match + temporal 三策略就能拿大部分收益
- **Supermemory 的 selective forgetting**告诉我们：高频被检索的事实一旦过期就 confidently wrong。CRM 数据 stale 得比通用记忆快（半年前的"客户最近很在意预算"今天还成立吗？），decay 机制必须做
- **Mem0/Zep 上 Neo4j 的代价**告诉我们：图数据库让产品复杂到要单独定价。我们不上图，用 SQLite 关系模型 + 多策略 retrieval 已经够 CRM 场景

### 2.3 和 v0.1（SaaS/MCP 版）的关系

v0.1 不是被废弃，是被推迟。完整路径：

- **v1.0 (本文档)**：本地 Skill + CLI，开源免费，OCP 主战场
- **后续 Pro 版本**：可能加托管同步、后台 ingestion、向量增强等。**作为加法，不解锁现有功能**
- **远期 SaaS 版本 (v0.1 复活)**：如果团队/企业市场被验证，可能做。但绝不影响本地版本的完整性

---

## 3. 用户与场景

### 3.1 主用户画像

**Persona A：AI-first solopreneur (主战场)**

- 一人公司，靠产品/服务赚钱
- 工作环境：Claude Desktop / Claude Code / OpenClaw / Hermes / Codex 中的若干个并存
- 客户量级：50-500 个活跃联系人，10-50 个 in-flight 商机
- 痛点：客户信息散在 Gmail、Telegram、Notion；每次跟 Agent 协作要重新粘贴上下文；忘记关键节点

**Persona B：5 人以下 AI-first 小团队 (扩展市场)**

- 共享一个客户池（通过 Git 同步）
- 每人有自己的 Agent，多 Agent 协作场景重要
- 高风险动作（合同/折扣）需要老板确认

**明确不做的用户**：10 人以上销售团队、需要复杂权限和工作流的企业、传统行业。先聚焦再说。

### 3.2 四个核心场景

**场景 1：被动捕获（用户零动作）**

```
用户在 Claude Desktop: "把刚才那封张三的邮件加一下"
Agent → agentcrm contact upsert / activity log
Agent: "已加入张三 (ABC 科技)，并记录了这次邮件互动。"
```

**场景 2：自然语言查询（temporal-aware）**

```
用户: "李四上次报价多少？"
Agent → agentcrm contact search "李四"
Agent → agentcrm deal list --contact <id>
Agent → agentcrm timeline <id> --since 90d --as-of last
Agent: "李四 (XYZ 公司) 当前一个 in-flight 商机 5 万 CNY。
       最近一次报价是 5/10，是分两期付款的修订版。"
```

**场景 3：主动提醒**

```
Agent 启动会话时:
Agent → agentcrm alert scan --since-last
Agent: "早上好。3 件事建议关注：
       1. 王五合同已发 14 天未签
       2. 赵六 Q2 截止日明天
       3. 冯七邮件提到要 demo 但没排时间"
```

**场景 4：多 Agent 协作（v0.3 新增重点）**

```
邮箱 Agent (持续后台): 收到张三邮件
  → AGENTCRM_ACTOR=email-agent agentcrm activity log ...
  → 触发 contact.field_updated 事件

社交 Agent: 看到张三在 Twitter mention
  → AGENTCRM_ACTOR=social-agent agentcrm activity log ...

生日 Agent (每日扫描): 检查今天有谁生日
  → agentcrm contact list --has-field birthday --field "birthday:05-28"
  → 推送提醒

用户主对话 Agent: "今天客户那边有什么动静？"
  → agentcrm events poll --as main-chat --since 24h
  → 综合汇报
```

---

## 4. 数据模型

### 4.0 记忆分层（v0.3 的核心新增）

AgentCRM 借鉴 Letta/Supermemory 的记忆分层架构，把数据明确分成四层。这是理解整个数据模型的钥匙——每层有不同的写入路径、检索时机、生命周期。

|层|在 AgentCRM 中|对应通用记忆概念|性质|生命周期|
|---|---|---|---|---|
|**Static facts (谁)**|`contacts/*.md` frontmatter|static profile|缓慢变化的事实|一年改不了几次|
|**Semantic memory (怎么打交道)**|`memory/contacts/*.md` 等|semantic memory|Agent 蒸馏的稳定经验|几个月内增删|
|**Episodic memory (发生了什么)**|`activities/*.jsonl`|episodic memory|append-only 事件流|永不修改，可归档|
|**Working memory (当前对话)**|不存储|working memory|Agent 自己管|单次会话|

**为什么分层**：混在一起的后果是 prompt 噪音大、推理质量低——Agent 分不清"这是稳定事实"还是"这是某次会议的临时印象"。分层后每种检索操作能精准定位到目标层。

**实际表现**：Agent 准备跟客户沟通时主要读 1+2，做周回顾时主要读 3，没有任何一种场景需要读全部四层。

### 4.1 目录结构（完整版）

```
~/.agentcrm/
├── config.json
├── index.db                    # SQLite 索引，可重建
│
├── contacts/
│   ├── zhang-san.md            # 一个联系人一个文件
│   ├── li-si.md
│   └── _index.json             # 快速查找，可重建
│
├── deals/
│   ├── 2026-q2-abc-tech.md
│   └── _index.json
│
├── activities/
│   ├── 2026-05.jsonl           # 按月分文件，append-only
│   ├── 2026-04.jsonl
│   └── ...
│
├── memory/
│   ├── contacts/
│   │   └── zhang-san.md        # 关于此联系人的长期记忆
│   ├── deals/
│   │   └── 2026-q2-abc-tech.md
│   └── global.md               # 全局偏好
│
├── events/                     # v0.3 新增：事件日志
│   ├── 2026-05.jsonl
│   └── ...
│
├── subscribers/                # v0.3 新增：订阅者 cursor
│   ├── calendar-agent.json
│   ├── email-agent.json
│   └── ...
│
├── proposals/                  # v0.3 新增：待裁决的记忆建议
│   └── pending.jsonl
│
├── alerts/
│   ├── pending.json
│   └── archive.jsonl
│
├── rules/
│   ├── default.yaml
│   └── user.yaml
│
└── .audit/
    └── 2026-05.jsonl
```

### 4.2 Contact 文件格式（带 temporal 历史）

`contacts/zhang-san.md`:

```markdown
---
id: cnt_01HX9...
name: 张三
emails:
  - zhang.san@abc.com
phones:
  - "+86 138 xxxx xxxx"
social:
  twitter: "@zhangsan_xyz"
  linkedin: "linkedin.com/in/zhangsan"
  wechat: "zhangsan_wx"
  github: "zhangsan"

# === 当前值 (Agent 默认读这些) ===
company: ABC 科技
title: CTO
tags: [vip, technical-buyer]

# === 历史 (v0.3 新增，自动维护) ===
company_history:
  - value: XYZ 公司
    from: 2022-01-01
    to: 2024-03-15
    set_by: email-agent
  - value: ABC 科技
    from: 2024-03-15
    to: ~                # ~ 表示当前
    set_by: claude-main

title_history:
  - value: Engineering Manager
    from: 2022-01-01
    to: 2024-03-15
  - value: CTO
    from: 2024-03-15
    to: ~

# === 元数据 ===
source: gmail
birthday: 1985-07-20
anniversaries:
  - date: 2024-03-15
    label: 首次合作
created_at: 2026-03-15T10:23:00Z
updated_at: 2026-05-28T14:00:00Z
last_activity_at: 2026-05-20T09:15:00Z
external_ids: {}
---

# 张三

ABC 科技的技术负责人。负责技术选型决策。

## 背景
通过黄五介绍认识。
```

**设计要点**：

- frontmatter 是机器读的；正文是人和 LLM 都可读
- 关键可变字段（company、title、phone 等）都有对应的 `_history` 字段
- `to: ~` 表示当前生效；非当前的都有明确 from/to
- `set_by` 记录写入这个值的 actor，便于审计

### 4.3 Deal 文件格式

```markdown
---
id: del_01HX9...
contact_ids: [cnt_01HX9...]
title: ABC 科技 Q3 AI 集成项目
stage: proposal
amount: 50000
currency: CNY
expected_close_at: 2026-08-31
owner: self
created_at: 2026-04-01T...
updated_at: 2026-05-20T...
stage_history:
  - stage: lead
    entered_at: 2026-04-01T...
    by: claude-main
  - stage: qualified
    entered_at: 2026-04-15T...
    by: claude-main
  - stage: proposal
    entered_at: 2026-05-10T...
    by: email-agent
amount_history:
  - value: 80000
    from: 2026-05-01
    to: 2026-05-15
    reason: 初版报价
  - value: 50000
    from: 2026-05-15
    to: ~
    reason: 客户预算调整
---

# ABC 科技 Q3 AI 集成项目

## 范围
- 内部知识库 RAG 接入
- 销售 SDR 自动化（二期）

## 进展笔记
- 5/10 发出第一版提案 8 万 CNY
- 5/15 李四反馈预算偏紧
- 5/20 调整为 5 万分两期
```

### 4.4 Activity 日志（按月 JSONL）

`activities/2026-05.jsonl`:

```jsonl
{"id":"act_01HX","ts":"2026-05-20T09:15:00Z","actor":"email-agent","type":"email","direction":"in","channel":"gmail","contact_ids":["cnt_01HX9"],"deal_ids":["del_01HX9"],"subject":"Re: Q3 项目报价","summary":"李四反馈预算紧，要求分阶段","dedupe_key":"<abc@gmail.com>","source_url":null}
{"id":"act_02HX","ts":"2026-05-20T10:30:00Z","actor":"social-agent","type":"social","direction":"in","channel":"twitter","contact_ids":["cnt_01HX9"],"summary":"在 Twitter 询问 demo 可能性","dedupe_key":"tweet:1234567890"}
```

**为什么 JSONL 不是每条一个文件**：activities 一年几千条，每条一文件碎片化严重；JSONL 天然 append-only；grep/jq 可直接处理；按月分文件方便归档。

### 4.5 Memory 文件（带 decay 和 propose 状态）

`memory/contacts/zhang-san.md`:

```markdown
---
scope: contact
scope_id: cnt_01HX9...
last_updated: 2026-05-28T...
---

# 关于张三的记忆

## 沟通偏好 (稳定事实)
- [valid_from: 2025-01-01, decay: never] 偏好邮件 + 偶尔微信
- [valid_from: 2026-04-20, decay: never, supersedes: above] 周五下午可用
- [valid_from: 2026-04-20, decay: never] 周三下午通常空闲

## 决策模式 (稳定事实)
- [valid_from: 2026-03-15, decay: never] 技术细节自己拍，预算回去对齐 CEO 李四
- [valid_from: 2026-03-15, decay: never] 决策周期约 2 周

## 短期观察 (会过期)
- [valid_from: 2026-05-01, decay: 90d] 最近接触竞品 X，正在评估
- [valid_from: 2026-05-15, decay: 30d, expired: 2026-06-14] 这周很忙下周再约

## 历史关键事件
- 2026-04-15 首次正式接洽，对我们开源背景认可
- 2026-05-15 提出分阶段付款诉求
```

**memo 行的标注语法**：

|标记|含义|
|---|---|
|`valid_from:`|生效起始时间|
|`valid_to:`|失效时间（可选，缺省=当前有效）|
|`decay:`|衰减策略：`never` / `30d` / `90d` / `180d` 等|
|`supersedes: above`|替换上一条（保留旧条作为历史）|
|`expired: <date>`|CLI 扫描标记的过期日|

### 4.6 Event 日志（v0.3 新增）

`events/2026-05.jsonl`:

```jsonl
{"ts":"2026-05-28T10:00:00Z","seq":12847,"actor":"email-agent","type":"contact.created","contact_id":"cnt_xxx","fields":{"name":"张三","email":"..."}}
{"ts":"2026-05-28T10:00:01Z","seq":12848,"actor":"email-agent","type":"activity.logged","activity_id":"act_yyy","contact_id":"cnt_xxx","activity_type":"email"}
{"ts":"2026-05-28T10:15:00Z","seq":12849,"actor":"calendar-agent","type":"contact.field_updated","contact_id":"cnt_xxx","field":"birthday","old":null,"new":"1985-07-20"}
{"ts":"2026-05-28T11:00:00Z","seq":12850,"actor":"main-chat","type":"deal.stage_changed","deal_id":"del_xxx","old":"qualified","new":"proposal"}
```

**事件类型清单（v1.0 固定）**：

```
contact.created
contact.updated
contact.field_updated
contact.tagged
contact.merged
contact.deleted

deal.created
deal.updated
deal.stage_changed
deal.amount_changed
deal.closed_won
deal.closed_lost

activity.logged

memory.written
memory.superseded
memory.decayed
memory.forgotten

alert.triggered
alert.dismissed
```

**契约**：事件类型一旦发布，schema 永不变，永不删，只能加新的。

### 4.7 Subscribers cursor

`subscribers/calendar-agent.json`:

```json
{
  "actor": "calendar-agent",
  "last_seq": 12849,
  "last_check_at": "2026-05-28T10:20:00Z",
  "filter_default": "type=contact.created OR (type=contact.field_updated AND field=birthday)"
}
```

每个订阅 Agent 维护自己的 cursor。CLI 帮它读、写、推进。

### 4.8 Proposals（v0.3 新增：矛盾消解机制）

`proposals/pending.jsonl`:

```jsonl
{"id":"prop_01","ts":"2026-05-28T...","actor":"email-agent","scope":"contact:cnt_xxx","statement":"张三的 title 从 CTO 变为 CEO","source_snippet":"...","confidence":0.85,"conflict_with":[{"memo_id":"mem_xxx","text":"张三的 title 是 CTO","valid_from":"2024-03-15"}],"status":"pending"}
```

新陈述与已有记忆冲突时，先写到 proposals 里等 Agent 决策（commit 时选 supersede / keep-both / reject）。

### 4.9 SQLite 索引

```sql
CREATE TABLE contacts (
  id TEXT PRIMARY KEY,
  slug TEXT UNIQUE,
  name TEXT,
  emails TEXT,              -- JSON array
  company TEXT,             -- 当前值（历史在文件里）
  last_activity_at TEXT,
  updated_at TEXT,
  has_birthday INTEGER,
  social_twitter TEXT,      -- 反查用
  social_linkedin TEXT,
  social_wechat TEXT,
  social_github TEXT
);

CREATE TABLE deals (
  id TEXT PRIMARY KEY,
  slug TEXT UNIQUE,
  title TEXT,
  stage TEXT,
  amount INTEGER,
  currency TEXT,
  contact_ids TEXT,         -- JSON array
  expected_close_at TEXT,
  updated_at TEXT
);

CREATE TABLE activities (
  id TEXT PRIMARY KEY,
  ts TEXT,
  type TEXT,
  channel TEXT,
  direction TEXT,
  actor TEXT,
  contact_ids TEXT,
  deal_ids TEXT,
  summary TEXT,
  dedupe_key TEXT UNIQUE
);

CREATE TABLE events (
  seq INTEGER PRIMARY KEY,
  ts TEXT,
  actor TEXT,
  type TEXT,
  payload TEXT
);

CREATE TABLE memos (
  id TEXT PRIMARY KEY,
  scope_type TEXT,
  scope_id TEXT,
  text TEXT,
  valid_from TEXT,
  valid_to TEXT,
  decay_policy TEXT,
  decayed_at TEXT,
  source TEXT,
  actor TEXT
);

CREATE VIRTUAL TABLE contacts_fts USING fts5(id, name, company, body, content='');
CREATE VIRTUAL TABLE activities_fts USING fts5(id, summary, body, content='');
CREATE VIRTUAL TABLE memos_fts USING fts5(id, text, content='');
```

**索引可重建**：`agentcrm reindex` 从所有 markdown/jsonl 文件重建整个 index.db。权威永远是文件。

### 4.10 Config 文件

```json
{
  "version": "1.0.0",
  "data_dir": "~/.agentcrm",
  "user": {
    "name": "张大牛",
    "primary_email": "me@example.com",
    "timezone": "Asia/Shanghai"
  },
  "memory": {
    "default_decay": "180d",
    "preserve_patterns": ["决策*", "偏好*", "* 性格*"]
  },
  "search": {
    "strategies": ["fts", "entity", "temporal"],
    "embedding": {
      "enabled": false,
      "model": "all-MiniLM-L6-v2"
    }
  },
  "alerts": {
    "stale_deal_days": 14,
    "channel": "stdout"
  },
  "privacy": {
    "redact_in_audit": ["phones"]
  }
}
```

---

## 5. CLI 工具设计

### 5.1 选型与设计目标

- **语言**：Go 1.22+（静态二进制、跨平台、启动 < 50ms）
- **存储驱动**：`modernc.org/sqlite`（pure Go，无 CGO）
- **FTS**：SQLite FTS5 内置
- **可选向量**：`sqlite-vec` 扩展（用户在 config 启用）
- **二进制名**：`agentcrm`
- **零运行时依赖**

### 5.2 命令清单（v1.0 完整）

按域分组列出。每个命令都接受 `--format json|text` 切换输出。

#### 5.2.1 联系人

```bash
agentcrm contact upsert \
  --email zhang.san@abc.com \
  --name "张三" \
  --company "ABC 科技" \
  [--phone "..."] [--social.twitter "@..."] \
  [--source gmail] [--tag vip]

agentcrm contact get <id-or-slug> [--as-of <date>] [--format json|text]
agentcrm contact search <query> [--limit 10] [--strategy fts|entity|hybrid]
agentcrm contact list [--tag vip] [--updated-since 7d] [--has-field birthday]
agentcrm contact history <id> --field <name>     # v0.3 新增
agentcrm contact update <id> --set "company=新公司" [--reason "..."]
agentcrm contact merge <id1> <id2>
```

`--field` 路径查询（点路径）：

```bash
agentcrm contact search --field "social.twitter:@johndoe"
agentcrm contact list --has-field "social.linkedin"
agentcrm contact list --field "birthday:05-*"   # 5 月生日的所有人
```

#### 5.2.2 商机

```bash
agentcrm deal create --title "..." --contact <id> --amount 50000 --currency CNY
agentcrm deal update <id> --stage proposal [--reason "..."]
agentcrm deal list [--stage proposal] [--owner self] [--as-of <date>]
agentcrm deal get <id> [--as-of <date>]
agentcrm deal history <id> --field stage|amount
agentcrm deal summarize <id>           # 输出给 Agent 用的自然语言总结
```

#### 5.2.3 活动

```bash
agentcrm activity log \
  --contact <id> \
  --type email|call|meeting|chat|note|social \
  --direction in|out \
  [--channel gmail|twitter|wechat|...] \
  --summary "..." \
  [--body-file path] \
  --dedupe-key "<unique>"

agentcrm activity list --contact <id> [--since 90d] [--type email]
agentcrm timeline <contact-or-deal-id> [--since 90d] [--detail brief|standard|full]
```

#### 5.2.4 记忆（v0.3 重写）

```bash
# 直接写入（旧 API，保留用于明确写入场景）
agentcrm memory write --scope contact:<id> --text "..." \
  [--decay 90d|never] [--valid-from <date>]

# 提议写入（v0.3 新增，推荐路径）
agentcrm memory propose \
  --scope contact:<id> \
  --statement "张三的 title 从 CTO 变为 CEO" \
  --source-snippet "..." \
  --confidence 0.85
# 输出（JSON）：
# {"proposal_id":"prop_01","status":"conflict|clean",
#  "conflict_with":[{"memo_id":"mem_xxx","text":"...","valid_from":"..."}],
#  "suggested_action":"supersede|keep-both|reject"}

agentcrm memory commit --proposal-id <id> --action supersede|keep-both|reject

# 检索
agentcrm memory recall <query> [--scope contact:<id>] [--top-k 5] [--as-of <date>]
agentcrm memory list --scope contact:<id>
agentcrm memory forget <memo-id>

# 衰减
agentcrm memory decay-scan [--dry-run]    # 扫描所有 memo，标记过期
```

#### 5.2.5 提醒与规则

```bash
agentcrm alert scan [--since-last-scan]
agentcrm alert list
agentcrm alert dismiss <id>
agentcrm alert snooze <id> --days 3

agentcrm rule list
agentcrm rule add --from-yaml ./my-rule.yaml
agentcrm rule disable <name>
```

#### 5.2.6 事件订阅（v0.3 新增）

```bash
# 拉自上次 cursor 以来的新事件
agentcrm events poll --as <actor> [--filter "type=..."] [--limit 100]

# 推进 cursor
agentcrm events ack --as <actor> --up-to-seq <N>

# 阻塞式轮询（适合用户起的 long-running 脚本）
agentcrm events watch --as <actor> [--filter "..."] [--interval 60s]

# 列出所有订阅者
agentcrm subscribers list
agentcrm subscribers reset <actor>     # 把 cursor 重置到 0 或指定 seq
```

#### 5.2.7 系统

```bash
agentcrm init [--data-dir ~/.agentcrm]
agentcrm reindex                  # 从文件重建 SQLite 索引
agentcrm doctor                   # 数据完整性检查
agentcrm doctor --fix-conflicts   # Git 合并冲突协助
agentcrm export --format json|csv [--out ./export/]
agentcrm import gmail|vcard|csv ...
agentcrm version
```

### 5.3 Actor 标记（v0.3 重点）

每次 CLI 调用都必须带 actor 标记，三种传入方式：

1. **命令行参数** `--actor <name>`（最高优先级）
2. **环境变量** `AGENTCRM_ACTOR`
3. **缺省** `unknown`（会触发警告）

**推荐 actor 命名规范**：

|Agent 用途|推荐 actor 名|
|---|---|
|主对话 Claude Desktop|`claude-main`|
|Claude Code|`claude-code`|
|邮箱 Agent|`email-agent`|
|社交 Agent|`social-agent`|
|日历/生日 Agent|`calendar-agent`|
|OpenClaw 内的具体 skill|`openclaw:<skill-name>`|
|用户手动|`user`|

Skill 包安装时，setup 步骤会让用户选/填这个名字，写到 Skill 配置里。

### 5.4 CLI 输出约定

**默认输出对 LLM 友好**（简洁、结构化、关键字段标注）：

```bash
$ agentcrm contact get zhang-san
ID: cnt_01HX9
Name: 张三
Company: ABC 科技 (since 2024-03-15)
Title: CTO (since 2024-03-15)
Emails: zhang.san@abc.com
Social: twitter @zhangsan_xyz, linkedin /in/zhangsan
Tags: vip, technical-buyer
Birthday: 1985-07-20

Last activity: 8 days ago (email in)
Active deals: 1 (Q3 AI 集成, proposal, 50,000 CNY)

Memory (semantic):
- 偏好邮件 + 偶尔微信; 周三下午通常空闲
- 决策周期约 2 周; 预算回去对齐李四 (CEO)
- [短期] 最近接触竞品 X (until 2026-08-01)

Recent context (3 most recent):
- 5/20 email-agent: 反馈预算紧，要求分阶段
- 5/15 claude-main: 收到提案
- 5/10 email-agent: 发出提案 v1
```

`--format json` 切换为机器可解析。

### 5.5 写入幂等与并发

- 所有写入用 `O_EXCL` 临时文件 + atomic rename
- `contact upsert` 按 `(tenant, email)` 唯一约束
- `activity log` 用 `--dedupe-key` 去重（重复直接返回原 id 而非错误）
- SQLite WAL 模式支持并发读写
- 多 Agent 并发写不同 contact 文件零冲突；同一 contact 的并发写通过文件锁串行化

### 5.6 性能预算

|操作|目标|用途|
|---|---|---|
|`contact get`, `contact search` (FTS)|< 50ms|Agent 高频读取|
|`contact upsert`, `activity log`|< 100ms|包含索引更新|
|`alert scan`(<1000 deals)|< 200ms|Agent 启动时调一次|
|`timeline` 90 天|< 200ms|准备跟进时调|
|`events poll`|< 50ms|协作 Agent 频繁调|
|`memory propose` (含冲突检测)|< 150ms|Agent 写入路径|
|`memory decay-scan` (全量)|< 1s|后台/cron 每日跑一次|

---

## 6. Skill 包设计

### 6.1 Skill 包结构

```
skills/agentcrm/
├── SKILL.md                    # 主 skill (Claude/OpenClaw)
├── AGENTS.md                   # Codex 编译产物
├── SOUL.md                     # Hermes 编译产物
├── README.md
├── install.sh
│
├── flows/                      # 业务流程
│   ├── new-lead.md
│   ├── log-interaction.md
│   ├── deal-update.md
│   ├── prep-followup.md
│   ├── weekly-review.md
│   ├── who-is-this.md
│   ├── handle-alert.md
│   ├── cleanup-dups.md
│   └── update-fact.md          # v0.3 新增：处理事实变化
│
├── reference/
│   ├── data-model.md
│   ├── cli-reference.md
│   ├── memory-layers.md        # v0.3 新增：记忆分层指南
│   ├── temporal-queries.md     # v0.3 新增：时间查询模式
│   └── common-patterns.md
│
└── bin/
    ├── agentcrm-darwin-arm64
    ├── agentcrm-darwin-amd64
    ├── agentcrm-linux-amd64
    └── agentcrm-windows-amd64.exe
```

### 6.2 主 SKILL.md（顶层入口）

```markdown
---
name: agentcrm
title: AgentCRM - 本地客户记忆
description: |
  AgentCRM 是一个本地化的客户记忆系统。它给你（AI Agent）一个
  跨会话、跨 Agent 的客户大脑。所有数据是用户本机的 markdown 文件，
  通过 `agentcrm` CLI 读写。

  你在以下场景中应该使用它：
  - 用户提到"客户"、"联系人"、"商机"、"跟进"、"记一下"
  - 你识别到任何业务联系人 (邮件、聊天、笔记中)
  - 用户问"X 怎么样了"、"上次和 X 聊什么"
  - 你需要长期记住关于某个客户的事实或偏好

  不要在以下场景使用：
  - 单纯的工程任务
  - 用户明确指定其他 CRM
triggers:
  - 客户 / 联系人 / 商机 / 跟进 / 记一下
  - AgentCRM / agentcrm
  - contact / deal / follow up / customer
---

# AgentCRM Skill

## 你的角色
你正在帮用户管理客户记忆。数据是本机 markdown 文件，通过 `agentcrm` CLI 读写。

## 数据分层（重要）
AgentCRM 把数据分四层。理解分层，写入时放对位置：

1. **Static facts (事实)** → `contacts/*.md` frontmatter
   - 姓名、邮箱、电话、公司、职位、生日等
   - 用 `agentcrm contact upsert / update` 写

2. **Semantic memory (经验)** → `memory/contacts/*.md`
   - "他偏好周三下午沟通"、"决策周期约 2 周"
   - 长期稳定的、关于"如何与此人打交道"的洞察
   - 用 `agentcrm memory propose` 写（不是 write）

3. **Episodic memory (事件)** → `activities/*.jsonl`
   - 一次邮件、一通电话、一场会议
   - 用 `agentcrm activity log` 写

4. **Working memory (临时)** → 你自己的上下文，不入 CRM

## 你的工具
所有操作通过 `agentcrm` CLI。详细参考 @reference/cli-reference.md

## 常用 flow
- 加新联系人 → @flows/new-lead.md
- 记录互动 → @flows/log-interaction.md
- 商机推进/丢失 → @flows/deal-update.md
- 事实变化（换工作、改职位等）→ @flows/update-fact.md
- 准备跟进 → @flows/prep-followup.md
- 周回顾 → @flows/weekly-review.md
- 识别新名字 → @flows/who-is-this.md
- 处理提醒 → @flows/handle-alert.md
- 清理重复 → @flows/cleanup-dups.md

## 核心原则
1. **写入要分层**。事实进 frontmatter，经验进 memory，事件进 activity。三选一不要弄混
2. **事实变化用 propose 不用 write**。当你听到"张三现在是 CEO 了"，先 `memory propose`，让 CLI 帮你检测和旧记忆的冲突
3. **写入带 actor**。你的 actor 名应该是 `<在 install 时配置的名字>`。默认通过环境变量 `AGENTCRM_ACTOR` 传
4. **写入要幂等**。`activity log` 必须带 `--dedupe-key`，重复会被 CLI 自动跳过
5. **不确定不要瞎写**。识别不到关键信息（邮箱、唯一标识），先问用户
6. **失败原样转达**。CLI 返回非零状态时，把错误告诉用户，不要假装成功

## 时间感知（重要）
所有"当前"查询默认返回最新值。要看历史用 `--as-of <date>` 或 `history` 子命令。
当用户问"上次报价多少"、"那时候他在哪家公司"，主动用 temporal 查询。

## 多 Agent 协作
其他 Agent 可能也在写入 AgentCRM。事件日志记录了所有变化。
当用户问"今天发生了什么"或"X 那边有动静吗"，用 `agentcrm events poll`。

## 隐私
所有数据在用户本机。你不要：
- 不要把客户信息发送给任何外部 API（除非用户明确要求）
- 不要在不必要时读取其他客户的文件
- 大量数据导出/删除时，先和用户确认
```

### 6.3 v1.0 完整版必须包含的 9 个 flow

|Flow|触发场景|涉及 CLI|
|---|---|---|
|`new-lead`|识别新人|`contact upsert` + `activity log` + 可能 `memory propose`|
|`log-interaction`|记一次邮件/通话/会议|`contact search` + `activity log` + 可能 `deal update`|
|`deal-update`|商机推进或丢失|`deal update` + `activity log` + `memory propose`|
|`update-fact` (v0.3 新增)|事实变化（换工作、改 title）|`contact get` + `contact update` + `memory propose/commit`|
|`prep-followup`|"我要跟 X 谈"准备|`contact get` + `timeline` + `memory recall` + `deal summarize`|
|`weekly-review`|周回顾|`deal list` + `alert scan` + `events poll` + 汇总|
|`who-is-this`|看到陌生名字|`contact search` + `memory recall`|
|`handle-alert`|处理提醒|`alert scan` + 逐条建议|
|`cleanup-dups`|清理重复|`contact search` + `contact merge`（with confirmation）|

### 6.4 关键 flow 示例：`update-fact.md` (v0.3 新增)

````markdown
# Flow: 处理事实变化

## 何时触发
- 用户/对方提到一个关键事实变化（"我换工作了"、"X 升 CEO 了"、"公司搬到上海了"）
- 你在邮件/聊天里识别到任何当前 contact frontmatter 字段的新值

## 这个 flow 的核心：不要直接覆盖，先 propose

直接 `contact update` 会把旧值归档但**不会**问"这个变化合理吗"。
而 `memory propose` 会触发冲突检测，让你判断如何处理。

## 标准流程

### 1. 识别变化字段
确定是 contact frontmatter 字段（company、title、phone、location 等），
还是 memory 层事实（偏好、习惯）。

### 2. 如果是 frontmatter 字段
```bash
# 先查当前值
agentcrm contact get <id> --format json

# 直接 update（CLI 会自动把旧值归档到 _history）
agentcrm contact update <id> --set "company=新公司" --reason "用户告知 X 时间换工作"
````

### 3. 如果是 memory 层

```bash
# 用 propose 而不是 write，CLI 检测和已有记忆的冲突
agentcrm memory propose \
  --scope contact:<id> \
  --statement "..." \
  --source-snippet "..." \
  --confidence 0.85 \
  --format json
```

CLI 返回：

```json
{
  "proposal_id": "prop_01",
  "status": "conflict",
  "conflict_with": [{"memo_id": "mem_xxx", "text": "...", "valid_from": "2024-03-15"}],
  "suggested_action": "supersede"
}
```

### 4. 决策

|情况|你应该|
|---|---|
|新陈述明确取代旧的（"现在是 X"）|`--action supersede`|
|新陈述补充旧的（"另外他也喜欢 X"）|`--action keep-both`|
|不确定 / 信息不足|`--action reject` + 问用户|

```bash
agentcrm memory commit --proposal-id prop_01 --action supersede
```

### 5. 通知用户

简短确认变化和处理方式：

> "已更新：张三的职位 CTO → CEO（旧值归档到历史，今后查 history 可看到）。"

## 不要做

- 不要在不确定时直接 supersede（容易丢信息）
- 不要把短期观察当稳定事实写入（"他这周很忙"不是 memory，是 activity）
- 不要忘记 actor 标记

```

### 6.5 多 Agent 编译

源 `SKILL.md` + flows + reference → 编译到四个 Agent 平台。脚本走 Go `text/template`，简单拼接。

Claude 和 OpenClaw 共用 `SKILL.md` 主格式，frontmatter 触发词补丁（OpenClaw 加中文）。Codex 编译成单一 `AGENTS.md`。Hermes 给一个轻量 `SOUL.md` + 几条 bootstrap 经验。

### 6.6 `agentcrm-bridge`：给职能 Agent 的轻量 Skill

主对话 Agent 装 `agentcrm` 完整 skill；职能 Agent（email-agent、social-agent、calendar-agent）装一个精简版 `agentcrm-bridge`：

```

skills/agentcrm-bridge/ ├── SKILL.md # 简化版，只讲读写常用模式 ├── reference/ │ └── cli-reference.md # 精简 CLI 命令子集 └── examples/ ├── email-agent-recipes.md ├── social-agent-recipes.md └── calendar-agent-recipes.md

```

`agentcrm-bridge` 不包含完整业务 flow，只包含：
- 怎么 read（contact search、timeline）
- 怎么 write（activity log with dedupe_key）
- 怎么 subscribe（events poll、ack）
- Actor 命名约定

让职能 Agent 的体积尽量小。

---

## 7. 多 Agent 协作机制（v0.3 重点章节）

### 7.1 心智模型：共享黑板

AgentCRM 在用户的 Agent 生态里扮演**共享黑板** (blackboard) 角色：

```

```
   User
    ↕
```

┌───────────────────────────────────────────────────────┐ │ Agent Orchestration Layer │ │ (OpenClaw / Claude / Codex 的编排平台) │ └───┬───────────┬──────────┬──────────┬─────────────────┘ │ │ │ │ ┌───▼──┐ ┌────▼───┐ ┌───▼────┐ ┌───▼─────┐ │ Main │ │ Email │ │ Social │ │Calendar │ │ Chat │ │ Agent │ │ Agent │ │ Agent │ └───┬──┘ └────┬───┘ └───┬────┘ └───┬─────┘ │ │ │ │ │ CLI: read / write / subscribe │ │ │ │ │ ▼ ▼ ▼ ▼ ┌────────────────────────────────────────┐ │ AgentCRM (本地文件) │ │ contacts/ deals/ activities/ │ │ memory/ events/ subscribers/ │ └────────────────────────────────────────┘ ↑ └── 用户可以直接打开文件看/改/git push

````

**关键设计**：Agent 之间不直接通信。它们通过读写同一片文件协作。AgentCRM 不发指令、不接指令，它只在那儿存着。

### 7.2 三种协作角色

| 角色 | 含义 | 例子 |
|---|---|---|
| **Reader** | 其他 Agent 读取客户上下文 | 邮箱 Agent 回邮件前查"这人是谁" |
| **Writer** | 其他 Agent 写入客户事件 | 邮箱 Agent 发完邮件记 activity |
| **Subscriber** | 其他 Agent 订阅变化 | 生日 Agent 订阅"新联系人加入" |

### 7.3 Reader 模式

任何 Agent 跑 shell 都能读：

```bash
$ agentcrm contact get --email zhang.san@abc.com --format json
$ agentcrm timeline cnt_xxx --since 30d --detail brief
$ agentcrm memory recall "张三的沟通偏好" --scope contact:cnt_xxx
````

无需协议、无需 SDK。

### 7.4 Writer 模式

带 actor 标记 + dedupe_key：

```bash
AGENTCRM_ACTOR=email-agent agentcrm activity log \
  --contact cnt_xxx \
  --type email \
  --direction in \
  --channel gmail \
  --summary "..." \
  --dedupe-key "<message-id@gmail.com>"
```

Dedupe key 来源约定：

|事件|dedupe_key|
|---|---|
|邮件|RFC822 message-id|
|日历事件|iCal UID|
|Twitter|`tweet:<id>`|
|微信|`wechat:<chat_id>:<msg_id>`|
|电话|recording UUID|

### 7.5 Subscriber 模式

用 cursor + filter 拉新事件：

```bash
# 拉新事件
agentcrm events poll --as calendar-agent \
  --filter "type=contact.created OR (type=contact.field_updated AND field=birthday)" \
  --format json

# 处理完成后推进 cursor
agentcrm events ack --as calendar-agent --up-to-seq 12849

# 阻塞式 watch（用户起 long-running 脚本时用）
agentcrm events watch --as social-agent --filter "type=activity.logged AND channel=twitter"
```

**关键**：`watch` 是用户起的进程，**不是 AgentCRM 的 daemon**。AgentCRM 仍然零后台进程。

### 7.6 防循环

CLI 默认在 `events poll` 时**排除调用者自己产生的事件**：

```bash
# calendar-agent 拉事件时，默认不会拿到 actor=calendar-agent 的事件
agentcrm events poll --as calendar-agent

# 要拿回自己的也行：
agentcrm events poll --as calendar-agent --include-self
```

这避免了 calendar-agent 写了 birthday → 触发 field_updated → calendar-agent 自己又看到 → 循环。

### 7.7 协作场景示例

**邮箱 Agent 收新邮件**：

```
1. 收到 zhang.san@abc.com 的邮件
2. agentcrm contact get --email zhang.san@abc.com --format json
   → 拿到 contact_id 和当前画像
3. agentcrm timeline <id> --since 30d --detail brief
   → 拿到最近上下文
4. 起草回复（邮箱 Agent 自己的事）
5. 用户审阅 → 发送
6. AGENTCRM_ACTOR=email-agent agentcrm activity log \
     --contact <id> --type email --direction out \
     --summary "..." --dedupe-key "<sent-msg-id>"
```

**社交 Agent 监测 Twitter mention**：

```
1. 看到 @you 的 mention 来自 @johndoe_xyz
2. agentcrm contact search --field "social.twitter:@johndoe_xyz" --format json
   情况 A: 找到 → 用这个 contact_id
   情况 B: 没找到，bio 看起来业务相关 → contact upsert
   情况 C: 没找到，bio 不相关 → 不入 CRM
3. AGENTCRM_ACTOR=social-agent agentcrm activity log \
     --contact <id> --type social --channel twitter \
     --direction in --summary "..." \
     --dedupe-key "tweet:<id>"
```

**生日 Agent 每日扫描**：

```
1. （cron 起，每天早上）
2. agentcrm events poll --as calendar-agent \
     --filter "type=contact.created OR field=birthday"
3. 对新联系人：尝试在 LinkedIn 等查 birthday
   找到 → AGENTCRM_ACTOR=calendar-agent agentcrm contact update \
            <id> --set "birthday=..."
4. 今天的生日：agentcrm contact list --field "birthday:05-28"
   → 推送提醒给用户主对话 Agent
5. agentcrm events ack --as calendar-agent --up-to-seq <N>
```

---

## 8. 记忆系统能力（v0.3 重点章节）

借鉴自 Mem0/Supermemory/Zep/Hindsight 的五项关键能力，本地化实现。

### 8.1 记忆分层（已在 §4.0 详述）

四层分明：static / semantic / episodic / working。每层独立存储和检索。

### 8.2 Temporal validity

**字段历史**：contact/deal 的关键可变字段自动保留 `_history`：

```yaml
title: CTO
title_history:
  - {value: Engineer, from: 2022-01-01, to: 2024-03-15}
  - {value: CTO, from: 2024-03-15, to: ~}
```

**Memo validity**：每条 memo 带 `valid_from / valid_to / decay`：

```
- [valid_from: 2026-04-20, decay: never] 周五下午可用
- [valid_from: 2026-05-01, decay: 90d] 最近接触竞品 X
```

**时间查询**：

```bash
agentcrm contact get cnt_xxx --as-of 2024-06-01    # 那时候的样子
agentcrm contact history cnt_xxx --field company    # 历史变化
agentcrm deal get del_xxx --as-of 2026-05-01       # 那时候 deal 状态
agentcrm timeline cnt_xxx --as-of 2026-05-01       # 那时候为止的时间线
```

### 8.3 Propose / Commit 矛盾消解

新事实写入走 propose 而非 write：

```bash
agentcrm memory propose \
  --scope contact:<id> \
  --statement "张三的 title 从 CTO 变为 CEO" \
  --confidence 0.85
```

CLI 内部检测和现有 memo 的冲突，返回结构化结果。Agent 根据结果选择：

- `supersede`：新值取代，旧值归档为 history
- `keep-both`：保留两条（两者都成立或互补）
- `reject`：撤销 proposal（不确定时）

**CLI 不替 Agent 做推理决策**——它只检测冲突，决策权在 Skill 指导的 Agent 手里。

### 8.4 多策略 hybrid retrieval

`contact search` / `memory recall` 默认融合三策略：

```
score = w1·fts_score + w2·entity_match_boost + w3·recency_boost
```

- **FTS**：SQLite FTS5 关键词
- **Entity**：query 里的人名/公司名直接精确匹配
- **Temporal**：query 里有时间词（"最近"、"上周"），过滤候选

三策略零额外依赖，全用 SQLite。

**可选第四策略：semantic embedding**（用户在 config 启用）：

```json
"search": {
  "embedding": {
    "enabled": true,
    "model": "all-MiniLM-L6-v2"
  }
}
```

启用后跑本机小模型（~80MB），用 `sqlite-vec` 存向量。`contact search` 自动加入第四策略融合。

不启用时降级到三策略，仍然可用。

### 8.5 Selective forgetting

每条 memo 可标 `decay`：

```
- [valid_from: 2026-05-01, decay: 30d] 这周很忙
- [valid_from: 2026-04-15, decay: 90d] 最近评估竞品 X
- [valid_from: 2025-01-01, decay: never] 决策周期约 2 周
```

CLI 提供：

```bash
agentcrm memory decay-scan [--dry-run]
# 扫描所有 memo
# 把 ts + decay 过期的 memo 标记 expired
# expired memo 默认不进 recall 结果，可用 --include-expired 显式拉
```

**Skill 指引**：

|类型|推荐 decay|
|---|---|
|性格、决策模式、长期偏好|`never`|
|业务节奏（"决策周期 2 周"）|`never`|
|当前关心的事（"这季度重点 X"）|`180d`|
|短期状态（"最近接触竞品 X"）|`90d`|
|临时安排（"这周很忙"）|`30d`|

用户/cron 定期跑 `memory decay-scan` 让过期 memo 自动失效。

### 8.6 反向选择：为什么不上图数据库

借鉴自通用记忆系统的弯路：

- Mem0/Zep/Cognee 上 Neo4j 让产品复杂到要单独定价（Mem0 graph 收 $249/月）
- Hindsight 用四路并行 + cross-encoder reranking 跑 91.4%，**不靠图**
- CRM 数据天然层次清晰（contact → activity / deal），关系比通用知识图谱简单
- SQLite JOIN + 多策略融合在 CRM 场景已经够用

**明确决策**：v1.0 不引入图数据库。如果用户真有"和张三的同事们最近聊得怎样"这种 multi-hop 需求，留到 Pro 或社区扩展。

---

## 9. 提醒与规则

### 9.1 触发方式（两种）

**方式 A：Agent 启动会话主动扫描**

每个 flow 启动前，Skill 引导 Agent：

```bash
agentcrm alert scan --since-last-scan
```

CLI 内部判断距上次扫描 > 6 小时才实际跑。

**方式 B：用户 cron**

```bash
agentcrm cron install --daily 9am
```

加一行到 crontab，每天扫一次，结果写 `alerts/pending.json`。Agent 下次会话读这个。

**不做 daemon**。

### 9.2 默认规则

`rules/default.yaml`:

```yaml
rules:
  - name: stale_deal
    when:
      object: deal
      stage_in: [qualified, proposal, negotiation]
      last_activity_age_days: ">14"
    then:
      alert:
        title: "{deal.title} 已 {age_days} 天无互动"
        suggestion: "建议本周内跟进 {primary_contact.name}"

  - name: closing_deadline
    when:
      object: deal
      expected_close_in_days: "<7"
      stage_not_in: [won, lost]
    then:
      alert:
        title: "{deal.title} 预计 {days_left} 天后成交"

  - name: vip_silence
    when:
      object: contact
      tag_includes: vip
      last_activity_age_days: ">30"
    then:
      alert:
        title: "VIP 客户 {contact.name} 30 天无互动"

  # v0.3 新增：用记忆系统的能力
  - name: stale_memory
    when:
      object: contact
      has_expired_memos: true
    then:
      alert:
        title: "{contact.name} 有过期记忆需要更新"
```

### 9.3 自定义规则

用户对话里说"金额超 10 万 deal 超过 7 天没动就提醒"，Agent 走一个 sub-skill 翻译成 YAML 追加到 `rules/user.yaml`。用户可直接编辑文件。

---

## 10. 跨设备同步

不做同步功能，做到 **Git-friendly**：

- 每个 contact/deal 一个文件 → 不同设备改不同人不冲突
- Activity / events JSONL append-only → 行级 merge 不冲突
- SQLite 索引不进 Git（`.gitignore` 默认含 `index.db`）
- Pull 后自动 `agentcrm reindex`

**真冲突处理**：

```bash
agentcrm doctor --fix-conflicts
# 检测 <<<<<<< marker，引导用户用 Agent merge
```

**ULID 保证 ID 跨设备唯一**：两个设备同时创建"张三"会有两个文件，需后续 merge——这是合理的，且很少发生。

---

## 11. Ingestion

v1.0 完全靠 Agent 驱动，不做后台监听。理由同 v0.2：违反"不要进程"原则，且 Agent 触发的 ingestion 已覆盖 95% 场景。

提供一次性导入工具：

```bash
agentcrm import gmail --since 2026-01-01 --dry-run
agentcrm import vcard ./contacts.vcf
agentcrm import csv ./hubspot-export.csv --mapping ./mapping.json
```

后台 Gmail watcher 留给 Pro 版本作为收费功能。

---

## 12. 安全与隐私

### 12.1 数据完全本地

`~/.agentcrm/` 永远不主动传到任何远端。除非：

- 用户自配 Git remote
- 用户 Agent 在处理 skill 时把数据片段发给 LLM（这是 Agent 自己的事）
- 用户主动 `agentcrm export`

### 12.2 Agent 调用 LLM 的隐私

Skill 设计约束 Agent："只发必要字段给 LLM"。例如 `prep-followup` 用 `timeline --detail brief` 输出摘要再喂 LLM，不喂全量。

### 12.3 审计

CLI 每次写入记录到 `.audit/YYYY-MM.jsonl`：

```jsonl
{"ts":"...","actor":"email-agent","action":"contact.upsert","args":{"email":"<redacted>"},"result":"ok","contact_id":"cnt_xxx"}
```

Actor 字段保证可追溯——用户能回看"哪个 Agent 在什么时候做了什么"。

### 12.4 零遥测

v1.0 完全零遥测。任何采集必须 opt-in 且文档化。

---

## 13. 技术栈与项目结构

|部分|选型|
|---|---|
|CLI|Go 1.22+|
|SQLite|`modernc.org/sqlite`（pure Go）|
|FTS|SQLite FTS5|
|可选向量|`sqlite-vec` 扩展 + ONNX runtime 本地小模型|
|中文分词|`gojieba`（评估体积成本后定）|
|Markdown|`goldmark`|
|模板|`text/template`|
|测试|Go `testing` + `testify`|

**显式拒绝**：Node、Python、Docker、ORM、Web 框架、daemon 进程。

### 13.1 仓库结构

```
agentcrm/
├── cmd/agentcrm/               # CLI 主入口
├── internal/
│   ├── store/                  # 文件读写 + SQLite
│   ├── model/                  # Contact/Deal/Activity/Memo/Event
│   ├── search/                 # 多策略融合
│   ├── memory/                 # propose / commit / decay
│   ├── events/                 # 事件日志 + cursor
│   ├── alert/                  # 规则引擎
│   └── audit/
├── skills/
│   ├── agentcrm/               # 主 skill
│   ├── agentcrm-bridge/        # 协作 skill
│   └── examples/               # 职能 agent 模板
├── compile/                    # Skill 多平台编译
├── install.sh
├── Makefile
├── go.mod
└── README.md
```

### 13.2 发布产物

每次 release：

- 四平台 binary
- 编译好的 skill 包（按目标 Agent 分）
- universal 包（含所有 binary + 所有 skill）

`install.sh` 检测平台和已装 Agent，下对应包放对位置。

---

## 14. 开源与未来商业化

### 14.1 v1.0 完全免费 + 开源

- 许可证：MIT 或 Apache 2.0（法务定）
- **不分社区版/企业版**。完整功能开源

### 14.2 未来 Pro 可能方向（不影响 v1.0）

- **托管同步服务**（替代 Git/iCloud 自管）
- **后台 Gmail/Outlook watcher**（v1.0 没有的全自动 ingestion）
- **多人协作 + 权限**
- **向量检索增强**（更强本地模型 + 远程 embedding 选项）
- **图增强**（如果真有用户场景需要 multi-hop）
- **第三方 CRM 双向同步**（HubSpot / Salesforce / Attio）
- **企业部署 + auth/RBAC**

**纪律**：v1.0 不为了商业化故意阉割。Pro 是加法，不是解锁。

### 14.3 项目治理

- 公开 GitHub 仓库
- 标准 Issue + PR 流程
- Flow 社区可贡献。维护 `awesome-agentcrm-flows` 仓库
- 文档全开源

---

## 15. v1.0 完整版交付清单

**注意**：这里列的是 v1.0 完整版的全部范围，**不是 MVP**。每项 release 时都要齐。

### 15.1 CLI

- [ ] 全部 §5.2 命令实现
- [ ] 四平台 binary
- [ ] FTS5 中英文搜索
- [ ] 多策略融合 retrieval（FTS + entity + temporal，可选 embedding）
- [ ] Propose / commit 矛盾消解
- [ ] Decay 扫描
- [ ] Field history 自动维护
- [ ] As-of temporal 查询
- [ ] 事件日志 + cursor 订阅
- [ ] Actor 标记强制
- [ ] 数据完整性检查 `doctor`
- [ ] 导入导出
- [ ] 性能预算达标（§5.6）
- [ ] 单元测试覆盖率 > 70%

### 15.2 Skill 包

- [ ] 主 SKILL.md
- [ ] 全部 9 个 flow
- [ ] reference/ 完整（含 memory-layers、temporal-queries）
- [ ] 四平台 Agent 编译
- [ ] `agentcrm-bridge` 精简版
- [ ] 3 个职能 Agent 示例模板（email/social/calendar）
- [ ] 每个 flow 的真实 LLM 评测

### 15.3 数据模型

- [ ] 全部四层（static / semantic / episodic / working）
- [ ] Contact/Deal frontmatter + _history
- [ ] Memory + valid/decay
- [ ] Activity JSONL
- [ ] Event JSONL + Subscribers cursor
- [ ] Proposals JSONL
- [ ] Markdown 格式稳定，文档化
- [ ] SQLite 索引可重建
- [ ] Git-friendly 验证（多设备模拟）

### 15.4 规则与提醒

- [ ] 默认四条规则（stale_deal、closing_deadline、vip_silence、stale_memory）
- [ ] 用户自定义 YAML 规则
- [ ] `alert scan` 性能达标

### 15.5 多 Agent 协作

- [ ] Actor 标记机制
- [ ] 事件订阅 poll/ack/watch
- [ ] 防循环过滤
- [ ] 三个协作场景跑通真实测试（email-agent / social-agent / calendar-agent）

### 15.6 安装与文档

- [ ] 一行命令安装脚本
- [ ] 各 Agent 平台 README
- [ ] Getting Started + 进阶用户文档
- [ ] 开发者贡献指南（flow / CLI）
- [ ] 记忆系统理论说明（让用户理解四层和 temporal）

### 15.7 发布渠道

- [ ] GitHub Release 自动化
- [ ] 提交 Claude Skill Marketplace
- [ ] PR 到 OpenClaw `awesome-skills`
- [ ] 自有 `awesome-agentcrm-flows` 仓库
- [ ] 博客文章 + Demo 视频

---

## 16. 工程纪律

写给每个 commit 的开发者：

1. **每个 PR 必须能跑 `make install && agentcrm init`**。安装链路不能断
2. **数据格式变更必须有迁移脚本**。我们对用户数据有契约责任
3. **CLI 输出格式是 API 的一部分**。Skill 依赖它，改要走 v 升级
4. **新增 flow 必须真实 LLM 评测**。不是单元测试，是用 Claude/OpenClaw 真跑场景，看 Agent 选不选得到、调用对不对
5. **事件类型不变**。事件 schema 一旦发布，永不删，永不改语义，只能加新的
6. **不引入 daemon、Node、Python、Docker**。任何破例需要架构 review
7. **不收集用户数据**。任何遥测必须 opt-in 且文档化

---

## 17. 开放问题

留给团队讨论：

1. **Embedding 默认状态**：默认关闭（保持轻量）还是默认开启（更强 retrieval，但二进制 +80MB）？倾向默认关闭，README 强调可一键开启
2. **History 全字段开启**：所有 frontmatter 字段都自动归档，还是只对显式 mark 的字段？倾向白名单：company / title / phone / location / amount 等关键字段强制 history，其他字段直接覆盖
3. **Decay 默认策略**：硬编码 180d，还是用户必须自己配？倾向硬编码默认 180d 且文档化"会过期"，用户可改
4. **中文分词体积**：gojieba 词典 ~10MB 还是接受？还是 unigram？倾向默认 unigram（够 FTS 用），让用户在 config 启用 gojieba（按需下载词典）
5. **事件日志留存**：默认永久 vs 自动归档？倾向永久，提供手动归档命令
6. **Actor 唯一性**：多设备情况下，每个设备应该有不同 actor 名（"calendar-agent-mac"、"calendar-agent-laptop"），还是同名？倾向同名 + device tag 字段
7. **`memory write` 是否保留**：propose 是推荐路径，write 是否还应该存在（明确写入场景）？倾向保留但文档强调 propose 优先

---

## 附录 A：对外定位

**给开发者**：

> AgentCRM 是一个本地化的开源 Skill 包 + CLI。装上之后，你的 Claude / OpenClaw / Codex / Hermes 就有了一个共享的客户记忆——所有数据是你 home 目录下的 markdown，能 git 同步、能用任何编辑器看、能随时离开。它借鉴了 Mem0 / Supermemory / Zep / Hindsight 的记忆系统设计，但聚焦在客户关系这一个垂直领域。

**给终端用户**：

> 给你每一个 AI 助手装一个共享的客户大脑。它们一起记得每个客户，自动整理来往，关键事情会提醒你。所有数据存在你电脑上的文件夹，永远是你的。

**给 hacker news / GitHub**：

> Your AI agents' shared CRM brain, in your filesystem. Local-first, file-based, zero servers. Inspired by Mem0/Supermemory but specialized for customer relationships. Install in one line. Works with Claude, OpenClaw, Codex, Hermes.

**给投资人（远期）**：

> 通用记忆系统（Mem0、Supermemory）在卖通用 SDK 给开发者。我们在做领域特化的垂直版本——一个能直接被终端用户的 Agent 装上就用、不需要写代码的客户记忆。开源版本是产品入口和分发抓手，Pro 版本服务那些愿意为省事和增强能力付费的用户。

---

**文档结束**。建议第一周 review 通过后开始拆任务，优先级：

1. CLI 骨架 + 数据模型 + contact/activity/memory 三大命令
2. propose/commit + decay + temporal 三大记忆能力
3. events poll/ack + actor 标记 多 Agent 协作
4. Skill 包 + 9 个 flow + Claude Code 真实场景跑通
5. 其他三个 Agent 平台编译 + 职能 Agent 示例
6. 安装脚本 + 文档 + 发布渠道

---

写完了。v0.3 相对 v0.2 的核心变化：

- **定位升级**：从"轻量 CRM"变成"客户领域记忆系统"，理论根基拉高，故事更清晰
- **记忆分层架构**（§4.0）成为理解所有数据模型的钥匙
- **Temporal validity** 渗透到 contact/deal frontmatter (`_history`) 和 memory (`valid_from/decay`)，CLI 加 `--as-of` / `history` 子命令
- **Propose/commit 矛盾消解**（§8.3）让 Agent 不会粗暴覆盖事实
- **Selective forgetting**（§8.5）让 CRM 在长期使用后比传统 CRM 数据质量高一个数量级
- **多 Agent 协作**完整章节（§7）：actor 标记 + 事件订阅 + 防循环
- **多策略 retrieval**（§8.4）：FTS + entity + temporal 默认，embedding 可选
- **新增第 9 个 flow** `update-fact.md` 教 Agent 用 propose 流程
- **`agentcrm-bridge`** 给职能 Agent 的精简 skill 包

