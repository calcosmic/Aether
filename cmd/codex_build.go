package cmd

import (
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/agent"
	"github.com/calcosmic/Aether/pkg/colony"
)

type codexBuildDispatch struct {
	Stage     string   `json:"stage"`
	Wave      int      `json:"wave,omitempty"`
	Caste     string   `json:"caste"`
	Name      string   `json:"name"`
	Task      string   `json:"task"`
	Status    string   `json:"status"`
	TaskID    string   `json:"task_id,omitempty"`
	TaskIndex int      `json:"task_index,omitempty"`
	DependsOn []string `json:"depends_on,omitempty"`
	Outputs   []string `json:"outputs,omitempty"`
}

type codexBuildTaskPlan struct {
	ID        string   `json:"id,omitempty"`
	Goal      string   `json:"goal"`
	Status    string   `json:"status"`
	Wave      int      `json:"wave,omitempty"`
	DependsOn []string `json:"depends_on,omitempty"`
}

type codexBuildManifest struct {
	Phase           int                  `json:"phase"`
	PhaseName       string               `json:"phase_name"`
	Goal            string               `json:"goal,omitempty"`
	Root            string               `json:"root"`
	ColonyDepth     string               `json:"colony_depth"`
	GeneratedAt     string               `json:"generated_at"`
	State           string               `json:"state"`
	Checkpoint      string               `json:"checkpoint"`
	ClaimsPath      string               `json:"claims_path"`
	Playbooks       []string             `json:"playbooks"`
	WorkerBriefs    []string             `json:"worker_briefs"`
	Dispatches      []codexBuildDispatch `json:"dispatches"`
	Tasks           []codexBuildTaskPlan `json:"tasks"`
	SuccessCriteria []string             `json:"success_criteria"`
}

type codexBuildClaims struct {
	FilesCreated  []string `json:"files_created"`
	FilesModified []string `json:"files_modified"`
	BuildPhase    int      `json:"build_phase"`
	Timestamp     string   `json:"timestamp"`
}

func runCodexBuild(root string, phaseNum int) (map[string]interface{}, error) {
	if store == nil {
		return nil, fmt.Errorf("no store initialized")
	}

	var state colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
		return nil, fmt.Errorf("No colony initialized. Run `aether init \"goal\"` first.")
	}
	if len(state.Plan.Phases) == 0 {
		return nil, fmt.Errorf("No project plan. Run `aether plan` first.")
	}
	if phaseNum < 1 || phaseNum > len(state.Plan.Phases) {
		return nil, fmt.Errorf("phase %d not found (plan has %d phases)", phaseNum, len(state.Plan.Phases))
	}
	if err := validateCodexBuildState(state, phaseNum); err != nil {
		return nil, err
	}

	startedAt := time.Now().UTC()
	phase := state.Plan.Phases[phaseNum-1]
	depth := strings.TrimSpace(state.ColonyDepth)
	if depth == "" {
		depth = "standard"
	}
	playbooks := codexBuildPlaybooks()
	dispatches := plannedBuildDispatches(phase, depth)
	dispatches, err := ensureUniqueBuildDispatchNames(dispatches)
	if err != nil {
		return nil, err
	}
	parallelWaves := buildParallelWaves(dispatches)
	checkpointRel := filepath.ToSlash(filepath.Join("checkpoints", fmt.Sprintf("pre-build-phase-%d.json", phaseNum)))
	buildDirRel := filepath.ToSlash(filepath.Join("build", fmt.Sprintf("phase-%d", phaseNum)))
	manifestRel := filepath.ToSlash(filepath.Join(buildDirRel, "manifest.json"))
	claimsRel := "last-build-claims.json"

	if err := store.SaveJSON(checkpointRel, state); err != nil {
		return nil, fmt.Errorf("failed to checkpoint colony state: %w", err)
	}

	updatedState := state
	applyCodexBuildState(&updatedState, phaseNum, startedAt)
	updatedPhase := updatedState.Plan.Phases[phaseNum-1]

	briefPaths, dispatches, err := writeCodexBuildArtifacts(root, updatedState, updatedPhase, buildDirRel, checkpointRel, claimsRel, playbooks, dispatches, startedAt)
	if err != nil {
		return nil, err
	}
	if err := writeCodexBuildClaims(claimsRel, phaseNum, startedAt); err != nil {
		return nil, err
	}
	if err := recordCodexBuildDispatches(dispatches); err != nil {
		return nil, err
	}
	if err := store.SaveJSON("COLONY_STATE.json", updatedState); err != nil {
		return nil, fmt.Errorf("failed to save colony state: %w", err)
	}
	_ = updateCodexBuildContext(updatedPhase, dispatches, parallelWaves, startedAt)

	updateSessionSummary("build", "aether continue", fmt.Sprintf("Phase %d dispatched to %d workers across %d waves", phaseNum, len(dispatches), max(parallelWaves, 1)))

	dispatchMaps := make([]map[string]interface{}, 0, len(dispatches))
	for _, dispatch := range dispatches {
		entry := map[string]interface{}{
			"stage":  dispatch.Stage,
			"caste":  dispatch.Caste,
			"name":   dispatch.Name,
			"task":   dispatch.Task,
			"status": dispatch.Status,
		}
		if dispatch.Wave > 0 {
			entry["wave"] = dispatch.Wave
		}
		if dispatch.TaskID != "" {
			entry["task_id"] = dispatch.TaskID
		}
		if len(dispatch.DependsOn) > 0 {
			entry["depends_on"] = dispatch.DependsOn
		}
		if len(dispatch.Outputs) > 0 {
			entry["outputs"] = dispatch.Outputs
		}
		dispatchMaps = append(dispatchMaps, entry)
	}

	result := map[string]interface{}{
		"phase":          phaseNum,
		"phase_name":     updatedPhase.Name,
		"state":          updatedState.State,
		"playbooks":      playbooks,
		"next":           "aether continue",
		"currentTask":    updatedPhase.Tasks,
		"dispatches":     dispatchMaps,
		"dispatch_count": len(dispatches),
		"parallel_waves": parallelWaves,
		"checkpoint":     displayDataPath(checkpointRel),
		"build_dir":      displayDataPath(buildDirRel),
		"manifest":       displayDataPath(manifestRel),
		"worker_briefs":  briefPaths,
		"claims_path":    displayDataPath(claimsRel),
	}
	return result, nil
}

func validateCodexBuildState(state colony.ColonyState, phaseNum int) error {
	if state.State == colony.StateEXECUTING && state.CurrentPhase > 0 && state.CurrentPhase != phaseNum {
		return fmt.Errorf("phase %d is already active; run `aether continue` before dispatching phase %d", state.CurrentPhase, phaseNum)
	}

	for i := 0; i < phaseNum-1; i++ {
		if state.Plan.Phases[i].Status != colony.PhaseCompleted {
			return fmt.Errorf("phase %d is not complete yet; build phases in order", state.Plan.Phases[i].ID)
		}
	}

	selected := state.Plan.Phases[phaseNum-1]
	if selected.Status == colony.PhaseCompleted {
		return fmt.Errorf("phase %d is already completed", phaseNum)
	}

	if err := colony.Transition(state.State, colony.StateEXECUTING); err != nil && state.State != colony.StateEXECUTING {
		return err
	}
	return nil
}

func applyCodexBuildState(state *colony.ColonyState, phaseNum int, startedAt time.Time) {
	state.State = colony.StateEXECUTING
	state.CurrentPhase = phaseNum
	state.BuildStartedAt = &startedAt

	for i := range state.Plan.Phases {
		switch {
		case state.Plan.Phases[i].ID < phaseNum && state.Plan.Phases[i].Status != colony.PhaseCompleted:
			state.Plan.Phases[i].Status = colony.PhaseCompleted
		case state.Plan.Phases[i].ID == phaseNum:
			state.Plan.Phases[i].Status = colony.PhaseInProgress
			applyBuildTaskStatuses(&state.Plan.Phases[i])
		case state.Plan.Phases[i].Status == "":
			state.Plan.Phases[i].Status = colony.PhasePending
		}
	}

	phase := state.Plan.Phases[phaseNum-1]
	state.Events = append(trimmedEvents(state.Events),
		fmt.Sprintf("%s|phase_started|build|Phase %d: %s", startedAt.Format(time.RFC3339), phaseNum, phase.Name),
		fmt.Sprintf("%s|build_dispatched|build|Dispatched %d workers for phase %d", startedAt.Format(time.RFC3339), len(plannedBuildDispatches(phase, normalizedBuildDepth(state.ColonyDepth))), phaseNum),
	)
}

func applyBuildTaskStatuses(phase *colony.Phase) {
	waves := taskWaves(phase.Tasks)
	firstWave := map[int]bool{}
	if len(waves) > 0 {
		for _, idx := range waves[0] {
			firstWave[idx] = true
		}
	}

	for i := range phase.Tasks {
		if phase.Tasks[i].Status == colony.TaskCompleted {
			continue
		}
		if firstWave[i] {
			phase.Tasks[i].Status = colony.TaskInProgress
			continue
		}
		if phase.Tasks[i].Status == "" {
			phase.Tasks[i].Status = colony.TaskPending
		}
	}
}

func codexBuildPlaybooks() []string {
	return []string{
		".aether/docs/command-playbooks/build-prep.md",
		".aether/docs/command-playbooks/build-wave.md",
		".aether/docs/command-playbooks/build-verify.md",
		".aether/docs/command-playbooks/build-complete.md",
	}
}

func plannedBuildDispatches(phase colony.Phase, depth string) []codexBuildDispatch {
	depth = normalizedBuildDepth(depth)
	waves := taskWaves(phase.Tasks)
	dispatches := make([]codexBuildDispatch, 0, len(phase.Tasks)+4)

	for waveIdx, wave := range waves {
		for _, taskIdx := range wave {
			task := phase.Tasks[taskIdx]
			taskID := buildTaskID(task, taskIdx)
			dispatches = append(dispatches, codexBuildDispatch{
				Stage:     "wave",
				Wave:      waveIdx + 1,
				Caste:     suggestedBuildCaste(task),
				Name:      deterministicAntName(suggestedBuildCaste(task), fmt.Sprintf("phase:%d:task:%d:%s", phase.ID, taskIdx, task.Goal)),
				Task:      strings.TrimSpace(task.Goal),
				Status:    "spawned",
				TaskID:    taskID,
				TaskIndex: taskIdx,
				DependsOn: append([]string{}, task.DependsOn...),
			})
		}
	}

	if len(waves) == 0 {
		dispatches = append(dispatches, codexBuildDispatch{
			Stage:  "wave",
			Wave:   1,
			Caste:  "builder",
			Name:   deterministicAntName("builder", fmt.Sprintf("phase:%d:default", phase.ID)),
			Task:   "Build the phase objective",
			Status: "spawned",
		})
	}

	if depth == "deep" || depth == "full" {
		dispatches = append(dispatches,
			codexBuildDispatch{
				Stage:  "strategy",
				Caste:  "oracle",
				Name:   deterministicAntName("oracle", fmt.Sprintf("phase:%d:oracle", phase.ID)),
				Task:   "Phase research and implementation risks",
				Status: "spawned",
			},
			codexBuildDispatch{
				Stage:  "strategy",
				Caste:  "architect",
				Name:   deterministicAntName("architect", fmt.Sprintf("phase:%d:architect", phase.ID)),
				Task:   "Design boundaries before coding",
				Status: "spawned",
			},
		)
	}

	dispatches = append(dispatches, codexBuildDispatch{
		Stage:  "verification",
		Caste:  "watcher",
		Name:   deterministicAntName("watcher", fmt.Sprintf("phase:%d:watcher", phase.ID)),
		Task:   "Independent verification before advancement",
		Status: "spawned",
	})
	if depth == "full" {
		dispatches = append(dispatches, codexBuildDispatch{
			Stage:  "resilience",
			Caste:  "chaos",
			Name:   deterministicAntName("chaos", fmt.Sprintf("phase:%d:chaos", phase.ID)),
			Task:   "Resilience probing after verification",
			Status: "spawned",
		})
	}

	return dispatches
}

func buildParallelWaves(dispatches []codexBuildDispatch) int {
	maxWave := 0
	for _, dispatch := range dispatches {
		if dispatch.Wave > maxWave {
			maxWave = dispatch.Wave
		}
	}
	return maxWave
}

func normalizedBuildDepth(depth string) string {
	depth = strings.TrimSpace(depth)
	if depth == "" {
		return "standard"
	}
	return depth
}

func buildTaskID(task colony.Task, idx int) string {
	if task.ID != nil && strings.TrimSpace(*task.ID) != "" {
		return strings.TrimSpace(*task.ID)
	}
	return fmt.Sprintf("task-%d", idx+1)
}

func writeCodexBuildArtifacts(root string, state colony.ColonyState, phase colony.Phase, buildDirRel, checkpointRel, claimsRel string, playbooks []string, dispatches []codexBuildDispatch, startedAt time.Time) ([]string, []codexBuildDispatch, error) {
	briefPaths := make([]string, 0, len(dispatches))
	briefOutputs := map[string]string{}

	for i := range dispatches {
		briefRel := filepath.ToSlash(filepath.Join(buildDirRel, "worker-briefs", fmt.Sprintf("%s.md", dispatches[i].Name)))
		content := renderCodexBuildWorkerBrief(root, phase, dispatches[i], playbooks, startedAt)
		if err := store.AtomicWrite(briefRel, []byte(content)); err != nil {
			return nil, nil, fmt.Errorf("failed to write worker brief for %s: %w", dispatches[i].Name, err)
		}
		displayPath := displayDataPath(briefRel)
		briefPaths = append(briefPaths, displayPath)
		briefOutputs[dispatches[i].Name] = displayPath
	}
	sort.Strings(briefPaths)

	taskPlans := make([]codexBuildTaskPlan, 0, len(phase.Tasks))
	waves := taskWaves(phase.Tasks)
	taskWave := map[int]int{}
	for waveIdx, wave := range waves {
		for _, idx := range wave {
			taskWave[idx] = waveIdx + 1
		}
	}
	for idx, task := range phase.Tasks {
		taskPlans = append(taskPlans, codexBuildTaskPlan{
			ID:        buildTaskID(task, idx),
			Goal:      task.Goal,
			Status:    task.Status,
			Wave:      taskWave[idx],
			DependsOn: append([]string{}, task.DependsOn...),
		})
	}

	goal := ""
	if state.Goal != nil {
		goal = strings.TrimSpace(*state.Goal)
	}
	for i := range dispatches {
		if output := briefOutputs[dispatches[i].Name]; output != "" {
			dispatches[i].Outputs = []string{output}
		}
	}

	manifest := codexBuildManifest{
		Phase:           phase.ID,
		PhaseName:       phase.Name,
		Goal:            goal,
		Root:            root,
		ColonyDepth:     normalizedBuildDepth(state.ColonyDepth),
		GeneratedAt:     startedAt.Format(time.RFC3339),
		State:           string(state.State),
		Checkpoint:      displayDataPath(checkpointRel),
		ClaimsPath:      displayDataPath(claimsRel),
		Playbooks:       append([]string{}, playbooks...),
		WorkerBriefs:    briefPaths,
		Dispatches:      dispatches,
		Tasks:           taskPlans,
		SuccessCriteria: append([]string{}, phase.SuccessCriteria...),
	}
	manifestRel := filepath.ToSlash(filepath.Join(buildDirRel, "manifest.json"))
	if err := store.SaveJSON(manifestRel, manifest); err != nil {
		return nil, nil, fmt.Errorf("failed to write build manifest: %w", err)
	}

	return briefPaths, dispatches, nil
}

func renderCodexBuildWorkerBrief(root string, phase colony.Phase, dispatch codexBuildDispatch, playbooks []string, startedAt time.Time) string {
	var b strings.Builder
	b.WriteString("# Codex Build Dispatch\n\n")
	b.WriteString(fmt.Sprintf("- Worker: %s\n", dispatch.Name))
	b.WriteString(fmt.Sprintf("- Caste: %s\n", dispatch.Caste))
	if dispatch.Wave > 0 {
		b.WriteString(fmt.Sprintf("- Wave: %d\n", dispatch.Wave))
	}
	b.WriteString(fmt.Sprintf("- Phase: %d — %s\n", phase.ID, phase.Name))
	b.WriteString(fmt.Sprintf("- Started: %s\n", startedAt.Format(time.RFC3339)))
	b.WriteString(fmt.Sprintf("- Workspace: %s\n", root))
	b.WriteString("\n## Assignment\n\n")
	b.WriteString(strings.TrimSpace(dispatch.Task))
	b.WriteString("\n")

	if strings.TrimSpace(phase.Description) != "" {
		b.WriteString("\n## Phase Objective\n\n")
		b.WriteString(strings.TrimSpace(phase.Description))
		b.WriteString("\n")
	}

	if len(dispatch.DependsOn) > 0 {
		b.WriteString("\n## Dependencies\n\n")
		for _, dep := range dispatch.DependsOn {
			dep = strings.TrimSpace(dep)
			if dep == "" {
				continue
			}
			b.WriteString("- ")
			b.WriteString(dep)
			b.WriteString("\n")
		}
	}

	relatedTask := findDispatchTask(phase, dispatch)
	if relatedTask != nil {
		if len(relatedTask.Constraints) > 0 {
			b.WriteString("\n## Constraints\n\n")
			for _, item := range relatedTask.Constraints {
				item = strings.TrimSpace(item)
				if item == "" {
					continue
				}
				b.WriteString("- ")
				b.WriteString(item)
				b.WriteString("\n")
			}
		}
		if len(relatedTask.Hints) > 0 {
			b.WriteString("\n## Hints\n\n")
			for _, item := range relatedTask.Hints {
				item = strings.TrimSpace(item)
				if item == "" {
					continue
				}
				b.WriteString("- ")
				b.WriteString(item)
				b.WriteString("\n")
			}
		}
		if len(relatedTask.SuccessCriteria) > 0 {
			b.WriteString("\n## Task Success Criteria\n\n")
			for _, item := range relatedTask.SuccessCriteria {
				item = strings.TrimSpace(item)
				if item == "" {
					continue
				}
				b.WriteString("- ")
				b.WriteString(item)
				b.WriteString("\n")
			}
		}
	}

	if len(phase.SuccessCriteria) > 0 {
		b.WriteString("\n## Phase Success Criteria\n\n")
		for _, item := range phase.SuccessCriteria {
			item = strings.TrimSpace(item)
			if item == "" {
				continue
			}
			b.WriteString("- ")
			b.WriteString(item)
			b.WriteString("\n")
		}
	}

	b.WriteString("\n## Relevant Playbooks\n\n")
	for _, playbook := range buildPlaybooksForDispatch(dispatch, playbooks) {
		b.WriteString("- ")
		b.WriteString(playbook)
		b.WriteString("\n")
	}

	b.WriteString("\n## Expected Output\n\n")
	b.WriteString("- ")
	b.WriteString(expectedDispatchOutcome(dispatch))
	b.WriteString("\n")
	return b.String()
}

func findDispatchTask(phase colony.Phase, dispatch codexBuildDispatch) *colony.Task {
	if dispatch.TaskID == "" {
		return nil
	}
	for i := range phase.Tasks {
		if buildTaskID(phase.Tasks[i], i) == dispatch.TaskID {
			return &phase.Tasks[i]
		}
	}
	return nil
}

func buildPlaybooksForDispatch(dispatch codexBuildDispatch, playbooks []string) []string {
	filtered := make([]string, 0, len(playbooks))
	for _, playbook := range playbooks {
		switch dispatch.Caste {
		case "oracle", "architect":
			if strings.Contains(playbook, "build-prep") || strings.Contains(playbook, "build-wave") {
				filtered = append(filtered, playbook)
			}
		case "watcher", "chaos":
			if strings.Contains(playbook, "build-verify") || strings.Contains(playbook, "build-complete") {
				filtered = append(filtered, playbook)
			}
		default:
			if strings.Contains(playbook, "build-wave") || strings.Contains(playbook, "build-complete") {
				filtered = append(filtered, playbook)
			}
		}
	}
	if len(filtered) == 0 {
		return append([]string{}, playbooks...)
	}
	return filtered
}

func expectedDispatchOutcome(dispatch codexBuildDispatch) string {
	switch dispatch.Caste {
	case "scout":
		return "Research notes or documentation updates that unblock implementation."
	case "watcher":
		return "Independent verification notes with concrete evidence for `aether continue`."
	case "oracle":
		return "Implementation risks, unknowns, and recommended handling before deeper coding."
	case "architect":
		return "Design boundaries, interfaces, and sequencing guidance for the phase."
	case "chaos":
		return "Resilience findings and failure cases worth checking before advancement."
	default:
		return "Concrete code changes plus a truthful summary of files touched and verification run."
	}
}

func writeCodexBuildClaims(relPath string, phaseNum int, startedAt time.Time) error {
	claims := codexBuildClaims{
		FilesCreated:  []string{},
		FilesModified: []string{},
		BuildPhase:    phaseNum,
		Timestamp:     startedAt.Format(time.RFC3339),
	}
	if err := store.SaveJSON(relPath, claims); err != nil {
		return fmt.Errorf("failed to write build claims: %w", err)
	}
	return nil
}

func recordCodexBuildDispatches(dispatches []codexBuildDispatch) error {
	spawnTree := agent.NewSpawnTree(store, "spawn-tree.txt")
	for _, dispatch := range dispatches {
		if err := spawnTree.RecordSpawn("Queen", dispatch.Caste, dispatch.Name, dispatch.Task, 1); err != nil {
			return fmt.Errorf("failed to record build dispatch %s: %w", dispatch.Name, err)
		}
	}
	return nil
}

func ensureUniqueBuildDispatchNames(dispatches []codexBuildDispatch) ([]codexBuildDispatch, error) {
	spawnTree := agent.NewSpawnTree(store, "spawn-tree.txt")
	entries, err := spawnTree.Parse()
	if err != nil {
		return nil, fmt.Errorf("failed to read spawn tree for name allocation: %w", err)
	}

	used := make(map[string]bool, len(entries)+len(dispatches))
	for _, entry := range entries {
		used[entry.AgentName] = true
	}

	allocated := make([]codexBuildDispatch, len(dispatches))
	for i, dispatch := range dispatches {
		candidate := dispatch.Name
		if used[candidate] {
			base := candidate
			for attempt := 2; ; attempt++ {
				candidate = fmt.Sprintf("%s-r%d", base, attempt)
				if !used[candidate] {
					break
				}
			}
		}
		dispatch.Name = candidate
		used[candidate] = true
		allocated[i] = dispatch
	}
	return allocated, nil
}

func updateCodexBuildContext(phase colony.Phase, dispatches []codexBuildDispatch, parallelWaves int, startedAt time.Time) error {
	data, err := store.ReadFile(contextFileName)
	if err != nil {
		return nil
	}

	content := string(data)
	content = replaceContextTableRow(content, "Last Updated", startedAt.Format(time.RFC3339))
	content = replaceContextTableRow(content, "Current Phase", strconv.Itoa(phase.ID))
	content = replaceContextTableRow(content, "Phase Name", phase.Name)
	content = replaceContextTableRow(content, "Safe to Clear?", "NO — Build in progress")
	content = replaceContextSectionContent(content, "What's In Progress", fmt.Sprintf(
		"**Phase %d Build IN PROGRESS**\n- Workers: %d | Tasks: %d | Waves: %d\n- Phase: %s\n- Started: %s",
		phase.ID, len(dispatches), len(phase.Tasks), max(parallelWaves, 1), phase.Name, startedAt.Format(time.RFC3339),
	))
	for _, dispatch := range dispatches {
		content = appendWorkerSpawnEntry(content, dispatch.Name, dispatch.Caste, dispatch.Task, startedAt.Format(time.RFC3339))
	}

	return store.AtomicWrite(contextFileName, []byte(content))
}

func displayDataPath(rel string) string {
	return filepath.ToSlash(filepath.Join(".aether", "data", rel))
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
