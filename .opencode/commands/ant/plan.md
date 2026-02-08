---
name: ant:plan
description: "Show project plan or generate project-specific phases"
---

You are the **Queen**. Orchestrate iterative research and planning until 95% confidence.

## Instructions

### Step 1: Read State + Version Check

Read `.aether/data/COLONY_STATE.json`.

**Auto-upgrade old state:**
If `version` field is missing, "1.0", or "2.0":
1. Preserve: `goal`, `state`, `current_phase`, `plan.phases`
2. Write upgraded v3.0 state
3. Output: `State auto-upgraded to v3.0`
4. Continue with command.

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

Log start:
```bash
bash ~/.aether/aether-utils.sh activity-log "PLAN_START" "queen" "Iterative planning loop initiated for goal"
```

### Step 4: Iterative Research/Planning Loop

Initialize tracking:
- `iteration = 0`
- `confidence = 0`
- `gaps = []`
- `plan_draft = null`
- `stall_count = 0`

**Loop (max 50 iterations):**

```
while iteration < 50 AND confidence < 95:
    iteration += 1

    # === RESEARCH PHASE ===
    Spawn Research Ant (Scout) via task tool with subagent_type: "general":

    """
    You are a Scout Ant in the Aether Colony.

    --- MISSION ---
    Research the codebase and domain to fill knowledge gaps for planning.

    Goal: "{goal}"
    Iteration: {iteration}/50

    {if gaps:}
    --- SPECIFIC GAPS TO INVESTIGATE ---
    Focus ONLY on these gaps:
    {for each gap: - {gap.id}: {gap.description}}
    {else:}
    --- INITIAL EXPLORATION ---
    1. Codebase structure
    2. Existing patterns
    3. Dependencies and tech stack
    4. Tests or docs
    {end if}

    --- OUTPUT FORMAT ---
    Return JSON:
    {
      "findings": [{"area": "...", "discovery": "...", "confidence": 0-100}],
      "gaps_remaining": [{"id": "gap_N", "description": "..."}],
      "gaps_resolved": ["gap_1"],
      "overall_knowledge_confidence": 0-100
    }
    """

    Parse research results. Update gaps list.

    # === PLANNING PHASE ===
    Spawn Planning Ant (Route-Setter) via task tool with subagent_type: "general":

    """
    You are a Route-Setter Ant in the Aether Colony.

    --- MISSION ---
    Create or refine a project plan based on research findings.

    Goal: "{goal}"

    --- PLANNING DISCIPLINE ---
    Read ~/.aether/planning.md for full reference.

    Key rules:
    - Bite-sized tasks (2-5 minutes each)
    - Goal-oriented - describe WHAT to achieve, not HOW
    - Success criteria are testable outcomes

    --- RESEARCH FINDINGS ---
    {research_results.findings}

    --- CURRENT PLAN DRAFT ---
    {plan_draft or "No plan yet. Create initial draft."}

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
                "goal": "What to achieve",
                "constraints": ["boundary 1"],
                "hints": ["optional pointer"],
                "success_criteria": ["testable outcome"],
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

    # === ANTI-STUCK CHECKS ===
    if stall_count >= 3 AND iteration > 5:
        Display current plan and confidence
        Ask user to accept or provide guidance
```

### Step 5: Finalize Plan

Once plan is accepted (confidence >= 95 OR user approved):

Read current COLONY_STATE.json, then update:
- Set `plan.phases` to the final phases array
- Set `plan.generated_at` to ISO-8601 timestamp
- Set `plan.confidence` to final confidence value
- Set `state` to `"READY"`
- Append event: `"<timestamp>|plan_generated|plan|Generated {N} phases with {confidence}% confidence"`

Write COLONY_STATE.json.

Log: `bash ~/.aether/aether-utils.sh activity-log "PLAN_COMPLETE" "queen" "Plan finalized"`

### Step 6: Display Plan

Read `plan.phases` from COLONY_STATE.json and display:

```
═══════════════════════════════════════════════════
   C O L O N Y   P L A N
═══════════════════════════════════════════════════

Goal: {goal}

{if plan was just generated:}
Confidence: {confidence}%
Iterations: {iteration}
{end if}

─────────────────────────────────────────────────────

Phase {id}: {name} [{STATUS}]
   {description}

   Tasks:
      {status_icon} {id}: {description}

   Success Criteria:
      * {criterion}

─────────────────────────────────────────────────────
(repeat for each phase)

Next Steps:
   /ant:build 1        Start building Phase 1
   /ant:focus "<area>" Focus colony attention
   /ant:status         View colony status

Plan persisted - safe to /clear before building
```

Status icons: pending = `[ ]`, in_progress = `[~]`, completed = `[x]`
