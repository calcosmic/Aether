package cmd

// LEARN-05 (204-07-PLAN.md): tests for the seven named gates and the
// run-integrity layer that proves a gate ran everything it discovered.

import (
	"path/filepath"
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

// TestTruncatedGateRunFails drives assertEvalGateCoverage with a report
// whose executed count is lower than discovered, asserting both figures and
// the gate name appear in the resulting error. A clean report (discovered
// == executed) must not fail.
func TestTruncatedGateRunFails(t *testing.T) {
	truncated := evalGateRunReport{GateName: "fast", Discovered: 120, Executed: 100}
	err := assertEvalGateCoverage(truncated)
	if err == nil {
		t.Fatal("expected a truncated run (discovered != executed) to fail coverage assertion")
	}
	if !strings.Contains(err.Error(), "fast") || !strings.Contains(err.Error(), "120") || !strings.Contains(err.Error(), "100") {
		t.Fatalf("coverage error does not name the gate and both figures: %v", err)
	}

	clean := evalGateRunReport{GateName: "fast", Discovered: 100, Executed: 100}
	if err := assertEvalGateCoverage(clean); err != nil {
		t.Fatalf("a clean report (discovered == executed) unexpectedly failed coverage: %v", err)
	}
}

// TestGateBudgetOverrunIsReported drives assertEvalGateBudget with a report
// whose elapsed time exceeds its declared budget, asserting both figures
// appear in the resulting error. A within-budget report must not fail.
func TestGateBudgetOverrunIsReported(t *testing.T) {
	overrun := evalGateRunReport{GateName: "fast", ElapsedSeconds: 90, DeclaredBudgetSeconds: 60}
	err := assertEvalGateBudget(overrun)
	if err == nil {
		t.Fatal("expected a budget overrun to be reported")
	}
	if !strings.Contains(err.Error(), "fast") || !strings.Contains(err.Error(), "90") || !strings.Contains(err.Error(), "60") {
		t.Fatalf("budget overrun error does not name the gate and both figures: %v", err)
	}

	within := evalGateRunReport{GateName: "fast", ElapsedSeconds: 30, DeclaredBudgetSeconds: 60}
	if err := assertEvalGateBudget(within); err != nil {
		t.Fatalf("a within-budget report unexpectedly failed: %v", err)
	}
}

// TestEverySentinelStillExists resolves each declared sentinel against the
// real repository via the shared AST test-function index.
func TestEverySentinelStillExists(t *testing.T) {
	repoRoot := findTestModuleRoot(t)
	sentinels, err := loadEvalGateSentinels()
	if err != nil {
		t.Fatalf("load eval gate sentinels: %v", err)
	}
	if len(sentinels.Sentinels) == 0 {
		t.Fatal("no sentinels declared")
	}
	locations, err := evalGateTestFunctionLocations(repoRoot, []string{"cmd", "pkg"})
	if err != nil {
		t.Fatalf("index real test function locations: %v", err)
	}
	for _, s := range sentinels.Sentinels {
		key := s.Package + "::" + s.Test
		if !locations[key] {
			t.Errorf("sentinel %s (package %s) no longer exists in the repository", s.Test, s.Package)
		}
	}
}

// TestSentinelsAreSkipProof asserts each sentinel's own function body
// contains no call to t.Skip/t.Skipf/t.SkipNow.
func TestSentinelsAreSkipProof(t *testing.T) {
	repoRoot := findTestModuleRoot(t)
	sentinels, err := loadEvalGateSentinels()
	if err != nil {
		t.Fatalf("load eval gate sentinels: %v", err)
	}
	for _, s := range sentinels.Sentinels {
		dir := filepath.Join(repoRoot, s.Package)
		fn, err := evalGateFuncDeclByName(dir, s.Test)
		if err != nil {
			t.Fatalf("parse sentinel package %s: %v", dir, err)
		}
		if fn == nil {
			t.Errorf("sentinel %s not found as a top-level function in package %s", s.Test, s.Package)
			continue
		}
		if evalGateFuncBodyHasSkipCall(fn) {
			t.Errorf("sentinel %s (package %s) contains a Skip call -- a hard-gate sentinel may never be skippable", s.Test, s.Package)
		}
	}
}

// TestSentinelListOnlyGrows is the ratchet against the recorded floor: every
// name in evalGateSentinelFloor must still be present in the committed
// sentinel list, naming any that has been removed.
func TestSentinelListOnlyGrows(t *testing.T) {
	sentinels, err := loadEvalGateSentinels()
	if err != nil {
		t.Fatalf("load eval gate sentinels: %v", err)
	}
	present := make(map[string]bool, len(sentinels.Sentinels))
	for _, s := range sentinels.Sentinels {
		present[s.Test] = true
	}
	for _, floor := range evalGateSentinelFloor {
		if !present[floor] {
			t.Errorf("sentinel %q from the recorded floor is missing from the current sentinel list -- the sentinel list may only grow; give it a fixture/guard or restore it, never silently drop it", floor)
		}
	}
}

// TestHoldoutsCarryNoReadableName asserts no digest in the holdout file
// appears as a substring of any fixture title or any real test function
// name in the repository -- the proof that the holdout mechanism is real.
func TestHoldoutsCarryNoReadableName(t *testing.T) {
	repoRoot := findTestModuleRoot(t)
	holdouts, err := loadEvalGateHoldouts()
	if err != nil {
		t.Fatalf("load eval gate holdouts: %v", err)
	}
	if len(holdouts.Holdouts) == 0 {
		t.Fatal("no holdouts declared")
	}

	bank, err := loadFixtureBank()
	if err != nil {
		t.Fatalf("load fixture bank: %v", err)
	}
	var titles []string
	for _, f := range bank.Fixtures {
		titles = append(titles, strings.ToLower(f.Title))
	}

	locations, err := evalGateTestFunctionLocations(repoRoot, []string{"cmd", "pkg"})
	if err != nil {
		t.Fatalf("index real test function locations: %v", err)
	}
	var testNames []string
	for key := range locations {
		parts := strings.SplitN(key, "::", 2)
		if len(parts) == 2 {
			testNames = append(testNames, strings.ToLower(parts[1]))
		}
	}

	for _, h := range holdouts.Holdouts {
		digest := strings.ToLower(strings.TrimSpace(h.Digest))
		if digest == "" {
			t.Fatal("holdout entry carries an empty digest")
		}
		for _, title := range titles {
			if strings.Contains(title, digest) {
				t.Fatalf("holdout digest %q appears as a substring of fixture title %q -- the holdout set must never be readable by name", digest, title)
			}
		}
		for _, name := range testNames {
			if strings.Contains(name, digest) {
				t.Fatalf("holdout digest %q appears as a substring of test name %q -- the holdout set must never be readable by name", digest, name)
			}
		}
	}
}

// TestHoldoutResolutionIsNonEmpty proves the holdout mechanism is real: the
// digests in the committed holdout file must resolve to real fixtures in
// the committed bank, not merely declare a mechanism that never matches.
func TestHoldoutResolutionIsNonEmpty(t *testing.T) {
	bank, err := loadFixtureBank()
	if err != nil {
		t.Fatalf("load fixture bank: %v", err)
	}
	matched := resolveEvalGateHoldouts(bank)
	if len(matched) == 0 {
		t.Fatal("resolveEvalGateHoldouts returned no fixtures against the real bank and holdout file -- the holdout mechanism is not proven live")
	}
}

// TestDefaultSuiteDiscoveredCountIsUnchangedByTags lists the default-build
// tests before and after the integration/provider/overnight tags exist and
// asserts the count is identical -- these tags are purely additive and must
// never shrink or grow the untagged (fast/focused) selection.
func TestDefaultSuiteDiscoveredCountIsUnchangedByTags(t *testing.T) {
	before, err := resolveEvalGateSelection(evalGate{Packages: []string{"./..."}})
	if err != nil {
		t.Fatalf("resolve default (no-tag) selection: %v", err)
	}
	after, err := resolveEvalGateSelection(evalGate{
		Packages:  []string{"./..."},
		BuildTags: []string{"integration", "provider", "overnight"},
	})
	if err != nil {
		t.Fatalf("resolve tagged selection: %v", err)
	}
	if len(before) != len(after) {
		t.Fatalf("default suite discovered count changed when the integration/provider/overnight tags were added: before=%d after=%d", len(before), len(after))
	}
}
