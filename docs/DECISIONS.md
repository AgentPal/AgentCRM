# Architecture Decision Records

This file documents key architectural decisions made during AgentCRM
development. Each entry is numbered chronologically.

---

## ADR-001: Remove unimplemented PreservePatterns config field

**Date**: 2026-05-29
**PR**: refactor/remove-dead-preserve-patterns
**Status**: Accepted

### Context

`MemoryConfig.PreservePatterns` was a schema field intended to protect
memories matching text patterns from automatic decay. It was defined in
`internal/model/config.go` with default value `["决策*", "偏好*", "*性格*"]`.

The matching logic was **never implemented**. `DecayScan()` only checks
each memo's per-memo `Decay` field (`"never" | "30d" | "90d" | "180d"`),
and never reads `PreservePatterns`.

### Decision

Remove the field entirely — both the struct definition and the default
value — rather than keep it as dead schema.

### Rationale

- A non-functional config field creates false sense of control
- Removing now (pre-release) has zero cost; removing post-v1.0 would be
  a breaking change
- Per-memo `decay: "never"` already provides per-memory protection
- Pattern-based batch protection is YAGNI for v1.0

### Consequences

- `config.memory.preserve_patterns` translation key removed from
  `messages_{en,zh}.json`
- Config example in technical docs updated to remove field
- Users who manually set this field in `config.json` will see no error
  (unknown keys are silently ignored by JSON unmarshal)

### Future

If scope-pattern batch protection is needed in v0.2+, re-add the field
AND implement the matching logic in `DecayScan` together, never one
without the other.
