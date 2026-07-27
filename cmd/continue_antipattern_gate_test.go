package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

// These tests prove that the anti_pattern security gate has a live caller
// inside the continue gate pipeline (RESEARCH.md Pitfall 1 / T-160-01), not
// just a working checkAntiPatternGate producer in isolation. At least the
// pipeline-wiring and blocking cases go through runCodexContinueGates rather
// than calling checkAntiPatternGate directly.

func minimalContinuePhaseAndAssessment() (colony.Phase, codexContinueManifest, codexContinueAssessment) {
	phase := colony.Phase{ID: 1, Name: "Test", Status: colony.PhaseInProgress}
	manifest := codexContinueManifest{Present: true}
	assessment := codexContinueAssessment{PositiveEvidence: true, Passed: true}
	return phase, manifest, assessment
}

func TestContinueAntiPatternGateBlocksOnCriticalFinding(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	// A fixture under the resolved colony root with a hardcoded credential —
	// the universal critical pattern scanFileForAntipatterns detects.
	fixtureRel := "leaky_config.go"
	fixturePath := filepath.Join(tmpDir, fixtureRel)
	if err := os.WriteFile(fixturePath, []byte("package config\n\nvar apiKey = \"sk-live-4f9a8b7c6d\"\n"), 0644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	phase, manifest, assessment := minimalContinuePhaseAndAssessment()
	verification := codexContinueVerificationReport{
		ChecksPassed: true,
		Passed:       true,
		Claims: codexClaimVerification{
			Present:      true,
			Passed:       true,
			ScannedFiles: []string{fixtureRel},
		},
	}

	report := runCodexContinueGates(phase, manifest, verification, assessment, time.Now(), nil)

	var found *gateCheck
	for i := range report.Checks {
		if report.Checks[i].Name == "anti_pattern" {
			found = &report.Checks[i]
			break
		}
	}
	if found == nil {
		t.Fatalf("anti_pattern check not present in gate report: %+v", report.Checks)
	}
	if found.Passed {
		t.Fatalf("anti_pattern check should have failed on a hardcoded credential, got: %+v", found)
	}
	if !strings.Contains(found.Detail, fixtureRel) {
		t.Errorf("anti_pattern Detail should name the fixture file %q, got: %s", fixtureRel, found.Detail)
	}

	blockingFound := false
	for _, b := range report.BlockingIssues {
		if strings.Contains(b, fixtureRel) {
			blockingFound = true
			break
		}
	}
	if !blockingFound {
		t.Errorf("report.BlockingIssues should contain the anti_pattern detail naming %s, got: %v", fixtureRel, report.BlockingIssues)
	}
}

func TestContinueAntiPatternGatePassesOnCleanFiles(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	fixtureRel := "clean.go"
	fixturePath := filepath.Join(tmpDir, fixtureRel)
	if err := os.WriteFile(fixturePath, []byte("package clean\n\nfunc Add(a, b int) int { return a + b }\n"), 0644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	phase, manifest, assessment := minimalContinuePhaseAndAssessment()
	verification := codexContinueVerificationReport{
		ChecksPassed: true,
		Passed:       true,
		Claims: codexClaimVerification{
			Present:      true,
			Passed:       true,
			ScannedFiles: []string{fixtureRel},
		},
	}

	report := runCodexContinueGates(phase, manifest, verification, assessment, time.Now(), nil)

	var found *gateCheck
	for i := range report.Checks {
		if report.Checks[i].Name == "anti_pattern" {
			found = &report.Checks[i]
			break
		}
	}
	if found == nil {
		t.Fatalf("anti_pattern check not present in gate report: %+v", report.Checks)
	}
	if !found.Passed {
		t.Fatalf("anti_pattern check should pass on a clean file, got: %+v", found)
	}
	if !strings.Contains(found.Detail, "1") {
		t.Errorf("pass Detail should state the number of files scanned (1), got: %s", found.Detail)
	}
}

// TestContinueAntiPatternGateRunsEvenWhenNoFilesChanged asserts the zero-file
// case is stated explicitly (T-160-10): a scan over zero files must not be
// indistinguishable from a scan over everything.
func TestContinueAntiPatternGateRunsEvenWhenNoFilesChanged(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	phase, manifest, assessment := minimalContinuePhaseAndAssessment()
	verification := codexContinueVerificationReport{
		ChecksPassed: true,
		Passed:       true,
		Claims: codexClaimVerification{
			Present:      true,
			Passed:       true,
			ScannedFiles: []string{},
		},
	}

	report := runCodexContinueGates(phase, manifest, verification, assessment, time.Now(), nil)

	var antiPattern, antiPatternExecuted *gateCheck
	for i := range report.Checks {
		switch report.Checks[i].Name {
		case "anti_pattern":
			antiPattern = &report.Checks[i]
		case "anti_pattern_executed":
			antiPatternExecuted = &report.Checks[i]
		}
	}
	if antiPattern == nil {
		t.Fatalf("anti_pattern check missing when no files changed: %+v", report.Checks)
	}
	if antiPatternExecuted == nil {
		t.Fatalf("anti_pattern_executed check missing when no files changed: %+v", report.Checks)
	}
	if !antiPatternExecuted.Passed {
		t.Fatalf("anti_pattern_executed should pass when the claims file list is legitimately empty, got: %+v", antiPatternExecuted)
	}
	if !strings.Contains(strings.ToLower(antiPatternExecuted.Detail), "no changed files") {
		t.Errorf("anti_pattern_executed Detail should explicitly say no changed files were claimed, got: %s", antiPatternExecuted.Detail)
	}
}

// TestContinueAntiPatternGateIsWiredIntoThePipeline is the anti-regression
// test for RESEARCH.md Pitfall 1: the gate must be reachable through
// runCodexContinueGates, not merely exist as a standalone producer. This test
// fails if a future change removes the call site inside runCodexContinueGates
// — a unit test of checkAntiPatternGate alone would not catch that removal.
func TestContinueAntiPatternGateIsWiredIntoThePipeline(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	phase, manifest, assessment := minimalContinuePhaseAndAssessment()
	verification := codexContinueVerificationReport{
		ChecksPassed: true,
		Passed:       true,
		Claims: codexClaimVerification{
			Present:      true,
			Passed:       true,
			ScannedFiles: nil,
		},
	}

	report := runCodexContinueGates(phase, manifest, verification, assessment, time.Now(), nil)

	names := map[string]bool{}
	for _, c := range report.Checks {
		names[c.Name] = true
	}
	if !names["anti_pattern"] {
		t.Error("runCodexContinueGates report is missing the 'anti_pattern' check — the security gate call site may have been removed")
	}
	if !names["anti_pattern_executed"] {
		t.Error("runCodexContinueGates report is missing the 'anti_pattern_executed' check — the security gate call site may have been removed")
	}
}

// TestAntiPatternScanFailureHardBlocks asserts D-01's HALT rule: a scan that
// cannot execute must hard-block and must never auto-resolve, at any depth.
func TestAntiPatternScanFailureHardBlocks(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	// A claimed "file" that resolves to a directory: scanFileForAntipatterns
	// will fail to os.ReadFile it (it exists, so the "missing file = clean"
	// branch does not apply, but reading it as a file errors), driving the
	// cannot-execute path rather than the not-found path.
	unreadableRel := "not_actually_a_file"
	if err := os.MkdirAll(filepath.Join(tmpDir, unreadableRel), 0755); err != nil {
		t.Fatalf("create unreadable target: %v", err)
	}

	phase, manifest, assessment := minimalContinuePhaseAndAssessment()
	verification := codexContinueVerificationReport{
		ChecksPassed: true,
		Passed:       true,
		Claims: codexClaimVerification{
			Present:      true,
			Passed:       true,
			ScannedFiles: []string{unreadableRel},
		},
	}

	report := runCodexContinueGates(phase, manifest, verification, assessment, time.Now(), nil)

	var antiPatternExecuted *gateCheck
	for i := range report.Checks {
		if report.Checks[i].Name == "anti_pattern_executed" {
			antiPatternExecuted = &report.Checks[i]
			break
		}
	}
	if antiPatternExecuted == nil {
		t.Fatalf("anti_pattern_executed check not present: %+v", report.Checks)
	}
	if antiPatternExecuted.Passed {
		t.Fatalf("anti_pattern_executed should fail when the scan cannot execute, got: %+v", antiPatternExecuted)
	}

	tier, _ := gateClassify("anti_pattern_executed")
	if tier != hardBlock {
		t.Fatalf("gateClassify(anti_pattern_executed) = %q, want hardBlock", tier)
	}

	// D-01's HALT rule, asserted: autoResolveSoftBlockGates must not flip a
	// hardBlock gate to passed, even at the most permissive (light) depth.
	resolved, autoResolved := autoResolveSoftBlockGates(phase.ID, report, "light", colony.PhaseModeProduction)
	for _, name := range autoResolved {
		if name == "anti_pattern_executed" {
			t.Fatal("anti_pattern_executed was auto-resolved at light depth; hardBlock gates must never auto-resolve (D-01)")
		}
	}
	for _, c := range resolved.Checks {
		if c.Name == "anti_pattern_executed" && c.Passed {
			t.Fatal("anti_pattern_executed reads as passed after autoResolveSoftBlockGates; a scan that could not run must never be reported as passing")
		}
	}
}
