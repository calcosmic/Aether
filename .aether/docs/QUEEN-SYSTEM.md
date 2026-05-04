# QUEEN.md System

The QUEEN.md system is Aether's wisdom feedback loop - a mechanism for capturing, validating, and propagating learnings across colonies and within a single repo.

## Overview

The Queen has two scopes:

- Global Queen: `~/.aether/QUEEN.md`, shared across repos on this machine. It
  stores cross-colony wisdom and user preferences.
- Local Queen: `.aether/QUEEN.md`, scoped to the current repo. It stores the
  project charter, repo-specific lessons, and local preferences.

Colony-prime reads the global Queen first, then reads the local Queen. Local
entries extend global entries; they do not replace the global file.

## File Location

```
~/.aether/QUEEN.md      # global hub wisdom and preferences
.aether/QUEEN.md        # repo-local project wisdom
```

## Structure

### 📜 Philosophies

Core beliefs validated through repeated successful application across multiple colonies.

**Threshold:** 1 successful validation required for promotion

Example:
```markdown
- **colony-name** (2026-02-15T13:08:24Z): Test-driven development ensures quality
```

### 🧭 Patterns

Validated approaches that consistently work. These represent discovered best practices.

**Threshold:** 1 successful validation required for promotion

Example:
```markdown
- **colony-name** (2026-02-15T13:08:28Z): Always validate inputs
```

### ⚠️ Redirects

Anti-patterns to avoid. Approaches that have caused problems.

**Threshold:** 1 failed validation required for promotion

Example:
```markdown
- **colony-name** (2026-02-15T13:08:31Z): Never skip security checks
```

### 🔧 Stack Wisdom

Technology-specific insights detected through codebase analysis.

**Threshold:** 1 validation required for promotion

Example:
```markdown
- **colony-name** (2026-02-15T13:08:36Z): Use jq for JSON in bash
```

### 🏛️ Decrees

User-mandated rules that override other guidance.

**Threshold:** 0 validations required (immediate promotion)

Example:
```markdown
- **colony-name** (2026-02-15T13:08:40Z): All code must have tests
```

### 📊 Evolution Log

Track how wisdom has evolved over time.

## Commands

### queen-init

Initialize global and repo-local QUEEN.md files from the standard template if
they do not exist.

```bash
aether queen-init
```

**Returns:**
```json
{"created": true, "path": "~/.aether/QUEEN.md", "local_created": true, "local_path": ".aether/QUEEN.md"}
```

### queen-read

Read hub-global QUEEN.md content. Worker priming uses colony-prime to merge
global and local Queen wisdom.

```bash
aether queen-read
```

**Returns:**
```json
{
  "metadata": {
    "version": "1.0.0",
    "last_evolved": "2026-02-15T13:08:40Z",
    "colonies_contributed": ["colony-a"],
    "promotion_thresholds": {...},
    "stats": {...}
  },
  "wisdom": {
    "philosophies": "...",
    "patterns": "...",
    "redirects": "...",
    "stack_wisdom": "...",
    "decrees": "..."
  },
  "priming": {
    "has_philosophies": true,
    "has_patterns": true,
    ...
  }
}
```

### queen-promote

Promote a learning or preference to hub-global QUEEN.md.

```bash
aether queen-promote <type> <content> <colony_name>
```

**Types:** `philosophy`, `pattern`, `redirect`, `stack`, `decree`

**Example:**
```bash
aether queen-promote pattern "Always validate inputs" "my-colony"
```

### queen-write-learnings

Write phase learnings to repo-local QUEEN.md.

```bash
aether queen-write-learnings --learnings '[{"claim":"Prefer focused tests before full suites"}]'
```

## Integration with Commands

### init.md

Calls `queen-init` after bootstrap to ensure global and local Queen files exist.

### build.md

Uses colony-prime before spawning workers. Colony-prime reads global Queen
wisdom, local Queen wisdom, and preferences from both files.

### continue.md

After verification, phase learnings are written to repo-local QUEEN.md using
`queen-write-learnings`.

### seal.md

Before archiving, significant cross-colony patterns can be promoted to the
hub-global Queen and Hive Brain.

### entomb.md

Before creating the chamber, validated local learnings can be preserved in the
repo-local Queen file.

## Promotion Thresholds

| Type | Threshold | Rationale |
|------|-----------|-----------|
| Philosophy | 1 | Promote validated guidance quickly |
| Pattern | 1 | Promote reusable practices quickly |
| Redirect | 1 | Promote anti-pattern protection immediately |
| Stack | 1 | Tech insights are domain-specific |
| Decree | 0 | User mandates are immediate |

## Metadata

The QUEEN.md file includes a METADATA block at the end:

```html
<!-- METADATA
{
  "version": "1.0.0",
  "last_evolved": "2026-02-15T13:08:40Z",
  "colonies_contributed": ["colony-a"],
  "promotion_thresholds": {
    "philosophy": 1,
    "pattern": 1,
    "redirect": 1,
    "stack": 1,
    "decree": 0
  },
  "stats": {
    "total_philosophies": 1,
    "total_patterns": 1,
    ...
  }
}
-->
```

## Best Practices

1. **Don't manually edit QUEEN.md** - Use Queen runtime commands to preserve formatting
2. **Validate before promoting** - Only promote learnings that have been tested
3. **Use descriptive colony names** - Helps track wisdom origins
4. **Read wisdom at build start** - Workers benefit from global and local knowledge
5. **Review periodically** - Some wisdom may become outdated as the system evolves

## See Also

- `/ant-init` - Initializes QUEEN.md for new colonies
- `/ant-build` - Reads QUEEN.md wisdom for worker priming
- `/ant-continue` - Promotes validated learnings
- `/ant-seal` - Promotes final colony wisdom
- `/ant-entomb` - Promotes wisdom before archiving
