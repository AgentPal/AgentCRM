# AgentCRM

**本地化、文件存储、零服务进程的客户领域记忆系统。让你的 AI Agent 拥有一个长期、可读、可备份的客户记忆。**

```
agentcrm init          # 初始化
agentcrm contact upsert --name "张三" --email "zhang@example.com"
agentcrm activity log --contact <id> --type email --summary "..." --dedupe-key "..."
agentcrm deal create --title "企业版" --contact <id> --amount 50000
agentcrm memory write --scope contact:<id> --text "决策偏好"
```

## 核心理念

- **文件即真相**：所有数据存储在 `~/.agentcrm/` 目录的 markdown 和 JSONL 文件中，可直接用编辑器查看和修改
- **零服务进程**：没有需要守护的 server，CLI 直接读写本地文件
- **AI Agent 优先**：CLI 输出格式对 LLM 友好（JSON/text 可选），配合 Skill 包让 Agent 自主管理客户记忆
- **多 Agent 协作**：通过事件日志和 Actor 标记实现多个 Agent 共享同一数据目录
- **可移植**：支持 JSON/CSV 导出和 vCard/CSV 导入，无供应商锁定

## 快速开始

```bash
# 1. 安装
curl -sfL https://github.com/agentcrm/agentcrm/releases/latest/download/install.sh | sh

# 2. 初始化
agentcrm init

# 3. 创建第一个联系人
agentcrm contact upsert --name "客户姓名" --email "email@example.com" --actor "me"

# 4. 查看帮助
agentcrm --help

# 5. （可选）导出数据
agentcrm export --format json --out ./backup
```

详见 [GETTING_STARTED.md](GETTING_STARTED.md)。

## 文档

- [GETTING_STARTED.md](GETTING_STARTED.md) — 新用户快速上手
- `skills/agentcrm/SKILL.md` — AI Agent Skill 包入口
- `skills/agentcrm/flows/` — 9 个标准操作流程
- `skills/agentcrm/reference/` — 数据模型、CLI 参考、记忆分层指南

## 系统要求

- Go 1.22+（编译）或预编译二进制
- 支持 Windows、macOS、Linux
- 无需数据库服务（使用嵌入式 SQLite）

## 构建

```bash
# 方式一：Go 原生安装（需要 Go 1.22+）
go install github.com/agentcrm/agentcrm/cmd/agentcrm@latest

# 方式二：从源码构建
cd cmd/agentcrm
go build -o agentcrm .
```

## 许可证

MIT
