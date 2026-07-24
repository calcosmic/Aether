# Behaviour Extraction Audit

> **Version:** v1.24
> **Last Updated:** 2026-05-22
> **Applies to:** Phases 152-159
> **Purpose:** Machine-readable inventory of every Go symbol containing agent/prompt/phase/skill/memory/ritual/ceremony logic, classified by disposition for Phase 155 (Go Boundary Refactor).

---

## The Hard Rule

**Compiled code may execute behaviour, but editable assets must define behaviour.**

This means:
- Go can run the logic, but it must not be the only place where that logic's rules live.
- If a human cannot open a text file and change what an agent does, where it is routed, or what it is told, then the architecture has leaked behaviour into compiled code.

For dummies: The engine (Go) can drive the car, but the map, the driver instructions, and the paint job must be things you can edit with a text editor — not hidden inside the engine block.

---

## Summary

| Classification | Count | Description |
|----------------|-------|-------------|
| KEEP_IN_GO | 42 | Pure runtime spine logic (state mutation, file locking, CLI parsing, event bus plumbing, install/update/publish) |
| MOVE_TO_TS | 8 | Orchestration logic that should live in TypeScript control plane (agent selection, phase sequencing, prompt assembly) |
| MOVE_TO_YAML | 6 | Structured definitions that fit YAML (agent specs, phase specs, policy rules) |
| MOVE_TO_MARKDOWN | 14 | Narrative/ceremony content that fits Markdown (prompt text, playbook content, ceremony templates) |
| MOVE_TO_JSON | 4 | Machine-readable config or state schemas (event schemas, memory schemas) |
| DELETE | 2 | Dead code or behaviour already superseded |
| MIXED | 18 | File contains both spine logic and behaviour strings (function-level notes below) |
| UNKNOWN | 0 | Cannot determine without deeper analysis — flag for human review |
| **Total** | **94** | |

---

## Detailed Audit Table

| File | Symbol | Classification | Target Location | Notes |
|------|--------|----------------|-----------------|-------|
| `cmd/alias_cmds.go` | `init` | MIXED | `.aether/config/aliases.yaml` | Registers alias commands (spine) but aliases themselves are behaviour definitions |
| `cmd/assumptions.go` | `buildAssumptionFeedback`, `buildAssumptionFocus`, `buildIntegrationAssumption`, `buildScopeAssumption`, `buildSurfaceAssumption`, `buildVerificationAssumption`, `renderAssumptionListVisual`, `renderAssumptionsAnalyzeVisual`, `renderAssumptionValidateVisual`, `runAssumptionList`, `runAssumptionsAnalyze`, `synthesizeAssumptions` | MIXED | `colony/ceremony/assumptions.md` | Visual rendering and assumption synthesis are ceremony behaviour; command registration is spine |
| `cmd/audit_catalog.go` | `renderAuditCatalogVisual` | MOVE_TO_MARKDOWN | `colony/ceremony/audit-catalog.md` | Pure visual rendering of audit catalog |
| `cmd/autopilot.go` | `init`, `normalizeAutopilotPhaseStatus` | MIXED | `colony/policies/autopilot.yaml` | Autopilot state machine is orchestration behaviour; command wiring is spine |
| `cmd/binary_download.go` | package-level | KEEP_IN_GO | — | Binary download plumbing — pure runtime |
| `cmd/build_flow_cmds.go` | `init`, `nextUpSuggestionsForState` | MIXED | `colony/playbooks/build-flow.md` | Build flow orchestration is behaviour; CLI wiring is spine |
| `cmd/build_playbook_context.go` | `renderBuildPlaybookContext` | MOVE_TO_MARKDOWN | `colony/playbooks/build-context.md` | Playbook context rendering — pure behaviour |
| `cmd/build_reconcile.go` | `runBuildReconcile` | KEEP_IN_GO | — | Build reconciliation logic — runtime spine |
| `cmd/caste_relevance.go` | `applySpecialRules`, `casteAllowedForFlow`, `casteRelevanceScore`, `conditionScore`, `FilterCastesByMinScore`, `findProfile`, `HasCaste`, `hasImplementationTask`, `isAlwaysRequired`, `isCasteSuppressed`, `normalizeQueenFlowType`, `queenCandidateDispatches`, `queenOrchestrate`, `spawnThreshold` | MOVE_TO_TS | `control-ts/src/orchestration/caste-relevance.ts` | Caste scoring and orchestration are control plane behaviour |
| `cmd/ceremony_cmd.go` | `ceremonyCasteCounts`, `ceremonyCasteCountSummary`, `ceremonyDispatchesForExecutionWave`, `ceremonyDispatchesFromManifest`, `ceremonyDispatchFromMap`, `ceremonyExecutionPlansFromManifest`, `ceremonyInlineCasteCounts`, `ceremonyPhaseSummariesFromAny`, `ceremonyPhaseSummariesFromColony`, `ceremonyPhaseSummariesFromCompletion`, `ceremonyPlanLabel`, `ceremonyStringMapValue`, `dispatchesForPlan`, `dominantCeremonyCaste`, `enrichCeremonyDispatchesFromManifest`, `extractCeremonyManifest`, `extractCeremonyWorker`, `normalizedCeremonyWorkflow`, `pluralizeCaste`, `readCeremonyJSONFile`, `renderCeremonyCloseout`, `renderCeremonyCloseoutVisual`, `renderCeremonyQueenFrame`, `renderCeremonyQueenSpawnBudget`, `renderCeremonySkillAssignments`, `renderCeremonySpawnPlan`, `renderCeremonySpawnPlanFromFile`, `renderCeremonyWaveStart`, `renderCeremonyWaveStartFromFile`, `renderCeremonyWorkerComplete`, `renderCeremonyWorkerCompleteFromFile`, `renderOldStyleCeremonyHeader`, `skillNamesFromSkillSection`, `writeCeremonyCloseoutNotice`, `writeCeremonyDispatchLine`, `writeCeremonyPhaseLine`, `writeCeremonyPlanSummary`, `writeCeremonyWorkerSummary` | MIXED | `colony/ceremony/*.md` | Ceremony rendering is behaviour; JSON parsing and file I/O are spine |
| `cmd/ceremony_emitter.go` | `ceremonyPayloadForDispatch`, `ceremonyStepCompleted`, `ceremonyStepStatus`, `continueCeremonyWaveForStage`, `currentBuildCeremony`, `emitBuildCeremony`, `emitBuildCeremonyCircuitBreak`, `emitBuildCeremonyPrewave`, `emitBuildCeremonyWaveEnd`, `emitBuildCeremonyWaveStart`, `emitBuildCeremonyWorkerFailed`, `emitBuildCeremonyWorkerFinished`, `emitBuildCeremonyWorkerRunning`, `emitBuildCeremonyWorkerStarting`, `emitBuildCeremonyWorkerTimeout`, `emitColonizeCeremonyDispatchSequence`, `emitContinueCeremonyFlowSequence`, `emitLifecycleCeremony`, `emitLifecycleCeremonySequence`, `emitLoopBreakEvent`, `emitOracleIteration`, `emitOraclePhaseTransition`, `emitPlanCeremonyDispatchSequence`, `emitSkillActivationCeremonies`, `newBuildCeremonyEmitter`, `setActiveBuildCeremony`, `syntheticCeremonyEvent`, `trimCeremonyList`, `trimCeremonyPayload`, `trimCeremonyText` | MIXED | `colony/ceremony/emitter.md` | Ceremony event emission is behaviour; event bus plumbing is spine |
| `cmd/chamber.go` | `init` | KEEP_IN_GO | — | Chamber archive management — runtime spine |
| `cmd/changelog.go` | `init` | KEEP_IN_GO | — | Changelog append — runtime spine |
| `cmd/circuit_breaker.go` | `emitCircuitBreakerNoPeer`, `emitCircuitBreakerRedistributed`, `findSameCastePeer`, `gateRetryKey`, `NewCircuitBreaker` | KEEP_IN_GO | — | Circuit breaker logic — safety-critical runtime |
| `cmd/clash.go` | package-level | KEEP_IN_GO | — | Worktree clash detection — runtime spine |
| `cmd/closeout_cmd.go` | `closeoutNextCommand`, `closeoutWorkerMaps`, `isVerifiedCloseoutWorkerResult`, `normalizeCloseoutWorkerStatus` | KEEP_IN_GO | — | Closeout orchestration — runtime spine |
| `cmd/codegraph_context.go` | `codegraphTextPartsForBuildBrief`, `renderCodegraphContext` | MOVE_TO_MARKDOWN | `colony/playbooks/codegraph-context.md` | Codegraph context rendering — pure behaviour |
| `cmd/codex_build_finalize.go` | `appendRecoveryOutcomesToLog`, `buildExternalBuildRecoveryInstructions`, `buildExternalBuildResultCollectionReport`, `candidateClaimAbsolutePath`, `codexWorkerDispatchesForRecovery`, `findRepoRelativePath`, `findUnambiguousRepoRelativeClaimPath`, `hasCompletedBuilders`, `mergeExternalBuildResults`, `parsePositivePhaseArg`, `persistExternalBuildHandoffs`, `recordExternalBuildSpawnTree`, `repoRelativeClaimPath`, `runCodexBuildFinalize`, `selectExternalBuildResultForDispatch`, `stripWorkerRetrySuffix`, `validateExternalResultIdentity`, `validateExternalWorkerResultClaimPaths` | KEEP_IN_GO | — | Build finalize — runtime spine (state mutation, file I/O) |
| `cmd/codex_build_progress.go` | `buildDispatchActiveSummary`, `buildDispatchResultSummary`, `continueWorkerCloseSummary`, `dispatchResultSummary`, `emitCodexBuildWaveProgress`, `emitCodexBuildWorkerFinished`, `emitCodexBuildWorkerStarted`, `emitCodexDispatchWaveProgress`, `emitCodexDispatchWorkerFinished`, `emitCodexDispatchWorkerRunning`, `emitCodexDispatchWorkerStarted`, `updateCodexBuildDispatchRuntimeStatus`, `workerDispatchSummary` | MIXED | `colony/ceremony/build-progress.md` | Progress rendering is behaviour; dispatch status tracking is spine |
| `cmd/codex_build_worktree.go` | `allocateBuildWorktree`, `applyObservedClaims`, `cleanupBuildWorktrees`, `collectRepoTouchedPaths`, `collectWorktreeTouchedPaths`, `detectOrphanedWorktrees`, `dispatchCodexBuildWorkers`, `dispatchCodexBuildWorkersInRepo`, `invokeCodexWorkerWithRuntimeProgress`, `mergePhaseWorktrees`, `sanitizeWorktreeLabel`, `syncRootRuntimeIntoWorktree` | KEEP_IN_GO | — | Worktree allocation and merge — runtime spine |
| `cmd/codex_build.go` | `applyBuildDispatchPolicyCastes`, `applyBuildTaskStatuses`, `applyCodexBuildState`, `applyPriorCompletedPhaseTaskRepairs`, `attachBuildDispatchContext`, `buildCodexBuildManifest`, `buildDispatchClaimOutputs`, `buildDispatchContractForDispatches`, `buildExecutionOwner`, `buildPlaybooksForDispatch`, `buildWaveExecutionPlan`, `buildWaveExecutionPlans`, `buildWorkerDispatchOptIn`, `canRetryBuiltPhase`, `cleanupStaleBuildAttemptArtifacts`, `codexAgentFileForCaste`, `codexAgentNameForCaste`, `codexBuildDispatchMaps`, `codexBuildSpecialistDispatch`, `codexBuildTaskPlans`, `completedPhaseTaskRepairError`, `effectiveBuildDispatchTimeout`, `ensureUniqueBuildDispatchNames`, `executeCodexBuildDispatches`, `executionReasonForBuildStep`, `executionStrategyForBuildStep`, `expectedDispatchOutcome`, `findDispatchTask`, `findingsInjectionForCaste`, `incompletePhaseTaskSummary`, `nonAssignmentBuildOutputs`, `normalizedDispatchTaskID`, `normalizedDispatchWave`, `phaseNeedsAmbassador`, `phaseTaskIDSet`, `phaseTasksAllCompleted`, `plannedBuildDispatches`, `plannedBuildDispatchesForSelection`, `plannedBuildDispatchesForSelectionWithState`, `queenBuildCasteSet`, `queenBuildFallbackTaskCaste`, `queenBuildPostWaveDispatches`, `queenBuildPreWaveDispatches`, `queenBuildTaskCaste`, `reconcileCompletedBuildTasks`, `reconcilePriorCompletedPhaseTasksForPlanOnly`, `reconcilePriorCompletedPhaseTasksFromTrustedManifests`, `recordCodexBuildDispatches`, `renderCodexBuildWorkerBrief`, `renderCodexBuildWorkerOutcomeReport`, `resolvePheromoneSection`, `resolveSkillSection`, `resolveSkillSectionForWorkflow`, `resolveSkillSectionResult`, `resolveSkillSectionResultForWorkflow`, `resolveWorkerSkillAssignment`, `resolveWorkerSkillAssignmentForWorkflow`, `rollbackCodexBuildFailure`, `runCodexBuild`, `runCodexBuildPlanOnly`, `runCodexBuildPlanOnlyWithOptions`, `runCodexBuildQueenLed`, `runCodexBuildWithOptions`, `runtimeStateSupersededError`, `trustedCompletedPhaseTaskEvidence`, `updateCodexBuildContext`, `validateBuildManifestTaskSetForPhase`, `validateCodexBuildState`, `validateRuntimeStateMatchesExpected`, `validateRuntimeStateStillCurrent`, `validateSelectedBuildTasks`, `validateTrustedCompletedPhaseManifest`, `writeCodexBuildArtifacts`, `writeCodexBuildClaims`, `writeCodexBuildOutcomeReports` | MIXED | `colony/playbooks/build.md`, `control-ts/src/orchestration/build.ts` | Build orchestration is control plane behaviour; state mutation and file I/O are spine |
| `cmd/codex_colonize_finalize.go` | `recordExternalSurveySpawnTree`, `runCodexColonizeFinalize` | KEEP_IN_GO | — | Colonize finalize — runtime spine |
| `cmd/codex_colonize.go` | `applySurveyDispatchResult`, `buildCodexColonizeManifest`, `bulletList`, `dispatchRealSurveyors`, `dispatchRealSurveyorsWithTimeout`, `isPlatformGuidanceFile`, `plannedSurveyors`, `queenSurveyorSpecs`, `renderSurveyDoc`, `runCodexColonizePlanOnly`, `runCodexColonizeWithOptions`, `surveyDispatchFromSpec`, `surveyDispatchResultHasIdentity`, `surveyDispatchResultMatches`, `surveyorDispatchMaps`, `updateSurveyState` | MIXED | `colony/playbooks/colonize.md` | Surveyor dispatch orchestration is behaviour; state mutation is spine |
| `cmd/codex_continue_finalize.go` | `actualContinueReviewCastes`, `advanceExternalContinue`, `attachExternalContinueWatcher`, `buildLearningContent`, `continueReviewCastesArePlannedSubset`, `expectedContinueReviewCastes`, `externalContinueReviewReport`, `finalizeBlockedExternalContinue`, `mergeExternalContinueResults`, `persistExternalContinueHandoffs`, `recordExternalContinueWorkerFlow`, `renderContinueWorkerOutcomeReport`, `runCodexContinueFinalize`, `validateExternalContinueIdentity`, `validateExternalContinueState`, `writeCodexContinueWorkerOutcomeReports` | KEEP_IN_GO | — | Continue finalize — runtime spine |
| `cmd/codex_continue_plan.go` | `continuePlanArtifactsPath`, `continuePlanOnlySourceCommand`, `plannedExternalContinueDispatches`, `runCodexContinuePlanOnly`, `runCodexContinueVerificationSnapshot` | MIXED | `colony/playbooks/continue.md` | Continue plan orchestration is behaviour; state mutation is spine |
| `cmd/codex_continue.go` | `applyCodexContinueWorkerClosures`, `assessCodexContinue`, `buildContinueReconcileCommand`, `buildContinueReconcileFlagSuffix`, `buildContinueTimeoutRecoveryCommand`, `buildContinueVerificationTimeoutRecoveryCommand`, `buildForceRedispatchCommand`, `buildSkipPhaseCommand`, `buildTargetedRedispatchCommand`, `classifyContinueTaskAssessment`, `cleanupStaleContinueReports`, `closedWorkerNames`, `continueBlockersContainWatcherFailure`, `continueBlockersContainWorkerTimeout`, `continueDeterministicVerificationFlowStep`, `continueHousekeepingFlowStep`, `continueHousekeepingSummary`, `continueNextCommandForBlocked`, `continueOptionsMatchCurrent`, `continueOptionsToJSON`, `continueReviewFlowSummary`, `continueReviewFlowTask`, `continueReviewSkippedFlowStep`, `continueReviewSkippedSummary`, `continueReviewSpecForCaste`, `continueReviewStatusBlocks`, `continueReviewStepBlocks`, `continueReviewTaskForCaste`, `continueReviewWorkerFlowSteps`, `continueSkippedWatcherFlowStep`, `continueSupersededResult`, `continueWatcherDefaultSummary`, `continueWatcherFlowSummary`, `continueWatcherHostBoundarySkipSummary`, `continueWatcherResultSummary`, `continueWorkerFlowEnvironmentBlocked`, `continueWorkerFlowEvents`, `continueWorkerFlowForVerification`, `continueWorkerFlowIsDeterministicVerification`, `continueWorkerFlowLogSummary`, `continueWorkerFlowStatus`, `continueWorkerFlowTask`, `continueWorkerFlowWithWatcher`, `emitContinueVerificationStart`, `emptyClaimsFailureSummary`, `environmentBlockedWatcher`, `evaluateContinueWatcherVerification`, `getWatcherFailureCount`, `incrementWatcherFailureCount`, `isCodexWorkerAvailable`, `isEnvironmentBlockedWatcher`, `loadCodexContinueManifest`, `loadLastContinueOptions`, `manifestRequiresBuilderClaims`, `manifestTaskSetBlockedResult`, `missingBuildPacketBlockedResult`, `missingClaimsSummary`, `plannedCodexContinueClosedWorkers`, `plannedContinueReviewDispatches`, `plannedContinueWatcherDispatch`, `queenContinueDispatches`, `queenContinueHasCaste`, `queenContinueReviewSpecs`, `reconcileContinueCompletedBuildTasks`, `recordBlockedContinueWorkerFlow`, `recordContinueWorkerFlow`, `renderCodexContinueReviewBrief`, `renderCodexContinueWatcherBrief`, `resetWatcherFailureCount`, `resolveCodexVerificationCommands`, `runCodexContinue`, `runCodexContinueGates`, `runCodexContinueReview`, `runCodexContinueVerification`, `runCodexContinueWatcherVerification`, `trimBriefList`, `updateCodexContinueContext`, `validateContinueReconcileTasks`, `verifyCodexBuildClaims` | MIXED | `colony/playbooks/continue.md`, `control-ts/src/orchestration/continue.ts` | Continue orchestration is control plane behaviour; state mutation is spine |
| `cmd/codex_dispatch_contract.go` | `appendHandoffList`, `buildQueenSpawnBudgetContract`, `buildToWorkerDispatches`, `buildWorkerHandoffRecord`, `casteDispatchSummary`, `concreteDispatchCasteSummary`, `effectiveContinueVerificationTimeout`, `effectivePlanningDispatchTimeout`, `effectiveWave`, `enrichQueenExecutionPolicyWithSpawnBudget`, `loadWorkerHandoffRecords`, `maxDuration`, `persistDispatchWorkerHandoff`, `planningDispatchContractForDispatches`, `planningDispatchContractWithTimeout`, `pruneWorkerHandoffRecords`, `recommendQueenExecutionPolicy`, `recommendQueenWorkflowProfile`, `renderDispatchContract`, `renderWorkerHandoffSection`, `surveyDispatchContractWithTimeout`, `verificationStatusForWorkerStatus`, `workerHandoffEmpty`, `workflowProfileContract` | MIXED | `colony/policies/dispatch-contract.yaml` | Dispatch contract rendering is behaviour; handoff persistence is spine |
| `cmd/codex_plan_finalize.go` | `activePlanFinalizeFailureFlag`, `buildablePlanTaskCount`, `loadPlanFinalizeFlagsFile`, `planningDispatchMaps`, `recordExternalPlanSpawnTree`, `routeSetterPhasePlan`, `runCodexPlanFinalize`, `synthesizedPlanDispatches`, `validateClaimedPlanArtifactFreshness`, `validateExternalPlanIdentity`, `validateExternalPlanState` | KEEP_IN_GO | — | Plan finalize — runtime spine |
| `cmd/codex_plan.go` | `activeRedirectPlanConstraints`, `attachPlanningDispatchSkillAssignments`, `buildWorkerPlanPhases`, `clearFallbackPhaseResearchArtifacts`, `clearFallbackPlanningArtifacts`, `convertPlanningDispatchResults`, `dispatchRealPlanningWorkers`, `dispatchRealPlanningWorkersWithTimeout`, `firstBuildablePhase`, `firstKnownTaskIDExample`, `isAetherOrchestrationGoal`, `knownWorkerPlanTaskIDs`, `limitScoutFindings`, `loadWorkerPlanArtifact`, `normalizeScoutPlanningReport`, `normalizeWorkerPlanArtifactDependencies`, `normalizeWorkerPlanTaskDependencies`, `plannedPlanningWorkers`, `plannedPlanningWorkersForGoal`, `planningDispatchByCaste`, `planningDispatchIndexByCaste`, `planningStageForCaste`, `planningTemplates`, `planningWorkerSpecForCaste`, `planningWorkerSpecsForGoal`, `removeFallbackArtifact`, `renderPhasePlanSchemaGuidance`, `renderPlanningWorkerBrief`, `renderScoutPlanningGuidance`, `resolvePlanningDepth`, `resolvePlanningDepthSmart`, `runCodexPlanAgentDelegate`, `runCodexPlanPlanOnly`, `runCodexPlanRepairArtifact`, `runCodexPlanWithOptions`, `scoutReportForPlanningDispatches`, `scoutReportFromWorkerResult`, `scoutReportHasContent`, `synthesizeRouteSetterPlan`, `synthesizeScoutPlanningReport`, `unknownPhasePlanDependencyError`, `workerPlanTaskIDs`, `writePhaseResearchArtifacts`, `writePlanningScoutArtifact`, `writeRouteSetterArtifact`, `writeWorkerPlanArtifact` | MIXED | `colony/playbooks/plan.md`, `control-ts/src/orchestration/plan.ts` | Plan orchestration is control plane behaviour; artifact I/O is spine |
| `cmd/codex_project_docs.go` | `isAetherManagedAgentsDoc`, `platformRestartTargets`, `syncProjectDocs` | KEEP_IN_GO | — | Project docs sync — runtime spine |
| `cmd/codex_visuals.go` | `buildCasteKeywordWords`, `buildCasteWordMatches`, `buildExecutionPlanLabel`, `casteANSIColor`, `casteEmoji`, `casteIdentity`, `casteLabel`, `classifyCommandCeremonyLevel`, `colorizeCaste`, `commandEmoji`, `deterministicAntName`, `filterBuildDispatches`, `flagEntriesValue`, `hasRealExecutionData`, `hasRealPlanningExecutionData`, `normalizeCasteKey`, `parsePlanningDispatchMaps`, `parseSurveyorMaps`, `phaseSliceValue`, `renderAetherWordmark`, `renderArtifactsSection`, `renderBanner`, `renderBinaryActionNextUp`, `renderBinaryActionVisual`, `renderBuildDispatchPreview`, `renderBuildFinalizeVisual`, `renderBuildPlanOnlyVisual`, `renderBuildVisual`, `renderBuildVisualWithDispatches`, `renderCharterDisplay`, `renderCloseoutCompletionSection`, `renderCloseoutVisual`, `renderColonizeDispatchPreview`, `renderColonizeVisual`, `renderContinueBlockedVisual`, `renderContinueGateSummaryMap`, `renderContinuePlanOnlyVisual`, `renderContinueVerificationSummaryMap`, `renderContinueVisual`, `renderContinueWorkerFlowLine`, `renderContinueWorkerFlowMap`, `renderContinueWorkerFlowValue`, `renderExportSignalsVisual`, `renderFlagActionVisual`, `renderFlagsVisual`, `renderHistoryVisual`, `renderImportSignalsVisual`, `renderIndentedList`, `renderInitVisual`, `renderInstallPaths`, `renderInstallVisual`, `renderNextUp`, `renderPatrolVisual`, `renderPauseVisual`, `renderPhaseVisual`, `renderPlanDispatchPreview`, `renderPlanningWorkerResults`, `renderPlanVisual`, `renderProgressSummary`, `renderQueenActionVisual`, `renderReferenceIndexVisual`, `renderReferenceListVisual`, `renderReferenceMatchVisual`, `renderResearchDisplay`, `renderResumeVisual`, `renderReviewDepthLine`, `renderReviewDepthLineWithReason`, `renderSealVisual`, `renderSetupVisual`, `renderShelfActionVisual`, `renderShelfListVisual`, `renderSignalVisual`, `renderSmartDepthReason`, `renderSpawnPlan`, `renderSpawnPlanForDispatches`, `renderStageMarker`, `renderStalePublishBanner`, `renderSurveyorResults`, `renderSyncSummary`, `renderTunnelsCompareVisual`, `renderTunnelsDetailVisual`, `renderTunnelsImportVisual`, `renderTunnelsListVisual`, `renderUpdatePaths`, `renderUpdateVisual`, `shouldUseANSIColors`, `suggestedBuildCaste`, `workflowSuggestionsForState`, `writeDispatchExecutionStatus`, `writeHistoryEntry`, `writePhaseTaskLine`, `writeReferenceSummaryLines` | MOVE_TO_MARKDOWN | `.aether/config/visuals.md` | All visual rendering — pure behaviour. See Mixed Files Deep-Dive below. |
| `cmd/codex_worker_artifacts.go` | `shouldPreserveWorkerArtifact` | KEEP_IN_GO | — | Artifact preservation logic — runtime spine |
| `cmd/codex_worker_cleanup.go` | `cleanupStaleWorkersBeforeDispatch` | KEEP_IN_GO | — | Worker cleanup — runtime spine |
| `cmd/codex_workflow_cmds.go` | `buildSealSummary`, `checkSealBlockers`, `completeSealRuntime`, `countResolvedFlags`, `createPheromoneSignal`, `detectMDSWorkspace`, `init`, `newSignalShortcutCommand`, `renderBlockerSummary`, `resolveWorkerTimeoutFlag`, `synthesizePlan` | MIXED | `colony/ceremony/seal.md`, `colony/policies/signal-rules.yaml` | Seal ceremony and signal creation are behaviour; command wiring is spine |
| `cmd/colony_prime_context.go` | `colonyPrimeLedgerItemFromRanked`, `resolveCodexWorkerContext` | MIXED | `colony/prompts/colony-prime.md` | Context assembly is spine; prompt section text is behaviour |
| `cmd/command_guide.go` | `adaptCommandGuideDefinitionForPlatform`, `commandGuideCatalog`, `commandGuideLiteralCommands`, `intelligentCommandDriftGuards` | MOVE_TO_YAML | `colony/policies/command-guide.yaml` | Command guide definitions — structured behaviour |
| `cmd/command_truth.go` | `deriveMilestoneProgress`, `loadCasteAssignments`, `renderBumpVersionVisual`, `renderMaturityVisual`, `renderMigrateStateVisual`, `renderQuickContextCapsule`, `renderQuickVisual`, `renderVerifyCastesVisual`, `runQuickScout`, `verifyCasteSurfaces` | MIXED | `colony/ceremony/command-truth.md` | Visual rendering is behaviour; caste verification logic is spine |
| `cmd/compatibility_cmds.go` | `buildRunDryRunResult`, `buildRunExecutionResult`, `renderOracleCompatibilityVisual`, `renderRunCompatibilityVisual`, `runCompatibilityAutopilot`, `syncRunAutopilotState`, `writeWatchArtifacts` | MIXED | `colony/ceremony/compatibility.md` | Compatibility visual rendering is behaviour; state sync is spine |
| `cmd/context_update.go` | `appendActivityEntry`, `appendDecisionEntry`, `appendWorkerSpawnEntry`, `markWorkerComplete`, `runContextBuildStart`, `runContextInit`, `runContextSubAction`, `runContextUpdatePhase`, `runContextWorkerComplete`, `runContextWorkerSpawn` | KEEP_IN_GO | — | Context update — runtime spine (state mutation) |
| `cmd/context_weighting.go` | `buildHiveWisdomLines`, `confidenceScoreFromDecisions`, `confidenceScoreFromHive`, `confidenceScoreFromInstinctEntries`, `confidenceScoreFromLearnings`, `confidenceScoreFromLegacyInstincts`, `confidenceScoreFromSignals`, `decisionPhases`, `filterHiveWisdomEntriesByDomain`, `hiveFreshnessScore`, `instinctConfidenceScore`, `latestInstinctFreshness`, `latestPhaseLearningFreshness`, `phaseLearningConfidenceScore`, `phaseLearningPhases`, `phaseScopedRelevance`, `protectedSectionPolicy`, `readHiveWisdomEntries`, `readHiveWisdomEntriesForDomains`, `sectionRelevanceScore` | KEEP_IN_GO | — | Context scoring — runtime spine (algorithmic) |
| `cmd/context.go` | `aetherRootFromDataPath`, `buildContextCapsuleOutput`, `computeEffectiveStrength`, `computeNextAction`, `extractRecentDecisions`, `extractRiskEntries`, `extractSignalTexts`, `lookupPhaseName`, `readHiveWisdom`, `readQUEENMd`, `removePRSection`, `trimSection`, `wordCount` | KEEP_IN_GO | — | Context assembly — runtime spine |
| `cmd/council.go` | `init` | KEEP_IN_GO | — | Council command — runtime spine |
| `cmd/curation_cmds.go` | `init` | KEEP_IN_GO | — | Curation commands — runtime spine |
| `cmd/discuss_analyze.go` | `buildAnalyzeDeploymentQuestion` | MOVE_TO_MARKDOWN | `colony/prompts/discuss.md` | Deployment question text — behaviour |
| `cmd/discuss.go` | `activeSignalTexts`, `boundedClarifiedIntentPromptLine`, `buildDiscussVerificationQuestion`, `clarifiedIntentPromptEntries`, `clarifiedIntentPromptRenderResult`, `clarifiedIntentPromptRenderResultForScope`, `nextAfterClarificationResolution`, `orchestratorBoundaryAfterDiscussCommand`, `renderBoundedClarifiedIntentPromptLines`, `renderClarifiedIntentPromptEntries`, `renderClarifiedIntentPromptEntriesWithIntegrity`, `renderDiscussVisual` | MIXED | `colony/prompts/discuss.md` | Prompt text rendering is behaviour; verification logic is spine |
| `cmd/dispatch_platform_helpers.go` | `defaultAvailabilityCause`, `dispatchAgentPath`, `dispatchAgentPathForPlatform`, `dispatchAvailabilityDiagnosticMessage`, `dispatchAvailabilityMessage`, `dispatchAvailabilityNextAction`, `dispatchProviderDiagnostics`, `dispatchUnavailableError`, `formatDispatchAvailabilityStatus`, `providerProbeCommand` | KEEP_IN_GO | — | Platform dispatch helpers — runtime spine |
| `cmd/dispatch_runtime.go` | `dispatchBatchByWaveWithVisuals`, `dispatchLifecycleResult`, `dispatchLifecycleSummary`, `isWorkerProviderPreflightError`, `preflightWorkerProvider`, `runtimeVisualDispatchObserver`, `spawnTreeDispatchObserver` | KEEP_IN_GO | — | Dispatch runtime — runtime spine |
| `cmd/emoji_audit.go` | `countEmojisInString`, `emojiAuditFile`, `init` | KEEP_IN_GO | — | Emoji audit — runtime spine |
| `cmd/entomb_cmd.go` | `chamberStateMemorySummary`, `clearActiveColonyRuntimeFiles`, `compareChambers`, `copyEntombArtifacts`, `countTotalPlans`, `entombTempSweep`, `errString`, `exportArchiveXMLToFile`, `extractNearMissInstincts`, `extractPheromonesFromArchiveXML`, `importSignalsFromChamber`, `renderEntombVisual`, `resetColonyStateForEntomb`, `sanitizeChamberGoal`, `verifyEntombedChamber`, `writeEntombManifest`, `writeEntombRecoveryDocs` | KEEP_IN_GO | — | Entomb — runtime spine |
| `cmd/error_cmds.go` | `init` | KEEP_IN_GO | — | Error commands — runtime spine |
| `cmd/exchange.go` | `importPheromonesData`, `runExportArchive`, `runExportPheromones`, `runExportWisdom`, `runImportPheromones`, `runImportWisdom`, `sanitizeImportedSignalPrefix` | KEEP_IN_GO | — | Exchange import/export — runtime spine |
| `cmd/finalizer_completion_contract.go` | `finalizerCompletionContractStep` | KEEP_IN_GO | — | Completion contract — runtime spine |
| `cmd/fixer_dispatch.go` | `checkAttemptCap`, `dispatchFixer`, `incrementUnblockAttempts`, `isFixerDispatchBlocked`, `readGateResultsPhase`, `readUnblockAttempts`, `recordFixerFailure`, `resolveFixedGates` | KEEP_IN_GO | — | Fixer dispatch — runtime spine |
| `cmd/flag_cmds.go` | `init`, `matchingEnvironmentBlockedLaunchFlag`, `sameFlagPhase` | KEEP_IN_GO | — | Flag commands — runtime spine |
| `cmd/flags.go` | `filterFlags`, `init`, `renderFlagsTable` | MIXED | `colony/ceremony/flags.md` | Flag table rendering is behaviour; flag filtering is spine |
| `cmd/gate.go` | `annotateGateResult`, `autoResolveDepthMultiplier`, `autoResolveSoftBlockGates`, `checkAllTasksCompleted`, `checkPhaseAdvance`, `checkPhaseBuildable`, `checkPhaseBuilt`, `gateRecoveryTemplate`, `gateResultsRead`, `gateResultsReadPhase`, `gateResultsWritePhase`, `init`, `isGateSkippedForMode`, `isHardBlockGate`, `phaseModeAwareGateClassify`, `runPreBuildGates`, `runPreContinueGates` | KEEP_IN_GO | — | Gate logic — runtime spine (safety-critical) |
| `cmd/generate_cmds.go` | package-level | KEEP_IN_GO | — | Generate commands — runtime spine |
| `cmd/graph_consolidation_cmds.go` | `pipelineConfigForStore` | KEEP_IN_GO | — | Graph consolidation — runtime spine |
| `cmd/grave.go` | `init` | KEEP_IN_GO | — | Grave command — runtime spine |
| `cmd/heartbeat_monitor.go` | `cleanupHeartbeatFiles`, `ValidateHeartbeatFile` | KEEP_IN_GO | — | Heartbeat monitor — runtime spine |
| `cmd/helpers.go` | `newLearningValidator`, `renderVisualError` | MIXED | `colony/ceremony/helpers.md` | Visual error rendering is behaviour; validator is spine |
| `cmd/hive_search.go` | `init` | KEEP_IN_GO | — | Hive search — runtime spine |
| `cmd/hive.go` | `hiveConfidenceForRepoCount`, `hiveSourceRepos`, `init`, `promoteToHive`, `reinforceHiveWisdomEntry`, `writeWisdom` | KEEP_IN_GO | — | Hive brain — runtime spine |
| `cmd/hook_cmds.go` | `nextCommandForHookState`, `protectedHookWriteReason`, `redirectWriteReason`, `summarizeHookState` | KEEP_IN_GO | — | Hook commands — runtime spine |
| `cmd/host_cmd.go` | `init` | KEEP_IN_GO | — | Host command — runtime spine |
| `cmd/immune.go` | package-level | KEEP_IN_GO | — | Immune system — runtime spine |
| `cmd/init_ceremony.go` | `createCeremonyColony`, `extractCeremonyResearchData`, `init`, `isTerm`, `promptNumberedChoice`, `promptString`, `runCeremonyResearch`, `runInitCeremony`, `stringOrEmpty`, `synthesizeLaunchBrief` | MIXED | `colony/ceremony/init.md` | Init ceremony text is behaviour; colony creation is spine |
| `cmd/init_cmd.go` | package-level | KEEP_IN_GO | — | Init command — runtime spine |
| `cmd/init_research.go` | `generateCharter`, `generateColonyContextSummary`, `generateKeyRisks`, `generatePheromoneSuggestions`, `joinWithCommaAnd`, `parseGitlabCIDeep`, `parseJenkinsfileDeep` | MOVE_TO_MARKDOWN | `colony/prompts/init-research.md` | Research generation text — behaviour |
| `cmd/install_cmd.go` | `cleanEmptyDirs`, `isInstallPackageDir`, `pathHasExcludedComponent`, `syncPathHasComponent`, `syncPathProtected` | KEEP_IN_GO | — | Install command — runtime spine |
| `cmd/instinct_runtime.go` | `activeInstinctCount`, `instinctApplicationStats`, `instinctEntryToLegacy`, `loadActiveInstinctEntriesFromStore`, `loadInstinctFileOrEmpty`, `loadRecentRuntimeInstincts`, `loadRuntimeInstincts`, `parseInstinctTimestamp`, `rankedInstinctEntries`, `recentInstinctEntries`, `sortedActiveInstinctEntries` | KEEP_IN_GO | — | Instinct runtime — runtime spine |
| `cmd/instinct.go` | `init` | KEEP_IN_GO | — | Instinct commands — runtime spine |
| `cmd/integrity_cmd.go` | `buildIntegrityVisual`, `checkHubCompanionFiles` | MIXED | `colony/ceremony/integrity.md` | Integrity visual rendering is behaviour; file checking is spine |
| `cmd/internal_cmds.go` | `init` | KEEP_IN_GO | — | Internal commands — runtime spine |
| `cmd/learn_export.go` | `init` | KEEP_IN_GO | — | Learn export — runtime spine |
| `cmd/learning_cmds.go` | `init`, `trimSpace` | KEEP_IN_GO | — | Learning commands — runtime spine |
| `cmd/learning.go` | `init`, `isLearningEnabled` | KEEP_IN_GO | — | Learning logic — runtime spine |
| `cmd/lifecycle_helpers.go` | `allPhasesCompleted`, `colonyNeedsEntomb`, `completedPhaseCount` | KEEP_IN_GO | — | Lifecycle helpers — runtime spine |
| `cmd/maintenance.go` | package-level | DELETE | — | Superseded by signal housekeeping |
| `cmd/medic_auto_spawn.go` | `renderMedicAutoSpawnVisual`, `saveMedicLastScan` | MIXED | `colony/ceremony/medic.md` | Medic visual rendering is behaviour; scan persistence is spine |
| `cmd/medic_ceremony.go` | `checkContextClearGuidance`, `checkEmojiConsistency`, `checkStageMarkers`, `extractEmojisFromMarkdown`, `getCommandEmoji`, `scanCeremonyIntegrity` | MOVE_TO_MARKDOWN | `colony/ceremony/medic.md` | Ceremony integrity checks — behaviour definitions |
| `cmd/medic_cmd.go` | `init`, `performBasicHealthScan`, `renderMedicJSON`, `renderMedicReport`, `renderNoColonyMedicVisual`, `severityColor`, `writeIssueLine` | MIXED | `colony/ceremony/medic.md` | Medic visual rendering is behaviour; health scan is spine |
| `cmd/medic_repair.go` | `repairPheromoneIssues` | KEEP_IN_GO | — | Medic repair — runtime spine |
| `cmd/medic_scanner.go` | `parseTimestamp`, `performHealthScan`, `scanDataFiles`, `scanPheromones` | KEEP_IN_GO | — | Medic scanner — runtime spine |
| `cmd/medic_trace.go` | `computeHealthScore`, `extractTimeline`, `findErrorClusters`, `findStalledPhases`, `findStateGaps`, `generateDiagnosticSuggestions`, `renderTraceDiagnostic` | MIXED | `colony/ceremony/medic.md` | Trace diagnostic rendering is behaviour; analysis is spine |
| `cmd/medic_wrapper.go` | `scanHubPublishIntegrity`, `scanWrapperParity` | KEEP_IN_GO | — | Medic wrapper — runtime spine |
| `cmd/memory_details.go` | `init` | KEEP_IN_GO | — | Memory details — runtime spine |
| `cmd/memory_health.go` | `latestMemoryHealthTimestamp`, `loadMemoryHealthSummary`, `memoryHealthDaysSince` | KEEP_IN_GO | — | Memory health — runtime spine |
| `cmd/midden_cmds.go` | `init` | KEEP_IN_GO | — | Midden commands — runtime spine |
| `cmd/narrator_launcher.go` | `writeNarratorVisualContract` | MOVE_TO_MARKDOWN | `colony/ceremony/narrator.md` | Narrator visual contract — behaviour |
| `cmd/oracle_background_unix.go` | `configureOracleBackgroundProcess` | KEEP_IN_GO | — | Oracle background — runtime spine |
| `cmd/oracle_background_windows.go` | `configureOracleBackgroundProcess` | KEEP_IN_GO | — | Oracle background — runtime spine |
| `cmd/oracle_iterate_cmd.go` | `init`, `loadExternalOracleIterationCompletion`, `loadOracleState`, `oracleStatePath`, `saveOracleState` | KEEP_IN_GO | — | Oracle iterate — runtime spine |
| `cmd/oracle_loop.go` | `appendOracleFinding`, `applyOracleWorkerResponse`, `archiveOracleWorkspace`, `buildBriefInformedQuestions`, `buildOraclePhaseDirective`, `buildOracleQuestions`, `buildOracleRubric`, `buildOracleWorkerConfig`, `buildSynthesizedPrompt`, `clampOracleConfidence`, `collectEvidence`, `compactOracleFocusAreas`, `compactOracleStrings`, `compactOracleText`, `containsAnyOracleKeyword`, `containsOracleIteration`, `copyOracleWorkspaceEntry`, `currentOracleFocusAreas`, `currentOracleRedirectAreas`, `defaultOracleAttemptPolicy`, `detectOracleProjectProfile`, `ensureOracleSource`, `ensureOracleWorkspace`, `escapeOracleTableCell`, `extractKeywordsSet`, `finalizeOracleLoop`, `firstAvailableOracleInvoker`, `formulateOracleBrief`, `hasHardBlocker`, `identifyGaps`, `inferOracleAutoScope`, `inferOracleEvidenceType`, `inferOracleTemplate`, `invokeOracleIteration`, `loadColonyLearnings`, `loadOraclePlanFile`, `loadOracleStateFile`, `loadOracleWorkerResponse`, `loadOracleWorkerResponseForAttempt`, `mergeOracleNotes`, `newDefaultOracleWorkerInvoker`, `nextOraclePhase`, `nextOracleSourceID`, `normalizeOracleWorkerResponse`, `oracleAllowsOpenCodeDispatch`, `oracleBackgroundEnv`, `oracleDetectedPlatform`, `oracleDurationLabel`, `oracleEvidenceStrategyForScope`, `oracleInvokerPlatform`, `oracleJSONObjectCandidates`, `oracleMedicBoundaryNote`, `oracleOverallConfidence`, `oraclePhaseDirective`, `oracleProgressedSince`, `oracleQuestionCounts`, `oracleQuestionLabel`, `oracleReadyForCompletion`, `oracleResponsePath`, `oracleResponseStatusToWorkerStatus`, `oracleRetryableFailure`, `oracleScopeFromOptional`, `oracleStatusResult`, `oracleStopRequested`, `oracleTopicHeadline`, `oracleTopicQuestionLabel`, `oracleTopicSummary`, `oracleTruthyEnv`, `oracleTruthyValue`, `oracleWorkerAttemptTimedOut`, `oracleWorkerConfigOverrides`, `oracleWorkerFailureSummary`, `oracleWorkspacePaths`, `parseOracleMaxIterations`, `parseOracleWorkerResponseFromText`, `recoverStaleOracleController`, `renderOracleContextCapsule`, `renderOracleCurrentGaps`, `renderOracleIterationPreview`, `renderOraclePriorFindings`, `renderOracleRetryPreview`, `resolveOracleDepth`, `resolveOracleScope`, `resolveOracleTemplate`, `resumeOracleLoopCompatibility`, `runOracleCompatibility`, `runOracleIterationAttempt`, `runOracleLoop`, `scoreQuestionImpact`, `selectOracleQuestion`, `selectOracleQuestionSmart`, `snapshotOracleProgress`, `startOracleBackgroundLoop`, `startOracleCompatibility`, `stopOracleCompatibility`, `validateOracleConfidence`, `waitOracleAttemptResult`, `writeArchitectureReviewReport`, `writeBugInvestigationReport`, `writeGenericReport`, `writeOracleDerivedReports`, `writeOracleGapsReport`, `writeOracleIterationArtifact`, `writeOracleLoopMarker`, `writeOraclePlanFile`, `writeOracleResearchPlan`, `writeOracleStateFile`, `writeOracleWorkerResponseFile`, `writeTechEvaluationReport` | MIXED | `colony/playbooks/oracle.md`, `control-ts/src/orchestration/oracle.ts` | Oracle prompt text and rubric are behaviour; workspace management is spine |
| `cmd/oracle_process_unix.go` | `oracleProcessExists`, `oracleProcessTable`, `oracleProcessTree`, `terminateOracleProcessTree` | KEEP_IN_GO | — | Oracle process management — runtime spine |
| `cmd/oracle_process_windows.go` | `oracleProcessExists`, `terminateOracleProcessTree` | KEEP_IN_GO | — | Oracle process management — runtime spine |
| `cmd/orchestrator_boundary_clarification.go` | `materializeOrchestratorBoundaryClarifications`, `normalizeOrchestratorBoundaryPhase`, `normalizeOrchestratorBoundarySourcePart`, `orchestratorBoundaryClarificationSource` | MOVE_TO_MARKDOWN | `colony/prompts/boundary-clarification.md` | Boundary clarification text — behaviour |
| `cmd/orchestrator_boundary_questions.go` | `boundaryPhaseLabel`, `buildBoundaryQuestionCandidates`, `continueBoundaryQuestionCandidates`, `materializeOrchestratorBoundaryQuestions`, `sealBoundaryQuestionCandidates` | MOVE_TO_MARKDOWN | `colony/prompts/boundary-questions.md` | Boundary question text — behaviour |
| `cmd/output_filter.go` | `setBuildVerbose` | KEEP_IN_GO | — | Output filter — runtime spine |
| `cmd/patrol_check.go` | `runJSONValidityCheck`, `runStalePheromonesCheck` | KEEP_IN_GO | — | Patrol check — runtime spine |
| `cmd/pending_decision.go` | `init` | KEEP_IN_GO | — | Pending decision — runtime spine |
| `cmd/phase_skip.go` | `renderSkipPhaseVisual`, `runSkipPhase`, `validateSkipPhaseTarget` | MIXED | `colony/ceremony/phase-skip.md` | Skip phase visual is behaviour; validation is spine |
| `cmd/phase.go` | `buildPhaseResult`, `init` | KEEP_IN_GO | — | Phase commands — runtime spine |
| `cmd/pheromone_display_helpers.go` | `humanizePheromoneDuration`, `remainingSignalDecay`, `signalLifetimeSummary` | KEEP_IN_GO | — | Pheromone display — runtime spine |
| `cmd/pheromone_loader.go` | `extractSignalTextsFrom`, `filterSignalsForPrompt`, `loadPheromones`, `loadPheromonesOnce`, `signalActiveForPrompt` | KEEP_IN_GO | — | Pheromone loader — runtime spine |
| `cmd/pheromone_mgmt.go` | `colonyLifecycleSignalContext`, `init` | MIXED | `colony/policies/pheromone-lifecycle.yaml` | Signal context text is behaviour; management is spine |
| `cmd/pheromone_sync.go` | `contentHashPtr`, `firstScopePtr`, `formatPheromoneSyncSummary`, `loadPheromoneFileWithFallback`, `mergePheromoneFiles`, `mergePheromoneSignalsByID`, `mergePheromoneSignalsByKey`, `newPheromoneStoreForRoot`, `normalizePheromoneFile`, `normalizePheromoneSignal`, `pheromoneSignalKey`, `resolvePheromoneRoot`, `setSignalObservationCount`, `signalMutationAfter`, `signalMutationTimestamp`, `signalObservationCount`, `syncPheromoneStores`, `unionPheromoneTags` | KEEP_IN_GO | — | Pheromone sync — runtime spine |
| `cmd/pheromone_write.go` | `expireSignalsByType`, `init`, `sha256Sum` | KEEP_IN_GO | — | Pheromone write — runtime spine |
| `cmd/pheromones_read.go` | `init` | KEEP_IN_GO | — | Pheromone read — runtime spine |
| `cmd/plan_grounding.go` | `checkPlanGrounding`, `isGroundedTask`, `isResearchPhase` | KEEP_IN_GO | — | Plan grounding — runtime spine |
| `cmd/platform_sync.go` | `codexCommandSkillShims`, `codexSkillShims`, `ensureRepoLocalScaffold`, `installSyncPairs`, `isShippedAetherCodexAgent`, `neverSyncPath`, `platformHomeHubSyncPairs`, `pruneRepoCodexSkillMirror`, `pruneShippedFromUserSkillsDir`, `pruneShippedRepoSkills`, `renderCodexCommandSkillShimBody`, `renderCodexSkillShim`, `repoSyncPairs`, `skillDirDeclaresSource`, `syncCodexSkillShims`, `validateCodexAgentFile`, `validateOpenCodeAgentFile`, `writeCodexCommandSkillList` | KEEP_IN_GO | — | Platform sync — runtime spine |
| `cmd/porter_cmd.go` | `buildPorterReadinessSummary`, `buildPorterVisual`, `formatVersionStatus`, `lastPorterOutputLines` | MIXED | `colony/ceremony/porter.md` | Porter visual rendering is behaviour; readiness logic is spine |
| `cmd/profile.go` | `ensureProfiledDirective`, `promoteProfileDirectives`, `promoteQueenPreferenceDirective`, `renderBehaviorObserveVisual`, `renderProfileReadVisual`, `renderProfileUpdateVisual` | MIXED | `colony/ceremony/profile.md` | Profile visual rendering is behaviour; directive promotion is spine |
| `cmd/prompt_integrity.go` | `colonyPrimeIntegrityRecords`, `emitPromptIntegrityEvents` | KEEP_IN_GO | — | Prompt integrity — runtime spine |
| `cmd/proof_cmd.go` | `buildProofContext`, `buildProofOutput`, `buildProofSkillProof`, `convertCapsuleBlocked`, `convertColonyPrimeLedger`, `proofDispatchesForState`, `renderProofDecisionSection`, `renderProofReasons`, `renderProofSkillEntries`, `renderProofVisual` | MIXED | `colony/ceremony/proof.md` | Proof visual rendering is behaviour; context building is spine |
| `cmd/provenance.go` | `isBuildImplementationWorker`, `traceContinueProvenance`, `validateBuildProvenance`, `validateBuildProvenanceForManifest` | KEEP_IN_GO | — | Provenance tracking — runtime spine |
| `cmd/queen_audit.go` | `consolidateQueenAudit`, `readAuditFile`, `writeAuditFile` | KEEP_IN_GO | — | Queen audit — runtime spine |
| `cmd/queen_decision.go` | `buildRationale`, `buildRecoveryPreview`, `queenDecide`, `queenLogEscalation`, `queenStateRead`, `queenStateWrite` | KEEP_IN_GO | — | Queen decision — runtime spine |
| `cmd/queen_phase_summary.go` | `renderActionsNeeded`, `renderPhaseEndSummary` | MOVE_TO_MARKDOWN | `colony/ceremony/queen-phase-summary.md` | Phase end summary text — behaviour |
| `cmd/queen_spawn_budget.go` | `appendQueenBudgetRationale`, `appendQueenPrunedRationale`, `applyQueenSpawnBudget`, `effectiveQueenPhaseMode`, `queenBuildSafetyRequiredCaste`, `queenBuildSafetyRequiredCastes`, `queenBuildSafetyReviewRequired`, `queenMaxWorkersForBudget`, `queenPhaseLooksDocumentationOrMaintenance`, `queenRequiredCastesForBudget`, `queenSpawnBudgetDecisions`, `queenSpawnBudgetForPhase` | MOVE_TO_TS | `control-ts/src/orchestration/spawn-budget.ts` | Spawn budget orchestration is control plane behaviour |
| `cmd/queen_wave_lifecycle.go` | `queenWaveLifecycle`, `readWaveSummary`, `renderWaveSummaryTable`, `writeWaveSummary` | KEEP_IN_GO | — | Wave lifecycle — runtime spine |
| `cmd/queen.go` | `appendEntriesToQueenSection`, `appendEntryToQueenSection`, `buildCharterLines`, `buildQueenComposeCharterLines`, `hasQueenComposeInput`, `init`, `isEntryInText`, `loadLocalQueenText`, `loadQueenText`, `localQueenPath`, `mapLegacyQueenSection`, `normalizeQueenEntry`, `promoteInstinctLocal`, `queenComposeEntry`, `queenComposeQuestions`, `replaceQueenSection`, `sanitizeQueenInline`, `writeLocalQueenText`, `writeQueenText` | KEEP_IN_GO | — | Queen.md management — runtime spine |
| `cmd/recipes.go` | `renderRecipesVisual` | MOVE_TO_MARKDOWN | `colony/ceremony/recipes.md` | Recipes visual — behaviour |
| `cmd/recover_repair.go` | `dispatchRecoverRepair`, `isDestructiveCategory`, `repairMissingAgentFiles`, `repairPartialPhase`, `repairUnreconciledWorkerChanges` | KEEP_IN_GO | — | Recover repair — runtime spine |
| `cmd/recover_scanner.go` | `performStuckStateScan`, `recoverAgentSurfaces`, `scanBadManifest`, `scanMissingAgentFiles`, `scanMissingBuildPacket`, `scanPartialPhase`, `scanStaleSpawnedWorkers`, `scanUnreconciledWorkerChanges` | KEEP_IN_GO | — | Recover scanner — runtime spine |
| `cmd/recover_visuals.go` | `recoverNextStep`, `renderRecoverDiagnosis`, `renderRepairLog`, `writeRecoverIssueLine` | MOVE_TO_MARKDOWN | `colony/ceremony/recover.md` | Recover visual rendering — behaviour |
| `cmd/recover.go` | `renderNoColonyRecoverVisual` | MOVE_TO_MARKDOWN | `colony/ceremony/recover.md` | Recover visual — behaviour |
| `cmd/recovery_classify.go` | `classifyRuntimeOwnedRecoveryIssue`, `classifyWorkerFailure`, `init`, `matchesAllRecoveryKeywords`, `recoveryLogReadPhase`, `recoveryLogWritePhase` | KEEP_IN_GO | — | Recovery classify — runtime spine |
| `cmd/recovery_engine.go` | `buildVisualRecoveryMenu`, `recoveryCandidates` | MOVE_TO_MARKDOWN | `colony/ceremony/recovery.md` | Recovery menu rendering — behaviour |
| `cmd/recovery_orchestrator.go` | `budgetFromRecoveryLog`, `escalateOutcome`, `fixerDispatchOutcome`, `orchestrateRecovery`, `peerReassignmentOutcome`, `persistBudgetToRecoveryLog`, `recoveryHistorySummary`, `sequenceRecoverable`, `sequenceRequiresAttempt` | KEEP_IN_GO | — | Recovery orchestrator — runtime spine |
| `cmd/recovery_snapshot.go` | `defaultProgressSummary`, `defaultSafeToClear`, `loadActiveRecoveryGuidance`, `nextCommandFromState`, `recoveryPhase`, `renderContextSnapshot`, `renderHandoffSnapshot`, `renderHandoffStateSnapshot`, `sessionActiveTodosFromState` | KEEP_IN_GO | — | Recovery snapshot — runtime spine |
| `cmd/references.go` | `init`, `referenceLibraryRoots`, `resolveReferenceSection`, `resolveReferenceSectionWithOutput`, `scoreReference` | KEEP_IN_GO | — | References — runtime spine |
| `cmd/registry.go` | package-level | KEEP_IN_GO | — | Registry — runtime spine |
| `cmd/result_validation.go` | `validateWorkerResultIdentity` | KEEP_IN_GO | — | Result validation — runtime spine |
| `cmd/review_depth.go` | `chaosShouldRunInLightMode`, `collectPhaseText`, `matchesAnyKeyword`, `phaseHasHeavyKeywords`, `phasePositionLevel`, `phaseRiskLevel`, `resolveEffectiveContinueDepth`, `resolveReviewDepth`, `resolveSmartPlanningDepth`, `resolveSmartVerificationDepth`, `resolveVerificationDepth`, `resolveVerificationDepthSmart` | MOVE_TO_TS | `control-ts/src/orchestration/review-depth.ts` | Review depth orchestration is control plane behaviour |
| `cmd/review_ledger.go` | `agentList`, `init` | KEEP_IN_GO | — | Review ledger — runtime spine |
| `cmd/seal_final_review.go` | `ensureSealFinalReview`, `finalCompletedPhase`, `loadFreshSealFinalReview`, `mergeExternalSealReviewResults`, `persistSealFinalReviewFindings`, `plannedExternalSealReviewDispatches`, `plannedSealFinalReviewDispatches`, `queenSealReviewSpecs`, `renderSealFinalReviewBlockers`, `renderSealFinalReviewBrief`, `renderSealPlanOnlyVisual`, `reviewLedgerHasEquivalentFinding`, `runSealFinalize`, `runSealFinalReview`, `runSealPlanOnly`, `sealFinalReviewFindings`, `sealFinalReviewFlowSummary`, `sealFinalReviewResultMap`, `sealFinalReviewReusableLessons`, `sealFinalReviewSatisfiesGate`, `sealFinalReviewCasteAllowsDomain`, `sealFinalReviewDepthForColony`, `sealFinalReviewDomainForCaste`, `sealFinalReviewFindingBlockingIssues`, `sealFinalReviewRequiredCastes`, `validateSealReady`, `writeSealReusableLessonsToQueen` | MIXED | `colony/playbooks/seal.md`, `control-ts/src/orchestration/seal.ts` | Seal review orchestration is behaviour; state mutation is spine |
| `cmd/serve.go` | `isLocalhost` | KEEP_IN_GO | — | Serve command — runtime spine |
| `cmd/session_cmds.go` | `commandFileMap`, `getGitHEAD`, `rotateSpawnTree` | KEEP_IN_GO | — | Session commands — runtime spine |
| `cmd/session_flow_cmds.go` | `buildHandoffDocument`, `currentOpenTasks`, `detectStaleFocusSignals`, `loadOrCreateSessionSummary`, `parseHandoffPhase`, `parseHandoffTasks`, `restoreStateFromLegacyHandoff`, `resumeStateIsRunnable` | KEEP_IN_GO | — | Session flow — runtime spine |
| `cmd/session.go` | `init` | KEEP_IN_GO | — | Session management — runtime spine |
| `cmd/setup_cmd.go` | package-level | KEEP_IN_GO | — | Setup command — runtime spine |
| `cmd/shelf_cmd.go` | `init` | KEEP_IN_GO | — | Shelf command — runtime spine |
| `cmd/shelf_init.go` | `formatShelfForInit` | KEEP_IN_GO | — | Shelf init — runtime spine |
| `cmd/shelf_seal.go` | `detectExpiredFocusPheromones`, `detectLowConfidenceInstincts`, `detectRecurringRedirects`, `detectShelfCandidates`, `shelfCandidateSummary` | KEEP_IN_GO | — | Shelf seal — runtime spine |
| `cmd/signal_housekeeping.go` | `applySignalHousekeeping`, `countActiveSignals`, `deactivateSignal`, `init`, `phaseCompletionsSince`, `runSignalHousekeepingWithState`, `signalExpiredByTime` | KEEP_IN_GO | — | Signal housekeeping — runtime spine |
| `cmd/skill_curator.go` | `init`, `resolveColonyDBPath`, `resolveSkillBaseDir` | KEEP_IN_GO | — | Skill curator — runtime spine |
| `cmd/skill_lifecycle.go` | `init` | KEEP_IN_GO | — | Skill lifecycle — runtime spine |
| `cmd/skills.go` | `appendWorkspaceSnapshotPath`, `buildFullIndex`, `extractResolvedSkillNames`, `findSkillDirs`, `indexSkillDir`, `init`, `loadSkillIndexOrBuild`, `matchSkills`, `matchSkillsForWorkflow`, `normalizeSkillInjectBudget`, `parseSkillFrontmatter`, `reasonScoreTotal`, `renderSkillInjectResult`, `renderSkillInjectResultWithBudget`, `resolveSkillInjectBudget`, `resolveSkillMatchesForRoot`, `resolveSkillMatchesForRootWithWorkflow`, `resolveSkillMatchInput`, `resolveSkillMatchReasons`, `samePathOrAncestor`, `shouldSkipSkillScanPath`, `skillFileDeclaresSource`, `skillHasAnyReason`, `skillHasNonRoleReason`, `skillHasRequiredCuratedSkillEvidence`, `skillHasRoleGate`, `skillIndexEntryPriority`, `skillIndexRuntimeCacheKey`, `skillIsUserCreatedFromSource`, `skillMatchesWorkspace`, `skillRequiresWorkflowOrTaskEvidence`, `skillRoleMatches`, `skillScanRoots`, `skillTaskDomainEvidence`, `skillTaskKeywordEvidence`, `skillWorkspaceMatchReasons`, `skillWorkspaceRoot`, `sortScoredResolvedEntries`, `sortScoredSkills`, `topResolvedSkillEntries`, `truncateSkillInjectSection`, `uniqueSortedSkillStrings` | KEEP_IN_GO | — | Skills logic — runtime spine |
| `cmd/source_check.go` | `checkCanonicalSourceSurfaces`, `checkRetiredSourceMirrors`, `looksLikeAetherSourceRoot`, `renderSourceCheckVisual` | MIXED | `colony/ceremony/source-check.md` | Source check visual is behaviour; checking logic is spine |
| `cmd/spawn_runs.go` | `beginRuntimeSpawnRun` | KEEP_IN_GO | — | Spawn runs — runtime spine |
| `cmd/spawn_track.go` | `init`, `spawnTrackCheck`, `spawnTrackStart` | KEEP_IN_GO | — | Spawn track — runtime spine |
| `cmd/spawn.go` | `emitSpawnTreeCeremony`, `init`, `latestSpawnEntryByName` | KEEP_IN_GO | — | Spawn — runtime spine |
| `cmd/state_cmds.go` | `enforceGuard`, `executeFieldMode`, `executeRevertGuard`, `exprError`, `init`, `resetMutateFlags`, `runGateCheck`, `splitChainedAssignments` | KEEP_IN_GO | — | State commands — runtime spine |
| `cmd/state_extra.go` | `init`, `shouldReopenInsertedPhase` | KEEP_IN_GO | — | State extra — runtime spine |
| `cmd/state_load.go` | `normalizeLegacyColonyState`, `repairLegacyNumericStringFields` | KEEP_IN_GO | — | State load — runtime spine |
| `cmd/state_repair.go` | `applyRecoveredPlanProgress`, `hasMissingPlanRecoveryContext`, `highestCompletedPhaseFromContinueReports`, `highestCompletedPhaseFromEvents`, `inferRecoveredPhaseProgress`, `loadPersistedPlanArtifact`, `repairMissingPlanFromArtifacts`, `setRecoveredActiveTaskStatuses`, `setRecoveredTaskStatuses` | KEEP_IN_GO | — | State repair — runtime spine |
| `cmd/status.go` | `activeFlagGuidedAction`, `buildStatusResult`, `computeWarnings`, `depthLabel`, `filterSpawnEntriesByParent`, `filterSpawnEntriesSince`, `firstActionableFlag`, `granularityLabel`, `loadActiveSpawnEntries`, `loadGuidedActions`, `loadRecentLoopBreakEvents`, `loadSpawnActivitySummaryForState`, `middenGuidedAction`, `oracleGuidedAction`, `renderActiveWorkers`, `renderDashboard`, `renderGateStatusSection`, `renderGuidedActions`, `renderLoopSafetySection`, `renderMemoryHealthTable`, `renderNoColonyStatusVisual`, `renderPheromoneSummary`, `renderRecentInstincts`, `renderRecentWorkerOutcomes`, `renderReconciliationSection`, `renderReviewFindingsTable`, `renderSignalSummaryLine`, `renderSpawnActivity`, `renderSpawnEntry`, `renderSpawnEntrySection`, `renderVersionLine`, `renderWarningsSection`, `spawnEntryTimestamp`, `withoutLiveSpawnEntries` | MIXED | `colony/ceremony/status.md` | Status visual rendering is behaviour; state aggregation is spine |
| `cmd/suggest_analyze.go` | `buildSpecificPatterns`, `countChangedFiles`, `loadActivePheromoneHashes` | KEEP_IN_GO | — | Suggest analyze — runtime spine |
| `cmd/suggest_approve.go` | package-level | KEEP_IN_GO | — | Suggest approve — runtime spine |
| `cmd/swarm_cmd.go` | `allSwarmPlans`, `buildLegacySwarmWatcherPlan`, `buildSwarmBuilderPlan`, `buildSwarmFixPlans`, `buildSwarmInvestigationPlans`, `buildSwarmManifest`, `buildSwarmPlanForCaste`, `buildSwarmPlansForWave`, `buildSwarmVerificationPlans`, `buildSwarmWatchResult`, `collectSwarmTouchedFiles`, `enrichSwarmPlanForManifest`, `executeSwarmWave`, `firstSwarmTimeout`, `init`, `initializeSwarmRun`, `invokeSwarmWorker`, `loadSwarmWorkerResponse`, `mergeExternalSwarmResults`, `queenSwarmSelectedCastes`, `recordExternalSwarmRun`, `recordSwarmFinding`, `renderExternalSwarmWorkerBrief`, `renderSwarmCompatibilityVisual`, `renderSwarmDispatchPreview`, `renderSwarmFindingSummary`, `renderSwarmWorkers`, `renderSwarmWorkerSection`, `runSwarmCompatibility`, `runSwarmDestroy`, `runSwarmFinalize`, `runSwarmPlanOnly`, `spawnEntriesToWatchMaps`, `summarizeSwarmOutcome`, `swarmExecutionsForJSON`, `swarmPlanMaps`, `swarmResponsePath`, `swarmStageName`, `swarmTaskForCaste`, `swarmWaveForCaste`, `swarmWorkerConfigOverrides`, `updateSwarmDisplayStatus`, `workerMapsFromResult` | MIXED | `colony/playbooks/swarm.md`, `control-ts/src/orchestration/swarm.ts` | Swarm orchestration is behaviour; state mutation is spine |
| `cmd/swarm_display.go` | `init` | KEEP_IN_GO | — | Swarm display — runtime spine |
| `cmd/swarm.go` | `init` | KEEP_IN_GO | — | Swarm command — runtime spine |
| `cmd/trace_cmds.go` | `generateInspectSuggestions`, `init`, `summarizeTraceEntries` | KEEP_IN_GO | — | Trace commands — runtime spine |
| `cmd/trust.go` | `init` | KEEP_IN_GO | — | Trust scoring — runtime spine |
| `cmd/unblock_cmd.go` | `buildGateRecoverySummary`, `init` | KEEP_IN_GO | — | Unblock command — runtime spine |
| `cmd/update_cmd.go` | `runUpdateSync` | KEEP_IN_GO | — | Update command — runtime spine |
| `cmd/ux_firstrun.go` | `checkAndEmitFirstRun`, `renderWelcomeBanner` | MOVE_TO_MARKDOWN | `colony/ceremony/firstrun.md` | Welcome banner text — behaviour |
| `cmd/ux_friendly_errors.go` | `renderFriendlyError` | MOVE_TO_MARKDOWN | `colony/ceremony/friendly-errors.md` | Friendly error text — behaviour |
| `cmd/ux_progress.go` | `newCeremonyProgress`, `NewCeremonyProgress` | KEEP_IN_GO | — | Progress tracking — runtime spine |
| `cmd/validation_v113.go` | `ValidateLearningEntry`, `ValidateSkillFrontmatter`, `ValidateTrackedProcessJSON` | KEEP_IN_GO | — | Validation — runtime spine |
| `cmd/verify_claims.go` | package-level | KEEP_IN_GO | — | Verify claims — runtime spine |
| `cmd/visuals_dump.go` | `casteVisualContracts` | MOVE_TO_MARKDOWN | `colony/ceremony/visuals-dump.md` | Visual contract rendering — behaviour |
| `cmd/worker_cleanup_signal_common.go` | package-level | KEEP_IN_GO | — | Worker cleanup signal — runtime spine |
| `cmd/worker_cleanup_signal_unix.go` | `setupWorkerCleanupHandler` | KEEP_IN_GO | — | Worker cleanup signal — runtime spine |
| `cmd/worker_cleanup_signal_windows.go` | `setupWorkerCleanupHandler` | KEEP_IN_GO | — | Worker cleanup signal — runtime spine |
| `cmd/worker_read_cache_discipline.go` | `renderWorkerReadCacheDiscipline` | MOVE_TO_MARKDOWN | `colony/ceremony/worker-cache.md` | Cache discipline rendering — behaviour |
| `cmd/worktree.go` | `createBlocker`, `init`, `validateBranchName` | KEEP_IN_GO | — | Worktree management — runtime spine |

---

## Mixed Files Deep-Dive

Per decision D-10, files annotated as `MIXED` contain both spine logic (stays in Go) and behaviour strings (move to editable assets). Below are function-level stay/move notes for each MIXED file.

### `cmd/alias_cmds.go`
- **Stay in Go:** `init` (command registration, CLI wiring)
- **Move to asset:** Alias definitions themselves should live in `.aether/config/aliases.yaml`

### `cmd/assumptions.go`
- **Stay in Go:** `runAssumptionList`, `runAssumptionsAnalyze` (command handlers, state mutation)
- **Move to asset:** `renderAssumptionListVisual`, `renderAssumptionsAnalyzeVisual`, `renderAssumptionValidateVisual` (visual rendering → `colony/ceremony/assumptions.md`); `buildAssumptionFeedback`, `buildAssumptionFocus`, `buildIntegrationAssumption`, `buildScopeAssumption`, `buildSurfaceAssumption`, `buildVerificationAssumption`, `synthesizeAssumptions` (assumption synthesis text → `colony/prompts/assumptions.md`)

### `cmd/autopilot.go`
- **Stay in Go:** `init` (command registration); `normalizeAutopilotPhaseStatus` (state normalization)
- **Move to asset:** Autopilot state machine rules and phase transition logic → `colony/policies/autopilot.yaml`

### `cmd/build_flow_cmds.go`
- **Stay in Go:** `init` (command registration)
- **Move to asset:** `nextUpSuggestionsForState` (suggestion text → `colony/playbooks/build-flow.md`)

### `cmd/ceremony_cmd.go`
- **Stay in Go:** `extractCeremonyManifest`, `readCeremonyJSONFile`, `ceremonyExecutionPlansFromManifest`, `ceremonyDispatchesFromManifest`, `ceremonyDispatchFromMap` (JSON parsing, file I/O); `normalizedCeremonyWorkflow` (workflow normalization)
- **Move to asset:** All `renderCeremony*` functions (ceremony visual rendering → `colony/ceremony/*.md`); `writeCeremony*` functions (ceremony output text → `colony/ceremony/*.md`); `ceremonyCasteCounts`, `ceremonyCasteCountSummary`, `dominantCeremonyCaste`, `pluralizeCaste` (caste summary text → `colony/ceremony/caste-summaries.md`)

### `cmd/ceremony_emitter.go`
- **Stay in Go:** `newBuildCeremonyEmitter`, `setActiveBuildCeremony` (emitter lifecycle); `trimCeremonyList`, `trimCeremonyPayload`, `trimCeremonyText` (text truncation — algorithmic)
- **Move to asset:** All `emitBuildCeremony*` functions (ceremony event text → `colony/ceremony/emitter.md`); `emitLifecycleCeremony`, `emitLifecycleCeremonySequence` (lifecycle ceremony text → `colony/ceremony/lifecycle.md`)

### `cmd/codex_build_progress.go`
- **Stay in Go:** `updateCodexBuildDispatchRuntimeStatus` (runtime status tracking)
- **Move to asset:** All `emitCodexBuild*` and `emitCodexDispatch*` functions (progress rendering → `colony/ceremony/build-progress.md`); `buildDispatchActiveSummary`, `buildDispatchResultSummary`, `continueWorkerCloseSummary`, `dispatchResultSummary`, `workerDispatchSummary` (summary text → `colony/ceremony/build-progress.md`)

### `cmd/codex_build.go`
- **Stay in Go:** `runCodexBuild`, `runCodexBuildWithOptions`, `runCodexBuildPlanOnly`, `runCodexBuildQueenLed` (command handlers); `buildCodexBuildManifest`, `writeCodexBuildArtifacts`, `writeCodexBuildClaims`, `writeCodexBuildOutcomeReports` (file I/O); `validateCodexBuildState`, `validateRuntimeStateMatchesExpected`, `validateRuntimeStateStillCurrent` (validation); `executeCodexBuildDispatches` (dispatch execution)
- **Move to asset:** `buildPlaybooksForDispatch` (playbook assembly → `colony/playbooks/build.md`); `renderCodexBuildWorkerBrief`, `renderCodexBuildWorkerOutcomeReport` (worker brief text → `colony/playbooks/build.md`); `resolveSkillSection`, `resolveSkillSectionForWorkflow`, `resolveWorkerSkillAssignment`, `resolveWorkerSkillAssignmentForWorkflow` (skill assignment logic → `control-ts/src/orchestration/build.ts`); `queenBuildPreWaveDispatches`, `queenBuildPostWaveDispatches`, `queenBuildTaskCaste`, `queenBuildFallbackTaskCaste` (orchestration rules → `control-ts/src/orchestration/build.ts`)

### `cmd/codex_colonize.go`
- **Stay in Go:** `runCodexColonizeWithOptions`, `runCodexColonizePlanOnly` (command handlers); `buildCodexColonizeManifest`, `updateSurveyState` (state mutation); `dispatchRealSurveyors`, `dispatchRealSurveyorsWithTimeout` (dispatch execution)
- **Move to asset:** `renderSurveyDoc` (survey doc rendering → `colony/playbooks/colonize.md`); `queenSurveyorSpecs`, `plannedSurveyors`, `surveyorDispatchMaps` (surveyor orchestration → `control-ts/src/orchestration/colonize.ts`)

### `cmd/codex_continue_plan.go`
- **Stay in Go:** `runCodexContinuePlanOnly`, `runCodexContinueVerificationSnapshot` (command handlers)
- **Move to asset:** `plannedExternalContinueDispatches` (continue dispatch planning → `colony/playbooks/continue.md`)

### `cmd/codex_continue.go`
- **Stay in Go:** `runCodexContinue`, `runCodexContinueGates`, `runCodexContinueReview`, `runCodexContinueVerification`, `runCodexContinueWatcherVerification` (command handlers); `loadCodexContinueManifest`, `updateCodexContinueContext` (state mutation); `validateContinueReconcileTasks`, `verifyCodexBuildClaims` (validation)
- **Move to asset:** All `continueWorkerFlow*` functions (worker flow text → `colony/playbooks/continue.md`); `continueReviewFlowSummary`, `continueReviewFlowTask`, `continueReviewSpecForCaste`, `continueReviewTaskForCaste` (review orchestration → `control-ts/src/orchestration/continue.ts`); `renderCodexContinueReviewBrief`, `renderCodexContinueWatcherBrief` (brief rendering → `colony/playbooks/continue.md`)

### `cmd/codex_dispatch_contract.go`
- **Stay in Go:** `persistDispatchWorkerHandoff`, `loadWorkerHandoffRecords`, `pruneWorkerHandoffRecords` (handoff persistence); `maxDuration` (utility)
- **Move to asset:** `renderDispatchContract`, `renderWorkerHandoffSection` (contract rendering → `colony/policies/dispatch-contract.yaml`); `buildQueenSpawnBudgetContract`, `enrichQueenExecutionPolicyWithSpawnBudget`, `recommendQueenExecutionPolicy`, `recommendQueenWorkflowProfile` (policy rules → `colony/policies/dispatch-contract.yaml`)

### `cmd/codex_plan.go`
- **Stay in Go:** `runCodexPlanWithOptions`, `runCodexPlanPlanOnly`, `runCodexPlanAgentDelegate`, `runCodexPlanRepairArtifact` (command handlers); `writePhaseResearchArtifacts`, `writePlanningScoutArtifact`, `writeRouteSetterArtifact`, `writeWorkerPlanArtifact` (file I/O); `loadWorkerPlanArtifact` (artifact loading)
- **Move to asset:** `renderPlanningWorkerBrief`, `renderScoutPlanningGuidance`, `renderPhasePlanSchemaGuidance` (planning brief text → `colony/playbooks/plan.md`); `planningWorkerSpecForCaste`, `planningWorkerSpecsForGoal`, `planningDispatchByCaste` (planning orchestration → `control-ts/src/orchestration/plan.ts`); `synthesizeRouteSetterPlan`, `synthesizeScoutPlanningReport` (plan synthesis text → `colony/playbooks/plan.md`)

### `cmd/codex_workflow_cmds.go`
- **Stay in Go:** `init` (command registration); `completeSealRuntime` (state mutation); `detectMDSWorkspace` (detection)
- **Move to asset:** `buildSealSummary`, `renderBlockerSummary` (seal ceremony text → `colony/ceremony/seal.md`); `createPheromoneSignal`, `newSignalShortcutCommand` (signal rules → `colony/policies/signal-rules.yaml`); `synthesizePlan` (plan synthesis → `colony/playbooks/plan.md`)

### `cmd/colony_prime_context.go`
- **Stay in Go:** `buildColonyPrimeOutput` (context assembly algorithm); `colonyPrimeLedgerItemFromRanked`, `colonyPrimeLedgerItem` (data transformation); `resolveCodexWorkerContext` (worker context resolution)
- **Move to asset:** All hardcoded prompt section text (e.g., "## Colony State", "## Pheromone Signals", "## Active Instincts", "## Key Decisions", "## Phase Learnings", "## HIVE WISDOM", "## LEARNED MEMORY", "## GLOBAL QUEEN WISDOM", "## USER PREFERENCES", "## Active Blockers", "## Colony Health Issues", "## CLARIFIED INTENT", "## Prior Reviews", "## Local Queen Wisdom") → `colony/prompts/colony-prime.md`

### `cmd/command_truth.go`
- **Stay in Go:** `deriveMilestoneProgress`, `loadCasteAssignments`, `verifyCasteSurfaces`, `runQuickScout` (logic and verification)
- **Move to asset:** `renderBumpVersionVisual`, `renderMaturityVisual`, `renderMigrateStateVisual`, `renderQuickContextCapsule`, `renderQuickVisual`, `renderVerifyCastesVisual` (visual rendering → `colony/ceremony/command-truth.md`)

### `cmd/compatibility_cmds.go`
- **Stay in Go:** `buildRunDryRunResult`, `buildRunExecutionResult`, `runCompatibilityAutopilot`, `syncRunAutopilotState`, `writeWatchArtifacts` (state mutation and I/O)
- **Move to asset:** `renderOracleCompatibilityVisual`, `renderRunCompatibilityVisual` (compatibility visual rendering → `colony/ceremony/compatibility.md`)

### `cmd/discuss.go`
- **Stay in Go:** `clarifiedIntentPromptRenderResult`, `clarifiedIntentPromptRenderResultForScope`, `renderClarifiedIntentPromptEntriesWithIntegrity` (integrity checking)
- **Move to asset:** `buildDiscussVerificationQuestion`, `renderDiscussVisual`, `renderBoundedClarifiedIntentPromptLines`, `renderClarifiedIntentPromptEntries` (prompt text and visual rendering → `colony/prompts/discuss.md`)

### `cmd/flags.go`
- **Stay in Go:** `filterFlags`, `init` (command registration, flag filtering)
- **Move to asset:** `renderFlagsTable` (flag table rendering → `colony/ceremony/flags.md`)

### `cmd/helpers.go`
- **Stay in Go:** `newLearningValidator` (validator logic)
- **Move to asset:** `renderVisualError` (visual error rendering → `colony/ceremony/helpers.md`)

### `cmd/init_ceremony.go`
- **Stay in Go:** `createCeremonyColony`, `extractCeremonyResearchData`, `isTerm`, `promptNumberedChoice`, `promptString` (colony creation, I/O helpers)
- **Move to asset:** `runCeremonyResearch`, `runInitCeremony`, `synthesizeLaunchBrief` (ceremony text and launch brief → `colony/ceremony/init.md`)

### `cmd/integrity_cmd.go`
- **Stay in Go:** `checkHubCompanionFiles` (file checking)
- **Move to asset:** `buildIntegrityVisual` (integrity visual rendering → `colony/ceremony/integrity.md`)

### `cmd/medic_auto_spawn.go`
- **Stay in Go:** `saveMedicLastScan` (scan persistence)
- **Move to asset:** `renderMedicAutoSpawnVisual` (medic visual rendering → `colony/ceremony/medic.md`)

### `cmd/medic_cmd.go`
- **Stay in Go:** `init`, `performBasicHealthScan`, `renderMedicJSON` (command registration, health scan, JSON output)
- **Move to asset:** `renderMedicReport`, `renderNoColonyMedicVisual`, `severityColor`, `writeIssueLine` (medic visual rendering → `colony/ceremony/medic.md`)

### `cmd/medic_trace.go`
- **Stay in Go:** `computeHealthScore`, `extractTimeline`, `findErrorClusters`, `findStalledPhases`, `findStateGaps`, `generateDiagnosticSuggestions` (analysis logic)
- **Move to asset:** `renderTraceDiagnostic` (trace diagnostic rendering → `colony/ceremony/medic.md`)

### `cmd/oracle_loop.go`
- **Stay in Go:** `ensureOracleSource`, `ensureOracleWorkspace`, `archiveOracleWorkspace`, `copyOracleWorkspaceEntry`, `oracleWorkspacePaths` (workspace management); `loadOracleStateFile`, `saveOracleState`, `writeOracleStateFile` (state I/O); `invokeOracleIteration`, `waitOracleAttemptResult` (dispatch execution)
- **Move to asset:** `buildOraclePhaseDirective`, `buildOracleQuestions`, `buildOracleRubric`, `buildSynthesizedPrompt`, `formulateOracleBrief` (oracle prompt text and rubric → `colony/playbooks/oracle.md`); `renderOracleContextCapsule`, `renderOracleCurrentGaps`, `renderOracleIterationPreview`, `renderOraclePriorFindings`, `renderOracleRetryPreview` (oracle visual rendering → `colony/playbooks/oracle.md`)

### `cmd/phase_skip.go`
- **Stay in Go:** `runSkipPhase`, `validateSkipPhaseTarget` (command handler, validation)
- **Move to asset:** `renderSkipPhaseVisual` (skip phase visual → `colony/ceremony/phase-skip.md`)

### `cmd/pheromone_mgmt.go`
- **Stay in Go:** `init` (command registration)
- **Move to asset:** `colonyLifecycleSignalContext` (signal context text → `colony/policies/pheromone-lifecycle.yaml`)

### `cmd/porter_cmd.go`
- **Stay in Go:** `buildPorterReadinessSummary`, `formatVersionStatus`, `lastPorterOutputLines` (readiness logic)
- **Move to asset:** `buildPorterVisual` (porter visual rendering → `colony/ceremony/porter.md`)

### `cmd/profile.go`
- **Stay in Go:** `ensureProfiledDirective`, `promoteProfileDirectives`, `promoteQueenPreferenceDirective` (directive promotion logic)
- **Move to asset:** `renderBehaviorObserveVisual`, `renderProfileReadVisual`, `renderProfileUpdateVisual` (profile visual rendering → `colony/ceremony/profile.md`)

### `cmd/proof_cmd.go`
- **Stay in Go:** `buildProofContext`, `buildProofOutput`, `buildProofSkillProof`, `convertCapsuleBlocked`, `convertColonyPrimeLedger`, `proofDispatchesForState` (proof logic)
- **Move to asset:** `renderProofDecisionSection`, `renderProofReasons`, `renderProofSkillEntries`, `renderProofVisual` (proof visual rendering → `colony/ceremony/proof.md`)

### `cmd/seal_final_review.go`
- **Stay in Go:** `ensureSealFinalReview`, `runSealFinalize`, `runSealFinalReview`, `runSealPlanOnly` (command handlers); `persistSealFinalReviewFindings`, `mergeExternalSealReviewResults` (state mutation); `validateSealReady` (validation)
- **Move to asset:** `renderSealFinalReviewBlockers`, `renderSealFinalReviewBrief`, `renderSealPlanOnlyVisual` (seal visual rendering → `colony/playbooks/seal.md`); `sealFinalReviewFlowSummary`, `sealFinalReviewSpecForCaste`, `sealFinalReviewTaskForCaste` (seal orchestration → `control-ts/src/orchestration/seal.ts`)

### `cmd/source_check.go`
- **Stay in Go:** `checkCanonicalSourceSurfaces`, `checkRetiredSourceMirrors`, `looksLikeAetherSourceRoot` (source checking logic)
- **Move to asset:** `renderSourceCheckVisual` (source check visual → `colony/ceremony/source-check.md`)

### `cmd/status.go`
- **Stay in Go:** `buildStatusResult`, `computeWarnings`, `loadActiveSpawnEntries`, `loadGuidedActions`, `loadRecentLoopBreakEvents`, `loadSpawnActivitySummaryForState` (state aggregation); `activeFlagGuidedAction`, `firstActionableFlag`, `middenGuidedAction`, `oracleGuidedAction` (guided action logic)
- **Move to asset:** All `render*` functions (status visual rendering → `colony/ceremony/status.md`)

### `cmd/swarm_cmd.go`
- **Stay in Go:** `init`, `initializeSwarmRun`, `invokeSwarmWorker`, `executeSwarmWave` (command handlers, dispatch execution); `recordExternalSwarmRun`, `mergeExternalSwarmResults` (state mutation); `loadSwarmWorkerResponse` (artifact loading)
- **Move to asset:** `buildSwarmBuilderPlan`, `buildSwarmFixPlans`, `buildSwarmInvestigationPlans`, `buildSwarmVerificationPlans`, `buildSwarmWatchResult`, `buildSwarmPlanForCaste`, `buildSwarmPlansForWave` (swarm plan orchestration → `control-ts/src/orchestration/swarm.ts`); `renderSwarmCompatibilityVisual`, `renderSwarmDispatchPreview`, `renderSwarmFindingSummary`, `renderSwarmWorkers`, `renderSwarmWorkerSection`, `renderExternalSwarmWorkerBrief` (swarm visual rendering → `colony/playbooks/swarm.md`)

---

## Cross-References

- **Asset-type matrix:** See [ARCHITECTURE_BOUNDARY.md](ARCHITECTURE_BOUNDARY.md) for the complete asset-type matrix defining target locations for each asset type.
- **Parity checklist:** See [PARITY_CLASSIC_VS_GO.md](PARITY_CLASSIC_VS_GO.md) for the Classic v5.4.0 behaviour baseline.
- **Requirements:** See `.planning/REQUIREMENTS.md` §Boundary & Parity for BOUNDARY-01 through BOUNDARY-04.
- **Migration sequence:** See ARCHITECTURE_BOUNDARY.md §Migration Sequence (Phases 152-159) for the planned extraction timeline.

---

*Documented for Phase 152. Serves as the work list for Phase 155 (Go Boundary Refactor).*
