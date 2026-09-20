package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// result.json is the durable event that lets later swarm runs reconstruct a
// same-problem failure streak. Cleanup may remove disposable live-display and
// response artifacts, but it must not erase that history or claim otherwise.
func TestSwarmCleanupTruthPreservesDurableResult(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, _ := newTestStore(t)
	store = s

	swarmDir := filepath.Join(s.BasePath(), "swarms", "swarm-truth")
	if err := os.MkdirAll(swarmDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	resultPath := filepath.Join(swarmDir, "result.json")
	resultData := []byte(`{"swarm_id":"swarm-truth","target":"auth panic","status":"failed","completed_at":"2026-08-31T12:00:00Z"}`)
	if err := os.WriteFile(resultPath, resultData, 0644); err != nil {
		t.Fatalf("write result: %v", err)
	}
	if err := os.WriteFile(filepath.Join(swarmDir, "findings.json"), []byte(`{"findings":[]}`), 0644); err != nil {
		t.Fatalf("write findings: %v", err)
	}
	if err := os.WriteFile(filepath.Join(swarmDir, "display.json"), []byte(`{"agents":[]}`), 0644); err != nil {
		t.Fatalf("write display: %v", err)
	}
	responsesDir := filepath.Join(swarmDir, "responses")
	if err := os.MkdirAll(responsesDir, 0755); err != nil {
		t.Fatalf("mkdir responses: %v", err)
	}
	if err := os.WriteFile(filepath.Join(responsesDir, "worker.json"), []byte(`{"status":"failed"}`), 0644); err != nil {
		t.Fatalf("write response: %v", err)
	}

	rootCmd.SetArgs([]string{"swarm-cleanup", "--id", "swarm-truth"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("swarm-cleanup: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})
	if result["cleaned"] != true {
		t.Fatalf("cleaned = %v, want true: %s", result["cleaned"], buf.String())
	}
	if got, err := os.ReadFile(resultPath); err != nil {
		t.Fatalf("swarm-cleanup removed durable result history: %v", err)
	} else if string(got) != string(resultData) {
		t.Fatalf("swarm-cleanup changed durable result history:\ngot:  %s\nwant: %s", got, resultData)
	}
	for _, disposable := range []string{
		filepath.Join(swarmDir, "findings.json"),
		filepath.Join(swarmDir, "display.json"),
		responsesDir,
	} {
		if _, err := os.Stat(disposable); !os.IsNotExist(err) {
			t.Fatalf("swarm-cleanup left disposable artifact %s", disposable)
		}
	}

	// Cleaning a nonexistent swarm must not claim it cleaned anything.
	buf.Reset()
	resetRootCmd(t)
	rootCmd.SetArgs([]string{"swarm-cleanup", "--id", "never-existed"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("swarm-cleanup (absent): %v", err)
	}
	env = parseEnvelope(t, buf.String())
	result = env["result"].(map[string]interface{})
	if result["cleaned"] != false {
		t.Fatalf("cleaned = %v for a swarm that never existed, want false", result["cleaned"])
	}
}
