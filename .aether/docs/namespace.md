# Namespace Distinctiveness -- Command System

This document explains how Aether's command namespace (`ant`) is distinct from other agent namespaces in the global Claude Code configuration, ensuring bulletproof isolation and preventing command collisions.

---

## Overview

Aether uses a directory-based command namespace system that is distinct from other agent systems. The namespace is designed to be bulletproof against collisions with other agent frameworks.

---

## Existing Agent Namespaces

The global Claude Code commands directory (`~/.claude/commands/`) contains multiple agent namespaces:

| Namespace | Type | Location | Example Commands |
|-----------|------|----------|-------------------|
| `ant` | Directory | `ant/` | `/ant:build`, `/ant:plan`, `/ant:focus` |
| `cds` | Directory | `cds/` | `/cds:new-project`, `/cds:execute-phase` |
| `mds` | Directory | `mds/` | `/mds:build`, `/mds:plan`, `/mds:test` |
| `st:` | Prefix | Root files | `/st:caption`, `/st:research` |

---

## How Commands Are Invoked

### Claude Code (Slash Commands)

Claude Code uses a slash-based command syntax with namespace prefixing:

```
/ant:build 1                    # Aether build command
/ant:focus "auth"               # Aether focus command
/ant:plan                       # Aether plan command

/cds:new-project "myapp"       # Claude Development System
/mds:build                      # MDS system
/st:caption "image.png"        # ST system (prefix in filename)
```

**Format:** `/<namespace>:<command> <arguments>`

### OpenCode

OpenCode uses a different syntax with namespace prefixes:

```
mds:ant:build 1                 # Build using Aether in OpenCode
mds:cds:new-project myapp       # CDS in OpenCode
```

**Format:** `<tool>:<namespace>:<command> <arguments>`

---

## Why 'ant' Won't Collide

### 1. Directory-Based Isolation

The `ant` namespace is stored in a dedicated directory (`~/.claude/commands/ant/`). This provides:

- **Physical separation**: Files exist in a distinct directory, not mixed with other commands
- **Clear ownership**: The `ant/` directory is explicitly owned by Aether
- **No filename conflicts**: Each command is a separate file in the `ant/` subdirectory

### 2. Unique Naming Convention

Aether commands use biological/colony metaphors that don't overlap with other systems:

| Aether Commands | Other Systems |
|-----------------|---------------|
| `/ant:build` | `/cds:execute-phase` |
| `/ant:plan` | `/mds:plan` |
| `/ant:focus` | `/st:research` |
| `/ant:colonize` | `/cds:new-milestone` |

**Key distinction:** The `ant:` prefix is unique to Aether. No other agent system uses this prefix.

### 3. File Extension Pattern

Aether commands are `.md` files within the `ant/` directory:

```
.claude/commands/ant/build.md      # Invoked as /ant:build
.claude/commands/ant/plan.md      # Invoked as /ant:plan
.claude/commands/ant/focus.md     # Invoked as /ant:focus
```

This is distinct from:
- `st:*` commands which use prefixes in filenames (e.g., `st:caption.md`)
- Root-level commands which have no namespace prefix (e.g., `create-prompt.md`)

### 4. Sync Mechanism Provides Safety

The Aether CLI (`bin/cli.js`) implements hash-based idempotent sync:

- **Source**: `.claude/commands/ant/` in the package
- **Destination**: `~/.aether/commands/claude/` (hub) and `~/.claude/commands/ant/` (global)
- **Behavior**: Files are only copied when content changes (hash comparison)
- **Cleanup**: Stale files are automatically removed

This ensures:
- Local modifications are preserved
- Only Aether-owned files exist in the `ant/` directory
- No cross-contamination from other namespaces

---

## Collision Prevention Checklist

When adding new commands to Aether, verify:

- [ ] Command file is in `.claude/commands/ant/` directory
- [ ] Filename uses lowercase alphanumeric with hyphens (e.g., `build.md`)
- [ ] Frontmatter uses `ant:` prefix (e.g., `name: ant:build`)
- [ ] No naming overlap with existing commands in `cds/`, `mds/`, or root

---

## Command Count

| Namespace | Command Count | Files |
|-----------|---------------|-------|
| `ant` | 29 commands | `ant/*.md` |
| `cds` | 20 commands | `cds/*.md` |
| `mds` | 19 commands | `mds/*.md` |
| `st:` | 13 commands | `st:*.md` |

---

## Best Practices

1. **Never modify commands outside the `ant/` directory** - This maintains clear ownership
2. **Use the sync mechanism** - Let `aether-cli` handle global distribution
3. **Follow naming conventions** - Use lowercase, hyphens, biological metaphors
4. **Keep commands atomic** - Each command should do one thing well

---

## Summary

The `ant` namespace is bulletproof because:

1. **Directory isolation** - Commands live in `~/.claude/commands/ant/`, separate from other namespaces
2. **Unique prefix** - `/ant:` is exclusively used by Aether
3. **Hash-based sync** - Idempotent updates prevent drift and contamination
4. **Clear ownership** - No other system uses the `ant` prefix or directory
