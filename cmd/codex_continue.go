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
	Phase          int                     `json:"phase"`
	GeneratedAt    string                  `json:"generated_at"`
	Steps          []codexVerificationStep `json:"steps"`
	Claims         codexClaimVerification  `json:"claims"`
	ChecksPassed   bool                    `json:"checks_passed"`
	Passed         bool                    `json:"passed"`
	BlockingIssues []string                `json:"blocking_issues,omitempty"`
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
	Summary            string                        `json:"summary,omitempty"`
	ClosedWorkers      []string                      `json:"closed_workers,omitempty"`
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
	Name    string `json:"name"`
	Status  string `json:"status"`
	Summary string `json:"summary,omitempty"`
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
	now := time.Now().UTC()

	verification := runCodexContinueVerification(root, phase.ID, manifest)
	assessment := assessCodexContinue(phase, manifest, verification, options, now)
	gates := runCodexContinueGates(phase, manifest, verification, assessment, now)

	if tracer != nil && state.RunID != nil {
		_ = tracer.LogArtifact(*state.RunID, "continue.verification", map[string]interface{}{
			"phase":           phase.ID,
			"checks_passed":   verification.ChecksPassed,
			"steps_count":     len(verification.Steps),
			"claims_present":  verification.Claims.Present,
			"claims_passed":   verification.Claims.Passed,
		})
		_ = tracer.LogArtifact(*state.RunID, "continue.assessment", map[string]interface{}{
			"phase":            phase.ID,
			"passed":           assessment.Passed,
			"partial_success":  assessment.PartialSuccess,
			"tasks_count":      len(assessment.Tasks),
			"operational_issues": len(assessment.OperationalIssues),
		})
		_ = tracer.LogArtifact(*state.RunID, "continue.gates", map[string]interface{}{
			"phase":   phase.ID,
			"passed":  gates.Passed,
			"checks":  len(gates.Checks),
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
		_ = store.SaveJSON(continueReportRel, codexContinueReport{
			Phase:              phase.ID,
			GeneratedAt:        now.Format(time.RFC3339),
			Manifest:           displayOptionalDataPath(manifest.Path),
			VerificationReport: displayDataPath(verificationReportRel),
			GateReport:         displayDataPath(gateReportRel),
			Summary:            summary,
			PartialSuccess:     assessment.PartialSuccess,
			OperationalIssues:  append([]string{}, assessment.OperationalIssues...),
			Tasks:              append([]codexContinueTaskAssessment{}, assessment.Tasks...),
			Recovery:           assessment.Recovery,
			Advanced:           false,
			Completed:          false,
			Next:               continueNextCommandForAssessment(assessment),
		})
		updateSessionSummary("continue", continueNextCommandForAssessment(assessment), summary)

		result := map[string]interface{}{
			"advanced":            false,
			"blocked":             true,
			"partial_success":     assessment.PartialSuccess,
			"current_phase":       state.CurrentPhase,
			"phase_name":          phase.Name,
			"state":               state.State,
			"next":                continueNextCommandForAssessment(assessment),
			"verification":        verification,
			"assessment":          assessment,
			"task_evidence":       assessment.Tasks,
			"gates":               gates,
			"verification_report": displayDataPath(verificationReportRel),
			"gate_report":         displayDataPath(gateReportRel),
			"continue_report":     displayDataPath(continueReportRel),
			"operational_issues":  assessment.OperationalIssues,
			"recovery":            assessment.Recovery,
			"reconciled_tasks":    assessment.ReconciledTasks,
			"blocking_issues":     blockers,
		}
		return result, state, phase, nil, nil, false, nil
	}

	closedWorkerDetails, err := closeCodexContinueWorkers(manifest, assessment)
	if err != nil {
		return nil, state, phase, nil, nil, false, err
	}
	closedWorkers := closedWorkerNames(closedWorkerDetails)
	updated := state
	updated.Plan.Phases[currentIdx].Status = colony.PhaseCompleted
	for i := range updated.Plan.Phases[currentIdx].Tasks {
		updated.Plan.Phases[currentIdx].Tasks[i].Status = colony.TaskCompleted
	}
	updated.BuildStartedAt = nil

	final := currentIdx == len(updated.Plan.Phases)-1
	var nextPhase *colony.Phase
	nextCommand := "aether seal"
	if final {
		updated.State = colony.StateCOMPLETED
		updated.CurrentPhase = phase.ID
		updated.Events = append(trimmedEvents(updated.Events),
			fmt.Sprintf("%s|verification_passed|continue|Build verification passed for final phase %d", now.Format(time.RFC3339), phase.ID),
			fmt.Sprintf("%s|gate_passed|continue|Continue gates passed for final phase %d", now.Format(time.RFC3339), phase.ID),
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
		updated.Events = append(trimmedEvents(updated.Events),
			fmt.Sprintf("%s|verification_passed|continue|Build verification passed for phase %d", now.Format(time.RFC3339), phase.ID),
			fmt.Sprintf("%s|gate_passed|continue|Continue gates passed for phase %d", now.Format(time.RFC3339), phase.ID),
			fmt.Sprintf("%s|phase_advanced|continue|Completed phase %d, ready for phase %d", now.Format(time.RFC3339), phase.ID, nextIdx+1),
		)
	}

	if err := store.SaveJSON("COLONY_STATE.json", updated); err != nil {
		return nil, state, phase, nextPhase, nil, final, fmt.Errorf("failed to save colony state: %w", err)
	}
	if err := updateCodexContinueContext(phase, manifest, closedWorkerDetails, now); err != nil {
		return nil, updated, phase, nextPhase, nil, final, err
	}
	housekeeping, housekeepingErr := runSignalHousekeeping(now, false)
	if housekeepingErr != nil {
		return nil, updated, phase, nextPhase, nil, final, housekeepingErr
	}

	summary := fmt.Sprintf("Phase %d verified and advanced", phase.ID)
	if assessment.PartialSuccess {
		summary = fmt.Sprintf("Phase %d verified and advanced with partial operational success", phase.ID)
	}

	continueReportRel := filepath.ToSlash(filepath.Join("build", fmt.Sprintf("phase-%d", phase.ID), "continue.json"))
	if err := store.SaveJSON(continueReportRel, codexContinueReport{
		Phase:              phase.ID,
		GeneratedAt:        now.Format(time.RFC3339),
		Manifest:           displayOptionalDataPath(manifest.Path),
		VerificationReport: displayDataPath(verificationReportRel),
		GateReport:         displayDataPath(gateReportRel),
		Summary:            summary,
		ClosedWorkers:      closedWorkers,
		PartialSuccess:     assessment.PartialSuccess,
		OperationalIssues:  append([]string{}, assessment.OperationalIssues...),
		Tasks:              append([]codexContinueTaskAssessment{}, assessment.Tasks...),
		Recovery:           assessment.Recovery,
		Advanced:           true,
		Completed:          final,
		Next:               nextCommand,
	}); err != nil {
		return nil, updated, phase, nextPhase, &housekeeping, final, fmt.Errorf("failed to write continue report: %w", err)
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
		"verification_report": displayDataPath(verificationReportRel),
		"gate_report":         displayDataPath(gateReportRel),
		"continue_report":     displayDataPath(continueReportRel),
		"closed_workers":      closedWorkers,
		"operational_issues":  assessment.OperationalIssues,
		"recovery":            assessment.Recovery,
		"reconciled_tasks":    assessment.ReconciledTasks,
		"signal_housekeeping": housekeeping,
	}
	if nextPhase != nil {
		result["next_phase"] = nextPhase.ID
		result["next_phase_name"] = nextPhase.Name
	}
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

func runCodexContinueVerification(root string, phaseID int, manifest codexContinueManifest) codexContinueVerificationReport {
	now := time.Now().UTC()
	commands := resolveCodexVerificationCommands(root)
	steps := []codexVerificationStep{
		runVerificationStep(root, "build", commands.Build),
		runVerificationStep(root, "types", commands.Type),
		runVerificationStep(root, "lint", commands.Lint),
		runVerificationStep(root, "tests", commands.Test),
	}
	claims := verifyCodexBuildClaims(root, manifest)

	checksPassed := true
	blockers := []string{}
	for _, step := range steps {
		if !step.Passed && !step.Skipped {
			checksPassed = false
			blockers = append(blockers, fmt.Sprintf("%s failed: %s", step.Name, step.Summary))
		}
	}

	return codexContinueVerificationReport{
		Phase:          phaseID,
		GeneratedAt:    now.Format(time.RFC3339),
		Steps:          steps,
		Claims:         claims,
		ChecksPassed:   checksPassed,
		Passed:         checksPassed,
		BlockingIssues: blockers,
	}
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
	completedTaskIDs := make(map[string]bool, len(phase.Tasks))
	operationalIssues := []string{}
	for _, dispatch := range manifest.Data.Dispatches {
		status := strings.TrimSpace(dispatch.Status)
		if dispatch.TaskID != "" {
			dispatchStatuses[dispatch.TaskID] = append(dispatchStatuses[dispatch.TaskID], status)
		}
		if status == "completed" && dispatch.TaskID != "" {
			completedTaskIDs[dispatch.TaskID] = true
		}
		if status != "" && status != "completed" {
			operationalIssues = append(operationalIssues, fmt.Sprintf("%s (%s)", dispatch.Name, status))
		}
	}
	operationalIssues = uniqueSortedStrings(operationalIssues)

	positiveEvidence := verification.Claims.Passed || verification.Claims.Skipped || len(completedTaskIDs) > 0 || len(reconciled) > 0
	tasks := make([]codexContinueTaskAssessment, 0, len(phase.Tasks))
	redispatchTasks := make([]string, 0, len(phase.Tasks))

	for idx, task := range phase.Tasks {
		taskID := buildTaskID(task, idx)
		statuses := uniqueSortedStrings(dispatchStatuses[taskID])
		_, reconciledTask := reconciled[taskID]
		outcome, summary, recovery := classifyContinueTaskAssessment(taskID, statuses, verification.ChecksPassed, positiveEvidence, reconciledTask)
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

	blockingIssues := []string{}
	if !verification.ChecksPassed {
		blockingIssues = append(blockingIssues, verification.BlockingIssues...)
	}
	if verification.ChecksPassed && !positiveEvidence {
		blockingIssues = append(blockingIssues, "verification passed but no implementation evidence was recorded; reconcile completed tasks or redispatch missing work")
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

func classifyContinueTaskAssessment(taskID string, statuses []string, verificationPassed, positiveEvidence, reconciled bool) (string, string, string) {
	if reconciled {
		if verificationPassed {
			return "manually_reconciled", "Task was manually reconciled and the phase verification passed.", "reverify"
		}
		return "manually_reconciled", "Task was manually reconciled, but phase verification still failed.", "reverify"
	}

	if verificationPassed {
		if containsString(statuses, "completed") {
			return "verified", "Task has completed worker evidence and the phase verification passed.", ""
		}
		if len(statuses) == 0 {
			if positiveEvidence {
				return "verified_partial", "Phase verification passed and overall code evidence supports this task even without a direct task dispatch record.", ""
			}
			return "missing", "No dispatch or reconciliation evidence was recorded for this task.", "redispatch"
		}
		return "verified_partial", fmt.Sprintf("Phase verification passed despite operational worker issues: %s.", strings.Join(statuses, ", ")), ""
	}

	if len(statuses) == 0 {
		return "missing", "No dispatch evidence was recorded for this task.", "redispatch"
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
		case "missing", "needs_redispatch", "implemented_unverified":
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
	if strings.TrimSpace(assessment.Recovery.ReconcileCommand) != "" {
		return assessment.Recovery.ReconcileCommand
	}
	if strings.TrimSpace(assessment.Recovery.ReverifyCommand) != "" {
		return assessment.Recovery.ReverifyCommand
	}
	return "aether continue"
}

func resolveCodexVerificationCommands(root string) codexVerificationCommands {
	commands := codexVerificationCommands{}
	switch {
	case fileExists(filepath.Join(root, "go.mod")):
		commands.Build = "go build ./..."
		commands.Type = "go vet ./..."
		commands.Test = "go test ./..."
	case fileExists(filepath.Join(root, "package.json")):
		commands.Build = "npm run build"
		commands.Type = "npx tsc --noEmit"
		commands.Lint = "npm run lint"
		commands.Test = "npm test"
	case fileExists(filepath.Join(root, "Cargo.toml")):
		commands.Build = "cargo build"
		commands.Lint = "cargo clippy"
		commands.Test = "cargo test"
	case fileExists(filepath.Join(root, "pyproject.toml")):
		commands.Build = "python -m build"
		commands.Type = "pyright ."
		commands.Lint = "ruff check ."
		commands.Test = "pytest"
	case fileExists(filepath.Join(root, "Makefile")):
		commands.Build = "make build"
		commands.Lint = "make lint"
		commands.Test = "make test"
	}
	return commands
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
		if manifestAllowsEmptyBuilderClaims(manifest) {
			return codexClaimVerification{
				Present: true,
				Passed:  true,
				Summary: "builder claims file is empty but the build ran in simulated mode",
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

func closeCodexContinueWorkers(manifest codexContinueManifest, assessment codexContinueAssessment) ([]codexContinueClosedWorker, error) {
	if !manifest.Present || len(manifest.Data.Dispatches) == 0 {
		return nil, nil
	}
	spawnTree := agent.NewSpawnTree(store, "spawn-tree.txt")
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
		if err := spawnTree.UpdateStatus(dispatch.Name, status, summary); err != nil {
			return closed, fmt.Errorf("failed to close worker %s: %w", dispatch.Name, err)
		}
		closed = append(closed, codexContinueClosedWorker{Name: dispatch.Name, Status: status, Summary: summary})
	}
	return closed, nil
}

func closedWorkerNames(details []codexContinueClosedWorker) []string {
	names := make([]string, 0, len(details))
	for _, detail := range details {
		names = append(names, detail.Name)
	}
	return names
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

func manifestAllowsEmptyBuilderClaims(manifest codexContinueManifest) bool {
	if !manifestRequiresBuilderClaims(manifest) {
		return true
	}
	return strings.EqualFold(strings.TrimSpace(manifest.Data.DispatchMode), "simulated") && allDispatchesCompleted(manifest)
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

func missingClaimsSummary(manifest codexContinueManifest) string {
	if manifestRequiresBuilderClaims(manifest) {
		return "no builder claims file found for a phase that dispatched builders"
	}
	return "no builder claims file found; skipped"
}
