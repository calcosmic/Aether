package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/agent"
	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/learn"
	"github.com/calcosmic/Aether/pkg/storage"
	"github.com/spf13/cobra"
)

type codexExternalContinueCompletion struct {
	ContinueManifest *codexContinuePlanManifest      `json:"continue_manifest,omitempty"`
	Manifest         *codexContinuePlanManifest      `json:"manifest,omitempty"`
	Dispatches       []codexContinueExternalDispatch `json:"dispatches,omitempty"`
	Results          []codexContinueExternalDispatch `json:"results,omitempty"`
	Workers          []codexContinueExternalDispatch `json:"workers,omitempty"`
}

var continueFinalizeCmd = &cobra.Command{
	Use:   "continue-finalize",
	Short: "Record externally spawned wrapper continue workers and advance through runtime gates",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		completionPath, _ := cmd.Flags().GetString("completion-file")
		skipMissing, _ := cmd.Flags().GetBool("skip-missing")
		noLearn, _ := cmd.Flags().GetBool("no-learn")
		verificationTimeout, verificationTimeoutExplicit, err := resolveContinueVerificationTimeoutFlag(cmd)
		if err != nil {
			outputError(1, err.Error(), nil)
			return nil
		}
		if !verificationTimeoutExplicit {
			verificationTimeout = 0
		}
		completion, err := loadExternalContinueCompletion(completionPath)
		if err != nil {
			outputError(1, err.Error(), nil)
			return nil
		}
		result, state, phase, nextPhase, housekeeping, final, err := runCodexContinueFinalize(skillWorkspaceRoot(), completion, skipMissing, verificationTimeout, noLearn)
		if err != nil {
			outputError(1, err.Error(), nil)
			return nil
		}
		if blocked, _ := result["blocked"].(bool); blocked {
			outputWorkflow(result, renderContinueBlockedVisual(state, phase, result, reviewDepthFromResult(result)))
			return nil
		}
		reviewDepthFinalize := colony.VerificationDepthLight
		if rd, ok := result["review_depth"].(string); ok {
			reviewDepthFinalize = colony.NormalizeVerificationDepth(rd)
		}
		outputWorkflow(result, renderContinueVisual(state, phase, housekeeping, final, nextPhase, result, reviewDepthFinalize))
		return nil
	},
}

func loadExternalContinueCompletion(path string) (codexExternalContinueCompletion, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return codexExternalContinueCompletion{}, fmt.Errorf("flag --completion-file is required")
	}
	var data []byte
	var err error
	if path == "-" {
		data, err = io.ReadAll(os.Stdin)
	} else {
		data, err = os.ReadFile(path)
	}
	if err != nil {
		return codexExternalContinueCompletion{}, fmt.Errorf("read completion file: %w", err)
	}

	var completion codexExternalContinueCompletion
	if err := json.Unmarshal(data, &completion); err != nil {
		return codexExternalContinueCompletion{}, fmt.Errorf("parse completion file: %w", err)
	}
	if completion.activeManifest() != nil {
		return completion, nil
	}

	var envelope struct {
		Result codexExternalContinueCompletion `json:"result"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		return codexExternalContinueCompletion{}, fmt.Errorf("parse completion envelope: %w", err)
	}
	if envelope.Result.activeManifest() == nil {
		return codexExternalContinueCompletion{}, fmt.Errorf("completion file must include continue_manifest")
	}
	return envelope.Result, nil
}

func (c codexExternalContinueCompletion) activeManifest() *codexContinuePlanManifest {
	if c.ContinueManifest != nil {
		return c.ContinueManifest
	}
	return c.Manifest
}

func (c codexExternalContinueCompletion) workerResults() []codexContinueExternalDispatch {
	results := make([]codexContinueExternalDispatch, 0, len(c.Dispatches)+len(c.Results)+len(c.Workers))
	results = append(results, c.Dispatches...)
	results = append(results, c.Results...)
	results = append(results, c.Workers...)
	return results
}

func runCodexContinueFinalize(root string, completion codexExternalContinueCompletion, skipMissing bool, verificationTimeoutOverride time.Duration, noLearn bool) (map[string]interface{}, colony.ColonyState, colony.Phase, *colony.Phase, *signalHousekeepingResult, bool, error) {
	if store == nil {
		return nil, colony.ColonyState{}, colony.Phase{}, nil, nil, false, fmt.Errorf("no store initialized")
	}
	plan := completion.activeManifest()
	if plan == nil {
		return nil, colony.ColonyState{}, colony.Phase{}, nil, nil, false, fmt.Errorf("completion file must include continue_manifest")
	}
	if plan.DispatchMode != "plan-only" || !plan.RequiresFinalizer {
		return nil, colony.ColonyState{}, colony.Phase{}, nil, nil, false, fmt.Errorf("continue_manifest must come from `aether continue --plan-only`")
	}
	if len(plan.Dispatches) == 0 && !(plan.SkipWatchers && plan.ReviewDepth == string(colony.VerificationDepthLight)) {
		return nil, colony.ColonyState{}, colony.Phase{}, nil, nil, false, fmt.Errorf("continue_manifest contains no dispatches")
	}
	if err := validateFinalizerManifestRoot("continue_manifest", plan.Root, root); err != nil {
		return nil, colony.ColonyState{}, colony.Phase{}, nil, nil, false, err
	}

	state, phase, manifest, err := validateExternalContinueState(plan)
	if err != nil {
		return nil, state, phase, nil, nil, false, err
	}
	if abandoned, _, summary := detectAbandonedBuild(manifest, state); abandoned {
		return nil, state, phase, nil, nil, false, fmt.Errorf("%s", summary)
	}

	now := time.Now().UTC()
	runHandle, err := beginRuntimeSpawnRun("continue", now)
	if err != nil {
		return nil, state, phase, nil, nil, false, fmt.Errorf("failed to initialize continue run: %w", err)
	}
	runStatus := "failed"
	defer func() {
		finishRuntimeSpawnRun(runHandle, runStatus, time.Now().UTC())
	}()

	cleanupStaleContinueReports(phase.ID)

	workerFlow, err := mergeExternalContinueResults(*plan, completion.workerResults())
	if err != nil {
		return nil, state, phase, nil, nil, false, err
	}
	if err := persistExternalContinueHandoffs(root, phase.ID, plan.Dispatches, completion.workerResults()); err != nil {
		return nil, state, phase, nil, nil, false, err
	}

	verificationTimeout := continueFinalizeVerificationTimeout(plan, verificationTimeoutOverride)
	verification := runCodexContinueVerificationSnapshot(root, phase, manifest, now, verificationTimeout, plan.SkipWatchers)
	var watcherFlow *codexContinueWorkerFlowStep
	if plan.SkipWatchers {
		verification.Watcher = codexWatcherVerification{Present: true, Passed: true, Status: "skipped", Worker: "skip-watchers", Summary: "watcher skipped; relying on runtime-owned verification commands"}
	} else {
		verification, watcherFlow = attachExternalContinueWatcher(verification, workerFlow)
	}
	assessment := assessCodexContinue(phase, manifest, verification, codexContinueOptions{ReconcileTaskIDs: plan.ReconcileTaskIDs, VerificationTimeout: verificationTimeout}, now)
	verification = attachContinueClaimVerification(verification, assessment)
	priorGateResults, _ := gateResultsReadPhase(phase.ID)
	if priorGateResults == nil {
		priorGateResults = []GateCheckResult{}
	}
	// Per SAFE-03, SAFE-04: trace continue provenance against stored manifest data.
	// Rejects claims that reference missing or stale worker results.
	if err := traceContinueProvenance(manifest.Data.Dispatches); err != nil {
		return nil, state, phase, nil, nil, false, err
	}
	finalizeReviewDepth := colony.VerificationDepthLight
	if plan.ReviewDepth != "" {
		finalizeReviewDepth = colony.NormalizeVerificationDepth(plan.ReviewDepth)
	}

	// Phase 97: Read queen-state as advisory context (D-09, D-11)
	queenAdvisory, _ := queenStateRead(phase.ID)
	// queenAdvisory.Decisions provides advisory context for logging -- finalize re-evaluates gates live
	// queenAdvisory is NOT used to skip or alter gate evaluation -- it is purely informational
	_ = queenAdvisory
	gates := runCodexContinueGates(phase, manifest, verification, assessment, now, priorGateResults)

	verificationReportRel := continuePlanArtifactsPath(phase.ID, "verification.json")
	gateReportRel := continuePlanArtifactsPath(phase.ID, "gates.json")
	if err := store.SaveJSON(verificationReportRel, verification); err != nil {
		return nil, state, phase, nil, nil, false, fmt.Errorf("failed to write verification report: %w", err)
	}
	if err := store.SaveJSON(gateReportRel, gates); err != nil {
		return nil, state, phase, nil, nil, false, fmt.Errorf("failed to write gate report: %w", err)
	}

	// Persist gate results after gate run
	var gateResultEntries []colony.GateResultEntry
	for _, c := range gates.Checks {
		gateResultEntries = append(gateResultEntries, colony.GateResultEntry{
			Name:      c.Name,
			Passed:    c.Passed,
			Timestamp: now.Format(time.RFC3339),
			Detail:    c.Detail,
		})
	}
	if err := gateResultsWrite(gateResultEntries); err != nil {
		return nil, state, phase, nil, nil, false, fmt.Errorf("failed to persist gate results: %w", err)
	}

	// Per-phase gate results persistence (D-14)
	var phaseGateResults []GateCheckResult
	for _, c := range gates.Checks {
		status := "passed"
		if !c.Passed {
			status = "failed"
		}
		phaseGateResults = append(phaseGateResults, GateCheckResult{
			Name:            c.Name,
			Status:          status,
			Detail:          c.Detail,
			FixHint:         c.FixHint,
			RecoveryOptions: c.RecoveryOptions,
			Timestamp:       now.Format(time.RFC3339),
		})
	}
	if err := gateResultsWritePhase(phase.ID, phaseGateResults); err != nil {
		return nil, state, phase, nil, nil, false, fmt.Errorf("failed to persist phase gate results: %w", err)
	}

	if err := writeCodexContinueWorkerOutcomeReports(root, phase, workerFlow, now); err != nil {
		return nil, state, phase, nil, nil, false, err
	}

	// --- Auto-resolve soft_block gates (Phase 95, GATE-03) ---
	// Per D-02: auto-resolve runs inside continue command, no new commands.
	// Per D-04: only soft_block gates are auto-resolved.
	if !gates.Passed {
		resolveDepth := ""
		if plan.ReviewDepth != "" {
			resolveDepth = plan.ReviewDepth
		}
		gates, autoResolved := autoResolveSoftBlockGates(phase.ID, gates, resolveDepth, phase.Mode)

		if len(autoResolved) > 0 {
			// Re-persist gate results with auto-resolved annotations
			phaseGateResults = nil
			for _, c := range gates.Checks {
				status := "passed"
				if !c.Passed {
					status = "failed"
				}
				entry := GateCheckResult{
					Name:            c.Name,
					Status:          status,
					Detail:          c.Detail,
					FixHint:         c.FixHint,
					RecoveryOptions: c.RecoveryOptions,
					Timestamp:       now.Format(time.RFC3339),
				}
				for _, resolved := range autoResolved {
					if resolved == c.Name {
						entry.QueenAnnotation = &QueenAnnotation{
							Decision:     "auto-resolved",
							Rationale:    fmt.Sprintf("soft_block gate %q auto-resolved at depth %s", c.Name, colony.NormalizeVerificationDepth(resolveDepth)),
							Timestamp:    now.Format(time.RFC3339),
							QueenVersion: "1.0.27",
						}
						break
					}
				}
				phaseGateResults = append(phaseGateResults, entry)
			}
			if err := gateResultsWritePhase(phase.ID, phaseGateResults); err != nil {
				return nil, state, phase, nil, nil, false, fmt.Errorf("failed to persist auto-resolved phase gate results: %w", err)
			}

			// Also update COLONY_STATE.json gate results
			var updatedGateEntries []colony.GateResultEntry
			for _, c := range gates.Checks {
				updatedGateEntries = append(updatedGateEntries, colony.GateResultEntry{
					Name:      c.Name,
					Passed:    c.Passed,
					Timestamp: now.Format(time.RFC3339),
					Detail:    c.Detail,
				})
			}
			if err := gateResultsWrite(updatedGateEntries); err != nil {
				return nil, state, phase, nil, nil, false, fmt.Errorf("failed to persist auto-resolved gate results: %w", err)
			}

			// Log recovery actions (per RECV-06, using Phase 94 recovery log)
			var recoveryEntries []RecoveryLogEntry
			for idx, resolved := range autoResolved {
				recoveryEntries = append(recoveryEntries, RecoveryLogEntry{
					ID: fmt.Sprintf("auto-resolve-%s-%d-%s", resolved, phase.ID, now.Format("20060102-150405")),
					Failure: FailureRecord{
						WorkerName:     "",
						TaskID:         "",
						Caste:          "",
						Phase:          phase.ID,
						Status:         "failed",
						Classification: Recoverable,
						FailureType:    Transient,
						ErrorMessage:   fmt.Sprintf("soft_block gate %q failed", resolved),
						Timestamp:      now.Format(time.RFC3339),
					},
					ActionTaken:   "auto-resolved",
					Outcome:       "gate threshold met -- auto-resolved by queen",
					AttemptNumber: idx + 1,
					Timestamp:     now.Format(time.RFC3339),
					Detail:        fmt.Sprintf("gate %q auto-resolved at depth %s", resolved, colony.NormalizeVerificationDepth(resolveDepth)),
				})
			}
			if len(recoveryEntries) > 0 {
				existingLog, _ := recoveryLogReadPhase(phase.ID)
				existingLog.Entries = append(existingLog.Entries, recoveryEntries...)
				existingLog.Phase = phase.ID
				_ = recoveryLogWritePhase(phase.ID, existingLog.Entries)
			}
		}

		// Per D-03: if auto-resolve didn't clear all failures, dispatch Fixer for remaining soft_block gates
		if !gates.Passed {
			hasSoftBlockRemaining := false
			for _, c := range gates.Checks {
				if !c.Passed {
					tier, _ := phaseModeAwareGateClassify(c.Name, phase.Mode)
					if tier == softBlock {
						hasSoftBlockRemaining = true
						break
					}
				}
			}

			if hasSoftBlockRemaining {
				_ = dispatchFixer(phase.ID, "propose")
			}

			// --- Phase 96: Auto-recovery orchestrator for gate failures (RECV-04) ---
			// Per D-09: this is a NEW trigger path, distinct from Phase 95's auto-resolve.
			// The orchestrator evaluates whether retry/peer/fixer strategies apply to gate failures.
			// This runs AFTER auto-resolve attempt, AFTER Phase 95's dispatchFixer call.
			// Per D-05: both build and continue call orchestrateRecovery for their failure types.
			var gateRecoveryInstructions []map[string]interface{}
			if !gates.Passed {
				budget := budgetFromRecoveryLog(phase.ID, 1) // continue uses wave 1
				if budget == nil {
					budget = newRecoveryBudget(1)
				}

				for _, c := range gates.Checks {
					if c.Passed {
						continue
					}
					tier, _ := gateClassify(c.Name)
					// Per D-04: blocking failures escalate immediately, no orchestrator
					if tier == hardBlock {
						gateRecoveryInstructions = append(gateRecoveryInstructions, map[string]interface{}{
							"gate":           c.Name,
							"classification": "hard_block",
							"action":         "escalate",
							"detail":         "hard_block gate failure requires human intervention",
						})
						continue
					}

					// Build recovery context from gate failure
					ctx := RecoveryContext{
						Phase:          phase.ID,
						Wave:           1,
						WorkerName:     fmt.Sprintf("gate-%s", c.Name),
						Caste:          "watcher",
						Status:         "failed",
						ErrorMessage:   c.Detail,
						Budget:         budget,
						CircuitBreaker: globalCircuitBreaker,
					}
					outcome := orchestrateRecovery(ctx)

					// Persist recovery log entries
					if len(outcome.LogEntries) > 0 {
						existingLog, _ := recoveryLogReadPhase(phase.ID)
						existingLog.Entries = append(existingLog.Entries, outcome.LogEntries...)
						existingLog.Phase = phase.ID
						_ = recoveryLogWritePhase(phase.ID, existingLog.Entries)
					}

					gateRecoveryInstructions = append(gateRecoveryInstructions, map[string]interface{}{
						"gate":           c.Name,
						"classification": string(outcome.Classification),
						"action":         outcome.Action.Type,
						"detail":         outcome.Action.Detail,
						"exhausted":      outcome.Exhausted,
						"rationale":      outcome.Rationale,
					})
				}

				// Persist updated budget
				_ = persistBudgetToRecoveryLog(phase.ID, budget)

				// Phase 97: Log circuit breaker escalation events (D-12, COORD-04)
				if globalCircuitBreaker != nil {
					tripped := globalCircuitBreaker.TrippedWorkers()
					if len(tripped) > 0 {
						queenLogEscalation(phase.ID, tripped, "circuit breaker tripped during finalize -- escalation required")
					}
				}
			}

			result, blockedState, err := finalizeBlockedExternalContinue(state, phase, manifest, verification, assessment, gates, nil, "", workerFlow, now, verificationReportRel, gateReportRel, gateRecoveryInstructions, finalizeReviewDepth)
			if err != nil {
				return nil, state, phase, nil, nil, false, err
			}
			runStatus = "blocked"
			return result, blockedState, phase, nil, nil, false, nil
		}
	}

	review := externalContinueReviewReport(phase.ID, workerFlow, now, skipMissing, finalizeReviewDepth, plan.Dispatches)
	reviewReportRel := continuePlanArtifactsPath(phase.ID, "review.json")
	if err := store.SaveJSON(reviewReportRel, review); err != nil {
		return nil, state, phase, nil, nil, false, fmt.Errorf("failed to write review report: %w", err)
	}
	if !review.Passed {
		result, blockedState, err := finalizeBlockedExternalContinue(state, phase, manifest, verification, assessment, gates, &review, reviewReportRel, workerFlow, now, verificationReportRel, gateReportRel, nil, finalizeReviewDepth)
		if err != nil {
			return nil, state, phase, nil, nil, false, err
		}
		runStatus = "blocked"
		return result, blockedState, phase, nil, nil, false, nil
	}

	// --- Learning capture (D-01, D-02, D-03, D-04) ---
	// Learning fires only after gates pass AND review passes AND provenance valid AND all workers succeeded.
	// This is the ONLY path that produces durable learning (D-03).
	captureLearning := func() {
		// Check if all workers succeeded (D-02)
		allWorkersSucceeded := true
		for _, step := range workerFlow {
			if step.Status != "completed" {
				allWorkersSucceeded = false
				break
			}
		}

		// Check if learning is enabled (D-16) -- config + flag
		learningEnabled := isLearningEnabled(noLearn)

		if !learn.IsLearningEligible(allWorkersSucceeded, true, gates.Passed, learningEnabled) {
			return // Not eligible -- no durable learning
		}

		// Collect evidence (D-09, LRN-02)
		workerResults := make([]learn.WorkerResult, 0, len(workerFlow))
		for _, step := range workerFlow {
			workerResults = append(workerResults, learn.WorkerResult{
				Name:         step.Name,
				Caste:        step.Caste,
				Status:       step.Status,
				FilesTouched: nil, // codexContinueWorkerFlowStep has no FilesModified field
			})
		}

		gatesPassed := 0
		for _, c := range gates.Checks {
			if c.Passed {
				gatesPassed++
			}
		}

		runID := ""
		if runHandle != nil {
			runID = runHandle.Run.ID
		}
		if runID == "" {
			runID = fmt.Sprintf("run_%d_%s", phase.ID, now.Format("20060102_150405"))
		}

		evidence := learn.CollectEvidence(
			runID, phase.ID, workerResults,
			learn.GateResult{Passed: gatesPassed, Total: len(gates.Checks)},
			"repo-local",
		)

		// Build learning content from phase summary
		content := fmt.Sprintf("Phase %d completed successfully: %s", phase.ID, phase.Name)

		// Run privacy scan + classify (D-10, D-11, PRIV-03)
		scanResult := privacyScan(content)
		classification := learn.ClassifyEntry(content, learn.PrivacyScanResult{
			Blocked:  scanResult.Blocked,
			Clean:    scanResult.Clean,
			Findings: scanResult.Findings,
		})

		if classification == learn.ClassBlocked {
			return // Blocked content never stored
		}

		// Store via ColonyStore (D-06: .aether/data/learn/)
		learnStore := learn.NewColonyStore(store)
		entry := learn.Entry{
			Content:        scanResult.Clean, // use cleaned content
			Evidence:       evidence,
			Classification: classification,
			Phase:          phase.ID,
			Confidence:     evidence.Confidence,
		}
		if err := learnStore.Add(entry); err != nil {
			// Non-blocking: learning failure must not prevent phase advancement
			fmt.Fprintf(os.Stderr, "warning: failed to capture learning: %v\n", err)
		} else {
			// Phase 91: Auto-skill creation hook (AUTO-01)
			// Only fires after successful learning capture for difficult verified tasks.
			// Reads auto_skill_mode config to determine behavior (off/propose/auto, default propose).
			sqliteStore, sqliteErr := learn.NewSQLiteColonyStore(filepath.Join(store.BasePath(), "colony.db"))
			if sqliteErr == nil {
				defer sqliteStore.Close()
				aetherRoot := storage.ResolveAetherRoot(context.Background())
				mode := learn.LoadAutoSkillMode(store.BasePath())
				if err := learn.AutoCreateSkillIfDifficult(entry, sqliteStore, aetherRoot, mode); err != nil {
					// Non-blocking: auto-skill failure must not prevent phase advancement
					fmt.Fprintf(os.Stderr, "warning: failed to auto-create skill: %v\n", err)
				}
			}
		}
	}
	captureLearning()

	result, updated, nextPhase, housekeeping, final, err := advanceExternalContinue(root, state, phase, manifest, verification, assessment, gates, review, reviewReportRel, watcherFlow, workerFlow, now, verificationReportRel, gateReportRel, finalizeReviewDepth)
	if err != nil {
		return nil, state, phase, nil, housekeeping, final, err
	}
	runStatus = "completed"
	return result, updated, phase, nextPhase, housekeeping, final, nil
}

func validateExternalContinueState(plan *codexContinuePlanManifest) (colony.ColonyState, colony.Phase, codexContinueManifest, error) {
	state, err := loadActiveColonyState()
	if err != nil {
		return state, colony.Phase{}, codexContinueManifest{}, fmt.Errorf("%s", colonyStateLoadMessage(err))
	}
	if len(state.Plan.Phases) == 0 {
		return state, colony.Phase{}, codexContinueManifest{}, fmt.Errorf("No project plan. Run `aether plan` first.")
	}
	if state.State != colony.StateEXECUTING && state.State != colony.StateBUILT {
		return state, colony.Phase{}, codexContinueManifest{}, fmt.Errorf("No active phase to continue. Run `aether build <phase>` first.")
	}
	if state.CurrentPhase < 1 || state.CurrentPhase > len(state.Plan.Phases) {
		return state, colony.Phase{}, codexContinueManifest{}, fmt.Errorf("No active phase to continue. Run `aether build <phase>` first.")
	}
	if plan.Phase != state.CurrentPhase {
		return state, colony.Phase{}, codexContinueManifest{}, fmt.Errorf("continue_manifest phase %d does not match active phase %d", plan.Phase, state.CurrentPhase)
	}
	if err := validateFinalizerManifestColonyMode("continue_manifest", plan.ColonyMode, state); err != nil {
		return state, colony.Phase{}, codexContinueManifest{}, err
	}
	phase := state.Plan.Phases[state.CurrentPhase-1]
	if phase.Status != colony.PhaseInProgress {
		return state, phase, codexContinueManifest{}, fmt.Errorf("phase %d is not in progress; run `aether build %d` first", phase.ID, phase.ID)
	}
	if err := validateContinueReconcileTasks(phase, plan.ReconcileTaskIDs); err != nil {
		return state, phase, codexContinueManifest{}, err
	}
	manifest := loadCodexContinueManifest(phase.ID)
	if state.BuildStartedAt == nil && !manifest.Present {
		return state, phase, manifest, fmt.Errorf("No active build packet found. Run `aether build <phase>` first.")
	}
	return state, phase, manifest, nil
}

func continueFinalizeVerificationTimeout(plan *codexContinuePlanManifest, override time.Duration) time.Duration {
	if override > 0 {
		return override
	}
	if plan != nil && plan.VerificationTimeout > 0 {
		return time.Duration(plan.VerificationTimeout) * time.Second
	}
	return continueVerificationTimeout
}

func mergeExternalContinueResults(plan codexContinuePlanManifest, results []codexContinueExternalDispatch) ([]codexContinueWorkerFlowStep, error) {
	resultByName := make(map[string]codexContinueExternalDispatch, len(results))
	for _, result := range results {
		name := strings.TrimSpace(result.Name)
		if name == "" {
			return nil, fmt.Errorf("external continue result missing name")
		}
		if _, exists := resultByName[name]; exists {
			return nil, fmt.Errorf("duplicate external continue result for %s", name)
		}
		resultByName[name] = result
	}

	flow := make([]codexContinueWorkerFlowStep, 0, len(plan.Dispatches))
	for _, dispatch := range plan.Dispatches {
		result, ok := resultByName[dispatch.Name]
		if !ok {
			result = codexContinueExternalDispatch{
				Stage:   dispatch.Stage,
				Caste:   dispatch.Caste,
				Name:    dispatch.Name,
				Task:    dispatch.Task,
				TaskID:  dispatch.TaskID,
				Status:  "timeout",
				Summary: "worker result was not provided; treated as timed out",
			}
		}
		if err := validateExternalContinueIdentity(dispatch, result); err != nil {
			return nil, err
		}
		status := normalizeExternalBuildStatus(result.Status)
		if !isTerminalExternalBuildStatus(status) {
			return nil, fmt.Errorf("external continue result for %s has non-terminal status %q", dispatch.Name, result.Status)
		}
		if ok {
			if err := codex.ValidateWorkerHandoff(result.Handoff); err != nil {
				return nil, fmt.Errorf("external continue result for %s has invalid handoff: %w", dispatch.Name, err)
			}
		}
		summary := strings.TrimSpace(result.Summary)
		blockers := uniqueSortedStrings(result.Blockers)
		if summary == "" && len(blockers) > 0 {
			summary = strings.Join(blockers, "; ")
		}
		flow = append(flow, codexContinueWorkerFlowStep{
			Stage:           dispatch.Stage,
			Caste:           dispatch.Caste,
			Name:            dispatch.Name,
			Task:            dispatch.Task,
			Status:          status,
			Summary:         summary,
			Blockers:        blockers,
			Duration:        result.Duration,
			Report:          strings.TrimSpace(result.Report),
			Findings:        mergeCodexReviewFindings(result.Findings, result.Issues),
			Recommendations: uniqueSortedStrings(result.Recommendations),
			WeakSpots:       uniqueSortedStrings(result.WeakSpots),
			EdgeCases:       uniqueSortedStrings(result.EdgeCases),
			ReusableLessons: uniqueSortedStrings(result.ReusableLessons),
		})
	}
	return flow, nil
}

func mergeCodexReviewFindings(groups ...[]codexReviewFinding) []codexReviewFinding {
	var merged []codexReviewFinding
	seen := map[string]bool{}
	for _, group := range groups {
		for _, finding := range group {
			finding.Description = strings.TrimSpace(finding.Description)
			if finding.Description == "" {
				finding.Description = strings.TrimSpace(finding.Title)
			}
			finding.Title = strings.TrimSpace(finding.Title)
			finding.Domain = strings.TrimSpace(strings.ToLower(finding.Domain))
			finding.Severity = strings.TrimSpace(strings.ToUpper(finding.Severity))
			finding.File = strings.TrimSpace(finding.File)
			finding.Category = strings.TrimSpace(finding.Category)
			finding.Suggestion = strings.TrimSpace(finding.Suggestion)
			if finding.Description == "" && finding.Suggestion == "" {
				continue
			}
			key := strings.Join([]string{
				finding.Domain,
				finding.Severity,
				finding.File,
				fmt.Sprintf("%d", finding.Line),
				finding.Category,
				finding.Description,
				finding.Suggestion,
			}, "\x00")
			if seen[key] {
				continue
			}
			seen[key] = true
			merged = append(merged, finding)
		}
	}
	return merged
}

func persistExternalContinueHandoffs(root string, phaseNum int, dispatches []codexContinueExternalDispatch, results []codexContinueExternalDispatch) error {
	resultByName := make(map[string]codexContinueExternalDispatch, len(results))
	for _, result := range results {
		if name := strings.TrimSpace(result.Name); name != "" {
			resultByName[name] = result
		}
	}
	for _, dispatch := range dispatches {
		result, ok := resultByName[dispatch.Name]
		if !ok {
			continue
		}
		status := normalizeExternalBuildStatus(result.Status)
		workerResult := &codex.WorkerResult{
			WorkerName: dispatch.Name,
			Caste:      dispatch.Caste,
			TaskID:     dispatch.TaskID,
			Status:     status,
			Summary:    result.Summary,
			Handoff:    codex.NormalizeWorkerHandoff(root, result.Handoff),
			Blockers:   result.Blockers,
		}
		if err := persistDispatchWorkerHandoff(codex.WorkerDispatch{
			WorkerName: dispatch.Name,
			Caste:      dispatch.Caste,
			TaskID:     dispatch.TaskID,
			Workflow:   "continue",
			Phase:      phaseNum,
			Wave:       dispatch.Wave,
			Root:       root,
		}, codex.DispatchResult{
			WorkerName:   dispatch.Name,
			Status:       status,
			WorkerResult: workerResult,
		}); err != nil {
			return err
		}
	}
	return nil
}

func validateExternalContinueIdentity(dispatch codexContinueExternalDispatch, result codexContinueExternalDispatch) error {
	dispatchSpec := workerIdentitySpec{
		Caste:  dispatch.Caste,
		Stage:  dispatch.Stage,
		TaskID: dispatch.TaskID,
		Wave:   dispatch.Wave,
	}
	resultSpec := workerIdentitySpec{
		Caste:  result.Caste,
		Stage:  result.Stage,
		TaskID: result.TaskID,
		Wave:   result.Wave,
	}
	return validateWorkerResultIdentity(dispatch.Name, dispatchSpec, resultSpec)
}

func attachExternalContinueWatcher(verification codexContinueVerificationReport, workerFlow []codexContinueWorkerFlowStep) (codexContinueVerificationReport, *codexContinueWorkerFlowStep) {
	for _, step := range workerFlow {
		if strings.TrimSpace(step.Stage) != "verification" || !strings.EqualFold(strings.TrimSpace(step.Caste), "watcher") {
			continue
		}
		status := continueWorkerFlowStatus(step.Status)
		summary := strings.TrimSpace(step.Summary)
		if summary == "" {
			summary = continueWatcherDefaultSummary(status)
		}
		watcher := codexWatcherVerification{
			Present: true,
			Passed:  status == "completed" || status == "manually-reconciled",
			Status:  status,
			Worker:  strings.TrimSpace(step.Name),
			Summary: summary,
		}
		if isEnvironmentBlockedLaunchVerification(strings.Join(append([]string{summary}, step.Blockers...), "\n")) {
			watcher = environmentBlockedWatcher(watcher)
		}
		verification.Watcher = watcher
		if !watcher.Passed {
			if watcher.Status == "timeout" && verification.ChecksPassed {
				// Watcher timed out but runtime verification (build, types, lint, tests)
				// passed independently. Treat as advisory, not a hard block.
				verification.BlockingIssues = uniqueSortedStrings(append(
					verification.BlockingIssues,
					fmt.Sprintf("watcher %s timed out; runtime verification passed independently", watcher.Worker),
				))
			} else {
				verification.ChecksPassed = false
				verification.Passed = false
				verification.BlockingIssues = uniqueSortedStrings(append(verification.BlockingIssues, summary))
			}
		}
		watcherFlow := step
		watcherFlow.Summary = continueWatcherFlowSummary(watcher.Worker, watcher.Status, watcher.Summary)
		return verification, &watcherFlow
	}
	verification.ChecksPassed = false
	verification.Passed = false
	verification.BlockingIssues = uniqueSortedStrings(append(verification.BlockingIssues, "wrapper continue watcher result is missing"))
	return verification, nil
}

func externalContinueReviewReport(phaseID int, workerFlow []codexContinueWorkerFlowStep, now time.Time, skipMissing bool, reviewDepth colony.VerificationDepth, plannedDispatches ...[]codexContinueExternalDispatch) codexContinueReviewReport {
	report := codexContinueReviewReport{
		Phase:       phaseID,
		GeneratedAt: now.Format(time.RFC3339),
		Workers:     []codexContinueWorkerFlowStep{},
		Passed:      true,
	}
	blockers := []string{}
	warnings := []string{}
	for _, step := range workerFlow {
		if strings.TrimSpace(step.Stage) != "review" {
			continue
		}
		status := continueWorkerFlowStatus(step.Status)
		if skipMissing && status == "timeout" {
			continue
		}
		report.Workers = append(report.Workers, step)
		if status == "completed" || status == "manually-reconciled" {
			continue
		}
		if continueWorkerFlowEnvironmentBlocked(step) {
			continue
		}
		if status == "timeout" {
			warnings = append(warnings, fmt.Sprintf("%s review timed out; review was not completed", step.Name))
			continue
		}
		report.Passed = false
		blockers = append(blockers, fmt.Sprintf("%s review did not complete cleanly: %s", step.Name, status))
		if summary := strings.TrimSpace(step.Summary); summary != "" {
			blockers = append(blockers, fmt.Sprintf("%s reported blocker: %s", step.Name, summary))
		}
	}
	if expectedCastes := expectedContinueReviewCastes(reviewDepth, plannedDispatches...); len(expectedCastes) > 0 {
		actualCastes := actualContinueReviewCastes(report.Workers)
		if skipMissing {
			if !continueReviewCastesArePlannedSubset(actualCastes, expectedCastes) {
				report.Passed = false
				blockers = append(blockers, fmt.Sprintf("expected review castes %v, got %v", expectedCastes, actualCastes))
			}
		} else if !equalStringSlices(actualCastes, expectedCastes) {
			report.Passed = false
			blockers = append(blockers, fmt.Sprintf("expected review castes %v, got %v", expectedCastes, actualCastes))
		}
	}
	report.BlockingIssues = uniqueSortedStrings(blockers)
	report.Passed = report.Passed && len(report.BlockingIssues) == 0
	// Timed-out review agents produce warnings, not blocks, when no
	// completing agent reported a hard blocker.
	if report.Passed && len(warnings) > 0 {
		report.BlockingIssues = uniqueSortedStrings(warnings)
	}
	return report
}

func expectedContinueReviewCastes(reviewDepth colony.VerificationDepth, plannedDispatches ...[]codexContinueExternalDispatch) []string {
	for _, dispatches := range plannedDispatches {
		castes := make([]string, 0, len(dispatches))
		for _, dispatch := range dispatches {
			if strings.TrimSpace(dispatch.Stage) != "review" {
				continue
			}
			caste := strings.TrimSpace(dispatch.Caste)
			if caste == "" {
				continue
			}
			castes = append(castes, caste)
		}
		if len(castes) > 0 {
			return castes
		}
	}

	switch colony.NormalizeVerificationDepth(string(reviewDepth)) {
	case colony.VerificationDepthLight:
		return nil
	case colony.VerificationDepthStandard:
		return []string{"gatekeeper"}
	default:
		castes := make([]string, 0, len(codexContinueReviewSpecs))
		for _, spec := range codexContinueReviewSpecs {
			castes = append(castes, spec.Caste)
		}
		return castes
	}
}

func actualContinueReviewCastes(workers []codexContinueWorkerFlowStep) []string {
	castes := make([]string, 0, len(workers))
	for _, worker := range workers {
		if caste := strings.TrimSpace(worker.Caste); caste != "" {
			castes = append(castes, caste)
		}
	}
	return castes
}

func equalStringSlices(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

func continueReviewCastesArePlannedSubset(actual, expected []string) bool {
	expectedIndex := 0
	for _, actualCaste := range actual {
		matched := false
		for expectedIndex < len(expected) {
			if actualCaste == expected[expectedIndex] {
				expectedIndex++
				matched = true
				break
			}
			expectedIndex++
		}
		if !matched {
			return false
		}
	}
	return true
}

func finalizeBlockedExternalContinue(state colony.ColonyState, phase colony.Phase, manifest codexContinueManifest, verification codexContinueVerificationReport, assessment codexContinueAssessment, gates codexContinueGateReport, review *codexContinueReviewReport, reviewReportRel string, workerFlow []codexContinueWorkerFlowStep, now time.Time, verificationReportRel, gateReportRel string, gateRecoveryInstructions []map[string]interface{}, reviewDepth colony.VerificationDepth) (map[string]interface{}, colony.ColonyState, error) {
	blockers := append([]string{}, gates.BlockingIssues...)
	if review != nil {
		blockers = append(blockers, review.BlockingIssues...)
	}
	blockers = uniqueSortedStrings(blockers)
	summary := "Continue blocked by verification, gate, or review failures"
	if len(blockers) > 0 {
		summary = blockers[0]
	}
	continueReportRel := continuePlanArtifactsPath(phase.ID, "continue.json")
	nextCommand := continueNextCommandForAssessment(assessment)
	_ = store.SaveJSON(continueReportRel, codexContinueReport{
		Phase:              phase.ID,
		GeneratedAt:        now.Format(time.RFC3339),
		Manifest:           displayOptionalDataPath(manifest.Path),
		VerificationReport: displayDataPath(verificationReportRel),
		GateReport:         displayDataPath(gateReportRel),
		ReviewReport:       displayOptionalDataPath(reviewReportRel),
		Summary:            summary,
		WorkerFlow:         workerFlow,
		PartialSuccess:     assessment.PartialSuccess,
		OperationalIssues:  append([]string{}, assessment.OperationalIssues...),
		Tasks:              append([]codexContinueTaskAssessment{}, assessment.Tasks...),
		Recovery:           assessment.Recovery,
		Advanced:           false,
		Completed:          false,
		Next:               nextCommand,
	})
	if err := recordExternalContinueWorkerFlow(workerFlow); err != nil {
		return nil, state, err
	}
	blockedState := state
	blockedState.Events = append(trimmedEvents(blockedState.Events), continueWorkerFlowEvents(now, workerFlow)...)
	if err := store.SaveJSON("COLONY_STATE.json", blockedState); err != nil {
		return nil, state, fmt.Errorf("failed to save colony state: %w", err)
	}
	emitContinueCeremonyFlowSequence("aether-continue-finalize", phase, workerFlow)
	updateSessionSummary("continue-finalize", nextCommand, summary)
	result := map[string]interface{}{
		"advanced":            false,
		"blocked":             true,
		"partial_success":     assessment.PartialSuccess,
		"current_phase":       blockedState.CurrentPhase,
		"phase_name":          phase.Name,
		"state":               blockedState.State,
		"next":                nextCommand,
			"review_depth":        string(reviewDepth),
		"verification":        verification,
		"assessment":          assessment,
		"task_evidence":       assessment.Tasks,
		"gates":               gates,
		"verification_report": displayDataPath(verificationReportRel),
		"gate_report":         displayDataPath(gateReportRel),
		"continue_report":     displayDataPath(continueReportRel),
		"worker_flow":         workerFlow,
		"operational_issues":  assessment.OperationalIssues,
		"recovery":            assessment.Recovery,
		"reconciled_tasks":    assessment.ReconciledTasks,
		"blocking_issues":     blockers,
	}
	if review != nil {
		result["review"] = *review
		result["review_report"] = displayDataPath(reviewReportRel)
	}
	if len(gateRecoveryInstructions) > 0 {
		result["recovery_instructions"] = gateRecoveryInstructions
	}
	addOrchestratorBoundaryGuidance(result, "continue", blockedState, nextCommand, nil)
	return result, blockedState, nil
}

func advanceExternalContinue(root string, state colony.ColonyState, phase colony.Phase, manifest codexContinueManifest, verification codexContinueVerificationReport, assessment codexContinueAssessment, gates codexContinueGateReport, review codexContinueReviewReport, reviewReportRel string, watcherFlow *codexContinueWorkerFlowStep, workerFlow []codexContinueWorkerFlowStep, now time.Time, verificationReportRel, gateReportRel string, reviewDepth colony.VerificationDepth) (map[string]interface{}, colony.ColonyState, *colony.Phase, *signalHousekeepingResult, bool, error) {
	currentIdx := state.CurrentPhase - 1
	closedWorkerDetails := plannedCodexContinueClosedWorkers(manifest, assessment)
	closedWorkers := closedWorkerNames(closedWorkerDetails)

	var (
		nextPhase   *colony.Phase
		nextCommand string
		final       bool
		updated     colony.ColonyState
	)
	if err := store.UpdateJSONAtomically("COLONY_STATE.json", &updated, func() error {
		updated = state
		updated.Events = append(trimmedEvents(updated.Events),
			fmt.Sprintf("%s|verification_passed|continue-finalize|Build verification passed for phase %d", now.Format(time.RFC3339), phase.ID),
			fmt.Sprintf("%s|gate_passed|continue-finalize|Continue gates passed for phase %d", now.Format(time.RFC3339), phase.ID),
		)
		updated.Plan.Phases[currentIdx].Status = colony.PhaseCompleted
		for i := range updated.Plan.Phases[currentIdx].Tasks {
			updated.Plan.Phases[currentIdx].Tasks[i].Status = colony.TaskCompleted
		}
		updated.BuildStartedAt = nil
		updated.GateResults = nil

		final = currentIdx == len(updated.Plan.Phases)-1
		nextCommand = "aether seal"
		if final {
			updated.State = colony.StateCOMPLETED
			updated.CurrentPhase = phase.ID
			updated.Events = append(updated.Events,
				fmt.Sprintf("%s|phase_completed|continue-finalize|Completed final phase %d", now.Format(time.RFC3339), updated.CurrentPhase),
			)
		} else {
			nextIdx := currentIdx + 1
			if updated.Plan.Phases[nextIdx].Status == colony.PhasePending || updated.Plan.Phases[nextIdx].Status == "" {
				updated.Plan.Phases[nextIdx].Status = colony.PhaseReady
			}
			updated.CurrentPhase = nextIdx + 1
			nextPhase = &updated.Plan.Phases[nextIdx]
			updated.State = colony.StateREADY
			nextCommand = fmt.Sprintf("aether build %d", nextIdx+1)
			updated.Events = append(updated.Events,
				fmt.Sprintf("%s|phase_advanced|continue-finalize|Completed phase %d, ready for phase %d", now.Format(time.RFC3339), phase.ID, nextIdx+1),
			)
		}
		return nil
	}); err != nil {
		return nil, state, nil, nil, false, fmt.Errorf("failed to atomically advance phase: %w", err)
	}

	housekeeping, housekeepingErr := continueSignalHousekeeper(now, updated)
	if housekeepingErr != nil {
		return nil, state, nil, nil, final, housekeepingErr
	}
	if err := continueContextUpdater(phase, manifest, closedWorkerDetails, now); err != nil {
		return nil, state, nil, &housekeeping, final, err
	}
	fullWorkerFlow := continueWorkerFlowWithWatcher(review.Workers, watcherFlow)
	fullWorkerFlow = append(fullWorkerFlow, continueHousekeepingFlowStep(housekeeping))
	if err := recordExternalContinueWorkerFlow(fullWorkerFlow); err != nil {
		return nil, state, nil, &housekeeping, final, err
	}
	if err := applyCodexContinueWorkerClosures(closedWorkerDetails); err != nil {
		return nil, state, nil, &housekeeping, final, err
	}
	emitContinueCeremonyFlowSequence("aether-continue-finalize", phase, fullWorkerFlow)
	updated.Events = append(updated.Events, continueWorkerFlowEvents(now, fullWorkerFlow)...)
	_ = store.SaveJSON("COLONY_STATE.json", updated)

	summary := fmt.Sprintf("Phase %d verified and advanced", phase.ID)
	if assessment.PartialSuccess {
		summary = fmt.Sprintf("Phase %d verified and advanced with partial operational success", phase.ID)
	}
	continueReportRel := continuePlanArtifactsPath(phase.ID, "continue.json")
	if err := store.SaveJSON(continueReportRel, codexContinueReport{
		Phase:              phase.ID,
		GeneratedAt:        now.Format(time.RFC3339),
		Manifest:           displayOptionalDataPath(manifest.Path),
		VerificationReport: displayDataPath(verificationReportRel),
		GateReport:         displayDataPath(gateReportRel),
		ReviewReport:       displayDataPath(reviewReportRel),
		Summary:            summary,
		ClosedWorkers:      closedWorkers,
		WorkerFlow:         fullWorkerFlow,
		PartialSuccess:     assessment.PartialSuccess,
		OperationalIssues:  append([]string{}, assessment.OperationalIssues...),
		Tasks:              append([]codexContinueTaskAssessment{}, assessment.Tasks...),
		Recovery:           assessment.Recovery,
		Advanced:           true,
		Completed:          final,
		Next:               nextCommand,
	}); err != nil {
		return nil, state, nextPhase, &housekeeping, final, fmt.Errorf("failed to write continue report: %w", err)
	}
	updateSessionSummary("continue-finalize", nextCommand, summary)
	result := map[string]interface{}{
		"advanced":            true,
		"completed":           final,
		"partial_success":     assessment.PartialSuccess,
		"current_phase":       updated.CurrentPhase,
		"state":               updated.State,
		"next":                nextCommand,
		"review_depth":        string(reviewDepth),
		"verification":        verification,
		"assessment":          assessment,
		"task_evidence":       assessment.Tasks,
		"gates":               gates,
		"review":              review,
		"verification_report": displayDataPath(verificationReportRel),
		"gate_report":         displayDataPath(gateReportRel),
		"review_report":       displayDataPath(reviewReportRel),
		"continue_report":     displayDataPath(continueReportRel),
		"closed_workers":      closedWorkers,
		"worker_flow":         fullWorkerFlow,
		"operational_issues":  assessment.OperationalIssues,
		"recovery":            assessment.Recovery,
		"reconciled_tasks":    assessment.ReconciledTasks,
		"signal_housekeeping": housekeeping,
	}
	if nextPhase != nil {
		result["next_phase"] = nextPhase.ID
		result["next_phase_name"] = nextPhase.Name
	}
	addOrchestratorBoundaryGuidance(result, "continue", updated, nextCommand, nil)
	return result, updated, nextPhase, &housekeeping, final, nil
}

func recordExternalContinueWorkerFlow(workerFlow []codexContinueWorkerFlowStep) error {
	if len(workerFlow) == 0 || store == nil {
		return nil
	}

	spawnTree := agent.NewSpawnTree(store, "spawn-tree.txt")
	entries, err := spawnTree.Parse()
	if err != nil {
		return fmt.Errorf("failed to read spawn tree: %w", err)
	}
	known := make(map[string]struct{}, len(entries))
	for _, entry := range entries {
		known[entry.AgentName] = struct{}{}
	}

	for _, step := range workerFlow {
		name := strings.TrimSpace(step.Name)
		if name == "" {
			continue
		}
		if _, ok := known[name]; !ok {
			if err := spawnTree.RecordSpawn("Continue", strings.TrimSpace(step.Caste), name, continueWorkerFlowTask(step), 1); err != nil {
				return fmt.Errorf("failed to record continue flow %s: %w", name, err)
			}
			known[name] = struct{}{}
		}
		if err := spawnTree.UpdateStatus(name, continueWorkerFlowStatus(step.Status), continueWorkerFlowLogSummary(step)); err != nil {
			return fmt.Errorf("failed to finalize continue flow %s: %w", name, err)
		}
	}
	return nil
}
func renderContinueWorkerOutcomeReport(root string, phase colony.Phase, step codexContinueWorkerFlowStep, recordedAt time.Time) string {
	var b strings.Builder
	b.WriteString("# Worker Outcome: ")
	b.WriteString(step.Name)
	b.WriteString("\n\n")

	b.WriteString("## Assignment\n")
	b.WriteString("- Phase: ")
	b.WriteString(strconv.Itoa(phase.ID))
	if phase.Name != "" {
		b.WriteString(" - ")
		b.WriteString(phase.Name)
	}
	b.WriteString("\n")
	b.WriteString("- Caste: ")
	b.WriteString(step.Caste)
	b.WriteString("\n")
	b.WriteString("- Task: ")
	b.WriteString(step.Task)
	b.WriteString("\n")
	if root != "" {
		b.WriteString("- Root: ")
		b.WriteString(root)
		b.WriteString("\n")
	}

	b.WriteString("\n## Recorded Outcome\n")
	b.WriteString("- Status: ")
	b.WriteString(step.Status)
	b.WriteString("\n")
	b.WriteString("- Recorded at: ")
	b.WriteString(recordedAt.UTC().Format(time.RFC3339))
	b.WriteString("\n")
	if step.Duration > 0 {
		b.WriteString("- Duration seconds: ")
		b.WriteString(strconv.FormatFloat(step.Duration, 'f', 3, 64))
		b.WriteString("\n")
	}
	if summary := strings.TrimSpace(step.Summary); summary != "" {
		b.WriteString("- Summary: ")
		b.WriteString(summary)
		b.WriteString("\n")
	}

	b.WriteString("\n## Blockers\n")
	if len(step.Blockers) > 0 {
		for _, blocker := range step.Blockers {
			b.WriteString("- ")
			b.WriteString(blocker)
			b.WriteString("\n")
		}
	} else {
		b.WriteString("none\n")
	}

	b.WriteString("\n## Report\n")
	if report := strings.TrimSpace(step.Report); report != "" {
		b.WriteString(report)
		b.WriteString("\n")
	} else {
		b.WriteString("No detailed report provided.\n")
	}

	return b.String()
}

func writeCodexContinueWorkerOutcomeReports(root string, phase colony.Phase, workerFlow []codexContinueWorkerFlowStep, recordedAt time.Time) error {
	for _, step := range workerFlow {
		name := strings.TrimSpace(step.Name)
		if name == "" {
			continue
		}
		reportRel := filepath.ToSlash(filepath.Join("build", fmt.Sprintf("phase-%d", phase.ID), "worker-reports", fmt.Sprintf("%s.md", name)))
		content := renderContinueWorkerOutcomeReport(root, phase, step, recordedAt)
		if err := store.AtomicWrite(reportRel, []byte(content)); err != nil {
			return fmt.Errorf("failed to write continue worker outcome report for %s: %w", name, err)
		}
	}
	return nil
}
