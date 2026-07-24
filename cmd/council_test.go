package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"
)

func TestCouncilDeliberateCreatesTopic(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	rootCmd.SetArgs([]string{"council-deliberate", "--topic", "Should we refactor?"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("council-deliberate returned error: %v", err)
	}

	out := parseEnvelope(t, buf.String())
	if out["ok"] != true {
		t.Fatalf("expected ok=true, got: %v", out["ok"])
	}
	result := out["result"].(map[string]interface{})
	if result["initiated"] != true {
		t.Fatalf("expected initiated=true, got: %v", result["initiated"])
	}

	// Verify filesystem
	historyPath := tmpDir + "/.aether/data/council/history.json"
	raw, err := os.ReadFile(historyPath)
	if err != nil {
		t.Fatalf("read history.json: %v", err)
	}
	var ch councilHistoryData
	if err := json.Unmarshal(raw, &ch); err != nil {
		t.Fatalf("parse history.json: %v", err)
	}
	if len(ch.Deliberations) != 1 {
		t.Fatalf("expected 1 deliberation, got %d", len(ch.Deliberations))
	}
	if ch.Deliberations[0].Topic != "Should we refactor?" {
		t.Fatalf("expected topic match, got %q", ch.Deliberations[0].Topic)
	}
}

func TestCouncilDeliberateDuplicateTopic(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	rootCmd.SetArgs([]string{"council-deliberate", "--topic", "Same topic"})
	_ = rootCmd.Execute()

	resetRootCmd(t)
	stdout = &buf
	buf.Reset()
	rootCmd.SetArgs([]string{"council-deliberate", "--topic", "Same topic"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("council-deliberate returned error: %v", err)
	}

	out := parseEnvelope(t, buf.String())
	result := out["result"].(map[string]interface{})
	if result["initiated"] != false {
		t.Fatalf("expected initiated=false for duplicate, got: %v", result["initiated"])
	}
	if result["reason"] != "topic already exists" {
		t.Fatalf("expected reason 'topic already exists', got: %q", result["reason"])
	}
}

func TestCouncilAdvocateSubmitsPosition(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	rootCmd.SetArgs([]string{"council-deliberate", "--topic", "Architecture"})
	_ = rootCmd.Execute()

	resetRootCmd(t)
	stdout = &buf
	buf.Reset()
	rootCmd.SetArgs([]string{"council-advocate", "--topic", "Architecture", "--position", "Microservices are better"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("council-advocate returned error: %v", err)
	}

	out := parseEnvelope(t, buf.String())
	if out["ok"] != true {
		t.Fatalf("expected ok=true, got: %v", out["ok"])
	}
	result := out["result"].(map[string]interface{})
	if result["submitted"] != true {
		t.Fatalf("expected submitted=true, got: %v", result["submitted"])
	}
	if result["role"] != "advocate" {
		t.Fatalf("expected role=advocate, got: %q", result["role"])
	}

	// Verify history
	historyPath := tmpDir + "/.aether/data/council/history.json"
	raw, _ := os.ReadFile(historyPath)
	var ch councilHistoryData
	json.Unmarshal(raw, &ch)
	if len(ch.Deliberations[0].Positions) != 1 {
		t.Fatalf("expected 1 position, got %d", len(ch.Deliberations[0].Positions))
	}
	if ch.Deliberations[0].Positions[0].Role != "advocate" {
		t.Fatalf("expected advocate position, got %q", ch.Deliberations[0].Positions[0].Role)
	}
}

func TestCouncilChallengerSubmitsPosition(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	rootCmd.SetArgs([]string{"council-deliberate", "--topic", "Architecture"})
	_ = rootCmd.Execute()

	resetRootCmd(t)
	stdout = &buf
	buf.Reset()
	rootCmd.SetArgs([]string{"council-challenger", "--topic", "Architecture", "--challenge", "Monoliths are simpler"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("council-challenger returned error: %v", err)
	}

	out := parseEnvelope(t, buf.String())
	result := out["result"].(map[string]interface{})
	if result["submitted"] != true {
		t.Fatalf("expected submitted=true, got: %v", result["submitted"])
	}
	if result["role"] != "challenger" {
		t.Fatalf("expected role=challenger, got: %q", result["role"])
	}
}

func TestCouncilSageSubmitsWisdom(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	rootCmd.SetArgs([]string{"council-deliberate", "--topic", "Architecture"})
	_ = rootCmd.Execute()

	resetRootCmd(t)
	stdout = &buf
	buf.Reset()
	rootCmd.SetArgs([]string{"council-sage", "--topic", "Architecture", "--wisdom", "Consider hybrid approach"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("council-sage returned error: %v", err)
	}

	out := parseEnvelope(t, buf.String())
	result := out["result"].(map[string]interface{})
	if result["submitted"] != true {
		t.Fatalf("expected submitted=true, got: %v", result["submitted"])
	}
	if result["role"] != "sage" {
		t.Fatalf("expected role=sage, got: %q", result["role"])
	}
}

func TestCouncilHistoryReturnsDeliberations(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	rootCmd.SetArgs([]string{"council-deliberate", "--topic", "Topic A"})
	_ = rootCmd.Execute()

	resetRootCmd(t)
	stdout = &buf
	buf.Reset()
	rootCmd.SetArgs([]string{"council-history"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("council-history returned error: %v", err)
	}

	out := parseEnvelope(t, buf.String())
	if out["ok"] != true {
		t.Fatalf("expected ok=true, got: %v", out["ok"])
	}
	result := out["result"].(map[string]interface{})
	total := result["total"].(float64)
	if total != 1 {
		t.Fatalf("expected total=1, got %v", total)
	}
	deliberations, ok := result["deliberations"].([]interface{})
	if !ok {
		t.Fatalf("expected deliberations array, got: %T", result["deliberations"])
	}
	if len(deliberations) != 1 {
		t.Fatalf("expected 1 deliberation, got %d", len(deliberations))
	}
}

func TestCouncilBudgetCheck(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	rootCmd.SetArgs([]string{"council-deliberate", "--topic", "Budget test"})
	_ = rootCmd.Execute()

	resetRootCmd(t)
	stdout = &buf
	buf.Reset()
	rootCmd.SetArgs([]string{"council-advocate", "--topic", "Budget test", "--position", "Yes"})
	_ = rootCmd.Execute()

	resetRootCmd(t)
	stdout = &buf
	buf.Reset()
	rootCmd.SetArgs([]string{"council-budget-check"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("council-budget-check returned error: %v", err)
	}

	out := parseEnvelope(t, buf.String())
	if out["ok"] != true {
		t.Fatalf("expected ok=true, got: %v", out["ok"])
	}
	result := out["result"].(map[string]interface{})
	topics := result["topics"].(float64)
	positions := result["positions"].(float64)
	remaining := result["remaining"].(float64)
	maxBudget := result["max_budget"].(float64)

	if topics != 1 {
		t.Fatalf("expected topics=1, got %v", topics)
	}
	if positions != 1 {
		t.Fatalf("expected positions=1, got %v", positions)
	}
	if maxBudget != float64(maxDeliberations*maxPositionsPerTopic) {
		t.Fatalf("expected max_budget=%d, got %v", maxDeliberations*maxPositionsPerTopic, maxBudget)
	}
	expectedRemaining := maxBudget - 1
	if remaining != expectedRemaining {
		t.Fatalf("expected remaining=%v, got %v", expectedRemaining, remaining)
	}
}
