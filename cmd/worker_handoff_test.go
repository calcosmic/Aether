package cmd

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/storage"
)

func TestPersistAndRenderWorkerHandoff(t *testing.T) {
	saveGlobals(t)
	root := t.TempDir()
	s, err := storage.NewStore(filepath.Join(root, ".aether", "data"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	store = s

	dispatch := codex.WorkerDispatch{
		WorkerName: "Builder-1",
		Caste:      "builder",
		TaskID:     "1.1",
		Workflow:   "build",
		Phase:      1,
		Wave:       1,
		Root:       root,
	}
	result := codex.DispatchResult{
		WorkerName: "Builder-1",
		Status:     "completed",
		WorkerResult: &codex.WorkerResult{
			WorkerName: "Builder-1",
			Caste:      "builder",
			TaskID:     "1.1",
			Status:     "completed",
			Summary:    "wired dispatch path lookup",
			Handoff: codex.WorkerHandoff{
				ChangedFiles:           []string{filepath.Join(root, "cmd", "dispatch.go")},
				CommandsRun:            []string{"go test ./cmd -run TestDispatch"},
				VerificationStatus:     "pass",
				NextWorkerInstructions: []string{"Check OpenCode parity next."},
				DoNotRepeat:            []string{"Do not recreate repo-local agent mirrors."},
				Freshness:              "2026-05-04T10:00:00Z",
			},
		},
	}
	if err := persistDispatchWorkerHandoff(dispatch, result); err != nil {
		t.Fatalf("persist handoff: %v", err)
	}

	raw, err := s.ReadFile(workerHandoffsPath)
	if err != nil {
		t.Fatalf("read handoffs: %v", err)
	}
	var file workerHandoffFile
	if err := json.Unmarshal(raw, &file); err != nil {
		t.Fatalf("unmarshal handoffs: %v", err)
	}
	if len(file.Entries) != 1 {
		t.Fatalf("entries = %d, want 1", len(file.Entries))
	}
	if got := file.Entries[0].ChangedFiles; len(got) != 1 || got[0] != "cmd/dispatch.go" {
		t.Fatalf("changed files = %#v, want normalized repo-relative path", got)
	}

	section := renderWorkerHandoffSection("build", 1, "Watcher-1")
	for _, want := range []string{"Previous Worker Handoffs", "cmd/dispatch.go", "Check OpenCode parity next.", "Do not recreate repo-local agent mirrors."} {
		if !strings.Contains(section, want) {
			t.Fatalf("rendered handoff missing %q:\n%s", want, section)
		}
	}
}

func TestColonyPrimeIncludesWorkerHandoffs(t *testing.T) {
	saveGlobals(t)
	root := t.TempDir()
	s, err := storage.NewStore(filepath.Join(root, ".aether", "data"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	store = s

	goal := "handoff context"
	state := colony.ColonyState{
		Version:      "1.0",
		Goal:         &goal,
		State:        colony.StateREADY,
		CurrentPhase: 2,
		Plan: colony.Plan{Phases: []colony.Phase{
			{ID: 1, Name: "Previous", Status: colony.PhaseCompleted},
			{ID: 2, Name: "Current", Status: colony.PhaseReady, Tasks: []colony.Task{{Goal: "Use handoff", Status: colony.TaskPending}}},
		}},
	}
	if err := s.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatalf("save state: %v", err)
	}
	if err := persistDispatchWorkerHandoff(codex.WorkerDispatch{
		WorkerName: "Builder-2",
		Caste:      "builder",
		TaskID:     "2.1",
		Workflow:   "build",
		Phase:      2,
		Root:       root,
	}, codex.DispatchResult{
		WorkerName: "Builder-2",
		Status:     "blocked",
		WorkerResult: &codex.WorkerResult{
			WorkerName: "Builder-2",
			Status:     "blocked",
			Summary:    "blocked by missing published agent files",
			Handoff: codex.WorkerHandoff{
				VerificationStatus:     "fail",
				KnownFailures:          []string{"global hub missing Claude agent files"},
				NextWorkerInstructions: []string{"Verify publish and update both use hub-backed agents."},
				Freshness:              "2026-05-04T11:00:00Z",
			},
		},
	}); err != nil {
		t.Fatalf("persist handoff: %v", err)
	}

	output := buildColonyPrimeOutput(false)
	if !strings.Contains(output.Context, "Previous Worker Handoffs") {
		t.Fatalf("expected colony-prime to include worker handoffs:\n%s", output.Context)
	}
	if !strings.Contains(output.Context, "Verify publish and update both use hub-backed agents.") {
		t.Fatalf("expected handoff instructions in colony-prime:\n%s", output.Context)
	}
}
