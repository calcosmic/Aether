package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

// directoryBindingTestPhase builds a single-criterion, phase-level evidence
// binding to exactly one artifact path, for exercising the build-time
// directory refusal in isolation.
func directoryBindingTestPhase(artifactRel string) colony.Phase {
	criterion := fmt.Sprintf("No file under %s was modified", artifactRel)
	return colony.Phase{
		ID:              1,
		Name:            "Directory binding",
		Status:          colony.PhaseInProgress,
		SuccessCriteria: []string{criterion},
		EvidenceRequirements: []colony.CriterionEvidenceRequirement{
			{Criterion: criterion, Artifacts: []string{artifactRel}},
		},
	}
}

// TestDirectoryArtifactIsRefusedByNameAtBuildTime is the field-report case
// (WINDOWS.md row 49): a phase binding an existing directory as a criterion
// artifact must be refused, by name, at the earliest point the phase is
// used -- never left to fail later as an artifact nobody could ever claim.
func TestDirectoryArtifactIsRefusedByNameAtBuildTime(t *testing.T) {
	t.Run("existing directory is refused", func(t *testing.T) {
		saveGlobals(t)
		dataDir := setupBuildFlowTest(t)
		root := filepath.Dir(filepath.Dir(dataDir))

		if err := os.MkdirAll(filepath.Join(root, "server", "app"), 0755); err != nil {
			t.Fatalf("create directory fixture: %v", err)
		}

		phase := directoryBindingTestPhase("server/app")
		err := validatePhaseCriterionEvidenceAgainstDisk(root, phase)
		if err == nil {
			t.Fatal("expected a directory binding to be refused, got nil")
		}
		var r refusal
		if !errors.As(err, &r) {
			t.Fatalf("expected a typed refusal, got: %v", err)
		}
		if r.ID != "criterion-artifact-is-a-directory" {
			t.Fatalf("unexpected refusal id: %q", r.ID)
		}
		if !strings.Contains(r.Why, "server/app") {
			t.Errorf("refusal does not name the offending path: %q", r.Why)
		}
		if !strings.Contains(r.Why, phase.SuccessCriteria[0]) {
			t.Errorf("refusal does not name the criterion text: %q", r.Why)
		}
	})

	t.Run("symlink to a directory is refused the same way", func(t *testing.T) {
		saveGlobals(t)
		dataDir := setupBuildFlowTest(t)
		root := filepath.Dir(filepath.Dir(dataDir))

		if err := os.MkdirAll(filepath.Join(root, "real-target"), 0755); err != nil {
			t.Fatalf("create symlink target: %v", err)
		}
		if err := os.Symlink(filepath.Join(root, "real-target"), filepath.Join(root, "shortcut")); err != nil {
			t.Skipf("symbolic links unavailable: %v", err)
		}

		phase := directoryBindingTestPhase("shortcut")
		err := validatePhaseCriterionEvidenceAgainstDisk(root, phase)
		if err == nil {
			t.Fatal("expected a symlink-to-a-directory binding to be refused, got nil")
		}
		var r refusal
		if !errors.As(err, &r) {
			t.Fatalf("expected a typed refusal, got: %v", err)
		}
		if !strings.Contains(r.Why, "shortcut") {
			t.Errorf("refusal does not name the offending shortcut path: %q", r.Why)
		}
	})

	t.Run("not-yet-existing path is never refused", func(t *testing.T) {
		saveGlobals(t)
		dataDir := setupBuildFlowTest(t)
		root := filepath.Dir(filepath.Dir(dataDir))

		phase := directoryBindingTestPhase("output/report-not-written-yet.md")
		if err := validatePhaseCriterionEvidenceAgainstDisk(root, phase); err != nil {
			t.Fatalf("a not-yet-existing artifact path must never be refused, got: %v", err)
		}
	})
}

// TestDirectoryArtifactRefusalNamesTheWayPast proves the refusal's next
// command is not just non-empty text, but a command genuinely registered on
// the live cobra tree -- derived by looking the command up (availableCommand,
// the same resolver the what-next card uses), never by comparing against a
// typed string that could silently rot.
func TestDirectoryArtifactRefusalNamesTheWayPast(t *testing.T) {
	saveGlobals(t)
	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))

	if err := os.MkdirAll(filepath.Join(root, "server", "app"), 0755); err != nil {
		t.Fatalf("create directory fixture: %v", err)
	}

	phase := directoryBindingTestPhase("server/app")
	err := validatePhaseCriterionEvidenceAgainstDisk(root, phase)
	var r refusal
	if !errors.As(err, &r) {
		t.Fatalf("expected a typed refusal, got: %v", err)
	}
	if strings.TrimSpace(r.NextCommand) == "" {
		t.Fatal("expected a non-empty next command")
	}
	if r.ProtectsWork {
		t.Error("expected ProtectsWork=false: rewriting the binding loses no prior work")
	}
	if _, ok := availableCommand(r.NextCommand); !ok {
		t.Fatalf("next command %q does not resolve against the live cobra tree", r.NextCommand)
	}
}

// TestOrdinaryCriterionIsUnaffected proves the ordinary, regular-file
// evidence path -- both the new build-time disk check and the existing
// evaluatePhaseCriterionEvidence run-time path -- is byte-for-byte unchanged
// by this phase's additions.
func TestOrdinaryCriterionIsUnaffected(t *testing.T) {
	phase := criterionEvidenceTestPhase()
	root, manifest := setupCriterionEvidenceTest(t, phase)

	if err := validatePhaseCriterionEvidenceAgainstDisk(root, phase); err != nil {
		t.Fatalf("ordinary regular-file bindings must validate against disk, got: %v", err)
	}

	evaluation := evaluatePhaseCriterionEvidence(root, phase, manifest, passingCriterionSteps(), codexClaimVerification{Present: true, Passed: true}, codexWatcherVerification{})
	if !evaluation.Enforced || !evaluation.Passed || !evaluation.Deterministic {
		t.Fatalf("evaluation = %+v, want enforced deterministic pass (unaffected by this phase)", evaluation)
	}
	if len(evaluation.Criteria) != 2 || !evaluation.Criteria[0].Passed || !evaluation.Criteria[1].Passed {
		t.Fatalf("criteria = %+v, want two passes", evaluation.Criteria)
	}
	for _, c := range evaluation.Criteria {
		if c.State == criterionStateNeedsOwnerConfirmation {
			t.Fatalf("ordinary criterion was wrongly routed to owner confirmation: %+v", c)
		}
	}
}
