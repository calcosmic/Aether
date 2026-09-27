package cmd

// 208-06-PLAN.md Task 3 (UED-11): the checked-in refusal table is not a
// document -- flipping a row's Disposition in the register without changing
// the program that drives it fails TestBehaviourMatchesTheRefusalTable by
// name. Each driver below puts the program in the exact state one row's own
// call site fires from and asserts the real, observable behaviour -- a
// non-nil typed error for "stop", no error plus a rendered notice plus
// continued work for "warn" -- matches what that row declares.

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/spf13/cobra"
)

// refusalDriveResult is what a driver reports back: the error the site
// returned (nil for a warn row that carried on), the human-facing text the
// drive produced (an error's own Error() string for a stop row, or the
// rendered stdout block for a warn row), and whether the work the refusal
// gates actually ran afterward.
type refusalDriveResult struct {
	Err           error
	Rendered      string
	WorkContinued bool
}

// refusalRowsWithoutDrivers is the declared, reason-carrying list of rows
// TestBehaviourMatchesTheRefusalTable does not drive directly. Asserted
// shorter than the number of rows WITH a driver (TestBehaviourMatchesTheRefusalTable
// itself), so this can never become the majority of the table by omission.
var refusalRowsWithoutDrivers = map[string]string{
	"colonize-finalize-existing-survey-found": "Reaching this exact site (runCodexColonizeFinalize) requires reconstructing a full plan-only manifest with Snapshots marking the survey markdown files Existed, plus its freshness/workspace preconditions all satisfied; the identical dead end (an existing survey without --force-resurvey) is driven end to end by the sibling row \"colonize-existing-survey-found\" below, which exercises codex_colonize.go's direct-dispatch copy of the same check.",
	"criterion-binding-unsatisfiable":         "Registered for its NextCommand text only (criterionBindingIsUnsatisfiable routes to the pre-existing owner-confirmation state and renders its own dynamic command via ownerConfirmationCommand, never a refuse(...) call at a single site) -- covered by 208-05's TestUnsatisfiableCriterionRoutesToTheOwnerAnswer.",
	"entomb-manifest-digest-mismatch":         "verifyPublishedEntombManifest requires a schema-valid ArchiveManifest (Seal/Entries/CrossReferences/Transaction/Receipt each with their own Validate()) before reaching this check; constructing one is exercised end to end by TestEntomb* (Task 2 verification), not duplicated here for driver-cost reasons.",
	"entomb-manifest-receipt-mismatch":        "Same ArchiveManifest construction cost as entomb-manifest-digest-mismatch immediately above; covered by TestEntomb*.",
	"failed-to-initialize-store":              "Legacy pattern-matched row (friendlyErrorForPattern, driven via renderVisualError for the other five legacy rows below) with no refuse(...) call site anywhere -- fires only from a real storage.NewStore failure (e.g. an unwritable data directory), which is environment-dependent and not reproduced here; the pattern-lookup path itself is proven for the other five.",
	"pause-colony-state-not-runnable":         "pauseColonyInMutationSession requires a full planningMutationSession plus a real CONTEXT.md/HANDOFF.md read before reaching this check; exercised end to end by the existing pause/resume test suite (TestPauseResume*), not duplicated here for driver-cost reasons.",
	"pause-colony-state-unconfirmed":          "Same planningMutationSession construction cost as pause-colony-state-not-runnable immediately above; covered by TestPauseResume*.",
	"plan-empty-phase-list":                   "runCodexPlanFinalizeInSession requires territory-snapshot, workspace and manifest-freshness validation to all pass before reaching this check; exercised end to end by TestCodexPlanFinalize* (Task 2 verification), not duplicated here for driver-cost reasons.",
	"resume-handoff-reference-missing":        "resumeColonyAt requires a full lifecycle-facts fixture with handoff reconstruction deliberately disabled; exercised end to end by the existing pause/resume test suite (TestPauseResume*), not duplicated here for driver-cost reasons.",
}

// refusalDrivers maps a row id to the closure that drives its own site.
// Every row in refusalRegistry must appear here OR in refusalRowsWithoutDrivers
// -- TestBehaviourMatchesTheRefusalTable fails by name for a row in neither.
var refusalDrivers = map[string]func(t *testing.T) refusalDriveResult{
	// The five legacy pattern-matched rows below have no refuse(...) call
	// site anywhere (they predate the typed refusal contract, folded in
	// from the old hinted-error map by 208-01) -- their real, live site is
	// renderVisualError (cmd/helpers.go), the one production caller of
	// friendlyErrorForPattern. Sample messages match 208-01's own
	// TestFriendlyErrorPatternMatch* fixtures.
	"corrupted-colony-data": func(t *testing.T) refusalDriveResult {
		rendered := renderVisualError("json: cannot unmarshal string into Go value", nil)
		return refusalDriveResult{Rendered: rendered}
	},
	"failed-to-load-colony-state": func(t *testing.T) refusalDriveResult {
		rendered := renderVisualError("failed to load colony state: file not found", nil)
		return refusalDriveResult{Rendered: rendered}
	},
	"invalid-charter-json": func(t *testing.T) refusalDriveResult {
		rendered := renderVisualError("invalid charter JSON: invalid character 'o' in literal null", nil)
		return refusalDriveResult{Rendered: rendered}
	},
	"no-colony-initialized": func(t *testing.T) refusalDriveResult {
		rendered := renderVisualError("no colony initialized", nil)
		return refusalDriveResult{Rendered: rendered}
	},
	"permission-denied": func(t *testing.T) refusalDriveResult {
		rendered := renderVisualError("permission denied: /some/path", nil)
		return refusalDriveResult{Rendered: rendered}
	},
	"missing-required-flag": func(t *testing.T) refusalDriveResult {
		_, err := loadExternalColonizeCompletion("")
		return refusalDriveResult{Err: err}
	},
	"charter-field-too-long": func(t *testing.T) refusalDriveResult {
		err := validateCharterFieldLength(colony.Charter{Intent: strings.Repeat("x", 2001)})
		return refusalDriveResult{Err: err}
	},
	"plan-missing-success-criteria": func(t *testing.T) refusalDriveResult {
		err := validateNewPlanEvidenceContract([]colony.Phase{{ID: 1, Name: "No criteria phase"}})
		return refusalDriveResult{Err: err}
	},
	"colonize-finalize-missing-timestamp": func(t *testing.T) refusalDriveResult {
		err := validateCodexColonizeManifestFreshness(codexColonizeManifest{}, time.Now().UTC())
		return refusalDriveResult{Err: err}
	},
	"colonize-finalize-invalid-timestamp": func(t *testing.T) refusalDriveResult {
		err := validateCodexColonizeManifestFreshness(codexColonizeManifest{GeneratedAt: "not-a-timestamp"}, time.Now().UTC())
		return refusalDriveResult{Err: err}
	},
	"colonize-finalize-timestamp-in-future": func(t *testing.T) refusalDriveResult {
		now := time.Now().UTC()
		err := validateCodexColonizeManifestFreshness(codexColonizeManifest{GeneratedAt: now.Add(24 * time.Hour).Format(time.RFC3339)}, now)
		return refusalDriveResult{Err: err}
	},
	"build-dispatch-manifest-wrong-source": func(t *testing.T) refusalDriveResult {
		saveGlobals(t)
		s, tmpDir := newTestStore(t)
		defer os.RemoveAll(tmpDir)
		store = s
		completion := codexExternalBuildCompletion{DispatchManifest: &codexBuildManifest{PlanOnly: false}}
		result, state, phase, dispatches, err := runCodexBuildFinalizeWithHooks(tmpDir, 1, completion, false, codexNativeFinalizeHooks{})
		return refusalDriveResult{
			Err:           err,
			WorkContinued: result != nil || state.Goal != nil || phase.ID != 0 || len(dispatches) != 0,
		}
	},
	"no-project-plan": func(t *testing.T) refusalDriveResult {
		saveGlobals(t)
		s, tmpDir := newTestStore(t)
		defer os.RemoveAll(tmpDir)
		store = s
		dataDir := filepath.Join(tmpDir, ".aether", "data")
		goal := "test goal"
		createTestColonyState(t, dataDir, colony.ColonyState{
			Version: "3.0", Goal: &goal, State: colony.StateEXECUTING,
			Plan: colony.Plan{Phases: []colony.Phase{}},
		})
		_, phase, _, err := validateExternalContinueState(&codexContinuePlanManifest{})
		return refusalDriveResult{Err: err, WorkContinued: phase.ID != 0}
	},
	"continue-on-paused-project": func(t *testing.T) refusalDriveResult {
		saveGlobals(t)
		s, tmpDir := newTestStore(t)
		defer os.RemoveAll(tmpDir)
		store = s
		dataDir := filepath.Join(tmpDir, ".aether", "data")
		goal := "test goal"
		createTestColonyState(t, dataDir, colony.ColonyState{
			Version: "3.0", Goal: &goal, State: colony.StateBUILT, CurrentPhase: 1, Paused: true,
			Plan: colony.Plan{Phases: []colony.Phase{{ID: 1, Name: "Phase one"}}},
		})
		_, _, phase, _, err := runCodexContinuePlanOnly(tmpDir, codexContinueOptions{})
		return refusalDriveResult{Err: err, WorkContinued: phase.ID != 0}
	},
	"no-active-phase-to-continue": func(t *testing.T) refusalDriveResult {
		saveGlobals(t)
		s, tmpDir := newTestStore(t)
		defer os.RemoveAll(tmpDir)
		store = s
		dataDir := filepath.Join(tmpDir, ".aether", "data")
		goal := "test goal"
		createTestColonyState(t, dataDir, colony.ColonyState{
			Version: "3.0", Goal: &goal, State: colony.StateREADY,
			Plan: colony.Plan{Phases: []colony.Phase{{ID: 1, Name: "Phase one"}}},
		})
		_, phase, _, err := validateExternalContinueState(&codexContinuePlanManifest{})
		return refusalDriveResult{Err: err, WorkContinued: phase.ID != 0}
	},
	"pause-handoff-missing-from-state": func(t *testing.T) refusalDriveResult {
		saveGlobals(t)
		s, tmpDir := newTestStore(t)
		defer os.RemoveAll(tmpDir)
		store = s
		dataDir := filepath.Join(tmpDir, ".aether", "data")
		goal := "test goal"
		createTestColonyState(t, dataDir, colony.ColonyState{
			Version: "3.0", Goal: &goal, State: colony.StateREADY, PauseHandoff: nil,
		})
		handoff, err := loadValidatedPauseHandoffReferenceFromState()
		return refusalDriveResult{Err: err, WorkContinued: handoff.HandoffID != ""}
	},
	"criterion-artifact-is-a-directory": func(t *testing.T) refusalDriveResult {
		saveGlobals(t)
		dataDir := setupBuildFlowTest(t)
		root := filepath.Dir(filepath.Dir(dataDir))
		if err := os.MkdirAll(filepath.Join(root, "server", "app"), 0755); err != nil {
			t.Fatalf("create directory fixture: %v", err)
		}
		phase := directoryBindingTestPhase("server/app")
		err := validatePhaseCriterionEvidenceAgainstDisk(root, phase)
		return refusalDriveResult{Err: err}
	},
	"invalid-timeout-value": func(t *testing.T) refusalDriveResult {
		saveGlobals(t)
		var buf bytes.Buffer
		stdout = &buf
		t.Setenv("AETHER_OUTPUT_MODE", "visual")
		c := &cobra.Command{Use: "x"}
		c.Flags().Duration("worker-timeout", 0, "")
		if err := c.Flags().Set("worker-timeout", "-5m"); err != nil {
			t.Fatalf("set worker-timeout: %v", err)
		}
		timeout, err := resolveWorkerTimeoutFlag(c)
		return refusalDriveResult{
			Err:           err,
			Rendered:      buf.String(),
			WorkContinued: err == nil && timeout == 0,
		}
	},
	"verification-command-not-understood": func(t *testing.T) refusalDriveResult {
		saveGlobals(t)
		s, root := newTestStore(t)
		store = s
		unreadableLine := "- tests: frobnicate the widgets"
		if err := os.WriteFile(filepath.Join(root, "CLAUDE.md"), []byte(
			"## Verification Commands\n\n"+unreadableLine+"\n"), 0644); err != nil {
			t.Fatalf("write CLAUDE.md: %v", err)
		}
		phase := colony.Phase{ID: 1, Name: "Unreadable verification line"}
		manifest := codexContinueManifest{}
		floor := runDeterministicFloor(context.Background(), root, phase, manifest, codexWatcherVerification{}, 5*time.Second)
		var found string
		for _, issue := range floor.BlockingIssues {
			if strings.Contains(issue, unreadableLine) {
				found = issue
			}
		}
		var err error
		if found == "" {
			err = errNoRefusalObserved
		}
		return refusalDriveResult{Err: err, Rendered: found}
	},
	"colonize-existing-survey-found": func(t *testing.T) refusalDriveResult {
		saveGlobals(t)
		resetRootCmd(t)
		dataDir := setupBuildFlowTest(t)
		root := filepath.Dir(filepath.Dir(dataDir))
		oldDir, err := os.Getwd()
		if err != nil {
			t.Fatalf("getwd: %v", err)
		}
		if err := os.Chdir(root); err != nil {
			t.Fatalf("chdir: %v", err)
		}
		defer os.Chdir(oldDir)
		if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/aether-test\n\ngo 1.24\n"), 0644); err != nil {
			t.Fatalf("write go.mod: %v", err)
		}
		if err := os.MkdirAll(filepath.Join(dataDir, "survey"), 0755); err != nil {
			t.Fatalf("create survey dir: %v", err)
		}
		if err := os.WriteFile(filepath.Join(dataDir, "survey", "PROVISIONS.md"), []byte("# old survey\n"), 0644); err != nil {
			t.Fatalf("write old survey: %v", err)
		}
		goal := "Survey the repo"
		createTestColonyState(t, dataDir, colony.ColonyState{
			Version: "3.0", Goal: &goal, State: colony.StateREADY,
			Plan: colony.Plan{Phases: []colony.Phase{}},
		})
		var errBuf bytes.Buffer
		stderr = &errBuf
		rootCmd.SetArgs([]string{"colonize"})
		if execErr := rootCmd.Execute(); execErr != nil {
			t.Fatalf("colonize returned error: %v", execErr)
		}
		rendered := errBuf.String()
		var driveErr error
		if !strings.Contains(rendered, "A territory survey already exists for this project") {
			driveErr = errNoRefusalObserved
		}
		return refusalDriveResult{Err: driveErr, Rendered: rendered}
	},
}

// errNoRefusalObserved is a sentinel a driver returns when it could not
// observe the refusal through an indirect (CLI or multi-step) path -- kept
// distinct from a real refusal error so a test failure names the right
// cause.
var errNoRefusalObserved = errors.New("refusal not observed by its driver")

// TestBehaviourMatchesTheRefusalTable is the ratchet: it does not pass
// because the table LOOKS right, it passes because driving the real program
// at each row's own site produces the behaviour that row declares.
func TestBehaviourMatchesTheRefusalTable(t *testing.T) {
	var noDriverCount, withDriverCount int
	for _, row := range refusalRegistry {
		row := row
		driver, hasDriver := refusalDrivers[row.ID]
		if !hasDriver {
			reason, declared := refusalRowsWithoutDrivers[row.ID]
			if !declared || strings.TrimSpace(reason) == "" {
				t.Errorf("refusal row %q has no behavioural driver and is not in the declared, reason-carrying refusalRowsWithoutDrivers list", row.ID)
			}
			noDriverCount++
			continue
		}
		withDriverCount++
		t.Run(row.ID, func(t *testing.T) {
			result := driver(t)
			switch row.Disposition {
			case "stop":
				switch {
				case result.Err != nil:
					// The ordinary case: the site returns a typed refusal
					// as a Go error.
					var r refusal
					if !errors.As(result.Err, &r) {
						t.Fatalf("row %q is Disposition=stop but its driver's error is not a typed refusal: %v", row.ID, result.Err)
					}
					if r.ID != row.ID {
						t.Fatalf("row %q's driver produced a refusal carrying id %q instead", row.ID, r.ID)
					}
					if !strings.Contains(r.Error(), row.NextCommand) {
						t.Errorf("row %q's refusal text does not carry its own next command %q: %s", row.ID, row.NextCommand, r.Error())
					}
				case result.Rendered != "":
					// The legacy pattern-matched path: friendlyErrorForPattern
					// renders text directly (cmd/helpers.go's renderVisualError)
					// rather than returning a typed refusal error.
					if !strings.Contains(result.Rendered, row.NextCommand) {
						t.Errorf("row %q's rendered output does not carry its own next command %q: %s", row.ID, row.NextCommand, result.Rendered)
					}
				default:
					t.Fatalf("row %q is Disposition=stop but its driver produced neither an error nor rendered output", row.ID)
				}
				if result.WorkContinued {
					t.Errorf("row %q is Disposition=stop but the work after its site ran anyway", row.ID)
				}
			case "warn":
				if result.Err != nil {
					t.Fatalf("row %q is Disposition=warn but its driver produced an error: %v", row.ID, result.Err)
				}
				if !strings.Contains(result.Rendered, row.NextCommand) {
					t.Errorf("row %q is Disposition=warn but its rendered output does not carry its own next command %q: %s", row.ID, row.NextCommand, result.Rendered)
				}
				if !result.WorkContinued {
					t.Errorf("row %q is Disposition=warn but the work after its site did not run", row.ID)
				}
			default:
				t.Fatalf("row %q has an unrecognised Disposition %q", row.ID, row.Disposition)
			}
		})
	}
	if noDriverCount >= withDriverCount {
		t.Fatalf("refusalRowsWithoutDrivers (%d) is not shorter than the number of rows with a driver (%d) -- coverage would be mostly excuses", noDriverCount, withDriverCount)
	}
}

// TestNoRefusalBothWarnsAndStops scans the declared files with the Task 1
// enumerator's own AST walk (never a second, hand-rolled one) for any id
// that appears both inside a return statement's refuse(...) call (a stop
// use) and as an argument to warnAndCarryOn(...) (a warn use) -- a refusal
// must never both protect against losing work and be told to carry on.
func TestNoRefusalBothWarnsAndStops(t *testing.T) {
	stopIDs, warnIDs, err := refusalIDUsagesByDisposition(refusalDeclaredFiles)
	if err != nil {
		t.Fatalf("refusalIDUsagesByDisposition: %v", err)
	}
	for id := range stopIDs {
		if warnIDs[id] {
			t.Errorf("refusal id %q is used both inside a return (stop) and as a warnAndCarryOn(...) argument (warn) in the declared files", id)
		}
	}
}
