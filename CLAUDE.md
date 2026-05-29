# AgentCRM — Project Instructions for Claude Code

## 这个项目是什么

AgentCRM 是一个本地化、文件存储、零服务进程的客户关系记忆系统。它以 Skill 包 + CLI 工具的形式分发，让 Claude/OpenClaw/Codex/Hermes 等 Agent 在装上之后立刻拥有一个共享的客户大脑。

- **不是**：云端 CRM、SaaS、AI 增强插件
- **是**：本地 CLI 二进制 + Markdown 文件 + Skill 包

完整产品定位见 `README.md`。设计文档见 `docs/v0.3-design.md`（如已创建）。

## 技术栈

- **CLI**：Go 1.22+，pure Go SQLite (`modernc.org/sqlite`)，零 CGO 依赖
- **存储**：Markdown + YAML frontmatter + JSONL + SQLite 索引
- **Skill 包**：Markdown 格式，编译到 Claude/OpenClaw/Codex/Hermes 四个目标
- **测试**：Go `testing` + `testify`

## 核心原则（不可违反）

1. **不引入 daemon 进程**。CLI 跑完就退。任何"长期运行的后台服务"提案需要架构 review
2. **不引入 Node / Python / Docker / 重量级依赖**。binary 必须静态编译、单文件、跨平台
3. **文件即真相**。SQLite 只是索引，永远可从 Markdown/JSONL 重建
4. **每次写入必须带 actor**。`--actor` 参数或 `AGENTCRM_ACTOR` 环境变量
5. **每次 activity 写入必须带 dedupe_key**。否则 Agent 重试会污染数据

## 数据目录约定

- **默认数据目录**：`~/AgentCRM/`（注意大写 A 和大写 CRM）
- **环境变量覆盖**：`AGENTCRM_HOME`
- **命令行覆盖**：`--data-dir <path>`
- **子目录命名规则**：用户可读写的用普通名字（`contacts/`、`deals/`...），机器维护的用 dotfile 前缀（`.index.db`、`.audit/`、`.subscribers/`）
- **路径拼接**：永远使用 `filepath.Join()` 拼路径，不要写死分隔符

## 仓库结构（目标）

```
agentcrm/
├── cmd/agentcrm/        # CLI 主入口
├── internal/
│   ├── store/           # 文件读写 + SQLite
│   ├── model/           # Contact/Deal/Activity/Memo/Event
│   ├── search/          # 多策略融合
│   ├── memory/          # propose/commit/decay
│   ├── events/          # 事件日志 + cursor
│   └── ...
├── skills/              # Skill 源文件 + 多平台编译
│   ├── agentcrm/
│   ├── agentcrm-bridge/
│   └── examples/
├── docs/                # 设计文档
├── .claude/             # Claude Code 工作流配置（commands/hooks/skills）
├── install.sh           # 一行安装脚本
├── Makefile
├── go.mod
├── CLAUDE.md            # 你正在读的这个
├── README.md            # 英文主版本（面向国际受众）
├── README.zh.md         # 中文版本
└── LICENSE              # MIT
```

## 工作流约定

### 分支策略
- `main` 是稳定分支，**永远不要直接 push**
- 所有变更走 feature branch + PR
- 分支命名：`feat/<short-name>` / `fix/<short-name>` / `docs/<short-name>` / `refactor/<short-name>`

### Commit 规范
使用 Conventional Commits：
- `feat: 新功能`
- `fix: bug 修复`
- `docs: 文档`
- `refactor: 重构`
- `test: 测试`
- `chore: 杂项（依赖更新、配置变更等）`

每个 commit 一个原子变更。**不要把 docs 改和代码改放一个 commit**。

### PR 流程
1. 从 main 开新分支
2. 完成变更 + 写测试 + 跑通本地测试
3. 推送分支
4. `gh pr create` 开 PR，标题用 Conventional Commits 格式
5. 等 CI 通过，merge 后删分支

### 代码规范
- Go 代码必须 `gofmt -s -w .` 和 `go vet ./...` 通过
- 公开函数必须有文档注释
- 错误必须显式处理，不允许 `_ = err`
- 测试覆盖率目标 > 70%

## CLI 的对外契约（重要）

CLI 的命令名、参数、输出格式**是 API**。Skill 依赖它们。任何变更必须：
- 加新参数/命令：直接做，向后兼容
- 改输出格式：走 major version bump（v1 → v2）
- 删除参数/命令：先 deprecate 一个版本，再删

具体 API 见 `docs/cli-reference.md`（如已创建）。

## 当前阶段

项目处于 **v1.0.0-dev 阶段**，核心功能已完整实现并发布到 GitHub。当前状态：

- CLI 完整骨架：init/version/reindex/doctor + contact/deal/activity/memory/events/alert/timeline/export/import
- 数据模型：Contact/Deal/Activity/Memo/Event/Alert 完整定义，含历史归档和时间点重建
- 存储层：Markdown 文件 + SQLite 索引（FTS5），文件即真相
- 记忆系统：propose/commit（supersede/keep-both/reject）+ decay 扫描 + 语义冲突检测
- 事件系统：event poll/ack/watch + subscriber cursor + actor 防循环
- 规则引擎：4 条内置规则 + YAML 配置 + alert scan/dismiss/snooze
- 导入导出：JSON / CSV / vCard 全支持
- 测试覆盖：model 95.4%, store 79.4%, cmd 56.4%
- Skill 包：agentcrm 完整版 + agentcrm-bridge 精简版 + 9 个 flow
- 构建发布：GoReleaser 四平台交叉编译 + GitHub Actions CI

**剩余方向**（按需推进）：
- 提升 cmd 覆盖到 80%+（主要是 cobra RunE 匿名函数）
- 端到端集成测试
- 性能验证（1000 contact 下查询 < 500ms）
- 跨平台测试（macOS/Linux）
- i18n 错误消息

## 给 Claude Code 的特殊指令

- **不要主动修改 README.md 的产品定位描述**。这是经过深度讨论的，要改先问我
- **不要自己决定加新依赖**。go.mod 变更必须先告知我并说明理由
- **大改动先写 plan 再动手**。涉及多文件、跨模块的变更，先用 markdown 列计划等我确认
- **测试是必需品不是 nice-to-have**。任何新增的 internal/ 代码都必须有对应的 _test.go
- **Test plan 必须附执行证据**。每个 PR 的 Test plan 中每条勾选必须配
  实际命令输出，不能空打勾：
  - `go vet ./...` → 贴 vet 输出（应为空或 "no issues found"）
  - `go test ./...` → 贴最后 5-10 行（含 PASS/FAIL summary + 包列表）
  - `go build ./...` → 贴 build 输出（所有包列出且无错误）
  
  如果命令失败或报告问题，PR 必须：
  - 在 PR description 中明确报告失败
  - 修复，或说明为何是 known issue
  - **不能**标记为通过
  
  无证据的勾选视为虚假报告，PR 会被退回修订。
- **不要 push 到 main**。所有变更走 PR
