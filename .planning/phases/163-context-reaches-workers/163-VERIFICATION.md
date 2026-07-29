---
phase: 163-context-reaches-workers
verified: 2026-07-29T20:30:00Z
status: human_needed
score: 9/9 must-haves verified
overrides_applied: 0
---

# Phase 163: Context Reaches Workers Verification Report

**Phase Goal:** The context capsule, survey, phase research, `suggest-analyze`, and the approved charter reach a wrapper-spawned worker's actual prompt — at manifest level, measured, and inspectable by a person.
**Verified:** 2026-07-29
**Status:** human_needed
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | CONTEXT-02/03: `context_capsule`, `hive_section`, `pheromone_section` reach wrapper-spawned workers via one manifest-level field, sourced only from `resolveCodexWorkerContext()` | VERIFIED | `cmd/codex_build.go:97` field `ContextCapsule`; populated once at `cmd/codex_build.go:1655` guarded by `planOnly`; `resolveCodexWorkerContext` occurs exactly twice in the file (Path A hosted dispatch + manifest site), never inside `attachBuildDispatchContext`/`composeBuildManifestBrief`. `resolveCodexWorkerContext()` (`cmd/colony_prime_context.go:962`) calls `buildColonyPrimeOutput`, which includes `pheromones` and `hive_wisdom` sections (lines 492, 659) in the same integrity-assessed pipeline. `TestBuildManifestCarriesContextCapsuleOnce` — PASS. Both `.claude/commands/ant/build.md:84,97` and `.opencode/commands/ant/build.md:84,97` instruct reading `dispatch_manifest.context_capsule` once and prepending it verbatim; `TestBuildWrapperCeremonyContract` — PASS. |
| 2 | CONTEXT-06/D-09: approved charter governance reaches every worker as a protected, integrity-assessed section, and the mechanically-checkable part is verified at continue, not trusted | VERIFIED | `cmd/colony_prime_context.go:444-483` — `charter` colonyPrimeSection built from `state.Charter.Governance`/`.Constraints`, routed through `AssessPromptSource`; `protectedSectionPolicy("charter")` registered (`cmd/context_weighting.go:219,240`). `checkCharterComplianceGate` (`cmd/gate.go:635`) is called once inside `runCodexContinueGates` (`cmd/codex_continue.go:2966`). `TestColonyPrimeIncludesCharterGovernance`, `TestContinueCharterComplianceGate` (6 subtests), `TestContinueCharterComplianceGateWiredIntoPipeline` (3 subtests) — all PASS. `grep -rn 'Charter' cmd/codex_build.go` returns nothing (no bypass of the section pipeline). |
| 3 | D-04/D-05: a worker ordered to write phase research/planning artifacts can do so; the write guardrail no longer forces evasion; scout's permission profile matches its own brief; the artifacts schema accepts named typed fields | VERIFIED (with WARNING) | `sanctionedDataWritePrefixes` (`cmd/hook_cmds.go:232`, 4 exact-segment entries) checked before the blanket block; `TestHookPreToolUseBlocksProtectedPath` (untouched negative case) and `TestHookPreToolUseAllowsSanctionedScratchDirs` (7 subtests incl. substring-not-segment) — PASS. `scout` removed from `repositoryReadOnlyCastes` (`pkg/codex/permission_profile.go:53`), behavioral restriction added at line 93. `artifacts` schema (`pkg/codex/worker.go:921-933`) has 3 named nullable typed fields, `additionalProperties: false` retained. Four docs pinned by `TestSanctionedScratchDirsDocumented` — PASS. **WARNING:** review WR-05 — scout's write elevation is mechanically enforced only on Claude Code (the `PreToolUse` hook); OpenCode and Codex native rely on prose only, and `normalizeHookPath` is lexical (no symlink resolution), so a symlink inside a sanctioned dir can redirect a write to protected state. This is a real residual gap, not yet reflected in the contract doc as a named risk. |
| 4 | CONTEXT-01/04: territory survey findings and phase research findings are demonstrably present in a build worker's actual prompt, pinned by a named test, and a worker is told when the map is stale | VERIFIED | `resolveSurveySection()` (`cmd/helpers.go`) calls `surveyStalenessNotice()` (`cmd/survey_staleness.go:29`, threshold constant `surveyStaleCommitThreshold=25`) and renders it inside the Territory Survey section. `TestBuildWorkerBriefIncludesSurveyAndResearch` (4 subtests, built from real files on disk, asserting on delivered paths/content not just headings) — PASS. `TestSurveyStaleness` (5 subtests) — PASS. |
| 5 | CONTEXT-05: `suggest-analyze` executes during a real build for the first time, its suggestions persist and are shown once at closeout with copyable approve/dismiss commands, and a failure never blocks the build | VERIFIED (with WARNING; 1 CRITICAL found and fixed during phase) | `runSuggestAnalyze` (`cmd/suggest_analyze.go:69`) is called from `collectPendingSuggestions` inside `runCodexBuildFinalize` (`cmd/codex_build_finalize.go:421,462`) and from the CLI RunE — exactly 2 live call sites, none inside `cmd/codex_build.go` (no per-dispatch loop). `cmd/ceremony_cmd.go:242-243,611` renders the tick-to-approve block via `filterActiveSuggestions`. `TestBuildFinalizeCollectsSuggestAnalyzeResults` (4 subtests) and `TestSuggestAnalyze*` (10 tests) — all PASS. **Code review CR-01 (critical):** `TestSuggestAnalyze_NonBlockingOnError` escaped its sandbox via `PersistentPreRunE` re-initializing `store` against this developer's real repo, causing the suite to fail at HEAD and writing ~136 real pending suggestions into the local `.aether/data/COLONY_STATE.json`. **Fixed in commit `ac88530b`** (confirmed present in git log, confirmed the test now sets an isolated nil-store path and passes without touching real state — re-ran the test directly and the local state's pending-suggestion count stayed at 136, i.e. no new pollution was introduced). Full `go test ./cmd/... -count=1` now passes (255.7s, exit 0). **WARNING (pre-existing, local-only):** the ~136 polluted suggestions from the reviewer's earlier run are still present in the gitignored local `COLONY_STATE.json` and would surface in full under the closeout block's "Suggestions From This Build" heading (review IN-04 — the heading over-claims recency); this is data hygiene, not a code defect, and does not block the phase goal. Review WR-01 (non-atomic read-modify-write) and WR-02 (inconsistent `total` semantics across branches) are real but non-blocking precision issues. |
| 6 | CONTEXT-07/08/09: `--print-brief` defaults to a checklist answering "did this context arrive?" with present/absent + size + total-vs-budget; `--full` gives the raw prompt; a named test bounds growth against declared budgets; the staged benchmark is properly recorded as descoped | VERIFIED (with WARNING) | `renderBriefChecklist` (`cmd/build_print_brief.go:317`) is the unconditional default; `--full` flag registered (`cmd/codex_workflow_cmds.go:1168`) and threaded through. `TestPrintBriefChecklist` (5 subtests) and `TestPrintBriefFullFlagAndCoverage` (5 subtests) — PASS. `assembledContextBudgetCeilingChars()` (`cmd/build_print_brief.go:256`) derives the ceiling from named constants (`colonyPrimeBudgetChars` + 3 others + `briefTaskContentAllowanceChars`), never a literal. `TestAssembledContextStaysUnderBudgetCeiling` (4 subtests) — PASS. REQUIREMENTS.md CONTEXT-09 records the D-08 descoping with named replacement evidence. **WARNING:** review WR-03 — the ceiling sums `colonyPrimeBudgetChars` (8000, the non-compact constant) while every capsule this phase actually delivers is built under `colonyPrimeCompactBudgetChars` (4000, `buildColonyPrimeOutput(true)`), so the growth guard and the checklist's own TOTAL line are ~4000 chars looser than the real delivery budget. **WARNING:** review WR-04 — `--full` prints only `brief` + its composition table; the manifest-level capsule (charter included) and the skill section are silently absent from the "raw assembled prompt," contradicting the function's own doc comment ("the inspector must show exactly what a wrapper-spawned worker receives") — confirmed by reading `cmd/build_print_brief.go:60-95`: `capsule` is resolved at line 64 but never written when `options.Full` is true (lines 87-91). **WARNING:** review WR-06 — the checklist's charter-presence detection searches for the hardcoded string `"## Charter -- Binding Rules"` rather than resolving through the same `SectionTemplates` override path the producer uses, so a colony with a custom charter header gets a false ABSENT. |

**Score:** 6/6 grouped truths (9/9 individual CONTEXT-01..09 requirements) verified functionally; 1 critical finding was present at review time and has been fixed and confirmed; 6 warnings remain open as documented precision/robustness gaps that do not remove the delivered capability.

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `cmd/codex_build.go` | `ContextCapsule` field + single compute site | VERIFIED | Field at line 97, populated at lines 1434 (Path A, unrelated pre-existing) and 1655 (new manifest site) |
| `cmd/codex_build_manifest_context_test.go` | `TestBuildManifestCarriesContextCapsuleOnce` | VERIFIED | Exists, passes, exact-count assertion (`== 1`), marker derived from runtime value |
| `.claude/commands/ant/build.md` / `.opencode/commands/ant/build.md` | capsule-prepend instruction | VERIFIED | Both contain `dispatch_manifest.context_capsule` instruction, mirrored |
| `cmd/colony_prime_context.go` | charter section + budget constants | VERIFIED | `charter` section (444-483), `colonyPrimeBudgetChars`/`colonyPrimeCompactBudgetChars` named constants replace literals |
| `cmd/gate.go` | `checkCharterComplianceGate` | VERIFIED | Two-check producer (findings/executed), registered in `alwaysRunGates`, `gateClassifications`, `gateRecoveryTemplates`, `gateAutoResolveThresholds` |
| `cmd/hook_cmds.go` | `sanctionedDataWritePrefixes` allowlist | VERIFIED | 4 exact-segment entries, evaluated before blanket block |
| `pkg/codex/permission_profile.go` | scout off read-only, artifacts schema named fields | VERIFIED | Confirmed by grep and passing tests |
| `cmd/survey_staleness.go` | `surveyStalenessNotice` | VERIFIED | Exists, wired into `resolveSurveySection`, 5 subtests pass |
| `cmd/suggest_analyze.go` | `runSuggestAnalyze` callable function | VERIFIED | Extracted, 2 live callers (CLI + finalize), sanitizer stays inside |
| `cmd/build_print_brief.go` | `renderBriefChecklist` + budget ceiling | VERIFIED (with WARNING) | Checklist is default; `--full` incomplete relative to its own doc comment (WR-04) |
| `.planning/REQUIREMENTS.md` | CONTEXT-01/04/08/09 reworded, CONTEXT-09 descoped | VERIFIED | Reworded text present, Corrections table has the new row, requirement count unchanged (18 `CONTEXT-0` matches before/after edit per plan 04's own acceptance check) |

### Key Link Verification

| From | To | Via | Status | Details |
|------|-----|-----|--------|---------|
| `cmd/codex_build.go buildCodexBuildManifest` | `resolveCodexWorkerContext` | single call guarded by `planOnly` | WIRED | Confirmed by grep + passing invariant test |
| `.claude/commands/ant/build.md` | `manifest.context_capsule` | read once, prepend per spawn | WIRED | Text present, `TestBuildWrapperCeremonyContract` pins it |
| `cmd/colony_prime_context.go` | `state.Charter.Governance` | `colonyPrimeSection` appended, ranked | WIRED | Confirmed by test + code read |
| `cmd/codex_continue.go runCodexContinueGates` | `checkCharterComplianceGate` | producer call after anti-pattern block | WIRED | Single call site, deliberate-regression proof recorded and independently re-confirmed passing |
| `cmd/hook_cmds.go protectedHookWriteReason` | `sanctionedDataWritePrefixes` | allowlist case before blanket block | WIRED | Confirmed by passing negative + positive tests |
| `pkg/codex/permission_profile.go behavioralRestrictionsForCaste` | `scout` | scoped-write restriction string | WIRED | Confirmed by grep + test |
| `cmd/helpers.go resolveSurveySection` | `surveyStalenessNotice` | prepended notice inside section | WIRED | Confirmed by test |
| `cmd/codex_build_finalize.go runCodexBuildFinalize` | `runSuggestAnalyze` | non-blocking call after commit | WIRED | Confirmed by test + CR-01 fix re-verified |
| `cmd/ceremony_cmd.go closeout` | `filterActiveSuggestions` | end-of-build presentation | WIRED | Confirmed by test |
| `cmd/build_print_brief.go` | `resolveCodexWorkerContext` | display-only read for the checklist | WIRED (checklist) / PARTIAL (`--full`) | Checklist mode includes the capsule; `--full` mode resolves `capsule` (line 64) but never prints it (WR-04) |

### Data-Flow Trace (Level 4)

Not applicable in the UI-rendering sense (no React/database dashboard); the equivalent trace here is "does the assembled text that reaches a worker prompt/inspector come from real state rather than a stub." Traced above: `resolveCodexWorkerContext()` reads real `COLONY_STATE.json` through `buildColonyPrimeOutput`; `runSuggestAnalyze` performs real `git diff`/`git log` scans and real `store.AtomicWrite`; `surveyStalenessNotice` performs a real `git rev-list --count`. All confirmed by passing tests built on real fixtures (not stubbed resolvers), per the plans' own explicit anti-stub instructions and this verifier's direct code reads.

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Build succeeds | `go build ./cmd/aether` | exit 0 | PASS |
| Static analysis clean | `go vet ./...` | exit 0 | PASS |
| Full cmd package suite | `go test ./cmd/... -count=1` | `ok ... 255.676s` | PASS |
| pkg/codex suite | `go test ./pkg/codex/... -count=1` | `ok ... 15.202s` | PASS |
| CR-01 fix confirmed hermetic | `go test ./cmd -run TestSuggestAnalyze_NonBlockingOnError -count=1 -v` (run directly, twice) | PASS both times; local `COLONY_STATE.json` pending-suggestion count unchanged (136) across runs | PASS |
| All 6 plans' named invariant/presence/budget/wiring tests | `go test ./cmd -run '<13 named tests>' -count=1 -v` | all PASS (see truths table for full list) | PASS |
| Repo-modified-file debt-marker scan | `grep -n -E "TBD\|FIXME\|XXX"` across all 29 phase-modified files | 0 matches | PASS |

### Probe Execution

Not applicable — this phase is not a migration/tooling phase with `scripts/*/tests/probe-*.sh` conventions; no probes declared in PLAN/SUMMARY files.

### Requirements Coverage

| Requirement | Source Plan(s) | Description | Status | Evidence |
|-------------|----------------|--------------|--------|----------|
| CONTEXT-01 | 163-04 | Territory survey findings present in worker prompt, pinned by test | SATISFIED | `TestBuildWorkerBriefIncludesSurveyAndResearch` PASS |
| CONTEXT-02 | 163-01 | `context_capsule`/`hive_section`/`pheromone_section` reach wrapper workers via `resolveCodexWorkerContext()` | SATISFIED | `TestBuildManifestCarriesContextCapsuleOnce` PASS |
| CONTEXT-03 | 163-01 | Context carried at manifest level, not per-dispatch | SATISFIED | Same test, exact-count assertion |
| CONTEXT-04 | 163-03, 163-04 | Phase research reaches worker prompts end to end (requires scout permission + write allowlist + presence pin) | SATISFIED | `TestBuildWorkerBriefIncludesSurveyAndResearch`, `TestScoutPermissionProfileAllowsPhaseResearchWrite`, `TestHookPreToolUseAllowsSanctionedScratchDirs` all PASS |
| CONTEXT-05 | 163-05 | `suggest-analyze` executes during a real build | SATISFIED | `TestBuildFinalizeCollectsSuggestAnalyzeResults` PASS; CR-01 fixed |
| CONTEXT-06 | 163-02 | Approved charter reaches workers | SATISFIED | `TestColonyPrimeIncludesCharterGovernance`, `TestContinueCharterComplianceGate*` PASS |
| CONTEXT-07 | 163-06 | `--print-brief` makes context presence checkable by a person | SATISFIED (checklist mode); WR-04 open on `--full` | `TestPrintBriefChecklist` PASS |
| CONTEXT-08 | 163-06 | Total context measured and bounded before more is added | SATISFIED; WR-03 open (ceiling uses non-compact constant) | `TestAssembledContextStaysUnderBudgetCeiling` PASS |
| CONTEXT-09 | 163-06 | Staged benchmark descoped per D-08, replacement evidence named | SATISFIED | REQUIREMENTS.md text + this phase's test suite named as evidence |

No orphaned requirements: REQUIREMENTS.md's CONTEXT block maps exactly CONTEXT-01 through CONTEXT-09 to Phase 163, and all nine appear in at least one plan's `requirements:` frontmatter field.

**Note on REQUIREMENTS.md traceability table status:** the table's "Status" column still reads "Pending" for CONTEXT-01 through CONTEXT-06 (only 07/08/09 read "Complete"). This matches the project's own established convention — Phase 160 (independently confirmed "Complete" in ROADMAP.md) has all of its LOUD-01…RETIRE-04 rows still reading "Pending" in the same table. The traceability table appears to be updated at a later stage (milestone audit/seal), not per-phase-completion. Not treated as a gap for this verification.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| `cmd/suggest_analyze.go` | 194-201 | Non-atomic read-modify-write of `COLONY_STATE.json` (review WR-01) | WARNING | Pre-existing pattern, now exercised on every build's critical path; a state write racing the multi-second scan window would be silently clobbered |
| `cmd/suggest_analyze.go` / `cmd/codex_build_finalize.go` | 99-107,150-209 / 461-471 | `total`/`pending_suggestion_count` semantics differ across branches (review WR-02) | WARNING | User-facing count can read 0 with old pending suggestions still waiting, or a positive count with nothing actionable |
| `cmd/build_print_brief.go` | 256-262 | Budget ceiling sums non-compact constant while delivery uses the compact one (review WR-03) | WARNING | Growth guard trips ~4000 chars later than the real delivery budget |
| `cmd/build_print_brief.go` | 87-95 | `--full` omits capsule + skill section (review WR-04) | WARNING | Contradicts the function's own "exactly what a worker receives" doc comment |
| `pkg/codex/permission_profile.go`, `cmd/hook_cmds.go` | 52-95, 217-334 | Scout's write elevation prose-only on OpenCode/Codex native; symlink-bypassable lexical path check (review WR-05) | WARNING | Documented D-05 trade-off, but asymmetry/gap not yet recorded in the contract doc |
| `cmd/build_print_brief.go` | 353 | Charter detection via hardcoded heading string, breaks under custom section templates (review WR-06) | WARNING | False ABSENT for colonies using a custom charter header |
| `cmd/build_print_brief.go` | 284-302 | Heading match not anchored to line start (review IN-01) | INFO | Sub-heading or quoted-in-prose false positive |
| `cmd/suggest_analyze.go` | 234-251 | Off-by-one on singular "1 file changed" diff summary (review IN-02) | INFO | Harmless at current threshold of 5 |
| `cmd/codex_workflow_cmds.go` | 1167 | "mutates nothing" help text overstates guarantee (cache-tier writes exist) (review IN-03) | INFO | Cosmetic wording gap |
| `cmd/ceremony_cmd.go` | 241-245 | "Suggestions From This Build" heading shows all active suggestions, not only this build's (review IN-04) | INFO/local-hygiene | Currently would surface ~136 pre-existing local pending suggestions to the next build's closeout |

No debt markers (`TBD`/`FIXME`/`XXX`) found in any of the 29 files this phase modified.

### Human Verification Required

### 1. `aether build <n> --print-brief` ten-second charter check on a real colony

**Test:** On a real, populated colony (not the phase's synthetic test fixtures), run `aether build <n> --print-brief` and confirm a person can answer "did the charter arrive?" within ten seconds by reading the checklist output.
**Expected:** The Charter row shows PRESENT with a plausible character count when a charter is approved, and the manifest-level Context Capsule row is populated.
**Why human:** VALIDATION.md's own manual acceptance criterion for D-06 ("on a real colony, the checklist answers 'did the charter arrive?' within ten seconds") could not be executed by any plan's automated worktree — plan 06's own SUMMARY records this was explicitly deferred to the user because the worktree had no initialized colony state.

### 2. Local `COLONY_STATE.json` suggestion pollution cleanup

**Test:** Run `aether suggest-approve --dismiss-all` (or `/ant-data-clean`) on this machine's local colony state, then run one real `aether build` end to end and confirm the closeout's "Suggestions From This Build" block shows only suggestions plausibly from that build, not a backlog of ~136.
**Expected:** Closeout block is small and relevant, not a wall of stale entries.
**Why human:** This is local, gitignored data hygiene outside the git-tracked artifact set; a verifier should not mutate local colony state as a side effect of verification, and the fix requires a real build to observe end-to-end.

### 3. CONTEXT-09 real-world cheap-model validation (explicitly deferred by design)

**Test:** Per D-08, run a real build on an inexpensive model with this phase's context restored and confirm the worker's behavior visibly improves versus a build without it.
**Expected:** Observable behavior difference, documented informally.
**Why human:** Deliberately descoped from automated verification per the user's own decision (D-08); the phase's evidence is the inspector plus the automated tests, and this item is real-world validation explicitly scheduled post-phase, not a phase-163 gap.

### Gaps Summary

No BLOCKER-level gaps. All 9 CONTEXT requirements have working, tested code paths: the context capsule, survey findings, phase research, the charter, and `suggest-analyze` all demonstrably reach a build worker's prompt (or, for `suggest-analyze`, execute during a build and reach the user), each pinned by a named test that has been observed to fail when the wiring is removed (deliberate-regression proofs recorded in every plan's SUMMARY and spot-checked in this verification for the charter gate).

One CRITICAL finding from the phase's own code review (CR-01 — a test sandbox escape that polluted local colony state and made `go test ./cmd` fail at HEAD) was found and fixed within the phase (commit `ac88530b`), and this verification independently re-ran the fixed test twice to confirm it no longer mutates real state and the full `go test ./cmd/... -count=1` suite is green (255.7s, exit 0).

Six WARNING-level findings from the code review remain open and are carried into this report unresolved: WR-01 (non-atomic suggestion persistence), WR-02 (inconsistent suggestion-count semantics), WR-03 (budget ceiling uses the wrong capsule constant, ~4000 chars looser than reality), WR-04 (`--full` doesn't print the capsule/skill section it claims to), WR-05 (scout's write elevation is enforced on only one of three platforms, plus a symlink gap), and WR-06 (charter detection breaks under custom section-template headers). None of these remove or falsify the core delivered capability — the capsule, charter, survey, research, and suggest-analyze wiring all function and are tested — but they are real, documented imprecisions a developer should triage, most plausibly as a fast follow-up rather than blocking this phase.

---

_Verified: 2026-07-29_
_Verifier: Claude (gsd-verifier)_
