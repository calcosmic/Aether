# Phase 17: Integration & Dashboard - Research

**Researched:** 2026-02-03
**Domain:** Claude-native prompt engineering (command enrichment, JSON state integration, Bayesian spawn tracking)
**Confidence:** HIGH

## Summary

Phase 17 is the final phase of v3.0 "Restore the Soul." It integrates all JSON state files (COLONY_STATE.json, pheromones.json, errors.json, memory.json, events.json) into a unified dashboard via status.md, adds a phase review workflow to continue.md, and introduces Bayesian spawn outcome tracking to COLONY_STATE.json / build.md / continue.md / worker specs.

This phase modifies existing command prompts and worker specs -- no new files, no new commands, no Python, no bash. All work is text editing of .md prompt files and updating the COLONY_STATE.json schema. The "standard stack" is the existing Aether architecture: Claude Code skill prompts that instruct Claude how to read JSON files and format text output.

**Primary recommendation:** Execute as 3 plans mapping directly to the 3 roadmap items. Plan 17-01 (status dashboard) is independent. Plan 17-02 (phase review) is independent. Plan 17-03 (spawn tracking) is independent but touches build.md, continue.md, COLONY_STATE.json, and all 6 worker specs. All 3 can execute in parallel as Wave 1 since they touch different sections of the files.

## Standard Stack

This is a Claude-native project. There are no libraries, frameworks, or dependencies. The "stack" is:

### Core
| Component | Location | Purpose | Why Standard |
|-----------|----------|---------|--------------|
| Command prompts | `.claude/commands/ant/*.md` | Instruct Claude how to execute commands | Claude Code skill prompt model |
| Worker specs | `.aether/workers/*-ant.md` | Define worker behavior, knowledge, spawning | Claude-native worker architecture |
| JSON state files | `.aether/data/*.json` | Persist colony state across sessions | Simple, readable, no runtime needed |
| Task tool | Claude Code built-in | Spawn sub-agents (workers) | Only way to create sub-agents in Claude Code |

### State Files (all created by init.md Step 4)
| File | Schema | Dashboard Reads | Commands Write |
|------|--------|-----------------|----------------|
| `COLONY_STATE.json` | `{goal, state, current_phase, session_id, initialized_at, workers: {...}}` | YES - header, workers | init, build, continue |
| `pheromones.json` | `{signals: [{id, type, content, strength, half_life_seconds, created_at}]}` | YES - pheromone section | focus, redirect, feedback, continue (cleanup) |
| `PROJECT_PLAN.json` | `{goal, generated_at, phases: [{id, name, description, status, tasks, success_criteria}]}` | YES - phase progress | plan, build |
| `errors.json` | `{errors: [{id, category, severity, description, root_cause, phase, task_id, timestamp}], flagged_patterns: [...]}` | YES - error section | build |
| `memory.json` | `{phase_learnings: [...], decisions: [...], patterns: []}` | YES - memory section | continue (learnings), focus/redirect/feedback (decisions) |
| `events.json` | `{events: [{id, type, source, content, timestamp}]}` | YES - events section | init, build, continue, focus, redirect, feedback |

## Architecture Patterns

### Pattern 1: Additive Section Enhancement

status.md already has WORKERS, ACTIVE PHEROMONES, ERRORS, PHASE PROGRESS, and NEXT ACTIONS sections. Phase 17 adds MEMORY and EVENTS sections, enriches what already exists.

**Current status.md structure (182 lines):**
```
Step 1: Read State (reads 4 files)
Step 2: Compute Pheromone Decay
Step 3: Display Status
  - Header (box-drawing)
  - WORKERS section
  - ACTIVE PHEROMONES section
  - ERRORS section
  - PHASE PROGRESS section
  - NEXT ACTIONS section
```

**Target status.md structure:**
```
Step 1: Read State (reads ALL 6 JSON files -- add memory.json and events.json)
Step 2: Compute Pheromone Decay (unchanged)
Step 3: Display Status
  - Header (unchanged)
  - WORKERS section (unchanged)
  - ACTIVE PHEROMONES section (unchanged -- already has decay bars)
  - ERRORS section (unchanged -- already shows flagged patterns)
  - MEMORY section (NEW -- recent learnings from memory.json)
  - EVENTS section (NEW -- recent events from events.json)
  - PHASE PROGRESS section (unchanged)
  - NEXT ACTIONS section (unchanged)
```

**Key insight:** status.md already reads errors.json and shows the ERRORS section with flagged patterns and recent errors. DASH-02 (pheromone decay bars) and DASH-03 (error section) are ALREADY SATISFIED by the current status.md. Phase 17-01 needs to add DASH-01 completeness (memory + events sections) and DASH-04 (memory section with recent learnings).

### Pattern 2: Phase Review Before Advancement

continue.md currently does learning extraction (Step 3) but does not DISPLAY a phase completion summary to the user before advancing. Phase 17-02 adds a display step between learning extraction and phase advancement.

**Current continue.md flow:**
```
Step 1: Read State (6 files)
Step 2: Determine Next Phase
Step 3: Extract Phase Learnings (writes to memory.json)
Step 4: Clean Expired Pheromones
Step 5: Write Events
Step 6: Update Colony State
Step 7: Display Result (shows "Phase X approved. Advancing to Phase Y.")
```

**Target continue.md flow:**
```
Step 1: Read State (6 files) -- unchanged
Step 2: Determine Next Phase -- unchanged
Step 3: Phase Completion Summary (NEW DISPLAY -- show before advancing)
  - Tasks completed vs total
  - Key decisions made (from memory.json decisions array)
  - Errors encountered (from errors.json filtered by phase)
  - Time elapsed (from events.json phase_started to now)
Step 4: Extract Phase Learnings -- unchanged (renumber from 3)
Step 5: Clean Expired Pheromones -- unchanged (renumber from 4)
Step 6: Write Events -- unchanged (renumber from 5)
Step 7: Update Colony State -- unchanged (renumber from 6)
Step 8: Display Result -- unchanged (renumber from 7)
```

**Key insight:** continue.md already extracts learnings and stores them. REV-03 is ALREADY SATISFIED. Phase 17-02 adds REV-01 (display summary before advancing) and REV-02 (show tasks completed, decisions, errors).

### Pattern 3: Bayesian Spawn Tracking (alpha/beta in COLONY_STATE.json)

This pattern requires changes across 4 file types:
1. **COLONY_STATE.json** -- add `spawn_outcomes` field
2. **build.md** -- record spawn event when Phase Lead is spawned (Step 5)
3. **continue.md** -- record spawn success/failure on phase completion
4. **Worker specs (all 6)** -- check spawn confidence before spawning

**Schema addition to COLONY_STATE.json:**
```json
{
  "goal": "...",
  "state": "...",
  "current_phase": 0,
  "session_id": "...",
  "initialized_at": "...",
  "workers": {...},
  "spawn_outcomes": {
    "colonizer": {"alpha": 1, "beta": 1, "total_spawns": 0, "successes": 0, "failures": 0},
    "route-setter": {"alpha": 1, "beta": 1, "total_spawns": 0, "successes": 0, "failures": 0},
    "builder": {"alpha": 1, "beta": 1, "total_spawns": 0, "successes": 0, "failures": 0},
    "watcher": {"alpha": 1, "beta": 1, "total_spawns": 0, "successes": 0, "failures": 0},
    "scout": {"alpha": 1, "beta": 1, "total_spawns": 0, "successes": 0, "failures": 0},
    "architect": {"alpha": 1, "beta": 1, "total_spawns": 0, "successes": 0, "failures": 0}
  }
}
```

**Bayesian confidence formula:**
```
confidence = alpha / (alpha + beta)
```

Starting with alpha=1, beta=1 (uniform prior, confidence=0.50).
- Success: alpha += 1 (after 1 success: 2/(2+1) = 0.67)
- Failure: beta += 1 (after 1 failure: 1/(1+2) = 0.33)
- After 4 successes, 1 failure: 5/(5+2) = 0.71

**When workers check confidence:**
Workers read COLONY_STATE.json `spawn_outcomes` for the target caste. If confidence < 0.3, consider an alternative caste. This is advisory, not blocking -- workers still decide autonomously.

**Where spawn events are recorded:**
- `build.md` Step 5 spawns the Phase Lead ant. After the ant returns (Step 6), build.md determines if the phase succeeded or failed. It updates `spawn_outcomes` for the caste that was spawned as Phase Lead.
- BUT -- build.md does NOT pick a caste. It spawns a generic ant that self-organizes. The spawned ant's report should indicate what castes were used.
- Alternative approach: track at the SUB-SPAWN level. Each worker spec's spawning section records outcomes. But workers don't persist state -- they run and return.

**Resolution:** The simplest Claude-native approach is:
1. `build.md` records the overall phase spawn (the Phase Lead) as a generic "spawn" event. Since the Phase Lead is always a generic ant (not a specific caste), track spawn outcomes at the phase level, not per-caste.
2. Worker specs read `spawn_outcomes` and use it to inform spawning decisions. When a worker spawns a sub-ant of a specific caste, it cannot record the outcome persistently (sub-agents don't write to COLONY_STATE.json). Instead, the parent ant reports outcomes in its report, and `continue.md` aggregates.
3. `continue.md` at phase boundary reviews the build report events, infers which castes were used, and updates `spawn_outcomes` per caste.

**Simpler alternative (RECOMMENDED):** Track spawn outcomes ONLY at the phase-level through build.md and continue.md. Per-caste tracking would require workers to write to COLONY_STATE.json mid-execution (unreliable). Instead:
- build.md records which castes the Phase Lead spawned (from the ant's report)
- continue.md records success/failure per caste based on the phase outcome
- Worker specs read the per-caste confidence when deciding to spawn

This requires build.md Step 6 (Record Outcome) to parse the ant's report for spawned castes and update `spawn_outcomes` accordingly.

### Pattern 4: Box-Drawing Formatting Standards

All commands use the same visual language established in Phase 14:

```
+=====================================================+
|  AETHER COLONY STATUS                                |
|-----------------------------------------------------|
|  Session: <session_id>                               |
|  State:   <state>                                    |
|  Goal:    "<goal>"                                   |
+=====================================================+
```

Section dividers: `---------------------------------------------------`
Section headers: plain text like `WORKERS`, `ACTIVE PHEROMONES`, `ERRORS`, etc.
Pheromone bars: `[====================] 1.00` (20 chars, = filled, spaces empty)
Step progress: `  ✓ Step N: Description`

### Anti-Patterns to Avoid

- **Don't make status.md read files sequentially.** All 6 JSON files should be read in parallel (Step 1 already says "in parallel").
- **Don't add complex formatting logic.** Keep display templates as simple text blocks with placeholders.
- **Don't block spawning on low confidence.** The spawn confidence check is advisory -- workers decide autonomously.
- **Don't track spawn outcomes at sub-ant level.** Only track at Phase Lead level via build.md/continue.md.
- **Don't add new commands.** Everything goes into existing status.md, continue.md, build.md.
- **Don't over-count sections in status.md.** Keep new sections (MEMORY, EVENTS) compact -- 5-10 lines max each.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Bayesian math | Complex formula system | `confidence = alpha / (alpha + beta)` | One-line formula, no utility needed |
| Event filtering | Timestamp parsing library | Prose instruction: "compare timestamps" | Claude can compare ISO-8601 strings |
| State persistence | File locking or transactions | Atomic Write tool (built-in) | Write tool is atomic; race conditions impossible in single-agent commands |
| Dashboard rendering | Template engine | Text block templates in prompt | Claude follows formatting instructions directly |

**Key insight:** This is a Claude-native project. There are no libraries to use and no code to run. Every "implementation" is a text editing task on .md prompt files.

## Common Pitfalls

### Pitfall 1: Partially Satisfied Requirements
**What goes wrong:** Assuming all DASH-* requirements need full implementation when some are already satisfied.
**Why it happens:** Not checking current status.md against requirements before planning.
**How to avoid:** Cross-reference each requirement against current file content.
**Warning signs:** Plan creates tasks for features that already exist in status.md.

**Current satisfaction analysis:**
- DASH-01 (full colony health with workers, pheromones, errors, memory, events): PARTIALLY SATISFIED. status.md already shows workers, pheromones, errors, phase progress. MISSING: memory section, events section.
- DASH-02 (pheromone decay bars): ALREADY SATISFIED. status.md Step 2 computes decay, Step 3 shows bars.
- DASH-03 (error section with recent errors and flagged patterns): ALREADY SATISFIED. status.md has ERRORS section.
- DASH-04 (memory section with recent learnings): NOT SATISFIED. status.md does not read memory.json.
- REV-01 (phase completion summary): NOT SATISFIED. continue.md shows result but no summary before advancing.
- REV-02 (tasks completed, decisions, errors): NOT SATISFIED.
- REV-03 (learning extraction stores to memory.json): ALREADY SATISFIED. continue.md Step 3 does this.
- SPAWN-01 through SPAWN-04: NOT SATISFIED. No spawn tracking exists.

### Pitfall 2: Spawn Tracking Scope Creep
**What goes wrong:** Trying to track every sub-ant spawn at every depth level.
**Why it happens:** The old system (outcome_tracker.py, 355 lines) tracked granularly.
**How to avoid:** Only track at Phase Lead level. build.md spawns one ant, that ant's success/failure is the outcome.
**Warning signs:** Plan asks workers to write to COLONY_STATE.json during execution.

### Pitfall 3: status.md Length Explosion
**What goes wrong:** Adding too many sections makes status.md unwieldy (currently 182 lines, was 456 before rebuild).
**Why it happens:** Each section template requires formatting instructions.
**How to avoid:** Keep new sections (MEMORY, EVENTS) compact. Show last 3 learnings, last 5 events. Use conditional display: "If array is empty, skip section."
**Warning signs:** status.md exceeding 250 lines.

### Pitfall 4: continue.md Phase Review Duplication
**What goes wrong:** Phase review displays duplicate information from what's already in Step 7 output.
**Why it happens:** Step 7 already shows learnings extracted. Adding a summary risks redundancy.
**How to avoid:** Phase Completion Summary (new step) shows DIFFERENT information than the final output. Summary = retrospective (what happened). Final output = prospective (what's next).
**Warning signs:** Same data shown twice in continue output.

### Pitfall 5: init.md Must Create spawn_outcomes Field
**What goes wrong:** COLONY_STATE.json created by init.md doesn't include spawn_outcomes, breaking later reads.
**Why it happens:** init.md Step 3 writes COLONY_STATE.json with current schema (no spawn_outcomes).
**How to avoid:** Plan 17-03 must update init.md Step 3 to include spawn_outcomes in the COLONY_STATE.json template.
**Warning signs:** build.md tries to read spawn_outcomes and gets undefined.

### Pitfall 6: Worker Spec Spawn Confidence Section Placement
**What goes wrong:** Adding spawn confidence check in wrong location within worker specs.
**Why it happens:** Worker specs have a specific section order established in Phase 16.
**How to avoid:** Add spawn confidence check within the existing "You Can Spawn Other Ants" section, before the spawn prompt. Not a new section -- an enhancement to an existing section.
**Warning signs:** Creating a new "## Spawn Confidence" section that breaks the established section order.

## Code Examples

### Example 1: MEMORY Section for status.md

```markdown
```
MEMORY
```

If `memory.json` was read successfully and has content:

Display recent phase learnings (last 3 from `phase_learnings` array, newest first):
```
  Recent Learnings:
    Phase <phase>: <first learning from learnings array>
    Phase <phase>: <first learning from learnings array>
    Phase <phase>: <first learning from learnings array>
```

Display decision count:
```
  Decisions logged: <count of decisions array>
```

If `phase_learnings` array is empty and `decisions` array is empty:
```
  (no memory recorded)
```

If `memory.json` doesn't exist or couldn't be read, skip this section silently.
```

### Example 2: EVENTS Section for status.md

```markdown
```
EVENTS
```

If `events.json` was read successfully and has content:

Display recent events (last 5 from `events` array, newest first):
```
  Recent:
    [<type>] <content> (<relative time, e.g., "2m ago", "1h ago">)
```

If `events` array is empty:
```
  (no events recorded)
```

If `events.json` doesn't exist or couldn't be read, skip this section silently.
```

### Example 3: Phase Completion Summary for continue.md

```markdown
### Step 3: Phase Completion Summary

Display a summary of the completed phase:

```
---------------------------------------------------
PHASE <N> REVIEW: <phase_name>
---------------------------------------------------

  Tasks:
    [x] <task_id>: <description>      (or [ ] for incomplete)
    [x] <task_id>: <description>
    ...
    Completed: <N>/<total>

  Errors:
    <count> errors encountered
    (list severity counts: N critical, N high, N medium, N low)

  Decisions:
    <count> decisions logged during this phase
    (list last 3 decisions: "<content>")

  Learnings Being Extracted:
    - <learning 1>
    - <learning 2>

---------------------------------------------------
```

If no errors were encountered during this phase:
```
  Errors: None
```

If no decisions were logged during this phase:
```
  Decisions: None
```
```

### Example 4: spawn_outcomes Schema in COLONY_STATE.json

```json
{
  "goal": "Build a REST API with authentication",
  "state": "READY",
  "current_phase": 2,
  "session_id": "session_1706900000_abc",
  "initialized_at": "2026-02-03T00:00:00Z",
  "workers": {
    "colonizer": "idle",
    "route-setter": "idle",
    "builder": "idle",
    "watcher": "idle",
    "scout": "idle",
    "architect": "idle"
  },
  "spawn_outcomes": {
    "colonizer":    {"alpha": 1, "beta": 1, "total_spawns": 0, "successes": 0, "failures": 0},
    "route-setter": {"alpha": 1, "beta": 1, "total_spawns": 0, "successes": 0, "failures": 0},
    "builder":      {"alpha": 1, "beta": 1, "total_spawns": 0, "successes": 0, "failures": 0},
    "watcher":      {"alpha": 1, "beta": 1, "total_spawns": 0, "successes": 0, "failures": 0},
    "scout":        {"alpha": 1, "beta": 1, "total_spawns": 0, "successes": 0, "failures": 0},
    "architect":    {"alpha": 1, "beta": 1, "total_spawns": 0, "successes": 0, "failures": 0}
  }
}
```

### Example 5: Spawn Confidence Check in Worker Specs

Addition to the existing "You Can Spawn Other Ants" section:

```markdown
### Spawn Confidence Check

Before spawning, read `.aether/data/COLONY_STATE.json` and check `spawn_outcomes` for the target caste:

```
confidence = alpha / (alpha + beta)
```

**Interpretation:**
- confidence >= 0.5: Spawn freely -- this caste has a positive track record
- confidence 0.3-0.5: Spawn with caution -- consider if another caste could handle the task
- confidence < 0.3: Prefer an alternative caste -- this caste has a poor track record

**Example:**
```
spawn_outcomes.scout: {alpha: 3, beta: 4}
confidence = 3 / (3 + 4) = 0.43

Scout has marginal confidence. Consider: could a colonizer handle this
research task instead? If the task specifically needs web research (scout
specialty), spawn anyway. If it's codebase exploration, use a colonizer.
```

This is advisory, not blocking. You always retain autonomy to spawn any caste based on task requirements.
```

### Example 6: build.md Spawn Outcome Recording

Addition to build.md Step 6 (Record Outcome):

```markdown
**Record Spawn Outcomes:** Read `.aether/data/COLONY_STATE.json`. Look at the ant's report to identify which castes were spawned (look for mentions of "spawned a builder", "spawned a scout", etc. in the report).

For each caste that was spawned during the phase:
- If the phase completed successfully: increment `alpha` and `successes` for that caste
- If the phase failed: increment `beta` and `failures` for that caste
- Increment `total_spawns` for that caste regardless

If the report doesn't clearly identify which castes were spawned, skip spawn outcome tracking for this phase.

Use the Write tool to write the updated COLONY_STATE.json.
```

## State of the Art

| Before Phase 17 | After Phase 17 | Impact |
|------------------|----------------|--------|
| status.md reads 4 files | status.md reads 6 files (adds memory.json, events.json) | Full colony health dashboard |
| No memory/events display | MEMORY and EVENTS sections in dashboard | Users see colony learning and activity |
| continue.md advances silently | continue.md shows phase review first | Quality gate with retrospective |
| No spawn tracking | Bayesian alpha/beta per caste | Colony learns which castes perform well |
| Workers spawn blindly | Workers check confidence before spawning | Informed autonomous decisions |
| COLONY_STATE.json has 6 fields | COLONY_STATE.json has 7 fields (adds spawn_outcomes) | Richer colony state |

## Dependency Analysis

### What Phase 17 Depends On

**Phase 15 (Infrastructure State):**
- errors.json, memory.json, events.json exist and are populated by commands
- Error logging in build.md (ALREADY DONE)
- Learning extraction in continue.md (ALREADY DONE)
- Event writing in init/build/continue (ALREADY DONE)

**Phase 16 (Worker Knowledge):**
- Worker specs have "You Can Spawn Other Ants" section (ALREADY EXISTS -- add spawn confidence check here)
- Worker specs have event awareness and memory reading (ALREADY EXISTS)

Both dependencies are fully satisfied. Phase 17 can proceed.

### File Modification Map

| File | Plan 17-01 | Plan 17-02 | Plan 17-03 | Conflict? |
|------|------------|------------|------------|-----------|
| status.md | MAJOR (add MEMORY, EVENTS sections) | - | - | No |
| continue.md | - | MAJOR (add phase review step) | MINOR (add spawn outcome update) | Low risk -- different steps |
| build.md | - | - | MINOR (add spawn outcome recording in Step 6) | No |
| init.md | - | - | MINOR (add spawn_outcomes to COLONY_STATE template) | No |
| COLONY_STATE.json | - | - | SCHEMA CHANGE (add spawn_outcomes) | No |
| Worker specs (6 files) | - | - | MINOR (add spawn confidence check to spawning section) | No |

**Parallelization:** All 3 plans can execute as Wave 1. The only shared file (continue.md) is modified in different steps by 17-02 and 17-03, so conflicts are minimal if 17-02 runs first.

**Recommended execution order within Wave 1:** 17-01 and 17-02 first (independent), then 17-03 (touches more files, benefits from 17-01 and 17-02 being done).

## Plan Scope Summary

### Plan 17-01: Status Dashboard Enhancement
**Files modified:** 1 (status.md)
**What changes:**
- Step 1: Add memory.json and events.json to the parallel read list
- Add MEMORY section after ERRORS section
- Add EVENTS section after MEMORY section
- Verify all 6 state file sections are present and correctly formatted

**Requirements satisfied:** DASH-01, DASH-04
**Requirements already satisfied:** DASH-02, DASH-03

### Plan 17-02: Phase Review Workflow
**Files modified:** 1 (continue.md)
**What changes:**
- Insert new Step 3: Phase Completion Summary (display before advancing)
- Renumber subsequent steps (4-8 instead of 3-7)
- Update step progress display to show 8 steps
- Phase review shows: tasks completed, errors encountered, decisions made

**Requirements satisfied:** REV-01, REV-02
**Requirements already satisfied:** REV-03

### Plan 17-03: Spawn Outcome Tracking
**Files modified:** 9 (init.md, build.md, continue.md, COLONY_STATE.json schema, 6 worker specs)
**What changes:**
- init.md Step 3: Add spawn_outcomes field to COLONY_STATE.json template
- build.md Step 6: Add spawn outcome recording logic
- continue.md: Add spawn outcome update on phase success/failure
- All 6 worker specs: Add spawn confidence check to "You Can Spawn Other Ants" section

**Requirements satisfied:** SPAWN-01, SPAWN-02, SPAWN-03, SPAWN-04

## Open Questions

1. **Phase Lead caste identification**
   - What we know: build.md spawns a generic ant that self-organizes. The ant may spawn sub-ants of specific castes.
   - What's unclear: How reliably can build.md Step 6 parse the ant's free-text report to identify which castes were spawned?
   - Recommendation: Use best-effort parsing. Look for "spawned a {caste}" patterns in the report. If unclear, don't update spawn_outcomes for that phase. This is acceptable because the Bayesian prior (alpha=1, beta=1) already provides a reasonable default.

2. **Events section time formatting**
   - What we know: Events have ISO-8601 timestamps.
   - What's unclear: Should the dashboard show absolute timestamps or relative times ("2m ago")?
   - Recommendation: Use relative times for readability. Claude can compute relative time from ISO-8601 strings.

3. **Spawn confidence threshold tuning**
   - What we know: Decided thresholds are >= 0.5 (go), 0.3-0.5 (caution), < 0.3 (prefer alternative).
   - What's unclear: With alpha=1, beta=1 priors, confidence starts at 0.50. One failure drops to 0.33. Is this too aggressive?
   - Recommendation: Start with these thresholds. The advisory nature means workers can override. Real usage will show if thresholds need adjustment.

## Sources

### Primary (HIGH confidence)
- `.claude/commands/ant/status.md` -- current 182-line status command (full read)
- `.claude/commands/ant/continue.md` -- current 158-line continue command (full read)
- `.claude/commands/ant/build.md` -- current 293-line build command (full read)
- `.claude/commands/ant/init.md` -- current 175-line init command (full read)
- `.aether/data/COLONY_STATE.json` -- current schema (full read)
- All 6 worker specs in `.aether/workers/` -- current ~210-line specs (full read)
- `.planning/ROADMAP.md` -- v3.0 roadmap with Phase 17 requirements (full read)
- `.planning/research/V3_LOST_FEATURES.md` -- what was lost and needs restoring (full read)
- `.planning/phases/16-worker-knowledge/16-RESEARCH.md` -- Phase 16 research (full read)

### Secondary (MEDIUM confidence)
- `.planning/v2-MILESTONE-AUDIT.md` -- v2.0 audit showing integration patterns

### Tertiary (LOW confidence)
- None -- all findings based on direct file reads of the codebase.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH -- directly read all files, no external dependencies
- Architecture: HIGH -- patterns derived from existing codebase conventions
- Pitfalls: HIGH -- identified through cross-referencing requirements vs current state
- Spawn tracking design: MEDIUM -- the caste identification from free-text reports is inherently fuzzy

**Research date:** 2026-02-03
**Valid until:** indefinite (this is an internal project, not dependent on external library versions)
