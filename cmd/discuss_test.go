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

func TestDiscussCreatesClarificationQuestions(t *testing.T) {
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
	if got := int(result["question_count"].(float64)); got != 3 {
		t.Fatalf("question_count = %d, want 3", got)
	}
	if got := int(result["created_count"].(float64)); got != 3 {
		t.Fatalf("created_count = %d, want 3", got)
	}

	var file PendingDecisionFile
	if err := store.LoadJSON(pendingDecisionsFile, &file); err != nil {
		t.Fatalf("load pending decisions: %v", err)
	}
	if len(file.Decisions) != 3 {
		t.Fatalf("expected 3 clarification decisions, got %d", len(file.Decisions))
	}
	for _, decision := range file.Decisions {
		if decision.Type != clarificationDecisionType {
			t.Fatalf("decision type = %q, want clarification", decision.Type)
		}
		if decision.Resolved {
			t.Fatal("new discussion questions should be unresolved")
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
	if got := int(result["existing_count"].(float64)); got != 3 {
		t.Fatalf("existing_count = %d, want 3 on repeat run", got)
	}
	if status := stringValue(result["discussion_status"]); status != "pending_questions" {
		t.Fatalf("discussion_status = %q, want pending_questions", status)
	}

	var file PendingDecisionFile
	if err := store.LoadJSON(pendingDecisionsFile, &file); err != nil {
		t.Fatalf("load pending decisions: %v", err)
	}
	if len(file.Decisions) != 3 {
		t.Fatalf("expected 3 clarification decisions after repeat run, got %d", len(file.Decisions))
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
	if got := int(result["question_count"].(float64)); got == 0 {
		t.Fatal("expected dry-run to preview clarification questions")
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
