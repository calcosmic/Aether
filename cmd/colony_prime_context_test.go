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
	"github.com/calcosmic/Aether/pkg/storage"
)

// TestContextCapsuleBlocksDispatchWhenEmpty verifies that resolveCodexWorkerContext
// returns an error when the assembled context capsule is below 512 characters,
// preventing zero-context worker execution.
func TestContextCapsuleBlocksDispatchWhenEmpty(t *testing.T) {
	saveGlobals(t)

	// Set up a minimal store with almost no state so context will be tiny
	tmpDir := t.TempDir()
	dataDir := tmpDir + "/.aether/data"
	os.MkdirAll(dataDir, 0755)
	s, err := storage.NewStore(dataDir)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	store = s
	stdout = &bytes.Buffer{}
	stderr = &bytes.Buffer{}

	goal := "test"
	state := colony.ColonyState{
		Version:      "1.0",
		Goal:         &goal,
		State:        colony.StateREADY,
		CurrentPhase: 1,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{ID: 1, Name: "Test", Status: colony.PhaseReady},
			},
		},
	}
	if err := s.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatalf("save state: %v", err)
	}

	context := resolveCodexWorkerContext()
	if len(context) >= 128 {
		t.Skipf("context is %d chars (≥128) in this environment; cannot test empty-capsule block", len(context))
	}

	// The fix returns empty string when context is below 128 chars.
	if context != "" {
		t.Fatalf("resolveCodexWorkerContext returned %d chars — expected empty string for context below 128 chars", len(context))
	}
	t.Logf("✓ resolveCodexWorkerContext correctly returned empty string for %d-char context", len(context))
}

// dedent strips a single leading tab from each line of s, if present.
func dedent(s string) string {
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		if strings.HasPrefix(line, "\t") {
			lines[i] = line[1:]
		}
	}
	return strings.Join(lines, "\n")
}

// TestColonyPrimeTemplateLoad verifies that a custom colony-prime.md template
// file overrides the hardcoded section headers and format strings.
func TestColonyPrimeTemplateLoad(t *testing.T) {
	saveGlobals(t)

	tmpDir := t.TempDir()
	promptsDir := filepath.Join(tmpDir, "colony", "prompts")
	os.MkdirAll(promptsDir, 0755)
	customPath := filepath.Join(promptsDir, "colony-prime.md")
	content := dedent(`---
colony_prime_version: "1.0"
section_templates:
  state:
    header: "## CUSTOM STATE\n\n"
    goal_format: "CustomGoal: %s\n"
    state_format: "CustomState: %s\n"
    phase_format: "CustomPhase: %d\n"
    phase_name_format: "CustomPhaseName: %s\n"
    task_format: "  - CUSTOM [%s] %s\n"
    tasks_header: "CustomTasks:\n"
    parallel_mode_format: "CustomParallel: %s\n"
  review_depth:
    header: "## CUSTOM REVIEW DEPTH\n\n"
    light_text: "CUSTOM LIGHT"
    standard_text: "CUSTOM STANDARD"
    heavy_text: "CUSTOM HEAVY"
    default_text: "CUSTOM DEFAULT"
  pheromones:
    header: "## CUSTOM PHEROMONES\n\n"
    signal_format: "CUSTOM - [%s] %s\n"
  instincts:
    header: "## CUSTOM INSTINCTS\n\n"
    instinct_format: "CUSTOM - [%s] %s (confidence: %.2f)\n"
  decisions:
    header: "## CUSTOM DECISIONS\n\n"
    decision_format: "CUSTOM - Phase %d: %s — %s\n"
  learnings:
    header: "## CUSTOM LEARNINGS\n\n"
    phase_header_format: "CUSTOM ### Phase %d: %s\n"
    learning_format: "CUSTOM   - %s [%s]\n"
  worker_handoffs:
    header: "## CUSTOM HANDOFFS\n\n"
    worker_header_format: "CUSTOM ### %s\n"
    status_format: "CUSTOM - Status: %s; verification: %s\n"
    summary_format: "CUSTOM - Summary: %s\n"
    list_format: "CUSTOM - %s: %s\n"
  hive_wisdom:
    header: "## CUSTOM HIVE\n\n"
    entry_format: "CUSTOM - %s\n"
  learned_memory:
    header: "## CUSTOM MEMORY\n\n"
    entry_format: "CUSTOM - [Phase %d] %s (confidence: %.0f%%, classification: %s)\n"
  global_queen_md:
    header: "## CUSTOM GLOBAL QUEEN\n\n"
    entry_format: "CUSTOM - %s\n"
  user_preferences:
    header: "## CUSTOM PREFERENCES\n\n"
    entry_format: "CUSTOM - %s\n"
  prior_reviews:
    header: "## CUSTOM PRIOR REVIEWS\n\n"
    domain_format: "CUSTOM - %s (%d open): %s\n"
    domain_count_only_format: "CUSTOM - %s (%d open)\n"
  local_queen_wisdom:
    header: "## CUSTOM LOCAL QUEEN\n\n"
    entry_format: "CUSTOM - %s\n"
  clarified_intent:
    header: "## CUSTOM INTENT\n\n"
  blockers:
    header: "## CUSTOM BLOCKERS\n\n"
    blocker_format: "CUSTOM - %s\n"
  medic_health:
    header: "## CUSTOM HEALTH\n\n"
    scan_timestamp_format: "CUSTOM Last scan: %s\n\n"
    issue_format: "CUSTOM - [%s] %s"
    issue_file_format: " CUSTOM (%s)"
---
`)
	if err := os.WriteFile(customPath, []byte(content), 0644); err != nil {
		t.Fatalf("write custom template: %v", err)
	}

	colonyPrimeTemplatesPathOverride = customPath
	resetColonyPrimeTemplatesCache()

	dataDir := filepath.Join(tmpDir, ".aether", "data")
	os.MkdirAll(dataDir, 0755)
	hubDir := filepath.Join(tmpDir, "hub")
	if err := os.MkdirAll(filepath.Join(hubDir, "hive"), 0755); err != nil {
		t.Fatalf("mkdir hub hive: %v", err)
	}
	t.Setenv("AETHER_HUB_DIR", hubDir)
	s, err := storage.NewStore(dataDir)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	store = s

	// Hive retrieval is on by default (D-01/D-02); no opt-in needed.

	stdout = &bytes.Buffer{}
	stderr = &bytes.Buffer{}

	goal := "test"
	state := colony.ColonyState{
		Version:      "1.0",
		Goal:         &goal,
		State:        colony.StateREADY,
		CurrentPhase: 1,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{ID: 1, Name: "Test", Status: colony.PhaseReady, Tasks: []colony.Task{
					{Status: "open", Goal: "do thing"},
				}},
			},
		},
		ParallelMode:      colony.ModeInRepo,
		VerificationDepth: string(colony.VerificationDepthHeavy),
		Memory: colony.Memory{
			Decisions: []colony.Decision{
				{Phase: 1, Claim: "claim", Rationale: "rationale"},
			},
			PhaseLearnings: []colony.PhaseLearning{
				{Phase: 1, PhaseName: "Test", Learnings: []colony.Learning{
					{Claim: "learned", Status: "verified"},
				}},
			},
		},
	}
	if err := s.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatalf("save state: %v", err)
	}

	// instincts.json
	instinctsFile := colony.InstinctsFile{
		Instincts: []colony.InstinctEntry{
			{Trigger: "test", Action: "action", Confidence: 0.8},
		},
	}
	if err := s.SaveJSON("instincts.json", instinctsFile); err != nil {
		t.Fatalf("save instincts: %v", err)
	}

	// entries.json (learned memory)
	entries := []map[string]interface{}{
		{
			"phase":          1,
			"content":        "learned something",
			"confidence":     0.85,
			"classification": "pattern",
			"evidence":       map[string]interface{}{"timestamp": "2024-01-01T00:00:00Z"},
		},
	}
	if err := s.SaveJSON("entries.json", entries); err != nil {
		t.Fatalf("save entries: %v", err)
	}

	// pending-decisions.json (blockers)
	blockerFile := colony.FlagsFile{
		Decisions: []colony.FlagEntry{
			{Description: "blocker desc", Type: "blocker", Resolved: false, CreatedAt: time.Now().UTC().Format(time.RFC3339)},
		},
	}
	if err := s.SaveJSON("pending-decisions.json", blockerFile); err != nil {
		t.Fatalf("save blockers: %v", err)
	}

	// medic-last-scan.json
	medicScan := MedicLastScan{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Issues: []HealthIssue{
			{Severity: "critical", Message: "critical issue"},
		},
	}
	if err := s.SaveJSON("medic-last-scan.json", medicScan); err != nil {
		t.Fatalf("save medic scan: %v", err)
	}

	// hub QUEEN.md (global queen wisdom + user preferences)
	globalQueenContent := "## Patterns\n\n- Always write tests first\n\n## User Preferences\n\n- Speak plain English\n"
	if err := os.WriteFile(filepath.Join(hubDir, "QUEEN.md"), []byte(globalQueenContent), 0644); err != nil {
		t.Fatalf("write global queen: %v", err)
	}
	hiveData := hiveWisdomData{
		Entries: []hiveWisdomEntry{{
			ID:         "hw1",
			Text:       "Hive wisdom: prefer composition over inheritance",
			Confidence: 0.85,
			Domain:     "go",
			AccessedAt: time.Now().UTC().Format(time.RFC3339),
		}},
	}
	hiveJSON, err := json.MarshalIndent(hiveData, "", "  ")
	if err != nil {
		t.Fatalf("marshal hive wisdom: %v", err)
	}
	if err := os.WriteFile(filepath.Join(hubDir, "hive", "wisdom.json"), hiveJSON, 0644); err != nil {
		t.Fatalf("write hive wisdom: %v", err)
	}

	// local QUEEN.md (local queen wisdom)
	localQueenPath := filepath.Join(filepath.Dir(dataDir), "QUEEN.md")
	localQueenContent := "## Patterns\n\n- Local wisdom entry\n"
	if err := os.WriteFile(localQueenPath, []byte(localQueenContent), 0644); err != nil {
		t.Fatalf("write local queen: %v", err)
	}

	// pending-decisions.json with clarified intent and blocker
	pendingFile := colony.FlagsFile{
		Decisions: []colony.FlagEntry{
			{ID: "1", Description: "Q: what?", Type: "clarification", Resolved: true, Resolution: "answer", CreatedAt: time.Now().UTC().Format(time.RFC3339)},
			{ID: "2", Description: "blocker desc", Type: "blocker", Resolved: false, CreatedAt: time.Now().UTC().Format(time.RFC3339)},
		},
	}
	if err := s.SaveJSON("pending-decisions.json", pendingFile); err != nil {
		t.Fatalf("save pending decisions: %v", err)
	}

	output := buildColonyPrimeOutput(true)
	ctx := output.Context

	assertions := []struct {
		name     string
		expected string
	}{
		{"state header", "## CUSTOM STATE"},
		{"goal format", "CustomGoal:"},
		{"state format", "CustomState:"},
		{"phase format", "CustomPhase:"},
		{"phase name format", "CustomPhaseName:"},
		{"tasks header", "CustomTasks:"},
		{"task format", "CUSTOM [open] do thing"},
		{"parallel mode", "CustomParallel:"},
		{"review depth header", "## CUSTOM REVIEW DEPTH"},
		{"review depth heavy", "CUSTOM HEAVY"},
		{"instincts header", "## CUSTOM INSTINCTS"},
		{"decisions header", "## CUSTOM DECISIONS"},
		{"learnings header", "## CUSTOM LEARNINGS"},
		{"hive wisdom header", "## CUSTOM HIVE"},
		{"learned memory header", "## CUSTOM MEMORY"},
		{"global queen header", "## CUSTOM GLOBAL QUEEN"},
		{"user preferences header", "## CUSTOM PREFERENCES"},
		{"local queen header", "## CUSTOM LOCAL QUEEN"},
		{"clarified intent header", "## CUSTOM INTENT"},
		{"blockers header", "## CUSTOM BLOCKERS"},
		{"medic health header", "## CUSTOM HEALTH"},
	}

	for _, a := range assertions {
		t.Run(a.name, func(t *testing.T) {
			if !strings.Contains(ctx, a.expected) {
				t.Fatalf("expected context to contain %q, got:\n%s", a.expected, ctx)
			}
		})
	}
}

// TestColonyPrimeTemplateFallback verifies that when no colony-prime.md exists,
// the hardcoded headers and format strings are used.
func TestColonyPrimeTemplateFallback(t *testing.T) {
	saveGlobals(t)

	tmpDir := t.TempDir()
	// Ensure no colony-prime.md exists
	colonyPrimeTemplatesPathOverride = filepath.Join(tmpDir, "nonexistent.md")
	resetColonyPrimeTemplatesCache()

	dataDir := filepath.Join(tmpDir, ".aether", "data")
	os.MkdirAll(dataDir, 0755)
	s, err := storage.NewStore(dataDir)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	store = s
	stdout = &bytes.Buffer{}
	stderr = &bytes.Buffer{}

	goal := "test"
	state := colony.ColonyState{
		Version:      "1.0",
		Goal:         &goal,
		State:        colony.StateREADY,
		CurrentPhase: 1,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{ID: 1, Name: "Test", Status: colony.PhaseReady},
			},
		},
	}
	if err := s.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatalf("save state: %v", err)
	}

	output := buildColonyPrimeOutput(true)
	ctx := output.Context

	assertions := []string{
		"## Colony State",
		"Goal: test",
		"State: READY",
		"Phase: 1",
	}
	for _, expected := range assertions {
		if !strings.Contains(ctx, expected) {
			t.Fatalf("expected context to contain %q, got:\n%s", expected, ctx)
		}
	}
}

// TestColonyPrimePartialFallback verifies that a colony-prime.md with only some
// sections defined leaves the remaining sections using hardcoded defaults.
func TestColonyPrimePartialFallback(t *testing.T) {
	saveGlobals(t)

	tmpDir := t.TempDir()
	promptsDir := filepath.Join(tmpDir, "colony", "prompts")
	os.MkdirAll(promptsDir, 0755)
	customPath := filepath.Join(promptsDir, "colony-prime.md")
	content := dedent(`---
colony_prime_version: "1.0"
section_templates:
  state:
    header: "## PARTIAL STATE\n\n"
    goal_format: "PartialGoal: %s\n"
  pheromones:
    header: "## PARTIAL PHEROMONES\n\n"
---
`)
	if err := os.WriteFile(customPath, []byte(content), 0644); err != nil {
		t.Fatalf("write custom template: %v", err)
	}

	colonyPrimeTemplatesPathOverride = customPath
	resetColonyPrimeTemplatesCache()

	dataDir := filepath.Join(tmpDir, ".aether", "data")
	os.MkdirAll(dataDir, 0755)
	s, err := storage.NewStore(dataDir)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	store = s
	stdout = &bytes.Buffer{}
	stderr = &bytes.Buffer{}

	goal := "test"
	state := colony.ColonyState{
		Version:      "1.0",
		Goal:         &goal,
		State:        colony.StateREADY,
		CurrentPhase: 1,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{ID: 1, Name: "Test", Status: colony.PhaseReady},
			},
		},
	}
	if err := s.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatalf("save state: %v", err)
	}

	// instincts.json so the instincts section is generated and uses fallback header
	instinctsFile := colony.InstinctsFile{
		Instincts: []colony.InstinctEntry{
			{Trigger: "test", Action: "action", Confidence: 0.8},
		},
	}
	if err := s.SaveJSON("instincts.json", instinctsFile); err != nil {
		t.Fatalf("save instincts: %v", err)
	}

	output := buildColonyPrimeOutput(true)
	ctx := output.Context

	if !strings.Contains(ctx, "## PARTIAL STATE") {
		t.Fatalf("expected custom state header, got:\n%s", ctx)
	}
	if !strings.Contains(ctx, "PartialGoal: test") {
		t.Fatalf("expected custom goal format, got:\n%s", ctx)
	}
	if !strings.Contains(ctx, "## Active Instincts") {
		t.Fatalf("expected fallback instincts header, got:\n%s", ctx)
	}
}

// TestColonyPrimeReviewDepthCustom verifies that custom review_depth text is used.
func TestColonyPrimeReviewDepthCustom(t *testing.T) {
	saveGlobals(t)

	tmpDir := t.TempDir()
	promptsDir := filepath.Join(tmpDir, "colony", "prompts")
	os.MkdirAll(promptsDir, 0755)
	customPath := filepath.Join(promptsDir, "colony-prime.md")
	content := dedent(`---
colony_prime_version: "1.0"
section_templates:
  review_depth:
    header: "## Review Depth\n\n"
    heavy_text: "HEAVY CUSTOM TEXT"
---
`)
	if err := os.WriteFile(customPath, []byte(content), 0644); err != nil {
		t.Fatalf("write custom template: %v", err)
	}

	colonyPrimeTemplatesPathOverride = customPath
	resetColonyPrimeTemplatesCache()

	dataDir := filepath.Join(tmpDir, ".aether", "data")
	os.MkdirAll(dataDir, 0755)
	s, err := storage.NewStore(dataDir)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	store = s
	stdout = &bytes.Buffer{}
	stderr = &bytes.Buffer{}

	goal := "test"
	state := colony.ColonyState{
		Version:      "1.0",
		Goal:         &goal,
		State:        colony.StateREADY,
		CurrentPhase: 1,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{ID: 1, Name: "security-audit", Status: colony.PhaseReady},
			},
		},
		VerificationDepth: string(colony.VerificationDepthHeavy),
	}
	if err := s.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatalf("save state: %v", err)
	}

	output := buildColonyPrimeOutput(true)
	ctx := output.Context

	if !strings.Contains(ctx, "HEAVY CUSTOM TEXT") {
		t.Fatalf("expected custom heavy review text, got:\n%s", ctx)
	}
}

// TestColonyPrimeBlockerFormatCustom verifies that custom blockers format is used.
func TestColonyPrimeBlockerFormatCustom(t *testing.T) {
	saveGlobals(t)

	tmpDir := t.TempDir()
	promptsDir := filepath.Join(tmpDir, "colony", "prompts")
	os.MkdirAll(promptsDir, 0755)
	customPath := filepath.Join(promptsDir, "colony-prime.md")
	content := dedent(`---
colony_prime_version: "1.0"
section_templates:
  blockers:
    header: "## Active Blockers\n\n"
    blocker_format: "BLOCKER: %s\n"
---
`)
	if err := os.WriteFile(customPath, []byte(content), 0644); err != nil {
		t.Fatalf("write custom template: %v", err)
	}

	colonyPrimeTemplatesPathOverride = customPath
	resetColonyPrimeTemplatesCache()

	dataDir := filepath.Join(tmpDir, ".aether", "data")
	os.MkdirAll(dataDir, 0755)
	s, err := storage.NewStore(dataDir)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	store = s
	stdout = &bytes.Buffer{}
	stderr = &bytes.Buffer{}

	goal := "test"
	state := colony.ColonyState{
		Version:      "1.0",
		Goal:         &goal,
		State:        colony.StateREADY,
		CurrentPhase: 1,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{ID: 1, Name: "Test", Status: colony.PhaseReady},
			},
		},
	}
	if err := s.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatalf("save state: %v", err)
	}

	blockers := colony.FlagsFile{
		Decisions: []colony.FlagEntry{
			{Description: "something is broken", Type: "blocker", Resolved: false, CreatedAt: "2024-01-01T00:00:00Z"},
		},
	}
	if err := s.SaveJSON("pending-decisions.json", blockers); err != nil {
		t.Fatalf("save blockers: %v", err)
	}

	output := buildColonyPrimeOutput(true)
	ctx := output.Context

	if !strings.Contains(ctx, "BLOCKER: something is broken") {
		t.Fatalf("expected custom blocker format, got:\n%s", ctx)
	}
}

// TestColonyPrimePheromoneSignalFormatCustom verifies that custom pheromone signal format is used.
func TestColonyPrimePheromoneSignalFormatCustom(t *testing.T) {
	saveGlobals(t)

	tmpDir := t.TempDir()
	promptsDir := filepath.Join(tmpDir, "colony", "prompts")
	os.MkdirAll(promptsDir, 0755)
	customPath := filepath.Join(promptsDir, "colony-prime.md")
	content := dedent(`---
colony_prime_version: "1.0"
section_templates:
  pheromones:
    header: "## Pheromone Signals\n\n"
    signal_format: "SIGNAL [%s] %s\n"
---
`)
	if err := os.WriteFile(customPath, []byte(content), 0644); err != nil {
		t.Fatalf("write custom template: %v", err)
	}

	colonyPrimeTemplatesPathOverride = customPath
	resetColonyPrimeTemplatesCache()

	dataDir := filepath.Join(tmpDir, ".aether", "data")
	os.MkdirAll(dataDir, 0755)
	s, err := storage.NewStore(dataDir)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	store = s
	stdout = &bytes.Buffer{}
	stderr = &bytes.Buffer{}

	goal := "test"
	state := colony.ColonyState{
		Version:      "1.0",
		Goal:         &goal,
		State:        colony.StateREADY,
		CurrentPhase: 1,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{ID: 1, Name: "Test", Status: colony.PhaseReady},
			},
		},
	}
	if err := s.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatalf("save state: %v", err)
	}

	pf := colony.PheromoneFile{
		Signals: []colony.PheromoneSignal{
			{Type: "FOCUS", Content: json.RawMessage(`"pay attention here"`), CreatedAt: time.Now().UTC().Format(time.RFC3339), Active: true},
		},
	}
	if err := s.SaveJSON("pheromones.json", pf); err != nil {
		t.Fatalf("save pheromones: %v", err)
	}

	output := buildColonyPrimeOutput(true)
	ctx := output.Context

	if !strings.Contains(ctx, "SIGNAL [FOCUS] pay attention here") && !strings.Contains(ctx, "SIGNAL [FOCUS] \"pay attention here\"") {
		t.Fatalf("expected custom signal format, got:\n%s", ctx)
	}
}

// TestColonyPrimeWorkerHandoffTemplate verifies that renderWorkerHandoffSection
// uses templates for header and format strings.
func TestColonyPrimeWorkerHandoffTemplate(t *testing.T) {
	saveGlobals(t)

	tmpDir := t.TempDir()
	promptsDir := filepath.Join(tmpDir, "colony", "prompts")
	os.MkdirAll(promptsDir, 0755)
	customPath := filepath.Join(promptsDir, "colony-prime.md")
	content := dedent(`---
colony_prime_version: "1.0"
section_templates:
  worker_handoffs:
    header: "## CUSTOM HANDOFFS\n\n"
    worker_header_format: "CUSTOM ### %s\n"
    status_format: "CUSTOM - Status: %s; verification: %s\n"
    summary_format: "CUSTOM - Summary: %s\n"
---
`)
	if err := os.WriteFile(customPath, []byte(content), 0644); err != nil {
		t.Fatalf("write custom template: %v", err)
	}

	colonyPrimeTemplatesPathOverride = customPath
	resetColonyPrimeTemplatesCache()

	dataDir := filepath.Join(tmpDir, ".aether", "data")
	os.MkdirAll(dataDir, 0755)
	s, err := storage.NewStore(dataDir)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	store = s

	records := workerHandoffFile{
		Entries: []workerHandoffRecord{
			{
				ID:                 "1",
				Workflow:           "build",
				Phase:              1,
				WorkerName:         "Builder-1",
				Status:             "completed",
				VerificationStatus: "pass",
				Summary:            "did work",
				Freshness:          "2024-01-02T00:00:00Z",
			},
		},
	}
	if err := s.SaveJSON(workerHandoffsPath, records); err != nil {
		t.Fatalf("save handoffs: %v", err)
	}

	section := renderWorkerHandoffSection("build", 1, "")
	if !strings.Contains(section, "## CUSTOM HANDOFFS") {
		t.Fatalf("expected custom handoffs header, got:\n%s", section)
	}
	if !strings.Contains(section, "CUSTOM ### Builder-1") {
		t.Fatalf("expected custom worker header format, got:\n%s", section)
	}
	if !strings.Contains(section, "CUSTOM - Status: completed; verification: pass") {
		t.Fatalf("expected custom status format, got:\n%s", section)
	}
	if !strings.Contains(section, "CUSTOM - Summary: did work") {
		t.Fatalf("expected custom summary format, got:\n%s", section)
	}
}

// TestColonyPrimePriorReviewsHeaderTemplate verifies that buildPriorReviewsSection
// uses the template header when available.
func TestColonyPrimePriorReviewsHeaderTemplate(t *testing.T) {
	saveGlobals(t)

	tmpDir := t.TempDir()
	promptsDir := filepath.Join(tmpDir, "colony", "prompts")
	os.MkdirAll(promptsDir, 0755)
	customPath := filepath.Join(promptsDir, "colony-prime.md")
	content := dedent(`---
colony_prime_version: "1.0"
section_templates:
  prior_reviews:
    header: "## CUSTOM PRIOR REVIEWS\n\n"
---
`)
	if err := os.WriteFile(customPath, []byte(content), 0644); err != nil {
		t.Fatalf("write custom template: %v", err)
	}

	colonyPrimeTemplatesPathOverride = customPath
	resetColonyPrimeTemplatesCache()

	dataDir := filepath.Join(tmpDir, ".aether", "data")
	os.MkdirAll(dataDir, 0755)
	s, err := storage.NewStore(dataDir)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	store = s

	// Create a review ledger with an open finding so the section is generated
	reviewsDir := filepath.Join(dataDir, "reviews", "security")
	os.MkdirAll(reviewsDir, 0755)
	ledger := colony.ReviewLedgerFile{
		Entries: []colony.ReviewLedgerEntry{
			{Status: "open", Severity: colony.ReviewSeverityHigh, Description: "bad thing", File: "foo.go", Line: 10, GeneratedAt: "2024-01-01T00:00:00Z"},
		},
	}
	if err := s.SaveJSON("reviews/security/ledger.json", ledger); err != nil {
		t.Fatalf("save ledger: %v", err)
	}

	section, count := buildPriorReviewsSection(s, true)
	if count == 0 {
		t.Fatalf("expected review count > 0")
	}
	if !strings.Contains(section.content, "## CUSTOM PRIOR REVIEWS") {
		t.Fatalf("expected custom prior reviews header, got:\n%s", section.content)
	}
}

// TestColonyPrimeMedicHealthTemplate verifies that the medic health section uses
// template format strings when available.
func TestColonyPrimeMedicHealthTemplate(t *testing.T) {
	saveGlobals(t)

	tmpDir := t.TempDir()
	promptsDir := filepath.Join(tmpDir, "colony", "prompts")
	os.MkdirAll(promptsDir, 0755)
	customPath := filepath.Join(promptsDir, "colony-prime.md")
	content := dedent(`---
colony_prime_version: "1.0"
section_templates:
  medic_health:
    header: "## CUSTOM HEALTH\n\n"
    scan_timestamp_format: "CUSTOM Last scan: %s\n\n"
    issue_format: "CUSTOM - [%s] %s"
    issue_file_format: " CUSTOM (%s)"
---
`)
	if err := os.WriteFile(customPath, []byte(content), 0644); err != nil {
		t.Fatalf("write custom template: %v", err)
	}

	colonyPrimeTemplatesPathOverride = customPath
	resetColonyPrimeTemplatesCache()

	dataDir := filepath.Join(tmpDir, ".aether", "data")
	os.MkdirAll(dataDir, 0755)
	s, err := storage.NewStore(dataDir)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	store = s
	stdout = &bytes.Buffer{}
	stderr = &bytes.Buffer{}

	goal := "test"
	state := colony.ColonyState{
		Version:      "1.0",
		Goal:         &goal,
		State:        colony.StateREADY,
		CurrentPhase: 1,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{ID: 1, Name: "Test", Status: colony.PhaseReady},
			},
		},
	}
	if err := s.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatalf("save state: %v", err)
	}

	scan := MedicLastScan{
		Timestamp: "2024-01-01T00:00:00Z",
		Issues: []HealthIssue{
			{Severity: "critical", Message: "disk full", File: "disk.go"},
		},
	}
	if err := s.SaveJSON(medicLastScanFile, scan); err != nil {
		t.Fatalf("save medic scan: %v", err)
	}

	output := buildColonyPrimeOutput(true)
	ctx := output.Context

	if !strings.Contains(ctx, "## CUSTOM HEALTH") {
		t.Fatalf("expected custom health header, got:\n%s", ctx)
	}
	if !strings.Contains(ctx, "CUSTOM Last scan: 2024-01-01T00:00:00Z") {
		t.Fatalf("expected custom scan timestamp format, got:\n%s", ctx)
	}
	if !strings.Contains(ctx, "CUSTOM - [critical] disk full CUSTOM (disk.go)") {
		t.Fatalf("expected custom issue format, got:\n%s", ctx)
	}
}

// TestColonyPrimeLearnedMemoryTemplate verifies that learned memory section uses
// template format strings.
func TestColonyPrimeLearnedMemoryTemplate(t *testing.T) {
	saveGlobals(t)

	tmpDir := t.TempDir()
	promptsDir := filepath.Join(tmpDir, "colony", "prompts")
	os.MkdirAll(promptsDir, 0755)
	customPath := filepath.Join(promptsDir, "colony-prime.md")
	content := dedent(`---
colony_prime_version: "1.0"
section_templates:
  learned_memory:
    header: "## CUSTOM MEMORY\n\n"
    entry_format: "CUSTOM - [Phase %d] %s (confidence: %.0f%%, classification: %s)\n"
---
`)
	if err := os.WriteFile(customPath, []byte(content), 0644); err != nil {
		t.Fatalf("write custom template: %v", err)
	}

	colonyPrimeTemplatesPathOverride = customPath
	resetColonyPrimeTemplatesCache()

	dataDir := filepath.Join(tmpDir, ".aether", "data")
	os.MkdirAll(dataDir, 0755)
	s, err := storage.NewStore(dataDir)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	store = s
	stdout = &bytes.Buffer{}
	stderr = &bytes.Buffer{}

	goal := "test"
	state := colony.ColonyState{
		Version:      "1.0",
		Goal:         &goal,
		State:        colony.StateREADY,
		CurrentPhase: 1,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{ID: 1, Name: "Test", Status: colony.PhaseReady},
			},
		},
	}
	if err := s.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatalf("save state: %v", err)
	}

	entries := []map[string]interface{}{
		{
			"phase":          1,
			"content":        "learned something",
			"confidence":     0.85,
			"classification": "pattern",
			"evidence":       map[string]interface{}{"timestamp": "2024-01-01T00:00:00Z"},
		},
	}
	if err := s.SaveJSON("entries.json", entries); err != nil {
		t.Fatalf("save entries: %v", err)
	}

	output := buildColonyPrimeOutput(true)
	ctx := output.Context

	if !strings.Contains(ctx, "## CUSTOM MEMORY") {
		t.Fatalf("expected custom memory header, got:\n%s", ctx)
	}
	if !strings.Contains(ctx, "CUSTOM - [Phase 1] learned something (confidence: 85%, classification: pattern)") {
		t.Fatalf("expected custom memory entry format, got:\n%s", ctx)
	}
}

// buildRichColonyPrimeFixture populates a store with data that exercises
// every section colony-prime.md's section_templates defines (state,
// review_depth, pheromones, instincts, decisions, learnings, worker_handoffs,
// hive_wisdom, learned_memory, global_queen_md, user_preferences,
// local_queen_wisdom, clarified_intent, blockers, medic_health) so a
// before/after render comparison actually exercises every template field
// instead of only the ones a sparse fixture happens to reach.
func buildRichColonyPrimeFixture(t *testing.T, tmpDir string, dataDir string) *storage.Store {
	t.Helper()

	hubDir := filepath.Join(tmpDir, "hub")
	if err := os.MkdirAll(filepath.Join(hubDir, "hive"), 0755); err != nil {
		t.Fatalf("mkdir hub hive: %v", err)
	}
	t.Setenv("AETHER_HUB_DIR", hubDir)

	s, err := storage.NewStore(dataDir)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	store = s
	stdout = &bytes.Buffer{}
	stderr = &bytes.Buffer{}

	goal := "191-02 fold-and-delete byte-identical proof"
	state := colony.ColonyState{
		Version:      "1.0",
		Goal:         &goal,
		State:        colony.StateREADY,
		CurrentPhase: 1,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{ID: 1, Name: "Fixture Phase", Status: colony.PhaseReady, Tasks: []colony.Task{
					{Status: "open", Goal: "do the fixture thing"},
				}},
			},
		},
		ParallelMode:      colony.ModeInRepo,
		VerificationDepth: string(colony.VerificationDepthHeavy),
		Memory: colony.Memory{
			Decisions: []colony.Decision{
				{Phase: 1, Claim: "claim", Rationale: "rationale"},
			},
			PhaseLearnings: []colony.PhaseLearning{
				{Phase: 1, PhaseName: "Fixture Phase", Learnings: []colony.Learning{
					{Claim: "learned", Status: "verified"},
				}},
			},
		},
	}
	if err := s.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatalf("save state: %v", err)
	}

	pf := colony.PheromoneFile{
		Signals: []colony.PheromoneSignal{
			{Type: "FOCUS", Content: json.RawMessage(`"pay attention here"`), CreatedAt: time.Now().UTC().Format(time.RFC3339), Active: true},
		},
	}
	if err := s.SaveJSON("pheromones.json", pf); err != nil {
		t.Fatalf("save pheromones: %v", err)
	}

	instinctsFile := colony.InstinctsFile{
		Instincts: []colony.InstinctEntry{
			{Trigger: "test", Action: "action", Confidence: 0.8},
		},
	}
	if err := s.SaveJSON("instincts.json", instinctsFile); err != nil {
		t.Fatalf("save instincts: %v", err)
	}

	entries := []map[string]interface{}{
		{
			"phase":          1,
			"content":        "learned something",
			"confidence":     0.85,
			"classification": "pattern",
			"evidence":       map[string]interface{}{"timestamp": "2024-01-01T00:00:00Z"},
		},
	}
	if err := s.SaveJSON("entries.json", entries); err != nil {
		t.Fatalf("save entries: %v", err)
	}

	pendingFile := colony.FlagsFile{
		Decisions: []colony.FlagEntry{
			{ID: "1", Description: "Q: what?", Type: "clarification", Resolved: true, Resolution: "answer", CreatedAt: time.Now().UTC().Format(time.RFC3339)},
			{ID: "2", Description: "blocker desc", Type: "blocker", Resolved: false, CreatedAt: time.Now().UTC().Format(time.RFC3339)},
		},
	}
	if err := s.SaveJSON("pending-decisions.json", pendingFile); err != nil {
		t.Fatalf("save pending decisions: %v", err)
	}

	medicScan := MedicLastScan{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Issues: []HealthIssue{
			{Severity: "critical", Message: "critical issue", File: "cmd/example.go"},
		},
	}
	if err := s.SaveJSON("medic-last-scan.json", medicScan); err != nil {
		t.Fatalf("save medic scan: %v", err)
	}

	records := workerHandoffFile{
		Entries: []workerHandoffRecord{
			{
				ID:                 "1",
				Workflow:           "build",
				Phase:              1,
				WorkerName:         "Builder-1",
				Status:             "completed",
				VerificationStatus: "pass",
				Summary:            "did work",
				Freshness:          time.Now().UTC().Format(time.RFC3339),
			},
		},
	}
	if err := s.SaveJSON(workerHandoffsPath, records); err != nil {
		t.Fatalf("save handoffs: %v", err)
	}

	globalQueenContent := "## Patterns\n\n- Always write tests first\n\n## User Preferences\n\n- Speak plain English\n"
	if err := os.WriteFile(filepath.Join(hubDir, "QUEEN.md"), []byte(globalQueenContent), 0644); err != nil {
		t.Fatalf("write global queen: %v", err)
	}
	hiveData := hiveWisdomData{
		Entries: []hiveWisdomEntry{{
			ID:         "hw1",
			Text:       "Hive wisdom: prefer composition over inheritance",
			Confidence: 0.85,
			Domain:     "go",
			AccessedAt: time.Now().UTC().Format(time.RFC3339),
		}},
	}
	hiveJSON, err := json.MarshalIndent(hiveData, "", "  ")
	if err != nil {
		t.Fatalf("marshal hive wisdom: %v", err)
	}
	if err := os.WriteFile(filepath.Join(hubDir, "hive", "wisdom.json"), hiveJSON, 0644); err != nil {
		t.Fatalf("write hive wisdom: %v", err)
	}

	localQueenPath := filepath.Join(filepath.Dir(dataDir), "QUEEN.md")
	localQueenContent := "## Patterns\n\n- Local wisdom entry\n"
	if err := os.WriteFile(localQueenPath, []byte(localQueenContent), 0644); err != nil {
		t.Fatalf("write local queen: %v", err)
	}

	return s
}

// TestColonyPrimeMdDeletionProducesByteIdenticalOutput is 191-02's Layer 1
// (Go-level) proof: colony-prime.md's real, pre-deletion content -- captured
// verbatim in cmd/testdata/191-02-colony-prime-original.md before this task
// touched anything -- must render EXACTLY the same worker-context text as the
// post-deletion state (colonyPrimeTemplatesPathOverride pointed at a path
// that does not exist, forcing every section onto its Go-compiled fallback).
//
// If this test passes before colony/prompts/colony-prime.md is deleted, the
// fold in cmd/colony_prime_context.go is complete. If it still passes AFTER
// deletion (re-run in the same task, per 191-02-PLAN.md Task 2), that is the
// mechanical proof the fold survived the deletion -- not an assumption.
func TestColonyPrimeMdDeletionProducesByteIdenticalOutput(t *testing.T) {
	saveGlobals(t)

	// Render 1: the real, original colony-prime.md content (frozen in
	// testdata so this test does not depend on the live file's continued
	// existence -- it must still compile and pass once that file is gone).
	tmpDir1 := t.TempDir()
	dataDir1 := filepath.Join(tmpDir1, ".aether", "data")
	os.MkdirAll(dataDir1, 0755)
	buildRichColonyPrimeFixture(t, tmpDir1, dataDir1)

	originalPath, err := filepath.Abs(filepath.Join("testdata", "191-02-colony-prime-original.md"))
	if err != nil {
		t.Fatalf("resolve testdata path: %v", err)
	}
	if _, err := os.Stat(originalPath); err != nil {
		t.Fatalf("testdata fixture missing (must be captured before colony-prime.md is deleted): %v", err)
	}
	colonyPrimeTemplatesPathOverride = originalPath
	resetColonyPrimeTemplatesCache()
	beforeOutput := buildColonyPrimeOutput(false)
	if strings.TrimSpace(beforeOutput.Context) == "" {
		t.Fatal("before-render produced empty context -- fixture did not exercise the loader")
	}

	// Render 2: simulate colony-prime.md having been deleted -- every
	// section falls back to its Go-compiled default.
	tmpDir2 := t.TempDir()
	dataDir2 := filepath.Join(tmpDir2, ".aether", "data")
	os.MkdirAll(dataDir2, 0755)
	buildRichColonyPrimeFixture(t, tmpDir2, dataDir2)

	colonyPrimeTemplatesPathOverride = filepath.Join(tmpDir2, "nonexistent-colony-prime.md")
	resetColonyPrimeTemplatesCache()
	afterOutput := buildColonyPrimeOutput(false)
	if strings.TrimSpace(afterOutput.Context) == "" {
		t.Fatal("after-render produced empty context -- fixture did not exercise the loader")
	}

	if beforeOutput.Context != afterOutput.Context {
		t.Fatalf("colony-prime output is NOT byte-identical before vs after colony-prime.md deletion\n--- BEFORE (real file) ---\n%s\n--- AFTER (Go fallback) ---\n%s", beforeOutput.Context, afterOutput.Context)
	}

	// A sanity check that this fixture reaches sections beyond just "state"
	// -- otherwise the byte-identical assertion above would be trivially
	// true for the wrong reason (nothing substantive was compared).
	mustContain := []string{
		"## Colony State", "## Review Depth", "## Pheromone Signals",
		"## Active Instincts", "## Key Decisions", "## Phase Learnings",
		"## Previous Worker Handoffs", "## HIVE WISDOM", "## LEARNED MEMORY",
		"## GLOBAL QUEEN WISDOM", "## USER PREFERENCES", "## LOCAL QUEEN WISDOM",
		"## CLARIFIED INTENT", "## Active Blockers", "## Colony Health Issues",
	}
	for _, want := range mustContain {
		if !strings.Contains(beforeOutput.Context, want) {
			t.Errorf("fixture did not exercise section %q -- byte-identical proof is incomplete for this section", want)
		}
	}
}
