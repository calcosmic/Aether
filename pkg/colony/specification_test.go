package colony

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestSpecRevisionStatusIsClosed(t *testing.T) {
	t.Parallel()

	valid := []SpecRevisionStatus{
		SpecStatusDraft,
		SpecStatusApproved,
		SpecStatusSuperseded,
	}
	for _, status := range valid {
		status := status
		t.Run(string(status), func(t *testing.T) {
			t.Parallel()
			if !status.Valid() {
				t.Fatalf("status %q is not valid", status)
			}
			encoded, err := json.Marshal(status)
			if err != nil {
				t.Fatalf("marshal status: %v", err)
			}
			var decoded SpecRevisionStatus
			if err := json.Unmarshal(encoded, &decoded); err != nil {
				t.Fatalf("unmarshal status: %v", err)
			}
			if decoded != status {
				t.Fatalf("status round trip = %q, want %q", decoded, status)
			}
		})
	}

	if SpecRevisionStatus("accepted").Valid() {
		t.Fatal("plan-acceptance vocabulary was accepted as a specification status")
	}
	if _, err := json.Marshal(SpecRevisionStatus("future_status")); err == nil {
		t.Fatal("unknown specification status marshaled successfully")
	}
	var decoded SpecRevisionStatus
	if err := json.Unmarshal([]byte(`"future_status"`), &decoded); err == nil {
		t.Fatal("unknown specification status unmarshaled successfully")
	}
}

func TestSpecWholeGoalRevisionRoundTripPreservesTypedBody(t *testing.T) {
	t.Parallel()

	revision := validSpecRevision(SpecScope{
		Kind:      SpecScopeWholeGoal,
		GoalID:    "goal-200",
		SessionID: "session-200",
	})
	revision.Status = SpecStatusApproved
	revision.Approval = validSpecApproval(revision)

	specification := Specification{
		SchemaVersion:     SpecificationSchemaVersion,
		ID:                revision.SpecificationID,
		GoalID:            "goal-200",
		CurrentRevisionID: revision.ID,
		Revisions:         []SpecRevision{revision},
	}
	if err := specification.Validate(); err != nil {
		t.Fatalf("valid whole-goal specification rejected: %v", err)
	}
	if err := specification.ValidateCanonical(); err != nil {
		t.Fatalf("canonical whole-goal specification rejected: %v", err)
	}

	encoded, err := json.Marshal(specification)
	if err != nil {
		t.Fatalf("marshal specification: %v", err)
	}
	var decoded Specification
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("unmarshal specification: %v", err)
	}
	if err := decoded.Validate(); err != nil {
		t.Fatalf("round-tripped specification rejected: %v", err)
	}
	if !reflect.DeepEqual(decoded, specification) {
		t.Fatalf("specification round trip changed data:\n got: %#v\nwant: %#v", decoded, specification)
	}

	got := decoded.Revisions[0]
	sections := map[string]int{
		"outcomes":              len(got.Outcomes),
		"included_behaviors":    len(got.IncludedBehaviors),
		"exclusions":            len(got.Exclusions),
		"binding_decisions":     len(got.BindingDecisions),
		"requirements":          len(got.Requirements),
		"acceptance_checks":     len(got.AcceptanceChecks),
		"negative_expectations": len(got.NegativeExpectations),
		"recovery_expectations": len(got.RecoveryExpectations),
		"affected_public_paths": len(got.AffectedPublicPaths),
	}
	for section, count := range sections {
		if count != 1 {
			t.Errorf("%s contains %d entries, want 1", section, count)
		}
	}
	if got.Outcomes[0].ID == "" || got.AcceptanceChecks[0].ID == "" || got.AffectedPublicPaths[0].ID == "" {
		t.Fatal("typed body entries lost their stable IDs")
	}
	if strings.Contains(string(encoded), `"items"`) {
		t.Fatalf("specification collapsed typed body sections into a generic items bag: %s", encoded)
	}
}

func TestSpecFeatureSuccessorRoundTripClassifiesEveryBodySection(t *testing.T) {
	t.Parallel()

	predecessor := validSpecRevision(SpecScope{
		Kind:      SpecScopeWholeGoal,
		GoalID:    "goal-200",
		SessionID: "session-200",
	})
	predecessor.Status = SpecStatusSuperseded
	predecessor.Approval = validSpecApproval(predecessor)

	successor := validSpecRevision(SpecScope{
		Kind:               SpecScopeFeature,
		GoalID:             "goal-200",
		SessionID:          "session-200",
		FeatureID:          "feature-safe-revision",
		RequirementIDs:     []string{predecessor.Requirements[0].ID},
		AcceptanceCheckIDs: []string{predecessor.AcceptanceChecks[0].ID},
	})
	successor.PredecessorID = predecessor.ID
	successor.CreatedAt = predecessor.CreatedAt.Add(time.Minute)
	reviseValidSpecRevisionItems(&successor)
	successor.Delta = CanonicalSpecRevisionDelta(&predecessor, successor)
	mustAddressSpecRevision(&successor)

	specification := Specification{
		SchemaVersion:     SpecificationSchemaVersion,
		ID:                predecessor.SpecificationID,
		GoalID:            "goal-200",
		CurrentRevisionID: successor.ID,
		Revisions:         []SpecRevision{predecessor, successor},
	}
	if err := specification.Validate(); err != nil {
		t.Fatalf("valid feature-scoped successor rejected: %v", err)
	}
	if err := specification.ValidateCanonical(); err != nil {
		t.Fatalf("canonical feature-scoped successor rejected: %v", err)
	}

	encoded, err := json.Marshal(specification)
	if err != nil {
		t.Fatalf("marshal scoped successor: %v", err)
	}
	var decoded Specification
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("unmarshal scoped successor: %v", err)
	}
	if !reflect.DeepEqual(decoded, specification) {
		t.Fatalf("scoped successor round trip changed data:\n got: %#v\nwant: %#v", decoded, specification)
	}

	deltas := []SpecItemDelta{
		successor.Delta.Outcomes,
		successor.Delta.IncludedBehaviors,
		successor.Delta.Exclusions,
		successor.Delta.BindingDecisions,
		successor.Delta.Requirements,
		successor.Delta.AcceptanceChecks,
		successor.Delta.NegativeExpectations,
		successor.Delta.RecoveryExpectations,
		successor.Delta.AffectedPublicPaths,
	}
	for i, delta := range deltas {
		if len(delta.ModifiedIDs) != 1 || len(delta.AddedIDs) != 0 || len(delta.RemovedIDs) != 0 || len(delta.UnchangedIDs) != 0 {
			t.Errorf("delta section %d did not classify its canonical modification: %#v", i, delta)
		}
	}
}

func TestSpecApprovalReceiptBindsExactRevisionAndHash(t *testing.T) {
	t.Parallel()

	revision := validSpecRevision(SpecScope{
		Kind:      SpecScopeWholeGoal,
		GoalID:    "goal-200",
		SessionID: "session-200",
	})
	revision.Status = SpecStatusApproved
	revision.Approval = validSpecApproval(revision)
	if err := revision.Validate(); err != nil {
		t.Fatalf("valid approved revision rejected: %v", err)
	}

	withoutReceipt := revision
	withoutReceipt.Approval = nil
	assertSpecValidationError(t, withoutReceipt.Validate(), "approval")

	wrongRevision := revision
	approval := *revision.Approval
	approval.RevisionID = "spec-rev-other"
	wrongRevision.Approval = &approval
	assertSpecValidationError(t, wrongRevision.Validate(), "revision_id")

	wrongHash := revision
	approval = *revision.Approval
	approval.RevisionContentHash = "different-content-hash"
	wrongHash.Approval = &approval
	assertSpecValidationError(t, wrongHash.Validate(), "revision_content_hash")

	draftWithApproval := revision
	draftWithApproval.Status = SpecStatusDraft
	assertSpecValidationError(t, draftWithApproval.Validate(), "draft")
}

func TestSpecLegacyColonyStateDecodesWithoutAuthority(t *testing.T) {
	t.Parallel()

	const legacy = `{
		"version":"3.0",
		"goal":"legacy goal",
		"state":"READY",
		"current_phase":1,
		"plan":{"generated_at":null,"confidence":null,"phases":[]},
		"memory":{"phase_learnings":[],"decisions":[],"instincts":[]},
		"errors":{"records":[],"flagged_patterns":[]},
		"signals":[],"graveyards":[],"events":[]
	}`
	var state ColonyState
	if err := json.Unmarshal([]byte(legacy), &state); err != nil {
		t.Fatalf("unmarshal legacy colony state: %v", err)
	}
	if state.Specification != nil {
		t.Fatalf("legacy state invented specification authority: %#v", state.Specification)
	}
	encoded, err := json.Marshal(state)
	if err != nil {
		t.Fatalf("re-marshal legacy state: %v", err)
	}
	if strings.Contains(string(encoded), `"specification"`) {
		t.Fatalf("legacy round trip fabricated specification field: %s", encoded)
	}
}

func TestSpecContractsDoNotExposePlanAcceptanceAuthority(t *testing.T) {
	t.Parallel()

	for _, value := range []any{Specification{}, SpecRevision{}, SpecApprovalReceipt{}} {
		typeOf := reflect.TypeOf(value)
		for i := 0; i < typeOf.NumField(); i++ {
			field := typeOf.Field(i)
			name := strings.ToLower(field.Name + " " + field.Tag.Get("json"))
			if strings.Contains(name, "candidate") || strings.Contains(name, "plan_accept") || field.Name == "Accepted" {
				t.Fatalf("%s exposes plan-candidate acceptance authority through field %s", typeOf.Name(), field.Name)
			}
		}
	}
}

func validSpecRevision(scope SpecScope) SpecRevision {
	specificationID, err := CanonicalSpecificationID(scope.GoalID)
	if err != nil {
		panic(err)
	}
	outcome := mustCanonicalSpecItem(SpecSectionOutcomes, "owner-understands-plan", "The owner understands why the route changed", "", "", []string{"evidence-context"})
	behavior := mustCanonicalSpecItem(SpecSectionIncludedBehaviors, "visible-loop", "Show each grounded planning pass", "", "", []string{"evidence-classic"})
	exclusion := mustCanonicalSpecItem(SpecSectionExclusions, "build-cycle", "Do not change build execution", "", "", []string{"evidence-scope"})
	decision := mustCanonicalSpecItem(SpecSectionBindingDecisions, "explicit-acceptance", "Plan acceptance remains explicit", "", "", []string{"evidence-decision"})
	requirement := mustCanonicalSpecItem(SpecSectionRequirements, "safe-revision", "Preserve unaffected work", "", "", []string{"evidence-requirement"})
	check := mustCanonicalSpecItem(SpecSectionAcceptanceChecks, "safe-revision", "A scoped revision retains unaffected stable IDs", "Run the scoped revision fixture", "", []string{"evidence-acceptance"})
	negative := mustCanonicalSpecItem(SpecSectionNegativeExpectations, "no-auto-accept", "Specification approval cannot accept a plan candidate", "", "", []string{"evidence-negative"})
	recovery := mustCanonicalSpecItem(SpecSectionRecoveryExpectations, "exact-replay", "Exact replay returns the same revision", "", "", []string{"evidence-recovery"})
	publicPath := mustCanonicalSpecItem(SpecSectionAffectedPublicPaths, "ant-spec", "Review and approve the specification", "", "/ant-spec", []string{"evidence-path"})
	revision := SpecRevision{
		SchemaVersion:   SpecificationSchemaVersion,
		SpecificationID: specificationID,
		CreatedAt:       time.Date(2026, time.September, 7, 10, 0, 0, 0, time.UTC),
		Scope:           scope,
		Status:          SpecStatusDraft,
		Outcomes: []SpecOutcome{{
			ID: outcome.ID, Description: outcome.Description, ContentHash: outcome.ContentHash, EvidenceIDs: outcome.EvidenceIDs,
		}},
		IncludedBehaviors: []SpecIncludedBehavior{{
			ID: behavior.ID, Description: behavior.Description, ContentHash: behavior.ContentHash, EvidenceIDs: behavior.EvidenceIDs,
		}},
		Exclusions: []SpecExclusion{{
			ID: exclusion.ID, Description: exclusion.Description, ContentHash: exclusion.ContentHash, EvidenceIDs: exclusion.EvidenceIDs,
		}},
		BindingDecisions: []SpecBindingDecision{{
			ID: decision.ID, Description: decision.Description, ContentHash: decision.ContentHash, EvidenceIDs: decision.EvidenceIDs,
		}},
		Requirements: []SpecRequirement{{
			ID: requirement.ID, Description: requirement.Description, ContentHash: requirement.ContentHash, EvidenceIDs: requirement.EvidenceIDs,
		}},
		AcceptanceChecks: []SpecAcceptanceCheck{{
			ID: check.ID, Description: check.Description, Verification: check.Verification, ContentHash: check.ContentHash, EvidenceIDs: check.EvidenceIDs,
		}},
		NegativeExpectations: []SpecNegativeExpectation{{
			ID: negative.ID, Description: negative.Description, ContentHash: negative.ContentHash, EvidenceIDs: negative.EvidenceIDs,
		}},
		RecoveryExpectations: []SpecRecoveryExpectation{{
			ID: recovery.ID, Description: recovery.Description, ContentHash: recovery.ContentHash, EvidenceIDs: recovery.EvidenceIDs,
		}},
		AffectedPublicPaths: []SpecPublicPath{{
			ID: publicPath.ID, Path: publicPath.Path, Description: publicPath.Description, ContentHash: publicPath.ContentHash, EvidenceIDs: publicPath.EvidenceIDs,
		}},
	}
	revision.Scope, err = CanonicalSpecScope(revision.Scope)
	if err != nil {
		panic(err)
	}
	revision.Delta = CanonicalSpecRevisionDelta(nil, revision)
	mustAddressSpecRevision(&revision)
	return revision
}

func validSpecApproval(revision SpecRevision) *SpecApprovalReceipt {
	token := CanonicalSpecificationApprovalToken(revision.SpecificationID, revision.ID, revision.ContentHash)
	receipt := &SpecApprovalReceipt{
		SchemaVersion:       SpecificationSchemaVersion,
		SpecificationID:     revision.SpecificationID,
		RevisionID:          revision.ID,
		RevisionContentHash: revision.ContentHash,
		ApprovalTokenHash:   CanonicalSpecificationApprovalTokenHash(token),
		ApprovedBy:          "owner-callum",
		ApprovedAt:          revision.CreatedAt.Add(30 * time.Second),
	}
	id, err := CanonicalSpecApprovalReceiptID(*receipt)
	if err != nil {
		panic(err)
	}
	receipt.ID = id
	return receipt
}

func mustCanonicalSpecItem(section SpecSection, lineage, description, verification, publicPath string, evidenceIDs []string) SpecCanonicalItemMaterial {
	id, err := CanonicalSpecItemID(section, lineage)
	if err != nil {
		panic(err)
	}
	item, err := CanonicalizeSpecItem(section, id, description, verification, publicPath, evidenceIDs)
	if err != nil {
		panic(err)
	}
	return item
}

func mustAddressSpecRevision(revision *SpecRevision) {
	if err := AddressSpecRevision(revision); err != nil {
		panic(err)
	}
}

func reviseValidSpecRevisionItems(revision *SpecRevision) {
	revise := func(section SpecSection, id, description, verification, publicPath string, evidenceIDs []string) SpecCanonicalItemMaterial {
		item, err := CanonicalizeSpecItem(section, id, description+" revised", verification, publicPath, evidenceIDs)
		if err != nil {
			panic(err)
		}
		return item
	}
	outcome := revise(SpecSectionOutcomes, revision.Outcomes[0].ID, revision.Outcomes[0].Description, "", "", revision.Outcomes[0].EvidenceIDs)
	revision.Outcomes[0].Description, revision.Outcomes[0].ContentHash = outcome.Description, outcome.ContentHash
	behavior := revise(SpecSectionIncludedBehaviors, revision.IncludedBehaviors[0].ID, revision.IncludedBehaviors[0].Description, "", "", revision.IncludedBehaviors[0].EvidenceIDs)
	revision.IncludedBehaviors[0].Description, revision.IncludedBehaviors[0].ContentHash = behavior.Description, behavior.ContentHash
	exclusion := revise(SpecSectionExclusions, revision.Exclusions[0].ID, revision.Exclusions[0].Description, "", "", revision.Exclusions[0].EvidenceIDs)
	revision.Exclusions[0].Description, revision.Exclusions[0].ContentHash = exclusion.Description, exclusion.ContentHash
	decision := revise(SpecSectionBindingDecisions, revision.BindingDecisions[0].ID, revision.BindingDecisions[0].Description, "", "", revision.BindingDecisions[0].EvidenceIDs)
	revision.BindingDecisions[0].Description, revision.BindingDecisions[0].ContentHash = decision.Description, decision.ContentHash
	requirement := revise(SpecSectionRequirements, revision.Requirements[0].ID, revision.Requirements[0].Description, "", "", revision.Requirements[0].EvidenceIDs)
	revision.Requirements[0].Description, revision.Requirements[0].ContentHash = requirement.Description, requirement.ContentHash
	check := revise(SpecSectionAcceptanceChecks, revision.AcceptanceChecks[0].ID, revision.AcceptanceChecks[0].Description, revision.AcceptanceChecks[0].Verification, "", revision.AcceptanceChecks[0].EvidenceIDs)
	revision.AcceptanceChecks[0].Description, revision.AcceptanceChecks[0].ContentHash = check.Description, check.ContentHash
	negative := revise(SpecSectionNegativeExpectations, revision.NegativeExpectations[0].ID, revision.NegativeExpectations[0].Description, "", "", revision.NegativeExpectations[0].EvidenceIDs)
	revision.NegativeExpectations[0].Description, revision.NegativeExpectations[0].ContentHash = negative.Description, negative.ContentHash
	recovery := revise(SpecSectionRecoveryExpectations, revision.RecoveryExpectations[0].ID, revision.RecoveryExpectations[0].Description, "", "", revision.RecoveryExpectations[0].EvidenceIDs)
	revision.RecoveryExpectations[0].Description, revision.RecoveryExpectations[0].ContentHash = recovery.Description, recovery.ContentHash
	publicPath := revise(SpecSectionAffectedPublicPaths, revision.AffectedPublicPaths[0].ID, revision.AffectedPublicPaths[0].Description, "", revision.AffectedPublicPaths[0].Path, revision.AffectedPublicPaths[0].EvidenceIDs)
	revision.AffectedPublicPaths[0].Description, revision.AffectedPublicPaths[0].ContentHash = publicPath.Description, publicPath.ContentHash
}

func assertSpecValidationError(t *testing.T, err error, field string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected validation error containing %q", field)
	}
	if !strings.Contains(err.Error(), field) {
		t.Fatalf("validation error %q does not contain %q", err, field)
	}
}
