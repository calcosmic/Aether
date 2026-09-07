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
		ID:                "spec-goal-200",
		GoalID:            "goal-200",
		CurrentRevisionID: revision.ID,
		Revisions:         []SpecRevision{revision},
	}
	if err := specification.Validate(); err != nil {
		t.Fatalf("valid whole-goal specification rejected: %v", err)
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
		RequirementIDs:     []string{"req-safe-revision"},
		AcceptanceCheckIDs: []string{"check-safe-revision"},
	})
	successor.ID = "spec-rev-2"
	successor.PredecessorID = predecessor.ID
	successor.ContentHash = "spec-rev-2-content-hash"
	successor.CreatedAt = predecessor.CreatedAt.Add(time.Minute)
	successor.Delta = fullSpecRevisionDelta(predecessor.ID)

	specification := Specification{
		SchemaVersion:     SpecificationSchemaVersion,
		ID:                "spec-goal-200",
		GoalID:            "goal-200",
		CurrentRevisionID: successor.ID,
		Revisions:         []SpecRevision{predecessor, successor},
	}
	if err := specification.Validate(); err != nil {
		t.Fatalf("valid feature-scoped successor rejected: %v", err)
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
		if len(delta.AddedIDs) != 1 || len(delta.ModifiedIDs) != 1 || len(delta.RemovedIDs) != 1 || len(delta.UnchangedIDs) != 1 {
			t.Errorf("delta section %d did not preserve explicit add/modify/remove/preserve classifications: %#v", i, delta)
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
	revision := SpecRevision{
		SchemaVersion:   SpecificationSchemaVersion,
		ID:              "spec-rev-1",
		SpecificationID: "spec-goal-200",
		CreatedAt:       time.Date(2026, time.September, 7, 10, 0, 0, 0, time.UTC),
		ContentHash:     "spec-rev-1-content-hash",
		Scope:           scope,
		Status:          SpecStatusDraft,
		Outcomes: []SpecOutcome{{
			ID: "outcome-owner-understands-plan", Description: "The owner understands why the route changed", ContentHash: "hash-outcome", EvidenceIDs: []string{"evidence-context"},
		}},
		IncludedBehaviors: []SpecIncludedBehavior{{
			ID: "behavior-visible-loop", Description: "Show each grounded planning pass", ContentHash: "hash-behavior", EvidenceIDs: []string{"evidence-classic"},
		}},
		Exclusions: []SpecExclusion{{
			ID: "exclusion-build-cycle", Description: "Do not change build execution", ContentHash: "hash-exclusion", EvidenceIDs: []string{"evidence-scope"},
		}},
		BindingDecisions: []SpecBindingDecision{{
			ID: "decision-explicit-acceptance", Description: "Plan acceptance remains explicit", ContentHash: "hash-decision", EvidenceIDs: []string{"evidence-decision"},
		}},
		Requirements: []SpecRequirement{{
			ID: "req-safe-revision", Description: "Preserve unaffected work", ContentHash: "hash-requirement", EvidenceIDs: []string{"evidence-requirement"},
		}},
		AcceptanceChecks: []SpecAcceptanceCheck{{
			ID: "check-safe-revision", Description: "A scoped revision retains unaffected stable IDs", Verification: "Run the scoped revision fixture", ContentHash: "hash-check", EvidenceIDs: []string{"evidence-acceptance"},
		}},
		NegativeExpectations: []SpecNegativeExpectation{{
			ID: "negative-no-auto-accept", Description: "Specification approval cannot accept a plan candidate", ContentHash: "hash-negative", EvidenceIDs: []string{"evidence-negative"},
		}},
		RecoveryExpectations: []SpecRecoveryExpectation{{
			ID: "recovery-exact-replay", Description: "Exact replay returns the same revision", ContentHash: "hash-recovery", EvidenceIDs: []string{"evidence-recovery"},
		}},
		AffectedPublicPaths: []SpecPublicPath{{
			ID: "path-ant-spec", Path: "/ant-spec", Description: "Review and approve the specification", ContentHash: "hash-path", EvidenceIDs: []string{"evidence-path"},
		}},
	}
	revision.Delta = SpecRevisionDelta{
		PredecessorRevisionID: revision.PredecessorID,
		Outcomes:              SpecItemDelta{AddedIDs: []string{revision.Outcomes[0].ID}},
		IncludedBehaviors:     SpecItemDelta{AddedIDs: []string{revision.IncludedBehaviors[0].ID}},
		Exclusions:            SpecItemDelta{AddedIDs: []string{revision.Exclusions[0].ID}},
		BindingDecisions:      SpecItemDelta{AddedIDs: []string{revision.BindingDecisions[0].ID}},
		Requirements:          SpecItemDelta{AddedIDs: []string{revision.Requirements[0].ID}},
		AcceptanceChecks:      SpecItemDelta{AddedIDs: []string{revision.AcceptanceChecks[0].ID}},
		NegativeExpectations:  SpecItemDelta{AddedIDs: []string{revision.NegativeExpectations[0].ID}},
		RecoveryExpectations:  SpecItemDelta{AddedIDs: []string{revision.RecoveryExpectations[0].ID}},
		AffectedPublicPaths:   SpecItemDelta{AddedIDs: []string{revision.AffectedPublicPaths[0].ID}},
	}
	return revision
}

func validSpecApproval(revision SpecRevision) *SpecApprovalReceipt {
	return &SpecApprovalReceipt{
		SchemaVersion:       SpecificationSchemaVersion,
		ID:                  "spec-approval-1",
		SpecificationID:     revision.SpecificationID,
		RevisionID:          revision.ID,
		RevisionContentHash: revision.ContentHash,
		ApprovalTokenHash:   "approval-token-hash",
		ApprovedBy:          "owner-callum",
		ApprovedAt:          revision.CreatedAt.Add(30 * time.Second),
	}
}

func fullSpecRevisionDelta(predecessorID string) SpecRevisionDelta {
	delta := SpecItemDelta{
		AddedIDs:     []string{"added-id"},
		ModifiedIDs:  []string{"modified-id"},
		RemovedIDs:   []string{"removed-id"},
		UnchangedIDs: []string{"unchanged-id"},
	}
	return SpecRevisionDelta{
		PredecessorRevisionID: predecessorID,
		Outcomes:              delta,
		IncludedBehaviors:     delta,
		Exclusions:            delta,
		BindingDecisions:      delta,
		Requirements:          delta,
		AcceptanceChecks:      delta,
		NegativeExpectations:  delta,
		RecoveryExpectations:  delta,
		AffectedPublicPaths:   delta,
	}
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
