# QUEEN.md System

The QUEEN.md system is Aether's wisdom feedback loop - a mechanism for capturing, validating, and propagating learnings across colonies.

## Overview

The Queen represents the accumulation of validated knowledge from all colonies. As colonies complete work, they promote significant patterns, decisions, and insights to QUEEN.md. Future colonies can then read this wisdom to benefit from previous experience.

## File Location

```
.aether/docs/QUEEN.md
```

## Structure

### 📜 Philosophies

Core beliefs validated through repeated successful application across multiple colonies.

**Threshold:** 5 successful validations required for promotion

Example:
```markdown
- **colony-name** (2026-02-15T13:08:24Z): Test-driven development ensures quality
```

### 🧭 Patterns

Validated approaches that consistently work. These represent discovered best practices.

**Threshold:** 3 successful validations required for promotion

Example:
```markdown
- **colony-name** (2026-02-15T13:08:28Z): Always validate inputs
```

### ⚠️ Redirects

Anti-patterns to avoid. Approaches that have caused problems.

**Threshold:** 2 failed validations required for promotion

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

Initialize QUEEN.md from template if it doesn't exist.

```bash
bash .aether/aether-utils.sh queen-init
```

**Returns:**
```json
{"created": true, "path": ".aether/docs/QUEEN.md", "source": "runtime/templates/QUEEN.md.template"}
```

### queen-read

Read QUEEN.md and return wisdom as JSON for worker priming.

```bash
bash .aether/aether-utils.sh queen-read
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

Promote a learning to QUEEN.md wisdom.

```bash
bash .aether/aether-utils.sh queen-promote <type> <content> <colony_name>
```

**Types:** `philosophy`, `pattern`, `redirect`, `stack`, `decree`

**Example:**
```bash
bash .aether/aether-utils.sh queen-promote pattern "Always validate inputs" "my-colony"
```

## Integration with Commands

### init.md

Calls `queen-init` after bootstrap to ensure QUEEN.md exists for the colony.

### build.md

Calls `queen-read` before spawning workers to inject wisdom into worker prompts.

### continue.md

After verification, promotes validated learnings to QUEEN.md using `queen-promote`.

### seal.md

Before archiving, promotes significant patterns and decisions to QUEEN.md.

### entomb.md

Before creating the chamber, promotes validated learnings to QUEEN.md.

## Promotion Thresholds

| Type | Threshold | Rationale |
|------|-----------|-----------|
| Philosophy | 5 | Core beliefs need strongest validation |
| Pattern | 3 | Best practices need multiple confirmations |
| Redirect | 2 | Anti-patterns need fewer failures to document |
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
    "philosophy": 5,
    "pattern": 3,
    "redirect": 2,
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

1. **Don't manually edit QUEEN.md** - Use `queen-promote` to ensure proper formatting
2. **Validate before promoting** - Only promote learnings that have been tested
3. **Use descriptive colony names** - Helps track wisdom origins
4. **Read wisdom at build start** - Workers benefit from accumulated knowledge
5. **Review periodically** - Some wisdom may become outdated as the system evolves

## See Also

- `/ant:init` - Initializes QUEEN.md for new colonies
- `/ant:build` - Reads QUEEN.md wisdom for worker priming
- `/ant:continue` - Promotes validated learnings
- `/ant:seal` - Promotes final colony wisdom
- `/ant:entomb` - Promotes wisdom before archiving
