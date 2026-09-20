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

// TestSpawnedWorkerCarriesItsIdentityInTheEnvironment spawns a REAL subprocess
// and reads back the environment it actually received. workerProcessEnv existed
// with no caller for months, so AETHER_WORKER_NAME never reached a worker and
// `aether hook-stop` could not tell an Aether worker from a person -- it blocked
// a live build worker and advised `aether pause`, which paused a running colony.
// Asserting the env at the process that receives it, rather than asserting the
// builder function in isolation, is the whole point: an isolated builder test
// passed the entire time the wiring was missing.
func TestSpawnedWorkerCarriesItsIdentityInTheEnvironment(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell stub uses POSIX sh")
	}

	dir := t.TempDir()
	envPath := filepath.Join(dir, "worker-env.txt")
	binary := filepath.Join(dir, "claude")
	script := `#!/bin/sh
if [ "$*" = "auth status --json" ]; then
  echo '{"loggedIn":true}'
  exit 0
fi
env > "` + envPath + `"
echo '{"status":"completed","summary":"ok"}'
exit 0
`
	if err := os.WriteFile(binary, []byte(script), 0755); err != nil {
		t.Fatalf("write fake claude: %v", err)
	}

	agentPath := filepath.Join(dir, "aether-builder.md")
	if err := os.WriteFile(agentPath, []byte("---\nname: aether-builder\ndescription: builder\n---\n\nBuild things.\n"), 0644); err != nil {
		t.Fatalf("write agent definition: %v", err)
	}

	dispatcher := &ClaudeDispatcher{binaryName: binary}
	res, invokeErr := dispatcher.Invoke(context.Background(), WorkerConfig{
		AgentName:     "aether-builder",
		AgentTOMLPath: agentPath,
		WorkerName:    "Weld-32",
		Caste:         "builder",
		TaskID:        "1.1",
		Root:          dir,
		TaskBrief:     "do the work",
		Timeout:       30 * time.Second,
	})

	t.Logf("invoke status=%q err=%v error=%v", res.Status, invokeErr, res.Error)
	data, err := os.ReadFile(envPath)
	if err != nil {
		t.Fatalf("the spawned worker never recorded its environment (it may not have run): %v", err)
	}
	received := string(data)
	for _, want := range []string{
		"AETHER_WORKER_NAME=Weld-32",
		"AETHER_WORKER_CASTE=builder",
	} {
		if !strings.Contains(received, want) {
			t.Errorf("spawned worker did not receive %q; a hook cannot tell it from a person.\nreceived:\n%s", want, received)
		}
	}
}

// The check above must be able to fail: an environment with no worker identity
// is caught, not quietly accepted.
func TestWorkerIdentityEnvCheckCanFail(t *testing.T) {
	plain := strings.Join(os.Environ(), "\n")
	if strings.Contains(plain, "AETHER_WORKER_NAME=") {
		t.Skip("this test process already carries a worker identity")
	}
	if strings.Contains(plain, "AETHER_WORKER_NAME=Weld-32") {
		t.Fatal("a plain environment must not satisfy the identity assertion")
	}
	built := workerProcessEnv(os.Environ(), WorkerConfig{WorkerName: "Weld-32", Caste: "builder"}, PlatformClaude)
	if !strings.Contains(strings.Join(built, "\n"), "AETHER_WORKER_NAME=Weld-32") {
		t.Fatal("workerProcessEnv stopped setting the worker name, so the assertion above could never pass")
	}
}
