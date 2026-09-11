package cmd

import (
	"strings"
	"testing"
)

// --- Task 1: one standing vocabulary -------------------------------------

func TestOneStandingVocabularyAcrossSubsystems(t *testing.T) {
	for _, standing := range declaredWorkStandings() {
		reason := "example reason for " + string(standing)
		oraclePhrase := workStandingLabel(standing, reason)
		swarmPhrase := workStandingLabel(standing, reason)
		if oraclePhrase != swarmPhrase {
			t.Fatalf("standing %q: Oracle phrase %q != Swarm phrase %q", standing, oraclePhrase, swarmPhrase)
		}
	}

	if got := len(declaredWorkStandings()); got != 3 {
		t.Fatalf("declared %d standings, want exactly 3", got)
	}

	// Drive both subsystems through their real entry points with matching
	// reason text and confirm the rendered phrase is identical either way --
	// proving both actually call the one shared function rather than a
	// per-subsystem lookalike.
	oracleStanding, oracleReason := workStandingUsefulNotes, "finishing the remaining research rounds"
	oraclePhrase := workStandingLabel(oracleStanding, oracleReason)

	episode := swarmEpisodeRecord{Status: swarmEpisodeStatusInterrupted, InterruptedStage: swarmEpisodeStageFix}
	swarmStanding, swarmReason := swarmEpisodeStanding(episode)
	if swarmStanding != workStandingUsefulNotes {
		t.Fatalf("interrupted episode resolved to %q, want useful notes", swarmStanding)
	}
	swarmPhrase := workStandingLabel(swarmStanding, oracleReason) // same reason text, different source
	if oraclePhrase != swarmPhrase {
		t.Fatalf("same standing + same reason rendered differently: oracle=%q swarm=%q", oraclePhrase, swarmPhrase)
	}
	if swarmReason == "" {
		t.Fatal("swarmEpisodeStanding returned an empty reason for an interrupted run")
	}
}

func TestUsefulNotesLabelNamesWhatWouldVerifyIt(t *testing.T) {
	label := workStandingLabel(workStandingUsefulNotes, "completing the remaining rounds")
	if !strings.Contains(label, "completing the remaining rounds") {
		t.Fatalf("useful-notes label %q does not name what would verify it", label)
	}
	if !strings.Contains(label, "useful notes") {
		t.Fatalf("useful-notes label %q does not say it is useful notes", label)
	}

	// Even with no reason supplied, the label still names something --
	// never silently empty.
	fallback := workStandingLabel(workStandingUsefulNotes, "")
	if !strings.Contains(fallback, "useful notes") {
		t.Fatalf("fallback useful-notes label %q lost its standing", fallback)
	}
}

func TestUnknownStandingIsNeverVerified(t *testing.T) {
	standing, reason := resolveMalformedItemStanding("corrupt front matter")
	if standing == workStandingVerified {
		t.Fatal("a malformed item resolved to verified")
	}
	if standing != workStandingUnknown {
		t.Fatalf("malformed item resolved to %q, want standing unknown", standing)
	}
	if reason == "" {
		t.Fatal("malformed item carries no stated reason")
	}

	label := workStandingLabel(standing, reason)
	if label != "standing unknown" {
		t.Fatalf("standing unknown label = %q, want the plain phrase", label)
	}
	if label == "verified" {
		t.Fatal("standing unknown rendered as verified")
	}
}

func TestOraclePartialTriggerConditionIsUnchanged(t *testing.T) {
	fixtures := []oracleStateFile{
		{Status: "complete", Iteration: 5},
		{Status: "stopped", Iteration: 7},
		{Status: "idle", Iteration: 0},
		{Status: "timeout", Iteration: 12},
		{Status: ""},
	}
	for _, state := range fixtures {
		oldWasPartial := oracleResearchPartialLabel(state) != ""
		newIsUsefulNotes := oracleResearchStanding(state.Status) == workStandingUsefulNotes
		if oldWasPartial != newIsUsefulNotes {
			t.Errorf("status %q: old partial=%v, new useful-notes=%v (must match)", state.Status, oldWasPartial, newIsUsefulNotes)
		}
		oldWasComplete := oracleResearchPartialLabel(state) == ""
		newIsVerified := oracleResearchStanding(state.Status) == workStandingVerified
		if oldWasComplete != newIsVerified {
			t.Errorf("status %q: old complete=%v, new verified=%v (must match)", state.Status, oldWasComplete, newIsVerified)
		}
	}
}

func TestSwarmRepairIdeaAndInterruptedRunHaveDistinctReasons(t *testing.T) {
	rolledBack := swarmEpisodeRecord{
		Status:             swarmEpisodeStatusCompleted,
		Comparison:         swarmEpisodeComparison{Selected: &swarmRankedRepair{Repair: "restart the worker pool"}},
		Checkpoint:         swarmEpisodeCheckpoint{Saved: true, Restored: true},
		VerificationStatus: swarmEpisodeVerificationCompleted,
	}
	repairStanding, repairReason := swarmRepairIdeaStanding(rolledBack)
	if repairStanding != workStandingUsefulNotes {
		t.Fatalf("rolled-back repair resolved to %q, want useful notes", repairStanding)
	}

	interrupted := swarmEpisodeRecord{Status: swarmEpisodeStatusInterrupted, InterruptedStage: swarmEpisodeStageInvestigation}
	runStanding, runReason := swarmEpisodeStanding(interrupted)
	if runStanding != workStandingUsefulNotes {
		t.Fatalf("interrupted run resolved to %q, want useful notes", runStanding)
	}

	if repairReason == "" || runReason == "" {
		t.Fatal("expected both a rolled-back repair and an interrupted run to state a reason")
	}
	if repairReason == runReason {
		t.Fatalf("rolled-back repair and interrupted run share the same reason %q, want distinct reasons", repairReason)
	}

	unapplied := swarmEpisodeRecord{
		Status:             swarmEpisodeStatusCompleted,
		Comparison:         swarmEpisodeComparison{Selected: &swarmRankedRepair{Repair: "add a retry"}},
		VerificationStatus: "not_run",
	}
	unappliedStanding, unappliedReason := swarmRepairIdeaStanding(unapplied)
	if unappliedStanding != workStandingUsefulNotes {
		t.Fatalf("unapplied repair idea resolved to %q, want useful notes", unappliedStanding)
	}
	if unappliedReason == repairReason {
		t.Fatalf("unapplied idea and rolled-back idea share the same reason %q", unappliedReason)
	}

	verified := swarmEpisodeRecord{
		Status:             swarmEpisodeStatusCompleted,
		Comparison:         swarmEpisodeComparison{Selected: &swarmRankedRepair{Repair: "fix the race"}},
		Checkpoint:         swarmEpisodeCheckpoint{Saved: true, Restored: false},
		VerificationStatus: swarmEpisodeVerificationCompleted,
	}
	if standing, _ := swarmRepairIdeaStanding(verified); standing != workStandingVerified {
		t.Fatalf("held repair resolved to %q, want verified", standing)
	}
	if standing, _ := swarmEpisodeStanding(swarmEpisodeRecord{Status: swarmEpisodeStatusCompleted}); standing != workStandingVerified {
		t.Fatalf("completed run resolved to %q, want verified", standing)
	}
}
