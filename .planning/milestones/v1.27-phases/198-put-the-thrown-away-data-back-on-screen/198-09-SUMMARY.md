---
phase: 198-put-the-thrown-away-data-back-on-screen
plan: 09
subsystem: cli-visuals
tags: [go, go/ast, invariant, ratchet, closeout, seal, plan, continue, SHOW-02, SHOW-05]

# Dependency graph
requires:
  - phase: 198-04
    provides: "closeoutPlanDirectVisual / closeoutSealDirectVisual / closeoutContinueDirectVisual bridge functions this invariant seeds its call-graph walk from"
  - phase: 198-05
    provides: "workerMeasurementFigures / the measured-vs-not-reported figure pair pattern (context for the invariant's continue-worker-flow coverage)"
  - phase: 198-06
    provides: "renderCriterionEvidenceLines / task_evidence-is-a-misnomer decision, reused verbatim as this invariant's allowlist reason for task_evidence"
  - phase: 198-07
    provides: "the build-start blocker advisory (unrelated surface, listed only because this plan's frontmatter names it as a dependency)"
provides:
  - "TestRenderedVisualsShowEveryCarriedField -- an invariant over the five finalizer result maps (plan completed/mid-loop, continue advanced/blocked, seal): every top-level key is either read by a render function reachable from that workflow's screen (traced via go/ast, not hand-typed) or listed in a reasoned exception file"
  - "TestRenderedFieldAllowlistOnlyShrinks -- the shrink-only ratchet over that exception file, keyed on (finalizer, key)"
  - "TestDeliberatelyDroppedDisplayChoicesStayDropped -- locks two of SHOW-05's three drops: no tmux/terminal-multiplexer/second-terminal-window dependency, no boxed/bordered section frame other than the kept └── nested-detail marker"
  - "TestSafeToClearLineStaysAStatement -- locks the third: the kept 'safe to clear your context now' line stays a statement, the retired question form stays retired, both asserted on the real function return value and a wording derived from it at test time"
  - "A fixed SEE-03 violation: cmd/status.go no longer tells the owner to tail a log 'in another terminal'"
  - "Two honestly-recorded, still-open discovered gaps in .planning/WINDOWS.md (entries 6, 7): plan's own research_warning/research_failed_phases/gaps fields and its 'next' command never reach the screen, because runCodexPlanFinalize never calls closeLifecycleRun"
affects: [any-future-phase-adding-a-key-to-a-finalizer-result-map, any-future-phase-touching-cmd/codex_visuals.go-render-call-graph]

# Actuals (#2632)
actuals:
  tokens: 13586
  tasks: 3
  commits: 3

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "go/ast call-graph tracing over a literal key list: both the 'carried' side (top-level keys read off a fixture map built the way the runtime builds it) and the 'read' side (string-literal index expressions on a map/raw identifier, followed recursively through unnarrowed pass-through calls like renderLifecycleClosing(result, ...)) are derived programmatically at test run time, never typed into the test as a list -- so the invariant cannot go stale when a render function is renamed or a helper is added."
    - "Multi-root call-graph seeding: each finalizer's read-set is the union of its direct-path entry renderer (cmd/codex_visuals.go), its chat-path closeout bridge (cmd/closeout_direct_render.go), and closeLifecycleRun (the finalize-time step that folds the map's own 'next' into the unified next-action envelope for continue/seal, but never for plan) -- reflecting the real, asymmetric pipeline rather than a single idealized call path."
    - "Shrink-only ratchet keyed on a composite ID: TestRenderedFieldAllowlistOnlyShrinks compares (finalizer, key) pairs, not bare key names, because the same key legitimately recurs across finalizers with different reasons (e.g. \"next\") -- a key moving to a new finalizer it wasn't previously exempted for is exactly the addition the ratchet exists to catch."

key-files:
  created:
    - cmd/rendered_fields_invariant_test.go
    - cmd/deliberate_drops_lock_test.go
    - cmd/testdata/rendered_field_allowlist.json
    - cmd/testdata/rendered_field_allowlist_baseline.json
  modified:
    - cmd/status.go
    - .planning/WINDOWS.md

key-decisions:
  - "The invariant's call-graph walk seeds THREE roots per finalizer (entry renderer, closeout bridge, closeLifecycleRun) rather than one, after discovering that several genuinely-read fields (sealed, summary, review_depth for continue, next for continue/seal) are read by functions the direct entry renderer never calls directly -- a single-root walk would have force-allowlisted a dozen keys that are, in fact, honestly consumed by a different part of the real pipeline."
  - "task_evidence is allow-listed for both continue finalizers, citing 198-06's own documented decision (a misnomer distinct from the actual D-11 evidence field, verification.Criteria) verbatim, rather than re-litigating it."
  - "Four discovered display gaps (research_warning, research_failed_phases, gaps for plan completed, next for plan completed/mid-loop) are honestly allow-listed with reasons stating they are NOT provenance-only, plus recorded as open WINDOWS.md deviations (entries 6, 7) -- not fixed, because fixing them requires editing cmd/codex_visuals.go and cmd/codex_plan_finalize.go, both outside this plan's declared files_modified and (for codex_visuals.go) owned this wave by sibling plan 198-08."
  - "A fifth, previously-undiscovered SEE-03 violation (cmd/status.go telling the owner to tail a log 'in another terminal') WAS fixed in this plan, not deferred: it is a one-line prose change directly caused by writing the very test meant to catch it, in a file no sibling plan owns, and leaving a known rule violation live specifically to avoid a diff would have been perverse."
  - "The box-corner scan forbids all box-drawing corner glyphs (light, double, heavy) except light └ (U+2514), which SEE-12's nested-detail marker uses; a file-level allowlist exempts codex_visuals.go's AETHER ASCII-art wordmark banner, a decorative logo unrelated to the boxed-table style SHOW-05 retires."

requirements-completed: [SHOW-02, SHOW-05]

coverage:
  - id: D1
    description: "An invariant over the five finalizer result maps proves every top-level key is either rendered or explicitly, reasonably excepted -- not a check that a page has the right headings."
    requirement: "SHOW-02"
    verification:
      - kind: unit
        ref: "cmd/rendered_fields_invariant_test.go#TestRenderedVisualsShowEveryCarriedField"
        status: pass
      - kind: other
        ref: "manual: temporarily added a throwaway key to the seal fixture, confirmed the test named it exactly, reverted"
        status: pass
    human_judgment: false
  - id: D2
    description: "The allow-listed exception file is a ratchet: it can shrink freely, and growing it requires an explicit, reviewed baseline edit."
    requirement: "SHOW-02"
    verification:
      - kind: unit
        ref: "cmd/rendered_fields_invariant_test.go#TestRenderedFieldAllowlistOnlyShrinks"
        status: pass
      - kind: other
        ref: "manual: added an entry to the live list only, confirmed the ratchet named it and failed, reverted"
        status: pass
    human_judgment: false
  - id: D3
    description: "The three things the team deliberately left out (tmux/second-terminal-window dependency, boxed section frames) stay out, and the one line they deliberately kept (the safe-to-clear statement) stays a statement, never a question."
    requirement: "SHOW-05"
    verification:
      - kind: unit
        ref: "cmd/deliberate_drops_lock_test.go#TestDeliberatelyDroppedDisplayChoicesStayDropped"
        status: pass
      - kind: unit
        ref: "cmd/deliberate_drops_lock_test.go#TestSafeToClearLineStaysAStatement"
        status: pass
      - kind: other
        ref: "manual: a throwaway probe file with a tmux reference and a forbidden box corner, and a temporary question-mark regression in codex_visuals.go, each proved the corresponding test fails by name; all reverted"
        status: pass
    human_judgment: false
  - id: D4
    description: "The whole phase gate is green: the full cmd package suite, a full module build, and go vet."
    verification:
      - kind: other
        ref: "go build ./... && go vet ./... && go test ./cmd -count=1 (227s) && go test ./... -count=1 (all packages ok)"
        status: pass
    human_judgment: false

# Metrics
duration: 95min
completed: 2026-08-29
status: complete
---

# Phase 198 Plan 09: Close the Two Ratchets Summary

**A go/ast-traced invariant proves every finalizer result-map key is rendered or reasonably excepted (shrink-only), and a second lock test proves the three deliberately-dropped display choices stay dropped while the one kept line stays a statement -- catching and fixing one real, previously-undiscovered SEE-03 violation live in the process.**

## Performance

- **Duration:** ~95 min
- **Tasks:** 3 completed
- **Files modified:** 6 (4 created, 2 modified)

## Accomplishments

- `TestRenderedVisualsShowEveryCarriedField` derives BOTH sides of the comparison programmatically: the "carried" side reads keys off the five existing `wrapperParity*Result()` fixtures (already the runtime-faithful fixture pattern 198-04 established), and the "read" side walks the actual call graph via `go/ast` from each workflow's entry renderer plus its closeout bridge plus `closeLifecycleRun`, following any call that passes the map through unnarrowed (`renderLifecycleClosing(result, ...)`). No literal key list appears on either side.
- Seeded `cmd/testdata/rendered_field_allowlist.json` with 46 reasoned entries: most are genuinely provenance/machine-only or redundant with a live-state-resolved value already shown; four are honestly-labeled discovered-and-deferred gaps (not fixed here, since the fix lives in a file this plan cannot touch this wave), and one (`task_evidence`, both continue finalizers) cites 198-06's own prior decision verbatim rather than re-litigating it.
- `TestRenderedFieldAllowlistOnlyShrinks` freezes that exception list against a committed baseline, keyed on `(finalizer, key)` since the same key name recurs across finalizers with different reasons.
- `TestDeliberatelyDroppedDisplayChoicesStayDropped` scans shipped Go sources and the `.claude`/`.opencode` wrapper markdown for tmux/terminal-multiplexer/second-terminal-window phrasing and for forbidden box-drawing corner glyphs (permitting only the kept `└──` nested-detail marker), with a file-level allowlist for the AETHER ASCII-art wordmark banner.
- `TestSafeToClearLineStaysAStatement` asserts `renderContextClearGuidanceForPlatform`'s real return value is a statement (never a question) for both branches, then derives the "clear ... context" wording from that same return value and scans every source/wrapper file for it appearing as a question -- so a future reword can't make the scan go stale.
- Discovered and fixed a genuine, previously-uncaught SEE-03 violation while writing the multiplexer scan: `cmd/status.go`'s "still running" tip told the owner to tail a log file "in another terminal" -- exactly the second-terminal-window framing SEE-03 retired.
- Every new test's fail path was proven live during execution (a throwaway probe file, a temporary allowlist entry, a temporary question-mark regression) and reverted before committing.
- Ran the full phase gate: `go build ./...`, `go vet ./...`, `go test ./cmd -count=1` (227s, all green), and `go test ./... -count=1` (every package ok).

## Task Commits

Each task was committed atomically:

1. **Task 1: The invariant — nothing carried is silently dropped** - `55f57213` (test)
2. **Task 2: Freeze the exception list so it can only shrink** - `fdf4c521` (test)
3. **Task 3: Lock the three deliberate drops and run the phase gate** - `c514461e` (test)

**Plan metadata:** committed with this SUMMARY.

## Files Created/Modified

- `cmd/rendered_fields_invariant_test.go` - `TestRenderedVisualsShowEveryCarriedField`, `TestRenderedFieldAllowlistOnlyShrinks`, and the go/ast call-graph walker (`renderedFieldASTFuncs`, `renderedFieldKeysReadByFunc`, `renderedFieldKeysReadForEntries`)
- `cmd/testdata/rendered_field_allowlist.json` - 46 reasoned exception entries across the five finalizers
- `cmd/testdata/rendered_field_allowlist_baseline.json` - frozen copy of the above, the shrink-only ratchet's comparison target
- `cmd/deliberate_drops_lock_test.go` - `TestDeliberatelyDroppedDisplayChoicesStayDropped`, `TestSafeToClearLineStaysAStatement`, and their scan/derivation helpers
- `cmd/status.go` - dropped "in another terminal" from the still-running tip (SEE-03 fix)
- `.planning/WINDOWS.md` - two new open deviation entries (6, 7) for the discovered plan-side rendering gaps

## Decisions Made

See `key-decisions` in frontmatter: three-root call-graph seeding per finalizer; `task_evidence` allow-listed by citing 198-06's decision; four discovered gaps honestly allow-listed and recorded in WINDOWS.md rather than fixed (out of declared file scope); the fifth discovered gap (`cmd/status.go`) WAS fixed, being a one-line prose change in a file no sibling plan owns; the box-corner scan's exception shape.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed a genuine SEE-03 violation discovered while writing the multiplexer/second-terminal-window scan**
- **Found during:** Task 3, while sanity-checking the scan's phrase list against the real codebase before writing the test
- **Issue:** `cmd/status.go`'s "still running" tip read: `` Watch progress with `tail -f .aether/data/spawn-tree.txt` in another terminal, or run `aether proof`... `` — literally instructing the owner to open a second terminal, the exact "second-terminal watch emphasis" `.planning/decisions/SEE-03-one-terminal-streaming.md` retires.
- **Fix:** Dropped "in another terminal" from the sentence; the command itself (`tail -f ...`) is unchanged, only the instruction to run it in a second window is removed.
- **Files modified:** `cmd/status.go`
- **Verification:** No golden fixture or other test referenced the old exact string (grepped for it before and after); `go build ./...` and the full `go test ./cmd -count=1` pass.
- **Committed in:** `c514461e` (Task 3 commit)

### Discovered, Recorded (deferred — out of this plan's file scope)

**2. [Discovered gap] `renderPlanVisual` never surfaces `research_warning` / `research_failed_phases` / `gaps` for a completed plan**
- **Found during:** Task 1, deriving the "carried" key set for the "plan completed" fixture and finding these three keys unread by any function in the render call graph
- **Issue:** `renderResearchFailedWarning`'s own doc comment states the warning exists "so the omission is durable and visible" when a phase was planned without its research — but `renderPlanVisual` never reads `result["research_warning"]` (or the phase-ID list that feeds it, or the plan's own unresolved `gaps`). The warning is computed and carried, then silently dropped on both the direct and chat paths.
- **Why not fixed here:** the fix lives in `cmd/codex_visuals.go`, outside this plan's declared `files_modified` and owned this wave by sibling plan 198-08.
- **Recorded:** `cmd/testdata/rendered_field_allowlist.json` (honest, non-provenance-only reasons) and `.planning/WINDOWS.md` entry 6.

**3. [Discovered gap] Plan finalize's own suggested `next` command never reaches the screen**
- **Found during:** Task 1, tracing why "next" was unread for plan while it WAS correctly traced (via `closeLifecycleRun`) for continue and seal
- **Issue:** `runCodexPlanFinalize` never calls `closeLifecycleRun` (unlike continue-finalize and `completeSealRuntime`), so the plan's own `result["next"]` never folds into the unified next-action envelope `renderLifecycleClosing` reads back — the closing card instead independently resolves a next step from live colony state, which usually matches but is not guaranteed to.
- **Why not fixed here:** the fix lives in `cmd/codex_plan_finalize.go`, outside this plan's declared files.
- **Recorded:** `cmd/testdata/rendered_field_allowlist.json` (both "plan completed" and "plan mid-loop") and `.planning/WINDOWS.md` entry 7.

---

**Total deviations:** 1 auto-fixed (Rule 1 bug, in a file with no ownership conflict), 2 discovered-and-deferred (recorded honestly in the allowlist and WINDOWS.md, not silently patched, per this plan's file-ownership prohibition and 198-04's established precedent for the same situation).
**Impact on plan:** The fixed issue was a real, user-visible violation of a settled written decision (SEE-03) with no scope conflict. The two deferred gaps are genuine but require editing files this plan is explicitly forbidden from touching this wave; they are now tracked for whichever future plan legitimately owns `cmd/codex_visuals.go`'s plan-rendering section or `cmd/codex_plan_finalize.go`.

## Issues Encountered

Two implementation-time near-misses, both caught and fixed before committing (not shipped as bugs):

1. **False negative on the "statement, not a question" check.** The first version of `TestSafeToClearLineStaysAStatement` checked whether the WHOLE return value ended in "?", but both branches append a further sentence after the clearing claim (`"... now." + " Run \`aether resume\` to restore.\n"`), so a regression in the clearing sentence itself wouldn't have moved the string's trailing character. Fixed by extracting the specific sentence containing "clear" (bounded by the nearest `.`/`?`/newline on each side) and checking THAT sentence's terminator instead. Proved the fix by reproducing the exact regression and confirming the corrected test caught it.
2. **Over-broad call-graph seeding produced false "unread" flags for `sealed`, `summary`, `review_depth`, `blocked`, and several others.** The first version seeded only the entry renderer (`renderPlanVisual`/`renderContinueVisual`/`renderSealVisual`); several genuinely-consumed keys are read by the closeout bridge functions (`closeoutSealRenderInputs`, `closeoutContinueRenderInputs`) BEFORE the renderer is ever called, not by the renderer itself. Fixed by seeding each finalizer with the closeout bridge and `closeLifecycleRun` as additional call-graph roots, which correctly resolved all of these without adding a single allowlist entry for them.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Phase 198's two roadmap-closing ratchets (criteria 2 and 5) are now enforced by named, provably-failing tests rather than by inspection.
- Two genuine, previously-invisible display gaps were found and honestly recorded (WINDOWS.md entries 6, 7) rather than silently patched or silently ignored — visible to whichever future plan next touches `cmd/codex_visuals.go`'s plan-rendering section or `cmd/codex_plan_finalize.go`.
- No blockers for phase completion; this was the last plan (wave 4) in phase 198's dependency graph.
- Full `go build ./...`, `go vet ./...`, `go test ./cmd -count=1` (227s), and `go test ./... -count=1` all green.

---
*Phase: 198-put-the-thrown-away-data-back-on-screen*
*Completed: 2026-08-29*

## Self-Check: PASSED

- Verified all created/modified files exist on disk: `cmd/rendered_fields_invariant_test.go`, `cmd/deliberate_drops_lock_test.go`, `cmd/testdata/rendered_field_allowlist.json`, `cmd/testdata/rendered_field_allowlist_baseline.json`, `cmd/status.go`, `.planning/WINDOWS.md`.
- Verified all three task commit hashes (`55f57213`, `fdf4c521`, `c514461e`) exist in `git log`.
- Re-ran the plan's full `<verification>` command set: `go build ./...`, `go vet ./...`, `go test ./cmd -run 'TestRenderedVisualsShowEveryCarriedField' -count=1`, `go test ./cmd -run 'TestRenderedFieldAllowlistOnlyShrinks|TestOrphanAllowlistOnlyShrinks' -count=1`, `go test ./cmd -run 'TestDeliberatelyDroppedDisplayChoicesStayDropped|TestSafeToClearLineStaysAStatement' -count=1` -- all pass.
- Ran the full `go test ./cmd -count=1` suite once as the final verification pass -- `ok`, 227.118s, zero failures.
- Ran `go test ./... -count=1` across the whole module -- every package `ok`.
- Confirmed both allowlist JSON files parse and every live entry appears in the frozen baseline (`TestRenderedFieldAllowlistOnlyShrinks` passing is this proof).
