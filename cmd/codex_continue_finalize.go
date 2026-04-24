package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/agent"
	"github.com/calcosmic/Aether/pkg/colony"
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
		completion, err := loadExternalContinueCompletion(completionPath)
		if err != nil {
			outputError(1, err.Error(), nil)
			return nil
		}
		result, state, phase, nextPhase, housekeeping, final, err := runCodexContinueFinalize(skillWorkspaceRoot(), completion)
		if err != nil {
			outputError(1, err.Error(), nil)
			return nil
		}
		if blocked, _ := result["blocked"].(bool); blocked {
			outputWorkflow(result, renderContinueBlockedVisual(state, phase, result))
			return nil
		}
		outputWorkflow(result, renderContinueVisual(state, phase, housekeeping, final, nextPhase, result))
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

func runCodexContinueFinalize(root string, completion codexExternalContinueCompletion) (map[string]interface{}, colony.ColonyState, colony.Phase, *colony.Phase, *signalHousekeepingResult, bool, error) {
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
	if len(plan.Dispatches) == 0 {
		return nil, colony.ColonyState{}, colony.Phase{}, nil, nil, false, fmt.Errorf("continue_manifest contains no dispatches")
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

	verification := runCodexContinueVerificationSnapshot(root, phase, manifest, now)
	verification, watcherFlow := attachExternalContinueWatcher(verification, workerFlow)
	assessment := assessCodexContinue(phase, manifest, verification, codexContinueOptions{ReconcileTaskIDs: plan.ReconcileTaskIDs}, now)
	verification = attachContinueClaimVerification(verification, assessment)
	gates := runCodexContinueGates(phase, manifest, verification, assessment, now)

	verificationReportRel := continuePlanArtifactsPath(phase.ID, "verification.json")
	gateReportRel := continuePlanArtifactsPath(phase.ID, "gates.json")
	if err := store.SaveJSON(verificationReportRel, verification); err != nil {
		return nil, state, phase, nil, nil, false, fmt.Errorf("failed to write verification report: %w", err)
	}
	if err := store.SaveJSON(gateReportRel, gates); err != nil {
		return nil, state, phase, nil, nil, false, fmt.Errorf("failed to write gate report: %w", err)
	}

	if !gates.Passed {
		result, blockedState, err := finalizeBlockedExternalContinue(state, phase, manifest, verification, assessment, gates, nil, "", workerFlow, now, verificationReportRel, gateReportRel)
		if err != nil {
			return nil, state, phase, nil, nil, false, err
		}
		runStatus = "blocked"
		return result, blockedState, phase, nil, nil, false, nil
	}

	review := externalContinueReviewReport(phase.ID, workerFlow, now)
	reviewReportRel := continuePlanArtifactsPath(phase.ID, "review.json")
	if err := store.SaveJSON(reviewReportRel, review); err != nil {
		return nil, state, phase, nil, nil, false, fmt.Errorf("failed to write review report: %w", err)
	}
	if !review.Passed {
		result, blockedState, err := finalizeBlockedExternalContinue(state, phase, manifest, verification, assessment, gates, &review, reviewReportRel, workerFlow, now, verificationReportRel, gateReportRel)
		if err != nil {
			return nil, state, phase, nil, nil, false, err
		}
		runStatus = "blocked"
		return result, blockedState, phase, nil, nil, false, nil
	}

	result, updated, nextPhase, housekeeping, final, err := advanceExternalContinue(root, state, phase, manifest, verification, assessment, gates, review, reviewReportRel, watcherFlow, workerFlow, now, verificationReportRel, gateReportRel)
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
			return nil, fmt.Errorf("missing external continue result for %s", dispatch.Name)
		}
		if err := validateExternalContinueIdentity(dispatch, result); err != nil {
			return nil, err
		}
		status := normalizeExternalBuildStatus(result.Status)
		if !isTerminalExternalBuildStatus(status) {
			return nil, fmt.Errorf("external continue result for %s has non-terminal status %q", dispatch.Name, result.Status)
		}
		summary := strings.TrimSpace(result.Summary)
		blockers := uniqueSortedStrings(result.Blockers)
		if summary == "" && len(blockers) > 0 {
			summary = strings.Join(blockers, "; ")
		}
		flow = append(flow, codexContinueWorkerFlowStep{
			Stage:   dispatch.Stage,
			Caste:   dispatch.Caste,
			Name:    dispatch.Name,
			Task:    dispatch.Task,
			Status:  status,
			Summary: summary,
		})
	}
	return flow, nil
}

func validateExternalContinueIdentity(dispatch codexContinueExternalDispatch, result codexContinueExternalDispatch) error {
	if value := strings.TrimSpace(result.Caste); value != "" && !strings.EqualFold(value, dispatch.Caste) {
		return fmt.Errorf("external continue result %s caste = %q, want %q", dispatch.Name, value, dispatch.Caste)
	}
	if value := strings.TrimSpace(result.Stage); value != "" && !strings.EqualFold(value, dispatch.Stage) {
		return fmt.Errorf("external continue result %s stage = %q, want %q", dispatch.Name, value, dispatch.Stage)
	}
	if value := strings.TrimSpace(result.TaskID); value != "" && value != strings.TrimSpace(dispatch.TaskID) {
		return fmt.Errorf("external continue result %s task_id = %q, want %q", dispatch.Name, value, dispatch.TaskID)
	}
	if result.Wave > 0 && dispatch.Wave > 0 && result.Wave != dispatch.Wave {
		return fmt.Errorf("external continue result %s wave = %d, want %d", dispatch.Name, result.Wave, dispatch.Wave)
	}
	return nil
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
		verification.Watcher = watcher
		if !watcher.Passed {
			verification.ChecksPassed = false
			verification.Passed = false
			verification.BlockingIssues = uniqueSortedStrings(append(verification.BlockingIssues, summary))
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

func externalContinueReviewReport(phaseID int, workerFlow []codexContinueWorkerFlowStep, now time.Time) codexContinueReviewReport {
	report := codexContinueReviewReport{
		Phase:       phaseID,
		GeneratedAt: now.Format(time.RFC3339),
		Workers:     []codexContinueWorkerFlowStep{},
		Passed:      true,
	}
	blockers := []string{}
	for _, step := range workerFlow {
		if strings.TrimSpace(step.Stage) != "review" {
			continue
		}
		report.Workers = append(report.Workers, step)
		status := continueWorkerFlowStatus(step.Status)
		if status != "completed" && status != "manually-reconciled" {
			report.Passed = false
			blockers = append(blockers, fmt.Sprintf("%s review did not complete cleanly: %s", step.Name, status))
		}
		if summary := strings.TrimSpace(step.Summary); summary != "" && status != "completed" {
			blockers = append(blockers, fmt.Sprintf("%s reported blocker: %s", step.Name, summary))
		}
	}
	if len(report.Workers) != len(codexContinueReviewSpecs) {
		report.Passed = false
		blockers = append(blockers, fmt.Sprintf("expected %d review workers, got %d", len(codexContinueReviewSpecs), len(report.Workers)))
	}
	report.BlockingIssues = uniqueSortedStrings(blockers)
	report.Passed = report.Passed && len(report.BlockingIssues) == 0
	return report
}

func finalizeBlockedExternalContinue(state colony.ColonyState, phase colony.Phase, manifest codexContinueManifest, verification codexContinueVerificationReport, assessment codexContinueAssessment, gates codexContinueGateReport, review *codexContinueReviewReport, reviewReportRel string, workerFlow []codexContinueWorkerFlowStep, now time.Time, verificationReportRel, gateReportRel string) (map[string]interface{}, colony.ColonyState, error) {
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
	updateSessionSummary("continue-finalize", nextCommand, summary)
	result := map[string]interface{}{
		"advanced":            false,
		"blocked":             true,
		"partial_success":     assessment.PartialSuccess,
		"current_phase":       blockedState.CurrentPhase,
		"phase_name":          phase.Name,
		"state":               blockedState.State,
		"next":                nextCommand,
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
	return result, blockedState, nil
}

func advanceExternalContinue(root string, state colony.ColonyState, phase colony.Phase, manifest codexContinueManifest, verification codexContinueVerificationReport, assessment codexContinueAssessment, gates codexContinueGateReport, review codexContinueReviewReport, reviewReportRel string, watcherFlow *codexContinueWorkerFlowStep, workerFlow []codexContinueWorkerFlowStep, now time.Time, verificationReportRel, gateReportRel string) (map[string]interface{}, colony.ColonyState, *colony.Phase, *signalHousekeepingResult, bool, error) {
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
