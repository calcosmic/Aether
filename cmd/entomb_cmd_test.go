package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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

func TestClearActiveColonyRuntimeFilesPreservesShippedExchangeXML(t *testing.T) {
	root := t.TempDir()
	dataDir := filepath.Join(root, ".aether", "data")
	exchangeDir := filepath.Join(root, ".aether", "exchange")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		t.Fatalf("create data dir: %v", err)
	}
	if err := os.MkdirAll(exchangeDir, 0755); err != nil {
		t.Fatalf("create exchange dir: %v", err)
	}

	for _, name := range sourceCheckRequiredExchangeXMLAssets {
		if err := os.WriteFile(filepath.Join(exchangeDir, name), []byte("<fixture />\n"), 0644); err != nil {
			t.Fatalf("write exchange fixture %s: %v", name, err)
		}
	}
	if err := os.WriteFile(filepath.Join(dataDir, "session.json"), []byte("{}\n"), 0644); err != nil {
		t.Fatalf("write runtime session: %v", err)
	}

	if err := clearActiveColonyRuntimeFiles(root, dataDir); err != nil {
		t.Fatalf("clear runtime files: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dataDir, "session.json")); !os.IsNotExist(err) {
		t.Fatalf("session.json should be removed, stat err=%v", err)
	}
	for _, name := range sourceCheckRequiredExchangeXMLAssets {
		if _, err := os.Stat(filepath.Join(exchangeDir, name)); err != nil {
			t.Fatalf("shipped exchange XML %s should be preserved: %v", name, err)
		}
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
	charter := colony.Charter{Intent: "ship release readiness"}
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:       "3.0",
		Goal:          &goal,
		Charter:       &charter,
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
	if reset.Charter != nil {
		t.Fatalf("reset charter = %+v, want nil", reset.Charter)
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

func TestEntombNearMissExtraction(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	aetherRoot := os.Getenv("AETHER_ROOT")
	if aetherRoot == "" {
		t.Fatal("AETHER_ROOT not set by setupBuildFlowTest")
	}

	var buf bytes.Buffer
	stdout = &buf

	goal := "Near miss colony"
	taskID := "task-1"
	instinctLow := colony.Instinct{ID: "inst-1", Trigger: "low", Action: "skip", Confidence: 0.4, Status: "active"}
	instinctMid := colony.Instinct{ID: "inst-2", Trigger: "mid", Action: "maybe", Confidence: 0.6, Status: "active"}
	instinctHigh := colony.Instinct{ID: "inst-3", Trigger: "high", Action: "definitely", Confidence: 0.9, Status: "active"}
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:       "3.0",
		Goal:          &goal,
		ColonyVersion: 2,
		Scope:         colony.ScopeProject,
		State:         colony.StateCOMPLETED,
		CurrentPhase:  1,
		Milestone:     "Crowned Anthill",
		Memory: colony.Memory{
			Instincts: []colony.Instinct{instinctLow, instinctMid, instinctHigh},
		},
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{ID: 1, Name: "Release", Status: colony.PhaseCompleted, Tasks: []colony.Task{{ID: &taskID, Goal: "Build", Status: colony.TaskCompleted}}},
			},
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

	output := buf.String()
	if !strings.Contains(output, `"entombed":true`) {
		t.Fatalf("expected entomb success JSON, got: %s", output)
	}

	// Verify near-miss file exists in chamber
	chambersDir := filepath.Join(aetherRoot, ".aether", "chambers")
	entries, err := os.ReadDir(chambersDir)
	if err != nil {
		t.Fatalf("read chambers dir: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 chamber, got %d", len(entries))
	}
	chamberDir := filepath.Join(chambersDir, entries[0].Name())

	nmData, err := os.ReadFile(filepath.Join(chamberDir, "near-miss-instincts.json"))
	if err != nil {
		t.Fatalf("expected near-miss-instincts.json in chamber: %v", err)
	}

	var nearMiss []colony.Instinct
	if err := json.Unmarshal(nmData, &nearMiss); err != nil {
		t.Fatalf("unmarshal near-miss instincts: %v", err)
	}
	if len(nearMiss) != 1 {
		t.Fatalf("expected 1 near-miss instinct, got %d", len(nearMiss))
	}
	if nearMiss[0].ID != "inst-2" {
		t.Fatalf("expected near-miss to be inst-2 (confidence 0.6), got %s", nearMiss[0].ID)
	}

	// Verify manifest has near_miss_instincts count
	manifestData, err := os.ReadFile(filepath.Join(chamberDir, "manifest.json"))
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	var manifest map[string]interface{}
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		t.Fatalf("unmarshal manifest: %v", err)
	}
	if manifest["near_miss_instincts"] != float64(1) {
		t.Fatalf("manifest near_miss_instincts = %v, want 1", manifest["near_miss_instincts"])
	}
}

func TestEntombTempSweepMidden(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	aetherRoot := os.Getenv("AETHER_ROOT")
	if aetherRoot == "" {
		t.Fatal("AETHER_ROOT not set by setupBuildFlowTest")
	}

	var buf bytes.Buffer
	stdout = &buf

	goal := "Temp sweep colony"
	taskID := "task-1"
	oldTime := time.Now().UTC().Add(-40 * 24 * time.Hour).Format(time.RFC3339)
	recentTime := time.Now().UTC().Add(-5 * 24 * time.Hour).Format(time.RFC3339)

	// Create midden with old and recent entries
	midden := colony.MiddenFile{
		Entries: []colony.MiddenEntry{
			{ID: "old-1", Timestamp: oldTime, Category: "build", Message: "old failure"},
			{ID: "recent-1", Timestamp: recentTime, Category: "build", Message: "recent failure"},
		},
	}
	middenData, _ := json.MarshalIndent(midden, "", "  ")
	if err := os.WriteFile(filepath.Join(dataDir, "midden.json"), append(middenData, '\n'), 0644); err != nil {
		t.Fatalf("write midden: %v", err)
	}

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
				{ID: 1, Name: "Release", Status: colony.PhaseCompleted, Tasks: []colony.Task{{ID: &taskID, Goal: "Build", Status: colony.TaskCompleted}}},
			},
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

	// Verify archived midden in chamber contains only the recent entry
	chambersDir := filepath.Join(aetherRoot, ".aether", "chambers")
	entries, err := os.ReadDir(chambersDir)
	if err != nil {
		t.Fatalf("read chambers dir: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 chamber, got %d", len(entries))
	}
	chamberDir := filepath.Join(chambersDir, entries[0].Name())

	var archivedMidden colony.MiddenFile
	archivedData, err := os.ReadFile(filepath.Join(chamberDir, "midden.json"))
	if err != nil {
		t.Fatalf("read archived midden: %v", err)
	}
	if err := json.Unmarshal(archivedData, &archivedMidden); err != nil {
		t.Fatalf("unmarshal archived midden: %v", err)
	}
	// The temp sweep runs after copy, so the archived copy should have both entries
	// (temp sweep cleans the active data, not the chamber copy)
	if len(archivedMidden.Entries) != 2 {
		t.Fatalf("expected 2 entries in archived midden (both preserved before sweep), got %d", len(archivedMidden.Entries))
	}
}

func TestEntombTempSweepExpiredPheromones(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	aetherRoot := os.Getenv("AETHER_ROOT")
	if aetherRoot == "" {
		t.Fatal("AETHER_ROOT not set by setupBuildFlowTest")
	}

	var buf bytes.Buffer
	stdout = &buf

	goal := "Pheromone sweep colony"
	taskID := "task-1"

	strengthZero := 0.0
	strengthHigh := 0.8
	// Create pheromones with expired and active signals
	pheromones := colony.PheromoneFile{
		Signals: []colony.PheromoneSignal{
			{ID: "expired-1", Type: "FOCUS", Active: false, Strength: &strengthZero, CreatedAt: "2026-01-01T00:00:00Z"},
			{ID: "active-1", Type: "REDIRECT", Active: true, Strength: &strengthHigh, CreatedAt: "2026-04-01T00:00:00Z"},
		},
	}
	phData, _ := json.MarshalIndent(pheromones, "", "  ")
	if err := os.WriteFile(filepath.Join(dataDir, "pheromones.json"), append(phData, '\n'), 0644); err != nil {
		t.Fatalf("write pheromones: %v", err)
	}

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
				{ID: 1, Name: "Release", Status: colony.PhaseCompleted, Tasks: []colony.Task{{ID: &taskID, Goal: "Build", Status: colony.TaskCompleted}}},
			},
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

	// Verify archived pheromones in chamber contain both signals (copied before sweep)
	chambersDir := filepath.Join(aetherRoot, ".aether", "chambers")
	entries, err := os.ReadDir(chambersDir)
	if err != nil {
		t.Fatalf("read chambers dir: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 chamber, got %d", len(entries))
	}
	chamberDir := filepath.Join(chambersDir, entries[0].Name())

	var archivedPh colony.PheromoneFile
	archivedData, err := os.ReadFile(filepath.Join(chamberDir, "pheromones.json"))
	if err != nil {
		t.Fatalf("read archived pheromones: %v", err)
	}
	if err := json.Unmarshal(archivedData, &archivedPh); err != nil {
		t.Fatalf("unmarshal archived pheromones: %v", err)
	}
	if len(archivedPh.Signals) != 2 {
		t.Fatalf("expected 2 signals in archived pheromones (both preserved before sweep), got %d", len(archivedPh.Signals))
	}
}

func TestEntombRegistryFinalStats(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	aetherRoot := os.Getenv("AETHER_ROOT")
	if aetherRoot == "" {
		t.Fatal("AETHER_ROOT not set by setupBuildFlowTest")
	}

	var buf bytes.Buffer
	stdout = &buf

	goal := "Registry stats colony"
	taskID := "task-1"

	// Set up registry in the hub path
	t.Setenv("AETHER_HUB_DIR", filepath.Join(t.TempDir(), ".aether"))
	hub := resolveHubPath()
	registryDir := filepath.Join(hub, "registry")
	if err := os.MkdirAll(registryDir, 0755); err != nil {
		t.Fatalf("create registry dir: %v", err)
	}

	repoPath, _ := os.Getwd()
	rd := registryData{
		Colonies: []registryEntry{
			{RepoPath: repoPath, Domains: []string{"test"}, Active: true, RegisteredAt: "2026-04-01T00:00:00Z", LastGoal: goal},
		},
	}
	regData, _ := json.MarshalIndent(rd, "", "  ")
	if err := os.WriteFile(filepath.Join(registryDir, "registry.json"), append(regData, '\n'), 0644); err != nil {
		t.Fatalf("write registry: %v", err)
	}

	initTime := "2026-04-01T00:00:00Z"
	parsedTime, _ := time.Parse(time.RFC3339, initTime)

	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:       "3.0",
		Goal:          &goal,
		ColonyVersion: 2,
		Scope:         colony.ScopeProject,
		State:         colony.StateCOMPLETED,
		CurrentPhase:  1,
		Milestone:     "Crowned Anthill",
		InitializedAt: &parsedTime,
		Memory: colony.Memory{
			PhaseLearnings: []colony.PhaseLearning{
				{Phase: 1, Learnings: []colony.Learning{{Claim: "learned something"}}},
			},
			Instincts: []colony.Instinct{
				{ID: "inst-1", Confidence: 0.8, Status: "active"},
			},
		},
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{ID: 1, Name: "Release", Status: colony.PhaseCompleted, Tasks: []colony.Task{{ID: &taskID, Goal: "Build", Status: colony.TaskCompleted}}},
			},
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

	// Verify registry entry was updated
	regResult, err := os.ReadFile(filepath.Join(registryDir, "registry.json"))
	if err != nil {
		t.Fatalf("read registry: %v", err)
	}
	var updatedRD registryData
	if err := json.Unmarshal(regResult, &updatedRD); err != nil {
		t.Fatalf("unmarshal registry: %v", err)
	}
	if len(updatedRD.Colonies) != 1 {
		t.Fatalf("expected 1 colony in registry, got %d", len(updatedRD.Colonies))
	}
	entry := updatedRD.Colonies[0]
	if entry.Active {
		t.Fatalf("expected Active=false after entomb, got true")
	}
	if entry.FinalStats == nil {
		t.Fatal("expected FinalStats to be set after entomb")
	}
	if entry.FinalStats.PhaseCount != 1 {
		t.Fatalf("FinalStats.PhaseCount = %d, want 1", entry.FinalStats.PhaseCount)
	}
	if entry.FinalStats.LearningCount != 1 {
		t.Fatalf("FinalStats.LearningCount = %d, want 1", entry.FinalStats.LearningCount)
	}
	if entry.FinalStats.InstinctCount != 1 {
		t.Fatalf("FinalStats.InstinctCount = %d, want 1", entry.FinalStats.InstinctCount)
	}
	if entry.FinalStats.SealDate == "" {
		t.Fatal("FinalStats.SealDate should not be empty")
	}
	if entry.FinalStats.Duration == "" {
		t.Fatal("FinalStats.Duration should not be empty")
	}
}

func TestEntombRegistryNoEntry(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	aetherRoot := os.Getenv("AETHER_ROOT")
	if aetherRoot == "" {
		t.Fatal("AETHER_ROOT not set by setupBuildFlowTest")
	}

	var buf bytes.Buffer
	stdout = &buf

	goal := "No registry colony"
	taskID := "task-1"

	// Ensure no registry file exists
	hub := resolveHubPath()
	registryPath := filepath.Join(hub, "registry", "registry.json")
	os.Remove(registryPath)

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
				{ID: 1, Name: "Release", Status: colony.PhaseCompleted, Tasks: []colony.Task{{ID: &taskID, Goal: "Build", Status: colony.TaskCompleted}}},
			},
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
		t.Fatalf("entomb should succeed even without registry entry, got error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, `"entombed":true`) {
		t.Fatalf("expected entomb success JSON, got: %s", output)
	}
}

func TestNearMissSuggestionOutput(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	aetherRoot := os.Getenv("AETHER_ROOT")
	if aetherRoot == "" {
		t.Fatal("AETHER_ROOT not set by setupBuildFlowTest")
	}

	var buf bytes.Buffer
	stdout = &buf

	goal := "Suggestion colony"
	taskID := "task-1"
	instinctMid := colony.Instinct{ID: "inst-1", Trigger: "mid", Action: "maybe", Confidence: 0.6, Status: "active"}
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:       "3.0",
		Goal:          &goal,
		ColonyVersion: 2,
		Scope:         colony.ScopeProject,
		State:         colony.StateCOMPLETED,
		CurrentPhase:  1,
		Milestone:     "Crowned Anthill",
		Memory: colony.Memory{
			Instincts: []colony.Instinct{instinctMid},
		},
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{ID: 1, Name: "Release", Status: colony.PhaseCompleted, Tasks: []colony.Task{{ID: &taskID, Goal: "Build", Status: colony.TaskCompleted}}},
			},
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

	output := buf.String()
	if !strings.Contains(output, "eligible for hive promotion") {
		t.Fatalf("expected suggestion about hive promotion in output, got: %s", output)
	}
	if !strings.Contains(output, "1 instincts eligible") {
		t.Fatalf("expected '1 instincts eligible' in output, got: %s", output)
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
