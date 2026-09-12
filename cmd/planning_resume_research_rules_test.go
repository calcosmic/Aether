package cmd

import (
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

// planningScoutEvidenceRulePhrases are the parts of the automatic research
// evidence contract a Scout must be told: turn every source into typed
// evidence with its revision and content hash, and never approve anything.
var planningScoutEvidenceRulePhrases = []string{"typed planning evidence", "source revision", "content hash", "must not approve"}

// TestSecondRoundScoutCarriesTheEvidenceRules covers the path every second and
// later research round takes. The resume path rebuilt the Scout's brief from
// scratch and never appended the research policy, so a Scout sent back for
// another round was not told how to record evidence or that it may never
// approve a specification, accept a candidate, or activate a plan.
func TestSecondRoundScoutCarriesTheEvidenceRules(t *testing.T) {
	root, _, _ := planningStageResumeTestSecondScoutPass(t)
	resume, err := resolvePlanningStageResumeForTest(t, root)
	if err != nil || resume == nil {
		t.Fatalf("resolve second-round Scout: resume=%v err=%v", resume, err)
	}
	goal := "resume the second research round"
	manifest, err := buildPlanningStageResumeManifest(
		root, colony.ColonyState{Goal: &goal}, *resume, colony.GranularitySprint,
		"balanced", "standard", "standard",
		codexSurveyContext{}, "", codexPlanOptions{}, time.Now().UTC(),
	)
	if err != nil {
		t.Fatalf("build resumed Scout manifest: %v", err)
	}
	if len(manifest.Dispatches) != 1 {
		t.Fatalf("resumed manifest has %d dispatches, want one Scout", len(manifest.Dispatches))
	}
	brief := strings.ToLower(manifest.Dispatches[0].Brief)
	for _, want := range planningScoutEvidenceRulePhrases {
		if !strings.Contains(brief, want) {
			t.Errorf("second-round Scout brief is missing %q", want)
		}
	}
}
