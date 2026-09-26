package cmd

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

// The owner's interview adds and answers questions after the first draft
// description already exists. These tests drive the same functions the
// `aether discuss --add-question`, `--resolve` and bare `aether discuss`
// commands call, so an answer that never reaches the description the owner
// approves fails here by name.

const lateAnswerText = "Keep the offline cache for thirty days"

func settleLateAnswerFixture(t *testing.T, suffix string) (root string, first discussSpecificationCloseout) {
	t.Helper()
	saveGlobals(t)
	dataDir := setupBuildFlowTest(t)
	root = filepath.Dir(filepath.Dir(dataDir))
	seedSettledDiscussState(t, dataDir, suffix)

	result, err := runDiscuss(root, 3, false)
	if err != nil {
		t.Fatalf("first settled discuss: %v", err)
	}
	closeout, ok := result["draft_spec"].(discussSpecificationCloseout)
	if !ok {
		t.Fatalf("first settled discuss returned no draft description: %#v", result)
	}
	return root, closeout
}

func addLateOwnerQuestion(t *testing.T) string {
	t.Helper()
	added, err := addComposedDiscussQuestion(
		"How long should offline data be kept?",
		lateAnswerText+"|Delete offline data on sign-out",
		"behavior",
		"The goal wording names an offline experience but no retention rule.",
		"offline-retention",
		false,
	)
	if err != nil {
		t.Fatalf("add late owner question: %v", err)
	}
	id := stringValue(added["id"])
	if id == "" || !boolValue(added["created"]) {
		t.Fatalf("late owner question was not materialized: %#v", added)
	}
	return id
}

func specificationBodyMentions(t *testing.T, revision colony.SpecRevision, text string) bool {
	t.Helper()
	body := discussResultJSONMap(t, discussSpecificationBody{
		Outcomes: revision.Outcomes, IncludedBehaviors: revision.IncludedBehaviors, Exclusions: revision.Exclusions,
		BindingDecisions: revision.BindingDecisions, Requirements: revision.Requirements, AcceptanceChecks: revision.AcceptanceChecks,
		NegativeExpectations: revision.NegativeExpectations, RecoveryExpectations: revision.RecoveryExpectations,
		AffectedPublicPaths: revision.AffectedPublicPaths,
	})
	for _, section := range body {
		for _, entry := range section.([]interface{}) {
			if strings.Contains(stringValue(entry.(map[string]interface{})["description"]), text) {
				return true
			}
		}
	}
	return false
}

func TestLateOwnerAnswerReachesTheDraftDescription(t *testing.T) {
	root, first := settleLateAnswerFixture(t, "late-answer")

	id := addLateOwnerQuestion(t)
	resolved, err := resolveDiscussQuestion(id, lateAnswerText)
	if err != nil {
		t.Fatalf("resolve late owner question: %v", err)
	}

	result, err := runDiscuss(root, 3, false)
	if err != nil {
		t.Fatalf("settle discuss after the late answer: %v", err)
	}
	closeout, ok := result["draft_spec"].(discussSpecificationCloseout)
	if !ok {
		t.Fatalf("settled discuss returned no draft description: %#v", result)
	}
	if closeout.RevisionID == first.RevisionID {
		t.Fatalf("late owner answer left the draft description at revision %s; the answer never reached it", first.RevisionID)
	}
	if closeout.Status != colony.SpecStatusDraft {
		t.Fatalf("revised description status = %q, want draft (never approved on the owner's behalf)", closeout.Status)
	}
	if got := stringValue(discussResultJSONMap(t, resolved["draft_spec"])["revision_id"]); got != closeout.RevisionID {
		t.Fatalf("resolve reported revision %q, later discuss reported %q; both must name the revised draft", got, closeout.RevisionID)
	}

	state := mustReadSpecificationTestState(t, root)
	if state.Specification == nil || len(state.Specification.Revisions) != 2 {
		t.Fatalf("specification = %#v, want the first draft plus one successor", state.Specification)
	}
	predecessor, successor := state.Specification.Revisions[0], state.Specification.Revisions[1]
	if predecessor.ID != first.RevisionID || predecessor.Status != colony.SpecStatusSuperseded {
		t.Fatalf("first draft = %s (%s), want %s superseded", predecessor.ID, predecessor.Status, first.RevisionID)
	}
	if successor.ID != closeout.RevisionID || successor.PredecessorID != first.RevisionID || successor.Approval != nil {
		t.Fatalf("successor = %#v, want an unapproved draft that follows %s", successor, first.RevisionID)
	}
	if !specificationBodyMentions(t, successor, lateAnswerText) {
		t.Fatalf("revised draft body does not carry the owner's answer %q: %#v", lateAnswerText, successor.BindingDecisions)
	}
	projection := mustReadSpecificationTestProjection(t, root)
	for _, want := range []string{lateAnswerText, successor.ID, "Status: `DRAFT`"} {
		if !strings.Contains(projection, want) {
			t.Fatalf("owner-readable description missing %q:\n%s", want, projection)
		}
	}
}

func TestSettledDiscussWithNoNewAnswerKeepsTheSameDraft(t *testing.T) {
	root, _ := settleLateAnswerFixture(t, "no-new-answer")
	id := addLateOwnerQuestion(t)
	if _, err := resolveDiscussQuestion(id, lateAnswerText); err != nil {
		t.Fatalf("resolve late owner question: %v", err)
	}
	settled, err := runDiscuss(root, 3, false)
	if err != nil {
		t.Fatalf("settle after the late answer: %v", err)
	}
	settledCloseout := settled["draft_spec"].(discussSpecificationCloseout)
	beforeState := mustReadSpecificationTestStateBytes(t, root)
	beforeProjection := mustReadSpecificationTestProjection(t, root)

	again, err := runDiscuss(root, 3, false)
	if err != nil {
		t.Fatalf("rerun discuss with nothing new: %v", err)
	}
	againCloseout := again["draft_spec"].(discussSpecificationCloseout)
	if againCloseout.RevisionID != settledCloseout.RevisionID || againCloseout.ContentHash != settledCloseout.ContentHash {
		t.Fatalf("rerun with nothing new moved the draft from %s to %s", settledCloseout.RevisionID, againCloseout.RevisionID)
	}
	if !bytes.Equal(beforeState, mustReadSpecificationTestStateBytes(t, root)) || beforeProjection != mustReadSpecificationTestProjection(t, root) {
		t.Fatal("rerun with nothing new rewrote the saved description")
	}
}

func TestApprovedDescriptionIsNeverRevisedByDiscuss(t *testing.T) {
	root, first := settleLateAnswerFixture(t, "approved-late")
	state := mustReadSpecificationTestState(t, root)
	approvedAt := time.Now().UTC()
	if _, err := approveSpecification(root, specificationApprovalRequest{
		RevisionID:          first.RevisionID,
		RevisionContentHash: first.ContentHash,
		ApprovalToken:       specificationApprovalToken(state.Specification.ID, first.RevisionID, first.ContentHash),
		ApprovedBy:          "owner",
		ApprovedAt:          approvedAt,
	}, specificationMutationOptions{}); err != nil {
		t.Fatalf("approve the first draft: %v", err)
	}
	beforeState := mustReadSpecificationTestStateBytes(t, root)
	beforeProjection := mustReadSpecificationTestProjection(t, root)

	id := addLateOwnerQuestion(t)
	resolved, err := resolveDiscussQuestion(id, lateAnswerText)
	if err != nil {
		t.Fatalf("resolve late owner question: %v", err)
	}
	result, err := runDiscuss(root, 3, false)
	if err != nil {
		t.Fatalf("discuss after approval: %v", err)
	}
	if !bytes.Equal(beforeState, mustReadSpecificationTestStateBytes(t, root)) || beforeProjection != mustReadSpecificationTestProjection(t, root) {
		t.Fatal("discuss revised or rewrote an approved description")
	}
	for name, closeoutValue := range map[string]interface{}{"resolve": resolved["approved_spec"], "discuss": result["approved_spec"]} {
		closeout := discussResultJSONMap(t, closeoutValue)
		if !boolValue(closeout["approved_spec_preserved"]) || stringValue(closeout["revision_id"]) != first.RevisionID ||
			stringValue(closeout["status"]) != string(colony.SpecStatusApproved) {
			t.Fatalf("%s closeout = %#v, want approved revision %s preserved", name, closeout, first.RevisionID)
		}
	}
}

func TestDiscussDryRunNeverRevisesTheDraft(t *testing.T) {
	root, first := settleLateAnswerFixture(t, "dry-run-late")
	id := addLateOwnerQuestion(t)

	// Resolve exactly as resolveDiscussQuestion stores an answer, but without
	// its automatic non-dry settlement, so the dry run is the first settle
	// that sees the answer.
	file := loadPendingDecisionFile()
	scope := loadCurrentPendingDecisionScope()
	for index := range file.Decisions {
		if file.Decisions[index].ID == id {
			file.Decisions[index].Resolved = true
			file.Decisions[index].Resolution = lateAnswerText
			file.Decisions[index].ResolvedAt = time.Now().UTC().Format(time.RFC3339)
			stampPendingDecisionScope(&file.Decisions[index], scope)
		}
	}
	if err := store.SaveJSON(pendingDecisionsFile, file); err != nil {
		t.Fatalf("store the late answer: %v", err)
	}
	beforeState := mustReadSpecificationTestStateBytes(t, root)
	beforeProjection := mustReadSpecificationTestProjection(t, root)

	result, err := runDiscuss(root, 3, true)
	if err != nil {
		t.Fatalf("dry-run discuss: %v", err)
	}
	if !bytes.Equal(beforeState, mustReadSpecificationTestStateBytes(t, root)) || beforeProjection != mustReadSpecificationTestProjection(t, root) {
		t.Fatal("dry-run discuss wrote a revised description")
	}
	if closeout, ok := result["draft_spec"].(discussSpecificationCloseout); !ok || closeout.RevisionID != first.RevisionID {
		t.Fatalf("dry-run closeout = %#v, want the unchanged draft %s", result["draft_spec"], first.RevisionID)
	}

	// The same stored answer does reach the draft on a real run, so the
	// dry run above was genuinely withholding a pending revision.
	real, err := runDiscuss(root, 3, false)
	if err != nil {
		t.Fatalf("real discuss after dry run: %v", err)
	}
	if closeout := real["draft_spec"].(discussSpecificationCloseout); closeout.RevisionID == first.RevisionID {
		t.Fatal("stored late answer never reached the draft on a real run")
	}
}
