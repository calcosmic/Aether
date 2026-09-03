package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

type lifecycleTransactionFixture struct {
	repositoryRoot    string
	dataRoot          string
	hubRoot           string
	claudeHome        string
	openCodeHome      string
	codexHome         string
	binaryDestination string
}

func newLifecycleTransactionFixture(t *testing.T) lifecycleTransactionFixture {
	t.Helper()

	base := t.TempDir()
	fixture := lifecycleTransactionFixture{
		repositoryRoot:    filepath.Join(base, "repository"),
		dataRoot:          filepath.Join(base, "lifecycle-data"),
		hubRoot:           filepath.Join(base, "stable-hub"),
		claudeHome:        filepath.Join(base, "claude-home"),
		openCodeHome:      filepath.Join(base, "opencode-home"),
		codexHome:         filepath.Join(base, "codex-home"),
		binaryDestination: filepath.Join(base, "bin", "aether"),
	}

	for _, directory := range []string{
		fixture.repositoryRoot,
		fixture.dataRoot,
		fixture.hubRoot,
		fixture.claudeHome,
		fixture.openCodeHome,
		fixture.codexHome,
		filepath.Dir(fixture.binaryDestination),
	} {
		if err := os.MkdirAll(directory, 0o755); err != nil {
			t.Fatalf("create fixture directory %s: %v", directory, err)
		}
	}

	return fixture
}

func (fixture lifecycleTransactionFixture) config(id string) lifecycleTransactionConfig {
	return lifecycleTransactionConfig{
		TransactionID: id,
		Command:       "test lifecycle transaction",
		Allowlist: lifecycleTransactionAllowlist{
			RepositoryRoot:    fixture.repositoryRoot,
			LifecycleDataRoot: fixture.dataRoot,
			Hub: lifecycleTransactionHubRoot{
				Channel: lifecycleTransactionHubStable,
				Path:    fixture.hubRoot,
			},
			ClaudeHome:        fixture.claudeHome,
			OpenCodeHome:      fixture.openCodeHome,
			CodexHome:         fixture.codexHome,
			BinaryDestination: fixture.binaryDestination,
		},
	}
}

func mustWriteLifecycleFixtureFile(t *testing.T, path string, content []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("create parent for %s: %v", path, err)
	}
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("write fixture file %s: %v", path, err)
	}
}

func mustReadLifecycleFixtureFile(t *testing.T, path string) []byte {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture file %s: %v", path, err)
	}
	return content
}

func lifecycleJournalPath(fixture lifecycleTransactionFixture, id string) string {
	return filepath.Join(fixture.dataRoot, "transactions", id)
}

func TestLifecycleTransactionCommit(t *testing.T) {
	fixture := newLifecycleTransactionFixture(t)
	repositoryTarget := filepath.Join(fixture.repositoryRoot, ".aether", "settings.json")
	dataTarget := filepath.Join(fixture.dataRoot, "COLONY_STATE.json")
	mustWriteLifecycleFixtureFile(t, repositoryTarget, []byte("repository-before"))

	tx, err := beginLifecycleTransaction(fixture.config("commit"))
	if err != nil {
		t.Fatalf("begin transaction: %v", err)
	}
	if err := tx.DeclareWrite(lifecycleTransactionRootRepository, ".aether/settings.json", []byte("repository-after")); err != nil {
		t.Fatalf("declare repository write: %v", err)
	}
	if err := tx.DeclareWrite(lifecycleTransactionRootData, "COLONY_STATE.json", []byte("data-after")); err != nil {
		t.Fatalf("declare data write: %v", err)
	}
	if err := tx.Validate(); err != nil {
		t.Fatalf("validate transaction: %v", err)
	}
	if _, err := os.Stat(filepath.Join(lifecycleJournalPath(fixture, "commit"), "intent.json")); !os.IsNotExist(err) {
		t.Fatalf("validation made intent authoritative before staging: %v", err)
	}

	receipt, err := tx.Commit()
	if err != nil {
		t.Fatalf("commit transaction: %v", err)
	}
	if got := string(mustReadLifecycleFixtureFile(t, repositoryTarget)); got != "repository-after" {
		t.Fatalf("repository target = %q, want repository-after", got)
	}
	if got := string(mustReadLifecycleFixtureFile(t, dataTarget)); got != "data-after" {
		t.Fatalf("data target = %q, want data-after", got)
	}
	if err := receipt.Validate(); err != nil {
		t.Fatalf("receipt does not satisfy lifecycle contract: %v", err)
	}
	if receipt.OutcomeKind != colony.OutcomeKindCompleted || receipt.StateEffect != colony.LifecycleStateEffectCommitted {
		t.Fatalf("receipt outcome/state = %q/%q, want completed/committed", receipt.OutcomeKind, receipt.StateEffect)
	}
	if receipt.Transaction == nil || receipt.Transaction.Stage != colony.TransactionStageVerified {
		t.Fatalf("receipt transaction = %#v, want verified transaction", receipt.Transaction)
	}
	for _, name := range []string{"intent.json", "progress.json", "receipt.json", "receipt.sha256"} {
		if _, err := os.Stat(filepath.Join(lifecycleJournalPath(fixture, "commit"), name)); err != nil {
			t.Fatalf("journal artifact %s: %v", name, err)
		}
	}
}

func TestLifecycleTransactionRejectsInvalidTarget(t *testing.T) {
	tests := []struct {
		name    string
		prepare func(t *testing.T, fixture lifecycleTransactionFixture) string
		target  func(fixture lifecycleTransactionFixture) string
	}{
		{
			name: "absolute",
			target: func(fixture lifecycleTransactionFixture) string {
				return filepath.Join(fixture.repositoryRoot, "absolute")
			},
		},
		{
			name:   "parent traversal",
			target: func(lifecycleTransactionFixture) string { return "../escape" },
		},
		{
			name: "symlink ancestor",
			prepare: func(t *testing.T, fixture lifecycleTransactionFixture) string {
				outside := filepath.Join(t.TempDir(), "outside")
				if err := os.MkdirAll(outside, 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(outside, filepath.Join(fixture.repositoryRoot, "linked")); err != nil {
					t.Fatal(err)
				}
				return "linked/escape"
			},
		},
		{
			name: "symlink target",
			prepare: func(t *testing.T, fixture lifecycleTransactionFixture) string {
				outside := filepath.Join(t.TempDir(), "outside")
				mustWriteLifecycleFixtureFile(t, outside, []byte("outside"))
				if err := os.Symlink(outside, filepath.Join(fixture.repositoryRoot, "linked-file")); err != nil {
					t.Fatal(err)
				}
				return "linked-file"
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fixture := newLifecycleTransactionFixture(t)
			target := "unused"
			if test.prepare != nil {
				target = test.prepare(t, fixture)
			} else {
				target = test.target(fixture)
			}
			tx, err := beginLifecycleTransaction(fixture.config("invalid-target"))
			if err != nil {
				t.Fatalf("begin transaction: %v", err)
			}
			err = tx.DeclareWrite(lifecycleTransactionRootRepository, target, []byte("unsafe"))
			if err == nil {
				err = tx.Validate()
			}
			if err == nil {
				t.Fatal("invalid target was accepted")
			}
			if _, statErr := os.Stat(filepath.Join(lifecycleJournalPath(fixture, "invalid-target"), "intent.json")); !os.IsNotExist(statErr) {
				t.Fatalf("invalid transaction wrote authoritative intent: %v", statErr)
			}
		})
	}

	t.Run("duplicate", func(t *testing.T) {
		fixture := newLifecycleTransactionFixture(t)
		tx, err := beginLifecycleTransaction(fixture.config("duplicate"))
		if err != nil {
			t.Fatalf("begin transaction: %v", err)
		}
		if err := tx.DeclareWrite(lifecycleTransactionRootRepository, "same", []byte("one")); err != nil {
			t.Fatalf("declare first target: %v", err)
		}
		if err := tx.DeclareWrite(lifecycleTransactionRootRepository, "same", []byte("two")); err == nil {
			t.Fatal("duplicate target was accepted")
		}
	})
}

func TestLifecycleTransactionRejectsChangedBaseline(t *testing.T) {
	fixture := newLifecycleTransactionFixture(t)
	target := filepath.Join(fixture.repositoryRoot, "mutable.txt")
	mustWriteLifecycleFixtureFile(t, target, []byte("declared-baseline"))

	tx, err := beginLifecycleTransaction(fixture.config("changed-baseline"))
	if err != nil {
		t.Fatalf("begin transaction: %v", err)
	}
	if err := tx.DeclareWrite(lifecycleTransactionRootRepository, "mutable.txt", []byte("transaction-value")); err != nil {
		t.Fatalf("declare write: %v", err)
	}
	mustWriteLifecycleFixtureFile(t, target, []byte("external-change"))

	if _, err := tx.Commit(); err == nil || !strings.Contains(err.Error(), "baseline") {
		t.Fatalf("commit error = %v, want changed baseline refusal", err)
	}
	if got := string(mustReadLifecycleFixtureFile(t, target)); got != "external-change" {
		t.Fatalf("changed target = %q, transaction mutated it", got)
	}
	if _, err := os.Stat(filepath.Join(lifecycleJournalPath(fixture, "changed-baseline"), "intent.json")); !os.IsNotExist(err) {
		t.Fatalf("changed baseline wrote authoritative intent: %v", err)
	}
}

func TestLifecycleTransactionReceipt(t *testing.T) {
	fixture := newLifecycleTransactionFixture(t)
	mustWriteLifecycleFixtureFile(t, filepath.Join(fixture.repositoryRoot, "remove-me"), []byte("remove-before"))

	tx, err := beginLifecycleTransaction(fixture.config("receipt"))
	if err != nil {
		t.Fatal(err)
	}
	if err := tx.DeclareWrite(lifecycleTransactionRootHubStable, "installed/file", []byte("hub-after")); err != nil {
		t.Fatal(err)
	}
	if err := tx.DeclareRemoval(lifecycleTransactionRootRepository, "remove-me"); err != nil {
		t.Fatal(err)
	}
	receipt, err := tx.Commit()
	if err != nil {
		t.Fatal(err)
	}

	if receipt.ReceiptID != "receipt-receipt" {
		t.Fatalf("receipt id = %q, want receipt-receipt", receipt.ReceiptID)
	}
	if len(receipt.Changes) != 2 || len(receipt.Verification) != 2 {
		t.Fatalf("receipt changes/verifications = %d/%d, want 2/2", len(receipt.Changes), len(receipt.Verification))
	}
	for _, verification := range receipt.Verification {
		if !verification.Passed || len(verification.EvidenceIDs) == 0 {
			t.Fatalf("unproved verification: %#v", verification)
		}
	}
	joinedEvidence := ""
	for _, evidence := range receipt.Evidence {
		joinedEvidence += evidence.Summary + " " + evidence.Source + "\n"
	}
	for _, rootName := range []string{string(lifecycleTransactionRootRepository), string(lifecycleTransactionRootHubStable)} {
		if !strings.Contains(joinedEvidence, rootName) {
			t.Fatalf("receipt evidence does not name root %q: %s", rootName, joinedEvidence)
		}
	}
	if receipt.Recovery == nil || !strings.Contains(receipt.Recovery.SafeNextStep, "aether resume") {
		t.Fatalf("receipt recovery command = %#v, want aether resume guidance", receipt.Recovery)
	}
	if receipt.Transaction == nil || receipt.Transaction.JournalPath != lifecycleJournalPath(fixture, "receipt") {
		t.Fatalf("receipt journal reference = %#v", receipt.Transaction)
	}
}

func TestLifecycleTransactionRootAllowlist(t *testing.T) {
	t.Run("relative repository root", func(t *testing.T) {
		fixture := newLifecycleTransactionFixture(t)
		config := fixture.config("relative-root")
		config.Allowlist.RepositoryRoot = "relative"
		if _, err := beginLifecycleTransaction(config); err == nil {
			t.Fatal("relative root was accepted")
		}
	})

	t.Run("symlink repository root", func(t *testing.T) {
		fixture := newLifecycleTransactionFixture(t)
		realRoot := filepath.Join(t.TempDir(), "real")
		if err := os.MkdirAll(realRoot, 0o755); err != nil {
			t.Fatal(err)
		}
		linkedRoot := filepath.Join(t.TempDir(), "linked")
		if err := os.Symlink(realRoot, linkedRoot); err != nil {
			t.Fatal(err)
		}
		config := fixture.config("symlink-root")
		config.Allowlist.RepositoryRoot = linkedRoot
		if _, err := beginLifecycleTransaction(config); err == nil {
			t.Fatal("symlink root was accepted")
		}
	})

	tests := []struct {
		name      string
		configure func(*lifecycleTransactionConfig)
		root      lifecycleTransactionRootKind
		target    string
	}{
		{name: "unknown root", root: lifecycleTransactionRootKind("arbitrary-root"), target: "file"},
		{
			name:      "unconfigured codex home",
			configure: func(config *lifecycleTransactionConfig) { config.Allowlist.CodexHome = "" },
			root:      lifecycleTransactionRootCodexHome,
			target:    "file",
		},
		{name: "unselected dev hub", root: lifecycleTransactionRootHubDev, target: "file"},
		{name: "non-exact binary", root: lifecycleTransactionRootBinaryDestination, target: "another-binary"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fixture := newLifecycleTransactionFixture(t)
			config := fixture.config("root-allowlist")
			if test.configure != nil {
				test.configure(&config)
			}
			tx, err := beginLifecycleTransaction(config)
			if err != nil {
				t.Fatalf("begin transaction: %v", err)
			}
			if err := tx.DeclareWrite(test.root, test.target, []byte("unsafe")); err == nil {
				t.Fatal("unallowlisted root or target was accepted")
			}
		})
	}
}

func TestLifecycleTransactionPerRootStaging(t *testing.T) {
	fixture := newLifecycleTransactionFixture(t)
	mustWriteLifecycleFixtureFile(t, filepath.Join(fixture.repositoryRoot, "repo-file"), []byte("repo-before"))
	mustWriteLifecycleFixtureFile(t, filepath.Join(fixture.hubRoot, "hub-file"), []byte("hub-before"))

	tx, err := beginLifecycleTransaction(fixture.config("per-root-staging"))
	if err != nil {
		t.Fatal(err)
	}
	if err := tx.DeclareWrite(lifecycleTransactionRootRepository, "repo-file", []byte("repo-after")); err != nil {
		t.Fatal(err)
	}
	if err := tx.DeclareWrite(lifecycleTransactionRootHubStable, "hub-file", []byte("hub-after")); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Commit(); err != nil {
		t.Fatal(err)
	}

	intentBytes := mustReadLifecycleFixtureFile(t, filepath.Join(lifecycleJournalPath(fixture, "per-root-staging"), "intent.json"))
	var intent struct {
		Roots []struct {
			Kind         string `json:"kind"`
			RootPath     string `json:"root_path"`
			ManifestPath string `json:"manifest_path"`
		} `json:"roots"`
	}
	if err := json.Unmarshal(intentBytes, &intent); err != nil {
		t.Fatalf("decode intent: %v", err)
	}
	if len(intent.Roots) != 2 {
		t.Fatalf("intent roots = %d, want 2", len(intent.Roots))
	}
	for _, root := range intent.Roots {
		if !pathIsWithin(root.RootPath, root.ManifestPath) {
			t.Fatalf("manifest %s is not local to root %s", root.ManifestPath, root.RootPath)
		}
		manifestBytes := mustReadLifecycleFixtureFile(t, root.ManifestPath)
		var manifest struct {
			Targets []struct {
				StagePath    string `json:"stage_path"`
				PreimagePath string `json:"preimage_path"`
			} `json:"targets"`
		}
		if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
			t.Fatalf("decode root manifest: %v", err)
		}
		if len(manifest.Targets) != 1 {
			t.Fatalf("root %s targets = %d, want 1", root.Kind, len(manifest.Targets))
		}
		if !pathIsWithin(root.RootPath, manifest.Targets[0].StagePath) || !pathIsWithin(root.RootPath, manifest.Targets[0].PreimagePath) {
			t.Fatalf("root %s staging escaped root: %#v", root.Kind, manifest.Targets[0])
		}
	}
}

func pathIsWithin(root, candidate string) bool {
	relative, err := filepath.Rel(root, candidate)
	return err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) && !filepath.IsAbs(relative)
}
