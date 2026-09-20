package cmd

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/storage"
)

// TestTwoWaitingPlansPointStatusAtTheReview covers the state a planning
// restart leaves: two plans waiting for review. Status read that as planning
// evidence that was "incomplete or conflicting" and sent the owner to
// `aether resume`. It must send them to the review -- which names each waiting
// plan and how to review it by name -- and must not offer to accept either one,
// because choosing between them is the owner's decision.
func TestTwoWaitingPlansPointStatusAtTheReview(t *testing.T) {
	saveGlobals(t)
	root, first := planCandidateTestPending(t)
	planCandidateTestSecondPending(t, root, "planning-route-stage-restart")

	var err error
	store, err = storage.NewStore(filepath.Join(root, ".aether", "data"))
	if err != nil {
		t.Fatal(err)
	}
	state := mustReadSpecificationTestState(t, root)
	goal := "Choose between two waiting plans"
	state.Goal = &goal
	if err := store.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatal(err)
	}

	facts, err := loadLifecycleFacts(root, store, first.CreatedAt.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	answer := resolveNextAction(nextActionInput{Facts: facts})
	if answer.Projection == nil {
		t.Fatal("status answer carries no lifecycle projection")
	}
	action := answer.Projection.NextAction
	if action.ID != "review_plan_candidate" || action.RuntimeCommand != "aether plan --candidate" {
		t.Fatalf("next action = %q %q (%s), want the review, which names each waiting plan", action.ID, action.RuntimeCommand, action.Reason)
	}
	for _, alternative := range answer.Projection.Alternatives {
		if strings.Contains(alternative.RuntimeCommand, "--accept-candidate") {
			t.Fatalf("status offered %q while two plans wait; choosing between them is the owner's decision", alternative.RuntimeCommand)
		}
	}
}
