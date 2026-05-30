# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.0-alpha.1] - 2026-05-30

### Added

#### Core CLI
- `init` — initialize data directory with config file and subdirectory structure
- `version` — display version info and product tagline
- `reindex` — rebuild SQLite index from Markdown/JSONL files
- `doctor` — check data integrity and health

#### Contact Management
- `contact upsert` — create or update contact (deduped by email)
- `contact get` — show contact details
- `contact list` — list contacts
- `contact update` — update specific fields (history auto-archived)
- `contact search` — search contacts
- `contact merge` — merge two contacts
- `contact history` — view field history with point-in-time reconstruction

#### Deal Management
- `deal create` — create deal with stage, amount, currency
- `deal get` — show deal details
- `deal list` — list deals with optional stage/owner/amount filters
- `deal update` — update stage/amount/other fields
- `deal history` — view stage and amount change history
- `deal summarize` — generate natural-language deal summary
- Stage tracking: lead → qualified → proposal → negotiation → won/lost

#### Activity Logging
- `activity log` — log interactions (email/call/meeting/chat/note/social)
- `activity list` — list/filter activities
- `activity timeline` — unified timeline (activities + deal changes)
- Deduplication via `--dedupe-key` (e.g. RFC822 message-id)

#### Memory System
- `memory write` — write a memory entry
- `memory recall` — recall memories by scope and text
- `memory list` — list all memories in a scope
- `memory propose` — propose new memory with semantic conflict detection
- `memory commit` — commit/resolve a proposal (supersede/keep-both/reject)
- `memory decay` — scan and expire old memories with configurable TTL
- `memory forget` — delete a memory

#### Event System (Multi-Agent Collaboration)
- `events poll` — poll for new events
- `events watch` — blocking event listener
- `events ack` — advance event cursor
- `events subscribers list` — list subscribers
- `events subscribers reset` — reset subscriber cursor
- Actor-based event deduplication (prevent self-triggering loops)

#### Alert / Rule Engine
- 4 built-in rules: stale deal, closing deadline, VIP silence, stale memory
- `alert scan` — scan data and generate alerts
- `alert list` — view pending alerts
- `alert dismiss` — dismiss (permanent or one-time)
- `alert snooze` — snooze for N days
- `rule add` — add custom YAML rules
- `rule list` — list rules
- `rule disable` — disable a rule

#### Import / Export
- `export` — export all data to JSON or CSV
- `import` — import contacts from vCard (.vcf) or CSV
- Supported CSV columns: name, email, company, title, phone, tags, source

#### Storage
- File-based: Markdown files for contacts/deals, JSONL for activities/events
- SQLite index with FTS5 full-text search
- `.index.db` rebuildable from files at any time
- Config file (`config.json`) with customizable options

#### Internationalization (i18n)
- English and Chinese (zh) translations
- `AGENTCRM_LANG` environment variable for language selection
- Runtime language switching via `SetLang()`
- English pluralization support via `Tn()` (e.g. "1 record" vs "3 records")
- Glossary enforcement: "contact" not "customer", "deal" not "opportunity"
- TestPlaceholderConsistency guards against printf placeholder mismatches

#### Skill Packages
- `agentcrm` — full skill package for Claude/OpenClaw/Codex/Hermes
- `agentcrm-bridge` — lightweight bridge skill
- 9 built-in workflow flows
- Multi-platform compilation (Claude, OpenClaw, Codex, Hermes)

#### Developer Experience
- GoReleaser cross-platform builds (linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, windows/amd64)
- GitHub Actions CI with lint, test, build
- `install.sh` one-line installation script
- 95%+ test coverage on model layer, 79% on store layer
- MIT License

### Known Limitations

- i18n: command descriptions in `--help` output always appear in English
  (set at init time, before language selection is available). Runtime output
  respects `AGENTCRM_LANG`. See `docs/i18n-design.md` §13 for details.
- Cross-platform testing: only Windows (amd64) has been fully tested.
  macOS and Linux builds are verified to compile but not exercised.
- `go vet ./...` reports 6 non-constant format string warnings — these are
  all safe (translated messages guarded by TestPlaceholderConsistency).
- CLI argument names and output format are not yet frozen (may change before
  v1.0.0 stable).
