---
phase: 196-see-what-it-cost
plan: 04
subsystem: infra
tags: [model-selection, cost, team-checkin, ceremony, go]

requires:
  - phase: 194-the-queen-decides-the-team
    provides: the pre-build team check-in card, its required/optional marking and forced-reviewer signals
  - phase: 195-coherent-jobs
    provides: the one-worker fast-path summary that replaces the card when nothing is left to decide
provides:
  - Three routine roles (documentation writer, knowledge-keeper, accessibility checker) pinned to the cheaper model in their own agent files
  - A written, plain-English reason for every one of the ten roles kept on the expensive model
  - Model and reason on every worker line of the pre-build team card, and on the one-worker fast-path summary
  - Model and reason as first-class fields on both renderers' result maps, so wrappers never scrape prose
affects: [196-07 closeout cost line, 196-06 aether spend, any wrapper narrating the team card]

actuals:
  tokens: 5335
  tasks: 2
  commits: 4

tech-stack:
  added: []
  patterns:
    - "Owner-facing justification tables live beside the display table they explain and are checked in both directions, so neither can go stale"
    - "A plain-English gloss accompanies every internal model name shown to the owner"

key-files:
  created:
    - cmd/caste_model_reason.go
  modified:
    - cmd/caste_model_test.go
    - cmd/codex_visuals.go
    - cmd/ceremony_team_checkin.go
    - cmd/ceremony_team_checkin_test.go
    - .claude/agents/ant/aether-chronicler.md
    - .claude/agents/ant/aether-keeper.md
    - .claude/agents/ant/aether-includer.md

key-decisions:
  - "The three routine roles are pinned in their own agent files (the source that actually routes), not only in the Go display table — the tests assert the resolved model, so a pin that does not route fails"
  - "The reason table is keyed only to roles on the expensive model and is checked in both directions: a missing reason fails by name, and a reason outliving its role's move to the cheaper model fails by name"
  - "Reasons are held to CLAUDE.md's owner-facing rule by test: no invented repo vocabulary, no role's own name, no file paths, and each must say what a cheaper model would get wrong"
  - "The card names the model in plain words ('the cheaper model' / 'kept on the more expensive model because…') because 'sonnet' and 'opus' mean nothing without a glossary"
  - "No code selects or overrides a model at dispatch time — automatic model routing stays declined (2026-07-28)"

patterns-established:
  - "Two-way table integrity: an explanatory table is tested for both missing entries and stale ones"
  - "Owner-facing text tested as text: forbidden-vocabulary and placeholder checks run against strings the owner will read"

requirements-completed: [COST-04]

coverage:
  - id: D1
    description: "The documentation writer, the knowledge-keeper and the accessibility checker run on the cheaper model, pinned in their own agent files, and no role anywhere falls back to whatever model happened to run last"
    requirement: "COST-04"
    verification:
      - kind: unit
        ref: "cmd/caste_model_test.go#TestRoutineBuilderIsSonnetNeverInherit"
        status: pass
      - kind: unit
        ref: "cmd/caste_model_test.go#TestCasteModelSlotMatchesAgentFrontmatter"
        status: pass
    human_judgment: false
  - id: D2
    description: "Every role kept on the expensive model carries a written reason; a missing reason fails by name and a stale one fails by name"
    requirement: "COST-04"
    verification:
      - kind: unit
        ref: "cmd/caste_model_test.go#TestOpusRequiresRecordedReason"
        status: pass
      - kind: manual_procedural
        ref: "removed the gatekeeper reason -> TestOpusRequiresRecordedReason failed naming gatekeeper; reverted aether-keeper.md to inherit -> TestRoutineBuilderIsSonnetNeverInherit failed naming keeper"
        status: pass
    human_judgment: false
  - id: D3
    description: "The pre-build team card names each worker's model and shows the recorded reason for an expensive one, carrying both as result fields rather than prose only"
    requirement: "COST-04"
    verification:
      - kind: unit
        ref: "cmd/ceremony_team_checkin_test.go#TestTeamCardNamesModelAndReasonForEveryWorker"
        status: pass
      - kind: unit
        ref: "cmd/ceremony_team_checkin_test.go#TestCheapModelWorkerNeedsNoReasonOnTheCard"
        status: pass
    human_judgment: false
  - id: D4
    description: "The one-worker fast-path summary, which never pauses, carries the same model and reason"
    requirement: "COST-04"
    verification:
      - kind: unit
        ref: "cmd/ceremony_team_checkin_test.go#TestFastPathSummaryCarriesModelAndReason"
        status: pass
    human_judgment: false
  - id: D5
    description: "A rendered team card reads as plain English to someone who has never opened a file in this repository"
    requirement: "COST-04"
    verification:
      - kind: manual_procedural
        ref: "aether ceremony team-checkin --workflow build --manifest-file <fixture> with AETHER_OUTPUT_MODE=visual"
        status: pass
    human_judgment: true
    rationale: "Whether a sentence reads plainly to the owner is a judgement no assertion settles. The automated forbidden-vocabulary and placeholder checks narrow it but cannot replace reading the card."

duration: 20min
completed: 2026-08-28
status: complete
---

# Phase 196 Plan 04: Model Choice Carries A Reason Summary

**Three routine roles pinned off `inherit` onto the cheaper model in the agent files that actually route, plus a written plain-English reason for all ten expensive roles, shown on the pre-build team card and the one-worker fast-path summary.**

## Performance

- **Duration:** 20 min
- **Started:** 2026-08-28T10:25:00Z
- **Completed:** 2026-08-28T10:45:00Z
- **Tasks:** 2 (both TDD, RED then GREEN)
- **Files modified:** 8 (1 created, 7 modified)

## Accomplishments

- The documentation writer (`chronicler`), knowledge-keeper (`keeper`) and accessibility checker (`includer`) now declare `model: sonnet` in `.claude/agents/ant/`, so their cost is predictable and attributable. No role anywhere is left on `inherit`, and nothing resolves to the session's leftover model.
- All ten roles on the expensive model carry a one-sentence written reason saying what they judge and what a cheaper model would get wrong. The table is checked in both directions, so it cannot go stale from either end.
- The pre-build team card shows each worker's model in plain words and the recorded reason beside an expensive one, before anything spawns. The one-worker fast-path summary — the surface that never pauses — carries the same two facts.
- Both renderers expose `model` / `model_reason` as result fields, so a wrapper narrating this reads structured values instead of scraping rendered text.
- Nothing added chooses a model. The display table's own comment still says it is display-only, `TestCasteModelSlotMatchesAgentFrontmatter` still enforces the two-way agreement with the agent files, and automatic model routing stays declined.

## Task Commits

1. **Task 1 RED: failing model-pinning and reason tests** — `acc95937` (test)
2. **Task 1 GREEN: pin routine roles, add the reason table** — `47b0af20` (feat)
3. **Task 2 RED: failing team card model-and-reason assertions** — `865f2075` (test)
4. **Task 2 GREEN: model and reason on the card and the fast path** — `4103d2ae` (feat)

## RED evidence

Task 1 RED, tests only (`go test ./cmd -run 'Test(RoutineBuilderIsSonnetNeverInherit|OpusRequiresRecordedReason|CasteModelSlotMatchesAgentFrontmatter)'`):

```
cmd/caste_model_test.go:278:13: undefined: casteModelReason
cmd/caste_model_test.go:310:21: undefined: casteModelReasons
FAIL	github.com/calcosmic/Aether/cmd [build failed]
```

With the reason table present but before the roles were pinned, the same test failed on assertions rather than on compilation:

```
--- FAIL: TestRoutineBuilderIsSonnetNeverInherit (0.00s)
    caste_model_test.go:196: chronicler (the documentation writer) resolves to "session", want sonnet — D-02 pins the routine roles to the cheaper model
    caste_model_test.go:199: chronicler (the documentation writer) declares model "inherit" in its own agent file, want sonnet — a pin that lives only in the display table does not route
    caste_model_test.go:196: keeper (the knowledge-keeper) resolves to "session", want sonnet — …
    caste_model_test.go:196: includer (the accessibility checker) resolves to "session", want sonnet — …
    caste_model_test.go:212: casteModelSlot["keeper"] is still "inherit" — no role may run on whatever model happened to run last (D-02)
```

Task 2 RED (`go test ./cmd -run 'Test(TeamCardNamesModelAndReason|CheapModelWorkerNeedsNoReason|FastPathSummaryCarriesModelAndReason)'`):

```
--- FAIL: TestTeamCardNamesModelAndReasonForEveryWorker (0.00s)
    ceremony_team_checkin_test.go:989: result carries no models field — a wrapper would have to scrape the card text
--- FAIL: TestFastPathSummaryCarriesModelAndReason/expensive_worker_shows_its_reason (0.00s)
    ceremony_team_checkin_test.go:1077: result model = "", want opus
    ceremony_team_checkin_test.go:1087: fast-path summary never names the model:
        ── One Worker, No Approval Needed ──
          🐛🐜 Tracker Trail-3
          Covers: 1 (Find why the export drops rows)
```

## Acceptance-criteria mutations (proof the tests bite)

Both required mutations were run and reverted:

```
--- MUTATION A: gatekeeper reason removed ---
--- FAIL: TestOpusRequiresRecordedReason
    caste_model_test.go:283: gatekeeper runs on the expensive model but records no reason why — D-02 requires a written one

--- MUTATION B: keeper agent file reverted to inherit ---
    caste_model_test.go:202: keeper (the knowledge-keeper) declares model "inherit" in its own agent file, want sonnet — a pin that lives only in the display table does not route
    caste_model_test.go:210: keeper still declares model: inherit — no role may run on whatever model happened to run last (D-02)
```

## Files Created/Modified

- `cmd/caste_model_reason.go` (new) — `casteModelReasons`, the role→reason table for the ten expensive roles, and `casteModelReason`, the resolver the card and the fast path consume. Chooses nothing; only explains a choice the agent files already made.
- `cmd/caste_model_test.go` — added `agentModelLines`, `TestRoutineBuilderIsSonnetNeverInherit` and `TestOpusRequiresRecordedReason` with the forbidden-vocabulary list. Updated the one stale assertion in `TestModelTagEnvOverride` (see deviations).
- `cmd/codex_visuals.go` — `casteModelSlot` entries for chronicler/includer/keeper moved `inherit` → `sonnet`; the `resolveCasteModel` comment now records that the `inherit` branch is a defensive fallback D-02 forbids reaching.
- `cmd/ceremony_team_checkin.go` — the card collects `models`/`model_reasons` per caste, renders the plain-English model gloss and the expensive-model justification on each worker line, and returns both as fields; `renderBuildFastPathSummary` does the same for the one-worker surface.
- `cmd/ceremony_team_checkin_test.go` — the three new card tests.
- `.claude/agents/ant/aether-chronicler.md`, `aether-keeper.md`, `aether-includer.md` — `model: inherit` → `model: sonnet`.

The OpenCode and Codex agent definitions declare no model field at all (`grep -l '^model:' .opencode/agents/*.md` and the Codex TOML equivalent both return zero files), so cross-platform model parity does not apply and nothing there was touched.

## Rendered card (verification step 3)

```
━━━ 🔨 T E A M   C H E C K - I N ━━━

── Team ──
  🔨🐜 Builder [sonnet]  REQUIRED  — writes the code for 3 tasks: add the password reset flow  (the cheaper model)

  ⚔️🐜 Gatekeeper [opus]  OPTIONAL  — the phase changes how people log in, so the work is checked for security holes  (kept on the more expensive model because it hunts for security holes — leaked passwords, a missing permission check — where one miss is the whole point of the check; a cheaper model finds the textbook cases and misses the ones that matter)

  📝🐜 Chronicler [sonnet]  OPTIONAL  — writes up what changed so the next person can follow it  (the cheaper model)

── Summary ──
Required workers stay — they are the safety floor. Optional workers can be trimmed.
```

## Decisions Made

- **Assert the resolved model, not the config text.** `TestRoutineBuilderIsSonnetNeverInherit` checks `resolveCasteModel` *and* re-reads the `model:` line out of the agent file that actually routes. A pin recorded only in the Go display table would satisfy a text check and route nothing.
- **The reason table is keyed to the expensive model only.** An entry for a cheap role fails the test, so a role moving to the cheaper model cannot leave a stale justification behind claiming it is expensive.
- **Reasons are tested as owner-facing text.** Each must contain the word "cheaper" (so it says what a cheaper model gets wrong), must not be a placeholder, must contain no file path, and must contain no word from a forbidden list built from CLAUDE.md's invented-vocabulary table plus every role's own lookup word — matched as whole words, so "important" survives while "ant" does not.
- **Named the models in plain words.** `[sonnet]` and `[opus]` are the real model names and stay on the line, but they mean nothing without a glossary, so each line also says "the cheaper model" or "kept on the more expensive model because…". Asserted, not just written.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] `TestModelTagEnvOverride` asserted the behaviour D-02 abolishes**

- **Found during:** Task 1 (pinning the three routine roles)
- **Issue:** The existing test asserted `resolveCasteModel("chronicler") == "session"` — that chronicler runs on whatever model the session used. That is exactly the sentinel D-02 removes, so pinning chronicler made the test fail on a stale expectation, not on a real regression.
- **Fix:** Replaced the assertion with a stronger one: with `ANTHROPIC_DEFAULT_SONNET_MODEL=glm-5-turbo`, chronicler resolves to `glm-5-turbo` — proving it now follows the cheaper slot's override like every other pinned role. A comment records what the assertion used to say and why it changed.
- **Files modified:** `cmd/caste_model_test.go`
- **Verification:** `go test ./cmd -run 'Test(ModelTagEnvOverride|SpawnLineCarriesModelTag)' -count=1` passes
- **Committed in:** `47b0af20` (Task 1 GREEN)

**2. [Rule 2 - Missing Critical] The card named models the owner cannot interpret**

- **Found during:** Task 2, at the plan's own third verification step ("render one team card by hand and read it as someone who has never opened a file here")
- **Issue:** Each worker line named its model as `[sonnet]` or `[opus]`. Neither word tells a non-technical reader which one costs more, so the card satisfied "names the model" while failing the plan's readability check and CLAUDE.md's binding rule for owner-facing strings.
- **Fix:** Every worker line now also states the fact in words — `(the cheaper model)`, or `(kept on the more expensive model because it …)` where there is an expense to justify. Same on the fast-path summary. Two assertions added so the gloss cannot be dropped silently.
- **Files modified:** `cmd/ceremony_team_checkin.go`, `cmd/ceremony_team_checkin_test.go`
- **Verification:** `TestCheapModelWorkerNeedsNoReasonOnTheCard` and `TestFastPathSummaryCarriesModelAndReason` now fail if the gloss is removed; card re-rendered and read (above)
- **Committed in:** `4103d2ae` (Task 2 GREEN)

---

**Total deviations:** 2 auto-fixed (1 bug, 1 missing critical)
**Impact on plan:** Both were required for the plan's own success criteria. Neither expanded scope: no code selects a model, and the card gained clauses, not structure.

## Issues Encountered

None. The `model := "inherit"` fallback in `loadCasteAssignments` (`cmd/command_truth.go`) was reviewed and deliberately left alone — it is the default reported when an agent file declares no model line at all, not a role pinned to the sentinel, and it now reads `sonnet` for all three routine roles because it reads from disk.

## Verification

- `go test ./cmd -run 'Test(CasteModel|RoutineBuilder|OpusRequires|TeamCard|TeamCheckin|FastPathSummary|CheapModelWorker|OneWorker|RequiredMeansBuilderOrNamedSignal|NoTokenCountIsDerivedFromLength|ReviewerForcedOnlyByNamedRisk|NoWorkerWithoutStatedReason|OneTaskBugFixIsOneWorker)' -count=1` — **ok**, 22 tests pass, including every Phase 194/195 card test unchanged
- `go build ./...` — **ok**
- `go vet ./cmd/ ./pkg/codex/` — **ok**
- `gofmt -l cmd/ pkg/` — **clean**
- Team card rendered by hand and read (above) — plain English throughout

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- COST-04 is satisfied and the model facts are now available as structured fields, which plan 196-07's closeout cost line can read without re-deriving anything.
- No blockers. Nothing in this plan touches the spend ledger, the transcript readers, or `aether spend`.

## Known Stubs

None.

## Self-Check: PASSED

- `cmd/caste_model_reason.go` exists on disk — FOUND
- `acc95937`, `47b0af20`, `865f2075`, `4103d2ae` all present in `git log` — FOUND
- Both tasks' `<acceptance_criteria>` re-run and passing, including both required mutation proofs
- Plan-level `<verification>` commands re-run and passing (above)

---
*Phase: 196-see-what-it-cost*
*Completed: 2026-08-28*
