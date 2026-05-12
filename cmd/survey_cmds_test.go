package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

// --- Survey Load Tests ---

func TestSurveyLoadNoSurveyDir(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	rootCmd.SetArgs([]string{"survey-load"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	if env["ok"] != true {
		t.Fatalf("expected ok:true, got: %v", env["ok"])
	}

	result := env["result"].(map[string]interface{})
	if result["loaded"] != false {
		t.Errorf("loaded = %v, want false when no survey dir", result["loaded"])
	}
	if result["data"] != nil {
		t.Errorf("data = %v, want null when no survey dir", result["data"])
	}
}

func TestSurveyLoadWithFiles(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	// Create survey directory and some files
	surveyDir := s.BasePath() + "/survey"
	os.MkdirAll(surveyDir, 0755)
	os.WriteFile(surveyDir+"/blueprint.json", []byte(`{"name":"test-blueprint"}`+"\n"), 0644)
	os.WriteFile(surveyDir+"/chambers.json", []byte(`{"chambers":[]}`+"\n"), 0644)

	rootCmd.SetArgs([]string{"survey-load"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	if env["ok"] != true {
		t.Fatalf("expected ok:true, got: %v", env["ok"])
	}

	result := env["result"].(map[string]interface{})
	if result["loaded"] != true {
		t.Errorf("loaded = %v, want true", result["loaded"])
	}

	files := result["files"].(map[string]interface{})
	if files["blueprint"] != true {
		t.Errorf("files.blueprint = %v, want true", files["blueprint"])
	}
	if files["chambers"] != true {
		t.Errorf("files.chambers = %v, want true", files["chambers"])
	}
	if files["disciplines"] != false {
		t.Errorf("files.disciplines = %v, want false (not present)", files["disciplines"])
	}
}

func TestSurveyLoadNilStore(t *testing.T) {
	var buf bytes.Buffer
	saveGlobals(t)
	stderr = &buf

	store = nil

	// Call RunE directly to bypass PersistentPreRunE which would init store
	err := surveyLoadCmd.RunE(surveyLoadCmd, []string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	if env["ok"] != false {
		t.Errorf("expected ok:false for nil store, got: %v", env["ok"])
	}
}

// --- Survey Verify Tests ---

func TestSurveyVerifyNoSurveyDir(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	rootCmd.SetArgs([]string{"survey-verify"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	if env["ok"] != true {
		t.Fatalf("expected ok:true, got: %v", env["ok"])
	}

	result := env["result"].(map[string]interface{})
	if result["valid"] != false {
		t.Errorf("valid = %v, want false when no files exist", result["valid"])
	}

	files := result["files"].([]interface{})
	if len(files) != 12 {
		t.Errorf("expected 12 file checks, got %d", len(files))
	}
}

func TestSurveyVerifyWithValidFiles(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	surveyDir := s.BasePath() + "/survey"
	os.MkdirAll(surveyDir, 0755)
	os.WriteFile(surveyDir+"/blueprint.json", []byte(`{"name":"bp"}`+"\n"), 0644)
	os.WriteFile(surveyDir+"/chambers.json", []byte(`{"chambers":[]}`+"\n"), 0644)

	rootCmd.SetArgs([]string{"survey-verify"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})
	if result["valid"] != false {
		t.Errorf("valid = %v, want false (not all 5 files present)", result["valid"])
	}
}

func TestSurveyVerifyRequiresMarkdownSurveyDocs(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	surveyDir := s.BasePath() + "/survey"
	os.MkdirAll(surveyDir, 0755)
	for _, name := range surveyFiles {
		os.WriteFile(surveyDir+"/"+name+".json", []byte(`{"ok":true}`+"\n"), 0644)
	}

	rootCmd.SetArgs([]string{"survey-verify"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})
	if result["valid"] != false {
		t.Fatalf("valid = %v, want false when markdown survey docs are missing", result["valid"])
	}
	if !issuesContain(result["issues"].([]interface{}), "PROVISIONS.md: file not found") {
		t.Fatalf("issues did not report missing PROVISIONS.md: %v", result["issues"])
	}
}

func TestSurveyVerifyWithCompleteArtifacts(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	surveyDir := s.BasePath() + "/survey"
	os.MkdirAll(surveyDir, 0755)
	for _, name := range surveyFiles {
		os.WriteFile(surveyDir+"/"+name+".json", []byte(`{"ok":true}`+"\n"), 0644)
	}
	for _, name := range []string{"PROVISIONS.md", "TRAILS.md", "BLUEPRINT.md", "CHAMBERS.md", "DISCIPLINES.md", "SENTINEL-PROTOCOLS.md", "PATHOGENS.md"} {
		os.WriteFile(surveyDir+"/"+name, []byte("# "+name+"\n"), 0644)
	}

	rootCmd.SetArgs([]string{"survey-verify"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})
	if result["valid"] != true {
		t.Fatalf("valid = %v, want true with all survey artifacts present: %v", result["valid"], result["issues"])
	}
}

func TestSurveyVerifyRejectsMalformedMarkdownSurveyDocs(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	surveyDir := s.BasePath() + "/survey"
	os.MkdirAll(surveyDir, 0755)
	for _, name := range surveyFiles {
		os.WriteFile(surveyDir+"/"+name+".json", []byte(`{"ok":true}`+"\n"), 0644)
	}
	for _, name := range requiredSurveyMarkdownFiles {
		content := []byte("# " + name + "\n")
		if name == "PROVISIONS.md" {
			content = []byte{'#', ' ', 'b', 'a', 'd', 0, '\n'}
		}
		os.WriteFile(surveyDir+"/"+name, content, 0644)
	}

	rootCmd.SetArgs([]string{"survey-verify"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})
	if result["valid"] != false {
		t.Fatalf("valid = %v, want false with malformed markdown", result["valid"])
	}
	if !issuesContain(result["issues"].([]interface{}), "PROVISIONS.md: invalid markdown") {
		t.Fatalf("issues did not report malformed PROVISIONS.md: %v", result["issues"])
	}
}

func TestSurveyVerifyInvalidJSON(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	surveyDir := s.BasePath() + "/survey"
	os.MkdirAll(surveyDir, 0755)
	os.WriteFile(surveyDir+"/blueprint.json", []byte(`not json`), 0644)

	rootCmd.SetArgs([]string{"survey-verify"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})
	files := result["files"].([]interface{})

	// Find blueprint entry
	foundBlueprint := false
	for _, f := range files {
		fm := f.(map[string]interface{})
		if fm["name"] == "blueprint" {
			foundBlueprint = true
			if fm["exists"] != true {
				t.Errorf("blueprint exists = %v, want true", fm["exists"])
			}
			if fm["valid_json"] != false {
				t.Errorf("blueprint valid_json = %v, want false for invalid JSON", fm["valid_json"])
			}
		}
	}
	if !foundBlueprint {
		t.Error("did not find blueprint in file checks")
	}
}

func issuesContain(issues []interface{}, want string) bool {
	for _, issue := range issues {
		if strings.Contains(issue.(string), want) {
			return true
		}
	}
	return false
}

// --- Verify Claims Tests ---

func TestVerifyClaimsValid(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	state := map[string]interface{}{
		"version": "3.0",
		"state":   "READY",
		"plan": map[string]interface{}{
			"phases": []interface{}{
				map[string]interface{}{"title": "phase 1"},
				map[string]interface{}{"title": "phase 2"},
			},
		},
		"current_phase":        float64(1),
		"milestone":            "Brood Stable",
		"milestone_updated_at": "2026-01-01T00:00:00Z",
	}
	s.SaveJSON("COLONY_STATE.json", state)

	rootCmd.SetArgs([]string{"verify-claims"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	if env["ok"] != true {
		t.Fatalf("expected ok:true, got: %v", env["ok"])
	}

	result := env["result"].(map[string]interface{})
	if result["valid"] != true {
		t.Errorf("valid = %v, want true for valid state", result["valid"])
	}
}

func TestVerifyClaimsMissingVersion(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	state := map[string]interface{}{
		"state": "READY",
	}
	s.SaveJSON("COLONY_STATE.json", state)

	rootCmd.SetArgs([]string{"verify-claims"})

	rootCmd.Execute()

	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})
	if result["valid"] != false {
		t.Errorf("valid = %v, want false when version missing", result["valid"])
	}

	issues := result["issues"].([]interface{})
	if len(issues) == 0 {
		t.Error("expected issues for missing version")
	}
}

func TestVerifyClaimsExecutingNoTimestamp(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	state := map[string]interface{}{
		"version": "3.0",
		"state":   "EXECUTING",
	}
	s.SaveJSON("COLONY_STATE.json", state)

	rootCmd.SetArgs([]string{"verify-claims"})

	rootCmd.Execute()

	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})
	if result["valid"] != false {
		t.Errorf("valid = %v, want false when EXECUTING without build_started_at", result["valid"])
	}
}

func TestVerifyClaimsPhaseOutOfRange(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	state := map[string]interface{}{
		"version":       "3.0",
		"state":         "READY",
		"current_phase": float64(5),
		"plan": map[string]interface{}{
			"phases": []interface{}{
				map[string]interface{}{"title": "phase 1"},
			},
		},
	}
	s.SaveJSON("COLONY_STATE.json", state)

	rootCmd.SetArgs([]string{"verify-claims"})

	rootCmd.Execute()

	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})
	if result["valid"] != false {
		t.Errorf("valid = %v, want false when phase out of range", result["valid"])
	}
}

// --- Autofix Checkpoint Tests ---

func TestAutofixCheckpoint(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	goal := "test goal"
	state := map[string]interface{}{
		"version": "3.0",
		"goal":    goal,
		"state":   "READY",
	}
	s.SaveJSON("COLONY_STATE.json", state)

	rootCmd.SetArgs([]string{"autofix-checkpoint", "--issue", "state corruption detected"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	if env["ok"] != true {
		t.Fatalf("expected ok:true, got: %v", env["ok"])
	}

	result := env["result"].(map[string]interface{})
	if result["checkpoint"] == nil || result["checkpoint"].(string) == "" {
		t.Error("checkpoint should be non-empty")
	}
	if result["issue"] != "state corruption detected" {
		t.Errorf("issue = %v, want 'state corruption detected'", result["issue"])
	}

	// Verify checkpoint file was created
	path := result["path"].(string)
	data, err := os.ReadFile(s.BasePath() + "/" + path)
	if err != nil {
		t.Fatalf("checkpoint file not created: %v", err)
	}
	if !json.Valid(data) {
		t.Error("checkpoint file should be valid JSON")
	}
}

func TestAutofixCheckpointMissingIssue(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stderr = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	rootCmd.SetArgs([]string{"autofix-checkpoint"})

	rootCmd.Execute()

	env := parseEnvelope(t, buf.String())
	if env["ok"] != false {
		t.Errorf("expected ok:false for missing --issue, got: %v", env["ok"])
	}
}

func TestAutofixRollback(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	// Create checkpoint
	checkpointData := `{"version":"2.0","goal":"old goal","state":"READY"}` + "\n"
	s.AtomicWrite("checkpoints/autofix-test-rollback.json", []byte(checkpointData))

	// Set current state to something different
	currentState := map[string]interface{}{
		"version": "3.0",
		"goal":    "new goal",
		"state":   "BUILT",
	}
	s.SaveJSON("COLONY_STATE.json", currentState)

	rootCmd.SetArgs([]string{"autofix-rollback", "--checkpoint-id", "autofix-test-rollback"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	if env["ok"] != true {
		t.Fatalf("expected ok:true, got: %v", env["ok"])
	}

	result := env["result"].(map[string]interface{})
	if result["rolled_back"] != true {
		t.Errorf("rolled_back = %v, want true", result["rolled_back"])
	}

	// Verify COLONY_STATE.json was overwritten
	var restored map[string]interface{}
	s.LoadJSON("COLONY_STATE.json", &restored)
	if restored["version"] != "2.0" {
		t.Errorf("version after rollback = %v, want 2.0", restored["version"])
	}
}

func TestAutofixRollbackNotFound(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stderr = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	rootCmd.SetArgs([]string{"autofix-rollback", "--checkpoint-id", "nonexistent"})

	rootCmd.Execute()

	env := parseEnvelope(t, buf.String())
	if env["ok"] != false {
		t.Errorf("expected ok:false for nonexistent checkpoint, got: %v", env["ok"])
	}
}
