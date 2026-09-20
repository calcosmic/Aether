---
phase: 165-core-lifecycle-commands
plan: 10
subsystem: init.md wrapper / shelf backlog ceremony
tags: [init, shelf, ceremony, gap-closure, wrapper-runtime-contract]
requires:
  - "aether init --promote-shelf/--dismiss-shelf (165-09)"
provides:
  - "init.md Shelf Backlog stage that only collects promoted_shelf_ids/dismissed_shelf_ids, never mutates shelf.json"
  - "init.md Approval stage that spends shelf choices inside the same aether init call the user consented to"
  - "TestInitWrapperCeremonyContract/shelf_ids_are_spent_only_inside_the_approval_init_call fence across all three init.md surfaces"
affects:
  - cmd/init_wrapper_ceremony_test.go
  - cmd/lifecycle_wrapper_contract_test.go
  - .claude/commands/ant/init.md
  - .opencode/commands/ant/init.md
  - .claude/commands/ant-init.md
  - .aether/commands/init.yaml
  - cmd/command_guide.go
tech-stack:
  added: []
  patterns:
    - "Fence a wrapper prose ordering hazard with a test that scans all three surfaces (canonical .claude, canonical .opencode, flat installed mirror) directly, not only transitively through byte-equality"
    - "Prove RED before the fix: run the new test against the unmodified wrapper and capture the failure output as evidence the fence is not decorative"
key-files:
  created: []
  modified:
    - cmd/init_wrapper_ceremony_test.go
    - cmd/lifecycle_wrapper_contract_test.go
    - .claude/commands/ant/init.md
    - .opencode/commands/ant/init.md
    - .claude/commands/ant-init.md
    - .aether/commands/init.yaml
    - cmd/command_guide.go
decisions:
  - "Removed shelf-promote-batch/shelf-dismiss-batch from init.md's Shelf Backlog stage entirely rather than reordering around them -- the runtime's --promote-shelf/--dismiss-shelf flags (165-09) make the standalone batch calls structurally unnecessary in this wrapper, so the hazard becomes unrepresentable rather than merely documented-around"
  - "Kept the ordering-invariant test in cmd/init_wrapper_ceremony_test.go rather than cmd/platform_doc_hygiene_test.go, per the plan's instruction that the init ceremony test is this phase's own fence"
metrics:
  duration: "~35 minutes"
  completed: "2026-08-03"
---

# Phase 165 Plan 10: Close the Wrapper Half of the Shelf-Promotion Ordering Hazard Summary

Closed the wrapper half of the Phase 165 Truth 8 blocker (review CR-01): all
three `init.md` surfaces (`.claude`, `.opencode`, and the flat installed
mirror) no longer contain any shelf-mutating command. The Shelf Backlog stage
now only records the user's chosen IDs; the Approval stage spends them inside
the same `aether init` call that creates the colony, using the flags plan
165-09 added to the runtime (`--promote-shelf` / `--dismiss-shelf`).

## What Changed

**Task 1 — RED-proven ceremony test fence:** Extended
`TestInitWrapperCeremonyContract` with a new subtest,
`shelf_ids_are_spent_only_inside_the_approval_init_call`, that runs directly
against all three init.md surfaces (both canonical wrappers plus the flat
mirror via `flatMirrorPath(repoRoot, "init")`) and asserts three invariants:
(a) no `aether shelf-` occurrence other than the read-only `aether
shelf-list`; (b) every `--promote-shelf` occurrence sits after the `##
Approval` heading; (c) the `## Shelf Backlog` section itself contains only
`aether shelf-list`. Added `aether shelf-promote-batch` and `aether
shelf-dismiss-batch` to the existing `forbidden` slice and `--promote-shelf` /
`promoted_shelf_ids` to `required`. Ran the test against the unmodified
wrapper first and confirmed it failed with the expected messages (both
forbidden commands named, `--promote-shelf` reported missing) before touching
any wrapper file — the RED state is the proof the fence is not decorative.
Also rewrote the stale doc comment on `TestLifecycleFlatMirrorsMatchCanonical`
(review WR-06), which described a specific two-verb RED/GREEN transition from
an earlier plan as though it were still current; the new comment describes
only the standing invariant (`aether install`/`aether update` copy the flat
mirror from canonical, never hand-edit it).

**Task 2 — Wrapper rewrite across all surfaces, plus YAML and Codex guide:**
In `.claude/commands/ant/init.md`:
- `## Shelf Backlog` no longer runs `aether shelf-promote-batch` or `aether
  shelf-dismiss-batch`. It now records the chosen IDs as `promoted_shelf_ids`
  / `dismissed_shelf_ids`, shows the chosen entries' `text` and `category`
  back to the user, and states plainly that nothing is written at this stage.
- `## Required Cross-Stage State` gained the two new carried values.
- `## Approval`'s init call now reads `... --promote-shelf
  "{promoted_shelf_ids}" --dismiss-shelf "{dismissed_shelf_ids}" "<refined
  goal>"`, with both flags omitted when the user chose nothing on the shelf.
  A new bullet tells the user which backlog ideas didn't carry forward if the
  runtime reports `shelf_failed`.
- `<failure_modes>` "User Cancels At Approval" now names the shelf
  explicitly: "no charter call, no pheromone writes, no shelf promotion or
  dismissal, nothing persisted," plus a bullet stating shelf choices
  collected earlier are discarded.

Propagated byte-for-byte to `.opencode/commands/ant/init.md` (reapplying the
single sanctioned `AskUserQuestion` vs `Ask` wording delta) and to
`.claude/commands/ant-init.md` (the flat mirror, byte-identical to canonical).
Updated `.aether/commands/init.yaml`'s shelf guardrail bullet to state the
promotion happens inside the `aether init` call via the two flags, and added
an `intent_refinement` bullet describing the Shelf Backlog / Approval split.
Added one `PreSteps` entry to `cmd/command_guide.go`'s `catalog["init"]`
describing the same collect-then-spend flow for the Codex guide, without
touching `RunCommand` (asserted verbatim elsewhere).

## Verification

- `go build ./cmd/aether` exits 0; `go vet ./...` clean
- `go test ./cmd/... ./pkg/...` passes in full (264.2s for `cmd`, all `pkg/*` green)
- `grep -rn 'aether shelf-promote-batch\|aether shelf-dismiss-batch' .claude/ .opencode/` returns no matches
- `.claude/commands/ant-init.md` is byte-identical to `.claude/commands/ant/init.md` (`diff -q` produces no output)
- `.claude` and `.opencode` init.md files differ on exactly 2 diff lines (one changed line pair: `AskUserQuestion` vs `Ask`)
- Every `--promote-shelf` occurrence (line 287) sits after `## Approval` (line 275) in `.claude/commands/ant/init.md`
- `grep -c 'currently RED for build and init' cmd/lifecycle_wrapper_contract_test.go` → 0
- `TestInitWrapperCeremonyContract` (including the new subtest), `TestInitWrapperStageSkeletonAndParity`, `TestLifecycleFlatMirrorsMatchCanonical`, `TestLifecycleWrapperReadOnlyBlocksAreConsistent`, `TestLifecycleWrappersDoNotParseEnvelopeAsPrimaryJob`, `TestSpecialistCommandSurfacesUnchanged`, `TestBuildMdOwnershipHandshake`, `TestCommandGuide*`, and `TestExecutionPathAudit_OneConductorPerWorkflow` all PASS
- `.aether/commands/init.yaml` parses as valid YAML (verified via `python3 -c "import yaml; yaml.safe_load(...)"`); `runtime.command` unchanged

## Deviations from Plan

None — plan executed exactly as written across both tasks. The RED state
was confirmed by running `TestInitWrapperCeremonyContract` against the
unmodified wrapper before any wrapper edit, showing both forbidden batch
commands and the missing `--promote-shelf`/`promoted_shelf_ids` strings;
Task 2 then turned it fully GREEN.

## TDD Gate Compliance

Task 1 is `tdd="true"`: RED confirmed (`e96bf540`, test commit, ceremony
contract subtest failing against the real wrappers) before Task 2's GREEN
fix (`cbab53fb`, fix commit). No REFACTOR commit was needed.

## Self-Check: PASSED

- `cmd/init_wrapper_ceremony_test.go` contains `shelf_ids_are_spent_only_inside_the_approval_init_call` and `initShelfBacklogSection` — FOUND
- `.claude/commands/ant/init.md` contains `--promote-shelf` and `promoted_shelf_ids` after `## Approval` — FOUND
- `.opencode/commands/ant/init.md` and `.claude/commands/ant-init.md` carry the same fix — FOUND
- `.aether/commands/init.yaml` and `cmd/command_guide.go` updated in lockstep — FOUND
- Commits `e96bf540` (test, RED confirmed) and `cbab53fb` (fix, GREEN confirmed) both present in `git log` — FOUND
