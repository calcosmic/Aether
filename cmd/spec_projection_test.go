package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

func TestSpecProjectionRendersTypedBodyInCanonicalOrder(t *testing.T) {
	wantHeadings := []string{
		"## Outcome",
		"## Included Behavior",
		"## Explicit Exclusions",
		"## Binding Decisions",
		"## Stable Requirements",
		"## Owner-Checkable Acceptance",
		"## Negative Expectations",
		"## Recovery Expectations",
		"## Affected Public Paths",
		"## Revision and Status",
		"## Scope",
		"## Evidence Summary",
		"## Classified Delta",
		"## Affected Downstream Scope",
		"## Exact Next Command",
	}
	for _, scopeKind := range []colony.SpecScopeKind{colony.SpecScopeWholeGoal, colony.SpecScopeFeature} {
		t.Run(string(scopeKind), func(t *testing.T) {
			specification, revision, err := buildSpecificationDraft(specificationTestDraftRequest(t, scopeKind))
			if err != nil {
				t.Fatalf("build draft: %v", err)
			}
			content, err := renderSpecificationProjection(specification, specificationTestPlanState().Plan)
			if err != nil {
				t.Fatalf("render projection: %v", err)
			}
			projection := string(content)
			previous := -1
			for _, heading := range wantHeadings {
				index := strings.Index(projection, heading)
				if index < 0 {
					t.Fatalf("projection omitted %q:\n%s", heading, projection)
				}
				if index <= previous {
					t.Fatalf("projection heading %q is out of order:\n%s", heading, projection)
				}
				previous = index
			}
			for _, id := range specificationProjectionTestBodyIDs(revision) {
				if !strings.Contains(projection, "`"+id+"`") {
					t.Errorf("projection omitted stable ID %q", id)
				}
			}
			if strings.Contains(strings.ToLower(projection), "## items") {
				t.Fatalf("projection collapsed typed sections into a generic items section:\n%s", projection)
			}
			if !strings.Contains(projection, "Status: `DRAFT`") || !strings.Contains(projection, "This draft does not authorize planning") {
				t.Fatalf("draft projection omitted non-authority warning:\n%s", projection)
			}
			if !strings.Contains(projection, "`aether spec --approve --revision-id "+revision.ID) || !strings.Contains(projection, "--revision-hash "+revision.ContentHash) {
				t.Fatalf("draft projection omitted exact next command:\n%s", projection)
			}
			if scopeKind == colony.SpecScopeWholeGoal {
				if !strings.Contains(projection, "Kind: `whole_goal`") {
					t.Fatalf("whole-goal scope missing:\n%s", projection)
				}
			} else {
				if !strings.Contains(projection, "Feature: `feature-visible-planning`") || !strings.Contains(projection, "Requirement IDs: `"+revision.Scope.RequirementIDs[0]+"`") {
					t.Fatalf("feature scope missing exact boundary:\n%s", projection)
				}
			}
		})
	}
}

func TestSpecProjectionRendersExplicitNoneForEmptyLists(t *testing.T) {
	var builder strings.Builder
	renderSpecificationProjectionList(&builder, nil)
	if got := builder.String(); got != "- none\n" {
		t.Fatalf("empty projection list = %q, want explicit none", got)
	}
}

func TestSpecProjectionDetectsAndRepairsTamperOrDeletionFromCanonicalState(t *testing.T) {
	root := newSpecificationTestRepository(t, specificationTestPlanState())
	draft, err := createSpecificationDraft(root, specificationTestDraftRequest(t, colony.SpecScopeFeature), specificationMutationOptions{})
	if err != nil {
		t.Fatalf("create draft: %v", err)
	}
	canonicalState := mustReadSpecificationTestStateBytes(t, root)
	projectionPath := filepath.Join(root, specificationProjectionRelativePath)

	if err := os.WriteFile(projectionPath, []byte("# forged\nStatus: APPROVED\n"), 0o644); err != nil {
		t.Fatalf("tamper projection: %v", err)
	}
	inspection, err := inspectSpecificationProjection(root)
	if err != nil {
		t.Fatalf("inspect tampered projection: %v", err)
	}
	if !inspection.Drifted || inspection.Missing {
		t.Fatalf("tamper inspection = %#v, want drifted existing file", inspection)
	}
	if state := mustReadSpecificationTestState(t, root); state.Specification == nil || state.Specification.Revisions[len(state.Specification.Revisions)-1].Status != colony.SpecStatusDraft {
		t.Fatalf("forged Markdown changed canonical authority: %#v", state.Specification)
	}
	repair, err := repairSpecificationProjection(root, specificationMutationOptions{})
	if err != nil {
		t.Fatalf("repair tampered projection: %v", err)
	}
	if !repair.Repaired || repair.Receipt.ReceiptID == "" {
		t.Fatalf("tamper repair did not produce a durable write receipt: %#v", repair)
	}
	if !bytes.Equal(canonicalState, mustReadSpecificationTestStateBytes(t, root)) {
		t.Fatal("projection repair changed revision or approval state")
	}
	if projection := mustReadSpecificationTestProjection(t, root); !strings.Contains(projection, draft.Revision.ID) || strings.Contains(projection, "# forged") {
		t.Fatalf("repair did not regenerate canonical projection:\n%s", projection)
	}
	clean, err := inspectSpecificationProjection(root)
	if err != nil || clean.Drifted || clean.Missing {
		t.Fatalf("repaired projection inspection = %#v, err=%v", clean, err)
	}

	if err := os.Remove(projectionPath); err != nil {
		t.Fatalf("delete projection: %v", err)
	}
	missing, err := inspectSpecificationProjection(root)
	if err != nil {
		t.Fatalf("inspect missing projection: %v", err)
	}
	if !missing.Drifted || !missing.Missing {
		t.Fatalf("missing inspection = %#v, want drifted and missing", missing)
	}
	secondRepair, err := repairSpecificationProjection(root, specificationMutationOptions{})
	if err != nil {
		t.Fatalf("repair deleted projection: %v", err)
	}
	if !secondRepair.Repaired || !bytes.Equal(canonicalState, mustReadSpecificationTestStateBytes(t, root)) {
		t.Fatalf("deleted projection repair = %#v or changed canonical state", secondRepair)
	}
	noChange, err := repairSpecificationProjection(root, specificationMutationOptions{})
	if err != nil {
		t.Fatalf("repair synchronized projection: %v", err)
	}
	if noChange.Repaired || noChange.Receipt.ReceiptID != "" {
		t.Fatalf("synchronized repair should be a read-only no-op: %#v", noChange)
	}
}

func specificationProjectionTestBodyIDs(revision colony.SpecRevision) []string {
	result := []string{}
	for _, item := range revision.Outcomes {
		result = append(result, item.ID)
	}
	for _, item := range revision.IncludedBehaviors {
		result = append(result, item.ID)
	}
	for _, item := range revision.Exclusions {
		result = append(result, item.ID)
	}
	for _, item := range revision.BindingDecisions {
		result = append(result, item.ID)
	}
	for _, item := range revision.Requirements {
		result = append(result, item.ID)
	}
	for _, item := range revision.AcceptanceChecks {
		result = append(result, item.ID)
	}
	for _, item := range revision.NegativeExpectations {
		result = append(result, item.ID)
	}
	for _, item := range revision.RecoveryExpectations {
		result = append(result, item.ID)
	}
	for _, item := range revision.AffectedPublicPaths {
		result = append(result, item.ID)
	}
	return result
}

func mustReadSpecificationTestProjection(t *testing.T, root string) string {
	t.Helper()
	return string(mustReadSpecificationTestProjectionBytes(t, root))
}

func mustReadSpecificationTestProjectionBytes(t *testing.T, root string) []byte {
	t.Helper()
	content, err := os.ReadFile(filepath.Join(root, specificationProjectionRelativePath))
	if err != nil {
		t.Fatalf("read specification projection: %v", err)
	}
	return content
}
