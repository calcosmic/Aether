package cmd

import "github.com/calcosmic/Aether/pkg/colony"

// These four helpers exist only so the many tests that plan a KNOWN-GOOD phase
// stay readable after the production planners started returning their refusal
// instead of swallowing it (WR-06, 195-REVIEW.md).
//
// They are deliberately test-only. A test that cares about a planning refusal
// calls the production function and asserts on the error -- see
// TestPlanningRefusalIsNeverShownAsAnEmptyTeam.

func testPlannedBuildDispatches(phase colony.Phase, depth string) []codexBuildDispatch {
	dispatches, _ := plannedBuildDispatches(phase, depth)
	return dispatches
}

func testPlannedBuildDispatchesForSelection(phase colony.Phase, depth string, selectedTaskIDs []string, reviewDepth colony.VerificationDepth) []codexBuildDispatch {
	dispatches, _ := plannedBuildDispatchesForSelection(phase, depth, selectedTaskIDs, reviewDepth)
	return dispatches
}

func testPlannedBuildDispatchesForSelectionWithState(phase colony.Phase, state colony.ColonyState, selectedTaskIDs []string, reviewDepth colony.VerificationDepth) []codexBuildDispatch {
	dispatches, _ := plannedBuildDispatchesForSelectionWithState(phase, state, selectedTaskIDs, reviewDepth)
	return dispatches
}

func testPlannedBuildDispatchesWithJudgement(phase colony.Phase, state colony.ColonyState, selectedTaskIDs []string, reviewDepth colony.VerificationDepth, proposedCastes []string, casteReason string, reasons ...map[string]string) []codexBuildDispatch {
	dispatches, _ := plannedBuildDispatchesWithJudgement(phase, state, selectedTaskIDs, reviewDepth, proposedCastes, casteReason, reasons...)
	return dispatches
}
