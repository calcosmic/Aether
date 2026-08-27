package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// This file guards CLAUDE.md's owner-facing description of Phase 195's
// coherent jobs, completion-evidence boundary, and one-worker check-in fast
// path.
//
// It exists because CLAUDE.md is exactly where the drift this repo keeps
// rediscovering happened again: Phase 194 wrote "Builds pause for the owner
// before spawning -- including a one-worker team" into CLAUDE.md, Phase 195
// plan 05 replaced that behavior with decideBuildCheckin's decision-aware fast
// path, and for the whole of Phase 195 nothing failed while the doc described
// behavior the runtime no longer had. Plan 195-09 closed the same gap for the
// five shipped command surfaces (YAML, Codex guide, Codex skill, and the three
// byte-identical build wrappers) with shared anchors plus a forbidden-anchor
// list; CLAUDE.md had no equivalent guard, which is why it was the surface
// that rotted.
//
// Per CLAUDE.md's own Definition of Done, a documentation claim about runtime
// behaviour must be testable or removed. These tests are the command that
// fails when the claim is untrue.
//
// The forbidden-anchor half is the load-bearing half. It only appears in Go
// comments here, never in the markdown under test, so a match is always a real
// regression rather than this test finding its own documentation.

const claudeMDPathForCoherentJobs = "../CLAUDE.md"

// retiredCheckinDocClaims are the pre-Phase-195 CLAUDE.md sentences that the
// one-worker fast path made untrue. None of them may reappear.
var retiredCheckinDocClaims = []string{
	"including a one-worker team",
	"Builds pause for the owner before spawning",
}

// requiredCoherentJobDocClaims are short, deliberately un-wrappable anchors for
// the facts CLAUDE.md must state about grouping, evidence, and the check-in
// decision. Each maps to a shipped runtime behaviour with its own named test.
var requiredCoherentJobDocClaims = []string{
	// One-worker fast path and its overrides (D-11, D-12, D-14).
	"goes straight through",
	"counts workers, not jobs of work",
	"`--checkin` and `--no-checkin` together is refused",
	// Grouping is validated in Go, before ownership, in both modes (D-01..D-07).
	"grouping pass runs first",
	"refused by name",
	"in-repo",
	"worktree",
	// The completion-evidence boundary (D-08, D-09).
	"`covered_task_ids` is assignment scope only",
	"grants no credit whatsoever",
	"can never become completion credit",
	"A partially finished job never reports the project as built",
	// Recovery (D-10).
	"only the uncredited tasks",
}

// citedCoherentJobTests are the test names CLAUDE.md cites as locking the
// behaviour it describes. Each must be BOTH named in CLAUDE.md and a real
// function in this package, so renaming or deleting a lock fails the doc.
var citedCoherentJobTests = []string{
	// Required by plan 195-10 as the folded todo's closure evidence.
	"TestOneWorkerBuildSkipsCheckin",
	"TestOneWorkerWithForcedReviewerWaiverStillPauses",
	"TestCheckinFlagConflictHasNoSideEffects",
	// The rest of the check-in decision surface.
	"TestBuildCheckinDecisionMatrix",
	"TestOneWorkerFastPathSummaryCarriesEveryFact",
	"TestFastPathSummaryIsNonBlocking",
	"TestOneWorkerWithBoundaryQuestionStillPauses",
	"TestOneWorkerWithPersistedOwnerDecisionStillPauses",
	"TestPendingDecisionStillRendersFullCheckinCard",
	// Grouping validation.
	"TestCoherentJobProposalOrderRefusedByName",
	"TestCoherentJobRepairKeepsSafeProposals",
	"TestCoherentJobGraphErrorsHaveNoPlan",
	"TestCoherentJobsIgnoreIncidentalPaths",
	"TestCalVaultSixBatchesPlanAsOneCoherentJob",
	// Grouping before ownership, both modes.
	"TestCalVaultSixBatchesBecomeOneWorktreeJob",
	"TestGroupedWorktreeOwnsUnionedPaths",
	"TestDistinctWorktreeJobsStillRejectOverlap",
	// Evidence boundary and partial credit.
	"TestCoherentJobReceiptAdmission",
	"TestCoherentJobReceiptFinalization",
	"TestGroupedWorktreePartialReceiptsSyncBeforeCredit",
	"TestGroupedWorktreeUncreditedEditsRemainOrphaned",
	"TestExternalGroupedPartialPersistsExactTaskState",
	// Recovery. The first two assert what planCoherentJobRetry RETURNS; the
	// third asserts what the command the owner is handed actually dispatches
	// (CR-04, 195-REVIEW.md -- both of the first two passed while the surfaced
	// command re-planned every credited task).
	"TestCoherentJobRetryContainsOnlyUnfinishedTasks",
	"TestGroupedJobRetryNeverReassignsCreditedTasks",
	"TestPartialRetryCommandNeverRedispatchesCreditedWork",
}

func readCLAUDEMDForCoherentJobs(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile(claudeMDPathForCoherentJobs)
	if err != nil {
		t.Fatalf("read %s: %v", claudeMDPathForCoherentJobs, err)
	}
	return string(data)
}

// TestCLAUDEMDDoesNotClaimUnconditionalCheckinPause fails if CLAUDE.md
// reintroduces the pre-Phase-195 claim that every build pauses, including a
// build with a single worker. The runtime is authoritative and stopped doing
// that in plan 195-05.
func TestCLAUDEMDDoesNotClaimUnconditionalCheckinPause(t *testing.T) {
	content := readCLAUDEMDForCoherentJobs(t)
	for _, banned := range retiredCheckinDocClaims {
		if strings.Contains(content, banned) {
			t.Errorf("CLAUDE.md still contains retired check-in claim %q -- decideBuildCheckin (cmd/ceremony_team_checkin.go) takes the one-worker fast path when no owner decision is pending", banned)
		}
	}
}

// TestCLAUDEMDStatesCoherentJobContract fails if CLAUDE.md stops stating any
// of the Phase 195 facts an owner needs: what makes one worker go straight
// through, that grouping is validated before file ownership in both execution
// modes, and that an assignment list is not completion credit.
func TestCLAUDEMDStatesCoherentJobContract(t *testing.T) {
	content := readCLAUDEMDForCoherentJobs(t)
	for _, claim := range requiredCoherentJobDocClaims {
		if !strings.Contains(content, claim) {
			t.Errorf("CLAUDE.md is missing required Phase 195 claim %q", claim)
		}
	}
}

// TestCLAUDEMDCitesLiveTestsForCoherentJobClaims fails if CLAUDE.md cites a
// lock that no longer exists, or stops citing one of the locks that make its
// claims checkable. The surrounding CLAUDE.md sections already name their own
// tests; this makes that convention enforceable rather than decorative.
func TestCLAUDEMDCitesLiveTestsForCoherentJobClaims(t *testing.T) {
	content := readCLAUDEMDForCoherentJobs(t)

	entries, err := filepath.Glob("*_test.go")
	if err != nil {
		t.Fatalf("glob package test files: %v", err)
	}
	var sources strings.Builder
	for _, entry := range entries {
		data, err := os.ReadFile(entry)
		if err != nil {
			t.Fatalf("read %s: %v", entry, err)
		}
		sources.Write(data)
		sources.WriteString("\n")
	}
	pkgSources := sources.String()

	for _, name := range citedCoherentJobTests {
		if !strings.Contains(content, "`"+name+"`") {
			t.Errorf("CLAUDE.md no longer cites %s as a lock for its Phase 195 claims", name)
		}
		if !strings.Contains(pkgSources, "func "+name+"(") {
			t.Errorf("CLAUDE.md cites %s but no such test function exists in package cmd -- the doc names a lock that cannot run", name)
		}
	}
}

// TestCLAUDEMDCheckinPrecedenceMatchesRuntime evaluates the documented
// precedence against decideBuildCheckin itself, so the doc cannot describe an
// order the policy does not implement. This mirrors
// TestCLAUDEMDVerificationDepthClaims, which likewise checks its documented
// table against the live resolver rather than against prose alone.
func TestCLAUDEMDCheckinPrecedenceMatchesRuntime(t *testing.T) {
	cases := []struct {
		name      string
		input     buildCheckinDecisionInput
		wantPause bool
		wantCode  buildCheckinReasonCode
	}{
		{
			name:      "one worker, nothing pending, goes straight through",
			input:     buildCheckinDecisionInput{ImplementationDispatches: 1},
			wantPause: false,
			wantCode:  buildCheckinReasonOneWorkerFastPath,
		},
		{
			name:      "one worker but an owner decision is pending, still pauses",
			input:     buildCheckinDecisionInput{ImplementationDispatches: 1, PendingOwnerDecision: true},
			wantPause: true,
			wantCode:  buildCheckinReasonPendingOwnerDecision,
		},
		{
			name:      "explicit --checkin forces the pause for one worker",
			input:     buildCheckinDecisionInput{ImplementationDispatches: 1, Checkin: true},
			wantPause: true,
			wantCode:  buildCheckinReasonExplicitCheckin,
		},
		{
			name:      "--no-checkin stays non-interactive",
			input:     buildCheckinDecisionInput{ImplementationDispatches: 3, NoCheckin: true},
			wantPause: false,
			wantCode:  buildCheckinReasonNonInteractive,
		},
		{
			name:      "more than one worker pauses as before",
			input:     buildCheckinDecisionInput{ImplementationDispatches: 3},
			wantPause: true,
			wantCode:  buildCheckinReasonDefaultPause,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got := decideBuildCheckin(tc.input)
			if got.Requested != tc.wantPause {
				t.Errorf("decideBuildCheckin(%+v).Requested = %v, want %v -- CLAUDE.md documents the opposite", tc.input, got.Requested, tc.wantPause)
			}
			if got.Reason != tc.wantCode {
				t.Errorf("decideBuildCheckin(%+v).Reason = %q, want %q", tc.input, got.Reason, tc.wantCode)
			}
		})
	}
}
