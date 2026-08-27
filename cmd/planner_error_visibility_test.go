package cmd

import (
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

// cyclicPhase has two tasks that each wait for the other, which the planner's
// own preflight refuses by name.
func cyclicPhase() colony.Phase {
	a, b := "1.1", "1.2"
	return colony.Phase{
		ID:     1,
		Name:   "Circular dependency",
		Status: colony.PhaseReady,
		Tasks: []colony.Task{
			{ID: &a, Goal: "First step", Status: colony.TaskPending, DependsOn: []string{b}},
			{ID: &b, Goal: "Second step", Status: colony.TaskPending, DependsOn: []string{a}},
		},
	}
}

// TestPlanningRefusalIsNeverShownAsAnEmptyTeam is the permanent regression lock
// for the 195 review's sixth warning (WR-06).
//
// The compatibility shim in front of the planner did `if err != nil { return
// nil }`. Every surface that showed a phase's planned team therefore rendered a
// dependency cycle -- or a task naming a step that does not exist -- as a phase
// with no work to do, instead of the named, actionable error the planner had
// gone to the trouble of producing.
func TestPlanningRefusalIsNeverShownAsAnEmptyTeam(t *testing.T) {
	phase := cyclicPhase()
	state := colony.ColonyState{
		ColonyDepth: "standard",
		Plan:        colony.Plan{Phases: []colony.Phase{phase}},
	}

	// The planner really does refuse this fixture by name. If it stops doing
	// so, everything below proves nothing.
	_, _, err := plannedBuildDispatchesWithJobProposals(
		phase, state, nil, colony.VerificationDepthStandard, nil, "", nil, nil,
	)
	if err == nil {
		t.Fatal("the planner no longer refuses a circular dependency; this fixture proves nothing")
	}
	if !strings.Contains(err.Error(), "1.1") || !strings.Contains(err.Error(), "1.2") {
		t.Fatalf("the planner's refusal no longer names the offending steps: %v", err)
	}

	t.Run("the spawn plan names the refusal", func(t *testing.T) {
		out := renderSpawnPlan(phase, "standard")
		if !strings.Contains(out, "1.1") || !strings.Contains(out, "1.2") {
			t.Fatalf("a phase that cannot be planned renders as an ordinary spawn plan with no explanation:\n%s", out)
		}
	})

	t.Run("the build screen names the refusal", func(t *testing.T) {
		out := renderBuildVisual(state, phase)
		if !strings.Contains(out, "1.1") || !strings.Contains(out, "1.2") {
			t.Fatalf("a phase that cannot be planned renders as an ordinary build screen with an empty team:\n%s", out)
		}
	})
}
