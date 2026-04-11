### Step 4.5: Checkpoint State

Before modifying colony state during the build, create a rolling backup:

Run using the Bash tool with description "Checkpointing colony state...":
```bash
aether state-checkpoint "pre-build-wave" 2>/dev/null || echo "Warning: State checkpoint failed -- continuing without backup" >&2
```

This creates a timestamped backup of COLONY_STATE.json in `.aether/data/backups/` with at most 3 retained.

### Step 5: Analyze Tasks

**YOU (the Queen) will spawn workers directly. Do NOT delegate to a single Prime Worker.**

**Show build header:**
```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Phase {id}: {name} — {N} waves, {M} tasks
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

Where N = number of builder waves (excluding watcher/chaos) and M = total builder tasks.

Record `build_started_at_epoch=$(date +%s)` — this epoch integer is used by the BUILD SUMMARY block in Step 7 to calculate elapsed time.

Analyze the phase tasks:

1. **Group tasks by dependencies:**
   - **Wave 1:** Tasks with `depends_on: "none"` or `depends_on: []` (can run in parallel)
   - **Wave 2:** Tasks depending on Wave 1 tasks
   - **Wave 3+:** Continue until all tasks assigned

2. **Assign castes:**
   - Implementation tasks → 🔨🐜 Builder
   - Research/docs tasks → 🔍🐜 Scout (**only if `colony_depth` is "standard", "deep", or "full"**; at "light" depth, reassign to Builder or skip)
   - Testing/validation → 👁️🐜 Watcher (ALWAYS spawn at least one)
   - Resilience testing → 🎲🐜 Chaos (ALWAYS spawn one after Watcher)

3. **Generate ant names for each worker:**

Run using the Bash tool with description "Naming builder ant...": `aether generate-ant-name "builder"`
Run using the Bash tool with description "Naming watcher ant...": `aether generate-ant-name "watcher"`
Run using the Bash tool with description "Naming chaos ant...": `aether generate-ant-name "chaos"`

Display spawn plan with caste emojis:
```
━━━ 🐜 S P A W N   P L A N ━━━

Wave 1  — Parallel
  🔨🐜 {Builder-Name}  Task {id}  {description}
  🔨🐜 {Builder-Name}  Task {id}  {description}

Wave 2  — After Wave 1
  🔨🐜 {Builder-Name}  Task {id}  {description}

Verification
  👁️🐜 {Watcher-Name}  Verify all work independently
  {if colony_depth == "full": 🎲🐜 {Chaos-Name}   Resilience testing (after Watcher)}

Total: {N} Builders + 1 Watcher{if colony_depth == "full": " + 1 Chaos"}{if colony_depth in ["deep","full"]: " + 1 Oracle + 1 Architect"} = {total} spawns
```

**Caste Emoji Legend:**
- 🔨🐜 Builder  (cyan if color enabled)
- 👁️🐜 Watcher  (green if color enabled)
- 🎲🐜 Chaos    (red if color enabled)
- 🔍🐜 Scout    (yellow if color enabled)
- 🏺🐜 Archaeologist (magenta if color enabled)
- 🔮🐜 Oracle    (indigo if color enabled) — deep research specialist
- 🏛️🐜 Architect  (violet if color enabled) — architecture design specialist
- 🥚 Queen/Prime

**Every spawn must show its caste emoji.**

**Add to Caste Emoji Legend:**
- 🔌🐜 Ambassador (blue if color enabled) — external integration specialist

### Step 5.0.1: Oracle Research Step (Non-Blocking)

**DEPTH CHECK: Skip if colony depth is "light" or "standard".**

The `colony_depth` value is available from build-prep.md cross-stage state.
- If `colony_depth` is "light" or "standard": Display `Oracle skipped (depth: {colony_depth})` and skip to Step 5.0.5.
- If `colony_depth` is "deep" or "full": Proceed with existing Oracle spawn logic below.

**Oracle runs BEFORE worker waves. Failure is non-blocking -- the build continues with a warning.**

1. **Generate Oracle name:**
   Run using the Bash tool with description "Naming oracle ant...": `aether generate-ant-name "oracle"` (store as `{oracle_name}`)

2. **Log spawn:**
   Run using the Bash tool with description "Dispatching oracle...": `aether spawn-log --parent "Queen" --caste "oracle" --name "{oracle_name}" --task "Phase {phase_id} research" --depth 0`

3. **Display announcement:**
```
━━━ 🔮 O R A C L E   R E S E A R C H ━━━
──── 🔮🐜 Spawning {oracle_name} — Phase {phase_id} research ────
```

4. **Spawn Oracle using Task tool with `subagent_type="aether-oracle"`**, include `description: "🔮 Oracle {oracle_name}: Phase {phase_id} research"` (DO NOT use run_in_background):

   > **Platform note**: In Claude Code, use `Task tool with subagent_type`. In OpenCode, use the equivalent agent spawning mechanism for your platform.

   **Oracle Worker Prompt:**
   ```
   You are {oracle_name}, a 🔮 Oracle Ant.

   Mission: Conduct deep research for Phase {phase_id}

   Phase: {phase_name}
   Colony goal: "{colony_goal}"

   Active pheromone signals (if any):
   {pheromone_summary from prompt_section}

   Tasks to research:
   {list of phase tasks with descriptions}

   Work:
   1. Analyze the codebase for patterns relevant to these tasks
   2. Research best practices, gotchas, and recommended approaches
   3. Identify potential risks or dependencies
   4. Write findings to `.aether/data/research/oracle-{phase_id}.md`

   **IMPORTANT:** This is a single-pass research invocation (not iterative RALF loop). Produce your best findings in one pass.

   Return ONLY this JSON (no other text):
   {
     "ant_name": "{oracle_name}",
     "caste": "oracle",
     "status": "completed" | "failed" | "blocked",
     "summary": "Key findings and recommendations",
     "findings": ["finding1", "finding2"],
     "recommendations": ["rec1", "rec2"],
     "risks": ["risk1"],
     "research_file": ".aether/data/research/oracle-{phase_id}.md",
     "blockers": []
   }
   ```

5. **Parse Oracle JSON output:**
   - If status is `"completed"`, store `oracle_findings` for injection into Architect and Builder prompts.
   - If status is `"failed"` or `"blocked"`, log warning and set `oracle_findings` to empty:

```
⚠ Oracle {oracle_name} research unavailable — proceeding without research context
```

6. **Log completion:**
   Run using the Bash tool with description "Recording oracle completion...": `aether spawn-complete --name "{oracle_name}" --status "{status}" --summary "{summary}"`

### Step 5.0.2: Architect Design Step (Non-Blocking)

**DEPTH CHECK: Skip if colony depth is "light" or "standard".**

Architect depends on Oracle findings. If Oracle was skipped, Architect must also be skipped.
- If `colony_depth` is "light" or "standard": Display `Architect skipped (depth: {colony_depth})` and skip to Step 5.0.5.
- If `colony_depth` is "deep" or "full": Proceed with existing Architect spawn logic below.

**Architect runs AFTER Oracle, BEFORE worker waves. Failure is non-blocking -- the build continues with a warning.**

1. **Generate Architect name:**
   Run using the Bash tool with description "Naming architect ant...": `aether generate-ant-name "architect"` (store as `{architect_name}`)

2. **Log spawn:**
   Run using the Bash tool with description "Dispatching architect...": `aether spawn-log --parent "Queen" --caste "architect" --name "{architect_name}" --task "Phase {phase_id} design" --depth 0`

3. **Display announcement:**
```
━━━ 🏛️ A R C H I T E C T   D E S I G N ━━━
──── 🏛️🐜 Spawning {architect_name} — Phase {phase_id} design ────
```

4. **Spawn Architect using Task tool with `subagent_type="aether-architect"`**, include `description: "🏛️ Architect {architect_name}: Phase {phase_id} design"` (DO NOT use run_in_background):

   > **Platform note**: In Claude Code, use `Task tool with subagent_type`. In OpenCode, use the equivalent agent spawning mechanism for your platform.

   **Architect Worker Prompt:**
   ```
   You are {architect_name}, a 🏛️ Architect Ant.

   Mission: Design architecture for Phase {phase_id}

   Phase: {phase_name}
   Colony goal: "{colony_goal}"

   {oracle_findings if available, otherwise: "No Oracle research available for this phase."}

   Active pheromone signals (if any):
   {pheromone_summary from prompt_section}

   Tasks to design for:
   {list of phase tasks with descriptions}

   Work:
   1. Analyze codebase structure and existing patterns
   2. Identify architectural boundaries and component relationships
   3. Design approach (component structure, data flow, interfaces)
   4. Write design document to `.aether/data/research/architect-{phase_id}.md`
   5. Return actionable design decisions for Builder consumption

   Return ONLY this JSON (no other text):
   {
     "ant_name": "{architect_name}",
     "caste": "architect",
     "status": "completed" | "failed" | "blocked",
     "summary": "Design approach and key decisions",
     "design_decisions": ["decision1", "decision2"],
     "component_structure": {"overview": "..."},
     "data_flow": {"overview": "..."},
     "design_file": ".aether/data/research/architect-{phase_id}.md",
     "blockers": []
   }
   ```

5. **Parse Architect JSON output:**
   - If status is `"completed"`, store `architect_design` for injection into Builder prompts.
   - If status is `"failed"` or `"blocked"`, log warning and set `architect_design` to empty:

```
⚠ Architect {architect_name} design unavailable — proceeding without design context
```

6. **Log completion:**
   Run using the Bash tool with description "Recording architect completion...": `aether spawn-complete --name "{architect_name}" --status "{status}" --summary "{summary}"`

### Step 5.0.5: Select and Announce Workflow Pattern

Examine the phase name and task descriptions. Select the first matching pattern:

| Phase contains | Pattern |
|----------------|---------|
| "bug", "fix", "error", "broken", "failing" | Investigate-Fix |
| "research", "oracle", "explore", "investigate" | Deep Research |
| "refactor", "restructure", "clean", "reorganize" | Refactor |
| "security", "audit", "compliance", "accessibility", "license" | Compliance |
| "docs", "documentation", "readme", "guide" | Documentation Sprint |
| (default) | SPBV |

Display the selected pattern:
```
━━ Pattern: {pattern_name} ━━
{announce_line from Queen's Workflow Patterns definition}
```

Store `selected_pattern` for inclusion in the BUILD SUMMARY (Step 7).

### Step 5.0.7: Pre-Wave Worktree Allocation

**This step runs BEFORE each wave (Wave 1, Wave 2, etc.). It allocates git worktrees for workers when parallel mode is "worktree".**

1. **Read the current parallel mode:**

Run using the Bash tool with description "Reading parallel mode...":
```bash
aether parallel-mode get 2>/dev/null || echo '{"result":{"mode":"in-repo","source":"default"}}'
```

Parse the JSON result. Extract `mode` (string) and `source` (string).

2. **Branch on mode:**

**If `mode` is `"in-repo"` (or empty/missing -- default):**

Display:
```
━━━ 🌿 Worktree allocation skipped (mode: in-repo) ━━━
```

Set `worktree_allocations` to empty `{}` and proceed to wave spawning.

**If `mode` is `"worktree"`:**

Display:
```
━━━ 🌳 W O R K T R E E   A L L O C A T I O N ━━━
```

3. **Allocate worktrees for each worker in the upcoming wave:**

For each worker in the current wave (builders and scouts -- castes that modify files):

Run using the Bash tool with description "Allocating worktree for {ant_name}...":
```bash
aether worktree-allocate --phase {phase_number} --agent {ant_name}
```

Parse the JSON result. Extract `path` (string), `branch` (string), and `id` (string).

- **If allocation succeeds** (result contains `"ok": true`):
  - Store the mapping: `worktree_allocations[{ant_name}] = {path, branch, id}`
  - Display: `  🌳 {ant_name}: {branch} -> {path}`

- **If allocation fails** (result contains error):
  - Display warning: `  ⚠ {ant_name}: worktree allocation failed -- {error_message}`
  - The worker will operate in-repo as fallback (do not halt the build)
  - Store `worktree_allocations[{ant_name}] = null` to signal no worktree available

4. **Display allocation summary:**

```
Allocated: {count} / {total} workers
{if any failed: "Fallback: {failed_count} workers will operate in-repo"}
```

5. **Store for injection:**

Store `worktree_allocations` for use in builder prompts (Task 5.2 handles the actual injection into worker context).

> **Design note:** Worktree allocation happens per-wave so that Wave 2+ workers get fresh worktrees only when needed. Wave 1 allocations are reused if the worker appears in a later wave.

### Step 5.1: Spawn Wave 1 Workers (Parallel)

**CRITICAL: Spawn ALL Wave 1 workers in a SINGLE message using multiple Task tool calls.**

**Announce the wave before spawning:**

Display the spawn announcement immediately before firing Task calls:

For single-caste waves (typical — all builders):
```
──── 🔨🐜 Spawning {N} Builders in parallel ────
```

For mixed-caste waves (uncommon):
```
──── 🐜 Spawning {N} workers ({X} 🔨 Builder, {Y} 🔍 Scout) ────
```

For a single worker:
```
──── 🔨🐜 Spawning {ant_name} — {task_summary} ────
```

### Step 5.1.1: Ambassador External Integration (Conditional Caste Replacement)

**Check if any Wave 1 tasks involve external integration:**

For each task in Wave 1, examine the task description and constraints for external integration keywords (case-insensitive):
- "API", "SDK", "OAuth", "external service", "integration", "webhook", "third-party", "stripe", "sendgrid", "twilio", "openai", "aws", "azure", "gcp"

Run using the Bash tool with description "Checking for external integration tasks...":
```bash
# Check phase name and task descriptions for external integration keywords
phase_name="{phase_name_from_state}"
task_descriptions="{concatenated task descriptions from Wave 1}"

integration_keywords="api sdk oauth external integration webhook third-party stripe sendgrid twilio openai aws azure gcp"
is_integration_phase="false"

for keyword in $integration_keywords; do
  if [[ "${phase_name,,}" == *"$keyword"* ]] || [[ "${task_descriptions,,}" == *"$keyword"* ]]; then
    is_integration_phase="true"
    matched_keyword="$keyword"
    break
  fi
done

echo "{\"is_integration_phase\": \"$is_integration_phase\", \"matched_keyword\": \"$matched_keyword\"}"
```

Parse the JSON result. If `is_integration_phase` is `"false"`:
- Skip to standard Builder spawning (continue with existing Step 5.1 logic)

If `is_integration_phase` is `"true"`:

1. **Generate Ambassador name and dispatch:**
   Run using the Bash tool with description "Naming ambassador...": `aether generate-ant-name "ambassador"` (store as `{ambassador_name}`)
   Run using the Bash tool with description "Dispatching ambassador...": `aether spawn-log --parent "Queen" --caste "ambassador" --name "{ambassador_name}" --task "External integration design" --depth 0`

   Display:
   ```
   ━━━ 🔌🐜 A M B A S S A D O R ━━━
   ──── 🔌🐜 Spawning {ambassador_name} — external integration design ────
   ```

2. **Spawn Ambassador using Task tool:**
   > **Platform note**: In Claude Code, use `Task tool with subagent_type`. In OpenCode, use the equivalent agent spawning mechanism for your platform (e.g., invoke the agent definition from `.opencode/agents/`).

   Spawn the Ambassador using Task tool with `subagent_type="aether-ambassador"`, include `description: "🔌 Ambassador {Ambassador-Name}: External integration design"` (DO NOT use run_in_background - task blocks until complete):

   # FALLBACK: If "Agent type not found", use general-purpose and inject role: "You are an Ambassador Ant - integration specialist that designs external API connections."

   **Ambassador Worker Prompt (CLEAN OUTPUT):**
   ```
   You are {Ambassador-Name}, a 🔌 Ambassador Ant.

   Mission: Design external integration for Phase {id}

   Phase: {phase_name}
   Trigger keyword: {matched_keyword}

   Task context:
   - Task descriptions: {Wave 1 task descriptions}
   - Files to be created/modified: {from task files}

   Work:
   1. Research the external service/API requirements
   2. Design integration pattern (Client Wrapper, Circuit Breaker, Retry with Backoff)
   3. Plan authentication method (OAuth, API keys, tokens)
   4. Design rate limiting handling
   5. Plan error scenarios (timeout, auth failure, rate limit)
   6. Document required environment variables
   7. Create integration plan for Builder execution

   **Integration Patterns to Consider:**
   - Client Wrapper: Abstract API complexity
   - Circuit Breaker: Handle service failures
   - Retry with Backoff: Handle transient errors
   - Caching: Reduce API calls
   - Queue Integration: Async processing

   **Security Requirements:**
   - API keys must use environment variables
   - No secrets in tracked files
   - HTTPS only
   - Validate SSL certificates

   Log activity: aether activity-log --command "RESEARCH" --details "{Ambassador-Name}: description"

   Return ONLY this JSON (no other text):
   {
     "ant_name": "{Ambassador-Name}",
     "caste": "ambassador",
     "status": "completed" | "failed" | "blocked",
     "summary": "Integration design summary",
     "integration_plan": {
       "service_name": "...",
       "authentication_method": "OAuth|API Key|Token",
       "env_vars_required": ["API_KEY", "..."],
       "integration_pattern": "Client Wrapper|Circuit Breaker|...",
       "rate_limit_handling": "...",
       "error_scenarios_covered": ["timeout", "auth_failure", "rate_limit"],
       "files_to_create": ["..."],
       "implementation_steps": ["..."]
     },
     "endpoints_integrated": [],
     "rate_limits_handled": true,
     "documentation_pages": 0,
     "blockers": []
   }
   ```

3. **Parse Ambassador JSON output:**
   Extract from response: `integration_plan`, `env_vars_required`, `error_scenarios_covered`, `blockers`

   Log completion:
   Run using the Bash tool with description "Recording ambassador completion...": `aether spawn-complete --name "{ambassador_name}" --status "completed" --summary "Integration design complete"`

   **Display Ambassador completion line:**
   ```
   🔌 {Ambassador-Name}: Integration design ({integration_plan.service_name}) ✓
   ```

4. **Log integration plan to midden:**
   Run using the Bash tool with description "Logging integration plan...":
   ```bash
   aether midden-write --category "integration" --message "Plan for {integration_plan.service_name}: {integration_plan.integration_pattern} pattern, auth via {integration_plan.authentication_method}" --source "ambassador"
   ```

   For each env var required:
   ```bash
   aether midden-write --category "integration" --message "Required env var: {env_var}" --source "ambassador"
   ```

5. **Display integration summary:**
   ```
   🔌 Ambassador complete — Integration plan ready for {integration_plan.service_name}

   Authentication: {integration_plan.authentication_method}
   Pattern: {integration_plan.integration_pattern}
   Env vars: {integration_plan.env_vars_required | join: ", "}

   Builder will execute this plan in Wave 1.
   ```

6. **Store integration plan for Builder injection:**
   Store the `integration_plan` object to be injected into Builder prompts in the standard Wave 1 spawn.

**First, mark build start in context:**
Run using the Bash tool with description "Marking build start...": `aether context-update build-start {phase_id} {wave_1_worker_count} {wave_1_task_count}`

Before dispatching each worker, refresh colony context so new pheromones/memory are visible:
Run using the Bash tool with description "Refreshing colony context...": `prime_result=$(aether colony-prime --compact 2>/dev/null)` and update `prompt_section` from `prime_result.result.prompt_section`.

**PER WAVE:** Query midden for recent failures to inject into builder context:
Run using the Bash tool with description "Checking midden for recent failures...":
`midden_result=$(aether midden-recent-failures 3 2>/dev/null || echo '{"count":0,"failures":[]}')`

Parse `midden_result`. If `count > 0`, format as `midden_context`:
```
**Previous Failures (from colony midden):**
- [{category}] {message} (source: {source}, {timestamp})
...
```

**Budget cap:** `midden_context` must not exceed 2000 characters. If it exceeds the cap, truncate and append `[midden truncated]`.

If `count == 0`, set `midden_context` to empty.

> **Platform note**: In Claude Code, use `Task tool with subagent_type`. In OpenCode, use the equivalent agent spawning mechanism for your platform (e.g., invoke the agent definition from `.opencode/agents/`).

For each Wave 1 task, use Task tool with `subagent_type="aether-builder"`, include `description: "🔨 Builder {Ant-Name}: {task_description}"` (DO NOT use run_in_background - multiple Task calls in a single message run in parallel and block until complete):

**PER WORKER:** Build graveyard caution context automatically:
- Identify explicit repo file paths from the task metadata (`files`, `hints`, `constraints`, and description when a concrete path is present).
- For each identified file path, run using the Bash tool with description "Checking graveyard cautions for {file}...":
  `aether grave-check "{file}"`
- Parse each JSON result and keep only entries where `caution_level` is `high` or `low`.
- Merge these into a single `grave_context` block for that worker.
- **Budget cap:** `grave_context` must not exceed 2000 characters per worker. If it exceeds the cap, truncate and append `[graveyard truncated]`.
- If no file paths are identified, or all checks return `none`, set `grave_context` to empty.
- If `grave_context` is non-empty, display a visible line before spawning that worker:
  `⚰️ Graveyard caution for {ant_name}: {file_1} ({level_1}), {file_2} ({level_2})`

**PER WORKER:** Inject worktree context (if allocated):
- Check `worktree_allocations[{ant_name}]` (from Step 5.0.7)
- If the allocation exists and is not null (i.e., `path`, `branch`, `id` are present), set `worktree_context` to:
  ```
  **Worktree Assignment:**
  You are working in an isolated git worktree. Your changes will NOT affect the main working tree.

  - Worktree path: {path}
  - Branch: {branch}
  - Worktree ID: {id}

  IMPORTANT rules for worktree operation:
  1. Before starting work, cd into the worktree path: cd {path}
  2. All file reads, edits, and writes must use absolute paths within this worktree
  3. Run tests from within the worktree path (cd {path} && {test_command})
  4. Commit your changes in the worktree (do NOT commit to main repo)
  5. Do NOT modify files outside your worktree path
  6. Your working directory for all Bash commands must be {path}
  ```
- If the allocation is null, missing, or empty, set `worktree_context` to empty (no injection needed).
- **Budget cap:** `worktree_context` must not exceed 1000 characters per worker.

**PER WORKER:** Match and inject skills for the worker's role and task:
Run using the Bash tool with description "Matching skills for {ant_name}...":
```bash
skill_match_result=$(aether skill-match "builder" "{task_description}" 2>/dev/null) || skill_match_result='{"result":{"colony_skills":[],"domain_skills":[]}}'
skill_inject_result=$(aether skill-inject "$(printf '%s\n' "$skill_match_result" | jq -r '.result')" 2>/dev/null) || skill_inject_result='{"result":{"skill_section":"","colony_count":0,"domain_count":0}}'
skill_section=$(printf '%s\n' "$skill_inject_result" | jq -r '.result.skill_section // ""')
skill_colony_count=$(printf '%s\n' "$skill_inject_result" | jq -r '.result.colony_count // 0')
skill_domain_count=$(printf '%s\n' "$skill_inject_result" | jq -r '.result.domain_count // 0')
```

Display per worker:
```
  🧠 Skills: {colony_count} colony + {domain_count} domain loaded for builder
```

**PER WORKER:** Run using the Bash tool with description "Preparing worker {name}...": `aether spawn-log --parent "Queen" --caste "builder" --name "{ant_name}" --task "{task_description}" --depth 0 && aether context-update worker-spawn "{ant_name}" "builder" "{task_description}"`

**Context layer budget caps (enforce before injecting into prompt):**
- `archaeology_context`: cap at 4000 characters. If it exceeds the cap, truncate and append `[archaeology truncated]`.
- `midden_context`: cap at 2000 characters (already enforced above).
- `grave_context`: cap at 2000 characters per worker (already enforced above).

**Builder Worker Prompt (CLEAN OUTPUT):**
```
You are {Ant-Name}, a 🔨🐜 Builder Ant.

{ worktree_context if exists }

Task {id}: {description}

Goal: "{colony_goal}"

{ archaeology_context if exists }

{ integration_plan if exists }

{ research_context if exists }

**Phase Research Context (if provided):**
- This is domain research conducted during planning. Use it to understand patterns, avoid gotchas, and follow the recommended approach.
- If the research mentions specific files to study, read them before implementing.

{ grave_context if exists }

{ midden_context if exists }

**Midden Context (if provided):**
- These are previous failures from this colony. Avoid repeating these patterns.
- If a failure is related to your task, take extra care or try a different approach.

**External Integration Context (if provided by Ambassador):**
If integration_plan is provided above, you MUST:
1. Follow the implementation_steps in order
2. Use the specified authentication_method
3. Implement the integration_pattern as designed
4. Handle all error_scenarios_covered
5. Reference required env_vars_required (do NOT hardcode values)

{ prompt_section }

{ skill_section }

**Graveyard Caution Context (if provided):**
- Treat `high` caution files as unstable terrain.
- Preserve proven behavior first, then make minimal safe edits.
- Add tests around any high-caution file before broader refactors.

**IMPORTANT:** When using the Bash tool for activity calls, always include a description parameter:
- activity-log calls → "Logging {action}..."
- pheromone-read calls → "Checking colony signals..."
- spawn-can-spawn calls → "Checking spawn budget..."
- generate-ant-name calls → "Naming sub-worker..."
- spawn-log calls → "Dispatching sub-worker..."

Use colony-flavored language, 4-8 words, trailing ellipsis.

Work:
1. Read .aether/workers.md for Builder discipline
2. Implement task, write tests
3. Log activity using Bash tool with description
4. At natural breakpoints (between tasks, after errors): Check for new signals using Bash tool with description

**Approach Change Logging:**
If you try an approach that doesn't work and switch to a different approach, log it:
```bash
colony_name=$(aether colony-name 2>/dev/null | jq -r '.result.name // ""')
[[ -z "$colony_name" ]] && colony_name="unknown"
phase_num=$(jq -r '.phase.number // "unknown"' .aether/data/COLONY_STATE.json 2>/dev/null || echo "unknown")

cat >> .aether/midden/approach-changes.md << EOF
- timestamp: "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
  phase: ${phase_num}
  colony: "${colony_name}"
  worker: "{Ant-Name}"
  task: "{task_id}"
  tried: "initial approach that failed"
  why_it_failed: "reason it didn't work"
  switched_to: "new approach that worked"
EOF

# Write to structured midden for threshold detection (MID-02)
aether midden-write --category "abandoned-approach" --message "Tried: initial approach that failed. Switched to: new approach. Reason: reason it didn't work" --source "builder" 2>/dev/null || true

# Enter memory pipeline for learning observation tracking (MID-02)
aether memory-capture \
  --type "failure" \
  --content "Approach abandoned: initial approach that failed -> new approach (reason it didn't work)" 2>/dev/null || true
```

Spawn sub-workers ONLY if 3x complexity:
- Check spawn budget using Bash tool with description: `aether spawn-can-spawn {depth} --enforce`
- Generate name using Bash tool with description
- Announce: "🐜 Spawning {child_name} for {reason}"
- Log spawn using Bash tool with description

Count your total tool calls (Read + Grep + Edit + Bash + Write) and report as tool_count.

Return ONLY this JSON (no other text):
{"ant_name": "{Ant-Name}", "task_id": "{id}", "status": "code_written|failed|blocked", "summary": "What you did", "tool_count": 0, "files_created": [], "files_modified": [], "tests_written": [], "blockers": []}
```

### Step 5.2: Process Wave 1 Results

**Task calls return results directly (no TaskOutput needed).**

Before using any worker payload, validate schema:
Run using the Bash tool with description "Validating worker response...": `aether validate-worker-response builder '{worker_json}'`
If validation fails, treat the worker as failed with blocker `invalid_worker_response`.

**As each worker result arrives, IMMEDIATELY display a single completion line — do not wait for other workers:**

For successful workers:
```
🔨 {Ant-Name}: {task_description} ({tool_count} tools) ✓
```

For failed workers:
```
🔨 {Ant-Name}: {task_description} ✗ ({failure_reason} after {tool_count} tools)
```

Where `tool_count` comes from the worker's returned JSON `tool_count` field, and `failure_reason` is extracted from the first item in the worker's `blockers` array or "unknown error" if empty.

**Log failure to midden and record observation (MEM-02):**

After displaying a failed worker, run using the Bash tool with description "Logging failure to midden...":
```bash
colony_name=$(aether colony-name 2>/dev/null | jq -r '.result.name // ""')
[[ -z "$colony_name" ]] && colony_name="unknown"
phase_num=$(jq -r '.phase.number // "unknown"' .aether/data/COLONY_STATE.json 2>/dev/null || echo "unknown")

# Append to build-failures.md
cat >> .aether/midden/build-failures.md << EOF
- timestamp: "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
  phase: ${phase_num}
  colony: "${colony_name}"
  worker: "${ant_name}"
  task: "${task_id}"
  what_failed: "${blockers[0]:-$failure_reason}"
  why: "worker returned failed status"
  what_worked: null
  error_type: "worker_failure"
EOF

# Write to structured midden for threshold detection (MID-01)
aether midden-write --category "worker_failure" --message "Builder ${ant_name} failed on task ${task_id}: ${blockers[0]:-$failure_reason}" --source "builder" 2>/dev/null || true

# Capture failure in memory pipeline (observe + pheromone + auto-promotion)
aether memory-capture \
  --type "failure" \
  --content "Builder ${ant_name} failed on task ${task_id}: ${blockers[0]:-$failure_reason}" 2>/dev/null || true
```

**PER WORKER:** Run using the Bash tool with description "Recording {name} completion...": `aether spawn-complete --name "{ant_name}" --status "completed" --summary "{summary}" && aether context-update worker-complete "{ant_name}" "completed"`

**Check for total wave failure:**

After processing all worker results in this wave, check if EVERY worker returned `status: "failed"`. If ALL workers in the wave failed:

Display a prominent halt alert:
```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  ⚠ WAVE FAILURE — BUILD HALTED
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

All {N} workers in Wave {X} failed. Something is fundamentally wrong.

Failed workers:
  {for each failed worker in this wave:}
  {caste_emoji} {Ant-Name}: {task_description} ✗ ({failure_reason} after {tool_count} tools)
  {end for}

Next steps:
  /ant:flags      Review blockers
  /ant:swarm      Auto-repair mode
```

Then STOP — do not proceed to subsequent waves, Watcher, or Chaos. Skip directly to Step 5.9 synthesis with `status: "failed"`.

**Partial wave failure — escalation path:**

If SOME (but not all) workers in the wave failed:
1. For each failed worker, attempt Tier 3 escalation: Queen spawns a different caste for the same task
2. If Tier 3 succeeds: continue to next wave
3. If Tier 3 fails: display the Tier 4 ESCALATION banner (from Queen agent definition):

```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  ⚠ ESCALATION — QUEEN NEEDS YOU
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Task: {failed task description}
Phase: {phase number} — {phase name}

Tried:
  • Worker retry (2 attempts) — {what failed}
  • Parent tried alternate approach — {what failed}
  • Queen reassigned to {other caste} — {what failed}

Options:
  A) {recommended option} — RECOMMENDED
  B) {alternate option}
  C) Skip and continue — this task will be marked blocked

Awaiting your choice.
```

Log escalation as flag:
Run using the Bash tool with description "Logging escalation...": `aether flag-add "blocker" "{task title}" "{failure summary}" "escalation" {phase_number}`

If at least one worker succeeded, continue normally to the next wave.

**Parse each worker's validated JSON output to collect:** status, files_created, files_modified, blockers

**Intra-phase midden threshold check (MID-03):**

After processing all wave results, check if any midden error category has reached 3+ occurrences. If so, emit a REDIRECT pheromone mid-build to alert the colony.

Run using the Bash tool with description "Checking midden thresholds...":
```bash
midden_result=$(aether midden-recent-failures 50 2>/dev/null || echo '{"count":0,"failures":[]}')
midden_count=$(echo "$midden_result" | jq '.count // 0')

if [[ "$midden_count" -gt 0 ]]; then
  recurring_categories=$(echo "$midden_result" | jq -r '
    [.failures[] | .category]
    | group_by(.)
    | map(select(length >= 3))
    | map({category: .[0], count: length})
    | .[]
    | @base64
  ' 2>/dev/null || echo "")

  redirect_emit_count=0
  for encoded in $recurring_categories; do
    [[ $redirect_emit_count -ge 3 ]] && break
    [[ -z "$encoded" ]] && continue
    category=$(echo "$encoded" | base64 -d | jq -r '.category')
    count=$(echo "$encoded" | base64 -d | jq -r '.count')

    existing=$(jq -r --arg cat "$category" '
      [.signals[] | select(.active == true and .source == "auto:error" and (.content.text | contains($cat)))] | length
    ' .aether/data/pheromones.json 2>/dev/null || echo "0")

    if [[ "$existing" == "0" ]]; then
      aether pheromone-write --type REDIRECT \
        --content "[error-pattern] Category \"$category\" recurring ($count occurrences)" \
        --strength 0.7 \
        --source "auto:error" \
        --reason "Auto-emitted: midden error pattern recurred 3+ times mid-build" \
        --ttl "30d" 2>/dev/null || true
      redirect_emit_count=$((redirect_emit_count + 1))
    fi
  done

  if [[ $redirect_emit_count -gt 0 ]]; then
    echo "Warning: Midden threshold triggered -- $redirect_emit_count REDIRECT pheromone(s) emitted mid-build"
  fi
fi
```

Display if any REDIRECT was emitted:
```
Warning: Midden threshold: "{category}" recurring ({count}x) -- REDIRECT emitted mid-build
```

### Step 5.2.5: Post-Wave Worktree Merge-Back

**This step runs after each wave's results are processed (Step 5.2). It merges worktree branches back to main so that subsequent waves build on the latest state.**

**This step is non-blocking: merge failures create blockers but do not halt the build.**

1. **Check if worktree_allocations is non-empty:**

If `worktree_allocations` is empty `{}` (in-repo mode or no allocations):
- Skip silently, proceed to Step 5.3.

If `worktree_allocations` has entries for workers in this wave:
- Display:
```
━━━ 🔀 W O R K T R E E   M E R G E - B A C K ━━━
```

2. **For each worker in the current wave that has a worktree allocation (non-null):**

Run using the Bash tool with description "Merging worktree for {ant_name}...":
```bash
aether worktree-merge-back --branch {branch}
```

Where `{branch}` comes from `worktree_allocations[{ant_name}].branch`.

3. **Process merge results:**

**If merge succeeds** (exit code 0, output contains merge confirmation):
- Display: `  🔀 {ant_name}: {branch} merged and worktree cleaned`
- Remove `{ant_name}` from `worktree_allocations` (set to null or delete)
- Store `last_merged_branch = {branch}` for downstream steps

**If merge fails** (non-zero exit or blocker created):
- Display: `  ⚠ {ant_name}: {branch} merge failed -- blocker created, worker task flagged`
- Mark the worker's task as having a `merge-blocker` in the task status map
- The worktree remains in place for manual investigation
- Do NOT remove from `worktree_allocations` (the worktree still exists)

4. **Handle partial success:**

If some merges succeed and some fail in the same wave:
- Display summary:
```
Merge-back: {success_count} merged, {fail_count} blocked
```
- Proceed to the next wave regardless -- subsequent waves will work on main which has the successfully-merged changes.
- Failed worktrees remain available for manual merge or retry via `/ant:swarm`.

5. **Log merge-back activity:**

Run using the Bash tool with description "Logging merge-back activity...":
```bash
aether activity-log --command "MERGE_BACK" --details "Queen: Wave {wave_number}: {success_count} merged, {fail_count} blocked"
```

> **Design note:** This step runs after EVERY wave (not just the last), ensuring that Wave 2+ workers always build on top of Wave 1's merged results. This prevents orphaned worktree branches and keeps the main branch up-to-date throughout the build.

### Step 5.3: Spawn Wave 2+ Workers (Sequential Waves)

**Before each subsequent wave, display a wave separator:**
```
━━━ 🐜 Wave {X} of {N} ━━━
```

**Run Step 5.0.7 (Pre-Wave Worktree Allocation) for each subsequent wave.** Only workers not already in `worktree_allocations` need allocation; skip workers that already have an assigned worktree from a prior wave.

Then display the spawn announcement (same format as Step 5.1).

Repeat Step 5.0.7 + Step 5.1-5.2 + Step 5.2.5 for each subsequent wave, waiting for previous wave to complete.

### Step 5.3.5: Builder-Probe Lock (MANDATORY — All Waves)

**This step runs after ALL waves have completed. No task may be marked `completed` without independent Probe verification.**

**The Builder-Probe Lock enforces that builders cannot self-certify completion.** Builders return `code_written`; only Probe can verify the work, and only the Queen can mark tasks complete.

#### 5.3.5.1: Collect Code-Written Tasks

After all waves finish, identify all workers that returned `status: "code_written"`:

```
━━━ 🔬 B U I L D E R - P R O B E   L O C K ━━━
```

For each builder that returned `code_written`:
- Add to the probe verification queue
- Display: `  ⏳ {Ant-Name}: Task {id} — awaiting Probe verification`

If no builders returned `code_written` (all failed or blocked), skip to Step 5.9 synthesis.

If any builder returned `status: "completed"` instead of `code_written`:
- Display a warning: `  ⚠ {Ant-Name}: Returned "completed" instead of "code_written" — this violates the Builder-Probe Lock. Requiring Probe verification before acceptance.`
- Treat as `code_written` and add to the probe verification queue (do not reject — the lock is enforced by the Queen, not by the builder).

#### 5.3.5.2: Spawn Probe for Independent Verification

For each task in the probe verification queue, spawn a Probe agent:

1. **Generate Probe name:**
   Run using the Bash tool with description "Naming probe ant...": `aether generate-ant-name "probe"` (store as `{probe_name}`)

2. **Log spawn:**
   Run using the Bash tool with description "Dispatching probe...": `aether spawn-log --parent "Queen" --caste "probe" --name "{probe_name}" --task "Verify task {task_id}" --depth 0`

3. **Display announcement:**
   ```
   ──── 🔬 Spawning {probe_name} — Verify {Ant-Name}'s work on Task {task_id} ────
   ```

4. **Spawn Probe using Task tool with `subagent_type="aether-probe"`**, include `description: "Probe {probe_name}: Verify task {task_id}"` (DO NOT use run_in_background):

   **Probe Worker Prompt:**
   ```
   You are {probe_name}, a 🔬 Probe Ant.

   Mission: Independently verify the work of Builder {Ant-Name} on Task {task_id}

   Task: {task_description}
   Builder's summary: {builder_summary}
   Files claimed: {files_created + files_modified from builder output}
   Tests claimed: {tests_written from builder output}

   Work:
   1. Run the project test command and verify all tests pass
   2. Check that claimed files exist and contain valid implementations
   3. Verify test coverage meets 80% threshold for new code
   4. Confirm the deliverable matches the task specification
   5. Check for obvious issues: hardcoded values, missing error handling, dead code

   Return ONLY this JSON (no other text):
   {
     "ant_name": "{probe_name}",
     "caste": "probe",
     "task_id": "{task_id}",
     "verified_builder": "{Ant-Name}",
     "status": "passed" | "failed",
     "verdict": "Brief explanation of verification result",
     "test_results": {"command": "...", "passed": N, "failed": N, "total": N},
     "coverage": {"percent": N, "meets_threshold": true|false},
     "issues_found": ["issue1", "issue2"],
     "files_checked": ["path1", "path2"],
     "blockers": []
   }
   ```

5. **Parse Probe JSON output:**

   For each probe result:

   **If Probe status is `"passed"`:**
   - Run using the Bash tool with description "Marking task complete with guard...":
     ```bash
     aether state-mutate --guard "task-complete:{task_id}" 2>/dev/null
     ```
   - If the guard succeeds (exit code 0):
     - Display: `  ✅ {probe_name}: Task {task_id} PASSED — marked complete via guard`
     - The task is now `completed`
   - If the guard fails (non-zero exit):
     - Display: `  ⚠ {probe_name}: Task {task_id} PASSED verification but guard failed — task remains in code_written state`
     - The task stays `code_written`; log for investigation

   **If Probe status is `"failed"`:**
   - Display: `  ✗ {probe_name}: Task {task_id} FAILED verification — {verdict}`
   - List issues found from `issues_found` array
   - The task stays in `code_written` state with a probe failure recorded

   Log all probe results:
   Run using the Bash tool with description "Recording probe completion...":
   ```bash
   aether spawn-complete --name "{probe_name}" --status "{probe_status}" --summary "{verdict}" && aether context-update worker-complete "{probe_name}" "probe"
   ```

#### 5.3.5.3: Handle Probe Failures

After all probes have completed, check for tasks that failed verification:

**If ALL tasks passed Probe verification:**
- Display: `All {N} tasks verified by Probe — Builder-Probe Lock satisfied`
- Proceed to Step 5.9 synthesis

**If SOME tasks failed Probe verification:**

Display:
```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  ⚠ PROBE VERIFICATION FAILURES
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

{N} of {M} tasks failed independent verification:

  ✗ Task {id} ({Ant-Name}): {verdict}
    Issues: {issues_found, comma-separated}

Options:
  A) Re-run failed tasks — send Builders back to fix issues (RECOMMENDED)
  B) Accept partial — mark passed tasks complete, failed tasks as blocked
  C) Halt build — all tasks must pass before proceeding
```

Use AskUserQuestion to get the user's choice.

- **If A (re-run):** Re-spawn builders for failed tasks only (same as Wave retry logic in Step 5.2 partial failure handling). After re-run, re-run Probe on the fixed tasks.
- **If B (accept partial):** Mark passed tasks as `completed` via guard. Mark failed tasks as `blocked` with `"probe_verification_failed"` as the blocker reason. Proceed to Step 5.9.
- **If C (halt):** Skip to Step 5.9 with `status: "blocked"` and `"probe_verification_failed"` in the summary.

#### 5.3.5.4: Update Task Status Map

After probe resolution, build the final task status map for synthesis:

For each task:
- `code_written` + Probe passed + Guard succeeded → `completed`
- `code_written` + Probe passed + Guard failed → `code_written` (flagged)
- `code_written` + Probe failed + User chose partial → `blocked`
- `failed` (builder) → `failed`
- `blocked` (builder) → `blocked`

Store this map for use in Step 5.9 synthesis.

**Log probe lock outcome:**
Run using the Bash tool with description "Logging probe lock outcome...":
```bash
aether activity-log --command "PROBE_LOCK" --details "Queen: Builder-Probe Lock: {passed_count} passed, {failed_count} failed, {total_count} total"
```
