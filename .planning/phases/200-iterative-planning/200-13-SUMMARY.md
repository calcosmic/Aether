---
phase: 200-iterative-planning
plan: 13
subsystem: iterative-planning
tags: [go, planning-presets, staged-dispatch, evidence-catalogue, autonomous-research, tdd]

# Dependency graph
requires:
  - phase: 200-iterative-planning
    plan: 08
    provides: Immutable specification revisions, exact approval receipts, and deterministic SPEC projection
  - phase: 200-iterative-planning
    plan: 09
    provides: Typed planning-stage state, manifests, evidence frontiers, and durable dispatch boundaries
provides:
  - Exact Fast, Balanced, Deep, and Exhaustive planning presets with no implicit default
  - Approved-specification and projection gate before any fresh planning dispatch
  - One durable, evidence-grounded Scout-stage manifest bound to run, pass, preset, specification, base plan, and evidence frontier
  - Deterministic autonomous phase-research policy with typed Scout attribution and safe source rejection
affects: [200-14-scout-finalization, 200-15-route-finalization, 200-16-candidate-acceptance, 200-19-planning-renderers, 200-22-platform-parity]

# Tech tracking
tech-stack:
  added: []
  patterns: [closed preset resolver, approved-spec dispatch gate, single-stage authorization, typed research evidence, deterministic research policy]

key-files:
  created: []
  modified:
    - cmd/codex_plan.go
    - cmd/codex_plan_test.go
    - cmd/codex_workflow_cmds.go
    - cmd/phase_research_decision.go
    - cmd/phase_research_decision_test.go
    - cmd/phase_research_dispatch_test.go
    - cmd/plan_depth_proposal_manifest_test.go

key-decisions:
  - "Planning starts only after the owner selects one exact preset or supplies the complete matching confidence-target/pass-cap pair; an unflagged invocation is a zero-dispatch selection result."
  - "The public planning boundary authorizes exactly one Scout stage and persists its run header before any Route-Setter, score, candidate, acceptance, or activation authority can exist."
  - "Routine phase research is selected deterministically from preset, weakest gap, and evidence freshness; research remains Scout-attributed evidence and can surface, but never answer, material owner decisions."
  - "Repository research input is accepted only through contained regular-file paths and is rejected when unavailable, out of scope, symlinked, or secret-bearing."

patterns-established:
  - "Closed preset pattern: normalize both named presets and explicit pairs into the same immutable target/pass-cap policy before state mutation."
  - "Stage-first planning pattern: bind exact approved inputs into a durable run header, then issue only the next legal worker authorization."
  - "Authority-neutral research pattern: discover and hash Scout-authored research as typed evidence with separate producer attribution; evidence cannot self-approve a specification or plan."

requirements-completed: [PLAN-01, PLAN-02, PLAN-04, PLAN-05, PLAN-06]

# Metrics
duration: 55 min
completed: 2026-09-07
---

# Phase 200 Plan 13: Explicit Presets and Scout-First Planning Summary

**Fresh planning now requires an exact owner-selected preset and approved specification, then starts one evidence-bound Scout stage while routine research runs autonomously under typed, attributable safeguards.**

## Performance

- **Duration:** 55 min
- **Started:** 2026-09-07T18:35:06Z
- **Completed:** 2026-09-07T19:29:46Z
- **Tasks:** 3
- **Files modified:** 7 plan-owned code/test files

## Accomplishments

- Replaced ambiguous depth defaults with the exact Fast 80/4, Balanced 90/6, Deep 95/8, and Exhaustive 99/12 policies. Unflagged planning now returns all four choices without selecting one or writing a planning run; complete exact target/cap pairs remain a non-interactive equivalent.
- Rejected partial, conflicting, unknown, and non-preset target/cap combinations, and made legacy `--accept` fail before state access with exact-candidate acceptance guidance rather than silently accepting a below-target plan.
- Required an exact approved specification and matching deterministic projection before fresh planning can dispatch, with recovery directed through `aether spec` when the binding is missing, draft, stale, superseded, or hash-mismatched.
- Primed specification, survey, charter, resolved decision, context, research, Hive, verified outcome, and prior Scout-research evidence into a typed catalogue; missing categories remain explicit gaps rather than invented confidence.
- Persisted a planning-run header binding goal/session, specification revision/hash, base plan revision/hash, normalized preset, weakest gap, evidence frontier, and the first Scout authorization.
- Reduced the public plan-only/agent-delegate envelope to one Scout worker and one stage manifest. It contains no Route-Setter output slot, score, candidate, acceptance, or activation authority.
- Replaced routine phase-research approval prompts with a deterministic preset/gap/freshness policy. Worker-authored research is normalized into typed evidence with Scout attribution, source origin/revision, observation time, applicability, and content hash.
- Preserved fail-loudly behavior for unavailable research and added containment, regular-file/symlink, scope, and privacy checks for repository research sources.

## Task Commits

Each TDD task was committed atomically with RED before GREEN; compatibility corrections were committed separately:

1. **Task 1: Resolve the four exact presets without an implicit default** — `397f7ae8` (test/RED), `182dc829` (feat/GREEN)
2. **Task 2: Gate on approved spec, prime evidence, and emit one Scout manifest** — `9bfdb6de` (test/RED), `76132236` (feat/GREEN)
3. **Task 3: Make phase research autonomous and attributable** — `39e3bbdd` (test/RED), `69e3b57f` (feat/GREEN)
4. **Cross-task fixture and contract corrections** — `890e18f4` (test), `f6ce9eb3` (fix)

**Plan metadata:** committed separately after state synchronization.

## Files Created/Modified

- `cmd/codex_plan.go` — Closed preset resolution, approved-spec gate, evidence priming, durable run header, Scout-only public manifest, and autonomous research-policy/result fields.
- `cmd/codex_plan_test.go` — Exact preset, spec-gate, evidence, Scout-stage, compatibility, direct-runtime, synthetic, and fail-closed regression coverage.
- `cmd/codex_workflow_cmds.go` — Public preset and explicit target/pass-cap flags plus non-mutating legacy-accept rejection plumbing.
- `cmd/phase_research_decision.go` — Deterministic automatic research policy, worker-authored research discovery, typed evidence normalization, Scout attribution, and source safeguards.
- `cmd/phase_research_decision_test.go` — Automatic needed/not-needed, preset/gap/freshness, attribution, material-decision, and unsafe-source tests.
- `cmd/phase_research_dispatch_test.go` — Migrated autonomous-research tests while retaining bounded injection, dispatch-once, failure visibility, source preservation, and re-research coverage.
- `cmd/plan_depth_proposal_manifest_test.go` — Four exact choices, no recommendation/default, zero unflagged dispatch, explicit bypass, and single-Scout manifest contract.

## Decisions Made

- A preset is a closed execution policy, not presentation text: every accepted input resolves to one of four exact target/pass-cap pairs before a planning run can be created.
- Selection and authority are separate. Returning the four preset choices is read-only; legacy `--accept` cannot choose a preset, stop an iteration, accept a candidate, or activate a revision.
- Fresh planning is specification-bound. Existing accepted plans may still be inspected without inventing a new specification dependency, but any new or refreshed run must bind the exact approved revision and projection.
- The first public planning envelope is intentionally incomplete: one Scout is authorized now, and a later receipt-bound finalizer must decide whether Route-Setter or an owner-decision boundary follows.
- Research depth is runtime policy rather than an owner checkpoint. The Scout may collect warranted in-scope evidence, but product ambiguity remains an `owner_decision` after the Scout pass.
- The deprecated phase-research approval data shape remains readable for legacy rows, but no active `aether plan` path emits a proposal card, approval prompt, or awaiting-approval state.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Bound the synthetic regression fixture to staged prerequisites**

- **Found during:** Task 3 verification
- **Issue:** A retained synthetic regression still created a fresh planning state without the approved specification that Task 2 now correctly requires.
- **Fix:** Added the exact approved specification fixture and deterministic projection without weakening the test's original synthetic-planning assertion.
- **Files modified:** `cmd/codex_plan_test.go`
- **Verification:** Focused compatibility test and the 35-test aggregate planning gate pass.
- **Committed in:** `890e18f4`

**2. [Rule 1 - Bug] Corrected the public dispatch contract's worker count and dependency wording**

- **Found during:** Overall compatibility verification
- **Issue:** The manifest contained one Scout, but the reused generic contract still described a two-worker Scout/Route-Setter wave, overstating present authority.
- **Fix:** Added a Scout-stage contract that reports one worker/one wave and states that Route-Setter requires a later receipt-bound manifest.
- **Files modified:** `cmd/codex_plan.go`, `cmd/codex_plan_test.go`
- **Verification:** Dispatch-contract, plan-only, and agent-delegate fixtures pass; normal and race aggregate gates pass 35/35.
- **Committed in:** `f6ce9eb3`

**3. [Rule 3 - Blocking] Migrated retained command fixtures to the new start contract**

- **Found during:** Overall compatibility verification
- **Issue:** Older direct-runtime, synthetic, recovery, depth, failure, and wrapper tests invoked fresh planning without an explicit preset or approved specification, so they returned at the new intentional gates before exercising their original behavior.
- **Fix:** Supplied exact presets and approved-spec/projection fixtures, changed the non-preset 90/4 pair to Balanced 90/6, and replaced the obsolete whole-chain wrapper assertion with the Scout-first/no-activation boundary.
- **Files modified:** `cmd/codex_plan_test.go`
- **Verification:** All 14 migrated compatibility tests and `go vet ./cmd` pass.
- **Committed in:** `f6ce9eb3`

---

**Total deviations:** 3 auto-fixed (1 bug, 2 blocking fixture migrations)
**Impact on plan:** The fixes made the new authority boundary truthful and retained existing regression coverage without broadening runtime scope.

## Issues Encountered

- The exact task gates pass 12/12, 9/9, and 19/19; the required aggregate passes 35/35 in both normal and race-detector runs.
- The repository-wide command-package run reached the three-attempt auto-fix limit with 29 failures outside Plan 13 ownership or intentionally scheduled for later Phase 200 plans. They cover the Plan 14 legacy whole-chain finalizer fixtures, future candidate flags and command/catalog/schema ratchets, Plan 19 visual/golden migration, lifecycle-card/orchestrator-boundary fixtures that still assume an implicit planning default, and Phase 199 vocabulary drift caused by an unstaged external planning artifact. These files were not modified.
- The dirty user-owned `.planning/config.json`, `.gsd/`, and `.planning/phases/199-front-door-and-classic-contract/199-PATTERNS.md` remained unstaged and uncommitted throughout execution.

## TDD Gate Compliance

- Task 1 RED (`397f7ae8`) introduced the exact four-preset/no-default/legacy-accept contract; GREEN (`182dc829`) implemented the resolver and command flags.
- Task 2 RED (`9bfdb6de`) introduced approved-spec, evidence-frontier, and Scout-only manifest failures; GREEN (`76132236`) implemented the staged planning start.
- Task 3 RED (`39e3bbdd`) replaced approval-gated research expectations with autonomous policy, typed evidence, and safety failures; GREEN (`69e3b57f`) implemented those contracts.
- All three RED commits precede their corresponding GREEN commits. Later compatibility commits preserve, rather than replace, the RED/GREEN gates.

## Known Stubs

None. The modified files contain no TODO, FIXME, placeholder, coming-soon, or UI-flowing hardcoded-empty implementation that blocks the plan goal.

## Threat Flags

| Flag | File | Description |
|------|------|-------------|
| threat_flag: local-research-file-ingestion | `cmd/phase_research_decision.go` | Scout-authored phase-research files become planning evidence. Discovery is restricted to canonical `phase-N-research.md` regular files under `.aether/data/phase-research`, rejects symlinks/path escape, and privacy-scans bytes before admission. |
| threat_flag: configured-research-document-ingestion | `cmd/codex_plan.go` | Owner-configured repository research documents are read into the initial evidence frontier only after existing repository-path validation and are rejected when unreadable or secret-bearing. |

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Plan 200-14 can finalize the exact Scout manifest and receipt now persisted here, then authorize either the first material owner-decision batch or the first Route-Setter stage.
- Plan 200-15 can consume the later Route-Setter authorization without changing the preset, specification, base-plan, or evidence identities established at run start.
- Plans 200-16 and 200-19 still need to implement candidate commands and migrate renderer/catalog fixtures before the full repository suite reflects the restored journey.
- No Plan 200-13 implementation blocker remains; all plan-owned normal and race gates are green.

## Self-Check: PASSED

- All seven plan-owned code/test files and this summary exist; all eight task/compatibility commits are present in repository history in the documented order.
- The exact task gates pass 12/12, 9/9, and 19/19; the required aggregate passes 35/35 normally and 35/35 with `-race`; `go vet ./cmd` is clean.
- Unflagged-selection tests prove zero worker dispatch and no planning-run write, while approved-start tests prove exactly one Scout authorization bound to the normalized preset, approved specification, base revision, evidence frontier, and expected result contract.
- Stub scanning found no goal-blocking placeholder implementation. Threat scanning identified the two planned local research-ingestion surfaces and confirmed their containment, regular-file, symlink, availability, and privacy controls.
- Commit deletion checks found no tracked-file deletion, and the three unrelated dirty paths remain unstaged.

---
*Phase: 200-iterative-planning*
*Completed: 2026-09-07*
