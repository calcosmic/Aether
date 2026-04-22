package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
)

func TestColonizeWritesSurveyArtifactsAndUpdatesState(t *testing.T) {
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

	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/aether-test\n\ngo 1.24\n\nrequire github.com/spf13/cobra v1.9.0\n"), 0644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("# Test workspace\n"), 0644); err != nil {
		t.Fatalf("failed to write README: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "cmd"), 0755); err != nil {
		t.Fatalf("failed to create cmd dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "pkg"), 0755); err != nil {
		t.Fatalf("failed to create pkg dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "cmd", "main.go"), []byte("package main\n\nfunc main() {}\n"), 0644); err != nil {
		t.Fatalf("failed to write main.go: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "cmd", "main_test.go"), []byte("package main\n\nimport \"testing\"\n\nfunc TestMain(t *testing.T) {}\n"), 0644); err != nil {
		t.Fatalf("failed to write main_test.go: %v", err)
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

	var envelope map[string]interface{}
	if err := json.Unmarshal(stdout.(*bytes.Buffer).Bytes(), &envelope); err != nil {
		t.Fatalf("failed to parse colonize output: %v\n%s", err, stdout.(*bytes.Buffer).String())
	}
	if envelope["ok"] != true {
		t.Fatalf("expected ok:true, got %v", envelope)
	}

	result := envelope["result"].(map[string]interface{})
	surveyFiles := result["survey_files"].([]interface{})
	if len(surveyFiles) != 7 {
		t.Fatalf("expected 7 survey files, got %d", len(surveyFiles))
	}
	surveyors := result["surveyors"].([]interface{})
	if len(surveyors) != 4 {
		t.Fatalf("expected 4 surveyors, got %d", len(surveyors))
	}

	for _, name := range []string{"PROVISIONS.md", "TRAILS.md", "BLUEPRINT.md", "CHAMBERS.md", "DISCIPLINES.md", "SENTINEL-PROTOCOLS.md", "PATHOGENS.md"} {
		if _, err := os.Stat(filepath.Join(dataDir, "survey", name)); err != nil {
			t.Fatalf("expected survey file %s: %v", name, err)
		}
	}
	for _, name := range []string{"blueprint.json", "chambers.json", "disciplines.json", "provisions.json", "pathogens.json"} {
		if _, err := os.Stat(filepath.Join(dataDir, "survey", name)); err != nil {
			t.Fatalf("expected compatibility file %s: %v", name, err)
		}
	}

	spawnTreeData, err := os.ReadFile(filepath.Join(dataDir, "spawn-tree.txt"))
	if err != nil {
		t.Fatalf("expected spawn-tree.txt: %v", err)
	}
	if count := strings.Count(string(spawnTreeData), "|Queen|surveyor|"); count != 4 {
		t.Fatalf("expected 4 surveyor spawn entries, got %d\n%s", count, string(spawnTreeData))
	}

	var state colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
		t.Fatalf("failed to reload colony state: %v", err)
	}
	if state.TerritorySurveyed == nil || *state.TerritorySurveyed == "" {
		t.Fatal("expected TerritorySurveyed to be set")
	}
	if len(state.Events) == 0 || !strings.Contains(state.Events[len(state.Events)-1], "territory_surveyed|colonize") {
		t.Fatalf("expected territory_surveyed event, got %v", state.Events)
	}
}

func TestColonizeRequiresForceResurveyWhenSurveyExists(t *testing.T) {
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

	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/aether-test\n\ngo 1.24\n"), 0644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(dataDir, "survey"), 0755); err != nil {
		t.Fatalf("failed to create survey dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dataDir, "survey", "PROVISIONS.md"), []byte("# old survey\n"), 0644); err != nil {
		t.Fatalf("failed to write old survey: %v", err)
	}

	goal := "Survey the repo"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version: "3.0",
		Goal:    &goal,
		State:   colony.StateREADY,
		Plan:    colony.Plan{Phases: []colony.Phase{}},
	})

	var errBuf bytes.Buffer
	stderr = &errBuf

	rootCmd.SetArgs([]string{"colonize"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("colonize returned error: %v", err)
	}

	if !strings.Contains(errBuf.String(), "existing territory survey found") {
		t.Fatalf("expected force-resurvey guidance, got: %s", errBuf.String())
	}
}

func TestColonizePreservesWorkerWrittenSurveyArtifacts(t *testing.T) {
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

	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/aether-test\n\ngo 1.24\n"), 0644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	goal := "Survey the repo"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version: "3.0",
		Goal:    &goal,
		State:   colony.StateREADY,
		Plan:    colony.Plan{Phases: []colony.Phase{}},
	})

	originalInvoker := newCodexWorkerInvoker
	newCodexWorkerInvoker = func() codex.WorkerInvoker { return &surveyorArtifactInvoker{} }
	defer func() { newCodexWorkerInvoker = originalInvoker }()

	rootCmd.SetArgs([]string{"colonize"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("colonize returned error: %v", err)
	}

	env := parseEnvelope(t, stdout.(*bytes.Buffer).String())
	result := env["result"].(map[string]interface{})
	if got := result["dispatch_mode"]; got != "real" {
		t.Fatalf("dispatch_mode = %v, want real", got)
	}
	if got := result["artifact_source"]; got != "worker-written" {
		t.Fatalf("artifact_source = %v, want worker-written", got)
	}

	data, err := os.ReadFile(filepath.Join(dataDir, "survey", "PROVISIONS.md"))
	if err != nil {
		t.Fatalf("read PROVISIONS.md: %v", err)
	}
	if !strings.Contains(string(data), "worker-authored provisions") {
		t.Fatalf("expected worker-authored PROVISIONS.md to be preserved, got:\n%s", string(data))
	}
}

func TestColonizeFallsBackWhenRealSurveyorsFail(t *testing.T) {
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

	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/aether-test\n\ngo 1.24\n"), 0644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("# Test workspace\n"), 0644); err != nil {
		t.Fatalf("failed to write README: %v", err)
	}

	goal := "Survey fallback"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version: "3.0",
		Goal:    &goal,
		State:   colony.StateREADY,
		Plan:    colony.Plan{Phases: []colony.Phase{}},
	})

	originalInvoker := newCodexWorkerInvoker
	newCodexWorkerInvoker = func() codex.WorkerInvoker { return &timeoutTestInvoker{} }
	defer func() { newCodexWorkerInvoker = originalInvoker }()

	rootCmd.SetArgs([]string{"colonize"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("colonize returned error: %v", err)
	}

	env := parseEnvelope(t, stdout.(*bytes.Buffer).String())
	result := env["result"].(map[string]interface{})
	if got := result["dispatch_mode"]; got != "fallback" {
		t.Fatalf("dispatch_mode = %v, want fallback", got)
	}
	if got := strings.TrimSpace(result["survey_warning"].(string)); got == "" {
		t.Fatal("expected survey_warning to be populated")
	}
	surveyors := result["surveyors"].([]interface{})
	if len(surveyors) != 4 {
		t.Fatalf("expected 4 surveyors, got %d", len(surveyors))
	}
	first := surveyors[0].(map[string]interface{})
	if first["status"] != "timeout" {
		t.Fatalf("first surveyor status = %v, want timeout", first["status"])
	}
	if files := result["survey_files"].([]interface{}); len(files) == 0 {
		t.Fatal("expected fallback colonize to still write survey files")
	}
}

// unavailableInvoker always reports not available, forcing fallback.
type unavailableInvoker struct{}

func (u *unavailableInvoker) Invoke(_ context.Context, _ codex.WorkerConfig) (codex.WorkerResult, error) {
	return codex.WorkerResult{}, nil
}

func (u *unavailableInvoker) IsAvailable(_ context.Context) bool {
	return false
}

func (u *unavailableInvoker) ValidateAgent(_ string) error {
	return nil
}

// TestDispatchRealSurveyors_FakeInvokerFallback verifies that when the
// worker invoker is not available, dispatchRealSurveyors falls back to
// the same results as plannedSurveyors (same castes, names, tasks, outputs).
func TestDispatchRealSurveyors_FakeInvokerFallback(t *testing.T) {
	tmpDir := t.TempDir()
	root := tmpDir

	expected := plannedSurveyors(root)

	got, err := dispatchRealSurveyors(context.Background(), root, &unavailableInvoker{})
	if err != nil {
		t.Fatalf("dispatchRealSurveyors returned error on unavailable invoker: %v", err)
	}

	if len(got) != len(expected) {
		t.Fatalf("expected %d dispatches, got %d", len(expected), len(got))
	}

	for i, exp := range expected {
		if got[i].Caste != exp.Caste {
			t.Errorf("dispatch[%d].Caste = %q, want %q", i, got[i].Caste, exp.Caste)
		}
		if got[i].Name != exp.Name {
			t.Errorf("dispatch[%d].Name = %q, want %q", i, got[i].Name, exp.Name)
		}
		if got[i].Task != exp.Task {
			t.Errorf("dispatch[%d].Task = %q, want %q", i, got[i].Task, exp.Task)
		}
		if len(got[i].Outputs) != len(exp.Outputs) {
			t.Errorf("dispatch[%d].Outputs length = %d, want %d", i, len(got[i].Outputs), len(exp.Outputs))
		}
	}
}

// TestDispatchRealSurveyors_ConvertsResults verifies that DispatchResult
// values are correctly mapped to codexSurveyorDispatch structs.
func TestDispatchRealSurveyors_ConvertsResults(t *testing.T) {
	tmpDir := t.TempDir()
	root := tmpDir

	// Create minimal surveyor TOML files so AssemblePrompt can read them
	codexDir := filepath.Join(root, ".codex", "agents")
	if err := os.MkdirAll(codexDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	tomlContent := `name = "test"
description = "test agent"
developer_instructions = "You are a test surveyor."
`
	for _, name := range []string{
		"aether-surveyor-nest.toml",
		"aether-surveyor-disciplines.toml",
		"aether-surveyor-pathogens.toml",
		"aether-surveyor-provisions.toml",
	} {
		if err := os.WriteFile(filepath.Join(codexDir, name), []byte(tomlContent), 0644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}

	// Use FakeInvoker which always reports available and returns completed
	invoker := &codex.FakeInvoker{}

	got, err := dispatchRealSurveyors(context.Background(), root, invoker)
	if err != nil {
		t.Fatalf("dispatchRealSurveyors returned error: %v", err)
	}

	if len(got) != 4 {
		t.Fatalf("expected 4 dispatches, got %d", len(got))
	}

	for i, d := range got {
		if d.Status != "completed" {
			t.Errorf("dispatch[%d].Status = %q, want %q", i, d.Status, "completed")
		}
		if d.Caste == "" {
			t.Errorf("dispatch[%d].Caste is empty", i)
		}
		if d.Name == "" {
			t.Errorf("dispatch[%d].Name is empty", i)
		}
	}
}

// TestDispatchRealSurveyors_TimeoutMapsToFailed verifies that "timeout"
// status from DispatchResult is mapped to "failed" in codexSurveyorDispatch.
func TestDispatchRealSurveyors_TimeoutMapsToFailed(t *testing.T) {
	tmpDir := t.TempDir()
	root := tmpDir

	// Create minimal surveyor TOML files
	codexDir := filepath.Join(root, ".codex", "agents")
	if err := os.MkdirAll(codexDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	tomlContent := `name = "test"
description = "test agent"
developer_instructions = "You are a test surveyor."
`
	for _, name := range []string{
		"aether-surveyor-nest.toml",
		"aether-surveyor-disciplines.toml",
		"aether-surveyor-pathogens.toml",
		"aether-surveyor-provisions.toml",
	} {
		if err := os.WriteFile(filepath.Join(codexDir, name), []byte(tomlContent), 0644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}

	// timeoutInvoker returns timeout status for all workers
	timeoutInvoker := &timeoutTestInvoker{}

	got, err := dispatchRealSurveyors(context.Background(), root, timeoutInvoker)
	if err == nil {
		t.Fatal("expected timeout error from dispatchRealSurveyors")
	}

	for i, d := range got {
		if d.Status != "timeout" {
			t.Errorf("dispatch[%d].Status = %q, want %q", i, d.Status, "timeout")
		}
	}
}

// timeoutTestInvoker simulates a worker that always times out.
type timeoutTestInvoker struct{}

func (ti *timeoutTestInvoker) Invoke(ctx context.Context, config codex.WorkerConfig) (codex.WorkerResult, error) {
	return codex.WorkerResult{
		WorkerName: config.WorkerName,
		Caste:      config.Caste,
		TaskID:     config.TaskID,
		Status:     "timeout",
		Duration:   time.Since(time.Now()),
		Error:      context.DeadlineExceeded,
	}, nil
}

func (ti *timeoutTestInvoker) IsAvailable(_ context.Context) bool {
	return true
}

func (ti *timeoutTestInvoker) ValidateAgent(_ string) error {
	return nil
}

type surveyorArtifactInvoker struct{}

func (s *surveyorArtifactInvoker) Invoke(_ context.Context, config codex.WorkerConfig) (codex.WorkerResult, error) {
	claims := codex.WorkerResult{
		WorkerName: config.WorkerName,
		Caste:      config.Caste,
		TaskID:     config.TaskID,
		Status:     "completed",
		Summary:    "worker-authored survey artifact",
	}
	if config.Caste == "surveyor-provisions" {
		target := filepath.Join(config.Root, ".aether", "data", "survey", "PROVISIONS.md")
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return codex.WorkerResult{}, err
		}
		if err := os.WriteFile(target, []byte("# worker-authored provisions\n"), 0644); err != nil {
			return codex.WorkerResult{}, err
		}
		claims.FilesCreated = []string{filepath.ToSlash(filepath.Join(".aether", "data", "survey", "PROVISIONS.md"))}
	}
	return claims, nil
}

func (s *surveyorArtifactInvoker) IsAvailable(_ context.Context) bool {
	return true
}

func (s *surveyorArtifactInvoker) ValidateAgent(_ string) error {
	return nil
}

func TestIdentifyPathogens_NoTestsDetected(t *testing.T) {
	facts := codexWorkspaceFacts{
		TestFiles: []string{},
	}
	got := identifyPathogens(facts)
	found := false
	for _, issue := range got {
		if strings.Contains(issue, "No test files detected") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected 'No test files detected' issue, got %v", got)
	}
}

func TestIdentifyPathogens_TypeSafetyGaps(t *testing.T) {
	facts := codexWorkspaceFacts{
		TestFiles:      []string{"main_test.go"},
		TypeSafetyGaps: []string{"file1.go:10 interface{}", "file2.go:20 : any"},
	}
	got := identifyPathogens(facts)
	found := false
	for _, issue := range got {
		if strings.Contains(issue, "Type safety gaps found in 2 file(s)") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected 'Type safety gaps found in 2 file(s)' issue, got %v", got)
	}
}

func TestIdentifyPathogens_HighSecurityPatterns(t *testing.T) {
	facts := codexWorkspaceFacts{
		TestFiles:        []string{"main_test.go"},
		SecurityPatterns: []string{"a", "b", "c", "d"},
	}
	got := identifyPathogens(facts)
	found := false
	for _, issue := range got {
		if strings.Contains(issue, "High volume of env/eval patterns (4)") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected 'High volume of env/eval patterns (4)' issue, got %v", got)
	}
}

func TestIdentifyPathogens_ManyTODOs(t *testing.T) {
	facts := codexWorkspaceFacts{
		TestFiles: []string{"main_test.go"},
		TODOs:     []string{"a", "b", "c", "d", "e", "f"},
	}
	got := identifyPathogens(facts)
	found := false
	for _, issue := range got {
		if strings.Contains(issue, "6 TODO/FIXME/HACK markers need review") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected '6 TODO/FIXME/HACK markers need review' issue, got %v", got)
	}
}

func TestIdentifyPathogens_NoDependenciesLargeRepo(t *testing.T) {
	facts := codexWorkspaceFacts{
		TestFiles:       []string{"main_test.go"},
		KeyDependencies: []string{},
		FileCount:       20,
	}
	got := identifyPathogens(facts)
	found := false
	for _, issue := range got {
		if strings.Contains(issue, "No dependency manifest detected") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected 'No dependency manifest detected' issue, got %v", got)
	}
}

func TestIdentifyPathogens_NoIssuesCleanRepo(t *testing.T) {
	facts := codexWorkspaceFacts{
		TestFiles:        []string{"main_test.go"},
		TypeSafetyGaps:   []string{},
		SecurityPatterns: []string{"a", "b"},
		TODOs:            []string{"a", "b"},
		KeyDependencies:  []string{"github.com/spf13/cobra"},
		FileCount:        5,
	}
	got := identifyPathogens(facts)
	if len(got) != 1 {
		t.Fatalf("expected exactly 1 issue for clean repo, got %d: %v", len(got), got)
	}
	if got[0] != "No obvious technical debt markers detected." {
		t.Errorf("expected clean message, got %q", got[0])
	}
}

func TestIdentifyPathogens_MultipleIssues(t *testing.T) {
	facts := codexWorkspaceFacts{
		TestFiles:        []string{},
		TypeSafetyGaps:   []string{"file.go:1 interface{}"},
		SecurityPatterns: []string{"a", "b", "c", "d"},
		TODOs:            []string{"a", "b", "c", "d", "e", "f"},
		KeyDependencies:  []string{},
		FileCount:        20,
	}
	got := identifyPathogens(facts)
	if len(got) < 4 {
		t.Errorf("expected at least 4 issues, got %d: %v", len(got), got)
	}
}
