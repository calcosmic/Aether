package cmd

import (
	"regexp"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

var phaseZeroCommand = regexp.MustCompile(`\bbuild 0\b`)

// TestNoCloseoutRecommendsPhaseZero: on 2026-10-02 the owner's closing card
// recommended "/ant-build 0 --force" after a check moved the project from
// phase 2 to phase 3. A verdict with no recorded build attempt must never be
// turned into a command for phase 0.
func TestNoCloseoutRecommendsPhaseZero(t *testing.T) {
	for _, verdict := range colony.AllWorkOutcomes() {
		action, err := recommendedActionForWorkOutcome(verdict, buildAttemptRecord{})
		if err != nil {
			t.Fatalf("%s: %v", verdict, err)
		}
		if phaseZeroCommand.MatchString(action.Command) {
			t.Errorf("%s with no recorded attempt recommends a phase-0 command: %q", verdict, action.Command)
		}
	}
}

// TestCloseoutWithNoAttemptToRedoNamesTheSharedNextStep: when there is no
// build attempt to redo for the phase the project now stands on (a check that
// moved the project forward), the closing card's recommendation must be the
// same shared next step the Next Up line shows, never a second answer.
func TestCloseoutWithNoAttemptToRedoNamesTheSharedNextStep(t *testing.T) {
	saveGlobals(t)
	s, _ := newTestStore(t)
	store = s

	projection := LifecycleProjection{
		ProjectionRevision: "test-revision",
		Phase:              LifecycleFact[LifecyclePhaseProjection]{Value: LifecyclePhaseProjection{CurrentNumber: 3, TotalPhases: 6}},
		NextAction:         LifecycleProjectedAction{ID: "build", RuntimeCommand: "aether build 3", Reason: "phase 3 is ready to build"},
	}
	for _, verdict := range []colony.WorkOutcome{colony.WorkOutcomePartial, colony.WorkOutcomeTimeout} {
		closeout, err := buildLifecycleCloseout(projection, "continue", colony.OutcomeKindInProgress, colony.LifecycleStateEffectNone, LifecycleCloseoutDetails{WorkOutcome: verdict})
		if err != nil {
			t.Fatalf("%s: build closeout: %v", verdict, err)
		}
		if closeout.RecommendedAction == nil {
			t.Fatalf("%s: closeout carries no recommended action", verdict)
		}
		if got := closeout.RecommendedAction.Command; got != projection.NextAction.RuntimeCommand {
			t.Errorf("%s: closing card recommends %q while Next Up says %q", verdict, got, projection.NextAction.RuntimeCommand)
		}
	}
}
