# AgentCRM 快速上手

## 安装

### macOS / Linux

```bash
curl -sfL https://github.com/AgentPal/AgentCRM/releases/latest/download/install.sh | sh
```

### Windows

从 [Releases 页面](https://github.com/AgentPal/AgentCRM/releases) 下载最新版 `agentcrm-windows-amd64.exe`，重命名为 `agentcrm.exe` 并加入 PATH。

### 从源码构建

```bash
# 方式一：Go 原生安装（推荐）
go install github.com/AgentPal/AgentCRM/cmd/agentcrm@latest

# 方式二：手动构建
git clone https://github.com/AgentPal/AgentCRM.git
cd agentcrm/cmd/agentcrm
go build -o agentcrm .
sudo mv agentcrm /usr/local/bin/
```

## 初始化

```bash
# 创建数据目录和索引
agentcrm init

# 验证安装
agentcrm version
```

数据存储在 `~/AgentCRM/` 目录：

```
~/AgentCRM/
├── config.json      # 配置文件
├── contacts/        # 联系人 markdown 文件
├── deals/           # 商机 markdown 文件
├── activities/      # 活动 JSONL 文件
├── memory/          # 记忆文件
├── events/          # 事件 JSONL 文件
├── rules/           # 规则配置
├── proposals/       # 待裁决提议
├── alerts/          # 提醒
├── .index.db        # SQLite 搜索索引
├── .subscribers/    # 事件订阅 cursor
└── .audit/          # 操作审计日志
```

## 第一个联系人

```bash
# 设置你的身份（或在每次命令中加 --actor "你的名字"）
export AGENTCRM_ACTOR="my-agent"

# 创建联系人
agentcrm contact upsert --name "李四" --email "lisi@example.com" --company "科技有限公司"

# 搜索
agentcrm contact search "李四"

# 查看详情
agentcrm contact get <上一步输出的 id>
```

## 记录第一次互动

```bash
agentcrm activity log \
  --contact <contact-id> \
  --type email \
  --direction in \
  --summary "客户询问产品报价，已发送方案" \
  --dedupe-key "email:msg-12345" \
  --actor "$AGENTCRM_ACTOR"
```

## 创建商机

```bash
agentcrm deal create \
  --title "标准版采购" \
  --contact <contact-id> \
  --amount 30000 \
  --stage qualified \
  --actor "$AGENTCRM_ACTOR"
```

## 查看完整时间线

```bash
agentcrm timeline <contact-id> --detail full
```

## 导出数据

```bash
# 导出为 JSON（完整保留所有字段）
agentcrm export --format json --out ./agentcrm-backup

# 导出为 CSV（适合导入 Excel 或其他系统）
agentcrm export --format csv --out ./agentcrm-backup
```

## 导入联系人

```bash
# 从 vCard 文件导入
agentcrm import --source vcard --file ./contacts.vcf

# 从 CSV 导入（需要 name 列，支持 email/company/title/phone/tags/source）
agentcrm import --source csv --file ./contacts.csv
```

## JSON 输出模式

所有命令支持 `--format json` 参数，适合脚本处理和 AI Agent 消费：

```bash
agentcrm contact get <id> --format json
agentcrm deal list --format json
agentcrm activity list --contact <id> --format json
```

## 下一步

- 阅读 `skills/agentcrm/SKILL.md` 了解 AI Agent 如何自主使用
- 运行 `agentcrm alert scan` 检查待处理事项
- 配置 `AGENTCRM_ACTOR` 环境变量让每个 Agent 有身份
- 定期使用 `agentcrm export --format json --out ./backup` 备份数据
