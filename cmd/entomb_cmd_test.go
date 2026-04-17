package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

func TestEntombCommandExists(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	cmd, _, err := rootCmd.Find([]string{"entomb"})
	if err != nil {
		t.Fatalf("entomb command not found: %v", err)
	}
	if cmd == nil {
		t.Fatal("entomb command is nil")
	}
	if cmd.Use != "entomb" {
		t.Fatalf("entomb command Use = %q, want entomb", cmd.Use)
	}
}

func TestEntombArchivesAndResetsSealedColony(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	aetherRoot := os.Getenv("AETHER_ROOT")
	if aetherRoot == "" {
		t.Fatal("AETHER_ROOT not set by setupBuildFlowTest")
	}

	var buf bytes.Buffer
	stdout = &buf

	goal := "Ship release readiness"
	taskID := "task-1"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:       "3.0",
		Goal:          &goal,
		ColonyVersion: 2,
		State:         colony.StateCOMPLETED,
		CurrentPhase:  1,
		Milestone:     "Crowned Anthill",
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{
					ID:     1,
					Name:   "Release",
					Status: colony.PhaseCompleted,
					Tasks:  []colony.Task{{ID: &taskID, Goal: "Seal the colony", Status: colony.TaskCompleted}},
				},
			},
		},
	})

	legacySessionDir := filepath.Join(aetherRoot, ".aether", "data", "colonies", "ship-release-readiness")
	if err := os.MkdirAll(legacySessionDir, 0755); err != nil {
		t.Fatalf("failed to create legacy session dir: %v", err)
	}
	legacySession := colony.SessionFile{
		SessionID:     "legacy-session",
		ColonyGoal:    goal,
		LastCommand:   "seal",
		SuggestedNext: "aether entomb",
		Summary:       "Ready to archive",
	}
	legacyData, err := json.MarshalIndent(legacySession, "", "  ")
	if err != nil {
		t.Fatalf("marshal legacy session: %v", err)
	}
	if err := os.WriteFile(filepath.Join(legacySessionDir, "session.json"), append(legacyData, '\n'), 0644); err != nil {
		t.Fatalf("write legacy session: %v", err)
	}

	if err := os.MkdirAll(filepath.Join(aetherRoot, ".aether", "exchange"), 0755); err != nil {
		t.Fatalf("create exchange dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(aetherRoot, ".aether", "dreams"), 0755); err != nil {
		t.Fatalf("create dreams dir: %v", err)
	}
	for path, content := range map[string]string{
		filepath.Join(aetherRoot, ".aether", "CROWNED-ANTHILL.md"):         "# Crowned Anthill\n",
		filepath.Join(aetherRoot, ".aether", "HANDOFF.md"):                 "# Old handoff\n",
		filepath.Join(aetherRoot, ".aether", "CONTEXT.md"):                 "# Old context\n",
		filepath.Join(aetherRoot, ".aether", "dreams", "dream.md"):         "dream\n",
		filepath.Join(aetherRoot, ".aether", "exchange", "pheromones.xml"): "<pheromones />\n",
	} {
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("write fixture %s: %v", path, err)
		}
	}

	rootCmd.SetArgs([]string{"entomb"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("entomb returned error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, `"entombed":true`) {
		t.Fatalf("expected entomb success JSON, got: %s", output)
	}

	chambersDir := filepath.Join(aetherRoot, ".aether", "chambers")
	entries, err := os.ReadDir(chambersDir)
	if err != nil {
		t.Fatalf("read chambers dir: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 chamber, got %d", len(entries))
	}
	chamberDir := filepath.Join(chambersDir, entries[0].Name())
	for _, required := range []string{
		"manifest.json",
		"COLONY_STATE.json",
		"CROWNED-ANTHILL.md",
		"colony-archive.xml",
		"session.json",
		filepath.Join("colonies", "ship-release-readiness", "session.json"),
	} {
		if _, err := os.Stat(filepath.Join(chamberDir, required)); err != nil {
			t.Fatalf("expected archived file %s: %v", required, err)
		}
	}

	var reset colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &reset); err != nil {
		t.Fatalf("reload reset state: %v", err)
	}
	if reset.State != colony.StateIDLE {
		t.Fatalf("reset state = %q, want IDLE", reset.State)
	}
	if reset.Goal != nil {
		t.Fatalf("reset goal = %v, want nil", *reset.Goal)
	}
	if reset.CurrentPhase != 0 {
		t.Fatalf("reset current phase = %d, want 0", reset.CurrentPhase)
	}
	if len(reset.Plan.Phases) != 0 {
		t.Fatalf("reset plan phases = %d, want 0", len(reset.Plan.Phases))
	}

	for _, cleared := range []string{
		filepath.Join(aetherRoot, ".aether", "CROWNED-ANTHILL.md"),
		filepath.Join(dataDir, "session.json"),
		filepath.Join(aetherRoot, ".aether", "data", "colonies"),
	} {
		if _, err := os.Stat(cleared); !os.IsNotExist(err) {
			t.Fatalf("expected %s to be cleared, stat err=%v", cleared, err)
		}
	}

	handoff, err := os.ReadFile(filepath.Join(aetherRoot, ".aether", "HANDOFF.md"))
	if err != nil {
		t.Fatalf("expected new HANDOFF.md: %v", err)
	}
	for _, want := range []string{"entombed", "aether init", "aether tunnels"} {
		if !strings.Contains(string(handoff), want) {
			t.Fatalf("HANDOFF.md missing %q\n%s", want, string(handoff))
		}
	}
}
