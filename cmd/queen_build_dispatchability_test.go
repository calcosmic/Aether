package cmd

import (
	"testing"
)

// TestEveryBuildSelectableCasteCanDispatch is an invariant, not a named-caste
// check: a caste the Queen may select for a build must be able to produce at
// least one dispatch in the build composer. route_setter passed the flow
// filter but appeared in no dispatch table, so selecting it consumed a budget
// slot and silently displaced a specialist that would actually have run. If a
// new caste is ever added to the registry and allowed for build without a
// dispatch route, this fails naming it.
func TestEveryBuildSelectableCasteCanDispatch(t *testing.T) {
	dispatchable := map[string]bool{
		// Composed outside the wave-plan tables:
		"watcher": true, // final gate wave
		"probe":   true, // coverage dispatch when selected
		"builder": true, // task-caste default
	}
	for _, plan := range queenBuildPreWavePlans {
		dispatchable[plan.caste] = true
	}
	for _, plan := range queenBuildPostWavePlans {
		dispatchable[plan.caste] = true
	}
	for _, caste := range queenBuildTaskFallbackCastes {
		dispatchable[caste] = true
	}

	for _, profile := range casteRelevanceRegistry {
		if !casteAllowedForFlow(profile.Caste, "build") {
			continue
		}
		if !dispatchable[profile.Caste] {
			t.Errorf("caste %q is selectable for build but no build dispatch path can spawn it — it would waste a budget slot", profile.Caste)
		}
	}
}
