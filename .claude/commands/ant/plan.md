---
name: ant:plan
description: "📊🐜🗺️🐜📊 Show project plan or generate project-specific phases"
---

You are the **Queen**. Orchestrate iterative research and planning until 99% confidence.

## Instructions

### Step 0: Version Check (Non-blocking)

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
Iteration: 0/8

Active Workers:
  [Research Alpha] Starting...
  [Research Beta] Starting...
  [Synthesis] Waiting...
  [Planning] Waiting...

Last Activity:
  Planning loop initiated
```

Write `.aether/data/watch-progress.txt`:
```
Progress
========

[                    ] 0%

Target: 99% confidence

Iteration: 0/8
Gaps: (analyzing...)
```

Log start:
```bash
bash .aether/aether-utils.sh activity-log "PLAN_START" "queen" "Iterative planning loop initiated for goal"
```

### Step 4: Iterative Research/Planning Loop

Initialize tracking:
- `iteration = 0`
- `confidence = 0`
- `gaps = []` (list of knowledge gaps)
- `stuck_counts = {}` (gap_id -> count of iterations without progress)
- `plan_draft = null`
- `last_confidence = 0`
- `stall_count = 0` (consecutive iterations with < 5% improvement)

**Loop (max 8 iterations):**

```
while iteration < 8 AND confidence < 99:
    iteration += 1

    # === DUAL RESEARCH PHASE ===
    # Spawn two researchers in parallel for broader coverage and cross-validation

    Spawn Research Ant Alpha (Scout) via Task tool with subagent_type="general-purpose", run_in_background: true:

    """
    You are Scout Ant Alpha in the Aether Colony.

    --- MISSION ---
    Research the codebase from YOUR unique angle. Cover different ground than Beta.

    Goal: "{goal}"
    Iteration: {iteration}/8

    {if gaps is not empty:}
    --- SPECIFIC GAPS TO INVESTIGATE ---
    Focus on these gaps (Alpha takes odd-numbered gaps):
    {for each gap in gaps where index % 2 == 0:}
      - {gap.id}: {gap.description}
    {end for}
    {else:}
    --- INITIAL EXPLORATION (Alpha Focus) ---
    This is iteration 1. Alpha explores:
    1. Core architecture and entry points
    2. Main business logic and domain models
    3. Configuration and environment setup
    4. Critical dependencies and frameworks
    {end if}

    --- YOUR ANGLE ---
    Look for: structural patterns, architectural decisions, core abstractions
    Check: Main modules, domain layer, service boundaries
    Note: Integration points, external APIs, data flow

    --- TOOLS ---
    Use: Glob, Grep, Read, WebSearch, WebFetch
    Do NOT use: Task, Write, Edit

    --- OUTPUT FORMAT ---
    Return JSON:
    {
      "researcher_id": "alpha",
      "findings": [
        {"area": "...", "discovery": "...", "confidence": 0-100, "source": "file or search"}
      ],
      "gaps_remaining": [
        {"id": "gap_N", "description": "..."}
      ],
      "gaps_resolved": ["gap_1", "gap_2"],
      "overall_knowledge_confidence": 0-100,
      "unique_insights": ["insight not covered by typical exploration"]
    }
    """

    Spawn Research Ant Beta (Scout) via Task tool with subagent_type="general-purpose", run_in_background: true:

    """
    You are Scout Ant Beta in the Aether Colony.

    --- MISSION ---
    Research the codebase from YOUR unique angle. Cover different ground than Alpha.

    Goal: "{goal}"
    Iteration: {iteration}/8

    {if gaps is not empty:}
    --- SPECIFIC GAPS TO INVESTIGATE ---
    Focus on these gaps (Beta takes even-numbered gaps):
    {for each gap in gaps where index % 2 == 1:}
      - {gap.id}: {gap.description}
    {end for}
    {else:}
    --- INITIAL EXPLORATION (Beta Focus) ---
    This is iteration 1. Beta explores:
    1. UI/presentation layer and user flows
    2. Testing patterns and quality practices
    3. Edge cases, error handling, and validation
    4. Infrastructure, deployment, and ops concerns
    {end if}

    --- YOUR ANGLE ---
    Look for: edge cases, test coverage, UI/UX patterns, operational concerns
    Check: Test files, validation logic, error paths, deployment configs
    Note: Security considerations, performance bottlenecks, monitoring

    --- TOOLS ---
    Use: Glob, Grep, Read, WebSearch, WebFetch
    Do NOT use: Task, Write, Edit

    --- OUTPUT FORMAT ---
    Return JSON:
    {
      "researcher_id": "beta",
      "findings": [
        {"area": "...", "discovery": "...", "confidence": 0-100, "source": "file or search"}
      ],
      "gaps_remaining": [
        {"id": "gap_N", "description": "..."}
      ],
      "gaps_resolved": ["gap_1", "gap_2"],
      "overall_knowledge_confidence": 0-100,
      "unique_insights": ["insight not covered by typical exploration"]
    }
    """

    # === SYNTHESIS PHASE ===
    # Wait for both researchers, then synthesize their findings

    Wait for both Research Alpha and Research Beta to complete (TaskOutput with block: true).

    Log: `bash .aether/aether-utils.sh activity-log "RESEARCH" "scout-alpha" "Iteration {iteration}: {alpha.summary}"`
    Log: `bash .aether/aether-utils.sh activity-log "RESEARCH" "scout-beta" "Iteration {iteration}: {beta.summary}"`

    Spawn Synthesis Ant (Analyst) via Task tool with subagent_type="general-purpose":

    """
    You are a Synthesis Analyst Ant in the Aether Colony.

    --- MISSION ---
    Combine findings from Alpha and Beta researchers into unified understanding.
    Resolve conflicts, identify gaps, and extract the best insights from both.

    Goal: "{goal}"
    Iteration: {iteration}/8

    --- RESEARCH FINDINGS ---

    ## Alpha Findings (Architectural Focus):
    {alpha.findings formatted}
    Alpha Confidence: {alpha.overall_knowledge_confidence}%
    Alpha Gaps Resolved: {alpha.gaps_resolved}
    Alpha Gaps Remaining: {alpha.gaps_remaining}
    Alpha Unique Insights: {alpha.unique_insights}

    ## Beta Findings (Edge Cases/Operations Focus):
    {beta.findings formatted}
    Beta Confidence: {beta.overall_knowledge_confidence}%
    Beta Gaps Resolved: {beta.gaps_resolved}
    Beta Gaps Remaining: {beta.gaps_remaining}
    Beta Unique Insights: {beta.unique_insights}

    --- SYNTHESIS TASKS ---
    1. Merge findings: Combine both perspectives without duplication
    2. Resolve conflicts: If Alpha and Beta disagree, note the tension
    3. Identify gaps: What did BOTH miss? What's still unknown?
    4. Rate confidence: Combined knowledge confidence (not just average)

    --- OUTPUT FORMAT ---
    Return JSON:
    {
      "synthesis": {
        "combined_findings": [
          {"area": "...", "discovery": "...", "sources": ["alpha", "beta"], "confidence": 0-100}
        ],
        "conflicts": [
          {"topic": "...", "alpha_view": "...", "beta_view": "...", "resolution": "..."}
        ],
        "gaps_remaining": [
          {"id": "gap_N", "description": "...", "priority": "high|medium|low"}
        ],
        "gaps_resolved": ["gap_1", "gap_2"],
        "overall_knowledge_confidence": 0-100,
        "key_insights": ["most important discoveries from both researchers"]
      }
    }
    """

    Parse synthesis results. Update gaps list.

    Log: `bash .aether/aether-utils.sh activity-log "SYNTHESIS" "analyst" "Iteration {iteration}: {synthesis.key_insights.length} insights, {synthesis.gaps_remaining.length} gaps remain"`

    # === PLANNING PHASE ===

    Spawn Planning Ant (Route-Setter) via Task tool with subagent_type="general-purpose":

    """
    You are a Route-Setter Ant in the Aether Colony.

    --- MISSION ---
    Create or refine a project plan based on research findings.

    Goal: "{goal}"
    Iteration: {iteration}/8

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

    --- SYNTHESIZED RESEARCH FINDINGS ---
    Combined findings from Alpha and Beta researchers:
    {synthesis.combined_findings formatted}

    Key Insights:
    {synthesis.key_insights formatted}

    Remaining Gaps:
    {synthesis.gaps_remaining formatted}

    --- CURRENT PLAN DRAFT ---
    {if plan_draft:}
    {plan_draft}
    {else:}
    No plan yet. Create initial draft.
    {end if}

    --- REMAINING GAPS ---
    {synthesis.gaps_remaining}

    --- INSTRUCTIONS ---
    1. If no plan exists, create 3-6 phases with concrete tasks
    2. Each task must have: exact file paths, steps, expected outputs
    3. If plan exists, refine based on NEW information only
    4. Rate confidence across 5 dimensions
    5. Explain what changed from last iteration

    Do NOT assign castes to tasks - describe the work only.

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
      "delta_reasoning": "What NEW information changed the plan",
      "unresolved_gaps": ["...", "..."]
    }
    """

    Parse planning results. Update plan_draft and confidence.

    Log: `bash .aether/aether-utils.sh activity-log "PLANNING" "route-setter" "Confidence: {confidence}% (+{delta}%)"`

    # === UPDATE WATCH FILES ===

    Update `.aether/data/watch-status.txt` with current state.
    Update `.aether/data/watch-progress.txt` with progress bar.

    # === ANTI-STUCK CHECKS ===

    # Check 1: Gap stuck detection
    for each gap in gaps:
        if gap was in previous iteration's gaps:
            stuck_counts[gap.id] += 1
            if stuck_counts[gap.id] >= 2:
                mark gap as "needs_human_input"
                remove from active gaps

    # Check 2: Stall detection
    delta = confidence - last_confidence
    if delta < 5:
        stall_count += 1
    else:
        stall_count = 0

    if stall_count >= 2 AND iteration > 3:
        # Stalled - ask user
        Display current plan and confidence
        Ask: "Planning has stalled at {confidence}%. Options:"
        1. Continue iterating (may not improve)
        2. Accept current plan
        3. Provide guidance on: {gaps marked needs_human_input}

        if user chooses "accept" OR force_accept:
            break loop
        if user provides guidance:
            add guidance to gaps as FOCUS constraint
            reset stall_count

    # Check 3: Diminishing returns (tighter bound for 8 iterations)
    if delta < 2 AND iteration >= 5:
        Display: "Approaching local maximum at {confidence}%."
        Ask: "Continue or accept current plan?"
        if user accepts:
            break loop

    last_confidence = confidence
```

**After loop exits:**

If `iteration == 8` and `confidence < 99`:
```
Planning reached maximum iterations (8).

Current confidence: {confidence}%
Unresolved gaps:
{list gaps marked needs_human_input}

Recommendation: {proceed if confidence > 85, else get human input}
```
Ask user to accept or provide input.

### Step 5: Finalize Plan

Once plan is accepted (confidence >= 99 OR user approved):

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

**Target: 99%** - High confidence plan ready for autonomous execution.

---

## Anti-Stuck Safeguards

1. **Gap Tracking**: Each gap has a `stuck_count`. After 2 iterations without progress, marked "needs human input"

2. **Stall Detection**: If confidence improves < 5% for 2 consecutive iterations after iteration 3, pause for user decision

3. **Diminishing Returns**: If confidence improves < 2% after iteration 5, offer to accept current plan

4. **Dual Research**: Two researchers (Alpha + Beta) explore different angles, then synthesize findings

5. **Escape Hatch**: `/ant:plan --accept` accepts current plan regardless of confidence
