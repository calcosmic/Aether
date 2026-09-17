package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/calcosmic/Aether/pkg/colony"
)

// These hooks exercise the real admission/commit boundaries without global
// callbacks or a second completion writer. Production uses the zero value.
type codexNativeFinalizeHooks struct {
	BeforeStage        func()
	BeforeCommit       func()
	AfterCurrencyCheck func()
}

func buildAttemptHasNativeWorkers(record buildAttemptRecord) bool {
	for _, worker := range record.WorkerRuns {
		if worker.Native != nil {
			return true
		}
	}
	return false
}

// validateCodexNativeCompletion trusts only the saved attempt's lane and
// terminal journal. Neither an unbound manifest nor wrapper-authored claims
// can select the generic compatibility path for an existing native attempt.
// This is admission, not credit: receipts still go through the shared finalizer.
func validateCodexNativeCompletion(record buildAttemptRecord, completion codexExternalBuildCompletion) error {
	if !buildAttemptHasNativeWorkers(record) {
		return nil
	}
	// Preserve unknown-field refusal even on the byte-stable replay path,
	// which deliberately skips filesystem-dependent semantic checks.
	raw, err := completion.structuralInput()
	if err != nil {
		return err
	}
	if violations := validateCompletionPacketStructure(raw); len(violations) != 0 {
		return &completionContractError{Violations: violations}
	}
	manifest := completion.activeManifest()
	if manifest == nil || record.PlanManifest == nil || manifest.ExecutionBinding == nil || record.PlanManifest.ExecutionBinding == nil {
		return fmt.Errorf("native completion requires the saved execution binding")
	}
	if record.ExecutionOwner != "host-queen" || !record.PlanManifest.PlanOnly || manifest.AttemptID != record.ID || manifest.Phase != record.Phase {
		return fmt.Errorf("native completion does not identify its accepted attempt")
	}
	if err := validateBuildExecutionBinding(record, *manifest.ExecutionBinding, record.ManifestSHA256, false); err != nil {
		return err
	}
	if *manifest.ExecutionBinding != *record.PlanManifest.ExecutionBinding {
		return fmt.Errorf("native completion execution binding changed")
	}
	digest, err := buildManifestSHA256(*record.PlanManifest)
	if err != nil || digest != record.ManifestSHA256 {
		return fmt.Errorf("saved native manifest content no longer matches its accepted digest")
	}
	_, latest, ok := loadLatestBuildAttempt(record.Phase)
	if !ok || latest.ID != record.ID {
		return fmt.Errorf("native completion attempt has been superseded")
	}
	if len(record.WorkerRuns) != len(record.PlanManifest.Dispatches) {
		return fmt.Errorf("native completion requires exactly one saved result per dispatch")
	}
	seen := make(map[string]bool)
	for _, dispatch := range record.PlanManifest.Dispatches {
		var matched *buildAttemptWorkerRun
		for i := range record.WorkerRuns {
			worker := &record.WorkerRuns[i]
			if worker.WorkerName == dispatch.Name && worker.TaskID == normalizedDispatchTaskID(dispatch) {
				if matched != nil {
					return fmt.Errorf("native completion has duplicate saved worker %s", dispatch.Name)
				}
				matched = worker
			}
		}
		if matched == nil || !codexNativeWorkerIsTerminal(*matched) {
			return fmt.Errorf("native completion requires terminal evidence for %s", dispatch.Name)
		}
		worker, native := *matched, matched.Native
		if err := validateCodexNativeSavedWorker(record, dispatch, worker); err != nil {
			return err
		}
		if seen[worker.ProviderRunID] {
			return fmt.Errorf("native completion has duplicate launch identity")
		}
		seen[worker.ProviderRunID] = true
		if _, err := canonicalCodexNativeEventHash(native.SourceEventID, native.SourceEventSHA256); err != nil {
			return err
		}
		if native.LaunchState == "no_launch" {
			// An actual no-launch receipt has no child and can never carry work.
			if native.ChildID != "" || native.Release != "" || worker.Status != buildWorkerCancelled || len(worker.Result.TaskReceipts) != 0 || len(worker.Result.FilesCreated)+len(worker.Result.FilesModified)+len(worker.Result.TestsWritten) != 0 {
				return fmt.Errorf("native no-launch result cannot claim a child or completed work")
			}
		} else {
			release := strings.Fields(native.Release)
			if native.ChildID == "" || native.BoundAt == "" || len(release) != 4 || release[0] != "AETHER_NATIVE_RELEASE" || release[1] != worker.ProviderRunID || release[2] != native.ChildID || seen["child:"+native.ChildID] {
				return fmt.Errorf("native completion child no longer matches its accepted release")
			}
			seen["child:"+native.ChildID] = true
		}
		result, err := normalizedBuildWorkerResult(worker.Result)
		if err != nil {
			return err
		}
		resultDigest, err := jsonSHA256(result)
		if err != nil || resultDigest != worker.ResultSHA256 || result.Name != worker.WorkerName || result.Caste != worker.Caste || result.TaskID != worker.TaskID || result.Status != worker.Status {
			return fmt.Errorf("native completion result no longer matches accepted terminal digest for %s", dispatch.Name)
		}
	}
	projected, complete := buildCompletionFromWorkerRuns(record)
	if !complete {
		return fmt.Errorf("native completion has unfinished workers")
	}
	want, err := jsonSHA256(projected)
	if err != nil {
		return err
	}
	got, err := jsonSHA256(completion)
	if err != nil {
		return err
	}
	if got != want {
		return fmt.Errorf("native completion does not match the exact saved journal projection")
	}
	return nil
}

// The caller holds the existing repository session, worker mutex and attempt
// file lock in that order. The state primitive then reads the latest state.
func validateCodexNativeFinalizeCurrency(record buildAttemptRecord, completion codexExternalBuildCompletion, state colony.ColonyState, expectedStateDigest string) error {
	if err := validateCodexNativeCompletion(record, completion); err != nil {
		return err
	}
	if !buildAttemptHasNativeWorkers(record) {
		return nil
	}
	if err := validateBuildFinalizeStateStillCurrent(state, record.Phase); err != nil {
		return err
	}
	normalized, err := migrateLoadedPlanningState(normalizeLegacyColonyState(state))
	if err != nil {
		return err
	}
	stateDigest, err := jsonSHA256(normalized)
	if err != nil {
		return err
	}
	if stateDigest != expectedStateDigest {
		return runtimeStateSupersededError(record.Phase, "native completion's validated colony state changed before credit")
	}
	return nil
}

// A real pause/resume changes session metadata, not the accepted work. The
// existing transaction receipt validates the exact resumed bytes first; then
// compare all work/colony fields against the hash-pinned pause preimage.
// This is deliberately not a general state-hash exemption for native workers.
func codexNativeAttemptSessionState(record buildAttemptRecord, state colony.ColonyState) bool {
	nativeIntent := record.HostPlatform == "codex" && record.ExecutionOwner == "host-queen" && record.PlanManifest != nil && record.PlanManifest.PlanOnly
	if (!buildAttemptHasNativeWorkers(record) && !nativeIntent) || state.PauseHandoff == nil {
		return false
	}
	handoff, err := loadValidatedPauseHandoff(*state.PauseHandoff)
	if err != nil || handoff.AttemptID != record.ID {
		return false
	}
	command, transactionID := "pause", handoff.Transaction.ID
	if !state.Paused {
		if state.RunID == nil || *state.RunID != "resume-"+compactLifecycleID(handoff.HandoffID) {
			return false
		}
		command, transactionID = "resume", resumeTransactionID(handoff.HandoffID)
	}
	tx, err := beginLifecycleTransaction(pauseResumeTransactionConfig(transactionID, command))
	if err != nil {
		return false
	}
	if receipt, ok, err := tx.loadCommittedReceipt(); err != nil || !ok || receipt.Command != command {
		return false
	}
	original, ok := codexNativePrePauseState(record, handoff)
	if !ok {
		return false
	}
	// Only fields written by the ordinary pause/resume transaction may differ.
	state.Events, state.SessionID, state.RunID = original.Events, original.SessionID, original.RunID
	state.Paused, state.PausedAt, state.PauseHandoff = original.Paused, original.PausedAt, original.PauseHandoff
	state.RecoveryProvenance = original.RecoveryProvenance
	state.BuildStartedAt = original.BuildStartedAt
	if original.State == colony.StateEXECUTING && state.State == colony.StateREADY {
		state.State = original.State
	}
	digest, err := jsonSHA256(state)
	return err == nil && digest == record.OriginalStateSHA
}

var errCodexNativeFinalizeReadOnly = errors.New("native finalizer attempt read: no journal write")

// Recovery and pre-credit admission share the same current work-state check.
// Committed built/partial replay continues using its established receipt path.
func validateCodexNativeAttemptState(record buildAttemptRecord, state colony.ColonyState) error {
	state, err := migrateLoadedPlanningState(normalizeLegacyColonyState(state))
	if err != nil {
		return err
	}
	currencyState := state
	currencyState.Paused = false // Session provenance below must justify an actual pause.
	if err := validateBuildFinalizeStateStillCurrent(currencyState, record.Phase); err != nil {
		return err
	}
	if record.PlanManifest == nil {
		return fmt.Errorf("native attempt has no accepted plan")
	}
	if err := validateBuildManifestPlanRevision(*record.PlanManifest, state, false); err != nil {
		return err
	}
	digest, err := jsonSHA256(state)
	if err != nil {
		return err
	}
	if record.OriginalStateSHA == "" || digest != record.OriginalStateSHA {
		if !codexNativeAttemptSessionState(record, state) {
			return runtimeStateSupersededError(record.Phase, "native attempt's accepted colony state changed")
		}
	}
	return nil
}

// Plan-only attempts need not have a build checkpoint. The pause transaction's
// validated preimage provides the exact earlier state without a new ledger.
// Follow earlier pause references for repeated pause/resume episodes, bounded
// and cycle checked, until the accepted attempt hash proves the baseline.
func codexNativePrePauseState(record buildAttemptRecord, handoff colony.PauseHandoff) (colony.ColonyState, bool) {
	seen := make(map[string]bool)
	for depth := 0; depth < 32; depth++ {
		if seen[handoff.Transaction.ID] {
			return colony.ColonyState{}, false
		}
		seen[handoff.Transaction.ID] = true
		tx, err := beginLifecycleTransaction(pauseResumeTransactionConfig(handoff.Transaction.ID, "pause"))
		if err != nil {
			return colony.ColonyState{}, false
		}
		tx.intent, tx.progress, err = tx.loadJournal()
		if err != nil || tx.intent.Record.Command != "pause" {
			return colony.ColonyState{}, false
		}
		manifests, err := tx.validateRecoveryEvidence()
		if err != nil {
			return colony.ColonyState{}, false
		}
		var original colony.ColonyState
		found := false
		for _, manifest := range manifests {
			if manifest.Kind != lifecycleTransactionRootData {
				continue
			}
			for _, target := range manifest.Targets {
				if target.RelativeTarget != "COLONY_STATE.json" || target.TargetPath != filepath.Join(store.BasePath(), "COLONY_STATE.json") {
					continue
				}
				raw, err := readLifecycleEvidenceFile(target.PreimagePath)
				if err != nil || lifecycleDigest(raw) != target.BeforeDigest || json.Unmarshal(raw, &original) != nil {
					return colony.ColonyState{}, false
				}
				found = true
			}
		}
		if !found {
			return colony.ColonyState{}, false
		}
		original, err = migrateLoadedPlanningState(normalizeLegacyColonyState(original))
		if err != nil {
			return colony.ColonyState{}, false
		}
		digest, err := jsonSHA256(original)
		if err == nil && digest == record.OriginalStateSHA {
			return original, true
		}
		if original.PauseHandoff == nil {
			return colony.ColonyState{}, false
		}
		handoff, err = loadValidatedPauseHandoff(*original.PauseHandoff)
		if err != nil {
			return colony.ColonyState{}, false
		}
	}
	return colony.ColonyState{}, false
}
