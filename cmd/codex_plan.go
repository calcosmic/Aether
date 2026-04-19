package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/agent"
	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
)

type codexPlanningDispatch struct {
	Caste    string   `json:"caste"`
	Name     string   `json:"name"`
	Task     string   `json:"task"`
	Outputs  []string `json:"outputs"`
	Status   string   `json:"status"`
	Duration float64  `json:"duration,omitempty"` // Wall-clock seconds (0 = not measured)
	Claimed  []string `json:"-"`
}

type codexSurveyContext struct {
	SurveyDir        string
	SurveyDocs       []string
	Languages        []string
	Frameworks       []string
	Directories      []string
	EntryPoints      []string
	Dependencies     []string
	TestFiles        []string
	Issues           []string
	SecurityPatterns []string
}

type codexScoutFinding struct {
	Area      string `json:"area"`
	Discovery string `json:"discovery"`
	Source    string `json:"source"`
}

type codexScoutReport struct {
	Findings   []codexScoutFinding `json:"findings"`
	Gaps       []string            `json:"gaps"`
	Confidence int                 `json:"confidence"`
	StudyFiles []string            `json:"study_files"`
}

type codexPlanConfidence struct {
	Knowledge    int `json:"knowledge"`
	Requirements int `json:"requirements"`
	Risks        int `json:"risks"`
	Dependencies int `json:"dependencies"`
	Effort       int `json:"effort"`
	Overall      int `json:"overall"`
}

type codexWorkerPlanArtifact struct {
	Phases     []codexWorkerPlanPhase `json:"phases"`
	Confidence codexPlanConfidence    `json:"confidence"`
	Gaps       []string               `json:"gaps,omitempty"`
}

type codexWorkerPlanPhase struct {
	Name            string                `json:"name"`
	Description     string                `json:"description"`
	Tasks           []codexWorkerPlanTask `json:"tasks"`
	SuccessCriteria []string              `json:"success_criteria,omitempty"`
}

type codexWorkerPlanTask struct {
	Goal            string   `json:"goal"`
	Constraints     []string `json:"constraints,omitempty"`
	Hints           []string `json:"hints,omitempty"`
	SuccessCriteria []string `json:"success_criteria,omitempty"`
	DependsOn       []string `json:"depends_on,omitempty"`
}

type phaseTemplate struct {
	Name            string
	Description     string
	Tasks           []phaseTaskTemplate
	SuccessCriteria []string
}

type phaseTaskTemplate struct {
	Goal            string
	Constraints     []string
	Hints           []string
	SuccessCriteria []string
	DependsOn       []string
}

func runCodexPlan(root string, refresh bool) (map[string]interface{}, error) {
	if store == nil {
		return nil, fmt.Errorf("no store initialized")
	}

	state, err := loadActiveColonyState()
	if err != nil {
		return nil, fmt.Errorf("%s", colonyStateLoadMessage(err))
	}

	granularity := normalizedGranularity(state.PlanGranularity)
	pending := loadPendingDecisionFile()
	unresolvedClarifications := countPendingClarifications(pending)
	clarificationWarning := ""
	if unresolvedClarifications > 0 {
		clarificationWarning = "Unresolved clarifications exist. Run `aether discuss` to resolve them before planning, or proceed with implicit assumptions."
	}

	if len(state.Plan.Phases) > 0 && !refresh {
		nextPhase := firstBuildablePhase(state.Plan.Phases)
		nextCommand := "aether build 1"
		if nextPhase > 0 {
			nextCommand = fmt.Sprintf("aether build %d", nextPhase)
		}
		updateSessionSummary("plan", nextCommand, fmt.Sprintf("Loaded existing plan (%d phases)", len(state.Plan.Phases)))
		return map[string]interface{}{
			"planned":                   true,
			"existing_plan":             true,
			"goal":                      *state.Goal,
			"phases":                    state.Plan.Phases,
			"count":                     len(state.Plan.Phases),
			"granularity":               string(granularity),
			"unresolved_clarifications": unresolvedClarifications,
			"clarification_warning":     clarificationWarning,
			"next":                      nextCommand,
		}, nil
	}

	if refresh && state.CurrentPhase > 0 {
		return nil, fmt.Errorf("cannot refresh the plan while phase %d is already active", state.CurrentPhase)
	}

	survey, err := loadCodexSurveyContext(root)
	if err != nil {
		return nil, err
	}

	planningDir := filepath.Join(store.BasePath(), "planning")
	phaseResearchDir := filepath.Join(store.BasePath(), "phase-research")
	if err := os.MkdirAll(planningDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create planning directory: %w", err)
	}
	if err := os.MkdirAll(phaseResearchDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create phase research directory: %w", err)
	}
	artifactSnapshots := snapshotRelativeFiles(root,
		filepath.ToSlash(filepath.Join(".aether", "data", "planning")),
		filepath.ToSlash(filepath.Join(".aether", "data", "phase-research")),
	)

	dispatches := plannedPlanningWorkers(root)
	dispatchMode := "synthetic"
	artifactSource := "local-synthesis"
	planSource := "local-synthesis"
	spawnTree := agent.NewSpawnTree(store, "spawn-tree.txt")
	for _, dispatch := range dispatches {
		if err := spawnTree.RecordSpawn("Queen", dispatch.Caste, dispatch.Name, dispatch.Task, 1); err != nil {
			return nil, fmt.Errorf("failed to record planning spawn: %w", err)
		}
	}

	invoker := newCodexWorkerInvoker()
	if _, ok := invoker.(*codex.FakeInvoker); !ok && !invoker.IsAvailable(context.Background()) {
		return nil, fmt.Errorf("codex CLI is not available in PATH")
	}
	emitVisualProgress(renderPlanDispatchPreview(*state.Goal, dispatches))
	realDispatches, dispatchErr := dispatchRealPlanningWorkers(context.Background(), root, invoker)
	if dispatchErr != nil {
		if _, ok := invoker.(*codex.FakeInvoker); !ok {
			for _, dispatch := range dispatches {
				_ = spawnTree.UpdateStatus(dispatch.Name, "failed", dispatchErr.Error())
			}
			return nil, dispatchErr
		}
	} else if realDispatches != nil {
		dispatches = realDispatches
		if _, ok := invoker.(*codex.FakeInvoker); ok {
			dispatchMode = "simulated"
		} else {
			dispatchMode = "real"
		}
	}

	scoutReport := synthesizeScoutPlanningReport(*state.Goal, survey)
	scoutFile, preservedScoutArtifact, err := writePlanningScoutArtifact(root, planningDir, *state.Goal, granularity, survey, dispatches[0], scoutReport, artifactSnapshots)
	if err != nil {
		return nil, err
	}

	phases, confidence, unresolvedGaps := synthesizeRouteSetterPlan(*state.Goal, granularity, survey, scoutReport)
	if workerPlan, ok, note := loadWorkerPlanArtifact(root, artifactSnapshots, dispatches); ok {
		phases = buildWorkerPlanPhases(workerPlan)
		confidence = mergePlanConfidence(confidence, workerPlan.Confidence)
		unresolvedGaps = limitStrings(uniqueSortedStrings(append(unresolvedGaps, workerPlan.Gaps...)), 4)
		planSource = "worker-artifact"
	} else if note != "" {
		unresolvedGaps = limitStrings(uniqueSortedStrings(append(unresolvedGaps, note)), 4)
	}
	routeSetterFile, preservedRouteArtifact, err := writeRouteSetterArtifact(root, planningDir, *state.Goal, granularity, survey, dispatches[1], confidence, unresolvedGaps, phases, artifactSnapshots)
	if err != nil {
		return nil, err
	}
	planArtifactFile, preservedPlanArtifact, err := writeWorkerPlanArtifact(root, planningDir, confidence, unresolvedGaps, phases, artifactSnapshots, dispatches)
	if err != nil {
		return nil, err
	}
	phaseResearchFiles, preservedResearchArtifacts, err := writePhaseResearchArtifacts(root, phaseResearchDir, survey, scoutReport, phases, artifactSnapshots, dispatches)
	if err != nil {
		return nil, err
	}
	if preservedScoutArtifact || preservedRouteArtifact || preservedPlanArtifact || preservedResearchArtifacts > 0 {
		artifactSource = "worker-written"
	}

	for i := range dispatches {
		status := dispatches[i].Status
		if strings.TrimSpace(status) == "" || status == "spawned" {
			status = "completed"
		}
		summary := strings.Join(dispatches[i].Outputs, ", ")
		if dispatchMode != "real" {
			summary = "Local planning synthesis fallback"
		}
		if err := spawnTree.UpdateStatus(dispatches[i].Name, status, summary); err != nil {
			return nil, fmt.Errorf("failed to update planning completion: %w", err)
		}
	}

	now := time.Now().UTC()
	state.State = colony.StateREADY
	state.CurrentPhase = firstBuildablePhase(phases)
	state.BuildStartedAt = nil
	state.PlanGranularity = granularity
	planConfidence := float64(confidence.Overall) / 100.0
	state.Plan = colony.Plan{
		GeneratedAt: &now,
		Confidence:  &planConfidence,
		Phases:      phases,
	}
	state.Events = append(trimmedEvents(state.Events),
		fmt.Sprintf("%s|planning_scout|plan|Scout summarized surveyed repo context", now.Format(time.RFC3339)),
		fmt.Sprintf("%s|plan_generated|plan|Generated %d phases with %d%% confidence", now.Format(time.RFC3339), len(phases), confidence.Overall),
	)
	if err := store.SaveJSON("COLONY_STATE.json", state); err != nil {
		return nil, fmt.Errorf("failed to save colony state: %w", err)
	}

	nextPhase := firstBuildablePhase(phases)
	nextCommand := "aether build 1"
	if nextPhase > 0 {
		nextCommand = fmt.Sprintf("aether build %d", nextPhase)
	}
	updateSessionSummary("plan", nextCommand, fmt.Sprintf("Generated %d plan phases with %d%% confidence", len(phases), confidence.Overall))

	dispatchMaps := make([]map[string]interface{}, 0, len(dispatches))
	for _, dispatch := range dispatches {
		entry := map[string]interface{}{
			"caste":   dispatch.Caste,
			"name":    dispatch.Name,
			"task":    dispatch.Task,
			"outputs": dispatch.Outputs,
			"status":  dispatch.Status,
		}
		if dispatch.Duration > 0 {
			entry["duration"] = dispatch.Duration
		}
		dispatchMaps = append(dispatchMaps, entry)
	}

	return map[string]interface{}{
		"planned":                   true,
		"existing_plan":             false,
		"refreshed":                 refresh,
		"goal":                      *state.Goal,
		"phases":                    phases,
		"count":                     len(phases),
		"granularity":               string(granularity),
		"granularity_min":           granularityMin(granularity),
		"granularity_max":           granularityMax(granularity),
		"confidence":                confidence,
		"planning_dir":              planningDir,
		"planning_files":            []string{filepath.Base(scoutFile), filepath.Base(routeSetterFile)},
		"plan_artifact":             filepath.Base(planArtifactFile),
		"phase_research_dir":        phaseResearchDir,
		"phase_research_files":      phaseResearchFiles,
		"dispatches":                dispatchMaps,
		"dispatch_mode":             dispatchMode,
		"artifact_source":           artifactSource,
		"plan_source":               planSource,
		"gaps":                      unresolvedGaps,
		"survey_docs":               survey.SurveyDocs,
		"unresolved_clarifications": unresolvedClarifications,
		"clarification_warning":     clarificationWarning,
		"next":                      nextCommand,
	}, nil
}

func normalizedGranularity(value colony.PlanGranularity) colony.PlanGranularity {
	if value.Valid() {
		return value
	}
	return colony.GranularityMilestone
}

func granularityMin(value colony.PlanGranularity) int {
	min, _ := colony.GranularityRange(value)
	return min
}

func granularityMax(value colony.PlanGranularity) int {
	_, max := colony.GranularityRange(value)
	return max
}

func firstBuildablePhase(phases []colony.Phase) int {
	for _, phase := range phases {
		if phase.Status != colony.PhaseCompleted {
			if phase.ID > 0 {
				return phase.ID
			}
		}
	}
	if len(phases) > 0 {
		return phases[0].ID
	}
	return 0
}

func plannedPlanningWorkers(root string) []codexPlanningDispatch {
	return []codexPlanningDispatch{
		{
			Caste:   "scout",
			Name:    deterministicAntName("scout", root+"|plan|scout"),
			Task:    "Survey the repo and distill planning findings from available territory reports",
			Outputs: []string{"SCOUT.md"},
			Status:  "spawned",
		},
		{
			Caste:   "route_setter",
			Name:    deterministicAntName("route_setter", root+"|plan|route-setter"),
			Task:    "Convert surveyed findings into an executable multi-phase colony plan",
			Outputs: []string{"ROUTE-SETTER.md"},
			Status:  "spawned",
		},
	}
}

// planningWorkerSpec defines a single planning worker for real dispatch.
type planningWorkerSpec struct {
	Caste     string // Worker caste (scout, route_setter)
	AgentFile string // TOML filename (e.g., "aether-scout.toml")
	Task      string // Task brief
	Outputs   []string
}

// planningWorkerSpecs is the canonical list of planning workers, matching plannedPlanningWorkers order.
var planningWorkerSpecs = []planningWorkerSpec{
	{
		Caste:     "scout",
		AgentFile: "aether-scout.toml",
		Task:      "Survey the repo and distill planning findings from available territory reports",
		Outputs:   []string{"SCOUT.md"},
	},
	{
		Caste:     "route_setter",
		AgentFile: "aether-route-setter.toml",
		Task:      "Convert surveyed findings into an executable multi-phase colony plan",
		Outputs:   []string{"ROUTE-SETTER.md"},
	},
}

// dispatchRealPlanningWorkers attempts real worker invocation for planning.
// If the invoker is not available, it returns nil, nil (caller falls back to plannedPlanningWorkers).
func dispatchRealPlanningWorkers(ctx context.Context, root string, invoker codex.WorkerInvoker) ([]codexPlanningDispatch, error) {
	if invoker == nil || !invoker.IsAvailable(ctx) {
		return nil, nil
	}

	codexAgentsDir := filepath.Join(root, ".codex", "agents")

	dispatches := make([]codex.WorkerDispatch, 0, len(planningWorkerSpecs))
	capsule := resolveCodexWorkerContext()
	pheromoneSection := resolvePheromoneSection()
	for i, spec := range planningWorkerSpecs {
		tomlPath := filepath.Join(codexAgentsDir, spec.AgentFile)

		planned := plannedPlanningWorkers(root)
		workerName := planned[i].Name

		taskBrief := renderPlanningWorkerBrief(root, spec)

		dispatches = append(dispatches, codex.WorkerDispatch{
			ID:               fmt.Sprintf("planning-%d", i),
			WorkerName:       workerName,
			AgentName:        strings.TrimSuffix(spec.AgentFile, ".toml"),
			AgentTOMLPath:    tomlPath,
			Caste:            spec.Caste,
			TaskID:           fmt.Sprintf("plan-%d", i),
			TaskBrief:        taskBrief,
			ContextCapsule:   capsule,
			SkillSection:     resolveSkillSection(spec.Caste, spec.Task),
			PheromoneSection: pheromoneSection,
			Root:             root,
			Wave:             1,
		})
	}

	results, err := codex.DispatchBatch(ctx, invoker, dispatches)
	if err != nil {
		return nil, err
	}
	for _, result := range results {
		if result.Status != "completed" {
			return convertPlanningDispatchResults(results, root), fmt.Errorf("planning worker %s did not complete: %s", result.WorkerName, result.Status)
		}
	}

	return convertPlanningDispatchResults(results, root), nil
}

// convertPlanningDispatchResults maps a slice of DispatchResult to codexPlanningDispatch.
// "timeout" status is mapped to "failed". If results don't cover all specs,
// remaining specs get the planned defaults.
func convertPlanningDispatchResults(results []codex.DispatchResult, root string) []codexPlanningDispatch {
	planned := plannedPlanningWorkers(root)
	dispatches := make([]codexPlanningDispatch, 0, len(planned))

	for i, planned := range planned {
		d := codexPlanningDispatch{
			Caste:   planned.Caste,
			Name:    planned.Name,
			Task:    planned.Task,
			Outputs: planned.Outputs,
			Status:  "spawned",
		}

		if i < len(results) {
			r := results[i]
			if r.WorkerName != "" {
				d.Name = r.WorkerName
			}
			switch r.Status {
			case "completed":
				d.Status = "completed"
			case "timeout":
				d.Status = "failed"
			case "failed":
				d.Status = "failed"
			default:
				d.Status = "failed"
			}
			if r.WorkerResult != nil {
				d.Duration = r.WorkerResult.Duration.Seconds()
				d.Claimed = append(d.Claimed, r.WorkerResult.FilesCreated...)
				d.Claimed = append(d.Claimed, r.WorkerResult.FilesModified...)
				d.Claimed = uniqueSortedStrings(d.Claimed)
			}
		}

		dispatches = append(dispatches, d)
	}

	return dispatches
}

func loadCodexSurveyContext(root string) (codexSurveyContext, error) {
	surveyDir := ""
	if store != nil {
		surveyDir = filepath.Join(store.BasePath(), "survey")
	}
	ctx := codexSurveyContext{
		SurveyDir:        surveyDir,
		SurveyDocs:       []string{},
		Languages:        []string{},
		Frameworks:       []string{},
		Directories:      []string{},
		EntryPoints:      []string{},
		Dependencies:     []string{},
		TestFiles:        []string{},
		Issues:           []string{},
		SecurityPatterns: []string{},
	}

	for _, name := range []string{"PROVISIONS.md", "TRAILS.md", "BLUEPRINT.md", "CHAMBERS.md", "DISCIPLINES.md", "SENTINEL-PROTOCOLS.md", "PATHOGENS.md"} {
		if surveyDir == "" {
			break
		}
		if _, err := os.Stat(filepath.Join(surveyDir, name)); err == nil {
			ctx.SurveyDocs = append(ctx.SurveyDocs, name)
		}
	}

	readSummary := func(file string) map[string]interface{} {
		if surveyDir == "" {
			return nil
		}
		data, err := os.ReadFile(filepath.Join(surveyDir, file))
		if err != nil {
			return nil
		}
		var payload map[string]interface{}
		if err := json.Unmarshal(data, &payload); err != nil {
			return nil
		}
		return payload
	}

	if payload := readSummary("provisions.json"); payload != nil {
		ctx.Languages = append(ctx.Languages, jsonStringSlice(payload["languages"])...)
		ctx.Dependencies = append(ctx.Dependencies, jsonStringSlice(payload["dependencies"])...)
	}
	if payload := readSummary("blueprint.json"); payload != nil {
		ctx.Frameworks = append(ctx.Frameworks, jsonStringSlice(payload["frameworks"])...)
		ctx.EntryPoints = append(ctx.EntryPoints, jsonStringSlice(payload["entry_points"])...)
	}
	if payload := readSummary("chambers.json"); payload != nil {
		ctx.Directories = append(ctx.Directories, jsonStringSlice(payload["directories"])...)
	}
	if payload := readSummary("disciplines.json"); payload != nil {
		ctx.TestFiles = append(ctx.TestFiles, jsonStringSlice(payload["tests"])...)
	}
	if payload := readSummary("pathogens.json"); payload != nil {
		ctx.Issues = append(ctx.Issues, jsonStringSlice(payload["issues"])...)
	}

	facts, err := surveyWorkspace(root)
	if err == nil {
		ctx.Languages = append(ctx.Languages, facts.Languages...)
		ctx.Frameworks = append(ctx.Frameworks, facts.Frameworks...)
		ctx.Directories = append(ctx.Directories, facts.TopLevelDirs...)
		ctx.EntryPoints = append(ctx.EntryPoints, facts.EntryPoints...)
		ctx.Dependencies = append(ctx.Dependencies, facts.KeyDependencies...)
		ctx.TestFiles = append(ctx.TestFiles, facts.TestFiles...)
		ctx.SecurityPatterns = append(ctx.SecurityPatterns, facts.SecurityPatterns...)
		if len(ctx.Issues) == 0 {
			ctx.Issues = identifyPathogens(facts)
		}
	}

	ctx.SurveyDocs = uniqueSortedStrings(ctx.SurveyDocs)
	ctx.Languages = uniqueSortedStrings(ctx.Languages)
	ctx.Frameworks = uniqueSortedStrings(ctx.Frameworks)
	ctx.Directories = uniqueSortedStrings(ctx.Directories)
	ctx.EntryPoints = uniqueSortedStrings(ctx.EntryPoints)
	ctx.Dependencies = uniqueSortedStrings(ctx.Dependencies)
	ctx.TestFiles = uniqueSortedStrings(ctx.TestFiles)
	ctx.Issues = uniqueSortedStrings(ctx.Issues)
	ctx.SecurityPatterns = uniqueSortedStrings(ctx.SecurityPatterns)
	return ctx, nil
}

func jsonStringSlice(raw interface{}) []string {
	switch value := raw.(type) {
	case []string:
		return append([]string{}, value...)
	case []interface{}:
		result := make([]string, 0, len(value))
		for _, entry := range value {
			if text, ok := entry.(string); ok && strings.TrimSpace(text) != "" {
				result = append(result, text)
			}
		}
		return result
	default:
		return nil
	}
}

func uniqueSortedStrings(values []string) []string {
	seen := map[string]bool{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func synthesizeScoutPlanningReport(goal string, survey codexSurveyContext) codexScoutReport {
	report := codexScoutReport{
		Findings:   []codexScoutFinding{},
		Gaps:       []string{},
		Confidence: 60,
		StudyFiles: []string{},
	}

	if len(survey.SurveyDocs) > 0 {
		report.Findings = append(report.Findings, codexScoutFinding{
			Area:      "territory survey",
			Discovery: fmt.Sprintf("Existing survey artifacts are available (%s), so planning can build on repo-specific context instead of starting blind.", strings.Join(limitStrings(survey.SurveyDocs, 4), ", ")),
			Source:    filepath.Join(".aether", "data", "survey"),
		})
	}
	if len(survey.EntryPoints) > 0 || len(survey.Directories) > 0 {
		report.Findings = append(report.Findings, codexScoutFinding{
			Area:      "architecture",
			Discovery: fmt.Sprintf("Primary execution surfaces live around %s, with key directories %s.", renderCSV(limitStrings(survey.EntryPoints, 3), "no explicit entry points"), renderCSV(limitStrings(survey.Directories, 4), "no top-level directories")),
			Source:    "BLUEPRINT.md / CHAMBERS.md",
		})
	}
	if len(survey.Languages) > 0 || len(survey.Frameworks) > 0 || len(survey.Dependencies) > 0 {
		report.Findings = append(report.Findings, codexScoutFinding{
			Area:      "stack",
			Discovery: fmt.Sprintf("The implementation surface spans %s with frameworks %s and dependencies such as %s.", renderCSV(limitStrings(survey.Languages, 3), "unknown languages"), renderCSV(limitStrings(survey.Frameworks, 4), "no framework markers"), renderCSV(limitStrings(survey.Dependencies, 5), "no obvious dependency anchors")),
			Source:    "PROVISIONS.md / TRAILS.md",
		})
	}
	if len(survey.TestFiles) > 0 {
		report.Findings = append(report.Findings, codexScoutFinding{
			Area:      "verification",
			Discovery: fmt.Sprintf("Existing test coverage already exercises representative paths like %s.", renderCSV(limitStrings(survey.TestFiles, 4), "no tests")),
			Source:    "DISCIPLINES.md / SENTINEL-PROTOCOLS.md",
		})
	} else {
		report.Gaps = append(report.Gaps, "No obvious test files were detected, so the plan must reserve explicit verification work.")
	}
	if len(survey.Issues) > 0 {
		report.Findings = append(report.Findings, codexScoutFinding{
			Area:      "risk",
			Discovery: fmt.Sprintf("Known repository concerns already include %s.", renderCSV(limitStrings(survey.Issues, 2), "no documented risks")),
			Source:    "PATHOGENS.md",
		})
	}

	if len(survey.SurveyDocs) == 0 {
		report.Gaps = append(report.Gaps, "No territory survey artifacts were present, so the plan relies on direct filesystem inspection only.")
	}
	if len(survey.EntryPoints) == 0 {
		report.Gaps = append(report.Gaps, "No obvious entry points were detected, so implementation ownership must be inferred from directories and tests.")
	}
	if len(survey.Dependencies) == 0 {
		report.Gaps = append(report.Gaps, "Dependency manifests did not provide many anchors, so integration boundaries may still need confirmation during build.")
	}
	report.Gaps = limitStrings(uniqueSortedStrings(report.Gaps), 3)

	report.StudyFiles = append(report.StudyFiles, survey.EntryPoints...)
	report.StudyFiles = append(report.StudyFiles, survey.TestFiles...)
	report.StudyFiles = limitStrings(uniqueSortedStrings(report.StudyFiles), 6)

	confidence := 55 + len(report.Findings)*6 + len(limitStrings(survey.SurveyDocs, 6))*3
	if len(report.StudyFiles) > 0 {
		confidence += 8
	}
	confidence -= len(report.Gaps) * 4
	report.Confidence = clampInt(confidence, 55, 94)
	if len(report.Findings) > 5 {
		report.Findings = append([]codexScoutFinding{}, report.Findings[:5]...)
	}
	return report
}

func limitStrings(values []string, limit int) []string {
	if len(values) <= limit {
		return append([]string{}, values...)
	}
	return append([]string{}, values[:limit]...)
}

func clampInt(value, min, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func renderPlanningWorkerBrief(root string, spec planningWorkerSpec) string {
	planningDir := filepath.ToSlash(filepath.Join(".aether", "data", "planning"))
	phaseResearchDir := filepath.ToSlash(filepath.Join(".aether", "data", "phase-research"))
	primaryOutputs := make([]string, 0, len(spec.Outputs))
	for _, output := range spec.Outputs {
		primaryOutputs = append(primaryOutputs, filepath.ToSlash(filepath.Join(planningDir, output)))
	}

	var b strings.Builder
	b.WriteString("Planning task: ")
	b.WriteString(spec.Task)
	b.WriteString("\n\n")
	b.WriteString("Write planning outputs directly into the repository.\n")
	b.WriteString("- Primary outputs: ")
	b.WriteString(strings.Join(primaryOutputs, ", "))
	b.WriteString("\n")
	b.WriteString("- Planning dir: ")
	b.WriteString(planningDir)
	b.WriteString("\n")
	b.WriteString("- Phase research dir: ")
	b.WriteString(phaseResearchDir)
	b.WriteString("\n")
	if spec.Caste == "route_setter" {
		b.WriteString("- Also write a machine-readable plan artifact at ")
		b.WriteString(filepath.ToSlash(filepath.Join(planningDir, "phase-plan.json")))
		b.WriteString(" using this JSON shape:\n")
		b.WriteString(`  {"phases":[{"name":"","description":"","tasks":[{"goal":"","constraints":[],"hints":[],"success_criteria":[],"depends_on":[]}],"success_criteria":[]}],"confidence":{"knowledge":0,"requirements":0,"risks":0,"dependencies":0,"effort":0,"overall":0},"gaps":[]}` + "\n")
	}
	b.WriteString("\nPlan the colony at ")
	b.WriteString(root)
	return b.String()
}

func claimedPlanningFiles(dispatches []codexPlanningDispatch) map[string]bool {
	claimed := map[string]bool{}
	for _, dispatch := range dispatches {
		for relPath := range claimedArtifactSet(dispatch.Claimed) {
			claimed[relPath] = true
		}
	}
	return claimed
}

func loadWorkerPlanArtifact(root string, snapshots map[string]codexArtifactSnapshot, dispatches []codexPlanningDispatch) (codexWorkerPlanArtifact, bool, string) {
	relPath := filepath.ToSlash(filepath.Join(".aether", "data", "planning", "phase-plan.json"))
	if !shouldPreserveWorkerArtifact(root, relPath, snapshots, claimedPlanningFiles(dispatches)) {
		return codexWorkerPlanArtifact{}, false, ""
	}

	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(relPath)))
	if err != nil {
		return codexWorkerPlanArtifact{}, false, "Route-setter wrote phase-plan.json but it could not be read, so planning fell back to local synthesis."
	}

	var artifact codexWorkerPlanArtifact
	if err := json.Unmarshal(data, &artifact); err != nil {
		return codexWorkerPlanArtifact{}, false, "Route-setter phase-plan.json was invalid, so planning fell back to local synthesis."
	}
	if len(artifact.Phases) == 0 {
		return codexWorkerPlanArtifact{}, false, "Route-setter phase-plan.json contained no phases, so planning fell back to local synthesis."
	}
	return artifact, true, ""
}

func buildWorkerPlanPhases(artifact codexWorkerPlanArtifact) []colony.Phase {
	phases := make([]colony.Phase, 0, len(artifact.Phases))
	for i, sourcePhase := range artifact.Phases {
		phase := colony.Phase{
			ID:              i + 1,
			Name:            strings.TrimSpace(sourcePhase.Name),
			Description:     strings.TrimSpace(sourcePhase.Description),
			Status:          colony.PhasePending,
			Tasks:           []colony.Task{},
			SuccessCriteria: uniqueSortedStrings(sourcePhase.SuccessCriteria),
		}
		if phase.Name == "" {
			phase.Name = fmt.Sprintf("Phase %d", i+1)
		}
		if i == 0 {
			phase.Status = colony.PhaseReady
		}
		for j, sourceTask := range sourcePhase.Tasks {
			goal := strings.TrimSpace(sourceTask.Goal)
			if goal == "" {
				continue
			}
			taskID := fmt.Sprintf("%d.%d", i+1, j+1)
			phase.Tasks = append(phase.Tasks, colony.Task{
				ID:              &taskID,
				Goal:            goal,
				Status:          colony.TaskPending,
				Constraints:     uniqueSortedStrings(sourceTask.Constraints),
				Hints:           uniqueSortedStrings(sourceTask.Hints),
				SuccessCriteria: uniqueSortedStrings(sourceTask.SuccessCriteria),
				DependsOn:       uniqueSortedStrings(sourceTask.DependsOn),
			})
		}
		phases = append(phases, phase)
	}
	return phases
}

func mergePlanConfidence(base codexPlanConfidence, override codexPlanConfidence) codexPlanConfidence {
	if override.Knowledge > 0 {
		base.Knowledge = override.Knowledge
	}
	if override.Requirements > 0 {
		base.Requirements = override.Requirements
	}
	if override.Risks > 0 {
		base.Risks = override.Risks
	}
	if override.Dependencies > 0 {
		base.Dependencies = override.Dependencies
	}
	if override.Effort > 0 {
		base.Effort = override.Effort
	}
	if override.Overall > 0 {
		base.Overall = override.Overall
	} else {
		base.Overall = int(float64(base.Knowledge)*0.25 +
			float64(base.Requirements)*0.25 +
			float64(base.Risks)*0.20 +
			float64(base.Dependencies)*0.15 +
			float64(base.Effort)*0.15 + 0.5)
	}
	return base
}

func writePlanningScoutArtifact(root, planningDir, goal string, granularity colony.PlanGranularity, survey codexSurveyContext, dispatch codexPlanningDispatch, report codexScoutReport, snapshots map[string]codexArtifactSnapshot) (string, bool, error) {
	path := filepath.Join(planningDir, "SCOUT.md")
	relPath := filepath.ToSlash(filepath.Join(".aether", "data", "planning", "SCOUT.md"))
	if shouldPreserveWorkerArtifact(root, relPath, snapshots, claimedArtifactSet(dispatch.Claimed)) {
		return path, true, nil
	}
	var b strings.Builder
	b.WriteString("# Planning Scout Report\n\n")
	b.WriteString(fmt.Sprintf("- Generated: %s\n", time.Now().UTC().Format(time.RFC3339)))
	b.WriteString(fmt.Sprintf("- Scout: %s\n", dispatch.Name))
	b.WriteString(fmt.Sprintf("- Goal: %s\n", goal))
	b.WriteString(fmt.Sprintf("- Granularity: %s\n\n", granularity))
	b.WriteString("## Findings\n")
	for _, finding := range report.Findings {
		b.WriteString(fmt.Sprintf("- **%s:** %s (Source: %s)\n", finding.Area, finding.Discovery, finding.Source))
	}
	if len(report.Findings) == 0 {
		b.WriteString("- No significant findings were synthesized.\n")
	}
	b.WriteString("\n## Gaps\n")
	b.WriteString(bulletList(report.Gaps, "No material knowledge gaps remain at planning time."))
	b.WriteString("\n\n## Study Files\n")
	b.WriteString(bulletList(report.StudyFiles, "No representative files were identified."))
	b.WriteString("\n\n## Survey Inputs\n")
	b.WriteString(bulletList(survey.SurveyDocs, "No territory survey docs were available."))
	b.WriteString("\n")
	if err := os.WriteFile(path, []byte(b.String()), 0644); err != nil {
		return "", false, fmt.Errorf("failed to write planning scout report: %w", err)
	}
	return path, false, nil
}

func synthesizeRouteSetterPlan(goal string, granularity colony.PlanGranularity, survey codexSurveyContext, report codexScoutReport) ([]colony.Phase, codexPlanConfidence, []string) {
	templates := planningTemplates(goal, survey, report)
	minPhases, maxPhases := colony.GranularityRange(granularity)
	count := len(templates)
	if count < minPhases {
		count = minPhases
	}
	if count > maxPhases {
		count = maxPhases
	}
	if len(templates) > count {
		templates = append([]phaseTemplate{}, templates[:count]...)
	}

	phases := make([]colony.Phase, 0, len(templates))
	for i, template := range templates {
		phase := colony.Phase{
			ID:              i + 1,
			Name:            template.Name,
			Description:     template.Description,
			Status:          colony.PhasePending,
			Tasks:           []colony.Task{},
			SuccessCriteria: append([]string{}, template.SuccessCriteria...),
		}
		if i == 0 {
			phase.Status = colony.PhaseReady
		}
		for j, taskTemplate := range template.Tasks {
			taskID := fmt.Sprintf("%d.%d", i+1, j+1)
			phase.Tasks = append(phase.Tasks, colony.Task{
				ID:              &taskID,
				Goal:            taskTemplate.Goal,
				Status:          colony.TaskPending,
				Constraints:     uniqueSortedStrings(taskTemplate.Constraints),
				Hints:           uniqueSortedStrings(taskTemplate.Hints),
				SuccessCriteria: uniqueSortedStrings(taskTemplate.SuccessCriteria),
				DependsOn:       uniqueSortedStrings(taskTemplate.DependsOn),
			})
		}
		phases = append(phases, phase)
	}

	confidence := codexPlanConfidence{
		Knowledge:    clampInt(report.Confidence, 55, 96),
		Requirements: clampInt(70+len(templates)*2, 68, 94),
		Risks:        clampInt(88-len(survey.Issues)*4-len(report.Gaps)*5, 55, 90),
		Dependencies: clampInt(60+len(survey.EntryPoints)*5+len(survey.Dependencies)*2, 58, 92),
		Effort:       clampInt(80-len(templates), 62, 88),
	}
	confidence.Overall = int(float64(confidence.Knowledge)*0.25 +
		float64(confidence.Requirements)*0.25 +
		float64(confidence.Risks)*0.20 +
		float64(confidence.Dependencies)*0.15 +
		float64(confidence.Effort)*0.15 + 0.5)

	unresolved := append([]string{}, report.Gaps...)
	if len(survey.Issues) > 0 {
		unresolved = append(unresolved, fmt.Sprintf("Known repo risks remain active: %s.", renderCSV(limitStrings(survey.Issues, 2), "none")))
	}
	return phases, confidence, limitStrings(uniqueSortedStrings(unresolved), 3)
}

func planningTemplates(goal string, survey codexSurveyContext, report codexScoutReport) []phaseTemplate {
	goalLower := strings.ToLower(goal)
	switch {
	case containsAny(goalLower, []string{"parity", "orchestrat", "workflow", "command", "spawn"}):
		return []phaseTemplate{
			{
				Name:        "Contract and gap mapping",
				Description: "Lock the expected ant-process behavior against the current Go command surface.",
				Tasks: []phaseTaskTemplate{
					{
						Goal:            "Compare the documented ant workflow with the current Codex command behavior",
						Constraints:     []string{"Use Claude/OpenCode command specs as the external contract", "Keep the Go binary as the execution source of truth"},
						Hints:           []string{".claude/commands/ant/colonize.md", ".claude/commands/ant/plan.md", "cmd/codex_workflow_cmds.go"},
						SuccessCriteria: []string{"The parity gaps are explicit", "The next implementation slices are dependency-ordered"},
					},
					{
						Goal:            "Decide the observable ant-process outputs Codex must emit during each core command",
						Constraints:     []string{"Do not claim capabilities the Go binary does not actually perform"},
						Hints:           append(commonHints(survey), report.StudyFiles...),
						SuccessCriteria: []string{"Each command has a concrete dispatch contract", "Spawn tree and artifacts are part of the contract"},
					},
				},
				SuccessCriteria: []string{"The parity contract is explicit", "The colony has an honest execution target"},
			},
			{
				Name:        "Colonize orchestration",
				Description: "Deliver surveyor-driven territory mapping that writes usable survey artifacts and spawn records.",
				Tasks: []phaseTaskTemplate{
					{
						Goal:            "Implement the surveyor workflow for Codex colonize",
						Constraints:     commonConstraints(survey),
						Hints:           []string{"cmd/codex_colonize.go", "spawn-tree.txt", ".aether/data/survey/"},
						SuccessCriteria: []string{"Survey artifacts are written", "Surveyor spawns are recorded and completed"},
					},
					{
						Goal:            "Keep survey artifacts stable enough for planning to consume",
						Constraints:     []string{"Avoid polluting reports with cache or generated artifacts"},
						Hints:           []string{"PATHOGENS.md should only mention real repo concerns"},
						SuccessCriteria: []string{"Survey artifacts are reusable inputs for plan", "Noise from generated/cache files is excluded"},
					},
				},
				SuccessCriteria: []string{"Territory survey is reliable", "Planning can trust the survey output"},
			},
			{
				Name:        "Planning orchestration",
				Description: "Add a scout plus route-setter planning pass that produces artifacts and a grounded phase plan.",
				Tasks: []phaseTaskTemplate{
					{
						Goal:            "Implement a scout planning pass that summarizes survey findings into buildable guidance",
						Constraints:     []string{"Scout output must be persisted as a planning artifact", "Planning must still work when survey docs are missing"},
						Hints:           []string{"SCOUT.md", "phase-research/", "cmd/codex_plan.go"},
						SuccessCriteria: []string{"Planning produces a scout artifact", "The route-setter consumes scout output and survey context"},
					},
					{
						Goal:            "Generate a route-setter plan with task constraints, hints, and success criteria",
						Constraints:     []string{"The first phase must become ready", "The saved plan must match the displayed plan"},
						Hints:           []string{"COLONY_STATE.json", "renderPlanVisual"},
						SuccessCriteria: []string{"Plan generation is grounded in repo context", "Spawn records show scout and route-setter activity"},
					},
				},
				SuccessCriteria: []string{"Plan generation is ant-driven", "The colony has a grounded next phase"},
			},
			{
				Name:        "Build orchestration",
				Description: "Replace visual-only build dispatch with real worker execution slices and artifact handling.",
				Tasks: []phaseTaskTemplate{
					{
						Goal:            "Define and implement the builder, watcher, and specialist work sequence for build",
						Constraints:     []string{"Keep command behavior inside Go", "Recorded spawns must reflect real work performed"},
						Hints:           []string{"cmd/codex_workflow_cmds.go", "cmd/codex_visuals.go", ".claude/commands/ant/build.md"},
						SuccessCriteria: []string{"Build does more than set state", "The spawn plan matches actual command behavior"},
					},
					{
						Goal:            "Persist build artifacts and phase context needed by continue",
						Constraints:     []string{"Continue must not rely on implied work that never happened"},
						Hints:           commonHints(survey),
						SuccessCriteria: []string{"Continue can verify concrete outputs", "Phase state stays consistent across reruns"},
					},
				},
				SuccessCriteria: []string{"Build behavior matches its visuals", "Continue has real evidence to consume"},
			},
			{
				Name:        "Continue orchestration",
				Description: "Make continue perform the real verification and advancement work instead of only flipping statuses.",
				Tasks: []phaseTaskTemplate{
					{
						Goal:            "Implement watcher-led verification and housekeeping before phase advancement",
						Constraints:     []string{"Signal housekeeping should stay wired into continue finalize", "Completed phases must only advance after verification"},
						Hints:           []string{"cmd/signal_housekeeping.go", ".claude/commands/ant/continue.md"},
						SuccessCriteria: []string{"Continue verifies actual artifacts", "Advance only happens after verification succeeds"},
					},
					{
						Goal:            "Record the continue worker flow in state, spawn logs, and user-facing output",
						Constraints:     []string{"The Next Up block must stay valid for the resulting state"},
						Hints:           []string{"renderContinueVisual", "print-next-up"},
						SuccessCriteria: []string{"Continue output matches actual work", "The next phase becomes ready when appropriate"},
					},
				},
				SuccessCriteria: []string{"Continue is no longer a pure state flip", "Phase advancement is defensible"},
			},
			{
				Name:        "End-to-end verification",
				Description: "Run the colony from init through seal and prove the Codex ant process is now honest.",
				Tasks: []phaseTaskTemplate{
					{
						Goal:            "Add tests that prove colonize, plan, build, and continue record real worker activity",
						Constraints:     []string{"Test the user path, not just helper functions"},
						Hints:           append([]string{"cmd/codex_colonize_test.go"}, survey.TestFiles...),
						SuccessCriteria: []string{"Core parity regressions are caught by tests", "Spawn tree entries are asserted in tests"},
					},
					{
						Goal:            "Run a live colony loop and compare its outputs with the documented ant process",
						Constraints:     []string{"Call out any remaining parity gap explicitly"},
						Hints:           []string{"spawn-tree.txt", "COLONY_STATE.json", ".aether/data/survey/"},
						SuccessCriteria: []string{"The live loop proves the implemented parity", "Remaining gaps are small and explicit"},
					},
				},
				SuccessCriteria: []string{"The live colony loop is credible", "The remaining parity gap is narrow and testable"},
			},
		}
	default:
		return []phaseTemplate{
			{
				Name:        "Discovery and boundaries",
				Description: "Map the relevant code paths, constraints, and success criteria before implementation.",
				Tasks: []phaseTaskTemplate{
					{
						Goal:            "Read the current implementation paths relevant to the goal",
						Constraints:     commonConstraints(survey),
						Hints:           commonHints(survey),
						SuccessCriteria: []string{"The working surface is explicit", "Key dependencies and boundaries are known"},
					},
					{
						Goal:            "Capture risks, constraints, and a testable target state",
						Constraints:     []string{"Keep the scope bounded to the requested outcome"},
						Hints:           survey.Issues,
						SuccessCriteria: []string{"Success criteria are explicit", "Known risks are visible before coding"},
					},
				},
				SuccessCriteria: []string{"The colony has a bounded target", "Implementation risks are known"},
			},
			{
				Name:        "Implementation",
				Description: "Make the core changes required to land the goal.",
				Tasks: []phaseTaskTemplate{
					{
						Goal:            "Implement the main behavior changes required by the goal",
						Constraints:     commonConstraints(survey),
						Hints:           commonHints(survey),
						SuccessCriteria: []string{"The main behavior lands", "The change integrates cleanly with the existing structure"},
					},
					{
						Goal:            "Update or add focused automated coverage",
						Constraints:     []string{"Use existing test patterns where possible"},
						Hints:           survey.TestFiles,
						SuccessCriteria: []string{"The new behavior is covered", "Important adjacent behavior is exercised"},
					},
				},
				SuccessCriteria: []string{"The goal is implemented", "Coverage exists for the changed behavior"},
			},
			{
				Name:        "Verification and polish",
				Description: "Verify the result, tighten loose ends, and prepare the colony for seal.",
				Tasks: []phaseTaskTemplate{
					{
						Goal:            "Run focused verification and address regressions",
						Constraints:     []string{"Prefer the smallest verification set that proves the result"},
						Hints:           survey.TestFiles,
						SuccessCriteria: []string{"Relevant verification is green", "Regressions are addressed"},
					},
					{
						Goal:            "Capture decisions, follow-ups, and user-visible changes",
						Constraints:     []string{"Do not hide residual risk"},
						Hints:           survey.Issues,
						SuccessCriteria: []string{"Key decisions are documented", "Remaining follow-ups are explicit"},
					},
				},
				SuccessCriteria: []string{"The result is verified", "The colony can move toward seal cleanly"},
			},
		}
	}
}

func containsAny(text string, needles []string) bool {
	for _, needle := range needles {
		if strings.Contains(text, needle) {
			return true
		}
	}
	return false
}

func commonConstraints(survey codexSurveyContext) []string {
	constraints := []string{
		"Follow the repository's existing structure and conventions",
		"Keep changes scoped to the current goal",
	}
	if len(survey.Issues) > 0 {
		constraints = append(constraints, survey.Issues[0])
	}
	return limitStrings(uniqueSortedStrings(constraints), 4)
}

func commonHints(survey codexSurveyContext) []string {
	hints := []string{}
	hints = append(hints, survey.EntryPoints...)
	hints = append(hints, survey.TestFiles...)
	if len(survey.Directories) > 0 {
		hints = append(hints, fmt.Sprintf("Top-level dirs: %s", strings.Join(limitStrings(survey.Directories, 5), ", ")))
	}
	return limitStrings(uniqueSortedStrings(hints), 5)
}

func writeRouteSetterArtifact(root, planningDir, goal string, granularity colony.PlanGranularity, survey codexSurveyContext, dispatch codexPlanningDispatch, confidence codexPlanConfidence, unresolvedGaps []string, phases []colony.Phase, snapshots map[string]codexArtifactSnapshot) (string, bool, error) {
	path := filepath.Join(planningDir, "ROUTE-SETTER.md")
	relPath := filepath.ToSlash(filepath.Join(".aether", "data", "planning", "ROUTE-SETTER.md"))
	if shouldPreserveWorkerArtifact(root, relPath, snapshots, claimedArtifactSet(dispatch.Claimed)) {
		return path, true, nil
	}

	var b strings.Builder
	b.WriteString("# Route-Setter Plan\n\n")
	b.WriteString(fmt.Sprintf("- Generated: %s\n", time.Now().UTC().Format(time.RFC3339)))
	b.WriteString(fmt.Sprintf("- Route-Setter: %s\n", dispatch.Name))
	b.WriteString(fmt.Sprintf("- Goal: %s\n", goal))
	b.WriteString(fmt.Sprintf("- Granularity: %s (%d-%d phases)\n", granularity, granularityMin(granularity), granularityMax(granularity)))
	b.WriteString(fmt.Sprintf("- Confidence: %d%% overall\n\n", confidence.Overall))
	b.WriteString("## Unresolved Gaps\n")
	b.WriteString(bulletList(unresolvedGaps, "No planning gaps remain."))
	b.WriteString("\n\n## Survey Inputs\n")
	b.WriteString(bulletList(survey.SurveyDocs, "No survey docs were available."))
	b.WriteString("\n\n## Phase Outline\n")
	for _, phase := range phases {
		b.WriteString(fmt.Sprintf("- Phase %d: %s\n", phase.ID, phase.Name))
	}
	b.WriteString("\n")
	if err := os.WriteFile(path, []byte(b.String()), 0644); err != nil {
		return "", false, fmt.Errorf("failed to write route-setter artifact: %w", err)
	}
	return path, false, nil
}

func writeWorkerPlanArtifact(root, planningDir string, confidence codexPlanConfidence, unresolvedGaps []string, phases []colony.Phase, snapshots map[string]codexArtifactSnapshot, dispatches []codexPlanningDispatch) (string, bool, error) {
	path := filepath.Join(planningDir, "phase-plan.json")
	relPath := filepath.ToSlash(filepath.Join(".aether", "data", "planning", "phase-plan.json"))
	if shouldPreserveWorkerArtifact(root, relPath, snapshots, claimedPlanningFiles(dispatches)) {
		return path, true, nil
	}

	artifact := codexWorkerPlanArtifact{
		Confidence: confidence,
		Gaps:       limitStrings(uniqueSortedStrings(unresolvedGaps), 4),
		Phases:     make([]codexWorkerPlanPhase, 0, len(phases)),
	}
	for _, phase := range phases {
		entry := codexWorkerPlanPhase{
			Name:            phase.Name,
			Description:     phase.Description,
			Tasks:           make([]codexWorkerPlanTask, 0, len(phase.Tasks)),
			SuccessCriteria: uniqueSortedStrings(phase.SuccessCriteria),
		}
		for _, task := range phase.Tasks {
			entry.Tasks = append(entry.Tasks, codexWorkerPlanTask{
				Goal:            task.Goal,
				Constraints:     uniqueSortedStrings(task.Constraints),
				Hints:           uniqueSortedStrings(task.Hints),
				SuccessCriteria: uniqueSortedStrings(task.SuccessCriteria),
				DependsOn:       uniqueSortedStrings(task.DependsOn),
			})
		}
		artifact.Phases = append(artifact.Phases, entry)
	}

	data, err := json.MarshalIndent(artifact, "", "  ")
	if err != nil {
		return "", false, fmt.Errorf("failed to marshal worker plan artifact: %w", err)
	}
	if err := os.WriteFile(path, append(data, '\n'), 0644); err != nil {
		return "", false, fmt.Errorf("failed to write worker plan artifact: %w", err)
	}
	return path, false, nil
}

func writePhaseResearchArtifacts(root, dir string, survey codexSurveyContext, report codexScoutReport, phases []colony.Phase, snapshots map[string]codexArtifactSnapshot, dispatches []codexPlanningDispatch) ([]string, int, error) {
	written := make([]string, 0, len(phases))
	claimed := claimedPlanningFiles(dispatches)
	preserved := 0
	for _, phase := range phases {
		name := fmt.Sprintf("phase-%d-research.md", phase.ID)
		path := filepath.Join(dir, name)
		relPath := filepath.ToSlash(filepath.Join(".aether", "data", "phase-research", name))
		if shouldPreserveWorkerArtifact(root, relPath, snapshots, claimed) {
			written = append(written, name)
			preserved++
			continue
		}
		var b strings.Builder
		b.WriteString(fmt.Sprintf("# Phase %d Research: %s\n\n", phase.ID, phase.Name))
		b.WriteString(fmt.Sprintf("- Generated: %s\n", time.Now().UTC().Format(time.RFC3339)))
		b.WriteString(fmt.Sprintf("- Phase: %d - %s\n\n", phase.ID, phase.Name))
		b.WriteString("## Goal Alignment\n")
		b.WriteString(strings.TrimSpace(phase.Description))
		b.WriteString("\n\n## Key Patterns\n")
		patterns := []string{}
		for _, finding := range report.Findings {
			patterns = append(patterns, fmt.Sprintf("%s: %s", finding.Area, finding.Discovery))
			if len(patterns) == 3 {
				break
			}
		}
		b.WriteString(bulletList(patterns, "No extra repo patterns were synthesized for this phase."))
		b.WriteString("\n\n## Risks\n")
		risks := append([]string{}, report.Gaps...)
		risks = append(risks, survey.Issues...)
		b.WriteString(bulletList(limitStrings(uniqueSortedStrings(risks), 4), "No additional risks captured for this phase."))
		b.WriteString("\n\n## Files to Study\n")
		files := append([]string{}, report.StudyFiles...)
		for _, task := range phase.Tasks {
			files = append(files, task.Hints...)
		}
		b.WriteString(bulletList(limitStrings(uniqueSortedStrings(files), 6), "No specific file anchors were identified."))
		b.WriteString("\n")
		if err := os.WriteFile(path, []byte(b.String()), 0644); err != nil {
			return nil, 0, fmt.Errorf("failed to write %s: %w", name, err)
		}
		written = append(written, name)
	}
	sort.Strings(written)
	return written, preserved, nil
}
