---
name: ant:continue
description: "➡️🐜🚪🐜➡️ Detect build completion, reconcile state, and advance to next phase"
---

You are the **Queen Ant Colony**. Reconcile completed work and advance to the next phase.

## Instructions

Parse `$ARGUMENTS`:
- If contains `--no-visual`: set `visual_mode = false` (visual is ON by default)
- Otherwise: set `visual_mode = true`

### Step 1: Read State

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
- If `milestone` == `"Crowned Anthill"` -> output "This colony has been sealed. Start a new colony with `/ant:init \"new goal\"`." and stop.
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

### Step 1.5.2: Load Survey Context (Non-blocking)

Run using the Bash tool with description "Checking survey context...":
```bash
survey_check=$(bash .aether/aether-utils.sh survey-verify 2>/dev/null || true)
survey_docs=$(ls -1 .aether/data/survey/*.md 2>/dev/null | wc -l | tr -d ' ')
survey_latest=$(ls -t .aether/data/survey/*.md 2>/dev/null | head -1)
if [[ -n "$survey_latest" ]]; then
  now_epoch=$(date +%s)
  modified_epoch=$(stat -f %m "$survey_latest" 2>/dev/null || stat -c %Y "$survey_latest" 2>/dev/null || echo 0)
  survey_age_days=$(( (now_epoch - modified_epoch) / 86400 ))
else
  survey_age_days=-1
fi
echo "{\"docs\":$survey_docs,\"age_days\":$survey_age_days,\"verify\":$survey_check}"
```

Interpretation:
- If survey docs are missing (`docs == 0`), continue without blocking and display:
  `🗺️ Survey: not found (run /ant:colonize for stronger context)`
- If survey exists but is stale (`age_days > 14`), continue without blocking and display:
  `🗺️ Survey: {docs} docs loaded ({age_days}d old, consider /ant:colonize --force-resurvey)`
- Otherwise display:
  `🗺️ Survey: {docs} docs loaded ({age_days}d old)`

Use this survey status as advisory context for the verification report only.

### Step 1.5: Verification Loop Gate (MANDATORY)

**The Iron Law:** No phase advancement without fresh verification evidence.

Before ANY phase can advance, execute the 6-phase verification loop. See `.aether/docs/disciplines/verification-loop.md` for full reference.

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
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
👁️🐜 V E R I F I C A T I O N   L O O P
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Phase {id} — Checking colony work...
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

**IMPORTANT:** Store the test command exit code in a variable (e.g., `test_exit_code`) for use in Step 1.5.3 verify-claims.

**Coverage Check** (if coverage command exists):
Run using the Bash tool with description "Checking test coverage...": `{coverage_command}  # e.g., npm run test:coverage`
Record: coverage percentage (target: 80%+ for new code)

#### Step 1.5.1: Probe Coverage Agent (Conditional)

**Test coverage improvement — runs when coverage < 80% AND tests pass.**

1. **Check coverage threshold condition:**
   - Coverage data is already available from Phase 4 coverage check
   - If tests failed: Skip Probe silently (coverage data unreliable)
   - If coverage_percent >= 80%: Skip Probe silently, continue to Phase 5
   - If coverage_percent < 80% AND tests passed: Proceed to spawn Probe

2. **If skipping Probe:**
```
🧪🐜 Probe: Coverage at {coverage_percent}% — {reason_for_skip}
```
Continue to Phase 5: Secrets Scan.

3. **If spawning Probe:**

   a. Generate Probe name and dispatch:
   Run using the Bash tool with description "Generating Probe name...": `probe_name=$(bash .aether/aether-utils.sh generate-ant-name "probe" | jq -r '.result') && bash .aether/aether-utils.sh spawn-log "Queen" "probe" "$probe_name" "Coverage improvement: ${coverage_percent}%" && echo "{\"name\":\"$probe_name\"}"`

   b. Display:
   ```
   ━━━ 🧪🐜 P R O B E ━━━
   ──── 🧪🐜 Spawning {probe_name} — Coverage improvement ────
   ```

   d. Determine uncovered files:
   Run using the Bash tool with description "Getting modified source files...": `modified_source_files=$(git diff --name-only HEAD~1 2>/dev/null || git diff --name-only) && source_files=$(echo "$modified_source_files" | grep -v "\.test\." | grep -v "\.spec\." | grep -v "__tests__") && echo "$source_files"`

   e. Spawn Probe agent:

   Use the Task tool with subagent_type="aether-probe" (if available; otherwise use general-purpose and inject the Probe role from `.opencode/agents/aether-probe.md`):

   ```xml
   <mission>
   Improve test coverage for uncovered code paths in the modified files.
   </mission>

   <work>
   1. Analyze the modified source files for uncovered branches and edge cases
   2. Identify which paths lack test coverage
   3. Generate test cases that exercise uncovered code paths
   4. Run the new tests to verify they pass
   5. Report coverage improvements and edge cases discovered
   </work>

   <context>
   Current coverage: {coverage_percent}%
   Target coverage: 80%
   Modified source files: {modified_source_files}
   </context>

   <constraints>
   - Test files ONLY — never modify source code
   - Follow existing test conventions in the codebase
   - Do NOT delete or modify existing tests
   </constraints>

   <output>
   Provide JSON output matching this schema:
   {
     "ant_name": "your probe name",
     "caste": "probe",
     "status": "completed" | "failed" | "blocked",
     "summary": "Brief summary of coverage improvements",
     "coverage": {
       "lines": 0,
       "branches": 0,
       "functions": 0
     },
     "tests_added": ["file1.test.js", "file2.test.js"],
     "edge_cases_discovered": ["edge case 1", "edge case 2"],
     "mutation_score": 0,
     "weak_spots": [],
     "blockers": []
   }
   </output>
   ```

   f. Parse Probe JSON output and log completion:
   Extract: `tests_added`, `coverage.lines`, `coverage.branches`, `coverage.functions`, `edge_cases_discovered`, `mutation_score`

   Run using the Bash tool with description "Logging Probe completion...": `bash .aether/aether-utils.sh spawn-complete "$probe_name" "completed" "{\"tests_added\":${#tests_added[@]},\"coverage\":{\"lines\":${coverage_lines},\"branches\":${coverage_branches},\"functions\":${coverage_functions}}}"`

   g. Log findings to midden:
   Run using the Bash tool with description "Logging Probe findings to midden...": `bash .aether/aether-utils.sh midden-write "coverage" "Probe generated tests, coverage: ${coverage_lines}%/${coverage_branches}%/${coverage_functions}%" "probe"`

   If edge cases found:
   Run using the Bash tool with description "Logging edge cases to midden...": `bash .aether/aether-utils.sh midden-write "edge_cases" "Found ${#edge_cases_discovered[@]} edge cases" "probe"`

4. **NON-BLOCKING continuation:**
   Display Probe findings summary:
   ```
   🧪🐜 Probe complete — Findings logged to midden, continuing verification...
      Tests added: {count}
      Edge cases discovered: {count}
   ```

   **CRITICAL:** ALWAYS continue to Phase 5 (Secrets Scan) regardless of Probe results. Probe is strictly non-blocking — phase advancement continues even if Probe cannot improve coverage.

5. **Record Probe status for verification report:**
   Set `probe_status = "ACTIVE"` and store tests_added count and edge_cases count for the verification report.

**Phase 5: Secrets Scan** (basic grep-based secret detection):
Run using the Bash tool with description "Scanning for exposed secrets...": `grep -rn "sk-\|api_key\|password\s*=" --include="*.ts" --include="*.js" --include="*.py" src/ 2>/dev/null | head -10`
Run using the Bash tool with description "Scanning for debug artifacts...": `grep -rn "console\.log\|debugger" --include="*.ts" --include="*.tsx" --include="*.js" src/ 2>/dev/null | head -10`
Record: potential secrets (critical), debug artifacts (warning).

Note: Professional security scanning happens in Step 1.8 (Gatekeeper for CVEs) and Step 1.9 (Auditor for code quality).

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
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
👁️🐜 V E R I F I C A T I O N   R E P O R T
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

🔨 Build        [PASS/FAIL/SKIP]
🔍 Types        [PASS/FAIL/SKIP] (X errors)
🧹 Lint         [PASS/FAIL/SKIP] (X warnings)
🧪 Tests        [PASS/FAIL/SKIP] (X/Y passed)
   Coverage     {percent}% (target: 80%)
   🧪 Probe     [ACTIVE/SKIP] (tests added: X, edge cases: Y)
🔒 Secrets      [PASS/FAIL] (X issues)
📦 Gatekeeper   [PASS/WARN/SKIP] (X critical, X high)
👥 Auditor      [PASS/FAIL] (score: X/100)
📋 Diff         [X files changed]

──────────────────────────────────────────────────
🐜 Success Criteria
──────────────────────────────────────────────────
  ✅ {criterion 1}: {specific evidence}
  ✅ {criterion 2}: {specific evidence}
  ❌ {criterion 3}: {what's missing}

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Overall: READY / NOT READY
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

#### 3. Gate Decision

**If NOT READY (any of: build fails, tests fail, critical security issues, success criteria unmet):**

```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
⛔🐜 V E R I F I C A T I O N   F A I L E D
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Phase {id} cannot advance until issues are resolved.

🚨 Issues Found:
{list each failure with specific evidence}

🔧 Required Actions:
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
✅🐜 VERIFICATION PASSED

All checks completed with evidence:
{list each check and its evidence}

Proceeding to gate checks...
```

Continue to Step 1.5.3.

### Step 1.5.3: Verify Worker Claims (MANDATORY)

Cross-reference worker claims against reality. This step catches fabricated success claims.

**Always runs. No skip flag.**

1. Check if `.aether/data/last-build-claims.json` exists.
   - If not found: Display "No builder claims file found -- skipping file verification" and continue to Step 1.6. (This handles first-time runs and manual builds.)

2. Capture the test exit code from Phase 4 of the verification loop (stored in `test_exit_code` variable during Phase 4 execution above).

3. Run verification:
   Run using the Bash tool with description "Verifying worker claims...":
   ```bash
   bash .aether/aether-utils.sh verify-claims ".aether/data/last-build-claims.json" "<watcher_json_or_path>" "<test_exit_code>"
   ```

   For the watcher JSON: use the Watcher output from the most recent build (if available in COLONY_STATE.json events or build synthesis). If no Watcher output is available, pass `'{"verification_passed":true}'` as default (conservative -- only test exit code mismatch can trigger).

4. Parse the result:

   **If verification_status is "passed":**
   ```
   Verification passed
   ```
   Continue to Step 1.6.

   **If verification_status is "blocked" AND this is the first attempt (retry_count == 0):**
   Display each mismatch as a plain one-liner:
   ```
   Verification issue: Worker claimed src/api/auth.ts was created, but file does not exist.
   Retrying build for current phase...
   ```

   Log a flag for each mismatch:
   Run using the Bash tool with description "Flagging verification mismatch...":
   ```bash
   bash .aether/aether-utils.sh flag-create "Verification mismatch: <summary>" --type blocker
   ```

   **Auto-retry once** (locked decision):
   - Re-run `/ant:build <current_phase>` for the current phase
   - After retry build completes, re-run the verification loop from Phase 1
   - If verify-claims passes on retry: Display "Retrying... passed on retry" and continue to Step 1.6
   - If verify-claims still blocked on retry: proceed to hard stop below

   **If verification_status is "blocked" AND retry_count >= 1 (retry already attempted):**
   ```
   Verification failed after retry.
   <plain summary of each mismatch>
   Phase will NOT advance. Fix the issues and run /ant:continue again.
   ```

   **CRITICAL:** Do NOT proceed to Step 1.6. Do NOT advance the phase. The verification failure is a hard block just like a test failure.

Continue to Step 1.6.
