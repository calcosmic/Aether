package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// Opt-in replay reads saved metadata only. It never launches workers or writes
// back to the supplied snapshot or its original repository.
func TestContinuePhaseClaimsSavedSnapshot(t *testing.T) {
	snapshot := os.Getenv("AETHER_BOOKKEEPING_SNAPSHOT")
	if snapshot == "" {
		t.Skip("saved-state replay is opt-in")
	}
	saveGlobals(t)
	store, _ = newTestStore(t)
	data, err := os.ReadFile(filepath.Join(snapshot, "build/phase-1/manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	manifest := codexContinueManifest{Present: true}
	if err := json.Unmarshal(data, &manifest.Data); err != nil {
		t.Fatal(err)
	}
	data, err = os.ReadFile(filepath.Join(snapshot, "last-build-claims.json"))
	if err != nil {
		t.Fatal(err)
	}
	var current codexBuildClaims
	if err := json.Unmarshal(data, &current); err != nil {
		t.Fatal(err)
	}
	files, err := filepath.Glob(filepath.Join(snapshot, "build/phase-1/attempts/*.json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range files {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		var record buildAttemptRecord
		if err := json.Unmarshal(data, &record); err != nil {
			t.Fatal(err)
		}
		if record.ID == "" {
			continue
		}
		if err := store.SaveJSON("build/phase-1/attempts/"+filepath.Base(path), record); err != nil {
			t.Fatal(err)
		}
	}
	got := phaseClaimsForContinue(manifest, current)
	if len(got.TaskClaims) <= len(current.TaskClaims) {
		t.Fatalf("lost phase evidence: %d -> %d", len(current.TaskClaims), len(got.TaskClaims))
	}
	statuses, _ := priorAttemptDispatchStatuses(1, map[string][]string{"1.1": {"completed"}, "1.3": {"completed"}})
	if len(statuses["1.2"]) != 1 || statuses["1.2"][0] != "completed" {
		t.Fatalf("repair still conflicts: %v", statuses)
	}
	t.Logf("phase claims %d -> %d; historical task 1.2=%v; snapshot unchanged", len(current.TaskClaims), len(got.TaskClaims), statuses["1.2"])
}

func TestContinuePhaseClaimsRetainsPriorTaskProof(t *testing.T) {
	for _, variant := range []string{"retained", "current-failure", "newer-failure", "different-plan", "synthetic"} {
		t.Run(variant, func(t *testing.T) {
			saveGlobals(t)
			store, _ = newTestStore(t)
			manifest := codexContinueManifest{Present: true, Data: codexBuildManifest{Phase: 1, Root: "fixture", PlanRevisionID: "r1", DispatchMode: "real", SelectedTasks: []string{"1.1"}, Dispatches: []codexBuildDispatch{{TaskID: "1.1", Status: "completed"}}}}
			prior := buildAttemptRecord{ID: "old", Phase: 1, StartedAt: "2026-09-18T01:00:00Z", Status: buildAttemptBuilt, DispatchMode: "real", PlanManifest: &manifest.Data, Dispatches: []codexBuildDispatch{{TaskID: "1.2", Status: "completed"}}, Claims: &codexBuildClaims{BuildPhase: 1, TaskClaims: []codexBuildTaskClaim{{TaskID: "1.2", FilesCreated: []string{"old.txt"}}}, ArtifactEvidence: []codexBuildArtifactEvidence{{Path: "old.txt", SHA256: "original-hash"}}}}
			if variant == "different-plan" {
				m := manifest.Data
				m.PlanRevisionID = "other"
				prior.PlanManifest = &m
			}
			if variant == "synthetic" {
				prior.DispatchMode = "synthetic"
			}
			if variant == "current-failure" {
				manifest.Data.Dispatches = append(manifest.Data.Dispatches, codexBuildDispatch{TaskID: "1.2", Status: "blocked"})
			}
			if err := store.SaveJSON("build/phase-1/attempts/old.json", prior); err != nil {
				t.Fatal(err)
			}
			if variant == "newer-failure" {
				failed := prior
				failed.ID = "new"
				failed.StartedAt = "2026-09-18T02:00:00Z"
				failed.Status = buildAttemptFailed
				if err := store.SaveJSON("build/phase-1/attempts/new.json", failed); err != nil {
					t.Fatal(err)
				}
			}
			current := codexBuildClaims{BuildPhase: 1, FilesCreated: []string{"new.txt"}, TaskClaims: []codexBuildTaskClaim{{TaskID: "1.1", FilesCreated: []string{"new.txt"}}}}
			got := phaseClaimsForContinue(manifest, current)
			want := variant == "retained"
			if containsString(got.FilesCreated, "old.txt") != want {
				t.Fatalf("unexpected historical proof: %+v", got)
			}
			if want && (len(got.ArtifactEvidence) != 1 || got.ArtifactEvidence[0].SHA256 != "original-hash") {
				t.Fatalf("old hash was lost or refreshed: %+v", got)
			}
			if len(current.TaskClaims) != 1 {
				t.Fatal("mutated current claims")
			}
		})
	}
}

func TestContinuePriorTaskLatestOutcomeWins(t *testing.T) {
	for _, last := range []string{buildAttemptBuilt, buildAttemptFailed} {
		t.Run(last, func(t *testing.T) {
			saveGlobals(t)
			store, _ = newTestStore(t)
			for i, status := range []string{buildAttemptPartial, last} {
				dispatchStatus := "blocked"
				if i == 1 {
					dispatchStatus = "completed"
				}
				record := buildAttemptRecord{ID: fmt.Sprint(i), Phase: 1, StartedAt: fmt.Sprintf("2026-09-18T0%d:00:00Z", i), Status: status, DispatchMode: "real", Dispatches: []codexBuildDispatch{{TaskID: "1.2", Status: dispatchStatus}}}
				if err := store.SaveJSON(fmt.Sprintf("build/phase-1/attempts/%d.json", i), record); err != nil {
					t.Fatal(err)
				}
			}
			got, _ := priorAttemptDispatchStatuses(1, map[string][]string{})
			want := "completed"
			if last == buildAttemptFailed {
				want = "failed"
			}
			if len(got["1.2"]) != 1 || got["1.2"][0] != want {
				t.Fatalf("latest outcome lost: %v", got)
			}
			got, _ = priorAttemptDispatchStatuses(1, map[string][]string{"1.2": {"blocked"}})
			if len(got) != 0 {
				t.Fatal("history masked current result")
			}
		})
	}
}
