package cmd

import (
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/codex"
)

// Phase 210 blocker 7 (French Basics, 2026-09-26): builder Anvil-7 found
// its three tasks already done, checked each one (every task receipt
// verification_status "pass" with the command it ran) and honestly marked
// the overall handoff "partial" because a full pytest run it also started
// timed out before it could read the result. The no-change rule read only
// the overall line and refused the whole wave. Per-task receipts that all
// passed with named commands are the evidence; the deterministic floor at
// continue still runs the full suite itself. A failing receipt, or an
// overall "fail", is still refused.
func anvil7NoChangeReceipts(status11 string) []codex.TaskReceipt {
	return []codex.TaskReceipt{
		{TaskID: "1.1", Status: "completed_no_change", Summary: "Rules file exists, 14 bullet lines, each rule cites its source.",
			Handoff: codex.WorkerHandoff{CommandsRun: []string{"wc -l docs/FRENCH_BASICS_RULES.md"}, VerificationStatus: status11}},
		{TaskID: "1.2", Status: "completed_no_change", Summary: "Overlap table is present at the bottom of docs/FRENCH_BASICS_BLOCKS.md.",
			Handoff: codex.WorkerHandoff{CommandsRun: []string{"sed -n 20,153p docs/FRENCH_BASICS_BLOCKS.md"}, VerificationStatus: "pass"}},
		{TaskID: "1.3", Status: "completed_no_change", Summary: "103 numbered blocks in study order.",
			Handoff: codex.WorkerHandoff{CommandsRun: []string{"grep -cE '^[0-9]+\\. ' docs/FRENCH_BASICS_BLOCKS.md"}, VerificationStatus: "pass"}},
	}
}

func anvil7NoChangeResult(overall string, receipts []codex.TaskReceipt) codex.DispatchResult {
	return codex.DispatchResult{
		WorkerName: "Anvil-7",
		Status:     "completed_no_change",
		WorkerResult: &codex.WorkerResult{
			WorkerName:   "Anvil-7",
			Status:       "completed_no_change",
			Summary:      "docs/FRENCH_BASICS_RULES.md and docs/FRENCH_BASICS_BLOCKS.md were already written, so I checked them and changed nothing.",
			TaskReceipts: receipts,
			Handoff: codex.WorkerHandoff{
				CommandsRun:        []string{"python3 scripts/validate.py", "wc -l docs/FRENCH_BASICS_*.md"},
				VerificationStatus: overall,
				KnownFailures:      []string{"python3 -m pytest -q timed out at 120s; result not read"},
			},
		},
	}
}

func TestNoChangeProvenTaskByTaskIsAccepted(t *testing.T) {
	if err := validateRuntimeNoChangeEvidence([]codex.DispatchResult{anvil7NoChangeResult("partial", anvil7NoChangeReceipts("pass"))}); err != nil {
		t.Fatalf("three passing task receipts were refused because the overall line was partial: %v", err)
	}
	external := codexExternalBuildWorkerResult{
		Status:       "completed_no_change",
		Summary:      "already written; checked and changed nothing",
		TaskReceipts: anvil7NoChangeReceipts("pass"),
		Handoff:      codex.WorkerHandoff{CommandsRun: []string{"python3 scripts/validate.py"}, VerificationStatus: "partial"},
	}
	if missing := noChangeEvidenceMissing(external); len(missing) != 0 {
		t.Fatalf("external lane refused passing task receipts: %v", missing)
	}
}

func TestNoChangeWithoutProofIsStillRefused(t *testing.T) {
	cases := map[string]codex.DispatchResult{
		"one receipt not run":     anvil7NoChangeResult("partial", anvil7NoChangeReceipts("not_run")),
		"overall fail":            anvil7NoChangeResult("fail", anvil7NoChangeReceipts("pass")),
		"no receipts and partial": anvil7NoChangeResult("partial", nil),
	}
	for name, result := range cases {
		if err := validateRuntimeNoChangeEvidence([]codex.DispatchResult{result}); err == nil || !strings.Contains(err.Error(), "without evidence") {
			t.Fatalf("%s: unproven no-change was accepted (err=%v)", name, err)
		}
	}
	noCommand := anvil7NoChangeReceipts("pass")
	noCommand[2].Handoff.CommandsRun = nil
	if err := validateRuntimeNoChangeEvidence([]codex.DispatchResult{anvil7NoChangeResult("partial", noCommand)}); err == nil {
		t.Fatal("a receipt naming no command it ran was accepted as proof")
	}
}
