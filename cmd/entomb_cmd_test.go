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
		Scope:         colony.ScopeMeta,
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
	if !strings.Contains(entries[0].Name(), "-meta-") {
		t.Fatalf("expected scoped chamber name to include -meta-, got %q", entries[0].Name())
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
	var manifest map[string]interface{}
	manifestData, err := os.ReadFile(filepath.Join(chamberDir, "manifest.json"))
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		t.Fatalf("unmarshal manifest: %v", err)
	}
	if got := manifest["scope"]; got != "meta" {
		t.Fatalf("manifest scope = %v, want meta", got)
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
	if reset.Scope != "" {
		t.Fatalf("reset scope = %q, want empty", reset.Scope)
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

func TestEntomb_ReviewsArchive(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	aetherRoot := os.Getenv("AETHER_ROOT")
	if aetherRoot == "" {
		t.Fatal("AETHER_ROOT not set by setupBuildFlowTest")
	}

	var buf bytes.Buffer
	stdout = &buf

	goal := "Archive reviews colony"
	taskID := "task-1"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:       "3.0",
		Goal:          &goal,
		ColonyVersion: 2,
		Scope:         colony.ScopeMeta,
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

	// Create review data in the active data directory
	reviewsDir := filepath.Join(dataDir, "reviews", "security")
	if err := os.MkdirAll(reviewsDir, 0755); err != nil {
		t.Fatalf("failed to create reviews dir: %v", err)
	}
	ledger := colony.ReviewLedgerFile{
		Entries: []colony.ReviewLedgerEntry{
			{ID: "sec-1-001", Phase: 1, Agent: "gatekeeper", Status: "open", Severity: colony.ReviewSeverityHigh, Description: "Exposed secret"},
			{ID: "sec-1-002", Phase: 1, Agent: "gatekeeper", Status: "resolved", Severity: colony.ReviewSeverityMedium, Description: "Missing validation"},
		},
		Summary: colony.ComputeSummary([]colony.ReviewLedgerEntry{
			{ID: "sec-1-001", Phase: 1, Agent: "gatekeeper", Status: "open", Severity: colony.ReviewSeverityHigh, Description: "Exposed secret"},
			{ID: "sec-1-002", Phase: 1, Agent: "gatekeeper", Status: "resolved", Severity: colony.ReviewSeverityMedium, Description: "Missing validation"},
		}),
	}
	ledgerData, err := json.MarshalIndent(ledger, "", "  ")
	if err != nil {
		t.Fatalf("marshal ledger: %v", err)
	}
	if err := os.WriteFile(filepath.Join(reviewsDir, "ledger.json"), append(ledgerData, '\n'), 0644); err != nil {
		t.Fatalf("write review ledger: %v", err)
	}

	for path, content := range map[string]string{
		filepath.Join(aetherRoot, ".aether", "CROWNED-ANTHILL.md"): "# Crowned Anthill\n",
		filepath.Join(aetherRoot, ".aether", "HANDOFF.md"):         "# Old handoff\n",
		filepath.Join(aetherRoot, ".aether", "CONTEXT.md"):         "# Old context\n",
	} {
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatalf("create parent for %s: %v", path, err)
		}
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

	// Verify reviews were archived into the chamber
	chambersDir := filepath.Join(aetherRoot, ".aether", "chambers")
	entries, err := os.ReadDir(chambersDir)
	if err != nil {
		t.Fatalf("read chambers dir: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 chamber, got %d", len(entries))
	}
	chamberDir := filepath.Join(chambersDir, entries[0].Name())
	archivedLedger := filepath.Join(chamberDir, "reviews", "security", "ledger.json")
	if _, err := os.Stat(archivedLedger); err != nil {
		t.Fatalf("expected archived reviews/security/ledger.json: %v", err)
	}

	// Verify reviews were cleaned from active data
	if _, err := os.Stat(filepath.Join(dataDir, "reviews")); !os.IsNotExist(err) {
		t.Fatalf("expected reviews directory to be removed after entomb, stat err=%v", err)
	}
}

func TestEntomb_NoReviewsArchive(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	aetherRoot := os.Getenv("AETHER_ROOT")
	if aetherRoot == "" {
		t.Fatal("AETHER_ROOT not set by setupBuildFlowTest")
	}

	var buf bytes.Buffer
	stdout = &buf

	goal := "Archive colony without reviews"
	taskID := "task-1"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:       "3.0",
		Goal:          &goal,
		ColonyVersion: 2,
		Scope:         colony.ScopeProject,
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

	// No reviews directory created -- backward compatible colony

	for path, content := range map[string]string{
		filepath.Join(aetherRoot, ".aether", "CROWNED-ANTHILL.md"): "# Crowned Anthill\n",
		filepath.Join(aetherRoot, ".aether", "HANDOFF.md"):         "# Old handoff\n",
		filepath.Join(aetherRoot, ".aether", "CONTEXT.md"):         "# Old context\n",
	} {
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatalf("create parent for %s: %v", path, err)
		}
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
		t.Fatalf("expected entomb success JSON without reviews, got: %s", output)
	}

	// Verify chamber has required files (reviews not in required list)
	chambersDir := filepath.Join(aetherRoot, ".aether", "chambers")
	entries, err := os.ReadDir(chambersDir)
	if err != nil {
		t.Fatalf("read chambers dir: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 chamber, got %d", len(entries))
	}
	chamberDir := filepath.Join(chambersDir, entries[0].Name())
	for _, required := range []string{"manifest.json", "COLONY_STATE.json", "CROWNED-ANTHILL.md", "colony-archive.xml"} {
		if _, err := os.Stat(filepath.Join(chamberDir, required)); err != nil {
			t.Fatalf("expected archived file %s: %v", required, err)
		}
	}
}

func TestEntombLegacyScopeDefaultsToProject(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	aetherRoot := os.Getenv("AETHER_ROOT")
	if aetherRoot == "" {
		t.Fatal("AETHER_ROOT not set by setupBuildFlowTest")
	}

	var buf bytes.Buffer
	stdout = &buf

	goal := "Archive legacy colony"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:       "3.0",
		Goal:          &goal,
		ColonyVersion: 2,
		State:         colony.StateCOMPLETED,
		CurrentPhase:  1,
		Milestone:     "Crowned Anthill",
		Plan: colony.Plan{
			Phases: []colony.Phase{{ID: 1, Name: "Archive", Status: colony.PhaseCompleted}},
		},
	})

	for path, content := range map[string]string{
		filepath.Join(aetherRoot, ".aether", "CROWNED-ANTHILL.md"): "# Crowned Anthill\n",
		filepath.Join(aetherRoot, ".aether", "HANDOFF.md"):         "# Old handoff\n",
		filepath.Join(aetherRoot, ".aether", "CONTEXT.md"):         "# Old context\n",
	} {
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatalf("create parent for %s: %v", path, err)
		}
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("write fixture %s: %v", path, err)
		}
	}

	rootCmd.SetArgs([]string{"entomb"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("entomb returned error: %v", err)
	}

	chambersDir := filepath.Join(aetherRoot, ".aether", "chambers")
	entries, err := os.ReadDir(chambersDir)
	if err != nil {
		t.Fatalf("read chambers dir: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 chamber, got %d", len(entries))
	}
	if !strings.Contains(entries[0].Name(), "-project-") {
		t.Fatalf("expected legacy chamber name to include -project-, got %q", entries[0].Name())
	}

	manifestData, err := os.ReadFile(filepath.Join(chambersDir, entries[0].Name(), "manifest.json"))
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	var manifest map[string]interface{}
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		t.Fatalf("unmarshal manifest: %v", err)
	}
	if got := manifest["scope"]; got != "project" {
		t.Fatalf("legacy manifest scope = %v, want project", got)
	}
}
