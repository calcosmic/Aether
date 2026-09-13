---
phase: 203-biological-runtime
plan: "13"
subsystem: pheromones
tags: [pheromone-outcome, strength-tuning, quarantine, pin, bio-08, cec-07, go]

# Dependency graph
requires:
  - phase: 203-biological-runtime
    provides: "203-11's append-only pheromones-history.json (appendInfluenceHistory/readInfluenceHistory) and its closed action vocabulary; 203-12's recruitmentCreditRecord (helpful/neutral/harmful/pending) and recordRecruitmentCredit"
provides:
  - "tuneNoteStrengthFromOutcomes (cmd/pheromone_outcome.go): the one function that moves a note's strength, driven exclusively by recorded credit records, never delivery or a passing phase"
  - "Automatic quarantine at noteHarmfulQuarantineThreshold harmful credit records, one-directional (never auto-cleared)"
  - "colony.PheromoneSignal.Pinned, and the owner-only pinNote/unpinNote actions, exempting a note from all automatic tuning"
  - "pheromoneActionWeakened: a real, manual, owner-facing --weaken action (symmetric to --reinforce) and the same action name the automatic pass uses for a strength decrease"
  - "pheromoneStrengthOrigin: reads whether a note's current strength was last set by the owner or learned, derived from the history itself"
  - "runPheromoneOutcomeTuning (cmd/phase_end_signals.go): the one caller of tuneNoteStrengthFromOutcomes, wired into both continue lanes (cmd/codex_continue.go, cmd/codex_continue_finalize.go), never a build path"
affects: ["any future plan reading/writing pheromone strength, or extending the closed influence-action vocabulary"]

# Actuals (#2632)
actuals:
  tokens: 15744
  tasks: 3
  commits: 1

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "A learning-actor strength change is recorded through the SAME closed-vocabulary appendInfluenceHistory 203-11 built, under two existing-shaped-but-new action names (reinforced for up, a new weakened for down) rather than inventing a parallel history mechanism -- direction is named by the action, magnitude by the entry's own before/after text, which legitimately differs between a manual full snap (weakenNote/reinforceNote) and an automatic partial step (tuneNoteStrengthFromOutcomes)."
    - "A dedicated, plan-owned idempotency ledger (pheromone-outcome-tuning-state.json, recording consumed credit record IDs) rather than mutating credit/records.json (203-12's own single-writer file) or pheromones-history.json (203-11's own single-writer file) -- replay-safety is this plan's own bookkeeping, layered on top of two files it never writes to directly except through their existing sole writers."
    - "Reading credit records through a plan-owned wrapper (pheromoneOutcomeReadCreditRecords) instead of the existing recruitmentCreditAll, because the existing function deliberately collapses 'file missing' and 'file corrupted' to the same empty result -- correct for its own callers, but this plan's own acceptance criteria require the two to be told apart (Ran=false + Error only for genuine corruption)."

key-files:
  created:
    - cmd/pheromone_outcome.go
    - cmd/pheromone_outcome_test.go
  modified:
    - cmd/pheromone_influence.go
    - cmd/pheromone_mgmt.go
    - cmd/pheromone_resolver_test.go
    - cmd/phase_end_signals.go
    - cmd/codex_continue.go
    - cmd/codex_continue_finalize.go
    - pkg/colony/pheromones.go

key-decisions:
  - "Extended 203-11's closed 8-action vocabulary to 11 (adding weakened/pinned/unpinned) rather than reusing an existing action or bypassing the closed vocabulary, because TestEveryDeclaredActionIsReachable (203-11's own structural guard) requires every declared action to have a real, owner-facing CLI surface -- an automatic-only action would have failed that guard, and the plan's own Task 2 text explicitly asks for new declared pin/unpin actions with real flags, confirming this is the intended shape."
  - "weakenNote lowers a note straight to the floor (0.0), the exact symmetric opposite of the existing reinforceNote's snap-to-ceiling -- a real, useful owner capability in its own right, and the action name the automatic pass's SMALLER partial-step decreases (neutral/harmful outcomes) also use, with the actual magnitude distinguished in the history entry's own before/after text rather than by a second action name."
  - "Quarantine crossing is computed from the FULL harmful-record history for a note (every earned harmful credit record, processed or not, across every past run), not merely today's newly-processed slice -- so the threshold reflects the note's whole harmful track record and is unaffected by how the processing happened to be batched across multiple check-lane runs."
  - "A pinned or revoked note's credit record is still marked consumed (idempotency-wise) even though it changes nothing -- avoids the pass re-evaluating (and re-appending an identical skip entry for) the same already-decided record on every future check, while still recording that the pass 'considered and declined' via a history entry naming the reason, exactly as the plan's own Task 2 action text specifies."
  - "The three tasks landed in one commit: the core tuning loop already depends on the pin/revoke skip logic and the extended action vocabulary from the start, so splitting by task would require fabricating a non-compiling intermediate state -- the identical precedent 203-05/203-11/203-12 already recorded in this same phase for the same class of interdependency."

requirements-completed: [BIO-08, CEC-07]

coverage:
  - id: D1
    description: "A note's strength moves up on a helpful credit record, down (a smaller amount) on neutral, down further on harmful, and is completely untouched by a delivered note plus a passing phase with no credit record at all. Every movement is clamped to a declared floor/ceiling with the clamp recorded, and a note with 3+ harmful credit records is automatically quarantined with a history entry naming the threshold. Running the pass twice over the same records changes nothing the second time."
    requirement: "CEC-07"
    verification:
      - kind: unit
        ref: "cmd/pheromone_outcome_test.go#TestNoteStrengthTuningHelpfulNeutralHarmfulMovements"
        status: pass
      - kind: unit
        ref: "cmd/pheromone_outcome_test.go#TestNoteStrengthTuningNoRecordsLeavesNoteUntouched"
        status: pass
      - kind: unit
        ref: "cmd/pheromone_outcome_test.go#TestNoteStrengthTuningClampRecordsClamp"
        status: pass
      - kind: unit
        ref: "cmd/pheromone_outcome_test.go#TestNoteStrengthTuningQuarantinesOnHarmfulThreshold"
        status: pass
      - kind: unit
        ref: "cmd/pheromone_outcome_test.go#TestNoteStrengthTuningIsIdempotent"
        status: pass
      - kind: unit
        ref: "cmd/pheromone_outcome_test.go#TestDeliveryWithoutCreditChangesNothing"
        status: pass
      - kind: other
        ref: "go test ./cmd -run '^TestNoteStrengthTuning' -count=1"
        status: pass
    human_judgment: false
  - id: D2
    description: "An owner-pinned note is skipped by tuning in both directions (the skip is recorded, not silently ignored); the runtime cannot pin, unpin, revoke, or appeal -- only the owner; a tuning failure (an unreadable credit store, or a panic) is recorded and never blocks, fails, or pauses the check that called it, proven on both continue lanes; and the owner can read, for any note, whether its current strength was last set by them or learned, derived from the history itself rather than a separate flag."
    requirement: "BIO-08"
    verification:
      - kind: unit
        ref: "cmd/pheromone_outcome_test.go#TestPinnedNoteIsNeverTuned"
        status: pass
      - kind: unit
        ref: "cmd/pheromone_outcome_test.go#TestTuningNeverBlocksAPhase"
        status: pass
      - kind: unit
        ref: "cmd/pheromone_outcome_test.go#TestStrengthOriginIsReadable"
        status: pass
      - kind: other
        ref: "go test ./cmd -run '^(TestPinnedNoteIsNeverTuned|TestTuningNeverBlocksAPhase|TestStrengthOriginIsReadable)$' -count=1"
        status: pass
    human_judgment: false
  - id: D3
    description: "Both continue lanes (the default fast lane and the wrapper-driven external lane) reach the tuning pass through one real caller; no function other than the tuning pass appends a strength change with actor kind learning (an AST guard, manually verified this session to fail by name against a synthetic violation, then restored); and no build-path source file can reach the tuning pass at all."
    requirement: "BIO-08"
    verification:
      - kind: unit
        ref: "cmd/pheromone_outcome_test.go#TestBothCheckLanesTuneNotes"
        status: pass
      - kind: unit
        ref: "cmd/pheromone_outcome_test.go#TestOnlyTheTuningPassLearnsStrength"
        status: pass
      - kind: unit
        ref: "cmd/pheromone_outcome_test.go#TestTuningIsNotOnTheBuildPath"
        status: pass
      - kind: other
        ref: "go test ./cmd -run '^(TestBothCheckLanesTuneNotes|TestOnlyTheTuningPassLearnsStrength|TestDeliveryWithoutCreditChangesNothing|TestTuningIsNotOnTheBuildPath)$' -count=1"
        status: pass
    human_judgment: false

# Metrics
duration: 70min
completed: 2026-09-13
status: complete
---

# Phase 203 Plan 13: Outcome-Weighted Pheromone Strength Tuning Summary

**A note's strength now rises or falls only on recorded evidence of what it did — never on delivery or a passing phase alone — with automatic quarantine after repeated harm, an owner-pinned note untouchable by any automatic path, and every movement traceable to the exact credit record that caused it.**

## Performance

- **Duration:** ~70 min
- **Completed:** 2026-09-13
- **Tasks:** 3 completed
- **Files modified:** 9 (2 created, 7 modified)

## The Three Preconditions — Discharged Before Any Code Moved a Strength

The plan's own gate required these proven in place BEFORE writing a single line that moves a number automatically. All three were re-verified this session, not assumed from a SUMMARY:

1. **203-11's append-only history actually exists and is enforced.** Read `cmd/pheromone_influence.go` directly: `appendInfluenceHistory` is the one exported mutation on `pheromones-history.json`, backed by `store.UpdateJSONAtomically`, and `TestInfluenceHistoryAppendOnly`'s own `structural_single_writer` subtest asserts by AST scan that exactly one function in the package writes that file. Ran `go test ./cmd -run '^TestInfluenceHistoryAppendOnly$' -count=1` this session — PASS.
2. **An owner-set note is untouchable — proven by breaking the guard first.** Before writing `tuneNoteStrengthFromOutcomes`'s pin-skip logic, I temporarily removed `pheromoneActionPinned`/`pheromoneActionUnpinned` from `pheromoneInfluenceActorAllowed`'s owner-only case and re-ran `TestPinnedNoteIsNeverTuned` — it failed by name (`"expected a runtime actor's pin attempt to be refused"`). Restored the check, re-ran — PASS. The same break/red/restore cycle was run against `TestOnlyTheTuningPassLearnsStrength` (added a probe function calling `appendInfluenceHistory` with actor kind learning outside the tuning pass — failed by name, naming the probe) and `TestTuningIsNotOnTheBuildPath` (added a call to `tuneNoteStrengthFromOutcomes` inside `cmd/codex_build.go` — failed by name, naming the file). All three restorations left `git status --short` clean.
3. **Every automatic movement is distinguishable from an owner's own setting, in the stored data.** Every `tuneNoteStrengthFromOutcomes` call to `appendInfluenceHistory` passes `pheromoneActorLearning` as the actor kind — the SAME field an owner's manual `reinforceNote`/`weakenNote`/`pinNote` call records as `pheromoneActorOwner`. `pheromoneStrengthOrigin` answers "which was it?" by reading the most recent strength-changing history entry's own `ActorKind`, not a separate flag that could drift out of sync with the truth. `TestStrengthOriginIsReadable` proves both directions on real data.

None of the three preconditions required a halt — all were genuinely in place, and the third's ordinary reader (`pheromoneStrengthOrigin`) was built as part of this plan's own Task 2, per the plan's action text.

## Accomplishments

- `cmd/pheromone_outcome.go` (new): `tuneNoteStrengthFromOutcomes` reads exclusively from `cmd/recruitment_credit.go`'s credit records (kind `note`, non-pending outcomes only) and moves a note's strength by one of three named constants (`noteStrengthHelpfulStep`/`-Neutral-`/`-Harmful-`), clamped to `[noteStrengthFloor, noteStrengthCeiling]` with the clamp recorded, and quarantines automatically at `noteHarmfulQuarantineThreshold` (3) accumulated harmful records. Every change is appended via `appendInfluenceHistory` with actor kind `learning` and the credit record's own identifier as the reason. A second pass over the same records is a no-op, tracked via a small plan-owned idempotency ledger (`pheromone-outcome-tuning-state.json`) that never touches either the credit store or the history file directly.
- `pinNote`/`unpinNote` (in `cmd/pheromone_influence.go`, owner-only, new `--pin`/`--unpin` flags on `pheromoneDisplayCmd`) let the owner exempt a note from tuning in either direction; the runtime cannot pin or unpin. `weakenNote` (new `--weaken` flag) is a real, manual, symmetric-to-reinforce owner action that also names the direction the automatic pass uses for a decrease.
- `colony.PheromoneSignal.Pinned` (new pointer-backed, `omitempty` field) — a legacy signal reads as not pinned.
- `pheromoneStrengthOrigin` reads, for any note, whether its current strength was last set by the owner or learned, derived from the most recent strength-changing history entry.
- `runPheromoneOutcomeTuning` (`cmd/phase_end_signals.go`) is the one caller of the tuning pass, wired into both continue lanes (`cmd/codex_continue.go`'s default lane, `cmd/codex_continue_finalize.go`'s wrapper-driven lane) immediately after hive promotion, recovering from a panic and never propagating a tuning error to the check that called it.

## Task Commits

Tasks 1-3 landed as one commit — the core tuning loop already depends on the pin/revoke skip logic and the closed action vocabulary extension from the start, so splitting by task would require fabricating a non-compiling intermediate state. Same precedent already recorded twice earlier in this phase (203-05, 203-11) for the identical class of interdependency.

1. **Tasks 1-3 (feat):** `3878dd4d` — `cmd/pheromone_outcome.go`, `cmd/pheromone_outcome_test.go`, `cmd/pheromone_influence.go`, `cmd/pheromone_mgmt.go`, `cmd/pheromone_resolver_test.go`, `cmd/phase_end_signals.go`, `cmd/codex_continue.go`, `cmd/codex_continue_finalize.go`, `pkg/colony/pheromones.go`

**Plan metadata:** this commit (docs: complete plan)

## Files Created/Modified

- `cmd/pheromone_outcome.go` — `tuneNoteStrengthFromOutcomes`, the six named step/bound constants, `pheromoneOutcomeReadCreditRecords`, the tuning idempotency ledger, `pheromoneStrengthOrigin`, `clampNoteStrength`
- `cmd/pheromone_outcome_test.go` — all Task 1/2/3 tests
- `cmd/pheromone_influence.go` — three new declared actions (`weakened`/`pinned`/`unpinned`), `weakenNote`/`pinNote`/`unpinNote`, owner-only enforcement extended
- `cmd/pheromone_mgmt.go` — `--weaken`/`--pin`/`--unpin` flags wired into `runPheromoneInfluenceFlags`
- `cmd/pheromone_resolver_test.go` — `TestNoUngovernedQuarantineClear`'s allowlist extended to name `tuneNoteStrengthFromOutcomes` as a function permitted to SET (never clear) quarantine
- `cmd/phase_end_signals.go` — `runPheromoneOutcomeTuning`, `attachPheromoneOutcomeTuningSummary`
- `cmd/codex_continue.go`, `cmd/codex_continue_finalize.go` — both continue lanes call the tuning pass and attach its summary
- `pkg/colony/pheromones.go` — `PheromoneSignal.Pinned`

## Decisions Made

See `key-decisions` in frontmatter for the full list. The most consequential: extending 203-11's closed action vocabulary (rather than reusing an existing action or bypassing it) was required by that plan's own `TestEveryDeclaredActionIsReachable` guard, which demands every declared action have a real owner-facing CLI surface — confirmed as the intended shape by Task 2's own action text explicitly asking for new pin/unpin actions with real flags.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing Critical] Edited four files outside the plan's declared `files_modified`**
- **Found during:** Task 1/2 (the plan's own action text)
- **Issue:** The plan's frontmatter `files_modified` lists only `cmd/pheromone_outcome.go`, `cmd/phase_end_signals.go`, `cmd/pheromone_outcome_test.go` — but Task 1's own action text requires appending through `appendInfluenceHistory` (whose closed action vocabulary lives in `cmd/pheromone_influence.go`), and Task 2's own action text explicitly requires "Add `Pinned *bool`... on `colony.PheromoneSignal`" (`pkg/colony/pheromones.go`), "add `pinNote` and `unpinNote` to `cmd/pheromone_influence.go`'s action set... and add matching flags to the pheromone management command" (`cmd/pheromone_mgmt.go`), and wiring the automatic quarantine set requires extending `cmd/pheromone_resolver_test.go`'s own `TestNoUngovernedQuarantineClear` allowlist so the new setter is a deliberately named, reasoned exception rather than an unnoticed structural-guard gap.
- **Fix:** Edited all four files as required by the plan's own acceptance criteria and must_haves, following the identical precedent 203-05, 203-11, and 203-12 already recorded in this same phase for the same class of frontmatter/action-text mismatch.
- **Files modified:** `cmd/pheromone_influence.go`, `pkg/colony/pheromones.go`, `cmd/pheromone_mgmt.go`, `cmd/pheromone_resolver_test.go`
- **Verification:** Full targeted regression sweep (`go test ./cmd -run 'Pheromone|Influence|Resolver|Quarantine|Credit|Agency|PhaseEnd|Hive|Consolidation|CodexContinue|Continue' -count=1`) passes except the pre-existing, documented-red `TestGoldenContinueVisualOutput` (unrelated visual-voice golden mismatch, listed in this plan's own project-specific warnings).
- **Committed in:** `3878dd4d`

**2. [Rule 2 - Missing Critical] Wired both continue lanes despite them not being in `files_modified`**
- **Found during:** Task 2 (must_haves key_link: "the end-of-check pass is the one caller")
- **Issue:** The plan's must_haves explicitly require `cmd/phase_end_signals.go` to be `tuneNoteStrengthFromOutcomes`'s one caller, and Task 2's own action text says "Call the tuning pass from `cmd/phase_end_signals.go`... on both check lanes" — this cannot be satisfied without also editing the two files that ARE those lanes (`cmd/codex_continue.go`, `cmd/codex_continue_finalize.go`), neither of which is owned by the parallel sibling plan 203-14 (`cmd/recruitment_subtree.go`, `cmd/live_projection.go`, `cmd/watch_live.go`, `cmd/status.go`, `cmd/codex_visuals.go`, `cmd/recruitment_subtree_test.go`).
- **Fix:** Added one call site (`runPheromoneOutcomeTuning()`) plus one summary-attach call in each lane, immediately after the existing `promotePhaseEndInstinctsToHive` call, mirroring that function's own non-blocking placement exactly.
- **Files modified:** `cmd/codex_continue.go`, `cmd/codex_continue_finalize.go`
- **Verification:** `TestBothCheckLanesTuneNotes` and `TestTuningNeverBlocksAPhase` drive both real lanes end-to-end and pass; `TestStrongInstinctReachesTheSharedStoreAtCheck` (203-11's own pre-existing two-lane hive-promotion test) still passes unchanged, confirming no interference with the existing consolidation/hive-promotion call sequence.
- **Committed in:** `3878dd4d`

---

**Total deviations:** 2 auto-fixed (both Rule 2 — missing critical functionality required by the plan's own must_haves/action text). **Impact:** No scope creep; both closed a genuine gap between the plan's frontmatter header and what its own action text required. No file owned by parallel sibling plan 203-14 was touched.

## Issues Encountered

- `store.SaveRawJSON` (and every `*.json` write path through `pkg/storage`'s `atomicWriteLocked`) validates JSON on write and refuses invalid content by design — so simulating "the credit store cannot be read" for `TestTuningNeverBlocksAPhase` required writing the corrupted fixture directly to the filesystem path with `os.WriteFile`, bypassing the store, rather than through any store API. This is a test-fixture detail only; no production code was affected.
- The full, unscoped `go test ./cmd -count=1` run was not re-run in full this session (it runs several minutes on this machine per this plan's own project-specific warnings); the targeted regression sweep named above covers every test category this plan's changes could plausibly affect, plus the phase's own named ratchets (`TestColonyStateWriteAllowlistOnlyShrinks` was not re-verified directly this session, but no new `pheromones.json`/`credit/records.json` write path bypasses `store.SaveJSON`/`UpdateJSONAtomically` — every write in `cmd/pheromone_outcome.go` goes through `pheromoneInfluenceSaveSignals`, `savePheromoneOutcomeState`, or `store.UpdateJSONAtomically` directly). `TestGoldenContinueVisualOutput` fails exactly as this plan's own known-red baseline documents (unrelated visual-voice regression, pre-existing).
- **BIO-08 and CEC-07 are NOT marked complete in `.planning/REQUIREMENTS.md`, despite `requirements.ready-ids` confirming both are ready** (every plan declaring either ID — 203-08, 203-11, 203-13 for BIO-08; 203-12, 203-13 for CEC-07 — now has a SUMMARY.md). `gsd-tools requirements.mark-complete BIO-08 CEC-07` returned `not_found` for both and made **zero writes** to REQUIREMENTS.md (confirmed via `git status --short` — the file is untouched). The tool's checkbox regex expects `- [ ] **REQ-ID**` with the bold closing immediately after the ID; this repo's actual format is `- [ ] **CEC-07 — Colony Effect:**` (a title inside the same bold span), which the regex does not match. This is a pre-existing format mismatch between this repo's REQUIREMENTS.md and gsd-tools' assumption, not something introduced by this plan, and I deliberately did not hand-edit the checkbox to avoid diverging from the tool's canonical write path. The owner or a follow-up plan should either update REQUIREMENTS.md's own convention or the tool's regex, then re-run `requirements mark-complete BIO-08 CEC-07`.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- Outcome-weighted tuning, quarantine, and pin/unpin are all real, tested, and wired into both continue lanes — ready for a future plan to expose `pheromoneStrengthOrigin` on the owner-facing pheromone display, or to extend `tuneNoteStrengthFromOutcomes`'s inputs to the other three declared contribution kinds (recruitment result, memory item, specialist contribution) beyond `note`, which this plan deliberately scoped to `note` contributions only (the only kind that names a pheromone signal's own strength).
- No blockers for sibling plan 203-14 (no file overlap occurred). BIO-08 and CEC-07 are functionally complete and ready per `requirements.ready-ids` (every declaring plan has a SUMMARY), but see "Issues Encountered" above — the REQUIREMENTS.md checkbox flip itself did not apply due to a pre-existing tool/format mismatch, and needs a follow-up run once that is resolved.

---
*Phase: 203-biological-runtime*
*Completed: 2026-09-13*

## Self-Check: PASSED

- FOUND: `cmd/pheromone_outcome.go` (created)
- FOUND: `cmd/pheromone_outcome_test.go` (created)
- FOUND: commit `3878dd4d` (feat(203-13): outcome-weighted pheromone strength tuning, pin/unpin, never-block wiring) in `git log --oneline`
- Re-ran plan-level task `<verify>` commands individually:
  - Task 1: `go test ./cmd -run '^TestNoteStrengthTuning' -count=1` — PASS
  - Task 2: `go test ./cmd -run '^(TestPinnedNoteIsNeverTuned|TestTuningNeverBlocksAPhase|TestStrengthOriginIsReadable)$' -count=1` — PASS
  - Task 3: `go test ./cmd -run '^(TestBothCheckLanesTuneNotes|TestOnlyTheTuningPassLearnsStrength|TestDeliveryWithoutCreditChangesNothing|TestTuningIsNotOnTheBuildPath)$' -count=1` — PASS
- Re-ran `go build ./...` and `go vet ./...` — clean
- Re-ran the targeted regression sweep: `go test ./cmd -run 'Pheromone|Influence|Resolver|Quarantine|Credit|Agency|PhaseEnd|Hive|Consolidation|CodexContinue|Continue' -count=1` — PASS except the documented-red `TestGoldenContinueVisualOutput`
- Re-ran `go test ./pkg/colony/... -count=1` — PASS
- Three guard tests (`TestPinnedNoteIsNeverTuned`'s owner-only enforcement, `TestOnlyTheTuningPassLearnsStrength`, `TestTuningIsNotOnTheBuildPath`) independently verified this session to fail-by-name against a deliberately introduced violation, then restored to the clean/passing state (`git status --short` empty after each restoration)
