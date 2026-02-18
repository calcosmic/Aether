---
name: resume
description: "Resume Previous Session"
symbol: refresh
---

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

Run using the Bash tool with description "Restoring colony session...":
```bash
bash .aether/aether-utils.sh session-read
```

Parse the JSON result.

- If `exists` is `false`: display the following and **stop**:

```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
RESUME SESSION

No previous session found.

Start fresh: /ant:init "your goal"
Or check: /ant:status
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
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
- `state` (READY, PLANNING, EXECUTING, PAUSED)
- `current_phase`
- `plan.phases` (array with id, name, status for each phase)
- `plan.generated_at`
- `memory.decisions` (flat list — do NOT distinguish user vs Claude origin)
- `events` (last 5 for recent activity context)

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

Use the Read tool to read `.aether/data/constraints.json`.

Extract the following top-level keys:
- `focus` array — active focus signals (if key missing, treat as empty array)
- `constraints` array — active redirect/constraint signals (if key missing, treat as empty array)

If the file is missing: skip silently (no pheromones active).

Pheromones persist until explicitly cleared — no decay.

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
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
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

{if focus array or constraints array is not empty:}
Active Signals:
{for each focus signal:}
  FOCUS: {focus text}
{end}
{for each constraint signal:}
  REDIRECT: {constraint text}
{end}
{end}

Last Command: {last_command}
Session: {session_id}
```

---

### Step 9: Mark Session Resumed

Run using the Bash tool with description "Marking session as resumed...":
```bash
bash .aether/aether-utils.sh session-mark-resumed
```

### Step 10: Next Up

Generate the state-based Next Up block by running using the Bash tool with description "Generating Next Up suggestions...":
```bash
state=$(jq -r '.state // "IDLE"' .aether/data/COLONY_STATE.json)
current_phase=$(jq -r '.current_phase // 0' .aether/data/COLONY_STATE.json)
total_phases=$(jq -r '.plan.phases | length' .aether/data/COLONY_STATE.json)
bash .aether/aether-utils.sh print-next-up "$state" "$current_phase" "$total_phases"
```

---

## Error Handling Reference

| Condition | Response |
|-----------|----------|
| session.json missing (exists=false) | "No previous session found" — offer /ant:init and /ant:status |
| COLONY_STATE.json missing or corrupted | Pause, ask user: start fresh or recover |
| constraints.json missing | Skip silently (no pheromones) |
| CONTEXT.md missing | Fall back to COLONY_STATE.json narrative |
| No plan phases, no generated_at | BLOCK — redirect to /ant:plan |
| Plan attempted but no phases | BLOCK — redirect to /ant:plan |
| State EXECUTING, events show no activity | BLOCK — redirect to /ant:continue |
| baseline_commit matches current HEAD | No drift warning shown |
| baseline_commit differs from current HEAD | Show informational drift note |

---

## Key Constraints

- Use Read tool for COLONY_STATE.json and constraints.json (not bash cat/jq)
- Use Bash tool only for aether-utils.sh commands and git commands
- Handle ALL missing/corrupted file cases gracefully
- Time-agnostic: restore identically regardless of how long ago the session was
- Decisions shown as flat list — no user vs Claude distinction
- Blocking guards run BEFORE dashboard rendering (early-return pattern)
- Drift detection is informational only — not alarming, not a blocker
