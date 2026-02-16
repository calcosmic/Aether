# Aether Command System Analysis

## Executive Summary

The Aether command system is a dual-platform architecture supporting both **Claude Code** and **OpenCode** AI assistants. The system implements a sophisticated multi-agent colony metaphor with 36 commands duplicated across both platforms.

---

## Command Counts

| Platform | Count | Location |
|----------|-------|----------|
| Claude Code | 36 | `.claude/commands/ant/` |
| OpenCode | 35 | `.opencode/commands/ant/` |
| **Total** | **71** | (35 shared + 1 Claude-only) |

### Claude-Only Command
- `resume.md` - Exists only in Claude Code (OpenCode has `resume-colony.md` instead)

### Missing from OpenCode
- `resume.md` (OpenCode uses `resume-colony.md` naming)

---

## Command Categories

### 1. Lifecycle Commands (Core Workflow)
| Command | Purpose |
|---------|---------|
| `init` | Initialize colony with goal |
| `plan` | Generate project phases |
| `build` | Execute phase with parallel workers |
| `continue` | Verify work and advance phase |
| `seal` | Archive completed colony (Crowned Anthill) |
| `entomb` | Archive colony to chambers |
| `lay-eggs` | Start new colony from existing |

### 2. Pheromone Commands (User Guidance)
| Command | Purpose | Priority |
|---------|---------|----------|
| `focus` | Guide colony attention | Normal |
| `redirect` | Hard constraint (avoid pattern) | High |
| `feedback` | Gentle adjustment | Low |

### 3. Status & Information Commands
| Command | Purpose |
|---------|---------|
| `status` | Colony dashboard |
| `phase` | View phase details |
| `flags` | List active flags/blockers |
| `flag` | Create a flag |
| `history` | Browse event history |
| `help` | Command reference |

### 4. Session Management Commands
| Command | Purpose |
|---------|---------|
| `watch` | Live tmux visibility |
| `pause-colony` | Save state and handoff |
| `resume-colony` | Restore from pause |
| `resume` | Claude-specific resume |
| `update` | Update system from hub |

### 5. Advanced/Utility Commands
| Command | Purpose |
|---------|---------|
| `swarm` | Parallel bug investigation |
| `chaos` | Resilience testing |
| `archaeology` | Git history analysis |
| `oracle` | Deep research (RALF loop) |
| `colonize` | Territory survey |
| `organize` | Codebase hygiene report |
| `council` | Intent clarification |
| `dream` | Philosophical observation |
| `interpret` | Dream validation |
| `tunnels` | Browse archived colonies |
| `verify-castes` | System status check |
| `migrate-state` | State migration utility |
| `maturity` | Colony maturity assessment |

---

## Implementation Patterns

### 1. Frontmatter Header
All commands use YAML frontmatter:
```yaml
---
name: ant:<command>
description: "<emoji> <description>"
---
```

### 2. Visual Mode Pattern
Most commands support `--no-visual` flag:
```markdown
Parse `$ARGUMENTS`:
- If contains `--no-visual`: set `visual_mode = false`
- Otherwise: set `visual_mode = true`
```

### 3. Session Freshness Detection
Stateful commands include timestamp verification:
```bash
COMMAND_START=$(date +%s)
stale_check=$(bash .aether/aether-utils.sh session-verify-fresh --command <name> "" "$COMMAND_START")
```

### 4. State Validation Pattern
Commands validate COLONY_STATE.json before proceeding:
```markdown
Read `.aether/data/COLONY_STATE.json`.
If `goal: null` -> "No colony initialized. Run /ant:init first."
```

### 5. Worker Spawn Pattern
Build commands spawn parallel workers:
```markdown
**CRITICAL: Spawn ALL Wave 1 workers in a SINGLE message using multiple Task tool calls.**
```

### 6. JSON Output Pattern
Workers return structured JSON:
```json
{"ant_name": "...", "status": "completed|failed", "summary": "...",
 "files_created": [], "files_modified": [], "blockers": []}
```

---

## Platform Differences

### 1. Agent Type References

**Claude Code** uses specialized agent types:
```markdown
Task tool with `subagent_type="aether-builder"`
Task tool with `subagent_type="aether-watcher"`
Task tool with `subagent_type="aether-chaos"`
```

**OpenCode** uses general-purpose with role injection:
```markdown
Task tool with `subagent_type="general-purpose"`
# NOTE: Claude Code uses aether-chaos; OpenCode uses general-purpose with role injection
```

### 2. Argument Normalization (OpenCode)

OpenCode includes argument normalization:
```markdown
### Step -1: Normalize Arguments
Run: `normalized_args=$(bash .aether/aether-utils.sh normalize-args "$@")`
```

### 3. Help Command Differences

OpenCode help includes additional section:
```markdown
OPENCODE USERS

  Argument syntax: OpenCode handles multi-word arguments differently than Claude.
  Wrap text arguments in quotes for reliable parsing:
```

### 4. Caste Emoji Display

**Claude Code** includes ant emoji in caste display:
```markdown
🔨🐜 Builder  (cyan if color enabled)
👁️🐜 Watcher  (green if color enabled)
🎲🐜 Chaos    (red if color enabled)
```

**OpenCode** omits ant emoji:
```markdown
🔨 Builder  (cyan if color enabled)
👁️ Watcher  (green if color enabled)
🎲 Chaos    (red if color enabled)
```

### 5. Missing Session Features in OpenCode

OpenCode `init.md` is missing:
- Step 1.6: Initialize QUEEN.md Wisdom Document
- Step 5: Initialize Context Document
- Step 8: Initialize Session

---

## File Sizes (Line Counts)

### Largest Commands
| Command | Claude | OpenCode | Notes |
|---------|--------|----------|-------|
| `build` | 1051 | 989 | Most complex |
| `continue` | 1037 | ~1037 | Gate-heavy |
| `plan` | 534 | ~534 | Iterative loop |
| `oracle` | 380 | ~380 | Research wizard |
| `swarm` | 380 | ~380 | Parallel scouts |
| `chaos` | 341 | ~341 | 5 scenarios |
| `entomb` | 407 | ~407 | Archive flow |
| `seal` | 337 | ~337 | Crown milestone |
| `init` | 316 | 272 | Missing features |

### Smallest Commands
| Command | Lines | Purpose |
|---------|-------|---------|
| `focus` | 51 | Simple constraint add |
| `redirect` | 51 | Simple constraint add |
| `feedback` | 51 | Simple constraint add |
| `help` | 113 | Static reference |
| `verify-castes` | 86 | Status display |

---

## Duplication Analysis

### Near-Identical Files (>95% match)
- `help.md` - Only OpenCode section differs
- `status.md` - Identical
- `phase.md` - Identical
- `flags.md` / `flag.md` - Identical
- `watch.md` - Identical
- `focus.md` / `redirect.md` / `feedback.md` - Identical
- `swarm.md` - Identical
- `chaos.md` - Identical
- `oracle.md` - Identical
- `archaeology.md` - Identical
- `colonize.md` - Identical
- `organize.md` - Identical
- `council.md` - Identical
- `dream.md` / `interpret.md` - Identical
- `tunnels.md` - Identical
- `history.md` - Identical
- `maturity.md` - Identical
- `migrate-state.md` - Identical
- `update.md` - Identical
- `verify-castes.md` - Identical
- `seal.md` - Identical
- `entomb.md` - Identical
- `pause-colony.md` / `resume-colony.md` - Identical

### Moderate Differences (75-95% match)
- `build.md` - Agent type references, caste emojis
- `plan.md` - Likely identical (not fully diffed)
- `continue.md` - Likely identical

### Significant Differences
- `init.md` - OpenCode missing session/context init steps

---

## Issues and Inconsistencies

### 1. Command Naming Inconsistency
- Claude: `resume.md`
- OpenCode: `resume-colony.md`

### 2. Missing Session Initialization (OpenCode)
OpenCode `init.md` lacks:
- QUEEN.md initialization
- CONTEXT.md creation
- Session tracking setup

### 3. Agent Type Fallbacks
Claude commands include fallback comments:
```markdown
# FALLBACK: If "Agent type not found", use general-purpose and inject role
```
OpenCode uses general-purpose directly.

### 4. Caste Emoji Inconsistency
Claude uses combined emoji (🔨🐜), OpenCode uses single (🔨).
This affects visual consistency across platforms.

### 5. Commented Code Artifacts
Some files contain commented-out sections or TODOs:
- `build.md` has duplicate "Analyze the phase tasks" lines
- Some commands have commented alternative implementations

---

## Recommendations

### High Priority
1. **Unify `init.md`** - Add missing session/context steps to OpenCode
2. **Standardize naming** - Align `resume.md` vs `resume-colony.md`
3. **Fix caste emojis** - Use consistent emoji format across platforms

### Medium Priority
4. **Generate OpenCode from Claude** - Create a transformation script
5. **Add diff checking to CI** - Prevent drift between platforms
6. **Document platform differences** - Add comments explaining why differences exist

### Low Priority
7. **Consolidate common patterns** - Extract shared templates
8. **Add version metadata** - Track command versions in frontmatter

---

## Architecture Notes

### Command Distribution Flow
```
Aether Repo
├── .claude/commands/ant/ ──────┐
├── .opencode/commands/ant/ ────┤──→ npm install -g .
│                               │       ↓
│                               │   ~/.aether/commands/
│                               │   ├── claude/
│                               │   └── opencode/
│                               │       ↓
│                               │   Target repos via
│                               │   `aether update`
```

### Command Categories by Complexity
1. **Simple** (50-100 lines): Pheromone commands, utilities
2. **Medium** (200-400 lines): Status, lifecycle, advanced
3. **Complex** (500+ lines): Build, continue, plan

### Worker Castes Referenced
| Caste | Emoji | Used In |
|-------|-------|---------|
| Builder | 🔨🐜 | build |
| Watcher | 👁️🐜 | build, continue |
| Chaos | 🎲🐜 | build, chaos |
| Scout | 🔍🐜 | plan, swarm |
| Archaeologist | 🏺🐜 | build, archaeology |
| Surveyor | 📊🐜 | colonize |
| Oracle | 🔮🐜 | oracle |
| Route Setter | 🗺️🐜 | plan |

---

*Analysis generated: 2026-02-16*
*Files analyzed: 36 Claude + 35 OpenCode commands*
