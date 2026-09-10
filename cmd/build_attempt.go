package cmd

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
)

const buildAttemptSchemaVersion = 1

const buildAttemptPublicDataPrefix = ".aether/data/"

// canonicalBuildAttemptDataPath converts either the internal data-root-
// relative representation or the public repository-relative representation
// into the one internal form. Persisted paths are untrusted input on replay:
// reject ambiguity instead of cleaning traversal, duplicate prefixes, or
// platform-specific absolute paths into a different target.
func canonicalBuildAttemptDataPath(value string) (string, error) {
	if value == "" {
		return "", nil
	}
	if value != strings.TrimSpace(value) || strings.Contains(value, `\`) {
		return "", fmt.Errorf("build attempt path %q is not canonical", value)
	}
	if filepath.IsAbs(value) || path.IsAbs(value) || filepath.VolumeName(value) != "" ||
		(len(value) >= 2 && value[1] == ':') {
		return "", fmt.Errorf("build attempt path %q must be relative to the lifecycle data root", value)
	}

	relative := value
	if relative == strings.TrimSuffix(buildAttemptPublicDataPrefix, "/") {
		return "", fmt.Errorf("build attempt path %q names the data root, not a file", value)
	}
	if strings.HasPrefix(relative, buildAttemptPublicDataPrefix) {
		relative = strings.TrimPrefix(relative, buildAttemptPublicDataPrefix)
	}
	if relative == strings.TrimSuffix(buildAttemptPublicDataPrefix, "/") ||
		strings.HasPrefix(relative, buildAttemptPublicDataPrefix) ||
		relative == ".aether" || strings.HasPrefix(relative, ".aether/") {
		return "", fmt.Errorf("build attempt path %q contains an ambiguous data-root prefix", value)
	}
	clean := path.Clean(relative)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") || clean != relative {
		return "", fmt.Errorf("build attempt path %q must be a canonical contained data path", value)
	}
	return relative, nil
}

// displayBuildAttemptDataPath is the single conversion from an accepted
// internal build-attempt path to its owner-facing repository path. It also
// accepts one already displayed prefix for compatibility with durable older
// records, normalizing it rather than adding a second prefix.
func displayBuildAttemptDataPath(value string) (string, error) {
	relative, err := canonicalBuildAttemptDataPath(value)
	if err != nil {
		return "", err
	}
	if relative == "" {
		return "", nil
	}
	return buildAttemptPublicDataPrefix + relative, nil
}

const (
	buildAttemptPrepared    = "prepared"
	buildAttemptAwaiting    = "awaiting_external"
	buildAttemptDispatching = "dispatching"
	buildAttemptTerminal    = "terminal"
	buildAttemptBuilt       = "built"
	buildAttemptFailed      = "failed"
	buildAttemptInterrupted = "interrupted"
	// buildAttemptPartial marks an attempt that ended with SOME, but not all,
	// of its covered tasks credited (D-08/D-09 receipt evidence) and D-10
	// append-only recovery already created a child attempt for the rest.
	// Deliberately distinct from buildAttemptBuilt (which means "every
	// covered task is proven done") and from buildAttemptFailed (which means
	// "nothing of this attempt was credited") -- a partial attempt is neither.
	buildAttemptPartial = "partial"
)

type buildAttemptTransition struct {
	Status    string `json:"status"`
	Timestamp string `json:"timestamp"`
	Summary   string `json:"summary,omitempty"`
}

// outOfBandVerificationRecord is the provenance a build attempt carries when
// it was closed by the operator-invoked `verify-out-of-band` ceremony
// (cmd/verify_out_of_band.go) instead of a worker dispatch: real work
// happened outside the build pipeline (a hand edit, a pair session), and
// this records that the ceremony re-verified the phase's OWN success
// criteria against CURRENT disk state, fresh, before closing -- never a
// worker's self-report. Its presence on a buildAttemptRecord is what
// distinguishes an out-of-band closure from a genuine worker-verified one
// (191.1-CONTEXT.md D-09/D-11); a record carrying this field never also
// carries a fabricated Dispatches/WorkerRuns entry -- see
// closeBuildAttemptOutOfBand's own doc comment and
// TestVerifyOutOfBandNeverSynthesizesWorkerReceipts
// (cmd/verify_out_of_band_test.go).
type outOfBandVerificationRecord struct {
	VerifiedAt string `json:"verified_at"`
	Phase      int    `json:"phase"`
	// Policy is the phase's criterion evidence policy at verification time
	// (criterionEvidencePolicyBoundV1 / criterionEvidencePolicyLegacyUnbound
	// / criterionEvidencePolicyNotRequired) -- what kind of evidence this
	// verification pass was actually able to gather.
	Policy string `json:"policy"`
	// CriteriaChecked lists every success criterion (bound or free-prose)
	// this pass evaluated.
	CriteriaChecked []string `json:"criteria_checked,omitempty"`
	// ChecksRun lists which of build/types/lint/tests actually executed
	// (skipped checks -- no command resolved -- are excluded).
	ChecksRun []string `json:"checks_run,omitempty"`
	// ArtifactsHashed lists every declared artifact path that was hash-
	// verified against disk NOW as part of this pass.
	ArtifactsHashed []string `json:"artifacts_hashed,omitempty"`
	// AcknowledgedLegacy records whether the operator supplied the separate,
	// explicit acknowledgment required for a phase with only free-prose
	// success criteria (T-191.1-03-04) -- false for a bound phase, where no
	// acknowledgment is needed or accepted.
	AcknowledgedLegacy bool   `json:"acknowledged_legacy,omitempty"`
	Summary            string `json:"summary"`
}

// buildFreeCheckReport is a report of what the program's own deterministic
// checks (build, types, lint, tests) found at build-finalize time. It is
// explicitly NOT an advancement gate -- advancement (phase status, task
// status) is `continue`'s decision alone, proven by
// TestBuildFinalizeFreeChecksDoNotAdvanceThePhase
// (cmd/phase_verified_once_test.go), which asserts a passing-checks finalize
// and a failing-checks finalize commit byte-identical phase/task status. D-08
// / ruling D11 rule 2: the program's free checks are the floor everywhere,
// but the build boundary only records them here -- it never blocks or
// advances on them itself.
type buildFreeCheckReport struct {
	RecordedAt string `json:"recorded_at"`
	Phase      int    `json:"phase"`
	// ChecksRun lists only the checks (of build/types/lint/tests) that
	// actually executed a resolved command.
	ChecksRun []string `json:"checks_run,omitempty"`
	// ChecksSkipped lists checks with no resolved command for this project.
	ChecksSkipped []string `json:"checks_skipped,omitempty"`
	// Failed lists the checks that ran and did not pass.
	Failed  []string `json:"failed,omitempty"`
	Passed  bool     `json:"passed"`
	Summary string   `json:"summary"`
}

// buildFreeCheckReportFromFloor converts a deterministicFloorResult (the
// single shared floor body from cmd/deterministic_floor.go, also used by both
// continue lanes) into the report build-finalize attaches to the attempt
// journal. It is a pure conversion -- no I/O, no decision.
func buildFreeCheckReportFromFloor(phaseNum int, recordedAt time.Time, floor deterministicFloorResult) buildFreeCheckReport {
	report := buildFreeCheckReport{
		RecordedAt: recordedAt.UTC().Format(time.RFC3339),
		Phase:      phaseNum,
		Passed:     floor.ChecksPassed,
	}
	for _, step := range floor.Steps {
		switch {
		case step.Skipped:
			report.ChecksSkipped = append(report.ChecksSkipped, step.Name)
		case !step.Passed:
			report.ChecksRun = append(report.ChecksRun, step.Name)
			report.Failed = append(report.Failed, step.Name)
		default:
			report.ChecksRun = append(report.ChecksRun, step.Name)
		}
	}
	switch {
	case len(report.Failed) > 0:
		report.Summary = fmt.Sprintf("checks failed: %s", strings.Join(report.Failed, ", "))
	case len(report.ChecksRun) == 0:
		report.Summary = "no tests to run in this project"
	default:
		report.Summary = fmt.Sprintf("checks passed: %s", strings.Join(report.ChecksRun, ", "))
	}
	return report
}

type buildAttemptRecord struct {
	SchemaVersion    int                     `json:"schema_version"`
	ID               string                  `json:"id"`
	Phase            int                     `json:"phase"`
	PhaseName        string                  `json:"phase_name,omitempty"`
	Status           string                  `json:"status"`
	StartedAt        string                  `json:"started_at"`
	UpdatedAt        string                  `json:"updated_at"`
	CompletedAt      string                  `json:"completed_at,omitempty"`
	ProcessID        int                     `json:"process_id"`
	HostPlatform     string                  `json:"host_platform,omitempty"`
	ExecutionOwner   string                  `json:"execution_owner,omitempty"`
	RunID            string                  `json:"run_id,omitempty"`
	WorkspaceSHA256  string                  `json:"workspace_fingerprint,omitempty"`
	SelectedTasks    []string                `json:"selected_tasks,omitempty"`
	Checkpoint       string                  `json:"checkpoint"`
	Manifest         string                  `json:"manifest"`
	PlanManifest     *codexBuildManifest     `json:"plan_manifest,omitempty"`
	ClaimsPath       string                  `json:"claims_path"`
	OriginalStateSHA string                  `json:"original_state_sha256"`
	ManifestSHA256   string                  `json:"manifest_sha256,omitempty"`
	CompletionSHA256 string                  `json:"completion_sha256,omitempty"`
	CompletionPath   string                  `json:"completion_path,omitempty"`
	Dispatches       []codexBuildDispatch    `json:"dispatches"`
	WorkerRuns       []buildAttemptWorkerRun `json:"worker_runs,omitempty"`
	Claims           *codexBuildClaims       `json:"claims,omitempty"`
	DispatchMode     string                  `json:"dispatch_mode,omitempty"`
	Error            string                  `json:"error,omitempty"`
	Recoverable      bool                    `json:"recoverable"`
	RecoveryCommand  string                  `json:"recovery_command,omitempty"`
	// RecoveryTaskIDs is the exact unfinished-only task set named by
	// RecoveryCommand. Partial parents and their canonical child both persist
	// it so replay never has to infer owner-facing authority from prose.
	RecoveryTaskIDs []string                 `json:"recovery_task_ids,omitempty"`
	History         []buildAttemptTransition `json:"history"`
	// OutOfBandVerification is set ONLY by closeBuildAttemptOutOfBand, never
	// by any worker dispatch or build-finalize path. Its presence marks this
	// attempt as closed by the operator-invoked verify-out-of-band ceremony
	// rather than by real worker results.
	OutOfBandVerification *outOfBandVerificationRecord `json:"out_of_band_verification,omitempty"`
	// FreeChecks is set ONLY by attachBuildFreeCheckReport, called from
	// runCodexBuildFinalize when skipVerify is false. It is a report, never
	// an advancement gate -- see buildFreeCheckReport's doc comment.
	FreeChecks *buildFreeCheckReport `json:"free_checks,omitempty"`
	// CheckFix is set ONLY on a NEW attempt record created for D-02/D-03's
	// single bounded automatic builder fix attempt (attachCheckFixAttempt,
	// called from the continue verification path in cmd/codex_continue.go).
	// Its presence is what distinguishes a check-fix attempt from an
	// ordinary phase build attempt -- see checkFixAttemptRecord's own doc
	// comment for the append-only guarantee this field depends on.
	CheckFix *checkFixAttemptRecord `json:"check_fix,omitempty"`
	// ParentAttemptID is set only while the canonical build-start transaction
	// derives a NEW D-10 unfinished-only recovery attempt -- never on the
	// parent attempt whose partial credit triggered it. Optional and
	// omitempty: existing attempt JSON written before this field existed still
	// decodes cleanly with an empty string here (mirrors CheckFix's precedent).
	ParentAttemptID string `json:"parent_attempt_id,omitempty"`
	// ParentJobName is the coherent job name (cmd/coherent_jobs.go) this
	// attempt's retry job was derived from -- the original grouped job whose
	// partial credit created this recovery attempt.
	ParentJobName string `json:"parent_job_name,omitempty"`
	// VerificationBoundary is set ONLY by attachVerificationBoundary, and by
	// nothing else -- see verificationBoundaryDecision's own doc comment
	// (cmd/verification_boundary.go) for how its value is derived. An
	// attempt record written before this field existed decodes cleanly with
	// this left nil, not a fabricated default; readers use
	// verificationBoundaryForAttempt, which reports ok=false rather than
	// inventing one.
	VerificationBoundary *verificationBoundaryDecision `json:"verification_boundary,omitempty"`
	// CreditedFiles is the union of files the two-stage receipt trust
	// boundary (admitCoherentJobTaskReceipts / finalizeCoherentJobTaskReceiptEvidence)
	// actually finalised for this attempt's completed tasks (D-08): a whole-
	// success dispatch's own reported outputs, or a grouped job's root-
	// evidenced per-task claims -- never a file merely touched. Set ONLY by
	// attachResultFilePrecision, and never overwrites Status, Dispatches, or
	// Claims. An attempt record written before this field existed decodes
	// cleanly with this left nil.
	CreditedFiles []string `json:"credited_files,omitempty"`
	// UncreditedFiles names every file this attempt touched that no admitted
	// receipt claimed for a completed task, together with where that file
	// currently lives. Set ONLY by attachResultFilePrecision. An uncredited
	// file is never removed from disk and never folded into CreditedFiles --
	// see deriveResultFilePrecision.
	UncreditedFiles []buildAttemptUncreditedFile `json:"uncredited_files,omitempty"`
	// PlanReality is CAP-022's per-task plan-versus-reality comparison: what
	// the plan declared each finalised task's artifacts to be, and whether
	// each was actually present in the repository at finalisation time. Set
	// ONLY by attachBuildPlanRealityReport, and touches nothing else on the
	// record -- mirrors attachBuildFreeCheckReport's narrow-setter
	// discipline.
	PlanReality *buildPlanRealityReport `json:"plan_reality,omitempty"`
	// KnowledgeDeltas is CAP-066's content-level decision and learning
	// deltas this attempt produced, bound to this attempt's own ID. Set
	// ONLY by attachBuildKnowledgeDeltas. Rendering
	// (cmd/lifecycle_closeout.go's lifecycleCloseoutKnowledgeDeltaEvidence)
	// is read-only: it displays these, it never writes or promotes them.
	KnowledgeDeltas []buildAttemptKnowledgeDelta `json:"knowledge_deltas,omitempty"`
}

// buildAttemptUncreditedFile names one file a build attempt touched that no
// admitted receipt claimed for any completed task (D-08), together with
// where it currently lives -- the repository root in shared-checkout mode,
// or the exact isolated workspace path and branch in worktree mode. An
// uncredited file is never deleted and never folded into CreditedFiles.
type buildAttemptUncreditedFile struct {
	Path     string `json:"path"`
	Location string `json:"location"`
}

type latestBuildAttemptPointer struct {
	SchemaVersion int    `json:"schema_version"`
	AttemptID     string `json:"attempt_id"`
	Path          string `json:"path"`
	UpdatedAt     string `json:"updated_at"`
}

// buildAttemptDerivation contains every input needed to calculate an attempt
// before any write begins. Keeping the clock, random run ID, process identity,
// platform, and workspace fingerprint outside the derivation lets canonical
// build start calculate every byte before opening its transaction.
type buildAttemptDerivation struct {
	State               colony.ColonyState
	Phase               colony.Phase
	PhaseNumber         int
	StartedAt           time.Time
	AttemptID           string
	RunID               string
	ProcessID           int
	HostPlatform        string
	WorkspaceSHA256     string
	SelectedTaskIDs     []string
	CheckpointPath      string
	ManifestPath        string
	ClaimsPath          string
	ExecutionOwner      string
	Dispatches          []codexBuildDispatch
	MakeLatest          bool
	ParentAttemptID     string
	ParentJobName       string
	CheckFix            *checkFixAttemptRecord
	InitialStatus       string
	InitialSummary      string
	InitialDispatchMode string
}

// deriveBuildAttemptID retains the historical attempt ID format without
// consulting the process or clock itself. Callers decide which explicit
// process identity and timestamp are part of their canonical request.
func deriveBuildAttemptID(startedAt time.Time, processID int) string {
	return fmt.Sprintf("attempt-%s-%d", startedAt.UTC().Format("20060102T150405.000000000Z"), processID)
}

// deriveBuildAttempt is the pure build-attempt constructor shared by the
// legacy per-file writer and the canonical build-start transaction. It does
// not read a store, inspect a workspace, generate randomness, or mutate its
// inputs.
func deriveBuildAttempt(input buildAttemptDerivation) (string, buildAttemptRecord, *latestBuildAttemptPointer, error) {
	if input.PhaseNumber < 1 {
		return "", buildAttemptRecord{}, nil, fmt.Errorf("build attempt phase must be positive")
	}
	if input.Phase.ID != 0 && input.Phase.ID != input.PhaseNumber {
		return "", buildAttemptRecord{}, nil, fmt.Errorf("build attempt phase %d does not match phase record %d", input.PhaseNumber, input.Phase.ID)
	}
	attemptID := strings.TrimSpace(input.AttemptID)
	if !validBuildAttemptID(attemptID) {
		return "", buildAttemptRecord{}, nil, fmt.Errorf("build attempt id %q is invalid", attemptID)
	}
	if !strings.HasPrefix(strings.TrimSpace(input.RunID), "run-") {
		return "", buildAttemptRecord{}, nil, fmt.Errorf("build attempt run id is invalid")
	}
	workspaceSHA := strings.TrimSpace(input.WorkspaceSHA256)
	if len(workspaceSHA) != sha256.Size*2 {
		return "", buildAttemptRecord{}, nil, fmt.Errorf("build attempt workspace fingerprint must be a SHA-256 digest")
	}
	if input.ProcessID < 1 {
		return "", buildAttemptRecord{}, nil, fmt.Errorf("build attempt process id must be positive")
	}
	checkpointPath, err := displayBuildAttemptDataPath(input.CheckpointPath)
	if err != nil {
		return "", buildAttemptRecord{}, nil, fmt.Errorf("build attempt checkpoint: %w", err)
	}
	manifestPath, err := displayBuildAttemptDataPath(input.ManifestPath)
	if err != nil {
		return "", buildAttemptRecord{}, nil, fmt.Errorf("build attempt manifest: %w", err)
	}
	claimsPath, err := displayBuildAttemptDataPath(input.ClaimsPath)
	if err != nil {
		return "", buildAttemptRecord{}, nil, fmt.Errorf("build attempt claims: %w", err)
	}
	stateDigest, err := jsonSHA256(input.State)
	if err != nil {
		return "", buildAttemptRecord{}, nil, fmt.Errorf("marshal build attempt state: %w", err)
	}
	startedAt := input.StartedAt.UTC()
	now := startedAt.Format(time.RFC3339Nano)
	status := strings.TrimSpace(input.InitialStatus)
	if status == "" {
		status = buildAttemptPrepared
	}
	summary := strings.TrimSpace(input.InitialSummary)
	if summary == "" {
		summary = "checkpoint recorded before lifecycle projection"
	}
	attemptRel := filepath.ToSlash(filepath.Join("build", fmt.Sprintf("phase-%d", input.PhaseNumber), "attempts", attemptID+".json"))
	recoveryCommand := buildForceRedispatchCommand(input.PhaseNumber)
	var recoveryTaskIDs []string
	if strings.TrimSpace(input.ParentAttemptID) != "" {
		recoveryTaskIDs = uniqueSortedStrings(input.SelectedTaskIDs)
		recoveryCommand = buildUnfinishedRetryRedispatchCommand(input.PhaseNumber, recoveryTaskIDs)
	}
	record := buildAttemptRecord{
		SchemaVersion:    buildAttemptSchemaVersion,
		ID:               attemptID,
		Phase:            input.PhaseNumber,
		PhaseName:        strings.TrimSpace(input.Phase.Name),
		Status:           status,
		StartedAt:        now,
		UpdatedAt:        now,
		ProcessID:        input.ProcessID,
		HostPlatform:     strings.TrimSpace(input.HostPlatform),
		ExecutionOwner:   strings.TrimSpace(input.ExecutionOwner),
		RunID:            strings.TrimSpace(input.RunID),
		WorkspaceSHA256:  workspaceSHA,
		SelectedTasks:    append([]string{}, input.SelectedTaskIDs...),
		Checkpoint:       checkpointPath,
		Manifest:         manifestPath,
		ClaimsPath:       claimsPath,
		OriginalStateSHA: stateDigest,
		Dispatches:       append([]codexBuildDispatch{}, input.Dispatches...),
		DispatchMode:     strings.TrimSpace(input.InitialDispatchMode),
		Recoverable:      true,
		RecoveryCommand:  recoveryCommand,
		RecoveryTaskIDs:  recoveryTaskIDs,
		ParentAttemptID:  strings.TrimSpace(input.ParentAttemptID),
		ParentJobName:    strings.TrimSpace(input.ParentJobName),
		History: []buildAttemptTransition{{
			Status: status, Timestamp: now, Summary: summary,
		}},
	}
	if input.CheckFix != nil {
		copyRecord := *input.CheckFix
		record.CheckFix = &copyRecord
	}
	var pointer *latestBuildAttemptPointer
	if input.MakeLatest {
		attemptDisplayPath, err := displayBuildAttemptDataPath(attemptRel)
		if err != nil {
			return "", buildAttemptRecord{}, nil, fmt.Errorf("build attempt journal: %w", err)
		}
		pointer = &latestBuildAttemptPointer{
			SchemaVersion: buildAttemptSchemaVersion,
			AttemptID:     attemptID,
			Path:          attemptDisplayPath,
			UpdatedAt:     now,
		}
	}
	return attemptRel, record, pointer, nil
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
			claimsPath, pathErr := displayBuildAttemptDataPath("last-build-claims.json")
			if pathErr != nil {
				return pathErr
			}
			record.ClaimsPath = claimsPath
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
			record.RecoveryTaskIDs = nil
			record.Error = ""
		} else if status == buildAttemptPartial {
			record.Recoverable = true
			record.RecoveryTaskIDs = unfinishedBuildTaskIDs(record.SelectedTasks, record.Dispatches)
			record.RecoveryCommand = buildUnfinishedRetryRedispatchCommand(record.Phase, record.RecoveryTaskIDs)
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

// closeBuildAttemptOutOfBand transitions an existing build attempt to a
// closed, sealed state (buildAttemptBuilt) carrying out-of-band verification
// provenance -- called ONLY from cmd/verify_out_of_band.go's
// closeOutOfBandCeremony, which is itself only reachable when a human types
// `aether verify-out-of-band <phase> --force`
// (TestVerifyOutOfBandHasNoLifecycleCaller,
// cmd/verify_out_of_band_reachability_test.go).
//
// T-191.1-03-02 (the honesty invariant): this closure deliberately never
// reads OR writes record.Dispatches or record.WorkerRuns anywhere in its
// body. That is not an oversight to be caught by review -- it is the whole
// mechanism the honesty ratchet
// (TestVerifyOutOfBandNeverSynthesizesWorkerReceipts) depends on: whatever
// UpdateJSONAtomically loaded from disk for those two fields is exactly what
// gets written back, unchanged, because nothing in this function's closure
// ever assigns to them. A future edit that adds such an assignment is
// exactly the fabrication path that ratchet exists to catch.
func closeBuildAttemptOutOfBand(attemptRel string, provenance outOfBandVerificationRecord) error {
	if store == nil || strings.TrimSpace(attemptRel) == "" {
		return fmt.Errorf("build attempt is not initialized")
	}
	now := time.Now().UTC()
	var record buildAttemptRecord
	if err := store.UpdateJSONAtomically(attemptRel, &record, func() error {
		if record.SchemaVersion != buildAttemptSchemaVersion || strings.TrimSpace(record.ID) == "" {
			return fmt.Errorf("invalid build attempt record")
		}
		record.Status = buildAttemptBuilt
		record.UpdatedAt = now.Format(time.RFC3339Nano)
		record.CompletedAt = now.Format(time.RFC3339Nano)
		record.Recoverable = false
		record.RecoveryCommand = ""
		record.Error = ""
		provenanceCopy := provenance
		record.OutOfBandVerification = &provenanceCopy
		record.History = append(record.History, buildAttemptTransition{
			Status:    buildAttemptBuilt,
			Timestamp: now.Format(time.RFC3339Nano),
			Summary:   "closed by verify-out-of-band: " + strings.TrimSpace(provenance.Summary),
		})
		return nil
	}); err != nil {
		return fmt.Errorf("close build attempt out-of-band: %w", err)
	}
	return nil
}

// attachBuildFreeCheckReport attaches the program's own deterministic
// build-time checks to an attempt record as a report field, and touches
// nothing else on the record: not Status, not Dispatches, not History, not
// Recoverable. D-08 / ruling D11 rule 2 require the free checks to be
// recorded, never to gate or advance a phase -- a setter this narrow makes
// that impossible to violate by accident, the same discipline
// closeBuildAttemptOutOfBand uses for OutOfBandVerification. Called only
// from runCodexBuildFinalize, and only when skipVerify is false.
func attachBuildFreeCheckReport(attemptRel string, report buildFreeCheckReport) error {
	if store == nil || strings.TrimSpace(attemptRel) == "" {
		return fmt.Errorf("build attempt is not initialized")
	}
	var record buildAttemptRecord
	return store.UpdateJSONAtomically(attemptRel, &record, func() error {
		if record.SchemaVersion != buildAttemptSchemaVersion || strings.TrimSpace(record.ID) == "" {
			return fmt.Errorf("invalid build attempt record")
		}
		reportCopy := report
		record.FreeChecks = &reportCopy
		return nil
	})
}

// checkFixAttemptRecord is the provenance a build attempt carries when it
// exists ONLY because the verification floor's single bounded automatic
// builder fix attempt created it (D-02, D-03), rather than a fresh phase
// build. It always lives on a NEW record produced by canonical build start
// and advanced through transitionBuildAttempt like any other attempt. It
// never overwrites or mutates the parent attempt's own Dispatches, Claims, or
// Status (TestFixAttemptNeverOverwritesTheFirstResult); the two exist as
// separate, independently-readable journal entries.
type checkFixAttemptRecord struct {
	// RecordedAt is when this fix attempt's outcome was recorded (after the
	// re-run floor completed), not when the attempt began.
	RecordedAt string `json:"recorded_at"`
	Phase      int    `json:"phase"`
	// Check is the name of the failing verification step this attempt was
	// sent to fix (for example "tests").
	Check string `json:"check"`
	// Reason is a plain-English clause naming why this attempt exists (D-03:
	// "fixing the failed tests check") -- distinct from the parent build
	// attempt's own summary, and never overwrites it.
	Reason string `json:"reason"`
	// ParentAttemptID is the build attempt ID the failing check belonged to
	// -- the attempt this fix attempt is following up on, not replacing.
	ParentAttemptID string `json:"parent_attempt_id,omitempty"`
	// FailureIndex is the compact index the fix builder was given -- never
	// the whole command log (D-02, cmd/check_fix_attempt.go).
	FailureIndex checkFailureIndex `json:"failure_index"`
	// Outcome is "fixed" when the re-run floor passed, or "still_failing"
	// when it did not -- checked by the check_fix_attempt gate
	// (cmd/codex_continue.go) to decide whether continue blocks.
	Outcome string `json:"outcome"`
}

// attachCheckFixAttempt attaches D-02/D-03's fix-attempt provenance to a
// build attempt record, and touches nothing else on the record: not Status,
// not Dispatches (those are set through transitionBuildAttempt, called
// separately, the same as any other attempt), not History. Mirrors
// attachBuildFreeCheckReport's narrow-setter discipline. Called only from
// the continue verification path (cmd/codex_continue.go), and only on the
// NEW canonical attempt created for this fix -- never on the parent attempt
// whose failing check triggered it.
func attachCheckFixAttempt(attemptRel string, record checkFixAttemptRecord) error {
	if store == nil || strings.TrimSpace(attemptRel) == "" {
		return fmt.Errorf("build attempt is not initialized")
	}
	var existing buildAttemptRecord
	return store.UpdateJSONAtomically(attemptRel, &existing, func() error {
		if existing.SchemaVersion != buildAttemptSchemaVersion || strings.TrimSpace(existing.ID) == "" {
			return fmt.Errorf("invalid build attempt record")
		}
		recordCopy := record
		existing.CheckFix = &recordCopy
		return nil
	})
}

// attachVerificationBoundary attaches the Queen's reconciled verification-
// boundary decision (queenApplyVerificationBoundary) to the exact build
// attempt it was made for, and touches nothing else on the record: not
// Status, not Dispatches, not Claims, not History. Mirrors
// attachBuildFreeCheckReport's and attachCheckFixAttempt's narrow-setter
// discipline.
//
// Re-attaching the identical decision is accepted as a no-op success -- an
// idempotent retry after a partial write must not be mistaken for a rewrite
// attempt. Attaching a DIFFERENT decision to an attempt that already carries
// one is refused, naming both the stored and the offered choice: once
// dispatch has begun against a recorded boundary choice, that choice cannot
// be silently rewritten out from under it.
func attachVerificationBoundary(attemptRel string, decision verificationBoundaryDecision) error {
	if store == nil || strings.TrimSpace(attemptRel) == "" {
		return fmt.Errorf("build attempt is not initialized")
	}
	var existing buildAttemptRecord
	return store.UpdateJSONAtomically(attemptRel, &existing, func() error {
		if existing.SchemaVersion != buildAttemptSchemaVersion || strings.TrimSpace(existing.ID) == "" {
			return fmt.Errorf("invalid build attempt record")
		}
		if existing.VerificationBoundary != nil && *existing.VerificationBoundary != decision {
			return fmt.Errorf("verification boundary for attempt %q is already recorded as %q -- refusing to silently rewrite it to %q",
				existing.ID, existing.VerificationBoundary.Choice, decision.Choice)
		}
		decisionCopy := decision
		existing.VerificationBoundary = &decisionCopy
		return nil
	})
}

// attachResultFilePrecision attaches the credited/uncredited file split
// (D-08) to a build attempt record, and touches nothing else on the record:
// not Status, not Dispatches, not Claims, not History. Mirrors
// attachBuildFreeCheckReport's and attachCheckFixAttempt's narrow-setter
// discipline. Called only from build-finalize, at the point the
// finalisation stage already knows the completed task set.
func attachResultFilePrecision(attemptRel string, credited []string, uncredited []buildAttemptUncreditedFile) error {
	if store == nil || strings.TrimSpace(attemptRel) == "" {
		return fmt.Errorf("build attempt is not initialized")
	}
	var existing buildAttemptRecord
	return store.UpdateJSONAtomically(attemptRel, &existing, func() error {
		if existing.SchemaVersion != buildAttemptSchemaVersion || strings.TrimSpace(existing.ID) == "" {
			return fmt.Errorf("invalid build attempt record")
		}
		existing.CreditedFiles = append([]string{}, credited...)
		existing.UncreditedFiles = append([]buildAttemptUncreditedFile{}, uncredited...)
		return nil
	})
}

// attachBuildPlanRealityReport attaches CAP-022's per-task plan-versus-
// reality comparison to a build attempt record, and touches nothing else on
// the record. Mirrors attachResultFilePrecision's and
// attachBuildFreeCheckReport's narrow-setter discipline.
func attachBuildPlanRealityReport(attemptRel string, report buildPlanRealityReport) error {
	if store == nil || strings.TrimSpace(attemptRel) == "" {
		return fmt.Errorf("build attempt is not initialized")
	}
	var existing buildAttemptRecord
	return store.UpdateJSONAtomically(attemptRel, &existing, func() error {
		if existing.SchemaVersion != buildAttemptSchemaVersion || strings.TrimSpace(existing.ID) == "" {
			return fmt.Errorf("invalid build attempt record")
		}
		reportCopy := report
		existing.PlanReality = &reportCopy
		return nil
	})
}

// attachBuildKnowledgeDeltas attaches CAP-066's content-level decision and
// learning deltas to the exact attempt that produced them, and touches
// nothing else on the record. Mirrors attachResultFilePrecision's narrow-
// setter discipline. Rendering these (cmd/lifecycle_closeout.go) is
// read-only -- this setter is the only writer.
func attachBuildKnowledgeDeltas(attemptRel string, deltas []buildAttemptKnowledgeDelta) error {
	if store == nil || strings.TrimSpace(attemptRel) == "" {
		return fmt.Errorf("build attempt is not initialized")
	}
	var existing buildAttemptRecord
	return store.UpdateJSONAtomically(attemptRel, &existing, func() error {
		if existing.SchemaVersion != buildAttemptSchemaVersion || strings.TrimSpace(existing.ID) == "" {
			return fmt.Errorf("invalid build attempt record")
		}
		existing.KnowledgeDeltas = append([]buildAttemptKnowledgeDelta{}, deltas...)
		return nil
	})
}

// buildAttemptWorktreeLocation resolves the owner-facing location for a
// dispatch's worker: the exact isolated workspace path and branch it lives
// on (colony.WorktreeEntry), matched by worker name, in worktree mode --
// or "the repository" when no worktree entry matches (shared-checkout mode,
// or a worktree already merged and removed).
func buildAttemptWorktreeLocation(workerName string, worktrees []colony.WorktreeEntry) string {
	workerName = strings.TrimSpace(workerName)
	if workerName == "" {
		return "the repository"
	}
	for _, entry := range worktrees {
		if strings.TrimSpace(entry.Agent) == workerName {
			return fmt.Sprintf("workspace %s on branch %s", strings.TrimSpace(entry.Path), strings.TrimSpace(entry.Branch))
		}
	}
	return "the repository"
}

// deriveResultFilePrecision computes the credited/uncredited file split for
// an attempt from its resolved dispatches (D-08). Credited files are every
// path the two-stage receipt trust boundary
// (admitCoherentJobTaskReceipts/finalizeCoherentJobTaskReceiptEvidence)
// actually finalised for a completed task -- either a whole-success
// dispatch's own reported outputs, or a grouped job's root-evidenced
// per-task TaskClaims (keyed off dispatch.CompletedTaskIDs). Uncredited
// files are every other path the attempt touched, together with where it
// lives. blockedTasks additionally withholds credit from a task CAP-022's
// plan-reality comparison (buildPlanRealityForDispatches) found missing a
// declared artifact for -- pass nil when that gate does not apply.
//
// This is a pure, read-only computation over already-resolved dispatches:
// it never deletes a file, never moves one into the credited set, and never
// makes its own decision about what counts as done -- that decision was
// already made by the two-stage boundary (or, for a plan-reality block, by
// buildPlanRealityForDispatches). It only reports the result honestly.
func deriveResultFilePrecision(dispatches []codexBuildDispatch, worktrees []colony.WorktreeEntry, blockedTasks map[string]struct{}) ([]string, []buildAttemptUncreditedFile) {
	credited := map[string]struct{}{}
	touched := map[string]struct{}{}
	location := map[string]string{}

	blockedReason := func(taskIDs []string) (string, bool) {
		for _, id := range taskIDs {
			if _, ok := blockedTasks[id]; ok {
				return fmt.Sprintf("credit blocked: task %s's plan-declared artifact is missing from the repository", id), true
			}
		}
		return "", false
	}

	for _, dispatch := range dispatches {
		status := strings.TrimSpace(dispatch.Status)
		wholeSuccess := status == "completed" || isNoChangeExternalBuildStatus(status)
		loc := buildAttemptWorktreeLocation(dispatch.Name, worktrees)
		covered := dispatchCoveredTaskIDs(dispatch)

		for _, path := range dispatch.Outputs {
			path = strings.TrimSpace(path)
			if path == "" {
				continue
			}
			touched[path] = struct{}{}
			if _, ok := location[path]; !ok {
				location[path] = loc
			}
		}

		if wholeSuccess {
			if reason, blocked := blockedReason(covered); blocked {
				// A flat whole-success result cannot be attributed to
				// individual covered tasks, so every output this dispatch
				// reported is conservatively withheld from credit rather
				// than guessing which belongs to the blocked task.
				for _, path := range dispatch.Outputs {
					if path = strings.TrimSpace(path); path != "" {
						location[path] = reason
					}
				}
				continue
			}
			for _, path := range dispatch.Outputs {
				if path = strings.TrimSpace(path); path != "" {
					credited[path] = struct{}{}
				}
			}
			continue
		}

		for _, claim := range dispatch.TaskClaims {
			paths := claimedPaths(claim)
			if _, ok := blockedTasks[claim.TaskID]; ok {
				reason := fmt.Sprintf("credit blocked: task %s's plan-declared artifact is missing from the repository", claim.TaskID)
				for _, path := range paths {
					touched[path] = struct{}{}
					location[path] = reason
				}
				continue
			}
			for _, path := range paths {
				credited[path] = struct{}{}
				touched[path] = struct{}{}
				if _, ok := location[path]; !ok {
					location[path] = loc
				}
			}
		}
	}

	creditedList := make([]string, 0, len(credited))
	for path := range credited {
		creditedList = append(creditedList, path)
	}
	sort.Strings(creditedList)

	var uncredited []buildAttemptUncreditedFile
	for path := range touched {
		if _, ok := credited[path]; ok {
			continue
		}
		uncredited = append(uncredited, buildAttemptUncreditedFile{Path: path, Location: location[path]})
	}
	sort.Slice(uncredited, func(i, j int) bool { return uncredited[i].Path < uncredited[j].Path })
	return creditedList, uncredited
}

// renderResultFilePrecisionCard renders the credited/uncredited file split
// already attached to a build attempt record. It is pure presentation over
// already-loaded, already-decided fields -- it reads nothing from disk and
// writes nothing, so rendering the same attempt twice is byte-identical by
// construction (D-08's idempotent-rendering guarantee).
func renderResultFilePrecisionCard(record buildAttemptRecord) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Credited files (%d):\n", len(record.CreditedFiles))
	if len(record.CreditedFiles) == 0 {
		b.WriteString("  none\n")
	}
	for _, path := range record.CreditedFiles {
		fmt.Fprintf(&b, "  - %s\n", path)
	}
	fmt.Fprintf(&b, "Uncredited files (%d):\n", len(record.UncreditedFiles))
	if len(record.UncreditedFiles) == 0 {
		b.WriteString("  none\n")
	}
	for _, file := range record.UncreditedFiles {
		fmt.Fprintf(&b, "  - %s (%s)\n", file.Path, file.Location)
	}
	return b.String()
}

// listBuildAttemptsForPhase loads every attempt record recorded for a phase
// (every "build/phase-<N>/attempts/*.json" file, skipping completion packets
// and durable "*.start-receipt.json" siblings), oldest and newest alike --
// the append-only journal in full, not just the latest pointer. Used to detect whether a
// check-fix attempt for a given check has already been recorded for this
// phase (planCheckFixAttempt, cmd/check_fix_attempt.go), which
// loadLatestBuildAttempt alone cannot answer once a later ordinary build
// attempt has superseded the fix attempt as "latest".
func listBuildAttemptsForPhase(phaseNum int) []buildAttemptRecord {
	if store == nil || phaseNum < 1 {
		return nil
	}
	dir := filepath.Join(store.BasePath(), "build", fmt.Sprintf("phase-%d", phaseNum), "attempts")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var records []buildAttemptRecord
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".json") || strings.HasSuffix(name, ".completion.json") || strings.HasSuffix(name, ".start-receipt.json") {
			continue
		}
		rel := filepath.ToSlash(filepath.Join("build", fmt.Sprintf("phase-%d", phaseNum), "attempts", name))
		var record buildAttemptRecord
		if err := store.LoadJSON(rel, &record); err != nil {
			continue
		}
		records = append(records, record)
	}
	return records
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
		if buildAttemptCompletionSealed(record) && record.CompletionSHA256 != "" && record.CompletionSHA256 != digest {
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
	displayPath, err := displayBuildAttemptDataPath(completionRel)
	if err != nil {
		return "", "", fmt.Errorf("display durable build completion path: %w", err)
	}
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
	if buildAttemptCompletionSealed(existing) && existing.CompletionSHA256 != "" && existing.CompletionSHA256 != digest {
		return "", "", fmt.Errorf("completion packet does not match the result already bound to attempt %s", existing.ID)
	}
	if existing.CompletionPath != "" {
		existingDisplayPath, pathErr := displayBuildAttemptDataPath(existing.CompletionPath)
		if pathErr != nil {
			return "", "", fmt.Errorf("build attempt %s has an invalid completion path: %w", existing.ID, pathErr)
		}
		if buildAttemptCompletionSealed(existing) && existingDisplayPath != displayPath {
			return "", "", fmt.Errorf("build attempt %s already points to another completion packet", existing.ID)
		}
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
	var previousCompletionPath string
	if err := store.UpdateJSONAtomically(attemptRel, &record, func() error {
		if record.ID != strings.TrimSpace(manifest.AttemptID) || record.Phase != manifest.Phase {
			return fmt.Errorf("build completion attempt identity does not match journal record")
		}
		if buildAttemptCompletionSealed(record) && record.CompletionSHA256 != "" && record.CompletionSHA256 != digest {
			return fmt.Errorf("completion packet does not match the result already bound to attempt %s", record.ID)
		}
		if record.CompletionPath != "" {
			recordDisplayPath, pathErr := displayBuildAttemptDataPath(record.CompletionPath)
			if pathErr != nil {
				return fmt.Errorf("build attempt %s has an invalid completion path: %w", record.ID, pathErr)
			}
			if buildAttemptCompletionSealed(record) && recordDisplayPath != displayPath {
				return fmt.Errorf("build attempt %s already points to another completion packet", record.ID)
			}
		}
		previousCompletionPath = strings.TrimSpace(record.CompletionPath)
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
	// D-08: while unsealed, a rebind may point the attempt at a different
	// durable completion path (durableBuildCompletionPath is a pure function
	// of phase+attempt ID today, so this rarely changes in practice, but
	// nothing should be left on disk claiming to belong to this attempt once
	// the record no longer points at it -- T-163.1-13).
	if previousCompletionPath != "" && previousCompletionPath != displayPath {
		previousRel, pathErr := canonicalBuildAttemptDataPath(previousCompletionPath)
		if pathErr != nil {
			return "", "", fmt.Errorf("build attempt previous completion path: %w", pathErr)
		}
		_ = os.Remove(filepath.Join(store.BasePath(), filepath.FromSlash(previousRel)))
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

// isCommittedPartialAttemptReplay reports whether phase manifest.Phase's
// latest attempt is a committed PARTIAL whose durable completion digest is
// exactly completionDigest -- i.e. the caller is resubmitting the identical
// completion packet that produced the partial credit already on disk.
//
// This is the same proof-by-digest the BUILT path has always used ("exact
// retry safety is proved by the durable attempt and completion hashes"). It
// exists because a committed partial legitimately moves the colony on -- it
// credits tasks and appends events -- so the state-drift and plan-hash guards
// that protect a stale packet would otherwise refuse the runtime's own
// documented instruction to "rerun build-finalize with the same completion
// packet" (WR-02, 195-REVIEW.md). A DIFFERENT packet for the same partial
// attempt is still refused by every one of those guards, unchanged.
func isCommittedPartialAttemptReplay(manifest codexBuildManifest, completionDigest string) bool {
	attemptID := strings.TrimSpace(manifest.AttemptID)
	completionDigest = strings.TrimSpace(completionDigest)
	if attemptID == "" || completionDigest == "" {
		return false
	}
	_, latest, ok := loadLatestBuildAttempt(manifest.Phase)
	if !ok {
		return false
	}
	return latest.ID == attemptID &&
		strings.TrimSpace(latest.Status) == buildAttemptPartial &&
		strings.TrimSpace(latest.CompletionSHA256) == completionDigest
}

// isCommittedPartialAttempt reports only whether the manifest names the
// current partial parent. Finalize uses this narrower fact to reach the
// read-only partial-replay verifier even for a changed packet; manifest,
// receipt, child, and completion digests are still validated there before a
// result is returned, and no mutation is permitted on that route.
func isCommittedPartialAttempt(manifest codexBuildManifest) bool {
	attemptID := strings.TrimSpace(manifest.AttemptID)
	if attemptID == "" {
		return false
	}
	_, latest, ok := loadLatestBuildAttempt(manifest.Phase)
	return ok && latest.ID == attemptID && strings.TrimSpace(latest.Status) == buildAttemptPartial
}

func validateBuildAttemptManifestBinding(manifest codexBuildManifest, state colony.ColonyState, partialReplay bool) (buildAttemptManifestBinding, error) {
	classification, err := classifyBuildManifestBinding(manifest)
	if err != nil {
		return buildAttemptManifestBinding{}, err
	}
	if classification == buildManifestBindingLegacy {
		return buildAttemptManifestBinding{Legacy: true}, nil
	}
	attemptID := strings.TrimSpace(manifest.AttemptID)
	attemptPath := strings.TrimSpace(manifest.AttemptPath)
	if !validBuildAttemptID(attemptID) {
		return buildAttemptManifestBinding{}, fmt.Errorf("dispatch_manifest attempt_id %q is invalid", attemptID)
	}
	expectedRel := filepath.ToSlash(filepath.Join("build", fmt.Sprintf("phase-%d", manifest.Phase), "attempts", attemptID+".json"))
	attemptRel, err := canonicalBuildAttemptDataPath(attemptPath)
	if err != nil {
		return buildAttemptManifestBinding{}, fmt.Errorf("dispatch_manifest attempt_path %q is invalid: %w", attemptPath, err)
	}
	if attemptRel != expectedRel {
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
	if latest.Status != buildAttemptBuilt && !projectedBuiltState && !partialReplay {
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
	case buildAttemptPartial:
		// WR-02: a committed partial is a terminal outcome the runtime itself
		// tells callers to resubmit. The identical packet replays; a different
		// one is refused by name rather than silently replacing the credit
		// already committed.
		if !partialReplay {
			return buildAttemptManifestBinding{}, fmt.Errorf("build attempt %s recorded partial credit from a different completion packet; resubmit that same packet to see its recovery command, or run its recovery command to finish the remaining tasks", attemptID)
		}
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
	attemptRel, err := canonicalBuildAttemptDataPath(pointer.Path)
	if err != nil || attemptRel == "" {
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
	case buildAttemptBuilt, buildAttemptFailed, buildAttemptInterrupted, buildAttemptPartial:
		return true
	default:
		return false
	}
}

// buildAttemptCompletionSealed reports whether record's completion packet
// binding is permanently locked (D-08). It compares against
// buildAttemptBuilt specifically -- not buildAttemptStatusTerminal and not
// the buildAttemptTerminal status constant -- because buildAttemptTerminal
// is written by build-finalize (cmd/codex_build_finalize.go, at the
// "external terminal worker results recorded" transition) BEFORE finalize
// has actually committed the built lifecycle state (cmd/codex_build_finalize.go,
// at the "external built lifecycle state committed" transition). Gating the
// seal on buildAttemptTerminal would close the rebind window before finalize
// had succeeded -- the exact regression D-08 exists to prevent. `failed` and
// `interrupted` attempts are recoverable and must stay rebindable too.
func buildAttemptCompletionSealed(record buildAttemptRecord) bool {
	return strings.TrimSpace(record.Status) == buildAttemptBuilt
}

// buildAttemptRecordedTerminalEvidence reports whether this attempt itself got
// as far as recording terminal worker evidence -- the transition that writes
// the completion digest and the claims together, immediately before the
// lifecycle commit.
//
// It is the difference between "this attempt was committed and its journal
// write was lost" and "this attempt has never been finalized at all". Only the
// first is reconcilable; treating the second as reconcilable produced a
// deadlock with no in-band exit (see the routing comment in
// cmd/codex_build_finalize.go and TestForcedRedispatchAfterBuiltIsNotADeadlock).
//
// Both halves are required. The digest alone would admit an attempt whose
// evidence was recorded but whose claims never landed, and Claims alone would
// admit one with no packet bound to it.
func buildAttemptRecordedTerminalEvidence(record buildAttemptRecord) bool {
	return strings.TrimSpace(record.CompletionSHA256) != "" && record.Claims != nil
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
