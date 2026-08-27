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
	// ParentAttemptID is set ONLY by attachBuildAttemptParentLink, on a NEW
	// attempt record beginChildBuildAttempt just created for D-10's
	// unfinished-only recovery job -- never on the parent attempt whose
	// partial credit triggered it. Optional and omitempty: existing attempt
	// JSON written before this field existed still decodes cleanly with an
	// empty string here (mirrors CheckFix's own precedent).
	ParentAttemptID string `json:"parent_attempt_id,omitempty"`
	// ParentJobName is the coherent job name (cmd/coherent_jobs.go) this
	// attempt's retry job was derived from -- the original grouped job whose
	// partial credit created this recovery attempt.
	ParentJobName string `json:"parent_job_name,omitempty"`
}

type latestBuildAttemptPointer struct {
	SchemaVersion int    `json:"schema_version"`
	AttemptID     string `json:"attempt_id"`
	Path          string `json:"path"`
	UpdatedAt     string `json:"updated_at"`
}

func beginBuildAttempt(state colony.ColonyState, phaseNum int, phase colony.Phase, startedAt time.Time, selectedTaskIDs []string, checkpointRel, manifestRel, claimsRel, executionOwner string, dispatches []codexBuildDispatch) (string, error) {
	return beginBuildAttemptRecord(state, phaseNum, phase, startedAt, selectedTaskIDs, checkpointRel, manifestRel, claimsRel, executionOwner, dispatches, true)
}

// beginBuildAttemptRecord is beginBuildAttempt's implementation, plus the one
// switch beginChildBuildAttempt needs: whether this new record becomes the
// phase's "latest attempt" (WR-03, 195-REVIEW.md).
//
// Every attempt that actually dispatches workers must become the latest one --
// that pointer is how the runtime finds the in-flight attempt. A D-10 recovery
// record is the opposite: it describes work still to do and never dispatches
// anything itself. Pointing "latest" at it left the phase looking like it had
// an ACTIVE attempt while the parent build's dispatch-start marker was still
// set, so the very next `aether build <N> --plan-only` -- the interactive
// wrapper's only build path -- refused the owner's own recovery command with
// "already has workers in flight", naming an attempt that had no completion
// packet to finalize. The recovery record stays fully discoverable through the
// phase's attempt journal (findExistingBuildAttemptRetry), which is the only
// reader it ever had.
func beginBuildAttemptRecord(state colony.ColonyState, phaseNum int, phase colony.Phase, startedAt time.Time, selectedTaskIDs []string, checkpointRel, manifestRel, claimsRel, executionOwner string, dispatches []codexBuildDispatch, makeLatest bool) (string, error) {
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
	if makeLatest {
		pointerRel := latestBuildAttemptPointerPath(phaseNum)
		if err := store.SaveJSON(pointerRel, latestBuildAttemptPointer{
			SchemaVersion: buildAttemptSchemaVersion,
			AttemptID:     attemptID,
			Path:          displayDataPath(attemptRel),
			UpdatedAt:     now,
		}); err != nil {
			return "", fmt.Errorf("save latest build attempt pointer: %w", err)
		}
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
// build. It always lives on a NEW attempt record produced through
// beginBuildAttempt/transitionBuildAttempt like any other attempt -- it
// never overwrites or mutates the parent attempt's own Dispatches, Claims,
// or Status (TestFixAttemptNeverOverwritesTheFirstResult); the two exist as
// separate, independently-readable journal entries, the same append-only
// discipline outOfBandVerificationRecord already established for a
// differently-caused non-worker closure.
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
// NEW attempt record beginBuildAttempt created for this fix attempt -- never
// on the parent attempt whose failing check triggered it.
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

// listBuildAttemptsForPhase loads every attempt record recorded for a phase
// (every "build/phase-<N>/attempts/*.json" file, skipping the
// "*.completion.json" siblings), oldest and newest alike -- the append-only
// journal in full, not just the latest pointer. Used to detect whether a
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
		if entry.IsDir() || !strings.HasSuffix(name, ".json") || strings.HasSuffix(name, ".completion.json") {
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
	if buildAttemptCompletionSealed(existing) && existing.CompletionSHA256 != "" && existing.CompletionSHA256 != digest {
		return "", "", fmt.Errorf("completion packet does not match the result already bound to attempt %s", existing.ID)
	}
	if buildAttemptCompletionSealed(existing) && existing.CompletionPath != "" && filepath.ToSlash(existing.CompletionPath) != displayPath {
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
	var previousCompletionPath string
	if err := store.UpdateJSONAtomically(attemptRel, &record, func() error {
		if record.ID != strings.TrimSpace(manifest.AttemptID) || record.Phase != manifest.Phase {
			return fmt.Errorf("build completion attempt identity does not match journal record")
		}
		if buildAttemptCompletionSealed(record) && record.CompletionSHA256 != "" && record.CompletionSHA256 != digest {
			return fmt.Errorf("completion packet does not match the result already bound to attempt %s", record.ID)
		}
		if buildAttemptCompletionSealed(record) && record.CompletionPath != "" && filepath.ToSlash(record.CompletionPath) != displayPath {
			return fmt.Errorf("build attempt %s already points to another completion packet", record.ID)
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
		if previousRel := strings.TrimPrefix(filepath.ToSlash(previousCompletionPath), ".aether/data/"); previousRel != "" {
			_ = os.Remove(filepath.Join(store.BasePath(), filepath.FromSlash(previousRel)))
		}
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
	case buildAttemptBuilt, buildAttemptFailed, buildAttemptInterrupted, buildAttemptPartial:
		return true
	default:
		return false
	}
}

// attachBuildAttemptParentLink attaches D-10 append-only retry provenance to
// a NEW attempt record -- the second half of "begin, then attach" that
// mirrors attachCheckFixAttempt's narrow-setter discipline (touches nothing
// on the record besides these two fields). Called only on the retry attempt
// beginBuildAttempt just created, identified by its own attemptRel, NEVER on
// the parent attempt whose partial credit triggered this retry.
func attachBuildAttemptParentLink(attemptRel, parentAttemptID, parentJobName string) error {
	if store == nil || strings.TrimSpace(attemptRel) == "" {
		return fmt.Errorf("build attempt is not initialized")
	}
	parentAttemptID = strings.TrimSpace(parentAttemptID)
	parentJobName = strings.TrimSpace(parentJobName)
	if parentAttemptID == "" {
		return fmt.Errorf("parent attempt id is required to link a retry attempt")
	}
	var record buildAttemptRecord
	return store.UpdateJSONAtomically(attemptRel, &record, func() error {
		if record.SchemaVersion != buildAttemptSchemaVersion || strings.TrimSpace(record.ID) == "" {
			return fmt.Errorf("invalid build attempt record")
		}
		record.ParentAttemptID = parentAttemptID
		record.ParentJobName = parentJobName
		return nil
	})
}

// beginChildBuildAttempt creates a NEW append-only build attempt for a D-10
// unfinished-only retry job, linked to -- but never mutating -- the parent
// attempt that produced the partial credit that triggered it. It reuses
// beginBuildAttempt's own record-writing path, then attaches parent provenance
// onto the brand-new record only. The parent attempt's own file is never
// opened by this function.
//
// It deliberately does NOT move the phase's latest-attempt pointer (WR-03):
// a recovery record dispatches nothing, so treating it as the phase's live
// attempt blocked the very plan-only call the owner's recovery command makes.
// See beginBuildAttemptRecord's doc comment.
func beginChildBuildAttempt(state colony.ColonyState, phaseNum int, phase colony.Phase, startedAt time.Time, parentAttemptID, parentJobName string, retryTaskIDs []string, checkpointRel, manifestRel, claimsRel, executionOwner string, dispatches []codexBuildDispatch) (string, error) {
	attemptRel, err := beginBuildAttemptRecord(state, phaseNum, phase, startedAt, retryTaskIDs, checkpointRel, manifestRel, claimsRel, executionOwner, dispatches, false)
	if err != nil {
		return "", err
	}
	if err := attachBuildAttemptParentLink(attemptRel, parentAttemptID, parentJobName); err != nil {
		return "", err
	}
	return attemptRel, nil
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
