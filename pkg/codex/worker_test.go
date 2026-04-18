package codex

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// --- WorkerConfig tests ---

func TestWorkerConfig_DefaultTimeout(t *testing.T) {
	cfg := WorkerConfig{
		AgentName:     "aether-builder",
		AgentTOMLPath: "/some/path.toml",
		Caste:         "builder",
		WorkerName:    "Hammer-23",
		TaskID:        "2.1",
		TaskBrief:     "Do work",
		Root:          "/tmp/repo",
	}

	if cfg.Timeout != 0 {
		t.Fatalf("expected zero-value timeout, got %v", cfg.Timeout)
	}

	// The default timeout of 10 minutes is applied by the invoker,
	// not the config struct itself. Zero means "use default".
	effective := cfg.effectiveTimeout()
	if effective != 10*time.Minute {
		t.Fatalf("expected default effective timeout of 10m, got %v", effective)
	}
}

func TestWorkerConfig_CustomTimeout(t *testing.T) {
	cfg := WorkerConfig{
		Timeout: 5 * time.Minute,
	}

	effective := cfg.effectiveTimeout()
	if effective != 5*time.Minute {
		t.Fatalf("expected custom timeout of 5m, got %v", effective)
	}
}

// --- FakeInvoker tests ---

func TestFakeInvoker_ReturnsDeterministicResults(t *testing.T) {
	invoker := &FakeInvoker{}

	ctx := context.Background()
	cfg := WorkerConfig{
		AgentName:      "aether-builder",
		AgentTOMLPath:  "/path/to/aether-builder.toml",
		Caste:          "builder",
		WorkerName:     "Hammer-23",
		TaskID:         "2.1",
		TaskBrief:      "Build the thing",
		ContextCapsule: "--- CONTEXT CAPSULE ---\nGoal: test\n--- END CONTEXT CAPSULE ---",
		Root:           "/tmp/repo",
	}

	result, err := invoker.Invoke(ctx, cfg)
	if err != nil {
		t.Fatalf("FakeInvoker.Invoke returned error: %v", err)
	}

	// Verify deterministic result fields
	if result.WorkerName != "Hammer-23" {
		t.Errorf("WorkerName = %q, want %q", result.WorkerName, "Hammer-23")
	}
	if result.Caste != "builder" {
		t.Errorf("Caste = %q, want %q", result.Caste, "builder")
	}
	if result.TaskID != "2.1" {
		t.Errorf("TaskID = %q, want %q", result.TaskID, "2.1")
	}
	if result.Status != "completed" {
		t.Errorf("Status = %q, want %q", result.Status, "completed")
	}
	if result.Summary == "" {
		t.Error("Summary is empty, expected non-empty")
	}
	if result.Duration <= 0 {
		t.Error("Duration should be positive")
	}
	if result.RawOutput == "" {
		t.Error("RawOutput should not be empty")
	}

	// Invoke again to verify determinism
	result2, err := invoker.Invoke(ctx, cfg)
	if err != nil {
		t.Fatalf("second FakeInvoker.Invoke returned error: %v", err)
	}
	if result2.Summary != result.Summary {
		t.Errorf("FakeInvoker not deterministic: summaries differ: %q vs %q", result.Summary, result2.Summary)
	}
}

func TestFakeInvoker_IsAvailable_AlwaysTrue(t *testing.T) {
	invoker := &FakeInvoker{}
	if !invoker.IsAvailable(context.Background()) {
		t.Error("FakeInvoker.IsAvailable should always return true")
	}
}

func TestFakeInvoker_ValidateAgent_ReturnsNil(t *testing.T) {
	invoker := &FakeInvoker{}
	err := invoker.ValidateAgent("/any/path.toml")
	if err != nil {
		t.Errorf("FakeInvoker.ValidateAgent should return nil, got: %v", err)
	}
}

// --- ParseWorkerOutput tests ---

func TestParseWorkerOutput_ExtractsTrailingJSON(t *testing.T) {
	output := `Some text output from the worker.
More lines of output.
{"ant_name":"Hammer-23","caste":"builder","task_id":"2.1","status":"completed","summary":"Did work","files_created":["file.go"],"files_modified":[],"tests_written":[],"tool_count":5,"blockers":[],"spawns":[]}`

	result, err := ParseWorkerOutput(output)
	if err != nil {
		t.Fatalf("ParseWorkerOutput returned error: %v", err)
	}
	if result.AntName != "Hammer-23" {
		t.Errorf("AntName = %q, want %q", result.AntName, "Hammer-23")
	}
	if result.Status != "completed" {
		t.Errorf("Status = %q, want %q", result.Status, "completed")
	}
	if result.ToolCount != 5 {
		t.Errorf("ToolCount = %d, want %d", result.ToolCount, 5)
	}
	if len(result.FilesCreated) != 1 || result.FilesCreated[0] != "file.go" {
		t.Errorf("FilesCreated = %v, want [file.go]", result.FilesCreated)
	}
}

func TestParseWorkerOutput_HandlesNoJSON(t *testing.T) {
	output := `Just plain text output
with no JSON at all
not even a brace`

	_, err := ParseWorkerOutput(output)
	if err == nil {
		t.Fatal("ParseWorkerOutput should return error for no JSON")
	}
	if !strings.Contains(err.Error(), "no JSON") {
		t.Errorf("error should mention 'no JSON', got: %v", err)
	}
}

func TestParseWorkerOutput_HandlesMultipleJSONObjects(t *testing.T) {
	output := `{"event":"log","message":"started"}
Some text between
{"event":"progress","percent":50}
Final output text
{"ant_name":"Forge-98","caste":"builder","task_id":"1.1","status":"completed","summary":"Last one","files_created":[],"files_modified":[],"tests_written":[],"tool_count":0,"blockers":[],"spawns":[]}`

	result, err := ParseWorkerOutput(output)
	if err != nil {
		t.Fatalf("ParseWorkerOutput returned error: %v", err)
	}
	// Should take the LAST JSON object (the claims block)
	if result.AntName != "Forge-98" {
		t.Errorf("AntName = %q, want %q (should be last JSON object)", result.AntName, "Forge-98")
	}
	if result.Status != "completed" {
		t.Errorf("Status = %q, want %q", result.Status, "completed")
	}
}

func TestParseWorkerOutput_HandlesInvalidJSON(t *testing.T) {
	output := `text
{not valid json at all}`

	_, err := ParseWorkerOutput(output)
	if err == nil {
		t.Fatal("ParseWorkerOutput should return error for invalid JSON")
	}
}

func TestParseWorkerOutput_HandlesEmptyInput(t *testing.T) {
	_, err := ParseWorkerOutput("")
	if err == nil {
		t.Fatal("ParseWorkerOutput should return error for empty input")
	}
}

// --- NewWorkerInvoker factory tests ---

func TestNewWorkerInvoker_FakeByDefaultInTests(t *testing.T) {
	t.Setenv("AETHER_CODEX_REAL_DISPATCH", "")

	invoker := NewWorkerInvoker()
	if _, ok := invoker.(*FakeInvoker); !ok {
		t.Errorf("expected FakeInvoker inside go test binary, got %T", invoker)
	}
}

func TestNewWorkerInvoker_RealWhenEnvSet(t *testing.T) {
	t.Setenv("AETHER_CODEX_REAL_DISPATCH", "1")

	invoker := NewWorkerInvoker()
	if _, ok := invoker.(*RealInvoker); !ok {
		t.Errorf("expected RealInvoker when env=1, got %T", invoker)
	}
}

func TestNewWorkerInvoker_RealWhenEnvTrue(t *testing.T) {
	t.Setenv("AETHER_CODEX_REAL_DISPATCH", "true")

	invoker := NewWorkerInvoker()
	if _, ok := invoker.(*RealInvoker); !ok {
		t.Errorf("expected RealInvoker when env=true, got %T", invoker)
	}
}

func TestNewWorkerInvoker_FakeWhenEnvExplicitFalse(t *testing.T) {
	t.Setenv("AETHER_CODEX_REAL_DISPATCH", "false")

	invoker := NewWorkerInvoker()
	if _, ok := invoker.(*FakeInvoker); !ok {
		t.Errorf("expected FakeInvoker when env=false, got %T", invoker)
	}
}

// --- RealInvoker tests ---

func TestRealInvoker_IsAvailable_ChecksBinary(t *testing.T) {
	invoker := NewRealInvoker()

	// Use a binary that definitely exists
	invoker.binaryName = "go"
	if !invoker.IsAvailable(context.Background()) {
		t.Error("RealInvoker.IsAvailable should return true for 'go' binary")
	}

	// Use a binary that definitely does not exist
	invoker.binaryName = "nonexistent_codex_binary_12345"
	if invoker.IsAvailable(context.Background()) {
		t.Error("RealInvoker.IsAvailable should return false for nonexistent binary")
	}
}

func TestRealInvoker_Invoke_ErrorWhenNotAvailable(t *testing.T) {
	invoker := NewRealInvoker()
	invoker.binaryName = "nonexistent_codex_binary_12345"

	ctx := context.Background()
	cfg := WorkerConfig{
		AgentName:     "aether-builder",
		AgentTOMLPath: "/path/to/aether-builder.toml",
		Caste:         "builder",
		WorkerName:    "Hammer-23",
		TaskID:        "2.1",
		Root:          "/tmp/repo",
	}

	_, err := invoker.Invoke(ctx, cfg)
	if err == nil {
		t.Fatal("RealInvoker.Invoke should return error when binary not found")
	}
	if !strings.Contains(err.Error(), "not found") && !strings.Contains(err.Error(), "no such file") {
		t.Errorf("error should mention binary not found, got: %v", err)
	}
}

func TestRealInvoker_Invoke_UsesAgentPromptAndFinalMessageFile(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell stub uses POSIX sh")
	}

	dir := t.TempDir()
	tomlPath := filepath.Join(dir, "aether-builder.toml")
	if err := os.WriteFile(tomlPath, []byte(`name = "aether-builder"
description = "Builder"
nickname_candidates = ["builder", "hammer"]
developer_instructions = '''
You are the Builder.
'''
`), 0644); err != nil {
		t.Fatalf("failed to write agent TOML: %v", err)
	}

	capturePath := filepath.Join(dir, "captured-prompt.txt")
	scriptPath := filepath.Join(dir, "fake-codex.sh")
	script := `#!/bin/sh
out=""
schema=""
while [ "$#" -gt 0 ]; do
  case "$1" in
    --output-last-message)
      out="$2"
      shift 2
      ;;
    --output-schema)
      schema="$2"
      shift 2
      ;;
    *)
      shift
      ;;
  esac
done
cat > "$CAPTURE_PATH"
test -n "$schema" || exit 13
printf '{"ant_name":"Hammer-23","caste":"builder","task_id":"2.1","status":"completed","summary":"implemented the task","files_created":["new.go"],"files_modified":["existing.go"],"tests_written":["existing_test.go"],"tool_count":3,"blockers":[],"spawns":[]}' > "$out"
printf '{"event":"done"}\n'
`
	if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
		t.Fatalf("failed to write fake codex script: %v", err)
	}

	invoker := NewRealInvoker()
	invoker.binaryName = scriptPath

	cfg := WorkerConfig{
		AgentName:      "aether-builder",
		AgentTOMLPath:  tomlPath,
		Caste:          "builder",
		WorkerName:     "Hammer-23",
		TaskID:         "2.1",
		TaskBrief:      "Build the feature.",
		ContextCapsule: "--- CONTEXT CAPSULE ---\nGoal: test\n--- END CONTEXT CAPSULE ---",
		Root:           dir,
	}

	oldCapture := os.Getenv("CAPTURE_PATH")
	t.Setenv("CAPTURE_PATH", capturePath)
	defer os.Setenv("CAPTURE_PATH", oldCapture)

	result, err := invoker.Invoke(context.Background(), cfg)
	if err != nil {
		t.Fatalf("RealInvoker.Invoke returned error: %v", err)
	}
	if result.Status != "completed" {
		t.Fatalf("status = %q, want completed", result.Status)
	}
	if len(result.FilesCreated) != 1 || result.FilesCreated[0] != "new.go" {
		t.Fatalf("FilesCreated = %v, want [new.go]", result.FilesCreated)
	}
	if len(result.FilesModified) != 1 || result.FilesModified[0] != "existing.go" {
		t.Fatalf("FilesModified = %v, want [existing.go]", result.FilesModified)
	}

	promptData, err := os.ReadFile(capturePath)
	if err != nil {
		t.Fatalf("failed to read captured prompt: %v", err)
	}
	prompt := string(promptData)
	for _, want := range []string{"You are the Builder.", "Goal: test", "Build the feature.", "Final Response Contract"} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("prompt missing %q\n%s", want, prompt)
		}
	}
}

func TestRealInvoker_Invoke_PassesConfigOverrides(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell stub uses POSIX sh")
	}

	dir := t.TempDir()
	tomlPath := filepath.Join(dir, "aether-builder.toml")
	if err := os.WriteFile(tomlPath, []byte(`name = "aether-builder"
description = "Builder"
nickname_candidates = ["builder", "hammer"]
developer_instructions = '''
You are the Builder.
'''
`), 0644); err != nil {
		t.Fatalf("failed to write agent TOML: %v", err)
	}

	argsPath := filepath.Join(dir, "captured-args.txt")
	scriptPath := filepath.Join(dir, "fake-codex.sh")
	script := `#!/bin/sh
printf '%s\n' "$@" > "$ARGS_PATH"
out=""
while [ "$#" -gt 0 ]; do
  case "$1" in
    --output-last-message)
      out="$2"
      shift 2
      ;;
    *)
      shift
      ;;
  esac
done
cat >/dev/null
printf '{"ant_name":"Hammer-23","caste":"builder","task_id":"2.1","status":"completed","summary":"implemented the task","files_created":[],"files_modified":[],"tests_written":[],"tool_count":0,"blockers":[],"spawns":[]}' > "$out"
`
	if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
		t.Fatalf("failed to write fake codex script: %v", err)
	}

	invoker := NewRealInvoker()
	invoker.binaryName = scriptPath

	t.Setenv("ARGS_PATH", argsPath)
	t.Setenv("CODEX_HOME", filepath.Join(dir, "codex-home"))

	cfg := WorkerConfig{
		AgentName:       "aether-builder",
		AgentTOMLPath:   tomlPath,
		Caste:           "builder",
		WorkerName:      "Hammer-23",
		TaskID:          "2.1",
		TaskBrief:       "Build the feature.",
		ContextCapsule:  "Goal: test",
		Root:            dir,
		ConfigOverrides: []string{`model_reasoning_effort="medium"`, `model="gpt-5.4"`},
	}

	if _, err := invoker.Invoke(context.Background(), cfg); err != nil {
		t.Fatalf("RealInvoker.Invoke returned error: %v", err)
	}

	argsData, err := os.ReadFile(argsPath)
	if err != nil {
		t.Fatalf("failed to read captured args: %v", err)
	}
	argsText := string(argsData)
	for _, want := range []string{"--add-dir", filepath.Join(dir, "codex-home"), "-c", `model_reasoning_effort="medium"`, `model="gpt-5.4"`} {
		if !strings.Contains(argsText, want) {
			t.Fatalf("captured args missing %q\n%s", want, argsText)
		}
	}
}

func TestCodexWritableDirs_PrefersEnv(t *testing.T) {
	t.Setenv("CODEX_HOME", "  /tmp/custom-codex-home  ")
	t.Setenv("HOME", t.TempDir())

	dirs := codexWritableDirs()
	if len(dirs) != 1 {
		t.Fatalf("expected 1 writable dir, got %v", dirs)
	}
	if dirs[0] != "/tmp/custom-codex-home" {
		t.Fatalf("expected CODEX_HOME to win, got %q", dirs[0])
	}
}

func TestCodexWritableDirs_FallsBackToUserHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("CODEX_HOME", "")
	t.Setenv("HOME", home)

	dirs := codexWritableDirs()
	if len(dirs) != 1 {
		t.Fatalf("expected 1 writable dir, got %v", dirs)
	}
	want := filepath.Join(home, ".codex")
	if dirs[0] != want {
		t.Fatalf("expected fallback writable dir %q, got %q", want, dirs[0])
	}
}

// --- ValidateAgent tests ---

func TestValidateAgent_ValidTOML(t *testing.T) {
	dir := t.TempDir()
	tomlPath := filepath.Join(dir, "valid-agent.toml")
	content := `name = "test-agent"
description = "A test agent"
nickname_candidates = ["test"]

developer_instructions = '''
You are a test agent.
Do things.
'''
`
	if err := os.WriteFile(tomlPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test TOML: %v", err)
	}

	invoker := NewRealInvoker()
	err := invoker.ValidateAgent(tomlPath)
	if err != nil {
		t.Errorf("ValidateAgent should return nil for valid TOML, got: %v", err)
	}
}

func TestValidateAgent_MissingRequiredFields(t *testing.T) {
	dir := t.TempDir()

	tests := []struct {
		name    string
		content string
	}{
		{
			name:    "missing_name",
			content: `description = "no name"\ndeveloper_instructions = "no name"`,
		},
		{
			name:    "missing_description",
			content: `name = "test"\ndeveloper_instructions = "no desc"`,
		},
		{
			name:    "missing_developer_instructions",
			content: `name = "test"\ndescription = "no instructions"`,
		},
	}

	invoker := NewRealInvoker()

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tomlPath := filepath.Join(dir, tc.name+".toml")
			if err := os.WriteFile(tomlPath, []byte(tc.content), 0644); err != nil {
				t.Fatalf("failed to write test TOML: %v", err)
			}

			err := invoker.ValidateAgent(tomlPath)
			if err == nil {
				t.Errorf("ValidateAgent should return error for %s", tc.name)
			}
		})
	}
}

func TestValidateAgent_FileNotFound(t *testing.T) {
	invoker := NewRealInvoker()
	err := invoker.ValidateAgent("/nonexistent/path/agent.toml")
	if err == nil {
		t.Error("ValidateAgent should return error for nonexistent file")
	}
}

func TestValidateAgent_InvalidTOML(t *testing.T) {
	dir := t.TempDir()
	tomlPath := filepath.Join(dir, "bad-toml.toml")
	content := `this is not valid [TOML {{{`
	if err := os.WriteFile(tomlPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test TOML: %v", err)
	}

	invoker := NewRealInvoker()
	err := invoker.ValidateAgent(tomlPath)
	if err == nil {
		t.Error("ValidateAgent should return error for invalid TOML syntax")
	}
}

// --- WorkerInvoker interface compliance ---

func TestFakeInvoker_ImplementsInterface(t *testing.T) {
	var _ WorkerInvoker = &FakeInvoker{}
}

func TestRealInvoker_ImplementsInterface(t *testing.T) {
	var _ WorkerInvoker = &RealInvoker{}
}
