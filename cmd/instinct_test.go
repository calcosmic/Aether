package cmd

import (
	"bytes"
	"os"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

func TestInstinctCreatePersistsNewInstinct(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	rootCmd.SetArgs([]string{
		"instinct-create",
		"--trigger", "memory leak",
		"--action", "run pprof",
		"--confidence", "0.85",
		"--domain", "performance",
	})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	if env["ok"] != true {
		t.Fatalf("expected ok:true, got: %v", env)
	}

	result := env["result"].(map[string]interface{})
	if result["status"] != "active" {
		t.Errorf("status = %v, want active", result["status"])
	}
	if result["duplicate"] != false {
		t.Errorf("duplicate = %v, want false", result["duplicate"])
	}
	if result["confidence"] != float64(0.85) {
		t.Errorf("confidence = %v, want 0.85", result["confidence"])
	}

	// Verify filesystem state
	var file colony.InstinctsFile
	if err := s.LoadJSON("instincts.json", &file); err != nil {
		t.Fatalf("load instincts.json: %v", err)
	}
	if len(file.Instincts) != 1 {
		t.Fatalf("expected 1 instinct, got %d", len(file.Instincts))
	}
	inst := file.Instincts[0]
	if inst.Trigger != "memory leak" {
		t.Errorf("trigger = %q, want 'memory leak'", inst.Trigger)
	}
	if inst.Action != "run pprof" {
		t.Errorf("action = %q, want 'run pprof'", inst.Action)
	}
	if inst.Domain != "performance" {
		t.Errorf("domain = %q, want performance", inst.Domain)
	}
	if inst.Archived {
		t.Error("new instinct should not be archived")
	}
}

func TestInstinctCreateReinforcesDuplicate(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	// Create first instinct
	rootCmd.SetArgs([]string{
		"instinct-create",
		"--trigger", "race condition",
		"--action", "add mutex",
		"--confidence", "0.75",
	})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("first create error: %v", err)
	}

	buf.Reset()
	stdout = &buf

	// Create duplicate
	rootCmd.SetArgs([]string{
		"instinct-create",
		"--trigger", "race condition",
		"--action", "add mutex",
		"--confidence", "0.75",
	})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("second create error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	if env["ok"] != true {
		t.Fatalf("expected ok:true, got: %v", env)
	}

	result := env["result"].(map[string]interface{})
	if result["duplicate"] != true {
		t.Errorf("duplicate = %v, want true", result["duplicate"])
	}
	if result["status"] != "reinforced" {
		t.Errorf("status = %v, want reinforced", result["status"])
	}
	if result["confidence"] != float64(0.80) {
		t.Errorf("confidence = %v, want 0.80", result["confidence"])
	}

	// Verify filesystem state: still only 1 instinct
	var file colony.InstinctsFile
	if err := s.LoadJSON("instincts.json", &file); err != nil {
		t.Fatalf("load instincts.json: %v", err)
	}
	if len(file.Instincts) != 1 {
		t.Fatalf("expected 1 instinct after reinforce, got %d", len(file.Instincts))
	}
}

func TestInstinctReadTrustedFiltersByScore(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	// Seed instincts directly
	file := colony.InstinctsFile{
		Version: "1.0",
		Instincts: []colony.InstinctEntry{
			{ID: "high", Trigger: "t1", Action: "a1", TrustScore: 0.90, Confidence: 0.90, TrustTier: "canonical"},
			{ID: "mid", Trigger: "t2", Action: "a2", TrustScore: 0.60, Confidence: 0.60, TrustTier: "emerging"},
			{ID: "low", Trigger: "t3", Action: "a3", TrustScore: 0.30, Confidence: 0.30, TrustTier: "nascent"},
		},
	}
	if err := s.SaveJSON("instincts.json", file); err != nil {
		t.Fatalf("seed instincts.json: %v", err)
	}

	rootCmd.SetArgs([]string{
		"instinct-read-trusted",
		"--min-score", "0.65",
	})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	if env["ok"] != true {
		t.Fatalf("expected ok:true, got: %v", env)
	}

	result := env["result"].(map[string]interface{})
	if result["count"] != float64(1) {
		t.Errorf("count = %v, want 1", result["count"])
	}

	instincts := result["instincts"].([]interface{})
	if len(instincts) != 1 {
		t.Fatalf("instincts length = %d, want 1", len(instincts))
	}
	inst := instincts[0].(map[string]interface{})
	if inst["id"] != "high" {
		t.Errorf("id = %v, want high", inst["id"])
	}
}

func TestInstinctDecayAllReducesTrustScores(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	// Seed instincts
	file := colony.InstinctsFile{
		Version: "1.0",
		Instincts: []colony.InstinctEntry{
			{ID: "decay1", Trigger: "t1", Action: "a1", TrustScore: 0.90, Confidence: 0.90, TrustTier: "canonical"},
			{ID: "decay2", Trigger: "t2", Action: "a2", TrustScore: 0.50, Confidence: 0.50, TrustTier: "emerging"},
		},
	}
	if err := s.SaveJSON("instincts.json", file); err != nil {
		t.Fatalf("seed instincts.json: %v", err)
	}

	rootCmd.SetArgs([]string{
		"instinct-decay-all",
		"--days", "60",
	})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	if env["ok"] != true {
		t.Fatalf("expected ok:true, got: %v", env)
	}

	result := env["result"].(map[string]interface{})
	if result["processed"] != float64(2) {
		t.Errorf("processed = %v, want 2", result["processed"])
	}

	// Verify filesystem state: scores should be lower
	var updated colony.InstinctsFile
	if err := s.LoadJSON("instincts.json", &updated); err != nil {
		t.Fatalf("load instincts.json: %v", err)
	}
	if len(updated.Instincts) != 2 {
		t.Fatalf("expected 2 instincts, got %d", len(updated.Instincts))
	}
	if updated.Instincts[0].TrustScore >= 0.90 {
		t.Errorf("trust_score should have decayed, got %v", updated.Instincts[0].TrustScore)
	}
}

func TestInstinctDecayAllDryRunDoesNotSave(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	// Seed instincts
	file := colony.InstinctsFile{
		Version: "1.0",
		Instincts: []colony.InstinctEntry{
			{ID: "dry1", Trigger: "t1", Action: "a1", TrustScore: 0.90, Confidence: 0.90, TrustTier: "canonical"},
		},
	}
	if err := s.SaveJSON("instincts.json", file); err != nil {
		t.Fatalf("seed instincts.json: %v", err)
	}

	rootCmd.SetArgs([]string{
		"instinct-decay-all",
		"--days", "60",
		"--dry-run",
	})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	if env["ok"] != true {
		t.Fatalf("expected ok:true, got: %v", env)
	}

	result := env["result"].(map[string]interface{})
	if result["dry_run"] != true {
		t.Errorf("dry_run = %v, want true", result["dry_run"])
	}

	// Verify filesystem state: score should remain unchanged
	var updated colony.InstinctsFile
	if err := s.LoadJSON("instincts.json", &updated); err != nil {
		t.Fatalf("load instincts.json: %v", err)
	}
	if updated.Instincts[0].TrustScore != 0.90 {
		t.Errorf("dry-run should not modify scores, got %v", updated.Instincts[0].TrustScore)
	}
}

func TestInstinctArchiveMarksAsArchived(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	// Seed instincts
	file := colony.InstinctsFile{
		Version: "1.0",
		Instincts: []colony.InstinctEntry{
			{ID: "arch-1", Trigger: "t1", Action: "a1", TrustScore: 0.80, Confidence: 0.80, TrustTier: "trusted", Archived: false},
		},
	}
	if err := s.SaveJSON("instincts.json", file); err != nil {
		t.Fatalf("seed instincts.json: %v", err)
	}

	rootCmd.SetArgs([]string{
		"instinct-archive",
		"--id", "arch-1",
	})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	if env["ok"] != true {
		t.Fatalf("expected ok:true, got: %v", env)
	}

	result := env["result"].(map[string]interface{})
	if result["archived"] != "arch-1" {
		t.Errorf("archived = %v, want arch-1", result["archived"])
	}

	// Verify filesystem state
	var updated colony.InstinctsFile
	if err := s.LoadJSON("instincts.json", &updated); err != nil {
		t.Fatalf("load instincts.json: %v", err)
	}
	if !updated.Instincts[0].Archived {
		t.Error("instinct should be archived")
	}
}
