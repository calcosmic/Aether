# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-02-18)

**Core value:** Prevent context rot across Claude Code sessions with self-managing colony that learns and guides users
**Current focus:** Phase 16 — Lock Lifecycle Hardening

## Current Position

Phase: 16 of 18 (Lock Lifecycle Hardening)
Plan: 01 complete (16-01-SUMMARY.md)
Status: In progress — 16-01 done, continuing phase 16
Last activity: 2026-02-18 — 16-01 complete: stale-lock user prompt, uniform trap pattern in all 4 flag commands, atomic_write_from_file backup ordering fixed

Progress: ██░░░░░░░░░░░░░░░░░░ 22% (v1.2 — Phase 15 in progress)

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
| 15-18 (v1.2) | 4/TBD | In progress (15-01, 15-02, 15-03, 16-01 complete) |

*Updated after each plan completion*

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
- ERR-01 (14-01): fallback json_err emits `{code, message}` object — separate commits per fix strategy confirmed
- ARCH-01 (14-01): hub path first in template search loop; error message includes exact install command

### Key Findings from Research
- update-transaction.js:909 reads from hub root instead of hub/system/ — affects all three methods (syncFiles, verifyIntegrity, detectPartialUpdate)
- Fallback json_err (lines 81-86 of aether-utils.sh) ignores code parameter — FIXED in 14-01 (commit 56039bf)
- Two parallel lock tracking systems (global LOCK_ACQUIRED from file-lock.sh, local lock_acquired in aether-utils.sh) can disagree
- .aether/agents/ and .aether/commands/ REMOVED (15-02, commit 0ebda62) — were dead duplicates not in any distribution chain
- caste-system.md missing from sync allowlist — not reaching target repos
- `flock` not available on macOS without Homebrew — use mkdir-based locking
- chamber-utils.sh and chamber-compare.sh define their own bare-string `json_err` that overwrites error-handler.sh's enhanced version — pre-existing bug, deferred to Phase 17

### Blockers / Concerns
- Phase 16 (lock audit): full surface of acquire/release pairs not yet enumerated — budget discovery time
- Phase 15 (EXCLUDE_DIRS): verify current hub directory structure at implementation time, not just research snapshot

## Session Continuity

Last session: 2026-02-18
Stopped at: Completed 16-01-PLAN.md (lock lifecycle hardening — stale-lock prompt, uniform trap pattern, LOCK-03 fix)
Resume file: .planning/phases/16-lock-lifecycle-hardening/16-01-SUMMARY.md
