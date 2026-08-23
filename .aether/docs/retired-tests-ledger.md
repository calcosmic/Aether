# Retired Tests Ledger

Updated: 2026-07-27

This file records every test file removed from this repository during the
v1.25 "Switch It On" milestone. The rule it enforces (RETIRE-04): no test
leaves the repository without a recorded disposition — either its coverage
survived in a named replacement test, or its loss is knowingly accepted and
written down, never silently dropped.

Each entry below carries four labeled fields: original path (repo-relative
path of the deleted file), what it covered (the concrete behaviour the test
asserted), disposition (exactly one of `dead-with-no-replacement` or
`recovered-by:<path-to-surviving-test>`), and removed in (the commit or
phase/plan that removed it).

### `control-ts/tests/schemas/policy.schema.test.ts`

- **Original path:** `control-ts/tests/schemas/policy.schema.test.ts`
- **What it covered:** Zod schema validation of the policy YAML field
  surface — `model_routing.default_provider`, `memory_rules.*`,
  `skill_creation.*`, `safety_gates.*`, `dispatch_contract.*`,
  `pheromone_lifecycle.*`, `signal_rules.*`, `autopilot.*`.
- **Disposition:** `recovered-by:cmd/policy_schema_test.go`. The Go
  replacement is a strict improvement in scope, not just language: it reads
  the live `colony/policies/*.yaml` files directly, while the retired TS
  test read fixture copies under `control-ts/tests/fixtures/policies/`
  that could silently drift from what the runtime actually loads.
- **Removed in:** Phase 160 Plan 06 (`control-ts/` deletion).

### `.aether/ts-host/test/playbook-loader.test.ts`

- **Original path:** `.aether/ts-host/test/playbook-loader.test.ts`
- **What it covered:** The TS host's playbook loader — loading
  `.aether/docs/command-playbooks/*.md` and injecting them into worker
  prompts.
- **Disposition:** `dead-with-no-replacement`. This is a retroactive entry:
  the feature it tested was deliberately removed, not relocated. Since
  v1.25, build/continue execution behavior lives in the host-manifest flow
  (`aether host build` → `dispatch_manifest` → `build-finalize`) instead of
  playbook injection, so there is no successor test to name.
- **Removed in:** commit `b2b41486`.

### `TestAutopilotCheckReplan` (function in `cmd/autopilot_test.go`)

- **Original path:** `cmd/autopilot_test.go` (single function removed; file
  survives)
- **What it covered:** the `autopilot-check-replan` subcommand's interval
  arithmetic — replan recommended after N completed phases.
- **Disposition:** `recovered-by:cmd/compatibility_cmds_test.go
  (TestRunAutopilotReplanDue)`. The subcommand was retired 2026-08-16: it had
  no caller anywhere (orphan allowlist entry removed in the same change), and
  the real autopilot loop in `runCompatibilityAutopilot` owns the replan
  arithmetic directly. The replacement test asserts the same behaviour where
  it actually runs. `TestAutopilotSuccessStatusCountsAsCompleted`, which had
  used check-replan as a probe for status normalization, was rewritten in the
  same change to assert normalization through `autopilot-update` state.
- **Removed in:** v5.4.0-richness restoration, Stage 2 (autopilot).

### Interactive `init-ceremony` tests (functions in `cmd/init_ceremony_test.go` and `cmd/shelf_todo_wiring_test.go`)

- **Original path:** `cmd/init_ceremony_test.go` (7 functions:
  `TestInitCeremonyRegistered`, `TestInitCeremonyProceed`,
  `TestInitCeremonyOrchestratorSelection`, `TestInitCeremonyCancel`,
  `TestInitCeremonyRejectThenApprove`, `TestInitCeremonyApproveBrief`,
  `TestInitCeremonyRejectBrief`) and `cmd/shelf_todo_wiring_test.go`
  (`TestInitCeremonySeedsSessionTodosFromPromotedShelf`). Files survive.
- **What they covered:** the interactive `aether init-ceremony` command — a
  TTY-gated approve/edit/cancel prompt loop and its parallel colony-creation
  path (`createCeremonyColony`).
- **Disposition:** the *command* was retired 2026-08-16: it was an orphan (no
  wrapper called it, allowlist entry removed in the same change), demanded an
  interactive terminal no wrapper platform provides, auto-approved pheromone
  suggestions — the exact "click-through" failure SEE-10 forbids — and its
  stderr output evaded the visual-writer discipline. Its *renderers* survived
  and gained live callers: `recovered-by:cmd/init_wrapper_ceremony_test.go`
  (wrapper approve/edit/cancel contract), `TestInitRendersCharterCeremony`
  (charter panel + colony-born close on the real `aether init` path),
  `TestInitResearchRendersScanPanels` (research panels + consent-framed signal
  suggestions on `aether init-research`), and
  `TestInitPromotesShelfEntriesAtomically` (shelf-todo seeding on the one
  remaining colony-creation path).
- **Removed in:** v5.4.0-richness restoration, Stage 3 (ceremonies).

### `TestWatcherIsAlwaysRequiredOnBuild` (function in `cmd/queen_probe_gating_test.go`)

- **Original path:** `cmd/queen_probe_gating_test.go` (single function
  removed; file survives).
- **What it covered:** that Watcher was unconditionally present in
  `queenBuildSafetyRequiredCastes`'s output for every build phase regardless
  of mode, content, or wording.
- **Disposition:** `dead-with-no-replacement`. Ruling D11
  (`.planning/decisions/2026-08-22-queen-decides-program-checks.md`)
  explicitly supersedes this rule: the build side no longer requires or
  dispatches a Watcher at all (Phase 193 D-08 stopped the dispatch; plan
  194-02 removes the requirement that used to restore it). The thing that
  checks the build now lives entirely in `continue`'s deterministic floor
  (`TestPhaseVerifiedOnce`), which this test never asserted against.
- **Removed in:** Phase 194 Plan 02.

### `TestQueenCannotDropTheWatcher` (function in `cmd/queen_judgement_test.go`)

- **Original path:** `cmd/queen_judgement_test.go` (single function removed;
  file survives).
- **What it covered:** the same claim as `TestWatcherIsAlwaysRequiredOnBuild`
  seen from the judgement side — `queenApplyJudgement` restoring Watcher into
  `Final` when a Queen proposal for the `build` flow omitted it.
- **Disposition:** `dead-with-no-replacement`, same ruling (D11). Watcher is
  no longer a required build caste for `queenApplyJudgement` to restore.
- **Removed in:** Phase 194 Plan 02.

### `TestSafetyCastesSurviveProbeGating` (function in `cmd/queen_probe_gating_test.go`)

- **Original path:** `cmd/queen_probe_gating_test.go` (single function
  removed; file survives).
- **What it covered:** that `queenBuildSafetyRequiredCastes` kept Auditor,
  Gatekeeper, and Watcher required on a production/security-worded phase
  regardless of what Probe-gating did.
- **Disposition:** `dead-with-no-replacement`. It directly asserted the
  inference D-06 deletes ("production mode ⇒ auditor", the unconditional
  security-wording ⇒ gatekeeper rule, and the always-required Watcher) — the
  exact "'Add a CSV export' summons a security auditor" failure mode the
  ruling exists to stop.
- **Removed in:** Phase 194 Plan 02.

### `TestGatekeeperNeedsASecuritySignal` (function in `cmd/queen_probe_gating_test.go`)

- **Original path:** `cmd/queen_probe_gating_test.go` (single function
  removed; file survives).
- **What it covered:** that Gatekeeper was required by
  `queenBuildSafetyRequiredCastes` only when a phase carried high risk or
  matched `queenPhaseHasSecuritySignal`'s keyword list, tested across five
  fixtures (CSV export, credential rotation, auth, final sign-off, release).
- **Disposition:** `recovered-by:cmd/queen_forced_reviewer_test.go
  (TestReviewerForcedOnlyByNamedRisk)`. The claim survives in changed form —
  a reviewer forced only by a named risk signal — but the mechanism and
  boundary both moved (D-05): forcing now happens once, at the `continue`
  step, off the five-signal table in `cmd/queen_risk_signals.go`, not at
  `build` off a keyword list `queenPhaseHasSecuritySignal` no longer exists
  to hold. The replacement test asserts on the real continue dispatch list,
  per this repo's own established pattern (assert the spawn list, not the
  decision record), which this retired test did not.
- **Removed in:** Phase 194 Plan 02.

### `TestHighRiskPhaseKeepsBothReviewers` (function in `cmd/queen_probe_gating_test.go`)

- **Original path:** `cmd/queen_probe_gating_test.go` (single function
  removed; file survives).
- **What it covered:** that a phase classified `phaseRiskLevel == "high"`
  kept both Auditor and Gatekeeper (plus Watcher) required at build,
  regardless of its own wording.
- **Disposition:** `dead-with-no-replacement`. D-04
  (`.planning/decisions/2026-08-22-queen-decides-program-checks.md`) replaces
  "both reviewers on high risk" with one reviewer per named signal —
  Gatekeeper for credentials/auth, payments, release sign-off; Auditor for
  data deletion and migrations — never both from a single risk-level
  computation. The plan's own fixture ("Rework the permissions model") names
  no signal in the new table, so a rewritten version would have to assert a
  rule that no longer exists; retiring it was the documented choice over
  rewriting (194-02-PLAN.md flagged judgement call).
- **Removed in:** Phase 194 Plan 02.

### `TestQueenOrchestratePreservesSafetyCastes` (function in `cmd/caste_relevance_test.go`)

- **Original path:** `cmd/caste_relevance_test.go` (single function removed;
  file survives).
- **What it covered:** the full-pipeline (`queenOrchestrate`, candidate
  scoring + budget) version of the same claim as
  `TestSafetyCastesSurviveProbeGating` — Builder, Watcher, Probe, Gatekeeper
  and Auditor all present on security/release/final-review production
  phases at build.
- **Disposition:** `dead-with-no-replacement`, ruling D11. Watcher and Probe
  are no longer required at build under any condition, and Auditor/Gatekeeper
  are no longer inferred from mode or blast-radius wording at build — the
  equivalent "cannot be dropped" behaviour for a named-risk signal now lives
  at `continue` only (D-05).
- **Removed in:** Phase 194 Plan 02.

### `TestQueenSpawnBudgetDecisionsPreservesSafetyCastesUnderPressure` (function in `cmd/caste_relevance_test.go`)

- **Original path:** `cmd/caste_relevance_test.go` (single function removed;
  its helper `budgetPressureDispatches` removed with it; file survives).
- **What it covered:** the same "safety castes survive" claim under
  synthetic budget pressure (a 12-candidate dispatch list, all scored below
  the required castes) for security/release/final-review phase fixtures.
- **Disposition:** `dead-with-no-replacement`, ruling D11 — same reasoning as
  `TestQueenOrchestratePreservesSafetyCastes`, with budget pressure added.
  The budget-pressure survival property this plan actually keeps is proven
  at `continue` by `unionForcedContinueReviewers`
  (`TestForcedReviewerCrossesTheBuildContinueBoundary`,
  `cmd/queen_forced_reviewer_test.go`, plan 194-01), which unions a forced
  reviewer in AFTER budget trimming so no trim can drop it.
- **Removed in:** Phase 194 Plan 02.

### `TestCodexBuildPlanOnlySpawnBudgetPreservesSafetyCastesUnderLightAndHeavy` (function in `cmd/codex_build_test.go`)

- **Original path:** `cmd/codex_build_test.go` (single function removed;
  file survives).
- **What it covered:** the manifest-level (`dispatch_manifest.dispatches` +
  `queen_execution_policy.spawn_budget`) version of the same claim, checked
  at both `--light` and `--heavy` for a security-hardening and a
  final-review production phase.
- **Disposition:** `dead-with-no-replacement`, ruling D11. Same reasoning as
  the lower-level tests above — the manifest's `required_castes` for build is
  now just `[builder]`.
- **Removed in:** Phase 194 Plan 02.

### `TestCodexBuildPlanOnlyPhaseFiveSafetyVerificationKeepsRequiredCastes` (function in `cmd/codex_build_test.go`)

- **Original path:** `cmd/codex_build_test.go` (single function removed;
  file survives).
- **What it covered:** the same manifest-level claim for a single
  release/security/final-safeguard phase fixture at `--heavy`, added to
  guard a specific historical regression ("adaptive pruning weakens state,
  security, release, or final safeguards").
- **Disposition:** `dead-with-no-replacement`, ruling D11 — same reasoning.
  The regression it guarded against (pruning silently dropping a forced
  reviewer) is now guarded at `continue` by
  `TestForcedReviewerCrossesTheBuildContinueBoundary`
  (`cmd/queen_forced_reviewer_test.go`, plan 194-01).
- **Removed in:** Phase 194 Plan 02.

### `TestTeamCheckinFallsBackToRosterProduces` (function in `cmd/ceremony_team_checkin_test.go`)

- **Original path:** `cmd/ceremony_team_checkin_test.go` (single function
  removed; file survives).
- **What it covered:** that when the spawn budget carried no per-caste
  rationale for a worker, the check-in card fell back to the roster's
  generic "produces" prose so no worker line was ever reason-less.
- **Disposition:** `recovered-by:cmd/ceremony_team_checkin_test.go
  (TestTeamCheckinNeverShowsAGenericBlurbAsAReason)`. The retired test
  asserted exactly the behaviour D-09 forbids: a generic caste description
  standing in for a per-phase reason reads as a justification for sending
  the worker and is not one. The replacement test inverts the assertion —
  the roster description must NOT appear in the reason slot — and pins the
  roster prose to a separately labelled `what_it_does` slot instead. A
  renamed test whose assertion inverted is a removal, per RETIRE-04.
- **Removed in:** Phase 194 Plan 04.
