package cmd

import (
	"github.com/calcosmic/Aether/pkg/colony"
)

// continuePassSourceDeterministicFloor names the sole source of a pass on
// every continue lane: the deterministic floor (runDeterministicFloor,
// cmd/deterministic_floor.go). A decision only ever carries this pass
// source when it advances -- a dispatched reviewer's verdict can only ADD a
// block on top of an already-passing gate report, never supply the pass
// itself (TestDeterministicFloorIsTheOnlySourceOfAPass,
// cmd/deterministic_floor_test.go).
const continuePassSourceDeterministicFloor = "deterministic_floor"

// continueAdvanceVerdict is the two-value outcome runContinueAcceptVerifyAdvance
// returns: either the phase advances, or it blocks.
type continueAdvanceVerdict string

const (
	continueAdvanceVerdictAdvance continueAdvanceVerdict = "advance"
	continueAdvanceVerdictBlock   continueAdvanceVerdict = "block"
)

// continueAcceptVerifyAdvanceDecision is the one typed decision value every
// continue lane -- the direct lane (runCodexContinue), the snapshot/plan-only
// lane (runCodexContinuePlanOnly) and the external finalize lane
// (runCodexContinueFinalize) -- reaches through runContinueAcceptVerifyAdvance,
// and only through it. It carries the advancement verdict, the phase it
// applies to, the named pass source, the ordered blocking reasons, and how
// any supplied reconciliation record was disposed.
type continueAcceptVerifyAdvanceDecision struct {
	Verdict continueAdvanceVerdict
	Phase   int
	// PassSource names why the phase is allowed to advance. It is always
	// continuePassSourceDeterministicFloor when Verdict is advance, and
	// empty when Verdict is block -- a blocked decision has no pass to
	// attribute a source to.
	PassSource string
	// BlockingReasons is the ordered set of reasons the phase did not
	// advance. Empty when Verdict is advance.
	BlockingReasons []string
	// Warnings carries the non-blocking observations the gate report
	// already surfaces (codexContinueGateReport.Warnings) -- passed through
	// unchanged, never re-derived.
	Warnings []string
	// ReconciliationAccepted and ReconciledTasks report how a supplied
	// reconciliation record (assessment.ReconciledTasks, populated from
	// --reconcile-task on whichever lane supplied it) was disposed. This is
	// the single acceptance rule every lane shares: a reconciled task still
	// blocks if its own deterministic floor failed
	// (continueTasksSupportAdvancement, cmd/codex_continue.go), but the
	// SUPPLYING of a reconciliation record is never itself accepted or
	// refused differently lane to lane -- it is folded into assessment
	// (assessCodexContinue) identically everywhere, and this decision body
	// reports that single disposition regardless of which lane called it.
	ReconciliationAccepted bool
	ReconciledTasks        []string
	// PartialSuccess mirrors assessment.PartialSuccess: the phase can
	// advance with recorded, non-blocking operational issues.
	PartialSuccess bool
	// Gates is the full gate report the decision was computed from, so a
	// caller that already needs the granular per-check detail (queen
	// advisory decisions, persisted gate results, blocked-result rendering)
	// does not have to re-derive it -- it is exactly the report passed in.
	Gates codexContinueGateReport
}

// Advances reports whether this decision allows the phase to advance.
func (d continueAcceptVerifyAdvanceDecision) Advances() bool {
	return d.Verdict == continueAdvanceVerdictAdvance
}

// runContinueAcceptVerifyAdvance is the single accept/verify/advance
// decision body every continue lane reaches through, and only through, to
// determine whether a phase advances (SYN-201-04). Modeled on
// runDeterministicFloor's own doc comment (cmd/deterministic_floor.go): each
// lane keeps its own untrusted-input parsing and I/O responsibilities --
// the finalize lane's completion-contract validation and its own
// soft_block-gate auto-resolve/recovery-orchestration side effects, the
// plan-only lane's dispatch-manifest assembly, the direct lane's in-process
// reviewer dispatch -- and calls this body only with already-normalized
// inputs: an already-evaluated gate report (runCodexContinueGates /
// runCodexContinueGatesWithAutopilotBaseline, the single shared gate
// evaluator every lane already called before this plan) and, once one
// exists, an already-computed reviewer report. This function performs no
// I/O and mutates nothing; it is the fold of "did the gates pass" and "did
// the reviewer add a block" into one verdict, so lane parity is structural
// rather than a discipline to keep in sync by hand
// (TestAllThreeContinueLanesShareOneAcceptVerifyAdvanceBody).
//
// Taking the gate report as an input, rather than re-evaluating it inside
// this function, is deliberate: the finalize lane's soft_block auto-resolve
// pass (cmd/codex_continue_finalize.go) mutates its own gates value BEFORE
// the accept/verify/advance decision is made, and recomputing gates from
// scratch inside this function would silently discard that resolution.
// Every lane still funnels through the one shared gate evaluator -- it is
// simply each lane's own responsibility to call it (and apply its own
// lane-specific recovery steps, if any) before reaching this decision body.
//
// review may be nil: a caller deciding whether a reviewer needs to be
// dispatched at all calls this once with review == nil to evaluate the gate
// report alone (mirroring the existing "block on gates before ever
// dispatching a reviewer" behavior -- an already-failing deterministic
// floor should never pay for a review wave), then calls it again with the
// real review report once one exists. A nil review can never itself cause a
// block; only a non-nil review whose own Passed is false can.
//
// runDeterministicFloor remains the ONLY source of a pass: gates.Passed can
// only be true when verification.ChecksPassed was true (the
// verification_steps_passed gate mirrors it directly), and
// verification.ChecksPassed is itself entirely determined by the
// deterministic floor (runDeterministicFloor). A non-nil review's blocking
// issues can only ADD to an already-passing gate report; a passing review
// can never flip an already-failing gate report to advance.
func runContinueAcceptVerifyAdvance(
	phase colony.Phase,
	assessment codexContinueAssessment,
	gates codexContinueGateReport,
	review *codexContinueReviewReport,
	state colony.ColonyState,
) continueAcceptVerifyAdvanceDecision {
	decision := continueAcceptVerifyAdvanceDecision{
		Phase:                  phase.ID,
		ReconciliationAccepted: len(assessment.ReconciledTasks) > 0,
		ReconciledTasks:        append([]string{}, assessment.ReconciledTasks...),
		PartialSuccess:         assessment.PartialSuccess,
		Gates:                  gates,
		Warnings:               append([]string{}, gates.Warnings...),
	}

	if !gates.Passed {
		decision.Verdict = continueAdvanceVerdictBlock
		decision.BlockingReasons = append([]string{}, gates.BlockingIssues...)
		return decision
	}

	if review != nil && !review.Passed {
		decision.Verdict = continueAdvanceVerdictBlock
		decision.BlockingReasons = append([]string{}, review.BlockingIssues...)
		return decision
	}

	decision.Verdict = continueAdvanceVerdictAdvance
	decision.PassSource = continuePassSourceDeterministicFloor
	return decision
}
