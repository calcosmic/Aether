# Plan 160-03 — Summary

**Status:** Complete (all 3 tasks executed and committed)
**Requirements:** LOUD-06, LOUD-07
**Completed:** 2026-07-27

> **Provenance note:** this SUMMARY was written by the phase orchestrator, not by the
> executor agent. The executor completed and committed all three tasks, then was
> terminated by an account spend limit while running the final full-suite check —
> before it could write its own summary. The content below is reconstructed from the
> three task commits and verified against the merged tree, not from the agent's report.
> Every claim here was re-checked after merge.

## What changed

**Task 1 — stop the false runtime claims** (`261fe4d7`)

Three files asserted, in the present tense, that memory consolidation runs automatically
at `/ant-continue` and `/ant-seal`. It never has. Corrected:

- `.aether/docs/structural-learning-stack.md` — four separate claims fixed, including the
  lifecycle table, plus a new explicit "Wiring status" callout stating that both subcommands
  exist and work but no lifecycle command invokes them.
- `AGENTS.md` — the "Lifecycle integration" line, plus a stale `Last Updated: 2026-05-20`
  header that predated the corrections.
- `CLAUDE.md` — the one remaining false line, which contradicted an honest statement
  earlier in the same file.

Each correction names Phase 162 as the phase that will make the original claim true.
The wording is "does not run today", not a promise about the future.

**Scope correction found during execution:** the requirement text (LOUD-07) names only
`consolidation-phase-end`. Both `consolidation-phase-end` *and* `consolidation-seal` have
zero callers, so both claims were false. Both were corrected.

**Task 2 — make the claim untestable-by-assertion** (`8f32d29d`)
`cmd/doc_consolidation_claims_test.go` (194 lines) fails if any of the three documents
re-asserts that consolidation runs at continue or seal while no caller exists.

**Task 3 — stderr suppression invariant** (`40f1fa72`)
`cmd/live_wrapper_stderr_test.go` (109 lines) pins the count of `2>/dev/null` occurrences
in the two live wrapper directories. Per CLAUDE.md's Definition of Done corollary, this is
a count invariant rather than a check that named files exist — it catches a new suppression
site whatever it is called.

**Scope decision recorded:** the audit is scoped to the two *live* wrapper directories.
The `.aether/docs/command-playbooks/` corpus contains 179 further `2>/dev/null` sites, but
those files are not loaded by the runtime, so they are documentation, not live behaviour.

## Verification

Run after merge into main, on the combined result of plans 01+02+03:

```
go build ./cmd/aether     → clean
go vet ./...              → clean
go test ./... -count=1    → exit 0
```

Both new tests pass:
- `TestDocsDoNotClaimConsolidationRunsToday` — PASS
- `TestLiveWrapperStderrSuppressionCount` — PASS

## Caveat

The executor was killed during its own final full-suite run, so its self-check never
completed. The orchestrator ran the full suite after merge instead, and it passed. The
deliberate-regression checks in this plan's acceptance criteria (break it, observe the
failure, revert, quote the message) were **not** independently confirmed by the orchestrator
— the tests pass in the green state, but their failure behaviour has not been demonstrated
for this plan. Worth confirming during phase verification.

## Files

- `.aether/docs/structural-learning-stack.md` (modified)
- `AGENTS.md` (modified)
- `CLAUDE.md` (modified)
- `cmd/doc_consolidation_claims_test.go` (created)
- `cmd/live_wrapper_stderr_test.go` (created)

## Commits

- `261fe4d7` docs(160-03): stop claiming consolidation runs at phase-end or seal
- `8f32d29d` test(160-03): add static assertion against consolidation-runs claims
- `40f1fa72` test(160-03): pin live-wrapper stderr suppression to a count invariant
