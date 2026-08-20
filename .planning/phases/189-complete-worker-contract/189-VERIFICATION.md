---
phase: 189-complete-worker-contract
verified: 2026-08-20T08:09:21Z
status: passed
score: 7/7 must-haves verified
overrides_applied: 0
---

# Phase 189: Complete Worker Contract Verification Report

**Phase Goal:** Every worker sees its full task and the output contract the finalizer enforces; reviewers get the same context on every path.
**Verified:** 2026-08-20T08:09:21Z
**Status:** passed
**Re-verification:** No — initial verification

## Method

This is goal-backward, execution-based verification, not a re-read of SUMMARY.md claims. Beyond
reading the source, I traced every claimed fix to a cobra command / TS-host bridge entry point,
wrote and ran my own throwaway adversarial probes against the REAL merge/dispatch pipeline (not the
phase's own hand-built test fixtures), deliberately broke the schema drift-lock to prove it actually
fires, ran the full targeted test suite plus the full `cmd` and `pkg/codex` packages myself, and
diffed all 6 fixture-repair files by hand. All probe files were deleted and the git tree confirmed
clean (`git status --porcelain` empty) before, during, and after.

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | (SC1) A merged dispatch's brief lists every covered task's constraints, hints and success criteria, correctly numbered | VERIFIED | Read `findDispatchTasks`/`renderDispatchTaskItemsSection` (`cmd/codex_build.go:2803-2919`). **Demonstrated by execution**: wrote a throwaway test driving the REAL merge pipeline (`plannedBuildDispatchesForSelectionWithState` → real `coalesceSequentialDispatches`/`mergeDispatchInto`, not a hand-built `CoveredTaskIDs` literal) on a 3-task dependency chain; the real merge produced `CoveredTaskIDs=[t1 t2 t3]`, and `composeBuildManifestBrief`'s output contained all 9 distinct constraint/hint/criteria strings exactly once each, correctly labeled `**Task 1:**`/`**Task 2:**`/`**Task 3:**`, in the same order as the `## Assignment` numbering. Probe file deleted, tree clean after. |
| 2 | (SC2, build) The brief states the handoff/return schema the finalizer enforces, wrapper-only, never duplicated for native-Codex workers | VERIFIED | `composeBuildManifestBrief` (`cmd/codex_build.go:3230-3263`) appends `codex.HandoffFieldsSummary` after the base brief; `renderCodexBuildWorkerBrief`'s raw output (fed as `TaskBrief` to native-Codex workers, `cmd/codex_build.go:1705`) never contains it — confirmed in my own probe's output and by `TestComposeBuildManifestBriefStatesHandoffSchemaOnceNotOnNativePath` (PASS). |
| 3 | (SC2, continue) Continue's finalizer actually enforces the "empty handoff is rejected" promise its brief states (CR-01, blocker closed) | VERIFIED | `mergeExternalContinueResults` (`cmd/codex_continue_finalize.go:679-681`) now rejects a `"completed"` result with `codex.IsEmptyWorkerHandoff(result.Handoff)`. Traced live: `continueFinalizeCmd` (`Use: "continue-finalize"`, registered on `rootCmd`) → `runCodexContinueFinalize` → `mergeExternalContinueResults`, the exact command `continue.md` instructs the wrapper to run. Ran `TestContinueFinalizeRejectsCompletedWorkerWithoutHandoff` myself: PASS, covering both the explicit-empty-object and omitted-key cases through the real `runCodexContinueFinalize` entry point on a plan built by the real `runCodexContinuePlanOnly`. |
| 4 | (SC3) Continue's external dispatches carry capsule, skill, pheromone; all three wrappers instruct delivery, parity-tested | VERIFIED | `codexContinuePlanManifest.ContextCapsule`/`.PheromoneSection` (`cmd/codex_continue_plan.go:62-63`) resolved once in `runCodexContinuePlanOnly`. Traced the full live chain: `.claude/commands/ant/continue.md` → `aether host continue --dry-run` (TS host subprocess) → `command-registry.ts:continueArgs` → `aether continue --plan-only` (Go binary) → `runCodexContinuePlanOnly` → JSON on stdout → wrapper step 5 explicitly reads `continue_manifest.context_capsule` + `.pheromone_section` + `dispatch.skill_section`. All three wrapper files byte-identical (`diff` exit 0, confirmed independently). `TestContinueWrapperInstructsCapsuleAndPheromoneDelivery` PASS; `TestLifecycleFlatMirrorsMatchCanonical/continue` PASS. `.codex/CODEX.md` confirmed naming `continue` in both Skills and Pheromone Signals sentences. |
| 5 | (WR-01) Merged-dispatch numbering survives an unresolvable covered task ID without shifting later labels | VERIFIED | `coveredDispatchTask{Position, ID, Task}` (`cmd/codex_build.go:2803-2836`) carries a stable original position; `renderUnresolvedDispatchTaskNotice` visibly flags gaps. `TestBuildWorkerBriefMergedDispatchKeepsNumberingAcrossUnresolvedTask` PASS (run directly). |
| 6 | (WR-02) The scaffolding-ratio regression lock also covers the merged-dispatch code path | VERIFIED | `TestBuildWorkerBriefIsMostlyTaskForMergedDispatch` PASS (run directly), logs 67.2% task-relevant share for a real 3-task merged brief, well clear of the 40% floor; original `TestBuildWorkerBriefIsMostlyTask` untouched and still PASS. |
| 7 | (IN-01) The three near-duplicate "is this handoff empty" functions are consolidated to two genuinely distinct ones | VERIFIED | `pkg/codex/handoff.go` now has `IsEmptyWorkerHandoff` (terminal rejection) and `IsEmptyWorkerHandoffIncludingFreshness` (pre-synthesis fallback), both exported, with an inline doc explaining why they stay distinct. `cmd/codex_dispatch_contract.go`'s old `workerHandoffEmpty` duplicate is gone (grep: zero matches repo-wide); `buildWorkerHandoffRecord` calls the renamed exported function; `pkg/codex/worker.go`'s `normalizeWorkerClaims` does too. |

**Score:** 7/7 truths verified

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `pkg/codex/handoff.go` | `HandoffFieldsSummary` constant, single canonical schema source | VERIFIED | Exported, non-empty, referenced by symbol from `pkg/codex/worker.go:835`, `cmd/codex_build.go:3233`, `cmd/codex_continue_plan.go` (`continueExternalBriefWithHandoffSchema`) — no third hand-copy found anywhere (`grep -rn "changed_files, commands_run"` finds only the one declaration). |
| `cmd/codex_build.go` | `findDispatchTasks` (multi-task resolution) | VERIFIED | Exists, substantive (handles single/multi/unresolved cases distinctly), wired at both call sites (`renderCodexBuildWorkerBrief`, `codegraphTextPartsForBuildBrief`); `findDispatchTask` (singular) fully deleted repo-wide. |
| `cmd/codex_build_test.go` | Invariant tests for merged-task coverage | VERIFIED | `TestBuildWorkerBriefCoversEveryMergedTaskConstraintsAndCriteria`, `TestBuildWorkerBriefMergedDispatchKeepsNumberingAcrossUnresolvedTask`, `TestBuildWorkerBriefIsMostlyTaskForMergedDispatch`, `TestComposeBuildManifestBriefStatesHandoffSchemaOnceNotOnNativePath` all present and PASS. |
| `cmd/codex_continue_plan.go` | `ContextCapsule`/`PheromoneSection` manifest fields; handoff-schema append on every external dispatch's brief | VERIFIED | Fields present, JSON-tagged correctly, resolved once (not per-dispatch); `continueExternalBriefWithHandoffSchema` wraps both watcher and reviewer `Brief:` assignments. |
| `.claude/commands/ant/continue.md` (+ flat mirror + OpenCode) | Delivery instruction naming `context_capsule`, `pheromone_section`, `skill_section` | VERIFIED | All three byte-identical (`diff` exit 0 both pairs); instruction text present and specific (step 5, "Reads:" line). |
| `cmd/continue_wrapper_ceremony_test.go` | `TestContinueWrapperInstructsCapsuleAndPheromoneDelivery` | VERIFIED | Present, PASS, asserts against both canonical wrapper paths with exact substrings. |
| `cmd/codex_continue_finalize.go` | Empty-handoff rejection on the external finalize path (CR-01) | VERIFIED | Present at line 679-681; live-wired to `continue-finalize` cobra command. |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|----|--------|---------|
| `renderCodexBuildWorkerBrief` | `findDispatchTasks` | direct call, loop over covered tasks | WIRED | Confirmed by source + my own real-merge-pipeline probe. |
| `composeBuildManifestBrief` | `pkg/codex.HandoffFieldsSummary` | `fmt.Sprintf` append after base brief | WIRED | Confirmed present in composed output, absent from raw `renderCodexBuildWorkerBrief` output. |
| `codegraphTextPartsForBuildBrief` | `findDispatchTasks` | loop over covered tasks for codegraph relevance text | WIRED | Read directly (`cmd/codegraph_context.go:105-123`); correctly skips unresolved (`covered.Task == nil`) entries, consistent with WR-01's fix. |
| `runCodexContinuePlanOnly` | `resolveCodexWorkerContext`/`resolvePheromoneSection` | called once each, set on manifest | WIRED | Read directly; matches build's "resolve once" precedent. |
| `plannedExternalContinueDispatches` | `pkg/codex.HandoffFieldsSummary` | `continueExternalBriefWithHandoffSchema` wraps both Brief assignments | WIRED | Confirmed via source and passing `TestContinueExternalDispatchBriefsStateHandoffSchemaOnceNotOnNativePath`. |
| `.claude/commands/ant/continue.md` | `.claude/commands/ant-continue.md`, `.opencode/commands/ant/continue.md` | byte-identical delivery instruction | WIRED | `diff` exit 0 both pairs, confirmed independently. |
| `buildCmd --plan-only` (live CLI) | `attachBuildDispatchContext` → `composeBuildManifestBrief` → `renderCodexBuildWorkerBrief` | `runCodexBuildPlanOnlyWithOptions` (`cmd/codex_workflow_cmds.go:157`) | WIRED | Traced to `rootCmd.AddCommand(buildCmd)`; `build.md` explicitly instructs `aether build $ARGUMENTS --plan-only` and forbids the TS-host build path. |
| `continue.md` (live wrapper) | `runCodexContinuePlanOnly` | `aether host continue --dry-run` → TS host `continueArgs` → `aether continue --plan-only` (Go) | WIRED | Traced through `.aether/ts-host/src/host.ts` (`runDryRunDispatchedCommand`) and `command-registry.ts` (`continueArgs` emits `["continue","--plan-only",...]`); stdout JSON is exactly the Go manifest the wrapper parses. |
| `continueFinalizeCmd` (live CLI) | `mergeExternalContinueResults` empty-handoff check | `runCodexContinueFinalize` | WIRED | Traced to `rootCmd.AddCommand(continueFinalizeCmd)`; matches `continue.md`'s instructed `aether continue-finalize --completion-file`. |

### Behavioral Spot-Checks / Adversarial Probes (executed, not inferred)

| Probe | Command / Method | Result | Status |
|-------|------------------|--------|--------|
| Real merge (N=3, dependency chain) through `coalesceSequentialDispatches`/`mergeDispatchInto` → `composeBuildManifestBrief` | Throwaway Go test in `cmd` package, deleted after | All 9 fields present exactly once, correctly labeled Task 1/2/3, handoff schema present in composed / absent in raw | PASS |
| Duplicate covered task ID (`CoveredTaskIDs: ["1","1"]`) at the render layer | Throwaway Go test | Confirmed: renders the same content twice under `**Task 1:**` and `**Task 2:**`. Independently confirmed (as REVIEW.md's WR-01 also found) that no live path (`coalesceSequentialDispatches`) can ever produce a duplicate ID — each source dispatch/task is visited exactly once during coalescing. Not reachable in production; not required to fix by REVIEW.md's own WR-01 scope (position-stability only). **Documented residual, not a gap.** | INFO (non-blocking) |
| Schema drift-lock: add a field to `WorkerHandoff`, confirm tests catch it | Manually added `VerifierProbeField string` to `pkg/codex/handoff.go`, ran `TestCompletionPacketSchemaMatchesStructs` + `TestWrapperFieldListMatchesSchema` | Both FAILED with clear diagnostics ("schema drift", "missing: [verifier_probe_field]"); reverted via `git checkout --`, both PASS again, tree clean | PASS (drift-lock proven to fire) |
| 13 repaired fixtures — diffed all 6 files by hand (not just 2-3 spot-checked) | `git show 133f4840 -- <6 files>` | Every repair adds realistic `codex.WorkerHandoff{VerificationStatus: "pass"/"fail", NextWorkerInstructions/KnownFailures: [...]}` content; zero assertion-weakening found | PASS |
| Full `go build ./...` / `go vet ./...` | run directly | exit 0 both | PASS |
| Full `go test ./cmd/... -count=1` | run directly | ok, 414.7s, zero failures | PASS |
| Full `go test ./pkg/codex/... -count=1` | run directly | ok, 20.6s, zero failures | PASS |
| All named tests from 189-01/02/03 plans + REVIEW remediation | run directly, `-v -count=1` | 20+ named tests, all PASS | PASS |

Full-repo `-race` suite was NOT re-run — the orchestrator's brief states this was already independently confirmed (18/20 packages, zero race warnings) and re-running it would not change any of the above findings; the two packages actually touched by this phase (`cmd`, `pkg/codex`) were independently re-run by me without `-race` and passed cleanly.

### Requirements Coverage

Not applicable. ROADMAP.md's Phase 189 entry carries no `Requirements:` line, and both plans declare `requirements: []` in frontmatter. No orphaned requirement IDs found in REQUIREMENTS.md for Phase 189 (zero matches).

### Anti-Patterns Found

None. Swept every non-test file touched across all three plans (`pkg/codex/handoff.go`, `pkg/codex/worker.go`, `cmd/codex_build.go`, `cmd/codegraph_context.go`, `cmd/codex_continue_plan.go`, `cmd/codex_continue_finalize.go`, `cmd/build_print_brief.go`, `cmd/codex_dispatch_contract.go`) plus all four wrapper markdown files for `TODO|FIXME|XXX|TBD|HACK|PLACEHOLDER|coming soon|not yet implemented` and empty-return stub patterns. Zero hits (the only "placeholder" matches are pre-existing, legitimate domain terminology describing timeout-result semantics, unrelated to code completeness).

### Human Verification Required

None. Every claim in this phase is either a Go code-level guarantee (verified by reading + execution) or a wrapper-prose instruction whose maximum feasible automated verification — the instruction text existing, being specific, and being pinned by a parity test — was checked and passes. Whether a live Claude Code/OpenCode session actually follows that prose at runtime is the same class of residual uncertainty this repo's own test suite already accepts as its verification ceiling for wrapper markdown (see `TestContinueWrapperCeremonyContract`, `TestContinueWrapperInstructsCapsuleAndPheromoneDelivery`) — not a phase-189-specific gap, and not something a code-level verifier can close further.

### Gaps Summary

No gaps. All three ROADMAP success criteria hold under adversarial, execution-based verification — not just SUMMARY.md narrative. All four 189-REVIEW.md findings (1 blocker, 2 warnings, 1 info) are genuinely closed: I independently re-derived and executed the blocker's regression test, read the warning fixes' implementation directly, and confirmed the info consolidation left no stray references to the old duplicate function. The one residual observation (duplicate covered-task-ID would double-render if it ever occurred) is confirmed unreachable via the live merge pipeline by two independent people (the original reviewer and me, via a fresh throwaway probe), and was correctly scoped out of WR-01's required fix — it is documented here for the record, not withheld, but does not block the phase.

---

_Verified: 2026-08-20T08:09:21Z_
_Verifier: Claude (gsd-verifier)_
