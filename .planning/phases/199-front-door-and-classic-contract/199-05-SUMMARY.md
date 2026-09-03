---
phase: 199-front-door-and-classic-contract
plan: "05"
subsystem: lifecycle-transactions
tags: [go, transactions, recovery, rollback, receipts, containment]

requires:
  - phase: 199-front-door-and-classic-contract
    plan: "03"
    provides: Typed lifecycle/v1 transaction, receipt, state-effect, and recovery-provenance contracts
provides:
  - Typed multi-root lifecycle transaction API with explicit repository, data, hub, platform-home, and binary allowlists
  - Per-root staging and pre-images coordinated by immutable digest-backed intent and durable progress journals
  - Deterministic resume, reverse rollback, tamper conflict reporting, and exactly-once receipt replay
affects: [pause-resume, seal, entomb, territory-publication, maintenance, update]

tech-stack:
  added: []
  patterns: [immutable intent plus mutable progress, per-root same-filesystem staging, digest-proven recovery, typed root allowlist]

key-files:
  created:
    - cmd/lifecycle_transaction.go
    - cmd/lifecycle_transaction_test.go
    - cmd/lifecycle_transaction_fault_test.go
    - cmd/lifecycle_transaction_cross_root_test.go
  modified: []

key-decisions:
  - "Keep immutable coordinator intent separate from mutable progress so recovery never rewrites the evidence that authorized a mutation."
  - "Stage outputs and pre-images on each destination root, then replace targets only through temporary files in the target directory."
  - "Classify missing evidence as unknown and altered evidence as conflicting; only matching digests may finish or roll back a transaction."

patterns-established:
  - "Validate, stage, recheck, journal, commit, verify: no live target changes before durable intent exists."
  - "Exactly-once replay: a verified receipt is rebound to intent, progress, manifests, and current target digests before it is returned."
  - "Conflict-safe recovery: restore the proven committed prefix in reverse order while leaving unrelated conflicting bytes untouched."

requirements-completed: [CEC-04, CEC-08, LIFE-04, LIFE-05, LIFE-06]

duration: 27min
completed: 2026-09-03
---

# Phase 199 Plan 05: Crash-Safe Lifecycle Transactions Summary

**Digest-backed multi-root transactions now stage locally, commit only after durable intent, recover deterministically after interruption, and replay one verified receipt without duplicating effects.**

## Performance

- **Duration:** 27 minutes
- **Started:** 2026-09-03T19:14:45Z
- **Completed:** 2026-09-03T19:41:49Z
- **Tasks:** 2/2
- **Files modified:** 4 production/test files

## Accomplishments

- Added a closed typed allowlist for repository, lifecycle-data, selected stable/dev hub, Claude/OpenCode/Codex homes, and one exact binary destination, with canonical path, containment, duplicate, and symlink checks.
- Added per-root staged outputs and pre-images plus one coordinator journal containing immutable intent, root subtransaction manifests and digests, commit order, durable progress, the canonical lifecycle record, and one receipt.
- Added byte-verified commit and reverse rollback across roots, including a fake cross-filesystem failure seam that proves no rename crosses a destination directory.
- Added deterministic interruption recovery, safe committed-prefix rollback, missing-versus-conflicting provenance, and replay that revalidates all durable evidence and current target bytes before returning the existing receipt.
- Added fault coverage after validation, every staged output, intent persistence, every target/root commit, and global verification, plus stage, pre-image, intent, target, receipt, and symlink tamper cases.

## Task Commits

Each TDD task was committed as a RED contract followed by its GREEN implementation:

1. **Task 1: Implement contained lifecycle transactions** — `e53a2ab8` (RED), `32c449e2` (GREEN)
2. **Task 2: Prove rollback, interruption recovery, and exactly-once replay** — `577a5a7d` (RED), `8a079770` (GREEN)

Correctness and security hardening discovered during Task 2 verification: `95529939`, `6ae1e710`, `c1a18ed6`, `6984673d`, `f7cb4324`, `80af7b69`.

**Plan metadata:** committed with this summary.

## Files Created/Modified

- `cmd/lifecycle_transaction.go` — typed allowlist, declarations, staging, intent/progress journal, commit, verification, rollback, resume, receipts, and stable evidence reads.
- `cmd/lifecycle_transaction_test.go` — commit, path containment, root allowlist, changed-baseline, receipt, and per-root staging contracts.
- `cmd/lifecycle_transaction_fault_test.go` — complete named fault matrix, exactly-once replay, missing/altered evidence, and post-receipt tamper proof.
- `cmd/lifecycle_transaction_cross_root_test.go` — byte-exact rollback across roots and fake `EXDEV` same-directory replacement checks.

## Decisions Made

- Made `intent.json` immutable and self-digesting; mutable stage transitions live in a separately self-digesting `progress.json`, with a projected `LifecycleTransactionRecord` retained for consumers.
- Kept every root's staged bytes, pre-images, and manifest on that root's filesystem. The coordinator records their exact paths and digests rather than attempting a cross-filesystem rename.
- Treat the committed receipt as authoritative only while its checksum, coordinator intent, progress stage, root manifests, and every target digest still agree.
- Preserve all journals and local evidence after success, rollback, and conflict so `/ant-resume` and later lifecycle commands can explain what happened without guessing.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing Critical] Closed symlink substitution paths in transaction evidence**

- **Found during:** Task 2 tamper verification
- **Issue:** Matching bytes reached through a substituted symlink, or a symlinked staging/coordinator directory, could otherwise redirect reads or writes outside the owned evidence tree.
- **Fix:** Added stable regular-file reads and explicit ancestry checks for stage, pre-image, manifest, receipt, and coordinator paths; preserved substitutions as conflicting evidence.
- **Files modified:** `cmd/lifecycle_transaction.go`, `cmd/lifecycle_transaction_test.go`, `cmd/lifecycle_transaction_fault_test.go`
- **Verification:** Symlinked staged bytes and both local/coordinator directory escape cases are rejected without target or out-of-root mutation.
- **Committed in:** `95529939`, `80af7b69`

**2. [Rule 1 - Bug] Made recovery and replay honor the full transaction identity**

- **Found during:** Task 2 interruption/replay review
- **Issue:** An unsafe remaining target could leave an earlier committed root applied; a verified ID could be reused or rolled back; and receipt replay initially did not re-check current target bytes.
- **Fix:** Roll back the proven committed prefix in reverse order, seal IDs after intent/receipt, and bind receipt replay to current intent, progress, manifest, and target digests.
- **Files modified:** `cmd/lifecycle_transaction.go`, `cmd/lifecycle_transaction_fault_test.go`
- **Verification:** Committed-prefix conflict, post-receipt tamper, rollback-after-receipt, and repeat-commit cases all fail closed or return the same receipt without another effect.
- **Committed in:** `6ae1e710`, `c1a18ed6`, `6984673d`

**3. [Rule 1 - Bug] Deferred coordinator creation until validation succeeds**

- **Found during:** Task 2 zero-mutation audit
- **Issue:** Constructing a transaction initialized its coordinator store before an invalid target or changed baseline had been rejected.
- **Fix:** Lazily create coordinator storage only after staging and the second baseline check succeed.
- **Files modified:** `cmd/lifecycle_transaction.go`, `cmd/lifecycle_transaction_test.go`
- **Verification:** Invalid targets and changed baselines now leave no target or coordinator artifact.
- **Committed in:** `f7cb4324`

---

**Total deviations:** 3 auto-fixed (2 correctness bugs, 1 missing critical security control).
**Impact on plan:** All fixes strengthen the planned containment, rollback, and evidence requirements; no owner-visible feature scope was added.

## Issues Encountered

None - the planned transaction and fault suites pass after the inline correctness fixes above.

## Verification

- `go test ./cmd -run '^TestLifecycleTransaction(Commit|RejectsInvalidTarget|RejectsChangedBaseline|Receipt|RootAllowlist|PerRootStaging|CrossFilesystemRollback)$' -count=1` — 20 passed.
- `go test ./cmd -run '^TestLifecycleTransaction(FaultMatrix|MultiRootFaultMatrix|ReplayExactlyOnce|TamperConflict)$' -count=1` — 23 passed.
- `go test ./pkg/colony ./pkg/storage -count=1` — 309 passed.
- `go test -race ./cmd -run '^TestLifecycleTransaction' -count=1` — passed.
- `git diff --check` — clean.
- Stub scan found no TODO, FIXME, placeholder, coming-soon, unavailable, or unwired UI value in the four created files.
- `os.Rename`, `storage.SaveJSON`, and `os.Remove` call sites are confined to tested same-directory replacement, durable journal/staging writes, and verified commit/rollback removal paths.

## Threat Flags

| Flag | File | Description |
|------|------|-------------|
| threat_flag: multi-root-filesystem-mutation | `cmd/lifecycle_transaction.go` | New internal trust-boundary primitive can write or remove files under explicitly configured repository, hub, platform-home, data, and binary roots. Typed allowlists, symlink refusal, immutable digests, local pre-images, and fault/tamper tests guard the surface; downstream callers must still construct the allowlist from runtime-owned resolved roots. |

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Pause/resume, seal, entomb, survey publication, and maintenance plans can declare their multi-file effects through one shared transaction primitive instead of sequencing `SaveJSON` calls.
- Recovery callers receive confirmed, conflicting, or unknown provenance plus one exact coordinator journal and `aether resume` action.
- No Plan 199-05 implementation blocker remains.

## Self-Check: PASSED

- All four planned production/test files and this summary exist.
- Both RED commits, both GREEN commits, and all six inline hardening commits are present in Git history.
- The exact 20-case and 23-case plan suites, 309 related package tests, and race-instrumented lifecycle suite pass.

---
*Phase: 199-front-door-and-classic-contract*
*Completed: 2026-09-03*
