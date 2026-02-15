---
name: ant:plan
description: "📊🐜🗺️🐜📊 Show project plan or generate project-specific phases"
---

You are the **Queen**. Orchestrate research and planning until 80% confidence (maximum 4 iterations).

## Instructions

Parse `$ARGUMENTS`:
- If contains `--no-visual`: set `visual_mode = false` (visual is ON by default)
- Otherwise: set `visual_mode = true`

### Step 0: Initialize Visual Mode (if enabled)

If `visual_mode` is true:
```bash
# Generate session ID
plan_id="plan-$(date +%s)"

# Initialize swarm display
bash .aether/aether-utils.sh swarm-display-init "$plan_id"
bash .aether/aether-utils.sh swarm-display-update "Queen" "prime" "excavating" "Generating colony plan" "Colony" '{"read":0,"grep":0,"edit":0,"bash":0}' 0 "fungus_garden" 0
```

### Step 0.5: Version Check (Non-blocking)

Run using the Bash tool: `bash .aether/aether-utils.sh version-check 2>/dev/null || true`

If the command succeeds and the JSON result contains a non-empty string, display it as a one-line notice. Proceed regardless of outcome.

### Step 1: Read State + Version Check

Read `.aether/data/COLONY_STATE.json`.

**Auto-upgrade old state:**
If `version` field is missing, "1.0", or "2.0":
1. Preserve: `goal`, `state`, `current_phase`, `plan.phases`
2. Write upgraded v3.0 state (same structure as /ant:init but preserving data)
3. Output: `State auto-upgraded to v3.0`
4. Continue with command.

Extract: `goal`, `plan.phases`

**Validate:** If `goal: null`:
```
No colony initialized. Run /ant:init "<goal>" first.
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

Parse `$ARGUMENTS`:
- If contains `--accept`: Set `force_accept = true` (accept current plan regardless of confidence)
- Otherwise: `force_accept = false`

### Step 3: Initialize Planning State

Update watch files for tmux visibility:

Write `.aether/data/watch-status.txt`:
```
AETHER COLONY :: PLANNING
==========================

State: PLANNING
Phase: 0/0 (generating plan)
Confidence: 0%
Iteration: 0/4

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

Target: 80% confidence

Iteration: 0/4
Gaps: (analyzing...)
```

Log start:
```bash
bash .aether/aether-utils.sh activity-log "PLAN_START" "queen" "Iterative planning loop initiated for goal"
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

### Step 4: Research and Planning Loop

Initialize tracking:
- `iteration = 0`
- `confidence = 0`
- `gaps = []` (list of knowledge gaps)
- `plan_draft = null`
- `last_confidence = 0`
- `stall_count = 0` (consecutive iterations with < 5% improvement)

**Loop (max 4 iterations, 2 agents per iteration: 1 scout + 1 planner):**

```
while iteration < 4 AND confidence < 80:
    iteration += 1

    # === AUTO-BREAK CHECKS (no user prompt needed) ===
    if iteration > 1:
        if confidence >= 80:
            Log: "Confidence threshold reached ({confidence}%), finalizing plan"
            break
        if stall_count >= 2:
            Log: "Planning stalled at {confidence}%, finalizing current plan"
            break

    # === RESEARCH PHASE (always runs — 1 scout per iteration) ===

    if iteration == 1:

        # Broad exploration on first pass
        Spawn Research Scout via Task tool with subagent_type="general-purpose":

        """
        You are a Scout Ant in the Aether Colony.

        --- MISSION ---
        Research the codebase to understand what exists and how it works.

        Goal: "{goal}"
        Iteration: {iteration}/4

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
        Spawn Gap-Focused Scout via Task tool with subagent_type="general-purpose":

        """
        You are a Scout Ant in the Aether Colony (gap-focused research).

        --- MISSION ---
        Investigate ONLY these specific knowledge gaps. Do not explore broadly.

        Goal: "{goal}"
        Iteration: {iteration}/4

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

    Log: `bash .aether/aether-utils.sh activity-log "RESEARCH" "scout" "Iteration {iteration}: {scout.findings.length} findings, {scout.gaps_remaining.length} gaps"`

    # === PLANNING PHASE (always runs — 1 planner per iteration) ===

    Spawn Planning Ant (Route-Setter) via Task tool with subagent_type="general-purpose":
    # NOTE: Claude Code uses aether-route-setter; OpenCode uses general-purpose with role injection

    """
    You are a Route-Setter Ant in the Aether Colony.

    --- MISSION ---
    Create or refine a project plan based on research findings.

    Goal: "{goal}"
    Iteration: {iteration}/4

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

    Log: `bash .aether/aether-utils.sh activity-log "PLANNING" "route-setter" "Confidence: {confidence}% (+{delta}%)"`

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

Once loop exits (confidence >= 80, max iterations reached, or stall detected):

Read current COLONY_STATE.json, then update:
- Set `plan.phases` to the final phases array
- Set `plan.generated_at` to ISO-8601 timestamp
- Set `state` to `"READY"`
- Append event: `"<timestamp>|plan_generated|plan|Generated {N} phases with {confidence}% confidence"`

Write COLONY_STATE.json.

Log: `bash .aether/aether-utils.sh activity-log "PLAN_COMPLETE" "queen" "Plan finalized with {confidence}% confidence"`

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

**If visual_mode is true, render final swarm display:**
```bash
bash .aether/aether-utils.sh swarm-display-update "Queen" "prime" "completed" "Plan generated" "Colony" '{"read":8,"grep":4,"edit":2,"bash":1}' 100 "fungus_garden" 100
bash .aether/aether-utils.sh swarm-display-render "$plan_id"
```

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

**Target: 80%** - Sufficient confidence for autonomous execution. Higher confidence is achieved during builds as gaps are resolved.

---

## Auto-Termination Safeguards

The planning loop terminates automatically without requiring user input:

1. **Confidence Threshold**: Loop exits when overall confidence reaches 80%

2. **Hard Iteration Cap**: Maximum 4 iterations (8 subagents total: 1 scout + 1 planner per iteration)

3. **Stall Detection**: If confidence improves < 5% for 2 consecutive iterations, auto-finalize current plan

4. **Single Scout Research**: One researcher per iteration (broad on iteration 1, gap-focused on 2+) — no parallel Alpha/Beta or synthesis agent

5. **Compressed Output**: Subagents limited to 5 findings max, 2-sentence summaries, compact JSON

6. **Escape Hatch**: `/ant:plan --accept` accepts current plan regardless of confidence
