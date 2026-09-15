# Phase 200 Classic Coverage Audit

This is the human rendering of `200-CLASSIC-COVERAGE.json`. The JSON ledger is
the machine-validated source; each line below names one capability the old
Classic system had, where that capability lives in the program today, and the
name of the check that would fail if it ever stopped being true.

## GOAL

| ID | Disposition | Modern home | Plan/task | Proof |
|---|---|---|---|---|
| CAP-005 | replace-better | cmd/planning_decision.go | 200-07 | TestPlanningDecisionClassifyMaterialAndSuppressesNonOwnerChoices |
| CAP-010 | restore-modern | cmd/state_extra.go | 200-17 | TestInsertPhaseCandidatePreservesActiveRevisionAndAcceptsExactly |
| CAP-011 | restore-modern | cmd/state_extra.go | 200-17 | TestInsertPhaseImmutableStableIDsAndCompletedStatus |
| CAP-012 | restore-modern | cmd/state_extra.go | 200-17 | TestInsertPhaseRefusesMissingCoverageWithoutWrites |
| CAP-056 | restore-modern | cmd/planning_evidence.go | 200-03 | TestPlanningEvidenceFreshRejectsInactiveHiveStates |
| CAP-061 | replace-better | cmd/planning_evidence.go | 200-03 | TestPlanningEvidenceCollectAllKindsInStableOrder |
| CAP-069 | restore-modern | cmd/planning_evidence.go | 200-03 | TestPlanningEvidenceScopeRequiresExactPlanningFrontier |
