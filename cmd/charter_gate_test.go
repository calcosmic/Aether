package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

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
