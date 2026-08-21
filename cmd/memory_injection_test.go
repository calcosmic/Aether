package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/storage"
)

// Sentinel strings used across both memory-injection tests. Each one is
// distinctive enough that it could not occur in the assembled brief by
// accident — if any of these strings appear in a wiped fixture's output,
// something is leaking colony memory unconditionally instead of reading it
// from the seeded files.
const (
	sentinelInstinctAction = "SENTINEL-INSTINCT-ACTION-162"
	sentinelQueenWisdom    = "SENTINEL-QUEEN-WISDOM-162"
	sentinelHiveWisdom     = "SENTINEL-HIVE-WISDOM-162"
)

// seedMemoryColony creates a fresh colony fixture (temp data dir + temp hub
// dir) and returns the constructed store. When populate is true, instincts.json,
// .aether/QUEEN.md and the hub's hive/wisdom.json all carry their sentinel.
// When populate is false, none of the three memory files carry any sentinel:
// instincts.json and QUEEN.md are absent entirely, and the hub wisdom.json
// exists but is empty — proving the assembled brief differs only because of
// what those files contain, not because of some other structural difference.
func seedMemoryColony(t *testing.T, populate bool) {
	t.Helper()

	tmpDir := t.TempDir()
	dataDir := filepath.Join(tmpDir, ".aether", "data")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		t.Fatalf("mkdir data dir: %v", err)
	}
	hubDir := filepath.Join(tmpDir, "hub")
	if err := os.MkdirAll(filepath.Join(hubDir, "hive"), 0755); err != nil {
		t.Fatalf("mkdir hub hive dir: %v", err)
	}
	t.Setenv("AETHER_HUB_DIR", hubDir)

	s, err := storage.NewStore(dataDir)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	store = s

	goal := "prove the learning loop reaches a worker"
	state := colony.ColonyState{
		Version:      "1.0",
		Goal:         &goal,
		State:        colony.StateREADY,
		CurrentPhase: 1,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{ID: 1, Name: "Switch On Learning", Status: colony.PhaseReady},
			},
		},
		ParallelMode: colony.ModeInRepo,
	}
	if err := s.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatalf("save colony state: %v", err)
	}

	localQueenPath := filepath.Join(tmpDir, ".aether", "QUEEN.md")
	hiveWisdomPath := filepath.Join(hubDir, "hive", "wisdom.json")

	if populate {
		instinctsFile := colony.InstinctsFile{
			Version: "1.0",
			Instincts: []colony.InstinctEntry{
				{
					ID:         "inst-162",
					Trigger:    "memory injection proof",
					Action:     sentinelInstinctAction,
					Confidence: 0.8,
					Archived:   false,
				},
			},
		}
		if err := s.SaveJSON("instincts.json", instinctsFile); err != nil {
			t.Fatalf("save instincts: %v", err)
		}

		queenContent := "## Wisdom\n\n- " + sentinelQueenWisdom + "\n"
		if err := os.WriteFile(localQueenPath, []byte(queenContent), 0644); err != nil {
			t.Fatalf("write local QUEEN.md: %v", err)
		}

		hiveData := hiveWisdomData{
			Version: hiveWisdomSchemaVersion,
			Entries: []hiveWisdomEntry{
				{
					ID:         "hive-162",
					Text:       sentinelHiveWisdom,
					Domain:     "general",
					Confidence: 0.9,
				},
			},
		}
		hiveJSON, err := json.MarshalIndent(hiveData, "", "  ")
		if err != nil {
			t.Fatalf("marshal hive wisdom: %v", err)
		}
		if err := os.WriteFile(hiveWisdomPath, hiveJSON, 0644); err != nil {
			t.Fatalf("write hive wisdom: %v", err)
		}
		return
	}

	// Wiped: no instincts.json, no .aether/QUEEN.md. The hub wisdom.json
	// exists (matching what a real, memory-less colony's hub directory looks
	// like after `aether hive-init`) but carries zero entries.
	emptyHive := hiveWisdomData{Version: hiveWisdomSchemaVersion, Entries: []hiveWisdomEntry{}}
	emptyHiveJSON, err := json.MarshalIndent(emptyHive, "", "  ")
	if err != nil {
		t.Fatalf("marshal empty hive wisdom: %v", err)
	}
	if err := os.WriteFile(hiveWisdomPath, emptyHiveJSON, 0644); err != nil {
		t.Fatalf("write empty hive wisdom: %v", err)
	}
}

// TestWorkerBriefMemoryInjection is Layer 1 of D-08's two-layer proof: a
// permanent, deterministic, model-free CI test asserting that populated
// colony memory changes what buildColonyPrimeOutput assembles for a worker,
// and that wiped colony memory does not carry any of it. It makes no model
// calls and spawns no workers — every input is a fixture file on disk.
func TestWorkerBriefMemoryInjection(t *testing.T) {
	var populatedLen, wipedLen int

	t.Run("populated", func(t *testing.T) {
		saveGlobals(t)

		// Deliberately does not set AETHER_HIVE_POLICY. After plan 01 the
		// default runtime policy is "promote", so hive wisdom must reach the
		// assembled brief with no configuration at all. If this subtest only
		// passes with the env var set, plan 01's default flip regressed —
		// fix that, not this test.
		seedMemoryColony(t, true)

		output := buildColonyPrimeOutput(true)
		section := output.PromptSection
		populatedLen = len(section)

		for _, sentinel := range []string{sentinelInstinctAction, sentinelQueenWisdom, sentinelHiveWisdom} {
			if !strings.Contains(section, sentinel) {
				t.Errorf("populated brief missing sentinel %q:\n%s", sentinel, section)
			}
		}
		for _, header := range []string{"## Active Instincts", "## LOCAL QUEEN WISDOM", "## HIVE WISDOM"} {
			if !strings.Contains(section, header) {
				t.Errorf("populated brief missing section header %q:\n%s", header, section)
			}
		}
	})

	t.Run("wiped", func(t *testing.T) {
		saveGlobals(t)

		seedMemoryColony(t, false)

		output := buildColonyPrimeOutput(true)
		section := output.PromptSection
		wipedLen = len(section)

		for _, sentinel := range []string{sentinelInstinctAction, sentinelQueenWisdom, sentinelHiveWisdom} {
			if strings.Contains(section, sentinel) {
				t.Errorf("wiped brief unexpectedly contains sentinel %q — memory injection is not reading from the seeded files:\n%s", sentinel, section)
			}
		}
		for _, header := range []string{"## Active Instincts", "## LOCAL QUEEN WISDOM", "## HIVE WISDOM"} {
			if strings.Contains(section, header) {
				t.Errorf("wiped brief unexpectedly contains section header %q with no backing data:\n%s", header, section)
			}
		}
	})

	// Prefer this proportion-style invariant over a bare header-exists check
	// (CLAUDE.md: a test that only checks a named section exists cannot catch
	// its replacement). A populated colony must produce a strictly longer
	// assembled brief than a wiped one.
	if populatedLen == 0 || wipedLen == 0 {
		t.Fatalf("subtests did not record brief lengths: populated=%d wiped=%d", populatedLen, wipedLen)
	}
	if !(populatedLen > wipedLen) {
		t.Fatalf("expected populated brief (%d chars) to be strictly longer than wiped brief (%d chars)", populatedLen, wipedLen)
	}
}

// TestResolveCodexWorkerContextCarriesMemory proves the funnel every real
// dispatch path calls — resolveCodexWorkerContext — carries populated colony
// memory through to a real worker, not merely buildColonyPrimeOutput in
// isolation.
func TestResolveCodexWorkerContextCarriesMemory(t *testing.T) {
	saveGlobals(t)

	seedMemoryColony(t, true)

	context := resolveCodexWorkerContext()
	if context == "" {
		t.Fatalf("resolveCodexWorkerContext returned empty context for a populated colony")
	}

	for _, sentinel := range []string{sentinelInstinctAction, sentinelQueenWisdom, sentinelHiveWisdom} {
		if !strings.Contains(context, sentinel) {
			t.Errorf("resolveCodexWorkerContext output missing sentinel %q:\n%s", sentinel, context)
		}
	}
}
