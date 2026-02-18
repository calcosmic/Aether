# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-02-18)

**Core value:** Prevent context rot across Claude Code sessions with self-managing colony that learns and guides users
**Current focus:** Phase 15 — Distribution Chain

## Current Position

Phase: 15 of 18 (Distribution Chain)
Plan: 01 complete (15-01-SUMMARY.md), 02 complete (15-02-SUMMARY.md)
Status: In progress — 15-01 and 15-02 done, continuing phase 15
Last activity: 2026-02-18 — 15-01 complete: source dir fixed (HUB_SYSTEM_DIR), EXCLUDE_DIRS expanded, allowlists synced

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
| 15-18 (v1.2) | 2/TBD | In progress (15-01 complete, 15-02 complete) |

*Updated after each plan completion*

## Accumulated Context

### Decisions
- Full cleanup scope: fix ALL documented bugs, issues, and gaps (not just critical)
- All 5 v1.2 phases publish together in one `npm install -g .` cycle — no intermediate states
- Phase 14 is prerequisite gate: fallback json_err fix must land before Phase 17; ARCH-01 must land before any npm-install user testing
- Phase 15-01: source-dir fix (HUB_SYSTEM_DIR) applied in three methods; EXCLUDE_DIRS expanded with agents/commands/rules; caste-system.md added and planning.md removed from both SYSTEM_FILES allowlists (58 entries each)
- Phase 16 requires full lock audit before any code changes (local vs. global variable discrepancy)
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
Stopped at: Completed 15-01-PLAN.md (source dir fix + allowlist sync)
Resume file: .planning/phases/15-distribution-chain/15-01-SUMMARY.md
