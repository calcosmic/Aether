# Phase 37: Command Trim & Utilities - Research

**Researched:** 2026-02-06
**Domain:** Command file reduction, shell utility consolidation
**Confidence:** HIGH

## Summary

This phase focuses on code reduction, not technology selection. The research identifies what can be removed vs what must be preserved based on current usage patterns and Phase 36 signal simplification work.

Key findings:
1. **Colonize** (529 lines) contains extensive multi-colonizer spawning logic that can be replaced with single-pass surface scan per CONTEXT.md decisions
2. **Status** (308 lines) has verbose display templates and sensitivity matrix displays that should be removed (TTL system replaced sensitivity)
3. **Signal commands** (99-102 lines) are already close to target (~40 lines each) but have verbose output templates and stale sensitivity references
4. **aether-utils.sh** (317 lines) has 8 obsolete functions from Phase 36, keeping only validate-state, error-add, and minimal activity logging

**Primary recommendation:** Delete or inline obsolete code following Phase 36 TTL migration; let LLM format output naturally instead of verbose ASCII templates.

## Standard Stack

This phase is a reduction task, not a new technology integration. No new libraries or tools.

### Tools Used
| Tool | Purpose | Phase 37 Usage |
|------|---------|----------------|
| jq | JSON processing in bash | KEEP in aether-utils.sh for validate-state |
| bash | Shell scripting | KEEP minimal utils |

### Removal Candidates
| Item | Current State | Action |
|------|---------------|--------|
| pheromone-batch | Called in 7 commands | REMOVE (TTL filtering in commands now) |
| pheromone-cleanup | Called in 2 commands | REMOVE (filter-on-read replaces cleanup) |
| memory-compress | Called in 1 command | REMOVE or INLINE |
| activity-log | Called in 3 commands | KEEP minimal version |
| learning-inject | Called in 1 command | EVALUATE (colonize only) |
| learning-promote | Referenced in 1 command | EVALUATE (continue only) |

## Architecture Patterns

### Pattern 1: Single-Pass Surface Scan (Colonize)

**What:** Replace multi-colonizer spawning with direct file tree analysis
**When to use:** New codebase colonization
**Current:** 3 colonizer spawns with synthesis step (~350 lines)
**Target:** Direct Read/Glob/Grep with ~50 line output to CODEBASE.md

```markdown
## Reduced Colonize Pattern

### Step 1: Validate state (same as current)
### Step 2: Surface scan (NEW - replaces Steps 2-4.5)
  - Glob for key files: package.json, README, entry points
  - Read up to 20 key files
  - Build tech stack, directory structure, entry points list
  - Write to .planning/CODEBASE.md
### Step 3: Update state (simplified)
### Step 4: Display minimal confirmation
```

**Evidence:** CONTEXT.md decision: "Surface scan only: file tree + key files (package.json, README, entry points), ~20 files max"

### Pattern 2: Minimal Status Display

**What:** Quick glance summary instead of verbose sectioned display
**Current:** 8 sections with dividers, sensitivity tables, strength bars
**Target:** 5-line summary answering "where are we?"

```markdown
## Reduced Status Output

Phase 3/7: Building authentication layer
Tasks: 4/6 complete
Signals: 3 active (2 focus, 1 redirect)
Workers: 1 active (builder)
Next: /ant:continue
```

**Evidence:** CONTEXT.md decision: "Quick glance purpose: answer 'where are we?' in ~5 lines"

### Pattern 3: One-Line Signal Confirmation

**What:** Minimal confirmation for signal commands instead of verbose response with sensitivity tables
**Current:** 20+ lines including sensitivity breakdown per caste
**Target:** Single confirmation line

```markdown
## Reduced Signal Output

/ant:focus "WebSocket security"
-> FOCUS signal emitted: "WebSocket security" (expires: phase_end)
```

**Evidence:** CONTEXT.md decision: "One-line confirmation output"

### Anti-Patterns to Avoid

- **Sensitivity matrix displays:** Phase 36 removed sensitivity calculations; don't display them
- **ASCII art headers:** Let LLM format naturally
- **Step-by-step checkmarks:** Unnecessary ceremony for simple commands
- **Pheromone strength bars:** TTL system doesn't use strength; show priority and expiration instead

## Don't Hand-Roll

Not applicable for this reduction phase. The focus is removing hand-rolled complexity, not adding libraries.

| Problem | What To Remove | Why |
|---------|----------------|-----|
| Signal filtering | pheromone-batch calls | Phase 36 moved this to inline TTL checks |
| Signal cleanup | pheromone-cleanup calls | Filter-on-read pattern eliminates need |
| Decay math | strength calculations | TTL replaced exponential decay |
| Sensitivity display | caste sensitivity tables | Keywords replaced sensitivity matrix |

## Common Pitfalls

### Pitfall 1: Forgetting to update both command directories

**What goes wrong:** Changes applied to `.claude/commands/ant/` but not `commands/ant/`
**Why it happens:** Repo has two command directories, one appears to be source
**How to avoid:** Update `commands/ant/` as source, copy to `.claude/commands/ant/` or use symlinks
**Warning signs:** Line counts differ between directories (already the case)

### Pitfall 2: Removing utility functions still in use

**What goes wrong:** Delete utility function that a command still calls
**Why it happens:** Grep shows usage but command might not be updated yet
**How to avoid:** Check all usages before removing; update commands first, then remove utilities
**Warning signs:** `command not found` errors from aether-utils.sh

### Pitfall 3: Breaking TTL filtering by removing wrong code

**What goes wrong:** Accidentally remove inline TTL filtering while trimming
**Why it happens:** TTL filtering was added in Phase 36; might look like verbose code to cut
**How to avoid:** Preserve `expires_at` checks; only remove strength/decay references
**Warning signs:** Expired signals appearing in output

### Pitfall 4: Output too terse for user orientation

**What goes wrong:** Trim so aggressively that users can't understand what happened
**Why it happens:** Overcorrecting from verbose templates
**How to avoid:** Keep essential context (what, where, next step); remove ceremony
**Warning signs:** Users asking "what did that do?" after commands

## Code Examples

### Example 1: Reduced Signal Command Structure (~40 lines)

```markdown
---
name: ant:focus
description: Emit FOCUS signal to guide colony attention
---

You are the **Queen**. Emit a FOCUS signal.

## Instructions

The focus area is: `$ARGUMENTS`

### Step 1: Validate
If `$ARGUMENTS` empty -> show usage, stop.

### Step 2: Read + Update State
Read `.aether/data/COLONY_STATE.json`.
If `goal: null` -> "No colony initialized." stop.

Generate timestamp and ID.

Append to `signals` array:
```json
{
  "id": "focus_<timestamp>",
  "type": "FOCUS",
  "content": "<focus area>",
  "priority": "normal",
  "created_at": "<ISO-8601>",
  "expires_at": "phase_end"
}
```

Write COLONY_STATE.json.

### Step 3: Confirm
Output: `FOCUS signal emitted: "<content>" (expires: phase_end)`
```

### Example 2: Minimal Utility Function (~15 lines)

```bash
# validate-state: Check JSON structure
validate-state() {
  case "${1:-}" in
    colony) jq 'has("goal") and has("state")' "$DATA_DIR/COLONY_STATE.json" ;;
    all) # validate all required fields exist
      jq '
        (has("goal") and has("state") and has("signals") and has("plan"))
      ' "$DATA_DIR/COLONY_STATE.json"
      ;;
    *) json_err "Usage: validate-state colony|all" ;;
  esac
}
```

### Example 3: Reduced Colonize Pattern (~150 lines)

```markdown
### Step 2: Surface Scan

Use Glob to find key files:
- `package.json`, `Cargo.toml`, `pyproject.toml`, `go.mod` (package manifest)
- `README.md` or `README.*`
- Entry points: `src/index.*`, `src/main.*`, `main.*`, `app.*`
- Config: `tsconfig.json`, `.eslintrc.*`, `jest.config.*`

Read up to 20 files. Extract:
- Tech stack (language, framework, dependencies)
- Entry points (main files)
- Key directories (src/, lib/, tests/)
- File counts per top-level directory

Write to `.planning/CODEBASE.md`:
```markdown
# Codebase Overview

**Stack:** TypeScript + React
**Entry:** src/index.tsx
**Dirs:** src/ (45 files), tests/ (12 files)
**Deps:** react, react-dom, axios (3 production)
```

Display: "Codebase scan complete. See .planning/CODEBASE.md"
```

## State of the Art

| Old Approach | Current Approach | Phase 36 Change | Impact on Phase 37 |
|--------------|------------------|-----------------|-------------------|
| Exponential decay | TTL with expires_at | Strength bars obsolete | Remove from status |
| Sensitivity matrices | Keyword-based priority | Caste tables obsolete | Remove from signal commands |
| pheromone-batch utility | Inline TTL filtering | Utility obsolete | Delete from aether-utils.sh |
| pheromone-cleanup utility | Filter-on-read | Utility obsolete | Delete from aether-utils.sh |
| Multi-colonizer spawns | Direct analysis | CONTEXT.md decision | Replace in colonize |
| Verbose ASCII templates | LLM-formatted output | CONTEXT.md decision | Remove templates |

**Deprecated/outdated:**
- `pheromone-batch`: Replaced by inline TTL checks
- `pheromone-cleanup`: Replaced by filter-on-read
- `memory-compress`: Can be inlined or removed (single caller)
- `spawn-check`: Bayesian tracking removed per REQUIREMENTS.md
- Sensitivity matrix display: Phase 36 removed

## Utility Function Disposition

Based on current usage and CONTEXT.md decisions:

| Function | Lines | Callers | Decision | Rationale |
|----------|-------|---------|----------|-----------|
| validate-state | ~50 | init.md | KEEP | Useful sanity check |
| error-add | ~15 | multiple | KEEP | Error tracking |
| pheromone-validate | ~10 | colonize | KEEP | Content length validation |
| activity-log | ~10 | build, organize | KEEP minimal | Logging helper |
| activity-log-init | ~15 | build | INLINE or DELETE | Single caller |
| activity-log-read | ~10 | status | INLINE or DELETE | Single caller |
| pheromone-batch | ~20 | 0 after updates | DELETE | TTL filtering replaces |
| pheromone-cleanup | ~15 | 0 after updates | DELETE | Filter-on-read replaces |
| memory-compress | ~15 | continue | INLINE or DELETE | Single caller |
| spawn-check | ~15 | 0 | DELETE | Bayesian tracking removed |
| error-pattern-check | ~10 | 0 | DELETE | Unused |
| error-summary | ~10 | continue | INLINE | Single caller, trivial |
| learning-promote | ~40 | continue | EVALUATE | Global learning feature |
| learning-inject | ~25 | colonize | EVALUATE | Global learning feature |

**Budget analysis:** Keep validate-state (~50) + error-add (~15) + pheromone-validate (~10) + activity-log (~10) = ~85 lines plus help/version = ~80 target achievable.

## Open Questions

1. **Global learning functions (learning-promote, learning-inject)**
   - What we know: Only used in colonize and continue
   - What's unclear: Are global learnings still a desired feature?
   - Recommendation: Move to DEFERRED unless explicitly needed; they add ~65 lines

2. **File-lock and atomic-write utilities**
   - What we know: Sourced by aether-utils.sh (~200+ lines in utils/ folder)
   - What's unclear: Are these still needed with single COLONY_STATE.json?
   - Recommendation: Keep if concurrent access is a concern; otherwise inline or remove

3. **commands/ant/ vs .claude/commands/ant/ sync**
   - What we know: Two directories with different file contents
   - What's unclear: Which is source of truth? How are they synced?
   - Recommendation: Establish canonical source and update both

## Sources

### Primary (HIGH confidence)
- `.claude/commands/ant/colonize.md` - Current 529 lines, analyzed for reduction
- `.claude/commands/ant/status.md` - Current 308 lines, analyzed for reduction
- `.claude/commands/ant/focus.md`, `redirect.md`, `feedback.md` - Current 99-102 lines each
- `runtime/aether-utils.sh` - Current 317 lines, function-by-function analysis
- `.planning/phases/37-command-trim-utilities/37-CONTEXT.md` - User decisions

### Secondary (HIGH confidence)
- `.planning/phases/36-signal-simplification/36-VERIFICATION.md` - Confirms TTL migration complete
- `.planning/REQUIREMENTS.md` - SIMP-05 and SIMP-06 specifications

## Metadata

**Confidence breakdown:**
- Colonize reduction: HIGH - Clear CONTEXT.md decisions, code analyzed
- Status reduction: HIGH - Clear CONTEXT.md decisions, code analyzed
- Signal command reduction: HIGH - Commands already near target, clear pattern
- Utility reduction: HIGH - Usage analysis complete, function-by-function disposition

**Research date:** 2026-02-06
**Valid until:** Indefinite (reduction phase, not technology-dependent)

---

*Phase: 37-command-trim-utilities*
*Research completed: 2026-02-06*
