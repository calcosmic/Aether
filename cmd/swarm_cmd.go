package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/agent"
	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/spf13/cobra"
)

const defaultSwarmWorkerTimeout = 6 * time.Minute

var newSwarmWorkerInvoker = codex.NewWorkerInvoker

type swarmWorkerPlan struct {
	Name      string
	Caste     string
	Role      string
	Task      string
	AgentTOML string
	Wave      int
	Timeout   time.Duration
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
		target := strings.TrimSpace(strings.Join(args, " "))
		root := resolveAetherRootPath()

		result, err := runSwarmCompatibility(root, target, watch)
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
	rootCmd.AddCommand(swarmCmd)
}

func runSwarmCompatibility(root, target string, watch bool) (map[string]interface{}, error) {
	if watch || strings.TrimSpace(target) == "" {
		return buildSwarmWatchResult(target, watch), nil
	}
	return runSwarmDestroy(root, target)
}

func buildSwarmWatchResult(target string, watch bool) map[string]interface{} {
	state, _ := loadColonyState()
	active := loadActiveSpawnEntries(store)

	next := `aether init "describe the goal"`
	phaseName := ""
	stateName := ""
	goal := ""
	if state != nil {
		next = nextCommandFromState(*state)
		phaseName = lookupPhaseName(*state, state.CurrentPhase)
		stateName = string(state.State)
		if state.Goal != nil {
			goal = strings.TrimSpace(*state.Goal)
		}
		if strings.TrimSpace(next) == "" {
			next = "aether status"
		}
	}

	workers := make([]map[string]interface{}, 0, len(active))
	for _, entry := range active {
		workers = append(workers, map[string]interface{}{
			"name":   entry.AgentName,
			"caste":  entry.Caste,
			"task":   entry.Task,
			"status": entry.Status,
		})
	}

	return map[string]interface{}{
		"mode":                "watch",
		"target":              target,
		"autopilot_available": true,
		"goal":                goal,
		"state":               stateName,
		"phase_name":          phaseName,
		"active_workers":      workers,
		"active_count":        len(workers),
		"next":                next,
		"watch":               watch || target == "",
	}
}

func runSwarmDestroy(root, target string) (map[string]interface{}, error) {
	invoker := newSwarmWorkerInvoker()
	if invoker == nil {
		return nil, fmt.Errorf("swarm worker invoker is not configured")
	}
	if !invoker.IsAvailable(context.Background()) {
		return nil, fmt.Errorf("codex CLI is not available in PATH")
	}

	state, _ := loadColonyState()
	swarmID := fmt.Sprintf("swarm-%d", time.Now().UTC().Unix())
	if err := initializeSwarmRun(swarmID); err != nil {
		return nil, fmt.Errorf("initialize swarm workspace: %w", err)
	}

	investigation := buildSwarmInvestigationPlans(root, target)
	emitVisualProgress(renderSwarmDispatchPreview(swarmID, target, investigation, "Investigation Wave"))
	investigationRuns, err := executeSwarmWave(context.Background(), root, swarmID, target, investigation, "", invoker)
	if err != nil {
		return nil, err
	}

	findingSummary := renderSwarmFindingSummary(investigationRuns)
	builderPlan := buildSwarmBuilderPlan(root, target)
	emitVisualProgress(renderSwarmDispatchPreview(swarmID, target, []swarmWorkerPlan{builderPlan}, "Fix Wave"))
	builderRuns, err := executeSwarmWave(context.Background(), root, swarmID, target, []swarmWorkerPlan{builderPlan}, findingSummary, invoker)
	if err != nil {
		return nil, err
	}

	builderSummary := renderSwarmFindingSummary(builderRuns)
	watcherPlan := buildSwarmWatcherPlan(root, target)
	emitVisualProgress(renderSwarmDispatchPreview(swarmID, target, []swarmWorkerPlan{watcherPlan}, "Verification Wave"))
	watcherRuns, err := executeSwarmWave(context.Background(), root, swarmID, target, []swarmWorkerPlan{watcherPlan}, findingSummary+"\n\n"+builderSummary, invoker)
	if err != nil {
		return nil, err
	}

	allRuns := append(append([]swarmWorkerExecution{}, investigationRuns...), builderRuns...)
	allRuns = append(allRuns, watcherRuns...)

	status, recommendation, rootCause, solution, blockers := summarizeSwarmOutcome(allRuns)
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

func buildSwarmInvestigationPlans(root, target string) []swarmWorkerPlan {
	seed := strings.ToLower(strings.TrimSpace(target))
	return []swarmWorkerPlan{
		{
			Name:      deterministicAntName("tracker", root+"|swarm|tracker|"+seed),
			Caste:     "tracker",
			Role:      "tracker",
			Task:      "Reproduce the issue, trace the failure path, and identify the most likely root cause.",
			AgentTOML: filepath.Join(root, ".codex", "agents", "aether-tracker.toml"),
			Wave:      1,
			Timeout:   defaultSwarmWorkerTimeout,
		},
		{
			Name:      deterministicAntName("scout", root+"|swarm|scout|"+seed),
			Caste:     "scout",
			Role:      "scout",
			Task:      "Search the repo for the most relevant files, patterns, tests, and documentation tied to the reported bug.",
			AgentTOML: filepath.Join(root, ".codex", "agents", "aether-scout.toml"),
			Wave:      1,
			Timeout:   defaultSwarmWorkerTimeout,
		},
		{
			Name:      deterministicAntName("archaeologist", root+"|swarm|archaeology|"+seed),
			Caste:     "archaeologist",
			Role:      "archaeologist",
			Task:      "Inspect git history and prior fixes around the bug area to identify historical context, fragile zones, and regressions.",
			AgentTOML: filepath.Join(root, ".codex", "agents", "aether-archaeologist.toml"),
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
		AgentTOML: filepath.Join(root, ".codex", "agents", "aether-builder.toml"),
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
		AgentTOML: filepath.Join(root, ".codex", "agents", "aether-watcher.toml"),
		Wave:      3,
		Timeout:   defaultSwarmWorkerTimeout,
	}
}

func executeSwarmWave(ctx context.Context, root, swarmID, target string, plans []swarmWorkerPlan, priorSummary string, invoker codex.WorkerInvoker) ([]swarmWorkerExecution, error) {
	spawnTree := agent.NewSpawnTree(store, "spawn-tree.txt")
	runs := make([]swarmWorkerExecution, 0, len(plans))
	for _, plan := range plans {
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
		AgentName:        strings.TrimSuffix(filepath.Base(plan.AgentTOML), ".toml"),
		AgentTOMLPath:    plan.AgentTOML,
		Caste:            plan.Caste,
		WorkerName:       plan.Name,
		TaskID:           fmt.Sprintf("swarm.%s", plan.Role),
		TaskBrief:        brief,
		ContextCapsule:   resolveCodexWorkerContext(),
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
		case "failed":
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
	b.WriteString(renderBanner("🔥", title))
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
	b.WriteString(renderBanner("🔥", "Swarm"))
	b.WriteString(visualDivider)

	mode := strings.TrimSpace(stringValue(result["mode"]))
	target := strings.TrimSpace(stringValue(result["target"]))
	if mode == "watch" {
		b.WriteString("Live colony activity view.\n")
		if target != "" {
			b.WriteString("Target: " + target + "\n")
		}
		renderSwarmWorkers(&b, result)
		next := strings.TrimSpace(stringValue(result["next"]))
		if next == "" {
			next = "aether status"
		}
		b.WriteString(renderNextUp(
			fmt.Sprintf("Run `%s` to inspect the colony in more detail.", next),
			`Run `+"`aether swarm \"describe the problem\"`"+` to launch the bug-destroyer flow.`,
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
	raw, ok := result["workers"].([]map[string]interface{})
	if !ok || raw == nil {
		if list, ok := result["workers"].([]interface{}); ok {
			raw = make([]map[string]interface{}, 0, len(list))
			for _, item := range list {
				if entry, ok := item.(map[string]interface{}); ok {
					raw = append(raw, entry)
				}
			}
		}
	}
	if len(raw) == 0 {
		raw, _ = result["active_workers"].([]map[string]interface{})
		if raw == nil {
			if list, ok := result["active_workers"].([]interface{}); ok {
				raw = make([]map[string]interface{}, 0, len(list))
				for _, item := range list {
					if entry, ok := item.(map[string]interface{}); ok {
						raw = append(raw, entry)
					}
				}
			}
		}
	}
	if len(raw) == 0 {
		return
	}

	b.WriteString("\nWorkers\n")
	for _, entry := range raw {
		caste := strings.TrimSpace(stringValue(entry["caste"]))
		b.WriteString("  ")
		b.WriteString(casteEmoji(caste))
		b.WriteString(" ")
		b.WriteString(stringValue(entry["name"]))
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
		if status := strings.TrimSpace(stringValue(entry["status"])); status != "" {
			b.WriteString(" [")
			b.WriteString(status)
			b.WriteString("]")
		}
		b.WriteString("\n")
		if summary := strings.TrimSpace(stringValue(entry["summary"])); summary != "" {
			b.WriteString("    ")
			b.WriteString(summary)
			b.WriteString("\n")
		}
	}
}
