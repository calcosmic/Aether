# Phase 201 Classic Coverage Audit

This is the human rendering of `201-CLASSIC-COVERAGE.json`. The JSON ledger is
the machine-validated source; each line below names one capability the old
Classic system had, where that capability lives in the program today, and the
name of the check that would fail if it ever stopped being true.

## GOAL

| ID | Disposition | Modern home | Plan/task | Proof |
|---|---|---|---|---|
| CAP-003 | restore-modern | cmd/memory_feed.go | 201-10 | TestFailureEvidenceCarriesTheAttemptIdentity |
| CAP-004 | restore-modern | pkg/colony/flags.go | 201-10 | TestBlockerTruthIsOneStore |
| CAP-022 | restore-modern | cmd/deterministic_floor.go | 201-02 | TestDeterministicFloorIsTheOnlySourceOfAPass |
| CAP-024 | restore-modern | cmd/work_repair.go | 201-10 | TestRepairEvaluationNamesItsDrivingInput |
| CAP-029 | replace-better | cmd/command_truth.go | 201-11 | TestQuickRunsOnTheSharedAttemptModel |
| CAP-051 | restore-modern | cmd/status.go | 201-10 | TestEscalatedCountHasOneCountingPath |
| CAP-066 | replace-better | cmd/build_knowledge_deltas.go | 201-18 | TestDeriveBuildKnowledgeDeltas |
| CAP-071 | replace-better | cmd/attempt_artifacts.go | 201-08 | TestLegacyArtifactReadIsValidatedIdentically |
