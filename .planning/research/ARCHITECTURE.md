# Architecture Research: Colony Integration Gap Fixes

**Domain:** Multi-agent colony system — wiring existing components together
**Researched:** 2026-03-14
**Confidence:** HIGH (based on direct reading of all relevant source files)

## What This Milestone Is

This is not a feature build. It is an integration wiring milestone. The four integration gaps are:

1. **Decisions -> Pheromones (PHER-01):** Key architectural decisions captured in CONTEXT.md never become pheromone signals. Future phases miss them.
2. **Learnings -> Instincts (promotion path):** Phase learnings are extracted and passed through `memory-capture`, but the promotion-proposal UI (`learning-check-promotion` / `learning-approve-proposals`) doesn't run at the right moments or is skipped silently when it should be surfaced.
3. **Midden -> Behavior (PHER-02):** Recurring error categories in `midden.json` are read for context but never converted to REDIRECT signals that steer workers away from known failure modes.
4. **Memory capture consistency:** `memory-capture` is called from some playbook steps but missing in others where failures and successes are recorded.

**All four fixes involve calling existing `aether-utils.sh` subcommands from existing playbook steps.** No new subcommands are required. The implementation surface is playbook markdown files and one or two shell functions.

## System Overview

```
+-------------------------------------------------------------------+
|                    COMMAND LAYER (orchestrators)                   |
|   build.md  continue.md  (load split playbooks by reference)      |
+-------------------------------------------------------------------+
                              |
          +-------------------+-------------------+
          |                                       |
          v                                       v
+-----------------------+             +---------------------------+
|    BUILD PLAYBOOKS    |             |    CONTINUE PLAYBOOKS     |
|                       |             |                           |
|  build-prep.md        |             |  continue-verify.md       |
|  build-context.md     |             |  continue-gates.md        |
|  build-wave.md        |             |  continue-advance.md      |
|  build-verify.md      |             |  continue-finalize.md     |
|  build-complete.md    |             |                           |
+-----------+-----------+             +-------------+-------------+
            |                                       |
            |   calls                               |   calls
            v                                       v
+-------------------------------------------------------------------+
|                      aether-utils.sh (150 subcommands)            |
|                                                                   |
|  MEMORY PIPELINE:                                                 |
|    memory-capture -> learning-observe -> pheromone-write          |
|                   -> learning-promote-auto -> instinct-create     |
|                                                                   |
|  PROMOTION PIPELINE:                                              |
|    learning-check-promotion -> learning-approve-proposals         |
|    learning-promote-auto -> queen-promote -> QUEEN.md             |
|                                                                   |
|  PHEROMONE PIPELINE:                                              |
|    pheromone-write -> pheromones.json                             |
|    colony-prime -> prompt_section (injected into worker prompts)  |
|                                                                   |
|  MIDDEN PIPELINE:                                                 |
|    midden-write -> midden.json                                    |
|    midden-recent-failures -> failure context for workers          |
|                                                                   |
|  INSTINCT PIPELINE:                                               |
|    instinct-create -> COLONY_STATE.json .memory.instincts[]       |
|    instinct-read -> included in colony-prime prompt_section       |
+-------------------------------------------------------------------+
                              |
          +-------------------+-------------------+
          |                                       |
          v                                       v
+---------------------------+         +---------------------------+
|   DATA LAYER (local only)  |         |   KNOWLEDGE LAYER         |
|                            |         |                           |
|  .aether/data/             |         |  .aether/CONTEXT.md       |
|    COLONY_STATE.json       |         |  .aether/QUEEN.md         |
|    pheromones.json         |         |  .aether/HANDOFF.md       |
|    learning-obs.json       |         |                           |
|    midden/midden.json      |         |                           |
|    midden/build-failures.md|         |                           |
|    midden/test-failures.md |         |                           |
+---------------------------+         +---------------------------+
```

## Component Responsibilities (Integration-Relevant)

| Component | Responsibility | Current Gap |
|-----------|----------------|-------------|
| `continue-advance.md` Step 2.1b | Emit FEEDBACK pheromone per CONTEXT.md decision | GAP-1: Already present in playbook but must be verified to fire every continue run |
| `continue-advance.md` Step 2.1c | Emit REDIRECT per midden category with 3+ occurrences | GAP-3: Step exists but threshold logic must match actual midden data format |
| `continue-advance.md` Step 2.5 | memory-capture for each extracted learning | GAP-2: Currently calls memory-capture but promotion-check step (2.1.5) may silently skip |
| `continue-advance.md` Step 3a | instinct-create from midden error patterns | GAP-3/4: Uses midden-recent-failures but only creates instincts, doesn't emit REDIRECT |
| `build-wave.md` Step 5.2 | memory-capture on builder failure | GAP-4: Present in MEM-02 block but needs consistent source tagging |
| `build-verify.md` Step 5.7 | memory-capture on chaos findings | GAP-4: Present in MEM-02 block, pattern needs to be confirmed complete |
| `build-verify.md` Step 5.8 | memory-capture on watcher failures | GAP-4: Present in MEM-02 block, same confirmation needed |
| `build-complete.md` Step 5.10 | learning-check-promotion after build | GAP-2: Present but runs after every build; must handle no-proposals case silently |
| `context-update decision` | auto-emit FEEDBACK pheromone on decision record | GAP-1: `context-update` already calls pheromone-write for decisions at line 508 of aether-utils.sh |

## Integration Points: Where Each Gap Is Wired

### Gap 1: Decisions → Pheromones (PHER-01)

**Current state:** `context-update decision` (aether-utils.sh line ~508) already emits a FEEDBACK pheromone with `source: "system:decision"` when a decision is recorded. The continue-advance.md Step 2.1b reads the CONTEXT.md "Recent Decisions" table and emits FEEDBACK pheromones with `source: "auto:decision"`.

**The gap:** Deduplication in 2.1b checks for both `"auto:decision"` and `"system:decision"` sources — this is correct. However, if `context-update decision` is not called consistently when decisions are made (e.g., during build rather than just continue), some decisions are never promoted.

**Integration point:** `continue-advance.md` Step 2.1b is the primary wiring point. This step already exists and is correct. The milestone should verify it fires on every continue run, not just when learnings exist.

**Subcommands called:**
```
pheromone-write FEEDBACK "[decision] {text}" --strength 0.6 --source "auto:decision" --ttl "30d"
```

**New vs. modified:** MODIFIED — Step 2.1b already exists, may need robustness fixes.

### Gap 2: Learnings → Instincts (Promotion Path)

**Current state:** `continue-advance.md` has:
- Step 2 extracts learnings into COLONY_STATE.json `memory.phase_learnings`
- Step 2.5 calls `memory-capture "learning"` for each extracted learning
- Step 2.1.5 calls `learning-check-promotion` and if proposals exist, calls `learning-approve-proposals`
- Step 2.1.6 batch-sweeps with `learning-promote-auto`
- Step 3 calls `instinct-create` directly for observed patterns
- Step 3a calls `instinct-create` from midden error patterns

**The gap:** The promotion pipeline (Steps 2.1.5 and 2.1.6) is in continue-advance.md but step ordering means batch promotion runs AFTER the phase advance. The silent-skip condition (no proposals = no output) is correct per spec, but the ordering matters: promotion should happen before state is written (Step 4), not after.

**Integration point:** The ordering in `continue-advance.md` between Steps 2.5 → 2.1.5 → 2.1.6 → 4. Verify promotion runs before phase advance, not after.

**Subcommands called:**
```
memory-capture "learning" "{claim}" "pattern" "worker:continue"
learning-check-promotion
learning-approve-proposals [--verbose]
learning-promote-auto "{wisdom_type}" "{content}" "{colony}" "learning"
instinct-create --trigger "..." --action "..." --confidence N --domain "..." --source "..." --evidence "..."
```

**New vs. modified:** MODIFIED — ordering fix in continue-advance.md.

### Gap 3: Midden → Behavior (PHER-02)

**Current state:** `continue-advance.md` Step 2.1c already queries `midden-recent-failures 50` and emits REDIRECT pheromones for categories with 3+ occurrences. Step 3a also creates instincts from midden error patterns.

**The gap:** Two-part. First, the midden stores all entries (not just failures — it includes `"coverage"`, `"performance"`, `"refactoring"`, `"integration"`, `"security"`, `"quality"` categories from various agents). The `midden-recent-failures` subcommand returns ALL entries sorted by timestamp, not filtered by failure categories. The Step 2.1c category grouping works correctly on this data, but the 3+ threshold means low-volume genuine failures never trigger REDIRECT.

Second, `build-wave.md` Step 5.2 reads midden at the start of each wave (`midden-recent-failures 5`) to inject context into builder prompts, but this is a display-only read — it does not emit pheromones. Failures found here should also feed back.

**Integration points:**
1. `continue-advance.md` Step 2.1c — existing, verify threshold and deduplication logic is robust
2. `build-wave.md` Step 5.2 — existing midden read, no pheromone emission (acceptable — pheromone emission belongs in continue, not build)

**Subcommands called:**
```
midden-recent-failures 50
pheromone-write REDIRECT "[error-pattern] ..." --strength 0.7 --source "auto:error" --ttl "30d"
memory-capture "resolution" "..." "pattern" "worker:continue"
```

**New vs. modified:** MODIFIED — verify Step 2.1c threshold and deduplication.

### Gap 4: Memory Capture Consistency

**Current state:** `memory-capture` is called from:
- `build-wave.md` Step 5.2 on builder failure (MEM-02 block)
- `build-verify.md` Step 5.7 on chaos findings (MEM-02 block)
- `build-verify.md` Step 5.8 on watcher failures (MEM-02 block)
- `continue-advance.md` Step 2.5 on learnings

**The gap:** Success events are not captured via `memory-capture`. When a builder succeeds, no observation is recorded in `learning-observations.json`, so the pattern never accumulates toward promotion threshold. The midden-write calls on success (e.g., Gatekeeper findings, Auditor scores, Measurer baselines) record context but don't trigger the pheromone/promotion pipeline.

**Integration points:**
1. `build-complete.md` Step 5.9 — synthesis step; add `memory-capture "success"` for tasks_completed with patterns_observed from learning section of synthesis JSON
2. `build-verify.md` Step 5.7 — after chaos completion, add `memory-capture "success"` for `overall_resilience: "strong"`

**Subcommands called (new calls to add):**
```
memory-capture "success" "{pattern.trigger}: {pattern.action}" "pattern" "worker:builder"
memory-capture "failure" "{issue_title}" "failure" "worker:chaos"
```

**New vs. modified:** NEW calls in build-complete.md Step 5.9 and build-verify.md Step 5.7.

## Data Flow: Memory Pipeline

The memory pipeline is the central nervous system of all four integration fixes:

```
Event Source (builder failure / chaos finding / learning / decision)
    |
    v
memory-capture <event_type> <content> <wisdom_type> <source>
    |
    +-- learning-observe -> learning-observations.json
    |       content_hash: dedup key
    |       observation_count: increments on repeat
    |       colonies: [colony_name]
    |
    +-- pheromone-write -> pheromones.json
    |       type: REDIRECT (failure) | FEEDBACK (learning/success)
    |       strength: 0.7 (failure) | 0.6 (learning/success) | 0.75 (resolution)
    |       source: worker:continue | worker:builder | worker:chaos | worker:watcher
    |
    +-- learning-promote-auto -> (if threshold met)
            |
            +-- instinct-create -> COLONY_STATE.json .memory.instincts[]
            |
            +-- queen-promote -> QUEEN.md (appended pattern/redirect/philosophy)
```

## Data Flow: Colony Prime (Worker Context Injection)

Every worker prompt is primed with accumulated colony knowledge via `colony-prime`:

```
colony-prime --compact
    |
    +-- reads pheromones.json -> active signals (FOCUS/REDIRECT/FEEDBACK)
    +-- reads COLONY_STATE.json .memory.instincts[] -> instinct list
    +-- reads QUEEN.md -> queen wisdom entries
    +-- reads CONTEXT.md -> recent decisions, current phase
    |
    v
prompt_section (formatted markdown)
    |
    v
Builder/Watcher/Chaos worker prompts (injected as {prompt_section})
```

This means: pheromones emitted in `continue-advance.md` are picked up by `colony-prime` before the next build's workers are spawned. The signal propagation chain is:

```
continue run N -> pheromone-write -> pheromones.json
                                         |
                                         v
build run N+1 -> colony-prime -> prompt_section -> builder worker sees signal
```

## Playbook Step Integration Map

Complete map of which playbook steps call which subcommands for the four gaps:

### Build Flow

| Playbook | Step | Subcommand | Gap | Status |
|----------|------|-----------|-----|--------|
| build-wave.md | 5.2 (builder failure) | `memory-capture "failure" ...` | GAP-4 | EXISTS (MEM-02) |
| build-wave.md | 5.2 (wave start) | `midden-recent-failures 5` | GAP-3 | EXISTS (display only) |
| build-verify.md | 5.7 (chaos critical finding) | `memory-capture "failure" ...` | GAP-4 | EXISTS (MEM-02) |
| build-verify.md | 5.7 (chaos strong resilience) | `memory-capture "success" ...` | GAP-4 | MISSING |
| build-verify.md | 5.8 (watcher failure) | `memory-capture "failure" ...` | GAP-4 | EXISTS (MEM-02) |
| build-complete.md | 5.9 (synthesis patterns_observed) | `memory-capture "success" ...` | GAP-4 | MISSING |
| build-complete.md | 5.10 (post-build) | `learning-check-promotion` | GAP-2 | EXISTS |
| build-complete.md | 5.10 (post-build) | `learning-approve-proposals` | GAP-2 | EXISTS |

### Continue Flow

| Playbook | Step | Subcommand | Gap | Status |
|----------|------|-----------|-----|--------|
| continue-advance.md | 2.5 (per learning) | `memory-capture "learning" ...` | GAP-2 | EXISTS |
| continue-advance.md | 2.1.5 | `learning-check-promotion` | GAP-2 | EXISTS |
| continue-advance.md | 2.1.5 | `learning-approve-proposals` | GAP-2 | EXISTS |
| continue-advance.md | 2.1.6 | `learning-promote-auto` (batch) | GAP-2 | EXISTS |
| continue-advance.md | 2.1b | `pheromone-write FEEDBACK "[decision] ..."` | GAP-1 | EXISTS |
| continue-advance.md | 2.1c | `midden-recent-failures 50` | GAP-3 | EXISTS |
| continue-advance.md | 2.1c | `pheromone-write REDIRECT "[error-pattern] ..."` | GAP-3 | EXISTS |
| continue-advance.md | 2.1c | `memory-capture "resolution" ...` | GAP-3 | EXISTS |
| continue-advance.md | 3 | `instinct-create` (success patterns) | GAP-2 | EXISTS |
| continue-advance.md | 3a | `instinct-create` (midden errors) | GAP-3 | EXISTS |

## What Is New vs. What Is Modified

### New (does not exist today)

1. `memory-capture "success"` call in `build-verify.md` Step 5.7 — after chaos reports `overall_resilience: "strong"`, capture it as a positive signal.

2. `memory-capture "success"` call in `build-complete.md` Step 5.9 — after synthesis collects `learning.patterns_observed`, emit each pattern through the memory pipeline.

Both calls are additions of ~5-10 lines to existing steps.

### Modified (exists, needs verification or fixing)

1. `continue-advance.md` Step 2.1b — verify it fires on every continue run regardless of whether learnings were extracted. Current code only runs if `[[ -n "$decisions" ]]`. This is correct behavior; the question is whether `decisions` extraction via awk is robust against CONTEXT.md format variations.

2. `continue-advance.md` Step 2.1c — verify the category grouping jq query correctly handles the midden.json format. The `midden-recent-failures` subcommand returns `{count, failures[{timestamp, category, source, message}]}`. The Step 2.1c jq extracts `.failures[].category`, which matches. Threshold is 3+ occurrences. Verify deduplication check queries `pheromones.json .signals[]` with correct field paths.

3. `continue-advance.md` Step 2.1.5 ordering — confirm this step runs before Step 4 (Advance State). Current playbook ordering: Step 2 (extract learnings) → Step 2.5 (memory-capture) → Step 2.1 (auto-emit pheromones) → Step 2.1.5 (promotion proposals) → Step 2.1.6 (batch promotion) → Step 2.2 (handoff) → Step 2.3 (changelog) → Step 4 (advance state). This ordering is correct. No change needed — just confirm.

4. Source tagging consistency in MEM-02 blocks — current `build-wave.md` uses `"worker:builder"` as source. `build-verify.md` chaos block uses `"worker:chaos"`. `build-verify.md` watcher block uses `"worker:watcher"`. These are correct and distinct. The new success calls should use the same source tags.

## Architectural Patterns to Follow

### Pattern 1: Fail-Safe Execution (All Integration Points)

Every call to `aether-utils.sh` in playbook steps uses `2>/dev/null || true`. This is the established pattern. Integration fixes must follow it:

```bash
bash .aether/aether-utils.sh memory-capture \
  "success" \
  "Resilience strong: {finding summary}" \
  "pattern" \
  "worker:chaos" 2>/dev/null || true
```

The `|| true` ensures no integration step blocks the primary flow. Pheromone emission, memory capture, and instinct creation are all advisory — they cannot fail the build or continue.

### Pattern 2: Silent When Empty (GAP-2 Promotion)

Per the existing spec and user decision: if `learning-check-promotion` returns zero proposals, produce no output. The promotion UI only appears when there is something to review. The `build-complete.md` Step 5.10 already implements this correctly with:

```bash
if [[ "$proposal_count" -gt 0 ]]; then
  bash .aether/aether-utils.sh learning-approve-proposals
fi
```

New additions should follow the same pattern.

### Pattern 3: Capped Emissions Per Run (GAP-1 and GAP-3)

Decision pheromones (Step 2.1b): cap at 3 per continue run via `emit_count` counter.
Error pattern pheromones (Step 2.1c): cap at 3 per continue run via `emit_count` counter.
Success patterns (new): cap at 2 per build run (matches `instinct-create` success pattern limit in Step 3b).

These caps prevent pheromone inflation. New additions must respect them.

### Pattern 4: Deduplication Before Emission (GAP-1 and GAP-3)

Both Step 2.1b and 2.1c check `pheromones.json` for existing active signals with matching source before emitting. This prevents duplicate signals across multiple continue runs. New calls in build-complete.md Step 5.9 should also check for existing signals before emitting if they are repeatable.

However, `memory-capture` handles its own deduplication internally via content hash in `learning-observations.json`. Calling `memory-capture` multiple times with the same content increments the count rather than creating duplicates. So new `memory-capture "success"` calls do not need external deduplication checks.

## Anti-Patterns to Avoid

### Anti-Pattern 1: Blocking Integration Calls

**What:** Calling `memory-capture` without `2>/dev/null || true`
**Why it is wrong:** If the subcommand fails (e.g., COLONY_STATE.json missing, jq error), it blocks the entire build or continue flow.
**Do this instead:** Always append `2>/dev/null || true` to all integration calls.

### Anti-Pattern 2: Duplicating Logic That Already Exists

**What:** Adding a second midden→pheromone emission path inside the build flow (e.g., after each builder failure in Step 5.2)
**Why it is wrong:** The midden→pheromone conversion already happens in `continue-advance.md` Step 2.1c. Adding it to build creates duplicate REDIRECT signals and runs the same conversion multiple times per phase.
**Do this instead:** `build-wave.md` records to midden. `continue-advance.md` reads midden and emits pheromones. One-directional, one place.

### Anti-Pattern 3: Emitting Pheromones Inside Worker Prompts

**What:** Instructing builder/watcher agents to call `pheromone-write` directly in their task execution
**Why it is wrong:** Workers are stateless sub-agents. They complete tasks and return JSON. The Queen (orchestrator playbook) processes results and makes state decisions. Pheromone emission is a Queen responsibility.
**Do this instead:** Workers write to midden or record in their output JSON. The Queen reads results and emits pheromones.

### Anti-Pattern 4: Skipping the memory-capture Pipeline

**What:** Calling `pheromone-write` directly from playbook steps instead of routing through `memory-capture`
**Why it is wrong:** `memory-capture` does three things atomically: records observation count, emits pheromone, and checks auto-promotion. Bypassing it means the observation count never accumulates, so threshold-based promotion never fires.
**Do this instead:** For learnings and failures, always use `memory-capture`. Only call `pheromone-write` directly for phase-level signals (Step 2.1a) and decision signals (Step 2.1b/2.1c) that are not learning/failure events.

## Build Order for This Milestone

The four gaps have no hard dependencies on each other. They can be built in any order. However, the following ordering is recommended for risk management:

```
Phase 1: Verification Only (read the code, write tests)
  Confirm what exists in each playbook step
  Write integration tests that assert each pipeline fires
  Read midden.json format to verify Step 2.1c jq queries
  --> Zero code changes. Pure reading and test writing.

Phase 2: Gap-4 Success Capture (lowest risk, additive only)
  Add memory-capture "success" to build-verify.md Step 5.7
  Add memory-capture "success" to build-complete.md Step 5.9
  --> Additive changes. Cannot break existing behavior.

Phase 3: Gap-3 Midden→REDIRECT Verification (existing code)
  Run continue with a populated midden to verify Step 2.1c fires
  Fix jq query if midden format mismatch found
  --> Fixes existing code. Low risk.

Phase 4: Gap-1 Decision→Pheromone Verification (existing code)
  Verify Step 2.1b fires and CONTEXT.md awk extraction is robust
  Test with CONTEXT.md that has Recent Decisions table
  --> Fixes existing code. Low risk.

Phase 5: Gap-2 Promotion Pipeline Ordering (existing code)
  Confirm step ordering in continue-advance.md
  If reordering needed, move steps (not add new ones)
  --> Structural change. Moderate risk, requires careful testing.
```

**Critical path:** Phase 1 → Phase 5. Phases 2-4 can be parallelized after Phase 1 confirms what exists.

**Lowest risk first:** Phases 2 (additive) before 3-4 (fixes) before 5 (structural).

## Integration with Existing Test Architecture

The project uses Ava (Node.js) for unit tests and bash scripts for integration tests. Based on the git log (recent commits reference Ava unit tests and bash integration tests), the pattern is:

- `tests/unit/` — Ava tests for subcommand logic
- `tests/integration/` — Bash scripts that run full playbook flows

Integration fixes should be tested at the integration level (bash scripts that run a mini-continue or mini-build and assert that `pheromones.json`, `learning-observations.json`, and `COLONY_STATE.json` contain the expected entries).

Unit tests are appropriate for any new helper functions but the gaps are in playbook steps, not subcommand logic — so integration tests are the primary verification mechanism.

## Sources

All findings are HIGH confidence — sourced directly from the codebase:

- `/Users/callumcowie/repos/Aether/.aether/docs/command-playbooks/continue-advance.md` — Steps 2, 2.1a-2.1e, 2.1.5, 2.1.6, 3, 3a, 3b
- `/Users/callumcowie/repos/Aether/.aether/docs/command-playbooks/continue-gates.md` — Steps 1.6-1.12
- `/Users/callumcowie/repos/Aether/.aether/docs/command-playbooks/continue-verify.md` — Steps 1, 1.5, 1.5.1
- `/Users/callumcowie/repos/Aether/.aether/docs/command-playbooks/continue-finalize.md` — Steps 2.1.6, 2.2-2.7
- `/Users/callumcowie/repos/Aether/.aether/docs/command-playbooks/build-wave.md` — Steps 5-5.3
- `/Users/callumcowie/repos/Aether/.aether/docs/command-playbooks/build-verify.md` — Steps 5.4-5.8
- `/Users/callumcowie/repos/Aether/.aether/docs/command-playbooks/build-complete.md` — Steps 5.9, 5.10, 6, 7
- `/Users/callumcowie/repos/Aether/.aether/docs/command-playbooks/build-prep.md` — Steps 0.5-3
- `/Users/callumcowie/repos/Aether/.aether/docs/command-playbooks/build-context.md` — Steps 4-4.2
- `/Users/callumcowie/repos/Aether/.aether/aether-utils.sh` — `memory-capture` (line 5402), `midden-write` (line 8211), `midden-recent-failures` (line 9581), `instinct-create` (line 7252), `pheromone-write` (line 6774), `context-update` (line 2763)

---
*Architecture research for: Colony Integration Gap Fixes (PHER-01, PHER-02, learnings→instincts, memory-capture consistency)*
*Researched: 2026-03-14*
