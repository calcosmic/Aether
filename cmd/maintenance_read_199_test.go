package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestMaintenanceInspectionReadOnly(t *testing.T) {
	cases := []struct {
		name    string
		prepare func(t *testing.T, workspace, hub string) []string
	}{
		{
			name: "source pass",
			prepare: func(t *testing.T, workspace, _ string) []string {
				populateSourceCheckRoot(t, workspace)
				return []string{"source-check", "--root", workspace, "--json"}
			},
		},
		{
			name: "source drift",
			prepare: func(t *testing.T, workspace, _ string) []string {
				populateSourceCheckRoot(t, workspace)
				if err := os.Remove(filepath.Join(workspace, ".claude", "commands", "ant", "status.md")); err != nil {
					t.Fatal(err)
				}
				return []string{"source-check", "--root", workspace, "--json"}
			},
		},
		{
			name: "source missing",
			prepare: func(_ *testing.T, workspace, _ string) []string {
				return []string{"source-check", "--root", filepath.Join(workspace, "missing-source"), "--json"}
			},
		},
		{
			name: "integrity valid hub",
			prepare: func(t *testing.T, _ string, hub string) []string {
				createHubWithExpectedCounts(t, hub)
				if err := os.WriteFile(filepath.Join(hub, "version.json"), []byte(`{"version":"1.0.66"}`), 0o644); err != nil {
					t.Fatal(err)
				}
				return []string{"integrity", "--json"}
			},
		},
		{
			name: "integrity missing hub",
			prepare: func(_ *testing.T, _, _ string) []string {
				return []string{"integrity", "--json"}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			saveGlobals(t)
			resetRootCmd(t)
			workspace := t.TempDir()
			hub := filepath.Join(t.TempDir(), "hub")
			if err := os.MkdirAll(filepath.Join(workspace, ".aether", "data"), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.MkdirAll(hub, 0o755); err != nil {
				t.Fatal(err)
			}
			t.Setenv("AETHER_ROOT", workspace)
			t.Setenv("COLONY_DATA_DIR", filepath.Join(workspace, ".aether", "data"))
			t.Setenv("AETHER_HUB_DIR", hub)
			args := tc.prepare(t, workspace, hub)

			before := fingerprintMaintenanceTrees(t, workspace, hub)
			_, _ = executeMaintenanceInspectionJSON(t, args...)
			after := fingerprintMaintenanceTrees(t, workspace, hub)
			if !reflect.DeepEqual(before, after) {
				t.Fatalf("%s mutated workspace or hub\nbefore: %#v\nafter:  %#v", tc.name, before, after)
			}
		})
	}
}

func TestMaintenanceInspectionStructuredResult(t *testing.T) {
	sourceRoot := minimalSourceCheckRoot(t)
	source := structToJSONMap(t, runSourceCheck(sourceRoot))
	assertMaintenanceInspectionShape(t, source, "source.parity.inspect")

	saveGlobals(t)
	resetRootCmd(t)
	workspace := t.TempDir()
	hub := filepath.Join(t.TempDir(), "hub")
	if err := os.MkdirAll(filepath.Join(workspace, ".aether", "data"), 0o755); err != nil {
		t.Fatal(err)
	}
	createHubWithExpectedCounts(t, hub)
	if err := os.WriteFile(filepath.Join(hub, "version.json"), []byte(`{"version":"1.0.66"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AETHER_ROOT", workspace)
	t.Setenv("COLONY_DATA_DIR", filepath.Join(workspace, ".aether", "data"))
	t.Setenv("AETHER_HUB_DIR", hub)
	integrity, _ := executeMaintenanceInspectionJSON(t, "integrity", "--json")
	assertMaintenanceInspectionShape(t, integrity, "integrity.inspect")

	catalog := buildMaintenanceCatalog("runtime")
	wantSchemas := map[string]string{
		"integrity.inspect":     stringField(t, integrity, "schema_version"),
		"source.parity.inspect": stringField(t, source, "schema_version"),
	}
	for _, operation := range catalog.Inspection {
		if want, ok := wantSchemas[operation.OperationID]; ok && operation.ReceiptType != want {
			t.Errorf("catalog contract for %s = %q, result schema = %q", operation.OperationID, operation.ReceiptType, want)
		}
	}
}

func TestMaintenanceInspectionDriftEvidence(t *testing.T) {
	root := minimalSourceCheckRoot(t)
	generatedPath := ".claude/commands/ant/status.md"
	sourcePath := ".aether/commands/status.yaml"
	if err := os.Remove(filepath.Join(root, filepath.FromSlash(generatedPath))); err != nil {
		t.Fatal(err)
	}

	result := structToJSONMap(t, runSourceCheck(root))
	findings, ok := result["findings"].([]any)
	if !ok || len(findings) == 0 {
		t.Fatalf("drift result findings = %T %#v, want non-empty array", result["findings"], result["findings"])
	}
	for _, raw := range findings {
		finding, ok := raw.(map[string]any)
		if !ok || finding["source_path"] != sourcePath || finding["generated_path"] != generatedPath {
			continue
		}
		if command, _ := finding["recovery_command"].(string); strings.TrimSpace(command) == "" {
			t.Fatalf("drift finding omitted recovery command: %#v", finding)
		}
		paths, ok := finding["evidence_paths"].([]any)
		if !ok || !containsJSONText(paths, sourcePath) || !containsJSONText(paths, generatedPath) {
			t.Fatalf("drift evidence paths = %#v, want exact source and generated paths", finding["evidence_paths"])
		}
		return
	}
	t.Fatalf("findings did not pair exact source %q with generated path %q: %#v", sourcePath, generatedPath, findings)
}

func populateSourceCheckRoot(t *testing.T, root string) {
	t.Helper()
	fixture := minimalSourceCheckRoot(t)
	copyDirForTest(t, fixture, root)
}

func executeMaintenanceInspectionJSON(t *testing.T, args ...string) (map[string]any, error) {
	t.Helper()
	resetFlags(rootCmd)
	var out bytes.Buffer
	stdout = &out
	rootCmd.SetArgs(args)
	err := rootCmd.Execute()
	if out.Len() == 0 {
		return nil, err
	}
	var result map[string]any
	if decodeErr := json.Unmarshal(out.Bytes(), &result); decodeErr != nil {
		t.Fatalf("decode %v output %q: %v", args, out.String(), decodeErr)
	}
	return result, err
}

func structToJSONMap(t *testing.T, value any) map[string]any {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatal(err)
	}
	return result
}

func assertMaintenanceInspectionShape(t *testing.T, result map[string]any, operation string) {
	t.Helper()
	if result == nil {
		t.Fatal("inspection emitted no structured result")
	}
	if got := stringField(t, result, "operation_id"); got != operation {
		t.Fatalf("operation_id = %q, want %q", got, operation)
	}
	if got := stringField(t, result, "state_effect"); got != "none" {
		t.Fatalf("state_effect = %q, want none", got)
	}
	if strings.TrimSpace(stringField(t, result, "schema_version")) == "" {
		t.Fatal("schema_version is empty")
	}
	if strings.TrimSpace(stringField(t, result, "next_action")) == "" {
		t.Fatal("next_action is empty")
	}
	for _, field := range []string{"findings", "evidence", "blockers"} {
		if _, ok := result[field].([]any); !ok {
			t.Errorf("%s = %T, want stable array", field, result[field])
		}
	}
	verification, ok := result["verification"].(map[string]any)
	if !ok || strings.TrimSpace(stringField(t, verification, "status")) == "" {
		t.Fatalf("verification = %#v, want object with status", result["verification"])
	}
}

func containsJSONText(values []any, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
