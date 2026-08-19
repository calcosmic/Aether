package cmd

import (
	"fmt"
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
