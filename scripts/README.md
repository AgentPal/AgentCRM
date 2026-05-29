# scripts/ — Development Scripts

This directory contains one-shot Node.js scripts used during AgentCRM
v1.0.0-dev development (PRs 6–11) to automate repetitive code migrations,
test fixes, and documentation updates.

## Convention

| Prefix   | Purpose                                              |
|----------|------------------------------------------------------|
| `migrate_*` | Bulk i18n.T() migration in Go source files         |
| `fix_*`     | Regression fixes or test assertion updates         |
| `add_*`     | Add translation keys or documentation sections     |

## One-shot vs Reusable

Most scripts here are **one-shot**: they were written for a specific PR,
run exactly once, and their work is done. They are kept as reference
because:

1. They document *how* a migration was done — more precise than a commit
   message.
2. v1/v2 pairs (e.g., `fix_events_pr8_bug.js` / `fix_events_pr8_bug2.js`)
   record real debugging history: the first attempt didn't handle CJK
   fullwidth characters, the second fixed it.
3. Some patterns (`migrate_*.js`, `add_i18n_keys_*.js`) are reusable
   templates if similar batch migrations are needed in the future.

## When to Clean Up

After each PR cycle, scripts whose work has been merged should be either:

- **Deleted** — if the pattern is unlikely to be reused.
- **Moved to `.claude/`** — if they document a process worth keeping
  long-term (e.g., the CJK fullwidth character fix workflow).

## Running a Script

```bash
node scripts/<script-name>.js
```

All scripts read/write files relative to the repository root.
