# Ship Progress — restoration & hardening loop

Working memory for the Ralph restoration loop. Read first, update every round.
Branch: oracle-reinstate. Promise fires only when every item below is DONE and
the full verify gate is green in the same round.

## Item 1 — LIFECYCLE PROOF: **DONE** (round 1)

Audited WORKFLOW-01..09 + RUNTIME-01/02 for executable evidence; ticked each in
.planning/PROJECT.md with the proving test's name. Findings:

- All nine workflows already had real end-to-end evidence (the repo's phases
  165/172 did the restoration; the requirements were simply never ticked):
  colonize `TestColonizeWritesSurveyArtifactsAndUpdatesState`; plan
  `TestFullLifecycleInDownstreamRepo` + `TestPlanningLoopOptionsClampAndEvaluateStopReasons`;
  build + continue + seal + entomb `TestFullLifecycleInDownstreamRepo` (+
  `TestCLIInterruptedBuildResumesThroughForceRedispatch`,
  `TestCLIContinueEnforcesFreshCriterionEvidence`,
  `TestCLICompiledInstallToSealJourney`); run
  `TestRunCompatibilityExecutesSinglePhase`; swarm
  `TestSwarmDestroyRunsWorkerWavesAndReturnsStructuredResult`.
- RUNTIME-01 was a REAL BUG, found and fixed this round: init cleared only
  session.json — pending-decisions.json, assumptions.json and
  handoffs/worker-handoffs.json all survived re-init and are injected into the
  next colony's worker prompts (CLARIFIED INTENT + Previous Worker Handoffs).
  Failing test written first: `TestInitClearsPriorColonyDecisionResidue`
  (cmd/init_residue_test.go); fix in cmd/init_cmd.go (~line 167).
- RUNTIME-02 already held: `TestMergeExternalBuildResults_RejectsMissingResult`
  proves an unreported worker blocks finalize with worker.result_missing.
- NOT in scope of item 1, left unticked: CATALOG-01/02, TEST-01/02,
  QUEEN-01/02, PROOF-01 (PROOF-01 needs a real downstream repo run — operator).

## Item 2 — INIT HARDENING (field report): **DONE** (round 1)

Spec: .planning/field-reports/2026-08-16-init-obsidian-vault.md. Delivered:
- knowledge_base classification in cmd/init_research.go — markdown/source
  counting in the walk, .obsidian//.logseq/ detection, only when zero
  languages; suppresses CI/LICENSE/README/formatter/no-docs pheromones; risks
  swap to content-loss/broken-links; charter describes note counts.
- "A unknown project" filler fixed; all-empty scans emit one honest line.
- Low-signal branch in init.yaml + all three init wrapper mirrors (.opencode
  edited by hand — its content differs).
- .aether/WHAT-IS-THIS.md durable-state marker written by
  ensureRepoLocalScaffold (cmd/platform_sync.go).
Tests: TestObsidianVaultClassifiesAsKnowledgeBase,
TestMarkdownOnlyRepoClassifiesAsKnowledgeBase,
TestKnowledgeRepoGetsNoCodeHousekeepingPheromones,
TestCodeRepoNeverClassifiesAsKnowledgeBase,
TestUnknownProjectCharterNeverPrintsBrokenFiller,
TestKnowledgeRepoRisksAreAboutContent,
TestAetherScaffoldWritesDurableStateMarker, TestInitWrappersCarryLowSignalBranch
(cmd/init_research_knowledge_test.go). All green round 1.

## Item 3 — TIE IT TOGETHER (Phase 168 spirit + Phase 175): **DONE** (round 1)

- Queen's team choice rendered at the Dispatch stage from the manifest's
  SelectedReasons/PrunedReasons/PreservedCastes (renderQueenTeamChoice,
  cmd/codex_visuals.go), wired into both build render paths via
  queenPolicyFromResult. One clause per selected caste; not-called castes are
  ONE short clause (max 3 named + count); safety restorations named with
  reason ("Kept by safety policy: ...").
- Run summary distinguishes finding vs clean: completed workers render
  "— flagged N issue(s)" or "— nothing to flag" (writeDispatchExecutionStatus).
- Colony health line in `aether status` from the extracted
  computeColonyVitalSigns (cmd/memory_details.go) — surfaced, not
  reimplemented. Next-step guidance was already wired (guided actions +
  renderNextUp on every lifecycle visual); verified, not duplicated.
Tests (cmd/queen_team_choice_test.go): TestQueenTeamChoiceRendersManifestRationaleVerbatim
(anti-hardcode fence — blanking the manifest blanks the render),
TestNotCalledCastesRenderAsOneShortClause, TestLightBuildOfProductionPhaseShowsSafetyRestorationLine
(real contract builder), TestBuildVisualCarriesQueenTeamChoice,
TestRunSummaryDistinguishesFindingFromCleanWorkers,
TestStatusHealthLineReflectsComputedVitals.

## Item 4 — ORPHAN RECLAMATION: **not started**

Highest-value entries in cmd/testdata/orphan_allowlist.json: wire with a test,
retire per existing pattern, or record dated disposition. Shrink, never grow.

## Item 5 — RELEASE READINESS: **not started**

Version bump + CHANGELOG.md + goreleaser check + aether integrity. NO publish,
NO push.

## Verify gate (must be green in the SAME round as the promise)

go vet ./... && go test ./... -count=1 && goreleaser check && aether integrity

## Standing landmines learned

- Golden refreshers: TestAuditCatalogGolden / TestRegressionSnapshot /
  TestPlatformParityGolden take -update-golden.
- .opencode wrapper copies differ from .claude — edit, never blind-copy.
- .planning is gitignored-but-tracked: git add -f for new files.
- pkg/codex has one known-flaky heartbeat test — rerun once before believing a
  failure there.
