package codex

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// --- AETHER_OPENCODE_AGENT_URL env var injection tests ---

func TestInvokeHostedWorkerEnvVarOverride(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell stub uses POSIX sh")
	}

	t.Setenv(envOpenCodeAgentURL, "http://localhost:9876/agent")

	dir := t.TempDir()
	agentPath := createTestMarkdownAgent(t, dir, "aether-builder", "Builder")

	envCapturePath := filepath.Join(dir, "captured-env.txt")
	scriptPath := filepath.Join(dir, "fake-opencode.sh")
	script := `#!/bin/sh
	env | grep -i AETHER > "$ENV_CAPTURE_PATH"
	cat <<'EOF'
{"type":"message.part.updated","part":{"type":"text","text":"{\"ant_name\":\"Forge-1\",\"caste\":\"builder\",\"task_id\":\"1.1\",\"status\":\"completed\",\"summary\":\"done\",\"files_created\":[],\"files_modified\":[],\"tests_written\":[],\"tool_count\":0,\"blockers\":[],\"spawns\":[]}"}}
EOF
`
	if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
		t.Fatalf("failed to write fake opencode script: %v", err)
	}

	invoker := &OpenCodeDispatcher{binaryName: scriptPath}
	t.Setenv("ENV_CAPTURE_PATH", envCapturePath)
	t.Setenv(envOpenCodePrimary, "")

	_, err := invoker.Invoke(t.Context(), WorkerConfig{
		AgentName:      "aether-builder",
		AgentTOMLPath:  agentPath,
		Caste:          "builder",
		WorkerName:     "Forge-1",
		TaskID:         "1.1",
		TaskBrief:      "Build the feature.",
		ContextCapsule: "Goal: test",
		Root:           dir,
	})
	if err != nil {
		t.Fatalf("OpenCode Invoke returned error: %v", err)
	}

	envData, err := os.ReadFile(envCapturePath)
	if err != nil {
		t.Fatalf("failed to read captured env: %v", err)
	}
	envText := string(envData)

	// Verify the subprocess received AETHER_OPENCODE_AGENT_URL
	if !strings.Contains(envText, "AETHER_OPENCODE_AGENT_URL=http://localhost:9876/agent") {
		t.Fatalf("expected subprocess to receive AETHER_OPENCODE_AGENT_URL in env, got:\n%s", envText)
	}
}

func TestInvokeHostedWorkerNoEnvVarOverride(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell stub uses POSIX sh")
	}

	// Ensure the env var is NOT set
	t.Setenv(envOpenCodeAgentURL, "")

	dir := t.TempDir()
	agentPath := createTestMarkdownAgent(t, dir, "aether-builder", "Builder")

	envCapturePath := filepath.Join(dir, "captured-env.txt")
	scriptPath := filepath.Join(dir, "fake-opencode.sh")
	script := `#!/bin/sh
	env | grep -i AETHER > "$ENV_CAPTURE_PATH"
	cat <<'EOF'
{"type":"message.part.updated","part":{"type":"text","text":"{\"ant_name\":\"Forge-2\",\"caste\":\"builder\",\"task_id\":\"2.1\",\"status\":\"completed\",\"summary\":\"done\",\"files_created\":[],\"files_modified\":[],\"tests_written\":[],\"tool_count\":0,\"blockers\":[],\"spawns\":[]}"}}
EOF
`
	if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
		t.Fatalf("failed to write fake opencode script: %v", err)
	}

	invoker := &OpenCodeDispatcher{binaryName: scriptPath}
	t.Setenv("ENV_CAPTURE_PATH", envCapturePath)
	t.Setenv(envOpenCodePrimary, "")

	_, err := invoker.Invoke(t.Context(), WorkerConfig{
		AgentName:      "aether-builder",
		AgentTOMLPath:  agentPath,
		Caste:          "builder",
		WorkerName:     "Forge-2",
		TaskID:         "2.1",
		TaskBrief:      "Build the feature.",
		ContextCapsule: "Goal: test",
		Root:           dir,
	})
	if err != nil {
		t.Fatalf("OpenCode Invoke returned error: %v", err)
	}

	envData, err := os.ReadFile(envCapturePath)
	if err != nil {
		t.Fatalf("failed to read captured env: %v", err)
	}
	envText := string(envData)

	// Verify AETHER_OPENCODE_AGENT_URL is empty (not overridden) in the subprocess env.
	// When t.Setenv sets it to "", the var name still appears in env but with no value.
	// The important check: it should NOT have a non-empty URL value.
	for _, line := range strings.Split(envText, "\n") {
		if strings.HasPrefix(line, "AETHER_OPENCODE_AGENT_URL=") {
			val := strings.TrimPrefix(line, "AETHER_OPENCODE_AGENT_URL=")
			if val != "" {
				t.Fatalf("expected AETHER_OPENCODE_AGENT_URL to be empty in subprocess, got value: %q", val)
			}
		}
	}
}

// createTestMarkdownAgent creates a minimal markdown agent file for testing.
func createTestMarkdownAgent(t *testing.T, dir, name, description string) string {
	t.Helper()
	agentPath := filepath.Join(dir, name+".md")
	content := "---\nname: " + name + "\ndescription: " + description + "\nmode: subagent\n---\nYou are the " + description + ".\n"
	if err := os.WriteFile(agentPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write agent markdown: %v", err)
	}
	return agentPath
}
