package cmd

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestShouldAutoSpawnMedic_HealthyColony(t *testing.T) {
	dir := t.TempDir()
	check := shouldAutoSpawnMedic(dir)
	if check.ShouldSpawn {
		t.Error("expected no auto-spawn for empty (healthy) colony")
	}
}

func TestShouldAutoSpawnMedic_StaleSession(t *testing.T) {
	dir := t.TempDir()
	// Create session.json with old modification time
	sessionPath := filepath.Join(dir, "session.json")
	if err := os.WriteFile(sessionPath, []byte(`{"session_id":"test"}`), 0644); err != nil {
		t.Fatal(err)
	}
	// Set modification time to 25 hours ago
	oldTime := time.Now().Add(-25 * time.Hour)
	if err := os.Chtimes(sessionPath, oldTime, oldTime); err != nil {
		t.Fatal(err)
	}

	check := shouldAutoSpawnMedic(dir)
	if !check.ShouldSpawn {
		t.Error("expected auto-spawn for stale session")
	}
	if check.Reason == "" {
		t.Error("expected reason for stale session")
	}
	if check.Severity != "warning" {
		t.Errorf("expected severity warning, got %s", check.Severity)
	}
}

func TestShouldAutoSpawnMedic_CriticalBlocker(t *testing.T) {
	dir := t.TempDir()
	// Create pending-decisions.json with unresolved blocker
	flags := map[string]interface{}{
		"decisions": []map[string]interface{}{
			{"id": "b1", "type": "blocker", "description": "test blocker", "resolved": false},
		},
	}
	data, _ := json.Marshal(flags)
	if err := os.WriteFile(filepath.Join(dir, "pending-decisions.json"), data, 0644); err != nil {
		t.Fatal(err)
	}

	check := shouldAutoSpawnMedic(dir)
	if !check.ShouldSpawn {
		t.Error("expected auto-spawn for critical blocker")
	}
	if check.Severity != "critical" {
		t.Errorf("expected severity critical, got %s", check.Severity)
	}
}

func TestShouldAutoSpawnMedic_ResolvedBlocker(t *testing.T) {
	dir := t.TempDir()
	// Create pending-decisions.json with resolved blocker (should NOT trigger)
	flags := map[string]interface{}{
		"decisions": []map[string]interface{}{
			{"id": "b1", "type": "blocker", "description": "test", "resolved": true},
		},
	}
	data, _ := json.Marshal(flags)
	if err := os.WriteFile(filepath.Join(dir, "pending-decisions.json"), data, 0644); err != nil {
		t.Fatal(err)
	}

	check := shouldAutoSpawnMedic(dir)
	if check.ShouldSpawn {
		t.Error("expected no auto-spawn for resolved blocker")
	}
}

func TestShouldAutoSpawnMedic_CriticalHealthIssue(t *testing.T) {
	dir := t.TempDir()
	// Create a fresh session.json so stale check doesn't trigger
	sessionPath := filepath.Join(dir, "session.json")
	os.WriteFile(sessionPath, []byte(`{}`), 0644)

	// Create medic-last-scan.json with critical issue
	scan := MedicLastScan{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Issues: []HealthIssue{
			{Severity: "critical", Category: "state", Message: "State corrupted"},
		},
	}
	data, _ := json.Marshal(scan)
	if err := os.WriteFile(filepath.Join(dir, "medic-last-scan.json"), data, 0644); err != nil {
		t.Fatal(err)
	}

	check := shouldAutoSpawnMedic(dir)
	if !check.ShouldSpawn {
		t.Error("expected auto-spawn for critical health issue")
	}
}

func TestCheckStaleSession_NoFile(t *testing.T) {
	stale, _ := checkStaleSession(t.TempDir())
	if stale {
		t.Error("expected not stale when session.json missing")
	}
}

func TestCheckStaleSession_FreshSession(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "session.json"), []byte(`{}`), 0644)
	stale, _ := checkStaleSession(dir)
	if stale {
		t.Error("expected not stale for fresh session")
	}
}

func TestSaveAndLoadMedicLastScan(t *testing.T) {
	dir := t.TempDir()
	issues := []HealthIssue{
		{Severity: "warning", Category: "test", Message: "test issue"},
	}
	if err := saveMedicLastScan(dir, issues, "test goal", 3); err != nil {
		t.Fatalf("saveMedicLastScan failed: %v", err)
	}

	scan, err := loadMedicLastScan(dir)
	if err != nil {
		t.Fatalf("loadMedicLastScan failed: %v", err)
	}
	if scan.Goal != "test goal" {
		t.Errorf("expected goal 'test goal', got %s", scan.Goal)
	}
	if scan.Phase != 3 {
		t.Errorf("expected phase 3, got %d", scan.Phase)
	}
	if len(scan.Issues) != 1 {
		t.Errorf("expected 1 issue, got %d", len(scan.Issues))
	}
	if scan.Issues[0].Message != "test issue" {
		t.Errorf("expected 'test issue', got %s", scan.Issues[0].Message)
	}
}

func TestLoadMedicLastScan_NoFile(t *testing.T) {
	_, err := loadMedicLastScan(t.TempDir())
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestHasCriticalBlocker_FlagsFallback(t *testing.T) {
	dir := t.TempDir()
	// Only flags.json, no pending-decisions.json
	flags := map[string]interface{}{
		"decisions": []map[string]interface{}{
			{"id": "b1", "type": "blocker", "description": "test", "resolved": false},
		},
	}
	data, _ := json.Marshal(flags)
	if err := os.WriteFile(filepath.Join(dir, "flags.json"), data, 0644); err != nil {
		t.Fatal(err)
	}

	if !hasCriticalBlocker(dir) {
		t.Error("expected critical blocker in flags.json")
	}
}

func TestHasCriticalHealthIssue_NoFile(t *testing.T) {
	if hasCriticalHealthIssue(t.TempDir()) {
		t.Error("expected false when no scan file exists")
	}
}

func TestHasCriticalHealthIssue_WarningsOnly(t *testing.T) {
	dir := t.TempDir()
	scan := MedicLastScan{
		Issues: []HealthIssue{
			{Severity: "warning", Message: "just a warning"},
		},
	}
	data, _ := json.Marshal(scan)
	os.WriteFile(filepath.Join(dir, "medic-last-scan.json"), data, 0644)

	if hasCriticalHealthIssue(dir) {
		t.Error("expected false when only warnings exist")
	}
}

func TestMedicAutoSpawnEligibilityIsInternalOnly(t *testing.T) {
	for _, entry := range buildAuditCatalog(rootCmd) {
		if entry.Name == "medic-auto-spawn-check" {
			t.Error("retired Medic eligibility command is still registered")
		}
	}

	file, err := parser.ParseFile(token.NewFileSet(), "medic_auto_spawn.go", nil, 0)
	if err != nil {
		t.Fatalf("parse medic_auto_spawn.go: %v", err)
	}
	retiredSymbols := map[string]struct{}{
		"medicAutoSpawnCheckCmd":     {},
		"renderMedicAutoSpawnVisual": {},
	}
	ast.Inspect(file, func(node ast.Node) bool {
		identifier, ok := node.(*ast.Ident)
		if !ok {
			return true
		}
		if _, retired := retiredSymbols[identifier.Name]; retired {
			t.Errorf("retired Medic adapter symbol %q remains", identifier.Name)
		}
		return true
	})

	if !sourceCallsFunction(t, "autopilot_report.go", "shouldAutoSpawnMedic") {
		t.Error("autopilot recovery must call retained Medic eligibility directly")
	}
}
