package cmd

import (
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

func TestCheckPlanGrounding_NoAnchors(t *testing.T) {
	phases := []colony.Phase{
		{ID: 1, Name: "Build API", Tasks: []colony.Task{
			{Goal: "Set up the authentication system"},
		}},
	}
	warnings := checkPlanGrounding(phases, nil)
	if warnings != nil {
		t.Fatalf("expected nil warnings with nil anchors, got %v", warnings)
	}

	warnings = checkPlanGrounding(phases, []string{})
	if warnings != nil {
		t.Fatalf("expected nil warnings with empty anchors, got %v", warnings)
	}
}

func TestCheckPlanGrounding_AnchorsWithFileRefs(t *testing.T) {
	anchors := []string{"cmd/main.go", "pkg/storage/store.go"}
	phases := []colony.Phase{
		{ID: 1, Name: "Build API", Tasks: []colony.Task{
			{Goal: "Edit cmd/main.go to add new endpoint"},
		}},
	}
	warnings := checkPlanGrounding(phases, anchors)
	if len(warnings) != 0 {
		t.Fatalf("expected no warnings when tasks have file refs, got %v", warnings)
	}
}

func TestCheckPlanGrounding_AnchorsNoFileRefs(t *testing.T) {
	anchors := []string{"cmd/main.go", "pkg/storage/store.go"}
	phases := []colony.Phase{
		{ID: 1, Name: "Build API", Tasks: []colony.Task{
			{Goal: "Set up the authentication system"},
			{Goal: "Configure user roles"},
		}},
	}
	warnings := checkPlanGrounding(phases, anchors)
	if len(warnings) != 1 {
		t.Fatalf("expected 1 warning for ungrounded phase, got %d: %v", len(warnings), warnings)
	}
	if warnings[0].PhaseID != 1 {
		t.Fatalf("expected warning for phase 1, got phase %d", warnings[0].PhaseID)
	}
	if warnings[0].UngroundedTasks != 2 {
		t.Fatalf("expected 2 ungrounded tasks, got %d", warnings[0].UngroundedTasks)
	}
	if warnings[0].AnchorCount != 2 {
		t.Fatalf("expected 2 anchors, got %d", warnings[0].AnchorCount)
	}
}

func TestCheckPlanGrounding_ResearchPhaseExempt(t *testing.T) {
	anchors := []string{"cmd/main.go", "pkg/storage/store.go"}
	researchNames := []string{"Research", "Architecture Review", "Survey Cleanup", "Design Phase", "Planning Discovery"}
	for _, name := range researchNames {
		phases := []colony.Phase{
			{ID: 1, Name: name, Tasks: []colony.Task{
				{Goal: "Investigate options"},
			}},
		}
		warnings := checkPlanGrounding(phases, anchors)
		if len(warnings) != 0 {
			t.Fatalf("expected no warnings for research phase %q, got %v", name, warnings)
		}
	}
}

func TestCheckPlanGrounding_MixedPhases(t *testing.T) {
	anchors := []string{"cmd/main.go", "pkg/storage/store.go"}
	phases := []colony.Phase{
		{ID: 1, Name: "Research Options", Tasks: []colony.Task{
			{Goal: "Investigate frameworks"},
		}},
		{ID: 2, Name: "Build API", Tasks: []colony.Task{
			{Goal: "Edit cmd/main.go to add route"},
		}},
		{ID: 3, Name: "Testing", Tasks: []colony.Task{
			{Goal: "Set up the test suite"},
		}},
	}
	warnings := checkPlanGrounding(phases, anchors)
	if len(warnings) != 1 {
		t.Fatalf("expected 1 warning (phase 3 only), got %d: %v", len(warnings), warnings)
	}
	if warnings[0].PhaseID != 3 {
		t.Fatalf("expected warning for phase 3, got phase %d", warnings[0].PhaseID)
	}
}

func TestIsGroundedTask_FileInGoal(t *testing.T) {
	task := colony.Task{Goal: "Edit cmd/main.go to add new endpoint"}
	if !isGroundedTask(task) {
		t.Fatal("expected task with file path in Goal to be grounded")
	}
}

func TestIsGroundedTask_FileInConstraints(t *testing.T) {
	task := colony.Task{
		Goal:        "Refactor storage layer",
		Constraints: []string{"Must not modify pkg/storage/store.go"},
	}
	if !isGroundedTask(task) {
		t.Fatal("expected task with file path in Constraints to be grounded")
	}
}

func TestIsGroundedTask_FileInHints(t *testing.T) {
	task := colony.Task{
		Goal:  "Update CLI commands",
		Hints: []string{"See cmd/skills.go for pattern"},
	}
	if !isGroundedTask(task) {
		t.Fatal("expected task with file path in Hints to be grounded")
	}
}

func TestIsGroundedTask_NoFileRef(t *testing.T) {
	task := colony.Task{Goal: "Set up the authentication system"}
	if isGroundedTask(task) {
		t.Fatal("expected task with no file references to not be grounded")
	}
}

func TestIsGroundedTask_ExtensionOnlyInHints(t *testing.T) {
	task := colony.Task{
		Goal:  "Fix the bug",
		Hints: []string{"Check main.go for the issue"},
	}
	if !isGroundedTask(task) {
		t.Fatal("expected task with extension-only file reference in Hints to be grounded")
	}
}

func TestIsResearchPhase(t *testing.T) {
	researchNames := []string{
		"Research", "research phase", "Architecture Review",
		"Survey Cleanup", "Design Phase", "Planning Discovery",
	}
	for _, name := range researchNames {
		if !isResearchPhase(name) {
			t.Fatalf("expected %q to be a research phase", name)
		}
	}

	nonResearchNames := []string{"User Auth", "Build API", "Testing", "Deployment"}
	for _, name := range nonResearchNames {
		if isResearchPhase(name) {
			t.Fatalf("expected %q to NOT be a research phase", name)
		}
	}
}

func TestPlanFinalize_GroundingSoftGate(t *testing.T) {
	// Verify that grounding warnings do not block plan-finalize.
	// This test verifies the grounding gate is soft by calling checkPlanGrounding
	// and confirming it returns warnings (not an error).
	anchors := []string{"cmd/main.go", "pkg/storage/store.go"}
	phases := []colony.Phase{
		{ID: 1, Name: "Build API", Tasks: []colony.Task{
			{Goal: "Set up the authentication system"},
		}},
	}
	warnings := checkPlanGrounding(phases, anchors)

	// The function must return warnings, not panic or cause errors.
	// The integration in runCodexPlanFinalize must never return an error
	// from the grounding check -- it only adds warnings to the result map.
	if warnings == nil {
		t.Fatal("expected grounding warnings for ungrounded plan with anchors")
	}
	if len(warnings) != 1 {
		t.Fatalf("expected 1 warning, got %d", len(warnings))
	}

	// Verify the warning format is suitable for result map inclusion.
	if warnings[0].PhaseID != 1 {
		t.Fatalf("expected warning for phase 1, got phase %d", warnings[0].PhaseID)
	}
	if warnings[0].PhaseName != "Build API" {
		t.Fatalf("expected phase name 'Build API', got %q", warnings[0].PhaseName)
	}
}
