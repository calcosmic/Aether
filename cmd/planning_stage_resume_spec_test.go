package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

// newerApprovedSpecificationBinding is what the owner's NEXT approved
// specification revision looks like next to the one a parked run was made
// against: a different revision id and content hash, the old one as its
// predecessor, and its own approval receipt.
func newerApprovedSpecificationBinding(old planningStageSpecificationBinding) planningStageSpecificationBinding {
	next := old
	next.PredecessorRevisionID = old.RevisionID
	next.RevisionID = "spec-revision-" + planningStageTestHash("9")[:12]
	next.ContentHash = planningStageTestHash("9")
	next.ApprovalReceiptID = "spec-approval-" + planningStageTestHash("8")[:12]
	next.ApprovalReceiptHash = planningStageTestHash("8")
	return next
}

func resolveResumeAgainst(t *testing.T, root string, approved planningStageSpecificationBinding) *planningStageResume {
	t.Helper()
	var resume *planningStageResume
	err := withPlanningMutationSession(root, "test-resolve-resume-spec", func(session *planningMutationSession) error {
		var inner error
		resume, inner = resolvePlanningStageResume(session, approved)
		return inner
	})
	if err != nil {
		t.Fatalf("resolve planning stage resume: %v", err)
	}
	return resume
}

// TestParkedRunAgainstASupersededSpecificationIsNotResumed is the regression
// for a real downstream dead end (2026-09-21): the owner corrected and
// re-approved the specification while a planning run sat parked waiting on a
// helper. That run can never finish -- every finalizer refuses a result bound
// to a specification that is no longer the approved one -- but it still
// counted as "parked", so every `aether plan` resumed it and was refused
// again (nine times), and no command could move past it.
func TestParkedRunAgainstASupersededSpecificationIsNotResumed(t *testing.T) {
	root, routeManifest := planningStageResumeTestParkedRun(t)
	current := routeManifest.Specification

	// Not vacuous: against the specification it was made for, the run resumes.
	if resume := resolveResumeAgainst(t, root, current); resume == nil || resume.RunID != routeManifest.RunID {
		t.Fatalf("a parked run bound to the CURRENT specification was not resumed (%+v)", resume)
	}

	newer := newerApprovedSpecificationBinding(current)
	if resume := resolveResumeAgainst(t, root, newer); resume != nil {
		t.Fatalf("run %s is bound to superseded specification %s but was resumed against %s; it can never finish", resume.RunID, current.RevisionID, newer.RevisionID)
	}

	// The same through discovery, with no convenience pointer (the shape the
	// reporting project was in).
	if err := os.Remove(filepath.Join(store.BasePath(), planningIterationStateRel)); err != nil {
		t.Fatalf("remove iteration state: %v", err)
	}
	if resume := resolveResumeAgainst(t, root, newer); resume != nil {
		t.Fatalf("discovery resumed run %s although it is bound to a superseded specification", resume.RunID)
	}
	if resume := resolveResumeAgainst(t, root, current); resume == nil {
		t.Fatal("discovery no longer finds a parked run bound to the current specification")
	}

	// Nothing was deleted: the superseded run stays on disk as history.
	if _, err := os.Stat(filepath.Join(store.BasePath(), "planning", routeManifest.RunID, "stage-state.json")); err != nil {
		t.Fatalf("the superseded run's records were removed: %v", err)
	}
}
