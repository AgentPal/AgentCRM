好。我先把 9 个 flow 的整体设计原则想清楚，避免每个都重复造轮子。

**统一结构**：每个 flow 都按 `何时触发 / 何时不触发 / 标准流程 / 不要做 / 失败处理 / 示例对话` 组织。这是上文档已经定下的模式。

**统一的注意点**（不在每个 flow 里重复写，但 Agent 要内化）：

- 所有写入都带 `AGENTCRM_ACTOR` env（由 Skill 安装时配置）
- 所有 activity log 都带 `--dedupe-key`
- 所有事实变化优先走 `memory propose` 而非 `memory write`
- CLI 失败原样转达给用户

**flows 之间的相互引用关系**：

```
new-lead ────┐
             ├──> log-interaction (邻近场景)
who-is-this ─┘
                                   ┌─> update-fact (事实变化)
log-interaction ──> deal-update ───┤
                                   └─> handle-alert (触发提醒)

prep-followup ──> (用所有读取能力)
weekly-review ──> (用 alert scan + events poll)
cleanup-dups ──> (低频，独立)
```

写得足够具体，让 Agent 拿来就能用。让我把 9 个文件依次写出来。

---

## flows/new-lead.md

````markdown
# Flow: 新线索捕获

## 何时触发
- 用户分享邮件 / 聊天截图 / 名片，里面出现一个新人
- 用户口头说"我刚认识了 X" / "把 X 加进来" / "记一下这个人"
- 你在处理别的任务时遇到一个陌生的业务相关名字，且**有足够信息识别**
- ingestion 流程（如邮件抽取）识别到当前 contacts 里没有的发件人

## 何时不触发
- 只有名字没有任何其他识别信息（先问用户）
- 看不出业务关联（家人、朋友、明显是个人事务的对象）
- 用户在抱怨某人但没说要"加"或"记"
- 你不确定要不要加 → 问，不要默认加

## 标准流程

### 1. 提取信息
从上下文中提取以下字段：

**必需**（任一即可作为主标识）：
- 邮箱（首选）
- 电话
- 社交账号 (twitter / linkedin / wechat / github)

**推荐**：
- 姓名
- 公司
- 职位
- 来源（明确从哪儿来的）

**可选**：
- 标签
- 备注 / 认识背景

如果一个能识别的字段都没有，先问用户：
> "我看到了「X」这个名字，能告诉我他的邮箱、电话或某个社交账号吗？这样以后能正确关联到他的其他互动。"

### 2. 检查重复
不要假设是新的，先搜一下：

```bash
agentcrm contact search "<识别字段>" --format json
````

如果有高匹配（同邮箱 / 同公司+同名 / 同 social handle），**告诉用户并等确认**：

> "我找到一个可能是同一人的联系人「张三 (ABC 科技, CTO, 最近互动 12 天前)」。是同一个人吗？"

- 用户说"是" → 跳到步骤 4（用现有 contact_id 走后续）
- 用户说"不是" → 继续步骤 3
- 用户不确定 → 仍然新建，加 tag `possible-dup-of-<id>`，以后再合并

### 3. 创建联系人

```bash
agentcrm contact upsert \
  --name "张三" \
  --email "zhang.san@abc.com" \
  --company "ABC 科技" \
  --title "CTO" \
  --source "<明确来源: gmail | wechat | event | referral:黄五 | ...>" \
  [--social.linkedin "..."] \
  [--phone "..."] \
  [--tag <相关标签>] \
  --format json
```

记下返回的 `contact_id`。

**source 字段必须明确**。不要默认 "manual"——尽量精确（"gmail" / "twitter-mention" / "referral:<某人>"）。这是以后回溯线索质量的关键。

### 4. 如果有"认识背景"信息，加 memory

注意：这里是**稳定的关系背景**，不是这次互动事件。

```bash
agentcrm memory write \
  --scope contact:<id> \
  --text "通过黄五介绍认识，对 Q3 AI 集成项目感兴趣" \
  --decay never
```

什么算稳定背景：

- 推荐人、首次相遇场合、关系起点
- 长期身份信息（"是 X 圈子的"、"做 Y 行业的"）

什么不算（应该走 activity 而不是 memory）：

- "今天聊了什么" → activity
- "这次邮件提到 X" → activity

### 5. 如果这次接触本身是个具体事件，记 activity

```bash
agentcrm activity log \
  --contact <id> \
  --type email | call | meeting | chat \
  --direction in | out \
  --channel <gmail | telegram | wechat | ...> \
  --summary "<这次互动的一句话总结>" \
  --dedupe-key "<唯一标识，如邮件 message-id>"
```

如果没有自然的 dedupe_key（手动输入的场景），用 `manual:<contact_id>:<timestamp>` 格式。

### 6. 简短确认给用户

不要罗嗦。一句话告诉用户：发生了什么 + 是否有可继续的下一步。

> "已加入张三 (ABC 科技, CTO)，记录了你们认识的背景和这次邮件互动。"

如果创建过程中有任何**判断**（比如 source 是猜的），告诉用户：

> "已加入。我把 source 标为 'gmail'，如果不对的话告诉我。"

## 不要做

- **不要自动创建 deal**。即使对方"看起来想合作"。除非用户明说"这是个商机"或描述了具体项目+金额，才考虑走 deal-create
- **不要去查第三方数据 enrich**（LinkedIn 爬虫、企业信息接口等）。除非用户明确要求
- **不要群发欢迎邮件**或任何对外动作
- **不要假设 tags**。"vip" / "decision-maker" 这种重要标签必须用户授权才打
- **不要把识别置信度低的字段填进去**。如果"猜"张三的公司是 ABC，但只有 50% 把握，不要写。或者写进去但加 tag `unverified` 让用户后续确认

## 失败处理

|情况|处理|
|---|---|
|`contact upsert` 返回 `merged_into: <id>`|说明已存在，告诉用户合并情况并返回|
|CLI 报错（参数错、文件锁、权限）|把错误原样告诉用户，**不要自动重试**，问用户怎么办|
|信息不全（只有 first name + 公司）|暂存提议，告诉用户"我先没加入，因为信息不全。等你告诉我邮箱后再加"|
|多个候选重复（搜出 3 个张三）|列出全部候选 + 关键区分字段（公司 / 最近互动），让用户选|

## 示例对话

**示例 1：来自邮件**

```
用户: 这是张三刚发来的邮件 (附件)，把发件人加进来吧。
你:   (调 contact search "zhang.san@abc.com" → 无结果)
      (调 contact upsert ...)
      (调 activity log ...)
      已加入张三 (ABC 科技, CTO)，并记了这封邮件。要不要我从邮件正文里也提取一些关于他职责或项目的背景？
用户: 好的，他提到他们 Q3 想做 AI 集成。
你:   (调 memory write --scope contact:cnt_xxx --text "Q3 想做 AI 集成，是他主导的方向" --decay 180d)
      记下了。Q3 集成的事在他那边是主导方向。如果之后聊到这个我会主动关联。
```

**示例 2：信息不全**

```
用户: 黄五说有个朋友想认识我，叫陈七。
你:   只有名字"陈七"我不好建档（容易和其他陈七混）。等你拿到他的邮箱或微信再加？或者你现在能告诉我他的公司或职位吗？
用户: 他在某 AI 创业公司当 CEO。
你:   还需要一个唯一识别（邮箱 / 微信 / linkedin）。要不我先记一条 memory 在 global 里："黄五的朋友陈七，AI 创业公司 CEO"，等下次见到联系方式再正式建档？
用户: 好。
你:   (调 memory write --scope tenant --text "黄五的朋友陈七，AI 创业公司 CEO，待补联系方式" --decay 90d)
      记下了。下次有他联系方式就能正式加入。
```

**示例 3：重复检测**

```
用户: (转发一封 li.si@xyz.com 的邮件) 把这个 li.si@xyz.com 加一下。
你:   (调 contact search "li.si@xyz.com" → 找到 cnt_existing "李四 (XYZ 公司)")
      已经有李四 (XYZ 公司, 最近互动 3 天前) 在 CRM 里。我直接记一下这封新邮件就行？
用户: 好。
你:   (调 activity log ...)
      已记录。
```

````

---

## flows/log-interaction.md

```markdown
# Flow: 记录一次互动

## 何时触发
- 用户说"刚跟 X 通了电话" / "见了 Y" / "在群里聊了 Z"
- 用户分享一段聊天记录 / 邮件 / 会议纪要 / 通话转录
- 你（或另一个 Agent）自动捕获到一次互动（邮件 ingestion、社交 monitor 等）
- 用户主动转发某个事件让你"记一下"

## 何时不触发
- 用户只是闲聊，没有要记录的明确意图，且对方不是已知客户
- 互动内容明显是私人事务（生日聚会、家庭话题）
- 已经有同 dedupe_key 的 activity 存在（重复触发会被 CLI 自动跳过，但你不应该主动重复调用）

## 标准流程

### 1. 识别对方是谁

```bash
agentcrm contact search "<姓名 | 邮箱 | 其他识别>" --format json
````

**找到**：拿到 `contact_id`，进入步骤 2。

**没找到**：

- 如果用户语境里有足够信息加新人 → 先走 [new-lead](https://claude.ai/chat/new-lead.md)，再回来
- 如果信息不足 → 告诉用户"我没找到 X，要不要先加？" 不要瞎记到 global

**找到多个候选**：列出关键区分字段（公司、最近互动），让用户选。

### 2. 判断关联的 deal

如果用户提到具体项目 / 报价 / 阶段，先查活跃 deals：

```bash
agentcrm deal list --contact <id> --stage-not-in won,lost --format json
```

- 一个 in-flight deal → 用它的 deal_id
- 多个 → 看互动内容明确指向哪个；不确定就问用户
- 没有 → activity 只关联 contact 不关联 deal

### 3. 写入 activity

```bash
agentcrm activity log \
  --contact <id> \
  [--deal <id>] \
  --type email | call | meeting | chat | note | social \
  --direction in | out \
  [--channel gmail | wechat | twitter | phone | ...] \
  --summary "<一句话核心>" \
  [--body-file <path>] \
  --dedupe-key "<unique>"
```

**summary 怎么写**：

- 一句话，最长 1-2 行
- 包含"谁说了什么"或"达成了什么"的核心
- 不写情绪化形容词，写事实

好例子：

- "反馈预算偏紧，要求分阶段付款"
- "确认 Q3 启动，技术方案下周二评审"
- "未接电话，留言请其回拨"

坏例子：

- "聊得很愉快"（没信息）
- "讨论了项目"（没具体内容）
- "客户很喜欢我们的方案，未来很有希望"（情绪+猜测）

**body-file**：完整内容（邮件正文、会议纪要全文）放文件里，summary 是索引。如果原始内容只有几句话也可以省略 body-file。

### 4. 判断是否触发其他动作

记完 activity 不一定就完事了。看互动内容判断：

|触发信号|后续 flow|
|---|---|
|对方说"我现在在 X 公司" / 换 title / 改电话|→ [update-fact](https://claude.ai/chat/update-fact.md)|
|对方明确同意推进 / 拒绝 / 卡在某阶段|→ [deal-update](https://claude.ai/chat/deal-update.md)|
|提到长期偏好 / 决策模式 / 性格特征|→ 走 memory propose（见 update-fact）|
|提到具体未来事项（"下周二开会"）|→ 提醒用户记进日历，不入 AgentCRM|

如果不确定，**先记 activity，再问用户**"听起来 X，要不要更新他的 Y / 推进商机阶段？"

### 5. 简短回复用户

> "已记录这次邮件互动。注意到他提到预算紧，要不要把 deal 阶段从 proposal 调整一下？"

## 不要做

- **不要把 summary 写得像营销文案**。"客户表示极大兴趣"——错。"对方说会再考虑两周"——对
- **不要在 activity 里塞稳定事实**。"张三是 CTO" 不该塞 summary，应该是 contact frontmatter 字段
- **不要忘记 dedupe_key**。即使是 manual 输入也要拼一个（`manual:<contact_id>:<iso_timestamp>`）。重复调用 CLI 会被自动跳过，但前提是你有 dedupe_key
- **不要跨 contact 一次记多条**。如果一场会议有 3 个对方，调用 3 次 activity log，每条关联一个 contact（用相同的 dedupe_key 加后缀如 `meeting:abc:zhang`/`:li`/`:wang`）

## 失败处理

|情况|处理|
|---|---|
|`activity log` 返回 "duplicate, returning existing id"|正常，告诉用户"已经记过了"，不重复操作|
|找不到对应 contact 且信息不足|不要瞎写到 global memory；告诉用户"我没识别出这是谁"|
|body-file 路径无效|重试不带 body-file，但保留 summary|

## 示例对话

**示例 1：通话记录**

```
用户: 刚跟李四通了 20 分钟电话，他说他们 CEO 同意了 50K 的方案，但希望分两期。
你:   (调 contact search "李四" → cnt_li)
      (调 deal list --contact cnt_li --stage-not-in won,lost → 1 个: del_xyz "Q3 项目" proposal 50K)
      (调 activity log --contact cnt_li --deal del_xyz --type call --direction in
         --summary "CEO 同意 50K 方案，要求分两期付款"
         --dedupe-key "manual:cnt_li:2026-05-28T15:30:00Z")
      已记下。听起来 deal 可以推进了——要不要把阶段从 proposal 改成 negotiation？另外分两期的事要不要让我帮你写到 deal 的 notes 里？
用户: 推进到 negotiation，分期的事我自己改。
你:   (触发 deal-update flow)
```

**示例 2：邮件 ingestion 触发**

```
邮箱 Agent 调用（非交互）:
  (识别发件人 zhang.san@abc.com)
  (调 contact search → cnt_zhang)
  (调 activity log --contact cnt_zhang --type email --direction in
     --channel gmail --summary "..." --dedupe-key "<message-id>")
  → 不需要回复用户，静默完成
```

**示例 3：多人会议**

```
用户: 上午跟 ABC 那边开了个会，张三、李四、王五三个人都在。
你:   (调 contact search → 找到 3 个)
      要不要把会议纪要给我，我分别给每人记一条 activity？或者你想我只在张三那里记一条主纪要、其他两人各加一条"参与了会议"？
用户: 主纪要在张三那里，其他两个加参与。
你:   (3 次 activity log，dedupe-key 后缀不同)
```

````

---

## flows/deal-update.md

```markdown
# Flow: 商机推进与丢失

## 何时触发
- 用户说"X 同意了" / "Y 黄了" / "Z 改成 negotiation 阶段"
- 用户报告价格 / 范围 / 截止日变化
- 互动中识别出阶段跃迁信号（"我们决定签" / "我们再看看" / "预算砍了"）
- 系统提醒（alert scan）建议跟进某个 stale deal，用户决定推进

## 何时不触发
- 信息模糊（"客户那边有点意思" → 没有可执行的阶段变化）
- 涉及金额或不可逆变化但用户没明确表态（先确认，不要自作主张）
- 创建新 deal（走 deal-create，不在本 flow 范围）

## 阶段定义（v1.0 标准）

````

lead → qualified → proposal → negotiation → won → lost

````

| 阶段 | 含义 |
|---|---|
| lead | 初步接触，对方还没明确表态 |
| qualified | 对方有需求且匹配，但还没开始正式商务 |
| proposal | 已发送报价 / 方案 |
| negotiation | 双方在具体条款上推进（金额、范围、节奏） |
| won | 已签约 / 已成交 |
| lost | 明确不做了 |

## 标准流程

### 1. 定位 deal

```bash
agentcrm deal list --contact <id> --stage-not-in won,lost --format json
````

- 一个 → 用它
- 多个 → 让用户明确指哪个
- 零个 → 这不是 deal-update，是 deal-create。走另一个流程或问用户"听起来要创建新商机？"

### 2. 识别变更类型

|变更|字段|风险|
|---|---|---|
|阶段推进 (lead → qualified → ...)|`stage`|低-中|
|金额变化 < 20%|`amount`|低|
|金额变化 > 20%|`amount`|中|
|阶段进入 won 或 lost|`stage`|中（不可逆，要确认）|
|预计成交日变化|`expected_close_at`|低|
|标题 / 描述 修改|`title` / 正文|低|

### 3. 高风险变更需要确认

**进入 won / lost 前必须明确确认**。即使用户说了"X 同意签了"，你也要回一句：

> "确认把「ABC Q3 项目」标记为 won 吗？金额 50,000 CNY，关联李四。一旦标 won 我会同步触发后续动作。"

为什么：won/lost 是产品的关键信号，影响后续 alert、报表、follow-up 策略。错标代价大。

**金额变化超 20% 也要复述确认**。

### 4. 执行更新

```bash
agentcrm deal update <id> \
  --set "stage=negotiation" \
  --reason "<一句话原因，会进 stage_history>"
```

CLI 会自动：

- 把旧 stage 归档到 `stage_history`
- 写一条 system activity 记录"stage X → Y by <actor>"
- 触发 `deal.stage_changed` 事件（其他订阅 Agent 会拿到）

多字段同时改：

```bash
agentcrm deal update <id> \
  --set "stage=won" \
  --set "amount=45000" \
  --reason "最终签约价 4.5 万，降了 5K"
```

### 5. 关联记录这次互动

如果这次更新来自一个具体事件（电话 / 邮件 / 会议），先走 [log-interaction](https://claude.ai/chat/log-interaction.md) 记 activity，再做 stage 变更。两条记录指向同一个 deal_id，时间顺序自然形成 timeline。

### 6. won / lost 的特殊后续

**won 后**：

- 提醒用户"要不要给客户发感谢邮件？"（让用户决定，不自动发）
- 提醒用户"要不要在 memory 里记一条决胜原因？"（成功经验长期价值高）
    
    ```bash
    agentcrm memory write \  --scope contact:<id> \  --text "Q3 项目签约关键：分两期付款让客户安心，技术深度路演打动了 CTO" \  --decay never
    ```
    

**lost 后**：

- **必须**记一条 lost reason 到 memory（这是 CRM 长期最有价值的数据之一）：
    
    ```bash
    agentcrm memory write \  --scope contact:<id> \  --text "Q3 项目失败原因：客户最终选了竞品 X，主要因为价格低 20%" \  --decay never
    ```
    
- 不要追加跟进 alert。lost 的 deal 不应该再被 stale alert 提醒

### 7. 回复用户

简短，包含：发生了什么 + 已触发的副作用。

> "已把「ABC Q3 项目」推进到 negotiation。同步记了你刚才描述的通话。"

> "已标记为 won。要不要记一条决胜原因？这种数据以后做复盘很有用。"

## 不要做

- **不要在用户说"看起来要签了"时直接标 won**。won 需要明确事实（合同签了 / 款到了 / 客户口头确认签约）
- **不要在用户抱怨某 deal 难时标 lost**。难和黄不一样
- **不要跳过 `--reason`**。stage_history 没 reason 是哑数据，半年后翻不出价值
- **不要忘记 lost 必须记 memory**。这条是反直觉但重要的纪律
- **不要自动发感谢 / 道歉邮件**。即使用户说"该感谢一下"——告诉他你可以起草，让他确认后再发

## 失败处理

|情况|处理|
|---|---|
|deal 已是目标 stage（重复操作）|CLI 返回 "no change"，告诉用户"已经是 X 阶段了"|
|stage 跃迁不合规（lost → won 这种）|CLI 拒绝，告诉用户"lost 的 deal 不能直接转 won，要么撤销 lost 要么开新 deal"|
|amount 变化巨大但未确认|不要强行更新，问用户|

## 示例对话

**示例 1：常规推进**

```
用户: 李四刚电话里说他们决定推进了，方案细节下周敲定。
你:   (定位 deal del_xyz "Q3 项目" 当前 proposal)
      把 Q3 项目从 proposal 推进到 negotiation，对吗？我会同时记一下这次通话。
用户: 对。
你:   (调 activity log ... + deal update --set stage=negotiation --reason "李四电话确认推进，下周敲细节")
      已推进到 negotiation，通话也记下了。
```

**示例 2：丢单**

```
用户: ABC 那个项目黄了，他们选了竞品。
你:   (定位 deal del_xyz)
      确认把「ABC Q3 项目」标记为 lost 吗？这之后我不会再就这个 deal 提醒你了。
用户: 是。
你:   你说他们选了竞品——能告诉我是哪家竞品？以及他们选的主要原因（价格、功能、关系）？
用户: 选了 X 公司，主要是价格低。
你:   (调 deal update --set stage=lost --reason "客户选竞品 X")
      (调 memory write --scope contact:<id>
         --text "Q3 项目失败：客户选竞品 X，主因价格"
         --decay never)
      已标 lost 并记下了失败原因。
```

**示例 3：金额大幅调整**

```
用户: ABC 项目价格调到 3 万了。
你:   (deal 当前 amount = 50000)
      你说要从 5 万改到 3 万？这是 40% 的下调，确认一下：原因是什么？scope 缩了，还是单纯降价？
用户: scope 砍掉了第二期。
你:   (调 deal update --set "amount=30000" --reason "scope 砍二期，仅保留一期")
      已改 5 万 → 3 万，原因记为"砍二期"。要不要我帮你在 deal 的描述里把 scope 也更新一下？
```

````

---

## flows/update-fact.md

```markdown
# Flow: 处理事实变化

## 何时触发
- 对方告诉你他自己的事实变化（"我换工作了" / "我升 CEO 了" / "公司搬到上海了"）
- 用户告诉你某联系人的事实变化（"张三跳槽到 X 公司了"）
- 你从邮件签名 / 社交资料 / 名片识别到当前 frontmatter 字段的新值
- 用户主动让你"更新 X 的信息"

## 何时不触发
- 临时状态（"他这周很忙" / "最近在外地"）→ 这不是事实，是状态。走 activity 或不记
- 短期项目相关的事（"他现在主导 X 项目"）→ 走 memory propose（180d decay），不是 fact
- 用户表达不确定（"听说张三好像换工作了"）→ 先问清楚，不要在不确定时改

## 这个 flow 的核心理念

直接 `contact update` 会归档旧值但**不会问"这个变化合理吗"**。粗暴覆盖是 CRM 数据质量的最大杀手。

正确路径：
1. **如果是 frontmatter 字段**（company / title / phone / location 等）：用 `contact update`，CLI 自动归档历史
2. **如果是 memory 层事实**（偏好 / 决策模式 / 关系背景）：用 `memory propose`，CLI 检测冲突，你来决策

## 标准流程

### 1. 判断字段类型

| 信号 | 字段类型 |
|---|---|
| 公司、职位、地点、电话、邮箱、social handle | frontmatter |
| 沟通偏好、决策模式、性格特征、长期关系 | memory |
| 这周这月的状态 / 临时事项 | activity（不在本 flow） |

不确定就走 memory（更保守，propose 流程会检测冲突）。

### 2. 如果是 frontmatter 字段

#### 2a. 先查当前值

```bash
agentcrm contact get <id> --format json
````

确认现有值。如果新值和现有相同，告诉用户"已经是这个值了"，结束。

#### 2b. 执行 update

```bash
agentcrm contact update <id> \
  --set "<field>=<new_value>" \
  --reason "<谁告诉你的 / 何时告诉你的>"
```

CLI 自动：

- 把旧值归档到 `<field>_history`，带 from / to / set_by
- 触发 `contact.field_updated` 事件
- 写 audit log

例：

```bash
agentcrm contact update cnt_zhang \
  --set "company=DEF 公司" \
  --set "title=CEO" \
  --reason "2026-05-28 邮件签名变化，对方在邮件中确认"
```

#### 2c. 记关联 activity（如果有具体事件触发）

如果这个变化来自一封邮件 / 一次通话，先走 log-interaction 记下，让事实变化和事件关联起来。

### 3. 如果是 memory 层事实

#### 3a. propose

```bash
agentcrm memory propose \
  --scope contact:<id> \
  --statement "<新陈述>" \
  --source-snippet "<来源原文片段，如邮件中的一句话>" \
  --confidence <0-1> \
  --format json
```

confidence 怎么定：

- 直接听对方亲口说 → 0.9+
- 用户告诉你的 → 0.85
- 你从上下文推断 → 0.6-0.7
- 不确定但想记一下 → 0.4-0.5（这种 CLI 可能会 reject）

CLI 返回（JSON）：

```json
{
  "proposal_id": "prop_01",
  "status": "clean" | "conflict",
  "conflict_with": [
    {"memo_id": "mem_xxx", "text": "...", "valid_from": "...", "decay": "..."}
  ],
  "suggested_action": "supersede" | "keep-both" | "reject"
}
```

#### 3b. 决策

**status: clean** → 直接 commit accept：

```bash
agentcrm memory commit --proposal-id prop_01 --action accept
```

**status: conflict** → 看冲突情况选择：

|情况|选择|例子|
|---|---|---|
|新陈述明确取代旧的（"现在是 X"）|`supersede`|"现在偏好周五" 取代 "偏好周三"|
|新陈述补充旧的（两者并存）|`keep-both`|"偏好邮件" + "急事可微信"|
|不确定 / 信息不足|`reject` + 问用户|模糊的、矛盾的|

```bash
agentcrm memory commit --proposal-id prop_01 --action supersede
```

`supersede` 不会真删旧 memo——它把旧 memo 标记 `valid_to: <now>`，归档为历史。`memory history` 仍可查。

### 4. 通知用户

简短，包含：变化、归档情况、是否需要其他动作。

> "已更新：张三的职位 CTO → CEO（旧值 2024-03-15 至今的 CTO 归档到历史）。"

> "更新了张三的沟通偏好：现在记的是「周五下午可用」，把之前「周三下午」归档。如果两者都成立告诉我，我改成保留两条。"

## 不要做

- **不要在 propose 返回 conflict 时直接 supersede**。先看冲突内容判断
- **不要把"这周很忙"当事实变化更新**。这是 activity，不是 fact
- **不要忘记 reason**。无 reason 的历史归档半年后看不懂
- **不要批量更新多个字段而不告诉用户**。如果你识别到一封邮件签名变化里 company / title / phone 都不同了，分两步：先告诉用户检测到三处变化，确认后再批量改
- **不要在用户说"听说 X 换工作了"时直接更新**。"听说"是低 confidence，先 propose 让 CLI 标记冲突 + 问用户来源

## 失败处理

|情况|处理|
|---|---|
|`contact update` 字段不存在|CLI 拒绝，告诉用户"<字段> 不是已知 frontmatter 字段，要不要加到 memo？"|
|propose 返回 reject（confidence 太低）|告诉用户"我没把握，确认一下：是 X 还是 Y？"|
|多个冲突 memo|把全部冲突列给用户看，让用户决定怎么 supersede|

## 示例对话

**示例 1：换工作（frontmatter）**

```
用户: 张三刚发邮件说他从 ABC 跳到 DEF 了，title 升 CEO 了。
你:   (调 contact get cnt_zhang → 当前 company=ABC, title=CTO)
      要更新张三：company ABC → DEF, title CTO → CEO。同时把当前邮件记一条 activity。确认吗？
用户: 确认。
你:   (调 activity log ...)
      (调 contact update cnt_zhang --set "company=DEF" --set "title=CEO"
         --reason "2026-05-28 邮件，对方亲口告知")
      已更新。company 和 title 的旧值归档到 history，半年后查 `agentcrm contact history cnt_zhang --field company` 能看到。
```

**示例 2：偏好变化（memory）**

```
用户: 李四说他周五下午改去带娃了，以后约他还是周三吧。
你:   (调 memory propose --scope contact:cnt_lisi
         --statement "李四沟通偏好：周三下午可用，周五下午改为带娃时间"
         --source-snippet "周五下午改去带娃了，以后约他还是周三"
         --confidence 0.9)
      → CLI 返回 conflict_with: [{memo_id: mem_xxx, text: "偏好周五下午沟通", valid_from: 2025-04-20}]
        suggested_action: supersede
      之前 2025-04 记的是"偏好周五下午"，现在改回周三。要 supersede（保留历史能查）还是 keep-both？
用户: supersede。
你:   (调 memory commit --proposal-id prop_xx --action supersede)
      已更新。旧的"偏好周五"归档到 2026-05-28 失效，现在生效的是"周三可用"。
```

**示例 3：低 confidence**

```
用户: 听说王五好像跳槽去 X 公司了？
你:   "听说"和"好像"——我不直接改 frontmatter。要不你确认一下来源？是他自己说的、还是社交平台看到的、还是别人转述的？另外你查过他最新的邮件签名吗？
用户: 是别人转述的，我没核实。
你:   那我先不改。给你一条提醒：「核实王五是否跳槽到 X」，下次有机会确认了再来更新。要不要我建一条 alert？
用户: 不用，我自己记着。
你:   好，没改任何东西。
```

**示例 4：keep-both 场景**

```
用户: 张三说急事可以微信。
你:   (memory propose --statement "张三急事偏好微信沟通" --confidence 0.95)
      → conflict_with: [{text: "偏好邮件 + 偶尔微信", valid_from: 2025-01-01}]
        suggested_action: keep-both
      和现有"偏好邮件+偶尔微信"不矛盾，可以追加为"急事用微信"。我用 keep-both 处理？
用户: 对。
你:   (commit --action keep-both)
      已追加。现在两条都生效。
```

````

---

## flows/prep-followup.md

```markdown
# Flow: 准备跟进

## 何时触发
- 用户说"我要跟 X 谈，给我准备一下" / "明天要见 X，帮我看看背景"
- 用户问"X 最近怎么样" / "上次和 X 聊到哪了"
- alert 提醒触发某个 stale deal，用户决定准备跟进
- 用户发起类似话术准备 / 邮件起草请求，需要客户上下文

## 何时不触发
- 用户只是问简单事实（"X 的邮箱是？" → 直接 contact get 就行，不用整套 follow-up 准备）
- 用户问的是统计 / 总览（"我有多少 deal" → 用 deal list，不是这个 flow）

## 标准流程

### 1. 定位对象

可能是 contact 也可能是 deal。

```bash
agentcrm contact search "<X>" --format json
# 或
agentcrm deal list --contact <id> --stage-not-in won,lost --format json
````

如果有多个候选 → 让用户明确。

### 2. 拉全景上下文（三层都读）

这是 flow 的核心——把记忆系统的四层读出三层（除了 working）综合给用户。

```bash
# Layer 1: Static facts
agentcrm contact get <id> --format json

# Layer 1.5: 相关 in-flight deals
agentcrm deal list --contact <id> --stage-not-in won,lost --format json

# Layer 2: Semantic memory (经验)
agentcrm memory recall "<跟进相关 query>" --scope contact:<id> --top-k 5 --format json
# 或拉全部
agentcrm memory list --scope contact:<id> --format json

# Layer 3: Recent episodic (最近发生的)
agentcrm timeline <contact_id> --since 90d --detail standard --format json
```

如果有具体 deal，也拉 deal 的 summary：

```bash
agentcrm deal summarize <deal_id>
```

### 3. 综合（不是机械拼接）

不要把四个 CLI 输出原样喂给用户。综合成一段可用的简报：

**结构**：

1. **一句话定位**：谁，当前状态
2. **关键事实**：跟跟进话题相关的稳定事实（title、决策角色、偏好）
3. **当前 deal 状态**：如果有 in-flight deal
4. **最近发生了什么**：最近 3-5 条互动，按时间倒序，每条一行
5. **可能的卡点 / 需注意的事**：从 memory + activity 抽取
6. **建议的下一步**：基于上下文给具体建议（可选）

### 4. 提醒过期记忆

```bash
agentcrm memory list --scope contact:<id> --format json
```

如果有 `expired: true` 的 memo，提醒用户：

> "注意：有 2 条关于他的记忆已过期（"最近接触竞品 X" - 2026-08-01 失效）。要不要我先帮你确认还成立吗？"

### 5. 输出给用户

简报式，结构化，不冗长。

如果用户问的是"准备见面"，建议附上"建议聊的话题"和"应该避开的话题"。

### 6. 不写入

**这个 flow 通常是只读的**。除非用户在准备过程中告诉你新信息（"哦对，他刚搬家了"），那才触发其他 flow（update-fact）。

## 不要做

- **不要把全部 timeline 喂给用户**。挑相关的，最多 5 条
- **不要瞎猜 next-step**。"建议你跟进"是废话。给具体建议（"建议先确认上次报价分期方案的最终意见"）或不给
- **不要把过期记忆当当前事实呈现**。过期就标记，不要默默用
- **不要在 prep 过程中改数据**。读优先

## 失败处理

|情况|处理|
|---|---|
|找不到对应 contact|告诉用户"我没找到 X，要不要先加？"或问拼写|
|Memory recall 返回空|OK，正常。简报里只有 facts + activities|
|Timeline 太短（新联系人）|简报短就短，告诉用户"互动较少，目前只有 X 条记录"|

## 示例对话

**示例 1：完整准备**

```
用户: 明天下午要跟张三视频，帮我看下背景。
你:   (调 contact get cnt_zhang)
      (调 deal list --contact cnt_zhang → 1 个 in-flight deal_xyz "Q3 项目")
      (调 memory list --scope contact:cnt_zhang)
      (调 timeline cnt_zhang --since 90d --detail standard)
      (调 deal summarize deal_xyz)

      张三 (DEF 公司, CEO) 跟进准备：

      **当前 deal**: ABC Q3 AI 集成项目，5 万 CNY，目前 negotiation 阶段，
      预计 8/31 成交。

      **关键事实**:
      - 决策模式：技术细节自己拍，预算回去对齐李四 (但李四已不再是 CEO)
      - 沟通偏好：邮件 + 急事微信，周三下午可用

      **最近发生的事**（最近 4 次互动）:
      - 5/20 通话 (you/in)：他说 CEO 同意 5 万，要求分两期
      - 5/15 邮件 (you/in)：反馈预算紧
      - 5/10 邮件 (you/out)：发出提案 v1
      - 5/05 会议 (you/in)：scope 确认

      **可能聊到的卡点**:
      - 分两期付款的具体节奏（一期 / 二期金额、节点）
      - 他刚升 CEO（5/28 才更新），决策链可能变了——他现在自己拍板？

      **建议先聊**:
      1. 确认分期方案细节（这是上次卡的点）
      2. 借机了解他升 CEO 后决策流程变化，未来回款节奏
      3. （可选）问下他对二期 SDR 自动化的预期时间

      要不要我帮你起草开场？
用户: 不用，我自己来。
```

**示例 2：极简上下文**

```
用户: 王五是谁？
你:   (调 contact get → 信息很少)
      王五 (HJK 公司, 销售总监, 通过黄五介绍认识)。
      没有 in-flight deal，最近互动是 2 个月前一条邮件，他说想下季度再聊。
      你想现在做什么？
```

**示例 3：过期提醒**

```
用户: 跟李四的会前准备一下。
你:   (...全部读取...)
      ...
      **⚠️ 注意**: 有一条记忆已过期（"最近评估竞品 X" - 90d decay 已过）。要不要先在会上侧面问下他还在评估吗？
      ...
```

````

---

## flows/weekly-review.md

```markdown
# Flow: 周回顾

## 何时触发
- 用户说"这周怎么样" / "本周总结" / "周回顾" / "盘一下"
- 周一早上 / 周五下午这种典型时段，用户问开放问题
- cron 触发的 alert scan 累积了多条事项，用户决定一起看
- 用户从 alert 列表跳进来（"今天 / 本周有什么要看的？"）

## 何时不触发
- 用户问的是某个具体客户 / deal（走 prep-followup）
- 用户问的是某次具体事件（走 log-interaction 的查询场景）

## 标准流程

### 1. 拉全局快照（多 CLI 并行）

```bash
# 1. 当前 pipeline 概览
agentcrm deal list --stage-not-in won,lost --format json

# 2. 本周内活跃的客户（有 activity 的）
agentcrm contact list --updated-since 7d --format json

# 3. 本周事件汇总
agentcrm events poll --as <your-actor> --since 7d --format json
# 或者用范围查询
agentcrm activity list --since 7d --format json

# 4. 当前所有 alert
agentcrm alert scan
agentcrm alert list --format json

# 5. 本周 stage 变化（重要里程碑）
# 通过 events 过滤 type=deal.stage_changed
````

### 2. 综合结构

用户要的不是数据列表，是洞察。组织成：

**结构**：

**A. 大事件**（本周发生的不寻常 / 关键事项）：

- 新签的 deal（won）
- 黄掉的 deal（lost）
- 新增的 VIP 客户
- 大额变化的 deal

**B. Pipeline 状态**：

- 各 stage 数量 + 金额
- 比上周变化（如果有数据）
- 即将成交的（expected_close < 7d）

**C. 需要你关注的**（alert 整理后）：

- 按优先级排（VIP > 大额 > 一般）
- 同类合并（"3 个 deal 都超过 14 天未动"）

**D. 多 Agent 动态**（如果有其他 Agent 在写入）：

- 邮箱 Agent / 社交 Agent 自动捕获了什么新东西
- 是否有需要你处理的（如新联系人没确认 source）

**E. 建议聚焦**（最多 3 件）：

- 这周值得花时间做的事

### 3. 输出风格

- **不超过 1 屏**。再长用户就不看了
- 用 emoji / bullet 适度区分（但符合 user preferences）
- 每条事项后附上**可以触发的下一步**（让用户能直接说"做 #2"）

### 4. 后续动作

通常用户会从回顾里选一两件让你处理：

> 用户: "做第 2 个" / "帮我跟进 #3 那个"

那就转入对应 flow（prep-followup / log-interaction / deal-update）。

### 5. 不写入

回顾本身是只读的。除非用户在回顾中提供新信息（"哦对，X 那个上周签了，忘记记")，再触发对应 flow。

## 不要做

- **不要列原始数据**。"本周有 15 个 activity" 不是回顾，是 dump
- **不要超过 3 个"建议聚焦"**。多了等于没建议
- **不要把所有 alert 都列出**。低优先级合并 / 略过
- **不要在没真实数据时编造"洞察"**。如果一周很安静，就说"本周比较安静，2 个邮件回复，无 deal 变化"
- **不要假装积极**。如果数据显示丢了 3 个 deal，告诉用户事实，不要避而不谈

## 失败处理

|情况|处理|
|---|---|
|数据为空（新用户 / 真的没事）|"本周没什么动静。要不要我帮你看看有谁很久没联系了？"|
|Events 过多|只显示 top 类别 + 总数，详细让用户按需查|
|Alert scan 失败|跳过 alert 部分，告诉用户"alert 扫描出错了，但其他数据正常"|

## 示例对话

**示例 1：常规周回顾**

```
用户: 这周怎么样？
你:   (...并行调用 5 个 CLI...)

      📊 本周回顾 (5/22 - 5/28)

      **大事件**
      ✅ ABC 项目推进到 negotiation（5/20）
      ⚠️ XYZ 项目预算被砍 5K（5/24）

      **Pipeline**
      - lead: 3 (¥0 - 待估值)
      - qualified: 5 (¥230K)
      - proposal: 4 (¥180K)
      - negotiation: 2 (¥80K) ← 包括上面那个 ABC

      即将成交（7 天内）: 2 个，总额 ¥110K

      **需要关注**
      1. 王五合同已发 14 天未签（VIP）
      2. 3 个 proposal 阶段 deal 超过 10 天无互动
      3. 冯七邮件提到要 demo 但没安排时间

      **多 Agent 动态**
      - 邮箱 Agent 自动捕获了 8 封新邮件，关联到 5 个 contact
      - 社交 Agent 标记了张三在 Twitter 一条 mention（5/26）

      **建议聚焦**
      1. 跟进王五合同（VIP + 大额 + 14 天）
      2. 给冯七排 demo（明确意向但没节点）
      3. 处理本周积压的 2 个新联系人（source 未确认）

      要做哪一件？
用户: 做第 1 个。
你:   (转入 prep-followup flow for 王五)
```

**示例 2：安静的一周**

```
用户: 本周总结。
你:   本周比较安静：
      - 3 封邮件互动（张三 / 李四 / 王五）
      - Pipeline 无变化
      - 无新增联系人
      - 1 个 alert: 冯七已 18 天无互动

      要不要我帮你看看哪些客户很久没碰了？或者准备一下下周的跟进？
```

**示例 3：糟糕的一周**

```
用户: 这周如何？
你:   说实话，不太理想：
      - ❌ ABC 项目丢了（lost on 5/24，原因：竞品价格低）
      - ❌ XYZ 项目 scope 缩水 40%
      - ⚠️ 4 个 deal 整周无互动

      但有 1 个亮点：
      - ✅ 新增了 2 个 qualified 阶段联系人（黄五 referral）

      **建议这周做的事**
      1. 复盘 ABC 丢单原因（我可以拉详细 timeline）
      2. 处理 4 个 stale deal 中至少 2 个
      3. 跟进黄五推荐的 2 个新联系人

      要从哪个开始？
```

````

---

## flows/who-is-this.md

```markdown
# Flow: 识别陌生名字

## 何时触发
- 用户在对话中提到一个名字，你不确定是谁（"和张三的事怎样了"——但你不确定他指哪个张三）
- 用户问"X 是谁" / "我和 X 聊过吗"
- 你在处理别的任务时看到一个名字，不确定是不是已知 contact
- 另一个 Agent（邮箱 / 社交）转过来一个名字让你查

## 何时不触发
- 用户给了足够信息明确指向（"DEF 公司的张三" → 直接搜 + 用，不需要这个 flow）
- 完全没头绪的名字且没上下文（"X 是谁" 没任何线索 → 告诉用户"我需要更多信息"）

## 这个 flow 和 prep-followup 的区别

- **who-is-this**：解决"这是哪个人" - 输出是身份识别
- **prep-followup**：解决"准备跟这个人怎么谈" - 输出是行动准备

如果用户既想识别又想准备，先 who-is-this 确认对象，再 prep-followup 准备。

## 标准流程

### 1. 搜索

```bash
agentcrm contact search "<name 或 partial>" --format json
````

四种结果：

|情况|处理|
|---|---|
|0 个结果|走步骤 2（unknown）|
|1 个明确匹配|走步骤 3（confirm）|
|多个候选|走步骤 4（disambiguate）|
|模糊匹配（拼音不准、缩写）|走步骤 4（disambiguate）|

### 2. 找不到时

不要立刻说"没有这个人"。先扩大检索：

```bash
# 在 memory 里搜（可能在 global memory 提过但还没正式建档）
agentcrm memory recall "<name>" --top-k 5 --format json

# 在 activity 里搜 summary（可能在 note 里出现过）
agentcrm activity list --search "<name>" --limit 10 --format json
```

如果还是没有：

> "我没在 CRM 里找到「X」。能告诉我他的邮箱、公司、或你是从哪儿认识的吗？或者要不要我先建一条占位 memory，等下次见到联系方式再正式建档？"

不要默认建 contact——信息不足时建档只会增加垃圾数据。

### 3. 一个明确匹配

简短确认 + 给关键定位信息：

> "张三 (ABC 公司, CTO, 最近互动 8 天前关于 Q3 项目)。是这个张三吗？"

如果用户接下来要做什么（跟进 / 记录 / 查询），不要等他重新指明，直接用这个 contact_id 进入下一个 flow。

### 4. 多个候选

列出区分字段，让用户选：

> 我找到 3 个匹配「张三」的联系人：
> 
> 1. 张三 (ABC 科技, CTO) - 上次互动 8 天前
> 2. 张三 (DEF 公司, 销售) - 上次互动 3 个月前
> 3. 张三明 (HIJ 公司, CEO) - 上次互动 1 年前
> 
> 你说的是哪个？

如果你能从上下文推测（"上次邮件那个张三" → 最近互动的那个），主动推测并请用户确认：

> 你刚提到上次邮件——我猜是 ABC 科技那个张三？

### 5. 给下游 flow 用

once 用户确认了，记下 contact_id，转入下一个 flow（log-interaction / prep-followup / etc）。

## 不要做

- **不要在没确认对象时进行写入操作**。先识别再行动
- **不要把 memory recall 结果当成正式 contact**。memory 里提过一个名字 ≠ 这人在 CRM 里
- **不要凭名字猜性别 / 国籍 / 行业**。用 CRM 里的实际数据
- **不要在多候选时随便选最近的那个**。让用户明确

## 失败处理

|情况|处理|
|---|---|
|Search 出错|重试一次；仍失败告诉用户"CRM 查询出错，能直接告诉我他的邮箱吗？"|
|用户说"都不是"|当作 unknown 处理（步骤 2）|

## 示例对话

**示例 1：单一匹配**

```
用户: 张三最近有什么动静？
你:   (调 contact search "张三" → 1 个: cnt_zhang "ABC 科技, CTO")
      张三 (ABC 科技, CTO) — 上次互动是 5/20 的电话，他说 CEO 同意 5 万方案要求分期。
      要看完整时间线还是准备下次跟进？
```

**示例 2：多候选**

```
用户: 帮我准备见李四。
你:   (调 contact search "李四" → 2 个)
      有两个李四：
      1. 李四 (XYZ 公司, CEO) - 最近互动 3 天前
      2. 李四 (KLM 公司, 产品经理) - 最近互动 2 个月前

      是 XYZ 的那个吗？（最近聊得多的）
用户: 对。
你:   (转入 prep-followup flow)
```

**示例 3：找不到**

```
用户: 黄六最近发我什么了吗？
你:   (调 contact search "黄六" → 0)
      (调 memory recall "黄六" → 0)
      (调 activity list --search "黄六" → 0)

      我没找到「黄六」。能告诉我他的邮箱或公司吗？或者你想搜的是其他人（拼音相近的）？
用户: 是黄六，不是别人。他没在 CRM 里？
你:   没有。要不要现在加一个？告诉我他的邮箱 / 微信 / linkedin 任一个就行。
```

**示例 4：从 memory 里反查**

```
用户: 那个"AI 创业的陈七"还有联系吗？
你:   (调 contact search "陈七" → 0)
      (调 memory recall "陈七 AI 创业" → 1 个 global memo: "黄五的朋友陈七，AI 创业公司 CEO，待补联系方式")

      陈七还没正式建档（在 global memory 里有一条："黄五的朋友，AI 创业公司 CEO，待补联系方式"，记于 2026-03-15）。

      也就是说我们一直没拿到他联系方式。要不要现在通过黄五要一下？
```

````

---

## flows/handle-alert.md

```markdown
# Flow: 处理提醒

## 何时触发
- 用户主动问"有什么要看的" / "有什么 alert"
- 你在会话开始时调用 `agentcrm alert scan` 拿到了若干 pending alert
- weekly-review flow 里列出了 alert，用户说"处理 #N"
- cron 起的 daily scan 写入了 `pending.json`，用户进会话后第一句问

## 何时不触发
- 用户没主动问且没明显时机点（不要主动 spam alert）
- 单一 contact / deal 的具体问题（走对应 flow，不是这个）

## 标准流程

### 1. 拉 alert

```bash
agentcrm alert scan --since-last-scan
agentcrm alert list --format json
````

注意 alert 的来源（rule name）：

- `stale_deal`：deal 超过 N 天无互动
- `closing_deadline`：deal 预计成交日临近
- `vip_silence`：VIP 客户长期无互动
- `stale_memory`：有 memo 过期需要更新
- `<user-defined>`：用户自定义规则触发

### 2. 优先级排序

不是所有 alert 都同等重要。按以下优先级展示：

|优先级|类型|
|---|---|
|P0|closing_deadline 且 < 3 天|
|P0|VIP + 大额 deal 同时触发 stale|
|P1|closing_deadline 4-7 天|
|P1|VIP_silence|
|P1|stale_deal 大额（top 20%）|
|P2|stale_deal 一般|
|P2|stale_memory|
|P3|user-defined 低风险规则|

### 3. 同类合并

如果有多个同类 alert（"5 个 deal 都超过 14 天无动"），合并显示：

> ⚠️ 5 个 deal 在 proposal 阶段超过 14 天无互动（总金额 ¥230K）
> 
> - ABC Q3 项目（21 天）
> - DEF 二期（19 天）
> - GHI 平台（17 天）
> - ... 还有 2 个
> 
> 想批量处理还是单个看？

### 4. 每条 alert 给可执行的下一步

不要只是说"X 超过 14 天"。说**该怎么办**：

|Alert 类型|建议下一步|
|---|---|
|stale_deal|"起草一封跟进邮件 / 安排电话"|
|closing_deadline|"确认状态 / 推一下进度 / 调整 expected_close"|
|vip_silence|"发个问候 / 借由头联系（生日 / 行业新闻）"|
|stale_memory|"联系本人确认信息是否仍成立"|

### 5. 让用户选动作

每条 alert 后给明确的可选动作：

> 1. 跟进（起草邮件 → 让你确认 → 发送）
> 2. 推迟（snooze 3/7/14 天）
> 3. 忽略（dismiss，但 14 天后还会再出现）
> 4. 不再提醒（dismiss permanent，建议附理由）

### 6. 执行 dismiss / snooze

用户选择后：

```bash
# 推迟
agentcrm alert snooze <id> --days 7

# 忽略（这次）
agentcrm alert dismiss <id>

# 永久忽略 + 理由
agentcrm alert dismiss <id> --permanent --reason "对方在休假，自然恢复后联系"
```

如果用户选"跟进"，转入 prep-followup flow。

### 7. 总结

处理完一批 alert 后：

> 已处理: 3 个跟进（你已确认起草发送）、2 个推迟（7 天后再看）、1 个永久忽略。

## 不要做

- **不要把所有 alert 一股脑丢出来**。分优先级，先看 P0
- **不要把 stale memory 当紧急事**。这是质量维护，不是行动信号
- **不要自动 dismiss**。即使你觉得某 alert 不重要，让用户决定
- **不要在用户没说"处理"时 snooze**。"先放着"不等于"snooze 7 天"——可能他就想看看，不操作

## 失败处理

|情况|处理|
|---|---|
|Alert scan 失败|告诉用户"扫描出错，但我能列出上次 cron 写入的 pending"，用 `alert list`|
|没有任何 alert|"没有 pending alert，今天清净 ✓"|
|大量 alert（>20）|只展示 P0+P1，告诉用户"还有 N 个 P2+P3，要不要也看？"|

## 示例对话

**示例 1：常规处理**

```
用户: 有什么要看的？
你:   (调 alert scan + list)

      3 个事项要看：

      🔴 **P0**: ABC Q3 项目（VIP + ¥50K）22 天无互动，且预计 7/1 成交（34 天后）

      🟡 **P1**:
      - 王五（VIP）18 天未联系
      - DEF 二期 ¥80K 在 proposal 阶段 15 天

      🟢 **P2**:
      - 2 条 memory 过期（关于"张三决策周期"）

      从 P0 开始？
用户: 嗯。
你:   ABC 项目 22 天没动。建议起草一封短邮件问下进度，同时附上你提到的分期方案细节。
      可选：
      1. 我起草邮件给你看
      2. 安排电话（建议时间？）
      3. 推迟（你确认下周再看？）
      4. dismiss（如果实际有进展只是没记录）
用户: 1。
你:   (调用邮件起草 — 这是另一个 skill / 主对话 Agent 的事)
      ... 起草内容 ...
      要发吗？
用户: 改一下，第二段...
你:   (修改 → 发送 → log-interaction 记 activity)
      已发送并记下了 activity。这条 alert 自动 dismiss 了（因为产生了新互动）。

      下一个？王五还是 DEF？
```

**示例 2：批量同类**

```
用户: 看 alert。
你:   主要是 5 个 proposal stage deal 都超过 14 天无动（共 ¥230K）：
      - ABC Q3 (21d, ¥50K)
      - DEF 二期 (19d, ¥80K)
      - GHI 平台 (17d, ¥60K)
      - JKL 工具 (15d, ¥30K)
      - MNO 咨询 (14d, ¥10K)

      建议：要不要批量起草一封 "check-in" 模板邮件？每封我会用各自上下文定制。
      或者你想一个一个看 / 全部 snooze 7 天 / 其他？
用户: 批量草稿。
你:   好，我会用每个的上下文（最后一次互动 + deal 详情）定制 5 封。先给你看 ABC 的，模式确认后批量出剩下 4 封。
      ... (起草 ABC) ...
```

**示例 3：清净**

```
用户: 有什么要处理的吗？
你:   (alert scan → 0)
      没有 pending alert。要不要我主动看看哪些客户很久没碰了？
```

````

---

## flows/cleanup-dups.md

```markdown
# Flow: 清理重复联系人

## 何时触发
- 用户怀疑有重复（"我刚才好像加了两个张三"）
- 你在 search 时发现明显重复（同 email、同 phone、不同 contact）
- doctor 命令报告了潜在重复
- 用户主动让你"清理一下 contact"

## 何时不触发
- 同名不同人（"张三 (ABC)" vs "张三 (DEF)" — 这是两个人不是重复）
- 信息冲突但人不一样（除非有强证据是同人，否则不合并）
- 大批量定期清理（这应该是用户主动操作，不是 Agent 自发的）

## 这个 flow 是高风险

`contact merge` 是不可逆的（虽然审计日志可查，但用户体感是数据消失了）。
**必须在 merge 前明确用户确认**。

## 标准流程

### 1. 找出候选重复

CLI 提供检测：

```bash
agentcrm doctor --check dups --format json
````

返回结构：

```json
{
  "dup_groups": [
    {
      "reason": "same_email",
      "contacts": [
        {"id": "cnt_a", "name": "张三", "email": "zhang.san@abc.com", "last_activity_at": "2026-05-20"},
        {"id": "cnt_b", "name": "张三", "email": "zhang.san@abc.com", "last_activity_at": "2026-03-15"}
      ]
    },
    {
      "reason": "same_phone",
      "contacts": [...]
    }
  ]
}
```

或者手动搜（用户说"我加了两个张三"）：

```bash
agentcrm contact search "张三" --limit 10 --format json
# 然后挑出名字 + 公司一致的
```

### 2. 判断真重复 vs 同名异人

**强信号（同人）**：

- 同 email
- 同 phone
- 同 social handle（twitter / linkedin）
- 同 name + 同 company + 同 title

**弱信号（同人）**：

- 同 name + 同 company
- 同 email 域名 + 同 name
- 信息互补（A 有 email 没 phone，B 有 phone 没 email，name 相同）

**反信号（不要合并）**：

- 同 name 但不同 company
- 同 name 但 last_activity 隔了好几年（可能 historical archive 误录）
- 用户明确说"是两个人"

### 3. 展示给用户确认

不要悄悄 merge。把候选并排展示：

> 我检测到可能重复：
> 
> ```
> Contact A: cnt_001
>   名字: 张三
>   邮箱: zhang.san@abc.com
>   公司: ABC 科技
>   职位: CTO
>   最近互动: 2026-05-20
>   activity 总数: 23
>   memory 条数: 5
> 
> Contact B: cnt_002
>   名字: 张三
>   邮箱: zhang.san@abc.com  ← 与 A 相同
>   公司: ABC 科技             ← 与 A 相同
>   职位: (空)
>   最近互动: 2026-03-15
>   activity 总数: 2
>   memory 条数: 0
> ```
> 
> 看起来 B 是 A 的早期重复。要把 B 合并到 A 吗？
> 
> - 合并后 B 的 2 条 activity 会归到 A
> - B 的 contact_id 不会被复用，merged_into 标记保留可追溯
> - 这个操作可在 audit log 里看到，但用户层面不可撤销

### 4. 选择保留方向

通常合并到"信息更全 / 互动更多"的那个。明确告诉用户：

- A 保留（B 合并到 A）
- B 保留（A 合并到 B）
- 都不动

让用户选，不要自己拍板。

### 5. 执行 merge

```bash
agentcrm contact merge <keeper_id> <to_merge_id> \
  --reason "<明确的合并原因>"
```

CLI 自动：

- 把 to_merge 的所有 activity 重新关联到 keeper
- 把 to_merge 的 memory 迁移到 keeper（如果有冲突走类似 propose 流程）
- 合并 frontmatter（keeper 优先，to_merge 的非空字段补充进 keeper）
- to_merge 的 contact 文件改名加 `.merged-into-<keeper_id>.md` 后缀（保留可追溯）
- 触发 `contact.merged` 事件

### 6. Memory 冲突处理

如果两个 contact 各自有 memory 且互相冲突，CLI 不会自动决策。它会：

```json
{
  "status": "memory_conflicts",
  "conflicts": [
    {
      "keeper_memo": "偏好周三沟通",
      "merged_memo": "偏好周五沟通"
    }
  ]
}
```

你需要逐条让用户决定（类似 update-fact 的 propose/commit 模式）。

### 7. 报告结果

> 合并完成：
> 
> - cnt_002 已合并到 cnt_001
> - 2 条 activity 已迁移
> - 0 条 memory 冲突
> - 历史在 audit log 里可查

## 不要做

- **绝对不要自动 merge**。即使两个 contact 邮箱完全一样
- **不要把同公司同 title 的人当重复**（"ABC 公司的销售"可能有好几个）
- **不要在 merge 时静默丢数据**。任何 frontmatter 冲突、memory 冲突，都让用户决定
- **不要批量 merge 多组**。一次处理一组，每组单独确认
- **不要在用户犹豫时帮他下决心**。"看起来是同一个人吧？" — 不行。让用户明确说"是"

## 失败处理

|情况|处理|
|---|---|
|merge 时 CLI 拒绝（比如有 in-flight deal 关联）|CLI 返回详情，告诉用户"需要先把 deal X 的关联清理"，不要硬合并|
|用户中途反悔|告诉用户"还没执行，没影响"|
|Merge 后用户说"合错了"|告诉用户审计日志可看到 merge 详情；恢复需要手动从 audit log 重建（无 unmerge 命令）。这是为什么必须 merge 前 100% 确认|

## 示例对话

**示例 1：明显重复**

```
用户: 我刚才好像加了两个张三。
你:   (调 contact search "张三" --limit 10)
      (找到两个：cnt_001 完整版，cnt_002 只有 name + email)
      检测到可能重复：

      Contact A: cnt_001
        张三, ABC 科技, CTO
        邮箱: zhang.san@abc.com
        最近互动: 5/20 (23 条 activity, 5 条 memory)

      Contact B: cnt_002
        张三 (无公司无 title)
        邮箱: zhang.san@abc.com    ← 与 A 同
        最近互动: 5/28 (1 条 activity, 0 memory)

      看起来 B 是你刚才加的，但和 A 是同一人。要合并 B → A 吗？B 的 1 条 activity 会归到 A，B 这条记录会归档。
用户: 嗯。
你:   (调 contact merge cnt_001 cnt_002 --reason "同 email，B 为今日重复创建")
      合并完成。1 条 activity 已迁移到 cnt_001。
```

**示例 2：拒绝合并**

```
用户: doctor 报告说"张三 ABC" 和"张三 DEF" 可能重复。
你:   它们 email 不同、公司不同。我倾向认为这是两个人（同名）。要不要我跳过这条？
用户: 等等，让我看看 DEF 那个。
你:   (调 contact get cnt_zhang_def)
      ...
      看起来确实是不同人。我建议 dismiss 这个 doctor 警告。要在 doctor 里加白名单避免下次再报吗？
```

**示例 3：memory 冲突**

```
用户: 合并 cnt_001 和 cnt_002。
你:   (调 contact merge → 返回 memory_conflicts)
      合并前有 1 条 memory 冲突要处理：

      A (cnt_001) 里: "偏好周三下午沟通" (写于 2025-04)
      B (cnt_002) 里: "偏好周五下午沟通" (写于 2026-03)

      B 的更新，但你之前合并 update 时可能没同步。怎么处理？
      1. keep A 的（保留周三）
      2. keep B 的（用周五取代周三）
      3. keep-both（两条都保留）
用户: 用 B 的。
你:   (调 memory commit 替代 + 执行 merge)
      合并完成，memory 已用 B 版本更新。
```

```

---

## flows/handle-alert.md 中遗漏的 update-fact 链接修正

刚才 update-fact.md 是单独写的，handle-alert 中 stale_memory 类提到"联系本人确认信息是否仍成立"——实际上这会触发 prep-followup（去确认）或直接 update-fact（如果用户已经知道新值）。这个串联在每个 flow 里已经隐含说明，不需要额外修订。

---

## 9 个 flow 写完了。一些关键点总结：

**贯穿 9 个 flow 的统一设计**：

1. **每个 flow 都有"不要做"清单**——这比"标准流程"还重要。LLM 容易自作主张，明确禁令比正向引导更有效
2. **示例对话用真实长度**——不是"用户说 X 你回 Y"，是带 CLI 调用、用户中间打断、修正的完整对话
3. **failure 处理写实**——CLI 怎么报错、怎么 recover、什么情况要问用户

**flow 之间的衔接**：

- `who-is-this` 是几乎所有其他 flow 的前置（"先识别再行动"）
- `update-fact` 是被其他 flow 调用的子流程（log-interaction 识别到事实变化 → 调用 update-fact）
- `prep-followup` 是 alert 处理后的常见后续
- `cleanup-dups` 是最独立的，但和 new-lead 关联（new-lead 时的重复检测可能转到 cleanup-dups）

**v1.0 完整版的判断标准**：每个 flow 都应该通过一个真实评测——用 Claude/OpenClaw 跑一遍场景对话，看 Agent 能不能：

1. 选到正确的 flow（不是其他）
2. 调用正确的 CLI 命令（参数对、actor 对、dedupe-key 对）
3. 按 "不要做" 清单避免常见错误
4. 失败时正确 recover 而不是假装成功

需要的话我可以接着把这 9 个 flow 编译成实际可放进 `skills/agentcrm/flows/` 的文件（每个一个 `.md`），打成一个 tarball；或者写一份"flow 评测脚本"，定义每个 flow 的 5-10 个测试 scenario，用真实 LLM 跑通的判定标准。
```