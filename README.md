<div align="center">

# AgentCRM

**Customer brain for the OPC era · Lives in your home directory**

[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/github/go-mod/go-version/AgentPal/AgentCRM)](go.mod)
[![Skill Format](https://img.shields.io/badge/skill-Claude%20%7C%20OpenClaw%20%7C%20Codex%20%7C%20Hermes-purple)](#supported-agents)

A **shared customer brain** for your Claude / OpenClaw / Codex / Hermes agents.
Zero servers · Local files · One-line install.

[Quick Start](#quick-start) · [What it is](#what-it-is) · [Why you need it](#why-you-need-agentcrm) · [Docs](docs/) · [中文](README.zh.md)

</div>

---

## What it is

AgentCRM is a **local-first, file-based, zero-process** customer relationship memory system. Distributed as a Skill package plus a CLI binary, it gives every AI agent on your machine an immediate **persistent, readable, backup-friendly, cross-agent shared customer brain**.

It's not a cloud CRM. It's not a SaaS. **It's a folder in your home directory:**

```
~/AgentCRM/
├── config.json        # User configuration
├── contacts/          # One Markdown file per contact
├── deals/             # One Markdown file per deal
├── activities/        # Monthly-archived JSONL of interactions
├── memory/            # Long-term memories
│   ├── contacts/      #   per-contact memory files
│   └── deals/         #   per-deal memory files
├── events/            # Multi-agent collaboration event stream
├── rules/             # YAML rule configuration (user-defined)
├── alerts/            # Pending alerts from rule engine
├── proposals/         # Pending memory proposals
├── .index.db          # SQLite search index (rebuildable from files)
├── .subscribers/      # Actor cursors for event polling
└── .audit/            # Operation audit trail
```

Plus an `agentcrm` Skill package installed into your agent's skills directory, and an `agentcrm` CLI binary. **That's it.**

Your data, always on your machine — open it in any editor, sync via git, walk away anytime.

---

## Why you need AgentCRM

### The work is the same. The workers have changed.

Customer work used to be done by people in companies — sales reps filing tickets, account managers tracking deals, support staff logging tickets. **In the OPC (one-person company) era, those jobs go to agents:**

| Task | Before | Now |
|---|---|---|
| Add new contact | Sales assistant fills a form | Email agent captures automatically |
| Log interaction | Salesperson types it in | Various agents write in the background |
| Track changes | Account manager follows up | Monitoring agents push alerts |
| Prep follow-ups | Salesperson reads history | Agent synthesizes context |

But today's tools assume the wrong things:

- **Traditional CRM** (Salesforce / HubSpot / Attio) assumes "you have a team, they fill forms, they read dashboards" — none of which holds in OPC
- **AI-augmented CRM** adds a chat sidebar and auto-fill — same architecture, still "humans serving the system"
- **General agent memory** (Mem0 / Supermemory) doesn't speak customer relationship business — it can store "Zhang San is a CTO" but doesn't know how a lost deal should be handled

### Where AgentCRM fits

```
┌─ General memory systems (Mem0 / Supermemory) ────┐
│  Store any fact. Don't understand CRM workflows. │
└──────────────────────────────────────────────────┘

┌─ Traditional CRM (Salesforce / HubSpot / Attio) ─┐
│  Understands CRM. Assumes humans fill forms.     │
└──────────────────────────────────────────────────┘

┌─ AgentCRM ──────────────────────────────────────┐
│  Understands CRM + Built for agents + Local files│
└──────────────────────────────────────────────────┘
```

**In one sentence:** AgentCRM is what Mem0/Supermemory would look like if specialized for customer relationships and shipped as a local Skill instead of a cloud service.

### Three things you cannot get elsewhere

**1. Shared customer brain across agents**

Your email agent, social-listening agent, birthday agent, and main chat agent all read and write the same customer data. One agent writes, all agents see it immediately.

**2. Business grammar baked in**

Nine business flows — `new-lead`, `deal-update`, `prep-followup`, etc. — teach your agent **the order in which things should happen**, not just "how to call an API."

**3. 100% data ownership**

The files are the truth. Open them in Obsidian, grep them, git-sync them. No portability problem because the data never left in the first place.

---

## How it compares

|  | AgentCRM | Traditional CRM | General Memory |
|---|---|---|---|
| Primary user | AI agents | Humans | AI agents |
| Deployment | Local files + CLI | Cloud SaaS | Cloud SaaS / self-hosted |
| Storage format | Markdown + JSONL | Proprietary DB | Proprietary DB + vector |
| Domain knowledge | 9 CRM flows built-in | UI workflows | None (generic add/search) |
| Multi-agent sharing | Native | No | Single-agent scope |
| Data ownership | Your files | Vendor | Vendor |
| Temporal awareness | Field-level history | Basic timestamps | Improving |
| Monthly cost | $0 | $25–300/seat | $19–249 |

---

## Quick Start

### One-line install

```bash
curl -fsSL https://raw.githubusercontent.com/AgentPal/AgentCRM/main/install.sh | sh
```

The installer will download the `agentcrm` CLI binary and place it in `/usr/local/bin/`.

Then initialize:

```bash
agentcrm init
```

### Build from source

```bash
git clone https://github.com/AgentPal/AgentCRM.git
cd AgentCRM
go install ./cmd/agentcrm
```

Requires Go 1.22+.

### Verify

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

### Start using it

Open any of your AI agents (Claude, OpenClaw, Codex, etc.) and say:

> Use agentcrm to help me get started

The agent invokes the AgentCRM Skill and walks you through it. You don't need to learn any commands — the Skill teaches the agent how to do everything.

---

## Using it (real scenarios)

After install, **you never "open the CRM"**. You just keep talking to the agents you already use.

### Scenario 1: Auto-capture a new contact

```
You: Add the email I just got from Zhang San to the CRM
Agent: ✓ Added Zhang San (ABC Tech, CTO) and logged the email interaction.
        He mentioned wanting AI integration in Q3 — want me to add that
        to his long-term notes?
You: Yes
Agent: ✓ Noted: "Wants AI integration in Q3, he's the driving force."
```

### Scenario 2: Natural language query

```
You: What was the last quote we gave Li Si?
Agent: Li Si (XYZ Co.) has one in-flight deal worth 50,000 CNY.
        The most recent quote was a revision on 5/10 — split into two
        installments. Want to see the full timeline?
```

### Scenario 3: Proactive briefing

```
(When the agent opens a session)
Agent: Good morning. Three things to keep an eye on:
       1. Wang Wu's contract was sent 14 days ago, still unsigned (VIP)
       2. Zhao Liu's Q2 deadline is tomorrow
       3. Feng Qi mentioned wanting a demo but no time was scheduled

       Where would you like to start?
```

### Scenario 4: Multi-agent collaboration

```
(Overnight, the email agent captures new mail)
Email agent → agentcrm activity log --actor email-agent ...

(In the morning, the birthday agent checks today's birthdays)
Birthday agent → agentcrm contact list --field "birthday:05-28"
              → pushes a reminder

(You open Claude)
You: Anything happening with customers today?
Agent: The email agent captured 3 new emails overnight (linked to 2 contacts).
       Today is Zhang San's birthday (ABC Tech).
       Want me to draft a quick birthday note?
```

### Scenario 5: Just open the files

The data is yours. Take `~/AgentCRM/contacts/zhang-san.md`:

```markdown
---
id: cnt_01HX9...
name: Zhang San
emails: [zhang.san@abc.com]
company: ABC Tech
title: CTO
tags: [vip, technical-buyer]
last_activity_at: 2026-05-20T09:15:00Z
---

# Zhang San

CTO at ABC Tech. Owns the technical decision.

## Background
Introduced by Huang Wu. Driver of the Q3 AI integration project.
```

Open it in Obsidian, VSCode, Vim — any editor. Run `cd ~/AgentCRM && git init` to sync across devices. Run `grep -r "Q3" .` to search everything.

---

## How it works

### Three layers

```
┌─────────────────────────────────────────────────┐
│  Skill layer (domain knowledge)                  │
│  9 business flows: new-lead / deal-update / ...  │
│  Teaches agents "in scenario X, do Y in order Z" │
└─────────────────────────────────────────────────┘
                       ↕
┌─────────────────────────────────────────────────┐
│  CLI layer (atomic capabilities)                 │
│  agentcrm contact / deal / activity / memory ... │
│  Dependency-free Go binary, startup <50ms        │
└─────────────────────────────────────────────────┘
                       ↕
┌─────────────────────────────────────────────────┐
│  Data layer (your files)                         │
│  Markdown + JSONL + SQLite index (rebuildable)   │
│  Just one folder: ~/AgentCRM/                   │
└─────────────────────────────────────────────────┘
```

### Multi-agent shared blackboard

AgentCRM plays a **shared blackboard** role in your agent ecosystem. Agents don't talk to each other directly — they collaborate by reading and writing the same files:

```
   ┌──────┐   ┌────────┐   ┌────────┐   ┌─────────┐
   │ Main │   │ Email  │   │ Social │   │Birthday │
   │ Chat │   │ Agent  │   │ Agent  │   │ Agent   │
   └───┬──┘   └────┬───┘   └───┬────┘   └────┬────┘
       │           │           │             │
       │  CLI: read / write / subscribe      │
       ▼           ▼           ▼             ▼
   ┌─────────────────────────────────────────────┐
   │            AgentCRM (local files)            │
   │  contacts/ deals/ activities/ memory/ ...   │
   └─────────────────────────────────────────────┘
```

Every CLI call carries an `--actor` tag (which agent is writing). Writes generate events. Other subscribed agents pull events since their last cursor. **No central broker, no daemon, just files.**

### Memory system design

Inspired by the mature work in general agent memory systems, localized for the customer-relations domain:

- **Four-layer memory architecture**: static facts / semantic memory / episodic memory / working memory
- **Temporal validity**: every mutable field carries `_history`, answers questions like "which company did he work at in 2024?"
- **Propose/Commit**: new facts are automatically checked for conflicts before write, letting agents choose `supersede` / `keep-both` / `reject`
- **Multi-strategy retrieval**: FTS5 keyword + entity match + temporal filter
- **Selective forgetting**: every memo can be decayed; expired ones auto-archive

---

## Supported agents

| Agent | Skill format | Status |
|---|---|---|
| Claude Desktop / Claude Code | `SKILL.md` | ✅ v1.0 |
| OpenClaw | `SKILL.md` (with Chinese triggers) | ✅ v1.0 |
| Codex | `AGENTS.md` | ✅ v1.0 |
| Hermes | `SOUL.md` | ✅ v1.0 |
| Other Skill / MCP-compatible agents | Adapter in progress | 🚧 |

The CLI is universal — any agent that can run shell commands can use it. The Skill just teaches the agent **how to use it well**.

---

## Documentation

- [Getting Started](GETTING_STARTED.md)
- [Skill Package](skills/agentcrm/SKILL.md)
- [Nine Business Flows](skills/agentcrm/flows/)
- [CLI Reference](internal/cmd/root.go)
- [Data Model](internal/model/)
- [Design Doc](docs/)

---

## FAQ

<details>
<summary><b>Do I need internet?</b></summary>

The CLI itself is fully offline. Your agents (Claude / OpenClaw / etc.) need internet to call LLMs — that's between the agent and its provider. AgentCRM is not involved.

</details>

<details>
<summary><b>Does my data go to your servers?</b></summary>

No. AgentCRM has no servers. No telemetry. 100% of your data lives on your machine. Unless you configure a git remote yourself, not a single byte ever leaves your device.

</details>

<details>
<summary><b>How do I sync across devices?</b></summary>

We don't ship a sync service, but everything is git-friendly:

```bash
cd ~/AgentCRM
git init
git remote add origin git@github.com:you/my-crm-data.git
git add . && git commit -m "init"
git push
```

You can also symlink the folder to iCloud Drive / Dropbox / OneDrive.

</details>

<details>
<summary><b>Windows support?</b></summary>

Yes. Native Windows binary, PowerShell-compatible. WSL works too.

</details>

<details>
<summary><b>Is there a web UI?</b></summary>

**Intentionally no.** Design philosophy: do daily work through agents, do configuration and auditing through editors. Opening a Markdown file is faster than navigating a web UI.

</details>

---

## Roadmap

- [x] **v1.0** — CLI + Skill packages + 4 agent platforms + 9 core flows
- [ ] **v1.1** — More agent platform adapters, community flow repository
- [ ] **v1.2** — Optional local embedding for enhanced retrieval
- [ ] **v1.x** — Bi-directional sync adapters for third-party CRMs (HubSpot / Attio / Salesforce)

---

## License

[MIT](LICENSE) © 2026 AgentCRM Contributors

Your data, always yours.

---

<div align="center">

**[⬆ Back to top](#agentcrm)** · **[GitHub](https://github.com/AgentPal/AgentCRM)**

If AgentCRM is useful to you, please ⭐ Star — it's the simplest and most effective way to support us.

</div>
