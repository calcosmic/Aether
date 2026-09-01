---
phase: 195-coherent-jobs
reviewed: 2026-08-27T12:00:00Z
depth: deep
iteration: 2
scope: fix-verification of 195-REVIEW.md (CR-01..CR-05, WR-01..WR-10, IN-01..IN-05)
diff_range: 67b1873c..HEAD
files_reviewed: 22
files_reviewed_list:
  - cmd/provenance.go
  - cmd/coherent_job_receipts.go
  - cmd/coherent_job_retry.go
  - cmd/coherent_jobs.go
  - cmd/codex_build.go
  - cmd/codex_build_finalize.go
  - cmd/codex_build_worktree.go
  - cmd/codex_continue.go
  - cmd/codex_visuals.go
  - cmd/codex_workflow_cmds.go
  - cmd/build_attempt.go
  - cmd/build_print_brief.go
  - cmd/proof_cmd.go
  - pkg/codex/task_receipt.go
  - .aether/schemas/completion-packet.schema.json
  - CLAUDE.md
  - cmd/build_attempt_test.go
  - cmd/claudemd_coherent_jobs_test.go
  - cmd/provenance_receipt_carveout_test.go
  - cmd/coherent_job_receipt_root_evidence_test.go
  - cmd/codex_build_worktree_rejected_wave_test.go
  - cmd/codex_build_native_partial_test.go
findings:
  critical: 1
  warning: 4
  info: 7
  total: 12
status: issues_found
---

# Phase 195: Fix Re-Review (iteration 2)

**Reviewed:** 2026-08-27
**Depth:** deep (fix verification, not a fresh review)
**Diff range:** `67b1873c..HEAD` (32 commits)
**Status:** issues_found

## Summary

The fix round is substantially better than the code it repaired. Thirteen of the
twenty findings are genuinely closed with locks that fail by name — I mutation-tested
five of them (CR-01, CR-05, WR-03, WR-04, WR-10) by breaking the fix in place and
confirming the named test fails, then restored the files byte-for-byte. `go vet` is
clean, `pkg/codex` passes, and an 82-second targeted subset covering every migrated
call site passes.

Two findings are **not** closed, both on the trust boundary:

- **CR-02 is the serious one.** The fix narrowed *who* may trip the carve-out but
  kept the packet-wide `return nil`, and placed it **before** the SAFE-02 error
  rather than folding it into the SAFE-01 error as the original review prescribed.
  Probe-confirmed: a builder reporting `completed` with zero files still gets full
  credit for every task it covers, and the phase still reaches fully-credited, as
  long as any other failed implementation worker in the same packet carries one
  file-naming receipt. Pre-195 that exact packet was rejected as "the build produced
  no changes". The prescribed fix would have kept rejecting it without breaking D-08.
- **CR-01's exemption is a live sibling path.** Probe-confirmed: a worker that spells
  its receipt status `completed_no_change` instead of `completed` still gets credit
  for a task that declares no paths, against a root that does not exist. Nothing
  backs `commands_run` — it is a string the worker types. The original review
  proposed this exemption itself, so the fixer is not at fault, but the root cause
  ("a task with no declared paths can be credited on self-assertion alone") is not
  closed; it is gated behind one word.

Two more findings are partially closed: WR-05 fixed the direct lane's screen and
left the wrapper lane's (`aether build-finalize`) still saying "Run `aether continue`"
after a partial; and the CR-05 fix, while correct, introduced a new
denial-of-progress path where one hallucinated receipt path from a failed worker
rejects an entire otherwise-clean wave.

All five Info findings from the original review are untouched and undocumented — no
deferral record exists for them.

**The two inverted test assertions are legitimate.** I verified independently:
`parent.Status != buildAttemptFailed` had to become `buildAttemptPartial` because
that is precisely WR-04's prescribed fix, and `latest.ID != retryAttemptID` had to
become `parentAttemptID` because that is precisely WR-03's. Both new assertions are
load-bearing — under mutation (`makeLatest: true`) `TestGroupedJobPartialRetryIsAppendOnlyDirect`
fails by name. The only thing lost is that the old status assertion doubled as an
"append-only, parent untouched" guard; the test still checks `parent.ParentAttemptID == ""`
and the child's separate record, so the coverage loss is small.

## Verdict Table

| Finding | Verdict | Evidence |
|---|---|---|
| CR-01 | **PARTIALLY CLOSED** | `completed` zero-path credit closed and mutation-proven; `completed_no_change` sibling still credits a no-declared-path task against a nonexistent root (probe). See NEW-02. |
| CR-02 | **NOT CLOSED** | `receiptEvidenceCount > 0 → return nil` still short-circuits the whole packet, and now sits *ahead* of the SAFE-02 error. Probe: 3 tasks credited from a zero-file `completed` builder, `fullyCredited=true`. See NEW-01. |
| CR-03 | **CLOSED** | `json:"-"` on both fields, both removed from the packet schema, defensively cleared in `mergeExternalBuildResults`. `TestWrapperManifestCannotHandItselfTaskCredit` asserts the decode boundary, the merge boundary, and that the newly-emitted `completed_task_ids` in the result map does not round-trip back in. Verified `codexBuildManifest.Dispatches` is the only inbound path. |
| CR-04 | **CLOSED** | `buildUnfinishedRetryRedispatchCommand` emits repeatable `--task`; `--task` is a `StringArray` on `buildCmd` read once at `codex_workflow_cmds.go:163` and passed to **both** `runCodexBuildPlanOnlyWithOptions` and `runCodexBuildWithOptions`. Lock feeds the surfaced command's own argv through the real planner; counter-fixture `TestForceOnlyRedispatchWouldRedoCreditedWork` proves the bare `--force` really would redo credited work. |
| CR-05 | **CLOSED** (with a new side effect) | Partial-receipt sync gated on `len(conflicts) == 0`; failed workers' claimed paths now feed `touchedBy`. No normalization mismatch: detection and admission both use `lexicallyNormalizeReceiptPath(root, …)`, and `outcome.touched` is repo-relative from `collectWorktreeTouchedPaths`. Mutation (`if false`) makes `TestRejectedWaveNeverSyncsAFailedWorkersReceipt` fail with the contested file's real content. See NEW-04. |
| WR-01 | **CLOSED** | `reportCoherentJobReceiptRefusals` called on all three lanes (`resolveCoherentJobDispatchReceipts`, `resolveWorktreeExternalDispatchReceipts`, `resolveWorktreePartialReceipts`), printing all twelve rules, not two. |
| WR-02 | **CLOSED** | `buildAttemptPartial` added to the accepted-status switch behind a digest-proved `isCommittedPartialAttemptReplay`; idempotent partial branch added. `TestPartialFinalizeCanBeReplayedWithTheSamePacket` asserts the second call succeeds, credits nothing new, and returns the same recovery command. A *different* packet is still refused by name. |
| WR-03 | **CLOSED** | `beginBuildAttemptRecord(..., makeLatest=false)` for children. I checked every reader of `loadLatestBuildAttempt` (worker-run, waiver window, plan revision, verify-out-of-band, check-fix, build guard) — all want the *in-flight* attempt, none wants the recovery record, which stays discoverable via `findExistingBuildAttemptRetry`. Mutation to `true` fails both plan-only locks by name. |
| WR-04 | **CLOSED** | `finishAttempt(buildAttemptPartial, …)` on the direct lane; `transitionBuildAttempt` has no terminal guard so `failed → partial` lands. Locked end-to-end by `TestNativePartialCreditIsJournalledAsPartial` (runs a real build with a partial-failing invoker). |
| WR-05 | **PARTIALLY CLOSED** | Direct lane gets `renderBuildPartialCreditVisual` and a discriminating stdout lock. The **wrapper lane is untouched**: `buildFinalizeCmd` (`codex_build_finalize.go:185`) has no partial branch and renders `renderBuildFinalizeVisual`, which unconditionally prints "Run `aether continue` to verify the external Task work and advance honestly" with no recovery command. See NEW-03. |
| WR-06 | **CLOSED** | Error propagated; all five production call sites handled (`build_print_brief.go` returns, `proof_cmd.go` returns `phase_plan_unavailable`, two visuals render `renderUnplannablePhase`, `codex_workflow_cmds.go` calls `outputError`). The one stderr-warning site (`applyCodexBuildState`) is provably unreachable-on-error: the same plan already succeeded in the caller and cycle preflight is proposal-independent. The four test-only wrappers do not hide anything — I checked all 33 call sites and none asserts an empty dispatch list; the three that assert on length assert **non**-empty. |
| WR-07 | **CLOSED** | Chunks chained via `job.DependsOn = []string{previousChunkName}` (post-`reserveCoherentJobName`, reset per component, acyclic by construction) and `populateCoherentJobDAG` now preserves pre-seeded edges. `TestSplitJobsNeverShareAFileInTheSameWave` passes. |
| WR-08 | **CLOSED** | `codex.NormalizeTaskReceipts` exported and called at the top of `admitCoherentJobTaskReceipts`. Verified it is genuinely lexical (`normalizeClaimPaths` uses `filepath.Rel`/`Clean` only, no stat) so it stays safe to run before a worktree sync, and idempotent so the native lane's second pass is harmless. `TestReceiptAliasesAreAcceptedOnEveryLane` passes. |
| WR-09 | **CLOSED, and better than asked** | Three generic anchors replaced with Phase-195-unique phrases, plus two meta-tests: `TestCoherentJobDocAnchorsAreDiscriminating` (every anchor must occur exactly once, inside the 195 section) and `TestCoherentJobDocAnchorsFailWithTheSectionRemoved` (every anchor must vanish when the section is cut). This is the right shape for this repo's Definition of Done. |
| WR-10 | **CLOSED** | Order is now plan (pure) → commit credit → journal `partial` → create recovery record. `TestNativePartialCreditCommitsBeforeCreatingTheRecoveryRecord` pauses the colony mid-invoke so the commit is genuinely refused, then asserts no record carries a `ParentAttemptID`. Failure window between commit and record creation is benign: the parent is already terminal `partial`, so the next plan-only proceeds, and the external replay path re-creates the record. |
| IN-01 | **NOT CLOSED** | `cmd/coherent_jobs.go:235-237` still writes `proposal.TaskIDs[idx]` through the caller's backing array. |
| IN-02 | **NOT CLOSED** | `cmd/ceremony_team_checkin.go:551` still `strings.ToUpper(whyNoApproval[:1])`. |
| IN-03 | **NOT CLOSED** | `cmd/codex_build_worktree.go:120` still `key: strings.TrimSpace(dispatch.TaskID)`. |
| IN-04 | **NOT CLOSED** | `findExistingBuildAttemptRetry` still returns any child for the parent without comparing `SelectedTasks`; `plan.Jobs[0].Name` still the only recorded `ParentJobName`. |
| IN-05 | **NOT CLOSED** | `allSelectedBuildTasksCredited` still recomputes `buildFullBuildTaskIDSet` + `completedBuildTaskIDs` independently, as do its two siblings. |

No deferral record exists for IN-01..IN-05 in `deferred-items.md` or anywhere else.

## Critical Issues

### CR-06 (NEW-01): The phantom-build guard still returns nil for the whole packet, and now does so ahead of SAFE-02

**Severity:** BLOCKER
**File:** `cmd/provenance.go:74-76`, reached from `cmd/codex_build_finalize.go:573`

**Issue:** The CR-02 fix replaced a bare `return nil` inside the loop with a
per-worker counter, but then added

```go
if receiptEvidenceCount > 0 {
    return nil
}
if completedCount == 0 { … SAFE-01 error … }
return … SAFE-02 error …
```

so the relaxation still terminates the guard for the entire packet, and it now
pre-empts the SAFE-02 error rather than only the SAFE-01 one. The original review
prescribed `if completedCount == 0 && receiptEvidenceCount == 0 { return err }`,
which leaves SAFE-02 armed. The implemented form is strictly weaker.

Probe (removed after running):

```
results: Mason-1 builder "completed", covers 1.1/1.2/1.3, ZERO files
         Mason-2 builder "failed",    covers 1.4, one root-real receipt naming cmd/real.go
→ validateBuildProvenanceForManifest = <nil>
→ completedBuildTaskIDs = map[1.1 1.2 1.3 1.4]
→ allSelectedBuildTasksCredited = true      // the phase is marked BUILT
```

Three tasks credited with no file evidence of any kind. Verified against
`git show 85d78686^:cmd/provenance.go`: pre-195 this exact packet returned
`"1 worker(s) completed but none reported file changes … the build produced no
changes"`. This is a live regression of SAFE-02 that the fix round did not close.

A second, smaller hole in the same branch: `isBuildImplementationWorker` returns
true whenever `result.TaskID != ""`, so a *reviewer* worker carrying a task ID
counts as an implementation worker and can trip the carve-out.
`TestReceiptCarveOutIsScopedToImplementationWorkers` only proves the negative for a
watcher with an **empty** `TaskID`, so it does not lock the claim its own name makes.

**Fix:**

```go
// cmd/provenance.go
if completedCount == 0 {
    if receiptEvidenceCount > 0 {
        return nil
    }
    return fmt.Errorf("build provenance: no workers completed successfully -- …")
}
return fmt.Errorf("build provenance: %d worker(s) completed but none reported file changes …", completedCount)
```

and tighten the carve-out's worker test to `strings.TrimSpace(r.Caste) == "" || r.Caste == "builder"`
(i.e. not `isBuildImplementationWorker`), then extend
`TestReceiptCarveOutIsScopedToImplementationWorkers` with a watcher that *does*
carry a `TaskID`. Add a lock reproducing the probe above and asserting
`allSelectedBuildTasksCredited == false`.

## Warnings

### WR-11 (NEW-02): `completed_no_change` is unverified self-declared credit, and it is the same hole CR-01 closed

**Severity:** WARNING
**File:** `cmd/coherent_job_receipts.go:286-309`

**Issue:** The CR-01 fix refuses a zero-path `completed` receipt but exempts
`completed_no_change`. Everything that status has to clear is authored by the worker:
a task ID in scope, a non-empty summary, `verification_status: "pass"`, and one
non-empty `commands_run` string. Nothing executes, records, or corroborates
`commands_run` — it is a sentence. Probe (removed after running):

```
task 1.1 declares no artifacts and no file-shaped hints
receipt: {"task_id":"1.1","status":"completed_no_change","summary":"already true",
          "handoff":{"verification_status":"pass","commands_run":["go test ./... (i promise)"]}}
root = <tempdir>/nope   (does not exist)
→ resolveCoherentJobDispatchReceipts → CompletedTaskIDs [1.1], completedBuildTaskIDs map[1.1]
```

That is byte-for-byte CR-01's outcome with one word changed in the receipt. The
review's own prescribed fix contained this exemption, so this is not a fixer error —
but the root cause the finding named ("a task with no declared paths credited on
self-assertion alone") is not closed, and `TestNoChangeReceiptStillCreditsWithoutFiles`
now *asserts* the behaviour, which makes it harder to remove later.

Mitigations that keep this out of BLOCKER: the exploit needs a task with no declared
paths **and** a non-success dispatch status, and the phantom-build guard still
requires some worker in the packet to name a real file. Combined with CR-06 above,
though, a single packet can credit an entire phase's no-declared-path tasks with
nothing but prose.

**Fix (pick one):**
- Require a `completed_no_change` receipt's `commands_run` to match the dispatch's
  own recorded command evidence (`noChangeEvidenceMissing` already exists for the
  worker-level equivalent — reuse it at receipt level), or
- Require the task to declare at least one path before it may be credited at all, and
  make "task declares nothing" a planner-time refusal rather than a runtime free pass.

Either way, add a lock that fails if a `completed_no_change` receipt is credited
against a root where the task's own declared/aggregate paths cannot be resolved.

### WR-12 (NEW-03): The wrapper lane still shows the finished-build screen after a partial

**Severity:** WARNING
**File:** `cmd/codex_build_finalize.go:185`, `cmd/codex_visuals.go:1891-1911`

**Issue:** WR-05's fix was applied only to `buildCmd`'s direct-execution branch
(`cmd/codex_workflow_cmds.go:280`). `buildFinalizeCmd` — the second half of the
interactive wrapper's only build path (`aether build --plan-only` → wrapper spawns →
`aether build-finalize`) — sets `result["recovery_job"] = true` at
`codex_build_finalize.go:832` and then unconditionally renders
`renderBuildFinalizeVisual`, which prints
`Run `aether continue` to verify the external Task work and advance honestly` and
never mentions `recovery_command`, `unfinished_task_ids`, or that the phase is half
built. The identical hazard, on the lane CLAUDE.md documents as primary.

**Fix:** Mirror the direct-lane branch:

```go
// cmd/codex_build_finalize.go, buildFinalizeCmd
if partial, _ := result["recovery_job"].(bool); partial {
    unfinished, _ := result["unfinished_task_ids"].([]string)
    recovery, _ := result["recovery_command"].(string)
    outputWorkflow(result, renderBuildPartialCreditVisual(state, phase, unfinished, recovery))
    return nil
}
```

and extend `TestPartialBuildDoesNotShowTheOrdinaryBuildDoneScreen` with a
`build-finalize` case (or add a sibling test) so both lanes are locked.

### WR-13 (NEW-04): One hallucinated receipt path from a failed worker now rejects a whole clean wave

**Severity:** WARNING
**File:** `cmd/codex_build_worktree.go:720-742` (`worktreeReceiptClaimedPaths`)

**Issue:** The CR-05 fix feeds a non-completed worker's receipt paths into conflict
detection **before any admission at all**: no covered-task scope check, no status
check, no aggregate-claims check, no root check. Only `lexicallyNormalizeReceiptPath`
runs. A failed worker whose receipt names a file it never touched — an entirely
ordinary LLM output error — now collides with the declared owner and rejects the wave,
which blocks *every* completed worker in it (`dr.Status = "blocked"`, worktrees
orphaned). Before the fix that receipt was simply refused; now it costs the wave.

The doc comment overstates the set as "the paths its own task receipts CLAIM —
because those, and only those, are what `resolveWorktreePartialReceipts` would copy
back". That is not accurate: admission would refuse most of them.

The conservative direction is defensible and nothing is destroyed, so this is a
WARNING, not a BLOCKER — but it is a new failure mode introduced by the fix, and it
is not covered by any test.

**Fix:** Filter to receipts that are at least in scope and terminal-successful before
adding their paths (cheap, still lexical):

```go
for _, receipt := range outcome.result.WorkerResult.TaskReceipts {
    if _, inScope := coveredSet[strings.TrimSpace(receipt.TaskID)]; !inScope {
        continue
    }
    status := strings.ToLower(strings.TrimSpace(receipt.Status))
    if status != codex.TaskReceiptStatusCompleted && status != codex.TaskReceiptStatusCompletedNoChange {
        continue
    }
    …
}
```

and add a test proving a completed worker is not blocked by a failed worker's
out-of-scope receipt.

### WR-14 (NEW-05): The "idempotent" partial replay can write a new attempt record

**Severity:** WARNING
**File:** `cmd/codex_build_finalize.go:1163-1213`

**Issue:** `idempotentExternalPartialFinalizeResult`'s doc comment says it "mutates
nothing: no colony state write, no attempt transition, no new credit", and describes
`reconcilePartialBuildRetry` as "itself idempotent — it finds the recovery record it
already created for this parent rather than creating a second one". That is only true
when the record exists. If the first finalize's record creation failed (it warns to
stderr and continues), the replay reaches `beginChildBuildAttempt` and writes a new
attempt file. CLAUDE.md is explicit that a documentation claim about runtime behaviour
must be testable or removed, and that inspection paths must not mutate.

**Fix:** Either restrict the replay to `findExistingBuildAttemptRetry` (re-derive the
command from `planPartialBuildRetry` alone, which writes nothing), or reword the
comment to state the one condition under which it writes, and add a test for it.

## Info

- **IN-06:** IN-01 unfixed — `cmd/coherent_jobs.go:235-237` mutates the caller's
  `TaskIDs` backing array, contradicting `planCoherentJobs`' "intentionally pure"
  contract at line 87.
- **IN-07:** IN-02 unfixed — `cmd/ceremony_team_checkin.go:551` byte-slices a string
  for capitalization.
- **IN-08:** IN-03 unfixed — `worktreeOwnershipIdentity` still keys on `TaskID`, so
  two empty-`TaskID` dispatches compare equal.
- **IN-09:** IN-04 unfixed — retry idempotency does not compare `SelectedTasks`;
  `ParentJobName` records only the first of N retry jobs.
- **IN-10:** IN-05 unfixed — three finalize helpers each recompute the same two sets.
- **IN-11:** No deferral record exists for IN-01..IN-05. This repo's own history says
  undocumented deferrals become invisible debt; one line each in
  `deferred-items.md` would close it.
- **IN-12:** WR-01's fix moved `sync_failed` / `root_evidence_missing` messages on the
  worktree lane from `emitVisualProgress` (owner-facing ceremony on stdout) to
  `visualFprintf(stderr, …)`. Consistent and still visible, but it is a channel change
  no finding asked for.

## Overall Judgement

**Not shippable as-is.** One BLOCKER (CR-06/NEW-01) is a live regression of the
phantom-build guard that a build packet can reach today, and it is a straight
mis-application of the fix the first review prescribed — the correct version is three
lines and does not break D-08. WR-11 (self-declared `completed_no_change`) and WR-12
(wrapper lane still shows the finished-build screen) should land in the same pass;
both are the "closed one path, left the sibling open" pattern the fix round was
supposed to be checking for.

Everything else in the round is genuinely good work: the doc-anchor meta-tests, the
CR-04 counter-fixture, and the WR-10 pause-mid-invoke test are the strongest locks in
this phase and are exactly the shape CLAUDE.md's Definition of Done asks for.

---

_Reviewed: 2026-08-27_
_Reviewer: Claude (gsd-code-reviewer), iteration 2_
_Depth: deep — five fixes mutation-tested in place and restored byte-for-byte; no source files modified_
