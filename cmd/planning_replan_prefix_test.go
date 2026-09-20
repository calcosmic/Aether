package cmd

import (
	"reflect"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

// TestReplanAfterFinishedPhaseIsAcceptable is the replan the Route-Setter is
// told to make once some phases are finished (renderPlanRevisionWorkerAppendix):
// send only the replacement phases, number their tasks locally from 1.1, and
// Aether keeps the finished phases and places the new work after them. The
// drafting contract judged that proposal against a history that still held the
// finished phases, so it demanded they be removed -- and a finished phase
// "removed" at drafting but restored when the candidate is built is a plan
// acceptance refuses. The replan must be accepted with the finished phase kept,
// still finished, and the local dependency pointing at the right task.
func TestReplanAfterFinishedPhaseIsAcceptable(t *testing.T) {
	saveGlobals(t)
	goal := "Replan after finishing the first phase"
	firstTaskID, secondTaskID := "1.1", "2.1"
	accepted := createApprovedAcceptedBuildTestColony(t, colony.ColonyState{
		Version: "3.0", Goal: &goal, State: colony.StateREADY, CurrentPhase: 2,
		Plan: colony.Plan{Phases: []colony.Phase{
			{ID: 1, Name: "Finished foundation", Status: colony.PhaseCompleted,
				Tasks: []colony.Task{{ID: &firstTaskID, Goal: "Lay the foundation", Status: colony.TaskCompleted}}},
			{ID: 2, Name: "Unfinished follow-up", Status: colony.PhaseReady,
				Tasks: []colony.Task{{ID: &secondTaskID, Goal: "Follow up", Status: colony.TaskPending}}},
		}},
	})
	root := accepted.Root
	before := mustReadSpecificationTestState(t, root).Plan
	finished, unfinished := before.Phases[0], before.Phases[1]
	if finished.Status != colony.PhaseCompleted {
		t.Fatalf("fixture's first phase is %q, want a finished phase to replan around", finished.Status)
	}

	manifest, result := planningRouteStageTestRunFromState(t, root, "planning-route-replan")
	planningRouteStageSetPolicy(t, root, manifest.RunID, 70, 6)
	result.Proposal = planningReplanSuffixProposal(result.Proposal, unfinished)

	candidate, err := planningParityDraft(t, root, manifest, result)
	if err != nil {
		t.Fatalf("drafting refused the replan the Route-Setter's own brief asks for: %v", err)
	}
	if err := planningParityAccept(root, candidate); err != nil {
		t.Fatalf("drafting admitted this replan, so acceptance must accept it: %v", err)
	}

	after := mustReadSpecificationTestState(t, root).Plan.Phases
	if len(after) != 2 || after[0].SemanticID != finished.SemanticID || after[0].Status != colony.PhaseCompleted {
		t.Fatalf("accepted replan phases = %+v, want the finished phase kept first and still finished", planningReplanPhaseSummary(after))
	}
	replacement := after[1]
	if len(replacement.Tasks) != 2 || ptrStr(replacement.Tasks[0].ID) != "2.1" || ptrStr(replacement.Tasks[1].ID) != "2.2" {
		t.Fatalf("replacement work was not placed after the finished phase: %+v", planningReplanPhaseSummary(after))
	}
	if !reflect.DeepEqual(replacement.Tasks[1].DependsOn, []string{"2.1"}) {
		t.Fatalf("local dependency 1.1 became %v, want 2.1 -- the replacement task it named", replacement.Tasks[1].DependsOn)
	}
}

// planningReplanSuffixProposal shapes a Route-Setter result the way its brief
// asks during a replan: only replacement work, tasks numbered locally from 1.1,
// a local dependency, and explicit removals for the unfinished phase it
// replaces. The finished phase is not mentioned at all.
func planningReplanSuffixProposal(template planningRoutePlanProposal, unfinished colony.Phase) planningRoutePlanProposal {
	proposal := planningRoutePlanProposal{SemanticID: template.SemanticID}
	phase := clonePhases(template.Phases[:1])[0]
	phase.ID = 1
	phase.SemanticID = "phase-replan-follow-through"
	phase.Name, phase.Description = "Follow-through", "Replace the unfinished follow-up"
	first := phase.Tasks[0]
	firstID, secondID := "1.1", "1.2"
	first.ID, first.SemanticID, first.Goal = &firstID, "task-replan-rebuild", "Rebuild the follow-up"
	second := first
	second.ID, second.SemanticID, second.Goal = &secondID, "task-replan-verify", "Verify the rebuilt follow-up"
	second.PublicPathProofLinks = nil
	second.DependsOn = []string{"1.1"}
	phase.Tasks = []colony.Task{first, second}
	proposal.Phases = []colony.Phase{phase}
	proposal.UserFacingSemanticIDs = []string{phase.SemanticID, first.SemanticID}
	proposal.TaskDeclarations = []planningRouteTaskDeclaration{
		{TaskSemanticID: first.SemanticID, Files: []string{"cmd/codex_plan_finalize.go"}, UserFacing: true},
		{TaskSemanticID: second.SemanticID, Files: []string{".gitignore"}},
	}
	proposal.Removals = append(proposal.Removals, planningRouteRemoval{
		SemanticID: unfinished.SemanticID, Classification: colony.PlanningSemanticChangeRemoved,
		Rationale: "The replan replaces this unfinished phase.",
	})
	for _, task := range unfinished.Tasks {
		proposal.Removals = append(proposal.Removals, planningRouteRemoval{
			SemanticID: task.SemanticID, Classification: colony.PlanningSemanticChangeRemoved,
			Rationale: "The replan replaces this unfinished task.",
		})
	}
	return proposal
}

// planningRouteStageTestRunFromState drives a new planning run in a repository
// that already holds an approved specification and a plan, deriving the
// specification binding and base plan revision exactly as the runtime does.
func planningRouteStageTestRunFromState(t *testing.T, root, runID string) (planningStageManifest, planningRouteStageResult) {
	t.Helper()
	state := mustReadSpecificationTestState(t, root)
	if state.Specification == nil {
		t.Fatal("repository has no specification")
	}
	current, ok := currentSpecificationRevision(*state.Specification)
	if !ok || current.Approval == nil {
		t.Fatal("repository has no approved specification revision")
	}
	approvalHash, err := jsonSHA256(*current.Approval)
	if err != nil {
		t.Fatal(err)
	}
	baseHash, err := planStateHash(state.Plan)
	if err != nil {
		t.Fatal(err)
	}
	baseID, baseHash := planningBaseRevisionIdentity(state.Plan, baseHash)
	binding := planningStageSpecificationBinding{
		RevisionID: current.ID, ContentHash: current.ContentHash, Status: colony.SpecStatusApproved,
		ApprovalReceiptID: current.Approval.ID, ApprovalReceiptHash: approvalHash,
	}
	return planningRouteStageTestRun(t, root, current, binding, baseID, baseHash, runID)
}

func planningReplanPhaseSummary(phases []colony.Phase) []string {
	summary := make([]string, 0, len(phases))
	for _, phase := range phases {
		line := phase.SemanticID + "(" + string(phase.Status) + "):"
		for _, task := range phase.Tasks {
			line += " " + ptrStr(task.ID) + "->" + task.SemanticID
		}
		summary = append(summary, line)
	}
	return summary
}
