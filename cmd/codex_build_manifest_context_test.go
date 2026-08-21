package cmd

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

// TestBuildManifestCarriesContextCapsuleOnce pins CONTEXT-02/CONTEXT-03: the
// colony-prime context capsule must reach the plan-only build manifest,
// computed once, at the manifest's top level — never duplicated into every
// dispatch's brief. A reader of a failure from this test should conclude
// either that the capsule stopped reaching the wrapper-manifest path
// (CONTEXT-02 regressed) or that it started leaking into per-dispatch briefs
// again (CONTEXT-03 regressed).
func TestBuildManifestCarriesContextCapsuleOnce(t *testing.T) {
	saveGlobalsCmd(t)

	s, tmpDir := newTestStoreCmd(t)
	defer os.RemoveAll(tmpDir)
	store = s

	now := time.Now().Format(time.RFC3339)
	goal := "Context capsule manifest test colony"

	taskID1 := "1.1"
	taskID2 := "1.2"
	phase := colony.Phase{
		ID:     1,
		Name:   "Manifest Capsule Phase",
		Status: colony.PhaseReady,
		Tasks: []colony.Task{
			{ID: &taskID1, Goal: "First task", Status: colony.TaskPending},
			{ID: &taskID2, Goal: "Second task", Status: colony.TaskPending},
		},
	}

	// Enough memory (decisions + phase learnings) that resolveCodexWorkerContext()
	// clears its 128-char minimum threshold.
	state := colony.ColonyState{
		Version:      "1.0",
		Goal:         &goal,
		State:        colony.StateEXECUTING,
		CurrentPhase: 1,
		Plan: colony.Plan{
			Phases: []colony.Phase{phase},
		},
		Memory: colony.Memory{
			Decisions: []colony.Decision{
				{ID: "d1", Phase: 1, Claim: "Use manifest-level capsule carriage", Rationale: "Avoid per-dispatch duplication", Timestamp: now},
			},
			PhaseLearnings: []colony.PhaseLearning{
				{
					ID: "pl1", Phase: 1, PhaseName: "Manifest Capsule Phase", Timestamp: now,
					Learnings: []colony.Learning{
						{Claim: "Capsule must be computed once per manifest", Status: "validated", Tested: true},
					},
				},
			},
		},
	}
	if err := s.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatal(err)
	}

	reviewDepth := colony.VerificationDepthStandard
	dispatches := plannedBuildDispatchesForSelectionWithState(phase, state, nil, reviewDepth)
	if len(dispatches) < 2 {
		t.Fatalf("expected 2+ planned dispatches from a 2-task phase, got %d", len(dispatches))
	}
	for i := range dispatches {
		dispatches[i].Status = "planned"
	}
	dispatches, err := ensureUniqueBuildDispatchNames(dispatches, phase.ID)
	if err != nil {
		t.Fatalf("ensureUniqueBuildDispatchNames: %v", err)
	}
	generatedAt := time.Now()
	attachBuildDispatchContext(tmpDir, phase, dispatches, generatedAt)

	// --- Test 1 + 2 + 3: planOnly=true manifest ---
	manifest := buildCodexBuildManifest(tmpDir, state, phase, "", "", dispatches, generatedAt, "plan-only", nil, nil, true, reviewDepth)

	if strings.TrimSpace(manifest.ContextCapsule) == "" {
		t.Fatal("CONTEXT-02 regressed: plan-only manifest.ContextCapsule is empty on a populated colony — the wrapper-spawned worker would receive no colony-prime grounding")
	}

	// Derive the marker from the manifest value itself, not a hardcoded
	// heading string, so a heading rename can't make this test pass while the
	// content underneath silently disappears.
	capsuleMarker := strings.TrimSpace(strings.SplitN(manifest.ContextCapsule, "\n", 2)[0])
	if len(capsuleMarker) < 8 {
		t.Fatalf("capsule marker too short to be a reliable uniqueness probe: %q", capsuleMarker)
	}

	manifestJSON, err := json.Marshal(manifest)
	if err != nil {
		t.Fatalf("marshal manifest: %v", err)
	}

	occurrences := strings.Count(string(manifestJSON), capsuleMarker)
	if occurrences != 1 {
		t.Fatalf("CONTEXT-03 regressed: capsule marker %q appears %d time(s) in the plan-only manifest JSON, want exactly 1 — the capsule is either missing from the manifest field or duplicated into dispatch briefs", capsuleMarker, occurrences)
	}

	for i, dispatch := range dispatches {
		if strings.Contains(dispatch.Brief, capsuleMarker) {
			t.Fatalf("CONTEXT-03 regressed: dispatch[%d] (%s) brief contains the context capsule marker — the capsule must be carried once at manifest level, not copied into every dispatch brief", i, dispatch.Name)
		}
	}

	// --- Test 4: planOnly=false manifest omits the capsule ---
	hostedManifest := buildCodexBuildManifest(tmpDir, state, phase, "", "", dispatches, generatedAt, "hosted", nil, nil, false, reviewDepth)
	if hostedManifest.ContextCapsule != "" {
		t.Fatalf("hosted (planOnly=false) manifest.ContextCapsule = %q, want empty — Path A (executeCodexBuildDispatches) already delivers the capsule via WorkerDispatch.ContextCapsule, so the manifest must not duplicate it here", hostedManifest.ContextCapsule)
	}
}
