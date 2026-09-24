package cmd

// Phase 209 plan 04 -- planning reached through the single door never
// stops for a missing specification (UED-17). Task 1's tests
// (TestGoBigRouteLeavesPlanningReadyToRun, TestGoNeverTouchesAnOwnerApprovedSpecification)
// prove requireApprovedPlanningSpecification's refusal is gone for the
// single door and that an owner-approved specification is never touched.

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

// TestGoBigRouteLeavesPlanningReadyToRun drives the real `aether go` command
// body (runGoJob) over a project with a recorded goal and no specification
// at all -- the ordinary shape of a project reached purely through
// init-then-go, since aether init records an accepted charter and a goal
// but never a specification -- with a sentence the route authority sizes
// as big, and asserts requireApprovedPlanningSpecification (the exact call
// that refuses today) returns no error immediately afterwards. It also
// asserts the derived outcome carries the owner's sentence verbatim and
// that a second run with the same sentence leaves exactly one approved
// revision, not two.
func TestGoBigRouteLeavesPlanningReadyToRun(t *testing.T) {
	saveGlobals(t)
	s, root := newTestStore(t)
	store = s
	chdirForTest190_05(t, root)

	goal := "ship the new invoicing flow"
	state := colony.ColonyState{
		Version: "3.0",
		Goal:    &goal,
		State:   colony.StateREADY,
	}
	writeGoRouteFixtureState(t, filepath.Join(root, ".aether", "data"), state)

	sentence := "invent a brand new capability nothing here has ever heard of"
	result, err := runGoJob(sentence, 2*time.Second)
	if err != nil {
		t.Fatalf("runGoJob: %v", err)
	}
	if got := stringValue(result["route"]); got != string(jobSizeRouteBig) {
		t.Fatalf("route = %q, want %q", got, jobSizeRouteBig)
	}
	if got := stringValue(result["specification_source"]); got != goSpecificationSourceDerivedFromRequest {
		t.Fatalf("specification_source = %q, want %q", got, goSpecificationSourceDerivedFromRequest)
	}

	reloaded, err := loadSpecificationColonyState(root)
	if err != nil {
		t.Fatalf("reload colony state: %v", err)
	}
	if _, err := requireApprovedPlanningSpecification(root, reloaded); err != nil {
		t.Fatalf("requireApprovedPlanningSpecification refused after the single door's big route: %v", err)
	}
	if reloaded.Specification == nil {
		t.Fatal("expected a specification to exist after the big route")
	}
	current, ok := currentSpecificationRevision(*reloaded.Specification)
	if !ok {
		t.Fatal("expected a current specification revision")
	}
	if len(current.Outcomes) != 1 {
		t.Fatalf("expected exactly one outcome item, got %d", len(current.Outcomes))
	}
	if current.Outcomes[0].Description != sentence {
		t.Fatalf("outcome description = %q, want the owner's sentence verbatim %q", current.Outcomes[0].Description, sentence)
	}
	revisionsAfterFirstRun := len(reloaded.Specification.Revisions)

	// Running it again with the same sentence must not create a second
	// derived, approved revision -- it should find the first one already
	// settled and report the equivalent of "nothing to do".
	result2, err := runGoJob(sentence, 2*time.Second)
	if err != nil {
		t.Fatalf("runGoJob (second run): %v", err)
	}
	if got := stringValue(result2["specification_source"]); got != goSpecificationSourceOwnerApproved {
		t.Fatalf("specification_source (second run) = %q, want %q (already-settled, no change)", got, goSpecificationSourceOwnerApproved)
	}
	reloaded2, err := loadSpecificationColonyState(root)
	if err != nil {
		t.Fatalf("reload colony state (second run): %v", err)
	}
	if got := len(reloaded2.Specification.Revisions); got != revisionsAfterFirstRun {
		t.Fatalf("revision count changed on a no-op second run: %d -> %d", revisionsAfterFirstRun, got)
	}
}

// TestGoNeverTouchesAnOwnerApprovedSpecification approves a specification by
// hand -- through runSpecCommand's own Approve operation, exactly as an
// owner's real approval would be recorded -- before the single door ever
// runs, then drives the real `aether go` command body and asserts both the
// current revision ID and its content hash are unchanged, and that the
// result reports the specification as the owner's own.
func TestGoNeverTouchesAnOwnerApprovedSpecification(t *testing.T) {
	saveGlobals(t)
	s, root := newTestStore(t)
	store = s
	chdirForTest190_05(t, root)

	goal := "ship the new invoicing flow"
	sessionID := "sess-owner-hand"
	state := colony.ColonyState{
		Version:   "3.0",
		Goal:      &goal,
		SessionID: &sessionID,
		State:     colony.StateREADY,
	}
	writeGoRouteFixtureState(t, filepath.Join(root, ".aether", "data"), state)

	loaded, err := loadSpecificationColonyState(root)
	if err != nil {
		t.Fatalf("load colony state: %v", err)
	}
	specID, revID, hash, err := createGoDerivedSpecificationDraft(
		root, loaded, "an owner-authored specification, settled by hand", goAcceptedGoalEvidenceID(loaded),
	)
	if err != nil {
		t.Fatalf("build owner draft: %v", err)
	}
	token := colony.CanonicalSpecificationApprovalToken(specID, revID, hash)
	if _, err := runSpecCommand(root, specCommandOperationApprove, specCommandOptions{
		Approve:       true,
		RevisionID:    revID,
		RevisionHash:  hash,
		ApprovalToken: token,
		ApprovedBy:    "owner",
	}); err != nil {
		t.Fatalf("approve owner draft by hand: %v", err)
	}

	sentence := "invent a brand new capability nothing here has ever heard of"
	result, err := runGoJob(sentence, 2*time.Second)
	if err != nil {
		t.Fatalf("runGoJob: %v", err)
	}
	if got := stringValue(result["specification_source"]); got != goSpecificationSourceOwnerApproved {
		t.Fatalf("specification_source = %q, want %q", got, goSpecificationSourceOwnerApproved)
	}

	reloaded, err := loadSpecificationColonyState(root)
	if err != nil {
		t.Fatalf("reload colony state: %v", err)
	}
	if reloaded.Specification == nil {
		t.Fatal("expected the owner's specification to still exist")
	}
	if reloaded.Specification.CurrentRevisionID != revID {
		t.Fatalf("current revision changed: %q -> %q", revID, reloaded.Specification.CurrentRevisionID)
	}
	current, ok := currentSpecificationRevision(*reloaded.Specification)
	if !ok {
		t.Fatal("expected a current revision")
	}
	if current.ContentHash != hash {
		t.Fatalf("content hash changed: %q -> %q", hash, current.ContentHash)
	}
	if current.Approval == nil || current.Approval.ApprovedBy != "owner" {
		t.Fatalf("approval record changed: %+v", current.Approval)
	}
	if len(reloaded.Specification.Revisions) != 1 {
		t.Fatalf("expected exactly one revision, got %d", len(reloaded.Specification.Revisions))
	}
}
