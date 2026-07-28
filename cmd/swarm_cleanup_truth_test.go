package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// swarm-cleanup used to load a file "to check existence", note in a comment
// that the store cannot delete, and then report cleaned:true having removed
// nothing. This test binds the report to the filesystem.
func TestSwarmCleanupReportMatchesFilesystem(t *testing.T) {
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
	if err := os.WriteFile(filepath.Join(swarmDir, "findings.json"), []byte(`{"findings":[]}`), 0644); err != nil {
		t.Fatalf("write findings: %v", err)
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
	if _, err := os.Stat(swarmDir); !os.IsNotExist(err) {
		t.Fatal("swarm-cleanup reported cleaned:true but the directory still exists")
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
