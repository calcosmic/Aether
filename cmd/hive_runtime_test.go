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

func TestPromoteToHiveBoostsConfidenceAcrossDistinctRepos(t *testing.T) {
	hubDir := t.TempDir()
	t.Setenv("AETHER_HUB_DIR", hubDir)
	if err := os.MkdirAll(filepath.Join(hubDir, "hive"), 0755); err != nil {
		t.Fatalf("mkdir hive: %v", err)
	}

	for _, repo := range []string{"repo-one", "repo-two", "repo-three", "repo-four"} {
		if err := promoteToHive("Prefer shared helpers for dispatch paths", "go", repo, 0.60); err != nil {
			t.Fatalf("promote %s: %v", repo, err)
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
	if wisdom.Entries[0].Confidence != 0.95 {
		t.Fatalf("confidence = %.2f, want 0.95", wisdom.Entries[0].Confidence)
	}
	if len(wisdom.Entries[0].SourceRepos) != 4 {
		t.Fatalf("source_repos = %#v, want 4 distinct repos", wisdom.Entries[0].SourceRepos)
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
