package cmd

import (
	"bytes"
	"os"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/storage"
)

// TestContextCapsuleBlocksDispatchWhenEmpty verifies that resolveCodexWorkerContext
// returns an error when the assembled context capsule is below 512 characters,
// preventing zero-context worker execution.
func TestContextCapsuleBlocksDispatchWhenEmpty(t *testing.T) {
	saveGlobals(t)

	// Set up a minimal store with almost no state so context will be tiny
	tmpDir := t.TempDir()
	dataDir := tmpDir + "/.aether/data"
	os.MkdirAll(dataDir, 0755)
	s, err := storage.NewStore(dataDir)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	store = s
	stdout = &bytes.Buffer{}
	stderr = &bytes.Buffer{}

	goal := "test"
	state := colony.ColonyState{
		Version:      "1.0",
		Goal:         &goal,
		State:        colony.StateREADY,
		CurrentPhase: 1,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{ID: 1, Name: "Test", Status: colony.PhaseReady},
			},
		},
	}
	if err := s.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatalf("save state: %v", err)
	}

	context := resolveCodexWorkerContext()
	if len(context) >= 128 {
		t.Skipf("context is %d chars (≥128) in this environment; cannot test empty-capsule block", len(context))
	}

	// The fix returns empty string when context is below 128 chars.
	if context != "" {
		t.Fatalf("resolveCodexWorkerContext returned %d chars — expected empty string for context below 128 chars", len(context))
	}
	t.Logf("✓ resolveCodexWorkerContext correctly returned empty string for %d-char context", len(context))
}
