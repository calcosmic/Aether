package cmd

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/agent"
	"github.com/calcosmic/Aether/pkg/colony"
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

const (
	colonizeFinalizeManifestMaxAge     = 24 * time.Hour
	colonizeFinalizeManifestFutureSkew = 5 * time.Minute
	territoryPublicationTransactional  = "lifecycle-transaction/v1"
)

type territoryPublicationRequest struct {
	Root           string
	GeneratedAt    time.Time
	TransactionID  string
	BaselineDigest string
	Artifacts      map[string][]byte
	Fault          lifecycleTransactionFaultHook
}

type territoryPublicationResult struct {
	Freshness     SurveyFreshnessResult
	Receipt       colony.LifecycleReceipt
	StateRecorded bool
}

// publishTerritorySnapshot is the sole live-write boundary for a complete
// territory refresh. Every survey artifact, the immutable snapshot metadata,
// and (when present) colony state are committed through one recoverable
// lifecycle transaction.
func publishTerritorySnapshot(request territoryPublicationRequest) (territoryPublicationResult, error) {
	if store == nil {
		return territoryPublicationResult{}, fmt.Errorf("no store initialized")
	}
	root, err := canonicalTerritoryRoot(request.Root)
	if err != nil {
		return territoryPublicationResult{}, fmt.Errorf("publish territory: %w", err)
	}
	dataRoot, err := filepath.Abs(store.BasePath())
	if err != nil {
		return territoryPublicationResult{}, fmt.Errorf("publish territory: resolve lifecycle data root: %w", err)
	}
	dataRoot = filepath.Clean(dataRoot)
	if !sameCleanPath(filepath.Dir(filepath.Dir(dataRoot)), root) {
		return territoryPublicationResult{}, fmt.Errorf("publish territory: lifecycle store does not belong to repository root")
	}
	if request.GeneratedAt.IsZero() {
		request.GeneratedAt = time.Now().UTC()
	}
	request.GeneratedAt = request.GeneratedAt.UTC()
	if strings.TrimSpace(request.TransactionID) == "" {
		return territoryPublicationResult{}, fmt.Errorf("publish territory: transaction_id is required")
	}
	if strings.TrimSpace(request.BaselineDigest) == "" {
		return territoryPublicationResult{}, fmt.Errorf("publish territory: baseline_digest is required")
	}
	if actual := territorySurveyBaselineDigest(root); actual != request.BaselineDigest {
		return territoryPublicationResult{}, fmt.Errorf("publish territory: survey baseline changed (manifest=%s current=%s)", request.BaselineDigest, actual)
	}

	expected := requiredTerritoryArtifactPaths()
	expectedSet := make(map[string]struct{}, len(expected))
	normalized := make(map[string][]byte, len(expected))
	for _, rel := range expected {
		rel = filepath.ToSlash(rel)
		expectedSet[rel] = struct{}{}
		content, ok := request.Artifacts[rel]
		if !ok {
			return territoryPublicationResult{}, fmt.Errorf("publish territory: required artifact %s is missing", rel)
		}
		if strings.HasSuffix(strings.ToLower(rel), ".md") {
			if err := validateTerritoryPublicationMarkdown(content); err != nil {
				return territoryPublicationResult{}, fmt.Errorf("publish territory: invalid artifact %s: %w", rel, err)
			}
		} else if !json.Valid(bytes.TrimSpace(content)) {
			return territoryPublicationResult{}, fmt.Errorf("publish territory: invalid JSON artifact %s", rel)
		}
		normalized[rel] = bytes.Clone(content)
	}
	for rel := range request.Artifacts {
		clean := filepath.ToSlash(filepath.Clean(strings.TrimSpace(rel)))
		if clean != rel {
			return territoryPublicationResult{}, fmt.Errorf("publish territory: artifact path %q is not canonical", rel)
		}
		if _, ok := expectedSet[clean]; !ok {
			return territoryPublicationResult{}, fmt.Errorf("publish territory: undeclared artifact %s", clean)
		}
	}

	revision, err := currentTerritoryRevision(root)
	if err != nil {
		return territoryPublicationResult{}, fmt.Errorf("publish territory: %w", err)
	}
	digests := make(map[string]string, len(normalized))
	for rel, content := range normalized {
		digest := sha256.Sum256(content)
		digests[rel] = hex.EncodeToString(digest[:])
	}
	snapshot := territorySnapshotMetadata{
		SchemaVersion:      territorySnapshotSchemaVersion,
		RepositoryIdentity: stableRepoIdentity(root),
		RepositoryRoot:     root,
		SourceRevision:     revision,
		GeneratedAt:        request.GeneratedAt,
		ArtifactDigests:    digests,
	}
	snapshot.SnapshotID = computeTerritorySnapshotID(snapshot)
	snapshotBytes, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return territoryPublicationResult{}, fmt.Errorf("publish territory: encode snapshot metadata: %w", err)
	}
	snapshotBytes = append(snapshotBytes, '\n')

	repositoryRoot, err := filepath.Abs(request.Root)
	if err != nil {
		return territoryPublicationResult{}, fmt.Errorf("publish territory: resolve repository transaction root: %w", err)
	}
	tx, err := beginLifecycleTransaction(lifecycleTransactionConfig{
		TransactionID: request.TransactionID,
		Command:       "territory-refresh",
		Allowlist: lifecycleTransactionAllowlist{
			RepositoryRoot:    filepath.Clean(repositoryRoot),
			LifecycleDataRoot: dataRoot,
		},
		Fault: request.Fault,
	})
	if err != nil {
		return territoryPublicationResult{}, err
	}
	for _, rel := range expected {
		dataRelative := strings.TrimPrefix(filepath.ToSlash(rel), ".aether/data/")
		if err := tx.DeclareWrite(lifecycleTransactionRootData, filepath.FromSlash(dataRelative), normalized[filepath.ToSlash(rel)]); err != nil {
			return territoryPublicationResult{}, err
		}
	}
	metadataRelative := strings.TrimPrefix(filepath.ToSlash(territorySnapshotRelativePath), ".aether/data/")
	if err := tx.DeclareWrite(lifecycleTransactionRootData, filepath.FromSlash(metadataRelative), snapshotBytes); err != nil {
		return territoryPublicationResult{}, err
	}

	stateRecorded := false
	statePath := filepath.Join(dataRoot, "COLONY_STATE.json")
	if _, statErr := os.Lstat(statePath); statErr == nil {
		var state colony.ColonyState
		if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
			return territoryPublicationResult{}, fmt.Errorf("publish territory: load colony state: %w", err)
		}
		surveyedAt := request.GeneratedAt.Format(time.RFC3339)
		state.State = colony.StateREADY
		state.TerritorySurveyed = &surveyedAt
		state.Events = append(trimmedEvents(state.Events), fmt.Sprintf("%s|territory_surveyed|colonize|Territory surveyed: %d documents", surveyedAt, len(requiredSurveyMarkdownFiles)))
		stateBytes, err := json.MarshalIndent(state, "", "  ")
		if err != nil {
			return territoryPublicationResult{}, fmt.Errorf("publish territory: encode colony state: %w", err)
		}
		stateBytes = append(stateBytes, '\n')
		if err := tx.DeclareWrite(lifecycleTransactionRootData, "COLONY_STATE.json", stateBytes); err != nil {
			return territoryPublicationResult{}, err
		}
		stateRecorded = true
	} else if !os.IsNotExist(statErr) {
		return territoryPublicationResult{}, fmt.Errorf("publish territory: inspect colony state: %w", statErr)
	}

	receipt, err := tx.Commit()
	if err != nil {
		return territoryPublicationResult{}, err
	}
	freshness := classifySurveyFreshness(root, time.Now().UTC())
	if freshness.Freshness != colony.SurveyFreshnessFresh || freshness.SnapshotID != snapshot.SnapshotID {
		return territoryPublicationResult{}, fmt.Errorf("publish territory: committed snapshot failed freshness verification (%s: %v)", freshness.Freshness, freshness.ReasonCodes)
	}
	freshness.Refreshed = true
	return territoryPublicationResult{Freshness: freshness, Receipt: receipt, StateRecorded: stateRecorded}, nil
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
			return renderedErrorExit(1)
		}
		result, err := runCodexColonizeFinalize(skillWorkspaceRoot(), completion)
		if err != nil {
			outputError(1, err.Error(), nil)
			return renderedErrorExit(1)
		}
		closeLifecycleCommand(result, "colonize", "", "")
		outputWorkflow(result, renderColonizeVisual(result))
		return nil
	},
}

func loadExternalColonizeCompletion(path string) (codexExternalColonizeCompletion, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return codexExternalColonizeCompletion{}, fmt.Errorf("flag --completion-file is required")
	}
	if err := validateFinalizerCompletionFilePath(path); err != nil {
		return codexExternalColonizeCompletion{}, err
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
	now := time.Now().UTC()
	if err := validateCodexColonizeManifestFreshness(*manifest, now); err != nil {
		return nil, err
	}

	facts, err := surveyWorkspace(root)
	if err != nil {
		return nil, err
	}
	if err := validateCodexColonizeManifestWorkspace(*manifest, facts); err != nil {
		return nil, err
	}
	if manifest.PublicationMode == territoryPublicationTransactional {
		return runTransactionalColonizeFinalize(root, *manifest, completion.workerResults(), facts, now)
	}

	surveyDir := filepath.Join(store.BasePath(), "survey")
	if surveyDocsExist(surveyDir) && !manifest.ForceResurvey && !manifest.ExistingSurvey && surveyDocsExistedInManifest(*manifest) {
		return nil, fmt.Errorf("existing territory survey found; rerun `aether colonize --plan-only --force-resurvey` before finalizing a replacement")
	}
	if err := os.MkdirAll(surveyDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create survey directory: %w", err)
	}

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

	surveyFiles, preservedWorkerArtifacts, err := writeSurveyArtifacts(root, surveyDir, facts, dispatches, queenSurveyorSpecs(), manifest.Snapshots)
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
	stateRecorded, err := updateSurveyState(surveyedAt, len(surveyFiles))
	if err != nil {
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
	if !stateRecorded {
		result["state_note"] = surveyWithoutColonyNote
		result["next"] = "aether init"
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

func runTransactionalColonizeFinalize(root string, manifest codexColonizeManifest, results []codexSurveyorDispatch, facts codexWorkspaceFacts, now time.Time) (map[string]interface{}, error) {
	if err := validateTransactionalColonizeManifest(root, manifest); err != nil {
		return nil, err
	}
	if err := validateTransactionalSurveyResults(manifest, results); err != nil {
		return nil, err
	}
	dispatches, err := mergeExternalSurveyResults(manifest, results)
	if err != nil {
		return nil, err
	}
	if err := validateTransactionalSurveyOutputs(root, manifest, dispatches); err != nil {
		return nil, err
	}
	if err := validateExternalSurveySpawnEvidence(dispatches); err != nil {
		return nil, err
	}

	candidateDir := filepath.Join(root, filepath.FromSlash(manifest.CandidateSurveyDir))
	surveyFiles, preservedWorkerArtifacts, err := writeSurveyArtifacts(root, candidateDir, facts, dispatches, queenSurveyorSpecs(), map[string]codexArtifactSnapshot{})
	if err != nil {
		return nil, err
	}
	if preservedWorkerArtifacts != len(requiredSurveyMarkdownFiles) {
		return nil, fmt.Errorf("transactional territory refresh accepted %d of %d required worker-authored survey artifacts", preservedWorkerArtifacts, len(requiredSurveyMarkdownFiles))
	}
	if err := writeSurveyCompatibilityJSON(candidateDir, facts); err != nil {
		return nil, err
	}
	artifacts, err := collectTerritoryPublicationArtifacts(root, candidateDir)
	if err != nil {
		return nil, err
	}
	publication, err := publishTerritorySnapshot(territoryPublicationRequest{
		Root:           root,
		GeneratedAt:    now,
		TransactionID:  manifest.TransactionID,
		BaselineDigest: manifest.BaselineDigest,
		Artifacts:      artifacts,
	})
	if err != nil {
		return nil, err
	}

	runHandle, err := beginRuntimeSpawnRun("colonize", now)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize colonize run: %w", err)
	}
	runStatus := "failed"
	defer func() {
		finishRuntimeSpawnRun(runHandle, runStatus, time.Now().UTC())
	}()
	if err := recordExternalSurveySpawnTree(dispatches); err != nil {
		return nil, err
	}

	codegraphStats, codegraphWarning := runColonizeCodebaseGraph(root)
	emitColonizeCeremonyDispatchSequence("aether-colonize-finalize", dispatches)
	updateSessionSummary("colonize-finalize", "aether plan", fmt.Sprintf("Territory surveyed by external workers (%d documents)", len(surveyFiles)))
	runStatus = summarizeRunStatus(surveyorStatuses(dispatches)...)

	liveSurveyDir := filepath.Join(store.BasePath(), "survey")
	result := map[string]interface{}{
		"root":                  facts.Root,
		"detected_type":         facts.DetectedType,
		"languages":             facts.Languages,
		"frameworks":            facts.Frameworks,
		"domains":               facts.Domains,
		"entry_points":          facts.EntryPoints,
		"key_dirs":              facts.TopLevelDirs,
		"survey_dir":            liveSurveyDir,
		"survey_files":          surveyFiles,
		"surveyors":             surveyorDispatchMaps(dispatches),
		"dispatches":            surveyorDispatchMaps(dispatches),
		"existing_survey":       manifest.ExistingSurvey,
		"force_resurvey":        manifest.ForceResurvey,
		"territory_surveyed":    now.Format(time.RFC3339),
		"territory_freshness":   publication.Freshness,
		"territory_snapshot_id": publication.Freshness.SnapshotID,
		"transaction_id":        manifest.TransactionID,
		"lifecycle_receipt":     publication.Receipt,
		"dispatch_mode":         "external-task",
		"dispatch_contract":     manifest.DispatchContract,
		"artifact_source":       "external-task",
		"survey_warning":        "",
		"stats": map[string]interface{}{
			"files":       facts.FileCount,
			"directories": facts.DirectoryCount,
		},
		"next": "aether plan",
	}
	if !publication.StateRecorded {
		result["state_note"] = surveyWithoutColonyNote
		result["next"] = "aether init"
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
	if err := os.RemoveAll(filepath.Join(root, filepath.FromSlash(filepath.Join(".aether", "data", "territory-candidates", manifest.TransactionID)))); err != nil {
		result["candidate_cleanup_warning"] = err.Error()
	}
	return result, nil
}

func validateTerritoryPublicationMarkdown(content []byte) error {
	if err := validateSurveyMarkdown(content); err != nil {
		return err
	}
	var prose []string
	for _, line := range strings.Split(string(content), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		prose = append(prose, line)
	}
	normalized := strings.ToLower(strings.Trim(strings.Join(prose, " "), " \t\r\n.:-"))
	switch normalized {
	case "", "placeholder", "todo", "tbd", "coming soon", "not available", "synthetic":
		return fmt.Errorf("placeholder or synthetic survey content")
	default:
		return nil
	}
}

func validateTransactionalColonizeManifest(root string, manifest codexColonizeManifest) error {
	if !lifecycleTransactionIDPattern.MatchString(strings.TrimSpace(manifest.TransactionID)) {
		return fmt.Errorf("colonize_manifest transaction_id is invalid")
	}
	if !validSHA256(strings.TrimSpace(manifest.BaselineDigest)) {
		return fmt.Errorf("colonize_manifest baseline_digest must be a SHA-256 digest")
	}
	wantCandidate := filepath.ToSlash(filepath.Join(".aether", "data", "territory-candidates", manifest.TransactionID, "survey"))
	if filepath.ToSlash(filepath.Clean(manifest.CandidateSurveyDir)) != wantCandidate {
		return fmt.Errorf("colonize_manifest candidate_survey_dir does not match transaction_id")
	}
	if actual := territorySurveyBaselineDigest(root); actual != manifest.BaselineDigest {
		return fmt.Errorf("colonize_manifest territory baseline changed; request a fresh automatic refresh manifest")
	}
	return nil
}

func validateTransactionalSurveyResults(manifest codexColonizeManifest, results []codexSurveyorDispatch) error {
	plannedByTask := make(map[string]codexSurveyorDispatch, len(manifest.Dispatches))
	plannedNames := make(map[string]bool, len(manifest.Dispatches))
	for _, planned := range manifest.Dispatches {
		taskID := strings.TrimSpace(planned.TaskID)
		name := strings.TrimSpace(planned.Name)
		if taskID == "" || name == "" {
			return fmt.Errorf("colonize_manifest contains a survey dispatch without protocol identity")
		}
		if _, duplicate := plannedByTask[taskID]; duplicate {
			return fmt.Errorf("colonize_manifest contains duplicate survey task_id %q", taskID)
		}
		if plannedNames[name] {
			return fmt.Errorf("colonize_manifest contains duplicate surveyor name %q", name)
		}
		plannedByTask[taskID] = planned
		plannedNames[name] = true
	}
	if len(results) != len(plannedByTask) {
		return fmt.Errorf("transactional territory refresh received %d survey results for %d dispatches", len(results), len(plannedByTask))
	}
	seen := make(map[string]bool, len(results))
	for _, result := range results {
		taskID := strings.TrimSpace(result.TaskID)
		planned, ok := plannedByTask[taskID]
		if !ok || taskID == "" {
			return fmt.Errorf("transactional territory refresh received an unknown survey task_id %q", taskID)
		}
		if seen[taskID] {
			return fmt.Errorf("transactional territory refresh received duplicate result for %q", taskID)
		}
		seen[taskID] = true
		if strings.TrimSpace(result.Name) != strings.TrimSpace(planned.Name) {
			return fmt.Errorf("surveyor result %s does not match manifest worker identity", taskID)
		}
		if caste := strings.TrimSpace(result.Caste); caste != "" && caste != strings.TrimSpace(planned.Caste) {
			return fmt.Errorf("surveyor result %s does not match manifest caste", taskID)
		}
	}
	return nil
}

func validateTransactionalSurveyOutputs(root string, manifest codexColonizeManifest, dispatches []codexSurveyorDispatch) error {
	for _, dispatch := range dispatches {
		for _, rel := range declaredSurveyOutputPaths(dispatch) {
			if !strings.HasPrefix(rel, manifest.CandidateSurveyDir+"/") {
				return fmt.Errorf("surveyor %s output %q is outside transaction candidate directory", dispatch.Name, rel)
			}
			path := filepath.Join(root, filepath.FromSlash(rel))
			info, err := os.Lstat(path)
			if err != nil {
				return fmt.Errorf("surveyor %s declared output %q is missing: %w", dispatch.Name, rel, err)
			}
			if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
				return fmt.Errorf("surveyor %s declared output %q is not a regular file", dispatch.Name, rel)
			}
		}
	}
	return nil
}

func validateExternalSurveySpawnEvidence(dispatches []codexSurveyorDispatch) error {
	entries, err := agent.NewSpawnTree(store, "spawn-tree.txt").Parse()
	if err != nil {
		return fmt.Errorf("validate survey spawn-tree evidence: %w", err)
	}
	latest := make(map[string]agent.SpawnEntry, len(entries))
	for _, entry := range entries {
		latest[entry.AgentName] = entry
	}
	for _, dispatch := range dispatches {
		entry, ok := latest[dispatch.Name]
		if !ok || entry.Caste != dispatch.Caste || !agent.IsTerminalSpawnStatus(entry.Status) {
			return fmt.Errorf("surveyor %s lacks matching terminal spawn-tree evidence", dispatch.Name)
		}
	}
	return nil
}

func collectTerritoryPublicationArtifacts(root, sourceDir string) (map[string][]byte, error) {
	artifacts := make(map[string][]byte, len(requiredTerritoryArtifactPaths()))
	for _, rel := range requiredTerritoryArtifactPaths() {
		name := filepath.Base(filepath.FromSlash(rel))
		path := filepath.Join(sourceDir, name)
		info, err := os.Lstat(path)
		if err != nil {
			return nil, fmt.Errorf("collect territory artifact %s: %w", rel, err)
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
			return nil, fmt.Errorf("collect territory artifact %s: source is not a regular file", rel)
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("collect territory artifact %s: %w", rel, err)
		}
		artifacts[filepath.ToSlash(rel)] = body
	}
	return artifacts, nil
}

func validateCodexColonizeManifestFreshness(manifest codexColonizeManifest, now time.Time) error {
	raw := strings.TrimSpace(manifest.GeneratedAt)
	if raw == "" {
		return fmt.Errorf("colonize_manifest generated_at is required for freshness validation")
	}
	generatedAt, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return fmt.Errorf("colonize_manifest generated_at is invalid: %w", err)
	}
	generatedAt = generatedAt.UTC()
	if generatedAt.After(now.Add(colonizeFinalizeManifestFutureSkew)) {
		return fmt.Errorf("colonize_manifest generated_at %s is too far in the future", raw)
	}
	if now.Sub(generatedAt) > colonizeFinalizeManifestMaxAge {
		return fmt.Errorf("stale colonize_manifest generated_at %s exceeds max age %s; rerun `aether colonize --plan-only`", raw, colonizeFinalizeManifestMaxAge)
	}
	return nil
}

func validateCodexColonizeManifestWorkspace(manifest codexColonizeManifest, facts codexWorkspaceFacts) error {
	if strings.TrimSpace(manifest.DetectedType) != "" && manifest.DetectedType != facts.DetectedType {
		return fmt.Errorf("colonize_manifest workspace changed since generation: detected_type was %q, now %q; rerun `aether colonize --plan-only`", manifest.DetectedType, facts.DetectedType)
	}
	comparisons := []struct {
		name string
		was  []string
		now  []string
	}{
		{name: "languages", was: manifest.Languages, now: facts.Languages},
		{name: "frameworks", was: manifest.Frameworks, now: facts.Frameworks},
		{name: "entry_points", was: manifest.EntryPoints, now: facts.EntryPoints},
		{name: "key_dirs", was: manifest.KeyDirs, now: facts.TopLevelDirs},
	}
	for _, comparison := range comparisons {
		if !sameStringSet(comparison.was, comparison.now) {
			return fmt.Errorf("colonize_manifest workspace changed since generation: %s changed; rerun `aether colonize --plan-only`", comparison.name)
		}
	}
	if manifest.Stats != nil {
		if files, ok := intFromInterface(manifest.Stats["files"]); ok && files != facts.FileCount {
			return fmt.Errorf("colonize_manifest workspace changed since generation: file count was %d, now %d; rerun `aether colonize --plan-only`", files, facts.FileCount)
		}
		if dirs, ok := intFromInterface(manifest.Stats["directories"]); ok && dirs != facts.DirectoryCount {
			return fmt.Errorf("colonize_manifest workspace changed since generation: directory count was %d, now %d; rerun `aether colonize --plan-only`", dirs, facts.DirectoryCount)
		}
	}
	return nil
}

func sameStringSet(a, b []string) bool {
	a = uniqueSortedStrings(a)
	b = uniqueSortedStrings(b)
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func intFromInterface(value interface{}) (int, bool) {
	switch v := value.(type) {
	case int:
		return v, true
	case int64:
		return int(v), true
	case float64:
		return int(v), true
	case json.Number:
		i, err := v.Int64()
		return int(i), err == nil
	default:
		return 0, false
	}
}

func surveyDocsExistedInManifest(manifest codexColonizeManifest) bool {
	for _, name := range requiredSurveyMarkdownFiles {
		relPath := filepath.ToSlash(filepath.Join(".aether", "data", "survey", name))
		if snapshot, ok := manifest.Snapshots[relPath]; ok && snapshot.Existed {
			return true
		}
	}
	return false
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
		// Task IDs are unique within a manifest; deterministic display names are
		// intentionally compact and can collide. Prefer the protocol identity so
		// one surveyor's claims can never be attributed to another dispatch.
		result, ok := resultByTaskID[strings.TrimSpace(planned.TaskID)]
		if !ok {
			result, ok = resultByName[strings.TrimSpace(planned.Name)]
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

		filesCreated, err := cleanExternalSurveyClaims("files_created", result.FilesCreated, planned)
		if err != nil {
			return nil, err
		}
		filesModified, err := cleanExternalSurveyClaims("files_modified", result.FilesModified, planned)
		if err != nil {
			return nil, err
		}

		planned.Status = "completed"
		if strings.TrimSpace(result.Name) != "" {
			planned.Name = strings.TrimSpace(result.Name)
		}
		planned.Summary = strings.TrimSpace(result.Summary)
		planned.Blockers = append([]string{}, result.Blockers...)
		planned.Duration = result.Duration
		planned.FilesCreated = filesCreated
		planned.FilesModified = filesModified
		planned.Claimed = uniqueSortedStrings(append(append([]string{}, planned.FilesCreated...), planned.FilesModified...))
		if len(planned.Claimed) == 0 {
			claimed := make([]string, 0, len(planned.OutputPaths))
			for _, outputPath := range declaredSurveyOutputPaths(planned) {
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

func cleanExternalSurveyClaims(field string, claims []string, dispatch codexSurveyorDispatch) ([]string, error) {
	if len(claims) == 0 {
		return []string{}, nil
	}
	allowed := map[string]bool{}
	for _, outputPath := range declaredSurveyOutputPaths(dispatch) {
		allowed[outputPath] = true
	}

	cleaned := make([]string, 0, len(claims))
	for _, claim := range claims {
		path, err := cleanExternalSurveyClaimPath(claim)
		if err != nil {
			return nil, fmt.Errorf("surveyor %s %s %q is invalid: %w", dispatch.Name, field, claim, err)
		}
		if len(allowed) > 0 && !allowed[path] {
			return nil, fmt.Errorf("surveyor %s %s claim %q is not a declared output for this dispatch", dispatch.Name, field, path)
		}
		cleaned = append(cleaned, path)
	}
	return uniqueSortedStrings(cleaned), nil
}

func cleanExternalSurveyClaimPath(claim string) (string, error) {
	raw := strings.TrimSpace(strings.ReplaceAll(claim, "\\", "/"))
	if raw == "" {
		return "", fmt.Errorf("claim path is empty")
	}
	if filepath.IsAbs(raw) {
		return "", fmt.Errorf("claim path must be repo-relative under .aether/data/survey/")
	}
	cleaned := filepath.ToSlash(filepath.Clean(raw))
	if cleaned == "." || cleaned == ".." || strings.HasPrefix(cleaned, "../") {
		return "", fmt.Errorf("claim path must stay within the repository")
	}
	if !strings.HasPrefix(cleaned, ".aether/data/survey/") && !strings.HasPrefix(cleaned, ".aether/data/territory-candidates/") {
		return "", fmt.Errorf("claim path must be under .aether/data/survey/ or a runtime territory candidate directory")
	}
	return cleaned, nil
}

func declaredSurveyOutputPaths(dispatch codexSurveyorDispatch) []string {
	if len(dispatch.OutputPaths) > 0 {
		return uniqueSortedStrings(dispatch.OutputPaths)
	}
	return uniqueSortedStrings(surveyOutputPaths(dispatch.Outputs))
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
