package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/agent"
	"github.com/spf13/cobra"
)

type codexExternalColonizeCompletion struct {
	ColonizeManifest *codexColonizeManifest  `json:"colonize_manifest,omitempty"`
	SurveyManifest   *codexColonizeManifest  `json:"survey_manifest,omitempty"`
	Manifest         *codexColonizeManifest  `json:"manifest,omitempty"`
	Dispatches       []codexSurveyorDispatch `json:"dispatches,omitempty"`
	Results          []codexSurveyorDispatch `json:"results,omitempty"`
	Workers          []codexSurveyorDispatch `json:"workers,omitempty"`
}

var colonizeFinalizeCmd = &cobra.Command{
	Use:   "colonize-finalize",
	Short: "Record externally spawned surveyor workers as the territory survey",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		completionPath, _ := cmd.Flags().GetString("completion-file")
		completion, err := loadExternalColonizeCompletion(completionPath)
		if err != nil {
			outputError(1, err.Error(), nil)
			return nil
		}
		result, err := runCodexColonizeFinalize(skillWorkspaceRoot(), completion)
		if err != nil {
			outputError(1, err.Error(), nil)
			return nil
		}
		outputWorkflow(result, renderColonizeVisual(result))
		return nil
	},
}

func loadExternalColonizeCompletion(path string) (codexExternalColonizeCompletion, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return codexExternalColonizeCompletion{}, fmt.Errorf("flag --completion-file is required")
	}
	var data []byte
	var err error
	if path == "-" {
		data, err = io.ReadAll(os.Stdin)
	} else {
		data, err = os.ReadFile(path)
	}
	if err != nil {
		return codexExternalColonizeCompletion{}, fmt.Errorf("read completion file: %w", err)
	}

	var completion codexExternalColonizeCompletion
	if err := json.Unmarshal(data, &completion); err != nil {
		return codexExternalColonizeCompletion{}, fmt.Errorf("parse completion file: %w", err)
	}
	if completion.activeManifest() != nil {
		return completion, nil
	}

	var envelope struct {
		Result codexExternalColonizeCompletion `json:"result"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		return codexExternalColonizeCompletion{}, fmt.Errorf("parse completion envelope: %w", err)
	}
	if envelope.Result.activeManifest() == nil {
		return codexExternalColonizeCompletion{}, fmt.Errorf("completion file must include colonize_manifest")
	}
	return envelope.Result, nil
}

func (c codexExternalColonizeCompletion) activeManifest() *codexColonizeManifest {
	if c.ColonizeManifest != nil {
		return c.ColonizeManifest
	}
	if c.SurveyManifest != nil {
		return c.SurveyManifest
	}
	return c.Manifest
}

func (c codexExternalColonizeCompletion) workerResults() []codexSurveyorDispatch {
	results := make([]codexSurveyorDispatch, 0, len(c.Dispatches)+len(c.Results)+len(c.Workers))
	results = append(results, c.Dispatches...)
	results = append(results, c.Results...)
	results = append(results, c.Workers...)
	return results
}

func runCodexColonizeFinalize(root string, completion codexExternalColonizeCompletion) (map[string]interface{}, error) {
	if store == nil {
		return nil, fmt.Errorf("no store initialized")
	}
	manifest := completion.activeManifest()
	if manifest == nil {
		return nil, fmt.Errorf("completion file must include colonize_manifest")
	}
	if (manifest.DispatchMode != "plan-only" && manifest.DispatchMode != "agent-delegate") || !manifest.RequiresFinalizer {
		return nil, fmt.Errorf("colonize_manifest must come from `aether colonize --plan-only` or an agent-delegate colonize response")
	}
	if len(manifest.Dispatches) == 0 {
		return nil, fmt.Errorf("colonize_manifest contains no dispatches")
	}
	if strings.TrimSpace(manifest.Root) != "" && !sameCleanPath(manifest.Root, root) {
		return nil, fmt.Errorf("colonize_manifest root does not match current workspace (manifest=%s current=%s)", manifest.Root, root)
	}

	facts, err := surveyWorkspace(root)
	if err != nil {
		return nil, err
	}

	surveyDir := filepath.Join(store.BasePath(), "survey")
	if surveyDocsExist(surveyDir) && !manifest.ForceResurvey && !manifest.ExistingSurvey {
		return nil, fmt.Errorf("existing territory survey found; rerun `aether colonize --plan-only --force-resurvey` before finalizing a replacement")
	}
	if err := os.MkdirAll(surveyDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create survey directory: %w", err)
	}

	now := time.Now().UTC()
	runHandle, err := beginRuntimeSpawnRun("colonize", now)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize colonize run: %w", err)
	}
	runStatus := "failed"
	defer func() {
		finishRuntimeSpawnRun(runHandle, runStatus, time.Now().UTC())
	}()

	dispatches, err := mergeExternalSurveyResults(*manifest, completion.workerResults())
	if err != nil {
		return nil, err
	}
	if err := recordExternalSurveySpawnTree(dispatches); err != nil {
		return nil, err
	}

	surveyFiles, preservedWorkerArtifacts, err := writeSurveyArtifacts(root, surveyDir, facts, dispatches, manifest.Snapshots)
	if err != nil {
		return nil, err
	}
	artifactSource := "runtime-synthesis"
	if preservedWorkerArtifacts > 0 {
		artifactSource = "external-task"
	}
	if err := writeSurveyCompatibilityJSON(surveyDir, facts); err != nil {
		return nil, err
	}
	codegraphStats, codegraphWarning := runColonizeCodebaseGraph(root)

	surveyedAt := now.Format(time.RFC3339)
	if err := updateSurveyState(surveyedAt, len(surveyFiles)); err != nil {
		return nil, err
	}
	emitColonizeCeremonyDispatchSequence("aether-colonize-finalize", dispatches)
	updateSessionSummary("colonize-finalize", "aether plan", fmt.Sprintf("Territory surveyed by external workers (%d documents)", len(surveyFiles)))
	runStatus = summarizeRunStatus(surveyorStatuses(dispatches)...)

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
		"surveyors":          surveyorDispatchMaps(dispatches),
		"dispatches":         surveyorDispatchMaps(dispatches),
		"existing_survey":    manifest.ExistingSurvey,
		"force_resurvey":     manifest.ForceResurvey,
		"territory_surveyed": surveyedAt,
		"dispatch_mode":      "external-task",
		"dispatch_contract":  manifest.DispatchContract,
		"artifact_source":    artifactSource,
		"survey_warning":     "",
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
	return result, nil
}

func sameCleanPath(a, b string) bool {
	aAbs, aErr := filepath.Abs(filepath.Clean(strings.TrimSpace(a)))
	bAbs, bErr := filepath.Abs(filepath.Clean(strings.TrimSpace(b)))
	if aErr != nil || bErr != nil {
		return filepath.Clean(a) == filepath.Clean(b)
	}
	if aReal, err := filepath.EvalSymlinks(aAbs); err == nil {
		aAbs = aReal
	}
	if bReal, err := filepath.EvalSymlinks(bAbs); err == nil {
		bAbs = bReal
	}
	return aAbs == bAbs
}

func mergeExternalSurveyResults(manifest codexColonizeManifest, results []codexSurveyorDispatch) ([]codexSurveyorDispatch, error) {
	resultByName := make(map[string]codexSurveyorDispatch, len(results))
	resultByTaskID := make(map[string]codexSurveyorDispatch, len(results))
	for _, result := range results {
		if name := strings.TrimSpace(result.Name); name != "" {
			resultByName[name] = result
		}
		if taskID := strings.TrimSpace(result.TaskID); taskID != "" {
			resultByTaskID[taskID] = result
		}
	}

	merged := make([]codexSurveyorDispatch, 0, len(manifest.Dispatches))
	for _, planned := range manifest.Dispatches {
		result, ok := resultByName[strings.TrimSpace(planned.Name)]
		if !ok && strings.TrimSpace(planned.TaskID) != "" {
			result, ok = resultByTaskID[strings.TrimSpace(planned.TaskID)]
		}
		if !ok {
			return nil, fmt.Errorf("missing external surveyor result for %s", planned.Name)
		}
		status := normalizeRuntimeDispatchStatus(result.Status)
		if status == "" || status == "spawned" {
			status = "completed"
		}
		if status != "completed" && status != "passed" && status != "code_written" {
			summary := strings.TrimSpace(result.Summary)
			if summary == "" && len(result.Blockers) > 0 {
				summary = strings.Join(result.Blockers, "; ")
			}
			if summary == "" {
				summary = "no summary provided"
			}
			return nil, fmt.Errorf("surveyor %s did not complete: %s (%s)", planned.Name, status, summary)
		}

		planned.Status = "completed"
		if strings.TrimSpace(result.Name) != "" {
			planned.Name = strings.TrimSpace(result.Name)
		}
		planned.Summary = strings.TrimSpace(result.Summary)
		planned.Blockers = append([]string{}, result.Blockers...)
		planned.Duration = result.Duration
		planned.FilesCreated = append([]string{}, result.FilesCreated...)
		planned.FilesModified = append([]string{}, result.FilesModified...)
		planned.Claimed = uniqueSortedStrings(append(append([]string{}, planned.FilesCreated...), planned.FilesModified...))
		if len(planned.Claimed) == 0 {
			claimed := make([]string, 0, len(planned.OutputPaths))
			for _, outputPath := range planned.OutputPaths {
				if _, err := os.Stat(filepath.Join(manifest.Root, filepath.FromSlash(outputPath))); err == nil {
					claimed = append(claimed, outputPath)
				}
			}
			planned.Claimed = uniqueSortedStrings(claimed)
		}
		merged = append(merged, planned)
	}
	return merged, nil
}

func recordExternalSurveySpawnTree(dispatches []codexSurveyorDispatch) error {
	spawnTree := agent.NewSpawnTree(store, "spawn-tree.txt")
	for _, dispatch := range dispatches {
		if err := spawnTree.RecordSpawn("Queen", "surveyor", dispatch.Name, dispatch.Task, 1); err != nil {
			return fmt.Errorf("failed to record surveyor spawn: %w", err)
		}
		summary := strings.TrimSpace(dispatch.Summary)
		if summary == "" {
			summary = strings.Join(dispatch.Outputs, ", ")
		}
		if err := spawnTree.UpdateStatus(dispatch.Name, dispatch.Status, summary); err != nil {
			return fmt.Errorf("failed to update surveyor completion: %w", err)
		}
	}
	return nil
}

func surveyorStatuses(dispatches []codexSurveyorDispatch) []string {
	statuses := make([]string, 0, len(dispatches))
	for _, dispatch := range dispatches {
		statuses = append(statuses, dispatch.Status)
	}
	return statuses
}
