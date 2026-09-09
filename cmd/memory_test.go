package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

func TestMemoryMetrics(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer

	stdout = &buf

	bindSeededCommandTestRepository(t)
	rootCmd.SetArgs([]string{"memory-metrics"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("memory-metrics returned error: %v", err)
	}

	output := strings.TrimSpace(buf.String())

	var envelope map[string]interface{}
	if err := json.Unmarshal([]byte(output), &envelope); err != nil {
		t.Fatalf("memory-metrics produced invalid JSON: %v", err)
	}
	if envelope["ok"] != true {
		t.Errorf("ok = %v, want true", envelope["ok"])
	}

	result, ok := envelope["result"].(map[string]interface{})
	if !ok {
		t.Fatal("result is not a map")
	}

	// Should have wisdom, pending, recent_failures, last_activity keys
	for _, key := range []string{"wisdom", "pending", "recent_failures", "last_activity"} {
		if _, exists := result[key]; !exists {
			t.Errorf("result missing key %q", key)
		}
	}
}

func TestMemoryMetricsEmpty(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer

	stdout = &buf

	bindCommandTestRepository(t)

	rootCmd.SetArgs([]string{"memory-metrics"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("memory-metrics returned error on empty store: %v", err)
	}

	output := strings.TrimSpace(buf.String())
	if !strings.Contains(output, `"ok":true`) {
		t.Errorf("expected ok:true, got: %s", output)
	}
}

func TestColonyVitalSigns(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer

	stdout = &buf

	bindSeededCommandTestRepository(t)
	rootCmd.SetArgs([]string{"colony-vital-signs"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("colony-vital-signs returned error: %v", err)
	}

	output := strings.TrimSpace(buf.String())

	var envelope map[string]interface{}
	if err := json.Unmarshal([]byte(output), &envelope); err != nil {
		t.Fatalf("colony-vital-signs produced invalid JSON: %v", err)
	}

	result, ok := envelope["result"].(map[string]interface{})
	if !ok {
		t.Fatal("result is not a map")
	}

	// Should have health metrics
	for _, key := range []string{"build_velocity", "error_rate", "signal_health", "memory_pressure", "overall_health", "health_label"} {
		if _, exists := result[key]; !exists {
			t.Errorf("result missing key %q", key)
		}
	}

	// Health label should be a known value
	label := result["health_label"].(string)
	validLabels := map[string]bool{"Thriving": true, "Healthy": true, "Stable": true, "Struggling": true, "Critical": true}
	if !validLabels[label] {
		t.Errorf("health_label = %q, want one of Thriving/Healthy/Stable/Struggling/Critical", label)
	}
}

func TestColonyVitalSignsUsesStandaloneInstincts(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer

	stdout = &buf

	binding := bindSeededCommandTestRepository(t)
	s := binding.Store

	var state colony.ColonyState
	if err := s.LoadJSON("COLONY_STATE.json", &state); err != nil {
		t.Fatalf("load colony state: %v", err)
	}
	state.Memory.Instincts = []colony.Instinct{}
	if err := s.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatalf("save colony state: %v", err)
	}

	instincts := colony.InstinctsFile{
		Version: "1.0",
		Instincts: []colony.InstinctEntry{
			{
				ID:         "inst_1",
				Trigger:    "one",
				Action:     "two",
				Confidence: 0.8,
				TrustScore: 0.8,
				TrustTier:  "trusted",
				Provenance: colony.InstinctProvenance{CreatedAt: "2026-04-01T00:00:00Z"},
			},
		},
	}
	if err := s.SaveJSON("instincts.json", instincts); err != nil {
		t.Fatalf("save instincts: %v", err)
	}

	rootCmd.SetArgs([]string{"colony-vital-signs"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("colony-vital-signs returned error: %v", err)
	}

	output := strings.TrimSpace(buf.String())

	var envelope map[string]interface{}
	if err := json.Unmarshal([]byte(output), &envelope); err != nil {
		t.Fatalf("colony-vital-signs produced invalid JSON: %v", err)
	}
	result := envelope["result"].(map[string]interface{})
	memoryPressure := result["memory_pressure"].(map[string]interface{})
	if memoryPressure["instinct_count"] != float64(1) {
		t.Fatalf("instinct_count = %v, want 1", memoryPressure["instinct_count"])
	}
}

func TestMemoryMetricsReportsApplicationAwareHealth(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer

	stdout = &buf

	binding := bindCommandTestRepository(t)
	s := binding.Store

	writeTestJSON(t, s.BasePath(), "learning-observations.json", map[string]interface{}{
		"observations": []interface{}{
			map[string]interface{}{
				"content_hash":      "obs_pending",
				"content":           "pending promotion",
				"wisdom_type":       "pattern",
				"observation_count": 3,
				"first_seen":        "2026-03-01T10:00:00Z",
				"last_seen":         "2026-03-10T10:00:00Z",
				"colonies":          []interface{}{"test-colony"},
				"trust_score":       0.7,
			},
		},
	})
	writeTestJSON(t, s.BasePath(), "instincts.json", map[string]interface{}{
		"version": "1.0",
		"instincts": []interface{}{
			map[string]interface{}{
				"id":          "inst_review",
				"trigger":     "review trigger",
				"action":      "review action",
				"confidence":  0.6,
				"trust_score": 0.4,
				"provenance": map[string]interface{}{
					"created_at":        "2026-01-01T10:00:00Z",
					"last_applied":      "2026-04-21T10:00:00Z",
					"application_count": 2,
				},
				"application_history": []interface{}{
					map[string]interface{}{"timestamp": "2026-04-20T10:00:00Z", "success": false},
					map[string]interface{}{"timestamp": "2026-04-21T10:00:00Z", "success": false},
				},
				"related_instincts": []interface{}{},
				"archived":          false,
			},
		},
	})

	rootCmd.SetArgs([]string{"memory-metrics"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("memory-metrics returned error: %v", err)
	}

	output := strings.TrimSpace(buf.String())
	var envelope map[string]interface{}
	if err := json.Unmarshal([]byte(output), &envelope); err != nil {
		t.Fatalf("memory-metrics produced invalid JSON: %v", err)
	}
	result := envelope["result"].(map[string]interface{})
	pending := result["pending"].(map[string]interface{})
	if pending["total"] != float64(1) {
		t.Fatalf("pending total = %v, want 1", pending["total"])
	}
	instincts := result["instincts"].(map[string]interface{})
	if instincts["applied"] != float64(1) {
		t.Fatalf("applied instincts = %v, want 1", instincts["applied"])
	}
	curation := result["curation"].(map[string]interface{})
	if curation["review_candidates"] != float64(1) {
		t.Fatalf("review_candidates = %v, want 1", curation["review_candidates"])
	}
}

func TestMemoryPhaseRestoreRepositoryAuthority200(t *testing.T) {
	tests := []struct {
		name string
		run  func(*testing.T)
	}{
		{name: "memory", run: TestMemoryMetrics},
		{name: "phase", run: TestPhaseJSON},
	}

	orders := []struct {
		name  string
		tests []int
	}{
		{name: "memory then phase", tests: []int{0, 1}},
		{name: "phase then memory", tests: []int{1, 0}},
	}

	for _, order := range orders {
		t.Run(order.name, func(t *testing.T) {
			originalRoot := t.TempDir()
			originalDataDir := originalRoot + "/.aether/data"
			if err := os.MkdirAll(originalDataDir, 0755); err != nil {
				t.Fatalf("create original data directory: %v", err)
			}
			t.Setenv("AETHER_ROOT", originalRoot)
			t.Setenv("COLONY_DATA_DIR", originalDataDir)
			store = nil
			tracer = nil

			for _, testIndex := range order.tests {
				tt := tests[testIndex]
				t.Run(tt.name, tt.run)

				if got := os.Getenv("AETHER_ROOT"); got != originalRoot {
					t.Errorf("AETHER_ROOT = %q after %s cleanup, want %q", got, tt.name, originalRoot)
				}
				if got := os.Getenv("COLONY_DATA_DIR"); got != originalDataDir {
					t.Errorf("COLONY_DATA_DIR = %q after %s cleanup, want %q", got, tt.name, originalDataDir)
				}
				if store != nil || tracer != nil {
					t.Errorf("repository authority leaked after %s cleanup: store=%p tracer=%p", tt.name, store, tracer)
				}
			}
		})
	}
}
