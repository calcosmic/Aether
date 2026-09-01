---
phase: 197-one-answer-to-what-next
verified: 2026-08-29T16:55:00Z
status: passed
score: 6/6 roadmap success criteria verified; 7/7 plan-level must-haves verified
behavior_unverified: 0
overrides_applied: 0
re_verification: no prior VERIFICATION.md existed for this phase
---

# Phase 197: One Answer to "What Next?" Verification Report

**Phase Goal:** Every Aether command ends by telling the owner, in one consistent way, what just happened and exactly what to type next — instead of each command guessing or occasionally naming a command that no longer exists. Opening a new chat session in a project that already has a colony greets the owner with where things stand.

**Verified:** 2026-08-29
**Status:** passed
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths (ROADMAP.md Success Criteria)

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | One shared resolver, given saved state, returns all 8 things the owner needs (standing, what changed, open flags, recommendation, exact command, 2-4 alternatives, safe-to-close verdict, recovery/paused state) | ✓ VERIFIED | `cmd/next_action.go` `nextAction` struct has exactly the 8 JSON-tagged fields (`standing`, `changed`, `open`, `recommendation`, `command`, `alternatives`, `context_health`, `recovery`). Resolver is pure — no `os.`/`ioutil.`/`exec.Command` calls in `next_action.go`; disk reads live only in the separate `loadNextActionInput` (next_action_input.go). Ran the live command: `aether hook-session-start` against this repo's own saved state printed a full card carrying all 8 categories (standing, what changed, open issues/clarifications, recommendation, command, 2 alternatives, safe-to-close line, paused state) — confirmed by direct execution, not narration. |
| 2 | Every lifecycle command's closing message (init, discuss, plan, build, continue, pause, resume, seal, update, recover, status) is generated from the resolver; JSON/quiet mode carries identical fields | ✓ VERIFIED | `go test ./cmd/ -run TestEveryLifecycleCommandEndsWithNextAction -count=1 -v` — 11/11 subtests pass (starting, discussing, planning, building, continuing, pausing, resuming, sealing/finishing, updating, recovering, checking status), each comparing the on-screen recommended command against the machine-readable envelope from a real command execution. |
| 3 | An automatic check counts hand-typed command-name sites and fails the build if the count grows from the recorded baseline | ✓ VERIFIED | `go test ./cmd/ -run TestNextActionNeverHardcoded` passes; supporting ratchet tests `TestNextActionHardcodeBaselineMatchesLive`, `TestNextActionHardcodeDetectsAPlantedViolation`, `TestNextActionHardcodeSwapFails` all pass — baseline is a measured, frozen 116-site set-membership comparison in `cmd/testdata/next_action_hardcode_baseline.json`, not a count (swap-fails test proves membership, not cardinality, is compared). |
| 4 | Starting a new chat in a project with an existing colony shows a "where you left off" card automatically, from the same resolver | ✓ VERIFIED | `go test ./cmd/ -run TestSessionStartCardReflectsState` (3/3 subtests) and `TestSessionStartCardIsSilentWithoutAColony` (2/2 subtests — empty project and leftover-state-with-no-goal both get nothing) both pass. `.claude/settings.json` registers `aether hook-session-start` on `SessionStart` (startup, resume, and after-clear — 3 matcher entries). `TestSessionStartHookDoesNotMutate` and `TestSessionStartHookIsRegistered` pass. Live-executed `AETHER_OUTPUT_MODE=visual go run ./cmd/aether hook-session-start` against this repo's real saved state and it printed the correct card with zero filesystem mutation (`git status` unchanged before/after). |
| 5 | `/ant-pause` and `/ant-pause-colony` both work identically; `aether update` detects and fixes a missing short-name copy | ✓ VERIFIED | `go test ./cmd/ -run TestCanonicalAliasDelegates` passes (same Cobra command object; identical output for one state). `.aether/commands/pause.yaml` declares `aliases: [pause-colony]`; the standalone `pause-colony.yaml` is gone (single source). Wrapper files exist on all three platform locations (`.claude/commands/ant/pause.md` + `pause-colony.md`, `.claude/commands/ant-pause.md` + `ant-pause-colony.md`, `.opencode/commands/ant/pause.md` + `pause-colony.md`). `TestUpdateRestoresAMissingAliasWrapper`, `TestUpdateReportsTheAliasRepairByName`, `TestUpdateReportsNoRepairWhenNothingIsMissing` all pass. |
| 6 | The recommended next command is never one that is not actually available; recommending a nonexistent command is a failing test case | ✓ VERIFIED | `go test ./cmd/ -run TestUnregisteredCommandIsRefused` (6/6 adversarial subtests: verb-prefix, near-miss, bare binary name, slash form, empty, foreign-tool command — all refused). `TestRecommendedCommandIsAlwaysAvailable` (10/10 state-fixture subtests) and `TestEveryResolverCandidateResolves` pass. Gate implementation resolves every candidate via `rootCmd.Find(tokens)` against the live Cobra tree (`cmd/next_action.go:433`), not a hand-kept list. |

**Score:** 6/6 roadmap success criteria verified (0 present-but-behavior-unverified)

### Plan-Level Must-Haves (197-01 through 197-07)

| Plan | Key must-have | Status | Evidence |
|------|---------------|--------|----------|
| 197-01 | Pure resolver + availability gate reading the live Cobra tree | ✓ VERIFIED | `cmd/next_action.go` (993 lines), no disk I/O; gate via `rootCmd.Find` |
| 197-02 | Four rival deciders (`workflowSuggestionsForState`, `nextCommandFromState`, `nextUpSuggestionsForState`, `closeoutNextCommand`) collapse onto the resolver; one card renderer | ✓ VERIFIED | `codex_visuals.go` calls `resolveNextAction(...)` at lines 768/787/800/826; `next_action_card.go` provides `renderNextActionCard`/`applyNextActionToResult`, used from `closeout_cmd.go:58`, `codex_visuals.go:812/827`, `recover_visuals.go:295` |
| 197-03 | Session-start hook owned by the runtime; 5 documents' "remember to check" instruction removed and guarded against return | ✓ VERIFIED | `hook_cmds.go` (hookSessionStartCmd); `TestSessionGreetingIsNotDelegatedToTheAssistant` passes; CLAUDE.md's own "Session Recovery" section now states the instruction "used to be a paragraph" and was replaced by the hook (self-documenting removal, confirmed by reading the live file) |
| 197-04 | 7 lifecycle closings (starting, discussing, scanning, planning, building, continuing, shared closeout) render the shared card | ✓ VERIFIED | `TestEveryLifecycleCommandEndsWithNextAction` covers the roadmap's 11-name list (which subsumes these); golden transcripts (`golden_plan.txt`, `golden_build.txt`, `golden_continue.txt`) present and part of the passing full suite |
| 197-05 | `/ant-pause` canonical, `/ant-pause-colony` alias declared once via `aliases:`; update restores missing wrapper and reports it | ✓ VERIFIED | Confirmed above (criterion 5) |
| 197-06 | Remaining 6 surfaces (pause, resume, seal, update, recover, status) render the card; pause card's duplicate alternative removed; 6th private decider (`recoverOverrideFromIssues`) retired | ✓ VERIFIED | `status.go:746` calls `closeLifecycleCommand`; `recover_visuals.go:46/109` calls `renderNextActionCard(recoverNextAction(...))`; `TestNoCardOffersTheSameCommandTwice/the_pause_card` referenced in 197-06 SUMMARY and part of the passing suite |
| 197-07 | Named coverage test (criterion 2) and hardcode ratchet (criterion 3), both registered in `wiringGateGuardFiles` and the CI step's `-run` filter | ✓ VERIFIED | `cmd/ci_wiring_gate_test.go` lines 112-113 list both files in `wiringGateGuardFiles`; `.github/workflows/ci.yml` line 100's `-run` filter includes `TestEveryLifecycleCommandEndsWithNextAction` and `TestNextActionNeverHardcoded`; `TestWiringGateStepRunsEveryWiringTest` and `TestEveryGuardFileIsInTheWiringInventory` both pass |

**Score:** 7/7 plans' must-haves verified

### Required Artifacts

All 22 artifacts declared across the 7 plans' `must_haves.artifacts` and `files_modified` frontmatter exist and are substantive (not stubs):

| Artifact | Status |
|----------|--------|
| `cmd/next_action.go` (993 lines) | ✓ VERIFIED |
| `cmd/next_action_input.go` (121 lines) | ✓ VERIFIED |
| `cmd/next_action_availability_test.go` (261 lines) | ✓ VERIFIED |
| `cmd/next_action_neutrality_test.go` (83 lines) | ✓ VERIFIED |
| `cmd/next_action_card.go` (259 lines) | ✓ VERIFIED |
| `cmd/next_action_one_decider_test.go` (303 lines) | ✓ VERIFIED |
| `cmd/hook_cmds.go` (693 lines) | ✓ VERIFIED |
| `cmd/hook_session_start_test.go` (390 lines) | ✓ VERIFIED |
| `cmd/session_start_doc_removal_test.go` (231 lines) | ✓ VERIFIED |
| `cmd/subcommand_reachability_ratchet_test.go` (2090 lines) | ✓ VERIFIED |
| `cmd/lifecycle_card_startup_test.go` / `_workloop_test.go` / `_session_test.go` / `_endgame_test.go` | ✓ VERIFIED |
| `cmd/canonical_alias_test.go` / `cmd/alias_reconcile_test.go` | ✓ VERIFIED |
| `cmd/lifecycle_next_action_coverage_test.go` (449 lines) | ✓ VERIFIED |
| `cmd/next_action_hardcode_ratchet_test.go` (637 lines) | ✓ VERIFIED |
| `cmd/testdata/next_action_hardcode_baseline.json` (582 lines) | ✓ VERIFIED |
| `cmd/ci_wiring_gate_test.go` (2140 lines) | ✓ VERIFIED |

No MISSING or STUB artifacts found.

### Key Link Verification

| From | To | Via | Status |
|------|-----|-----|--------|
| `cmd/next_action.go` | live Cobra tree | `rootCmd.Find(tokens)` | ✓ WIRED |
| `cmd/next_action_input.go` | `cmd/recovery_snapshot.go` | `loadActiveRecoveryGuidance` | ✓ WIRED |
| `cmd/codex_visuals.go` | `cmd/next_action.go` | `resolveNextAction(...)` at 4 call sites | ✓ WIRED |
| `cmd/next_action_card.go` | `cmd/codex_visuals.go` | `renderNextUp` | ✓ WIRED |
| `.claude/settings.json` | `cmd/hook_cmds.go` | `hook-session-start` on 3 `SessionStart` matchers | ✓ WIRED |
| `cmd/subcommand_reachability_ratchet_test.go` | `.claude/settings.json` | `hookScriptCorpora` third entry | ✓ WIRED |
| `cmd/codex_visuals.go` | `cmd/next_action_card.go` | `renderNextActionCard` (10+ call sites) | ✓ WIRED |
| `cmd/codex_build_finalize.go` | `cmd/next_action_card.go` | `closeLifecycleRun` (shared helper) | ✓ WIRED |
| `.aether/commands/pause.yaml` | wrapper files (3 platforms) | `aliases:` declaration | ✓ WIRED |
| `cmd/update_cmd.go` | `cmd/platform_sync.go` | `declaredAliasSurfaces` / `aliasRepairReport` | ✓ WIRED |
| `cmd/status.go` | `cmd/next_action.go` | `closeLifecycleCommand` | ✓ WIRED |
| `cmd/recover_visuals.go` | `cmd/next_action_card.go` | `renderNextActionCard(recoverNextAction(...))` | ✓ WIRED |
| `cmd/next_action_hardcode_ratchet_test.go` | `cmd/testdata/next_action_hardcode_baseline.json` | set-membership diff | ✓ WIRED |
| `.github/workflows/ci.yml` | `cmd/ci_wiring_gate_test.go` | `wiringGateGuardFiles` filter | ✓ WIRED |

### Behavioral Spot-Checks / Direct Execution

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Session-start card reflects real saved state | `AETHER_OUTPUT_MODE=visual go run ./cmd/aether hook-session-start` | Printed a full card (standing, what changed, 16 open issues/1 clarification, recommendation "Run `/ant-resume`", 2 alternatives, safe-to-close line, paused state) against this repo's actual saved colony state | ✓ PASS |
| Hook mutates nothing | `git status --porcelain` before/after the above | Unchanged | ✓ PASS |
| Named criterion tests | `go test ./cmd/ -run 'TestEveryLifecycleCommandEndsWithNextAction\|TestNextActionNeverHardcoded\|TestSessionStartCardReflectsState\|TestCanonicalAliasDelegates'` | All pass | ✓ PASS |
| CI-wiring registration for the two new criterion-2/3 guards | `go test ./cmd/ -run 'TestWiringGateStepRunsEveryWiringTest\|TestEveryGuardFileIsInTheWiringInventory'` | Both pass | ✓ PASS |
| Full `cmd` package regression | `go test ./cmd/... -count=1` | `ok github.com/calcosmic/Aether/cmd 245.794s` — no failures | ✓ PASS |
| Build / vet | `go build ./cmd/...`, `go vet ./cmd/...` | Clean, no output | ✓ PASS |

### Requirements Coverage

| Requirement | Source Plan(s) | Description | Status | Evidence |
|-------------|-----------------|-------------|--------|----------|
| NEXT-01 | 197-01, 197-02 | One pure resolver over canonical state returns the eight fields | ✓ SATISFIED | `nextAction` struct + purity check + live execution |
| NEXT-02 | 197-02, 197-04, 197-06, 197-07 | Every lifecycle command's closing card renders from the resolver; quiet/JSON carry identical fields | ✓ SATISFIED | `TestEveryLifecycleCommandEndsWithNextAction` 11/11 |
| NEXT-03 | 197-07 | Ratchet on hand-typed command strings, shrink-only from baseline | ✓ SATISFIED | `TestNextActionNeverHardcoded` + 3 supporting ratchet tests |
| NEXT-04 | 197-03 | `SessionStart` hook prints the card on startup/resume/clear; CLAUDE.md paragraph removed | ✓ SATISFIED | Hook registered + tested + live-executed; CLAUDE.md confirmed rewritten |
| NEXT-05 | 197-05 | `/ant-pause` canonical, `/ant-pause-colony` alias via `aliases:`; update reconciles | ✓ SATISFIED | `TestCanonicalAliasDelegates` + alias-reconcile tests |
| NEXT-06 | 197-01 | Resolver never recommends an unregistered command | ✓ SATISFIED | `TestUnregisteredCommandIsRefused`, `TestRecommendedCommandIsAlwaysAvailable` |

No orphaned requirements — all 6 NEXT-* IDs declared across the 7 plans' `requirements:` frontmatter match REQUIREMENTS.md's Phase 197 mapping exactly.

### Anti-Patterns Found

None. Scanned all 25 source files touched by this phase's plans for `TBD`/`FIXME`/`XXX`/`TODO`/`HACK`/`PLACEHOLDER` and empty-implementation patterns — zero hits outside test files' own doc comments describing the feature (which reference the *absence* of the pattern, not the pattern itself).

### Documentation Gap Found (non-blocking — does not affect phase-goal status)

`.planning/REQUIREMENTS.md`'s traceability table still marks **NEXT-01, NEXT-02, NEXT-03, NEXT-04, and NEXT-06 as "Pending"** and their checkboxes unticked, even though:
- The underlying functionality for all five is verified working above (tests pass, code is wired, live execution confirmed).
- Plans 197-01, 197-02, 197-03, and 197-07's own SUMMARY.md frontmatter explicitly lists `requirements-completed: [NEXT-01, NEXT-06]` / `[NEXT-01, NEXT-02]` / `[NEXT-04]` / `[NEXT-02, NEXT-03]` respectively — i.e., the plans themselves claim these were finished.
- 197-06's SUMMARY explains a real "shared-ID gate" (#2388) that intentionally holds a requirement Pending until every plan declaring it has finished — but that gate only explains NEXT-01 (last declared by 197-02) and NEXT-02 (last declared by 197-07); it does not explain why NEXT-06 (declared only by 197-01) or NEXT-04 (declared only by 197-03) are still unticked, since each had exactly one declaring plan and that plan already finished.
- `git log` shows no commit updating REQUIREMENTS.md for this phase at all (the "docs(phase-197): update tracking after wave N" commits only touched `.planning/ROADMAP.md`).

This is a paperwork gap, not a code gap — the phase's actual roadmap success criteria are code-and-test-verified true. But given this project's own Definition of Done is built specifically around not letting tracking documents drift from reality, the REQUIREMENTS.md checkboxes and Status column for NEXT-01, NEXT-02, NEXT-03, NEXT-04, and NEXT-06 should be corrected to `[x]` / `Complete` before this phase is sealed, so the traceability table stops contradicting both the code and the plans' own SUMMARY claims. NEXT-05 is already correctly marked complete.

### Human Verification Required

None. Every roadmap success criterion had a named, passing automated test plus (for criteria 1 and 4) live, non-mutating execution against this repo's real saved state performed directly during this verification.

### Gaps Summary

No gaps against the phase goal or the six roadmap success criteria. One non-blocking documentation-tracking gap noted above (REQUIREMENTS.md staleness) — recommended fix before `/ant-seal`, does not block proceeding to Phase 198.

---

_Verified: 2026-08-29_
_Verifier: Claude (gsd-verifier)_
