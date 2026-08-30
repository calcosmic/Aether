package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/storage"
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

// memorySourceSeeder198_2 seeds one of the seven memory sources named in
// WIRE-01/198.2-CONTEXT.md through the runtime's own writer -- never a
// hand-typed JSON literal -- carrying the given sentinel string.
type memorySourceSeeder198_2 struct {
	name string
	seed func(t *testing.T, s *storage.Store, hubDir, sentinel string)
}

var memorySeeders198_2 = []memorySourceSeeder198_2{
	{
		name: "queen_file",
		seed: func(t *testing.T, s *storage.Store, hubDir, sentinel string) {
			t.Helper()
			t.Setenv("AETHER_HUB_DIR", hubDir)
			if err := preferencesCmd.RunE(preferencesCmd, []string{sentinel}); err != nil {
				t.Fatalf("seed queen-file preference via preferencesCmd: %v", err)
			}
		},
	},
	{
		name: "hive_entry",
		seed: func(t *testing.T, s *storage.Store, hubDir, sentinel string) {
			t.Helper()
			t.Setenv("AETHER_HUB_DIR", hubDir)
			content := sentinel + " -- cmd/wire_capsule_198_2_test.go"
			if err := hiveStoreCmd.RunE(hiveStoreCmd, []string{content, "general", "test-repo-198-2-01"}); err != nil {
				t.Fatalf("seed hive entry via hiveStoreCmd: %v", err)
			}
		},
	},
	{
		name: "learned_habit",
		seed: func(t *testing.T, s *storage.Store, hubDir, sentinel string) {
			t.Helper()
			content := sentinel + " -- cmd/wire_capsule_198_2_test.go"
			promoteRealInstinct(t, s, content, "success_pattern")
		},
	},
	{
		name: "failure_log",
		seed: func(t *testing.T, s *storage.Store, hubDir, sentinel string) {
			t.Helper()
			if err := appendMiddenEntry(s, "test_category", "test", sentinel, nil); err != nil {
				t.Fatalf("seed failure-log entry via appendMiddenEntry: %v", err)
			}
		},
	},
	{
		name: "steering_note",
		seed: func(t *testing.T, s *storage.Store, hubDir, sentinel string) {
			t.Helper()
			if _, _, err := writePheromoneSignal("FOCUS", sentinel, "", "test", "", "", 0.9, nil); err != nil {
				t.Fatalf("seed steering note via writePheromoneSignal: %v", err)
			}
		},
	},
	{
		name: "owner_answer",
		seed: func(t *testing.T, s *storage.Store, hubDir, sentinel string) {
			t.Helper()
			if _, err := recordDecisionAnswer("What should the worker do?", sentinel, 1, "test"); err != nil {
				t.Fatalf("seed owner answer via recordDecisionAnswer: %v", err)
			}
		},
	},
	{
		name: "relay_note",
		seed: func(t *testing.T, s *storage.Store, hubDir, sentinel string) {
			t.Helper()
			seedRelayNote198_2(t, sentinel)
		},
	},
}

// newSeededColony198_2 creates an isolated store + colony state + hub
// directory for one (command, source) subtest, with CurrentPhase=1 so the
// relay-note seeder's phase-scoped handoff render matches.
func newSeededColony198_2(t *testing.T) (s *storage.Store, tmpDir, hubDir string) {
	t.Helper()
	s, tmpDir = newTestStoreCmd(t)
	t.Cleanup(func() { os.RemoveAll(tmpDir) })
	store = s

	goal := "198.2-01 every-memory-source test colony"
	if err := s.SaveJSON("COLONY_STATE.json", colony.ColonyState{
		Version:      "1.0",
		Goal:         &goal,
		State:        colony.StateREADY,
		CurrentPhase: 1,
	}); err != nil {
		t.Fatal(err)
	}

	hubDir = filepath.Join(t.TempDir(), "hub")
	return s, tmpDir, hubDir
}

// seedRelayNote198_2 persists one worker handoff carrying sentinel in
// NextWorkerInstructions, on the "build" workflow at phase 1 (matching
// newSeededColony198_2's CurrentPhase), the same way persistDispatchWorkerHandoff
// is called at the end of a real build dispatch.
func seedRelayNote198_2(t *testing.T, sentinel string) {
	t.Helper()
	dispatch := codex.WorkerDispatch{
		WorkerName: "Sentinel-Worker",
		Caste:      "builder",
		TaskID:     "1.1",
		Workflow:   "build",
		Phase:      1,
		Wave:       1,
	}
	result := codex.DispatchResult{
		WorkerName: "Sentinel-Worker",
		Status:     "completed",
		WorkerResult: &codex.WorkerResult{
			WorkerName: "Sentinel-Worker",
			Caste:      "builder",
			TaskID:     "1.1",
			Status:     "completed",
			Summary:    "seeded relay note for 198.2-01",
			Handoff: codex.WorkerHandoff{
				ChangedFiles:           []string{"cmd/seeded.go"},
				VerificationStatus:     "pass",
				NextWorkerInstructions: []string{sentinel},
				Freshness:              time.Now().UTC().Format(time.RFC3339),
			},
		},
	}
	if err := persistDispatchWorkerHandoff(dispatch, result); err != nil {
		t.Fatalf("seed relay note via persistDispatchWorkerHandoff: %v", err)
	}
}

// TestPlanAndColonizeDelegateLanesCarryEveryMemorySource is the WIRE-01
// breadth proof: a table test with one row per command (plan, colonize)
// crossed with one row per memory source (7), each seeded through the
// runtime's own writer and asserted present in that command's delegate-lane
// manifest capsule. Every subtest runs the plan-only/delegate path -- never
// only the in-process dispatch (see the package note above).
func TestPlanAndColonizeDelegateLanesCarryEveryMemorySource(t *testing.T) {
	commands := []struct {
		name    string
		capsule func(t *testing.T, tmpDir string) string
	}{
		{
			name: "plan",
			capsule: func(t *testing.T, tmpDir string) string {
				t.Helper()
				result, err := runCodexPlanWithOptions(tmpDir, codexPlanOptions{PlanOnly: true})
				if err != nil {
					t.Fatalf("runCodexPlanWithOptions(plan-only): %v", err)
				}
				manifest, ok := result["plan_manifest"].(codexPlanManifest)
				if !ok {
					t.Fatalf("result[plan_manifest] is not a codexPlanManifest: %#v", result["plan_manifest"])
				}
				return manifest.ContextCapsule
			},
		},
		{
			name: "colonize",
			capsule: func(t *testing.T, tmpDir string) string {
				t.Helper()
				result, err := runCodexColonizeWithOptions(tmpDir, codexColonizeOptions{PlanOnly: true})
				if err != nil {
					t.Fatalf("runCodexColonizeWithOptions(plan-only): %v", err)
				}
				manifest, ok := result["colonize_manifest"].(codexColonizeManifest)
				if !ok {
					t.Fatalf("result[colonize_manifest] is not a codexColonizeManifest: %#v", result["colonize_manifest"])
				}
				return manifest.ContextCapsule
			},
		},
	}

	for _, cmdCase := range commands {
		cmdCase := cmdCase
		for _, source := range memorySeeders198_2 {
			source := source
			t.Run(cmdCase.name+"/"+source.name, func(t *testing.T) {
				saveGlobalsCmd(t)
				_, tmpDir, hubDir := newSeededColony198_2(t)

				sentinel := "SENTINEL-198-2-01-" + strings.ToUpper(source.name) + "-" + strings.ToUpper(cmdCase.name)
				source.seed(t, store, hubDir, sentinel)

				capsule := cmdCase.capsule(t, tmpDir)
				if !strings.Contains(capsule, sentinel) {
					t.Fatalf("the %s did not reach the %s's prompt -- capsule:\n%s", strings.ReplaceAll(source.name, "_", " "), cmdCase.name, capsule)
				}
			})
		}
	}
}

// TestPlanAndColonizeCapsulesMatchTheInProcessLane pins D-03: for the same
// seeded colony, the capsule the delegate/plan-only manifest carries and the
// capsule the in-process lane sets on its own worker dispatch come from one
// call to the same assembler with the same arguments, so they must be
// byte-identical.
func TestPlanAndColonizeCapsulesMatchTheInProcessLane(t *testing.T) {
	t.Run("plan", func(t *testing.T) {
		saveGlobalsCmd(t)
		_, tmpDir, hubDir := newSeededColony198_2(t)
		sentinel := "SENTINEL-198-2-01-D03-PLAN"
		t.Setenv("AETHER_HUB_DIR", hubDir)
		if _, _, err := writePheromoneSignal("FOCUS", sentinel, "", "test", "", "", 0.9, nil); err != nil {
			t.Fatalf("seed steering note: %v", err)
		}

		result, err := runCodexPlanWithOptions(tmpDir, codexPlanOptions{PlanOnly: true})
		if err != nil {
			t.Fatalf("runCodexPlanWithOptions(plan-only): %v", err)
		}
		manifest, ok := result["plan_manifest"].(codexPlanManifest)
		if !ok {
			t.Fatalf("result[plan_manifest] is not a codexPlanManifest: %#v", result["plan_manifest"])
		}

		spy := &pheromoneCaptureInvoker190_05{}
		if _, err := dispatchRealPlanningWorkersWithIterationContext(context.Background(), tmpDir, codexSurveyContext{}, spy, 0, ""); err != nil {
			t.Fatalf("dispatchRealPlanningWorkersWithIterationContext: %v", err)
		}
		configs := spy.captured()
		if len(configs) == 0 {
			t.Fatal("expected at least one captured planning worker config")
		}

		if !strings.Contains(manifest.ContextCapsule, sentinel) || !strings.Contains(configs[0].ContextCapsule, sentinel) {
			t.Fatalf("fixture broken: seeded sentinel missing from one or both lanes -- delegate=%q in-process=%q", manifest.ContextCapsule, configs[0].ContextCapsule)
		}
		if manifest.ContextCapsule != configs[0].ContextCapsule {
			t.Fatalf("D-03 regressed: plan delegate-manifest capsule and in-process capsule differ:\ndelegate:\n%s\n---\nin-process:\n%s", manifest.ContextCapsule, configs[0].ContextCapsule)
		}
	})

	t.Run("colonize", func(t *testing.T) {
		saveGlobalsCmd(t)
		_, tmpDir, hubDir := newSeededColony198_2(t)
		sentinel := "SENTINEL-198-2-01-D03-COLONIZE"
		t.Setenv("AETHER_HUB_DIR", hubDir)
		if _, _, err := writePheromoneSignal("FOCUS", sentinel, "", "test", "", "", 0.9, nil); err != nil {
			t.Fatalf("seed steering note: %v", err)
		}

		result, err := runCodexColonizeWithOptions(tmpDir, codexColonizeOptions{PlanOnly: true})
		if err != nil {
			t.Fatalf("runCodexColonizeWithOptions(plan-only): %v", err)
		}
		manifest, ok := result["colonize_manifest"].(codexColonizeManifest)
		if !ok {
			t.Fatalf("result[colonize_manifest] is not a codexColonizeManifest: %#v", result["colonize_manifest"])
		}

		spy := &pheromoneCaptureInvoker190_05{}
		if _, err := dispatchRealSurveyorsWithTimeout(context.Background(), tmpDir, spy, 0); err != nil {
			t.Fatalf("dispatchRealSurveyorsWithTimeout: %v", err)
		}
		configs := spy.captured()
		if len(configs) == 0 {
			t.Fatal("expected at least one captured surveyor worker config")
		}

		if !strings.Contains(manifest.ContextCapsule, sentinel) || !strings.Contains(configs[0].ContextCapsule, sentinel) {
			t.Fatalf("fixture broken: seeded sentinel missing from one or both lanes -- delegate=%q in-process=%q", manifest.ContextCapsule, configs[0].ContextCapsule)
		}
		if manifest.ContextCapsule != configs[0].ContextCapsule {
			t.Fatalf("D-03 regressed: colonize delegate-manifest capsule and in-process capsule differ:\ndelegate:\n%s\n---\nin-process:\n%s", manifest.ContextCapsule, configs[0].ContextCapsule)
		}
	})
}

// TestDelegateCapsuleRendersSteeringAndRelayExactlyOnce counts occurrences of
// the steering-notes heading ("## Pheromone Signals") and the
// previous-helper-relay heading ("## Previous Worker Handoffs") across the
// full assembled worker prompt on the delegate lane -- the manifest capsule
// plus every per-dispatch field the wrapper is told to append (brief,
// skill_section; neither codexPlanningDispatch nor codexSurveyorDispatch
// carries an independent pheromone/handoff field) -- and fails if either
// heading appears more than once.
func TestDelegateCapsuleRendersSteeringAndRelayExactlyOnce(t *testing.T) {
	const pheromoneHeading = "## Pheromone Signals"
	const handoffHeading = "## Previous Worker Handoffs"

	t.Run("plan", func(t *testing.T) {
		saveGlobalsCmd(t)
		_, tmpDir, hubDir := newSeededColony198_2(t)
		steeringSentinel := "SENTINEL-198-2-01-ONCE-STEERING-PLAN"
		relaySentinel := "SENTINEL-198-2-01-ONCE-RELAY-PLAN"
		t.Setenv("AETHER_HUB_DIR", hubDir)
		if _, _, err := writePheromoneSignal("FOCUS", steeringSentinel, "", "test", "", "", 0.9, nil); err != nil {
			t.Fatalf("seed steering note: %v", err)
		}
		seedRelayNote198_2(t, relaySentinel)

		result, err := runCodexPlanWithOptions(tmpDir, codexPlanOptions{PlanOnly: true})
		if err != nil {
			t.Fatalf("runCodexPlanWithOptions(plan-only): %v", err)
		}
		manifest, ok := result["plan_manifest"].(codexPlanManifest)
		if !ok {
			t.Fatalf("result[plan_manifest] is not a codexPlanManifest: %#v", result["plan_manifest"])
		}
		if len(manifest.Dispatches) == 0 {
			t.Fatal("fixture broken: expected non-empty planning dispatches")
		}
		assembled := manifest.ContextCapsule + manifest.Dispatches[0].Brief + manifest.Dispatches[0].SkillSection
		if !strings.Contains(assembled, steeringSentinel) || !strings.Contains(assembled, relaySentinel) {
			t.Fatalf("fixture broken: seeded sentinels missing from assembled prompt:\n%s", assembled)
		}
		if n := strings.Count(assembled, pheromoneHeading); n != 1 {
			t.Errorf("plan: %q heading appears %d times in the assembled prompt, want exactly 1", pheromoneHeading, n)
		}
		if n := strings.Count(assembled, handoffHeading); n != 1 {
			t.Errorf("plan: %q heading appears %d times in the assembled prompt, want exactly 1", handoffHeading, n)
		}
	})

	t.Run("colonize", func(t *testing.T) {
		saveGlobalsCmd(t)
		_, tmpDir, hubDir := newSeededColony198_2(t)
		steeringSentinel := "SENTINEL-198-2-01-ONCE-STEERING-COLONIZE"
		relaySentinel := "SENTINEL-198-2-01-ONCE-RELAY-COLONIZE"
		t.Setenv("AETHER_HUB_DIR", hubDir)
		if _, _, err := writePheromoneSignal("FOCUS", steeringSentinel, "", "test", "", "", 0.9, nil); err != nil {
			t.Fatalf("seed steering note: %v", err)
		}
		seedRelayNote198_2(t, relaySentinel)

		result, err := runCodexColonizeWithOptions(tmpDir, codexColonizeOptions{PlanOnly: true})
		if err != nil {
			t.Fatalf("runCodexColonizeWithOptions(plan-only): %v", err)
		}
		manifest, ok := result["colonize_manifest"].(codexColonizeManifest)
		if !ok {
			t.Fatalf("result[colonize_manifest] is not a codexColonizeManifest: %#v", result["colonize_manifest"])
		}
		if len(manifest.Dispatches) == 0 {
			t.Fatal("fixture broken: expected non-empty surveyor dispatches")
		}
		assembled := manifest.ContextCapsule + manifest.Dispatches[0].Brief + manifest.Dispatches[0].SkillSection
		if !strings.Contains(assembled, steeringSentinel) || !strings.Contains(assembled, relaySentinel) {
			t.Fatalf("fixture broken: seeded sentinels missing from assembled prompt:\n%s", assembled)
		}
		if n := strings.Count(assembled, pheromoneHeading); n != 1 {
			t.Errorf("colonize: %q heading appears %d times in the assembled prompt, want exactly 1", pheromoneHeading, n)
		}
		if n := strings.Count(assembled, handoffHeading); n != 1 {
			t.Errorf("colonize: %q heading appears %d times in the assembled prompt, want exactly 1", handoffHeading, n)
		}
	})
}

// TestDelegateCapsuleIsStableAcrossRuns runs the plan-only path twice against
// unchanged colony state and asserts the two capsules are byte-identical, so
// equal-ranked capsule sections never reorder between runs.
func TestDelegateCapsuleIsStableAcrossRuns(t *testing.T) {
	t.Run("plan", func(t *testing.T) {
		saveGlobalsCmd(t)
		_, tmpDir, hubDir := newSeededColony198_2(t)
		t.Setenv("AETHER_HUB_DIR", hubDir)
		if _, _, err := writePheromoneSignal("FOCUS", "SENTINEL-198-2-01-STABLE-PLAN", "", "test", "", "", 0.9, nil); err != nil {
			t.Fatalf("seed steering note: %v", err)
		}

		first, err := runCodexPlanWithOptions(tmpDir, codexPlanOptions{PlanOnly: true})
		if err != nil {
			t.Fatalf("runCodexPlanWithOptions(plan-only) first run: %v", err)
		}
		second, err := runCodexPlanWithOptions(tmpDir, codexPlanOptions{PlanOnly: true})
		if err != nil {
			t.Fatalf("runCodexPlanWithOptions(plan-only) second run: %v", err)
		}
		firstManifest := first["plan_manifest"].(codexPlanManifest)
		secondManifest := second["plan_manifest"].(codexPlanManifest)
		if firstManifest.ContextCapsule == "" {
			t.Fatal("fixture broken: first run's capsule is empty")
		}
		if firstManifest.ContextCapsule != secondManifest.ContextCapsule {
			t.Fatalf("plan capsule is not stable across runs:\nfirst:\n%s\n---\nsecond:\n%s", firstManifest.ContextCapsule, secondManifest.ContextCapsule)
		}
	})

	t.Run("colonize", func(t *testing.T) {
		saveGlobalsCmd(t)
		_, tmpDir, hubDir := newSeededColony198_2(t)
		t.Setenv("AETHER_HUB_DIR", hubDir)
		if _, _, err := writePheromoneSignal("FOCUS", "SENTINEL-198-2-01-STABLE-COLONIZE", "", "test", "", "", 0.9, nil); err != nil {
			t.Fatalf("seed steering note: %v", err)
		}

		first, err := runCodexColonizeWithOptions(tmpDir, codexColonizeOptions{PlanOnly: true})
		if err != nil {
			t.Fatalf("runCodexColonizeWithOptions(plan-only) first run: %v", err)
		}
		second, err := runCodexColonizeWithOptions(tmpDir, codexColonizeOptions{PlanOnly: true})
		if err != nil {
			t.Fatalf("runCodexColonizeWithOptions(plan-only) second run: %v", err)
		}
		firstManifest := first["colonize_manifest"].(codexColonizeManifest)
		secondManifest := second["colonize_manifest"].(codexColonizeManifest)
		if firstManifest.ContextCapsule == "" {
			t.Fatal("fixture broken: first run's capsule is empty")
		}
		if firstManifest.ContextCapsule != secondManifest.ContextCapsule {
			t.Fatalf("colonize capsule is not stable across runs:\nfirst:\n%s\n---\nsecond:\n%s", firstManifest.ContextCapsule, secondManifest.ContextCapsule)
		}
	})
}
