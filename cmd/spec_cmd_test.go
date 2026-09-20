package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

func TestSpecCommandInspectRoundTripsEveryTypedSection(t *testing.T) {
	for _, scopeKind := range []colony.SpecScopeKind{colony.SpecScopeWholeGoal, colony.SpecScopeFeature} {
		t.Run(string(scopeKind), func(t *testing.T) {
			root := newSpecificationTestRepository(t, specificationTestPlanState())
			draft, err := createSpecificationDraft(root, specificationTestDraftRequest(t, scopeKind), specificationMutationOptions{})
			if err != nil {
				t.Fatalf("create draft: %v", err)
			}
			before := mustReadSpecificationTestStateBytes(t, root)

			result, output := runSpecCommandSuccess(t, root)

			if result.Operation != specCommandOperationInspect || result.ResultKind != specCommandResultInspection {
				t.Fatalf("inspect kind = %q/%q", result.Operation, result.ResultKind)
			}
			if result.BeforeRevisionID != draft.Revision.ID || result.AfterRevisionID != draft.Revision.ID || result.Status != colony.SpecStatusDraft {
				t.Fatalf("inspect revision = %#v", result)
			}
			assertSpecCommandBodyMatchesRevision(t, result, draft.Revision)
			if result.Scope.Kind != scopeKind || result.ClassifiedDelta.PredecessorRevisionID != draft.Revision.PredecessorID {
				t.Fatalf("inspect scope/delta = %#v / %#v", result.Scope, result.ClassifiedDelta)
			}
			if result.Projection.Drifted || result.NextAction == "" || !strings.Contains(result.NextAction, "--approval-token") {
				t.Fatalf("inspect projection/next = %#v / %q", result.Projection, result.NextAction)
			}
			if strings.Contains(output, `"items"`) {
				t.Fatalf("inspect collapsed typed sections into generic items: %s", output)
			}
			if after := mustReadSpecificationTestStateBytes(t, root); !bytes.Equal(before, after) {
				t.Fatal("inspect changed canonical state")
			}
		})
	}
}

func TestSpecCommandAddModifyRemoveAndExactReplay(t *testing.T) {
	root := newSpecificationTestRepository(t, specificationTestPlanState())
	request := specificationTestDraftRequest(t, colony.SpecScopeWholeGoal)
	request.Exclusions = append(request.Exclusions, specificationItemInput{
		Lineage:     "no-marketplace",
		Description: "Do not add a marketplace.",
		EvidenceIDs: []string{"charter:scope"},
	})
	draft, err := createSpecificationDraft(root, request, specificationMutationOptions{})
	if err != nil {
		t.Fatalf("create draft: %v", err)
	}

	addArgs := []string{
		"--add",
		"--section", string(specificationSectionIncludedBehaviors),
		"--lineage", "show-weakest-gap",
		"--text", "Show the weakest remaining planning gap after each pass.",
		"--evidence", "research:classic-loop",
		"--predecessor-revision", draft.Revision.ID,
	}
	added, _ := runSpecCommandSuccess(t, root, addArgs...)
	if added.Operation != specCommandOperationAdd || added.ResultKind != specCommandResultRevisionCreation || added.Replayed {
		t.Fatalf("add result = %#v", added)
	}
	if len(added.ClassifiedDelta.IncludedBehaviors.AddedIDs) != 1 {
		t.Fatalf("add delta = %#v", added.ClassifiedDelta.IncludedBehaviors)
	}

	replayed, _ := runSpecCommandSuccess(t, root, addArgs...)
	if !replayed.Replayed || replayed.AfterRevisionID != added.AfterRevisionID || replayed.Receipt == nil || added.Receipt == nil || replayed.Receipt.ReceiptID != added.Receipt.ReceiptID {
		t.Fatalf("exact add replay changed identity\nfirst: %#v\nagain: %#v", added, replayed)
	}

	wordingPath := filepath.Join(root, "requirement.txt")
	if err := os.WriteFile(wordingPath, []byte("The owner sees a causal Scout to Route-Setter planning loop.\n"), 0o644); err != nil {
		t.Fatalf("write source text: %v", err)
	}
	modified, _ := runSpecCommandSuccess(t, root,
		"--modify",
		"--section", string(specificationSectionRequirements),
		"--item-id", draft.Revision.Requirements[0].ID,
		"--file", wordingPath,
		"--evidence", "owner-answer:visible-loop",
		"--predecessor-revision", added.AfterRevisionID,
	)
	if modified.Operation != specCommandOperationModify || !reflect.DeepEqual(modified.ClassifiedDelta.Requirements.ModifiedIDs, []string{draft.Revision.Requirements[0].ID}) {
		t.Fatalf("modify result = %#v", modified)
	}

	removedID := draft.Revision.Exclusions[0].ID
	removed, _ := runSpecCommandSuccess(t, root,
		"--remove",
		"--section", string(specificationSectionExclusions),
		"--item-id", removedID,
		"--predecessor-revision", modified.AfterRevisionID,
	)
	if removed.Operation != specCommandOperationRemove || !reflect.DeepEqual(removed.ClassifiedDelta.Exclusions.RemovedIDs, []string{removedID}) {
		t.Fatalf("remove result = %#v", removed)
	}
	if len(mustReadSpecificationTestState(t, root).Specification.Revisions) != 4 {
		t.Fatal("add, modify, and remove did not create exactly three successor revisions")
	}
}

func TestSpecCommandApproveRequiresExactInputsAndReplaysReceipt(t *testing.T) {
	root := newSpecificationTestRepository(t, specificationTestPlanState())
	draft, err := createSpecificationDraft(root, specificationTestDraftRequest(t, colony.SpecScopeWholeGoal), specificationMutationOptions{})
	if err != nil {
		t.Fatalf("create draft: %v", err)
	}
	planBefore := mustReadSpecificationTestState(t, root).Plan
	token := specificationApprovalToken(draft.Specification.ID, draft.Revision.ID, draft.Revision.ContentHash)
	args := []string{
		"--approve",
		"--revision-id", draft.Revision.ID,
		"--revision-hash", draft.Revision.ContentHash,
		"--approval-token", token,
	}

	approved, _ := runSpecCommandSuccess(t, root, args...)
	if approved.Operation != specCommandOperationApprove || approved.ResultKind != specCommandResultApproval || approved.Status != colony.SpecStatusApproved {
		t.Fatalf("approval result = %#v", approved)
	}
	if approved.Approval == nil || approved.Approval.RevisionID != draft.Revision.ID || approved.Receipt == nil {
		t.Fatalf("approval receipts = %#v / %#v", approved.Approval, approved.Receipt)
	}
	if !reflect.DeepEqual(planBefore, mustReadSpecificationTestState(t, root).Plan) {
		t.Fatal("spec approval accepted or mutated the plan")
	}

	replayed, _ := runSpecCommandSuccess(t, root, args...)
	if !replayed.Replayed || replayed.Receipt == nil || replayed.Receipt.ReceiptID != approved.Receipt.ReceiptID {
		t.Fatalf("approval replay = %#v", replayed)
	}

	wrongRoot := newSpecificationTestRepository(t, colony.ColonyState{})
	wrongDraft, err := createSpecificationDraft(wrongRoot, specificationTestDraftRequest(t, colony.SpecScopeWholeGoal), specificationMutationOptions{})
	if err != nil {
		t.Fatalf("create wrong-token fixture: %v", err)
	}
	before := mustReadSpecificationTestStateBytes(t, wrongRoot)
	failure := runSpecCommandFailure(t, wrongRoot,
		"--approve",
		"--revision-id", wrongDraft.Revision.ID,
		"--revision-hash", wrongDraft.Revision.ContentHash,
		"--approval-token", "yes",
	)
	if !strings.Contains(failure, "approval token") {
		t.Fatalf("generic approval refusal = %q", failure)
	}
	if !bytes.Equal(before, mustReadSpecificationTestStateBytes(t, wrongRoot)) {
		t.Fatal("generic approval changed state")
	}
}

func TestSpecCommandRepairProjectionWithoutChangingAuthority(t *testing.T) {
	root := newSpecificationTestRepository(t, specificationTestPlanState())
	_, err := createSpecificationDraft(root, specificationTestDraftRequest(t, colony.SpecScopeFeature), specificationMutationOptions{})
	if err != nil {
		t.Fatalf("create draft: %v", err)
	}
	before := mustReadSpecificationTestStateBytes(t, root)
	if err := os.WriteFile(filepath.Join(root, specificationProjectionRelativePath), []byte("# forged\nStatus: APPROVED\n"), 0o644); err != nil {
		t.Fatalf("tamper projection: %v", err)
	}

	repaired, _ := runSpecCommandSuccess(t, root, "--repair-projection")
	if repaired.Operation != specCommandOperationProjectionRepair || repaired.ResultKind != specCommandResultProjectionRepair || !repaired.ProjectionRepaired || repaired.Receipt == nil {
		t.Fatalf("repair result = %#v", repaired)
	}
	if repaired.Projection.Drifted || !bytes.Equal(before, mustReadSpecificationTestStateBytes(t, root)) {
		t.Fatal("projection repair changed authority or left drift")
	}

	clean, _ := runSpecCommandSuccess(t, root, "--repair-projection")
	if clean.ProjectionRepaired || clean.Receipt != nil || clean.OutcomeKind != colony.OutcomeKindNoChange {
		t.Fatalf("clean repair should be a read-only no-op: %#v", clean)
	}
}

func TestSpecCommandRejectsInvalidCombinationsBeforeStateAccess(t *testing.T) {
	root := t.TempDir()
	failure := runSpecCommandFailure(t, root,
		"--add", "--remove",
		"--section", string(specificationSectionOutcomes),
		"--item-id", "outcome-1",
		"--text", "Changed text",
		"--evidence", "owner:answer",
		"--predecessor-revision", "spec-revision-missing",
	)
	if !strings.Contains(failure, "mutually exclusive") {
		t.Fatalf("invalid operation failure = %q", failure)
	}
	if _, err := os.Stat(filepath.Join(root, ".aether")); !os.IsNotExist(err) {
		t.Fatalf("invalid operation accessed or created state: %v", err)
	}
}

func TestSpecCommandRejectsStaleRevisionWithoutMutation(t *testing.T) {
	root := newSpecificationTestRepository(t, colony.ColonyState{})
	_, err := createSpecificationDraft(root, specificationTestDraftRequest(t, colony.SpecScopeWholeGoal), specificationMutationOptions{})
	if err != nil {
		t.Fatalf("create draft: %v", err)
	}
	before := mustReadSpecificationTestStateBytes(t, root)
	failure := runSpecCommandFailure(t, root,
		"--modify",
		"--section", string(specificationSectionOutcomes),
		"--item-id", "outcome-missing",
		"--text", "Changed outcome",
		"--evidence", "owner:answer",
		"--predecessor-revision", "spec-revision-stale",
	)
	if !strings.Contains(failure, "stale predecessor") {
		t.Fatalf("stale predecessor failure = %q", failure)
	}
	if !bytes.Equal(before, mustReadSpecificationTestStateBytes(t, root)) {
		t.Fatal("stale predecessor changed canonical state")
	}
}

func TestSpecCommandHelpDocumentsExactApprovalInputs(t *testing.T) {
	command := newSpecCommand()
	var output bytes.Buffer
	command.SetOut(&output)
	command.SetErr(&output)
	command.SetArgs([]string{"--help"})
	if err := command.Execute(); err != nil {
		t.Fatalf("spec help: %v", err)
	}
	for _, flag := range []string{"--approve", "--revision-id", "--revision-hash", "--approval-token"} {
		if !strings.Contains(output.String(), flag) {
			t.Errorf("spec help omitted exact approval input %s:\n%s", flag, output.String())
		}
	}
	for _, forbidden := range []string{"generic yes", "accept plan", "activate plan"} {
		if strings.Contains(strings.ToLower(output.String()), forbidden) {
			t.Errorf("spec help grants false authority with %q", forbidden)
		}
	}
}

func runSpecCommandSuccess(t *testing.T, root string, args ...string) (specCommandResult, string) {
	t.Helper()
	output, executeErr := runSpecCommandRaw(t, root, args...)
	if executeErr != nil {
		t.Fatalf("spec command %v: %v", args, executeErr)
	}
	var envelope struct {
		OK     bool              `json:"ok"`
		Result specCommandResult `json:"result"`
		Error  string            `json:"error"`
	}
	if err := json.Unmarshal([]byte(output), &envelope); err != nil {
		t.Fatalf("decode spec output %q: %v", output, err)
	}
	if !envelope.OK {
		t.Fatalf("spec command %v failed: %s", args, envelope.Error)
	}
	return envelope.Result, output
}

func runSpecCommandFailure(t *testing.T, root string, args ...string) string {
	t.Helper()
	output, executeErr := runSpecCommandRaw(t, root, args...)
	if executeErr != nil {
		return executeErr.Error()
	}
	var envelope struct {
		OK    bool   `json:"ok"`
		Error string `json:"error"`
	}
	if err := json.Unmarshal([]byte(output), &envelope); err != nil {
		t.Fatalf("decode spec failure %q: %v", output, err)
	}
	if envelope.OK || envelope.Error == "" {
		t.Fatalf("spec command %v unexpectedly succeeded: %s", args, output)
	}
	return envelope.Error
}

func runSpecCommandRaw(t *testing.T, root string, args ...string) (string, error) {
	t.Helper()
	saveGlobals(t)
	t.Setenv("AETHER_ROOT", root)
	t.Setenv("COLONY_DATA_DIR", "")
	t.Setenv("AETHER_OUTPUT_MODE", "json")
	var output bytes.Buffer
	stdout = &output
	stderr = &output
	renderedCommandExitCode.Store(0)
	t.Cleanup(func() { renderedCommandExitCode.Store(0) })
	command := newSpecCommand()
	command.SetOut(&output)
	command.SetErr(&output)
	command.SetArgs(args)
	err := command.Execute()
	return strings.TrimSpace(output.String()), err
}

func assertSpecCommandBodyMatchesRevision(t *testing.T, result specCommandResult, revision colony.SpecRevision) {
	t.Helper()
	if !reflect.DeepEqual(result.Outcomes, revision.Outcomes) ||
		!reflect.DeepEqual(result.IncludedBehaviors, revision.IncludedBehaviors) ||
		!reflect.DeepEqual(result.Exclusions, revision.Exclusions) ||
		!reflect.DeepEqual(result.BindingDecisions, revision.BindingDecisions) ||
		!reflect.DeepEqual(result.Requirements, revision.Requirements) ||
		!reflect.DeepEqual(result.AcceptanceChecks, revision.AcceptanceChecks) ||
		!reflect.DeepEqual(result.NegativeExpectations, revision.NegativeExpectations) ||
		!reflect.DeepEqual(result.RecoveryExpectations, revision.RecoveryExpectations) ||
		!reflect.DeepEqual(result.AffectedPublicPaths, revision.AffectedPublicPaths) {
		t.Fatalf("typed command body does not round-trip revision\nresult: %#v\nrevision: %#v", result, revision)
	}
}
