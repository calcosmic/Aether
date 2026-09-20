package cmd

import (
	"strings"
	"testing"
)

// TestPaddedLinkIDIsRefusedWithAClearReason covers a link ID sent with stray
// spaces. Drafting trimmed it before resolving it, but the candidate stored it
// as sent and acceptance compares exactly, so the owner could be handed a plan
// acceptance refused. Drafting now refuses it, and the refusal must say how to
// fix it rather than claim the specification item does not exist.
func TestPaddedLinkIDIsRefusedWithAClearReason(t *testing.T) {
	root, manifest, result := planningRouteStageTestFixture(t)
	planningRouteStageSetPolicy(t, root, manifest.RunID, 70, 6)
	task := &result.Proposal.Phases[0].Tasks[0]
	task.RequirementProofLinks = []string{" " + task.RequirementProofLinks[0] + " "}

	before := planCandidateTestSnapshot(t, root)
	_, err := coordinatePlanningRouteStage(root, manifest, planningRouteStageTestBytes(t, result))
	if err == nil || !strings.Contains(err.Error(), "exactly as the approved specification spells it") {
		t.Fatalf("drafting error = %v, want a refusal that says to copy the ID exactly as the specification spells it", err)
	}
	planCandidateTestAssertSnapshot(t, root, before)
}
