package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestReconcileMissingStateReportsCacheAndRecovery(t *testing.T) {
	root := t.TempDir()
	dataDir := filepath.Join(root, ".aether", "data")
	mustWriteTestFile(t, filepath.Join(dataDir, ".cache_COLONY_STATE.json"), `{"version":"3.0","goal":"cached"}`)
	mustWriteTestFile(t, filepath.Join(dataDir, "COLONY_STATE.json.bak-plan"), `{"version":"3.0","goal":"backup"}`)
	mustWriteTestFile(t, filepath.Join(dataDir, "session.json"), `{
		"session_id":"s1",
		"colony_goal":"Recover missing state",
		"last_command":"continue",
		"last_command_at":"2026-05-20T00:00:00Z",
		"current_phase":2,
		"suggested_next":"aether reconcile --json",
		"baseline_commit":"old"
	}`)

	report := buildReconcileReport(root, time.Date(2026, 7, 20, 12, 0, 0, 0, time.UTC))

	if report.Status != "action_required" {
		t.Fatalf("status = %q, want action_required", report.Status)
	}
	if report.State.Present {
		t.Fatal("state.present = true, want false")
	}
	if !report.State.CachePresent {
		t.Fatal("state.cache_present = false, want true")
	}
	if report.State.BackupCandidates != 1 {
		t.Fatalf("backup_candidates = %d, want 1", report.State.BackupCandidates)
	}
	if !hasReconcileFinding(report, "critical", "state", "COLONY_STATE.json is missing") {
		t.Fatalf("missing state finding not found: %+v", report.Findings)
	}
	if !hasRecoveryReason(report, "Inspect .aether/data/.cache_COLONY_STATE.json") {
		t.Fatalf("missing cache/backup recovery action: %+v", report.Recovery)
	}
}

func TestReconcileVersionAndPlanningDocMismatch(t *testing.T) {
	root := t.TempDir()
	mustWriteTestFile(t, filepath.Join(root, ".aether", "version.json"), `{"version":"1.0.42"}`)
	mustWriteTestFile(t, filepath.Join(root, "npm", "package.json"), `{"version":"1.0.41"}`)
	mustWriteTestFile(t, filepath.Join(root, ".planning", "STATE.md"), "Product version: v1.0.41\n")
	mustWriteTestFile(t, filepath.Join(root, ".planning", "v1.24-MILESTONE-AUDIT.md"), "audit\n")
	mustWriteTestFile(t, filepath.Join(root, ".aether", "data", "COLONY_STATE.json"), minimalReconcileStateJSON("Version reconciliation"))

	report := buildReconcileReport(root, time.Date(2026, 7, 20, 12, 0, 0, 0, time.UTC))

	if report.Versions.Source != "1.0.42" {
		t.Fatalf("source version = %q, want 1.0.42", report.Versions.Source)
	}
	if report.Versions.NPM != "1.0.41" {
		t.Fatalf("npm version = %q, want 1.0.41", report.Versions.NPM)
	}
	if report.Versions.Aligned {
		t.Fatal("versions aligned = true, want false")
	}
	if !hasReconcileFinding(report, "warning", "version", "npm package version 1.0.41 does not match source version 1.0.42") {
		t.Fatalf("npm version mismatch finding not found: %+v", report.Findings)
	}
	if !hasReconcileFinding(report, "info", "version", "planning docs mention product version 1.0.41") {
		t.Fatalf("version doc mismatch finding not found: %+v", report.Findings)
	}
	if !hasReconcileFinding(report, "info", "planning_docs", "v1.24 audit exists outside the linked milestones path") {
		t.Fatalf("audit path finding not found: %+v", report.Findings)
	}
}

func TestReconcileCountsExpiredSignalsAndUnresolvedFlags(t *testing.T) {
	root := t.TempDir()
	dataDir := filepath.Join(root, ".aether", "data")
	mustWriteTestFile(t, filepath.Join(dataDir, "COLONY_STATE.json"), minimalReconcileStateJSON("Signal reconciliation"))
	mustWriteTestFile(t, filepath.Join(dataDir, "pending-decisions.json"), `{
		"version":"1.0",
		"decisions":[
			{"id":"b1","type":"blocker","description":"blocked","created_at":"2026-05-20T00:00:00Z","resolved":false},
			{"id":"i1","type":"issue","description":"issue","created_at":"2026-05-20T00:00:00Z","resolved":false},
			{"id":"n1","type":"note","description":"done","created_at":"2026-05-20T00:00:00Z","resolved":true}
		]
	}`)
	mustWriteTestFile(t, filepath.Join(dataDir, "pheromones.json"), `{
		"version":"1.0",
		"signals":[
			{"id":"p1","type":"REDIRECT","priority":"high","source":"test","created_at":"2026-05-20T00:00:00Z","expires_at":"2026-06-01T00:00:00Z","active":true,"content":{"text":"expired"}},
			{"id":"p2","type":"FOCUS","priority":"normal","source":"test","created_at":"2026-05-20T00:00:00Z","expires_at":"2026-08-01T00:00:00Z","active":true,"content":{"text":"future"}}
		]
	}`)

	report := buildReconcileReport(root, time.Date(2026, 7, 20, 12, 0, 0, 0, time.UTC))

	if report.Flags.Unresolved != 2 || report.Flags.Blockers != 1 || report.Flags.Issues != 1 {
		t.Fatalf("flag summary = %+v, want unresolved=2 blockers=1 issues=1", report.Flags)
	}
	if report.Pheromones.Active != 2 || report.Pheromones.ExpiredActive != 1 {
		t.Fatalf("pheromone summary = %+v, want active=2 expired_active=1", report.Pheromones)
	}
	if !hasReconcileFinding(report, "warning", "pheromones", "active pheromone signal(s) are expired") {
		t.Fatalf("expired signal finding not found: %+v", report.Findings)
	}
	if !hasReconcileFinding(report, "warning", "flags", "2 unresolved flag(s)") {
		t.Fatalf("unresolved flags finding not found: %+v", report.Findings)
	}
}

func TestReconcileCommandDoesNotCreateColonyState(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	t.Setenv("AETHER_OUTPUT_MODE", "json")

	root := t.TempDir()
	dataDir := filepath.Join(root, ".aether", "data")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		t.Fatalf("mkdir data dir: %v", err)
	}
	t.Setenv("AETHER_ROOT", root)

	var out bytes.Buffer
	stdout = &out
	rootCmd.SetArgs([]string{"reconcile", "--json"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("reconcile returned error: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dataDir, "COLONY_STATE.json")); !os.IsNotExist(err) {
		t.Fatalf("reconcile should not create COLONY_STATE.json, stat err=%v", err)
	}

	env := parseEnvelope(t, out.String())
	if env["ok"] != true {
		t.Fatalf("ok = %v, want true", env["ok"])
	}
	result := env["result"].(map[string]interface{})
	if result["status"] != "action_required" {
		t.Fatalf("status = %v, want action_required", result["status"])
	}
	state := result["state"].(map[string]interface{})
	if state["present"] != false {
		t.Fatalf("state.present = %v, want false", state["present"])
	}
}

func minimalReconcileStateJSON(goal string) string {
	payload := map[string]interface{}{
		"version":       "3.0",
		"goal":          goal,
		"state":         "READY",
		"current_phase": 1,
		"plan": map[string]interface{}{
			"phases": []interface{}{},
		},
		"memory": map[string]interface{}{
			"phase_learnings": []interface{}{},
			"decisions":       []interface{}{},
			"instincts":       []interface{}{},
		},
		"errors": map[string]interface{}{
			"records": []interface{}{},
		},
		"events": []interface{}{},
	}
	data, _ := json.Marshal(payload)
	return string(data)
}

func mustWriteTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func hasReconcileFinding(report reconcileReport, severity, category, contains string) bool {
	for _, finding := range report.Findings {
		if finding.Severity == severity && finding.Category == category && strings.Contains(finding.Message, contains) {
			return true
		}
	}
	return false
}

func hasRecoveryReason(report reconcileReport, contains string) bool {
	for _, action := range report.Recovery {
		if strings.Contains(action.Reason, contains) {
			return true
		}
	}
	return false
}
