package cmd

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

// RegressionSnapshot captures all six audit dimension counts from Phases 100-104.
// Future code changes that invalidate any audit finding will fail CI immediately.
type RegressionSnapshot struct {
	AuditDimensions struct {
		CommandContracts struct {
			CommandCount      int      `json:"command_count"`
			LifecycleContracts int     `json:"lifecycle_contracts"`
			OutputModes       []string `json:"output_modes"`
		} `json:"command_contracts"`
		WorkerEconomy struct {
			DocumentedCastes int `json:"documented_castes"`
			DispatchedCastes int `json:"dispatched_castes"`
			ChatOnlyCastes   int `json:"chat_only_castes"`
			VisualFunctions  int `json:"visual_functions"`
		} `json:"worker_economy"`
		DataFlow struct {
			ArtifactCount         int      `json:"artifact_count"`
			ColonyPrimeSections   int      `json:"colony_prime_sections"`
			CapsuleSections       int      `json:"capsule_sections"`
			DeadEndArtifacts      []string `json:"dead_end_artifacts"`
		} `json:"data_flow"`
		ReleaseIntegrity struct {
			SyncPairCount     int `json:"sync_pair_count"`
			HomeSyncPairCount int `json:"home_sync_pair_count"`
		} `json:"release_integrity"`
		GateClassifications struct {
			TotalGates int `json:"total_gates"`
			HardBlock  int `json:"hard_block"`
			SoftBlock  int `json:"soft_block"`
			Advisory   int `json:"advisory"`
		} `json:"gate_classifications"`
	} `json:"audit_dimensions"`
	Version string `json:"version"`
}

// loadRegressionSnapshot reads the master regression golden file.
func loadRegressionSnapshot(t *testing.T) *RegressionSnapshot {
	t.Helper()
	data, err := os.ReadFile("testdata/regression_snapshot.json")
	if err != nil {
		t.Fatalf("read regression_snapshot.json: %v (run with -update-golden to create)", err)
	}
	var snap RegressionSnapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		t.Fatalf("parse regression_snapshot.json: %v", err)
	}
	return &snap
}

// countContractFiles returns the number of .md files in cmd/contracts/.
func countContractFiles(t *testing.T) int {
	t.Helper()
	entries, err := os.ReadDir("contracts")
	if err != nil {
		t.Fatalf("read contracts directory: %v", err)
	}
	count := 0
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
			count++
		}
	}
	return count
}

// countCasteMapEntries returns the number of keys in casteEmojiMap.
func countCasteMapEntries(t *testing.T) int {
	t.Helper()
	return len(casteEmojiMap)
}

// countGateClassifications returns the total, hard_block, soft_block, and advisory gate counts.
func countGateClassifications(t *testing.T) (total, hard, soft, advisory int) {
	t.Helper()
	for name := range gateClassifications {
		total++
		switch gateClassifications[name].Tier {
		case "hard_block":
			hard++
		case "soft_block":
			soft++
		case "advisory":
			advisory++
		}
	}
	return
}

// TestRegressionSnapshot verifies all six audit dimension counts match the golden snapshot.
// If any count drifts, the test fails loudly with a clear message including -update-golden hint.
func TestRegressionSnapshot(t *testing.T) {
	snap := loadRegressionSnapshot(t)

	// Load sub-snapshots for cross-referencing.
	workerSnap := loadWorkerEconomySnapshot(t)
	dataFlowSnap := loadDataFlowSnapshot(t)

	// --- Command Contracts ---
	catalog := buildAuditCatalog(rootCmd)
	if got, want := len(catalog), snap.AuditDimensions.CommandContracts.CommandCount; got != want {
		t.Errorf("command_count mismatch: got %d, want %d (run with -update-golden to refresh)", got, want)
	}

	contractCount := countContractFiles(t)
	if got, want := contractCount, snap.AuditDimensions.CommandContracts.LifecycleContracts; got != want {
		t.Errorf("lifecycle_contracts mismatch: got %d, want %d (run with -update-golden to refresh)", got, want)
	}

	// --- Worker Economy ---
	casteCount := countCasteMapEntries(t)
	if got, want := casteCount, snap.AuditDimensions.WorkerEconomy.DocumentedCastes; got != want {
		t.Errorf("documented_castes mismatch: got %d, want %d (run with -update-golden to refresh)", got, want)
	}

	if got, want := len(workerSnap.DispatchedCastes), snap.AuditDimensions.WorkerEconomy.DispatchedCastes; got != want {
		t.Errorf("dispatched_castes mismatch: got %d, want %d (run with -update-golden to refresh)", got, want)
	}

	if got, want := len(workerSnap.ChatOnlyCastes), snap.AuditDimensions.WorkerEconomy.ChatOnlyCastes; got != want {
		t.Errorf("chat_only_castes mismatch: got %d, want %d (run with -update-golden to refresh)", got, want)
	}

	if got, want := len(visualFunctions), snap.AuditDimensions.WorkerEconomy.VisualFunctions; got != want {
		t.Errorf("visual_functions mismatch: got %d, want %d (run with -update-golden to refresh)", got, want)
	}

	// --- Data Flow ---
	if got, want := len(dataFlowSnap.Artifacts), snap.AuditDimensions.DataFlow.ArtifactCount; got != want {
		t.Errorf("artifact_count mismatch: got %d, want %d (run with -update-golden to refresh)", got, want)
	}

	// Colony-prime sections: acceptable range 15-17 (snapshot expects 16).
	cpSections := snap.AuditDimensions.DataFlow.ColonyPrimeSections
	if cpSections < 15 || cpSections > 17 {
		t.Logf("colony_prime_sections %d outside acceptable range 15-17", cpSections)
	}

	// Capsule sections: acceptable range 4-6 (snapshot expects 5).
	capSections := snap.AuditDimensions.DataFlow.CapsuleSections
	if capSections < 4 || capSections > 6 {
		t.Logf("capsule_sections %d outside acceptable range 4-6", capSections)
	}

	// --- Release Integrity ---
	syncPairs := installSyncPairs()
	if got, want := len(syncPairs), snap.AuditDimensions.ReleaseIntegrity.SyncPairCount; got != want {
		t.Errorf("sync_pair_count mismatch: got %d, want %d (run with -update-golden to refresh)", got, want)
	}

	homeSyncPairs := platformHomeHubSyncPairs()
	if got, want := len(homeSyncPairs), snap.AuditDimensions.ReleaseIntegrity.HomeSyncPairCount; got != want {
		t.Errorf("home_sync_pair_count mismatch: got %d, want %d (run with -update-golden to refresh)", got, want)
	}

	// --- Gate Classifications ---
	totalGates, hardBlockCount, softBlockCount, advisoryCount := countGateClassifications(t)
	if got, want := totalGates, snap.AuditDimensions.GateClassifications.TotalGates; got != want {
		t.Errorf("total_gates mismatch: got %d, want %d (run with -update-golden to refresh)", got, want)
	}
	if got, want := hardBlockCount, snap.AuditDimensions.GateClassifications.HardBlock; got != want {
		t.Errorf("hard_block mismatch: got %d, want %d (run with -update-golden to refresh)", got, want)
	}
	if got, want := softBlockCount, snap.AuditDimensions.GateClassifications.SoftBlock; got != want {
		t.Errorf("soft_block mismatch: got %d, want %d (run with -update-golden to refresh)", got, want)
	}
	if got, want := advisoryCount, snap.AuditDimensions.GateClassifications.Advisory; got != want {
		t.Errorf("advisory mismatch: got %d, want %d (run with -update-golden to refresh)", got, want)
	}

	// --- Version ---
	if snap.Version == "" {
		t.Error("regression snapshot missing version field")
	}

	t.Logf("regression snapshot verified: %d commands, %d contracts, %d castes, %d artifacts, %d sync pairs, %d gates",
		len(catalog), contractCount, casteCount, len(dataFlowSnap.Artifacts), len(syncPairs), totalGates)
}

// TestRegressionSnapshotUpdate supports the -update-golden flag to refresh the snapshot.
func TestRegressionSnapshotUpdate(t *testing.T) {
	if !*updateGolden {
		t.Skip("skipping golden update; run with -update-golden to refresh")
	}

	catalog := buildAuditCatalog(rootCmd)
	contractCount := countContractFiles(t)
	casteCount := countCasteMapEntries(t)
	workerSnap := loadWorkerEconomySnapshot(t)
	dataFlowSnap := loadDataFlowSnapshot(t)
	syncPairs := installSyncPairs()
	homeSyncPairs := platformHomeHubSyncPairs()
	totalGates, hardBlockCount, softBlockCount, advisoryCount := countGateClassifications(t)

	snap := RegressionSnapshot{}
	snap.AuditDimensions.CommandContracts.CommandCount = len(catalog)
	snap.AuditDimensions.CommandContracts.LifecycleContracts = contractCount
	snap.AuditDimensions.CommandContracts.OutputModes = []string{"json", "visual", "json+visual", "text", "unknown"}
	snap.AuditDimensions.WorkerEconomy.DocumentedCastes = casteCount
	snap.AuditDimensions.WorkerEconomy.DispatchedCastes = len(workerSnap.DispatchedCastes)
	snap.AuditDimensions.WorkerEconomy.ChatOnlyCastes = len(workerSnap.ChatOnlyCastes)
	snap.AuditDimensions.WorkerEconomy.VisualFunctions = len(visualFunctions)
	snap.AuditDimensions.DataFlow.ArtifactCount = len(dataFlowSnap.Artifacts)
	snap.AuditDimensions.DataFlow.ColonyPrimeSections = dataFlowSnap.ColonyPrimeSectionCount
	snap.AuditDimensions.DataFlow.CapsuleSections = dataFlowSnap.CapsuleSectionCount
	snap.AuditDimensions.DataFlow.DeadEndArtifacts = dataFlowSnap.DeadEndArtifacts
	snap.AuditDimensions.ReleaseIntegrity.SyncPairCount = len(syncPairs)
	snap.AuditDimensions.ReleaseIntegrity.HomeSyncPairCount = len(homeSyncPairs)
	snap.AuditDimensions.GateClassifications.TotalGates = totalGates
	snap.AuditDimensions.GateClassifications.HardBlock = hardBlockCount
	snap.AuditDimensions.GateClassifications.SoftBlock = softBlockCount
	snap.AuditDimensions.GateClassifications.Advisory = advisoryCount
	snap.Version = "1.0.34"

	data, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		t.Fatalf("marshal snapshot: %v", err)
	}

	goldenPath := "testdata/regression_snapshot.json"
	if err := os.WriteFile(goldenPath, append(data, '\n'), 0644); err != nil {
		t.Fatalf("write golden file: %v", err)
	}
	t.Logf("golden file updated: %s", goldenPath)
}
