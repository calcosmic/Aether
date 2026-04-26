---
phase: 58-smart-review-depth
plan: 01
subsystem: review-depth
tags: [tdd, cli-flags, depth-resolution]
dependency_graph:
  requires: []
  provides: [resolveReviewDepth, phaseHasHeavyKeywords, chaosShouldRunInLightMode, --light, --heavy]
  affects: [cmd/codex_build.go, cmd/codex_continue.go]
tech_stack:
  added: []
  patterns: [type-alias constants, table-driven tests, deterministic modulo sampling]
key_files:
  created:
    - cmd/review_depth.go
    - cmd/review_depth_test.go
  modified:
    - cmd/codex_workflow_cmds.go
decisions:
  - ReviewDepth is a string type alias with light/heavy constants (not an iota enum) for JSON readability
  - 12 heavy keywords matched via case-insensitive substring against phase names
  - chaosShouldRunInLightMode uses phaseID % 10 < 3 for deterministic ~30% sampling
  - heavy flag wins over light flag when both are set (safer default)
metrics:
  duration: 3m
  completed: "2026-04-27"
  tasks: 2
  files: 3
  tests_added: 36
---

# Phase 58 Plan 01: Depth Resolution and CLI Flags Summary

Pure depth resolution logic and CLI flags that determine whether a phase gets light or heavy review.

## What Was Built

Three core functions in `cmd/review_depth.go` that form the foundation layer for smart review depth:

1. **`resolveReviewDepth()`** -- Determines light vs heavy review for a phase based on: final-phase override, explicit --heavy flag, keyword auto-detection, or default-to-light. Priority chain is unambiguous and safe (heavy always wins in conflict).

2. **`phaseHasHeavyKeywords()`** -- Case-insensitive substring matching against 12 security/release keywords (security, auth, crypto, secrets, permissions, compliance, audit, release, deploy, production, ship, launch).

3. **`chaosShouldRunInLightMode()`** -- Deterministic 30% sampling via `phaseID % 10 < 3`, ensuring chaos agent runs on a predictable subset of light-review phases.

Four CLI flags registered on build and continue commands: `--light` and `--heavy` on both `buildCmd` and `continueCmd`, following existing Cobra Bool flag patterns.

## TDD Gate Compliance

| Gate | Commit | Hash |
|------|--------|------|
| RED (depth logic) | test(58-01): add failing tests for review depth resolution | a902701d |
| GREEN (depth logic) | feat(58-01): implement review depth resolution logic | 983c31c8 |
| RED (flags) | test(58-01): add failing tests for --light/--heavy flags | 0480c6f2 |
| GREEN (flags) | feat(58-01): register --light and --heavy flags on build and continue | 915c821d |

All four gate commits present and verified in git log.

## Commits

| Hash | Message |
|------|---------|
| a902701d | test(58-01): add failing tests for review depth resolution |
| 983c31c8 | feat(58-01): implement review depth resolution logic |
| 0480c6f2 | test(58-01): add failing tests for --light/--heavy flags |
| 915c821d | feat(58-01): register --light and --heavy flags on build and continue |

## Deviations from Plan

None - plan executed exactly as written.

## Self-Check

- cmd/review_depth.go: exists
- cmd/review_depth_test.go: exists (244 lines, exceeds 100 minimum)
- cmd/codex_workflow_cmds.go: modified with 4 new flag registrations
- All 4 gate commits verified in git log
- All tests passing (36 test cases across 12 test functions)
- Binary builds cleanly

## Self-Check: PASSED

## Known Stubs

None.

## Threat Flags

None. No new network endpoints, auth paths, or trust boundaries beyond those documented in the plan threat model.
