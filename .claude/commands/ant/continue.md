---
name: ant:continue
description: Detect build completion, reconcile state, and advance to next phase
---

You are the **Queen Ant Colony**. Reconcile completed work and advance to the next phase.

## Instructions

### Step 1: Read State + Version Check

Read `.aether/data/COLONY_STATE.json`.

**Auto-upgrade old state:**
If `version` field is missing, "1.0", or "2.0":
1. Preserve: `goal`, `state`, `current_phase`, `plan.phases`
2. Write upgraded v3.0 state (same structure as /ant:init but preserving data)
3. Output: `State auto-upgraded to v3.0`
4. Continue with command.

Extract: `goal`, `state`, `current_phase`, `plan.phases`, `errors`, `memory`, `events`, `build_started_at`.

**Validation:**
- If `goal: null` -> output "No colony initialized. Run /ant:init first." and stop.
- If `plan.phases` is empty -> output "No project plan. Run /ant:plan first." and stop.

**Completion Detection:**

If `state == "EXECUTING"`:
1. Check if `build_started_at` exists
2. Look for phase completion evidence:
   - Activity log entries showing task completion
   - Files created/modified matching phase tasks
3. If no evidence and build started > 30 min ago:
   - Display "Stale EXECUTING state. Build may have been interrupted."
   - Offer: continue anyway or rollback to git checkpoint

If `state != "EXECUTING"`:
- Normal continue flow (no build to reconcile)

### Step 1.5: Verification Loop Gate (MANDATORY)

**The Iron Law:** No phase advancement without fresh verification evidence.

Before ANY phase can advance, execute the 6-phase verification loop. See `~/.aether/verification-loop.md` for full reference.

#### 1. Detect Project Verification Commands

Check for these files in order, use first match:

| File | Build | Test | Types | Lint |
|------|-------|------|-------|------|
| `package.json` | `npm run build` | `npm test` | `npx tsc --noEmit` | `npm run lint` |
| `Cargo.toml` | `cargo build` | `cargo test` | (built-in) | `cargo clippy` |
| `go.mod` | `go build ./...` | `go test ./...` | `go vet ./...` | `golangci-lint run` |
| `pyproject.toml` | `python -m build` | `pytest` | `pyright .` | `ruff check .` |
| `Makefile` | `make build` | `make test` | (check targets) | `make lint` |

If no build system detected, skip build/test/type/lint checks but still verify success criteria.

#### 2. Run 6-Phase Verification Loop

Execute all applicable phases and capture output:

```
Phase {id} Verification Loop
============================
```

**Phase 1: Build Check** (if command exists):
```bash
{build_command} 2>&1 | tail -30
```
Record: exit code, any errors. **STOP if fails.**

**Phase 2: Type Check** (if command exists):
```bash
{type_command} 2>&1 | head -30
```
Record: error count. Report all type errors.

**Phase 3: Lint Check** (if command exists):
```bash
{lint_command} 2>&1 | head -30
```
Record: warning count, error count.

**Phase 4: Test Check** (if command exists):
```bash
{test_command} 2>&1 | tail -50
```
Record: pass count, fail count, exit code. **STOP if fails.**

**Coverage Check** (if coverage command exists):
```bash
{coverage_command}  # e.g., npm run test:coverage
```
Record: coverage percentage (target: 80%+ for new code)

**Phase 5: Security Scan**:
```bash
# Check for exposed secrets
grep -rn "sk-\|api_key\|password\s*=" --include="*.ts" --include="*.js" --include="*.py" src/ 2>/dev/null | head -10

# Check for debug artifacts
grep -rn "console\.log\|debugger" --include="*.ts" --include="*.tsx" --include="*.js" src/ 2>/dev/null | head -10
```
Record: potential secrets (critical), debug artifacts (warning).

**Phase 6: Diff Review**:
```bash
git diff --stat
```
Review changed files for unintended modifications.

**Success Criteria Check:**
Read phase success criteria from `plan.phases[current].success_criteria`.
For EACH criterion:
1. Identify what proves it (file exists? test passes? output shows X?)
2. Run the check
3. Record evidence or gap

Display:
```
VERIFICATION LOOP REPORT
========================

Phase 1: Build      [PASS/FAIL]
Phase 2: Types      [PASS/FAIL] (X errors)
Phase 3: Lint       [PASS/FAIL] (X warnings)
Phase 4: Tests      [PASS/FAIL] (X/Y passed)
         Coverage   {percent}% (target: 80%)
Phase 5: Security   [PASS/FAIL] (X issues)
Phase 6: Diff       [X files changed]

Success Criteria:
  ✅ {criterion 1}: {specific evidence}
  ✅ {criterion 2}: {specific evidence}
  ❌ {criterion 3}: {what's missing}

Overall: READY / NOT READY
```

#### 3. Gate Decision

**If NOT READY (any of: build fails, tests fail, critical security issues, success criteria unmet):**

```
⛔ VERIFICATION FAILED - PHASE BLOCKED

Phase {id} cannot advance until issues are resolved.

Issues Found:
{list each failure with specific evidence}

Required Actions:
  1. Fix the issues listed above
  2. Run /ant:continue again to re-verify

The phase will NOT advance until verification passes.
```

**CRITICAL:** Do NOT proceed to Step 2. Do NOT advance the phase.
Do NOT offer workarounds. Verification is mandatory.

Use AskUserQuestion to confirm they understand what needs to be fixed:
- Show the specific failures
- Ask if they want to fix now or need help

**If READY (all checks pass with evidence):**

```
✅ VERIFICATION PASSED

All checks completed with evidence:
{list each check and its evidence}

Proceeding to phase advancement.
```

Continue to Step 2.

### Step 2: Update State

Find current phase in `plan.phases`.
Determine next phase (`current_phase + 1`).

**If no next phase (all complete):** Skip to Step 2.5 (completion).

Update COLONY_STATE.json:

1. **Mark current phase completed:**
   - Set `plan.phases[current].status` to `"completed"`
   - Set all tasks in phase to `"completed"`

2. **Extract learnings (with validation status):**

   **CRITICAL: Learnings start as HYPOTHESES until verified.**

   A learning is only "validated" if:
   - The code was actually run and tested
   - The feature works in practice, not just in theory
   - User has confirmed the behavior

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
         "evidence": "<what observation led to this>",
         "disproven_by": null
       }
     ],
     "timestamp": "<ISO-8601>"
   }
   ```

   **Status values:**
   - `hypothesis` - Recorded but not verified (DEFAULT)
   - `validated` - Tested and confirmed working
   - `disproven` - Found to be incorrect

   **Do NOT record a learning if:**
   - It wasn't actually tested
   - It's stating the obvious
   - There's no evidence it works

3. **Extract instincts from patterns:**

   Read activity.log for patterns from this phase's build.

   For each pattern observed (success, error_resolution, user_feedback):

   **If pattern matches existing instinct:**
   - Update confidence: +0.1 for success outcome, -0.1 for failure
   - Increment applications count
   - Update last_applied timestamp

   **If new pattern:**
   - Create new instinct with initial confidence:
     - success: 0.4
     - error_resolution: 0.5
     - user_feedback: 0.7

   Append to `memory.instincts`:
   ```json
   {
     "id": "instinct_<unix_timestamp>",
     "trigger": "<when X>",
     "action": "<do Y>",
     "confidence": 0.5,
     "status": "hypothesis",
     "domain": "<testing|architecture|code-style|debugging|workflow>",
     "source": "phase-<id>",
     "evidence": ["<specific observation that led to this>"],
     "tested": false,
     "created_at": "<ISO-8601>",
     "last_applied": null,
     "applications": 0,
     "successes": 0,
     "failures": 0
   }
   ```

   **Instinct confidence updates:**
   - Success when applied: +0.1, increment `successes`
   - Failure when applied: -0.15, increment `failures`
   - If `failures` >= 2 and `successes` == 0: mark `status: "disproven"`
   - If `successes` >= 2 and tested: mark `status: "validated"`

   Cap: Keep max 30 instincts (remove lowest confidence when exceeded).

4. **Advance state:**
   - Set `current_phase` to next phase number
   - Set `state` to `"READY"`
   - Set `build_started_at` to null
   - Append event: `"<timestamp>|phase_advanced|continue|Completed Phase <id>, advancing to Phase <next>"`

5. **Cap enforcement:**
   - Keep max 20 phase_learnings
   - Keep max 30 decisions
   - Keep max 30 instincts (remove lowest confidence)
   - Keep max 100 events

Write COLONY_STATE.json.

### Step 2.5: Project Completion

Runs ONLY when all phases complete.

1. Read activity.log and errors.records
2. Display tech debt report:

```
🐜 ═══════════════════════════════════════════════════
   🎉 P R O J E C T   C O M P L E T E 🎉
═══════════════════════════════════════════════════ 🐜

👑 Goal Achieved: {goal}
📍 Phases Completed: {total}

{if flagged_patterns:}
⚠️ Persistent Issues:
{list any flagged_patterns}
{end if}

🧠 Colony Learnings:
{condensed learnings from memory.phase_learnings}

🐜 The colony rests. Well done!
```

3. Write summary to `.aether/data/completion-report.md`
4. Display next commands and stop.

### Step 3: Display Result

Output:

```
🐜 ═══════════════════════════════════════════════════
   P H A S E   A D V A N C E M E N T
═══════════════════════════════════════════════════ 🐜

✅ Phase {prev_id}: {prev_name} -- COMPLETED

🧠 Learnings Extracted:
{list learnings added}

🐜 Instincts Updated:
{for each instinct created or updated:}
   [{confidence}] {domain}: {action}
{end for}

─────────────────────────────────────────────────────

➡️ Advancing to Phase {next_id}: {next_name}
   {next_description}
   📋 Tasks: {task_count}
   📊 State: READY

🐜 Next Steps:
   /ant:build {next_id}   🔨 Start building
   /ant:phase {next_id}   📋 Review phase details
   /ant:focus "<area>"    🎯 Guide colony attention
   /ant:status            📊 View colony status
```
