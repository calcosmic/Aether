package cmd

import (
	"encoding/json"
	"os"
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
