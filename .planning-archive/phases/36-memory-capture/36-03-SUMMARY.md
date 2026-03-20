---
phase: 36-memory-capture
plan: "03"
subsystem: colony-memory
tags: [midden, failure-logging, learning-observe]
dependency_graph:
  requires:
    - 36-01 (learning-observe foundation)
  provides:
    - Failure capture infrastructure
    - Structured failure logging
  affects:
    - /ant:build command
    - Worker prompts
tech_stack:
  added: []
  patterns:
    - YAML structured logging
    - Midden pattern for failure capture
key-files:
  created:
    - .aether/midden/build-failures.md
    - .aether/midden/test-failures.md
    - .aether/midden/approach-changes.md
  modified:
    - .aether/aether-utils.sh (failure type)
    - .claude/commands/ant/build.md (logging hooks)
decisions:
  - Midden location changed from .aether/data/midden/ to .aether/midden/ to respect data/ protection
  - Failure type uses threshold=1 for immediate promotion consideration
  - Workers can self-report approach changes via convention in prompts
metrics:
  duration: "45 minutes"
  completed_date: "2026-02-21"
---

# Phase 36 Plan 03: Midden Failure Logging Integration

## Summary

Integrated failure logging into `/ant:build` so worker failures, test failures, and approach changes are captured in structured midden files AND recorded as observations for potential promotion to QUEEN.md.

## One-Liner

"Build command now auto-logs failures to structured midden files and records them as learning observations."

## What Was Built

### 1. "failure" Wisdom Type (Task 1)

Added "failure" to the `valid_types` array in `learning-observe` function:
- Location: `.aether/aether-utils.sh` around line 4100
- Threshold: 1 (promotes after 1 observation + approval)
- Purpose: Failed approaches are as valuable as successful ones

### 2. Midden Directory Structure (Task 2)

Created `.aether/midden/` directory (outside `data/` for protection):

| File | Purpose | Format |
|------|---------|--------|
| `build-failures.md` | Worker failures during build | YAML list entries |
| `test-failures.md` | Test/verification failures | YAML list entries |
| `approach-changes.md` | Approach switches (X didn't work, tried Y) | YAML list entries |

All files use structured YAML with fields: timestamp, phase, colony, worker, what_failed, why, what_worked

### 3. Build.md Failure Logging (Tasks 3-4)

Added automatic failure logging to three build steps:

**Step 5.2 - Builder Failures:**
- Logs to `midden/build-failures.md`
- Records: worker name, task, blockers, error type
- Calls `learning-observe` with type=failure

**Step 5.8 - Watcher Verification Failures:**
- Logs to `midden/test-failures.md`
- Records: issue title, description, severity
- Calls `learning-observe` with type=failure

**Step 5.7 - Chaos Resilience Findings:**
- Logs critical/high findings to `midden/build-failures.md`
- Records: finding title, description, severity
- Calls `learning-observe` with type=failure

### 4. Approach Change Convention (Task 5)

Added self-reporting convention to Builder worker prompt:
- Workers can log when they try X, it fails, and switch to Y
- Logs to `midden/approach-changes.md`
- Fields: tried, why_it_failed, switched_to

## Deviations from Plan

### Path Change: .aether/data/midden/ → .aether/midden/

**User decision during checkpoint:** Place midden files at `.aether/midden/` instead of `.aether/data/midden/`.

**Rationale:** Respects the `data/` protection boundary while keeping failure logs with other `.aether/` system files.

**Impact:** All references updated in:
- Plan file frontmatter (documented intention)
- build.md logging code (4 locations)
- Worker prompt approach change example

## Verification Results

| Check | Result |
|-------|--------|
| "failure" in valid_types | ✓ |
| "failure" threshold=1 in case statement | ✓ |
| Three midden .md files exist | ✓ |
| build.md Step 5.2 logs builder failures | ✓ |
| build.md logs watcher verification failures | ✓ |
| build.md logs chaos resilience findings | ✓ |
| All failure logging includes learning-observe | ✓ (3 calls) |
| Worker prompts include approach change convention | ✓ |

## Commits

| Hash | Message |
|------|---------|
| a2d305a | feat(36-03): add failure wisdom type to learning-observe |
| e71b33a | feat(36-03): create midden directory and log files |
| bff2c8e | feat(36-03): add failure logging to build.md |
| 908158b | feat(36-03): add approach change logging convention to build.md |

## Self-Check: PASSED

- [x] All created files exist at `.aether/midden/`
- [x] All commits recorded and verified
- [x] build.md contains 3 learning-observe calls
- [x] build.md contains 4 midden path references
- [x] Approach change convention documented in worker prompt
