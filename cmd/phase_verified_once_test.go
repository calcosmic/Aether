package cmd

import (
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

// This file asserts ruling D11 rule 4 (.planning/decisions/2026-08-22-queen-decides-program-checks.md):
// each phase is verified once. The build-side "verification" stage and the
// follow-up continue review are one pass, not two, and the same caste is not
// dispatched at both boundaries for the same phase unless the Queen asks.
//
// phaseVerifiedOncePhase builds a minimal phase fixture with one task, so the
// dispatch planner has real work to plan around (an empty-task phase takes a
// different, less representative code path).
func phaseVerifiedOncePhase(name, description string, mode colony.PhaseMode) colony.Phase {
	taskID := "1.1"
	return colony.Phase{
		ID:          1,
		Name:        name,
		Description: description,
		Mode:        mode,
		Status:      colony.PhaseReady,
		Tasks: []colony.Task{{
			ID:     &taskID,
			Goal:   description,
			Status: colony.TaskPending,
		}},
	}
}

// TestBuildPlansNoReviewerWithoutAProposal is FLOOR-04's build-side half: a
// build planned with no caste proposal produces zero dispatches whose Stage is
// the verification stage, for a documentation phase, a prototype phase and a
// production phase alike. The program's free checks are the floor; nothing
// implicit reviews the work a second time.
func TestBuildPlansNoReviewerWithoutAProposal(t *testing.T) {
	for _, phase := range []colony.Phase{
		phaseVerifiedOncePhase("Write the README", "Documentation only", colony.PhaseModeMaintenance),
		phaseVerifiedOncePhase("Add a hello endpoint", "Implement the /hello route", colony.PhaseModePrototype),
		phaseVerifiedOncePhase("Ship the release", "Production deploy", colony.PhaseModeProduction),
	} {
		state := colony.ColonyState{Plan: colony.Plan{Phases: []colony.Phase{phase}}}
		dispatches := plannedBuildDispatchesWithJudgement(phase, state, nil, colony.VerificationDepthStandard, nil, "")
		for _, d := range dispatches {
			if d.Stage == "verification" {
				t.Errorf("phase %q (%s): build planned a verification-stage dispatch with no Queen proposal: %+v",
					phase.Name, phase.Mode, d)
			}
		}
	}
}

// TestBuildStillDispatchesAWatcherTheQueenAskedFor proves the owner override
// survives: a production phase whose proposal explicitly names the watcher
// still produces exactly one verification-stage watcher dispatch.
func TestBuildStillDispatchesAWatcherTheQueenAskedFor(t *testing.T) {
	phase := phaseVerifiedOncePhase("Ship the release", "Production deploy", colony.PhaseModeProduction)
	state := colony.ColonyState{Plan: colony.Plan{Phases: []colony.Phase{phase}}}

	dispatches := plannedBuildDispatchesWithJudgement(
		phase, state, nil, colony.VerificationDepthStandard,
		[]string{"builder", "watcher"}, "owner asked for an explicit watcher pass",
	)

	count := 0
	for _, d := range dispatches {
		if d.Stage == "verification" && d.Caste == "watcher" {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("expected exactly one verification-stage watcher dispatch when the Queen's proposal named it, got %d: %+v", count, dispatches)
	}
}
