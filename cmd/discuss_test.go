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
)

func TestDiscussDoesNotCreateGenericClarificationQuestions(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldDir)

	goal := "Build a dashboard for internal operations"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateREADY,
		CurrentPhase: 0,
		ColonyDepth:  "light",
		Plan:         colony.Plan{},
	})

	rootCmd.SetArgs([]string{"discuss"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("discuss returned error: %v", err)
	}

	env := parseEnvelope(t, stdout.(*bytes.Buffer).String())
	result := env["result"].(map[string]interface{})
	if got := int(result["question_count"].(float64)); got != 0 {
		t.Fatalf("question_count = %d, want 0 without an evidence-backed material boundary", got)
	}
	if got := int(result["created_count"].(float64)); got != 0 {
		t.Fatalf("created_count = %d, want 0 because discuss no longer invents generic menus", got)
	}

	file := loadPendingDecisionFile()
	if len(file.Decisions) != 0 {
		t.Fatalf("generic discuss run persisted clarification decisions: %#v", file.Decisions)
	}
}

func TestDiscussUsesCodebaseContextAsEvidenceWithoutInventingQuestion(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldDir)

	if err := os.MkdirAll(filepath.Join(root, "cmd"), 0755); err != nil {
		t.Fatalf("mkdir cmd: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/discuss\n"), 0644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "Dockerfile"), []byte("FROM scratch\n"), 0644); err != nil {
		t.Fatalf("write Dockerfile: %v", err)
	}

	goal := "Refactor the architecture and improve test coverage"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateREADY,
		CurrentPhase: 0,
		ColonyDepth:  "light",
		Plan:         colony.Plan{},
	})

	rootCmd.SetArgs([]string{"discuss"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("discuss returned error: %v", err)
	}

	env := parseEnvelope(t, stdout.(*bytes.Buffer).String())
	result := env["result"].(map[string]interface{})
	questions := result["questions"].([]interface{})
	if len(questions) != 0 {
		t.Fatalf("codebase inventory became generic questions: %#v", questions)
	}
	if got := int(result["evidence_count"].(float64)); got == 0 {
		t.Fatal("expected repository context to be represented in the evidence frontier")
	}
}

func TestDiscussVisualPendingQuestionsAvoidsWorkerTheatre(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldDir)
	t.Setenv("AETHER_OUTPUT_MODE", "visual")

	goal := "Build a dashboard for internal operations"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateREADY,
		CurrentPhase: 0,
		ColonyDepth:  "light",
		Plan:         colony.Plan{},
	})
	if err := store.SaveJSON(pendingDecisionsFile, PendingDecisionFile{Decisions: []PendingDecision{{
		ID:             "pd_visual_scope",
		Type:           clarificationDecisionType,
		Description:    formatClarificationDescription("Which public surface owns this work?", []string{"current command", "new command"}),
		Source:         "wrapper:scope:visual-scope",
		Grounding:      "Two public surfaces would create materially different scope.",
		HardConstraint: true,
		CreatedAt:      time.Now().UTC().Format(time.RFC3339),
	}}}); err != nil {
		t.Fatalf("seed material visual question: %v", err)
	}

	rootCmd.SetArgs([]string{"discuss"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("discuss returned error: %v", err)
	}

	output := stdout.(*bytes.Buffer).String()
	for _, want := range []string{
		"D I S C U S S",
		"Questions: 1",
		"Owner decisions required",
		"Queen recommends:",
		"Answer exactly: $ant-discuss --resolve pd_visual_scope",
		"This answer becomes a hard constraint.",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("discuss visual missing %q\n%s", want, output)
		}
	}
	for _, forbidden := range []string{
		"workers completed",
		"Worker Results",
		"S P A W N",
		"worker-complete",
	} {
		if strings.Contains(output, forbidden) {
			t.Fatalf("discuss visual rendered worker theatre via %q\n%s", forbidden, output)
		}
	}
}

func TestDiscussVisualSettledPathAvoidsWorkerTheatre(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldDir)
	t.Setenv("AETHER_OUTPUT_MODE", "visual")

	goal := "Build a dashboard for internal operations"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version: "3.0",
		Goal:    &goal,
		State:   colony.StateREADY,
	})

	strength := 1.0
	now := time.Now().UTC().Format(time.RFC3339)
	makeContent := func(text string) json.RawMessage {
		payload, _ := json.Marshal(map[string]string{"text": text})
		return payload
	}
	if err := store.SaveJSON("pheromones.json", colony.PheromoneFile{
		Signals: []colony.PheromoneSignal{
			{ID: "sig_surface", Type: "REDIRECT", Priority: "high", Source: "test", CreatedAt: now, Active: true, Strength: &strength, Content: makeContent("use the current surface stack")},
			{ID: "sig_integration", Type: "FOCUS", Priority: "normal", Source: "test", CreatedAt: now, Active: true, Strength: &strength, Content: makeContent("preserve the current api contract")},
			{ID: "sig_scope", Type: "FEEDBACK", Priority: "low", Source: "test", CreatedAt: now, Active: true, Strength: &strength, Content: makeContent("prefer the smallest scope slice first")},
			{ID: "sig_verification", Type: "FEEDBACK", Priority: "low", Source: "test", CreatedAt: now, Active: true, Strength: &strength, Content: makeContent("keep the test verification bar explicit")},
		},
	}); err != nil {
		t.Fatalf("seed pheromones: %v", err)
	}

	rootCmd.SetArgs([]string{"discuss"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("discuss returned error: %v", err)
	}

	output := stdout.(*bytes.Buffer).String()
	for _, want := range []string{
		"D I S C U S S",
		"Questions: 0",
		"No unresolved material owner questions remain; evidence answered the rest.",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("settled discuss visual missing %q\n%s", want, output)
		}
	}
	for _, forbidden := range []string{
		"workers completed",
		"Worker Results",
		"S P A W N",
		"worker-complete",
	} {
		if strings.Contains(output, forbidden) {
			t.Fatalf("settled discuss visual rendered worker theatre via %q\n%s", forbidden, output)
		}
	}
}

func TestDiscussResolveHardConstraintEmitsRedirect(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldDir)

	if err := store.SaveJSON(pendingDecisionsFile, PendingDecisionFile{
		Decisions: []PendingDecision{{
			ID:          "pd_surface",
			Type:        clarificationDecisionType,
			Description: formatClarificationDescription("Which existing surface should own the first implementation slice?", []string{"admin-app", "new-module", "research-first"}),
			Source:      discussSource("surface", true),
			Resolved:    false,
			CreatedAt:   "2026-04-19T10:00:00Z",
		}},
	}); err != nil {
		t.Fatalf("save pending decisions: %v", err)
	}

	rootCmd.SetArgs([]string{"discuss", "--resolve", "pd_surface", "--answer", "Use the existing admin-app surface"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("discuss resolve returned error: %v", err)
	}

	env := parseEnvelope(t, stdout.(*bytes.Buffer).String())
	result := env["result"].(map[string]interface{})
	if emitted, _ := result["redirect_emitted"].(bool); !emitted {
		t.Fatal("expected resolving a hard clarification to emit a redirect pheromone")
	}

	var file PendingDecisionFile
	if err := store.LoadJSON(pendingDecisionsFile, &file); err != nil {
		t.Fatalf("reload pending decisions: %v", err)
	}
	if !file.Decisions[0].Resolved {
		t.Fatal("clarification should be resolved after discuss --resolve")
	}
	if file.Decisions[0].Resolution != "Use the existing admin-app surface" {
		t.Fatalf("resolution = %q", file.Decisions[0].Resolution)
	}

	var pheromones colony.PheromoneFile
	if err := store.LoadJSON("pheromones.json", &pheromones); err != nil {
		t.Fatalf("load pheromones: %v", err)
	}
	if len(pheromones.Signals) != 1 {
		t.Fatalf("expected 1 redirect pheromone, got %d", len(pheromones.Signals))
	}
	if pheromones.Signals[0].Type != "REDIRECT" {
		t.Fatalf("pheromone type = %q, want REDIRECT", pheromones.Signals[0].Type)
	}
	if text := extractText(pheromones.Signals[0].Content); text == "" || text == "Use the existing admin-app surface" {
		t.Fatalf("redirect text should include question context, got %q", text)
	}
}

func TestDiscussResolveBoundaryRoutesToDraftSpecification(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldDir)

	goal := "Resolve build boundary"
	sessionID := "session_resolve_build_boundary"
	initializedAt := time.Date(2026, 5, 12, 9, 0, 0, 0, time.UTC)
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:       "3.0",
		Goal:          &goal,
		State:         colony.StateREADY,
		SessionID:     &sessionID,
		InitializedAt: &initializedAt,
		ColonyMode:    colony.ColonyModeOrchestrator,
	})

	source := orchestratorBoundaryClarificationSource("build", 2, "build-scope", true)
	if err := store.SaveJSON(pendingDecisionsFile, PendingDecisionFile{
		Decisions: []PendingDecision{{
			ID:          "pd_build_boundary",
			Type:        clarificationDecisionType,
			Description: formatClarificationDescription("What boundary should builders protect for Phase 2?", []string{"phase tasks only", "pause"}),
			Source:      source,
			Resolved:    false,
			CreatedAt:   "2026-05-12T09:01:00Z",
			GoalHash:    pendingDecisionGoalHash(goal),
			SessionID:   sessionID,
		}},
	}); err != nil {
		t.Fatalf("seed boundary pending decision: %v", err)
	}

	rootCmd.SetArgs([]string{"discuss", "--resolve", "pd_build_boundary", "--answer", "Keep builders to phase tasks only."})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("discuss resolve returned error: %v", err)
	}

	env := parseEnvelope(t, stdout.(*bytes.Buffer).String())
	result := env["result"].(map[string]interface{})
	next := stringValue(result["next"])
	if !strings.Contains(next, "aether spec") || strings.Contains(next, "aether build") || strings.Contains(next, "aether plan") {
		t.Fatalf("next = %q, want draft specification review before workflow redispatch", next)
	}
	if got := stringValue(result["specification_status"]); got != string(colony.SpecStatusDraft) {
		t.Fatalf("specification_status = %q, want draft", got)
	}
}

func TestDiscussRejectsEmptyGoal(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldDir)

	goal := "   "
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version: "3.0",
		Goal:    &goal,
		State:   colony.StateREADY,
	})
	stderr = &bytes.Buffer{}

	rootCmd.SetArgs([]string{"discuss"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("discuss returned error: %v", err)
	}

	env := parseEnvelope(t, stderr.(*bytes.Buffer).String())
	if env["ok"] != false {
		t.Fatalf("expected ok:false, got %v", env)
	}
	if !strings.Contains(stringValue(env["error"]), "the colony goal is empty") {
		t.Fatalf("expected empty-goal error, got %v", env["error"])
	}
}

func TestDiscussIsIdempotentAcrossRuns(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldDir)

	goal := "Build a dashboard for internal operations"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version: "3.0",
		Goal:    &goal,
		State:   colony.StateREADY,
	})

	rootCmd.SetArgs([]string{"discuss"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("first discuss returned error: %v", err)
	}

	stdout.(*bytes.Buffer).Reset()

	rootCmd.SetArgs([]string{"discuss"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("second discuss returned error: %v", err)
	}

	env := parseEnvelope(t, stdout.(*bytes.Buffer).String())
	result := env["result"].(map[string]interface{})
	if got := int(result["created_count"].(float64)); got != 0 {
		t.Fatalf("created_count = %d, want 0 on repeat run", got)
	}
	if got := int(result["existing_count"].(float64)); got != 0 {
		t.Fatalf("existing_count = %d, want 0 on repeat evidence-only run", got)
	}
	if status := stringValue(result["discussion_status"]); status != "settled" {
		t.Fatalf("discussion_status = %q, want settled", status)
	}

	file := loadPendingDecisionFile()
	if len(file.Decisions) != 0 {
		t.Fatalf("repeat run persisted generic decisions: %#v", file.Decisions)
	}
}

func TestDiscussIgnoresStalePendingClarificationsFromPriorGoal(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldDir)

	currentGoal := "Build a dashboard for current operations"
	currentSession := "current_session"
	initializedAt := time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC)
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:       "3.0",
		Goal:          &currentGoal,
		State:         colony.StateREADY,
		SessionID:     &currentSession,
		InitializedAt: &initializedAt,
	})

	if err := store.SaveJSON(pendingDecisionsFile, PendingDecisionFile{
		Decisions: []PendingDecision{{
			ID:          "pd_old_surface",
			Type:        clarificationDecisionType,
			Description: formatClarificationDescription("Which existing surface should own the first implementation slice?", []string{"legacy-admin", "new-module", "research-first"}),
			Source:      discussSource("surface", true),
			Resolved:    false,
			CreatedAt:   "2026-04-19T10:00:00Z",
		}},
	}); err != nil {
		t.Fatalf("seed stale pending decisions: %v", err)
	}

	rootCmd.SetArgs([]string{"discuss"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("discuss returned error: %v", err)
	}

	env := parseEnvelope(t, stdout.(*bytes.Buffer).String())
	result := env["result"].(map[string]interface{})
	if got := int(result["existing_count"].(float64)); got != 0 {
		t.Fatalf("existing_count = %d, want 0 when only prior-goal decisions exist", got)
	}
	if got := int(result["created_count"].(float64)); got != 0 {
		t.Fatalf("created_count = %d, want no invented replacement questions", got)
	}
	if got := int(result["ignored_stale_count"].(float64)); got != 1 {
		t.Fatalf("ignored_stale_count = %d, want 1", got)
	}
	if notice := stringValue(result["stale_state_notice"]); !strings.Contains(notice, "Ignored 1 stale clarification") {
		t.Fatalf("stale_state_notice = %q, want ignored-stale explanation", notice)
	}

	var file PendingDecisionFile
	if err := store.LoadJSON(pendingDecisionsFile, &file); err != nil {
		t.Fatalf("load pending decisions: %v", err)
	}
	if len(file.Decisions) != 1 {
		t.Fatalf("expected only the quarantined old decision, got %d", len(file.Decisions))
	}
	currentScoped := 0
	for _, decision := range file.Decisions {
		if decision.SessionID == currentSession && decision.GoalHash == pendingDecisionGoalHash(currentGoal) {
			currentScoped++
		}
	}
	if currentScoped != 0 {
		t.Fatalf("current scoped decisions = %d, want 0", currentScoped)
	}
}

func TestPendingDecisionScopeRejectsMalformedLegacyClarificationTimestamps(t *testing.T) {
	initializedAt := time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC)
	scope := pendingDecisionScope{InitializedAt: &initializedAt}

	cases := []struct {
		name      string
		createdAt string
		want      bool
	}{
		{name: "before initialization", createdAt: "2026-05-10T11:59:59Z", want: false},
		{name: "at initialization", createdAt: "2026-05-10T12:00:00Z", want: true},
		{name: "after initialization", createdAt: "2026-05-10T12:00:01Z", want: true},
		{name: "missing timestamp", createdAt: "", want: false},
		{name: "invalid timestamp", createdAt: "not-a-time", want: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := pendingDecisionMatchesScope(PendingDecision{
				ID:        "pd_legacy",
				Type:      clarificationDecisionType,
				CreatedAt: tc.createdAt,
			}, scope)
			if got != tc.want {
				t.Fatalf("pendingDecisionMatchesScope created_at %q = %v, want %v", tc.createdAt, got, tc.want)
			}
		})
	}
}

func TestDiscussDryRunDoesNotPersistQuestions(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldDir)

	goal := "Build a dashboard for internal operations"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version: "3.0",
		Goal:    &goal,
		State:   colony.StateREADY,
	})

	rootCmd.SetArgs([]string{"discuss", "--dry-run"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("dry-run discuss returned error: %v", err)
	}

	env := parseEnvelope(t, stdout.(*bytes.Buffer).String())
	result := env["result"].(map[string]interface{})
	if dryRun, _ := result["dry_run"].(bool); !dryRun {
		t.Fatal("expected dry_run:true in discuss output")
	}
	if got := int(result["question_count"].(float64)); got != 0 {
		t.Fatalf("dry-run previewed %d generic clarification questions", got)
	}

	var file PendingDecisionFile
	if err := store.LoadJSON(pendingDecisionsFile, &file); err == nil && len(file.Decisions) > 0 {
		t.Fatalf("dry-run should not persist pending decisions, got %d", len(file.Decisions))
	}
}

func TestDiscussSettledWhenSignalsSuppressAllQuestions(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldDir)

	goal := "Build a dashboard for internal operations"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version: "3.0",
		Goal:    &goal,
		State:   colony.StateREADY,
	})

	strength := 1.0
	now := time.Now().UTC().Format(time.RFC3339)
	makeContent := func(text string) json.RawMessage {
		payload, _ := json.Marshal(map[string]string{"text": text})
		return payload
	}
	if err := store.SaveJSON("pheromones.json", colony.PheromoneFile{
		Signals: []colony.PheromoneSignal{
			{ID: "sig_surface", Type: "REDIRECT", Priority: "high", Source: "test", CreatedAt: now, Active: true, Strength: &strength, Content: makeContent("use the current surface stack")},
			{ID: "sig_integration", Type: "FOCUS", Priority: "normal", Source: "test", CreatedAt: now, Active: true, Strength: &strength, Content: makeContent("preserve the current api contract")},
			{ID: "sig_scope", Type: "FEEDBACK", Priority: "low", Source: "test", CreatedAt: now, Active: true, Strength: &strength, Content: makeContent("prefer the smallest scope slice first")},
			{ID: "sig_verification", Type: "FEEDBACK", Priority: "low", Source: "test", CreatedAt: now, Active: true, Strength: &strength, Content: makeContent("keep the test verification bar explicit")},
		},
	}); err != nil {
		t.Fatalf("seed pheromones: %v", err)
	}

	rootCmd.SetArgs([]string{"discuss"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("discuss returned error: %v", err)
	}

	env := parseEnvelope(t, stdout.(*bytes.Buffer).String())
	result := env["result"].(map[string]interface{})
	if status := stringValue(result["discussion_status"]); status != "settled" {
		t.Fatalf("discussion_status = %q, want settled", status)
	}
	if got := int(result["question_count"].(float64)); got != 0 {
		t.Fatalf("question_count = %d, want 0 when all questions are suppressed", got)
	}
}

func TestDiscussSurfacesOrchestratorBoundaryQuestions(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldDir)

	goal := "Build a dashboard for internal operations"
	sessionID := "sess_boundary_discuss_test"
	initializedAt := time.Date(2026, 5, 11, 10, 0, 0, 0, time.UTC)
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:       "3.0",
		Goal:          &goal,
		State:         colony.StateREADY,
		SessionID:     &sessionID,
		InitializedAt: &initializedAt,
	})

	boundarySource := orchestratorBoundaryClarificationSource("plan", 1, "scope", true)
	if err := store.SaveJSON(pendingDecisionsFile, PendingDecisionFile{
		Decisions: []PendingDecision{{
			ID:          "pd_boundary_scope",
			Type:        clarificationDecisionType,
			Description: formatClarificationDescription("What boundary should the plan respect for the first phase?", []string{"narrow scope", "broad scope"}),
			Source:      boundarySource,
			Resolved:    false,
			CreatedAt:   "2026-05-11T10:05:00Z",
			GoalHash:    pendingDecisionGoalHash(goal),
			SessionID:   sessionID,
		}},
	}); err != nil {
		t.Fatalf("seed boundary pending decision: %v", err)
	}

	rootCmd.SetArgs([]string{"discuss"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("discuss returned error: %v", err)
	}

	env := parseEnvelope(t, stdout.(*bytes.Buffer).String())
	result := env["result"].(map[string]interface{})
	totalQuestions := int(result["question_count"].(float64))

	rawQuestions := result["questions"].([]interface{})
	boundaryCount := 0
	for _, rq := range rawQuestions {
		qm := rq.(map[string]interface{})
		if src, _ := qm["source"].(string); strings.HasPrefix(src, "orchestrator:") {
			boundaryCount++
		}
	}
	if boundaryCount == 0 {
		t.Fatalf("expected discuss to surface orchestrator boundary questions, but found 0 among %d total questions", totalQuestions)
	}
	if totalQuestions != 1 {
		t.Fatalf("expected only the explicit orchestrator boundary, got %d", totalQuestions)
	}
}

// TestDiscussSurfacesCandidatesDespiteOldColonyResolvedDecisions verifies that
// discuss surfaces candidates when old-colony resolved decisions share the same
// goal (and thus goal hash) but have a different session. pendingDecisionMatchesScope
// must prefer session ID over goal hash when both are present.
func TestDiscussSurfacesCandidatesDespiteOldColonyResolvedDecisions(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldDir)

	goal := "Build a dashboard for internal operations"
	currentSession := "session_new_colony"
	initializedAt := time.Date(2026, 5, 11, 12, 0, 0, 0, time.UTC)
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:       "3.0",
		Goal:          &goal,
		State:         colony.StateREADY,
		SessionID:     &currentSession,
		InitializedAt: &initializedAt,
	})

	// Seed an old-colony resolved decision with the SAME goal hash but different session.
	oldSession := "session_old_colony"
	goalHash := pendingDecisionGoalHash(goal)
	if err := store.SaveJSON(pendingDecisionsFile, PendingDecisionFile{
		Decisions: []PendingDecision{
			{
				ID:          "pd_old_surface",
				Type:        clarificationDecisionType,
				Description: formatClarificationDescription("Which existing surface should own the first implementation slice?", []string{"admin-app", "new-module"}),
				Source:      discussSource("surface", true),
				Resolved:    true,
				Resolution:  "Use admin-app",
				ResolvedAt:  "2026-05-10T10:00:00Z",
				CreatedAt:   "2026-05-10T09:00:00Z",
				GoalHash:    goalHash,
				SessionID:   oldSession,
			},
		},
	}); err != nil {
		t.Fatalf("seed old-colony resolved decisions: %v", err)
	}

	rootCmd.SetArgs([]string{"discuss"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("discuss returned error: %v", err)
	}

	env := parseEnvelope(t, stdout.(*bytes.Buffer).String())
	result := env["result"].(map[string]interface{})
	if got := int(result["created_count"].(float64)); got != 0 {
		t.Fatalf("created_count = %d, want 0 because stale answers do not authorize generic replacements", got)
	}
	if got := int(result["question_count"].(float64)); got != 0 {
		t.Fatalf("question_count = %d, want 0", got)
	}
	if got := int(result["ignored_stale_count"].(float64)); got != 1 {
		t.Fatalf("ignored_stale_count = %d, want 1 (old session decision should be stale)", got)
	}
}

func TestDetectDecisionConflicts_NoDecisions(t *testing.T) {
	result := detectDecisionConflicts(nil)
	if result != nil {
		t.Fatalf("expected nil for nil input, got %v", result)
	}

	result = detectDecisionConflicts([]PendingDecision{})
	if result != nil {
		t.Fatalf("expected nil for empty input, got %v", result)
	}

	result = detectDecisionConflicts([]PendingDecision{
		{ID: "pd_1", Resolved: true, Resolution: "use Go"},
	})
	if result != nil {
		t.Fatalf("expected nil for single decision, got %v", result)
	}
}

func TestDetectDecisionConflicts_NoConflict(t *testing.T) {
	decisions := []PendingDecision{
		{ID: "pd_1", Resolved: true, Resolution: "use Go"},
		{ID: "pd_2", Resolved: true, Resolution: "add tests"},
	}
	result := detectDecisionConflicts(decisions)
	if result != nil {
		t.Fatalf("expected nil for compatible decisions, got %v", result)
	}
}

func TestDetectDecisionConflicts_DatabaseConflict(t *testing.T) {
	decisions := []PendingDecision{
		{ID: "pd_1", Resolved: true, Resolution: "use PostgreSQL for persistence"},
		{ID: "pd_2", Resolved: true, Resolution: "keep the stack serverless"},
	}
	result := detectDecisionConflicts(decisions)
	if len(result) != 1 {
		t.Fatalf("expected 1 conflict, got %d: %v", len(result), result)
	}
	if !strings.Contains(result[0], "database") {
		t.Fatalf("expected conflict to contain 'database', got %q", result[0])
	}
}

func TestDetectDecisionConflicts_ArchitectureConflict(t *testing.T) {
	decisions := []PendingDecision{
		{ID: "pd_1", Resolved: true, Resolution: "build a monolith for now"},
		{ID: "pd_2", Resolved: true, Resolution: "use microservices from the start"},
	}
	result := detectDecisionConflicts(decisions)
	if len(result) != 1 {
		t.Fatalf("expected 1 conflict, got %d: %v", len(result), result)
	}
	if !strings.Contains(result[0], "architecture") {
		t.Fatalf("expected conflict to contain 'architecture', got %q", result[0])
	}
}

func TestDetectDecisionConflicts_FrontendConflict(t *testing.T) {
	decisions := []PendingDecision{
		{ID: "pd_1", Resolved: true, Resolution: "use React for the UI"},
		{ID: "pd_2", Resolved: true, Resolution: "use Vue for the UI"},
	}
	result := detectDecisionConflicts(decisions)
	if len(result) != 1 {
		t.Fatalf("expected 1 conflict, got %d: %v", len(result), result)
	}
	if !strings.Contains(result[0], "frontend") {
		t.Fatalf("expected conflict to contain 'frontend', got %q", result[0])
	}
}

func TestDetectDecisionConflicts_MultipleConflicts(t *testing.T) {
	decisions := []PendingDecision{
		{ID: "pd_1", Resolved: true, Resolution: "use React for the UI"},
		{ID: "pd_2", Resolved: true, Resolution: "use Vue for the frontend"},
		{ID: "pd_3", Resolved: true, Resolution: "build a monolith"},
		{ID: "pd_4", Resolved: true, Resolution: "use microservices for scaling"},
	}
	result := detectDecisionConflicts(decisions)
	if len(result) != 2 {
		t.Fatalf("expected 2 conflicts, got %d: %v", len(result), result)
	}
	hasFrontend := false
	hasArchitecture := false
	for _, c := range result {
		if strings.Contains(c, "frontend") {
			hasFrontend = true
		}
		if strings.Contains(c, "architecture") {
			hasArchitecture = true
		}
	}
	if !hasFrontend || !hasArchitecture {
		t.Fatalf("expected both frontend and architecture conflicts, got %v", result)
	}
}

func TestDetectDecisionConflicts_UnresolvedIgnored(t *testing.T) {
	decisions := []PendingDecision{
		{ID: "pd_1", Resolved: true, Resolution: "use PostgreSQL for persistence"},
		{ID: "pd_2", Resolved: false, Resolution: "keep the stack serverless"},
	}
	result := detectDecisionConflicts(decisions)
	if result != nil {
		t.Fatalf("expected nil when only one resolved decision, got %v", result)
	}
}

func TestDetectDecisionConflicts_EmptyResolutionIgnored(t *testing.T) {
	decisions := []PendingDecision{
		{ID: "pd_1", Resolved: true, Resolution: "use PostgreSQL for persistence"},
		{ID: "pd_2", Resolved: true, Resolution: ""},
	}
	result := detectDecisionConflicts(decisions)
	if result != nil {
		t.Fatalf("expected nil when one resolution is empty, got %v", result)
	}
}

func TestDiscussSurfacesCandidatesDespiteLegacySameGoalResolvedDecisionWithoutSession(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldDir)

	goal := "Build a dashboard for internal operations"
	currentSession := "session_new_colony"
	initializedAt := time.Date(2026, 5, 11, 12, 0, 0, 0, time.UTC)
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:       "3.0",
		Goal:          &goal,
		State:         colony.StateREADY,
		SessionID:     &currentSession,
		InitializedAt: &initializedAt,
	})

	if err := store.SaveJSON(pendingDecisionsFile, PendingDecisionFile{
		Decisions: []PendingDecision{{
			ID:          "pd_legacy_surface",
			Type:        clarificationDecisionType,
			Description: formatClarificationDescription("Which existing surface should own the first implementation slice?", []string{"admin-app", "new-module"}),
			Source:      discussSource("surface", true),
			Resolved:    true,
			Resolution:  "Use admin-app",
			ResolvedAt:  "2026-05-10T10:00:00Z",
			CreatedAt:   "2026-05-10T09:00:00Z",
			GoalHash:    pendingDecisionGoalHash(goal),
		}},
	}); err != nil {
		t.Fatalf("seed legacy resolved decision: %v", err)
	}

	rootCmd.SetArgs([]string{"discuss"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("discuss returned error: %v", err)
	}

	env := parseEnvelope(t, stdout.(*bytes.Buffer).String())
	result := env["result"].(map[string]interface{})
	if got := int(result["created_count"].(float64)); got != 0 {
		t.Fatalf("created_count = %d, want 0 because legacy state remains evidence rather than a generic prompt template", got)
	}
	if got := int(result["ignored_stale_count"].(float64)); got != 1 {
		t.Fatalf("ignored_stale_count = %d, want 1 legacy stale clarification", got)
	}
}
