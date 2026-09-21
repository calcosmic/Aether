package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

// Found in the owner's first real trial (2026-09-21, v1.0.82): a project whose
// main page sits in a long-named folder could not get past `aether discuss`.
// The specification item ID for a public path was "known-public-path-" plus
// the whole path, against a 40-character canonical cap, so any entry point
// deeper than ~22 characters failed with "semantic lineage exceeds 40
// canonical characters". The fixture is a real repository run through the real
// `init` and `discuss` commands, never a hand-typed survey.
func TestDiscussSettlesWithALongEntryPointPath(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	forceJSONOutputModeForTest(t)

	downstream := createDownstreamRepo(t)
	dataDir := filepath.Join(downstream, ".aether", "data")
	longDir := filepath.Join(downstream, "exports", "pack-2026-09-21-093000-render-a7f3c9e2b4d1-final-preview")
	if err := os.MkdirAll(longDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(longDir, "index.html"), []byte("<html></html>\n"), 0o644); err != nil {
		t.Fatalf("write entry point: %v", err)
	}

	t.Setenv("AETHER_ROOT", downstream)
	t.Setenv("COLONY_DATA_DIR", dataDir)
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(downstream); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldDir) })

	var out bytes.Buffer
	stdout, stderr = &out, &out
	t.Cleanup(func() { stdout, stderr = os.Stdout, os.Stderr })

	rootCmd.SetArgs([]string{"init", "Tidy the export packs"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("init failed: %v\n%s", err, out.String())
	}
	out.Reset()
	rootCmd.SetArgs([]string{"discuss"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("discuss returned error: %v\n%s", err, out.String())
	}
	got := out.String()
	if strings.Contains(got, "semantic lineage exceeds") || strings.Contains(got, `"ok":false`) {
		t.Fatalf("discuss refused a project with a long entry-point path:\n%s", got)
	}
	// Guard the fixture: the long path must really have been surveyed as an
	// entry point, or this test proves nothing.
	var state colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
		t.Fatalf("load state: %v", err)
	}
	if state.Specification == nil || len(state.Specification.Revisions) == 0 {
		t.Fatalf("discuss did not create a specification draft:\n%s", got)
	}
	found := false
	for _, revision := range state.Specification.Revisions {
		for _, item := range revision.AffectedPublicPaths {
			if strings.Contains(item.Path, "pack-2026-09-21-093000-render-a7f3c9e2b4d1-final-preview") {
				found = true
			}
		}
	}
	if !found {
		t.Fatalf("the long entry-point path never reached the specification, so the fixture is vacuous:\n%s", got)
	}
}

func TestSpecPublicPathLineageIsBoundedStableAndDistinct(t *testing.T) {
	short := "cmd/main.go"
	if got := specPublicPathLineage(short); got != "known-public-path-"+short {
		t.Fatalf("a short path must keep its existing lineage (stable IDs for existing specifications): %q", got)
	}
	longA := "exports/pack-2026-09-21-093000-render-a7f3c9e2b4d1-final-preview/index.html"
	longB := "exports/pack-2026-09-21-101500-render-0c1d2e3f4a5b-final-preview/index.html"
	a, b := specPublicPathLineage(longA), specPublicPathLineage(longB)
	if a == b {
		t.Fatalf("two different long paths share one lineage: %q", a)
	}
	if a != specPublicPathLineage(longA) {
		t.Fatalf("lineage is not deterministic")
	}
	for _, path := range []string{short, longA, longB, strings.Repeat("deep/", 60) + "index.html"} {
		if _, err := colony.CanonicalSpecItemID(colony.SpecSectionAffectedPublicPaths, specPublicPathLineage(path)); err != nil {
			t.Fatalf("lineage for %q is not a valid specification item lineage: %v", path, err)
		}
	}
}
