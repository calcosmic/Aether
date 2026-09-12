package cmd

import (
	"reflect"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

// TestCustomDependencyIDsSurviveToAcceptance covers the renumbering Go does
// when it builds a candidate: every task becomes phase.task by position, but
// only dependencies that already looked numeric were rewritten. A Route-Setter's
// own task names (P1-T1) were left pointing at nothing, and a plan whose phases
// were numbered from 2 had a dependency silently re-pointed at the wrong task --
// either way drafting admitted a plan the owner could then not accept.
func TestCustomDependencyIDsSurviveToAcceptance(t *testing.T) {
	tests := []struct {
		name      string
		shape     func(*planningRouteStageResult)
		dependent [2]int // phase, task of the task that depends on the first task
	}{
		{
			name: "the Route-Setter's own task names",
			shape: func(result *planningRouteStageResult) {
				planningParityAddHousekeepingTask(result)
				phase := &result.Proposal.Phases[0]
				first, second := "P1-T1", "P1-T2"
				phase.Tasks[0].ID, phase.Tasks[1].ID = &first, &second
				phase.Tasks[1].DependsOn = []string{"P1-T1"}
			},
			dependent: [2]int{0, 1},
		},
		{
			name: "phases numbered from 2",
			shape: func(result *planningRouteStageResult) {
				planningParityAddHousekeepingTask(result)
				first := &result.Proposal.Phases[0]
				housekeeping := first.Tasks[1]
				first.Tasks = first.Tasks[:1]
				first.ID = 2
				firstTaskID := "2.1"
				first.Tasks[0].ID = &firstTaskID

				second := *first
				second.ID = 3
				second.SemanticID = "phase-route-housekeeping"
				second.Name, second.Description = "Route housekeeping", "Keep planning staging out of version control"
				second.PublicPathProofLinks = nil
				secondTaskID := "3.1"
				housekeeping.ID = &secondTaskID
				housekeeping.DependsOn = []string{"2.1"}
				second.Tasks = []colony.Task{housekeeping}
				result.Proposal.Phases = append(result.Proposal.Phases, second)
			},
			dependent: [2]int{1, 0},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root, manifest, result := planningRouteStageTestFixture(t)
			planningRouteStageSetPolicy(t, root, manifest.RunID, 70, 6)
			test.shape(&result)

			candidate, err := planningParityDraft(t, root, manifest, result)
			if err != nil {
				t.Fatalf("drafting refused a plan whose dependencies resolve: %v", err)
			}
			if err := planningParityAccept(root, candidate); err != nil {
				t.Fatalf("drafting admitted this plan, so acceptance must accept it: %v", err)
			}
			phases := mustReadSpecificationTestState(t, root).Plan.Phases
			dependent := phases[test.dependent[0]].Tasks[test.dependent[1]]
			if want := []string{ptrStr(phases[0].Tasks[0].ID)}; !reflect.DeepEqual(dependent.DependsOn, want) {
				t.Fatalf("accepted task %s depends on %v, want %v -- the task it named before renumbering", ptrStr(dependent.ID), dependent.DependsOn, want)
			}
		})
	}
}
