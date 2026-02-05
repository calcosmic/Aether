# Phase 31: Architecture Evolution - Research

**Researched:** 2026-02-05
**Domain:** Two-tier learning system (project-local + global), spawn tree engine (hierarchical task delegation), Claude Code Task tool platform constraints
**Confidence:** HIGH

## Summary

Phase 31 implements two independent requirements: ARCH-01 (two-tier learning system) and ARCH-02 (spawn tree engine). Research reveals these two requirements have very different risk profiles.

ARCH-01 (two-tier learning) is straightforward. The current memory.json already stores project-local learnings in its `phase_learnings` array. The global tier adds a new file at `~/.aether/learnings.json` with a 50-entry cap. Promotion happens at end-of-project in continue.md Step 2.5, and injection happens after colonization in colonize.md. All changes are prompt-level modifications to existing command files plus one new utility subcommand for managing the global file. The `~/.aether/` directory does not currently exist and must be created.

ARCH-02 (spawn tree engine) faces a confirmed platform blocker. **Claude Code does not allow subagents to use the Task tool.** This is documented in the official docs ("Subagents cannot spawn other subagents") and confirmed by GitHub issues #4182 and #5528 (both closed, no resolution). Workers spawned by the Queen via Task tool cannot themselves spawn sub-workers via Task tool. The CONTEXT.md anticipated this as CP-1 and specifies a fallback: Queen-mediated delegation where workers describe sub-spawn needs in their output and the Queen reads and spawns new workers directly. This fallback is the only viable path. The spawn tree is observable in COLONY_STATE.json but the execution is Queen-orchestrated, not worker-initiated.

**Primary recommendation:** Split into 3 plans: Plan 1 implements the global learnings file and promotion UX in continue.md. Plan 2 implements learning injection in colonize.md. Plan 3 implements the Queen-mediated spawn tree engine in build.md with SPAWN pheromone signaling. Begin with Plan 1 (zero platform risk) and address Plan 3 (spawn tree) last, since it requires the most careful design around the platform constraint.

## Standard Stack

### Core

| File | Current State | Purpose | Change Type |
|------|---------------|---------|-------------|
| `continue.md` | ~500 lines, Step 2.5 has tech debt report | Phase advancement | Add learning promotion UX at project completion (Step 2.5) |
| `colonize.md` | ~476 lines, Step 5 persists findings | Codebase analysis | Add global learning injection after colonization (new Step 5.5) |
| `build.md` | ~1054 lines, Step 5c has wave execution loop | Build orchestration | Add SPAWN pheromone detection and Queen-mediated sub-spawn fulfillment |
| `COLONY_STATE.json` | Has workers, spawn_outcomes, no spawn_tree | Colony state | Add `spawn_tree` object tracking parent-child delegation chains |
| `memory.json` | `{phase_learnings:[], decisions:[], patterns:[]}` | Project-local learnings | No schema changes needed -- phase_learnings already stores learnings |
| `~/.aether/learnings.json` | Does not exist | Global cross-project learnings | New file, created on first promotion |
| `aether-utils.sh` | 302 lines, 16 subcommands | Deterministic shell operations | Add `learning-promote` and `learning-inject` subcommands |

### Supporting (Already Exists, No Changes)

| Component | Purpose | Used By |
|-----------|---------|---------|
| `memory-compress` subcommand | Enforces 20-learning cap, token threshold | Called after writing to memory.json |
| `pheromone-validate` subcommand | Validates pheromone content (min 20 chars) | Called before writing injected FEEDBACK pheromones |
| `pheromone-batch` subcommand | Computes current pheromone strengths | Used in colonize.md to check active signals |
| `validate-state` subcommand | Validates all state files | Used in persistence confirmation steps |
| Worker spec files (6 castes) | Worker behavior definitions | Spawned by Queen in build.md |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `~/.aether/learnings.json` for global tier | SQLite or dedicated config file | JSON is consistent with all other Aether state files. No reason to introduce a new format. |
| Queen-mediated delegation (ARCH-02 fallback) | `claude -p` bash workaround for nested spawning | Discussed in GitHub issue #4182 as a workaround. Loses visibility, error handling, context sharing, and structured output. Not viable for production use. |
| Manual promotion at end-of-project | Auto-promotion based on heuristics | REQUIREMENTS.md explicitly marks auto-promotion as out of scope (ADV-03). Manual promotion preserves user agency. |
| aether-utils.sh subcommands for learning ops | Inline jq in prompts | Centralized utility ensures consistency and reduces prompt length. |

### Installation

No new dependencies. The `~/.aether/` directory is created on first use:

```bash
mkdir -p ~/.aether
```

## Architecture Patterns

### ARCH-01: Two-Tier Learning System

#### Current Learning Flow

```
/ant:build Phase N
  Step 7a: Extract learnings -> memory.json phase_learnings[]
  Step 7b: Emit FEEDBACK pheromone

/ant:continue
  Step 4: Extract learnings (or skip if auto-extracted) -> memory.json phase_learnings[]
  Step 4.5: Emit FEEDBACK/REDIRECT pheromones

memory.json phase_learnings: [{id, phase, phase_name, learnings:[], errors_encountered, timestamp}]
  Cap: 20 entries (enforced by memory-compress)
  Lifecycle: Project-scoped, reset on /ant:init
```

#### Target Two-Tier Architecture

```
Tier 1: Project-Local (UNCHANGED)
  File: .aether/data/memory.json -> phase_learnings[]
  Cap: 20 entries
  Lifecycle: Per-project, reset on /ant:init
  Written by: build.md Step 7a, continue.md Step 4
  Read by: All workers (via Memory Reading in specs)

Tier 2: Global (NEW)
  File: ~/.aether/learnings.json
  Cap: 50 entries (CP-5 decision)
  Lifecycle: Persists across projects, no expiry
  Written by: Manual promotion at end-of-project
  Read by: colonize.md Step 5.5 (injected as FEEDBACK pheromones)
```

#### Global Learnings File Schema

```json
{
  "learnings": [
    {
      "id": "global_<unix_timestamp>_<4_random_hex>",
      "content": "<the learning text>",
      "source_project": "<goal from COLONY_STATE.json when promoted>",
      "source_phase": "<phase number>",
      "tags": ["<tech stack>", "<domain>"],
      "promoted_at": "<ISO-8601 UTC>"
    }
  ],
  "version": 1
}
```

Key design decisions:
- `tags` array enables relevance filtering during injection. Tags are inferred from the colonization results (tech stack, domain keywords).
- `source_project` preserves provenance so users can trace where a learning came from.
- `version` field for future schema evolution.
- 50-entry cap enforced during promotion. When cap reached, present list for user to curate (per CONTEXT.md: overflow strategy is Claude's discretion -- recommend forced curation over FIFO since learnings have different value).

#### Promotion Flow (continue.md Step 2.5)

Promotion happens at end-of-project (all phases complete), as part of the tech debt report flow:

```
continue.md Step 2.5: Generate Tech Debt Report + Promote Learnings
  1. Generate tech debt report (existing)
  2. NEW: Read memory.json phase_learnings
  3. Present all project learnings to user
  4. Suggest top candidates for promotion (Queen's discretion)
     - Criteria: learnings that reference specific tech/patterns, not project-specific details
     - Example good candidate: "bcrypt with 12 rounds causes 800ms delay -- use 10 rounds"
     - Example bad candidate: "Phase 3 had 2 errors" (too project-specific)
  5. User selects which to promote (or "none")
  6. For each selected:
     - Run: bash .aether/aether-utils.sh learning-promote "<content>" "<source_project>" <source_phase> "<tags>"
     - This creates ~/.aether/ if needed, creates learnings.json if needed, appends entry
     - Enforces 50-entry cap (if at cap, display existing learnings and ask user to remove one)
  7. Display: "Promoted N learnings to global tier (~/.aether/learnings.json)"
```

#### Injection Flow (colonize.md Step 5.5)

Injection happens after colonization provides project context:

```
colonize.md (after Phase 31):
  Steps 1-5: Unchanged (read state, pheromones, detect complexity, spawn colonizers, persist findings)
  Step 5.5: Inject Global Learnings (NEW)
    1. Check if ~/.aether/learnings.json exists
       - If not: skip (no global learnings yet)
    2. Read ~/.aether/learnings.json
    3. Read the colonization findings from memory.json (just persisted in Step 5)
       - Extract: tech stack, languages, frameworks, domain keywords
    4. Filter learnings by relevance:
       - Match learning tags against colonization findings
       - Example: if project uses "TypeScript + React", include learnings tagged ["typescript"], ["react"], ["frontend"]
       - Example: exclude learnings tagged ["python", "django"] for a React project
       - If no tag match, skip the learning
    5. For each relevant learning, emit a FEEDBACK pheromone:
       {
         "id": "global_inject_<timestamp>_<hex>",
         "type": "FEEDBACK",
         "content": "Global learning: <learning_content>",
         "strength": 0.5,
         "half_life_seconds": 86400,
         "created_at": "<ISO-8601 UTC>",
         "source": "global:inject",
         "auto": true
       }
       - Validate via pheromone-validate
       - 24h half-life (longer than normal FEEDBACK 6h) so learnings persist through planning
    6. Display injected learnings to user:
       "Injected N global learnings as FEEDBACK pheromones:"
       For each: "  FEEDBACK (0.5, 24h): <first 80 chars>"
  Steps 6-8: Unchanged (display results, reset state, persistence confirmation)
```

### ARCH-02: Spawn Tree Engine (Queen-Mediated Delegation)

#### Platform Constraint (CONFIRMED)

**Claude Code does not allow subagents to spawn other subagents.** This is:
- Documented in official Claude Code docs: "Subagents cannot spawn other subagents"
- Confirmed by GitHub issues #4182 (closed) and #5528 (closed as duplicate)
- The Task tool is not available to subagents at runtime

This means the existing worker spawn pattern (worker uses Task tool to spawn sub-worker) is IMPOSSIBLE. The current worker specs say "Max depth 3 (ant -> sub-ant -> sub-sub-ant)" but this was never achievable.

#### Fallback: Queen-Mediated Delegation

Per CONTEXT.md CP-1 decision: workers describe sub-spawn needs in their output, Queen reads and spawns new workers directly.

The observable result is the same: a depth-2 delegation chain recorded in COLONY_STATE.json spawn_tree. The user should not need to know which path was taken.

#### SPAWN Pheromone Signal

Workers signal sub-spawn needs by including a structured SPAWN request in their output:

```
SPAWN REQUEST:
  caste: builder-ant
  reason: "Need to implement auth middleware separately from routes"
  task: "Create src/middleware/auth.ts with JWT validation"
  context: "Parent task is implementing auth routes. Middleware is an independent sub-task."
  files: ["src/middleware/auth.ts"]
```

This is NOT a pheromone in pheromones.json -- it is a structured output format that the Queen parses from the worker's result text. The term "SPAWN pheromone" from CONTEXT.md refers to this signal mechanism, not the pheromones.json file.

#### Queen-Mediated Execution Flow

```
build.md Step 5c (modified):

  For each worker in wave:
    a-e. Spawn worker (existing flow)

    e2. NEW: Parse SPAWN requests from worker output
        If worker output contains "SPAWN REQUEST:" blocks:
          For each SPAWN request:
            1. Validate depth: current worker is depth 1 (spawned by Queen)
               -> sub-spawn would be depth 2 -> ALLOWED
            2. Record in spawn_tree:
               spawn_tree[child_id] = {
                 parent: worker_id,
                 depth: 2,
                 caste: requested_caste,
                 task: requested_task,
                 status: "pending"
               }
            3. Queue for execution after current wave completes

    f-h. Retry/debugger/conflict check (existing flow)

    i. Post-wave review (existing flow)

    j. NEW: Fulfill SPAWN requests from this wave
       For each queued sub-spawn:
         1. Read the caste's spec file
         2. Spawn via Task tool:
            --- WORKER SPEC ---
            {caste spec}
            --- ACTIVE PHEROMONES ---
            {pheromone block}
            --- PARENT CONTEXT ---
            Parent worker: {parent caste} - {parent task}
            Parent pheromone context (FOCUS/REDIRECT): {inherited from parent}
            --- TASK ---
            {sub-task from SPAWN request}
            You are at depth 2. You CANNOT request further sub-spawns.
            If you need additional work done, handle it inline.
         3. After sub-worker returns:
            Update spawn_tree entry: status -> "completed" or "failed"
            Log: activity-log "COMPLETE" "sub-{caste}-ant" "{task}"
         4. Display in delegation tree visual
```

#### spawn_tree in COLONY_STATE.json

```json
{
  "spawn_tree": {
    "wave1_builder1": {
      "id": "wave1_builder1",
      "caste": "builder-ant",
      "task": "Implement auth routes",
      "depth": 1,
      "parent": "queen",
      "children": ["sub_builder1_1"],
      "status": "completed"
    },
    "sub_builder1_1": {
      "id": "sub_builder1_1",
      "caste": "builder-ant",
      "task": "Create auth middleware",
      "depth": 2,
      "parent": "wave1_builder1",
      "children": [],
      "status": "completed"
    }
  }
}
```

#### Depth Enforcement

- Depth 1 workers (spawned by Queen): CAN include SPAWN REQUEST blocks
- Depth 2 workers (sub-spawned by Queen on behalf of depth-1): CANNOT sub-spawn further
  - Their prompt explicitly states: "You are at depth 2. You CANNOT request further sub-spawns."
  - Even if they include SPAWN REQUEST, Queen ignores it
- Depth cap: 2 (enforced by Queen's parsing logic, not by platform)

#### Delegation Tree Visual Display

Added to build.md Step 7e:

```
Delegation Tree:
  Queen
  ├── builder-ant: Implement auth routes [COMPLETE]
  │   └── builder-ant (sub): Create auth middleware [COMPLETE]
  ├── builder-ant: Implement user endpoints [COMPLETE]
  └── watcher-ant: Verify auth module [COMPLETE]
```

Rendered via Bash tool with ANSI colors:
```bash
bash -c 'printf "\e[1;33mDelegation Tree:\e[0m\n"'
bash -c 'printf "  \e[1;33mQueen\e[0m\n"'
bash -c 'printf "  ├── \e[32mbuilder-ant\e[0m: Implement auth routes [\e[32mCOMPLETE\e[0m]\n"'
bash -c 'printf "  │   └── \e[32mbuilder-ant\e[0m (sub): Create auth middleware [\e[32mCOMPLETE\e[0m]\n"'
```

#### Spawn Tree Mode Invariance

Per CONTEXT.md: spawn tree works the same across LIGHTWEIGHT/STANDARD/FULL modes. Always allows depth-2. No mode-specific behavior changes.

### Anti-Patterns to Avoid

- **Worker-initiated spawning via Task tool:** Workers CANNOT use the Task tool. This is a platform constraint, not a configuration issue. All spawning must go through the Queen.
- **Using `claude -p` bash workaround:** While mentioned in GitHub issue #4182, this loses visibility, structured output, error handling, and context sharing. Not viable.
- **Storing global learnings in .aether/data/:** Global learnings MUST be in `~/.aether/` (user home directory) to persist across projects. The `.aether/data/` directory is project-scoped.
- **Auto-promoting learnings:** REQUIREMENTS.md explicitly defers this (ADV-03). Only manual promotion.
- **Injecting learnings during /ant:init:** Per CONTEXT.md, injection happens after colonization because colonization provides the tech stack context needed for relevance filtering.
- **SPAWN pheromone in pheromones.json:** The SPAWN signal is a worker output format, not a pheromone signal stored in pheromones.json. Using pheromones.json would require workers to write to files (they can, but it adds complexity and race conditions).

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Global learnings file management | Inline jq in prompts for ~/.aether/learnings.json | New `learning-promote` subcommand in aether-utils.sh | Centralized creation/append/cap enforcement; handles mkdir -p ~/.aether |
| Learning injection filtering | Complex prompt-level tag matching | New `learning-inject` subcommand in aether-utils.sh | Takes tech stack keywords, returns matching learnings as JSON |
| Pheromone validation for injected learnings | Custom validation | Existing `pheromone-validate` subcommand | Already validates content length (>20 chars) |
| spawn_tree state management | Manual JSON manipulation in prompts | Queen reads/writes spawn_tree in COLONY_STATE.json directly | The spawn_tree is simple enough for prompt-level Write tool usage, but validate-state should check it |
| Depth enforcement for sub-spawns | Worker-side depth checking | Queen-side depth checking when parsing SPAWN requests | Workers can't enforce platform constraints; Queen controls all spawning |

**Key insight:** ARCH-01 (learning system) needs 2 new utility subcommands because it manages a file outside the project directory. ARCH-02 (spawn tree) does NOT need new utilities -- the Queen orchestrates directly and the spawn_tree is just a JSON object in COLONY_STATE.json.

## Common Pitfalls

### Pitfall 1: Assuming Workers Can Spawn Sub-Workers

**What goes wrong:** Implementation assumes the Task tool is available to subagents, tries to make workers spawn directly, and silently fails.
**Why it happens:** The existing worker specs say "Max depth 3" and include spawn instructions. This was aspirational, not tested.
**How to avoid:** All spawning goes through the Queen. Workers signal needs via structured output (SPAWN REQUEST). Queen parses and fulfills. The worker specs' spawn instructions remain useful for describing WHAT the worker needs, but the HOW is always Queen-mediated.
**Warning signs:** Workers reporting "Task tool not available" or silently failing to spawn.

### Pitfall 2: Global Learnings File Not Created

**What goes wrong:** `~/.aether/learnings.json` doesn't exist on first use, causing injection to fail.
**Why it happens:** The `~/.aether/` directory doesn't exist by default. It's only created when the user first promotes a learning.
**How to avoid:** The `learning-promote` subcommand must `mkdir -p ~/.aether` before writing. The `learning-inject` subcommand must check existence and gracefully return empty results if the file doesn't exist. Colonize.md must handle "no global learnings" as a normal case, not an error.
**Warning signs:** Errors during colonization about missing files.

### Pitfall 3: Learning Injection Irrelevance

**What goes wrong:** Global learnings from a Python/Django project are injected into a React/TypeScript project, providing misleading guidance.
**Why it happens:** Tag matching is too loose or tags are too generic.
**How to avoid:** Tags must be specific enough to filter effectively. During promotion, infer tags from the colonization results at the time of the source project (tech stack, frameworks, domain). During injection, match against the current project's colonization results. If no tags match, don't inject.
**Warning signs:** Learnings about Python appearing in a JavaScript project's pheromone context.

### Pitfall 4: SPAWN Requests Overwhelming the Build

**What goes wrong:** Every worker includes SPAWN REQUEST blocks for minor sub-tasks, causing the Queen to spawn dozens of sub-workers.
**Why it happens:** Workers are told they CAN signal sub-spawn needs, so they aggressively use the feature.
**How to avoid:** The worker prompt must emphasize: "Only signal SPAWN REQUEST for INDEPENDENT sub-tasks that are genuinely separate from your main task. If you can handle it inline, handle it inline." Additionally, the Queen should cap sub-spawns at a reasonable number (e.g., max 2 sub-spawns per wave).
**Warning signs:** Build times doubling due to excessive sub-spawning.

### Pitfall 5: Spawn Tree Growing build.md Beyond Effective Length

**What goes wrong:** Adding SPAWN request parsing, sub-spawn fulfillment, and delegation tree display to build.md (already ~1054 lines) pushes it past the effective prompt length limit.
**Why it happens:** build.md is the most complex command file and accumulates features each phase.
**How to avoid:** Keep the spawn tree logic concise. The Queen already does wave-based spawning -- sub-spawns are just additional spawns in a post-wave step. The logic is "parse structured text, spawn Task tool, update JSON" -- not fundamentally new. Target: ~40-50 lines of additions.
**Warning signs:** build.md exceeding ~1100 lines. Late steps being ignored by Claude.

### Pitfall 6: Promotion UX Blocking Project Completion

**What goes wrong:** The promotion step requires interactive user input at project completion, but the user is running auto-continue mode (--all) and the build stalls.
**Why it happens:** Auto-continue mode skips user interaction. The promotion step assumes interactive flow.
**How to avoid:** In auto-continue mode, skip promotion entirely. Display: "Global learning promotion available. Run /ant:continue (without --all) to promote learnings." Promotion requires conscious user choice -- it should never be automatic.
**Warning signs:** Auto-continue hanging at project completion.

### Pitfall 7: Updating Worker Specs with Stale Spawn Instructions

**What goes wrong:** Worker specs still say "Max depth 3, max 5 sub-ants per ant" and include full spawn instructions, but spawning doesn't work as described.
**Why it happens:** The specs were written before the platform constraint was confirmed. Phase 31 changes the spawning model but doesn't update worker specs.
**How to avoid:** Update worker specs to replace the "You Can Spawn Other Ants" section with a "You Can Request Sub-Spawns" section that describes the SPAWN REQUEST output format. Remove references to max depth 3 (now max depth 2). Remove the spawn-check gate instructions from worker specs (the Queen handles depth checking). Keep the spawn confidence check as advisory guidance for what caste to request.
**Warning signs:** Workers confused by contradictory instructions about spawning.

## Code Examples

### Global Learnings File (new file at ~/.aether/learnings.json)

```json
{
  "learnings": [
    {
      "id": "global_1707123456_a1b2",
      "content": "bcrypt with 12 salt rounds causes 800ms delay per hash -- use 10 rounds for 200ms",
      "source_project": "Build a REST API with authentication",
      "source_phase": 3,
      "tags": ["typescript", "authentication", "bcrypt", "performance"],
      "promoted_at": "2026-02-05T12:30:00Z"
    },
    {
      "id": "global_1707123789_c3d4",
      "content": "Integration tests caught missing error handlers that unit tests missed -- always include integration tests for API endpoints",
      "source_project": "Build a REST API with authentication",
      "source_phase": 5,
      "tags": ["testing", "api", "integration-tests"],
      "promoted_at": "2026-02-05T12:30:00Z"
    }
  ],
  "version": 1
}
```

### learning-promote Subcommand (new in aether-utils.sh)

```bash
learning-promote)
  [[ $# -ge 3 ]] || json_err "Usage: learning-promote <content> <source_project> <source_phase> [tags]"
  content="$1"
  source_project="$2"
  source_phase="$3"
  tags="${4:-}"

  # Ensure global directory exists
  global_dir="$HOME/.aether"
  global_file="$global_dir/learnings.json"
  mkdir -p "$global_dir"

  # Create file if it doesn't exist
  if [[ ! -f "$global_file" ]]; then
    echo '{"learnings":[],"version":1}' > "$global_file"
  fi

  id="global_$(date -u +%s)_$(head -c 2 /dev/urandom | od -An -tx1 | tr -d ' ')"
  ts=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

  # Parse tags into JSON array
  if [[ -n "$tags" ]]; then
    tags_json=$(echo "$tags" | jq -R 'split(",")')
  else
    tags_json="[]"
  fi

  # Check cap (50 entries)
  current_count=$(jq '.learnings | length' "$global_file")
  if [[ $current_count -ge 50 ]]; then
    json_ok "{\"promoted\":false,\"reason\":\"cap_reached\",\"current_count\":$current_count,\"cap\":50}"
    exit 0
  fi

  # Append learning
  updated=$(jq --arg id "$id" --arg content "$content" --arg sp "$source_project" \
    --argjson phase "$source_phase" --argjson tags "$tags_json" --arg ts "$ts" '
    .learnings += [{
      id: $id,
      content: $content,
      source_project: $sp,
      source_phase: $phase,
      tags: $tags,
      promoted_at: $ts
    }]
  ' "$global_file") || json_err "Failed to update learnings.json"

  echo "$updated" > "$global_file"
  json_ok "{\"promoted\":true,\"id\":\"$id\",\"count\":$((current_count + 1)),\"cap\":50}"
  ;;
```

### learning-inject Subcommand (new in aether-utils.sh)

```bash
learning-inject)
  [[ $# -ge 1 ]] || json_err "Usage: learning-inject <tech_keywords_csv>"
  keywords="$1"

  global_file="$HOME/.aether/learnings.json"

  # If no global learnings file exists, return empty
  if [[ ! -f "$global_file" ]]; then
    json_ok '{"learnings":[],"count":0}'
    exit 0
  fi

  # Filter learnings by tag match against provided keywords
  json_ok "$(jq --arg kw "$keywords" '
    ($kw | split(",") | map(ascii_downcase | ltrimstr(" ") | rtrimstr(" "))) as $keywords |
    .learnings | map(
      select(
        .tags as $tags |
        ($keywords | any(. as $k | $tags | any(ascii_downcase | contains($k))))
      )
    ) | {learnings: ., count: length}
  ' "$global_file")"
  ;;
```

### SPAWN REQUEST Output Format (for worker specs)

```markdown
## Requesting Sub-Spawns

If you encounter a sub-task that is genuinely INDEPENDENT from your main task and would benefit from a separate worker, include a SPAWN REQUEST block in your output:

SPAWN REQUEST:
  caste: builder-ant
  reason: "Need to implement auth middleware separately from routes"
  task: "Create src/middleware/auth.ts with JWT validation logic"
  context: "Parent task is implementing auth routes. Middleware is independent."
  files: ["src/middleware/auth.ts"]

The Queen will read your SPAWN REQUEST and spawn a sub-worker on your behalf.

Rules:
- Only use SPAWN REQUEST for truly independent sub-tasks
- If you can handle the task inline, DO handle it inline
- Maximum 1-2 SPAWN REQUESTs per worker -- don't fragment your work
- You are at depth {your_depth}. If depth is 2, you CANNOT include SPAWN REQUESTs
- The sub-worker will inherit your pheromone context (FOCUS/REDIRECT)
```

### Promotion UX in continue.md Step 2.5

```markdown
### Step 2.5b: Promote Learnings to Global Tier (NEW)

After generating the tech debt report:

1. Read `.aether/data/memory.json` phase_learnings array
2. If phase_learnings is empty: skip to Step 2.5c

3. Display learnings with promotion candidates:
   ```
   PROJECT LEARNINGS:

   Candidates for global promotion (applicable across projects):
     [1] "bcrypt with 12 rounds causes 800ms delay -- use 10 rounds" (Phase 3)
     [2] "Integration tests caught missing error handler" (Phase 5)

   Project-specific (not recommended for promotion):
     [-] "Phase 2 had config issues with local env" (Phase 2)
     [-] "Auth routes needed 3 retries" (Phase 4)

   Which learnings would you like to promote? (numbers, "all candidates", or "none")
   ```

4. Queen suggests candidates based on:
   - References specific tech/patterns (not project-specific details)
   - Would be useful in a different project
   - Not overly narrow or time-sensitive

5. For each selected learning:
   - Infer tags from colonization findings (tech stack, domain)
   - Run: `bash .aether/aether-utils.sh learning-promote "<content>" "<goal>" <phase> "<tags>"`
   - If cap_reached: display existing learnings, ask user to remove one first

6. Display promotion result:
   ```
   Promoted {N} learnings to global tier.
     ~/.aether/learnings.json ({count}/50 entries)
   ```
```

### Injected Learning as FEEDBACK Pheromone (colonize.md Step 5.5)

```json
{
  "id": "global_inject_1707123456_a1b2",
  "type": "FEEDBACK",
  "content": "Global learning: bcrypt with 12 salt rounds causes 800ms delay -- use 10 rounds for 200ms",
  "strength": 0.5,
  "half_life_seconds": 86400,
  "created_at": "2026-02-05T14:00:00Z",
  "source": "global:inject",
  "auto": true
}
```

Note: 24-hour half-life (vs normal 6-hour FEEDBACK) ensures learning persists through planning phase.

### spawn_tree Update (COLONY_STATE.json after build with sub-spawns)

```json
{
  "goal": "Build a REST API with authentication",
  "state": "READY",
  "current_phase": 4,
  "spawn_tree": {
    "phase3_wave1_builder1": {
      "id": "phase3_wave1_builder1",
      "caste": "builder-ant",
      "task": "Implement auth routes",
      "depth": 1,
      "parent": "queen",
      "children": ["phase3_sub_builder1_1"],
      "status": "completed",
      "phase": 3,
      "wave": 1
    },
    "phase3_sub_builder1_1": {
      "id": "phase3_sub_builder1_1",
      "caste": "builder-ant",
      "task": "Create auth middleware",
      "depth": 2,
      "parent": "phase3_wave1_builder1",
      "children": [],
      "status": "completed",
      "phase": 3,
      "wave": 1
    },
    "phase3_wave1_builder2": {
      "id": "phase3_wave1_builder2",
      "caste": "builder-ant",
      "task": "Implement user endpoints",
      "depth": 1,
      "parent": "queen",
      "children": [],
      "status": "completed",
      "phase": 3,
      "wave": 1
    }
  },
  "workers": { ... },
  "spawn_outcomes": { ... }
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Single-tier learning (memory.json only) | Two-tier: project-local + global (~/.aether/) | Phase 31 (new) | Knowledge persists across projects; relevant learnings injected into new projects |
| Workers claim to spawn sub-workers (Task tool) | Queen-mediated delegation: workers signal, Queen spawns | Phase 31 (new) | Acknowledges platform constraint; achieves same observable result (depth-2 delegation chain) |
| Worker specs say "Max depth 3" | Max depth 2 (Queen -> worker -> sub-worker via Queen) | Phase 31 (new) | Honest depth limit matching platform capability |
| No spawn tree recorded | spawn_tree in COLONY_STATE.json with parent-child relationships | Phase 31 (new) | Delegation chain visible and auditable |

**What stays unchanged:**
- memory.json schema (phase_learnings format)
- Pheromone system (FEEDBACK/REDIRECT/FOCUS/INIT types)
- Worker caste system (6 castes, same specs)
- Build loop (waves, workers, retry, debugger, reviewer)
- Activity log system

## Open Questions

### 1. Tag Inference Quality

**What we know:** Tags for global learnings are inferred from colonization results. The colonize command identifies tech stack, languages, and frameworks.
**What's unclear:** How well the Queen can infer meaningful tags at promotion time. If tags are too generic ("code"), everything matches. If too specific ("bcrypt-version-12"), nothing matches.
**Recommendation:** Start simple. Use language names and framework names as tags. The colonize command already identifies these. Users can manually adjust tags during promotion if the suggestions are poor. Monitor and refine in future phases.

### 2. Sub-Spawn Timing (Claude's Discretion)

**What we know:** CONTEXT.md leaves spawn timing to Claude's discretion -- whether Queen fulfills SPAWN pheromones between waves (batched) or immediately mid-wave.
**What's unclear:** Which timing produces better results.
**Recommendation:** Batch fulfillment between waves (post-wave step j). Reasons: (a) mid-wave spawning complicates the wave execution loop, (b) sub-tasks may benefit from the parent worker's full output (not just partial), (c) simpler implementation. If a worker's SPAWN REQUEST is urgent, the worker can flag it and the Queen can prioritize it in the next batch.

### 3. validate-state Extension for spawn_tree

**What we know:** The `validate-state colony` subcommand currently checks for goal, state, current_phase, workers, spawn_outcomes.
**What's unclear:** Whether spawn_tree needs schema validation.
**Recommendation:** Add spawn_tree as an optional field in validation (pass if missing, validate structure if present). This keeps backward compatibility with existing state files.

### 4. Overflow Strategy for Global Learnings Cap

**What we know:** CONTEXT.md says overflow strategy is Claude's discretion. Options: FIFO replacement vs forced curation.
**What's unclear:** Which is better UX.
**Recommendation:** Forced curation. When cap is reached, display all 50 learnings and ask user to select one to remove before adding new one. Learnings have different value; FIFO would discard the most valuable old learning. Forced curation ensures the user maintains a curated set. This only triggers at 50 entries -- infrequent enough to not be annoying.

## Sources

### Primary (HIGH confidence)

- **Claude Code official docs** ([Create custom subagents](https://code.claude.com/docs/en/sub-agents)) -- Confirmed: "Subagents cannot spawn other subagents." This is the authoritative source for the platform constraint.
- **GitHub issue #4182** ([Sub-Agent Task Tool Not Exposed](https://github.com/anthropics/claude-code/issues/4182)) -- Closed. Task tool not available to subagents. Workaround via `claude -p` documented but not viable.
- **GitHub issue #5528** (Closed as duplicate of #4182) -- Additional confirmation that hierarchical task decomposition via nested Task tool is not supported.
- `.claude/commands/ant/build.md` -- Full build orchestration, ~1054 lines. Integration point for spawn tree engine (Step 5c).
- `.claude/commands/ant/continue.md` -- Full continue flow, ~500 lines. Integration point for learning promotion (Step 2.5).
- `.claude/commands/ant/colonize.md` -- Full colonize flow, ~476 lines. Integration point for learning injection (Step 5.5).
- `.claude/commands/ant/init.md` -- Init flow, ~199 lines. Creates initial state. No changes needed.
- `.aether/data/COLONY_STATE.json` -- Current schema: goal, state, current_phase, mode, workers, spawn_outcomes. spawn_tree to be added.
- `.aether/data/memory.json` -- Current schema: phase_learnings[], decisions[], patterns[]. No changes needed.
- `.aether/aether-utils.sh` -- 302 lines, 16 subcommands. 2 new subcommands needed for learning ops.
- `.aether/workers/builder-ant.md` -- Worker spec with current spawn instructions that need updating.
- `.aether/workers/watcher-ant.md` -- Worker spec with current spawn instructions that need updating.
- `.planning/REQUIREMENTS.md` -- ARCH-01, ARCH-02 definitions. ADV-03 (auto-promote) explicitly out of scope.
- `.planning/ROADMAP.md` -- Phase 31 success criteria, CP-1 blocker flag, CP-5 (50 entry cap).
- `.planning/phases/31-architecture-evolution/31-CONTEXT.md` -- User decisions constraining implementation.
- `.planning/phases/26-auto-learning/26-RESEARCH.md` -- Prior research on learning extraction flow (directly relevant for promotion trigger point).
- `.planning/phases/30-automation/30-RESEARCH.md` -- Prior research on reviewer/debugger/visual output (context for build.md integration points).

### Secondary (MEDIUM confidence)

None -- all findings verified against primary codebase sources and official documentation.

### Tertiary (LOW confidence)

None -- this phase is entirely internal prompt modifications plus one new global file.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH -- All components are existing codebase files plus one new global file. No external dependencies.
- Architecture (two-tier learning): HIGH -- Follows established patterns from Phase 26 auto-learning. Simple file operations.
- Architecture (spawn tree engine): HIGH -- Platform constraint confirmed from multiple authoritative sources. Fallback design (Queen-mediated) is architecturally sound and matches existing build.md patterns.
- Pitfalls: HIGH -- Based on direct analysis of platform constraints, existing code patterns, and CONTEXT.md decisions.
- Code examples: HIGH -- Based on existing patterns in aether-utils.sh and command files.

**Research date:** 2026-02-05
**Valid until:** 2026-03-07 (stable for learning system; spawn tree depends on Claude Code platform -- monitor GitHub issues for nested Task tool support)
