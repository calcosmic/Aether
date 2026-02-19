---
phase: 18-reliability-architecture-gaps
verified: 2026-02-19T16:30:00Z
status: passed
score: 9/9 must-haves verified
re_verification: true
gaps: []
human_verification:
  - test: "Run aether help and confirm Queen Commands section is visible"
    expected: "Three commands listed: queen-init, queen-read, queen-promote, each with a one-liner description"
    why_human: "Visual formatting of help output depends on how the consumer renders the JSON"
---

# Phase 18: Reliability Architecture Gaps — Verification Report

**Phase Goal:** Stale resources stop accumulating, exec errors are caught, queen commands are discoverable, and JSON output is validated before leaving the read layer
**Verified:** 2026-02-19T16:30:00Z
**Status:** passed
**Re-verification:** Yes — gap fixed (file-existence guard added to _migrate_colony_state, commit babb7ba)

---

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | Feature detection runs AFTER all fallback definitions — no startup race | VERIFIED | `json_err()` fallback at line 68; feature_disable block at line 83 (feature detection follows fallback). Bash test 24 passes. |
| 2 | Composed EXIT trap fires both cleanup_locks and cleanup_temp_files on exit | VERIFIED | `_aether_exit_cleanup()` at line 100 calls both; `trap '_aether_exit_cleanup' EXIT TERM INT HUP` at line 104. Bash test 25 passes. |
| 3 | Orphaned .tmp files from dead PIDs are removed on startup | VERIFIED | `_cleanup_orphaned_temp_files()` at line 108 uses kill -0 PID liveness check; called at line 120 via `type cleanup_temp_files && _cleanup_orphaned_temp_files`. |
| 4 | Spawn-tree.txt is rotated at session-init with 5-archive cap | VERIFIED | `_rotate_spawn_tree()` defined and called inside `session-init)` case at lines 5126-5138; archives to `$DATA_DIR/spawn-tree-archive/spawn-tree.${archive_ts}.txt`; `ls -t | tail -n +6 | xargs rm -f` enforces 5-archive cap. Bash test 26 passes. |
| 5 | model-get and model-list produce clear JSON errors when model-profile fails — not ERR trap | VERIFIED | `exec bash` pattern removed (0 matches); subprocess pattern at lines 2438-2444 and 2450-2455 with `set +e; result=$(bash "$0" model-profile ...); exit_code=$?; set -e; if [[ $exit_code -ne 0 ]]; then json_err ...`. Bash tests 29-30 pass. |
| 6 | Failed spawn completions are logged to COLONY_STATE.json events array | VERIFIED | `spawn_failed` event logged at line 993 inside `if [[ "$status" == "failed" ]] || [[ "$status" == "error" ]]` block; uses atomic_write on COLONY_STATE.json events array. |
| 7 | aether help shows Queen Commands section with queen-init, queen-read, queen-promote | VERIFIED | `bash .aether/aether-utils.sh help` returns sections key with 10 groups; Queen Commands section contains `[queen-init, queen-read, queen-promote]` each with name + description. Flat commands array preserved (88 commands). Bash test 31 passes. |
| 8 | queen-commands.md is in .aether/docs/ and in both sync allowlists | VERIFIED | File exists at `.aether/docs/queen-commands.md` (3098 bytes, covers all 3 commands with usage examples). Listed in `bin/sync-to-runtime.sh` line 54 and `bin/lib/update-transaction.js` line 210. |
| 9 | queen-read rejects malformed METADATA with E_JSON_INVALID — no ERR trap fire | VERIFIED | Gate 1 at line 3411 validates metadata with `jq -e .` before `--argjson` use. Gate 2 at line 3451 validates assembled result before `json_ok`. Both emit E_JSON_INVALID with actionable "Try:" suggestions. Bash test 27 passes. |
| 10 | validate-state colony auto-migrates pre-3.0 state files with W_MIGRATED notification | VERIFIED | `_migrate_colony_state()` at line 700 checks JSON validity, detects version != "3.0", adds missing fields (signals, graveyards, events) with additive-only jq, writes via atomic_write, emits W_MIGRATED to stderr. Bash test 28 passes. |
| 11 | Corrupt COLONY_STATE.json triggers backup + E_JSON_INVALID — not silent failure | VERIFIED | `_migrate_colony_state` lines 704-711: `if ! jq -e . "$state_file"` then `create_backup` + `json_err "$E_JSON_INVALID"`. Backup directory confirmed at `.aether/data/backups/`. |
| 12 | All 31 bash regression tests pass | VERIFIED | `bash tests/bash/test-aether-utils.sh`: 31 tests run, 0 failures. Includes 8 new Phase 18 tests (ARCH-02, ARCH-03, ARCH-06, ARCH-07x2, ARCH-08, ARCH-09, ARCH-10). |
| 13 | AVA test suite has no new regressions from Phase 18 | VERIFIED | File-existence guard `[[ -f "$state_file" ]] || return 0` added to _migrate_colony_state (commit babb7ba). 2 remaining AVA failures are pre-existing (Phase 17 error structure change). 31/31 bash tests pass. |

**Score:** 13/13 truths verified

---

## Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `.aether/aether-utils.sh` | Startup ordering, composed trap, orphan cleanup, spawn-tree rotation (18-01) | VERIFIED | Feature detection at line 83 (after fallback at 68); `_aether_exit_cleanup` at line 100; `_cleanup_orphaned_temp_files` at line 108; `_rotate_spawn_tree` at line 5126 |
| `.aether/aether-utils.sh` | model-get/model-list subprocess error handling, spawn-complete failure logging (18-02) | VERIFIED | Subprocess pattern at lines 2432-2456; `spawn_failed` event at line 993; no `exec bash.*model-profile` |
| `.aether/aether-utils.sh` | Help sections key with Queen Commands (18-03) | VERIFIED | `sections` key with 10 groups; Queen Commands: [queen-init, queen-read, queen-promote] with descriptions; 88-command flat array preserved |
| `.aether/aether-utils.sh` | queen-read validation gates, validate-state schema migration (18-04) | VERIFIED | Gate 1 at 3411, Gate 2 at 3451; `_migrate_colony_state` at 700; `W_MIGRATED` at 729; corrupt backup at 706-711 |
| `.aether/aether-utils.sh` | Contains "Couldn't get model" error message (18-02 artifact spec) | VERIFIED | Line 2443: `"Couldn't get model assignment for caste '$caste'..."` |
| `.aether/aether-utils.sh` | Contains "malformed METADATA" (18-04 artifact spec) | VERIFIED | Line 3413: `"QUEEN.md has a malformed METADATA block..."` |
| `.aether/docs/queen-commands.md` | Queen command reference with queen-init (18-03) | VERIFIED | 3098 bytes; contains queen-init, queen-read, queen-promote with usage examples, argument tables, return formats |
| `bin/sync-to-runtime.sh` | Contains queen-commands.md in SYSTEM_FILES (18-03) | VERIFIED | Line 54: `"docs/queen-commands.md"` adjacent to error-codes.md |
| `bin/lib/update-transaction.js` | Contains queen-commands.md in SYSTEM_FILES (18-03) | VERIFIED | Line 210: `'docs/queen-commands.md'` adjacent to error-codes.md |
| `tests/bash/test-aether-utils.sh` | 8 new regression tests for Phase 18 requirements | VERIFIED | tests at lines 1020, 1040, 1059, 1074, 1091, 1108, 1124, 1141; all registered in main() at lines 1215-1228 |

---

## Key Link Verification

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| `.aether/aether-utils.sh` | `.aether/utils/atomic-write.sh (cleanup_temp_files)` | `_aether_exit_cleanup calls cleanup_temp_files` | WIRED | `cleanup_temp_files()` defined at atomic-write.sh:211; called in `_aether_exit_cleanup` at aether-utils.sh:101 |
| `.aether/aether-utils.sh` | `.aether/utils/file-lock.sh (cleanup_locks)` | `_aether_exit_cleanup calls cleanup_locks` | WIRED | `cleanup_locks()` defined at file-lock.sh:114; called in `_aether_exit_cleanup` at aether-utils.sh:100 |
| `.aether/aether-utils.sh (model-get)` | `.aether/aether-utils.sh (model-profile)` | `bash "$0" model-profile get/list subprocess call` | WIRED | Lines 2439, 2451: `bash "$0" model-profile get/list` with `set +e; exit_code=$?; set -e` wrapper; no `exec bash.*model-profile` (0 matches) |
| `.aether/docs/queen-commands.md` | `bin/sync-to-runtime.sh` | SYSTEM_FILES allowlist entry | WIRED | Line 54 in sync-to-runtime.sh: `"docs/queen-commands.md"` |
| `.aether/docs/queen-commands.md` | `bin/lib/update-transaction.js` | SYSTEM_FILES allowlist entry | WIRED | Line 210 in update-transaction.js: `'docs/queen-commands.md'` |
| `.aether/aether-utils.sh (queen-read)` | `jq validation` | `jq -e . before --argjson` | WIRED | Gate 1 at line 3411 validates metadata; Gate 2 at line 3451 validates assembled result; both precede or follow the jq --argjson call correctly |
| `.aether/aether-utils.sh (validate-state colony)` | `.aether/utils/atomic-write.sh (atomic_write)` | `atomic_write in _migrate_colony_state` | WIRED | `_migrate_colony_state` at line 727 calls `atomic_write "$state_file" "$updated"` |

---

## Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|------------|-------------|--------|----------|
| ARCH-02 | 18-04 | State files validated against schema version on load (GAP-001) | SATISFIED | `_migrate_colony_state` in validate-state colony; auto-migrates pre-3.0; W_MIGRATED notification; corrupt backup path |
| ARCH-03 | 18-01 | Spawn-tree entries cleaned up on session end (GAP-002) | SATISFIED | `_rotate_spawn_tree` at session-init rotates and archives; 5-archive cap; in-place truncation |
| ARCH-04 | 18-02 | Failed Task spawns have retry logic (GAP-003) | SATISFIED (RESOLVED) | User decision: fail-fast not retry. `spawn_failed` event logged to COLONY_STATE.json events array. Known-issues GAP-003 marked RESOLVED. |
| ARCH-05 | 18-03 | queen-* commands documented (GAP-004, GAP-006) | SATISFIED | queen-commands.md in .aether/docs/; in both sync allowlists; covers all 3 commands with usage, args, examples |
| ARCH-06 | 18-04 | queen-read validates JSON output before returning (GAP-005) | SATISFIED | Gate 1 (metadata) + Gate 2 (assembled result) both verify parseable JSON; E_JSON_INVALID with Try: on failure |
| ARCH-07 | 18-02 | model-get/model-list have exec error handling (ISSUE-002) | SATISFIED | exec pattern removed; subprocess pattern with exit_code capture; E_BASH_ERROR + "Try:" on failure |
| ARCH-08 | 18-03 | Help command lists all available commands including queen-* (ISSUE-003) | SATISFIED | sections key added with Queen Commands group; queen-init/read/promote with descriptions; backward-compat flat array |
| ARCH-09 | 18-01 | Feature detection doesn't race with error handler loading (ISSUE-007) | SATISFIED | Feature detection block moved after fallback json_err (line 68) to line 83; ARCH-09 comment in code confirms intent |
| ARCH-10 | 18-01 | Temp files cleaned up via exit trap (cleanup_temp_files wired to trap) | SATISFIED | `_aether_exit_cleanup` composes both cleanup_locks and cleanup_temp_files; trap set after file-lock.sh source to override individual trap |

All 9 requirement IDs (ARCH-02 through ARCH-10) are claimed by plans and have implementation evidence. No orphaned requirements found.

---

## Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| `.aether/aether-utils.sh` | 700-738 | `_migrate_colony_state` reads and potentially writes real `.aether/data/COLONY_STATE.json` during test execution | Warning | Causes 2 non-deterministic AVA test failures when concurrent tests backup/restore COLONY_STATE.json; race condition with state-loader test suite |

No placeholder implementations, TODO stubs, or empty handlers found in Phase 18 changes.

---

## Human Verification Required

### 1. Help Command Display

**Test:** Run `aether help` (or `bash .aether/aether-utils.sh help | jq .`) and review the output
**Expected:** A sections key with "Queen Commands" group showing queen-init, queen-read, queen-promote each with a one-liner description; all other section groups present (Core, Colony State, Model Routing, etc.)
**Why human:** Visual readability and grouping quality cannot be verified programmatically

---

## Gaps Summary

**One gap found with two components:**

The `_migrate_colony_state` function introduced in 18-04 interacts with the real `.aether/data/COLONY_STATE.json` during test execution. When `npm test` runs the full AVA suite in parallel, the state-loader test file briefly removes COLONY_STATE.json during its backup/restore cycle. If `_migrate_colony_state`'s `jq -e .` corruption check runs at exactly this moment, it finds no file and treats it as "corrupted", triggering the backup path and emitting E_JSON_INVALID. This produces 2 non-deterministic AVA test failures that only appear during the full concurrent test run — both tests pass when run in isolation.

**Root cause:** `_migrate_colony_state` does not guard against the file being absent (as distinct from corrupt) before calling `jq -e .`. A missing file is not a corrupt file. The fix is a one-line existence check: `[[ -f "$state_file" ]] || return 0` before the `jq -e` call.

**Pre-existing failures (not Phase 18):** 2 validate-state AVA tests (`validate-state with invalid target` and `validate-state without argument`) fail because they assert `error.error.includes('Usage:')` but Phase 17 changed `error.error` from a string to an object. These failures predate Phase 18 and were present before Phase 18 work began. 3 additional `namespace-isolation`, `sync-dir-hash`, and `user-modification-detection` test files have "No tests found" failures — also pre-existing.

**All 31 bash regression tests pass. All 9 ARCH requirements have substantive implementation. All key links are wired. The gap is scoped to a test isolation concern, not a functional concern.**

---

_Verified: 2026-02-19T16:30:00Z_
_Verifier: Claude (gsd-verifier)_
