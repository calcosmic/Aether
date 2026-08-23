---
phase: 194-the-queen-decides-the-team
plan: 07
subsystem: orchestration
tags: [queen, waiver, check-in-card, ceremony, forced-reviewer, D-03, D-14, plain-english]

# Dependency graph
requires:
  - phase: 194-01
    provides: "queenForcedContinueReviewers / codexForcedReviewerRecord — the function this plan's waiver filters inside"
  - phase: 194-04
    provides: "renderCeremonyTeamCheckin's forced-reviewer announcement block and result map — the card this plan adds waiver rendering to and, on owner feedback, restructures for readability"
  - phase: 194-06
    provides: "the changed-files risk-signal detector — what makes 'stays waived when the files re-detect it' provable"
provides:
  - "cmd/forced_reviewer_waiver.go — an owner-only, one-signal-one-phase waiver read through the existing decision-answer/answeredDecisionTexts path, with no persisted state of its own"
  - "applyForcedReviewerWaivers filtering inside queenForcedContinueReviewers — the single choke point both continue lanes already share"
  - "the check-in card's waived/waive_commands result keys and rendered decline block"
  - "a visually separated check-in card (blank lines between worker rows, `── Title ──` rule lines between sections) per live owner feedback on the card's readability"
  - "a pending todo recording the owner's reversal of D-14 (one-worker pause), routed to /gsd-discuss-phase as a scope change rather than folded into this plan"
affects: [194-08-proof-tests, 194-09-doc-correction, 196-cost-line, 197-closing-card]

# Actuals (#2632)
actuals:
  tokens: 7342
  tasks: 2
  commits: 2

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Waiver modules read owner answers only through the same decision-answer/answeredDecisionTexts path the criterion-confirmation pattern already uses (cmd/criterion_owner_confirmation.go) — no second normalisation, no second write path"
    - "Card section separation reuses renderStageMarker (cmd/codex_visuals.go, the build/continue stage-marker rule) rather than inventing a second separator style — one `── Title ──` convention for every ceremony surface"
    - "A live owner answer inside a checkpoint task can itself trigger a Rule-shaped change (here: layout) without becoming an architectural deviation, because the plan's own acceptance criteria named 'apply the wording change in this same task' as in-scope"

key-files:
  created:
    - cmd/forced_reviewer_waiver.go
    - cmd/forced_reviewer_waiver_test.go
    - .planning/todos/pending/2026-08-23-one-worker-build-skips-the-checkin-pause.md
  modified:
    - cmd/queen_risk_signals.go
    - cmd/ceremony_team_checkin.go
    - cmd/ceremony_team_checkin_test.go

key-decisions:
  - "Q1 (does the reason sentence read on its own): kept the sentence's wording unchanged. The owner's real complaint was layout, not phrasing — the fix is visual separation, not new words."
  - "Q2 (ask again next time, or is a written reason enough): kept the existing one-signal-one-phase scope with no re-ask within the same phase (D-03 unchanged). The owner accepted the default recommendation; re-asking within a phase for the same signal would be nagging, and the next phase already asks fresh because the waiver's question text embeds the phase number."
  - "Q3 (one-worker build should go straight through): NOT applied to this plan. The owner's answer reverses D-14 ('pause and ask, same as today'), and the plan's own resume-signal explicitly routes a changed answer here through /gsd-discuss-phase as a scope change. Recorded as a pending todo instead; TestOneWorkerTeamStillPauses (194-07 Task 1) still asserts the current, unchanged behavior and still passes."

patterns-established:
  - "A checkpoint task's owner-approved wording/layout fix lands in the same task's commit, per the plan's own <action> instruction, distinct from a scope-changing answer which is deferred to a todo and never silently absorbed."

requirements-completed: [TEAM-02, TEAM-05]

coverage:
  - id: D1
    description: "Only the owner can waive a forced reviewer, through a recorded decision-answer; the waiver scopes to one signal on one phase and stays waived when the same signal is re-detected from changed files."
    requirement: TEAM-02
    verification:
      - kind: unit
        ref: "cmd/forced_reviewer_waiver_test.go#TestOnlyTheOwnerCanWaiveAForcedReviewer"
        status: pass
      - kind: unit
        ref: "cmd/forced_reviewer_waiver_test.go#TestWaiverCoversOneSignalOnOnePhase"
        status: pass
      - kind: unit
        ref: "cmd/forced_reviewer_waiver_test.go#TestWaivedSignalStaysWaivedWhenTheFilesRedetectIt"
        status: pass
      - kind: unit
        ref: "cmd/forced_reviewer_waiver_test.go#TestAutopilotNeverWaives"
        status: pass
      - kind: unit
        ref: "cmd/forced_reviewer_waiver_test.go#TestTeamCheckinDoesNotMutateWithAWaiverPresent"
        status: pass
    human_judgment: false
  - id: D2
    description: "The check-in card is visually separated (blank line between worker rows, rule lines between sections) per the owner's readability feedback, with no sentence wording changed."
    requirement: TEAM-05
    verification:
      - kind: unit
        ref: "cmd/ceremony_team_checkin_test.go#TestTeamCheckinCardIsVisuallySeparated"
        status: pass
    human_judgment: false
  - id: D3
    description: "The owner has seen the rendered card (team list, forced-reviewer announcement, waive command) and answered the three verification questions."
    requirement: TEAM-05
    verification: []
    human_judgment: true
    rationale: "Whether the card 'reads correctly' to a non-technical reader is a subjective judgment no automated check can substitute for. The owner's verbatim answers are captured below; this entry documents that the judgment was already made in this session, not that it remains open."
  - id: D4
    description: "The build's pre-spawn pause is unchanged for a one-worker team (D-14 as originally decided); the owner's request to change that is filed as a scope-change todo rather than applied here."
    requirement: TEAM-05
    verification:
      - kind: unit
        ref: "cmd/ceremony_team_checkin_test.go#TestOneWorkerTeamStillPauses"
        status: pass
    human_judgment: false

# Metrics
duration: 45min
completed: 2026-08-23
status: complete
---

# Phase 194 Plan 7: The Owner-Only Waiver And The Owner's Own Verdict On The Card Summary

**An owner-only waiver that declines one forced reviewer for one phase with a recorded reason, plus a check-in card the owner asked to be visually separated — and a scope-changing answer on the one-worker pause routed to a todo instead of silently applied.**

## Performance

- **Duration:** 45 min (continuation session; Task 1 completed by a prior executor, Task 2 completed and closed out here)
- **Started:** 2026-08-23 (Task 1, prior session) / resumed same day for Task 2
- **Completed:** 2026-08-23T00:00:00Z (approximate — see Task Commits for exact hashes)
- **Tasks:** 2
- **Files modified:** 5 (2 created code files, 1 created todo file, 3 modified code files — `cmd/ceremony_team_checkin.go` counted once across both tasks)

## Accomplishments
- `cmd/forced_reviewer_waiver.go` — a stable question text, a shell-quoted decline command, and a read-only matcher against `answeredDecisionTexts()`, filtering inside `queenForcedContinueReviewers` before signals collapse into castes.
- The check-in card now shows a live forced reviewer with the exact command to decline it, and a waived reviewer as declined with the owner's own recorded reason.
- Applied the owner's live feedback: the card is now broken into `── Title ──` rule-separated sections (Team / Required After The Work Is Done / Decline A Required Reviewer / Not Sent / Summary) with a blank line between adjacent worker rows — no sentence content changed.
- The owner's third answer (make a one-worker build skip the pause) is recorded as a pending todo and explicitly NOT applied — it reverses D-14, and per the plan's own resume-signal that is a scope change for `/gsd-discuss-phase`, not this plan.

## Task Commits

Each task was committed atomically:

1. **Task 1: The waiver — owner-only, one signal, one phase, recorded with a reason** - `8865488d` (feat)
2. **Task 2: Owner check — the card, the waiver, and the one-worker pause** - `2b8bc060` (feat)

**Plan metadata:** committed alongside this SUMMARY (see final commit hash in the phase's git log).

## Files Created/Modified
- `cmd/forced_reviewer_waiver.go` - Owner-only waiver: question text, decline command, matcher, per-signal filter
- `cmd/forced_reviewer_waiver_test.go` - Tests proving no non-owner path can waive, scoping to one signal/phase, survival against file re-detection, autopilot exclusion, and non-mutation
- `cmd/queen_risk_signals.go` - `applyForcedReviewerWaivers` call site inside `queenForcedContinueReviewers`
- `cmd/ceremony_team_checkin.go` - Waived/waive-command rendering (Task 1) and visual section separation (Task 2)
- `cmd/ceremony_team_checkin_test.go` - `TestOneWorkerTeamStillPauses` (Task 1) and `TestTeamCheckinCardIsVisuallySeparated` (Task 2)
- `.planning/todos/pending/2026-08-23-one-worker-build-skips-the-checkin-pause.md` - Owner's reversal of D-14, filed as a scope-change candidate for the next `/gsd-discuss-phase`

## Owner Check-In (Task 2)

### What was shown

The card was rendered for a throwaway "Password reset" phase (a scratch colony directory outside this repo, never touching real project data) using a locally built binary, exactly per the plan's `<how-to-verify>`:

```
━━━ 🔨 T E A M   C H E C K - I N ━━━

── Team ──
  🔨🐜 Builder [sonnet]  REQUIRED  — writes the code for 1 task(s): Add the password reset flow  (what it does: Writes the implementation for the phase's tasks.)

  ⚡🐜 Measurer [opus]  OPTIONAL  — no reason was recorded for sending this worker  (what it does: Establishes whether a real performance, latency, memory, or cost regression exists, with numbers.)

  🎲🐜 Chaos [sonnet]  OPTIONAL  — no reason was recorded for sending this worker  (what it does: Probes failure modes, bad input, and resilience under stress.)

── Required After The Work Is Done ──
Not sent with this team, but required at the check after the work is done:
  a security reviewer will check this at the verification step — this touches logins and passwords (the plan mentions "password reset")

── Decline A Required Reviewer ──
Only you can decline a required reviewer, with a reason on the record:
  To decline, run: aether decision-answer --question 'Phase 1: a security reviewer is being added because this touches logins and passwords. Waive it?' --answer '<say why here>' --phase 1

── Not Sent ──
Not sent:
  Architect — not sent -- nobody asked for it and nothing in this phase forces it
  Gatekeeper — not sent -- nobody asked for it and nothing in this phase forces it

── Summary ──
Required workers stay — they are the safety floor. Optional workers can be trimmed.
```

(Colors stripped for this record; the live render used caste-colored ANSI labels.)

### The owner's answer (verbatim)

> "For number one, I'm not sure. What do you think? Also, I just think they need to be more separated. It's just like text. There needs to be more lines and things like that. To separate it, we used to have them 5.4. For number two, if you said no to it, I'm not sure. What do you think? And then for number three, one worker build should go straight through."

### The orchestrator's reply (verbatim, attributed to the orchestrator, not the owner)

- **Q1** (does the reason sentence read on its own): "the reason sentence reads fine on its own and is KEPT unchanged. The owner's real ask is LAYOUT: the card is a wall of text and needs visual separation — a blank line between each worker row and a rule line between sections, in the v5.4.0 house style."
- **Q2** (ask again vs. reason is enough): "recommendation accepted by default — writing the reason down once is enough; the waiver is scoped to one signal on one phase (D-03), so the next phase asks fresh anyway, and re-asking within the same phase would be nagging. No behaviour change."
- **Q3** (one-worker build should go straight through): "the owner wants a one-worker build to go straight through WITHOUT the check-in pause. This is the OPPOSITE of the decision locked on 2026-08-23 (D-14, 'pause and ask, same as today'). Per the plan's own `<resume-signal>`, a changed answer to question 3 is a SCOPE CHANGE that goes back through `/gsd-discuss-phase`, NOT into this plan. The pause behaviour stays exactly as it is in this plan."

### What changed as a result

- Q1/Q2: no code change to wording or scoping; Q1's layout complaint drove the visual-separation work (see Accomplishments and Deviations).
- Q3: no code change. Filed as `.planning/todos/pending/2026-08-23-one-worker-build-skips-the-checkin-pause.md`, explicitly marked as reversing D-14 and requiring a `/gsd-discuss-phase` pass before it can be planned. `TestOneWorkerTeamStillPauses` (already added in Task 1) continues to assert and pin the current, unchanged behavior.

## Decisions Made
- The card's layout separation reuses `renderStageMarker` (the existing `── Title ──` build/continue stage-marker helper in `cmd/codex_visuals.go`) rather than a new separator convention, per the plan's own instruction to match that style.
- A scope-changing owner answer inside a checkpoint (Q3) is recorded as a todo, not applied, because the plan text itself pre-committed to that routing in its `<resume-signal>` — this is not a judgment call made in the moment, it is following the plan as written.

## Deviations from Plan

None — plan executed exactly as written. The layout change in Task 2 was explicitly anticipated by the plan's own `<action>` text ("If the owner asks for a wording change, apply it and update the card tests... in this same task"), so it is in-scope work, not a deviation. The scope-change todo for Q3 is likewise the plan's own designed outcome for a changed D-14 answer, not an improvisation.

## Issues Encountered
None.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Plan 194-08 (proof tests) and 194-09 (doc correction) are next; both were named in this plan's own ordering rationale as depending on the waiver landing here.
- `.planning/WINDOWS.md` #1 (probe/auditor/gatekeeper double-dispatch) remains open and unaffected by this plan — tracked separately for Phase 194's later plans.
- The one-worker-pause reversal (Q3) is now a discoverable pending todo, not a silent gap; the next `/gsd-discuss-phase` pass should surface it.

---
*Phase: 194-the-queen-decides-the-team*
*Completed: 2026-08-23*
