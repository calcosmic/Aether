<!-- Generated from .aether/commands/resume.yaml - DO NOT EDIT DIRECTLY -->
---
name: ant:resume
description: "Resume Previous Session"
---

### Step -1: Normalize Arguments

Run: `normalized_args=$(bash .aether/aether-utils.sh normalize-args "$@")`

This ensures arguments work correctly in both Claude Code and OpenCode. Use `$normalized_args` throughout this command.

# /ant:resume — Resume Previous Session

Resume work after `/clear` or in a new session. Reads colony state, detects codebase drift, and gives you a clear "do this next" recommendation.

## Usage

```bash
/ant:resume
```

---

## Implementation

Execute the following steps in order when the user runs `/ant:resume`.

---

### Step 1: Read Session State

Run::
```bash
aether session-read
```

Parse the JSON result.

- If `exists` is `false`: display the following and **stop**:

```

═══════════════════════════════════════════════════
RESUME SESSION

No previous session found.

Start fresh: /ant:init "your goal"
Or check: /ant:status

═══════════════════════════════════════════════════
```

- If `exists` is `true`: extract from the session data:
  - `colony_goal`
  - `current_phase`
  - `last_command`
  - `suggested_next`
  - `baseline_commit`
  - `session_id`

---

### Step 2: Read COLONY_STATE.json (Authoritative Source)

Use the Read tool to read `.aether/data/COLONY_STATE.json`.

COLONY_STATE.json is the authoritative source for goal and state (session.json may be stale). Extract:
- `goal` (use this as authoritative, overriding session.json colony_goal)
- `milestone` (check for sealed colony)
- `state` (READY, PLANNING, EXECUTING, PAUSED)
- `current_phase`
- `plan.phases` (array with id, name, status for each phase)
- `plan.generated_at`
- `memory.decisions` (flat list — do NOT distinguish user vs Claude origin)
- `events` (last 5 for recent activity context)

**If `milestone` == `"Crowned Anthill"`:** This colony has been sealed. Display:
```
This colony has been sealed (Crowned Anthill).

Start a new colony with /ant:init "new goal"
```
Stop here — do NOT display stale phase data from the sealed colony.

If the file is missing or the JSON cannot be parsed, **stop immediately** and display:

```
State file missing or corrupted.

Options:
1. Start fresh with /ant:init "goal"
2. Try to recover (I'll look for backup files)

What would you like to do?
```

Do NOT proceed with stale or fabricated data.

---

### Step 3: Read Pheromone Signals

Run::
```bash
aether pheromone-read all
```

Parse the JSON result. Extract `.result.signals` array.

- If `ok` is `true` and `.result.signals` is non-empty: store signals for dashboard rendering in Step 8
- If `ok` is `true` and `.result.signals` is empty: no active pheromones (skip in dashboard)
- If the command fails or returns an error: skip silently (no pheromones active)

Note: pheromone-read applies decay calculation automatically. The `effective_strength` field reflects current signal strength after time-based decay. Signals below 0.1 effective strength are already filtered out.

---

### Step 4: Read CONTEXT.md

Use the Read tool to read `.aether/CONTEXT.md` if it exists.

If missing: fall back to COLONY_STATE.json for narrative context. Note: "Context document not found — reconstructing from state."

---

### Step 5: Drift Detection

Extract `baseline_commit` from the session.json data read in Step 1.

```bash
current_commit=$(git rev-parse HEAD 2>/dev/null || echo "")
```

If `baseline_commit` is non-empty and differs from `current_commit`:

```bash
commit_count=$(git rev-list --count "$baseline_commit..HEAD" 2>/dev/null || echo "0")
changed_count=$(git diff --stat "$baseline_commit" HEAD 2>/dev/null | tail -1 | grep -oE '[0-9]+ file' | grep -oE '[0-9]+' || echo "0")
```

Store `drift_detected=true`, `commit_count`, `changed_count` for dashboard rendering.

If `baseline_commit` is empty or matches `current_commit`: set `drift_detected=false`.

Restore identically regardless of time elapsed — no warnings about session age.

---

### Step 6: Compute Workflow Position and Next-Step Guidance

Compute `suggested_next` dynamically from COLONY_STATE.json data. Do not use the static value from session.json.

Use this decision tree:

```
Case 1 — No plan created yet:
  Check: plan.phases is empty AND plan.generated_at is null
  recommended = "/ant:plan"
  reason = "No plan created yet"
  alternatives = ["/ant:colonize — analyze codebase first"]

Case 2 — Plan ready, first phase not started:
  Check: plan.phases is not empty AND state == "READY" AND current_phase == 0
  recommended = "/ant:build 1"
  reason = "Plan ready, first phase not started"
  alternatives = ["/ant:plan — review or regenerate plan"]

Case 3 — Build in progress:
  Check: state == "EXECUTING"
  recommended = "/ant:continue"
  reason = "Build in progress"
  alternatives = ["/ant:build {current_phase} — rebuild current phase", "/ant:flags — check for blockers"]

Case 4 — Phase complete, next phase available:
  Check: state == "READY" AND current_phase > 0 AND current_phase < plan.phases.length
  next = current_phase + 1
  recommended = "/ant:build {next}"
  reason = "Phase {current_phase} complete, ready for next"
  alternatives = ["/ant:plan — regenerate plan", "/ant:phase {next} — preview next phase"]

Case 5 — All phases complete:
  Check: state == "READY" AND current_phase > 0 AND current_phase >= plan.phases.length
  recommended = "/ant:seal"
  reason = "All phases complete"
  alternatives = ["/ant:status — view final state"]

Case 6 — Colony paused:
  Check: state == "PAUSED"
  recommended = "/ant:resume-colony"
  reason = "Colony is paused"
  alternatives = ["/ant:status — check state first"]

Default:
  recommended = "/ant:status"
  reason = "Check colony status"
  alternatives = []
```

---

### Step 7: Workflow-Step Blocking (Early-Return Guards)

Run these guards BEFORE rendering the dashboard. If a blocking condition is detected, output the block message and STOP. Do not render the dashboard. Do not offer alternative commands.

**BLOCK CONDITION 1: No plan exists**

Check: plan.phases is empty AND plan.generated_at is null

Output and STOP:

```
BLOCKED: No plan exists yet.
Required: Run /ant:plan to create a build plan.
Goal: {goal}
```

Stop here — do not continue to Step 8 or render the dashboard.

---

**BLOCK CONDITION 2: Plan attempted but failed**

Check: plan.phases is empty AND plan.generated_at is not null

Output and STOP:

```
BLOCKED: Plan was attempted but has no phases.
Required: Run /ant:plan to regenerate the plan.
Goal: {goal}
```

Stop here — do not continue to Step 8 or render the dashboard.

---

**BLOCK CONDITION 3: Build interrupted**

Check: state == "EXECUTING" AND the last 3 events show no recent build activity

Output and STOP:

```
BLOCKED: Build may have been interrupted.
Required: Run /ant:continue to check and advance.
Goal: {goal}
```

Stop here — do not continue to Step 8 or render the dashboard.

---

### Step 8: Render Dashboard

Lead with the next-step recommendation. Context follows underneath ("straight to action" ordering).

```

═══════════════════════════════════════════════════
RESUME SESSION

Next: {recommended}
      {reason}
{if alternatives exist:}
Also: {alternatives, comma-separated}
{end}

{if drift_detected:}
Note: Codebase changed since last session ({commit_count} commit(s), {changed_count} file(s) modified)
{end}

Goal: {goal}
State: {state}
Phase: {current_phase}/{total_phases}

Phase Progress:
{for each phase in plan.phases:}
  [{status_icon}] Phase {id}: {name}
{end}
```

Status icons:
- completed: `v` (checkmark)
- in_progress: `~` (tilde)
- pending: ` ` (space)

```
{if memory.decisions is not empty:}
Recent Decisions:
{for each of the last 5 decisions:}
  - {decision text}
{end}
{end}

{if signals array from Step 3 is not empty:}
Active Signals:
{for each signal in signals:}
  {signal.type}: "{signal.content}" [{signal.effective_strength * 100 | floor}%]
{end}
{end}
```

---

### Step 8.5: Display Memory Health (Secondary)

Run::
```bash
aether resume-dashboard
```

Extract memory_health from the JSON result:
- wisdom_count
- pending_promotions
- recent_failures

Display after the main dashboard:
```
📊 Memory Health
   Wisdom: {wisdom_count} entries | Pending: {pending_promotions} promotions | Failures: {recent_failures} recent

   Run /ant:memory-details for full breakdown
```

If all counts are 0, show:
```
📊 Memory Health
   No accumulated wisdom yet. Complete phases to build colony memory.
```

Last Command: {last_command}
Session: {session_id}
```

---

### Step 9: Mark Session Resumed

Run::
```bash
aether session-mark-resumed
```

### Step 10: Next Up

Generate the state-based Next Up block by Run::
```bash
state=$(jq -r '.state // "IDLE"' .aether/data/COLONY_STATE.json)
current_phase=$(jq -r '.current_phase // 0' .aether/data/COLONY_STATE.json)
total_phases=$(jq -r '.plan.phases | length' .aether/data/COLONY_STATE.json)
aether print-next-up
```

---

## Error Handling Reference

| Condition | Response |
|-----------|----------|
| session.json missing (exists=false) | "No previous session found" — offer /ant:init and /ant:status |
| COLONY_STATE.json missing or corrupted | Pause, ask user: start fresh or recover |
| pheromone-read fails | Skip silently (no pheromones) |
| CONTEXT.md missing | Fall back to COLONY_STATE.json narrative |
| No plan phases, no generated_at | BLOCK — redirect to /ant:plan |
| Plan attempted but no phases | BLOCK — redirect to /ant:plan |
| State EXECUTING, events show no activity | BLOCK — redirect to /ant:continue |
| baseline_commit matches current HEAD | No drift warning shown |
| baseline_commit differs from current HEAD | Show informational drift note |

---

## Key Constraints

- Use Read tool for COLONY_STATE.json (not bash cat/jq). Use Bash tool for pheromone-read (applies decay calculation).
- Use Bash tool only for aether-utils.sh commands and git commands
- Handle ALL missing/corrupted file cases gracefully
- Time-agnostic: restore identically regardless of how long ago the session was
- Decisions shown as flat list — no user vs Claude distinction
- Blocking guards run BEFORE dashboard rendering (early-return pattern)
- Drift detection is informational only — not alarming, not a blocker
