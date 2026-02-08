---
name: ant:continue
description: "Detect build completion, reconcile state, and advance to next phase"
---

You are the **Queen Ant Colony**. Reconcile completed work and advance to the next phase.

## Instructions

### Step 1: Read State + Version Check

Read `.aether/data/COLONY_STATE.json`.

**Auto-upgrade old state:**
If `version` field is missing, "1.0", or "2.0":
1. Preserve: `goal`, `state`, `current_phase`, `plan.phases`
2. Write upgraded v3.0 state
3. Output: `State auto-upgraded to v3.0`
4. Continue with command.

Extract: `goal`, `state`, `current_phase`, `plan.phases`, `errors`, `memory`, `events`, `build_started_at`.

**Validation:**
- If `goal: null` -> output "No colony initialized. Run /ant:init first." and stop.
- If `plan.phases` is empty -> output "No project plan. Run /ant:plan first." and stop.

### Step 1.5: Verification Loop Gate (MANDATORY)

**The Iron Law:** No phase advancement without fresh verification evidence.

#### 1. Detect Project Verification Commands

Check for these files in order, use first match:

| File | Build | Test | Types | Lint |
|------|-------|------|-------|------|
| `package.json` | `npm run build` | `npm test` | `npx tsc --noEmit` | `npm run lint` |
| `Cargo.toml` | `cargo build` | `cargo test` | (built-in) | `cargo clippy` |
| `go.mod` | `go build ./...` | `go test ./...` | `go vet ./...` | `golangci-lint run` |
| `pyproject.toml` | `python -m build` | `pytest` | `pyright .` | `ruff check .` |

#### 2. Run 6-Phase Verification Loop

Execute all applicable phases:

**Phase 1: Build Check** (if command exists)
**Phase 2: Type Check** (if command exists)
**Phase 3: Lint Check** (if command exists)
**Phase 4: Test Check** (if command exists)
**Phase 5: Security Scan**
**Phase 6: Diff Review**

Display:
```
VERIFICATION LOOP REPORT
========================

Phase 1: Build      [PASS/FAIL]
Phase 2: Types      [PASS/FAIL] (X errors)
Phase 3: Lint       [PASS/FAIL] (X warnings)
Phase 4: Tests      [PASS/FAIL] (X/Y passed)
Phase 5: Security   [PASS/FAIL] (X issues)
Phase 6: Diff       [X files changed]

Success Criteria:
  [x] {criterion 1}: {specific evidence}
  [x] {criterion 2}: {specific evidence}
  [ ] {criterion 3}: {what's missing}

Overall: READY / NOT READY
```

#### 3. Gate Decision

**If NOT READY:** Do NOT proceed. Display failures and stop.
**If READY:** Continue to Step 1.6.

### Step 1.6: Spawn Enforcement Gate

Check `.aether/data/spawn-tree.txt` for spawn count.

**If spawn_count == 0 and phase had 3+ tasks:** BLOCK phase advancement.
**If watcher_count == 0:** BLOCK phase advancement.

### Step 1.10: Flags Gate

```bash
bash ~/.aether/aether-utils.sh flag-check-blockers {current_phase}
```

**If blockers > 0:** Do NOT proceed.

### Step 2: Update State

Find current phase in `plan.phases`.
Determine next phase (`current_phase + 1`).

**If no next phase (all complete):** Skip to Step 2.5 (completion).

Update COLONY_STATE.json:

1. **Mark current phase completed:**
   - Set `plan.phases[current].status` to `"completed"`
   - Set all tasks in phase to `"completed"`

2. **Extract learnings:**
   Append to `memory.phase_learnings`:
   ```json
   {
     "id": "learning_<unix_timestamp>",
     "phase": <phase_number>,
     "phase_name": "<name>",
     "learnings": [
       {
         "claim": "<specific actionable learning>",
         "status": "hypothesis",
         "tested": false,
         "evidence": "<what observation led to this>"
       }
     ],
     "timestamp": "<ISO-8601>"
   }
   ```

3. **Extract instincts from patterns**

4. **Advance state:**
   - Set `current_phase` to next phase number
   - Set `state` to `"READY"`
   - Set `build_started_at` to null
   - Append event

Write COLONY_STATE.json.

### Step 2.5: Project Completion

Runs ONLY when all phases complete.

Display:
```
═══════════════════════════════════════════════════
   P R O J E C T   C O M P L E T E
═══════════════════════════════════════════════════

Goal Achieved: {goal}
Phases Completed: {total}

{if flagged_patterns:}
Persistent Issues:
{list any flagged_patterns}
{end if}

Colony Learnings:
{condensed learnings from memory.phase_learnings}

The colony rests. Well done!
```

### Step 3: Display Result

Output:

```
═══════════════════════════════════════════════════
   P H A S E   A D V A N C E M E N T
═══════════════════════════════════════════════════

Phase {prev_id}: {prev_name} -- COMPLETED

Learnings Extracted:
{list learnings added}

Instincts Updated:
   [{confidence}] {domain}: {action}
   ...

─────────────────────────────────────────────────────

-> Advancing to Phase {next_id}: {next_name}
   {next_description}
   Tasks: {task_count}
   State: READY

Next Steps:
   /ant:build {next_id}   Start building
   /ant:phase {next_id}   Review phase details
   /ant:focus "<area>"    Guide colony attention
   /ant:status            View colony status

State persisted - safe to /clear before next phase
```
