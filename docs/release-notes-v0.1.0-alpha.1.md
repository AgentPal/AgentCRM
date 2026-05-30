# AgentCRM v0.1.0-alpha.1 — Release Notes

**Release date:** 2026-05-30

AgentCRM is a local-first, file-based, zero-process customer relationship memory system. Distributed as a CLI tool, it gives every AI agent an immediate persistent, readable, backup-friendly customer memory.

## Alpha Notice

This is an **alpha pre-release**. Expect:
- Possible bugs and edge cases
- API instability: CLI flags, output format, and storage layout may change before v1.0.0 stable
- Limited cross-platform validation (see Known Limitations below)

## Installation

```bash
curl -sfL https://github.com/AgentPal/AgentCRM/releases/download/v0.1.0-alpha.1/install.sh | sh
```

Or download the binary for your platform from the [releases page](https://github.com/AgentPal/AgentCRM/releases/tag/v0.1.0-alpha.1).

## Quick Start

```bash
# Initialize
agentcrm init

# Create a contact
agentcrm contact upsert --actor alice --email user@example.com --name "Jane Doe"

# Log an interaction
agentcrm activity log --actor alice --contact cnt_xxx --type email \
  --summary "Discussed project timeline" --dedupe-key "<msgid@example.com>"

# Write a memory
agentcrm memory write --actor alice --scope contact:cnt_xxx --text "Prefers morning calls"

# View your data
agentcrm contact list
agentcrm activity list --contact cnt_xxx
agentcrm deal list
```

## What's Included

- **CLI commands:** init, version, reindex, doctor, contact (upsert/get/list/update/search/merge/history), deal (create/get/list/update/history/summarize), activity (log/list/timeline), memory (write/recall/list/propose/commit/decay/forget), events (poll/watch/ack/subscribers), alert (scan/list/dismiss/snooze), import, export
- **Storage:** Markdown + JSONL files + SQLite index with FTS5 full-text search
- **i18n:** English and Chinese translations
- **Multi-agent collaboration:** Actor tracking, event log with subscriber cursors, deduplication keys
- **Skill packages:** Full agent skill + bridge skill + 9 workflow flows

## Known Limitations

- i18n: `--help` command descriptions appear in English regardless of `AGENTCRM_LANG`. Runtime output respects the language setting. See `docs/i18n-design.md` §13 for details.
- Cross-platform: only Windows (amd64) has been fully tested. macOS and Linux compile but have not been exercised in CI.
- `go vet`: 6 format-string warnings — all safe, guarded by `TestPlaceholderConsistency`.
- CLI argument names and output format may change.

## Feedback

Report issues at: https://github.com/AgentPal/AgentCRM/issues
