---
phase: 199-front-door-and-classic-contract
plan: "14"
subsystem: command-distribution
tags: [go, cobra, aliases, platform-sync, lifecycle, source-hygiene]

requires:
  - phase: 199-front-door-and-classic-contract
    plan: "07"
    provides: Canonical progressive front door and public lifecycle vocabulary
  - phase: 199-front-door-and-classic-contract
    plan: "13"
    provides: Canonical pause/resume Cobra commands and bounded hidden pre-Cobra migration redirects
provides:
  - Managed-header-gated pruning for retired command wrappers across Claude and OpenCode install homes
  - Public-alias reconciliation that cannot promote hidden parser compatibility into generated surfaces
  - Removal of the obsolete resume-colony canonical source and all three repository-managed wrappers
  - Current resume contract covering evidence provenance, idempotent transactions, and bounded hidden migration
affects: [plan-199-22, plan-199-24, update, install, publish, command-source-hygiene]

tech-stack:
  added: []
  patterns: [managed-header ownership proof, public-metadata alias boundary, cross-platform retired-wrapper pruning]

key-files:
  created: []
  modified:
    - cmd/platform_sync.go
    - cmd/wrapper_command_names.go
    - cmd/alias_reconcile_test.go
    - cmd/canonical_alias_test.go
    - cmd/contracts/resume.md
    - .aether/commands/resume-colony.yaml
    - .claude/commands/ant-resume-colony.md
    - .claude/commands/ant/resume-colony.md
    - .opencode/commands/ant/resume-colony.md

key-decisions:
  - "Treat pre-Cobra lifecycle redirects as input migration only: they are excluded from wrapper inventory, Cobra alias repair, help, completion, YAML, and generated files."
  - "Delete a retired installed command only when its managed header proves Aether ownership; preserve same-named custom files byte-for-byte."
  - "Describe recovery only through canonical resume while recording the hidden redirect as bounded, expiring parser plumbing rather than a user option."

patterns-established:
  - "Retired-wrapper pruning: normalize each generated wrapper basename, compare it with the public wrapper allowlist, and require the Aether-managed source header before deletion."
  - "Platform-home cleanup: derive one home from each command destination and apply the same ownership-aware retirement rule to Claude flat/nested and both OpenCode locations."

requirements-completed: [CEC-04, LIFE-01, LIFE-04]

duration: 28min
completed: 2026-09-04
---

# Phase 199 Plan 14: Retired Lifecycle Alias Pruning Summary

**Aether now removes its own stale suffixed pause/resume wrappers across Claude and OpenCode, preserves custom commands, and cannot regenerate hidden parser redirects as public aliases.**

## Performance

- **Duration:** 28 minutes
- **Started:** 2026-09-04T10:43:24Z
- **Completed:** 2026-09-04T11:11:32Z
- **Tasks:** 2/2
- **Files changed:** 9 production, test, contract, and managed-surface files

## Accomplishments

- Removed the retired lifecycle names from the public wrapper registry and filtered declared alias reconciliation through that registry, while retaining synchronization for genuine public aliases such as `flags`.
- Added ownership-aware retirement cleanup across Claude flat/nested, OpenCode home, and OpenCode config destinations; managed old-name wrappers are deleted while same-named custom commands survive byte-for-byte.
- Deleted the obsolete canonical resume source and its three repository-managed wrappers, so update, install, publish, and source checks cannot restore that public surface.
- Rewrote the current resume lifecycle contract around confirmed/reconstructed/conflicting/unknown provenance, handoff-keyed transaction replay, zero-write conflicts, and one expiring pre-Cobra compatibility rewrite.

## Task Commits

Each task was committed atomically:

1. **Task 1: Prevent managed sync from reviving parser-only aliases**
   - `d53055c7` — test(199-14): define retired alias pruning contract (RED)
   - `785ca441` — feat(199-14): keep retired aliases pruned (GREEN)
2. **Task 2: Remove the resume-colony source and generated surfaces**
   - `ef4bd978` — docs(199-14): retire resume-colony public surfaces

**Plan metadata:** committed with this summary.

## Files Created/Modified

- `cmd/platform_sync.go` — public-metadata alias filter plus managed-header retirement cleanup for every supported platform command home.
- `cmd/wrapper_command_names.go` — public wrapper inventory with the two parser-only lifecycle tokens removed.
- `cmd/alias_reconcile_test.go` — real install-to-hub-to-update coverage for public alias repair, managed retired-wrapper deletion, and custom-file preservation.
- `cmd/canonical_alias_test.go` — canonical/public alias metadata boundary and update reconciliation coverage.
- `cmd/contracts/resume.md` — canonical recovery evidence, provenance, transaction, state-effect, and bounded-parser-compatibility contract.
- `.aether/commands/resume-colony.yaml` — intentionally deleted obsolete canonical command source.
- `.claude/commands/ant-resume-colony.md`, `.claude/commands/ant/resume-colony.md`, `.opencode/commands/ant/resume-colony.md` — intentionally deleted obsolete managed wrappers.

## Decisions Made

- The hidden argument normalizer is not command metadata. Only aliases that exist in the public wrapper registry may participate in generated-wrapper repair.
- A matching filename does not prove ownership. Retirement cleanup reads the wrapper header and removes only Aether-generated content, leaving user-authored commands untouched.
- The contract teaches `aether resume` and `/ant-resume` only; the one-milestone input rewrite is documented without turning it into an alias, flag, or selectable route.

## Deviations from Plan

None - plan executed exactly as written.

## Verification

- PASS — `go test ./cmd -run '^(TestRetiredLifecycleAliasPruning199|TestCanonicalAlias|TestUpdateDoesNotRestoreParserOnlyAlias|TestCommandSourceHygiene)$' -count=1` (14 cases including subtests).
- PASS — all four obsolete public/source files are absent.
- PASS — `aether source-check --root . --json` reports `ok: true`, 128 generated wrappers aligned, and zero findings.
- PASS — `cmd/contracts/resume.md` contains neither the retired command spelling nor a separate public recovery command.

## TDD Gate Compliance

- RED `d53055c7` precedes GREEN `785ca441`.
- The RED gate failed because managed retired wrappers survived update, unmanaged same-named commands were deleted, and the old lifecycle names remained in the public wrapper registry.
- GREEN made the exact Task 1 suite pass without weakening the custom-file or public-alias assertions.

## Known Stubs

None. Plan-owned code and current guidance contain no TODO/FIXME markers, placeholder or coming-soon copy, or hardcoded empty user-facing data source.

## Issues Encountered

- The additional broader `TestWrapperCommandNamesMatchCanonicalCorpus` check still reports the already-landed `/ant-maintenance` wrapper missing from the shared wrapper registry. That staged Plan 199 integration is outside Plan 14's retired-lifecycle scope; no later-plan behavior or tests were changed to hide it. All exact Plan 14 tests and the production source checker pass.
- The installed state progress handler counted 16 of 34 summaries but could not match this repository's body format and wrote `0%`; the frontmatter percentage was corrected to the established plan-count value, `47%`. The requirements handler likewise could not match the repository's bold-description format, but `CEC-04`, `LIFE-01`, and `LIFE-04` were already checked complete, so no requirements edit was needed.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Plan 199-22 can remove remaining historical pause/resume guidance without any updater path recreating the retired wrappers.
- Plan 199-24 can reuse the managed-header ownership rule for broader obsolete-wrapper cleanup.
- The aggregate Phase 199 suite still needs its owning maintenance-registry and staged snapshot migrations before the final normal/race repository gate.
- Protected pre-existing `.planning/config.json`, `.gsd/`, and `199-PATTERNS.md` changes remain untouched and unstaged.

## Self-Check: PASSED

- All five retained Plan 14 implementation/test/contract files and this summary exist; all four intentionally retired source/wrapper files are absent.
- RED `d53055c7`, GREEN `785ca441`, and surface-removal `ef4bd978` resolve in Git in the documented order.
- The summary and all remaining unstaged changes pass whitespace validation.
- Exact Plan 14 tests and the production source checker pass, and protected pre-existing paths remain unstaged.

---
*Phase: 199-front-door-and-classic-contract*
*Completed: 2026-09-04*
