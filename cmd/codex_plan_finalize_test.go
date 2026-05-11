package cmd

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

func TestValidateExternalPlanStateSuggestsStaleCleanupForFreshManifest(t *testing.T) {
	saveGlobals(t)

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	goal := "Build a fresh dashboard"
	taskID := "1.1"
	createTestColonyState(t, s.BasePath(), colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateREADY,
		CurrentPhase: 1,
		Plan: colony.Plan{
			Phases: []colony.Phase{{
				ID:    1,
				Name:  "Old stale phase",
				Status: colony.PhaseReady,
				Tasks: []colony.Task{{
					ID:    &taskID,
					Goal:  "Old task",
					Status: colony.TaskPending,
				}},
			}},
		},
	})

	manifest := &codexPlanManifest{
		Goal:         goal,
		Root:         tmpDir,
		Granularity:  "milestone",
		ExistingPlan: false,
		Refresh:      false,
	}

	_, _, err := validateExternalPlanState(manifest)
	if err == nil {
		t.Fatal("expected error when colony has stale phases but manifest says fresh plan")
	}
	errMsg := err.Error()
	// The error should mention stale state to help the user understand
	// that existing phases from a prior session are blocking finalization.
	if !strings.Contains(errMsg, "stale") {
		t.Fatalf("error should mention stale state for fresh manifest with existing phases, got: %s", errMsg)
	}
}

// TestValidateExternalPlanStateAllowsExistingPlanWhenManifestAcknowledges verifies
// that plan-finalize succeeds when manifest.ExistingPlan is true and phases exist.
// FIX C regression: the guard was over-rejecting by not checking manifest.ExistingPlan.
func TestValidateExternalPlanStateAllowsExistingPlanWhenManifestAcknowledges(t *testing.T) {
	saveGlobals(t)

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	goal := "Build a dashboard"
	taskID := "1.1"
	createTestColonyState(t, s.BasePath(), colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateREADY,
		CurrentPhase: 1,
		Plan: colony.Plan{
			Phases: []colony.Phase{{
				ID:    1,
				Name:  "Foundation phase",
				Status: colony.PhaseReady,
				Tasks: []colony.Task{{
					ID:    &taskID,
					Goal:  "Set up foundation",
					Status: colony.TaskPending,
				}},
			}},
		},
	})

	manifest := &codexPlanManifest{
		Goal:         goal,
		Root:         tmpDir,
		Granularity:  "milestone",
		ExistingPlan: true,
		Refresh:      false,
	}

	_, _, err := validateExternalPlanState(manifest)
	if err != nil {
		t.Fatalf("expected no error when manifest.ExistingPlan is true, got: %s", err)
	}
}

// TestEffectiveContinueReviewTimeoutDefaultsTo10Minutes verifies that the
// default review timeout for continue workers is 10 minutes (increased from 5m).
// FIX B regression: review worker timeout was too short for complex verification.
func TestEffectiveContinueReviewTimeoutDefaultsTo10Minutes(t *testing.T) {
	got := effectiveContinueReviewTimeout(0)
	want := 10 * time.Minute
	if got != want {
		t.Fatalf("effectiveContinueReviewTimeout(0) = %v, want %v", got, want)
	}
}

// TestEffectiveContinueReviewTimeoutHonorsOverride verifies that an explicit
// override is respected even when the default is increased.
func TestEffectiveContinueReviewTimeoutHonorsOverride(t *testing.T) {
	override := 3 * time.Minute
	got := effectiveContinueReviewTimeout(override)
	if got != override {
		t.Fatalf("effectiveContinueReviewTimeout(%v) = %v, want %v", override, got, override)
	}
}
