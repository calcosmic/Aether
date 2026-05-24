# Playbook: Continue

> Consolidated from continue-verify, continue-gates, continue-advance, and continue-finalize.

## Overview

The Queen reconciles completed build work, runs verification loops and gates, and advances to the next phase.

## Stage 1: Verify

### Step 1: Read State

Read `.aether/data/COLONY_STATE.json`. Auto-upgrade old state if needed.

Validate:
- If `goal: null` -> stop, suggest `/ant-init`
- If `milestone == "Crowned Anthill"` -> stop, suggest new colony
- If `plan.phases` empty -> stop, suggest `/ant-plan`

### Step 1.5: Load State and Show Resumption Context

Run `aether load-state`. Display brief resumption context. Release lock via `aether unload-state`.

### Step 1.5: Verification Loop Gate (Mandatory)

**Iron Law:** No phase advancement without fresh verification evidence.

#### 1. Command Resolution (Priority Chain)

Resolve build/test/type/lint commands:
1. CLAUDE.md system context
2. `.aether/data/codebase.md` ## Commands
3. Fallback heuristic table (package.json, Cargo.toml, go.mod, etc.)

#### 2. Run 6-Phase Verification Loop

```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
👁️🐜 V E R I F I C A T I O N   L O O P
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

- **Phase 1: Build Check** — run build command, record exit code
- **Phase 2: Type Check** — run type command, record errors
- **Phase 3: Lint Check** — run lint command, record warnings/errors
- **Phase 4: Test Check** — run test command, record pass/fail
- **Coverage Check** — if command exists, record percentage
- **Phase 5: Secrets Scan** — grep for exposed secrets and debug artifacts
- **Phase 6: Diff Review** — `git diff --stat`

Success Criteria Check: read phase success criteria and verify each with evidence.

Display Verification Report with PASS/FAIL per check.

#### 3. Gate Decision

If NOT READY (build fails, tests fail, critical security issues, success criteria unmet):
- Display failures and required actions
- Do NOT proceed to Step 2

If READY:
- Proceed to gate checks

#### Step 1.5.1: Probe Coverage Agent (Conditional)

If coverage < 80% and tests passed, spawn Probe to generate tests for uncovered paths. Probe is strictly non-blocking.

#### Step 1.5.3: Verify Worker Claims (Mandatory)

Cross-reference builder claims from `.aether/data/last-build-claims.json` against reality. If mismatches found, auto-retry build once. If still blocked, halt.

## Stage 2: Gates

### Step 1.6: Spawn Enforcement Gate (Mandatory)

**Iron Law:** No phase advancement without worker spawning for non-trivial phases.

Check `.aether/data/spawn-tree.txt` for spawn and watcher counts.

- If spawn_count == 0 and phase had 3+ tasks: HARD REJECTION
- If watcher_count == 0: HARD REJECTION

### Step 1.7: Anti-Pattern Gate

Scan modified/created files via `aether check-antipattern`. Critical issues block advancement.

### Step 1.7.1: Proactive Refactoring Gate (Conditional)

If code exceeds complexity thresholds (file > 300 lines, function > 50 lines, directory density > 10 files), spawn Weaver to refactor. Non-blocking.

### Step 1.8: Gatekeeper Security Gate (Conditional)

If `package.json` exists, spawn Gatekeeper to audit dependencies for CVEs and license compliance. Critical CVEs block advancement.

### Step 1.9: Auditor Quality Gate (Mandatory)

Spawn Auditor for multi-lens quality audit. Critical findings or score < 60 block advancement.

### Step 1.10: TDD Evidence Gate (Mandatory)

If Prime Worker claimed tests but no test files found: HARD REJECTION for fabricated metrics.

### Step 1.11: Runtime Verification Gate (Mandatory)

Ask user to confirm the app actually runs. If not tested, do not proceed.

### Step 1.12: Flags Gate (Mandatory)

Auto-resolve eligible flags. If blockers remain, display them and halt.

### Step 1.13: Watcher Veto Gate (Mandatory)

If Watcher quality score < 7 or critical issues found, present choices:
1. Keep changes and retry
2. Keep working (stay blocked)
3. Force advance (accept risk)

### Step 1.14: Medic Health Gate (Conditional)

If `aether medic-auto-spawn-check` indicates issues, spawn Medic. Critical health issues block advancement.

## Stage 3: Advance

### Step 2: Update State

Mark current phase `completed`. Extract learnings (as hypotheses). Extract instincts from patterns. Advance `current_phase` and set `state` to `"READY"`.

Cap enforcement: max 20 phase_learnings, 30 decisions, 30 instincts, 100 events.

### Step 2.0.4: Worktree Merge-Back (Non-blocking)

If parallel mode is `worktree`, merge any remaining worktree branches back to main.

### Step 2.0.5: Pheromone Merge-Back (Non-blocking)

If `pheromone-branch-export.json` exists, merge branch signals into main.

### Step 2.0.6: Midden Collection (Non-blocking)

Collect failure records from recently merged branch worktrees.

### Step 2.0.7: Cross-PR Midden Analysis (Non-blocking)

Run cross-PR analysis to detect systemic failure patterns.

### Step 2.1: Auto-Emit Phase Pheromones (Silent)

- 2.1a: FEEDBACK for phase outcome
- 2.1b: FEEDBACK for recent decisions
- 2.1c: REDIRECT for recurring midden error patterns
- 2.1d: FEEDBACK for recurring success criteria
- 2.1e: Expire `phase_end` signals

### Step 2.1.5: Check for Promotion Proposals

If observations meet thresholds, present tick-to-approve UX via `aether learning-approve-proposals`.

### Step 2.1.6: Batch Wisdom Auto-Promotion

Sweep observations and auto-promote to QUEEN.md if thresholds met.

## Stage 4: Finalize

### Step 2.2: Update Handoff Document

Write `.aether/HANDOFF.md` with post-advancement state for session recovery.

### Step 2.3: Update Changelog

Append changelog entry for completed phase via `aether changelog-append`.

### Step 2.4: Commit Suggestion (Optional)

Suggest a commit with enhanced message from `aether generate-commit-message`. Non-blocking.

### Step 2.5: Context Clear Suggestion (Optional)

Suggest clearing context before next phase. Non-blocking.

### Step 2.6: Update Context Document

Log activity and update phase in `.aether/CONTEXT.md`.

### Step 2.7: Project Completion

If all phases complete, display completion report with learnings and wisdom summary.

### Step 3: Display Result

Show phase advancement summary:
- Completed phase
- Learnings extracted
- Wisdom promoted
- Instincts updated
- Next phase with tasks and state

### Step 4: Update Session

Run `aether session-update` for resume support.

### Step 4.5: Housekeeping

Prune stale backups and temp files. Non-blocking.
