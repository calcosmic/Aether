package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

// These tests prove CONTEXT-06/D-09: a charter compliance gate exists that
// fails when a declared, still-installed governance tool went unexercised by
// any verification step, and separately reports when it could not run at
// all. Modeled directly on checkAntiPatternGate's two-check producer shape
// (cmd/gate.go:432).

// TestContinueCharterComplianceGate exercises checkCharterComplianceGate in
// isolation across the five behaviors the plan specifies.
func TestContinueCharterComplianceGate(t *testing.T) {
	t.Run("ViolationWhenToolNotExercised", func(t *testing.T) {
		saveGlobals(t)
		resetRootCmd(t)
		s, tmpDir := newTestStore(t)
		defer os.RemoveAll(tmpDir)
		store = s

		if err := os.WriteFile(filepath.Join(tmpDir, ".golangci.yml"), []byte("run:\n"), 0644); err != nil {
			t.Fatalf("write golangci config: %v", err)
		}

		state := colony.ColonyState{
			Charter: &colony.Charter{Governance: "Linting: golangci-lint"},
		}
		if err := s.SaveJSON("COLONY_STATE.json", state); err != nil {
			t.Fatal(err)
		}

		steps := []codexVerificationStep{
			{Name: "tests", Command: "go test ./..."},
		}

		findings, executed := checkCharterComplianceGate(steps)

		if !executed.Passed {
			t.Fatalf("executed check should pass when the scan ran, got: %+v", executed)
		}
		if findings.Passed {
			t.Fatalf("findings check should fail when a declared, still-installed tool was never exercised, got: %+v", findings)
		}
		if !strings.Contains(findings.Detail, "golangci-lint") {
			t.Errorf("findings Detail should name the unexercised tool, got: %q", findings.Detail)
		}
	})

	t.Run("PassesWhenToolIsExercised", func(t *testing.T) {
		saveGlobals(t)
		resetRootCmd(t)
		s, tmpDir := newTestStore(t)
		defer os.RemoveAll(tmpDir)
		store = s

		if err := os.WriteFile(filepath.Join(tmpDir, ".golangci.yml"), []byte("run:\n"), 0644); err != nil {
			t.Fatalf("write golangci config: %v", err)
		}

		state := colony.ColonyState{
			Charter: &colony.Charter{Governance: "Linting: golangci-lint"},
		}
		if err := s.SaveJSON("COLONY_STATE.json", state); err != nil {
			t.Fatal(err)
		}

		steps := []codexVerificationStep{
			{Name: "lint", Command: "golangci-lint run ./..."},
		}

		findings, executed := checkCharterComplianceGate(steps)

		if !executed.Passed {
			t.Fatalf("executed check should pass, got: %+v", executed)
		}
		if !findings.Passed {
			t.Fatalf("findings check should pass when the declared tool was exercised by a verification step, got: %+v", findings)
		}
	})

	t.Run("NoViolationWhenConfigFileGone", func(t *testing.T) {
		saveGlobals(t)
		resetRootCmd(t)
		s, tmpDir := newTestStore(t)
		defer os.RemoveAll(tmpDir)
		store = s

		// Deliberately do NOT create .golangci.yml -- the charter has drifted
		// from the repo, which is not a worker's fault to be blocked for.
		state := colony.ColonyState{
			Charter: &colony.Charter{Governance: "Linting: golangci-lint"},
		}
		if err := s.SaveJSON("COLONY_STATE.json", state); err != nil {
			t.Fatal(err)
		}

		findings, executed := checkCharterComplianceGate(nil)

		if !executed.Passed {
			t.Fatalf("executed check should pass, got: %+v", executed)
		}
		if !findings.Passed {
			t.Fatalf("findings check must not raise a violation when the declared tool's config file no longer exists, got: %+v", findings)
		}
	})

	t.Run("StoreUnavailableFailsExecutedNotFindings", func(t *testing.T) {
		saveGlobals(t)
		resetRootCmd(t)
		store = nil

		findings, executed := checkCharterComplianceGate(nil)

		if executed.Passed {
			t.Fatalf("executed check should fail when store is unavailable, got: %+v", executed)
		}
		if !strings.Contains(executed.Detail, "could not execute") {
			t.Errorf("executed Detail should say the scan could not execute, got: %q", executed.Detail)
		}
		// The findings check must not silently read as "success" -- its
		// Passed=true (matching the anti-pattern gate's honesty pattern) is
		// only acceptable because its Detail explicitly says it did not run.
		if !findings.Passed {
			t.Fatalf("findings check should not itself hard-fail when unable to run, got: %+v", findings)
		}
		if strings.Contains(strings.ToLower(findings.Detail), "no violation") || strings.Contains(strings.ToLower(findings.Detail), "no compliance violation") {
			t.Errorf("findings Detail must not imply a completed, clean scan when nothing ran, got: %q", findings.Detail)
		}
		if !strings.Contains(strings.ToLower(findings.Detail), "did not run") && !strings.Contains(strings.ToLower(findings.Detail), "not run") {
			t.Errorf("findings Detail should say the scan did not run, got: %q", findings.Detail)
		}
	})

	t.Run("EmptyOrFallbackCharterHasNothingToEnforce", func(t *testing.T) {
		saveGlobals(t)
		resetRootCmd(t)
		s, tmpDir := newTestStore(t)
		defer os.RemoveAll(tmpDir)
		store = s

		state := colony.ColonyState{
			Charter: &colony.Charter{Governance: "No formal governance detected -- colony should establish conventions"},
		}
		if err := s.SaveJSON("COLONY_STATE.json", state); err != nil {
			t.Fatal(err)
		}

		findings, executed := checkCharterComplianceGate(nil)

		if !executed.Passed {
			t.Fatalf("executed check should pass, got: %+v", executed)
		}
		if !findings.Passed {
			t.Fatalf("findings check should pass when there is nothing mechanical to enforce, got: %+v", findings)
		}
		if !strings.Contains(strings.ToLower(findings.Detail), "nothing") {
			t.Errorf("findings Detail should say there is nothing mechanical to enforce, got: %q", findings.Detail)
		}

		// Also cover a genuinely empty charter (nil Charter pointer).
		s2, tmpDir2 := newTestStore(t)
		defer os.RemoveAll(tmpDir2)
		store = s2
		if err := s2.SaveJSON("COLONY_STATE.json", colony.ColonyState{}); err != nil {
			t.Fatal(err)
		}
		findings2, executed2 := checkCharterComplianceGate(nil)
		if !executed2.Passed || !findings2.Passed {
			t.Fatalf("nil charter should produce two passing checks, got findings=%+v executed=%+v", findings2, executed2)
		}
	})

	// Structural assertion demanded by the plan's acceptance criteria: the
	// findings and executed checks must be distinct gateCheck values with
	// distinct names, never a single collapsed boolean.
	t.Run("FindingsAndExecutedAreDistinctChecks", func(t *testing.T) {
		saveGlobals(t)
		resetRootCmd(t)
		s, tmpDir := newTestStore(t)
		defer os.RemoveAll(tmpDir)
		store = s

		if err := s.SaveJSON("COLONY_STATE.json", colony.ColonyState{}); err != nil {
			t.Fatal(err)
		}

		findings, executed := checkCharterComplianceGate(nil)
		if findings.Name != "charter_compliance" {
			t.Errorf("findings check Name = %q, want charter_compliance", findings.Name)
		}
		if executed.Name != "charter_compliance_executed" {
			t.Errorf("executed check Name = %q, want charter_compliance_executed", executed.Name)
		}
	})
}

// TestCharterComplianceGateAlwaysRuns pins that charter_compliance_executed
// is registered in alwaysRunGates, matching anti_pattern_executed -- a
// skipped prior result must never suppress the "did it run" signal.
func TestCharterComplianceGateAlwaysRuns(t *testing.T) {
	if !alwaysRunGates["charter_compliance_executed"] {
		t.Fatal("charter_compliance_executed must be registered in alwaysRunGates")
	}
}

// These tests prove the charter compliance gate has a live caller inside the
// continue gate pipeline (task 3), not just a working
// checkCharterComplianceGate producer in isolation -- mirroring
// continue_antipattern_gate_test.go's wiring proof for the anti_pattern gate.

func TestContinueCharterComplianceGateWiredIntoPipeline(t *testing.T) {
	t.Run("BlocksOnUnexercisedTool", func(t *testing.T) {
		saveGlobals(t)
		resetRootCmd(t)
		s, tmpDir := newTestStore(t)
		defer os.RemoveAll(tmpDir)
		store = s

		if err := os.WriteFile(filepath.Join(tmpDir, ".golangci.yml"), []byte("run:\n"), 0644); err != nil {
			t.Fatalf("write golangci config: %v", err)
		}

		state := colony.ColonyState{Charter: &colony.Charter{Governance: "Linting: golangci-lint"}}
		if err := s.SaveJSON("COLONY_STATE.json", state); err != nil {
			t.Fatal(err)
		}

		phase, manifest, assessment := minimalContinuePhaseAndAssessment()
		verification := codexContinueVerificationReport{
			ChecksPassed: true,
			Passed:       true,
			Claims:       codexClaimVerification{Present: true, Passed: true},
			Steps: []codexVerificationStep{
				{Name: "tests", Command: "go test ./..."},
			},
		}

		report := runCodexContinueGates(phase, manifest, verification, assessment, time.Now(), nil)

		var found *gateCheck
		for i := range report.Checks {
			if report.Checks[i].Name == "charter_compliance" {
				found = &report.Checks[i]
				break
			}
		}
		if found == nil {
			t.Fatalf("charter_compliance check not present in gate report: %+v", report.Checks)
		}
		if found.Passed {
			t.Fatalf("charter_compliance should have failed on an unexercised declared tool, got: %+v", found)
		}
		if !strings.Contains(found.Detail, "golangci-lint") {
			t.Errorf("charter_compliance Detail should name the unexercised tool, got: %s", found.Detail)
		}

		blockingFound := false
		for _, b := range report.BlockingIssues {
			if strings.Contains(b, "golangci-lint") {
				blockingFound = true
				break
			}
		}
		if !blockingFound {
			t.Errorf("report.BlockingIssues should contain the charter_compliance detail naming golangci-lint, got: %v", report.BlockingIssues)
		}
	})

	t.Run("ExecutedCheckAlwaysPresentEvenWhenFindingsSkipped", func(t *testing.T) {
		saveGlobals(t)
		resetRootCmd(t)
		s, tmpDir := newTestStore(t)
		defer os.RemoveAll(tmpDir)
		store = s

		if err := s.SaveJSON("COLONY_STATE.json", colony.ColonyState{}); err != nil {
			t.Fatal(err)
		}

		phase, manifest, assessment := minimalContinuePhaseAndAssessment()
		verification := codexContinueVerificationReport{
			ChecksPassed: true,
			Passed:       true,
			Claims:       codexClaimVerification{Present: true, Passed: true},
		}

		priorResults := []GateCheckResult{
			{Name: "charter_compliance", Status: "passed"},
		}

		report := runCodexContinueGates(phase, manifest, verification, assessment, time.Now(), priorResults)

		var findingsCheck, executedCheck *gateCheck
		for i := range report.Checks {
			switch report.Checks[i].Name {
			case "charter_compliance":
				findingsCheck = &report.Checks[i]
			case "charter_compliance_executed":
				executedCheck = &report.Checks[i]
			}
		}
		if findingsCheck == nil {
			t.Fatalf("charter_compliance check missing: %+v", report.Checks)
		}
		if !strings.Contains(findingsCheck.Detail, "skipped") {
			t.Errorf("charter_compliance should report as skipped when previously passed, got: %s", findingsCheck.Detail)
		}
		if executedCheck == nil {
			t.Fatalf("charter_compliance_executed check missing even though findings was skipped as previously passed: %+v", report.Checks)
		}
		if !executedCheck.Passed {
			t.Errorf("charter_compliance_executed should pass when the scan ran, got: %+v", executedCheck)
		}
	})

	// Anti-regression test mirroring
	// TestContinueAntiPatternGateIsWiredIntoThePipeline: compares the
	// pipeline's checks against what checkCharterComplianceGate itself
	// produces for the same input, so a stubbed call site (a literal
	// gateCheck{Name: "charter_compliance", Passed: true} substituted for
	// the producer call) would be caught even though it matches on Name and
	// Passed.
	t.Run("PipelineUsesRealProducer", func(t *testing.T) {
		saveGlobals(t)
		resetRootCmd(t)
		s, tmpDir := newTestStore(t)
		defer os.RemoveAll(tmpDir)
		store = s

		state := colony.ColonyState{Charter: &colony.Charter{Governance: "Testing: pytest"}}
		if err := s.SaveJSON("COLONY_STATE.json", state); err != nil {
			t.Fatal(err)
		}

		phase, manifest, assessment := minimalContinuePhaseAndAssessment()
		verification := codexContinueVerificationReport{
			ChecksPassed: true,
			Passed:       true,
			Claims:       codexClaimVerification{Present: true, Passed: true},
			Steps:        []codexVerificationStep{{Name: "tests", Command: "pytest -q"}},
		}

		report := runCodexContinueGates(phase, manifest, verification, assessment, time.Now(), nil)

		wantFindings, wantExecuted := checkCharterComplianceGate(verification.Steps)

		got := map[string]gateCheck{}
		for _, c := range report.Checks {
			got[c.Name] = c
		}
		for _, want := range []gateCheck{wantFindings, wantExecuted} {
			have, ok := got[want.Name]
			if !ok {
				t.Fatalf("runCodexContinueGates report is missing the %q check — the charter gate call site has been removed from the continue pipeline", want.Name)
			}
			if have.Passed != want.Passed {
				t.Errorf("%s: pipeline reported Passed=%v but checkCharterComplianceGate produced Passed=%v — the pipeline is not using the real producer", want.Name, have.Passed, want.Passed)
			}
			if have.Detail != want.Detail {
				t.Errorf("%s: pipeline Detail %q does not match checkCharterComplianceGate's %q — the call site appears to be stubbed rather than calling the producer", want.Name, have.Detail, want.Detail)
			}
		}
	})
}
