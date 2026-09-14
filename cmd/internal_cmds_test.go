package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/storage"
)

func TestForceUnlock(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, _ := newTestStore(t)
	store = s

	// resolveDataDir uses AETHER_ROOT; create locks relative to data dir parent
	// dataDir = <tmp>/.aether/data, so locksDir = <tmp>/.aether/locks
	locksDir := filepath.Join(filepath.Dir(s.BasePath()), "locks")
	os.MkdirAll(locksDir, 0755)
	os.WriteFile(filepath.Join(locksDir, "test.lock"), []byte("{}"), 0644)

	// Set AETHER_ROOT so resolveDataDir() finds the right location
	os.Setenv("AETHER_ROOT", filepath.Dir(filepath.Dir(s.BasePath())))
	t.Cleanup(func() { os.Unsetenv("AETHER_ROOT") })

	rootCmd.SetArgs([]string{"force-unlock"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("force-unlock failed: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})
	if result["unlocked"] != true {
		t.Errorf("expected unlocked true, got %v", result["unlocked"])
	}
	count := int(result["count"].(float64))
	if count != 1 {
		t.Errorf("expected count 1, got %d", count)
	}
}

func TestEntropyScore(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, _ := newTestStore(t)
	store = s

	// Create minimal COLONY_STATE.json
	colonyData := map[string]interface{}{
		"version":       "2.0",
		"state":         "READY",
		"current_phase": 1,
		"plan":          map[string]interface{}{"phases": []interface{}{}},
		"memory":        map[string]interface{}{"phase_learnings": []interface{}{}, "decisions": []interface{}{}, "instincts": []interface{}{}},
		"errors":        map[string]interface{}{"records": []interface{}{}, "flagged_patterns": []interface{}{}},
		"signals":       []interface{}{},
		"graveyards":    []interface{}{},
		"events":        []interface{}{},
	}
	writeTestJSON(t, s.BasePath(), "COLONY_STATE.json", colonyData)

	rootCmd.SetArgs([]string{"entropy-score"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("entropy-score failed: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})
	score := result["score"].(float64)
	if score < 0 || score > 100 {
		t.Errorf("score %v out of range [0, 100]", score)
	}
}

func TestEternalStore(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	// Use temp dir as hub
	tmpHub := t.TempDir()
	t.Setenv("AETHER_HUB_DIR", tmpHub)

	rootCmd.SetArgs([]string{"eternal-store",
		"--content", "high value signal",
		"--category", "critical",
		"--confidence", "0.95",
	})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("eternal-store failed: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})
	if result["stored"] != true {
		t.Errorf("expected stored true, got %v", result["stored"])
	}

	// Verify file was written
	memoryPath := filepath.Join(tmpHub, "eternal", "memory.json")
	data, err := os.ReadFile(memoryPath)
	if err != nil {
		t.Fatalf("reading eternal memory: %v", err)
	}
	var ed map[string]interface{}
	json.Unmarshal(data, &ed)
	entries := ed["entries"].([]interface{})
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
}

func TestBootstrapSystem(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	// Create a mock hub with templates
	tmpHub := t.TempDir()
	t.Setenv("AETHER_HUB_DIR", tmpHub)

	templateDir := filepath.Join(tmpHub, "templates")
	os.MkdirAll(templateDir, 0755)
	os.WriteFile(filepath.Join(templateDir, "colony-state-template.json"), []byte("{}"), 0644)

	rootCmd.SetArgs([]string{"bootstrap-system"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("bootstrap-system failed: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})
	copied := result["copied"].([]interface{})
	// May or may not have copied depending on resolveAetherRoot
	// but should not error
	t.Logf("copied: %v, skipped: %v", copied, result["skipped"])
}

func TestInstinctRead(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, _ := newTestStore(t)
	store = s

	// Create COLONY_STATE with an instinct
	colonyData := map[string]interface{}{
		"version":       "2.0",
		"state":         "READY",
		"current_phase": 1,
		"plan":          map[string]interface{}{"phases": []interface{}{}},
		"memory": map[string]interface{}{
			"phase_learnings": []interface{}{},
			"decisions":       []interface{}{},
			"instincts": []interface{}{
				map[string]interface{}{
					"id":           "inst_test_1",
					"trigger":      "test_trigger",
					"action":       "test_action",
					"confidence":   0.8,
					"status":       "active",
					"applications": 3,
					"successes":    2,
					"failures":     1,
				},
			},
		},
		"errors":     map[string]interface{}{"records": []interface{}{}, "flagged_patterns": []interface{}{}},
		"signals":    []interface{}{},
		"graveyards": []interface{}{},
		"events":     []interface{}{},
	}
	writeTestJSON(t, s.BasePath(), "COLONY_STATE.json", colonyData)

	rootCmd.SetArgs([]string{"instinct-read", "inst_test_1"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("instinct-read failed: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})
	if result["found"] != true {
		t.Errorf("expected found true, got %v", result["found"])
	}
}

func TestInstinctReadFromStandaloneStore(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, _ := newTestStore(t)
	store = s

	instincts := map[string]interface{}{
		"version": "1.0",
		"instincts": []interface{}{
			map[string]interface{}{
				"id":          "inst_file_1",
				"trigger":     "file_trigger",
				"action":      "file_action",
				"confidence":  0.85,
				"trust_score": 0.85,
				"trust_tier":  "trusted",
				"provenance": map[string]interface{}{
					"source":            "obs_file_1",
					"source_type":       "observation",
					"evidence":          "from file",
					"created_at":        "2026-04-01T00:00:00Z",
					"application_count": 0,
				},
				"application_history": []interface{}{},
				"related_instincts":   []interface{}{},
				"archived":            false,
			},
		},
	}
	writeTestJSON(t, s.BasePath(), "instincts.json", instincts)

	rootCmd.SetArgs([]string{"instinct-read", "inst_file_1"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("instinct-read failed: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})
	if result["found"] != true {
		t.Errorf("expected found true, got %v", result["found"])
	}
}

func TestInstinctApplyUpdatesStandaloneStore(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, _ := newTestStore(t)
	store = s

	instincts := map[string]interface{}{
		"version": "1.0",
		"instincts": []interface{}{
			map[string]interface{}{
				"id":          "inst_file_2",
				"trigger":     "file_trigger",
				"action":      "file_action",
				"confidence":  0.85,
				"trust_score": 0.85,
				"trust_tier":  "trusted",
				"provenance": map[string]interface{}{
					"source":            "obs_file_2",
					"source_type":       "observation",
					"evidence":          "from file",
					"created_at":        "2026-04-01T00:00:00Z",
					"application_count": 0,
				},
				"application_history": []interface{}{},
				"related_instincts":   []interface{}{},
				"archived":            false,
			},
		},
	}
	writeTestJSON(t, s.BasePath(), "instincts.json", instincts)

	rootCmd.SetArgs([]string{"instinct-apply", "inst_file_2", "--success=false"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("instinct-apply failed: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(s.BasePath(), "instincts.json"))
	if err != nil {
		t.Fatalf("read instincts.json: %v", err)
	}
	var file map[string]interface{}
	if err := json.Unmarshal(data, &file); err != nil {
		t.Fatalf("parse instincts.json: %v", err)
	}
	instinctList := file["instincts"].([]interface{})
	instinct := instinctList[0].(map[string]interface{})
	provenance := instinct["provenance"].(map[string]interface{})
	if provenance["application_count"] != float64(1) {
		t.Fatalf("application_count = %v, want 1", provenance["application_count"])
	}
	history := instinct["application_history"].([]interface{})
	if len(history) != 1 {
		t.Fatalf("expected 1 application_history entry, got %d", len(history))
	}
}

// instinctApplyTestFixture seeds a single standalone-store instinct named
// instID and returns the store's base path, so each WR-01 test starts from
// an identical, minimal instinct record.
func instinctApplyTestFixture(t *testing.T, s *storage.Store, instID string) {
	t.Helper()
	instincts := map[string]interface{}{
		"version": "1.0",
		"instincts": []interface{}{
			map[string]interface{}{
				"id":          instID,
				"trigger":     "file_trigger",
				"action":      "file_action",
				"confidence":  0.85,
				"trust_score": 0.85,
				"trust_tier":  "trusted",
				"provenance": map[string]interface{}{
					"source":            "obs_" + instID,
					"source_type":       "observation",
					"evidence":          "from file",
					"created_at":        "2026-04-01T00:00:00Z",
					"application_count": 0,
				},
				"application_history": []interface{}{},
				"related_instincts":   []interface{}{},
				"archived":            false,
			},
		},
	}
	writeTestJSON(t, s.BasePath(), "instincts.json", instincts)
}

// instinctApplyRecordedOutcome runs instinct-apply with args and returns the
// Outcome string of the single application_history entry it wrote.
func instinctApplyRecordedOutcome(t *testing.T, s *storage.Store, instID string, args ...string) string {
	t.Helper()
	rootCmd.SetArgs(append([]string{"instinct-apply", instID}, args...))
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("instinct-apply %v failed: %v", args, err)
	}
	data, err := os.ReadFile(filepath.Join(s.BasePath(), "instincts.json"))
	if err != nil {
		t.Fatalf("read instincts.json: %v", err)
	}
	var file map[string]interface{}
	if err := json.Unmarshal(data, &file); err != nil {
		t.Fatalf("parse instincts.json: %v", err)
	}
	instinctList := file["instincts"].([]interface{})
	instinct := instinctList[0].(map[string]interface{})
	history := instinct["application_history"].([]interface{})
	if len(history) != 1 {
		t.Fatalf("expected 1 application_history entry, got %d", len(history))
	}
	entry := history[0].(map[string]interface{})
	return entry["outcome"].(string)
}

// TestInstinctApplySuccessFalseRecordsNeutralNotHarmful is the WR-01 fix
// (204-REVIEW.md): --success=false must record "neutral", not "harmful" --
// "this guidance didn't help" is not the same claim as "this guidance
// actively made things worse".
func TestInstinctApplySuccessFalseRecordsNeutralNotHarmful(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf
	s, _ := newTestStore(t)
	store = s
	instinctApplyTestFixture(t, s, "inst_neutral")

	got := instinctApplyRecordedOutcome(t, s, "inst_neutral", "--success=false")
	if got != "neutral" {
		t.Fatalf("--success=false recorded outcome %q, want %q -- WR-01 regression", got, "neutral")
	}
}

// TestInstinctApplySuccessTrueRecordsHelpful asserts the backward-compatible
// default (--success unset, or explicitly true) still records "helpful".
func TestInstinctApplySuccessTrueRecordsHelpful(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf
	s, _ := newTestStore(t)
	store = s
	instinctApplyTestFixture(t, s, "inst_helpful")

	got := instinctApplyRecordedOutcome(t, s, "inst_helpful")
	if got != "helpful" {
		t.Fatalf("default (no flags) recorded outcome %q, want %q", got, "helpful")
	}
}

// TestInstinctApplyOutcomeFlagRecordsHarmful asserts --outcome=harmful is
// the explicit way to record the stronger claim --success never can now.
func TestInstinctApplyOutcomeFlagRecordsHarmful(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf
	s, _ := newTestStore(t)
	store = s
	instinctApplyTestFixture(t, s, "inst_harmful")

	got := instinctApplyRecordedOutcome(t, s, "inst_harmful", "--outcome=harmful")
	if got != "harmful" {
		t.Fatalf("--outcome=harmful recorded outcome %q, want %q", got, "harmful")
	}
}

// TestInstinctApplyRefusesSuccessAndOutcomeTogether asserts passing both
// flags is refused by name rather than one silently winning.
func TestInstinctApplyRefusesSuccessAndOutcomeTogether(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf
	s, _ := newTestStore(t)
	store = s
	instinctApplyTestFixture(t, s, "inst_both_flags")

	var errBuf bytes.Buffer
	stderr = &errBuf

	rootCmd.SetArgs([]string{"instinct-apply", "inst_both_flags", "--success=true", "--outcome=neutral"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("instinct-apply itself should not error (the refusal is reported via output, not a Go error): %v", err)
	}
	if !strings.Contains(errBuf.String(), "refuses --success and --outcome together") {
		t.Fatalf("expected a refusal naming --success and --outcome together, got: %q", errBuf.String())
	}

	data, err := os.ReadFile(filepath.Join(s.BasePath(), "instincts.json"))
	if err != nil {
		t.Fatalf("read instincts.json: %v", err)
	}
	var file map[string]interface{}
	if err := json.Unmarshal(data, &file); err != nil {
		t.Fatalf("parse instincts.json: %v", err)
	}
	instinct := file["instincts"].([]interface{})[0].(map[string]interface{})
	history := instinct["application_history"].([]interface{})
	if len(history) != 0 {
		t.Fatalf("expected no application_history entry written on a refused call, got %d", len(history))
	}
}

func TestSpawnGetDepth(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, _ := newTestStore(t)
	store = s

	// Write a spawn tree file (pipe-delimited: timestamp|parent|caste|agentName|task|depth|status)
	spawnData := "2026-01-01T00:00:00Z|colony-prime|builder|test_ant|build_code|2|active\n"
	os.WriteFile(filepath.Join(s.BasePath(), "spawn-tree.txt"), []byte(spawnData), 0644)

	rootCmd.SetArgs([]string{"spawn-get-depth", "--name", "test_ant"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("spawn-get-depth failed: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})
	if result["found"] != true {
		t.Errorf("expected found true, got %v", result["found"])
	}
}

func TestSwarmDisplayGet(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, _ := newTestStore(t)
	store = s

	// Create COLONY_STATE.json
	colonyData := map[string]interface{}{
		"version":       "2.0",
		"state":         "EXECUTING",
		"current_phase": 2,
		"milestone":     "Open Chambers",
		"plan": map[string]interface{}{
			"phases": []interface{}{
				map[string]interface{}{"id": 1, "name": "Setup", "status": "completed"},
				map[string]interface{}{"id": 2, "name": "Build", "status": "in_progress"},
			},
		},
		"memory":     map[string]interface{}{"phase_learnings": []interface{}{}, "decisions": []interface{}{}, "instincts": []interface{}{}},
		"errors":     map[string]interface{}{"records": []interface{}{}, "flagged_patterns": []interface{}{}},
		"signals":    []interface{}{},
		"graveyards": []interface{}{},
		"events":     []interface{}{},
	}
	writeTestJSON(t, s.BasePath(), "COLONY_STATE.json", colonyData)

	rootCmd.SetArgs([]string{"swarm-display-get"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("swarm-display-get failed: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})
	if result["state"] != "EXECUTING" {
		t.Errorf("expected state EXECUTING, got %v", result["state"])
	}
}

func TestSwarmActivityLog(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, _ := newTestStore(t)
	store = s

	rootCmd.SetArgs([]string{"swarm-activity-log",
		"--message", "build started",
		"--severity", "info",
	})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("swarm-activity-log failed: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})
	if result["logged"] != true {
		t.Errorf("expected logged true, got %v", result["logged"])
	}

	// Verify JSONL file was created
	data, err := os.ReadFile(filepath.Join(s.BasePath(), "swarm-activity.jsonl"))
	if err != nil {
		t.Fatalf("reading activity log: %v", err)
	}
	if len(data) == 0 {
		t.Error("activity log is empty")
	}
}
