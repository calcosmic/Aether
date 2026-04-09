package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/storage"
)

// setupBenchmarkStore creates a temp store with realistic fixture data for benchmarking.
// It returns the store, tmpDir, and a cleanup function.
func setupBenchmarkStore(b *testing.B) *storage.Store {
	b.Helper()
	tmpDir, err := os.MkdirTemp("", "aether-bench-*")
	if err != nil {
		b.Fatal(err)
	}
	dataDir := tmpDir + "/.aether/data"
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		b.Fatal(err)
	}

	s, err := storage.NewStore(dataDir)
	if err != nil {
		b.Fatal(err)
	}

	// Set COLONY_DATA_DIR so PersistentPreRunE resolves to our temp dir
	os.Setenv("COLONY_DATA_DIR", dataDir)

	// Helper for string pointer
	strPtr := func(s string) *string { return &s }

	// Create fixture colony state
	goal := "Benchmark pheromone injection performance across all data sources"
	now := time.Now().Format(time.RFC3339)
	state := colony.ColonyState{
		Version:      "1.0",
		Goal:         &goal,
		State:        colony.StateEXECUTING,
		CurrentPhase: 2,
		ColonyDepth:  "standard",
		Milestone:    "Open Chambers",
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{
					ID: 1, Name: "Foundation", Status: "completed",
					Tasks: []colony.Task{
						{ID: strPtr("t1"), Goal: "Set up project structure", Status: "completed"},
						{ID: strPtr("t2"), Goal: "Initialize colony state", Status: "completed"},
					},
				},
				{
					ID: 2, Name: "Core Features", Status: "in_progress",
					Tasks: []colony.Task{
						{ID: strPtr("t3"), Goal: "Implement pheromone system", Status: "completed"},
						{ID: strPtr("t4"), Goal: "Add benchmark suite", Status: "in_progress"},
					},
				},
				{
					ID: 3, Name: "Optimization", Status: "pending",
					Tasks: []colony.Task{
						{ID: strPtr("t5"), Goal: "Eliminate redundant spawns", Status: "pending"},
					},
				},
			},
		},
		Memory: colony.Memory{
			Decisions: []colony.Decision{
				{ID: "d1", Phase: 1, Claim: "Use Cobra for CLI framework", Rationale: "Industry standard for Go CLIs", Timestamp: now},
				{ID: "d2", Phase: 1, Claim: "Use JSON envelope for output", Rationale: "Machine-parseable responses", Timestamp: now},
				{ID: "d3", Phase: 2, Claim: "Add allocation tracking to benchmarks", Rationale: "Memory profiling is critical for optimization", Timestamp: now},
				{ID: "d4", Phase: 2, Claim: "Centralize validation in shared layer", Rationale: "Reduce code duplication and improve consistency", Timestamp: now},
				{ID: "d5", Phase: 2, Claim: "Implement session-level caching", Rationale: "Avoid redundant file reads within a session", Timestamp: now},
			},
			Instincts: []colony.Instinct{
				{ID: "i1", Trigger: "test fails", Action: "investigate root cause before fixing", Confidence: 0.9, Status: "active"},
				{ID: "i2", Trigger: "file not found", Action: "check relative vs absolute path", Confidence: 0.85, Status: "active"},
				{ID: "i3", Trigger: "json unmarshal error", Action: "validate input structure first", Confidence: 0.8, Status: "active"},
			},
			PhaseLearnings: []colony.PhaseLearning{
				{
					ID: "pl1", Phase: 1, PhaseName: "Foundation", Timestamp: now,
					Learnings: []colony.Learning{
						{Claim: "Cobra commands need careful flag reset between tests", Status: "validated", Tested: true},
						{Claim: "Storage locking prevents concurrent write corruption", Status: "validated", Tested: true},
					},
				},
				{
					ID: "pl2", Phase: 2, PhaseName: "Core Features", Timestamp: now,
					Learnings: []colony.Learning{
						{Claim: "Pheromone injection reads from 10+ data sources", Status: "observed", Tested: false},
						{Claim: "Budget enforcement trims lowest-priority sections first", Status: "validated", Tested: true},
					},
				},
			},
		},
		Events: []string{
			now + "|init|system|Colony initialized",
			now + "|build|system|Build started phase 1",
			now + "|complete|system|Phase 1 completed",
			now + "|build|system|Build started phase 2",
			now + "|worker|builder|Worker spawned for task t4",
		},
	}
	if err := s.SaveJSON("COLONY_STATE.json", state); err != nil {
		b.Fatal(err)
	}

	// Create pheromone signals
	s0_9 := 0.9
	s1_0 := 1.0
	s0_8 := 0.8
	s0_7 := 0.7
	s0_5 := 0.5
	pf := colony.PheromoneFile{
		Signals: []colony.PheromoneSignal{
			{ID: "s1", Type: "REDIRECT", Priority: "high", Source: "user", CreatedAt: now, Active: true, Strength: &s1_0, Content: json.RawMessage(`{"text": "Avoid redundant process spawns in hot paths"}`)},
			{ID: "s2", Type: "REDIRECT", Priority: "high", Source: "user", CreatedAt: now, Active: true, Strength: &s0_9, Content: json.RawMessage(`{"text": "No shell script fallbacks for core commands"}`)},
			{ID: "s3", Type: "FOCUS", Priority: "normal", Source: "user", CreatedAt: now, Active: true, Strength: &s0_8, Content: json.RawMessage(`{"text": "Focus on pheromone injection performance"}`)},
			{ID: "s4", Type: "FOCUS", Priority: "normal", Source: "user", CreatedAt: now, Active: true, Strength: &s0_7, Content: json.RawMessage(`{"text": "Centralize validation logic"}`)},
			{ID: "s5", Type: "FEEDBACK", Priority: "low", Source: "auto", CreatedAt: now, Active: true, Strength: &s0_5, Content: json.RawMessage(`{"text": "Current benchmarks show acceptable baseline performance"}`)},
		},
	}
	if err := s.SaveJSON("pheromones.json", pf); err != nil {
		b.Fatal(err)
	}

	// Create flags with blockers
	phase := 2
	flags := colony.FlagsFile{
		Version: "1.0",
		Decisions: []colony.FlagEntry{
			{ID: "f1", Type: "blocker", Description: "Must establish baseline before optimizing", Phase: &phase, Source: "route_setter", CreatedAt: now, Resolved: false},
			{ID: "f2", Type: "issue", Description: "Git branch detection adds latency in CI", Phase: &phase, Source: "scout", CreatedAt: now, Resolved: false},
			{ID: "f3", Type: "note", Description: "Consider caching queen wisdom reads", Source: "builder", CreatedAt: now, Resolved: true},
		},
	}
	if err := s.SaveJSON("flags.json", flags); err != nil {
		b.Fatal(err)
	}

	// Create midden entries
	if err := os.MkdirAll(s.BasePath()+"/midden", 0755); err != nil {
		b.Fatal(err)
	}
	midden := colony.MiddenFile{
		Entries: []colony.MiddenEntry{
			{ID: "m1", Timestamp: "2026-04-01T10:00:00Z", Category: "build", Source: "builder", Message: "Build failed: missing dependency in context.go"},
			{ID: "m2", Timestamp: "2026-04-01T11:00:00Z", Category: "test", Source: "watcher", Message: "Test timeout: pr-context exceeded 5s threshold"},
			{ID: "m3", Timestamp: "2026-04-01T12:00:00Z", Category: "perf", Source: "measurer", Message: "Pheromone injection latency: 450ms per worker spawn"},
		},
	}
	if err := s.SaveJSON("midden/midden.json", midden); err != nil {
		b.Fatal(err)
	}

	// Create rolling summary with realistic entries
	var summaryLines []string
	entries := []string{
		"Colony initialized with performance optimization goal",
		"Phase 1 foundation completed successfully",
		"Build started for core features phase",
		"Worker spawned for benchmark implementation",
		"Pheromone signals loaded and injected into context",
		"Budget enforcement verified: 6000 char limit respected",
		"Rolling summary captures 20 most recent entries",
		"Context capsule assembled with all 10 data sources",
		"Hive wisdom checked: 3 entries retrieved from hub",
		"Queen wisdom loaded from global and local QUEEN.md",
		"Flags loaded: 1 blocker, 1 issue active",
		"Midden reviewed: 3 recent failure entries",
		"Instincts injected: 3 active patterns",
		"Phase learnings from phases 1 and 2 included",
		"Key decisions: 5 claims assembled into prompt",
	}
	for i, entry := range entries {
		ts := time.Now().Add(-time.Duration(len(entries)-i) * time.Minute).Format(time.RFC3339)
		summaryLines = append(summaryLines, ts+"|build|system|"+entry)
	}
	summaryData := []byte(joinLines(summaryLines))
	if err := s.AtomicWrite("rolling-summary.log", summaryData); err != nil {
		b.Fatal(err)
	}

	// Create instincts.json (standalone file)
	instFile := colony.InstinctsFile{
		Version: "1.0",
		Instincts: []colony.InstinctEntry{
			{ID: "si1", Trigger: "test fails", Action: "investigate root cause before fixing", Confidence: 0.9, Archived: false},
			{ID: "si2", Trigger: "file not found", Action: "check relative vs absolute path", Confidence: 0.85, Archived: false},
			{ID: "si3", Trigger: "json unmarshal error", Action: "validate input structure first", Confidence: 0.8, Archived: false},
			{ID: "si4", Trigger: "build timeout", Action: "check for infinite loops in command chain", Confidence: 0.7, Archived: true},
		},
	}
	if err := s.SaveJSON("instincts.json", instFile); err != nil {
		b.Fatal(err)
	}

	// Set up a hub directory for queen/hive reads
	hubDir := filepath.Join(tmpDir, "hub")
	os.MkdirAll(filepath.Join(hubDir, "hive"), 0755)
	os.MkdirAll(filepath.Join(hubDir, "eternal"), 0755)
	os.Setenv("AETHER_HUB_DIR", hubDir)

	// Create QUEEN.md with wisdom and patterns
	queenContent := `# QUEEN.md

## Wisdom
Prefer simplicity over cleverness: Choose the straightforward solution.
Test before optimize: Never optimize without a baseline measurement.

## Patterns
TDD discipline: Write failing test first, then implement.
Fail fast: Return errors early rather than accumulating state.

## User Preferences
- Plain English explanations preferred
- Prefer working software over perfect architecture
- Ship often, iterate quickly
`
	if err := os.WriteFile(filepath.Join(hubDir, "QUEEN.md"), []byte(queenContent), 0644); err != nil {
		b.Fatal(err)
	}

	// Create hive wisdom
	hiveWisdom := `{"entries": [
		{"text": "Pheromone injection should be cached per session to avoid redundant reads", "confidence": 0.85},
		{"text": "Centralized validation reduces code duplication and improves error consistency", "confidence": 0.9},
		{"text": "Process spawns are expensive: batch operations where possible", "confidence": 0.8}
	]}`
	if err := os.WriteFile(filepath.Join(hubDir, "hive", "wisdom.json"), []byte(hiveWisdom), 0644); err != nil {
		b.Fatal(err)
	}

	b.Cleanup(func() {
		os.RemoveAll(tmpDir)
	})

	return s
}

// joinLines joins strings with newlines.
func joinLines(lines []string) string {
	result := ""
	for i, line := range lines {
		if i > 0 {
			result += "\n"
		}
		result += line
	}
	return result
}

// BenchmarkPRContext benchmarks the pr-context command path using realistic fixture data.
// This measures the full context assembly from 10+ data sources including pheromones,
// colony state, decisions, blockers, midden, rolling summary, and queen/hive wisdom.
func BenchmarkPRContext(b *testing.B) {
	b.ReportAllocs()

	s := setupBenchmarkStore(b)

	// Save and restore globals
	origStdout := stdout
	origStderr := stderr
	origStore := store
	defer func() {
		stdout = origStdout
		stderr = origStderr
		store = origStore
	}()

	var buf bytes.Buffer
	var errBuf bytes.Buffer

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Reset()
		errBuf.Reset()

		stdout = &buf
		stderr = &errBuf
		store = s

		rootCmd.SetArgs([]string{"pr-context"})
		if err := rootCmd.Execute(); err != nil {
			b.Fatalf("pr-context failed: %v", err)
		}
		rootCmd.SetArgs([]string{})
	}
}

// BenchmarkColonyPrime benchmarks the colony-prime command path using a fixture store.
// This measures the full colony context assembly including state loading, pheromone
// injection, instinct loading, decision formatting, and budget enforcement.
func BenchmarkColonyPrime(b *testing.B) {
	b.ReportAllocs()

	s := setupBenchmarkStore(b)

	// Save and restore globals
	origStdout := stdout
	origStderr := stderr
	origStore := store
	defer func() {
		stdout = origStdout
		stderr = origStderr
		store = origStore
	}()

	var buf bytes.Buffer
	var errBuf bytes.Buffer

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Reset()
		errBuf.Reset()

		stdout = &buf
		stderr = &errBuf
		store = s

		rootCmd.SetArgs([]string{"colony-prime"})
		if err := rootCmd.Execute(); err != nil {
			b.Fatalf("colony-prime failed: %v", err)
		}
		rootCmd.SetArgs([]string{})
	}
}
