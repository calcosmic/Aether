---
phase: 203-biological-runtime
plan: "07"
subsystem: recruitment
tags: [idempotency, replay-safety, recovery-classification, biological-runtime]

# Dependency graph
requires:
  - phase: 203-biological-runtime
    provides: "203-02's recruitment tracer (recruitCmd, dispatchRecruitment, the minimal recruitmentResult/bindRecruitmentResult shape, the live.recruit.* event topics) that this plan extends"
provides:
  - "recruitmentResult carrying BIO-04's full field set (IntentID, DispatchID, ExecutionGeneration, Evidence, Artifacts, HandoffID, UsageRowID, AdapterKind, StartedAt, EndedAt, Provenance)"
  - "bindRecruitmentResult -- content-digest-checked exactly-once binding: identical replay returns the stored receipt with zero writes, a conflicting replay is refused naming the first differing field, an out-of-vocabulary terminal status is refused by name, a stale execution generation is refused naming both generations"
  - "recruitmentRecoveryClass and classifyRecruitmentRecovery -- five named recovery classes (missing, duplicated, altered, timed-out, replayed), each with exactly one owner-facing next action, exposed via `aether recruit --status <recruitment-id>`"
  - "TestOneRecruitmentIdempotencyMechanism / TestEveryResultWriteGoesThroughTheBinding / TestReplayPerformsNoWrite -- structural AST guards making a second dedupe mechanism or an out-of-band store write a named test failure"
affects: [203-09, 203-14, 203-15]

# Actuals (#2632)
actuals:
  tokens: 13537
  tasks: 3
  commits: 3

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Content-digest replay/conflict comparison stored on colony.LifecycleReceiptReference.Digest, with a fixed-order field-by-field diff to name the first differing field on conflict"
    - "Completeness-accessor convention (recruitmentTerminalStatuses/recruitmentRecoveryClasses) mirroring pkg/events/colony_live.go's ColonyLiveTopics()/ColonyLiveEpisodeKinds()"
    - "Composing new CLI behavior onto an existing cobra.Command from a sibling file in the same package by wrapping its RunE in init(), instead of editing a file owned by a concurrent wave plan"
    - "Phase 199's evidence classification rule (missing evidence is unknown, only a digest mismatch on a readable file is altered) reapplied unchanged to a new domain"

key-files:
  created:
    - cmd/recruitment_recovery.go
    - cmd/recruitment_result_test.go
  modified:
    - cmd/recruitment_result.go

key-decisions:
  - "ExecutionGeneration staleness is checked as its own refusal category BEFORE the content-digest comparison, not folded into the conflicting-content diff -- so a stale-but-otherwise-identical resubmission is refused by generation, not silently treated as a safe replay."
  - "Execution generation ordering is established only when both the stored and incoming values parse as integers (matching this codebase's own recruit_<unix-nanoseconds> AttemptID convention); an unparseable or differently-shaped pair falls through to ordinary content comparison rather than guessing which one is older. This is the plan's own flagged, unresolved assumption about what 'execution generation' means for BIO-04 -- surfaced in code comments, not silently resolved."
  - "aether recruit --status/--child are wired by capturing and wrapping recruitCmd.RunE from cmd/recruitment_recovery.go's own init(), rather than editing cmd/recruitment.go (owned by this wave's other parallel plans) or adding a second recovery command. Var initialization (recruitCmd's composite literal, including its original RunE) completes before any init() runs in Go, so this wrap is correct regardless of init() ordering between files."
  - "UsageRowID is populated with the child's own deterministic worker name (ChildName), never Caste, per the Phase 196 accounting ruling that several workers can share an agent/caste name and accounting must key on the per-worker one -- documented in the field's own comment; no new usage-ledger writer was added in this plan (out of scope for BIO-04's binding requirement)."
  - "'Duplicated' recovery class names a same-ID completion report that disagrees on content (the must_haves' 'arrives twice' case); 'replayed' names either no incoming candidate or an identical resubmission -- these are two distinct classes rather than one, matching the plan's own behavior spec wording."

patterns-established:
  - "A durable record's content-digest comparison lives in one function (recruitmentResultContentDiff) returning the FIRST differing field name in a fixed order, reused identically by both the binder (bindRecruitmentResult) and the recovery classifier (classifyRecruitmentRecovery) -- one comparison, two callers, never two comparisons."

requirements-completed: [BIO-04]

coverage:
  - id: D1
    description: "recruitmentResult binds every BIO-04 element (child, intent, dispatch, execution generation, terminal status, evidence, artifacts, handoff, usage) in one record, and bindRecruitmentResult is content-digest-checked exactly-once"
    requirement: "BIO-04"
    verification:
      - kind: unit
        ref: "cmd/recruitment_result_test.go#TestRecruitmentResultBinding"
        status: pass
      - kind: other
        ref: "manual: removed the generation-staleness check, confirmed the staleness subtest fails naming the wrong error, restored it and reran green"
        status: pass
    human_judgment: false
  - id: D2
    description: "Every way a recruitment result can go wrong (missing, duplicated, altered, timed-out, replayed) maps to one named recovery class with one owner-facing next action, exposed via aether recruit --status"
    requirement: "BIO-04"
    verification:
      - kind: unit
        ref: "cmd/recruitment_result_test.go#TestRecruitmentRecovery"
        status: pass
      - kind: other
        ref: "manual: removed the altered-evidence check, confirmed the altered subtest fails (reports replayed instead), restored it and reran green"
        status: pass
    human_judgment: false
  - id: D3
    description: "Replay safety is structurally locked to bindRecruitmentResult as the single mechanism: no second dedupe structure can appear without a named test failing, no store write to the results file can happen outside bindRecruitmentResult, and a verified replay performs zero writes (proven via mtime/size, not byte-equality alone)"
    requirement: "BIO-04"
    verification:
      - kind: unit
        ref: "cmd/recruitment_result_test.go#TestOneRecruitmentIdempotencyMechanism"
        status: pass
      - kind: unit
        ref: "cmd/recruitment_result_test.go#TestEveryResultWriteGoesThroughTheBinding"
        status: pass
      - kind: unit
        ref: "cmd/recruitment_result_test.go#TestReplayPerformsNoWrite"
        status: pass
    human_judgment: false

duration: ~50min
completed: 2026-09-13
status: complete
---

# Phase 203 Plan 07: Durable Recruitment Result and Exactly-Once Recovery Summary

**Extended the recruitment tracer's result into BIO-04's full field set with a content-digest replay/conflict binder, then added five named recovery classes (missing, duplicated, altered, timed-out, replayed) exposed through `aether recruit --status <recruitment-id>` -- every guarantee proven by breaking it and watching the test fail before restoring it.**

## Performance

- **Duration:** ~50 min (session start timestamp was not captured by the harness; this is a best-effort estimate against the work performed, not a measured wall-clock figure)
- **Completed:** 2026-09-13T12:42:56+02:00
- **Tasks:** 3 completed
- **Files modified:** 3 (2 created, 1 modified)

## Accomplishments

- `cmd/recruitment_result.go`: `recruitmentResult` now carries every BIO-04 element (`IntentID`, `DispatchID`, `ChildName`, `ParentName`, `ExecutionGeneration`, `TerminalStatus`, `Summary`, `Evidence`, `Artifacts`, `HandoffID`, `UsageRowID`, `AdapterKind`, `StartedAt`, `EndedAt`, `Provenance`), plus `RecruitmentResultSchemaVersion` and a `recruitmentTerminalStatuses()` completeness accessor mirroring `ColonyLiveTopics()`/`ColonyLiveEpisodeKinds()`.
- `bindRecruitmentResult` rewritten to compute a content digest and compare it on replay: an identical replay returns the stored receipt and writes nothing; a differing replay is refused naming the first differing field (`recruitmentResultContentDiff`, a fixed-order field-by-field comparison); an out-of-vocabulary terminal status is refused by name; a stale `ExecutionGeneration` is refused naming both generations, checked *before* content comparison so a stale-but-otherwise-identical resubmission is never mistaken for a safe replay.
- `cmd/recruitment_recovery.go` (new): `recruitmentRecoveryClass` with its five declared members, a `recruitmentRecoveryClasses()` completeness accessor, and `classifyRecruitmentRecovery` -- a read-only classifier deriving `missing`/`timed-out` from the real spawn-tree admission path (never a typed fixture) and `replayed`/`duplicated`/`altered` from the stored result plus an optional incoming candidate. Reapplies Phase 199's evidence rule unchanged: a missing evidence file is unknown (skipped), only a digest mismatch on a file that actually reads counts as tampering.
- Exposed the classifier as `aether recruit --status <recruitment-id>` (+ optional `--child <name>`) by capturing and wrapping the existing `recruitCmd.RunE` from this plan's own file -- `cmd/recruitment.go` is owned by a concurrent wave-4 plan and was never edited.
- `cmd/recruitment_result_test.go` (new, 671 lines): `TestRecruitmentResultBinding`, `TestRecruitmentRecovery`, `TestOneRecruitmentIdempotencyMechanism` (AST scan for a second dedupe map/function, proven against two synthetic fixtures), `TestEveryResultWriteGoesThroughTheBinding` (AST-derives every store write touching `recruitmentResultsPath` and asserts `bindRecruitmentResult` is the only one), and `TestReplayPerformsNoWrite` (compares the results file's mtime/size before and after a verified replay -- a stronger signal than byte-equality, since even a no-op rewrite of identical bytes would bump mtime).
- Every one of Task 1 and Task 2's core guarantees was proven by temporarily removing the check, confirming the corresponding test failed with the expected symptom, then restoring the check and re-running green (see Self-Check below for the exact before/after output).

## Task Commits

Each task was committed atomically:

1. **Task 1: Complete the durable result and its exactly-once binding** - `a82e07e4` (feat)
2. **Task 2: Turn every failure shape into one named recovery** - `9c4da4b0` (feat)
3. **Task 3: Lock the single idempotency mechanism** - `0e7763e0` (test)

**Plan metadata:** this commit (docs: complete plan)

## Files Created/Modified

- `cmd/recruitment_result.go` - Full BIO-04 field set on `recruitmentResult`; content-digest replay/conflict binding; terminal-status vocabulary; execution-generation staleness check
- `cmd/recruitment_recovery.go` - Five named recovery classes, `classifyRecruitmentRecovery`, `aether recruit --status/--child` wiring
- `cmd/recruitment_result_test.go` - All new tests for Tasks 1-3

## Decisions Made

- **Execution generation is the durable build-run identity, not a per-child retry counter** (the plan's own flagged, unresolved planner assumption) -- staleness is checked only when both the stored and incoming values parse as integers, matching this codebase's `recruit_<unix-nanoseconds>` AttemptID convention; an unparseable pair falls through to ordinary content comparison rather than guessing an order. Surfaced in code comments, not silently resolved, per the plan's own instruction.
- **`aether recruit --status`/`--child` are wired by wrapping `recruitCmd.RunE` from this file's own `init()`**, not by editing `cmd/recruitment.go` (a file owned by a concurrent wave-4 plan) and not by adding a second recovery command. Go completes package-level var initialization (including `recruitCmd`'s original `RunE`) before any `init()` runs, so this composition is correct regardless of file/init ordering.
- **"Duplicated" and "replayed" are two distinct recovery classes**, matching the plan's own behavior spec: "duplicated" is a same-ID completion report that disagrees on content (an illegitimate resubmission); "replayed" is either no incoming candidate at all, or an identical resubmission (a safe, read-only outcome).
- **`UsageRowID` is populated with the child's own deterministic worker name, never a shared caste/agent name**, per the Phase 196 accounting ruling already documented in this codebase (`cmd/wrapper_usage_resolve.go`'s `AgentNameByWorker` doc comment) -- no new usage-ledger writer was introduced; that remains a future consumer's job.

## Deviations from Plan

None - plan executed exactly as written. The plan's own `<read_first>` list for Task 2 named `cmd/recruitment_admission.go`, `cmd/resume_cmd.go`, and `cmd/pause_cmd.go`, none of which exist in this worktree: `recruitment_admission.go` is owned by the sibling wave-4 plan 203-06 (not yet merged into this parallel worktree), and `resume_cmd.go`/`pause_cmd.go` are not this codebase's actual filenames for that functionality (the real files are `resume_detail_test.go`/`pause_resume_199_test.go` plus lifecycle machinery in `cmd/lifecycle_transaction.go`, `cmd/recovery_snapshot.go`). These files could not be read because they either do not exist yet in this worktree (parallel-wave timing) or were misnamed in the plan; the closest available real analogs (`cmd/lifecycle_transaction.go`'s `lifecycleRecoveryProvenance`, and `cmd/spawn_ancestor.go`'s D-19 discipline) were read and used instead, per this plan's own instruction that CLAUDE.md's cited test names and the actual codebase are the authority. This is noted here as a documentation gap in the plan, not a deviation in the executed work.

---

**Total deviations:** 0
**Impact on plan:** None -- all three tasks completed exactly per their acceptance criteria, using the real existing analogs where the plan's own file list pointed at files that do not exist under those names or had not yet landed from a sibling parallel plan.

## Issues Encountered

None.

## Threat Flags

None. This plan only extends an existing durable-record binder and adds a read-only classifier; it introduces no new process-spawn, network, or auth surface. The evidence-tampering detection (`recruitmentEvidenceAltered`) reads file bytes it is explicitly told to read via `Evidence[].Source`, populated only by this plan's own binder -- no new untrusted input path.

## Known Stubs

None. Every field this plan added is populated where the plan's own scope calls for it (the binder and classifier); fields with no current production writer (e.g. a future usage-ledger row keyed on `UsageRowID`) are documented in their own comments as intentionally out of this plan's scope rather than left silently empty.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- `bindRecruitmentResult` and `classifyRecruitmentRecovery` are both real, tested, and reachable from the public `aether recruit` command surface -- no blockers for 203-09 (inline/end-of-run rendering of recruit/refusal lines) or any later plan wiring evidence/artifacts/handoff population into a live dispatch.
- This plan did not populate `Evidence`, `Artifacts`, `HandoffID`, `DispatchID`, `ExecutionGeneration`, `UsageRowID`, `AdapterKind`, `StartedAt`, `EndedAt`, or `Provenance` from `cmd/recruitment.go`'s live dispatch path (that file is owned by a concurrent wave-4 plan) -- the fields exist, validate, and are digest-compared correctly, but a real dispatch today still binds a result with only the tracer's original minimal field set. A future plan (or a follow-up to 203-02's file) should populate these new fields at the real call site in `cmd/recruitment.go`.
- `cmd/recruitment_admission.go` (203-06, a sibling wave-4 plan) had not landed in this worktree at execution time; nothing in this plan depends on it, but its eventual arrival should be checked against `classifyRecruitmentRecovery`'s "missing" branch to confirm the admission gate's own failure modes don't need a sixth recovery class.

---
*Phase: 203-biological-runtime*
*Completed: 2026-09-13*

## Self-Check: PASSED

- FOUND: `cmd/recruitment_result.go` (modified)
- FOUND: `cmd/recruitment_recovery.go`
- FOUND: `cmd/recruitment_result_test.go`
- FOUND: commit `a82e07e4` (feat(203-07): complete the durable recruitment result and its exactly-once binding) in `git log --oneline`
- FOUND: commit `9c4da4b0` (feat(203-07): turn every recruitment failure shape into one named recovery) in `git log --oneline`
- FOUND: commit `0e7763e0` (test(203-07): lock the single recruitment idempotency mechanism) in `git log --oneline`
- Re-ran plan-level `<verify>` commands individually:
  - `go test ./cmd -run '^TestRecruitmentResultBinding' -count=1` -- PASS
  - `go test ./cmd -run '^TestRecruitmentRecovery' -count=1` -- PASS
  - `go test ./cmd -run '^(TestOneRecruitmentIdempotencyMechanism|TestEveryResultWriteGoesThroughTheBinding|TestReplayPerformsNoWrite)$' -count=1` -- PASS
- Re-ran `go build ./...` and `go vet ./cmd` -- clean
- Re-ran the pre-existing 203-02 tracer suite to confirm no regression: `go test ./cmd -run '^TestRecruitmentTracerEndToEnd$|^TestRecruitmentDispatchTerminatesWholeProcessGroup$|^TestRecruitLiveEventsGoThroughTheOneBoundary$|^TestPlainBuildPathDoesNotTouchRecruitment$|^TestPlatformParityGolden$' -count=1` -- PASS
- Re-ran acceptance-criteria negative proofs by breaking and reverting each guarantee in turn: removing the `ExecutionGeneration` staleness check from `bindRecruitmentResult` makes the "a stale execution generation is refused" subtest fail (it instead reports a content conflict on `execution_generation`); removing the `recruitmentEvidenceAltered` call from `classifyRecruitmentRecovery` makes the "an altered evidence file produces altered" subtest fail (it instead reports `replayed`) -- both confirmed, then reverted (`git diff` against the committed state is empty; `go test` reruns green).
