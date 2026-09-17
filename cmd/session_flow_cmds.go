package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/storage"
	"github.com/spf13/cobra"
)

const sessionStaleThreshold = 24 * time.Hour
const handoffStateFence = "aether-colony-state"

const (
	pauseHandoffDataPath = "pause-handoff.json"
	pauseReplayMessage   = "Already paused; the existing validated handoff was retained."
)

var handoffPhaseLinePattern = regexp.MustCompile(`(?i)(?:Current:|Phase:)\s*([0-9]+)\s*/\s*([0-9]+)(?:\s*(?:—|-)\s*(.*))?`)
var resumeNoHandoff bool

// pauseResumeLifecycleFault is intentionally nil in production. Focused crash
// tests inject failures at the transaction coordinator's named stage hooks so
// pause and resume prove the same restart behavior as every lifecycle writer.
var pauseResumeLifecycleFault lifecycleTransactionFaultHook

type pauseResumeLifecycleOutcome struct {
	NativeRecovery   *codexNativeRecovery
	Handoff          colony.PauseHandoff
	Receipt          colony.LifecycleReceipt
	Provenance       colony.RecoveryProvenance
	StateEffect      colony.LifecycleStateEffect
	Replay           bool
	Message          string
	WorktreesCleaned int
}

type pauseBoundaryPendingError struct {
	Boundary string
	Attempt  string
}

func (err pauseBoundaryPendingError) Error() string {
	if err.Attempt == "" {
		return fmt.Sprintf("pause is waiting for safe boundary %q", err.Boundary)
	}
	return fmt.Sprintf("pause is waiting for safe boundary %q on attempt %s", err.Boundary, err.Attempt)
}

// sessionFreshnessResult describes how fresh a session is for resume.
type sessionFreshnessResult struct {
	Fresh       bool
	Age         time.Duration
	GitMatch    bool
	GitCheck    bool // whether git HEAD comparison was performed
	SessionID   string
	BaselineSHA string
	CurrentSHA  string
}

// sessionVerifyFresh checks session age and git HEAD to detect stale sessions.
func sessionVerifyFresh(s *storage.Store) sessionFreshnessResult {
	var session colony.SessionFile
	if err := store.LoadJSON("session.json", &session); err != nil {
		return sessionFreshnessResult{Fresh: false}
	}

	result := sessionFreshnessResult{
		SessionID:   session.SessionID,
		BaselineSHA: session.BaselineCommit,
	}

	// Check age from started_at
	if startedAt := strings.TrimSpace(session.StartedAt); startedAt != "" {
		if t, err := time.Parse(time.RFC3339, startedAt); err == nil {
			result.Age = time.Since(t)
			result.Fresh = result.Age < sessionStaleThreshold
		} else {
			result.Fresh = false
		}
	} else {
		result.Fresh = false
	}

	// Check git HEAD match
	currentHEAD := getGitHEAD()
	result.CurrentSHA = currentHEAD
	if session.BaselineCommit != "" && currentHEAD != "" {
		result.GitCheck = true
		result.GitMatch = session.BaselineCommit == currentHEAD
		// Git mismatch means repo changed since session — treat as stale
		if !result.GitMatch {
			result.Fresh = false
		}
	}

	return result
}

var pauseColonyCmd = &cobra.Command{
	Use:   "pause",
	Short: "Stop at a safe boundary and save one resumable handoff.",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}
		writeLegacySessionRedirectNotice()
		outcome, err := pauseColonyAt(time.Now().UTC())
		if err != nil {
			if pending, ok := err.(pauseBoundaryPendingError); ok {
				result := map[string]interface{}{
					"paused": false, "safe_boundary": pending.Boundary,
					"attempt_id": pending.Attempt, "state_effect": colony.LifecycleStateEffectNone,
					"outcome_kind": colony.OutcomeKindNoChange,
					"message":      pending.Error(), "next": "aether pause",
				}
				closeLifecycleCommand(result, "pause", "aether pause", "The current work has not reached a safe pause boundary yet.")
				if closeErr := applyLifecycleCloseout(result, "pause", LifecycleCloseoutDetails{
					Summary: pending.Error(),
					Evidence: []colony.LifecycleEvidence{{
						ID: pending.Attempt, Kind: "safe_boundary", Summary: "The requested safe boundary is still pending",
					}},
				}); closeErr != nil {
					outputError(1, closeErr.Error(), result)
					return nil
				}
				outputWorkflow(result, pending.Error()+"\n"+renderLifecycleCloseoutFromResult(result, detectPlatform()))
				return nil
			}
			renderRecoveryMenu("pause", err.Error(), []string{"aether status", "aether pause"})
			return nil
		}
		state := loadStateAfterLifecycleOutcome()
		result := map[string]interface{}{
			"paused":        true,
			"goal":          colonyStateGoalText(state),
			"state":         state.State,
			"current_phase": state.CurrentPhase,
			"phase_name":    lookupPhaseName(state, state.CurrentPhase),
			"handoff_path":  handoffDocumentPath(),
			"handoff_id":    outcome.Handoff.HandoffID,
			"safe_boundary": outcome.Handoff.SafeBoundary,
			"receipt_id":    outcome.Receipt.ReceiptID,
			"provenance":    outcome.Provenance,
			"state_effect":  outcome.StateEffect,
			"replay":        outcome.Replay,
			"next":          "aether resume",
		}
		if outcome.Message != "" {
			result["message"] = outcome.Message
		}
		closeLifecycleCommand(result, "pause", "", "")
		pauseSummary := "The colony paused at a validated safe boundary."
		if outcome.Replay {
			pauseSummary = "The existing validated pause receipt was replayed without changing the saved handoff."
		}
		if err := applyLifecycleCloseout(result, "pause", LifecycleCloseoutDetails{
			Summary: pauseSummary,
			Evidence: []colony.LifecycleEvidence{
				{ID: outcome.Handoff.HandoffID, Kind: "handoff", Source: handoffDocumentPath(), Summary: "Validated safe-boundary handoff"},
				{ID: outcome.Receipt.ReceiptID, Kind: "receipt", Source: pauseHandoffDataPath, Summary: "Durable pause receipt"},
			},
		}); err != nil {
			outputError(1, err.Error(), result)
			return nil
		}
		visual := appendLifecycleCloseoutVisual(renderPauseVisual(result), result, detectPlatform())
		if outcome.Replay {
			visual = pauseReplayMessage + "\n" + renderLifecycleCloseoutFromResult(result, detectPlatform())
		}
		outputWorkflow(result, visual)
		return nil
	},
}

// staleSignalInfo represents a stale FOCUS signal for wrapper consumption.
type staleSignalInfo struct {
	ID           string `json:"id"`
	Type         string `json:"type"`
	Content      string `json:"content"`
	SourcePhase  int    `json:"source_phase"`
	CurrentPhase int    `json:"current_phase"`
}

// detectStaleFocusSignals finds active FOCUS signals whose source_phase
// is less than the current colony phase. Signals without source_phase are
// NOT flagged (backward compatible with signals created before this feature).
// Only FOCUS signals are checked per D-07.
func detectStaleFocusSignals(s *storage.Store, currentPhase int) []staleSignalInfo {
	var pf colony.PheromoneFile
	if err := s.LoadJSON("pheromones.json", &pf); err != nil {
		return nil
	}
	var stale []staleSignalInfo
	for _, sig := range pf.Signals {
		if !sig.Active || sig.Type != "FOCUS" {
			continue
		}
		if sig.SourcePhase == nil {
			continue // Unknown phase -- backward compat, don't flag
		}
		if *sig.SourcePhase < currentPhase {
			stale = append(stale, staleSignalInfo{
				ID:           sig.ID,
				Type:         sig.Type,
				Content:      extractContentText(sig.Content),
				SourcePhase:  *sig.SourcePhase,
				CurrentPhase: currentPhase,
			})
		}
	}
	return stale
}

var resumeColonyCmd = &cobra.Command{
	Use:   "resume",
	Short: "Validate and restore the safest honest recovery point.",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}
		writeLegacySessionRedirectNotice()
		outcome, err := resumeColonyAt(time.Now().UTC())
		if err != nil {
			renderRecoveryMenu("resume", err.Error(), []string{"aether status", "aether resume"})
			return nil
		}
		if outcome.NativeRecovery != nil {
			result := buildResumeDashboardResult()
			applyCodexNativeRecovery(result, outcome.NativeRecovery)
			result["resumed"] = false
			result["state_effect"] = colony.LifecycleStateEffectNone
			result["outcome_kind"] = colony.OutcomeKindNoChange
			result["message"] = "Saved native work is available for recovery; no new lifecycle episode was created."
			closeLifecycleCommand(result, "picking the project back up", outcome.NativeRecovery.Next, outcome.NativeRecovery.Why)
			outputWorkflow(result, renderCodexNativeRecovery(outcome.NativeRecovery))
			return nil
		}
		if outcome.Provenance == colony.RecoveryProvenanceConflicting || outcome.Provenance == colony.RecoveryProvenanceUnknown {
			result := map[string]interface{}{
				"resumed": false, "provenance": outcome.Provenance,
				"state_effect": outcome.StateEffect, "message": outcome.Message,
				"outcome_kind": colony.OutcomeKindRecoveryRequired,
				"next":         "aether status",
			}
			closeLifecycleCommand(result, "resume", "aether status", "Inspect the conflicting recovery evidence before trying again.")
			if closeErr := applyLifecycleCloseout(result, "resume", LifecycleCloseoutDetails{
				Summary: outcome.Message,
				Blockers: []colony.LifecycleIssue{{
					ID: "resume-provenance-" + string(outcome.Provenance), Summary: "Recovery evidence is " + string(outcome.Provenance),
				}},
			}); closeErr != nil {
				outputError(1, closeErr.Error(), result)
				return nil
			}
			outputWorkflow(result, outcome.Message+"\n"+renderLifecycleCloseoutFromResult(result, detectPlatform()))
			return nil
		}

		result := buildResumeDashboardResult()
		if staleSignals := detectStaleFocusSignals(store, loadStateAfterLifecycleOutcome().CurrentPhase); len(staleSignals) > 0 {
			result["stale_signals"] = staleSignals
		}
		result["resumed"] = true
		result["handoff_found"] = outcome.Handoff.HandoffID != ""
		result["handoff_path"] = handoffDocumentPath()
		result["handoff_removed"] = true
		result["handoff_id"] = outcome.Handoff.HandoffID
		result["receipt_id"] = outcome.Receipt.ReceiptID
		result["provenance"] = outcome.Provenance
		result["state_effect"] = outcome.StateEffect
		result["replay"] = outcome.Replay
		if outcome.Provenance == colony.RecoveryProvenanceReconstructed {
			result["state_recovered_from_handoff"] = true
		}
		if outcome.WorktreesCleaned > 0 {
			result["worktrees_preserved"] = map[string]interface{}{"cleaned": outcome.WorktreesCleaned, "preserved": 0}
		}
		closeLifecycleCommand(result, "picking the project back up",
			stringValue(result["resume_override_command"]), stringValue(result["resume_override_why"]))
		resumeSummary := "The validated recovery point was restored."
		if outcome.Replay {
			resumeSummary = "The existing resume receipt was replayed without applying the recovery twice."
		}
		if err := applyLifecycleCloseout(result, "resume", LifecycleCloseoutDetails{
			Summary: resumeSummary,
			Evidence: []colony.LifecycleEvidence{
				{ID: outcome.Handoff.HandoffID, Kind: "handoff", Source: handoffDocumentPath(), Summary: "Validated recovery handoff"},
				{ID: outcome.Receipt.ReceiptID, Kind: "receipt", Source: pauseHandoffDataPath, Summary: "Durable resume receipt"},
			},
		}); err != nil {
			outputError(1, err.Error(), result)
			return nil
		}
		visual := appendLifecycleCloseoutVisual(renderResumeVisual(result, "", true), result, detectPlatform())
		if outcome.Replay && outcome.Message != "" {
			visual = outcome.Message + "\n" + renderLifecycleCloseoutFromResult(result, detectPlatform())
		}
		outputWorkflow(result, visual)
		return nil
	},
}

// pauseColonyAt is the single mutating pause boundary. Every durable target is
// declared before intent is persisted, and the handoff ID names the same
// transaction on retry. A crash therefore resumes the existing journal rather
// than rebuilding partial state or emitting another handoff.
func pauseColonyAt(now time.Time) (pauseResumeLifecycleOutcome, error) {
	var outcome pauseResumeLifecycleOutcome
	err := withPlanningMutationSession(resolveAetherRootPath(), "pause", func(session *planningMutationSession) error {
		var err error
		outcome, err = pauseColonyInMutationSession(now, session)
		return err
	})
	return outcome, err
}

// Hold the same repository authority as build-start/native admission from the
// first facts read through the final transaction. A previously idle decision
// must never cross a newly reserved or released child.
func pauseColonyInMutationSession(now time.Time, mutation *planningMutationSession) (pauseResumeLifecycleOutcome, error) {
	for _, target := range []struct {
		root lifecycleTransactionRootKind
		path string
	}{
		{lifecycleTransactionRootData, "COLONY_STATE.json"},
		{lifecycleTransactionRootData, "session.json"},
		{lifecycleTransactionRootData, pauseHandoffDataPath},
		{lifecycleTransactionRootRepository, filepath.Join(".aether", "CONTEXT.md")},
		{lifecycleTransactionRootRepository, filepath.Join(".aether", "HANDOFF.md")},
	} {
		if _, _, err := mutation.ReadFile(target.root, target.path); err != nil {
			return pauseResumeLifecycleOutcome{}, err
		}
	}
	root := resolveAetherRootPath()
	facts, err := loadLifecycleFacts(root, store, now.UTC())
	if err != nil {
		return pauseResumeLifecycleOutcome{}, err
	}
	if facts.State.Source.Provenance != LifecycleFactConfirmed {
		return pauseResumeLifecycleOutcome{}, fmt.Errorf("colony state is %s: %s", facts.State.Source.Provenance, facts.State.Source.Diagnostic)
	}
	state := normalizeLegacyColonyState(facts.State.Value)
	if !resumeStateIsRunnable(state) {
		return pauseResumeLifecycleOutcome{}, fmt.Errorf("COLONY_STATE.json is not a runnable recovery source")
	}
	session := pauseResumeSessionFromFacts(facts, state, now)

	if state.Paused && state.PauseHandoff != nil {
		return replayPauseTransaction(*state.PauseHandoff)
	}

	repository, err := pauseRepositoryEvidence(root)
	if err != nil {
		return pauseResumeLifecycleOutcome{}, err
	}
	handoffID := pauseEpisodeHandoffID(state, session, repository)
	transactionID := "pause-" + handoffID
	config := pauseResumeTransactionConfig(transactionID, "pause")
	config.Session = mutation
	if lifecycleTransactionHasIntentOrReceipt(transactionID) {
		receipt, resumeErr := resumeLifecycleTransaction(config)
		if resumeErr != nil {
			return pauseResumeLifecycleOutcome{}, resumeErr
		}
		return loadPauseOutcome(receipt, true, pauseReplayMessage)
	}
	if err := discardUncommittedLifecycleStaging(transactionID); err != nil {
		return pauseResumeLifecycleOutcome{}, err
	}

	safeBoundary, attemptID, pending := pauseSafeBoundary(state)
	if pending {
		return pauseResumeLifecycleOutcome{}, pauseBoundaryPendingError{Boundary: safeBoundary, Attempt: attemptID}
	}

	handoff, err := buildPauseHandoff(facts, state, session, repository, now, handoffID, transactionID, safeBoundary, attemptID, colony.RecoveryProvenanceConfirmed)
	if err != nil {
		return pauseResumeLifecycleOutcome{}, err
	}
	handoffBytes, err := pauseResumeJSON(handoff)
	if err != nil {
		return pauseResumeLifecycleOutcome{}, fmt.Errorf("marshal pause handoff: %w", err)
	}
	reference := colony.PauseHandoffReference{
		ID: handoff.HandoffID, TransactionID: transactionID,
		Digest: lifecycleDigest(handoffBytes), Path: pauseHandoffDataPath,
	}
	if err := reference.Validate(); err != nil {
		return pauseResumeLifecycleOutcome{}, fmt.Errorf("validate pause handoff reference: %w", err)
	}

	pausedAt := now.UTC().Format(time.RFC3339)
	state.Paused = true
	state.PausedAt = &pausedAt
	state.PauseHandoff = &reference
	confirmed := colony.RecoveryProvenanceConfirmed
	state.RecoveryProvenance = &confirmed
	if state.SessionID == nil || strings.TrimSpace(*state.SessionID) == "" {
		id := session.SessionID
		state.SessionID = &id
	}
	state.Events = append(state.Events, pauseResumeLifecycleEvent(now, "pause", pauseBoundarySentence(safeBoundary)))

	session.LastCommand = "pause"
	session.LastCommandAt = pausedAt
	session.CurrentPhase = state.CurrentPhase
	session.CurrentMilestone = state.Milestone
	session.SuggestedNext = "aether resume"
	session.ContextCleared = true
	session.ResumedAt = nil
	session.ActiveTodos = mergeShelfTodos(session.ActiveTodos, currentOpenTasks(state))
	session.Summary = fmt.Sprintf("Paused once at %s; resume from %s.", safeBoundary, handoff.RestartPoint)
	session.PauseHandoff = &reference
	session.RecoveryProvenance = &confirmed
	session.BaselineCommit = repository.Head

	stateBytes, err := pauseResumeJSON(state)
	if err != nil {
		return pauseResumeLifecycleOutcome{}, fmt.Errorf("marshal paused state: %w", err)
	}
	sessionBytes, err := pauseResumeJSON(session)
	if err != nil {
		return pauseResumeLifecycleOutcome{}, fmt.Errorf("marshal paused session: %w", err)
	}
	contextText := renderContextSnapshot(state, session, "aether resume", session.Summary, "YES — one validated handoff is committed")
	humanHandoff := buildTransactionalHandoffDocument(now, state, session, handoff)

	tx, err := beginLifecycleTransaction(config)
	if err != nil {
		return pauseResumeLifecycleOutcome{}, err
	}
	for _, declaration := range []struct {
		root    lifecycleTransactionRootKind
		path    string
		content []byte
	}{
		{lifecycleTransactionRootData, "COLONY_STATE.json", stateBytes},
		{lifecycleTransactionRootData, "session.json", sessionBytes},
		{lifecycleTransactionRootData, pauseHandoffDataPath, handoffBytes},
		{lifecycleTransactionRootRepository, filepath.Join(".aether", "CONTEXT.md"), []byte(contextText)},
		{lifecycleTransactionRootRepository, filepath.Join(".aether", "HANDOFF.md"), []byte(humanHandoff)},
	} {
		if err := tx.DeclareWrite(declaration.root, declaration.path, declaration.content); err != nil {
			return pauseResumeLifecycleOutcome{}, err
		}
	}
	receipt, err := tx.Commit()
	if err != nil {
		return pauseResumeLifecycleOutcome{}, err
	}
	return loadPauseOutcome(receipt, false, "")
}

// resumeColonyAt validates every independent copy of the handoff identity
// before it declares a write. Conflicting and unknown paths return a rendered
// stop with StateEffectNone; only confirmed or explicitly reconstructed facts
// reach the transaction coordinator.
func resumeColonyAt(now time.Time) (pauseResumeLifecycleOutcome, error) {
	root := resolveAetherRootPath()
	facts, err := loadLifecycleFacts(root, store, now.UTC())
	if err != nil {
		return pauseResumeLifecycleOutcome{}, err
	}
	state := normalizeLegacyColonyState(facts.State.Value)
	session := pauseResumeSessionFromFacts(facts, state, now)

	// A pause crash can leave durable intent before its state target lands. A
	// resume request first completes that already-authorized pause transaction;
	// it never invents a second handoff to get past the interruption.
	if state.PauseHandoff == nil {
		if repository, repoErr := pauseRepositoryEvidence(root); repoErr == nil {
			candidateID := pauseEpisodeHandoffID(state, session, repository)
			candidateTx := "pause-" + candidateID
			if lifecycleTransactionHasIntentOrReceipt(candidateTx) {
				if _, resumeErr := resumeLifecycleTransaction(pauseResumeTransactionConfig(candidateTx, "pause")); resumeErr != nil {
					return pauseResumeLifecycleOutcome{}, resumeErr
				}
				facts, err = loadLifecycleFacts(root, store, now.UTC())
				if err != nil {
					return pauseResumeLifecycleOutcome{}, err
				}
				state = normalizeLegacyColonyState(facts.State.Value)
				session = pauseResumeSessionFromFacts(facts, state, now)
			}
		}
	}

	if state.PauseHandoff != nil && !state.Paused {
		resumeTxID := resumeTransactionID(state.PauseHandoff.ID)
		if lifecycleTransactionHasIntentOrReceipt(resumeTxID) {
			receipt, resumeErr := resumeLifecycleTransaction(pauseResumeTransactionConfig(resumeTxID, "resume"))
			if resumeErr != nil {
				return pauseResumeConflictOutcome("Recovery evidence conflicts with the previously committed resume receipt. Preserve the transaction journal, inspect `aether status`, choose which named evidence is authoritative, then run `aether resume`."), nil
			}
			handoff, loadErr := loadValidatedPauseHandoff(*state.PauseHandoff)
			if loadErr != nil {
				return pauseResumeConflictOutcome("Recovery evidence conflicts after resume: the retained handoff no longer matches its reference. Inspect `aether status` before deciding whether to repair the handoff evidence."), nil
			}
			return pauseResumeLifecycleOutcome{
				Handoff: handoff, Receipt: receipt, Provenance: handoff.Provenance,
				StateEffect: receipt.StateEffect, Replay: true,
				Message: "Already resumed; the existing validated recovery receipt was retained.",
			}, nil
		}
	}
	// Finish authorized pause/resume transactions and validate retained handoff
	// evidence before native advice. An unpaused state target may be only the
	// first committed prefix of an interrupted resume transaction.
	if state.PauseHandoff == nil {
		if recovery := buildCodexNativeRecovery(state); recovery != nil && (!state.Paused || recovery.pendingBoundary()) {
			return pauseResumeLifecycleOutcome{NativeRecovery: recovery, StateEffect: colony.LifecycleStateEffectNone, Provenance: colony.RecoveryProvenanceConfirmed}, nil
		}
	}

	var handoff colony.PauseHandoff
	provenance := colony.RecoveryProvenanceConfirmed
	reconstructed := false
	if state.PauseHandoff != nil {
		if facts.Session.Source.Provenance != LifecycleFactConfirmed || session.PauseHandoff == nil {
			return pauseResumeConflictOutcome("Recovery evidence conflicts: state names a handoff but session does not. Run `aether status`, decide whether state or session evidence is authoritative. If the handoff is genuinely stale because the runtime kept writing after the pause, retire it with `aether state-mutate --field pause_handoff --value null`, then `aether resume` will reconstruct a recovery point and label its provenance reconstructed."), nil
		}
		if !pauseHandoffReferencesEqual(*state.PauseHandoff, *session.PauseHandoff) {
			return pauseResumeConflictOutcome("Recovery evidence conflicts: state and session name different handoff evidence. Run `aether status`, decide which handoff is authoritative. If the handoff is genuinely stale because the runtime kept writing after the pause, retire it with `aether state-mutate --field pause_handoff --value null`, then `aether resume` will reconstruct a recovery point and label its provenance reconstructed."), nil
		}
		handoff, err = loadValidatedPauseHandoff(*state.PauseHandoff)
		if err != nil {
			return pauseResumeConflictOutcome(fmt.Sprintf("Recovery evidence conflicts: the referenced handoff is invalid (%v). Run `aether status`, preserve the named evidence, and choose the authoritative recovery point.", err)), nil
		}
		pauseReceipt, receiptErr := resumeLifecycleTransaction(pauseResumeTransactionConfig(handoff.Transaction.ID, "pause"))
		if receiptErr != nil || handoff.Receipt == nil || pauseReceipt.ReceiptID != handoff.Receipt.ID {
			return pauseResumeConflictOutcome("Recovery evidence conflicts: the pause transaction receipt does not validate against state, session, and handoff bytes. Run `aether status` to inspect what changed. If the handoff is genuinely stale because the runtime kept writing after the pause, retire it with `aether state-mutate --field pause_handoff --value null`, then `aether resume` will reconstruct a recovery point and label its provenance reconstructed."), nil
		}
		currentRepository, repoErr := pauseRepositoryEvidence(root)
		if repoErr != nil {
			return pauseResumeUnknownOutcome(fmt.Sprintf("Recovery evidence is unknown: repository evidence could not be read (%v). Run `aether status` and retry when repository evidence is available.", repoErr)), nil
		}
		if currentRepository.Head != handoff.Repository.Head || currentRepository.DirtyDigest != handoff.Repository.DirtyDigest {
			return pauseResumeConflictOutcome("Recovery evidence conflicts: repository HEAD or working bytes changed after the handoff. Run `aether status`, decide whether to keep those changes. If the handoff is genuinely stale because the runtime kept writing after the pause, retire it with `aether state-mutate --field pause_handoff --value null`, then `aether resume` will reconstruct a recovery point and label its provenance reconstructed."), nil
		}
		if !reflectPauseWorktreesEqual(handoff.Worktrees, pauseWorktreeEvidence(root, state.Worktrees)) {
			return pauseResumeConflictOutcome("Recovery evidence conflicts: worktree evidence changed after the handoff. Run `aether status`, inspect the named worktrees, and choose which work is authoritative before resuming."), nil
		}
		if recovery := buildCodexNativeRecovery(state); recovery != nil && recovery.pendingBoundary() {
			return pauseResumeLifecycleOutcome{NativeRecovery: recovery, StateEffect: colony.LifecycleStateEffectNone, Provenance: colony.RecoveryProvenanceConfirmed}, nil
		}
		if state.State == colony.StateEXECUTING {
			if _, attempt, ok := loadRelevantBuildAttemptReadOnly(state); ok && buildAttemptProcessAlive(attempt) {
				return pauseResumeConflictOutcome("Recovery evidence conflicts: the paused attempt still has a live worker process. Run `aether status` and wait for the worker's safe boundary before resuming."), nil
			}
		}
	} else {
		if !resumeStateIsRunnable(state) {
			if resumeNoHandoff {
				return pauseResumeUnknownOutcome("Recovery evidence is unknown: runtime state is not runnable and HANDOFF.md reconstruction is disabled. Run `aether status` and repair COLONY_STATE.json."), nil
			}
			handoffText, readErr := readHandoffDocument()
			if readErr != nil {
				return pauseResumeUnknownOutcome("Recovery evidence is unknown: neither runnable state nor a readable handoff exists. Run `aether status` and restore durable recovery evidence."), nil
			}
			restored, restoreErr := restoreStateFromHandoff(string(handoffText))
			if restoreErr != nil {
				return pauseResumeUnknownOutcome(fmt.Sprintf("Recovery evidence is unknown: HANDOFF.md cannot reconstruct a runnable point (%v). Run `aether status` and repair the named evidence.", restoreErr)), nil
			}
			// A syntactically readable but non-runnable state is still evidence.
			// Never let a handoff silently replace its known colony identity: that
			// would turn a recovery attempt into cross-colony data loss.
			if state.Goal != nil && strings.TrimSpace(*state.Goal) != "" && restored.Goal != nil && strings.TrimSpace(*restored.Goal) != "" && !goalsMatch(*state.Goal, *restored.Goal) {
				return pauseResumeConflictOutcome("Recovery evidence conflicts: HANDOFF.md does not match current COLONY_STATE.json goal and appears to belong to a different colony. Run `aether status`, preserve both records, and choose the authoritative recovery point before `aether resume`."), nil
			}
			state = restored
			state.Paused = true
			session = pauseResumeSessionFromFacts(facts, state, now)
		} else {
			if conflict := pauseResumeStateSessionConflict(state, facts.Session); conflict != "" {
				return pauseResumeConflictOutcome(conflict), nil
			}
			if state.State == colony.StateEXECUTING {
				if _, attempt, ok := loadRelevantBuildAttemptReadOnly(state); ok && buildAttemptProcessAlive(attempt) {
					return pauseResumeConflictOutcome("Recovery evidence conflicts: durable state still names a live build attempt. Run `aether status` and wait for its safe boundary instead of redispatching finished work."), nil
				}
			}
		}
		provenance = colony.RecoveryProvenanceReconstructed
		reconstructed = true
		repository, repoErr := pauseRepositoryEvidence(root)
		if repoErr != nil {
			return pauseResumeUnknownOutcome(fmt.Sprintf("Recovery evidence is unknown: repository evidence could not be reconstructed (%v).", repoErr)), nil
		}
		handoffID := "reconstructed-" + pauseEpisodeHandoffID(state, session, repository)
		resumeTxID := resumeTransactionID(handoffID)
		boundary := "reconstructed_state_boundary"
		if facts.State.Source.Provenance != LifecycleFactConfirmed {
			boundary = "reconstructed_legacy_boundary"
		}
		handoff, err = buildPauseHandoff(facts, state, session, repository, now, handoffID, resumeTxID, boundary, "", provenance)
		if err != nil {
			return pauseResumeLifecycleOutcome{}, err
		}
	}

	resumeTxID := resumeTransactionID(handoff.HandoffID)
	config := pauseResumeTransactionConfig(resumeTxID, "resume")
	if lifecycleTransactionHasIntentOrReceipt(resumeTxID) {
		receipt, resumeErr := resumeLifecycleTransaction(config)
		if resumeErr != nil {
			return pauseResumeLifecycleOutcome{}, resumeErr
		}
		loaded, loadErr := loadValidatedPauseHandoffReferenceFromState()
		if loadErr == nil {
			handoff = loaded
		}
		return pauseResumeLifecycleOutcome{Handoff: handoff, Receipt: receipt, Provenance: provenance, StateEffect: receipt.StateEffect, Replay: true}, nil
	}
	if err := discardUncommittedLifecycleStaging(resumeTxID); err != nil {
		return pauseResumeLifecycleOutcome{}, err
	}

	if reconstructed {
		handoff.Transaction = pauseResumeTransactionReference(resumeTxID)
		handoff.Receipt = &colony.LifecycleReceiptReference{ID: resumeTxID + "-receipt"}
	}
	handoffBytes, err := pauseResumeJSON(handoff)
	if err != nil {
		return pauseResumeLifecycleOutcome{}, fmt.Errorf("marshal resume handoff: %w", err)
	}
	reference := state.PauseHandoff
	if reconstructed {
		reference = &colony.PauseHandoffReference{
			ID: handoff.HandoffID, TransactionID: handoff.Transaction.ID,
			Digest: lifecycleDigest(handoffBytes), Path: pauseHandoffDataPath,
		}
	}
	if reference == nil {
		return pauseResumeLifecycleOutcome{}, fmt.Errorf("resume handoff reference is missing")
	}

	staleSession := pauseResumeSessionIsStale(session, now, handoff.Repository.Head)
	// A stale run must not leave its old worker roster attached to the next
	// recovery episode.  Keep the historical evidence, but make that move part
	// of this resume transaction: calling rotateSpawnTree after commit used to
	// make the restoration only half durable when the process stopped between
	// the state write and the archive copy.
	spawnTreeArchive, spawnRunsArchive, spawnTreeBytes, spawnRunsBytes := resumeStaleSpawnArtifacts(resumeTxID, staleSession)
	worktreesCleaned := 0
	if staleSession {
		remaining := make([]colony.WorktreeEntry, 0, len(state.Worktrees))
		for _, entry := range state.Worktrees {
			safety := worktreeDestructionSafety(root, entry)
			if safety.Safe && safety.Reason == "worktree path no longer exists on disk" {
				worktreesCleaned++
				continue
			}
			remaining = append(remaining, entry)
		}
		state.Worktrees = remaining
	}
	state.Paused = false
	state.PausedAt = nil
	if state.State == colony.StateEXECUTING {
		state.State = colony.StateREADY
		state.BuildStartedAt = nil
	}
	if staleSession {
		state.BuildStartedAt = nil
	}
	newRunID := "resume-" + compactLifecycleID(handoff.HandoffID)
	state.RunID = &newRunID
	state.PauseHandoff = reference
	state.RecoveryProvenance = &provenance
	state.Events = append(state.Events, pauseResumeLifecycleEvent(now, "resume", resumeProvenanceSentence(provenance)))

	resumedAt := now.UTC().Format(time.RFC3339)
	session.LastCommand = "resume"
	session.LastCommandAt = resumedAt
	session.CurrentPhase = state.CurrentPhase
	session.CurrentMilestone = state.Milestone
	session.SuggestedNext = nextCommandFromState(state)
	session.ContextCleared = false
	session.ResumedAt = &resumedAt
	session.ActiveTodos = mergeShelfTodos(session.ActiveTodos, currentOpenTasks(state))
	session.Summary = fmt.Sprintf("Recovery point %s resumed with %s provenance.", handoff.HandoffID, provenance)
	session.PauseHandoff = reference
	session.RecoveryProvenance = &provenance
	session.BaselineCommit = handoff.Repository.Head

	stateBytes, err := pauseResumeJSON(state)
	if err != nil {
		return pauseResumeLifecycleOutcome{}, fmt.Errorf("marshal resumed state: %w", err)
	}
	sessionBytes, err := pauseResumeJSON(session)
	if err != nil {
		return pauseResumeLifecycleOutcome{}, fmt.Errorf("marshal resumed session: %w", err)
	}
	contextText := renderContextSnapshot(state, session, session.SuggestedNext, session.Summary, "NO — recovery context is active")

	tx, err := beginLifecycleTransaction(config)
	if err != nil {
		return pauseResumeLifecycleOutcome{}, err
	}
	if err := tx.DeclareWrite(lifecycleTransactionRootData, "COLONY_STATE.json", stateBytes); err != nil {
		return pauseResumeLifecycleOutcome{}, err
	}
	if err := tx.DeclareWrite(lifecycleTransactionRootData, "session.json", sessionBytes); err != nil {
		return pauseResumeLifecycleOutcome{}, err
	}
	if reconstructed {
		if err := tx.DeclareWrite(lifecycleTransactionRootData, pauseHandoffDataPath, handoffBytes); err != nil {
			return pauseResumeLifecycleOutcome{}, err
		}
	}
	if err := tx.DeclareWrite(lifecycleTransactionRootRepository, filepath.Join(".aether", "CONTEXT.md"), []byte(contextText)); err != nil {
		return pauseResumeLifecycleOutcome{}, err
	}
	// Pair the removal of the pre-resume hand-off note with a fresh, minimal
	// one written in the same transaction (not a plain DeclareRemoval), so a
	// "return to work then archive" sequence never sees an empty slot: entomb
	// requires .aether/HANDOFF.md as a required tombstone_input source. The
	// transaction coordinator refuses two declarations for the same target
	// path, so this write is the single staged replacement — old stale
	// content is gone, fresh content is committed, all inside this
	// transaction's existing staging discipline.
	freshHandoff := buildHandoffDocument(now.UTC(), state, session, session.SuggestedNext)
	if err := tx.DeclareWrite(lifecycleTransactionRootRepository, filepath.Join(".aether", "HANDOFF.md"), []byte(freshHandoff)); err != nil {
		return pauseResumeLifecycleOutcome{}, err
	}
	if staleSession {
		if len(spawnTreeBytes) > 0 {
			if err := tx.DeclareWrite(lifecycleTransactionRootData, spawnTreeArchive, spawnTreeBytes); err != nil {
				return pauseResumeLifecycleOutcome{}, err
			}
		}
		if err := tx.DeclareWrite(lifecycleTransactionRootData, "spawn-tree.txt", []byte{}); err != nil {
			return pauseResumeLifecycleOutcome{}, err
		}
		if len(spawnRunsBytes) > 0 {
			if err := tx.DeclareWrite(lifecycleTransactionRootData, spawnRunsArchive, spawnRunsBytes); err != nil {
				return pauseResumeLifecycleOutcome{}, err
			}
		}
		if err := tx.DeclareWrite(lifecycleTransactionRootData, "spawn-runs.json", []byte{}); err != nil {
			return pauseResumeLifecycleOutcome{}, err
		}
	}
	receipt, err := tx.Commit()
	if err != nil {
		return pauseResumeLifecycleOutcome{}, err
	}
	if staleSession && tracer != nil {
		_ = tracer.LogIntervention(newRunID, "resume.spawn-clear", "resume", map[string]interface{}{
			"reason": "stale_session",
		})
	}
	outcome := pauseResumeLifecycleOutcome{
		Handoff: handoff, Receipt: receipt, Provenance: provenance,
		StateEffect: receipt.StateEffect, WorktreesCleaned: worktreesCleaned,
	}
	if worktreesCleaned > 0 {
		outcome.Message = fmt.Sprintf("%d stale worker workspace record(s) were removed after the missing paths were proven empty.", worktreesCleaned)
	}
	return outcome, nil
}

// resumeStaleSpawnArtifacts snapshots stale roster data before the caller
// declares its transaction targets.  The transaction ID provides a stable
// archive identity, so replay resumes the same archive instead of creating a
// timestamp-dependent second copy.
func resumeStaleSpawnArtifacts(transactionID string, stale bool) (treeArchive, runsArchive string, treeBytes, runsBytes []byte) {
	if !stale || store == nil {
		return "", "", nil, nil
	}
	treeBytes, _ = os.ReadFile(filepath.Join(store.BasePath(), "spawn-tree.txt"))
	runsBytes, _ = os.ReadFile(filepath.Join(store.BasePath(), "spawn-runs.json"))
	archiveID := compactLifecycleID(transactionID)
	if len(strings.TrimSpace(string(treeBytes))) > 0 {
		treeArchive = filepath.Join("spawn-tree-archive", "spawn-tree."+archiveID+".txt")
	}
	if len(strings.TrimSpace(string(runsBytes))) > 0 {
		runsArchive = filepath.Join("spawn-tree-archive", "spawn-runs."+archiveID+".json")
	}
	return treeArchive, runsArchive, treeBytes, runsBytes
}

func replayPauseTransaction(reference colony.PauseHandoffReference) (pauseResumeLifecycleOutcome, error) {
	if err := reference.Validate(); err != nil {
		return pauseResumeLifecycleOutcome{}, fmt.Errorf("paused state has an invalid handoff reference: %w", err)
	}
	receipt, err := resumeLifecycleTransaction(pauseResumeTransactionConfig(reference.TransactionID, "pause"))
	if err != nil {
		return pauseResumeLifecycleOutcome{}, err
	}
	return loadPauseOutcome(receipt, true, pauseReplayMessage)
}

func loadPauseOutcome(receipt colony.LifecycleReceipt, replay bool, message string) (pauseResumeLifecycleOutcome, error) {
	handoff, err := loadValidatedPauseHandoffReferenceFromState()
	if err != nil {
		return pauseResumeLifecycleOutcome{}, err
	}
	if handoff.Receipt == nil || handoff.Receipt.ID != receipt.ReceiptID || handoff.Transaction.ID != receipt.Transaction.ID {
		return pauseResumeLifecycleOutcome{}, fmt.Errorf("pause handoff and transaction receipt do not cross-reference one another")
	}
	return pauseResumeLifecycleOutcome{
		Handoff: handoff, Receipt: receipt, Provenance: colony.RecoveryProvenanceConfirmed,
		StateEffect: receipt.StateEffect, Replay: replay, Message: message,
	}, nil
}

func loadValidatedPauseHandoffReferenceFromState() (colony.PauseHandoff, error) {
	var state colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
		return colony.PauseHandoff{}, err
	}
	if state.PauseHandoff == nil {
		return colony.PauseHandoff{}, fmt.Errorf("state does not reference a pause handoff")
	}
	return loadValidatedPauseHandoff(*state.PauseHandoff)
}

func loadValidatedPauseHandoff(reference colony.PauseHandoffReference) (colony.PauseHandoff, error) {
	if err := reference.Validate(); err != nil {
		return colony.PauseHandoff{}, err
	}
	relative := strings.TrimSpace(reference.Path)
	if relative == "" {
		relative = pauseHandoffDataPath
	}
	clean := filepath.Clean(relative)
	if filepath.IsAbs(relative) || clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) || clean != relative {
		return colony.PauseHandoff{}, fmt.Errorf("handoff path %q is not a contained data path", relative)
	}
	content, err := os.ReadFile(filepath.Join(store.BasePath(), clean))
	if err != nil {
		return colony.PauseHandoff{}, err
	}
	if lifecycleDigest(content) != reference.Digest {
		return colony.PauseHandoff{}, fmt.Errorf("handoff digest does not match state/session reference")
	}
	var handoff colony.PauseHandoff
	if err := json.Unmarshal(content, &handoff); err != nil {
		return colony.PauseHandoff{}, err
	}
	if err := handoff.Validate(); err != nil {
		return colony.PauseHandoff{}, err
	}
	if handoff.HandoffID != reference.ID || handoff.Transaction.ID != reference.TransactionID {
		return colony.PauseHandoff{}, fmt.Errorf("handoff identity conflicts with its reference")
	}
	return handoff, nil
}

func pauseResumeConflictOutcome(message string) pauseResumeLifecycleOutcome {
	return pauseResumeLifecycleOutcome{
		Provenance:  colony.RecoveryProvenanceConflicting,
		StateEffect: colony.LifecycleStateEffectNone,
		Message:     message,
	}
}

func pauseResumeUnknownOutcome(message string) pauseResumeLifecycleOutcome {
	return pauseResumeLifecycleOutcome{
		Provenance:  colony.RecoveryProvenanceUnknown,
		StateEffect: colony.LifecycleStateEffectNone,
		Message:     message,
	}
}

func pauseResumeTransactionConfig(transactionID, command string) lifecycleTransactionConfig {
	return lifecycleTransactionConfig{
		TransactionID: transactionID,
		Command:       command,
		Allowlist: lifecycleTransactionAllowlist{
			RepositoryRoot:    resolveAetherRootPath(),
			LifecycleDataRoot: store.BasePath(),
		},
		Fault: pauseResumeLifecycleFault,
	}
}

func lifecycleTransactionHasIntentOrReceipt(transactionID string) bool {
	journal := filepath.Join(store.BasePath(), "transactions", transactionID)
	for _, name := range []string{"intent.json", "receipt.json"} {
		if info, err := os.Lstat(filepath.Join(journal, name)); err == nil && info.Mode().IsRegular() {
			return true
		}
	}
	return false
}

// discardUncommittedLifecycleStaging removes only transaction-owned scratch
// directories for an ID that never reached durable intent. Once intent exists,
// recovery must use the coordinator journal and this helper refuses to act.
func discardUncommittedLifecycleStaging(transactionID string) error {
	if !lifecycleTransactionIDPattern.MatchString(transactionID) {
		return fmt.Errorf("invalid lifecycle transaction id %q", transactionID)
	}
	journal := filepath.Join(store.BasePath(), "transactions", transactionID)
	if _, err := os.Lstat(filepath.Join(journal, "intent.json")); err == nil {
		return fmt.Errorf("lifecycle transaction %s has durable intent; resume it instead of discarding staging", transactionID)
	} else if !os.IsNotExist(err) {
		return err
	}
	for _, root := range []string{resolveAetherRootPath(), store.BasePath()} {
		owned := filepath.Join(root, lifecycleTransactionDirectory, transactionID)
		if !pathIsWithin(filepath.Join(root, lifecycleTransactionDirectory), owned) {
			return fmt.Errorf("transaction staging path %q escaped its owned root", owned)
		}
		if err := os.RemoveAll(owned); err != nil {
			return fmt.Errorf("discard uncommitted transaction staging %q: %w", owned, err)
		}
	}
	return nil
}

func pauseResumeJSON(value interface{}) ([]byte, error) {
	content, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(content, '\n'), nil
}

func pauseResumeSessionFromFacts(facts LifecycleFacts, state colony.ColonyState, now time.Time) colony.SessionFile {
	if facts.Session.Source.Provenance == LifecycleFactConfirmed {
		return facts.Session.Value
	}
	goal := colonyStateGoalText(state)
	sessionID := "session-" + compactLifecycleID(lifecycleDigest([]byte(strings.Join([]string{
		goal, fmt.Sprint(state.CurrentPhase), pauseResumeRunID(state), resolveAetherRootPath(),
	}, "\x00"))))
	startedAt := now.UTC().Format(time.RFC3339)
	if state.InitializedAt != nil {
		startedAt = state.InitializedAt.UTC().Format(time.RFC3339)
	}
	return colony.SessionFile{
		SessionID: sessionID, StartedAt: startedAt, ColonyGoal: goal,
		ColonyMode: state.EffectiveColonyMode(), CurrentPhase: state.CurrentPhase,
		CurrentMilestone: state.Milestone, SuggestedNext: nextCommandFromState(state),
		ActiveTodos: currentOpenTasks(state), Summary: "Recovery session reconstructed from durable state.",
	}
}

func pauseResumeRunID(state colony.ColonyState) string {
	if state.RunID != nil && strings.TrimSpace(*state.RunID) != "" {
		return strings.TrimSpace(*state.RunID)
	}
	if state.SessionID != nil && strings.TrimSpace(*state.SessionID) != "" {
		return strings.TrimSpace(*state.SessionID)
	}
	if state.AcceptedCharter != nil && strings.TrimSpace(state.AcceptedCharter.EpisodeID) != "" {
		return strings.TrimSpace(state.AcceptedCharter.EpisodeID)
	}
	return "legacy-episode"
}

func pauseEpisodeHandoffID(state colony.ColonyState, session colony.SessionFile, repository colony.LifecycleRepositoryEvidence) string {
	material := strings.Join([]string{
		"pause-handoff-v1", resolveAetherRootPath(), strings.TrimSpace(session.SessionID),
		pauseResumeRunID(state), fmt.Sprint(state.CurrentPhase), colonyStateGoalText(state),
		repository.Head, repository.DirtyDigest,
	}, "\x00")
	return "handoff-" + compactLifecycleID(lifecycleDigest([]byte(material)))
}

func compactLifecycleID(value string) string {
	value = strings.TrimSpace(value)
	value = strings.TrimPrefix(value, "sha256:")
	var b strings.Builder
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '.' || r == '_' || r == '-' {
			b.WriteRune(r)
		}
	}
	clean := b.String()
	if len(clean) > 24 {
		clean = clean[:24]
	}
	if clean == "" {
		return "unknown"
	}
	return clean
}

func resumeTransactionID(handoffID string) string {
	return "resume-" + compactLifecycleID(handoffID)
}

func pauseResumeTransactionReference(transactionID string) colony.LifecycleTransactionReference {
	return colony.LifecycleTransactionReference{
		ID: transactionID, Stage: colony.TransactionStageVerified,
		JournalPath: filepath.Join(store.BasePath(), "transactions", transactionID),
	}
}

func pauseResumeStateSessionConflict(state colony.ColonyState, sessionFact LifecycleFact[colony.SessionFile]) string {
	if sessionFact.Source.Provenance != LifecycleFactConfirmed {
		return ""
	}
	session := sessionFact.Value
	if state.Goal != nil && strings.TrimSpace(session.ColonyGoal) != "" && !goalsMatch(*state.Goal, session.ColonyGoal) {
		return "Recovery evidence conflicts: state and session name different goals. Run `aether status`, decide which episode is authoritative, then run `aether resume`."
	}
	if session.CurrentPhase > 0 && state.CurrentPhase > 0 && session.CurrentPhase != state.CurrentPhase {
		return "Recovery evidence conflicts: state and session name different current phases. Run `aether status`, choose the proven phase boundary, then run `aether resume`."
	}
	if state.SessionID != nil && strings.TrimSpace(*state.SessionID) != "" && strings.TrimSpace(session.SessionID) != "" && strings.TrimSpace(*state.SessionID) != strings.TrimSpace(session.SessionID) {
		return "Recovery evidence conflicts: state and session name different session IDs. Run `aether status`, preserve both records, and choose the authoritative recovery episode."
	}
	return ""
}

// loadRelevantBuildAttemptReadOnly mirrors the selection rule used by build,
// but uses direct file reads so determining a pause boundary cannot create
// storage lock files before transaction intent exists.
func loadRelevantBuildAttemptReadOnly(state colony.ColonyState) (string, buildAttemptRecord, bool) {
	if store == nil {
		return "", buildAttemptRecord{}, false
	}
	phaseIDs := make([]int, 0, len(state.Plan.Phases)+1)
	seen := make(map[int]bool)
	if state.CurrentPhase > 0 {
		phaseIDs = append(phaseIDs, state.CurrentPhase)
		seen[state.CurrentPhase] = true
	}
	for index := len(state.Plan.Phases) - 1; index >= 0; index-- {
		phaseID := state.Plan.Phases[index].ID
		if phaseID > 0 && !seen[phaseID] {
			phaseIDs = append(phaseIDs, phaseID)
			seen[phaseID] = true
		}
	}
	var selectedRel string
	var selected buildAttemptRecord
	found := false
	for _, phaseID := range phaseIDs {
		pointerRel := latestBuildAttemptPointerPath(phaseID)
		pointerBytes, err := os.ReadFile(filepath.Join(store.BasePath(), filepath.FromSlash(pointerRel)))
		if err != nil {
			continue
		}
		var pointer latestBuildAttemptPointer
		if json.Unmarshal(pointerBytes, &pointer) != nil || strings.TrimSpace(pointer.AttemptID) == "" {
			continue
		}
		attemptRel := strings.TrimPrefix(filepath.ToSlash(strings.TrimSpace(pointer.Path)), ".aether/data/")
		if attemptRel == "" {
			continue
		}
		attemptBytes, err := os.ReadFile(filepath.Join(store.BasePath(), filepath.FromSlash(attemptRel)))
		if err != nil {
			continue
		}
		var record buildAttemptRecord
		if json.Unmarshal(attemptBytes, &record) != nil || record.ID != pointer.AttemptID || record.Phase != phaseID {
			continue
		}
		if !found || record.UpdatedAt > selected.UpdatedAt {
			selectedRel, selected, found = attemptRel, record, true
		}
	}
	return selectedRel, selected, found
}

// lifecycleEventSentence is the message field of a stored lifecycle event
// (`timestamp|event_type|source|message`, per nextActionEventSentence's own
// documented contract in cmd/next_action.go). That field is what an owner
// reads on the what-next card, and nextActionEventSentence trusts -- by
// design, not by accident -- that it is already a finished sentence rather
// than a raw internal token. This named type exists so a plain string
// variable or a typed enum value can never be assigned into that field
// directly: only pauseBoundarySentence and resumeProvenanceSentence, the two
// constructors below, may produce a lifecycleEventSentence value.
// TestLifecycleEventSentenceTypeCannotBeBypassed (cmd/classic_voice_event_test.go)
// refuses any other conversion into this type anywhere in this package.
type lifecycleEventSentence string

// pauseBoundarySentence turns one of pauseSafeBoundary's seven internal
// boundary values into a sentence an owner can read, describing what the
// pause stopped at in ordinary words. It never echoes the raw boundary
// value, including for a boundary this project does not yet name.
func pauseBoundarySentence(boundary string) lifecycleEventSentence {
	switch boundary {
	case "worker_completion_boundary":
		return lifecycleEventSentence("A helper had just finished its work when this was paused.")
	case "interrupted_attempt_boundary":
		return lifecycleEventSentence("Work was interrupted partway through, and the in-progress attempt was preserved.")
	case "interrupted_execution_boundary":
		return lifecycleEventSentence("Work was interrupted partway through building this phase.")
	case "post_build_verification_boundary":
		return lifecycleEventSentence("The phase was built and is waiting to be checked.")
	case "completed_episode_boundary":
		return lifecycleEventSentence("The phase had just finished when this was paused.")
	case "idle_command_boundary":
		return lifecycleEventSentence("Nothing was running when this was paused.")
	case "between_commands_boundary":
		return lifecycleEventSentence("This was paused between one command finishing and the next.")
	default:
		return lifecycleEventSentence("This was paused at a point this project cannot describe more precisely.")
	}
}

// resumeProvenanceSentence turns one of the four declared colony.RecoveryProvenance
// values into a sentence an owner can read, keeping the distinction the type
// exists to carry: a verified recovery point, a rebuilt one, one where the
// evidence disagreed, and one where the evidence could not be read. It never
// echoes the raw provenance value, including for a value this project does
// not yet name.
func resumeProvenanceSentence(provenance colony.RecoveryProvenance) lifecycleEventSentence {
	switch provenance {
	case colony.RecoveryProvenanceConfirmed:
		return lifecycleEventSentence("Resumed from a verified recovery point.")
	case colony.RecoveryProvenanceReconstructed:
		return lifecycleEventSentence("Resumed from a recovery point rebuilt from saved evidence.")
	case colony.RecoveryProvenanceConflicting:
		return lifecycleEventSentence("Resume found evidence that disagreed with itself and stopped rather than guess.")
	case colony.RecoveryProvenanceUnknown:
		return lifecycleEventSentence("Resume could not read enough evidence to confirm a recovery point.")
	default:
		return lifecycleEventSentence("This was resumed at a point this project cannot describe more precisely.")
	}
}

func pauseSafeBoundary(state colony.ColonyState) (boundary, attemptID string, pending bool) {
	if _, attempt, ok := loadRelevantBuildAttemptReadOnly(state); ok {
		attemptID = strings.TrimSpace(attempt.ID)
		for _, worker := range attempt.WorkerRuns {
			if worker.Native != nil && !codexNativeWorkerIsTerminal(worker) {
				return "worker_completion_boundary", attemptID, true
			}
		}
		if state.State == colony.StateEXECUTING && buildAttemptProcessAlive(attempt) {
			return "worker_completion_boundary", attemptID, true
		}
		if state.State == colony.StateEXECUTING {
			return "interrupted_attempt_boundary", attemptID, false
		}
	}
	switch state.State {
	case colony.StateEXECUTING:
		return "interrupted_execution_boundary", attemptID, false
	case colony.StateBUILT:
		return "post_build_verification_boundary", attemptID, false
	case colony.StateCOMPLETED:
		return "completed_episode_boundary", attemptID, false
	case colony.StateIDLE:
		return "idle_command_boundary", attemptID, false
	default:
		return "between_commands_boundary", attemptID, false
	}
}

func buildPauseHandoff(
	facts LifecycleFacts,
	state colony.ColonyState,
	session colony.SessionFile,
	repository colony.LifecycleRepositoryEvidence,
	now time.Time,
	handoffID, transactionID, safeBoundary, attemptID string,
	provenance colony.RecoveryProvenance,
) (colony.PauseHandoff, error) {
	contextBytes, contextErr := os.ReadFile(contextDocumentPath())
	contextDigest := lifecycleTransactionMissingDigest
	if contextErr == nil {
		contextDigest = lifecycleDigest(contextBytes)
	} else if !os.IsNotExist(contextErr) {
		return colony.PauseHandoff{}, fmt.Errorf("read context evidence: %w", contextErr)
	}

	evidence := pauseHandoffEvidence(facts, repository, contextDigest)
	transaction := pauseResumeTransactionReference(transactionID)
	receipt := &colony.LifecycleReceiptReference{ID: transactionID + "-receipt"}
	restartPoint := pauseRestartPoint(state)
	handoff := colony.PauseHandoff{
		SchemaVersion: colony.LifecycleSchemaVersion, HandoffID: handoffID,
		Command: "pause", OutcomeKind: colony.OutcomeKindPaused, ProjectionRevision: "199-13",
		PhaseID: state.CurrentPhase, TaskIDs: pauseTaskIDs(state), AttemptID: attemptID,
		RunID: pauseResumeRunID(state), SafeBoundary: safeBoundary, RestartPoint: restartPoint,
		ContextDigest: contextDigest, Repository: repository,
		Worktrees: pauseWorktreeEvidence(resolveAetherRootPath(), state.Worktrees),
		Lineage:   pauseWorkerLineage(facts.Actors.Value), Signals: pauseSignalSnapshot(facts.Signals.Value),
		Changes: []colony.LifecycleChange{
			{Target: ".aether/data/COLONY_STATE.json", Action: "write"},
			{Target: ".aether/data/session.json", Action: "write"},
			{Target: ".aether/data/" + pauseHandoffDataPath, Action: "write"},
			{Target: ".aether/CONTEXT.md", Action: "write"},
			{Target: ".aether/HANDOFF.md", Action: "write"},
		},
		Evidence: evidence, Verification: pauseVerificationEvidence(facts.Verification.Value),
		Blockers: pauseBlockerEvidence(facts.Blockers.Value), Decisions: pauseDecisionEvidence(state, facts.Blockers.Value),
		StateEffect: colony.LifecycleStateEffectCommitted, Transaction: transaction, Receipt: receipt,
		Recovery: &colony.LifecycleRecovery{
			Provenance: provenance,
			Facts: []colony.LifecycleRecoveryFact{{
				Name: "safe-boundary-handoff", Summary: fmt.Sprintf("Handoff %s records %s and restart point %s.", handoffID, safeBoundary, restartPoint),
				Provenance: provenance, Evidence: append([]colony.LifecycleEvidence(nil), evidence...),
			}},
			Transaction: &transaction, Receipt: receipt,
			SafeNextStep: "Run `aether resume`; it validates state, session, repository, worktrees, activity, and this receipt before mutation.",
		},
		Provenance: provenance,
	}
	if strings.TrimSpace(handoff.AttemptID) == "" && strings.TrimSpace(handoff.RunID) == "" {
		handoff.RunID = session.SessionID
	}
	if err := handoff.Validate(); err != nil {
		return colony.PauseHandoff{}, fmt.Errorf("validate pause handoff: %w", err)
	}
	_ = now // captured evidence timestamps already come from the lifecycle fact snapshot
	return handoff, nil
}

func pauseRestartPoint(state colony.ColonyState) string {
	for _, phase := range state.Plan.Phases {
		if phase.ID != state.CurrentPhase {
			continue
		}
		for _, task := range phase.Tasks {
			if task.Status == colony.TaskCompleted {
				continue
			}
			if task.ID != nil && strings.TrimSpace(*task.ID) != "" {
				return fmt.Sprintf("phase %d task %s", state.CurrentPhase, strings.TrimSpace(*task.ID))
			}
			if strings.TrimSpace(task.Goal) != "" {
				return fmt.Sprintf("phase %d: %s", state.CurrentPhase, strings.TrimSpace(task.Goal))
			}
		}
	}
	return fmt.Sprintf("phase %d next unresolved step", state.CurrentPhase)
}

func pauseTaskIDs(state colony.ColonyState) []string {
	var ids []string
	for _, phase := range state.Plan.Phases {
		if phase.ID != state.CurrentPhase {
			continue
		}
		for index, task := range phase.Tasks {
			if task.Status == colony.TaskCompleted {
				continue
			}
			id := fmt.Sprintf("%d.%d", phase.ID, index+1)
			if task.ID != nil && strings.TrimSpace(*task.ID) != "" {
				id = strings.TrimSpace(*task.ID)
			}
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	return ids
}

func pauseHandoffEvidence(facts LifecycleFacts, repository colony.LifecycleRepositoryEvidence, contextDigest string) []colony.LifecycleEvidence {
	evidence := []colony.LifecycleEvidence{
		{ID: "state", Kind: "state", Source: facts.State.Source.Path, Digest: pauseFactDigest(facts.State.Source.Path), Summary: "Authoritative colony state at the pause boundary."},
		{ID: "session", Kind: "session", Source: facts.Session.Source.Path, Digest: pauseFactDigest(facts.Session.Source.Path), Summary: "Session context present at the pause boundary."},
		{ID: "context", Kind: "context", Source: contextDocumentPath(), Digest: contextDigest, Summary: "Context snapshot consumed by the handoff."},
		{ID: "repository", Kind: "git", Source: repository.Root, Digest: repository.DirtyDigest, Summary: "Repository HEAD and content-sensitive dirty fingerprint."},
		{ID: "actors", Kind: "lineage", Source: facts.Actors.Source.Path, Digest: pauseFactDigest(facts.Actors.Source.Path), Summary: "Actual worker lineage and statuses."},
		{ID: "signals", Kind: "signals", Source: facts.Signals.Source.Path, Digest: pauseFactDigest(facts.Signals.Source.Path), Summary: "Active direction snapshot."},
		{ID: "blockers", Kind: "decisions", Source: facts.Blockers.Source.Path, Digest: pauseFactDigest(facts.Blockers.Source.Path), Summary: "Unresolved owner/runtime decisions."},
	}
	return evidence
}

func pauseFactDigest(path string) string {
	content, err := os.ReadFile(filepath.FromSlash(path))
	if err != nil {
		return lifecycleTransactionMissingDigest
	}
	return lifecycleDigest(content)
}

func pauseVerificationEvidence(facts LifecycleVerificationFacts) []colony.LifecycleVerification {
	verification := make([]colony.LifecycleVerification, 0, len(facts.Gates)+1)
	for _, gate := range facts.Gates {
		verification = append(verification, colony.LifecycleVerification{
			Name: strings.TrimSpace(gate.Name), Passed: gate.Passed,
			EvidenceIDs: []string{"state", "repository"}, Detail: strings.TrimSpace(gate.Detail),
		})
	}
	if len(verification) == 0 {
		verification = append(verification, colony.LifecycleVerification{
			Name: "recovery-evidence-readable", Passed: true,
			EvidenceIDs: []string{"state", "session", "repository"}, Detail: "Required recovery sources were read without repair.",
		})
	}
	return verification
}

func pauseBlockerEvidence(flags []colony.FlagEntry) []colony.LifecycleIssue {
	var blockers []colony.LifecycleIssue
	for index, flag := range flags {
		if flag.Resolved {
			continue
		}
		id := strings.TrimSpace(flag.ID)
		if id == "" {
			id = fmt.Sprintf("pending-%d", index+1)
		}
		summary := strings.TrimSpace(flag.Description)
		if summary == "" {
			summary = "Pending recovery decision"
		}
		blockers = append(blockers, colony.LifecycleIssue{ID: id, Summary: summary, EvidenceIDs: []string{"blockers"}})
	}
	return blockers
}

func pauseDecisionEvidence(state colony.ColonyState, flags []colony.FlagEntry) []colony.LifecycleDecision {
	var decisions []colony.LifecycleDecision
	for index, decision := range state.Memory.Decisions {
		id := strings.TrimSpace(decision.ID)
		if id == "" {
			id = fmt.Sprintf("memory-decision-%d", index+1)
		}
		summary := strings.TrimSpace(decision.Claim)
		if summary == "" {
			summary = strings.TrimSpace(decision.Rationale)
		}
		if summary == "" {
			continue
		}
		decisions = append(decisions, colony.LifecycleDecision{
			ID: id, Scope: fmt.Sprintf("phase-%d", decision.Phase), Summary: summary, EvidenceIDs: []string{"state"},
		})
	}
	for index, flag := range flags {
		if flag.Resolved || !strings.EqualFold(strings.TrimSpace(flag.Type), "decision") {
			continue
		}
		id := strings.TrimSpace(flag.ID)
		if id == "" {
			id = fmt.Sprintf("pending-decision-%d", index+1)
		}
		scope := "project"
		if flag.Phase != nil {
			scope = fmt.Sprintf("phase-%d", *flag.Phase)
		}
		decisions = append(decisions, colony.LifecycleDecision{ID: id, Scope: scope, Summary: strings.TrimSpace(flag.Description), EvidenceIDs: []string{"blockers"}})
	}
	return decisions
}

func pauseWorkerLineage(actors []LifecycleActorFact) []colony.LifecycleWorkerLineage {
	lineage := make([]colony.LifecycleWorkerLineage, 0, len(actors))
	for _, actor := range actors {
		workerID := strings.TrimSpace(actor.Name)
		caste := strings.TrimSpace(actor.Caste)
		status := strings.TrimSpace(actor.Status)
		if workerID == "" || caste == "" || status == "" {
			continue
		}
		lineage = append(lineage, colony.LifecycleWorkerLineage{
			WorkerID: workerID, ParentWorkerID: strings.TrimSpace(actor.Parent), Caste: caste, Status: status,
		})
	}
	sort.Slice(lineage, func(i, j int) bool { return lineage[i].WorkerID < lineage[j].WorkerID })
	return lineage
}

func pauseSignalSnapshot(signals []colony.PheromoneSignal) []colony.LifecycleSignalSnapshot {
	var snapshot []colony.LifecycleSignalSnapshot
	for _, signal := range signals {
		if !signal.Active || strings.TrimSpace(signal.ID) == "" {
			continue
		}
		digest := ""
		if signal.ContentHash != nil {
			digest = strings.TrimSpace(*signal.ContentHash)
		}
		if digest == "" {
			digest = lifecycleDigest(signal.Content)
		}
		snapshot = append(snapshot, colony.LifecycleSignalSnapshot{
			SignalID: strings.TrimSpace(signal.ID), Type: strings.TrimSpace(signal.Type), Digest: digest,
		})
	}
	sort.Slice(snapshot, func(i, j int) bool { return snapshot[i].SignalID < snapshot[j].SignalID })
	return snapshot
}

func pauseWorktreeEvidence(root string, entries []colony.WorktreeEntry) []colony.LifecycleWorktreeEvidence {
	worktrees := make([]colony.LifecycleWorktreeEvidence, 0, len(entries))
	for _, entry := range entries {
		path := strings.TrimSpace(entry.Path)
		absolute := path
		if !filepath.IsAbs(absolute) {
			absolute = filepath.Join(root, filepath.FromSlash(path))
		}
		head, _ := gitOutputAt(absolute, "rev-parse", "HEAD")
		digest := lifecycleTransactionMissingDigest
		if info, err := os.Stat(absolute); err == nil && info.IsDir() {
			if dirty, dirtyErr := repositoryDirtyDigest(absolute); dirtyErr == nil {
				digest = dirty
			}
		}
		worktrees = append(worktrees, colony.LifecycleWorktreeEvidence{
			ID: strings.TrimSpace(entry.ID), Path: path, Branch: strings.TrimSpace(entry.Branch),
			Head: strings.TrimSpace(head), Digest: digest,
		})
	}
	sort.Slice(worktrees, func(i, j int) bool {
		if worktrees[i].ID == worktrees[j].ID {
			return worktrees[i].Path < worktrees[j].Path
		}
		return worktrees[i].ID < worktrees[j].ID
	})
	return worktrees
}

func reflectPauseWorktreesEqual(left, right []colony.LifecycleWorktreeEvidence) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func pauseRepositoryEvidence(root string) (colony.LifecycleRepositoryEvidence, error) {
	head, err := gitOutputAt(root, "rev-parse", "HEAD")
	if err != nil || strings.TrimSpace(head) == "" {
		treeDigest, treeErr := repositoryTreeDigest(root)
		if treeErr != nil {
			if err == nil {
				err = fmt.Errorf("empty HEAD")
			}
			return colony.LifecycleRepositoryEvidence{}, fmt.Errorf("read repository evidence: %w", errors.Join(err, treeErr))
		}
		return colony.LifecycleRepositoryEvidence{Root: root, Head: "unavailable", DirtyDigest: treeDigest}, nil
	}
	dirty, err := repositoryDirtyDigest(root)
	if err != nil {
		return colony.LifecycleRepositoryEvidence{}, fmt.Errorf("fingerprint repository changes: %w", err)
	}
	return colony.LifecycleRepositoryEvidence{Root: root, Head: strings.TrimSpace(head), DirtyDigest: dirty}, nil
}

func repositoryTreeDigest(root string) (string, error) {
	var records []string
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		relative = filepath.ToSlash(relative)
		if relative == "." {
			return nil
		}
		if entry.IsDir() && (relative == ".git" || relative == ".aether-transactions" || relative == ".aether/data" || relative == ".aether/locks" ||
			strings.HasPrefix(relative, ".git/") || strings.HasPrefix(relative, ".aether-transactions/") || strings.HasPrefix(relative, ".aether/data/") || strings.HasPrefix(relative, ".aether/locks/")) {
			return filepath.SkipDir
		}
		if pauseOwnedRepositoryPath(relative) {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		switch {
		case info.Mode().IsRegular():
			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			records = append(records, relative+"\x00"+lifecycleDigest(content))
		case info.Mode()&os.ModeSymlink != 0:
			target, err := os.Readlink(path)
			if err != nil {
				return err
			}
			records = append(records, relative+"\x00symlink:"+target)
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	sort.Strings(records)
	return lifecycleDigest([]byte(strings.Join(records, "\n"))), nil
}

func gitOutputAt(root string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), GitTimeout)
	defer cancel()
	gitArgs := append([]string{"-C", root}, args...)
	output, err := exec.CommandContext(ctx, "git", gitArgs...).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(string(output)))
	}
	return strings.TrimSpace(string(output)), nil
}

// repositoryDirtyDigest is content-sensitive and ignores only the runtime's
// own pause artifacts and transaction evidence. It therefore notices changed
// working bytes even when porcelain's two-letter status is unchanged, while a
// pause/resume replay does not conflict with files the same transaction owns.
func repositoryDirtyDigest(root string) (string, error) {
	status, err := gitOutputAt(root, "status", "--porcelain=v1", "--untracked-files=all")
	if err != nil {
		return "", err
	}
	lines := strings.Split(status, "\n")
	var records []string
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		path := pausePorcelainPath(line)
		if pauseOwnedRepositoryPath(path) {
			continue
		}
		absolute := filepath.Join(root, filepath.FromSlash(path))
		content, readErr := os.ReadFile(absolute)
		digest := lifecycleTransactionMissingDigest
		if readErr == nil {
			digest = lifecycleDigest(content)
		} else if !os.IsNotExist(readErr) {
			return "", readErr
		}
		records = append(records, line+"\x00"+digest)
	}
	sort.Strings(records)
	return lifecycleDigest([]byte(strings.Join(records, "\n"))), nil
}

func pausePorcelainPath(line string) string {
	path := strings.TrimSpace(line)
	if len(line) >= 4 {
		path = strings.TrimSpace(line[3:])
	}
	if _, after, ok := strings.Cut(path, " -> "); ok {
		path = after
	}
	return strings.Trim(path, `"`)
}

func pauseOwnedRepositoryPath(path string) bool {
	path = filepath.ToSlash(strings.TrimSpace(path))
	switch path {
	case ".aether/HANDOFF.md", ".aether/CONTEXT.md":
		return true
	}
	return strings.HasPrefix(path, ".aether/data/") ||
		strings.HasPrefix(path, ".aether/locks/") ||
		strings.HasPrefix(path, ".aether-transactions/") ||
		strings.HasPrefix(path, ".aether/.aether-transactions/")
}

func pauseHandoffReferencesEqual(left, right colony.PauseHandoffReference) bool {
	return left.ID == right.ID && left.TransactionID == right.TransactionID && left.Digest == right.Digest && left.Path == right.Path
}

func pauseResumeSessionIsStale(session colony.SessionFile, now time.Time, expectedHead string) bool {
	startedAt, err := time.Parse(time.RFC3339, strings.TrimSpace(session.StartedAt))
	if err != nil || now.Sub(startedAt) >= sessionStaleThreshold {
		return true
	}
	return strings.TrimSpace(session.BaselineCommit) != "" && strings.TrimSpace(expectedHead) != "" && strings.TrimSpace(session.BaselineCommit) != strings.TrimSpace(expectedHead)
}

// pauseResumeLifecycleEvent builds the stored four-field lifecycle event
// (`timestamp|event_type|source|message`) that nextActionEventSentence
// (cmd/next_action.go) reads. sentence is REQUIRED to already be a finished,
// owner-readable sentence -- the lifecycleEventSentence type refuses any
// other value at compile time -- so no internal identifier or raw
// enumeration value can reach this record's last field. The handoff
// identifier is deliberately not part of this record: it is already durably
// recorded on colony state as the pause-handoff reference
// (colony.PauseHandoffReference), which is where recovery reads it.
func pauseResumeLifecycleEvent(now time.Time, command string, sentence lifecycleEventSentence) string {
	return fmt.Sprintf("%s|lifecycle_%s|%s|%s", now.UTC().Format(time.RFC3339), command, command, sentence)
}

func buildTransactionalHandoffDocument(now time.Time, state colony.ColonyState, session colony.SessionFile, handoff colony.PauseHandoff) string {
	base := buildHandoffDocument(now.UTC(), state, session, "aether resume")
	var b strings.Builder
	b.WriteString(base)
	if !strings.HasSuffix(base, "\n") {
		b.WriteByte('\n')
	}
	b.WriteString("\n## Recovery Receipt\n")
	b.WriteString("Handoff ID: ")
	b.WriteString(handoff.HandoffID)
	b.WriteString("\nSafe boundary: ")
	b.WriteString(handoff.SafeBoundary)
	b.WriteString("\nRestart point: ")
	b.WriteString(handoff.RestartPoint)
	b.WriteString("\nTransaction: ")
	b.WriteString(handoff.Transaction.ID)
	b.WriteString("\nReceipt: ")
	if handoff.Receipt != nil {
		b.WriteString(handoff.Receipt.ID)
	}
	b.WriteString("\nProvenance: ")
	b.WriteString(string(handoff.Provenance))
	b.WriteString("\n\nClose this handoff with `/ant-resume` (runtime: `aether resume`).\n")
	return b.String()
}

func loadStateAfterLifecycleOutcome() colony.ColonyState {
	var state colony.ColonyState
	if store != nil && store.LoadJSON("COLONY_STATE.json", &state) == nil {
		return normalizeLegacyColonyState(state)
	}
	return colony.ColonyState{}
}

func writeLegacySessionRedirectNotice() {
	if legacySessionProcessRedirect == nil {
		return
	}
	notice := legacySessionRedirectNotice(legacySessionProcessRedirect)
	if notice != "" {
		visualFprintln(stderr, notice)
	}
}

func loadResumeState(handoffText string, noHandoff bool) (colony.ColonyState, bool, error) {
	var rawState colony.ColonyState
	stateLoaded := false
	if err := store.LoadJSON("COLONY_STATE.json", &rawState); err == nil {
		stateLoaded = true
		state := normalizeLegacyColonyState(rawState)
		if resumeStateIsRunnable(state) {
			return state, false, nil
		}
	}

	if noHandoff {
		return colony.ColonyState{}, false, fmt.Errorf("COLONY_STATE.json is not runnable and HANDOFF.md fallback is disabled")
	}

	state, err := restoreStateFromHandoff(handoffText)
	if err != nil {
		return colony.ColonyState{}, false, fmt.Errorf("runtime state is broken and could not be restored from handoff: %w", err)
	}
	if stateLoaded {
		warnResumeHandoffFallback(rawState, state)
		if resumeHandoffGoalMismatch(rawState, state) {
			return colony.ColonyState{}, false, fmt.Errorf("HANDOFF.md appears to belong to a different colony; current COLONY_STATE.json goal %q does not match handoff goal %q", colonyStateGoalText(rawState), colonyStateGoalText(state))
		}
	}
	if err := store.SaveJSON("COLONY_STATE.json", state); err != nil {
		return colony.ColonyState{}, false, fmt.Errorf("failed to restore COLONY_STATE.json from handoff: %w", err)
	}
	return state, true, nil
}

func warnResumeHandoffFallback(currentState, handoffState colony.ColonyState) {
	visualFprintln(stderr, "warning: COLONY_STATE.json is not runnable; attempting to restore from HANDOFF.md")

	currentGoal := colonyStateGoalText(currentState)
	handoffGoal := colonyStateGoalText(handoffState)
	if currentGoal != "" && handoffGoal != "" && !goalsMatch(currentGoal, handoffGoal) {
		visualFprintf(stderr, "warning: HANDOFF.md goal %q does not match current COLONY_STATE.json goal %q\n", handoffGoal, currentGoal)
	}

	warnIfHandoffOlderThanCurrentState()
}

func resumeHandoffGoalMismatch(currentState, handoffState colony.ColonyState) bool {
	currentGoal := colonyStateGoalText(currentState)
	handoffGoal := colonyStateGoalText(handoffState)
	return currentGoal != "" && handoffGoal != "" && !goalsMatch(currentGoal, handoffGoal)
}

func warnIfHandoffOlderThanCurrentState() {
	if store == nil {
		return
	}
	stateInfo, stateErr := os.Stat(filepath.Join(store.BasePath(), "COLONY_STATE.json"))
	handoffInfo, handoffErr := os.Stat(handoffDocumentPath())
	if stateErr != nil || handoffErr != nil {
		return
	}
	if handoffInfo.ModTime().Before(stateInfo.ModTime()) {
		visualFprintf(stderr, "warning: HANDOFF.md is older than COLONY_STATE.json; verify the recovered colony before continuing\n")
	}
}

func colonyStateGoalText(state colony.ColonyState) string {
	if state.Goal == nil {
		return ""
	}
	return strings.TrimSpace(*state.Goal)
}

func goalsMatch(a, b string) bool {
	return strings.EqualFold(strings.TrimSpace(a), strings.TrimSpace(b))
}

func resumeStateIsRunnable(state colony.ColonyState) bool {
	if state.Goal == nil || strings.TrimSpace(*state.Goal) == "" {
		return false
	}
	state = normalizeLegacyColonyState(state)
	switch state.State {
	case colony.StateIDLE, colony.StateREADY, colony.StateEXECUTING, colony.StateBUILT, colony.StateCOMPLETED:
	default:
		return false
	}
	if state.CurrentPhase < 0 {
		return false
	}
	if len(state.Plan.Phases) > 0 && state.CurrentPhase > len(state.Plan.Phases) {
		return false
	}
	return true
}

func restoreStateFromHandoff(handoffText string) (colony.ColonyState, error) {
	if strings.TrimSpace(handoffText) == "" {
		return colony.ColonyState{}, fmt.Errorf("HANDOFF.md is missing or empty")
	}
	if state, ok := restoreStateFromHandoffSnapshot(handoffText); ok {
		return state, nil
	}
	return restoreStateFromLegacyHandoff(handoffText)
}

func restoreStateFromHandoffSnapshot(handoffText string) (colony.ColonyState, bool) {
	block := extractFencedBlock(handoffText, handoffStateFence)
	if block == "" {
		return colony.ColonyState{}, false
	}
	var state colony.ColonyState
	if err := json.Unmarshal([]byte(block), &state); err != nil {
		return colony.ColonyState{}, false
	}
	state = normalizeLegacyColonyState(state)
	if !resumeStateIsRunnable(state) {
		return colony.ColonyState{}, false
	}
	return state, true
}

func restoreStateFromLegacyHandoff(handoffText string) (colony.ColonyState, error) {
	goal := parseHandoffGoal(handoffText)
	phaseID, totalPhases, phaseName := parseHandoffPhase(handoffText)
	stateValue := parseHandoffState(handoffText)
	if strings.TrimSpace(goal) == "" {
		return colony.ColonyState{}, fmt.Errorf("HANDOFF.md does not contain a recoverable goal")
	}
	if phaseID < 0 {
		return colony.ColonyState{}, fmt.Errorf("HANDOFF.md does not contain a recoverable phase")
	}
	if totalPhases < phaseID {
		totalPhases = phaseID
	}
	if totalPhases < 0 {
		totalPhases = 0
	}
	if stateValue == "" {
		stateValue = colony.StateREADY
	}

	phases := make([]colony.Phase, 0, totalPhases)
	for i := 1; i <= totalPhases; i++ {
		status := colony.PhaseReady
		if i < phaseID {
			status = colony.PhaseCompleted
		}
		if i == phaseID {
			status = colony.PhaseInProgress
		}
		name := fmt.Sprintf("Phase %d", i)
		if i == phaseID && strings.TrimSpace(phaseName) != "" {
			name = strings.TrimSpace(phaseName)
		}
		phases = append(phases, colony.Phase{ID: i, Name: name, Status: status})
	}
	if phaseID > 0 && phaseID <= len(phases) {
		tasks := parseHandoffTasks(handoffText, phaseID)
		phases[phaseID-1].Tasks = tasks
	}

	return colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		Scope:        colony.ScopeProject,
		State:        stateValue,
		CurrentPhase: phaseID,
		Plan:         colony.Plan{Phases: phases},
		Events: []string{
			fmt.Sprintf("%s|state_recovered|resume|Restored COLONY_STATE.json from HANDOFF.md", time.Now().UTC().Format(time.RFC3339)),
		},
	}, nil
}

func extractFencedBlock(text, fenceName string) string {
	lines := strings.Split(text, "\n")
	inBlock := false
	var block []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !inBlock {
			if strings.HasPrefix(trimmed, "```") && strings.TrimSpace(strings.TrimPrefix(trimmed, "```")) == fenceName {
				inBlock = true
			}
			continue
		}
		if strings.HasPrefix(trimmed, "```") {
			return strings.TrimSpace(strings.Join(block, "\n"))
		}
		block = append(block, line)
	}
	return ""
}

func parseHandoffGoal(text string) string {
	if section := extractHandoffMarkdownSection(text, "Goal"); section != "" {
		if value := firstMeaningfulHandoffLine(section); value != "" {
			return value
		}
	}
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if value, ok := strings.CutPrefix(line, "Goal:"); ok {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func parseHandoffPhase(text string) (int, int, string) {
	for _, line := range strings.Split(text, "\n") {
		matches := handoffPhaseLinePattern.FindStringSubmatch(strings.TrimSpace(line))
		if len(matches) == 0 {
			continue
		}
		phaseID, _ := strconv.Atoi(matches[1])
		total, _ := strconv.Atoi(matches[2])
		phaseName := ""
		if len(matches) > 3 {
			phaseName = strings.TrimSpace(matches[3])
		}
		return phaseID, total, phaseName
	}
	return -1, -1, ""
}

func parseHandoffState(text string) colony.State {
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "- "))
		var value string
		if raw, ok := strings.CutPrefix(line, "State:"); ok {
			value = strings.TrimSpace(raw)
		}
		switch colony.State(strings.ToUpper(value)) {
		case colony.StateIDLE, colony.StateREADY, colony.StateEXECUTING, colony.StateBUILT, colony.StateCOMPLETED:
			return colony.State(strings.ToUpper(value))
		}
	}
	return ""
}

func parseHandoffTasks(text string, phaseID int) []colony.Task {
	section := extractHandoffMarkdownSection(text, "Tasks")
	if section == "" {
		section = extractHandoffMarkdownSection(text, "Open Tasks")
	}
	var tasks []colony.Task
	taskNum := 1
	for _, line := range strings.Split(section, "\n") {
		value := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "- "))
		value = strings.TrimSpace(strings.TrimPrefix(value, "[ ]"))
		value = strings.TrimSpace(strings.TrimPrefix(value, "[>]"))
		value = strings.TrimSpace(strings.TrimPrefix(value, "[x]"))
		if value == "" || strings.EqualFold(value, "none") {
			continue
		}
		id := fmt.Sprintf("%d.%d", phaseID, taskNum)
		tasks = append(tasks, colony.Task{ID: &id, Goal: value, Status: colony.TaskPending})
		taskNum++
	}
	return tasks
}

func extractHandoffMarkdownSection(text, heading string) string {
	lines := strings.Split(text, "\n")
	inSection := false
	var section []string
	target := "## " + strings.TrimSpace(heading)
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.EqualFold(trimmed, target) {
			inSection = true
			continue
		}
		if inSection && strings.HasPrefix(trimmed, "## ") {
			break
		}
		if inSection {
			section = append(section, line)
		}
	}
	return strings.TrimSpace(strings.Join(section, "\n"))
}

func firstMeaningfulHandoffLine(section string) string {
	for _, line := range strings.Split(section, "\n") {
		line = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "- "))
		if line == "" || strings.EqualFold(line, "none") {
			continue
		}
		return line
	}
	return ""
}

func buildHandoffDocument(now time.Time, state colony.ColonyState, session colony.SessionFile, nextAction string) string {
	var b strings.Builder
	goal := session.ColonyGoal
	if goal == "" && state.Goal != nil {
		goal = *state.Goal
	}
	totalPhases := len(state.Plan.Phases)
	phaseName := lookupPhaseName(state, state.CurrentPhase)

	b.WriteString("# Colony Handoff\n\n")
	b.WriteString("Paused: ")
	b.WriteString(now.Format(time.RFC3339))
	b.WriteString("\n")
	b.WriteString("Goal: ")
	b.WriteString(emptyFallback(goal, "No goal set"))
	b.WriteString("\n")
	b.WriteString("State: ")
	b.WriteString(string(state.State))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("Phase: %d/%d", state.CurrentPhase, totalPhases))
	if strings.TrimSpace(phaseName) != "" && phaseName != "(unnamed)" {
		b.WriteString(" — ")
		b.WriteString(phaseName)
	}
	b.WriteString("\n")
	b.WriteString("Next: ")
	b.WriteString(nextAction)
	b.WriteString("\n")
	b.WriteString("Suggested resume: aether resume\n\n")

	openTasks := currentOpenTasks(state)
	if len(openTasks) > 0 {
		b.WriteString("## Open Tasks\n")
		for _, task := range openTasks {
			b.WriteString("- ")
			b.WriteString(task)
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	if strings.TrimSpace(session.Summary) != "" {
		b.WriteString("## Session Summary\n")
		b.WriteString(session.Summary)
		b.WriteString("\n")
	}
	b.WriteString(renderHandoffStateSnapshot(state))

	return b.String()
}

func currentOpenTasks(state colony.ColonyState) []string {
	if state.CurrentPhase < 1 || state.CurrentPhase > len(state.Plan.Phases) {
		return nil
	}
	phase := state.Plan.Phases[state.CurrentPhase-1]
	var tasks []string
	for _, task := range phase.Tasks {
		if task.Status == colony.TaskCompleted {
			continue
		}
		if strings.TrimSpace(task.Goal) == "" {
			continue
		}
		tasks = append(tasks, strings.TrimSpace(task.Goal))
	}
	return tasks
}

func loadOrCreateSessionSummary(now time.Time, state colony.ColonyState) (colony.SessionFile, error) {
	var session colony.SessionFile
	if err := store.LoadJSON("session.json", &session); err == nil {
		return session, nil
	}

	goal := ""
	if state.Goal != nil {
		goal = *state.Goal
	}
	return colony.SessionFile{
		SessionID:        fmt.Sprintf("%d_%s", now.Unix(), randomHex(4)),
		StartedAt:        now.Format(time.RFC3339),
		ColonyGoal:       goal,
		ColonyMode:       state.EffectiveColonyMode(),
		CurrentPhase:     state.CurrentPhase,
		CurrentMilestone: state.Milestone,
		SuggestedNext:    "aether resume",
		ContextCleared:   true,
		BaselineCommit:   getGitHEAD(),
		ResumedAt:        nil,
		ActiveTodos:      currentOpenTasks(state),
		Summary:          "Session paused",
	}, nil
}

func init() {
	resumeColonyCmd.Flags().BoolVar(&resumeNoHandoff, "no-handoff", false, "disable HANDOFF.md fallback when COLONY_STATE.json is not runnable")
	rootCmd.AddCommand(pauseColonyCmd)
	rootCmd.AddCommand(resumeColonyCmd)
}
