---
phase: 198-put-the-thrown-away-data-back-on-screen
plan: 05
subsystem: cli-visuals
tags: [go, ceremony, continue, worker-measurements, D-03, SHOW-02]

# Dependency graph
requires:
  - phase: 198-01
    provides: closeoutDirectVisual / closeoutContinueRenderInputs bridge, the dual-type rendering precedent (renderContinueWorkerFlowValue), and the byte-equality parity test fixtures (wrapperParityColonyState, wrapperParityAdvanceResult, roundTripToMap, firstDiffLine)
  - phase: 198-02
    provides: verificationStepDisplayName / live check-line pattern for build/types/lint/tests (a sibling measurement surface, not touched by this plan)
provides:
  - "codexContinueWorkerFlowStep.ToolCount + DurationReported/ToolCountReported reported flags, serialized and populated from the trusted in-process WorkerResult"
  - "workerMeasurementFigures (cmd/codex_visuals.go) — the single formatter for a worker's measured duration and tool-call count, shared by the live finishing line and the continue worker-flow summary"
  - "A per-worker 'measured:' detail line under the continue worker-flow summary render, for both the typed and JSON-round-tripped branches"
affects: [any-future-phase-touching-continue-worker-flow-rendering, 198-06-onward]

# Actuals (#2632)
actuals:
  tokens: 6333
  tasks: 3
  commits: 3

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "One formatter, two call sites: workerMeasurementFigures is the single place a worker duration or tool count becomes display text, called by both the live finishing line and the closing summary, locked by an AST guard scanning for inline \"tool call\" text in every other function in the two files it touches."
    - "Reported flags travel with the figure, not inferred from a zero: DurationReported/ToolCountReported are explicit serialized booleans set only when a real WorkerResult existed, so a worker that genuinely reported zero tool calls is distinguishable from one whose platform reported nothing at all (Phase 196 D-01 applied to a new figure pair)."

key-files:
  created:
    - cmd/continue_worker_measurements_test.go
  modified:
    - cmd/codex_continue.go
    - cmd/codex_visuals.go
    - cmd/codex_build_progress.go
    - cmd/testdata/golden_build.txt
    - cmd/testdata/golden_continue.txt
    - cmd/testdata/golden_plan.txt

key-decisions:
  - "\"Reported\" is scoped to whether a real WorkerResult existed, not to whether the worker's own JSON claims block included the tool_count key. pkg/codex/worker.go's workerClaims.ToolCount has no omitempty and no reported flag of its own, so that finer-grained ambiguity (explicit zero vs. absent key in a worker's own claims) is a pre-existing condition upstream of this plan's file scope (cmd/codex_continue.go, cmd/codex_continue_finalize.go) and was left untouched -- this plan closes the gap between WorkerResult and the screen, not between a worker's raw output and WorkerResult."
  - "The per-worker 'measured:' detail line is scoped to Stage==\"review\" && Caste!=\"system\" -- the exact flow this plan's Task 1 wires ToolCount into. Synthetic bookkeeping steps (deterministic verification, signal housekeeping, phase-end consolidation, and the 'review wave skipped' placeholder) never went through a worker dispatch this plan measures, so they carry no measurement line rather than a spurious 'not reported' on every non-worker line."
  - "The external/wrapper continue lane (mergeExternalContinueResults, codexContinueExternalDispatch) was left unwired for ToolCount: that struct has no tool_count field and lives in cmd/codex_continue_plan.go, outside this plan's stated file scope (cmd/codex_continue.go, cmd/codex_continue_finalize.go). Only the native/in-process review-worker flow populates ToolCount in this plan; wiring the external lane's completion-packet schema is a distinct, larger change (adding a field workers submit) and is left for a future plan if needed."
  - "The finishing-line format changed from a bare ' %.1fs' suffix to a parenthetical '(3m 10s, 14 tool calls)' per D-03's literal example. This is a visible, intentional format change (not a bug) affecting emitCodexDispatchWorkerFinished, shared by build and continue review dispatch via runtimeVisualDispatchObserver — golden_build.txt, golden_plan.txt, and golden_continue.txt were regenerated to match, and durations under a minute keep the existing '%.1fs' precision inside the new parenthetical."

requirements-completed: [SHOW-02]

coverage:
  - id: D1
    description: "Each reviewer helper's finishing line says how long it took and how many tool calls it made (D-03)."
    requirement: "SHOW-02"
    verification:
      - kind: unit
        ref: "cmd/continue_worker_measurements_test.go#TestLiveAndSummaryWorkerFiguresShareOneSource"
        status: pass
    human_judgment: false
  - id: D2
    description: "The closing summary shows the same per-helper time and tool-call count the live line showed, read from one source (D-03)."
    requirement: "SHOW-02"
    verification:
      - kind: unit
        ref: "cmd/continue_worker_measurements_test.go#TestLiveAndSummaryWorkerFiguresShareOneSource"
        status: pass
      - kind: unit
        ref: "cmd/continue_worker_measurements_test.go#TestChatPathShowsWorkerMeasurements"
        status: pass
    human_judgment: false
  - id: D3
    description: "A figure that was never measured is shown as not reported, never as zero and never as a guess (Phase 196 D-01)."
    verification:
      - kind: unit
        ref: "cmd/continue_worker_measurements_test.go#TestUnmeasuredWorkerFiguresRenderAsNotReported"
        status: pass
    human_judgment: false
  - id: D4
    description: "The tool-call count survives being written to and read back from the saved result file, so the chat path shows it too (SHOW-02)."
    requirement: "SHOW-02"
    verification:
      - kind: unit
        ref: "cmd/continue_worker_measurements_test.go#TestWorkerFlowStepCarriesToolCount"
        status: pass
      - kind: unit
        ref: "cmd/continue_worker_measurements_test.go#TestChatPathShowsWorkerMeasurements"
        status: pass
    human_judgment: false
  - id: D5
    description: "The token-usage figure stays unserialized; this plan does not give WorkerUsage a JSON tag."
    verification:
      - kind: unit
        ref: "cmd/continue_worker_measurements_test.go#TestWorkerUsageStaysUnserialized"
        status: pass
    human_judgment: false

duration: 55min
completed: 2026-08-29
status: complete
---

# Phase 198 Plan 5: One Formatter for Worker Time and Tool Calls Summary

**Every reviewer helper's finishing line and the closing summary now show the same measured duration and tool-call count from one shared formatter, and an unmeasured figure reads "not reported" instead of a fabricated zero.**

## Performance

- **Duration:** 55 min (approx, not separately timestamped by the executor harness)
- **Tasks:** 3 completed
- **Files modified:** 7 (1 created, 6 modified)

## Accomplishments

- `codexContinueWorkerFlowStep` gained `ToolCount` plus `DurationReported`/`ToolCountReported` boolean flags, populated from the trusted in-process `WorkerResult` in the review-worker flow loop (`cmd/codex_continue.go`) — the exact same place `Duration` was already being carried. `Usage` remains `json:"-"`; `ToolCount` is a plain measured int, not subject to the outside-caller-assertion risk that keeps `Usage` unserialized.
- `workerMeasurementFigures` (`cmd/codex_visuals.go`) is now the single formatter turning a worker's duration and tool-call count into text — "3m 10s, 14 tool calls" when both are measured, spend's existing "not reported" wording when either is missing, and "0 tool calls" (never "not reported") for a worker that genuinely made zero calls.
- The live finishing line (`emitCodexDispatchWorkerFinished`, shared by build and continue review dispatch) and the continue worker-flow summary render (both the typed-struct and JSON-round-tripped branches) now call that one formatter — an AST-based test proves no other function in either touched file formats "tool call" text itself.
- `TestChatPathShowsWorkerMeasurements` proves the whole chain end to end: a continue-finalize result's worker flow, round-tripped through JSON the way a real completion file is, renders identically to the direct in-process path and carries every worker's measurement fragment — including the honest "not reported" case for an unmeasured worker.

## Task Commits

Each task was committed atomically:

1. **Task 1: Carry the tool-call count on the continue worker step** — `4203e323` (feat)
2. **Task 2: One formatter for the figures, used by the live line and the summary** — `d6a66f6d` (feat)
3. **Task 3: Prove the figures survive the chat path end to end** — `aab35176` (test)

## Files Created/Modified

- `cmd/codex_continue.go` — `codexContinueWorkerFlowStep` gains `ToolCount`, `DurationReported`, `ToolCountReported`; populated alongside the existing `Duration` assignment in the review-worker flow loop.
- `cmd/codex_visuals.go` — new `workerMeasurementFigures` and `formatWorkerMeasuredDuration`; `renderContinueWorkerFlowDetail` gains a `measured` parameter rendering one `└── measured: ...` line per review-stage worker (typed and map branches both updated).
- `cmd/codex_build_progress.go` — `emitCodexDispatchWorkerFinished` builds its parenthetical from `workerMeasurementFigures` instead of formatting `%.1fs` inline, and now passes `ToolCount` through.
- `cmd/continue_worker_measurements_test.go` (new) — all five named tests: `TestWorkerFlowStepCarriesToolCount`, `TestWorkerUsageStaysUnserialized`, `TestLiveAndSummaryWorkerFiguresShareOneSource` (plus its AST sub-test), `TestUnmeasuredWorkerFiguresRenderAsNotReported`, `TestChatPathShowsWorkerMeasurements`.
- `cmd/testdata/golden_build.txt`, `cmd/testdata/golden_continue.txt`, `cmd/testdata/golden_plan.txt` — regenerated via `-update-golden` for the new finishing-line shape; inspected diff-by-diff to confirm only the intended lines changed.

## Decisions Made

See `key-decisions` in frontmatter: the "reported" distinction is scoped to WorkerResult presence (not the worker's own raw-claims key presence); the summary's `measured:` line is scoped to genuine review-stage worker dispatches, excluding synthetic bookkeeping steps; the external/wrapper continue lane's completion-packet schema (`codexContinueExternalDispatch`) was left unwired for `ToolCount`, out of this plan's file scope; and the finishing-line format intentionally changed to the parenthetical shape from D-03's example, requiring golden fixture regeneration.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Measurement line initially leaked onto synthetic "review wave skipped" bookkeeping step**
- **Found during:** Task 2, first golden-fixture regeneration pass
- **Issue:** Gating the new `measured:` detail line on `Stage == "review"` alone also matched `continueReviewSkippedFlowStep`'s synthetic placeholder (`Caste: "system"`, used when no review workers ran at all), producing a spurious `└── measured: not reported` line under a step that was never a dispatched worker.
- **Fix:** Narrowed the gate to `Stage == "review" && Caste != "system"` in both the typed and map render branches, matching exactly the flow Task 1 populates `ToolCount` for.
- **Files modified:** `cmd/codex_visuals.go`
- **Verification:** `go test ./cmd -run TestGoldenContinueVisualOutput -count=1` passes without `-update-golden`; golden diff confirmed to contain only the intended finishing-line change.
- **Committed in:** `d6a66f6d` (Task 2 commit)

**2. [Rule 3 - Blocking] Golden fixtures needed regeneration for the intentional finishing-line format change**
- **Found during:** Task 2, after wiring `workerMeasurementFigures` into `emitCodexDispatchWorkerFinished`
- **Issue:** `TestGoldenBuildVisualOutput`, `TestGoldenPlanVisualOutput`, and `TestGoldenContinueVisualOutput` diff byte-for-byte against checked-in golden files; the finishing-line shape genuinely changed (from `completed 0.0s` to `completed (0.0s, 0 tool calls)`), by design, per D-03.
- **Fix:** Regenerated all three golden fixtures with `-update-golden`, inspected each diff line-by-line to confirm it contained only the expected finishing-line change (and, before the fix above, the spurious skipped-step line).
- **Files modified:** `cmd/testdata/golden_build.txt`, `cmd/testdata/golden_continue.txt`, `cmd/testdata/golden_plan.txt`
- **Verification:** All three golden tests pass without `-update-golden`; full `go test ./cmd/...` green.
- **Committed in:** `d6a66f6d` (Task 2 commit)

---

**Total deviations:** 2 auto-fixed (1 bug in initial scope of the new render line, 1 expected golden-fixture regeneration for an intentional format change). **Impact:** Both fixes were necessary for the new behavior to be correct and for the existing golden-fixture test suite to pass; no scope creep beyond this plan's stated files.

## Issues Encountered

None beyond the two deviations documented above.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- `workerMeasurementFigures` and the `DurationReported`/`ToolCountReported` pattern are now available for any later plan in this phase that needs a measured-vs-not-reported figure pair.
- The external/wrapper continue lane (`codexContinueExternalDispatch`) does not yet carry `ToolCount` from a wrapper-submitted completion packet — if a future plan needs that lane's `worker_flow` entries to show a real tool-call count (rather than "not reported"), it will need to add a `tool_count` field to that struct in `cmd/codex_continue_plan.go` and thread it through `mergeExternalContinueResults`.
- No blockers for subsequent plans in phase 198.

## Self-Check: PASSED

- Verified all created/modified files exist on disk: `cmd/codex_continue.go`, `cmd/codex_visuals.go`, `cmd/codex_build_progress.go`, `cmd/continue_worker_measurements_test.go`, `cmd/testdata/golden_build.txt`, `cmd/testdata/golden_continue.txt`, `cmd/testdata/golden_plan.txt`.
- Verified commits `4203e323`, `d6a66f6d`, `aab35176` exist in git log.
- Re-ran the plan-level `<verification>` command set: `go build ./...`, `go vet ./cmd`, and the full named test list (`TestWorkerFlowStepCarriesToolCount`, `TestWorkerUsageStaysUnserialized`, `TestLiveAndSummaryWorkerFiguresShareOneSource`, `TestUnmeasuredWorkerFiguresRenderAsNotReported`, `TestChatPathShowsWorkerMeasurements`, `TestWrapperPathRendersSameCeremonyAsDirectPath`, `TestHumanDisplaysUseHeadedSectionsNotMachineTables`) all exit 0.
- Re-ran full `go test ./cmd/...` (~240s): all green, no regressions.

---
*Phase: 198-put-the-thrown-away-data-back-on-screen*
*Completed: 2026-08-29*
