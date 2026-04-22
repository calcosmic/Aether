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

type codexSurveyorDispatch struct {
	Caste    string   `json:"caste"`
	Name     string   `json:"name"`
	Task     string   `json:"task"`
	Outputs  []string `json:"outputs"`
	Status   string   `json:"status"`
	Summary  string   `json:"summary,omitempty"`
	Duration float64  `json:"duration,omitempty"` // Wall-clock seconds (0 = not measured)
	Claimed  []string `json:"-"`
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

	surveyedAt := time.Now().UTC().Format(time.RFC3339)
	if err := updateSurveyState(surveyedAt, len(surveyFiles)); err != nil {
		return nil, err
	}
	updateSessionSummary("colonize", "aether plan", fmt.Sprintf("Territory surveyed (%d documents)", len(surveyFiles)))

	dispatchMaps := make([]map[string]interface{}, 0, len(dispatches))
	for _, dispatch := range dispatches {
		entry := map[string]interface{}{
			"caste":   dispatch.Caste,
			"name":    dispatch.Name,
			"task":    dispatch.Task,
			"outputs": dispatch.Outputs,
			"status":  dispatch.Status,
		}
		if summary := strings.TrimSpace(dispatch.Summary); summary != "" {
			entry["summary"] = summary
		}
		if dispatch.Duration > 0 {
			entry["duration"] = dispatch.Duration
		}
		dispatchMaps = append(dispatchMaps, entry)
	}

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
	statuses := make([]string, 0, len(dispatches))
	for _, dispatch := range dispatches {
		statuses = append(statuses, dispatch.Status)
	}
	runStatus = summarizeRunStatus(statuses...)
	return result, nil
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
		if !seenLang[detector.typ] {
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
		appendMatches(&facts.Integrations, rel, text, []string{"github.com/spf13/cobra", "goreleaser", "GitHub", "OpenAI", "Codex", "Claude", "OpenCode"}, 10)
		return nil
	})

	sort.Strings(facts.EntryPoints)
	sort.Strings(facts.Frameworks)
	sort.Strings(facts.Languages)
	sort.Strings(facts.Integrations)
	return facts, nil
}

func plannedSurveyors(root string) []codexSurveyorDispatch {
	return []codexSurveyorDispatch{
		{
			Caste:   "surveyor-provisions",
			Name:    deterministicAntName("surveyor", root+"|provisions"),
			Task:    "Map provisions and external trails",
			Outputs: []string{"PROVISIONS.md", "TRAILS.md"},
			Status:  "spawned",
		},
		{
			Caste:   "surveyor-nest",
			Name:    deterministicAntName("surveyor", root+"|nest"),
			Task:    "Map architecture and chamber layout",
			Outputs: []string{"BLUEPRINT.md", "CHAMBERS.md"},
			Status:  "spawned",
		},
		{
			Caste:   "surveyor-disciplines",
			Name:    deterministicAntName("surveyor", root+"|disciplines"),
			Task:    "Map disciplines and sentinel protocols",
			Outputs: []string{"DISCIPLINES.md", "SENTINEL-PROTOCOLS.md"},
			Status:  "spawned",
		},
		{
			Caste:   "surveyor-pathogens",
			Name:    deterministicAntName("surveyor", root+"|pathogens"),
			Task:    "Identify pathogens and fragile boundaries",
			Outputs: []string{"PATHOGENS.md"},
			Status:  "spawned",
		},
	}
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

	dispatches := make([]codex.WorkerDispatch, 0, len(surveyorSpecs))
	capsule := resolveCodexWorkerContext()
	pheromoneSection := resolvePheromoneSection()
	workerTimeout := effectiveSurveyorDispatchTimeout(timeoutOverride)
	for i, spec := range surveyorSpecs {
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
			SkillSection:     resolveSkillSection(spec.Caste, spec.Task),
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
			return convertDispatchResults(results, surveyorSpecs, root), fmt.Errorf("surveyor %s did not complete: %s", result.WorkerName, result.Status)
		}
	}

	return convertDispatchResults(results, surveyorSpecs, root), nil
}

// convertDispatchResults maps a slice of DispatchResult to codexSurveyorDispatch.
// If results don't cover all specs, remaining specs get the planned-surveyor defaults.
func convertDispatchResults(results []codex.DispatchResult, specs []surveyorSpec, root string) []codexSurveyorDispatch {
	dispatches := make([]codexSurveyorDispatch, 0, len(specs))

	for i, spec := range specs {
		seed := fmt.Sprintf("%s|%s", root, spec.AgentSuffix)
		name := deterministicAntName("surveyor", seed)

		d := codexSurveyorDispatch{
			Caste:   spec.Caste,
			Name:    name,
			Task:    spec.Task,
			Outputs: spec.Outputs,
			Status:  "spawned",
		}

		if i < len(results) {
			r := results[i]
			d.Name = r.WorkerName
			if d.Name == "" {
				d.Name = name
			}
			d.Status = normalizeRuntimeDispatchStatus(r.Status)
			if r.WorkerResult != nil {
				d.Duration = r.WorkerResult.Duration.Seconds()
				d.Claimed = append(d.Claimed, r.WorkerResult.FilesCreated...)
				d.Claimed = append(d.Claimed, r.WorkerResult.FilesModified...)
				d.Claimed = uniqueSortedStrings(d.Claimed)
				d.Summary = strings.TrimSpace(r.WorkerResult.Summary)
				if d.Summary == "" && len(r.WorkerResult.Blockers) > 0 {
					d.Summary = strings.Join(r.WorkerResult.Blockers, "; ")
				}
			}
			if strings.TrimSpace(d.Summary) == "" && r.Error != nil {
				d.Summary = strings.TrimSpace(r.Error.Error())
			}
		}

		dispatches = append(dispatches, d)
	}

	return dispatches
}

func writeSurveyArtifacts(root, surveyDir string, facts codexWorkspaceFacts, dispatches []codexSurveyorDispatch, snapshots map[string]codexArtifactSnapshot) ([]string, int, error) {
	generatedAt := time.Now().UTC().Format(time.RFC3339)
	files := map[string]string{
		"PROVISIONS.md":         renderSurveyProvisions(generatedAt, facts, dispatches[0]),
		"TRAILS.md":             renderSurveyTrails(generatedAt, facts, dispatches[0]),
		"BLUEPRINT.md":          renderSurveyBlueprint(generatedAt, facts, dispatches[1]),
		"CHAMBERS.md":           renderSurveyChambers(generatedAt, facts, dispatches[1]),
		"DISCIPLINES.md":        renderSurveyDisciplines(generatedAt, facts, dispatches[2]),
		"SENTINEL-PROTOCOLS.md": renderSurveySentinel(generatedAt, facts, dispatches[2]),
		"PATHOGENS.md":          renderSurveyPathogens(generatedAt, facts, dispatches[3]),
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
	for _, name := range []string{"PROVISIONS.md", "TRAILS.md", "BLUEPRINT.md", "CHAMBERS.md", "DISCIPLINES.md", "SENTINEL-PROTOCOLS.md", "PATHOGENS.md"} {
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
	return strings.HasSuffix(name, "_test.go") || strings.Contains(name, ".test.") || strings.Contains(name, ".spec.")
}

func isExampleSource(name string) bool {
	return strings.HasSuffix(name, ".go") || strings.HasSuffix(name, ".ts") || strings.HasSuffix(name, ".tsx") || strings.HasSuffix(name, ".js")
}

func surveyReadableFile(name string) bool {
	for _, suffix := range []string{".go", ".md", ".json", ".yaml", ".yml", ".toml"} {
		if strings.HasSuffix(name, suffix) {
			return true
		}
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
	return renderSurveyDoc("TRAILS", generatedAt, dispatch.Name, []string{
		"## APIs & External Services",
		bulletList(integrations, "No explicit API/service integrations detected."),
		"## Data Storage",
		bulletList(filterContains(facts.KeyDependencies, []string{"sqlite", "postgres", "mysql", "mongo"}), "No dedicated database client package detected."),
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

func renderSurveyBlueprint(generatedAt string, facts codexWorkspaceFacts, dispatch codexSurveyorDispatch) string {
	layers := []string{
		"`cmd/` holds user-facing CLI command implementations.",
		"`pkg/` holds reusable colony packages and storage/agent infrastructure.",
		"Platform companion assets live in `.aether/`, `.claude/`, `.opencode/`, and `.codex/`.",
	}
	return renderSurveyDoc("BLUEPRINT", generatedAt, dispatch.Name, []string{
		"## Pattern Overview",
		fmt.Sprintf("- The repo is organized as a Go CLI plus companion asset trees for multiple AI platforms."),
		"## Layers",
		bulletList(layers, "No layered structure detected."),
		"## Data Flow",
		bulletList([]string{
			"`aether install` publishes companion files into the hub.",
			"`aether update` / `lay-eggs` sync hub assets into working repos.",
			"Colony commands mutate `.aether/data/COLONY_STATE.json` and related runtime files.",
		}, "No data flow summary available."),
		"## Key Abstractions",
		bulletList([]string{"Cobra commands", "Colony state machine", "Spawn tree", "Pheromone/context assembly", "Platform sync"}, "No abstractions detected."),
		"## Entry Points",
		bulletList(facts.EntryPoints, "No entry points detected."),
		"## Cross-Cutting Concerns",
		bulletList([]string{"state safety", "context assembly", "agent definitions", "hub sync"}, "No cross-cutting concerns detected."),
	})
}

func renderSurveyChambers(generatedAt string, facts codexWorkspaceFacts, dispatch codexSurveyorDispatch) string {
	purposes := make([]string, 0, len(facts.TopLevelDirs))
	for _, dir := range facts.TopLevelDirs {
		switch dir {
		case "cmd":
			purposes = append(purposes, "`cmd/` — Go CLI command surface.")
		case "pkg":
			purposes = append(purposes, "`pkg/` — reusable packages and runtime internals.")
		case ".aether":
			purposes = append(purposes, "`.aether/` — companion assets and local colony state.")
		case ".claude", ".opencode", ".codex":
			purposes = append(purposes, fmt.Sprintf("`%s/` — platform-specific command or agent assets.", dir))
		case "docs":
			purposes = append(purposes, "`docs/` — supporting reference and marketing docs.")
		}
	}
	return renderSurveyDoc("CHAMBERS", generatedAt, dispatch.Name, []string{
		"## Directory Layout",
		bulletList(facts.TopLevelDirs, "No top-level directories detected."),
		"## Directory Purposes",
		bulletList(purposes, "No directory purpose summaries available."),
		"## Key File Locations",
		bulletList(facts.EntryPoints, "No key file locations detected."),
		"## Naming Conventions",
		bulletList([]string{"Go commands live in `cmd/*.go`", "Tests use `*_test.go`", "Agent definitions use platform-specific directories"}, "No naming conventions inferred."),
		"## Special Directories",
		bulletList(filterContains(facts.TopLevelDirs, []string{".aether", ".claude", ".opencode", ".codex"}), "No special directories detected."),
	})
}

func renderSurveyDisciplines(generatedAt string, facts codexWorkspaceFacts, dispatch codexSurveyorDispatch) string {
	examples := append([]string{}, facts.ExampleFiles...)
	if len(examples) > 6 {
		examples = examples[:6]
	}
	return renderSurveyDoc("DISCIPLINES", generatedAt, dispatch.Name, []string{
		"## Naming Patterns",
		bulletList([]string{"snake/cobra-style command ids", "Go package directories under `cmd/` and `pkg/`", "`*_test.go` for tests"}, "No naming patterns inferred."),
		"## Code Style",
		bulletList([]string{"Go-first implementation", "JSON envelopes for machine-safe command output", "visual renderer layered on top of JSON output"}, "No code style clues detected."),
		"## Import Organization",
		bulletList([]string{"standard library first, then internal Aether packages, then third-party packages"}, "No import organization clues detected."),
		"## Error Handling",
		bulletList([]string{"command handlers prefer `outputError` / `outputErrorMessage` rather than panics", "state writes are explicit and error-checked"}, "No error handling conventions inferred."),
		"## Testing",
		bulletList(facts.TestFiles, "No test files detected."),
		"## Example Source Files",
		bulletList(examples, "No representative source files detected."),
	})
}

func renderSurveySentinel(generatedAt string, facts codexWorkspaceFacts, dispatch codexSurveyorDispatch) string {
	testFramework := "Go `testing` package"
	if len(facts.TestFiles) == 0 {
		testFramework = "No test suite detected"
	}
	return renderSurveyDoc("SENTINEL-PROTOCOLS", generatedAt, dispatch.Name, []string{
		"## Test Framework",
		fmt.Sprintf("- %s", testFramework),
		"## Test File Organization",
		bulletList(facts.TestFiles, "No test files detected."),
		"## Test Structure",
		bulletList([]string{"table-driven Go tests", "temporary workspace helpers", "JSON envelope assertions"}, "No test structure inferred."),
		"## Coverage Targets",
		bulletList([]string{"command behavior", "state mutation", "visual rendering", "cross-platform storage"}, "No coverage targets inferred."),
		"## Common Patterns",
		bulletList([]string{"`saveGlobals(t)` and `resetRootCmd(t)` helpers", "`setupBuildFlowTest` temp workspace setup", "`go test ./...` as the main verification command"}, "No testing patterns inferred."),
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
	if len(facts.KeyDependencies) == 0 && facts.FileCount > 10 {
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
