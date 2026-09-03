package cmd

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

var errLifecycleTransactionInjectedFault = errors.New("injected lifecycle transaction fault")

type lifecycleTransactionTestTarget struct {
	path   string
	before lifecycleTransactionTestSnapshot
	after  lifecycleTransactionTestSnapshot
}

type lifecycleTransactionTestSnapshot struct {
	exists  bool
	content []byte
}

func prepareLifecycleFaultTransaction(t *testing.T, id, faultPoint string) (lifecycleTransactionFixture, lifecycleTransactionConfig, *lifecycleTransaction, []lifecycleTransactionTestTarget) {
	t.Helper()
	fixture := newLifecycleTransactionFixture(t)
	repositoryWrite := filepath.Join(fixture.repositoryRoot, "existing.txt")
	repositoryRemove := filepath.Join(fixture.repositoryRoot, "remove.txt")
	hubWrite := filepath.Join(fixture.hubRoot, "nested", "hub.txt")
	mustWriteLifecycleFixtureFile(t, repositoryWrite, []byte("repository-before"))
	mustWriteLifecycleFixtureFile(t, repositoryRemove, []byte("remove-before"))

	config := fixture.config(id)
	if faultPoint != "" {
		config.Fault = func(point string) error {
			if point == faultPoint {
				return errLifecycleTransactionInjectedFault
			}
			return nil
		}
	}
	tx, err := beginLifecycleTransaction(config)
	if err != nil {
		t.Fatalf("begin transaction: %v", err)
	}
	if err := tx.DeclareWrite(lifecycleTransactionRootRepository, "existing.txt", []byte("repository-after")); err != nil {
		t.Fatalf("declare repository write: %v", err)
	}
	if err := tx.DeclareRemoval(lifecycleTransactionRootRepository, "remove.txt"); err != nil {
		t.Fatalf("declare repository removal: %v", err)
	}
	if err := tx.DeclareWrite(lifecycleTransactionRootHubStable, "nested/hub.txt", []byte("hub-after")); err != nil {
		t.Fatalf("declare hub write: %v", err)
	}
	targets := []lifecycleTransactionTestTarget{
		{
			path:   repositoryWrite,
			before: lifecycleTransactionTestSnapshot{exists: true, content: []byte("repository-before")},
			after:  lifecycleTransactionTestSnapshot{exists: true, content: []byte("repository-after")},
		},
		{
			path:   repositoryRemove,
			before: lifecycleTransactionTestSnapshot{exists: true, content: []byte("remove-before")},
			after:  lifecycleTransactionTestSnapshot{},
		},
		{
			path:   hubWrite,
			before: lifecycleTransactionTestSnapshot{},
			after:  lifecycleTransactionTestSnapshot{exists: true, content: []byte("hub-after")},
		},
	}
	return fixture, config, tx, targets
}

func assertLifecycleTransactionSnapshot(t *testing.T, targets []lifecycleTransactionTestTarget, after bool) {
	t.Helper()
	for _, target := range targets {
		want := target.before
		if after {
			want = target.after
		}
		content, err := os.ReadFile(target.path)
		if !want.exists {
			if !os.IsNotExist(err) {
				t.Fatalf("target %s exists or returned unexpected error %v; want absent", target.path, err)
			}
			continue
		}
		if err != nil {
			t.Fatalf("read target %s: %v", target.path, err)
		}
		if !bytes.Equal(content, want.content) {
			t.Fatalf("target %s = %q, want %q", target.path, content, want.content)
		}
	}
}

func TestLifecycleTransactionFaultMatrix(t *testing.T) {
	faultPoints := []string{
		"after_validation",
		"after_stage:target-0001",
		"after_stage:target-0002",
		"after_stage:target-0003",
		"after_intent",
		"after_target_commit:target-0001",
		"after_target_commit:target-0002",
		"after_root_commit:root-01-repository",
		"after_target_commit:target-0003",
		"after_root_commit:root-02-hub_stable",
		"after_global_verification",
	}

	for _, faultPoint := range faultPoints {
		t.Run(strings.ReplaceAll(faultPoint, ":", "_"), func(t *testing.T) {
			id := "fault-" + strings.NewReplacer(":", "-", "_", "-").Replace(faultPoint)
			fixture, config, tx, targets := prepareLifecycleFaultTransaction(t, id, faultPoint)
			if _, err := tx.Commit(); !errors.Is(err, errLifecycleTransactionInjectedFault) {
				t.Fatalf("commit error = %v, want injected fault at %s", err, faultPoint)
			}

			intentPath := filepath.Join(lifecycleJournalPath(fixture, id), "intent.json")
			if _, err := os.Stat(intentPath); os.IsNotExist(err) {
				assertLifecycleTransactionSnapshot(t, targets, false)
				return
			} else if err != nil {
				t.Fatalf("inspect intent: %v", err)
			}

			config.Fault = nil
			receipt, err := resumeLifecycleTransaction(config)
			if err != nil {
				t.Fatalf("resume after %s: %v", faultPoint, err)
			}
			if receipt.Transaction.Stage != colony.TransactionStageVerified || receipt.StateEffect != colony.LifecycleStateEffectCommitted {
				t.Fatalf("resume receipt = %#v", receipt)
			}
			assertLifecycleTransactionSnapshot(t, targets, true)

			replayed, err := resumeLifecycleTransaction(config)
			if err != nil {
				t.Fatalf("replay after resume: %v", err)
			}
			if !reflect.DeepEqual(replayed, receipt) {
				t.Fatalf("replay returned a different receipt\nfirst: %#v\nreplay: %#v", receipt, replayed)
			}
			receipts, err := filepath.Glob(filepath.Join(lifecycleJournalPath(fixture, id), "receipt.json"))
			if err != nil || len(receipts) != 1 {
				t.Fatalf("receipt count = %d, err=%v", len(receipts), err)
			}
		})
	}
}

func TestLifecycleTransactionReplayExactlyOnce(t *testing.T) {
	fixture := newLifecycleTransactionFixture(t)
	target := filepath.Join(fixture.repositoryRoot, "once.txt")
	mustWriteLifecycleFixtureFile(t, target, []byte("before"))

	renames := 0
	config := fixture.config("replay-exactly-once")
	config.Rename = func(oldPath, newPath string) error {
		if newPath == target {
			renames++
		}
		return os.Rename(oldPath, newPath)
	}
	tx, err := beginLifecycleTransaction(config)
	if err != nil {
		t.Fatal(err)
	}
	if err := tx.DeclareWrite(lifecycleTransactionRootRepository, "once.txt", []byte("after")); err != nil {
		t.Fatal(err)
	}
	first, err := tx.Commit()
	if err != nil {
		t.Fatal(err)
	}
	if renames != 1 {
		t.Fatalf("initial effect count = %d, want 1", renames)
	}

	second, err := tx.Commit()
	if err != nil {
		t.Fatal(err)
	}
	third, err := resumeLifecycleTransaction(config)
	if err != nil {
		t.Fatal(err)
	}
	replayedTx, err := beginLifecycleTransaction(config)
	if err != nil {
		t.Fatal(err)
	}
	fourth, err := replayedTx.Commit()
	if err != nil {
		t.Fatal(err)
	}
	for index, receipt := range []colony.LifecycleReceipt{second, third, fourth} {
		if !reflect.DeepEqual(receipt, first) {
			t.Fatalf("replay %d returned a different receipt", index+1)
		}
	}
	if renames != 1 {
		t.Fatalf("replay applied effect %d times, want exactly once", renames)
	}
	if got := string(mustReadLifecycleFixtureFile(t, target)); got != "after" {
		t.Fatalf("target = %q, want after", got)
	}
}

func TestLifecycleTransactionTamperConflict(t *testing.T) {
	tests := []struct {
		name       string
		faultPoint string
		provenance colony.RecoveryProvenance
		tamper     func(t *testing.T, fixture lifecycleTransactionFixture, id string, targets []lifecycleTransactionTestTarget) func()
	}{
		{
			name:       "staged bytes",
			faultPoint: "after_intent",
			provenance: colony.RecoveryProvenanceConflicting,
			tamper: func(t *testing.T, fixture lifecycleTransactionFixture, id string, _ []lifecycleTransactionTestTarget) func() {
				targets := readLifecycleTransactionManifestTargets(t, fixture, id)
				mustWriteLifecycleFixtureFile(t, targets["target-0001"].StagePath, []byte("tampered-stage"))
				return func() {
					if got := string(mustReadLifecycleFixtureFile(t, targets["target-0001"].StagePath)); got != "tampered-stage" {
						t.Fatalf("staged conflict evidence was replaced: %q", got)
					}
				}
			},
		},
		{
			name:       "missing staged bytes",
			faultPoint: "after_intent",
			provenance: colony.RecoveryProvenanceUnknown,
			tamper: func(t *testing.T, fixture lifecycleTransactionFixture, id string, _ []lifecycleTransactionTestTarget) func() {
				targets := readLifecycleTransactionManifestTargets(t, fixture, id)
				if err := os.Remove(targets["target-0001"].StagePath); err != nil {
					t.Fatal(err)
				}
				return func() {
					if _, err := os.Stat(targets["target-0001"].StagePath); !os.IsNotExist(err) {
						t.Fatalf("missing stage evidence was recreated: %v", err)
					}
				}
			},
		},
		{
			name:       "symlinked staged bytes",
			faultPoint: "after_intent",
			provenance: colony.RecoveryProvenanceConflicting,
			tamper: func(t *testing.T, fixture lifecycleTransactionFixture, id string, _ []lifecycleTransactionTestTarget) func() {
				targets := readLifecycleTransactionManifestTargets(t, fixture, id)
				stagePath := targets["target-0001"].StagePath
				movedPath := stagePath + ".moved"
				if err := os.Rename(stagePath, movedPath); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(movedPath, stagePath); err != nil {
					t.Fatal(err)
				}
				return func() {
					info, err := os.Lstat(stagePath)
					if err != nil || info.Mode()&os.ModeSymlink == 0 {
						t.Fatalf("symlink conflict evidence was replaced: info=%v err=%v", info, err)
					}
				}
			},
		},
		{
			name:       "intent",
			faultPoint: "after_intent",
			provenance: colony.RecoveryProvenanceConflicting,
			tamper: func(t *testing.T, fixture lifecycleTransactionFixture, id string, _ []lifecycleTransactionTestTarget) func() {
				path := filepath.Join(lifecycleJournalPath(fixture, id), "intent.json")
				content := mustReadLifecycleFixtureFile(t, path)
				tampered := bytes.Replace(content, []byte("test lifecycle transaction"), []byte("tampered lifecycle transaction"), 1)
				if bytes.Equal(content, tampered) {
					t.Fatal("intent fixture did not contain command")
				}
				mustWriteLifecycleFixtureFile(t, path, tampered)
				return func() {
					if got := mustReadLifecycleFixtureFile(t, path); !bytes.Equal(got, tampered) {
						t.Fatal("tampered intent evidence was replaced")
					}
				}
			},
		},
		{
			name:       "preimage",
			faultPoint: "after_intent",
			provenance: colony.RecoveryProvenanceConflicting,
			tamper: func(t *testing.T, fixture lifecycleTransactionFixture, id string, _ []lifecycleTransactionTestTarget) func() {
				targets := readLifecycleTransactionManifestTargets(t, fixture, id)
				mustWriteLifecycleFixtureFile(t, targets["target-0001"].PreimagePath, []byte("tampered-preimage"))
				return func() {
					if got := string(mustReadLifecycleFixtureFile(t, targets["target-0001"].PreimagePath)); got != "tampered-preimage" {
						t.Fatalf("preimage conflict evidence was replaced: %q", got)
					}
				}
			},
		},
		{
			name:       "committed target",
			faultPoint: "after_target_commit:target-0001",
			provenance: colony.RecoveryProvenanceConflicting,
			tamper: func(t *testing.T, _ lifecycleTransactionFixture, _ string, targets []lifecycleTransactionTestTarget) func() {
				mustWriteLifecycleFixtureFile(t, targets[0].path, []byte("hostile-target"))
				return func() {
					if got := string(mustReadLifecycleFixtureFile(t, targets[0].path)); got != "hostile-target" {
						t.Fatalf("conflicting target was overwritten: %q", got)
					}
				}
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			id := "tamper-" + strings.ReplaceAll(test.name, " ", "-")
			fixture, config, tx, targets := prepareLifecycleFaultTransaction(t, id, test.faultPoint)
			if _, err := tx.Commit(); !errors.Is(err, errLifecycleTransactionInjectedFault) {
				t.Fatalf("commit error = %v, want injected fault", err)
			}
			assertPreserved := test.tamper(t, fixture, id, targets)
			config.Fault = nil
			receipt, err := resumeLifecycleTransaction(config)
			if err == nil {
				t.Fatal("tampered transaction resumed without conflict")
			}
			if receipt.OutcomeKind != colony.OutcomeKindRecoveryRequired || receipt.StateEffect != colony.LifecycleStateEffectRecoveryRequired {
				t.Fatalf("tamper receipt outcome/state = %q/%q", receipt.OutcomeKind, receipt.StateEffect)
			}
			if receipt.Recovery == nil || receipt.Recovery.Provenance != test.provenance || receipt.Provenance != test.provenance {
				t.Fatalf("tamper provenance = receipt:%q recovery:%#v, want %q", receipt.Provenance, receipt.Recovery, test.provenance)
			}
			if validateErr := receipt.Validate(); validateErr != nil {
				t.Fatalf("recovery receipt invalid: %v", validateErr)
			}
			assertPreserved()
			if _, statErr := os.Stat(filepath.Join(lifecycleJournalPath(fixture, id), "receipt.json")); !os.IsNotExist(statErr) {
				t.Fatalf("tamper conflict invented committed receipt: %v", statErr)
			}
		})
	}
}

func readLifecycleTransactionManifestTargets(t *testing.T, fixture lifecycleTransactionFixture, id string) map[string]lifecycleTransactionTargetManifest {
	t.Helper()
	intentBytes := mustReadLifecycleFixtureFile(t, filepath.Join(lifecycleJournalPath(fixture, id), "intent.json"))
	var intent lifecycleTransactionIntent
	if err := decodeLifecycleJSON(intentBytes, &intent); err != nil {
		t.Fatalf("decode intent: %v", err)
	}
	result := make(map[string]lifecycleTransactionTargetManifest)
	for _, root := range intent.Roots {
		manifestBytes := mustReadLifecycleFixtureFile(t, root.ManifestPath)
		var manifest lifecycleTransactionRootManifest
		if err := decodeLifecycleJSON(manifestBytes, &manifest); err != nil {
			t.Fatalf("decode root manifest: %v", err)
		}
		for _, target := range manifest.Targets {
			result[target.ID] = target
		}
	}
	return result
}
