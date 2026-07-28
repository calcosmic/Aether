package cmd

import (
	"os"
	"strings"
	"testing"
)

// Found by the v1.0.43 release run, not by any local test.
//
// The colony-state fixture was committed as `colony_state.json` while the
// store reads `COLONY_STATE.json`. On macOS (case-insensitive filesystem) the
// lookup silently succeeds anyway, so every developer machine was green; on
// Linux CI the fixture copy produced a file the store could not see, and 28
// tests in status/history/phase/colony-vital-signs failed with the first-run
// welcome banner instead of colony data. The 5-minute CI timeout then hid
// those failures for months by killing the suite before it reached them.
//
// This test reads the ACTUAL on-disk names via os.ReadDir — which returns
// true byte-exact names even on case-insensitive filesystems — so it fails on
// a developer Mac, not only on Linux, if the fixture case ever drifts again.
func TestTestdataFixtureFilenamesMatchStoreCanonicalCase(t *testing.T) {
	entries, err := os.ReadDir("testdata")
	if err != nil {
		t.Fatalf("read testdata: %v", err)
	}

	// Names the store loads from .aether/data that setupTestStore seeds from
	// cmd/testdata. Extend this list if new store-read fixtures are added.
	canonical := []string{"COLONY_STATE.json", "pheromones.json", "flags.json"}

	onDisk := map[string]string{} // lowercase -> exact on-disk name
	for _, e := range entries {
		if !e.IsDir() {
			onDisk[strings.ToLower(e.Name())] = e.Name()
		}
	}

	for _, want := range canonical {
		got, present := onDisk[strings.ToLower(want)]
		if !present {
			continue // fixture not provided at all — that is a different failure, surfaced by the tests that need it
		}
		if got != want {
			t.Errorf("testdata fixture %q must be named %q exactly: the store reads that name, and a case mismatch passes on macOS (case-insensitive) while failing on Linux CI — which is how 28 tests broke invisibly before v1.0.43", got, want)
		}
	}
}
