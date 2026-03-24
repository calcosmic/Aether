# Architecture Research: Production Hardening of Aether CLI

**Domain:** Multi-agent CLI tool (bash + Node.js dual runtime)
**Researched:** 2026-03-23
**Confidence:** HIGH (based on direct codebase analysis + established patterns)

## Current Architecture (As-Is)

```
+-------------------------------------------------------------------+
|                    CONSUMER LAYER                                  |
|  +--------------------+  +--------------------+                    |
|  | 43 Slash Commands  |  | 22 Agent Defns     |  (Markdown files  |
|  | (.claude/commands) |  | (.claude/agents)   |   interpreted by  |
|  +--------+-----------+  +--------+-----------+   Claude Code)    |
|           | bash exec             | bash exec                     |
+-----------|------------------------|-----------+------------------+
|                    MONOLITH (aether-utils.sh)                      |
|  11,272 lines | 178 subcommands | single case dispatch             |
|  +---------+ +----------+ +----------+ +----------+               |
|  | Colony  | | Pheromone| | Learning | | Swarm    | <-- inline     |
|  | State   | | System   | | Engine   | | Ops      |     in the    |
|  +----+----+ +----+-----+ +----+-----+ +----+-----+    monolith   |
|       |           |            |             |                     |
|  sources at startup:                                               |
|  +----------+ +----------+ +----------+ +----------+              |
|  | hive.sh  | |midden.sh | |skills.sh | | xml-*.sh | <-- already  |
|  | (561 ln) | | (260 ln) | | (502 ln) | | (1023 ln)|   extracted  |
|  +----------+ +----------+ +----------+ +----------+              |
|  + file-lock.sh, atomic-write.sh, error-handler.sh (infra)        |
+-------------------------------------------------------------------+
|                    NODE.JS CLI (bin/cli.js)                         |
|  Distribution, hub management, model profiles, telemetry           |
|  16 modules in bin/lib/ (6,578 lines total)                        |
|  Does NOT call aether-utils.sh at runtime                          |
+-------------------------------------------------------------------+
|                    STATE LAYER                                      |
|  +------------------+  +--------------+  +--------------+         |
|  | COLONY_STATE.json|  |pheromones.json|  | midden.json  |         |
|  | (central nexus)  |  |              |  |              |         |
|  | 90 refs in bash  |  |              |  |              |         |
|  | 20+ refs in cmds |  |              |  |              |         |
|  +------------------+  +--------------+  +--------------+         |
|  + constraints.json, flags.json, session.json, spawn-tree.txt     |
|  + ~/.aether/hive/wisdom.json (hub-level)                          |
+-------------------------------------------------------------------+
```

### Key Structural Observations

1. **The monolith has a clear extraction precedent.** Hive (561 lines), midden (260 lines), and skills (502 lines) were already extracted into `.aether/utils/` and wired back via thin dispatch entries (e.g., `hive-init) _hive_init "$@" ;;`). This pattern works and should be the model for further extraction.

2. **Two runtimes, minimal cross-talk.** Node.js (bin/cli.js) handles distribution/hub/install. Bash (aether-utils.sh) handles runtime operations. They share error code constants but Node does NOT shell out to bash at runtime (only model-verify.js checks for the file's existence). This is good -- the boundary is clean.

3. **Consumers invoke bash directly.** The 43 slash commands and playbooks contain 412 references to `bash .aether/aether-utils.sh <subcommand>`. This is the actual API surface. Subcommand names are the contract.

4. **COLONY_STATE.json is the coupling nexus.** 90 references in the monolith, 20+ inline jq reads in commands/playbooks that bypass aether-utils.sh entirely (dual write paths).

5. **Error swallowing is endemic.** 418 instances of `2>/dev/null`, 104 instances of `|| true`, 89 combined `2>/dev/null || true`. Many are legitimate (optional feature checks), but many hide real failures.

## Recommended Architecture (To-Be)

```
+-------------------------------------------------------------------+
|                    CONSUMER LAYER (unchanged)                       |
|  Slash commands + Agents --> bash .aether/aether-utils.sh <cmd>    |
+-------------------------------------------------------------------+
|                    DISPATCHER (aether-utils.sh, slimmed)            |
|  ~1,500 lines: setup, sourcing, case dispatch, shared helpers      |
|  Sources domain modules on demand (not all at startup)             |
+-------------------------------------------------------------------+
|                    DOMAIN MODULES (.aether/utils/)                  |
|  +----------+ +----------+ +----------+ +----------+              |
|  |pheromone | | learning | |  colony  | |  swarm   |              |
|  |  .sh     | |  .sh     | |  .sh     | |  .sh     |              |
|  +----------+ +----------+ +----------+ +----------+              |
|  +----------+ +----------+ +----------+ +----------+              |
|  |  queen   | | session  | | suggest  | |autopilot |              |
|  |  .sh     | |  .sh     | |  .sh     | |  .sh     |              |
|  +----------+ +----------+ +----------+ +----------+              |
|  + existing: hive.sh, midden.sh, skills.sh, xml-*.sh              |
|  + infra: file-lock.sh, atomic-write.sh, error-handler.sh         |
+-------------------------------------------------------------------+
|                    STATE ACCESS LAYER (new)                         |
|  +----------------------------------------------------------+     |
|  | state-api.sh                                              |     |
|  | Facade functions for COLONY_STATE.json reads/writes       |     |
|  | state_get_phase(), state_get_goal(), state_add_event()    |     |
|  | All locking/validation encapsulated                       |     |
|  +----------------------------------------------------------+     |
|  All domain modules and consumers use state-api.sh                 |
|  No direct jq on COLONY_STATE.json outside this file               |
+-------------------------------------------------------------------+
|                    STATE FILES (unchanged paths)                    |
|  .aether/data/COLONY_STATE.json, pheromones.json, etc.             |
+-------------------------------------------------------------------+
```

### Component Responsibilities

| Component | Responsibility | Lines (est.) |
|-----------|----------------|--------------|
| aether-utils.sh (dispatcher) | Setup, source modules, case dispatch, shared helpers (json_ok, get_caste_emoji, generate-ant-name, etc.) | ~1,500 |
| state-api.sh | All COLONY_STATE.json reads/writes, locking, validation, migration | ~400 |
| pheromone.sh | pheromone-write/read/count/display/prime/expire/export, colony-prime, instinct-* | ~1,800 |
| learning.sh | learning-observe/promote/inject/check-promotion/approve/select/display/defer/undo | ~1,200 |
| queen.sh | queen-init/read/promote/thresholds, incident-rule-add | ~500 |
| colony.sh | validate-state, load-state, unload-state, milestone-detect, memory-capture, context-capsule, rolling-summary | ~600 |
| swarm.sh | All swarm-* subcommands + swarm-display-* | ~800 |
| session.sh | session-init/update/read/is-stale/clear/mark-resumed/summary, session-verify-fresh, session-clear | ~400 |
| spawn.sh | spawn-log/complete/can-spawn/get-depth/tree-*/efficiency, generate-ant-name, validate-worker-response | ~500 |
| flag.sh | flag-add/check-blockers/resolve/acknowledge/list/auto-resolve | ~300 |
| suggest.sh | suggest-analyze/record/check/clear/approve/quick-dismiss | ~500 |
| autopilot.sh | autopilot-init/update/status/stop/check-replan | ~200 |
| changelog.sh | changelog-append/collect-plan-data (already functions, just move) | ~200 |
| semantic.sh | semantic-init/index/search/rebuild/status/context | ~100 |
| misc.sh | error-add/summary, signature-scan/match, check-antipattern, grave-add/check, generate-commit-message, bootstrap-system, etc. | ~600 |

## Modularization Strategy

### Phase 1: State Access Layer (foundation -- everything else depends on this)

**What:** Create `state-api.sh` with facade functions for COLONY_STATE.json.

**Why first:** 90 references in the monolith + 20+ inline jq reads in commands create the dual-write-path risk. Until state access is centralized, extracting other modules risks creating MORE coupling, not less.

**Pattern:**

```bash
#!/bin/bash
# state-api.sh - All COLONY_STATE.json access goes through here

# Read functions (no locking needed for atomic reads)
state_get_goal() {
    local state_file="$DATA_DIR/COLONY_STATE.json"
    [[ -f "$state_file" ]] || { echo ""; return 1; }
    jq -r '.goal // ""' "$state_file"
}

state_get_phase() {
    local state_file="$DATA_DIR/COLONY_STATE.json"
    [[ -f "$state_file" ]] || { echo "0"; return 1; }
    jq -r '.current_phase // 0' "$state_file"
}

state_get_session_id() {
    local state_file="$DATA_DIR/COLONY_STATE.json"
    [[ -f "$state_file" ]] || { echo "unknown"; return 1; }
    jq -r '.session_id // "unknown"' "$state_file"
}

# Write functions (with locking)
state_add_event() {
    local event_type="$1" description="$2" source="${3:-unknown}"
    local state_file="$DATA_DIR/COLONY_STATE.json"
    acquire_lock "$state_file" || return 1
    # ... jq update ...
    release_lock
}

# Composite read (load full state for multi-field reads)
state_load_snapshot() {
    cat "$DATA_DIR/COLONY_STATE.json"
}
```

**Validation approach:** After creating state-api.sh, run the existing 530+ tests. Nothing should break because you are adding new accessor functions, not removing old code. Then incrementally replace inline jq calls in the monolith to use the new functions. Each replacement is testable in isolation.

**Important constraint:** Commands/playbooks that do inline `jq ... .aether/data/COLONY_STATE.json` cannot be migrated in this phase -- they are markdown files interpreted by Claude Code, not bash scripts. Those get a migration guide and are updated as a follow-up task, not a blocker.

### Phase 2: Extract pheromone/instinct system (largest cohesive domain)

**What:** Extract pheromone-write/read/count/display/prime/expire/export, colony-prime, instinct-read/create/apply, and eternal-* into `pheromone.sh`.

**Why second:** This is the largest contiguous block (~1,800 lines, anchored by colony-prime at 662 lines alone). It is cohesive -- these subcommands share helper functions and read from pheromones.json. colony-prime is the most complex single subcommand and benefits most from isolation for testing.

**Extraction pattern (same as hive.sh/midden.sh/skills.sh):**

1. Move function bodies into `_pheromone_write()`, `_pheromone_read()`, etc. in `.aether/utils/pheromone.sh`
2. Source pheromone.sh from aether-utils.sh at startup (same as line 33: `source "$SCRIPT_DIR/utils/hive.sh"`)
3. Replace case entries with thin dispatchers: `pheromone-write) _pheromone_write "$@" ;;`
4. Run tests after each subcommand migration

**Risk:** colony-prime depends on pheromone-prime (calls it via `"$SCRIPT_DIR/aether-utils.sh" pheromone-prime`). After extraction, this self-invocation still works because the dispatch entry remains. No change needed. However, after extraction, the self-invocation can be replaced with a direct function call `_pheromone_prime "$@"` for a ~200ms performance improvement (avoids re-spawning bash + re-sourcing modules).

### Phase 3: Extract learning engine

**What:** Extract learning-observe/promote/inject/check-promotion/approve-proposals/select-proposals/display-proposals/defer-proposals/undo-promotions, memory-capture into `learning.sh`.

**Why third:** Second-largest cohesive block (~1,200 lines). Depends on state-api.sh (reads COLONY_STATE.json for instincts/learnings) and on pheromone.sh (memory-capture emits pheromones). So it must come after phases 1-2.

### Phase 4: Extract queen system

**What:** Extract queen-init/read/promote/thresholds, incident-rule-add into `queen.sh`.

**Why fourth:** queen-read and queen-promote are called by colony-prime (extracted in phase 2). After phase 2, colony-prime calls them via the dispatch, so extracting queen functions is safe.

### Phase 5: Extract remaining domains

**What:** Extract swarm, session, spawn, flag, suggest, autopilot, changelog, and misc subcommands into their respective modules.

**Why last:** These are smaller, less coupled, and lower risk. Can be done in parallel or sequentially without dependency issues.

**Ordering within phase 5:**
1. **swarm.sh** (800 lines, self-contained, 13 subcommands)
2. **session.sh** (400 lines, 10 subcommands)
3. **spawn.sh** (500 lines, depends on spawn-tree.sh already extracted)
4. **flag.sh** (300 lines, standalone)
5. **suggest.sh** (500 lines, depends on pheromone)
6. **autopilot.sh** (200 lines, standalone)
7. **remaining misc** (error-add, signature-scan, grave-*, etc.)

### Phase 6: Error handling audit

**What:** Classify all 418 `2>/dev/null` instances and replace inappropriate ones.

**Why last:** Must happen after modularization. The error patterns span the entire monolith. After extraction, each module can be audited in isolation. Doing this before extraction would mean touching lines that are about to move, creating merge conflicts.

## State File Coupling Reduction

### The Problem

COLONY_STATE.json has two coupling problems:

1. **Write coupling:** 38 subcommands write to it, often with inline jq. If the schema changes, 38 places need updating.
2. **Read coupling via bypass:** Commands/playbooks do `jq '.current_phase' .aether/data/COLONY_STATE.json` directly, bypassing any validation or locking in aether-utils.sh.

### The Solution: State API Facade

```
BEFORE (scattered access):                 AFTER (centralized):

Command A --jq--> COLONY_STATE.json        Command A --> state-api.sh --> COLONY_STATE.json
Command B --jq--> COLONY_STATE.json        Command B --> state-api.sh -->       |
aether-utils.sh --jq--> COLONY_STATE       aether-utils.sh --> state-api.sh --> |
```

### Implementation Details

**Reader functions** (no lock needed for atomic reads):

| Function | Returns | Replaces |
|----------|---------|----------|
| `state_get_goal` | string | `jq -r '.goal'` |
| `state_get_phase` | int | `jq -r '.current_phase'` |
| `state_get_state` | string (IDLE/BUILDING/COMPLETED) | `jq -r '.state'` |
| `state_get_session_id` | string | `jq -r '.session_id'` |
| `state_get_milestone` | string | `jq -r '.milestone'` |
| `state_get_phases_json` | JSON array | `jq '.plan.phases'` |
| `state_get_phase_count` | int | `jq '.plan.phases \| length'` |
| `state_get_completed_count` | int | `jq '[.plan.phases[] \| select(.status=="completed")] \| length'` |
| `state_get_instincts` | JSON array | `jq '.instincts // []'` |
| `state_get_initialized_at` | string | `jq -r '.initialized_at'` |

**Writer functions** (with locking via acquire_lock/release_lock):

| Function | Does | Replaces |
|----------|------|----------|
| `state_set_phase` | Updates current_phase | inline jq + write |
| `state_set_state` | Updates state field | inline jq + write |
| `state_add_event` | Appends to events[] | inline jq + write |
| `state_update_task_status` | Updates task status in phases | inline jq + write |

**Migration path for commands/playbooks:**

The 20+ inline jq reads in markdown commands cannot call bash functions directly (they run `bash .aether/aether-utils.sh <subcommand>`). Instead, add thin subcommands:

```bash
# In aether-utils.sh dispatch:
state-get) state_get_${1:-goal} ;;
```

Then commands replace:
```bash
# BEFORE
state=$(jq -r '.state // "IDLE"' .aether/data/COLONY_STATE.json)

# AFTER
state=$(bash .aether/aether-utils.sh state-get state)
```

This adds one process spawn per call vs zero for inline jq, but gains correctness (consistent locking, validation, schema migration).

## Error Handling Improvement Pattern

### Classification Framework

Not all `2>/dev/null` is bad. Classify each instance into one of four categories:

| Category | Pattern | Action | Example |
|----------|---------|--------|---------|
| **Legitimate probe** | Checking if command/file exists | Keep as-is | `command -v jq &>/dev/null` |
| **Optional degradation** | Feature works without this | Keep, add comment | `[[ -f "$optional_file" ]] && source ... 2>/dev/null` |
| **Silent data loss** | Write/update fails silently | Replace with json_err or logging | `echo "$updated" > "$file" 2>/dev/null \|\| true` |
| **Debug sabotage** | Error info discarded on a critical path | Replace with structured error | `jq '...' "$state_file" 2>/dev/null` |

### Estimated Distribution (from codebase analysis)

| Category | Est. Count | Action Needed |
|----------|------------|---------------|
| Legitimate probe | ~150 (36%) | None -- these are correct |
| Optional degradation | ~120 (29%) | Add comment explaining why |
| Silent data loss | ~80 (19%) | Replace with error handling |
| Debug sabotage | ~68 (16%) | Replace with structured error |

### Replacement Pattern

**For silent data loss:**

```bash
# BEFORE
echo "$updated" > "$state_file" 2>/dev/null || true

# AFTER
if ! echo "$updated" > "$state_file" 2>/dev/null; then
    json_err "$E_JSON_INVALID" "Failed to write state file" \
        "{\"file\":\"$state_file\"}" \
        "Check disk space and file permissions"
fi
```

**For debug sabotage:**

```bash
# BEFORE
result=$(jq '.plan.phases' "$DATA_DIR/COLONY_STATE.json" 2>/dev/null)

# AFTER
result=$(jq '.plan.phases' "$DATA_DIR/COLONY_STATE.json") || {
    json_err "$E_JSON_INVALID" "Failed to read phases from COLONY_STATE.json"
}
```

### Backward Compatibility

The error handling improvements must NOT change the stdout contract. Subcommands that currently output `{"ok":true,"result":...}` on success must continue to do so. The changes affect:
- stderr output (adding structured errors where there were none)
- exit codes (changing from 0 to non-zero on actual failures)
- Removing `|| true` on critical writes

Callers that check `jq -e '.ok'` on stdout will see no change. Callers that ignore exit codes will see no change. Only callers that specifically relied on silent failure (process continuing after a write fails) will be affected -- and those are bugs, not features.

## Data Flow

### Subcommand Invocation Flow

```
Slash command (markdown)
    |
    v
bash .aether/aether-utils.sh <subcommand> [args]
    |
    v
aether-utils.sh: set -euo pipefail, source infra modules
    |
    v
case "$cmd" in
    <subcommand>) _function_name "$@" ;;
    |
    v
Domain module function (_function_name in utils/<domain>.sh)
    |
    +--> state-api.sh (for COLONY_STATE.json access)
    +--> file-lock.sh (for concurrent access protection)
    +--> json_ok/json_err (stdout/stderr JSON output)
    |
    v
JSON to stdout (consumed by Claude Code / slash commands)
```

### State Mutation Flow (current vs target)

```
CURRENT:
  subcommand --inline jq--> COLONY_STATE.json
  subcommand --inline jq--> COLONY_STATE.json   <-- no lock!
  subcommand --load-state--> COLONY_STATE.json  <-- with lock

TARGET:
  subcommand --state_api--> COLONY_STATE.json   <-- always through API
  subcommand --state_api--> COLONY_STATE.json   <-- always through API
  (API handles lock acquisition internally)
```

## Build Order (Dependency Graph)

```
Phase 1: state-api.sh
    |
    v
Phase 2: pheromone.sh (uses state-api for colony-prime)
    |
    v
Phase 3: learning.sh (uses state-api + pheromone)
    |
    v
Phase 4: queen.sh (used by colony-prime in pheromone.sh)
    |
    +---------------------------------------------+
    v                                             v
Phase 5a: swarm.sh (independent)       Phase 5b: session.sh (independent)
Phase 5c: spawn.sh (independent)       Phase 5d: flag.sh (independent)
Phase 5e: suggest.sh (uses pheromone)  Phase 5f: autopilot.sh (independent)
Phase 5g: changelog.sh (independent)   Phase 5h: misc (remaining)
    |
    v
Phase 6: Error handling audit (all modules)
```

**Each phase is independently testable.** After each extraction:
1. Run `npm run test:bash` (42 bash test files)
2. Run `npm run test:unit` (41 unit test files)
3. Verify subcommand contract: `bash .aether/aether-utils.sh <subcommand>` still returns same JSON

## Dead Code Removal Strategy

The oracle findings identified 76 subcommands (43%) that are never called. These should NOT be removed during modularization. Instead:

1. **During extraction:** Extract all subcommands, including suspected dead ones. This avoids the risk of removing something that IS used but wasn't detected by the static analysis (e.g., dynamically constructed subcommand names).

2. **After extraction:** Add usage tracking via an optional `activity-log` call at the top of each subcommand function. Run for 2-4 weeks across real colonies.

3. **After data collection:** Remove subcommands with zero invocations. Move to a `deprecated/` directory first (not delete) with a 1-version deprecation period.

## Anti-Patterns to Avoid

### Anti-Pattern 1: Lazy Loading via Self-Invocation

**What people do:** `"$SCRIPT_DIR/aether-utils.sh" pheromone-prime` from inside aether-utils.sh itself (colony-prime does this at line 8018).

**Why it's bad:** Spawns a new bash process, re-sources all modules, re-runs setup. For colony-prime (called on every build), this adds ~200ms latency.

**Do this instead:** After extraction, colony-prime and pheromone-prime are in the same module file. Replace the self-invocation with a direct function call: `_pheromone_prime "$@"`.

### Anti-Pattern 2: Inline JSON Construction Without jq

**What people do:** `printf '{"ok":true,"result":{"key":"%s"}}' "$value"` -- building JSON with printf/echo and string interpolation.

**Why it's bad:** Special characters in `$value` (quotes, newlines, backslashes) break the JSON. This is a latent injection/corruption vector.

**Do this instead:** Use jq for all JSON construction: `json_ok "$(jq -n --arg v "$value" '{"key":$v}')"`. This is already done in newer subcommands but not consistently.

### Anti-Pattern 3: Sourcing Everything at Startup

**What people do:** All 10 utility modules are sourced unconditionally on every invocation of aether-utils.sh (lines 26-34).

**Why it's bad:** Running `bash .aether/aether-utils.sh version` sources hive.sh (561 lines), skills.sh (502 lines), midden.sh (260 lines), etc. just to print a version string.

**Do this instead:** Source only infra modules (file-lock, atomic-write, error-handler) at startup. Source domain modules on demand in the case dispatch:

```bash
pheromone-write|pheromone-read|colony-prime|...)
    source "$SCRIPT_DIR/utils/pheromone.sh"
    _${cmd//-/_} "$@"
    ;;
```

**Trade-off:** Adds complexity to the dispatcher but reduces startup time for simple subcommands by ~50ms. Only worth doing after modules are stable. This is an optimization phase, not part of the initial extraction.

## Scaling Considerations

| Scale | Architecture Adjustments |
|-------|--------------------------|
| 1 colony (current) | Monolith is fine for single-user CLI |
| 5-10 concurrent colonies | File locking prevents corruption but serial access may feel slow |
| Package distribution | npm pack size matters -- dead code removal reduces download |
| Developer onboarding | Modular structure makes the codebase navigable for contributors |

### Why This Matters for a CLI Tool

Aether is not a web service -- "scaling" means scaling maintainability and developer comprehension, not throughput. The 11K-line monolith is the single biggest barrier to both. After modularization, a new contributor can understand `pheromone.sh` (1,800 lines in one domain) without reading the other 9,400 lines.

## Integration Points

### Bash <-> Node.js Boundary

| Boundary | Communication | Contract |
|----------|---------------|----------|
| Node CLI install -> Hub setup | Node writes files, bash reads them | File paths and JSON schemas |
| Node update -> bash subcommands | Node copies bash files to hub | File identity (byte-for-byte copy) |
| Bash error codes -> Node error codes | Shared constants | error-handler.sh matches errors.js |
| model-verify.js -> aether-utils.sh | Node checks file existence only | File path convention |

**Key insight:** The boundary is clean. Node and bash share data through files, not process invocation. This means modularization of the bash side has ZERO impact on the Node.js side. The only shared contract is error code strings, and those are defined in both `error-handler.sh` and `errors.js` independently.

### Consumer (Slash Commands) -> Bash Contract

| Contract Element | Specification |
|-----------------|---------------|
| Invocation | `bash .aether/aether-utils.sh <subcommand> [args]` |
| Success output | `{"ok":true,"result":{...}}\n` to stdout |
| Error output | `{"ok":false,"error":{...}}\n` to stderr |
| Exit codes | 0 on success, 1 on error |
| Subcommand names | Stable -- renaming breaks 412 references |

**This contract must be preserved perfectly during modularization.** No subcommand name changes. No output format changes. No exit code changes. The dispatcher pattern (thin case entries delegating to module functions) ensures this naturally.

## Sources

- Direct codebase analysis of `.aether/aether-utils.sh` (11,272 lines, 178 subcommands, main case dispatch at line 981)
- Direct codebase analysis of `.aether/utils/` (21 utility files, 5,237 lines total)
- Direct codebase analysis of `bin/lib/` (16 Node.js modules, 6,578 lines total)
- Existing extraction precedent: hive.sh, midden.sh, skills.sh dispatch pattern (verified working)
- [Bash modularization best practices](https://medium.com/mkdir-awesome/the-ultimate-guide-to-modularizing-bash-script-code-f4a4d53000c2) -- confirms source-based modularization has no meaningful performance overhead
- [Bash scripting best practices 2026](https://oneuptime.com/blog/post/2026-02-13-bash-best-practices/view) -- set -euo pipefail, namespace variables, modularize into separate .sh files
- [Bash subcommand dispatch pattern](https://github.com/xwmx/bash-boilerplate/blob/master/bash-subcommands) -- function-based routing via case dispatch
- [Bash source performance](https://www.baeldung.com/linux/source-include-files) -- no observable performance difference between monolith and split files
- [Error handling with /dev/null](https://www.cyberciti.biz/faq/how-to-redirect-output-and-errors-to-devnull/) -- production best practice: log to file rather than discard

---
*Architecture research for: Aether CLI production hardening*
*Researched: 2026-03-23*
