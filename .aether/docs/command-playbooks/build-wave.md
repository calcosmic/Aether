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
   - Research/docs tasks → 🔍🐜 Scout
   - Testing/validation → 👁️🐜 Watcher (ALWAYS spawn at least one)
   - Resilience testing → 🎲🐜 Chaos (ALWAYS spawn one after Watcher)

3. **Generate ant names for each worker:**

Run using the Bash tool with description "Naming builder ant...": `bash .aether/aether-utils.sh generate-ant-name "builder"`
Run using the Bash tool with description "Naming watcher ant...": `bash .aether/aether-utils.sh generate-ant-name "watcher"`
Run using the Bash tool with description "Naming chaos ant...": `bash .aether/aether-utils.sh generate-ant-name "chaos"`

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
  🎲🐜 {Chaos-Name}   Resilience testing (after Watcher)

Total: {N} Builders + 1 Watcher + 1 Chaos = {N+2} spawns
```

**Caste Emoji Legend:**
- 🔨🐜 Builder  (cyan if color enabled)
- 👁️🐜 Watcher  (green if color enabled)
- 🎲🐜 Chaos    (red if color enabled)
- 🔍🐜 Scout    (yellow if color enabled)
- 🏺🐜 Archaeologist (magenta if color enabled)
- 🥚 Queen/Prime

**Every spawn must show its caste emoji.**

**Add to Caste Emoji Legend:**
- 🔌🐜 Ambassador (blue if color enabled) — external integration specialist

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
   Run using the Bash tool with description "Naming ambassador...": `bash .aether/aether-utils.sh generate-ant-name "ambassador"` (store as `{ambassador_name}`)
   Run using the Bash tool with description "Dispatching ambassador...": `bash .aether/aether-utils.sh spawn-log "Queen" "ambassador" "{ambassador_name}" "External integration design"`

   Display:
   ```
   ━━━ 🔌🐜 A M B A S S A D O R ━━━
   ──── 🔌🐜 Spawning {ambassador_name} — external integration design ────
   🔌 Ambassador {ambassador_name} spawning — Designing integration for {matched_keyword}...
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

   Log activity: bash .aether/aether-utils.sh activity-log "RESEARCH" "{Ambassador-Name}" "description"

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
   Run using the Bash tool with description "Recording ambassador completion...": `bash .aether/aether-utils.sh spawn-complete "{ambassador_name}" "completed" "Integration design complete"`

   **Display Ambassador completion line:**
   ```
   🔌 {Ambassador-Name}: Integration design ({integration_plan.service_name}) ✓
   ```

4. **Log integration plan to midden:**
   Run using the Bash tool with description "Logging integration plan...":
   ```bash
   bash .aether/aether-utils.sh midden-write "integration" "Plan for {integration_plan.service_name}: {integration_plan.integration_pattern} pattern, auth via {integration_plan.authentication_method}" "ambassador"
   ```

   For each env var required:
   ```bash
   bash .aether/aether-utils.sh midden-write "integration" "Required env var: {env_var}" "ambassador"
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
Run using the Bash tool with description "Marking build start...": `bash .aether/aether-utils.sh context-update build-start {phase_id} {wave_1_worker_count} {wave_1_task_count}`

Before dispatching each worker, refresh colony context so new pheromones/memory are visible:
Run using the Bash tool with description "Refreshing colony context...": `prime_result=$(bash .aether/aether-utils.sh colony-prime --compact 2>/dev/null)` and update `prompt_section` from `prime_result.result.prompt_section`.

> **Platform note**: In Claude Code, use `Task tool with subagent_type`. In OpenCode, use the equivalent agent spawning mechanism for your platform (e.g., invoke the agent definition from `.opencode/agents/`).

For each Wave 1 task, use Task tool with `subagent_type="aether-builder"`, include `description: "🔨 Builder {Ant-Name}: {task_description}"` (DO NOT use run_in_background - multiple Task calls in a single message run in parallel and block until complete):

**PER WORKER:** Build graveyard caution context automatically:
- Identify explicit repo file paths from the task metadata (`files`, `hints`, `constraints`, and description when a concrete path is present).
- For each identified file path, run using the Bash tool with description "Checking graveyard cautions for {file}...":
  `bash .aether/aether-utils.sh grave-check "{file}"`
- Parse each JSON result and keep only entries where `caution_level` is `high` or `low`.
- Merge these into a single `grave_context` block for that worker.
- If no file paths are identified, or all checks return `none`, set `grave_context` to empty.
- If `grave_context` is non-empty, display a visible line before spawning that worker:
  `⚰️ Graveyard caution for {ant_name}: {file_1} ({level_1}), {file_2} ({level_2})`

**PER WORKER:** Run using the Bash tool with description "Preparing worker {name}...": `bash .aether/aether-utils.sh spawn-log "Queen" "builder" "{ant_name}" "{task_description}" && bash .aether/aether-utils.sh context-update worker-spawn "{ant_name}" "builder" "{task_description}"`

**Builder Worker Prompt (CLEAN OUTPUT):**
```
You are {Ant-Name}, a 🔨🐜 Builder Ant.

Task {id}: {description}

Goal: "{colony_goal}"

{ archaeology_context if exists }

{ integration_plan if exists }

{ grave_context if exists }

**External Integration Context (if provided by Ambassador):**
If integration_plan is provided above, you MUST:
1. Follow the implementation_steps in order
2. Use the specified authentication_method
3. Implement the integration_pattern as designed
4. Handle all error_scenarios_covered
5. Reference required env_vars_required (do NOT hardcode values)

{ prompt_section }

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
colony_name=$(jq -r '.session_id | split("_")[1] // "unknown"' .aether/data/COLONY_STATE.json 2>/dev/null || echo "unknown")
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
```

Spawn sub-workers ONLY if 3x complexity:
- Check spawn budget using Bash tool with description: `bash .aether/aether-utils.sh spawn-can-spawn {depth} --enforce`
- Generate name using Bash tool with description
- Announce: "🐜 Spawning {child_name} for {reason}"
- Log spawn using Bash tool with description

Count your total tool calls (Read + Grep + Edit + Bash + Write) and report as tool_count.

Return ONLY this JSON (no other text):
{"ant_name": "{Ant-Name}", "task_id": "{id}", "status": "completed|failed|blocked", "summary": "What you did", "tool_count": 0, "files_created": [], "files_modified": [], "tests_written": [], "blockers": []}
```

### Step 5.2: Process Wave 1 Results

**Task calls return results directly (no TaskOutput needed).**

Before using any worker payload, validate schema:
Run using the Bash tool with description "Validating worker response...": `bash .aether/aether-utils.sh validate-worker-response builder '{worker_json}'`
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
colony_name=$(jq -r '.session_id | split("_")[1] // "unknown"' .aether/data/COLONY_STATE.json 2>/dev/null || echo "unknown")
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

# Capture failure in memory pipeline (observe + pheromone + auto-promotion)
bash .aether/aether-utils.sh memory-capture \
  "failure" \
  "Builder ${ant_name} failed on task ${task_id}: ${blockers[0]:-$failure_reason}" \
  "failure" \
  "worker:builder" 2>/dev/null || true
```

**PER WORKER:** Run using the Bash tool with description "Recording {name} completion...": `bash .aether/aether-utils.sh spawn-complete "{ant_name}" "completed" "{summary}" && bash .aether/aether-utils.sh context-update worker-complete "{ant_name}" "completed"`

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
Run using the Bash tool with description "Logging escalation...": `bash .aether/aether-utils.sh flag-add "blocker" "{task title}" "{failure summary}" "escalation" {phase_number}`

If at least one worker succeeded, continue normally to the next wave.

**Parse each worker's validated JSON output to collect:** status, files_created, files_modified, blockers

### Step 5.3: Spawn Wave 2+ Workers (Sequential Waves)

**Before each subsequent wave, display a wave separator:**
```
━━━ 🐜 Wave {X} of {N} ━━━
```
Then display the spawn announcement (same format as Step 5.1).

Repeat Step 5.1-5.2 for each subsequent wave, waiting for previous wave to complete.
