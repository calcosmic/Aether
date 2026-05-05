package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/calcosmic/Aether/pkg/agent"
	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/spf13/cobra"
)

const (
	defaultSwarmWorkerTimeout = 6 * time.Minute
	defaultSwarmRunTimeout    = 30 * time.Minute
)

var newSwarmWorkerInvoker = codex.NewWorkerInvoker

type swarmWorkerPlan struct {
	Stage            string                 `json:"stage,omitempty"`
	Wave             int                    `json:"wave"`
	Name             string                 `json:"name"`
	Caste            string                 `json:"caste"`
	Role             string                 `json:"role"`
	Task             string                 `json:"task"`
	TaskID           string                 `json:"task_id,omitempty"`
	AgentName        string                 `json:"agent_name"`
	Brief            string                 `json:"brief,omitempty"`
	OutputPaths      []string               `json:"output_paths,omitempty"`
	ResponseContract map[string]interface{} `json:"response_contract,omitempty"`
	TimeoutSeconds   int                    `json:"timeout_seconds,omitempty"`
	Timeout          time.Duration          `json:"-"`
}

type swarmWorkerResponse struct {
	Role           string   `json:"role"`
	Status         string   `json:"status"`
	Summary        string   `json:"summary"`
	Findings       []string `json:"findings,omitempty"`
	Evidence       []string `json:"evidence,omitempty"`
	RootCause      string   `json:"root_cause,omitempty"`
	Recommendation string   `json:"recommendation,omitempty"`
	ProposedFix    string   `json:"proposed_fix,omitempty"`
	FilesTouched   []string `json:"files_touched,omitempty"`
	TestsWritten   []string `json:"tests_written,omitempty"`
	Verification   []string `json:"verification,omitempty"`
}

type swarmWorkerExecution struct {
	Name         string              `json:"name"`
	Caste        string              `json:"caste"`
	Role         string              `json:"role"`
	Task         string              `json:"task"`
	Status       string              `json:"status"`
	Summary      string              `json:"summary"`
	Duration     float64             `json:"duration,omitempty"`
	Files        []string            `json:"files,omitempty"`
	Tests        []string            `json:"tests,omitempty"`
	Blockers     []string            `json:"blockers,omitempty"`
	Response     swarmWorkerResponse `json:"response,omitempty"`
	Claims       *codex.WorkerResult `json:"-"`
	ResponsePath string              `json:"response_path,omitempty"`
}

type swarmManifest struct {
	Workflow             string                   `json:"workflow"`
	DispatchMode         string                   `json:"dispatch_mode"`
	RequiresFinalizer    bool                     `json:"requires_finalizer"`
	GeneratedAt          string                   `json:"generated_at"`
	Root                 string                   `json:"root"`
	SwarmID              string                   `json:"swarm_id"`
	Target               string                   `json:"target"`
	WaveCount            int                      `json:"wave_count"`
	WorkerCount          int                      `json:"worker_count"`
	RunTimeoutSeconds    int                      `json:"run_timeout_seconds"`
	WorkerTimeoutSeconds int                      `json:"worker_timeout_seconds"`
	DispatchContract     map[string]interface{}   `json:"dispatch_contract"`
	Dispatches           []swarmWorkerPlan        `json:"dispatches"`
	ExecutionPlan        []map[string]interface{} `json:"execution_plan"`
	FinalizerCommand     string                   `json:"finalizer_command"`
}

type externalSwarmCompletion struct {
	SwarmManifest *swarmManifest         `json:"swarm_manifest,omitempty"`
	Manifest      *swarmManifest         `json:"manifest,omitempty"`
	Dispatches    []swarmWorkerExecution `json:"dispatches,omitempty"`
	Workers       []swarmWorkerExecution `json:"workers,omitempty"`
	Results       []swarmWorkerExecution `json:"results,omitempty"`
	Responses     []swarmWorkerResponse  `json:"responses,omitempty"`
	WorkerResults []codex.WorkerResult   `json:"worker_results,omitempty"`
}

var swarmCmd = &cobra.Command{
	Use:   "swarm [problem]",
	Short: "Launch the Aether swarm bug-destroyer or watch live colony activity",
	Args:  cobra.ArbitraryArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		watch, _ := cmd.Flags().GetBool("watch")
		planOnly, _ := cmd.Flags().GetBool("plan-only")
		target := strings.TrimSpace(strings.Join(args, " "))
		root := resolveAetherRootPath()

		result, err := runSwarmCompatibility(root, target, watch, planOnly)
		if err != nil {
			outputError(1, err.Error(), nil)
			return nil
		}
		outputWorkflow(result, renderSwarmCompatibilityVisual(result))
		return nil
	},
}

var swarmFinalizeCmd = &cobra.Command{
	Use:   "swarm-finalize",
	Short: "Record externally spawned swarm workers and write the swarm result",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		completionPath, _ := cmd.Flags().GetString("completion-file")
		completion, err := loadExternalSwarmCompletion(completionPath)
		if err != nil {
			outputError(1, err.Error(), nil)
			return nil
		}
		result, err := runSwarmFinalize(resolveAetherRootPath(), completion)
		if err != nil {
			outputError(1, err.Error(), nil)
			return nil
		}
		outputWorkflow(result, renderSwarmCompatibilityVisual(result))
		return nil
	},
}

func init() {
	swarmCmd.Flags().Bool("watch", false, "Show live colony activity instead of launching a swarm run")
	swarmCmd.Flags().Bool("plan-only", false, "Print a host-dispatch swarm manifest without running workers")
	rootCmd.AddCommand(swarmCmd)
	swarmFinalizeCmd.Flags().String("completion-file", "", "JSON file containing the swarm manifest and external worker results")
	rootCmd.AddCommand(swarmFinalizeCmd)
}

func runSwarmCompatibility(root, target string, watch, planOnly bool) (map[string]interface{}, error) {
	if watch || strings.TrimSpace(target) == "" {
		if planOnly && strings.TrimSpace(target) == "" && !watch {
			return nil, fmt.Errorf("swarm --plan-only requires a problem description")
		}
		return buildSwarmWatchResult(target, watch, false), nil
	}
	if planOnly || codex.ShouldUseAgentDelegatePath() {
		return runSwarmPlanOnly(root, target)
	}
	return runSwarmDestroy(root, target)
}

func buildSwarmWatchResult(target string, watch, liveRefresh bool) map[string]interface{} {
	state, _ := loadColonyState()
	spawnSummary := loadSpawnActivitySummaryForState(store, state)
	active := spawnSummary.ActiveEntries
	recent := spawnSummary.RecentOutcomeEntries

	next := `aether init "describe the goal"`
	phaseName := ""
	stateName := ""
	goal := ""
	scope := "project"
	if state != nil {
		next = nextCommandFromState(*state)
		phaseName = lookupPhaseName(*state, state.CurrentPhase)
		stateName = string(state.State)
		scope = string(state.EffectiveScope())
		if state.Goal != nil {
			goal = strings.TrimSpace(*state.Goal)
		}
		if strings.TrimSpace(next) == "" {
			next = "aether status"
		}
	}
	recoverySummary := ""
	recoveryCommand := ""
	continueReport := ""
	if state != nil {
		if guidance := loadActiveRecoveryGuidance(*state); guidance != nil {
			recoverySummary = guidance.Summary
			recoveryCommand = guidance.Next
			continueReport = guidance.ReportPath
		}
	}

	workers := spawnEntriesToWatchMaps(active)
	recentWorkers := spawnEntriesToWatchMaps(recent)

	return map[string]interface{}{
		"mode":                "watch",
		"target":              target,
		"autopilot_available": true,
		"goal":                goal,
		"scope":               scope,
		"state":               stateName,
		"phase_name":          phaseName,
		"current_run_id":      spawnSummary.CurrentRunID,
		"current_run_command": spawnSummary.CurrentCommand,
		"active_workers":      workers,
		"recent_workers":      recentWorkers,
		"active_count":        len(workers),
		"recent_count":        len(recentWorkers),
		"completed_count":     spawnSummary.CompletedCount,
		"blocked_count":       spawnSummary.BlockedCount,
		"failed_count":        spawnSummary.FailedCount,
		"live_refresh":        liveRefresh,
		"next":                next,
		"recovery_summary":    recoverySummary,
		"recovery_command":    recoveryCommand,
		"continue_report":     continueReport,
		"watch":               watch || target == "",
	}
}

func spawnEntriesToWatchMaps(entries []agent.SpawnEntry) []map[string]interface{} {
	workers := make([]map[string]interface{}, 0, len(entries))
	for _, entry := range entries {
		timestamp := entry.ActivityTimestamp
		if strings.TrimSpace(timestamp) == "" {
			timestamp = entry.Timestamp
		}
		workers = append(workers, map[string]interface{}{
			"name":      entry.AgentName,
			"caste":     entry.Caste,
			"task":      entry.Task,
			"status":    entry.Status,
			"summary":   entry.Summary,
			"timestamp": timestamp,
		})
	}
	return workers
}

func runSwarmDestroy(root, target string) (map[string]interface{}, error) {
	invoker := newSwarmWorkerInvoker()
	if invoker == nil {
		return nil, fmt.Errorf("swarm worker invoker is not configured")
	}
	parentCtx, stopSignals := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stopSignals()
	ctx, cancel := context.WithTimeout(parentCtx, defaultSwarmRunTimeout)
	defer cancel()

	if !invoker.IsAvailable(ctx) {
		return nil, dispatchUnavailableError(invoker)
	}

	state, _ := loadColonyState()
	startedAt := time.Now().UTC()
	runHandle, err := beginRuntimeSpawnRun("swarm", startedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize swarm run: %w", err)
	}
	runStatus := "failed"
	defer func() {
		finishRuntimeSpawnRun(runHandle, runStatus, time.Now().UTC())
	}()

	swarmID := fmt.Sprintf("swarm-%d", startedAt.Unix())
	if err := initializeSwarmRun(swarmID); err != nil {
		return nil, fmt.Errorf("initialize swarm workspace: %w", err)
	}

	investigation := buildSwarmInvestigationPlans(root, target)
	emitVisualProgress(renderSwarmDispatchPreview(swarmID, target, investigation, "Investigation Wave"))
	investigationRuns, err := executeSwarmWave(ctx, root, swarmID, target, investigation, "", invoker)
	if err != nil {
		if ctx.Err() != nil {
			runStatus = "timeout"
			return nil, fmt.Errorf("swarm stopped: %w", ctx.Err())
		}
		return nil, err
	}

	findingSummary := renderSwarmFindingSummary(investigationRuns)
	builderPlan := buildSwarmBuilderPlan(root, target)
	emitVisualProgress(renderSwarmDispatchPreview(swarmID, target, []swarmWorkerPlan{builderPlan}, "Fix Wave"))
	builderRuns, err := executeSwarmWave(ctx, root, swarmID, target, []swarmWorkerPlan{builderPlan}, findingSummary, invoker)
	if err != nil {
		if ctx.Err() != nil {
			runStatus = "timeout"
			return nil, fmt.Errorf("swarm stopped: %w", ctx.Err())
		}
		return nil, err
	}

	builderSummary := renderSwarmFindingSummary(builderRuns)
	watcherPlan := buildSwarmWatcherPlan(root, target)
	emitVisualProgress(renderSwarmDispatchPreview(swarmID, target, []swarmWorkerPlan{watcherPlan}, "Verification Wave"))
	watcherRuns, err := executeSwarmWave(ctx, root, swarmID, target, []swarmWorkerPlan{watcherPlan}, findingSummary+"\n\n"+builderSummary, invoker)
	if err != nil {
		if ctx.Err() != nil {
			runStatus = "timeout"
			return nil, fmt.Errorf("swarm stopped: %w", ctx.Err())
		}
		return nil, err
	}

	allRuns := append(append([]swarmWorkerExecution{}, investigationRuns...), builderRuns...)
	allRuns = append(allRuns, watcherRuns...)

	status, recommendation, rootCause, solution, blockers := summarizeSwarmOutcome(allRuns)
	runStatus = summarizeRunStatus(status)
	filesTouched, testsWritten := collectSwarmTouchedFiles(allRuns)
	next := swarmNextCommand(state, status)

	_ = store.SaveJSON(filepath.ToSlash(filepath.Join("swarms", swarmID, "result.json")), map[string]interface{}{
		"swarm_id":       swarmID,
		"target":         target,
		"status":         status,
		"root_cause":     rootCause,
		"solution":       solution,
		"recommendation": recommendation,
		"workers":        allRuns,
		"files":          filesTouched,
		"tests":          testsWritten,
		"blockers":       blockers,
		"completed_at":   time.Now().UTC().Format(time.RFC3339),
	})

	return map[string]interface{}{
		"mode":                "destroy",
		"autopilot_available": true,
		"swarm_id":            swarmID,
		"target":              target,
		"status":              status,
		"root_cause":          rootCause,
		"solution":            solution,
		"recommendation":      recommendation,
		"workers":             swarmExecutionsForJSON(allRuns),
		"worker_count":        len(allRuns),
		"files_touched":       filesTouched,
		"tests_written":       testsWritten,
		"blockers":            blockers,
		"next":                next,
		"watch":               false,
	}, nil
}

func runSwarmPlanOnly(root, target string) (map[string]interface{}, error) {
	if store == nil {
		return nil, fmt.Errorf("no store initialized")
	}
	target = strings.TrimSpace(target)
	if target == "" {
		return nil, fmt.Errorf("swarm --plan-only requires a problem description")
	}

	dispatchMode := "plan-only"
	status := "plan-only"
	if codex.ShouldUseAgentDelegatePath() {
		dispatchMode = "agent-delegate"
		status = "agent-delegate"
	}

	manifest := buildSwarmManifest(root, target, dispatchMode, time.Now().UTC())
	dispatchMaps := swarmPlanMaps(manifest.Dispatches)
	return map[string]interface{}{
		"mode":                  "destroy",
		"status":                status,
		"dispatch_mode":         dispatchMode,
		"requires_finalizer":    true,
		"execution_owner":       "host-platform",
		"agent_delegate":        dispatchMode == "agent-delegate",
		"agent_delegate_reason": strings.TrimSpace(codex.AgentDelegateFallbackReason()),
		"swarm_id":              manifest.SwarmID,
		"target":                target,
		"root":                  root,
		"swarm_manifest":        manifest,
		"dispatch_manifest":     manifest,
		"dispatches":            dispatchMaps,
		"workers":               dispatchMaps,
		"worker_count":          len(dispatchMaps),
		"wave_count":            manifest.WaveCount,
		"dispatch_contract":     manifest.DispatchContract,
		"finalizer_command":     manifest.FinalizerCommand,
		"next":                  "dispatch host swarm workers, then run `aether swarm-finalize --completion-file <file>`",
		"watch":                 false,
	}, nil
}

func buildSwarmManifest(root, target, dispatchMode string, now time.Time) swarmManifest {
	swarmID := fmt.Sprintf("swarm-%d", now.Unix())
	dispatches := allSwarmPlans(root, target)
	for i := range dispatches {
		dispatches[i] = enrichSwarmPlanForManifest(root, target, swarmID, dispatches[i])
	}
	return swarmManifest{
		Workflow:             "swarm",
		DispatchMode:         dispatchMode,
		RequiresFinalizer:    true,
		GeneratedAt:          now.Format(time.RFC3339),
		Root:                 root,
		SwarmID:              swarmID,
		Target:               strings.TrimSpace(target),
		WaveCount:            3,
		WorkerCount:          len(dispatches),
		RunTimeoutSeconds:    int(defaultSwarmRunTimeout / time.Second),
		WorkerTimeoutSeconds: int(defaultSwarmWorkerTimeout / time.Second),
		DispatchContract: map[string]interface{}{
			"execution_model":        "3 waves: investigation, fix, verification",
			"wave_count":             3,
			"worker_count":           len(dispatches),
			"coordination_path":      ".aether/data/spawn-tree.txt",
			"artifact_path":          filepath.ToSlash(filepath.Join(".aether", "data", "swarms", swarmID, "result.json")),
			"completion_shape":       "completion JSON must include swarm_manifest plus dispatches/workers/results",
			"state_authority":        "runtime finalizer writes swarm artifacts and spawn-tree status",
			"wrapper_write_policy":   "workers report structured terminal results to the wrapper; wrappers do not hand-edit .aether/data",
			"run_timeout_seconds":    int(defaultSwarmRunTimeout / time.Second),
			"worker_status_values":   []string{"completed", "passed", "code_written", "blocked", "failed", "timeout"},
			"required_result_fields": []string{"name", "caste", "role", "task", "status", "summary"},
		},
		Dispatches:       dispatches,
		ExecutionPlan:    swarmPlanMaps(dispatches),
		FinalizerCommand: "AETHER_OUTPUT_MODE=json aether swarm-finalize --completion-file <file>",
	}
}

func allSwarmPlans(root, target string) []swarmWorkerPlan {
	plans := append([]swarmWorkerPlan{}, buildSwarmInvestigationPlans(root, target)...)
	builder := buildSwarmBuilderPlan(root, target)
	watcher := buildSwarmWatcherPlan(root, target)
	plans = append(plans, builder, watcher)
	return plans
}

func enrichSwarmPlanForManifest(root, target, swarmID string, plan swarmWorkerPlan) swarmWorkerPlan {
	plan.Stage = swarmStageName(plan.Wave)
	if strings.TrimSpace(plan.TaskID) == "" {
		plan.TaskID = fmt.Sprintf("swarm.%s", plan.Role)
	}
	if plan.TimeoutSeconds == 0 {
		plan.TimeoutSeconds = int(firstSwarmTimeout(plan.Timeout) / time.Second)
	}
	if len(plan.OutputPaths) == 0 {
		plan.OutputPaths = []string{
			filepath.ToSlash(filepath.Join(".aether", "external", "swarm", swarmID, plan.Name+".json")),
		}
	}
	if plan.ResponseContract == nil {
		plan.ResponseContract = map[string]interface{}{
			"format": "terminal structured result in wrapper completion JSON",
			"fields": []string{
				"name", "caste", "role", "task", "status", "summary",
				"files", "tests", "blockers", "response",
			},
			"response_fields": []string{
				"role", "status", "summary", "findings", "evidence",
				"root_cause", "recommendation", "proposed_fix",
				"files_touched", "tests_written", "verification",
			},
		}
	}
	if strings.TrimSpace(plan.Brief) == "" {
		plan.Brief = renderExternalSwarmWorkerBrief(root, target, swarmID, plan)
	}
	return plan
}

func swarmStageName(wave int) string {
	switch wave {
	case 1:
		return "investigation"
	case 2:
		return "fix"
	case 3:
		return "verification"
	default:
		return "swarm"
	}
}

func renderExternalSwarmWorkerBrief(root, target, swarmID string, plan swarmWorkerPlan) string {
	var b strings.Builder
	b.WriteString("Swarm ID: " + swarmID + "\n")
	b.WriteString("Target: " + strings.TrimSpace(target) + "\n")
	b.WriteString("Workspace: " + root + "\n")
	b.WriteString("Role: " + plan.Role + "\n")
	b.WriteString("Wave: " + fmt.Sprintf("%d", plan.Wave) + " (" + swarmStageName(plan.Wave) + ")\n\n")
	b.WriteString("Assignment:\n")
	b.WriteString(plan.Task)
	b.WriteString("\n\nReturn a terminal structured result to the wrapper. Do not hand-edit `.aether/data/`; the wrapper will pass your result to `aether swarm-finalize`.\n")
	b.WriteString("\nRequired result fields: name, caste, role, task, status, summary. Include files/tests/blockers/response when relevant.\n")
	b.WriteString("Builder may edit code. Tracker, Scout, Archaeologist, and Watcher should stay read-only unless the user explicitly asked otherwise.\n")
	return strings.TrimSpace(b.String())
}

func swarmPlanMaps(plans []swarmWorkerPlan) []map[string]interface{} {
	out := make([]map[string]interface{}, 0, len(plans))
	for _, plan := range plans {
		entry := map[string]interface{}{
			"stage":           plan.Stage,
			"wave":            plan.Wave,
			"name":            plan.Name,
			"caste":           plan.Caste,
			"role":            plan.Role,
			"task":            plan.Task,
			"task_id":         plan.TaskID,
			"agent_name":      plan.AgentName,
			"brief":           plan.Brief,
			"output_paths":    plan.OutputPaths,
			"status":          "planned",
			"timeout_seconds": plan.TimeoutSeconds,
		}
		if plan.ResponseContract != nil {
			entry["response_contract"] = plan.ResponseContract
		}
		out = append(out, entry)
	}
	return out
}

func initializeSwarmRun(swarmID string) error {
	if err := os.MkdirAll(filepath.Join(store.BasePath(), "swarms", swarmID, "responses"), 0755); err != nil {
		return err
	}
	if err := store.SaveJSON(filepath.ToSlash(filepath.Join("swarms", swarmID, "findings.json")), swarmFindingsFile{
		SwarmID:  swarmID,
		Findings: []swarmFinding{},
	}); err != nil {
		return err
	}
	if err := store.SaveJSON(filepath.ToSlash(filepath.Join("swarms", swarmID, "display.json")), swarmDisplayFile{
		SwarmID: swarmID,
		Agents:  []swarmAgentStatus{},
	}); err != nil {
		return err
	}
	return store.SaveJSON(filepath.ToSlash(filepath.Join("swarms", swarmID, "timing.json")), swarmTimingFile{
		SwarmID: swarmID,
		StartAt: time.Now().UTC().Format(time.RFC3339),
	})
}

func loadExternalSwarmCompletion(path string) (externalSwarmCompletion, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return externalSwarmCompletion{}, fmt.Errorf("flag --completion-file is required")
	}
	var data []byte
	var err error
	if path == "-" {
		data, err = io.ReadAll(os.Stdin)
	} else {
		data, err = os.ReadFile(path)
	}
	if err != nil {
		return externalSwarmCompletion{}, fmt.Errorf("read completion file: %w", err)
	}

	var completion externalSwarmCompletion
	if err := json.Unmarshal(data, &completion); err != nil {
		return externalSwarmCompletion{}, fmt.Errorf("parse completion file: %w", err)
	}
	if completion.activeManifest() != nil {
		return completion, nil
	}

	var envelope struct {
		Result externalSwarmCompletion `json:"result"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		return externalSwarmCompletion{}, fmt.Errorf("parse completion envelope: %w", err)
	}
	if envelope.Result.activeManifest() == nil {
		return externalSwarmCompletion{}, fmt.Errorf("completion file must include swarm_manifest")
	}
	return envelope.Result, nil
}

func (c externalSwarmCompletion) activeManifest() *swarmManifest {
	if c.SwarmManifest != nil {
		return c.SwarmManifest
	}
	return c.Manifest
}

func (c externalSwarmCompletion) workerResults() []swarmWorkerExecution {
	results := make([]swarmWorkerExecution, 0, len(c.Dispatches)+len(c.Workers)+len(c.Results)+len(c.Responses)+len(c.WorkerResults))
	results = append(results, c.Dispatches...)
	results = append(results, c.Workers...)
	results = append(results, c.Results...)
	for _, response := range c.Responses {
		results = append(results, swarmWorkerExecution{
			Role:     response.Role,
			Caste:    response.Role,
			Status:   response.Status,
			Summary:  response.Summary,
			Files:    append([]string{}, response.FilesTouched...),
			Tests:    append([]string{}, response.TestsWritten...),
			Response: response,
		})
	}
	for _, workerResult := range c.WorkerResults {
		results = append(results, swarmWorkerExecution{
			Name:     workerResult.WorkerName,
			Caste:    workerResult.Caste,
			Status:   workerResult.Status,
			Summary:  workerResult.Summary,
			Duration: workerResult.Duration.Seconds(),
			Files:    append(append([]string{}, workerResult.FilesCreated...), workerResult.FilesModified...),
			Tests:    append([]string{}, workerResult.TestsWritten...),
			Blockers: append([]string{}, workerResult.Blockers...),
		})
	}
	return results
}

func runSwarmFinalize(root string, completion externalSwarmCompletion) (map[string]interface{}, error) {
	if store == nil {
		return nil, fmt.Errorf("no store initialized")
	}
	manifest := completion.activeManifest()
	if manifest == nil {
		return nil, fmt.Errorf("completion file must include swarm_manifest")
	}
	if (manifest.DispatchMode != "plan-only" && manifest.DispatchMode != "agent-delegate") || !manifest.RequiresFinalizer {
		return nil, fmt.Errorf("swarm_manifest must come from `aether swarm --plan-only` or an agent-delegate swarm response")
	}
	if len(manifest.Dispatches) == 0 {
		return nil, fmt.Errorf("swarm_manifest contains no dispatches")
	}
	if strings.TrimSpace(manifest.Root) != "" && !sameCleanPath(manifest.Root, root) {
		return nil, fmt.Errorf("swarm_manifest root does not match current workspace (manifest=%s current=%s)", manifest.Root, root)
	}

	state, _ := loadColonyState()
	startedAt := time.Now().UTC()
	runHandle, err := beginRuntimeSpawnRun("swarm", startedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize swarm run: %w", err)
	}
	runStatus := "failed"
	defer func() {
		finishRuntimeSpawnRun(runHandle, runStatus, time.Now().UTC())
	}()

	swarmID := strings.TrimSpace(manifest.SwarmID)
	if swarmID == "" {
		swarmID = fmt.Sprintf("swarm-%d", startedAt.Unix())
	}
	if err := initializeSwarmRun(swarmID); err != nil {
		return nil, fmt.Errorf("initialize swarm workspace: %w", err)
	}

	runs, err := mergeExternalSwarmResults(*manifest, completion.workerResults())
	if err != nil {
		return nil, err
	}
	if err := recordExternalSwarmRun(swarmID, runs); err != nil {
		return nil, err
	}

	status, recommendation, rootCause, solution, blockers := summarizeSwarmOutcome(runs)
	runStatus = summarizeRunStatus(status)
	filesTouched, testsWritten := collectSwarmTouchedFiles(runs)
	next := swarmNextCommand(state, status)

	if err := store.SaveJSON(filepath.ToSlash(filepath.Join("swarms", swarmID, "result.json")), map[string]interface{}{
		"swarm_id":       swarmID,
		"target":         manifest.Target,
		"status":         status,
		"root_cause":     rootCause,
		"solution":       solution,
		"recommendation": recommendation,
		"workers":        runs,
		"files":          filesTouched,
		"tests":          testsWritten,
		"blockers":       blockers,
		"completed_at":   time.Now().UTC().Format(time.RFC3339),
		"dispatch_mode":  "external-task",
	}); err != nil {
		return nil, fmt.Errorf("write swarm result: %w", err)
	}

	return map[string]interface{}{
		"mode":                "destroy",
		"autopilot_available": true,
		"swarm_id":            swarmID,
		"target":              manifest.Target,
		"status":              status,
		"root_cause":          rootCause,
		"solution":            solution,
		"recommendation":      recommendation,
		"workers":             swarmExecutionsForJSON(runs),
		"dispatches":          swarmExecutionsForJSON(runs),
		"worker_count":        len(runs),
		"wave_count":          manifest.WaveCount,
		"files_touched":       filesTouched,
		"tests_written":       testsWritten,
		"blockers":            blockers,
		"dispatch_mode":       "external-task",
		"dispatch_contract":   manifest.DispatchContract,
		"next":                next,
		"watch":               false,
	}, nil
}

func mergeExternalSwarmResults(manifest swarmManifest, results []swarmWorkerExecution) ([]swarmWorkerExecution, error) {
	resultByName := make(map[string]swarmWorkerExecution, len(results))
	resultByRole := make(map[string]swarmWorkerExecution, len(results))
	for _, result := range results {
		if name := strings.TrimSpace(result.Name); name != "" {
			resultByName[name] = result
		}
		if role := strings.TrimSpace(result.Role); role != "" {
			resultByRole[role] = result
		}
		if result.Response.Role != "" {
			resultByRole[result.Response.Role] = result
		}
	}

	merged := make([]swarmWorkerExecution, 0, len(manifest.Dispatches))
	for _, plan := range manifest.Dispatches {
		result, ok := resultByName[strings.TrimSpace(plan.Name)]
		if !ok {
			result, ok = resultByRole[strings.TrimSpace(plan.Role)]
		}
		if !ok {
			return nil, fmt.Errorf("missing external swarm worker result for %s", plan.Name)
		}

		execution := swarmWorkerExecution{
			Name:         plan.Name,
			Caste:        plan.Caste,
			Role:         plan.Role,
			Task:         plan.Task,
			Status:       normalizeRuntimeDispatchStatus(result.Status),
			Summary:      strings.TrimSpace(result.Summary),
			Duration:     result.Duration,
			Files:        append([]string{}, result.Files...),
			Tests:        append([]string{}, result.Tests...),
			Blockers:     append([]string{}, result.Blockers...),
			Response:     result.Response,
			ResponsePath: result.ResponsePath,
		}
		if strings.TrimSpace(result.Name) != "" {
			execution.Name = strings.TrimSpace(result.Name)
		}
		if strings.TrimSpace(result.Caste) != "" {
			execution.Caste = strings.TrimSpace(result.Caste)
		}
		if strings.TrimSpace(result.Role) != "" {
			execution.Role = strings.TrimSpace(result.Role)
		}
		if strings.TrimSpace(result.Task) != "" {
			execution.Task = strings.TrimSpace(result.Task)
		}
		if execution.Response.Role == "" {
			execution.Response.Role = execution.Role
		}
		if execution.Status == "" || execution.Status == "spawned" {
			execution.Status = "completed"
		}
		if execution.Summary == "" && execution.Response.Summary != "" {
			execution.Summary = execution.Response.Summary
		}
		if execution.Summary == "" && len(execution.Blockers) > 0 {
			execution.Summary = strings.Join(execution.Blockers, "; ")
		}
		if execution.Summary == "" {
			execution.Summary = fmt.Sprintf("%s worker completed without a structured summary.", execution.Role)
		}
		execution.Files = append(execution.Files, execution.Response.FilesTouched...)
		execution.Tests = append(execution.Tests, execution.Response.TestsWritten...)
		execution.Files = swarmCompactStrings(execution.Files)
		execution.Tests = swarmCompactStrings(execution.Tests)
		execution.Blockers = swarmCompactStrings(execution.Blockers)
		merged = append(merged, execution)
	}
	return merged, nil
}

func recordExternalSwarmRun(swarmID string, runs []swarmWorkerExecution) error {
	spawnTree := agent.NewSpawnTree(store, "spawn-tree.txt")
	for _, run := range runs {
		wave := 1
		switch run.Role {
		case "builder":
			wave = 2
		case "watcher":
			wave = 3
		}
		if err := spawnTree.RecordSpawn("Swarm", run.Caste, run.Name, run.Task, wave); err != nil {
			return fmt.Errorf("record swarm spawn %s: %w", run.Name, err)
		}
		summary := strings.TrimSpace(run.Summary)
		if summary == "" {
			summary = fmt.Sprintf("%s worker finished with status %s", run.Role, run.Status)
		}
		if err := spawnTree.UpdateStatus(run.Name, run.Status, summary); err != nil {
			return fmt.Errorf("complete swarm worker %s: %w", run.Name, err)
		}
		if err := updateSwarmDisplayStatus(swarmID, run.Name, run.Status); err != nil {
			return fmt.Errorf("update swarm display %s: %w", run.Name, err)
		}
		response := run.Response
		if err := recordSwarmFinding(swarmID, run.Name, &response, run); err != nil {
			return fmt.Errorf("record swarm finding %s: %w", run.Name, err)
		}
	}
	return nil
}

func buildSwarmInvestigationPlans(root, target string) []swarmWorkerPlan {
	seed := strings.ToLower(strings.TrimSpace(target))
	return []swarmWorkerPlan{
		{
			Name:      deterministicAntName("tracker", root+"|swarm|tracker|"+seed),
			Caste:     "tracker",
			Role:      "tracker",
			Task:      "Reproduce the issue, trace the failure path, and identify the most likely root cause.",
			AgentName: "aether-tracker",
			Wave:      1,
			Timeout:   defaultSwarmWorkerTimeout,
		},
		{
			Name:      deterministicAntName("scout", root+"|swarm|scout|"+seed),
			Caste:     "scout",
			Role:      "scout",
			Task:      "Search the repo for the most relevant files, patterns, tests, and documentation tied to the reported bug.",
			AgentName: "aether-scout",
			Wave:      1,
			Timeout:   defaultSwarmWorkerTimeout,
		},
		{
			Name:      deterministicAntName("archaeologist", root+"|swarm|archaeology|"+seed),
			Caste:     "archaeologist",
			Role:      "archaeologist",
			Task:      "Inspect git history and prior fixes around the bug area to identify historical context, fragile zones, and regressions.",
			AgentName: "aether-archaeologist",
			Wave:      1,
			Timeout:   defaultSwarmWorkerTimeout,
		},
	}
}

func buildSwarmBuilderPlan(root, target string) swarmWorkerPlan {
	return swarmWorkerPlan{
		Name:      deterministicAntName("builder", root+"|swarm|builder|"+strings.ToLower(strings.TrimSpace(target))),
		Caste:     "builder",
		Role:      "builder",
		Task:      "Implement the smallest safe fix for the reported bug and add or update tests that prove the regression is covered.",
		AgentName: "aether-builder",
		Wave:      2,
		Timeout:   8 * time.Minute,
	}
}

func buildSwarmWatcherPlan(root, target string) swarmWorkerPlan {
	return swarmWorkerPlan{
		Name:      deterministicAntName("watcher", root+"|swarm|watcher|"+strings.ToLower(strings.TrimSpace(target))),
		Caste:     "watcher",
		Role:      "watcher",
		Task:      "Verify the fix independently, run the most relevant checks, and confirm whether the bug is actually resolved.",
		AgentName: "aether-watcher",
		Wave:      3,
		Timeout:   defaultSwarmWorkerTimeout,
	}
}

func executeSwarmWave(ctx context.Context, root, swarmID, target string, plans []swarmWorkerPlan, priorSummary string, invoker codex.WorkerInvoker) ([]swarmWorkerExecution, error) {
	spawnTree := agent.NewSpawnTree(store, "spawn-tree.txt")
	runs := make([]swarmWorkerExecution, 0, len(plans))
	for _, plan := range plans {
		if err := ctx.Err(); err != nil {
			return runs, err
		}
		if err := spawnTree.RecordSpawn("Swarm", plan.Caste, plan.Name, plan.Task, plan.Wave); err != nil {
			return nil, fmt.Errorf("record swarm spawn %s: %w", plan.Name, err)
		}
		if err := spawnTree.UpdateStatus(plan.Name, "active", "swarm worker active"); err != nil {
			return nil, fmt.Errorf("mark swarm worker active %s: %w", plan.Name, err)
		}
		if err := updateSwarmDisplayStatus(swarmID, plan.Name, "active"); err != nil {
			return nil, fmt.Errorf("update swarm display %s: %w", plan.Name, err)
		}

		responsePath := swarmResponsePath(swarmID, plan.Name)
		result, response, execErr := invokeSwarmWorker(ctx, root, target, swarmID, plan, priorSummary, responsePath, invoker)

		execution := swarmWorkerExecution{
			Name:         plan.Name,
			Caste:        plan.Caste,
			Role:         plan.Role,
			Task:         plan.Task,
			ResponsePath: filepath.ToSlash(responsePath),
		}
		if result != nil {
			execution.Claims = result
			execution.Status = strings.TrimSpace(result.Status)
			execution.Summary = strings.TrimSpace(result.Summary)
			execution.Duration = result.Duration.Seconds()
			execution.Blockers = append([]string{}, result.Blockers...)
			execution.Files = append([]string{}, result.FilesCreated...)
			execution.Files = append(execution.Files, result.FilesModified...)
			execution.Tests = append([]string{}, result.TestsWritten...)
		}
		if response != nil {
			execution.Response = *response
			if execution.Status == "" {
				execution.Status = response.Status
			}
			if execution.Summary == "" {
				execution.Summary = response.Summary
			}
			execution.Files = append(execution.Files, response.FilesTouched...)
			execution.Tests = append(execution.Tests, response.TestsWritten...)
		}
		execution.Files = swarmCompactStrings(execution.Files)
		execution.Tests = swarmCompactStrings(execution.Tests)
		if execution.Status == "" {
			execution.Status = "failed"
		}
		if execErr != nil {
			execution.Blockers = append(execution.Blockers, execErr.Error())
			if execution.Status == "completed" {
				execution.Status = "failed"
			}
		}

		summary := execution.Summary
		if summary == "" {
			summary = fmt.Sprintf("%s worker %s finished without a structured summary.", plan.Role, plan.Name)
		}
		if err := spawnTree.UpdateStatus(plan.Name, execution.Status, summary); err != nil {
			return nil, fmt.Errorf("complete swarm worker %s: %w", plan.Name, err)
		}
		if err := updateSwarmDisplayStatus(swarmID, plan.Name, execution.Status); err != nil {
			return nil, fmt.Errorf("update swarm display final %s: %w", plan.Name, err)
		}
		if err := recordSwarmFinding(swarmID, plan.Name, response, execution); err != nil {
			return nil, fmt.Errorf("record swarm finding %s: %w", plan.Name, err)
		}

		runs = append(runs, execution)
	}
	return runs, nil
}

func invokeSwarmWorker(ctx context.Context, root, target, swarmID string, plan swarmWorkerPlan, priorSummary, responsePath string, invoker codex.WorkerInvoker) (*codex.WorkerResult, *swarmWorkerResponse, error) {
	brief := renderSwarmWorkerBrief(root, target, swarmID, plan, priorSummary, responsePath)
	cfg := codex.WorkerConfig{
		AgentName:        plan.AgentName,
		AgentTOMLPath:    dispatchAgentPath(root, invoker, plan.AgentName),
		Caste:            plan.Caste,
		WorkerName:       plan.Name,
		TaskID:           fmt.Sprintf("swarm.%s", plan.Role),
		TaskBrief:        brief,
		ContextCapsule:   resolveCodexWorkerContext(),
		HandoffSection:   renderWorkerHandoffSection("swarm", 0, plan.Name),
		Root:             root,
		Timeout:          firstSwarmTimeout(plan.Timeout),
		SkillSection:     resolveSkillSection(plan.Caste, plan.Task),
		PheromoneSection: resolvePheromoneSection(),
		ConfigOverrides:  swarmWorkerConfigOverrides(plan),
		ResponsePath:     responsePath,
	}

	result, err := invoker.Invoke(ctx, cfg)
	if err != nil {
		return nil, nil, err
	}

	var response *swarmWorkerResponse
	if loaded, loadErr := loadSwarmWorkerResponse(responsePath, plan.Role); loadErr == nil {
		response = &loaded
	}
	return &result, response, result.Error
}

func renderSwarmWorkerBrief(root, target, swarmID string, plan swarmWorkerPlan, priorSummary, responsePath string) string {
	responseRelPath, _ := filepath.Rel(root, responsePath)
	responseRelPath = filepath.ToSlash(responseRelPath)
	task := codex.RenderTaskBrief(codex.TaskBriefData{
		TaskID: fmt.Sprintf("swarm.%s", plan.Role),
		Goal:   fmt.Sprintf("Help destroy the reported bug: %s", target),
		Constraints: []string{
			fmt.Sprintf("Write exactly one structured swarm response file to %s.", emptyFallback(responseRelPath, responsePath)),
			"Be truthful. If you cannot safely make progress, return blocked with the concrete blocker.",
			"Use repo-relative paths for any files you mention.",
		},
		Hints: []string{
			fmt.Sprintf("Swarm ID: %s", swarmID),
			fmt.Sprintf("Role: %s", plan.Role),
			fmt.Sprintf("Assignment: %s", plan.Task),
			"If you are not the builder, stay read-only and report findings that help the builder and watcher.",
			"If you are the builder, implement the smallest safe fix and add or update tests.",
			"If you are the watcher, verify independently rather than trusting the builder summary.",
		},
		SuccessCriteria: []string{
			"The swarm response file is written with a concrete summary and actionable findings.",
			"The final worker claims JSON matches the real work performed.",
			"The pass reduces uncertainty about the bug or fixes it safely.",
		},
	})

	var b strings.Builder
	b.WriteString(task)
	b.WriteString("\n\n## Swarm Context\n\n")
	b.WriteString("- Target: " + strings.TrimSpace(target) + "\n")
	if strings.TrimSpace(priorSummary) != "" {
		b.WriteString("\n### Prior Swarm Findings\n\n")
		b.WriteString(strings.TrimSpace(priorSummary))
		b.WriteString("\n")
	}
	b.WriteString("\n## Swarm Response Contract\n\n")
	b.WriteString("Response File: " + emptyFallback(responseRelPath, responsePath) + "\n\n")
	b.WriteString("Write this JSON object to the response file:\n")
	b.WriteString("```json\n")
	b.WriteString("{\n")
	b.WriteString(`  "role": "` + plan.Role + `",` + "\n")
	b.WriteString(`  "status": "completed | blocked | failed",` + "\n")
	b.WriteString(`  "summary": "short concrete summary",` + "\n")
	b.WriteString(`  "findings": ["important discovery"],` + "\n")
	b.WriteString(`  "evidence": ["file path, command, or runtime output"],` + "\n")
	b.WriteString(`  "root_cause": "most likely root cause if known",` + "\n")
	b.WriteString(`  "recommendation": "next concrete action",` + "\n")
	b.WriteString(`  "proposed_fix": "what should change or what changed",` + "\n")
	b.WriteString(`  "files_touched": ["path/to/file"],` + "\n")
	b.WriteString(`  "tests_written": ["path/to/test"],` + "\n")
	b.WriteString(`  "verification": ["command or evidence of validation"]` + "\n")
	b.WriteString("}\n")
	b.WriteString("```\n")
	b.WriteString("- Do not write markdown to the response file.\n")
	b.WriteString("- Non-builder roles should leave files_touched/tests_written empty unless they truly changed something.\n")
	b.WriteString("- Builder and watcher responses must mention concrete verification evidence.\n")
	return strings.TrimSpace(b.String())
}

func loadSwarmWorkerResponse(path, role string) (swarmWorkerResponse, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return swarmWorkerResponse{}, err
	}
	var response swarmWorkerResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return swarmWorkerResponse{}, err
	}
	if strings.TrimSpace(response.Role) == "" {
		response.Role = role
	}
	status := strings.ToLower(strings.TrimSpace(response.Status))
	switch status {
	case "completed", "blocked", "failed":
		response.Status = status
	default:
		if len(response.Findings) > 0 || strings.TrimSpace(response.Summary) != "" {
			response.Status = "completed"
		} else {
			response.Status = "failed"
		}
	}
	response.Summary = strings.TrimSpace(response.Summary)
	response.Findings = swarmCompactStrings(response.Findings)
	response.Evidence = swarmCompactStrings(response.Evidence)
	response.FilesTouched = swarmCompactStrings(response.FilesTouched)
	response.TestsWritten = swarmCompactStrings(response.TestsWritten)
	response.Verification = swarmCompactStrings(response.Verification)
	response.RootCause = strings.TrimSpace(response.RootCause)
	response.Recommendation = strings.TrimSpace(response.Recommendation)
	response.ProposedFix = strings.TrimSpace(response.ProposedFix)
	return response, nil
}

func swarmResponsePath(swarmID, workerName string) string {
	name := strings.ToLower(strings.TrimSpace(workerName))
	name = strings.ReplaceAll(name, " ", "-")
	name = strings.ReplaceAll(name, "/", "-")
	return filepath.Join(store.BasePath(), "swarms", swarmID, "responses", name+".json")
}

func updateSwarmDisplayStatus(swarmID, agentName, status string) error {
	path := filepath.ToSlash(filepath.Join("swarms", swarmID, "display.json"))
	var display swarmDisplayFile
	if err := store.LoadJSON(path, &display); err != nil {
		return err
	}
	for i := range display.Agents {
		if display.Agents[i].Agent == agentName {
			display.Agents[i].Status = status
			return store.SaveJSON(path, display)
		}
	}
	display.Agents = append(display.Agents, swarmAgentStatus{Agent: agentName, Status: status})
	return store.SaveJSON(path, display)
}

func recordSwarmFinding(swarmID, agentName string, response *swarmWorkerResponse, execution swarmWorkerExecution) error {
	path := filepath.ToSlash(filepath.Join("swarms", swarmID, "findings.json"))
	var findings swarmFindingsFile
	if err := store.LoadJSON(path, &findings); err != nil {
		return err
	}
	text := strings.TrimSpace(execution.Summary)
	if response != nil {
		if response.RootCause != "" {
			text = fmt.Sprintf("%s Root cause: %s", emptyFallback(response.Summary, text), response.RootCause)
		} else if response.Summary != "" {
			text = response.Summary
		}
	}
	if text == "" {
		text = fmt.Sprintf("%s finished with status %s", agentName, execution.Status)
	}
	findings.Findings = append(findings.Findings, swarmFinding{Agent: agentName, Finding: text})
	if response != nil && response.Role == "builder" && response.ProposedFix != "" {
		findings.Solution = response.ProposedFix
	}
	if response != nil && response.Role == "watcher" && response.Summary != "" {
		findings.Solution = response.Summary
	}
	return store.SaveJSON(path, findings)
}

func renderSwarmFindingSummary(runs []swarmWorkerExecution) string {
	var lines []string
	for _, run := range runs {
		line := fmt.Sprintf("- %s (%s): %s", run.Name, run.Caste, emptyFallback(run.Summary, run.Status))
		if run.Response.RootCause != "" {
			line += " Root cause: " + run.Response.RootCause
		}
		if run.Response.ProposedFix != "" {
			line += " Proposed fix: " + run.Response.ProposedFix
		}
		lines = append(lines, line)
		for _, finding := range run.Response.Findings {
			lines = append(lines, "  - "+finding)
		}
	}
	return strings.Join(lines, "\n")
}

func summarizeSwarmOutcome(runs []swarmWorkerExecution) (status, recommendation, rootCause, solution string, blockers []string) {
	status = "completed"
	for _, run := range runs {
		if run.Response.RootCause != "" && rootCause == "" {
			rootCause = run.Response.RootCause
		}
		if run.Response.ProposedFix != "" && solution == "" {
			solution = run.Response.ProposedFix
		}
		if run.Response.Recommendation != "" {
			recommendation = run.Response.Recommendation
		}
		if len(run.Blockers) > 0 {
			blockers = append(blockers, run.Blockers...)
		}
		switch strings.ToLower(strings.TrimSpace(run.Status)) {
		case "blocked":
			if status != "failed" {
				status = "blocked"
			}
		case "failed", "timeout":
			status = "failed"
		}
	}
	if recommendation == "" {
		switch status {
		case "completed":
			recommendation = "Swarm completed a full investigate-fix-verify pass."
		case "blocked":
			recommendation = "Swarm found a blocker before the bug could be fully destroyed."
		default:
			recommendation = "Swarm failed before it could complete the bug-destroyer loop."
		}
	}
	if solution == "" {
		for _, run := range runs {
			if run.Caste == "builder" && run.Summary != "" {
				solution = run.Summary
				break
			}
		}
	}
	blockers = swarmCompactStrings(blockers)
	return status, recommendation, rootCause, solution, blockers
}

func collectSwarmTouchedFiles(runs []swarmWorkerExecution) ([]string, []string) {
	var files []string
	var tests []string
	for _, run := range runs {
		files = append(files, run.Files...)
		tests = append(tests, run.Tests...)
	}
	return swarmCompactStrings(files), swarmCompactStrings(tests)
}

func swarmNextCommand(state *colony.ColonyState, status string) string {
	if state != nil {
		if next := strings.TrimSpace(nextCommandFromState(*state)); next != "" {
			return next
		}
	}
	switch status {
	case "blocked", "failed":
		return "aether status"
	default:
		return "aether status"
	}
}

func firstSwarmTimeout(timeout time.Duration) time.Duration {
	if timeout > 0 {
		return timeout
	}
	return defaultSwarmWorkerTimeout
}

func swarmWorkerConfigOverrides(plan swarmWorkerPlan) []string {
	effort := "medium"
	if plan.Caste == "watcher" {
		effort = "high"
	}
	return []string{fmt.Sprintf("model_reasoning_effort=%q", effort)}
}

func swarmCompactStrings(values []string) []string {
	out := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func swarmExecutionsForJSON(runs []swarmWorkerExecution) []map[string]interface{} {
	out := make([]map[string]interface{}, 0, len(runs))
	for _, run := range runs {
		entry := map[string]interface{}{
			"name":     run.Name,
			"caste":    run.Caste,
			"role":     run.Role,
			"task":     run.Task,
			"status":   run.Status,
			"summary":  run.Summary,
			"duration": run.Duration,
		}
		if len(run.Files) > 0 {
			entry["files"] = run.Files
		}
		if len(run.Tests) > 0 {
			entry["tests"] = run.Tests
		}
		if len(run.Blockers) > 0 {
			entry["blockers"] = run.Blockers
		}
		if run.Response.Role != "" {
			entry["response"] = run.Response
		}
		out = append(out, entry)
	}
	return out
}

func renderSwarmDispatchPreview(swarmID, target string, plans []swarmWorkerPlan, title string) string {
	var b strings.Builder
	b.WriteString(renderBanner(commandEmoji("swarm-display"), title))
	b.WriteString(visualDivider)
	b.WriteString("Swarm ID: " + swarmID + "\n")
	b.WriteString("Target: " + strings.TrimSpace(target) + "\n\n")
	for _, plan := range plans {
		b.WriteString("  ")
		b.WriteString(casteEmoji(plan.Caste))
		b.WriteString(" ")
		b.WriteString(plan.Name)
		b.WriteString(" (")
		b.WriteString(plan.Caste)
		b.WriteString(") — ")
		b.WriteString(plan.Task)
		b.WriteString("\n")
	}
	return b.String()
}

func renderSwarmCompatibilityVisual(result map[string]interface{}) string {
	var b strings.Builder
	b.WriteString(renderBanner(commandEmoji("swarm"), "Swarm"))
	b.WriteString(visualDivider)

	mode := strings.TrimSpace(stringValue(result["mode"]))
	target := strings.TrimSpace(stringValue(result["target"]))
	if mode == "watch" {
		if live, _ := result["live_refresh"].(bool); live {
			b.WriteString("Live colony activity view.\n")
			b.WriteString("Refreshing automatically. Press Ctrl+C to exit.\n")
		} else {
			b.WriteString("Colony activity snapshot.\n")
			b.WriteString("Run in a TTY for live refresh.\n")
		}
		if target != "" {
			b.WriteString("Target: " + target + "\n")
		}
		if goal := strings.TrimSpace(stringValue(result["goal"])); goal != "" {
			b.WriteString("Goal: " + goal + "\n")
		}
		if scope := strings.TrimSpace(stringValue(result["scope"])); scope != "" {
			b.WriteString("Scope: " + scope + "\n")
		}
		if state := strings.TrimSpace(stringValue(result["state"])); state != "" {
			b.WriteString("State: " + state + "\n")
		}
		if phaseName := strings.TrimSpace(stringValue(result["phase_name"])); phaseName != "" {
			b.WriteString("Phase: " + phaseName + "\n")
		}
		if runCommand := strings.TrimSpace(stringValue(result["current_run_command"])); runCommand != "" {
			b.WriteString("Current Run: " + runCommand)
			if runID := strings.TrimSpace(stringValue(result["current_run_id"])); runID != "" {
				b.WriteString(" (" + runID + ")")
			}
			b.WriteString("\n")
		}
		b.WriteString(fmt.Sprintf("Workers: %d active | %d completed | %d blocked",
			intValue(result["active_count"]),
			intValue(result["completed_count"]),
			intValue(result["blocked_count"]),
		))
		if failed := intValue(result["failed_count"]); failed > 0 {
			b.WriteString(fmt.Sprintf(" | %d failed", failed))
		}
		b.WriteString("\n")
		renderSwarmWorkerSection(&b, "Active Workers", workerMapsFromResult(result, "active_workers"))
		renderSwarmWorkerSection(&b, "Recent Outcomes", workerMapsFromResult(result, "recent_workers"))
		if recoverySummary := strings.TrimSpace(stringValue(result["recovery_summary"])); recoverySummary != "" {
			b.WriteString("\nRecovery\n")
			b.WriteString("  ")
			b.WriteString(recoverySummary)
			b.WriteString("\n")
			if recoveryCommand := strings.TrimSpace(stringValue(result["recovery_command"])); recoveryCommand != "" {
				b.WriteString("  Next: ")
				b.WriteString(recoveryCommand)
				b.WriteString("\n")
			}
			if continueReport := strings.TrimSpace(stringValue(result["continue_report"])); continueReport != "" {
				b.WriteString("  Report: ")
				b.WriteString(continueReport)
				b.WriteString("\n")
			}
		}
		b.WriteString(renderArtifactsSection(
			displayDataPath("spawn-tree.txt"),
			displayDataPath("watch-status.txt"),
			displayDataPath("watch-progress.txt"),
		))
		next := strings.TrimSpace(stringValue(result["next"]))
		if next == "" {
			next = "aether status"
		}
		primary := fmt.Sprintf("Run `%s` to inspect the colony in more detail.", next)
		if recoveryCommand := strings.TrimSpace(stringValue(result["recovery_command"])); recoveryCommand != "" && recoveryCommand == next {
			primary = fmt.Sprintf("Run `%s` to recover the blocked work.", next)
		}
		b.WriteString(renderNextUp(
			primary,
			`Run `+"`aether swarm \"describe the problem\"`"+` to launch the bug-destroyer flow.`,
		))
		return b.String()
	}

	dispatchMode := strings.TrimSpace(stringValue(result["dispatch_mode"]))
	requiresFinalizer, _ := result["requires_finalizer"].(bool)
	if requiresFinalizer || dispatchMode == "plan-only" || dispatchMode == "agent-delegate" {
		b.WriteString("Swarm dispatch manifest ready.\n")
		if swarmID := strings.TrimSpace(stringValue(result["swarm_id"])); swarmID != "" {
			b.WriteString("Swarm ID: " + swarmID + "\n")
		}
		if target != "" {
			b.WriteString("Target: " + target + "\n")
		}
		if dispatchMode != "" {
			b.WriteString("Dispatch: " + dispatchMode + "\n")
		}
		renderSwarmWorkers(&b, result)
		finalizer := strings.TrimSpace(stringValue(result["finalizer_command"]))
		if finalizer == "" {
			finalizer = "AETHER_OUTPUT_MODE=json aether swarm-finalize --completion-file <file>"
		}
		b.WriteString(renderNextUp(
			"Host platform should dispatch the swarm workers above, then run `"+finalizer+"`.",
			"Do not hand-edit `.aether/data/`; the finalizer writes swarm artifacts and status.",
		))
		return b.String()
	}

	b.WriteString("Swarm bug-destroyer completed a real worker pass.\n")
	if swarmID := strings.TrimSpace(stringValue(result["swarm_id"])); swarmID != "" {
		b.WriteString("Swarm ID: " + swarmID + "\n")
	}
	if target != "" {
		b.WriteString("Target: " + target + "\n")
	}
	if status := strings.TrimSpace(stringValue(result["status"])); status != "" {
		b.WriteString("Status: " + status + "\n")
	}
	if rootCause := strings.TrimSpace(stringValue(result["root_cause"])); rootCause != "" {
		b.WriteString("Root Cause: " + rootCause + "\n")
	}
	if solution := strings.TrimSpace(stringValue(result["solution"])); solution != "" {
		b.WriteString("Solution: " + solution + "\n")
	}
	if recommendation := strings.TrimSpace(stringValue(result["recommendation"])); recommendation != "" {
		b.WriteString("Recommendation: " + recommendation + "\n")
	}
	renderSwarmWorkers(&b, result)

	files, _ := result["files_touched"].([]interface{})
	if len(files) > 0 {
		b.WriteString("\nFiles Touched\n")
		for _, file := range files {
			b.WriteString("  - " + stringValue(file) + "\n")
		}
	}
	tests, _ := result["tests_written"].([]interface{})
	if len(tests) > 0 {
		b.WriteString("\nTests Written\n")
		for _, file := range tests {
			b.WriteString("  - " + stringValue(file) + "\n")
		}
	}

	next := strings.TrimSpace(stringValue(result["next"]))
	if next == "" {
		next = "aether status"
	}
	b.WriteString(renderNextUp(
		fmt.Sprintf("Run `%s` for the next lifecycle step.", next),
		"Run `aether swarm --watch` to inspect any remaining live worker activity.",
	))
	return b.String()
}

func renderSwarmWorkers(b *strings.Builder, result map[string]interface{}) {
	raw := workerMapsFromResult(result, "workers")
	if len(raw) == 0 {
		raw = workerMapsFromResult(result, "active_workers")
	}
	if len(raw) == 0 {
		return
	}

	renderSwarmWorkerSection(b, "Workers", raw)
}

func workerMapsFromResult(result map[string]interface{}, key string) []map[string]interface{} {
	raw, ok := result[key].([]map[string]interface{})
	if ok && raw != nil {
		return raw
	}
	list, ok := result[key].([]interface{})
	if !ok {
		return nil
	}
	raw = make([]map[string]interface{}, 0, len(list))
	for _, item := range list {
		if entry, ok := item.(map[string]interface{}); ok {
			raw = append(raw, entry)
		}
	}
	return raw
}

func renderSwarmWorkerSection(b *strings.Builder, title string, raw []map[string]interface{}) {
	if len(raw) == 0 {
		return
	}
	b.WriteString("\n")
	b.WriteString(title)
	b.WriteString("\n")
	for _, entry := range raw {
		caste := strings.TrimSpace(stringValue(entry["caste"]))
		status := strings.TrimSpace(stringValue(entry["status"]))
		fmt.Fprintf(b, "  %s %s %s",
			dispatchStatusIcon(status),
			casteIdentity(caste),
			stringValue(entry["name"]),
		)
		if role := strings.TrimSpace(stringValue(entry["role"])); role != "" {
			b.WriteString(" (")
			b.WriteString(role)
			b.WriteString(")")
		}
		task := strings.TrimSpace(stringValue(entry["task"]))
		if task != "" {
			b.WriteString(" — ")
			b.WriteString(task)
		}
		if status != "" {
			b.WriteString(" [")
			b.WriteString(status)
			b.WriteString("]")
		}
		b.WriteString("\n")
		if summary := strings.TrimSpace(stringValue(entry["summary"])); summary != "" && summary != task {
			b.WriteString("    ")
			b.WriteString(summary)
			b.WriteString("\n")
		}
	}
}
