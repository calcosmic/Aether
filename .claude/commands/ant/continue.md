---
name: ant:continue
description: "➡️🐜🚪🐜➡️ Detect build completion, reconcile state, and advance to next phase"
---

You are the **Queen Ant Colony**. Reconcile completed work and advance to the next phase.

## Instructions

Parse `$ARGUMENTS`:
- If contains `--no-visual`: set `visual_mode = false` (visual is ON by default)
- Otherwise: set `visual_mode = true`

### Step 0: Initialize Visual Mode (if enabled)

If `visual_mode` is true:
Run using the Bash tool with description "Initializing continue display...": `continue_id="continue-$(date +%s)" && bash .aether/aether-utils.sh swarm-display-init "$continue_id" && bash .aether/aether-utils.sh swarm-display-update "Queen" "prime" "excavating" "Phase continuation" "Colony" '{"read":0,"grep":0,"edit":0,"bash":0}' 0 "fungus_garden" 0`

### Step 0.5: Version Check (Non-blocking)

Run using the Bash tool with description "Checking colony version...": `bash .aether/aether-utils.sh version-check-cached 2>/dev/null || true`

If the command succeeds and the JSON result contains a non-empty string, display it as a one-line notice. Proceed regardless of outcome.

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

### Step 1.5: Load State and Show Resumption Context

Run using the Bash tool with description "Loading colony state...": `bash .aether/aether-utils.sh load-state`

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

Run using the Bash tool with description "Releasing colony lock...": `bash .aether/aether-utils.sh unload-state` to release lock.

**Error handling:**
- If E_FILE_NOT_FOUND: "No colony initialized. Run /ant:init first." and stop
- If validation error: Display error details with recovery suggestion and stop
- For other errors: Display generic error and suggest /ant:status for diagnostics

**Completion Detection:**

If `state == "EXECUTING"`:
1. Check if `build_started_at` exists
2. Look for phase completion evidence:
   - Activity log entries showing task completion
   - Files created/modified matching phase tasks
3. If no evidence and build started > 30 min ago:
   - Display "Stale EXECUTING state. Build may have been interrupted."
   - Offer: continue anyway or rollback to git checkpoint
   - Rollback procedure: `git stash list | grep "aether-checkpoint"` to find ref, then `git stash pop <ref>` to restore

If `state != "EXECUTING"`:
- Normal continue flow (no build to reconcile)

### Step 1.5: Verification Loop Gate (MANDATORY)

**The Iron Law:** No phase advancement without fresh verification evidence.

Before ANY phase can advance, execute the 6-phase verification loop. See `.aether/verification-loop.md` for full reference.

#### 1. Command Resolution (Priority Chain)

Resolve each command (build, test, types, lint) using this priority chain. Stop at the first source that provides a value for each command:

**Priority 1 — CLAUDE.md (System Context):**
Check the CLAUDE.md instructions already loaded in your system context for explicit build, test, type-check, or lint commands. These are authoritative and override all other sources.

**Priority 2 — codebase.md `## Commands`:**
Read `.aether/data/codebase.md` and look for the `## Commands` section. Use any commands listed there for slots not yet filled by Priority 1.

**Priority 3 — Fallback Heuristic Table:**
For any commands still unresolved, check for these files in order, use first match:

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
Run using the Bash tool with description "Running build check...": `{build_command} 2>&1 | tail -30`
Record: exit code, any errors. **STOP if fails.**

**Phase 2: Type Check** (if command exists):
Run using the Bash tool with description "Running type check...": `{type_command} 2>&1 | head -30`
Record: error count. Report all type errors.

**Phase 3: Lint Check** (if command exists):
Run using the Bash tool with description "Running lint check...": `{lint_command} 2>&1 | head -30`
Record: warning count, error count.

**Phase 4: Test Check** (if command exists):
Run using the Bash tool with description "Running test suite...": `{test_command} 2>&1 | tail -50`
Record: pass count, fail count, exit code. **STOP if fails.**

**Coverage Check** (if coverage command exists):
Run using the Bash tool with description "Checking test coverage...": `{coverage_command}  # e.g., npm run test:coverage`
Record: coverage percentage (target: 80%+ for new code)

**Phase 5: Security Scan**:
Run using the Bash tool with description "Scanning for exposed secrets...": `grep -rn "sk-\|api_key\|password\s*=" --include="*.ts" --include="*.js" --include="*.py" src/ 2>/dev/null | head -10`
Run using the Bash tool with description "Scanning for debug artifacts...": `grep -rn "console\.log\|debugger" --include="*.ts" --include="*.tsx" --include="*.js" src/ 2>/dev/null | head -10`
Record: potential secrets (critical), debug artifacts (warning).

**Phase 6: Diff Review**:
Run using the Bash tool with description "Reviewing file changes...": `git diff --stat`
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

Continue to Step 1.6.

### Step 1.6: Spawn Enforcement Gate (MANDATORY)

**The Iron Law:** No phase advancement without worker spawning for non-trivial phases.

Read `.aether/data/spawn-tree.txt` to count spawns for this phase.

Run using the Bash tool with description "Verifying spawn requirements...": `spawn_count=$(grep -c "spawned" .aether/data/spawn-tree.txt 2>/dev/null || echo "0") && watcher_count=$(grep -c "watcher" .aether/data/spawn-tree.txt 2>/dev/null || echo "0") && echo "{\"spawn_count\": $spawn_count, \"watcher_count\": $watcher_count}"`

**HARD REJECTION - If spawn_count == 0 and phase had 3+ tasks:**

```
⛔ SPAWN GATE FAILED - PHASE BLOCKED

This phase had {task_count} tasks but spawn_count: 0
The Prime Worker violated the spawn protocol.

The colony metaphor requires actual parallelism:
  - Prime Worker MUST spawn specialists for non-trivial work
  - A single agent doing everything is NOT a colony
  - "Justifications" for not spawning are not accepted

Required Actions:
  1. Run /ant:build {phase} again
  2. Prime Worker MUST spawn at least 1 specialist
  3. Re-run /ant:continue after spawns complete

The phase will NOT advance until spawning occurs.
```

**CRITICAL:** Do NOT proceed to Step 1.7. Do NOT advance the phase.
Log the violation:
```bash
bash .aether/aether-utils.sh activity-log "BLOCKED" "colony" "Spawn gate failed: {task_count} tasks, 0 spawns"
bash .aether/aether-utils.sh error-flag-pattern "no-spawn-violation" "Prime Worker completed phase without spawning specialists" "critical"
```

**HARD REJECTION - If watcher_count == 0 (no testing separation):**

```
⛔ WATCHER GATE FAILED - PHASE BLOCKED

No Watcher ant was spawned for testing/verification.
Testing MUST be performed by a separate agent, not the builder.

Why this matters:
  - Builders verify their own work = confirmation bias
  - Independent Watchers catch bugs builders miss
  - "Build passing" ≠ "App working"

Required Actions:
  1. Run /ant:build {phase} again
  2. Prime Worker MUST spawn at least 1 Watcher
  3. Watcher must independently verify the work

The phase will NOT advance until a Watcher validates.
```

**CRITICAL:** Do NOT proceed. Log the violation.

**If spawn_count >= 1 AND watcher_count >= 1:**

```
✅ SPAWN GATE PASSED

Spawns: {spawn_count} workers
Watchers: {watcher_count} (independent verification)

Proceeding to runtime verification.
```

Continue to Step 1.7.

### Step 1.7: Anti-Pattern Gate

Scan all modified/created files for known anti-patterns. This catches recurring bugs before they reach production.

For each file, run using the Bash tool with description "Scanning for anti-patterns...": `bash .aether/aether-utils.sh check-antipattern "{file_path}"`

Run for each file in `files_created` and `files_modified` from Prime Worker output.

**Anti-Pattern Report:**

```
Anti-Pattern Scan
=================
Files scanned: {count}

{if critical issues:}
🛑 CRITICAL ISSUES (must fix):
{list each with file:line and description}

{if warnings:}
⚠️ WARNINGS (review recommended):
{list each with file:line and description}

{if clean:}
✅ No anti-patterns detected
```

**CRITICAL issues block phase advancement:**
- Swift didSet infinite recursion
- Exposed secrets/credentials
- SQL injection patterns
- Known crash patterns

**WARNINGS are logged but don't block:**
- TypeScript `any` usage
- Console.log in production code
- TODO/FIXME comments

If CRITICAL issues found, display:

```
⛔ ANTI-PATTERN GATE FAILED

Critical anti-patterns detected that must be fixed:
{list issues with file paths}

Run /ant:build {phase} again after fixing.
```

Do NOT proceed to Step 2.

If no CRITICAL issues, continue to Step 1.8.

### Step 1.8: TDD Evidence Gate (MANDATORY)

**The Iron Law:** No TDD claims without actual test files.

If Prime Worker reported TDD metrics (tests_added, tests_total, coverage_percent), verify test files exist:

Run using the Bash tool with description "Locating test files...": `find . -name "*.test.*" -o -name "*_test.*" -o -name "*Tests.swift" -o -name "test_*.py" 2>/dev/null | head -10`

**If Prime Worker claimed tests_added > 0 but no test files found:**

```
⛔ TDD GATE FAILED - FABRICATED METRICS

Prime Worker claimed:
  tests_added: {claimed_count}
  tests_total: {claimed_total}
  coverage_percent: {claimed_coverage}%

But no test files were found in the codebase.

This is a CRITICAL violation:
  - TDD metrics were fabricated
  - No actual tests were written
  - "All passing: true" was a lie

Required Actions:
  1. Run /ant:build {phase} again
  2. Actually write test files (not just claim them)
  3. Tests must exist and be runnable

The phase will NOT advance with fabricated metrics.
```

**CRITICAL:** Do NOT proceed. Log the violation:
```bash
bash .aether/aether-utils.sh error-flag-pattern "fabricated-tdd" "Prime Worker reported TDD metrics without creating test files" "critical"
```

**If tests_added == 0 or test files exist matching claims:**

Continue to Step 1.9.

### Step 1.9: Runtime Verification Gate (MANDATORY)

**The Iron Law:** Build passing ≠ App working.

Before advancing, the user must confirm the application actually runs.

Use AskUserQuestion:

```
Runtime Verification Required
=============================

Build and compile checks passed, but we need to verify the app actually works.

Have you tested the application at runtime?
```

Options:
1. **Yes, tested and working** - App runs correctly, features work
2. **Yes, tested but has issues** - App runs but has bugs (describe)
3. **No, haven't tested yet** - Need to test before continuing
4. **Skip (not applicable)** - No runnable app in this phase (e.g., library code)

**If "Yes, tested and working":**
```
✅ RUNTIME VERIFICATION PASSED

User confirmed application runs correctly.
Proceeding to phase advancement.
```
Continue to Step 2.

**If "Yes, tested but has issues":**
```
⛔ RUNTIME GATE FAILED

User reported runtime issues. The phase cannot advance with a broken app.

Please describe the issues so they can be addressed:
```

Use AskUserQuestion to get issue details. Log to errors.records:
```bash
bash .aether/aether-utils.sh error-add "runtime" "critical" "{user_description}" {phase}
```

Do NOT proceed to Step 2.

**If "No, haven't tested yet":**
```
⏸️ RUNTIME VERIFICATION PENDING

Please test the application and run /ant:continue again.

Testing checklist:
  - [ ] App launches without crashing
  - [ ] Core features work as expected
  - [ ] UI responds to user interaction
  - [ ] No freezes or hangs

Come back when you've tested.
```

Do NOT proceed to Step 2.

**If "Skip (not applicable)":**

Only valid for phases that don't produce runnable code (e.g., documentation, config files, library code with no entry point).

```
⏭️ RUNTIME CHECK SKIPPED

User indicated no runnable app for this phase.
Proceeding to phase advancement.
```

Continue to Step 1.10.

### Step 1.10: Flags Gate (MANDATORY)

**The Iron Law:** No phase advancement with unresolved blockers.

First, auto-resolve any flags eligible for resolution now that verification has passed:
Run using the Bash tool with description "Auto-resolving flags...": `bash .aether/aether-utils.sh flag-auto-resolve "build_pass"`

Then check for remaining blocking flags:
Run using the Bash tool with description "Checking for blockers...": `bash .aether/aether-utils.sh flag-check-blockers {current_phase}`

Parse result for `blockers`, `issues`, and `notes` counts.

**If blockers > 0:**

```
⛔ FLAGS GATE FAILED - BLOCKERS ACTIVE

{blockers} blocking flag(s) must be resolved before phase advancement.

Active Blockers:
{list each blocker flag with ID, title, and description}

Required Actions:
  1. Fix the issues described in each blocker
  2. Resolve flags: /ant:flags --resolve {flag_id} "resolution message"
  3. Run /ant:continue again after resolving all blockers

The phase will NOT advance with active blockers.
```

**CRITICAL:** Do NOT proceed to Step 2. Do NOT advance the phase.

**If blockers == 0 but issues > 0:**

```
⚠️ FLAGS GATE: ISSUES NOTED

No blockers, but {issues} issue(s) are active.
These don't block advancement but should be addressed.

Active Issues:
{list each issue flag}

Use /ant:flags to review and acknowledge or resolve.
```

Continue to Step 2.

**If all clear (no blockers or issues):**

```
✅ FLAGS GATE PASSED

No blocking flags. Proceeding to phase advancement.
```

Continue to Step 2.

### Step 2: Update State

Find current phase in `plan.phases`.
Determine next phase (`current_phase + 1`).

**If no next phase (all complete):** Skip to Step 2.6 (commit suggestion), then Step 2.5 (completion).

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

Validate the state file:
Run using the Bash tool with description "Validating colony state...": `bash .aether/aether-utils.sh validate-state colony`

### Step 2.1: Auto-Emit Phase Pheromones (SILENT)

**This entire step produces NO user-visible output.** All pheromone operations run silently — learnings are deposited in the background. If any pheromone call fails, log the error and continue. Phase advancement must never fail due to pheromone errors.

#### 2.1a: Auto-emit FEEDBACK pheromone for phase outcome

After learning extraction completes in Step 2, auto-emit a FEEDBACK signal summarizing the phase:

```bash
# phase_id and phase_name come from Step 2 state update
# Take the top 1-3 learnings by evidence strength from memory.phase_learnings
# Compress into a single summary sentence

# If learnings were extracted, build a brief summary from them (first 1-3 claims)
# Otherwise use the minimal fallback
phase_feedback="Phase $phase_id ($phase_name) completed. Key patterns: {brief summary of 1-3 learnings from Step 2}"
# Fallback if no learnings: "Phase $phase_id ($phase_name) completed without notable patterns."

bash .aether/aether-utils.sh pheromone-write FEEDBACK "$phase_feedback" \
  --strength 0.6 \
  --source "worker:continue" \
  --reason "Auto-emitted on phase advance: captures what worked and what was learned" \
  --ttl "phase_end" 2>/dev/null || true
```

The strength is 0.6 (auto-emitted = lower than user-emitted 0.7). Source is "worker:continue" to distinguish from user-emitted feedback. TTL is "phase_end" so the signal survives through the NEXT phase and expires when THAT phase advances.

#### 2.1b: Auto-emit REDIRECT for recurring error patterns

Check `errors.flagged_patterns[]` in COLONY_STATE.json for patterns that have appeared in 2+ phases:

```bash
flagged_patterns=$(jq -r '.errors.flagged_patterns[]? | select(.count >= 2) | .pattern' .aether/data/COLONY_STATE.json 2>/dev/null || true)
```

For each pattern returned by the above query, emit a REDIRECT signal:

```bash
bash .aether/aether-utils.sh pheromone-write REDIRECT "$pattern_text" \
  --strength 0.7 \
  --source "system" \
  --reason "Auto-emitted: error pattern recurred across 2+ phases" \
  --ttl "30d" 2>/dev/null || true
```

REDIRECT strength is 0.7 (higher than auto FEEDBACK 0.6 — anti-patterns produce stronger signals than successes). TTL is 30d (not phase_end) because recurring errors should persist across multiple phases.

If `errors.flagged_patterns` doesn't exist or is empty, skip silently.

#### 2.1c: Expire phase_end signals and archive to midden

After auto-emission, expire all signals with `expires_at == "phase_end"`. The newly-emitted FEEDBACK from 2.1a will survive this call (it was just written and is active) — it will expire when the NEXT phase advances.

Run using the Bash tool with description "Maintaining pheromone memory...": `bash .aether/aether-utils.sh pheromone-expire --phase-end-only 2>/dev/null && bash .aether/aether-utils.sh eternal-init 2>/dev/null`

This is idempotent — runs every time continue fires but only creates the directory/file once.

### Step 2.2: Promote Validated Learnings to QUEEN.md

After extracting learnings in Step 2, promote high-confidence validated learnings to QUEEN.md wisdom.

**Promotion Criteria:**
- Only learnings with `status: "validated"` are eligible
- Must have been tested and confirmed working
- Should represent actionable patterns, not one-off observations

**Process:**

1. **Get colony name from state:**
   ```bash
   colony_name=$(jq -r '.session_id | split("_")[1] // "unknown"' .aether/data/COLONY_STATE.json)
   ```

2. **Extract validated learnings from current phase:**
   Read `memory.phase_learnings` and filter for entries where:
   - `phase` matches the completed phase number
   - Any learning in `learnings[]` has `status: "validated"`

3. **Promote to QUEEN.md using queen-promote:**

   For each validated learning, determine the wisdom type and call queen-promote:

   **Type Mapping:**
   - Learning about success patterns → `pattern`
   - Learning about what to avoid/corrections → `redirect`
   - Learning about fundamental principles → `philosophy`
   - Learning about technology/stack → `stack`

   ```bash
   # Example promotions
   bash .aether/aether-utils.sh queen-promote "pattern" "<learning claim>" "$colony_name"
   bash .aether/aether-utils.sh queen-promote "redirect" "<what to avoid>" "$colony_name"
   ```

4. **Log promotion results:**
   ```bash
   bash .aether/aether-utils.sh activity-log "PROMOTED" "Queen" "Promoted N validated learnings to QUEEN.md wisdom"
   ```

**Display promotion summary:**
```
🧠 Wisdom Promotion
==================
{count} validated learning(s) promoted to QUEEN.md:
  - [{type}] {brief claim preview}
```

Skip this step if:
- No validated learnings exist for this phase
- QUEEN.md does not exist (run queen-init first if needed)
- All learnings are still hypotheses or disproven

### Step 2.3: Update Handoff Document

After advancing the phase, update the handoff document with the new current state:

```bash
# Determine if there's a next phase
next_phase_id=$((current_phase + 1))
has_next_phase=$(jq --arg next "$next_phase_id" '.plan.phases | map(select(.id == ($next | tonumber))) | length' .aether/data/COLONY_STATE.json)

# Write updated handoff
cat > .aether/HANDOFF.md << 'HANDOFF_EOF'
# Colony Session — Phase Advanced

## Quick Resume
Run `/ant:build {next_phase_id}` to start working on the current phase.

## State at Advancement
- Goal: "$(jq -r '.goal' .aether/data/COLONY_STATE.json)"
- Completed Phase: {completed_phase_id} — {completed_phase_name}
- Current Phase: {next_phase_id} — {next_phase_name}
- State: READY
- Updated: $(date -u +%Y-%m-%dT%H:%M:%SZ)

## What Was Completed
- Phase {completed_phase_id} marked as completed
- Learnings extracted: {learning_count}
- Instincts updated: {instinct_count}
- Wisdom promoted to QUEEN.md: {promoted_count}

## Current Phase Tasks
$(jq -r '.plan.phases[] | select(.id == next_phase_id) | .tasks[] | "- [ ] \(.id): \(.description)"' .aether/data/COLONY_STATE.json)

## Next Steps
- Build current phase: `/ant:build {next_phase_id}`
- Review phase details: `/ant:phase {next_phase_id}`
- Pause colony: `/ant:pause-colony`

## Session Note
Phase advanced successfully. Colony is READY to build Phase {next_phase_id}.
HANDOFF_EOF
```

This handoff reflects the post-advancement state, allowing seamless resumption even if the session is lost.

### Step 2.4: Update Changelog

**Append a changelog entry for the completed phase.**

If `CHANGELOG.md` exists in the project root:

1. Read the file
2. Find the `## [Unreleased]` section
3. Under the appropriate sub-heading (`### Added`, `### Changed`, or `### Fixed`), append a bullet for the completed phase:

```
- **Phase {id}: {phase_name}** — {one-line summary of what was accomplished}. ({list of key files modified})
```

**Determining the sub-heading:**
- If the phase created new features/commands → `### Added`
- If the phase modified existing behavior → `### Changed`
- If the phase fixed bugs → `### Fixed`
- If unclear, default to `### Changed`

**The one-line summary** should describe the user-visible outcome, not implementation details. Derive it from the phase description and task summaries.

**If no `## [Unreleased]` section exists**, create one at the top of the file (after the header).

**If no `CHANGELOG.md` exists**, skip this step silently.

### Step 2.6: Commit Suggestion (Optional)

**This step is non-blocking. Skipping does not affect phase advancement or any subsequent steps. Failure to commit has zero consequences.**

After the phase is advanced and changelog updated, suggest a commit to preserve the milestone.

#### Step 2.6.1: Capture AI Description

**As the AI, briefly describe what was accomplished in this phase.**

Look at:
1. The phase PLAN.md `<objective>` section (what we set out to do)
2. Tasks that were marked complete
3. Files that were modified (from git diff --stat)
4. Any patterns or decisions recorded

**Provide a brief, memorable description** (10-15 words, imperative mood):
- Good: "Implement task-based model routing with keyword detection and precedence chain"
- Good: "Fix build timing by removing background execution from worker spawns"
- Bad: "Phase complete" (too vague)
- Bad: "Modified files in bin/lib" (too mechanical)

Store this as `ai_description` for the commit message.

#### Step 2.6.2: Generate Enhanced Commit Message

```bash
bash .aether/aether-utils.sh generate-commit-message "contextual" {phase_id} "{phase_name}" "{ai_description}" {plan_number}
```

Parse the returned JSON to extract:
- `message` - the commit subject line
- `body` - structured metadata (Scope, Files)
- `files_changed` - file count
- `subsystem` - derived subsystem name
- `scope` - phase.plan format

**Check files changed:**
```bash
git diff --stat HEAD 2>/dev/null | tail -5
```
If not in a git repo or no changes detected, skip this step silently.

**Display the enhanced suggestion:**
```
──────────────────────────────────────────────────
Commit Suggestion
──────────────────────────────────────────────────

  AI Description: {ai_description}

  Formatted Message:
  {message}

  Metadata:
  Scope: {scope}
  Files: {files_changed} files changed
  Preview: {first 5 lines of git diff --stat}

──────────────────────────────────────────────────
```

**Use AskUserQuestion:**
```
Commit this milestone?

1. Yes, commit with this message
2. Yes, but let me edit the description
3. No, I'll commit later
```

**If option 1 ("Yes, commit with this message"):**
```bash
git add -A && git commit -m "{message}" -m "{body}"
```
Display: `Committed: {message} ({files_changed} files)`

**If option 2 ("Yes, but let me edit"):**
Use AskUserQuestion to get the user's custom description:
```
Enter your description (or press Enter to keep: '{ai_description}'):
```
Then regenerate the commit message with the new description and commit.

**If option 3 ("No, I'll commit later"):**
Display: `Skipped. Your changes are saved on disk but not committed.`

**Record the suggestion to prevent double-prompting:**
Set `last_commit_suggestion_phase` to `{phase_id}` in COLONY_STATE.json (add the field at the top level if it does not exist).

**Error handling:** If any git command fails (not a repo, merge conflict, pre-commit hook rejection), display the error output and continue to the next step. The commit suggestion is advisory only -- it never blocks the flow.

Continue to Step 2.7 (Context Clear Suggestion), then to Step 2.5 (Project Completion) or Step 3 (Display Result).

### Step 2.7: Context Clear Suggestion (Optional)

**This step is non-blocking. Skipping does not affect phase advancement.**

After committing (or skipping commit), suggest clearing context to refresh before the next phase.

1. **Display the suggestion:**
```
──────────────────────────────────────────────────
Context Refresh
──────────────────────────────────────────────────

State is fully persisted and committed.
Phase {next_id} is ready to build.

──────────────────────────────────────────────────
```

2. **Use AskUserQuestion:**
```
Clear context now?

1. Yes, clear context then run /ant:build {next_id}
2. No, continue in current context
```

3. **If option 1 ("Yes, clear context"):**

   **IMPORTANT:** Claude Code does not support programmatic /clear. Display instructions:
   ```
   Please type: /clear
   
   Then run: /ant:build {next_id}
   ```
   
   Record the suggestion: Set `context_clear_suggested` to `true` in COLONY_STATE.json.

4. **If option 2 ("No, continue in current context"):**
   Display: `Continuing in current context. State is saved.`

Continue to Step 2.5 (Project Completion) or Step 3 (Display Result).

### Step 2.8: Update Context Document

After phase advancement is complete, update `.aether/CONTEXT.md`:

**Log the activity:**
```bash
bash .aether/aether-utils.sh context-update activity "continue" "Phase {prev_id} completed, advanced to {next_id}" "—"
```

**Update the phase:**
```bash
bash .aether/aether-utils.sh context-update update-phase {next_id} "{next_phase_name}" "YES" "Phase advanced, ready to build"
```

**Log any decisions from this session:**
If any architectural decisions were made during verification, also run:
```bash
bash .aether/aether-utils.sh context-update decision "{decision_description}" "{rationale}" "Queen"
```

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

👑 Wisdom Added to QUEEN.md:
{count} patterns/redirects/philosophies promoted across all phases

🐜 The colony rests. Well done!
```

3. Write summary to `.aether/data/completion-report.md`
4. Display next commands and stop.

### Step 3: Display Result

**If visual_mode is true, render final swarm display:**
Run using the Bash tool with description "Rendering advancement summary...": `bash .aether/aether-utils.sh swarm-display-update "Queen" "prime" "completed" "Phase advanced" "Colony" '{"read":5,"grep":2,"edit":3,"bash":2}' 100 "fungus_garden" 100 && bash .aether/aether-utils.sh swarm-display-text "$continue_id"`

Output:

```
🐜 ═══════════════════════════════════════════════════
   P H A S E   A D V A N C E M E N T
═══════════════════════════════════════════════════ 🐜

✅ Phase {prev_id}: {prev_name} -- COMPLETED

🧠 Learnings Extracted:
{list learnings added}

👑 Wisdom Promoted to QUEEN.md:
{for each promoted learning:}
   [{type}] {brief claim}
{end for}

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
   /ant:build {next_id}   🔨 Start building Phase {next_id}: {next_name}
   /ant:phase {next_id}   📋 Review phase details first
   /ant:focus "<area>"    🎯 Guide colony attention

💾 State persisted — context clear suggested above

📋 Context document updated at `.aether/CONTEXT.md`
```

**IMPORTANT:** In the "Next Steps" section above, substitute the actual phase number for `{next_id}` (calculated in Step 2 as `current_phase + 1`). For example, if advancing to phase 4, output `/ant:build 4` not `/ant:build {next_id}`.

### Step 4: Update Session

Update the session tracking file to enable `/ant:resume` after context clear:

```bash
bash .aether/aether-utils.sh session-update "/ant:continue" "/ant:build {next_id}" "Phase {prev_id} completed, advanced to Phase {next_id}"
```

Run using the Bash tool with description "Saving session state...": `bash .aether/aether-utils.sh session-update "/ant:continue" "/ant:build {next_id}" "Phase {prev_id} completed, advanced to Phase {next_id}"`
