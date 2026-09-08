---
phase: 200-iterative-planning
verified: 2026-09-08T06:16:14Z
status: gaps_found
score: 47/64 must-haves verified
overrides_applied: 0
gaps:
  - truth: "Only the exact owner-approved specification can ground planning and execution."
    status: failed
    reason: "Specification validation checks stored digest syntax and cross-field equality but never recomputes item, revision, or approval-receipt digests from the stored body. Approved content can therefore be edited while retaining accepted identifiers."
    artifacts:
      - path: "cmd/planning_state.go"
        issue: "validateSpecificationState and validateSpecificationRevisionHashes validate stored hash shape, not canonical body equality."
      - path: "pkg/colony/specification.go"
        issue: "SpecRevision.Validate and SpecApprovalReceipt.Validate only validate structure and stored bindings."
      - path: "cmd/plan_authority.go"
        issue: "Execution authority trusts those unrecomputed stored values."
    missing:
      - "Recompute every typed item digest and the complete specification revision digest during state validation."
      - "Recompute approval receipt identity/token bindings and reject an edited approved body before plan or build authority is granted."
      - "Add post-approval tamper tests for every owner-readable specification section."
  - truth: "Candidate acceptance binds the exact semantic delta used to reconcile changed specification scope."
    status: failed
    reason: "The candidate content hash excludes SemanticDelta, and nested delta validators only check stored hash syntax/ID suffixes. An edited candidate can claim affected semantic IDs and pass impact coverage without changing the owner acceptance binding."
    artifacts:
      - path: "cmd/codex_plan_finalize.go"
        issue: "Candidate hash input at lines 4084-4094 omits SemanticDelta, assessments, gaps, recommendation, and expiry."
      - path: "cmd/planning_state.go"
        issue: "validatePlanningDeltaShape does not recompute delta/change/authority-impact bodies."
      - path: "cmd/plan_impact.go"
        issue: "Impact coverage trusts mutable candidate.SemanticDelta values."
      - path: "cmd/plan_revision.go"
        issue: "Acceptance recomputes proposal phases but not the semantic delta used for reconciliation."
    missing:
      - "Bind all security-relevant candidate fields into a recomputed candidate digest."
      - "Derive or recompute reconciliation coverage from the exact base/current-spec/proposal tuple at acceptance."
      - "Add persisted-candidate semantic-delta tamper/refusal tests."
  - truth: "Planning and specification writes cannot escape the repository through an intermediate symlink."
    status: failed
    reason: "Root validation rejects only the final root component. An intermediate .aether symlink is followed by Lstat/EvalSymlinks, and no physical containment comparison is made before MkdirAll or lifecycle writes."
    artifacts:
      - path: "cmd/planning_timeline.go"
        issue: "canonicalPlanningTimelineRoot validates only the repository root; append then joins and creates .aether/data."
      - path: "cmd/planning_stage_receipt.go"
        issue: "planningStageRoots calls MkdirAll on a joined data root without validating intermediate components."
      - path: "cmd/specification.go"
        issue: "canonicalSpecificationRoot Lstats .aether/data but follows an intermediate .aether symlink."
      - path: "cmd/lifecycle_transaction.go"
        issue: "validateLifecycleDirectoryRoot resolves ancestors but does not prove data root containment beneath the physical repository root."
    missing:
      - "Resolve repository, .aether, and data roots physically and prove containment before any directory creation or write."
      - "Reject intermediate symlink components and add .aether/data path-escape tests for all Phase 200 writers."
  - truth: "Concurrent planning mutations preserve every committed update or reject a stale writer."
    status: failed
    reason: "Specification, timeline, candidate-acceptance, and build paths derive next state before entering their transaction/lock. The transaction captures the then-current file as its baseline, so a stale derived snapshot can overwrite a writer that committed between the initial read and transaction declaration. Several build transitions still use direct SaveJSON."
    artifacts:
      - path: "cmd/specification.go"
        issue: "State is loaded and next bytes are derived before commitSpecificationTargets opens its lifecycle transaction."
      - path: "cmd/planning_timeline.go"
        issue: "The index is read/extended before beginLifecycleTransaction."
      - path: "cmd/plan_revision.go"
        issue: "Acceptance loads and derives state at lines 799-959, then opens the transaction at line 964."
      - path: "cmd/codex_build.go"
        issue: "Build reads state then writes checkpoint/current state with direct SaveJSON."
    missing:
      - "Hold one repository-scoped lock from authoritative read through commit, or compare-and-swap against the digest of the state used to derive the write."
      - "Route remaining build state transitions through the same stale-writer guard."
      - "Add separate-process concurrency tests for spec revision/approval, timeline append, candidate acceptance, and build start/finalize."
  - truth: "A plan candidate cannot be accepted after its declared expiry, and the owner can see that boundary."
    status: failed
    reason: "PlanCandidate requires ExpiresAt and creation sets seven days, but acceptance never compares time with ExpiresAt or transitions pending_review to expired. The terminal candidate projection also omits expiry."
    artifacts:
      - path: "pkg/colony/planning.go"
        issue: "ExpiresAt is presence-validated only."
      - path: "cmd/plan_revision.go"
        issue: "acceptPlanCandidate accepts any pending_review candidate regardless of ExpiresAt."
      - path: "cmd/planning_visuals.go"
        issue: "planningCandidateProjection and terminal review expose no expiry field or recovery action."
    missing:
      - "Enforce an injected/current-time expiry boundary before acceptance and persist/refuse with an explicit expired status."
      - "Render the expiry and exact regeneration/review action in terminal and JSON projections."
      - "Test just-before, exact-boundary, just-after, replay, and clock-source cases."
  - truth: "The finished repository passes the targeted, full, and race suites with an auditable gate receipt."
    status: failed
    reason: "The independent uncached full suite has four failures. All are protected inherited Phase 199 bookkeeping nodes, not Phase 200 behavior regressions, but the Phase 200 plan's repository-wide green-gate truth is still not met."
    artifacts:
      - path: "cmd/current_vocabulary_199_test.go"
        issue: "TestCurrentVocabulary199 and one subtest fail on inherited bookkeeping."
      - path: "cmd/phase199_gate_receipt_test.go"
        issue: "TestPhase199GateReceiptSchema and TestPhase199GateReceipt fail."
      - path: ".planning/phases/200-iterative-planning/200-GATE-RECEIPT.md"
        issue: "Receipt honestly records a non-green inherited baseline rather than a clean repository gate."
    missing:
      - "Resolve or formally waive the protected Phase 199 vocabulary/gate-receipt bookkeeping, then rerun uncached full and race suites."
---

# Phase 200: Iterative Planning Verification Report

**Phase Goal:** Restore the visible Scout → Route-Setter research and confidence loop over the modern plan revision/finalizer.

**Verified:** 2026-09-08T06:16:14Z

**Status:** `gaps_found`

**Re-verification:** No — initial verification; no earlier `200-VERIFICATION.md` existed.

## Executive Verdict

The visible iterative-planning loop is real, substantive, and wired: focused production-path tests drive an approved specification through two Scout/Route-Setter passes, show five evidence-backed confidence dimensions, target the prior weakest gap, stop into a non-active candidate, require explicit acceptance, and then grant the same build/run authority.

The phase goal is nevertheless **not safely achieved**. Five independently confirmed Phase 200 defects let stored approval/candidate data be forged, permit writes through an intermediate `.aether` symlink, lose concurrent updates, and accept expired candidates. In plain English: the dashboard and route-planning journey exist, but some of the seals and locks behind them can be bypassed, so the resulting "exact accepted plan" cannot yet be trusted.

The four repository-suite failures are listed separately as inherited Phase 199 bookkeeping debt. No focused Phase 200 test failed, but the plan's repository-wide green-gate claim remains false.

## Goal Achievement

### Roadmap Success Criteria

The roadmap parser returned no structured `success_criteria` array, so the five written Phase 200 success criteria in `.planning/ROADMAP.md` were verified directly.

| # | Observable truth | Status | Evidence |
|---|---|---|---|
| 1 | A cited mechanism study reconstructs the Classic loop, audits the current Go/finalizer path, compares options, and selects a synthesis before implementation. | ✓ VERIFIED | `200-CLASSIC-SYNTHESIS.md` contains 11 Classic mechanisms, 11 current mechanisms, a 12-row comparative decision matrix, selected architecture, rejected alternatives, requirement/CAP linkage, and verification contract with immutable Git/path citations. |
| 2 | Production code visibly improves a real-repository plan across multiple grounded iterations and exposes the five dimensions, weakest gaps, deltas, and Queen recommendation. | ✓ VERIFIED | `coordinatePlanningScoutStage`, `coordinatePlanningRouteStage`, `projectPlanningIteration`, and `TestPlanningRealRepo200`; independent focused run: 4 checks passed. |
| 3 | Each additional research pass explains what changed, why confidence moved, and why the loop continued or stopped for a reproducible reason. | ✓ VERIFIED | `cmd/planning_confidence.go`, persisted iteration cards/timeline, and focused Route-stage tests; independent focused public/causality run: 16 checks passed. |
| 4 | Approved spec, survey/context/research/Hive/outcome evidence can prime the loop without stale answers suppressing current questions or widening scope. | ✓ VERIFIED | Typed evidence collection/validation in `cmd/planning_evidence.go`, scoped decision batches in `cmd/planning_decision.go`, staged manifests, and negative tests in the Phase 200 Scout/decision suites. |
| 5 | Intent yields an owner-readable exact specification; accepted plans carry complete proof/recovery/public-path/research bindings; insert/revision preserves history and explains change. | ✗ FAILED | Normal flow exists, but exactness/safety fail under unrecomputed spec hashes, forgeable semantic deltas, intermediate symlink escape, stale-writer overwrite, and unenforced expiry. See Gaps 1-5. |

**Roadmap score:** 4/5 criteria verified.

### Plan Frontmatter Must-Haves

Score accounting uses all 64 frontmatter truths across the 25 plans. Roadmap statements that restate these contracts are reported above and are not double-counted: **47 VERIFIED, 16 FAILED, 1 UNCERTAIN**.

| Plan | Truth | Status | Evidence / reason |
|---|---|---|---|
| 200-01 | Specification approval, planning stop, and candidate acceptance are separate authorities. | ✓ VERIFIED | Separate types, statuses, receipts, commands, and activation transitions exist. |
| 200-01 | Every accepted revision names its exact specification, candidate, base revision, and evidence timeline. | ✗ FAILED | Fields exist, but specification and candidate semantic bodies are not cryptographically re-bound (Gaps 1-2). |
| 200-01 | The canonical specification carries every owner-readable section as typed stable-ID content. | ✓ VERIFIED | `pkg/colony/specification.go` and projection code implement all nine typed sections. |
| 200-02 | Existing colonies remain buildable without fabricated modern approvals. | ✓ VERIFIED | Legacy authority is explicitly classified and retained. |
| 200-02 | Interrupted/upgraded planning resumes from a classified, validated representation. | ✓ VERIFIED | Migration and stage-state compatibility paths are substantive and tested. |
| 200-03 | Every confidence movement cites fresh, applicable, content-addressed evidence. | ✓ VERIFIED | Assessment validators bind evidence IDs/frontier/scope and reject unsupported increases. |
| 200-03 | Stale or scope-mismatched context cannot silently inflate confidence. | ✓ VERIFIED | Evidence admissibility/freshness/scope validators and refusal tests exist. |
| 200-04 | Each pass shows five evidence-backed integer scores and the weakest material gap. | ✓ VERIFIED | Canonical dimensions and derived ranking are implemented in `planning_confidence.go`. |
| 200-04 | The loop stops only for an explicit reproducible policy reason. | ✓ VERIFIED | Target, diminishing return, stall, pass cap, material decision, and continue reasons are typed/derived. |
| 200-05 | Each pass explains semantic changes to phases, tasks, dependencies, proof checks, and recovery paths. | ✓ VERIFIED | Go derives all semantic-delta categories and the iteration renderer exposes them. |
| 200-05 | Authority changes are visible and cannot masquerade as plan-content improvements. | ✗ FAILED | Mutable, unrecomputed authority impacts in `SemanticDelta` can be used as reconciliation coverage (Gap 2). |
| 200-06 | Every accepted plan retains an ordered append-only pass chain. | ✗ FAILED | Serial chain validation works, but stale-read writers can replace an index derived before transaction entry (Gap 4). |
| 200-06 | Completed-pass retry is idempotent and conflicting replay is rejected without mutation. | ✓ VERIFIED | Receipt/request-digest replay checks and snapshot tests pass. |
| 200-07 | The owner is interrupted only for material choices evidence cannot answer. | ✓ VERIFIED | Decision candidates are materiality-filtered at the staged boundary. |
| 200-07 | Prompts are batched, evidence-first, scoped, and reusable only for equivalent meaning. | ✓ VERIFIED | Batch/resume tokens bind goal/session/spec/pass/card and revalidation state. |
| 200-08 | Settled intent produces one canonical draft specification lineage. | ✗ FAILED | Serial lineage works, but concurrent drafts/revisions can overwrite a committed lineage (Gap 4). |
| 200-08 | Only exact explicit revision approval makes a specification planning-eligible. | ✗ FAILED | Approval checks stored hashes without recomputing the approved body (Gap 1). |
| 200-08 | Scoped revisions preserve unaffected items and invalidate only affected downstream scope. | ✗ FAILED | Happy path works, but forged delta coverage and stale-writer overwrite defeat honest reconciliation (Gaps 2 and 4). |
| 200-09 | Scout and Route-Setter run as separately authorized visible stages. | ✓ VERIFIED | Stage reducer and finalizer authorize Scout first, then one Route-Setter from the committed Scout receipt. |
| 200-09 | A crash resumes from the last committed receipt without rerun or skip. | ✓ VERIFIED | Stage receipts, replay reducers, and fault-injection tests cover resume. |
| 200-09 | No manifest pre-authorizes later stages or directly activates a plan. | ✓ VERIFIED | Per-stage manifests and non-active stop candidate are enforced. |
| 200-10 | The owner can inspect, revise, and exactly approve a canonical spec via a real public command. | ✗ FAILED | The public command is real, but approval is not exact because an edited body can retain its stored revision hash (Gap 1). |
| 200-10 | Normal help includes discuss and spec without inventing Codex slash commands. | ✓ VERIFIED | Command catalog/guide and platform-specific wording are covered by public-path tests. |
| 200-11 | Discuss asks only unresolved material, evidence-grounded questions. | ✓ VERIFIED | Scoped decision analysis and settled-answer filtering are implemented/tested. |
| 200-11 | Settled discussion creates/renders a draft spec and directs review. | ✓ VERIFIED | `runDiscuss` hands off to canonical spec draft and lifecycle next action. |
| 200-12 | Every lifecycle surface derives one honest next step from one snapshot. | ✓ VERIFIED | Shared lifecycle facts/projection/next-action plumbing is wired. |
| 200-12 | Visible journey is init → discuss → spec → plan → candidate acceptance → build/run. | ✓ VERIFIED | Go next actions, YAML, generated wrappers, docs, and parity tests align. |
| 200-13 | Unflagged planning requires one of four presets; valid flags bypass only that prompt. | ✓ VERIFIED | `resolvePlanningPreset` and host-plan tests cover selection/refusal. |
| 200-13 | Planning begins only from the exact approved spec and dispatches one autonomous Scout. | ✗ FAILED | One Scout is dispatched correctly, but the approval gate trusts an unrecomputed specification body (Gap 1). |
| 200-13 | Routine phase research no longer asks for approval. | ✓ VERIFIED | Automatic research policy is bound into run headers/manifests. |
| 200-14 | A completed Scout is visible and durably finalized before Route-Setter starts. | ✓ VERIFIED | Scout receipt/state commits precede Route authorization. |
| 200-14 | Known material choices are batched after the full first Scout pass. | ✓ VERIFIED | First-pass decision checkpoint tests pass. |
| 200-14 | Later Scout material findings reach Route before the owner pause. | ✓ VERIFIED | Two-pass real-repo subtest verifies complete card persistence before decision boundary. |
| 200-14 | Invalid/replay-conflicting Scout output changes no score/card/plan. | ✓ VERIFIED | Strict result schema, exact authority, and snapshot refusal tests exist. |
| 200-15 | Route-Setter converts the exact Scout receipt into a complete proposal and compact card. | ✓ VERIFIED | Route manifest/result binding and card derivation are substantive. |
| 200-15 | Route proposes five evidence-backed assessments; Go validates/recomputes overall/delta/gap/stop. | ✓ VERIFIED | Derived confidence/delta functions and focused tests pass. |
| 200-15 | Every stop creates a non-active candidate with causal evidence and typed Queen recommendation. | ✓ VERIFIED | Normal stop path persists `pending_review`; no stop activates the plan. |
| 200-16 | A stopped plan remains reviewable until explicit acceptance of that exact candidate. | ✗ FAILED | Exact candidate identity omits mutable semantic data and expiry is unenforced (Gaps 2 and 5). |
| 200-16 | Acceptance atomically binds candidate, approved spec, base, timeline, and proposal hash. | ✗ FAILED | Stored fields are co-written atomically, but spec/candidate bodies are not exact and stale derivation can overwrite intervening state (Gaps 1, 2, 4). |
| 200-16 | Stale/divergent acceptance cannot mutate the active plan. | ✗ FAILED | Request/base mismatch checks exist, but forged semantic coverage and read-before-lock races remain accepted paths. |
| 200-17 | Material spec revision invalidates only the affected requirement/task/proof closure. | ✓ VERIFIED | Impact graph derives affected/preserved IDs and serial tests cover scoped invalidation. |
| 200-17 | Unaffected/completed compatible work stays preserved across accepted revisions. | ✓ VERIFIED | Compatibility hashing/preservation and real-repo subtest cover normal revisions. |
| 200-17 | Build/seal cannot verify while affected scope lacks an approved reconciled plan. | ✗ FAILED | Gate exists, but forgeable semantic-delta coverage can falsely satisfy reconciliation (Gap 2). |
| 200-18 | Build and run consume the same accepted-plan authority policy. | ✓ VERIFIED | Both use shared plan-authority projection; real-repo test asserts parity. |
| 200-18 | Draft specs and pending candidates are never build-eligible. | ✓ VERIFIED | Authority refusals and recovery actions are enforced/tested. |
| 200-18 | Legacy active plans retain explicit compatibility. | ✓ VERIFIED | Legacy-unbound classification is retained and tested. |
| 200-19 | Users can see alternating castes, evidence-backed cards, and stop reasons. | ✓ VERIFIED | Runtime visual projections and generated wrapper contracts contain the sequence. |
| 200-19 | Spec/decision/candidate/acceptance/impact/recovery output is legible at every width. | ? UNCERTAIN | Width/golden tests pass, but actual terminal readability is a human UX judgment (Human Check 2). |
| 200-19 | JSON and terminal expose the same semantic model without hidden authority. | ✗ FAILED | Candidate expiry exists in the model/JSON but is absent from terminal candidate projection (Gap 5 warning facet). |
| 200-20 | Claude/OpenCode discuss wrappers preserve Go's evidence-first decision/spec boundaries. | ✓ VERIFIED | Canonical YAML and generated projections are synchronized and parity-tested. |
| 200-20 | Managed projections do not own state or fabricate approval. | ✓ VERIFIED | Wrappers invoke runtime commands; command-call audits enforce boundary. |
| 200-21 | Claude/OpenCode expose four presets, staged loop, and exact acceptance without state ownership. | ✗ FAILED | Presets/stages and thin wrappers are correct, but the exposed acceptance is not exact because candidate identity omits mutable semantic data (Gap 2). |
| 200-21 | Canonical and managed planning projections stay synchronized without legacy research approval/depth. | ✓ VERIFIED | Sync/parity tests and source inspection pass. |
| 200-22 | Runtime contracts and Codex guidance agree on spec, loop, and acceptance authorities. | ✓ VERIFIED | `cmd/contracts/*.md`, command guide, and validation tests align. |
| 200-22 | Public host boundaries document replay/refusal/state ownership. | ✓ VERIFIED | Contract documents are substantive and `TestPlanningContractDocuments200` passes. |
| 200-23 | Executable proof covers intent-to-spec-to-two-pass-plan-to-acceptance. | ✓ VERIFIED | `TestPlanningRealRepo200` independently passed all four top/subtests. |
| 200-23 | Every confidence/decision/stop/acceptance/revision/replay/platform invariant has positive/refusal proof. | ✗ FAILED | No proof covers the five confirmed tamper/path/concurrency/expiry failures. |
| 200-23 | Targeted, full, and race suites pass with an auditable receipt. | ✗ FAILED | Targeted Phase 200 tests pass; full/race retain four inherited Phase 199 failures. |
| 200-24 | New colony init directs to discuss, not plan. | ✓ VERIFIED | Shared next action and init tests enforce it. |
| 200-24 | Claude/OpenCode init wrappers show the same init-to-discuss-to-spec journey. | ✓ VERIFIED | Canonical/generated source and parity tests align. |
| 200-24 | Init never creates/approves/accepts spec or plan for the owner. | ✓ VERIFIED | Runtime/wrapper boundary tests show no authority mutation. |
| 200-25 | Real `/ant-spec` exists on Claude/OpenCode and delegates mutations to Go. | ✓ VERIFIED | Canonical `spec.yaml`, four generated projections, command catalog and call audits are wired. |
| 200-25 | Spec approval remains visibly separate from stop and candidate acceptance. | ✓ VERIFIED | Separate screens/actions and explicit next steps exist. |
| 200-25 | Public inventory records discuss/spec only when runtime/managed paths exist. | ✓ VERIFIED | Classic command parity dataset and tests verify inventory truth. |

**Plan-frontmatter score:** 47/64 truths verified.

## Required Artifacts

The SDK artifact check passed **48/48 declarations** for existence and declared-pattern substance. Manual Level 3/4 inspection found these semantic exceptions:

| Artifact group | Expected | Exists / substantive / wired | Final status | Details |
|---|---|---|---|---|
| `200-CLASSIC-SYNTHESIS.md` | Pre-implementation mechanism study and selected design | yes / yes / linked to plans and tests | ✓ VERIFIED | Exact historical/current evidence, alternatives, decisions, requirements and proof IDs are present. |
| `pkg/colony/specification.go`, `cmd/specification.go`, `cmd/spec_projection.go`, `cmd/spec_cmd.go` | Canonical typed specification lifecycle | yes / yes / command and state wired | ✗ INTEGRITY GAP | Real data flows, but stored hashes are not recomputed and write roots/races are unsafe. |
| `pkg/colony/planning.go`, `cmd/planning_evidence.go`, `cmd/planning_confidence.go`, `cmd/planning_delta.go` | Typed evidence, five scores, gaps, deltas, stop policy | yes / yes / finalizer and renderer wired | ⚠ PARTIAL | Normal derivation works; persisted delta integrity is forgeable. |
| `cmd/planning_stage.go`, `cmd/planning_stage_receipt.go`, `cmd/planning_timeline.go` | Separate authorized stages, receipts, append-only timeline | yes / yes / coordinator wired | ⚠ PARTIAL | Serial replay/crash behavior works; symlink escape and stale-index overwrite remain. |
| `cmd/planning_decision.go`, `cmd/discuss.go` | Evidence-first scoped material decisions | yes / yes / discuss/scout/route wired | ✓ VERIFIED | Decision batch and revalidation data flow is real. |
| `cmd/plan_revision.go`, `cmd/plan_impact.go`, `cmd/plan_authority.go`, `cmd/plan_candidate.go` | Exact candidate acceptance and living-plan reconciliation | yes / yes / state and build/run wired | ✗ INTEGRITY GAP | The acceptance link consumes mutable delta/spec authority, races, and ignores expiry. |
| `cmd/planning_visuals.go`, `cmd/codex_visuals.go` | Honest terminal/JSON experience | yes / yes / public command wired | ⚠ PARTIAL | Main journey is rendered; candidate expiry is omitted and final legibility needs human confirmation. |
| `.aether/commands/{init,discuss,spec,plan}.yaml` plus generated Claude/OpenCode projections | Thin synchronized platform orchestration | yes / yes / runtime commands wired | ✓ VERIFIED | Wrappers dispatch runtime-issued stages and do not own state. |
| Phase 200 tests/corpus/contracts/gate receipt | Executable proof and auditable gate | yes / yes / test suite wired | ⚠ PARTIAL | Focused Phase 200 tests pass; coverage misses confirmed adversarial cases; four inherited Phase 199 suite failures remain. |

## Key Link Verification

The SDK key-link check found **47/47 declared patterns**, but pattern presence is not behavioral integrity. Manual tracing produced:

| From | To | Via | Status | Details |
|---|---|---|---|---|
| Settled discussion | Canonical spec draft | `runDiscuss` → specification mutation/projection | ✓ WIRED | State and `.aether/SPEC.md` projection are derived from typed content. |
| Approved spec | First Scout stage | host-plan approval/preset/evidence checks | ⚠ PARTIAL | Serial link works; tampered approved body is trusted under its stored hash. |
| Scout result | Route-Setter authorization | committed Scout receipt and exact stage manifest | ✓ WIRED | Later stage is not pre-authorized. |
| Route result | Confidence/delta/card/timeline | Go validation and derivation | ✓ WIRED | Five scores, weakest gap and stop/continue causality flow to durable cards. |
| Timeline/card | Next Scout or candidate | weakest-gap dispatch / stop reducer | ✓ WIRED | Two-pass focused proof confirms data flows. |
| Candidate | Impact reconciliation | `validatePlanCandidateImpactCoverage` | ✗ NOT TRUSTWORTHY | Consumes an unrecomputed, candidate-unbound semantic delta. |
| Candidate acceptance | Active revision | lifecycle transaction | ✗ NOT TRUSTWORTHY | Multi-file commit is atomic, but inputs can be forged/expired and state can be stale before transaction entry. |
| Active revision | Build and run | shared `plan_authority` decision | ⚠ PARTIAL | Both consumers agree, but both can trust the same tampered authority. |
| Canonical YAML | Claude/OpenCode surfaces | managed generation/parity contract | ✓ WIRED | No wrapper-side state authority found. |

## Data-Flow Trace (Level 4)

| Artifact | Data variable | Source | Produces real data | Status |
|---|---|---|---|---|
| Specification projection | typed spec sections/status/delta | `COLONY_STATE.json` specification lineage | Yes | ⚠ FLOWING BUT UNAUTHENTICATED BODY |
| Iteration card | five assessments, evidence, weakest gap, semantic delta, decision | strict Scout/Route results validated by Go | Yes | ✓ FLOWING |
| Timeline | ordered card IDs/hashes/digest | Route-stage finalization receipts | Yes | ⚠ FLOWING WITH LOST-UPDATE RISK |
| Candidate review | proposal, scores, gaps, recommendation, timeline | stopped Route pass and candidate artifact | Yes | ⚠ FLOWING; EXPIRY HIDDEN |
| Accepted plan authority | spec/candidate/base/timeline/proposal bindings | candidate acceptance receipt and active revision | Yes | ✗ FLOWING FROM FORGEABLE/STALABLE INPUT |
| Build/run decision | shared eligibility projection | accepted plan authority | Yes | ⚠ SAME POLICY, COMPROMISED INPUT INTEGRITY |

## Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|---|---|---|---|
| Full approved-spec → two-pass loop → pending candidate → exact acceptance → build/run parity | `rtk go test ./cmd -run '^TestPlanningRealRepo200$' -count=1` | `Go test: 4 passed in 1 packages` | ✓ PASS |
| Public paths, contracts, confidence/delta, weakest-gap dispatch, non-active stop, material decision boundary | focused six-test `rtk go test ./cmd ... -count=1` | `Go test: 16 passed in 1 packages` | ✓ PASS |
| Independent uncached repository suite | `rtk go test ./... -count=1` | 9,554 passed, 4 failed, 11 skipped; all failures are Phase 199 bookkeeping | ✗ FAIL (inherited) |
| Race suite | executor `rtk go test ./... -race -count=1` | 8,526 passed, 4 failed, 8 skipped; no race diagnostics; same Phase 199 failures | ✗ FAIL (inherited) |

The race detector result does not clear Gap 4: it detects in-process memory races, not separate-process stale-read/lost-update behavior on JSON files.

## Probe Execution

No probe path was declared in any Phase 200 plan/summary, and no conventional `scripts/*/tests/probe-*.sh` file exists. **SKIPPED (no Phase 200 probes).**

## Requirements Coverage

Every requirement ID from all 25 plan frontmatters was found in `.planning/REQUIREMENTS.md`. The roadmap maps exactly these eight IDs to Phase 200; there are no orphaned Phase 200 requirements.

| Requirement | Source plans | Status | Codebase evidence |
|---|---|---|---|
| SYNTH-02 | 200-20..25 | ✓ SATISFIED | `200-CLASSIC-SYNTHESIS.md` is substantive, pre-plan dated, cited, comparative, and linked to implementation/proof. |
| CEC-03 | 16 plans | ✓ SATISFIED | Iteration cards, decision cards, stop decisions and Queen recommendation carry rationale and `evidence_that_would_change`. |
| PLAN-01 | 9 plans | ✓ SATISFIED | Separately authorized Scout/Route stages and visible cards run over Go finalizer/revision authority. |
| PLAN-02 | 7 plans | ✓ SATISFIED | Five canonical dimensions, evidence-bound assessments, weakest-gap ranking and targeted next Scout are implemented. |
| PLAN-03 | 10 plans | ✓ SATISFIED | Before/after scores, semantic deltas, evidence and typed stop reasons are persisted/rendered. |
| PLAN-04 | 12 plans | ✓ SATISFIED | Materiality filters, batched choices, scoped resume tokens, revalidation and successor-spec boundaries are real. |
| PLAN-05 | 20 plans | ✗ BLOCKED | Owner-readable spec/plan exist, but approved spec and accepted candidate cannot be proven exact; expiry is ineffective. |
| PLAN-06 | 17 plans | ✗ BLOCKED | Evidence/history/impact flows exist, but forged reconciliation, symlink escape and lost updates violate safe living-plan semantics. |

**Requirement score:** 6/8 satisfied.

## Anti-Patterns and Security Findings

The 143-file phase review set was rescanned. No unreferenced `TBD`, `FIXME`, or `XXX` debt marker was found. The one folded `todo` comment points to formal tracked work; placeholder hits were test normalization or unrelated legacy command descriptions, not Phase 200 user-visible stubs. No empty-render or log-only implementation was found.

| File | Lines | Pattern | Severity | Impact |
|---|---:|---|---|---|
| `cmd/planning_state.go` / `pkg/colony/specification.go` / `cmd/plan_authority.go` | 55-119 / 289-311,339-396 / 131-145 | Stored digest shape checked without body recomputation | BLOCKER | Edited approved intent can remain planning/build eligible. |
| `cmd/codex_plan_finalize.go` / `cmd/planning_state.go` / `cmd/plan_impact.go` | 4084-4164 / 514-555,739-770 / 244-287 | Candidate hash omits delta; nested hashes not recomputed | BLOCKER | Reconciliation can be forged without changing acceptance binding. |
| `cmd/planning_timeline.go` / `cmd/planning_stage_receipt.go` / `cmd/specification.go` / `cmd/lifecycle_transaction.go` | 225-279,449-465 / 1079-1088 / 1400-1424 / 310-331 | Intermediate symlink not physically contained | BLOCKER | Planning/spec writes can escape repository scope. |
| `cmd/specification.go` / `cmd/planning_timeline.go` / `cmd/plan_revision.go` / `cmd/codex_build.go` | 255-305,422-485 / 147-279 / 799-985 / 835-948 | Authoritative read occurs before lock/transaction baseline | BLOCKER | Concurrent writers can silently lose committed changes. |
| `pkg/colony/planning.go` / `cmd/plan_revision.go` | 858-895 / 789-905 | Expiry stored but never enforced | BLOCKER | Stale candidate remains accept-eligible indefinitely. |
| `cmd/planning_visuals.go` | 120-142,385-434 | Candidate projection omits expiry | WARNING | Owner cannot see when review authority becomes stale. |

## Human Verification Required

Automated verification is blocked first by the five code gaps. After those are fixed, the following UAT remains necessary:

### 1. Real primary-platform planning journey

**Test:** In a disposable real repository, use both Claude and OpenCode managed surfaces to run init → discuss → spec review/approval → a two-pass Balanced plan → candidate review/acceptance.

**Expected:** Only runtime-authorized workers run; Scout and Route-Setter visibly alternate; each card explains new evidence, five score movements, weakest gap, delta and stop/continue cause; the owner is interrupted only for a material decision; no build/run action appears before exact acceptance.

**Why human:** The repository tests exercise the Go/runtime and generated contracts with fixtures, not an actual host interaction with live external worker responses.

### 2. Terminal readability and expiry clarity

**Test:** Review spec, decision, iteration, candidate, acceptance, impact and recovery screens at narrow, normal and wide terminal widths after expiry is implemented.

**Expected:** No clipped/ambiguous authority; candidate expiry and recovery are obvious; residual gaps, evidence and next action remain understandable without reading JSON.

**Why human:** Golden/width tests can prove stable wrapping and fields, but not human readability or decision clarity.

## Deferred-Item Filter

No gap was deferred. Phases 201-204 address work-cycle/live/biological/learning behavior, and Phase 205 validates the completed journey; none explicitly schedules specification digest recomputation, candidate-delta integrity, planning path containment, stale-writer concurrency control, or candidate expiry enforcement.

## Inherited Phase 199 Debt (Distinct from Phase 200 Code Gaps)

The independent uncached and race suites each fail four nodes rooted in protected Phase 199 vocabulary/gate-receipt bookkeeping: `TestCurrentVocabulary199` plus one subtest, `TestPhase199GateReceiptSchema`, and `TestPhase199GateReceipt`. No Phase 200 test failed. This does not explain or reduce the five Phase 200 blockers above; it separately prevents the 200-23 green-repository truth.

## Gaps Summary

Five Phase 200 root causes block trusted goal achievement:

1. Approved specification bodies are not re-hashed during validation.
2. Candidate acceptance does not bind the semantic delta used for impact reconciliation.
3. Intermediate `.aether` symlinks can redirect lifecycle writes outside the repository.
4. State is derived before transaction/lock entry, enabling stale overwrites.
5. Candidate expiry is recorded but neither enforced nor rendered.

One separate inherited Phase 199 bookkeeping failure keeps repository-wide gates red. The next action is to run `$gsd-plan-phase 200 --gaps` (or the equivalent gap-closure planner), fix the five Phase 200 blockers, resolve/waive the protected Phase 199 gate debt separately, then re-run this verification and human UAT.

---

_Verified: 2026-09-08T06:16:14Z_
_Verifier: the agent (gsd-verifier)_
