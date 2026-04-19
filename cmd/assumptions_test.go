package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

func TestAssumptionsAnalyzeCreatesAssumptions(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	surveyDir := filepath.Join(dataDir, "survey")
	if err := os.MkdirAll(surveyDir, 0755); err != nil {
		t.Fatalf("mkdir survey: %v", err)
	}
	if err := os.WriteFile(filepath.Join(surveyDir, "blueprint.json"), []byte(`{"frameworks":["react"],"entry_points":["web/app.tsx"]}`), 0644); err != nil {
		t.Fatalf("write blueprint: %v", err)
	}
	if err := os.WriteFile(filepath.Join(surveyDir, "disciplines.json"), []byte(`{"tests":["web/app_test.tsx"]}`), 0644); err != nil {
		t.Fatalf("write disciplines: %v", err)
	}
	if err := os.WriteFile(filepath.Join(surveyDir, "chambers.json"), []byte(`{"directories":["web"]}`), 0644); err != nil {
		t.Fatalf("write chambers: %v", err)
	}
	if err := os.WriteFile(filepath.Join(surveyDir, "provisions.json"), []byte(`{"dependencies":["postgres"]}`), 0644); err != nil {
		t.Fatalf("write provisions: %v", err)
	}

	goal := "Build an internal dashboard"
	taskID := "1.1"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version: "3.0",
		Goal:    &goal,
		State:   colony.StateREADY,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{
					ID:     1,
					Name:   "Phase 1",
					Status: colony.PhaseReady,
					Tasks:  []colony.Task{{ID: &taskID, Goal: "Build the dashboard shell", Status: colony.TaskPending}},
				},
			},
		},
	})

	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldDir)

	rootCmd.SetArgs([]string{"assumptions-analyze"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("assumptions-analyze returned error: %v", err)
	}

	env := parseEnvelope(t, stdout.(*bytes.Buffer).String())
	result := env["result"].(map[string]interface{})
	if got := int(result["assumption_count"].(float64)); got == 0 {
		t.Fatal("expected assumptions-analyze to create assumptions")
	}
	if got := int(result["feedback_emitted"].(float64)); got == 0 {
		t.Fatal("expected confident assumptions to emit FEEDBACK pheromones")
	}

	var file colony.AssumptionsFile
	if err := store.LoadJSON(assumptionsFile, &file); err != nil {
		t.Fatalf("load assumptions: %v", err)
	}
	if len(file.Assumptions) == 0 {
		t.Fatal("expected assumptions.json to persist assumptions")
	}
}

func TestAssumptionValidateMarksAssumption(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	setupBuildFlowTest(t)
	if err := store.SaveJSON(assumptionsFile, colony.AssumptionsFile{
		Version:     "1.0",
		GeneratedAt: "2026-04-19T10:00:00Z",
		Goal:        "Build an internal dashboard",
		Assumptions: []colony.Assumption{{
			ID:             "asm_phase1_surface",
			Phase:          1,
			Category:       "surface",
			AssumptionText: "The first implementation slice should stay in the existing react surface.",
			Confidence:     colony.AssumptionConfidenceConfident,
			CreatedAt:      "2026-04-19T10:00:00Z",
		}},
	}); err != nil {
		t.Fatalf("seed assumptions: %v", err)
	}

	rootCmd.SetArgs([]string{"assumption-validate", "--id", "asm_phase1_surface", "--note", "Confirmed against current dashboard module"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("assumption-validate returned error: %v", err)
	}

	env := parseEnvelope(t, stdout.(*bytes.Buffer).String())
	result := env["result"].(map[string]interface{})
	if validated, _ := result["validated"].(bool); !validated {
		t.Fatalf("expected validated:true, got %v", result)
	}

	var file colony.AssumptionsFile
	if err := store.LoadJSON(assumptionsFile, &file); err != nil {
		t.Fatalf("reload assumptions: %v", err)
	}
	if !file.Assumptions[0].Validated {
		t.Fatal("expected assumption to be marked validated")
	}
	if file.Assumptions[0].ValidationNote != "Confirmed against current dashboard module" {
		t.Fatalf("validation note = %q", file.Assumptions[0].ValidationNote)
	}
}

func TestAssumptionsAnalyzeEmitsFocusForUnclearAssumptions(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	surveyDir := filepath.Join(dataDir, "survey")
	if err := os.MkdirAll(surveyDir, 0755); err != nil {
		t.Fatalf("mkdir survey: %v", err)
	}
	if err := os.WriteFile(filepath.Join(surveyDir, "blueprint.json"), []byte(`{"frameworks":["react","vue"],"entry_points":["web/app.tsx","admin/app.tsx"]}`), 0644); err != nil {
		t.Fatalf("write blueprint: %v", err)
	}
	if err := os.WriteFile(filepath.Join(surveyDir, "chambers.json"), []byte(`{"directories":["web","admin"]}`), 0644); err != nil {
		t.Fatalf("write chambers: %v", err)
	}

	goal := "Build an internal dashboard"
	taskID := "1.1"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version: "3.0",
		Goal:    &goal,
		State:   colony.StateREADY,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{
					ID:     1,
					Name:   "Phase 1",
					Status: colony.PhaseReady,
					Tasks:  []colony.Task{{ID: &taskID, Goal: "Build the dashboard shell", Status: colony.TaskPending}},
				},
			},
		},
	})

	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldDir)

	rootCmd.SetArgs([]string{"assumptions-analyze"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("assumptions-analyze returned error: %v", err)
	}

	env := parseEnvelope(t, stdout.(*bytes.Buffer).String())
	result := env["result"].(map[string]interface{})
	if got := int(result["focus_emitted"].(float64)); got == 0 {
		t.Fatal("expected unclear assumptions to emit FOCUS pheromones")
	}

	var pheromones colony.PheromoneFile
	if err := store.LoadJSON("pheromones.json", &pheromones); err != nil {
		t.Fatalf("load pheromones: %v", err)
	}
	foundFocus := false
	for _, signal := range pheromones.Signals {
		if signal.Type == "FOCUS" {
			foundFocus = true
			break
		}
	}
	if !foundFocus {
		encoded, _ := json.MarshalIndent(pheromones, "", "  ")
		t.Fatalf("expected at least one FOCUS pheromone, got:\n%s", string(encoded))
	}
}
