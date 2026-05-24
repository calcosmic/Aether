# Playbook: Swarm

## Overview

Parallel bug investigation. The Queen dispatches multiple specialists simultaneously to investigate, track, and fix a bug.

## Stage 1: Investigate

### Step 1: Parse Bug Report

Extract from `$ARGUMENTS`:
- Bug description
- Priority (critical, high, medium, low)
- Affected areas (if known)
- Reproduction steps (if provided)

If bug description is empty, show usage and stop.

### Step 2: Initialize Swarm Session

Run `aether load-state` to get colony context.

Update state:
- Set `state` to `"SWARMING"`
- Add `swarm_target` with bug description
- Append event: `"<timestamp>|swarm_started|swarm|Bug: {description}"`

### Step 3: Dispatch Tracker

Generate Tracker name: `aether generate-ant-name "tracker"`
Log spawn: `aether spawn-log --parent "Queen" --caste "tracker" --name "{name}" --task "Bug investigation: {description}" --depth 0`

Spawn Tracker with prompt:
- Investigate bug in codebase
- Find root cause with file:line references
- Identify related code paths
- Return JSON with: `root_cause`, `affected_files`, `severity`, `reproduction_steps`

### Step 4: Dispatch Fixer (Parallel with Tracker)

Generate Fixer name: `aether generate-ant-name "fixer"`
Log spawn: `aether spawn-log --parent "Queen" --caste "fixer" --name "{name}" --task "Auto-fix: {description}" --depth 0`

Spawn Fixer with prompt:
- Attempt to fix the bug automatically
- Write tests that reproduce the bug first (TDD)
- Apply minimal fix
- Verify fix with tests
- Return JSON with: `fix_applied`, `files_modified`, `tests_added`, `verification_result`

## Stage 2: Track

### Step 5: Process Tracker Results

Parse Tracker JSON:
- If root cause found: store for Fixer reference
- If not found: spawn additional Scout to broaden search

Display:
```
🔍 Tracker: {root_cause_summary}
   Affected: {affected_files}
   Severity: {severity}
```

Log to midden if critical: `aether midden-write --category "bug" --message "{description}: {root_cause}" --source "tracker"`

### Step 6: Process Fixer Results

Parse Fixer JSON:
- If fix applied and verified: proceed to verification
- If fix failed: analyze why and decide next action

Display:
```
🔧 Fixer: {status}
   Files modified: {count}
   Tests added: {count}
```

## Stage 3: Fix

### Step 7: Validate Fix

If Fixer succeeded, run verification:
1. Build check
2. Test check (including new tests)
3. Type check
4. Diff review

If all pass, proceed to finalize.

If Fixer failed or verification fails:
- Display failure details
- Offer options:
  1. Retry with more context
  2. Escalate to human
  3. Skip and document

### Step 8: Spawn Builder (if needed)

If the fix requires broader changes (refactoring, API changes), spawn Builder with:
- Tracker findings as context
- Fixer attempt as reference
- Clear scope to avoid over-engineering

## Stage 4: Verify

### Step 9: Final Verification

Run full verification loop:
- Build passes
- All tests pass (including new ones)
- No regressions in related areas
- Anti-pattern scan clean

### Step 10: Update State

If fix verified:
- Mark swarm as completed
- Add fix details to `memory.bug_fixes`
- Append event: `"<timestamp>|swarm_completed|swarm|Bug fixed: {description}"`

If not verified:
- Mark as blocked
- Create blocker flag
- Update handoff with failure context

### Step 11: Display Results

```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
🐜 S W A R M   R E S U L T S
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Bug: {description}
Status: {fixed|blocked|failed}

Investigation:
  🔍 Tracker: {root_cause}
  🔧 Fixer: {fix_summary}

Verification:
  Build: {PASS/FAIL}
  Tests: {PASS/FAIL}

Next steps:
  {recommendation}
```

### Step 12: Update Session

Run `aether session-update --command "/ant-swarm" --suggested-next "/ant-continue" --summary "Swarm completed: {status}"`
