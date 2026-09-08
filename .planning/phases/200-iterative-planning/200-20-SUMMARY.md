---
phase: 200-iterative-planning
plan: 20
subsystem: discuss-wrapper-contract
tags: [discuss, wrappers, evidence, owner-decisions, specification, claude, opencode, codex]

requires:
  - phase: 200-11
    provides: evidence-first material decision batches and settled Discuss-to-draft-spec handoff
  - phase: 200-12
    provides: typed lifecycle authority facts and exact next-action policy
  - phase: 200-19
    provides: canonical terminal and JSON projections for planning and specification state
provides:
  - Claude and OpenCode Discuss wrappers that render one runtime-issued material-decision batch
  - exact answer, revalidation, resume, and settled-to-Specification wrapper guidance
  - a public Discuss lifecycle contract and Codex research skill aligned to current Go behavior
  - semantic regression coverage across YAML, all public wrappers, documentation, skill, and runtime flag help
affects: [200-21, 200-23, 200-24, 200-25, discuss, specification, command-wrappers]

tech-stack:
  added: []
  patterns: [runtime-owned materiality, render-only wrappers, exact answer binding, draft-before-plan authority]

key-files:
  created: [cmd/discuss_wrapper_contract_200_test.go]
  modified:
    - .aether/commands/discuss.yaml
    - .claude/commands/ant-discuss.md
    - .claude/commands/ant/discuss.md
    - .opencode/commands/ant/discuss.md
    - cmd/contracts/discuss.md
    - .aether/skills/colony/aether-colony-research/SKILL.md

key-decisions:
  - "Wrappers render the complete Go-issued material batch and never compose, classify, truncate, or answer decision cards."
  - "Every answer is bound to the runtime-issued decision ID and followed by a fresh Discuss result; prior answers remain reusable only under exact equivalence."
  - "Settled discussion ends at a draft or retained approved specification and routes to Specification review, never directly to planning."
  - "Oracle guidance remains unchanged while Codex Discuss orchestration adopts the same runtime-owned evidence boundary as Claude and OpenCode."

patterns-established:
  - "Wrapper parity: one YAML-owned runtime call, one structured result, platform-native rendering, and no host-side lifecycle writes."
  - "Semantic contract testing: normalize prose punctuation while preserving required fields, authority boundaries, runtime help, and forbidden positive shortcuts."

requirements-completed: [SYNTH-02, CEC-03, PLAN-04, PLAN-05]

duration: 13 min
completed: 2026-09-08
---

# Phase 200 Plan 20: Discuss Wrapper and Research Contract Summary

**Claude, OpenCode, and Codex now present the Go runtime's complete evidence-backed decision batch and require separate draft-specification review before planning.**

## Performance

- **Duration:** 13 min
- **Started:** 2026-09-08T00:03:56Z
- **Completed:** 2026-09-08T00:16:06Z
- **Tasks:** 2
- **Files modified:** 7 plan-owned wrapper, documentation, skill, and test files

## Accomplishments

- Replaced fixed-count, wrapper-authored clarification guidance with one JSON runtime call whose complete `material_batch.cards` result is rendered without additions, truncation, or fallback categories.
- Named the owner-facing card contract across all platforms: decision, why-now evidence, Queen recommendation, choice consequences, affected semantic IDs, prior-answer revalidation, planning-resume condition, and exact answer syntax.
- Bound every answer to the exact runtime-issued decision ID, required a fresh Discuss result after each answer, and documented the full goal/session/specification/base-plan equivalence boundary for answer reuse.
- Routed settled intent to the returned draft or retained approved specification and `/ant-spec` (or `aether spec` in Codex), keeping exact-revision approval separate from discussion and planning.
- Rewrote the public lifecycle contract to cover current flags, evidence ownership, structured outputs, draft-spec transactions, projections, receipts, and state effects.
- Updated only the Discuss portion of the combined research skill; the Oracle interview, template, depth, confidence, background, fallback, and status behavior remains intact.
- Added semantic parity coverage for the canonical YAML, both Claude entry points, OpenCode wrapper, public contract, research skill, Oracle anchors, and live Cobra flag help.

## Task Commits

1. **Task 1: Align evidence-first Discuss wrappers** — `6fb3c0e9` (feat)
2. **Task 2: Codify public Discuss and research-skill contract** — `3b9c5e1e` (feat)

## Files Created/Modified

- `.aether/commands/discuss.yaml` — Canonical runtime-first ceremony, material-batch, answer-resume, and draft-spec routing contract.
- `.claude/commands/ant-discuss.md` — Root Claude shortcut rendering the structured Go result.
- `.claude/commands/ant/discuss.md` — Canonical Claude wrapper with the same render-only contract.
- `.opencode/commands/ant/discuss.md` — OpenCode wrapper aligned with the Claude and YAML-owned semantics.
- `cmd/contracts/discuss.md` — Current flags, evidence frontier, card fields, equivalence scope, state effects, and specification handoff.
- `.aether/skills/colony/aether-colony-research/SKILL.md` — Codex Discuss flow using runtime-issued cards and `aether spec`, with Oracle behavior preserved.
- `cmd/discuss_wrapper_contract_200_test.go` — Cross-platform semantic, ownership, Oracle-preservation, and runtime-help regression test.

## Decisions Made

- Wrapper intelligence stops at presentation. Go remains the only authority for evidence admission, decision materiality, answer reuse, persistence, specification creation, receipts, and next-command truth.
- A complete decision batch has no fixed numerical quota. The legacy `--max-questions` flag remains documented as compatibility-only because the runtime deliberately returns every material clarification.
- A platform-native question card may display the runtime's choices, but it cannot change the card or record anything without an explicit owner selection sent through the exact answer command.
- Negative guidance may name `/ant-plan` only to forbid the shortcut; semantic tests reject affirmative settled-to-plan routing while requiring the explicit prohibition.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

- The installed requirement updater does not parse this milestone's bold-ID checkbox format. `CEC-03`, `PLAN-04`, and `PLAN-05` were already complete; `SYNTH-02` was checked manually after the handler reported all four IDs as not found.
- The installed state SDK expects named flags for metrics and decisions despite the executor reference's positional examples. The calls were rerun with the locally documented signatures and succeeded. `state.update-progress` found no Markdown progress field, matching prior Phase 200 plans; roadmap plan counts advanced normally.

## Verification

- `go test ./cmd -run 'TestDiscussWrapperContract200|TestDiscuss.*DraftSpec|Test.*ResearchSkill' -count=1` — 5 passed.
- `go run ./cmd/aether source-check` — passed all 16 canonical-source, 5 retired-mirror, and 124 generated-wrapper checks with no findings.
- `git diff --check` — passed before both task commits.
- Stub scan across all seven plan files — no functional stubs; the only textual match was the intentional Go slice declaration `[]string` in the regression test.

## Known Stubs

None.

## User Setup Required

None - no external service or local configuration is required.

## Next Phase Readiness

- Plan 200-21 can build on one documented runtime-authority boundary when aligning the remaining public planning wrappers.
- Plans 200-23 through 200-25 can use the semantic parity test pattern for end-to-end proof and wrapper rollout.
- No unresolved issues block the next plan.

## Self-Check: PASSED

- All seven plan-owned wrapper, documentation, skill, and test files plus this summary exist.
- Both task commit hashes are present in repository history.

---
*Phase: 200-iterative-planning*
*Completed: 2026-09-08*
