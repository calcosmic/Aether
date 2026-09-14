package cmd

// LEARN-05 (204-07-PLAN.md): tests for the seven named gates and the
// run-integrity layer that proves a gate ran everything it discovered.

import (
	"reflect"
	"strings"
	"testing"
)

// withStubEvalGateSelectionResolver overrides evalGateSelectionResolver for
// the duration of a test, restoring it on cleanup. Tests that only exercise
// vocabulary/budget validation use this so they never pay for a real `go
// test -list` subprocess call -- TestEvalGateSelectionMatchesRealTests is
// the one test that deliberately uses the real resolver.
func withStubEvalGateSelectionResolver(t *testing.T, stub func(evalGate) ([]string, error)) {
	t.Helper()
	orig := evalGateSelectionResolver
	evalGateSelectionResolver = stub
	t.Cleanup(func() { evalGateSelectionResolver = orig })
}

// alwaysNonEmptyEvalGateSelection is the stub every vocabulary/budget test
// uses: every gate resolves to a single synthetic test name, so selection
// resolution never itself produces a violation in tests that are not
// exercising the selection-matching rule.
func alwaysNonEmptyEvalGateSelection(g evalGate) ([]string, error) {
	return []string{"TestSynthetic"}, nil
}

func realEvalGateManifestForTest(t *testing.T) evalGateManifest {
	t.Helper()
	manifest, err := loadEvalGateManifest()
	if err != nil {
		t.Fatalf("load real eval gate manifest: %v", err)
	}
	return manifest
}

// TestEvalGateVocabularyMatchesTheManifest covers both directions of the
// mismatch: a gate declared in the constants vocabulary but missing from
// the manifest, and a gate present in the manifest but absent from the
// constants vocabulary.
func TestEvalGateVocabularyMatchesTheManifest(t *testing.T) {
	withStubEvalGateSelectionResolver(t, alwaysNonEmptyEvalGateSelection)

	t.Run("the real manifest matches the real vocabulary", func(t *testing.T) {
		manifest := realEvalGateManifestForTest(t)
		if err := validateEvalGateManifest(manifest); err != nil {
			t.Fatalf("real gate manifest failed validation: %v", err)
		}
		if len(manifest.Gates) != len(evalGateNameVocabulary) {
			t.Fatalf("manifest declares %d gates, want %d (%v)", len(manifest.Gates), len(evalGateNameVocabulary), evalGateNames())
		}
	})

	t.Run("a vocabulary member missing from the manifest is named", func(t *testing.T) {
		manifest := realEvalGateManifestForTest(t)
		var trimmed []evalGate
		for _, g := range manifest.Gates {
			if g.Name == evalGateOvernight {
				continue
			}
			trimmed = append(trimmed, g)
		}
		manifest.Gates = trimmed

		err := validateEvalGateManifest(manifest)
		if err == nil {
			t.Fatal("expected validation to fail when a vocabulary member is missing from the manifest")
		}
		if !strings.Contains(err.Error(), "overnight") || !strings.Contains(err.Error(), "absent from the manifest") {
			t.Fatalf("error does not name the missing gate and which list it is missing from: %v", err)
		}
	})

	t.Run("a manifest gate absent from the vocabulary is named", func(t *testing.T) {
		manifest := realEvalGateManifestForTest(t)
		manifest.Gates = append(manifest.Gates, evalGate{
			Name:          evalGateName("nightly-extra"),
			BudgetSeconds: 60,
			Requires:      evalGateRequiresNone,
		})

		err := validateEvalGateManifest(manifest)
		if err == nil {
			t.Fatal("expected validation to fail when the manifest declares an undeclared gate")
		}
		if !strings.Contains(err.Error(), "nightly-extra") || !strings.Contains(err.Error(), "absent from the constants vocabulary") {
			t.Fatalf("error does not name the offending gate and which list it is missing from: %v", err)
		}
	})
}

// TestEvalGateBudgetsArePositive covers both the real manifest (must pass)
// and a synthetic non-positive budget (must fail, naming the gate).
func TestEvalGateBudgetsArePositive(t *testing.T) {
	withStubEvalGateSelectionResolver(t, alwaysNonEmptyEvalGateSelection)

	t.Run("every real gate has a positive budget", func(t *testing.T) {
		manifest := realEvalGateManifestForTest(t)
		for _, g := range manifest.Gates {
			if g.BudgetSeconds <= 0 {
				t.Errorf("gate %q has non-positive budget_seconds %d", g.Name, g.BudgetSeconds)
			}
		}
	})

	t.Run("a zero budget fails validation by name", func(t *testing.T) {
		manifest := realEvalGateManifestForTest(t)
		for i := range manifest.Gates {
			if manifest.Gates[i].Name == evalGateFast {
				manifest.Gates[i].BudgetSeconds = 0
			}
		}
		err := validateEvalGateManifest(manifest)
		if err == nil {
			t.Fatal("expected validation to fail on a zero budget")
		}
		if !strings.Contains(err.Error(), "fast") || !strings.Contains(err.Error(), "non-positive budget_seconds") {
			t.Fatalf("error does not name the offending gate: %v", err)
		}
	})

	t.Run("a negative budget fails validation by name", func(t *testing.T) {
		manifest := realEvalGateManifestForTest(t)
		for i := range manifest.Gates {
			if manifest.Gates[i].Name == evalGateRace {
				manifest.Gates[i].BudgetSeconds = -5
			}
		}
		err := validateEvalGateManifest(manifest)
		if err == nil {
			t.Fatal("expected validation to fail on a negative budget")
		}
		if !strings.Contains(err.Error(), "race") {
			t.Fatalf("error does not name the offending gate: %v", err)
		}
	})
}

// TestEvalGateSelectionMatchesRealTests is the one test in this file that
// deliberately uses the REAL evalGateSelectionResolver (resolveEvalGateSelection)
// against the real repository -- it is the test that would have caught the
// original truncation-trap defect on day one, applied here to every gate's
// own declared selection.
func TestEvalGateSelectionMatchesRealTests(t *testing.T) {
	manifest := realEvalGateManifestForTest(t)
	for _, g := range manifest.Gates {
		g := g
		t.Run(string(g.Name), func(t *testing.T) {
			names, err := resolveEvalGateSelection(g)
			if err != nil {
				t.Fatalf("gate %q: resolve selection (packages=%v build_tags=%v run_pattern=%q): %v", g.Name, g.Packages, g.BuildTags, g.RunPattern, err)
			}
			if len(names) == 0 {
				t.Fatalf("gate %q selection (packages=%v build_tags=%v run_pattern=%q) matched no test in the repository", g.Name, g.Packages, g.BuildTags, g.RunPattern)
			}
		})
	}
}

// TestEvalGateManifestLoadIsStable proves loading the manifest twice
// returns identical gates in identical declared order.
func TestEvalGateManifestLoadIsStable(t *testing.T) {
	first, err := loadEvalGateManifest()
	if err != nil {
		t.Fatalf("first load: %v", err)
	}
	second, err := loadEvalGateManifest()
	if err != nil {
		t.Fatalf("second load: %v", err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("loadEvalGateManifest is not stable across repeated loads:\nfirst:  %+v\nsecond: %+v", first, second)
	}
	var names []string
	for _, g := range first.Gates {
		names = append(names, string(g.Name))
	}
	var namesAgain []string
	for _, g := range second.Gates {
		namesAgain = append(namesAgain, string(g.Name))
	}
	if !reflect.DeepEqual(names, namesAgain) {
		t.Fatalf("gate declared order changed across loads: %v vs %v", names, namesAgain)
	}
}
