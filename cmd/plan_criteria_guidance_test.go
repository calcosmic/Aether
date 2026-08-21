package cmd

import (
	"strings"
	"testing"
)

// TestPlanGuidanceWarnsAboutUnclaimedArtifactCriteria is the WP-6 gate.
//
// Route-Setter plans routinely bound "npm test still passes" and "package.json
// stays dependency-free" criteria to artifacts the task never modified. The
// runtime then blocks the phase, and the only recovery is a pair of expert
// flags (--reconcile-task, --read-only-artifact) a non-technical operator
// cannot be expected to discover. Three of the four phases in the v1.0.47
// acceptance run needed exactly that rescue.
//
// This is an invariant test, not a "string exists" test: it triggers the real
// runtime failure and asserts the planner guidance quotes the message the
// runtime actually emits. If that message is ever reworded, the guidance goes
// stale and this fails.
func TestPlanGuidanceWarnsAboutUnclaimedArtifactCriteria(t *testing.T) {
	saveGlobals(t)
	phase := criterionEvidenceTestPhase()
	root, manifest := setupCriterionEvidenceTest(t, phase)

	// Strip every claim so the phase's bound artifacts are unclaimed — the
	// shape a plan produces when it binds an "unchanged file" criterion.
	var claims codexBuildClaims
	if err := store.LoadJSON("last-build-claims.json", &claims); err != nil {
		t.Fatalf("load claims: %v", err)
	}
	claims.TaskClaims = nil
	claims.FilesCreated = nil
	claims.FilesModified = nil
	claims.TestsWritten = nil
	claims.ArtifactEvidence = nil
	if err := store.SaveJSON("last-build-claims.json", claims); err != nil {
		t.Fatalf("save empty claims: %v", err)
	}

	evaluation := evaluatePhaseCriterionEvidence(root, phase, manifest, passingCriterionSteps(), codexClaimVerification{Present: true, Passed: true}, codexWatcherVerification{})
	if evaluation.Passed {
		t.Fatal("expected an unclaimed bound artifact to block the phase")
	}
	runtimeMessage := strings.Join(evaluation.BlockingIssues, "\n")
	if !strings.Contains(runtimeMessage, "was not claimed by the current build") {
		t.Fatalf("runtime no longer emits the message the planner guidance quotes; update both:\n%s", runtimeMessage)
	}

	guidance := renderPhasePlanSchemaGuidance()
	for _, want := range []string{
		"Only list a file under artifacts when THIS task changes it",
		"was not claimed by the current build",
		`"checks":["tests"]`,
	} {
		if !strings.Contains(guidance, want) {
			t.Fatalf("planner guidance missing %q — planners will keep authoring unsatisfiable criteria:\n%s", want, guidance)
		}
	}
}

// A criterion that asserts pre-existing state via checks (not artifacts) is
// the shape the guidance steers planners toward, and it must pass.
func TestChecksOnlyCriterionNeedsNoArtifactClaim(t *testing.T) {
	saveGlobals(t)
	phase := criterionEvidenceTestPhase()
	// Rebind every requirement to a checks-only form: no artifacts at all.
	for i := range phase.EvidenceRequirements {
		phase.EvidenceRequirements[i].Artifacts = nil
		phase.EvidenceRequirements[i].Checks = []string{"tests"}
	}
	for ti := range phase.Tasks {
		for ri := range phase.Tasks[ti].EvidenceRequirements {
			phase.Tasks[ti].EvidenceRequirements[ri].Artifacts = nil
			phase.Tasks[ti].EvidenceRequirements[ri].Checks = []string{"tests"}
		}
	}
	root, manifest := setupCriterionEvidenceTest(t, phase)

	var claims codexBuildClaims
	if err := store.LoadJSON("last-build-claims.json", &claims); err != nil {
		t.Fatalf("load claims: %v", err)
	}
	claims.TaskClaims = nil
	claims.FilesCreated = nil
	claims.FilesModified = nil
	claims.TestsWritten = nil
	if err := store.SaveJSON("last-build-claims.json", claims); err != nil {
		t.Fatalf("save empty claims: %v", err)
	}

	evaluation := evaluatePhaseCriterionEvidence(root, phase, manifest, passingCriterionSteps(), codexClaimVerification{Present: true, Passed: true}, codexWatcherVerification{})
	if !evaluation.Passed {
		t.Fatalf("a checks-only criterion must not require artifact claims: %v", evaluation.BlockingIssues)
	}
}
