package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/aether-colony/aether/pkg/storage"
)

func TestPheromoneRead(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf
	

	s, tmpDir := setupTestStore(t)
	defer os.RemoveAll(tmpDir)

	os.Setenv("AETHER_ROOT", tmpDir)
	defer os.Setenv("AETHER_ROOT", os.Getenv("AETHER_ROOT"))

	store = s
	rootCmd.SetArgs([]string{"pheromone-read"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("pheromone-read returned error: %v", err)
	}

	output := strings.TrimSpace(buf.String())

	// Verify JSON envelope
	var envelope map[string]interface{}
	if err := json.Unmarshal([]byte(output), &envelope); err != nil {
		t.Fatalf("pheromone-read produced invalid JSON: %v", err)
	}
	if envelope["ok"] != true {
		t.Errorf("ok = %v, want true", envelope["ok"])
	}

	result, ok := envelope["result"].(map[string]interface{})
	if !ok {
		t.Fatal("result is not a map")
	}

	signals, ok := result["signals"].([]interface{})
	if !ok {
		t.Fatal("result.signals is not an array")
	}

	// Testdata has 3 active signals
	if len(signals) != 3 {
		t.Errorf("expected 3 signals, got %d", len(signals))
	}
}

func TestPheromoneReadEmpty(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf
	

	// Create empty store with no pheromones
	tmpDir := t.TempDir()
	dataDir := tmpDir + "/.aether/data"
	os.MkdirAll(dataDir, 0755)

	os.Setenv("AETHER_ROOT", tmpDir)
	defer os.Setenv("AETHER_ROOT", os.Getenv("AETHER_ROOT"))

	s, _ := storage.NewStore(dataDir)
	store = s

	rootCmd.SetArgs([]string{"pheromone-read"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("pheromone-read returned error: %v", err)
	}

	output := strings.TrimSpace(buf.String())
	if !strings.Contains(output, `"ok":true`) {
		t.Errorf("expected ok:true for empty pheromones, got: %s", output)
	}
	if !strings.Contains(output, `"signals":[]`) {
		t.Errorf("expected empty signals array, got: %s", output)
	}
}

func TestPheromoneCount(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf
	

	s, tmpDir := setupTestStore(t)
	defer os.RemoveAll(tmpDir)

	os.Setenv("AETHER_ROOT", tmpDir)
	defer os.Setenv("AETHER_ROOT", os.Getenv("AETHER_ROOT"))

	store = s
	rootCmd.SetArgs([]string{"pheromone-count"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("pheromone-count returned error: %v", err)
	}

	output := strings.TrimSpace(buf.String())

	var envelope map[string]interface{}
	if err := json.Unmarshal([]byte(output), &envelope); err != nil {
		t.Fatalf("pheromone-count produced invalid JSON: %v", err)
	}

	result, ok := envelope["result"].(map[string]interface{})
	if !ok {
		t.Fatal("result is not a map")
	}

	// Testdata: 1 FOCUS, 1 REDIRECT, 1 FEEDBACK
	if result["focus"] != float64(1) {
		t.Errorf("focus = %v, want 1", result["focus"])
	}
	if result["redirect"] != float64(1) {
		t.Errorf("redirect = %v, want 1", result["redirect"])
	}
	if result["feedback"] != float64(1) {
		t.Errorf("feedback = %v, want 1", result["feedback"])
	}
}

func TestPheromoneCountEmpty(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf
	

	tmpDir := t.TempDir()
	dataDir := tmpDir + "/.aether/data"
	os.MkdirAll(dataDir, 0755)

	os.Setenv("AETHER_ROOT", tmpDir)
	defer os.Setenv("AETHER_ROOT", os.Getenv("AETHER_ROOT"))

	s, _ := storage.NewStore(dataDir)
	store = s

	rootCmd.SetArgs([]string{"pheromone-count"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("pheromone-count returned error: %v", err)
	}

	output := strings.TrimSpace(buf.String())
	var envelope map[string]interface{}
	json.Unmarshal([]byte(output), &envelope)

	result := envelope["result"].(map[string]interface{})
	if result["focus"] != float64(0) {
		t.Errorf("focus = %v, want 0 for empty store", result["focus"])
	}
}
