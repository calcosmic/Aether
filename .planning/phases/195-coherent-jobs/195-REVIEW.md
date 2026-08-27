---
phase: 195-coherent-jobs
reviewed: 2026-08-27T00:00:00Z
depth: standard
files_reviewed: 24
files_reviewed_list:
  - cmd/coherent_jobs.go
  - cmd/coherent_job_receipts.go
  - cmd/coherent_job_retry.go
  - cmd/codex_build.go
  - cmd/codex_build_finalize.go
  - cmd/codex_build_worktree.go
  - cmd/ceremony_team_checkin.go
  - cmd/codex_workflow_cmds.go
  - cmd/codex_continue.go
  - cmd/build_attempt.go
  - cmd/build_worker_run.go
  - cmd/internal_worker_adapter.go
  - cmd/provenance.go
  - cmd/command_guide.go
  - cmd/codex_visuals.go
  - cmd/criterion_owner_confirmation.go
  - pkg/codex/dispatch.go
  - pkg/codex/task_receipt.go
  - pkg/codex/worker.go
  - .aether/schemas/completion-packet.schema.json
  - .aether/commands/build.yaml
  - .claude/commands/ant/build.md
  - cmd/claudemd_coherent_jobs_test.go
  - CLAUDE.md
findings:
  critical: 5
  warning: 10
  info: 5
  total: 20
status: issues_found
---

# Phase 195: Code Review Report

**Reviewed:** 2026-08-27
**Depth:** standard
**Files Reviewed:** 24 production/contract files plus the new test surface
**Status:** issues_found

## Summary

The planner (`planCoherentJobs`) is genuinely careful — cycle preflight, per-proposal
refusal by name, union-safety re-checks before every contraction, deterministic
ordering. Path normalization on the receipt boundary is also solid: `normalizeCriterionArtifactPath`
plus `snapshotBuildArtifact`'s `Lstat`/`EvalSymlinks`/`Rel` chain closes traversal,
symlink and absolute-path escapes. `go vet` is clean and the full `cmd` +
`pkg/codex` suites pass (536s).

The failures are all on the trust boundary and the recovery story, not in the
grouping maths:

- The stage-2 "root-backed" finalizer credits a task **with no root check at all**
  when the receipt claims zero paths and the task declares none. Proven by probe
  against a nonexistent root.
- The new `hasGenuineTaskReceiptEvidence` carve-out returns `nil` for the **entire**
  phantom-build guard on the strength of one unverified receipt object.
- `completed_task_ids` and `task_claims` became declared, schema-accepted fields on
  the dispatch object this phase, and the legacy (unbound) `build-finalize` path
  reads them straight from a wrapper-authored manifest into completion credit.
- D-10's unfinished-only retry job is planned, journaled, and **never executed**.
  The command actually surfaced to the owner is `aether build <phase> --force`,
  which re-dispatches every task in the phase — the exact "redo proven work" the
  phase, the wrapper triplet, the command guide and CLAUDE.md all promise never
  happens. That is this repo's named orphan pattern, shipped again.
- In worktree mode, a failed worker's receipt-scoped sync runs *inside a wave the
  runtime just rejected for conflicts*, and outside conflict detection entirely,
  so it can overwrite a same-wave successful worker's already-synced file.

Findings CR-01, CR-02 and CR-03 were each reproduced with a throwaway probe test
in `package cmd` (removed afterwards; no source files were modified).

## Critical Issues

### CR-01: Stage 2 credits a task with zero root evidence when the receipt claims no paths

**File:** `cmd/coherent_job_receipts.go:220` and `cmd/coherent_job_receipts.go:271`
**Issue:** `admitCoherentJobTaskReceipts` skips the task-binding check entirely when
`declaredPathsForTask(task)` is empty (line 220 — common: a task with no
`evidence_requirements.artifacts` and no file-shaped `hints` has none).
`finalizeCoherentJobTaskReceiptEvidence` then guards its whole root-evidence block
behind `if len(paths) > 0` (line 271), so a candidate with no claimed paths falls
through to `claims = append(...)` / `completedTaskIDs = append(...)` untouched.

Concrete failure: a `failed` dispatch covering `1.1`, where `1.1` declares no paths,
submits
`{"task_id":"1.1","status":"completed","summary":"done","files_created":[],"files_modified":[],"tests_written":[],"handoff":{"verification_status":"pass","commands_run":["echo ok"]}}`.
Probe output against `root = "/nonexistent-root"`:

```
violations=[] candidates=[{TaskID:1.1 Claim:{TaskID:1.1 FilesCreated:[] ...}}]
finalViolations=[] completed=[1.1]
```

The task is marked complete by `completedBuildTaskIDs` → `reconcileCompletedBuildTasks`
on the strength of a self-asserted sentence. The function's own doc comment ("the
only function ... allowed to populate a runtime-owned completed-task-ID set ... A
candidate whose claimed paths are not readable regular files in root right now is
dropped without credit") is not what the code does.

**Fix:** Require positive root-backed evidence for every credited candidate, and make
the no-file case an explicit, narrower rule rather than a fall-through:

```go
paths := claimedPaths(claim)
if len(paths) == 0 {
    // Only completed_no_change may credit with no files, and only when the
    // receipt's own commands_run are recorded as the evidence.
    if candidate.Status != codex.TaskReceiptStatusCompletedNoChange {
        violations = append(violations, contractViolation{
            Worker: worker, Field: "task_receipts.files_modified", Value: candidate.TaskID,
            Rule: violationRuleTaskReceiptUnevidenced,
            Message: fmt.Sprintf("task %s's receipt claims completion but names no file in the repository; not credited", candidate.TaskID),
        })
        continue
    }
}
```

and carry `Status` onto `coherentJobReceiptCandidate` so stage 2 can see it. Separately,
tighten line 220: a task with no declared paths should still require its claimed paths
to exist in root (it already would, once the zero-path fall-through is closed).

### CR-02: One unverified task receipt disarms the whole phantom-build guard

**File:** `cmd/provenance.go:24-37` (carve-out) and `cmd/provenance.go:59-91` (`hasGenuineTaskReceiptEvidence`)
**Issue:** The carve-out sits inside the `!isSuccessfulExternalBuildStatus(status) || !isBuildImplementationWorker(r)`
branch and does `return nil` — terminating `validateBuildProvenance` for the *entire
packet*, not just excusing that one worker. `hasGenuineTaskReceiptEvidence` checks
only the receipt's own self-reported shape: a non-empty `task_id` (never resolved
against the phase), `status`, `summary`, `handoff.verification_status == "pass"`, and
one `commands_run` string. Nothing is checked against the repository or the manifest.

Concrete failure (probe-confirmed): a packet where the builder reports
`status:"completed"` with zero `files_created`/`files_modified`/`tests_written`, plus one
watcher reporting `status:"failed"` with
`task_receipts:[{"task_id":"totally-made-up","status":"completed","summary":"x","handoff":{"verification_status":"pass","commands_run":["echo hi"]}}]`.
`validateBuildProvenance` returns `nil`. The builder's `completed` status then credits
every task it covers through `completedBuildTaskIDs`' whole-success branch,
`allSelectedBuildTasksCredited` is true, and the colony goes to `BUILT` having changed
no file. Before this phase that packet was rejected as
`"the build produced no changes"`.

**Fix:** Do not let a receipt terminate the guard; let it satisfy the same
per-worker requirement the loop already applies, and validate the receipt against the
phase first:

```go
if hasGenuineTaskReceiptEvidence(r) {
    receiptEvidenceCount++   // do not `return nil`
}
continue
...
if completedCount == 0 && receiptEvidenceCount == 0 { return fmt.Errorf(...) }
```

and require at least one receipt whose `files_created|files_modified|tests_written`
is non-empty, so the carve-out cannot be satisfied by a receipt naming no file.

### CR-03: A wrapper-authored manifest can hand itself `completed_task_ids` on the legacy finalize path

**File:** `cmd/codex_build.go:39` and `cmd/codex_build.go:43` (new JSON tags),
`cmd/coherent_job_receipts.go:335` and `cmd/coherent_job_receipts.go:485` (early
`continue` preserves the injected value), `cmd/codex_build.go:2709` (reader)
**Issue:** This phase gave `codexBuildDispatch` serialized `completed_task_ids` and
`task_claims` fields and added both to `.aether/schemas/completion-packet.schema.json`
(so `additionalProperties:false` no longer rejects them). The completion packet
carries `dispatch_manifest`, and `mergeExternalBuildResults` seeds `dispatches[i] = dispatch`
directly from `manifest.Dispatches[i]`. `resolveCoherentJobDispatchReceipts` skips any
dispatch with no `task_receipts` (line 335), leaving the injected `CompletedTaskIDs`
intact, and `completedBuildTaskIDs` now reads exactly that field for non-success
dispatches.

`validateBuildAttemptManifestBinding` only compares the manifest digest when
`attempt_id` **and** `attempt_path` are present; with both omitted it returns
`{Legacy:true}` and a stderr warning ("deliberately kept, not hardened into a
refusal", `cmd/codex_build_finalize.go:469`). So an unbound packet's manifest is
entirely attacker-shaped. Probe:

```
manifest dispatch: {Status:"planned", CoveredTaskIDs:[1.1 1.2 1.3], CompletedTaskIDs:[1.1 1.2 1.3]}
result:            {Name:"Mason-1", Status:"failed"}
→ credited=map[1.1:{} 1.2:{} 1.3:{}]  fullyCredited=true
```

Three tasks credited, zero receipts, zero evidence, and `BUILT`. `task_claims` rides
the same path into `last-build-claims.json` via `claimsOrAggregate`.

**Fix:** These two fields are runtime-owned in-process state, not wire contract.
Mark them `json:"-"` like `ReceiptsResolved` already is, drop them from the packet
schema, and clear them defensively at the merge boundary:

```go
// cmd/codex_build.go
CompletedTaskIDs []string              `json:"-"`
TaskClaims       []codexBuildTaskClaim `json:"-"`

// cmd/codex_build_finalize.go, mergeExternalBuildResults
dispatch.CompletedTaskIDs = nil
dispatch.TaskClaims = nil
```

If they must be serialized for diagnostics, write them to a separate
runtime-authored artifact rather than the round-trippable dispatch object.

### CR-04: The D-10 recovery job is never executed; the surfaced command redoes credited work

**File:** `cmd/coherent_job_retry.go:216` and `cmd/coherent_job_retry.go:231-241`
**Issue:** `reconcilePartialBuildRetry` plans the unfinished-only job, writes it into
a new child attempt's `Dispatches`, then sets
`RedispatchCommand = buildForceRedispatchCommand(phaseNum)` — literally
`"aether build <N> --force"` (`cmd/codex_continue.go:2463`), with no `--task` filter
and no reference to the retry attempt. Nothing anywhere reads
`buildAttemptRecord.ParentAttemptID`, `ParentJobName`, or the child attempt's
`Dispatches`: `grep` for those identifiers outside tests returns only the definition,
the setter, and `findExistingBuildAttemptRetry`'s idempotency scan.

Meanwhile `plannedBuildDispatchesWithJobProposals` (`cmd/codex_build.go:1333-1348`)
seeds from `phase.Tasks` filtered only by `selectedTaskIDs`, never by
`colony.TaskCompleted`. So `aether build N --force` re-plans and re-dispatches every
task in the phase, credited ones included.

Concrete failure: a six-task grouped job credits 4 and fails 2. The owner is told
`recovery_command: aether build 12 --force`; running it sends a worker at all six
tasks. This directly contradicts four shipped surfaces that this phase wrote:
`.claude/commands/ant/build.md:378` ("the exact command that redispatches only the
unfinished tasks ... never ask a new worker to redo credited work"), the identical
text in the other two wrapper copies, `cmd/command_guide.go:333`, and
`CLAUDE.md:293` ("Retrying picks up only the uncredited tasks"). The two cited locks
(`TestCoherentJobRetryContainsOnlyUnfinishedTasks`,
`TestGroupedJobRetryNeverReassignsCreditedTasks`) assert only that
`planCoherentJobRetry` *returns* the right job — neither asserts anything about what
is dispatched, so both pass with the execution wiring absent. Per CLAUDE.md's
Definition of Done this is a requirement with no command that fails when it is unmet.

**Fix:** Emit a command that carries the retry job's task IDs, and make it the
tested contract:

```go
redispatchCommand := fmt.Sprintf("aether build %d --force --task %s",
    phaseNum, strings.Join(uniqueSortedStrings(allUnfinished), " --task "))
```

Then add a test that runs the emitted command's argv through the build planner and
fails if any credited task ID appears in the resulting dispatch list. (A stronger
fix is to have the retry attempt itself be resumable, but the flag-based one closes
the false claim.)

### CR-05: A failed worker's receipt sync writes to root inside a rejected wave, and outside conflict detection

**File:** `cmd/codex_build_worktree.go:527-528` (the `dr.Status != "completed"` case)
and `cmd/codex_build_worktree.go:594-620` (`resolveWorktreePartialReceipts`)
**Issue:** Two problems in one branch.

1. `reconcileWorktreeWave`'s switch orders `case len(conflicts) > 0 && dr.Status == "completed"`
   *before* `case dr.Status != "completed" && session != nil`. When the wave is
   rejected for conflicts, every completed worker is blocked and preserved — but a
   *failed* worker in the same wave still falls into the third case and
   `resolveWorktreePartialReceipts` copies its admitted paths into root via
   `syncRelativePath`. The function's own doc comment two lines above says
   "conflict-free workers are blocked rather than silently accepted, so the root
   never carries a partial wave." It now does.

2. `detectWorktreeWaveConflicts` (`cmd/codex_build_worktree.go:651-654`) only builds
   `touchedBy` from outcomes with `Status == "completed"`. A failed worker's writes are
   invisible to it. Combined with CR-01's gap (a task with no declared paths skips the
   binding check at `cmd/coherent_job_receipts.go:220`, so its receipt may claim any
   path in its own outputs), a failed worker's worktree copy of a file can overwrite
   a same-wave successful worker's already-synced version. Both syncs happen in the
   same `for i, outcome := range outcomes` pass, so index order decides the winner —
   silently, with no conflict reported.

Concrete failure: wave 1 has job A (fails; a covered task declares no paths; its
worker also edited `cmd/foo.go` and receipts it) and job B (completes; declares and
owns `cmd/foo.go`). If A's outcome index is higher than B's, root ends up with A's
copy of `cmd/foo.go` — B's changes are gone, B is reported `completed`, and no
conflict is raised.

**Fix:** Gate the partial-receipt sync on the wave actually being reconcilable, and
feed its paths into conflict detection:

```go
case dr.Status != "completed" && session != nil:
    preserveWorktree = true
    if len(conflicts) == 0 {
        resolveWorktreePartialReceipts(root, phase, outcome, ledger)
    }
```

and include admitted receipt paths from non-completed outcomes in
`detectWorktreeWaveConflicts`'s `touchedBy` map so a cross-owner write is refused
rather than raced.

## Warnings

### WR-01: Every receipt refusal is discarded in both in-repo lanes

**File:** `cmd/coherent_job_receipts.go:339-340` and `cmd/coherent_job_receipts.go:493-497`
**Issue:** `resolveCoherentJobDispatchReceipts` throws away both violation slices
(`admission, _ :=` / `claims, completedTaskIDs, _ :=`), and
`resolveWorktreeExternalDispatchReceipts` never reads `outcome.Violations`. Only
`resolveWorktreePartialReceipts` surfaces anything, and only two of the twelve rules.
So a receipt refused for `task_receipt.out_of_scope`, `task_receipt.path_laundered`,
`task_receipt.path_not_in_aggregate_claims`, or `task_receipt.requirement_mismatch`
vanishes with no owner-visible trace — the file's own header (line 14-16) says
"never a silent drop". An LLM wrapper mis-shaping every receipt looks identical to a
worker that legitimately finished nothing.
**Fix:** Return the violations from `resolveCoherentJobDispatchReceipts` and either
fold them into `codexResultCollectionReport.Issues` (which already exists for
exactly this) or emit them through `visualFprintf(stderr, ...)`.

### WR-02: A `partial` attempt cannot be re-finalized with the same packet

**File:** `cmd/build_attempt.go:800-808` (status switch) and `cmd/codex_build_finalize.go:513`
(idempotency branch)
**Issue:** `buildAttemptPartial` was added to `buildAttemptStatusTerminal` but not to
`validateBuildAttemptManifestBinding`'s accepted-status switch, so it hits `default:`
→ `"build attempt %s is partial and cannot be finalized"`. The idempotent branch above
only matches `binding.Record.Status == buildAttemptBuilt`, and
`reconcileCommittedExternalBuildAttempt` requires `state.State == colony.StateBUILT`
(a partial deliberately leaves `EXECUTING`). Re-submitting the identical packet —
which the code elsewhere explicitly instructs ("rerun build-finalize with the same
completion packet", `cmd/codex_build_finalize.go:754`) — hard-errors.
**Fix:** Add `buildAttemptPartial` to the accepted case list, and add an idempotent
branch for `partial` that re-returns the stored `partialRetryOutcome` when
`CompletionSHA256` matches.

### WR-03: The D-10 retry attempt blocks the plan-only path it tells the owner to use

**File:** `cmd/codex_build.go:382-387`, `cmd/coherent_job_retry.go:231-241`
**Issue:** `beginChildBuildAttempt` → `beginBuildAttempt` writes the child with
`Status: buildAttemptPrepared` (active, `cmd/build_attempt.go:901`) **and** repoints
`latest-attempt.json` at it. The phase's dispatch-start marker is still set from the
parent build, so the next `aether build <N> --plan-only` — the wrapper's only build
path (`.claude/commands/ant/build.md`) — hits
`if _, dispatchStarted := phaseDispatchStartedAt(phaseNum); dispatchStarted` and
returns `"phase N already has workers in flight for build attempt <retry-id>; finalize
or fail that attempt before requesting a fresh plan"`. `--force` is checked *after*
that guard, so it does not bypass it, and the retry attempt has no completion packet
to finalize. The direct `runCodexBuildWithOptions` path recovers (it calls
`interruptLatestBuildAttempt` under `--force`), but the wrapper never takes it.
**Fix:** Either do not move the latest pointer when creating a D-10 child (write the
record and link it without `SaveJSON(pointerRel, ...)`), or create the child with a
terminal-ish, non-active status, or `clearPhaseDispatchWindow(phaseNum)` when the
partial outcome is committed.

### WR-04: The native lane records `failed` for an attempt it just credited

**File:** `cmd/codex_build.go:838` (`recordBuildAttemptTerminal` already set `failed`)
and `cmd/codex_build.go:848` (`attemptFinished = true` with no transition)
**Issue:** `buildAttemptPartial` is only ever written by the external lane
(`cmd/codex_build_finalize.go:761`). In the native lane the partial branch sets
`attemptFinished = true` without transitioning, leaving the journal at `failed` while
`commitPartialBuildCredit` writes real task credit to colony state. Anything reading
the journal (`aether status`, recovery, audits) sees "nothing of this attempt was
credited" — the exact meaning `buildAttemptFailed`'s own comment assigns it
(`cmd/build_attempt.go:30-32`). The two lanes disagree about the same outcome, which
this phase's stated goal says they must not.
**Fix:** `finishAttempt(buildAttemptPartial, "partial credit committed; a D-10 recovery job covers the unfinished tasks", nil)`
before setting `attemptFinished`.

### WR-05: After a native partial credit the owner sees the ordinary "build done" screen

**File:** `cmd/codex_workflow_cmds.go:283`, `cmd/codex_visuals.go:1732-1746`
**Issue:** The partial branch returns a result map with no `"manifest"` key, so the
CLI falls back to `plannedBuildDispatches(...)` and renders
`renderBuildVisualWithDispatches` — which unconditionally prints
`"Verification happens during \`aether continue\`"`, `"Phase N+1 follows after continue"`,
and `NEXT UP: Run \`aether continue\` after the work is implemented`. The one-off
`renderDecisionBlock` warning emitted earlier scrolls above it, and `result["next"]`
(the recovery command) is never shown. A partially-built phase reads as a finished
one on the surface the owner actually looks at — the phase's own "silent partial
success" hazard.
**Fix:** Give the partial branch its own renderer (or pass a `partial` flag into
`renderBuildVisualWithDispatches`) that replaces the Next Up block with
`retryOutcome.RedispatchCommand` and names the unfinished tasks.

### WR-06: `plannedBuildDispatchesWithJudgement` swallows planner errors and returns nil

**File:** `cmd/codex_build.go:1331-1341`
**Issue:** The compatibility shim does `if err != nil { return nil }`. Its callers —
`build_print_brief.go:55`, `codex_build.go:1159`, `codex_visuals.go:1605`,
`codex_visuals.go:3922`, `proof_cmd.go:230` — have no way to distinguish "this phase
plans no workers" from "this phase has a dependency cycle". A phase whose
`depends_on` forms a cycle, or a task resolving to an unknown caste, now renders as
an empty spawn plan everywhere instead of the named, actionable error
`coherentJobGraphPreflight` went to the trouble of producing
(`cmd/coherent_jobs.go:171-173`).
**Fix:** Propagate the error. Change the five call sites to handle
`([]codexBuildDispatch, error)`, or at minimum have the shim log the error to stderr
before returning nil.

### WR-07: Brief-allowance chunking can put two path-overlapping jobs in the same wave

**File:** `cmd/coherent_jobs.go:462-481` (`splitCoherentJobTasksByBriefAllowance`
call site) and `cmd/coherent_jobs.go:821-849` (`populateCoherentJobDAG`)
**Issue:** A component formed purely by `sharedMeaningfulCoherentJobPaths` (no
dependency edges) whose combined task content exceeds
`briefTaskContentAllowanceChars` (6000, `cmd/build_print_brief.go:509`) is split into
`-part-1`/`-part-2` jobs. `populateCoherentJobDAG` derives `DependsOn` only from
`task.DependsOn`, so the two parts have no edge between them and land in the same
wave — while still declaring the same file, which is precisely why they were grouped.

In worktree mode `validateDeclaredWorktreeOwnership` then refuses the whole build
with `"in wave 1, job automatic-X-part-1 (...) and job automatic-X-part-2 (...) both
declare cmd/foo.go; declare disjoint paths, move the work into different waves, or
run with --parallel-mode in-repo"` — advice the owner cannot act on, for a conflict
the runtime created. In in-repo mode the two workers run concurrently against the
same file.
**Fix:** Chain the chunks explicitly when they were split from one component:

```go
if chunkIndex > 0 {
    job.DependsOn = append(job.DependsOn, jobs[len(jobs)-1].Name)
}
```

and have `populateCoherentJobDAG` preserve (rather than overwrite) pre-seeded
`DependsOn` entries.

### WR-08: External-lane receipts are never normalized, so a legal alias is refused

**File:** `cmd/coherent_job_receipts.go:162` vs `pkg/codex/task_receipt.go:28-41`
**Issue:** `normalizeTaskReceipts` (which lowercases status, normalizes claim paths
and runs `NormalizeWorkerHandoff`, mapping `"passed"→"pass"`, `"not run"→"not_run"`)
is only reached from `normalizeWorkerClaims` on the native worker-output path. The
external lane decodes `TaskReceipts` straight off the JSON packet and never
normalizes. `admitCoherentJobTaskReceipts` then tests `verification != "pass"` on the
raw value, so an external receipt carrying `"verification_status":"passed"` — an
alias `ValidateWorkerHandoff` explicitly accepts (`pkg/codex/handoff.go:58`) — is
refused as `task_receipt.unevidenced`, and (per WR-01) refused silently. The two
lanes credit different sets from identical evidence.
**Fix:** Call `codex.NormalizeTaskReceipts(root, receipts)` at the top of
`admitCoherentJobTaskReceipts` (exporting it), so the boundary normalizes once for
every lane rather than depending on which door the receipt came through.

### WR-09: Three of the CLAUDE.md "required" doc anchors cannot fail

**File:** `cmd/claudemd_coherent_jobs_test.go:53-55`
**Issue:** `requiredCoherentJobDocClaims` includes `"in-repo"`, `"worktree"` and
`"refused by name"`. All three already appear in `CLAUDE.md` at the phase's diff base
(3, 5 and 1 occurrences respectively, verified against `git show 85d78686^:CLAUDE.md`)
in the Parallel Execution Modes table and the Queen-judgement section. Deleting every
sentence Phase 195 added to CLAUDE.md would leave
`TestCLAUDEMDStatesCoherentJobContract` green for those three anchors. Per CLAUDE.md's
own Definition of Done, an assertion that passes identically with the feature removed
is not a lock. The other ten anchors are genuinely new and do their job.
**Fix:** Replace the three generic anchors with phrases unique to the Phase 195 text,
e.g. `"grouping is validated before file ownership"` and
`"one job, one worktree, one merge-back"`, and assert each appears exactly once.

### WR-10: The retry child attempt is created before the credit it exists to record is committed

**File:** `cmd/codex_build.go:864-871`
**Issue:** `reconcilePartialBuildRetry` (which writes a new attempt file *and*
repoints `latest-attempt.json`) runs before `commitPartialBuildCredit`. If the commit
fails — `validateRuntimeStateStillCurrent` rejecting a concurrent pause, or a store
write error — the code calls `rollbackCodexBuildFailure` and returns, leaving an
orphaned active child attempt pointing at unfinished tasks whose credit was rolled
back. Combined with WR-03, that orphan then blocks the wrapper's plan-only path.
**Fix:** Commit the credit first, then create the recovery attempt from the committed
state:

```go
partialState, commitErr := commitPartialBuildCredit(phaseNum, startedAt, dispatches)
if commitErr != nil { rollbackCodexBuildFailure(...); return nil, commitErr }
retryOutcome, retryErr := reconcilePartialBuildRetry(partialState, phaseNum, updatedPhase, ...)
```

which also removes the current oddity of passing `originalState` (the pre-build
state) to the retry planner.

## Info

### IN-01: The "intentionally pure" planner mutates its caller's input slice

**File:** `cmd/coherent_jobs.go:235-237`
**Issue:** `normalizeCoherentJobProposal` takes `proposal` by value but writes
`proposal.TaskIDs[idx] = strings.TrimSpace(...)` — the slice header is copied, the
backing array is not. The caller's `[]coherentJobProposal` is mutated in place,
contradicting `planCoherentJobs`' "It is intentionally pure" contract at line 87.
**Fix:** `proposal.TaskIDs = append([]string(nil), proposal.TaskIDs...)` before the
trim loop.

### IN-02: `renderBuildFastPathSummary` byte-slices a string for capitalization

**File:** `cmd/ceremony_team_checkin.go:544-546`
**Issue:** `strings.ToUpper(whyNoApproval[:1]) + whyNoApproval[1:]` splits a
multi-byte rune if any future `Why` string starts with a non-ASCII character,
producing mojibake in owner-facing output. All current values are ASCII.
**Fix:** Use `r, size := utf8.DecodeRuneInString(s)` or `cases.Title`.

### IN-03: Worktree ownership keys on `TaskID`, so two empty-TaskID dispatches collide

**File:** `cmd/codex_build_worktree.go:118`
**Issue:** `worktreeOwnershipIdentity` returns `key: strings.TrimSpace(dispatch.TaskID)`.
Two dispatches that both have an empty `TaskID` (pre-wave/review dispatches) and
non-empty `DeclaredPaths` compare equal, so `previous.key != owner.key` is false and a
genuine overlap between them is not refused. No current dispatch shape hits this.
**Fix:** Fall back to `WorkerName` when `TaskID` is empty.

### IN-04: Retry idempotency and parent-job attribution are approximate

**File:** `cmd/coherent_job_retry.go:221-229` and `cmd/coherent_job_retry.go:233`
**Issue:** `findExistingBuildAttemptRetry` returns *any* child linked to the parent
regardless of whether the unfinished set has since changed, and
`beginChildBuildAttempt(..., retryJobs[0].Name, ...)` records only the first of N
retry jobs as `ParentJobName` when several dispatches partially failed.
**Fix:** Compare the existing child's `SelectedTasks` against `allUnfinished` before
reusing it; join all retry job names for `ParentJobName`.

### IN-05: Three helpers recompute the same two sets

**File:** `cmd/codex_build_finalize.go:936-985`
**Issue:** `allSelectedBuildTasksCredited`, `unfinishedBuildTaskIDs` and
`creditedBuildTaskIDs` each call `buildFullBuildTaskIDSet` + `completedBuildTaskIDs`
independently; all three run per finalize on the same inputs.
**Fix:** Compute the pair once in `runCodexBuildFinalize` and pass the partition down.

---

_Reviewed: 2026-08-27_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
