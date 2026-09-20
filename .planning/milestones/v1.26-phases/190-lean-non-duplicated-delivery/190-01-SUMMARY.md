---
phase: 190-lean-non-duplicated-delivery
plan: 01
subsystem: cli/build-orchestration
tags: [go, cli, worker-dispatch, wrapper-contract, cobra]

# Dependency graph
requires:
  - phase: 189-complete-worker-contract
    provides: codex.HandoffFieldsSummary (the anchor text the duplication check counts occurrences of) and the handoff-schema-once-not-twice precedent this plan extends to pheromones/headings
provides:
  - "aether build <phase> --plan-only writes every dispatch's composed brief to a real file on disk and sets brief_path, blanking the inline brief once the write succeeds"
  - "aether build <phase> --print-brief fails with a specific error when any owned context section or the handoff schema is delivered more than once"
  - "Six prose surfaces (build.md wrapper triplet, build.yaml, command_guide.go, the aether-colony-build-cycle Codex skill) describe brief_path as the routine channel, not a co-equal alternative"
affects: [190-02-lean-non-duplicated-delivery, any future phase touching cmd/codex_build.go's dispatch/brief composition or cmd/build_print_brief.go's inspection path]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "A boolean flag distinguishes two still-valid caller contracts on one shared helper (writeBuildWorkerBriefFiles's clearInlineBrief), rather than forking the function or changing shared behavior for one caller at the other's expense"
    - "Extend an attribution-only registry (briefOwnedSections/splitBriefSections) into a counting check (duplicatedBriefSections) instead of replacing it"
    - "Anchor a duplication check on a shared constant's own content (codex.HandoffFieldsSummary[:40]), never a hand-copied substring of English prose"

key-files:
  created: []
  modified:
    - cmd/codex_build.go
    - cmd/codex_build_test.go
    - cmd/build_manifest_brief_composition_test.go
    - cmd/build_print_brief.go
    - cmd/build_print_brief_test.go
    - .claude/commands/ant/build.md
    - .claude/commands/ant-build.md
    - .opencode/commands/ant/build.md
    - .aether/commands/build.yaml
    - cmd/command_guide.go
    - .aether/skills/colony/aether-colony-build-cycle/SKILL.md

key-decisions:
  - "writeBuildWorkerBriefFiles extracted from writeCodexBuildArtifacts's inline loop; writeCodexBuildArtifacts calls it with clearInlineBrief=false (byte-identical, tested), runCodexBuildPlanOnlyWithOptions calls it with clearInlineBrief=true (D-01/D-02)"
  - "TestBuildPlanOnlyPrintsDispatchManifestWithoutMutatingState's worker_briefs assertion inverted deliberately per D-04 -- plan-only legitimately writes brief files as a side effect now, consistent with its other existing side effects"
  - "duplicatedBriefSections performs two independent counting passes (owned-heading occurrences, handoff-sentence-anchor occurrences) and is wired into printWorkerBriefs ahead of both the checklist and --full branches (D-05)"
  - "A pre-existing test, TestPlanOnlyManifestBriefCarriesPheromones, encoded the old inline-brief contract and was not named in 190-CONTEXT.md; fixed to read the pheromone content from the file brief_path names instead of the inline field, following the same fail-then-pass discipline as the named test inversion"
  - "Discovered but NOT fixed: pheromone signals and prior-worker handoffs currently render in both the manifest-level capsule and the per-dispatch brief for the plan-only path -- a real, proven duplicate the new detector correctly catches, but the fix is an architectural question (which composition layer owns which section for which caller) outside this plan's scope; logged in deferred-items.md"

requirements-completed: []

# Metrics
duration: ~75min
completed: 2026-08-20
---

# Phase 190 Plan 01: Lean, Non-Duplicated Delivery (build) Summary

**Plan-only build dispatches now write real brief files to disk and set `brief_path` instead of shipping the composed prompt inline twice; `--print-brief` fails when a context section repeats; six prose surfaces stopped describing the inline copy as a routine, co-equal alternative.**

## Performance

- **Duration:** ~75 min (estimate; no explicit start marker was captured before work began)
- **Completed:** 2026-08-20
- **Tasks:** 3 completed
- **Files modified:** 11 (3 Go source, 5 Go test, 3 wrapper markdown, 1 YAML, 1 Codex skill markdown -- command_guide.go counted once above under Task 3's Go-source edit)

## Accomplishments

- `aether build <phase> --plan-only` — the ONLY command the interactive wrapper is allowed to run
  — now writes every dispatch's composed brief to `build/phase-N/worker-briefs/{name}.md`, sets
  `brief_path` to that file, and removes the `brief` key entirely (not merely empties it) from both
  `result.dispatches[]` and `result.dispatch_manifest.dispatches[]`. Measured on a 7-dispatch test
  fixture: 6,848 bytes would have shipped inline per JSON representation; 364 bytes actually ship
  (the `brief_path` strings) — a 94.7% reduction per representation, doubled across both
  representations.
- The direct/native `aether build <phase>` dispatch path (no `--plan-only`) is provably unaffected:
  its two pre-existing tests pass with zero edits to those two test functions.
- `aether build <phase> --print-brief` now fails, naming the dispatch and the duplicated section(s),
  when any owned context section or the handoff schema is delivered more than once in a worker's
  assembled context — proven against a REAL, already-shipping duplicate (active pheromone signals
  rendering in both the manifest-level capsule and the per-dispatch brief), not just a hand-built
  string.
- All six prose surfaces describing brief delivery (the `build.md` wrapper triplet, `build.yaml`,
  `cmd/command_guide.go`, the `aether-colony-build-cycle` Codex skill) now say `brief_path` is the
  routine channel and inline `brief` is the documented rare fallback, not an equally-likely
  alternative carrying the same bytes.

## Task Commits

Each task was committed atomically:

1. **Task 1: Write plan-only worker briefs to disk; stop shipping the same bytes twice** - `2476f681` (feat)
2. **Task 2: Make `--print-brief` fail when a context section is delivered twice** - `ecb1161b` (feat)
3. **Task 3: Bring the six delivery-prose surfaces up to date with the new brief_path reality** - `5fadf214` (docs)

_Note: Task 1's commit also includes the fix to `TestPlanOnlyManifestBriefCarriesPheromones`, a
pre-existing test discovered during verification (not named in the plan) that encoded the same old
inline-brief contract Task 1 replaces — see Deviations below._

## Files Created/Modified

- `cmd/codex_build.go` - Adds `writeBuildWorkerBriefFiles` (the shared, flag-distinguished brief
  writer); `writeCodexBuildArtifacts` and `runCodexBuildPlanOnlyWithOptions` both converge on it
- `cmd/codex_build_test.go` - Inverts the stale `worker_briefs`-must-be-empty assertion; adds
  `TestBuildPlanOnlyManifestOmitsInlineBriefWhenBriefPathPresent`
- `cmd/build_manifest_brief_composition_test.go` - `TestPlanOnlyManifestBriefCarriesPheromones`
  updated to read steering content from the file `brief_path` names instead of the inline field
- `cmd/build_print_brief.go` - Adds `duplicatedBriefSections`, `handoffSectionDuplicationAnchor`,
  wires the check into `printWorkerBriefs`
- `cmd/build_print_brief_test.go` - `TestPrintBriefFailsOnDuplicatedSection` (unit-level),
  `TestPrintBriefCommandFailsWhenPrintWorkerBriefsFindsDuplication` and
  `TestPrintBriefCommandStaysCleanOnHealthyFixtureWithoutDuplication` (CLI-level)
- `.claude/commands/ant/build.md`, `.claude/commands/ant-build.md`, `.opencode/commands/ant/build.md`
  - Identical rewrite of the two brief_path/brief sentences; verified byte-identical after editing
- `.aether/commands/build.yaml` - Same rewrite in the YAML's terser style
- `cmd/command_guide.go` - Same rewrite inside `catalog["build"]`'s `PreSteps`
- `.aether/skills/colony/aether-colony-build-cycle/SKILL.md` - Same rewrite in the Codex skill

## Decisions Made

See `key-decisions` in frontmatter. The most consequential: `clearInlineBrief` is a boolean flag on
one shared helper rather than a behavior change to `writeCodexBuildArtifacts` itself, because two
existing, still-valid tests require the direct dispatch path's `.Brief` to survive on disk — changing
shared behavior to satisfy plan-only's new requirement would have silently broken that contract.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug in a pre-existing test, same class as the plan's own named inversion] Fixed `TestPlanOnlyManifestBriefCarriesPheromones`**
- **Found during:** Task 1, final verification sweep (a broader-than-prescribed regression pass)
- **Issue:** This test (`cmd/build_manifest_brief_composition_test.go`, not named anywhere in
  190-CONTEXT.md, 190-01-PLAN.md, or 190-PATTERNS.md) asserted every plan-only dispatch's inline
  `d["brief"]` field is non-empty and contains the active pheromone signal text. It encodes the exact
  OLD contract Task 1 intentionally replaces — once `clearInlineBrief` blanks `.Brief`, this test
  fails deterministically for every dispatch.
- **Fix:** Updated the test to resolve `brief_path` to the file on disk and read the pheromone
  content from there instead of the inline field, plus added an explicit assertion that no dispatch
  carries both `brief` and `brief_path` at once — preserving the test's original protective intent
  (steering reaches the worker) while matching the new delivery mechanism.
- **Fail-then-pass proof:** Pre-fix run:
  `build_manifest_brief_composition_test.go:75: 3 of 3 plan-only dispatches carry no brief —
  wrapper-spawned workers would get a bare task line again`. Post-fix run: PASS.
- **Files modified:** `cmd/build_manifest_brief_composition_test.go`
- **Committed in:** `2476f681` (Task 1 commit)

**2. [Rule 2 discovery, NOT auto-fixed — logged as deferred] Pheromone signals and prior-worker handoffs render twice for plan-only dispatches**
- **Found during:** Task 2, while proving `duplicatedBriefSections` against real (not synthetic) output
- **Issue:** `resolveCodexWorkerContext()` (the manifest-level capsule) and
  `composeBuildManifestBrief` (the per-dispatch brief) independently render their own
  `## Pheromone Signals` and `## Previous Worker Handoffs` sections from the same underlying data.
  Since the capsule is prepended once ahead of every brief, a wrapper-spawned worker sees each
  twice whenever either is active.
- **Why not fixed:** Not named by 190-CONTEXT.md's criterion 3 (which scoped the double-injection
  fix to the TS-host hive channel specifically, owned by plan 190-02). Fixing this one requires
  `composeBuildManifestBrief` to know whether a capsule will also be prepended (plan-only vs. the
  direct dispatch path, which has no capsule and needs these sections) — a real design decision
  (Rule 4), not a mechanical fix, and it risks the protected `TestWorkerBriefFileHoldsComposedBrief`.
- **Verification that this is real, not hypothetical:**
  `TestPrintBriefCommandFailsWhenPrintWorkerBriefsFindsDuplication` seeds one active pheromone
  signal against the ordinary fixture and the new detector correctly flags `Pheromone Signals` as
  duplicated — a genuine, already-shipping duplicate, not a constructed edge case.
- **Logged in:** `.planning/phases/190-lean-non-duplicated-delivery/deferred-items.md`

---

**Total deviations:** 2 (1 auto-fixed test correction, 1 discovered-and-deferred architectural finding)
**Impact on plan:** The auto-fixed test correction was necessary — without it, Task 1's own intended,
planned behavior change would leave a real test red. The deferred finding does not block this plan's
stated success criteria (which name only the build.md prose and the print-brief detector's
existence/correctness) but is flagged prominently since the new detector this plan built will
correctly, visibly fail on any real colony with active pheromone signals until a future phase
resolves it.

## Issues Encountered

**Full-package test suite (`go test ./cmd/... -count=1`) times out in this sandboxed worktree
environment for reasons unrelated to this plan's changes.** Three separate full-suite attempts hit
Go's internal test timeout or a tight explicit timeout, each time stuck on a DIFFERENT, unrelated
test (`TestPreserveWorktreeWorkIgnoresCleanWorktree` waiting on `os/exec.LookPath` for `git`;
`TestCLIPlanOnlyCompletionIsBoundAndIdempotent` waiting on a spawned `aether` binary subprocess to
exit; a goroutine inside `TestGoldenAutopilotPauseConditions` stuck `[runnable]` on a plain
`os.Getwd()` syscall). Each of these three tests, run in isolation with a generous timeout,
completed in under 30 seconds with no failures (`TestPreserveWorktreeWorkIgnoresCleanWorktree`:
0.15s; `TestGoldenAutopilotPauseConditions`: 0.91s; `TestCLIPlanOnlyCompletionIsBoundAndIdempotent`:
26.9s alone, since it spawns a real compiled binary). This points to cumulative subprocess-spawn
resource contention across the package's ~5000 tests in this specific sandboxed environment, not a
regression from this plan's changes. A broader regression pass excluding only the confirmed
slow/subprocess-heavy test name families (`TestCLI*`, `TestWorktreeSafety*`,
`TestPreserveWorktreeWork*`, `TestPreservationReportIsPlainEnglish`, `TestGoldenAutopilot*`) passed
cleanly in 184.8s (`go test ./cmd/ -count=1 -skip '...' ` — exit 0), and every test directly relevant
to this plan's three tasks (plus the plan's own prescribed `-run` verification pattern) passed with
zero failures. `go build ./...` and `go vet ./...` both exit 0.

## Next Phase Readiness

- Plan 190-02 (TS-host hive double-injection removal) shares zero files with this plan and can
  proceed/complete independently.
- The pheromone/handoff double-delivery finding in `deferred-items.md` is ready for a future phase
  to pick up; it names the exact functions involved and the test pattern to prove a fix against.

---
*Phase: 190-lean-non-duplicated-delivery*
*Completed: 2026-08-20*

## Self-Check: PASSED

**Files verified to exist:**
- FOUND: cmd/codex_build.go
- FOUND: cmd/codex_build_test.go
- FOUND: cmd/build_manifest_brief_composition_test.go
- FOUND: cmd/build_print_brief.go
- FOUND: cmd/build_print_brief_test.go
- FOUND: .claude/commands/ant/build.md
- FOUND: .claude/commands/ant-build.md
- FOUND: .opencode/commands/ant/build.md
- FOUND: .aether/commands/build.yaml
- FOUND: cmd/command_guide.go
- FOUND: .aether/skills/colony/aether-colony-build-cycle/SKILL.md
- FOUND: .planning/phases/190-lean-non-duplicated-delivery/deferred-items.md

**Commits verified to exist (`git log --oneline --all`):**
- FOUND: 2476f681 (Task 1)
- FOUND: ecb1161b (Task 2)
- FOUND: 5fadf214 (Task 3)
