---
phase: 162-switch-on-learning
audited: 2026-08-04
asvs_level: L1
block_on: open_threats
threats_total: 24
threats_closed: 24
threats_open: 0
status: secured
---

# Security Audit — Phase 162: Switch On Learning

**Audited:** 2026-08-04
**ASVS Level:** L1
**Block on:** open_threats
**Register source:** PLAN.md `<threat_model>` blocks, plans 01-06 (register authored at plan time)
**Verified against:** current HEAD (includes post-review fix commits 94c36e5e / CR-01, 6c97cca0 / WR-01, baea5253 / WR-02, 2d9c723d / WR-03, plus the post-verification CLAUDE.md gap closure)

Every threat below was verified by disposition: `mitigate` threats were checked by grepping the cited implementation file for the mitigation pattern AND running the named pinning test(s) to confirm they currently pass (not just exist). `accept` threats were checked by confirming the acceptance rationale is present in the register and, where the rationale depends on other code being unchanged, spot-checking that code. No implementation file was modified.

## Threat Verification

| Threat ID | Category | Component | Disposition | Status | Evidence |
|-----------|----------|-----------|-------------|--------|----------|
| T-162-01 | Info Disclosure | `promoteToHiveWithReference` reachable by default | accept | CLOSED | Rationale documented in 162-01-PLAN.md register (only instinct `Action` text crosses the boundary; write side unchanged); AGENTS.md's rewritten hive section (plan 01, task 3) surfaces this to the operator. |
| T-162-02 | Tampering | Cross-repo wisdom pollution, more readers by default | accept | CLOSED | Rationale holds: `pkg/learn/hive_store.go:14` `maxHiveWisdomEntries = 200` (LRU cap present), `:35-36` `Quarantined`/`QuarantineReason` fields present — write-side controls this plan relies on are unchanged. |
| T-162-03 | Elevation of Privilege | Consent-gate removal | mitigate | CLOSED | `cmd/hive_policy.go:39-40` `case "off": return hivePolicyOff` — explicit case, not default-fallthrough. `go test ./cmd/ -run TestHiveRuntimePolicyDefault -count=1 -v` passes (12 subtests, including `off`/`OFF`/`  off  `). |
| T-162-04 | Spoofing | Typo in `AETHER_HIVE_POLICY` | mitigate | CLOSED | `cmd/hive_policy.go:45-48` `default:` branch, `sync.Once`-guarded stderr warning naming the value, returns `hivePolicyOff`. `go test ./cmd/ -run TestHiveRuntimePolicyUnrecognizedWarns -count=1 -v` passes (exactly-once assertion). |
| T-162-05 | Repudiation | Prompt injection via hive wisdom text | accept | CLOSED | Rationale documented in register: out of scope per REQUIREMENTS.md Non-Goals; sanitization on the write path unchanged by this plan. |
| T-162-06 | Tampering | `ensureQueenInstinctsSection()` rewriting user QUEEN.md | mitigate | CLOSED | `cmd/queen.go:693-733` — append-only at end of file, idempotent (returns early if header exists on disk). `go test ./cmd/ -run 'TestEnsureQueenInstinctsSectionAddsHeaderOnce|TestEnsureQueenInstinctsSectionCreatesFileWhenAbsent' -count=1 -v` both pass. |
| T-162-07 | Info Disclosure | `readQUEENMd` allowlist widening | mitigate | CLOSED | `cmd/context.go:1528-1530` — `sectionName == "Instincts"` exact match + prefix term only, same style as existing terms; no `Preferences`/`Charter` term added. `go test ./cmd/ -run TestReadQUEENMdIngestsInstinctsSection -count=1 -v` passes; test file comment (`cmd/consolidation_promotion_target_test.go:298-299`) confirms preferences/charter exclusion is asserted. |
| T-162-08 | DoS (soft) | Concurrent QUEEN.md writers (consolidation vs. `promoteInstinctLocal`) | accept | CLOSED | Rationale (plan 02 register) said plan 04 removes the overlap at seal — confirmed: plan 04's D-09 skip-set (`cmd/codex_workflow_cmds.go`) sequences the two writers rather than running them concurrently; see T-162-15. |
| T-162-09 | Tampering | Prompt injection via instinct `Action` field | accept | CLOSED | Rationale confirmed: `cmd/colony_prime_context.go:561` `## Active Instincts` section already reads `instincts.json` unconditionally, pre-dating this phase; this plan changes the route (QUEEN.md), not the exposure. |
| T-162-10 | Tampering | Partial consolidation failure after `COLONY_STATE.json` already advanced | mitigate | CLOSED | `cmd/consolidation_lifecycle.go` — `runPhaseEndConsolidation` called only after the atomic write (verified in code, see T-162-13 wiring tests); `pkg/memory/consolidate.go:88,98,113` use `store.SaveJSON` (temp+rename), `pkg/memory/queen.go:134` uses `s.store.AtomicWrite`. `go test ./cmd/ -run TestRunPhaseEndConsolidationIsNonBlockingOnFailure -count=1 -v` passes. |
| T-162-11 | DoS (soft) | Sentinel abort reported as generic failure | mitigate | CLOSED | `cmd/consolidation_lifecycle.go` — both `runPhaseEndConsolidation` and `runSealConsolidation` check `strings.Contains(reason, "sentinel abort")` and prepend `"curation sentinel detected corrupt stores: "`. Confirmed live in `TestSealRendersLoudFailure` output: `colony sealed WITHOUT consolidation — curation sentinel detected corrupt stores: ...`. |
| T-162-12 | DoS | Wedged/slow consolidation blocking phase advance | mitigate | CLOSED | `cmd/consolidation_lifecycle.go` — `runConsolidationStageBounded` (goroutine + `select` on `ctx.Done()`), used by both `runPhaseEndConsolidation` and `runSealConsolidation`. This is the post-review WR-02 strengthening (commit baea5253) that made the original `context.WithTimeout` alone enforceable — confirmed present at current HEAD. `go test ./cmd/ -run TestRunPhaseEndConsolidationTimesOutOnStaleLock -count=1 -v` passes (real stale-flock test, not a mock). |
| T-162-13 | Repudiation | Silent non-run of consolidation | mitigate | CLOSED | `go test ./cmd/ -run 'TestContinueAdvanceInvokesPhaseEndConsolidation|TestExternalContinueAdvanceInvokesPhaseEndConsolidation|TestContinueWithoutAdvanceDoesNotConsolidate|TestContinueVisualAlwaysRendersLearningBeat' -count=1 -v` — all pass, including the negative case and all four learning-beat states (populated/zero/failed/absent). |
| T-162-14 | Tampering | Consolidation corrupting `instincts.json` mid-seal | mitigate | CLOSED | `cmd/consolidation_lifecycle.go` `runSealConsolidation` — CR-01 fix (commit 94c36e5e) confirmed live in code: on `curErr != nil` (sentinel abort), function returns `Ran:false` immediately, BEFORE `pipeline.RunConsolidation` is reached — the mutating pipeline never runs against sentinel-flagged corrupt stores. `go test ./cmd/ -run TestRunSealConsolidationSentinelAbortShortCircuitsPipeline -count=1 -v` passes. |
| T-162-15 | Tampering | Double-promotion writing an instinct into QUEEN.md twice | mitigate | CLOSED | WR-01 fix (commit 6c97cca0) confirmed live: `sealConsolidationSummary.QueenPromotedIDs` sourced from `consResult.QueenPromoted` (actual successful writes only, `pkg/memory/pipeline.go`), not from `QueenEligible`. `go test ./cmd/ -run 'TestSealDoesNotDoublePromoteInstincts|TestSealDoesNotDoublePromoteOnCurationFailure' -count=1 -v` both pass (occurrence-counting assertions, not presence checks). |
| T-162-16 | Info Disclosure | `CURATION-REPORT.md` committed to a shared repo | accept | CLOSED | Rationale confirmed: `cmd/consolidation_lifecycle.go:339` writes to `filepath.Join(filepath.Dir(store.BasePath()), "CURATION-REPORT.md")`, i.e. `.aether/CURATION-REPORT.md`, beside the already-present `CROWNED-ANTHILL.md`; content sourced from the same `instincts.json`/QUEEN.md already in the working tree. |
| T-162-17 | DoS | Eight-ant pass hanging, blocking seal indefinitely | mitigate | CLOSED | Same `runConsolidationStageBounded` mechanism as T-162-12, applied to the orchestrator run in `runSealConsolidation` — WR-02 strengthening confirmed live. `go test ./cmd/ -run TestRunSealConsolidationTimesOutOnStaleLock -count=1 -v` passes (real stale-flock test). |
| T-162-18 | Repudiation | Silent seal non-run | mitigate | CLOSED | `go test ./cmd/ -run 'TestSealRendersEightNamedAnts|TestSealRendersLoudFailure' -count=1 -v` both pass. |
| T-162-19 | Tampering | Layer-2 memory-proof exercise destroying real colony memory | mitigate | CLOSED | `.planning/phases/162-switch-on-learning/162-MEMORY-PROOF.md` documents `mv instincts.json instincts.json.bak` / `mv QUEEN.md QUEEN.md.bak` (rename, not delete), explicit restoration steps, and states `git status --short` was clean before and after. |
| T-162-20 | Info Disclosure | Recorded exhibit committing real instinct/wisdom text into `.planning/` | accept | CLOSED | Rationale confirmed: exhibit text sourced from the same class of content already in `.aether/QUEEN.md`/`.aether/data/`; no credentials in the seeded sentinels (reviewed the exhibit's own quoted text). |
| T-162-21 | Repudiation | Null result recorded as a pass | mitigate | CLOSED | `162-MEMORY-PROOF.md` shows a genuine, non-identical difference (Context Capsule 958 chars populated vs. 615 chars wiped; `## Active Instincts` and `## LOCAL QUEEN WISDOM` sections present vs. absent) — not an identical-output null result, satisfying the mitigation's core requirement. Note: this task was gated as `checkpoint:human-verify`; the repo's static artifacts prove the exhibit itself is non-null, but literal confirmation that the user typed "approved" is not independently recoverable from repo files — flagged as a process note, not a code gap. |
| T-162-22 | Spoofing | Doc claims a security control (consent gate) that does not exist | mitigate | CLOSED | `cmd/docs_truth_test.go` `TestLearningDocsDoNotClaimUnwiredConsolidation` — bans `hive-opt-in`, `opt-in, twice over`, etc. across CLAUDE.md/AGENTS.md/structural-learning-stack.md. Passes at current HEAD. `.aether/docs/learning-system-authority.md` Consequences table names a real test per claim (spot-checked several: `TestHiveRuntimePolicyDefault`, `TestSealDoesNotDoublePromoteInstincts` — both confirmed passing above). |
| T-162-23 | Repudiation | Corrected claims rotting back over future milestones | mitigate | CLOSED | `go test ./cmd/ -run 'TestLearningDocsDoNotClaimUnwiredConsolidation|TestLearningDecisionRecordExists' -count=1 -v` both pass; runs on every `go test ./cmd/...`/CI. Additionally hardened post-verification: 162-VERIFICATION.md documented a gap (CLAUDE.md:37 "still have no caller" — a claim outside plan 06's original three targeted locations) that was closed the same day — `retiredLearningDocClaims` in `cmd/docs_truth_test.go:42` now includes `"still have no caller"`, and CLAUDE.md:37 confirmed reworded to past tense. `go test ./cmd/ -run TestDocsDoNotClaimConsolidationRunsToday -count=1 -v` (the pre-existing pinning test for that exact sentence) passes. |
| T-162-24 | Info Disclosure | Decision record documenting instinct text leaves the repo by default, published to hub | accept | CLOSED | Rationale: this is the intended disclosure — `.aether/docs/learning-system-authority.md:127,144,169` documents the `AETHER_HIVE_POLICY` default and full resolution table, satisfying T-162-01's dependency that this be written down. |

## Unregistered Flags

None. No `## Threat Flags` section was found in any of the six SUMMARY.md files (`162-01` through `162-06`) — the executor did not flag new attack surface beyond what the plan-time register anticipated.

## Notes on Verification Method

- All `mitigate` dispositions were verified by (1) grepping the cited implementation file for the described mitigation pattern, and (2) actually running the named pinning test(s) at current HEAD (`go test ./cmd/...` / `go test ./pkg/memory/...`) rather than trusting SUMMARY.md's claim that they pass. All ran green.
- Four threats (T-162-12, T-162-14, T-162-15, T-162-17) were explicitly re-verified against the post-review-fix commits (94c36e5e / CR-01, 6c97cca0 / WR-01, baea5253 / WR-02) per the task's instruction that these commits landed after the SUMMARYs were written and materially strengthened the original mitigations. All four are confirmed live in `cmd/consolidation_lifecycle.go` at current HEAD, not merely described in 162-REVIEW-FIX.md.
- One post-verification gap (162-VERIFICATION.md's documented CLAUDE.md:37 staleness, feeding T-162-23) was independently confirmed closed by re-reading current CLAUDE.md and re-running the specific pinning test named in the closure note.
- `accept` dispositions were treated as closed once (a) the rationale is present in the plan-time register (which is the accepted-risk log of record for this phase, reproduced above), and (b) any code fact the rationale depends on (e.g. "write-side controls unchanged," "no new content class") was spot-checked rather than taken on faith.

## Summary

**Threats Closed:** 24/24
**Threats Open:** 0/24

No implementation file was modified during this audit. This file (`SECURITY.md`) is the only artifact created.


## Audit Trail

### Security Audit 2026-08-04
| Metric | Count |
|--------|-------|
| Threats found | 24 |
| Closed | 24 |
| Open | 0 |

Process caveat resolution (T-162-21): the auditor noted the human "approved"
confirmation for plan 162-05's checkpoint:human-verify gate is not
independently recoverable from static files. For the record: the approval was
given interactively during this execution session — the orchestrator presented
the 162-MEMORY-PROOF.md exhibit at the checkpoint and the user selected
"Approved" before 162-05 was completed. Recorded here as the durable trace.
