package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

// Package note (198.2-01, WIRE-01): every test in this file must exercise
// the delegate/plan-only manifest path — codexPlanManifest.ContextCapsule
// (and, once Task 2 lands, codexColonizeManifest.ContextCapsule) — never
// only the in-process/native dispatch path
// (dispatchRealPlanningWorkersWithIterationContext /
// codex.WorkerDispatch.ContextCapsule). A subtest that only exercises the
// in-process dispatch does not satisfy WIRE-01: that is exactly the shape of
// TestEightCommandsDeliverPheromoneExactlyOnce, named in
// 198.2-CONTEXT.md as the test that stayed green while the shipped
// (delegate/wrapper) lane carried nothing.

// TestPlanDelegateManifestCarriesOneSteeringNote pins WIRE-01/D-03 for the
// plan command: a --plan-only manifest run on a colony carrying one active
// owner steering note (a pheromone signal, seeded through the runtime's own
// writePheromoneSignal path -- never a hand-typed JSON literal) must carry
// that note's exact text inside plan_manifest.context_capsule. It also
// pins the two boundary cases named in the task's <behavior> block: a
// memory-free colony omits the field entirely (omitempty, not empty
// string), and the hosted (non-plan-only, non-delegate) path leaves the
// field empty because that path already shares its own capsule with each
// worker dispatch.
func TestPlanDelegateManifestCarriesOneSteeringNote(t *testing.T) {
	saveGlobalsCmd(t)

	s, tmpDir := newTestStoreCmd(t)
	defer os.RemoveAll(tmpDir)
	store = s

	goal := "198.2-01 plan delegate capsule test colony"
	if err := s.SaveJSON("COLONY_STATE.json", colony.ColonyState{
		Version: "1.0",
		Goal:    &goal,
		State:   colony.StateREADY,
	}); err != nil {
		t.Fatal(err)
	}

	const sentinel = "SENTINEL-198-2-01-STEERING-PLAN"
	if _, _, err := writePheromoneSignal("FOCUS", sentinel, "", "test", "", "", 0.9, nil); err != nil {
		t.Fatalf("seed steering note via writePheromoneSignal: %v", err)
	}

	result, err := runCodexPlanWithOptions(tmpDir, codexPlanOptions{PlanOnly: true})
	if err != nil {
		t.Fatalf("runCodexPlanWithOptions(plan-only): %v", err)
	}
	manifest, ok := result["plan_manifest"].(codexPlanManifest)
	if !ok {
		t.Fatalf("result[plan_manifest] is not a codexPlanManifest: %#v", result["plan_manifest"])
	}
	if !strings.Contains(manifest.ContextCapsule, sentinel) {
		t.Fatalf("plan_manifest.context_capsule does not carry the seeded steering note %q:\n%s", sentinel, manifest.ContextCapsule)
	}

	t.Run("memory_free_colony_omits_capsule", func(t *testing.T) {
		saveGlobalsCmd(t)
		s2, tmpDir2 := newTestStoreCmd(t)
		defer os.RemoveAll(tmpDir2)
		store = s2

		goal2 := "198.2-01 memory-free colony"
		if err := s2.SaveJSON("COLONY_STATE.json", colony.ColonyState{
			Version: "1.0",
			Goal:    &goal2,
			State:   colony.StateREADY,
		}); err != nil {
			t.Fatal(err)
		}

		result2, err := runCodexPlanWithOptions(tmpDir2, codexPlanOptions{PlanOnly: true})
		if err != nil {
			t.Fatalf("runCodexPlanWithOptions(plan-only, memory-free): %v", err)
		}
		manifest2, ok := result2["plan_manifest"].(codexPlanManifest)
		if !ok {
			t.Fatalf("result[plan_manifest] is not a codexPlanManifest: %#v", result2["plan_manifest"])
		}
		if manifest2.ContextCapsule != "" {
			t.Fatalf("memory-free colony: manifest.ContextCapsule = %q, want empty", manifest2.ContextCapsule)
		}

		data, err := json.Marshal(manifest2)
		if err != nil {
			t.Fatalf("marshal manifest: %v", err)
		}
		if strings.Contains(string(data), `"context_capsule"`) {
			t.Fatalf("memory-free colony: manifest JSON still carries the context_capsule key (omitempty broken): %s", data)
		}
	})

	t.Run("hosted_path_leaves_capsule_empty", func(t *testing.T) {
		saveGlobalsCmd(t)
		s3, tmpDir3 := newTestStoreCmd(t)
		defer os.RemoveAll(tmpDir3)
		store = s3

		goal3 := "198.2-01 hosted path colony"
		if err := s3.SaveJSON("COLONY_STATE.json", colony.ColonyState{
			Version: "1.0",
			Goal:    &goal3,
			State:   colony.StateREADY,
		}); err != nil {
			t.Fatal(err)
		}

		result3, err := runCodexPlanWithOptions(tmpDir3, codexPlanOptions{Synthetic: true})
		if err != nil {
			t.Fatalf("runCodexPlanWithOptions(hosted/synthetic): %v", err)
		}
		// The hosted (non-plan-only, non-delegate) path activates the plan
		// directly and never exposes a codexPlanManifest to the caller at
		// all -- "the finalize record does not deliver prompts" (see the
		// ContextCapsule doc comment on codexPlanManifest). It cannot leak
		// a capsule through this result map because nothing here carries
		// one; assert that no key or nested value anywhere in the result
		// contains a context_capsule payload, so a future refactor that
		// starts exposing a manifest on this path cannot silently start
		// carrying the capsule too.
		data, err := json.Marshal(result3)
		if err != nil {
			t.Fatalf("marshal hosted-path result: %v", err)
		}
		if strings.Contains(string(data), `"context_capsule"`) {
			t.Fatalf("hosted path result unexpectedly carries a context_capsule key -- the hosted/native path shares its own capsule via codex.WorkerDispatch.ContextCapsule per-dispatch, and must not also deliver it via a manifest field: %s", data)
		}
	})
}

// TestColonizeDelegateManifestCarriesOneSteeringNote is the colonize twin of
// TestPlanDelegateManifestCarriesOneSteeringNote (WIRE-01/D-03): a
// --plan-only colonize manifest run on a colony carrying one active owner
// steering note must carry that note's exact text inside
// colonize_manifest.context_capsule, reaching all four surveyors. It also
// pins the memory-free boundary case: an empty colony omits the field
// entirely (omitempty), not as an empty string.
func TestColonizeDelegateManifestCarriesOneSteeringNote(t *testing.T) {
	saveGlobalsCmd(t)

	s, tmpDir := newTestStoreCmd(t)
	defer os.RemoveAll(tmpDir)
	store = s

	goal := "198.2-01 colonize delegate capsule test colony"
	if err := s.SaveJSON("COLONY_STATE.json", colony.ColonyState{
		Version: "1.0",
		Goal:    &goal,
		State:   colony.StateREADY,
	}); err != nil {
		t.Fatal(err)
	}

	const sentinel = "SENTINEL-198-2-01-STEERING-COLONIZE"
	if _, _, err := writePheromoneSignal("FOCUS", sentinel, "", "test", "", "", 0.9, nil); err != nil {
		t.Fatalf("seed steering note via writePheromoneSignal: %v", err)
	}

	result, err := runCodexColonizeWithOptions(tmpDir, codexColonizeOptions{PlanOnly: true})
	if err != nil {
		t.Fatalf("runCodexColonizeWithOptions(plan-only): %v", err)
	}
	manifest, ok := result["colonize_manifest"].(codexColonizeManifest)
	if !ok {
		t.Fatalf("result[colonize_manifest] is not a codexColonizeManifest: %#v", result["colonize_manifest"])
	}
	if !strings.Contains(manifest.ContextCapsule, sentinel) {
		t.Fatalf("colonize_manifest.context_capsule does not carry the seeded steering note %q:\n%s", sentinel, manifest.ContextCapsule)
	}
	if len(manifest.Dispatches) == 0 {
		t.Fatal("fixture broken: expected non-empty surveyor dispatches on the colonize plan-only manifest")
	}

	t.Run("memory_free_colony_omits_capsule", func(t *testing.T) {
		saveGlobalsCmd(t)
		s2, tmpDir2 := newTestStoreCmd(t)
		defer os.RemoveAll(tmpDir2)
		store = s2

		goal2 := "198.2-01 colonize memory-free colony"
		if err := s2.SaveJSON("COLONY_STATE.json", colony.ColonyState{
			Version: "1.0",
			Goal:    &goal2,
			State:   colony.StateREADY,
		}); err != nil {
			t.Fatal(err)
		}

		result2, err := runCodexColonizeWithOptions(tmpDir2, codexColonizeOptions{PlanOnly: true})
		if err != nil {
			t.Fatalf("runCodexColonizeWithOptions(plan-only, memory-free): %v", err)
		}
		manifest2, ok := result2["colonize_manifest"].(codexColonizeManifest)
		if !ok {
			t.Fatalf("result[colonize_manifest] is not a codexColonizeManifest: %#v", result2["colonize_manifest"])
		}
		if manifest2.ContextCapsule != "" {
			t.Fatalf("memory-free colony: manifest.ContextCapsule = %q, want empty", manifest2.ContextCapsule)
		}

		data, err := json.Marshal(manifest2)
		if err != nil {
			t.Fatalf("marshal manifest: %v", err)
		}
		if strings.Contains(string(data), `"context_capsule"`) {
			t.Fatalf("memory-free colony: manifest JSON still carries the context_capsule key (omitempty broken): %s", data)
		}
	})
}

// TestPlanAndColonizeWrappersAreByteIdentical is the three-way parity test
// named in 198.2-01 Task 2: all three hand-maintained copies of the plan
// wrapper (canonical nested Claude, flat installed-consumer Claude mirror,
// OpenCode) must stay byte-identical to each other, and likewise for
// colonize -- and the capsule pass-through instruction added in this plan
// must be present in every one of the six wrapper files plus both YAML
// sources, so a two-copy edit fails by name and reports which copy is
// missing it. This reuses the exact triplet_is_byte_identical shape
// TestLifecycleWrappersCarryCoherentJobContract already asserts for build.md.
func TestPlanAndColonizeWrappersAreByteIdentical(t *testing.T) {
	repoRoot, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("failed to find repo root: %v", err)
	}

	const capsuleMarker = "context_capsule"

	cases := []struct {
		verb  string
		paths []string
	}{
		{"plan", planWrapperTripletPaths(repoRoot)},
		{"colonize", colonizeWrapperTripletPaths(repoRoot)},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.verb, func(t *testing.T) {
			bodies := make([][]byte, 0, len(tc.paths))
			for _, path := range tc.paths {
				content, err := os.ReadFile(path)
				if err != nil {
					t.Fatalf("read %s: %v", path, err)
				}
				bodies = append(bodies, content)
				if !bytes.Contains(content, []byte(capsuleMarker)) {
					t.Errorf("%s: missing the %q pass-through instruction -- a two-copy edit ships the change to nobody", path, capsuleMarker)
				}
			}
			for i := 1; i < len(bodies); i++ {
				if !bytes.Equal(bodies[0], bodies[i]) {
					t.Errorf(
						"%s wrapper copies drifted: %s (%d bytes) != %s (%d bytes) -- the three copies are hand-maintained and must be byte-identical",
						tc.verb, tc.paths[0], len(bodies[0]), tc.paths[i], len(bodies[i]),
					)
				}
			}
		})
	}

	yamlPaths := map[string]string{
		"plan.yaml":     filepath.Join(repoRoot, ".aether", "commands", "plan.yaml"),
		"colonize.yaml": filepath.Join(repoRoot, ".aether", "commands", "colonize.yaml"),
	}
	for name, path := range yamlPaths {
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		if !bytes.Contains(content, []byte(capsuleMarker)) {
			t.Errorf("%s: missing the %q pass-through instruction -- the YAML source must not drift from the wrapper copies it corresponds to", name, capsuleMarker)
		}
	}
}
