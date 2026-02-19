# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-02-18)

**Core value:** Prevent context rot across Claude Code sessions with self-managing colony that learns and guides users
**Current focus:** Phase 18 — Reliability & Architecture Gaps

## Current Position

Phase: 18 of 18 (Reliability & Architecture Gaps)
Plan: 4 of TBD — 18-01, 18-02, 18-03, 18-04 complete
Status: Phase 18 in progress (18-01 thru 18-04 done)
Last activity: 2026-02-19 — 18-04 complete: queen-read JSON validation gates added (ARCH-06), validate-state schema migration added (ARCH-02), known-issues.md updated for all Phase 18 fixes, 31 bash tests, 0 failures

Progress: ██████████████░░░░░░ 70% (v1.2 — Phases 14-18 partial, 18-02+ remaining)

## Performance Metrics

**Velocity (v1.0 + v1.1):**
- Total plans completed: 41
- Average duration: — (not tracked)
- Total execution time: — (not tracked)

**By Phase:**

| Phase | Plans | Status |
|-------|-------|--------|
| 1-9 (v1.0) | 27/27 | Complete |
| 10-13 (v1.1) | 13/13 | Complete |
| 14 (v1.2) | 1/1 | Complete |
| 14-16 (v1.2) | 7/7 | Complete (14-01, 15-01 thru 15-03, 16-01 thru 16-03) |
| 17 (v1.2) | 3/TBD | In progress (17-01, 17-02, and 17-03 done) |
| 18 (v1.2) | 4/TBD | In progress (18-01, 18-02, 18-03, 18-04 done) |

*Updated after each plan completion*

| Phase 17 P01 | 2 | 2 tasks | 2 files |
| Phase 17 P02 | 2 | 2 tasks | 2 files |
| Phase 17 P03 | 5 | 2 tasks | 4 files |
| Phase 18 P01 | 4 | 2 tasks | 2 files |
| Phase 18 P03 | 6 | 2 tasks | 5 files |
| Phase 18 P04 | 7 | 2 tasks | 3 files |

## Accumulated Context

### Decisions
- Full cleanup scope: fix ALL documented bugs, issues, and gaps (not just critical)
- All 5 v1.2 phases publish together in one `npm install -g .` cycle — no intermediate states
- Phase 14 is prerequisite gate: fallback json_err fix must land before Phase 17; ARCH-01 must land before any npm-install user testing
- Phase 15-01: source-dir fix (HUB_SYSTEM_DIR) applied in three methods; EXCLUDE_DIRS expanded with agents/commands/rules; caste-system.md added and planning.md removed from both SYSTEM_FILES allowlists (58 entries each)
- Phase 15-03: cleanupStaleAetherDirs() runs before syncFiles in execute(); reports cleanup with colony symbols; clean repos see "Distribution chain: checkmark clean"; 6 new unit tests added; all pre-3.0.0 npm versions removed from registry (unpublish succeeded)
- Phase 16 requires full lock audit before any code changes (local vs. global variable discrepancy)
- Phase 16-01: stale-lock prompt replaces silent auto-removal; [y/N] TTY prompt in interactive mode, JSON error in non-interactive; lock age checked before PID to handle PID reuse; SIGHUP added to trap
- Phase 16-01: uniform trap pattern (acquire -> trap EXIT -> work -> trap - EXIT -> release -> json_ok) across all 4 flag commands; local lock_acquired variables removed; release_lock takes no args
- Phase 16-01: atomic_write_from_file backup ordering fixed to match atomic_write (backup before validation, LOCK-03)
- Phase 16-02: _ctx_lock_held local variable is primary release gate; EXIT trap stays permanently active as safety net (not cleared on success path) because _cmd_context_update returns not exits
- Phase 16-02: force-unlock requires --yes in non-interactive mode; prompts [y/N] in interactive mode
- Phase 16-03: AETHER_ROOT isolation in atomic-write tests via fake git binary — cleanest approach for testing scripts that detect project root via git
- ERR-01 (14-01): fallback json_err emits `{code, message}` object — separate commits per fix strategy confirmed
- ARCH-01 (14-01): hub path first in template search loop; error message includes exact install command
- ERR-02 (17-01): error message format locked: friendly tone ("Couldn't find...") + mandatory "Try:" suggestion; E_DEPENDENCY_MISSING for missing utility scripts/binaries; E_RESOURCE_NOT_FOUND for missing runtime state; xmllint uses E_FEATURE_UNAVAILABLE (optional feature, not hard dep)
- [Phase 17]: Guard pattern chosen for chamber json_err: if ! type json_err preserves standalone fallback while yielding to error-handler.sh when loaded
- [Phase 17]: chamber-compare.sh sources error-handler.sh directly since it always runs standalone
- ERR-03/04 (17-03): grep -c exit code handling uses set +e/set -e to avoid double-output on zero matches; lock failure test uses nonexistent PID to trigger stale-lock path in non-interactive mode, then parses last JSON line for E_LOCK_FAILED
- ARCH-09 (18-01): feature detection block moved after fallback json_err (line 68 -> 81) so all fallback infrastructure available when feature detection runs; correctness over ordering speed
- ARCH-10 (18-01): composed _aether_exit_cleanup trap overrides file-lock.sh individual trap; startup orphan cleanup uses kill -0 (macOS-compatible PID liveness check)
- ARCH-03 (18-01): spawn-tree rotation uses archive-not-wipe strategy with timestamped files; in-place truncation (> file) preserves tail -f file handles; 5-archive cap
- ARCH-07 (18-02): model-get/model-list use subprocess (set +e; result=$(bash "$0" model-profile ...); exit_code=$?; set -e) not exec — allows exit code capture and friendly E_BASH_ERROR with Try: suggestion
- ARCH-04 (18-02): spawn-complete logs spawn_failed events to COLONY_STATE.json events array on "failed"/"error" status; independent tasks not blocked (fail-fast); local keyword not valid in case blocks — use prefixed var names
- ARCH-08 (18-03): flat commands array preserved exactly for backward compat; HELP_EOF delimiter used to avoid EOF collision; sections key added alongside existing structure
- ARCH-05 (18-03): queen-commands.md added to both allowlists adjacent to error-codes.md (same distribution pattern established in 17-03)
- [Phase 18]: queen-read: do not auto-reset QUEEN.md on malformed metadata — emit actionable E_JSON_INVALID with Try: suggestion; user decides
- [Phase 18]: validate-state migration additive only — never removes fields, adds missing v3.0 fields with empty defaults; W_MIGRATED to stderr

### Key Findings from Research
- update-transaction.js:909 reads from hub root instead of hub/system/ — affects all three methods (syncFiles, verifyIntegrity, detectPartialUpdate)
- Fallback json_err (lines 81-86 of aether-utils.sh) ignores code parameter — FIXED in 14-01 (commit 56039bf)
- Two parallel lock tracking systems (global LOCK_ACQUIRED from file-lock.sh, local lock_acquired in aether-utils.sh) can disagree
- .aether/agents/ and .aether/commands/ REMOVED (15-02, commit 0ebda62) — were dead duplicates not in any distribution chain
- caste-system.md missing from sync allowlist — not reaching target repos
- `flock` not available on macOS without Homebrew — use mkdir-based locking
- chamber-utils.sh and chamber-compare.sh define their own bare-string `json_err` that overwrites error-handler.sh's enhanced version — FIXED in 17-02

### Blockers / Concerns
- None — ERR-02, ERR-03, ERR-04 complete; Phase 17 core work done; any remaining 17 plans are optional

## Session Continuity

Last session: 2026-02-19
Stopped at: Completed 18-02-PLAN.md (exec, all prior plans 18-01 through 18-04 also complete)
Resume file: .planning/phases/18-reliability-architecture-gaps/18-02-SUMMARY.md
