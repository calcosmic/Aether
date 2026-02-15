# Namespace Distinctiveness -- Command System

This document explains Aether's command namespace (`ant`) and how it maintains isolation from other agent systems.

---

## Overview

Aether uses a directory-based command namespace system that ensures bulletproof isolation and prevents command collisions with other agent frameworks.

---

## The `ant` Namespace

Aether commands use the `ant:` prefix with biological/colony metaphors:

| Command | Purpose |
|---------|---------|
| `/ant:build` | Build a phase with workers |
| `/ant:plan` | Generate project plan |
| `/ant:focus` | Deposit FOCUS pheromone |
| `/ant:colonize` | Analyze codebase |
| `/ant:swarm` | Deploy parallel scouts |
| `/ant:init` | Initialize colony |

---

## How Commands Are Invoked

### Claude Code (Slash Commands)

```
/ant:build 1                    # Aether build command
/ant:focus "auth"               # Aether focus command
/ant:plan                       # Aether plan command
```

**Format:** `/<namespace>:<command> <arguments>`

### OpenCode

```
ant:build 1                     # Build using Aether in OpenCode
ant:init "myapp"                # Initialize colony
```

**Format:** `<namespace>:<command> <arguments>`

---

## Directory Structure

```
.claude/commands/
└── ant/                          # Aether namespace
    ├── build.md
    ├── plan.md
    ├── focus.md
    ├── colonize.md
    ├── swarm.md
    └── ...
```

Each command is a separate `.md` file in the `ant/` subdirectory.

---

## Why 'ant' Is Distinct

### 1. Directory-Based Isolation

The `ant` namespace is stored in a dedicated directory (`~/.claude/commands/ant/`):

- **Physical separation**: Files exist in a distinct directory
- **Clear ownership**: The `ant/` directory is explicitly owned by Aether
- **No filename conflicts**: Each command is a separate file

### 2. Unique Naming Convention

Aether commands use biological/colony metaphors:

- `build` → Colony construction
- `plan` → Colony planning
- `colonize` → Codebase exploration
- `swarm` → Parallel scouts
- `init` → Colony founding

### 3. File Extension Pattern

Aether commands are `.md` files within the `ant/` directory:

```
.claude/commands/ant/build.md      # Invoked as /ant:build
.claude/commands/ant/plan.md      # Invoked as /ant:plan
.claude/commands/ant/focus.md     # Invoked as /ant:focus
```

### 4. Sync Mechanism Provides Safety

The Aether CLI (`bin/cli.js`) implements hash-based idempotent sync:

- **Source**: `.claude/commands/ant/` in the package
- **Destination**: `~/.aether/commands/claude/` (hub) and `~/.claude/commands/ant/` (global)
- **Behavior**: Files are only copied when content changes (hash comparison)
- **Cleanup**: Stale files are automatically removed

---

## Collision Prevention Checklist

When adding new commands to Aether, verify:

- [ ] Command file is in `.claude/commands/ant/` directory
- [ ] Filename uses lowercase alphanumeric with hyphens (e.g., `build.md`)
- [ ] Frontmatter uses `ant:` prefix (e.g., `name: ant:build`)
- [ ] Command uses biological/colony metaphor

---

## Command Count

| Namespace | Command Count | Files |
|-----------|---------------|-------|
| `ant` | 29+ commands | `ant/*.md` |

---

## Best Practices

1. **Never modify commands outside the `ant/` directory** - This maintains clear ownership
2. **Use the sync mechanism** - Let `aether-cli` handle global distribution
3. **Follow naming conventions** - Use lowercase, hyphens, biological metaphors
4. **Keep commands atomic** - Each command should do one thing well

---

## Summary

The `ant` namespace is bulletproof because:

1. **Directory isolation** - Commands live in `~/.claude/commands/ant/`, separate from other systems
2. **Unique prefix** - `/ant:` is exclusively used by Aether
3. **Hash-based sync** - Idempotent updates prevent drift and contamination
4. **Clear ownership** - The `ant` prefix and directory are unique to Aether
