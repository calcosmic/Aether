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
	Phase              int      `json:"phase"`
	GeneratedAt        string   `json:"generated_at"`
	Manifest           string   `json:"manifest,omitempty"`
	VerificationReport string   `json:"verification_report"`
	GateReport         string   `json:"gate_report"`
	ClosedWorkers      []string `json:"closed_workers,omitempty"`
	Advanced           bool     `json:"advanced"`
	Completed          bool     `json:"completed"`
	Next               string   `json:"next"`
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

func runCodexContinue(root string) (map[string]interface{}, colony.ColonyState, colony.Phase, *colony.Phase, *signalHousekeepingResult, bool, error) {
	if store == nil {
		return nil, colony.ColonyState{}, colony.Phase{}, nil, nil, false, fmt.Errorf("no store initialized")
	}

	var state colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
		return nil, state, colony.Phase{}, nil, nil, false, fmt.Errorf("No colony initialized. Run `aether init \"goal\"` first.")
	}
	if len(state.Plan.Phases) == 0 {
		return nil, state, colony.Phase{}, nil, nil, false, fmt.Errorf("No project plan. Run `aether plan` first.")
	}
	if state.CurrentPhase < 1 || state.CurrentPhase > len(state.Plan.Phases) {
		return nil, state, colony.Phase{}, nil, nil, false, fmt.Errorf("No active phase to continue. Run `aether build <phase>` first.")
	}

	currentIdx := state.CurrentPhase - 1
	phase := state.Plan.Phases[currentIdx]
	manifest := loadCodexContinueManifest(phase.ID)
	now := time.Now().UTC()

	verification := runCodexContinueVerification(root, phase.ID, manifest)
	gates := runCodexContinueGates(state, phase, manifest, verification, now)

	verificationReportRel := filepath.ToSlash(filepath.Join("build", fmt.Sprintf("phase-%d", phase.ID), "verification.json"))
	gateReportRel := filepath.ToSlash(filepath.Join("build", fmt.Sprintf("phase-%d", phase.ID), "gates.json"))
	if err := store.SaveJSON(verificationReportRel, verification); err != nil {
		return nil, state, phase, nil, nil, false, fmt.Errorf("failed to write verification report: %w", err)
	}
	if err := store.SaveJSON(gateReportRel, gates); err != nil {
		return nil, state, phase, nil, nil, false, fmt.Errorf("failed to write gate report: %w", err)
	}

	if !verification.Passed || !gates.Passed {
		blockers := append([]string{}, verification.BlockingIssues...)
		blockers = append(blockers, gates.BlockingIssues...)
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
			Advanced:           false,
			Completed:          false,
			Next:               "aether continue",
		})
		updateSessionSummary("continue", "aether continue", summary)

		result := map[string]interface{}{
			"advanced":            false,
			"blocked":             true,
			"current_phase":       state.CurrentPhase,
			"phase_name":          phase.Name,
			"state":               state.State,
			"next":                "aether continue",
			"verification":        verification,
			"gates":               gates,
			"verification_report": displayDataPath(verificationReportRel),
			"gate_report":         displayDataPath(gateReportRel),
			"continue_report":     displayDataPath(continueReportRel),
			"blocking_issues":     blockers,
		}
		return result, state, phase, nil, nil, false, nil
	}

	closedWorkers, err := closeCodexContinueWorkers(manifest)
	if err != nil {
		return nil, state, phase, nil, nil, false, err
	}
	_ = updateCodexContinueContext(phase, manifest, closedWorkers, now)

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
		nextPhase = &updated.Plan.Phases[nextIdx]
		updated.State = colony.StateREADY
		nextCommand = fmt.Sprintf("aether build %d", nextIdx+1)
		updated.Events = append(trimmedEvents(updated.Events),
			fmt.Sprintf("%s|verification_passed|continue|Build verification passed for phase %d", now.Format(time.RFC3339), phase.ID),
			fmt.Sprintf("%s|gate_passed|continue|Continue gates passed for phase %d", now.Format(time.RFC3339), phase.ID),
			fmt.Sprintf("%s|phase_advanced|continue|Completed phase %d, ready for phase %d", now.Format(time.RFC3339), updated.CurrentPhase, nextIdx+1),
		)
	}

	if err := store.SaveJSON("COLONY_STATE.json", updated); err != nil {
		return nil, state, phase, nextPhase, nil, final, fmt.Errorf("failed to save colony state: %w", err)
	}
	housekeeping, housekeepingErr := runSignalHousekeeping(now, false)
	if housekeepingErr != nil {
		return nil, updated, phase, nextPhase, nil, final, housekeepingErr
	}

	continueReportRel := filepath.ToSlash(filepath.Join("build", fmt.Sprintf("phase-%d", phase.ID), "continue.json"))
	if err := store.SaveJSON(continueReportRel, codexContinueReport{
		Phase:              phase.ID,
		GeneratedAt:        now.Format(time.RFC3339),
		Manifest:           displayOptionalDataPath(manifest.Path),
		VerificationReport: displayDataPath(verificationReportRel),
		GateReport:         displayDataPath(gateReportRel),
		ClosedWorkers:      closedWorkers,
		Advanced:           true,
		Completed:          final,
		Next:               nextCommand,
	}); err != nil {
		return nil, updated, phase, nextPhase, &housekeeping, final, fmt.Errorf("failed to write continue report: %w", err)
	}

	updateSessionSummary("continue", nextCommand, fmt.Sprintf("Phase %d verified and advanced", phase.ID))
	result := map[string]interface{}{
		"advanced":            true,
		"completed":           final,
		"current_phase":       updated.CurrentPhase,
		"state":               updated.State,
		"next":                nextCommand,
		"verification":        verification,
		"gates":               gates,
		"verification_report": displayDataPath(verificationReportRel),
		"gate_report":         displayDataPath(gateReportRel),
		"continue_report":     displayDataPath(continueReportRel),
		"closed_workers":      closedWorkers,
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

	passed := true
	blockers := []string{}
	for _, step := range steps {
		if !step.Passed && !step.Skipped {
			passed = false
			blockers = append(blockers, fmt.Sprintf("%s failed: %s", step.Name, step.Summary))
		}
	}
	if !claims.Passed && !claims.Skipped {
		passed = false
		blockers = append(blockers, claims.Summary)
	}

	return codexContinueVerificationReport{
		Phase:          phaseID,
		GeneratedAt:    now.Format(time.RFC3339),
		Steps:          steps,
		Claims:         claims,
		Passed:         passed,
		BlockingIssues: blockers,
	}
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
			Passed:  true,
			Skipped: true,
			Summary: "no builder claims file found; skipped",
		}
	}

	checked := 0
	mismatches := []string{}
	for _, rel := range append(append([]string{}, claims.FilesCreated...), claims.FilesModified...) {
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

func runCodexContinueGates(state colony.ColonyState, phase colony.Phase, manifest codexContinueManifest, verification codexContinueVerificationReport, now time.Time) codexContinueGateReport {
	checks := []gateCheck{}
	blockers := []string{}

	spawnCount := 0
	watcherCount := 0
	if manifest.Present {
		for _, dispatch := range manifest.Data.Dispatches {
			spawnCount++
			if dispatch.Caste == "watcher" {
				watcherCount++
			}
		}
	}

	spawnCheck := gateCheck{Name: "spawn_gate", Passed: true, Detail: fmt.Sprintf("%d workers dispatched", spawnCount)}
	if manifest.Present && len(phase.Tasks) >= 3 && spawnCount == 0 {
		spawnCheck.Passed = false
		spawnCheck.Detail = fmt.Sprintf("phase had %d tasks but manifest recorded 0 worker dispatches", len(phase.Tasks))
		blockers = append(blockers, spawnCheck.Detail)
	}
	checks = append(checks, spawnCheck)

	watcherCheck := gateCheck{Name: "watcher_gate", Passed: true, Detail: fmt.Sprintf("%d watcher dispatches recorded", watcherCount)}
	if manifest.Present && watcherCount == 0 {
		watcherCheck.Passed = false
		watcherCheck.Detail = "no watcher dispatch recorded for this build"
		blockers = append(blockers, watcherCheck.Detail)
	}
	checks = append(checks, watcherCheck)

	flagCheck := checkNoCriticalFlags()
	checks = append(checks, flagCheck)
	if !flagCheck.Passed {
		blockers = append(blockers, flagCheck.Detail)
	}

	verifyCheck := gateCheck{Name: "verification_passed", Passed: verification.Passed, Detail: "all verification checks passed"}
	if !verification.Passed {
		verifyCheck.Detail = strings.Join(verification.BlockingIssues, "; ")
		blockers = append(blockers, verification.BlockingIssues...)
	}
	checks = append(checks, verifyCheck)

	return codexContinueGateReport{
		Phase:          phase.ID,
		GeneratedAt:    now.Format(time.RFC3339),
		Checks:         checks,
		Passed:         len(blockers) == 0,
		BlockingIssues: uniqueSortedStrings(blockers),
	}
}

func closeCodexContinueWorkers(manifest codexContinueManifest) ([]string, error) {
	if !manifest.Present || len(manifest.Data.Dispatches) == 0 {
		return nil, nil
	}
	spawnTree := agent.NewSpawnTree(store, "spawn-tree.txt")
	closed := make([]string, 0, len(manifest.Data.Dispatches))
	for _, dispatch := range manifest.Data.Dispatches {
		summary := "Closed by continue after verification"
		if dispatch.Caste == "watcher" {
			summary = "Verification passed during continue"
		}
		if err := spawnTree.UpdateStatus(dispatch.Name, "completed", summary); err != nil {
			return closed, fmt.Errorf("failed to close worker %s: %w", dispatch.Name, err)
		}
		closed = append(closed, dispatch.Name)
	}
	return closed, nil
}

func updateCodexContinueContext(phase colony.Phase, manifest codexContinueManifest, closedWorkers []string, now time.Time) error {
	data, err := store.ReadFile(contextFileName)
	if err != nil {
		return nil
	}
	content := string(data)
	content = replaceContextTableRow(content, "Last Updated", now.Format(time.RFC3339))
	content = replaceContextTableRow(content, "Safe to Clear?", "YES — Build complete, ready to continue")
	content = replaceBuildInProgressWithComplete(content, "verified", fmt.Sprintf("Phase %d ready to advance", phase.ID))
	for _, name := range closedWorkers {
		content = markWorkerComplete(content, name, "completed", now.Format(time.RFC3339))
	}
	return store.AtomicWrite(contextFileName, []byte(content))
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
