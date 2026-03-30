<!-- Generated from .aether/commands/plan.yaml - DO NOT EDIT DIRECTLY -->
---
name: ant:plan
description: "📊🐜🗺️🐜📊 Show project plan or generate project-specific phases"
---

### Step -1: Normalize Arguments

Run: `normalized_args=$(bash .aether/aether-utils.sh normalize-args "$@")`

This ensures arguments work correctly in both Claude Code and OpenCode. Use `$normalized_args` throughout this command.


You are the **Queen**. Orchestrate research and planning until the selected confidence target is reached within the selected iteration budget.

## Instructions

Parse `$normalized_args`:
- If contains `--no-visual`: set `visual_mode = false` (visual is ON by default)
- Otherwise: set `visual_mode = true`

### Step 1: Read State

Read `.aether/data/COLONY_STATE.json`.

**Auto-upgrade old state:**
If `version` field is missing, "1.0", or "2.0":
1. Preserve: `goal`, `state`, `current_phase`, `plan.phases`
2. Write upgraded v3.0 state (same structure as /ant:init but preserving data)
3. Output: `State auto-upgraded to v3.0`
4. Continue with command.

Extract: `goal`, `milestone`, `plan.phases`

**Validate:** If `goal: null`:
```
No colony initialized. Run /ant:init "<goal>" first.
```
Stop here.

If `milestone` == `"Crowned Anthill"`:
```
This colony has been sealed. Start a new colony with /ant:init "new goal".
```
Stop here.

### Step 1.5: Load State and Show Resumption Context

Run using Bash tool: `bash .aether/aether-utils.sh load-state`

If successful and goal is not null:
1. Extract current_phase from state
2. Get phase name from plan.phases[current_phase - 1].name (or "(unnamed)")
3. Display brief resumption context:
   ```
   🔄 Resuming: Phase X - Name
   ```

If .aether/HANDOFF.md exists (detected in load-state output):
- Display "Resuming from paused session"
- Read .aether/HANDOFF.md for additional context
- Remove .aether/HANDOFF.md after display (cleanup)

Run: `bash .aether/aether-utils.sh unload-state` to release lock.

**Error handling:**
- If E_FILE_NOT_FOUND: "No colony initialized. Run /ant:init first." and stop
- If validation error: Display error details with recovery suggestion and stop
- For other errors: Display generic error and suggest /ant:status for diagnostics

### Step 2: Check Existing Plan

If `plan.phases` has entries (non-empty array), skip to **Step 6** (Display Plan).

Parse `$normalized_args`:
- If contains `--accept`: Set `force_accept = true` (accept current plan regardless of confidence)
- Otherwise: `force_accept = false`

Select planning depth (prompt user if not explicitly provided):
- Presets:
  - `fast`: `target_confidence = 80`, `max_iterations = 4`
  - `balanced`: `target_confidence = 90`, `max_iterations = 6`
  - `deep`: `target_confidence = 95`, `max_iterations = 8`
  - `exhaustive`: `target_confidence = 99`, `max_iterations = 12`
- Preset selectors:
  - `--fast`, `--balanced`, `--deep`, `--exhaustive`
  - `--quality fast|balanced|deep|exhaustive`
- CLI overrides:
  - `--target <70-99>` to set `target_confidence`
  - `--max-iterations <2-12>` to set `max_iterations`
- If no preset/overrides are provided, ask:
  `Planning depth? 1) Fast 2) Balanced 3) Deep (recommended) 4) Exhaustive`
- Map user choice to a preset, default to `deep` on unclear input.
- If overrides are out of range, clamp to valid ranges and continue.

### Step 2.5: Load Compact Context Capsule

Run using the Bash tool with description "Loading compact planning context...":
```bash
bash .aether/aether-utils.sh context-capsule --compact --json 2>/dev/null
```

If JSON is valid and `.ok == true`, extract `.result.prompt_section` into `context_capsule_prompt`.
If command fails or returns invalid JSON, set `context_capsule_prompt = ""` and continue.

### Step 3: Initialize Planning State

Update watch files for tmux visibility:

Write `.aether/data/watch-status.txt`:
```
AETHER COLONY :: PLANNING
==========================

State: PLANNING
Phase: 0/0 (generating plan)
Confidence: 0%
Iteration: 0/{max_iterations}

Active Workers:
  [Research] Starting...
  [Planning] Waiting...

Last Activity:
  Planning loop initiated
```

Write `.aether/data/watch-progress.txt`:
```
Progress
========

[                    ] 0%

Target: {target_confidence}% confidence

Iteration: 0/{max_iterations}
Gaps: (analyzing...)
```

### Step 3.5: Load Territory Survey

Check if territory survey exists before research:

```bash
ls .aether/data/survey/*.md 2>/dev/null
```

**If survey exists:**
1. **Always read PATHOGENS.md first** — understand known concerns before planning
2. Read other relevant docs based on goal keywords:

| Goal Contains | Additional Documents |
|---------------|---------------------|
| UI, frontend, component, page | DISCIPLINES.md, CHAMBERS.md |
| API, backend, endpoint | BLUEPRINT.md, DISCIPLINES.md |
| database, schema, model | BLUEPRINT.md, PROVISIONS.md |
| test, spec | SENTINEL-PROTOCOLS.md, DISCIPLINES.md |
| integration, external | TRAILS.md, PROVISIONS.md |
| refactor, cleanup | PATHOGENS.md, BLUEPRINT.md |

**Inject survey context into scout and planner prompts:**
- Include key patterns from DISCIPLINES.md
- Reference architecture from BLUEPRINT.md
- Note tech stack from PROVISIONS.md
- Flag concerns from PATHOGENS.md

**Display:**
```
🗺️ Territory survey loaded — incorporating context into planning
```

**If no survey:** Continue without survey context (scouts will do fresh exploration)

### Step 3.6: Phase Domain Research

Investigate domain knowledge for each phase before the planning loop begins. This runs every time `/ant:plan` generates a new plan -- no skip flag.

**1. Retrieve hive wisdom for research priming:**

```bash
hive_context=$(bash .aether/aether-utils.sh hive-read --limit 5 --format text 2>/dev/null)
```

Parse the JSON result to extract `.result.text` as `hive_text`. If command fails or returns empty, set `hive_text = ""`.

**2. Clean any previous research for this phase:**

```bash
research_dir=".aether/data/phase-research"
mkdir -p "$research_dir"
rm -f "$research_dir/phase-{phase_number}-research.md"
```

Re-running /ant:plan always re-researches from scratch.

**3. Spawn Research Scout** via Task tool with `subagent_type="aether-scout"`:

FALLBACK: If "Agent type not found", use general-purpose and inject role: "You are a Scout Ant performing Phase Domain Research."

```
You are a Scout Ant performing Phase Domain Research.

--- MISSION ---
Investigate the domain knowledge needed for Phase {phase_number}: {phase_name}
Goal: "{goal}"
Phase description: "{phase_description}"

{context_capsule_prompt}

--- PRE-EXISTING COLONY WISDOM ---
{hive_text}

--- RESEARCH AREAS ---
1. Key patterns in the existing codebase relevant to this phase
2. External library/API documentation if the phase involves external tools
3. Common gotchas and pitfalls in this domain
4. Recommended implementation approach based on findings

--- SCOPE CONSTRAINTS ---
- Maximum 5 key patterns
- Maximum 3 gotchas
- Maximum 1 recommended approach paragraph
- Total output under 3000 words
- Prioritize actionable guidance over exhaustive documentation
- Check hive wisdom above first -- do not re-discover known patterns

--- TOOLS ---
Use: Glob, Grep, Read, WebSearch, WebFetch
Do NOT use: Task, Write, Edit

--- OUTPUT FORMAT ---
Return JSON:
{
  "hive_wisdom_used": ["list of hive entries that were relevant"],
  "key_patterns": [
    {"pattern": "description", "source": "file path or URL", "relevance": "why it matters for this phase"}
  ],
  "external_context": [
    {"topic": "what", "finding": "description", "source": "URL or doc reference"}
  ],
  "gotchas": [
    {"issue": "what can go wrong", "prevention": "how to avoid it", "source": "evidence"}
  ],
  "recommended_approach": "synthesis paragraph",
  "files_to_study": ["path1", "path2"]
}
```

**4. Wait for scout to complete** (blocking -- direct Task return).

**5. Parse scout JSON output and write RESEARCH.md to disk.** The Queen (plan.md orchestrator) writes the file -- scout is read-only. Write to: `.aether/data/phase-research/phase-{phase_number}-research.md`

RESEARCH.md format (6 fixed sections):

```markdown
# Phase {N} Research: {Phase Name}

**Generated:** {ISO-8601 timestamp}
**Phase:** {N} - {Phase Name}
**Research scope:** {brief summary of what was investigated}

## Hive Wisdom (Pre-existing Knowledge)
{Format hive_wisdom_used entries, or "No relevant hive wisdom found" if empty}

## Key Patterns
{Format each key_patterns entry as: **{pattern}:** {relevance} (Source: {source})}

## External Context
{Format each external_context entry as: **{topic}:** {finding} (Source: {source})}
{If empty: "No external research needed for this phase"}

## Gotchas
{Format each gotchas entry as: **{issue}:** {prevention} (Source: {source})}

## Recommended Approach
{recommended_approach text}

## Files to Study
{Format as bullet list of file paths}
```

**6. Store research findings** in a variable `research_findings_summary` for injection into the Route-Setter prompt in Step 4. This is a compact summary (not the full RESEARCH.md):

```
Key Patterns: {bullet list of pattern names}
Gotchas: {bullet list of gotcha titles}
Recommended: {recommended_approach, first sentence only}
Files: {comma-separated file paths}
```

**7. Display completion:**

```
Research complete: phase-{phase_number}-research.md ({word_count} words)
```

### Step 4: Research and Planning Loop

Initialize tracking:
- `iteration = 0`
- `confidence = 0`
- `gaps = []` (list of knowledge gaps)
- `plan_draft = null`
- `last_confidence = 0`
- `stall_count = 0` (consecutive iterations with < 5% improvement)

**Loop (max {max_iterations} iterations, 2 agents per iteration: 1 scout + 1 planner):**

```
while iteration < max_iterations AND confidence < target_confidence:
    iteration += 1

    # === AUTO-BREAK CHECKS (no user prompt needed) ===
    if iteration > 1:
        if confidence >= target_confidence:
            Log: "Confidence threshold reached ({confidence}%), finalizing plan"
            break
        if stall_count >= 2:
            Log: "Planning stalled at {confidence}%, finalizing current plan"
            break

    # === RESEARCH PHASE (always runs — 1 scout per iteration) ===

    if iteration == 1:

        # Broad exploration on first pass
        Spawn Research Scout via Task tool with subagent_type="aether-scout":

        """
        You are a Scout Ant in the Aether Colony.

        --- MISSION ---
        Research the codebase to understand what exists and how it works.

        Goal: "{goal}"
        Iteration: {iteration}/{max_iterations}

        {context_capsule_prompt}

        --- EXPLORATION AREAS ---
        Cover ALL of these in a single pass:
        1. Core architecture, entry points, and main modules
        2. Business logic and domain models
        3. Testing patterns and quality practices
        4. Configuration, dependencies, and infrastructure
        5. Edge cases, error handling, and validation

        --- TOOLS ---
        Use: Glob, Grep, Read, WebSearch, WebFetch
        Do NOT use: Task, Write, Edit

        --- OUTPUT CONSTRAINTS ---
        Maximum 5 findings (prioritize by impact on the goal).
        Maximum 2 sentences per finding.
        Maximum 3 knowledge gaps identified.

        --- OUTPUT FORMAT ---
        Return JSON:
        {
          "findings": [
            {"area": "...", "discovery": "...", "source": "file or search"}
          ],
          "gaps_remaining": [
            {"id": "gap_N", "description": "..."}
          ],
          "overall_knowledge_confidence": 0-100
        }
        """

    else:

        # Gap-focused research on subsequent passes
        Spawn Gap-Focused Scout via Task tool with subagent_type="aether-scout":

        """
        You are a Scout Ant in the Aether Colony (gap-focused research).

        --- MISSION ---
        Investigate ONLY these specific knowledge gaps. Do not explore broadly.

        Goal: "{goal}"
        Iteration: {iteration}/{max_iterations}

        {context_capsule_prompt}

        --- GAPS TO INVESTIGATE ---
        {for each gap in gaps:}
          - {gap.id}: {gap.description}
        {end for}

        --- TOOLS ---
        Use: Glob, Grep, Read, WebSearch, WebFetch
        Do NOT use: Task, Write, Edit

        --- OUTPUT CONSTRAINTS ---
        Maximum 3 findings (one per gap investigated).
        Maximum 2 sentences per finding.
        Only report gaps that are STILL unresolved after your research.

        --- OUTPUT FORMAT ---
        Return JSON:
        {
          "findings": [
            {"area": "...", "discovery": "...", "source": "file or search"}
          ],
          "gaps_remaining": [
            {"id": "gap_N", "description": "..."}
          ],
          "gaps_resolved": ["gap_1", "gap_2"],
          "overall_knowledge_confidence": 0-100
        }
        """

    # Wait for scout to complete.
    # Update gaps list from scout results.

    # === PLANNING PHASE (always runs — 1 planner per iteration) ===

    Spawn Planning Ant (Route-Setter) via Task tool with subagent_type="aether-route-setter":
    # NOTE: Claude Code uses aether-route-setter; OpenCode now uses same specialist agent

    """
    You are a Route-Setter Ant in the Aether Colony.

    --- MISSION ---
    Create or refine a project plan based on research findings.

    Goal: "{goal}"
    Iteration: {iteration}/{max_iterations}

    {context_capsule_prompt}

    --- PLANNING DISCIPLINE ---
    Read .aether/planning.md for full reference.

    Key rules:
    - Bite-sized tasks (2-5 minutes each) - one action per task
    - Goal-oriented - describe WHAT to achieve, not HOW
    - Constraints define boundaries, not implementation
    - Hints point toward patterns, not solutions
    - Success criteria are testable outcomes

    Task format (GOAL-ORIENTED):
    ```
    Task N.1: {goal description}
    Goal: What to achieve (not how)
    Constraints:
      - Boundaries and requirements
      - Integration points
    Hints:
      - Pointer to existing patterns (optional)
      - Relevant files to reference (optional)
    Success Criteria:
      - Testable outcome 1
      - Testable outcome 2
    ```

    DO NOT include:
    - Exact code to write
    - Specific function names (unless critical API)
    - Implementation details
    - Line-by-line instructions

    Workers discover implementations by reading existing code and patterns.
    This enables TRUE EMERGENCE - different approaches based on context.

    --- RESEARCH FINDINGS ---
    {scout.findings formatted — compact, max 5 items}

    --- PHASE DOMAIN RESEARCH (from Step 3.6) ---
    {research_findings_summary if available, otherwise omit this section}

    Remaining Gaps:
    {gaps formatted — compact, max 3 items}

    --- CURRENT PLAN DRAFT ---
    {if plan_draft:}
    {plan_draft}
    {else:}
    No plan yet. Create initial draft.
    {end if}

    --- INSTRUCTIONS ---
    1. If no plan exists, create 3-6 phases with concrete tasks
    2. If plan exists, refine based on NEW information only
    3. Rate confidence across 5 dimensions
    4. Keep response concise — no verbose explanations

    Do NOT assign castes to tasks - describe the work only.

    --- OUTPUT CONSTRAINTS ---
    Maximum 6 phases. Maximum 4 tasks per phase.
    Maximum 2 sentence description per task.
    Confidence dimensions as single numbers, not paragraphs.

    --- OUTPUT FORMAT ---
    Return JSON:
    {
      "plan": {
        "phases": [
          {
            "id": 1,
            "name": "...",
            "description": "...",
            "tasks": [
              {
                "id": "1.1",
                "goal": "What to achieve (not how)",
                "constraints": ["boundary 1", "boundary 2"],
                "hints": ["optional pointer to pattern"],
                "success_criteria": ["testable outcome 1", "testable outcome 2"],
                "depends_on": []
              }
            ],
            "success_criteria": ["...", "..."]
          }
        ]
      },
      "confidence": {
        "knowledge": 0-100,
        "requirements": 0-100,
        "risks": 0-100,
        "dependencies": 0-100,
        "effort": 0-100,
        "overall": 0-100
      },
      "delta_reasoning": "One sentence: what changed from last iteration",
      "unresolved_gaps": ["...", "..."]
    }
    """

    Parse planning results. Update plan_draft and confidence.

    # === UPDATE WATCH FILES ===

    Update `.aether/data/watch-status.txt` with current state.
    Update `.aether/data/watch-progress.txt` with progress bar.

    # === STALL TRACKING ===

    delta = confidence - last_confidence
    if delta < 5:
        stall_count += 1
    else:
        stall_count = 0

    last_confidence = confidence
```

**After loop exits (auto-finalize, no user prompt needed):**

```
Planning complete after {iteration} iteration(s).

Confidence: {confidence}%
{if gaps remain:}
Note: {gaps.length} knowledge gap(s) deferred — these can be resolved during builds.
{end if}
```

Proceed directly to Step 5. No user confirmation needed — the plan auto-finalizes.

### Step 5: Finalize Plan

Once loop exits (confidence >= {target_confidence}, max iterations reached, or stall detected):

Read current COLONY_STATE.json, then update:
- Set `plan.phases` to the final phases array
- Set `plan.generated_at` to ISO-8601 timestamp
- Set `state` to `"READY"`
- Append event: `"<timestamp>|plan_generated|plan|Generated {N} phases with {confidence}% confidence"`

Write COLONY_STATE.json using state-mutate (atomic, locked, validated):
```bash
bash .aether/aether-utils.sh state-mutate \
  --argjson phases "$(echo '$PLAN_JSON' | jq '.plan.phases')" \
  --arg ts "$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
  --argjson n "$PHASE_COUNT" \
  --argjson conf "$CONFIDENCE" \
  '.plan.phases = $phases | .plan.generated_at = $ts | .state = "READY" | .events += [$ts + "|plan_generated|plan|Generated " + ($n|tostring) + " phases with " + ($conf|tostring) + "% confidence"]'
```

**Verify the write** — read back and confirm the plan persisted:
```bash
verify_phases=$(jq '.plan.phases | length' .aether/data/COLONY_STATE.json)
verify_timestamp=$(jq -r '.plan.generated_at' .aether/data/COLONY_STATE.json)
verify_state=$(jq -r '.state' .aether/data/COLONY_STATE.json)
if [[ "$verify_phases" -lt 1 || "$verify_timestamp" == "null" || "$verify_state" != "READY" ]]; then
  echo "ERROR: Plan write verification failed (phases=$verify_phases, generated_at=$verify_timestamp, state=$verify_state)"
  echo "Attempting retry write..."
  bash .aether/aether-utils.sh state-mutate --argjson phases "$(echo '$PLAN_JSON' | jq '.plan.phases')" --arg ts "$(date -u +%Y-%m-%dT%H:%M:%SZ)" '.plan.phases = $phases | .plan.generated_at = $ts | .state = "READY"'
  verify_phases=$(jq '.plan.phases | length' .aether/data/COLONY_STATE.json)
  if [[ "$verify_phases" -lt 1 ]]; then
    echo "FATAL: Retry write also failed. Plan was not persisted."
    echo "Re-run /ant:plan to regenerate."
    stop
  fi
fi
echo "Plan verified: $verify_phases phases, generated_at=$verify_timestamp, state=$verify_state"
```

Log plan completion: `bash .aether/aether-utils.sh activity-log "PLAN_COMPLETE" "queen" "Plan finalized with {confidence}% confidence"`

Update watch-status.txt:
```
AETHER COLONY :: READY
=======================

State: READY
Plan: {N} phases generated
Confidence: {confidence}%

Ready to build.
```

### Step 6: Display Plan

Read `plan.phases` from COLONY_STATE.json and display:

```
📊🐜🗺️🐜📊 ═══════════════════════════════════════════════════
   C O L O N Y   P L A N
═══════════════════════════════════════════════════ 📊🐜🗺️🐜📊

👑 Goal: {goal}

{if plan was just generated:}
📊 Confidence: {confidence}%
🔄 Iterations: {iteration}
{end if}

─────────────────────────────────────────────────────

📍 Phase {id}: {name} [{STATUS}]
   {description}

   🐜 Tasks:
      {status_icon} {id}: {description}

   ✅ Success Criteria:
      • {criterion}

─────────────────────────────────────────────────────
(repeat for each phase)

🐜 Next Steps:
   {Calculate first_incomplete_phase: iterate through phases, find first where status != 'completed'. Default to 1 if all complete or no phases. Look up its name from plan.phases[id].name.}
   /ant:build {first_incomplete_phase}   🔨 Phase {first_incomplete_phase}: {phase_name}
   /ant:focus "<area>"                   🎯 Focus colony attention
   /ant:status                           📊 View colony status

💾 Plan persisted — safe to /clear before building
```

Status icons: pending = `[ ]`, in_progress = `[~]`, completed = `[✓]`

---

## Confidence Scoring Reference

Each dimension rated 0-100%:

| Dimension | What It Measures |
|-----------|------------------|
| Knowledge | Understanding of codebase structure, patterns, tech stack |
| Requirements | Clarity of success criteria and acceptance conditions |
| Risks | Identification of potential blockers and failure modes |
| Dependencies | Understanding of what affects what, ordering constraints |
| Effort | Ability to estimate relative complexity of tasks |

**Overall** = weighted average (knowledge 25%, requirements 25%, risks 20%, dependencies 15%, effort 15%)

**Target:** Use the selected planning depth target. Higher targets trade latency for stronger up-front plan quality.

---

## Auto-Termination Safeguards

The planning loop terminates automatically without requiring user input:

1. **Confidence Threshold**: Loop exits when overall confidence reaches `{target_confidence}%`

2. **Hard Iteration Cap**: Maximum `{max_iterations}` iterations (2 subagents per iteration: 1 scout + 1 planner)

3. **Stall Detection**: If confidence improves < 5% for 2 consecutive iterations, auto-finalize current plan

4. **Single Scout Research**: One researcher per iteration (broad on iteration 1, gap-focused on 2+) — no parallel Alpha/Beta or synthesis agent

5. **Compressed Output**: Subagents limited to 5 findings max, 2-sentence summaries, compact JSON

6. **Escape Hatch**: `/ant:plan --accept` accepts current plan regardless of confidence
