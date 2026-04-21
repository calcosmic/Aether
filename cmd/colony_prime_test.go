package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

// colonyPrimeTestEnv creates a minimal colony state + optional pheromones
// and returns the store, temp dir, and a cleanup function.
func colonyPrimeTestEnv(t *testing.T, pheromones *colony.PheromoneFile) (*colony.ColonyState, *colony.PheromoneFile) {
	t.Helper()

	goal := "test pheromone injection"
	state := &colony.ColonyState{
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

	if pheromones == nil {
		pheromones = &colony.PheromoneFile{Signals: []colony.PheromoneSignal{}}
	}

	return state, pheromones
}

// runColonyPrime executes colony-prime with the given flags and returns parsed output.
func runColonyPrime(t *testing.T, s interface {
	SaveJSON(string, interface{}) error
}, flags []string) map[string]interface{} {
	t.Helper()
	var buf bytes.Buffer
	stdout = &buf
	var errBuf bytes.Buffer
	stderr = &errBuf

	args := append([]string{"colony-prime"}, flags...)
	rootCmd.SetArgs(args)

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("colony-prime returned error: %v (stderr: %s)", err, errBuf.String())
	}

	return parseEnvelopeCmd(t, buf.String())
}

// --- Test: colony-prime includes pheromone signals in assembled context ---

func TestColonyPrime_IncludesPheromoneSignals(t *testing.T) {
	saveGlobalsCmd(t)
	resetRootCmd(t)

	s, tmpDir := newTestStoreCmd(t)
	defer os.RemoveAll(tmpDir)
	store = s

	state, pheromones := colonyPrimeTestEnv(t, nil)
	now := time.Now().Format(time.RFC3339)
	s0_9 := 0.9

	pheromones.Signals = []colony.PheromoneSignal{
		{ID: "s1", Type: "FOCUS", Priority: "normal", Source: "user", CreatedAt: now, Active: true, Strength: &s0_9, Content: json.RawMessage(`{"text": "Focus on error handling"}`)},
		{ID: "s2", Type: "REDIRECT", Priority: "high", Source: "user", CreatedAt: now, Active: true, Strength: &s0_9, Content: json.RawMessage(`{"text": "Avoid global state"}`)},
		{ID: "s3", Type: "FEEDBACK", Priority: "low", Source: "auto", CreatedAt: now, Active: true, Strength: &s0_9, Content: json.RawMessage(`{"text": "Tests looking good"}`)},
	}

	if err := s.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatal(err)
	}
	if err := s.SaveJSON("pheromones.json", pheromones); err != nil {
		t.Fatal(err)
	}

	envelope := runColonyPrime(t, s, nil)
	result := envelope["result"].(map[string]interface{})
	contextStr := result["context"].(string)

	// Verify the pheromone section header appears
	if !strings.Contains(contextStr, "## Pheromone Signals") {
		t.Error("colony-prime context missing '## Pheromone Signals' section header")
	}
	if !strings.Contains(contextStr, "Colony is EXECUTING. Signals are active implementation constraints.") {
		t.Error("colony-prime context missing EXECUTING lifecycle framing")
	}

	// Verify all three signal type markers appear in the output
	for _, sigType := range []string{"FOCUS", "REDIRECT", "FEEDBACK"} {
		if !strings.Contains(contextStr, sigType) {
			t.Errorf("colony-prime context missing signal type marker '%s'", sigType)
		}
	}

	// Verify the actual signal content text appears
	if !strings.Contains(contextStr, "Focus on error handling") {
		t.Error("colony-prime context missing FOCUS signal content 'Focus on error handling'")
	}
	if !strings.Contains(contextStr, "Avoid global state") {
		t.Error("colony-prime context missing REDIRECT signal content 'Avoid global state'")
	}
	if !strings.Contains(contextStr, "Tests looking good") {
		t.Error("colony-prime context missing FEEDBACK signal content 'Tests looking good'")
	}

	// Verify the format: each signal should be formatted as [TYPE] content
	if !strings.Contains(contextStr, "[FOCUS] Focus on error handling") {
		t.Error("colony-prime context missing formatted FOCUS signal '[FOCUS] Focus on error handling'")
	}
	if !strings.Contains(contextStr, "[REDIRECT] Avoid global state") {
		t.Error("colony-prime context missing formatted REDIRECT signal '[REDIRECT] Avoid global state'")
	}
	if !strings.Contains(contextStr, "[FEEDBACK] Tests looking good") {
		t.Error("colony-prime context missing formatted FEEDBACK signal '[FEEDBACK] Tests looking good'")
	}

	// Verify sections count reflects pheromone section being present
	sections := result["sections"].(float64)
	if sections < 2 {
		t.Errorf("expected at least 2 sections (state + pheromones), got %f", sections)
	}
}

func TestColonyPrime_IncludesResolvedClarifiedIntent(t *testing.T) {
	saveGlobalsCmd(t)
	resetRootCmd(t)

	s, tmpDir := newTestStoreCmd(t)
	defer os.RemoveAll(tmpDir)
	store = s

	state, pheromones := colonyPrimeTestEnv(t, nil)
	if err := s.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatal(err)
	}
	if err := s.SaveJSON("pheromones.json", pheromones); err != nil {
		t.Fatal(err)
	}
	if err := s.SaveJSON(pendingDecisionsFile, PendingDecisionFile{
		Decisions: []PendingDecision{
			{
				ID:          "pd_1",
				Type:        clarificationDecisionType,
				Description: formatClarificationDescription("Which existing surface should own the first implementation slice?", []string{"admin-app", "new-module", "research-first"}),
				Source:      discussSource("surface", true),
				Resolution:  "Use the existing admin-app surface",
				Resolved:    true,
				CreatedAt:   "2026-04-19T10:00:00Z",
				ResolvedAt:  "2026-04-19T10:05:00Z",
			},
			{
				ID:          "pd_2",
				Type:        clarificationDecisionType,
				Description: formatClarificationDescription("What verification bar do you want on the first pass?", []string{"focused tests", "broad coverage", "prototype first"}),
				Source:      discussSource("verification", false),
				Resolved:    false,
				CreatedAt:   "2026-04-19T10:06:00Z",
			},
		},
	}); err != nil {
		t.Fatal(err)
	}

	envelope := runColonyPrime(t, s, nil)
	result := envelope["result"].(map[string]interface{})
	contextStr := result["context"].(string)

	if !strings.Contains(contextStr, "## CLARIFIED INTENT") {
		t.Fatal("colony-prime context missing clarified intent section")
	}
	if !strings.Contains(contextStr, "Which existing surface should own the first implementation slice? => Use the existing admin-app surface") {
		t.Fatal("colony-prime context missing resolved clarification content")
	}
	if strings.Contains(contextStr, "What verification bar do you want on the first pass?") {
		t.Fatal("colony-prime context should not include unresolved clarification questions")
	}
}

func TestColonyPrime_LifecycleContextChangesWithState(t *testing.T) {
	cases := []struct {
		name         string
		state        colony.State
		currentPhase int
		phases       []colony.Phase
		want         string
		flags        []string
	}{
		{
			name:         "ready without plan",
			state:        colony.StateREADY,
			currentPhase: 0,
			phases:       nil,
			want:         "Colony is READY. Signals should guide planning scope and approach.",
		},
		{
			name:  "ready with plan",
			state: colony.StateREADY,
			phases: []colony.Phase{
				{ID: 1, Name: "Phase One", Status: colony.PhaseReady},
			},
			want: "Colony is READY. Signals are pre-build guidance for upcoming execution.",
		},
		{
			name:  "built",
			state: colony.StateBUILT,
			phases: []colony.Phase{
				{ID: 1, Name: "Phase One", Status: colony.PhaseCompleted},
			},
			want: "Colony is BUILT. Signals guide verification and learning extraction.",
		},
		{
			name:  "executing compact",
			state: colony.StateEXECUTING,
			phases: []colony.Phase{
				{ID: 1, Name: "Phase One", Status: colony.PhaseInProgress},
			},
			want:  "Colony is EXECUTING. Signals are active implementation constraints.",
			flags: []string{"--compact"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			saveGlobalsCmd(t)
			resetRootCmd(t)

			s, tmpDir := newTestStoreCmd(t)
			defer os.RemoveAll(tmpDir)
			store = s

			goal := "lifecycle framing test"
			currentPhase := tc.currentPhase
			if currentPhase == 0 && len(tc.phases) > 0 {
				currentPhase = 1
			}

			state := colony.ColonyState{
				Version:      "1.0",
				Goal:         &goal,
				State:        tc.state,
				CurrentPhase: currentPhase,
				Plan: colony.Plan{
					Phases: tc.phases,
				},
			}

			now := time.Now().Format(time.RFC3339)
			s0_9 := 0.9
			pheromones := colony.PheromoneFile{
				Signals: []colony.PheromoneSignal{
					{ID: "s1", Type: "FOCUS", Priority: "normal", Source: "user", CreatedAt: now, Active: true, Strength: &s0_9, Content: json.RawMessage(`{"text": "Focus on lifecycle framing"}`)},
				},
			}

			if err := s.SaveJSON("COLONY_STATE.json", state); err != nil {
				t.Fatal(err)
			}
			if err := s.SaveJSON("pheromones.json", pheromones); err != nil {
				t.Fatal(err)
			}

			envelope := runColonyPrime(t, s, tc.flags)
			result := envelope["result"].(map[string]interface{})
			contextStr := result["context"].(string)
			promptSection := result["prompt_section"].(string)

			if !strings.Contains(contextStr, tc.want) {
				t.Fatalf("context missing lifecycle framing %q\ncontext:\n%s", tc.want, contextStr)
			}
			if !strings.Contains(promptSection, tc.want) {
				t.Fatalf("prompt_section missing lifecycle framing %q\nprompt_section:\n%s", tc.want, promptSection)
			}
		})
	}
}

// --- Test: colony-prime with no pheromones shows no pheromone section ---

func TestColonyPrime_NoPheromones_NoPheromoneSection(t *testing.T) {
	saveGlobalsCmd(t)
	resetRootCmd(t)

	s, tmpDir := newTestStoreCmd(t)
	defer os.RemoveAll(tmpDir)
	store = s

	state, _ := colonyPrimeTestEnv(t, nil)

	if err := s.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatal(err)
	}

	envelope := runColonyPrime(t, s, nil)
	result := envelope["result"].(map[string]interface{})
	contextStr := result["context"].(string)

	// Context should contain the goal but no pheromone section
	if !strings.Contains(contextStr, "test pheromone injection") {
		t.Error("colony-prime context missing colony goal text")
	}
	if strings.Contains(contextStr, "Pheromone Signals") {
		t.Error("colony-prime context should NOT contain 'Pheromone Signals' when no pheromones exist")
	}
}

func TestColonyPrime_ExcludesExpiredSignals(t *testing.T) {
	saveGlobalsCmd(t)
	resetRootCmd(t)

	s, tmpDir := newTestStoreCmd(t)
	defer os.RemoveAll(tmpDir)
	store = s

	state, pheromones := colonyPrimeTestEnv(t, nil)
	now := time.Now().UTC()
	createdAt := now.Add(-24 * time.Hour).Format(time.RFC3339)
	expiredAt := now.Add(-1 * time.Hour).Format(time.RFC3339)
	liveStrength := 0.9

	pheromones.Signals = []colony.PheromoneSignal{
		{
			ID:        "expired",
			Type:      "REDIRECT",
			Priority:  "high",
			Source:    "user",
			CreatedAt: createdAt,
			ExpiresAt: &expiredAt,
			Active:    true,
			Strength:  &liveStrength,
			Content:   json.RawMessage(`{"text": "Expired redirect"}`),
		},
		{
			ID:        "live",
			Type:      "FOCUS",
			Priority:  "normal",
			Source:    "user",
			CreatedAt: createdAt,
			Active:    true,
			Strength:  &liveStrength,
			Content:   json.RawMessage(`{"text": "Current focus"}`),
		},
	}

	if err := s.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatal(err)
	}
	if err := s.SaveJSON("pheromones.json", pheromones); err != nil {
		t.Fatal(err)
	}

	envelope := runColonyPrime(t, s, nil)
	result := envelope["result"].(map[string]interface{})
	if got := int(result["signal_count"].(float64)); got != 1 {
		t.Fatalf("signal_count = %d, want 1", got)
	}
	contextStr := result["context"].(string)
	if strings.Contains(contextStr, "Expired redirect") {
		t.Fatalf("expired signal should not appear in context:\n%s", contextStr)
	}
	if !strings.Contains(contextStr, "Current focus") {
		t.Fatalf("live signal missing from context:\n%s", contextStr)
	}
}

// --- Test: colony-prime with only inactive signals shows no pheromone section ---

func TestColonyPrime_OnlyInactiveSignals_NoPheromoneSection(t *testing.T) {
	saveGlobalsCmd(t)
	resetRootCmd(t)

	s, tmpDir := newTestStoreCmd(t)
	defer os.RemoveAll(tmpDir)
	store = s

	state, _ := colonyPrimeTestEnv(t, nil)
	now := time.Now().Format(time.RFC3339)
	s0_9 := 0.9

	pf := colony.PheromoneFile{
		Signals: []colony.PheromoneSignal{
			{ID: "s1", Type: "FOCUS", Priority: "normal", Source: "user", CreatedAt: now, Active: false, Strength: &s0_9, Content: json.RawMessage(`{"text": "Inactive focus"}`)},
			{ID: "s2", Type: "REDIRECT", Priority: "high", Source: "user", CreatedAt: now, Active: false, Strength: &s0_9, Content: json.RawMessage(`{"text": "Inactive redirect"}`)},
		},
	}

	if err := s.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatal(err)
	}
	if err := s.SaveJSON("pheromones.json", pf); err != nil {
		t.Fatal(err)
	}

	envelope := runColonyPrime(t, s, nil)
	result := envelope["result"].(map[string]interface{})
	contextStr := result["context"].(string)

	// Inactive signals should not appear in context
	if strings.Contains(contextStr, "Inactive focus") {
		t.Error("colony-prime context should not contain inactive signal content 'Inactive focus'")
	}
	if strings.Contains(contextStr, "Inactive redirect") {
		t.Error("colony-prime context should not contain inactive signal content 'Inactive redirect'")
	}
	if strings.Contains(contextStr, "Pheromone Signals") {
		t.Error("colony-prime context should not contain 'Pheromone Signals' section when all signals are inactive")
	}
}

// --- Test: colony-prime with mixed active and inactive signals includes only active ---

func TestColonyPrime_MixedActiveInactive_OnlyActiveIncluded(t *testing.T) {
	saveGlobalsCmd(t)
	resetRootCmd(t)

	s, tmpDir := newTestStoreCmd(t)
	defer os.RemoveAll(tmpDir)
	store = s

	state, _ := colonyPrimeTestEnv(t, nil)
	now := time.Now().Format(time.RFC3339)
	s0_9 := 0.9

	pf := colony.PheromoneFile{
		Signals: []colony.PheromoneSignal{
			{ID: "s1", Type: "FOCUS", Priority: "normal", Source: "user", CreatedAt: now, Active: true, Strength: &s0_9, Content: json.RawMessage(`{"text": "Active focus signal"}`)},
			{ID: "s2", Type: "REDIRECT", Priority: "high", Source: "user", CreatedAt: now, Active: false, Strength: &s0_9, Content: json.RawMessage(`{"text": "Hidden redirect"}`)},
			{ID: "s3", Type: "FEEDBACK", Priority: "low", Source: "auto", CreatedAt: now, Active: true, Strength: &s0_9, Content: json.RawMessage(`{"text": "Active feedback signal"}`)},
		},
	}

	if err := s.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatal(err)
	}
	if err := s.SaveJSON("pheromones.json", pf); err != nil {
		t.Fatal(err)
	}

	envelope := runColonyPrime(t, s, nil)
	result := envelope["result"].(map[string]interface{})
	contextStr := result["context"].(string)

	// Active signals should be present
	if !strings.Contains(contextStr, "Active focus signal") {
		t.Error("colony-prime context missing active FOCUS signal content")
	}
	if !strings.Contains(contextStr, "Active feedback signal") {
		t.Error("colony-prime context missing active FEEDBACK signal content")
	}

	// Inactive signal should NOT be present
	if strings.Contains(contextStr, "Hidden redirect") {
		t.Error("colony-prime context should not contain inactive REDIRECT signal 'Hidden redirect'")
	}
}

// --- Test: colony-prime --compact flag works with pheromones ---

func TestColonyPrime_CompactFlag_IncludesPheromones(t *testing.T) {
	saveGlobalsCmd(t)
	resetRootCmd(t)

	s, tmpDir := newTestStoreCmd(t)
	defer os.RemoveAll(tmpDir)
	store = s

	state, _ := colonyPrimeTestEnv(t, nil)
	now := time.Now().Format(time.RFC3339)
	s0_9 := 0.9

	pf := colony.PheromoneFile{
		Signals: []colony.PheromoneSignal{
			{ID: "s1", Type: "FOCUS", Priority: "normal", Source: "user", CreatedAt: now, Active: true, Strength: &s0_9, Content: json.RawMessage(`{"text": "Compact mode focus"}`)},
		},
	}

	if err := s.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatal(err)
	}
	if err := s.SaveJSON("pheromones.json", pf); err != nil {
		t.Fatal(err)
	}

	envelope := runColonyPrime(t, s, []string{"--compact"})
	result := envelope["result"].(map[string]interface{})
	contextStr := result["context"].(string)
	budget := result["budget"].(float64)

	// Verify compact uses 4000 char budget
	if budget != 4000 {
		t.Errorf("expected budget=4000 with --compact, got %f", budget)
	}

	// Verify pheromones still appear in compact mode
	if !strings.Contains(contextStr, "Pheromone Signals") {
		t.Error("colony-prime --compact context missing 'Pheromone Signals' section")
	}
	if !strings.Contains(contextStr, "Compact mode focus") {
		t.Error("colony-prime --compact context missing signal content 'Compact mode focus'")
	}
}

// --- Test: colony-prime excludes expired pheromones during prompt reads ---

func TestColonyPrime_OldSignalsAreFiltered(t *testing.T) {
	saveGlobalsCmd(t)
	resetRootCmd(t)

	s, tmpDir := newTestStoreCmd(t)
	defer os.RemoveAll(tmpDir)
	store = s

	state, _ := colonyPrimeTestEnv(t, nil)
	// Create a signal from 60 days ago
	oldTime := time.Now().Add(-60 * 24 * time.Hour).Format(time.RFC3339)
	s0_9 := 0.9

	pf := colony.PheromoneFile{
		Signals: []colony.PheromoneSignal{
			{ID: "s1", Type: "FOCUS", Priority: "normal", Source: "user", CreatedAt: oldTime, Active: true, Strength: &s0_9, Content: json.RawMessage(`{"text": "Old but active focus"}`)},
		},
	}

	if err := s.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatal(err)
	}
	if err := s.SaveJSON("pheromones.json", pf); err != nil {
		t.Fatal(err)
	}

	envelope := runColonyPrime(t, s, nil)
	result := envelope["result"].(map[string]interface{})
	contextStr := result["context"].(string)

	if strings.Contains(contextStr, "Old but active focus") {
		t.Error("colony-prime should exclude expired or decayed signals from worker context")
	}
}

// --- Test: colony-prime default budget is 8000 ---

func TestColonyPrime_DefaultBudget(t *testing.T) {
	saveGlobalsCmd(t)
	resetRootCmd(t)

	s, tmpDir := newTestStoreCmd(t)
	defer os.RemoveAll(tmpDir)
	store = s

	state, _ := colonyPrimeTestEnv(t, nil)

	if err := s.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatal(err)
	}

	envelope := runColonyPrime(t, s, nil)
	result := envelope["result"].(map[string]interface{})
	budget := result["budget"].(float64)

	if budget != 8000 {
		t.Errorf("expected default budget=8000, got %f", budget)
	}
}

// --- Test: colony-prime includes parallel mode defaulting to in-repo when unset ---

func TestColonyPrime_ParallelModeDefault(t *testing.T) {
	saveGlobalsCmd(t)
	resetRootCmd(t)

	s, tmpDir := newTestStoreCmd(t)
	defer os.RemoveAll(tmpDir)
	store = s

	state, _ := colonyPrimeTestEnv(t, nil)
	// ParallelMode is empty (zero value) -- should default to "in-repo"

	if err := s.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatal(err)
	}

	envelope := runColonyPrime(t, s, nil)
	result := envelope["result"].(map[string]interface{})
	contextStr := result["context"].(string)

	if !strings.Contains(contextStr, "Parallel Mode: in-repo") {
		t.Error("colony-prime context missing 'Parallel Mode: in-repo' when parallel_mode is unset")
	}
}

// --- Test: colony-prime includes parallel mode worktree when set ---

func TestColonyPrime_ParallelModeWorktree(t *testing.T) {
	saveGlobalsCmd(t)
	resetRootCmd(t)

	s, tmpDir := newTestStoreCmd(t)
	defer os.RemoveAll(tmpDir)
	store = s

	state, _ := colonyPrimeTestEnv(t, nil)
	state.ParallelMode = colony.ModeWorktree

	if err := s.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatal(err)
	}

	envelope := runColonyPrime(t, s, nil)
	result := envelope["result"].(map[string]interface{})
	contextStr := result["context"].(string)

	if !strings.Contains(contextStr, "Parallel Mode: worktree") {
		t.Error("colony-prime context missing 'Parallel Mode: worktree' when parallel_mode is set to worktree")
	}
	if strings.Contains(contextStr, "Parallel Mode: in-repo") {
		t.Error("colony-prime context should NOT show 'in-repo' when parallel_mode is set to worktree")
	}
}

// --- Test: colony-prime output structure is valid ---

func TestColonyPrime_OutputStructure(t *testing.T) {
	saveGlobalsCmd(t)
	resetRootCmd(t)

	s, tmpDir := newTestStoreCmd(t)
	defer os.RemoveAll(tmpDir)
	store = s

	state, _ := colonyPrimeTestEnv(t, nil)
	now := time.Now().Format(time.RFC3339)
	s0_9 := 0.9
	state.Memory.Instincts = []colony.Instinct{
		{
			ID:         "inst1",
			Trigger:    "old signal pileup",
			Action:     "prune low-value signals",
			Confidence: 0.9,
			Status:     "active",
			Domain:     "go",
			Source:     "test",
			CreatedAt:  now,
		},
	}

	pf := colony.PheromoneFile{
		Signals: []colony.PheromoneSignal{
			{ID: "s1", Type: "FOCUS", Priority: "normal", Source: "user", CreatedAt: now, Active: true, Strength: &s0_9, Content: json.RawMessage(`{"text": "Structure test"}`)},
		},
	}

	if err := s.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatal(err)
	}
	if err := s.SaveJSON("pheromones.json", pf); err != nil {
		t.Fatal(err)
	}

	envelope := runColonyPrime(t, s, nil)

	// Envelope must have ok:true
	if envelope["ok"] != true {
		t.Errorf("expected ok=true, got %v", envelope["ok"])
	}

	result := envelope["result"].(map[string]interface{})

	// Required fields
	requiredFields := []string{"context", "prompt_section", "signal_count", "instinct_count", "log_line", "budget", "used", "sections", "trimmed", "ledger"}
	for _, field := range requiredFields {
		if _, exists := result[field]; !exists {
			t.Errorf("colony-prime result missing required field '%s'", field)
		}
	}

	// context must be a non-empty string
	contextStr, ok := result["context"].(string)
	if !ok || contextStr == "" {
		t.Error("colony-prime result.context must be a non-empty string")
	}
	if promptSection, ok := result["prompt_section"].(string); !ok || promptSection != contextStr {
		t.Errorf("colony-prime result.prompt_section = %v, want same content as context", result["prompt_section"])
	}
	if result["signal_count"] != float64(1) {
		t.Errorf("colony-prime result.signal_count = %v, want 1", result["signal_count"])
	}
	if result["instinct_count"] != float64(1) {
		t.Errorf("colony-prime result.instinct_count = %v, want 1", result["instinct_count"])
	}
	if logLine, ok := result["log_line"].(string); !ok || !strings.Contains(logLine, "1 signal(s), 1 instinct(s)") {
		t.Errorf("colony-prime result.log_line = %v, want counts summary", result["log_line"])
	}

	// trimmed should be an array (may be empty)
	_, ok = result["trimmed"].([]interface{})
	if !ok {
		t.Error("colony-prime result.trimmed must be an array")
	}

	ledger, ok := result["ledger"].(map[string]interface{})
	if !ok {
		t.Fatal("colony-prime result.ledger must be an object")
	}
	included, ok := ledger["included"].([]interface{})
	if !ok || len(included) == 0 {
		t.Fatal("colony-prime result.ledger.included must be a non-empty array")
	}
	firstIncluded := included[0].(map[string]interface{})
	if firstIncluded["name"] == "" {
		t.Error("ledger included entry missing name")
	}
	if firstIncluded["title"] == "" {
		t.Error("ledger included entry missing title")
	}
	if firstIncluded["source"] == "" {
		t.Error("ledger included entry missing source")
	}
	if _, ok := ledger["trimmed"].([]interface{}); !ok {
		t.Error("colony-prime result.ledger.trimmed must be an array")
	}
}
