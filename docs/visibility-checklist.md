# Visibility / Public repo checklist

When this repository is made **public**, restore dynamic shields.io badges that
were replaced with static versions during the private phase. shields.io cannot
fetch GitHub metadata for private repositories.

## Dynamic badges to restore

| File | Current (static) | Restore to (dynamic) |
|---|---|---|
| `README.md` | `img.shields.io/badge/go-1.22+-blue` | `img.shields.io/github/go-mod/go-version/AgentPal/AgentCRM` |

## How to restore

```bash
# Go version badge
sed -i 's|img.shields.io/badge/go-1.22+-blue|img.shields.io/github/go-mod/go-version/AgentPal/AgentCRM|g' README.md
```

After restoring, verify by opening `README.md` in a browser preview — the badge
should display `1.22` (or whatever the current go.mod `go` directive says).
