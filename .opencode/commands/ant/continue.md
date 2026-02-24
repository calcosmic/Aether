---
name: ant:continue
description: "➡️🐜🚪🐜➡️ Detect build completion, reconcile state, and advance to next phase"
---

You are the **Queen Ant Colony**. Reconcile completed work and advance to the next phase.

## Instructions

### Step -1: Normalize Arguments

Run: `normalized_args=$(bash .aether/aether-utils.sh normalize-args "$@")`

This ensures arguments work correctly in both Claude Code and OpenCode. Use `$normalized_args` throughout this command.

Parse `$normalized_args`:
- If contains `--no-visual`: set `visual_mode = false` (visual is ON by default)
- Otherwise: set `visual_mode = true`

### Step 0: Initialize Visual Mode (if enabled)

If `visual_mode` is true:
```bash
# Generate session ID
continue_id="continue-$(date +%s)"

# Initialize swarm display
bash .aether/aether-utils.sh swarm-display-init "$continue_id"
bash .aether/aether-utils.sh swarm-display-update "Queen" "prime" "excavating" "Phase continuation" "Colony" '{"read":0,"grep":0,"edit":0,"bash":0}' 0 "fungus_garden" 0
```

### Step 0.5: Version Check (Non-blocking)

Run using the Bash tool: `bash .aether/aether-utils.sh version-check-cached 2>/dev/null || true`

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

Run:
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
- If `docs == 0`: display `🗺️ Survey: not found (run /ant:colonize for stronger context)` and continue.
- If `age_days > 14`: display `🗺️ Survey: {docs} docs loaded ({age_days}d old, consider /ant:colonize --force-resurvey)` and continue.
- Otherwise: display `🗺️ Survey: {docs} docs loaded ({age_days}d old)` and continue.

Survey context is advisory only and must not block advancement by itself.

### Step 1.5.3: Verification Loop Gate (MANDATORY)

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

#### Step 1.5.1: Probe Coverage Agent (Conditional)

**Test coverage improvement -- runs when coverage < 80% AND tests pass.**

1. **Check coverage threshold condition:**
   - Coverage data is already available from Phase 4 coverage check
   - If tests failed: Skip Probe silently (coverage data unreliable)
   - If coverage_percent >= 80%: Skip Probe silently, continue to Phase 5
   - If coverage_percent < 80% AND tests passed: Proceed to spawn Probe

2. **If skipping Probe:**
```
Probe: Coverage at {coverage_percent}% -- {reason_for_skip}
```
Continue to Phase 5: Secrets Scan.

3. **If spawning Probe:**

   a. Generate Probe name and dispatch:
   Run using the Bash tool with description "Generating Probe name...": `probe_name=$(bash .aether/aether-utils.sh generate-ant-name "probe") && bash .aether/aether-utils.sh spawn-log "Queen" "probe" "$probe_name" "Coverage improvement: ${coverage_percent}%" && echo "{\"name\":\"$probe_name\"}"`

   b. Update swarm display (if visual_mode is true):
   Run using the Bash tool with description "Updating swarm display...": `bash .aether/aether-utils.sh swarm-display-update "$probe_name" "probe" "scanning" "Coverage improvement" "Quality" '{"read":0,"grep":0,"edit":0,"bash":0}' 0 "fungus_garden" 0`

   c. Display: `Probe {probe_name} spawning -- Coverage at {coverage_percent}%, generating tests for uncovered paths...`

   d. Determine uncovered files:
   Run using the Bash tool with description "Getting modified source files...": `modified_source_files=$(git diff --name-only HEAD~1 2>/dev/null || git diff --name-only) && source_files=$(echo "$modified_source_files" | grep -v "\.test\." | grep -v "\.spec\." | grep -v "__tests__") && echo "$source_files"`

   e. Spawn Probe agent:

   > **Platform note**: In Claude Code, use `Task tool with subagent_type="aether-probe"`. In OpenCode, use the equivalent agent spawning mechanism for your platform (e.g., invoke the aether-probe agent definition from `.opencode/agents/aether-probe.md`).

   Probe mission: Improve test coverage for uncovered code paths in the modified files.
   - Analyze the modified source files for uncovered branches and edge cases
   - Identify which paths lack test coverage
   - Generate test cases that exercise uncovered code paths
   - Run the new tests to verify they pass
   - Report coverage improvements and edge cases discovered

   Constraints:
   - Test files ONLY -- never modify source code
   - Follow existing test conventions in the codebase
   - Do NOT delete or modify existing tests

   f. Parse Probe JSON output and log completion:
   Extract: `tests_added`, `coverage.lines`, `coverage.branches`, `coverage.functions`, `edge_cases_discovered`

   Run using the Bash tool with description "Logging Probe completion...": `bash .aether/aether-utils.sh spawn-complete "$probe_name" "completed" "{probe_summary}"`

   g. Log findings to midden:
   Run using the Bash tool with description "Logging Probe findings to midden...": `bash .aether/aether-utils.sh midden-write "coverage" "Probe generated tests, coverage: ${coverage_lines}%/${coverage_branches}%/${coverage_functions}%" "probe"`

4. **NON-BLOCKING continuation:**
   Display Probe findings summary:
   ```
   Probe complete -- Findings logged to midden, continuing verification...
      Tests added: {count}
      Edge cases discovered: {count}
   ```

   **CRITICAL:** ALWAYS continue to Phase 5 (Secrets Scan) regardless of Probe results. Probe is strictly non-blocking.

5. **Record Probe status for verification report:**
   Set `probe_status = "ACTIVE"` and store tests_added count and edge_cases count for the verification report.

**Phase 5: Secrets Scan**:
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

Continue to Step 1.6.

### Step 1.6: Spawn Enforcement Gate (MANDATORY)

**The Iron Law:** No phase advancement without worker spawning for non-trivial phases.

Read `.aether/data/spawn-tree.txt` to count spawns for this phase.

```bash
grep -c "spawned" .aether/data/spawn-tree.txt 2>/dev/null || echo "0"
```

Also check for Watcher spawns specifically:
```bash
grep -c "watcher" .aether/data/spawn-tree.txt 2>/dev/null || echo "0"
```

**HARD REJECTION - If spawn_count == 0 and phase had 3+ tasks:**

```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
⛔🐜 S P A W N   G A T E   F A I L E D
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

This phase had {task_count} tasks but spawn_count: 0
The Prime Worker violated the spawn protocol.

🐜 The colony requires actual parallelism:
  - Prime Worker MUST spawn specialists for non-trivial work
  - A single agent doing everything is NOT a colony
  - "Justifications" for not spawning are not accepted

🔧 Required Actions:
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
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
⛔👁️🐜 W A T C H E R   G A T E   F A I L E D
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

No Watcher ant was spawned for testing/verification.
Testing MUST be performed by a separate agent, not the builder.

🐜 Why this matters:
  - Builders verify their own work = confirmation bias
  - Independent Watchers catch bugs builders miss
  - "Build passing" ≠ "App working"

🔧 Required Actions:
  1. Run /ant:build {phase} again
  2. Prime Worker MUST spawn at least 1 Watcher
  3. Watcher must independently verify the work

The phase will NOT advance until a Watcher validates.
```

**CRITICAL:** Do NOT proceed. Log the violation.

**If spawn_count >= 1 AND watcher_count >= 1:**

```
✅🐜 SPAWN GATE PASSED — {spawn_count} workers | {watcher_count} watchers
```

Continue to Step 1.7.

### Step 1.7: Anti-Pattern Gate

Scan all modified/created files for known anti-patterns. This catches recurring bugs before they reach production.

```bash
bash .aether/aether-utils.sh check-antipattern "{file_path}"
```

Run for each file in `files_created` and `files_modified` from Prime Worker output.

**Anti-Pattern Report:**

```
🔍🐜 Anti-Pattern Scan — {count} files scanned

{if critical issues:}
🛑 CRITICAL (must fix):
{list each with file:line and description}

{if warnings:}
⚠️ WARNINGS:
{list each with file:line and description}

{if clean:}
✅🐜 No anti-patterns detected
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
⛔🐜 ANTI-PATTERN GATE FAILED

Critical anti-patterns detected:
{list issues with file paths}

Run /ant:build {phase} again after fixing.
```

Do NOT proceed to Step 2.

If no CRITICAL issues, continue to Step 1.7.1.

### Step 1.7.1: Proactive Refactoring Gate (Conditional)

**Complexity-based refactoring -- runs when code exceeds maintainability thresholds.**

1. **Get modified/created files from recent work:**
   Run using the Bash tool with description "Getting modified files for complexity check...": `modified_files=$(git diff --name-only HEAD~1 2>/dev/null || git diff --name-only) && echo "$modified_files"`

2. **Check complexity thresholds for each file:**

   For each file, check:
   - Line count > 300 lines
   - Long functions > 50 lines (simplified heuristic)
   - Directory density > 10 new files

   Run using the Bash tool with description "Checking complexity thresholds...":
   ```bash
   modified_files=$(git diff --name-only HEAD~1 2>/dev/null || git diff --name-only)

   complexity_trigger=false
   files_needing_refactor=""

   for file in $modified_files; do
     if [[ -f "$file" ]]; then
       line_count=$(wc -l < "$file" 2>/dev/null || echo "0")
       if [[ "$line_count" -gt 300 ]]; then
         complexity_trigger=true
         files_needing_refactor="$files_needing_refactor $file"
         continue
       fi

       long_funcs=$(grep -c "^[[:space:]]*[a-zA-Z_][a-zA-Z0-9_]*[[:space:]]*(" "$file" 2>/dev/null || echo "0")
       if [[ "$long_funcs" -gt 50 ]]; then
         complexity_trigger=true
         files_needing_refactor="$files_needing_refactor $file"
       fi
     fi
   done

   if [[ -n "$modified_files" ]]; then
     dir_counts=$(echo "$modified_files" | xargs -I {} dirname {} 2>/dev/null | sort | uniq -c | sort -rn)
     high_density_dir=$(echo "$dir_counts" | awk '$1 > 10 {print $2}' | head -1)
     if [[ -n "$high_density_dir" ]]; then
       complexity_trigger=true
     fi
   fi

   echo "{\"complexity_trigger\": \"$complexity_trigger\", \"files_needing_refactor\": \"$files_needing_refactor\"}"
   ```

3. **If complexity thresholds NOT exceeded:**
   ```
   Weaver: Complexity thresholds not exceeded -- skipping proactive refactoring
   ```
   Continue to Step 1.8.

4. **If complexity thresholds exceeded:**

   a. **Establish test baseline before refactoring:**
   Run using the Bash tool with description "Establishing test baseline...": `test_output_before=$(npm test 2>&1 || echo "TEST_FAILED") && tests_passing_before=$(echo "$test_output_before" | grep -oE '[0-9]+ passing' | grep -oE '[0-9]+' || echo "0") && echo "Baseline: $tests_passing_before tests passing"`

   b. **Generate Weaver name and dispatch:**
   Run using the Bash tool with description "Generating Weaver name...": `weaver_name=$(bash .aether/aether-utils.sh generate-ant-name "weaver") && bash .aether/aether-utils.sh spawn-log "Queen" "weaver" "$weaver_name" "Proactive refactoring" && echo "{\"name\":\"$weaver_name\"}"`

   c. **Update swarm display (if visual_mode is true):**
   Run using the Bash tool with description "Updating swarm display...": `bash .aether/aether-utils.sh swarm-display-update "$weaver_name" "weaver" "refactoring" "Proactive refactoring" "Quality" '{"read":0,"grep":0,"edit":0,"bash":0}' 0 "fungus_garden" 0`

   d. **Display:** `Weaver {weaver_name} spawning -- Refactoring complex code...`

   e. **Spawn Weaver agent:**

   > **Platform note**: In Claude Code, use `Task tool with subagent_type="aether-weaver"`. In OpenCode, use the equivalent agent spawning mechanism for your platform (e.g., invoke the aether-weaver agent definition from `.opencode/agents/aether-weaver.md`).

   Weaver mission: Refactor complex code to improve maintainability while preserving behavior.
   - Analyze target files for complexity issues
   - Plan incremental refactoring steps
   - Execute one step at a time
   - Run tests after each step
   - If tests pass, continue; if fail, revert and try smaller step
   - Report all changes made

   Constraints:
   - NEVER change behavior -- only structure
   - Run tests after each refactoring step
   - If tests fail, revert immediately
   - Do NOT modify test expectations to make tests pass
   - Do NOT modify .aether/ system files

   f. **Parse Weaver JSON output and verify tests:**
   Extract: `files_refactored`, `tests_all_passing`, `complexity_before`, `complexity_after`

   Run using the Bash tool with description "Verifying tests after refactoring...":
   ```bash
   test_output_after=$(npm test 2>&1 || echo "TEST_FAILED")
   tests_passing_after=$(echo "$test_output_after" | grep -oE '[0-9]+ passing' | grep -oE '[0-9]+' || echo "0")

   if [[ "$tests_passing_after" -lt "$tests_passing_before" ]]; then
     echo "REVERT_NEEDED: Tests failed after refactoring"
     git checkout -- $files_needing_refactor
     weaver_status="reverted"
   else
     echo "PASSING: Tests passing after refactoring ($tests_passing_after)"
     weaver_status="completed"
   fi
   ```

   g. **Log completion:**
   Run using the Bash tool with description "Logging Weaver completion...": `bash .aether/aether-utils.sh spawn-complete "$weaver_name" "$weaver_status" "Refactoring $weaver_status"`

   h. **Log to midden:**
   Run using the Bash tool with description "Logging refactoring activity to midden...": `bash .aether/aether-utils.sh midden-write "refactoring" "Weaver refactored files, complexity before/after: ${complexity_before}/${complexity_after}" "weaver"`

5. **Display completion:**
   ```
   Weaver: Proactive refactoring {weaver_status}
      Files refactored: {count} | Complexity: {before} -> {after}
   ```

6. **NON-BLOCKING continuation:**
   The Weaver step is NON-BLOCKING -- continue to Step 1.8 regardless of refactoring results.

Continue to Step 1.8.

### Step 1.8: Gatekeeper Security Gate (Conditional)

**Supply chain security audit -- runs only when package.json exists.**

First, check for package.json:
Run using the Bash tool with description "Checking for package.json...": `test -f package.json && echo "exists" || echo "missing"`

**If package.json is missing:**
```
Gatekeeper: No package.json found -- skipping supply chain audit
```
Continue to Step 1.9.

**If package.json exists:**

1. Generate Gatekeeper name and log spawn:
Run using the Bash tool with description "Generating Gatekeeper name...": `gatekeeper_name=$(bash .aether/aether-utils.sh generate-ant-name "gatekeeper") && bash .aether/aether-utils.sh spawn-log "Queen" "gatekeeper" "$gatekeeper_name" "Supply chain security audit" && echo "{\"name\":\"$gatekeeper_name\"}"`

2. Update swarm display (if visual_mode is true):
Run using the Bash tool with description "Updating swarm display...": `bash .aether/aether-utils.sh swarm-display-update "$gatekeeper_name" "gatekeeper" "scanning" "CVE and license audit" "Security" '{"read":0,"grep":0,"edit":0,"bash":0}' 0 "fungus_garden" 0`

3. Display: `Gatekeeper {name} spawning -- Scanning dependencies for CVEs and license compliance...`

4. Spawn Gatekeeper agent:

> **Platform note**: In Claude Code, use `Task tool with subagent_type="aether-gatekeeper"`. In OpenCode, use the equivalent agent spawning mechanism for your platform (e.g., invoke the aether-gatekeeper agent definition from `.opencode/agents/aether-gatekeeper.md`).

Gatekeeper mission: Perform supply chain security audit on this codebase.
- Inventory all dependencies from package.json
- Scan for known CVEs using npm audit or equivalent
- Check license compliance for all packages
- Assess dependency health (outdated, deprecated, maintenance status)
- Report findings with severity levels

5. Parse Gatekeeper JSON output and log completion:
Extract: `security.critical`, `security.high`, `status`

Run using the Bash tool with description "Logging Gatekeeper completion...": `bash .aether/aether-utils.sh spawn-complete "$gatekeeper_name" "completed" "{gatekeeper_summary}"`

**Gate Decision Logic:**

- **If `security.critical > 0`:**
```
GATEKEEPER GATE FAILED

Critical security vulnerabilities detected: {critical_count}

CRITICAL CVEs must be fixed before phase advancement.

Required Actions:
  1. Run `npm audit` to see full details
  2. Fix or update vulnerable dependencies
  3. Run /ant:continue again after resolving

The phase will NOT advance with critical CVEs.
```
**CRITICAL:** Do NOT proceed to Step 1.9. Stop here.

- **If `security.high > 0`:**
```
Gatekeeper: {high_count} high-severity issues found

Security warnings logged to midden for later review.
Proceeding with caution...
```
Run using the Bash tool with description "Logging high-severity warnings...": `bash .aether/aether-utils.sh midden-write "security" "High CVEs found: $high_count" "gatekeeper"`
Continue to Step 1.9.

- **If clean (no critical or high):**
```
Gatekeeper: No critical security issues found
```
Continue to Step 1.9.

### Step 1.9: Auditor Quality Gate (MANDATORY)

**Code quality audit -- runs on every `/ant:continue` for consistent coverage.**

1. Generate Auditor name and log spawn:
Run using the Bash tool with description "Generating Auditor name...": `auditor_name=$(bash .aether/aether-utils.sh generate-ant-name "auditor") && bash .aether/aether-utils.sh spawn-log "Queen" "auditor" "$auditor_name" "Code quality audit" && echo "{\"name\":\"$auditor_name\"}"`

2. Update swarm display (if visual_mode is true):
Run using the Bash tool with description "Updating swarm display...": `bash .aether/aether-utils.sh swarm-display-update "$auditor_name" "auditor" "reviewing" "Multi-lens code analysis" "Quality" '{"read":0,"grep":0,"edit":0,"bash":0}' 0 "fungus_garden" 0`

3. Display: `Auditor {name} spawning -- Reviewing code with multi-lens analysis...`

4. Get modified files for audit context:
Run using the Bash tool with description "Getting modified files...": `modified_files=$(git diff --name-only HEAD~1 2>/dev/null || git diff --name-only) && echo "$modified_files"`

5. Spawn Auditor agent:

> **Platform note**: In Claude Code, use `Task tool with subagent_type="aether-auditor"`. In OpenCode, use the equivalent agent spawning mechanism for your platform (e.g., invoke the aether-auditor agent definition from `.opencode/agents/aether-auditor.md`).

Auditor mission: Perform comprehensive code quality audit on this codebase.
- Review all modified files from the recent commit(s)
- Apply all 4 audit lenses: security, performance, quality, maintainability
- Score each finding by severity (CRITICAL/HIGH/MEDIUM/LOW/INFO)
- Calculate overall quality score (0-100)
- Document specific issues with file:line references and fix suggestions

6. Parse Auditor JSON output and log completion:
Extract: `findings.critical`, `findings.high`, `findings.medium`, `findings.low`, `findings.info`, `overall_score`, `dimensions_audited`

Run using the Bash tool with description "Logging Auditor completion...": `bash .aether/aether-utils.sh spawn-complete "$auditor_name" "completed" "{auditor_summary}"`

**Gate Decision Logic:**

- **If `findings.critical > 0`:**
```
AUDITOR GATE FAILED

Critical code quality issues detected: {critical_count}

CRITICAL findings must be fixed before phase advancement.

Required Actions:
  1. Review the critical issues listed below
  2. Fix each critical finding
  3. Run /ant:continue again after resolving

Critical Findings:
{list each critical finding with file:line and description}

The phase will NOT advance with critical quality issues.
```
Run using the Bash tool with description "Logging critical quality block...": `bash .aether/aether-utils.sh error-flag-pattern "auditor-critical-findings" "$critical_count critical quality issues found" "critical"`
**CRITICAL:** Do NOT proceed to Step 1.10. Stop here.

- **Else if `overall_score < 60`:**
```
AUDITOR GATE FAILED

Code quality score below threshold: {overall_score}/100 (threshold: 60)

Quality score must reach 60+ before phase advancement.

Required Actions:
  1. Address the top issues preventing score improvement
  2. Focus on HIGH severity items first
  3. Run /ant:continue again after improving quality

The phase will NOT advance with quality score below 60.
```
Run using the Bash tool with description "Logging quality score block...": `bash .aether/aether-utils.sh error-flag-pattern "auditor-quality-score" "Score $overall_score below threshold 60" "critical"`
**CRITICAL:** Do NOT proceed to Step 1.10. Stop here.

- **Else if `findings.high > 0`:**
```
Auditor: Quality score {overall_score}/100 -- PASSED with warnings

{high_count} high-severity quality issues found:
{list high findings}

Quality warnings logged to midden for later review.
Proceeding with caution...
```
Run using the Bash tool with description "Logging high-quality warnings...": `bash .aether/aether-utils.sh midden-write "quality" "High severity issues: $high_count (score: $overall_score)" "auditor"`
Continue to Step 1.10.

- **If clean (score >= 60, no critical):**
```
Auditor: Quality score {overall_score}/100 -- PASSED
```
Continue to Step 1.10.

### Step 1.10: TDD Evidence Gate (MANDATORY)

**The Iron Law:** No TDD claims without actual test files.

If Prime Worker reported TDD metrics (tests_added, tests_total, coverage_percent), verify test files exist:

```bash
# Check for test files based on project type
find . -name "*.test.*" -o -name "*_test.*" -o -name "*Tests.swift" -o -name "test_*.py" 2>/dev/null | head -10
```

**If Prime Worker claimed tests_added > 0 but no test files found:**

```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
⛔🧪🐜 T D D   G A T E   F A I L E D
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Prime Worker claimed:
  tests_added: {claimed_count}
  tests_total: {claimed_total}
  coverage: {claimed_coverage}%

But no test files were found in the codebase.

🚨 CRITICAL violation — fabricated TDD metrics.

🔧 Required Actions:
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

Continue to Step 1.11.

### Step 1.11: Runtime Verification Gate (MANDATORY)

**The Iron Law:** Build passing ≠ App working.

Before advancing, the user must confirm the application actually runs.

Use AskUserQuestion:

```
──────────────────────────────────────────────────
🐜 Runtime Verification Required
──────────────────────────────────────────────────

Build checks passed — but does the app actually work?

Have you tested the application at runtime?
```

Options:
1. **Yes, tested and working** - App runs correctly, features work
2. **Yes, tested but has issues** - App runs but has bugs (describe)
3. **No, haven't tested yet** - Need to test before continuing
4. **Skip (not applicable)** - No runnable app in this phase (e.g., library code)

**If "Yes, tested and working":**
```
✅🐜 RUNTIME VERIFIED — User confirmed app works.
```
Continue to Step 1.12.

**If "Yes, tested but has issues":**
```
⛔🐜 RUNTIME GATE FAILED — User reported issues.

Please describe the issues so they can be addressed:
```

Use AskUserQuestion to get issue details. Log to errors.records:
```bash
bash .aether/aether-utils.sh error-add "runtime" "critical" "{user_description}" {phase}
```

Do NOT proceed to Step 2.

**If "No, haven't tested yet":**
```
⏸️🐜 RUNTIME PENDING — Test the app, then run /ant:continue again.

  - [ ] App launches without crashing
  - [ ] Core features work as expected
  - [ ] UI responds to user interaction
  - [ ] No freezes or hangs
```

Do NOT proceed to Step 2.

**If "Skip (not applicable)":**

Only valid for phases that don't produce runnable code (e.g., documentation, config files, library code with no entry point).

```
⏭️ RUNTIME CHECK SKIPPED

User indicated no runnable app for this phase.
Proceeding to phase advancement.
```

Continue to Step 1.12.

### Step 1.12: Flags Gate (MANDATORY)

**The Iron Law:** No phase advancement with unresolved blockers.

First, auto-resolve any flags eligible for resolution now that verification has passed:
```bash
bash .aether/aether-utils.sh flag-auto-resolve "build_pass"
```

Then check for remaining blocking flags:
```bash
bash .aether/aether-utils.sh flag-check-blockers {current_phase}
```

Parse result for `blockers`, `issues`, and `notes` counts.

**If blockers > 0:**

```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
⛔🚩🐜 F L A G S   G A T E   F A I L E D
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

{blockers} blocker(s) must be resolved first.

🚩 Active Blockers:
{list each blocker flag with ID, title, and description}

🔧 Required Actions:
  1. Fix the issues described in each blocker
  2. Resolve flags: /ant:flags --resolve {flag_id} "resolution message"
  3. Run /ant:continue again after resolving all blockers
```

**CRITICAL:** Do NOT proceed to Step 2. Do NOT advance the phase.

**If blockers == 0 but issues > 0:**

```
⚠️🐜 FLAGS: {issues} issue(s) noted (non-blocking)

{list each issue flag}

Use /ant:flags to review.
```

Continue to Step 2.

**If all clear (no blockers or issues):**

```
✅🐜 FLAGS GATE PASSED — No blockers.
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

2.5. **Record learning observations for threshold tracking:**

   For each learning extracted, record an observation to enable threshold-based wisdom promotion.

   Run using the Bash tool with description "Recording learning observations...":
   ```bash
   colony_name=$(jq -r '.session_id | split("_")[1] // "unknown"' .aether/data/COLONY_STATE.json 2>/dev/null || echo "unknown")

   # Get learnings from the current phase
   current_phase_learnings=$(jq -r --argjson phase "$current_phase" '.memory.phase_learnings[] | select(.phase == $phase)' .aether/data/COLONY_STATE.json 2>/dev/null || echo "")

   if [[ -n "$current_phase_learnings" ]]; then
     echo "$current_phase_learnings" | jq -r '.learnings[]?.claim // empty' 2>/dev/null | while read -r claim; do
       if [[ -n "$claim" ]]; then
         # Default wisdom_type to "pattern" (threshold: 3 observations)
         bash .aether/aether-utils.sh memory-capture "learning" "$claim" "pattern" "worker:continue" 2>/dev/null || true
       fi
     done
     echo "Recorded observations for threshold tracking"
   else
     echo "No learnings to record"
   fi
   ```

   This records each learning in `learning-observations.json` with:
   - Content hash for deduplication (same claim across phases increments count)
   - Observation count (increments if seen before)
   - Colony name for cross-colony tracking

   When observations reach threshold (default: 3 for "pattern" type), they become eligible for promotion in Step 2.1.5.

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

**This entire step produces NO user-visible output.** All pheromone operations run silently -- learnings are deposited in the background. If any pheromone call fails, log the error and continue. Phase advancement must never fail due to pheromone errors.

#### 2.1a: Auto-emit FEEDBACK pheromone for phase outcome

After learning extraction completes in Step 2, auto-emit a FEEDBACK signal summarizing the phase:

```bash
# phase_id and phase_name come from Step 2 state update
# Take the top 1-3 learnings by evidence strength from memory.phase_learnings
# Compress into a single summary sentence

phase_feedback="Phase $phase_id ($phase_name) completed. Key patterns: {brief summary of 1-3 learnings from Step 2}"
# Fallback if no learnings: "Phase $phase_id ($phase_name) completed without notable patterns."

bash .aether/aether-utils.sh pheromone-write FEEDBACK "$phase_feedback" \
  --strength 0.6 \
  --source "worker:continue" \
  --reason "Auto-emitted on phase advance: captures what worked and what was learned" \
  --ttl "30d" 2>/dev/null || true
```

The strength is 0.6 (auto-emitted = lower than user-emitted 0.7). Source is "worker:continue" to distinguish from user-emitted feedback. TTL is 30d so it survives phase transitions and can guide subsequent work.

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

Also capture each recurring pattern as a resolution candidate:

```bash
bash .aether/aether-utils.sh memory-capture \
  "resolution" \
  "$pattern_text" \
  "pattern" \
  "worker:continue" 2>/dev/null || true
```

If `errors.flagged_patterns` doesn't exist or is empty, skip silently.

#### 2.1c: Expire phase_end signals and archive to midden

After auto-emission, expire all signals with `expires_at == "phase_end"`:

Run using the Bash tool with description "Maintaining pheromone memory...": `bash .aether/aether-utils.sh pheromone-expire --phase-end-only 2>/dev/null && bash .aether/aether-utils.sh eternal-init 2>/dev/null`

### Step 2.1.5: Check for Promotion Proposals

After extracting learnings, check for observations that have met promotion thresholds and present the tick-to-approve UX.

**Normal proposal flow (MEM-01: Silent skip if empty):**

1. **Check for proposals:**
   ```bash
   proposals=$(bash .aether/aether-utils.sh learning-check-promotion 2>/dev/null || echo '{"proposals":[]}')
   proposal_count=$(echo "$proposals" | jq '.proposals | length')
   ```

2. **If proposals exist, invoke the approval workflow:**

   Only show the approval UI when there are actual proposals to review:

   ```bash
   if [[ "$proposal_count" -gt 0 ]]; then
     bash .aether/aether-utils.sh learning-approve-proposals
   fi
   # If no proposals, silently skip without notice
   ```

   The learning-approve-proposals function handles:
   - Displaying proposals with checkbox UI
   - Capturing user selection
   - Executing batch promotions via queen-promote
   - Deferring unselected proposals
   - Offering undo after successful promotions

**Skip conditions:**
- learning-check-promotion returns empty or fails
- No proposals to review (silent skip - no output)
- QUEEN.md does not exist

### Step 2.2: Update Handoff Document

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

### Step 2.3: Update Changelog

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

### Step 2.4: Commit Suggestion (Optional)

**This step is non-blocking. Skipping does not affect phase advancement or any subsequent steps. Failure to commit has zero consequences.**

After the phase is advanced and changelog updated, suggest a commit to preserve the milestone.

1. **Generate the commit message:**
```bash
bash .aether/aether-utils.sh generate-commit-message "milestone" {phase_id} "{phase_name}" "{one_line_summary}"
```
Parse the returned JSON to extract `message` and `files_changed`.

2. **Check files changed:**
```bash
git diff --stat HEAD 2>/dev/null | tail -5
```
If not in a git repo or no changes detected, skip this step silently.

3. **Display the suggestion:**
```
──────────────────────────────────────────────────
Commit Suggestion
──────────────────────────────────────────────────

  Message:  {generated_message}
  Files:    {files_changed} files changed
  Preview:  {first 5 lines of git diff --stat}

──────────────────────────────────────────────────
```

4. **Use AskUserQuestion:**
```
Commit this milestone?

1. Yes, commit with this message
2. Yes, but let me write the message
3. No, I'll commit later
```

5. **If option 1 ("Yes, commit with this message"):**
```bash
git add -A && git commit -m "{generated_message}"
```
Display: `Committed: {generated_message} ({files_changed} files)`

6. **If option 2 ("Yes, but let me write the message"):**
Use AskUserQuestion to get the user's custom commit message, then:
```bash
git add -A && git commit -m "{custom_message}"
```
Display: `Committed: {custom_message} ({files_changed} files)`

7. **If option 3 ("No, I'll commit later"):**
Display: `Skipped. Your changes are saved on disk but not committed.`

8. **Record the suggestion to prevent double-prompting:**
Set `last_commit_suggestion_phase` to `{phase_id}` in COLONY_STATE.json (add the field at the top level if it does not exist).

**Error handling:** If any git command fails (not a repo, merge conflict, pre-commit hook rejection), display the error output and continue to the next step. The commit suggestion is advisory only -- it never blocks the flow.

Continue to Step 2.5 (Context Clear Suggestion), then to Step 2.7 (Project Completion) or Step 3 (Display Result).

### Step 2.5: Context Clear Suggestion (Optional)

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

   **IMPORTANT:** Most AI platforms do not support programmatic context clearing. Display instructions:
   ```
   Please clear your context/conversation, then run: /ant:build {next_id}
   ```
   
   Record the suggestion: Set `context_clear_suggested` to `true` in COLONY_STATE.json.

4. **If option 2 ("No, continue in current context"):**
   Display: `Continuing in current context. State is saved.`

Continue to Step 2.6 (Context Update), then Step 2.7 (Project Completion) or Step 3 (Display Result).

### Step 2.6: Update Context Document

After phase advancement is complete, update `.aether/CONTEXT.md`:

**Log the activity:**
```bash
bash .aether/aether-utils.sh context-update activity "continue" "Phase {prev_id} completed, advanced to {next_id}" "---"
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

### Step 2.7: Project Completion

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

**If visual_mode is true, render final swarm display:**
```bash
bash .aether/aether-utils.sh swarm-display-update "Queen" "prime" "completed" "Phase advanced" "Colony" '{"read":5,"grep":2,"edit":3,"bash":2}' 100 "fungus_garden" 100
bash .aether/aether-utils.sh swarm-display-text "$continue_id"
```

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
   /ant:build {next_id}   🔨 Start building Phase {next_id}: {next_name}
   /ant:phase {next_id}   📋 Review phase details first
   /ant:focus "<area>"    🎯 Guide colony attention

💾 State persisted — context clear suggested above
```

**IMPORTANT:** In the "Next Steps" section above, substitute the actual phase number for `{next_id}` (calculated in Step 2 as `current_phase + 1`). For example, if advancing to phase 4, output `/ant:build 4` not `/ant:build {next_id}`.

### Step 4: Update Session

Update the session tracking file to enable `/ant:resume` after context clear:

Run using the Bash tool with description "Saving session state...": `bash .aether/aether-utils.sh session-update "/ant:continue" "/ant:build {next_id}" "Phase {prev_id} completed, advanced to Phase {next_id}"`
