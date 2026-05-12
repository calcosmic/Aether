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
	"github.com/calcosmic/Aether/pkg/codegraph"
	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
)

type codexSurveyorDispatch struct {
	Stage         string   `json:"stage,omitempty"`
	Wave          int      `json:"wave,omitempty"`
	Caste         string   `json:"caste"`
	Name          string   `json:"name"`
	Task          string   `json:"task"`
	TaskID        string   `json:"task_id,omitempty"`
	AgentName     string   `json:"agent_name,omitempty"`
	Brief         string   `json:"brief,omitempty"`
	Outputs       []string `json:"outputs"`
	OutputPaths   []string `json:"output_paths,omitempty"`
	Status        string   `json:"status"`
	Summary       string   `json:"summary,omitempty"`
	Blockers      []string `json:"blockers,omitempty"`
	Duration      float64  `json:"duration,omitempty"` // Wall-clock seconds (0 = not measured)
	FilesCreated  []string `json:"files_created,omitempty"`
	FilesModified []string `json:"files_modified,omitempty"`
	SkillSection  string   `json:"skill_section,omitempty"`
	Claimed       []string `json:"-"`
}

type codexWorkspaceFacts struct {
	Root             string
	DetectedType     string
	Languages        []string
	Frameworks       []string
	Domains          []string
	EntryPoints      []string
	TopLevelDirs     []string
	ConfigFiles      []string
	PackageManagers  []string
	KeyDependencies  []string
	FileCount        int
	DirectoryCount   int
	TestFiles        []string
	ExampleFiles     []string
	TODOs            []string
	TypeSafetyGaps   []string
	SecurityPatterns []string
	Integrations     []string
}

type codexColonizeOptions struct {
	ForceResurvey bool
	WorkerTimeout time.Duration
	PlanOnly      bool
}

type codexColonizeManifest struct {
	Workflow             string                           `json:"workflow"`
	DispatchMode         string                           `json:"dispatch_mode"`
	RequiresFinalizer    bool                             `json:"requires_finalizer"`
	GeneratedAt          string                           `json:"generated_at"`
	Root                 string                           `json:"root"`
	DetectedType         string                           `json:"detected_type"`
	Languages            []string                         `json:"languages"`
	Frameworks           []string                         `json:"frameworks"`
	Domains              []string                         `json:"domains"`
	EntryPoints          []string                         `json:"entry_points"`
	KeyDirs              []string                         `json:"key_dirs"`
	ExistingSurvey       bool                             `json:"existing_survey"`
	ForceResurvey        bool                             `json:"force_resurvey"`
	WorkerTimeoutSeconds int                              `json:"worker_timeout_seconds"`
	DispatchContract     map[string]interface{}           `json:"dispatch_contract"`
	Dispatches           []codexSurveyorDispatch          `json:"dispatches"`
	Snapshots            map[string]codexArtifactSnapshot `json:"snapshots,omitempty"`
	FinalizerCommand     string                           `json:"finalizer_command"`
	Stats                map[string]interface{}           `json:"stats,omitempty"`
}

// logActivity appends an entry to the activity log. It is a no-op if the
// store is not initialized (e.g., during tests without a full colony setup).
func logActivity(command, details string) {
	if store == nil {
		return
	}
	entry := map[string]interface{}{
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"command":   command,
		"details":   details,
	}
	_ = store.AppendJSONL("activity.log", entry)
}

func runCodexColonize(root string, force bool) (map[string]interface{}, error) {
	return runCodexColonizeWithOptions(root, codexColonizeOptions{ForceResurvey: force})
}

func runCodexColonizeWithOptions(root string, opts codexColonizeOptions) (map[string]interface{}, error) {
	if store == nil {
		return nil, fmt.Errorf("no store initialized")
	}

	if opts.PlanOnly || codex.ShouldUseAgentDelegatePath() {
		return runCodexColonizePlanOnly(root, opts)
	}

	facts, err := surveyWorkspace(root)
	if err != nil {
		return nil, err
	}

	surveyDir := filepath.Join(store.BasePath(), "survey")
	existingSurvey := surveyDocsExist(surveyDir)
	if existingSurvey && !opts.ForceResurvey {
		return nil, fmt.Errorf("existing territory survey found; rerun with `aether colonize --force-resurvey` to refresh it")
	}

	if err := os.MkdirAll(surveyDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create survey directory: %w", err)
	}

	runHandle, err := beginRuntimeSpawnRun("colonize", time.Now().UTC())
	if err != nil {
		return nil, fmt.Errorf("failed to initialize colonize run: %w", err)
	}
	runStatus := "failed"
	defer func() {
		finishRuntimeSpawnRun(runHandle, runStatus, time.Now().UTC())
	}()
	surveySnapshots := snapshotRelativeFiles(root, filepath.ToSlash(filepath.Join(".aether", "data", "survey")))

	dispatches := plannedSurveyors(root)
	dispatchMode := "synthetic"
	artifactSource := "local-synthesis"
	surveyWarning := ""
	spawnTree := agent.NewSpawnTree(store, "spawn-tree.txt")
	for _, dispatch := range dispatches {
		if err := spawnTree.RecordSpawn("Queen", "surveyor", dispatch.Name, dispatch.Task, 1); err != nil {
			return nil, fmt.Errorf("failed to record surveyor spawn: %w", err)
		}
	}

	invoker := newCodexWorkerInvoker()
	if _, ok := invoker.(*codex.FakeInvoker); !ok && !invoker.IsAvailable(context.Background()) {
		dispatchMode = "fallback"
		surveyWarning = fmt.Sprintf("Real surveyors were unavailable, so Aether fell back to local survey synthesis. Cause: %s", dispatchAvailabilityMessage(invoker))
	} else {
		emitVisualProgress(renderColonizeDispatchPreview(facts.Root, dispatches))

		realDispatches, dispatchErr := dispatchRealSurveyorsWithTimeout(context.Background(), root, invoker, opts.WorkerTimeout)
		if realDispatches != nil {
			dispatches = realDispatches
		}
		if dispatchErr != nil {
			if _, ok := invoker.(*codex.FakeInvoker); ok {
				logActivity("colonize", "Brick-76: Fallback to planned surveyors (dispatch error)")
				dispatchMode = "simulated"
			} else {
				dispatchMode = "fallback"
				surveyWarning = fmt.Sprintf("Real surveyors did not finish cleanly, so Aether fell back to local survey synthesis. Cause: %s", dispatchErr.Error())
			}
		} else if realDispatches != nil {
			if _, ok := invoker.(*codex.FakeInvoker); ok {
				dispatchMode = "simulated"
			} else {
				dispatchMode = "real"
			}
			logActivity("colonize", fmt.Sprintf("Brick-76: %s surveyor dispatch, %d workers", dispatchMode, len(dispatches)))
		}
	}

	surveyFiles, preservedWorkerArtifacts, err := writeSurveyArtifacts(root, surveyDir, facts, dispatches, surveySnapshots)
	if err != nil {
		return nil, err
	}
	if preservedWorkerArtifacts > 0 {
		artifactSource = "worker-written"
	}
	if err := writeSurveyCompatibilityJSON(surveyDir, facts); err != nil {
		return nil, err
	}
	codegraphStats, codegraphWarning := runColonizeCodebaseGraph(root)

	for i := range dispatches {
		status := dispatches[i].Status
		if strings.TrimSpace(status) == "" || status == "spawned" {
			status = "completed"
		}
		summary := strings.TrimSpace(dispatches[i].Summary)
		if summary == "" {
			summary = strings.Join(dispatches[i].Outputs, ", ")
		}
		if summary == "" && dispatchMode != "real" {
			summary = "Local survey synthesis fallback"
		}
		if err := spawnTree.UpdateStatus(dispatches[i].Name, status, summary); err != nil {
			return nil, fmt.Errorf("failed to update surveyor completion: %w", err)
		}
	}
	emitColonizeCeremonyDispatchSequence("aether-colonize", dispatches)

	surveyedAt := time.Now().UTC().Format(time.RFC3339)
	if err := updateSurveyState(surveyedAt, len(surveyFiles)); err != nil {
		return nil, err
	}
	updateSessionSummary("colonize", "aether plan", fmt.Sprintf("Territory surveyed (%d documents)", len(surveyFiles)))

	dispatchMaps := surveyorDispatchMaps(dispatches)

	result := map[string]interface{}{
		"root":               facts.Root,
		"detected_type":      facts.DetectedType,
		"languages":          facts.Languages,
		"frameworks":         facts.Frameworks,
		"domains":            facts.Domains,
		"entry_points":       facts.EntryPoints,
		"key_dirs":           facts.TopLevelDirs,
		"survey_dir":         surveyDir,
		"survey_files":       surveyFiles,
		"surveyors":          dispatchMaps,
		"existing_survey":    existingSurvey,
		"force_resurvey":     opts.ForceResurvey,
		"territory_surveyed": surveyedAt,
		"dispatch_mode":      dispatchMode,
		"dispatch_contract":  surveyDispatchContractWithTimeout(opts.WorkerTimeout),
		"artifact_source":    artifactSource,
		"survey_warning":     surveyWarning,
		"stats": map[string]interface{}{
			"files":       facts.FileCount,
			"directories": facts.DirectoryCount,
		},
		"next": "aether plan",
	}
	if codegraphStats != nil {
		result["codebase_graph"] = map[string]interface{}{
			"files_scanned": codegraphStats.FilesScanned,
			"edges_found":   codegraphStats.EdgesFound,
			"languages":     codegraphStats.Languages,
			"output":        "codebase-graph.json",
		}
	}
	if codegraphWarning != "" {
		result["codebase_graph_warning"] = codegraphWarning
	}
	statuses := make([]string, 0, len(dispatches))
	for _, dispatch := range dispatches {
		statuses = append(statuses, dispatch.Status)
	}
	runStatus = summarizeRunStatus(statuses...)
	return result, nil
}

func surveyorDispatchMaps(dispatches []codexSurveyorDispatch) []map[string]interface{} {
	dispatchMaps := make([]map[string]interface{}, 0, len(dispatches))
	for _, dispatch := range dispatches {
		entry := map[string]interface{}{
			"stage":      dispatch.Stage,
			"wave":       dispatch.Wave,
			"caste":      dispatch.Caste,
			"name":       dispatch.Name,
			"task":       dispatch.Task,
			"task_id":    dispatch.TaskID,
			"agent_name": dispatch.AgentName,
			"outputs":    dispatch.Outputs,
			"status":     dispatch.Status,
		}
		if len(dispatch.OutputPaths) > 0 {
			entry["output_paths"] = dispatch.OutputPaths
		}
		if brief := strings.TrimSpace(dispatch.Brief); brief != "" {
			entry["brief"] = brief
		}
		if skillSection := strings.TrimSpace(dispatch.SkillSection); skillSection != "" {
			entry["skill_section"] = skillSection
		}
		if summary := strings.TrimSpace(dispatch.Summary); summary != "" {
			entry["summary"] = summary
		}
		if len(dispatch.Blockers) > 0 {
			entry["blockers"] = dispatch.Blockers
		}
		if dispatch.Duration > 0 {
			entry["duration"] = dispatch.Duration
		}
		if len(dispatch.FilesCreated) > 0 {
			entry["files_created"] = dispatch.FilesCreated
		}
		if len(dispatch.FilesModified) > 0 {
			entry["files_modified"] = dispatch.FilesModified
		}
		dispatchMaps = append(dispatchMaps, entry)
	}
	return dispatchMaps
}

func runColonizeCodebaseGraph(root string) (*codegraph.Stats, string) {
	stats, err := runCodebaseScanFromColonize(root, nil)
	if err != nil {
		logActivity("colonize", fmt.Sprintf("codebase graph skipped: %v", err))
		return nil, err.Error()
	}
	if stats != nil {
		logActivity("colonize", fmt.Sprintf("codebase graph scanned %d files, %d edges", stats.FilesScanned, stats.EdgesFound))
	}
	return stats, ""
}

func runCodexColonizePlanOnly(root string, opts codexColonizeOptions) (map[string]interface{}, error) {
	if store == nil {
		return nil, fmt.Errorf("no store initialized")
	}

	facts, err := surveyWorkspace(root)
	if err != nil {
		return nil, err
	}

	surveyDir := filepath.Join(store.BasePath(), "survey")
	existingSurvey := surveyDocsExist(surveyDir)
	if existingSurvey && !opts.ForceResurvey {
		return nil, fmt.Errorf("existing territory survey found; rerun with `aether colonize --force-resurvey` to refresh it")
	}

	dispatchMode := "plan-only"
	status := "plan-only"
	next := "dispatch host surveyor agents, then run `aether colonize-finalize --completion-file <file>`"
	if codex.ShouldUseAgentDelegatePath() {
		dispatchMode = "agent-delegate"
		status = "agent-delegate"
	}

	manifest := buildCodexColonizeManifest(root, facts, opts, dispatchMode, existingSurvey, snapshotRelativeFiles(root, filepath.ToSlash(filepath.Join(".aether", "data", "survey"))))
	dispatchMaps := surveyorDispatchMaps(manifest.Dispatches)
	result := map[string]interface{}{
		"status":                status,
		"root":                  facts.Root,
		"detected_type":         facts.DetectedType,
		"languages":             facts.Languages,
		"frameworks":            facts.Frameworks,
		"domains":               facts.Domains,
		"entry_points":          facts.EntryPoints,
		"key_dirs":              facts.TopLevelDirs,
		"existing_survey":       existingSurvey,
		"force_resurvey":        opts.ForceResurvey,
		"dispatch_mode":         dispatchMode,
		"dispatch_contract":     manifest.DispatchContract,
		"colonize_manifest":     manifest,
		"dispatches":            dispatchMaps,
		"surveyors":             dispatchMaps,
		"requires_finalizer":    true,
		"finalizer_command":     manifest.FinalizerCommand,
		"execution_owner":       "host-platform",
		"agent_delegate":        dispatchMode == "agent-delegate",
		"agent_delegate_reason": strings.TrimSpace(codex.AgentDelegateFallbackReason()),
		"stats":                 manifest.Stats,
		"next":                  next,
	}
	return result, nil
}

func buildCodexColonizeManifest(root string, facts codexWorkspaceFacts, opts codexColonizeOptions, dispatchMode string, existingSurvey bool, snapshots map[string]codexArtifactSnapshot) codexColonizeManifest {
	workerTimeout := effectiveSurveyorDispatchTimeout(opts.WorkerTimeout)
	dispatches := plannedSurveyors(root)
	for i := range dispatches {
		dispatches[i].Stage = "survey"
		dispatches[i].Wave = 1
		if dispatches[i].TaskID == "" {
			dispatches[i].TaskID = fmt.Sprintf("survey-%d", i)
		}
		outputPaths := make([]string, 0, len(dispatches[i].Outputs))
		for _, output := range dispatches[i].Outputs {
			outputPaths = append(outputPaths, filepath.ToSlash(filepath.Join(".aether", "data", "survey", output)))
		}
		dispatches[i].OutputPaths = outputPaths
		if dispatches[i].Brief == "" {
			dispatches[i].Brief = fmt.Sprintf("Survey task: %s\n\nWrite these survey outputs in the repo: %s\n\nSurvey the territory at %s", dispatches[i].Task, strings.Join(outputPaths, ", "), root)
		}
		if dispatches[i].SkillSection == "" {
			dispatches[i].SkillSection = resolveSkillSectionForWorkflow("colonize", dispatches[i].Caste, dispatches[i].Task)
		}
	}
	return codexColonizeManifest{
		Workflow:             "colonize",
		DispatchMode:         dispatchMode,
		RequiresFinalizer:    true,
		GeneratedAt:          time.Now().UTC().Format(time.RFC3339),
		Root:                 facts.Root,
		DetectedType:         facts.DetectedType,
		Languages:            facts.Languages,
		Frameworks:           facts.Frameworks,
		Domains:              facts.Domains,
		EntryPoints:          facts.EntryPoints,
		KeyDirs:              facts.TopLevelDirs,
		ExistingSurvey:       existingSurvey,
		ForceResurvey:        opts.ForceResurvey,
		WorkerTimeoutSeconds: int(workerTimeout / time.Second),
		DispatchContract:     surveyDispatchContractWithTimeout(opts.WorkerTimeout),
		Dispatches:           dispatches,
		Snapshots:            snapshots,
		FinalizerCommand:     "AETHER_OUTPUT_MODE=json aether colonize-finalize --completion-file <file>",
		Stats: map[string]interface{}{
			"files":       facts.FileCount,
			"directories": facts.DirectoryCount,
		},
	}
}

func surveyWorkspace(root string) (codexWorkspaceFacts, error) {
	facts := codexWorkspaceFacts{
		Root:             root,
		DetectedType:     "unknown",
		Languages:        []string{},
		Frameworks:       []string{},
		Domains:          detectDomainsFromRoot(root),
		EntryPoints:      []string{},
		TopLevelDirs:     []string{},
		ConfigFiles:      []string{},
		PackageManagers:  []string{},
		KeyDependencies:  []string{},
		TestFiles:        []string{},
		ExampleFiles:     []string{},
		TODOs:            []string{},
		TypeSafetyGaps:   []string{},
		SecurityPatterns: []string{},
		Integrations:     []string{},
	}

	entries, err := os.ReadDir(root)
	if err != nil {
		return facts, fmt.Errorf("failed to read workspace root: %w", err)
	}

	names := make(map[string]bool, len(entries))
	seenLang := map[string]bool{}
	seenFramework := map[string]bool{}
	seenConfig := map[string]bool{}
	seenDeps := map[string]bool{}
	seenIntegrations := map[string]bool{}
	seenDirs := map[string]bool{}
	seenEntrypoints := map[string]bool{}
	for _, entry := range entries {
		if entry.IsDir() {
			name := entry.Name()
			if shouldSkipSurveyDir(name) {
				continue
			}
			facts.TopLevelDirs = append(facts.TopLevelDirs, name)
			seenDirs[name] = true
			continue
		}
		names[entry.Name()] = true
		if isConfigFile(entry.Name()) && !seenConfig[entry.Name()] {
			facts.ConfigFiles = append(facts.ConfigFiles, entry.Name())
			seenConfig[entry.Name()] = true
		}
	}
	sort.Strings(facts.TopLevelDirs)

	for _, detector := range projectDetectors {
		if !names[detector.file] {
			continue
		}
		if facts.DetectedType == "unknown" {
			facts.DetectedType = detector.typ
		}
		if detectorContributesLanguage(detector.typ) && !seenLang[detector.typ] {
			facts.Languages = append(facts.Languages, detector.typ)
			seenLang[detector.typ] = true
		}
		for _, framework := range detector.frameworks {
			if seenFramework[framework] {
				continue
			}
			facts.Frameworks = append(facts.Frameworks, framework)
			seenFramework[framework] = true
		}
	}
	if hasAnyRootFile(names, dockerComposeFiles()) {
		facts.PackageManagers = appendUnique(facts.PackageManagers, "docker compose")
	}

	if names["go.mod"] {
		facts.PackageManagers = append(facts.PackageManagers, "go modules")
		deps, integrations := parseGoMod(filepath.Join(root, "go.mod"))
		for _, dep := range deps {
			if !seenDeps[dep] {
				facts.KeyDependencies = append(facts.KeyDependencies, dep)
				seenDeps[dep] = true
			}
		}
		for _, integration := range integrations {
			if !seenIntegrations[integration] {
				facts.Integrations = append(facts.Integrations, integration)
				seenIntegrations[integration] = true
			}
		}
	}
	if names["package.json"] {
		facts.PackageManagers = append(facts.PackageManagers, "npm")
	}
	if names["Makefile"] {
		facts.Frameworks = appendUnique(facts.Frameworks, "make")
	}

	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		if d.IsDir() {
			if path != root {
				facts.DirectoryCount++
			}
			if shouldSkipSurveyDir(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}

		facts.FileCount++
		base := filepath.Base(path)
		rel, err := filepath.Rel(root, path)
		if err != nil {
			rel = path
		}

		if isEntryPoint(base) && len(facts.EntryPoints) < 8 && !seenEntrypoints[rel] {
			facts.EntryPoints = append(facts.EntryPoints, rel)
			seenEntrypoints[rel] = true
		}
		if isTestFile(base) && len(facts.TestFiles) < 8 {
			facts.TestFiles = append(facts.TestFiles, rel)
		}
		if isExampleSource(base) && len(facts.ExampleFiles) < 8 {
			facts.ExampleFiles = append(facts.ExampleFiles, rel)
		}

		if !surveyReadableFile(base) {
			return nil
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		text := string(content)
		appendMatches(&facts.TODOs, rel, text, []string{"TODO", "FIXME", "HACK", "XXX"}, 10)
		appendMatches(&facts.TypeSafetyGaps, rel, text, []string{"interface{}", ": any", "@ts-ignore", "@ts-nocheck"}, 10)
		appendMatches(&facts.SecurityPatterns, rel, text, []string{"os.Getenv(", "process.env.", "dangerouslySetInnerHTML", "eval("}, 10)
		if !isPlatformGuidanceFile(rel) {
			appendMatches(&facts.Integrations, rel, text, []string{
				"github.com/spf13/cobra", "goreleaser", "GitHub", "OpenAI",
				"postgres", "postgresql", "mysql", "redis", "clickhouse",
				"metabase", "appsmith", "nocodb", "plausible", "listmonk",
			}, 10)
		}
		return nil
	})

	sort.Strings(facts.EntryPoints)
	sort.Strings(facts.Frameworks)
	sort.Strings(facts.Languages)
	sort.Strings(facts.Integrations)
	return facts, nil
}

func plannedSurveyors(root string) []codexSurveyorDispatch {
	specs := queenSurveyorSpecs()
	dispatches := make([]codexSurveyorDispatch, 0, len(specs))
	for i, spec := range specs {
		dispatches = append(dispatches, surveyDispatchFromSpec(root, spec, i))
	}
	return dispatches
}

// surveyorSpec defines a single surveyor for real dispatch.
type surveyorSpec struct {
	Caste       string
	AgentSuffix string // e.g., "nest" -> aether-surveyor-nest.toml
	Task        string
	Outputs     []string
}

// surveyorSpecs is the canonical list of surveyors, matching plannedSurveyors order.
var surveyorSpecs = []surveyorSpec{
	{Caste: "surveyor-provisions", AgentSuffix: "provisions", Task: "Map provisions and external trails", Outputs: []string{"PROVISIONS.md", "TRAILS.md"}},
	{Caste: "surveyor-nest", AgentSuffix: "nest", Task: "Map architecture and chamber layout", Outputs: []string{"BLUEPRINT.md", "CHAMBERS.md"}},
	{Caste: "surveyor-disciplines", AgentSuffix: "disciplines", Task: "Map disciplines and sentinel protocols", Outputs: []string{"DISCIPLINES.md", "SENTINEL-PROTOCOLS.md"}},
	{Caste: "surveyor-pathogens", AgentSuffix: "pathogens", Task: "Identify pathogens and fragile boundaries", Outputs: []string{"PATHOGENS.md"}},
}

func surveyDispatchFromSpec(root string, spec surveyorSpec, index int) codexSurveyorDispatch {
	seed := fmt.Sprintf("%s|%s", root, spec.AgentSuffix)
	return codexSurveyorDispatch{
		Stage:       "survey",
		Wave:        1,
		Caste:       spec.Caste,
		Name:        deterministicAntName("surveyor", seed),
		Task:        spec.Task,
		TaskID:      fmt.Sprintf("survey-%d", index),
		AgentName:   fmt.Sprintf("aether-surveyor-%s", spec.AgentSuffix),
		Outputs:     append([]string{}, spec.Outputs...),
		OutputPaths: surveyOutputPaths(spec.Outputs),
		Status:      "spawned",
	}
}

func surveyOutputPaths(outputs []string) []string {
	paths := make([]string, 0, len(outputs))
	for _, output := range outputs {
		paths = append(paths, filepath.ToSlash(filepath.Join(".aether", "data", "survey", output)))
	}
	return paths
}

func queenSurveyorSpecs() []surveyorSpec {
	phase := colony.Phase{
		Name:        "Colonize repository",
		Description: "Survey architecture, provisions, disciplines, and pathogens",
		Mode:        colony.PhaseModeDiscovery,
	}
	selected := queenBuildCasteSet(queenOrchestrate(phase, "colonize", colony.ColonyState{}))
	specs := make([]surveyorSpec, 0, len(surveyorSpecs))
	for _, spec := range surveyorSpecs {
		if selected[spec.Caste] {
			specs = append(specs, spec)
		}
	}
	return specs
}

// dispatchRealSurveyors attempts real worker invocation for surveyors.
// If the invoker is not available, it falls back to plannedSurveyors.
// The invoker parameter allows injection for testing.
func dispatchRealSurveyors(ctx context.Context, root string, invoker codex.WorkerInvoker) ([]codexSurveyorDispatch, error) {
	return dispatchRealSurveyorsWithTimeout(ctx, root, invoker, 0)
}

func dispatchRealSurveyorsWithTimeout(ctx context.Context, root string, invoker codex.WorkerInvoker, timeoutOverride time.Duration) ([]codexSurveyorDispatch, error) {
	if invoker == nil || !invoker.IsAvailable(ctx) {
		return plannedSurveyors(root), nil
	}

	specs := queenSurveyorSpecs()
	dispatches := make([]codex.WorkerDispatch, 0, len(specs))
	capsule := resolveCodexWorkerContext()
	pheromoneSection := resolvePheromoneSection()
	workerTimeout := effectiveSurveyorDispatchTimeout(timeoutOverride)
	for i, spec := range specs {
		tomlFile := fmt.Sprintf("aether-surveyor-%s.toml", spec.AgentSuffix)

		seed := fmt.Sprintf("%s|%s", root, spec.AgentSuffix)
		workerName := deterministicAntName("surveyor", seed)

		outputPaths := make([]string, 0, len(spec.Outputs))
		for _, output := range spec.Outputs {
			outputPaths = append(outputPaths, filepath.ToSlash(filepath.Join(".aether", "data", "survey", output)))
		}
		taskBrief := fmt.Sprintf("Survey task: %s\n\nWrite these survey outputs in the repo: %s\n\nSurvey the territory at %s", spec.Task, strings.Join(outputPaths, ", "), root)

		dispatches = append(dispatches, codex.WorkerDispatch{
			ID:               fmt.Sprintf("surveyor-%d", i),
			WorkerName:       workerName,
			AgentName:        fmt.Sprintf("aether-surveyor-%s", spec.AgentSuffix),
			AgentTOMLPath:    dispatchAgentPath(root, invoker, strings.TrimSuffix(tomlFile, ".toml")),
			Caste:            spec.Caste,
			TaskID:           fmt.Sprintf("survey-%d", i),
			TaskBrief:        taskBrief,
			ContextCapsule:   capsule,
			HandoffSection:   renderWorkerHandoffSection("colonize", 0, workerName),
			Workflow:         "colonize",
			SkillSection:     resolveSkillSectionForWorkflow("colonize", spec.Caste, spec.Task),
			PheromoneSection: pheromoneSection,
			Root:             root,
			Timeout:          workerTimeout,
			Wave:             1,
		})
	}

	spawnTree := agent.NewSpawnTree(store, "spawn-tree.txt")
	results, err := dispatchBatchByWaveWithVisuals(
		ctx,
		invoker,
		dispatches,
		colony.ModeInRepo,
		"Survey Wave",
		true,
		func(wave int) codex.DispatchObserver {
			return runtimeVisualDispatchObserver(spawnTree, "Survey running", wave)
		},
	)
	if err != nil {
		return nil, err
	}
	for _, result := range results {
		if result.Status != "completed" {
			return convertDispatchResults(results, specs, root), fmt.Errorf("surveyor %s did not complete: %s", result.WorkerName, result.Status)
		}
	}

	return convertDispatchResults(results, specs, root), nil
}

// convertDispatchResults maps a slice of DispatchResult to codexSurveyorDispatch.
// If results don't cover all specs, remaining specs get the planned-surveyor defaults.
func convertDispatchResults(results []codex.DispatchResult, specs []surveyorSpec, root string) []codexSurveyorDispatch {
	dispatches := make([]codexSurveyorDispatch, 0, len(specs))
	usedResults := make([]bool, len(results))

	for i, spec := range specs {
		d := surveyDispatchFromSpec(root, spec, i)

		if result, ok := findSurveyDispatchResult(d, results, usedResults); ok {
			applySurveyDispatchResult(&d, result)
		}

		dispatches = append(dispatches, d)
	}

	return dispatches
}

func findSurveyDispatchResult(dispatch codexSurveyorDispatch, results []codex.DispatchResult, used []bool) (codex.DispatchResult, bool) {
	for i, result := range results {
		if used[i] {
			continue
		}
		if surveyDispatchResultMatches(dispatch, result) {
			used[i] = true
			return result, true
		}
	}
	for i, result := range results {
		if used[i] || surveyDispatchResultHasIdentity(result) {
			continue
		}
		used[i] = true
		return result, true
	}
	return codex.DispatchResult{}, false
}

func surveyDispatchResultMatches(dispatch codexSurveyorDispatch, result codex.DispatchResult) bool {
	if result.WorkerResult != nil {
		if strings.TrimSpace(result.WorkerResult.TaskID) != "" && strings.TrimSpace(result.WorkerResult.TaskID) == dispatch.TaskID {
			return true
		}
		if strings.TrimSpace(result.WorkerResult.Caste) != "" && strings.TrimSpace(result.WorkerResult.Caste) == dispatch.Caste {
			return true
		}
		if strings.TrimSpace(result.WorkerResult.WorkerName) != "" && strings.TrimSpace(result.WorkerResult.WorkerName) == dispatch.Name {
			return true
		}
	}
	return strings.TrimSpace(result.WorkerName) != "" && strings.TrimSpace(result.WorkerName) == dispatch.Name
}

func surveyDispatchResultHasIdentity(result codex.DispatchResult) bool {
	if strings.TrimSpace(result.WorkerName) != "" {
		return true
	}
	if result.WorkerResult == nil {
		return false
	}
	return strings.TrimSpace(result.WorkerResult.TaskID) != "" ||
		strings.TrimSpace(result.WorkerResult.Caste) != "" ||
		strings.TrimSpace(result.WorkerResult.WorkerName) != ""
}

func applySurveyDispatchResult(dispatch *codexSurveyorDispatch, result codex.DispatchResult) {
	name := strings.TrimSpace(result.WorkerName)
	if result.WorkerResult != nil && strings.TrimSpace(result.WorkerResult.WorkerName) != "" {
		name = strings.TrimSpace(result.WorkerResult.WorkerName)
	}
	if name != "" {
		dispatch.Name = name
	}

	status := strings.TrimSpace(result.Status)
	if result.WorkerResult != nil && strings.TrimSpace(result.WorkerResult.Status) != "" {
		status = strings.TrimSpace(result.WorkerResult.Status)
	}
	dispatch.Status = normalizeRuntimeDispatchStatus(status)
	if result.WorkerResult != nil {
		dispatch.Duration = result.WorkerResult.Duration.Seconds()
		dispatch.FilesCreated = append(dispatch.FilesCreated, result.WorkerResult.FilesCreated...)
		dispatch.FilesModified = append(dispatch.FilesModified, result.WorkerResult.FilesModified...)
		dispatch.Claimed = append(dispatch.Claimed, dispatch.FilesCreated...)
		dispatch.Claimed = append(dispatch.Claimed, dispatch.FilesModified...)
		dispatch.Claimed = uniqueSortedStrings(dispatch.Claimed)
		dispatch.Summary = strings.TrimSpace(result.WorkerResult.Summary)
		if dispatch.Summary == "" && len(result.WorkerResult.Blockers) > 0 {
			dispatch.Blockers = append(dispatch.Blockers, result.WorkerResult.Blockers...)
			dispatch.Summary = strings.Join(result.WorkerResult.Blockers, "; ")
		}
	}
	if strings.TrimSpace(dispatch.Summary) == "" && result.Error != nil {
		dispatch.Summary = strings.TrimSpace(result.Error.Error())
	}
}

func writeSurveyArtifacts(root, surveyDir string, facts codexWorkspaceFacts, dispatches []codexSurveyorDispatch, snapshots map[string]codexArtifactSnapshot) ([]string, int, error) {
	generatedAt := time.Now().UTC().Format(time.RFC3339)
	dispatchByOutput, err := surveyDispatchesByRequiredOutput(dispatches)
	if err != nil {
		return nil, 0, err
	}
	files := map[string]string{
		"PROVISIONS.md":         renderSurveyProvisions(generatedAt, facts, dispatchByOutput["PROVISIONS.md"]),
		"TRAILS.md":             renderSurveyTrails(generatedAt, facts, dispatchByOutput["TRAILS.md"]),
		"BLUEPRINT.md":          renderSurveyBlueprint(generatedAt, facts, dispatchByOutput["BLUEPRINT.md"]),
		"CHAMBERS.md":           renderSurveyChambers(generatedAt, facts, dispatchByOutput["CHAMBERS.md"]),
		"DISCIPLINES.md":        renderSurveyDisciplines(generatedAt, facts, dispatchByOutput["DISCIPLINES.md"]),
		"SENTINEL-PROTOCOLS.md": renderSurveySentinel(generatedAt, facts, dispatchByOutput["SENTINEL-PROTOCOLS.md"]),
		"PATHOGENS.md":          renderSurveyPathogens(generatedAt, facts, dispatchByOutput["PATHOGENS.md"]),
	}
	claimed := make(map[string]bool)
	for _, dispatch := range dispatches {
		for relPath := range claimedArtifactSet(dispatch.Claimed) {
			claimed[relPath] = true
		}
	}

	names := make([]string, 0, len(files))
	preserved := 0
	for name, content := range files {
		relPath := filepath.ToSlash(filepath.Join(".aether", "data", "survey", name))
		if err := ensureSurveyArtifactPathWritable(filepath.Join(surveyDir, name), name); err != nil {
			return nil, 0, err
		}
		if shouldPreserveWorkerArtifact(root, relPath, snapshots, claimed) {
			names = append(names, name)
			preserved++
			continue
		}
		if err := os.WriteFile(filepath.Join(surveyDir, name), []byte(content), 0644); err != nil {
			return nil, 0, fmt.Errorf("failed to write %s: %w", name, err)
		}
		names = append(names, name)
	}
	sort.Strings(names)
	return names, preserved, nil
}

func ensureSurveyArtifactPathWritable(path, name string) error {
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to inspect %s: %w", name, err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("refusing to write survey artifact %s through a symlink", name)
	}
	if info.IsDir() {
		return fmt.Errorf("refusing to write survey artifact %s over a directory", name)
	}
	return nil
}

func surveyDispatchesByRequiredOutput(dispatches []codexSurveyorDispatch) (map[string]codexSurveyorDispatch, error) {
	required := make(map[string]bool, len(requiredSurveyMarkdownFiles))
	for _, name := range requiredSurveyMarkdownFiles {
		required[name] = true
	}

	byOutput := make(map[string]codexSurveyorDispatch, len(required))
	for _, dispatch := range dispatches {
		for _, output := range dispatch.Outputs {
			output = strings.TrimSpace(filepath.ToSlash(output))
			if !required[output] {
				continue
			}
			if existing, ok := byOutput[output]; ok {
				return nil, fmt.Errorf("duplicate surveyor output %s claimed by %s and %s", output, existing.Name, dispatch.Name)
			}
			byOutput[output] = dispatch
		}
	}

	for _, name := range requiredSurveyMarkdownFiles {
		if _, ok := byOutput[name]; !ok {
			return nil, fmt.Errorf("missing required surveyor output %s", name)
		}
	}
	return byOutput, nil
}

func writeSurveyCompatibilityJSON(surveyDir string, facts codexWorkspaceFacts) error {
	summaries := map[string]map[string]interface{}{
		"blueprint.json": {
			"entry_points": facts.EntryPoints,
			"frameworks":   facts.Frameworks,
			"summary":      "Architecture and entry points",
		},
		"chambers.json": {
			"directories": facts.TopLevelDirs,
			"summary":     "Directory layout",
		},
		"disciplines.json": {
			"tests":   facts.TestFiles,
			"summary": "Coding and testing disciplines",
		},
		"provisions.json": {
			"languages":    facts.Languages,
			"dependencies": facts.KeyDependencies,
			"summary":      "Technology stack and dependencies",
		},
		"pathogens.json": {
			"issues":  identifyPathogens(facts),
			"summary": "Known technical concerns",
		},
	}

	for fileName, payload := range summaries {
		data, err := json.MarshalIndent(payload, "", "  ")
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(surveyDir, fileName), append(data, '\n'), 0644); err != nil {
			return fmt.Errorf("failed to write %s: %w", fileName, err)
		}
	}
	return nil
}

func updateSurveyState(surveyedAt string, docCount int) error {
	if store == nil {
		return nil
	}

	var state colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
		state = colony.ColonyState{
			Version: "3.0",
			Plan:    colony.Plan{Phases: []colony.Phase{}},
			Memory: colony.Memory{
				PhaseLearnings: []colony.PhaseLearning{},
				Decisions:      []colony.Decision{},
				Instincts:      []colony.Instinct{},
			},
			Errors: colony.Errors{
				Records:         []colony.ErrorRecord{},
				FlaggedPatterns: []colony.FlaggedPattern{},
			},
			Signals:    []colony.Signal{},
			Graveyards: []colony.Graveyard{},
			Events:     []string{},
			State:      colony.StateREADY,
		}
	}

	state.State = colony.StateREADY
	state.TerritorySurveyed = &surveyedAt
	state.Events = append(trimmedEvents(state.Events), fmt.Sprintf("%s|territory_surveyed|colonize|Territory surveyed: %d documents", surveyedAt, docCount))
	return store.SaveJSON("COLONY_STATE.json", state)
}

func surveyDocsExist(surveyDir string) bool {
	for _, name := range requiredSurveyMarkdownFiles {
		if _, err := os.Stat(filepath.Join(surveyDir, name)); err == nil {
			return true
		}
	}
	return false
}

func shouldSkipSurveyDir(name string) bool {
	switch name {
	case ".git", ".cache", "node_modules", "dist", "build", "vendor", ".aether", ".claude", ".codex", ".opencode":
		return true
	}
	return false
}

func isConfigFile(name string) bool {
	for _, candidate := range []string{"go.mod", "go.sum", "package.json", "Makefile", "README.md", ".editorconfig", ".gitignore", ".goreleaser.yml"} {
		if name == candidate {
			return true
		}
	}
	return strings.HasSuffix(name, ".yaml") || strings.HasSuffix(name, ".yml") || strings.HasSuffix(name, ".toml")
}

func isEntryPoint(name string) bool {
	switch name {
	case "main.go", "main.ts", "main.js", "app.go", "app.ts", "app.js", "server.go":
		return true
	}
	return strings.HasPrefix(name, "index.")
}

func isTestFile(name string) bool {
	lower := strings.ToLower(name)
	if strings.HasSuffix(lower, "_test.go") || strings.Contains(lower, ".test.") || strings.Contains(lower, ".spec.") {
		return true
	}
	if strings.HasPrefix(lower, "test_") || strings.HasPrefix(lower, "test-") {
		return true
	}
	return strings.HasSuffix(lower, ".bats")
}

func isExampleSource(name string) bool {
	return strings.HasSuffix(name, ".go") || strings.HasSuffix(name, ".ts") || strings.HasSuffix(name, ".tsx") || strings.HasSuffix(name, ".js")
}

func surveyReadableFile(name string) bool {
	for _, suffix := range []string{".go", ".md", ".json", ".yaml", ".yml", ".toml", ".sh", ".bash", ".sql"} {
		if strings.HasSuffix(name, suffix) {
			return true
		}
	}
	return false
}

func dockerComposeFiles() []string {
	return []string{"docker-compose.yml", "docker-compose.yaml", "compose.yml", "compose.yaml"}
}

func hasAnyRootFile(names map[string]bool, candidates []string) bool {
	for _, candidate := range candidates {
		if names[candidate] {
			return true
		}
	}
	return false
}

func isPlatformGuidanceFile(rel string) bool {
	rel = filepath.ToSlash(rel)
	if strings.Contains(rel, "/") {
		return false
	}
	switch rel {
	case "AGENTS.md", "CLAUDE.md", "CODEX.md", "OPENCODE.md":
		return true
	}
	return false
}

func appendMatches(dest *[]string, rel, text string, needles []string, limit int) {
	if len(*dest) >= limit {
		return
	}
	lines := strings.Split(text, "\n")
	for idx, line := range lines {
		if len(*dest) >= limit {
			return
		}
		upper := strings.ToUpper(line)
		for _, needle := range needles {
			if strings.Contains(upper, strings.ToUpper(needle)) {
				*dest = append(*dest, fmt.Sprintf("%s:%d %s", rel, idx+1, strings.TrimSpace(line)))
				break
			}
		}
	}
}

func parseGoMod(path string) ([]string, []string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil
	}

	var deps []string
	var integrations []string
	inRequire := false
	for _, line := range strings.Split(string(data), "\n") {
		trimmed := strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(trimmed, "require ("):
			inRequire = true
		case inRequire && trimmed == ")":
			inRequire = false
		case strings.HasPrefix(trimmed, "require "):
			fields := strings.Fields(strings.TrimPrefix(trimmed, "require "))
			if len(fields) > 0 {
				deps = append(deps, fields[0])
			}
		case inRequire && trimmed != "":
			fields := strings.Fields(trimmed)
			if len(fields) > 0 {
				deps = append(deps, fields[0])
			}
		}
	}

	seenIntegration := map[string]bool{}
	for _, dep := range deps {
		switch {
		case strings.Contains(dep, "github.com/openai"):
			if !seenIntegration["OpenAI"] {
				integrations = append(integrations, "OpenAI")
				seenIntegration["OpenAI"] = true
			}
		case strings.Contains(dep, "github.com/google/go-github"), strings.Contains(dep, "github.com/cli/go-gh"):
			if !seenIntegration["GitHub"] {
				integrations = append(integrations, "GitHub")
				seenIntegration["GitHub"] = true
			}
		case strings.Contains(dep, "cobra"):
			if !seenIntegration["CLI orchestration"] {
				integrations = append(integrations, "CLI orchestration")
				seenIntegration["CLI orchestration"] = true
			}
		}
	}
	sort.Strings(deps)
	if len(deps) > 12 {
		deps = deps[:12]
	}
	return deps, integrations
}

func appendUnique(values []string, candidate string) []string {
	for _, existing := range values {
		if existing == candidate {
			return values
		}
	}
	return append(values, candidate)
}

func renderSurveyProvisions(generatedAt string, facts codexWorkspaceFacts, dispatch codexSurveyorDispatch) string {
	return renderSurveyDoc("PROVISIONS", generatedAt, dispatch.Name, []string{
		"## Languages",
		bulletList(facts.Languages, "No primary language markers detected."),
		"## Runtime",
		bulletList(facts.PackageManagers, "No explicit runtime/package manager markers detected."),
		"## Frameworks",
		bulletList(facts.Frameworks, "No framework markers detected."),
		"## Key Dependencies",
		bulletList(facts.KeyDependencies, "No key dependency manifests parsed."),
		"## Configuration",
		bulletList(facts.ConfigFiles, "No notable top-level config files detected."),
		"## Platform Requirements",
		fmt.Sprintf("- Root: `%s`", facts.Root),
		fmt.Sprintf("- Files scanned: %d", facts.FileCount),
		fmt.Sprintf("- Directories scanned: %d", facts.DirectoryCount),
	})
}

func renderSurveyTrails(generatedAt string, facts codexWorkspaceFacts, dispatch codexSurveyorDispatch) string {
	integrations := append([]string{}, facts.Integrations...)
	if len(integrations) == 0 {
		integrations = []string{"No direct third-party API client packages were detected in the scanned manifests."}
	}
	dataStorage := append([]string{}, filterContains(facts.KeyDependencies, []string{"sqlite", "postgres", "mysql", "mongo", "redis", "clickhouse"})...)
	dataStorage = append(dataStorage, filterContains(facts.Integrations, []string{"sqlite", "postgres", "mysql", "mongo", "redis", "clickhouse"})...)
	return renderSurveyDoc("TRAILS", generatedAt, dispatch.Name, []string{
		"## APIs & External Services",
		bulletList(integrations, "No explicit API/service integrations detected."),
		"## Data Storage",
		bulletList(dataStorage, "No dedicated database or storage service detected."),
		"## Authentication & Identity",
		bulletList(filterContains(facts.Integrations, []string{"GitHub", "OpenAI"}), "No dedicated identity provider package detected."),
		"## Monitoring & Observability",
		bulletList(filterContains(facts.KeyDependencies, []string{"slog", "zap", "otel"}), "No dedicated observability package detected."),
		"## CI/CD & Deployment",
		bulletList(filterContains(facts.ConfigFiles, []string{"goreleaser", "Makefile"}), "No dedicated release pipeline config detected."),
		"## Environment Configuration",
		bulletList(facts.SecurityPatterns, "No obvious environment variable patterns detected in sampled files."),
	})
}

func surveyPatternOverview(facts codexWorkspaceFacts) []string {
	var overview []string
	if facts.DetectedType != "" && facts.DetectedType != "unknown" {
		overview = append(overview, fmt.Sprintf("Detected project type: `%s`.", facts.DetectedType))
	}
	if len(facts.Frameworks) > 0 {
		overview = append(overview, fmt.Sprintf("Detected tools/frameworks: %s.", strings.Join(facts.Frameworks, ", ")))
	}
	if len(facts.Domains) > 0 {
		overview = append(overview, fmt.Sprintf("Detected domains: %s.", strings.Join(facts.Domains, ", ")))
	}
	if len(facts.TopLevelDirs) > 0 {
		overview = append(overview, fmt.Sprintf("Top-level layout includes: %s.", strings.Join(facts.TopLevelDirs, ", ")))
	}
	return overview
}

func inferArchitectureLayers(facts codexWorkspaceFacts) []string {
	var layers []string
	for _, dir := range facts.TopLevelDirs {
		switch {
		case dir == ".docker":
			layers = append(layers, "`.docker/` holds container bootstrap or service initialization assets.")
		case dir == "docs":
			layers = append(layers, "`docs/` holds project documentation and operating notes.")
		case dir == "tests":
			layers = append(layers, "`tests/` holds executable verification scripts or test suites.")
		case dir == "cmd":
			layers = append(layers, "`cmd/` holds executable entry points or command implementations.")
		case dir == "pkg":
			layers = append(layers, "`pkg/` holds reusable packages or libraries.")
		case dir == "src", dir == "app", dir == "pages", dir == "components":
			layers = append(layers, fmt.Sprintf("`%s/` holds application or frontend source.", dir))
		case strings.Contains(dir, "config"):
			layers = append(layers, fmt.Sprintf("`%s/` holds project or service configuration.", dir))
		case dir == "scripts" || dir == "bin":
			layers = append(layers, fmt.Sprintf("`%s/` holds automation scripts.", dir))
		case dir == "infra" || dir == "deploy" || dir == ".github":
			layers = append(layers, fmt.Sprintf("`%s/` holds infrastructure, deployment, or CI assets.", dir))
		case dir == "services" || dir == "apps" || dir == "packages":
			layers = append(layers, fmt.Sprintf("`%s/` suggests a multi-service or workspace layout.", dir))
		default:
			layers = append(layers, fmt.Sprintf("`%s/` is a top-level project area detected by the survey.", dir))
		}
	}
	if len(layers) == 0 && len(facts.EntryPoints) > 0 {
		layers = append(layers, fmt.Sprintf("Root-level entry points: %s.", strings.Join(facts.EntryPoints, ", ")))
	}
	return layers
}

func inferDataFlow(facts codexWorkspaceFacts) []string {
	var flow []string
	if hasDockerComposeConfig(facts) {
		flow = append(flow, "Docker Compose files coordinate local services and containerized dependencies.")
	}
	if len(facts.Integrations) > 0 {
		flow = append(flow, "Integration/service clues were found in scanned source or configuration files.")
	}
	if len(facts.EntryPoints) > 0 {
		flow = append(flow, fmt.Sprintf("Runtime flow likely starts from: %s.", strings.Join(facts.EntryPoints, ", ")))
	}
	if len(facts.ConfigFiles) > 0 {
		flow = append(flow, "Top-level config files shape runtime behavior and environment wiring.")
	}
	return flow
}

func inferKeyAbstractions(facts codexWorkspaceFacts) []string {
	var abstractions []string
	if hasDockerComposeConfig(facts) {
		abstractions = append(abstractions, "Service definitions", "container orchestration", "environment configuration")
	}
	if hasLanguage(facts, "go") {
		abstractions = append(abstractions, "Go packages", "executable entry points")
	}
	if hasLanguage(facts, "node") || hasFramework(facts, "node") {
		abstractions = append(abstractions, "Node package scripts", "application modules")
	}
	if hasTopLevelDir(facts, "docs") {
		abstractions = append(abstractions, "documentation-backed workflow")
	}
	if hasTopLevelDir(facts, "tests") {
		abstractions = append(abstractions, "scripted verification")
	}
	return abstractions
}

func inferCrossCuttingConcerns(facts codexWorkspaceFacts) []string {
	var concerns []string
	if len(facts.ConfigFiles) > 0 {
		concerns = append(concerns, "configuration management")
	}
	if len(facts.SecurityPatterns) > 0 {
		concerns = append(concerns, "environment variables or other security-sensitive settings")
	}
	if len(facts.TestFiles) > 0 {
		concerns = append(concerns, "verification coverage")
	}
	if hasTopLevelDir(facts, "docs") {
		concerns = append(concerns, "documentation upkeep")
	}
	if hasDockerComposeConfig(facts) {
		concerns = append(concerns, "service startup order and local dependency health")
	}
	return concerns
}

func inferDirectoryPurposes(facts codexWorkspaceFacts) []string {
	var purposes []string
	for _, layer := range inferArchitectureLayers(facts) {
		purposes = append(purposes, layer)
	}
	return purposes
}

func inferNamingConventions(facts codexWorkspaceFacts) []string {
	var conventions []string
	if hasLanguage(facts, "go") {
		conventions = append(conventions, "Go tests use `*_test.go` where present.")
	}
	if hasDockerComposeConfig(facts) {
		conventions = append(conventions, "Docker Compose configuration lives in root compose YAML files.")
	}
	for _, testFile := range facts.TestFiles {
		lower := strings.ToLower(filepath.Base(testFile))
		if strings.HasPrefix(lower, "test_") || strings.HasPrefix(lower, "test-") || strings.HasSuffix(lower, ".bats") {
			conventions = appendUnique(conventions, "Shell or command-level tests use `test_*`, `test-*`, or Bats-style filenames.")
		}
	}
	if len(filterContains(facts.ConfigFiles, []string{".yaml", ".yml", ".toml"})) > 0 {
		conventions = append(conventions, "Configuration uses YAML/TOML files.")
	}
	return conventions
}

func inferCodeStyleClues(facts codexWorkspaceFacts) []string {
	var clues []string
	if hasLanguage(facts, "go") {
		clues = append(clues, "Go source files are present; prefer gofmt/go test conventions.")
	}
	if hasLanguage(facts, "node") {
		clues = append(clues, "Node package metadata is present; follow package scripts and project linting if configured.")
	}
	if hasDockerComposeConfig(facts) {
		clues = append(clues, "Configuration-first service composition is driven by Docker Compose files.")
	}
	if hasTopLevelDir(facts, "docs") {
		clues = append(clues, "Documentation is a first-class project artifact.")
	}
	return clues
}

func inferImportOrganization(facts codexWorkspaceFacts) []string {
	var clues []string
	if hasLanguage(facts, "go") {
		clues = append(clues, "`go.mod` is the dependency source of truth.")
	}
	if hasLanguage(facts, "node") || hasFramework(facts, "node") {
		clues = append(clues, "`package.json` is the package/script source of truth.")
	}
	if len(facts.KeyDependencies) > 0 {
		clues = append(clues, fmt.Sprintf("Parsed dependency samples: %s.", strings.Join(facts.KeyDependencies, ", ")))
	}
	return clues
}

func inferErrorHandlingClues(facts codexWorkspaceFacts) []string {
	var clues []string
	if len(facts.SecurityPatterns) > 0 {
		clues = append(clues, "Environment-sensitive settings were found; validate secret handling and defaults.")
	}
	if len(facts.TestFiles) > 0 {
		clues = append(clues, "Tests exist and should guard behavior changes.")
	}
	if len(facts.TODOs) > 0 {
		clues = append(clues, "TODO/FIXME markers may identify incomplete error paths.")
	}
	return clues
}

func inferTestFrameworks(facts codexWorkspaceFacts) []string {
	var frameworks []string
	for _, testFile := range facts.TestFiles {
		lower := strings.ToLower(testFile)
		switch {
		case strings.HasSuffix(lower, "_test.go"):
			frameworks = appendUnique(frameworks, "Go `testing` package")
		case strings.Contains(lower, ".test.") || strings.Contains(lower, ".spec."):
			frameworks = appendUnique(frameworks, "JavaScript/TypeScript spec-style tests")
		case strings.HasSuffix(lower, ".sh") || strings.HasSuffix(lower, ".bash"):
			frameworks = appendUnique(frameworks, "shell-based smoke or regression tests")
		case strings.HasSuffix(lower, ".bats"):
			frameworks = appendUnique(frameworks, "Bats shell tests")
		}
	}
	return frameworks
}

func inferTestStructure(facts codexWorkspaceFacts) []string {
	var structure []string
	for _, testFile := range facts.TestFiles {
		lower := strings.ToLower(testFile)
		switch {
		case strings.Contains(lower, "/tests/") || strings.HasPrefix(lower, "tests/"):
			structure = appendUnique(structure, "Tests are grouped under `tests/`.")
		case strings.HasSuffix(lower, "_test.go"):
			structure = appendUnique(structure, "Go tests sit near package source files.")
		case strings.HasSuffix(lower, ".sh") || strings.HasSuffix(lower, ".bash"):
			structure = appendUnique(structure, "Shell test scripts verify command or service behavior.")
		}
	}
	return structure
}

func inferCoverageTargets(facts codexWorkspaceFacts) []string {
	var targets []string
	if hasDockerComposeConfig(facts) {
		targets = append(targets, "service composition and startup wiring")
	}
	if len(facts.EntryPoints) > 0 {
		targets = append(targets, "runtime entry points")
	}
	if len(facts.ConfigFiles) > 0 {
		targets = append(targets, "configuration and environment handling")
	}
	if len(facts.Integrations) > 0 {
		targets = append(targets, "external service integration points")
	}
	return targets
}

func inferCommonTestPatterns(facts codexWorkspaceFacts) []string {
	var patterns []string
	for _, testFile := range facts.TestFiles {
		lower := strings.ToLower(testFile)
		switch {
		case strings.HasSuffix(lower, "_test.go"):
			patterns = appendUnique(patterns, "`go test ./...` likely exercises Go tests.")
		case strings.HasSuffix(lower, ".sh") || strings.HasSuffix(lower, ".bash"):
			patterns = appendUnique(patterns, "Executable shell scripts likely drive smoke/regression checks.")
		case strings.Contains(lower, ".test.") || strings.Contains(lower, ".spec."):
			patterns = appendUnique(patterns, "Spec-style files likely run through the project package scripts.")
		}
	}
	return patterns
}

func hasDockerComposeConfig(facts codexWorkspaceFacts) bool {
	for _, configFile := range facts.ConfigFiles {
		for _, composeFile := range dockerComposeFiles() {
			if configFile == composeFile {
				return true
			}
		}
	}
	return false
}

func hasLanguage(facts codexWorkspaceFacts, language string) bool {
	return containsString(facts.Languages, language)
}

func hasFramework(facts codexWorkspaceFacts, framework string) bool {
	return containsString(facts.Frameworks, framework)
}

func hasTopLevelDir(facts codexWorkspaceFacts, dir string) bool {
	return containsString(facts.TopLevelDirs, dir)
}

func renderSurveyBlueprint(generatedAt string, facts codexWorkspaceFacts, dispatch codexSurveyorDispatch) string {
	return renderSurveyDoc("BLUEPRINT", generatedAt, dispatch.Name, []string{
		"## Pattern Overview",
		bulletList(surveyPatternOverview(facts), "No architectural pattern inferred from scanned files."),
		"## Layers",
		bulletList(inferArchitectureLayers(facts), "No layered structure detected."),
		"## Data Flow",
		bulletList(inferDataFlow(facts), "No data flow summary available from scanned files."),
		"## Key Abstractions",
		bulletList(inferKeyAbstractions(facts), "No abstractions detected from scanned files."),
		"## Entry Points",
		bulletList(facts.EntryPoints, "No entry points detected."),
		"## Cross-Cutting Concerns",
		bulletList(inferCrossCuttingConcerns(facts), "No cross-cutting concerns detected."),
	})
}

func renderSurveyChambers(generatedAt string, facts codexWorkspaceFacts, dispatch codexSurveyorDispatch) string {
	return renderSurveyDoc("CHAMBERS", generatedAt, dispatch.Name, []string{
		"## Directory Layout",
		bulletList(facts.TopLevelDirs, "No top-level directories detected."),
		"## Directory Purposes",
		bulletList(inferDirectoryPurposes(facts), "No directory purpose summaries available."),
		"## Key File Locations",
		bulletList(facts.EntryPoints, "No key file locations detected."),
		"## Naming Conventions",
		bulletList(inferNamingConventions(facts), "No naming conventions inferred."),
		"## Special Directories",
		bulletList(filterContains(facts.TopLevelDirs, []string{".docker", ".github", "infra", "deploy", "scripts", "tests"}), "No special directories detected."),
	})
}

func renderSurveyDisciplines(generatedAt string, facts codexWorkspaceFacts, dispatch codexSurveyorDispatch) string {
	examples := append([]string{}, facts.ExampleFiles...)
	if len(examples) > 6 {
		examples = examples[:6]
	}
	return renderSurveyDoc("DISCIPLINES", generatedAt, dispatch.Name, []string{
		"## Naming Patterns",
		bulletList(inferNamingConventions(facts), "No naming patterns inferred."),
		"## Code Style",
		bulletList(inferCodeStyleClues(facts), "No code style clues detected."),
		"## Import Organization",
		bulletList(inferImportOrganization(facts), "No import organization clues detected."),
		"## Error Handling",
		bulletList(inferErrorHandlingClues(facts), "No error handling conventions inferred."),
		"## Testing",
		bulletList(facts.TestFiles, "No test files detected."),
		"## Example Source Files",
		bulletList(examples, "No representative source files detected."),
	})
}

func renderSurveySentinel(generatedAt string, facts codexWorkspaceFacts, dispatch codexSurveyorDispatch) string {
	return renderSurveyDoc("SENTINEL-PROTOCOLS", generatedAt, dispatch.Name, []string{
		"## Test Framework",
		bulletList(inferTestFrameworks(facts), "No test suite detected."),
		"## Test File Organization",
		bulletList(facts.TestFiles, "No test files detected."),
		"## Test Structure",
		bulletList(inferTestStructure(facts), "No test structure inferred."),
		"## Coverage Targets",
		bulletList(inferCoverageTargets(facts), "No coverage targets inferred."),
		"## Common Patterns",
		bulletList(inferCommonTestPatterns(facts), "No testing patterns inferred."),
	})
}

func renderSurveyPathogens(generatedAt string, facts codexWorkspaceFacts, dispatch codexSurveyorDispatch) string {
	issues := identifyPathogens(facts)
	return renderSurveyDoc("PATHOGENS", generatedAt, dispatch.Name, []string{
		"## Tech Debt",
		bulletList(issues, "No obvious technical debt markers detected."),
		"## TODO / FIXME Markers",
		bulletList(facts.TODOs, "No TODO/FIXME/HACK markers detected in sampled files."),
		"## Type Safety Gaps",
		bulletList(facts.TypeSafetyGaps, "No obvious type-safety gaps detected in sampled files."),
		"## Security Considerations",
		bulletList(facts.SecurityPatterns, "No high-risk security patterns detected in sampled files."),
	})
}

func identifyPathogens(facts codexWorkspaceFacts) []string {
	var issues []string

	if len(facts.TestFiles) == 0 {
		issues = append(issues, "No test files detected — consider adding tests.")
	}
	if len(facts.TypeSafetyGaps) > 0 {
		issues = append(issues, fmt.Sprintf("Type safety gaps found in %d file(s) — review for correctness.", len(facts.TypeSafetyGaps)))
	}
	if len(facts.SecurityPatterns) > 3 {
		issues = append(issues, fmt.Sprintf("High volume of env/eval patterns (%d) — verify none leak secrets.", len(facts.SecurityPatterns)))
	}
	if len(facts.TODOs) > 5 {
		issues = append(issues, fmt.Sprintf("%d TODO/FIXME/HACK markers need review.", len(facts.TODOs)))
	}
	if len(facts.KeyDependencies) == 0 && facts.FileCount > 10 && !hasDockerComposeConfig(facts) {
		issues = append(issues, "No dependency manifest detected.")
	}

	if len(issues) == 0 {
		return []string{"No obvious technical debt markers detected."}
	}
	return issues
}

func renderSurveyDoc(title, generatedAt, surveyor string, sections []string) string {
	var b strings.Builder
	b.WriteString("# ")
	b.WriteString(title)
	b.WriteString("\n\n")
	b.WriteString(fmt.Sprintf("- Generated: %s\n", generatedAt))
	b.WriteString(fmt.Sprintf("- Surveyor: %s\n\n", surveyor))
	for i, section := range sections {
		trimmed := strings.TrimRight(section, "\n")
		if trimmed == "" {
			continue
		}
		b.WriteString(trimmed)
		b.WriteString("\n")
		if i < len(sections)-1 {
			b.WriteString("\n")
		}
	}
	return b.String()
}

func bulletList(values []string, fallback string) string {
	filtered := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		filtered = append(filtered, value)
	}
	if len(filtered) == 0 {
		return "- " + fallback
	}
	var b strings.Builder
	for _, value := range filtered {
		if strings.HasPrefix(value, "- ") || strings.HasPrefix(value, "`") {
			b.WriteString("- ")
			b.WriteString(value)
		} else {
			b.WriteString("- ")
			b.WriteString(value)
		}
		b.WriteString("\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

func filterContains(values []string, needles []string) []string {
	var matches []string
	for _, value := range values {
		for _, needle := range needles {
			if strings.Contains(strings.ToLower(value), strings.ToLower(needle)) {
				matches = append(matches, value)
				break
			}
		}
	}
	return matches
}
