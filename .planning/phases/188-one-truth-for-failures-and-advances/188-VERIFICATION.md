---
phase: 188-one-truth-for-failures-and-advances
verified: 2026-08-21T09:34:23Z
status: passed
score: 6/6 must-haves verified
overrides_applied: 0
re_verification:
  previous_status: gaps_found
  previous_score: 4/6
  gaps_closed:
    - "A failure written by any component is read by colony-prime, autopilot, immune and memory-health (midden path unified)"
    - "Ungated state-mutate --field current_phase is refused"
  gaps_remaining: []
  regressions: []
---

# Phase 188: One Truth for Failures and Advances Verification Report

**Phase Goal:** One failure ledger; one phase-advance discipline; every retry loop leaves a record.
**Verified:** 2026-08-21T09:34:23Z
**Status:** passed
**Re-verification:** Yes — final pass after 188-07 closed the two gaps this report's prior pass (0b790c40, 2026-08-20T00:24:59Z, gaps_found 4/6) found.

## Environment Note

This worktree's branch was checked out at a stale base commit (`6577f51c`, 2026-07-28, ~v1.0.43,
967 files behind) when this verification session started — the phase 188 directory did not exist
on disk at all. The working tree was already clean (nothing to commit), so it was hard-reset to
`0a30509f` ("docs(phase-188): update tracking after gap closure", the exact tip of the 188-07 gap
closure work, an ancestor of the main tree's `oracle-reinstate` branch) before any verification
work began. No uncommitted work existed at the stale commit, so nothing was lost. `go build ./...`
and `go vet ./...` were confirmed clean immediately after the reset.

## What Changed Since the Prior Ruling

Two commits (`dd3e584c`, `00eb11ce`) plus a doc commit (`b439f8ed`) landed as plan 188-07,
touching exactly six files: `cmd/state_cmds.go`, `cmd/state_cmds_test.go`,
`cmd/colony_prime_context.go`, `cmd/context_weighting.go`, `cmd/midden_unification_test.go`,
`cmd/colony_prime_audit_test.go`. No other file from the phase's earlier six plans (188-01
through 188-06) was touched.

## Goal Achievement

### Observable Truths

| # | Truth (ROADMAP Success Criterion) | Status | Evidence |
|---|---|---|---|
| 1 | A failure written by any component is read by colony-prime, autopilot, immune and memory-health (midden path unified) | ✓ VERIFIED | `buildColonyPrimeOutputOpts` (cmd/colony_prime_context.go:949-991) — the function `resolveCodexWorkerContext()`/`resolveCodexWorkerContextWithTrim()` actually calls, i.e. every live build/continue/colonize/plan/seal/swarm dispatch — now renders a "Recent Failures" section sourced from `loadMiddenFile`. Confirmed by DIRECT EXECUTION: ran `TestOneMiddenEntryReachesColonyPrimeCapsule` (writes via `appendMiddenEntry`, calls `resolveCodexWorkerContext()`, asserts the message appears exactly once) — PASS. Then reverted the fix (made the midden block's condition always-false), re-ran the same test plus `TestColonyPrimeAAC005Audit`/`TestColonyPrimeSectionsPresent` — all 3 FAILED with the correct symptom ("Gap 1 reopened" / "midden" section missing from the 16-section checklist) — then restored the file byte-for-byte (`git status --porcelain` confirmed clean) and re-ran green. Autopilot, memory-health, and immune's consumption (already fixed in 188-01) re-confirmed still passing via `TestOneMiddenEntryReachesAllFourConsumers`. |
| 2 | Both continue paths advance through one shared `advancePhase()` with the supersession check | ✓ VERIFIED | Unchanged by 188-07 (file not touched). Re-ran `TestAdvancePhaseHappyPath`, `TestAdvancePhaseRefusesOnBuildStartedAtMismatch`, `TestAdvancePhaseRefusesWhenPhaseNotInProgress`, `TestCommitBuildFinalizeStateDoesNotOverwritePausedState`, `TestFinalizeBlockedExternalContinueDoesNotOverwritePausedState` directly — all PASS, no regressions from the two 188-07 fixes. |
| 3 | Ungated `state-mutate --field current_phase` is refused | ✓ VERIFIED | `executeFieldMode` (cmd/state_cmds.go:270-272) now uses `strings.EqualFold` to route ANY casing of `current_phase` into the guarded switch case (never the generic passthrough); `guardCurrentPhaseSubExpression` (line 436-439) and the new `normalizeCurrentPhaseExpressionCasing` (line 460-472) close the identical hole for the expression-syntax form and prevent a stray differently-cased duplicate key. Confirmed by DIRECT EXECUTION of the phase's own 6 new tests (`TestStateMutateField[Case/Mixed]CaseInsensitiveCurrentPhase*`, `TestStateMutateExpression[Case/Mixed]CaseInsensitiveCurrentPhase*`) — all PASS. Then wrote and ran 5 of my own throwaway probes covering variants none of the existing tests used: a third casing style (`current_PHASE`, both `--field` and expression forms), a mismatched-guard-target on a differently-cased field (`--field CURRENT_PHASE --guard phase-advance:3` while writing phase 2 — still refused), confirmation that an unrelated field (`goal`) is NOT accidentally case-folded by the fix, and an independent stray-key check for `--field` mode. All 5 PASSED, disk unchanged on every refusal. Probe file deleted after execution; `git status --porcelain` confirmed clean. |
| 4 | Grep-ratchet against non-atomic COLONY_STATE writes | ✓ VERIFIED | Re-ran all 3 AST-based ratchet tests directly: `TestColonyStateWriteAllowlistOnlyShrinks`, `TestNoFunctionReachableFromAdvancePhaseWritesNonAtomically`, `TestNoUpdateJSONAtomicallyDiscardsFreshRead` — all PASS. The write-site baseline is still exactly 29 entries (verified by direct JSON parse); `cmd/state_cmds.go`'s 3 pre-existing allowlist entries (`executeExpression`/`AtomicWrite`, `executeFieldMode`/`SaveJSON`, `executeRevertGuard`/`AtomicWrite`) are unchanged by 188-07's guard-logic edits, since the fix only added case-insensitive dispatch/guarding, not new write primitives. `colonyStateDiscardedReadAllowlist` (the separate, in-file discard-site ledger) still has exactly its one pre-existing entry (`cmd/codex_build.go`:`rollbackCodexBuildFailure`) — see Deferred Items below. |
| 5 | An exhausted retry loop leaves a readable record of what was tried | ✓ VERIFIED | Unchanged by 188-07 (file not touched). Re-ran `TestRecordAutopilotRetryExhaustion` (all 4 subtests), `TestAutopilotRetryExhaustionCallSiteIsWired`, `TestGoldenAutopilotPauseConditions` (all 3 subtests) directly — all PASS. |
| 6 | A loud warning is logged when build-finalize accepts a legacy unbound manifest | ✓ VERIFIED | Unchanged by 188-07 (file not touched). Re-ran `TestBuildFinalizeWarnsOnLegacyUnboundManifest` (both subtests) directly — PASS. |

**Score:** 6/6 truths verified

### Required Artifacts

| Artifact | Expected | Status | Details |
|---|---|---|---|
| `cmd/colony_prime_context.go` (`buildColonyPrimeOutputOpts`) | Colony-prime's real worker-context midden section | ✓ VERIFIED | Lines 935-992: reads `loadMiddenFile`, filters to unacknowledged entries, sorts newest-first, hard-caps at 5 with a "+N more" tail, marks the section `protected: true`. Confirmed present, substantive, and wired (see truth 1 above). |
| `cmd/context_weighting.go` (`protectedSectionPolicy`/`sectionRelevanceScore`) | "midden" case in both functions | ✓ VERIFIED | `sectionRelevanceScore("midden")` = 0.90 (line 215-216); `protectedSectionPolicy("midden")` returns `(true, "unresolved failures must survive trimming")` (line 238-255), with an inline doc comment explaining the trim-order placement decision. |
| `cmd/state_cmds.go` | Case-insensitive current_phase guard, both invocation forms | ✓ VERIFIED | `executeFieldMode` (270-272), `guardCurrentPhaseSubExpression` (422-446), `normalizeCurrentPhaseExpressionCasing` (460-472, new function). All three read and confirmed to implement exactly what 188-07-SUMMARY.md claims. |
| `cmd/midden_unification_test.go` | Regression coverage for the real capsule path | ✓ VERIFIED | `TestOneMiddenEntryReachesColonyPrimeCapsule` (asserts message present exactly once — the Phase 190 one-home law), `TestMiddenCapsuleSectionOmittedWhenNoFailures`, `TestMiddenCapsuleSectionOmitsAcknowledgedEntries`. All read in full and re-executed; not self-referential (asserts against `resolveCodexWorkerContext()`'s real output, not a re-read of the same fixture path). |
| `cmd/colony_prime_audit_test.go` | Fixed stale nested-path fixtures | ✓ VERIFIED | `TestColonyPrimeAAC005Audit` and `TestColonyPrimeSectionsPresent` re-seeded at the canonical flat `midden.json` path with a real assertion against captured capsule text (not a self-consistent readback of the same wrong path as before). Confirmed via the mutation test: both now correctly FAIL when the midden section is disabled. |

### Key Link Verification

| From | To | Via | Status | Details |
|---|---|---|---|---|
| `resolveCodexWorkerContext()` → `buildColonyPrimeOutput` → `buildColonyPrimeOutputOpts` | `cmd/midden_shared.go` | `loadMiddenFile(store)` | ✓ WIRED | This is the fix itself — confirmed by direct execution (`TestOneMiddenEntryReachesColonyPrimeCapsule`) and by the mutation/revert test proving the wiring is load-bearing, not vestigial. |
| `cmd/state_cmds.go` (`executeFieldMode`) | `validateCurrentPhaseGuard` | `strings.EqualFold` dispatch routes any casing into the guarded switch case | ✓ WIRED | Confirmed by 11 total direct-execution probes (6 phase tests + 5 independent verifier probes) across both invocation forms and 3+ casing styles. |
| `cmd/state_cmds.go` (`executeExpression`) | `guardCurrentPhaseSubExpression` → `normalizeCurrentPhaseExpressionCasing` | Guard runs before every sub-expression is applied; normalization prevents stray duplicate keys | ✓ WIRED | Confirmed: guarded writes with mismatched casing succeed and leave no stray key (checked via `gjson.GetBytes(rawData, "CURRENT_PHASE").Exists()` returning false in both the phase's own tests and my independent probe). |
| One midden delivery channel (Phase 190's one-home law) | n/a | Structural sweep | ✓ CONFIRMED SINGLE-CHANNEL | Grepped every "midden" reference across `cmd/*.go` (non-test): exactly one worker-prompt-facing channel exists (`buildColonyPrimeOutputOpts`'s new section). `cmd/context.go:235` (`buildResumeDashboardResult`, the `/ant-resume` dashboard) and `cmd/context.go:857` (`prContextCmd`'s JSON `midden` field) are separate, non-prompt, human/API-facing outputs, not worker dispatch content. No `MiddenSection` field exists on `codex.WorkerConfig` or `codex.WorkerDispatch` (grepped both struct definitions) that could deliver a second copy the way Phase 190 found for pheromones. Re-ran the Phase 190 one-home ratchets (`TestNativeDispatchPheromoneStaysExactlyOnceViaCapsule`, `TestEightCommandsDeliverPheromoneExactlyOnce`, `TestColonyPrimeAndDispatchContractDoNotReappear`) as a regression check — all still PASS (these are pheromone-scoped, not midden-scoped, but confirm the surrounding capsule-assembly machinery is undisturbed). |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
|---|---|---|---|---|
| `buildColonyPrimeOutputOpts`'s "midden" section | `unacked` (filtered `midden.Entries`) | `loadMiddenFile(store)` → `middenCanonicalPath` ("midden.json") | Yes — direct execution proved a real `appendMiddenEntry` write reaches `resolveCodexWorkerContext()`'s output exactly once | ✓ FLOWING |
| `RankContextCandidates`'s protected-tier handling of the midden candidate | `sec.protected` | `protectedSectionPolicy("midden")` | Yes — read `pkg/colony/context_ranking.go:65-95`: protected candidates are appended to `result.Included` unconditionally, before any budget check runs against the regular tier | ✓ FLOWING, unconditional inclusion confirmed by source read |

### Trim-Order Honesty (Task Probe 3)

**Protected/never-trimmed claim:** Confirmed genuine, not aspirational. `pkg/colony/context_ranking.go`'s `RankContextCandidates` (read in full) partitions candidates into `protected`/`regular` before any budget arithmetic; the protected loop (lines 86-95) appends every protected item to `result.Included` with no `budget` check anywhere in that loop — only the `regular` loop (line 97+) checks `currentLen+itemCost > budget`. This exactly matches 188-07-SUMMARY.md's claim.

**Bounded-content safety claim:** Confirmed. The midden section's content is hard-capped at 5 unacknowledged entries (`middenCapsuleSectionEntryLimit`, cmd/colony_prime_context.go:961) regardless of how large `midden.json` grows, so "never trimmed" cannot become "unbounded" the way an ever-growing failure log could.

**CLAUDE.md contradiction check:** No contradiction found. This worktree's CLAUDE.md (v1.0.59) trim-order list — "Rolling summary → Phase learnings → Key decisions → Hive wisdom → Context capsule → User preferences → QUEEN wisdom (global) → QUEEN wisdom (local) → Pheromone signals; Blockers are NEVER trimmed" — does not mention "midden" at all, so there is nothing for the new section to contradict; it is an omission, not a false claim. This omission also is not new or specific to 188-07: the same trim-order list already excludes several other pre-existing protected sections (state, charter, clarified_intent, global_queen_md, user_preferences — all `protected: true` in source) and several ordinary sections (instincts, prior_reviews, worker_handoffs, medic_health, review_depth) that predate this phase entirely. This is pre-existing, broader documentation drift, not something 188-07 introduced or worsened. **It is also already scheduled**: ROADMAP.md's Phase 191 success criterion (4) explicitly reads "false docs corrected (workers.md:825, **CLAUDE.md trim-order** and host-build claims)" — i.e., this exact gap is a named, already-planned deliverable of a later phase. Per Step 9b, this is filed as informational/deferred, not a Phase 188 gap.

### Probe Execution

All probes below were executed directly by me, in this session, then deleted or reverted;
`git status --porcelain` confirmed clean after every one.

| Probe | Command | Result | Status |
|---|---|---|---|
| Gap 1 positive re-run | `go test ./cmd/ -run 'TestOneMiddenEntryReachesAllFourConsumers\|TestOneMiddenEntryReachesColonyPrimeCapsule\|TestMiddenCapsuleSectionOmittedWhenNoFailures\|TestMiddenCapsuleSectionOmitsAcknowledgedEntries' -v -count=1` | All 4 PASS | ✓ PASS |
| Gap 1 mutation/revert test | Disabled the midden section's render condition (`false && middenErr == nil && ...`) in `cmd/colony_prime_context.go`, re-ran the above plus `TestColonyPrimeAAC005Audit`/`TestColonyPrimeSectionsPresent` | 3 tests FAILED with "Gap 1 reopened" / missing "midden" section — correct symptom, not a vacuous pass | ✓ CONFIRMS TEST IS LOAD-BEARING |
| Restore + re-verify | Restored `cmd/colony_prime_context.go` from a pre-mutation backup; `git status --porcelain` empty; re-ran all 6 tests | All 6 PASS again | ✓ RESTORED BYTE-CLEAN |
| Gap 2 positive re-run | `go test ./cmd/ -run 'TestStateMutate' -v -count=1` (27 tests incl. the 6 new case-insensitivity tests) | All PASS | ✓ PASS |
| Gap 2 independent probes (new variants) | Wrote `cmd/zzz_verifier_probe_188_test.go`: third casing style (`current_PHASE`) for both invocation forms, mismatched-guard-target on a differently-cased field, unrelated-field (`goal`) non-interference, `--field`-mode stray-key check | All 5 PASS | ✓ PASS, then file deleted |
| Criterion 2 regression | `go test ./cmd/ -run 'TestAdvancePhase\|TestCommitBuildFinalizeStateDoesNotOverwritePausedState\|TestFinalizeBlockedExternalContinueDoesNotOverwritePausedState' -v -count=1` | All PASS | ✓ PASS |
| Criterion 4 regression | `go test ./cmd/ -run 'TestColonyStateWriteAllowlistOnlyShrinks\|TestNoFunctionReachableFromAdvancePhaseWritesNonAtomically\|TestNoUpdateJSONAtomicallyDiscardsFreshRead' -v -count=1` | All PASS; allowlist still 29 entries (write-site) + 1 entry (discard-site) | ✓ PASS |
| Criteria 5 & 6 regression | `go test ./cmd/ -run 'TestRecordAutopilotRetryExhaustion\|TestAutopilotRetryExhaustionCallSiteIsWired\|TestGoldenAutopilotPauseConditions\|TestBuildFinalizeWarnsOnLegacyUnboundManifest' -v -count=1` | All PASS | ✓ PASS |
| Full colony-prime suite regression | `go test ./cmd/ -run 'TestColonyPrime' -v -count=1` (49 tests) | All PASS | ✓ PASS |
| Phase 190 one-home ratchets regression | `go test ./cmd/ -run 'TestColonyPrimeAndDispatchContractDoNotReappear\|TestNativeDispatchPheromoneStaysExactlyOnceViaCapsule\|TestEightCommandsDeliverPheromoneExactlyOnce' -v -count=1` | All PASS | ✓ PASS |
| Deferred item (item 5) ledger check | `git show --stat dd3e584c 00eb11ce` (confirms `cmd/codex_build.go` untouched); grepped `current = rollback` still present at cmd/codex_build.go:2628 | `rollbackCodexBuildFailure`'s clobber pattern unchanged; `colonyStateDiscardedReadAllowlist` still has exactly its 1 pre-existing entry | ✓ CONFIRMED STILL OPEN BY DESIGN, NOTHING NEW JOINED |
| Anti-pattern sweep | `grep -n -E "TBD\|FIXME\|XXX\|TODO\|HACK\|PLACEHOLDER"` across all 19 phase-modified files; `gofmt -l` on the 188-07 subset | Zero hits on both | ✓ CLEAN |
| Whole-repo sanity | `go build ./...`, `go vet ./...` (not the full test suite, per instruction not to re-run it) | Both clean, exit 0 | ✓ PASS |
| Tree cleanliness | `git status --porcelain` after every probe/mutation | Empty every time | ✓ CONFIRMED |

### Deferred Items

Not roadmap gaps — informational, explicitly scoped as out-of-phase-188 by the phase's own
deferred-items.md and, for the second item, by ROADMAP.md itself.

| # | Item | Status | Evidence |
|---|---|---|---|
| 1 | `cmd/codex_build.go`'s `rollbackCodexBuildFailure` discards its own fresh `UpdateJSONAtomically` read (the "fifth" stale-write-discard instance found across the whole 188 fix round — CR-01/CR-02/CR-03/Gap-2 being the other four, this one found by WR-01's ratchet sweep) | Still open by design | `colonyStateDiscardedReadAllowlist` in `cmd/colony_state_atomicity_ratchet_test.go` still has exactly 1 entry (`cmd/codex_build.go`/`rollbackCodexBuildFailure`/`current`); the underlying code (`current = rollback` at line 2628) is byte-identical to what deferred-items.md #3 describes — confirmed via `git show --stat` that 188-07's commits never touched this file. `TestNoUpdateJSONAtomicallyDiscardsFreshRead` re-ran clean: no new unlisted discard sites, no stale allowlist entries. |
| 2 | CLAUDE.md's documented trim order never names "midden" (or several other pre-existing protected sections) | Addressed in Phase 191 | ROADMAP.md Phase 191 success criterion (4): "false docs corrected (workers.md:825, **CLAUDE.md trim-order** and host-build claims)". Not a contradiction (see Trim-Order Honesty above), and this repo's own Phase 191 scope already names the exact document. |

### Requirements Coverage

Not applicable. All seven plans (188-01 through 188-07) declare `requirements: []` in frontmatter,
and `.planning/REQUIREMENTS.md` has zero entries mapped to "Phase 188" (confirmed by direct grep,
no matches). No orphaned requirements.

### Anti-Patterns Found

None (blocking or warning). Scanned all 19 files touched across the phase's full history
(188-01 through 188-07) for `TBD|FIXME|XXX|TODO|HACK|PLACEHOLDER` and placeholder-language
patterns: zero hits. `gofmt -l` on the 188-07 subset: zero hits.

Two pre-existing INFO-level items carried forward unchanged from the prior VERIFICATION.md
(neither touched by 188-07, neither blocking):
- `cmd/medic_scanner.go:511` still writes the literal `"midden.json"` instead of referencing
  `middenCanonicalPath`. Correct value today; cosmetic.
- `cmd/midden_shared.go:38`: `appendMiddenEntry`'s ID (`midden_<unix-sec>_<pid>`) can collide
  within the same wall-clock second. Pre-existing pattern from the writers it replaces.

### Human Verification Required

None. Every observable truth in this phase is backend/CLI logic verifiable by direct code reading
and execution; nothing requires visual, real-time, or external-service testing.

### Gaps Summary

None. Both gaps from the prior pass (0b790c40, gaps_found 4/6) are closed and independently
reproduced by direct execution, including a mutation/revert test proving Gap 1's new regression
test is load-bearing (not vacuously passing) and 5 additional independent probes covering variants
the phase's own tests did not try for Gap 2. All four previously-passing criteria (2, 4, 5, 6) were
re-swept and show no regressions from the two fixes. The one still-open deferred item
(`rollbackCodexBuildFailure`'s discard site) is confirmed unchanged and still correctly tracked by
its shrink-only allowlist — nothing new joined it. The phase goal — "One failure ledger; one
phase-advance discipline; every retry loop leaves a record" — is now observably true in the
codebase, not merely claimed in a SUMMARY.

---

_Verified: 2026-08-21T09:34:23Z_
_Verifier: Claude (gsd-verifier)_
