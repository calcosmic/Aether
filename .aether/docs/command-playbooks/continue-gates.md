### Step 1.6: Spawn Enforcement Gate (MANDATORY)

**The Iron Law:** No phase advancement without worker spawning for non-trivial phases.

Read `.aether/data/spawn-tree.txt` to count spawns for this phase.

Run using the Bash tool with description "Verifying spawn requirements...": `spawn_count=$(grep -c "spawned" .aether/data/spawn-tree.txt 2>/dev/null || echo "0") && watcher_count=$(grep -c "watcher" .aether/data/spawn-tree.txt 2>/dev/null || echo "0") && echo "{\"spawn_count\": $spawn_count, \"watcher_count\": $watcher_count}"`

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
  1. Run /ant-build {phase} again
  2. Prime Worker MUST spawn at least 1 specialist
  3. Re-run /ant-continue after spawns complete

The phase will NOT advance until spawning occurs.
```

**CRITICAL:** Do NOT proceed to Step 1.7. Do NOT advance the phase.
Log the violation:
```bash
aether activity-log --command "BLOCKED" --details "colony: Spawn gate failed: {task_count} tasks, 0 spawns"
aether error-flag-pattern --name "no-spawn-violation" --description "Prime Worker completed phase without spawning specialists" --severity "critical"
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
  1. Run /ant-build {phase} again
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

For each file, run using the Bash tool with description "Scanning for anti-patterns...": `aether check-antipattern "{file_path}"`

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

Run /ant-build {phase} again after fixing.
```

Do NOT proceed to Step 2.

If no CRITICAL issues, continue to Step 1.7.1.

### Step 1.7.1: Proactive Refactoring Gate (Conditional)

**Complexity-based refactoring — runs when code exceeds maintainability thresholds.**

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
       # Check line count
       line_count=$(wc -l < "$file" 2>/dev/null || echo "0")
       if [[ "$line_count" -gt 300 ]]; then
         complexity_trigger=true
         files_needing_refactor="$files_needing_refactor $file"
         continue
       fi

       # Check for long functions (simplified heuristic)
       long_funcs=$(grep -c "^[[:space:]]*[a-zA-Z_][a-zA-Z0-9_]*[[:space:]]*(" "$file" 2>/dev/null || echo "0")
       if [[ "$long_funcs" -gt 50 ]]; then
         complexity_trigger=true
         files_needing_refactor="$files_needing_refactor $file"
       fi
     fi
   done

   # Check directory density
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
   🔄🐜 Weaver: Complexity thresholds not exceeded — skipping proactive refactoring
   ```
   Continue to Step 1.8.

4. **If complexity thresholds exceeded:**

   a. **Establish test baseline before refactoring:**
   Run using the Bash tool with description "Establishing test baseline...": `test_output_before=$(npm test 2>&1 || echo "TEST_FAILED") && tests_passing_before=$(echo "$test_output_before" | grep -oE '[0-9]+ passing' | grep -oE '[0-9]+' || echo "0") && echo "Baseline: $tests_passing_before tests passing"`

   b. **Generate Weaver name and dispatch:**
   Run using the Bash tool with description "Generating Weaver name...": `weaver_name=$(aether generate-ant-name "weaver" | jq -r '.result') && aether spawn-log --parent "Queen" --caste "weaver" --name "$weaver_name" --task "Proactive refactoring" --depth 0 && echo "{\"name\":\"$weaver_name\"}"`

   c. **Display:**
   ```
   ━━━ 🔄🐜 W E A V E R ━━━
   ──── 🔄🐜 Spawning {weaver_name} — Proactive refactoring ────
   ```

   e. **Spawn Weaver agent:**

   Use the Task tool with subagent_type="aether-weaver" (if available; otherwise use general-purpose and inject the Weaver role from `.opencode/agents/aether-weaver.md`):

   ```xml
   <mission>
   Refactor complex code to improve maintainability while preserving behavior.
   </mission>

   <work>
   1. Analyze target files for complexity issues
   2. Plan incremental refactoring steps
   3. Execute one step at a time
   4. Run tests after each step
   5. If tests pass, continue; if fail, revert and try smaller step
   6. Report all changes made
   </work>

   <context>
   Target Files: {files_needing_refactor}
   Test Baseline: {tests_passing_before} tests passing (MUST maintain after refactor)

   Refactoring Guidelines:
   - Extract methods/functions over 50 lines
   - Split files over 300 lines
   - Remove duplication (DRY)
   - Improve naming for clarity
   - Apply Single Responsibility Principle
   </context>

   <constraints>
   - NEVER change behavior — only structure
   - Run tests after each refactoring step
   - If tests fail, revert immediately
   - Do NOT modify test expectations to make tests pass
   - Do NOT modify .aether/ system files
   </constraints>

   <output>
   Provide JSON output matching this schema:
   {
     "ant_name": "your weaver name",
     "caste": "weaver",
     "status": "completed" | "failed" | "blocked",
     "summary": "Brief summary of refactoring",
     "files_refactored": [],
     "complexity_before": 0,
     "complexity_after": 0,
     "duplication_eliminated": 0,
     "methods_extracted": [],
     "patterns_applied": [],
     "tests_all_passing": true,
     "next_recommendations": [],
     "blockers": []
   }
   </output>
   ```

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
   Run using the Bash tool with description "Logging Weaver completion...": `aether spawn-complete --name "$weaver_name" --status "$weaver_status" --summary "Refactoring $weaver_status"`

   h. **Log to midden:**
   Run using the Bash tool with description "Logging refactoring activity to midden...": `aether midden-write --category "refactoring" --message "Weaver refactored files, complexity before/after: ${complexity_before}/${complexity_after}" --source "weaver"`

5. **Display completion:**
   ```
   🔄🐜 Weaver: Proactive refactoring {weaver_status}
      Files refactored: {count} | Complexity: {before} → {after}
   ```

6. **NON-BLOCKING continuation:**
   The Weaver step is NON-BLOCKING — continue to Step 1.8 regardless of refactoring results.

Continue to Step 1.8.

### Step 1.8: Gatekeeper Security Gate (Conditional)

**Supply chain security audit — runs only when package.json exists.**

First, check for package.json:
Run using the Bash tool with description "Checking for package.json...": `test -f package.json && echo "exists" || echo "missing"`

**If package.json is missing:**
```
📦🐜 Gatekeeper: No package.json found — skipping supply chain audit
```
Continue to Step 1.9.

**If package.json exists:**

1. Generate Gatekeeper name and log spawn:
Run using the Bash tool with description "Generating Gatekeeper name...": `gatekeeper_name=$(aether generate-ant-name "gatekeeper" | jq -r '.result') && aether spawn-log --parent "Queen" --caste "gatekeeper" --name "$gatekeeper_name" --task "Supply chain security audit" --depth 0 && echo "{\"name\":\"$gatekeeper_name\"}"`

2. Display:
```
━━━ 📦🐜 G A T E K E E P E R ━━━
──── 📦🐜 Spawning {gatekeeper_name} — Supply chain security audit ────
```

4. Spawn Gatekeeper agent:

Use the Task tool with subagent_type="aether-gatekeeper" (if available; otherwise use general-purpose and inject the Gatekeeper role from `.opencode/agents/aether-gatekeeper.md`):

```xml
<mission>
Perform supply chain security audit on this codebase.
</mission>

<work>
1. Inventory all dependencies from package.json
2. Scan for known CVEs using npm audit or equivalent
3. Check license compliance for all packages
4. Assess dependency health (outdated, deprecated, maintenance status)
5. Report findings with severity levels
</work>

<output>
Provide JSON output matching this schema:
{
  "ant_name": "your gatekeeper name",
  "caste": "gatekeeper",
  "status": "completed" | "failed" | "blocked",
  "summary": "Brief summary of findings",
  "security": {
    "critical": 0,
    "high": 0,
    "medium": 0,
    "low": 0
  },
  "licenses": {},
  "outdated_packages": [],
  "recommendations": [],
  "blockers": []
}
</output>
```

5. Parse Gatekeeper JSON output and log completion:
Extract: `security.critical`, `security.high`, `status`

Run using the Bash tool with description "Logging Gatekeeper completion...": `aether spawn-complete --name "$gatekeeper_name" --status "completed" --summary "{\"security\":{\"critical\":$critical_count,\"high\":$high_count}}"`

**Gate Decision Logic:**

- **If `security.critical > 0`:**
```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
⛔📦🐜 G A T E K E E P E R   G A T E   F A I L E D
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Critical security vulnerabilities detected: {critical_count}

🚨 CRITICAL CVEs must be fixed before phase advancement.

🔧 Required Actions:
  1. Run `npm audit` to see full details
  2. Fix or update vulnerable dependencies
  3. Run /ant-continue again after resolving

The phase will NOT advance with critical CVEs.
```
**CRITICAL:** Do NOT proceed to Step 1.9. Stop here.

- **If `security.high > 0`:**
```
⚠️📦🐜 Gatekeeper: {high_count} high-severity issues found

Security warnings logged to midden for later review.
Proceeding with caution...
```
Run using the Bash tool with description "Logging high-severity warnings...": `aether midden-write --category "security" --message "High CVEs found: $high_count" --source "gatekeeper"`
Continue to Step 1.9.

- **If clean (no critical or high):**
```
✅📦🐜 Gatekeeper: No critical security issues found
```
Continue to Step 1.9.

### Step 1.9: Auditor Quality Gate (MANDATORY)

**Code quality audit — runs on every `/ant-continue` for consistent coverage.**

1. Generate Auditor name and log spawn:
Run using the Bash tool with description "Generating Auditor name...": `auditor_name=$(aether generate-ant-name "auditor" | jq -r '.result') && aether spawn-log --parent "Queen" --caste "auditor" --name "$auditor_name" --task "Code quality audit" --depth 0 && echo "{\"name\":\"$auditor_name\"}"`

2. Display:
```
━━━ 👥🐜 A U D I T O R ━━━
──── 👥🐜 Spawning {auditor_name} — Code quality audit ────
```

4. Get modified files for audit context:
Run using the Bash tool with description "Getting modified files...": `modified_files=$(git diff --name-only HEAD~1 2>/dev/null || git diff --name-only) && echo "$modified_files"`

5. Spawn Auditor agent:

Use the Task tool with subagent_type="aether-auditor" (if available; otherwise use general-purpose and inject the Auditor role from `.opencode/agents/aether-auditor.md`):

```xml
<mission>
Perform comprehensive code quality audit on this codebase.
</mission>

<work>
1. Review all modified files from the recent commit(s)
2. Apply all 4 audit lenses: security, performance, quality, maintainability
3. Score each finding by severity (CRITICAL/HIGH/MEDIUM/LOW/INFO)
4. Calculate overall quality score (0-100)
5. Document specific issues with file:line references and fix suggestions
</work>

<context>
Phase: {current_phase}
Modified files: {modified_files}
</context>

<output>
Provide JSON output matching this schema:
{
  "ant_name": "your auditor name",
  "caste": "auditor",
  "status": "completed" | "failed" | "blocked",
  "summary": "Brief summary of findings",
  "dimensions_audited": ["security", "performance", "quality", "maintainability"],
  "findings": {
    "critical": 0,
    "high": 0,
    "medium": 0,
    "low": 0,
    "info": 0
  },
  "issues": [
    {"severity": "HIGH", "location": "file:line", "issue": "description", "fix": "suggestion"}
  ],
  "overall_score": 75,
  "recommendation": "Top priority fix",
  "blockers": []
}
</output>
```

6. Parse Auditor JSON output and log completion:
Extract: `findings.critical`, `findings.high`, `findings.medium`, `findings.low`, `findings.info`, `overall_score`, `dimensions_audited`

Run using the Bash tool with description "Logging Auditor completion...": `aether spawn-complete --name "$auditor_name" --status "completed" --summary "{\"findings\":{\"critical\":$critical_count,\"high\":$high_count,\"medium\":$medium_count,\"low\":$low_count,\"info\":$info_count},\"score\":$overall_score}"`

**Gate Decision Logic:**

- **If `findings.critical > 0`:**
```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
⛔👥🐜 A U D I T O R   G A T E   F A I L E D
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Critical code quality issues detected: {critical_count}

🚨 CRITICAL findings must be fixed before phase advancement.

🔧 Required Actions:
  1. Review the critical issues listed below
  2. Fix each critical finding
  3. Run /ant-continue again after resolving

Critical Findings:
{list each critical finding with file:line and description}

The phase will NOT advance with critical quality issues.
```
Run using the Bash tool with description "Logging critical quality block...": `aether error-flag-pattern --name "auditor-critical-findings" --description "$critical_count critical quality issues found" --severity "critical"`
**CRITICAL:** Do NOT proceed to Step 1.10. Stop here.

- **Else if `overall_score < 60`:**
```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
⛔👥🐜 A U D I T O R   G A T E   F A I L E D
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Code quality score below threshold: {overall_score}/100 (threshold: 60)

🚨 Quality score must reach 60+ before phase advancement.

🔧 Required Actions:
  1. Address the top issues preventing score improvement:
{list top 3-5 issues with severity and location}
  2. Focus on HIGH severity items first
  3. Run /ant-continue again after improving quality

The phase will NOT advance with quality score below 60.
```
Run using the Bash tool with description "Logging quality score block...": `aether error-flag-pattern --name "auditor-quality-score" --description "Score $overall_score below threshold 60" --severity "critical"`
**CRITICAL:** Do NOT proceed to Step 1.10. Stop here.

- **Else if `findings.high > 0`:**
```
⚠️👥🐜 Auditor: Quality score {overall_score}/100 — PASSED with warnings

{high_count} high-severity quality issues found:
{list high findings}

Quality warnings logged to midden for later review.
Proceeding with caution...
```
Run using the Bash tool with description "Logging high-quality warnings...": `aether midden-write --category "quality" --message "High severity issues: $high_count (score: $overall_score)" --source "auditor"`
Continue to Step 1.10.

- **If clean (score >= 60, no critical):**
```
✅👥🐜 Auditor: Quality score {overall_score}/100 — PASSED
```
Continue to Step 1.10.

### Step 1.10: TDD Evidence Gate (MANDATORY)

**The Iron Law:** No TDD claims without actual test files.

If Prime Worker reported TDD metrics (tests_added, tests_total, coverage_percent), verify test files exist:

Run using the Bash tool with description "Locating test files...": `find . -name "*.test.*" -o -name "*_test.*" -o -name "*Tests.swift" -o -name "test_*.py" 2>/dev/null | head -10`

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
  1. Run /ant-build {phase} again
  2. Actually write test files (not just claim them)
  3. Tests must exist and be runnable

The phase will NOT advance with fabricated metrics.
```

**CRITICAL:** Do NOT proceed. Log the violation:
```bash
aether error-flag-pattern --name "fabricated-tdd" --description "Prime Worker reported TDD metrics without creating test files" --severity "critical"
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
aether error-add --category "runtime" --severity "critical" --description "{user_description}" --phase {phase}
```

Do NOT proceed to Step 2.

**If "No, haven't tested yet":**
```
⏸️🐜 RUNTIME PENDING — Test the app, then run /ant-continue again.

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
Run using the Bash tool with description "Auto-resolving flags...": `aether flag-auto-resolve`

Then check for remaining blocking flags:
Run using the Bash tool with description "Checking for blockers...": `aether flag-check-blockers`

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
  2. Resolve flags: /ant-flags --resolve {flag_id} "resolution message"
  3. Run /ant-continue again after resolving all blockers
```

**CRITICAL:** Do NOT proceed to Step 2. Do NOT advance the phase.

**If blockers == 0 but issues > 0:**

```
⚠️🐜 FLAGS: {issues} issue(s) noted (non-blocking)

{list each issue flag}

Use /ant-flags to review.
```

Continue to Step 2.

**If all clear (no blockers or issues):**

```
✅🐜 FLAGS GATE PASSED — No blockers.
```

Continue to Step 1.13.

### Step 1.13: Watcher Veto Gate (MANDATORY)

**The Iron Law:** Watcher has final say. If Watcher scores below 7 or reports any CRITICAL findings, all changes are rolled back and phase advancement is blocked.

This gate enforces the Watcher's quality authority by stashing uncommitted work and creating a blocker flag when the Watcher's assessment is negative.

1. **Retrieve Watcher results** from the most recent build:
   Run using the Bash tool with description "Retrieving Watcher results...":
   ```bash
   watcher_result=$(aether state-read '.build_synthesis.watcher' 2>/dev/null || echo "{}")
   quality_score=$(echo "$watcher_result" | jq -r '.quality_score // 0')
   critical_count=$(echo "$watcher_result" | jq '[.issues_found[]? | select(.severity == "CRITICAL")] | length')
   echo "{\"quality_score\": $quality_score, \"critical_count\": $critical_count}"
   ```

   If Watcher results are not available in state (e.g., no Watcher was spawned), skip this gate with:
   ```
   ⏭️👁️🐜 Watcher Veto: No Watcher results found — skipping veto check
   ```
   Continue to Step 2.

2. **Evaluate veto conditions:**

   **If `quality_score < 7` OR `critical_count > 0`:**

   ```
   ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
   👁️🐜 W A T C H E R   V E T O
   ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

   Watcher has VETOED phase advancement.

   Quality Score: {quality_score}/10 (minimum: 7)
   Critical Issues: {critical_count}
   ```

   a. **Stash all uncommitted changes:**
   Run using the Bash tool with description "Stashing changes due to Watcher veto...":
   ```bash
   git stash push -m "watcher-veto-phase-$current_phase" 2>&1
   ```

   b. **Create ROLLBACK_VETO blocker flag:**
   Run using the Bash tool with description "Creating ROLLBACK_VETO blocker flag...":
   ```bash
   aether flag-create "WATCHER VETO: Quality score $quality_score (minimum 7), $critical_count critical issue(s). Changes stashed." --type blocker --phase "$current_phase"
   ```

   c. **Log the veto to midden:**
   Run using the Bash tool with description "Logging Watcher veto to midden...": `aether midden-write --category "watcher-veto" --message "Watcher vetoed phase $current_phase: score $quality_score, $critical_count critical issues" --source "watcher"`

   d. **Display required actions:**
   ```
   Changes from this phase have been stashed (git stash).
   A ROLLBACK_VETO blocker flag has been created.

   Required Actions:
     1. Review and fix all CRITICAL and HIGH issues identified by Watcher
     2. Restore changes: git stash pop
     3. Re-run /ant-build {current_phase} after fixes
     4. Watcher must re-verify with quality_score >= 7 and no CRITICAL issues

   Phase advancement is BLOCKED until Watcher approves.
   ```

   **CRITICAL:** Do NOT proceed to Step 2. Do NOT advance the phase. Stop here.

   **If `quality_score >= 7` AND `critical_count == 0`:**

   ```
   ✅👁️🐜 WATCHER VETO GATE PASSED — Score {quality_score}/10, no critical issues
   ```

   Continue to Step 2.


### Step 1.14: Medic Health Gate (Conditional Auto-Spawn)

**Colony health check — auto-spawns Medic when critical issues detected.**

1. Run auto-spawn check:
Run using the Bash tool with description "Checking colony health for Medic auto-spawn...": `aether medic-auto-spawn-check 2>/dev/null || echo "{\"should_spawn\":false}"`

2. Parse result. If `should_spawn` is `false`:
```
🩹🐜 Medic: Colony healthy — no auto-spawn needed
```
Continue to Step 2.

3. If `should_spawn` is `true`:

Generate Medic name and log spawn:
Run using the Bash tool with description "Generating Medic name...": `medic_name=$(aether generate-ant-name "medic" | jq -r '.result') && aether spawn-log --parent "Queen" --caste "medic" --name "$medic_name" --task "Colony health diagnosis: {reason}" --depth 0 && echo "{\"name\":\"$medic_name\"}"`

Display:
```
━━━ 🩹🐜 M E D I C ━━━
──── 🩹🐜 Spawning {medic_name} — Colony health diagnosis ────
```

Spawn Medic agent:

Use the Agent tool with subagent_type="aether-medic":

```xml
<mission>
Colony health has been flagged for automatic diagnosis.
Reason: {reason from auto-spawn check}

Perform a deep health scan and report findings.
</mission>

<work>
1. Run `aether medic --deep` to scan all colony data
2. Identify critical issues and root causes
3. If fixable issues found, recommend running `aether medic --fix`
4. Return structured health report
</work>

<output>
Provide JSON output matching this schema:
{
  "ant_name": "your medic name",
  "caste": "medic",
  "status": "completed" | "failed" | "blocked",
  "summary": "Brief health summary",
  "critical_count": 0,
  "warning_count": 0,
  "fixable_count": 0,
  "recommendations": [],
  "blockers": []
}
</output>
```

Log completion:
Run using the Bash tool with description "Logging Medic completion...": `aether spawn-complete --name "$medic_name" --status "completed" --summary "{\"critical_count\":$critical_count,\"warning_count\":$warning_count}"`

**Gate Decision Logic:**

- **If `critical_count > 0`:**
```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
🩹🐜 M E D I C   G A T E   F A I L E D
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Critical health issues detected: {critical_count}
Colony may not be in a healthy state for advancement.

Issues:
  {list critical issues}

Recommended Actions:
  1. Review the critical issues above
  2. Run `aether medic --fix` to attempt repairs
  3. Re-run /ant-continue after repairs
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

**CRITICAL:** Do NOT proceed to Step 2. Do NOT advance the phase.
Log the block:
```bash
aether activity-log --command "BLOCKED" --details "medic: {critical_count} critical health issues detected"
```

- **If `critical_count == 0` but `warning_count > 0`:**
```
🩹🐜 Medic Gate: Warnings detected ({warning_count}) — phase advancement allowed with caution
```
Continue to Step 2.

- **If no issues found:**
```
✅🩹🐜 MEDIC GATE PASSED — Colony is healthy
```
Continue to Step 2.
