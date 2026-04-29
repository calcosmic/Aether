package cmd

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/storage"
)

func TestComputeWarningsStaleState(t *testing.T) {
	staleTime := time.Now().Add(-10 * 24 * time.Hour)
	goal := "Stale colony test"
	warnings := computeWarnings(colony.ColonyState{
		Goal:          &goal,
		InitializedAt: &staleTime,
	}, nil)

	found := false
	for _, w := range warnings {
		if strings.Contains(w, "Stale") && strings.Contains(w, "7 days") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected stale warning, got: %v", warnings)
	}
}

func TestComputeWarningsFailedPhase(t *testing.T) {
	goal := "Failed phase test"
	taskID := "t-1"
	warnings := computeWarnings(colony.ColonyState{
		Goal:   &goal,
		State:  colony.StateREADY,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{ID: 1, Name: "Good phase", Status: colony.PhaseCompleted, Tasks: []colony.Task{{ID: &taskID, Goal: "done", Status: colony.TaskCompleted}}},
				{ID: 2, Name: "Good phase 2", Status: colony.PhaseCompleted, Tasks: []colony.Task{{ID: &taskID, Goal: "done", Status: colony.TaskCompleted}}},
				{ID: 3, Name: "Test Phase", Status: "failed", Tasks: []colony.Task{{ID: &taskID, Goal: "fail", Status: colony.TaskPending}}},
			},
		},
	}, nil)

	found := false
	for _, w := range warnings {
		if strings.Contains(w, "Failed phase 3") && strings.Contains(w, "aether build 3") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected failed phase 3 warning, got: %v", warnings)
	}
}

func TestComputeWarningsUnacknowledgedMidden(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "aether-midden-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	dataDir := tmpDir + "/.aether/data"
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		t.Fatal(err)
	}

	midden := colony.MiddenFile{
		Entries: []colony.MiddenEntry{
			{ID: "e1", Timestamp: time.Now().UTC().Format(time.RFC3339), Category: "build", Source: "builder", Message: "Build failed", Acknowledged: nil},
			{ID: "e2", Timestamp: time.Now().UTC().Format(time.RFC3339), Category: "test", Source: "watcher", Message: "Test failed"},
		},
	}
	s, err := createSeedStore(dataDir, &midden, nil)
	if err != nil {
		t.Fatal(err)
	}

	goal := "Midden test"
	warnings := computeWarnings(colony.ColonyState{Goal: &goal}, s)

	found := false
	for _, w := range warnings {
		if strings.Contains(w, "unacknowledged") && strings.Contains(w, "midden-review") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected unacknowledged midden warning, got: %v", warnings)
	}
}

func TestComputeWarningsExpiringPheromone(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "aether-pheromone-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	dataDir := tmpDir + "/.aether/data"
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		t.Fatal(err)
	}

	expiresIn := time.Now().Add(1 * 24 * time.Hour).Format(time.RFC3339)
	pf := colony.PheromoneFile{
		Signals: []colony.PheromoneSignal{
			{ID: "sig1", Type: "FOCUS", Active: true, ExpiresAt: &expiresIn, Content: []byte(`{"text":"Pay attention to this"}`)},
		},
	}
	s, err := createSeedStore(dataDir, nil, &pf)
	if err != nil {
		t.Fatal(err)
	}

	goal := "Expiry test"
	warnings := computeWarnings(colony.ColonyState{Goal: &goal}, s)

	found := false
	for _, w := range warnings {
		if strings.Contains(w, "Expiring signal") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected expiring signal warning, got: %v", warnings)
	}
}

func TestComputeWarningsNoWarnings(t *testing.T) {
	recentTime := time.Now().Add(-1 * time.Hour)
	goal := "Fresh colony"
	taskID := "t-1"
	warnings := computeWarnings(colony.ColonyState{
		Goal:          &goal,
		State:         colony.StateREADY,
		InitializedAt: &recentTime,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{ID: 1, Name: "Phase 1", Status: colony.PhaseCompleted, Tasks: []colony.Task{{ID: &taskID, Goal: "done", Status: colony.TaskCompleted}}},
			},
		},
	}, nil)

	if len(warnings) != 0 {
		t.Errorf("expected no warnings for fresh colony, got: %v", warnings)
	}
}

func TestRenderWarningsSectionEmpty(t *testing.T) {
	result := renderWarningsSection([]string{})
	if result != "" {
		t.Errorf("expected empty string for no warnings, got: %q", result)
	}
}

func TestRenderWarningsSectionWithWarnings(t *testing.T) {
	result := renderWarningsSection([]string{"Warning A", "Warning B"})
	// renderBanner uses spacedTitle which renders "W A R N I N G S"
	if !strings.Contains(result, "W A R N I N G S") {
		t.Errorf("expected banner header with spaced 'Warnings' in output, got: %q", result)
	}
	if !strings.Contains(result, "Warning A") {
		t.Errorf("expected 'Warning A' in output, got: %q", result)
	}
	if !strings.Contains(result, "Warning B") {
		t.Errorf("expected 'Warning B' in output, got: %q", result)
	}
}

func TestWorkflowSuggestionsFailedPhase(t *testing.T) {
	goal := "Failed phase suggestion test"
	taskID := "t-1"
	primary, _ := workflowSuggestionsForState(colony.ColonyState{
		Goal:   &goal,
		State:  colony.StateREADY,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{ID: 1, Name: "Done", Status: colony.PhaseCompleted, Tasks: []colony.Task{{ID: &taskID, Goal: "done", Status: colony.TaskCompleted}}},
				{ID: 2, Name: "Failed Phase", Status: "failed", Tasks: []colony.Task{{ID: &taskID, Goal: "fail", Status: colony.TaskPending}}},
			},
		},
	})

	if !strings.Contains(primary, "aether build") {
		t.Errorf("expected 'aether build' in failed phase suggestion, got: %s", primary)
	}
	if !strings.Contains(primary, "retry") {
		t.Errorf("expected 'retry' in failed phase suggestion, got: %s", primary)
	}
}

func TestWorkflowSuggestionsAllComplete(t *testing.T) {
	goal := "All complete test"
	taskID := "t-1"
	primary, _ := workflowSuggestionsForState(colony.ColonyState{
		Goal:   &goal,
		State:  colony.StateREADY,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{ID: 1, Name: "Phase 1", Status: colony.PhaseCompleted, Tasks: []colony.Task{{ID: &taskID, Goal: "done", Status: colony.TaskCompleted}}},
				{ID: 2, Name: "Phase 2", Status: colony.PhaseCompleted, Tasks: []colony.Task{{ID: &taskID, Goal: "done", Status: colony.TaskCompleted}}},
			},
		},
	})

	if !strings.Contains(primary, "aether seal") {
		t.Errorf("expected 'aether seal' in all-complete suggestion, got: %s", primary)
	}
}

// createSeedStore creates a minimal storage.Store for unit tests.
// Optionally seeds midden.json and/or pheromones.json.
func createSeedStore(dataDir string, midden *colony.MiddenFile, pheromones *colony.PheromoneFile) (*storage.Store, error) {
	s, err := storage.NewStore(dataDir)
	if err != nil {
		return nil, err
	}
	if midden != nil {
		if err := s.SaveJSON("midden.json", midden); err != nil {
			return nil, err
		}
	}
	if pheromones != nil {
		if err := s.SaveJSON("pheromones.json", pheromones); err != nil {
			return nil, err
		}
	}
	return s, nil
}
