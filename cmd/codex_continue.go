package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/calcosmic/Aether/pkg/agent"
	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
)

// VerificationErrorClass classifies why a verification step failed.
type VerificationErrorClass string

const (
	ErrorClassProduct     VerificationErrorClass = "product"     // real code/test failure
	ErrorClassEnvironment VerificationErrorClass = "environment" // EPERM, EADDRINUSE, missing DB, etc.
	ErrorClassTimeout     VerificationErrorClass = "timeout"
	ErrorClassSkipped     VerificationErrorClass = "skipped"
)

const watcherStatusEnvironmentBlocked = "environment_blocked"

type codexVerificationStep struct {
	Name           string                 `json:"name"`
	Command        string                 `json:"command,omitempty"`
	Passed         bool                   `json:"passed"`
	Skipped        bool                   `json:"skipped,omitempty"`
	Required       bool                   `json:"required,omitempty"`
	Blocked        bool                   `json:"blocked,omitempty"`
	TimedOut       bool                   `json:"timed_out,omitempty"`
	TimeoutSeconds int                    `json:"timeout_seconds,omitempty"`
	ExitCode       int                    `json:"exit_code,omitempty"`
	ErrorClass     VerificationErrorClass `json:"error_class,omitempty"`
	Summary        string                 `json:"summary"`
	Output         string                 `json:"output,omitempty"`
}

type codexClaimVerification struct {
	Present    bool     `json:"present"`
	Passed     bool     `json:"passed"`
	Skipped    bool     `json:"skipped,omitempty"`
	Summary    string   `json:"summary"`
	Checked    int      `json:"checked"`
	Mismatches []string `json:"mismatches,omitempty"`
	// ScannedFiles is the union of FilesCreated + FilesModified + TestsWritten
	// from the loaded build claims (blanks trimmed and skipped) — the single
	// notion of "what changed this phase" that checkAntiPatternGate scans.
	// Populated on every return path where claims were successfully loaded.
	ScannedFiles []string `json:"scanned_files,omitempty"`
}

type codexContinueVerificationReport struct {
	Phase                      int                          `json:"phase"`
	GeneratedAt                string                       `json:"generated_at"`
	VerificationTimeoutSeconds int                          `json:"verification_timeout_seconds,omitempty"`
	Steps                      []codexVerificationStep      `json:"steps"`
	Claims                     codexClaimVerification       `json:"claims"`
	Watcher                    codexWatcherVerification     `json:"watcher"`
	CriteriaPolicy             string                       `json:"criteria_policy"`
	CriteriaEnforced           bool                         `json:"criteria_enforced"`
	CriteriaPassed             bool                         `json:"criteria_passed"`
	Criteria                   []codexCriterionVerification `json:"criteria,omitempty"`
	ChecksPassed               bool                         `json:"checks_passed"`
	Passed                     bool                         `json:"passed"`
	BlockingIssues             []string                     `json:"blocking_issues,omitempty"`
	// Warnings surface non-blocking verification observations — most
	// importantly "no tests to run in this project" (D-01, Phase 193), which
	// used to be a silent hard-block; it is now a visible warning that the
	// floor is claimed files plus criterion evidence, with no fallback that
	// hands verification responsibility to a reviewer.
	Warnings []string `json:"warnings,omitempty"`
	// CheckFixAttempt is set when a failing check with no reviewer dispatched
	// drew D-02/D-03's single bounded automatic builder fix attempt. nil
	// means no attempt ran (the floor passed, a reviewer was dispatched, or
	// nothing was eligible) -- see applyAutomaticCheckFixAttempt
	// (cmd/check_fix_attempt.go).
	CheckFixAttempt *checkFixAttemptRecord `json:"check_fix_attempt,omitempty"`
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
	// Warnings carry non-blocking observations (for example recorded
	// operational worker issues). They replaced the operational_evidence
	// "gate", which hardcoded Passed=true and therefore asserted nothing.
	Warnings []string `json:"warnings,omitempty"`
}

type codexContinueReport struct {
	Phase               int                           `json:"phase"`
	GeneratedAt         string                        `json:"generated_at"`
	Manifest            string                        `json:"manifest,omitempty"`
	VerificationReport  string                        `json:"verification_report"`
	GateReport          string                        `json:"gate_report"`
	ReviewReport        string                        `json:"review_report,omitempty"`
	Summary             string                        `json:"summary,omitempty"`
	ClosedWorkers       []string                      `json:"closed_workers,omitempty"`
	WorkerFlow          []codexContinueWorkerFlowStep `json:"worker_flow,omitempty"`
	PartialSuccess      bool                          `json:"partial_success,omitempty"`
	OperationalIssues   []string                      `json:"operational_issues,omitempty"`
	Tasks               []codexContinueTaskAssessment `json:"tasks,omitempty"`
	Recovery            codexContinueRecoveryPlan     `json:"recovery,omitempty"`
	Advanced            bool                          `json:"advanced"`
	Completed           bool                          `json:"completed"`
	Next                string                        `json:"next"`
	LastContinueOptions *codexContinueOptionsJSON     `json:"last_continue_options,omitempty"`
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
	// ReadOnlyArtifacts is an escape hatch for a task named in
	// ReconcileTaskIDs: each entry is a "<task-id>:<path>" spec recording
	// hash-verified read-only evidence for an artifact that task legitimately
	// did not modify (D-01). It never widens claim satisfaction beyond the
	// named task (D-02).
	ReadOnlyArtifacts   []string
	WorkerTimeout       time.Duration
	VerificationTimeout time.Duration
	ParentContext       context.Context
	LightFlag           bool
	HeavyFlag           bool
	SkipWatchers        bool
	VerificationDepth   string
	// QueenCastes is the review team the Queen chose after reading the phase.
	// Continue is the expensive flow — each reviewer is a full agent run — and
	// until this existed the team came only from keyword scoring, so a phase
	// whose vocabulary happened to include "latency" or "memory" bought a
	// Measurer whether or not anything about the change was a performance
	// question. Empty means no judgement was offered and scoring decides.
	QueenCastes      []string
	QueenCasteReason string
	// QueenCasteWhy is one reason per proposed reviewer caste, as
	// "caste=reason" (D-08, D-09). A caste named in QueenCastes with no
	// matching entry here, and not required by the phase, is refused by name
	// rather than sent unexplained.
	QueenCasteWhy []string
}

// codexContinueOptionsJSON is a serializable snapshot of continue options,
// stored in the continue report so the next invocation can detect parameter loops.
type codexContinueOptionsJSON struct {
	VerificationTimeoutSec int      `json:"verification_timeout_sec,omitempty"`
	WorkerTimeoutSec       int      `json:"worker_timeout_sec,omitempty"`
	ReconcileTaskIDs       []string `json:"reconcile_task_ids,omitempty"`
	ReadOnlyArtifacts      []string `json:"read_only_artifacts,omitempty"`
	SkipWatchers           bool     `json:"skip_watchers,omitempty"`
	LightFlag              bool     `json:"light_flag,omitempty"`
	HeavyFlag              bool     `json:"heavy_flag,omitempty"`
	VerificationDepth      string   `json:"verification_depth,omitempty"`
	QueenCastes            []string `json:"queen_castes,omitempty"`
	QueenCasteReason       string   `json:"queen_caste_reason,omitempty"`
	QueenCasteWhy          []string `json:"queen_caste_why,omitempty"`
}

const abandonedBuildThreshold = 10 * time.Minute
const defaultWatcherFailureThreshold = 3

// getWatcherFailureCount returns the watcher failure count for a given phase.
func getWatcherFailureCount(state colony.ColonyState, phaseID int) int {
	for _, p := range state.Plan.Phases {
		if p.ID == phaseID {
			return p.WatcherFailureCount
		}
	}
	return 0
}

// incrementWatcherFailureCount atomically increments the watcher failure count for a phase.
func incrementWatcherFailureCount(phaseID int) error {
	var updated colony.ColonyState
	return store.UpdateJSONAtomically("COLONY_STATE.json", &updated, func() error {
		for i := range updated.Plan.Phases {
			if updated.Plan.Phases[i].ID == phaseID {
				updated.Plan.Phases[i].WatcherFailureCount++
				return nil
			}
		}
		return fmt.Errorf("phase %d not found in colony state", phaseID)
	})
}

// resetWatcherFailureCount atomically resets the watcher failure count for a phase to 0.
func resetWatcherFailureCount(phaseID int) error {
	var updated colony.ColonyState
	return store.UpdateJSONAtomically("COLONY_STATE.json", &updated, func() error {
		for i := range updated.Plan.Phases {
			if updated.Plan.Phases[i].ID == phaseID {
				updated.Plan.Phases[i].WatcherFailureCount = 0
				return nil
			}
		}
		return fmt.Errorf("phase %d not found in colony state", phaseID)
	})
}

// continueOptionsToJSON converts codexContinueOptions to a serializable snapshot.
func continueOptionsToJSON(opts codexContinueOptions) *codexContinueOptionsJSON {
	return &codexContinueOptionsJSON{
		VerificationTimeoutSec: int(opts.VerificationTimeout / time.Second),
		WorkerTimeoutSec:       int(opts.WorkerTimeout / time.Second),
		ReconcileTaskIDs:       opts.ReconcileTaskIDs,
		ReadOnlyArtifacts:      opts.ReadOnlyArtifacts,
		SkipWatchers:           opts.SkipWatchers,
		LightFlag:              opts.LightFlag,
		HeavyFlag:              opts.HeavyFlag,
		VerificationDepth:      opts.VerificationDepth,
		QueenCastes:            append([]string(nil), opts.QueenCastes...),
		QueenCasteReason:       opts.QueenCasteReason,
		QueenCasteWhy:          append([]string(nil), opts.QueenCasteWhy...),
	}
}

func normalizedContinueOptionList(values []string, splitCommas bool) []string {
	normalized := make([]string, 0, len(values))
	for _, value := range values {
		parts := []string{value}
		if splitCommas {
			parts = strings.Split(value, ",")
		}
		for _, part := range parts {
			if part = strings.ToLower(strings.TrimSpace(part)); part != "" {
				normalized = append(normalized, part)
			}
		}
	}
	return uniqueSortedStrings(normalized)
}

func normalizedContinueOptionsEqual(left, right []string, splitCommas bool) bool {
	left = normalizedContinueOptionList(left, splitCommas)
	right = normalizedContinueOptionList(right, splitCommas)
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

// continueOptionsMatchCurrent checks whether the current continue options match
// the last invocation's options, indicating a potential parameter loop.
func continueOptionsMatchCurrent(current codexContinueOptions, last *codexContinueOptionsJSON) bool {
	if last == nil {
		return false
	}
	currentSec := int(current.VerificationTimeout / time.Second)
	if currentSec != last.VerificationTimeoutSec {
		return false
	}
	workerSec := int(current.WorkerTimeout / time.Second)
	if workerSec != last.WorkerTimeoutSec {
		return false
	}
	if current.SkipWatchers != last.SkipWatchers {
		return false
	}
	if current.LightFlag != last.LightFlag {
		return false
	}
	if current.HeavyFlag != last.HeavyFlag {
		return false
	}
	if current.VerificationDepth != last.VerificationDepth {
		return false
	}
	if !normalizedContinueOptionsEqual(current.QueenCastes, last.QueenCastes, true) {
		return false
	}
	if strings.TrimSpace(current.QueenCasteReason) != strings.TrimSpace(last.QueenCasteReason) {
		return false
	}
	if !normalizedContinueOptionsEqual(current.QueenCasteWhy, last.QueenCasteWhy, false) {
		return false
	}
	// Compare reconcile task IDs (order-independent).
	if len(current.ReconcileTaskIDs) != len(last.ReconcileTaskIDs) {
		return false
	}
	currentSorted := uniqueSortedStrings(current.ReconcileTaskIDs)
	lastSorted := uniqueSortedStrings(last.ReconcileTaskIDs)
	for i := range currentSorted {
		if currentSorted[i] != lastSorted[i] {
			return false
		}
	}
	// Compare read-only artifact specs (order-independent). A changed set
	// invalidates a stale plan-only manifest just like reconcile task IDs.
	if len(current.ReadOnlyArtifacts) != len(last.ReadOnlyArtifacts) {
		return false
	}
	currentReadOnlySorted := uniqueSortedStrings(current.ReadOnlyArtifacts)
	lastReadOnlySorted := uniqueSortedStrings(last.ReadOnlyArtifacts)
	for i := range currentReadOnlySorted {
		if currentReadOnlySorted[i] != lastReadOnlySorted[i] {
			return false
		}
	}
	return true
}

// loadLastContinueOptions reads the last continue options from the saved continue report.
func loadLastContinueOptions(phaseID int) *codexContinueOptionsJSON {
	if store == nil {
		return nil
	}
	rel := filepath.ToSlash(filepath.Join("build", fmt.Sprintf("phase-%d", phaseID), "continue.json"))
	var report codexContinueReport
	if err := store.LoadJSON(rel, &report); err != nil {
		return nil
	}
	return report.LastContinueOptions
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
	if elapsed < abandonedBuildThreshold {
		return false, 0, ""
	}
	return true, elapsed, fmt.Sprintf("Build was abandoned %.0f minutes ago: all %d dispatches stuck at 'spawned'", elapsed.Minutes(), len(manifest.Data.Dispatches))
}

// abandonedBuildTaskIDs extracts every covered task ID from manifest
// dispatches. A grouped worker owns more than its primary compatibility ID,
// and recovery must never leave the later covered tasks invisible.
func abandonedBuildTaskIDs(manifest codexContinueManifest) []string {
	seen := make(map[string]struct{})
	var ids []string
	for _, d := range manifest.Data.Dispatches {
		for _, id := range dispatchCoveredTaskIDs(d) {
			if _, ok := seen[id]; !ok {
				seen[id] = struct{}{}
				ids = append(ids, id)
			}
		}
	}
	return ids
}

func missingBuildPacketBlockedResult(state colony.ColonyState, phase colony.Phase, options codexContinueOptions) map[string]interface{} {
	reviewDepth := resolveEffectiveContinueDepth(phase, len(state.Plan.Phases), options.LightFlag, options.HeavyFlag, options.VerificationDepth, state.VerificationDepth)
	now := time.Now().UTC()
	runHandle, _ := beginRuntimeSpawnRun("continue", now)
	recovery := codexContinueRecoveryPlan{
		RedispatchCommand: buildForceRedispatchCommand(phase.ID),
		SkipCommand:       buildSkipPhaseCommand(phase.ID),
	}
	summary := fmt.Sprintf("No active build packet was found for phase %d. The previous build may have been interrupted before worker results were recorded.", phase.ID)
	if state.BuildStartedAt != nil {
		elapsed := now.Sub(state.BuildStartedAt.UTC())
		if elapsed > 0 {
			summary = fmt.Sprintf("%s Build has been active for %s.", summary, formatDurationForCLI(elapsed.Round(time.Second)))
		}
	}
	continueReportRel := filepath.ToSlash(filepath.Join("build", fmt.Sprintf("phase-%d", phase.ID), "continue.json"))
	_ = store.SaveJSON(continueReportRel, codexContinueReport{
		Phase:               phase.ID,
		GeneratedAt:         now.Format(time.RFC3339),
		Summary:             summary,
		Recovery:            recovery,
		Advanced:            false,
		Completed:           false,
		Next:                recovery.RedispatchCommand,
		LastContinueOptions: continueOptionsToJSON(options),
	})
	updateSessionSummary("continue", recovery.RedispatchCommand, summary)
	finishRuntimeSpawnRun(runHandle, "blocked-missing-build-packet", now)
	result := map[string]interface{}{
		"advanced":        false,
		"blocked":         true,
		"missing_packet":  true,
		"current_phase":   state.CurrentPhase,
		"phase_name":      phase.Name,
		"state":           state.State,
		"recovery":        recovery,
		"next":            recovery.RedispatchCommand,
		"continue_report": displayDataPath(continueReportRel),
		"review_depth":    string(reviewDepth),
		"blocking_issues": []string{summary},
	}
	if options.WorkerTimeout > 0 {
		result["worker_timeout_sec"] = int(options.WorkerTimeout / time.Second)
	}
	return result
}

func manifestTaskSetBlockedResult(state colony.ColonyState, phase colony.Phase, manifest codexContinueManifest, options codexContinueOptions) (map[string]interface{}, bool) {
	err := validateBuildManifestTaskSetForPhase(manifest, phase, true)
	if err == nil {
		return nil, false
	}

	reviewDepth := resolveEffectiveContinueDepth(phase, len(state.Plan.Phases), options.LightFlag, options.HeavyFlag, options.VerificationDepth, state.VerificationDepth)
	now := time.Now().UTC()
	runHandle, _ := beginRuntimeSpawnRun("continue", now)
	nextCommand := buildForceRedispatchCommand(phase.ID)
	summary := fmt.Sprintf("%s; run `%s` to regenerate the build manifest before `aether continue`", err.Error(), nextCommand)
	recovery := codexContinueRecoveryPlan{
		RedispatchCommand: nextCommand,
		SkipCommand:       buildSkipPhaseCommand(phase.ID),
	}
	continueReportRel := filepath.ToSlash(filepath.Join("build", fmt.Sprintf("phase-%d", phase.ID), "continue.json"))
	_ = store.SaveJSON(continueReportRel, codexContinueReport{
		Phase:               phase.ID,
		GeneratedAt:         now.Format(time.RFC3339),
		Manifest:            displayOptionalDataPath(manifest.Path),
		Summary:             summary,
		Recovery:            recovery,
		Advanced:            false,
		Completed:           false,
		Next:                nextCommand,
		LastContinueOptions: continueOptionsToJSON(options),
	})
	updateSessionSummary("continue", nextCommand, summary)
	finishRuntimeSpawnRun(runHandle, "blocked-manifest-task-set", now)
	return map[string]interface{}{
		"advanced":        false,
		"blocked":         true,
		"current_phase":   state.CurrentPhase,
		"phase_name":      phase.Name,
		"state":           state.State,
		"recovery":        recovery,
		"next":            nextCommand,
		"continue_report": displayDataPath(continueReportRel),
		"review_depth":    string(reviewDepth),
		"blocking_issues": []string{summary},
	}, true
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
	SkipCommand       string   `json:"skip_command,omitempty"`
	// CheckFixCommand is set only when D-02's automatic fix attempt ran and
	// the re-run still failed: the single exact command to re-run the
	// builder by hand (D-02, FLOOR-02) -- never a menu.
	CheckFixCommand string `json:"check_fix_command,omitempty"`
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
	Stage           string               `json:"stage,omitempty"`
	Caste           string               `json:"caste,omitempty"`
	Name            string               `json:"name"`
	Task            string               `json:"task,omitempty"`
	Status          string               `json:"status"`
	Summary         string               `json:"summary,omitempty"`
	Blockers        []string             `json:"blockers,omitempty"`
	Duration        float64              `json:"duration,omitempty"`
	Report          string               `json:"report,omitempty"`
	Findings        []codexReviewFinding `json:"findings,omitempty"`
	Recommendations []string             `json:"recommendations,omitempty"`
	WeakSpots       []string             `json:"weak_spots,omitempty"`
	EdgeCases       []string             `json:"edge_cases_discovered,omitempty"`
	ReusableLessons []string             `json:"reusable_lessons,omitempty"`
}

type codexReviewFinding struct {
	Domain      string `json:"domain,omitempty"`
	Severity    string `json:"severity,omitempty"`
	File        string `json:"file,omitempty"`
	Line        int    `json:"line,omitempty"`
	Category    string `json:"category,omitempty"`
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	Suggestion  string `json:"suggestion,omitempty"`
	Blocking    bool   `json:"blocking,omitempty"`
}

type codexContinueReviewReport struct {
	Phase          int                           `json:"phase"`
	GeneratedAt    string                        `json:"generated_at"`
	Workers        []codexContinueWorkerFlowStep `json:"workers"`
	Passed         bool                          `json:"passed"`
	BlockingIssues []string                      `json:"blocking_issues,omitempty"`
}

var continueContextUpdater = updateCodexContinueContext

// circuitBreaker is the package-level circuit breaker for gate retry tracking.
// It is initialized when the continue flow starts and checked for nil before use.
var circuitBreaker *CircuitBreaker

var continueSignalHousekeeper = func(now time.Time, state colony.ColonyState) (signalHousekeepingResult, error) {
	return runSignalHousekeepingWithState(now, false, &state)
}

func runCodexContinue(root string, options codexContinueOptions) (map[string]interface{}, colony.ColonyState, colony.Phase, *colony.Phase, *signalHousekeepingResult, bool, error) {
	if store == nil {
		return nil, colony.ColonyState{}, colony.Phase{}, nil, nil, false, fmt.Errorf("no store initialized")
	}
	parentCtx := options.ParentContext
	if parentCtx == nil {
		parentCtx = context.Background()
	}
	ctx, stopSignals := signal.NotifyContext(parentCtx, os.Interrupt, syscall.SIGTERM)
	defer stopSignals()

	state, err := loadActiveColonyState()
	if err != nil {
		return nil, state, colony.Phase{}, nil, nil, false, fmt.Errorf("%s", colonyStateLoadMessage(err))
	}

	// Preserve-and-report pass over tracked worktrees. Synchronous — the
	// error is captured and surfaced rather than discarded, since a
	// discarded error here is how a preservation failure would go unnoticed.
	gcCleaned, gcPreserved, gcErr := gcOrphanedWorktrees()
	if gcErr != nil {
		emitVisualProgress(fmt.Sprintf("Could not check worktrees for leftover work: %v", gcErr))
	} else if gcCleaned > 0 || gcPreserved > 0 {
		emitVisualProgress(fmt.Sprintf("Worktrees: %d stale entry(s) forgotten (path already gone), %d kept because they still hold work", gcCleaned, gcPreserved))
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
	reviewDepth := resolveEffectiveContinueDepth(phase, len(state.Plan.Phases), options.LightFlag, options.HeavyFlag, options.VerificationDepth, state.VerificationDepth)
	if phase.Status != colony.PhaseInProgress {
		return nil, state, colony.Phase{}, nil, nil, false, fmt.Errorf("phase %d is not in progress; run `aether build %d` first", phase.ID, phase.ID)
	}
	if err := validateContinueReconcileTasks(phase, options.ReconcileTaskIDs); err != nil {
		return nil, state, colony.Phase{}, nil, nil, false, err
	}
	if err := validateReadOnlyArtifacts(phase, options.ReconcileTaskIDs, options.ReadOnlyArtifacts); err != nil {
		return nil, state, colony.Phase{}, nil, nil, false, err
	}
	manifest := loadCodexContinueManifest(phase.ID)
	if !manifest.Present {
		return missingBuildPacketBlockedResult(state, phase, options), state, phase, nil, nil, false, nil
	}
	if result, blocked := manifestTaskSetBlockedResult(state, phase, manifest, options); blocked {
		return result, state, phase, nil, nil, false, nil
	}
	if changed, reconcileErr := reconcileContinueCompletedBuildTasks(&state, &phase, &manifest); reconcileErr != nil {
		if errors.Is(reconcileErr, errRuntimeStateSuperseded) {
			return continueSupersededResult(state, phase, reconcileErr), state, phase, nil, nil, false, nil
		}
		return nil, state, colony.Phase{}, nil, nil, false, reconcileErr
	} else if changed {
		state.Plan.Phases[currentIdx] = phase
	}

	// D-01 escape hatch: record hash-verified read-only evidence for the
	// reconcile task IDs' declared artifacts BEFORE verification (and its
	// embedded criterion evidence evaluation) runs, so evaluatePhaseCriterionEvidence
	// sees the recorded evidence on this same invocation. Validation above
	// already confirmed every spec's task ID both exists in the phase and was
	// also passed to --reconcile-task.
	if len(options.ReadOnlyArtifacts) > 0 {
		if err := applyReadOnlyArtifactEvidence(root, manifest, options.ReadOnlyArtifacts); err != nil {
			return nil, state, colony.Phase{}, nil, nil, false, err
		}
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
			SkipCommand:       buildSkipPhaseCommand(phase.ID),
		}
		nextCommand := recovery.RedispatchCommand
		if strings.TrimSpace(nextCommand) == "" {
			nextCommand = buildForceRedispatchCommand(phase.ID)
		}
		_ = store.SaveJSON(
			filepath.ToSlash(filepath.Join("build", fmt.Sprintf("phase-%d", phase.ID), "continue.json")),
			codexContinueReport{
				Phase:       phase.ID,
				GeneratedAt: now.Format(time.RFC3339),
				Summary:     abandonedSummary,
				Recovery:    recovery,
				Advanced:    false,
				Completed:   false,
				Next:        nextCommand,
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
			"next":            nextCommand,
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

	// FIELD-04 (191.1-CONTEXT.md D-07/D-08): a completed, passing
	// verification from an earlier continue run may have lost the race to a
	// colony pause and been preserved instead of discarded (see
	// cmd/advance_phase.go). Check for it here, before any of the expensive
	// verification/watcher-dispatch work below, so a resumed colony applies
	// that already-verified result instead of re-running it.
	if outcome := replayPendingContinueAdvance(state, phase, "continue", now); outcome.Handled {
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

	// Ceremony progress tracking (visual mode only)
	var progress *ceremonyProgress
	if shouldRenderVisualOutput(stdout) {
		continueSteps := []string{"Verification", "Housekeeping", "Advance", "Complete"}
		progress = NewCeremonyProgress(continueSteps, stdout)
	}

	emitContinueVerificationStart(phase, options.VerificationTimeout)
	verification, watcherFlow := runCodexContinueVerification(ctx, root, state, phase, manifest, options.WorkerTimeout, options.VerificationTimeout, options.SkipWatchers)
	assessment := assessCodexContinue(phase, manifest, verification, options, now)
	verification = attachContinueClaimVerification(verification, assessment)
	priorGateResults, _ := gateResultsReadPhase(phase.ID)
	if priorGateResults == nil {
		priorGateResults = []GateCheckResult{}
	}
	// Evidence-based flag clearing BEFORE the gates evaluate: a failed
	// verification raised machine-source blocker flags; this green run is
	// the evidence that clears them, and the restored Iron Law flags gate
	// would otherwise deadlock on its own stale flags. Chaos-raised and
	// user-raised blockers never auto-clear.
	autoResolveVerificationBlockers(verification.ChecksPassed, phase.ID)
	gates := runCodexContinueGates(phase, manifest, verification, assessment, now, priorGateResults)
	if progress != nil {
		progress.Advance("Verification")
	}

	// Persist gate results after each gate run
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
		workerFlow := continueWorkerFlowForVerification(verification, nil, watcherFlow)
		emitContinueCeremonyFlowSequence("aether-continue", phase, workerFlow)
		nextCommand := continueNextCommandForBlocked(assessment, blockers, options, phase.ID)
		if strings.Contains(nextCommand, "build --force") || strings.Contains(nextCommand, "force-redispatch") {
			emitLoopBreakEvent("recovery_redirect",
				fmt.Sprintf("recovery command differs from last invocation for phase %d", phase.ID),
				fmt.Sprintf("redirected to %s to break potential loop", nextCommand),
				"aether-continue")
		}
		_ = store.SaveJSON(continueReportRel, codexContinueReport{
			Phase:               phase.ID,
			GeneratedAt:         now.Format(time.RFC3339),
			Manifest:            displayOptionalDataPath(manifest.Path),
			VerificationReport:  displayDataPath(verificationReportRel),
			GateReport:          displayDataPath(gateReportRel),
			Summary:             summary,
			WorkerFlow:          workerFlow,
			PartialSuccess:      assessment.PartialSuccess,
			OperationalIssues:   append([]string{}, assessment.OperationalIssues...),
			Tasks:               append([]codexContinueTaskAssessment{}, assessment.Tasks...),
			Recovery:            assessment.Recovery,
			Advanced:            false,
			Completed:           false,
			Next:                nextCommand,
			LastContinueOptions: continueOptionsToJSON(options),
		})
		blockedState, flowErr := recordBlockedContinueWorkerFlow(state, now, workerFlow)
		if flowErr != nil {
			if errors.Is(flowErr, errRuntimeStateSuperseded) {
				runStatus = "superseded"
				return continueSupersededResult(state, phase, flowErr), state, phase, nil, nil, false, nil
			}
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
			"review_depth":        string(reviewDepth),
			"plan_revision_option": planRevisionRecommendation(
				colony.PlanRevisionVerificationFailure,
				fmt.Sprintf("Verification blocked phase %d: %s", phase.ID, summary),
				displayDataPath(verificationReportRel),
			),
		}
		runStatus = "blocked"
		return result, blockedState, phase, nil, nil, false, nil
	}

	mergedContinueQueenCastes, continueQueenCasteWhyReasons := parseAndMergeCasteWhy(options.QueenCastes, options.QueenCasteWhy)
	review := runCodexContinueReview(root, phase, manifest, verification, assessment, options.WorkerTimeout, reviewDepth, options.SkipWatchers, mergedContinueQueenCastes, options.QueenCasteReason, continueQueenCasteWhyReasons)
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
		workerFlow := continueWorkerFlowForVerification(verification, review.Workers, watcherFlow)
		emitContinueCeremonyFlowSequence("aether-continue", phase, workerFlow)
		nextCommand := continueNextCommandForBlocked(assessment, review.BlockingIssues, options, phase.ID)
		if strings.Contains(nextCommand, "build --force") || strings.Contains(nextCommand, "force-redispatch") {
			emitLoopBreakEvent("recovery_redirect",
				fmt.Sprintf("recovery command differs from last invocation for phase %d", phase.ID),
				fmt.Sprintf("redirected to %s to break potential loop", nextCommand),
				"aether-continue")
		}
		_ = store.SaveJSON(continueReportRel, codexContinueReport{
			Phase:               phase.ID,
			GeneratedAt:         now.Format(time.RFC3339),
			Manifest:            displayOptionalDataPath(manifest.Path),
			VerificationReport:  displayDataPath(verificationReportRel),
			GateReport:          displayDataPath(gateReportRel),
			ReviewReport:        displayDataPath(reviewReportRel),
			Summary:             summary,
			WorkerFlow:          workerFlow,
			PartialSuccess:      assessment.PartialSuccess,
			OperationalIssues:   append(append([]string{}, assessment.OperationalIssues...), review.BlockingIssues...),
			Tasks:               append([]codexContinueTaskAssessment{}, assessment.Tasks...),
			Recovery:            assessment.Recovery,
			Advanced:            false,
			Completed:           false,
			Next:                nextCommand,
			LastContinueOptions: continueOptionsToJSON(options),
		})
		blockedState, flowErr := recordBlockedContinueWorkerFlow(state, now, workerFlow)
		if flowErr != nil {
			if errors.Is(flowErr, errRuntimeStateSuperseded) {
				runStatus = "superseded"
				return continueSupersededResult(state, phase, flowErr), state, phase, nil, nil, false, nil
			}
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
			"review_depth":        string(reviewDepth),
			"plan_revision_option": planRevisionRecommendation(
				colony.PlanRevisionVerificationFailure,
				fmt.Sprintf("Review blocked phase %d: %s", phase.ID, summary),
				displayDataPath(verificationReportRel),
				displayDataPath(reviewReportRel),
			),
		}
		runStatus = "blocked"
		return result, blockedState, phase, nil, nil, false, nil
	}

	closedWorkerDetails := plannedCodexContinueClosedWorkers(manifest, assessment)
	closedWorkers := closedWorkerNames(closedWorkerDetails)

	// --- ATOMIC STATE COMMIT ---
	// Mutate and save colony state in a single atomic read-modify-write cycle.
	// If the mutation fails, no write occurs. This runs BEFORE side effects
	// and report saves so no external observer can see a partially advanced
	// state. advancePhase (cmd/advance_phase.go) is the one shared core both
	// aether continue and aether continue-finalize call -- see 188-CONTEXT.md
	// D-04/D-05/D-06.
	advanceResult, err := advancePhase(advancePhaseParams{
		PhaseID:                phase.ID,
		ExpectedBuildStartedAt: state.BuildStartedAt,
		AllowedStates:          []colony.State{colony.StateEXECUTING, colony.StateBUILT},
		Source:                 "continue",
		Now:                    now,
	})
	if err != nil {
		if errors.Is(err, errRuntimeStateSuperseded) {
			// FIELD-04: if this supersession is specifically because the
			// colony is paused, preserve this already-computed, already-
			// passing payload for replay after resume instead of discarding
			// it (cmd/advance_phase.go). Any other supersession reason
			// (phase or build identity genuinely changed) preserves
			// nothing -- discard exactly as before.
			preserveIfPausedSupersession(phase.ID, state.BuildStartedAt, "continue", now, pendingContinueAdvancePayload{
				Verification: verification,
				Assessment:   assessment,
				Gates:        gates,
				Review:       review,
				ReviewDepth:  reviewDepth,
			})
			runStatus = "superseded"
			return continueSupersededResult(state, phase, err), state, phase, nil, nil, false, nil
		}
		return nil, state, phase, nil, nil, false, fmt.Errorf("failed to atomically advance phase: %w", err)
	}
	updated := advanceResult.Updated
	nextPhase := advanceResult.NextPhase
	nextCommand := advanceResult.NextCommand
	final := advanceResult.Final

	// --- SIDE EFFECTS (after state is durable) ---
	// These operations produce derived data. If any fails, state is already
	// committed and valid; the error is informational, not a rollback trigger.
	housekeeping, housekeepingErr := continueSignalHousekeeper(now, updated)
	if housekeepingErr != nil {
		return nil, state, phase, nil, nil, false, housekeepingErr
	}
	if progress != nil {
		progress.Advance("Housekeeping")
	}
	if err := continueContextUpdater(phase, manifest, closedWorkerDetails, now); err != nil {
		return nil, state, phase, nil, nil, false, err
	}
	workerFlow := continueWorkerFlowForVerification(verification, review.Workers, watcherFlow)
	workerFlow = append(workerFlow, continueHousekeepingFlowStep(housekeeping))
	if err := recordContinueWorkerFlow(workerFlow); err != nil {
		return nil, state, phase, nil, &housekeeping, false, err
	}
	if err := applyCodexContinueWorkerClosures(closedWorkerDetails); err != nil {
		return nil, state, phase, nil, &housekeeping, false, err
	}
	// Durable learning capture on the DEFAULT path. This call is the fix for
	// "the colony never learns": capture previously existed only inside
	// continue-finalize, which the wrapper forbids for fast continue, so
	// pkg/learn, hypothesis promotion, and auto-skill creation were unreachable
	// in normal daily use. Gates have passed by this point; state is committed;
	// learning failure is non-blocking inside the function.
	captureContinueLearning(phase, workerFlow, gates, "", false, now)
	// D-04: phase-end consolidation fires only now, beside the learning
	// capture above, because this point is reached only after the atomic
	// COLONY_STATE.json write (above) committed PhaseCompleted -- the phase
	// has durably advanced. Never on a mid-phase continue. Non-blocking: a
	// consolidation failure is reported via the summary, never propagated
	// as an error (D-05).
	consolidationSummary := runPhaseEndConsolidation(phase.ID)
	workerFlow = append(workerFlow, continueLearningFlowStep(consolidationSummary))
	// The phase save-point: one git commit of exactly the files this phase's
	// workers reported changing, so repo history mirrors colony history.
	// Same non-fatal contract as consolidation — a commit failure is
	// reported (and pauses autopilot via the marker), never blocks.
	phaseCommit := commitPhaseAdvance(root, updated, phase)
	emitContinueCeremonyFlowSequence("aether-continue", phase, workerFlow)
	flowEvents := continueWorkerFlowEvents(now, workerFlow)
	updated.Events = append(updated.Events, flowEvents...)
	// Persist side-effect events (review, housekeeping) into colony state.
	// This is a best-effort append; the core advancement was already committed.
	_ = appendRuntimeStateEventsIfCurrent(updated, flowEvents)

	summary := fmt.Sprintf("Phase %d verified and advanced", phase.ID)
	if assessment.PartialSuccess {
		summary = fmt.Sprintf("Phase %d verified and advanced with partial operational success", phase.ID)
	}

	// --- REPORT SAVES (after state is durable) ---
	continueReportRel := filepath.ToSlash(filepath.Join("build", fmt.Sprintf("phase-%d", phase.ID), "continue.json"))
	if err := store.SaveJSON(continueReportRel, codexContinueReport{
		Phase:               phase.ID,
		GeneratedAt:         now.Format(time.RFC3339),
		Manifest:            displayOptionalDataPath(manifest.Path),
		VerificationReport:  displayDataPath(verificationReportRel),
		GateReport:          displayDataPath(gateReportRel),
		ReviewReport:        displayDataPath(reviewReportRel),
		Summary:             summary,
		ClosedWorkers:       closedWorkers,
		WorkerFlow:          workerFlow,
		PartialSuccess:      assessment.PartialSuccess,
		OperationalIssues:   append([]string{}, assessment.OperationalIssues...),
		Tasks:               append([]codexContinueTaskAssessment{}, assessment.Tasks...),
		Recovery:            assessment.Recovery,
		Advanced:            true,
		Completed:           final,
		Next:                nextCommand,
		LastContinueOptions: continueOptionsToJSON(options),
	}); err != nil {
		return nil, state, phase, nextPhase, &housekeeping, final, fmt.Errorf("failed to write continue report: %w", err)
	}

	updateSessionSummary("continue", nextCommand, summary)
	if progress != nil {
		progress.Advance("Advance")
		progress.Finish()
	}
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
		"review_depth":        string(reviewDepth),
	}
	if nextPhase != nil {
		result["next_phase"] = nextPhase.ID
		result["next_phase_name"] = nextPhase.Name
	}
	attachConsolidationSummary(result, consolidationSummary)
	attachPhaseCommitResult(result, phaseCommit)
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

func reconcileContinueCompletedBuildTasks(state *colony.ColonyState, phase *colony.Phase, manifest *codexContinueManifest) (bool, error) {
	if state == nil || phase == nil || manifest == nil || !manifest.Present {
		return false, nil
	}
	completed := completedBuildTaskIDs(manifest.Data.Dispatches)
	if len(completed) == 0 {
		return false, nil
	}

	changedState := false
	for idx := range phase.Tasks {
		taskID := buildTaskID(phase.Tasks[idx], idx)
		if _, ok := completed[taskID]; !ok {
			continue
		}
		if phase.Tasks[idx].Status != colony.TaskCompleted {
			phase.Tasks[idx].Status = colony.TaskCompleted
			changedState = true
		}
	}
	for phaseIdx := range state.Plan.Phases {
		if state.Plan.Phases[phaseIdx].ID != phase.ID {
			continue
		}
		for idx := range state.Plan.Phases[phaseIdx].Tasks {
			taskID := buildTaskID(state.Plan.Phases[phaseIdx].Tasks[idx], idx)
			if _, ok := completed[taskID]; !ok {
				continue
			}
			if state.Plan.Phases[phaseIdx].Tasks[idx].Status != colony.TaskCompleted {
				state.Plan.Phases[phaseIdx].Tasks[idx].Status = colony.TaskCompleted
				changedState = true
			}
		}
		break
	}

	changedManifest := false
	if len(manifest.Data.Tasks) == 0 {
		manifest.Data.Tasks = codexBuildTaskPlans(*phase)
		changedManifest = true
	}
	for idx := range manifest.Data.Tasks {
		taskID := strings.TrimSpace(manifest.Data.Tasks[idx].ID)
		if taskID == "" {
			continue
		}
		if _, ok := completed[taskID]; !ok {
			continue
		}
		if manifest.Data.Tasks[idx].Status != colony.TaskCompleted {
			manifest.Data.Tasks[idx].Status = colony.TaskCompleted
			changedManifest = true
		}
	}

	if changedState {
		var updated colony.ColonyState
		if err := store.UpdateJSONAtomically("COLONY_STATE.json", &updated, func() error {
			if err := validateRuntimeStateStillCurrent(updated, phase.ID, state.BuildStartedAt, colony.StateEXECUTING, colony.StateBUILT); err != nil {
				return err
			}
			for phaseIdx := range updated.Plan.Phases {
				if updated.Plan.Phases[phaseIdx].ID != phase.ID {
					continue
				}
				for idx := range updated.Plan.Phases[phaseIdx].Tasks {
					taskID := buildTaskID(updated.Plan.Phases[phaseIdx].Tasks[idx], idx)
					if _, ok := completed[taskID]; ok {
						updated.Plan.Phases[phaseIdx].Tasks[idx].Status = colony.TaskCompleted
					}
				}
				break
			}
			return nil
		}); err != nil {
			return false, fmt.Errorf("failed to reconcile completed build tasks in colony state: %w", err)
		}
		*state = updated
		for phaseIdx := range state.Plan.Phases {
			if state.Plan.Phases[phaseIdx].ID == phase.ID {
				*phase = state.Plan.Phases[phaseIdx]
				break
			}
		}
	}
	if changedManifest && strings.TrimSpace(manifest.Path) != "" {
		if err := store.SaveJSON(manifest.Path, manifest.Data); err != nil {
			return false, fmt.Errorf("failed to reconcile completed build tasks in manifest: %w", err)
		}
	}
	return changedState || changedManifest, nil
}

type codexContinueReviewSpec struct {
	Caste string
	Task  string
	// Rationale is the plain-English reason a forced reviewer (D-01..D-05,
	// cmd/queen_risk_signals.go) was sent — empty for a Queen-proposed or
	// keyword-selected spec, which carries no per-caste reason today.
	Rationale string
}

// codexContinueReviewSpecs' task text must never instruct a caste to run a
// CLI command: gatekeeper and auditor have no Bash tool by explicit design
// ("strictly read-only"), and telling them to run `aether review-ledger-write`
// gave them an unsatisfiable brief — they self-reported blocked, which
// blocked phase advancement (Pocket-Chopper field report). Workers return
// findings in their result JSON; the RUNTIME persists them to the domain
// review ledgers in-process (persistReviewFindingsToLedgers). Locked by
// TestReviewSpecsDoNotInstructBashlessCastes.
var codexContinueReviewSpecs = []codexContinueReviewSpec{
	{
		Caste: "gatekeeper",
		Task: "Review the phase for security, release, and integrity blockers before advancement. Return blocked if it is unsafe to advance." +
			"\n\nReturn your security findings in this result's findings array; the runtime records them in the domain review ledger for you.",
	},
	{
		Caste: "auditor",
		Task: "Audit whether the completed work actually satisfies the phase tasks rather than just producing superficial artifacts. Return blocked if the evidence looks partial, generic, or docs-only." +
			"\n\nReturn your quality, security, and performance findings in this result's findings array; the runtime records them in the domain review ledger for you.",
	},
	{
		Caste: "probe",
		Task:  "Probe the verification evidence for missing edge cases, weak tests, or unexercised behavior. Return blocked if test evidence is too weak to trust advancement.",
	},
}

func queenContinueDispatches(phase colony.Phase, reviewDepth colony.VerificationDepth) []CasteDispatch {
	return queenContinueDispatchesWithJudgement(phase, reviewDepth, nil, "", nil, nil)
}

// queenContinueDispatchesWithJudgement applies the Queen's chosen review team,
// bounded by the same floors as a build: the Watcher is restored if omitted,
// and a phase that requires a security or quality review keeps it. forced is
// the build-recorded D-01..D-05 forced-reviewer set (codexBuildManifest's
// ForcedReviewers, threaded in by callers that have a manifest); nil falls
// back to re-deriving from the phase's own wording
// (queenForcedContinueReviewers). changedFiles is the union of the builder's
// own reported changed_files for this phase and an independent `git diff`
// (phaseChangedFilesForRiskSignals, WR-01, supplied by both continue
// boundaries — D-02, plan 194-06): it can only ADD a forced reviewer via
// unionForcedContinueReviewers, never remove one.
func queenContinueDispatchesWithJudgement(phase colony.Phase, reviewDepth colony.VerificationDepth, proposed []string, reason string, forced []codexForcedReviewerRecord, changedFiles []string, reasons ...map[string]string) []CasteDispatch {
	state := colony.ColonyState{VerificationDepth: string(reviewDepth)}
	var dispatches []CasteDispatch
	if len(proposed) == 0 {
		dispatches = queenOrchestrate(phase, "continue", state)
	} else {
		judgement := queenApplyJudgement(proposed, reason, phase, "continue", state, reasons...)
		dispatches = make([]CasteDispatch, 0, len(judgement.Final))
		for _, caste := range judgement.Final {
			dispatches = append(dispatches, CasteDispatch{
				Caste: caste,
				// A per-worker reason (D-08, D-10) beats the team summary
				// when one exists for this caste -- the summary survives
				// only as the fallback for a caste judgement.Reasons has
				// nothing entered for (a required/added caste the runtime
				// itself has not yet written a reason for).
				Rationale: casteDispatchRationale(judgement, caste),
				FlowType:  "continue",
			})
		}
	}
	return unionForcedContinueReviewers(dispatches, phase, forced, changedFiles, proposed, reviewDepth)
}

// casteDispatchRationale prefers the judgement's per-caste reason (D-08,
// D-10) over its single team-level Rationale string, so a continue dispatch
// states why THIS worker was sent rather than reusing the whole team's
// summary sentence for every member of it.
func casteDispatchRationale(judgement queenCasteJudgement, caste string) string {
	if r := strings.TrimSpace(judgement.Reasons[caste]); r != "" {
		return r
	}
	return judgement.Rationale
}

// unionForcedContinueReviewers adds any D-01..D-05 forced-reviewer caste
// missing from dispatches AFTER the proposal/budget handling above, so
// neither a Queen proposal nor a budget trim can ever drop one (D-05: "one
// derivation, one boundary" — closes .planning/WINDOWS.md #1's continue
// side). A forced caste already present has its Rationale overwritten with
// the forced reason, so the owner sees the real reason it is there even if
// it was also proposed or keyword-selected. changedFiles carries the D-02
// file-detected hits into the same union.
func unionForcedContinueReviewers(dispatches []CasteDispatch, phase colony.Phase, forced []codexForcedReviewerRecord, changedFiles []string, proposed []string, reviewDepth colony.VerificationDepth) []CasteDispatch {
	reviewers := queenForcedContinueReviewers(phase, forced, changedFiles)
	liveReviewerCastes := make(map[string]bool, len(reviewers))
	for _, reviewer := range reviewers {
		liveReviewerCastes[reviewer.Caste] = true
	}

	// queenFallbackTeam and queenApplyJudgement both consult the phase-wording
	// requirement before the owner waiver is applied. Reconcile that earlier
	// snapshot here, at the shared dispatch boundary, so a waived signal cannot
	// survive merely because it was already present in dispatches. Preserve a
	// reviewer that has an independent reason to run: another live signal for
	// the same caste, an explicit Queen proposal, or the owner's heavy-depth
	// review panel.
	phaseSignalCastes := make(map[string]bool)
	for _, reviewer := range queenForcedReviewersForPhase(phase) {
		phaseSignalCastes[reviewer.Caste] = true
	}
	// Preserve explicit reviewers using the exact same normalization as
	// queenApplyJudgement. Comparing raw flag values here dropped accepted
	// aliases ("security" -> gatekeeper), separator variants, and comma-packed
	// proposals after judgement had already produced the canonical dispatch.
	normalizedProposed, _ := normalizeProposedCastes(proposed)
	proposedCastes := stringSet(normalizedProposed)
	state := colony.ColonyState{VerificationDepth: string(reviewDepth)}
	result := make([]CasteDispatch, 0, len(dispatches)+len(reviewers))
	for _, dispatch := range dispatches {
		caste := strings.ToLower(strings.TrimSpace(dispatch.Caste))
		waivedSignalOnly := phaseSignalCastes[caste] && !liveReviewerCastes[caste] &&
			!proposedCastes[caste] && !isAlwaysRequired(caste, "continue", phase, state)
		if waivedSignalOnly {
			continue
		}
		result = append(result, dispatch)
	}

	indexByCaste := make(map[string]int, len(result))
	for i, dispatch := range result {
		indexByCaste[dispatch.Caste] = i
	}
	for _, reviewer := range reviewers {
		if idx, ok := indexByCaste[reviewer.Caste]; ok {
			result[idx].Rationale = reviewer.Reason
			continue
		}
		result = append(result, CasteDispatch{
			Caste:     reviewer.Caste,
			Rationale: reviewer.Reason,
			FlowType:  "continue",
		})
	}
	return result
}

func queenContinueHasCaste(dispatches []CasteDispatch, caste string) bool {
	for _, dispatch := range dispatches {
		if dispatch.Caste == caste {
			return true
		}
	}
	return false
}

func queenContinueReviewSpecs(phase colony.Phase, reviewDepth colony.VerificationDepth) []codexContinueReviewSpec {
	return queenContinueReviewSpecsWithJudgement(phase, reviewDepth, nil, "", nil, nil)
}

// changedFiles is the union of the builder's own reported changed_files for
// this phase and an independent `git diff` (phaseChangedFilesForRiskSignals,
// WR-01), supplied identically by both continue boundaries —
// plannedContinueReviewDispatches (this file) and
// plannedExternalContinueDispatches (cmd/codex_continue_plan.go) — so the
// two lanes can never derive a different forced-reviewer set for the same
// phase and the same changed files (D-02, two-lane parity discipline).
func queenContinueReviewSpecsWithJudgement(phase colony.Phase, reviewDepth colony.VerificationDepth, proposed []string, reason string, forced []codexForcedReviewerRecord, changedFiles []string, reasons ...map[string]string) []codexContinueReviewSpec {
	queenDispatches := queenContinueDispatchesWithJudgement(phase, reviewDepth, proposed, reason, forced, changedFiles, reasons...)
	forcedReasons := make(map[string]string, len(forced))
	for _, reviewer := range queenForcedContinueReviewers(phase, forced, changedFiles) {
		forcedReasons[reviewer.Caste] = reviewer.Reason
	}
	specs := make([]codexContinueReviewSpec, 0, len(queenDispatches))
	for _, dispatch := range queenDispatches {
		if dispatch.Caste == "watcher" {
			continue
		}
		spec, ok := continueReviewSpecForCaste(dispatch.Caste)
		if !ok {
			continue
		}
		// A forced signal's own sentence (D-01..D-05) always wins when this
		// caste was forced; otherwise the dispatch already carries the best
		// reason available (per-worker if the proposal gave one, the team
		// summary otherwise) via casteDispatchRationale.
		spec.Rationale = dispatch.Rationale
		if forcedReason := strings.TrimSpace(forcedReasons[dispatch.Caste]); forcedReason != "" {
			spec.Rationale = forcedReason
		}
		specs = append(specs, spec)
	}
	return specs
}

func continueReviewSpecForCaste(caste string) (codexContinueReviewSpec, bool) {
	for _, spec := range codexContinueReviewSpecs {
		if spec.Caste == caste {
			return spec, true
		}
	}
	switch caste {
	case "measurer":
		return codexContinueReviewSpec{
			Caste: "measurer",
			Task:  "Review whether the completed phase introduces performance, latency, memory, or cost regressions. Return blocked only for concrete regression evidence that makes advancement unsafe.",
		}, true
	case "chaos":
		return codexContinueReviewSpec{
			Caste: "chaos",
			Task:  "Probe resilience and failure-handling evidence for the completed phase. Return blocked only for reproducible failure paths or missing recovery behavior that make advancement unsafe.",
		}, true
	case "includer":
		return codexContinueReviewSpec{
			Caste: "includer",
			Task:  "Review accessibility and inclusive-use risks for the completed phase. Return blocked only for concrete accessibility regressions or missing required affordances.",
		}, true
	case "keeper":
		return codexContinueReviewSpec{
			Caste: "keeper",
			Task:  "Review whether important conventions, decisions, or knowledge from the phase were preserved for future workers. Return blocked only if missing knowledge creates immediate advancement risk.",
		}, true
	case "sage":
		return codexContinueReviewSpec{
			Caste: "sage",
			Task:  "Synthesize cross-phase learning and advancement risk from the verification evidence. Return blocked only if the evidence shows a concrete unresolved decision or contradiction.",
		}, true
	case "medic":
		return codexContinueReviewSpec{
			Caste: "medic",
			Task:  "Diagnose colony and runtime health before advancement. Return blocked only for state, manifest, or recovery issues that would make continue unsafe.",
		}, true
	case "fixer":
		return codexContinueReviewSpec{
			Caste: "fixer",
			Task:  "Review recoverable blockers and propose the smallest safe repair path. This review is read-only; return blocked only when a concrete repair is required before advancement.",
		}, true
	}
	return codexContinueReviewSpec{}, false
}

func runCodexContinueReview(root string, phase colony.Phase, manifest codexContinueManifest, verification codexContinueVerificationReport, assessment codexContinueAssessment, workerTimeout time.Duration, reviewDepth colony.VerificationDepth, skipWatchers bool, queenCastes []string, queenCasteReason string, reasons ...map[string]string) codexContinueReviewReport {
	report := codexContinueReviewReport{
		Phase:       phase.ID,
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Workers:     []codexContinueWorkerFlowStep{},
		Passed:      false,
	}
	if skipWatchers {
		// Verification and runtime gates already passed; --skip-watchers means
		// continue must not launch any platform watcher or review agents.
		report.Workers = append(report.Workers, continueReviewSkippedFlowStep("review wave skipped by --skip-watchers; no platform review agents were launched"))
		report.Passed = true
		return report
	}

	invoker := newCodexWorkerInvoker()
	if _, ok := invoker.(*codex.FakeInvoker); !ok && !invoker.IsAvailable(context.Background()) {
		availabilityMessage := dispatchAvailabilityMessage(invoker)
		fmt.Fprintf(os.Stderr, "⚠ Worker dispatcher unavailable — skipping review wave, proceeding with claims verification: %s\n", availabilityMessage)
		report.Workers = append(report.Workers, continueReviewSkippedFlowStep("review wave skipped; "+availabilityMessage+", proceeding with claims verification"))
		report.Passed = true
		return report
	}

	dispatches := plannedContinueReviewDispatches(root, phase, manifest, verification, assessment, invoker, workerTimeout, reviewDepth, queenCastes, queenCasteReason, reasons...)
	if len(dispatches) == 0 {
		report.Workers = append(report.Workers, continueReviewSkippedFlowStep(continueReviewSkippedSummary(reviewDepth)))
		report.Passed = true
		return report
	}
	spawnTree := agent.NewSpawnTree(store, "spawn-tree.txt")
	reviewCtx, reviewCancel := context.WithTimeout(context.Background(), effectiveContinueReviewTimeout(workerTimeout))
	defer reviewCancel()
	results, err := dispatchBatchByWaveWithVisuals(
		reviewCtx,
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
				step.Blockers = uniqueSortedStrings(result.WorkerResult.Blockers)
				step.Duration = result.WorkerResult.Duration.Seconds()
				step.Report = codex.SanitizeWorkerDiagnosticOutput(result.WorkerResult.RawOutput)
			}
			if step.Summary == "" && result.Error != nil {
				step.Summary = codex.SanitizeWorkerDiagnosticOutput(result.Error.Error())
			}
			if step.Summary == "" {
				step.Summary = continueReviewFlowSummary(step)
			}
			if verification.ChecksPassed && continueWorkerFlowEnvironmentBlocked(step) {
				step.Status = watcherStatusEnvironmentBlocked
				step.Summary = environmentBlockedLaunchSummary(step.Summary)
			}
			if continueReviewStepBlocks(step, verification) {
				for _, blocker := range step.Blockers {
					if strings.TrimSpace(blocker) != "" {
						blockers = append(blockers, fmt.Sprintf("%s reported blocker: %s", result.WorkerName, blocker))
					}
				}
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

func plannedContinueReviewDispatches(root string, phase colony.Phase, manifest codexContinueManifest, verification codexContinueVerificationReport, assessment codexContinueAssessment, invoker codex.WorkerInvoker, workerTimeout time.Duration, reviewDepth colony.VerificationDepth, queenCastes []string, queenCasteReason string, reasons ...map[string]string) []codex.WorkerDispatch {
	capsule := resolveCodexWorkerContext()
	// PheromoneSection is deliberately left unset (D-190-03-A / 190-05): capsule
	// already renders "## Pheromone Signals" unconditionally whenever a signal is
	// active (cmd/colony_prime_context.go:571). Populating a second, independent
	// PheromoneSection field here would deliver the same steering text twice into
	// AssemblePrompt/AssembleHostedPrompt. See resolvePheromoneSection's doc
	// comment for which callers still need it.
	timeout := effectiveContinueReviewTimeout(workerTimeout)
	// The Queen's --castes proposal used to be honoured only on the heavy
	// plan-only path; the default path called the nil-proposal variant, so on
	// the continue users actually run the keyword engine was unchallenged.
	// changedFiles feeds the D-02 file-detected forced-reviewer union
	// (queenForcedContinueReviewers): what the builder actually touched can
	// raise a reviewer the plan's own wording missed, on this lane exactly
	// as on the wrapper lane (plannedExternalContinueDispatches).
	// phaseChangedFilesForRiskSignals (not the raw phaseChangedFilesFromHandoffs)
	// per WR-01, 194-REVIEW.md: unions the builder's own self-report with an
	// independent `git diff`, so an incomplete or dishonest handoff cannot
	// defeat this detector by itself.
	changedFiles := phaseChangedFilesForRiskSignals(phase.ID)
	specs := queenContinueReviewSpecsWithJudgement(phase, reviewDepth, queenCastes, queenCasteReason, manifest.Data.ForcedReviewers, changedFiles, reasons...)
	dispatches := make([]codex.WorkerDispatch, 0, len(specs))
	for idx, spec := range specs {
		agentName := codexAgentNameForCaste(spec.Caste)
		dispatches = append(dispatches, codex.WorkerDispatch{
			ID:             fmt.Sprintf("continue-review-%d", idx),
			WorkerName:     deterministicAntName(spec.Caste, fmt.Sprintf("phase:%d:continue:%s", phase.ID, spec.Caste)),
			AgentName:      agentName,
			AgentTOMLPath:  dispatchAgentPath(root, invoker, agentName),
			Caste:          spec.Caste,
			TaskID:         fmt.Sprintf("continue-review-%s", spec.Caste),
			TaskBrief:      renderCodexContinueReviewBrief(root, phase, manifest, verification, assessment, spec),
			ContextCapsule: capsule,
			// D-190-05-A / 190-06: renderRelatedWorkflowHandoffSection, not
			// renderWorkerHandoffSection -- capsule (above) already renders
			// "## Previous Worker Handoffs" for "build"-workflow records
			// (cmd/colony_prime_context.go:695). This relays "continue"-workflow
			// records (sibling review/watcher dispatches) under a distinct
			// heading so both channels keep exactly one home each.
			HandoffSection: renderRelatedWorkflowHandoffSection("continue", phase.ID, deterministicAntName(spec.Caste, fmt.Sprintf("phase:%d:continue:%s", phase.ID, spec.Caste))),
			Workflow:       "continue",
			Phase:          phase.ID,
			SkillSection:   resolveSkillSectionForWorkflow("continue", spec.Caste, spec.Task),
			Root:           root,
			Timeout:        timeout,
			Wave:           1,
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
	if strings.TrimSpace(spec.Rationale) != "" {
		// D-05/D-10: a forced reviewer states why it was sent, in the same
		// plain-English sentence the owner sees on the check-in card and
		// manifest — the worker should not have to guess why it was called.
		b.WriteString("Why you were sent: ")
		b.WriteString(spec.Rationale)
		b.WriteString("\n\n")
	}
	if spec.Caste == "gatekeeper" || spec.Caste == "auditor" {
		// These two castes have no Bash tool by design — never instruct them
		// to run a CLI command. They return findings in result JSON and the
		// runtime persists to the ledger (persistReviewFindingsToLedgers).
		b.WriteString("This is a review task. Return your findings in this result's findings array — the runtime records them in the domain review ledger for you. Do not modify repo source files. Return status `blocked` if advancement is unsafe.\n\n")
	} else {
		b.WriteString("This is a read-only review. Do not modify repo files. Return status `blocked` if advancement is unsafe.\n\n")
	}
	if spec.Caste == "probe" {
		b.WriteString("Coverage guidance: if runtime verification checks passed, package-wide line coverage below an aspirational threshold is advisory by itself. Block only for red verification commands, missing focused regression coverage for changed behavior, or concrete unexercised edge cases that make advancement unsafe.\n\n")
	}
	b.WriteString(renderWorkerReadCacheDiscipline())
	b.WriteString("\n")
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
		b.WriteString(trimBriefList("", assessment.OperationalIssues, 5))
	}
	if surveySection := resolveSurveySection(); surveySection != "" {
		b.WriteString("\n")
		b.WriteString(surveySection)
		b.WriteString("\n")
	}
	b.WriteString(renderVerificationCommandSection())
	return b.String()
}

// trimBriefList truncates a list of items for inclusion in a worker brief.
// If there are more than max items, it shows the first max and a summary.
func trimBriefList(prefix string, items []string, max int) string {
	if len(items) == 0 {
		return ""
	}
	var b strings.Builder
	for i, item := range items {
		if i >= max {
			b.WriteString(fmt.Sprintf("%s... and %d more (see assessment report)\n", prefix, len(items)-max))
			break
		}
		b.WriteString(prefix)
		b.WriteString("- ")
		b.WriteString(item)
		b.WriteString("\n")
	}
	return b.String()
}

// isCodexWorkerAvailable checks whether the Codex CLI worker invoker is
// available for spawning watcher/review agents.
func isCodexWorkerAvailable() bool {
	invoker := newCodexWorkerInvoker()
	if _, ok := invoker.(*codex.FakeInvoker); ok {
		return true
	}
	return invoker.IsAvailable(context.Background())
}

func runCodexContinueVerification(ctx context.Context, root string, state colony.ColonyState, phase colony.Phase, manifest codexContinueManifest, workerTimeout time.Duration, verificationTimeout time.Duration, skipWatchers bool) (codexContinueVerificationReport, *codexContinueWorkerFlowStep) {
	if ctx == nil {
		ctx = context.Background()
	}
	now := time.Now().UTC()
	buildWatcher := evaluateContinueWatcherVerification(manifest)

	// The deterministic floor is computed first, with whatever watcher value
	// is already resolved (buildWatcher -- from the manifest, no live action
	// needed) -- never with a live dispatch this call might still make below.
	// It is what sets checksPassed (ruling D11 rule 2): a reviewer dispatched
	// afterward can only ever ADD a block on top of this result, never
	// supply the pass (TestDeterministicFloorIsTheOnlySourceOfAPass).
	floor := runDeterministicFloor(ctx, root, phase, manifest, buildWatcher, verificationTimeout)

	// continueWatcherDecision resolves only whether a reviewer is dispatched
	// at all -- it never consults the floor's result, so the deterministic
	// outcome can never cause a dispatch (D-08): dispatch happens because
	// the Queen sent a reviewer and none of the existing auto-skip reasons
	// apply, full stop. Dispatch (when it happens) reuses floor.Steps and
	// floor.Claims for the reviewer's brief context rather than re-running
	// verification a second time.
	dispatchReviewer, skipped := continueWatcherDecision(state, phase, manifest, buildWatcher, skipWatchers)
	var continueWatcher codexWatcherVerification
	var watcherFlow *codexContinueWorkerFlowStep
	if dispatchReviewer {
		continueWatcher, watcherFlow = runCodexContinueWatcherVerification(ctx, root, phase, manifest, floor.Steps, floor.Claims, buildWatcher, workerTimeout)
	} else {
		continueWatcher = skipped
	}
	watcher := continueWatcher
	if !watcher.Present {
		watcher = buildWatcher
	}

	// D-02/D-03: when a free check failed and no reviewer was dispatched,
	// attempt exactly one bounded automatic builder fix and use its result
	// as the effective floor for the rest of this function.
	// applyAutomaticCheckFixAttempt itself decides eligibility (the floor
	// already passing, a reviewer having been dispatched, no failing shell
	// step, or a fix attempt already recorded for this phase and check all
	// return the floor unchanged) -- this call site never re-derives that
	// decision, so there is exactly one place it is made.
	floor, checkFixAttempt := applyAutomaticCheckFixAttempt(ctx, root, state, phase, manifest, floor, buildWatcher, workerTimeout, verificationTimeout, dispatchReviewer)

	checksPassed := floor.ChecksPassed
	blockers := append([]string{}, floor.BlockingIssues...)
	warnings := append([]string{}, floor.Warnings...)
	if watcher.Present && !watcher.Passed && watcher.Status != "skipped" {
		summary := strings.TrimSpace(watcher.Summary)
		if summary == "" {
			summary = "watcher verification did not complete cleanly"
		}
		if watcher.Status == "timeout" && checksPassed {
			// Advisory: watcher timed out but the deterministic floor
			// passed independently. Treat as warning, not a hard block.
			blockers = append(blockers, fmt.Sprintf(
				"watcher %s timed out; runtime verification passed independently", watcher.Worker))
		} else if checksPassed && isEnvironmentBlockedWatcher(watcher) {
			watcher = environmentBlockedWatcher(watcher)
		} else {
			checksPassed = false
			blockers = append(blockers, summary)
		}
	}

	// LOOP-01: Update watcher failure counter based on watcher outcome.
	if continueWatcher.Present && !continueWatcher.Passed && continueWatcher.Status == "failed" {
		_ = incrementWatcherFailureCount(phase.ID)
	} else if continueWatcher.Passed && continueWatcher.Status != "skipped" {
		// Only reset on actual pass, not on skip (preserve count for diagnostic purposes).
		_ = resetWatcherFailureCount(phase.ID)
	}

	return codexContinueVerificationReport{
		Phase:                      phase.ID,
		GeneratedAt:                now.Format(time.RFC3339),
		VerificationTimeoutSeconds: int(effectiveContinueVerificationTimeout(verificationTimeout) / time.Second),
		Steps:                      floor.Steps,
		Claims:                     floor.Claims,
		Watcher:                    watcher,
		CriteriaPolicy:             floor.Criteria.Policy,
		CriteriaEnforced:           floor.Criteria.Enforced,
		CriteriaPassed:             floor.Criteria.Passed,
		Criteria:                   floor.Criteria.Criteria,
		ChecksPassed:               checksPassed,
		Passed:                     checksPassed,
		BlockingIssues:             blockers,
		Warnings:                   warnings,
		CheckFixAttempt:            checkFixAttempt,
	}, watcherFlow
}

// continueWatcherDecision decides whether continue dispatches a reviewer
// worker at all. It never consults the deterministic verification result --
// the removed comment block above named the exact fallback this replaces
// ("zero executed checks hands verification to a watcher"); that coupling is
// gone. Every existing non-dispatch reason keeps its own explicit skipped
// status: --skip-watchers, a build-side watcher already environment-blocked,
// a host-boundary (wrapper-mediated) skip, no worker provider available, or
// the consecutive-failure circuit breaker (LOOP-01). Returning dispatch=true
// means only "none of those apply" -- the caller still runs the reviewer and
// its own outcome is evaluated afterward.
func continueWatcherDecision(state colony.ColonyState, phase colony.Phase, manifest codexContinueManifest, buildWatcher codexWatcherVerification, skipWatchers bool) (bool, codexWatcherVerification) {
	if skipWatchers {
		return false, codexWatcherVerification{Present: true, Passed: true, Status: "skipped", Worker: "skip-watchers", Summary: "watcher skipped; relying on verification commands"}
	}
	if isEnvironmentBlockedWatcher(buildWatcher) {
		return false, buildWatcher
	}
	if summary, ok := continueWatcherHostBoundarySkipSummary(manifest); ok {
		return false, resolveHostBoundaryWatcher(buildWatcher, summary)
	}
	invoker := newCodexWorkerInvoker()
	if _, ok := invoker.(*codex.FakeInvoker); !ok && !invoker.IsAvailable(context.Background()) {
		// Auto-skip: no authenticated worker provider is available. No point
		// dispatching a watcher that will immediately fail; the deterministic
		// floor still decides.
		return false, codexWatcherVerification{Present: true, Passed: true, Status: "skipped", Worker: "auto-skip", Summary: "watcher auto-skipped; " + dispatchAvailabilityMessage(invoker) + "; advancing on the deterministic floor"}
	}
	if getWatcherFailureCount(state, phase.ID) >= defaultWatcherFailureThreshold {
		// LOOP-01: Auto-skip watcher after consecutive failure threshold.
		emitLoopBreakEvent("watcher_skip",
			fmt.Sprintf("%d consecutive watcher failures", getWatcherFailureCount(state, phase.ID)),
			"auto-skipped watcher, advancing on the deterministic floor",
			"aether-continue")
		return false, codexWatcherVerification{Present: true, Passed: true, Status: "skipped", Worker: "auto-skip", Summary: fmt.Sprintf("watcher auto-skipped after %d consecutive failures. Advancing on the deterministic floor.", getWatcherFailureCount(state, phase.ID))}
	}
	return true, codexWatcherVerification{}
}

func runCodexContinueWatcherVerification(ctx context.Context, root string, phase colony.Phase, manifest codexContinueManifest, steps []codexVerificationStep, claims codexClaimVerification, buildWatcher codexWatcherVerification, workerTimeout time.Duration) (codexWatcherVerification, *codexContinueWorkerFlowStep) {
	if ctx == nil {
		ctx = context.Background()
	}
	invoker := newCodexWorkerInvoker()
	dispatch := plannedContinueWatcherDispatch(root, phase, manifest, steps, claims, buildWatcher, invoker, workerTimeout)
	if !invoker.IsAvailable(ctx) {
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
	watcherCtx, watcherCancel := context.WithTimeout(ctx, effectiveContinueReviewTimeout(workerTimeout))
	defer watcherCancel()
	results, err := dispatchBatchByWaveWithVisuals(
		watcherCtx,
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
		status := "failed"
		if errors.Is(err, context.DeadlineExceeded) {
			status = "timeout"
		}
		return codexWatcherVerification{
				Present: true,
				Passed:  false,
				Status:  status,
				Worker:  dispatch.WorkerName,
				Summary: summary,
			}, &codexContinueWorkerFlowStep{
				Stage:   "verification",
				Caste:   "watcher",
				Name:    dispatch.WorkerName,
				Task:    "Independent verification before advancement",
				Status:  status,
				Summary: continueWatcherFlowSummary(dispatch.WorkerName, status, summary),
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
	if status == "failed" && result.Error != nil && errors.Is(result.Error, context.DeadlineExceeded) {
		status = "timeout"
	}
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
			Passed:  isSuccessfulExternalBuildStatus(status),
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

func plannedContinueWatcherDispatch(root string, phase colony.Phase, manifest codexContinueManifest, steps []codexVerificationStep, claims codexClaimVerification, buildWatcher codexWatcherVerification, invoker codex.WorkerInvoker, workerTimeout time.Duration) codex.WorkerDispatch {
	agentName := codexAgentNameForCaste("watcher")
	return codex.WorkerDispatch{
		ID:             fmt.Sprintf("continue-verification-%d", phase.ID),
		WorkerName:     deterministicAntName("watcher", fmt.Sprintf("phase:%d:continue:watcher", phase.ID)),
		AgentName:      agentName,
		AgentTOMLPath:  dispatchAgentPath(root, invoker, agentName),
		Caste:          "watcher",
		TaskID:         fmt.Sprintf("continue-verification-%d", phase.ID),
		TaskBrief:      renderCodexContinueWatcherBrief(root, phase, manifest, steps, claims, buildWatcher, workerTimeout),
		ContextCapsule: resolveCodexWorkerContext(),
		SkillSection:   resolveSkillSectionForWorkflow("continue", "watcher", "Independent verification before advancement"),
		// PheromoneSection is deliberately left unset (D-190-03-A / 190-05):
		// ContextCapsule above already renders "## Pheromone Signals"
		// unconditionally whenever a signal is active
		// (cmd/colony_prime_context.go:571) -- a second, independent
		// PheromoneSection field would deliver the same text twice. See
		// resolvePheromoneSection's doc comment for which callers still need it.
		// The watcher is the most expensive single worker in the flow and was
		// the only one dispatched without the relay — its sibling reviewers get
		// it. Nothing in the design justified the asymmetry; it was omitted.
		// D-190-05-A / 190-06: renderRelatedWorkflowHandoffSection, not
		// renderWorkerHandoffSection -- ContextCapsule above already renders
		// "## Previous Worker Handoffs" for "build"-workflow records
		// (cmd/colony_prime_context.go:695). This relays "continue"-workflow
		// records under a distinct heading so both channels keep exactly one
		// home each.
		HandoffSection: renderRelatedWorkflowHandoffSection("continue", phase.ID,
			deterministicAntName("watcher", fmt.Sprintf("phase:%d:continue:watcher", phase.ID))),
		Root:    root,
		Timeout: effectiveContinueReviewTimeout(workerTimeout),
		Wave:    1,
	}
}

func renderCodexContinueWatcherBrief(root string, phase colony.Phase, manifest codexContinueManifest, steps []codexVerificationStep, claims codexClaimVerification, buildWatcher codexWatcherVerification, workerTimeout time.Duration) string {
	var b strings.Builder
	b.WriteString("# Continue Verification\n\n")
	b.WriteString("- Phase: ")
	b.WriteString(fmt.Sprintf("%d — %s\n", phase.ID, phase.Name))
	b.WriteString("- Repo: ")
	b.WriteString(root)
	b.WriteString(fmt.Sprintf("\n- Time limit: %s — prioritize critical checks and return partial results rather than timing out\n",
		effectiveContinueReviewTimeout(workerTimeout)))
	b.WriteString("\n")
	b.WriteString("Confirm whether this phase is safe to advance right now.\n")
	b.WriteString("This is a read-only verification pass. Do not modify repo files. Return status `completed` only if the current workspace and recorded artifacts justify advancement. Return status `blocked` if anything is missing, stale, or misleading.\n\n")
	b.WriteString("The Fresh Evidence rule from your agent definition is SUSPENDED for this task — the verification commands below were executed by the runtime moments ago. Trust their output. Do NOT re-run the test suite or build commands.\n\n")
	b.WriteString(renderWorkerReadCacheDiscipline())
	b.WriteString("\n")
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
	b.WriteString("\nFull verification output: see .aether/data/build/phase-")
	b.WriteString(fmt.Sprintf("%d/verification.json\n", phase.ID))
	if len(phase.Tasks) > 0 {
		b.WriteString("\nPhase tasks:\n")
		if phase.Mode == colony.PhaseModeDiscovery && len(phase.Tasks) > 5 {
			b.WriteString(fmt.Sprintf("- %d tasks total (see manifest for full list)\n", len(phase.Tasks)))
		} else {
			for idx, task := range phase.Tasks {
				b.WriteString("- ")
				b.WriteString(buildTaskID(task, idx))
				b.WriteString(": ")
				b.WriteString(strings.TrimSpace(task.Goal))
				b.WriteString("\n")
			}
		}
	}
	b.WriteString(renderVerificationCommandSection())
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
			switch {
			case isSuccessfulExternalBuildStatus(status):
				summary = "watcher verification completed before advancement"
			default:
				summary = fmt.Sprintf("watcher verification did not complete cleanly: %s", status)
			}
		}
		watcher := codexWatcherVerification{
			Present: true,
			Passed:  isSuccessfulExternalBuildStatus(status),
			Status:  status,
			Worker:  strings.TrimSpace(dispatch.Name),
			Summary: summary,
		}
		if isEnvironmentBlockedLaunchVerification(strings.Join(append([]string{summary}, dispatch.Blockers...), "\n")) {
			return environmentBlockedWatcher(watcher)
		}
		return watcher
	}
	return codexWatcherVerification{}
}

func isEnvironmentBlockedWatcher(watcher codexWatcherVerification) bool {
	if !watcher.Present {
		return false
	}
	if strings.EqualFold(strings.TrimSpace(watcher.Status), watcherStatusEnvironmentBlocked) {
		return true
	}
	return isEnvironmentBlockedLaunchVerification(watcher.Summary)
}

func environmentBlockedWatcher(watcher codexWatcherVerification) codexWatcherVerification {
	watcher.Present = true
	watcher.Passed = true
	watcher.Status = watcherStatusEnvironmentBlocked
	watcher.Summary = environmentBlockedLaunchSummary(watcher.Summary)
	return watcher
}

func environmentBlockedLaunchSummary(original string) string {
	summary := "launch verification environment_blocked: local socket binding is denied in this host; external launch proof is required from an environment that permits socket binding"
	original = strings.TrimSpace(original)
	if original == "" || strings.Contains(strings.ToLower(original), "environment_blocked") {
		if original != "" {
			return original
		}
		return summary
	}
	return summary + ": " + original
}

// mergeReconcileTaskIDs unions --reconcile-task values recorded live at
// continue-finalize time with the reconcile task IDs already persisted on
// the plan manifest (recorded at `aether continue --plan-only` time),
// deduplicated and sorted. Closes the 2026-08-01 folded todo: before this,
// continue-finalize had no --reconcile-task flag at all, so an operator
// could only record reconciliation up front at plan-only time, never after.
func mergeReconcileTaskIDs(planIDs, flagIDs []string) []string {
	if len(flagIDs) == 0 {
		return append([]string{}, planIDs...)
	}
	merged := append(append([]string{}, planIDs...), flagIDs...)
	return uniqueSortedStrings(merged)
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
		for _, taskID := range dispatchCoveredTaskIDs(dispatch) {
			dispatchStatuses[taskID] = append(dispatchStatuses[taskID], status)
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
	if continueVerificationTimedOut(verification) {
		recovery.ReverifyCommand = buildContinueVerificationTimeoutRecoveryCommand(options, nil)
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
	// D-02: when the automatic fix attempt ran and the re-run still failed,
	// the recovery plan carries the single exact command to re-run the
	// builder by hand -- never a menu.
	if fix := verification.CheckFixAttempt; fix != nil && fix.Outcome == "still_failing" {
		fixCommand := buildTargetedRedispatchCommand(phase.ID, fix.FailureIndex.ImplicatedTaskIDs)
		if strings.TrimSpace(fixCommand) == "" {
			fixCommand = buildForceRedispatchCommand(phase.ID)
		}
		recovery.CheckFixCommand = fixCommand
	}

	passed := verification.ChecksPassed && positiveEvidence
	if len(options.ReconcileTaskIDs) > 0 {
		// Visible but non-blocking: the reconcile itself is legitimate
		// recovery (H-04); verification and evidence still gate advancement.
		operationalIssues = append(operationalIssues, fmt.Sprintf("%d task(s) were manually reconciled; verification was re-run before advancement", len(options.ReconcileTaskIDs)))
	}
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
		// Phases created by `aether phase-insert` carry no task list; their
		// implementation evidence is the build's verified claims. Requiring
		// task-bound evidence here made every inserted phase permanently
		// unadvanceable.
		return claimsSatisfied
	}
	for _, task := range tasks {
		switch task.Outcome {
		case "missing", "needs_redispatch", "implemented_unverified", "simulated":
			return false
		case "manually_reconciled":
			// H-04, extended by FLOOR-03 (.planning/todos/pending/2026-08-01-
			// finalize-reconcile-task-evidence-gate.md, folded into 193-04): a
			// manually reconciled task must be able to advance when the
			// deterministic floor (build/types/lint/tests, claimed-files-exist,
			// and each criterion's evidence -- task.Verified is
			// verification.ChecksPassed, which already folds criterion evidence
			// in via runDeterministicFloor) passed, even when no builder-claims
			// file exists to satisfy claimsSatisfied. An operator reconciling
			// work done entirely outside the pipeline has no claims file to
			// satisfy by construction -- requiring one anyway is precisely the
			// asymmetry that deadlocked the finalize lane (the direct
			// `aether continue` path could still hand-record claims; the
			// external wrapper lane had no way to record --reconcile-task at
			// finalize time at all until this plan added the flag). Reconcile
			// is still not a bypass: a reconciled task whose deterministic
			// floor failed still blocks, and an UNRECONCILED task's builder-
			// claim failure still blocks via the "implemented_unverified" /
			// "needs_redispatch" outcomes above, untouched by this change.
			if !task.Verified {
				return false
			}
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
		if !containsString(statuses, "failed") {
			return "verified", "Phase verification passed; no worker reported completion but all checks passed.", ""
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
		if strings.TrimSpace(task.RecoveryAction) == "redispatch" {
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

func buildForceRedispatchCommand(phaseID int) string {
	return fmt.Sprintf("aether build %d --force", phaseID)
}

// buildUnfinishedRetryRedispatchCommand is the D-10 recovery command: a forced
// redispatch of the active phase RESTRICTED to the tasks that were never
// credited. `--force` supersedes the dead attempt; the `--task` filter is what
// makes the runtime plan only the named tasks, so no worker is ever sent to
// redo work another worker already proved (CR-04, 195-REVIEW.md -- the plain
// `--force` command this replaced re-planned the whole phase, credited tasks
// included, which is the opposite of what every surface promised the owner).
//
// With no task IDs it degrades to the plain forced redispatch, because there
// is nothing to narrow to.
func buildUnfinishedRetryRedispatchCommand(phaseID int, taskIDs []string) string {
	unique := uniqueSortedStrings(taskIDs)
	if len(unique) == 0 {
		return buildForceRedispatchCommand(phaseID)
	}
	var b strings.Builder
	fmt.Fprintf(&b, "aether build %d --force", phaseID)
	for _, taskID := range unique {
		b.WriteString(" --task ")
		b.WriteString(taskID)
	}
	return b.String()
}

func buildSkipPhaseCommand(phaseID int) string {
	return fmt.Sprintf("aether skip-phase %d --force", phaseID)
}

func continueNextCommandForBlocked(assessment codexContinueAssessment, blockers []string, options codexContinueOptions, phaseID int) string {
	lastOptions := loadLastContinueOptions(phaseID)

	if continueBlockersContainVerificationTimeout(blockers) {
		cmd := buildContinueVerificationTimeoutRecoveryCommand(options, lastOptions)
		// If the generated command matches current options exactly, fall back to build --force (D-10).
		if continueOptionsMatchCurrent(options, lastOptions) {
			return buildForceRedispatchCommand(phaseID)
		}
		return cmd
	}
	if continueBlockersContainWorkerTimeout(blockers) {
		cmd := buildContinueTimeoutRecoveryCommand(options, lastOptions)
		if continueOptionsMatchCurrent(options, lastOptions) {
			return buildForceRedispatchCommand(phaseID)
		}
		return cmd
	}
	if continueBlockersContainWatcherFailure(blockers) && !options.SkipWatchers {
		return fmt.Sprintf("aether continue --skip-watchers%s", buildContinueReconcileFlagSuffix(options.ReconcileTaskIDs))
	}
	next := strings.TrimSpace(continueNextCommandForAssessment(assessment))
	if next == "aether continue" && len(blockers) > 0 {
		return "" // Don't suggest looping back to continue with blockers (D-08 preserved).
	}
	return next
}

// resolveHostBoundaryWatcher decides the watcher verdict when continue would
// auto-skip its own watcher for a wrapper-mediated build. If the build packet
// already carries a real watcher's passing terminal result, that result is
// trusted — otherwise criteria with a `watcher` check could never pass on the
// primary wrapper path, where continue never spawns its own watcher.
func resolveHostBoundaryWatcher(buildWatcher codexWatcherVerification, skipSummary string) codexWatcherVerification {
	if buildWatcher.Present && buildWatcher.Passed && !strings.EqualFold(strings.TrimSpace(buildWatcher.Status), "skipped") {
		return buildWatcher
	}
	return codexWatcherVerification{Present: true, Passed: true, Status: "skipped", Worker: "auto-skip", Summary: skipSummary}
}

func continueWatcherHostBoundarySkipSummary(manifest codexContinueManifest) (string, bool) {
	if manifestUsesExternalTask(manifest) {
		return "watcher auto-skipped; external-task build was wrapper-mediated, so continue trusts runtime verification instead of spawning an untracked watcher", true
	}
	if codex.ShouldUseAgentDelegatePath() {
		return fmt.Sprintf("watcher auto-skipped; %s", codex.AgentDelegateFallbackReason()), true
	}
	return "", false
}

func buildContinueReconcileFlagSuffix(taskIDs []string) string {
	if len(taskIDs) == 0 {
		return ""
	}
	var b strings.Builder
	for _, taskID := range uniqueSortedStrings(taskIDs) {
		b.WriteString(" --reconcile-task ")
		b.WriteString(taskID)
	}
	return b.String()
}

func continueBlockersContainWatcherFailure(blockers []string) bool {
	for _, blocker := range blockers {
		lower := strings.ToLower(strings.TrimSpace(blocker))
		if lower == "" {
			continue
		}
		if strings.Contains(lower, "watcher") &&
			(strings.Contains(lower, "failed") || strings.Contains(lower, "did not complete") || strings.Contains(lower, "timed out")) {
			return true
		}
	}
	return false
}

func continueNextCommandForAssessment(assessment codexContinueAssessment) string {
	if strings.TrimSpace(assessment.Recovery.RedispatchCommand) != "" {
		return assessment.Recovery.RedispatchCommand
	}
	if strings.TrimSpace(assessment.Recovery.ReconcileCommand) != "" {
		return assessment.Recovery.ReconcileCommand
	}
	if continueAssessmentPrefersReverify(assessment) && strings.TrimSpace(assessment.Recovery.ReverifyCommand) != "" {
		return assessment.Recovery.ReverifyCommand
	}
	if strings.TrimSpace(assessment.Recovery.ReverifyCommand) != "" {
		return assessment.Recovery.ReverifyCommand
	}
	return "aether continue"
}

func continueBlockersContainVerificationTimeout(blockers []string) bool {
	for _, blocker := range blockers {
		lower := strings.ToLower(strings.TrimSpace(blocker))
		if strings.Contains(lower, "verification command timed out") || strings.Contains(lower, "--verification-timeout") {
			return true
		}
	}
	return false
}

func continueBlockersContainWorkerTimeout(blockers []string) bool {
	for _, blocker := range blockers {
		lower := strings.ToLower(strings.TrimSpace(blocker))
		if lower == "" {
			continue
		}
		if (strings.Contains(lower, "worker timeout") || strings.Contains(lower, "timed out") || strings.Contains(lower, "timeout")) &&
			(strings.Contains(lower, "watcher") || strings.Contains(lower, "worker") || strings.Contains(lower, "verification")) {
			return true
		}
	}
	return false
}

func buildContinueVerificationTimeoutRecoveryCommand(options codexContinueOptions, lastOptions *codexContinueOptionsJSON) string {
	timeout := recommendedContinueVerificationRecoveryTimeout(options.VerificationTimeout)
	// Ensure the suggested timeout differs from the last invocation.
	if lastOptions != nil && lastOptions.VerificationTimeoutSec > 0 {
		lastTimeout := time.Duration(lastOptions.VerificationTimeoutSec) * time.Second
		if timeout <= lastTimeout {
			timeout = lastTimeout * 2
		}
	}
	var b strings.Builder
	b.WriteString("aether continue --verification-timeout ")
	b.WriteString(formatDurationForCLI(timeout))
	for _, taskID := range uniqueSortedStrings(options.ReconcileTaskIDs) {
		b.WriteString(" --reconcile-task ")
		b.WriteString(taskID)
	}
	return b.String()
}

func buildContinueTimeoutRecoveryCommand(options codexContinueOptions, lastOptions *codexContinueOptionsJSON) string {
	timeout := recommendedContinueRecoveryTimeout(options.WorkerTimeout)
	// Ensure the suggested timeout differs from the last invocation.
	if lastOptions != nil && lastOptions.WorkerTimeoutSec > 0 {
		lastTimeout := time.Duration(lastOptions.WorkerTimeoutSec) * time.Second
		if timeout <= lastTimeout {
			timeout = lastTimeout * 2
		}
	}
	var b strings.Builder
	b.WriteString("aether continue --worker-timeout ")
	b.WriteString(formatDurationForCLI(timeout))
	for _, taskID := range uniqueSortedStrings(options.ReconcileTaskIDs) {
		b.WriteString(" --reconcile-task ")
		b.WriteString(taskID)
	}
	return b.String()
}

func recommendedContinueVerificationRecoveryTimeout(current time.Duration) time.Duration {
	effective := effectiveContinueVerificationTimeout(current)
	recommended := effective * 2
	minimum := 30 * time.Minute
	if recommended < minimum {
		return minimum
	}
	return recommended
}

func recommendedContinueRecoveryTimeout(current time.Duration) time.Duration {
	effective := effectiveContinueReviewTimeout(current)
	recommended := effective * 2
	minimum := 15 * time.Minute
	if recommended < minimum {
		return minimum
	}
	return recommended
}

func continueVerificationTimedOut(verification codexContinueVerificationReport) bool {
	for _, step := range verification.Steps {
		if step.TimedOut {
			return true
		}
	}
	return false
}

func formatDurationForCLI(duration time.Duration) string {
	if duration > 0 && duration%time.Hour == 0 {
		return fmt.Sprintf("%dh", int(duration/time.Hour))
	}
	if duration > 0 && duration%time.Minute == 0 {
		return fmt.Sprintf("%dm", int(duration/time.Minute))
	}
	return duration.String()
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
		// tsc without a tsconfig prints usage help and exits 1, failing
		// verification in every plain-JavaScript repo; only default to a
		// type check when the project actually configures TypeScript.
		if commands.Type == "" && (fileExists(filepath.Join(root, "tsconfig.json")) || fileExists(filepath.Join(root, "tsconfig.build.json"))) {
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
			// python -m pytest, not bare pytest: the console script does not
			// put the repo root on sys.path, so bare pytest fails every test
			// with ModuleNotFoundError in src-layout-less repos.
			commands.Test = "python3 -m pytest"
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
	section := extractMarkdownSection(content, heading)
	if section == "" {
		return extractVerificationCommands(content)
	}
	// The named section is the more precise signal, so it wins slot by slot
	// -- but it must never DELETE a command the rest of the file already
	// provided. Narrowing to the section discarded everything outside it, so
	// adding a "## Verification Commands" section naming only a build command
	// silently unresolved the test command the same file had been supplying
	// all along, and the phase then failed a check it had been passing.
	commands := extractVerificationCommands(section)
	mergeCodexVerificationCommands(&commands, extractVerificationCommands(content))
	return commands
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

// joinFencedLineContinuations folds a backslash-continued command inside a
// fenced code block into a single line before extractVerificationCommands
// parses it line by line. Without this pre-pass a documented command like
//
//	go test ./... \
//	  -race
//
// is silently reduced to its first fragment ("go test ./..."), because the
// per-line loop below has no notion of a command spanning multiple lines.
// Joining is restricted to inside fenced blocks (``` ... ```) — joining
// prose lines outside a fence would corrupt table and label parsing that
// depends on line boundaries (e.g. markdown tables, "Kind: command" labels).
func joinFencedLineContinuations(content string) []string {
	rawLines := strings.Split(content, "\n")
	result := make([]string, 0, len(rawLines))
	inFence := false
	i := 0
	for i < len(rawLines) {
		line := rawLines[i]
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			inFence = !inFence
			result = append(result, line)
			i++
			continue
		}
		if !inFence {
			result = append(result, line)
			i++
			continue
		}

		accumulated := strings.TrimRight(line, " \t")
		i++
		for strings.HasSuffix(accumulated, "\\") {
			accumulated = strings.TrimRight(strings.TrimSuffix(accumulated, "\\"), " \t")
			if i >= len(rawLines) {
				// Trailing backslash on the last line of the content: drop it
				// and stop, no further consumption possible.
				break
			}
			next := rawLines[i]
			if strings.HasPrefix(strings.TrimSpace(next), "```") {
				// Trailing backslash on the last line before a closing fence:
				// drop it and stop without consuming the fence delimiter — the
				// outer loop must still see it to toggle inFence.
				break
			}
			accumulated += " " + strings.TrimSpace(next)
			i++
		}
		result = append(result, accumulated)
	}
	return result
}

func extractVerificationCommands(content string) codexVerificationCommands {
	commands := codexVerificationCommands{}
	pendingKind := ""

	for _, rawLine := range joinFencedLineContinuations(content) {
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
			command := strings.TrimSpace(text[start+1 : start+1+end])
			if looksLikeVerificationCommand(command) {
				return command
			}
			return ""
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
		strings.HasPrefix(lower, "bun run build"),
		strings.HasPrefix(lower, "pnpm build"),
		strings.HasPrefix(lower, "yarn build"),
		strings.HasPrefix(lower, "cargo build"),
		strings.HasPrefix(lower, "python -m build"),
		strings.HasPrefix(lower, "make build"):
		return "build"
	case strings.HasPrefix(lower, "go test"),
		strings.HasPrefix(lower, "npm test"),
		strings.HasPrefix(lower, "bun test"),
		strings.HasPrefix(lower, "bun run test"),
		strings.HasPrefix(lower, "pnpm test"),
		strings.HasPrefix(lower, "yarn test"),
		strings.HasPrefix(lower, "cargo test"),
		strings.HasPrefix(lower, "pytest"),
		strings.HasPrefix(lower, "python -m pytest"),
		strings.HasPrefix(lower, "python3 -m pytest"),
		strings.HasPrefix(lower, "python -m unittest"),
		strings.HasPrefix(lower, "python3 -m unittest"),
		strings.HasPrefix(lower, "uv run pytest"),
		strings.HasPrefix(lower, "make test"):
		return "tests"
	case strings.HasPrefix(lower, "go vet"),
		strings.Contains(lower, "tsc --noemit"),
		strings.HasPrefix(lower, "pyright"),
		strings.HasPrefix(lower, "mypy"):
		return "types"
	case strings.HasPrefix(lower, "golangci-lint"),
		strings.HasPrefix(lower, "npm run lint"),
		strings.HasPrefix(lower, "bun run lint"),
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

// blockedVerificationConfigGuidance names every location a user can configure
// a real verification command. Phase 160 D-01 draws the gate-vs-enrichment
// line at "can this halt cleanly point somewhere actionable" — this string is
// that actionable pointer, shared by both blocked-return sites below so the
// three locations never drift apart.
func blockedVerificationConfigGuidance() string {
	// .aether/data/codebase.md is deliberately NOT offered here. It is read
	// as a source, but .aether/data is a protected path -- an agent following
	// this guidance to write there is refused by the write guard, which left
	// the operator alternating between a blocked halt and a blocked write
	// with no exit. Only the two paths anyone can actually edit are named.
	return `configure a real command in AGENTS.md, or in CLAUDE.md under "## Verification Commands" (one line per check, e.g. "- tests: go test ./...")`
}

// applyExpectedTestFailure inverts the tests check's expectation for a
// deliberately-RED phase (Phase.ExpectFailingTests): the phase's deliverable
// is failing tests that prove a defect, so a genuine test failure is the
// expected outcome and a green suite means the deliverable was not produced.
// Aether's own route-setter plans RED-first phases; before this existed the
// gate treated their defining artifact as a blocker and the phase could never
// advance (Pocket-Chopper field report). Only a genuine test failure is
// inverted — a timeout, an execution block, or an environment fault is not a
// failing test suite and keeps its ordinary failure semantics.
func applyExpectedTestFailure(steps []codexVerificationStep, phase colony.Phase) []codexVerificationStep {
	if !phase.ExpectFailingTests {
		return steps
	}
	for i := range steps {
		if steps[i].Name != "tests" || steps[i].Skipped {
			continue
		}
		if steps[i].Passed {
			steps[i].Passed = false
			steps[i].Summary = "expected failing tests — this phase's deliverable is tests that prove the defect, but the test run passed; write the failing test first (" + strings.TrimSpace(steps[i].Summary) + ")"
			continue
		}
		if steps[i].TimedOut || steps[i].Blocked || steps[i].ErrorClass == ErrorClassEnvironment {
			continue
		}
		steps[i].Passed = true
		steps[i].Summary = "tests failed as expected — this phase's deliverable is failing tests that prove the defect (" + strings.TrimSpace(steps[i].Summary) + ")"
	}
	return steps
}

func runVerificationStep(ctx context.Context, root, name string, required bool, command string, timeout time.Duration) codexVerificationStep {
	if strings.TrimSpace(command) == "" {
		if required {
			return codexVerificationStep{
				Name:     name,
				Skipped:  true,
				Blocked:  true,
				Required: true,
				Passed:   false,
				Summary:  fmt.Sprintf("blocked: no verification command resolved for %s; %s", name, blockedVerificationConfigGuidance()),
			}
		}
		return codexVerificationStep{
			Name:     name,
			Skipped:  true,
			Passed:   true,
			Required: false,
			Summary:  "no command resolved; skipped",
		}
	}

	timeout = effectiveContinueVerificationTimeout(timeout)
	output, exitCode, timedOut, err := runShellCommandContext(ctx, root, command, timeout)
	step := codexVerificationStep{
		Name:           name,
		Command:        command,
		Passed:         err == nil,
		Required:       required,
		TimedOut:       timedOut,
		TimeoutSeconds: int(timeout / time.Second),
		ExitCode:       exitCode,
		Summary:        successSummaryForStep(name, exitCode, output, err),
		Output:         output,
	}
	if err != nil {
		// A command that does not exist is not a failed verification — it is a
		// wrong guess by the language-fallback resolver. `npm run lint` in a
		// project with no lint script, or `make test` with no such target, used
		// to hard-block phase advancement with no remedy the user could see.
		// The absence of a tool proves nothing about the code; classify it as
		// Skipped and let the watcher carry verification — unless this check is
		// required by the phase's own criteria, in which case reporting a pass
		// here would directly contradict evaluateCriterionCheck's gate below.
		if isCommandUnresolvable(output, exitCode) {
			if required {
				return codexVerificationStep{
					Name:     name,
					Command:  command,
					Skipped:  true,
					Blocked:  true,
					Required: true,
					Passed:   false,
					ExitCode: exitCode,
					Summary:  fmt.Sprintf("blocked: verification command %q for %s could not run (exit %d); %s", command, name, exitCode, blockedVerificationConfigGuidance()),
				}
			}
			return codexVerificationStep{
				Name:     name,
				Command:  command,
				Skipped:  true,
				Passed:   true,
				Required: false,
				ExitCode: exitCode,
				Summary:  fmt.Sprintf("%s: command unavailable in this repository (%s); skipped — configure a real command in CLAUDE.md to enable this check", name, command),
			}
		}
		step.Summary = failureSummaryForStep(name, exitCode, output, err, timedOut, timeout)
		if timedOut {
			step.ErrorClass = ErrorClassTimeout
		} else {
			step.ErrorClass = classifyVerificationError(output, exitCode)
		}
	}
	return step
}

// requiredVerificationChecks derives the set of shell verification checks
// ("build", "types", "lint", "tests") that at least one of the phase's bound
// criterion requirements names. A check absent from this set is enrichment
// (Phase 160 D-01) and keeps warning on skip; a check present in it is a gate
// and must halt loudly when it cannot run.
func requiredVerificationChecks(phase colony.Phase) map[string]bool {
	required := map[string]bool{}
	for _, requirement := range flattenPhaseCriterionEvidenceRequirements(phase) {
		for _, check := range requirement.Checks {
			check = strings.ToLower(strings.TrimSpace(check))
			if check == "" {
				continue
			}
			required[check] = true
		}
	}
	return required
}

// isCommandUnresolvable reports whether a verification failure means the
// command itself cannot run in this repository, as opposed to the code failing
// the check. Exit 127 is the shell's universal command-not-found; the string
// patterns cover the per-ecosystem equivalents of "no such script/target".
func isCommandUnresolvable(output string, exitCode int) bool {
	if exitCode == 127 {
		return true
	}
	lower := strings.ToLower(output)
	for _, marker := range []string{
		"command not found",
		"executable file not found",
		"missing script:",                                      // npm
		"npm error missing script",                             // npm >= 10 phrasing
		"could not determine executable to run",                // npx
		"no rule to make target",                               // make
		"no such command",                                      // cargo
		"unknown command",                                      // misc CLIs
		"is not recognized as an internal or external command", // windows
	} {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
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

	// The union of everything the claims file says changed this phase — the
	// single notion of "what changed" that both mismatch checking above and
	// checkAntiPatternGate below consume. Populated on every return path from
	// this point on, since claims were successfully loaded.
	scannedFiles := []string{}
	for _, rel := range append(append(append([]string{}, claims.FilesCreated...), claims.FilesModified...), claims.TestsWritten...) {
		rel = strings.TrimSpace(rel)
		if rel == "" {
			continue
		}
		scannedFiles = append(scannedFiles, rel)
	}

	if manifest.Present && manifest.Data.Phase > 0 && claims.BuildPhase != manifest.Data.Phase {
		return codexClaimVerification{
			Present:      true,
			Passed:       false,
			Summary:      fmt.Sprintf("builder claims build_phase %d does not match manifest phase %d; run `%s` to regenerate claims before `aether continue`", claims.BuildPhase, manifest.Data.Phase, buildForceRedispatchCommand(manifest.Data.Phase)),
			ScannedFiles: scannedFiles,
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
		fullPath := filepath.Join(root, filepath.FromSlash(rel))
		if !fileExists(fullPath) {
			if resolved := findRepoRelativePath(root, rel); resolved != "" && fileExists(filepath.Join(root, filepath.FromSlash(resolved))) {
				continue // defense-in-depth for pre-fix claims
			}
			mismatches = append(mismatches, fmt.Sprintf("%s does not exist", rel))
		}
	}
	if len(mismatches) > 0 {
		return codexClaimVerification{
			Present:      true,
			Passed:       false,
			Summary:      fmt.Sprintf("worker claims mismatch: %d missing paths", len(mismatches)),
			Checked:      checked,
			Mismatches:   mismatches,
			ScannedFiles: scannedFiles,
		}
	}

	if checked == 0 && manifestRequiresBuilderClaims(manifest) {
		if manifestUsesSyntheticDispatch(manifest) {
			return codexClaimVerification{
				Present:      true,
				Passed:       false,
				Summary:      "builder claims file is empty because the build ran in simulated mode; rerun `aether build <phase>` without `--synthetic` before `aether continue` can advance",
				Checked:      0,
				ScannedFiles: scannedFiles,
			}
		}
		if manifestUsesExternalTask(manifest) {
			return codexClaimVerification{
				Present:      true,
				Passed:       true,
				Summary:      "builder claims file is empty (external-task mode); verification-led truth applies",
				Checked:      0,
				ScannedFiles: scannedFiles,
			}
		}
		return codexClaimVerification{
			Present:      true,
			Passed:       false,
			Summary:      emptyClaimsFailureSummary(manifest),
			Checked:      0,
			ScannedFiles: scannedFiles,
		}
	}

	summary := "builder claims verified"
	if checked == 0 {
		summary = "builder claims file present but empty"
	}
	return codexClaimVerification{
		Present:      true,
		Passed:       true,
		Summary:      summary,
		Checked:      checked,
		ScannedFiles: scannedFiles,
	}
}

func runCodexContinueGates(phase colony.Phase, manifest codexContinueManifest, verification codexContinueVerificationReport, assessment codexContinueAssessment, now time.Time, priorGateResults []GateCheckResult) codexContinueGateReport {
	checks := []gateCheck{}
	blockers := []string{}
	warnings := []string{}

	// Circuit breaker integration (LOOP-01): check if any gate has exceeded retry threshold
	for _, prior := range priorGateResults {
		if prior.Status == "failed" {
			key := gateRetryKey(phase.ID, prior.Name)
			if circuitBreaker != nil && !circuitBreaker.Allow(key) {
				return codexContinueGateReport{
					Phase:          phase.ID,
					GeneratedAt:    now.Format(time.RFC3339),
					Checks:         checks,
					Passed:         false,
					BlockingIssues: []string{fmt.Sprintf("circuit breaker tripped for gate %q after %d failed retries -- manual intervention required", prior.Name, circuitBreaker.FailureCount(key))},
				}
			}
		}
	}

	// manifest_present gate
	if shouldSkipGate(priorGateResults, "manifest_present") {
		checks = append(checks, gateCheck{Name: "manifest_present", Passed: true, Detail: "skipped: previously passed"})
	} else {
		manifestCheck := gateCheck{Name: "manifest_present", Passed: manifest.Present, Detail: "build manifest present"}
		if !manifest.Present {
			manifestCheck.Detail = fmt.Sprintf("build manifest is missing for phase %d", phase.ID)
			manifestCheck.FixHint = "Ensure the build completed successfully and produced a manifest.json"
			manifestCheck.RecoveryOptions = []string{
				"Fix the build issue and run aether continue",
				"Run aether unblock --dispatch for guided recovery",
			}
			blockers = append(blockers, manifestCheck.Detail)
		}
		checks = append(checks, manifestCheck)
	}

	// verification_steps_passed gate
	if shouldSkipGate(priorGateResults, "verification_steps_passed") {
		checks = append(checks, gateCheck{Name: "verification_steps_passed", Passed: true, Detail: "skipped: previously passed"})
	} else {
		verifCheck := gateCheck{
			Name:   "verification_steps_passed",
			Passed: verification.ChecksPassed,
			Detail: continueVerificationDetail(verification),
		}
		if !verification.ChecksPassed {
			verifCheck.FixHint = gateRecoveryTemplate("verification_loop")
			verifCheck.RecoveryOptions = []string{
				"Fix manually and run aether continue",
				"Run aether unblock --dispatch for guided recovery",
			}
			blockers = append(blockers, verification.BlockingIssues...)
		}
		checks = append(checks, verifCheck)
	}

	// implementation_evidence gate
	if shouldSkipGate(priorGateResults, "implementation_evidence") {
		checks = append(checks, gateCheck{Name: "implementation_evidence", Passed: true, Detail: "skipped: previously passed"})
	} else {
		evidenceCheck := gateCheck{Name: "implementation_evidence", Passed: assessment.PositiveEvidence, Detail: "task or claim evidence recorded for the verified phase"}
		if !assessment.PositiveEvidence {
			evidenceCheck.Detail = "verification passed but no implementation evidence or reconciliation was recorded"
			evidenceCheck.FixHint = "Ensure workers reported task completion or claims were filed"
			evidenceCheck.RecoveryOptions = []string{
				"Fix manually and run aether continue",
				"Run aether unblock --dispatch for guided recovery",
			}
			blockers = append(blockers, assessment.BlockingIssues...)
		}
		checks = append(checks, evidenceCheck)
	}

	// owner_confirmation_pending gate (D-05, 193-CONTEXT.md): surfaces every
	// outstanding needs_owner_confirmation criterion without blocking
	// continue -- the phase still advances (D-05: "the phase advances;
	// aether seal blocks until the owner has confirmed it"). No worker is
	// dispatched because of this gate. The ONE exception: when this phase is
	// the plan's LAST phase, advancing here and sealing are the same act (an
	// unconfirmed criterion could otherwise ride straight through to a
	// completed colony with nobody ever asked), so the gate MUST fail here.
	// A gate that can never fail on the one boundary where it matters is
	// worse than none (the operational_evidence precedent immediately
	// below).
	ownerPending := outstandingOwnerConfirmations(phase.ID, verification.Criteria)
	ownerCheck := gateCheck{Name: "owner_confirmation_pending", Passed: true, Detail: "nothing is waiting on your confirmation"}
	if len(ownerPending) > 0 {
		details := make([]string, 0, len(ownerPending))
		recovery := make([]string, 0, len(ownerPending))
		for _, c := range ownerPending {
			details = append(details, fmt.Sprintf("%q%s", c.Criterion, criterionTaskSuffix(c.TaskID)))
			recovery = append(recovery, ownerConfirmationCommand(phase.ID, c.TaskID, c.Criterion))
		}
		ownerCheck.Detail = fmt.Sprintf("%d requirement(s) could not be checked automatically or by a reviewer and need your confirmation: %s", len(ownerPending), strings.Join(details, "; "))
		ownerCheck.RecoveryOptions = recovery
		if isLastPhaseOfActivePlan(phase.ID) {
			ownerCheck.Passed = false
			ownerCheck.FixHint = "Confirm each item above with the aether decision-answer command shown, then run aether continue again"
			blockers = append(blockers, ownerCheck.Detail)
		}
	}
	checks = append(checks, ownerCheck)

	// check_fix_attempt gate (D-02, D-03): when a failing check with no
	// reviewer dispatched drew the single bounded automatic builder fix
	// attempt and the re-run still failed, this gate fails and names the
	// exact command to re-run the builder by hand -- more specific than the
	// generic verification_steps_passed gate above, which already blocks for
	// the same underlying reason. nil CheckFixAttempt (no attempt ran)
	// carries no gate entry at all -- there is nothing to report.
	if fix := verification.CheckFixAttempt; fix != nil {
		// WR-02 (193-REVIEW.md): fix.Outcome is one of two internal
		// snake_case enum values ("fixed"/"still_failing", set in
		// check_fix_attempt.go). CLAUDE.md requires every string this
		// program writes to the (explicitly non-technical) project owner
		// to be plain English with no raw code-style tokens, so this
		// translates the enum before it ever reaches Detail rather than
		// interpolating it directly.
		outcomeText := "fixed it"
		if fix.Outcome == "still_failing" {
			outcomeText = "the check is still failing"
		}
		fixCheck := gateCheck{
			Name:   "check_fix_attempt",
			Passed: fix.Outcome != "still_failing",
			Detail: fmt.Sprintf("one automatic fix attempt ran for the %s check — %s", fix.Check, outcomeText),
		}
		if !fixCheck.Passed {
			fixCommand := buildTargetedRedispatchCommand(phase.ID, fix.FailureIndex.ImplicatedTaskIDs)
			if strings.TrimSpace(fixCommand) == "" {
				fixCommand = buildForceRedispatchCommand(phase.ID)
			}
			fixCheck.FixHint = fmt.Sprintf("The automatic fix attempt did not resolve the %s check; run this command by hand", fix.Check)
			fixCheck.RecoveryOptions = []string{fixCommand}
			blockers = append(blockers, fixCheck.Detail)
		}
		checks = append(checks, fixCheck)
	}

	// The operational_evidence gate was removed: it hardcoded Passed=true
	// regardless of assessment.OperationalIssues, so it was a gate that could
	// not gate. An always-pass check is worse than no check — it reads as
	// assurance while asserting nothing, which is the exact failure mode the
	// Definition of Done exists to prevent. Operational issues still surface as
	// warnings below; genuine operational evidence now arrives structurally via
	// mandatory worker handoffs (changed_files, commands_run,
	// verification_status), which the build finalizer rejects when empty.
	if len(assessment.OperationalIssues) > 0 {
		warnings = append(warnings, fmt.Sprintf("%d operational worker issues recorded; review worker output", len(assessment.OperationalIssues)))
	}

	// flags gate (no_critical_flags) — runs every time for safety
	flagCheck := checkNoCriticalFlags()
	if !flagCheck.Passed {
		flagCheck.FixHint = "Resolve critical flags before continuing"
		flagCheck.RecoveryOptions = []string{
			"Fix the issue, then resolve its flag: aether flag-resolve --id <id> --message \"what fixed it\"",
			"Run aether unblock --dispatch to dispatch the Fixer against the blocking issues",
			"Fix manually and run aether continue",
		}
		blockers = append(blockers, flagCheck.Detail)
	}
	checks = append(checks, flagCheck)

	// The Iron Law gate (classic Flags Gate): no phase advancement with
	// unresolved blockers. Advancement-scoped only — build is allowed with
	// an open blocker; passing this line is not.
	blockerFlagCheck := checkUnresolvedBlockerFlags()
	if !blockerFlagCheck.Passed {
		blockerFlagCheck.FixHint = "Every blocker must be resolved before the phase can advance"
		blockerFlagCheck.RecoveryOptions = []string{
			"Fix the issue, then resolve its flag: aether flag-resolve --id <id> --message \"what fixed it\"",
			"Run aether unblock --dispatch to dispatch the Fixer against the blocking issues",
		}
		blockers = append(blockers, blockerFlagCheck.Detail)
	}
	checks = append(checks, blockerFlagCheck)

	// anti_pattern / anti_pattern_executed gates — the live caller for the
	// security gate that RESEARCH.md found had no live caller (T-160-01).
	// anti_pattern_executed is in alwaysRunGates, so shouldSkipGate already
	// returns false for it; only the findings gate participates in skip logic.
	antiPatternCheck, antiPatternExecutedCheck := checkAntiPatternGate(verification.Claims.ScannedFiles)
	if shouldSkipGate(priorGateResults, "anti_pattern") {
		checks = append(checks, gateCheck{Name: "anti_pattern", Passed: true, Detail: "skipped: previously passed"})
	} else {
		if !antiPatternCheck.Passed {
			blockers = append(blockers, antiPatternCheck.Detail)
		}
		checks = append(checks, antiPatternCheck)
	}
	if !antiPatternExecutedCheck.Passed {
		blockers = append(blockers, antiPatternExecutedCheck.Detail)
	}
	checks = append(checks, antiPatternExecutedCheck)

	// charter_compliance / charter_compliance_executed gates — the live
	// caller for CONTEXT-06/D-09: Aether's recurring defect is producers
	// with no callers (18 of 25 milestones have been framed around
	// restoring something previously marked done), and this comment is
	// what makes a future reader check the call site is still here.
	// charter_compliance_executed is in alwaysRunGates, so shouldSkipGate
	// already returns false for it; only the findings gate participates in
	// skip logic.
	charterComplianceCheck, charterComplianceExecutedCheck := checkCharterComplianceGate(verification.Steps)
	if shouldSkipGate(priorGateResults, "charter_compliance") {
		checks = append(checks, gateCheck{Name: "charter_compliance", Passed: true, Detail: "skipped: previously passed"})
	} else {
		if !charterComplianceCheck.Passed {
			blockers = append(blockers, charterComplianceCheck.Detail)
		}
		checks = append(checks, charterComplianceCheck)
	}
	if !charterComplianceExecutedCheck.Passed {
		blockers = append(blockers, charterComplianceExecutedCheck.Detail)
	}
	checks = append(checks, charterComplianceExecutedCheck)

	// Record failures/successes in circuit breaker (LOOP-01)
	for _, c := range checks {
		key := gateRetryKey(phase.ID, c.Name)
		if !c.Passed {
			if circuitBreaker != nil {
				circuitBreaker.RecordFailure(key)
			}
		} else {
			if circuitBreaker != nil {
				circuitBreaker.RecordSuccess(key)
			}
		}
	}

	blockingIssues := uniqueSortedStrings(append(blockers, assessment.BlockingIssues...))
	return codexContinueGateReport{
		Phase:          phase.ID,
		GeneratedAt:    now.Format(time.RFC3339),
		Checks:         checks,
		Passed:         len(blockingIssues) == 0,
		BlockingIssues: blockingIssues,
		Warnings:       warnings,
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
			// A worker missing from the spawn tree is not a state failure:
			// the tree is a coordination/display artifact that pause/resume
			// cycles and fresh sessions rotate. The build packet remains the
			// truth for worker results — closing an untracked worker is a
			// no-op, not a reason to abort the lifecycle.
			if strings.Contains(err.Error(), "not found") {
				fmt.Fprintf(os.Stderr, "note: worker %s absent from spawn tree; closure skipped (%v)\n", detail.Name, err)
				continue
			}
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

func continueWorkerFlowForVerification(verification codexContinueVerificationReport, flow []codexContinueWorkerFlowStep, watcherFlow *codexContinueWorkerFlowStep) []codexContinueWorkerFlowStep {
	combined := []codexContinueWorkerFlowStep{continueDeterministicVerificationFlowStep(verification)}
	if watcherFlow != nil && strings.TrimSpace(watcherFlow.Name) != "" {
		combined = append(combined, *watcherFlow)
	} else if skipped, ok := continueSkippedWatcherFlowStep(verification.Watcher); ok {
		combined = append(combined, skipped)
	}
	return append(combined, flow...)
}

func continueReviewWorkerFlowSteps(flow []codexContinueWorkerFlowStep) []codexContinueWorkerFlowStep {
	review := make([]codexContinueWorkerFlowStep, 0, len(flow))
	for _, step := range flow {
		if strings.TrimSpace(step.Stage) == "review" {
			review = append(review, step)
		}
	}
	return review
}

func continueDeterministicVerificationFlowStep(verification codexContinueVerificationReport) codexContinueWorkerFlowStep {
	status := "completed"
	if !verification.ChecksPassed {
		status = "failed"
	} else if continueVerificationAllStepsSkipped(verification.Steps) {
		status = "skipped"
	}
	return codexContinueWorkerFlowStep{
		Stage:   "verification",
		Caste:   "system",
		Name:    "Deterministic verification",
		Task:    "Run deterministic verification commands before review workers",
		Status:  status,
		Summary: continueDeterministicVerificationSummary(verification),
	}
}

func continueSkippedWatcherFlowStep(watcher codexWatcherVerification) (codexContinueWorkerFlowStep, bool) {
	if !watcher.Present || continueWorkerFlowStatus(watcher.Status) != "skipped" {
		return codexContinueWorkerFlowStep{}, false
	}
	summary := strings.TrimSpace(watcher.Summary)
	if summary == "" {
		summary = "continue watcher skipped; relying on deterministic verification"
	}
	return codexContinueWorkerFlowStep{
		Stage:   "verification",
		Caste:   "system",
		Name:    "Continue watcher",
		Task:    "Record why the continue watcher did not run",
		Status:  "skipped",
		Summary: summary,
	}, true
}

func continueReviewSkippedFlowStep(summary string) codexContinueWorkerFlowStep {
	summary = strings.TrimSpace(summary)
	if summary == "" {
		summary = "review wave skipped; no review workers were required"
	}
	return codexContinueWorkerFlowStep{
		Stage:   "review",
		Caste:   "system",
		Name:    "Review wave",
		Task:    "Record why review workers did not run",
		Status:  "skipped",
		Summary: summary,
	}
}

func continueReviewSkippedSummary(reviewDepth colony.VerificationDepth) string {
	depth := colony.NormalizeVerificationDepth(string(reviewDepth))
	switch depth {
	case colony.VerificationDepthLight:
		return "review wave skipped; light verification depth does not require review workers"
	default:
		return fmt.Sprintf("review wave skipped; no %s review workers were selected", depth)
	}
}

func continueVerificationAllStepsSkipped(steps []codexVerificationStep) bool {
	if len(steps) == 0 {
		return false
	}
	for _, step := range steps {
		if !step.Skipped {
			return false
		}
	}
	return true
}

func continueDeterministicVerificationSummary(verification codexContinueVerificationReport) string {
	passed := 0
	skipped := 0
	failed := 0
	for _, step := range verification.Steps {
		switch {
		case step.Skipped:
			skipped++
		case step.Passed:
			passed++
		default:
			failed++
		}
	}
	if len(verification.Steps) == 0 {
		if verification.ChecksPassed {
			return "deterministic verification completed; no command steps were recorded"
		}
		return "deterministic verification failed; no command steps were recorded"
	}
	if failed > 0 && verification.ChecksPassed {
		return fmt.Sprintf("deterministic verification completed with warnings: %d passed, %d failed, %d skipped", passed, failed, skipped)
	}
	if failed > 0 {
		return fmt.Sprintf("deterministic verification failed: %d passed, %d failed, %d skipped", passed, failed, skipped)
	}
	if skipped == len(verification.Steps) {
		return fmt.Sprintf("deterministic verification skipped: %d commands skipped", skipped)
	}
	return fmt.Sprintf("deterministic verification completed: %d passed, %d skipped", passed, skipped)
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

// continueLearningFlowStep is the learning-stage analog of
// continueHousekeepingFlowStep (D-06): it folds the phase-end consolidation
// result into the same worker-flow / ceremony-event stream housekeeping
// already uses, so the 🧠 learning beat reaches
// emitContinueCeremonyFlowSequence's event stream, not only stdout.
// Summary is LearningBeatLine() -- the exact same one-line text
// renderLearningBeat prints -- so the two surfaces cannot drift apart.
func continueLearningFlowStep(s phaseEndConsolidationSummary) codexContinueWorkerFlowStep {
	status := "completed"
	if !s.Ran {
		status = "failed"
	}
	return codexContinueWorkerFlowStep{
		Stage:   "learning",
		Caste:   "librarian",
		Name:    "Phase-end consolidation",
		Status:  status,
		Summary: s.LearningBeatLine(),
	}
}

func continueWorkerFlowStatus(status string) string {
	status = strings.TrimSpace(status)
	if status == "" {
		return "completed"
	}
	return status
}

func continueReviewStatusBlocks(status string) bool {
	status = continueWorkerFlowStatus(status)
	// A reviewer that honestly reports completed_no_change verified the work
	// and found nothing to change (ruling D6). Blocking advancement on it
	// would punish the honest answer and force a fabricated finding --
	// interrupted still blocks, because that work is genuinely unfinished.
	if isSuccessfulExternalBuildStatus(status) {
		return false
	}
	switch status {
	case "skipped", watcherStatusEnvironmentBlocked:
		return false
	default:
		return true
	}
}

func continueReviewStepBlocks(step codexContinueWorkerFlowStep, verification codexContinueVerificationReport) bool {
	status := continueWorkerFlowStatus(step.Status)
	if status == "timeout" && verification.ChecksPassed {
		return false
	}
	return continueReviewStatusBlocks(status)
}

func continueWorkerFlowEnvironmentBlocked(step codexContinueWorkerFlowStep) bool {
	parts := []string{step.Summary}
	parts = append(parts, step.Blockers...)
	return isEnvironmentBlockedLaunchVerification(strings.Join(parts, "\n"))
}

func continueWorkerFlowIsDeterministicVerification(step codexContinueWorkerFlowStep) bool {
	return strings.TrimSpace(step.Stage) == "verification" &&
		strings.EqualFold(strings.TrimSpace(step.Caste), "system") &&
		strings.EqualFold(strings.TrimSpace(step.Name), "Deterministic verification")
}

func emitContinueVerificationStart(phase colony.Phase, verificationTimeout time.Duration) {
	if !shouldRenderVisualOutput(stdout) {
		return
	}
	timeout := effectiveContinueVerificationTimeout(verificationTimeout).Round(time.Second)
	writeVisualOutput(stdout, fmt.Sprintf(
		"Running deterministic verification for phase %d before watcher/review workers (timeout %s per command).\n",
		phase.ID,
		timeout,
	))
}

func continueWatcherDefaultSummary(status string) string {
	switch {
	case isSuccessfulExternalBuildStatus(continueWorkerFlowStatus(status)):
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
	switch {
	case isSuccessfulExternalBuildStatus(status):
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
				if continueWorkerFlowIsDeterministicVerification(step) {
					summary = "Deterministic verification completed"
				} else {
					summary = "Watcher verification completed"
				}
			}
			if continueWorkerFlowIsDeterministicVerification(step) {
				events = append(events, fmt.Sprintf("%s|deterministic_verification|continue|%s", now.Format(time.RFC3339), summary))
				continue
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

	var updated colony.ColonyState
	if err := store.UpdateJSONAtomically("COLONY_STATE.json", &updated, func() error {
		if err := validateRuntimeStateStillCurrent(updated, state.CurrentPhase, state.BuildStartedAt, colony.StateEXECUTING, colony.StateBUILT); err != nil {
			return err
		}
		updated.Events = append(trimmedEvents(updated.Events), continueWorkerFlowEvents(now, workerFlow)...)
		updated.Events = append(updated.Events, fmt.Sprintf("%s|continue_blocked|continue|Continue blocked before advancement", now.Format(time.RFC3339)))
		return nil
	}); err != nil {
		return state, fmt.Errorf("failed to save colony state: %w", err)
	}
	return updated, nil
}

func appendRuntimeStateEventsIfCurrent(expected colony.ColonyState, events []string) error {
	if len(events) == 0 || store == nil {
		return nil
	}
	var updated colony.ColonyState
	return store.UpdateJSONAtomically("COLONY_STATE.json", &updated, func() error {
		if err := validateRuntimeStateMatchesExpected(updated, expected); err != nil {
			return err
		}
		updated.Events = append(trimmedEvents(updated.Events), events...)
		return nil
	})
}

func continueSupersededResult(startState colony.ColonyState, phase colony.Phase, err error) map[string]interface{} {
	latest := startState
	if loaded, loadErr := loadActiveColonyState(); loadErr == nil {
		latest = loaded
	}
	summary := "Runtime command was superseded by newer colony state; no state changes were written."
	if err != nil {
		summary = strings.TrimSpace(err.Error())
	}
	next := nextCommandFromState(latest)
	if strings.TrimSpace(next) == "" {
		next = "aether status"
	}
	return map[string]interface{}{
		"advanced":        false,
		"blocked":         true,
		"superseded":      true,
		"current_phase":   latest.CurrentPhase,
		"phase_name":      phase.Name,
		"state":           latest.State,
		"next":            next,
		"blocking_issues": []string{summary},
	}
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
	case "measurer":
		return "Measurer continue review"
	case "chaos":
		return "Chaos continue review"
	case "includer":
		return "Includer continue review"
	case "keeper":
		return "Keeper continue review"
	case "sage":
		return "Sage continue review"
	case "medic":
		return "Medic continue review"
	case "fixer":
		return "Fixer continue review"
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
	switch {
	case isSuccessfulExternalBuildStatus(status):
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

func runShellCommand(root, command string, timeout time.Duration) (string, int, bool, error) {
	return runShellCommandContext(context.Background(), root, command, timeout)
}

func runShellCommandContext(parent context.Context, root, command string, timeout time.Duration) (string, int, bool, error) {
	if parent == nil {
		parent = context.Background()
	}
	ctx, cancel := context.WithTimeout(parent, timeout)
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
	return trimmed, exitCode, ctx.Err() == context.DeadlineExceeded, err
}

func trimCommandOutput(output string) string {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) <= 20 {
		return strings.TrimSpace(output)
	}
	return strings.Join(lines[len(lines)-20:], "\n")
}

// successSummaryForStep composes the evidence line for a PASSING
// verification step. WR-01 (191.1-REVIEW.md): this used to return a bare,
// hardcoded per-check-name constant ("tests passed", "build succeeded", ...)
// for every passing run -- honest about the outcome, but carrying nothing
// that varies per run, so a downstream reader could not tell "this run
// passed" from "the check named tests always says tests passed". This now
// mirrors failureSummaryForStep's own, already-correct pattern for a failing
// run: the base outcome plus the real exit code and the command's own
// trailing output line, when one exists.
func successSummaryForStep(name string, exitCode int, output string, err error) string {
	if err != nil {
		return fmt.Sprintf("%s failed", name)
	}
	base := ""
	switch name {
	case "build":
		base = "build succeeded"
	case "types":
		base = "type checks passed"
	case "lint":
		base = "lint passed"
	case "tests":
		base = "tests passed"
	default:
		base = fmt.Sprintf("%s passed", name)
	}
	if trimmed := strings.TrimSpace(output); trimmed != "" {
		lines := strings.Split(trimmed, "\n")
		if last := strings.TrimSpace(lines[len(lines)-1]); last != "" {
			return fmt.Sprintf("%s (exit %d): %s", base, exitCode, last)
		}
	}
	return fmt.Sprintf("%s (exit %d)", base, exitCode)
}

// classifyVerificationError inspects command output and exit code to determine
// whether a failure is a product issue or an environment issue (missing DB,
// port conflicts, permission errors, etc.).
func classifyVerificationError(output string, exitCode int) VerificationErrorClass {
	lower := strings.ToLower(output)
	envPatterns := []string{
		"eperm", "eacces", "permission denied",
		"eaddrinuse", "address already in use", "listen tcp", "bind: address already in use",
		"econnrefused", "connection refused", "dial tcp", "connect: connection refused",
		"postgres", "psql", "database", "connect",
		"no such file or directory", "command not found",
		"module not found", "cannot find module",
	}
	for _, p := range envPatterns {
		if strings.Contains(lower, p) {
			return ErrorClassEnvironment
		}
	}
	return ErrorClassProduct
}

func isEnvironmentBlockedLaunchVerification(text string) bool {
	lower := strings.ToLower(text)
	if strings.TrimSpace(lower) == "" {
		return false
	}
	hasPermissionDenial := strings.Contains(lower, "eperm") ||
		strings.Contains(lower, "eacces") ||
		strings.Contains(lower, "permission denied")
	if !hasPermissionDenial {
		return false
	}
	hasSocketBind := strings.Contains(lower, "listen") ||
		strings.Contains(lower, "bind") ||
		strings.Contains(lower, "socket")
	if !hasSocketBind {
		return false
	}
	hasRawPreflight := strings.Contains(lower, "raw bind") ||
		strings.Contains(lower, "raw local socket") ||
		strings.Contains(lower, "raw socket") ||
		strings.Contains(lower, "raw node") ||
		strings.Contains(lower, "minimal socket") ||
		strings.Contains(lower, "socket preflight") ||
		strings.Contains(lower, "node http bind")
	if !hasRawPreflight {
		return false
	}
	return true
}

func failureSummaryForStep(name string, exitCode int, output string, err error, timedOut bool, timeout time.Duration) string {
	if timedOut {
		return fmt.Sprintf("verification command timed out after %s; increase with --verification-timeout or narrow the repo verification command", formatDurationForCLI(timeout))
	}
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

func manifestUsesExternalTask(manifest codexContinueManifest) bool {
	if !manifest.Present {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(manifest.Data.DispatchMode), "external-task")
}

func missingClaimsSummary(manifest codexContinueManifest) string {
	if manifestRequiresBuilderClaims(manifest) {
		return "no builder claims file found for a phase that dispatched builders"
	}
	return "no builder claims file found; skipped"
}
