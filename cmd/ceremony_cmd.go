package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

var ceremonyFlags struct {
	Workflow       string
	ManifestFile   string
	WorkerFile     string
	CompletionFile string
	ExecutionWave  int
}

type ceremonyDispatch struct {
	Name          string
	Caste         string
	Task          string
	TaskID        string
	Stage         string
	Status        string
	Summary       string
	Wave          int
	ExecutionWave int
	ToolCount     int
	Blockers      []string
	Artifacts     []string
}

type ceremonyExecutionPlan struct {
	ExecutionWave int
	Wave          int
	Stage         string
	Strategy      string
	WorkerCount   int
	Reason        string
}

var ceremonyCmd = &cobra.Command{
	Use:   "ceremony",
	Short: "Render runtime-owned wrapper ceremony surfaces",
}

var ceremonySpawnPlanCmd = &cobra.Command{
	Use:   "spawn-plan",
	Short: "Render a visual spawn plan from a lifecycle manifest JSON file",
	RunE: func(cmd *cobra.Command, args []string) error {
		result, visual, err := renderCeremonySpawnPlanFromFile(ceremonyFlags.Workflow, ceremonyFlags.ManifestFile)
		if err != nil {
			return err
		}
		outputWorkflow(result, visual)
		return nil
	},
}

var ceremonyWaveStartCmd = &cobra.Command{
	Use:   "wave-start",
	Short: "Render a visual wave-start banner from a lifecycle manifest JSON file",
	RunE: func(cmd *cobra.Command, args []string) error {
		result, visual, err := renderCeremonyWaveStartFromFile(ceremonyFlags.Workflow, ceremonyFlags.ManifestFile, ceremonyFlags.ExecutionWave)
		if err != nil {
			return err
		}
		outputWorkflow(result, visual)
		return nil
	},
}

var ceremonyWorkerCompleteCmd = &cobra.Command{
	Use:   "worker-complete",
	Short: "Render a visual worker completion line from a worker result JSON file",
	RunE: func(cmd *cobra.Command, args []string) error {
		result, visual, err := renderCeremonyWorkerCompleteFromFile(ceremonyFlags.Workflow, ceremonyFlags.WorkerFile)
		if err != nil {
			return err
		}
		outputWorkflow(result, visual)
		return nil
	},
}

var ceremonyCloseoutCmd = &cobra.Command{
	Use:   "closeout",
	Short: "Render an old-style lifecycle summary from a completion JSON file",
	RunE: func(cmd *cobra.Command, args []string) error {
		result, visual := renderCeremonyCloseout(ceremonyFlags.Workflow, ceremonyFlags.CompletionFile)
		outputWorkflow(result, visual)
		return nil
	},
}

func init() {
	ceremonySpawnPlanCmd.Flags().StringVar(&ceremonyFlags.Workflow, "workflow", "build", "Lifecycle workflow name")
	ceremonySpawnPlanCmd.Flags().StringVar(&ceremonyFlags.ManifestFile, "manifest-file", "", "JSON file containing the runtime manifest envelope")
	_ = ceremonySpawnPlanCmd.MarkFlagRequired("manifest-file")

	ceremonyWaveStartCmd.Flags().StringVar(&ceremonyFlags.Workflow, "workflow", "build", "Lifecycle workflow name")
	ceremonyWaveStartCmd.Flags().StringVar(&ceremonyFlags.ManifestFile, "manifest-file", "", "JSON file containing the runtime manifest envelope")
	ceremonyWaveStartCmd.Flags().IntVar(&ceremonyFlags.ExecutionWave, "execution-wave", 0, "Execution wave to render")
	_ = ceremonyWaveStartCmd.MarkFlagRequired("manifest-file")
	_ = ceremonyWaveStartCmd.MarkFlagRequired("execution-wave")

	ceremonyWorkerCompleteCmd.Flags().StringVar(&ceremonyFlags.Workflow, "workflow", "build", "Lifecycle workflow name")
	ceremonyWorkerCompleteCmd.Flags().StringVar(&ceremonyFlags.WorkerFile, "worker-file", "", "JSON file containing one worker terminal result")
	_ = ceremonyWorkerCompleteCmd.MarkFlagRequired("worker-file")

	ceremonyCloseoutCmd.Flags().StringVar(&ceremonyFlags.Workflow, "workflow", "build", "Lifecycle workflow name")
	ceremonyCloseoutCmd.Flags().StringVar(&ceremonyFlags.CompletionFile, "completion-file", "", "Completion JSON packet used by a lifecycle finalizer")
	_ = ceremonyCloseoutCmd.MarkFlagRequired("completion-file")

	ceremonyCmd.AddCommand(ceremonySpawnPlanCmd, ceremonyWaveStartCmd, ceremonyWorkerCompleteCmd, ceremonyCloseoutCmd)
	rootCmd.AddCommand(ceremonyCmd)
}

func renderCeremonySpawnPlanFromFile(workflow, path string) (map[string]interface{}, string, error) {
	raw, err := readCeremonyJSONFile(path)
	if err != nil {
		return nil, "", err
	}
	manifest, manifestKey := extractCeremonyManifest(raw)
	if len(manifest) == 0 {
		return nil, "", fmt.Errorf("manifest file %s does not contain a lifecycle manifest", path)
	}
	dispatches := ceremonyDispatchesFromManifest(manifest)
	plans := ceremonyExecutionPlansFromManifest(manifest, dispatches)
	result := map[string]interface{}{
		"workflow":       normalizedCeremonyWorkflow(workflow),
		"manifest_file":  path,
		"manifest":       manifestKey,
		"dispatch_count": len(dispatches),
	}
	return result, renderCeremonySpawnPlan(normalizedCeremonyWorkflow(workflow), manifest, dispatches, plans), nil
}

func renderCeremonyWaveStartFromFile(workflow, path string, executionWave int) (map[string]interface{}, string, error) {
	raw, err := readCeremonyJSONFile(path)
	if err != nil {
		return nil, "", err
	}
	manifest, manifestKey := extractCeremonyManifest(raw)
	if len(manifest) == 0 {
		return nil, "", fmt.Errorf("manifest file %s does not contain a lifecycle manifest", path)
	}
	dispatches := ceremonyDispatchesFromManifest(manifest)
	plans := ceremonyExecutionPlansFromManifest(manifest, dispatches)
	filtered, plan := ceremonyDispatchesForExecutionWave(dispatches, plans, executionWave)
	result := map[string]interface{}{
		"workflow":       normalizedCeremonyWorkflow(workflow),
		"manifest_file":  path,
		"manifest":       manifestKey,
		"execution_wave": executionWave,
		"dispatch_count": len(filtered),
	}
	return result, renderCeremonyWaveStart(normalizedCeremonyWorkflow(workflow), executionWave, filtered, plan), nil
}

func renderCeremonyWorkerCompleteFromFile(workflow, path string) (map[string]interface{}, string, error) {
	raw, err := readCeremonyJSONFile(path)
	if err != nil {
		return nil, "", err
	}
	worker := extractCeremonyWorker(raw)
	if len(worker) == 0 {
		return nil, "", fmt.Errorf("worker file %s does not contain a worker result", path)
	}
	dispatch := ceremonyDispatchFromMap(worker)
	result := map[string]interface{}{
		"workflow":    normalizedCeremonyWorkflow(workflow),
		"worker_file": path,
		"name":        dispatch.Name,
		"caste":       dispatch.Caste,
		"status":      dispatch.Status,
	}
	return result, renderCeremonyWorkerComplete(normalizedCeremonyWorkflow(workflow), dispatch), nil
}

func renderCeremonyCloseout(workflow, completionFile string) (map[string]interface{}, string) {
	workflow = normalizedCeremonyWorkflow(workflow)
	result := map[string]interface{}{
		"workflow":        workflow,
		"completion_file": completionFile,
	}
	for key, value := range closeoutCompletionDetails(completionFile) {
		result[key] = value
	}
	if workflow == "seal" {
		result["porter_readiness"] = buildPorterReadinessSummary()
	}
	if state, err := loadActiveColonyState(); err == nil {
		result["state_available"] = true
		result["state"] = string(state.State)
		result["current_phase"] = state.CurrentPhase
		result["total_phases"] = len(state.Plan.Phases)
		result["completed_phases"] = completedPhaseCount(state)
		result["milestone"] = state.Milestone
		result["phase_name"] = lookupPhaseName(state, state.CurrentPhase)
		if state.Goal != nil {
			result["goal"] = *state.Goal
		}
		result["next"] = closeoutNextCommand(workflow, state)
	} else {
		result["state_available"] = false
		result["message"] = colonyStateLoadMessage(err)
		result["next"] = "Run `aether status` to inspect the colony."
	}
	return result, renderCeremonyCloseoutVisual(result)
}

func renderCeremonySpawnPlan(workflow string, manifest map[string]interface{}, dispatches []ceremonyDispatch, plans []ceremonyExecutionPlan) string {
	var b strings.Builder
	b.WriteString(renderOldStyleCeremonyHeader(commandEmoji(emptyFallback(workflow, "spawn-plan")), "Spawn Plan"))
	b.WriteString("\n")
	writeCeremonyPhaseLine(&b, manifest)
	if len(dispatches) == 0 {
		b.WriteString("No workers planned.\n")
		return b.String()
	}

	for idx, plan := range plans {
		if idx > 0 {
			b.WriteString("\n")
		}
		b.WriteString(ceremonyPlanLabel(plan))
		b.WriteString("\n")
		for _, dispatch := range dispatchesForPlan(dispatches, plan) {
			writeCeremonyDispatchLine(&b, dispatch, "  ")
		}
	}
	b.WriteString("\n")
	b.WriteString("Total: ")
	b.WriteString(ceremonyCasteCountSummary(dispatches))
	b.WriteString(fmt.Sprintf(" = %d spawns\n", len(dispatches)))
	return b.String()
}

func renderCeremonyWaveStart(workflow string, executionWave int, dispatches []ceremonyDispatch, plan ceremonyExecutionPlan) string {
	var b strings.Builder
	if len(dispatches) == 0 {
		fmt.Fprintf(&b, "──── 🐜 No workers planned for execution wave %d ────\n", executionWave)
		return b.String()
	}
	castes := ceremonyCasteCounts(dispatches)
	primaryCaste, primaryCount := dominantCeremonyCaste(castes)
	strategy := strings.ToLower(strings.TrimSpace(plan.Strategy))
	if strategy == "" {
		strategy = "parallel"
	}
	if len(castes) == 1 {
		fmt.Fprintf(&b, "──── %s Spawning %d %s in %s ────\n", casteEmoji(primaryCaste), primaryCount, pluralizeCaste(primaryCaste, primaryCount), strategy)
	} else {
		fmt.Fprintf(&b, "──── 🐜 Spawning %d workers (%s) in %s ────\n", len(dispatches), ceremonyInlineCasteCounts(castes), strategy)
	}
	for _, dispatch := range dispatches {
		writeCeremonyDispatchLine(&b, dispatch, "  ")
	}
	return b.String()
}

func renderCeremonyWorkerComplete(workflow string, dispatch ceremonyDispatch) string {
	var b strings.Builder
	status := normalizeRuntimeDispatchStatus(dispatch.Status)
	if status == "" {
		status = "completed"
	}
	fmt.Fprintf(&b, "%s %s %s", dispatchStatusIcon(status), casteIdentity(dispatch.Caste), emptyFallback(dispatch.Name, "worker"))
	if dispatch.TaskID != "" {
		b.WriteString("  Task ")
		b.WriteString(dispatch.TaskID)
	}
	if dispatch.Summary != "" {
		b.WriteString(" — ")
		b.WriteString(dispatch.Summary)
	} else if dispatch.Task != "" {
		b.WriteString(" — ")
		b.WriteString(dispatch.Task)
	}
	if dispatch.ToolCount > 0 {
		fmt.Fprintf(&b, " (%d tools)", dispatch.ToolCount)
	}
	b.WriteString("\n")
	if len(dispatch.Blockers) > 0 {
		b.WriteString("  Blockers:\n")
		b.WriteString(renderIndentedList(dispatch.Blockers))
	}
	return b.String()
}

func renderCeremonyCloseoutVisual(result map[string]interface{}) string {
	workflow := normalizedCeremonyWorkflow(stringValue(result["workflow"]))
	var b strings.Builder
	b.WriteString(visualDivider)
	b.WriteString(commandEmoji(workflow))
	b.WriteString(" ")
	b.WriteString(spacedTitle(fmt.Sprintf("%s Summary", workflow)))
	b.WriteString("\n")
	b.WriteString(visualDivider)
	if goal := strings.TrimSpace(stringValue(result["goal"])); goal != "" {
		b.WriteString("Goal: ")
		b.WriteString(goal)
		b.WriteString("\n")
	}
	if phase := intValue(result["completion_phase"]); phase > 0 {
		fmt.Fprintf(&b, "Phase: %d", phase)
		if phaseName := strings.TrimSpace(stringValue(result["completion_phase_name"])); phaseName != "" {
			b.WriteString(" - ")
			b.WriteString(phaseName)
		}
		b.WriteString("\n")
	} else if phase := intValue(result["current_phase"]); phase > 0 {
		fmt.Fprintf(&b, "Current phase: %d", phase)
		if phaseName := strings.TrimSpace(stringValue(result["phase_name"])); phaseName != "" && phaseName != "(unnamed)" {
			b.WriteString(" - ")
			b.WriteString(phaseName)
		}
		b.WriteString("\n")
	}
	writeCeremonyWorkerSummary(&b, result)
	if readiness := strings.TrimSpace(stringValue(result["porter_readiness"])); readiness != "" {
		b.WriteString("\n")
		b.WriteString(renderStageMarker("Post-Seal: Delivery Readiness"))
		b.WriteString(readiness)
		if !strings.HasSuffix(readiness, "\n") {
			b.WriteString("\n")
		}
	}
	next := emptyFallback(stringValue(result["next"]), "Run `aether status` to inspect the colony.")
	b.WriteString(renderNextUp(next))
	return b.String()
}

func writeCeremonyWorkerSummary(b *strings.Builder, result map[string]interface{}) {
	workers := mapSliceValue(result["completion_workers"])
	completed := intValue(result["completion_completed"])
	blocked := intValue(result["completion_blocked"])
	failed := intValue(result["completion_failed"])
	total := intValue(result["completion_worker_count"])
	if total == 0 && len(workers) > 0 {
		total = len(workers)
	}
	if total > 0 {
		fmt.Fprintf(b, "\nWorkers: %d completed  %d blocked  %d failed  (%d total)\n", completed, blocked, failed, total)
	}
	toolCount := 0
	for _, worker := range workers {
		toolCount += intValue(worker["tool_count"])
	}
	if toolCount > 0 {
		fmt.Fprintf(b, "Tools: %d calls across workers\n", toolCount)
	}
	artifacts := stringSliceValue(result["completion_artifacts"])
	if len(artifacts) > 0 {
		b.WriteString("\nChanged / Produced\n")
		limit := artifacts
		if len(limit) > 10 {
			limit = limit[:10]
		}
		b.WriteString(renderIndentedList(limit))
		if len(artifacts) > len(limit) {
			fmt.Fprintf(b, "  - ...and %d more\n", len(artifacts)-len(limit))
		}
	}
	if len(workers) > 0 {
		b.WriteString("\n")
		b.WriteString(renderStageMarker("Worker Results"))
		for _, worker := range workers {
			b.WriteString(renderCeremonyWorkerComplete(normalizedCeremonyWorkflow(stringValue(result["workflow"])), ceremonyDispatchFromMap(worker)))
		}
	}
	if blockers := stringSliceValue(result["completion_blockers"]); len(blockers) > 0 {
		b.WriteString("\nBlockers\n")
		b.WriteString(renderIndentedList(blockers))
	}
}

func readCeremonyJSONFile(path string) (map[string]interface{}, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, fmt.Errorf("JSON file path is required")
	}
	var data []byte
	var err error
	if path == "-" {
		data, err = io.ReadAll(os.Stdin)
	} else {
		data, err = os.ReadFile(path)
	}
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	if nested, ok := raw["result"].(map[string]interface{}); ok {
		return nested, nil
	}
	return raw, nil
}

func extractCeremonyManifest(raw map[string]interface{}) (map[string]interface{}, string) {
	if key, manifest := closeoutManifest(raw); key != "" {
		return manifest, key
	}
	if len(mapSliceValue(raw["dispatches"])) > 0 {
		return raw, "manifest"
	}
	return nil, ""
}

func extractCeremonyWorker(raw map[string]interface{}) map[string]interface{} {
	if nested, ok := raw["result"].(map[string]interface{}); ok {
		raw = nested
	}
	for _, key := range []string{"worker", "dispatch", "result"} {
		if worker := mapValue(raw[key]); len(worker) > 0 {
			return worker
		}
	}
	for _, key := range []string{"dispatches", "workers", "results"} {
		if workers := mapSliceValue(raw[key]); len(workers) > 0 {
			return workers[0]
		}
	}
	return raw
}

func ceremonyDispatchesFromManifest(manifest map[string]interface{}) []ceremonyDispatch {
	rawDispatches := mapSliceValue(manifest["dispatches"])
	dispatches := make([]ceremonyDispatch, 0, len(rawDispatches))
	for _, raw := range rawDispatches {
		dispatches = append(dispatches, ceremonyDispatchFromMap(raw))
	}
	return dispatches
}

func ceremonyDispatchFromMap(raw map[string]interface{}) ceremonyDispatch {
	name := emptyFallback(stringValue(raw["name"]), stringValue(raw["agent_name"]))
	name = emptyFallback(name, stringValue(raw["ant_name"]))
	status := emptyFallback(stringValue(raw["status"]), stringValue(raw["result_status"]))
	summary := emptyFallback(stringValue(raw["summary"]), stringValue(raw["message"]))
	artifacts := []string{}
	for _, field := range []string{"outputs", "files_created", "files_modified", "tests_written"} {
		artifacts = append(artifacts, stringSliceValue(raw[field])...)
	}
	dispatch := ceremonyDispatch{
		Name:          name,
		Caste:         stringValue(raw["caste"]),
		Task:          emptyFallback(stringValue(raw["task"]), stringValue(raw["goal"])),
		TaskID:        stringValue(raw["task_id"]),
		Stage:         stringValue(raw["stage"]),
		Status:        status,
		Summary:       summary,
		Wave:          intValue(raw["wave"]),
		ExecutionWave: intValue(raw["execution_wave"]),
		ToolCount:     intValue(raw["tool_count"]),
		Blockers:      stringSliceValue(raw["blockers"]),
		Artifacts:     uniqueSortedStrings(artifacts),
	}
	if dispatch.ExecutionWave <= 0 {
		dispatch.ExecutionWave = dispatch.Wave
	}
	if dispatch.ExecutionWave <= 0 {
		dispatch.ExecutionWave = 1
	}
	if dispatch.Wave <= 0 {
		dispatch.Wave = dispatch.ExecutionWave
	}
	return dispatch
}

func ceremonyExecutionPlansFromManifest(manifest map[string]interface{}, dispatches []ceremonyDispatch) []ceremonyExecutionPlan {
	rawPlans := mapSliceValue(manifest["execution_plan"])
	plans := make([]ceremonyExecutionPlan, 0, len(rawPlans))
	for _, raw := range rawPlans {
		executionWave := intValue(raw["execution_wave"])
		if executionWave <= 0 {
			executionWave = intValue(raw["wave"])
		}
		if executionWave <= 0 {
			executionWave = len(plans) + 1
		}
		wave := intValue(raw["wave"])
		if wave <= 0 {
			wave = executionWave
		}
		plans = append(plans, ceremonyExecutionPlan{
			ExecutionWave: executionWave,
			Wave:          wave,
			Stage:         stringValue(raw["stage"]),
			Strategy:      stringValue(raw["strategy"]),
			WorkerCount:   intValue(raw["worker_count"]),
			Reason:        stringValue(raw["reason"]),
		})
	}
	if len(plans) > 0 {
		return plans
	}
	seen := map[int]bool{}
	for _, dispatch := range dispatches {
		if seen[dispatch.ExecutionWave] {
			continue
		}
		seen[dispatch.ExecutionWave] = true
		plans = append(plans, ceremonyExecutionPlan{
			ExecutionWave: dispatch.ExecutionWave,
			Wave:          dispatch.Wave,
			Stage:         dispatch.Stage,
			Strategy:      "parallel",
		})
	}
	sort.Slice(plans, func(i, j int) bool { return plans[i].ExecutionWave < plans[j].ExecutionWave })
	return plans
}

func ceremonyDispatchesForExecutionWave(dispatches []ceremonyDispatch, plans []ceremonyExecutionPlan, executionWave int) ([]ceremonyDispatch, ceremonyExecutionPlan) {
	var plan ceremonyExecutionPlan
	for _, candidate := range plans {
		if candidate.ExecutionWave == executionWave {
			plan = candidate
			break
		}
	}
	filtered := []ceremonyDispatch{}
	for _, dispatch := range dispatches {
		if dispatch.ExecutionWave == executionWave {
			filtered = append(filtered, dispatch)
		}
	}
	return filtered, plan
}

func dispatchesForPlan(dispatches []ceremonyDispatch, plan ceremonyExecutionPlan) []ceremonyDispatch {
	out := []ceremonyDispatch{}
	for _, dispatch := range dispatches {
		if dispatch.ExecutionWave == plan.ExecutionWave {
			out = append(out, dispatch)
		}
	}
	return out
}

func writeCeremonyPhaseLine(b *strings.Builder, manifest map[string]interface{}) {
	phase := intValue(manifest["phase"])
	phaseName := strings.TrimSpace(stringValue(manifest["phase_name"]))
	if phase <= 0 && phaseName == "" {
		return
	}
	if phase > 0 {
		fmt.Fprintf(b, "Phase %d", phase)
		if phaseName != "" {
			b.WriteString(": ")
			b.WriteString(phaseName)
		}
		b.WriteString("\n\n")
		return
	}
	fmt.Fprintf(b, "%s\n\n", phaseName)
}

func writeCeremonyDispatchLine(b *strings.Builder, dispatch ceremonyDispatch, prefix string) {
	b.WriteString(prefix)
	b.WriteString(casteIdentity(dispatch.Caste))
	b.WriteString(" ")
	b.WriteString(emptyFallback(dispatch.Name, "worker"))
	if dispatch.TaskID != "" {
		b.WriteString("  Task ")
		b.WriteString(dispatch.TaskID)
	}
	if dispatch.Task != "" {
		b.WriteString("  ")
		b.WriteString(dispatch.Task)
	}
	if dispatch.Status != "" {
		b.WriteString(" ")
		b.WriteString(dispatchStatusIcon(normalizeRuntimeDispatchStatus(dispatch.Status)))
	}
	b.WriteString("\n")
}

func ceremonyPlanLabel(plan ceremonyExecutionPlan) string {
	stage := strings.ToLower(strings.TrimSpace(plan.Stage))
	strategy := strings.ToLower(strings.TrimSpace(plan.Strategy))
	if strategy == "" {
		strategy = "parallel"
	}
	switch stage {
	case "wave":
		if plan.Wave > 0 {
			return fmt.Sprintf("Wave %d — %s", plan.Wave, strings.Title(strategy))
		}
	case "verification", "review":
		return "Verification — " + strings.Title(strategy)
	case "integration":
		return "Integration — " + strings.Title(strategy)
	case "prep", "research", "design":
		return strings.Title(stage) + " — " + strings.Title(strategy)
	}
	if plan.ExecutionWave > 0 {
		return fmt.Sprintf("Execution Wave %d — %s", plan.ExecutionWave, strings.Title(strategy))
	}
	return "Worker Wave — " + strings.Title(strategy)
}

func renderOldStyleCeremonyHeader(emoji, title string) string {
	return fmt.Sprintf("━━━ %s %s ━━━\n", emoji, spacedTitle(title))
}

func ceremonyCasteCounts(dispatches []ceremonyDispatch) map[string]int {
	counts := map[string]int{}
	for _, dispatch := range dispatches {
		caste := normalizeCasteKey(dispatch.Caste)
		if caste == "" {
			caste = "worker"
		}
		counts[caste]++
	}
	return counts
}

func ceremonyCasteCountSummary(dispatches []ceremonyDispatch) string {
	counts := ceremonyCasteCounts(dispatches)
	return ceremonyInlineCasteCounts(counts)
}

func ceremonyInlineCasteCounts(counts map[string]int) string {
	castes := make([]string, 0, len(counts))
	for caste := range counts {
		castes = append(castes, caste)
	}
	sort.Strings(castes)
	parts := make([]string, 0, len(castes))
	for _, caste := range castes {
		parts = append(parts, fmt.Sprintf("%d %s", counts[caste], pluralizeCaste(caste, counts[caste])))
	}
	return strings.Join(parts, " + ")
}

func dominantCeremonyCaste(counts map[string]int) (string, int) {
	bestCaste := "worker"
	bestCount := 0
	for caste, count := range counts {
		if count > bestCount || (count == bestCount && caste < bestCaste) {
			bestCaste = caste
			bestCount = count
		}
	}
	return bestCaste, bestCount
}

func pluralizeCaste(caste string, count int) string {
	label := casteLabel(caste)
	if count == 1 {
		return label
	}
	if strings.HasSuffix(label, "s") {
		return label
	}
	return label + "s"
}

func normalizedCeremonyWorkflow(workflow string) string {
	workflow = strings.ToLower(strings.TrimSpace(workflow))
	if workflow == "" {
		return "build"
	}
	return workflow
}
