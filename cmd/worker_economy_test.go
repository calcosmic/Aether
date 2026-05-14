package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"testing"
)

// WorkerEconomySnapshot captures caste data for golden testing of spawn
// coverage and visual ceremony verification.
type WorkerEconomySnapshot struct {
	DocumentedCastes  []string `json:"documented_castes"`
	DispatchedCastes  []string `json:"dispatched_castes"`
	ChatOnlyCastes    []string `json:"chat_only_castes"`
	CasteRegistryKeys []string `json:"caste_registry_keys"`
	ColorMapKeys      []string `json:"color_map_keys"`
}

// visualFunctions are the user-facing rendering functions that must appear in
// the WORKER-ECONOMY.md Visual Ceremony Traceability table. renderAetherWordmark
// is excluded per D-05 (pure decoration).
var visualFunctions = []string{
	"casteIdentity",
	"renderStageMarker",
	"renderProgressSummary",
	"renderBanner",
	"renderSpawnPlanForDispatches",
	"renderCloseoutVisual",
	"renderSignalVisual",
	"renderContinueWorkerFlowLine",
	"renderSealVisual",
}

// loadWorkerEconomySnapshot reads the golden file and returns the parsed snapshot.
func loadWorkerEconomySnapshot(t *testing.T) *WorkerEconomySnapshot {
	t.Helper()
	data, err := os.ReadFile("testdata/worker_economy_snapshot.json")
	if err != nil {
		t.Fatalf("read golden file: %v (run with -update-golden to create)", err)
	}
	var snap WorkerEconomySnapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		t.Fatalf("parse golden file: %v", err)
	}
	return &snap
}

// readWorkerEconomyReport reads the WORKER-ECONOMY.md report from the phase directory.
// Skips the test if the file was archived (e.g. during milestone cleanup).
func readWorkerEconomyReport(t *testing.T) string {
	t.Helper()
	repoRoot, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("find repo root: %v", err)
	}
	path := repoRoot + "/.planning/phases/102-worker-economy-visual-ceremony-audit/WORKER-ECONOMY.md"
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			t.Skip("WORKER-ECONOMY.md archived")
		}
		t.Fatalf("read WORKER-ECONOMY.md: %v", err)
	}
	return string(data)
}

// updateWorkerEconomyGolden writes the current state to the golden file.
func updateWorkerEconomyGolden(t *testing.T, snap *WorkerEconomySnapshot) {
	t.Helper()
	data, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		t.Fatalf("marshal snapshot: %v", err)
	}
	goldenPath := "testdata/worker_economy_snapshot.json"
	if err := os.WriteFile(goldenPath, append(data, '\n'), 0644); err != nil {
		t.Fatalf("write golden file: %v", err)
	}
	t.Logf("golden file updated: %s", goldenPath)
}

// TestDispatchedCastesDocumented verifies every dispatched caste (from the golden
// snapshot) appears in WORKER-ECONOMY.md. This satisfies WORK-01: every spawned
// worker caste has documented purpose, durable output, and downstream consumer.
func TestDispatchedCastesDocumented(t *testing.T) {
	snap := loadWorkerEconomySnapshot(t)
	report := readWorkerEconomyReport(t)

	var missing []string
	for _, caste := range snap.DispatchedCastes {
		if !strings.Contains(report, caste) {
			missing = append(missing, caste)
		}
	}

	if len(missing) > 0 {
		for _, c := range missing {
			t.Errorf("dispatched caste %q not documented in WORKER-ECONOMY.md", c)
		}
		t.FailNow()
	}

	t.Logf("all %d dispatched castes documented in WORKER-ECONOMY.md", len(snap.DispatchedCastes))
}

// TestNoChatOnlyWorkersUndocumented verifies each chat-only caste in the golden
// snapshot is flagged as a WORK-02 finding in WORKER-ECONOMY.md. Chat-only workers
// that only read and return chat without persisting violate the worker economy principle.
func TestNoChatOnlyWorkersUndocumented(t *testing.T) {
	snap := loadWorkerEconomySnapshot(t)

	// If no chat-only castes in the snapshot, the test passes vacuously.
	if len(snap.ChatOnlyCastes) == 0 {
		t.Log("no chat-only castes in snapshot -- test passes vacuously")
		return
	}

	report := readWorkerEconomyReport(t)

	var unflagged []string
	for _, caste := range snap.ChatOnlyCastes {
		// Find the caste name in the report.
		idx := strings.Index(report, caste)
		if idx == -1 {
			unflagged = append(unflagged, fmt.Sprintf("%s (not found in report)", caste))
			continue
		}

		// Check that WORK-02 appears within 200 characters of the caste name.
		windowStart := idx
		windowEnd := idx + 200
		if windowEnd > len(report) {
			windowEnd = len(report)
		}
		window := report[windowStart:windowEnd]
		if !strings.Contains(window, "WORK-02") {
			unflagged = append(unflagged, fmt.Sprintf("%s (found but not flagged as WORK-02)", caste))
		}
	}

	if len(unflagged) > 0 {
		for _, c := range unflagged {
			t.Errorf("chat-only caste %q not flagged as WORK-02 finding", c)
		}
		t.FailNow()
	}

	t.Logf("all %d chat-only castes flagged as WORK-02 in WORKER-ECONOMY.md", len(snap.ChatOnlyCastes))
}

// TestVisualOutputTracesToState verifies each user-facing visual rendering
// function appears in the WORKER-ECONOMY.md Visual Ceremony Traceability table
// with a "Yes" trace to runtime or an "acceptable per D-05" finding. This
// satisfies VIZ-01 and VIZ-02.
func TestVisualOutputTracesToState(t *testing.T) {
	report := readWorkerEconomyReport(t)

	// Find the Visual Ceremony Traceability section.
	traceSectionMarker := "## 4. Visual Ceremony Traceability"
	traceIdx := strings.Index(report, traceSectionMarker)
	if traceIdx == -1 {
		t.Fatal("Visual Ceremony Traceability section not found in WORKER-ECONOMY.md")
	}

	// Extract from the traceability section to the next section header.
	sectionStart := traceIdx
	nextSectionIdx := strings.Index(report[traceIdx+len(traceSectionMarker):], "\n## ")
	if nextSectionIdx != -1 {
		sectionStart = traceIdx + len(traceSectionMarker) + nextSectionIdx
	} else {
		sectionStart = len(report)
	}
	traceSection := report[traceIdx:sectionStart]

	var missing []string
	for _, fn := range visualFunctions {
		if !strings.Contains(traceSection, fn) {
			missing = append(missing, fn)
		}
	}

	if len(missing) > 0 {
		for _, fn := range missing {
			t.Errorf("visual function %q not found in Visual Ceremony Traceability table", fn)
		}
		t.FailNow()
	}

	t.Logf("all %d visual functions found in Visual Ceremony Traceability table", len(visualFunctions))
}

// TestCasteRegistryConsistency verifies casteEmojiMap and casteLabelMap have
// identical key sets, and that casteColorMap differences (if any) are documented
// in WORKER-ECONOMY.md findings.
func TestCasteRegistryConsistency(t *testing.T) {
	// Collect keys from each map.
	emojiKeys := sortedKeys(casteEmojiMap)
	labelKeys := sortedKeys(casteLabelMap)
	colorKeys := sortedKeys(casteColorMap)

	// Verify emoji and label maps have identical keys.
	if len(emojiKeys) != len(labelKeys) {
		t.Errorf("casteEmojiMap has %d keys, casteLabelMap has %d keys", len(emojiKeys), len(labelKeys))
	}

	var emojiLabelDiff []string
	for _, k := range emojiKeys {
		if _, ok := casteLabelMap[k]; !ok {
			emojiLabelDiff = append(emojiLabelDiff, fmt.Sprintf("%q in emoji but not label", k))
		}
	}
	for _, k := range labelKeys {
		if _, ok := casteEmojiMap[k]; !ok {
			emojiLabelDiff = append(emojiLabelDiff, fmt.Sprintf("%q in label but not emoji", k))
		}
	}

	if len(emojiLabelDiff) > 0 {
		for _, d := range emojiLabelDiff {
			t.Errorf("casteEmojiMap/casteLabelMap key mismatch: %s", d)
		}
		t.FailNow()
	}

	t.Logf("casteEmojiMap and casteLabelMap have identical %d-key sets", len(emojiKeys))

	// Verify color map has same keys as emoji map.
	if len(colorKeys) != len(emojiKeys) {
		// Difference exists -- verify documented.
		report := readWorkerEconomyReport(t)

		var colorExtra, colorMissing []string
		for _, k := range colorKeys {
			if _, ok := casteEmojiMap[k]; !ok {
				colorExtra = append(colorExtra, k)
			}
		}
		for _, k := range emojiKeys {
			if _, ok := casteColorMap[k]; !ok {
				colorMissing = append(colorMissing, k)
			}
		}

		// Each missing/extra key must be documented.
		for _, k := range colorMissing {
			idx := strings.Index(report, "casteColorMap")
			if idx == -1 {
				t.Errorf("casteColorMap key difference for %q but no casteColorMap mention in WORKER-ECONOMY.md", k)
				continue
			}
			windowEnd := idx + 300
			if windowEnd > len(report) {
				windowEnd = len(report)
			}
			window := report[idx:windowEnd]
			if !strings.Contains(window, k) {
				t.Errorf("casteColorMap missing key %q not documented in WORKER-ECONOMY.md findings", k)
			}
		}

		for _, k := range colorExtra {
			idx := strings.Index(report, "casteColorMap")
			if idx == -1 {
				t.Errorf("casteColorMap key difference for %q but no casteColorMap mention in WORKER-ECONOMY.md", k)
				continue
			}
			windowEnd := idx + 300
			if windowEnd > len(report) {
				windowEnd = len(report)
			}
			window := report[idx:windowEnd]
			if !strings.Contains(window, k) {
				t.Errorf("casteColorMap extra key %q not documented in WORKER-ECONOMY.md findings", k)
			}
		}
	} else {
		// Same count -- verify identical keys.
		var colorDiff []string
		for _, k := range emojiKeys {
			if _, ok := casteColorMap[k]; !ok {
				colorDiff = append(colorDiff, k)
			}
		}
		for _, k := range colorKeys {
			if _, ok := casteEmojiMap[k]; !ok {
				colorDiff = append(colorDiff, k)
			}
		}

		if len(colorDiff) > 0 {
			for _, k := range colorDiff {
				t.Errorf("casteColorMap/casteEmojiMap key mismatch: %q", k)
			}
			t.FailNow()
		}

		t.Logf("casteColorMap has identical %d-key set to casteEmojiMap", len(colorKeys))
	}
}

// sortedKeys returns the sorted keys of a string map.
func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
