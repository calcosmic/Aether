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

	// Asserting only that checks NAMED "anti_pattern" appear is not enough: two
	// literal gateCheck{Name: "anti_pattern", Passed: true} structs substituted
	// for the producer call satisfy that and the gate never runs. CLAUDE.md's
	// Definition of Done says it directly — "a test that only checks for a named
	// section cannot catch its replacement". So compare the pipeline's checks
	// against what checkAntiPatternGate itself produces for the same input; a
	// stub diverges on Detail even when it matches on Name and Passed.
	wantFindings, wantExecuted := checkAntiPatternGate(verification.Claims.ScannedFiles)

	got := map[string]gateCheck{}
	for _, c := range report.Checks {
		got[c.Name] = c
	}

	for _, want := range []gateCheck{wantFindings, wantExecuted} {
		have, ok := got[want.Name]
		if !ok {
			t.Fatalf("runCodexContinueGates report is missing the %q check — the security gate call site has been removed from the continue pipeline", want.Name)
		}
		if have.Passed != want.Passed {
			t.Errorf("%s: pipeline reported Passed=%v but checkAntiPatternGate produced Passed=%v — the pipeline is not using the real producer", want.Name, have.Passed, want.Passed)
		}
		if have.Detail != want.Detail {
			t.Errorf("%s: pipeline Detail %q does not match checkAntiPatternGate's %q — the call site appears to be stubbed rather than calling the producer", want.Name, have.Detail, want.Detail)
		}
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

// Found during UAT of Phase 160, not by the plan's own tests.
//
// `scanFileForAntipatterns` returns clean (nil,nil,nil) for a file that does
// not exist — correct for the CLI, where a phase may legitimately have deleted
// the file. The gate originally inherited that and counted absent files as
// scanned, reporting "antipattern scan executed successfully across 1 file(s)"
// having read nothing. A security gate that reports success without running is
// the exact failure this phase exists to eliminate, so it is pinned here.
func TestAntiPatternGateDoesNotCountAbsentFilesAsScanned(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	t.Run("AllFilesAbsentHardBlocks", func(t *testing.T) {
		findings, executed := checkAntiPatternGate([]string{"no/such/file.go", "also/missing.go"})
		if executed.Passed {
			t.Errorf("anti_pattern_executed passed when 0 of 2 claimed files could be read — the scan did not happen, so the gate must not report success. Detail: %q", executed.Detail)
		}
		if !strings.Contains(executed.Detail, "0 of 2") {
			t.Errorf("detail should state how many of the claimed files were actually scanned, got: %q", executed.Detail)
		}
		if strings.Contains(findings.Detail, "scanned 2") {
			t.Errorf("findings detail claims 2 files were scanned when none existed: %q", findings.Detail)
		}
	})

	t.Run("PartialScanReportsBothCounts", func(t *testing.T) {
		real := filepath.Join(tmpDir, "real.go")
		if err := os.WriteFile(real, []byte("package main\n"), 0644); err != nil {
			t.Fatal(err)
		}
		findings, executed := checkAntiPatternGate([]string{real, "gone/missing.go"})
		if !executed.Passed {
			t.Errorf("one readable file is enough for the scan to have executed; got failure: %q", executed.Detail)
		}
		if !strings.Contains(executed.Detail, "absent") {
			t.Errorf("an absent claimed file must be named in the detail so the operator can see it, got: %q", executed.Detail)
		}
		if !strings.Contains(findings.Detail, "1 of 2") {
			t.Errorf("findings detail must distinguish scanned from claimed, got: %q", findings.Detail)
		}
	})
}

// Phase 210 blocker 16 (2026-10-02, Finish the Track deck): a helper listed
// its whole output folder ("finish-the-track/dist") instead of naming each
// file. The scan tried to read the folder as a file, could not, and failed
// the phase under the heading "Critical patterns detected" although nothing
// had been found. A claimed folder is now scanned file by file -- more is
// checked than before, not less -- and a scan that genuinely cannot run says
// so instead of claiming it found something.
func TestAntiPatternGateScansTheFilesInsideAClaimedFolder(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	dist := filepath.Join(tmpDir, "finish-the-track", "dist")
	if err := os.MkdirAll(filepath.Join(dist, "media"), 0755); err != nil {
		t.Fatal(err)
	}
	for name, body := range map[string]string{
		"PREVIEW.html":    "<!doctype html><title>Preview</title>\n",
		"media/style.css": "body { margin: 0 }\n",
	} {
		if err := os.WriteFile(filepath.Join(dist, filepath.FromSlash(name)), []byte(body), 0644); err != nil {
			t.Fatal(err)
		}
	}

	t.Run("CleanFolderIsScannedFileByFile", func(t *testing.T) {
		findings, executed := checkAntiPatternGate([]string{"finish-the-track/dist"})
		if !executed.Passed || strings.Contains(executed.Detail, "directory") {
			t.Fatalf("a claimed folder must be scanned, not read as a file: %q", executed.Detail)
		}
		if !findings.Passed || !strings.Contains(findings.Detail, "scanned 2 of 2") {
			t.Fatalf("both files inside the folder must be scanned and counted, got: %+v", findings)
		}
	})

	t.Run("AFindingInsideTheFolderIsStillCaught", func(t *testing.T) {
		leak := filepath.Join(dist, "media", "config.js")
		if err := os.WriteFile(leak, []byte("var apiKey = \"sk-live-4f9a8b7c6d\"\n"), 0644); err != nil {
			t.Fatal(err)
		}
		defer os.Remove(leak)
		findings, _ := checkAntiPatternGate([]string{"finish-the-track/dist"})
		if findings.Passed || !strings.Contains(findings.Detail, "config.js") {
			t.Fatalf("a credential inside a claimed folder must fail the scan and name the file, got: %+v", findings)
		}
	})

	t.Run("AScanThatCannotRunDoesNotClaimAFinding", func(t *testing.T) {
		if os.Geteuid() == 0 {
			t.Skip("root can read an unreadable file")
		}
		locked := filepath.Join(tmpDir, "locked.txt")
		if err := os.WriteFile(locked, []byte("x\n"), 0o000); err != nil {
			t.Fatal(err)
		}
		defer os.Remove(locked)
		findings, executed := checkAntiPatternGate([]string{"locked.txt"})
		if executed.Passed {
			t.Fatalf("an unreadable claimed file must fail the executed check, got: %+v", executed)
		}
		if strings.Contains(executed.FixHint, "Critical patterns detected") {
			t.Fatalf("a scan that could not run must not say critical patterns were detected: %q", executed.FixHint)
		}
		if !findings.Passed {
			t.Fatalf("nothing was found, so the findings check itself must not fail: %+v", findings)
		}
	})
}
