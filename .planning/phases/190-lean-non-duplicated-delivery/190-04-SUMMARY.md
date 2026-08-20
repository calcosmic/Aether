# 190-04 — Review findings closure (convergence record)

The four findings of 190-REVIEW.md (CR-01 handoff_section double-ship, WR-01 stale
brief accumulation, WR-02 zero-blind duplication gate, WR-03 hive-equivalence
overclaim) were closed TWICE in parallel on 2026-08-20:

1. **A parallel session** committed fixes directly to `oracle-reinstate`
   (`68a5101b` "fix(190): close the review's findings, with locks that fail
   without them" + `02753b19` tracking). This version also refreshed the three
   stale ceremony snapshots (crown→crown+ant) that predated the phase.
2. **The orchestrator's dispatched fixer** produced an equivalent, independently
   proven fix in an isolated worktree (its commits were preserved under a
   temporary `rescue-190-04` ref during convergence).

**Resolution:** the parallel session's version, already on the main line, was
audited finding-by-finding by the orchestrator and covers all four
(`TestPlanOnlyDispatchesCarryNoHandoffSection`,
`TestPlanOnlyRerunDoesNotAccumulateStaleWorkerBriefs`,
`TestPrintBriefGateCatchesAbsentSectionNotJustDuplicated`,
`TestPrintBriefFailsWhenAnExpectedSectionReachesNoWorker`, plus
`cleanupStaleWorkerBriefs` and the D-190-04-A deferred decision). The one
protection unique to the dispatched fixer —
`TestNativeDispatchHandoffStaysExactlyOnceViaCapsule`, the native-route
zero-vs-once guard — was ported onto the main line
(cmd/build_review_190_findings_test.go) and passes against this architecture.
The superseded implementation's full summary is preserved in the session
scratchpad; its branch was deleted after the port to avoid exactly the orphaned
-branch rot this milestone exists to end.

Process note for the record: two agents fixing the same findings concurrently
came from the owner operating a second session alongside this one. Harmless this
time — both fixes were correct — but merge conflicts between two honest fixes
are wasted work; coordinate future parallel sessions on disjoint phases.
