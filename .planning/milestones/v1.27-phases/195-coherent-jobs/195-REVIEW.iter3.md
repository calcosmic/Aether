---
phase: 195-coherent-jobs
reviewed: 2026-08-27T14:20:00Z
depth: deep
iteration: 3
scope: closure check of NEW-01..NEW-05, CR-01, CR-02, WR-05 (195-REVIEW.iter2.md) — narrow, not a fresh phase review
diff_range: 7aa51688..HEAD (14 commits)
files_reviewed: 12
files_reviewed_list:
  - cmd/provenance.go
  - cmd/coherent_job_receipts.go
  - cmd/codex_build_worktree.go
  - cmd/codex_build_finalize.go
  - cmd/coherent_jobs.go
  - cmd/ceremony_team_checkin.go
  - cmd/provenance_receipt_carveout_test.go
  - cmd/coherent_job_no_change_evidence_test.go
  - cmd/coherent_job_receipt_root_evidence_test.go
  - cmd/codex_build_worktree_hallucinated_path_test.go
  - cmd/build_finalize_partial_replay_test.go
  - cmd/build_finalize_partial_screen_test.go
  - cmd/coherent_jobs_small_defects_test.go
  - scripts/version-sync.sh
  - npm/package.json
  - .planning/phases/195-coherent-jobs/deferred-items.md
findings:
  critical: 0
  warning: 1
  info: 3
  total: 4
status: issues_found
---

# Phase 195: Fix Re-Review (iteration 3)

**Reviewed:** 2026-08-27
**Depth:** deep — closure verification with in-place mutation testing
**Diff range:** `7aa51688..HEAD`
**Status:** issues_found (no blockers; one warning is a named residual, not a defect in the fix)

## Summary

All eight carried-forward findings are closed at the root, not at the probe. I
mutation-tested seven of the load-bearing locks by breaking each fix in place and
confirming the named test fails, then restoring every file byte-for-byte (verified
by md5 against a pre-mutation copy; `git status` shows no modified source files).

The two judgement calls the fixer flagged itself both check out. The renamed test
did assert the hole — its original body credited a task that named no file
anywhere, against an empty directory, on the receipt's own status word. And the
`completed_no_change` exemption really was reachable only by tasks declaring no
paths: with declared paths present, the stage-1 declared-file binding refused a
receipt that claimed nothing, so the honest "it was already true" case never
worked and only the evidence-free case did. The fix's shape follows from that
being true, and it is.

One residual is worth recording, because it is a real exchange rather than a pure
win, and the next reader should not have to rediscover it: closing the
no-declared-path hole widened what a `completed_no_change` receipt can be credited
for on tasks that *do* declare paths (WR-15 below). It is the second of the two
options the previous review itself prescribed, it is bounded by the phantom-build
guard, and it contributes nothing to criterion evidence — so it is a WARNING and a
deliberate design point, not a regression to fix before shipping.

## Verdict Table

| Finding | Verdict | Evidence |
|---|---|---|
| NEW-01 (CR-06) — carve-out excused SAFE-02 packet-wide | **CLOSED** | `cmd/provenance.go:86-96`: the relaxation is now folded inside `if completedCount == 0`, so SAFE-02 is armed exactly as `git show 85d78686^:cmd/provenance.go` had it — a `completed` worker with zero files is rejected whatever its neighbours carry. The worker test is a new caste-only `isReceiptCarveOutWorker`, so a reviewer holding a task ID no longer trips it. Both directions locked by `TestZeroFileCompletedWorkerIsRejectedEvenBesideARealPartial` (rejection *and* a genuine partial still accepted *and* a normal mixed packet still finalizable) and `TestReceiptCarveOutIgnoresAReviewerCarryingATaskID`. Mutation: restoring the old placement fails the first by name; restoring `isBuildImplementationWorker` fails the second by name. |
| NEW-02 (WR-11) — `completed_no_change` was self-asserted credit | **CLOSED** | The stage-2 exemption now requires the TASK to declare ≥1 path and every one of those paths to be a readable regular file in root (`rootBackedArtifactEvidence` → `snapshotBuildArtifact`, which Lstats and refuses symlinks). A path-less task is refused by name. Locked in both directions by `TestNoChangeReceiptIsNotCreditedOnSelfAssertionAlone` (including through `resolveCoherentJobDispatchReceipts` and `completedBuildTaskIDs`, the lane a real build uses) and `TestGenuineNoChangeReceiptIsStillCreditedWhenTheProjectBacksIt`. Mutation of the stage-2 check fails both; mutation of the stage-1 exemption fails the honest-direction tests. See WR-15 for the residual. |
| NEW-03 (WR-12) — wrapper lane showed the finished-build screen | **CLOSED** | `cmd/codex_build_finalize.go:191-195` mirrors the direct lane. `renderBuildFinalizeVisual` now has exactly two call sites and the partial one is intercepted first. `TestWrapperPartialFinalizeDoesNotShowTheFinishedBuildScreen` runs the real `build-finalize` command end to end and asserts both the absence of "advance honestly" / "External Task worker results recorded." and the presence of every unfinished task id and the `--task` recovery command. Mutation (`false &&`) fails it by name; the direct-lane test is unaffected, so the two locks are independent. |
| NEW-04 (WR-13) — one invented path cancelled a clean wave | **CLOSED** | `worktreeReceiptClaimedPaths` now applies three of admission's own lexical rules before a failed worker's receipted path can collide: covered-task scope, successful terminal status, and membership of the worker's own reported claims. I verified the filter is not *narrower* than admission: the `reported` set is built from `buildDispatchClaimOutputs(*outcome.result.WorkerResult)`, byte-identical to the `aggregateClaims` the same lane passes to `admitCoherentJobTaskReceipts`; the covered set is a deliberate superset (adds `TaskID` unconditionally); the status comparison matches `NormalizeTaskReceipts`' own lowercase/trim; and both sides normalize against the same `root`. So nothing admission would accept is now invisible to conflict detection. Locked by `TestOneHallucinatedReceiptPathDoesNotBlockACleanWorker` (asserts the file's *content* in the project after the wave, not just the conflict list) and `TestAnOutOfScopeReceiptPathDoesNotBlockACleanWorker`; both fail by name when the filters are removed, while the CR-05 lock `TestRejectedWaveNeverSyncsAFailedWorkersReceipt` still passes in both states — the fix did not hollow out the finding it came from. |
| NEW-05 (WR-14) — the "read-only" replay could write | **CLOSED** | The replay now calls `planPartialBuildRetry` (verified pure: no store, no attempt write, returns a plan struct) and only *looks up* the recovery record via `findExistingBuildAttemptRetry` (verified read-only). `TestPartialFinalizeReplayWritesNothing` deletes the recovery record to simulate the exact failed-write state, replays, and asserts the attempt count is unchanged and no child record exists. Mutation (restoring `reconcilePartialBuildRetry`) fails it by name. The doc comment now states the one condition under which the record id is omitted, satisfying the "documentation claim must be testable" rule. |
| CR-01 — zero-path receipt credited | **CLOSED** | The `completed` half was already proven in iteration 2; the `completed_no_change` sibling that kept it open is NEW-02, now closed. |
| CR-02 — packet-wide phantom-build relaxation | **CLOSED** | Same evidence as NEW-01. The guard's behaviour for a zero-file `completed` worker is now identical to pre-195 in every packet shape I could construct. |
| WR-05 — partial build shows the done screen | **CLOSED** | Both lanes (`aether build` and `aether build-finalize`) branch to `renderBuildPartialCreditVisual`, each with its own stdout-discriminating lock. |
| IN-01 / IN-02 / IN-03 | **CLOSED** | Fixed with named locks: `TestJobPlannerDoesNotMutateItsCallersProposal` (slice copied before trimming), `TestSentenceCaseKeepsWholeCharacters` (rune-decoding `sentenceCase`, tested with accented and CJK input), `TestTwoUnnamedWorkersStillConflictOverOneFile` (ownership key falls back to the worker name, and a worker still does not conflict with itself). |
| IN-04 / IN-05 / IN-11 | **CLOSED as deferrals** | `deferred-items.md` now records all five Info findings with what was fixed, what was not, and why — including the behaviour decision IN-04 depends on. This is what IN-11 asked for. |

## Warnings

### WR-15: closing the no-change hole widened what a no-change receipt can claim on tasks that DO declare files

**Severity:** WARNING (residual design point, not a defect in the fix)
**File:** `cmd/coherent_job_receipts.go:349-399`

Before this pass, a `completed_no_change` receipt claiming no path was refused
outright whenever its task declared any path, because the stage-1 declared-file
binding found no match. After this pass it is admitted, and credited whenever
every file the task declares exists in the checkout. Declared paths are derived
from the task's evidence artifacts and file-shaped hints — which, for most tasks
that modify existing code, already exist before the build starts. So the credited
surface moved: a hole reachable only by path-less tasks was closed, and a
narrower-but-more-common one ("the file it names is there, so I say it was
already correct") was opened.

This is not a mis-implementation — it is the second of the two fixes the previous
review prescribed, and the honest D6 case genuinely could not be expressed before.
Three things bound it, which is why it is not a blocker:

- The phantom-build guard is unaffected: a packet whose only proof is no-change
  receipts still has `receiptEvidenceCount == 0` (that function requires a named
  file) and is rejected by SAFE-01. I traced this; a build made entirely of
  no-change credit cannot finalize.
- `criterionClaimSets` reads a task claim's `files_created/modified/tests_written`,
  not its artifact evidence — and a no-change claim has none — so this credit
  cannot satisfy a phase criterion that asks for artifacts.
- The receipt must still clear scope, status, summary, and handoff validation.

**Fix (only if the owner wants it tighter):** additionally require the declared
file's snapshot to match the evidence recorded for the same path at plan time, or
apply the worker-level `noChangeEvidenceMissing` rule at receipt level as the
previous review's first option suggested. Either way it needs a decision about
what "already true" is allowed to mean, which is a behaviour call, not a wiring
fix — so if it is not taken now it belongs in `deferred-items.md` alongside IN-04.

## Info

- **IN-13:** The replay's `retry_attempt_id` / `retry_attempt_path` branch (the
  case where the recovery record *does* still exist) runs during
  `TestPartialFinalizeCanBeReplayedWithTheSamePacket` but nothing asserts its
  output, so a wrong id there would not fail a test. One extra assertion in that
  test would close it.
- **IN-14:** `worktreeOwnershipIdentity`'s new fallback key is
  `"worker:" + WorkerName`, so two dispatches with an empty task id *and* an empty
  worker name still compare equal. No runtime dispatch shape produces that pair
  (a dispatch with no task id also carries no declared paths, so the ownership
  loop never runs), which is why this is Info and not a warning.
- **IN-15:** The `npm/package.json` version bump and the `version-sync.sh` line
  that derives it are unrelated to Phase 195 and rode in on this fix pass. The
  change itself is right — derived rather than remembered, and already gated by
  `TestPackedNPMReleaseCandidateContract`; I verified the new perl expression
  rewrites the version line correctly on a scratch copy. Noted only because a
  release-tooling edit inside a trust-boundary fix round is the kind of thing a
  later bisect trips over.
- **IN-16:** IN-12 from iteration 2 (worktree refusal messages moved from the
  owner-facing ceremony stream to stderr) is still unaddressed and unrecorded.
  Cosmetic; still visible; no finding asked for the change in the first place.

## Verification performed

- `go vet ./cmd/...` — clean.
- Targeted run of all 17 locks named above — pass (2.4s).
- Phase-195 cluster (`-run 'CoherentJob|Receipt|Partial|Worktree|GroupedJob|CarveOut|Provenance'`) — pass (28s).
- `pkg/codex`, `pkg/colony` — pass.
- Seven mutations applied in place and reverted; each intended test failed **by
  name** and no other test failed spuriously. All four touched files restored and
  md5-verified against pre-mutation copies; working tree carries no source
  modifications.

## Overall Judgement

**Shippable.** Every finding carried into this pass is closed at its root, each
with a test that fails when the fix is removed, on the lanes a real build uses.
The one remaining warning is a named trade-off the previous review itself asked
for, bounded by two independent guards, and it does not put a false phase
completion within reach.

---

_Reviewed: 2026-08-27_
_Reviewer: Claude (gsd-code-reviewer), iteration 3_
_Depth: deep — seven fixes mutation-tested in place and restored byte-for-byte; no source files modified_
