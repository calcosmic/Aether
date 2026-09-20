package cmd

import (
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

// Reported from a real colony (M4L-AnalogWave-System, 2026-08-21): a nine-task
// phase planned six workers because three dependent tasks were folded into
// their predecessors' chains. The plan said "(+2 more steps)" but never named
// which task IDs those were, so the operator concluded three tasks had "no
// slot" and hand-reconciled work that already had a proper claimant -- a long,
// wrong recovery caused purely by the plan not saying what it had done.
func TestSpawnPlanNamesTheTasksAMergedWorkerCovers(t *testing.T) {
	dispatches := []codexBuildDispatch{
		{
			Stage: "wave", Wave: 1, ExecutionWave: 11, Caste: "builder", Name: "Anvil-81",
			Task:           "1. Repair the bridge\n2. Widen the excerpt\n3. Add the wiring test",
			TaskID:         "1.1",
			CoveredTaskIDs: []string{"1.1", "1.2", "1.9"},
			Status:         "spawned",
		},
		{
			Stage: "wave", Wave: 2, ExecutionWave: 12, Caste: "watcher", Name: "Keen-6",
			Task: "Independent verification", TaskID: "", Status: "spawned",
		},
	}

	plan := renderSpawnPlanForDispatches(dispatches, colony.ModeInRepo)

	for _, taskID := range []string{"1.1", "1.2", "1.9"} {
		if !strings.Contains(plan, taskID) {
			t.Fatalf("spawn plan never names covered task %s -- an operator cannot tell it has a claimant:\n%s", taskID, plan)
		}
	}
	if !strings.Contains(plan, "every task still has a claimant") {
		t.Fatalf("spawn plan does not say the folded tasks are still claimed:\n%s", plan)
	}
}

// A plan with no merged chains must stay exactly as plain as it was: the note
// is for folding, not decoration on every build.
func TestSpawnPlanStaysQuietWhenNothingWasFolded(t *testing.T) {
	dispatches := []codexBuildDispatch{
		{Stage: "wave", Wave: 1, ExecutionWave: 11, Caste: "builder", Name: "Anvil-81", Task: "Do the thing", TaskID: "1.1", Status: "spawned"},
	}
	plan := renderSpawnPlanForDispatches(dispatches, colony.ModeInRepo)
	if strings.Contains(plan, "covers tasks") || strings.Contains(plan, "folded into") {
		t.Fatalf("a plan with no merged chain must not mention folding:\n%s", plan)
	}
}

// A colonize rendered four identical "📊🐜 Surveyor <name>" lines for four
// different specialists, because every surveyor sub-caste collapsed to one
// label. The operator could only tell them apart by reading the task text.
func TestEachSurveyorSaysWhatItSurveys(t *testing.T) {
	labels := map[string]string{}
	for _, caste := range []string{"surveyor-nest", "surveyor-provisions", "surveyor-disciplines", "surveyor-pathogens"} {
		label := casteLabel(caste)
		if label == "Ant" {
			t.Fatalf("%s has no label at all", caste)
		}
		if previous, clash := labels[label]; clash {
			t.Fatalf("%s and %s both render as %q -- four different specialists must not look identical", previous, caste, label)
		}
		labels[label] = caste
		if !strings.Contains(label, "Surveyor") {
			t.Fatalf("%s label %q no longer reads as a surveyor", caste, label)
		}
		if casteEmoji(caste) != "📊" {
			t.Fatalf("%s lost the surveyor family glyph, got %q", caste, casteEmoji(caste))
		}
	}
	// An unknown surveyor variant still falls back to the family identity.
	if label := casteLabel("surveyor-future"); label != "Surveyor" {
		t.Fatalf("an unrecognised surveyor variant must fall back to the family label, got %q", label)
	}
}
