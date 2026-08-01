package cmd

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
)

const buildAttemptSchemaVersion = 1

const (
	buildAttemptPrepared    = "prepared"
	buildAttemptAwaiting    = "awaiting_external"
	buildAttemptDispatching = "dispatching"
	buildAttemptTerminal    = "terminal"
	buildAttemptBuilt       = "built"
	buildAttemptFailed      = "failed"
	buildAttemptInterrupted = "interrupted"
)

type buildAttemptTransition struct {
	Status    string `json:"status"`
	Timestamp string `json:"timestamp"`
	Summary   string `json:"summary,omitempty"`
}

type buildAttemptRecord struct {
	SchemaVersion    int                      `json:"schema_version"`
	ID               string                   `json:"id"`
	Phase            int                      `json:"phase"`
	PhaseName        string                   `json:"phase_name,omitempty"`
	Status           string                   `json:"status"`
	StartedAt        string                   `json:"started_at"`
	UpdatedAt        string                   `json:"updated_at"`
	CompletedAt      string                   `json:"completed_at,omitempty"`
	ProcessID        int                      `json:"process_id"`
	HostPlatform     string                   `json:"host_platform,omitempty"`
	ExecutionOwner   string                   `json:"execution_owner,omitempty"`
	RunID            string                   `json:"run_id,omitempty"`
	WorkspaceSHA256  string                   `json:"workspace_fingerprint,omitempty"`
	SelectedTasks    []string                 `json:"selected_tasks,omitempty"`
	Checkpoint       string                   `json:"checkpoint"`
	Manifest         string                   `json:"manifest"`
	PlanManifest     *codexBuildManifest      `json:"plan_manifest,omitempty"`
	ClaimsPath       string                   `json:"claims_path"`
	OriginalStateSHA string                   `json:"original_state_sha256"`
	ManifestSHA256   string                   `json:"manifest_sha256,omitempty"`
	CompletionSHA256 string                   `json:"completion_sha256,omitempty"`
	CompletionPath   string                   `json:"completion_path,omitempty"`
	Dispatches       []codexBuildDispatch     `json:"dispatches"`
	WorkerRuns       []buildAttemptWorkerRun  `json:"worker_runs,omitempty"`
	Claims           *codexBuildClaims        `json:"claims,omitempty"`
	DispatchMode     string                   `json:"dispatch_mode,omitempty"`
	Error            string                   `json:"error,omitempty"`
	Recoverable      bool                     `json:"recoverable"`
	RecoveryCommand  string                   `json:"recovery_command,omitempty"`
	History          []buildAttemptTransition `json:"history"`
}

type latestBuildAttemptPointer struct {
	SchemaVersion int    `json:"schema_version"`
	AttemptID     string `json:"attempt_id"`
	Path          string `json:"path"`
	UpdatedAt     string `json:"updated_at"`
}

func beginBuildAttempt(state colony.ColonyState, phaseNum int, phase colony.Phase, startedAt time.Time, selectedTaskIDs []string, checkpointRel, manifestRel, claimsRel, executionOwner string, dispatches []codexBuildDispatch) (string, error) {
	if store == nil {
		return "", fmt.Errorf("no store initialized")
	}
	if phaseNum < 1 {
		return "", fmt.Errorf("build attempt phase must be positive")
	}
	stateDigest, err := jsonSHA256(state)
	if err != nil {
		return "", fmt.Errorf("marshal build attempt state: %w", err)
	}
	runID, err := codex.NewExecutionRunID()
	if err != nil {
		return "", err
	}
	workspaceFingerprint, err := codex.WorkspaceFingerprint(buildAttemptWorkspaceRoot())
	if err != nil {
		return "", fmt.Errorf("fingerprint build workspace: %w", err)
	}
	attemptID := fmt.Sprintf("attempt-%s-%d", startedAt.UTC().Format("20060102T150405.000000000Z"), os.Getpid())
	attemptRel := filepath.ToSlash(filepath.Join("build", fmt.Sprintf("phase-%d", phaseNum), "attempts", attemptID+".json"))
	now := startedAt.UTC().Format(time.RFC3339Nano)
	record := buildAttemptRecord{
		SchemaVersion:    buildAttemptSchemaVersion,
		ID:               attemptID,
		Phase:            phaseNum,
		PhaseName:        strings.TrimSpace(phase.Name),
		Status:           buildAttemptPrepared,
		StartedAt:        now,
		UpdatedAt:        now,
		ProcessID:        os.Getpid(),
		HostPlatform:     buildHostPlatform(),
		ExecutionOwner:   strings.TrimSpace(executionOwner),
		RunID:            runID,
		WorkspaceSHA256:  workspaceFingerprint,
		SelectedTasks:    append([]string{}, selectedTaskIDs...),
		Checkpoint:       displayDataPath(checkpointRel),
		Manifest:         displayDataPath(manifestRel),
		ClaimsPath:       displayDataPath(claimsRel),
		OriginalStateSHA: stateDigest,
		Dispatches:       append([]codexBuildDispatch{}, dispatches...),
		Recoverable:      true,
		RecoveryCommand:  buildForceRedispatchCommand(phaseNum),
		History: []buildAttemptTransition{
			{Status: buildAttemptPrepared, Timestamp: now, Summary: "checkpoint recorded before lifecycle projection"},
		},
	}
	if err := store.SaveJSON(attemptRel, record); err != nil {
		return "", fmt.Errorf("save build attempt: %w", err)
	}
	pointerRel := latestBuildAttemptPointerPath(phaseNum)
	if err := store.SaveJSON(pointerRel, latestBuildAttemptPointer{
		SchemaVersion: buildAttemptSchemaVersion,
		AttemptID:     attemptID,
		Path:          displayDataPath(attemptRel),
		UpdatedAt:     now,
	}); err != nil {
		return "", fmt.Errorf("save latest build attempt pointer: %w", err)
	}
	return attemptRel, nil
}

func transitionBuildAttempt(attemptRel, status, summary string, dispatches []codexBuildDispatch, claims *codexBuildClaims, dispatchMode string, transitionErr error) error {
	if store == nil || strings.TrimSpace(attemptRel) == "" {
		return fmt.Errorf("build attempt is not initialized")
	}
	now := time.Now().UTC()
	var record buildAttemptRecord
	if err := store.UpdateJSONAtomically(attemptRel, &record, func() error {
		if record.SchemaVersion != buildAttemptSchemaVersion || strings.TrimSpace(record.ID) == "" {
			return fmt.Errorf("invalid build attempt record")
		}
		status = strings.TrimSpace(status)
		if status == "" {
			return fmt.Errorf("build attempt transition status is required")
		}
		record.Status = status
		record.UpdatedAt = now.Format(time.RFC3339Nano)
		if len(dispatches) > 0 {
			record.Dispatches = append([]codexBuildDispatch{}, dispatches...)
		}
		if claims != nil {
			copyClaims := *claims
			record.Claims = &copyClaims
		}
		if strings.TrimSpace(dispatchMode) != "" {
			record.DispatchMode = strings.TrimSpace(dispatchMode)
		}
		if transitionErr != nil {
			record.Error = strings.TrimSpace(transitionErr.Error())
		}
		if status == buildAttemptBuilt {
			record.Recoverable = false
			record.RecoveryCommand = ""
			record.Error = ""
		} else if status == buildAttemptFailed || status == buildAttemptInterrupted {
			record.Recoverable = true
			if strings.TrimSpace(record.CompletionPath) != "" && strings.TrimSpace(record.CompletionSHA256) != "" {
				record.RecoveryCommand = buildFinalizeRecoveryCommand(record.Phase, record.CompletionPath)
			} else {
				record.RecoveryCommand = buildForceRedispatchCommand(record.Phase)
			}
		}
		if buildAttemptStatusTerminal(status) {
			record.CompletedAt = now.Format(time.RFC3339Nano)
		}
		record.History = append(record.History, buildAttemptTransition{
			Status:    status,
			Timestamp: now.Format(time.RFC3339Nano),
			Summary:   strings.TrimSpace(summary),
		})
		return nil
	}); err != nil {
		return fmt.Errorf("update build attempt: %w", err)
	}
	return nil
}

func prepareBuildAttemptManifestBinding(attemptRel string, manifest *codexBuildManifest) error {
	if manifest == nil {
		return fmt.Errorf("build manifest is required")
	}
	var record buildAttemptRecord
	if err := store.LoadJSON(attemptRel, &record); err != nil {
		return fmt.Errorf("load build attempt for manifest binding: %w", err)
	}
	if record.ID != strings.TrimSpace(manifest.AttemptID) || record.Phase != manifest.Phase {
		return fmt.Errorf("build manifest attempt identity does not match journal record")
	}
	binding := codex.ExecutionBinding{
		SchemaVersion:        codex.ExecutionBindingSchemaVersion,
		RunID:                record.RunID,
		AttemptID:            record.ID,
		WorkspaceFingerprint: record.WorkspaceSHA256,
		ExecutionOwner:       strings.TrimSpace(manifest.ExecutionOwner),
	}
	manifest.ExecutionBinding = &binding
	digest, err := buildManifestSHA256(*manifest)
	if err != nil {
		return fmt.Errorf("hash build manifest: %w", err)
	}
	manifest.ExecutionBinding.ManifestSHA256 = digest
	return manifest.ExecutionBinding.Validate()
}

func bindBuildAttemptManifest(attemptRel string, manifest codexBuildManifest) error {
	digest, err := buildManifestSHA256(manifest)
	if err != nil {
		return fmt.Errorf("hash build manifest: %w", err)
	}
	if manifest.ExecutionBinding == nil {
		return fmt.Errorf("build manifest requires execution_binding")
	}
	if err := manifest.ExecutionBinding.Validate(); err != nil {
		return err
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	var record buildAttemptRecord
	if err := store.UpdateJSONAtomically(attemptRel, &record, func() error {
		if record.ID != manifest.AttemptID || record.Phase != manifest.Phase {
			return fmt.Errorf("build manifest attempt identity does not match journal record")
		}
		if err := validateBuildExecutionBinding(record, *manifest.ExecutionBinding, digest, true); err != nil {
			return err
		}
		if record.CompletionSHA256 != "" {
			return fmt.Errorf("build attempt %s already has a bound completion packet", record.ID)
		}
		if record.Status != buildAttemptPrepared && record.Status != buildAttemptAwaiting {
			return fmt.Errorf("build attempt %s is %s and cannot publish another manifest", record.ID, record.Status)
		}
		record.ManifestSHA256 = digest
		record.ExecutionOwner = strings.TrimSpace(manifest.ExecutionBinding.ExecutionOwner)
		manifestCopy := manifest
		record.PlanManifest = &manifestCopy
		record.Status = buildAttemptAwaiting
		record.UpdatedAt = now
		record.Dispatches = append([]codexBuildDispatch{}, manifest.Dispatches...)
		record.DispatchMode = strings.TrimSpace(manifest.DispatchMode)
		record.History = append(record.History, buildAttemptTransition{
			Status:    buildAttemptAwaiting,
			Timestamp: now,
			Summary:   "plan-only manifest persisted before external worker dispatch",
		})
		return nil
	}); err != nil {
		return fmt.Errorf("bind build attempt manifest: %w", err)
	}
	return nil
}

func buildManifestSHA256(manifest codexBuildManifest) (string, error) {
	copyManifest := manifest
	if manifest.ExecutionBinding != nil {
		copyBinding := *manifest.ExecutionBinding
		copyBinding.ManifestSHA256 = ""
		copyManifest.ExecutionBinding = &copyBinding
	}
	return jsonSHA256(copyManifest)
}

func validateBuildExecutionBinding(record buildAttemptRecord, binding codex.ExecutionBinding, manifestDigest string, allowOwnerChange bool) error {
	if err := binding.Validate(); err != nil {
		return err
	}
	if binding.RunID != record.RunID || binding.AttemptID != record.ID {
		return fmt.Errorf("execution binding does not match build attempt %s", record.ID)
	}
	if binding.ManifestSHA256 != strings.TrimSpace(manifestDigest) {
		return fmt.Errorf("execution binding manifest digest does not match build attempt %s", record.ID)
	}
	if binding.WorkspaceFingerprint != record.WorkspaceSHA256 {
		return fmt.Errorf("execution binding workspace does not match build attempt %s", record.ID)
	}
	if !allowOwnerChange && binding.ExecutionOwner != strings.TrimSpace(record.ExecutionOwner) {
		return fmt.Errorf("execution binding owner does not match build attempt %s", record.ID)
	}
	return codex.ValidateExecutionWorkspace(buildAttemptWorkspaceRoot(), binding)
}

func buildAttemptWorkspaceRoot() string {
	if store == nil {
		return "."
	}
	return filepath.Dir(filepath.Dir(store.BasePath()))
}

func bindBuildAttemptCompletion(attemptRel string, completion codexExternalBuildCompletion) (string, error) {
	digest, err := jsonSHA256(completion)
	if err != nil {
		return "", fmt.Errorf("hash build completion: %w", err)
	}
	var record buildAttemptRecord
	if err := store.UpdateJSONAtomically(attemptRel, &record, func() error {
		if record.SchemaVersion != buildAttemptSchemaVersion || strings.TrimSpace(record.ID) == "" {
			return fmt.Errorf("invalid build attempt record")
		}
		if record.CompletionSHA256 != "" && record.CompletionSHA256 != digest {
			return fmt.Errorf("completion packet does not match the result already bound to attempt %s", record.ID)
		}
		record.CompletionSHA256 = digest
		record.UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)
		return nil
	}); err != nil {
		return "", fmt.Errorf("bind build attempt completion: %w", err)
	}
	return digest, nil
}

func stageBuildAttemptCompletion(attemptRel string, completion codexExternalBuildCompletion) (string, string, error) {
	manifest := completion.activeManifest()
	if manifest == nil {
		return "", "", fmt.Errorf("completion file must include dispatch_manifest")
	}
	digest, err := jsonSHA256(completion)
	if err != nil {
		return "", "", fmt.Errorf("hash build completion: %w", err)
	}
	completionRel := durableBuildCompletionPath(manifest.Phase, strings.TrimSpace(manifest.AttemptID))
	if completionRel == "" {
		return "", "", fmt.Errorf("cannot derive durable completion path from build manifest")
	}
	displayPath := displayDataPath(completionRel)
	var existing buildAttemptRecord
	if err := store.LoadJSON(attemptRel, &existing); err != nil {
		return "", "", fmt.Errorf("load build attempt before staging completion: %w", err)
	}
	if existing.ID != strings.TrimSpace(manifest.AttemptID) || existing.Phase != manifest.Phase {
		return "", "", fmt.Errorf("build completion attempt identity does not match journal record")
	}
	// D-07: staging validates exactly what finalize validates, before
	// anything is bound. A packet finalize would reject must never write the
	// durable completion file or bind a digest/path to the attempt -- this is
	// the same entrypoint build-finalize calls (cmd/codex_build_finalize.go),
	// run here before any write so staging and finalizing never disagree.
	if violations := validateCompletionPacketSemantics(buildAttemptWorkspaceRoot(), completion); len(violations) > 0 {
		return "", "", &completionContractError{Violations: violations}
	}
	if existing.CompletionSHA256 != "" && existing.CompletionSHA256 != digest {
		return "", "", fmt.Errorf("completion packet does not match the result already bound to attempt %s", existing.ID)
	}
	if existing.CompletionPath != "" && filepath.ToSlash(existing.CompletionPath) != displayPath {
		return "", "", fmt.Errorf("build attempt %s already points to another completion packet", existing.ID)
	}
	durableAbsolute := filepath.Join(store.BasePath(), filepath.FromSlash(completionRel))
	_, durableAlreadyExists := os.Stat(durableAbsolute)
	payload, err := json.MarshalIndent(map[string]interface{}{"result": completion}, "", "  ")
	if err != nil {
		return "", "", fmt.Errorf("encode durable build completion: %w", err)
	}
	payload = append(payload, '\n')
	if err := store.AtomicWrite(completionRel, payload); err != nil {
		return "", "", fmt.Errorf("persist durable build completion: %w", err)
	}

	now := time.Now().UTC().Format(time.RFC3339Nano)
	var record buildAttemptRecord
	if err := store.UpdateJSONAtomically(attemptRel, &record, func() error {
		if record.ID != strings.TrimSpace(manifest.AttemptID) || record.Phase != manifest.Phase {
			return fmt.Errorf("build completion attempt identity does not match journal record")
		}
		if record.CompletionSHA256 != "" && record.CompletionSHA256 != digest {
			return fmt.Errorf("completion packet does not match the result already bound to attempt %s", record.ID)
		}
		if record.CompletionPath != "" && filepath.ToSlash(record.CompletionPath) != displayPath {
			return fmt.Errorf("build attempt %s already points to another completion packet", record.ID)
		}
		record.CompletionSHA256 = digest
		record.CompletionPath = displayPath
		record.UpdatedAt = now
		record.Recoverable = true
		record.RecoveryCommand = buildFinalizeRecoveryCommand(record.Phase, displayPath)
		if len(record.History) == 0 || record.History[len(record.History)-1].Summary != "accepted external completion packet durably staged" {
			record.History = append(record.History, buildAttemptTransition{
				Status:    record.Status,
				Timestamp: now,
				Summary:   "accepted external completion packet durably staged",
			})
		}
		return nil
	}); err != nil {
		if os.IsNotExist(durableAlreadyExists) {
			_ = os.Remove(durableAbsolute)
		}
		return "", "", fmt.Errorf("stage build completion: %w", err)
	}
	return displayPath, digest, nil
}

func durableBuildCompletionPath(phase int, attemptID string) string {
	attemptID = strings.TrimSpace(attemptID)
	if phase < 1 || !validBuildAttemptID(attemptID) {
		return ""
	}
	return filepath.ToSlash(filepath.Join("build", fmt.Sprintf("phase-%d", phase), "attempts", attemptID+".completion.json"))
}

func validBuildAttemptID(attemptID string) bool {
	attemptID = strings.TrimSpace(attemptID)
	return strings.HasPrefix(attemptID, "attempt-") && filepath.Base(attemptID) == attemptID && !strings.ContainsAny(attemptID, `/\\`)
}

func buildFinalizeRecoveryCommand(phase int, completionPath string) string {
	return fmt.Sprintf("aether build-finalize %d --completion-file %s", phase, strings.TrimSpace(completionPath))
}

func jsonSHA256(value interface{}) (string, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	var normalized interface{}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err := decoder.Decode(&normalized); err != nil {
		return "", err
	}
	canonical, err := json.Marshal(normalized)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", sha256.Sum256(canonical)), nil
}

type buildAttemptManifestBinding struct {
	Bound  bool
	Path   string
	Record buildAttemptRecord
	Legacy bool
}

func validateBuildAttemptManifestBinding(manifest codexBuildManifest, state colony.ColonyState) (buildAttemptManifestBinding, error) {
	attemptID := strings.TrimSpace(manifest.AttemptID)
	attemptPath := filepath.ToSlash(strings.TrimSpace(manifest.AttemptPath))
	if attemptID == "" && attemptPath == "" {
		return buildAttemptManifestBinding{Legacy: true}, nil
	}
	if attemptID == "" || attemptPath == "" {
		return buildAttemptManifestBinding{}, fmt.Errorf("dispatch_manifest must include both attempt_id and attempt_path")
	}
	if !validBuildAttemptID(attemptID) {
		return buildAttemptManifestBinding{}, fmt.Errorf("dispatch_manifest attempt_id %q is invalid", attemptID)
	}
	expectedRel := filepath.ToSlash(filepath.Join("build", fmt.Sprintf("phase-%d", manifest.Phase), "attempts", attemptID+".json"))
	if attemptPath != displayDataPath(expectedRel) {
		return buildAttemptManifestBinding{}, fmt.Errorf("dispatch_manifest attempt_path %q does not match attempt_id %q", attemptPath, attemptID)
	}
	latestRel, latest, ok := loadLatestBuildAttempt(manifest.Phase)
	if !ok {
		return buildAttemptManifestBinding{}, fmt.Errorf("build attempt %s is missing; rerun `aether build %d --plan-only`", attemptID, manifest.Phase)
	}
	if latest.ID != attemptID || latestRel != expectedRel {
		return buildAttemptManifestBinding{}, fmt.Errorf("build attempt %s was superseded by %s; discard the stale completion packet", attemptID, latest.ID)
	}
	manifestDigest, err := buildManifestSHA256(manifest)
	if err != nil {
		return buildAttemptManifestBinding{}, fmt.Errorf("hash dispatch_manifest: %w", err)
	}
	if latest.ManifestSHA256 == "" || latest.ManifestSHA256 != manifestDigest {
		return buildAttemptManifestBinding{}, fmt.Errorf("dispatch_manifest content does not match durable build attempt %s (got %.12s, expected %.12s)", attemptID, manifestDigest, latest.ManifestSHA256)
	}
	if manifest.ExecutionBinding == nil {
		return buildAttemptManifestBinding{}, fmt.Errorf("dispatch_manifest is missing execution_binding")
	}
	if err := validateBuildExecutionBinding(latest, *manifest.ExecutionBinding, manifestDigest, false); err != nil {
		return buildAttemptManifestBinding{}, err
	}
	generatedAt, err := time.Parse(time.RFC3339, manifest.GeneratedAt)
	if err != nil {
		return buildAttemptManifestBinding{}, fmt.Errorf("dispatch_manifest generated_at is invalid: %w", err)
	}
	recordStartedAt, err := time.Parse(time.RFC3339Nano, latest.StartedAt)
	if err != nil || !recordStartedAt.Truncate(time.Second).Equal(generatedAt) {
		return buildAttemptManifestBinding{}, fmt.Errorf("dispatch_manifest timestamp does not match durable build attempt %s", attemptID)
	}
	projectedBuiltState := latest.CompletionSHA256 != "" && state.State == colony.StateBUILT && state.CurrentPhase == manifest.Phase
	if latest.Status != buildAttemptBuilt && !projectedBuiltState {
		stateDigest, err := jsonSHA256(state)
		if err != nil {
			return buildAttemptManifestBinding{}, fmt.Errorf("hash current colony state: %w", err)
		}
		if latest.OriginalStateSHA == "" || latest.OriginalStateSHA != stateDigest {
			return buildAttemptManifestBinding{}, fmt.Errorf("colony state changed after build attempt %s was prepared; discard the stale completion packet", attemptID)
		}
	}
	switch latest.Status {
	case buildAttemptAwaiting, buildAttemptDispatching, buildAttemptTerminal, buildAttemptBuilt:
	case buildAttemptFailed, buildAttemptInterrupted:
		if latest.CompletionSHA256 == "" {
			return buildAttemptManifestBinding{}, fmt.Errorf("build attempt %s ended before terminal external results were recorded; redispatch the phase", attemptID)
		}
	default:
		return buildAttemptManifestBinding{}, fmt.Errorf("build attempt %s is %s and cannot be finalized", attemptID, latest.Status)
	}
	return buildAttemptManifestBinding{Bound: true, Path: expectedRel, Record: latest}, nil
}

func recordBuildAttemptTerminal(root, attemptRel string, phaseNum int, startedAt time.Time, dispatches []codexBuildDispatch, summary *codex.ClaimsSummary, dispatchMode string, dispatchErr error) (*codexBuildClaims, error) {
	claims := buildClaimsFromSummary(root, phaseNum, startedAt, summary)
	status := buildAttemptTerminal
	message := "all worker results recorded before lifecycle projection"
	if dispatchErr != nil {
		status = buildAttemptFailed
		message = "worker dispatch ended without a successful terminal result set"
	}
	if err := transitionBuildAttempt(attemptRel, status, message, dispatches, &claims, dispatchMode, dispatchErr); err != nil {
		return nil, err
	}
	return &claims, nil
}

func interruptLatestBuildAttempt(phaseNum int, summary string) error {
	attemptRel, record, ok := loadLatestBuildAttempt(phaseNum)
	if !ok || !buildAttemptStatusActive(record.Status) {
		return nil
	}
	if strings.TrimSpace(record.RunID) != "" {
		cleanup, err := codex.GlobalProcessTracker().KillRun(buildAttemptWorkspaceRoot(), record.RunID)
		if err != nil {
			return fmt.Errorf("cancel provider processes for build run %s: %w", record.RunID, err)
		}
		if len(cleanup.Failures) > 0 {
			return fmt.Errorf("cannot supersede build run %s because provider cancellation failed: %s", record.RunID, strings.Join(cleanup.Failures, "; "))
		}
		if err := cancelBuildAttemptWorkerRuns(attemptRel, "build run superseded before terminal result"); err != nil {
			return err
		}
	}
	return transitionBuildAttempt(attemptRel, buildAttemptInterrupted, summary, nil, nil, "", fmt.Errorf("%s", strings.TrimSpace(summary)))
}

func loadLatestBuildAttempt(phaseNum int) (string, buildAttemptRecord, bool) {
	if store == nil || phaseNum < 1 {
		return "", buildAttemptRecord{}, false
	}
	var pointer latestBuildAttemptPointer
	if err := store.LoadJSON(latestBuildAttemptPointerPath(phaseNum), &pointer); err != nil {
		return "", buildAttemptRecord{}, false
	}
	attemptRel := strings.TrimPrefix(filepath.ToSlash(strings.TrimSpace(pointer.Path)), ".aether/data/")
	if attemptRel == "" {
		return "", buildAttemptRecord{}, false
	}
	var record buildAttemptRecord
	if err := store.LoadJSON(attemptRel, &record); err != nil || record.ID != pointer.AttemptID || record.Phase != phaseNum {
		return "", buildAttemptRecord{}, false
	}
	return attemptRel, record, true
}

func loadRelevantBuildAttempt(state colony.ColonyState) (string, buildAttemptRecord, bool) {
	phaseIDs := make([]int, 0, len(state.Plan.Phases)+1)
	seen := map[int]bool{}
	if state.CurrentPhase > 0 {
		phaseIDs = append(phaseIDs, state.CurrentPhase)
		seen[state.CurrentPhase] = true
	}
	for i := len(state.Plan.Phases) - 1; i >= 0; i-- {
		phaseID := state.Plan.Phases[i].ID
		if phaseID > 0 && !seen[phaseID] {
			phaseIDs = append(phaseIDs, phaseID)
			seen[phaseID] = true
		}
	}
	var selectedRel string
	var selected buildAttemptRecord
	found := false
	for _, phaseID := range phaseIDs {
		rel, record, ok := loadLatestBuildAttempt(phaseID)
		if !ok {
			continue
		}
		if !found || record.UpdatedAt > selected.UpdatedAt {
			selectedRel = rel
			selected = record
			found = true
		}
	}
	return selectedRel, selected, found
}

func latestBuildAttemptPointerPath(phaseNum int) string {
	return filepath.ToSlash(filepath.Join("build", fmt.Sprintf("phase-%d", phaseNum), "latest-attempt.json"))
}

func buildAttemptStatusActive(status string) bool {
	switch strings.TrimSpace(status) {
	case buildAttemptPrepared, buildAttemptAwaiting, buildAttemptDispatching, buildAttemptTerminal:
		return true
	default:
		return false
	}
}

func buildAttemptStatusTerminal(status string) bool {
	switch strings.TrimSpace(status) {
	case buildAttemptBuilt, buildAttemptFailed, buildAttemptInterrupted:
		return true
	default:
		return false
	}
}

func buildAttemptSummary(record buildAttemptRecord) map[string]interface{} {
	workerStatusCounts := map[string]int{}
	for _, workerRun := range record.WorkerRuns {
		workerStatusCounts[workerRun.Status]++
	}
	activeProviderProcesses := 0
	if strings.TrimSpace(record.RunID) != "" {
		if processes, err := codex.GlobalProcessTracker().ProcessesForRun(buildAttemptWorkspaceRoot(), record.RunID); err == nil {
			activeProviderProcesses = len(processes)
		}
	}
	result := map[string]interface{}{
		"id":                        record.ID,
		"phase":                     record.Phase,
		"status":                    record.Status,
		"started_at":                record.StartedAt,
		"updated_at":                record.UpdatedAt,
		"process_id":                record.ProcessID,
		"execution_owner":           record.ExecutionOwner,
		"run_id":                    record.RunID,
		"dispatch_mode":             record.DispatchMode,
		"worker_count":              len(record.Dispatches),
		"worker_runs":               len(record.WorkerRuns),
		"worker_statuses":           workerStatusCounts,
		"active_provider_processes": activeProviderProcesses,
		"recoverable":               record.Recoverable,
	}
	if record.ManifestSHA256 != "" {
		result["manifest_sha256"] = record.ManifestSHA256
	}
	if record.WorkspaceSHA256 != "" {
		result["workspace_fingerprint"] = record.WorkspaceSHA256
	}
	if record.CompletedAt != "" {
		result["completed_at"] = record.CompletedAt
	}
	if record.Error != "" {
		result["error"] = record.Error
	}
	if record.RecoveryCommand != "" {
		result["recovery_command"] = record.RecoveryCommand
	}
	if record.CompletionPath != "" {
		result["completion_path"] = record.CompletionPath
		result["completion_staged"] = record.CompletionSHA256 != ""
	}
	return result
}

func buildAttemptProcessAlive(record buildAttemptRecord) bool {
	if !buildAttemptStatusActive(record.Status) {
		return false
	}
	if processAlive(record.ProcessID) {
		return true
	}
	if strings.TrimSpace(record.RunID) == "" {
		return false
	}
	processes, err := codex.GlobalProcessTracker().ProcessesForRun(buildAttemptWorkspaceRoot(), record.RunID)
	return err == nil && len(processes) > 0
}

func buildClaimsFromSummary(root string, phaseNum int, startedAt time.Time, summary *codex.ClaimsSummary) codexBuildClaims {
	claims := codexBuildClaims{BuildPhase: phaseNum, Timestamp: startedAt.Format(time.RFC3339)}
	if summary != nil {
		claims.FilesCreated = append([]string{}, summary.FilesCreated...)
		claims.FilesModified = append([]string{}, summary.FilesModified...)
		claims.TestsWritten = append([]string{}, summary.TestsWritten...)
		if len(summary.TaskClaims) > 0 {
			claims.TaskClaims = make([]codexBuildTaskClaim, 0, len(summary.TaskClaims))
			for _, taskClaim := range summary.TaskClaims {
				claims.TaskClaims = append(claims.TaskClaims, codexBuildTaskClaim{
					TaskID:        taskClaim.TaskID,
					FilesCreated:  append([]string{}, taskClaim.FilesCreated...),
					FilesModified: append([]string{}, taskClaim.FilesModified...),
					TestsWritten:  append([]string{}, taskClaim.TestsWritten...),
				})
			}
		}
	}
	attachBuildArtifactEvidence(root, &claims)
	return claims
}
