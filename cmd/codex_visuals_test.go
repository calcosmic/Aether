package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

func TestPlanVisualOutput(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	t.Setenv("AETHER_OUTPUT_MODE", "visual")

	goal := "Ship visual output parity"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version: "3.0",
		Goal:    &goal,
		State:   colony.StateREADY,
		Plan:    colony.Plan{Phases: []colony.Phase{}},
	})

	rootCmd.SetArgs([]string{"plan"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("plan returned error: %v", err)
	}

	output := stdout.(*bytes.Buffer).String()
	if strings.Contains(output, `{"ok":true`) {
		t.Fatalf("expected visual output, got JSON: %s", output)
	}
	for _, want := range []string{"📋", "P L A N", "P L A N   D I S P A T C H", "aether build 1"} {
		if !strings.Contains(output, want) {
			t.Errorf("plan visual output missing %q\n%s", want, output)
		}
	}
}

func TestBuildVisualOutputShowsSpawnPlan(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	t.Setenv("AETHER_OUTPUT_MODE", "visual")

	goal := "Improve command visuals"
	taskOneID := "task-1"
	taskTwoID := "task-2"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version: "3.0",
		Goal:    &goal,
		State:   colony.StateREADY,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{
					ID:          1,
					Name:        "Visual pass",
					Description: "Bring ceremony to Codex lifecycle output",
					Status:      colony.PhaseReady,
					Tasks: []colony.Task{
						{ID: &taskOneID, Goal: "Implement lifecycle renderer", Status: colony.TaskPending},
						{ID: &taskTwoID, Goal: "Document the new output style", Status: colony.TaskPending, DependsOn: []string{taskOneID}},
					},
				},
			},
		},
	})

	rootCmd.SetArgs([]string{"build", "1"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("build returned error: %v", err)
	}

	output := stdout.(*bytes.Buffer).String()
	if strings.Contains(output, `{"ok":true`) {
		t.Fatalf("expected visual output, got JSON: %s", output)
	}
	for _, want := range []string{"🔨", "B U I L D   D I S P A T C H   1", "S P A W N   P L A N", "🔨🐜", "👁️🐜", "aether continue"} {
		if !strings.Contains(output, want) {
			t.Errorf("build visual output missing %q\n%s", want, output)
		}
	}
}

func TestColonizeVisualOutputShowsDispatchPreview(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatalf("failed to chdir to test root: %v", err)
	}
	defer os.Chdir(oldDir)

	t.Setenv("AETHER_OUTPUT_MODE", "visual")

	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/aether-test\n\ngo 1.24\n"), 0644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("# Test workspace\n"), 0644); err != nil {
		t.Fatalf("failed to write README: %v", err)
	}

	goal := "Survey the repo"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version: "3.0",
		Goal:    &goal,
		State:   colony.StateREADY,
		Plan:    colony.Plan{Phases: []colony.Phase{}},
	})

	rootCmd.SetArgs([]string{"colonize"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("colonize returned error: %v", err)
	}

	output := stdout.(*bytes.Buffer).String()
	if strings.Contains(output, `{"ok":true`) {
		t.Fatalf("expected visual output, got JSON: %s", output)
	}
	for _, want := range []string{"🗺️", "C O L O N I Z E   D I S P A T C H", "Surveyors", "C O L O N I Z E", "aether plan"} {
		if !strings.Contains(output, want) {
			t.Errorf("colonize visual output missing %q\n%s", want, output)
		}
	}
}

func TestPrintNextUpVisualOutput(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	t.Setenv("AETHER_OUTPUT_MODE", "visual")

	goal := "Test next-up visuals"
	name := "test-colony"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		ColonyName:   &name,
		State:        colony.StateEXECUTING,
		CurrentPhase: 1,
		Plan: colony.Plan{
			Phases: []colony.Phase{{ID: 1, Name: "Phase 1", Status: colony.PhaseInProgress}},
		},
	})

	rootCmd.SetArgs([]string{"print-next-up"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("print-next-up returned error: %v", err)
	}

	output := stdout.(*bytes.Buffer).String()
	if strings.Contains(output, `{"ok":true`) {
		t.Fatalf("expected visual output, got JSON: %s", output)
	}
	for _, want := range []string{"N E X T   U P", "aether continue"} {
		if !strings.Contains(output, want) {
			t.Errorf("next-up visual output missing %q\n%s", want, output)
		}
	}
}

func TestOutputErrorVisual(t *testing.T) {
	saveGlobals(t)

	var buf bytes.Buffer
	stderr = &buf
	t.Setenv("AETHER_OUTPUT_MODE", "visual")

	outputError(1, "something failed", nil)

	output := buf.String()
	if strings.Contains(output, `{"ok":false`) {
		t.Fatalf("expected visual error output, got JSON: %s", output)
	}
	for _, want := range []string{"❌", "E R R O R", "something failed"} {
		if !strings.Contains(output, want) {
			t.Errorf("visual error output missing %q\n%s", want, output)
		}
	}
}

func TestShouldRenderVisualOutputTTYOverride(t *testing.T) {
	t.Setenv("AETHER_OUTPUT_MODE", "visual")
	if !shouldRenderVisualOutput(&bytes.Buffer{}) {
		t.Fatal("expected visual mode override to force visual output")
	}

	t.Setenv("AETHER_OUTPUT_MODE", "json")
	if shouldRenderVisualOutput(os.Stdout) {
		t.Fatal("expected json mode override to disable visual output")
	}
}

func TestInstallVisualOutput(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	homeDir := t.TempDir()
	workDir := t.TempDir()
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}
	if err := os.Chdir(workDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	defer os.Chdir(oldDir)

	t.Setenv("AETHER_OUTPUT_MODE", "visual")

	var buf bytes.Buffer
	stdout = &buf

	rootCmd.SetArgs([]string{"install", "--home-dir", homeDir, "--skip-build-binary"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("install returned error: %v", err)
	}

	output := buf.String()
	if strings.Contains(output, `{"ok":true`) {
		t.Fatalf("expected visual output, got JSON: %s", output)
	}
	for _, want := range []string{"📦", "I N S T A L L", "aether lay-eggs"} {
		if !strings.Contains(output, want) {
			t.Errorf("install visual output missing %q\n%s", want, output)
		}
	}
}

func TestRenderUpdateVisualNoChangesSaysNoFollowUpRequired(t *testing.T) {
	output := renderUpdateVisual(
		"/tmp/example",
		"1.0.7",
		"1.0.7",
		false,
		false,
		[]map[string]interface{}{
			{"label": "System files", "copied": 0, "skipped": 10},
			{"label": "AGENTS.md", "skipped": 1, "reason": "unchanged"},
			{"label": ".codex/CODEX.md", "skipped": 1, "reason": "unchanged"},
		},
		0,
		12,
		nil,
	)

	if !strings.Contains(output, "No follow-up is required.") {
		t.Fatalf("expected no-follow-up guidance, got:\n%s", output)
	}
	if strings.Contains(output, "Run `aether status` to inspect the colony after the refresh.") {
		t.Fatalf("expected generic next-step guidance to be suppressed, got:\n%s", output)
	}
}

func TestWorkflowSuggestionsForPausedCurrentPhaseUsesCurrentPhaseNumber(t *testing.T) {
	goal := "Keep the phase suggestion sane"
	primary, _ := workflowSuggestionsForState(colony.ColonyState{
		Goal:         &goal,
		State:        colony.State("PAUSED"),
		CurrentPhase: 1,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{ID: 1, Name: "Reproduce and isolate", Status: colony.PhaseInProgress},
				{ID: 2, Name: "Targeted fix", Status: colony.PhasePending},
			},
		},
	})

	if !strings.Contains(primary, "aether resume") {
		t.Fatalf("expected paused colony to suggest resume, got: %s", primary)
	}
	if strings.Contains(primary, "aether build 2") {
		t.Fatalf("expected paused colony not to skip ahead, got: %s", primary)
	}
}

func TestWorkflowSuggestionsForPausedFlagSuggestsResume(t *testing.T) {
	goal := "Pause should route to resume"
	primary, _ := workflowSuggestionsForState(colony.ColonyState{
		Goal:   &goal,
		State:  colony.StateREADY,
		Paused: true,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{ID: 1, Name: "Reproduce and isolate", Status: colony.PhaseInProgress},
			},
		},
	})

	if !strings.Contains(primary, "aether resume") {
		t.Fatalf("expected paused colony to suggest resume, got: %s", primary)
	}
}

func TestWorkflowSuggestionsForInterruptedExecutingSuggestsRestartBuild(t *testing.T) {
	goal := "Interrupted build should restart clearly"
	primary, _ := workflowSuggestionsForState(colony.ColonyState{
		Goal:         &goal,
		State:        colony.StateEXECUTING,
		CurrentPhase: 2,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{ID: 1, Name: "Done", Status: colony.PhaseCompleted},
				{ID: 2, Name: "Interrupted", Status: colony.PhaseInProgress},
			},
		},
	})

	if !strings.Contains(primary, "aether build 2") {
		t.Fatalf("expected interrupted executing colony to suggest restarting build 2, got: %s", primary)
	}
}

func TestSetupVisualOutput(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	homeDir := t.TempDir()
	repoDir := t.TempDir()
	hubSystem := filepath.Join(homeDir, ".aether", "system")
	if err := os.MkdirAll(hubSystem, 0755); err != nil {
		t.Fatalf("failed to create hub system dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(homeDir, ".aether", "version.json"), []byte(`{"version":"1.0.0"}`), 0644); err != nil {
		t.Fatalf("failed to create hub version: %v", err)
	}
	if err := os.WriteFile(filepath.Join(hubSystem, "workers.md"), []byte("# Workers"), 0644); err != nil {
		t.Fatalf("failed to write workers.md: %v", err)
	}

	t.Setenv("AETHER_OUTPUT_MODE", "visual")

	var buf bytes.Buffer
	stdout = &buf

	rootCmd.SetArgs([]string{"setup", "--repo-dir", repoDir, "--home-dir", homeDir})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("setup returned error: %v", err)
	}

	output := buf.String()
	if strings.Contains(output, `{"ok":true`) {
		t.Fatalf("expected visual output, got JSON: %s", output)
	}
	for _, want := range []string{"🥚", "L A Y   E G G S", "aether init"} {
		if !strings.Contains(output, want) {
			t.Errorf("setup visual output missing %q\n%s", want, output)
		}
	}
}

func TestUpdateDryRunVisualOutput(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	homeDir := t.TempDir()
	repoDir := t.TempDir()
	hubDir := filepath.Join(homeDir, ".aether")
	hubSystem := filepath.Join(hubDir, "system")
	if err := os.MkdirAll(hubSystem, 0755); err != nil {
		t.Fatalf("failed to create hub system dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(hubDir, "version.json"), []byte(`{"version":"1.0.0"}`), 0644); err != nil {
		t.Fatalf("failed to create hub version: %v", err)
	}

	oldDir, _ := os.Getwd()
	if err := os.Chdir(repoDir); err != nil {
		t.Fatalf("failed to chdir to repo: %v", err)
	}
	defer os.Chdir(oldDir)
	t.Setenv("HOME", homeDir)
	t.Setenv("AETHER_OUTPUT_MODE", "visual")

	var buf bytes.Buffer
	stdout = &buf

	rootCmd.SetArgs([]string{"update", "--dry-run"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("update --dry-run returned error: %v", err)
	}

	output := buf.String()
	if strings.Contains(output, `{"ok":true`) {
		t.Fatalf("expected visual output, got JSON: %s", output)
	}
	for _, want := range []string{"🔄", "U P D A T E", "Dry run complete", "aether update"} {
		if !strings.Contains(output, want) {
			t.Errorf("update visual output missing %q\n%s", want, output)
		}
	}
}

func TestPauseResumePatrolPhaseAndHistoryVisualOutput(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	t.Setenv("AETHER_OUTPUT_MODE", "visual")

	goal := "Close the remaining Codex UX gaps"
	taskOneID := "task-1"
	taskTwoID := "task-2"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateEXECUTING,
		CurrentPhase: 1,
		Milestone:    "Open Chambers",
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{
					ID:          1,
					Name:        "Session UX",
					Description: "Add missing resume and handoff flow",
					Status:      colony.PhaseInProgress,
					Tasks: []colony.Task{
						{ID: &taskOneID, Goal: "Write pause-colony", Status: colony.TaskCompleted},
						{ID: &taskTwoID, Goal: "Write resume-colony", Status: colony.TaskInProgress},
					},
				},
			},
		},
		Events: []string{
			"2026-04-15T10:00:00Z|init|queen|Colony initialized",
			"2026-04-15T10:15:00Z|build|builder|Worker wave launched",
		},
	})

	if err := store.SaveJSON("pheromones.json", colony.PheromoneFile{
		Signals: []colony.PheromoneSignal{
			{Type: "FOCUS", Content: []byte(`{"text":"keep the Codex UX coherent"}`), Active: true},
		},
	}); err != nil {
		t.Fatalf("failed to seed pheromones: %v", err)
	}

	if err := store.SaveJSON("flags.json", colony.FlagsFile{
		Version: "1.0",
		Decisions: []colony.FlagEntry{
			{ID: "flag-1", Type: "blocker", Description: "resume parity still missing survey context", CreatedAt: "2026-04-15T10:20:00Z"},
		},
	}); err != nil {
		t.Fatalf("failed to seed flags: %v", err)
	}

	surveyedAt := "2026-04-15T10:05:00Z"
	var seededState colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &seededState); err != nil {
		t.Fatalf("failed to load seeded state: %v", err)
	}
	seededState.TerritorySurveyed = &surveyedAt
	if err := store.SaveJSON("COLONY_STATE.json", seededState); err != nil {
		t.Fatalf("failed to resave seeded state: %v", err)
	}
	surveyDir := filepath.Join(dataDir, "survey")
	if err := os.MkdirAll(surveyDir, 0755); err != nil {
		t.Fatalf("failed to create survey dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(surveyDir, "blueprint.json"), []byte(`{"name":"ux"}`), 0644); err != nil {
		t.Fatalf("failed to seed survey file: %v", err)
	}

	checkVisual := func(args []string, wants ...string) {
		stdout = &bytes.Buffer{}
		rootCmd.SetArgs(args)
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("%s returned error: %v", strings.Join(args, " "), err)
		}
		output := stdout.(*bytes.Buffer).String()
		if strings.Contains(output, `{"ok":true`) {
			t.Fatalf("%s unexpectedly returned JSON: %s", strings.Join(args, " "), output)
		}
		for _, want := range wants {
			if !strings.Contains(output, want) {
				t.Errorf("%s visual output missing %q\n%s", strings.Join(args, " "), want, output)
			}
		}
	}

	checkVisual([]string{"pause-colony"}, "💾", "P A U S E   C O L O N Y", "HANDOFF.md", "aether resume")
	checkVisual([]string{"resume-colony"}, "💾", "R E S U M E   C O L O N Y", "Session UX", "Active Signals", "Blockers", "Survey Context", "Source:", "aether build 1")
	checkVisual([]string{"patrol"}, "📊", "P A T R O L", "Signals: 1 active")
	checkVisual([]string{"phase"}, "🧱", "Session UX", "Write resume-colony")
	checkVisual([]string{"history"}, "📜", "Colony initialized", "Worker wave launched")
}

func TestRenderColonizeVisual_FakeDispatch(t *testing.T) {
	// When all surveyors have "spawned" status, the legacy display should be used.
	result := map[string]interface{}{
		"root":          "/tmp/test",
		"detected_type": "go",
		"languages":     []interface{}{"go"},
		"frameworks":    []interface{}{"cobra"},
		"domains":       []interface{}{"cli"},
		"stats": map[string]interface{}{
			"files":       42,
			"directories": 7,
		},
		"survey_dir": "/tmp/test/.aether/data/survey",
		"surveyors": []interface{}{
			map[string]interface{}{"name": "Nest-42", "caste": "surveyor-nest", "task": "Map architecture", "status": "spawned"},
			map[string]interface{}{"name": "Disc-7", "caste": "surveyor-disciplines", "task": "Map disciplines", "status": "spawned"},
			map[string]interface{}{"name": "Path-3", "caste": "surveyor-pathogens", "task": "Identify pathogens", "status": "spawned"},
			map[string]interface{}{"name": "Prov-1", "caste": "surveyor-provisions", "task": "Map provisions", "status": "spawned"},
		},
		"survey_files": []interface{}{"BLUEPRINT.md", "CHAMBERS.md"},
	}

	output := renderColonizeVisual(result)

	// Legacy display should show surveyor names and tasks (no status icons or durations)
	if !strings.Contains(output, "Nest-42") {
		t.Errorf("legacy output missing surveyor name %q\n%s", "Nest-42", output)
	}
	if !strings.Contains(output, "Map architecture") {
		t.Errorf("legacy output missing surveyor task %q\n%s", "Map architecture", output)
	}
	// Should show a non-real dispatch indicator
	if !strings.Contains(output, "Dispatch: Simulated") && !strings.Contains(output, "Dispatch: Synthetic") {
		t.Errorf("legacy output should contain a non-real dispatch indicator\n%s", output)
	}
	// Should NOT show real dispatch indicator
	if strings.Contains(output, "Dispatch: Real") {
		t.Errorf("legacy output should not contain 'Dispatch: Real'\n%s", output)
	}
	// Should still show standard sections
	if !strings.Contains(output, "C O L O N I Z E") {
		t.Errorf("missing banner in output\n%s", output)
	}
}

func TestRenderColonizeVisual_RealDispatch(t *testing.T) {
	// When surveyors have "completed" or "failed" status, show execution data.
	result := map[string]interface{}{
		"root":          "/tmp/test",
		"detected_type": "go",
		"languages":     []interface{}{"go"},
		"frameworks":    []interface{}{"cobra"},
		"domains":       []interface{}{"cli"},
		"stats": map[string]interface{}{
			"files":       42,
			"directories": 7,
		},
		"survey_dir": "/tmp/test/.aether/data/survey",
		"surveyors": []interface{}{
			map[string]interface{}{"name": "Nest-42", "caste": "surveyor-nest", "task": "Map architecture", "status": "completed", "duration": 12.3},
			map[string]interface{}{"name": "Disc-7", "caste": "surveyor-disciplines", "task": "Map disciplines", "status": "completed", "duration": 8.1},
			map[string]interface{}{"name": "Path-3", "caste": "surveyor-pathogens", "task": "Identify pathogens", "status": "failed", "duration": 5.2},
			map[string]interface{}{"name": "Prov-1", "caste": "surveyor-provisions", "task": "Map provisions", "status": "completed", "duration": 15.7},
		},
		"survey_files": []interface{}{"BLUEPRINT.md", "CHAMBERS.md"},
	}

	output := renderColonizeVisual(result)

	// Real dispatch should show the "Dispatch: Real" indicator
	if !strings.Contains(output, "Dispatch: Real") {
		t.Errorf("real dispatch output missing 'Dispatch: Real'\n%s", output)
	}
	// Should show status icons
	if !strings.Contains(output, "\u2713") {
		t.Errorf("real dispatch output missing checkmark for completed surveyors\n%s", output)
	}
	if !strings.Contains(output, "\u2717") {
		t.Errorf("real dispatch output missing X for failed surveyors\n%s", output)
	}
	// Should show durations
	if !strings.Contains(output, "12.3s") {
		t.Errorf("real dispatch output missing duration 12.3s\n%s", output)
	}
	if !strings.Contains(output, "8.1s") {
		t.Errorf("real dispatch output missing duration 8.1s\n%s", output)
	}
	// Should show summary line
	if !strings.Contains(output, "3/4 surveyors completed") {
		t.Errorf("real dispatch output missing summary '3/4 surveyors completed'\n%s", output)
	}
	// Should contain the "Surveyors" section header with real results
	if !strings.Contains(output, "\nSurveyors\n") {
		t.Errorf("real dispatch output should have 'Surveyors' header\n%s", output)
	}
}

func TestRenderColonizeVisual_MixedResults(t *testing.T) {
	// Mix of "spawned" and real statuses: spawned is treated as fake.
	result := map[string]interface{}{
		"root":          "/tmp/test",
		"detected_type": "go",
		"languages":     []interface{}{"go"},
		"frameworks":    []interface{}{},
		"domains":       []interface{}{},
		"stats": map[string]interface{}{
			"files":       10,
			"directories": 2,
		},
		"survey_dir": "/tmp/test/.aether/data/survey",
		"surveyors": []interface{}{
			map[string]interface{}{"name": "Nest-42", "caste": "surveyor-nest", "task": "Map architecture", "status": "completed", "duration": 5.0},
			map[string]interface{}{"name": "Disc-7", "caste": "surveyor-disciplines", "task": "Map disciplines", "status": "spawned"},
			map[string]interface{}{"name": "Path-3", "caste": "surveyor-pathogens", "task": "Identify pathogens", "status": "failed", "duration": 3.0},
		},
		"survey_files": []interface{}{"BLUEPRINT.md"},
	}

	output := renderColonizeVisual(result)

	// Should show "Dispatch: Real" because at least one has real execution data
	if !strings.Contains(output, "Dispatch: Real") {
		t.Errorf("mixed output should show 'Dispatch: Real'\n%s", output)
	}
	// Should show 1/3 completed (only Nest-42 completed; Disc-7 spawned; Path-3 failed)
	if !strings.Contains(output, "1/3 surveyors completed") {
		t.Errorf("mixed output missing correct summary '1/3 surveyors completed'\n%s", output)
	}
}

func TestRenderSurveyorResults_Formatting(t *testing.T) {
	surveyors := []codexSurveyorDispatch{
		{Caste: "surveyor-nest", Name: "Nest-42", Task: "Map architecture", Status: "completed", Duration: 12.3},
		{Caste: "surveyor-disciplines", Name: "Disc-7", Task: "Map disciplines", Status: "completed", Duration: 8.1},
		{Caste: "surveyor-pathogens", Name: "Path-3", Task: "Identify pathogens", Status: "failed", Duration: 5.2},
		{Caste: "surveyor-provisions", Name: "Prov-1", Task: "Map provisions", Status: "completed", Duration: 15.7},
	}

	output := renderSurveyorResults(surveyors)

	// Each surveyor should be on its own line with emoji, name, caste, status, and duration
	if !strings.Contains(output, "Nest-42") {
		t.Errorf("missing Nest-42\n%s", output)
	}
	if !strings.Contains(output, "Disc-7") {
		t.Errorf("missing Disc-7\n%s", output)
	}
	if !strings.Contains(output, "Path-3") {
		t.Errorf("missing Path-3\n%s", output)
	}
	if !strings.Contains(output, "Prov-1") {
		t.Errorf("missing Prov-1\n%s", output)
	}

	// Should use caste emojis from casteEmojiMap (surveyor = bar chart + ant)
	if !strings.Contains(output, "\U0001f4ca\U0001f41c") {
		t.Errorf("missing surveyor caste emoji\n%s", output)
	}

	// Completed should have checkmark, failed should have X
	if !strings.Contains(output, "\u2713 completed") {
		t.Errorf("missing checkmark for completed surveyors\n%s", output)
	}
	if !strings.Contains(output, "\u2717 failed") {
		t.Errorf("missing X for failed surveyors\n%s", output)
	}

	// Should have durations
	if !strings.Contains(output, "12.3s") {
		t.Errorf("missing duration 12.3s\n%s", output)
	}

	// Summary line at the end
	if !strings.Contains(output, "3/4 surveyors completed") {
		t.Errorf("missing summary line\n%s", output)
	}
}

func TestRenderSurveyorResults_Empty(t *testing.T) {
	output := renderSurveyorResults(nil)
	if output != "" {
		t.Errorf("expected empty output for nil surveyors, got %q", output)
	}
	output = renderSurveyorResults([]codexSurveyorDispatch{})
	if output != "" {
		t.Errorf("expected empty output for empty surveyors, got %q", output)
	}
}

func TestRenderPlanVisual_SimulatedDispatch(t *testing.T) {
	// When all planning workers have "spawned" status, the legacy display should be used.
	phases := []colony.Phase{
		{ID: 1, Name: "Discovery", Description: "Map the codebase", Status: colony.PhaseReady,
			Tasks: []colony.Task{{Goal: "Read code paths"}}},
	}
	phaseMaps := make([]interface{}, len(phases))
	for i, p := range phases {
		phaseMaps[i] = phaseToMap(p)
	}

	result := map[string]interface{}{
		"existing_plan": false,
		"goal":          "Build feature X",
		"granularity":   "milestone",
		"confidence":    map[string]interface{}{"overall": 72},
		"phases":        phaseMaps,
		"dispatches": []interface{}{
			map[string]interface{}{"name": "Scout-7", "caste": "scout", "task": "Survey the repo", "status": "spawned"},
			map[string]interface{}{"name": "Route-12", "caste": "route_setter", "task": "Convert findings into phases", "status": "spawned"},
		},
	}

	output := renderPlanVisual(result)

	// Should show worker names and tasks (legacy style, no status icons)
	if !strings.Contains(output, "Scout-7") {
		t.Errorf("simulated output missing scout name\n%s", output)
	}
	if !strings.Contains(output, "Route-12") {
		t.Errorf("simulated output missing route-setter name\n%s", output)
	}
	if !strings.Contains(output, "Survey the repo") {
		t.Errorf("simulated output missing scout task\n%s", output)
	}
	// Should NOT show dispatch mode indicator or status icons
	if strings.Contains(output, "Dispatch: Real") {
		t.Errorf("simulated output should not contain 'Dispatch: Real'\n%s", output)
	}
	if strings.Contains(output, "Dispatch: Simulated") {
		t.Errorf("simulated output should not contain 'Dispatch: Simulated'\n%s", output)
	}
	// Should still show standard sections
	if !strings.Contains(output, "P L A N") {
		t.Errorf("missing banner in output\n%s", output)
	}
}

func TestRenderPlanVisual_RealDispatch(t *testing.T) {
	// When planning workers have "completed" or "failed" status, show execution data.
	phases := []colony.Phase{
		{ID: 1, Name: "Discovery", Description: "Map the codebase", Status: colony.PhaseReady,
			Tasks: []colony.Task{{Goal: "Read code paths"}}},
	}
	phaseMaps := make([]interface{}, len(phases))
	for i, p := range phases {
		phaseMaps[i] = phaseToMap(p)
	}

	result := map[string]interface{}{
		"existing_plan": false,
		"goal":          "Build feature X",
		"granularity":   "milestone",
		"confidence":    map[string]interface{}{"overall": 72},
		"phases":        phaseMaps,
		"dispatches": []interface{}{
			map[string]interface{}{"name": "Scout-7", "caste": "scout", "task": "Survey the repo", "status": "completed", "duration": 3.5},
			map[string]interface{}{"name": "Route-12", "caste": "route_setter", "task": "Convert findings into phases", "status": "completed", "duration": 2.1},
		},
	}

	output := renderPlanVisual(result)

	// Should show "Dispatch: Real" indicator
	if !strings.Contains(output, "Dispatch: Real") {
		t.Errorf("real dispatch output missing 'Dispatch: Real'\n%s", output)
	}
	// Should show status icons (checkmark for completed)
	if !strings.Contains(output, "\u2713") {
		t.Errorf("real dispatch output missing checkmark for completed workers\n%s", output)
	}
	// Should show durations
	if !strings.Contains(output, "3.5s") {
		t.Errorf("real dispatch output missing duration 3.5s\n%s", output)
	}
	if !strings.Contains(output, "2.1s") {
		t.Errorf("real dispatch output missing duration 2.1s\n%s", output)
	}
	// Should show summary line
	if !strings.Contains(output, "2/2 workers completed") {
		t.Errorf("real dispatch output missing summary '2/2 workers completed'\n%s", output)
	}
	// Should show "Workers" section header
	if !strings.Contains(output, "\nWorkers\n") {
		t.Errorf("real dispatch output should have 'Workers' header\n%s", output)
	}
}

func TestRenderPlanVisual_RealDispatchWithFailure(t *testing.T) {
	// Scout completed but route-setter failed.
	phases := []colony.Phase{
		{ID: 1, Name: "Discovery", Description: "Map the codebase", Status: colony.PhaseReady,
			Tasks: []colony.Task{{Goal: "Read code paths"}}},
	}
	phaseMaps := make([]interface{}, len(phases))
	for i, p := range phases {
		phaseMaps[i] = phaseToMap(p)
	}

	result := map[string]interface{}{
		"existing_plan": false,
		"goal":          "Build feature X",
		"granularity":   "milestone",
		"confidence":    map[string]interface{}{"overall": 72},
		"phases":        phaseMaps,
		"dispatches": []interface{}{
			map[string]interface{}{"name": "Scout-7", "caste": "scout", "task": "Survey the repo", "status": "completed", "duration": 4.0},
			map[string]interface{}{"name": "Route-12", "caste": "route_setter", "task": "Convert findings into phases", "status": "failed", "duration": 1.2},
		},
	}

	output := renderPlanVisual(result)

	// Should show "Dispatch: Real"
	if !strings.Contains(output, "Dispatch: Real") {
		t.Errorf("output missing 'Dispatch: Real'\n%s", output)
	}
	// Should show checkmark for scout and X for route-setter
	if !strings.Contains(output, "\u2713") {
		t.Errorf("output missing checkmark\n%s", output)
	}
	if !strings.Contains(output, "\u2717") {
		t.Errorf("output missing X for failed worker\n%s", output)
	}
	// Should show 1/2 completed
	if !strings.Contains(output, "1/2 workers completed") {
		t.Errorf("output missing summary '1/2 workers completed'\n%s", output)
	}
	// Should show total duration
	if !strings.Contains(output, "5.2s") {
		t.Errorf("output missing total duration 5.2s\n%s", output)
	}
}

func TestRenderPlanVisual_NoDispatches(t *testing.T) {
	// No dispatches at all -- should not crash, should show legacy output.
	phases := []colony.Phase{
		{ID: 1, Name: "Discovery", Description: "Map the codebase", Status: colony.PhaseReady,
			Tasks: []colony.Task{{Goal: "Read code paths"}}},
	}
	phaseMaps := make([]interface{}, len(phases))
	for i, p := range phases {
		phaseMaps[i] = phaseToMap(p)
	}

	result := map[string]interface{}{
		"existing_plan": false,
		"goal":          "Build feature X",
		"granularity":   "milestone",
		"confidence":    map[string]interface{}{"overall": 72},
		"phases":        phaseMaps,
	}

	output := renderPlanVisual(result)

	// Should not crash, should still show phases
	if !strings.Contains(output, "Discovery") {
		t.Errorf("output missing phase name\n%s", output)
	}
	if !strings.Contains(output, "P L A N") {
		t.Errorf("missing banner\n%s", output)
	}
}

func TestRenderPlanVisual_ExistingPlan(t *testing.T) {
	// Existing plan with no dispatches -- should show the existing plan message.
	phases := []colony.Phase{
		{ID: 1, Name: "Phase 1", Status: colony.PhaseReady,
			Tasks: []colony.Task{{Goal: "Task A"}}},
	}
	phaseMaps := make([]interface{}, len(phases))
	for i, p := range phases {
		phaseMaps[i] = phaseToMap(p)
	}

	result := map[string]interface{}{
		"existing_plan": true,
		"goal":          "Previous goal",
		"phases":        phaseMaps,
	}

	output := renderPlanVisual(result)

	if !strings.Contains(output, "Existing colony plan loaded") {
		t.Errorf("existing plan output missing 'Existing colony plan loaded'\n%s", output)
	}
}

func TestRenderPlanVisual_ShowsClarificationWarning(t *testing.T) {
	phaseMaps := []interface{}{
		map[string]interface{}{
			"id":     1,
			"name":   "Phase 1",
			"status": colony.PhaseReady,
			"tasks": []interface{}{
				map[string]interface{}{"goal": "Task A", "status": colony.TaskPending},
			},
		},
	}

	result := map[string]interface{}{
		"existing_plan":             true,
		"goal":                      "Previous goal",
		"phases":                    phaseMaps,
		"unresolved_clarifications": 2,
		"clarification_warning":     "Unresolved clarifications exist. Run `aether discuss` to resolve them before planning, or proceed with implicit assumptions.",
	}

	output := renderPlanVisual(result)
	if !strings.Contains(output, "Clarifications") {
		t.Fatalf("expected clarifications section in plan output\n%s", output)
	}
	if !strings.Contains(output, "2 unresolved clarification(s)") {
		t.Fatalf("expected unresolved clarification count in plan output\n%s", output)
	}
	if !strings.Contains(output, "Run `aether discuss`") {
		t.Fatalf("expected discuss guidance in plan output\n%s", output)
	}
}

// phaseToMap converts a colony.Phase to a map[string]interface{} for test construction.
func phaseToMap(p colony.Phase) map[string]interface{} {
	tasks := make([]interface{}, len(p.Tasks))
	for i, task := range p.Tasks {
		m := map[string]interface{}{
			"goal":   task.Goal,
			"status": task.Status,
		}
		if task.ID != nil {
			m["id"] = *task.ID
		}
		tasks[i] = m
	}
	return map[string]interface{}{
		"id":          p.ID,
		"name":        p.Name,
		"description": p.Description,
		"status":      p.Status,
		"tasks":       tasks,
	}
}

func TestInstallJsonModeStillProducesJson(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	homeDir := t.TempDir()
	workDir := t.TempDir()
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}
	if err := os.Chdir(workDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	defer os.Chdir(oldDir)

	t.Setenv("AETHER_OUTPUT_MODE", "json")

	var buf bytes.Buffer
	stdout = &buf

	rootCmd.SetArgs([]string{"install", "--home-dir", homeDir, "--skip-build-binary"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("install returned error: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("expected JSON output, got parse error: %v\n%s", err, buf.String())
	}
	if parsed["ok"] != true {
		t.Fatalf("expected ok:true envelope, got %v", parsed)
	}
}
