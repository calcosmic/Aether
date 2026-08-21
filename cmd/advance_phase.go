package cmd

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

// advancePhaseParams carries exactly the scalar fields advancePhase needs to
// commit a phase advancement. Deliberately, this struct does NOT include a
// colony.ColonyState or colony.Phase field. That omission is not an
// oversight -- it is what makes the clobber bug this function replaces
// (cmd/codex_continue_finalize.go's former `updated = state`, which discarded
// the value UpdateJSONAtomically had just freshly loaded from disk and
// replaced it with a stale captured variable) structurally impossible to
// reintroduce: there is no full stale state value in scope for the closure
// below to assign from. Both callers keep their own state/phase locals for
// everything OUTSIDE this function -- only the mutation core itself is
// narrowed to scalars.
type advancePhaseParams struct {
	PhaseID                int
	ExpectedBuildStartedAt *time.Time
	AllowedStates          []colony.State
	Source                 string
	Now                    time.Time
}

// advancePhaseResult is what a caller needs to adapt back into its own,
// larger return shape after a successful advance.
type advancePhaseResult struct {
	Updated     colony.ColonyState
	NextPhase   *colony.Phase
	NextCommand string
	Final       bool
}

// advancePhase is the one shared, atomic core for "commit phase completion".
// Both aether continue (runCodexContinue) and aether continue-finalize
// (advanceExternalContinue) call this instead of each maintaining their own
// copy of the same ~60-line block. It reads the current on-disk
// COLONY_STATE.json, re-validates that the phase and build this call was
// asked to advance are still the ones actually in progress
// (validateRuntimeStateStillCurrent), and only then mutates and writes.
// A supersession refusal (errRuntimeStateSuperseded) writes nothing --
// UpdateJSONAtomically never marshals/renames when mutate returns an error.
func advancePhase(params advancePhaseParams) (advancePhaseResult, error) {
	if store == nil {
		return advancePhaseResult{}, fmt.Errorf("no store initialized")
	}

	var (
		nextPhase   *colony.Phase
		nextCommand string
		final       bool
		updated     colony.ColonyState
	)
	if err := store.UpdateJSONAtomically("COLONY_STATE.json", &updated, func() error {
		if err := validateRuntimeStateStillCurrent(updated, params.PhaseID, params.ExpectedBuildStartedAt, params.AllowedStates...); err != nil {
			return err
		}

		currentIdx := params.PhaseID - 1
		updated.Events = append(trimmedEvents(updated.Events),
			fmt.Sprintf("%s|verification_passed|%s|Build verification passed for phase %d", params.Now.Format(time.RFC3339), params.Source, params.PhaseID),
			fmt.Sprintf("%s|gate_passed|%s|Continue gates passed for phase %d", params.Now.Format(time.RFC3339), params.Source, params.PhaseID),
		)
		updated.Plan.Phases[currentIdx].Status = colony.PhaseCompleted
		for i := range updated.Plan.Phases[currentIdx].Tasks {
			updated.Plan.Phases[currentIdx].Tasks[i].Status = colony.TaskCompleted
		}
		updated.BuildStartedAt = nil
		updated.GateResults = nil

		final = currentIdx == len(updated.Plan.Phases)-1
		nextCommand = "aether seal"
		if final {
			updated.State = colony.StateCOMPLETED
			updated.CurrentPhase = params.PhaseID
			updated.Events = append(updated.Events,
				fmt.Sprintf("%s|phase_completed|%s|Completed final phase %d", params.Now.Format(time.RFC3339), params.Source, updated.CurrentPhase),
			)
		} else {
			nextIdx := currentIdx + 1
			if updated.Plan.Phases[nextIdx].Status == colony.PhasePending || updated.Plan.Phases[nextIdx].Status == "" {
				updated.Plan.Phases[nextIdx].Status = colony.PhaseReady
			}
			updated.CurrentPhase = nextIdx + 1
			nextPhase = &updated.Plan.Phases[nextIdx]
			updated.State = colony.StateREADY
			nextCommand = fmt.Sprintf("aether build %d", nextIdx+1)
			updated.Events = append(updated.Events,
				fmt.Sprintf("%s|phase_advanced|%s|Completed phase %d, ready for phase %d", params.Now.Format(time.RFC3339), params.Source, params.PhaseID, nextIdx+1),
			)
		}
		return nil
	}); err != nil {
		return advancePhaseResult{}, err
	}

	return advancePhaseResult{Updated: updated, NextPhase: nextPhase, NextCommand: nextCommand, Final: final}, nil
}

// --- FIELD-04: preserve-and-replay a continue advance that loses the pause race ---
//
// A ~10-minute continue verification can finish just as a stop hook pauses
// the colony underneath it. advancePhase above correctly refuses to commit
// (validateRuntimeStateStillCurrent's FIRST check, "colony is paused" --
// Phase 188's supersession protection working as designed) but, before this
// mechanism existed, both callers then discarded everything they had just
// computed and returned a generic superseded result. That is real, expensive
// work lost to UX, not a correctness problem -- so on this ONE specific
// supersession reason (never any other), the already-computed payload is
// preserved durably and replayed automatically the next time continue runs
// against a colony state whose phase and build identity still match exactly
// (191.1-CONTEXT.md D-07/D-08, 191.1-PATTERNS.md Pattern 5: one shared
// mechanism in this file, not two independent per-caller copies).

// pendingContinueAdvancePayload is the caller-supplied bundle preserved when
// a completed continue advance loses the race to a colony pause. Only the
// already-computed verification/gate/review reports are kept -- not the
// surrounding pipeline's worker-flow bookkeeping, learning capture, or
// phase-commit machinery, none of which is the expensive ~10-minute part
// this mechanism exists to avoid re-running.
type pendingContinueAdvancePayload struct {
	Verification codexContinueVerificationReport `json:"verification"`
	Assessment   codexContinueAssessment         `json:"assessment"`
	Gates        codexContinueGateReport         `json:"gates"`
	Review       codexContinueReviewReport       `json:"review"`
	ReviewDepth  colony.VerificationDepth        `json:"review_depth"`
}

// pendingContinueAdvanceRecord is the durable, on-disk form of a preserved
// continue advance, keyed to the exact phase + build identity it was
// computed against. Replay requires an exact match against CURRENT state
// (T-191.1-02-03) -- a genuinely different build that happened in between is
// discarded, never blindly replayed.
type pendingContinueAdvanceRecord struct {
	PhaseID        int        `json:"phase_id"`
	BuildStartedAt *time.Time `json:"build_started_at,omitempty"`
	// WorkspaceFingerprint is a content-sensitive digest of the on-disk
	// workspace at preservation time (CR-02, 191.1-REVIEW.md) -- see
	// pendingContinueAdvanceWorkspaceFingerprint's own doc comment for why
	// this is a distinct primitive from codex.WorkspaceFingerprint, not a
	// reuse of it.
	WorkspaceFingerprint string                        `json:"workspace_fingerprint,omitempty"`
	Source               string                        `json:"source"`
	CreatedAt            time.Time                     `json:"created_at"`
	Payload              pendingContinueAdvancePayload `json:"payload"`
}

// pendingContinueAdvanceWorkspaceFingerprint returns a content-sensitive
// digest of the on-disk workspace at root, for the CR-02 (191.1-REVIEW.md)
// freshness check: does the workspace a preserved verification covered still
// match the workspace that exists right now at replay time?
//
// This is deliberately a DIFFERENT primitive from codex.WorkspaceFingerprint
// (pkg/codex/execution_binding.go), not a reuse of it. That function
// identifies WHICH checkout/branch a worker is bound to and explicitly
// excludes file content by design ("workers are expected to modify files ...
// so content hashes would reject valid work") because its job spans the
// whole build, where legitimate worker edits are the norm. The
// preserve/replay window this guards is the opposite case: nothing is
// supposed to change here (the colony is paused, no build is running), so
// content sensitivity is exactly the property needed -- a hand edit between
// preserve and replay (no new commit, same branch) must change this value,
// which codex.WorkspaceFingerprint would deliberately never do.
//
// When root is a git checkout, the fingerprint covers the current commit
// plus every uncommitted change against it (git diff HEAD, which covers
// both staged and unstaged modifications to tracked files) and every
// untracked/status change (git status --porcelain) -- cheap, because git
// does the change-detection internally rather than this function hashing
// every file in the tree, matching D-07's "cheap re-apply" framing. .aether/data
// is gitignored in a real checkout, so the colony's own bookkeeping churn
// (verification.json, gates.json, COLONY_STATE.json) is naturally excluded
// without this function needing to know that path itself.
//
// When root is not a git checkout (or git is unavailable), this falls back
// to a directory-identity-only digest -- matching codex.WorkspaceFingerprint's
// own directory-mode fallback -- so a non-git workspace preserves today's
// behavior exactly (a constant fingerprint that always matches itself)
// rather than gaining a new refusal path where none existed before.
func pendingContinueAdvanceWorkspaceFingerprint(root string) string {
	root = strings.TrimSpace(root)
	if root == "" {
		root = "."
	}
	if head, err := runGitForWorkspaceFingerprint(root, "rev-parse", "HEAD"); err == nil {
		diff, _ := runGitForWorkspaceFingerprint(root, "diff", "HEAD", "--")
		status, _ := runGitForWorkspaceFingerprint(root, "status", "--porcelain")
		sum := sha256.Sum256([]byte("git\x00" + head + "\x00" + diff + "\x00" + status))
		return fmt.Sprintf("%x", sum)
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		abs = root
	}
	sum := sha256.Sum256([]byte("directory\x00" + filepath.Clean(abs)))
	return fmt.Sprintf("%x", sum)
}

func runGitForWorkspaceFingerprint(root string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = root
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(output), nil
}

// pendingContinueAdvancePath follows the existing
// .aether/data/build/phase-<N>/ naming convention already used for
// verification.json, gates.json, continue.json and review.json
// (cleanupStaleContinueReports, cmd/codex_continue.go).
func pendingContinueAdvancePath(phaseID int) string {
	return filepath.ToSlash(filepath.Join("build", fmt.Sprintf("phase-%d", phaseID), "pending-advance.json"))
}

// preservePendingContinueAdvance durably stores an already-computed continue
// advance payload for later replay, keyed to phaseID + buildStartedAt +
// (CR-02) a content fingerprint of the workspace right now.
func preservePendingContinueAdvance(phaseID int, buildStartedAt *time.Time, source string, now time.Time, payload pendingContinueAdvancePayload) error {
	if store == nil {
		return fmt.Errorf("no store initialized")
	}
	return store.SaveJSON(pendingContinueAdvancePath(phaseID), pendingContinueAdvanceRecord{
		PhaseID:              phaseID,
		BuildStartedAt:       buildStartedAt,
		WorkspaceFingerprint: pendingContinueAdvanceWorkspaceFingerprint(buildAttemptWorkspaceRoot()),
		Source:               source,
		CreatedAt:            now,
		Payload:              payload,
	})
}

// preserveIfPausedSupersession is called from both continue call sites'
// errRuntimeStateSuperseded handling (cmd/codex_continue.go's runCodexContinue,
// cmd/codex_continue_finalize.go's advanceExternalContinue) -- the SAME
// function from both, per Pattern 5. It reloads current colony state and
// checks its Paused field directly rather than string-matching the error
// text -- that field is exactly what runtimeStateSupersededError's "colony
// is paused" branch (cmd/codex_build.go, validateRuntimeStateStillCurrent)
// was built from. Only on that specific reason does it preserve the
// caller's already-computed payload; any other supersession reason (a
// genuinely different phase or build) is left to discard exactly as before
// -- nothing is written.
func preserveIfPausedSupersession(phaseID int, buildStartedAt *time.Time, source string, now time.Time, payload pendingContinueAdvancePayload) {
	latest, err := loadActiveColonyState()
	if err != nil || !latest.Paused {
		return
	}
	_ = preservePendingContinueAdvance(phaseID, buildStartedAt, source, now, payload)
}

// loadPendingContinueAdvance returns the pending record for phaseID if one
// exists AND its BuildStartedAt identity matches currentBuildStartedAt
// exactly AND (CR-02, 191.1-REVIEW.md) its preserved workspace fingerprint
// still matches the workspace right now. A missing record returns ok=false.
// A record whose identity or workspace fingerprint does NOT match is
// discarded (deleted) as a side effect and also returns ok=false
// (T-191.1-02-02/03) -- it can never be replayed, not now and not on a later
// call either.
func loadPendingContinueAdvance(phaseID int, currentBuildStartedAt *time.Time) (pendingContinueAdvanceRecord, bool) {
	var record pendingContinueAdvanceRecord
	if store == nil {
		return record, false
	}
	if err := store.LoadJSON(pendingContinueAdvancePath(phaseID), &record); err != nil {
		return pendingContinueAdvanceRecord{}, false
	}
	// runtimeStartedAtMatches (cmd/codex_build.go) is the exact same
	// nil-safe comparison advancePhase's own validateRuntimeStateStillCurrent
	// check uses -- reused here rather than duplicated, so this identity
	// check can never silently diverge from the one advancePhase itself
	// applies at commit time.
	identityMatches := record.PhaseID == phaseID && runtimeStartedAtMatches(record.BuildStartedAt, currentBuildStartedAt)
	// CR-02: identity (phase + build) matching is necessary but not
	// sufficient -- neither it nor advancePhase's own currency check touches
	// the filesystem. A workspace fingerprint mismatch means real work
	// happened outside the build pipeline between preserve and replay (a
	// hand edit, a pair session -- FIELD-05's own justification for
	// existing, cmd/verify_out_of_band.go) and the preserved payload no
	// longer describes the workspace that exists right now; it must never
	// be blindly replayed.
	workspaceMatches := record.WorkspaceFingerprint == pendingContinueAdvanceWorkspaceFingerprint(buildAttemptWorkspaceRoot())
	if !identityMatches || !workspaceMatches {
		clearPendingContinueAdvance(phaseID)
		return pendingContinueAdvanceRecord{}, false
	}
	return record, true
}

// clearPendingContinueAdvance deletes the pending record so it can never be
// replayed twice (after a successful replay) or ever again (after being
// discarded as stale).
func clearPendingContinueAdvance(phaseID int) {
	if store == nil {
		return
	}
	_ = os.Remove(filepath.Join(store.BasePath(), filepath.FromSlash(pendingContinueAdvancePath(phaseID))))
}

// pendingContinueReplayOutcome is the caller-agnostic result of checking for
// and attempting to replay a preserved pending continue advance. Handled is
// false when no matching pending record existed (or one existed but was
// discarded as stale) -- the caller proceeds with its own ordinary
// fresh-verification path exactly as if this check had never run. Handled
// is true whenever the caller should return immediately using the other
// fields: either a successfully replayed advance, or an ordinary
// blocked/superseded result (the colony is still paused).
type pendingContinueReplayOutcome struct {
	Handled      bool
	Result       map[string]interface{}
	State        colony.ColonyState
	Phase        colony.Phase
	NextPhase    *colony.Phase
	Housekeeping *signalHousekeepingResult
	Final        bool
	Err          error
}

// replayPendingContinueAdvance is the shared replay half of the FIELD-04
// preserve/replay mechanism. Both runCodexContinue and
// runCodexContinueFinalize call this at their natural entry point, before
// any expensive fresh verification/watcher-dispatch work begins, so a
// colony resumed after a pause applies an already-completed, already-passing
// result instead of re-running it.
func replayPendingContinueAdvance(state colony.ColonyState, phase colony.Phase, source string, now time.Time) pendingContinueReplayOutcome {
	record, ok := loadPendingContinueAdvance(phase.ID, state.BuildStartedAt)
	if !ok {
		return pendingContinueReplayOutcome{Handled: false}
	}

	advanceResult, err := advancePhase(advancePhaseParams{
		PhaseID:                phase.ID,
		ExpectedBuildStartedAt: state.BuildStartedAt,
		AllowedStates:          []colony.State{colony.StateEXECUTING, colony.StateBUILT},
		Source:                 source,
		Now:                    now,
	})
	if err != nil {
		if !errors.Is(err, errRuntimeStateSuperseded) {
			return pendingContinueReplayOutcome{Handled: true, Err: fmt.Errorf("failed to atomically advance phase: %w", err)}
		}
		// Still can't commit. Only a still-paused colony keeps the record
		// for a later retry -- any other reason means identity has
		// genuinely gone stale in the moment between the match above and
		// this commit attempt, so it must be discarded, never replayed
		// (T-191.1-02-03).
		latest, loadErr := loadActiveColonyState()
		if loadErr == nil && latest.Paused {
			return pendingContinueReplayOutcome{
				Handled: true,
				Result:  continueSupersededResult(state, phase, err),
				State:   state,
				Phase:   phase,
			}
		}
		clearPendingContinueAdvance(phase.ID)
		return pendingContinueReplayOutcome{Handled: false}
	}

	clearPendingContinueAdvance(phase.ID)
	updated := advanceResult.Updated
	completedPhase := phase
	if idx := phase.ID - 1; idx >= 0 && idx < len(updated.Plan.Phases) {
		completedPhase = updated.Plan.Phases[idx]
	}
	payload := record.Payload
	summary := fmt.Sprintf("Phase %d verified and advanced (replayed after a colony pause)", phase.ID)
	if payload.Assessment.PartialSuccess {
		summary = fmt.Sprintf("Phase %d verified and advanced with partial operational success (replayed after a colony pause)", phase.ID)
	}
	result := map[string]interface{}{
		"advanced":             true,
		"completed":            advanceResult.Final,
		"partial_success":      payload.Assessment.PartialSuccess,
		"current_phase":        updated.CurrentPhase,
		"state":                updated.State,
		"next":                 advanceResult.NextCommand,
		"review_depth":         string(payload.ReviewDepth),
		"verification":         payload.Verification,
		"assessment":           payload.Assessment,
		"task_evidence":        payload.Assessment.Tasks,
		"gates":                payload.Gates,
		"review":               payload.Review,
		"operational_issues":   payload.Assessment.OperationalIssues,
		"recovery":             payload.Assessment.Recovery,
		"reconciled_tasks":     payload.Assessment.ReconciledTasks,
		"replayed_after_pause": true,
		"summary":              summary,
	}
	if advanceResult.NextPhase != nil {
		result["next_phase"] = advanceResult.NextPhase.ID
		result["next_phase_name"] = advanceResult.NextPhase.Name
	}
	return pendingContinueReplayOutcome{
		Handled:   true,
		Result:    result,
		State:     updated,
		Phase:     completedPhase,
		NextPhase: advanceResult.NextPhase,
		Final:     advanceResult.Final,
	}
}
