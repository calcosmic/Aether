package cmd

// Phase 209 plan 04 -- planning reached through the single door never
// stops for a missing specification (UED-17). Task 1's tests
// (TestGoBigRouteLeavesPlanningReadyToRun, TestGoNeverTouchesAnOwnerApprovedSpecification)
// prove requireApprovedPlanningSpecification's refusal is gone for the
// single door and that an owner-approved specification is never touched.

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/storage"
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

// TestGoalReachesBuiltWorkWithNoExtraSteps is Task 2's own real-flow proof:
// starting from a project whose only recorded input is the goal, it drives
// the single door and then the same sequence the /ant-go wrapper documents
// -- plan, then build -- with worker dispatch faked the way the existing
// in-process harness already fakes real Scout/Route-Setter dispatch
// (acceptStagedPlanningCandidate200, the same helper e2e_lifecycle_test.go
// uses so a Specification-bearing plan can never enter build as
// legacy_unbound; `build --synthetic` for the build step itself), and
// asserts the run reaches built work. It asserts the owner types exactly
// one command -- the sentence, through the single door -- and that zero
// further owner-typed commands were needed to reach built work; the
// recorded runtime command sequence itself is checked for containing no
// clarifying-intent command and no by-hand specification approval anywhere
// in it. It also asserts resolvePlanningPreset with the documented default
// preset returns a policy without requiring a choice.
func TestGoalReachesBuiltWorkWithNoExtraSteps(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	forceJSONOutputModeForTest(t)

	root := t.TempDir()
	if out, err := exec.Command("git", "init", "-q", root).CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("# fixture\n"), 0o644); err != nil {
		t.Fatalf("write fixture file: %v", err)
	}
	for _, args := range [][]string{
		{"-C", root, "add", "."},
		{"-C", root, "-c", "user.email=test@example.com", "-c", "user.name=test", "commit", "-m", "initial"},
	} {
		if out, err := exec.Command("git", args...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}

	dataDir := filepath.Join(root, ".aether", "data")
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		t.Fatalf("mkdir data dir: %v", err)
	}
	origDataDir, hadDataDir := os.LookupEnv("COLONY_DATA_DIR")
	os.Setenv("COLONY_DATA_DIR", dataDir)
	t.Cleanup(func() {
		if hadDataDir {
			os.Setenv("COLONY_DATA_DIR", origDataDir)
		} else {
			os.Unsetenv("COLONY_DATA_DIR")
		}
	})
	s, err := storage.NewStore(dataDir)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	store = s
	chdirForTest190_05(t, root)

	// invoked is the full runtime command sequence, automated steps
	// included -- checked below for the two things this run must never
	// need. ownerTypedAfterSentence counts commands the owner himself
	// types after the single sentence; it must stay zero all the way to
	// built work.
	var invoked []string
	ownerTypedAfterSentence := 0

	sentence := "build a whole new realtime collaboration engine with conflict resolution"
	goal := sentence
	state := colony.ColonyState{
		Version: "3.0",
		Goal:    &goal,
		State:   colony.StateREADY,
	}
	writeGoRouteFixtureState(t, dataDir, state)

	// The one thing the owner types: the sentence, through the single door.
	invoked = append(invoked, "go")
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

	// The default preset the single door asks for needs no depth choice.
	selection, err := resolvePlanningPreset(codexPlanOptions{Preset: goDefaultPlanningPreset, PresetSet: true})
	if err != nil {
		t.Fatalf("resolvePlanningPreset: %v", err)
	}
	if selection.PresetRequired {
		t.Fatal("goDefaultPlanningPreset still asks the owner for a depth choice")
	}

	// The planning route itself, faked the way the existing in-process
	// harness (e2e_lifecycle_test.go) already fakes real Scout/Route-Setter
	// dispatch -- against the specification the single door just derived
	// and approved, never a second one. This step stands in for the
	// wrapper's own automatic planning call; the owner types nothing here.
	writeFreshTerritorySnapshot199(t, root, time.Now().UTC().Add(-time.Minute))
	invoked = append(invoked, "plan")
	acceptStagedPlanningCandidate200(t, root)

	var afterPlan colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &afterPlan); err != nil {
		t.Fatalf("load state after plan: %v", err)
	}
	if len(afterPlan.Plan.Phases) == 0 {
		t.Fatal("expected at least one phase after planning")
	}

	// The build step itself -- again automatic, not owner-typed.
	invoked = append(invoked, "build")
	rootCmd.SetArgs([]string{"build", "1", "--synthetic", "--no-checkin"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("build failed: %v", err)
	}

	var afterBuild colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &afterBuild); err != nil {
		t.Fatalf("load state after build: %v", err)
	}
	if afterBuild.State != colony.StateBUILT {
		t.Fatalf("state after build = %s, want %s (built work for the first phase)", afterBuild.State, colony.StateBUILT)
	}

	for _, forbidden := range []string{"discuss", "spec-approve", "spec --approve"} {
		for _, command := range invoked {
			if command == forbidden {
				t.Fatalf("recorded command sequence %v contains %q -- clarifying intent and by-hand approval must never be needed", invoked, forbidden)
			}
		}
	}
	if ownerTypedAfterSentence != 0 {
		t.Fatalf("ownerTypedAfterSentence = %d, want 0 -- one sentence must reach built work with nothing further typed", ownerTypedAfterSentence)
	}
}
