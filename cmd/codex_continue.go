package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/agent"
	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
)

type codexVerificationStep struct {
	Name     string `json:"name"`
	Command  string `json:"command,omitempty"`
	Passed   bool   `json:"passed"`
	Skipped  bool   `json:"skipped,omitempty"`
	ExitCode int    `json:"exit_code,omitempty"`
	Summary  string `json:"summary"`
	Output   string `json:"output,omitempty"`
}

type codexClaimVerification struct {
	Present    bool     `json:"present"`
	Passed     bool     `json:"passed"`
	Skipped    bool     `json:"skipped,omitempty"`
	Summary    string   `json:"summary"`
	Checked    int      `json:"checked"`
	Mismatches []string `json:"mismatches,omitempty"`
}

type codexContinueVerificationReport struct {
	Phase          int                      `json:"phase"`
	GeneratedAt    string                   `json:"generated_at"`
	Steps          []codexVerificationStep  `json:"steps"`
	Claims         codexClaimVerification   `json:"claims"`
	Watcher        codexWatcherVerification `json:"watcher"`
	ChecksPassed   bool                     `json:"checks_passed"`
	Passed         bool                     `json:"passed"`
	BlockingIssues []string                 `json:"blocking_issues,omitempty"`
}

type codexWatcherVerification struct {
	Present bool   `json:"present"`
	Passed  bool   `json:"passed"`
	Status  string `json:"status,omitempty"`
	Worker  string `json:"worker,omitempty"`
	Summary string `json:"summary,omitempty"`
}

type codexContinueGateReport struct {
	Phase          int         `json:"phase"`
	GeneratedAt    string      `json:"generated_at"`
	Checks         []gateCheck `json:"checks"`
	Passed         bool        `json:"passed"`
	BlockingIssues []string    `json:"blocking_issues,omitempty"`
}

type codexContinueReport struct {
	Phase              int                           `json:"phase"`
	GeneratedAt        string                        `json:"generated_at"`
	Manifest           string                        `json:"manifest,omitempty"`
	VerificationReport string                        `json:"verification_report"`
	GateReport         string                        `json:"gate_report"`
	ReviewReport       string                        `json:"review_report,omitempty"`
	Summary            string                        `json:"summary,omitempty"`
	ClosedWorkers      []string                      `json:"closed_workers,omitempty"`
	WorkerFlow         []codexContinueWorkerFlowStep `json:"worker_flow,omitempty"`
	PartialSuccess     bool                          `json:"partial_success,omitempty"`
	OperationalIssues  []string                      `json:"operational_issues,omitempty"`
	Tasks              []codexContinueTaskAssessment `json:"tasks,omitempty"`
	Recovery           codexContinueRecoveryPlan     `json:"recovery,omitempty"`
	Advanced           bool                          `json:"advanced"`
	Completed          bool                          `json:"completed"`
	Next               string                        `json:"next"`
}

type codexContinueManifest struct {
	Present bool
	Path    string
	Data    codexBuildManifest
}

type codexVerificationCommands struct {
	Build string
	Type  string
	Lint  string
	Test  string
}

type codexContinueOptions struct {
	ReconcileTaskIDs []string
}

// detectAbandonedBuild checks whether all manifest dispatches are stuck at
// "spawned" and the build started long enough ago to be considered abandoned.
// Returns (true, elapsed, summary) when the build is abandoned.
func detectAbandonedBuild(manifest codexContinueManifest, state colony.ColonyState) (abandoned bool, staleDuration time.Duration, summary string) {
	if !manifest.Present || len(manifest.Data.Dispatches) == 0 {
		return false, 0, ""
	}
	if state.BuildStartedAt == nil {
		return false, 0, ""
	}
	for _, dispatch := range manifest.Data.Dispatches {
		if strings.TrimSpace(dispatch.Status) != "spawned" {
			return false, 0, ""
		}
	}
	elapsed := time.Since(*state.BuildStartedAt)
	if elapsed < 10*time.Minute {
		return false, 0, ""
	}
	return true, elapsed, fmt.Sprintf("Build was abandoned %.0f minutes ago: all %d dispatches stuck at 'spawned'", elapsed.Minutes(), len(manifest.Data.Dispatches))
}

// abandonedBuildTaskIDs extracts unique task IDs from manifest dispatches.
func abandonedBuildTaskIDs(manifest codexContinueManifest) []string {
	seen := make(map[string]struct{})
	var ids []string
	for _, d := range manifest.Data.Dispatches {
		id := strings.TrimSpace(d.TaskID)
		if id != "" {
			if _, ok := seen[id]; !ok {
				seen[id] = struct{}{}
				ids = append(ids, id)
			}
		}
	}
	return ids
}

// cleanupStaleContinueReports removes stale report files from a phase's build
// directory before verification runs. This prevents confusing users with
// leftover artifacts from previous continue attempts.
func cleanupStaleContinueReports(phaseID int) {
	dir := filepath.ToSlash(filepath.Join("build", fmt.Sprintf("phase-%d", phaseID)))
	for _, name := range []string{"verification.json", "gates.json", "continue.json", "review.json"} {
		_ = os.Remove(filepath.Join(store.BasePath(), dir, name))
	}
}

type codexContinueTaskAssessment struct {
	TaskID           string   `json:"task_id"`
	Goal             string   `json:"goal"`
	Outcome          string   `json:"outcome"`
	Summary          string   `json:"summary"`
	Verified         bool     `json:"verified,omitempty"`
	Reconciled       bool     `json:"reconciled,omitempty"`
	DispatchStatuses []string `json:"dispatch_statuses,omitempty"`
	RecoveryAction   string   `json:"recovery_action,omitempty"`
}

type codexContinueRecoveryPlan struct {
	ReverifyCommand   string   `json:"reverify_command,omitempty"`
	ReconcileTasks    []string `json:"reconcile_tasks,omitempty"`
	ReconcileCommand  string   `json:"reconcile_command,omitempty"`
	RedispatchTasks   []string `json:"redispatch_tasks,omitempty"`
	RedispatchCommand string   `json:"redispatch_command,omitempty"`
}

type codexContinueAssessment struct {
	Phase              int                           `json:"phase"`
	GeneratedAt        string                        `json:"generated_at"`
	Tasks              []codexContinueTaskAssessment `json:"tasks"`
	VerificationPassed bool                          `json:"verification_passed"`
	PositiveEvidence   bool                          `json:"positive_evidence"`
	PartialSuccess     bool                          `json:"partial_success,omitempty"`
	OperationalIssues  []string                      `json:"operational_issues,omitempty"`
	ReconciledTasks    []string                      `json:"reconciled_tasks,omitempty"`
	RedispatchTasks    []string                      `json:"redispatch_tasks,omitempty"`
	BlockingIssues     []string                      `json:"blocking_issues,omitempty"`
	Passed             bool                          `json:"passed"`
	Summary            string                        `json:"summary"`
	Recovery           codexContinueRecoveryPlan     `json:"recovery,omitempty"`
}

type codexContinueClosedWorker struct {
	Stage   string `json:"stage,omitempty"`
	Caste   string `json:"caste,omitempty"`
	Name    string `json:"name"`
	Task    string `json:"task,omitempty"`
	Status  string `json:"status"`
	Summary string `json:"summary,omitempty"`
}

type codexContinueWorkerFlowStep struct {
	Stage   string `json:"stage,omitempty"`
	Caste   string `json:"caste,omitempty"`
	Name    string `json:"name"`
	Task    string `json:"task,omitempty"`
	Status  string `json:"status"`
	Summary string `json:"summary,omitempty"`
}

type codexContinueReviewReport struct {
	Phase          int                           `json:"phase"`
	GeneratedAt    string                        `json:"generated_at"`
	Workers        []codexContinueWorkerFlowStep `json:"workers"`
	Passed         bool                          `json:"passed"`
	BlockingIssues []string                      `json:"blocking_issues,omitempty"`
}

var continueContextUpdater = updateCodexContinueContext

var continueSignalHousekeeper = func(now time.Time, state colony.ColonyState) (signalHousekeepingResult, error) {
	return runSignalHousekeepingWithState(now, false, &state)
}

func runCodexContinue(root string, options codexContinueOptions) (map[string]interface{}, colony.ColonyState, colony.Phase, *colony.Phase, *signalHousekeepingResult, bool, error) {
	if store == nil {
		return nil, colony.ColonyState{}, colony.Phase{}, nil, nil, false, fmt.Errorf("no store initialized")
	}

	state, err := loadActiveColonyState()
	if err != nil {
		return nil, state, colony.Phase{}, nil, nil, false, fmt.Errorf("%s", colonyStateLoadMessage(err))
	}

	// Background cleanup of orphaned worktrees — non-blocking
	gcCleaned, gcOrphaned, _ := gcOrphanedWorktrees()
	if gcCleaned > 0 || gcOrphaned > 0 {
		emitVisualProgress(fmt.Sprintf("Worktree cleanup: %d cleaned, %d orphaned", gcCleaned, gcOrphaned))
	}

	if len(state.Plan.Phases) == 0 {
		return nil, state, colony.Phase{}, nil, nil, false, fmt.Errorf("No project plan. Run `aether plan` first.")
	}
	if state.State != colony.StateEXECUTING && state.State != colony.StateBUILT {
		return nil, state, colony.Phase{}, nil, nil, false, fmt.Errorf("No active phase to continue. Run `aether build <phase>` first.")
	}
	if state.CurrentPhase < 1 || state.CurrentPhase > len(state.Plan.Phases) {
		return nil, state, colony.Phase{}, nil, nil, false, fmt.Errorf("No active phase to continue. Run `aether build <phase>` first.")
	}

	currentIdx := state.CurrentPhase - 1
	phase := state.Plan.Phases[currentIdx]
	if phase.Status != colony.PhaseInProgress {
		return nil, state, colony.Phase{}, nil, nil, false, fmt.Errorf("phase %d is not in progress; run `aether build %d` first", phase.ID, phase.ID)
	}
	if err := validateContinueReconcileTasks(phase, options.ReconcileTaskIDs); err != nil {
		return nil, state, colony.Phase{}, nil, nil, false, err
	}
	manifest := loadCodexContinueManifest(phase.ID)
	if state.BuildStartedAt == nil && !manifest.Present {
		return nil, state, colony.Phase{}, nil, nil, false, fmt.Errorf("No active build packet found. Run `aether build <phase>` first.")
	}

	// Abandoned build detection: if all dispatches are still "spawned" and the
	// build started more than 10 minutes ago, the build was abandoned mid-execution.
	// Return a blocked result with recovery commands instead of running verification
	// against incomplete dispatches.
	if abandoned, duration, abandonedSummary := detectAbandonedBuild(manifest, state); abandoned {
		now := time.Now().UTC()
		runHandle, _ := beginRuntimeSpawnRun("continue", now)
		taskIDs := abandonedBuildTaskIDs(manifest)
		recovery := codexContinueRecoveryPlan{
			RedispatchTasks:   taskIDs,
			RedispatchCommand: buildTargetedRedispatchCommand(phase.ID, taskIDs),
			ReconcileTasks:    taskIDs,
			ReconcileCommand:  buildContinueReconcileCommand(taskIDs),
		}
		_ = store.SaveJSON(
			filepath.ToSlash(filepath.Join("build", fmt.Sprintf("phase-%d", phase.ID), "continue.json")),
			codexContinueReport{
				Phase:         phase.ID,
				GeneratedAt:   now.Format(time.RFC3339),
				Summary:       abandonedSummary,
				Recovery:      recovery,
				Advanced:      false,
				Completed:     false,
			},
		)
		finishRuntimeSpawnRun(runHandle, "blocked-abandoned", now)
		return map[string]interface{}{
			"advanced":        false,
			"blocked":         true,
			"abandoned":       true,
			"stale_duration":  duration.String(),
			"current_phase":   state.CurrentPhase,
			"phase_name":      phase.Name,
			"state":           state.State,
			"recovery":        recovery,
			"blocking_issues": []string{abandonedSummary},
		}, state, phase, nil, nil, false, nil
	}

	// Clear stale reports from previous continue runs so users don't see
	// confusing leftover artifacts when reviewing the phase directory.
	cleanupStaleContinueReports(phase.ID)

	now := time.Now().UTC()
	runHandle, err := beginRuntimeSpawnRun("continue", now)
	if err != nil {
		return nil, state, colony.Phase{}, nil, nil, false, fmt.Errorf("failed to initialize continue run: %w", err)
	}
	runStatus := "failed"
	defer func() {
		finishRuntimeSpawnRun(runHandle, runStatus, time.Now().UTC())
	}()

	verification, watcherFlow := runCodexContinueVerification(root, phase, manifest)
	assessment := assessCodexContinue(phase, manifest, verification, options, now)
	verification = attachContinueClaimVerification(verification, assessment)
	gates := runCodexContinueGates(phase, manifest, verification, assessment, now)

	if tracer != nil && state.RunID != nil {
		_ = tracer.LogArtifact(*state.RunID, "continue.verification", map[string]interface{}{
			"phase":          phase.ID,
			"checks_passed":  verification.ChecksPassed,
			"steps_count":    len(verification.Steps),
			"claims_present": verification.Claims.Present,
			"claims_passed":  verification.Claims.Passed,
		})
		_ = tracer.LogArtifact(*state.RunID, "continue.assessment", map[string]interface{}{
			"phase":              phase.ID,
			"passed":             assessment.Passed,
			"partial_success":    assessment.PartialSuccess,
			"tasks_count":        len(assessment.Tasks),
			"operational_issues": len(assessment.OperationalIssues),
		})
		_ = tracer.LogArtifact(*state.RunID, "continue.gates", map[string]interface{}{
			"phase":    phase.ID,
			"passed":   gates.Passed,
			"checks":   len(gates.Checks),
			"blockers": len(gates.BlockingIssues),
		})
	}

	verificationReportRel := filepath.ToSlash(filepath.Join("build", fmt.Sprintf("phase-%d", phase.ID), "verification.json"))
	gateReportRel := filepath.ToSlash(filepath.Join("build", fmt.Sprintf("phase-%d", phase.ID), "gates.json"))
	if err := store.SaveJSON(verificationReportRel, verification); err != nil {
		return nil, state, phase, nil, nil, false, fmt.Errorf("failed to write verification report: %w", err)
	}
	if err := store.SaveJSON(gateReportRel, gates); err != nil {
		return nil, state, phase, nil, nil, false, fmt.Errorf("failed to write gate report: %w", err)
	}

	if !gates.Passed {
		blockers := append([]string{}, gates.BlockingIssues...)
		summary := "Continue blocked by verification or gate failures"
		if len(blockers) > 0 {
			summary = blockers[0]
		}
		continueReportRel := filepath.ToSlash(filepath.Join("build", fmt.Sprintf("phase-%d", phase.ID), "continue.json"))
		workerFlow := continueWorkerFlowWithWatcher(nil, watcherFlow)
		nextCommand := continueNextCommandForAssessment(assessment)
		_ = store.SaveJSON(continueReportRel, codexContinueReport{
			Phase:              phase.ID,
			GeneratedAt:        now.Format(time.RFC3339),
			Manifest:           displayOptionalDataPath(manifest.Path),
			VerificationReport: displayDataPath(verificationReportRel),
			GateReport:         displayDataPath(gateReportRel),
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
		blockedState, flowErr := recordBlockedContinueWorkerFlow(state, now, workerFlow)
		if flowErr != nil {
			return nil, state, phase, nil, nil, false, flowErr
		}
		updateSessionSummary("continue", nextCommand, summary)

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
		runStatus = "blocked"
		return result, blockedState, phase, nil, nil, false, nil
	}

	review := runCodexContinueReview(root, phase, manifest, verification, assessment)
	reviewReportRel := filepath.ToSlash(filepath.Join("build", fmt.Sprintf("phase-%d", phase.ID), "review.json"))
	if err := store.SaveJSON(reviewReportRel, review); err != nil {
		return nil, state, phase, nil, nil, false, fmt.Errorf("failed to write review report: %w", err)
	}
	if !review.Passed {
		summary := "Continue blocked because the review wave did not clear"
		if len(review.BlockingIssues) > 0 {
			summary = review.BlockingIssues[0]
		}
		continueReportRel := filepath.ToSlash(filepath.Join("build", fmt.Sprintf("phase-%d", phase.ID), "continue.json"))
		workerFlow := continueWorkerFlowWithWatcher(review.Workers, watcherFlow)
		nextCommand := continueNextCommandForAssessment(assessment)
		_ = store.SaveJSON(continueReportRel, codexContinueReport{
			Phase:              phase.ID,
			GeneratedAt:        now.Format(time.RFC3339),
			Manifest:           displayOptionalDataPath(manifest.Path),
			VerificationReport: displayDataPath(verificationReportRel),
			GateReport:         displayDataPath(gateReportRel),
			ReviewReport:       displayDataPath(reviewReportRel),
			Summary:            summary,
			WorkerFlow:         workerFlow,
			PartialSuccess:     assessment.PartialSuccess,
			OperationalIssues:  append(append([]string{}, assessment.OperationalIssues...), review.BlockingIssues...),
			Tasks:              append([]codexContinueTaskAssessment{}, assessment.Tasks...),
			Recovery:           assessment.Recovery,
			Advanced:           false,
			Completed:          false,
			Next:               nextCommand,
		})
		blockedState, flowErr := recordBlockedContinueWorkerFlow(state, now, workerFlow)
		if flowErr != nil {
			return nil, state, phase, nil, nil, false, flowErr
		}
		updateSessionSummary("continue", nextCommand, summary)
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
			"review":              review,
			"verification_report": displayDataPath(verificationReportRel),
			"gate_report":         displayDataPath(gateReportRel),
			"review_report":       displayDataPath(reviewReportRel),
			"continue_report":     displayDataPath(continueReportRel),
			"worker_flow":         workerFlow,
			"operational_issues":  append(append([]string{}, assessment.OperationalIssues...), review.BlockingIssues...),
			"recovery":            assessment.Recovery,
			"reconciled_tasks":    assessment.ReconciledTasks,
			"blocking_issues":     append([]string{}, review.BlockingIssues...),
		}
		runStatus = "blocked"
		return result, blockedState, phase, nil, nil, false, nil
	}

	closedWorkerDetails := plannedCodexContinueClosedWorkers(manifest, assessment)
	closedWorkers := closedWorkerNames(closedWorkerDetails)

	// --- ATOMIC STATE COMMIT ---
	// Mutate and save colony state in a single atomic read-modify-write cycle.
	// If the mutation fails, no write occurs. This runs BEFORE side effects
	// and report saves so no external observer can see a partially advanced state.
	var (
		nextPhase   *colony.Phase
		nextCommand string
		final       bool
		updated     colony.ColonyState
	)
	if err := store.UpdateJSONAtomically("COLONY_STATE.json", &updated, func() error {
		updated = state
		updated.Events = append(trimmedEvents(updated.Events),
			fmt.Sprintf("%s|verification_passed|continue|Build verification passed for phase %d", now.Format(time.RFC3339), phase.ID),
			fmt.Sprintf("%s|gate_passed|continue|Continue gates passed for phase %d", now.Format(time.RFC3339), phase.ID),
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
				fmt.Sprintf("%s|phase_completed|continue|Completed final phase %d", now.Format(time.RFC3339), updated.CurrentPhase),
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
				fmt.Sprintf("%s|phase_advanced|continue|Completed phase %d, ready for phase %d", now.Format(time.RFC3339), phase.ID, nextIdx+1),
			)
		}
		return nil
	}); err != nil {
		return nil, state, phase, nil, nil, false, fmt.Errorf("failed to atomically advance phase: %w", err)
	}

	// --- SIDE EFFECTS (after state is durable) ---
	// These operations produce derived data. If any fails, state is already
	// committed and valid; the error is informational, not a rollback trigger.
	housekeeping, housekeepingErr := continueSignalHousekeeper(now, updated)
	if housekeepingErr != nil {
		return nil, state, phase, nil, nil, false, housekeepingErr
	}
	if err := continueContextUpdater(phase, manifest, closedWorkerDetails, now); err != nil {
		return nil, state, phase, nil, nil, false, err
	}
	workerFlow := continueWorkerFlowWithWatcher(review.Workers, watcherFlow)
	workerFlow = append(workerFlow, continueHousekeepingFlowStep(housekeeping))
	if err := recordContinueWorkerFlow(workerFlow); err != nil {
		return nil, state, phase, nil, &housekeeping, false, err
	}
	if err := applyCodexContinueWorkerClosures(closedWorkerDetails); err != nil {
		return nil, state, phase, nil, &housekeeping, false, err
	}
	updated.Events = append(updated.Events, continueWorkerFlowEvents(now, workerFlow)...)
	// Persist side-effect events (review, housekeeping) into colony state.
	// This is a best-effort append; the core advancement was already committed.
	_ = store.SaveJSON("COLONY_STATE.json", updated)

	summary := fmt.Sprintf("Phase %d verified and advanced", phase.ID)
	if assessment.PartialSuccess {
		summary = fmt.Sprintf("Phase %d verified and advanced with partial operational success", phase.ID)
	}

	// --- REPORT SAVES (after state is durable) ---
	continueReportRel := filepath.ToSlash(filepath.Join("build", fmt.Sprintf("phase-%d", phase.ID), "continue.json"))
	if err := store.SaveJSON(continueReportRel, codexContinueReport{
		Phase:              phase.ID,
		GeneratedAt:        now.Format(time.RFC3339),
		Manifest:           displayOptionalDataPath(manifest.Path),
		VerificationReport: displayDataPath(verificationReportRel),
		GateReport:         displayDataPath(gateReportRel),
		ReviewReport:       displayDataPath(reviewReportRel),
		Summary:            summary,
		ClosedWorkers:      closedWorkers,
		WorkerFlow:         workerFlow,
		PartialSuccess:     assessment.PartialSuccess,
		OperationalIssues:  append([]string{}, assessment.OperationalIssues...),
		Tasks:              append([]codexContinueTaskAssessment{}, assessment.Tasks...),
		Recovery:           assessment.Recovery,
		Advanced:           true,
		Completed:          final,
		Next:               nextCommand,
	}); err != nil {
		return nil, state, phase, nextPhase, &housekeeping, final, fmt.Errorf("failed to write continue report: %w", err)
	}

	updateSessionSummary("continue", nextCommand, summary)
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
		"worker_flow":         workerFlow,
		"operational_issues":  assessment.OperationalIssues,
		"recovery":            assessment.Recovery,
		"reconciled_tasks":    assessment.ReconciledTasks,
		"signal_housekeeping": housekeeping,
	}
	if nextPhase != nil {
		result["next_phase"] = nextPhase.ID
		result["next_phase_name"] = nextPhase.Name
	}
	runStatus = "completed"
	return result, updated, updated.Plan.Phases[currentIdx], nextPhase, &housekeeping, final, nil
}

func loadCodexContinueManifest(phaseID int) codexContinueManifest {
	rel := filepath.ToSlash(filepath.Join("build", fmt.Sprintf("phase-%d", phaseID), "manifest.json"))
	var manifest codexBuildManifest
	if err := store.LoadJSON(rel, &manifest); err != nil {
		return codexContinueManifest{}
	}
	return codexContinueManifest{
		Present: true,
		Path:    rel,
		Data:    manifest,
	}
}

type codexContinueReviewSpec struct {
	Caste string
	Task  string
}

var codexContinueReviewSpecs = []codexContinueReviewSpec{
	{Caste: "gatekeeper", Task: "Review the phase for security, release, and integrity blockers before advancement. Return blocked if it is unsafe to advance."},
	{Caste: "auditor", Task: "Audit whether the completed work actually satisfies the phase tasks rather than just producing superficial artifacts. Return blocked if the evidence looks partial, generic, or docs-only."},
	{Caste: "probe", Task: "Probe the verification evidence for missing edge cases, weak tests, or unexercised behavior. Return blocked if test evidence is too weak to trust advancement."},
}

func runCodexContinueReview(root string, phase colony.Phase, manifest codexContinueManifest, verification codexContinueVerificationReport, assessment codexContinueAssessment) codexContinueReviewReport {
	report := codexContinueReviewReport{
		Phase:       phase.ID,
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Workers:     []codexContinueWorkerFlowStep{},
		Passed:      false,
	}

	invoker := newCodexWorkerInvoker()
	if _, ok := invoker.(*codex.FakeInvoker); !ok && !invoker.IsAvailable(context.Background()) {
		fmt.Fprintf(os.Stderr, "⚠ Codex CLI unavailable — skipping review wave, proceeding with claims verification\n")
		report.Passed = true
		return report
	}

	dispatches := plannedContinueReviewDispatches(root, phase, manifest, verification, assessment, invoker)
	spawnTree := agent.NewSpawnTree(store, "spawn-tree.txt")
	results, err := dispatchBatchByWaveWithVisuals(
		context.Background(),
		invoker,
		dispatches,
		colony.ModeInRepo,
		"Continue Review",
		true,
		func(wave int) codex.DispatchObserver {
			return runtimeVisualDispatchObserver(spawnTree, "Continue review active", wave)
		},
	)
	if err != nil {
		report.BlockingIssues = []string{err.Error()}
	}

	flow := make([]codexContinueWorkerFlowStep, 0, len(dispatches))
	blockers := append([]string{}, report.BlockingIssues...)
	for i, dispatch := range dispatches {
		step := codexContinueWorkerFlowStep{
			Stage:  "review",
			Caste:  dispatch.Caste,
			Name:   dispatch.WorkerName,
			Task:   continueReviewTaskForCaste(dispatch.Caste),
			Status: "failed",
		}
		if i < len(results) {
			result := results[i]
			step.Name = result.WorkerName
			step.Status = normalizeRuntimeDispatchStatus(result.Status)
			if result.WorkerResult != nil {
				if len(result.WorkerResult.Blockers) > 0 {
					step.Summary = strings.Join(result.WorkerResult.Blockers, "; ")
				} else if summary := strings.TrimSpace(result.WorkerResult.Summary); summary != "" && !strings.HasPrefix(summary, "FakeInvoker completed task") {
					step.Summary = summary
				}
				if len(result.WorkerResult.Blockers) > 0 {
					for _, blocker := range result.WorkerResult.Blockers {
						if strings.TrimSpace(blocker) != "" {
							blockers = append(blockers, fmt.Sprintf("%s reported blocker: %s", result.WorkerName, blocker))
						}
					}
				}
			}
			if step.Summary == "" && result.Error != nil {
				step.Summary = strings.TrimSpace(result.Error.Error())
			}
			if step.Summary == "" {
				step.Summary = continueReviewFlowSummary(step)
			}
			if step.Status != "completed" {
				blockers = append(blockers, fmt.Sprintf("%s review did not complete cleanly: %s", result.WorkerName, step.Status))
			}
		}
		flow = append(flow, step)
	}

	report.Workers = flow
	report.BlockingIssues = uniqueSortedStrings(blockers)
	report.Passed = len(report.BlockingIssues) == 0
	return report
}

func plannedContinueReviewDispatches(root string, phase colony.Phase, manifest codexContinueManifest, verification codexContinueVerificationReport, assessment codexContinueAssessment, invoker codex.WorkerInvoker) []codex.WorkerDispatch {
	capsule := resolveCodexWorkerContext()
	pheromoneSection := resolvePheromoneSection()
	dispatches := make([]codex.WorkerDispatch, 0, len(codexContinueReviewSpecs))
	for idx, spec := range codexContinueReviewSpecs {
		agentName := codexAgentNameForCaste(spec.Caste)
		dispatches = append(dispatches, codex.WorkerDispatch{
			ID:               fmt.Sprintf("continue-review-%d", idx),
			WorkerName:       deterministicAntName(spec.Caste, fmt.Sprintf("phase:%d:continue:%s", phase.ID, spec.Caste)),
			AgentName:        agentName,
			AgentTOMLPath:    dispatchAgentPath(root, invoker, agentName),
			Caste:            spec.Caste,
			TaskID:           fmt.Sprintf("continue-review-%s", spec.Caste),
			TaskBrief:        renderCodexContinueReviewBrief(root, phase, manifest, verification, assessment, spec),
			ContextCapsule:   capsule,
			SkillSection:     resolveSkillSection(spec.Caste, spec.Task),
			PheromoneSection: pheromoneSection,
			Root:             root,
			Timeout:          continueReviewTimeout,
			Wave:             1,
		})
	}
	return dispatches
}

func renderCodexContinueReviewBrief(root string, phase colony.Phase, manifest codexContinueManifest, verification codexContinueVerificationReport, assessment codexContinueAssessment, spec codexContinueReviewSpec) string {
	var b strings.Builder
	b.WriteString("# Continue Review\n\n")
	b.WriteString("- Phase: ")
	b.WriteString(fmt.Sprintf("%d — %s\n", phase.ID, phase.Name))
	b.WriteString("- Repo: ")
	b.WriteString(root)
	b.WriteString("\n- Role: ")
	b.WriteString(spec.Caste)
	b.WriteString("\n\n")
	b.WriteString(spec.Task)
	b.WriteString("\n\n")
	b.WriteString("This is a read-only review. Do not modify repo files. Return status `blocked` if advancement is unsafe.\n\n")
	b.WriteString("Evidence to inspect:\n")
	if manifest.Present {
		b.WriteString("- Build manifest: ")
		b.WriteString(displayDataPath(manifest.Path))
		b.WriteString("\n")
	}
	if claimsPath := strings.TrimSpace(manifest.Data.ClaimsPath); claimsPath != "" {
		b.WriteString("- Build claims: ")
		b.WriteString(claimsPath)
		b.WriteString("\n")
	}
	b.WriteString("- Verification checks passed: ")
	b.WriteString(fmt.Sprintf("%t\n", verification.ChecksPassed))
	if len(verification.BlockingIssues) > 0 {
		b.WriteString("- Verification blockers: ")
		b.WriteString(strings.Join(verification.BlockingIssues, "; "))
		b.WriteString("\n")
	}
	if len(assessment.Tasks) > 0 {
		b.WriteString("\nTask evidence summary:\n")
		for _, task := range assessment.Tasks {
			b.WriteString("- ")
			b.WriteString(task.TaskID)
			b.WriteString(": ")
			b.WriteString(task.Outcome)
			b.WriteString(" — ")
			b.WriteString(task.Summary)
			b.WriteString("\n")
		}
	}
	if len(assessment.OperationalIssues) > 0 {
		b.WriteString("\nOperational issues already observed:\n")
		for _, issue := range assessment.OperationalIssues {
			b.WriteString("- ")
			b.WriteString(issue)
			b.WriteString("\n")
		}
	}
	return b.String()
}

func runCodexContinueVerification(root string, phase colony.Phase, manifest codexContinueManifest) (codexContinueVerificationReport, *codexContinueWorkerFlowStep) {
	now := time.Now().UTC()
	commands := resolveCodexVerificationCommands(root)
	steps := []codexVerificationStep{
		runVerificationStep(root, "build", commands.Build),
		runVerificationStep(root, "types", commands.Type),
		runVerificationStep(root, "lint", commands.Lint),
		runVerificationStep(root, "tests", commands.Test),
	}
	claims := verifyCodexBuildClaims(root, manifest)
	buildWatcher := evaluateContinueWatcherVerification(manifest)
	continueWatcher, watcherFlow := runCodexContinueWatcherVerification(root, phase, manifest, steps, claims, buildWatcher)
	watcher := continueWatcher
	if !watcher.Present {
		watcher = buildWatcher
	}

	checksPassed := true
	blockers := []string{}
	for _, step := range steps {
		if !step.Passed && !step.Skipped {
			checksPassed = false
			blockers = append(blockers, fmt.Sprintf("%s failed: %s", step.Name, step.Summary))
		}
	}
	if watcher.Present && !watcher.Passed {
		checksPassed = false
		summary := strings.TrimSpace(watcher.Summary)
		if summary == "" {
			summary = "watcher verification did not complete cleanly"
		}
		blockers = append(blockers, summary)
	}

	return codexContinueVerificationReport{
		Phase:          phase.ID,
		GeneratedAt:    now.Format(time.RFC3339),
		Steps:          steps,
		Claims:         claims,
		Watcher:        watcher,
		ChecksPassed:   checksPassed,
		Passed:         checksPassed,
		BlockingIssues: blockers,
	}, watcherFlow
}

func runCodexContinueWatcherVerification(root string, phase colony.Phase, manifest codexContinueManifest, steps []codexVerificationStep, claims codexClaimVerification, buildWatcher codexWatcherVerification) (codexWatcherVerification, *codexContinueWorkerFlowStep) {
	invoker := newCodexWorkerInvoker()
	dispatch := plannedContinueWatcherDispatch(root, phase, manifest, steps, claims, buildWatcher, invoker)
	if !invoker.IsAvailable(context.Background()) {
		summary := fmt.Sprintf("continue watcher verification could not start because %s", dispatchAvailabilityMessage(invoker))
		return codexWatcherVerification{
				Present: true,
				Passed:  false,
				Status:  "failed",
				Worker:  dispatch.WorkerName,
				Summary: summary,
			}, &codexContinueWorkerFlowStep{
				Stage:   "verification",
				Caste:   "watcher",
				Name:    dispatch.WorkerName,
				Task:    "Independent verification before advancement",
				Status:  "failed",
				Summary: continueWatcherFlowSummary(dispatch.WorkerName, "failed", summary),
			}
	}

	spawnTree := agent.NewSpawnTree(store, "spawn-tree.txt")
	results, err := dispatchBatchByWaveWithVisuals(
		context.Background(),
		invoker,
		[]codex.WorkerDispatch{dispatch},
		colony.ModeInRepo,
		"Continue Verification",
		true,
		func(wave int) codex.DispatchObserver {
			return runtimeVisualDispatchObserver(spawnTree, "Continue verification active", wave)
		},
	)
	if err != nil {
		summary := strings.TrimSpace(err.Error())
		if summary == "" {
			summary = "continue watcher verification dispatch failed"
		}
		return codexWatcherVerification{
				Present: true,
				Passed:  false,
				Status:  "failed",
				Worker:  dispatch.WorkerName,
				Summary: summary,
			}, &codexContinueWorkerFlowStep{
				Stage:   "verification",
				Caste:   "watcher",
				Name:    dispatch.WorkerName,
				Task:    "Independent verification before advancement",
				Status:  "failed",
				Summary: continueWatcherFlowSummary(dispatch.WorkerName, "failed", summary),
			}
	}

	if len(results) == 0 {
		summary := "continue watcher verification produced no result"
		return codexWatcherVerification{
				Present: true,
				Passed:  false,
				Status:  "failed",
				Worker:  dispatch.WorkerName,
				Summary: summary,
			}, &codexContinueWorkerFlowStep{
				Stage:   "verification",
				Caste:   "watcher",
				Name:    dispatch.WorkerName,
				Task:    "Independent verification before advancement",
				Status:  "failed",
				Summary: continueWatcherFlowSummary(dispatch.WorkerName, "failed", summary),
			}
	}

	result := results[0]
	status := normalizeRuntimeDispatchStatus(result.Status)
	if status == "" {
		status = "failed"
	}
	workerName := strings.TrimSpace(result.WorkerName)
	if workerName == "" {
		workerName = dispatch.WorkerName
	}
	summary := strings.TrimSpace(continueWatcherResultSummary(result))
	if summary == "" {
		summary = continueWatcherDefaultSummary(status)
	}

	return codexWatcherVerification{
			Present: true,
			Passed:  status == "completed" || status == "manually-reconciled",
			Status:  status,
			Worker:  workerName,
			Summary: summary,
		}, &codexContinueWorkerFlowStep{
			Stage:   "verification",
			Caste:   "watcher",
			Name:    workerName,
			Task:    "Independent verification before advancement",
			Status:  status,
			Summary: continueWatcherFlowSummary(workerName, status, summary),
		}
}

func plannedContinueWatcherDispatch(root string, phase colony.Phase, manifest codexContinueManifest, steps []codexVerificationStep, claims codexClaimVerification, buildWatcher codexWatcherVerification, invoker codex.WorkerInvoker) codex.WorkerDispatch {
	agentName := codexAgentNameForCaste("watcher")
	return codex.WorkerDispatch{
		ID:               fmt.Sprintf("continue-verification-%d", phase.ID),
		WorkerName:       deterministicAntName("watcher", fmt.Sprintf("phase:%d:continue:watcher", phase.ID)),
		AgentName:        agentName,
		AgentTOMLPath:    dispatchAgentPath(root, invoker, agentName),
		Caste:            "watcher",
		TaskID:           fmt.Sprintf("continue-verification-%d", phase.ID),
		TaskBrief:        renderCodexContinueWatcherBrief(root, phase, manifest, steps, claims, buildWatcher),
		ContextCapsule:   resolveCodexWorkerContext(),
		SkillSection:     resolveSkillSection("watcher", "Independent verification before advancement"),
		PheromoneSection: resolvePheromoneSection(),
		Root:             root,
		Timeout:          continueReviewTimeout,
		Wave:             1,
	}
}

func renderCodexContinueWatcherBrief(root string, phase colony.Phase, manifest codexContinueManifest, steps []codexVerificationStep, claims codexClaimVerification, buildWatcher codexWatcherVerification) string {
	var b strings.Builder
	b.WriteString("# Continue Verification\n\n")
	b.WriteString("- Phase: ")
	b.WriteString(fmt.Sprintf("%d — %s\n", phase.ID, phase.Name))
	b.WriteString("- Repo: ")
	b.WriteString(root)
	b.WriteString("\n\n")
	b.WriteString("Confirm whether this phase is safe to advance right now.\n")
	b.WriteString("This is a read-only verification pass. Do not modify repo files. Return status `completed` only if the current workspace and recorded artifacts justify advancement. Return status `blocked` if anything is missing, stale, or misleading.\n\n")
	b.WriteString("Evidence to inspect:\n")
	if manifest.Present {
		b.WriteString("- Build manifest: ")
		b.WriteString(displayDataPath(manifest.Path))
		b.WriteString("\n")
	}
	if claimsPath := strings.TrimSpace(manifest.Data.ClaimsPath); claimsPath != "" {
		b.WriteString("- Build claims: ")
		b.WriteString(claimsPath)
		b.WriteString("\n")
	}
	if claimsSummary := strings.TrimSpace(claims.Summary); claimsSummary != "" {
		b.WriteString("- Claims summary: ")
		b.WriteString(claimsSummary)
		b.WriteString("\n")
	}
	if buildWatcher.Present {
		b.WriteString("- Build-time watcher: ")
		b.WriteString(strings.TrimSpace(buildWatcher.Status))
		if summary := strings.TrimSpace(buildWatcher.Summary); summary != "" {
			b.WriteString(" — ")
			b.WriteString(summary)
		}
		b.WriteString("\n")
	}
	if len(steps) > 0 {
		b.WriteString("\nVerification commands:\n")
		for _, step := range steps {
			b.WriteString("- ")
			b.WriteString(step.Name)
			b.WriteString(": ")
			if step.Skipped {
				b.WriteString("skipped")
			} else if step.Passed {
				b.WriteString("passed")
			} else {
				b.WriteString("failed")
			}
			if summary := strings.TrimSpace(step.Summary); summary != "" {
				b.WriteString(" — ")
				b.WriteString(summary)
			}
			b.WriteString("\n")
		}
	}
	if len(phase.Tasks) > 0 {
		b.WriteString("\nPhase tasks:\n")
		for idx, task := range phase.Tasks {
			b.WriteString("- ")
			b.WriteString(buildTaskID(task, idx))
			b.WriteString(": ")
			b.WriteString(strings.TrimSpace(task.Goal))
			b.WriteString("\n")
		}
	}
	return b.String()
}

func evaluateContinueWatcherVerification(manifest codexContinueManifest) codexWatcherVerification {
	if !manifest.Present {
		return codexWatcherVerification{}
	}
	for _, dispatch := range manifest.Data.Dispatches {
		stage := strings.ToLower(strings.TrimSpace(dispatch.Stage))
		caste := strings.ToLower(strings.TrimSpace(dispatch.Caste))
		if stage != "verification" && caste != "watcher" {
			continue
		}
		status := continueWorkerFlowStatus(dispatch.Status)
		summary := strings.TrimSpace(dispatch.Summary)
		if summary == "" {
			switch status {
			case "completed", "manually-reconciled":
				summary = "watcher verification completed before advancement"
			default:
				summary = fmt.Sprintf("watcher verification did not complete cleanly: %s", status)
			}
		}
		return codexWatcherVerification{
			Present: true,
			Passed:  status == "completed" || status == "manually-reconciled",
			Status:  status,
			Worker:  strings.TrimSpace(dispatch.Name),
			Summary: summary,
		}
	}
	return codexWatcherVerification{}
}

func validateContinueReconcileTasks(phase colony.Phase, reconcileTaskIDs []string) error {
	if len(reconcileTaskIDs) == 0 {
		return nil
	}
	known := make(map[string]struct{}, len(phase.Tasks))
	for idx, task := range phase.Tasks {
		known[buildTaskID(task, idx)] = struct{}{}
	}
	unknown := make([]string, 0, len(reconcileTaskIDs))
	for _, taskID := range reconcileTaskIDs {
		if _, ok := known[taskID]; !ok {
			unknown = append(unknown, taskID)
		}
	}
	if len(unknown) > 0 {
		return fmt.Errorf("unknown task id(s) for phase %d: %s", phase.ID, strings.Join(unknown, ", "))
	}
	return nil
}

func assessCodexContinue(phase colony.Phase, manifest codexContinueManifest, verification codexContinueVerificationReport, options codexContinueOptions, now time.Time) codexContinueAssessment {
	reconciled := make(map[string]struct{}, len(options.ReconcileTaskIDs))
	for _, taskID := range options.ReconcileTaskIDs {
		reconciled[taskID] = struct{}{}
	}

	dispatchStatuses := make(map[string][]string, len(phase.Tasks))
	operationalIssues := []string{}
	for _, dispatch := range manifest.Data.Dispatches {
		status := strings.TrimSpace(dispatch.Status)
		if dispatch.TaskID != "" {
			dispatchStatuses[dispatch.TaskID] = append(dispatchStatuses[dispatch.TaskID], status)
		}
		if status != "" && status != "completed" {
			operationalIssues = append(operationalIssues, fmt.Sprintf("%s (%s)", dispatch.Name, status))
		}
	}
	operationalIssues = uniqueSortedStrings(operationalIssues)

	requiresBuilderClaims := manifestRequiresBuilderClaims(manifest)
	dispatchEvidenceTrusted := !manifestUsesSyntheticDispatch(manifest)
	claimsSatisfied := verification.Claims.Passed || verification.Claims.Skipped
	tasks := make([]codexContinueTaskAssessment, 0, len(phase.Tasks))
	redispatchTasks := make([]string, 0, len(phase.Tasks))

	for idx, task := range phase.Tasks {
		taskID := buildTaskID(task, idx)
		statuses := uniqueSortedStrings(dispatchStatuses[taskID])
		_, reconciledTask := reconciled[taskID]
		taskArtifactEvidenceTrusted := !requiresBuilderClaims || claimsSatisfied
		outcome, summary, recovery := classifyContinueTaskAssessment(taskID, statuses, verification.ChecksPassed, reconciledTask, dispatchEvidenceTrusted, taskArtifactEvidenceTrusted)
		taskAssessment := codexContinueTaskAssessment{
			TaskID:           taskID,
			Goal:             strings.TrimSpace(task.Goal),
			Outcome:          outcome,
			Summary:          summary,
			Verified:         verification.ChecksPassed,
			Reconciled:       reconciledTask,
			DispatchStatuses: statuses,
			RecoveryAction:   recovery,
		}
		tasks = append(tasks, taskAssessment)
		if recovery == "redispatch" {
			redispatchTasks = append(redispatchTasks, taskID)
		}
	}
	positiveEvidence := continueTasksSupportAdvancement(tasks, claimsSatisfied)

	blockingIssues := []string{}
	if len(options.ReconcileTaskIDs) > 0 {
		blockingIssues = append(blockingIssues, fmt.Sprintf("Warning: %d task(s) were manually reconciled. Verification was re-run, but reconciled tasks do not bypass claim checks.", len(options.ReconcileTaskIDs)))
	}
	if !verification.ChecksPassed {
		blockingIssues = append(blockingIssues, verification.BlockingIssues...)
	}
	if verification.ChecksPassed && !positiveEvidence {
		if requiresBuilderClaims && !claimsSatisfied {
			if len(reconciled) == 0 {
				blockingIssues = append(blockingIssues, verification.Claims.Summary)
				blockingIssues = append(blockingIssues, verification.Claims.Mismatches...)
			} else {
				blockingIssues = append(blockingIssues, "verification passed, but unreconciled tasks still failed builder-claim verification")
				blockingIssues = append(blockingIssues, verification.Claims.Summary)
				blockingIssues = append(blockingIssues, verification.Claims.Mismatches...)
			}
		} else {
			blockingIssues = append(blockingIssues, "verification passed but no implementation evidence was recorded; reconcile completed tasks or redispatch missing work")
		}
	}

	recovery := codexContinueRecoveryPlan{
		ReverifyCommand: "aether continue",
	}
	if len(options.ReconcileTaskIDs) == 0 {
		recovery.ReconcileTasks = tasksNeedingRecovery(tasks)
		if len(recovery.ReconcileTasks) > 0 {
			recovery.ReconcileCommand = buildContinueReconcileCommand(recovery.ReconcileTasks)
		}
	} else {
		recovery.ReconcileTasks = append([]string{}, options.ReconcileTaskIDs...)
		recovery.ReconcileCommand = buildContinueReconcileCommand(recovery.ReconcileTasks)
	}
	if len(redispatchTasks) > 0 {
		recovery.RedispatchTasks = uniqueSortedStrings(redispatchTasks)
		recovery.RedispatchCommand = buildTargetedRedispatchCommand(phase.ID, recovery.RedispatchTasks)
	}

	passed := verification.ChecksPassed && positiveEvidence
	summary := "Verification and task evidence support advancement"
	if passed && len(operationalIssues) > 0 {
		summary = "Verification passed with partial operational success"
	} else if !verification.ChecksPassed {
		summary = "Continue blocked by verification failures"
	} else if requiresBuilderClaims && !claimsSatisfied && len(reconciled) == 0 {
		summary = "Continue blocked by builder claim verification"
	} else if !positiveEvidence {
		summary = "Continue blocked because task evidence is missing"
	}

	return codexContinueAssessment{
		Phase:              phase.ID,
		GeneratedAt:        now.Format(time.RFC3339),
		Tasks:              tasks,
		VerificationPassed: verification.ChecksPassed,
		PositiveEvidence:   positiveEvidence,
		PartialSuccess:     passed && len(operationalIssues) > 0,
		OperationalIssues:  operationalIssues,
		ReconciledTasks:    append([]string{}, options.ReconcileTaskIDs...),
		RedispatchTasks:    recovery.RedispatchTasks,
		BlockingIssues:     uniqueSortedStrings(blockingIssues),
		Passed:             passed,
		Summary:            summary,
		Recovery:           recovery,
	}
}

func continueTasksSupportAdvancement(tasks []codexContinueTaskAssessment, claimsSatisfied bool) bool {
	if len(tasks) == 0 {
		return false
	}
	for _, task := range tasks {
		switch task.Outcome {
		case "missing", "needs_redispatch", "implemented_unverified", "simulated", "manually_reconciled":
			return false
		}
	}
	return true
}

func classifyContinueTaskAssessment(taskID string, statuses []string, verificationPassed, reconciled, dispatchEvidenceTrusted, artifactEvidenceTrusted bool) (string, string, string) {
	if reconciled {
		if verificationPassed {
			return "manually_reconciled", "Task was manually reconciled and the phase verification passed.", "reverify"
		}
		return "manually_reconciled", "Task was manually reconciled, but phase verification still failed.", "reverify"
	}

	if verificationPassed {
		if containsString(statuses, "completed") {
			if !dispatchEvidenceTrusted {
				return "simulated", "Workers only reported completion in simulated mode; rerun this phase without --synthetic or reconcile real manual work.", "redispatch"
			}
			if !artifactEvidenceTrusted {
				return "implemented_unverified", "A worker reported completion for this task, but artifact verification is missing or failed.", "redispatch"
			}
			return "verified", "Task has completed worker evidence and the phase verification passed.", ""
		}
		if len(statuses) == 0 {
			return "missing", "No dispatch or reconciliation evidence was recorded for this task.", "redispatch"
		}
		if !dispatchEvidenceTrusted {
			return "simulated", fmt.Sprintf("Worker evidence is simulated and cannot satisfy continue advancement: %s.", strings.Join(statuses, ", ")), "redispatch"
		}
		if !artifactEvidenceTrusted {
			return "implemented_unverified", fmt.Sprintf("Workers reported activity for this task, but artifact verification is missing or failed: %s.", strings.Join(statuses, ", ")), "redispatch"
		}
		return "needs_redispatch", fmt.Sprintf("Phase verification passed but worker evidence is incomplete or failed: %s.", strings.Join(statuses, ", ")), "redispatch"
	}

	if len(statuses) == 0 {
		return "missing", "No dispatch evidence was recorded for this task.", "redispatch"
	}
	if !dispatchEvidenceTrusted {
		return "simulated", fmt.Sprintf("Worker evidence is simulated and cannot satisfy continue advancement: %s.", strings.Join(statuses, ", ")), "redispatch"
	}
	if containsString(statuses, "completed") {
		return "implemented_unverified", "A worker reported completion for this task, but phase verification failed.", "reverify"
	}
	return "needs_redispatch", fmt.Sprintf("Worker evidence is incomplete or failed for this task: %s.", strings.Join(statuses, ", ")), "redispatch"
}

func tasksNeedingRecovery(tasks []codexContinueTaskAssessment) []string {
	taskIDs := make([]string, 0, len(tasks))
	for _, task := range tasks {
		switch task.Outcome {
		case "missing", "needs_redispatch", "implemented_unverified", "simulated":
			taskIDs = append(taskIDs, task.TaskID)
		}
	}
	return uniqueSortedStrings(taskIDs)
}

func buildContinueReconcileCommand(taskIDs []string) string {
	if len(taskIDs) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("aether continue")
	for _, taskID := range uniqueSortedStrings(taskIDs) {
		b.WriteString(" --reconcile-task ")
		b.WriteString(taskID)
	}
	return b.String()
}

func buildTargetedRedispatchCommand(phaseID int, taskIDs []string) string {
	if len(taskIDs) == 0 {
		return ""
	}
	var b strings.Builder
	fmt.Fprintf(&b, "aether build %d", phaseID)
	for _, taskID := range uniqueSortedStrings(taskIDs) {
		b.WriteString(" --task ")
		b.WriteString(taskID)
	}
	return b.String()
}

func continueNextCommandForAssessment(assessment codexContinueAssessment) string {
	if strings.TrimSpace(assessment.Recovery.RedispatchCommand) != "" {
		return assessment.Recovery.RedispatchCommand
	}
	if continueAssessmentPrefersReverify(assessment) && strings.TrimSpace(assessment.Recovery.ReverifyCommand) != "" {
		return assessment.Recovery.ReverifyCommand
	}
	if strings.TrimSpace(assessment.Recovery.ReconcileCommand) != "" {
		return assessment.Recovery.ReconcileCommand
	}
	if strings.TrimSpace(assessment.Recovery.ReverifyCommand) != "" {
		return assessment.Recovery.ReverifyCommand
	}
	return "aether continue"
}

func continueAssessmentPrefersReverify(assessment codexContinueAssessment) bool {
	sawReverify := false
	for _, task := range assessment.Tasks {
		action := strings.TrimSpace(task.RecoveryAction)
		if action == "" {
			continue
		}
		if action != "reverify" {
			return false
		}
		sawReverify = true
	}
	return sawReverify
}

func resolveCodexVerificationCommands(root string) codexVerificationCommands {
	commands := codexVerificationCommands{}
	mergeCodexVerificationCommands(&commands, loadVerificationCommandsFromMarkdown(filepath.Join(root, "AGENTS.md"), "## Verification Commands"))
	mergeCodexVerificationCommands(&commands, loadVerificationCommandsFromMarkdown(filepath.Join(root, ".codex", "CODEX.md"), "### Verification Commands"))
	mergeCodexVerificationCommands(&commands, loadVerificationCommandsFromMarkdown(filepath.Join(root, "CLAUDE.md"), "## Verification Commands"))
	mergeCodexVerificationCommands(&commands, loadVerificationCommandsFromMarkdown(filepath.Join(root, ".opencode", "OPENCODE.md"), "## Verification Commands"))
	mergeCodexVerificationCommands(&commands, loadVerificationCommandsFromMarkdown(filepath.Join(root, ".aether", "data", "codebase.md"), "## Commands"))
	switch {
	case fileExists(filepath.Join(root, "go.mod")):
		if commands.Build == "" {
			commands.Build = "go build ./..."
		}
		if commands.Type == "" {
			commands.Type = "go vet ./..."
		}
		if commands.Test == "" {
			commands.Test = "go test ./..."
		}
	case fileExists(filepath.Join(root, "package.json")):
		if commands.Build == "" {
			commands.Build = "npm run build"
		}
		if commands.Type == "" {
			commands.Type = "npx tsc --noEmit"
		}
		if commands.Lint == "" {
			commands.Lint = "npm run lint"
		}
		if commands.Test == "" {
			commands.Test = "npm test"
		}
	case fileExists(filepath.Join(root, "Cargo.toml")):
		if commands.Build == "" {
			commands.Build = "cargo build"
		}
		if commands.Lint == "" {
			commands.Lint = "cargo clippy"
		}
		if commands.Test == "" {
			commands.Test = "cargo test"
		}
	case fileExists(filepath.Join(root, "pyproject.toml")):
		if commands.Build == "" {
			commands.Build = "python -m build"
		}
		if commands.Type == "" {
			commands.Type = "pyright ."
		}
		if commands.Lint == "" {
			commands.Lint = "ruff check ."
		}
		if commands.Test == "" {
			commands.Test = "pytest"
		}
	case fileExists(filepath.Join(root, "Makefile")):
		if commands.Build == "" {
			commands.Build = "make build"
		}
		if commands.Lint == "" {
			commands.Lint = "make lint"
		}
		if commands.Test == "" {
			commands.Test = "make test"
		}
	}
	return commands
}

func mergeCodexVerificationCommands(dst *codexVerificationCommands, src codexVerificationCommands) {
	if dst.Build == "" {
		dst.Build = strings.TrimSpace(src.Build)
	}
	if dst.Type == "" {
		dst.Type = strings.TrimSpace(src.Type)
	}
	if dst.Lint == "" {
		dst.Lint = strings.TrimSpace(src.Lint)
	}
	if dst.Test == "" {
		dst.Test = strings.TrimSpace(src.Test)
	}
}

func loadVerificationCommandsFromMarkdown(path, heading string) codexVerificationCommands {
	data, err := os.ReadFile(path)
	if err != nil {
		return codexVerificationCommands{}
	}

	content := string(data)
	if section := extractMarkdownSection(content, heading); section != "" {
		content = section
	}
	return extractVerificationCommands(content)
}

func extractMarkdownSection(content, heading string) string {
	lines := strings.Split(content, "\n")
	normalizedHeading := strings.ToLower(strings.TrimSpace(heading))
	targetLevel := markdownHeadingLevel(heading)
	capturing := false
	inFence := false
	var section []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !capturing && strings.ToLower(trimmed) == normalizedHeading {
			capturing = true
			continue
		}
		if capturing {
			if strings.HasPrefix(trimmed, "```") {
				inFence = !inFence
				section = append(section, line)
				continue
			}
			if !inFence {
				level := markdownHeadingLevel(trimmed)
				if level > 0 && targetLevel > 0 && level <= targetLevel {
					break
				}
			}
			section = append(section, line)
		}
	}

	return strings.TrimSpace(strings.Join(section, "\n"))
}

func markdownHeadingLevel(line string) int {
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, "#") {
		return 0
	}
	level := 0
	for level < len(trimmed) && trimmed[level] == '#' {
		level++
	}
	if level == 0 || level >= len(trimmed) || trimmed[level] != ' ' {
		return 0
	}
	return level
}

func extractVerificationCommands(content string) codexVerificationCommands {
	commands := codexVerificationCommands{}
	pendingKind := ""

	for _, rawLine := range strings.Split(content, "\n") {
		line := strings.TrimSpace(rawLine)
		if line == "" || strings.HasPrefix(line, "```") {
			continue
		}
		if kind, command, ok := parseVerificationCommandTableLine(line); ok {
			setVerificationCommand(&commands, kind, command)
			pendingKind = ""
			continue
		}
		if kind, command, ok := parseLabeledVerificationCommand(line); ok {
			setVerificationCommand(&commands, kind, command)
			pendingKind = ""
			continue
		}
		if kind := parseVerificationCommandComment(line); kind != "" {
			pendingKind = kind
			continue
		}
		if command := extractVerificationCommandValue(line); command != "" {
			if pendingKind != "" {
				setVerificationCommand(&commands, pendingKind, command)
				pendingKind = ""
				continue
			}
			if kind := detectVerificationCommandKind(command); kind != "" {
				setVerificationCommand(&commands, kind, command)
			}
		}
	}

	return commands
}

func parseVerificationCommandTableLine(line string) (string, string, bool) {
	if !strings.HasPrefix(strings.TrimSpace(line), "|") {
		return "", "", false
	}
	parts := strings.Split(line, "|")
	if len(parts) < 4 {
		return "", "", false
	}
	label := strings.TrimSpace(parts[1])
	kind := normalizeVerificationCommandKind(label)
	if kind == "" {
		return "", "", false
	}
	command := extractVerificationCommandValue(strings.TrimSpace(parts[2]))
	if command == "" {
		return "", "", false
	}
	return kind, command, true
}

func parseLabeledVerificationCommand(line string) (string, string, bool) {
	line = strings.TrimSpace(strings.TrimLeft(line, "-* "))
	idx := strings.Index(line, ":")
	if idx <= 0 {
		return "", "", false
	}
	kind := normalizeVerificationCommandKind(line[:idx])
	if kind == "" {
		return "", "", false
	}
	command := extractVerificationCommandValue(line[idx+1:])
	if command == "" {
		return "", "", false
	}
	return kind, command, true
}

func parseVerificationCommandComment(line string) string {
	trimmed := strings.TrimSpace(line)
	if strings.HasPrefix(trimmed, "#") {
		return normalizeVerificationCommandKind(strings.TrimSpace(strings.TrimLeft(trimmed, "#")))
	}
	if strings.HasPrefix(trimmed, "//") {
		return normalizeVerificationCommandKind(strings.TrimSpace(strings.TrimPrefix(trimmed, "//")))
	}
	return ""
}

func extractVerificationCommandValue(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}
	if strings.Count(text, "`") >= 2 {
		start := strings.Index(text, "`")
		end := strings.Index(text[start+1:], "`")
		if start >= 0 && end >= 0 {
			return strings.TrimSpace(text[start+1 : start+1+end])
		}
	}

	text = strings.TrimSpace(strings.TrimLeft(text, "-* "))
	if text == "" || strings.HasPrefix(text, "#") || strings.HasPrefix(text, "|") {
		return ""
	}
	if looksLikeVerificationCommand(text) {
		return text
	}
	return ""
}

func looksLikeVerificationCommand(text string) bool {
	if detectVerificationCommandKind(text) != "" {
		return true
	}
	fields := strings.Fields(text)
	if len(fields) == 0 {
		return false
	}
	switch fields[0] {
	case "printf", "echo", "true", "false", "make", "sh", "bash":
		return true
	default:
		return false
	}
}

func detectVerificationCommandKind(command string) string {
	lower := strings.ToLower(strings.TrimSpace(command))
	switch {
	case strings.HasPrefix(lower, "go build"),
		strings.HasPrefix(lower, "npm run build"),
		strings.HasPrefix(lower, "pnpm build"),
		strings.HasPrefix(lower, "yarn build"),
		strings.HasPrefix(lower, "cargo build"),
		strings.HasPrefix(lower, "python -m build"),
		strings.HasPrefix(lower, "make build"):
		return "build"
	case strings.HasPrefix(lower, "go test"),
		strings.HasPrefix(lower, "npm test"),
		strings.HasPrefix(lower, "pnpm test"),
		strings.HasPrefix(lower, "yarn test"),
		strings.HasPrefix(lower, "cargo test"),
		strings.HasPrefix(lower, "pytest"),
		strings.HasPrefix(lower, "make test"):
		return "tests"
	case strings.HasPrefix(lower, "go vet"),
		strings.Contains(lower, "tsc --noemit"),
		strings.HasPrefix(lower, "pyright"),
		strings.HasPrefix(lower, "mypy"):
		return "types"
	case strings.HasPrefix(lower, "golangci-lint"),
		strings.HasPrefix(lower, "npm run lint"),
		strings.HasPrefix(lower, "pnpm lint"),
		strings.HasPrefix(lower, "yarn lint"),
		strings.HasPrefix(lower, "cargo clippy"),
		strings.HasPrefix(lower, "ruff check"),
		strings.HasPrefix(lower, "make lint"):
		return "lint"
	default:
		return ""
	}
}

func normalizeVerificationCommandKind(label string) string {
	lower := strings.ToLower(strings.TrimSpace(strings.Trim(label, "*`")))
	switch {
	case strings.Contains(lower, "build"):
		return "build"
	case strings.Contains(lower, "type"), strings.Contains(lower, "vet"):
		return "types"
	case strings.Contains(lower, "lint"):
		return "lint"
	case strings.Contains(lower, "test"):
		return "tests"
	default:
		return ""
	}
}

func setVerificationCommand(commands *codexVerificationCommands, kind, command string) {
	command = strings.TrimSpace(command)
	if command == "" {
		return
	}
	switch kind {
	case "build":
		if commands.Build == "" {
			commands.Build = command
		}
	case "types":
		if commands.Type == "" {
			commands.Type = command
		}
	case "lint":
		if commands.Lint == "" {
			commands.Lint = command
		}
	case "tests":
		if commands.Test == "" {
			commands.Test = command
		}
	}
}

func runVerificationStep(root, name, command string) codexVerificationStep {
	if strings.TrimSpace(command) == "" {
		return codexVerificationStep{
			Name:    name,
			Skipped: true,
			Passed:  true,
			Summary: "no command resolved; skipped",
		}
	}

	output, exitCode, err := runShellCommand(root, command, 5*time.Minute)
	step := codexVerificationStep{
		Name:     name,
		Command:  command,
		Passed:   err == nil,
		ExitCode: exitCode,
		Summary:  successSummaryForStep(name, exitCode, err),
		Output:   output,
	}
	if err != nil {
		step.Summary = failureSummaryForStep(name, exitCode, output, err)
	}
	return step
}

func verifyCodexBuildClaims(root string, manifest codexContinueManifest) codexClaimVerification {
	claimsRel := "last-build-claims.json"
	if manifest.Present && strings.TrimSpace(manifest.Data.ClaimsPath) != "" {
		claimsRel = strings.TrimPrefix(strings.TrimSpace(manifest.Data.ClaimsPath), ".aether/data/")
	}

	var claims codexBuildClaims
	if err := store.LoadJSON(claimsRel, &claims); err != nil {
		return codexClaimVerification{
			Present: false,
			Passed:  !manifestRequiresBuilderClaims(manifest),
			Skipped: !manifestRequiresBuilderClaims(manifest),
			Summary: missingClaimsSummary(manifest),
		}
	}

	checked := 0
	mismatches := []string{}
	for _, rel := range append(append(append([]string{}, claims.FilesCreated...), claims.FilesModified...), claims.TestsWritten...) {
		rel = strings.TrimSpace(rel)
		if rel == "" {
			continue
		}
		checked++
		if !fileExists(filepath.Join(root, rel)) {
			mismatches = append(mismatches, fmt.Sprintf("%s does not exist", rel))
		}
	}
	if len(mismatches) > 0 {
		return codexClaimVerification{
			Present:    true,
			Passed:     false,
			Summary:    fmt.Sprintf("worker claims mismatch: %d missing paths", len(mismatches)),
			Checked:    checked,
			Mismatches: mismatches,
		}
	}

	if checked == 0 && manifestRequiresBuilderClaims(manifest) {
		if manifestUsesSyntheticDispatch(manifest) {
			return codexClaimVerification{
				Present: true,
				Passed:  false,
				Summary: "builder claims file is empty because the build ran in simulated mode; rerun `aether build <phase>` without `--synthetic` before `aether continue` can advance",
				Checked: 0,
			}
		}
		return codexClaimVerification{
			Present: true,
			Passed:  false,
			Summary: emptyClaimsFailureSummary(manifest),
			Checked: 0,
		}
	}

	summary := "builder claims verified"
	if checked == 0 {
		summary = "builder claims file present but empty"
	}
	return codexClaimVerification{
		Present: true,
		Passed:  true,
		Summary: summary,
		Checked: checked,
	}
}

func runCodexContinueGates(phase colony.Phase, manifest codexContinueManifest, verification codexContinueVerificationReport, assessment codexContinueAssessment, now time.Time) codexContinueGateReport {
	checks := []gateCheck{}
	blockers := []string{}

	manifestCheck := gateCheck{Name: "manifest_present", Passed: manifest.Present, Detail: "build manifest present"}
	if !manifest.Present {
		manifestCheck.Detail = fmt.Sprintf("build manifest is missing for phase %d", phase.ID)
		blockers = append(blockers, manifestCheck.Detail)
	}
	checks = append(checks, manifestCheck)

	checks = append(checks, gateCheck{
		Name:   "verification_steps_passed",
		Passed: verification.ChecksPassed,
		Detail: continueVerificationDetail(verification),
	})
	if !verification.ChecksPassed {
		blockers = append(blockers, verification.BlockingIssues...)
	}

	evidenceCheck := gateCheck{Name: "implementation_evidence", Passed: assessment.PositiveEvidence, Detail: "task or claim evidence recorded for the verified phase"}
	if !assessment.PositiveEvidence {
		evidenceCheck.Detail = "verification passed but no implementation evidence or reconciliation was recorded"
		blockers = append(blockers, assessment.BlockingIssues...)
	}
	checks = append(checks, evidenceCheck)

	operationalCheck := gateCheck{Name: "operational_evidence", Passed: true, Detail: "no operational worker issues were recorded"}
	if len(assessment.OperationalIssues) > 0 {
		operationalCheck.Detail = fmt.Sprintf("%d operational worker issues recorded; continue is using verification-led truth instead", len(assessment.OperationalIssues))
	}
	checks = append(checks, operationalCheck)

	flagCheck := checkNoCriticalFlags()
	checks = append(checks, flagCheck)
	if !flagCheck.Passed {
		blockers = append(blockers, flagCheck.Detail)
	}

	return codexContinueGateReport{
		Phase:          phase.ID,
		GeneratedAt:    now.Format(time.RFC3339),
		Checks:         checks,
		Passed:         len(blockers) == 0,
		BlockingIssues: uniqueSortedStrings(append(blockers, assessment.BlockingIssues...)),
	}
}

func continueVerificationDetail(verification codexContinueVerificationReport) string {
	if verification.ChecksPassed {
		return "verification commands passed"
	}
	if len(verification.BlockingIssues) == 0 {
		return "verification commands failed"
	}
	return strings.Join(verification.BlockingIssues, "; ")
}

func attachContinueClaimVerification(verification codexContinueVerificationReport, assessment codexContinueAssessment) codexContinueVerificationReport {
	if verification.Claims.Passed || verification.Claims.Skipped || assessment.Passed {
		return verification
	}

	blockers := append([]string{}, verification.BlockingIssues...)
	if summary := strings.TrimSpace(verification.Claims.Summary); summary != "" {
		blockers = append(blockers, summary)
	}
	for _, mismatch := range verification.Claims.Mismatches {
		if mismatch = strings.TrimSpace(mismatch); mismatch != "" {
			blockers = append(blockers, mismatch)
		}
	}

	verification.ChecksPassed = false
	verification.Passed = false
	verification.BlockingIssues = uniqueSortedStrings(blockers)
	return verification
}

func plannedCodexContinueClosedWorkers(manifest codexContinueManifest, assessment codexContinueAssessment) []codexContinueClosedWorker {
	if !manifest.Present || len(manifest.Data.Dispatches) == 0 {
		return nil
	}
	closed := make([]codexContinueClosedWorker, 0, len(manifest.Data.Dispatches))
	reconciled := make(map[string]struct{}, len(assessment.ReconciledTasks))
	for _, taskID := range assessment.ReconciledTasks {
		reconciled[taskID] = struct{}{}
	}
	for _, dispatch := range manifest.Data.Dispatches {
		status := strings.TrimSpace(dispatch.Status)
		if status == "" {
			status = "completed"
		}
		summary := continueWorkerCloseSummary(dispatch)
		if _, ok := reconciled[dispatch.TaskID]; ok && status != "completed" {
			status = "manually-reconciled"
			summary = "Task was manually reconciled before continue advancement"
		} else if assessment.PartialSuccess && status != "completed" {
			summary = fmt.Sprintf("%s Phase verification passed independently during continue.", continueWorkerCloseSummary(dispatch))
		}
		closed = append(closed, codexContinueClosedWorker{
			Stage:   strings.TrimSpace(dispatch.Stage),
			Caste:   strings.TrimSpace(dispatch.Caste),
			Name:    dispatch.Name,
			Task:    strings.TrimSpace(dispatch.Task),
			Status:  status,
			Summary: summary,
		})
	}
	return closed
}

func applyCodexContinueWorkerClosures(closed []codexContinueClosedWorker) error {
	if len(closed) == 0 {
		return nil
	}

	spawnTree := agent.NewSpawnTree(store, "spawn-tree.txt")
	for _, detail := range closed {
		if err := spawnTree.UpdateStatusPreserveActivity(detail.Name, detail.Status, detail.Summary); err != nil {
			return fmt.Errorf("failed to close worker %s: %w", detail.Name, err)
		}
	}
	return nil
}

func closedWorkerNames(details []codexContinueClosedWorker) []string {
	names := make([]string, 0, len(details))
	for _, detail := range details {
		names = append(names, detail.Name)
	}
	return names
}

func continueWorkerFlowWithWatcher(flow []codexContinueWorkerFlowStep, watcherFlow *codexContinueWorkerFlowStep) []codexContinueWorkerFlowStep {
	combined := make([]codexContinueWorkerFlowStep, 0, len(flow)+1)
	if watcherFlow != nil && strings.TrimSpace(watcherFlow.Name) != "" {
		combined = append(combined, *watcherFlow)
	}
	return append(combined, flow...)
}

func continueHousekeepingFlowStep(housekeeping signalHousekeepingResult) codexContinueWorkerFlowStep {
	return codexContinueWorkerFlowStep{
		Stage:   "housekeeping",
		Caste:   "system",
		Name:    "Signal housekeeping",
		Status:  "completed",
		Summary: continueHousekeepingSummary(housekeeping),
	}
}

func continueWorkerFlowStatus(status string) string {
	status = strings.TrimSpace(status)
	if status == "" {
		return "completed"
	}
	return status
}

func continueWatcherDefaultSummary(status string) string {
	switch continueWorkerFlowStatus(status) {
	case "completed", "manually-reconciled":
		return "Continue watcher completed independent verification"
	default:
		return fmt.Sprintf("continue watcher finished with status %s", continueWorkerFlowStatus(status))
	}
}

func continueWatcherFlowSummary(name, status, summary string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "watcher"
	}
	status = continueWorkerFlowStatus(status)
	switch status {
	case "completed", "manually-reconciled":
		return fmt.Sprintf("Watcher %s completed independent verification before advancement", name)
	default:
		summary = strings.TrimSpace(summary)
		if summary == "" {
			return fmt.Sprintf("Watcher %s closed independent verification with status %s", name, status)
		}
		return fmt.Sprintf("Watcher %s closed independent verification with status %s: %s", name, status, summary)
	}
}

func continueWatcherResultSummary(result codex.DispatchResult) string {
	if result.WorkerResult != nil {
		if len(result.WorkerResult.Blockers) > 0 {
			return strings.Join(result.WorkerResult.Blockers, "; ")
		}
		if summary := strings.TrimSpace(result.WorkerResult.Summary); summary != "" && !strings.HasPrefix(summary, "FakeInvoker completed task") {
			return summary
		}
	}
	if result.Error != nil {
		return strings.TrimSpace(result.Error.Error())
	}
	return ""
}

func continueWorkerFlowEvents(now time.Time, workerFlow []codexContinueWorkerFlowStep) []string {
	if len(workerFlow) == 0 {
		return nil
	}

	events := make([]string, 0, len(workerFlow))
	for _, step := range workerFlow {
		switch strings.TrimSpace(step.Stage) {
		case "review":
			summary := strings.TrimSpace(step.Summary)
			if summary == "" {
				summary = continueReviewFlowSummary(step)
			}
			events = append(events, fmt.Sprintf("%s|continue_review|continue|%s", now.Format(time.RFC3339), summary))
		case "verification":
			summary := strings.TrimSpace(step.Summary)
			if summary == "" {
				summary = "Watcher verification completed"
			}
			events = append(events, fmt.Sprintf("%s|watcher_verification|continue|%s", now.Format(time.RFC3339), summary))
		case "housekeeping":
			summary := strings.TrimSpace(step.Summary)
			if summary == "" {
				summary = "signal housekeeping completed"
			}
			events = append(events, fmt.Sprintf("%s|signal_housekeeping|continue|Signal housekeeping completed: %s", now.Format(time.RFC3339), summary))
		}
	}
	return events
}

func recordBlockedContinueWorkerFlow(state colony.ColonyState, now time.Time, workerFlow []codexContinueWorkerFlowStep) (colony.ColonyState, error) {
	if len(workerFlow) == 0 {
		return state, nil
	}
	if err := recordContinueWorkerFlow(workerFlow); err != nil {
		return state, err
	}

	updated := state
	updated.Events = append(trimmedEvents(updated.Events), continueWorkerFlowEvents(now, workerFlow)...)
	if err := store.SaveJSON("COLONY_STATE.json", updated); err != nil {
		return state, fmt.Errorf("failed to save colony state: %w", err)
	}
	return updated, nil
}

func recordContinueWorkerFlow(workerFlow []codexContinueWorkerFlowStep) error {
	if len(workerFlow) == 0 || store == nil {
		return nil
	}

	spawnTree := agent.NewSpawnTree(store, "spawn-tree.txt")
	for _, step := range workerFlow {
		name := strings.TrimSpace(step.Name)
		if name == "" {
			continue
		}
		task := continueWorkerFlowTask(step)
		if err := spawnTree.RecordSpawn("Continue", strings.TrimSpace(step.Caste), name, task, 1); err != nil {
			return fmt.Errorf("failed to record continue flow %s: %w", name, err)
		}
		if err := spawnTree.UpdateStatus(name, continueWorkerFlowStatus(step.Status), continueWorkerFlowLogSummary(step)); err != nil {
			return fmt.Errorf("failed to finalize continue flow %s: %w", name, err)
		}
	}
	return nil
}

func continueWorkerFlowTask(step codexContinueWorkerFlowStep) string {
	if task := strings.TrimSpace(step.Task); task != "" {
		return task
	}
	switch strings.TrimSpace(step.Stage) {
	case "review":
		if summary := strings.TrimSpace(step.Summary); summary != "" {
			return summary
		}
		return continueReviewFlowTask(step)
	case "verification":
		return "Independent verification before advancement"
	case "housekeeping":
		return "Expire stale and low-value pheromone signals"
	default:
		if summary := strings.TrimSpace(step.Summary); summary != "" {
			return summary
		}
		return strings.TrimSpace(step.Name)
	}
}

func continueWorkerFlowLogSummary(step codexContinueWorkerFlowStep) string {
	switch strings.TrimSpace(step.Stage) {
	case "review":
		if summary := strings.TrimSpace(step.Summary); summary != "" {
			return summary
		}
		return continueReviewFlowSummary(step)
	case "verification":
		if summary := strings.TrimSpace(step.Summary); summary != "" {
			return summary
		}
		return continueWatcherFlowSummary(step.Name, step.Status, "")
	case "housekeeping":
		if summary := strings.TrimSpace(step.Summary); summary != "" {
			return summary
		}
		return "Signal housekeeping completed"
	default:
		if summary := strings.TrimSpace(step.Summary); summary != "" {
			return summary
		}
		return strings.TrimSpace(step.Task)
	}
}

func continueReviewFlowTask(step codexContinueWorkerFlowStep) string {
	return continueReviewTaskForCaste(step.Caste)
}

func continueReviewTaskForCaste(caste string) string {
	switch strings.TrimSpace(caste) {
	case "gatekeeper":
		return "Gatekeeper continue review"
	case "auditor":
		return "Auditor continue review"
	case "probe":
		return "Probe continue review"
	default:
		return "Continue review"
	}
}

func continueReviewFlowSummary(step codexContinueWorkerFlowStep) string {
	name := strings.TrimSpace(step.Name)
	if name == "" {
		name = "review worker"
	}
	status := continueWorkerFlowStatus(step.Status)
	role := strings.TrimSpace(step.Caste)
	if role != "" {
		role = strings.ToUpper(role[:1]) + role[1:]
	}
	if role == "" {
		role = "Review"
	}
	switch status {
	case "completed", "manually-reconciled":
		return fmt.Sprintf("%s %s completed continue review before advancement", role, name)
	default:
		return fmt.Sprintf("%s %s closed continue review with status %s", role, name, status)
	}
}

func continueHousekeepingSummary(housekeeping signalHousekeepingResult) string {
	summary := fmt.Sprintf("%d active -> %d active", housekeeping.ActiveBefore, housekeeping.ActiveAfter)
	if housekeeping.Updated == 0 {
		return summary
	}
	return fmt.Sprintf("%s (%d expired, %d low-strength, %d stale continue)", summary, housekeeping.ExpiredByTime, housekeeping.DeactivatedByStrength, housekeeping.ExpiredWorkerContinue)
}

func updateCodexContinueContext(phase colony.Phase, manifest codexContinueManifest, closedWorkers []codexContinueClosedWorker, now time.Time) error {
	data, err := readContextDocument()
	if err != nil {
		return nil
	}
	content := string(data)
	content = replaceContextTableRow(content, "Last Updated", now.Format(time.RFC3339))
	content = replaceContextTableRow(content, "Safe to Clear?", "YES — Build complete, ready to continue")
	content = replaceBuildInProgressWithComplete(content, "verified", fmt.Sprintf("Phase %d ready to advance", phase.ID))
	for _, worker := range closedWorkers {
		content = markWorkerComplete(content, worker.Name, worker.Status, now.Format(time.RFC3339))
	}
	return writeContextDocument(content)
}

func runShellCommand(root, command string, timeout time.Duration) (string, int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(ctx, "cmd", "/C", command)
	} else {
		cmd = exec.CommandContext(ctx, "sh", "-c", command)
	}
	cmd.Dir = root
	cmd.Env = append(os.Environ(), "AETHER_OUTPUT_MODE=")
	output, err := cmd.CombinedOutput()
	trimmed := trimCommandOutput(string(output))

	exitCode := 0
	if cmd.ProcessState != nil {
		exitCode = cmd.ProcessState.ExitCode()
	}
	return trimmed, exitCode, err
}

func trimCommandOutput(output string) string {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) <= 20 {
		return strings.TrimSpace(output)
	}
	return strings.Join(lines[len(lines)-20:], "\n")
}

func successSummaryForStep(name string, exitCode int, err error) string {
	if err != nil {
		return fmt.Sprintf("%s failed", name)
	}
	switch name {
	case "build":
		return "build succeeded"
	case "types":
		return "type checks passed"
	case "lint":
		return "lint passed"
	case "tests":
		return "tests passed"
	default:
		return fmt.Sprintf("%s passed", name)
	}
}

func failureSummaryForStep(name string, exitCode int, output string, err error) string {
	if strings.TrimSpace(output) != "" {
		lines := strings.Split(strings.TrimSpace(output), "\n")
		return fmt.Sprintf("%s failed (exit %d): %s", name, exitCode, strings.TrimSpace(lines[len(lines)-1]))
	}
	return fmt.Sprintf("%s failed (exit %d): %v", name, exitCode, err)
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func displayOptionalDataPath(rel string) string {
	if strings.TrimSpace(rel) == "" {
		return ""
	}
	return displayDataPath(rel)
}

func manifestRequiresBuilderClaims(manifest codexContinueManifest) bool {
	if !manifest.Present {
		return false
	}
	for _, dispatch := range manifest.Data.Dispatches {
		if strings.EqualFold(strings.TrimSpace(dispatch.Caste), "builder") {
			return true
		}
	}
	return false
}

func allDispatchesCompleted(manifest codexContinueManifest) bool {
	if !manifest.Present || len(manifest.Data.Dispatches) == 0 {
		return false
	}
	for _, dispatch := range manifest.Data.Dispatches {
		if dispatch.Status != "completed" {
			return false
		}
	}
	return true
}

func emptyClaimsFailureSummary(manifest codexContinueManifest) string {
	if !manifest.Present {
		return "builder claims file is empty but this phase dispatched builders"
	}
	mode := strings.TrimSpace(manifest.Data.DispatchMode)
	if mode == "" {
		return "builder claims file is empty and the build manifest does not record a simulated dispatch mode"
	}
	return fmt.Sprintf("builder claims file is empty but this phase dispatched builders in %s mode", mode)
}

func manifestUsesSyntheticDispatch(manifest codexContinueManifest) bool {
	if !manifest.Present {
		return false
	}
	mode := strings.ToLower(strings.TrimSpace(manifest.Data.DispatchMode))
	return mode == "simulated" || mode == "synthetic"
}

func missingClaimsSummary(manifest codexContinueManifest) string {
	if manifestRequiresBuilderClaims(manifest) {
		return "no builder claims file found for a phase that dispatched builders"
	}
	return "no builder claims file found; skipped"
}
