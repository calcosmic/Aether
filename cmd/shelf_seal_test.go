package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/storage"
)

func TestDetectExpiredFocus(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	tmpDir := t.TempDir()
	dataDir := tmpDir + "/.aether/data"
	os.MkdirAll(dataDir, 0755)
	s, _ := storage.NewStore(dataDir)
	store = s

	os.Setenv("AETHER_ROOT", tmpDir)
	defer os.Setenv("AETHER_ROOT", os.Getenv("AETHER_ROOT"))

	// Create expired FOCUS pheromone
	archivedAt := time.Now().UTC().Add(-1 * time.Hour).Format(time.RFC3339)
	pf := colony.PheromoneFile{
		Signals: []colony.PheromoneSignal{
			{
				ID:         "sig_1",
				Type:       "FOCUS",
				Content:    json.RawMessage(`{"text":"focus on tests"}`),
				CreatedAt:  time.Now().UTC().Add(-2 * time.Hour).Format(time.RFC3339),
				Active:     false,
				ArchivedAt: &archivedAt,
			},
		},
	}
	s.SaveJSON("pheromones.json", pf)

	var state colony.ColonyState
	candidates, err := detectShelfCandidates(state, s)
	if err != nil {
		t.Fatalf("detectShelfCandidates failed: %v", err)
	}
	if len(candidates) != 1 {
		t.Fatalf("expected 1 candidate, got %d", len(candidates))
	}
	if candidates[0].Category != colony.ShelfCategoryPheromone {
		t.Errorf("category = %v, want pheromone", candidates[0].Category)
	}
}

func TestDetectLowConfidenceInstinct(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	tmpDir := t.TempDir()
	dataDir := tmpDir + "/.aether/data"
	os.MkdirAll(dataDir, 0755)
	s, _ := storage.NewStore(dataDir)
	store = s

	os.Setenv("AETHER_ROOT", tmpDir)
	defer os.Setenv("AETHER_ROOT", os.Getenv("AETHER_ROOT"))

	state := colony.ColonyState{
		Memory: colony.Memory{
			Instincts: []colony.Instinct{
				{
					ID:         "inst_1",
					Trigger:    "global mutable state",
					Action:     "avoid in workers",
					Confidence: 0.6,
					CreatedAt:  time.Now().UTC().Format(time.RFC3339),
				},
			},
		},
	}

	candidates, err := detectShelfCandidates(state, s)
	if err != nil {
		t.Fatalf("detectShelfCandidates failed: %v", err)
	}
	if len(candidates) != 1 {
		t.Fatalf("expected 1 candidate, got %d", len(candidates))
	}
	if candidates[0].Category != colony.ShelfCategoryInstinct {
		t.Errorf("category = %v, want instinct", candidates[0].Category)
	}
}

func TestDetectUnresolvedFlag(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	tmpDir := t.TempDir()
	dataDir := tmpDir + "/.aether/data"
	os.MkdirAll(dataDir, 0755)
	s, _ := storage.NewStore(dataDir)
	store = s

	os.Setenv("AETHER_ROOT", tmpDir)
	defer os.Setenv("AETHER_ROOT", os.Getenv("AETHER_ROOT"))

	ff := colony.FlagsFile{
		Decisions: []colony.FlagEntry{
			{
				ID:          "flag_1",
				Type:        "blocker",
				Description: "Unresolved blocker",
				CreatedAt:   time.Now().UTC().Format(time.RFC3339),
				Resolved:    false,
			},
		},
	}
	s.SaveJSON("pending-decisions.json", ff)

	var state colony.ColonyState
	candidates, err := detectShelfCandidates(state, s)
	if err != nil {
		t.Fatalf("detectShelfCandidates failed: %v", err)
	}
	if len(candidates) != 1 {
		t.Fatalf("expected 1 candidate, got %d", len(candidates))
	}
	if candidates[0].Category != colony.ShelfCategoryUserNote {
		t.Errorf("category = %v, want user-note", candidates[0].Category)
	}
}

func TestDetectRecurringRedirect(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	tmpDir := t.TempDir()
	dataDir := tmpDir + "/.aether/data"
	os.MkdirAll(dataDir, 0755)
	s, _ := storage.NewStore(dataDir)
	store = s

	os.Setenv("AETHER_ROOT", tmpDir)
	defer os.Setenv("AETHER_ROOT", os.Getenv("AETHER_ROOT"))

	contentHash := "sha256:abc123"
	phase1 := 1
	phase2 := 2
	pf := colony.PheromoneFile{
		Signals: []colony.PheromoneSignal{
			{
				ID:          "sig_1",
				Type:        "REDIRECT",
				Content:     json.RawMessage(`{"text":"avoid global vars"}`),
				ContentHash: &contentHash,
				CreatedAt:   time.Now().UTC().Add(-2 * time.Hour).Format(time.RFC3339),
				SourcePhase: &phase1,
				Active:      true,
			},
			{
				ID:          "sig_2",
				Type:        "REDIRECT",
				Content:     json.RawMessage(`{"text":"avoid global vars"}`),
				ContentHash: &contentHash,
				CreatedAt:   time.Now().UTC().Add(-1 * time.Hour).Format(time.RFC3339),
				SourcePhase: &phase2,
				Active:      true,
			},
		},
	}
	s.SaveJSON("pheromones.json", pf)

	var state colony.ColonyState
	candidates, err := detectShelfCandidates(state, s)
	if err != nil {
		t.Fatalf("detectShelfCandidates failed: %v", err)
	}
	if len(candidates) != 1 {
		t.Fatalf("expected 1 candidate, got %d", len(candidates))
	}
	if candidates[0].Category != colony.ShelfCategoryRedirect {
		t.Errorf("category = %v, want redirect", candidates[0].Category)
	}
	if !strings.Contains(candidates[0].Text, "avoid global vars") {
		t.Errorf("text = %v, want 'avoid global vars'", candidates[0].Text)
	}
}

func TestDetectNoCandidates(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	tmpDir := t.TempDir()
	dataDir := tmpDir + "/.aether/data"
	os.MkdirAll(dataDir, 0755)
	s, _ := storage.NewStore(dataDir)
	store = s

	os.Setenv("AETHER_ROOT", tmpDir)
	defer os.Setenv("AETHER_ROOT", os.Getenv("AETHER_ROOT"))

	var state colony.ColonyState
	candidates, err := detectShelfCandidates(state, s)
	if err != nil {
		t.Fatalf("detectShelfCandidates failed: %v", err)
	}
	if len(candidates) != 0 {
		t.Fatalf("expected 0 candidates, got %d", len(candidates))
	}
}

func TestDetectDeduplicates(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	tmpDir := t.TempDir()
	dataDir := tmpDir + "/.aether/data"
	os.MkdirAll(dataDir, 0755)
	s, _ := storage.NewStore(dataDir)
	store = s

	os.Setenv("AETHER_ROOT", tmpDir)
	defer os.Setenv("AETHER_ROOT", os.Getenv("AETHER_ROOT"))

	// Two unresolved flags with same description
	ff := colony.FlagsFile{
		Decisions: []colony.FlagEntry{
			{
				ID:          "flag_1",
				Type:        "blocker",
				Description: "duplicate text",
				CreatedAt:   time.Now().UTC().Add(-2 * time.Hour).Format(time.RFC3339),
				Resolved:    false,
			},
			{
				ID:          "flag_2",
				Type:        "blocker",
				Description: "duplicate text",
				CreatedAt:   time.Now().UTC().Add(-1 * time.Hour).Format(time.RFC3339),
				Resolved:    false,
			},
		},
	}
	s.SaveJSON("pending-decisions.json", ff)

	var state colony.ColonyState
	candidates, err := detectShelfCandidates(state, s)
	if err != nil {
		t.Fatalf("detectShelfCandidates failed: %v", err)
	}
	if len(candidates) != 1 {
		t.Fatalf("expected 1 candidate after dedup, got %d", len(candidates))
	}
}
