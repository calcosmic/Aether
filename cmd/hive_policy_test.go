package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

func TestHiveWorkerReadIsOffByDefaultButManualReadRemainsAvailable(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	t.Setenv(hivePolicyEnv, "")
	hub := t.TempDir()
	t.Setenv("AETHER_HUB_DIR", hub)
	if err := os.MkdirAll(filepath.Join(hub, "hive"), 0o755); err != nil {
		t.Fatalf("create hive: %v", err)
	}
	data := `{"entries":[{"id":"go-safe","text":"Prefer table tests","domain":"go","source_repo":"one","confidence":0.9}]}`
	if err := os.WriteFile(filepath.Join(hub, "hive", "wisdom.json"), []byte(data), 0o644); err != nil {
		t.Fatalf("write wisdom: %v", err)
	}

	resetFlags(rootCmd)
	stdout = &bytes.Buffer{}
	rootCmd.SetArgs([]string{"hive-read", "--for-worker"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("worker hive-read: %v", err)
	}
	workerEnvelope := parseEnvelope(t, stdout.(*bytes.Buffer).String())
	workerResult := workerEnvelope["result"].(map[string]interface{})
	if workerResult["enabled"] != false || int(workerResult["total"].(float64)) != 0 {
		t.Fatalf("default worker read leaked hive wisdom: %+v", workerResult)
	}

	resetFlags(rootCmd)
	stdout = &bytes.Buffer{}
	rootCmd.SetArgs([]string{"hive-read"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("manual hive-read: %v", err)
	}
	manualEnvelope := parseEnvelope(t, stdout.(*bytes.Buffer).String())
	manualResult := manualEnvelope["result"].(map[string]interface{})
	if int(manualResult["total"].(float64)) != 1 {
		t.Fatalf("manual hive-read should remain inspectable: %+v", manualResult)
	}
}

func TestSealLeavesEligibleInstinctsLocalWhenHivePolicyIsOff(t *testing.T) {
	saveGlobals(t)
	t.Setenv(hivePolicyEnv, "off")
	s, _ := newTestStore(t)
	store = s
	t.Setenv("AETHER_HUB_DIR", t.TempDir())
	stdout = &bytes.Buffer{}
	goal := "Keep project knowledge scoped"
	state := colony.ColonyState{
		Version: "3.0", Goal: &goal, State: colony.StateREADY,
		Plan: colony.Plan{Phases: []colony.Phase{{ID: 1, Name: "Done", Status: colony.PhaseCompleted}}},
	}
	instincts := colony.InstinctsFile{Version: "1.0", Instincts: []colony.InstinctEntry{{
		ID: "local-only", Trigger: "test", Action: "Use project-specific convention", Domain: "go", Confidence: 0.95,
	}}}
	if err := store.SaveJSON("instincts.json", instincts); err != nil {
		t.Fatalf("save instincts: %v", err)
	}
	if err := completeSealRuntime(state); err != nil {
		t.Fatalf("complete seal: %v", err)
	}
	if !strings.Contains(stdout.(*bytes.Buffer).String(), "Hive auto-promotion is disabled") {
		t.Fatalf("seal did not disclose disabled promotion: %s", stdout.(*bytes.Buffer).String())
	}
	wisdomPath := filepath.Join(resolveHubPath(), "hive", "wisdom.json")
	if data, err := os.ReadFile(wisdomPath); err == nil {
		var wisdom hiveWisdomData
		if json.Unmarshal(data, &wisdom) == nil && len(wisdom.Entries) > 0 {
			t.Fatalf("seal promoted project instinct while policy was off: %+v", wisdom.Entries)
		}
	}
}

func TestAutomaticHivePromoteCommandHonorsSafeDefault(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	t.Setenv(hivePolicyEnv, "off")
	hub := t.TempDir()
	t.Setenv("AETHER_HUB_DIR", hub)
	stdout = &bytes.Buffer{}
	rootCmd.SetArgs([]string{
		"hive-promote",
		"--automatic",
		"--text", "Prefer the project-specific layout",
		"--domain", "go",
		"--source-repo", "project-one",
		"--confidence", "0.9",
	})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("automatic hive-promote: %v", err)
	}
	envelope := parseEnvelope(t, stdout.(*bytes.Buffer).String())
	result := envelope["result"].(map[string]interface{})
	if result["promoted"] != false || result["skipped"] != true || result["policy"] != "off" {
		t.Fatalf("automatic promotion did not fail closed: %+v", result)
	}
	if _, err := os.Stat(filepath.Join(hub, "hive", "wisdom.json")); !os.IsNotExist(err) {
		t.Fatalf("automatic promotion wrote global wisdom while disabled: %v", err)
	}
}
