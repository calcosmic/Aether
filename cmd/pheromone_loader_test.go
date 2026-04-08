package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/cache"
	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/storage"
)

// --- TDD Cycle 1: extractSignalTextsFrom with pre-loaded PheromoneFile ---

func TestExtractSignalTextsFromPreloaded(t *testing.T) {
	now := time.Now().Format(time.RFC3339)
	s0_9 := 0.9
	s0_8 := 0.8
	s0_5 := 0.5

	pf := &colony.PheromoneFile{
		Signals: []colony.PheromoneSignal{
			{ID: "s1", Type: "REDIRECT", Priority: "high", Source: "user", CreatedAt: now, Active: true, Strength: &s0_9, Content: json.RawMessage(`{"text": "Avoid globals"}`)},
			{ID: "s2", Type: "FOCUS", Priority: "normal", Source: "user", CreatedAt: now, Active: true, Strength: &s0_8, Content: json.RawMessage(`{"text": "Focus on tests"}`)},
			{ID: "s3", Type: "FEEDBACK", Priority: "low", Source: "auto", CreatedAt: now, Active: true, Strength: &s0_5, Content: json.RawMessage(`{"text": "Good progress"}`)},
			{ID: "s4", Type: "FOCUS", Priority: "normal", Source: "user", CreatedAt: now, Active: false, Strength: &s0_8, Content: json.RawMessage(`{"text": "Inactive signal"}`)},
		},
	}

	result := extractSignalTextsFrom(pf, 8)
	if len(result) != 3 {
		t.Errorf("expected 3 signals (active only), got %d", len(result))
	}

	// Verify ordering: REDIRECT first (priority 1), then FOCUS (priority 2), then FEEDBACK (priority 3)
	if result[0] != "REDIRECT: Avoid globals" {
		t.Errorf("expected first signal to be REDIRECT, got: %s", result[0])
	}
	if result[1] != "FOCUS: Focus on tests" {
		t.Errorf("expected second signal to be FOCUS, got: %s", result[1])
	}
	if result[2] != "FEEDBACK: Good progress" {
		t.Errorf("expected third signal to be FEEDBACK, got: %s", result[2])
	}
}

func TestExtractSignalTextsFromNil(t *testing.T) {
	result := extractSignalTextsFrom(nil, 8)
	if result != nil {
		t.Errorf("expected nil for nil PheromoneFile, got %v", result)
	}
}

func TestExtractSignalTextsFromEmpty(t *testing.T) {
	pf := &colony.PheromoneFile{Signals: []colony.PheromoneSignal{}}
	result := extractSignalTextsFrom(pf, 8)
	if result != nil {
		t.Errorf("expected nil for empty PheromoneFile, got %v", result)
	}
}

func TestExtractSignalTextsFromMaxSignals(t *testing.T) {
	now := time.Now().Format(time.RFC3339)
	s1 := 0.9

	pf := &colony.PheromoneFile{
		Signals: []colony.PheromoneSignal{
			{ID: "s1", Type: "REDIRECT", Priority: "high", Source: "user", CreatedAt: now, Active: true, Strength: &s1, Content: json.RawMessage(`{"text": "Redirect one"}`)},
			{ID: "s2", Type: "REDIRECT", Priority: "high", Source: "user", CreatedAt: now, Active: true, Strength: &s1, Content: json.RawMessage(`{"text": "Redirect two"}`)},
			{ID: "s3", Type: "FOCUS", Priority: "normal", Source: "user", CreatedAt: now, Active: true, Strength: &s1, Content: json.RawMessage(`{"text": "Focus one"}`)},
			{ID: "s4", Type: "FOCUS", Priority: "normal", Source: "user", CreatedAt: now, Active: true, Strength: &s1, Content: json.RawMessage(`{"text": "Focus two"}`)},
			{ID: "s5", Type: "FEEDBACK", Priority: "low", Source: "auto", CreatedAt: now, Active: true, Strength: &s1, Content: json.RawMessage(`{"text": "Feedback one"}`)},
		},
	}

	result := extractSignalTextsFrom(pf, 2)
	if len(result) != 2 {
		t.Errorf("expected 2 signals with maxSignals=2, got %d", len(result))
	}
}

// --- TDD Cycle 2: loadPheromones using global store ---

func TestLoadPheromones(t *testing.T) {
	saveGlobalsCmd(t)
	s, tmpDir := newTestStoreCmd(t)
	defer os.RemoveAll(tmpDir)
	store = s

	now := time.Now().Format(time.RFC3339)
	s1 := 0.9
	pf := colony.PheromoneFile{
		Signals: []colony.PheromoneSignal{
			{ID: "s1", Type: "FOCUS", Priority: "normal", Source: "user", CreatedAt: now, Active: true, Strength: &s1, Content: json.RawMessage(`{"text": "Focus on testing"}`)},
		},
	}
	if err := s.SaveJSON("pheromones.json", pf); err != nil {
		t.Fatal(err)
	}

	loaded := loadPheromones()
	if loaded == nil {
		t.Fatal("expected non-nil PheromoneFile, got nil")
	}
	if len(loaded.Signals) != 1 {
		t.Errorf("expected 1 signal, got %d", len(loaded.Signals))
	}
	if loaded.Signals[0].ID != "s1" {
		t.Errorf("expected signal ID 's1', got %s", loaded.Signals[0].ID)
	}
}

func TestLoadPheromonesMissing(t *testing.T) {
	saveGlobalsCmd(t)
	s, tmpDir := newTestStoreCmd(t)
	defer os.RemoveAll(tmpDir)
	store = s

	loaded := loadPheromones()
	if loaded != nil {
		t.Errorf("expected nil when pheromones.json is missing, got non-nil with %d signals", len(loaded.Signals))
	}
}

// --- TDD Cycle 3: colonyPrimeCmd uses shared pheromone load ---

func TestColonyPrimeWithPheromones(t *testing.T) {
	saveGlobalsCmd(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf
	var errBuf bytes.Buffer
	stderr = &errBuf

	s, tmpDir := newTestStoreCmd(t)
	defer os.RemoveAll(tmpDir)
	store = s

	goal := "colony prime test"
	state := colony.ColonyState{
		Version:      "1.0",
		Goal:         &goal,
		State:        colony.StateEXECUTING,
		CurrentPhase: 1,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{ID: 1, Name: "Phase One", Status: "in_progress", Tasks: []colony.Task{{Status: "in_progress", Goal: "Build feature"}}},
			},
		},
	}
	if err := s.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatal(err)
	}

	now := time.Now().Format(time.RFC3339)
	s1 := 0.9
	pf := colony.PheromoneFile{
		Signals: []colony.PheromoneSignal{
			{ID: "sig_1", Type: "FOCUS", Priority: "normal", Source: "user", CreatedAt: now, Active: true, Strength: &s1, Content: json.RawMessage(`{"text": "Focus on testing"}`)},
		},
	}
	if err := s.SaveJSON("pheromones.json", pf); err != nil {
		t.Fatal(err)
	}

	rootCmd.SetArgs([]string{"colony-prime"})
	defer rootCmd.SetArgs([]string{})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("colony-prime returned error: %v", err)
	}

	envelope := parseEnvelopeCmd(t, buf.String())
	if envelope["ok"] != true {
		t.Fatalf("expected ok=true, got %v", envelope["ok"])
	}

	result := envelope["result"].(map[string]interface{})
	contextStr := result["context"].(string)

	if !strings.Contains(contextStr, "Pheromone Signals") {
		t.Error("context missing 'Pheromone Signals' section")
	}
	if !strings.Contains(contextStr, "FOCUS") {
		t.Error("context missing FOCUS signal type")
	}
	if !strings.Contains(contextStr, "Focus on testing") {
		t.Error("context missing signal text")
	}
	if !strings.Contains(contextStr, "colony prime test") {
		t.Error("context missing goal text")
	}

	sections := result["sections"].(float64)
	if sections < 2 {
		t.Errorf("expected at least 2 sections (state + pheromones), got %f", sections)
	}
}

func TestColonyPrimeNoPheromones(t *testing.T) {
	saveGlobalsCmd(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf
	var errBuf bytes.Buffer
	stderr = &errBuf

	s, tmpDir := newTestStoreCmd(t)
	defer os.RemoveAll(tmpDir)
	store = s

	goal := "no pheromones test"
	state := colony.ColonyState{
		Version:      "1.0",
		Goal:         &goal,
		State:        colony.StateREADY,
		CurrentPhase: 1,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{ID: 1, Name: "Phase One", Status: "in_progress"},
			},
		},
	}
	if err := s.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatal(err)
	}

	rootCmd.SetArgs([]string{"colony-prime"})
	defer rootCmd.SetArgs([]string{})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("colony-prime returned error: %v", err)
	}

	envelope := parseEnvelopeCmd(t, buf.String())
	result := envelope["result"].(map[string]interface{})
	contextStr := result["context"].(string)

	if !strings.Contains(contextStr, "no pheromones test") {
		t.Error("context missing goal text")
	}
	if strings.Contains(contextStr, "Pheromone Signals") {
		t.Error("context should not contain 'Pheromone Signals' when no pheromones exist")
	}
}

// --- TDD Cycle 4: colonyPrimeCmd uses SessionCache for all loads ---

func TestColonyPrimePopulatesSessionCache(t *testing.T) {
	saveGlobalsCmd(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf
	var errBuf bytes.Buffer
	stderr = &errBuf

	s, tmpDir := newTestStoreCmd(t)
	defer os.RemoveAll(tmpDir)
	store = s

	goal := "cache wiring test"
	state := colony.ColonyState{
		Version:      "1.0",
		Goal:         &goal,
		State:        colony.StateEXECUTING,
		CurrentPhase: 1,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{ID: 1, Name: "Phase One", Status: "in_progress", Tasks: []colony.Task{{Status: "in_progress", Goal: "Build feature"}}},
			},
		},
	}
	if err := s.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatal(err)
	}

	now := time.Now().Format(time.RFC3339)
	s1 := 0.9
	pf := colony.PheromoneFile{
		Signals: []colony.PheromoneSignal{
			{ID: "sig_1", Type: "FOCUS", Priority: "normal", Source: "user", CreatedAt: now, Active: true, Strength: &s1, Content: json.RawMessage(`{"text": "Focus on caching"}`)},
		},
	}
	if err := s.SaveJSON("pheromones.json", pf); err != nil {
		t.Fatal(err)
	}

	instFile := colony.InstinctsFile{
		Instincts: []colony.InstinctEntry{
			{Trigger: "error in test", Action: "fix the test", Confidence: 0.9},
		},
	}
	if err := s.SaveJSON("instincts.json", instFile); err != nil {
		t.Fatal(err)
	}

	rootCmd.SetArgs([]string{"colony-prime"})
	defer rootCmd.SetArgs([]string{})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("colony-prime returned error: %v", err)
	}

	envelope := parseEnvelopeCmd(t, buf.String())
	if envelope["ok"] != true {
		t.Fatalf("expected ok=true, got %v", envelope["ok"])
	}

	result := envelope["result"].(map[string]interface{})
	contextStr := result["context"].(string)

	// Verify all three data sources appear in output
	if !strings.Contains(contextStr, "cache wiring test") {
		t.Error("context missing goal from COLONY_STATE.json")
	}
	if !strings.Contains(contextStr, "Pheromone Signals") {
		t.Error("context missing 'Pheromone Signals' section from pheromones.json")
	}
	if !strings.Contains(contextStr, "Active Instincts") {
		t.Error("context missing 'Active Instincts' section from instincts.json")
	}
	if !strings.Contains(contextStr, "Focus on caching") {
		t.Error("context missing pheromone signal text")
	}
	if !strings.Contains(contextStr, "fix the test") {
		t.Error("context missing instinct action text")
	}

	// Verify cache wrote .cache_* files to disk (proving cache.Load was exercised)
	sections := result["sections"].(float64)
	if sections < 3 {
		t.Errorf("expected at least 3 sections (state + pheromones + instincts), got %f", sections)
	}

	cacheStatePath := filepath.Join(s.BasePath(), ".cache_COLONY_STATE.json")
	cachePheromonesPath := filepath.Join(s.BasePath(), ".cache_pheromones.json")
	cacheInstinctsPath := filepath.Join(s.BasePath(), ".cache_instincts.json")

	if _, err := os.Stat(cacheStatePath); err != nil {
		t.Errorf("expected .cache_COLONY_STATE.json on disk, got: %v", err)
	}
	if _, err := os.Stat(cachePheromonesPath); err != nil {
		t.Errorf("expected .cache_pheromones.json on disk, got: %v", err)
	}
	if _, err := os.Stat(cacheInstinctsPath); err != nil {
		t.Errorf("expected .cache_instincts.json on disk, got: %v", err)
	}
}

func TestColonyPrimeWithInstincts(t *testing.T) {
	saveGlobalsCmd(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf
	var errBuf bytes.Buffer
	stderr = &errBuf

	s, tmpDir := newTestStoreCmd(t)
	defer os.RemoveAll(tmpDir)
	store = s

	goal := "instincts cache test"
	state := colony.ColonyState{
		Version:      "1.0",
		Goal:         &goal,
		State:        colony.StateEXECUTING,
		CurrentPhase: 1,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{ID: 1, Name: "Phase One", Status: "in_progress"},
			},
		},
	}
	if err := s.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatal(err)
	}

	instFile := colony.InstinctsFile{
		Instincts: []colony.InstinctEntry{
			{Trigger: "nil pointer", Action: "add nil check", Confidence: 0.85},
		},
	}
	if err := s.SaveJSON("instincts.json", instFile); err != nil {
		t.Fatal(err)
	}

	rootCmd.SetArgs([]string{"colony-prime"})
	defer rootCmd.SetArgs([]string{})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("colony-prime returned error: %v", err)
	}

	envelope := parseEnvelopeCmd(t, buf.String())
	result := envelope["result"].(map[string]interface{})
	contextStr := result["context"].(string)

	if !strings.Contains(contextStr, "Active Instincts") {
		t.Error("context missing 'Active Instincts' section")
	}
	if !strings.Contains(contextStr, "nil pointer") {
		t.Error("context missing instinct trigger")
	}
	if !strings.Contains(contextStr, "add nil check") {
		t.Error("context missing instinct action")
	}
}

// --- Original companion tests (5.2) ---

func TestLoadPheromonesOnce_NilCache_MissingFile(t *testing.T) {
	saveGlobals(t)
	tmpDir := t.TempDir()
	dataDir := tmpDir + "/data"
	os.MkdirAll(dataDir, 0755)

	s, err := storage.NewStore(dataDir)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	_, err = loadPheromonesOnce(s, nil)
	if err == nil {
		t.Fatal("expected error for missing pheromones.json, got nil")
	}
}

func TestLoadPheromonesOnce_NilCache_LoadsFromDisk(t *testing.T) {
	saveGlobals(t)
	tmpDir := t.TempDir()
	dataDir := tmpDir + "/data"
	os.MkdirAll(dataDir, 0755)

	s, err := storage.NewStore(dataDir)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	expected := colony.PheromoneFile{
		Signals: []colony.PheromoneSignal{
			{Type: "FOCUS", Active: true, Content: json.RawMessage(`"test focus"`)},
		},
	}
	raw, _ := json.Marshal(expected)
	os.WriteFile(dataDir+"/pheromones.json", raw, 0644)

	pf, err := loadPheromonesOnce(s, nil)
	if err != nil {
		t.Fatalf("loadPheromonesOnce: %v", err)
	}
	if len(pf.Signals) != 1 {
		t.Fatalf("expected 1 signal, got %d", len(pf.Signals))
	}
	if pf.Signals[0].Type != "FOCUS" {
		t.Errorf("signal type = %q, want FOCUS", pf.Signals[0].Type)
	}
}

func TestLoadPheromonesOnce_CacheHit(t *testing.T) {
	saveGlobals(t)
	tmpDir := t.TempDir()
	dataDir := tmpDir + "/data"
	os.MkdirAll(dataDir, 0755)

	s, err := storage.NewStore(dataDir)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	expected := colony.PheromoneFile{
		Signals: []colony.PheromoneSignal{
			{Type: "REDIRECT", Active: true, Content: json.RawMessage(`"no globals"`)},
		},
	}
	raw, _ := json.Marshal(expected)
	pheromonesPath := dataDir + "/pheromones.json"
	os.WriteFile(pheromonesPath, raw, 0644)

	c := cache.NewSessionCache(dataDir)

	// Pre-populate cache so we get a hit
	if err := c.Set(pheromonesPath, expected); err != nil {
		t.Fatalf("cache.Set: %v", err)
	}

	pf, err := loadPheromonesOnce(s, c)
	if err != nil {
		t.Fatalf("loadPheromonesOnce: %v", err)
	}
	if len(pf.Signals) != 1 {
		t.Fatalf("expected 1 signal, got %d", len(pf.Signals))
	}
	if pf.Signals[0].Type != "REDIRECT" {
		t.Errorf("signal type = %q, want REDIRECT", pf.Signals[0].Type)
	}
}

func TestLoadPheromonesOnce_CacheMiss_LoadsAndStores(t *testing.T) {
	saveGlobals(t)
	tmpDir := t.TempDir()
	dataDir := tmpDir + "/data"
	os.MkdirAll(dataDir, 0755)

	s, err := storage.NewStore(dataDir)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	expected := colony.PheromoneFile{
		Signals: []colony.PheromoneSignal{
			{Type: "FEEDBACK", Active: true, Content: json.RawMessage(`"use table-driven"`)},
		},
	}
	raw, _ := json.Marshal(expected)
	pheromonesPath := dataDir + "/pheromones.json"
	os.WriteFile(pheromonesPath, raw, 0644)

	c := cache.NewSessionCache(dataDir)

	// First call: cache miss, loads from disk
	pf, err := loadPheromonesOnce(s, c)
	if err != nil {
		t.Fatalf("loadPheromonesOnce (first): %v", err)
	}
	if len(pf.Signals) != 1 {
		t.Fatalf("expected 1 signal, got %d", len(pf.Signals))
	}
	if pf.Signals[0].Type != "FEEDBACK" {
		t.Errorf("signal type = %q, want FEEDBACK", pf.Signals[0].Type)
	}

	// Verify cache now has the entry
	fullPath := dataDir + "/pheromones.json"
	_, ok := c.Get(fullPath)
	if !ok {
		t.Fatal("expected cache to contain pheromones.json after load, but it does not")
	}
}
