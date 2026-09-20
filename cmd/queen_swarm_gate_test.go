package cmd

import (
	"testing"
)

// TestSwarmTrivialBugSkipsHistoryAndResearch pins the swarm half of the
// spawn trimming. Scout and Archaeologist used to be unconditionally
// required for every swarm, so a one-line typo hunt paid for a researcher
// and a git-history dig that could only report finding nothing. They now
// ride on keyword relevance like every other specialist.
func TestSwarmTrivialBugSkipsHistoryAndResearch(t *testing.T) {
	selected := queenSwarmSelectedCastes("fix typo in README header")
	for _, core := range []string{"tracker", "builder", "watcher"} {
		if !selected[core] {
			t.Fatalf("swarm must always keep %s; got %+v", core, selected)
		}
	}
	if selected["archaeologist"] {
		t.Fatalf("a trivial typo hunt must not summon a git-history specialist: %+v", selected)
	}
	if selected["scout"] {
		t.Fatalf("a trivial typo hunt must not summon a research specialist: %+v", selected)
	}
}

// TestSwarmHistoryBugKeepsArchaeologist is the floor under the trim: a bug
// whose wording actually names historical context still selects the
// Archaeologist through ordinary keyword relevance.
func TestSwarmHistoryBugKeepsArchaeologist(t *testing.T) {
	selected := queenSwarmSelectedCastes("regression after legacy migration refactor of the history module")
	if !selected["archaeologist"] {
		t.Fatalf("a bug rooted in legacy/migration history should still select the Archaeologist: %+v", selected)
	}
}

// TestSwarmInvestigationBugKeepsScout: wording that asks for investigation
// still buys the researcher — the trim removes the unconditional floor, not
// the capability.
func TestSwarmInvestigationBugKeepsScout(t *testing.T) {
	selected := queenSwarmSelectedCastes("investigate intermittent parser failure and research prior reports")
	if !selected["scout"] {
		t.Fatalf("an investigation-heavy bug should still select the Scout: %+v", selected)
	}
}
