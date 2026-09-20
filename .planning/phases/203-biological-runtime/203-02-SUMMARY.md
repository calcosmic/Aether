---
phase: 203-biological-runtime
plan: "02"
subsystem: recruitment
tags: [spawn-admission, biological-runtime, live-events, process-dispatch]

# Dependency graph
requires:
  - phase: 203-biological-runtime
    provides: "203-01's signed CLASSIC-SYNTHESIS.md (SYN-203-01/02/06/08 dispositions, ruling a/b/c/d) governing this plan's design"
provides:
  - "aether recruit -- the first real recruitment work path, admitted through the existing spawnCanSpawnDecision chokepoint with no second limiter"
  - "dispatchRecruitment -- leased-workspace child process dispatch under a bounded, process-group-safe timeout (AETHER_RECRUIT_TIMEOUT)"
  - "bindRecruitmentResult -- exactly-once recruitment result binding, PauseHandoff-shaped"
  - "live.recruit.admitted / live.recruit.refused topics on the existing Phase 202 live-event boundary"
  - "codex.ConfigureWorkerCommand -- exported process-group hardening for cross-package worker-shaped dispatch"
affects: [203-03, 203-04, 203-06, 203-07, 203-08, 203-09]

# Actuals (#2632)
actuals:
  tokens: 10830
  tasks: 2
  commits: 2

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Extend the singular admission chokepoint (spawnCanSpawnDecision), never fork a parallel one -- SYN-203-02"
    - "Reuse colony.PauseHandoff's ID+Transaction+Receipt idempotency shape verbatim for a structurally identical replay-safety problem, rather than inventing a second mechanism"
    - "Test-only dispatch overrides (AETHER_RECRUIT_BINARY/AETHER_RECRUIT_ARGS, newline-delimited) let a real end-to-end test drive a real subprocess without depending on an installed platform CLI"
    - "The go-test-binary-as-its-own-subprocess-child idiom (os.Args[0] + -test.run, already used elsewhere in this package) reused to prove real child-process dispatch deterministically in CI"

key-files:
  created:
    - cmd/recruitment_intent.go
    - cmd/recruitment.go
    - cmd/recruitment_dispatch.go
    - cmd/recruitment_result.go
    - cmd/recruitment_test.go
  modified:
    - pkg/events/colony_live.go
    - cmd/live_events.go
    - pkg/codex/worker.go

key-decisions:
  - "A refused recruitment must call outputOK, never outputError -- outputError marks a non-zero process exit via markRenderedCommandError (cmd/root.go), which would have made a routine, expected refusal look like a command failure and violated D-03 (a refused recruitment never blocks the caller). Found and fixed during manual smoke-testing before Task 2's tests were written."
  - "Exported codex.ConfigureWorkerCommand as a thin wrapper over the unexported configureWorkerCommand: cmd/recruitment_dispatch.go is package cmd and cannot call an unexported pkg/codex symbol across the package boundary. Every existing in-package caller is untouched; this is purely an additive seam (Rule 3 -- blocking cross-package access issue, not anticipated by the plan's literal wording)."
  - "Workspace containment validation happens inside dispatchRecruitment, AFTER the spawn entry is recorded and the admitted event is emitted, exactly as the plan's own paragraph ordering describes -- a bad workspace is a DISPATCH failure (spawn entry stays, terminal_status becomes failed), not an ADMISSION refusal. Only the depth/budget/ancestor-cycle triad (unchanged, inside spawnCanSpawnDecision) can produce the 'records no spawn entry' refusal Task 2's behavior spec names."
  - "recruitmentDispatchArgv's test-only override (AETHER_RECRUIT_ARGS) is newline-delimited, not space-delimited, so a test can drive a real shell child needing one arg that itself contains spaces (e.g. `sh -c \"sleep 30 & exit 0\"`) -- caught during test-writing when a naive strings.Fields() split broke exactly that case."

requirements-completed: [BIO-01, BIO-02, BIO-03, BIO-04]

coverage:
  - id: D1
    description: "A worker running with only Bash can emit a recruitment intent through the real public `aether recruit` command, and Go admits it through the existing spawnCanSpawnDecision chokepoint (one call site, no second limiter)"
    requirement: "BIO-01"
    verification:
      - kind: unit
        ref: "cmd/recruitment_test.go#TestRecruitmentTracerEndToEnd/admitted_recruitment_starts_a_real_child_and_binds_one_result"
        status: pass
      - kind: unit
        ref: "cmd/recruitment_test.go#TestRecruitmentTracerEndToEnd/refused_at_the_depth_cap_publishes_one_refusal,_records_nothing,_exits_0"
        status: pass
      - kind: other
        ref: "grep -c 'spawnCanSpawnDecision(' cmd/recruitment.go == 1"
        status: pass
    human_judgment: false
  - id: D2
    description: "A refused recruitment exits 0, records no spawn entry, starts no process, and tells the caller to carry on alone (D-03)"
    requirement: "BIO-02"
    verification:
      - kind: unit
        ref: "cmd/recruitment_test.go#TestRecruitmentTracerEndToEnd/refused_at_the_depth_cap_publishes_one_refusal,_records_nothing,_exits_0"
        status: pass
    human_judgment: false
  - id: D3
    description: "The admitted child runs in an exact, containment-validated workspace lease under a bounded, process-group-safe timeout (AETHER_RECRUIT_TIMEOUT), so the budget is real rather than advisory"
    requirement: "BIO-03"
    verification:
      - kind: unit
        ref: "cmd/recruitment_test.go#TestRecruitmentDispatchTerminatesWholeProcessGroup"
        status: pass
    human_judgment: false
  - id: D4
    description: "Binding the same recruitment result identifier twice returns the stored receipt and applies nothing a second time"
    requirement: "BIO-04"
    verification:
      - kind: unit
        ref: "cmd/recruitment_test.go#TestRecruitmentTracerEndToEnd/replaying_the_same_terminal_result_twice_is_idempotent"
        status: pass
    human_judgment: false
  - id: D5
    description: "One admitted recruit and one refusal each publish exactly one live event through the existing single emission boundary; no file outside cmd/live_events.go may publish a live.recruit.* topic"
    verification:
      - kind: unit
        ref: "cmd/recruitment_test.go#TestRecruitLiveEventsGoThroughTheOneBoundary"
        status: pass
      - kind: unit
        ref: "cmd/recruitment_test.go#TestRecruitmentTracerEndToEnd (both subtests assert exactly-one-event via bus.Replay)"
        status: pass
    human_judgment: false
  - id: D6
    description: "A plain build path that never recruits performs no recruitment work at all -- no probe, no extra file read, no new required step"
    verification:
      - kind: unit
        ref: "cmd/recruitment_test.go#TestPlainBuildPathDoesNotTouchRecruitment"
        status: pass
    human_judgment: false

duration: 90min
completed: 2026-09-13
status: complete
---

# Phase 203 Plan 02: Governed Recruitment Tracer Summary

**A worker can now run `aether recruit`, get admitted or refused through the same `spawnCanSpawnDecision` chokepoint every ordinary spawn already uses, watch a real leased child process run under a bounded timeout, and see the outcome bind exactly once -- proven end to end by a test that drives the real command, not the internal functions.**

## Performance

- **Duration:** 90 min
- **Started:** 2026-09-13T00:57:16+02:00 (immediately following 203-01)
- **Completed:** 2026-09-13
- **Tasks:** 2 completed
- **Files modified:** 8 (5 created, 3 modified)

## Accomplishments

- Built the whole recruitment skeleton in one thin, production-quality slice: `cmd/recruitment.go` (the public `aether recruit` command), `cmd/recruitment_dispatch.go` (leased-workspace child dispatch), `cmd/recruitment_result.go` (exactly-once result binding), and `cmd/recruitment_intent.go` (the wire shape)
- Admission runs through the SAME `spawnCanSpawnDecision` function every ordinary spawn already calls -- confirmed by grep (`cmd/recruitment.go` contains exactly one call site) and by a manual CLI run against a real spawn-tree fixture (a parent recorded at depth 2 is refused with reason class `depth`, names the parent and the cap number, and exits 0)
- Found and fixed a real bug before it shipped: the first refusal-path draft called `outputError`, which marks a non-zero process exit code through `cmd/root.go`'s `markRenderedCommandError` -- that would have made an ordinary, expected refusal look like a command failure, directly contradicting D-03 ("a refused recruitment never blocks the caller"). Switched to `outputOK`; confirmed by a live CLI run that the process now exits 0
- Wired the two new `live.recruit.admitted` / `live.recruit.refused` topics through the existing Phase 202 `emitColonyLive` boundary -- `ColonyLiveTopics()` now returns 19 entries, `ColonyLivePayload` gained exactly one new field (`Reason`), and an AST-based test proves no file besides `cmd/live_events.go` may ever publish a `live.recruit.*` topic
- Added `codex.ConfigureWorkerCommand` as a small exported wrapper over the already-tested `configureWorkerCommand` (process group + Cancel + WaitDelay), since `cmd/recruitment_dispatch.go` cannot call the unexported original across the package boundary -- proved load-bearing, not decorative, by a dedicated test that fails when the call is deleted
- `TestRecruitmentTracerEndToEnd` drives the real `recruitCmd` through `rootCmd`, never the internal functions directly: it writes a real spawn-tree entry via `SpawnTree.RecordSpawn`, then asserts the admitted branch (real child process, exact spawn-tree entry, one bound result, one live event) and the refused branch (byte-identical spawn-tree.txt, one live event, exit 0) end to end
- `TestPlainBuildPathDoesNotTouchRecruitment` turns the folded-todo constraint ("a plain build pays nothing for this feature") into a failing check: an AST scan of the core build-dispatch files (`codex_build.go`, `codex_build_worktree.go`, `codex_dispatch_contract.go`, `coherent_jobs.go`) for any reference to a recruitment-prefixed symbol

## Task Commits

Each task was committed atomically:

1. **Task 1: End-to-end governed recruitment -- one worker, one helper, one path** - `6e41a37c` (feat)
2. **Task 2: Prove the slice on the real public path and prove the no-recruitment path is untouched** - `ce754ced` (test)

**Plan metadata:** this commit (docs: complete plan)

## Files Created/Modified

- `cmd/recruitment_intent.go` - `recruitmentSchemaVersion`, `recruitmentIntent`, `recruitmentDecisionResult`
- `cmd/recruitment.go` - `recruitCmd` (the `aether recruit` command), the single `spawnCanSpawnDecision` call site, refusal/admission handling
- `cmd/recruitment_dispatch.go` - `dispatchRecruitment`, `resolvedRecruitmentTimeout` (`AETHER_RECRUIT_TIMEOUT`), `resolvedRecruitmentBinary`, `recruitmentDispatchArgv`
- `cmd/recruitment_result.go` - `recruitmentResult`, `bindRecruitmentResult` (exactly-once binding at `recruitment/results.json`)
- `cmd/recruitment_test.go` - `TestRecruitmentTracerEndToEnd`, `TestRecruitmentDispatchTerminatesWholeProcessGroup`, `TestRecruitLiveEventsGoThroughTheOneBoundary`, `TestPlainBuildPathDoesNotTouchRecruitment`
- `pkg/events/colony_live.go` - `LiveTopicRecruitAdmitted`, `LiveTopicRecruitRefused`, `ColonyLivePayload.Reason`
- `cmd/live_events.go` - `emitColonyLiveRecruitAdmitted`, `emitColonyLiveRecruitRefused`, `currentLiveRecruitmentEpisode`
- `pkg/codex/worker.go` - `ConfigureWorkerCommand` (exported wrapper; see Deviations)

## Decisions Made

- **`outputOK`, not `outputError`, on refusal.** `outputError` marks the process's own exit code via `markRenderedCommandError` -- correct for `spawn-log`'s deny path (a real failure), wrong here (D-03: a refusal is an expected, non-blocking outcome). Verified live: the CLI now prints `{"ok":true,"result":{"admitted":false,...}}` and exits 0.
- **`codex.ConfigureWorkerCommand` exported, nothing else in `pkg/codex` touched.** The plan's own text names `configureWorkerCommand` without qualification; it is unexported and package-private to `pkg/codex`, so `cmd/recruitment_dispatch.go` (package `cmd`) cannot call it directly. A one-line exported wrapper is the minimal fix; every existing in-package caller is untouched.
- **Workspace validation lives inside `dispatchRecruitment`, after the spawn is recorded** -- matching the plan's own paragraph order ("record the spawn... emit the admitted live event, then dispatch"). A bad workspace is therefore a DISPATCH failure (the spawn entry stays, terminal status becomes `failed`, the summary names the escaping path), never an ADMISSION refusal. Manually verified: `--workspace /etc` against a colony rooted elsewhere is admitted, then fails dispatch with a summary naming `/etc` and the colony root.
- **Newline-delimited test overrides, not space-delimited.** `AETHER_RECRUIT_ARGS` needed to carry a single argument containing spaces (`sh -c "sleep 30 & exit 0"`) for the process-group test; a naive `strings.Fields` split would have broken it into multiple args. Switched to `strings.Split(raw, "\n")`.
- **The recruited child's env carries `AETHER_RECRUIT_CHILD=1`, `AETHER_RECRUIT_PARENT`, `AETHER_RECRUIT_CASTE`, `AETHER_RECRUIT_CHILD_NAME` unconditionally** -- a legitimate production feature (the child knows it was recruited, and by whom), which also let `TestRecruitmentTracerEndToEnd` reuse itself, recursively invoked via `os.Args[0]`, as a real deterministic child process without any special test-only branching logic beyond checking that one env var.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] `configureWorkerCommand` is unexported; `cmd/recruitment_dispatch.go` cannot call it across the package boundary**
- **Found during:** Task 1 (writing `dispatchRecruitment`)
- **Issue:** The plan's action text instructs calling `configureWorkerCommand` directly, but that function is package-private to `pkg/codex`. `cmd/recruitment_dispatch.go` is package `cmd`.
- **Fix:** Added `codex.ConfigureWorkerCommand(cmd *exec.Cmd)`, a one-line exported wrapper immediately after the unexported original in `pkg/codex/worker.go`, calling straight through. No existing in-package caller was touched.
- **Files modified:** `pkg/codex/worker.go` (not in this plan's declared `files_modified` list -- flagged here rather than silently expanded)
- **Verification:** `TestRecruitmentDispatchTerminatesWholeProcessGroup` fails (12s timeout, names the leaked grandchild) when the `codex.ConfigureWorkerCommand(execCmd)` call is deleted from `dispatchRecruitment`, and passes with it present. Manually re-verified by temporarily commenting out the call, confirming the failure, then reverting.
- **Committed in:** `6e41a37c` (Task 1 commit)

**2. [Rule 1 - Bug] Refusal path exited non-zero via `outputError`**
- **Found during:** Task 1, manual smoke-testing before Task 2's tests existed
- **Issue:** `outputError` calls `markRenderedCommandError`, which makes `cmd/root.go`'s `Execute()` wrapper return a `renderedCommandError` and exit the process non-zero -- directly contradicting the plan's must_have ("A refused recruitment exits zero...") and D-03.
- **Fix:** Switched the refusal branch to `outputOK` with an `"admitted": false` result payload carrying the reason class, detail, and an explicit "carry on with the task alone" message.
- **Files modified:** `cmd/recruitment.go`
- **Verification:** Manual CLI run against a real depth-2 spawn-tree fixture: `EXIT:0`, output contains the parent name, the cap number, and the reason class `depth`. `TestRecruitmentTracerEndToEnd`'s refusal subtest asserts `renderedCommandExitCode.Load() == 0`.
- **Committed in:** `6e41a37c` (Task 1 commit)

---

**Total deviations:** 2 auto-fixed (1 blocking cross-package access, 1 bug)
**Impact on plan:** Both fixes were necessary for the tracer's own literal must_haves (a real cross-package call, and a genuinely zero-exit refusal). No scope creep: the `pkg/codex` change is a single additive exported line; no existing caller's behavior changed.

## Issues Encountered

None beyond the two deviations above, which were found and fixed within Task 1's own execution before Task 2's tests were written.

## Threat Flags

| Flag | File | Description |
|------|------|--------------|
| threat_flag: new-process-spawn-surface | cmd/recruitment_dispatch.go | `dispatchRecruitment` launches an external binary (resolved via `AETHER_RECRUIT_BINARY`/`AETHER_CODEX_PATH`/default `codex`) with argv built from caller-supplied `--caste`/`--objective` flag values, inside a workspace validated only for path containment against the colony root. Depth, whole-run budget, and ancestor-cycle admission are the existing `spawnCanSpawnDecision` checks (unchanged, already fail-closed and tested); permission, per-recruit cost, and duplicate-intent dimensions are explicitly out of this tracer's scope and are 203-06's job (BIO-02's full admission gate). No `<threat_model>` block exists in 203-02-PLAN.md to check this surface against -- flagged here for the verifier and for 203-06's planner. |

## Known Stubs

- **Ticker rendering for the two new topics falls back to the generic "activity recorded" line** (`cmd/watch_dashboard.go:colonyLiveTickerDescription`'s `default` case). This is honest (no jargon leak, `TestLiveTickerLinesArePlainEnglish` still passes) but not yet the "one line per recruit (who, why, cost so far)" D-04 describes, or the refusal-specific wording D-06 describes. `cmd/watch_dashboard.go` is not in this plan's `files_modified` list -- per the 203-CLASSIC-SYNTHESIS.md architecture, "Inline + end-of-run rendering (SYN-203-09)" is explicitly plan 203-09's job, reusing the existing caste-identity renderer rather than building a second one. Not fixed here to avoid scope creep into a file another plan owns.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- The skeleton is proven end to end: intent -> admission (existing chokepoint, unchanged) -> leased child dispatch (bounded, process-group-safe) -> exactly-once result -> one live event, all through a real CLI command a worker with only Bash can run.
- `requirements-completed` above lists BIO-01..04 as this plan's declared scope per its own frontmatter, but this tracer intentionally carries only the MINIMAL wire shape for each -- BIO-01's full field set (evidence, permissions, urgency, scope, cost limit) is plan 203-03's job; BIO-02's permission/path/cost/duplicate/parent-authority dimensions are plan 203-06's job; BIO-03's platform probe and native-nesting reporting are not yet built; BIO-04's execution-generation/artifacts/usage fields beyond the terminal status are not yet carried. The REQUIREMENTS.md checkboxes for BIO-01..04 are intentionally left unchecked by this plan -- sibling plans in this phase (203-03, 203-04, 203-06, 203-07) declare the same requirement IDs and extend this skeleton to their full scope; marking them complete here would be premature.
- No blockers for 203-03 onward: every symbol this plan's must_haves name (`recruitCmd`, `dispatchRecruitment`, `bindRecruitmentResult`, `LiveTopicRecruitAdmitted`, `LiveTopicRecruitRefused`) exists, is real, and is proven by a test driving the actual public command.

---
*Phase: 203-biological-runtime*
*Completed: 2026-09-13*

## Self-Check: PASSED

- FOUND: `cmd/recruitment_intent.go`
- FOUND: `cmd/recruitment.go`
- FOUND: `cmd/recruitment_dispatch.go`
- FOUND: `cmd/recruitment_result.go`
- FOUND: `cmd/recruitment_test.go`
- FOUND: commit `6e41a37c` (feat: governed recruitment tracer end-to-end) in `git log --oneline`
- FOUND: commit `ce754ced` (test: prove the recruitment tracer on the real public path) in `git log --oneline`
- Re-ran plan-level `<verification>`: `go build ./... && go vet ./cmd ./pkg/events && go test ./cmd -run '^(TestRecruitmentTracerEndToEnd|TestRecruitLiveEventsGoThroughTheOneBoundary|TestPlainBuildPathDoesNotTouchRecruitment)$' -count=1 && go test ./pkg/events -count=1` -- PASS
- Re-ran acceptance-criteria negative proofs by breaking and reverting each guarantee in turn: deleting `codex.ConfigureWorkerCommand(execCmd)` makes `TestRecruitmentDispatchTerminatesWholeProcessGroup` fail; making `bindRecruitmentResult` always append makes `TestRecruitmentTracerEndToEnd`'s replay subtest fail; adding a stray `bus.Publish("live.recruit.admitted", ...)` in `cmd/spawn.go` makes `TestRecruitLiveEventsGoThroughTheOneBoundary` fail and names `spawn.go` -- all three confirmed, then reverted (working tree clean, `git diff` empty)
