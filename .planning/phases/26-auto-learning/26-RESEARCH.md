# Phase 26: Auto-Learning - Research

**Researched:** 2026-02-04
**Domain:** Automated learning extraction, pheromone emission, duplicate detection in markdown-prompt colony system
**Confidence:** HIGH

## Summary

Phase 26 moves the learning extraction logic currently in `continue.md` (Steps 4 and 4.5) into `build.md` (Step 7), so learnings are captured automatically at the end of every build without requiring a manual `/ant:continue` call. The system also needs a duplicate detection mechanism so that if the user DOES run `/ant:continue` after a build, the learning extraction is not repeated.

The existing learning extraction logic in continue.md is well-defined: it reads errors.json, events.json, and task outcomes from PROJECT_PLAN.json, synthesizes actionable learnings, appends them to memory.json's `phase_learnings` array, runs `memory-compress`, updates spawn outcomes, and then emits FEEDBACK (always) and REDIRECT (conditionally) pheromones validated via `pheromone-validate`. This exact logic needs to be replicated in build.md Step 7, with a flag mechanism added so continue.md can detect and skip duplicate extraction.

The scope is narrow and well-bounded: three files to modify (build.md, continue.md, possibly events.json flag), no new utilities needed, no new data structures. Everything uses existing infrastructure.

**Primary recommendation:** Add learning extraction and FEEDBACK pheromone emission to build.md after the current Step 6 (Record Outcome), using the same logic from continue.md Steps 4 and 4.5. Use an `auto_learnings_extracted` event in events.json as the flag for duplicate detection. Update continue.md to check for this event before running its own extraction.

## Standard Stack

### Core

| Component | Current State | Purpose | Modification Needed |
|-----------|---------------|---------|---------------------|
| `build.md` | 574 lines, Step 7 is display-only | Queen-level build orchestration | Add learning extraction + FEEDBACK pheromone before display |
| `continue.md` | 304 lines, Steps 4/4.5 do learning extraction | Phase advancement + learning extraction | Add duplicate detection check at top of Step 4 |
| `memory.json` | `{phase_learnings:[], decisions:[], patterns:[]}` | Persistent learning storage | No structural changes |
| `pheromones.json` | `{signals:[]}` | Pheromone signal storage | No structural changes |
| `events.json` | `{events:[]}` | Event log | Used as flag storage (new event type) |
| `aether-utils.sh` | 265 lines, 16 subcommands | Deterministic shell operations | No changes needed |

### Supporting (Already Exists, No Changes)

| Component | Purpose | Used By |
|-----------|---------|---------|
| `memory-compress` subcommand | Enforces 20-learning cap, 30-decision cap, token threshold | Called after writing to memory.json |
| `pheromone-validate` subcommand | Validates pheromone content (non-empty, min 20 chars) | Called before writing pheromones |
| `error-summary` subcommand | Returns error counts by category/severity | Used for learning context |
| `error-pattern-check` subcommand | Returns recurring error categories (3+ occurrences) | Used for REDIRECT pheromone decision |

### No New Dependencies

This phase modifies existing prompt files only. No new utilities, no new data files, no new subcommands.

## Architecture Patterns

### Current Learning Extraction Flow (continue.md Steps 4 + 4.5)

This is the exact logic that needs to be replicated in build.md:

```
continue.md Step 4: Extract Phase Learnings
├── Read: PROJECT_PLAN.json (task outcomes for current phase)
├── Read: errors.json (filter by phase field matching current phase)
├── Read: events.json (recent events related to this phase)
├── Read: errors.json flagged_patterns array
├── SYNTHESIZE: Actionable learnings from task outcomes + errors + events
├── Write: Append learning entry to memory.json phase_learnings array
│   Format: {id, phase, phase_name, learnings:[], errors_encountered, timestamp}
├── Run: aether-utils.sh memory-compress
├── Update: COLONY_STATE.json spawn_outcomes (alpha/beta based on caste performance)
└── Write: Updated COLONY_STATE.json

continue.md Step 4.5: Auto-Emit Pheromones
├── ALWAYS emit FEEDBACK pheromone:
│   {id, type:"FEEDBACK", content:<summary>, strength:0.5, half_life_seconds:21600, source:"auto:continue"}
├── CONDITIONALLY emit REDIRECT pheromone (if flagged_patterns exist for this phase):
│   {id, type:"REDIRECT", content:<avoid pattern>, strength:0.9, half_life_seconds:86400, source:"auto:continue"}
├── For each pheromone, validate via: aether-utils.sh pheromone-validate "<content>"
│   If pass:false -> skip pheromone, log pheromone_rejected event
│   If pass:true -> append to pheromones.json signals array
├── Log: pheromone_auto_emitted event for each emitted pheromone
└── Write: Updated pheromones.json and events.json
```

### Current build.md Step 7 (Display Results Only)

Currently, Step 7 is purely a display step. It shows:
- Step progress checklist
- Git checkpoint hash
- Colony activity (per-worker results)
- Task results
- Caste pheromone sensitivity table
- Watcher report (quality score, issues)
- Warning to run `/ant:continue`
- Next steps menu

Step 7 does NOT read memory.json, does NOT write any learnings, and does NOT emit pheromones. The warning at the bottom says:

```
IMPORTANT: Run /ant:continue to extract learnings before building the next phase.
Skipping /ant:continue means phase learnings are lost and the feedback loop breaks.
```

This warning becomes obsolete once auto-learning is added.

### Target Architecture (What Phase 26 Creates)

```
build.md (after Phase 26)
  Steps 1-6: Unchanged (validate, read, pheromones, update, checkpoint, plan, execute, record)
  Step 7: AUTO-LEARNING (NEW — replaces display-only)
    7a: Extract Phase Learnings
        ├── Read: memory.json
        ├── Analyze: errors.json (already in memory from Step 6), events.json, worker results from Step 5c
        ├── Synthesize: Learnings attributed to worker/caste
        ├── Write: Append to memory.json phase_learnings
        ├── Run: memory-compress
        └── Update: spawn_outcomes in COLONY_STATE.json (NOTE: Step 6 already does this — avoid duplication)
    7b: Emit FEEDBACK Pheromone
        ├── Compose: Balanced summary of what worked + what failed
        ├── Validate: pheromone-validate
        ├── CONDITIONALLY: REDIRECT pheromone if flagged_patterns exist
        ├── Write: Append to pheromones.json
        └── Log: auto_learnings_extracted event + pheromone_auto_emitted events
    7c: Clean Expired Pheromones
        └── Run: pheromone-cleanup
    7d: Display Results (current Step 7 content, updated)
        ├── Remove "/ant:continue" warning
        ├── Add "Learnings Extracted:" section showing what was learned
        ├── Add "Auto-Emitted Pheromones:" section showing FEEDBACK/REDIRECT
        └── Update Next Steps to not emphasize /ant:continue

continue.md (after Phase 26)
  Step 4: Extract Phase Learnings
    ├── NEW: Check for auto_learnings_extracted event for current phase
    │   If found: Print "Learnings already captured during build — skipping" and skip to Step 5
    │   If not found: Proceed with normal extraction
    └── Rest unchanged
  Step 4.5: Auto-Emit Pheromones
    ├── NEW: Same flag check — skip if already emitted
    └── Rest unchanged
```

### Key Architectural Decisions

**1. Learning extraction placement in build.md:**

Insert AFTER Step 6 (Record Outcome) and BEFORE the display. Step 6 already has all the data needed:
- Worker results from Step 5c
- Watcher report from Step 5.5
- Errors logged during Step 6
- Phase completion status

The Queen has everything in memory at this point. No additional file reads needed beyond memory.json and pheromones.json.

**2. Spawn outcomes overlap:**

Step 6 already updates `spawn_outcomes` in COLONY_STATE.json. Continue.md Step 4 also does this. The auto-learning in build.md Step 7 should NOT duplicate the spawn outcomes update — Step 6 already handles it. This is a key difference from continue.md's logic.

**3. Source attribution:**

Pheromones and events from build.md should use `source: "auto:build"` (not `"auto:continue"`) to distinguish which command generated them. This is important for duplicate detection and for tracing where learnings came from.

### Anti-Patterns to Avoid

- **Do NOT move Step 6 logic into Step 7:** Step 6 (Record Outcome) handles task status, errors, pattern flagging, and spawn outcomes. Step 7 handles learnings and pheromones. Keep them separate.
- **Do NOT duplicate spawn_outcomes update:** Step 6 already does this. The learning extraction in Step 7 should skip this part.
- **Do NOT change the memory.json schema:** The `phase_learnings` entry format is established. Use the exact same format.
- **Do NOT use a separate flag file:** Use events.json (already the event log) with a specific event type. No new files needed.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Memory compression | Custom retention logic in build.md prompt | `aether-utils.sh memory-compress` | Already enforces 20-learning cap, token threshold |
| Pheromone validation | Inline content checks | `aether-utils.sh pheromone-validate` | Centralized validation, consistent behavior |
| Pheromone decay cleanup | Manual signal filtering | `aether-utils.sh pheromone-cleanup` | Removes signals below 0.05 strength automatically |
| Error summary for learning context | Manual error counting | `aether-utils.sh error-summary` | Returns by_category/by_severity JSON |
| Learning ID generation | Complex UUID logic | Same pattern as continue.md: `learn_<unix_timestamp>_<4_random_hex>` | Consistent with existing convention |

**Key insight:** ALL the utility infrastructure already exists. This phase is purely about prompt modifications — adding learning logic to build.md and duplicate detection to continue.md.

## Common Pitfalls

### Pitfall 1: build.md Exceeding Effective Prompt Length

**What goes wrong:** build.md is already 574 lines after Phase 25 restructuring. Adding learning extraction + pheromone emission + updated display could push it past 700+ lines, where Claude starts dropping instructions.
**Why it happens:** Learning extraction logic in continue.md is ~110 lines (Steps 4 + 4.5). Naively copying this adds significant prompt length.
**How to avoid:** Keep the learning extraction instructions CONCISE in build.md. The Queen already has all data in memory from Steps 5c and 6. The learning synthesis is reasoning (Claude does this naturally) — it doesn't need verbose instructions. Target: add ~60-80 net lines, not 110.
**Warning signs:** build.md exceeds 650 lines. Steps at the end of the file start being ignored.

### Pitfall 2: Flag Race Condition

**What goes wrong:** build.md writes the "learnings extracted" flag, but continue.md is run before the build fully completes (e.g., user interrupts and immediately runs continue).
**Why it happens:** The flag is written during Step 7, which is the LAST step. If the user interrupts mid-build and then runs continue, the flag won't exist yet.
**How to avoid:** This is actually not a real problem. If the build was interrupted, learnings were NOT extracted, so continue.md SHOULD extract them. The flag is only written after successful extraction. The behavior is correct by default.
**Warning signs:** None — this is a non-issue if the flag is written after extraction.

### Pitfall 3: Duplicate Spawn Outcomes Update

**What goes wrong:** Both Step 6 and the new learning extraction in Step 7 update spawn_outcomes, causing double-counting.
**Why it happens:** continue.md Step 4 includes spawn_outcomes update because it's the first point after build where caste performance is analyzed. But in build.md, Step 6 already does this.
**How to avoid:** Explicitly omit spawn_outcomes update from Step 7 learning extraction. Add a comment: "spawn_outcomes already updated in Step 6."
**Warning signs:** Alpha/beta values in COLONY_STATE.json incrementing by 2 instead of 1 per phase.

### Pitfall 4: Learning Quality Regression

**What goes wrong:** Learnings extracted automatically in build.md are generic ("Phase completed successfully") instead of specific ("TypeScript strict mode caught 12 type errors early").
**Why it happens:** In continue.md, the user has just run the command and the agent has full context from reading all state files. In build.md Step 7, the Queen has been running for many steps and may be running low on context/attention.
**How to avoid:** The Queen has ALL the data at Step 7 — worker results, watcher report, errors, events. The prompt should emphasize: "Draw from actual worker outcomes and error data. Each learning must reference a specific event, error, or outcome." Include examples of good vs bad learnings.
**Warning signs:** Learnings are vague or boilerplate. Compare against learnings that continue.md would produce.

### Pitfall 5: continue.md Skipping When It Shouldn't

**What goes wrong:** continue.md incorrectly detects learnings as already extracted and skips, but they weren't actually extracted (e.g., build.md was interrupted before Step 7 completed).
**Why it happens:** Flag check logic is too loose — checking for any event, not matching the specific phase.
**How to avoid:** The flag check must match BOTH:
1. Event type is `auto_learnings_extracted`
2. Event content references the CURRENT phase number
This ensures the flag only matches when learnings were extracted for THIS specific phase, not a previous one.
**Warning signs:** continue.md says "Learnings already captured" for a phase that was never fully built.

### Pitfall 6: Forgetting to Clear the Flag

**What goes wrong:** The "learnings extracted" flag from phase N persists into phase N+1, causing continue.md to skip extraction for the new phase.
**Why it happens:** Events.json accumulates events. The flag event from phase 5 would still be there during phase 6.
**How to avoid:** The flag check must match the current phase number in the event content. An event saying "Extracted 3 learnings from Phase 5" should NOT match when continue.md is running for Phase 6. Phase-specific matching eliminates the need for flag clearing.
**Warning signs:** Continue.md skips learning extraction for phase N+1 even though build for phase N+1 didn't include auto-learning.

## Code Examples

### Learning Entry Format (from continue.md, to be replicated in build.md)

```json
{
  "id": "learn_<unix_timestamp>_<4_random_hex>",
  "phase": 5,
  "phase_name": "Authentication Module",
  "learnings": [
    "builder-ant: Using bcrypt with 12 salt rounds caused 800ms delay per hash -- switched to 10 rounds for 200ms",
    "watcher-ant: Integration tests caught a missing error handler that unit tests missed",
    "Route-setter plans with explicit dependency chains reduced builder confusion"
  ],
  "errors_encountered": 3,
  "timestamp": "2026-02-04T15:30:00Z"
}
```

Note: Per CONTEXT.md decisions, learnings MUST be attributed to worker/caste and MUST capture both errors AND successes.

### FEEDBACK Pheromone Format (from continue.md)

```json
{
  "id": "auto_<unix_timestamp>_<4_random_hex>",
  "type": "FEEDBACK",
  "content": "Phase 5 Authentication: builder-ant successfully implemented bcrypt hashing but needed retry for error handler. Watcher caught integration gap. 3 errors total (2 medium, 1 low). Scout research on JWT patterns saved significant implementation time.",
  "strength": 0.5,
  "half_life_seconds": 21600,
  "created_at": "<ISO-8601 UTC>",
  "source": "auto:build",
  "auto": true
}
```

Note: `source` is `"auto:build"` not `"auto:continue"` to distinguish origin.

### Flag Event Format (NEW)

```json
{
  "id": "evt_<unix_timestamp>_<4_random_hex>",
  "type": "auto_learnings_extracted",
  "source": "build",
  "content": "Auto-extracted <N> learnings from Phase <id>: <name>",
  "timestamp": "<ISO-8601 UTC>"
}
```

### Duplicate Detection Check (for continue.md Step 4)

```
Before extracting learnings, check events.json for an event where:
  - type == "auto_learnings_extracted"
  - content contains "Phase <current_phase_number>:"

If found:
  Output: "Learnings already captured during build (auto-extracted at <timestamp>) -- skipping extraction."
  Skip Step 4 learning extraction and Step 4.5 pheromone emission.
  Proceed to Step 5.
```

### memory-compress Behavior (existing, for reference)

```bash
# From aether-utils.sh memory-compress subcommand:
# 1. Trim phase_learnings to last 20 entries (if over 20)
# 2. Trim decisions to last 30 entries (if over 30)
# 3. Estimate token count: count all string words * 1.3
# 4. If tokens > threshold (default 10000):
#    - Further trim phase_learnings to last 10
#    - Further trim decisions to last 15
```

This means the system keeps the MOST RECENT learnings. Older learnings are evicted first. Per CONTEXT.md, the user wants the system to "continuously get smarter" with visible eviction. The current memory-compress does silent trimming. A future enhancement could add visible notification, but that's within Claude's discretion and not blocking for Phase 26.

### pheromone-validate Behavior (existing, for reference)

```bash
# From aether-utils.sh pheromone-validate subcommand:
# 1. If content is empty: pass=false, reason="empty"
# 2. If content length < 20: pass=false, reason="too_short"
# 3. Otherwise: pass=true
```

Validation is minimal — just checks for non-empty and minimum length. No semantic validation.

## Data Flow Diagram

```
build.md execution:
  Steps 1-4.5: Setup
  Step 5a-c: Execute workers → worker_results[] in Queen's memory
  Step 5.5: Watcher → watcher_report in Queen's memory
  Step 6: Record → errors.json, PROJECT_PLAN.json, COLONY_STATE.json, events.json

  Step 7 (NEW auto-learning):
  ┌─────────────────────────────────────────────────────┐
  │  Read: memory.json                                   │
  │  Inputs already in memory:                           │
  │    - worker_results (from Step 5c)                   │
  │    - watcher_report (from Step 5.5)                  │
  │    - errors.json (read/written in Step 6)            │
  │    - events.json (read/written in Step 6)            │
  │    - PROJECT_PLAN.json (task outcomes from Step 6)   │
  │                                                      │
  │  7a: Synthesize learnings                            │
  │      → Append to memory.json phase_learnings         │
  │      → Run memory-compress                           │
  │                                                      │
  │  7b: Emit FEEDBACK pheromone                         │
  │      → pheromone-validate                            │
  │      → Append to pheromones.json                     │
  │      → (optional REDIRECT if flagged_patterns)       │
  │                                                      │
  │  7c: pheromone-cleanup                               │
  │                                                      │
  │  7d: Write auto_learnings_extracted event             │
  │      → events.json                                   │
  │                                                      │
  │  7e: Display results (updated from current Step 7)   │
  │      → Show learnings + pheromones in output          │
  │      → Remove /ant:continue warning                   │
  │      → Update Next Steps                              │
  └─────────────────────────────────────────────────────┘

continue.md execution (after Phase 26):
  Step 1-3: Read state, determine next phase, display summary
  Step 4: Check events.json for auto_learnings_extracted for current phase
          → If found: skip to Step 5, print "already captured"
          → If not found: extract learnings (unchanged logic)
  Step 4.5: Skip if Step 4 was skipped, else emit pheromones (unchanged)
  Steps 5-8: Clean pheromones, write events, update state, display (unchanged)
```

## Step-by-Step Change Plan

### File 1: build.md

**What changes:**
1. Current Step 7 (Display Results) becomes a multi-part step:
   - 7a: Extract phase learnings (logic from continue.md Step 4, minus spawn_outcomes update)
   - 7b: Emit FEEDBACK pheromone (logic from continue.md Step 4.5)
   - 7c: Clean expired pheromones (logic from continue.md Step 5)
   - 7d: Write events (learnings_extracted + pheromone events)
   - 7e: Display results (updated current Step 7 with learning/pheromone sections, without /ant:continue warning)
2. Step progress checklist updated to reflect new sub-steps
3. "Next Steps" section updated — /ant:continue becomes optional, not required

**Lines budget:** Current Step 7 is ~55 lines (lines 519-573). The new Step 7 needs ~120 lines. Net increase: ~65 lines. Total build.md: ~640 lines. Under 700 limit.

### File 2: continue.md

**What changes:**
1. Step 4 gets a new preamble: check events.json for `auto_learnings_extracted` event matching current phase
2. If flag found: print skip message, jump past Steps 4 and 4.5
3. Step 4.5 gets the same conditional (or is skipped as part of the Step 4 skip)
4. No other changes needed

**Lines impact:** ~10-15 lines added at top of Step 4.

### File 3: No other files need changes

events.json, memory.json, pheromones.json, aether-utils.sh — all used as-is.

## Open Questions

1. **Should memory-compress print what was evicted?**
   - What we know: CONTEXT.md says "when compression/eviction happens, it must be visible to the user." The current memory-compress subcommand silently trims. It returns `{compressed:true, tokens:N}` but doesn't list what was removed.
   - What's unclear: Whether to modify memory-compress to return evicted items, or handle visibility in the prompt (Queen displays what was in memory before vs after).
   - Recommendation: For Phase 26, handle visibility in the prompt — the Queen can note the before/after count. Modifying memory-compress is a separate enhancement. This phase's scope is auto-extraction, not retention strategy overhaul.

2. **Should the 20-learning cap be adjusted?**
   - What we know: CONTEXT.md says "Claude determines the best way to work within or evolve it." The current cap is 20 in aether-utils.sh memory-compress (hardcoded jq: `if length > 20 then .[-20:] else . end`).
   - What's unclear: Whether 20 is sufficient for a long-running project with many phases.
   - Recommendation: Keep at 20 for Phase 26. The cap has been working. If projects hit it regularly, it can be adjusted later. Changing it requires modifying aether-utils.sh which is out of the minimal scope for this phase.

3. **Force re-extraction in continue.md?**
   - What we know: CONTEXT.md lists this as Claude's discretion. Some users might want to re-extract learnings even if build already did.
   - Recommendation: Support it. If continue.md detects the flag, print the skip message but also note: "Run /ant:continue --force to re-extract." The `--force` handling would simply skip the flag check. This is a small addition and provides user control.
   - Implementation note: `$ARGUMENTS` in continue.md could be checked for "--force" or "force". Simple string check, no argument parsing needed.

## Sources

### Primary (HIGH confidence)

- **continue.md** (`/Users/callumcowie/repos/Aether/.claude/commands/ant/continue.md`) — Full current learning extraction logic, Steps 4 and 4.5, 304 lines
- **build.md** (`/Users/callumcowie/repos/Aether/.claude/commands/ant/build.md`) — Current build flow after Phase 25 restructuring, 574 lines
- **aether-utils.sh** (`/Users/callumcowie/repos/Aether/.aether/aether-utils.sh`) — memory-compress, pheromone-validate, pheromone-cleanup, error-summary, error-pattern-check subcommands, 265 lines
- **memory.json** (`/Users/callumcowie/repos/Aether/.aether/data/memory.json`) — Current schema: `{phase_learnings:[], decisions:[], patterns:[]}`
- **events.json** (`/Users/callumcowie/repos/Aether/.aether/data/events.json`) — Current schema: `{events:[]}`
- **errors.json** (`/Users/callumcowie/repos/Aether/.aether/data/errors.json`) — Current schema: `{errors:[], flagged_patterns:[]}`
- **pheromones.json** (`/Users/callumcowie/repos/Aether/.aether/data/pheromones.json`) — Current schema: `{signals:[]}`
- **COLONY_STATE.json** (`/Users/callumcowie/repos/Aether/.aether/data/COLONY_STATE.json`) — spawn_outcomes structure
- **26-CONTEXT.md** (`/Users/callumcowie/repos/Aether/.planning/phases/26-auto-learning/26-CONTEXT.md`) — User decisions from discussion
- **25-RESEARCH.md** (`/Users/callumcowie/repos/Aether/.planning/phases/25-live-visibility/25-RESEARCH.md`) — Phase 25 architecture changes (dependency)
- **REQUIREMENTS.md** (`/Users/callumcowie/repos/Aether/.planning/REQUIREMENTS.md`) — LEARN-01, LEARN-02, LEARN-03 definitions
- **ROADMAP.md** (`/Users/callumcowie/repos/Aether/.planning/ROADMAP.md`) — Phase 26 success criteria

### Secondary (MEDIUM confidence)

None — all findings are from direct codebase analysis.

### Tertiary (LOW confidence)

None — this phase is entirely internal prompt modifications.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — all components are existing codebase files thoroughly reviewed, no changes to infrastructure
- Architecture: HIGH — the change is well-defined by existing continue.md logic and CONTEXT.md decisions
- Pitfalls: HIGH — identified from direct analysis of both build.md and continue.md flow, data flow tracing
- Code examples: HIGH — all examples are direct copies or minimal modifications of existing code

**Research date:** 2026-02-04
**Valid until:** Indefinite (internal system, no external dependencies)
