---
name: ant:plan
description: Show project plan or generate project-specific phases
---

You are the **Queen**. Orchestrate iterative research and planning until 95% confidence.

## Instructions

### Step 1: Read State

Read `.aether/data/COLONY_STATE.json`.

Extract: `goal`, `plan.phases`

**Validate:** If `goal: null`:
```
No colony initialized. Run /ant:init "<goal>" first.
```
Stop here.

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
Iteration: 0/50

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

Target: 95% confidence

Iteration: 0/50
Gaps: (analyzing...)
```

Log start:
```bash
bash ~/.aether/aether-utils.sh activity-log "PLAN_START" "queen" "Iterative planning loop initiated for goal"
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

**Loop (max 50 iterations):**

```
while iteration < 50 AND confidence < 95:
    iteration += 1

    # === RESEARCH PHASE ===

    Spawn Research Ant (Scout) via Task tool with subagent_type="general-purpose":

    """
    You are a Scout Ant in the Aether Colony.

    --- MISSION ---
    Research the codebase and domain to fill knowledge gaps for planning.

    Goal: "{goal}"
    Iteration: {iteration}/50

    {if gaps is not empty:}
    --- SPECIFIC GAPS TO INVESTIGATE ---
    Focus ONLY on these gaps:
    {for each gap in gaps:}
      - {gap.id}: {gap.description}
    {end for}
    {else:}
    --- INITIAL EXPLORATION ---
    This is iteration 1. Explore:
    1. Codebase structure (Glob for key files)
    2. Existing patterns and conventions
    3. Dependencies and tech stack
    4. Any existing tests or docs
    {end if}

    --- TOOLS ---
    Use: Glob, Grep, Read, WebSearch, WebFetch
    Do NOT use: Task, Write, Edit

    --- OUTPUT FORMAT ---
    Return JSON:
    {
      "findings": [
        {"area": "...", "discovery": "...", "confidence": 0-100}
      ],
      "gaps_remaining": [
        {"id": "gap_N", "description": "..."}
      ],
      "gaps_resolved": ["gap_1", "gap_2"],
      "overall_knowledge_confidence": 0-100
    }
    """

    Parse research results. Update gaps list.

    Log: `bash ~/.aether/aether-utils.sh activity-log "RESEARCH" "scout" "Iteration {iteration}: {summary}"`

    # === PLANNING PHASE ===

    Spawn Planning Ant (Route-Setter) via Task tool with subagent_type="general-purpose":

    """
    You are a Route-Setter Ant in the Aether Colony.

    --- MISSION ---
    Create or refine a project plan based on research findings.

    Goal: "{goal}"
    Iteration: {iteration}/50

    --- RESEARCH FINDINGS ---
    {research_results.findings formatted}

    --- CURRENT PLAN DRAFT ---
    {if plan_draft:}
    {plan_draft}
    {else:}
    No plan yet. Create initial draft.
    {end if}

    --- REMAINING GAPS ---
    {gaps_remaining}

    --- INSTRUCTIONS ---
    1. If no plan exists, create 3-6 phases with concrete tasks
    2. If plan exists, refine based on NEW information only
    3. Rate confidence across 5 dimensions
    4. Explain what changed from last iteration

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
              {"id": "1.1", "description": "...", "depends_on": []}
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

    Log: `bash ~/.aether/aether-utils.sh activity-log "PLANNING" "route-setter" "Confidence: {confidence}% (+{delta}%)"`

    # === UPDATE WATCH FILES ===

    Update `.aether/data/watch-status.txt` with current state.
    Update `.aether/data/watch-progress.txt` with progress bar.

    # === ANTI-STUCK CHECKS ===

    # Check 1: Gap stuck detection
    for each gap in gaps:
        if gap was in previous iteration's gaps:
            stuck_counts[gap.id] += 1
            if stuck_counts[gap.id] >= 3:
                mark gap as "needs_human_input"
                remove from active gaps

    # Check 2: Stall detection
    delta = confidence - last_confidence
    if delta < 5:
        stall_count += 1
    else:
        stall_count = 0

    if stall_count >= 3 AND iteration > 5:
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

    # Check 3: Diminishing returns
    if delta < 2 AND iteration > 10:
        Display: "Approaching local maximum at {confidence}%."
        Ask: "Continue or accept current plan?"
        if user accepts:
            break loop

    last_confidence = confidence
```

**After loop exits:**

If `iteration == 50` and `confidence < 95`:
```
Planning reached maximum iterations.

Current confidence: {confidence}%
Unresolved gaps:
{list gaps marked needs_human_input}

Recommendation: {proceed if confidence > 70, else get human input}
```
Ask user to accept or provide input.

### Step 5: Finalize Plan

Once plan is accepted (confidence >= 95 OR user approved):

Read current COLONY_STATE.json, then update:
- Set `plan.phases` to the final phases array
- Set `plan.generated_at` to ISO-8601 timestamp
- Set `state` to `"READY"`
- Append event: `"<timestamp>|plan_generated|plan|Generated {N} phases with {confidence}% confidence"`

Write COLONY_STATE.json.

Log: `bash ~/.aether/aether-utils.sh activity-log "PLAN_COMPLETE" "queen" "Plan finalized with {confidence}% confidence"`

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
+=====================================================+
|  AETHER COLONY :: PLAN                               |
+=====================================================+

Goal: {goal}

{if plan was just generated:}
Confidence: {confidence}%
Iterations: {iteration}
{end if}

---

Phase {id}: {name} [{STATUS}]
  {description}

  Tasks:
    [{status_icon}] {id}: {description}

  Success Criteria:
    - {criterion}

---
(repeat for each phase)

NEXT STEPS:
  /ant:build 1           Start building Phase 1
  /ant:focus "<area>"    Focus colony attention
  /ant:status            View colony status
```

Status icons: pending = `[ ]`, in_progress = `[~]`, completed = `[x]`

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

**Target: 95%** - High confidence plan ready for autonomous execution.

---

## Anti-Stuck Safeguards

1. **Gap Tracking**: Each gap has a `stuck_count`. After 3 iterations without progress, marked "needs human input"

2. **Stall Detection**: If confidence improves < 5% for 3 consecutive iterations after iteration 5, pause for user decision

3. **Diminishing Returns**: If confidence improves < 2% after iteration 10, offer to accept current plan

4. **Focused Research**: Research Ant receives SPECIFIC gaps, not open-ended exploration (after iteration 1)

5. **Escape Hatch**: `/ant:plan --accept` accepts current plan regardless of confidence
