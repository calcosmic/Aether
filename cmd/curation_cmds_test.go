package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func bindSeededCommandTestRepository(t *testing.T) commandTestRepository {
	t.Helper()
	binding := bindCommandTestRepository(t)

	entries, err := os.ReadDir("testdata")
	if err != nil {
		t.Fatalf("read command test fixtures: %v", err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		data, err := os.ReadFile(filepath.Join("testdata", entry.Name()))
		if err != nil {
			t.Fatalf("read command test fixture %s: %v", entry.Name(), err)
		}
		if err := os.WriteFile(filepath.Join(binding.DataDir, entry.Name()), data, 0644); err != nil {
			t.Fatalf("seed command test fixture %s: %v", entry.Name(), err)
		}
	}

	return binding
}

func TestCurationSentinelDryRun(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	var buf bytes.Buffer
	stdout = &buf

	bindSeededCommandTestRepository(t)

	rootCmd.SetArgs([]string{"curation-sentinel", "--dry-run"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("curation-sentinel returned error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, `"ok":true`) {
		t.Errorf("expected ok:true, got: %s", output)
	}
	if !strings.Contains(output, `"name":"sentinel"`) {
		t.Errorf("expected name:sentinel, got: %s", output)
	}
	if !strings.Contains(output, `"dry_run":true`) {
		t.Errorf("expected dry_run:true, got: %s", output)
	}
}

func TestCurationNurseDryRun(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	var buf bytes.Buffer
	stdout = &buf

	bindSeededCommandTestRepository(t)

	rootCmd.SetArgs([]string{"curation-nurse", "--dry-run"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("curation-nurse returned error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, `"ok":true`) {
		t.Errorf("expected ok:true, got: %s", output)
	}
	if !strings.Contains(output, `"name":"nurse"`) {
		t.Errorf("expected name:nurse, got: %s", output)
	}
}

func TestCurationCriticDryRun(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	var buf bytes.Buffer
	stdout = &buf

	bindSeededCommandTestRepository(t)

	rootCmd.SetArgs([]string{"curation-critic", "--dry-run"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("curation-critic returned error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, `"ok":true`) {
		t.Errorf("expected ok:true, got: %s", output)
	}
	if !strings.Contains(output, `"name":"critic"`) {
		t.Errorf("expected name:critic, got: %s", output)
	}
}

func TestCurationHeraldDryRun(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	var buf bytes.Buffer
	stdout = &buf

	bindSeededCommandTestRepository(t)

	rootCmd.SetArgs([]string{"curation-herald", "--dry-run"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("curation-herald returned error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, `"ok":true`) {
		t.Errorf("expected ok:true, got: %s", output)
	}
	if !strings.Contains(output, `"name":"herald"`) {
		t.Errorf("expected name:herald, got: %s", output)
	}
}

func TestCurationJanitorDryRun(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	var buf bytes.Buffer
	stdout = &buf

	bindSeededCommandTestRepository(t)

	rootCmd.SetArgs([]string{"curation-janitor", "--dry-run"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("curation-janitor returned error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, `"ok":true`) {
		t.Errorf("expected ok:true, got: %s", output)
	}
	if !strings.Contains(output, `"name":"janitor"`) {
		t.Errorf("expected name:janitor, got: %s", output)
	}
}

func TestCurationArchivistDryRun(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	var buf bytes.Buffer
	stdout = &buf

	bindSeededCommandTestRepository(t)

	rootCmd.SetArgs([]string{"curation-archivist", "--dry-run"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("curation-archivist returned error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, `"ok":true`) {
		t.Errorf("expected ok:true, got: %s", output)
	}
	if !strings.Contains(output, `"name":"archivist"`) {
		t.Errorf("expected name:archivist, got: %s", output)
	}
}

func TestCurationLibrarianDryRun(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	var buf bytes.Buffer
	stdout = &buf

	bindSeededCommandTestRepository(t)

	rootCmd.SetArgs([]string{"curation-librarian", "--dry-run"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("curation-librarian returned error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, `"ok":true`) {
		t.Errorf("expected ok:true, got: %s", output)
	}
	if !strings.Contains(output, `"name":"librarian"`) {
		t.Errorf("expected name:librarian, got: %s", output)
	}
}

func TestCurationScribeDryRun(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	var buf bytes.Buffer
	stdout = &buf

	bindSeededCommandTestRepository(t)

	rootCmd.SetArgs([]string{"curation-scribe", "--dry-run"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("curation-scribe returned error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, `"ok":true`) {
		t.Errorf("expected ok:true, got: %s", output)
	}
	if !strings.Contains(output, `"name":"scribe"`) {
		t.Errorf("expected name:scribe, got: %s", output)
	}
}

func TestCurationRunDryRun(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	var buf bytes.Buffer
	stdout = &buf

	bindSeededCommandTestRepository(t)

	rootCmd.SetArgs([]string{"curation-run", "--dry-run"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("curation-run returned error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, `"ok":true`) {
		t.Errorf("expected ok:true, got: %s", output)
	}

	// Parse the JSON to verify structure
	var envelope struct {
		OK     bool `json:"ok"`
		Result struct {
			Steps []struct {
				Name    string `json:"name"`
				Success bool   `json:"success"`
			} `json:"steps"`
			Succeeded int  `json:"succeeded"`
			Failed    int  `json:"failed"`
			Skipped   int  `json:"skipped"`
			DryRun    bool `json:"dry_run"`
		} `json:"result"`
	}
	if err := json.Unmarshal([]byte(output), &envelope); err != nil {
		t.Fatalf("failed to parse JSON: %v\noutput: %s", err, output)
	}

	if !envelope.OK {
		t.Error("expected ok:true")
	}
	if envelope.Result.DryRun != true {
		t.Error("expected dry_run:true")
	}
	if len(envelope.Result.Steps) != 8 {
		t.Errorf("expected 8 steps, got %d", len(envelope.Result.Steps))
	}
	if envelope.Result.Succeeded != 8 {
		t.Errorf("expected 8 succeeded, got %d", envelope.Result.Succeeded)
	}
	if envelope.Result.Failed != 0 {
		t.Errorf("expected 0 failed, got %d", envelope.Result.Failed)
	}
	if envelope.Result.Skipped != 0 {
		t.Errorf("expected 0 skipped, got %d", envelope.Result.Skipped)
	}

	// Verify step order matches shell orchestrator.sh
	expectedOrder := []string{"sentinel", "nurse", "critic", "herald", "janitor", "archivist", "librarian", "scribe"}
	for i, expected := range expectedOrder {
		if envelope.Result.Steps[i].Name != expected {
			t.Errorf("step %d: expected name %q, got %q", i, expected, envelope.Result.Steps[i].Name)
		}
		if !envelope.Result.Steps[i].Success {
			t.Errorf("step %d (%s): expected success=true", i, expected)
		}
	}
}

func TestCurationRunNoStore(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	var errBuf bytes.Buffer
	stderr = &errBuf

	bindCommandTestRepository(t)
	store = nil

	rootCmd.SetArgs([]string{"curation-run", "--dry-run"})

	// This should not fail because PersistentPreRunE initializes store
	// from AETHER_ROOT. Instead we verify the command handles nil store
	// if it somehow occurs.
	_ = rootCmd.Execute()
}

func TestCurationCommandsRegistered(t *testing.T) {
	commands := []string{
		"curation-sentinel",
		"curation-nurse",
		"curation-critic",
		"curation-herald",
		"curation-janitor",
		"curation-archivist",
		"curation-librarian",
		"curation-scribe",
		"curation-run",
	}

	for _, name := range commands {
		found := false
		for _, cmd := range rootCmd.Commands() {
			if cmd.Use == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("command %q not registered in rootCmd", name)
		}
	}
}

func TestCurationDryRunFalse(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	var buf bytes.Buffer
	stdout = &buf

	bindSeededCommandTestRepository(t)

	rootCmd.SetArgs([]string{"curation-sentinel"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("curation-sentinel (no --dry-run) returned error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, `"ok":true`) {
		t.Errorf("expected ok:true, got: %s", output)
	}
	if strings.Contains(output, `"dry_run":true`) {
		t.Errorf("expected dry_run:false (default), got: %s", output)
	}
}

func TestCurationArchivistThresholdArchivesTypedInstincts(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	binding := bindCommandTestRepository(t)
	s := binding.Store

	writeTestJSON(t, s.BasePath(), "instincts.json", map[string]interface{}{
		"version": "1.0",
		"instincts": []interface{}{
			map[string]interface{}{
				"id":          "inst_low",
				"trigger":     "low trust",
				"action":      "archive me",
				"confidence":  0.4,
				"trust_score": 0.25,
				"provenance": map[string]interface{}{
					"created_at": "2026-04-01T00:00:00Z",
				},
				"application_history": []interface{}{},
				"related_instincts":   []interface{}{},
				"archived":            false,
			},
		},
	})

	rootCmd.SetArgs([]string{"curation-archivist", "--threshold", "0.30"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("curation-archivist failed: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(s.BasePath(), "instincts.json"))
	if err != nil {
		t.Fatalf("read instincts.json: %v", err)
	}
	var decoded map[string]interface{}
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("parse instincts.json: %v", err)
	}
	instincts := decoded["instincts"].([]interface{})
	instinct := instincts[0].(map[string]interface{})
	if instinct["archived"] != true {
		t.Fatalf("expected instinct to be archived, got %v", instinct["archived"])
	}
	if !strings.Contains(buf.String(), `"threshold":0.3`) {
		t.Fatalf("expected threshold in output, got %s", buf.String())
	}
}
