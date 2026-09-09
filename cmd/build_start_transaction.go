package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
)

const (
	buildStartSchemaVersion          = 1
	buildStartBeforeCommitFaultPoint = "before_build_start_commit"
)

// buildStartVariant closes the set of production build-start shapes. A new
// caller cannot silently acquire an arbitrary subset of effects: adding a
// variant requires updating buildStartRuleFor and its target-matrix tests.
type buildStartVariant string

const (
	buildStartDirect             buildStartVariant = "direct"
	buildStartPlanOnly           buildStartVariant = "plan-only"
	buildStartQueenLed           buildStartVariant = "queen-led"
	buildStartExternalUnbound    buildStartVariant = "external-finalized-unbound"
	buildStartAutomaticCheckFix  buildStartVariant = "automatic-check-fix"
	buildStartCoherentChildRetry buildStartVariant = "coherent-child-retry"
)

type buildStartReviewerWindowEffect string

const (
	buildStartReviewerNone   buildStartReviewerWindowEffect = ""
	buildStartReviewerReopen buildStartReviewerWindowEffect = "reopen"
	buildStartReviewerClose  buildStartReviewerWindowEffect = "close"
)

// buildStartRequest is the complete authorization input. Functions and fault
// hooks deliberately live in buildStartOptions so this value has one stable
// canonical JSON hash.
type buildStartRequest struct {
	SchemaVersion   int                   `json:"schema_version"`
	Variant         buildStartVariant     `json:"variant"`
	StateSHA256     string                `json:"state_sha256"`
	PlanAuthority   planAuthorityDecision `json:"plan_authority"`
	Phase           int                   `json:"phase"`
	SelectedTasks   []string              `json:"selected_tasks,omitempty"`
	ExecutionOwner  string                `json:"execution_owner"`
	DispatchMode    string                `json:"dispatch_mode"`
	GeneratedAt     time.Time             `json:"generated_at"`
	AttemptID       string                `json:"attempt_id"`
	RunID           string                `json:"run_id"`
	ProcessID       int                   `json:"process_id"`
	HostPlatform    string                `json:"host_platform,omitempty"`
	WorkspaceSHA256 string                `json:"workspace_fingerprint"`
	Dispatches      []codexBuildDispatch  `json:"dispatches,omitempty"`
	Effects         buildStartEffects     `json:"effects"`
}

// buildStartEffects contains the data-bearing optional effects selected by a
// variant. Attempt journal and durable receipt writes are implicit mandatory
// effects; every other target is visible here and checked against a closed
// per-variant rule before any mutation.
type buildStartEffects struct {
	CheckpointPath  string                         `json:"checkpoint_path,omitempty"`
	ManifestPath    string                         `json:"manifest_path,omitempty"`
	Manifest        *codexBuildManifest            `json:"manifest,omitempty"`
	Completion      *codexExternalBuildCompletion  `json:"completion,omitempty"`
	ClaimsPath      string                         `json:"claims_path,omitempty"`
	Claims          *codexBuildClaims              `json:"claims,omitempty"`
	PromoteState    bool                           `json:"promote_state,omitempty"`
	MakeLatest      bool                           `json:"make_latest,omitempty"`
	ParentAttemptID string                         `json:"parent_attempt_id,omitempty"`
	ParentJobName   string                         `json:"parent_job_name,omitempty"`
	CheckFix        *checkFixAttemptRecord         `json:"check_fix,omitempty"`
	ReviewerWindow  buildStartReviewerWindowEffect `json:"reviewer_window,omitempty"`
	StalePaths      []string                       `json:"stale_paths,omitempty"`
}

type buildStartTargetReceipt struct {
	Path   string `json:"path"`
	Action string `json:"action"`
	SHA256 string `json:"sha256"`
}

// buildStartReceipt is itself the last declared build-start target. Its
// content hash makes replay validation independent from the mutable attempt
// and state files that later build stages legitimately advance.
type buildStartReceipt struct {
	SchemaVersion int                       `json:"schema_version"`
	ID            string                    `json:"id"`
	ContentHash   string                    `json:"content_hash"`
	Path          string                    `json:"path"`
	RequestSHA256 string                    `json:"request_sha256"`
	TransactionID string                    `json:"transaction_id"`
	Phase         int                       `json:"phase"`
	AttemptID     string                    `json:"attempt_id"`
	AttemptPath   string                    `json:"attempt_path"`
	PlanAuthority planAuthorityDecision     `json:"plan_authority"`
	GeneratedAt   string                    `json:"generated_at"`
	Targets       []buildStartTargetReceipt `json:"targets"`
}

type buildStartOptions struct {
	Fault    lifecycleTransactionFaultHook
	Rename   func(oldPath, newPath string) error
	Dispatch func(buildStartReceipt) error
}

type buildStartRule struct {
	checkpoint bool
	manifest   bool
	completion bool
	claims     bool
	promote    bool
	latest     bool
	parent     bool
	checkFix   bool
	reviewer   buildStartReviewerWindowEffect
	stale      bool
}

type buildStartTarget struct {
	path    string
	action  lifecycleTransactionAction
	content []byte
}

type buildStartPrepared struct {
	requestSHA     string
	transaction    string
	attemptPath    string
	receiptPath    string
	record         buildAttemptRecord
	pointer        *latestBuildAttemptPointer
	manifest       *codexBuildManifest
	completion     *codexExternalBuildCompletion
	completionPath string
	claims         *codexBuildClaims
	state          *colony.ColonyState
	window         *phaseDispatchWindowFile
	pending        *PendingDecisionFile
	targets        []buildStartTarget
	receipt        buildStartReceipt
}

func buildStartRuleFor(variant buildStartVariant) (buildStartRule, bool) {
	switch variant {
	case buildStartDirect:
		return buildStartRule{checkpoint: true, manifest: true, promote: true, latest: true, reviewer: buildStartReviewerClose, stale: true}, true
	case buildStartPlanOnly, buildStartQueenLed:
		return buildStartRule{manifest: true, latest: true, reviewer: buildStartReviewerReopen, stale: true}, true
	case buildStartExternalUnbound:
		return buildStartRule{checkpoint: true, completion: true, claims: true, promote: true, latest: true}, true
	case buildStartAutomaticCheckFix:
		return buildStartRule{latest: true, checkFix: true, reviewer: buildStartReviewerClose}, true
	case buildStartCoherentChildRetry:
		return buildStartRule{parent: true}, true
	default:
		return buildStartRule{}, false
	}
}

// commitBuildStart is intentionally not wired to a production caller until
// Plan 34. It proves the complete start boundary in isolation: one repository
// lock spans authority reload, pure derivation, baseline validation, commit,
// and rollback; the optional worker callback runs only after Commit returns a
// durable lifecycle receipt.
func commitBuildStart(root string, request buildStartRequest, options buildStartOptions) (buildStartReceipt, error) {
	var committed buildStartReceipt
	var replay bool
	err := withPlanningMutationSession(root, "build-start", func(session *planningMutationSession) error {
		state, err := loadSpecificationColonyStateInSession(session)
		if err != nil {
			return err
		}
		authority, err := loadBuildStartPlanAuthorityInSession(session, state, request.GeneratedAt)
		if err != nil {
			return err
		}
		requestSHA, err := jsonSHA256(request)
		if err != nil {
			return fmt.Errorf("build start: hash request: %w", err)
		}
		receiptPath := buildStartReceiptPath(request.Phase, request.AttemptID)
		if receiptPath == "" {
			return fmt.Errorf("build start: request has invalid receipt identity")
		}
		receiptBytes, receiptExists, err := session.ReadFile(lifecycleTransactionRootData, receiptPath)
		if err != nil {
			return err
		}
		if receiptExists {
			var existing buildStartReceipt
			if err := decodeLifecycleJSON(receiptBytes, &existing); err != nil {
				return fmt.Errorf("build start: decode durable receipt: %w", err)
			}
			if err := validateBuildStartReceipt(existing, receiptPath, requestSHA, request, authority); err != nil {
				return err
			}
			committed = existing
			replay = true
			return nil
		}

		prepared, err := prepareBuildStart(session.RepositoryRoot(), request, state, authority)
		if err != nil {
			return err
		}
		if prepared.requestSHA != requestSHA || prepared.receiptPath != receiptPath {
			return fmt.Errorf("build start: pure derivation identity changed")
		}
		if err := deriveBuildStartReviewerTargets(session, request, &prepared); err != nil {
			return err
		}
		if err := finishBuildStartPreparation(&prepared); err != nil {
			return err
		}
		if err := captureBuildStartTargetBaselines(session, prepared.targets); err != nil {
			return err
		}
		attemptBytes, attemptExists, err := session.ReadFile(lifecycleTransactionRootData, prepared.attemptPath)
		if err != nil {
			return err
		}
		if attemptExists {
			return fmt.Errorf("build start: attempt %s conflicts with an existing journal (%d bytes)", request.AttemptID, len(attemptBytes))
		}
		if options.Fault != nil {
			if err := options.Fault(buildStartBeforeCommitFaultPoint); err != nil {
				return err
			}
		}
		if err := validateBuildStartSessionBaselines(session); err != nil {
			return err
		}

		tx, err := beginLifecycleTransaction(lifecycleTransactionConfig{
			TransactionID: prepared.transaction,
			Command:       "build-start",
			Allowlist: lifecycleTransactionAllowlist{
				RepositoryRoot:    session.RepositoryRoot(),
				LifecycleDataRoot: session.DataRoot(),
			},
			Session: session,
			Rename:  options.Rename,
			Fault:   options.Fault,
		})
		if err != nil {
			return err
		}
		for _, target := range prepared.targets {
			switch target.action {
			case lifecycleTransactionWrite:
				if err := tx.DeclareWrite(lifecycleTransactionRootData, target.path, target.content); err != nil {
					return err
				}
			case lifecycleTransactionRemove:
				if err := tx.DeclareRemoval(lifecycleTransactionRootData, target.path); err != nil {
					return err
				}
			default:
				return fmt.Errorf("build start: unsupported target action %q", target.action)
			}
		}
		if _, err := tx.Commit(); err != nil {
			if tx.intent != nil {
				if rollbackErr := tx.Rollback(); rollbackErr != nil {
					return errors.Join(err, fmt.Errorf("build start rollback: %w", rollbackErr))
				}
			}
			return err
		}
		committed = prepared.receipt
		return nil
	})
	if err != nil {
		return buildStartReceipt{}, err
	}
	if !replay && options.Dispatch != nil {
		if err := options.Dispatch(committed); err != nil {
			return committed, fmt.Errorf("build start committed but dispatch callback failed: %w", err)
		}
	}
	return committed, nil
}

func prepareBuildStart(root string, request buildStartRequest, state colony.ColonyState, authority planAuthorityDecision) (buildStartPrepared, error) {
	var prepared buildStartPrepared
	phase, err := validateBuildStartRequest(root, request, state, authority)
	if err != nil {
		return prepared, err
	}
	requestSHA, err := jsonSHA256(request)
	if err != nil {
		return prepared, fmt.Errorf("build start: hash request: %w", err)
	}
	prepared.requestSHA = requestSHA
	prepared.transaction = "build-start-" + requestSHA[:24]
	prepared.receiptPath = buildStartReceiptPath(request.Phase, request.AttemptID)

	attemptPath, attempt, pointer, err := deriveBuildAttempt(buildAttemptDerivation{
		State: state, Phase: phase, PhaseNumber: request.Phase, StartedAt: request.GeneratedAt,
		AttemptID: request.AttemptID, RunID: request.RunID, ProcessID: request.ProcessID,
		HostPlatform: request.HostPlatform, WorkspaceSHA256: request.WorkspaceSHA256,
		SelectedTaskIDs: request.SelectedTasks, CheckpointPath: request.Effects.CheckpointPath,
		ManifestPath: request.Effects.ManifestPath, ClaimsPath: request.Effects.ClaimsPath,
		ExecutionOwner: request.ExecutionOwner, Dispatches: request.Dispatches,
		MakeLatest: request.Effects.MakeLatest, ParentAttemptID: request.Effects.ParentAttemptID,
		ParentJobName: request.Effects.ParentJobName, CheckFix: request.Effects.CheckFix,
		InitialDispatchMode: request.DispatchMode,
	})
	if err != nil {
		return prepared, err
	}
	prepared.attemptPath, prepared.record, prepared.pointer = attemptPath, attempt, pointer

	if request.Effects.Manifest != nil {
		manifest, boundAttempt, err := deriveBuildStartManifest(attemptPath, attempt, *request.Effects.Manifest, request.GeneratedAt)
		if err != nil {
			return prepared, err
		}
		prepared.manifest = &manifest
		prepared.record = boundAttempt
	}
	if request.Effects.Completion != nil {
		completion := *request.Effects.Completion
		digest, err := jsonSHA256(completion)
		if err != nil {
			return prepared, fmt.Errorf("build start: hash completion: %w", err)
		}
		prepared.completion = &completion
		prepared.completionPath = durableBuildCompletionPath(request.Phase, request.AttemptID)
		prepared.record.CompletionSHA256 = digest
		prepared.record.CompletionPath = displayDataPath(prepared.completionPath)
		prepared.record.Status = buildAttemptTerminal
		prepared.record.UpdatedAt = request.GeneratedAt.UTC().Format(time.RFC3339Nano)
		prepared.record.DispatchMode = request.DispatchMode
		prepared.record.RecoveryCommand = buildFinalizeRecoveryCommand(request.Phase, prepared.record.CompletionPath)
		prepared.record.History = append(prepared.record.History, buildAttemptTransition{
			Status: buildAttemptTerminal, Timestamp: prepared.record.UpdatedAt,
			Summary: "external completion and claims bound before lifecycle projection",
		})
	}
	if request.Effects.Claims != nil {
		claims := *request.Effects.Claims
		prepared.claims = &claims
		prepared.record.Claims = &claims
	}
	if request.Effects.PromoteState {
		updated := deriveBuildStartState(state, request, phase)
		prepared.state = &updated
	}

	if request.Effects.CheckpointPath != "" {
		if err := prepared.addWrite(request.Effects.CheckpointPath, state); err != nil {
			return buildStartPrepared{}, err
		}
	}
	if err := prepared.addWrite(attemptPath, prepared.record); err != nil {
		return buildStartPrepared{}, err
	}
	if pointer != nil {
		if err := prepared.addWrite(latestBuildAttemptPointerPath(request.Phase), *pointer); err != nil {
			return buildStartPrepared{}, err
		}
	}
	if prepared.manifest != nil {
		if err := prepared.addWrite(request.Effects.ManifestPath, *prepared.manifest); err != nil {
			return buildStartPrepared{}, err
		}
	}
	if prepared.completion != nil {
		if err := prepared.addWrite(prepared.completionPath, map[string]any{"result": *prepared.completion}); err != nil {
			return buildStartPrepared{}, err
		}
	}
	if prepared.claims != nil {
		if err := prepared.addWrite(request.Effects.ClaimsPath, *prepared.claims); err != nil {
			return buildStartPrepared{}, err
		}
	}
	if prepared.state != nil {
		if err := prepared.addWrite("COLONY_STATE.json", *prepared.state); err != nil {
			return buildStartPrepared{}, err
		}
	}
	for _, stalePath := range request.Effects.StalePaths {
		if err := prepared.addRemoval(stalePath); err != nil {
			return buildStartPrepared{}, err
		}
	}

	prepared.receipt = buildStartReceipt{
		SchemaVersion: buildStartSchemaVersion,
		ID:            "build-start-receipt-" + requestSHA[:24],
		Path:          prepared.receiptPath,
		RequestSHA256: requestSHA,
		TransactionID: prepared.transaction,
		Phase:         request.Phase,
		AttemptID:     request.AttemptID,
		AttemptPath:   attemptPath,
		PlanAuthority: authority,
		GeneratedAt:   request.GeneratedAt.UTC().Format(time.RFC3339Nano),
	}
	return prepared, nil
}

func (prepared *buildStartPrepared) addWrite(path string, value any) error {
	content, err := marshalBuildStartJSON(value)
	if err != nil {
		return fmt.Errorf("build start: encode %s: %w", path, err)
	}
	return prepared.addTarget(buildStartTarget{path: path, action: lifecycleTransactionWrite, content: content})
}

func (prepared *buildStartPrepared) addRemoval(path string) error {
	return prepared.addTarget(buildStartTarget{path: path, action: lifecycleTransactionRemove})
}

func (prepared *buildStartPrepared) addTarget(target buildStartTarget) error {
	if err := validateBuildStartDataPath(target.path); err != nil {
		return err
	}
	for _, existing := range prepared.targets {
		if existing.path == target.path {
			return fmt.Errorf("build start: duplicate target %q", target.path)
		}
	}
	prepared.targets = append(prepared.targets, target)
	return nil
}

func finishBuildStartPreparation(prepared *buildStartPrepared) error {
	prepared.receipt.Targets = make([]buildStartTargetReceipt, 0, len(prepared.targets))
	for _, target := range prepared.targets {
		digest := lifecycleTransactionMissingDigest
		if target.action == lifecycleTransactionWrite {
			digest = lifecycleDigest(target.content)
		}
		prepared.receipt.Targets = append(prepared.receipt.Targets, buildStartTargetReceipt{
			Path: target.path, Action: string(target.action), SHA256: digest,
		})
	}
	payload := prepared.receipt
	payload.ContentHash = ""
	hash, err := jsonSHA256(payload)
	if err != nil {
		return fmt.Errorf("build start: hash receipt: %w", err)
	}
	prepared.receipt.ContentHash = hash
	return prepared.addWrite(prepared.receiptPath, prepared.receipt)
}

func marshalBuildStartJSON(value any) ([]byte, error) {
	content, err := jsonMarshalIndent(value)
	if err != nil {
		return nil, err
	}
	return append(content, '\n'), nil
}

// jsonMarshalIndent is kept as a named seam so all target bytes use the same
// formatting without ever calling storage.Store.SaveJSON from this path.
func jsonMarshalIndent(value any) ([]byte, error) {
	return json.MarshalIndent(value, "", "  ")
}

func validateBuildStartRequest(root string, request buildStartRequest, state colony.ColonyState, authority planAuthorityDecision) (colony.Phase, error) {
	if request.SchemaVersion != buildStartSchemaVersion {
		return colony.Phase{}, fmt.Errorf("build start: schema_version must be %d", buildStartSchemaVersion)
	}
	rule, ok := buildStartRuleFor(request.Variant)
	if !ok {
		return colony.Phase{}, fmt.Errorf("build start: unsupported variant %q", request.Variant)
	}
	if request.Phase < 1 {
		return colony.Phase{}, fmt.Errorf("build start: phase must be positive")
	}
	if request.GeneratedAt.IsZero() || request.GeneratedAt.Location() != time.UTC {
		return colony.Phase{}, fmt.Errorf("build start: generated_at must be an explicit UTC time")
	}
	if request.AttemptID != deriveBuildAttemptID(request.GeneratedAt, request.ProcessID) {
		return colony.Phase{}, fmt.Errorf("build start: attempt identity does not match generated_at and process_id")
	}
	if strings.TrimSpace(request.ExecutionOwner) == "" || strings.TrimSpace(request.DispatchMode) == "" {
		return colony.Phase{}, fmt.Errorf("build start: execution owner and mode are required")
	}
	if !reflect.DeepEqual(request.SelectedTasks, uniqueSortedStrings(request.SelectedTasks)) {
		return colony.Phase{}, fmt.Errorf("build start: selected tasks must be unique and sorted")
	}
	if !reflect.DeepEqual(request.PlanAuthority, authority) {
		return colony.Phase{}, fmt.Errorf("build start: accepted plan authority is stale or conflicting")
	}
	stateSHA, err := jsonSHA256(state)
	if err != nil {
		return colony.Phase{}, err
	}
	if request.StateSHA256 != stateSHA {
		return colony.Phase{}, fmt.Errorf("build start: state baseline is stale")
	}
	workspaceSHA, err := codex.WorkspaceFingerprint(root)
	if err != nil {
		return colony.Phase{}, fmt.Errorf("build start: fingerprint workspace: %w", err)
	}
	if request.WorkspaceSHA256 != workspaceSHA {
		return colony.Phase{}, fmt.Errorf("build start: workspace fingerprint is stale")
	}
	phase, ok := buildStartPhase(state, request.Phase)
	if !ok {
		return colony.Phase{}, fmt.Errorf("build start: phase %d is not present in accepted plan", request.Phase)
	}
	if phase.Status == colony.PhaseCompleted {
		return colony.Phase{}, fmt.Errorf("build start: phase %d is already complete", request.Phase)
	}
	if err := validateSelectedBuildTasks(phase, request.SelectedTasks); err != nil {
		return colony.Phase{}, err
	}
	if err := validateBuildStartDispatches(request, phase); err != nil {
		return colony.Phase{}, err
	}
	if err := validateBuildStartEffects(request, rule); err != nil {
		return colony.Phase{}, err
	}
	if request.Effects.Manifest != nil {
		if err := validateBuildStartManifest(root, request, state, *request.Effects.Manifest); err != nil {
			return colony.Phase{}, err
		}
	}
	if request.Effects.Completion != nil {
		manifest := request.Effects.Completion.activeManifest()
		if manifest == nil {
			return colony.Phase{}, fmt.Errorf("build start: completion requires a dispatch manifest")
		}
		if err := validateBuildStartManifest(root, request, state, *manifest); err != nil {
			return colony.Phase{}, fmt.Errorf("build start: completion manifest: %w", err)
		}
	}
	return phase, nil
}

func validateBuildStartEffects(request buildStartRequest, rule buildStartRule) error {
	effects := request.Effects
	hasCheckpoint := effects.CheckpointPath != ""
	hasManifest := effects.Manifest != nil || effects.ManifestPath != ""
	hasCompletion := effects.Completion != nil
	hasClaims := effects.Claims != nil || effects.ClaimsPath != ""
	hasParent := effects.ParentAttemptID != "" || effects.ParentJobName != ""
	if hasCheckpoint != rule.checkpoint || hasManifest != rule.manifest || hasCompletion != rule.completion ||
		hasClaims != rule.claims || effects.PromoteState != rule.promote || effects.MakeLatest != rule.latest ||
		hasParent != rule.parent || (effects.CheckFix != nil) != rule.checkFix || effects.ReviewerWindow != rule.reviewer {
		return fmt.Errorf("build start: %s effects do not match its closed target rule", request.Variant)
	}
	if hasCheckpoint {
		want := filepath.ToSlash(filepath.Join("checkpoints", fmt.Sprintf("pre-build-phase-%d.json", request.Phase)))
		if effects.CheckpointPath != want {
			return fmt.Errorf("build start: checkpoint path %q must be %q", effects.CheckpointPath, want)
		}
	}
	if rule.manifest {
		want := filepath.ToSlash(filepath.Join("build", fmt.Sprintf("phase-%d", request.Phase), "manifest.json"))
		if effects.Manifest == nil || effects.ManifestPath != want {
			return fmt.Errorf("build start: manifest target must be %q", want)
		}
	}
	if rule.claims && (effects.Claims == nil || effects.ClaimsPath != "last-build-claims.json") {
		return fmt.Errorf("build start: external unbound start requires canonical claims target")
	}
	if rule.parent && strings.TrimSpace(effects.ParentAttemptID) == "" {
		return fmt.Errorf("build start: child retry requires parent attempt id")
	}
	if !rule.stale && len(effects.StalePaths) > 0 {
		return fmt.Errorf("build start: %s does not permit stale cleanup", request.Variant)
	}
	if len(effects.StalePaths) > 0 && !reflect.DeepEqual(effects.StalePaths, uniqueSortedStrings(effects.StalePaths)) {
		return fmt.Errorf("build start: stale paths must be unique and sorted")
	}
	for _, path := range effects.StalePaths {
		if !allowedBuildStartStalePath(request.Phase, path) {
			return fmt.Errorf("build start: stale target %q is outside the closed cleanup set", path)
		}
	}
	return nil
}

func validateBuildStartManifest(root string, request buildStartRequest, state colony.ColonyState, manifest codexBuildManifest) error {
	if manifest.Phase != request.Phase || filepath.Clean(manifest.Root) != filepath.Clean(root) {
		return fmt.Errorf("manifest phase/root does not match request")
	}
	if manifest.DispatchMode != request.DispatchMode || manifest.ExecutionOwner != request.ExecutionOwner {
		return fmt.Errorf("manifest execution owner/mode does not match request")
	}
	generatedAt, err := time.Parse(time.RFC3339, manifest.GeneratedAt)
	if err != nil || !generatedAt.Equal(request.GeneratedAt.Truncate(time.Second)) {
		return fmt.Errorf("manifest generated_at does not match request")
	}
	if !reflect.DeepEqual(manifest.PlanAuthority, request.PlanAuthority) {
		return fmt.Errorf("manifest plan authority does not match request")
	}
	if manifest.PlanRevisionID != request.PlanAuthority.ActiveRevision.ID {
		return fmt.Errorf("manifest active plan revision does not match request")
	}
	planSHA, err := planStateHash(state.Plan)
	if err != nil || manifest.PlanStateHash != planSHA {
		return fmt.Errorf("manifest plan state hash does not match accepted state")
	}
	if !reflect.DeepEqual(manifest.SelectedTasks, request.SelectedTasks) || !reflect.DeepEqual(manifest.Dispatches, request.Dispatches) {
		return fmt.Errorf("manifest tasks or dispatches do not match request")
	}
	if manifest.AttemptID != "" && manifest.AttemptID != request.AttemptID {
		return fmt.Errorf("manifest attempt id conflicts with request")
	}
	wantPath := displayDataPath(buildStartAttemptPath(request.Phase, request.AttemptID))
	if manifest.AttemptPath != "" && filepath.ToSlash(manifest.AttemptPath) != wantPath {
		return fmt.Errorf("manifest attempt path conflicts with request")
	}
	if manifest.ExecutionBinding != nil {
		return fmt.Errorf("manifest execution binding must be derived by build start")
	}
	return nil
}

func validateBuildStartDispatches(request buildStartRequest, phase colony.Phase) error {
	known := make(map[string]struct{}, len(phase.Tasks))
	for index, task := range phase.Tasks {
		known[buildTaskID(task, index)] = struct{}{}
	}
	selected := make(map[string]struct{}, len(request.SelectedTasks))
	for _, id := range request.SelectedTasks {
		selected[id] = struct{}{}
	}
	covered := make(map[string]struct{})
	for _, dispatch := range request.Dispatches {
		if request.Variant == buildStartAutomaticCheckFix {
			if strings.TrimSpace(dispatch.Caste) != "builder" {
				return fmt.Errorf("build start: automatic check-fix dispatch must be a builder")
			}
			continue
		}
		for _, id := range dispatchCoveredTaskIDs(dispatch) {
			if _, ok := known[id]; !ok {
				return fmt.Errorf("build start: dispatch references stale task %q", id)
			}
			if len(selected) > 0 {
				if _, ok := selected[id]; !ok {
					return fmt.Errorf("build start: dispatch task %q is outside selected tasks", id)
				}
			}
			covered[id] = struct{}{}
		}
	}
	for id := range selected {
		if _, ok := covered[id]; !ok && request.Variant != buildStartCoherentChildRetry && request.Variant != buildStartAutomaticCheckFix {
			return fmt.Errorf("build start: selected task %q has no dispatch", id)
		}
	}
	return nil
}

func buildStartPhase(state colony.ColonyState, phaseID int) (colony.Phase, bool) {
	for _, phase := range state.Plan.Phases {
		if phase.ID == phaseID {
			return phase, true
		}
	}
	return colony.Phase{}, false
}

func allowedBuildStartStalePath(phase int, target string) bool {
	if validateBuildStartDataPath(target) != nil {
		return false
	}
	base := filepath.ToSlash(filepath.Join("build", fmt.Sprintf("phase-%d", phase)))
	for _, exact := range []string{"verification.json", "gates.json", "continue.json", "review.json"} {
		if target == base+"/"+exact {
			return true
		}
	}
	for _, directory := range []string{"worker-reports", "worker-briefs"} {
		prefix := base + "/" + directory + "/"
		if strings.HasPrefix(target, prefix) && len(strings.TrimPrefix(target, prefix)) > 0 {
			return true
		}
	}
	return false
}

func validateBuildStartDataPath(target string) error {
	clean := filepath.Clean(target)
	if strings.TrimSpace(target) == "" || filepath.IsAbs(target) || filepath.ToSlash(clean) != target ||
		clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return fmt.Errorf("build start: target %q must be a canonical contained data path", target)
	}
	if strings.HasPrefix(target, "transactions/") || strings.Contains(target, "/"+lifecycleTransactionDirectory+"/") {
		return fmt.Errorf("build start: target %q overlaps transaction evidence", target)
	}
	return nil
}

func deriveBuildStartManifest(attemptPath string, attempt buildAttemptRecord, manifest codexBuildManifest, at time.Time) (codexBuildManifest, buildAttemptRecord, error) {
	manifest.AttemptID = attempt.ID
	manifest.AttemptPath = displayDataPath(attemptPath)
	manifest.ExecutionBinding = &codex.ExecutionBinding{
		SchemaVersion: codex.ExecutionBindingSchemaVersion,
		RunID:         attempt.RunID, AttemptID: attempt.ID,
		WorkspaceFingerprint: attempt.WorkspaceSHA256,
		ExecutionOwner:       strings.TrimSpace(manifest.ExecutionOwner),
	}
	digest, err := buildManifestSHA256(manifest)
	if err != nil {
		return codexBuildManifest{}, buildAttemptRecord{}, fmt.Errorf("build start: hash manifest: %w", err)
	}
	manifest.ExecutionBinding.ManifestSHA256 = digest
	if err := manifest.ExecutionBinding.Validate(); err != nil {
		return codexBuildManifest{}, buildAttemptRecord{}, err
	}
	attempt.ManifestSHA256 = digest
	attempt.ExecutionOwner = manifest.ExecutionBinding.ExecutionOwner
	manifestCopy := manifest
	attempt.PlanManifest = &manifestCopy
	attempt.Status = buildAttemptAwaiting
	attempt.UpdatedAt = at.UTC().Format(time.RFC3339Nano)
	attempt.Dispatches = append([]codexBuildDispatch(nil), manifest.Dispatches...)
	attempt.DispatchMode = strings.TrimSpace(manifest.DispatchMode)
	attempt.History = append(attempt.History, buildAttemptTransition{
		Status: buildAttemptAwaiting, Timestamp: attempt.UpdatedAt,
		Summary: "manifest persisted before worker dispatch",
	})
	return manifest, attempt, nil
}

func deriveBuildStartState(state colony.ColonyState, request buildStartRequest, phase colony.Phase) colony.ColonyState {
	copyState, err := cloneColonyState(state)
	if err != nil {
		// validateBuildStartRequest already proved this exact state is JSON
		// encodable. Retaining the value is a defensive fallback only.
		copyState = state
	}
	copyState.State = colony.StateEXECUTING
	copyState.CurrentPhase = request.Phase
	at := request.GeneratedAt.UTC()
	copyState.BuildStartedAt = &at
	for index := range copyState.Plan.Phases {
		switch {
		case copyState.Plan.Phases[index].ID < request.Phase && copyState.Plan.Phases[index].Status != colony.PhaseCompleted:
			if phaseTasksAllCompleted(copyState.Plan.Phases[index]) {
				copyState.Plan.Phases[index].Status = colony.PhaseCompleted
			}
		case copyState.Plan.Phases[index].ID == request.Phase:
			copyState.Plan.Phases[index].Status = colony.PhaseInProgress
			applyBuildTaskStatuses(&copyState.Plan.Phases[index], request.SelectedTasks)
		case copyState.Plan.Phases[index].Status == "":
			copyState.Plan.Phases[index].Status = colony.PhasePending
		}
	}
	syncActivePlanRevisionExecutionFacts(&copyState.Plan)
	copyState.Events = append(trimmedEvents(copyState.Events),
		fmt.Sprintf("%s|phase_started|build|Phase %d: %s", at.Format(time.RFC3339), request.Phase, phase.Name),
		fmt.Sprintf("%s|build_dispatched|build|Dispatched %d workers for phase %d", at.Format(time.RFC3339), len(request.Dispatches), request.Phase),
	)
	return copyState
}

func loadBuildStartPlanAuthorityInSession(session *planningMutationSession, state colony.ColonyState, at time.Time) (planAuthorityDecision, error) {
	facts := lifecycleFactsFromStateSnapshot(state, false, at.UTC())
	facts.Root = session.RepositoryRoot()
	bindings := planAuthorityVerifiedBindings{}
	if state.Specification != nil {
		if err := validateCanonicalSpecificationState(*state.Specification); err != nil {
			bindings.SpecificationError = err.Error()
		}
	}
	active, activeOK := activePlanRevision(state.Plan)
	if activeOK && strings.TrimSpace(active.CandidateID) != "" {
		retained, retainedOK := planAuthorityCandidateByID(state.Plan.Candidates, active.CandidateID)
		if !retainedOK {
			bindings.CandidateError = "accepted candidate is not retained in state"
		} else if retained.Status != colony.PlanCandidateAccepted || retained.Acceptance == nil {
			candidate := retained
			bindings.Candidate = &candidate
		} else {
			artifact, err := loadPlanCandidateArtifactInSession(session, retained.ID)
			if err != nil {
				bindings.CandidateError = err.Error()
			} else if !reflect.DeepEqual(artifact.Candidate, retained) {
				bindings.CandidateError = "accepted candidate artifact diverges from retained state"
			} else {
				candidate := artifact.Candidate
				bindings.Candidate = &candidate
				timeline, timelineErr := verifiedPlanCandidateTimelineInSession(session, candidate)
				if timelineErr != nil {
					bindings.TimelineError = timelineErr.Error()
				} else {
					bindings.Timeline = timeline.Binding
					bindings.Cards = append([]colony.PlanningIterationCard(nil), timeline.Cards...)
					receipt, exists, receiptErr := loadPlanCandidateAcceptanceReceiptInSession(session, candidate)
					switch {
					case receiptErr != nil:
						bindings.AcceptanceError = receiptErr.Error()
					case !exists:
						bindings.AcceptanceError = "accepted plan receipt artifact is missing"
					default:
						wantTokenHash := strings.TrimPrefix(lifecycleDigest([]byte(planCandidateAcceptanceToken(candidate))), "sha256:")
						if receipt.CandidateID != candidate.ID || receipt.CandidateContentHash != candidate.ContentHash ||
							receipt.SpecificationRevisionID != candidate.SpecificationRevisionID || receipt.SpecificationRevisionHash != candidate.SpecificationRevisionHash ||
							receipt.BasePlanRevisionID != candidate.BasePlanRevisionID || receipt.BasePlanRevisionHash != candidate.BasePlanRevisionHash ||
							receipt.TimelineID != candidate.Timeline.ID || receipt.TimelineDigest != candidate.Timeline.TimelineDigest ||
							receipt.ProposalHash != candidate.ProposalHash || receipt.AcceptanceTokenHash != wantTokenHash {
							bindings.AcceptanceError = "accepted plan receipt does not bind exact canonical authority"
						} else {
							bindings.Acceptance = receipt
						}
					}
				}
			}
		}
	}
	decision, err := preflightCodexBuildPlanAuthority(facts, bindings)
	return decision, err
}

func captureBuildStartTargetBaselines(session *planningMutationSession, targets []buildStartTarget) error {
	for _, target := range targets {
		if _, _, err := session.ReadFile(lifecycleTransactionRootData, target.path); err != nil {
			return fmt.Errorf("build start: capture target baseline %s: %w", target.path, err)
		}
	}
	return nil
}

func validateBuildStartSessionBaselines(session *planningMutationSession) error {
	keys := make([]string, 0, len(session.baselines))
	for key := range session.baselines {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		parts := strings.SplitN(key, ":", 2)
		if len(parts) != 2 {
			return fmt.Errorf("build start: invalid session baseline key %q", key)
		}
		kind := lifecycleTransactionRootKind(parts[0])
		current, err := session.readCurrentFileState(kind, parts[1])
		if err != nil {
			return err
		}
		baseline := session.baselines[key]
		if current.Exists != baseline.Exists || current.Digest != baseline.Digest || current.Mode.Perm() != baseline.Mode.Perm() {
			return fmt.Errorf("build start: authority baseline changed for %s", key)
		}
	}
	return nil
}

func deriveBuildStartReviewerTargets(session *planningMutationSession, request buildStartRequest, prepared *buildStartPrepared) error {
	effect := request.Effects.ReviewerWindow
	if effect == buildStartReviewerNone {
		return nil
	}
	var window phaseDispatchWindowFile
	windowExists, err := session.LoadJSON(lifecycleTransactionRootData, phaseDispatchWindowFileName, &window)
	if err != nil {
		return fmt.Errorf("build start: load reviewer window: %w", err)
	}
	if window.Phases == nil {
		window.Phases = map[string]string{}
	}
	key := strconv.Itoa(request.Phase)
	changed := false
	switch effect {
	case buildStartReviewerReopen:
		if _, ok := window.Phases[key]; ok {
			delete(window.Phases, key)
			changed = true
		}
	case buildStartReviewerClose:
		if raw := strings.TrimSpace(window.Phases[key]); raw == "" {
			window.Phases[key] = request.GeneratedAt.UTC().Format(time.RFC3339Nano)
			changed = true
		} else if _, err := time.Parse(time.RFC3339Nano, raw); err != nil {
			return fmt.Errorf("build start: phase %d has invalid reviewer-window timestamp: %w", request.Phase, err)
		}
	default:
		return fmt.Errorf("build start: unsupported reviewer-window effect %q", effect)
	}
	if changed || !windowExists {
		windowCopy := window
		prepared.window = &windowCopy
		if err := prepared.addWrite(phaseDispatchWindowFileName, windowCopy); err != nil {
			return err
		}
	}
	if effect != buildStartReviewerClose {
		return nil
	}
	var pending PendingDecisionFile
	pendingExists, err := session.LoadJSON(lifecycleTransactionRootData, pendingDecisionsFile, &pending)
	if err != nil {
		return fmt.Errorf("build start: load pending reviewer decisions: %w", err)
	}
	if !pendingExists {
		return nil
	}
	kept := make([]PendingDecision, 0, len(pending.Decisions))
	for _, decision := range pending.Decisions {
		if !decision.Resolved && decision.Source == "forced-reviewer-waiver" && decision.Phase != nil && *decision.Phase == request.Phase {
			continue
		}
		kept = append(kept, decision)
	}
	if len(kept) != len(pending.Decisions) {
		pending.Decisions = kept
		pendingCopy := pending
		prepared.pending = &pendingCopy
		if err := prepared.addWrite(pendingDecisionsFile, pendingCopy); err != nil {
			return err
		}
	}
	return nil
}

func validateBuildStartReceipt(receipt buildStartReceipt, path, requestSHA string, request buildStartRequest, authority planAuthorityDecision) error {
	if receipt.SchemaVersion != buildStartSchemaVersion || receipt.Path != path || receipt.RequestSHA256 != requestSHA ||
		receipt.Phase != request.Phase || receipt.AttemptID != request.AttemptID || !reflect.DeepEqual(receipt.PlanAuthority, authority) {
		return fmt.Errorf("build start: durable receipt conflicts with canonical request or current authority")
	}
	if receipt.TransactionID != "build-start-"+requestSHA[:24] || receipt.ID != "build-start-receipt-"+requestSHA[:24] {
		return fmt.Errorf("build start: durable receipt has conflicting identity")
	}
	payload := receipt
	payload.ContentHash = ""
	hash, err := jsonSHA256(payload)
	if err != nil || hash != receipt.ContentHash {
		return fmt.Errorf("build start: durable receipt content hash conflicts with payload")
	}
	return nil
}

func buildStartAttemptPath(phase int, attemptID string) string {
	if phase < 1 || !validBuildAttemptID(attemptID) {
		return ""
	}
	return filepath.ToSlash(filepath.Join("build", fmt.Sprintf("phase-%d", phase), "attempts", attemptID+".json"))
}

func buildStartReceiptPath(phase int, attemptID string) string {
	if phase < 1 || !validBuildAttemptID(attemptID) {
		return ""
	}
	return filepath.ToSlash(filepath.Join("build", fmt.Sprintf("phase-%d", phase), "attempts", attemptID+".start-receipt.json"))
}
