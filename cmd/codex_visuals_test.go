package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/agent"
	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
)

// pinRawCommandNames fixes the rendering platform for tests that assert the raw
// `aether <verb>` next-step wording. translateHintCommandsForPlatform rewrites
// those verbs to `/ant-<verb>` on every platform except Codex, and
// detectPlatform falls back to "claude" when nothing in the environment says
// otherwise — so without this pin the same test passes or fails depending on
// which agent runtime happens to be running it.
func pinRawCommandNames(t *testing.T) {
	t.Helper()
	t.Setenv("AETHER_PLATFORM", "codex")
}

func TestCommandCeremonyTaxonomyLevelsAreExplicit(t *testing.T) {
	levels := map[commandCeremonyLevel]string{
		commandCeremonyLevelWorkerTheatre: "worker_theatre",
		commandCeremonyLevelGuidedRitual:  "guided_ritual",
		commandCeremonyLevelDashboard:     "dashboard",
		commandCeremonyLevelProgress:      "progress",
		commandCeremonyLevelQuiet:         "quiet",
	}

	if len(levels) != 5 {
		t.Fatalf("ceremony level count = %d, want 5", len(levels))
	}
	for level, want := range levels {
		if got := string(level); got != want {
			t.Errorf("ceremony level string = %q, want %q", got, want)
		}
	}
}

func TestCommandCeremonyTaxonomyMapsRepresentativeCommands(t *testing.T) {
	tests := []struct {
		command string
		want    commandCeremonyLevel
	}{
		{command: "plan", want: commandCeremonyLevelWorkerTheatre},
		{command: "build", want: commandCeremonyLevelWorkerTheatre},
		{command: "colonize", want: commandCeremonyLevelWorkerTheatre},
		{command: "seal", want: commandCeremonyLevelWorkerTheatre},
		{command: "swarm", want: commandCeremonyLevelWorkerTheatre},
		{command: "swarm --plan-only Fix auth panic", want: commandCeremonyLevelWorkerTheatre},
		{command: "swarm --watch", want: commandCeremonyLevelDashboard},
		{command: "aether swarm --watch", want: commandCeremonyLevelDashboard},
		{command: "init", want: commandCeremonyLevelGuidedRitual},
		{command: "discuss", want: commandCeremonyLevelGuidedRitual},
		{command: "oracle", want: commandCeremonyLevelGuidedRitual},
		{command: "status", want: commandCeremonyLevelDashboard},
		{command: "watch", want: commandCeremonyLevelDashboard},
		{command: "history", want: commandCeremonyLevelDashboard},
		{command: "phase", want: commandCeremonyLevelDashboard},
		{command: "resume", want: commandCeremonyLevelDashboard},
		{command: "run", want: commandCeremonyLevelProgress},
		{command: "update", want: commandCeremonyLevelProgress},
		{command: "publish", want: commandCeremonyLevelProgress},
		{command: "continue", want: commandCeremonyLevelProgress},
		{command: "plan-finalize", want: commandCeremonyLevelQuiet},
		{command: "build-finalize", want: commandCeremonyLevelQuiet},
		{command: "continue-finalize", want: commandCeremonyLevelQuiet},
		{command: "seal-finalize", want: commandCeremonyLevelQuiet},
		{command: "command-guide", want: commandCeremonyLevelQuiet},
		{command: "unknown-command", want: commandCeremonyLevelQuiet},
	}

	for _, tt := range tests {
		t.Run(tt.command, func(t *testing.T) {
			if got := classifyCommandCeremonyLevel(tt.command); got != tt.want {
				t.Fatalf("classifyCommandCeremonyLevel(%q) = %q, want %q", tt.command, got, tt.want)
			}
		})
	}
}

func TestDeliveryProgressCommandsClassifiedAsProgress(t *testing.T) {
	commands := []string{
		"run --max-phases 2",
		"continue --skip-watchers",
		"update --force --download-binary",
		"publish --channel stable",
		"install --download-binary",
		"lay-eggs",
		"porter check",
		"source-check --json",
		"bump-version 1.2.3",
	}

	for _, command := range commands {
		t.Run(command, func(t *testing.T) {
			if got := classifyCommandCeremonyLevel(command); got != commandCeremonyLevelProgress {
				t.Fatalf("classifyCommandCeremonyLevel(%q) = %q, want %q", command, got, commandCeremonyLevelProgress)
			}
		})
	}
}

func TestQuietInternalCommandsClassifiedAsQuiet(t *testing.T) {
	commands := []string{
		"plan-finalize --completion-file /tmp/plan.json",
		"build-finalize 5 --completion-file /tmp/build.json",
		"continue-finalize --completion-file /tmp/continue.json",
		"seal-finalize --completion-file /tmp/seal.json",
		"colonize-finalize --completion-file /tmp/colonize.json",
		"swarm-finalize --completion-file /tmp/swarm.json",
		"oracle-iterate-finalize --completion-file /tmp/oracle.json",
		"command-guide build --platform codex",
		"spawn-log --name Mason-1",
		"spawn-complete --name Mason-1",
		"ceremony spawn-plan --workflow build",
		"completion zsh",
		"version",
		"generate-progress-bar --current 1 --total 5",
		"version-check-cached",
	}

	for _, command := range commands {
		t.Run(command, func(t *testing.T) {
			if got := classifyCommandCeremonyLevel(command); got != commandCeremonyLevelQuiet {
				t.Fatalf("classifyCommandCeremonyLevel(%q) = %q, want %q", command, got, commandCeremonyLevelQuiet)
			}
		})
	}
}

func TestCeremonyCloseoutRealDispatchRendersWorkerTheatre(t *testing.T) {
	saveGlobals(t)
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	goal := "Render real worker closeout"
	state := colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateBUILT,
		CurrentPhase: 1,
		Plan: colony.Plan{Phases: []colony.Phase{
			{ID: 1, Name: "Real dispatch", Status: colony.PhaseInProgress},
		}},
	}
	if err := store.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatalf("save colony state: %v", err)
	}

	completionFile := writeCeremonyTestJSON(t, map[string]interface{}{
		"ok":                true,
		"dispatch_manifest": ceremonyTestManifest(),
		"dispatches": []map[string]interface{}{
			{"name": "Mason-37", "caste": "builder", "status": "completed", "summary": "Added ceremony tests", "tool_count": 9},
			{"name": "Keen-11", "caste": "watcher", "status": "completed", "summary": "Verified focused tests", "tool_count": 3},
		},
	})

	result, visual := renderCeremonyCloseout("build", completionFile)
	if got := intValue(result["completion_worker_count"]); got != 2 {
		t.Fatalf("completion_worker_count = %d, want 2", got)
	}
	for _, want := range []string{
		"B U I L D   S U M M A R Y",
		"Workers: 2 completed  0 blocked  0 failed",
		"Worker Results",
		"Mason-37",
		"Keen-11",
	} {
		if !strings.Contains(visual, want) {
			t.Fatalf("real dispatch closeout missing %q\n%s", want, visual)
		}
	}
	for _, forbidden := range []string{
		"No workers dispatched",
		"Existing colony plan loaded",
		"F I N A L I Z E R   F A I L E D",
	} {
		if strings.Contains(visual, forbidden) {
			t.Fatalf("real dispatch closeout should not contain %q\n%s", forbidden, visual)
		}
	}
}

func TestCeremonyCloseoutNoWorkerExistingPlanDoesNotRenderWorkerTheatre(t *testing.T) {
	saveGlobals(t)
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	goal := "Reuse the existing plan"
	taskID := "1.1"
	state := colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateREADY,
		CurrentPhase: 1,
		Plan: colony.Plan{Phases: []colony.Phase{
			{ID: 1, Name: "Existing phase", Status: colony.PhaseReady, Tasks: []colony.Task{{ID: &taskID, Goal: "Keep the existing work plan"}}},
		}},
	}
	if err := store.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatalf("save colony state: %v", err)
	}

	completionFile := writeCeremonyTestJSON(t, map[string]interface{}{
		"ok":                 true,
		"existing_plan":      true,
		"requires_finalizer": false,
		"message":            "Existing colony plan loaded; no planning workers dispatched.",
		"plan_manifest": map[string]interface{}{
			"existing_plan": true,
			"phase":         1,
			"phase_name":    "Existing phase",
			"dispatches":    []map[string]interface{}{},
		},
		"dispatches": []map[string]interface{}{},
	})

	result, visual := renderCeremonyCloseout("plan", completionFile)
	if got := intValue(result["completion_worker_count"]); got != 0 {
		t.Fatalf("completion_worker_count = %d, want 0", got)
	}
	for _, want := range []string{
		"P L A N   S U M M A R Y",
		"Existing colony plan loaded",
		"No workers dispatched",
	} {
		if !strings.Contains(visual, want) {
			t.Fatalf("no-worker closeout missing %q\n%s", want, visual)
		}
	}
	for _, forbidden := range []string{
		"\nWorkers:",
		"Worker Results",
		"S P A W N   P L A N",
		"Scout and Route-Setter mapped",
	} {
		if strings.Contains(visual, forbidden) {
			t.Fatalf("no-worker closeout rendered worker theatre via %q\n%s", forbidden, visual)
		}
	}
}

func TestCeremonyCloseoutBlockedPathRendersBlockedNotCompletion(t *testing.T) {
	pinRawCommandNames(t)
	saveGlobals(t)
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	goal := "Keep blocked work honest"
	taskID := "1.1"
	state := colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateBUILT,
		CurrentPhase: 1,
		Plan: colony.Plan{Phases: []colony.Phase{
			{ID: 1, Name: "Blocked verification", Status: colony.PhaseInProgress, Tasks: []colony.Task{{ID: &taskID, Goal: "Fix verification failure", Status: colony.TaskInProgress}}},
		}},
	}
	if err := store.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatalf("save colony state: %v", err)
	}

	completionFile := writeCeremonyTestJSON(t, map[string]interface{}{
		"ok":      true,
		"blocked": true,
		"next":    "Run `aether build 1 --force` after fixing blocked worker output.",
		"continue_manifest": map[string]interface{}{
			"phase":      1,
			"phase_name": "Blocked verification",
			"dispatches": []map[string]interface{}{
				{"name": "Keen-37", "caste": "watcher", "status": "planned", "task": "Verify the phase"},
			},
		},
		"dispatches": []map[string]interface{}{
			{
				"name":     "Keen-37",
				"caste":    "watcher",
				"status":   "blocked",
				"summary":  "Verification blocked the phase",
				"blockers": []string{"go test ./... failed"},
			},
		},
	})

	_, visual := renderCeremonyCloseout("continue", completionFile)
	for _, want := range []string{
		"C O N T I N U E   B L O C K E D",
		"Verification blocked the phase",
		"go test ./... failed",
		"Run `aether build 1 --force`",
	} {
		if !strings.Contains(visual, want) {
			t.Fatalf("blocked closeout missing %q\n%s", want, visual)
		}
	}
	for _, forbidden := range []string{
		"C O N T I N U E   S U M M A R Y",
		"Run `aether build 1` to dispatch the next phase.",
		"Run `aether seal`",
	} {
		if strings.Contains(visual, forbidden) {
			t.Fatalf("blocked closeout hid blocked state behind %q\n%s", forbidden, visual)
		}
	}
}

func TestCeremonyCloseoutFailedFinalizerRendersFailureNotCompletion(t *testing.T) {
	saveGlobals(t)
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	goal := "Expose finalizer failure"
	state := colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateREADY,
		CurrentPhase: 1,
		Plan: colony.Plan{Phases: []colony.Phase{
			{ID: 1, Name: "Rejected completion", Status: colony.PhaseReady},
		}},
	}
	if err := store.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatalf("save colony state: %v", err)
	}

	completionFile := writeCeremonyTestJSON(t, map[string]interface{}{
		"ok":    false,
		"error": "build-finalize rejected completion file: dispatch_manifest contains no dispatches",
		"next":  "Fix the completion file and rerun `aether build-finalize 1 --completion-file <file>`.",
	})

	_, visual := renderCeremonyCloseout("build", completionFile)
	for _, want := range []string{
		"F I N A L I Z E R   F A I L E D",
		"build-finalize rejected completion file",
		"rerun `aether build-finalize 1 --completion-file <file>`",
	} {
		if !strings.Contains(visual, want) {
			t.Fatalf("failed-finalizer closeout missing %q\n%s", want, visual)
		}
	}
	for _, forbidden := range []string{
		"B U I L D   S U M M A R Y",
		"Run `aether continue`",
		"Workers: 0 completed  0 blocked  0 failed",
	} {
		if strings.Contains(visual, forbidden) {
			t.Fatalf("failed-finalizer closeout hid failure behind %q\n%s", forbidden, visual)
		}
	}
}

func TestPlanVisualOutput(t *testing.T) {
	pinRawCommandNames(t)
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	withWorkingDir(t, filepath.Dir(filepath.Dir(dataDir)))
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
	for _, want := range []string{"📋", "P L A N", "Choose Planning Preset", "Fast", "Balanced", "Deep", "Exhaustive", "Planning did not start. State: unchanged."} {
		if !strings.Contains(output, want) {
			t.Errorf("plan visual output missing %q\n%s", want, output)
		}
	}
	for _, forbidden := range []string{"P L A N   D I S P A T C H", "Planning Wave 1 starting", "aether build 1"} {
		if strings.Contains(output, forbidden) {
			t.Errorf("unselected preset crossed the planning authority boundary via %q\n%s", forbidden, output)
		}
	}
}

func TestBuildVisualOutputShowsSpawnPlan(t *testing.T) {
	pinRawCommandNames(t)
	saveGlobals(t)
	resetRootCmd(t)

	goal := "Improve command visuals"
	taskOneID := "1.1"
	taskTwoID := "1.2"
	accepted := createApprovedAcceptedBuildTestColony(t, colony.ColonyState{
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
	withWorkingDir(t, accepted.Root)
	t.Setenv("AETHER_OUTPUT_MODE", "visual")

	rootCmd.SetArgs([]string{"build", "1"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("build returned error: %v", err)
	}

	output := stdout.(*bytes.Buffer).String()
	if strings.Contains(output, `{"ok":true`) {
		t.Fatalf("expected visual output, got JSON: %s", output)
	}
	// Phase 193 (D-08): no watcher dispatch without an explicit Queen
	// proposal, so the spawn plan no longer has a "Post-Wave: Watcher" step.
	// Plan 194-02 (D-07): the Queen's Team card lists the required-castes
	// floor, which shrank to the builder alone. The dispatch rationale may
	// still name Watcher truthfully in its "not called" explanation.
	// Plan 194-05 (D-06, D-11): this fixture is a single-phase plan, so
	// position used to imply heavy review; that implicit escalation is gone,
	// so review depth is standard, not heavy. D-11 also removed the
	// no-proposal keyword-scoring fallback entirely -- Scout used to ride
	// along here on task 2's "Document" wording, but the fallback no longer
	// scores anything, so the dispatch count drops to 1 (builder alone).
	for _, want := range []string{"🔨", "B U I L D   D I S P A T C H   1", "S P A W N   P L A N", "Builder", "Total planned dispatches: 1", "Execution: serial", "single task in this wave", "aether continue", "── Context ──", "── Tasks ──", "── Dispatch ──", "── Verification [standard] ──", "── Housekeeping ──", "── Colony Complete ──", "it is safe to close this chat"} {
		if !strings.Contains(output, want) {
			t.Errorf("build visual output missing %q\n%s", want, output)
		}
	}
	for _, unwanted := range []string{"Post-Wave: Watcher", "Post-Wave: Probe"} {
		if strings.Contains(output, unwanted) {
			t.Errorf("build visual output unexpectedly contains %q with no explicit Queen proposal\n%s", unwanted, output)
		}
	}
}

func TestBuildVisualOutputShowsArtifactContract(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	goal := "Lock the build packet contract"
	taskID := "task-1"
	accepted := createApprovedAcceptedBuildTestColony(t, colony.ColonyState{
		Version: "3.0",
		Goal:    &goal,
		State:   colony.StateREADY,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{
					ID:     1,
					Name:   "Contract mapping",
					Status: colony.PhaseReady,
					Tasks: []colony.Task{
						{ID: &taskID, Goal: "Render the build contract", Status: colony.TaskPending},
					},
				},
			},
		},
	})
	withWorkingDir(t, accepted.Root)
	t.Setenv("AETHER_OUTPUT_MODE", "visual")

	rootCmd.SetArgs([]string{"build", "1"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("build returned error: %v", err)
	}

	output := stdout.(*bytes.Buffer).String()
	for _, want := range []string{
		"A R T I F A C T S",
		".aether/data/build/phase-1/manifest.json",
		".aether/data/last-build-claims.json",
		".aether/data/spawn-tree.txt",
		"── Context ──",
		"── Tasks ──",
		"── Dispatch ──",
		"it is safe to close this chat",
	} {
		if !strings.Contains(output, want) {
			t.Errorf("build visual output missing %q\n%s", want, output)
		}
	}
}

func TestRenderSpawnPlanForDispatchesShowsWaveExecutionStrategy(t *testing.T) {
	dispatches := []codexBuildDispatch{
		{Stage: "wave", Wave: 1, Caste: "builder", Name: "Forge-11", Task: "Task one", Status: "spawned"},
		{Stage: "wave", Wave: 1, Caste: "builder", Name: "Forge-12", Task: "Task two", Status: "spawned"},
		{Stage: "verification", Caste: "watcher", Name: "Keen-13", Task: "Verify it", Status: "spawned"},
	}

	output := renderSpawnPlanForDispatches(dispatches, colony.ModeInRepo)
	for _, want := range []string{
		"Wave 1",
		"Execution: serial",
		"dependency-independent tasks share the main working tree in in-repo mode",
		"set `aether parallel-mode set worktree` for isolated parallel builders",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("spawn plan output missing %q\n%s", want, output)
		}
	}
}

func TestColonizeVisualOutputShowsDispatchPreview(t *testing.T) {
	pinRawCommandNames(t)
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
	for _, want := range []string{"🗺️", "C O L O N I Z E   D I S P A T C H", "Survey Wave 1 starting", "Surveyors", "C O L O N I Z E", "aether plan"} {
		if !strings.Contains(output, want) {
			t.Errorf("colonize visual output missing %q\n%s", want, output)
		}
	}
}

func TestColonizeVisualOutputShowsSpawnTreeContract(t *testing.T) {
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

	goal := "Surface colonize contracts"
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
	if !strings.Contains(output, ".aether/data/spawn-tree.txt") {
		t.Errorf("colonize visual output missing spawn tree contract\n%s", output)
	}
}

func TestColonizeVisualOutputShowsDispatchContractDetails(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withWorkingDir(t, root)
	t.Setenv("AETHER_OUTPUT_MODE", "visual")

	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/colonize-visual-contract-test\n\ngo 1.24\n"), 0644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	goal := "Render colonize contracts honestly"
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
	for _, want := range []string{
		"Contract",
		"1 wave, parallel read-only worker execution",
		effectiveSurveyorDispatchTimeout(0).String() + " worker max",
		"One surveyor timing out does not reduce sibling surveyor budgets",
		"authenticated platform dispatcher",
		"dispatch_mode, survey_warning, provider_diagnostics, artifact_source",
	} {
		if !strings.Contains(output, want) {
			t.Errorf("colonize visual output missing %q\n%s", want, output)
		}
	}
}

func TestPlanVisualOutputShowsSpawnTreeContract(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	withWorkingDir(t, filepath.Dir(filepath.Dir(dataDir)))
	t.Setenv("AETHER_OUTPUT_MODE", "visual")

	goal := "Surface planning contracts"
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
	if strings.Contains(output, ".aether/data/spawn-tree.txt") {
		t.Errorf("preset selection prompt claimed a spawn tree before dispatch\n%s", output)
	}
	if _, err := os.Stat(filepath.Join(dataDir, "spawn-tree.txt")); !os.IsNotExist(err) {
		t.Errorf("preset selection prompt wrote a spawn tree before dispatch: %v", err)
	}
}

func TestPlanVisualOutputShowsDispatchContractDetails(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withWorkingDir(t, root)
	t.Setenv("AETHER_OUTPUT_MODE", "visual")

	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/plan-visual-contract-test\n\ngo 1.24\n"), 0644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "cmd"), 0755); err != nil {
		t.Fatalf("mkdir cmd: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "cmd", "main.go"), []byte("package main\n\nfunc main() {}\n"), 0644); err != nil {
		t.Fatalf("write main.go: %v", err)
	}

	goal := "Render plan contracts honestly"
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
	for _, want := range []string{
		"Planning contract: SPEC Unreported [APPROVED]",
		"Choose Planning Preset",
		"Fast       Target 80  Up to 4 passes",
		"Balanced   Target 90  Up to 6 passes",
		"Deep       Target 95  Up to 8 passes",
		"Exhaustive Target 99  Up to 12 passes",
		"Planning did not start. State: unchanged.",
	} {
		if !strings.Contains(output, want) {
			t.Errorf("plan visual output missing %q\n%s", want, output)
		}
	}
	if strings.Contains(output, "2 staged workers") || strings.Contains(output, "authenticated platform dispatcher") {
		t.Errorf("preset selection prompt exposed a dispatch contract before owner selection\n%s", output)
	}
}

func TestContinueVisualOutputShowsVerificationArtifactsAndSpawnTree(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withTestWorkspace(t, root)
	withWorkingDir(t, root)
	t.Setenv("AETHER_OUTPUT_MODE", "visual")

	goal := "Surface continue contracts"
	now := mustParseRFC3339(t, "2026-04-20T11:00:00Z")
	taskID := "1.1"
	nextTaskID := "2.1"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:        "3.0",
		Goal:           &goal,
		State:          colony.StateBUILT,
		CurrentPhase:   1,
		BuildStartedAt: &now,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{
					ID:     1,
					Name:   "Verify contracts",
					Status: colony.PhaseInProgress,
					Tasks:  []colony.Task{{ID: &taskID, Goal: "Verify the build packet", Status: colony.TaskInProgress}},
				},
				{
					ID:     2,
					Name:   "Next phase",
					Status: colony.PhasePending,
					Tasks:  []colony.Task{{ID: &nextTaskID, Goal: "Keep going", Status: colony.TaskPending}},
				},
			},
		},
	})

	dispatches := []codexBuildDispatch{
		{Stage: "wave", Wave: 1, Caste: "builder", Name: "Forge-41", Task: "Verify the build packet", Status: "spawned", TaskID: taskID},
		{Stage: "verification", Caste: "watcher", Name: "Keen-42", Task: "Independent verification before advancement", Status: "spawned"},
	}
	seedContinueBuildPacket(t, dataDir, 1, "Verify contracts", goal, dispatches)

	rootCmd.SetArgs([]string{"continue"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("continue returned error: %v", err)
	}

	output := stdout.(*bytes.Buffer).String()
	expectedWatcher := deterministicAntName("watcher", "phase:1:continue:watcher")
	for _, want := range []string{
		"── Verification ──",
		"Phase 1 verified and completed: Verify contracts",
		"Continue Worker Flow",
		expectedWatcher + " completed",
		"Watcher " + expectedWatcher + " completed independent verification before advancement",
		"Signal housekeeping completed",
		"── Housekeeping ──",
		"A R T I F A C T S",
		"Workers",
		"Forge-41",
		"Keen-42",
		"Verification passed during continue",
		"── Next Phase ──",
		".aether/data/build/phase-1/verification.json",
		".aether/data/build/phase-1/gates.json",
		".aether/data/build/phase-1/review.json",
		".aether/data/build/phase-1/continue.json",
		".aether/data/spawn-tree.txt",
		"it is safe to close this chat",
	} {
		if !strings.Contains(output, want) {
			t.Errorf("continue visual output missing %q\n%s", want, output)
		}
	}
	if strings.Contains(output, "Forge-41 [builder] completed") {
		t.Fatalf("expected continue worker flow to avoid builder closure entries, got:\n%s", output)
	}
}

func TestContinueBlockedVisualOutputShowsWorkerFlow(t *testing.T) {
	pinRawCommandNames(t)
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withTestWorkspace(t, root)
	withWorkingDir(t, root)
	t.Setenv("AETHER_OUTPUT_MODE", "visual")

	goal := "Render blocked continue flow honestly"
	now := mustParseRFC3339(t, "2026-04-22T12:36:18Z")
	taskID := "1.1"
	nextTaskID := "2.1"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:        "3.0",
		Goal:           &goal,
		State:          colony.StateBUILT,
		CurrentPhase:   1,
		BuildStartedAt: &now,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{
					ID:     1,
					Name:   "Blocked flow",
					Status: colony.PhaseInProgress,
					Tasks:  []colony.Task{{ID: &taskID, Goal: "Stay blocked until watcher approval", Status: colony.TaskInProgress}},
				},
				{
					ID:     2,
					Name:   "Still pending",
					Status: colony.PhasePending,
					Tasks:  []colony.Task{{ID: &nextTaskID, Goal: "Wait for a clean verification", Status: colony.TaskPending}},
				},
			},
		},
	})

	seedContinueBuildPacket(t, dataDir, 1, "Blocked flow", goal, []codexBuildDispatch{
		{Stage: "wave", Wave: 1, Caste: "builder", Name: "Forge-141", Task: "Stay blocked until watcher approval", Status: "completed", TaskID: taskID},
		{Stage: "verification", Caste: "watcher", Name: "Keen-build-142", Task: "Build-time verification", Status: "completed"},
	})

	invoker := &continueWatcherTestInvoker{
		watcherStatus:  "blocked",
		watcherSummary: "Continue watcher rejected the phase",
	}
	originalInvoker := newCodexWorkerInvoker
	newCodexWorkerInvoker = func() codex.WorkerInvoker { return invoker }
	t.Cleanup(func() { newCodexWorkerInvoker = originalInvoker })

	rootCmd.SetArgs([]string{"continue"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("continue returned error: %v", err)
	}

	output := stdout.(*bytes.Buffer).String()
	for _, want := range []string{
		"C O N T I N U E   B L O C K E D",
		"Continue Worker Flow",
		"Continue watcher rejected the phase",
		"blocked",
		"W H A T   N E X T",
		"A R T I F A C T S",
		".aether/data/build/phase-1/verification.json",
		".aether/data/build/phase-1/gates.json",
		".aether/data/build/phase-1/continue.json",
		".aether/data/spawn-tree.txt",
	} {
		if !strings.Contains(output, want) {
			t.Errorf("blocked continue visual output missing %q\n%s", want, output)
		}
	}
	if strings.Contains(output, "Run `aether continue` to recover the blocked work") {
		t.Fatalf("blocked continue suggested the identical no-op retry:\n%s", output)
	}
	if strings.Contains(output, "/ant-") {
		t.Fatalf("Codex blocked guidance contains unsupported slash commands:\n%s", output)
	}
	if strings.Contains(output, "Forge-141 [builder] completed") {
		t.Fatalf("expected blocked continue worker flow to avoid builder closure entries, got:\n%s", output)
	}
}

func TestBlockedContinueRecoveryHintsTranslateFromCanonicalCommands(t *testing.T) {
	result := map[string]interface{}{
		"blocking_issues": []interface{}{"verification failed"},
		"gates": map[string]interface{}{
			"checks": []interface{}{
				map[string]interface{}{
					"name":             "verification_steps_passed",
					"passed":           false,
					"fix_hint":         "Fix manually and run aether continue",
					"recovery_options": []interface{}{"Run aether unblock --dispatch for guided recovery", "Resolve the flag with aether flag-resolve --id <id> --message \"what fixed it\""},
				},
			},
		},
	}

	for _, tc := range []struct {
		platform string
		want     []string
		notWant  []string
	}{
		{platform: "codex", want: []string{"aether continue", "aether unblock --dispatch", "aether flag-resolve"}, notWant: []string{"/ant-"}},
		{platform: "claude", want: []string{"/ant-continue", "/ant-unblock --dispatch", "aether flag-resolve"}, notWant: []string{"Fix manually and run aether continue", "Run aether unblock"}},
		{platform: "opencode", want: []string{"/ant-continue", "/ant-unblock --dispatch", "aether flag-resolve"}, notWant: []string{"Fix manually and run aether continue", "Run aether unblock"}},
	} {
		t.Run(tc.platform, func(t *testing.T) {
			t.Setenv("AETHER_OUTPUT_MODE", "visual")
			t.Setenv("AETHER_PLATFORM", tc.platform)
			raw := renderContinueBlockedVisual(
				colony.ColonyState{},
				colony.Phase{ID: 2, Name: "Blocked phase"},
				result,
				colony.VerificationDepthStandard,
			)
			var output bytes.Buffer
			writeVisualOutput(&output, raw)
			for _, want := range tc.want {
				if !strings.Contains(output.String(), want) {
					t.Errorf("%s blocked output missing %q:\n%s", tc.platform, want, output.String())
				}
			}
			for _, notWant := range tc.notWant {
				if strings.Contains(output.String(), notWant) {
					t.Errorf("%s blocked output contains unsupported form %q:\n%s", tc.platform, notWant, output.String())
				}
			}
		})
	}
}

func TestContinueVisualOutputShowsColonyCompleteStageMarker(t *testing.T) {
	pinRawCommandNames(t)
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withTestWorkspace(t, root)
	withWorkingDir(t, root)
	t.Setenv("AETHER_OUTPUT_MODE", "visual")

	goal := "Complete the colony honestly"
	now := mustParseRFC3339(t, "2026-04-20T11:15:00Z")
	taskID := "1.1"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:        "3.0",
		Goal:           &goal,
		State:          colony.StateBUILT,
		CurrentPhase:   1,
		BuildStartedAt: &now,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{
					ID:     1,
					Name:   "Finish the final slice",
					Status: colony.PhaseInProgress,
					Tasks:  []colony.Task{{ID: &taskID, Goal: "Finish cleanly", Status: colony.TaskInProgress}},
				},
			},
		},
	})

	dispatches := []codexBuildDispatch{
		{Stage: "wave", Wave: 1, Caste: "builder", Name: "Forge-51", Task: "Finish cleanly", Status: "spawned", TaskID: taskID},
		{Stage: "verification", Caste: "watcher", Name: "Keen-52", Task: "Independent verification before advancement", Status: "spawned"},
	}
	seedContinueBuildPacket(t, dataDir, 1, "Finish the final slice", goal, dispatches)

	rootCmd.SetArgs([]string{"continue"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("continue returned error: %v", err)
	}

	output := stdout.(*bytes.Buffer).String()
	for _, want := range []string{
		"Phase 1 verified and completed: Finish the final slice",
		// Widened for Phase "Classic Visual Voice" plan 04: the continue
		// screen's stage marker now names "project" beside "colony" (the
		// widened plain-English scan), unlike the build screen's marker,
		// which is unchanged.
		"── Project Complete (Colony) ──",
		"Every phase in the plan is finished.",
		"aether seal",
		"it is safe to close this chat",
	} {
		if !strings.Contains(output, want) {
			t.Errorf("continue visual output missing %q\n%s", want, output)
		}
	}
}

func TestWatchVisualOutputShowsHonestIdleFallback(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	t.Setenv("AETHER_OUTPUT_MODE", "visual")

	goal := "Watch snapshot contracts"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		Scope:        colony.ScopeMeta,
		State:        colony.StateEXECUTING,
		CurrentPhase: 1,
		Plan: colony.Plan{
			Phases: []colony.Phase{{ID: 1, Name: "Execution", Status: colony.PhaseInProgress}},
		},
	})

	spawnTree := agent.NewSpawnTree(store, "spawn-tree.txt")
	if err := spawnTree.RecordSpawn("Queen", "builder", "Hammer-9", "Inspect the watch contract", 1); err != nil {
		t.Fatalf("record spawn: %v", err)
	}
	if err := spawnTree.UpdateStatus("Hammer-9", "active", "Running"); err != nil {
		t.Fatalf("mark active: %v", err)
	}

	rootCmd.SetArgs([]string{"watch"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("watch returned error: %v", err)
	}

	output := stdout.(*bytes.Buffer).String()
	for _, want := range []string{
		"No ants are active right now",
		"Status is the authoritative snapshot",
		"Projection revision: " + LifecycleProjectionRevision,
		"Recent recorded activity",
		"Actor: Hammer-9",
		"recorded evidence only",
		"Live event source: unsupported until Phase 202",
	} {
		if !strings.Contains(output, want) {
			t.Errorf("watch visual output missing %q\n%s", want, output)
		}
	}
	for _, forbidden := range []string{
		"Active Workers",
		".aether/data/watch-status.txt",
		".aether/data/watch-progress.txt",
		"Run in a TTY for live refresh.",
	} {
		if strings.Contains(output, forbidden) {
			t.Errorf("idle watch visual output contains obsolete live claim %q\n%s", forbidden, output)
		}
	}
}

func TestWatchLiveRefreshRequiresTTY(t *testing.T) {
	t.Setenv("AETHER_OUTPUT_MODE", "visual")

	if shouldUseLiveWatchRefresh(&bytes.Buffer{}, false) {
		t.Fatal("expected non-TTY visual output to stay snapshot-friendly")
	}
	if shouldUseLiveWatchRefresh(&bytes.Buffer{}, true) {
		t.Fatal("expected --once style behavior to disable live refresh")
	}
}

func TestPrintNextUpVisualOutput(t *testing.T) {
	pinRawCommandNames(t)
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	t.Setenv("AETHER_OUTPUT_MODE", "visual")

	goal := "Test next-up visuals"
	name := "test-colony"
	now := time.Now().UTC()
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:        "3.0",
		Goal:           &goal,
		ColonyName:     &name,
		State:          colony.StateEXECUTING,
		CurrentPhase:   1,
		BuildStartedAt: &now,
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

func TestColorizeCasteUsesANSIForVisualOutput(t *testing.T) {
	saveGlobals(t)

	var buf bytes.Buffer
	stdout = &buf
	t.Setenv("AETHER_OUTPUT_MODE", "visual")
	t.Setenv("NO_COLOR", "")

	got := colorizeCaste("builder", "builder")
	if !strings.Contains(got, "\x1b[33m") {
		t.Fatalf("expected ANSI-highlighted builder caste text, got %q", got)
	}
	if !strings.Contains(got, "builder") || !strings.Contains(got, "\x1b[0m") {
		t.Fatalf("expected ANSI-highlighted builder label, got %q", got)
	}
}

func TestShouldUseANSIColorsUsesVisualModeOrForce(t *testing.T) {
	saveGlobals(t)

	var buf bytes.Buffer
	stdout = &buf
	t.Setenv("AETHER_OUTPUT_MODE", "visual")
	t.Setenv("NO_COLOR", "")

	if !shouldUseANSIColors() {
		t.Fatal("expected visual mode to allow ANSI colors for caste highlighting")
	}

	t.Setenv("AETHER_OUTPUT_MODE", "json")
	t.Setenv("AETHER_FORCE_COLOR", "1")
	if !shouldUseANSIColors() {
		t.Fatal("expected AETHER_FORCE_COLOR to override ANSI detection")
	}
}

func TestInstallVisualOutput(t *testing.T) {
	// Manages its own hub via --home-dir; opt out of suite-wide hub isolation.
	t.Setenv("AETHER_HUB_DIR", "")
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
	// Pinned to Codex: this test asserts output content, and Codex is the
	// platform that names commands in the raw CLI form. Naming itself is
	// covered by TestVisualOutputNeverLeaksRawWrapperCommands.
	t.Setenv("AETHER_PLATFORM", "codex")

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
	for _, want := range []string{"📦", "I N S T A L L", "█████╗", "aether lay-eggs", "shared `aether` binary", "Published release repos:", "aether update --force --download-binary", "aether update --force"} {
		if !strings.Contains(output, want) {
			t.Errorf("install visual output missing %q\n%s", want, output)
		}
	}
}

func TestRenderBinaryActionVisualPublishGuidanceSeparatesRepoSetupFromUpdate(t *testing.T) {
	pinRawCommandNames(t)
	output := renderBinaryActionVisual("Publish Complete", "Aether v1.0.24 published", "1.0.24", "/tmp/home/.aether")

	for _, want := range []string{
		"Existing repos: run `aether update --force` to refresh companion files from the hub.",
		"New repos: run `aether lay-eggs` to set up Aether.",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("publish visual output missing %q\n%s", want, output)
		}
	}
	if strings.Contains(output, "lay-eggs` in a repo to use the refreshed binary and companion files") {
		t.Fatalf("publish visual output still uses ambiguous lay-eggs guidance:\n%s", output)
	}
}

func TestRenderUpdateVisualNoChangesSaysNoFollowUpRequired(t *testing.T) {
	output := renderUpdateVisual(
		"/tmp/example",
		"1.0.7",
		"1.0.7",
		"",
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
		"unchanged",
		true,
		map[string]interface{}{},
	)

	if !strings.Contains(output, "Binary: already at the current hub version") {
		t.Fatalf("expected binary-already-current message in update visual, got:\n%s", output)
	}
	// The closing block used to hand-write a "No follow-up is required"
	// sentence for this exact case; that recommendation is the one resolver's
	// job now (Phase 197 plan 06), so this only asserts the shared card is
	// present rather than a specific hand-typed sentence.
	if !strings.Contains(output, spacedTitle("What Next")) {
		t.Fatalf("expected update to end with the shared closing card, got:\n%s", output)
	}
}

func TestRenderUpdateVisualShowsRemovedAssets(t *testing.T) {
	pinRawCommandNames(t)
	output := renderUpdateVisual(
		"/tmp/example",
		"1.0.27",
		"1.0.27",
		"",
		true,
		false,
		[]map[string]interface{}{
			{"label": "Repo .aether cleanup", "copied": 0, "skipped": 0, "removed": 7},
			{"label": "Prune legacy repo platform assets", "copied": 0, "skipped": 0, "removed": 3},
		},
		0,
		0,
		nil,
		"unchanged",
		true,
		map[string]interface{}{},
	)

	if !strings.Contains(output, "Assets: 0 copied, 0 unchanged, 10 removed") {
		t.Fatalf("expected top-level removal total, got:\n%s", output)
	}
	if !strings.Contains(output, "Repo .aether cleanup — 0 copied, 0 unchanged, 7 removed") {
		t.Fatalf("expected per-asset removal detail, got:\n%s", output)
	}
	if strings.Contains(output, "No follow-up is required.") {
		t.Fatalf("removed files should not be reported as a no-change update:\n%s", output)
	}
	// The specific "Run `aether status`..." sentence was a hand-typed
	// constant; what to do next is the one resolver's job now (Phase 197
	// plan 06), so this only asserts the shared card is present.
	if !strings.Contains(output, spacedTitle("What Next")) {
		t.Fatalf("expected update to end with the shared closing card, got:\n%s", output)
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

func TestWorkflowSuggestionsForInterruptedExecutingSuggestsResume(t *testing.T) {
	goal := "Interrupted build should reconcile safely"
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

	if !strings.Contains(primary, "aether resume") {
		t.Fatalf("expected interrupted executing colony to suggest canonical resume, got: %s", primary)
	}
}

func TestWorkflowSuggestionsBlockBuildAfterPlanFinalizeFailure(t *testing.T) {
	saveGlobals(t)

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s
	if err := store.SaveJSON("pending-decisions.json", colony.FlagsFile{
		Decisions: []colony.FlagEntry{{
			ID:          "flag-plan-finalize",
			Type:        "blocker",
			Description: "Planning finalization failed: invalid depends_on",
			Source:      planFinalizeFailureSource,
			CreatedAt:   time.Now().UTC().Format(time.RFC3339),
		}},
	}); err != nil {
		t.Fatalf("save flags: %v", err)
	}

	goal := "Do not build after failed plan-finalize"
	primary, alternatives := workflowSuggestionsForState(colony.ColonyState{
		Goal:         &goal,
		State:        colony.StateREADY,
		CurrentPhase: 1,
		Plan: colony.Plan{
			Phases: []colony.Phase{{ID: 1, Name: "Unsafe phase", Status: colony.PhaseReady}},
		},
	})
	if !strings.Contains(primary, "aether resume") {
		t.Fatalf("expected blocked lifecycle to use canonical resume, got: %s", primary)
	}
	all := primary + "\n" + strings.Join(alternatives, "\n")
	if strings.Contains(all, "aether build 1") {
		t.Fatalf("failed finalization must not suggest a direct build:\n%s", all)
	}
	if !strings.Contains(all, "aether status") || !strings.Contains(all, "aether history") {
		t.Fatalf("expected evidence-preserving alternatives, got:\n%s", all)
	}
}

func TestSetupVisualOutput(t *testing.T) {
	pinRawCommandNames(t)
	// Manages its own hub via --home-dir; opt out of suite-wide hub isolation.
	t.Setenv("AETHER_HUB_DIR", "")
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
	// Manages its own hub via --home-dir; opt out of suite-wide hub isolation.
	t.Setenv("AETHER_HUB_DIR", "")
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
	// Pinned to Codex: this test asserts output content, and Codex is the
	// platform that names commands in the raw CLI form. Naming itself is
	// covered by TestVisualOutputNeverLeaksRawWrapperCommands.
	t.Setenv("AETHER_PLATFORM", "codex")

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
	for _, want := range []string{"🔄", "U P D A T E", "█████╗", "Dry run complete", "aether update", "Published release:", "Local source checkout:"} {
		if !strings.Contains(output, want) {
			t.Errorf("update visual output missing %q\n%s", want, output)
		}
	}
}

func TestPauseResumePatrolPhaseAndHistoryVisualOutput(t *testing.T) {
	pinRawCommandNames(t)
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
						{ID: &taskOneID, Goal: "Write pause handoff", Status: colony.TaskCompleted},
						{ID: &taskTwoID, Goal: "Write resume orientation", Status: colony.TaskInProgress},
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

	checkVisual([]string{"pause"}, "💾", "P A U S E   C O L O N Y", "HANDOFF.md", "aether resume")
	checkVisual([]string{"resume"}, "💾", "R E S U M E   C O L O N Y", "Session UX", "Active Signals", "Blockers", "Survey Context", "Source:", "aether build 1")
	checkVisual([]string{"patrol"}, "📊", "P A T R O L", "Signals: 1 active")
	checkVisual([]string{"phase"}, "🧱", "Session UX", "Write resume orientation")
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
			map[string]interface{}{"name": "Nest-42", "caste": "surveyor-nest", "task": "Map architecture", "status": "completed", "summary": "Mapped the chamber layout", "duration": 12.3},
			map[string]interface{}{"name": "Disc-7", "caste": "surveyor-disciplines", "task": "Map disciplines", "status": "completed", "duration": 8.1},
			map[string]interface{}{"name": "Path-3", "caste": "surveyor-pathogens", "task": "Identify pathogens", "status": "failed", "summary": "Blocked on missing evidence", "duration": 5.2},
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
	if !strings.Contains(output, "Mapped the chamber layout") {
		t.Errorf("real dispatch output missing worker summary\n%s", output)
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

func TestRenderColonizeVisual_ShowsSurveyFallbackWarning(t *testing.T) {
	result := map[string]interface{}{
		"root":           "/tmp/test",
		"detected_type":  "go",
		"languages":      []interface{}{"go"},
		"frameworks":     []interface{}{"cobra"},
		"domains":        []interface{}{"cli"},
		"dispatch_mode":  "fallback",
		"survey_warning": "Real surveyors did not finish cleanly, so Aether fell back to local survey synthesis.",
		"stats": map[string]interface{}{
			"files":       42,
			"directories": 7,
		},
		"survey_dir": "/tmp/test/.aether/data/survey",
		"surveyors": []interface{}{
			map[string]interface{}{"name": "Nest-42", "caste": "surveyor-nest", "task": "Map architecture", "status": "timeout", "summary": "timed out", "duration": 60.0},
			map[string]interface{}{"name": "Disc-7", "caste": "surveyor-disciplines", "task": "Map disciplines", "status": "timeout", "summary": "timed out", "duration": 0.0},
		},
		"survey_files": []interface{}{"BLUEPRINT.md", "CHAMBERS.md"},
	}

	output := renderColonizeVisual(result)

	if !strings.Contains(output, "Survey Warning") {
		t.Errorf("output missing survey warning header\n%s", output)
	}
	if !strings.Contains(output, "fell back to local survey synthesis") {
		t.Errorf("output missing survey fallback explanation\n%s", output)
	}
	if !strings.Contains(output, "Dispatch: Fallback") {
		t.Errorf("output missing fallback dispatch label\n%s", output)
	}
}

func TestRenderSurveyorResults_Formatting(t *testing.T) {
	surveyors := []codexSurveyorDispatch{
		{Caste: "surveyor-nest", Name: "Nest-42", Task: "Map architecture", Status: "completed", Summary: "Mapped chamber layout", Duration: 12.3},
		{Caste: "surveyor-disciplines", Name: "Disc-7", Task: "Map disciplines", Status: "completed", Duration: 8.1},
		{Caste: "surveyor-pathogens", Name: "Path-3", Task: "Identify pathogens", Status: "failed", Summary: "Blocked by missing config", Duration: 5.2},
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

	// Should use caste label "Surveyor" from casteLabelMap
	if !strings.Contains(output, "Surveyor") {
		t.Errorf("missing Surveyor caste label\n%s", output)
	}
	// Should use single emoji from casteEmojiMap (surveyor = bar chart)
	if !strings.Contains(output, "\U0001f4ca") {
		t.Errorf("missing surveyor emoji\n%s", output)
	}

	// Completed should have checkmark, failed should have X
	if !strings.Contains(output, "\u2713") || !strings.Contains(output, "completed") {
		t.Errorf("missing completed worker status details\n%s", output)
	}
	if !strings.Contains(output, "\u2717") || !strings.Contains(output, "failed") {
		t.Errorf("missing failed worker status details\n%s", output)
	}

	// Should have durations
	if !strings.Contains(output, "12.3s") {
		t.Errorf("missing duration 12.3s\n%s", output)
	}
	if !strings.Contains(output, "Task: Map architecture") {
		t.Errorf("missing task detail line\n%s", output)
	}
	if !strings.Contains(output, "Mapped chamber layout") {
		t.Errorf("missing worker summary line\n%s", output)
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

func TestCasteIdentitySurveyorSubtypes(t *testing.T) {
	// Surveyor subtypes should resolve to the Surveyor emoji/label/color, not generic Ant.
	os.Setenv("AETHER_FORCE_COLOR", "1")
	defer os.Unsetenv("AETHER_FORCE_COLOR")

	subtypes := []string{"surveyor-nest", "surveyor-pathogens", "surveyor-provisions", "surveyor-disciplines"}
	for _, subtype := range subtypes {
		identity := casteIdentity(subtype)
		if !strings.Contains(identity, "📊") {
			t.Errorf("casteIdentity(%q): expected 📊 emoji, got %q", subtype, identity)
		}
		if !strings.Contains(identity, "Surveyor") {
			t.Errorf("casteIdentity(%q): expected 'Surveyor' label, got %q", subtype, identity)
		}
		emoji := casteEmoji(subtype)
		if emoji != "📊" {
			t.Errorf("casteEmoji(%q): expected 📊, got %q", subtype, emoji)
		}
		label := casteLabel(subtype)
		if !strings.Contains(label, "Surveyor") {
			t.Errorf("casteLabel(%q): expected 'Surveyor', got %q", subtype, label)
		}
		color := casteANSIColor(subtype)
		if color == "" {
			t.Errorf("casteANSIColor(%q): expected a color code, got empty", subtype)
		}
	}
}

func TestCasteIdentityUnknownCaste(t *testing.T) {
	os.Setenv("AETHER_FORCE_COLOR", "1")
	defer os.Unsetenv("AETHER_FORCE_COLOR")

	// Unknown castes should fall back to generic Ant.
	identity := casteIdentity("unknown-worker")
	if !strings.Contains(identity, "🐜") {
		t.Errorf("expected 🐜 fallback for unknown caste, got %q", identity)
	}
	if !strings.Contains(identity, "Ant") {
		t.Errorf("expected 'Ant' fallback for unknown caste, got %q", identity)
	}
}

func TestCasteIdentityExactMatchPreferred(t *testing.T) {
	os.Setenv("AETHER_FORCE_COLOR", "1")
	defer os.Unsetenv("AETHER_FORCE_COLOR")

	// Exact matches should be used, not prefix fallback.
	identity := casteIdentity("builder")
	if !strings.Contains(identity, "🔨") {
		t.Errorf("expected 🔨 for builder, got %q", identity)
	}
	if !strings.Contains(identity, "Builder") {
		t.Errorf("expected 'Builder' label, got %q", identity)
	}
}

func TestDispatchStatusIcon_DistinguishesRunningAndTimeout(t *testing.T) {
	if got := dispatchStatusIcon("running"); got != "…" {
		t.Fatalf("dispatchStatusIcon(running) = %q, want ellipsis", got)
	}
	if got := dispatchStatusIcon("timeout"); got != "✗" {
		t.Fatalf("dispatchStatusIcon(timeout) = %q, want failure mark", got)
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

func TestRenderPlanVisualAgentDelegatePlanOnly(t *testing.T) {
	result := map[string]interface{}{
		"plan_only":             true,
		"agent_delegate":        true,
		"existing_plan":         false,
		"goal":                  "Refresh plan through host agents",
		"granularity":           "sprint",
		"dispatch_mode":         "agent-delegate",
		"agent_delegate_reason": "agent-delegate session detected on opencode; host platform must dispatch workers directly",
		"dispatches": []interface{}{
			map[string]interface{}{"name": "Seek-40", "caste": "scout", "task": "Survey the repo", "status": "planned"},
			map[string]interface{}{"name": "Carrier-52", "caste": "route_setter", "task": "Convert findings into phases", "status": "planned"},
		},
	}

	output := renderPlanVisual(result)
	for _, want := range []string{
		"Agent-Delegate",
		"host platform must dispatch workers directly",
		// Phase 197-04 removed the hand-written "Host platform should dispatch
		// the JSON `plan_manifest` ..." instruction line: next-step guidance now
		// comes from the one shared card, and the manifest itself still reaches
		// the wrapper through result.plan_manifest in the JSON envelope, which
		// is what .opencode/commands/ant/plan.md actually reads ("Reads:
		// result.plan_manifest"). The line above still asserts the dispatch
		// intent, so the guarantee this test exists for is unchanged.
		"`aether plan-finalize --completion-file <file>`",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("agent-delegate plan visual missing %q\n%s", want, output)
		}
	}
}

func TestRenderPlanVisualPlanOnlyPlannedDispatchesAreNotExecutionResults(t *testing.T) {
	result := map[string]interface{}{
		"plan_only":          true,
		"existing_plan":      false,
		"requires_finalizer": true,
		"goal":               "Preview planning work honestly",
		"granularity":        "sprint",
		"dispatch_mode":      "plan-only",
		"dispatches": []interface{}{
			map[string]interface{}{"name": "Seek-70", "caste": "scout", "task": "Survey the repo", "status": "planned"},
			map[string]interface{}{"name": "Route-70", "caste": "route_setter", "task": "Convert findings into phases", "status": "planned"},
		},
	}

	output := renderPlanVisual(result)
	for _, forbidden := range []string{
		"Scout and Route-Setter mapped the colony goal into executable phases.",
		"Dispatch: Real",
		"workers completed",
	} {
		if strings.Contains(output, forbidden) {
			t.Fatalf("plan-only visual treated planned dispatches as execution results via %q\n%s", forbidden, output)
		}
	}
	// The guarantee here is that a PLANNED dispatch is never shown as work that
	// already happened. Phase 197-04 dropped the literal `plan_manifest` token
	// from the visual (the manifest still travels in the JSON envelope), so the
	// pending-ness is now carried by the wording and the dispatch mode. Assert
	// those, not the removed token -- deleting the check would give up the
	// guarantee, and asserting the token would only prove a string survived.
	for _, want := range []string{
		// The dispatch-mode line is the discriminating one: the forbidden list
		// above rejects "Dispatch: Real", so asserting the exact "Dispatch:
		// Plan-only" line closes the pair. A loose token like "prepared" does
		// NOT close it -- verified by mutation: changing the prose word left
		// this test green, so it was asserting a string rather than the
		// guarantee.
		"Dispatch: Plan-only",
		"Seek-70",
		"Route-70",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("plan-only visual missing pending cue %q\n%s", want, output)
		}
	}
}

func TestRenderPlanVisualExistingPlanNoFinalizerDoesNotPromptPlanningWorkers(t *testing.T) {
	phases := []colony.Phase{{
		ID:     1,
		Name:   "Existing phase",
		Status: colony.PhaseReady,
		Tasks:  []colony.Task{{Goal: "Keep using the existing plan"}},
	}}
	phaseMaps := make([]interface{}, len(phases))
	for i, phase := range phases {
		phaseMaps[i] = phaseToMap(phase)
	}
	result := map[string]interface{}{
		"plan_only":          true,
		"existing_plan":      true,
		"requires_finalizer": false,
		"goal":               "Reuse the current plan",
		"phases":             phaseMaps,
	}

	output := renderPlanVisual(result)
	if !strings.Contains(output, "Existing colony plan loaded") {
		t.Fatalf("existing-plan visual missing no-op message\n%s", output)
	}
	for _, forbidden := range []string{
		"Scout and Route-Setter mapped",
		"Use the JSON `plan_manifest`",
		"plan-finalize",
		"\nWorkers\n",
		"workers completed",
	} {
		if strings.Contains(output, forbidden) {
			t.Fatalf("existing-plan/no-finalizer visual implied planning worker execution via %q\n%s", forbidden, output)
		}
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
			map[string]interface{}{"name": "Scout-7", "caste": "scout", "task": "Survey the repo", "status": "completed", "summary": "Mapped the runtime terrain", "duration": 3.5},
			map[string]interface{}{"name": "Route-12", "caste": "route_setter", "task": "Convert findings into phases", "status": "completed", "summary": "Shaped the next phases", "duration": 2.1},
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
	if !strings.Contains(output, "Mapped the runtime terrain") {
		t.Errorf("real dispatch output missing scout summary\n%s", output)
	}
	// Should show summary line
	if !strings.Contains(output, "2/2 workers completed") {
		t.Errorf("real dispatch output missing summary '2/2 workers completed'\n%s", output)
	}
	// Should show "Workers" section header (Phase 202.1 plan 06: glyph-led)
	if !strings.Contains(output, "\n🐜 Workers\n") {
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
			map[string]interface{}{"name": "Scout-7", "caste": "scout", "task": "Survey the repo", "status": "completed", "summary": "Captured the repo shape", "duration": 4.0},
			map[string]interface{}{"name": "Route-12", "caste": "route_setter", "task": "Convert findings into phases", "status": "failed", "summary": "Blocked by missing planning file", "duration": 1.2},
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

func TestRenderPlanVisual_ShowsPlanningFallbackWarning(t *testing.T) {
	phases := []colony.Phase{
		{ID: 1, Name: "Discovery", Description: "Map the codebase", Status: colony.PhaseReady,
			Tasks: []colony.Task{{Goal: "Read code paths"}}},
	}
	phaseMaps := make([]interface{}, len(phases))
	for i, p := range phases {
		phaseMaps[i] = phaseToMap(p)
	}

	result := map[string]interface{}{
		"existing_plan":    false,
		"goal":             "Build feature X",
		"granularity":      "milestone",
		"confidence":       map[string]interface{}{"overall": 72},
		"phases":           phaseMaps,
		"dispatch_mode":    "fallback",
		"planning_warning": "Real planning workers did not finish cleanly, so Aether fell back to local synthesis.",
		"dispatches": []interface{}{
			map[string]interface{}{"name": "Scout-7", "caste": "scout", "task": "Survey the repo", "status": "timeout", "summary": "timed out", "duration": 60.0},
			map[string]interface{}{"name": "Route-12", "caste": "route_setter", "task": "Convert findings into phases", "status": "timeout", "summary": "timed out", "duration": 0.0},
		},
	}

	output := renderPlanVisual(result)

	if !strings.Contains(output, "Planning Warning") {
		t.Errorf("output missing planning warning header\n%s", output)
	}
	if !strings.Contains(output, "fell back to local synthesis") {
		t.Errorf("output missing fallback explanation\n%s", output)
	}
	if !strings.Contains(output, "Dispatch: Fallback") {
		t.Errorf("output missing fallback dispatch label\n%s", output)
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

func TestRenderPlanVisual_IncludesTaskMetadata(t *testing.T) {
	phaseMaps := []interface{}{
		map[string]interface{}{
			"id":               1,
			"name":             "Planning orchestration",
			"description":      "Add a scout plus route-setter planning pass.",
			"status":           colony.PhaseReady,
			"success_criteria": []interface{}{"Plan generation is ant-driven", "The colony has a grounded next phase"},
			"tasks": []interface{}{
				map[string]interface{}{
					"id":               "1.1",
					"goal":             "Generate a route-setter plan with task constraints, hints, and success criteria",
					"status":           colony.TaskPending,
					"constraints":      []interface{}{"The first phase must become ready", "The saved plan must match the displayed plan"},
					"hints":            []interface{}{"COLONY_STATE.json", "renderPlanVisual"},
					"success_criteria": []interface{}{"Plan generation is grounded in repo context", "Spawn records show scout and route-setter activity"},
					"depends_on":       []interface{}{"0.2"},
				},
			},
		},
	}

	result := map[string]interface{}{
		"existing_plan": false,
		"goal":          "Ground the plan in worker artifacts",
		"phases":        phaseMaps,
	}

	output := renderPlanVisual(result)

	for _, want := range []string{
		"Task 1.1",
		"Constraints:",
		"The first phase must become ready",
		"Hints:",
		"COLONY_STATE.json",
		"Success Criteria:",
		"Plan generation is grounded in repo context",
		"Depends on: 0.2",
		"Phase Success Criteria:",
		"The colony has a grounded next phase",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected plan visual to contain %q\n%s", want, output)
		}
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

func TestCasteIdentityAllCastes(t *testing.T) {
	t.Setenv("AETHER_FORCE_COLOR", "1")
	defer os.Unsetenv("AETHER_FORCE_COLOR")

	for caste := range casteEmojiMap {
		identity := casteIdentity(caste)
		emoji := casteEmoji(caste)
		label := casteLabel(caste)

		if !strings.Contains(identity, emoji) {
			t.Errorf("casteIdentity(%q): expected emoji %q, got %q", caste, emoji, identity)
		}
		if !strings.Contains(identity, label) {
			t.Errorf("casteIdentity(%q): expected label %q, got %q", caste, label, identity)
		}
	}
}

func TestBuildWaveProgressShowsCasteIdentity(t *testing.T) {
	t.Setenv("AETHER_FORCE_COLOR", "1")
	defer os.Unsetenv("AETHER_FORCE_COLOR")

	var buf bytes.Buffer
	stdout = &buf
	t.Setenv("AETHER_OUTPUT_MODE", "visual")

	dispatches := []codex.WorkerDispatch{
		{WorkerName: "Forge-41", Caste: "builder", TaskBrief: "Implement feature X"},
		{WorkerName: "Keen-42", Caste: "watcher", TaskBrief: "Verify tests pass"},
	}

	emitCodexBuildWaveProgress(colony.Phase{ID: 1, Name: "Test Phase"}, 1, dispatches, colony.ModeInRepo)

	output := buf.String()
	for _, want := range []string{"🔨", "Builder", "Forge-41", "👁️", "Watcher", "Keen-42"} {
		if !strings.Contains(output, want) {
			t.Errorf("wave progress missing %q\n%s", want, output)
		}
	}
}

func TestBuildWorkerStartedShowsCasteIdentity(t *testing.T) {
	t.Setenv("AETHER_FORCE_COLOR", "1")
	defer os.Unsetenv("AETHER_FORCE_COLOR")

	var buf bytes.Buffer
	stdout = &buf
	t.Setenv("AETHER_OUTPUT_MODE", "visual")

	dispatch := codex.WorkerDispatch{WorkerName: "Forge-41", Caste: "builder", TaskBrief: "Implement feature X"}
	emitCodexBuildWorkerStarted(dispatch, 1)

	output := buf.String()
	for _, want := range []string{"🔨", "Builder", "Forge-41"} {
		if !strings.Contains(output, want) {
			t.Errorf("worker started missing %q\n%s", want, output)
		}
	}
}

func TestBuildWorkerFinishedShowsCasteIdentity(t *testing.T) {
	t.Setenv("AETHER_FORCE_COLOR", "1")
	defer os.Unsetenv("AETHER_FORCE_COLOR")

	var buf bytes.Buffer
	stdout = &buf
	t.Setenv("AETHER_OUTPUT_MODE", "visual")

	dispatch := codex.WorkerDispatch{WorkerName: "Forge-41", Caste: "builder", TaskBrief: "Implement feature X"}
	result := codex.DispatchResult{WorkerName: "Forge-41", Status: "completed", WorkerResult: &codex.WorkerResult{Duration: 5 * time.Second, Summary: "Done"}}
	emitCodexBuildWorkerFinished(dispatch, result)

	output := buf.String()
	for _, want := range []string{"🔨", "Builder", "Forge-41", "completed"} {
		if !strings.Contains(output, want) {
			t.Errorf("worker finished missing %q\n%s", want, output)
		}
	}
}

func TestInstallJsonModeStillProducesJson(t *testing.T) {
	// Manages its own hub via --home-dir; opt out of suite-wide hub isolation.
	t.Setenv("AETHER_HUB_DIR", "")
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

// --- Medic Caste Tests (Phase 25) ---

func TestCasteIdentityMedic(t *testing.T) {
	os.Setenv("AETHER_FORCE_COLOR", "1")
	defer os.Unsetenv("AETHER_FORCE_COLOR")

	const medicEmoji = "🩹"

	identity := casteIdentity("medic")
	if !strings.Contains(identity, medicEmoji) {
		t.Errorf("casteIdentity(medic): expected 🩹 emoji, got %q", identity)
	}
	if !strings.Contains(identity, "Medic") {
		t.Errorf("casteIdentity(medic): expected 'Medic' label, got %q", identity)
	}
	emoji := casteEmoji("medic")
	if emoji != medicEmoji {
		t.Errorf("casteEmoji(medic): expected 🩹, got %q", emoji)
	}
	label := casteLabel("medic")
	if label != "Medic" {
		t.Errorf("casteLabel(medic): expected 'Medic', got %q", label)
	}
	color := casteANSIColor("medic")
	if color == "" {
		t.Errorf("casteANSIColor(medic): expected a color code, got empty")
	}
}

// --- Porter Caste Tests (Phase 61) ---

func TestCasteIdentityPorter(t *testing.T) {
	os.Setenv("AETHER_FORCE_COLOR", "1")
	defer os.Unsetenv("AETHER_FORCE_COLOR")

	const porterEmoji = "📦"
	const porterColor = "96"
	const porterLabel = "Porter"

	identity := casteIdentity("porter")
	if !strings.Contains(identity, porterEmoji) {
		t.Errorf("casteIdentity(porter): expected 📦 emoji, got %q", identity)
	}
	if !strings.Contains(identity, porterLabel) {
		t.Errorf("casteIdentity(porter): expected 'Porter' label, got %q", identity)
	}
	emoji := casteEmoji("porter")
	if emoji != porterEmoji {
		t.Errorf("casteEmoji(porter): expected 📦, got %q", emoji)
	}
	label := casteLabel("porter")
	if label != porterLabel {
		t.Errorf("casteLabel(porter): expected 'Porter', got %q", label)
	}
	color := casteANSIColor("porter")
	if color != porterColor {
		t.Errorf("casteANSIColor(porter): expected %q, got %q", porterColor, color)
	}
}

// --- Fixer Caste Tests (Phase 89) ---

func TestCasteIdentityFixer(t *testing.T) {
	os.Setenv("AETHER_FORCE_COLOR", "1")
	defer os.Unsetenv("AETHER_FORCE_COLOR")

	const fixerEmoji = "🔧"
	const fixerColor = "33"
	const fixerLabel = "Fixer"

	identity := casteIdentity("fixer")
	if !strings.Contains(identity, fixerEmoji) {
		t.Errorf("casteIdentity(fixer): expected wrench emoji, got %q", identity)
	}
	if !strings.Contains(identity, fixerLabel) {
		t.Errorf("casteIdentity(fixer): expected 'Fixer' label, got %q", identity)
	}
	emoji := casteEmoji("fixer")
	if emoji != fixerEmoji {
		t.Errorf("casteEmoji(fixer): expected 🔧, got %q", emoji)
	}
	label := casteLabel("fixer")
	if label != fixerLabel {
		t.Errorf("casteLabel(fixer): expected 'Fixer', got %q", label)
	}
	color := casteANSIColor("fixer")
	if color != fixerColor {
		t.Errorf("casteANSIColor(fixer): expected %q, got %q", fixerColor, color)
	}
}

func TestGatekeeperEmojiCrossedSwords(t *testing.T) {
	emoji := casteEmoji("gatekeeper")
	if emoji != "⚔️" {
		t.Errorf("casteEmoji(gatekeeper): expected ⚔️ (crossed swords), got %q", emoji)
	}
}

func TestPorterPrefixes(t *testing.T) {
	prefixes, ok := castePrefixes["porter"]
	if !ok {
		t.Fatalf("castePrefixes missing 'porter' key")
	}
	if len(prefixes) != 8 {
		t.Errorf("expected 8 porter prefixes, got %d", len(prefixes))
	}
	for _, p := range prefixes {
		if p == "" {
			t.Errorf("porter prefix is empty")
		}
	}
}

// --- Emoji Consistency Tests (Phase 20) ---

func TestCasteEmojiMapCompleteness(t *testing.T) {
	for key := range casteLabelMap {
		if _, ok := casteEmojiMap[key]; !ok {
			t.Errorf("casteEmojiMap missing key %q (present in casteLabelMap)", key)
		}
		if _, ok := casteColorMap[key]; !ok {
			t.Errorf("casteColorMap missing key %q (present in casteLabelMap)", key)
		}
	}
	for key := range casteEmojiMap {
		if _, ok := casteLabelMap[key]; !ok {
			t.Errorf("casteEmojiMap has extra key %q (not in casteLabelMap)", key)
		}
	}
	for key := range casteColorMap {
		if _, ok := casteLabelMap[key]; !ok {
			t.Errorf("casteColorMap has extra key %q (not in casteLabelMap)", key)
		}
	}
	for key, emoji := range casteEmojiMap {
		if emoji == "" {
			t.Errorf("casteEmojiMap[%q] is empty", key)
		}
	}
	for key, label := range casteLabelMap {
		if label == "" {
			t.Errorf("casteLabelMap[%q] is empty", key)
		}
	}
}

func TestCommandEmojiMapNoEmptyValues(t *testing.T) {
	for key, emoji := range commandEmojiMap {
		if emoji == "" {
			t.Errorf("commandEmojiMap[%q] is empty", key)
		}
	}
}

func TestCommandEmojiFallback(t *testing.T) {
	got := commandEmoji("nonexistent-command-xyz")
	if got != "🐜" {
		t.Errorf("commandEmoji fallback: got %q, want 🐜", got)
	}
}

func TestWrapperDescriptionEmojiConsistency(t *testing.T) {
	repoRoot, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("repo root: %v", err)
	}

	dirs := []string{
		filepath.Join(repoRoot, ".claude", "commands", "ant"),
		filepath.Join(repoRoot, ".opencode", "commands", "ant"),
	}

	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatalf("read dir %s: %v", dir, err)
		}
		for _, entry := range entries {
			if !strings.HasSuffix(entry.Name(), ".md") {
				continue
			}
			fp := filepath.Join(dir, entry.Name())
			content, err := os.ReadFile(fp)
			if err != nil {
				t.Fatalf("read %s: %v", fp, err)
			}
			text := string(content)

			for _, line := range strings.Split(text, "\n") {
				if !strings.HasPrefix(line, "description:") {
					continue
				}
				// Skip help.md — its command emoji IS 🐜
				if strings.Contains(entry.Name(), "help") {
					continue
				}
				// Check that 🐜 does not appear in a sandwich pattern
				// Sandwich = content between two emoji around 🐜
				if strings.Contains(line, "🐜") {
					t.Errorf("%s: description still contains 🐜 sandwich: %s", fp, line)
				}
				break
			}
		}
	}
}

// --- Codex Visual Parity Tests (Phase 21) ---

func TestCodexVisualParity(t *testing.T) {
	t.Run("CasteIdentity", func(t *testing.T) {
		// Codex uses the same casteIdentity() function — verify it renders correctly
		for _, caste := range []string{"builder", "watcher", "scout", "queen"} {
			identity := casteIdentity(caste)
			emoji := casteEmoji(caste)
			label := casteLabel(caste)
			if !strings.Contains(identity, emoji) {
				t.Errorf("casteIdentity(%q) missing emoji %q: %s", caste, emoji, identity)
			}
			if !strings.Contains(identity, label) {
				t.Errorf("casteIdentity(%q) missing label %q: %s", caste, label, identity)
			}
		}
	})

	t.Run("StageSeparators", func(t *testing.T) {
		saveGlobals(t)
		resetRootCmd(t)
		goal := "Codex parity check"
		taskID := "task-parity"
		accepted := createApprovedAcceptedBuildTestColony(t, colony.ColonyState{
			Version: "3.0",
			Goal:    &goal,
			State:   colony.StateREADY,
			Plan: colony.Plan{
				Phases: []colony.Phase{
					{ID: 1, Name: "Parity", Status: colony.PhaseReady, Tasks: []colony.Task{{ID: &taskID, Goal: "Check parity", Status: colony.TaskPending}}},
				},
			},
		})
		withWorkingDir(t, accepted.Root)
		t.Setenv("AETHER_OUTPUT_MODE", "visual")

		rootCmd.SetArgs([]string{"build", "1"})
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("build error: %v", err)
		}

		output := stdout.(*bytes.Buffer).String()
		for _, marker := range []string{"── Context ──", "── Tasks ──", "── Dispatch ──", "── Verification [", "── Housekeeping ──", "── Colony Complete ──"} {
			if !strings.Contains(output, marker) {
				t.Errorf("build visual missing stage marker %q (Codex parity)", marker)
			}
		}
	})

	t.Run("CommandEmojiParity", func(t *testing.T) {
		// Verify commandEmoji returns expected values for key commands
		parity := map[string]string{
			"build":    "🔨",
			"continue": "👁️",
			"init":     "🥚",
			"plan":     "📋",
			"seal":     "🏺",
			"status":   "📊",
			"patrol":   "📊",
			"history":  "📜",
			"pause":    "💾",
			"resume":   "💾",
		}
		for cmd, wantEmoji := range parity {
			got := commandEmoji(cmd)
			if got != wantEmoji {
				t.Errorf("commandEmoji(%q) = %q, want %q", cmd, got, wantEmoji)
			}
		}
	})

	t.Run("SpawnListParity", func(t *testing.T) {
		saveGlobals(t)
		resetRootCmd(t)
		goal := "Spawn parity"
		taskID := "task-spawn"
		accepted := createApprovedAcceptedBuildTestColony(t, colony.ColonyState{
			Version: "3.0",
			Goal:    &goal,
			State:   colony.StateREADY,
			Plan: colony.Plan{
				Phases: []colony.Phase{
					{ID: 1, Name: "Spawn", Status: colony.PhaseReady, Tasks: []colony.Task{{ID: &taskID, Goal: "Spawn check", Status: colony.TaskPending}}},
				},
			},
		})
		withWorkingDir(t, accepted.Root)
		t.Setenv("AETHER_OUTPUT_MODE", "visual")

		rootCmd.SetArgs([]string{"build", "1"})
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("build error: %v", err)
		}

		output := stdout.(*bytes.Buffer).String()
		if !strings.Contains(output, "S P A W N   P L A N") {
			t.Error("build visual missing SPAWN PLAN header (Codex parity)")
		}
		if !strings.Contains(output, "Builder") {
			t.Error("build visual missing Builder in spawn list (Codex parity)")
		}
	})

	t.Run("VisualModeRouting", func(t *testing.T) {
		// Simulate Codex invocation: AETHER_OUTPUT_MODE=visual
		t.Setenv("AETHER_OUTPUT_MODE", "visual")
		if !shouldRenderVisualOutput(stdout) {
			t.Error("shouldRenderVisualOutput() false with AETHER_OUTPUT_MODE=visual (Codex parity)")
		}

		// JSON mode should not render visual
		t.Setenv("AETHER_OUTPUT_MODE", "json")
		if shouldRenderVisualOutput(stdout) {
			t.Error("shouldRenderVisualOutput() true with AETHER_OUTPUT_MODE=json")
		}
	})

	t.Run("ContextClearParity", func(t *testing.T) {
		// The safe-to-clear claim is verified against a real handoff file,
		// so this subtest supplies one — it must not depend on whichever
		// store an earlier test happened to leave behind.
		saveGlobals(t)
		s, tmpDir := newTestStore(t)
		defer os.RemoveAll(tmpDir)
		store = s
		handoffPath := filepath.Join(resolveAetherRootPath(), ".aether", "HANDOFF.md")
		if err := os.MkdirAll(filepath.Dir(handoffPath), 0755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(handoffPath, []byte("# handoff\n"), 0644); err != nil {
			t.Fatalf("write handoff: %v", err)
		}
		guidance := renderContextClearGuidance()
		if !strings.Contains(guidance, "safe to clear") {
			t.Errorf("renderContextClearGuidance() missing clear guidance: %s", guidance)
		}
	})
}

// --- Visuals Config File Loading Tests (Phase 155-01) ---

func TestVisualsFileLoad(t *testing.T) {
	tmpDir := t.TempDir()
	visualsPath := filepath.Join(tmpDir, "visuals.md")
	content := `---
visuals_version: "test"
caste_emoji_map:
  builder: "TEST_BUILDER_EMOJI"
  watcher: "TEST_WATCHER_EMOJI"
---
`
	if err := os.WriteFile(visualsPath, []byte(content), 0644); err != nil {
		t.Fatalf("write visuals.md: %v", err)
	}

	resetVisualsCache()
	defer resetVisualsCache()
	visualsPathOverride = visualsPath
	defer func() { visualsPathOverride = "" }()

	got := casteEmoji("builder")
	if got != "TEST_BUILDER_EMOJI" {
		t.Errorf("casteEmoji(builder) with file: got %q, want TEST_BUILDER_EMOJI", got)
	}
}

func TestVisualsFallback(t *testing.T) {
	tmpDir := t.TempDir()
	resetVisualsCache()
	defer resetVisualsCache()
	visualsPathOverride = filepath.Join(tmpDir, "nonexistent-visuals.md")
	defer func() { visualsPathOverride = "" }()

	got := casteEmoji("builder")
	if got != "🔨" {
		t.Errorf("casteEmoji(builder) fallback: got %q, want 🔨", got)
	}
}

func TestVisualsPartialFallback(t *testing.T) {
	tmpDir := t.TempDir()
	visualsPath := filepath.Join(tmpDir, "visuals.md")
	content := `---
visuals_version: "test"
caste_emoji_map:
  builder: "CUSTOM_BUILDER"
---
`
	if err := os.WriteFile(visualsPath, []byte(content), 0644); err != nil {
		t.Fatalf("write visuals.md: %v", err)
	}

	resetVisualsCache()
	defer resetVisualsCache()
	visualsPathOverride = visualsPath
	defer func() { visualsPathOverride = "" }()

	// builder is in file
	if got := casteEmoji("builder"); got != "CUSTOM_BUILDER" {
		t.Errorf("casteEmoji(builder) partial: got %q, want CUSTOM_BUILDER", got)
	}
	// watcher is not in file — should fall back to hardcoded
	if got := casteEmoji("watcher"); got != "👁️" {
		t.Errorf("casteEmoji(watcher) partial fallback: got %q, want 👁️", got)
	}
}

func TestVisualsWordmarkLoad(t *testing.T) {
	tmpDir := t.TempDir()
	visualsPath := filepath.Join(tmpDir, "visuals.md")
	content := `---
visuals_version: "test"
aether_wordmark: |
  CUSTOM_WORDMARK_LINE
---
`
	if err := os.WriteFile(visualsPath, []byte(content), 0644); err != nil {
		t.Fatalf("write visuals.md: %v", err)
	}

	resetVisualsCache()
	defer resetVisualsCache()
	visualsPathOverride = visualsPath
	defer func() { visualsPathOverride = "" }()

	got := renderAetherWordmark()
	if !strings.Contains(got, "CUSTOM_WORDMARK_LINE") {
		t.Errorf("renderAetherWordmark() did not contain custom wordmark: %q", got)
	}
}

func TestVisualsCommandEmoji(t *testing.T) {
	tmpDir := t.TempDir()
	visualsPath := filepath.Join(tmpDir, "visuals.md")
	content := `---
visuals_version: "test"
command_emoji_map:
  init: "CUSTOM_INIT"
---
`
	if err := os.WriteFile(visualsPath, []byte(content), 0644); err != nil {
		t.Fatalf("write visuals.md: %v", err)
	}

	resetVisualsCache()
	defer resetVisualsCache()
	visualsPathOverride = visualsPath
	defer func() { visualsPathOverride = "" }()

	got := commandEmoji("init")
	if got != "CUSTOM_INIT" {
		t.Errorf("commandEmoji(init) with file: got %q, want CUSTOM_INIT", got)
	}
}

func TestVisualsDeterministicName(t *testing.T) {
	tmpDir := t.TempDir()
	visualsPath := filepath.Join(tmpDir, "visuals.md")
	content := `---
visuals_version: "test"
caste_prefixes:
  builder: ["Alpha", "Beta"]
default_prefixes: ["Gamma", "Delta"]
---
`
	if err := os.WriteFile(visualsPath, []byte(content), 0644); err != nil {
		t.Fatalf("write visuals.md: %v", err)
	}

	resetVisualsCache()
	defer resetVisualsCache()
	visualsPathOverride = visualsPath
	defer func() { visualsPathOverride = "" }()

	name := deterministicAntName("builder", "test-seed-1")
	if !strings.HasPrefix(name, "Alpha-") && !strings.HasPrefix(name, "Beta-") {
		t.Errorf("deterministicAntName(builder) did not use file prefixes: got %q", name)
	}

	// unknown caste should use file default_prefixes
	name2 := deterministicAntName("unknown_caste", "test-seed-2")
	if !strings.HasPrefix(name2, "Gamma-") && !strings.HasPrefix(name2, "Delta-") {
		t.Errorf("deterministicAntName(unknown_caste) did not use file default prefixes: got %q", name2)
	}
}

func TestVisualsCasteLabelAndColor(t *testing.T) {
	tmpDir := t.TempDir()
	visualsPath := filepath.Join(tmpDir, "visuals.md")
	content := `---
visuals_version: "test"
caste_label_map:
  builder: "CustomBuilder"
caste_color_map:
  builder: "99"
---
`
	if err := os.WriteFile(visualsPath, []byte(content), 0644); err != nil {
		t.Fatalf("write visuals.md: %v", err)
	}

	resetVisualsCache()
	defer resetVisualsCache()
	visualsPathOverride = visualsPath
	defer func() { visualsPathOverride = "" }()

	if got := casteLabel("builder"); got != "CustomBuilder" {
		t.Errorf("casteLabel(builder) with file: got %q, want CustomBuilder", got)
	}
	if got := casteANSIColor("builder"); got != "99" {
		t.Errorf("casteANSIColor(builder) with file: got %q, want 99", got)
	}
}

func TestVisualsDividerLoad(t *testing.T) {
	tmpDir := t.TempDir()
	visualsPath := filepath.Join(tmpDir, "visuals.md")
	content := `---
visuals_version: "test"
visual_divider: "CUSTOM_DIVIDER\n"
---
`
	if err := os.WriteFile(visualsPath, []byte(content), 0644); err != nil {
		t.Fatalf("write visuals.md: %v", err)
	}

	resetVisualsCache()
	defer resetVisualsCache()
	visualsPathOverride = visualsPath
	defer func() { visualsPathOverride = "" }()

	if got := visualDividerStr(); got != "CUSTOM_DIVIDER\n" {
		t.Errorf("visualDividerStr() with file: got %q, want CUSTOM_DIVIDER\\n", got)
	}
}

// --- Phase 191-03: fold-and-delete regression lock for colony/ceremony/visuals.md ---

// TestVisualsConfigOriginalFileHadPreexistingParseDefect is a forensic
// record, not a live guard: colony/ceremony/visuals.md's YAML frontmatter
// never successfully parsed, in any dev checkout, ever -- independent of,
// and in addition to, the CWD-relative-only defect 191-CONTEXT.md's
// criterion 2 documents for all four loaders. Its aether_wordmark
// block-literal scalar (`aether_wordmark: |`) auto-detected an indentation
// baseline of 6 spaces from its first content line, but the following five
// lines used only 5 -- one less than that baseline, which YAML's
// block-scalar rules treat as ending the block mid-document -- and the
// parser then fails on what follows ("did not find expected key"). Because
// gopkg.in/yaml.v3 parses the whole frontmatter as one document, this one
// defect silently invalidated every field in the file, not only the
// wordmark: loadVisualsConfig() has returned nil for this file
// unconditionally, forever, in every environment. This is why
// TestVisualsConfigFoldedDefaultsMatchOriginalFile below is trivially
// byte-identical -- the file never contributed a single live value to
// compare against -- and it is independent, stronger-than-required proof
// that deleting the file changes nothing observable.
func TestVisualsConfigOriginalFileHadPreexistingParseDefect(t *testing.T) {
	fixture := filepath.Join("testdata", "visuals_config_original.md")

	resetVisualsCache()
	visualsPathOverride = fixture
	loaded := loadVisualsConfig()
	visualsPathOverride = ""
	resetVisualsCache()

	if loaded != nil {
		t.Fatalf("loadVisualsConfig() successfully parsed the frozen original fixture %s -- the historical parse defect this test documents (inconsistent aether_wordmark block-scalar indentation) appears to be gone from the frozen copy; if colony/ceremony/visuals.md's real content ever differed from what 191-03-SUMMARY.md recorded, re-verify that summary's claims and this test's premise", fixture)
	}
}

// TestVisualsConfigFoldedDefaultsMatchOriginalFile is the permanent
// before/after regression lock for Phase 191 criterion 2's third loader,
// loadVisualsConfig(). colony/ceremony/visuals.md was deleted after
// confirming (see TestVisualsConfigOriginalFileHadPreexistingParseDefect
// and 191-03-SUMMARY.md) that the file never successfully parsed in any
// environment, so it never contributed a single live value -- there was
// nothing to fold. cmd/testdata/visuals_config_original.md is a byte-exact
// frozen copy of the deleted file (made with `cp`, verified with `diff`).
// This test renders every field the compiled defaults can produce -- every
// caste's emoji/ANSI color/label/composed identity, every command's emoji,
// every caste's name-prefix list plus the default prefixes, the ASCII
// wordmark, and the divider -- once with the frozen original file loaded
// (which resolves to the same nil-config fallback it always has) and once
// with the file entirely absent (the permanent, post-deletion state), and
// asserts the two renders are byte-identical. The key sets it iterates come
// from the compiled Go maps directly (casteEmojiMap, commandEmojiMap,
// castePrefixes) -- the sole authoritative source now that the file
// contributes nothing live -- so this test cannot silently shrink into a
// sample.
func TestVisualsConfigFoldedDefaultsMatchOriginalFile(t *testing.T) {
	fixture := filepath.Join("testdata", "visuals_config_original.md")

	// identityCastes: every caste key the compiled emoji/color/label maps
	// name -- drives the casteEmoji/casteANSIColor/casteLabel/casteIdentity
	// checks below.
	identityCastes := map[string]bool{}
	for caste := range casteEmojiMap {
		identityCastes[caste] = true
	}
	for caste := range casteColorMap {
		identityCastes[caste] = true
	}
	for caste := range casteLabelMap {
		identityCastes[caste] = true
	}

	// prefixCastes: every identity caste plus every caste with its own
	// name-prefix list (e.g. "prime" has prefixes but no emoji/color/label
	// entry) -- drives the deterministicAntName checks below, which
	// exercise both the CastePrefixes field (castes with their own list) and
	// the DefaultPrefixes field (identity castes with no list of their own).
	prefixCastes := map[string]bool{}
	for caste := range identityCastes {
		prefixCastes[caste] = true
	}
	for caste := range castePrefixes {
		prefixCastes[caste] = true
	}

	seeds := []string{"seed-alpha", "seed-beta", "seed-gamma", "seed-delta", "seed-epsilon"}

	render := func(t *testing.T, label string) map[string]string {
		t.Helper()
		out := make(map[string]string)
		for caste := range identityCastes {
			out["casteEmoji:"+caste] = casteEmoji(caste)
			out["casteANSIColor:"+caste] = casteANSIColor(caste)
			out["casteLabel:"+caste] = casteLabel(caste)
			out["casteIdentity:"+caste] = casteIdentity(caste)
		}
		for caste := range prefixCastes {
			for _, seed := range seeds {
				out["deterministicAntName:"+caste+":"+seed] = deterministicAntName(caste, seed)
			}
		}
		for command := range commandEmojiMap {
			out["commandEmoji:"+command] = commandEmoji(command)
		}
		out["renderAetherWordmark"] = renderAetherWordmark()
		out["visualDividerStr"] = visualDividerStr()
		if len(out) == 0 {
			t.Fatalf("%s render produced zero entries -- test is broken, not passing for the right reason", label)
		}
		return out
	}

	resetVisualsCache()
	visualsPathOverride = fixture
	before := render(t, "original colony/ceremony/visuals.md (frozen fixture)")
	visualsPathOverride = ""
	resetVisualsCache()

	visualsPathOverride = filepath.Join(t.TempDir(), "visuals-deleted-does-not-exist.md")
	after := render(t, "compiled defaults with the file absent")
	visualsPathOverride = ""
	resetVisualsCache()

	if len(before) != len(after) {
		t.Fatalf("render key-set size changed between renders: before=%d after=%d -- test itself is unstable", len(before), len(after))
	}
	mismatches := 0
	for key, wantVal := range before {
		if gotVal := after[key]; gotVal != wantVal {
			mismatches++
			t.Errorf("%s: with colony/ceremony/visuals.md present = %q, with it absent = %q -- deletion changed observable output", key, wantVal, gotVal)
		}
	}
	if mismatches > 0 {
		t.Fatalf("%d of %d rendered field(s) changed after deleting colony/ceremony/visuals.md -- see failures above", mismatches, len(before))
	}
}

// TestVisualsConfigOriginalFileValuesMatchCompiledDefaults is supporting
// evidence for the "nothing to fold" conclusion, independent of the parse
// defect documented above. cmd/testdata/visuals_config_original_parseable.md
// is the frozen original with a single line changed -- `aether_wordmark: |`
// became `aether_wordmark: |5`, an explicit YAML block-indentation
// indicator that supplies the baseline the parser could not auto-detect
// (see TestVisualsConfigOriginalFileHadPreexistingParseDefect) -- not one
// byte of any content line, including the wordmark's own art, was touched
// (verified with `diff` when this fixture was made; see 191-03-SUMMARY.md).
// With that single line fixed, the file parses, and this test proves its
// written values for every discrete field -- caste_emoji_map,
// caste_color_map, caste_label_map, command_emoji_map, caste_prefixes,
// default_prefixes, and visual_divider -- match cmd/codex_visuals.go's
// compiled defaults exactly, for every key the file names, not a sample.
//
// aether_wordmark is deliberately excluded from the byte-equality
// assertions above and checked separately, glyph content only: a `|`
// block-literal's indentation baseline is auto-detected from its first
// content line, so the minimal one-line fix that makes the document valid
// changes what absolute left-margin the parsed value carries (it cannot
// also recover what margin the file's author intended) -- only that the
// same characters, in the same order, are present is provable. The compiled
// default (already the only value ever actually rendered, per the parse
// defect finding above) is kept exactly as-is; there is no live value to
// fold it against.
func TestVisualsConfigOriginalFileValuesMatchCompiledDefaults(t *testing.T) {
	fixture := filepath.Join("testdata", "visuals_config_original_parseable.md")

	resetVisualsCache()
	visualsPathOverride = fixture
	parsed := loadVisualsConfig()
	visualsPathOverride = ""
	resetVisualsCache()
	if parsed == nil {
		t.Fatalf("loadVisualsConfig() returned nil for the syntax-fixed fixture %s -- it should parse cleanly; the single-line `|5` fix may have been lost", fixture)
	}
	if len(parsed.CasteEmojiMap) == 0 || len(parsed.CasteColorMap) == 0 || len(parsed.CasteLabelMap) == 0 ||
		len(parsed.CommandEmojiMap) == 0 || len(parsed.CastePrefixes) == 0 || len(parsed.DefaultPrefixes) == 0 ||
		parsed.VisualDivider == "" || parsed.AetherWordmark == "" {
		t.Fatalf("syntax-fixed fixture parsed with an unexpectedly empty field -- this test would pass vacuously instead of proving anything: %+v", parsed)
	}

	for caste, fileEmoji := range parsed.CasteEmojiMap {
		if compiled, ok := casteEmojiMap[caste]; ok && fileEmoji != compiled {
			t.Errorf("caste_emoji_map[%s]: file=%q compiled=%q", caste, fileEmoji, compiled)
		}
	}
	for caste, fileColor := range parsed.CasteColorMap {
		if compiled, ok := casteColorMap[caste]; ok && fileColor != compiled {
			t.Errorf("caste_color_map[%s]: file=%q compiled=%q", caste, fileColor, compiled)
		}
	}
	for caste, fileLabel := range parsed.CasteLabelMap {
		if compiled, ok := casteLabelMap[caste]; ok && fileLabel != compiled {
			t.Errorf("caste_label_map[%s]: file=%q compiled=%q", caste, fileLabel, compiled)
		}
	}
	for command, fileEmoji := range parsed.CommandEmojiMap {
		if compiled, ok := commandEmojiMap[command]; ok && fileEmoji != compiled {
			t.Errorf("command_emoji_map[%s]: file=%q compiled=%q", command, fileEmoji, compiled)
		}
	}
	for caste, filePrefixes := range parsed.CastePrefixes {
		compiled, ok := castePrefixes[caste]
		if !ok {
			continue
		}
		if len(filePrefixes) != len(compiled) {
			t.Errorf("caste_prefixes[%s]: file has %d entries, compiled has %d: file=%v compiled=%v", caste, len(filePrefixes), len(compiled), filePrefixes, compiled)
			continue
		}
		for i := range filePrefixes {
			if filePrefixes[i] != compiled[i] {
				t.Errorf("caste_prefixes[%s][%d]: file=%q compiled=%q", caste, i, filePrefixes[i], compiled[i])
			}
		}
	}
	if len(parsed.DefaultPrefixes) != len(defaultPrefixes) {
		t.Errorf("default_prefixes: file has %d entries, compiled has %d: file=%v compiled=%v", len(parsed.DefaultPrefixes), len(defaultPrefixes), parsed.DefaultPrefixes, defaultPrefixes)
	} else {
		for i := range parsed.DefaultPrefixes {
			if parsed.DefaultPrefixes[i] != defaultPrefixes[i] {
				t.Errorf("default_prefixes[%d]: file=%q compiled=%q", i, parsed.DefaultPrefixes[i], defaultPrefixes[i])
			}
		}
	}
	if parsed.VisualDivider != visualDividerFallback {
		t.Errorf("visual_divider: file=%q compiled=%q", parsed.VisualDivider, visualDividerFallback)
	}

	// Glyph-content-only comparison for the wordmark: strip each line's
	// leading spaces from both sides (the file's parsed value already had
	// the `|5` baseline stripped by the YAML parser; the compiled Go string
	// literal has not been stripped of anything) and compare what remains,
	// line by line. See the doc comment above for why absolute margin is
	// not asserted.
	fileLines := strings.Split(strings.Trim(parsed.AetherWordmark, "\n"), "\n")
	compiledLines := strings.Split(strings.Trim(aetherWordmark, "\n"), "\n")
	if len(fileLines) != len(compiledLines) {
		t.Fatalf("aether_wordmark: file has %d lines, compiled has %d -- glyph content cannot match", len(fileLines), len(compiledLines))
	}
	for i := range fileLines {
		fileGlyphs := strings.TrimLeft(fileLines[i], " ")
		compiledGlyphs := strings.TrimLeft(compiledLines[i], " ")
		if fileGlyphs != compiledGlyphs {
			t.Errorf("aether_wordmark line %d glyph content: file=%q compiled=%q", i, fileGlyphs, compiledGlyphs)
		}
	}
}

func TestCodexVisualsPlanningRoutesCanonicalCandidate(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	result := planCandidateReviewResult(planningVisualCandidateFixture())
	output := renderPlanVisual(result)

	for _, want := range []string{"Plan Candidate", "CANDIDATE — NOT ACTIVE", "Queen recommendation", "Accept this candidate?"} {
		if !strings.Contains(output, want) {
			t.Fatalf("canonical candidate route missing %q:\n%s", want, output)
		}
	}
	for _, forbidden := range []string{"Plan size: 0 phases", "Scout and Route-Setter mapped", "aether build", "aether run"} {
		if strings.Contains(output, forbidden) {
			t.Fatalf("canonical candidate route leaked legacy/unauthorized text %q:\n%s", forbidden, output)
		}
	}
}

func TestCodexVisualsPlanningRoutesCanonicalIteration(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	result := map[string]interface{}{
		"planned": false, "status": string(planningStageContinueReady),
		"iteration_card": planningVisualIterationFixture(colony.PlanningStopContinue),
	}
	output := renderPlanVisual(result)
	for _, want := range []string{"Planning Iteration", "Scout", "Route-Setter", "Fresh evidence", "CONTINUE"} {
		if !strings.Contains(output, want) {
			t.Fatalf("canonical iteration route missing %q:\n%s", want, output)
		}
	}
}

func TestCodexVisualsSpecIdentityContract(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	if got := commandEmoji("spec"); got != "📜" {
		t.Fatalf("commandEmoji(spec) = %q, want 📜", got)
	}
	output := renderSpecCommandVisual(planningVisualSpecificationFixture())
	if !strings.Contains(output, "📜 "+spacedTitle("Specification")) {
		t.Fatalf("specification visual missing full command identity:\n%s", output)
	}
	if strings.Contains(output, "Command: Spec\n") || strings.Contains(output, "Identity: Spec\n") {
		t.Fatalf("specification visual used forbidden full-label abbreviation:\n%s", output)
	}
}
