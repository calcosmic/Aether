package cmd

import (
	"fmt"
	"strings"
	"testing"
)

// classicCoverage201Row is one entry in the Phase 201 coverage ledger: a
// SYN-201 decision or a capability row this phase routes, each naming the
// requirement it serves, its disposition, and its delivered proof -- a
// declared Go test name (resolved from parsed declarations, never text
// search) or a corpus case identifier ("case:<id>"). A row may be marked
// Unresolved with a stated Reason instead of carrying a proof; it may never
// carry both.
type classicCoverage201Row struct {
	Type        string // "decision" or "capability"
	ID          string // SYN-201-XX or CAP-XXX
	Requirement string
	Disposition string
	Proof       string
	Unresolved  bool
	Reason      string
}

// classicCoverage201Rows is the Phase 201 coverage ledger: exactly one row
// per SYN-201 decision (14) and one row per capability row this phase
// routes (8, per classicContractPhase201Capabilities). Every proof below
// names a real, currently-passing top-level Go test in the cmd package --
// the same tests already exercised as go_test_symbol causal proofs by the
// Phase 201 corpus cases (cmd/testdata/classic-contract/v1/cases.json) and
// TestClassicContractPhase201CausalExecution.
var classicCoverage201Rows = []classicCoverage201Row{
	{Type: "decision", ID: "SYN-201-01", Requirement: "WORK-01", Disposition: "keep-current", Proof: "TestOneTaskBugFixIsOneWorkerPlusChecks"},
	{Type: "decision", ID: "SYN-201-02", Requirement: "WORK-03", Disposition: "keep-current", Proof: "TestCoherentJobProposalAccepted"},
	{Type: "decision", ID: "SYN-201-03", Requirement: "WORK-04", Disposition: "replace-better", Proof: "TestPhaseVerifiedOnce"},
	{Type: "decision", ID: "SYN-201-04", Requirement: "WORK-04", Disposition: "replace-better", Proof: "TestAllThreeContinueLanesShareOneAcceptVerifyAdvanceBody"},
	{Type: "decision", ID: "SYN-201-05", Requirement: "WORK-04", Disposition: "restore-modern", Proof: "TestBuildBeforeVerificationSaysItIsNotVerifiedYet"},
	{Type: "decision", ID: "SYN-201-06", Requirement: "WORK-05", Disposition: "replace-better", Proof: "TestEveryWorkVerdictRendersTheSameCeremonySlots"},
	{Type: "decision", ID: "SYN-201-07", Requirement: "WORK-05", Disposition: "replace-better", Proof: "TestElapsedTimeComesFromTheAttemptTimestamps"},
	{Type: "decision", ID: "SYN-201-08", Requirement: "WORK-05", Disposition: "restore-modern", Proof: "TestEveryDeciderAgreesOnTheNextCommand"},
	{Type: "decision", ID: "SYN-201-09", Requirement: "WORK-05", Disposition: "replace-better", Proof: "TestGroupedWorktreePartialReceiptsSyncBeforeCredit"},
	{Type: "decision", ID: "SYN-201-10", Requirement: "WORK-06", Disposition: "replace-better", Proof: "TestFailedRepairRestoresTheCheckpointExactly"},
	{Type: "decision", ID: "SYN-201-11", Requirement: "WORK-06", Disposition: "replace-better", Proof: "TestFailureBornSignalReachesTheRepairBrief"},
	{Type: "decision", ID: "SYN-201-12", Requirement: "WORK-07", Disposition: "replace-better", Proof: "TestAutopilotSelectsTransitionsFromOneAcceptedGoal"},
	{Type: "decision", ID: "SYN-201-13", Requirement: "WORK-08", Disposition: "replace-better", Proof: "TestRealBuildProducesMeasuredSegments"},
	{Type: "decision", ID: "SYN-201-14", Requirement: "WORK-08", Disposition: "replace-better", Proof: "TestScopedTestsRunInTheLoopAndTheFullSuiteAtTheBoundary"},

	{Type: "capability", ID: "CAP-003", Requirement: "WORK-06", Disposition: "restore-modern", Proof: "TestFailedCheckWritesOneFailureRecordOnBothLanes"},
	{Type: "capability", ID: "CAP-004", Requirement: "WORK-05", Disposition: "restore-modern", Proof: "TestFlagCheckBlockersCounts"},
	{Type: "capability", ID: "CAP-022", Requirement: "WORK-05", Disposition: "restore-modern", Proof: "TestPhaseVerifiedOnce"},
	{Type: "capability", ID: "CAP-024", Requirement: "WORK-06", Disposition: "restore-modern", Proof: "TestFailedRepairRestoresTheCheckpointExactly"},
	{Type: "capability", ID: "CAP-029", Requirement: "WORK-07", Disposition: "replace-better", Proof: "TestQuickRunsOnTheSharedAttemptModel"},
	{Type: "capability", ID: "CAP-051", Requirement: "WORK-05", Disposition: "restore-modern", Proof: "TestElapsedTimeComesFromTheAttemptTimestamps"},
	{Type: "capability", ID: "CAP-066", Requirement: "CEC-06", Disposition: "replace-better", Proof: "TestFinalizeLaneAcceptsReconciliationLikeTheDirectLane"},
	{Type: "capability", ID: "CAP-071", Requirement: "WORK-05", Disposition: "replace-better", Proof: "TestGroupedWorktreePartialReceiptsSyncBeforeCredit"},
}

func TestClassicCoverage201ExactSets(t *testing.T) {
	if err := validateClassicCoverage201ExactSets(classicCoverage201Rows); err != nil {
		t.Fatal(err)
	}
}

func TestClassicCoverage201ProofResolution(t *testing.T) {
	if err := validateClassicCoverage201ProofResolution(classicCoverage201Rows); err != nil {
		t.Fatal(err)
	}
	if err := validateClassicCoverage201Fields(classicCoverage201Rows); err != nil {
		t.Fatal(err)
	}
}

func TestClassicCoverage201RejectsDecoys(t *testing.T) {
	root := findTestModuleRootForClassicCoverage199()
	index, err := buildClassicCoverage199ProofIndex(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, decoy := range []string{
		"TestClassicCoverage201NoSuchTestXYZ",
		"case:no-such-classic-201-case-xyz",
	} {
		if index.has(decoy) {
			t.Fatalf("proof index unexpectedly accepted decoy %q", decoy)
		}
	}

	t.Run("nonexistent proof fails row validation", func(t *testing.T) {
		clone := cloneClassicCoverage201Rows(classicCoverage201Rows)
		clone[0].Proof = "TestClassicCoverage201NoSuchTestXYZ"
		if err := validateClassicCoverage201ProofResolution(clone); err == nil {
			t.Fatal("expected a nonexistent proof to fail validation")
		}
	})

	t.Run("unresolved row without a reason fails", func(t *testing.T) {
		clone := cloneClassicCoverage201Rows(classicCoverage201Rows)
		clone[0].Unresolved = true
		clone[0].Proof = ""
		clone[0].Reason = ""
		if err := validateClassicCoverage201ProofResolution(clone); err == nil {
			t.Fatal("expected an unresolved row without a reason to fail validation")
		}
	})

	t.Run("resolved row with an empty proof fails", func(t *testing.T) {
		clone := cloneClassicCoverage201Rows(classicCoverage201Rows)
		clone[0].Proof = ""
		if err := validateClassicCoverage201ProofResolution(clone); err == nil {
			t.Fatal("expected a resolved row with an empty proof to fail validation")
		}
	})

	t.Run("unresolved row carrying a proof reference fails", func(t *testing.T) {
		clone := cloneClassicCoverage201Rows(classicCoverage201Rows)
		clone[0].Unresolved = true
		clone[0].Reason = "not yet delivered"
		if err := validateClassicCoverage201ProofResolution(clone); err == nil {
			t.Fatal("expected an unresolved row carrying a proof reference to fail validation")
		}
	})

	t.Run("wrong cardinality fails naming the difference", func(t *testing.T) {
		clone := append(cloneClassicCoverage201Rows(classicCoverage201Rows), classicCoverage201Row{
			Type: "decision", ID: "SYN-201-01", Requirement: "WORK-01", Disposition: "keep-current", Proof: "TestOneTaskBugFixIsOneWorkerPlusChecks",
		})
		err := validateClassicCoverage201ExactSets(clone)
		if err == nil || !strings.Contains(err.Error(), "SYN-201-01") {
			t.Fatalf("duplicate-decision error = %v, want it to name SYN-201-01", err)
		}
	})
}

func cloneClassicCoverage201Rows(rows []classicCoverage201Row) []classicCoverage201Row {
	clone := make([]classicCoverage201Row, len(rows))
	copy(clone, rows)
	return clone
}

// validateClassicCoverage201ExactSets asserts the ledger has exactly one row
// per SYN-201 decision (14, from classicContractPhase201Decisions) and
// exactly one row per Phase 201 capability (8, from
// classicContractPhase201Capabilities) -- both vocabularies owned by
// classic_contract_test.go, never a second hardcoded list here. Any other
// cardinality fails naming the difference.
func validateClassicCoverage201ExactSets(rows []classicCoverage201Row) error {
	decisionSeen := make(map[string]int, len(classicContractPhase201Decisions))
	capabilitySeen := make(map[string]int, len(classicContractPhase201Capabilities))
	for _, row := range rows {
		switch row.Type {
		case "decision":
			decisionSeen[row.ID]++
		case "capability":
			capabilitySeen[row.ID]++
		default:
			return fmt.Errorf("unknown coverage row type %q for %q", row.Type, row.ID)
		}
	}
	if len(decisionSeen) != len(classicContractPhase201Decisions) {
		return fmt.Errorf("coverage has %d unique decision rows, want exactly %d", len(decisionSeen), len(classicContractPhase201Decisions))
	}
	for _, id := range classicContractPhase201Decisions {
		if decisionSeen[id] != 1 {
			return fmt.Errorf("decision %s has %d rows, want exactly 1", id, decisionSeen[id])
		}
	}
	if len(capabilitySeen) != len(classicContractPhase201Capabilities) {
		return fmt.Errorf("coverage has %d unique capability rows, want exactly %d", len(capabilitySeen), len(classicContractPhase201Capabilities))
	}
	for _, id := range classicContractPhase201Capabilities {
		if capabilitySeen[id] != 1 {
			return fmt.Errorf("capability %s has %d rows, want exactly 1", id, capabilitySeen[id])
		}
	}
	return nil
}

// validateClassicCoverage201Fields requires every row to name a requirement
// and a disposition drawn from the same closed vocabulary the Classic
// contract schema itself uses.
func validateClassicCoverage201Fields(rows []classicCoverage201Row) error {
	for _, row := range rows {
		if strings.TrimSpace(row.Requirement) == "" {
			return fmt.Errorf("%s/%s has no requirement", row.Type, row.ID)
		}
		switch row.Disposition {
		case "keep-current", "restore-modern", "replace-better", "retire-with-proof":
		default:
			return fmt.Errorf("%s/%s has unsupported disposition %q", row.Type, row.ID, row.Disposition)
		}
	}
	return nil
}

// validateClassicCoverage201ProofResolution requires every non-unresolved
// row to carry a non-empty proof that resolves to either a declared
// top-level Go test in the cmd package or a case identifier present in the
// Phase 201 corpus -- resolved from parsed declarations via the same
// buildClassicCoverage199ProofIndex the Phase 199 ledger already uses, so a
// commented-out test or a matching string literal is rejected exactly as it
// is there. A row may be explicitly Unresolved with a stated Reason instead
// of a proof, but never both.
func validateClassicCoverage201ProofResolution(rows []classicCoverage201Row) error {
	root := findTestModuleRootForClassicCoverage199()
	index, err := buildClassicCoverage199ProofIndex(root)
	if err != nil {
		return err
	}
	for _, row := range rows {
		if row.Unresolved {
			if strings.TrimSpace(row.Reason) == "" {
				return fmt.Errorf("%s/%s is unresolved without a stated reason", row.Type, row.ID)
			}
			if strings.TrimSpace(row.Proof) != "" {
				return fmt.Errorf("%s/%s is unresolved but carries a proof reference %q", row.Type, row.ID, row.Proof)
			}
			continue
		}
		if strings.TrimSpace(row.Proof) == "" {
			return fmt.Errorf("%s/%s is resolved but has no proof reference", row.Type, row.ID)
		}
		if !index.has(row.Proof) {
			return fmt.Errorf("%s/%s proof %q does not resolve to a declared test or a corpus case", row.Type, row.ID, row.Proof)
		}
	}
	return nil
}
