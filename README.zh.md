<div align="center">

# AgentCRM

**OPC 时代的客户大脑 · 就在你的 Home 目录里**

[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Skill Format](https://img.shields.io/badge/skill-Claude%20%7C%20OpenClaw%20%7C%20Codex%20%7C%20Hermes-purple)](#支持的-agent)

让你的 Claude / OpenClaw / Codex / Hermes Agent 共享一个客户大脑。
零服务进程 · 本地文件 · 一行安装。

[快速开始](#快速开始) · [它是什么](#它是什么) · [为什么需要](#为什么需要-agentcrm) · [文档](docs/) · [English](README.md)

</div>

---

## 它是什么

AgentCRM 是一个**本地化、文件存储、零服务进程**的客户关系记忆系统。以 Skill 包 + CLI 二进制双形态分发，让你的每个 AI Agent 立刻拥有一个**持久、可读、可备份、跨 Agent 共享的客户大脑**。

它不是云 CRM。不是 SaaS。**它是你 home 目录下的一个文件夹：**

```
~/AgentCRM/
├── config.json        # 用户配置
├── contacts/          # 每个联系人一个 Markdown 文件
├── deals/             # 每个商机一个 Markdown 文件
├── activities/        # 按月归档的 JSONL 交互记录
├── memory/            # 长期记忆
│   ├── contacts/      #   联系人的记忆文件
│   └── deals/         #   商机的记忆文件
├── events/            # 多 Agent 协作事件流
├── rules/             # YAML 规则配置（用户自定义）
├── alerts/            # 规则引擎产生的待处理提醒
├── proposals/         # 待裁决的记忆提案
├── .index.db          # SQLite 搜索索引（可从文件重建）
├── .subscribers/      # 各 Actor 的事件订阅游标
└── .audit/            # 操作审计日志
```

加上安装到 Agent Skill 目录的 `agentcrm` Skill 包，和一个 `agentcrm` CLI 二进制。**就这些。**

数据永远在你机器上——任意编辑器打开、git 同步、随时离开。

---

## 为什么需要 AgentCRM

### 工作没变，干活的人变了

客户工作曾经由公司里的人完成——销售代表填工单、客户经理跟单、支持人员记录问题。**在 OPC（一人公司）时代，这些工作交给了 Agent：**

| 任务 | 以前 | 现在 |
|---|---|---|
| 添加新联系人 | 销售助理填表单 | 邮件 Agent 自动捕获 |
| 记录交互 | 销售员手动输入 | 多个 Agent 后台写入 |
| 跟踪变更 | 客户经理跟进 | 监控 Agent 推送提醒 |
| 准备跟进 | 销售员读历史 | Agent 合成上下文 |

但今天的工具假设错了：

- **传统 CRM**（Salesforce / HubSpot / Attio）假设"你有一个团队，他们填表单、看仪表盘"——OPC 时代这些都不成立
- **AI 增强 CRM** 加了个聊天侧栏和自动填写——同样的架构，仍然是"人服务系统"
- **通用 Agent 记忆**（Mem0 / Supermemory）不懂客户关系业务——它能存"张三是 CTO"，但不知道丢掉一个商机该怎么处理

### AgentCRM 的位置

```
┌─ 通用记忆系统 (Mem0 / Supermemory) ────────────┐
│  存储任何事实。不懂 CRM 工作流。                │
└─────────────────────────────────────────────────┘

┌─ 传统 CRM (Salesforce / HubSpot / Attio) ──────┐
│  懂 CRM。假设人填表单。                         │
└─────────────────────────────────────────────────┘

┌─ AgentCRM ─────────────────────────────────────┐
│  懂 CRM + 为 Agent 设计 + 本地文件              │
└─────────────────────────────────────────────────┘
```

**一句话：** AgentCRM 就是把通用记忆系统（Mem0/Supermemory）做了客户关系领域的专业化，并以本地 Skill 包而非云服务的形式交付。

### 三个其他地方得不到的能力

**1. 跨 Agent 共享客户大脑**

你的邮件 Agent、社交监听 Agent、生日 Agent、主聊天 Agent 都在读写同一份客户数据。一个 Agent 写入，所有 Agent 立即看见。

**2. 内置商业语法**

九个标准化业务流——`new-lead`、`deal-update`、`prep-followup` 等——教会你的 Agent **事情的先后顺序**，而不只是"怎么调用 API"。

**3. 100% 数据所有权**

文件即真相。在 Obsidian 中打开、grep 搜索、git 同步。没有可移植性问题，因为数据从未离开过你的机器。

---

## 对比

|  | AgentCRM | 传统 CRM | 通用记忆系统 |
|---|---|---|---|
| 主要用户 | AI Agent | 人类 | AI Agent |
| 部署方式 | 本地文件 + CLI | 云 SaaS | 云 SaaS / 自建 |
| 存储格式 | Markdown + JSONL | 专有数据库 | 专有数据库 + 向量 |
| 领域知识 | 9 个 CRM 流内置 | UI 工作流 | 无（通用增删查） |
| 多 Agent 共享 | 原生支持 | 否 | 单 Agent 范围 |
| 数据所有权 | 你的文件 | 供应商 | 供应商 |
| 时间感知 | 字段级历史 | 基本时间戳 | 持续改进中 |
| 月费 | ￥0 | ￥200–2000/席位 | $19–249 |

---

## 快速开始

### 一行安装

```bash
curl -fsSL https://raw.githubusercontent.com/AgentPal/AgentCRM/main/install.sh | sh
```

安装程序将下载 `agentcrm` CLI 二进制到 `/usr/local/bin/`。

然后初始化：

```bash
agentcrm init
```

### 从源码构建

```bash
git clone https://github.com/AgentPal/AgentCRM.git
cd AgentCRM
go install ./cmd/agentcrm
```

需要 Go 1.22+。

### 验证

```bash
$ agentcrm version
agentcrm v1.0.0-dev

$ agentcrm doctor
AgentCRM data integrity check
==============================
  Data dir: ~/AgentCRM
  Database: ~/AgentCRM/.index.db
OK files: pass
OK orphans: pass
OK dedupe: pass
OK memos: pass

PASS: all checks passed
```

### 开始使用

打开你的任一 AI Agent（Claude、OpenClaw、Codex 等），对它说：

> 用 agentcrm 帮我开始

Agent 会调用 AgentCRM Skill 并引导你完成。你不需要学任何命令——Skill 教会 Agent 如何做一切。

---

## 使用场景

安装之后，**你不再需要"打开 CRM"**。只需要继续跟你已有的 Agent 说话。

### 场景 1：自动捕获新联系人

```
你：把刚才收到的张三的邮件加到 CRM
Agent：✓ 已添加张三（ABC科技，CTO），并记录了邮件交互。
        他提到 Q3 想做 AI 集成——要加到他的长期笔记里吗？
你：好的
Agent：✓ 已记录："Q3 想做 AI 集成，他是主要推动者"
```

### 场景 2：自然语言查询

```
你：我们上次给李四的报价是多少？
Agent：李四（XYZ 公司）有一个进行中的商机，金额 50,000 元。
        最近一次是 5/10 的修订版——拆成了两期付款。
        要看完整时间线吗？
```

### 场景 3：主动简报

```
（Agent 打开会话时）
Agent：早上好。今天有三件事需要关注：
       1. 王五的合同已发送 14 天，仍未签署（VIP 客户）
       2. 赵六的 Q2 截止日期是明天
       3. 冯七提到想要演示但还没约时间
       
       你想从哪个开始？
```

### 场景 4：多 Agent 协作

```
（夜间，邮件 Agent 捕获新邮件）
邮件 Agent → agentcrm activity log --actor email-agent ...

（早上，生日 Agent 检查今天的生日）
生日 Agent → agentcrm contact list --field "birthday:05-28"
          → 推送提醒

（你打开 Claude）
你：今天客户有什么情况吗？
Agent：邮件 Agent 昨晚捕获了 3 封新邮件（关联到 2 个联系人）。
       今天是张三的生日（ABC 科技）。
       要我起草一份简短的生日问候吗？
```

### 场景 5：直接打开文件

数据是你的。打开 `~/AgentCRM/contacts/zhang-san.md`：

```markdown
---
id: cnt_01HX9...
name: 张三
emails: [zhang.san@abc.com]
company: ABC 科技
title: CTO
tags: [vip, technical-buyer]
last_activity_at: 2026-05-20T09:15:00Z
---

# 张三

ABC 科技 CTO。技术决策者。

## 背景
由黄五介绍。Q3 AI 集成项目的推动者。
```

在 Obsidian、VSCode、Vim 中打开——任何编辑器都行。`cd ~/AgentCRM && git init` 跨设备同步。`grep -r "Q3" .` 全文搜索。

---

## 工作原理

### 三层架构

```
┌─────────────────────────────────────────────────┐
│  Skill 层（领域知识）                            │
│  9 个业务流: new-lead / deal-update / ...        │
│  教会 Agent"在场景 X 下，按 Y 顺序做 Z"        │
└─────────────────────────────────────────────────┘
                       ↕
┌─────────────────────────────────────────────────┐
│  CLI 层（原子能力）                              │
│  agentcrm contact / deal / activity / memory ...│
│  无依赖 Go 二进制，启动 <50ms                    │
└─────────────────────────────────────────────────┘
                       ↕
┌─────────────────────────────────────────────────┐
│  数据层（你的文件）                              │
│  Markdown + JSONL + SQLite 索引（可重建）        │
│  只有一个文件夹: ~/AgentCRM/                   │
└─────────────────────────────────────────────────┘
```

### 记忆系统设计

受通用 Agent 记忆系统启发，为客户关系领域做了本地化：

- **四层记忆架构**：静态事实 / 语义记忆 / 情景记忆 / 工作记忆
- **时间有效性**：每个可变字段带 `_history`，回答"2024 年他在哪家公司？"
- **Propose/Commit**：新事实写入前自动冲突检测，Agent 可选 `supersede` / `keep-both` / `reject`
- **多策略检索**：FTS5 关键词 + 实体匹配 + 时间过滤
- **选择性遗忘**：每条记忆可设 decay，过期自动归档

---

## 支持的 Agent

| Agent | Skill 格式 | 状态 |
|---|---|---|
| Claude Desktop / Claude Code | `SKILL.md` | ✅ v1.0 |
| OpenClaw | `SKILL.md`（中文触发） | ✅ v1.0 |
| Codex | `AGENTS.md` | ✅ v1.0 |
| Hermes | `SOUL.md` | ✅ v1.0 |
| 其他 Skill / MCP 兼容 Agent | 适配中 | 🚧 |

CLI 是通用的——任何能执行 shell 命令的 Agent 都能使用。Skill 只是教会 Agent **怎么用好它**。

---

## 文档

- [快速入门](GETTING_STARTED.md)
- [Skill 包](skills/agentcrm/SKILL.md)
- [九个业务流](skills/agentcrm/flows/)
- [CLI 参考](internal/cmd/root.go)
- [数据模型](internal/model/)
- [设计文档](docs/)

---

## 常见问题

<details>
<summary><b>需要联网吗？</b></summary>

CLI 本身完全离线。你的 Agent（Claude / OpenClaw 等）调用 LLM 时需要联网——那是 Agent 和它的提供商之间的事。AgentCRM 不参与。

</details>

<details>
<summary><b>数据会发送到你们的服务器吗？</b></summary>

不会。AgentCRM 没有服务器，没有遥测。100% 的数据在你的机器上。除非你自己配置了 git remote，否则不会有一个字节离开你的设备。

</details>

<details>
<summary><b>怎么跨设备同步？</b></summary>

我们不提供同步服务，但一切对 git 友好：

```bash
cd ~/AgentCRM
git init
git remote add origin git@github.com:you/my-crm-data.git
git add . && git commit -m "init"
git push
```

也可以把文件夹软链到 iCloud Drive / Dropbox / OneDrive。

</details>

<details>
<summary><b>支持 Windows 吗？</b></summary>

支持。原生 Windows 二进制，兼容 PowerShell。WSL 也可用。

</details>

<details>
<summary><b>有 Web UI 吗？</b></summary>

**刻意没有。** 设计理念：日常工作通过 Agent 完成，配置和审计通过编辑器完成。打开 Markdown 文件比操作 Web UI 更快。

</details>

---

## 路线图

- [x] **v1.0** — CLI + Skill 包 + 4 个 Agent 平台 + 9 个核心流
- [ ] **v1.1** — 更多 Agent 平台适配器、社区流仓库
- [ ] **v1.2** — 可选的本地 embedding 增强检索
- [ ] **v1.x** — 第三方 CRM 双向同步（HubSpot / Attio / Salesforce）

---

## 许可证

[MIT](LICENSE) © 2026 AgentCRM Contributors

你的数据，永远是你的。

---

<div align="center">

**[⬆ 回到顶部](#agentcrm)** · **[GitHub](https://github.com/AgentPal/AgentCRM)**

如果 AgentCRM 对你有帮助，请 ⭐ Star——这是支持我们最简单有效的方式。

</div>
