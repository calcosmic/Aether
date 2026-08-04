---
phase: 162
slug: switch-on-learning
status: planned
nyquist_compliant: true
wave_0_complete: false
created: 2026-08-04
updated: 2026-08-04
---

# Phase 162 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test (stdlib) |
| **Config file** | none — standard `go test` |
| **Quick run command** | `go test ./cmd/ -run 'Consolidation\|Hive\|ContinueFinalize\|Seal\|Learning\|MemoryInjection' -count=1` |
| **Full suite command** | `go test ./... -count=1 -timeout 900s` |
| **Estimated runtime** | ~120 seconds full, ~25 seconds quick |

---

## Sampling Rate

- **After every task commit:** Run quick command
- **After every plan wave:** Run full suite
- **Before `/gsd-verify-work`:** Full suite must be green, plus `go test ./... -race -count=1 -timeout 2400s`
- **Max feedback latency:** 120 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 01-T1 | 01 | 1 | LEARN-04 | T-162-03, T-162-04 | Unset means promote; explicit off still off; typo fails safe with a named warning | unit (table) | `go test ./cmd/ -run 'TestHiveRuntimePolicy' -count=1` | ❌ W0 | ⬜ pending |
| 01-T2 | 01 | 1 | LEARN-04 | T-162-03 | Single control surface — no consent file can veto the policy | unit/integration | `go test ./cmd/ -run 'Hive\|ColonyPrime\|Parity\|Catalog' -count=1` | ⚠️ partial | ⬜ pending |
| 01-T3 | 01 | 1 | LEARN-04 | T-162-01, T-162-03 | Docs name what leaves the repo boundary and the one switch that stops it | doc-assert | `! grep -n 'opt-in, twice over\|hive-opt-in' AGENTS.md` | ✅ | ⬜ pending |
| 02-T1 | 02 | 1 | LEARN-01, LEARN-03 | T-162-06 | Promotion target is the file colony-prime reads; existing QUEEN.md content preserved | integration | `go test ./cmd/ -run 'TestConsolidationQueenPath\|TestEnsureQueenInstinctsSection\|TestConsolidationPromotesIntoLocalQueen' -count=1` | ❌ W0 | ⬜ pending |
| 02-T2 | 02 | 1 | LEARN-01 | T-162-07 | Instincts section reaches the worker prompt; preferences/charter stay excluded | unit | `go test ./cmd/ -run 'TestPromotedInstinctReachesWorkerPrompt\|TestReadQUEENMdIngestsInstinctsSection' -count=1` | ❌ W0 | ⬜ pending |
| 02-T3 | 02 | 1 | LEARN-01 | T-162-06 | Dry run mutates nothing, including the relocated QUEEN.md | unit | `go test ./cmd/ -run 'TestConsolidation.*DryRun' -count=1` | ⚠️ extend | ⬜ pending |
| 03-T1 | 03 | 2 | LEARN-01 | T-162-10, T-162-11, T-162-12 | Non-blocking, timeout-bounded, sentinel abort named in the loud warning | unit/integration | `go test ./cmd/ -run 'TestRunPhaseEndConsolidation' -count=1` | ❌ W0 | ⬜ pending |
| 03-T2 | 03 | 2 | LEARN-01 | T-162-13 | Fires on durable advance only; never on a non-advancing continue | integration | `go test ./cmd/ -run 'ContinueAdvanceInvokesPhaseEndConsolidation\|ExternalContinueAdvanceInvokesPhaseEndConsolidation\|ContinueWithoutAdvanceDoesNotConsolidate' -count=1` | ❌ W0 | ⬜ pending |
| 03-T3 | 03 | 2 | LEARN-01 | T-162-13 | The beat prints in all four states — silence is unreachable | unit (render) | `go test ./cmd/ -run 'TestCurationAntCasteIdentitiesAreDistinct\|TestContinueVisualAlwaysRendersLearningBeat' -count=1` | ❌ W0 | ⬜ pending |
| 04-T1 | 04 | 3 | LEARN-02 | T-162-14, T-162-17 | Eight ants kept individually; report persisted; seal never blocked | integration | `go test ./cmd/ -run 'TestRunSealConsolidation' -count=1` | ❌ W0 | ⬜ pending |
| 04-T2 | 04 | 3 | LEARN-03 | T-162-15 | One instinct, exactly one QUEEN.md entry — counted, not merely present | integration | `go test ./cmd/ -run 'TestSeal' -count=1` | ⚠️ extend | ⬜ pending |
| 04-T3 | 04 | 3 | LEARN-02 | T-162-16, T-162-18 | Eight named ants and a report path that exists; loud failure line otherwise | integration (render) | `go test ./cmd/ -run 'TestSealRendersEightNamedAnts\|TestSealRendersReportPath\|TestSealRendersLoudFailure' -count=1` | ❌ W0 | ⬜ pending |
| 05-T1 | 05 | 4 | LEARN-05 | — | Brief differs populated vs wiped, with no model calls | unit (deterministic) | `go test ./cmd/ -run 'TestWorkerBriefMemoryInjection\|TestResolveCodexWorkerContextCarriesMemory' -count=1` | ❌ W0 | ⬜ pending |
| 05-T2 | 05 | 4 | LEARN-05 | T-162-19, T-162-21 | Real before/after recorded, not assumed; live memory restored afterwards | manual/exhibit | `test -s .planning/phases/162-switch-on-learning/162-MEMORY-PROOF.md` | ❌ W0 | ⬜ pending |
| 06-T1 | 06 | 4 | LEARN-03, LEARN-04 | T-162-24 | Both decisions written down with reasoning and named enforcing tests | doc-assert | `test -f .aether/docs/learning-system-authority.md` | ❌ W0 | ⬜ pending |
| 06-T2 | 06 | 4 | LEARN-03 | T-162-22 | No document claims consolidation is unwired or that phase-end runs three ants | doc-assert | `! grep -rn 'Phase 162 wires this\|three ants only' CLAUDE.md AGENTS.md .aether/docs/` | ✅ | ⬜ pending |
| 06-T3 | 06 | 4 | LEARN-03, LEARN-04 | T-162-22, T-162-23 | Retired claims cannot rot back; the record cannot vanish silently | unit (doc guard) | `go test ./cmd/ -run 'TestLearningDocsDoNotClaimUnwiredConsolidation\|TestLearningDecisionRecordExists' -count=1` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*
*File Exists: ✅ exists · ⚠️ partial (extend an existing file) · ❌ W0 (new file created by the task itself)*

---

## Wave 0 Requirements

- [x] Existing infrastructure covers framework needs (`go test` already configured; CI runs it)
- [ ] New test files are created by the same task that creates the behaviour they guard — every task in plans 01-06 that produces code carries its own `<automated>` command, so there is no separate Wave 0 scaffolding step
- [ ] Two pre-existing guards must keep passing unmodified throughout: `TestConsolidationPhaseEndDryRunDoesNotMutate`, `TestConsolidationSealDryRunDoesNotMutate`
- [ ] Five pre-existing seal guards must keep passing: `TestSealPromoteInstincts`, `TestSealHiveEligibleLog`, `TestSealHivePromote`, `TestSealHivePromoteNonBlocking`, `TestSealHivePromotedCount` (only permitted edit: adding an explicit `AETHER_HIVE_POLICY=off` where a test targets the disabled branch, since plan 01 flips the default)

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Layer-2 before/after artifact (one real worker run each way) | LEARN-05 | Requires live model calls — deliberately excluded from CI per D-08 | Plan 05 Task 2 (`checkpoint:human-verify`): run `aether build <n> --print-brief` with memory populated, rename `instincts.json` and `.aether/QUEEN.md` to `.bak`, rerun, restore, record both outputs and the stated difference in `162-MEMORY-PROOF.md`. A null result is a failure, not a pass. |
| Ceremony visual appearance (🧠 beat, eight-ant announcements, ANSI colour) | LEARN-01, LEARN-02 | Emoji width and ANSI rendering are terminal-dependent | Run `/ant-continue` at a phase end and confirm the Learning stage renders one 🧠 line; run `/ant-seal` on a scratch colony and confirm eight distinct named ants render with no generic `🐜 Ant` fallback. Note observations in `162-MEMORY-PROOF.md`. |
| Cross-repo hive arrival with no configuration | LEARN-04 | Requires a second repository on the same machine | Seal this colony, then in a different Aether repo run `aether build <n> --print-brief` with `AETHER_HIVE_POLICY` unset and confirm a `## HIVE WISDOM` section carrying this repo's wisdom. This is CONTEXT.md's acceptance story; the deterministic half is covered by `TestWorkerBriefMemoryInjection`. |
</content>
