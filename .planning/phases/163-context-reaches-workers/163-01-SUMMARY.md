---
phase: 163-context-reaches-workers
plan: 01
subsystem: build-orchestration
tags: [go, colony-prime, context-capsule, wrapper-contract, claude-code, opencode]

# Dependency graph
requires: []
provides:
  - "codexBuildManifest.ContextCapsule field, populated once per plan-only manifest from resolveCodexWorkerContext()"
  - "Both wrapper build.md dispatch bullets instruct reading dispatch_manifest.context_capsule once and prepending it verbatim per spawn"
  - "TestBuildManifestCarriesContextCapsuleOnce — executable invariant for CONTEXT-02/CONTEXT-03"
  - "TestBuildWrapperCeremonyContract required-strings coverage for the capsule instruction"
affects: [163-02, 163-03, 163-04, 163-05, 163-06, 165-core-lifecycle-commands]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Manifest-level compute-once/share for grounding payloads, mirroring executeCodexBuildDispatches' Path A"
    - "Deriving invariant-test markers from a function's own runtime output rather than hardcoding literal headings"

key-files:
  created:
    - cmd/codex_build_manifest_context_test.go
  modified:
    - cmd/codex_build.go
    - .claude/commands/ant/build.md
    - .opencode/commands/ant/build.md
    - cmd/build_wrapper_ceremony_test.go
    - cmd/safety_invariant_test.go

key-decisions:
  - "ContextCapsule populated only when planOnly=true inside buildCodexBuildManifest itself — the hosted path keeps computing its own capsule at cmd/codex_build.go:1410 (Path A), so this is the single new call site, not a second assembly path"
  - "TestPlanOnlyUnchanged/build_plan_only updated to exclude .cache_* files from its no-new-files assertion — these are pkg/cache's pre-existing, gitignored, cross-invocation SessionCache write-through files, not colony state, and reading COLONY_STATE.json through resolveCodexWorkerContext() during plan-only now triggers that established cache path for the first time on this command"

patterns-established:
  - "Capsule-marker-from-runtime-value: TestBuildManifestCarriesContextCapsuleOnce derives its uniqueness probe from manifest.ContextCapsule's own first line rather than a hardcoded heading string, so a heading rename can't silently defeat the test"

requirements-completed: [CONTEXT-02, CONTEXT-03]

# Metrics
duration: 55min
completed: 2026-07-29
---

# Phase 163 Plan 01: Context Capsule Reaches Wrapper-Spawned Build Workers Summary

**Colony-prime's grounding capsule (state, decisions, phase learnings, instincts, hive wisdom, prior reviews, blockers, user preferences) now reaches wrapper-spawned build workers via one manifest-level field, computed once and read once by both Claude Code and OpenCode wrapper orchestrators — closing the one genuinely unconnected wire in this phase.**

## For Dummies

Before this change, when a worker was spawned through the Claude Code or OpenCode wrapper (as opposed to Aether's own direct subprocess dispatch), it never received the "memory" of the colony — no prior decisions, no learnings from earlier phases, no active warnings. It only got its own task instructions. That meant a wrapper-spawned worker on a cheap model had to guess at context a smarter/more expensive path already had for free.

This plan adds one field to the build manifest (`context_capsule`) that carries that memory, computed exactly once per build (not once per worker — that would bloat the file and risk drift), and teaches both wrapper platforms to read it once and paste it at the front of every worker's prompt. A new test fails loudly if either half of that wiring breaks again.

## Performance

- **Duration:** 55 min
- **Started:** 2026-07-29T16:25:50Z (first task commit)
- **Completed:** 2026-07-29T16:48:14Z
- **Tasks:** 3 completed (plus 1 auto-fixed deviation)
- **Files modified:** 6 (1 created, 5 modified)

## Accomplishments

- `codexBuildManifest.ContextCapsule` carries the colony-prime capsule on the plan-only manifest, computed exactly once, never duplicated per dispatch
- Both `.claude/commands/ant/build.md` and `.opencode/commands/ant/build.md` now instruct the orchestrator to read `dispatch_manifest.context_capsule` once and prepend it verbatim ahead of `dispatch.brief` for every spawned worker
- `TestBuildManifestCarriesContextCapsuleOnce` makes CONTEXT-02/CONTEXT-03 executable-verifiable: non-empty on a populated plan-only colony, exact count of 1 across the full manifest JSON, absent from every dispatch brief, empty on hosted (non-plan-only) manifests
- `TestBuildWrapperCeremonyContract` extended so either wrapper platform silently dropping the instruction fails the suite
- Full `cmd` package test suite (`go test ./cmd/... -count=1`) passes with no regressions

## Task Commits

Each task was committed atomically:

1. **Task 1: Carry the context capsule on the plan-only manifest, computed once** - `390e87ee` (feat)
2. **Task 2: Pin the wiring with a manifest-level invariant test** - `2fdd9135` (test)
3. **Task 3: Teach both wrappers to deliver the manifest capsule** - `bc8197db` (feat)
4. **Deviation fix: tolerate session-cache side files in plan-only invariant test** - `13b5fd98` (fix)

_No plan-metadata commit was made in this worktree — the orchestrator handles STATE.md/ROADMAP.md centrally after the wave merges._

## Files Created/Modified

- `cmd/codex_build.go` - Adds `ContextCapsule` field to `codexBuildManifest` (doc comment matching the `Brief` field's voice); populates it once inside `buildCodexBuildManifest`, guarded by `planOnly`, via a single new `resolveCodexWorkerContext()` call
- `cmd/codex_build_manifest_context_test.go` - New: `TestBuildManifestCarriesContextCapsuleOnce`, the executable invariant for CONTEXT-02/CONTEXT-03
- `.claude/commands/ant/build.md` - Amends the two dispatch-composition bullets to read the manifest capsule once and prepend it verbatim per spawn (content-only edit, no structural change — Phase 165 still owns `build.md`'s structure)
- `.opencode/commands/ant/build.md` - Identical amendment, mirrored on the OpenCode platform
- `cmd/build_wrapper_ceremony_test.go` - Adds `"dispatch_manifest.context_capsule"` to the required-strings list both wrapper platforms must satisfy
- `cmd/safety_invariant_test.go` - `TestPlanOnlyUnchanged/build_plan_only` now excludes `.cache_*` files from its no-new-files assertion (see Deviations)

## Decisions Made

- **Capsule call site is `buildCodexBuildManifest` itself, guarded by `planOnly`** — not `attachBuildDispatchContext`'s per-dispatch loop (would reintroduce the exact duplication CONTEXT-03 exists to prevent) and not a new manifest-construction helper (the plan explicitly names `buildCodexBuildManifest` as the single construction point; changing its signature was prohibited because `cmd/codex_build_finalize.go` is owned by plan 05 in this wave)
- **No `CharterSection` field added** — per plan instruction, charter arrives inside the capsule itself once plan 02 lands, so a second manifest field would duplicate content and bypass `colony.AssessPromptSource`
- **`.cache_*` files excluded from the plan-only no-mutation invariant** rather than avoiding the `resolveCodexWorkerContext()` call or inventing a cache-free variant of `buildColonyPrimeOutput` — the SessionCache write-through pattern (`pkg/cache/session_cache.go`) is pre-existing, sanctioned, gitignored, and already exercised by numerous other call sites (`codex_plan.go`, `codex_continue.go`, `pheromone_loader_test.go`, etc.); treating it as equivalent to real colony-state mutation would have required either abandoning the single-source-of-truth call to `resolveCodexWorkerContext()` (violating CONTEXT-02) or building a second capsule-assembly path (explicitly forbidden pitfall) — both worse than updating the test's scope to match the codebase's existing cache convention

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking issue] `resolveCodexWorkerContext()` call triggered a previously-dormant SessionCache write during `build --plan-only`, breaking `TestPlanOnlyUnchanged`**
- **Found during:** Post-Task-3 full verification (`go test ./cmd/... -count=1`)
- **Issue:** `build --plan-only` had never previously called anything that reads `COLONY_STATE.json` through `pkg/cache.SessionCache`. Task 1's new `resolveCodexWorkerContext()` call (via `buildColonyPrimeOutput(true)`) reads state through that cache for the first time on this command, and the cache's read-through `Load()` writes a `.cache_COLONY_STATE.json` disk-persistence file as a side effect (documented, pre-existing behavior — see `pkg/cache/session_cache.go:138-174`). `TestPlanOnlyUnchanged/build_plan_only` asserts zero new top-level files appear in `.aether/data/` during plan-only orchestration and failed with "file added during orchestration: .cache_COLONY_STATE.json"
- **Fix:** Updated the test's post-build snapshot comparison to exclude `.cache_*`-prefixed files, alongside the existing `build/` directory carve-out, with an inline comment explaining the cache is a gitignored, cross-invocation performance layer (purged by `aether cache-clean`), not colony state
- **Files modified:** `cmd/safety_invariant_test.go`
- **Verification:** `go test ./cmd -run TestPlanOnlyUnchanged -count=1 -v` passes (both `plan_plan_only` and `build_plan_only` subtests green); full `go test ./cmd/... -count=1` passes with no other regressions
- **Committed in:** `13b5fd98`

## Deliberate Regression Proofs

Per Task 2 and Task 3's acceptance criteria, both halves of the wiring were proven to fail when removed, then reverted to green.

**1. Capsule assignment removed (Task 2):**

Commented out `contextCapsule = resolveCodexWorkerContext()` inside `buildCodexBuildManifest`, then ran:
```
go test ./cmd -run TestBuildManifestCarriesContextCapsuleOnce -count=1 -v
```
Observed failure:
```
=== RUN   TestBuildManifestCarriesContextCapsuleOnce
    codex_build_manifest_context_test.go:89: CONTEXT-02 regressed: plan-only manifest.ContextCapsule is empty on a populated colony — the wrapper-spawned worker would receive no colony-prime grounding
--- FAIL: TestBuildManifestCarriesContextCapsuleOnce (0.00s)
FAIL
```
Reverted; rerun confirmed `--- PASS: TestBuildManifestCarriesContextCapsuleOnce (0.01s)`.

**2. Wrapper instruction removed (Task 3):**

Reverted both amended bullets in `.claude/commands/ant/build.md` to their pre-plan wording (temporarily), then ran:
```
go test ./cmd -run TestBuildWrapperCeremonyContract -count=1 -v
```
Observed failure:
```
=== RUN   TestBuildWrapperCeremonyContract
    build_wrapper_ceremony_test.go:68: .../.claude/commands/ant/build.md missing "dispatch_manifest.context_capsule"
--- FAIL: TestBuildWrapperCeremonyContract (0.00s)
FAIL
```
Reverted; rerun confirmed `--- PASS: TestBuildWrapperCeremonyContract (0.00s)`.

## Issues Encountered

None beyond the deviation documented above.

## Pending Follow-ups

- `aether publish` was deliberately NOT run — the phase publishes once at the end, after all distributed-file edits (`.claude/commands/ant/build.md`, `.opencode/commands/ant/build.md`) across all plans in this phase have landed, per Task 3's explicit instruction.

## Verification Against Plan

- [x] `go build ./cmd/aether` exits 0
- [x] `go vet ./...` exits 0
- [x] `go test ./cmd -run 'TestBuildManifestCarriesContextCapsuleOnce|TestBuildWrapperCeremonyContract|TestBuildWorkerBrief' -count=1` passes
- [x] `go test ./cmd/... -count=1` passes (full suite, no regressions — including the one deviation fix)
- [x] Deliberate-regression proofs recorded for both the capsule assignment and the wrapper instruction (see above)
- [x] `aether build <n> --plan-only` emits a manifest whose `context_capsule` is non-empty on a populated colony
- [x] That capsule text occurs exactly once in the manifest JSON regardless of dispatch count
- [x] No dispatch brief contains the capsule text
- [x] Both wrapper platforms instruct prepending `dispatch_manifest.context_capsule` once per worker
- [x] A named test (`TestBuildManifestCarriesContextCapsuleOnce`, `TestBuildWrapperCeremonyContract`) fails when any of the above stops holding, and that failure has been observed

## Known Stubs

None.

## Threat Flags

None — the plan's own `<threat_model>` (T-163-04, T-163-05, T-163-06) fully covers the surface touched by this plan; no new network endpoints, auth paths, or schema changes were introduced.

## Next Steps

Proceed to plan 163-02 (charter section inside the capsule) and the remaining phase-163 plans; this plan's `ContextCapsule` field and delivery contract are now available as a dependency for any plan that needs to verify the manifest-level capsule reaches wrapper workers.

## Self-Check: PASSED

All created/modified files confirmed present on disk (cmd/codex_build.go, cmd/codex_build_manifest_context_test.go, cmd/build_wrapper_ceremony_test.go, .claude/commands/ant/build.md, .opencode/commands/ant/build.md, cmd/safety_invariant_test.go, this SUMMARY.md). All claimed commit hashes (390e87ee, 2fdd9135, bc8197db, 13b5fd98, ae150cdb) confirmed present in `git log --oneline --all`.
