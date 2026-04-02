<!-- Generated from .aether/commands/run.yaml - DO NOT EDIT DIRECTLY -->
---
name: ant:run
description: "🤖🐜🔄🐜🤖 Autopilot — builds, verifies, learns, and advances through phases automatically with smart pausing"
---

### Step -1: Normalize Arguments

Run: `normalized_args=$(bash .aether/aether-utils.sh normalize-args "$@")`

This ensures arguments work correctly in both Claude Code and OpenCode. Use `$normalized_args` throughout this command.

You are the **Queen**. Execute `/ant:run` — the adaptive autopilot loop.

The arguments are: `$normalized_args`

## Purpose

This command automates the build-continue-advance cycle across multiple phases.
It reads and executes the same playbooks used by `/ant:build` and `/ant:continue`,
chaining them in a loop with intelligent pause conditions.

## Rules

1. Do **not** invoke nested slash commands (`/ant:build`, `/ant:continue`, etc.).
2. Use the Read tool to load each playbook file, then execute it inline.
3. Preserve variables/results from prior stages and pass them forward.
4. Stop immediately on any pause condition (defined below).
5. Keep existing behavior and output format from the playbooks.
6. Log `autopilot_advance` events after each successful phase transition.

## Arguments

Parse `$normalized_args` for:
- `--max-phases N` — Max phases to process (default: all remaining)
- `--replan-interval N` — Pause for replan suggestion every N phases (default: 2)
- `--continue` — Resume after a replan pause without replanning
- `--dry-run` — Preview plan without executing
- `--headless` — Run without interactive prompts; queue decisions for later review
- `--verbose` / `-v`, `--no-visual`, `--no-suggest` — Pass through to playbooks

```
/ant:run                       Run all remaining phases
/ant:run --max-phases 2        Run at most 2 phases then stop
/ant:run --replan-interval 3   Suggest replan every 3 phases instead of 2
/ant:run --continue            Resume after replan pause without replanning
/ant:run --dry-run             Preview the autopilot plan
/ant:run --headless            Run all phases without interactive prompts
/ant:run --max-phases 3 -v     Run 3 phases with verbose output
```

## Dry Run Mode

If `--dry-run`: read COLONY_STATE.json, list remaining incomplete phases
(applying `--max-phases` cap), display the plan, then stop without executing.

```
━━━ 🤖 A U T O P I L O T   P R E V I E W ━━━
Goal: {goal} | Current: Phase {N} | Remaining: {count} | Max: {max or "all"}

  Phase {id}: {name} ({task_count} tasks) -> build -> continue -> advance
  ...

Pause triggers: test failures, critical Chaos findings, new blockers,
security gate failures, quality gate failures, runtime verification needed,
replan suggestion (every {replan_interval} phases)
```

## Autopilot Loop

### Step 0: Initialize

1. Read `.aether/data/COLONY_STATE.json`; validate goal + plan.phases exist
   - If `milestone` == `"Crowned Anthill"`: output "This colony has been sealed. Start a new colony with `/ant:init \"new goal\"`." and stop
2. Determine remaining incomplete phases; apply `--max-phases` cap
3. Set `phases_completed = 0`, `autopilot_start = $(date +%s)`
4. Record pre-build blocker count: `bash .aether/aether-utils.sh flag-check-blockers {phase}`
5. If `--headless` flag is present:
   - Run: `bash .aether/aether-utils.sh autopilot-set-headless true`
   - Display: `Headless mode: ON — interactive prompts will be queued as pending decisions`
6. Display: `AUTOPILOT ENGAGED | Goal: {goal} | Phase {N} | Max: {max or "all"}`

### Step 1: Build Phase

Execute build playbooks in order, carrying cross-stage state
(`phase_id`, `visual_mode`, `verbose_mode`, `suggest_enabled`,
`prompt_section`, `wave_results`,
`verification_status`, `synthesis_status`, `next_action`):

1. `.aether/docs/command-playbooks/build-prep.md`
2. `.aether/docs/command-playbooks/build-context.md`
3. `.aether/docs/command-playbooks/build-wave.md`
4. `.aether/docs/command-playbooks/build-verify.md`
5. `.aether/docs/command-playbooks/build-complete.md`

Capture the synthesis result for pause evaluation.

### Step 2: Build Pause Check

**PAUSE if ANY of these are true** (display reason, log event, STOP):

| # | Condition | Source |
|---|-----------|--------|
| 1 | Watcher `verification_passed == false` | build-verify synthesis |
| 2 | Any Chaos finding severity `critical` or `high` | build-verify synthesis |
| 3 | New blocker flags created (count increased since Step 0.4) | flag-check-blockers |

On pause, display the AUTOPILOT PAUSED banner with reason, affected phase,
specific issues, and instruction: "Fix issues, then run /ant:run to resume."
Log: `"<timestamp>|autopilot_paused|run|Paused at Phase {id}: {reason}"`

**Headless override for visual checkpoints:** If headless mode is active and a
visual checkpoint prompt would normally be shown to the user, instead queue it as
a pending decision:
```bash
bash .aether/aether-utils.sh pending-decision-add \
  --title "Visual checkpoint: Phase {id}" \
  --type "checkpoint" \
  --description "{checkpoint_description}" \
  --phase "{id}" \
  --source "build-pause-check"
```
Then continue without pausing.

**If no pause:** proceed to Step 3.

### Step 3: Continue (Verify + Advance)

Execute continue playbooks in order, carrying cross-stage state
(`visual_mode`, `state`, `current_phase`, `verification_results`,
`gate_results`, `advancement_result`, `next_phase_id`, `completion_state`):

1. `.aether/docs/command-playbooks/continue-verify.md`
2. `.aether/docs/command-playbooks/continue-gates.md`
3. `.aether/docs/command-playbooks/continue-advance.md`
4. `.aether/docs/command-playbooks/continue-finalize.md`

**Autopilot override for runtime verification (Step 1.11 in continue-gates):**
Skip the AskUserQuestion prompt. Instead, auto-PAUSE with reason
"Runtime verification required" so the user can test manually before resuming.

**Headless override for runtime verification:** If headless mode is active and
runtime verification would normally pause, queue as a pending decision instead:
```bash
bash .aether/aether-utils.sh pending-decision-add \
  --title "Runtime verification needed: Phase {id}" \
  --type "runtime-verification" \
  --description "Manual testing required before advancing past Phase {id}" \
  --phase "{id}" \
  --source "continue-gates"
```
Then continue without pausing.

### Step 4: Continue Pause Check

**PAUSE if ANY of these are true:**

| # | Condition | Source |
|---|-----------|--------|
| 4 | Verification loop reported NOT READY | continue-verify |
| 5 | Gatekeeper found critical CVEs | continue-gates |
| 6 | Auditor critical findings or score < 60 | continue-gates |
| 7 | Unresolved blocker flags remain | continue-gates |
| 8 | Runtime verification needed | continue-gates Step 1.11 |
| 9 | All phases complete (no next phase) | continue-advance |
| 10 | Replan trigger fires (unless `--continue`) | autopilot-check-replan (Step 5.5) |

For condition 9: jump to Step 6 (celebration). For condition 10: PAUSE with
replan suggestion (see Step 5.5). For all others: PAUSE with reason, log event, STOP.

**If no pause:** proceed to Step 5.

### Step 5: Auto-Advance and Loop

1. Increment `phases_completed`
2. Update autopilot state: `bash .aether/aether-utils.sh autopilot-update --action advance --phase {next} --result success`
3. Log: `"<timestamp>|autopilot_advance|run|Phase {prev} -> {next} ({phases_completed}/{max})"`
4. Display: `--- Autopilot: Phase {prev} done -> Phase {next} ({N}/{max}) ---`
5. **Replan check** (see Step 5.5)
6. If `phases_completed >= max_phases` or no incomplete phases: go to Step 6
7. Otherwise: update `current_phase`, return to Step 1

### Step 5.5: Replan Trigger Check

After each successful phase advance, check if a replan pause should fire:

```bash
bash .aether/aether-utils.sh autopilot-check-replan --interval {replan_interval}
```

If `--continue` flag was passed: skip this check entirely (user dismissed replan).

If `result.should_replan == true`: **PAUSE** with replan suggestion banner:

```
━━━ 🔄 R E P L A N   S U G G E S T E D ━━━
Phases auto-completed: {N} | Learnings accumulated: {learnings_since_last}

The colony has completed {N} phases since the last plan review.
New learnings may have changed the optimal path forward.

Options:
  /ant:plan              Regenerate phases with current learnings
  /ant:run --continue    Dismiss and continue without replanning
```

Log: `"<timestamp>|autopilot_replan_pause|run|Replan suggested after {N} phases ({learnings} learnings)"`

If `result.should_replan == false`: proceed normally (no pause).

### Step 6: Final Summary

```
━━━ ✅ A U T O P I L O T   C O M P L E T E ━━━
Phases completed: {N} | Elapsed: {Xm Ys} | Now at: Phase {current}

{all complete}  -> Colony goal achieved! Run /ant:seal
{max reached}   -> Run /ant:run to continue
{replan}        -> Run /ant:plan to replan, or /ant:run --continue to dismiss
{paused}        -> Fix {reason}, then /ant:run to resume
```

If headless mode was active and pending decisions were queued, display:
```
Pending decisions: {N} — run `pending-decision-list` to review
```

Update session:
`bash .aether/aether-utils.sh session-update "/ant:run" "/ant:run" "Autopilot: {N} phases, now Phase {current}"`

## Execution Contract

For each playbook stage: Read file, execute inline, track `{stage_name, status, key_outputs}`.
If `status == failed`: evaluate pause conditions. Pause = graceful stop with saved state.
Hard failure (e.g., state corruption) = halt immediately, no recovery attempt.

On every pause: save COLONY_STATE.json, log event, update session, display resume instructions.
