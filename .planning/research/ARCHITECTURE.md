# Architecture Research: v4.4 Colony System Hardening

**Domain:** Multi-agent colony system hardening -- recursive delegation, two-tier memory, adaptive scaling, lifecycle agents
**Researched:** 2026-02-04
**Confidence:** HIGH (based on direct codebase analysis + verified external research)

## Executive Summary

Aether v4.4 requires four architectural changes that interact with each other in non-obvious ways. This document maps those interactions, defines component boundaries, specifies data flows, and recommends build order.

**The central tension:** Claude Code's Task tool does NOT support nested spawning (GitHub issue #4182, closed as duplicate, still unresolved as of Feb 2026). Subagents cannot use the Task tool to spawn further subagents. Aether's current architecture already works around this -- the Queen (build.md) is the sole spawner. But the v4.4 goal of "recursive spawning where ants spawn sub-ants" must be designed within this constraint. The recommended approach is **Queen-mediated tree delegation** -- the Queen remains the sole entity with Task tool access, but executes a spawn tree planned by the Phase Lead.

**The second tension:** Two-tier memory (project-local + global) requires a new file location outside the project (`~/.aether/`), new aether-utils.sh subcommands, and a promotion mechanism. This is architecturally independent from recursive spawning but shares the same state files, creating ordering dependencies.

**The third tension:** Adaptive complexity (lightweight vs full mode) must be a decision made early in the colony lifecycle (at `/ant:init` or `/ant:colonize`) and must propagate through all commands without duplicating every command file.

## Current Architecture (As-Is)

```
CURRENT EXECUTION MODEL (v4.3)
===============================

User                Queen (build.md)           Phase Lead (Task)        Workers (Task)
  |                      |                          |                       |
  |-- /ant:build 1 ----->|                          |                       |
  |                      |-- Step 5a: spawn ------->|                       |
  |                      |                          |-- returns plan ------->|
  |                      |<-- plan returned ---------|                       |
  |                      |                                                   |
  |                      |-- Step 5c: for each worker in plan:              |
  |                      |   |-- spawn worker 1 ---------------------------->|
  |                      |   |<-- result 1 ----------------------------------|
  |                      |   |-- spawn worker 2 ---------------------------->|
  |                      |   |<-- result 2 ----------------------------------|
  |                      |                                                   |
  |                      |-- Step 5.5: spawn watcher ----------------------->|
  |                      |<-- watcher report --------------------------------|
  |                      |                                                   |
  |<-- display results --|                                                   |

KEY: Queen is the SOLE entity with Task tool access.
     All "spawning" is Queen using Task tool sequentially per wave.
     Workers have Read/Write/Bash/Glob/Grep/WebSearch/WebFetch but NOT Task.
```

### Current State Files

| File | Schema | Size Limits | Purpose |
|------|--------|-------------|---------|
| `COLONY_STATE.json` | `{goal, state, current_phase, session_id, initialized_at, workers:{}, spawn_outcomes:{}}` | N/A | Colony lifecycle |
| `pheromones.json` | `{signals:[{id, type, content, strength, half_life_seconds, created_at, source?, auto?}]}` | Cleaned when strength < 0.05 | Communication |
| `PROJECT_PLAN.json` | `{goal, generated_at, phases:[{id, name, description, status, tasks:[], success_criteria:[]}]}` | 3-6 phases, 3-8 tasks each | Work structure |
| `errors.json` | `{errors:[{id, category, severity, description, ...}], flagged_patterns:[]}` | 50 errors max | Error tracking |
| `memory.json` | `{phase_learnings:[], decisions:[], patterns:[]}` | 20 learnings, 30 decisions | Learning |
| `events.json` | `{events:[{id, type, source, content, timestamp}]}` | 100 events max | Activity log |

### Current Utility Layer

| Component | File | Functions |
|-----------|------|-----------|
| Main utils | `aether-utils.sh` | 16 subcommands: pheromone-*, validate-state, spawn-check, memory-compress, error-*, activity-log-* |
| File locking | `utils/file-lock.sh` | acquire_lock, release_lock, is_locked, wait_for_lock |
| Atomic writes | `utils/atomic-write.sh` | atomic_write, atomic_write_from_file, create_backup, restore_backup |

### Current Worker Specs

Six specs in `.aether/workers/`, each ~200-350 lines:
- colonizer-ant.md, route-setter-ant.md, builder-ant.md
- watcher-ant.md, scout-ant.md, architect-ant.md

Each contains: Purpose, Visual Identity, Pheromone Sensitivity, Pheromone Math, Combination Effects, Feedback Interpretation, Event Awareness, Memory Reading, Workflow, Output Format, Activity Log, Post-Action Validation, Spawn Gate, Spawn Confidence Check.

---

## Target Architecture (To-Be)

```
V4.4 TARGET EXECUTION MODEL
=============================

User              Queen (build.md)        Phase Lead (Task)     Workers (Task)
  |                    |                       |                     |
  |-- /ant:build 1 -->|                       |                     |
  |                    |                       |                     |
  |                    |-- spawn Phase Lead -->|                     |
  |                    |                       |-- returns spawn    |
  |                    |<-- SPAWN TREE --------|   tree plan        |
  |                    |                                             |
  |                    |  TREE EXECUTION (Queen mediates):           |
  |                    |                                             |
  |                    |  Wave 1:                                    |
  |                    |  |- spawn worker-A ----------------------->| (depth 1)
  |                    |  |<- result-A ----------------------------|
  |                    |  |- spawn worker-B ----------------------->| (depth 1)
  |                    |  |<- result-B ----------------------------|
  |                    |                                             |
  |                    |  Wave 2 (may include sub-delegations):     |
  |                    |  |- spawn worker-C with A's context ------>| (depth 1)
  |                    |  |   worker-C's result says "need scout"   |
  |                    |  |<- result-C + spawn-request ------------|
  |                    |                                             |
  |                    |  Sub-delegation (Queen fulfills request):   |
  |                    |  |- spawn scout for C's request ---------->| (depth 2)
  |                    |  |<- scout result ------------------------|
  |                    |  |- re-spawn worker-C with scout context ->| (depth 2)
  |                    |  |<- final result-C ----------------------|
  |                    |                                             |
  |                    |  Watcher + Learning + Display               |
  |<-- results --------|                                             |

KEY: Queen still sole Task tool holder. "Recursive spawning" is
     Queen executing a spawn tree, fulfilling sub-delegation requests.
     Depth tracking is logical (in COLONY_STATE), not runtime nesting.
```

---

## Component Boundaries

### 1. Spawn Tree Engine (NEW -- modifies build.md)

**Responsibility:** Replace flat wave execution with tree-structured delegation where workers can request sub-spawns that the Queen fulfills.

**Communicates With:**
- Phase Lead (produces the initial spawn tree)
- COLONY_STATE.json (tracks depth, active workers)
- Worker results (parses spawn-request signals from worker output)

**Key Design Decision:** Workers signal "I need a sub-spawn" by including a structured block in their output, NOT by calling the Task tool themselves. The Queen parses this and fulfills it.

```
Worker output format when requesting sub-spawn:
  --- SPAWN REQUEST ---
  caste: scout
  reason: Need to research auth library API before implementing
  context: <what the scout needs to know>
  --- END SPAWN REQUEST ---

Queen parses this, runs spawn-check with incremented depth,
spawns the scout, then either:
  (a) Re-spawns the original worker with scout's findings, OR
  (b) Passes scout findings to the next worker in the wave
```

**State Changes:**
- `COLONY_STATE.json`: Add `spawn_tree` field tracking parent-child relationships and depth
- `spawn-check` subcommand: Already supports depth parameter -- no change needed
- `build.md`: Major rewrite of Step 5c to support tree execution

**Depth Limits:**
- Max depth 3 (already enforced by spawn-check)
- Max 5 active workers colony-wide (already enforced)
- Max 2 sub-spawn requests per worker (NEW limit, prevents runaway chains)

### 2. Two-Tier Learning System (NEW -- modifies memory.json, adds global store)

**Responsibility:** Maintain per-project learnings in `.aether/data/memory.json` (existing) AND global learnings in `~/.aether/learnings.json` (NEW). Promote patterns that recur across projects.

**Communicates With:**
- `memory.json` (project-local, existing)
- `~/.aether/learnings.json` (global, NEW)
- `aether-utils.sh` (new subcommands)
- `build.md` Step 7a (learning extraction)
- `continue.md` Step 4 (learning extraction)
- `init.md` Step 2 (load global learnings at project start)

**Data Flow -- Promotion:**
```
Project A completes phase:
  memory.json.phase_learnings += [{phase: 3, learnings: ["bcrypt 12 rounds too slow"]}]
    |
    v
At /ant:continue or build Step 7a:
  Check if learning pattern matches global patterns:
    - Same category? (e.g., "performance", "security")
    - Similar keywords? (substring match in bash)
    |
    v
  If learning is NEW and not in global: add to global with count=1
  If learning MATCHES existing global: increment count
  If count >= 3 across different projects: mark as "promoted" (high confidence)

Project B initializes:
  /ant:init reads ~/.aether/learnings.json
  Injects promoted learnings as FEEDBACK pheromones:
    "Global learning: bcrypt 12 rounds causes 800ms+ delay, use 10"
```

**File Locations:**
- Project-local: `.aether/data/memory.json` (unchanged)
- Global: `~/.aether/learnings.json` (NEW)
- Global index: `~/.aether/projects.json` (NEW -- tracks which projects contributed)

**Schema -- `~/.aether/learnings.json`:**
```json
{
  "version": 1,
  "learnings": [
    {
      "id": "global_<timestamp>_<hex>",
      "content": "bcrypt 12 rounds causes 800ms+ delay on auth endpoints",
      "category": "performance",
      "source_projects": ["project-a-session-123", "project-c-session-456"],
      "occurrence_count": 3,
      "promoted": true,
      "first_seen": "2026-02-01T...",
      "last_seen": "2026-02-04T..."
    }
  ]
}
```

**New aether-utils.sh Subcommands:**
- `learning-promote <content> <category>`: Add or increment a global learning
- `learning-load`: Read global learnings, return promoted ones as JSON
- `learning-init-global`: Create `~/.aether/` and `learnings.json` if missing

### 3. Adaptive Complexity System (NEW -- modifies init.md, adds mode to COLONY_STATE)

**Responsibility:** Detect project complexity at colonization time and set a mode that reduces ceremony for simple projects.

**Communicates With:**
- `COLONY_STATE.json` (stores mode)
- `init.md` / `colonize.md` (sets mode)
- All command files (read mode to adjust behavior)

**Complexity Detection Heuristics (at /ant:colonize or /ant:init):**
```
LIGHTWEIGHT mode if ALL of:
  - Single language detected
  - < 20 files total
  - No existing test suite
  - No CI/CD config
  - Goal description < 50 words

FULL mode if ANY of:
  - Multiple languages
  - > 50 files
  - Existing test suite
  - CI/CD configuration
  - Complex goal (> 50 words or mentions "authentication", "database", "API", etc.)

STANDARD mode: everything else (default)
```

**What Changes Per Mode:**

| Aspect | LIGHTWEIGHT | STANDARD | FULL |
|--------|------------|----------|------|
| Phase count | 2-3 | 3-6 | 4-8 |
| Workers per wave | 1-2 | 2-4 | 3-5 |
| Watcher verification | Syntax check only | Full verification | Full + execution test |
| Learning extraction | End-of-project only | Per-phase | Per-phase + inter-phase |
| Pheromone auto-emit | FEEDBACK only | FEEDBACK + REDIRECT | All types |
| Activity log | Disabled | Enabled | Enabled + detailed |
| spawn-check limits | depth 2, workers 3 | depth 3, workers 5 | depth 3, workers 5 |

**State Change -- `COLONY_STATE.json`:**
```json
{
  "goal": "...",
  "mode": "STANDARD",
  ...
}
```

**Implementation Strategy:** Rather than duplicating command files, each command reads `COLONY_STATE.json` mode at the start and adjusts behavior inline. The mode is a single field, not a separate config system.

### 4. Auto-Spawned Lifecycle Ants (NEW -- modifies build.md, adds new worker specs)

**Responsibility:** Automatically trigger reviewer and debugger ants at appropriate stages, plus an organizer/archivist ant for hygiene.

**Two New Worker Specs:**
- `.aether/workers/reviewer-ant.md` (NEW) -- Code review specialist, spawned automatically after builder completes
- `.aether/workers/archivist-ant.md` (NEW) -- Reviews for stale files, dead code, orphaned configs

**Integration Points:**

```
build.md Step 5c (after each worker completes):
  IF worker was a builder-ant AND mode != LIGHTWEIGHT:
    Auto-spawn reviewer-ant with builder's output as context
    Reviewer checks: code style, potential bugs, missed edge cases
    Reviewer output appended to worker_results (not a separate wave)

build.md Step 5.5 (watcher verification):
  Watcher already exists -- reviewer is PRE-watcher, watcher is POST-phase

/ant:continue Step 4 (after phase completes):
  IF mode == FULL AND phase_number is divisible by 3:
    Auto-spawn archivist-ant to check for:
      - Stale files created in early phases but no longer referenced
      - Dead code from abandoned approaches
      - Orphaned config entries
      - Oversized state files that need compression
    Archivist output: list of cleanup recommendations
    User decides whether to act on them
```

**Reviewer Ant is NOT a new caste.** It uses the watcher-ant spec with a modified prompt ("review mode" vs "verification mode"). This avoids adding a 7th caste and keeps the sensitivity table unchanged.

**Archivist Ant IS a new concept** but reuses the architect-ant spec with an archivist-specific task prompt. The architect-ant already handles "synthesize knowledge, extract patterns" -- archiving is a specialization of this.

### 5. Same-File Conflict Prevention (NEW -- modifies build.md Step 5c)

**Responsibility:** When multiple workers in the same wave would modify the same file, route those tasks to the same worker.

**Detection Point:** Phase Lead's spawn tree plan (Step 5a). The Phase Lead already assigns tasks to workers. Add a constraint: "If two tasks reference the same files, assign them to the same worker."

**Enforcement Point:** build.md Step 5c. Before spawning workers in a wave, check for file overlaps. If detected, merge overlapping tasks into one worker's assignment.

**Implementation:**
```
Step 5b-post (after plan approval, before execution):
  Parse plan for file references in task descriptions
  Build file -> worker mapping
  If file appears in multiple workers' tasks:
    Merge those tasks into the worker with the most file references
    Log: "Merged tasks {ids} into single worker to prevent {file} conflicts"
```

This is lightweight -- it's a text analysis step in the Queen's logic, not a new utility or state file.

### 6. Pheromone-First Flow (REORDER -- modifies colonize.md, plan.md)

**Responsibility:** Change the recommended flow from `colonize -> plan` to `colonize -> pheromone injection -> plan`.

**This is not an architecture change.** It's a UX flow change:
- `colonize.md` Step 6 (display results): Change "Next" suggestions to emphasize `/ant:focus` and `/ant:redirect` before `/ant:plan`
- `plan.md` Step 4 (spawn planner): Add a check -- if no FOCUS or REDIRECT pheromones exist and colony has been colonized, display a suggestion to add pheromones first

No state file changes. No new components. Just prompt text changes.

---

## Data Flow Diagrams

### Recursive Spawning State Flow

```
/ant:build 3
    |
    v
Queen reads COLONY_STATE.json
  spawn_tree: null (no active tree)
    |
    v
Queen spawns Phase Lead
  Phase Lead returns SPAWN TREE:
    Wave 1:
      worker-A (builder): tasks 3.1, 3.2 [files: src/auth.ts, src/routes.ts]
      worker-B (builder): tasks 3.3 [files: src/db.ts]
    Wave 2:
      worker-C (builder): task 3.4 [depends on A, files: src/auth.ts]
        |
        v
Queen detects: worker-A and worker-C both touch src/auth.ts
  -> Merge task 3.4 into worker-A's assignment for Wave 2 re-run
  -> worker-C removed from plan
    |
    v
Queen writes COLONY_STATE.json:
  spawn_tree: {
    root: "phase_lead",
    waves: [
      {id: 1, workers: [{caste: "builder", tasks: ["3.1","3.2"], depth: 1},
                          {caste: "builder", tasks: ["3.3"], depth: 1}]},
      {id: 2, workers: [{caste: "builder", tasks: ["3.1","3.2","3.4"], depth: 1}]}
    ],
    active_depth: 0,
    max_depth: 3,
    spawn_requests: []
  }
    |
    v
Queen executes Wave 1:
  spawn worker-A (depth 1) -> result includes:
    --- SPAWN REQUEST ---
    caste: scout
    reason: Need OAuth2 library docs
    --- END SPAWN REQUEST ---
    |
    v
Queen parses spawn request, runs spawn-check depth=2:
  pass: true (depth 2 < 3, active_workers 2 < 5)
    |
    v
Queen spawns scout (depth 2) -> returns OAuth2 docs
    |
    v
Queen re-spawns worker-A with scout findings (depth 2)
  worker-A completes with full context
    |
    v
Queen updates spawn_tree.spawn_requests with completed request
Queen continues to Wave 2, worker-B, etc.
```

### Two-Tier Learning Promotion Flow

```
PROJECT A: /ant:build 3 completes
    |
    v
Step 7a: Extract learnings
  memory.json += {learnings: ["builder-ant: bcrypt 12 rounds caused 800ms delay"]}
    |
    v
Step 7a-post: Promote to global (NEW)
  bash .aether/aether-utils.sh learning-promote \
    "bcrypt 12 rounds causes 800ms+ auth delay, use 10 rounds" \
    "performance"
    |
    v
~/.aether/learnings.json:
  learnings += {content: "bcrypt 12...", count: 1, promoted: false, source: ["project-a"]}

---

PROJECT B: /ant:build 2 completes (weeks later)
  Same learning extracted -> count becomes 2

---

PROJECT C: /ant:build 1 completes
  Same learning extracted -> count becomes 3 -> promoted: true

---

PROJECT D: /ant:init "Build payment API"
    |
    v
Step 2.5 (NEW): Load global learnings
  bash .aether/aether-utils.sh learning-load
    |
    v
Returns promoted learnings (count >= 3)
    |
    v
Step 5 (init): Emit as FEEDBACK pheromones
  pheromones.json += {
    type: "FEEDBACK",
    content: "Global: bcrypt 12 rounds causes 800ms+ delay, use 10",
    source: "auto:global",
    strength: 0.3
  }
```

### Adaptive Complexity Detection Flow

```
/ant:colonize (or /ant:init with --detect-complexity)
    |
    v
Colonizer explores codebase:
  - Count files: ls -R | wc -l
  - Detect languages: check extensions
  - Check for tests: test/, __tests__, *.test.*, pytest.ini
  - Check for CI: .github/workflows, .gitlab-ci.yml, Jenkinsfile
    |
    v
Colonizer returns findings with complexity signals
    |
    v
Queen evaluates heuristics:
  files < 20 AND single_language AND no_tests AND no_ci
    -> mode = "LIGHTWEIGHT"
  files > 50 OR multi_language OR has_tests OR has_ci
    -> mode = "FULL"
  else
    -> mode = "STANDARD"
    |
    v
COLONY_STATE.json: mode = "STANDARD"
    |
    v
All subsequent commands check mode:
  if mode == "LIGHTWEIGHT": skip activity-log, reduce spawn limits
  if mode == "FULL": add reviewer auto-spawn, enable archivist
```

---

## State File Changes Summary

### Modified Files

**`COLONY_STATE.json` -- 3 new fields:**
```json
{
  "goal": "...",
  "state": "READY",
  "mode": "STANDARD",          // NEW: LIGHTWEIGHT | STANDARD | FULL
  "current_phase": 0,
  "session_id": "...",
  "initialized_at": "...",
  "workers": { ... },
  "spawn_outcomes": { ... },
  "spawn_tree": null            // NEW: populated during build, null when idle
}
```

**`memory.json` -- no schema change, but new promotion behavior:**
- phase_learnings array is unchanged
- After learning extraction, new promotion step writes to global

### New Files

**`~/.aether/learnings.json` (global, outside project):**
```json
{
  "version": 1,
  "learnings": [
    {
      "id": "global_<ts>_<hex>",
      "content": "<learning text>",
      "category": "<performance|security|architecture|convention|tooling>",
      "source_projects": ["session_id_1", "session_id_2"],
      "occurrence_count": 3,
      "promoted": true,
      "first_seen": "<ISO-8601>",
      "last_seen": "<ISO-8601>"
    }
  ]
}
```

**`~/.aether/projects.json` (global index, outside project):**
```json
{
  "version": 1,
  "projects": [
    {
      "session_id": "session_123_abc",
      "goal": "Build REST API",
      "mode": "STANDARD",
      "phases_completed": 5,
      "learnings_promoted": 3,
      "last_active": "<ISO-8601>"
    }
  ]
}
```

No new worker spec files are needed. Reviewer and archivist behaviors are prompt variations of existing watcher-ant.md and architect-ant.md respectively.

### New aether-utils.sh Subcommands

| Subcommand | Args | Returns | Purpose |
|------------|------|---------|---------|
| `learning-promote` | `<content> <category>` | `{promoted: bool, count: N}` | Add/increment global learning |
| `learning-load` | (none) | `{learnings: [...promoted only...]}` | Load promoted global learnings |
| `learning-init-global` | (none) | `{created: bool}` | Create ~/.aether/ if missing |
| `complexity-detect` | (none) | `{mode: "STANDARD", signals: {...}}` | Detect project complexity heuristics |

---

## Build Order (Critical for Phase Ordering)

The four features have the following dependency graph:

```
                    +-------------------+
                    | Adaptive          |
                    | Complexity (3)    |
                    +--------+----------+
                             |
                    depends on (mode field must exist)
                             |
         +-------------------+-------------------+
         |                                       |
+--------v----------+              +-------------v--------+
| Recursive         |              | Lifecycle Ants (4)   |
| Spawning (1)      |              | (reviewer, archivist)|
+--------+----------+              +-------------+--------+
         |                                       |
         | spawn_tree in COLONY_STATE            | mode determines
         | must exist for sub-spawns             | when to auto-spawn
         |                                       |
+--------v----------+              +-------------v--------+
| Same-File         |              | Two-Tier             |
| Conflict (1b)     |              | Learning (2)         |
+---------+---------+              +----------+-----------+
          |                                   |
          +------- both independent ----------+
          |                                   |
+---------v-----------------------------------v----------+
| Pheromone-First Flow (0) -- prerequisite prompt fix    |
+--------------------------------------------------------+
```

### Recommended Build Order

**Phase 0: Pheromone-First Flow + Mode Field Foundation**
- Modify colonize.md to suggest pheromone injection before planning
- Modify plan.md to check for pheromones before planning
- Add `mode` field to COLONY_STATE.json schema (default: "STANDARD")
- Add `mode` field to init.md output
- **Why first:** Zero-risk prompt text changes that improve the flow immediately. Mode field is needed by everything else.
- **Files:** colonize.md, plan.md, init.md, COLONY_STATE.json schema
- **Risk:** LOW -- text changes only

**Phase 1: Recursive Spawning + Same-File Conflict Prevention**
- Rewrite build.md Step 5c for tree execution
- Add spawn-request parsing to worker output handling
- Add spawn_tree field to COLONY_STATE.json
- Add file overlap detection in plan post-processing
- **Why second:** This is the highest-impact architectural change. All other features benefit from tree delegation.
- **Files:** build.md (major rewrite of Steps 5a-5c), COLONY_STATE.json schema
- **Risk:** HIGH -- changes the core execution loop. Needs thorough testing.
- **Dependency:** Phase 0 (mode field exists)

**Phase 2: Two-Tier Learning System**
- Create `~/.aether/` directory structure
- Add learning-promote, learning-load, learning-init-global to aether-utils.sh
- Modify build.md Step 7a to promote after extraction
- Modify continue.md Step 4 to promote after extraction
- Modify init.md to load global learnings
- **Why third:** Independent of recursive spawning. Can be built and tested separately.
- **Files:** aether-utils.sh (3 new subcommands), build.md Step 7a, continue.md Step 4, init.md Steps 2.5 and 5
- **Risk:** MEDIUM -- new external directory (~/.aether/), new subcommands
- **Dependency:** Phase 0 (mode field for conditional promotion behavior)

**Phase 3: Adaptive Complexity Detection**
- Add complexity-detect subcommand to aether-utils.sh
- Modify colonize.md to run complexity detection and set mode
- Modify all commands to read mode and adjust behavior
- **Why fourth:** Depends on mode field (Phase 0) and benefits from understanding how recursive spawning and learning behave before adding mode-dependent variation.
- **Files:** aether-utils.sh (1 new subcommand), colonize.md, init.md, build.md, continue.md, status.md
- **Risk:** MEDIUM -- touches many files but changes are conditional checks, not structural

**Phase 4: Auto-Spawned Lifecycle Ants**
- Add reviewer auto-spawn after builder workers in build.md Step 5c
- Add archivist auto-spawn at /ant:continue for FULL mode
- Write reviewer and archivist task prompt templates (reusing existing specs)
- **Why last:** Depends on recursive spawning (Phase 1) for sub-delegation, mode (Phase 3) for conditional triggering, and learning (Phase 2) for archivist to check stale patterns.
- **Files:** build.md Step 5c, continue.md Step 4, potentially new prompt template files
- **Risk:** LOW -- additive changes using existing spawn infrastructure
- **Dependency:** Phases 1, 2, 3

---

## Patterns to Follow

### Pattern 1: Queen-Mediated Tree Delegation

**What:** Workers signal spawn needs through structured output. Queen fulfills them.

**Why:** Claude Code Task tool does not support nested spawning. This is a verified platform limitation (GitHub #4182). Any architecture that assumes workers can use Task tool will fail.

**How:**
```
Worker includes in output:
  --- SPAWN REQUEST ---
  caste: scout
  reason: <why needed>
  context: <what the spawned ant needs>
  blocking: true|false
  --- END SPAWN REQUEST ---

Queen's build.md parses this block after worker returns.
If blocking=true: Queen spawns sub-ant, then re-runs original worker with results.
If blocking=false: Queen spawns sub-ant, passes results to next wave.
```

**Confidence:** HIGH -- this pattern is consistent with how the system already works (Queen mediates all spawning). It extends rather than replaces the existing model.

### Pattern 2: Filesystem-as-Shared-Memory

**What:** Agents coordinate through files, not direct communication.

**Why:** Task tool subagents have separate context windows. They cannot share context directly. But they share the filesystem. This is the official recommended pattern per Claude Code documentation.

**How:** Already implemented in Aether. State files in `.aether/data/` serve this role. The new `~/.aether/learnings.json` extends this pattern to cross-project scope.

**Confidence:** HIGH -- this is how Aether already works and is verified as the Claude Code best practice.

### Pattern 3: Conditional Behavior via Mode Flag

**What:** Single `mode` field in COLONY_STATE.json, read by all commands, adjusts behavior inline.

**Why:** Avoids duplicating command files (no lightweight-build.md vs full-build.md). Keeps the system DRY. Mode is set once and propagates.

**How:**
```
In each command, after reading COLONY_STATE.json:
  mode = colony_state.mode || "STANDARD"

  if mode == "LIGHTWEIGHT":
    skip activity-log calls
    reduce spawn limits
    skip reviewer auto-spawn
  elif mode == "FULL":
    enable reviewer auto-spawn
    enable archivist at milestones
    detailed activity logging
```

**Confidence:** HIGH -- this is standard feature-flag architecture applied to prompt-based commands.

### Pattern 4: Promotion via Occurrence Counting

**What:** Learnings promote from local to global based on cross-project recurrence, not single-project importance.

**Why:** A learning that appears in one project might be project-specific. One that appears in 3+ projects is likely generalizable. Simple occurrence counting with substring matching (bash+jq) avoids complexity.

**How:**
```bash
# In aether-utils.sh learning-promote:
# Check if content matches existing global learning (case-insensitive substring)
existing=$(jq --arg c "$content" '.learnings[] | select(.content | ascii_downcase | contains($c | ascii_downcase))' ~/.aether/learnings.json)

if [ -n "$existing" ]; then
  # Increment count, update last_seen, add source_project
  # If count >= 3: set promoted=true
else
  # Add new entry with count=1
fi
```

**Confidence:** MEDIUM -- substring matching is crude. May need refinement (e.g., keyword extraction). But it's simple, testable, and can be improved later without architectural change.

---

## Anti-Patterns to Avoid

### Anti-Pattern 1: Attempting Nested Task Tool Calls

**What:** Designing worker specs to use the Task tool for sub-spawning.

**Why bad:** Task tool is not available to subagents. This is a verified Claude Code platform constraint. Workers that try to call Task will fail silently or error.

**Instead:** Use the spawn-request output pattern described above. Queen mediates all spawning.

### Anti-Pattern 2: Complex Mode Configuration Files

**What:** Creating separate config files, YAML schemas, or multi-field configuration objects for adaptive complexity.

**Why bad:** Adds files to manage, schemas to validate, and increases cognitive load. The system is already JSON-heavy.

**Instead:** Single `mode` string field in COLONY_STATE.json. Three values: LIGHTWEIGHT, STANDARD, FULL. Inline conditional checks in commands.

### Anti-Pattern 3: Global Memory as Write-Heavy Store

**What:** Writing every learning to `~/.aether/learnings.json` immediately on extraction.

**Why bad:** Pollutes global store with project-specific noise. Makes the file grow unboundedly. Creates lock contention if multiple projects run concurrently (unlikely but possible).

**Instead:** Only promote learnings that pass a relevance filter. Cap global learnings at 100 entries. Rotate oldest non-promoted entries when full.

### Anti-Pattern 4: Reviewer Ant as Separate Caste

**What:** Creating a 7th worker caste with its own sensitivity profile, spec file, and spawn-check integration.

**Why bad:** Expands the sensitivity table (all commands reference 6 castes), requires updating every worker spec's "Available castes" section, and the reviewer's behavior is a subset of the watcher's.

**Instead:** Reviewer is a watcher-ant spawned with a "review mode" task prompt. Same spec, different instructions. Archivist is an architect-ant spawned with a "hygiene check" task prompt.

### Anti-Pattern 5: Blocking Spawn Requests

**What:** Worker hangs waiting for a sub-spawn to be fulfilled before continuing.

**Why bad:** Workers cannot actually wait -- they execute and return. There is no suspend/resume in the Task tool model.

**Instead:** Spawn requests are fulfilled between worker completions. If a worker needs information from a sub-spawn, the Queen fulfills the sub-spawn first, then re-runs the worker with the results. The worker sees this as a single execution with richer context, not as a wait.

---

## Integration Points with Existing System

| New Component | Integrates With | How |
|---------------|----------------|-----|
| Spawn Tree | build.md Step 5a-5c | Phase Lead outputs tree; Queen executes it |
| Spawn Tree | spawn-check subcommand | Already supports depth -- no change |
| Spawn Tree | COLONY_STATE.json | New spawn_tree field |
| Two-Tier Learning | aether-utils.sh | 3 new subcommands |
| Two-Tier Learning | build.md Step 7a | Promotion call after extraction |
| Two-Tier Learning | init.md | Load global learnings, emit as pheromones |
| Adaptive Complexity | COLONY_STATE.json | New mode field |
| Adaptive Complexity | colonize.md | Complexity detection |
| Adaptive Complexity | All commands | Conditional behavior checks |
| Lifecycle Ants | build.md Step 5c | Auto-spawn reviewer after builder |
| Lifecycle Ants | continue.md Step 4 | Auto-spawn archivist at milestones |
| Lifecycle Ants | watcher-ant.md | Reviewer uses watcher spec |
| Lifecycle Ants | architect-ant.md | Archivist uses architect spec |
| Conflict Prevention | build.md Step 5b-post | File overlap detection |
| Pheromone-First | colonize.md, plan.md | Text changes to "Next" suggestions |

---

## Scalability Considerations

| Concern | Current (v4.3) | After v4.4 |
|---------|----------------|------------|
| Spawn depth | Max 3, enforced | Max 3, tree-tracked in state |
| Active workers | Max 5, enforced | Max 5, spawn_tree provides visibility |
| Global learnings | N/A | 100 entry cap with rotation |
| Mode overhead | N/A | Single field check per command, negligible |
| build.md complexity | ~660 lines | ~800-900 lines (tree execution adds ~150) |
| State file size | 6 files, ~5KB total | 6 files + mode + spawn_tree, ~8KB total |
| Global files | N/A | 2 files in ~/.aether/, ~5KB each |

---

## Sources

### HIGH Confidence (Direct Codebase Analysis)
- `/Users/callumcowie/repos/Aether/.claude/commands/ant/build.md` -- 663 lines, full execution flow analyzed
- `/Users/callumcowie/repos/Aether/.claude/commands/ant/init.md` -- 199 lines, initialization flow analyzed
- `/Users/callumcowie/repos/Aether/.claude/commands/ant/colonize.md` -- 170 lines, colonization flow analyzed
- `/Users/callumcowie/repos/Aether/.claude/commands/ant/continue.md` -- 319 lines, continuation flow analyzed
- `/Users/callumcowie/repos/Aether/.claude/commands/ant/plan.md` -- 194 lines, planning flow analyzed
- `/Users/callumcowie/repos/Aether/.aether/aether-utils.sh` -- 265 lines, all 16 subcommands analyzed
- `/Users/callumcowie/repos/Aether/.aether/workers/builder-ant.md` -- 354 lines, spawn gate and output format analyzed
- `/Users/callumcowie/repos/Aether/.aether/data/COLONY_STATE.json` -- current schema verified
- `/Users/callumcowie/repos/Aether/.aether/HANDOFF.md` -- v4.3 session outcomes and known issues

### HIGH Confidence (Verified Platform Constraints)
- [Claude Code nested spawning limitation -- GitHub #4182](https://github.com/anthropics/claude-code/issues/4182) -- Task tool not available to subagents, closed as duplicate, still unresolved
- [Claude Code official subagent docs](https://code.claude.com/docs/en/sub-agents) -- confirms flat delegation model
- [Anthropic multi-agent research system](https://www.anthropic.com/engineering/multi-agent-research-system) -- orchestrator-worker pattern, 4x token overhead

### MEDIUM Confidence (Verified Research Patterns)
- [G-Memory: Hierarchical Memory for Multi-Agent Systems](https://arxiv.org/abs/2506.07398) -- three-tier graph hierarchy for MAS memory, NeurIPS 2025
- [Towards a Science of Scaling Agent Systems](https://arxiv.org/abs/2512.08296) -- adaptive complexity, diminishing returns from multi-agent overhead
- [Google ADK Multi-Agent Patterns](https://developers.googleblog.com/developers-guide-to-multi-agent-patterns-in-adk/) -- hierarchical decomposition, recursive delegation
- [Efficient Agents: Reducing Cost](https://arxiv.org/html/2508.02694v1) -- task-adaptive agent frameworks, lightweight mode wins for simple tasks

### LOW Confidence (Community Patterns)
- [Sub-Agent Spawning Pattern](https://agentic-patterns.com/patterns/sub-agent-spawning/) -- general pattern description
- [Claude Code Swarm Orchestration](https://gist.github.com/kieranklaassen/4f2aba89594a4aea4ad64d753984b2ea) -- TeammateTool pattern, tested with CC v2.1.19
