---
phase: 164-research-feeds-planning
plan: 09
subsystem: cli (Go runtime + Claude/OpenCode plan wrappers)
tags: [go, plan-depth, granularity, verification-depth, research-decision, queen-proposal, wrapper-parity]

# Dependency graph
requires:
  - phase: 164-04
    provides: "computeDepthProposal / renderDepthProposalCard — the three-knob depth proposal card computation and rendering"
  - phase: 164-05
    provides: "aether plan-research-approve, the four research_* manifest/result fields, and dispatch gated on an approved map[int]bool"
  - phase: 164-07
    provides: "Route-Setter brief carrying research content and research_failed_phases/research_warning on finalize"
provides:
  - "cmd/codex_plan.go: both plan-only result branches (existing-plan early return, fresh-plan) and codexPlanManifest now carry depth_proposal and depth_proposal_card alongside the existing research_* fields"
  - ".claude/commands/ant/plan.md, .opencode/commands/ant/plan.md, .claude/commands/ant-plan.md: the static Depth Ceremony/Planning Depth ceremonies are replaced with Decision Moment 1 (depth proposal card) and Decision Moment 2 (research batch card), both printed verbatim from runtime keys"
  - ".aether/commands/plan.yaml: wrapper_additions carries depth_proposal_card and research_batch_card in place of depth_ceremony/planning_depth_ceremony; runtime.command includes --verification-depth"
  - "cmd/plan_wrapper_cards_test.go: TestPlanWrapperCardsParity — an ordered-heading-set equality invariant across the two wrappers, not just named-section presence"
affects: []

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Runtime-computed card, wrapper-printed verbatim: the wrapper never composes its own recommendation text for either decision moment, matching the wrapper-runtime UX contract already established for the research batch card in Plan 05"
    - "Ordered-heading-set equality as a parity invariant (CLAUDE.md Definition of Done: prefer an invariant over a named-section-exists check) — a rename or one-sided addition to either wrapper now fails the suite instead of silently drifting"

key-files:
  created:
    - cmd/plan_depth_proposal_manifest_test.go
    - cmd/plan_wrapper_cards_test.go
  modified:
    - cmd/codex_plan.go
    - cmd/command_guide.go
    - cmd/plan_wrapper_ceremony_test.go
    - cmd/platform_doc_hygiene_test.go
    - .claude/commands/ant/plan.md
    - .claude/commands/ant-plan.md
    - .opencode/commands/ant/plan.md
    - .aether/commands/plan.yaml

key-decisions:
  - "Named the local variable `proposal`/`proposalCard` in runCodexPlanPlanOnly rather than the plan's literal suggestion (which would have shadowed the `depthProposal` type name from Plan 04) — same values, no naming collision"
  - "Placed Decision Moment 2 (research batch) between the Planning Manifest section and the Clarification Gate, rather than immediately before Worker Spawning, because the research batch answer needs a fresh manifest fetch before the Clarification Gate's boundary-question inspection is meaningful; this still satisfies the plan's 'after the manifest section and before worker spawning' placement instruction"
  - "Mirrored every wrapper edit into the flat legacy `.claude/commands/ant-plan.md` file (not listed in the plan's files_modified) because a pre-existing test, TestPlanWrapperCeremonyContract, checks that file alongside the two canonical `ant/plan.md` wrappers and it was byte-for-byte identical to the Claude wrapper before this plan touched either"
  - "Updated cmd/command_guide.go's Codex plan pre-steps (the line describing 'select planning depth ... with the user') because it directly contradicted the new no-typing, card-based flow — the plan's own instruction authorized this: 'update cmd/command_guide.go only if it contradicts the new two-card flow'"

patterns-established:
  - "Two-card plan flow: exactly two tap-to-approve decision moments (depth proposal, research batch), enforced by a wrapper guardrail in both platforms and a Go test asserting ordered heading-set parity"

requirements-completed: [RESEARCH-01, RESEARCH-09, RESEARCH-10]

# Metrics
duration: ~70min
completed: 2026-08-02
---

# Phase 164 Plan 09: Two Decision Moments, Runtime-Rendered Summary

**The plan-only manifest now carries a three-knob depth proposal alongside the existing research proposal, and both Claude/OpenCode wrappers present exactly two runtime-rendered, tap-to-approve cards — Decision Moment 1 (depth) and Decision Moment 2 (research) — replacing the old static Depth Ceremony/Planning Depth Q&A.**

## Performance

- **Duration:** ~70 min
- **Started:** 2026-08-02T13:00:00Z (approx, worktree wave 4 start)
- **Completed:** 2026-08-02T13:18:14Z
- **Tasks:** 3 completed
- **Files modified:** 10 (2 created, 8 modified)

## Accomplishments

- Both `runCodexPlanPlanOnly` result branches (existing-plan early return and fresh-plan) and `codexPlanManifest` carry `depth_proposal` and `depth_proposal_card`, computed once via Plan 04's `computeDepthProposal`/`renderDepthProposalCard` and reused for both response shapes
- `.claude/commands/ant/plan.md` and `.opencode/commands/ant/plan.md` present exactly two decision moments: Decision Moment 1 prints `result.depth_proposal_card` verbatim and accepts on one confirmation or a named knob+option; Decision Moment 2 prints `result.research_proposal_card` verbatim and resolves via `aether plan-research-approve --approve-all` / `--flip <ids>` / (autopilot) `--auto`
- Three new guardrails — exactly two decision moments, no `aether discuss` routing for either card, no wrapper-composed recommendations — added to both wrappers and `.aether/commands/plan.yaml`
- `TestPlanWrapperCardsParity` asserts the ordered `## ` heading slices from the two wrappers are deeply equal (not just that named sections exist), that both reference all four runtime card keys, that neither still contains the retired headings, and that `plan.yaml` carries both card entries
- A pre-existing test (`TestPlanWrapperCeremonyContract`) and a hygiene test (`TestLifecycleCommandDocsPreferRuntimeCLI`) that pinned the *old* Depth Ceremony contract were updated to assert the new contract — both would otherwise have failed as a direct, in-scope consequence of Task 2's edit

## Task Commits

Each task was committed atomically:

1. **Task 1: The manifest carries the depth proposal** - `b8ac45b5` (feat)
2. **Task 2: Two decision moments in both wrappers and the YAML source** - `87b5c18e` (feat)
3. **Task 3: Parity invariant for the two cards** - `87064ad2` (test)

## Files Created/Modified

- `cmd/codex_plan.go` - `depth_proposal`/`depth_proposal_card` added to both plan-only result maps and to `codexPlanManifest` (new `DepthProposal`/`DepthProposalCard` fields), computed once as `proposal`/`proposalCard` after `planningSmartDefault`/`verificationSmartDefault` are resolved
- `cmd/plan_depth_proposal_manifest_test.go` - `TestPlanManifestCarriesDepthProposal`: fresh-plan branch, existing-plan branch, recommendation parity with `result.granularity`/`planning_depth`/`verification_depth`, accept-line contents, and explicit-vs-smart-default reason text
- `.claude/commands/ant/plan.md`, `.opencode/commands/ant/plan.md`, `.claude/commands/ant-plan.md` - `## Depth Ceremony` and `## Planning Depth` replaced with `## Decision Moment 1 — Depth Proposal` and `## Decision Moment 2 — Research Batch`; three new guardrails appended
- `.aether/commands/plan.yaml` - `wrapper_additions.depth_ceremony`/`planning_depth_ceremony` replaced with `depth_proposal_card`/`research_batch_card`; `runtime.command` gained `--verification-depth`; `iterative_planning_ceremony` notes that `--target`/`--max-iterations` also bound per-phase research iteration; three matching guardrails added
- `cmd/command_guide.go` - the Codex `plan` pre-steps rewritten to describe requesting-then-printing the depth proposal card (Decision Moment 1) and the research batch card (Decision Moment 2) instead of asking the user directly, and the `aether host plan` example gained `--verification-depth`
- `cmd/plan_wrapper_cards_test.go` - `TestPlanWrapperCardsParity`: ordered-heading-set deep-equality, four-runtime-key coverage, retired-heading absence, exactly-one `--approve-all`/`--auto` instruction each, `plan.yaml` card-entry presence, and a self-verifying "mutated heading breaks equality" sub-test
- `cmd/plan_wrapper_ceremony_test.go` - `TestPlanWrapperCeremonyContract`'s required/in-order/forbidden lists updated from the old Depth Ceremony contract to the new two-decision-moment contract
- `cmd/platform_doc_hygiene_test.go` - the `.claude/commands/ant/plan.md` and `.opencode/commands/ant/plan.md` test cases in `TestLifecycleCommandDocsPreferRuntimeCLI` updated: require both new headings, forbid the two retired ones

## Decisions Made

See `key-decisions` in frontmatter. In short: avoided a variable/type name collision by naming the computed proposal `proposal`/`proposalCard` rather than `depthProposal`; placed Decision Moment 2 between Planning Manifest and Clarification Gate (still "after the manifest, before worker spawning" per the plan's instruction); mirrored the wrapper edit into the flat legacy `ant-plan.md` file because a pre-existing test checks it alongside the canonical wrapper and it was previously kept byte-identical; updated `command_guide.go`'s Codex pre-steps because the old text directly contradicted the new no-typing flow, which the plan's action text explicitly authorized.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Two pre-existing tests pinned the old Depth Ceremony contract and broke as a direct result of Task 2's edit**
- **Found during:** Task 2 (full `go test ./cmd/... -count=1` run after replacing the ceremonies)
- **Issue:** `TestPlanWrapperCeremonyContract` (`cmd/plan_wrapper_ceremony_test.go`) and one case each for `.claude/commands/ant/plan.md`/`.opencode/commands/ant/plan.md` inside `TestLifecycleCommandDocsPreferRuntimeCLI` (`cmd/platform_doc_hygiene_test.go`) both required the literal string `"## Depth Ceremony"` to be present — the exact heading Task 2's action text explicitly instructs deleting
- **Fix:** Updated both tests' required/forbidden/in-order string lists to assert the new `## Decision Moment 1 — Depth Proposal` / `## Decision Moment 2 — Research Batch` contract and forbid the retired headings, matching the same shape the plan's own Task 3 acceptance criteria describe for the new parity test
- **Files modified:** `cmd/plan_wrapper_ceremony_test.go`, `cmd/platform_doc_hygiene_test.go`
- **Verification:** `go test ./cmd/... -run 'TestPlanWrapperCeremonyContract|TestLifecycleCommandDocsPreferRuntimeCLI' -count=1` passes; full `go test ./cmd/... -count=1` passes (262s)
- **Committed in:** `87b5c18e` (Task 2 commit)

**2. [Rule 3 - Blocking] The flat legacy `.claude/commands/ant-plan.md` file was also checked by `TestPlanWrapperCeremonyContract` and drifted out of sync**
- **Found during:** Task 2, same test run as above
- **Issue:** `TestPlanWrapperCeremonyContract` checks three wrapper paths, not two: `.claude/commands/ant/plan.md`, `.opencode/commands/ant/plan.md`, and a flat `.claude/commands/ant-plan.md` that is not under the `ant/` subdirectory and is not listed in this plan's `files_modified`. It was byte-identical to the canonical Claude wrapper before this plan and would have failed the same required-string checks once Task 2 edited only the canonical wrapper
- **Fix:** Mirrored the identical Task 2 content into `.claude/commands/ant-plan.md`
- **Files modified:** `.claude/commands/ant-plan.md`
- **Verification:** `diff .claude/commands/ant/plan.md .claude/commands/ant-plan.md` reports no differences; `TestPlanWrapperCeremonyContract` passes
- **Committed in:** `87b5c18e` (Task 2 commit)

**3. [Rule 1 - Bug] `cmd/command_guide.go`'s Codex plan pre-steps still described asking the user directly**
- **Found during:** Task 2, verifying `aether command-guide plan --platform codex` "still describes a matching flow after the edit" per the plan's explicit action text
- **Issue:** The pre-step `"Select planning depth and decomposition depth with the user unless arguments already specify them."` contradicted the new zero-typing, card-based flow, and the `aether host plan` example lacked `--verification-depth`
- **Fix:** Rewrote the pre-step to describe requesting a manifest, printing `result.depth_proposal_card` verbatim, and accepting on confirmation or a named knob+option (mirroring Decision Moment 1's wrapper text); added an equivalent pre-step describing Decision Moment 2's research-batch resolution; added `--verification-depth` to the example command
- **Files modified:** `cmd/command_guide.go`
- **Verification:** `go run ./cmd/aether command-guide plan --platform codex` exits 0 and its `pre_steps` now describe both decision moments; no golden-snapshot fixture pins the exact pre_steps text (checked via `grep -rln '"plan"' cmd/testdata/*.json`), so no golden update was required
- **Committed in:** `87b5c18e` (Task 2 commit)

**4. [Rule 3 - Blocking] `.aether/ts-host` had no installed `node_modules` in this worktree**
- **Found during:** final `<verification>` step, `npm --prefix .aether/ts-host run test:all`
- **Issue:** The worktree checkout never had `npm install` run for the TS host (node_modules is gitignored, as expected); 17 of 68 suites failed with `Error [ERR_MODULE_NOT_FOUND]: Cannot find package 'log-update'` and similar — an environment setup gap, not a code defect introduced by this plan
- **Fix:** Ran `npm install` inside `.aether/ts-host` (installs exactly the dependencies already declared in `package.json`; no `package.json` or `package-lock.json` change)
- **Files modified:** none (node_modules is gitignored; `git status --short` confirms no tracked-file changes from the install)
- **Verification:** `npm --prefix .aether/ts-host run test:all` now reports 548/548 passing, 0 failing
- **Committed in:** not applicable — no tracked files changed

---

**Total deviations:** 4 auto-fixed (3 Rule 3 - blocking, 1 Rule 1 - bug)
**Impact on plan:** All four were necessary consequences of Task 2's intentional contract change (or, for #4, a pre-existing environment gap unrelated to any task). No scope creep: no production behavior changed beyond what the plan's own action text specified or explicitly authorized ("update `cmd/command_guide.go` only if it contradicts the new two-card flow").

## Issues Encountered

None beyond the four auto-fixed items documented above.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- `go test ./... -count=1` passes (Go + all `pkg/` packages)
- `go build ./cmd/aether` succeeds
- `go run ./cmd/aether command-guide plan --platform codex` exits 0
- `npm --prefix .aether/ts-host run test:all` passes (548/548)
- RESEARCH-01, RESEARCH-09, and RESEARCH-10 are now fully satisfied: the depth proposal is reachable and explained with a plain-English reason per knob at plan time, and the research decision is visible, overridable, and gated to exactly two tap-to-approve decision moments
- This is the final plan (09 of 9, wave 4) of Phase 164 (research-feeds-planning). No blockers for phase close-out.

## Self-Check: PASSED

- FOUND: cmd/codex_plan.go (modified)
- FOUND: cmd/plan_depth_proposal_manifest_test.go
- FOUND: cmd/command_guide.go (modified)
- FOUND: cmd/plan_wrapper_ceremony_test.go (modified)
- FOUND: cmd/platform_doc_hygiene_test.go (modified)
- FOUND: cmd/plan_wrapper_cards_test.go
- FOUND: .claude/commands/ant/plan.md (modified)
- FOUND: .claude/commands/ant-plan.md (modified)
- FOUND: .opencode/commands/ant/plan.md (modified)
- FOUND: .aether/commands/plan.yaml (modified)
- FOUND commit b8ac45b5: feat(164-09): manifest carries the three-knob depth proposal
- FOUND commit 87b5c18e: feat(164-09): two runtime-rendered decision moments in plan wrappers
- FOUND commit 87064ad2: test(164-09): parity invariant pins the two-card plan flow

---
*Phase: 164-research-feeds-planning*
*Completed: 2026-08-02*
