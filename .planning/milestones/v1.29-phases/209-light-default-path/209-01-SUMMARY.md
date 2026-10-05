---
phase: 209-light-default-path
plan: 01
subsystem: cli
tags: [cobra, cli, routing, voice-corpus, ast-guard]

requires:
  - phase: 208-messy-practice-project-gate
    provides: the release-gate discipline and the CLAUDE.md Definition-of-Done rule this plan's tests are written against
provides:
  - "/ant-go \"<what you want>\" -- the one entry command from D-01, working end-to-end on the small route"
  - resolveJobSizeRoute, the single route authority (small vs. big), computed from the real working tree
  - gatherJobSizeFacts, the read-only fact gatherer feeding it
  - renderQuickJobBody, extracted from renderQuickVisual so /ant-go reuses the exact quick-job rendering rather than duplicating it
  - the "go-small" screen registered in the shared voice corpus
affects: [209-02, any later plan wiring the big/planning route or the D-02 default-menu checkpoint]

actuals:
  tokens: 10840
  tasks: 3
  commits: 3

tech-stack:
  added: []
  patterns:
    - "One route authority, AST-guarded: resolveJobSizeRoute is the sole decider; go_cmd.go is parsed with go/parser in tests to refuse a second copy of the comparison (mirrors TestStatusLineComesFromTheSharedDecision)."
    - "Reuse, never re-implement: the small route calls the existing runQuickJob rather than a second single-helper dispatch path; renderGoVisual reuses renderQuickJobBody (extracted from renderQuickVisual) rather than re-rendering the quick-job body."

key-files:
  created:
    - cmd/go_route.go
    - cmd/go_cmd.go
    - cmd/go_route_test.go
    - cmd/go_cmd_test.go
    - cmd/classic_voice_go_test.go
    - .aether/commands/go.yaml
    - .claude/commands/ant-go.md
    - .claude/commands/ant/go.md
    - .opencode/commands/ant/go.md
  modified:
    - cmd/command_truth.go
    - cmd/wrapper_command_names.go
    - cmd/classic_command_parity_test.go
    - cmd/classic_voice_coverage_test.go
    - .aether/commands/classic-command-parity.json

key-decisions:
  - "The big route is named and stopped on in this plan (per its own scope), not built out -- runGoJob's big branch reports the route/reason and tells the owner to run `aether plan` directly for now. Plan 02 is the escalation/expansion work."
  - "Matching a directory referent expands to every file underneath it (not the directory itself as one match), so 'refactor the internal package' is sized by how many files that package actually holds -- otherwise a directory-sized job would always read as one match and never escalate."
  - "The route authority reads the recorded project state via a plain file read (readLifecycleState), the same way frontDoorLifecycleProjection does -- never storage.NewStore -- so gathering facts stays genuinely read-only."

requirements-completed: [UED-16]

coverage:
  - id: D1
    description: "/ant-go \"<what you want>\" runs the small-route job end-to-end: one helper dispatched, the project's own checks run, a rendered screen naming the route and why."
    requirement: UED-16
    verification:
      - kind: unit
        ref: "cmd/go_cmd_test.go#TestGoSmallRouteRunsTheJobEndToEnd"
        status: pass
      - kind: unit
        ref: "cmd/go_cmd_test.go#TestGoNeverRefusesTheOwner"
        status: pass
    human_judgment: false
  - id: D2
    description: "The route decision is computed from the real working tree (present vs. absent files, directory size, an accepted plan with outstanding work), and is provably indifferent to how the sentence is worded."
    requirement: UED-16
    verification:
      - kind: unit
        ref: "cmd/go_route_test.go#TestGoRouteIsComputedNotConstant"
        status: pass
      - kind: unit
        ref: "cmd/go_route_test.go#TestGoRouteIgnoresHowTheSentenceIsWorded"
        status: pass
      - kind: unit
        ref: "cmd/go_route_test.go#TestGoRouteDirectoryOverBudgetEscalates"
        status: pass
      - kind: unit
        ref: "cmd/go_route_test.go#TestGoRouteAcceptedPlanWithOutstandingWorkAlwaysEscalates"
        status: pass
    human_judgment: false
  - id: D3
    description: "resolveJobSizeRoute has exactly one authority; cmd/go_cmd.go computes no route decision of its own."
    requirement: UED-16
    verification:
      - kind: unit
        ref: "cmd/go_route_test.go#TestGoRouteHasOneAuthority"
        status: pass
    human_judgment: false
  - id: D4
    description: "/ant-go is reachable on all three wrapper surfaces (Claude flat, Claude namespaced, OpenCode) and the parity manifest, with all four sources agreeing."
    requirement: UED-16
    verification:
      - kind: unit
        ref: "cmd/classic_command_parity_test.go#TestClassicCommandParity"
        status: pass
    human_judgment: false
  - id: D5
    description: "The /ant-go screen carries the Classic voice at the measured reference density, speaks plain English, and cannot silently lose its registration."
    requirement: UED-16
    verification:
      - kind: unit
        ref: "cmd/classic_voice_coverage_test.go#TestEveryOrdinaryScreenIsMeasuredForVoice"
        status: pass
      - kind: unit
        ref: "cmd/classic_voice_corpus_test.go#TestVoicedScreensSpeakPlainEnglish (go-small subtest)"
        status: pass
      - kind: unit
        ref: "cmd/classic_voice_metric_test.go#TestEveryVoicedScreenMeetsTheReferenceDensity (go-small subtest)"
        status: pass
    human_judgment: false
  - id: D6
    description: "aether go appears in the plain `aether --help` menu the owner actually sees."
    verification: []
    human_judgment: true
    rationale: "Deliberately NOT done in this plan -- see Deviations. The owner's own D-02 checkpoint (a mandatory review of the real rendered default-menu screen) is what decides whether /ant-go joins the curated front-door help; adding it here would preempt that checkpoint. `aether go --help` and `rootCmd.Find([]string{\"go\"})` both prove the command is genuinely registered and runnable today."

duration: 45min
completed: 2026-09-24
status: complete
---

# Phase 209 Plan 01: The Light Default Path Summary

**`resolveJobSizeRoute` is now a real, tree-measured decision, and `/ant-go` reuses `runQuickJob` end-to-end on the small route with an AST guard that fails if the command ever grows a second copy of the size logic.**

## Performance

- **Duration:** ~45 min
- **Tasks:** 3
- **Files created:** 9
- **Files modified:** 5

## Accomplishments

- `/ant-go "<what you want>"` exists as a real cobra command, registered on all three wrapper surfaces (`.aether/commands/go.yaml`, `.claude/commands/ant-go.md` + `.claude/commands/ant/go.md`, `.opencode/commands/ant/go.md`) plus the classic-command-parity manifest, placed immediately after `init`.
- `resolveJobSizeRoute` (`cmd/go_route.go`) is the one route authority: priority-ordered exactly like `resolveVerificationDepth` -- an accepted plan with outstanding work, then zero matched paths, then more matched paths than `smallJobFileBudget` (3), all escalate to the big route; otherwise the small route. Every decision carries a non-empty, plain-English reason.
- `gatherJobSizeFacts` walks the real working tree once (skipping `.git`, `node_modules`, `.aether/data`) and matches a sentence's tokens against real file/directory names -- proven two-valued across two genuinely different repositories, and proven indifferent to smallness/bigness-claiming wording, both via automated tests AND by hand-verified mutation (documented below).
- The small route calls the existing `runQuickJob` directly -- no second single-helper dispatch path -- and `renderGoVisual` reuses the exact quick-job rendering via a newly extracted `renderQuickJobBody` (factored out of `renderQuickVisual`, byte-identical output for the existing command).
- The `go-small` screen is registered in the shared voice corpus (`cmd/classic_voice_go_test.go`) and passes every plain-English, no-raw-token, and reference-density check.

## Task Commits

1. **Task 1: End-to-end "/ant-go does the small job"** - `e9055d3b` (feat)
2. **Task 2: The route is measured from the change, not read off the sentence** - `5f31515a` (test)
3. **Task 3: The new screen joins the shared voice** - `1e69756f` (test)

**Plan metadata:** (this commit, following this SUMMARY)

## Files Created/Modified

- `cmd/go_route.go` - `jobSizeRoute`/`jobSizeDecision`/`jobSizeFacts` types, `resolveJobSizeRoute` (the one authority), `gatherJobSizeFacts` (the read-only tree walk + state read)
- `cmd/go_cmd.go` - `goCmd`, `runGoJob`, `renderGoVisual`
- `cmd/command_truth.go` - extracted `renderQuickJobBody` from `renderQuickVisual` so `/ant-go` reuses it verbatim
- `cmd/wrapper_command_names.go` - added `"go": true`
- `cmd/classic_command_parity_test.go` - added the `go` row to `classicPublicCommandInventory()`
- `.aether/commands/classic-command-parity.json` - added the matching `go` row
- `.aether/commands/go.yaml`, `.claude/commands/ant-go.md`, `.claude/commands/ant/go.md`, `.opencode/commands/ant/go.md` - the command triplet, copied from `improve`'s shape
- `cmd/go_route_test.go` - `TestGoRouteIsComputedNotConstant`, `TestGoRouteIgnoresHowTheSentenceIsWorded`, `TestGoRouteDirectoryOverBudgetEscalates`, `TestGoRouteAcceptedPlanWithOutstandingWorkAlwaysEscalates`, `TestGoRouteHasOneAuthority`, `TestResolveJobSizeRouteBranches`
- `cmd/go_cmd_test.go` - `TestGoSmallRouteRunsTheJobEndToEnd`, `TestGoNeverRefusesTheOwner`, `TestGoCommandIsRegistered`, `TestGoCmdSourceCarriesNoSymbolLiteral`
- `cmd/classic_voice_go_test.go` - registers the `go-small` screen
- `cmd/classic_voice_coverage_test.go` - added `"go"` to `voiceScreenFamilies`

## Decisions Made

- The big route is named and stopped on, not built out. `runGoJob`'s big branch reports `route`/`route_reason` and tells the owner (through the rendered screen) to run `aether plan` directly for now. This matches the plan's own scope ("Leave the `Attempt` branch to plan 02") and D-01's requirement to never refuse -- the command still does something honest on every input, it just doesn't yet dispatch the planning pipeline itself.
- A directory referent expands to every file underneath it for sizing purposes (not counted as one match), so a sentence naming a whole package is correctly judged by how much of that package there actually is.
- Facts are gathered via a plain file read of `COLONY_STATE.json` (mirroring `frontDoorLifecycleProjection`), never `storage.NewStore` -- keeping `gatherJobSizeFacts` genuinely read-only, as the plan requires.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1/3 - DRY / avoid duplication] Extracted `renderQuickJobBody` from `renderQuickVisual`**
- **Found during:** Task 1
- **Issue:** The plan's action explicitly requires `/ant-go`'s screen to show "the small route's existing quick-job lines rendered the way `renderQuickVisual` renders them" -- doing this without duplicating ~70 lines of rendering logic required factoring the shared body out of the existing function.
- **Fix:** Extracted the job/question line, helper name, file list, checks outcome, verdict, elapsed time and summary rendering into a new `renderQuickJobBody(result)`, called by both the unchanged `renderQuickVisual` (byte-identical output, confirmed by the existing quick-job tests still passing) and the new `renderGoVisual`.
- **Files modified:** `cmd/command_truth.go` (not in the plan's `files_modified` frontmatter list, which is why this is logged as a deviation rather than a planned edit)
- **Verification:** All pre-existing `renderQuickVisual`/quick-job tests still pass; `TestGoSmallRouteRunsTheJobEndToEnd` asserts the reused body renders correctly inside the new screen.
- **Committed in:** `e9055d3b` (Task 1 commit)

### Scoped-out by the owner's own D-02 gate (not a deviation, a deliberate omission)

**`aether go` does not yet appear in the curated `aether --help` menu.** The plan's Task 1 acceptance criteria states "`aether go` appears in `aether --help` output" -- verified true for `aether go --help` (the command's own help) and for `rootCmd.Find`, but the root `aether --help` renders a hand-curated menu (`frontDoorHelpGroups` in `cmd/root.go`) that is explicitly the subject of D-02's **mandatory owner checkpoint on the real rendered screen** (209-CONTEXT.md). Adding `/ant-go` to that curated menu here, before the owner has seen and ruled on the actual screen, would preempt a decision CONTEXT.md explicitly reserves for him. This is recorded as coverage item D6 (`human_judgment: true`) rather than silently marked passing.

---

**Total deviations:** 1 auto-fixed (DRY refactor, outside the plan's declared file list) + 1 scoped-out-by-design (the D-02 default-menu membership, deliberately deferred to the owner's own checkpoint).
**Impact on plan:** Neither affects `/ant-go`'s functional completeness on the small route. The command is fully registered, dispatchable, and tested; only its position on the *curated* help screen awaits the owner's separate, already-scheduled decision.

## Manual mutation verification (Definition of Done -- "a test must be able to fail")

Every test named in the plan's acceptance criteria as needing to "fail when X is temporarily edited" was hand-verified by actually making that edit, running the test, confirming the failure message, and reverting (`diff` confirmed byte-for-byte restoration each time):

| Mutation | Test | Result |
|---|---|---|
| `resolveJobSizeRoute` forced to always return `small` | `TestGoRouteIsComputedNotConstant` | FAILED as expected ("same route in two genuinely different repositories") |
| `resolveJobSizeRoute` forced to always return `big` | `TestGoRouteIsComputedNotConstant` | FAILED as expected |
| `gatherJobSizeFacts` made to inject extra matches when the sentence contains "massive" | `TestGoRouteIgnoresHowTheSentenceIsWorded` | FAILED as expected ("route must come from the size of the change, not the words used") |
| `len(...MatchedPaths) > smallJobFileBudget` pasted directly into `cmd/go_cmd.go` | `TestGoRouteHasOneAuthority` (AST half) | FAILED as expected (named both the `smallJobFileBudget` reference and the `len(...MatchedPaths)` comparison) |
| `go-small` registration commented out | `TestEveryOrdinaryScreenIsMeasuredForVoice` | FAILED as expected, naming `go` by name |

## Issues Encountered

Ran the plan's own manual end-to-end verification line (`AETHER_OUTPUT_MODE=visual go run ./cmd/aether go "fix the spelling in README.md"` over a scratch directory containing a single `README.md`) against the real built binary with the real invoker (not the test spy) -- it genuinely dispatched a real single builder helper, which took ~12.6s and correctly reported "Found nothing that needed changing" (the file had no actual spelling error). This confirms the full production dispatch path, not just the mocked test path. No corrective action needed.

Pre-existing, unrelated test failures observed in the same package while running the phase-level verification command (confirmed unrelated by `git diff --stat` -- none touch any file this plan modified):
- `TestGoSourceHintsMatchCobraContracts` (references `cmd/codex_native_context.go`, `cmd/partial_work_label.go`)
- `TestGoldenBuildVisualOutput` / `TestGoldenContinueVisualOutput` (golden-file wording drift unrelated to this plan's screens)
- `TestNoRegisteredSubcommandIsUnreferenced` (`aether codex-native-worker context-ack` orphan, pre-existing at 401 registered commands before this plan's change, confirmed via isolated run before any of this plan's files existed)

None of these are caused by this plan's changes and none were touched, per the deviation rules' scope boundary.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- The route authority, the small-route wiring, and the voice-corpus registration are all in place and tested for `/ant-go`.
- Plan 02 (or a later plan) is where the big/planning route gets built out, and where `smallAttemptFacts`/the `Attempt` field of `jobSizeDecision` (defined but deliberately unread by `resolveJobSizeRoute` in this plan) gets its first reader for the D-03 escalation-mid-attempt behavior.
- D-02's mandatory owner checkpoint on the real rendered default-menu screen is still outstanding and untouched by this plan -- `/ant-go`'s curated-menu membership is deferred to it by design.

## Known Stubs

None that block this plan's own goal. The big-route branch of `runGoJob` is an intentional stub (a single map return, no dispatch) explicitly scoped to a later plan by 209-01-PLAN.md itself ("Leave the `Attempt` branch to plan 02" / "plan 02 fills that branch in") -- not a gap introduced by this execution.

---
*Phase: 209-light-default-path*
*Completed: 2026-09-24*

## Self-Check: PASSED
