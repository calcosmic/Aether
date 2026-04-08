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
	"github.com/calcosmic/Aether/pkg/storage"
)

// newTestStoreCmd creates a temp directory with .aether/data/ and returns a Store.
// It also sets COLONY_DATA_DIR so PersistentPreRunE resolves to the temp dir.
func newTestStoreCmd(t *testing.T) (*storage.Store, string) {
	t.Helper()
	tmpDir, err := os.MkdirTemp("", "aether-context-test-*")
	if err != nil {
		t.Fatal(err)
	}
	dataDir := tmpDir + "/.aether/data"
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		t.Fatal(err)
	}
	s, err := storage.NewStore(dataDir)
	if err != nil {
		t.Fatal(err)
	}

	// Set COLONY_DATA_DIR so PersistentPreRunE initializes store to our temp dir
	origDataDir := os.Getenv("COLONY_DATA_DIR")
	os.Setenv("COLONY_DATA_DIR", dataDir)
	t.Cleanup(func() {
		os.Setenv("COLONY_DATA_DIR", origDataDir)
	})

	return s, tmpDir
}

// parseEnvelopeCmd parses JSON output into a map.
func parseEnvelopeCmd(t *testing.T, output string) map[string]interface{} {
	t.Helper()
	output = strings.TrimSpace(output)
	var envelope map[string]interface{}
	if err := json.Unmarshal([]byte(output), &envelope); err != nil {
		t.Fatalf("failed to parse envelope JSON: %v\noutput: %q", err, output)
	}
	return envelope
}

// saveGlobalsCmd saves and restores stdout, stderr, and store globals.
func saveGlobalsCmd(t *testing.T) {
	t.Helper()
	origStdout := stdout
	origStderr := stderr
	origStore := store
	t.Cleanup(func() {
		stdout = origStdout
		stderr = origStderr
		store = origStore
	})
}

// --- resume-dashboard tests ---

func TestResumeDashboard(t *testing.T) {
	saveGlobalsCmd(t)
	var buf bytes.Buffer
	stdout = &buf
	var errBuf bytes.Buffer
	stderr = &errBuf

	s, tmpDir := newTestStoreCmd(t)
	defer os.RemoveAll(tmpDir)
	store = s

	goal := "test goal"
	state := colony.ColonyState{
		Version:      "1.0",
		Goal:         &goal,
		State:        colony.StateEXECUTING,
		CurrentPhase: 2,
		ColonyDepth:  "standard",
		Milestone:    "Open Chambers",
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{ID: 1, Name: "Foundation", Status: "completed"},
				{ID: 2, Name: "Core Features", Status: "in_progress"},
			},
		},
		Memory: colony.Memory{
			Decisions: []colony.Decision{
				{ID: "d1", Phase: 1, Claim: "Use cobra for CLI", Rationale: "Standard pattern", Timestamp: "2026-04-01T10:00:00Z"},
				{ID: "d2", Phase: 1, Claim: "Use outputOK for responses", Rationale: "Matches shell", Timestamp: "2026-04-01T11:00:00Z"},
				{ID: "d3", Phase: 2, Claim: "Use typed structs", Rationale: "Type safety", Timestamp: "2026-04-01T12:00:00Z"},
			},
		},
		Events: []string{
			"2026-04-01T10:00:00Z|init|system|Colony initialized",
			"2026-04-01T11:00:00Z|build|system|Build started",
			"2026-04-01T12:00:00Z|complete|system|Phase completed",
			"2026-04-01T13:00:00Z|build|system|Build phase 2",
			"2026-04-01T14:00:00Z|event|system|Custom event",
		},
	}
	if err := s.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatal(err)
	}

	rootCmd.SetArgs([]string{"resume-dashboard"})
	defer rootCmd.SetArgs([]string{})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("resume-dashboard returned error: %v", err)
	}

	envelope := parseEnvelopeCmd(t, buf.String())
	if envelope["ok"] != true {
		t.Fatalf("expected ok=true, got %v", envelope["ok"])
	}

	result := envelope["result"].(map[string]interface{})
	current := result["current"].(map[string]interface{})

	if current["goal"] != "test goal" {
		t.Errorf("current.goal = %v, want 'test goal'", current["goal"])
	}
	if current["state"] != "EXECUTING" {
		t.Errorf("current.state = %v, want 'EXECUTING'", current["state"])
	}
	if current["phase"] != float64(2) {
		t.Errorf("current.phase = %v, want 2", current["phase"])
	}

	recent := result["recent"].(map[string]interface{})
	decisions := recent["decisions"].([]interface{})
	if len(decisions) != 3 {
		t.Errorf("len(recent.decisions) = %d, want 3", len(decisions))
	}
	events := recent["events"].([]interface{})
	if len(events) != 5 {
		t.Errorf("len(recent.events) = %d, want 5", len(events))
	}
}

func TestResumeDashboardNoState(t *testing.T) {
	saveGlobalsCmd(t)
	var buf bytes.Buffer
	stdout = &buf
	var errBuf bytes.Buffer
	stderr = &errBuf

	s, tmpDir := newTestStoreCmd(t)
	defer os.RemoveAll(tmpDir)
	store = s

	rootCmd.SetArgs([]string{"resume-dashboard"})
	defer rootCmd.SetArgs([]string{})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("resume-dashboard returned error: %v", err)
	}

	envelope := parseEnvelopeCmd(t, buf.String())
	result := envelope["result"].(map[string]interface{})
	current := result["current"].(map[string]interface{})

	if current["state"] != "UNKNOWN" {
		t.Errorf("current.state = %v, want 'UNKNOWN'", current["state"])
	}
	if current["phase"] != float64(0) {
		t.Errorf("current.phase = %v, want 0", current["phase"])
	}
	if current["goal"] != "" {
		t.Errorf("current.goal = %v, want ''", current["goal"])
	}
}

func TestResumeDashboardWithMemory(t *testing.T) {
	saveGlobalsCmd(t)
	var buf bytes.Buffer
	stdout = &buf
	var errBuf bytes.Buffer
	stderr = &errBuf

	s, tmpDir := newTestStoreCmd(t)
	defer os.RemoveAll(tmpDir)
	store = s

	goal := "memory test"
	state := colony.ColonyState{
		Version:      "1.0",
		Goal:         &goal,
		State:        colony.StateREADY,
		CurrentPhase: 1,
	}
	if err := s.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatal(err)
	}

	ts1 := 0.8
	ts2 := 0.6
	learnings := colony.LearningFile{
		Observations: []colony.Observation{
			{ContentHash: "h1", Content: "obs1", TrustScore: &ts1},
			{ContentHash: "h2", Content: "obs2", TrustScore: &ts2},
			{ContentHash: "h3", Content: "obs3", TrustScore: nil},
		},
	}
	if err := s.SaveJSON("learning-observations.json", learnings); err != nil {
		t.Fatal(err)
	}

	if err := os.MkdirAll(s.BasePath()+"/midden", 0755); err != nil {
		t.Fatal(err)
	}
	midden := colony.MiddenFile{
		Entries: []colony.MiddenEntry{
			{ID: "m1", Category: "build", Message: "test failure", Timestamp: "2026-04-01T10:00:00Z"},
		},
	}
	if err := s.SaveJSON("midden/midden.json", midden); err != nil {
		t.Fatal(err)
	}

	rootCmd.SetArgs([]string{"resume-dashboard"})
	defer rootCmd.SetArgs([]string{})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("resume-dashboard returned error: %v", err)
	}

	envelope := parseEnvelopeCmd(t, buf.String())
	result := envelope["result"].(map[string]interface{})
	mh := result["memory_health"].(map[string]interface{})

	if mh["wisdom_count"] != float64(2) {
		t.Errorf("wisdom_count = %v, want 2", mh["wisdom_count"])
	}
	if mh["pending_promotions"] != float64(1) {
		t.Errorf("pending_promotions = %v, want 1", mh["pending_promotions"])
	}
	if mh["recent_failures"] != float64(1) {
		t.Errorf("recent_failures = %v, want 1", mh["recent_failures"])
	}
}

// --- context-capsule tests ---

func TestContextCapsule(t *testing.T) {
	saveGlobalsCmd(t)
	var buf bytes.Buffer
	stdout = &buf
	var errBuf bytes.Buffer
	stderr = &errBuf

	s, tmpDir := newTestStoreCmd(t)
	defer os.RemoveAll(tmpDir)
	store = s

	goal := "test colony goal"
	state := colony.ColonyState{
		Version:      "1.0",
		Goal:         &goal,
		State:        colony.StateREADY,
		CurrentPhase: 2,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{ID: 1, Name: "Foundation", Status: "completed"},
				{ID: 2, Name: "Core Features", Status: "in_progress"},
				{ID: 3, Name: "Polish", Status: "pending"},
			},
		},
		Memory: colony.Memory{
			Decisions: []colony.Decision{
				{ID: "d1", Phase: 1, Claim: "Use cobra", Rationale: "standard", Timestamp: "2026-04-01T10:00:00Z"},
			},
		},
	}
	if err := s.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatal(err)
	}

	rootCmd.SetArgs([]string{"context-capsule"})
	defer rootCmd.SetArgs([]string{})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("context-capsule returned error: %v", err)
	}

	envelope := parseEnvelopeCmd(t, buf.String())
	result := envelope["result"].(map[string]interface{})

	if result["exists"] != true {
		t.Errorf("exists = %v, want true", result["exists"])
	}
	if result["state"] != "READY" {
		t.Errorf("state = %v, want 'READY'", result["state"])
	}
	nextAction, ok := result["next_action"].(string)
	if !ok || nextAction == "" {
		t.Errorf("next_action is empty or not a string")
	}
	promptSection := result["prompt_section"].(string)
	if !strings.Contains(promptSection, "--- CONTEXT CAPSULE ---") {
		t.Error("prompt_section missing opening marker")
	}
	if !strings.Contains(promptSection, "--- END CONTEXT CAPSULE ---") {
		t.Error("prompt_section missing closing marker")
	}
	if !strings.Contains(promptSection, "test colony goal") {
		t.Error("prompt_section missing goal text")
	}
	wc := result["word_count"].(float64)
	if wc == 0 {
		t.Errorf("word_count = %v, want > 0", result["word_count"])
	}
}

func TestContextCapsuleNoState(t *testing.T) {
	saveGlobalsCmd(t)
	var buf bytes.Buffer
	stdout = &buf
	var errBuf bytes.Buffer
	stderr = &errBuf

	s, tmpDir := newTestStoreCmd(t)
	defer os.RemoveAll(tmpDir)
	store = s

	rootCmd.SetArgs([]string{"context-capsule"})
	defer rootCmd.SetArgs([]string{})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("context-capsule returned error: %v", err)
	}

	envelope := parseEnvelopeCmd(t, buf.String())
	result := envelope["result"].(map[string]interface{})

	if result["exists"] != false {
		t.Errorf("exists = %v, want false", result["exists"])
	}
	if result["word_count"] != float64(0) {
		t.Errorf("word_count = %v, want 0", result["word_count"])
	}
	if result["prompt_section"] != "" {
		t.Errorf("prompt_section = %v, want empty string", result["prompt_section"])
	}
}

func TestContextCapsuleCompact(t *testing.T) {
	saveGlobalsCmd(t)
	var buf bytes.Buffer
	stdout = &buf
	var errBuf bytes.Buffer
	stderr = &errBuf

	s, tmpDir := newTestStoreCmd(t)
	defer os.RemoveAll(tmpDir)
	store = s

	goal := "compact test goal"
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

	var summaryLines []string
	for i := 0; i < 100; i++ {
		summaryLines = append(summaryLines, fmt.Sprintf("2026-04-01T10:%02d:00Z|summary-%d|system|narrative entry number %d with extra words to fill the budget", i%60, i, i))
	}
	summaryData := []byte(strings.Join(summaryLines, "\n"))
	if err := s.AtomicWrite("rolling-summary.log", summaryData); err != nil {
		t.Fatal(err)
	}

	rootCmd.SetArgs([]string{"context-capsule", "--compact", "--max-words", "100"})
	defer rootCmd.SetArgs([]string{})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("context-capsule returned error: %v", err)
	}

	envelope := parseEnvelopeCmd(t, buf.String())
	result := envelope["result"].(map[string]interface{})
	promptSection := result["prompt_section"].(string)
	wc := int(result["word_count"].(float64))

	if strings.Contains(promptSection, "Recent narrative:") && wc > 100 {
		t.Errorf("compact mode did not trim narrative section; word_count=%d, prompt still contains narrative", wc)
	}
}

func TestContextCapsuleWithSignals(t *testing.T) {
	saveGlobalsCmd(t)
	var buf bytes.Buffer
	stdout = &buf
	var errBuf bytes.Buffer
	stderr = &errBuf

	s, tmpDir := newTestStoreCmd(t)
	defer os.RemoveAll(tmpDir)
	store = s

	goal := "signals test"
	now := time.Now().Format(time.RFC3339)
	state := colony.ColonyState{
		Version:      "1.0",
		Goal:         &goal,
		State:        colony.StateEXECUTING,
		CurrentPhase: 1,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{ID: 1, Name: "Testing", Status: "in_progress"},
			},
		},
	}
	if err := s.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatal(err)
	}

	strength1 := 0.9
	strength2 := 1.0
	strength3 := 0.5
	pf := colony.PheromoneFile{
		Signals: []colony.PheromoneSignal{
			{
				ID: "sig_1", Type: "FOCUS", Priority: "normal", Source: "user",
				CreatedAt: now, Active: true, Strength: &strength1,
				Content: json.RawMessage(`{"text": "Focus on testing"}`),
			},
			{
				ID: "sig_2", Type: "REDIRECT", Priority: "high", Source: "user",
				CreatedAt: now, Active: true, Strength: &strength2,
				Content: json.RawMessage(`{"text": "Avoid global state"}`),
			},
			{
				ID: "sig_3", Type: "FEEDBACK", Priority: "low", Source: "auto",
				CreatedAt: now, Active: true, Strength: &strength3,
				Content: json.RawMessage(`{"text": "Tests pass consistently"}`),
			},
		},
	}
	if err := s.SaveJSON("pheromones.json", pf); err != nil {
		t.Fatal(err)
	}

	rootCmd.SetArgs([]string{"context-capsule"})
	defer rootCmd.SetArgs([]string{})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("context-capsule returned error: %v", err)
	}

	envelope := parseEnvelopeCmd(t, buf.String())
	result := envelope["result"].(map[string]interface{})
	promptSection := result["prompt_section"].(string)

	if !strings.Contains(promptSection, "FOCUS:") {
		t.Errorf("prompt_section missing FOCUS signal: %s", promptSection)
	}
	if !strings.Contains(promptSection, "REDIRECT:") {
		t.Errorf("prompt_section missing REDIRECT signal: %s", promptSection)
	}
}

func TestContextCapsuleWithDecisions(t *testing.T) {
	saveGlobalsCmd(t)
	var buf bytes.Buffer
	stdout = &buf
	var errBuf bytes.Buffer
	stderr = &errBuf

	s, tmpDir := newTestStoreCmd(t)
	defer os.RemoveAll(tmpDir)
	store = s

	goal := "decisions test"
	state := colony.ColonyState{
		Version:      "1.0",
		Goal:         &goal,
		State:        colony.StateREADY,
		CurrentPhase: 2,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{ID: 1, Name: "One", Status: "completed"},
				{ID: 2, Name: "Two", Status: "in_progress"},
			},
		},
		Memory: colony.Memory{
			Decisions: []colony.Decision{
				{ID: "d1", Phase: 1, Claim: "First decision claim text", Rationale: "reason1", Timestamp: "2026-04-01T10:00:00Z"},
				{ID: "d2", Phase: 1, Claim: "Second decision claim text", Rationale: "reason2", Timestamp: "2026-04-01T11:00:00Z"},
				{ID: "d3", Phase: 1, Claim: "Third decision claim text", Rationale: "reason3", Timestamp: "2026-04-01T12:00:00Z"},
				{ID: "d4", Phase: 2, Claim: "Fourth decision claim text", Rationale: "reason4", Timestamp: "2026-04-01T13:00:00Z"},
				{ID: "d5", Phase: 2, Claim: "Fifth decision claim text", Rationale: "reason5", Timestamp: "2026-04-01T14:00:00Z"},
			},
		},
	}
	if err := s.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatal(err)
	}

	rootCmd.SetArgs([]string{"context-capsule", "--max-decisions", "3"})
	defer rootCmd.SetArgs([]string{})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("context-capsule returned error: %v", err)
	}

	envelope := parseEnvelopeCmd(t, buf.String())
	result := envelope["result"].(map[string]interface{})
	promptSection := result["prompt_section"].(string)

	decSectionStart := strings.Index(promptSection, "Recent decisions:")
	decSectionEnd := len(promptSection)
	for _, marker := range []string{"Open risks:", "Recent narrative:", "--- END"} {
		if idx := strings.Index(promptSection, marker); idx > decSectionStart && idx < decSectionEnd {
			decSectionEnd = idx
		}
	}
	if decSectionStart >= 0 {
		decSection := promptSection[decSectionStart:decSectionEnd]
		decLines := strings.Count(decSection, "- ")
		if decLines > 3 {
			t.Errorf("found %d decision entries, expected at most 3", decLines)
		}
	}
}

func TestContextCapsuleWithRisks(t *testing.T) {
	saveGlobalsCmd(t)
	var buf bytes.Buffer
	stdout = &buf
	var errBuf bytes.Buffer
	stderr = &errBuf

	s, tmpDir := newTestStoreCmd(t)
	defer os.RemoveAll(tmpDir)
	store = s

	goal := "risks test"
	state := colony.ColonyState{
		Version:      "1.0",
		Goal:         &goal,
		State:        colony.StateEXECUTING,
		CurrentPhase: 1,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{ID: 1, Name: "Testing", Status: "in_progress"},
			},
		},
	}
	if err := s.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatal(err)
	}

	phase := 1
	flags := colony.FlagsFile{
		Version: "1.0",
		Decisions: []colony.FlagEntry{
			{ID: "f1", Type: "blocker", Description: "Critical dependency missing", Phase: &phase, Source: "builder", CreatedAt: "2026-04-01T10:00:00Z", Resolved: false},
			{ID: "f2", Type: "issue", Description: "Slow test suite", Phase: &phase, Source: "watcher", CreatedAt: "2026-04-01T11:00:00Z", Resolved: false},
			{ID: "f3", Type: "note", Description: "Consider refactoring", Source: "scout", CreatedAt: "2026-04-01T12:00:00Z", Resolved: true},
		},
	}
	if err := s.SaveJSON("flags.json", flags); err != nil {
		t.Fatal(err)
	}

	rootCmd.SetArgs([]string{"context-capsule"})
	defer rootCmd.SetArgs([]string{})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("context-capsule returned error: %v", err)
	}

	envelope := parseEnvelopeCmd(t, buf.String())
	result := envelope["result"].(map[string]interface{})
	promptSection := result["prompt_section"].(string)

	if !strings.Contains(promptSection, "Open risks:") {
		t.Errorf("prompt_section missing 'Open risks:' section: %s", promptSection)
	}
	if !strings.Contains(promptSection, "Critical dependency missing") {
		t.Errorf("prompt_section missing risk description: %s", promptSection)
	}
}

func TestContextCapsuleTypedStruct(t *testing.T) {
	saveGlobalsCmd(t)
	var buf bytes.Buffer
	stdout = &buf
	var errBuf bytes.Buffer
	stderr = &errBuf

	s, tmpDir := newTestStoreCmd(t)
	defer os.RemoveAll(tmpDir)
	store = s

	goal := "typed struct test"
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

	rootCmd.SetArgs([]string{"context-capsule"})
	defer rootCmd.SetArgs([]string{})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("context-capsule returned error: %v", err)
	}

	envelope := parseEnvelopeCmd(t, buf.String())
	result := envelope["result"].(map[string]interface{})

	expectedKeys := []string{"exists", "state", "next_action", "word_count", "prompt_section", "goal", "phase", "total_phases", "phase_name"}
	for _, key := range expectedKeys {
		if _, found := result[key]; !found {
			t.Errorf("result missing typed struct key: %s", key)
		}
	}

	if result["exists"] != true {
		t.Errorf("result.exists = %v, want true (should be direct bool, not nested)", result["exists"])
	}

	_, isNested := result["result"]
	if isNested {
		t.Error("result contains a nested 'result' key -- should be flat typed struct")
	}
}

// --- pr-context tests ---

// setupHubDir creates a temporary hub directory and sets AETHER_HUB_DIR.
func setupHubDir(t *testing.T) string {
	t.Helper()
	hubDir := t.TempDir()
	os.MkdirAll(filepath.Join(hubDir, "hive"), 0755)
	os.MkdirAll(filepath.Join(hubDir, "eternal"), 0755)
	origHub := os.Getenv("AETHER_HUB_DIR")
	os.Setenv("AETHER_HUB_DIR", hubDir)
	t.Cleanup(func() {
		os.Setenv("AETHER_HUB_DIR", origHub)
	})
	return hubDir
}

func TestPRContext(t *testing.T) {
	saveGlobalsCmd(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf
	var errBuf bytes.Buffer
	stderr = &errBuf

	s, tmpDir := newTestStoreCmd(t)
	defer os.RemoveAll(tmpDir)
	store = s
	setupHubDir(t)

	// Create colony state
	goal := "test goal"
	now := time.Now().Format(time.RFC3339)
	state := colony.ColonyState{
		Version:      "1.0",
		Goal:         &goal,
		State:        colony.StateREADY,
		CurrentPhase: 1,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{ID: 1, Name: "Foundation", Status: "in_progress"},
				{ID: 2, Name: "Core", Status: "pending"},
			},
		},
		Memory: colony.Memory{
			Decisions: []colony.Decision{
				{ID: "d1", Phase: 1, Claim: "Use Cobra CLI", Rationale: "standard", Timestamp: now},
				{ID: "d2", Phase: 1, Claim: "Use typed structs", Rationale: "safety", Timestamp: now},
			},
			Instincts: []colony.Instinct{
				{ID: "i1", Trigger: "test fails", Action: "run again", Confidence: 0.8, Status: "active"},
			},
		},
		Events: []string{
			now + "|init|system|Colony initialized",
			now + "|build|system|Build started",
			now + "|complete|system|Phase completed",
		},
	}
	if err := s.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatal(err)
	}

	// Create pheromones
	s0_8 := 0.8
	s0_9 := 0.9
	s0_5 := 0.5
	pf := colony.PheromoneFile{
		Signals: []colony.PheromoneSignal{
			{ID: "s1", Type: "FOCUS", Priority: "normal", Source: "user", CreatedAt: now, Active: true, Strength: &s0_8, Content: json.RawMessage(`{"text": "Focus on testing"}`)},
			{ID: "s2", Type: "REDIRECT", Priority: "high", Source: "user", CreatedAt: now, Active: true, Strength: &s0_9, Content: json.RawMessage(`{"text": "Avoid global state"}`)},
			{ID: "s3", Type: "FEEDBACK", Priority: "low", Source: "auto", CreatedAt: now, Active: true, Strength: &s0_5, Content: json.RawMessage(`{"text": "Tests pass consistently"}`)},
		},
	}
	if err := s.SaveJSON("pheromones.json", pf); err != nil {
		t.Fatal(err)
	}

	// Create flags with a blocker
	phase := 1
	flags := colony.FlagsFile{
		Version: "1.0",
		Decisions: []colony.FlagEntry{
			{ID: "f1", Type: "blocker", Description: "Critical dependency missing", Phase: &phase, Source: "builder", CreatedAt: now, Resolved: false},
			{ID: "f2", Type: "issue", Description: "Slow test suite", Phase: &phase, Source: "watcher", CreatedAt: now, Resolved: true},
		},
	}
	if err := s.SaveJSON("flags.json", flags); err != nil {
		t.Fatal(err)
	}

	// Create midden entries
	if err := os.MkdirAll(s.BasePath()+"/midden", 0755); err != nil {
		t.Fatal(err)
	}
	midden := colony.MiddenFile{
		Entries: []colony.MiddenEntry{
			{ID: "m1", Timestamp: "2026-04-01T10:00:00Z", Category: "build", Source: "builder", Message: "Build failed on test"},
			{ID: "m2", Timestamp: "2026-04-01T11:00:00Z", Category: "test", Source: "watcher", Message: "Test timeout error"},
		},
	}
	if err := s.SaveJSON("midden/midden.json", midden); err != nil {
		t.Fatal(err)
	}

	// Create rolling summary
	summaryData := []byte("2026-04-01T10:00:00Z|init|system|Colony initialized\n2026-04-01T11:00:00Z|build|system|Build started\n2026-04-01T12:00:00Z|complete|system|Phase done\n2026-04-01T13:00:00Z|build|system|New build\n2026-04-01T14:00:00Z|event|system|Custom event")
	if err := s.AtomicWrite("rolling-summary.log", summaryData); err != nil {
		t.Fatal(err)
	}

	rootCmd.SetArgs([]string{"pr-context"})
	defer rootCmd.SetArgs([]string{})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("pr-context returned error: %v", err)
	}

	envelope := parseEnvelopeCmd(t, buf.String())
	if envelope["ok"] != true {
		t.Fatalf("expected ok=true, got %v", envelope["ok"])
	}

	result := envelope["result"].(map[string]interface{})

	// Check schema
	if result["schema"] != "pr-context-v1" {
		t.Errorf("schema = %v, want pr-context-v1", result["schema"])
	}

	// Check colony_state
	cs := result["colony_state"].(map[string]interface{})
	if cs["exists"] != true {
		t.Errorf("colony_state.exists = %v, want true", cs["exists"])
	}

	// Check signals
	sig := result["signals"].(map[string]interface{})
	if sig["count"] != float64(3) {
		t.Errorf("signals.count = %v, want 3", sig["count"])
	}

	// Check blockers
	blk := result["blockers"].(map[string]interface{})
	if blk["count"] != float64(1) {
		t.Errorf("blockers.count = %v, want 1", blk["count"])
	}

	// Check midden
	mid := result["midden"].(map[string]interface{})
	if mid["count"] != float64(2) {
		t.Errorf("midden.count = %v, want 2", mid["count"])
	}

	// Check fallbacks -- queen/hive may be missing since test hub is empty
	if fb, ok := result["fallbacks_used"]; ok && fb != nil {
		fallbacks := fb.([]interface{})
		fallbackStr := fmt.Sprintf("%v", fallbacks)
		// These sources should NOT have fallbacks since we created them
		if strings.Contains(fallbackStr, "colony_state") {
			t.Error("colony_state should not be in fallbacks (we created COLONY_STATE.json)")
		}
		if strings.Contains(fallbackStr, "pheromones") {
			t.Error("pheromones should not be in fallbacks (we created pheromones.json)")
		}
	}

	// Check prompt_section contains expected sections
	ps := result["prompt_section"].(string)
	if !strings.Contains(ps, "--- ACTIVE SIGNALS") {
		t.Error("prompt_section missing ACTIVE SIGNALS section")
	}
	if !strings.Contains(ps, "--- BLOCKERS (CRITICAL) ---") {
		t.Error("prompt_section missing BLOCKERS section")
	}
	if !strings.Contains(ps, "REDIRECT (HARD CONSTRAINTS)") {
		t.Error("prompt_section missing REDIRECT section")
	}
	if !strings.Contains(ps, "FOCUS (Active Guidance)") {
		t.Error("prompt_section missing FOCUS section")
	}
	if !strings.Contains(ps, "FEEDBACK (Adjustments)") {
		t.Error("prompt_section missing FEEDBACK section")
	}
}

func TestPRContextCompact(t *testing.T) {
	saveGlobalsCmd(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf
	var errBuf bytes.Buffer
	stderr = &errBuf

	s, tmpDir := newTestStoreCmd(t)
	defer os.RemoveAll(tmpDir)
	store = s
	setupHubDir(t)

	goal := "compact test goal"
	state := colony.ColonyState{
		Version:      "1.0",
		Goal:         &goal,
		State:        colony.StateEXECUTING,
		CurrentPhase: 1,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{ID: 1, Name: "Big Phase", Status: "in_progress"},
			},
		},
	}
	if err := s.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatal(err)
	}

	// Create a massive rolling summary to force trimming
	var summaryLines []string
	for i := 0; i < 500; i++ {
		summaryLines = append(summaryLines, fmt.Sprintf("2026-04-01T10:%02d:00Z|event|system|This is narrative entry number %d with lots of extra text to fill up the budget and force the compact mode to trim some sections out of the output", i%60, i))
	}
	summaryData := []byte(strings.Join(summaryLines, "\n"))
	if err := s.AtomicWrite("rolling-summary.log", summaryData); err != nil {
		t.Fatal(err)
	}

	rootCmd.SetArgs([]string{"pr-context", "--compact"})
	defer rootCmd.SetArgs([]string{})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("pr-context returned error: %v", err)
	}

	envelope := parseEnvelopeCmd(t, buf.String())
	result := envelope["result"].(map[string]interface{})

	if result["budget"] != float64(3000) {
		t.Errorf("budget = %v, want 3000", result["budget"])
	}

	charCount := int(result["char_count"].(float64))
	if charCount > 3200 { // some slack for edge cases
		t.Errorf("char_count = %d, want <= 3200 (budget 3000 + slack)", charCount)
	}

	trimmedSections := result["trimmed_sections"].([]interface{})
	if len(trimmedSections) == 0 && charCount > 3000 {
		t.Error("expected trimmed_sections to be non-empty when over budget")
	}
}

func TestPRContextNoSources(t *testing.T) {
	saveGlobalsCmd(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf
	var errBuf bytes.Buffer
	stderr = &errBuf

	s, tmpDir := newTestStoreCmd(t)
	defer os.RemoveAll(tmpDir)
	store = s
	setupHubDir(t)

	// Do NOT create any data files

	rootCmd.SetArgs([]string{"pr-context"})
	defer rootCmd.SetArgs([]string{})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("pr-context returned error: %v", err)
	}

	envelope := parseEnvelopeCmd(t, buf.String())
	result := envelope["result"].(map[string]interface{})

	// Should have fallbacks
	fallbacks := result["fallbacks_used"].([]interface{})
	fallbackStr := fmt.Sprintf("%v", fallbacks)
	if !strings.Contains(fallbackStr, "colony_state") {
		t.Errorf("fallbacks_used should contain 'colony_state', got: %v", fallbacks)
	}

	// colony_state should not exist
	cs := result["colony_state"].(map[string]interface{})
	if cs["exists"] != false {
		t.Errorf("colony_state.exists = %v, want false", cs["exists"])
	}

	ps := result["prompt_section"].(string)
	if strings.Contains(ps, "Active signals") {
		t.Error("prompt_section should not contain 'Active signals' when no data")
	}
	if strings.Contains(ps, "BLOCKERS") {
		t.Error("prompt_section should not contain 'BLOCKERS' when no data")
	}
}

func TestPRContextWithSignals(t *testing.T) {
	saveGlobalsCmd(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf
	var errBuf bytes.Buffer
	stderr = &errBuf

	s, tmpDir := newTestStoreCmd(t)
	defer os.RemoveAll(tmpDir)
	store = s
	setupHubDir(t)

	goal := "signals test"
	now := time.Now().Format(time.RFC3339)
	state := colony.ColonyState{
		Version:      "1.0",
		Goal:         &goal,
		State:        colony.StateEXECUTING,
		CurrentPhase: 1,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{ID: 1, Name: "Testing", Status: "in_progress"},
			},
		},
	}
	if err := s.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatal(err)
	}

	s0_8 := 0.8
	s0_9 := 0.9
	s0_5 := 0.5
	s1_0 := 1.0
	s0_7 := 0.7
	pf := colony.PheromoneFile{
		Signals: []colony.PheromoneSignal{
			{ID: "s1", Type: "REDIRECT", Priority: "high", Source: "user", CreatedAt: now, Active: true, Strength: &s0_9, Content: json.RawMessage(`{"text": "Avoid globals"}`)},
			{ID: "s2", Type: "REDIRECT", Priority: "high", Source: "user", CreatedAt: now, Active: true, Strength: &s1_0, Content: json.RawMessage(`{"text": "No shell scripts"}`)},
			{ID: "s3", Type: "FOCUS", Priority: "normal", Source: "user", CreatedAt: now, Active: true, Strength: &s0_8, Content: json.RawMessage(`{"text": "Focus on Go"}`)},
			{ID: "s4", Type: "FOCUS", Priority: "normal", Source: "user", CreatedAt: now, Active: true, Strength: &s0_7, Content: json.RawMessage(`{"text": "Focus on tests"}`)},
			{ID: "s5", Type: "FEEDBACK", Priority: "low", Source: "auto", CreatedAt: now, Active: true, Strength: &s0_5, Content: json.RawMessage(`{"text": "Good progress"}`)},
		},
	}
	if err := s.SaveJSON("pheromones.json", pf); err != nil {
		t.Fatal(err)
	}

	rootCmd.SetArgs([]string{"pr-context"})
	defer rootCmd.SetArgs([]string{})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("pr-context returned error: %v", err)
	}

	envelope := parseEnvelopeCmd(t, buf.String())
	result := envelope["result"].(map[string]interface{})
	sig := result["signals"].(map[string]interface{})

	redirects := sig["redirects"].([]interface{})
	if len(redirects) != 2 {
		t.Errorf("signals.redirects length = %d, want 2", len(redirects))
	}

	focus := sig["focus"].([]interface{})
	if len(focus) != 2 {
		t.Errorf("signals.focus length = %d, want 2", len(focus))
	}

	feedback := sig["feedback"].([]interface{})
	if len(feedback) != 1 {
		t.Errorf("signals.feedback length = %d, want 1", len(feedback))
	}

	ps := result["prompt_section"].(string)
	if !strings.Contains(ps, "REDIRECT (HARD CONSTRAINTS)") {
		t.Error("prompt_section missing REDIRECT section")
	}
	if !strings.Contains(ps, "FOCUS (Active Guidance)") {
		t.Error("prompt_section missing FOCUS section")
	}
	if !strings.Contains(ps, "FEEDBACK (Adjustments)") {
		t.Error("prompt_section missing FEEDBACK section")
	}
}

func TestPRContextWithMidden(t *testing.T) {
	saveGlobalsCmd(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf
	var errBuf bytes.Buffer
	stderr = &errBuf

	s, tmpDir := newTestStoreCmd(t)
	defer os.RemoveAll(tmpDir)
	store = s
	setupHubDir(t)

	goal := "midden test"
	state := colony.ColonyState{
		Version:      "1.0",
		Goal:         &goal,
		State:        colony.StateEXECUTING,
		CurrentPhase: 1,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{ID: 1, Name: "Testing", Status: "in_progress"},
			},
		},
	}
	if err := s.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatal(err)
	}

	if err := os.MkdirAll(s.BasePath()+"/midden", 0755); err != nil {
		t.Fatal(err)
	}
	midden := colony.MiddenFile{
		Entries: []colony.MiddenEntry{
			{ID: "m1", Timestamp: "2026-04-01T10:00:00Z", Category: "build", Source: "builder", Message: "Build failed due to missing dependency"},
			{ID: "m2", Timestamp: "2026-04-01T12:00:00Z", Category: "test", Source: "watcher", Message: "Test suite timeout in integration tests"},
			{ID: "m3", Timestamp: "2026-04-01T14:00:00Z", Category: "deploy", Source: "builder", Message: "Deploy script failed on staging"},
		},
	}
	if err := s.SaveJSON("midden/midden.json", midden); err != nil {
		t.Fatal(err)
	}

	rootCmd.SetArgs([]string{"pr-context"})
	defer rootCmd.SetArgs([]string{})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("pr-context returned error: %v", err)
	}

	envelope := parseEnvelopeCmd(t, buf.String())
	result := envelope["result"].(map[string]interface{})
	mid := result["midden"].(map[string]interface{})

	if mid["count"] != float64(3) {
		t.Errorf("midden.count = %v, want 3", mid["count"])
	}

	// Check that midden items are present
	items := mid["items"].([]interface{})
	if len(items) != 3 {
		t.Errorf("midden.items length = %d, want 3", len(items))
	}

	// Prompt section should contain midden-related data in the context
	ps := result["prompt_section"].(string)
	if ps == "" {
		t.Error("prompt_section is empty, expected content")
	}
}

func TestPRContextWithBlockers(t *testing.T) {
	saveGlobalsCmd(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf
	var errBuf bytes.Buffer
	stderr = &errBuf

	s, tmpDir := newTestStoreCmd(t)
	defer os.RemoveAll(tmpDir)
	store = s
	setupHubDir(t)

	goal := "blocker test"
	state := colony.ColonyState{
		Version:      "1.0",
		Goal:         &goal,
		State:        colony.StateEXECUTING,
		CurrentPhase: 1,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{ID: 1, Name: "Testing", Status: "in_progress"},
			},
		},
	}
	if err := s.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatal(err)
	}

	phase := 1
	flags := colony.FlagsFile{
		Version: "1.0",
		Decisions: []colony.FlagEntry{
			{ID: "f1", Type: "blocker", Description: "Cannot proceed without API key", Phase: &phase, Source: "builder", CreatedAt: "2026-04-01T10:00:00Z", Resolved: false},
			{ID: "f2", Type: "blocker", Description: "Database migration stuck", Phase: &phase, Source: "builder", CreatedAt: "2026-04-01T11:00:00Z", Resolved: false},
			{ID: "f3", Type: "issue", Description: "Minor UI glitch", Phase: &phase, Source: "watcher", CreatedAt: "2026-04-01T12:00:00Z", Resolved: true},
		},
	}
	if err := s.SaveJSON("flags.json", flags); err != nil {
		t.Fatal(err)
	}

	rootCmd.SetArgs([]string{"pr-context"})
	defer rootCmd.SetArgs([]string{})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("pr-context returned error: %v", err)
	}

	envelope := parseEnvelopeCmd(t, buf.String())
	result := envelope["result"].(map[string]interface{})
	blk := result["blockers"].(map[string]interface{})

	if blk["count"] != float64(2) {
		t.Errorf("blockers.count = %v, want 2", blk["count"])
	}

	ps := result["prompt_section"].(string)
	if !strings.Contains(ps, "--- BLOCKERS (CRITICAL) ---") {
		t.Error("prompt_section missing BLOCKERS section")
	}
	if !strings.Contains(ps, "Cannot proceed without API key") {
		t.Error("prompt_section missing blocker text")
	}
}

func TestPRContextBudgetTrim(t *testing.T) {
	saveGlobalsCmd(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf
	var errBuf bytes.Buffer
	stderr = &errBuf

	s, tmpDir := newTestStoreCmd(t)
	defer os.RemoveAll(tmpDir)
	store = s
	setupHubDir(t)

	goal := "budget trim test"
	now := time.Now().Format(time.RFC3339)
	state := colony.ColonyState{
		Version:      "1.0",
		Goal:         &goal,
		State:        colony.StateEXECUTING,
		CurrentPhase: 1,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{ID: 1, Name: "Testing", Status: "in_progress"},
			},
		},
		Memory: colony.Memory{
			Decisions: []colony.Decision{
				{ID: "d1", Phase: 1, Claim: "Decision one that is quite long to take up budget space and ensure the prompt section gets big enough", Rationale: "reason", Timestamp: now},
				{ID: "d2", Phase: 1, Claim: "Decision two that is also quite long to take up more budget space in the assembled prompt section", Rationale: "reason", Timestamp: now},
				{ID: "d3", Phase: 1, Claim: "Decision three to further increase the size of the prompt section for budget enforcement", Rationale: "reason", Timestamp: now},
			},
			PhaseLearnings: []colony.PhaseLearning{
				{
					ID: "pl1", Phase: 1, PhaseName: "Testing", Timestamp: now,
					Learnings: []colony.Learning{
						{Claim: "This is a very long learning claim that takes up significant space in the prompt section to help test the budget enforcement trimming logic properly", Status: "validated", Tested: true},
					},
				},
			},
		},
	}
	if err := s.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatal(err)
	}

	// Add pheromones for more content
	s0_8 := 0.8
	s0_9 := 0.9
	s0_5 := 0.5
	pf := colony.PheromoneFile{
		Signals: []colony.PheromoneSignal{
			{ID: "s1", Type: "FOCUS", Priority: "normal", Source: "user", CreatedAt: now, Active: true, Strength: &s0_8, Content: json.RawMessage(`{"text": "Focus on important work and make sure this signal text is long enough to contribute to the budget overflow"}`)},
			{ID: "s2", Type: "REDIRECT", Priority: "high", Source: "user", CreatedAt: now, Active: true, Strength: &s0_9, Content: json.RawMessage(`{"text": "Avoid doing the wrong thing and make this signal long enough to contribute to the budget overflow"}`)},
			{ID: "s3", Type: "FEEDBACK", Priority: "low", Source: "auto", CreatedAt: now, Active: true, Strength: &s0_5, Content: json.RawMessage(`{"text": "This is feedback that should be included in the prompt section and add to the overall character count"}`)},
		},
	}
	if err := s.SaveJSON("pheromones.json", pf); err != nil {
		t.Fatal(err)
	}

	// Create rolling summary with many long entries
	var summaryLines []string
	for i := 0; i < 100; i++ {
		summaryLines = append(summaryLines, fmt.Sprintf("2026-04-01T10:%02d:00Z|build|system|%s", i%60, fmt.Sprintf("This is a detailed narrative entry number %d that contains enough text to fill the character budget when combined with all other sections so the trimming logic gets exercised properly and removes lower priority sections first", i)))
	}
	summaryData := []byte(strings.Join(summaryLines, "\n"))
	if err := s.AtomicWrite("rolling-summary.log", summaryData); err != nil {
		t.Fatal(err)
	}

	// Add blockers to verify they survive
	phase := 1
	flags := colony.FlagsFile{
		Version: "1.0",
		Decisions: []colony.FlagEntry{
			{ID: "f1", Type: "blocker", Description: "Must survive trimming because blockers are highest priority and should never be removed from the prompt section", Phase: &phase, Source: "builder", CreatedAt: now, Resolved: false},
		},
	}
	if err := s.SaveJSON("flags.json", flags); err != nil {
		t.Fatal(err)
	}

	// Use compact mode (3000 budget) to force trimming with less content
	rootCmd.SetArgs([]string{"pr-context", "--compact"})
	defer rootCmd.SetArgs([]string{})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("pr-context returned error: %v", err)
	}

	envelope := parseEnvelopeCmd(t, buf.String())
	result := envelope["result"].(map[string]interface{})

	trimmedSections := result["trimmed_sections"].([]interface{})
	if len(trimmedSections) == 0 {
		t.Error("expected trimmed_sections to be non-empty when content exceeds budget")
	}

	// Check that rolling summary was trimmed (it's first in trim order)
	trimmedStr := fmt.Sprintf("%v", trimmedSections)
	if !strings.Contains(trimmedStr, "ROLLING SUMMARY") {
		t.Errorf("expected ROLLING SUMMARY in trimmed_sections, got: %v", trimmedStr)
	}

	ps := result["prompt_section"].(string)

	// Verify the trimmed section is absent from prompt_section
	if strings.Contains(ps, "--- ROLLING SUMMARY ---") {
		t.Error("ROLLING SUMMARY should have been trimmed from prompt_section")
	}
}

func TestPRContextBlockersNeverTrimmed(t *testing.T) {
	saveGlobalsCmd(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf
	var errBuf bytes.Buffer
	stderr = &errBuf

	s, tmpDir := newTestStoreCmd(t)
	defer os.RemoveAll(tmpDir)
	store = s
	setupHubDir(t)

	goal := "blockers never trimmed test"
	now := time.Now().Format(time.RFC3339)
	state := colony.ColonyState{
		Version:      "1.0",
		Goal:         &goal,
		State:        colony.StateEXECUTING,
		CurrentPhase: 1,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{ID: 1, Name: "Testing", Status: "in_progress"},
			},
		},
		Memory: colony.Memory{
			Decisions: []colony.Decision{
				{ID: "d1", Phase: 1, Claim: "A decision that should get trimmed before blockers since blockers have highest priority", Rationale: "reason", Timestamp: now},
				{ID: "d2", Phase: 1, Claim: "Another decision that should get trimmed before blockers since blockers have highest priority", Rationale: "reason", Timestamp: now},
				{ID: "d3", Phase: 1, Claim: "Third decision to fill up more space in the prompt section and force trimming", Rationale: "reason", Timestamp: now},
			},
		},
	}
	if err := s.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatal(err)
	}

	// Create massive content to force trimming
	var summaryLines []string
	for i := 0; i < 500; i++ {
		summaryLines = append(summaryLines, fmt.Sprintf("2026-04-01T10:%02d:00Z|event|system|narrative entry number %d with lots of additional text to fill the character budget well beyond the default limit of six thousand characters so that the budget enforcement trimming logic gets exercised and we can verify blockers survive", i%60, i))
	}
	summaryData := []byte(strings.Join(summaryLines, "\n"))
	if err := s.AtomicWrite("rolling-summary.log", summaryData); err != nil {
		t.Fatal(err)
	}

	// Add pheromones for more content
	s0_8 := 0.8
	pf := colony.PheromoneFile{
		Signals: []colony.PheromoneSignal{
			{ID: "s1", Type: "FOCUS", Priority: "normal", Source: "user", CreatedAt: now, Active: true, Strength: &s0_8, Content: json.RawMessage(`{"text": "Focus on important work"}`)},
		},
	}
	if err := s.SaveJSON("pheromones.json", pf); err != nil {
		t.Fatal(err)
	}

	// Add blockers
	phase := 1
	flags := colony.FlagsFile{
		Version: "1.0",
		Decisions: []colony.FlagEntry{
			{ID: "f1", Type: "blocker", Description: "Critical blocker that must never be trimmed from the prompt section regardless of budget pressure", Phase: &phase, Source: "builder", CreatedAt: now, Resolved: false},
		},
	}
	if err := s.SaveJSON("flags.json", flags); err != nil {
		t.Fatal(err)
	}

	// Use compact mode for even tighter budget
	rootCmd.SetArgs([]string{"pr-context", "--compact"})
	defer rootCmd.SetArgs([]string{})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("pr-context returned error: %v", err)
	}

	envelope := parseEnvelopeCmd(t, buf.String())
	result := envelope["result"].(map[string]interface{})

	ps := result["prompt_section"].(string)
	if !strings.Contains(ps, "--- BLOCKERS (CRITICAL) ---") {
		t.Error("BLOCKERS section must survive budget trimming")
	}
	if !strings.Contains(ps, "Critical blocker that must never be trimmed") {
		t.Error("Blocker text must survive budget trimming")
	}

	// Verify BLOCKERS is not in trimmed_sections
	trimmedSections := result["trimmed_sections"].([]interface{})
	for _, ts := range trimmedSections {
		trimmed := ts.(string)
		if strings.Contains(trimmed, "BLOCKERS") {
			t.Errorf("BLOCKERS should never be in trimmed_sections, found: %s", trimmed)
		}
	}
}

// TestPRContextUsesSessionCache verifies that pr-context creates a SessionCache
// and that cache files (.cache_*) are created for the JSON files it loads.
func TestPRContextUsesSessionCache(t *testing.T) {
	saveGlobalsCmd(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf
	var errBuf bytes.Buffer
	stderr = &errBuf

	s, tmpDir := newTestStoreCmd(t)
	defer os.RemoveAll(tmpDir)
	store = s
	setupHubDir(t)

	goal := "cache test"
	now := time.Now().Format(time.RFC3339)
	state := colony.ColonyState{
		Version:      "1.0",
		Goal:         &goal,
		State:        colony.StateREADY,
		CurrentPhase: 1,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{ID: 1, Name: "Testing", Status: "in_progress"},
			},
		},
		Memory: colony.Memory{
			Decisions: []colony.Decision{
				{ID: "d1", Phase: 1, Claim: "Use cache", Rationale: "perf", Timestamp: now},
			},
		},
	}
	if err := s.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatal(err)
	}

	s0_8 := 0.8
	pf := colony.PheromoneFile{
		Signals: []colony.PheromoneSignal{
			{ID: "s1", Type: "FOCUS", Priority: "normal", Source: "user", CreatedAt: now, Active: true, Strength: &s0_8, Content: json.RawMessage(`{"text": "Focus on caching"}`)},
		},
	}
	if err := s.SaveJSON("pheromones.json", pf); err != nil {
		t.Fatal(err)
	}

	rootCmd.SetArgs([]string{"pr-context"})
	defer rootCmd.SetArgs([]string{})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("pr-context returned error: %v", err)
	}

	// Verify that cache files were created for the loaded JSON files
	dataDir := s.BasePath()

	// COLONY_STATE.json should have a cache file
	cacheFile := filepath.Join(dataDir, ".cache_COLONY_STATE.json")
	if _, err := os.Stat(cacheFile); os.IsNotExist(err) {
		t.Errorf("expected cache file %s to be created, but it does not exist", cacheFile)
	}

	// pheromones.json should have a cache file
	pheromonesCache := filepath.Join(dataDir, ".cache_pheromones.json")
	if _, err := os.Stat(pheromonesCache); os.IsNotExist(err) {
		t.Errorf("expected cache file %s to be created, but it does not exist", pheromonesCache)
	}
}
