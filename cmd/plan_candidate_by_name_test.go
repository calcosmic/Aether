package cmd

import (
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

// TestTwoWaitingPlansCanEachBeReviewedByName is the dead end a planning
// restart left in CosmicDashboard (2026-09-12): the earlier plan stayed waiting
// beside the new one, the review refused to pick between them, and no command
// could name either, so neither plan could be accepted. The refusal must name
// every waiting plan and the exact command that reviews it, reviewing by name
// must write nothing, and a plan reviewed by name must then be acceptable.
func TestTwoWaitingPlansCanEachBeReviewedByName(t *testing.T) {
	root, first := planCandidateTestPending(t)
	second := planCandidateTestSecondPending(t, root, "planning-route-stage-restart")

	_, _, err := runPlanCandidateCommand(root, planCandidateCommandInputs{Candidate: true})
	if err == nil {
		t.Fatal("review with two waiting plans picked one without being told which")
	}
	for _, candidate := range []colony.PlanCandidate{first, second} {
		if want := planCandidateReviewByNameCommand(candidate.ID); !strings.Contains(err.Error(), want) {
			t.Fatalf("two-plan refusal = %q, want it to name %q so the owner has a way on", err, want)
		}
	}

	for _, candidate := range []colony.PlanCandidate{first, second} {
		before := planCandidateTestSnapshot(t, root)
		result, handled, err := runPlanCandidateCommand(root, planCandidateCommandInputs{Candidate: true, CandidateID: candidate.ID})
		if err != nil || !handled {
			t.Fatalf("review of %s by name: handled=%t err=%v", candidate.ID, handled, err)
		}
		planCandidateTestAssertSnapshot(t, root, before)
		if got, want := result["acceptance_command"], planCandidateAcceptanceCommand(planCandidateTestAcceptanceRequest(candidate)); got != want {
			t.Fatalf("review of %s offered %v, want its own exact acceptance %q", candidate.ID, got, want)
		}
	}

	if _, err := acceptPlanCandidate(root, planCandidateTestAcceptanceRequest(first), planCandidateAcceptanceOptions{
		AcceptedBy: "owner", AcceptedAt: first.CreatedAt.Add(time.Minute),
	}); err != nil {
		t.Fatalf("accepting a plan reviewed by name, with another still waiting: %v", err)
	}
}

// TestCandidateIDOnlyNamesWhichPlanToReview keeps the new option from becoming
// a second way to accept: on its own it is refused.
func TestCandidateIDOnlyNamesWhichPlanToReview(t *testing.T) {
	_, err := resolvePlanCandidateOperation(planCandidateCommandInputs{CandidateID: "plan-candidate-000000000000"})
	if err == nil || !strings.Contains(err.Error(), "--candidate") {
		t.Fatalf("--candidate-id alone = %v, want a refusal pointing at --candidate", err)
	}
}

// planCandidateTestSecondPending drives a second planning run to a waiting
// candidate in a repository that already holds one -- the state a planning
// restart leaves behind. The specification binding and base plan are derived
// from the repository exactly as the first run derived them.
func planCandidateTestSecondPending(t *testing.T, root, runID string) colony.PlanCandidate {
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
	manifest, result := planningRouteStageTestRun(t, root, current, binding, baseID, baseHash, runID)
	planningRouteStageSetPolicy(t, root, manifest.RunID, 70, 6)
	candidate, err := planningParityDraft(t, root, manifest, result)
	if err != nil {
		t.Fatalf("second planning run did not reach a waiting candidate: %v", err)
	}
	return candidate
}
