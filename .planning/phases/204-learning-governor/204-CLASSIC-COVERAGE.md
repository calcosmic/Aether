# Phase 204 Classic Coverage Audit

This is the human rendering of `204-CLASSIC-COVERAGE.json`. The JSON ledger is
the machine-validated source; each line below names one capability the old
Classic system had, where that capability lives in the program today, and the
name of the check that would fail if it ever stopped being true.

Two of these rows (`CAP-001` and `CAP-057`) were flagged by
`204-CLASSIC-SYNTHESIS.md`'s ruling (c) as frozen against a state the code has
since left. In both cases the study confirmed the frozen ledger's disposition
is correct in direction, so both rows are signed with the ledger's own
disposition value and the ruling's refined evidence recorded in
`historical_evidence` -- neither required a re-adjudication marker.

## GOAL

| ID | Disposition | Modern home | Plan/task | Proof |
|---|---|---|---|---|
| CAP-001 | restore-modern | cmd/application_evidence.go | 204-02 Task 1 | TestPhaseApplicationCreditTracerEndToEnd |
| CAP-025 | restore-modern | cmd/eval_gates.go | 204-07 Task 1 | TestEvalGateVocabularyMatchesTheManifest |
| CAP-043 | restore-modern | pkg/learn/difficulty.go | 204-09 Task 3 | TestApprovingASkillProposalCreatesTheSkill |
| CAP-055 | restore-modern | cmd/instinct_application.go | 204-03 Task 2 | TestApplicationOutcomeComesFromCreditNotFromAdvancement |
| CAP-057 | replace-better | cmd/phase_end_signals.go | 198.1-04 | TestExpiringAValuableNoteKeepsItInLongTermMemory, TestExpiringAThrowawayNoteKeepsNothing |
| CAP-067 | replace-better | cmd/episode_ledger.go | 204-04 Task 3 | TestDerivedViewsAreIdempotent, TestDerivedViewOverNoEpisodesIsEmptyNotAnError |
| CAP-070 | replace-better | cmd/episode_ledger.go | 204-04 Task 3 | TestEpisodeWithNoOutcomeRendersAsNoOutcome |
