---
phase: 189-complete-worker-contract
reviewed: 2026-08-20T06:51:36Z
depth: standard
files_reviewed: 13
files_reviewed_list:
  - pkg/codex/handoff.go
  - pkg/codex/worker.go
  - cmd/codex_build.go
  - cmd/codex_build_test.go
  - cmd/codegraph_context.go
  - cmd/codex_continue_plan.go
  - cmd/codex_continue_plan_test.go
  - cmd/continue_wrapper_ceremony_test.go
  - cmd/contract_schema_test.go
  - .claude/commands/ant/continue.md
  - .claude/commands/ant-continue.md
  - .opencode/commands/ant/continue.md
  - .codex/CODEX.md
findings:
  critical: 1
  warning: 2
  info: 1
  total: 4
status: issues_found
---

# Phase 189: Code Review Report

**Reviewed:** 2026-08-20T06:51:36Z
**Depth:** standard (with targeted execution-verified probes per reviewer brief)
**Files Reviewed:** 13
**Status:** issues_found

## Summary

Phase 189 does what it claims on the build side: `renderCodexBuildWorkerBrief` now
resolves every task a merged dispatch covers (via the pre-existing, merge-aware
`dispatchCoveredTaskIDs`) and renders each one's constraints/hints/success criteria
under `**Task N:**` labels, proven by a real invariant test
(`TestBuildWorkerBriefCoversEveryMergedTaskConstraintsAndCriteria`) that exercises the
actual multi-task resolution path rather than asserting a hand-picked string. The
handoff-schema single-sourcing claim also holds up under direct inspection:
`codex.HandoffFieldsSummary` is the one place the field list is spelled out in Go
code, `TestWrapperFieldListMatchesSchema` checks that constant against a JSON Schema
reflected from the real `codex.WorkerHandoff` struct (not a second hand-maintained
copy), and `TestComposeBuildManifestBriefStatesHandoffSchemaOnceNotOnNativePath` /
`TestContinueExternalDispatchBriefsStateHandoffSchemaOnceNotOnNativePath` both prove,
field-by-field, that the schema statement appears exactly once per worker (wrapper
brief only, never the native-Codex raw brief). The three continue wrapper files are
genuinely byte-identical (verified with `diff`, exit 0 on both pairs), and the new
capsule/pheromone instructions name real manifest fields (`context_capsule`,
`pheromone_section`, `dispatch.skill_section`) that the runtime actually populates.

The problem is on the enforcement side of that same handoff-schema claim. Both
`composeBuildManifestBrief` (build) and the newly-added
`continueExternalBriefWithHandoffSchema` (continue) tell every wrapper-spawned worker
"An empty handoff is rejected" — but only build's finalizer actually rejects one.
I traced continue's finalize call chain end to end and proved by direct execution
that a completely empty handoff sails through it with no error. This is the kind of
gap the phase's own framing ("workers see... the output contract the finalizer
enforces") is supposed to close, and it closes it for build while silently leaving
continue exactly where build was before its own fix. I also found and reproduced a
numbering-desync bug in the new merged-dispatch renderer when a covered task ID
doesn't resolve against `phase.Tasks`, and confirmed the new regression-lock test for
brief bloat never exercises the merged-dispatch code path it was extended to cover.

## Critical Issues

### CR-01: Continue's finalizer does not enforce the "an empty handoff is rejected" promise it makes to every wrapper-spawned reviewer/watcher

**File:** `cmd/codex_continue_plan.go:309-311`
**File:** `cmd/codex_continue_finalize.go:614-658` (specifically 654-658)
**File:** `cmd/codex_continue_finalize.go:721-760`

**Issue:**

`continueExternalBriefWithHandoffSchema` — new in this phase — appends this sentence
to every watcher and reviewer brief on continue's wrapper-external path:

```go
// cmd/codex_continue_plan.go:309-311
func continueExternalBriefWithHandoffSchema(rendered string) string {
	return rendered + fmt.Sprintf("\nYour final result's handoff object must include %s. An empty handoff is rejected.\n", codex.HandoffFieldsSummary)
}
```

This is a verbatim copy of the sentence build already uses
(`cmd/codex_build.go:3170`, inside `composeBuildManifestBrief`). Build's finalizer
backs that promise with an explicit check —
`persistExternalBuildHandoffs` (`cmd/codex_build_finalize.go:1259-1284`) contains:

```go
// cmd/codex_build_finalize.go:1277-1284
if status == buildWorkerCompleted && codex.IsEmptyWorkerHandoff(result.Handoff) {
    return fmt.Errorf("worker %s completed without a handoff; completed workers must relay changed_files, commands_run, verification_status, and next_worker_instructions so the next phase inherits their context", resultName)
}
```

and this is locked in by `TestBuildFinalizeRejectsCompletedWorkerWithoutHandoff`
(`cmd/build_attempt_external_test.go:342-368`).

Continue's finalize call chain has no equivalent check anywhere:

- `mergeExternalContinueResults` (`cmd/codex_continue_finalize.go:614-682`) calls
  only `codex.ValidateWorkerHandoff(result.Handoff)` at line 655 — and
  `ValidateWorkerHandoff` (`pkg/codex/handoff.go:46-60`) only format-checks
  `VerificationStatus`, explicitly accepting `""` as valid. It never inspects any
  other field, so it cannot detect "nothing was relayed."
- `persistExternalContinueHandoffs` (`cmd/codex_continue_finalize.go:721-760`) then
  unconditionally calls `persistDispatchWorkerHandoff` for every result regardless of
  content — no `codex.IsEmptyWorkerHandoff` call anywhere in this file.
- `loadExternalContinueCompletion` (`cmd/codex_continue_finalize.go:70-107`), the
  entry point that parses the wrapper's submitted JSON, is a plain
  `json.Unmarshal` with no structural JSON-Schema validation step (build's
  equivalent path runs the completion packet through a reflected/compiled schema;
  continue's does not), so there is no earlier gate either.

I proved this by execution: I added a temporary test calling
`mergeExternalContinueResults` with a single "completed" watcher result carrying
`Handoff: codex.WorkerHandoff{}` (fully empty). It returned `err == nil`:

```
mergeExternalContinueResults err = <nil>
flow = [{Stage:verification Caste:watcher Name:Keen-1 ... Status:completed Summary:looks fine ...}]
```

The probe file was deleted after confirming the result (`git status --porcelain` is
clean; `go build ./cmd/...` still succeeds).

The consequence: any continue reviewer or watcher (including the Auditor and
Gatekeeper, which are explicitly briefed to report only via `findings`, not code
changes) can return a content-free handoff and have it silently accepted and
persisted into `handoffs/worker-handoffs.json` — precisely the "written but empty,
read but not delivered" failure mode the build-side comment at
`cmd/codex_build_finalize.go:1278-1280` describes as the reason its own check exists.
Phase 189's stated goal is that workers are told "the output contract the finalizer
enforces" on every path; for continue, the contract stated is not the contract
enforced.

**Fix:** Add the same rejection continue's brief already promises, e.g. in
`mergeExternalContinueResults` right after the existing `ValidateWorkerHandoff` call
(so it fails at the same point build effectively does, and with the same class of
worker-facing error):

```go
// cmd/codex_continue_finalize.go, inside mergeExternalContinueResults, after line 658
if ok && status == "completed" && codex.IsEmptyWorkerHandoff(result.Handoff) {
    return nil, fmt.Errorf("external continue result for %s completed without a handoff; completed reviewers must relay findings or next_worker_instructions so later phases inherit their context", dispatch.Name)
}
```
Add a `TestContinueFinalizeRejectsCompletedWorkerWithoutHandoff` mirroring
`TestBuildFinalizeRejectsCompletedWorkerWithoutHandoff` so this can't regress silently
again. Since the promise text is now duplicated verbatim in two files
(`codex_build.go:3170`, `codex_continue_plan.go:310`), consider factoring it into a
single helper (e.g. `codex.HandoffRejectionNotice()`) next to `HandoffFieldsSummary`
so the policy sentence and its enforcement can't drift apart the same way again.

## Warnings

### WR-01: Merged-dispatch brief mislabels "Task N:" sections when a covered task ID doesn't resolve against `phase.Tasks`

**File:** `cmd/codex_build.go:2794-2817` (`findDispatchTasks`)
**File:** `cmd/codex_build.go:2833-2886` (`renderDispatchTaskItemsSection`)

**Issue:** `findDispatchTasks`'s doc comment states it is defensive: "A covered ID
with no matching phase.Tasks entry is skipped silently... a stale or renamed task ID
must never panic or drop the rest of the brief." It does avoid a panic, but the
"skip" corrupts the numbering of every task after the gap, because
`renderDispatchTaskItemsSection` labels each block by its position in the *filtered*
`tasks` slice (`i+1`), while the "## Assignment" section's numbering comes from
`mergeDispatchInto` (`cmd/codex_build.go:1077-1091`), which numbers unconditionally
by original chain position and never sees the gap.

I proved this by execution: a 3-task merged dispatch (`CoveredTaskIDs: ["1","2","3"]`)
where task "2" was omitted from `phase.Tasks` (simulating a stale/renamed ID)
produced this brief:

```
## Assignment

1. first
2. second
3. third

## Task Constraints

**Task 1:**
- constraint-in-task-1
**Task 2:**
- constraint-in-task-3
```

Task 2 ("second") has no constraints section at all (silently invisible — there is
no indication anything was skipped), and task 3's ("third") constraint is displayed
under the label **Task 2**, misattributing it to the wrong assignment item. A worker
reading this brief has no way to tell that "Task 2" in Task Constraints is not the
same task as "2. second" in the Assignment. The same mechanism also means a duplicate
ID in `CoveredTaskIDs` (not currently produced by `coalesceSequentialDispatches`
under any path I could find, but not guarded against here either) renders one task's
content twice under two different "Task N:" labels — confirmed with a second executed
probe.

Within the current single-call planning path
(`plannedBuildDispatchesWithJudgement` → `coalesceSequentialDispatches` →
`renderCodexBuildWorkerBrief`, all reading the same `phase.Tasks` snapshot) I could
not find a live route that produces a stale ID today — `buildTaskID` only falls back
to a positional ID when `task.ID` is nil, and `--print-brief` recomputes dispatches
fresh from the current phase before rendering. This is a latent defect in new code
whose own comment claims a stronger guarantee than it delivers, not a demonstrated
production incident.

**Fix:** Thread the *original* 1-based covered-chain position through
`findDispatchTasks` instead of relying on the filtered slice's index, so a missing
task never shifts the labels of the ones after it:

```go
type coveredDispatchTask struct {
    Position int // 1-based position in CoveredTaskIDs, stable across a stale ID
    Task     *colony.Task
}

func findDispatchTasks(phase colony.Phase, dispatch codexBuildDispatch) []coveredDispatchTask {
    coveredIDs := dispatchCoveredTaskIDs(dispatch)
    byID := make(map[string]*colony.Task, len(phase.Tasks))
    for i := range phase.Tasks {
        byID[buildTaskID(phase.Tasks[i], i)] = &phase.Tasks[i]
    }
    var tasks []coveredDispatchTask
    for i, id := range coveredIDs {
        if task, ok := byID[id]; ok {
            tasks = append(tasks, coveredDispatchTask{Position: i + 1, Task: task})
        }
    }
    return tasks
}
```
and label with `.Position` in `renderDispatchTaskItemsSection` instead of the loop
index. Add a regression test with a 3-task chain where the middle ID is stale,
asserting the surviving labels read "Task 1" and "Task 3", not "Task 1" and "Task 2".

### WR-02: `TestBuildWorkerBriefIsMostlyTask`'s regression lock never exercises the merged-dispatch path it now needs to protect

**File:** `cmd/codex_build_test.go:3645-3697`

**Issue:** The test's own comment claims broad protection: "any future addition that
pushes framework scaffolding past half the prompt fails here regardless of what that
addition is called." Its fixture, however, is a single-task dispatch with no
`TaskID`, no `CoveredTaskIDs`, and an empty `phase.Tasks` — `dispatchCoveredTaskIDs`
returns nil for it, so `findDispatchTasks` returns nil and
`renderDispatchTaskItemsSection` is a no-op for every one of the three sections it
guards (Task Constraints, Hints, Task Success Criteria). The sibling test added in
this same phase, `TestBuildWorkerBriefCoversEveryMergedTaskConstraintsAndCriteria`
(`cmd/codex_build_test.go:3707-3766`), proves merged briefs render correctly but
asserts nothing about the brief's overall scaffolding ratio.

I could not prove a merged brief is actually at risk of tipping past the 40%
threshold today — task content (Assignment + N sets of constraints/hints/criteria)
generally grows faster than the fixed-size scaffolding sections as more tasks are
merged, so the ratio more plausibly improves than degrades. But `codegraphTextPartsForBuildBrief`
(`cmd/codegraph_context.go:105-120`, also touched by this phase) now aggregates every
covered task's Goal/Constraints/Hints/SuccessCriteria into the codegraph-context
inference, which can pull in a larger `## Codebase Graph Context` section (capped at
2200 chars) for a chain of many thinly-detailed tasks — a scenario this test cannot
detect either way, because it never runs a merged dispatch through the shared assertion.

**Fix:** Add a merged-dispatch case to (or a sibling of)
`TestBuildWorkerBriefIsMostlyTask` reusing a fixture like
`TestBuildWorkerBriefCoversEveryMergedTaskConstraintsAndCriteria`'s 3-task chain, and
assert the same `share >= 40` invariant against it, so the "regression lock" claim is
actually true for the code path this phase added.

## Info

### IN-01: Three near-identical "is this handoff empty" functions across two packages

**File:** `pkg/codex/handoff.go:34-43` (`IsEmptyWorkerHandoff`, exported)
**File:** `pkg/codex/handoff.go:100-110` (`workerHandoffIsEmpty`, unexported, same file)
**File:** `cmd/codex_dispatch_contract.go:888` (`workerHandoffEmpty`, unexported, different package)

**Issue:** All three test the same eight-or-nine `WorkerHandoff` fields for
blankness, with only `workerHandoffIsEmpty` also checking `Freshness`. They serve
different callers today (`IsEmptyWorkerHandoff` gates build's rejection;
`workerHandoffIsEmpty` gates the native-dispatch synthesis fallback in
`normalizeWorkerClaims`; `workerHandoffEmpty` gates the handoff-record synthesis
fallback in `buildWorkerHandoffRecord`), but the near-identical naming makes it easy
to see one of them called somewhere in the persistence path and reasonably — but
wrongly — conclude that emptiness rejection is already handled generally, which is
close to how CR-01 above went unnoticed. `pkg/codex/handoff.go` is a file this phase
already edited (to add `HandoffFieldsSummary`).

**Fix:** When fixing CR-01, reuse `codex.IsEmptyWorkerHandoff` rather than adding a
fourth variant, and consider consolidating `workerHandoffIsEmpty` into it (or
renaming it to make the distinction from the exported function obvious, e.g.
`isEmptyHandoffIncludingFreshness`) so a future reader doesn't have to trace three
functions to find out which one, if any, actually blocks something.

---

_Reviewed: 2026-08-20T06:51:36Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
