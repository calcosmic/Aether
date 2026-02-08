# Aether v2.0 Improvement Plan

**Created:** 2026-02-08
**Based on:** 6 parallel research agents + 2 test reports (FocusNudge, Zephyr)
**Baseline:** v1.2.1 (tagged)

---

## Executive Summary

The v1.2.1 release achieved Queen-direct spawning with parallel workers, but test reports reveal 5 critical gaps:

| Gap | Current State | Target |
|-----|---------------|--------|
| Nested spawning | Workers don't spawn sub-workers | Depth 1→2→3 chains |
| Visual output | Monotone green, no differentiation | Colored per-caste, emoji-rich |
| Runtime verification | User skips, Watcher doesn't run code | Mandatory execution checks |
| Task prescription | Exact code in tasks | Goal-oriented with constraints |
| Project tracking | No milestone/flag system | Per-project flags + versioning |

---

## Phase 1: Nested Spawning (Priority: CRITICAL)

**Root cause:** Worker prompts in build.md don't include spawn instructions despite workers.md documenting the capability.

### 1.1 Add Spawn Instructions to Worker Prompts

**File:** `.claude/commands/ant/build.md` (Step 5.1 Builder prompt)

Add after `--- INSTRUCTIONS ---`:
```
--- SPAWN CAPABILITY ---
You are at depth {depth}. You MAY spawn sub-workers if you encounter genuine surprise (3x expected complexity).

Spawn limits by depth:
- Depth 1: max 4 spawns
- Depth 2: max 2 spawns
- Depth 3: NO spawns (complete inline)

Before spawning:
  bash ~/.aether/aether-utils.sh spawn-log "{your_name}" "{child_caste}" "{child_name}" "{task}"

Use Task tool with subagent_type="general-purpose".

After spawn completes:
  bash ~/.aether/aether-utils.sh spawn-complete "{child_name}" "{status}" "{summary}"

Full format: ~/.aether/workers.md section "Spawning Sub-Workers"
```

### 1.2 Add Spawn Tracking Utilities

**File:** `.aether/aether-utils.sh`

Add new subcommands:
```bash
spawn-can-spawn)
  # Check if spawning allowed at given depth
  depth="${1:-1}"
  max_for_depth=$([[ $depth -eq 1 ]] && echo 4 || ([[ $depth -eq 2 ]] && echo 2 || echo 0))
  current=$(wc -l < "$DATA_DIR/spawn-tree.txt" 2>/dev/null || echo 0)
  can=$([[ $depth -lt 3 ]] && [[ $current -lt 10 ]] && echo "true" || echo "false")
  json_ok "{\"can_spawn\": $can, \"depth\": $depth, \"max_spawns\": $max_for_depth, \"current_total\": $current}"
  ;;

spawn-get-depth)
  # Return depth from spawn tree for given ant name
  ant_name="${1:-Queen}"
  # Count parents in spawn tree
  depth=$(grep -c ".*|.*|$ant_name" "$DATA_DIR/spawn-tree.txt" 2>/dev/null || echo 1)
  json_ok "\"$depth\""
  ;;
```

### 1.3 Pass Depth Through Build

**File:** `.claude/commands/ant/build.md`

Track depth when spawning:
- Queen spawns at depth 1
- Worker prompts include `depth: {current_depth}`
- Child spawns increment: `child_depth = parent_depth + 1`

### 1.4 Update workers.md Spawn Section

Make lines 228-286 copy-paste ready with step-by-step:
1. Check `spawn-can-spawn {depth}`
2. Log with `spawn-log`
3. Use Task tool with full prompt template
4. Log completion with `spawn-complete`

---

## Phase 2: Visual Improvements (Priority: HIGH)

### 2.1 Color Scheme by Caste

**File:** `.aether/utils/colorize-log.sh`

```bash
# ANSI color codes by caste
QUEEN="\e[35m"      # Magenta
BUILDER="\e[33m"    # Yellow
WATCHER="\e[36m"    # Cyan
SCOUT="\e[32m"      # Green
COLONIZER="\e[34m"  # Blue
ARCHITECT="\e[37m"  # White
RESET="\e[0m"

colorize_by_caste() {
  case "$1" in
    *Queen*|*QUEEN*)     printf "${QUEEN}%s${RESET}\n" "$1" ;;
    *Builder*|*BUILDER*) printf "${BUILDER}%s${RESET}\n" "$1" ;;
    *Watcher*|*WATCHER*) printf "${WATCHER}%s${RESET}\n" "$1" ;;
    *Scout*|*SCOUT*)     printf "${SCOUT}%s${RESET}\n" "$1" ;;
    *Colonizer*)         printf "${COLONIZER}%s${RESET}\n" "$1" ;;
    *Architect*)         printf "${ARCHITECT}%s${RESET}\n" "$1" ;;
    *)                   echo "$1" ;;
  esac
}
```

### 2.2 Enhanced Activity Log Format

**File:** `.aether/aether-utils.sh` (activity-log command)

Update output format:
```
[10:05:03] 🔨 Hammer-42 (Builder): Constructed auth module...
[10:05:05] 👁️ Vigil-17 (Watcher): Inspecting test coverage...
[10:05:08] 🔍 Swift-7 (Scout): Discovered pattern in utils...
```

### 2.3 ASCII Art Headers

**Files:** `build.md`, `status.md`, `continue.md`

Add colony header:
```
       .-.
      (o o)  AETHER COLONY
      | O |  ═══════════════
       `-`   Phase 3 Building...
```

### 2.4 Progress Bars with Unicode

```
Progress: [████████████░░░░░░░░] 60%
Pheromone: [━━━━━━━━░░░░░░░░░░░░] 0.4 (decaying)
```

---

## Phase 3: Runtime Verification (Priority: HIGH)

### 3.1 Watcher Execution Verification (Mandatory)

**File:** `.aether/workers.md` (Watcher section)

Add new section after Workflow:
```markdown
### Execution Verification (Mandatory)

Before assigning a quality score, you MUST attempt to execute the code:

1. **Syntax check:** Run the language's syntax checker
   - Python: `python3 -m py_compile {file}`
   - Swift: `swiftc -parse {file}`
   - TypeScript: `npx tsc --noEmit`

2. **Import check:** Verify main entry point can be imported
   - Python: `python3 -c "import {module}"`
   - Node: `node -e "require('{entry}')"`

3. **Launch test:** Attempt to start the application briefly
   - Run main entry point with timeout
   - If GUI, try headless mode if possible
   - If launches successfully = pass
   - If crashes = CRITICAL severity

4. **Test suite:** If tests exist, run them
   - Record pass/fail counts
   - Note "no test suite" if none exist

**CRITICAL:** If ANY execution check fails, quality_score CANNOT exceed 6/10.

Report format:
```
Execution Verification:
  ✅ Syntax: all files pass
  ✅ Import: main module loads
  ❌ Launch: crashed — [error message] (CRITICAL)
  ⚠️ Tests: no test suite found
```
```

### 3.2 Update Watcher Prompt in build.md

**File:** `.claude/commands/ant/build.md` (Step 5.4)

Add to Watcher prompt:
```
--- CRITICAL REQUIREMENTS ---
You MUST execute the code, not just read it.
1. Run syntax checks on all modified files
2. Run import check on main entry
3. Attempt to launch the application
4. Run test suite if it exists

If any execution check fails, your quality_score CANNOT exceed 6/10.
Include Execution Verification section in your output.
```

---

## Phase 4: Goal-Oriented Tasks (Priority: MEDIUM)

### 4.1 New Task Format

**Current (over-prescriptive):**
```json
{
  "description": "Add WaveformAnimationView(isPlaying: !isMuted, color: categoryColor, barCount: 3) next to mute icon"
}
```

**Proposed (goal-oriented):**
```json
{
  "goal": "Show visual feedback indicating sound is actively playing",
  "constraints": [
    "Must react to mute state",
    "Use category color for consistency"
  ],
  "hints": [
    "WaveformAnimationView exists in Animations.swift"
  ],
  "success_criteria": [
    "Animation visible when playing",
    "Animation stops when muted"
  ]
}
```

### 4.2 Update plan.md Task Generation

**File:** `.claude/commands/ant/plan.md`

Add task generation guidance:
```
When creating tasks:
- **goal**: What to achieve (not how)
- **constraints**: Boundaries and requirements
- **hints**: Optional pointers (not solutions)
- **success_criteria**: How to verify completion

DO NOT include:
- Exact code to write
- Specific function names (unless critical API)
- Implementation details
```

### 4.3 Update Worker Instructions

**File:** `.aether/workers.md` (Builder section)

Add discovery guidance:
```
When you receive a goal-oriented task:
1. Read existing code to understand patterns
2. Research similar implementations in codebase
3. Design solution that fits project style
4. Implement following existing conventions
5. Verify against success_criteria

You are a problem solver, not a code typist.
```

---

## Phase 5: Per-Project Flagging (Priority: MEDIUM)

### 5.1 flags.json Schema

**File:** `.aether/data/flags.json` (new)

```json
{
  "version": 1,
  "flags": [
    {
      "id": "flag_1707355200_a3f2",
      "type": "blocker",
      "severity": "critical",
      "title": "Build fails on auth module",
      "description": "TypeError in login.ts line 42",
      "source": "verification",
      "phase": 3,
      "created_at": "2026-02-08T01:00:00Z",
      "acknowledged_at": null,
      "resolved_at": null,
      "resolution": null,
      "auto_resolve_on": "build_pass"
    }
  ]
}
```

### 5.2 Flag Types

| Type | Severity | Behavior |
|------|----------|----------|
| blocker | critical | Blocks phase advancement |
| issue | high | Warning, can acknowledge and continue |
| note | low | Informational only |

### 5.3 New Commands

**File:** `.claude/commands/ant/flag.md` (new)
```
/ant:flag "description" [--type blocker|issue|note]
```

**File:** `.claude/commands/ant/flags.md` (new)
```
/ant:flags [--all]  # List active flags, --all includes resolved
```

### 5.4 Integration Points

- `continue.md`: Check for unacknowledged blockers before advancing
- `build.md`: Check for active blockers before starting phase
- `status.md`: Add FLAGS section showing active issues
- Verification gates: Auto-create blocker flag on failure

### 5.5 Utility Commands

**File:** `.aether/aether-utils.sh`

```bash
flag-add)
  type="$1"; title="$2"; desc="$3"; source="$4"; phase="${5:-null}"
  id="flag_$(date +%s)_$(head -c4 /dev/urandom | xxd -p)"
  # Add to flags.json
  ;;

flag-check-blockers)
  # Return count of unresolved blockers
  ;;

flag-resolve)
  flag_id="$1"; resolution="$2"
  # Update flag with resolution
  ;;
```

---

## Phase 6: Milestone/Versioning (Priority: LOW)

### 6.1 MILESTONES.md Format

**File:** `.aether/data/MILESTONES.md` (new)

```markdown
# Project Milestones

## v1.0 MVP (Shipped: 2026-02-01)

**Delivered:** Basic ADHD reminder app with notifications

**Phases completed:** 1-5 (30 tasks)

**Key accomplishments:**
- Electron app with system tray
- Native notifications
- Focus mode timer

**Stats:** 15 files, 2,400 LOC, 5 days

---
```

### 6.2 Version Tracking in COLONY_STATE.json

Add field:
```json
{
  "milestone": {
    "version": "1.0",
    "name": "MVP",
    "started_at": "2026-01-27T00:00:00Z",
    "phases_included": [1, 2, 3, 4, 5]
  }
}
```

### 6.3 Milestone Commands

- `/ant:milestone "v1.0 MVP"` — Mark current work as milestone
- Integration with `/ant:status` to show current milestone progress

---

## Implementation Order

1. **Nested Spawning** (Phase 1) — Core promise of the system
2. **Runtime Verification** (Phase 3) — Prevents shipping broken code
3. **Visual Improvements** (Phase 2) — User experience
4. **Goal-Oriented Tasks** (Phase 4) — Enables true emergence
5. **Per-Project Flagging** (Phase 5) — Issue tracking
6. **Milestone/Versioning** (Phase 6) — Project management

---

## Risk Mitigation

| Risk | Mitigation |
|------|------------|
| Context overflow from deep spawning | Compressed handoffs (already in workers.md) |
| Runaway spawning | Depth + total caps (10 workers max) |
| Watcher execution fails on GUI apps | Headless mode instructions |
| Goal tasks too vague | Include success_criteria always |
| Flag proliferation | Auto-resolve on build_pass |

---

## Success Metrics

| Metric | Current | Target |
|--------|---------|--------|
| Spawn depth achieved | 1 | 2-3 |
| Watcher execution rate | 0% | 100% |
| Visual differentiation | 1 color | 6 colors |
| Task prescription level | High | Low |
| Flags per project | 0 | 3-10 active |

---

## Files to Modify

| File | Phases | Changes |
|------|--------|---------|
| `.claude/commands/ant/build.md` | 1, 2, 3 | Spawn instructions, execution requirements |
| `.aether/aether-utils.sh` | 1, 2, 5 | Spawn utilities, flag utilities, colorize |
| `.aether/workers.md` | 1, 3, 4 | Spawn docs, execution verification, discovery |
| `.aether/utils/colorize-log.sh` | 2 | Color by caste |
| `.claude/commands/ant/plan.md` | 4 | Goal-oriented task generation |
| `.claude/commands/ant/continue.md` | 5 | Flag checks |
| `.claude/commands/ant/status.md` | 2, 5 | Visual improvements, flags section |

## New Files

| File | Phase | Purpose |
|------|-------|---------|
| `.aether/data/flags.json` | 5 | Per-project issue tracking |
| `.aether/data/MILESTONES.md` | 6 | Milestone history |
| `.claude/commands/ant/flag.md` | 5 | Create flags |
| `.claude/commands/ant/flags.md` | 5 | List flags |

---

## Verification Plan

After implementation:

1. **Spawn test:** Run `/ant:build` on phase with 4+ tasks, verify spawn depth > 1
2. **Visual test:** Run `/ant:watch`, confirm different colors per caste
3. **Execution test:** Create intentional syntax error, verify Watcher catches it
4. **Goal test:** Generate plan, verify tasks are goal-oriented
5. **Flag test:** Trigger verification failure, verify blocker flag created

---

*Plan synthesized from 6 parallel research agents*
*Ready for implementation*
