package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/storage"
)

func TestPromoteToHiveIgnoresDisplayNameWhenBoostingConfidence(t *testing.T) {
	// Confidence boosts count stable repository identities, never display names.
	// Four promotions of the same text from one machine and one working
	// directory are one repository, however they label themselves, so no
	// multi-repo boost is earned. This is the property that stops a single
	// colony inflating its own wisdom by renaming its directory.
	hubDir := t.TempDir()
	t.Setenv("AETHER_HUB_DIR", hubDir)
	if err := os.MkdirAll(filepath.Join(hubDir, "hive"), 0755); err != nil {
		t.Fatalf("mkdir hive: %v", err)
	}

	for _, displayName := range []string{"repo-one", "repo-two", "repo-three", "repo-four"} {
		if err := promoteToHive("Prefer shared helpers for dispatch paths in cmd/dispatch.go", "go", displayName, 0.60); err != nil {
			t.Fatalf("promote %s: %v", displayName, err)
		}
	}

	var wisdom hiveWisdomData
	data, err := os.ReadFile(filepath.Join(hubDir, "hive", "wisdom.json"))
	if err != nil {
		t.Fatalf("read wisdom: %v", err)
	}
	if err := json.Unmarshal(data, &wisdom); err != nil {
		t.Fatalf("unmarshal wisdom: %v", err)
	}
	if len(wisdom.Entries) != 1 {
		t.Fatalf("entries = %d, want 1: %#v", len(wisdom.Entries), wisdom.Entries)
	}

	entry := wisdom.Entries[0]
	if entry.Confidence != 0.60 {
		t.Errorf("confidence = %.2f, want 0.60 — four aliases of one repository must not earn a multi-repo boost", entry.Confidence)
	}
	if len(entry.SourceRepoIDs) != 1 {
		t.Errorf("source_repo_ids = %#v, want exactly 1 stable identity", entry.SourceRepoIDs)
	}
	// Display names are still recorded for human readability, just not counted.
	if len(entry.SourceRepos) != 4 {
		t.Errorf("source_repos = %#v, want all 4 display names recorded", entry.SourceRepos)
	}
}

func TestPromoteToHiveBoostsConfidenceAcrossDistinctRepos(t *testing.T) {
	// The complement of the test above: genuinely distinct repositories, proven
	// distinct by their stable identity rather than their name, do earn a boost.
	hubDir := t.TempDir()
	t.Setenv("AETHER_HUB_DIR", hubDir)
	if err := os.MkdirAll(filepath.Join(hubDir, "hive"), 0755); err != nil {
		t.Fatalf("mkdir hive: %v", err)
	}

	const text = "Prefer shared helpers for dispatch paths in cmd/dispatch.go"
	identities := map[string]bool{}

	for _, name := range []string{"alpha", "beta", "gamma", "delta"} {
		// Each promotion runs from its own directory, which is what makes the
		// repositories distinct — stableRepoIdentity falls back to a hash of the
		// absolute path when there is no git remote.
		repoDir := filepath.Join(t.TempDir(), name)
		if err := os.MkdirAll(repoDir, 0755); err != nil {
			t.Fatalf("mkdir %s: %v", name, err)
		}
		t.Chdir(repoDir)
		identities[currentRepoIdentity()] = true

		if err := promoteToHive(text, "go", name, 0.60); err != nil {
			t.Fatalf("promote %s: %v", name, err)
		}
	}

	if len(identities) != 4 {
		t.Fatalf("expected 4 distinct repo identities, got %d: %v", len(identities), identities)
	}

	var wisdom hiveWisdomData
	data, err := os.ReadFile(filepath.Join(hubDir, "hive", "wisdom.json"))
	if err != nil {
		t.Fatalf("read wisdom: %v", err)
	}
	if err := json.Unmarshal(data, &wisdom); err != nil {
		t.Fatalf("unmarshal wisdom: %v", err)
	}
	if len(wisdom.Entries) != 1 {
		t.Fatalf("entries = %d, want 1: %#v", len(wisdom.Entries), wisdom.Entries)
	}
	if got := wisdom.Entries[0].Confidence; got != 0.95 {
		t.Fatalf("confidence = %.2f, want 0.95 for four distinct repositories", got)
	}
	if len(wisdom.Entries[0].SourceRepoIDs) != 4 {
		t.Fatalf("source_repo_ids = %#v, want 4 distinct identities", wisdom.Entries[0].SourceRepoIDs)
	}
}

func TestColonyPrimeHiveWisdomFiltersByRegistryDomain(t *testing.T) {
	saveGlobals(t)

	root := t.TempDir()
	dataDir := filepath.Join(root, ".aether", "data")
	s, err := storage.NewStore(dataDir)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	store = s

	// Hive retrieval is on by default (D-01/D-02); no opt-in needed.

	hubDir := t.TempDir()
	t.Setenv("AETHER_HUB_DIR", hubDir)
	if err := os.MkdirAll(filepath.Join(hubDir, "hive"), 0755); err != nil {
		t.Fatalf("mkdir hive: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(hubDir, "registry"), 0755); err != nil {
		t.Fatalf("mkdir registry: %v", err)
	}

	wisdom := hiveWisdomData{Entries: []hiveWisdomEntry{
		{ID: "go_1", Text: "Prefer table-driven tests in Go", Domain: "go", Confidence: 0.9},
		{ID: "rails_1", Text: "Prefer Rails service objects", Domain: "rails", Confidence: 0.9},
	}}
	wisdomData, _ := json.Marshal(wisdom)
	if err := os.WriteFile(filepath.Join(hubDir, "hive", "wisdom.json"), wisdomData, 0644); err != nil {
		t.Fatalf("write wisdom: %v", err)
	}
	registryData := registryData{Colonies: []registryEntry{{RepoPath: root, Domains: []string{"go"}, Active: true}}}
	registryJSON, _ := json.Marshal(registryData)
	if err := os.WriteFile(filepath.Join(hubDir, "registry", "registry.json"), registryJSON, 0644); err != nil {
		t.Fatalf("write registry: %v", err)
	}

	goal := "domain-filtered hive wisdom"
	state := colony.ColonyState{
		Version:      "1.0",
		Goal:         &goal,
		State:        colony.StateREADY,
		CurrentPhase: 1,
		Plan: colony.Plan{Phases: []colony.Phase{{
			ID:     1,
			Name:   "Domain phase",
			Status: colony.PhaseReady,
			Tasks:  []colony.Task{{Goal: "Use Go conventions", Status: colony.TaskPending}},
		}}},
	}
	if err := s.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatalf("save state: %v", err)
	}

	output := buildColonyPrimeOutput(false)
	if !strings.Contains(output.Context, "Prefer table-driven tests in Go") {
		t.Fatalf("expected go hive wisdom in context:\n%s", output.Context)
	}
	if strings.Contains(output.Context, "Prefer Rails service objects") {
		t.Fatalf("unexpected rails hive wisdom in go-scoped context:\n%s", output.Context)
	}
}
