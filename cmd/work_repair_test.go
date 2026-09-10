package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// workRepairFixtureRoot builds a tiny, self-contained working tree with one
// source file a repair wave can mutate -- used by every test in this file
// as the checkpoint/restore target.
func workRepairFixtureRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "cmd"), 0o755); err != nil {
		t.Fatalf("mkdir cmd: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "cmd", "foo.go"), []byte("package cmd\n\nfunc Foo() int { return 1 }\n"), 0o644); err != nil {
		t.Fatalf("write foo.go: %v", err)
	}
	return root
}

func workRepairFixtureInput(root string, phase int, check string) repairRoundInput {
	return repairRoundInput{
		Phase: phase, Attempt: "attempt-1", Check: check,
		Root: root, Evidence: []string{"still failing"},
		PermittedScope: []string{"cmd/foo.go"},
		PlannedAction:  "patch cmd/foo.go",
		Baseline:       "sha256:before",
		SafetySafe:     true, AuthoritySafe: true, ScopeSafe: true,
	}
}

func noopPersist(autopilotRepairLedger) error { return nil }

// TestRepairRunsAtMostOnceAutomatically proves a failing fixture with an
// eligible repair produces exactly one checkpoint, one repair wave and one
// re-verification, and that a second automatic attempt for the same failing
// verification is refused by name rather than run again.
func TestRepairRunsAtMostOnceAutomatically(t *testing.T) {
	root := workRepairFixtureRoot(t)
	ledger := newAutopilotRepairLedger("run-201-09-once", 3)
	input := workRepairFixtureInput(root, 9, "go test ./cmd")

	repairCalls, verifyCalls := 0, 0
	repairFn := func() error {
		repairCalls++
		return os.WriteFile(filepath.Join(root, "cmd", "foo.go"), []byte("package cmd\n\nfunc Foo() int { return 2 }\n"), 0o644)
	}
	verifyFn := func() (bool, []string, error) {
		verifyCalls++
		return true, []string{"tests passed"}, nil
	}

	first, err := runBoundedRepairRound(&ledger, input, nil, noopPersist, repairFn, verifyFn, nil, nil)
	if err != nil {
		t.Fatalf("first round: %v", err)
	}
	if !first.Ran || first.Replayed || !first.Passed {
		t.Fatalf("first round outcome = %+v", first)
	}
	if repairCalls != 1 || verifyCalls != 1 {
		t.Fatalf("expected exactly one repair wave and one re-verification, got repair=%d verify=%d", repairCalls, verifyCalls)
	}

	second, err := runBoundedRepairRound(&ledger, input, nil, noopPersist, repairFn, verifyFn, nil, nil)
	if err != nil {
		t.Fatalf("second round: %v", err)
	}
	if second.Ran {
		t.Fatalf("a second automatic attempt ran instead of being refused: %+v", second)
	}
	if second.Reason == "" || !strings.Contains(second.Reason, "already spent") {
		t.Fatalf("refusal does not name the spent round: %q", second.Reason)
	}
	if repairCalls != 1 || verifyCalls != 1 {
		t.Fatalf("the refused second attempt still performed work: repair=%d verify=%d", repairCalls, verifyCalls)
	}
}

// TestFailedRepairRestoresTheCheckpointExactly proves a failed repair leaves
// the fixture's working tree byte-identical to its state at checkpoint time,
// and that a passed repair, by contrast, leaves the repaired files in place
// and records a passed receipt.
func TestFailedRepairRestoresTheCheckpointExactly(t *testing.T) {
	root := workRepairFixtureRoot(t)
	scope := []string{"cmd/foo.go"}
	before, err := repairCheckpointDirectoryDigest(root, scope)
	if err != nil {
		t.Fatalf("digest before: %v", err)
	}

	ledger := newAutopilotRepairLedger("run-201-09-restore", 1)
	input := workRepairFixtureInput(root, 10, "go test ./cmd")

	outcome, err := runBoundedRepairRound(&ledger, input, nil, noopPersist,
		func() error {
			return os.WriteFile(filepath.Join(root, "cmd", "foo.go"), []byte("package cmd\n\nfunc Foo() int { return 999 }\n"), 0o644)
		},
		func() (bool, []string, error) { return false, []string{"still failing"}, nil },
		nil, nil,
	)
	if err != nil {
		t.Fatalf("round: %v", err)
	}
	if outcome.Passed || !outcome.Restored {
		t.Fatalf("expected a failed, restored round: %+v", outcome)
	}

	after, err := repairCheckpointDirectoryDigest(root, scope)
	if err != nil {
		t.Fatalf("digest after: %v", err)
	}
	if before != after {
		t.Fatalf("restored working tree is not byte-identical: before=%s after=%s", before, after)
	}

	passLedger := newAutopilotRepairLedger("run-201-09-passed", 1)
	passInput := workRepairFixtureInput(root, 11, "go vet ./cmd")
	passOutcome, err := runBoundedRepairRound(&passLedger, passInput, nil, noopPersist,
		func() error {
			return os.WriteFile(filepath.Join(root, "cmd", "foo.go"), []byte("package cmd\n\nfunc Foo() int { return 3 }\n"), 0o644)
		},
		func() (bool, []string, error) { return true, nil, nil },
		nil, nil,
	)
	if err != nil {
		t.Fatalf("passed round: %v", err)
	}
	if !passOutcome.Passed || passOutcome.Restored {
		t.Fatalf("expected a passed, non-restored round: %+v", passOutcome)
	}
	if passOutcome.Receipt.Status != autopilotRepairPassed {
		t.Fatalf("passed round did not record a passed receipt: %+v", passOutcome.Receipt)
	}
	data, err := os.ReadFile(filepath.Join(root, "cmd", "foo.go"))
	if err != nil {
		t.Fatalf("read repaired file: %v", err)
	}
	if !strings.Contains(string(data), "return 3") {
		t.Fatalf("repaired file was not left in place after a passed repair: %s", data)
	}
}

// TestRepairReplayReturnsTheExistingReceipt proves replaying the same
// checkpoint identity returns the existing receipt identifier rather than
// performing a second repair wave or re-verification.
func TestRepairReplayReturnsTheExistingReceipt(t *testing.T) {
	root := workRepairFixtureRoot(t)
	ledger := newAutopilotRepairLedger("run-201-09-replay", 2)
	input := workRepairFixtureInput(root, 12, "go test ./cmd")

	first, err := runBoundedRepairRound(&ledger, input, nil, noopPersist,
		func() error { return nil },
		func() (bool, []string, error) { return true, nil, nil },
		nil, nil,
	)
	if err != nil {
		t.Fatalf("first round: %v", err)
	}
	if first.Receipt.ID == "" {
		t.Fatalf("first round produced no receipt id: %+v", first)
	}

	replay, err := runBoundedRepairRound(&ledger, input, nil, noopPersist,
		func() error { t.Fatal("replay must not run a repair wave"); return nil },
		func() (bool, []string, error) { t.Fatal("replay must not re-verify"); return false, nil, nil },
		nil, nil,
	)
	if err != nil {
		t.Fatalf("replay: %v", err)
	}
	if !replay.Replayed {
		t.Fatalf("replay was not reported as a replay: %+v", replay)
	}
	if replay.Receipt.ID != first.Receipt.ID {
		t.Fatalf("replay returned a different receipt id: first=%q replay=%q", first.Receipt.ID, replay.Receipt.ID)
	}
}

// TestRepairNeverOverwritesTheOriginalResult proves the original failing
// attempt record is unchanged, the repair record names it as parent, and an
// independent, later repair round appends a new receipt without mutating an
// earlier one.
func TestRepairNeverOverwritesTheOriginalResult(t *testing.T) {
	root := workRepairFixtureRoot(t)
	type fixtureOriginalAttempt struct {
		ID     string
		Status string
	}
	original := fixtureOriginalAttempt{ID: "attempt-original-1", Status: "failed"}

	ledger := newAutopilotRepairLedger("run-201-09-parent", 2)
	input := workRepairFixtureInput(root, 13, "go test ./cmd")
	input.Attempt = original.ID

	outcome, err := runBoundedRepairRound(&ledger, input, nil, noopPersist,
		func() error { return nil },
		func() (bool, []string, error) { return true, nil, nil },
		nil, nil,
	)
	if err != nil {
		t.Fatalf("round: %v", err)
	}

	if original.ID != "attempt-original-1" || original.Status != "failed" {
		t.Fatalf("original attempt record was mutated: %+v", original)
	}
	if outcome.Receipt.Attempt != original.ID {
		t.Fatalf("repair record does not name its parent attempt: got %q, want %q", outcome.Receipt.Attempt, original.ID)
	}

	secondInput := workRepairFixtureInput(root, 13, "go vet ./cmd")
	secondInput.Attempt = original.ID
	before := ledger.Receipts[0]
	if _, err := runBoundedRepairRound(&ledger, secondInput, nil, noopPersist,
		func() error { return nil },
		func() (bool, []string, error) { return true, nil, nil },
		nil, nil,
	); err != nil {
		t.Fatalf("second, independent round: %v", err)
	}
	if len(ledger.Receipts) != 2 {
		t.Fatalf("expected the ledger to append a second receipt, got %d", len(ledger.Receipts))
	}
	if !reflect.DeepEqual(ledger.Receipts[0], before) {
		t.Fatalf("first receipt was mutated by a later, independent repair round:\nbefore=%+v\nafter=%+v", before, ledger.Receipts[0])
	}
}

// TestCheckpointSaveAndRestoreAreAnnouncedOnBothLanes proves the save
// announcement precedes the repair dispatch and the restore announcement
// follows a failed re-verification, on both the direct check lane and the
// plan-only-plus-finalize lane -- both reach runBoundedRepairRound, the one
// bounded repair path build and check both use, so the same proof applies
// to either caller shape.
func TestCheckpointSaveAndRestoreAreAnnouncedOnBothLanes(t *testing.T) {
	t.Setenv("AETHER_OUTPUT_MODE", "visual")
	original := stdout
	defer func() { stdout = original }()

	lanes := []struct {
		name  string
		phase int
		check string
	}{
		{"direct check lane", 21, "go test ./cmd"},
		{"plan-only-plus-finalize lane", 22, "go vet ./cmd"},
	}
	for _, lane := range lanes {
		t.Run(lane.name, func(t *testing.T) {
			var buf bytes.Buffer
			stdout = &buf

			root := workRepairFixtureRoot(t)
			ledger := newAutopilotRepairLedger("run-201-09-"+lane.check, 1)
			input := workRepairFixtureInput(root, lane.phase, lane.check)

			var order []string
			outcome, err := runBoundedRepairRound(&ledger, input, nil, noopPersist,
				func() error { order = append(order, "repair"); return nil },
				func() (bool, []string, error) { order = append(order, "verify"); return false, []string{"still failing"}, nil },
				func(id string) { order = append(order, "save:"+id); emitRepairCheckpointSaved(lane.phase, lane.check) },
				func(id string) { order = append(order, "restore:"+id); emitRepairCheckpointRestored(lane.phase, lane.check) },
			)
			if err != nil {
				t.Fatalf("round: %v", err)
			}
			if !outcome.Restored {
				t.Fatalf("expected a restored round: %+v", outcome)
			}

			wantOrder := []string{"save:" + outcome.CheckpointID, "repair", "verify", "restore:" + outcome.CheckpointID}
			if strings.Join(order, ",") != strings.Join(wantOrder, ",") {
				t.Fatalf("%s: announcement/repair ordering = %v, want %v", lane.name, order, wantOrder)
			}

			visual := buf.String()
			if !strings.Contains(visual, "Saving your project's current state") {
				t.Fatalf("%s: save announcement missing from the rendered flow:\n%s", lane.name, visual)
			}
			if !strings.Contains(visual, "put back exactly to the state it was saved in") {
				t.Fatalf("%s: restore announcement missing from the rendered flow:\n%s", lane.name, visual)
			}
			savedAt := strings.Index(visual, "Saving your project's current state")
			restoredAt := strings.Index(visual, "put back exactly to the state it was saved in")
			if savedAt < 0 || restoredAt < 0 || savedAt > restoredAt {
				t.Fatalf("%s: save announcement must precede restore in the rendered flow:\n%s", lane.name, visual)
			}

			lowered := strings.ToLower(visual)
			for _, jargon := range []string{"verification boundary", "the queen", "post-wave", "deterministic floor", "caste"} {
				if strings.Contains(lowered, jargon) {
					t.Errorf("%s: rendered announcement contains repository-invented vocabulary %q:\n%s", lane.name, jargon, visual)
				}
			}
		})
	}
}

// TestPassedRepairAnnouncesNoRestore proves a passed repair still announces
// the save but never announces a restore.
func TestPassedRepairAnnouncesNoRestore(t *testing.T) {
	t.Setenv("AETHER_OUTPUT_MODE", "visual")
	original := stdout
	defer func() { stdout = original }()
	var buf bytes.Buffer
	stdout = &buf

	root := workRepairFixtureRoot(t)
	ledger := newAutopilotRepairLedger("run-201-09-pass-announce", 1)
	input := workRepairFixtureInput(root, 23, "go build ./cmd")

	outcome, err := runBoundedRepairRound(&ledger, input, nil, noopPersist,
		func() error { return nil },
		func() (bool, []string, error) { return true, nil, nil },
		func(string) { emitRepairCheckpointSaved(input.Phase, input.Check) },
		func(string) { emitRepairCheckpointRestored(input.Phase, input.Check) },
	)
	if err != nil {
		t.Fatalf("round: %v", err)
	}
	if !outcome.Passed || outcome.Restored {
		t.Fatalf("expected a passed, non-restored round: %+v", outcome)
	}

	visual := buf.String()
	if !strings.Contains(visual, "Saving your project's current state") {
		t.Fatalf("save announcement missing: %s", visual)
	}
	if strings.Contains(visual, "put back exactly to the state it was saved in") {
		t.Fatalf("a passed repair unexpectedly announced a restore:\n%s", visual)
	}
}
