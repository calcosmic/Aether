---
phase: 189-complete-worker-contract
plan: 02
subsystem: continue-orchestration
tags: [worker-dispatch, handoff-schema, codex, continue-brief, worker-contract, wrapper-parity]

# Dependency graph
requires:
  - phase: 189-01
    provides: codex.HandoffFieldsSummary, the one canonical handoff-schema constant, and the composeBuildManifestBrief wrapper-external-only placement precedent
provides:
  - codexContinuePlanManifest.ContextCapsule and .PheromoneSection, manifest-level fields resolved once per plan-only continue run (mirroring codexBuildManifest.ContextCapsule)
  - continueExternalBriefWithHandoffSchema, appending the handoff/return schema to every external continue dispatch's Brief (watcher and every reviewer) without duplicating it on continue's native-Codex dispatch path
  - All three canonical continue wrapper files (.claude/commands/ant/continue.md, its flat mirror, .opencode/commands/ant/continue.md) instructing delivery of continue_manifest.context_capsule, .pheromone_section, and dispatch.skill_section to every spawned reviewer
  - .codex/CODEX.md's Skills and Pheromone Signals sections naming `continue` alongside `build`, `colonize`, `plan`
affects: [190 (continue-path --print-brief zero-duplication cleanup builds on this plan's single-home placement)]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Manifest-level fields for colony-wide content, resolved once (codexContinuePlanManifest.ContextCapsule/.PheromoneSection mirrors codexBuildManifest.ContextCapsule) -- never per-dispatch"
    - "Wrapper-external-only content is appended at the composition call site (plannedExternalContinueDispatches), never inside the shared brief renderer (renderCodexContinueWatcherBrief/renderCodexContinueReviewBrief) both the external and native-Codex paths call"

key-files:
  created: []
  modified:
    - cmd/codex_continue_plan.go
    - cmd/codex_continue_plan_test.go
    - cmd/continue_wrapper_ceremony_test.go
    - .claude/commands/ant/continue.md
    - .claude/commands/ant-continue.md
    - .opencode/commands/ant/continue.md
    - .codex/CODEX.md

key-decisions:
  - "Followed CONTEXT.md D-11/D-12 exactly: ContextCapsule and PheromoneSection are manifest-level fields on codexContinuePlanManifest, resolved once via resolveCodexWorkerContext()/resolvePheromoneSection() inside runCodexContinuePlanOnly -- never per-dispatch fields on codexContinueExternalDispatch, matching build's existing split (SkillSection stays per-dispatch, capsule/pheromone are manifest-level)"
  - "Followed D-06 exactly: the handoff-schema note is appended in plannedExternalContinueDispatches (wrapper-external-only), never inside renderCodexContinueWatcherBrief/renderCodexContinueReviewBrief themselves, because both render functions are shared with continue's native-Codex dispatch path (plannedContinueReviewDispatches/plannedContinueWatcherDispatch), which already states the schema via a separate AssembleHostedPrompt+renderResponseContract channel -- embedding it in the shared renderer would have duplicated it for every native-Codex continue worker"
  - "Reused codex.HandoffFieldsSummary (189-01's exported constant) by symbol via continueExternalBriefWithHandoffSchema -- never hand-copied a third sentence, exactly as CONTEXT.md instructed"
  - "Followed D-09/D-13 exactly: verified by diff before editing that the real byte-identical triplet is .claude/commands/ant/continue.md, .claude/commands/ant-continue.md, and .opencode/commands/ant/continue.md -- not .codex/CODEX.md, which is not part of either parity test and gets only the two-sentence accuracy fix (D-10)"
  - "TestContinueExternalDispatchBriefsStateHandoffSchemaOnceNotOnNativePath calls plannedExternalContinueDispatches with skipWatchers=false and no explicit Queen caste override, letting the real deterministic queenOrchestrate path decide the team, rather than hand-picking castes -- this produced both a watcher AND a probe reviewer dispatch on the test's real source-code workspace, so the 'every returned dispatch's Brief contains the schema' assertion genuinely exercises both wrapping call sites, not just one"

patterns-established: []

requirements-completed: []

# Metrics
duration: ~25min
completed: 2026-08-20
---

# Phase 189 Plan 02: Complete Worker Contract (Reviewer Parity) Summary

**Continue's external (Claude Code / OpenCode) reviewer and watcher dispatches now carry the same colony-wide context capsule and pheromone signals build's external dispatches always have, and every external continue brief states the exact handoff schema the finalizer enforces -- via the same canonical `codex.HandoffFieldsSummary` constant 189-01 exported, delivered without duplicating it for continue's native-Codex reviewers.**

## Performance

- **Duration:** ~25 min
- **Started:** 2026-08-20T06:10:00Z (approx.)
- **Completed:** 2026-08-20T06:40:00Z (approx.)
- **Tasks:** 2 (Task 1 TDD, RED→GREEN; Task 2 non-TDD, doc + Go test parity)
- **Files modified:** 7

## Accomplishments

- Confirmed the gap described in the brief was real (not already handled): `codexContinuePlanManifest` and `codexContinueExternalDispatch` had neither a capsule field nor a pheromone field anywhere, and `.claude/commands/ant/continue.md`'s entire delivery instruction was "Pass each dispatch's runtime-provided `brief` verbatim" — no capsule, no skill, no pheromone, no handoff schema.
- `codexContinuePlanManifest.ContextCapsule` and `.PheromoneSection` are now resolved exactly once per plan-only continue run, mirroring how build already treats its own manifest-level capsule — proven by a test that seeds a real pheromone signal and asserts its distinctive text reaches the manifest field, plus a reflection-based check that `codexContinueExternalDispatch` never grew its own copy of either field.
- Every external continue dispatch's `Brief` (the watcher's and every reviewer's) now states the handoff schema via `continueExternalBriefWithHandoffSchema`, referencing `codex.HandoffFieldsSummary` by symbol — proven against a *real* Queen-orchestrated dispatch set (watcher + probe, not hand-picked), while `renderCodexContinueWatcherBrief`/`renderCodexContinueReviewBrief`'s own raw output (what continue's native-Codex path feeds as `TaskBrief`) was proven to NOT contain it, so a native-Codex continue worker never sees the schema twice.
- All three canonical continue wrapper files instruct reading `continue_manifest.context_capsule` and `continue_manifest.pheromone_section` once (not per-dispatch) and prepending both verbatim ahead of each dispatch's brief, then appending `dispatch.skill_section` when present — mirroring `build.md`'s established formula word-for-word in structure. Verified byte-identical across all three files both before and after the edit (`diff` exit 0 on both pairs).
- `.codex/CODEX.md`'s Skills and Pheromone Signals sections now name `continue` alongside `build`, `colonize`, `plan` — a two-sentence accuracy fix, since Codex's native dispatch path already injects both and the docs simply omitted it.

## Task Commits

Each task was committed atomically (Task 1 is TDD: test → feat, RED → GREEN):

1. **Task 1 RED: failing tests for continue capsule/pheromone/handoff-schema parity** - `7c588e2b` (test)
2. **Task 1 GREEN: carry capsule, pheromone and handoff schema on continue's external dispatches** - `45f6cfb2` (feat)
3. **Task 2: instruct continue wrappers to deliver capsule, pheromone and skill content** - `f2094439` (feat)

**Plan metadata:** committed separately after this SUMMARY (docs: complete plan)

## Files Created/Modified

- `cmd/codex_continue_plan.go` - Added `ContextCapsule`/`PheromoneSection` fields to `codexContinuePlanManifest`, resolved once in `runCodexContinuePlanOnly`; new `continueExternalBriefWithHandoffSchema` helper wraps both `Brief:` assignments in `plannedExternalContinueDispatches`
- `cmd/codex_continue_plan_test.go` - Two new invariant tests: `TestContinuePlanOnlyManifestCarriesCapsuleAndPheromoneSection`, `TestContinueExternalDispatchBriefsStateHandoffSchemaOnceNotOnNativePath`
- `cmd/continue_wrapper_ceremony_test.go` - New test `TestContinueWrapperInstructsCapsuleAndPheromoneDelivery`
- `.claude/commands/ant/continue.md` - "Heavy External Review" section's "Reads:" line, the prose line above the spawn steps, and step 5 now name `continue_manifest.context_capsule`, `continue_manifest.pheromone_section`, and `dispatch.skill_section`
- `.claude/commands/ant-continue.md` - Copied byte-identically from the canonical source above
- `.opencode/commands/ant/continue.md` - Copied byte-identically from the canonical source above
- `.codex/CODEX.md` - Skills and Pheromone Signals sentences extended to name `continue`

## Decisions Made

See `key-decisions` in the frontmatter above. In summary: manifest-level "resolve once" placement for capsule/pheromone (D-11/D-12), wrapper-external-only placement for the handoff-schema append to avoid native-Codex duplication (D-06), reuse of 189-01's exported constant (no third hand-copied sentence), and D-09's corrected triplet (the flat mirror + OpenCode file, not `.codex/CODEX.md`, which only needed the accuracy fix).

## Deviations from Plan

None - plan executed exactly as written. Both tasks matched their `<action>` blocks precisely; no bugs, missing functionality, blockers, or architectural questions were encountered.

## Proof of Each New/Changed Test (fail-then-pass)

### `TestContinuePlanOnlyManifestCarriesCapsuleAndPheromoneSection` (Task 1)

**RED**, quoted verbatim (against the unfixed `codexContinuePlanManifest`, captured by reverting `cmd/codex_continue_plan.go` to its pre-fix state and running `go vet ./cmd/...` with the new test file present):
```
vet: cmd/codex_continue_plan_test.go:207:28: plan.ContextCapsule undefined (type codexContinuePlanManifest has no field or method ContextCapsule)
```
This is a genuine compile-time RED: the field the test asserts on did not exist at all before the fix, so there is no way for this test to pass vacuously.

**GREEN:** `--- PASS: TestContinuePlanOnlyManifestCarriesCapsuleAndPheromoneSection (0.70-0.80s)`

### `TestContinueExternalDispatchBriefsStateHandoffSchemaOnceNotOnNativePath` (Task 1)

To isolate this test's own RED independent of the struct-field compile error above, the struct fields and manifest assignment were kept but the two `Brief:` assignments in `plannedExternalContinueDispatches` were temporarily reverted to call `renderCodexContinueWatcherBrief`/`renderCodexContinueReviewBrief` directly (unwrapped), then the test was run alone:

**RED, quoted verbatim:**
```
codex_continue_plan_test.go:275: dispatch Watch-36 (caste watcher)'s Brief is missing handoff-schema substring "changed_files" -- a wrapper-spawned continue worker was never told the finalizer's schema:
    # Continue Verification
    ...
codex_continue_plan_test.go:275: dispatch Excavat-92 (caste probe)'s Brief is missing handoff-schema substring "changed_files" -- a wrapper-spawned continue worker was never told the finalizer's schema:
    # Continue Review
    ...
--- FAIL: TestContinueExternalDispatchBriefsStateHandoffSchemaOnceNotOnNativePath (0.02s)
```
Both a **watcher** dispatch and a **probe reviewer** dispatch were present and both failed — confirming the deterministic Queen orchestration on this test's real source-code fixture genuinely produces more than just the watcher, so the assertion exercises both wrapping call sites (watcher and reviewer), not one vacuously.

**GREEN:** `--- PASS: TestContinueExternalDispatchBriefsStateHandoffSchemaOnceNotOnNativePath (0.01-0.02s)`

### `TestContinueWrapperInstructsCapsuleAndPheromoneDelivery` (Task 2)

**RED**, quoted verbatim (captured by reverting `.claude/commands/ant/continue.md` alone to its pre-fix content and running the test):
```
continue_wrapper_ceremony_test.go:182: /Users/.../.claude/commands/ant/continue.md missing delivery instruction "continue_manifest.context_capsule" -- a wrapper-spawned continue reviewer would never receive it
continue_wrapper_ceremony_test.go:182: /Users/.../.claude/commands/ant/continue.md missing delivery instruction "continue_manifest.pheromone_section" -- a wrapper-spawned continue reviewer would never receive it
continue_wrapper_ceremony_test.go:182: /Users/.../.claude/commands/ant/continue.md missing delivery instruction "dispatch.skill_section" -- a wrapper-spawned continue reviewer would never receive it
--- FAIL: TestContinueWrapperInstructsCapsuleAndPheromoneDelivery (0.00s)
```

**GREEN:** `--- PASS: TestContinueWrapperInstructsCapsuleAndPheromoneDelivery (0.00s)`

## Verification

```
go build ./...                                     -> exit 0
go vet ./...                                        -> exit 0
go test ./cmd/ -run 'TestContinuePlanOnlyManifestCarriesCapsuleAndPheromoneSection|TestContinueExternalDispatchBriefsStateHandoffSchemaOnceNotOnNativePath|TestContinuePlanOnlyEnforcesCriterionEvidence|TestContinuePlanOnlyWithoutCriterionRequirementsLeavesBlockersUnchanged|TestContinueWrapperInstructsCapsuleAndPheromoneDelivery|TestContinueWrapperCeremonyContract|TestContinueWrapperStageSkeletonAndParity|TestLifecycleFlatMirrorsMatchCanonical' -v -count=1
                                                     -> all 8 named tests PASS (TestLifecycleFlatMirrorsMatchCanonical covers all 60 wrapper verbs, "continue" subtest included)
go test ./cmd/... -count=1 (full package)           -> ok, 310.9s, zero failures
go test ./... -race (full repo)                     -> ok, all 20 packages, zero failures
```

`go test ./... -race` (full repo, required by the parallel-execution instructions) full output:
```
?   github.com/calcosmic/Aether               [no test files]
ok  github.com/calcosmic/Aether/cmd            401.734s
?   github.com/calcosmic/Aether/cmd/aether     [no test files]
ok  github.com/calcosmic/Aether/pkg/agent          7.761s
ok  github.com/calcosmic/Aether/pkg/agent/curation  4.345s
ok  github.com/calcosmic/Aether/pkg/cache          4.737s
ok  github.com/calcosmic/Aether/pkg/codegraph      1.740s
ok  github.com/calcosmic/Aether/pkg/codex         25.613s
ok  github.com/calcosmic/Aether/pkg/colony         3.125s
ok  github.com/calcosmic/Aether/pkg/downloader     4.018s
ok  github.com/calcosmic/Aether/pkg/events         3.533s
ok  github.com/calcosmic/Aether/pkg/exchange       2.118s
ok  github.com/calcosmic/Aether/pkg/graph          3.218s
ok  github.com/calcosmic/Aether/pkg/learn          9.147s
ok  github.com/calcosmic/Aether/pkg/llm            3.380s
ok  github.com/calcosmic/Aether/pkg/memory         4.374s
ok  github.com/calcosmic/Aether/pkg/smoke          3.050s
ok  github.com/calcosmic/Aether/pkg/storage        3.479s
ok  github.com/calcosmic/Aether/pkg/terminal       2.838s
ok  github.com/calcosmic/Aether/pkg/trace          2.974s
```
Exit code 0, all 20 packages, zero failures, zero race warnings.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- No new duplication was created for Phase 190 to remove: the handoff schema lives in exactly one place per delivery path on the continue side too (native via `renderResponseContract`, wrapper-external via `continueExternalBriefWithHandoffSchema`), never both — proven by `TestContinueExternalDispatchBriefsStateHandoffSchemaOnceNotOnNativePath`. Capsule and pheromone content are manifest-level (single home), never repeated per-dispatch in the JSON payload.
- All three ROADMAP success criteria for Phase 189 are now covered: criterion 1 and criterion 2's build half by 189-01, criterion 2's continue half and criterion 3 fully by this plan.
- No blockers for Phase 190.

## Self-Check: PASSED

Files verified present on disk:
- FOUND: cmd/codex_continue_plan.go
- FOUND: cmd/codex_continue_plan_test.go
- FOUND: cmd/continue_wrapper_ceremony_test.go
- FOUND: .claude/commands/ant/continue.md
- FOUND: .claude/commands/ant-continue.md
- FOUND: .opencode/commands/ant/continue.md
- FOUND: .codex/CODEX.md

Commits verified present in `git log`:
- FOUND: 7c588e2b (test)
- FOUND: 45f6cfb2 (feat)
- FOUND: f2094439 (feat)

---
*Phase: 189-complete-worker-contract*
*Completed: 2026-08-20*
