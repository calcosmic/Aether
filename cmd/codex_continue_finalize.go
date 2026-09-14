package cmd

import (
	"encoding/json"
	"errors"
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
	"github.com/calcosmic/Aether/pkg/events"
	"github.com/calcosmic/Aether/pkg/learn"
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
			return renderedErrorExit(1)
		}
		if !verificationTimeoutExplicit {
			verificationTimeout = 0
		}
		completion, err := loadExternalContinueCompletion(completionPath)
		if err != nil {
			outputError(1, err.Error(), nil)
			return renderedErrorExit(1)
		}
		// FLOOR-03 (closes the 2026-08-01 folded todo): --reconcile-task is
		// registered on THIS command too now, not only on `continue
		// --plan-only`. It unions with whatever ReconcileTaskIDs the plan
		// manifest already carries (recorded at plan-only time) rather than
		// replacing it, so an operator can record reconciliation at finalize
		// time even for a task nobody thought to reconcile earlier.
		// validateExternalContinueState (inside runCodexContinueFinalize)
		// validates the merged list against the phase exactly as it already
		// validates the plan's own list, so an unknown task ID is still
		// refused by name.
		if flagReconcileTaskIDs := normalizeCLIStringList(mustGetStringArray(cmd, "reconcile-task")); len(flagReconcileTaskIDs) > 0 {
			if activeManifest := completion.activeManifest(); activeManifest != nil {
				activeManifest.ReconcileTaskIDs = mergeReconcileTaskIDs(activeManifest.ReconcileTaskIDs, flagReconcileTaskIDs)
			}
		}
		result, state, phase, nextPhase, housekeeping, final, err := runCodexContinueFinalize(skillWorkspaceRoot(), completion, skipMissing, verificationTimeout, noLearn)
		if err != nil {
			outputError(1, err.Error(), nil)
			return renderedErrorExit(1)
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
	if err := validateFinalizerCompletionFilePath(path); err != nil {
		return codexExternalContinueCompletion{}, err
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
	now := time.Now().UTC()
	if err := validateFinalizerManifestFreshness("continue_manifest", plan.GeneratedAt, now); err != nil {
		return nil, colony.ColonyState{}, colony.Phase{}, nil, nil, false, err
	}

	state, phase, manifest, err := validateExternalContinueState(plan)
	if err != nil {
		return nil, state, phase, nil, nil, false, err
	}
	// This finalize path only VERIFIES that read-only artifact evidence was
	// already recorded (at `aether continue --plan-only` time); it never
	// records or re-hashes evidence itself. Doing so here would silently
	// overwrite the hash captured at plan-only time and destroy tamper
	// detection across the external review window (T-163.1-46) -- the one
	// property that makes this escape hatch safe.
	if err := verifyPlanReadOnlyArtifactEvidence(root, phase, plan, manifest); err != nil {
		return nil, state, phase, nil, nil, false, err
	}
	if abandoned, _, summary := detectAbandonedBuild(manifest, state); abandoned {
		return nil, state, phase, nil, nil, false, fmt.Errorf("%s", summary)
	}

	runHandle, err := beginRuntimeSpawnRun("continue", now)
	if err != nil {
		return nil, state, phase, nil, nil, false, fmt.Errorf("failed to initialize continue run: %w", err)
	}
	runStatus := "failed"
	defer func() {
		finishRuntimeSpawnRun(runHandle, runStatus, time.Now().UTC())
	}()

	// 204-15 (SC3a): the DELEGATE check lane's own episode boundary. Before
	// this plan, runCodexContinueFinalize opened and closed NO episode at
	// all -- the complete non-test call list for emitColonyLiveEpisodeEnded(
	// in cmd/ was codex_build.go, oracle_live.go, oracle_loop.go,
	// codex_plan.go, codex_continue.go; this file did not appear on it
	// (204-15-PLAN.md's own plan-time finding). continueEpisodeID reuses the
	// same run identifier finishRuntimeSpawnRun above persists, mirroring
	// the native lane's shape (cmd/codex_continue.go) line for line.
	// Registered immediately after runHandle exists and BEFORE the FIELD-04
	// replay early return just below, so every return path in this
	// function -- including that early return -- closes inside the
	// episode.
	continueEpisodeID := ""
	if runHandle != nil {
		continueEpisodeID = runHandle.Run.ID
	}
	emitColonyLiveEpisodeStarted(continueEpisodeID, events.EpisodeKindContinue)
	restoreLiveContinueEpisode := setActiveLiveContinueEpisode(continueEpisodeID)
	defer restoreLiveContinueEpisode()
	// The deferred close reuses checkEpisodeCloseRecord (cmd/episode_ledger.go)
	// -- the SAME helper the native check lane calls -- so the two lanes
	// cannot drift into recording different things (204-15-PLAN.md Task
	// 1(b)). phase.ID and runStatus are both read at DEFER-EXECUTION time,
	// after every later reassignment in this function's body.
	defer func() {
		record := checkEpisodeCloseRecord(phase.ID, runStatus)
		record.RuntimeVersion, record.PolicyVersion, record.EndedAt, record.ElapsedSeconds = episodeCloseBasics(continueEpisodeID, runStatus)
		emitColonyLiveOutcomeRecorded(continueEpisodeID, events.EpisodeKindContinue, record)
		emitColonyLiveEpisodeEndedEventOnly(continueEpisodeID, events.EpisodeKindContinue, runStatus)
	}()

	// FIELD-04 (191.1-CONTEXT.md D-07/D-08): a completed, passing
	// verification from an earlier continue-finalize run may have lost the
	// race to a colony pause and been preserved instead of discarded (see
	// cmd/advance_phase.go) -- the SAME shared mechanism runCodexContinue's
	// own entry point checks. Check for it here, before any of the
	// expensive fresh verification work below, so a resumed colony applies
	// that already-verified result instead of re-running it.
	if outcome := replayPendingContinueAdvance(state, phase, "continue-finalize", now); outcome.Handled {
		if outcome.Err != nil {
			runStatus = "failed"
			return nil, state, phase, nil, nil, false, outcome.Err
		}
		if superseded, _ := outcome.Result["superseded"].(bool); superseded {
			runStatus = "superseded"
		} else {
			runStatus = "completed"
		}
		return outcome.Result, outcome.State, outcome.Phase, outcome.NextPhase, outcome.Housekeeping, outcome.Final, nil
	}

	cleanupStaleContinueReports(phase.ID)

	workerFlow, err := mergeExternalContinueResults(*plan, completion.workerResults())
	if err != nil {
		return nil, state, phase, nil, nil, false, err
	}
	if err := persistExternalContinueHandoffs(root, phase.ID, plan.Dispatches, completion.workerResults()); err != nil {
		return nil, state, phase, nil, nil, false, err
	}
	// File this check's per-worker token record, under the continue key.
	//
	// Placed here, immediately after every reviewer and watcher has reached a
	// terminal status and before any result envelope is assembled, so the
	// record exists whether this check goes on to advance the phase or to
	// block it. A blocked check still spent what it spent.
	//
	// The build lane files its own rows under its own key
	// (cmd/codex_build_finalize.go). Neither file is opened by the other, so a
	// check can never erase what the build spent, and a later build cannot
	// erase what the check spent.
	//
	// Accounting is a record OF the check, never a gate ON it: a write that
	// fails is reported on stderr and the check still finishes.
	continueSpendOutcome, continueSpendErr := writeSpendRowsForRun(spendWriteRequest{
		Phase:      phase.ID,
		PhaseName:  phase.Name,
		Workflow:   spendWorkflowContinue,
		RepoRoot:   root,
		Platform:   buildHostPlatform(),
		RunID:      spendRunIDFromTimestamp("continue", plan.GeneratedAt),
		StartedAt:  spendRunStartFromManifest(plan.GeneratedAt),
		EndedAt:    now,
		Dispatches: spendDispatchesFromContinueFlow(workerFlow, plan.Dispatches),
	})
	if continueSpendErr != nil {
		fmt.Fprintf(os.Stderr, "warning: this check's per-worker token record could not be filed: %v\n", continueSpendErr)
	}
	for _, note := range continueSpendOutcome.Notes {
		fmt.Fprintf(os.Stderr, "\u26a0 %s\n", note)
	}
	// The runtime persists review findings itself — review castes return
	// them in result JSON and must never be briefed to run CLI commands
	// (auditor and gatekeeper have no Bash by design). Non-fatal: ledger
	// bookkeeping never blocks an advance.
	reviewFindingsPersisted, reviewFindingNotes := persistReviewFindingsToLedgers(phase.ID, phase.Name, workerFlow)
	for _, note := range reviewFindingNotes {
		fmt.Fprintf(os.Stderr, "⚠ %s\n", note)
	}

	verificationTimeout := continueFinalizeVerificationTimeout(plan, verificationTimeoutOverride)
	emitContinueVerificationStart(phase, verificationTimeout)
	verification := runCodexContinueVerificationSnapshot(root, phase, manifest, now, verificationTimeout, plan.SkipWatchers)
	var watcherFlow *codexContinueWorkerFlowStep
	if plan.SkipWatchers {
		verification.Watcher = codexWatcherVerification{Present: true, Passed: true, Status: "skipped", Worker: "skip-watchers", Summary: "watcher skipped; relying on runtime-owned verification commands"}
	} else {
		verification, watcherFlow = attachExternalContinueWatcher(verification, workerFlow)
	}
	// ReadOnlyArtifacts is threaded into these reconstructed options for
	// structural parity with the direct path and fail-fast validation
	// consistency only. It is NOT how read-only evidence reaches assessment
	// here: that already happened above, before verification ran, via the
	// persisted claims file (recorded at plan-only time and checked by
	// verifyPlanReadOnlyArtifactEvidence). Do not mistake this field for live
	// wiring into evaluatePhaseCriterionEvidence -- it plays no role in that
	// evaluation on the finalize path.
	assessment := assessCodexContinue(phase, manifest, verification, codexContinueOptions{ReconcileTaskIDs: plan.ReconcileTaskIDs, ReadOnlyArtifacts: plan.ReadOnlyArtifacts, VerificationTimeout: verificationTimeout}, now)
	verification = attachContinueClaimVerification(verification, assessment)
	var runtimeCheckpoints []autopilotCheckpointReference
	if hasRuntimeVerificationCheckpoint(verification.Criteria) {
		runtimeGeneration, generationErr := validatedRuntimeCheckpointGeneration(manifest, state, verification.Criteria)
		if generationErr != nil {
			return nil, state, phase, nil, nil, false, fmt.Errorf("failed to bind owner verification work: %w", generationErr)
		}
		runtimeCheckpoints, err = materializeRuntimeVerificationCheckpoints(phase.ID, verification.Criteria, runtimeGeneration)
		if err != nil {
			return nil, state, phase, nil, nil, false, fmt.Errorf("failed to preserve owner verification work: %w", err)
		}
	}
	priorGateResults, _ := gateResultsReadPhase(phase.ID)
	if priorGateResults == nil {
		priorGateResults = []GateCheckResult{}
	}
	// Per SAFE-03, SAFE-04: trace continue provenance against stored manifest data.
	// Rejects claims that reference missing or stale worker results.
	if err := traceContinueProvenanceForManifest(manifest); err != nil {
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
	// Evidence-based flag clearing before gates — same contract and same
	// reasoning as the fast path (see codex_continue.go): green verification
	// clears the machine-raised blockers a failed run created; chaos and
	// user flags never auto-clear.
	autoResolveVerificationBlockers(verification.ChecksPassed, phase.ID)
	gates := runCodexContinueGates(phase, manifest, verification, assessment, now, priorGateResults)
	budget := budgetFromRecoveryLog(phase.ID, 1)
	if budget == nil {
		budget = newRecoveryBudget(1)
	}
	queenDecisions := queenDecide(gates, budget, circuitBreaker, phase.ID, string(finalizeReviewDepth))
	queenState := QueenStateFile{
		Phase:          phase.ID,
		GeneratedAt:    now.Format(time.RFC3339),
		Decisions:      queenDecisions,
		BudgetSnapshot: budget,
	}
	if err := queenStateWrite(phase.ID, queenState); err != nil {
		return nil, state, phase, nil, nil, false, fmt.Errorf("failed to persist queen state: %w", err)
	}

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
		// [Rule 1 - Bug] `gates, autoResolved := ...` previously shadowed the
		// outer `gates` inside this if-block: a soft_block gate that got
		// auto-resolved here was invisible everywhere below this block
		// (review dispatch, advanceExternalContinue, the persisted report) --
		// the outer `gates.Passed` stayed stuck at its stale pre-resolution
		// value. `advanceExternalContinue` never re-checks `.Passed` itself,
		// so this went unnoticed as a functional block, but
		// runContinueAcceptVerifyAdvance (cmd/codex_verify_advance.go) DOES
		// gate on `gates.Passed` -- assigning to the outer variable with `=`
		// is required for the shared decision body to see the resolution
		// this lane already performed.
		var autoResolved []string
		gates, autoResolved = autoResolveSoftBlockGates(phase.ID, gates, resolveDepth, phase.Mode)

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

			blockedWorkerFlow := continueWorkerFlowForVerification(verification, continueReviewWorkerFlowSteps(workerFlow), watcherFlow)
			result, blockedState, err := finalizeBlockedExternalContinue(state, phase, manifest, verification, assessment, gates, nil, "", blockedWorkerFlow, now, verificationReportRel, gateReportRel, gateRecoveryInstructions, finalizeReviewDepth)
			if err != nil {
				return nil, state, phase, nil, nil, false, err
			}
			result["autopilot_signals"] = continueReviewAutopilotSignals(blockedWorkerFlow, runtimeCheckpoints)
			if superseded, _ := result["superseded"].(bool); superseded {
				// finalizeBlockedExternalContinue found the runtime state no
				// longer matches what this call was asked to record (T-188-CR-01)
				// and refused to write anything -- mirror advanceExternalContinue's
				// own supersession return a few lines below.
				runStatus = "superseded"
				return result, blockedState, phase, nil, nil, false, nil
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
	// SYN-201-04: the external finalize lane reaches its advancement verdict
	// only through the same shared decision body every other lane uses.
	// `gates` here is whatever this lane's own pre-review gate handling
	// (including its soft_block auto-resolve pass above) left it as --
	// runContinueAcceptVerifyAdvance does not re-evaluate gates, it folds
	// the already-decided report with this now-available review report.
	finalDecision := runContinueAcceptVerifyAdvance(phase, assessment, gates, &review, state)
	if !finalDecision.Advances() {
		// Hand the blocking findings to the Fixer's intake: `aether unblock
		// --dispatch` reads gate-results-<N>.json, so a review_findings gate
		// entry with each finding's suggestion as recovery options is what
		// puts the reviewers' proposed fixes in the Fixer's hands with zero
		// new plumbing.
		appendReviewFindingsGateResult(phase.ID, workerFlow, now)
		blockedWorkerFlow := continueWorkerFlowForVerification(verification, review.Workers, watcherFlow)
		result, blockedState, err := finalizeBlockedExternalContinue(state, phase, manifest, verification, assessment, gates, &review, reviewReportRel, blockedWorkerFlow, now, verificationReportRel, gateReportRel, nil, finalizeReviewDepth)
		if err != nil {
			return nil, state, phase, nil, nil, false, err
		}
		result["autopilot_signals"] = continueReviewAutopilotSignals(blockedWorkerFlow, runtimeCheckpoints)
		if superseded, _ := result["superseded"].(bool); superseded {
			runStatus = "superseded"
			return result, blockedState, phase, nil, nil, false, nil
		}
		runStatus = "blocked"
		return result, blockedState, phase, nil, nil, false, nil
	}

	// --- Learning capture (D-01, D-02, D-03, D-04) ---
	// Shared with the default continue path; see captureContinueLearning.
	runID := ""
	if runHandle != nil {
		runID = runHandle.Run.ID
	}
	captureContinueMemory(phase, workerFlow, gates, runID, noLearn, now)

	result, updated, nextPhase, housekeeping, final, err := advanceExternalContinue(root, state, phase, manifest, verification, assessment, gates, review, reviewReportRel, watcherFlow, workerFlow, now, verificationReportRel, gateReportRel, finalizeReviewDepth)
	if err != nil {
		return nil, state, phase, nil, housekeeping, final, err
	}
	result["autopilot_signals"] = continueReviewAutopilotSignals(workerFlow, runtimeCheckpoints)
	if superseded, _ := result["superseded"].(bool); superseded {
		// advanceExternalContinue found the runtime state no longer matches
		// what this call was asked to advance (188-CONTEXT.md D-04/D-05) and
		// refused to write anything -- nil error, blocked/superseded result,
		// mirroring exactly how the default continue path surfaces a
		// supersession. Return immediately, the same way the earlier gate-
		// and review-blocked branches above do: do NOT fall through to
		// phase-end consolidation or the phase-commit git commit below,
		// since nothing actually advanced.
		runStatus = "superseded"
		return result, updated, phase, nil, nil, false, nil
	}
	// D-04: phase-end consolidation fires only after advanceExternalContinue
	// returns with err == nil AND an actual advance (not a superseded
	// refusal, handled above), NOT colocated with captureContinueLearning
	// above. PhaseCompleted is written INSIDE advanceExternalContinue, so
	// only this post-return point guarantees the phase truly advanced --
	// the stricter-correct placement (RESEARCH.md assumption A1). Do not
	// "fix" this back to symmetry with the default continue path.
	consolidationSummary := runPhaseEndConsolidation(phase.ID)
	// 198.1-05/FEED-05: strong lessons (confidence >= 0.8) reach the shared
	// cross-project store at every check, not only at project close --
	// under the same AETHER_HIVE_POLICY switch seal already honours. Placed
	// immediately after consolidation and before attachConsolidationSummary,
	// mirroring the default continue lane above.
	hiveEligible, hivePromoted := promotePhaseEndInstinctsToHive(phase.ID)
	// 203-13 (BIO-08/CEC-07): outcome-weighted strength tuning runs right
	// after hive promotion, mirroring the default continue lane above --
	// same non-blocking discipline.
	outcomeTuning := runPheromoneOutcomeTuning()
	attachConsolidationSummary(result, consolidationSummary)
	attachHivePromotionSummary(result, hiveEligible, hivePromoted)
	attachPheromoneOutcomeTuningSummary(result, outcomeTuning)
	if reviewFindingsPersisted > 0 && result != nil {
		result["review_findings_persisted"] = reviewFindingsPersisted
	}
	// The phase save-point commit — strictly after advanceExternalContinue
	// returned with err == nil (PhaseCompleted is durable), same non-fatal
	// contract as consolidation above.
	attachPhaseCommitResult(result, commitPhaseAdvance(root, updated, phase))
	// advanceExternalContinue already emitted its own ceremony flow sequence
	// (containing the housekeeping step) before returning -- D-04 places
	// consolidation strictly after that call, so the learning beat cannot be
	// appended into that already-emitted batch the way the default path
	// appends it before its single emit call. Emit it as its own follow-up
	// ceremony step instead: it still reaches the ceremony event stream, not
	// only stdout (D-06/D-07).
	emitContinueCeremonyFlowSequence("aether-continue-finalize", phase, []codexContinueWorkerFlowStep{continueLearningFlowStep(consolidationSummary)})
	runStatus = "completed"
	// Written here, after every mutation above (consolidation, hive
	// promotion, phase commit), not inside advanceExternalContinue: those
	// attach* calls mutate this SAME result map after advanceExternalContinue
	// already returned it, and renderContinueVisual's Learning beat
	// (renderLearningBeat(result["consolidation"])) reads what they add. The
	// screen the owner actually sees is rendered from this fully-mutated
	// map; persisting any earlier snapshot would silently diverge from it.
	if err := writePhaseOutcomeDocument(phase.ID, result, updated); err != nil {
		fmt.Fprintf(os.Stderr, "This phase's closing summary could not be saved, so the next phase's helpers will not see it: %v\n", err)
	}
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
	if err := validateReadOnlyArtifacts(phase, plan.ReconcileTaskIDs, plan.ReadOnlyArtifacts); err != nil {
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
		if existing, exists := resultByName[name]; exists {
			if useIncoming, ok := preferCompletedResultOverTimeout(existing.Status, result.Status); ok {
				if useIncoming {
					resultByName[name] = result
				}
				continue
			}
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
		// CR-01 (189-REVIEW.md): continueExternalBriefWithHandoffSchema tells
		// every wrapper-spawned watcher and reviewer "An empty handoff is
		// rejected" (cmd/codex_continue_plan.go) -- the identical promise
		// build's brief makes -- but until this check, nothing on continue's
		// finalize chain enforced it: ValidateWorkerHandoff above only
		// format-checks VerificationStatus and explicitly accepts "" as
		// valid, so a completed worker could relay a fully empty handoff and
		// have it silently persisted. Mirrors persistExternalBuildHandoffs's
		// identical guard (cmd/codex_build_finalize.go) so both finalize
		// chains enforce the same promise their briefs state.
		//
		// `ok` here means "a result with this name was submitted at all" --
		// a genuinely missing worker (!ok) is a distinct, pre-existing
		// "timeout" placeholder path (see above), not conflated with this
		// check. A submitted "completed" result whose handoff field was
		// simply never set decodes to the identical zero value an explicit
		// empty object would (WorkerHandoff is a value, not a pointer, on
		// codexContinueExternalDispatch), so this same check already covers
		// "no handoff at all" -- there is no separate wire representation to
		// special-case.
		if ok && status == buildWorkerCompleted && codex.IsEmptyWorkerHandoff(result.Handoff) {
			return nil, fmt.Errorf("external continue result for %s completed without a handoff; completed reviewers and watchers must relay changed_files, commands_run, verification_status, and next_worker_instructions so later phases inherit their context", dispatch.Name)
		}
		summary := strings.TrimSpace(result.Summary)
		blockers := uniqueSortedStrings(result.Blockers)
		if summary == "" && len(blockers) > 0 {
			summary = strings.Join(blockers, "; ")
		}
		step := codexContinueWorkerFlowStep{
			Stage:    dispatch.Stage,
			Caste:    dispatch.Caste,
			Name:     dispatch.Name,
			Task:     dispatch.Task,
			Status:   status,
			Summary:  summary,
			Blockers: blockers,
			Duration: result.Duration,
			Report:   strings.TrimSpace(result.Report),
			// Keep legacy rows raw until normalizeContinueReviewEvidence has
			// validated their human-readable body. Merging here would discard a
			// suggestion-only row before it could produce fail-closed evidence.
			Findings:        append(append([]codexReviewFinding{}, result.Findings...), result.Issues...),
			Recommendations: uniqueSortedStrings(result.Recommendations),
			WeakSpots:       uniqueSortedStrings(result.WeakSpots),
			EdgeCases:       uniqueSortedStrings(result.EdgeCases),
			ReusableLessons: uniqueSortedStrings(result.ReusableLessons),
		}
		// artifacts.review is authoritative when present. Legacy top-level
		// findings remain compatible only for older workers that did not emit
		// the artifact at all.
		step = normalizeContinueReviewEvidence(step, result.Artifacts)
		flow = append(flow, step)
	}
	return flow, nil
}

// normalizeContinueReviewEvidence applies the same artifact contract to an
// in-process WorkerResult and a wrapper completion. An explicit review
// artifact is authoritative: prose and legacy top-level fields cannot
// override it, and malformed values become evidence errors rather than a
// fabricated zero score.
func normalizeContinueReviewEvidence(step codexContinueWorkerFlowStep, artifacts map[string]json.RawMessage) codexContinueWorkerFlowStep {
	raw, explicit := artifacts["review"]
	if !explicit {
		validFindings := make([]codexReviewFinding, 0, len(step.Findings))
		for index, finding := range step.Findings {
			severity := strings.ToUpper(strings.TrimSpace(finding.Severity))
			if !validReviewArtifactSeverity(severity) {
				step.EvidenceErrors = append(step.EvidenceErrors, fmt.Sprintf("%s legacy findings[%d].severity must be CRITICAL, HIGH, MEDIUM, LOW, or INFO", step.Name, index))
				continue
			}
			finding.Severity = severity
			if strings.TrimSpace(finding.Title) == "" && strings.TrimSpace(finding.Description) == "" {
				step.EvidenceErrors = append(step.EvidenceErrors, fmt.Sprintf("%s legacy findings[%d] must include a non-empty title or description", step.Name, index))
				continue
			}
			validFindings = append(validFindings, finding)
		}
		step.Findings = mergeCodexReviewFindings(validFindings)
		if strings.EqualFold(strings.TrimSpace(step.Caste), "auditor") && continueWorkerFlowStatus(step.Status) == buildWorkerCompleted {
			step.EvidenceErrors = uniqueSortedStrings(append(step.EvidenceErrors, fmt.Sprintf("%s artifacts.review is required for a completed Auditor", step.Name)))
		}
		step.EvidenceErrors = uniqueSortedStrings(step.EvidenceErrors)
		return step
	}

	step.Findings = nil
	step.OverallScore = nil
	step.EvidenceErrors = nil
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" || trimmed == "null" || !strings.HasPrefix(trimmed, "{") {
		step.EvidenceErrors = []string{fmt.Sprintf("%s artifacts.review must be a JSON object", step.Name)}
		return step
	}

	var artifact map[string]json.RawMessage
	if err := json.Unmarshal(raw, &artifact); err != nil {
		step.EvidenceErrors = []string{fmt.Sprintf("%s artifacts.review is malformed: %v", step.Name, err)}
		return step
	}
	if artifact == nil {
		step.EvidenceErrors = []string{fmt.Sprintf("%s artifacts.review must be a JSON object", step.Name)}
		return step
	}

	scoreRaw, scorePresent := artifact["overall_score"]
	if !scorePresent {
		if strings.EqualFold(strings.TrimSpace(step.Caste), "auditor") && continueWorkerFlowStatus(step.Status) == buildWorkerCompleted {
			step.EvidenceErrors = append(step.EvidenceErrors, fmt.Sprintf("%s artifacts.review overall_score is required for a completed Auditor", step.Name))
		}
	} else {
		var score int
		if strings.TrimSpace(string(scoreRaw)) == "null" || json.Unmarshal(scoreRaw, &score) != nil {
			step.EvidenceErrors = append(step.EvidenceErrors, fmt.Sprintf("%s artifacts.review overall_score must be an integer", step.Name))
		} else if score < 0 || score > 100 {
			step.EvidenceErrors = append(step.EvidenceErrors, fmt.Sprintf("%s artifacts.review overall_score must be between 0 and 100", step.Name))
		} else if strings.EqualFold(strings.TrimSpace(step.Caste), "auditor") && continueWorkerFlowStatus(step.Status) == buildWorkerCompleted {
			step.OverallScore = &score
		}
	}

	structuredFindings := []codexReviewFinding{}
	for _, field := range []string{"findings", "issues"} {
		fieldRaw, present := artifact[field]
		if !present {
			continue
		}
		var findings []codexReviewFinding
		if err := json.Unmarshal(fieldRaw, &findings); err != nil {
			step.EvidenceErrors = append(step.EvidenceErrors, fmt.Sprintf("%s artifacts.review %s must be an array: %v", step.Name, field, err))
			continue
		}
		structuredFindings = append(structuredFindings, findings...)
	}

	validFindings := make([]codexReviewFinding, 0, len(structuredFindings))
	for index, finding := range structuredFindings {
		severity := strings.ToUpper(strings.TrimSpace(finding.Severity))
		if !validReviewArtifactSeverity(severity) {
			step.EvidenceErrors = append(step.EvidenceErrors, fmt.Sprintf("%s artifacts.review findings[%d].severity must be CRITICAL, HIGH, MEDIUM, LOW, or INFO", step.Name, index))
			continue
		}
		finding.Severity = severity
		if strings.TrimSpace(finding.Title) == "" && strings.TrimSpace(finding.Description) == "" {
			step.EvidenceErrors = append(step.EvidenceErrors, fmt.Sprintf("%s artifacts.review findings[%d] must include a non-empty title or description", step.Name, index))
			continue
		}
		validFindings = append(validFindings, finding)
	}
	step.Findings = mergeCodexReviewFindings(validFindings)
	step.EvidenceErrors = uniqueSortedStrings(step.EvidenceErrors)
	return step
}

func continueReviewEvidenceBlockingIssues(step codexContinueWorkerFlowStep) []string {
	caste := strings.TrimSpace(step.Caste)
	if caste == "" {
		caste = "reviewer"
	}
	name := strings.TrimSpace(step.Name)
	if name == "" {
		name = "unnamed"
	}
	blockers := make([]string, 0, len(step.EvidenceErrors))
	for _, evidenceError := range step.EvidenceErrors {
		if reason := strings.TrimSpace(evidenceError); reason != "" {
			blockers = append(blockers, fmt.Sprintf("%s %s review evidence is invalid: %s", caste, name, reason))
		}
	}
	return uniqueSortedStrings(blockers)
}

func validReviewArtifactSeverity(severity string) bool {
	switch severity {
	case "CRITICAL", "HIGH", "MEDIUM", "LOW", "INFO":
		return true
	default:
		return false
	}
}

func continueReviewAutopilotSignals(workerFlow []codexContinueWorkerFlowStep, checkpointGroups ...[]autopilotCheckpointReference) codexContinueAutopilotSignals {
	signals := codexContinueAutopilotSignals{
		Evaluations: []autopilotTriggerEvaluation{
			continueAutopilotTriggerEvaluation(autopilotTriggerAuditorScoreBelowFloor, false, nil),
			continueAutopilotTriggerEvaluation(autopilotTriggerCriticalReviewFinding, false, nil),
			continueAutopilotTriggerEvaluation(autopilotTriggerRuntimeVerificationNeeded, false, nil),
			continueAutopilotTriggerEvaluation(autopilotTriggerVisualCheckpointNeeded, false, nil),
		},
	}
	var scoreWorker string
	criticalFindings := []codexReviewFinding{}
	for _, step := range workerFlow {
		signals.Findings = append(signals.Findings, step.Findings...)
		signals.EvidenceErrors = append(signals.EvidenceErrors, step.EvidenceErrors...)
		if step.OverallScore != nil && (signals.AuditorScore == nil || *step.OverallScore < *signals.AuditorScore) {
			score := *step.OverallScore
			signals.AuditorScore = &score
			scoreWorker = step.Name
		}
		for _, finding := range step.Findings {
			if strings.EqualFold(strings.TrimSpace(finding.Severity), "CRITICAL") {
				criticalFindings = append(criticalFindings, finding)
			}
		}
	}
	signals.Findings = mergeCodexReviewFindings(signals.Findings)
	signals.EvidenceErrors = uniqueSortedStrings(signals.EvidenceErrors)
	if signals.AuditorScore != nil {
		signals.Evaluations[0] = continueAutopilotTriggerEvaluation(
			autopilotTriggerAuditorScoreBelowFloor,
			*signals.AuditorScore < 60,
			map[string]interface{}{"worker": scoreWorker, "overall_score": *signals.AuditorScore, "floor": 60},
		)
	}
	if len(criticalFindings) > 0 {
		signals.Evaluations[1] = continueAutopilotTriggerEvaluation(
			autopilotTriggerCriticalReviewFinding,
			true,
			map[string]interface{}{"count": len(criticalFindings), "findings": criticalFindings},
		)
	}
	seenCheckpoints := map[string]bool{}
	for _, group := range checkpointGroups {
		for _, checkpoint := range group {
			if strings.TrimSpace(checkpoint.ID) == "" || seenCheckpoints[checkpoint.ID] {
				continue
			}
			seenCheckpoints[checkpoint.ID] = true
			signals.Checkpoints = append(signals.Checkpoints, checkpoint)
		}
	}
	for index, checkpointType := range []string{autopilotCheckpointTypeRuntimeVerification, autopilotCheckpointTypeVisual} {
		matching := []autopilotCheckpointReference{}
		for _, checkpoint := range signals.Checkpoints {
			if checkpoint.Type == checkpointType {
				matching = append(matching, checkpoint)
			}
		}
		if len(matching) == 0 {
			continue
		}
		code := autopilotTriggerRuntimeVerificationNeeded
		if checkpointType == autopilotCheckpointTypeVisual {
			code = autopilotTriggerVisualCheckpointNeeded
		}
		signals.Evaluations[index+2] = continueAutopilotTriggerEvaluation(code, true, map[string]interface{}{
			"count":       len(matching),
			"checkpoints": matching,
		})
	}
	return signals
}

func continueAutopilotTriggerEvaluation(code autopilotTriggerCode, active bool, evidence map[string]interface{}) autopilotTriggerEvaluation {
	spec, _ := autopilotTriggerSpecByCode(code)
	return autopilotTriggerEvaluation{Spec: spec, Active: active, Evidence: evidence}
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
			if finding.Description == "" {
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
			Passed:  isSuccessfulExternalBuildStatus(status),
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

// appendReviewFindingsGateResult writes a review_findings entry into the
// phase's gate-results file when review workers returned blocking findings,
// carrying each finding's suggestion as a recovery option — the bridge that
// puts reviewer-proposed fixes into `aether unblock --dispatch`'s Fixer
// context. Best-effort: gate-results bookkeeping never blocks anything.
func appendReviewFindingsGateResult(phaseID int, workerFlow []codexContinueWorkerFlowStep, now time.Time) {
	details := []string{}
	options := []string{"Run aether unblock --dispatch to dispatch the Fixer against these findings"}
	fixHint := ""
	for _, step := range workerFlow {
		for _, finding := range step.Findings {
			if !strings.EqualFold(finding.Severity, "CRITICAL") {
				continue
			}
			desc := strings.TrimSpace(finding.Description)
			if desc == "" {
				desc = strings.TrimSpace(finding.Title)
			}
			if desc == "" {
				continue
			}
			details = append(details, fmt.Sprintf("%s: %s", step.Name, desc))
			if suggestion := strings.TrimSpace(finding.Suggestion); suggestion != "" {
				options = append(options, fmt.Sprintf("Apply %s's fix: %s", step.Name, suggestion))
				if fixHint == "" {
					fixHint = suggestion
				}
			}
		}
	}
	if len(details) == 0 {
		return
	}
	if fixHint == "" {
		fixHint = "No reviewer supplied a fix — aether unblock --dispatch dispatches the Fixer to propose one"
	}
	entries, err := gateResultsReadPhase(phaseID)
	if err != nil || entries == nil {
		entries = []GateCheckResult{}
	}
	entries = append(entries, GateCheckResult{
		Name:            "review_findings",
		Status:          "failed",
		Detail:          strings.Join(details, "; "),
		FixHint:         fixHint,
		RecoveryOptions: options,
		Timestamp:       now.Format(time.RFC3339),
	})
	_ = gateResultsWritePhase(phaseID, entries)
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
		if evidenceBlockers := continueReviewEvidenceBlockingIssues(step); len(evidenceBlockers) > 0 {
			report.Passed = false
			blockers = append(blockers, evidenceBlockers...)
		}
		if isSuccessfulExternalBuildStatus(status) {
			// Severity is authoritative. Only CRITICAL structured findings
			// stop the line; the legacy worker-supplied blocking bit remains
			// reportable metadata and cannot promote High or lower evidence.
			for _, finding := range step.Findings {
				if !strings.EqualFold(finding.Severity, "CRITICAL") {
					continue
				}
				desc := strings.TrimSpace(finding.Description)
				if desc == "" {
					desc = strings.TrimSpace(finding.Title)
				}
				if desc == "" {
					continue
				}
				report.Passed = false
				if suggestion := strings.TrimSpace(finding.Suggestion); suggestion != "" {
					blockers = append(blockers, fmt.Sprintf("%s blocking finding: %s (fix: %s)", step.Name, desc, suggestion))
				} else {
					blockers = append(blockers, fmt.Sprintf("%s blocking finding: %s (next step: aether unblock --dispatch — dispatch the Fixer)", step.Name, desc))
				}
			}
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
	// T-188-CR-01: this used to be `blockedState := state; ...;
	// store.SaveJSON("COLONY_STATE.json", blockedState)` -- a raw, non-atomic
	// snapshot of `state`, the value the caller (runCodexContinueFinalize)
	// captured once at its very top, before verification, gates, and review
	// ran. Any concurrent write to COLONY_STATE.json during that window (an
	// operator pause, a background writer) was silently discarded and
	// replaced wholesale -- the identical bug class 188-02 fixed for the
	// successful-advance branch of this same file (advanceExternalContinue).
	// Route this sibling blocked-path write through the same atomic
	// read-modify-write + supersession discipline: mutate fields on the
	// value UpdateJSONAtomically just freshly read, never reassign it
	// wholesale from the stale `state` parameter. Mirrors
	// recordBlockedContinueWorkerFlow (cmd/codex_continue.go), the
	// correctly-guarded blocked path on the default (non-finalize) continue
	// flow.
	var updated colony.ColonyState
	if err := store.UpdateJSONAtomically("COLONY_STATE.json", &updated, func() error {
		if err := validateRuntimeStateStillCurrent(updated, phase.ID, state.BuildStartedAt, colony.StateEXECUTING, colony.StateBUILT); err != nil {
			return err
		}
		updated.Events = append(trimmedEvents(updated.Events), continueWorkerFlowEvents(now, workerFlow)...)
		updated.Events = append(updated.Events, fmt.Sprintf("%s|continue_blocked|continue-finalize|Continue blocked before advancement", now.Format(time.RFC3339)))
		return nil
	}); err != nil {
		if errors.Is(err, errRuntimeStateSuperseded) {
			// nil error: mirrors advanceExternalContinue's own supersession
			// handling a few lines away in this same file -- the caller
			// treats this as a completed-but-superseded result, not a hard
			// failure.
			return continueSupersededResult(state, phase, err), state, nil
		}
		return nil, state, fmt.Errorf("failed to save colony state: %w", err)
	}
	blockedState := updated
	emitContinueCeremonyFlowSequence("aether-continue-finalize", phase, workerFlow)
	updateSessionSummary("continue-finalize", nextCommand, summary)
	result := map[string]interface{}{
		"advanced":             false,
		"blocked":              true,
		"partial_success":      assessment.PartialSuccess,
		"current_phase":        blockedState.CurrentPhase,
		"phase_name":           phase.Name,
		"continued_phase":      phase.ID,
		"continued_phase_name": phase.Name,
		"state":                blockedState.State,
		"next":                 nextCommand,
		"review_depth":         string(reviewDepth),
		"verification":         verification,
		"assessment":           assessment,
		"task_evidence":        assessment.Tasks,
		"gates":                gates,
		"verification_report":  displayDataPath(verificationReportRel),
		"gate_report":          displayDataPath(gateReportRel),
		"continue_report":      displayDataPath(continueReportRel),
		"worker_flow":          workerFlow,
		"autopilot_signals":    continueReviewAutopilotSignals(workerFlow),
		"operational_issues":   assessment.OperationalIssues,
		"recovery":             assessment.Recovery,
		"reconciled_tasks":     assessment.ReconciledTasks,
		"blocking_issues":      blockers,
		"plan_revision_option": planRevisionRecommendation(
			colony.PlanRevisionVerificationFailure,
			fmt.Sprintf("Verification or review blocked phase %d: %s", phase.ID, summary),
			displayDataPath(verificationReportRel),
			displayOptionalDataPath(reviewReportRel),
		),
	}
	if review != nil {
		result["review"] = *review
		result["review_report"] = displayDataPath(reviewReportRel)
	}
	if len(gateRecoveryInstructions) > 0 {
		result["recovery_instructions"] = gateRecoveryInstructions
	}
	addOrchestratorBoundaryGuidance(result, "continue", blockedState, nextCommand, nil)
	closeLifecycleRun(result, blockedState, "continue")
	if err := writePhaseOutcomeDocument(phase.ID, result, blockedState); err != nil {
		fmt.Fprintf(os.Stderr, "This phase's closing summary could not be saved, so the next phase's helpers will not see it: %v\n", err)
	}
	return result, blockedState, nil
}

func advanceExternalContinue(root string, state colony.ColonyState, phase colony.Phase, manifest codexContinueManifest, verification codexContinueVerificationReport, assessment codexContinueAssessment, gates codexContinueGateReport, review codexContinueReviewReport, reviewReportRel string, watcherFlow *codexContinueWorkerFlowStep, workerFlow []codexContinueWorkerFlowStep, now time.Time, verificationReportRel, gateReportRel string, reviewDepth colony.VerificationDepth) (map[string]interface{}, colony.ColonyState, *colony.Phase, *signalHousekeepingResult, bool, error) {
	closedWorkerDetails := plannedCodexContinueClosedWorkers(manifest, assessment)
	closedWorkers := closedWorkerNames(closedWorkerDetails)

	// advancePhase (cmd/advance_phase.go) is the one shared atomic core both
	// aether continue and aether continue-finalize call -- see
	// 188-CONTEXT.md D-04/D-05/D-06. Unlike the block this replaced, it
	// re-validates that the phase and build this call was asked to advance
	// are still the ones actually in progress before writing anything, and
	// it has no full colony.ColonyState value in scope to clobber the fresh
	// read with (the `updated = state` bug this file used to have).
	advanceResult, err := advancePhase(advancePhaseParams{
		PhaseID:                phase.ID,
		ExpectedBuildStartedAt: state.BuildStartedAt,
		AllowedStates:          []colony.State{colony.StateEXECUTING, colony.StateBUILT},
		Source:                 "continue-finalize",
		Now:                    now,
	})
	if err != nil {
		if errors.Is(err, errRuntimeStateSuperseded) {
			// FIELD-04: if this supersession is specifically because the
			// colony is paused, preserve this already-computed, already-
			// passing payload for replay after resume instead of discarding
			// it (cmd/advance_phase.go) -- the SAME shared function
			// runCodexContinue's own advancePhase call site uses, per
			// Pattern 5 (one mechanism, not two per-caller copies). Any
			// other supersession reason preserves nothing -- discard
			// exactly as before.
			preserveIfPausedSupersession(phase.ID, state.BuildStartedAt, "continue-finalize", now, pendingContinueAdvancePayload{
				Verification: verification,
				Assessment:   assessment,
				Gates:        gates,
				Review:       review,
				ReviewDepth:  reviewDepth,
			})
			// nil error: the caller (runCodexContinueFinalize) treats this as
			// a completed-but-blocked result rather than a hard failure,
			// mirroring exactly how the default continue path surfaces a
			// supersession via continueSupersededResult.
			return continueSupersededResult(state, phase, err), state, nil, nil, false, nil
		}
		return nil, state, nil, nil, false, fmt.Errorf("failed to atomically advance phase: %w", err)
	}
	updated := advanceResult.Updated
	nextPhase := advanceResult.NextPhase
	nextCommand := advanceResult.NextCommand
	final := advanceResult.Final

	housekeeping, housekeepingErr := continueSignalHousekeeper(now, updated)
	if housekeepingErr != nil {
		return nil, state, nil, nil, final, housekeepingErr
	}
	if err := continueContextUpdater(phase, manifest, closedWorkerDetails, now); err != nil {
		return nil, state, nil, &housekeeping, final, err
	}
	fullWorkerFlow := continueWorkerFlowForVerification(verification, review.Workers, watcherFlow)
	fullWorkerFlow = append(fullWorkerFlow, continueHousekeepingFlowStep(housekeeping))
	if err := recordExternalContinueWorkerFlow(fullWorkerFlow); err != nil {
		return nil, state, nil, &housekeeping, final, err
	}
	if err := applyCodexContinueWorkerClosures(closedWorkerDetails); err != nil {
		return nil, state, nil, &housekeeping, final, err
	}
	emitContinueCeremonyFlowSequence("aether-continue-finalize", phase, fullWorkerFlow)
	// Persist side-effect events (review, housekeeping) into colony state.
	// This is a best-effort append; the core advancement was already committed
	// by advancePhase above. appendRuntimeStateEventsIfCurrent re-checks
	// currency before appending rather than blindly overwriting with `updated`
	// -- the same non-atomic-write hazard D-06 removes here that the default
	// continue path (cmd/codex_continue.go) never had.
	flowEvents := continueWorkerFlowEvents(now, fullWorkerFlow)
	updated.Events = append(updated.Events, flowEvents...)
	_ = appendRuntimeStateEventsIfCurrent(updated, flowEvents)

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
		"advanced":             true,
		"completed":            final,
		"partial_success":      assessment.PartialSuccess,
		"current_phase":        updated.CurrentPhase,
		"continued_phase":      phase.ID,
		"continued_phase_name": phase.Name,
		"state":                updated.State,
		"next":                 nextCommand,
		"review_depth":         string(reviewDepth),
		"verification":         verification,
		"assessment":           assessment,
		"task_evidence":        assessment.Tasks,
		"gates":                gates,
		"review":               review,
		"verification_report":  displayDataPath(verificationReportRel),
		"gate_report":          displayDataPath(gateReportRel),
		"review_report":        displayDataPath(reviewReportRel),
		"continue_report":      displayDataPath(continueReportRel),
		"closed_workers":       closedWorkers,
		"worker_flow":          fullWorkerFlow,
		"autopilot_signals":    continueReviewAutopilotSignals(fullWorkerFlow),
		"operational_issues":   assessment.OperationalIssues,
		"recovery":             assessment.Recovery,
		"reconciled_tasks":     assessment.ReconciledTasks,
		"signal_housekeeping":  housekeeping,
	}
	if nextPhase != nil {
		result["next_phase"] = nextPhase.ID
		result["next_phase_name"] = nextPhase.Name
	}
	addOrchestratorBoundaryGuidance(result, "continue", updated, nextCommand, nil)
	closeLifecycleRun(result, updated, "continue")
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

// buildLearningContent extracts structured learning content from worker flow steps.
// It aggregates findings, reusable lessons, recommendations, weak spots, and edge cases
// from completed workers into a multi-line content string.
func buildLearningContent(phase colony.Phase, workerFlow []codexContinueWorkerFlowStep) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Phase %d: %s\n", phase.ID, phase.Name))

	var completedWorkers []codexContinueWorkerFlowStep
	for _, step := range workerFlow {
		if step.Status == "completed" {
			completedWorkers = append(completedWorkers, step)
		}
	}

	if len(completedWorkers) == 0 {
		b.WriteString("\nNo workers completed successfully.\n")
		return b.String()
	}

	b.WriteString(fmt.Sprintf("\nWorkers completed: %d\n", len(completedWorkers)))

	for _, step := range completedWorkers {
		b.WriteString(fmt.Sprintf("\n--- %s (%s) ---\n", step.Name, step.Caste))
		if step.Task != "" {
			b.WriteString(fmt.Sprintf("Task: %s\n", step.Task))
		}
		if len(step.Findings) > 0 {
			b.WriteString("Findings:\n")
			for _, f := range step.Findings {
				b.WriteString(fmt.Sprintf("  - %s\n", f.Description))
			}
		}
		if len(step.ReusableLessons) > 0 {
			b.WriteString("Reusable Lessons:\n")
			for _, l := range step.ReusableLessons {
				b.WriteString(fmt.Sprintf("  - %s\n", l))
			}
		}
		if len(step.Recommendations) > 0 {
			b.WriteString("Recommendations:\n")
			for _, r := range step.Recommendations {
				b.WriteString(fmt.Sprintf("  - %s\n", r))
			}
		}
		if len(step.WeakSpots) > 0 {
			b.WriteString("Weak Spots:\n")
			for _, w := range step.WeakSpots {
				b.WriteString(fmt.Sprintf("  - %s\n", w))
			}
		}
		if len(step.EdgeCases) > 0 {
			b.WriteString("Edge Cases:\n")
			for _, e := range step.EdgeCases {
				b.WriteString(fmt.Sprintf("  - %s\n", e))
			}
		}
	}

	return b.String()
}

// captureContinueLearning is the single durable-learning capture point for
// BOTH continue paths. It previously lived as a closure inside
// runCodexContinueFinalizeExternal — a command the continue wrapper explicitly
// forbids on the default fast path — which meant learning had never once fired
// in normal daily use. The colony's entire learning story (pkg/learn capture,
// hypothesis promotion, auto-skill creation) was dark unless the user opted
// into heavy verification every phase.
//
// Eligibility is unchanged: all workers succeeded AND gates passed AND
// learning enabled. runID may be empty; a deterministic fallback is derived.
func captureContinueLearning(phase colony.Phase, workerFlow []codexContinueWorkerFlowStep, gates codexContinueGateReport, runID string, noLearn bool, now time.Time) {

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

	if runID == "" {
		runID = fmt.Sprintf("run_%d_%s", phase.ID, now.Format("20060102_150405"))
	}

	evidence := learn.CollectEvidence(
		runID, phase.ID, workerResults,
		learn.GateResult{Passed: gatesPassed, Total: len(gates.Checks)},
		"repo-local",
	)

	// Build learning content from deep worker extraction
	content := buildLearningContent(phase, workerFlow)

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
		// Add (below) only assigns an id on ITS OWN parameter copy when
		// entry.ID is empty -- Go's pass-by-value means that assignment
		// never reaches this caller's entry, so the id difficulty-triggered
		// skill proposals need to name their own source learning entry by
		// (SkillProposal.LearningEntryID, LEARN-07) is pre-assigned here,
		// before Add ever runs, rather than left for Add to silently drop.
		ID:             generateSignalID(),
		Content:        scanResult.Clean, // use cleaned content
		Evidence:       evidence,
		Classification: classification,
		Phase:          phase.ID,
		Confidence:     evidence.Confidence,
		Status:         learn.StatusHypothesis,
	}
	if err := learnStore.Add(entry); err != nil {
		// Non-blocking: learning failure must not prevent phase advancement
		fmt.Fprintf(os.Stderr, "warning: failed to capture learning: %v\n", err)
	} else {
		// Phase 91 / LEARN-07 (204-09-PLAN.md Task 3): auto-skill proposal
		// hook (AUTO-01). Only fires after successful learning capture for
		// difficult verified tasks. Reads auto_skill_mode config to
		// determine whether a proposal is raised at all (off/propose/auto,
		// default propose) -- neither mode creates an active skill
		// directly any more; both route through the SAME owner
		// tick-to-approve queue via colonySkillProposalSink
		// (cmd/suggest_approve.go).
		mode := learn.LoadAutoSkillMode(store.BasePath())
		if err := learn.AutoCreateSkillIfDifficult(entry, mode, colonySkillProposalSink{}); err != nil {
			// Non-blocking: a proposal failure must not prevent phase advancement
			fmt.Fprintf(os.Stderr, "warning: failed to raise skill proposal: %v\n", err)
		}
	}
}
