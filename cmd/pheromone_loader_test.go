package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
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

func TestExtractSignalTextsFromSkipsExpiredSignals(t *testing.T) {
	now := time.Now().UTC()
	expiredAt := now.Add(-1 * time.Hour).Format(time.RFC3339)
	futureAt := now.Add(1 * time.Hour).Format(time.RFC3339)
	createdAt := now.Add(-24 * time.Hour).Format(time.RFC3339)
	s1 := 0.9

	pf := &colony.PheromoneFile{
		Signals: []colony.PheromoneSignal{
			{
				ID:        "expired",
				Type:      "REDIRECT",
				Priority:  "high",
				Source:    "user",
				CreatedAt: createdAt,
				ExpiresAt: &expiredAt,
				Active:    true,
				Strength:  &s1,
				Content:   json.RawMessage(`{"text": "Expired constraint"}`),
			},
			{
				ID:        "live",
				Type:      "FOCUS",
				Priority:  "normal",
				Source:    "user",
				CreatedAt: createdAt,
				ExpiresAt: &futureAt,
				Active:    true,
				Strength:  &s1,
				Content:   json.RawMessage(`{"text": "Still active"}`),
			},
		},
	}

	result := extractSignalTextsFrom(pf, 8)
	if len(result) != 1 {
		t.Fatalf("expected 1 non-expired signal, got %d (%v)", len(result), result)
	}
	if result[0] != "FOCUS: Still active" {
		t.Fatalf("unexpected remaining signal: %v", result)
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

// --- TDD Cycle 5: colony-prime includes hive wisdom ---

func TestColonyPrimeWithHiveWisdom(t *testing.T) {
	saveGlobalsCmd(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf
	var errBuf bytes.Buffer
	stderr = &errBuf

	s, tmpDir := newTestStoreCmd(t)
	defer os.RemoveAll(tmpDir)
	store = s

	// Hive retrieval is on by default (D-01/D-02); no opt-in needed.

	// Set up hub directory with hive wisdom
	hubDir := t.TempDir()
	os.MkdirAll(filepath.Join(hubDir, "hive"), 0755)
	os.MkdirAll(filepath.Join(hubDir, "eternal"), 0755)

	wisdomData := `{"entries":[{"id":"go_abc123","text":"Prefer table-driven tests in Go","domain":"go","source_repo":"test-repo","confidence":0.85,"created_at":"2026-04-01T00:00:00Z","accessed_at":"2026-04-01T00:00:00Z","access_count":1},{"id":"go_def456","text":"Keep functions under 50 lines","domain":"go","source_repo":"test-repo","confidence":0.90,"created_at":"2026-04-01T00:00:00Z","accessed_at":"2026-04-01T00:00:00Z","access_count":2}]}`
	if err := os.WriteFile(filepath.Join(hubDir, "hive", "wisdom.json"), []byte(wisdomData), 0644); err != nil {
		t.Fatal(err)
	}

	origHubDir := os.Getenv("AETHER_HUB_DIR")
	os.Setenv("AETHER_HUB_DIR", hubDir)
	defer os.Setenv("AETHER_HUB_DIR", origHubDir)

	goal := "hive wisdom test"
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

	if !strings.Contains(contextStr, "HIVE WISDOM") {
		t.Error("context missing 'HIVE WISDOM' section")
	}
	if !strings.Contains(contextStr, "Prefer table-driven tests in Go") {
		t.Error("context missing hive wisdom entry text")
	}
	if !strings.Contains(contextStr, "Keep functions under 50 lines") {
		t.Error("context missing second hive wisdom entry text")
	}
}

// --- TDD Cycle 6: colony-prime includes user preferences ---

func TestColonyPrimeWithUserPreferences(t *testing.T) {
	saveGlobalsCmd(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf
	var errBuf bytes.Buffer
	stderr = &errBuf

	s, tmpDir := newTestStoreCmd(t)
	defer os.RemoveAll(tmpDir)
	store = s

	// Set up hub directory with QUEEN.md containing user preferences
	hubDir := t.TempDir()
	queenContent := `# QUEEN.md

## Wisdom
Prefer simplicity.

## User Preferences
- Explain things in plain English
- Prefer composition over inheritance
`
	if err := os.WriteFile(filepath.Join(hubDir, "QUEEN.md"), []byte(queenContent), 0644); err != nil {
		t.Fatal(err)
	}

	origHubDir := os.Getenv("AETHER_HUB_DIR")
	os.Setenv("AETHER_HUB_DIR", hubDir)
	defer os.Setenv("AETHER_HUB_DIR", origHubDir)

	goal := "user prefs test"
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

	if !strings.Contains(contextStr, "USER PREFERENCES") {
		t.Error("context missing 'USER PREFERENCES' section")
	}
	if !strings.Contains(contextStr, "Explain things in plain English") {
		t.Error("context missing user preference text")
	}
	if !strings.Contains(contextStr, "Prefer composition over inheritance") {
		t.Error("context missing second user preference text")
	}
}

// --- TDD Cycle 7: colony-prime trimming priority respects spec ---

func TestColonyPrimeTrimPriorityHiveWisdomAndUserPrefs(t *testing.T) {
	saveGlobalsCmd(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf
	var errBuf bytes.Buffer
	stderr = &errBuf

	s, tmpDir := newTestStoreCmd(t)
	defer os.RemoveAll(tmpDir)
	store = s

	// Set up hub directory with hive wisdom
	hubDir := t.TempDir()
	os.MkdirAll(filepath.Join(hubDir, "hive"), 0755)
	os.MkdirAll(filepath.Join(hubDir, "eternal"), 0755)

	// Create many hive wisdom entries to fill budget
	var entries []string
	for i := 0; i < 100; i++ {
		entries = append(entries, fmt.Sprintf(`{"id":"go_%d","text":"Wisdom entry number %d with enough text to consume budget space","domain":"go","source_repo":"test","confidence":0.85,"created_at":"2026-04-01T00:00:00Z","accessed_at":"2026-04-01T00:00:00Z","access_count":1}`, i, i))
	}
	wisdomData := `{"entries":[` + strings.Join(entries, ",") + `]}`
	if err := os.WriteFile(filepath.Join(hubDir, "hive", "wisdom.json"), []byte(wisdomData), 0644); err != nil {
		t.Fatal(err)
	}

	queenContent := `# QUEEN.md

## User Preferences
` + strings.Repeat("- Preference number %d with enough text to fill budget space\n", 100)
	if err := os.WriteFile(filepath.Join(hubDir, "QUEEN.md"), []byte(queenContent), 0644); err != nil {
		t.Fatal(err)
	}

	origHubDir := os.Getenv("AETHER_HUB_DIR")
	os.Setenv("AETHER_HUB_DIR", hubDir)
	defer os.Setenv("AETHER_HUB_DIR", origHubDir)

	// Use compact mode (4000 budget) to force trimming
	goal := "trim priority test"
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

	// Add lots of phase learnings to fill budget
	now := time.Now().Format(time.RFC3339)
	state.Memory.PhaseLearnings = []colony.PhaseLearning{
		{
			Phase:     1,
			PhaseName: "Phase One",
			Learnings: []colony.Learning{
				{Claim: strings.Repeat("This is a very long learning to fill budget space. ", 50), Status: "confirmed"},
			},
		},
	}
	if err := s.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatal(err)
	}

	// Add lots of decisions too
	state.Memory.Decisions = []colony.Decision{}
	for i := 0; i < 50; i++ {
		state.Memory.Decisions = append(state.Memory.Decisions, colony.Decision{
			ID:        fmt.Sprintf("d%d", i),
			Phase:     1,
			Claim:     fmt.Sprintf("Decision %d: %s", i, strings.Repeat("long text ", 20)),
			Rationale: "because",
			Timestamp: now,
		})
	}
	if err := s.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatal(err)
	}

	rootCmd.SetArgs([]string{"colony-prime", "--compact"})
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
	budget := int(result["budget"].(float64))

	if budget != 4000 {
		t.Errorf("budget = %d, want 4000 for compact mode", budget)
	}

	// State section (priority 5) should always be present
	if !strings.Contains(contextStr, "trim priority test") {
		t.Error("context should always contain colony goal (high priority)")
	}

	// Pheromones (priority 9) should NOT be trimmed before hive wisdom (priority 4)
	trimmed := result["trimmed"].([]interface{})
	for _, name := range trimmed {
		if name.(string) == "pheromones" {
			t.Error("pheromones should not be trimmed -- it has the highest priority (9)")
		}
	}
}

// --- TDD Cycle 8: colony-prime hive wisdom falls back to eternal memory ---

func TestColonyPrimeHiveWisdomEternalFallback(t *testing.T) {
	saveGlobalsCmd(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf
	var errBuf bytes.Buffer
	stderr = &errBuf

	s, tmpDir := newTestStoreCmd(t)
	defer os.RemoveAll(tmpDir)
	store = s

	// Hive retrieval is on by default (D-01/D-02); no opt-in needed.

	// Set up hub directory with ONLY eternal memory (no hive)
	hubDir := t.TempDir()
	os.MkdirAll(filepath.Join(hubDir, "eternal"), 0755)

	eternalData := `[{"text":"Eternal memory fallback entry"}]`
	if err := os.WriteFile(filepath.Join(hubDir, "eternal", "memory.json"), []byte(eternalData), 0644); err != nil {
		t.Fatal(err)
	}

	origHubDir := os.Getenv("AETHER_HUB_DIR")
	os.Setenv("AETHER_HUB_DIR", hubDir)
	defer os.Setenv("AETHER_HUB_DIR", origHubDir)

	goal := "eternal fallback test"
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

	if !strings.Contains(contextStr, "HIVE WISDOM") {
		t.Error("context missing 'HIVE WISDOM' section (should fall back to eternal)")
	}
	if !strings.Contains(contextStr, "Eternal memory fallback entry") {
		t.Error("context missing eternal memory fallback text")
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
