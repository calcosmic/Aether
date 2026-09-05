# Phase 199 Classic Coverage Audit

This is the human rendering of `199-CLASSIC-COVERAGE.json`. The JSON ledger is
the machine-validated source; each entry below carries its selected disposition,
concrete current home, owning implementation task, and executable proof.

## GOAL

| ID | Disposition | Modern home | Plan/task | Proof |
|---|---|---|---|---|
| CAP-006 | replace-better | cmd/entomb_cmd.go | 199-21 Task 1 | TestLifecycleCloseout199Entomb |
| CAP-007 | replace-better | cmd/entomb_cmd.go | 199-21 Task 1 | TestEntombManifest199Build |
| CAP-008 | restore-modern | cmd/maintenance.go | 199-18 Task 2 | TestMaintenanceState199CleanupOwnership |
| CAP-013 | restore-modern | cmd/pheromone_write.go | 199-08 Task 1 | TestPheromoneLifecycle_FocusSignal |
| CAP-015 | restore-modern | cmd/history.go | 199-10 Task 2 | TestLifecycleHistory199FocusedProjection |
| CAP-016 | restore-modern | .aether/commands/help.yaml | 199-30 Task 1 | TestCommandGuideLifecycle199 |
| CAP-017 | replace-better | cmd/init_cmd.go | 199-03 Task 1 | TestLifecycleCloseout199Init |
| CAP-018 | replace-better | cmd/status.go | 199-09 Task 1 | TestLifecycleStatus199FullOrder |
| CAP-019 | restore-modern | cmd/command_truth.go | 199-18 Task 1 | TestMigrateStateRollbackRestoresExactBackupAndKeepsSafetyCopy |
| CAP-020 | replace-better | cmd/next_action.go | 199-09 Task 2 | TestLifecycleNextAction |
| CAP-026 | restore-modern | cmd/seal_final_review.go | 199-20 Task 2 | TestLifecycleCloseout199Seal |
| CAP-027 | restore-modern | cmd/phase.go | 199-10 Task 1 | TestLifecyclePhase199List |
| CAP-028 | restore-modern | cmd/phase.go | 199-10 Task 1 | TestLifecyclePhase199FocusedProjection |
| CAP-032 | restore-modern | cmd/session_flow_cmds.go | 199-13 Task 1 | TestLifecycleCloseout199Resume |
| CAP-033 | restore-modern | cmd/session_flow_cmds.go | 199-13 Task 2 | TestResumeWrapperContract199 |
| CAP-034 | restore-modern | cmd/autopilot_policy.go | 199-06 Task 2 | TestAutopilotContract199CompletionDoesNotSeal |
| CAP-035 | replace-better | cmd/normalize_args.go | 199-34 Task 1 | TestRuntimeRecoveryCompatibility199 |
| CAP-036 | restore-modern | cmd/codex_workflow_cmds.go | 199-20 Task 1 | TestLifecycleCloseout199Seal |
| CAP-037 | restore-modern | cmd/consolidation_lifecycle.go | 199-20 Task 1 | TestLifecycleCloseout199Seal |
| CAP-038 | restore-modern | cmd/seal_final_review.go | 199-20 Task 2 | TestLifecycleCloseout199ForcedSeal |
| CAP-039 | restore-modern | cmd/codex_workflow_cmds.go | 199-20 Task 1 | TestLifecycleCloseout199Refusal |
| CAP-040 | restore-modern | cmd/codex_visuals.go | 199-20 Task 2 | TestLifecycleCloseout199Seal |
| CAP-041 | restore-modern | cmd/seal_confirmation.go | 199-20 Task 1 | TestLifecycleCloseout199ForcedSeal |
| CAP-042 | restore-modern | cmd/hive_policy.go | 199-20 Task 2 | TestSealPromotionSurvivesWithRepoNameInText |
| CAP-049 | restore-modern | cmd/survey_staleness.go | 199-04 Task 1 | TestLifecycleCloseout199Colonize |
| CAP-050 | restore-modern | cmd/status.go | 199-09 Task 1 | TestLifecycleStatus199NoInventedEvidence |
| CAP-052 | replace-better | cmd/entomb_cmd.go | 199-21 Task 2 | TestEntombManifest199CrossReferences |
| CAP-053 | restore-modern | cmd/update_cmd.go | 199-18 Task 2 | TestMaintenanceMutation199Update |
| CAP-059 | replace-better | cmd/survey_staleness.go | 199-04 Task 1 | TestLifecycleCloseout199Colonize |
| CAP-060 | replace-better | cmd/session_flow_cmds.go | 199-13 Task 2 | TestLifecycleCloseout199Pause |
| CAP-062 | replace-better | cmd/skills.go | 199-18 Task 2 | TestMaintenanceSkills199 |
| CAP-064 | replace-better | cmd/entomb_cmd.go | 199-21 Task 2 | TestEntombManifest199Deterministic |
| CAP-065 | replace-better | cmd/maintenance.go | 199-18 Task 2 | TestMaintenanceInspectionStructuredResult |
| CAP-068 | restore-modern | cmd/init_cmd.go | 199-03 Task 1 | TestInitWithCharterJSONFlag |

## REQ

| ID | Disposition | Modern home | Plan/task | Proof |
|---|---|---|---|---|
| SYNTH-01 | replace-better | 199-CLASSIC-SYNTHESIS.md | 199-28 Task 1 | TestClassicMechanismCoverage |
| CEC-01 | restore-modern | cmd/lifecycle_projection.go | 199-09 Task 1 | TestLifecycleProjectionStateTable |
| CEC-02 | restore-modern | cmd/next_action.go | 199-09 Task 2 | TestLifecycleNextAction |
| CEC-04 | restore-modern | cmd/session_flow_cmds.go | 199-13 Task 1 | TestLifecycleCloseout199Pause |
| CEC-08 | restore-modern | cmd/seal_final_review.go | 199-20 Task 2 | TestLifecycleCloseout199Seal |
| LIFE-01 | restore-modern | cmd/init_cmd.go | 199-03 Task 1 | TestLifecycleCloseout199FrontDoorRefusal |
| LIFE-02 | replace-better | cmd/survey_staleness.go | 199-04 Task 1 | TestLifecycleCloseout199Colonize |
| LIFE-03 | replace-better | cmd/lifecycle_projection.go | 199-09 Task 1 | TestLifecycleProjectionIsDeterministicAcrossViews |
| LIFE-04 | replace-better | cmd/recovery_engine.go | 199-34 Task 1 | TestRuntimeRecoveryCompatibility199 |
| LIFE-05 | restore-modern | cmd/entomb_cmd.go | 199-21 Task 1 | TestEntombManifest199ForcedMarker |
| LIFE-06 | replace-better | cmd/maintenance.go | 199-18 Task 2 | TestMaintenanceWrapperContract |
| PROOF-01 | replace-better | cmd/testdata/classic-contract/v1/cases.json | 199-02 Task 1 | TestClassicContractCorpusRequiredCategories |

## RESEARCH

| ID | Disposition | Modern home | Plan/task | Proof |
|---|---|---|---|---|
| SYN-199-01 | restore-modern | .aether/commands/help.yaml | 199-30 Task 1 | TestCommandGuideLifecycle199 |
| SYN-199-02 | replace-better | cmd/survey_staleness.go | 199-04 Task 1 | TestLifecycleCloseout199Colonize |
| SYN-199-03 | replace-better | cmd/lifecycle_projection.go | 199-09 Task 1 | TestLifecycleProjectionIsDeterministicAcrossViews |
| SYN-199-04 | restore-modern | cmd/pheromone_write.go | 199-08 Task 1 | TestPheromoneLifecycle_FocusSignal |
| SYN-199-05 | restore-modern | cmd/autopilot_policy.go | 199-06 Task 2 | TestAutopilotContract199CompletionDoesNotSeal |
| SYN-199-06 | replace-better | cmd/recovery_engine.go | 199-34 Task 1 | TestRuntimeRecoveryCompatibility199 |
| SYN-199-07 | restore-modern | cmd/seal_confirmation.go | 199-20 Task 1 | TestLifecycleCloseout199ForcedSeal |
| SYN-199-08 | restore-modern | cmd/entomb_cmd.go | 199-21 Task 1 | TestEntombManifest199Build |
| SYN-199-09 | replace-better | cmd/maintenance.go | 199-18 Task 2 | TestMaintenanceWrapperContract |
| SYN-199-10 | replace-better | cmd/testdata/classic-contract/v1/cases.json | 199-02 Task 1 | TestClassicContractCorpusCausalReceipts |

## CONTEXT

| ID | Disposition | Modern home | Plan/task | Proof |
|---|---|---|---|---|
| D-01 | restore-modern | .aether/commands/help.yaml | 199-30 Task 1 | TestCommandGuideLifecycle199 |
| D-02 | restore-modern | cmd/lifecycle_projection.go | 199-06 Task 1 | TestLifecycleProjectionAcceptedPlanChoicesAreCoequal |
| D-03 | replace-better | cmd/autopilot_policy.go | 199-06 Task 1 | TestAutopilotContract199CompletionDoesNotSeal |
| D-04 | restore-modern | cmd/autopilot_policy.go | 199-06 Task 2 | TestAutopilotContract199CompletionDoesNotSeal |
| D-05 | restore-modern | cmd/autopilot_report.go | 199-06 Task 2 | TestAutopilotContract199CompletionDoesNotSeal |
| D-06 | replace-better | cmd/autopilot_policy.go | 199-06 Task 2 | TestAutopilotContract199CompletionDoesNotSeal |
| D-07 | restore-modern | cmd/pheromone_write.go | 199-08 Task 1 | TestPheromoneLifecycle_FocusSignal |
| D-08 | retire-with-proof | cmd/pheromone_write.go | 199-08 Task 2 | TestPheromoneLifecycle_FocusSignal |
| D-09 | replace-better | cmd/lifecycle_projection.go | 199-09 Task 1 | TestLifecycleProjectionIsDeterministicAcrossViews |
| D-10 | restore-modern | cmd/lifecycle_projection.go | 199-10 Task 2 | TestLifecycleHistory199FocusedProjection |
| D-11 | restore-modern | cmd/codex_visuals.go | 199-20 Task 1 | TestLifecycleCloseout199Seal |
| D-12 | retire-with-proof | cmd/status.go | 199-09 Task 1 | TestLifecycleStatus199NoInventedEvidence |
| D-13 | replace-better | cmd/normalize_args.go | 199-34 Task 1 | TestRuntimeRecoveryCompatibility199 |
| D-14 | replace-better | cmd/recovery_engine.go | 199-14 Task 1 | TestLifecycleTransactionReplayExactlyOnce |
| D-15 | replace-better | cmd/recovery_engine.go | 199-34 Task 2 | TestRuntimeRecoveryCompatibility199 |
| D-16 | restore-modern | cmd/seal_confirmation.go | 199-20 Task 1 | TestLifecycleCloseout199ForcedSeal |
| D-17 | restore-modern | cmd/entomb_cmd.go | 199-33 Task 2 | TestLifecycleCloseout199Entomb |

## Scoped exclusions

Phase 200–205 implementation and native Codex `$ant-*` work are not Phase 199
implementation rows. They remain explicitly out of this exact signed coverage set.
