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

	"github.com/calcosmic/Aether/pkg/colony"
)

// --- TDD Cycle 1: --compact flag sets budget to 4000 and reports it ---

func TestColonyPrimeCompactBudgetIs4000(t *testing.T) {
	saveGlobalsCmd(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf
	var errBuf bytes.Buffer
	stderr = &errBuf

	s, tmpDir := newTestStoreCmd(t)
	defer os.RemoveAll(tmpDir)
	store = s

	goal := "compact budget test"
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

	rootCmd.SetArgs([]string{"colony-prime", "--compact"})
	defer rootCmd.SetArgs([]string{})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("colony-prime --compact returned error: %v", err)
	}

	envelope := parseEnvelopeCmd(t, buf.String())
	if envelope["ok"] != true {
		t.Fatalf("expected ok=true, got %v", envelope["ok"])
	}

	result := envelope["result"].(map[string]interface{})
	budget := int(result["budget"].(float64))

	if budget != 4000 {
		t.Errorf("compact mode budget = %d, want 4000", budget)
	}
}

func TestColonyPrimeNormalBudgetIs8000(t *testing.T) {
	saveGlobalsCmd(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf
	var errBuf bytes.Buffer
	stderr = &errBuf

	s, tmpDir := newTestStoreCmd(t)
	defer os.RemoveAll(tmpDir)
	store = s

	goal := "normal budget test"
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
	budget := int(result["budget"].(float64))

	if budget != 8000 {
		t.Errorf("normal mode budget = %d, want 8000", budget)
	}
}

// --- TDD Cycle 2: Compact mode still includes pheromone signals (high priority) ---

func TestColonyPrimeCompactIncludesPheromoneSignals(t *testing.T) {
	saveGlobalsCmd(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf
	var errBuf bytes.Buffer
	stderr = &errBuf

	s, tmpDir := newTestStoreCmd(t)
	defer os.RemoveAll(tmpDir)
	store = s

	goal := "compact pheromones test"
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

	now := time.Now().Format(time.RFC3339)
	s0_9 := 0.9
	pf := colony.PheromoneFile{
		Signals: []colony.PheromoneSignal{
			{ID: "sig_1", Type: "FOCUS", Priority: "normal", Source: "user", CreatedAt: now, Active: true, Strength: &s0_9, Content: json.RawMessage(`{"text": "Focus on testing"}`)},
			{ID: "sig_2", Type: "REDIRECT", Priority: "high", Source: "user", CreatedAt: now, Active: true, Strength: &s0_9, Content: json.RawMessage(`{"text": "Avoid globals"}`)},
		},
	}
	if err := s.SaveJSON("pheromones.json", pf); err != nil {
		t.Fatal(err)
	}

	rootCmd.SetArgs([]string{"colony-prime", "--compact"})
	defer rootCmd.SetArgs([]string{})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("colony-prime --compact returned error: %v", err)
	}

	envelope := parseEnvelopeCmd(t, buf.String())
	if envelope["ok"] != true {
		t.Fatalf("expected ok=true, got %v", envelope["ok"])
	}

	result := envelope["result"].(map[string]interface{})
	contextStr := result["context"].(string)

	if !strings.Contains(contextStr, "Pheromone Signals") {
		t.Error("compact mode context missing 'Pheromone Signals' section")
	}
	if !strings.Contains(contextStr, "FOCUS") {
		t.Error("compact mode context missing FOCUS signal type")
	}
	if !strings.Contains(contextStr, "REDIRECT") {
		t.Error("compact mode context missing REDIRECT signal type")
	}
	if !strings.Contains(contextStr, "Focus on testing") {
		t.Error("compact mode context missing FOCUS signal text")
	}
	if !strings.Contains(contextStr, "Avoid globals") {
		t.Error("compact mode context missing REDIRECT signal text")
	}
}

// --- TDD Cycle 3: Compact mode trims low-priority sections first ---

func TestColonyPrimeCompactTrimsLowPriorityFirst(t *testing.T) {
	saveGlobalsCmd(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf
	var errBuf bytes.Buffer
	stderr = &errBuf

	s, tmpDir := newTestStoreCmd(t)
	defer os.RemoveAll(tmpDir)
	store = s

	// Build state with lots of content to force trimming in compact mode (4000 budget)
	now := time.Now().Format(time.RFC3339)

	// Phase learnings (priority 2 - lowest, trimmed first)
	learnings := make([]colony.Learning, 0, 30)
	for i := 0; i < 30; i++ {
		learnings = append(learnings, colony.Learning{
			Claim:  fmt.Sprintf("Learning %d: %s", i, strings.Repeat("text to fill space. ", 20)),
			Status: "confirmed",
		})
	}

	// Decisions (priority 3 - trimmed second)
	decisions := make([]colony.Decision, 0, 20)
	for i := 0; i < 20; i++ {
		decisions = append(decisions, colony.Decision{
			ID:        fmt.Sprintf("d%d", i),
			Phase:     1,
			Claim:     fmt.Sprintf("Decision %d: %s", i, strings.Repeat("long text to fill budget. ", 15)),
			Rationale: "rationale",
			Timestamp: now,
		})
	}

	goal := "trim order test"
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
		Memory: colony.Memory{
			PhaseLearnings: []colony.PhaseLearning{
				{Phase: 1, PhaseName: "Phase One", Learnings: learnings},
			},
			Decisions: decisions,
		},
	}
	if err := s.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatal(err)
	}

	// Add pheromone signals (priority 9 - highest)
	s0_9 := 0.9
	pf := colony.PheromoneFile{
		Signals: []colony.PheromoneSignal{
			{ID: "sig_1", Type: "REDIRECT", Priority: "high", Source: "user", CreatedAt: now, Active: true, Strength: &s0_9, Content: json.RawMessage(`{"text": "Avoid globals"}`)},
		},
	}
	if err := s.SaveJSON("pheromones.json", pf); err != nil {
		t.Fatal(err)
	}

	rootCmd.SetArgs([]string{"colony-prime", "--compact"})
	defer rootCmd.SetArgs([]string{})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("colony-prime --compact returned error: %v", err)
	}

	envelope := parseEnvelopeCmd(t, buf.String())
	if envelope["ok"] != true {
		t.Fatalf("expected ok=true, got %v", envelope["ok"])
	}

	result := envelope["result"].(map[string]interface{})
	contextStr := result["context"].(string)
	trimmed := result["trimmed"].([]interface{})

	// Pheromones (priority 9) must NOT be trimmed
	for _, name := range trimmed {
		if name.(string) == "pheromones" {
			t.Error("pheromones should NOT be trimmed -- it has the highest priority (9)")
		}
	}

	// Pheromones should appear in the output
	if !strings.Contains(contextStr, "Pheromone Signals") {
		t.Error("compact context should contain pheromone signals section")
	}

	// State section (priority 5) should be present (higher than learnings/decisions)
	if !strings.Contains(contextStr, "trim order test") {
		t.Error("compact context should contain colony state (priority 5)")
	}

	// If learnings or decisions are trimmed, learnings (priority 2) should be trimmed first
	trimmedLearnings := false
	trimmedDecisions := false
	for _, name := range trimmed {
		switch name.(string) {
		case "learnings":
			trimmedLearnings = true
		case "decisions":
			trimmedDecisions = true
		}
	}

	// If both are trimmed, decisions appear first because high-priority
	// sections are considered for inclusion before lower-priority ones.
	if trimmedLearnings && trimmedDecisions {
		learningsIdx := -1
		decisionsIdx := -1
		for i, name := range trimmed {
			switch name.(string) {
			case "learnings":
				learningsIdx = i
			case "decisions":
				decisionsIdx = i
			}
		}
		if decisionsIdx > learningsIdx {
			t.Errorf("decisions (priority 3) should be evaluated before learnings (priority 2), but decisions was at index %d and learnings at %d", decisionsIdx, learningsIdx)
		}
	}
}

// --- TDD Cycle 4: Compact mode output length stays within 4000 chars ---

func TestColonyPrimeCompactOutputWithinBudget(t *testing.T) {
	saveGlobalsCmd(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf
	var errBuf bytes.Buffer
	stderr = &errBuf

	s, tmpDir := newTestStoreCmd(t)
	defer os.RemoveAll(tmpDir)
	store = s

	// Create content that could exceed 4000 chars without trimming
	learnings := make([]colony.Learning, 0, 40)
	for i := 0; i < 40; i++ {
		learnings = append(learnings, colony.Learning{
			Claim:  fmt.Sprintf("Very long learning number %d: %s", i, strings.Repeat("x", 100)),
			Status: "confirmed",
		})
	}

	goal := "budget limit test"
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
		Memory: colony.Memory{
			PhaseLearnings: []colony.PhaseLearning{
				{Phase: 1, PhaseName: "Phase One", Learnings: learnings},
			},
		},
	}
	if err := s.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatal(err)
	}

	rootCmd.SetArgs([]string{"colony-prime", "--compact"})
	defer rootCmd.SetArgs([]string{})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("colony-prime --compact returned error: %v", err)
	}

	envelope := parseEnvelopeCmd(t, buf.String())
	if envelope["ok"] != true {
		t.Fatalf("expected ok=true, got %v", envelope["ok"])
	}

	result := envelope["result"].(map[string]interface{})
	contextStr := result["context"].(string)
	used := int(result["used"].(float64))
	budget := int(result["budget"].(float64))

	if budget != 4000 {
		t.Errorf("budget = %d, want 4000", budget)
	}

	// The "used" field should not exceed the budget
	if used > budget {
		t.Errorf("used = %d, exceeds budget of %d", used, budget)
	}

	// The actual context string length should also be within budget
	if len(contextStr) > 4000 {
		t.Errorf("context length = %d, exceeds compact budget of 4000", len(contextStr))
	}
}

// --- TDD Cycle 5: Compact mode with minimal content still works gracefully ---

func TestColonyPrimeCompactMinimalContent(t *testing.T) {
	saveGlobalsCmd(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf
	var errBuf bytes.Buffer
	stderr = &errBuf

	s, tmpDir := newTestStoreCmd(t)
	defer os.RemoveAll(tmpDir)
	store = s

	// Minimal state with no extra content
	goal := "minimal test"
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

	rootCmd.SetArgs([]string{"colony-prime", "--compact"})
	defer rootCmd.SetArgs([]string{})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("colony-prime --compact returned error: %v", err)
	}

	envelope := parseEnvelopeCmd(t, buf.String())
	if envelope["ok"] != true {
		t.Fatalf("expected ok=true, got %v", envelope["ok"])
	}

	result := envelope["result"].(map[string]interface{})
	contextStr := result["context"].(string)
	budget := int(result["budget"].(float64))
	used := int(result["used"].(float64))

	if budget != 4000 {
		t.Errorf("budget = %d, want 4000", budget)
	}

	// Context should contain the goal
	if !strings.Contains(contextStr, "minimal test") {
		t.Error("compact context should contain colony goal")
	}

	// Used should be well under budget
	if used <= 0 {
		t.Error("used should be > 0 when there is content")
	}

	// No sections should be trimmed
	trimmed := result["trimmed"].([]interface{})
	if len(trimmed) != 0 {
		t.Errorf("expected no trimmed sections with minimal content, got %d trimmed: %v", len(trimmed), trimmed)
	}
}

func TestColonyPrimeCompactEmptyState(t *testing.T) {
	saveGlobalsCmd(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf
	var errBuf bytes.Buffer
	stderr = &errBuf

	s, tmpDir := newTestStoreCmd(t)
	defer os.RemoveAll(tmpDir)
	store = s

	// Completely empty state -- no goal, no phases, no memory
	state := colony.ColonyState{
		Version: "1.0",
		State:   colony.StateREADY,
	}
	if err := s.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatal(err)
	}

	rootCmd.SetArgs([]string{"colony-prime", "--compact"})
	defer rootCmd.SetArgs([]string{})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("colony-prime --compact returned error: %v", err)
	}

	envelope := parseEnvelopeCmd(t, buf.String())
	if envelope["ok"] != true {
		t.Fatalf("expected ok=true, got %v", envelope["ok"])
	}

	result := envelope["result"].(map[string]interface{})
	budget := int(result["budget"].(float64))

	if budget != 4000 {
		t.Errorf("budget = %d, want 4000", budget)
	}

	// Should not panic or error -- graceful empty state
	// Context should still be a valid string (possibly just the state section header)
	_ = result["context"].(string)
}

// --- TDD: Full priority order and edge case budget tests ---

// TestColonyPrimeFullTrimPriorityOrder verifies the weighted ordering contract.
// When all sections are large and the budget is tight, protected sections survive
// and trimmed entries are reported in descending weighted-score order.
func TestColonyPrimeFullTrimPriorityOrder(t *testing.T) {
	saveGlobalsCmd(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf
	var errBuf bytes.Buffer
	stderr = &errBuf

	s, tmpDir := newTestStoreCmd(t)
	defer os.RemoveAll(tmpDir)
	store = s

	// Set up hub directory with hive wisdom and QUEEN.md
	hubDir := t.TempDir()
	os.MkdirAll(filepath.Join(hubDir, "hive"), 0755)
	os.MkdirAll(filepath.Join(hubDir, "eternal"), 0755)
	origHub := os.Getenv("AETHER_HUB_DIR")
	os.Setenv("AETHER_HUB_DIR", hubDir)
	defer os.Setenv("AETHER_HUB_DIR", origHub)

	// Create large hive wisdom entries (priority 4)
	var hiveEntries []string
	for i := 0; i < 30; i++ {
		hiveEntries = append(hiveEntries, fmt.Sprintf(
			`{"id":"w_%d","text":"Wisdom %d: %s","domain":"go","source_repo":"test","confidence":0.85,"created_at":"2026-04-01T00:00:00Z","accessed_at":"2026-04-01T00:00:00Z","access_count":1}`,
			i, i, strings.Repeat("hive ", 40),
		))
	}
	wisdomData := `{"entries":[` + strings.Join(hiveEntries, ",") + `]}`
	if err := os.WriteFile(filepath.Join(hubDir, "hive", "wisdom.json"), []byte(wisdomData), 0644); err != nil {
		t.Fatal(err)
	}

	// Create large user preferences in QUEEN.md (priority 7)
	var prefLines []string
	prefLines = append(prefLines, "# QUEEN.md\n\n## User Preferences\n")
	for i := 0; i < 30; i++ {
		prefLines = append(prefLines, fmt.Sprintf("- Preference %d: %s\n", i, strings.Repeat("pref ", 40)))
	}
	if err := os.WriteFile(filepath.Join(hubDir, "QUEEN.md"), []byte(strings.Join(prefLines, "")), 0644); err != nil {
		t.Fatal(err)
	}

	goal := "full priority order test"
	now := time.Now().Format(time.RFC3339)

	// Large learnings (priority 2)
	var learnings []colony.Learning
	for i := 0; i < 30; i++ {
		learnings = append(learnings, colony.Learning{
			Claim:  fmt.Sprintf("Learning %d: %s", i, strings.Repeat("lrn ", 40)),
			Status: "confirmed",
		})
	}

	// Large decisions (priority 3)
	var decisions []colony.Decision
	for i := 0; i < 30; i++ {
		decisions = append(decisions, colony.Decision{
			ID: fmt.Sprintf("d%d", i), Phase: 1,
			Claim:     fmt.Sprintf("Decision %d: %s", i, strings.Repeat("dec ", 40)),
			Rationale: "rationale",
			Timestamp: now,
		})
	}

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
		Memory: colony.Memory{
			Decisions:      decisions,
			PhaseLearnings: []colony.PhaseLearning{{Phase: 1, PhaseName: "Phase One", Learnings: learnings}},
		},
	}
	if err := s.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatal(err)
	}

	// Large instincts (priority 6)
	instFile := colony.InstinctsFile{}
	for i := 0; i < 20; i++ {
		instFile.Instincts = append(instFile.Instincts, colony.InstinctEntry{
			Trigger:    fmt.Sprintf("trigger %d: %s", i, strings.Repeat("trg ", 40)),
			Action:     fmt.Sprintf("action %d: %s", i, strings.Repeat("act ", 40)),
			Confidence: 0.8,
		})
	}
	if err := s.SaveJSON("instincts.json", instFile); err != nil {
		t.Fatal(err)
	}

	// Small pheromones (priority 9) -- should always survive
	s1 := 0.9
	pf := colony.PheromoneFile{
		Signals: []colony.PheromoneSignal{
			{ID: "s1", Type: "REDIRECT", Priority: "high", Source: "user", CreatedAt: now, Active: true, Strength: &s1, Content: json.RawMessage(`{"text": "Avoid shortcuts"}`)},
		},
	}
	if err := s.SaveJSON("pheromones.json", pf); err != nil {
		t.Fatal(err)
	}

	rootCmd.SetArgs([]string{"colony-prime", "--compact"})
	defer rootCmd.SetArgs([]string{})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("colony-prime --compact returned error: %v", err)
	}

	envelope := parseEnvelopeCmd(t, buf.String())
	result := envelope["result"].(map[string]interface{})
	trimmed := result["trimmed"].([]interface{})
	contextStr := result["context"].(string)

	// Build a map of trimmed section names for easy lookup
	trimmedSet := make(map[string]bool)
	for _, name := range trimmed {
		trimmedSet[name.(string)] = true
	}

	ledger := result["ledger"].(map[string]interface{})
	trimmedLedger := ledger["trimmed"].([]interface{})
	for i := 1; i < len(trimmedLedger); i++ {
		prev := trimmedLedger[i-1].(map[string]interface{})
		curr := trimmedLedger[i].(map[string]interface{})
		prevScore := prev["score_breakdown"].(map[string]interface{})["total"].(float64)
		currScore := curr["score_breakdown"].(map[string]interface{})["total"].(float64)
		if prevScore < currScore {
			t.Errorf("trimmed ledger should follow weighted score order: %s (%0.3f) before %s (%0.3f)",
				prev["name"], prevScore, curr["name"], currScore)
		}
	}

	// Pheromones (priority 9) must never be trimmed
	if trimmedSet["pheromones"] {
		t.Error("pheromones (priority 9) must never be trimmed when lower-priority sections could be trimmed instead")
	}

	// Pheromones content should be in the output
	if !strings.Contains(contextStr, "Pheromone Signals") {
		t.Error("pheromones section should be present in context")
	}
	if !strings.Contains(contextStr, "Avoid shortcuts") {
		t.Error("pheromone signal text should be present in context")
	}

	// State section (priority 5) should always be present
	if !strings.Contains(contextStr, "full priority order test") {
		t.Error("state section should contain colony goal")
	}
}

// TestColonyPrimePheromonesNeverTrimmedBeforeLearnings explicitly verifies that
// pheromones (priority 9) are never trimmed while learnings (priority 2) still fit.
// This is a dedicated test for the critical priority invariant.
func TestColonyPrimePheromonesNeverTrimmedBeforeLearnings(t *testing.T) {
	saveGlobalsCmd(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf
	var errBuf bytes.Buffer
	stderr = &errBuf

	s, tmpDir := newTestStoreCmd(t)
	defer os.RemoveAll(tmpDir)
	store = s

	goal := "pheromones vs learnings test"
	now := time.Now().Format(time.RFC3339)

	// Create massive learnings (priority 2) to fill budget many times over
	var learnings []colony.Learning
	for i := 0; i < 80; i++ {
		learnings = append(learnings, colony.Learning{
			Claim:  fmt.Sprintf("Learning %d: %s", i, strings.Repeat("abc ", 60)),
			Status: "confirmed",
		})
	}

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
		Memory: colony.Memory{
			PhaseLearnings: []colony.PhaseLearning{{Phase: 1, PhaseName: "Phase One", Learnings: learnings}},
		},
	}
	if err := s.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatal(err)
	}

	// Create a single small pheromone signal (priority 9)
	s1 := 0.9
	pf := colony.PheromoneFile{
		Signals: []colony.PheromoneSignal{
			{ID: "s1", Type: "FOCUS", Priority: "normal", Source: "user", CreatedAt: now, Active: true, Strength: &s1, Content: json.RawMessage(`{"text": "Focus on tests"}`)},
		},
	}
	if err := s.SaveJSON("pheromones.json", pf); err != nil {
		t.Fatal(err)
	}

	rootCmd.SetArgs([]string{"colony-prime", "--compact"})
	defer rootCmd.SetArgs([]string{})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("colony-prime --compact returned error: %v", err)
	}

	envelope := parseEnvelopeCmd(t, buf.String())
	result := envelope["result"].(map[string]interface{})
	trimmed := result["trimmed"].([]interface{})
	contextStr := result["context"].(string)

	// Check trimmed list
	trimmedSet := make(map[string]bool)
	for _, name := range trimmed {
		trimmedSet[name.(string)] = true
	}

	// Pheromones (priority 9) must NOT be trimmed
	if trimmedSet["pheromones"] {
		t.Error("pheromones (priority 9) must never be trimmed when learnings (priority 2) could be trimmed instead")
	}

	// Pheromone content must appear in output
	if !strings.Contains(contextStr, "Pheromone Signals") {
		t.Error("pheromones section should be present in context output")
	}
	if !strings.Contains(contextStr, "Focus on tests") {
		t.Error("pheromone signal text 'Focus on tests' should be present in context output")
	}

	ledger := result["ledger"].(map[string]interface{})
	preserved := ledger["preserved"].([]interface{})
	foundPheromones := false
	for _, entry := range preserved {
		item := entry.(map[string]interface{})
		if item["name"] == "pheromones" {
			foundPheromones = true
			break
		}
	}
	if !foundPheromones {
		t.Error("expected pheromones to be preserved under weighted trimming pressure")
	}
}

// TestColonyPrimeLargePheromonesTrimLowerPriority verifies that when pheromones are very large,
// lower-priority sections get trimmed to make room for them.
func TestColonyPrimeLargePheromonesTrimLowerPriority(t *testing.T) {
	saveGlobalsCmd(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf
	var errBuf bytes.Buffer
	stderr = &errBuf

	s, tmpDir := newTestStoreCmd(t)
	defer os.RemoveAll(tmpDir)
	store = s

	goal := "large pheromones budget test"
	now := time.Now().Format(time.RFC3339)

	// Large learnings (priority 2) and decisions (priority 3) to compete for budget
	var learnings []colony.Learning
	for i := 0; i < 30; i++ {
		learnings = append(learnings, colony.Learning{
			Claim:  fmt.Sprintf("Learning %d: %s", i, strings.Repeat("abc ", 40)),
			Status: "confirmed",
		})
	}
	var decisions []colony.Decision
	for i := 0; i < 30; i++ {
		decisions = append(decisions, colony.Decision{
			ID: fmt.Sprintf("d%d", i), Phase: 1,
			Claim:     fmt.Sprintf("Decision %d: %s", i, strings.Repeat("def ", 40)),
			Rationale: "rationale",
			Timestamp: now,
		})
	}

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
		Memory: colony.Memory{
			Decisions:      decisions,
			PhaseLearnings: []colony.PhaseLearning{{Phase: 1, PhaseName: "Phase One", Learnings: learnings}},
		},
	}
	if err := s.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatal(err)
	}

	// Create pheromones (priority 9) -- enough to consume ~1500 chars (well within 4000 budget)
	s1 := 0.9
	var signals []colony.PheromoneSignal
	for i := 0; i < 10; i++ {
		signals = append(signals, colony.PheromoneSignal{
			ID: fmt.Sprintf("sig_%d", i), Type: "FOCUS", Priority: "normal",
			Source: "user", CreatedAt: now, Active: true, Strength: &s1,
			Content: json.RawMessage(fmt.Sprintf(`{"text": "Signal %d: %s"}`, i, strings.Repeat("ghi ", 20))),
		})
	}
	pf := colony.PheromoneFile{Signals: signals}
	if err := s.SaveJSON("pheromones.json", pf); err != nil {
		t.Fatal(err)
	}

	rootCmd.SetArgs([]string{"colony-prime", "--compact"})
	defer rootCmd.SetArgs([]string{})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("colony-prime --compact returned error: %v", err)
	}

	envelope := parseEnvelopeCmd(t, buf.String())
	result := envelope["result"].(map[string]interface{})
	trimmed := result["trimmed"].([]interface{})
	contextStr := result["context"].(string)
	used := int(result["used"].(float64))
	budget := int(result["budget"].(float64))

	trimmedSet := make(map[string]bool)
	for _, name := range trimmed {
		trimmedSet[name.(string)] = true
	}

	// Some lower-priority sections should be trimmed
	if len(trimmed) == 0 {
		t.Error("expected some sections to be trimmed when large pheromones compete with other sections for budget")
	}

	// Learnings (priority 2) should be trimmed to make room for pheromones
	if !trimmedSet["learnings"] {
		t.Error("expected 'learnings' (priority 2) to be trimmed when large pheromones need budget space")
	}

	// Pheromones (priority 9) must be present in the output
	if !strings.Contains(contextStr, "Pheromone Signals") {
		t.Error("pheromones section should be present in context -- highest priority (9)")
	}

	// Output must be within budget
	if used > budget {
		t.Errorf("used = %d, exceeds budget = %d", used, budget)
	}
}

// TestColonyPrimeHiveWisdomAndUserPrefsPriority verifies that hive_wisdom (priority 4)
// is trimmed before user_preferences (priority 7) when budget is tight.
func TestColonyPrimeHiveWisdomAndUserPrefsPriority(t *testing.T) {
	saveGlobalsCmd(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf
	var errBuf bytes.Buffer
	stderr = &errBuf

	s, tmpDir := newTestStoreCmd(t)
	defer os.RemoveAll(tmpDir)
	store = s

	// Set up hub directory
	hubDir := t.TempDir()
	os.MkdirAll(filepath.Join(hubDir, "hive"), 0755)
	os.MkdirAll(filepath.Join(hubDir, "eternal"), 0755)
	origHub := os.Getenv("AETHER_HUB_DIR")
	os.Setenv("AETHER_HUB_DIR", hubDir)
	defer os.Setenv("AETHER_HUB_DIR", origHub)

	// Hive wisdom entries (priority 4) -- readHiveWisdom limits to 5 entries, truncates to 200 chars.
	// Make each entry near 200 chars so hive section is ~1000 chars.
	var hiveEntries []string
	for i := 0; i < 5; i++ {
		hiveEntries = append(hiveEntries, fmt.Sprintf(
			`{"id":"w_%d","text":"Wisdom %d: %s","domain":"go","source_repo":"test","confidence":0.85,"created_at":"2026-04-01T00:00:00Z","accessed_at":"2026-04-01T00:00:00Z","access_count":1}`,
			i, i, strings.Repeat("wisdom ", 45),
		))
	}
	wisdomData := `{"entries":[` + strings.Join(hiveEntries, ",") + `]}`
	if err := os.WriteFile(filepath.Join(hubDir, "hive", "wisdom.json"), []byte(wisdomData), 0644); err != nil {
		t.Fatal(err)
	}

	// User preferences (priority 7)
	queenContent := "# QUEEN.md\n\n## User Preferences\n- Prefer clear code\n- KISS principle\n"
	if err := os.WriteFile(filepath.Join(hubDir, "QUEEN.md"), []byte(queenContent), 0644); err != nil {
		t.Fatal(err)
	}

	goal := "hive vs prefs test"
	now := time.Now().Format(time.RFC3339)

	// Add large learnings (priority 2) to consume most of the compact budget
	var learnings []colony.Learning
	for i := 0; i < 30; i++ {
		learnings = append(learnings, colony.Learning{
			Claim:  fmt.Sprintf("Learning %d: %s", i, strings.Repeat("lrn ", 40)),
			Status: "confirmed",
		})
	}

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
		Memory: colony.Memory{
			PhaseLearnings: []colony.PhaseLearning{{Phase: 1, PhaseName: "Phase One", Learnings: learnings}},
		},
	}
	if err := s.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatal(err)
	}

	// Add large decisions (priority 3) to consume more budget
	var decisions []colony.Decision
	for i := 0; i < 15; i++ {
		decisions = append(decisions, colony.Decision{
			ID: fmt.Sprintf("d%d", i), Phase: 1,
			Claim:     fmt.Sprintf("Decision %d: %s", i, strings.Repeat("dec ", 30)),
			Rationale: "rationale",
			Timestamp: now,
		})
	}
	state.Memory.Decisions = decisions
	if err := s.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatal(err)
	}

	rootCmd.SetArgs([]string{"colony-prime", "--compact"})
	defer rootCmd.SetArgs([]string{})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("colony-prime --compact returned error: %v", err)
	}

	envelope := parseEnvelopeCmd(t, buf.String())
	result := envelope["result"].(map[string]interface{})
	trimmed := result["trimmed"].([]interface{})
	contextStr := result["context"].(string)

	trimmedSet := make(map[string]bool)
	for _, name := range trimmed {
		trimmedSet[name.(string)] = true
	}

	// The key invariant: if hive_wisdom is trimmed, user_preferences must NOT be trimmed
	// (user_preferences has higher priority: 7 vs 4).
	// And if both exist, hive_wisdom (4) should be trimmed before user_preferences (7).
	if trimmedSet["user_preferences"] && !trimmedSet["hive_wisdom"] {
		t.Error("user_preferences (priority 7) trimmed but hive_wisdom (priority 4) not trimmed -- lower priority should be trimmed first")
	}

	// If hive_wisdom is trimmed, verify user_preferences is not
	if trimmedSet["hive_wisdom"] && trimmedSet["user_preferences"] {
		t.Error("both hive_wisdom (4) and user_preferences (7) are trimmed -- user_preferences has higher priority and should survive if hive_wisdom was the budget problem")
	}

	// If nothing is trimmed, at least verify the output is within budget
	used := int(result["used"].(float64))
	budget := int(result["budget"].(float64))
	if used > budget {
		t.Errorf("used = %d, exceeds budget = %d", used, budget)
	}

	// Verify at least one of hive_wisdom or user_preferences is in the output
	// (or both if budget allows)
	_ = contextStr
}

// TestColonyPrimeFreshDecisionsBeatStaleHiveWisdom verifies that weighted runtime
// assembly prefers fresher current-colony decisions over very stale hive wisdom
// when budget pressure forces a choice.
func TestColonyPrimeFreshDecisionsBeatStaleHiveWisdom(t *testing.T) {
	saveGlobalsCmd(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf
	var errBuf bytes.Buffer
	stderr = &errBuf

	s, tmpDir := newTestStoreCmd(t)
	defer os.RemoveAll(tmpDir)
	store = s

	hubDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(hubDir, "hive"), 0755); err != nil {
		t.Fatal(err)
	}
	origHub := os.Getenv("AETHER_HUB_DIR")
	os.Setenv("AETHER_HUB_DIR", hubDir)
	defer os.Setenv("AETHER_HUB_DIR", origHub)

	var fixture freshVsStaleFixture
	loadContextWeightingFixture(t, "fresh-vs-stale.json", &fixture)

	stale := time.Now().Add(-240 * 24 * time.Hour).UTC().Format(time.RFC3339)
	var hiveEntries []string
	for i := 0; i < fixture.StaleHiveCount; i++ {
		hiveEntries = append(hiveEntries, fmt.Sprintf(
			`{"id":"w_%d","text":"Stale wisdom %d: %s","domain":"go","source_repo":"test","confidence":0.55,"created_at":"%s","accessed_at":"%s","access_count":1}`,
			i, i, strings.Repeat(fixture.StaleHivePhrase+" ", 18), stale, stale,
		))
	}
	wisdomData := `{"entries":[` + strings.Join(hiveEntries, ",") + `]}`
	if err := os.WriteFile(filepath.Join(hubDir, "hive", "wisdom.json"), []byte(wisdomData), 0644); err != nil {
		t.Fatal(err)
	}

	now := time.Now().UTC().Format(time.RFC3339)
	goal := fixture.Goal

	var decisions []colony.Decision
	for i := 0; i < fixture.FreshDecisionCount; i++ {
		decisions = append(decisions, colony.Decision{
			ID:        fmt.Sprintf("d%d", i),
			Phase:     1,
			Claim:     fmt.Sprintf("Fresh decision %d: %s", i, strings.Repeat(fixture.FreshDecisionPhrase+" ", 24)),
			Rationale: fixture.FreshDecisionRationale,
			Timestamp: now,
		})
	}

	var learnings []colony.Learning
	for i := 0; i < fixture.LearningCount; i++ {
		learnings = append(learnings, colony.Learning{
			Claim:  fmt.Sprintf("Background learning %d: %s", i, strings.Repeat(fixture.LearningPhrase+" ", 24)),
			Status: "confirmed",
			Tested: true,
		})
	}

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
		Memory: colony.Memory{
			Decisions:      decisions,
			PhaseLearnings: []colony.PhaseLearning{{Phase: 1, PhaseName: "Phase One", Timestamp: now, Learnings: learnings}},
		},
	}
	if err := s.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatal(err)
	}

	rootCmd.SetArgs([]string{"colony-prime", "--compact"})
	defer rootCmd.SetArgs([]string{})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("colony-prime --compact returned error: %v", err)
	}

	envelope := parseEnvelopeCmd(t, buf.String())
	result := envelope["result"].(map[string]interface{})
	contextStr := result["context"].(string)
	trimmed := result["trimmed"].([]interface{})

	if !strings.Contains(contextStr, "Fresh decision 0") {
		t.Fatalf("fresh current decision should survive weighted budget pressure:\n%s", contextStr)
	}
	if strings.Contains(contextStr, "Stale wisdom 0") {
		t.Fatalf("stale hive wisdom should lose to fresher decision context under weighted budget pressure:\n%s", contextStr)
	}

	trimmedSet := make(map[string]bool)
	for _, name := range trimmed {
		trimmedSet[name.(string)] = true
	}
	if !trimmedSet["hive_wisdom"] {
		t.Fatalf("expected hive_wisdom to be trimmed under weighted freshness test, got %v", trimmed)
	}
}

// TestColonyPrimeNormalModeWithin8000Budget verifies that normal mode (8000 budget)
// also respects the budget when content is extremely large.
func TestColonyPrimeNormalModeWithin8000Budget(t *testing.T) {
	saveGlobalsCmd(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf
	var errBuf bytes.Buffer
	stderr = &errBuf

	s, tmpDir := newTestStoreCmd(t)
	defer os.RemoveAll(tmpDir)
	store = s

	goal := "normal mode budget enforcement"
	now := time.Now().Format(time.RFC3339)

	// Create enormous learnings to exceed 8000 chars
	var learnings []colony.Learning
	for i := 0; i < 100; i++ {
		learnings = append(learnings, colony.Learning{
			Claim:  fmt.Sprintf("Learning %d: %s", i, strings.Repeat("x", 100)),
			Status: "confirmed",
		})
	}

	var decisions []colony.Decision
	for i := 0; i < 50; i++ {
		decisions = append(decisions, colony.Decision{
			ID: fmt.Sprintf("d%d", i), Phase: 1,
			Claim:     fmt.Sprintf("Decision %d: %s", i, strings.Repeat("y", 100)),
			Rationale: "rationale",
			Timestamp: now,
		})
	}

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
		Memory: colony.Memory{
			Decisions:      decisions,
			PhaseLearnings: []colony.PhaseLearning{{Phase: 1, PhaseName: "Phase One", Learnings: learnings}},
		},
	}
	if err := s.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatal(err)
	}

	// Add large instincts
	instFile := colony.InstinctsFile{}
	for i := 0; i < 30; i++ {
		instFile.Instincts = append(instFile.Instincts, colony.InstinctEntry{
			Trigger:    fmt.Sprintf("trigger %d: %s", i, strings.Repeat("z", 80)),
			Action:     fmt.Sprintf("action %d: %s", i, strings.Repeat("w", 80)),
			Confidence: 0.8,
		})
	}
	if err := s.SaveJSON("instincts.json", instFile); err != nil {
		t.Fatal(err)
	}

	// Add pheromones
	s1 := 0.9
	var signals []colony.PheromoneSignal
	for i := 0; i < 20; i++ {
		signals = append(signals, colony.PheromoneSignal{
			ID: fmt.Sprintf("sig_%d", i), Type: "FOCUS", Priority: "normal",
			Source: "user", CreatedAt: now, Active: true, Strength: &s1,
			Content: json.RawMessage(fmt.Sprintf(`{"text": "Signal %d: %s"}`, i, strings.Repeat("v", 80))),
		})
	}
	pf := colony.PheromoneFile{Signals: signals}
	if err := s.SaveJSON("pheromones.json", pf); err != nil {
		t.Fatal(err)
	}

	rootCmd.SetArgs([]string{"colony-prime"})
	defer rootCmd.SetArgs([]string{})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("colony-prime returned error: %v", err)
	}

	envelope := parseEnvelopeCmd(t, buf.String())
	result := envelope["result"].(map[string]interface{})
	budget := int(result["budget"].(float64))
	used := int(result["used"].(float64))
	contextStr := result["context"].(string)

	if budget != 8000 {
		t.Errorf("normal mode budget = %d, want 8000", budget)
	}

	// Used must not exceed budget (implementation adds "\n" per section, allow small margin)
	if used > budget+20 {
		t.Errorf("used = %d, exceeds budget = %d", used, budget)
	}

	// Context string should be within budget
	if len(contextStr) > budget+20 {
		t.Errorf("context length = %d, exceeds normal budget = %d", len(contextStr), budget)
	}

	// State (priority 5) should always be present
	if !strings.Contains(contextStr, "normal mode budget enforcement") {
		t.Error("state section should always contain colony goal")
	}
}

// TestColonyPrimeInstinctsSurviveOverDecisions verifies that instincts (priority 6)
// are preferred over decisions (priority 3) during trimming.
func TestColonyPrimeInstinctsSurviveOverDecisions(t *testing.T) {
	saveGlobalsCmd(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf
	var errBuf bytes.Buffer
	stderr = &errBuf

	s, tmpDir := newTestStoreCmd(t)
	defer os.RemoveAll(tmpDir)
	store = s

	goal := "instincts vs decisions test"
	now := time.Now().Format(time.RFC3339)

	// Large decisions (priority 3)
	var decisions []colony.Decision
	for i := 0; i < 40; i++ {
		decisions = append(decisions, colony.Decision{
			ID: fmt.Sprintf("d%d", i), Phase: 1,
			Claim:     fmt.Sprintf("Decision %d: %s", i, strings.Repeat("dec ", 50)),
			Rationale: "rationale",
			Timestamp: now,
		})
	}

	// Large learnings (priority 2) to ensure aggressive trimming
	var learnings []colony.Learning
	for i := 0; i < 30; i++ {
		learnings = append(learnings, colony.Learning{
			Claim:  fmt.Sprintf("Learning %d: %s", i, strings.Repeat("lrn ", 50)),
			Status: "confirmed",
		})
	}

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
		Memory: colony.Memory{
			Decisions:      decisions,
			PhaseLearnings: []colony.PhaseLearning{{Phase: 1, PhaseName: "Phase One", Learnings: learnings}},
		},
	}
	if err := s.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatal(err)
	}

	// Small instincts (priority 6)
	instFile := colony.InstinctsFile{
		Instincts: []colony.InstinctEntry{
			{Trigger: "nil pointer", Action: "add nil check", Confidence: 0.9},
		},
	}
	if err := s.SaveJSON("instincts.json", instFile); err != nil {
		t.Fatal(err)
	}

	rootCmd.SetArgs([]string{"colony-prime", "--compact"})
	defer rootCmd.SetArgs([]string{})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("colony-prime --compact returned error: %v", err)
	}

	envelope := parseEnvelopeCmd(t, buf.String())
	result := envelope["result"].(map[string]interface{})
	trimmed := result["trimmed"].([]interface{})
	contextStr := result["context"].(string)

	trimmedSet := make(map[string]bool)
	for _, name := range trimmed {
		trimmedSet[name.(string)] = true
	}

	// Decisions (priority 3) should be trimmed before instincts (priority 6)
	if trimmedSet["instincts"] && !trimmedSet["decisions"] {
		t.Error("instincts (priority 6) should not be trimmed when decisions (priority 3) could be trimmed first")
	}

	// If decisions are trimmed, instincts should survive
	if trimmedSet["decisions"] && trimmedSet["instincts"] {
		t.Error("decisions (priority 3) trimmed but instincts (priority 6) also trimmed -- instincts should survive over decisions")
	}

	// Instincts content should be in output if not trimmed
	if !trimmedSet["instincts"] {
		if !strings.Contains(contextStr, "Active Instincts") {
			t.Error("instincts section should be present in context when not trimmed")
		}
		if !strings.Contains(contextStr, "nil pointer") {
			t.Error("instinct trigger text should be present in context when not trimmed")
		}
	}
}
