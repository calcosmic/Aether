# Phase 202 Classic Coverage Audit

This is the human rendering of `202-CLASSIC-COVERAGE.json`. The JSON ledger is
the machine-validated source; each line below names one capability the old
Classic system had, where that capability lives in the program today, and the
name of the check that would fail if it ever stopped being true.

Note: all three of Phase 202's owner walk-through tests (`202-UAT.md`) were
skipped -- none of the eight rows below has ever been directly observed by
the owner running a live Swarm or Oracle session. Every proof named here is a
program test, not an owner-witnessed behavior. See the plan 205-07 SUMMARY
for the full accounting.

## GOAL

| ID | Disposition | Modern home | Plan/task | Proof |
|---|---|---|---|---|
| CAP-021 | restore-modern | cmd/oracle_synthesis.go | 202-12 | TestSynthesisLeadsWithTheRecommendation |
| CAP-044 | restore-modern | cmd/swarm_repair_checkpoint.go | 202-07 | TestSwarmRepairCheckpointsBeforeTheFixWave |
| CAP-045 | restore-modern | cmd/swarm_episode.go | 202-10 | TestSuccessfulSwarmProposesOneScopedNote |
| CAP-046 | restore-modern | cmd/swarm_strikes.go | 202-07 | TestThirdStrikeRendersAnArchitecturalCase |
| CAP-047 | restore-modern | cmd/swarm_episode.go | 202-10 | TestSwarmRunProducesOneReplaySafeEpisode |
| CAP-048 | replace-better | cmd/swarm_episode.go | 202-10 | TestSwarmRemovalRequiresIdentifierAndDigest |
| CAP-063 | restore-modern | cmd/swarm_repair_checkpoint.go | 202-07 | TestSwarmRepairRollsBackOnFailedVerification |
| CAP-072 | restore-modern | cmd/episode_index.go | 202-14 | TestEpisodeIndexCoversThreeRecordSources |
