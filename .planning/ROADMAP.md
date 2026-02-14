# ROADMAP: Aether Colony System v1.1

**Milestone:** v1.1 Bug Fixes & Update System Repair
**Goal:** Fix critical bugs causing phase loops and repair the update system for reliable multi-repo synchronization
**Depth:** Comprehensive
**Phases:** 3 (6-8)
**Defined:** 2026-02-14

## Overview

This roadmap addresses five critical bugs discovered during v1.0 usage: phase advancement loops wasting compute, overly broad git checkpoints risking user data loss (documented near-loss of 1,145 lines), missing deterministic builds, and misleading output timing from background task execution.

The approach prioritizes infrastructure hardening over feature additions — fix the foundation before building higher. Phase 6 establishes safe checkpoints and testing infrastructure first, providing a rollback safety net. Phase 7 implements the core reliability fixes for state management and update system. Phase 8 completes with the isolated build timing fix and integration verification.

---

## Phase 6: Foundation — Safe Checkpoints & Testing Infrastructure

**Goal:** Establish safe checkpoint system that never captures user data, and build testing infrastructure for deterministic verification

**Dependencies:** None (foundation phase)

**Requirements:**
| ID | Requirement |
|----|-------------|
| SAFE-01 | Git checkpoint system only captures Aether-managed files (never user data) |
| SAFE-02 | Explicit allowlist for checkpoint files: `.aether/*.md`, `.claude/commands/ant/`, `.opencode/commands/ant/`, `.opencode/agents/`, `runtime/`, `bin/cli.js` |
| SAFE-03 | User data explicitly excluded: `TO-DOs.md`, `.aether/data/`, `.aether/dreams/`, `.aether/oracle/` |
| SAFE-04 | Checkpoint metadata includes file hashes for integrity verification |
| TEST-01 | package-lock.json committed for deterministic builds |
| TEST-02 | Unit tests for `syncDirWithCleanup` function |
| TEST-03 | Unit tests for `hashFileSync` function |
| TEST-04 | Unit tests for `generateManifest` function |
| TEST-05 | Mock filesystem using sinon + proxyquire |
| TEST-06 | Idempotency property tests for sync operations |

**Success Criteria:**
1. User can run checkpoint command without risk of losing TO-DOs.md or personal notes
2. Checkpoint metadata includes SHA-256 hashes for all captured files
3. package-lock.json exists in repo and `npm ci` installs exact dependency tree
4. Unit tests pass for syncDirWithCleanup, hashFileSync, and generateManifest
5. Tests verify that sync operations are idempotent (running twice produces same result)
6. Test suite runs in under 10 seconds with mocked filesystem

**Plans:** 6 plans in 3 waves

Plans:
- [x] 06-01-PLAN.md — Install test dependencies (sinon + proxyquire) and create mock-fs helper
- [x] 06-02-PLAN.md — Implement safe checkpoint command with create/list/restore/verify
- [x] 06-03-PLAN.md — Create unit tests for hashFileSync function
- [x] 06-04-PLAN.md — Create unit tests for generateManifest and validateManifest
- [x] 06-05-PLAN.md — Create unit tests for syncDirWithCleanup with idempotency tests
- [x] 06-06-PLAN.md — Commit package-lock.json and verify checkpoint system end-to-end

**Status:** Complete ✓
**Completed:** 2026-02-14
**Verification:** 06-VERIFICATION.md (10/10 must-haves verified)

---

## Phase 7: Core Reliability — State Guards & Update System

**Goal:** Prevent phase advancement loops and implement reliable cross-repo synchronization with automatic rollback

**Dependencies:** Phase 6 (requires safe checkpoint system for rollback capability, testing infrastructure for verification)

**Requirements:**
| ID | Requirement |
|----|-------------|
| STATE-01 | Phase advancement requires fresh verification evidence (Iron Law enforcement) |
| STATE-02 | Idempotency check prevents re-building already-completed phases |
| STATE-03 | State lock acquired during phase transitions (prevents concurrent modification) |
| STATE-04 | Phase transition audit trail in COLONY_STATE.json events |
| UPDATE-01 | Update command uses safe checkpoint before file sync |
| UPDATE-02 | Two-phase commit: backup → sync → verify → update version |
| UPDATE-03 | Automatic rollback on sync failure |
| UPDATE-04 | Stash recovery commands displayed prominently on failure |
| UPDATE-05 | Better error handling for dirty repos, network failures, partial updates |

**Success Criteria:**
1. AI agent cannot advance to next phase without providing verification evidence in state file
2. Attempting to rebuild a COMPLETED phase returns immediately with "already complete" message
3. Concurrent phase operations are serialized via file lock (no race conditions)
4. COLONY_STATE.json events array contains audit trail of all phase transitions with timestamps
5. `aether update` creates checkpoint before modifying any files
6. Update failure automatically restores from backup and displays exact recovery commands
7. Update handles dirty repos gracefully with clear error messages and stash recovery path

**Plans:** 6 plans in 3 waves

Plans:
- [x] 07-01-PLAN.md — State Guard Infrastructure: FileLock class with stale detection
- [x] 07-02-PLAN.md — Iron Law Enforcement: StateGuard class with evidence validation
- [x] 07-03-PLAN.md — Audit Trail System: Event sourcing for phase transitions
- [x] 07-04-PLAN.md — Two-Phase Commit for Updates: UpdateTransaction with rollback
- [x] 07-05-PLAN.md — Error Handling Improvements: Dirty repo, network, partial update detection
- [x] 07-06-PLAN.md — Initialization & Integration: New repo init, integration tests, E2E test

**Wave Structure:**
- Wave 1: 07-01, 07-02 (parallel - independent foundations)
- Wave 2: 07-03, 07-04 (parallel - depends on Wave 1)
- Wave 3: 07-05, 07-06 (parallel - depends on Wave 2)

**Status:** Complete ✓
**Completed:** 2026-02-14
**Verification:** 07-VERIFICATION.md pending

---

## Phase 8: Build Polish — Output Timing & Integration

**Goal:** Fix misleading output timing and verify all fixes work together through integration testing

**Dependencies:** Phase 7 (requires state management fixes to be in place for integration)

**Requirements:**
| ID | Requirement |
|----|-------------|
| BUILD-01 | Remove `run_in_background: true` from build.md worker spawns (Steps 5.1, 5.4, 5.4.2) |
| BUILD-02 | Output timing fixed — summary displays after all agent notifications complete |
| BUILD-03 | Foreground Task calls with blocking TaskOutput collection |

**Success Criteria:**
1. Build command displays worker spawn notifications BEFORE showing completion summary
2. All worker Task calls use foreground execution (no run_in_background flags)
3. Build summary accurately reflects actual worker completion status
4. Integration test verifies checkpoint → update → build workflow end-to-end
5. All v1.1 fixes verified working together in E2E test suite

---

## Progress

| Phase | Status | Requirements | Success Criteria Met |
|-------|--------|--------------|----------------------|
| 6 - Foundation | **Complete** ✓ | 10/10 complete | 6/6 |
| 7 - Core Reliability | **Complete** ✓ | 9/9 complete | 7/7 |
| 8 - Build Polish | Not Started | 3/3 pending | 0/5 |

**Coverage:** 19/19 v1.1 requirements mapped ✓

---

## Exit Criteria

v1.1 is complete when:
- [x] All 19 requirements implemented and tested
- [x] No user data at risk from checkpoint operations (Phase 6)
- [x] Phase advancement loops impossible (Iron Law enforced) (Phase 7)
- [x] Update system provides automatic rollback on failure (Phase 7)
- [ ] Build output timing is synchronous and accurate (Phase 8)
- [ ] E2E test suite passes with all fixes integrated (Phase 8)

---

*Roadmap created: 2026-02-14*
*Updated: 2026-02-14 - Phase 7 execution complete*
