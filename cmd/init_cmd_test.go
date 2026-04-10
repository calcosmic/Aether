package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/storage"
)

// --- aether init tests ---

func TestInitCmd_BasicInit(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	tmpDir := t.TempDir()
	dataDir := tmpDir + "/.aether/data"
	os.MkdirAll(dataDir, 0755)

	origDir := os.Getenv("COLONY_DATA_DIR")
	os.Setenv("COLONY_DATA_DIR", dataDir)
	defer os.Setenv("COLONY_DATA_DIR", origDir)

	rootCmd.SetArgs([]string{"init", "Build feature X"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	if env["ok"] != true {
		t.Fatalf("expected ok:true, got: %v", env["ok"])
	}

	// Verify COLONY_STATE.json was created
	s, err := storage.NewStore(dataDir)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}

	var state colony.ColonyState
	if err := s.LoadJSON("COLONY_STATE.json", &state); err != nil {
		t.Fatalf("COLONY_STATE.json not found: %v", err)
	}

	if state.Goal == nil || *state.Goal != "Build feature X" {
		t.Errorf("goal = %v, want 'Build feature X'", state.Goal)
	}
	if state.State != colony.StateREADY {
		t.Errorf("state = %v, want READY", state.State)
	}
	if state.CurrentPhase != 0 {
		t.Errorf("current_phase = %d, want 0", state.CurrentPhase)
	}
	if state.Version != "3.0" {
		t.Errorf("version = %q, want '3.0'", state.Version)
	}

	// Verify session.json was created
	var session colony.SessionFile
	if err := s.LoadJSON("session.json", &session); err != nil {
		t.Fatalf("session.json not found: %v", err)
	}
	if session.ColonyGoal != "Build feature X" {
		t.Errorf("session.colony_goal = %q, want 'Build feature X'", session.ColonyGoal)
	}

	// Verify CONTEXT.md was created
	contextPath := filepath.Join(tmpDir, ".aether", "CONTEXT.md")
	data, err := os.ReadFile(contextPath)
	if err != nil {
		t.Fatalf("CONTEXT.md not found: %v", err)
	}
	if !strings.Contains(string(data), "Build feature X") {
		t.Errorf("CONTEXT.md does not contain goal")
	}

	// Verify activity.log was created
	activityPath := filepath.Join(dataDir, "activity.log")
	if _, err := os.Stat(activityPath); err != nil {
		t.Fatalf("activity.log not found: %v", err)
	}
}

func TestInitCmd_Idempotent(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	tmpDir := t.TempDir()
	dataDir := tmpDir + "/.aether/data"
	os.MkdirAll(dataDir, 0755)

	origDir := os.Getenv("COLONY_DATA_DIR")
	os.Setenv("COLONY_DATA_DIR", dataDir)
	defer os.Setenv("COLONY_DATA_DIR", origDir)

	s, _ := storage.NewStore(dataDir)

	// Create initial state
	goal := "Original goal"
	state := colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateREADY,
		CurrentPhase: 2,
	}
	s.SaveJSON("COLONY_STATE.json", state)

	// Running init again should not overwrite
	// outputError writes to stderr
	var buf bytes.Buffer
	stderr = &buf

	rootCmd.SetArgs([]string{"init", "New goal"})
	rootCmd.Execute()

	env := parseEnvelope(t, buf.String())
	if env["ok"] != false {
		t.Fatalf("expected ok:false for already-initialized colony, got: %v", env["ok"])
	}

	// State should be unchanged
	var stateAfter colony.ColonyState
	s.LoadJSON("COLONY_STATE.json", &stateAfter)
	if stateAfter.Goal == nil || *stateAfter.Goal != "Original goal" {
		t.Errorf("goal was overwritten: %v, want 'Original goal'", stateAfter.Goal)
	}
	if stateAfter.CurrentPhase != 2 {
		t.Errorf("current_phase was reset: %d, want 2", stateAfter.CurrentPhase)
	}
}

func TestInitCmd_MissingGoal(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	tmpDir := t.TempDir()
	dataDir := tmpDir + "/.aether/data"
	os.MkdirAll(dataDir, 0755)

	origDir := os.Getenv("COLONY_DATA_DIR")
	os.Setenv("COLONY_DATA_DIR", dataDir)
	defer os.Setenv("COLONY_DATA_DIR", origDir)

	rootCmd.SetArgs([]string{"init"})

	err := rootCmd.Execute()

	// Cobra's ExactArgs(1) validation should return an error
	if err == nil {
		t.Error("expected error for missing goal argument, got nil")
	}
}

func TestInitCmd_CreatesDirectoryStructure(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	tmpDir := t.TempDir()

	origDir := os.Getenv("COLONY_DATA_DIR")
	os.Setenv("COLONY_DATA_DIR", tmpDir+"/.aether/data")
	defer os.Setenv("COLONY_DATA_DIR", origDir)

	rootCmd.SetArgs([]string{"init", "Test goal"})
	rootCmd.Execute()

	// Verify .aether/data directory exists
	dataDir := tmpDir + "/.aether/data"
	info, err := os.Stat(dataDir)
	if err != nil {
		t.Fatalf(".aether/data not created: %v", err)
	}
	if !info.IsDir() {
		t.Error(".aether/data is not a directory")
	}

	// Verify .aether/dreams directory exists
	dreamsDir := tmpDir + "/.aether/dreams"
	info, err = os.Stat(dreamsDir)
	if err != nil {
		t.Fatalf(".aether/dreams not created: %v", err)
	}
	if !info.IsDir() {
		t.Error(".aether/dreams is not a directory")
	}
}

func TestInitCmd_ColonyStateStructure(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	tmpDir := t.TempDir()
	dataDir := tmpDir + "/.aether/data"
	os.MkdirAll(dataDir, 0755)

	origDir := os.Getenv("COLONY_DATA_DIR")
	os.Setenv("COLONY_DATA_DIR", dataDir)
	defer os.Setenv("COLONY_DATA_DIR", origDir)

	rootCmd.SetArgs([]string{"init", "Structure test"})
	rootCmd.Execute()

	s, _ := storage.NewStore(dataDir)

	var state colony.ColonyState
	s.LoadJSON("COLONY_STATE.json", &state)

	// Verify all required fields have sensible defaults
	if state.ColonyVersion != 0 {
		t.Errorf("colony_version = %d, want 0", state.ColonyVersion)
	}
	if state.Milestone != "" {
		t.Errorf("milestone = %q, want empty", state.Milestone)
	}
	if state.InitializedAt == nil {
		t.Error("initialized_at should not be nil")
	}
	if state.Memory.PhaseLearnings == nil {
		t.Error("memory.phase_learnings should not be nil")
	}
	if state.Memory.Decisions == nil {
		t.Error("memory.decisions should not be nil")
	}
	if state.Memory.Instincts == nil {
		t.Error("memory.instincts should not be nil")
	}
	if state.Errors.Records == nil {
		t.Error("errors.records should not be nil")
	}
	if state.Errors.FlaggedPatterns == nil {
		t.Error("errors.flagged_patterns should not be nil")
	}
	if state.Signals == nil {
		t.Error("signals should not be nil")
	}
	if state.Graveyards == nil {
		t.Error("graveyards should not be nil")
	}
	if state.Events == nil {
		t.Error("events should not be nil")
	}
	if state.Plan.Phases == nil {
		t.Error("plan.phases should not be nil")
	}
}

func TestInitCmd_SessionStructure(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	tmpDir := t.TempDir()
	dataDir := tmpDir + "/.aether/data"
	os.MkdirAll(dataDir, 0755)

	origDir := os.Getenv("COLONY_DATA_DIR")
	os.Setenv("COLONY_DATA_DIR", dataDir)
	defer os.Setenv("COLONY_DATA_DIR", origDir)

	rootCmd.SetArgs([]string{"init", "Session test"})
	rootCmd.Execute()

	s, _ := storage.NewStore(dataDir)

	var session colony.SessionFile
	s.LoadJSON("session.json", &session)

	if session.SessionID == "" {
		t.Error("session.session_id should not be empty")
	}
	if session.ColonyGoal != "Session test" {
		t.Errorf("session.colony_goal = %q, want 'Session test'", session.ColonyGoal)
	}
	if session.CurrentPhase != 0 {
		t.Errorf("session.current_phase = %d, want 0", session.CurrentPhase)
	}
	if session.StartedAt == "" {
		t.Error("session.started_at should not be empty")
	}
}

func TestInitCmd_ContextMDContent(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	tmpDir := t.TempDir()
	dataDir := tmpDir + "/.aether/data"
	os.MkdirAll(dataDir, 0755)

	origDir := os.Getenv("COLONY_DATA_DIR")
	os.Setenv("COLONY_DATA_DIR", dataDir)
	defer os.Setenv("COLONY_DATA_DIR", origDir)

	rootCmd.SetArgs([]string{"init", "Context content test"})
	rootCmd.Execute()

	contextPath := filepath.Join(tmpDir, ".aether", "CONTEXT.md")
	data, err := os.ReadFile(contextPath)
	if err != nil {
		t.Fatalf("CONTEXT.md not found: %v", err)
	}

	content := string(data)
	requiredStrings := []string{
		"Context content test",
		"# Colony Context",
	}
	for _, s := range requiredStrings {
		if !strings.Contains(content, s) {
			t.Errorf("CONTEXT.md missing required string %q", s)
		}
	}
}

func TestInitCmd_ActivityLog(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	tmpDir := t.TempDir()
	dataDir := tmpDir + "/.aether/data"
	os.MkdirAll(dataDir, 0755)

	origDir := os.Getenv("COLONY_DATA_DIR")
	os.Setenv("COLONY_DATA_DIR", dataDir)
	defer os.Setenv("COLONY_DATA_DIR", origDir)

	rootCmd.SetArgs([]string{"init", "Activity log test"})
	rootCmd.Execute()

	activityPath := filepath.Join(dataDir, "activity.log")
	data, err := os.ReadFile(activityPath)
	if err != nil {
		t.Fatalf("activity.log not found: %v", err)
	}

	// Verify it's a JSONL entry
	content := strings.TrimSpace(string(data))
	var entry map[string]interface{}
	if err := json.Unmarshal([]byte(content), &entry); err != nil {
		t.Fatalf("activity.log entry is not valid JSON: %v", err)
	}

	if entry["action"] != "COLONY_INITIALIZED" {
		t.Errorf("activity.action = %v, want COLONY_INITIALIZED", entry["action"])
	}
}

func TestInitCmd_OutputFormat(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	tmpDir := t.TempDir()
	dataDir := tmpDir + "/.aether/data"
	os.MkdirAll(dataDir, 0755)

	origDir := os.Getenv("COLONY_DATA_DIR")
	os.Setenv("COLONY_DATA_DIR", dataDir)
	defer os.Setenv("COLONY_DATA_DIR", origDir)

	rootCmd.SetArgs([]string{"init", "Output format test"})
	rootCmd.Execute()

	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})

	if result["state"] != "READY" {
		t.Errorf("result.state = %v, want READY", result["state"])
	}
	if result["goal"] != "Output format test" {
		t.Errorf("result.goal = %v, want 'Output format test'", result["goal"])
	}
	if result["version"] != "3.0" {
		t.Errorf("result.version = %v, want '3.0'", result["version"])
	}
}

func TestInitCmd_SealedColonyAllowsFreshInit(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	tmpDir := t.TempDir()
	dataDir := tmpDir + "/.aether/data"
	os.MkdirAll(dataDir, 0755)

	origDir := os.Getenv("COLONY_DATA_DIR")
	os.Setenv("COLONY_DATA_DIR", dataDir)
	defer os.Setenv("COLONY_DATA_DIR", origDir)

	s, _ := storage.NewStore(dataDir)

	// Create a sealed colony state
	goal := "Old sealed goal"
	state := colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateCOMPLETED,
		CurrentPhase: 5,
		Milestone:    "Crowned Anthill",
	}
	s.SaveJSON("COLONY_STATE.json", state)

	// Running init on a sealed colony should allow fresh init
	rootCmd.SetArgs([]string{"init", "New colony goal"})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	if env["ok"] != true {
		t.Fatalf("expected ok:true for fresh init after seal, got: %v", env["ok"])
	}

	var stateAfter colony.ColonyState
	s.LoadJSON("COLONY_STATE.json", &stateAfter)
	if stateAfter.Goal == nil || *stateAfter.Goal != "New colony goal" {
		t.Errorf("goal should be 'New colony goal', got: %v", stateAfter.Goal)
	}
}

func TestInitCmd_SealedColonyCreatesBackup(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	tmpDir := t.TempDir()
	dataDir := tmpDir + "/.aether/data"
	os.MkdirAll(dataDir, 0755)

	origDir := os.Getenv("COLONY_DATA_DIR")
	os.Setenv("COLONY_DATA_DIR", dataDir)
	defer os.Setenv("COLONY_DATA_DIR", origDir)

	s, _ := storage.NewStore(dataDir)

	// Create a sealed colony state
	goal := "Old sealed goal"
	state := colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateCOMPLETED,
		CurrentPhase: 5,
		Milestone:    "Crowned Anthill",
	}
	s.SaveJSON("COLONY_STATE.json", state)

	rootCmd.SetArgs([]string{"init", "New colony goal"})
	rootCmd.Execute()

	// Verify backup was created
	backupDir := filepath.Join(dataDir, "backups")
	entries, err := os.ReadDir(backupDir)
	if err != nil {
		t.Fatalf("backups directory not created: %v", err)
	}

	found := false
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "COLONY_STATE.pre-init.") && strings.HasSuffix(e.Name(), ".bak") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected backup file COLONY_STATE.pre-init.*.bak in backups directory")
	}
}

func TestSealInProgress_NonGitRepo(t *testing.T) {
	// sealInProgress should return false in a non-git directory
	tmpDir := t.TempDir()
	if sealInProgress(tmpDir) {
		t.Error("sealInProgress should return false in non-git directory")
	}
}

func TestSealInProgress_NoUncommittedChanges(t *testing.T) {
	// sealInProgress should return false when there are no uncommitted changes
	tmpDir := t.TempDir()

	// Create a git repo
	runGit(t, tmpDir, "init")
	runGit(t, tmpDir, "commit", "--allow-empty", "-m", "initial")

	dataDir := tmpDir + "/.aether/data"
	os.MkdirAll(dataDir, 0755)

	// Write a committed sealed state
	writeSealedState(t, dataDir, "committed goal")
	runGit(t, tmpDir, "add", ".")
	runGit(t, tmpDir, "commit", "-m", "seal colony")

	if sealInProgress(dataDir) {
		t.Error("sealInProgress should return false when state is committed")
	}
}

func TestSealInProgress_UncommittedSeal(t *testing.T) {
	// sealInProgress should return true when seal is uncommitted
	tmpDir := t.TempDir()

	// Create a git repo
	runGit(t, tmpDir, "init")
	runGit(t, tmpDir, "commit", "--allow-empty", "-m", "initial")

	dataDir := tmpDir + "/.aether/data"
	os.MkdirAll(dataDir, 0755)

	// Write an active (non-sealed) state and commit it
	writeActiveState(t, dataDir, "active goal")
	runGit(t, tmpDir, "add", ".")
	runGit(t, tmpDir, "commit", "-m", "active colony")

	// Now modify the state to sealed (uncommitted)
	writeSealedState(t, dataDir, "sealed goal")

	result := sealInProgress(dataDir)
	if !result {
		t.Errorf("sealInProgress returned false for dataDir=%q", dataDir)
	}
}

func TestSealInProgress_CommittedSeal(t *testing.T) {
	// sealInProgress should return false when seal is committed
	tmpDir := t.TempDir()

	// Create a git repo
	runGit(t, tmpDir, "init")
	runGit(t, tmpDir, "commit", "--allow-empty", "-m", "initial")

	dataDir := tmpDir + "/.aether/data"
	os.MkdirAll(dataDir, 0755)

	// Write a sealed state and commit it
	writeSealedState(t, dataDir, "sealed goal")
	runGit(t, tmpDir, "add", ".")
	runGit(t, tmpDir, "commit", "-m", "seal colony")

	if sealInProgress(dataDir) {
		t.Error("sealInProgress should return false when seal is committed")
	}
}

func TestInitCmd_NoPlaceholderStrings(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	tmpDir := t.TempDir()
	dataDir := tmpDir + "/.aether/data"
	os.MkdirAll(dataDir, 0755)

	origDir := os.Getenv("COLONY_DATA_DIR")
	os.Setenv("COLONY_DATA_DIR", dataDir)
	defer os.Setenv("COLONY_DATA_DIR", origDir)

	rootCmd.SetArgs([]string{"init", "Placeholder verification test"})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	if env["ok"] != true {
		t.Fatalf("expected ok:true, got: %v", env["ok"])
	}

	// Read the raw COLONY_STATE.json content as a string
	statePath := filepath.Join(dataDir, "COLONY_STATE.json")
	raw, err := os.ReadFile(statePath)
	if err != nil {
		t.Fatalf("COLONY_STATE.json not found: %v", err)
	}
	content := string(raw)

	// Placeholder patterns to search for
	placeholderPatterns := []*regexp.Regexp{
		regexp.MustCompile(`__\w+__`),     // __PLACEHOLDER__ style
		regexp.MustCompile(`\bTODO\b`),    // TODO markers
		regexp.MustCompile(`\bFIXME\b`),   // FIXME markers
		regexp.MustCompile(`\bXXX\b`),     // XXX markers
		regexp.MustCompile(`\bHACK\b`),    // HACK markers
		regexp.MustCompile(`<\w+>`),       // <PLACEHOLDER> angle-bracket style
		regexp.MustCompile(`\{\{[^}]+\}\}`), // {{placeholder}} mustache style
	}

	for _, pattern := range placeholderPatterns {
		matches := pattern.FindAllString(content, -1)
		if len(matches) > 0 {
			t.Errorf("COLONY_STATE.json contains placeholder-like strings matching %s: %v",
				pattern.String(), matches)
		}
	}

	// Also verify the state has all required fields by parsing as ColonyState
	s, err := storage.NewStore(dataDir)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	var state colony.ColonyState
	if err := s.LoadJSON("COLONY_STATE.json", &state); err != nil {
		t.Fatalf("failed to parse COLONY_STATE.json: %v", err)
	}

	requiredFieldChecks := []struct {
		name  string
		check func() bool
	}{
		{"version", func() bool { return state.Version != "" }},
		{"goal", func() bool { return state.Goal != nil && *state.Goal != "" }},
		{"state", func() bool { return state.State != "" }},
		{"current_phase", func() bool { return true }}, // always has zero value
		{"session_id", func() bool { return state.SessionID != nil && *state.SessionID != "" }},
		{"initialized_at", func() bool { return state.InitializedAt != nil }},
		{"plan.phases", func() bool { return state.Plan.Phases != nil }},
		{"memory.phase_learnings", func() bool { return state.Memory.PhaseLearnings != nil }},
		{"memory.decisions", func() bool { return state.Memory.Decisions != nil }},
		{"memory.instincts", func() bool { return state.Memory.Instincts != nil }},
		{"errors.records", func() bool { return state.Errors.Records != nil }},
		{"errors.flagged_patterns", func() bool { return state.Errors.FlaggedPatterns != nil }},
		{"signals", func() bool { return state.Signals != nil }},
		{"graveyards", func() bool { return state.Graveyards != nil }},
		{"events", func() bool { return state.Events != nil }},
	}

	for _, fc := range requiredFieldChecks {
		if !fc.check() {
			t.Errorf("required field %s is missing or zero-value", fc.name)
		}
	}
}

func TestInitCmd_ParallelModeDefault(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	tmpDir := t.TempDir()
	dataDir := tmpDir + "/.aether/data"
	os.MkdirAll(dataDir, 0755)

	origDir := os.Getenv("COLONY_DATA_DIR")
	os.Setenv("COLONY_DATA_DIR", dataDir)
	defer os.Setenv("COLONY_DATA_DIR", origDir)

	rootCmd.SetArgs([]string{"init", "Parallel mode default test"})
	rootCmd.Execute()

	s, _ := storage.NewStore(dataDir)

	var state colony.ColonyState
	s.LoadJSON("COLONY_STATE.json", &state)

	if state.ParallelMode != colony.ModeInRepo {
		t.Errorf("parallel_mode = %q, want %q (in-repo)", state.ParallelMode, colony.ModeInRepo)
	}
}

// --- helpers ---

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=test", "GIT_AUTHOR_EMAIL=test@example.com",
		"GIT_COMMITTER_NAME=test", "GIT_COMMITTER_EMAIL=test@example.com")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s in %s: %v\n%s", args[0], dir, err, out)
	}
}

func writeSealedState(t *testing.T, dataDir, goal string) {
	t.Helper()
	s, err := storage.NewStore(dataDir)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	state := colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateCOMPLETED,
		CurrentPhase: 5,
		Milestone:    "Crowned Anthill",
	}
	if err := s.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatalf("write state: %v", err)
	}
}

func writeActiveState(t *testing.T, dataDir, goal string) {
	t.Helper()
	s, err := storage.NewStore(dataDir)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	state := colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateEXECUTING,
		CurrentPhase: 2,
	}
	if err := s.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatalf("write state: %v", err)
	}
}
