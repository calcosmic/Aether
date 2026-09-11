---
phase: 202-swarm-oracle-and-live-colony
verified: 2026-09-11T00:00:00Z
status: gaps_found
score: 3/5 must-haves verified
behavior_unverified: 0
overrides_applied: 0
gaps:
  - truth: "Watch shows real active workers, lineages, waves, workspaces, questions, confidence, contradictions, signals, findings, elapsed time, cost, and recovery state from replayable typed events (Roadmap Success Criterion 2; plan 202-03 must-have: 'the cockpit shows the whole colony rather than Swarm alone')."
    status: failed
    reason: "A recovery decision fired during an open build/continue episode is published under a synthetic 'recovery-phase-N' episode ID unrelated to the real running episode (cmd/recovery_orchestrator.go:215). resolveWatchMode's episode selector (cmd/watch_live.go:172-185, latestLiveEpisodeID) picks whichever episode owns the chronologically newest event with no concept of 'the episode that is still open', so the moment recovery fires mid-build, aether watch stops showing the real build/continue episode (with its active workers) and switches to the empty synthetic recovery episode -- which is itself never 'open' under the same start/end balance logic, so the dashboard falls into replay mode showing almost nothing while real work is running underneath. Verified directly against the code (not just SUMMARY.md): emitColonyLiveRecoveryChanged always constructs 'recovery-phase-%d' from ctx.Phase rather than reusing currentLiveBuildEpisode()/currentLiveContinueEpisode(), which the same file already exposes for exactly this purpose."
    artifacts:
      - path: "cmd/recovery_orchestrator.go"
        issue: "Line 215: emitColonyLiveRecoveryChanged(fmt.Sprintf(\"recovery-phase-%d\", ctx.Phase), events.EpisodeKindRecovery, outcome.Action.Type) ignores the real open episode ID."
      - path: "cmd/watch_live.go"
        issue: "latestLiveEpisodeID (lines 168-185) selects by newest single event, not by 'most recently started, still open', unlike watch_replay.go's mostRecentlyStartedLiveEpisode which already solves this for the replay path."
    missing:
      - "Route recovery's live event onto the real open episode (currentLiveBuildEpisode()/currentLiveContinueEpisode()), falling back to a synthetic ID only when no live episode is open at all."
      - "Make latestLiveEpisodeID select the most recently started, still-open episode rather than whichever episode owns the newest single event."
      - "A regression test that fires a recovery decision mid-build and asserts resolveWatchMode still returns the build episode with its workers, not the recovery episode."
  - truth: "A multi-round Oracle question visibly targets uncertainty, reports confidence and contradictions, detects diminishing returns, and produces a source-grounded final synthesis, watchable through aether watch (Roadmap Success Criterion 4; requirement CEC-05; plan 202-11 must-have: 'a running Oracle is watchable rather than pollable')."
    status: failed
    reason: "cmd/oracle_live.go never calls emitColonyLiveEpisodeStarted/emitColonyLiveEpisodeEnded for an Oracle run (zero call sites confirmed by grep), and none of the Oracle live topics (live.worker.started/progress, live.confidence.changed, live.contradiction.found, live.gap.targeted) increment the open/close balance foldColonyLiveEvents derives Open from (cmd/live_projection.go:352-367,436). The practical effect, reproduced directly: for every Oracle run that has ever existed, snapshot.Open is always false, so resolveWatchMode (cmd/watch_live.go:44) always returns watchModeReplay, never watchModeLive -- even while a round is actively iterating. This directly contradicts .aether/commands/oracle.yaml's own shipped promise ('A running Oracle round is visible in the live colony dashboard exactly like any other worker') and 202-11's own SUMMARY.md claim that 'a running Oracle round renders through the existing live dashboard' -- it never reaches that dashboard in the live branch a second terminal would actually see, because aether watch always classifies it as a stale replay instead. Confirmed the existing test (TestLiveDashboardShowsTheResearchRound) does not catch this: it discards the mode return value (`_, snapshot := resolveWatchMode(...)`) and drives renderColonyLiveDashboard directly with a hand-built snapshot, bypassing the mode resolution the bug is in."
    artifacts:
      - path: "cmd/oracle_live.go"
        issue: "No call ever opens or closes a colony-live episode boundary for an Oracle run; every Oracle event leaves the open/close balance untouched."
      - path: "cmd/watch_live.go"
        issue: "resolveWatchMode (lines 44-73) gates live-mode purely on snapshot.Open, which Oracle can never set to true."
    missing:
      - "Emit emitColonyLiveEpisodeStarted(oracleLiveEpisodeID(state), events.EpisodeKindOracle) when a round-based Oracle run genuinely begins, and emitColonyLiveEpisodeEnded(...) when it stops (completion, manual stop, or process exit) -- mirroring cmd/codex_continue.go's pattern -- or teach foldColonyLiveEvents to raise/lower openBalance on an unfinished live.worker.started/finished pair for episode kind oracle."
      - "A test that asserts mode == watchModeLive while an Oracle round is in flight (the existing test discards the mode value and does not check this)."
  - truth: "Every locked decision D-01 through D-12 traces to at least one SYN-202 row and at least one named implementation plan (plan 202-01 must-have)."
    status: partial
    reason: "The synthesis document and corpus registration are substantively present (35 SYN-202 entries in 202-CLASSIC-SYNTHESIS.md, 12 SYN-202 mechanism registry entries, corpus schema widened with V-202-EVENTS, all TestClassicContractPhase202Cases subtests pass) -- this truth is not failed on its own evidence. It is listed here only because the two failed truths above mean the implementation plans (202-03, 202-11) it traces to did not fully deliver what the synthesis committed to for D-01/D-02 (live cockpit) and the Oracle watchability decision; once those two gaps are closed this traceability truth needs no further work."
    artifacts: []
    missing:
      - "No independent action -- resolves automatically once the two failed truths above are fixed and their traces re-verified."
deferred: []
human_verification:
  - test: "Open aether watch during a live build/continue run, then trigger a recovery decision (e.g. force a worker failure) and observe whether the dashboard keeps showing the running build's active workers or switches to a near-empty recovery view."
    expected: "The build/continue episode and its active workers should remain visible, with recovery state layered on top -- not replaced by it."
    why_human: "Confirms the CR-02 gap's real-world user impact once the fix lands; the reproduction above is a direct code-level confirmation, but the visual/timing experience during a live run benefits from an eyes-on check."
  - test: "Start a long-running Oracle round (`aether oracle iterate` or equivalent) and, from a second terminal, run `aether watch` while it is in progress."
    expected: "The dashboard should show the Oracle round live (current question, confidence, contradictions) rather than falling back to a replay-labelled summary."
    why_human: "Confirms the CR-01 gap's real-world user impact once the fix lands."
  - test: "Look at the rendered `aether watch` live dashboard, Swarm's end-of-investigation comparison card, and Oracle's synthesis document for the Classic Feb-April colony character (caste emoji, ant glyphs, ceremony framing, Queen voice) named in D-04."
    expected: "The screens read as the Classic colony's own voice, not a generic status printout."
    why_human: "Visual/tonal character cannot be verified by grep or automated test; the code reads as houses-style-compliant (uses the shared caste identity/emoji helpers) but the actual on-screen feel needs a human read."
---

# Phase 202: Swarm, Oracle, and Live Colony Verification Report

**Phase Goal:** Restore substantive Swarm diagnosis, iterative Oracle research, and a real live colony cockpit driven by typed runtime events.
**Verified:** 2026-09-11
**Status:** gaps_found
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths (mapped to Roadmap Success Criteria)

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | A cited mechanism study reconstructs the exact Classic Swarm, Oracle, and live-display control loops and compares them with current issuance, research, event, and renderer paths before selecting the modern synthesis. | VERIFIED | `202-CLASSIC-SYNTHESIS.md` carries 35 `SYN-202-` entries and is signed; `cmd/testdata/classic-contract/v1/mechanisms.json` registers 12 `SYN-202-` entries; `schema.json` carries `V-202-EVENTS`; `TestClassicContractPhase202Cases` (all 3 subtests) pass. |
| 2 | Watch shows real active workers, lineages, waves, workspaces, questions, confidence, contradictions, signals, findings, elapsed time, cost, and recovery state from replayable typed events. | ✗ FAILED | Recovery events hijack the live view away from the real open build/continue episode (see gap below); worker-lineage plumbing (`ParentWorkerID`) exists and is read/rendered but has zero production call sites setting it (noted as a lower-severity, likely forward-looking gap — no current worker-spawns-worker producer exists in this codebase, so no lineage currently needs recording). |
| 3 | A stubborn defect runs four genuinely distinct Swarm lenses, compares hypotheses, ranks and checkpoints a repair, verifies or rolls back, preserves strike escalation, and stores one issuance-bound replay-safe episode. | VERIFIED | `TestFourSwarmLensesProduceDistinctEvidence`, `TestSwarmRepairRollsBackOnFailedVerification`, `TestSwarmRunProducesOneReplaySafeEpisode` all pass; `cmd/swarm_lens.go` (672 lines), `cmd/swarm_repair_checkpoint.go`, `cmd/swarm_episode.go` (695 lines) are substantive, no debt markers. Non-blocking caveat: the fix and verification waves (2 of 3 Swarm waves) still don't emit live events (WR-01 below), so `aether watch` goes dark for roughly the back two-thirds of a Swarm run's wall-clock time — `.aether/commands/swarm.yaml`'s own promise is scoped to wave 1 only, so this is an incompleteness, not a doc contradiction. |
| 4 | A multi-round Oracle question visibly targets uncertainty, reports confidence and contradictions, detects diminishing returns, and produces a source-grounded final synthesis. | ✗ FAILED | The research substance (confidence tracking, contradiction detection, diminishing-returns stop, recommendation-first synthesis) is real and tested (`TestOracleRoundsReachTheLiveStream`, `TestSynthesisLeadsWithTheRecommendation` pass). But the phase's own "watchable" promise for Oracle — CEC-05 and plan 202-11's explicit must-have — fails: a running Oracle round can never be classified `live` by `aether watch` (see gap below), contradicting both the shipped `.aether/commands/oracle.yaml` documentation and 202-11's own SUMMARY.md claim. |
| 5 | Useful partial Swarm/Oracle work, activity, learning, plan research, and local Dreams remain durable, discoverable, and accurately labelled rather than discarded or called verified. | VERIFIED | `cmd/partial_work_label.go` (403 lines) and `cmd/episode_index.go` (489 lines) substantive; `TestOneStandingVocabularyAcrossSubsystems`, `TestStatusHistoryAndWatchShareOneLineage` pass. |

**Score:** 3/5 roadmap success criteria verified; 0 present-but-behavior-unverified.

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `pkg/events/colony_live.go` | One versioned live-colony event vocabulary | ✓ VERIFIED | 158 lines, no debt markers, builds clean |
| `cmd/live_events.go` | Single emission boundary + per-lane helpers | ✓ VERIFIED | 339 lines; `TestOneLiveEventModelOnly`/`TestEveryLifecycleLaneEmitsLiveEvents` pass |
| `cmd/live_projection.go` | Pure replay reducer | ✓ VERIFIED | 447 lines; open/close balance logic is the root of both CR-01 and CR-02 gaps, but the reducer itself is deterministic and well-tested |
| `cmd/watch_live.go` | Go-side live/replay/idle mode resolution | ⚠️ HOLLOW (for Oracle and recovery cases) | 393 lines; correctly resolves for build/continue/plan/Swarm-investigation-wave, but `resolveWatchMode`/`latestLiveEpisodeID` misclassify Oracle runs and recovery-touched episodes per the gaps below |
| `cmd/watch_dashboard.go` | In-place live dashboard + ticker | ✓ VERIFIED | 337 lines; `TestLiveDashboardShowsCurrentWaveInDepth` passes; ticker doesn't surface check name/gap text (WR-02, non-blocking) |
| `cmd/swarm_lens.go` | Four-lens hypothesis comparison | ✓ VERIFIED | 672 lines; `TestFourSwarmLensesProduceDistinctEvidence` passes |
| `cmd/swarm_repair_checkpoint.go` | Checkpoint/verify/rollback adapter | ✓ VERIFIED | 135 lines; `TestSwarmRepairRollsBackOnFailedVerification` passes |
| `cmd/swarm_episode.go` | One durable replay-safe episode | ✓ VERIFIED | 695 lines; `TestSwarmRunProducesOneReplaySafeEpisode` passes |
| `cmd/oracle_preset.go` | Shared Fast/Balanced/Deep/Exhaustive vocabulary | ✓ VERIFIED | 111 lines; `TestOraclePresetLabelsMatchPlanningVocabulary` passes |
| `cmd/oracle_live.go` | Oracle live-event emission | ⚠️ ORPHANED (for watch's live-mode purposes) | 141 lines, well-tested for event emission itself, but never opens/closes an episode boundary, so its events never actually surface `aether watch`'s live mode (CR-01) |
| `cmd/oracle_synthesis.go` | Recommendation-first synthesis | ✓ VERIFIED | 238 lines; `TestSynthesisLeadsWithTheRecommendation` passes |
| `cmd/watch_replay.go` | Replay-backed idle summary | ✓ VERIFIED | 242 lines; `TestWatchReplaysTheMostRecentEpisode` passes |
| `cmd/partial_work_label.go` | Shared standing vocabulary | ✓ VERIFIED | 403 lines; `TestOneStandingVocabularyAcrossSubsystems` passes |
| `cmd/episode_index.go` | Shared status/history/watch lineage | ✓ VERIFIED | 489 lines; `TestStatusHistoryAndWatchShareOneLineage` passes |
| `cmd/classic_contract_202_test.go` | Phase 202 corpus case validation | ✓ VERIFIED | 201 lines; all subtests pass |
| `cmd/claudemd_live_colony_test.go` | Shipped-claim-names-a-test guard | ✓ VERIFIED | 148 lines; `TestEveryLiveColonyClaimInCLAUDEMDNamesALiveTest` passes |

### Key Link Verification

| From | To | Via | Status | Details |
|------|-----|-----|--------|---------|
| `cmd/swarm_cmd.go` | `cmd/live_events.go` | investigation wave emits through the boundary | ✓ WIRED | Confirmed; but 4 call sites in `swarm_cmd.go`/`swarm_lens.go` bypass the documented per-lane-helper contract and hand-construct `events.ColonyLivePayload` literals with a bare `"swarm"` string instead of `events.EpisodeKindSwarm` (WR-04, non-blocking today, latent drift hazard) |
| `cmd/watch_live.go` | `cmd/live_projection.go` | watch renders only what replay produced | ✓ WIRED | Confirmed structurally sound, but the *episode selection* (`latestLiveEpisodeID`) and *open-balance derivation* feeding this link are exactly where CR-01/CR-02 live |
| `cmd/codex_build.go`/`codex_continue.go`/`codex_plan.go` | `cmd/live_events.go` | build/continue/plan emit through the boundary | ✓ WIRED | `TestEveryLifecycleLaneEmitsLiveEvents` passes for all lanes including these three |
| `cmd/recovery_orchestrator.go` | `cmd/live_events.go` | recovery emits through the boundary | ⚠️ WIRED BUT MISROUTED | Recovery does emit through the one boundary (satisfies the letter of "every lane emits through one boundary"), but under a synthetic episode ID that hijacks the watch view (CR-02) |
| `cmd/oracle_loop.go` | `cmd/oracle_live.go` | each state mutation emits its matching live event | ⚠️ WIRED BUT INCOMPLETE | Events emit correctly (`TestOracleRoundsReachTheLiveStream` passes) but no episode-boundary event is ever emitted, so the events never make the run classify as live (CR-01) |
| `cmd/status.go`/`cmd/history.go` | `cmd/episode_index.go` | shared lineage | ✓ WIRED | `TestStatusHistoryAndWatchShareOneLineage` passes |
| `pkg/codex/dispatch.go` (`ParentWorkerID`) | `cmd/live_events.go`/`cmd/watch_dashboard.go` | lineage read/render path | ⚠️ ORPHANED | Field is declared, read, and rendered, but zero production call sites ever set it (WR-03) — no current worker-spawns-worker scenario exists in this codebase to exercise it, so this reads as forward-looking plumbing rather than a broken promise, but is worth a doc note per the reviewer's own suggested fix |

### Requirements Coverage

| Requirement | Source Plan(s) | Description | Status | Evidence |
|-------------|----------------|--------------|--------|----------|
| SYNTH-04 | 202-01 | Swarm/Oracle/live mechanism study | ✓ SATISFIED | `202-CLASSIC-SYNTHESIS.md`, corpus registration, tests pass |
| CEC-05 | 202-02, 202-03, 202-06, 202-11, 202-15 | Typed live events across planning/build/Swarm/Oracle/recovery/verification | ⚠️ PARTIALLY SATISFIED | Events are emitted on every lane (`TestEveryLifecycleLaneEmitsLiveEvents` passes for all six), satisfying the letter of "expose typed live events... rather than simulated activity" — but the deeper deliverable, a live cockpit that actually shows Oracle and recovery activity live rather than as a stale replay, fails per CR-01/CR-02 |
| LIVE-01 | 202-02, 202-04 | One typed event bridge | ✓ SATISFIED | `TestOneLiveEventModelOnly` passes; NDJSON stream deleted |
| LIVE-02 | 202-06, 202-09, 202-14 | Real live cockpit | ⚠️ PARTIALLY SATISFIED | Dashboard, ticker, replay-backed idle branch all work; recovery-state hijacking (CR-02) breaks the "shows active workers... AND recovery state" promise |
| LIVE-03 | 202-05 | Substantive Swarm diagnosis | ✓ SATISFIED | `TestFourSwarmLensesProduceDistinctEvidence` passes |
| LIVE-04 | 202-07 | Safe Swarm repair | ✓ SATISFIED | `TestSwarmRepairRollsBackOnFailedVerification` passes |
| LIVE-05 | 202-10, 202-13, 202-14 | Durable Swarm outcome | ✓ SATISFIED | `TestSwarmRunProducesOneReplaySafeEpisode` passes |
| LIVE-06 | 202-08, 202-11 | Iterative Oracle | ✓ SATISFIED (research mechanics) | Depth presets, clarify-once behavior confirmed; watchability gap tracked separately under CEC-05/LIVE-02 |
| LIVE-07 | 202-12, 202-13, 202-14 | Durable Oracle synthesis | ✓ SATISFIED | `TestSynthesisLeadsWithTheRecommendation` passes |

No orphaned requirements — every ID in the phase's requirement list (`SYNTH-04, CEC-05, LIVE-01..07`) is claimed by at least one plan's frontmatter and cross-referenced in REQUIREMENTS.md's Phase 202 row.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| `cmd/recovery_orchestrator.go` | 215 | Synthetic episode ID hijacks live view | 🛑 Blocker | CR-02 above |
| `cmd/oracle_live.go` | whole file | Missing episode boundary emission | 🛑 Blocker | CR-01 above |
| `cmd/swarm_cmd.go` | 452, 468 | Fix/verification waves never emit live events | ⚠️ Warning | `aether watch` goes dark for ~2/3 of a Swarm run; scoped-correctly per swarm.yaml, so not a doc contradiction |
| `cmd/live_projection.go` (switch in `foldColonyLiveEvents`) | 352-417 | Check name / gap text discarded, never reach the ticker | ⚠️ Warning | Ticker prints generic sentences instead of naming the check/gap |
| `pkg/codex/dispatch.go` | 49-57 | `ParentWorkerID` never set by any producer | ⚠️ Warning | Lineage rendering path is unreachable; likely forward-looking, no current producer scenario exists |
| `cmd/swarm_cmd.go`, `cmd/swarm_lens.go` | 328-333, 345-350, 1534-1545, 1619-1630, 642-652, 661-670 | 6 call sites hand-construct `events.ColonyLivePayload` and bypass the documented per-lane-helper contract, using bare `"swarm"` string instead of the `EpisodeKindSwarm` constant | ⚠️ Warning | Harmless today (values coincide) but a latent drift hazard; also untested by the boundary guard |

No `TODO`/`FIXME`/`XXX`/`TBD` debt markers found in any Phase 202 artifact file.

### Build/Test Verification

- `go build ./...` — clean
- `go vet ./...` — clean
- All 14 named must-have tests across the 15 plans — **pass** (`TestClassicContractPhase202Cases`, `TestEveryLiveColonyClaimInCLAUDEMDNamesALiveTest`, `TestOneLiveEventModelOnly`, `TestEveryLifecycleLaneEmitsLiveEvents`, `TestFourSwarmLensesProduceDistinctEvidence`, `TestSwarmRepairRollsBackOnFailedVerification`, `TestSwarmRunProducesOneReplaySafeEpisode`, `TestOraclePresetLabelsMatchPlanningVocabulary`, `TestOracleRoundsReachTheLiveStream`, `TestSynthesisLeadsWithTheRecommendation`, `TestOneStandingVocabularyAcrossSubsystems`, `TestStatusHistoryAndWatchShareOneLineage`, `TestWatchReplaysTheMostRecentEpisode`, `TestLiveDashboardShowsCurrentWaveInDepth`)
- `TestCurrentVocabulary199` and `TestPhase199GateReceipt` — **fail**, confirmed pre-existing at the pre-phase base commit (unrelated legacy vocabulary bookkeeping and a stale ownership-fingerprint receipt), not caused by this phase's code and not regressions.
- Note: full unscoped `go test ./cmd` was not run in full (known ~20-min machine ceiling per prior project notes); scoped `-run` filters covering every phase 202 named test plus the two known pre-existing failures were used instead.

## Gaps Summary

The phase's individual mechanisms — the mechanism study, the typed event model, the four-lens Swarm diagnosis, the checkpointed repair, the durable episode, the Oracle research loop, the recommendation-first synthesis, and the partial-work vocabulary — are all genuinely built, substantive, and covered by passing named tests. Where this phase falls short is exactly the seam the project's own Definition of Done exists to catch: the wiring between two already-working pieces.

Two blocker-level wiring gaps, both verified directly against the running code (not inferred from SUMMARY.md):

1. **Recovery hijacks the live view.** The moment a recovery decision fires during an active build or continue run, `aether watch` stops showing the real running episode and its active workers, and switches to an empty synthetic "recovery" episode instead — precisely the moment an owner watching the cockpit would most want visibility.
2. **Oracle can never be "live."** No Oracle run, no matter how actively it is iterating, can ever satisfy `aether watch`'s live-mode classification — it always renders as a stale replay. This directly contradicts the phase's own shipped documentation and one plan's own SUMMARY.md claim.

Both are narrow, well-understood fixes (reuse the existing `currentLiveBuildEpisode()`/`currentLiveContinueEpisode()` carriers for recovery; wrap Oracle's run boundary with the same episode-started/ended calls build/continue already use) rather than a redesign. Four additional warnings (Swarm's fix/verify waves staying dark, discarded check/gap ticker text, unset worker lineage, and Swarm's six hand-built payload literals) are lower-severity and do not block the phase goal on their own, but are worth tracking.
