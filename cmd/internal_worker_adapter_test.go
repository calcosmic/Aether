package cmd

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/codex"
)

func TestInternalWorkerAdapterSimulatesOnlyWithExplicitFlag(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	requestPath := writeInternalWorkerRequestForTest(t, map[string]interface{}{
		"schema_version": 1,
		"caste":          "builder",
		"worker_name":    "Builder-Test",
		"task_id":        "1.1",
		"task":           "Exercise the adapter boundary",
		"skill_section":  "skill guidance",
		"hive_section":   "verified hive guidance",
	})

	response, err := runInternalWorkerAdapter(context.Background(), requestPath, false, true)
	if err != nil {
		t.Fatalf("explicit simulated adapter failed: %v", err)
	}
	if response.ExecutionOwner != "go-adapter" || response.Platform != codex.PlatformFake {
		t.Fatalf("unexpected simulated ownership response: %+v", response)
	}
	if response.Worker == nil || response.Worker.Status != "completed" || response.Worker.Name != "Builder-Test" {
		t.Fatalf("simulated worker result = %+v", response.Worker)
	}
	if response.PermissionDecision == nil || !response.PermissionDecision.Allowed || response.PermissionDecision.Profile.Name != codex.PermissionWorkspaceWrite {
		t.Fatalf("simulated adapter omitted canonical permission decision: %+v", response.PermissionDecision)
	}

	t.Setenv("AETHER_CODEX_REAL_DISPATCH", "fake")
	if _, err := runInternalWorkerAdapter(context.Background(), requestPath, false, false); err == nil || !strings.Contains(err.Error(), "synthetic worker dispatch is disabled") {
		t.Fatalf("production adapter accepted implicit fake mode: %v", err)
	}
}

func TestInternalWorkerAdapterRequiresAndValidatesPermissionProfile(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	base := map[string]interface{}{
		"schema_version": 1,
		"caste":          "includer",
		"worker_name":    "Includer-Test",
		"task":           "Inspect without writing",
	}

	missing := writeInternalWorkerRequestForTestRaw(t, base)
	request, err := loadInternalWorkerRequest(missing)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := internalWorkerConfig(root, &codex.FakeInvoker{}, request); err == nil || !strings.Contains(err.Error(), "requires permission_profile") {
		t.Fatalf("missing permission profile was accepted: %v", err)
	}

	broad := cloneWorkerRequestMap(base)
	broad["permission_profile"] = codex.PermissionProfileForCaste("builder")
	broadPath := writeInternalWorkerRequestForTestRaw(t, broad)
	broadRequest, err := loadInternalWorkerRequest(broadPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := internalWorkerConfig(root, &codex.FakeInvoker{}, broadRequest); err == nil || !strings.Contains(err.Error(), "mismatch") {
		t.Fatalf("broadened includer profile was accepted: %v", err)
	}
}

func TestInternalWorkerAdapterRequestIsStrictAndNonSymlink(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	unknown := writeInternalWorkerRequestForTest(t, map[string]interface{}{
		"schema_version": 1,
		"caste":          "builder",
		"worker_name":    "Builder-Test",
		"task":           "Test strict decoding",
		"unknown":        true,
	})
	if _, err := loadInternalWorkerRequest(unknown); err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("strict request decoder did not reject unknown field: %v", err)
	}

	target := writeInternalWorkerRequestForTest(t, map[string]interface{}{
		"schema_version": 1,
		"caste":          "builder",
		"worker_name":    "Builder-Test",
		"task":           "Test symlink rejection",
	})
	link := filepath.Join(filepath.Dir(target), "worker-link.json")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if _, err := loadInternalWorkerRequest(link); err == nil || !strings.Contains(err.Error(), "non-symlink") {
		t.Fatalf("worker request symlink was not rejected: %v", err)
	}
}

func TestInternalWorkerAdapterRejectsMalformedTerminalResult(t *testing.T) {
	request := internalWorkerDispatchRequest{
		SchemaVersion: 1,
		Caste:         "builder",
		WorkerName:    "Builder-Test",
		TaskID:        "1.1",
		Task:          "Test malformed result",
	}
	result := codex.WorkerResult{
		WorkerName: "Builder-Test",
		Caste:      "builder",
		TaskID:     "1.1",
		Status:     "probably-fine",
	}
	if err := validateInternalWorkerResult(request, &result, nil); err == nil || !strings.Contains(err.Error(), "invalid terminal status") {
		t.Fatalf("malformed terminal result was accepted: %v", err)
	}
	result.Status = "completed"
	result.WorkerName = "Different-Worker"
	if err := validateInternalWorkerResult(request, &result, nil); err == nil || !strings.Contains(err.Error(), "identity") {
		t.Fatalf("mismatched worker identity was accepted: %v", err)
	}
}

func TestInternalWorkerAdapterIsHiddenFromPublicCommandCatalog(t *testing.T) {
	for _, entry := range buildAuditCatalog(rootCmd) {
		if entry.Name == "internal-worker-adapter" {
			t.Fatal("internal adapter boundary leaked into the public command catalog")
		}
	}
}

func writeInternalWorkerRequestForTest(t *testing.T, request map[string]interface{}) string {
	t.Helper()
	if _, ok := request["permission_profile"]; !ok {
		caste, _ := request["caste"].(string)
		request["permission_profile"] = codex.PermissionProfileForCaste(caste)
	}
	return writeInternalWorkerRequestForTestRaw(t, request)
}

func writeInternalWorkerRequestForTestRaw(t *testing.T, request map[string]interface{}) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "aether-worker-request-test-")
	if err != nil {
		t.Fatalf("create worker request temp dir: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	data, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("marshal worker request: %v", err)
	}
	path := filepath.Join(dir, "worker-request.json")
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatalf("write worker request: %v", err)
	}
	return path
}

func cloneWorkerRequestMap(input map[string]interface{}) map[string]interface{} {
	out := make(map[string]interface{}, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}
