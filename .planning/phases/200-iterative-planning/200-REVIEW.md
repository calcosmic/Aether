---
phase: 200-iterative-planning
reviewed: 2026-09-08T05:59:15Z
depth: standard
files_reviewed: 143
files_reviewed_list:
  - .aether/commands/classic-command-parity.json
  - .aether/commands/discuss.yaml
  - .aether/commands/init.yaml
  - .aether/commands/plan.yaml
  - .aether/commands/spec.yaml
  - .aether/docs/wrapper-host-contract.md
  - .aether/schemas/completion-packet.schema.json
  - .aether/skills/colony/aether-colony-build-cycle/SKILL.md
  - .aether/skills/colony/aether-colony-research/SKILL.md
  - .claude/commands/ant-discuss.md
  - .claude/commands/ant-init.md
  - .claude/commands/ant-plan.md
  - .claude/commands/ant-spec.md
  - .claude/commands/ant/discuss.md
  - .claude/commands/ant/init.md
  - .claude/commands/ant/plan.md
  - .claude/commands/ant/spec.md
  - .opencode/commands/ant/discuss.md
  - .opencode/commands/ant/init.md
  - .opencode/commands/ant/plan.md
  - .opencode/commands/ant/spec.md
  - cmd/advance_phase.go
  - cmd/autopilot_policy.go
  - cmd/autopilot_policy_test.go
  - cmd/blackbox_harness_test.go
  - cmd/ceremony_emitter_test.go
  - cmd/classic_command_parity_test.go
  - cmd/classic_contract_test.go
  - cmd/codex_build.go
  - cmd/codex_continue.go
  - cmd/codex_plan.go
  - cmd/codex_plan_finalize.go
  - cmd/codex_plan_finalize_test.go
  - cmd/codex_plan_test.go
  - cmd/codex_visuals.go
  - cmd/codex_visuals_test.go
  - cmd/codex_workflow_cmds.go
  - cmd/codex_workflow_cmds_test.go
  - cmd/colony_mode_manifest_test.go
  - cmd/command_call_audit_test.go
  - cmd/command_guide.go
  - cmd/command_guide_test.go
  - cmd/contract_validate_test.go
  - cmd/contracts/discuss.md
  - cmd/contracts/plan.md
  - cmd/contracts/spec.md
  - cmd/discuss.go
  - cmd/discuss_spec_200_test.go
  - cmd/discuss_test.go
  - cmd/discuss_wrapper_contract_200_test.go
  - cmd/e2e_lifecycle_test.go
  - cmd/e2e_regression_test.go
  - cmd/front_door_199_test.go
  - cmd/front_door_200_test.go
  - cmd/front_door_init_wrapper_199_test.go
  - cmd/golden_workflow_test.go
  - cmd/init_cmd.go
  - cmd/init_proposals_test.go
  - cmd/isolated_process_test.go
  - cmd/lifecycle_card_startup_test.go
  - cmd/lifecycle_facts.go
  - cmd/lifecycle_facts_test.go
  - cmd/lifecycle_projection.go
  - cmd/lifecycle_projection_test.go
  - cmd/lifecycle_wrapper_contract_test.go
  - cmd/next_action.go
  - cmd/next_action_test.go
  - cmd/orchestrator_boundary_guidance_test.go
  - cmd/orchestrator_boundary_questions_test.go
  - cmd/phase_research_decision.go
  - cmd/phase_research_decision_cmd.go
  - cmd/phase_research_decision_test.go
  - cmd/phase_research_dispatch_test.go
  - cmd/plan_acceptance_gate_200_test.go
  - cmd/plan_authority.go
  - cmd/plan_authority_test.go
  - cmd/plan_candidate.go
  - cmd/plan_candidate_test.go
  - cmd/plan_depth_proposal_manifest_test.go
  - cmd/plan_impact.go
  - cmd/plan_impact_test.go
  - cmd/plan_revision.go
  - cmd/plan_revision_test.go
  - cmd/plan_wrapper_cards_test.go
  - cmd/plan_wrapper_ceremony_test.go
  - cmd/planning_confidence.go
  - cmd/planning_confidence_test.go
  - cmd/planning_contract_docs_200_test.go
  - cmd/planning_decision.go
  - cmd/planning_decision_test.go
  - cmd/planning_delta.go
  - cmd/planning_delta_test.go
  - cmd/planning_evidence.go
  - cmd/planning_evidence_test.go
  - cmd/planning_migration.go
  - cmd/planning_migration_test.go
  - cmd/planning_public_paths_200_test.go
  - cmd/planning_real_repo_200_test.go
  - cmd/planning_route_stage_200_test.go
  - cmd/planning_scout_stage_200_test.go
  - cmd/planning_stage.go
  - cmd/planning_stage_receipt.go
  - cmd/planning_stage_receipt_test.go
  - cmd/planning_stage_test.go
  - cmd/planning_state.go
  - cmd/planning_state_test.go
  - cmd/planning_timeline.go
  - cmd/planning_timeline_test.go
  - cmd/planning_visuals.go
  - cmd/planning_visuals_test.go
  - cmd/platform_doc_hygiene_test.go
  - cmd/root.go
  - cmd/root_test.go
  - cmd/safety_invariant_test.go
  - cmd/seal_outcome.go
  - cmd/spec_cmd.go
  - cmd/spec_cmd_test.go
  - cmd/spec_plan_seal_gate_200_test.go
  - cmd/spec_projection.go
  - cmd/spec_projection_test.go
  - cmd/specification.go
  - cmd/specification_test.go
  - cmd/state_extra.go
  - cmd/state_extra_test.go
  - cmd/state_load.go
  - cmd/state_load_test.go
  - cmd/territory_wrapper_contract_199_test.go
  - cmd/testdata/classic-contract/v1/cases.json
  - cmd/testdata/classic-contract/v1/mechanisms.json
  - cmd/testdata/classic-contract/v1/schema.json
  - cmd/testdata/command_catalog.json
  - cmd/testdata/golden_plan.txt
  - cmd/testdata/parity_snapshot.json
  - cmd/testdata/regression_snapshot.json
  - cmd/testing_main_test.go
  - cmd/visual_wrapper_contract_test.go
  - cmd/wire_capsule_198_2_test.go
  - cmd/wrapper_command_names.go
  - pkg/colony/colony.go
  - pkg/colony/planning.go
  - pkg/colony/planning_test.go
  - pkg/colony/specification.go
  - pkg/colony/specification_test.go
findings:
  critical: 5
  warning: 1
  info: 0
  total: 6
status: issues_found
---

# Phase 200: Code Review Report

**Reviewed:** 2026-09-08T05:59:15Z
**Depth:** standard
**Files Reviewed:** 143
**Status:** issues_found

## Summary

The Phase 200 implementation has five ship-blocking correctness/security defects and one terminal parity warning. In plain English: several records described as immutable can still be changed without invalidating their approval, two simultaneous commands can silently erase one another's updates, a repository-local symlink can redirect planning reads and writes outside the repository, and candidate deadlines are never enforced or shown in terminal review.

`aether source-check` passed. `go test ./...` completed with 9,549 passes and 11 skips; its four failures are the already-protected Phase 199 vocabulary-inventory drift and expired Phase 199 gate-receipt timestamps recorded by the phase, not Phase 200 defects. Those known baseline failures are not counted below.

## Narrative Findings (AI reviewer)

## Critical Issues

### CR-01: Approved specification hashes are trusted instead of recomputed

**Classification:** BLOCKER

**File:** `/Users/callumcowie/repos/Aether/cmd/planning_state.go:55-119`

**Related files:**

- `/Users/callumcowie/repos/Aether/pkg/colony/specification.go:275-396`
- `/Users/callumcowie/repos/Aether/cmd/plan_authority.go:131-145`
- `/Users/callumcowie/repos/Aether/cmd/specification.go:623-655,703-763,1179-1216`

**Issue:** Canonical creation hashes every typed specification item and the complete revision payload, but the load/authority validators only check that stored digests look like SHA-256 values and that IDs end with the claimed digest. `SpecRevision.Validate` checks structural relationships, and `SpecApprovalReceipt.Validate` checks only non-empty fields; neither recomputes the revision, item, approval ID, or deterministic approval-token hash. `validateCurrentPlanAuthority` then treats those mutually copied stored values as proof. An edited outcome, requirement, acceptance check, public path, evidence binding, or delta can therefore retain the old item/revision hashes, revision ID, and approval receipt and still be accepted as the approved contract for build/run. This defeats the core owner-authority boundary.

**Fix:** Add one canonical validation path that recomputes every item hash using its section-specific payload, recomputes `specificationRevisionContentHash`, checks the revision ID, derives the expected approval token and token hash, and recomputes the approval receipt ID using the same payload as `buildSpecificationApprovalReceipt`. Invoke it while loading state and again inside `validateCurrentPlanAuthority`; fail closed on any mismatch. Add tamper tests that change each body field while retaining the stored hashes and assert both build and run refuse it.

### CR-02: A forged semantic delta can bypass required plan reconciliation

**Classification:** BLOCKER

**File:** `/Users/callumcowie/repos/Aether/cmd/codex_plan_finalize.go:4084-4165`

**Related files:**

- `/Users/callumcowie/repos/Aether/cmd/planning_state.go:739-770`
- `/Users/callumcowie/repos/Aether/pkg/colony/planning.go:555-665`
- `/Users/callumcowie/repos/Aether/cmd/plan_candidate.go:207-216,285-325`
- `/Users/callumcowie/repos/Aether/cmd/plan_impact.go:241-287`
- `/Users/callumcowie/repos/Aether/cmd/plan_revision.go:819-864`

**Issue:** The candidate content hash is built from the run/spec/base/timeline/proposal/stop tuple but excludes `SemanticDelta`; candidate loading validates the delta's claimed hashes only for syntax and ID suffixes. Acceptance then calls `validatePlanCandidateImpactCoverage`, which counts any non-`preserved` delta entry or authority-impact ID as proof that affected scope was reconciled. Consequently, a pending candidate artifact can be edited to add fabricated changed/removed entries for every affected semantic ID while leaving the proposal, candidate hash, and already displayed acceptance token unchanged. The acceptance gate will activate an unchanged/stale plan against a changed specification because the unverified delta claims satisfy its coverage set.

**Fix:** Recompute the canonical hash for every semantic change, authority impact, and the enclosing delta on load. Bind that verified delta (and the rest of the exact review payload) into the candidate content hash/token. At acceptance, independently derive the semantic delta and affected coverage from the immutable base proposal, current specification, and proposed plan; do not let self-asserted candidate delta entries alone prove reconciliation. Add a regression test that mutates only `candidate.SemanticDelta` after review and confirms the original acceptance command is refused with zero state mutation.

### CR-03: An intermediate `.aether` symlink escapes the repository boundary

**Classification:** BLOCKER

**File:** `/Users/callumcowie/repos/Aether/cmd/planning_timeline.go:225-238,449-465`

**Related files:**

- `/Users/callumcowie/repos/Aether/cmd/planning_stage_receipt.go:1079-1088`
- `/Users/callumcowie/repos/Aether/cmd/specification.go:1400-1424`

**Issue:** The new roots reject a symlink only at the repository root and, for specification operations, at the final `.aether/data` component. They never reject an intermediate `.aether` symlink or prove that the resolved data root remains physically beneath the resolved repository root. With `repo/.aether -> /chosen/external/directory` and a real external `data/` directory, `Lstat(repo/.aether/data)` reports that final directory as real, `MkdirAll` follows the symlink, and lifecycle transactions accept the logical data-root path. Planning/specification reads and writes can therefore consume or overwrite files outside the selected repository.

**Fix:** Before any read, directory creation, or transaction setup, validate every repository-relative component from the canonical root through `.aether/data`; reject `.aether` or `data` when either is a symlink and require the physically resolved data root to be contained by the physically resolved repository root. Use the containment logic already present in `validateLegacyPlanningArtifactPath` as a baseline, and use no-follow directory handles/open-at operations (or equivalent) for mutations so a component cannot be swapped after validation. Add read and write tests with an intermediate `.aether` symlink.

### CR-04: Transactions capture their baseline after stale derivation, allowing lost updates

**Classification:** BLOCKER

**File:** `/Users/callumcowie/repos/Aether/cmd/specification.go:416-605,1453-1495`

**Related files:**

- `/Users/callumcowie/repos/Aether/cmd/planning_timeline.go:130-283`
- `/Users/callumcowie/repos/Aether/cmd/plan_revision.go:789-994`
- `/Users/callumcowie/repos/Aether/cmd/codex_build.go:835-854,923-950`

**Issue:** These paths load state/index bytes, derive a complete replacement, and only then begin a lifecycle transaction (or call ordinary `SaveJSON`). The transaction's `before` digest is captured when targets are declared, not when the application data was read. If command B reads state S, command A commits S+A, and then B begins its transaction, B records S+A as the transaction baseline but writes its stale S+B replacement. Validation succeeds and A's specification revision, accepted candidate, phase mutation, or timeline-index entry is silently erased. For timeline appends, A's card can be left orphaned while B replaces the index. A commit-only mutex cannot repair this read/derive/write race, and separate CLI processes need the same protection.

**Fix:** Put every authoritative load, validation, derivation, and multi-file commit under one shared cross-process state/index lock, or extend lifecycle transaction declarations with an expected digest captured from the bytes initially read and return a retryable conflict when it differs. On conflict, reload and re-derive rather than writing the stale snapshot. Apply the same protocol to specification revision/approval, candidate acceptance, timeline append, and direct build state transition; add deterministic two-writer tests that pause B after its read, commit A, then resume B and require a conflict or merged result.

### CR-05: Candidate expiry is recorded but never enforced

**Classification:** BLOCKER

**File:** `/Users/callumcowie/repos/Aether/pkg/colony/planning.go:858-925`

**Related files:**

- `/Users/callumcowie/repos/Aether/cmd/codex_plan_finalize.go:4155-4165`
- `/Users/callumcowie/repos/Aether/cmd/plan_revision.go:789-905`

**Issue:** Candidates are issued with a seven-day `ExpiresAt` and the model defines an `expired` status, but validation only requires a non-zero timestamp. The sole pending-to-accepted transition never compares trusted current time or `AcceptedAt` with `ExpiresAt`, and no production path ever moves a candidate to `expired`. A captured acceptance command remains valid indefinitely, including after its advertised review window, while an old pending artifact can remain the only reviewable candidate forever. `ExpiresAt <= CreatedAt` is also accepted structurally.

**Fix:** Require `ExpiresAt.After(CreatedAt)`. Before accepting a `pending_review` candidate, obtain an injectable trusted clock and refuse/atomically mark it `expired` when `now >= ExpiresAt`; preserve replay of an already accepted receipt even after the deadline. Add an explicit cleanup/review path for expired artifacts and boundary tests for just before, exactly at, and after expiry.

## Warnings

### WR-01: Terminal candidate review hides the expiry present in JSON

**Classification:** WARNING

**File:** `/Users/callumcowie/repos/Aether/cmd/planning_visuals.go:214-255,385-434`

**Related files:**

- `/Users/callumcowie/repos/Aether/cmd/plan_candidate.go:238-252`
- `/Users/callumcowie/repos/Aether/cmd/contracts/plan.md:226-232`

**Issue:** JSON review includes the complete candidate and therefore `expires_at`, and the public contract explicitly promises candidate ID/hash and expiry. The terminal projection has no expiry/staleness field and the renderer never prints one. An interactive owner is asked to accept without seeing the deadline that programmatic consumers receive, breaking terminal/JSON semantic parity and making the unenforced-expiry defect harder to notice.

**Fix:** Add `ExpiresAt` and a derived `Expired`/staleness state to `planningCandidateProjection`, render an explicit `Expires:` or `Expired:` line beside candidate status, and assert the same semantics in color, no-color, and JSON projection tests.

---

_Reviewed: 2026-09-08T05:59:15Z_
_Reviewer: gsd-code-reviewer_
_Depth: standard_
