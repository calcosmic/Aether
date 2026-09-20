package cmd

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
)

var maintenanceLandingOperationIDs = []string{
	"integrity.inspect",
	"source.parity.inspect",
	"registry.inspect",
	"chamber.inspect",
	"context.inspect",
	"archive.inspect",
	"skills.inspect",
	"skills.diff",
	"update.apply",
	"state.migration.apply",
	"data.cleanup.apply",
	"backup.prune.apply",
	"temp.cleanup.apply",
	"registry.update",
	"chamber.create",
}

func TestMaintenanceLandingReadOnly(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	s, root := newTestStoreWithRoot(t)
	store = s
	hub := filepath.Join(root, "hub")
	if err := os.MkdirAll(filepath.Join(hub, "system"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AETHER_HUB_DIR", hub)
	t.Setenv("AETHER_OUTPUT_MODE", "visual")

	before := fingerprintMaintenanceTrees(t, root, hub)
	var out bytes.Buffer
	stdout = &out
	rootCmd.SetArgs([]string{"maintenance"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("aether maintenance: %v", err)
	}
	after := fingerprintMaintenanceTrees(t, root, hub)
	if !reflect.DeepEqual(before, after) {
		t.Fatalf("maintenance landing mutated repository or hub\nbefore: %#v\nafter:  %#v", before, after)
	}
	if !strings.Contains(out.String(), "State effect: none") {
		t.Fatalf("visual landing did not disclose its zero-write result:\n%s", out.String())
	}
}

func TestMaintenanceLandingCatalog(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	s, _ := newTestStoreWithRoot(t)
	store = s
	t.Setenv("AETHER_OUTPUT_MODE", "json")
	t.Setenv("AETHER_PLATFORM", "claude")

	command, _, err := rootCmd.Find([]string{"maintenance"})
	if err != nil {
		t.Fatalf("maintenance command is not registered: %v", err)
	}
	if got, want := command.Short, "Inspect or repair Aether internals with preview and rollback."; got != want {
		t.Fatalf("maintenance short help = %q, want %q", got, want)
	}

	result := runMaintenanceLandingForTest(t)
	if got := stringField(t, result, "schema_version"); got != "maintenance-catalog/v1" {
		t.Fatalf("schema_version = %q, want maintenance-catalog/v1", got)
	}
	if got := stringField(t, result, "state_effect"); got != "none" {
		t.Fatalf("state_effect = %q, want none", got)
	}
	lifecycle := mapField(t, result, "lifecycle")
	if got := stringField(t, lifecycle, "projection_revision"); got != LifecycleProjectionRevision {
		t.Fatalf("maintenance lifecycle projection = %q, want %q", got, LifecycleProjectionRevision)
	}

	catalog := mapField(t, result, "catalog")
	inspection := operationListField(t, catalog, "inspection")
	mutation := operationListField(t, catalog, "mutation")
	if len(inspection) == 0 || len(mutation) == 0 {
		t.Fatalf("catalog must separate inspection from mutation: %#v", catalog)
	}

	seen := map[string]string{}
	for _, operation := range append(append([]map[string]any{}, inspection...), mutation...) {
		id := stringField(t, operation, "operation_id")
		if _, duplicate := seen[id]; duplicate {
			t.Fatalf("duplicate maintenance operation id %q", id)
		}
		seen[id] = stringField(t, operation, "mutation_class")
		for _, field := range []string{"runtime_command", "display_command", "receipt_type", "recovery_action"} {
			if strings.TrimSpace(stringField(t, operation, field)) == "" {
				t.Errorf("operation %s has empty %s", id, field)
			}
		}
		for _, field := range []string{"preview_available", "transaction_required"} {
			if _, ok := operation[field].(bool); !ok {
				t.Errorf("operation %s field %s = %T, want bool", id, field, operation[field])
			}
		}
	}
	for _, operation := range inspection {
		if got := stringField(t, operation, "mutation_class"); got != "read_only" {
			t.Errorf("inspection %s mutation_class = %q, want read_only", stringField(t, operation, "operation_id"), got)
		}
	}
	for _, operation := range mutation {
		if got := stringField(t, operation, "mutation_class"); got != "mutating" {
			t.Errorf("mutation %s mutation_class = %q, want mutating", stringField(t, operation, "operation_id"), got)
		}
	}
	for _, id := range maintenanceLandingOperationIDs {
		if _, ok := seen[id]; !ok {
			t.Errorf("catalog missing stable operation id %q", id)
		}
	}

	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	for _, internal := range []string{"build-finalize", "continue-finalize", "spawn-complete", "colony-prime", "internal-worker-adapter"} {
		if strings.Contains(string(encoded), internal) {
			t.Errorf("maintenance landing promoted internal protocol command %q", internal)
		}
	}
}

func TestMaintenanceLandingOutputModes(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	s, _ := newTestStoreWithRoot(t)
	store = s

	t.Setenv("AETHER_OUTPUT_MODE", "json")
	result := runMaintenanceLandingForTest(t)
	catalog := mapField(t, result, "catalog")
	var operations []map[string]any
	operations = append(operations, operationListField(t, catalog, "inspection")...)
	operations = append(operations, operationListField(t, catalog, "mutation")...)

	resetFlags(rootCmd)
	t.Setenv("AETHER_OUTPUT_MODE", "visual")
	var out bytes.Buffer
	stdout = &out
	rootCmd.SetArgs([]string{"maintenance"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("visual maintenance landing: %v", err)
	}
	visual := out.String()
	for _, operation := range operations {
		id := stringField(t, operation, "operation_id")
		if !strings.Contains(visual, id) {
			t.Errorf("visual output omitted JSON operation id %q", id)
		}
	}
	if strings.Contains(visual, "build-finalize") || strings.Contains(visual, "spawn-complete") {
		t.Fatalf("visual maintenance landing promoted protocol plumbing:\n%s", visual)
	}
}

func runMaintenanceLandingForTest(t *testing.T) map[string]any {
	t.Helper()
	var out bytes.Buffer
	stdout = &out
	rootCmd.SetArgs([]string{"maintenance"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("aether maintenance: %v", err)
	}
	var envelope struct {
		OK     bool           `json:"ok"`
		Result map[string]any `json:"result"`
	}
	if err := json.Unmarshal(out.Bytes(), &envelope); err != nil {
		t.Fatalf("decode maintenance JSON %q: %v", out.String(), err)
	}
	if !envelope.OK {
		t.Fatalf("maintenance envelope not ok: %s", out.String())
	}
	return envelope.Result
}

func mapField(t *testing.T, value map[string]any, field string) map[string]any {
	t.Helper()
	result, ok := value[field].(map[string]any)
	if !ok {
		t.Fatalf("field %s = %T, want object", field, value[field])
	}
	return result
}

func stringField(t *testing.T, value map[string]any, field string) string {
	t.Helper()
	result, ok := value[field].(string)
	if !ok {
		t.Fatalf("field %s = %T, want string", field, value[field])
	}
	return result
}

func operationListField(t *testing.T, value map[string]any, field string) []map[string]any {
	t.Helper()
	raw, ok := value[field].([]any)
	if !ok {
		t.Fatalf("field %s = %T, want array", field, value[field])
	}
	result := make([]map[string]any, 0, len(raw))
	for i, item := range raw {
		operation, ok := item.(map[string]any)
		if !ok {
			t.Fatalf("field %s[%d] = %T, want object", field, i, item)
		}
		result = append(result, operation)
	}
	return result
}

func fingerprintMaintenanceTrees(t *testing.T, roots ...string) map[string][]string {
	t.Helper()
	result := make(map[string][]string, len(roots))
	for _, root := range roots {
		var entries []string
		err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			info, err := entry.Info()
			if err != nil {
				return err
			}
			rel, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			line := fmt.Sprintf("%s|%s|%d|%d", filepath.ToSlash(rel), info.Mode(), info.Size(), info.ModTime().UnixNano())
			if info.Mode().IsRegular() {
				content, err := os.ReadFile(path)
				if err != nil {
					return err
				}
				digest := sha256.Sum256(content)
				line += "|" + hex.EncodeToString(digest[:])
			}
			entries = append(entries, line)
			return nil
		})
		if err != nil {
			t.Fatalf("fingerprint %s: %v", root, err)
		}
		sort.Strings(entries)
		result[root] = entries
	}
	return result
}
