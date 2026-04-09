package cmd

import (
	"strings"
	"testing"
)

func TestPheromoneSnapshotInjectIsDeferred(t *testing.T) {
	cmd, _, err := rootCmd.Find([]string{"pheromone-snapshot-inject"})
	if err != nil {
		t.Fatalf("command not found: %v", err)
	}
	if cmd.Deprecated == "" {
		t.Error("pheromone-snapshot-inject should have Deprecated field set indicating it is a deferred placeholder")
	}
	if !strings.Contains(cmd.Deprecated, "deferred") {
		t.Errorf("Deprecated message should mention 'deferred', got: %s", cmd.Deprecated)
	}
}

func TestPheromoneMergeBackIsDeferred(t *testing.T) {
	cmd, _, err := rootCmd.Find([]string{"pheromone-merge-back"})
	if err != nil {
		t.Fatalf("command not found: %v", err)
	}
	if cmd.Deprecated == "" {
		t.Error("pheromone-merge-back should have Deprecated field set indicating it is a deferred placeholder")
	}
	if !strings.Contains(cmd.Deprecated, "deferred") {
		t.Errorf("Deprecated message should mention 'deferred', got: %s", cmd.Deprecated)
	}
}

func TestPheromoneSnapshotInjectShortMentionsDeferred(t *testing.T) {
	cmd, _, err := rootCmd.Find([]string{"pheromone-snapshot-inject"})
	if err != nil {
		t.Fatalf("command not found: %v", err)
	}
	if !strings.Contains(strings.ToLower(cmd.Short), "deferred") {
		t.Errorf("Short description should mention 'deferred', got: %s", cmd.Short)
	}
}

func TestPheromoneMergeBackShortMentionsDeferred(t *testing.T) {
	cmd, _, err := rootCmd.Find([]string{"pheromone-merge-back"})
	if err != nil {
		t.Fatalf("command not found: %v", err)
	}
	if !strings.Contains(strings.ToLower(cmd.Short), "deferred") {
		t.Errorf("Short description should mention 'deferred', got: %s", cmd.Short)
	}
}
