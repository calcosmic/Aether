package cmd

import (
	"encoding/json"
	"testing"
)

// TestWrapperManifestCannotHandItselfTaskCredit is the permanent regression
// lock for the 195 review's third critical finding (CR-03).
//
// Which tasks a build gets credit for is decided by the runtime, from evidence
// it checked itself. Phase 195 gave the dispatch object two serialized fields
// carrying exactly that verdict -- the credited task list and its per-task
// claims -- and the completion packet a build wrapper submits carries its own
// copy of the dispatch manifest. On the legacy path (a packet with no attempt
// binding, which the runtime accepts with a warning rather than a refusal),
// that manifest is entirely wrapper-shaped. A manifest that simply asserted
// "these three tasks are done" therefore credited three tasks with zero
// receipts, zero evidence, and marked the phase built.
//
// The rule this locks: those two fields are runtime-owned in-process state,
// never wire contract. Nothing arriving from outside may set them.
func TestWrapperManifestCannotHandItselfTaskCredit(t *testing.T) {
	// 1. The wire boundary: a dispatch decoded from wrapper-authored JSON must
	//    carry no credit, whatever the JSON says.
	raw := []byte(`{
		"name": "Mason-1",
		"caste": "builder",
		"stage": "wave",
		"task_id": "1.1",
		"covered_task_ids": ["1.1", "1.2", "1.3"],
		"status": "planned",
		"completed_task_ids": ["1.1", "1.2", "1.3"],
		"task_claims": [{"task_id": "1.1", "files_modified": ["cmd/invented.go"]}]
	}`)
	var decoded codexBuildDispatch
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("decode dispatch: %v", err)
	}
	if len(decoded.CompletedTaskIDs) != 0 {
		t.Fatalf("a wrapper-authored dispatch decoded straight into runtime-owned task credit %v", decoded.CompletedTaskIDs)
	}
	if len(decoded.TaskClaims) != 0 {
		t.Fatalf("a wrapper-authored dispatch decoded straight into runtime-owned task claims %+v", decoded.TaskClaims)
	}

	// 2. The merge boundary: even an in-memory manifest that somehow carries
	//    the fields must not survive into the credited set.
	manifest := codexBuildManifest{
		PlanOnly: true,
		Dispatches: []codexBuildDispatch{{
			Name:             "Mason-1",
			Caste:            "builder",
			Stage:            "wave",
			TaskID:           "1.1",
			CoveredTaskIDs:   []string{"1.1", "1.2", "1.3"},
			Status:           "planned",
			CompletedTaskIDs: []string{"1.1", "1.2", "1.3"},
			TaskClaims:       []codexBuildTaskClaim{{TaskID: "1.1", FilesModified: []string{"cmd/invented.go"}}},
		}},
	}
	results := []codexExternalBuildWorkerResult{{
		Name:    "Mason-1",
		Status:  "failed",
		Summary: "ran out of time",
	}}

	dispatches, _, err := mergeExternalBuildResults(manifest, results)
	if err != nil {
		t.Fatalf("mergeExternalBuildResults: %v", err)
	}
	credited := completedBuildTaskIDs(dispatches)
	if len(credited) != 0 {
		t.Fatalf("a manifest that simply asserted its own completion credited %d task(s) with no receipt and no evidence: %v", len(credited), credited)
	}
	for _, d := range dispatches {
		if len(d.TaskClaims) != 0 {
			t.Fatalf("wrapper-authored task claims survived the merge boundary: %+v", d.TaskClaims)
		}
	}
	if allSelectedBuildTasksCredited([]string{"1.1", "1.2", "1.3"}, dispatches) {
		t.Fatal("a self-asserting manifest was enough to report every task credited, which is what marks the phase built")
	}
}

// TestRuntimeResolvedTaskCreditStillReachesTheCreditedSet proves the CR-03 fix
// only closes the wire door: credit the runtime itself resolves, in process,
// from checked evidence still flows through to the credited set exactly as
// before.
func TestRuntimeResolvedTaskCreditStillReachesTheCreditedSet(t *testing.T) {
	dispatches := []codexBuildDispatch{{
		Name:             "Mason-1",
		Status:           "failed",
		CoveredTaskIDs:   []string{"1.1", "1.2"},
		CompletedTaskIDs: []string{"1.1"},
	}}
	credited := completedBuildTaskIDs(dispatches)
	if _, ok := credited["1.1"]; !ok {
		t.Fatalf("runtime-resolved credit for 1.1 was lost: %v", credited)
	}
	if _, ok := credited["1.2"]; ok {
		t.Fatalf("uncredited task 1.2 was credited: %v", credited)
	}
}

// TestBuildResultStillReportsWhatTheRuntimeCredited closes the other half of
// CR-03: taking the credit fields off the wire must not take them off the
// SCREEN. Three build wrapper copies, the command guide and the build-cycle
// skill all instruct the wrapper to read each dispatch's `completed_task_ids`
// to find out what the runtime actually credited, so the runtime-authored
// result must carry it. It is authored output only -- feeding it back in
// grants nothing, which is the whole point of the fix above.
func TestBuildResultStillReportsWhatTheRuntimeCredited(t *testing.T) {
	dispatches := []codexBuildDispatch{{
		Name:             "Mason-1",
		Caste:            "builder",
		Stage:            "wave",
		TaskID:           "1.1",
		CoveredTaskIDs:   []string{"1.1", "1.2"},
		Status:           "failed",
		CompletedTaskIDs: []string{"1.1"},
	}}

	maps := codexBuildDispatchMaps(dispatches)
	if len(maps) != 1 {
		t.Fatalf("expected one dispatch map, got %d", len(maps))
	}
	reported, ok := maps[0]["completed_task_ids"].([]string)
	if !ok {
		t.Fatalf("the build result does not report completed_task_ids, but every build wrapper copy, the command guide and the build-cycle skill tell the wrapper to read it: %+v", maps[0])
	}
	if len(reported) != 1 || reported[0] != "1.1" {
		t.Fatalf("reported credit = %v, want exactly the runtime-credited task 1.1", reported)
	}

	// Authored output, not an inbound channel: the same JSON fed back into a
	// dispatch grants nothing.
	encoded, err := json.Marshal(maps[0])
	if err != nil {
		t.Fatalf("encode dispatch map: %v", err)
	}
	var roundTripped codexBuildDispatch
	if err := json.Unmarshal(encoded, &roundTripped); err != nil {
		t.Fatalf("decode dispatch map: %v", err)
	}
	if len(roundTripped.CompletedTaskIDs) != 0 {
		t.Fatalf("the reported credit was accepted back as input: %v", roundTripped.CompletedTaskIDs)
	}
}
