package cmd

import (
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

// TestColonizeLightDepthTrimsSurveyors: colonize always sent exactly four
// surveyors whatever the repo or the operator's depth choice — the one flow
// where `--verification-depth light` changed nothing at all. Light now sends
// only the structure and dependency surveyors; standard keeps all four.
func TestColonizeLightDepthTrimsSurveyors(t *testing.T) {
	light := colony.ColonyState{VerificationDepth: string(colony.VerificationDepthLight)}
	standard := colony.ColonyState{}

	trimmed := queenSurveyorSpecsForState(light)
	if len(trimmed) != 2 {
		t.Fatalf("light colonize should send exactly two surveyors, got %d: %+v", len(trimmed), trimmed)
	}
	got := map[string]bool{}
	for _, spec := range trimmed {
		got[spec.Caste] = true
	}
	if !got["surveyor-nest"] || !got["surveyor-provisions"] {
		t.Fatalf("light colonize should keep the structure and dependency surveyors, got %+v", got)
	}

	full := queenSurveyorSpecsForState(standard)
	if len(full) != 4 {
		t.Fatalf("standard colonize must keep all four surveyors, got %d: %+v", len(full), full)
	}
}

// TestLightColonizeStillWritesItsSurvey: trimming the roster is only useful
// if the survey then runs. The output requirement demanded all seven survey
// documents whatever the roster, so a light colonize aborted with "missing
// required surveyor output DISCIPLINES.md" and wrote nothing at all — the
// depth choice did not just trim the team, it broke the command.
func TestLightColonizeStillWritesItsSurvey(t *testing.T) {
	dispatchesFor := func(state colony.ColonyState) ([]codexSurveyorDispatch, []surveyorSpec) {
		specs := queenSurveyorSpecsForState(state)
		dispatches := make([]codexSurveyorDispatch, 0, len(specs))
		for i, spec := range specs {
			dispatches = append(dispatches, surveyDispatchFromSpec(t.TempDir(), spec, i))
		}
		return dispatches, specs
	}

	light, lightRoster := dispatchesFor(colony.ColonyState{VerificationDepth: string(colony.VerificationDepthLight)})
	byOutput, err := surveyDispatchesByRequiredOutput(light, lightRoster)
	if err != nil {
		t.Fatalf("a light survey must be able to write the documents its own roster promised: %v", err)
	}
	for _, name := range []string{"BLUEPRINT.md", "CHAMBERS.md", "PROVISIONS.md", "TRAILS.md"} {
		if _, ok := byOutput[name]; !ok {
			t.Fatalf("light survey lost %s, which its dispatched surveyors own", name)
		}
	}
	for _, name := range []string{"DISCIPLINES.md", "PATHOGENS.md"} {
		if _, ok := byOutput[name]; ok {
			t.Fatalf("light survey claims %s, but no dispatched surveyor produces it", name)
		}
	}

	fullDispatches, fullRoster := dispatchesFor(colony.ColonyState{})
	full, err := surveyDispatchesByRequiredOutput(fullDispatches, fullRoster)
	if err != nil {
		t.Fatalf("standard survey must still resolve every required output: %v", err)
	}
	if len(full) != len(requiredSurveyMarkdownFiles) {
		t.Fatalf("standard survey covered %d of %d required outputs", len(full), len(requiredSurveyMarkdownFiles))
	}

	// A surveyor that was dispatched still owes every one of its own files.
	incomplete, incompleteRoster := dispatchesFor(colony.ColonyState{})
	for i := range incomplete {
		if incomplete[i].Caste == "surveyor-nest" {
			incomplete[i].Outputs = []string{"BLUEPRINT.md"}
		}
	}
	if _, err := surveyDispatchesByRequiredOutput(incomplete, incompleteRoster); err == nil {
		t.Fatal("a dispatched surveyor that drops one of its own outputs must still be rejected")
	}

	if _, err := surveyDispatchesByRequiredOutput(nil, nil); err == nil {
		t.Fatal("a survey with no surveyors must be rejected, not silently accepted as complete")
	}

	// A surveyor the roster planned but that never turned up in the results
	// is still a missing output -- the light-depth relaxation must not become
	// "whatever came back is by definition complete".
	dropped := make([]codexSurveyorDispatch, 0, len(fullDispatches))
	for _, dispatch := range fullDispatches {
		if dispatch.Caste == "surveyor-pathogens" {
			continue
		}
		dropped = append(dropped, dispatch)
	}
	if _, err := surveyDispatchesByRequiredOutput(dropped, fullRoster); err == nil {
		t.Fatal("a surveyor the roster planned but that never reported must still be caught")
	}
}
