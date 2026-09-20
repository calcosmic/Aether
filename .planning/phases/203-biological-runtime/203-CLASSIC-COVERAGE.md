# Phase 203 Classic Coverage Audit

This is the human rendering of `203-CLASSIC-COVERAGE.json`. The JSON ledger is
the machine-validated source; each line below names one capability the old
Classic system had, where that capability lives in the program today, and the
name of the check that would fail if it ever stopped being true.

## GOAL

| ID | Disposition | Modern home | Plan/task | Proof |
|---|---|---|---|---|
| CAP-002 | restore-modern | cmd/pheromone_resolver.go | 203-05 Task 3 | TestOneEffectivePheromonePredicate |
| CAP-009 | restore-modern | cmd/agency_contract.go | 203-12 Task 2 | TestAgencyEvidenceFromTrophallaxisDecision |
| CAP-014 | restore-modern | cmd/pheromone_resolver.go | 203-05 Task 3 | TestEveryBriefReaderUsesTheResolver |
| CAP-030 | restore-modern | cmd/codex_plan.go | 203-05 Task 1 | TestActiveStrongExpiredExcludedByEveryReader |
| CAP-031 | restore-modern | cmd/pheromone_influence.go | 203-11 Task 2 | TestRevokedNoteStaysOut |
| CAP-058 | replace-better | cmd/suggest_approve.go | 203-08 Task 2 | TestSuggestApprove_DismissSuggestion |
