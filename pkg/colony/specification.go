package colony

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// SpecificationSchemaVersion is the wire contract shared by the canonical
// specification aggregate, immutable revisions, and approval receipts.
const SpecificationSchemaVersion = "specification/v1"

// SpecRevisionStatus is the closed authority state of one immutable
// specification revision. Plan-candidate acceptance deliberately does not
// appear in this vocabulary.
type SpecRevisionStatus string

const (
	SpecStatusDraft      SpecRevisionStatus = "draft"
	SpecStatusApproved   SpecRevisionStatus = "approved"
	SpecStatusSuperseded SpecRevisionStatus = "superseded"
)

// Valid reports whether s is a supported specification revision status.
func (s SpecRevisionStatus) Valid() bool {
	switch s {
	case SpecStatusDraft, SpecStatusApproved, SpecStatusSuperseded:
		return true
	default:
		return false
	}
}

// MarshalJSON refuses unknown status values so future authority states cannot
// silently pass through current writers.
func (s SpecRevisionStatus) MarshalJSON() ([]byte, error) {
	return marshalLifecycleEnum("specification revision status", string(s), s.Valid())
}

// UnmarshalJSON refuses unknown status values so readers fail closed.
func (s *SpecRevisionStatus) UnmarshalJSON(data []byte) error {
	raw, err := unmarshalLifecycleEnum(data, "specification revision status", func(raw string) bool {
		return SpecRevisionStatus(raw).Valid()
	})
	if err != nil {
		return err
	}
	*s = SpecRevisionStatus(raw)
	return nil
}

// SpecScopeKind distinguishes a complete goal contract from a scoped successor
// that changes one named feature while retaining the same specification lineage.
type SpecScopeKind string

const (
	SpecScopeWholeGoal SpecScopeKind = "whole_goal"
	SpecScopeFeature   SpecScopeKind = "feature"
)

// Valid reports whether k is a supported specification scope.
func (k SpecScopeKind) Valid() bool {
	switch k {
	case SpecScopeWholeGoal, SpecScopeFeature:
		return true
	default:
		return false
	}
}

func (k SpecScopeKind) MarshalJSON() ([]byte, error) {
	return marshalLifecycleEnum("specification scope kind", string(k), k.Valid())
}

func (k *SpecScopeKind) UnmarshalJSON(data []byte) error {
	raw, err := unmarshalLifecycleEnum(data, "specification scope kind", func(raw string) bool {
		return SpecScopeKind(raw).Valid()
	})
	if err != nil {
		return err
	}
	*k = SpecScopeKind(raw)
	return nil
}

// SpecScope binds a revision to the goal and session whose owner decisions it
// represents. Feature revisions additionally name the exact requirement and
// acceptance-check scope they are allowed to change.
type SpecScope struct {
	Kind               SpecScopeKind `json:"kind"`
	GoalID             string        `json:"goal_id"`
	SessionID          string        `json:"session_id"`
	FeatureID          string        `json:"feature_id"`
	RequirementIDs     []string      `json:"requirement_ids"`
	AcceptanceCheckIDs []string      `json:"acceptance_check_ids"`
}

// Validate checks the identity boundary of a whole-goal or feature scope.
func (s SpecScope) Validate() error {
	if !s.Kind.Valid() {
		return fmt.Errorf("kind: invalid specification scope %q", s.Kind)
	}
	if strings.TrimSpace(s.GoalID) == "" {
		return fmt.Errorf("goal_id is required")
	}
	if strings.TrimSpace(s.SessionID) == "" {
		return fmt.Errorf("session_id is required")
	}
	if err := validateSpecIDList("requirement_ids", s.RequirementIDs); err != nil {
		return err
	}
	if err := validateSpecIDList("acceptance_check_ids", s.AcceptanceCheckIDs); err != nil {
		return err
	}
	switch s.Kind {
	case SpecScopeWholeGoal:
		if strings.TrimSpace(s.FeatureID) != "" || len(s.RequirementIDs) != 0 || len(s.AcceptanceCheckIDs) != 0 {
			return fmt.Errorf("whole_goal scope cannot contain feature-only identifiers")
		}
	case SpecScopeFeature:
		if strings.TrimSpace(s.FeatureID) == "" {
			return fmt.Errorf("feature_id is required for feature scope")
		}
		if len(s.RequirementIDs) == 0 && len(s.AcceptanceCheckIDs) == 0 {
			return fmt.Errorf("feature scope requires requirement_ids or acceptance_check_ids")
		}
	}
	return nil
}

// The distinct body entry types keep the owner-readable specification from
// becoming an untyped items bag. Each entry has stable semantic identity and a
// content hash so successor revisions can preserve unchanged entries exactly.
type SpecOutcome struct {
	ID          string   `json:"id"`
	Description string   `json:"description"`
	ContentHash string   `json:"content_hash"`
	EvidenceIDs []string `json:"evidence_ids"`
}

type SpecIncludedBehavior struct {
	ID          string   `json:"id"`
	Description string   `json:"description"`
	ContentHash string   `json:"content_hash"`
	EvidenceIDs []string `json:"evidence_ids"`
}

type SpecExclusion struct {
	ID          string   `json:"id"`
	Description string   `json:"description"`
	ContentHash string   `json:"content_hash"`
	EvidenceIDs []string `json:"evidence_ids"`
}

type SpecBindingDecision struct {
	ID          string   `json:"id"`
	Description string   `json:"description"`
	ContentHash string   `json:"content_hash"`
	EvidenceIDs []string `json:"evidence_ids"`
}

type SpecRequirement struct {
	ID          string   `json:"id"`
	Description string   `json:"description"`
	ContentHash string   `json:"content_hash"`
	EvidenceIDs []string `json:"evidence_ids"`
}

type SpecAcceptanceCheck struct {
	ID           string   `json:"id"`
	Description  string   `json:"description"`
	Verification string   `json:"verification"`
	ContentHash  string   `json:"content_hash"`
	EvidenceIDs  []string `json:"evidence_ids"`
}

type SpecNegativeExpectation struct {
	ID          string   `json:"id"`
	Description string   `json:"description"`
	ContentHash string   `json:"content_hash"`
	EvidenceIDs []string `json:"evidence_ids"`
}

type SpecRecoveryExpectation struct {
	ID          string   `json:"id"`
	Description string   `json:"description"`
	ContentHash string   `json:"content_hash"`
	EvidenceIDs []string `json:"evidence_ids"`
}

type SpecPublicPath struct {
	ID          string   `json:"id"`
	Path        string   `json:"path"`
	Description string   `json:"description"`
	ContentHash string   `json:"content_hash"`
	EvidenceIDs []string `json:"evidence_ids"`
}

// SpecItemDelta explicitly classifies semantic identity across two immutable
// revisions. Unchanged IDs make unaffected scope auditable rather than inferred.
type SpecItemDelta struct {
	AddedIDs     []string `json:"added_ids"`
	ModifiedIDs  []string `json:"modified_ids"`
	RemovedIDs   []string `json:"removed_ids"`
	UnchangedIDs []string `json:"unchanged_ids"`
}

// Validate rejects ambiguous classification within one body category.
func (d SpecItemDelta) Validate() error {
	seen := make(map[string]string)
	classifications := []struct {
		name string
		ids  []string
	}{
		{name: "added_ids", ids: d.AddedIDs},
		{name: "modified_ids", ids: d.ModifiedIDs},
		{name: "removed_ids", ids: d.RemovedIDs},
		{name: "unchanged_ids", ids: d.UnchangedIDs},
	}
	for _, classification := range classifications {
		for i, id := range classification.ids {
			id = strings.TrimSpace(id)
			if id == "" {
				return fmt.Errorf("%s[%d] is required", classification.name, i)
			}
			if prior, ok := seen[id]; ok {
				return fmt.Errorf("stable ID %q appears in both %s and %s", id, prior, classification.name)
			}
			seen[id] = classification.name
		}
	}
	return nil
}

// SpecRevisionDelta keeps add/modify/remove/preserve classifications separate
// for every owner-readable body category.
type SpecRevisionDelta struct {
	PredecessorRevisionID string        `json:"predecessor_revision_id"`
	Outcomes              SpecItemDelta `json:"outcomes"`
	IncludedBehaviors     SpecItemDelta `json:"included_behaviors"`
	Exclusions            SpecItemDelta `json:"exclusions"`
	BindingDecisions      SpecItemDelta `json:"binding_decisions"`
	Requirements          SpecItemDelta `json:"requirements"`
	AcceptanceChecks      SpecItemDelta `json:"acceptance_checks"`
	NegativeExpectations  SpecItemDelta `json:"negative_expectations"`
	RecoveryExpectations  SpecItemDelta `json:"recovery_expectations"`
	AffectedPublicPaths   SpecItemDelta `json:"affected_public_paths"`
}

// Validate checks every explicit body-category classification.
func (d SpecRevisionDelta) Validate() error {
	sections := []struct {
		name  string
		delta SpecItemDelta
	}{
		{name: "outcomes", delta: d.Outcomes},
		{name: "included_behaviors", delta: d.IncludedBehaviors},
		{name: "exclusions", delta: d.Exclusions},
		{name: "binding_decisions", delta: d.BindingDecisions},
		{name: "requirements", delta: d.Requirements},
		{name: "acceptance_checks", delta: d.AcceptanceChecks},
		{name: "negative_expectations", delta: d.NegativeExpectations},
		{name: "recovery_expectations", delta: d.RecoveryExpectations},
		{name: "affected_public_paths", delta: d.AffectedPublicPaths},
	}
	for _, section := range sections {
		if err := section.delta.Validate(); err != nil {
			return fmt.Errorf("%s: %w", section.name, err)
		}
	}
	return nil
}

// SpecApprovalReceipt is the sole specification-approval authority. It binds
// the owner action to one exact immutable revision and content hash.
type SpecApprovalReceipt struct {
	SchemaVersion       string    `json:"schema_version"`
	ID                  string    `json:"id"`
	SpecificationID     string    `json:"specification_id"`
	RevisionID          string    `json:"revision_id"`
	RevisionContentHash string    `json:"revision_content_hash"`
	ApprovalTokenHash   string    `json:"approval_token_hash"`
	ApprovedBy          string    `json:"approved_by"`
	ApprovedAt          time.Time `json:"approved_at"`
}

// Validate checks an approval receipt independently of its enclosing revision.
func (r SpecApprovalReceipt) Validate() error {
	if r.SchemaVersion != SpecificationSchemaVersion {
		return fmt.Errorf("schema_version must be %q", SpecificationSchemaVersion)
	}
	for _, required := range []struct {
		name  string
		value string
	}{
		{name: "id", value: r.ID},
		{name: "specification_id", value: r.SpecificationID},
		{name: "revision_id", value: r.RevisionID},
		{name: "revision_content_hash", value: r.RevisionContentHash},
		{name: "approval_token_hash", value: r.ApprovalTokenHash},
		{name: "approved_by", value: r.ApprovedBy},
	} {
		if strings.TrimSpace(required.value) == "" {
			return fmt.Errorf("%s is required", required.name)
		}
	}
	if r.ApprovedAt.IsZero() {
		return fmt.Errorf("approved_at is required")
	}
	return nil
}

// SpecRevision is one immutable canonical specification snapshot. The typed
// body is stored here; .aether/SPEC.md is only a projection of these fields.
type SpecRevision struct {
	SchemaVersion        string                    `json:"schema_version"`
	ID                   string                    `json:"id"`
	SpecificationID      string                    `json:"specification_id"`
	PredecessorID        string                    `json:"predecessor_id"`
	CreatedAt            time.Time                 `json:"created_at"`
	ContentHash          string                    `json:"content_hash"`
	Scope                SpecScope                 `json:"scope"`
	Status               SpecRevisionStatus        `json:"status"`
	Approval             *SpecApprovalReceipt      `json:"approval"`
	Outcomes             []SpecOutcome             `json:"outcomes"`
	IncludedBehaviors    []SpecIncludedBehavior    `json:"included_behaviors"`
	Exclusions           []SpecExclusion           `json:"exclusions"`
	BindingDecisions     []SpecBindingDecision     `json:"binding_decisions"`
	Requirements         []SpecRequirement         `json:"requirements"`
	AcceptanceChecks     []SpecAcceptanceCheck     `json:"acceptance_checks"`
	NegativeExpectations []SpecNegativeExpectation `json:"negative_expectations"`
	RecoveryExpectations []SpecRecoveryExpectation `json:"recovery_expectations"`
	AffectedPublicPaths  []SpecPublicPath          `json:"affected_public_paths"`
	Delta                SpecRevisionDelta         `json:"delta"`
}

// Validate checks one immutable revision and its exact approval relationship.
func (r SpecRevision) Validate() error {
	if r.SchemaVersion != SpecificationSchemaVersion {
		return fmt.Errorf("schema_version must be %q", SpecificationSchemaVersion)
	}
	if strings.TrimSpace(r.ID) == "" {
		return fmt.Errorf("id is required")
	}
	if strings.TrimSpace(r.SpecificationID) == "" {
		return fmt.Errorf("specification_id is required")
	}
	if r.CreatedAt.IsZero() {
		return fmt.Errorf("created_at is required")
	}
	if strings.TrimSpace(r.ContentHash) == "" {
		return fmt.Errorf("content_hash is required")
	}
	if err := r.Scope.Validate(); err != nil {
		return fmt.Errorf("scope: %w", err)
	}
	if !r.Status.Valid() {
		return fmt.Errorf("status: invalid specification revision status %q", r.Status)
	}
	if r.Delta.PredecessorRevisionID != r.PredecessorID {
		return fmt.Errorf("delta.predecessor_revision_id must match predecessor_id")
	}
	if err := r.Delta.Validate(); err != nil {
		return fmt.Errorf("delta: %w", err)
	}

	if err := validateSpecRevisionBody(r); err != nil {
		return err
	}

	switch r.Status {
	case SpecStatusDraft:
		if r.Approval != nil {
			return fmt.Errorf("draft revision cannot contain approval")
		}
	case SpecStatusApproved:
		if r.Approval == nil {
			return fmt.Errorf("approval is required for approved revision")
		}
	}
	if r.Approval != nil {
		if err := r.Approval.Validate(); err != nil {
			return fmt.Errorf("approval: %w", err)
		}
		if r.Approval.SpecificationID != r.SpecificationID {
			return fmt.Errorf("approval.specification_id must match specification_id")
		}
		if r.Approval.RevisionID != r.ID {
			return fmt.Errorf("approval.revision_id must match revision id")
		}
		if r.Approval.RevisionContentHash != r.ContentHash {
			return fmt.Errorf("approval.revision_content_hash must match content_hash")
		}
	}
	return nil
}

// Specification is the optional canonical specification lineage stored in
// ColonyState. Revisions are complete snapshots so a state transaction can
// commit content, approval, and impact classification together.
type Specification struct {
	SchemaVersion     string         `json:"schema_version"`
	ID                string         `json:"id"`
	GoalID            string         `json:"goal_id"`
	CurrentRevisionID string         `json:"current_revision_id"`
	Revisions         []SpecRevision `json:"revisions"`
}

// Validate checks an ordered immutable lineage and its current revision.
func (s Specification) Validate() error {
	if s.SchemaVersion != SpecificationSchemaVersion {
		return fmt.Errorf("schema_version must be %q", SpecificationSchemaVersion)
	}
	if strings.TrimSpace(s.ID) == "" {
		return fmt.Errorf("id is required")
	}
	if strings.TrimSpace(s.GoalID) == "" {
		return fmt.Errorf("goal_id is required")
	}
	if strings.TrimSpace(s.CurrentRevisionID) == "" {
		return fmt.Errorf("current_revision_id is required")
	}
	if len(s.Revisions) == 0 {
		return fmt.Errorf("revisions are required")
	}

	seen := make(map[string]struct{}, len(s.Revisions))
	currentIndex := -1
	for i := range s.Revisions {
		revision := s.Revisions[i]
		if err := revision.Validate(); err != nil {
			return fmt.Errorf("revisions[%d]: %w", i, err)
		}
		if revision.SpecificationID != s.ID {
			return fmt.Errorf("revisions[%d].specification_id must match specification id", i)
		}
		if revision.Scope.GoalID != s.GoalID {
			return fmt.Errorf("revisions[%d].scope.goal_id must match goal_id", i)
		}
		if _, duplicate := seen[revision.ID]; duplicate {
			return fmt.Errorf("revisions[%d].id %q is duplicated", i, revision.ID)
		}
		if i == 0 && strings.TrimSpace(revision.PredecessorID) != "" {
			return fmt.Errorf("revisions[0].predecessor_id must be empty")
		}
		if i > 0 {
			if _, ok := seen[revision.PredecessorID]; !ok {
				return fmt.Errorf("revisions[%d].predecessor_id %q does not name an earlier revision", i, revision.PredecessorID)
			}
		}
		if revision.ID == s.CurrentRevisionID {
			currentIndex = i
		}
		seen[revision.ID] = struct{}{}
	}
	if currentIndex < 0 {
		return fmt.Errorf("current_revision_id %q does not name a revision", s.CurrentRevisionID)
	}
	if currentIndex != len(s.Revisions)-1 {
		return fmt.Errorf("current_revision_id must name the last immutable revision")
	}
	for i := range s.Revisions {
		if i == currentIndex && s.Revisions[i].Status == SpecStatusSuperseded {
			return fmt.Errorf("current revision cannot be superseded")
		}
		if i != currentIndex && s.Revisions[i].Status != SpecStatusSuperseded {
			return fmt.Errorf("revisions[%d] must be superseded when it is not current", i)
		}
	}
	return nil
}

func validateSpecRevisionBody(r SpecRevision) error {
	for i, check := range r.AcceptanceChecks {
		if strings.TrimSpace(check.Description) == "" {
			return fmt.Errorf("acceptance_checks[%d].description is required", i)
		}
		if strings.TrimSpace(check.Verification) == "" {
			return fmt.Errorf("acceptance_checks[%d].verification is required", i)
		}
	}
	for i, publicPath := range r.AffectedPublicPaths {
		if strings.TrimSpace(publicPath.Path) == "" {
			return fmt.Errorf("affected_public_paths[%d].path is required", i)
		}
		if strings.TrimSpace(publicPath.Description) == "" {
			return fmt.Errorf("affected_public_paths[%d].description is required", i)
		}
	}

	seen := make(map[string]string)
	validate := func(section string, entries []specStableEntry) error {
		if len(entries) == 0 {
			return fmt.Errorf("%s must contain typed stable-ID content", section)
		}
		for i, entry := range entries {
			if strings.TrimSpace(entry.id) == "" {
				return fmt.Errorf("%s[%d].id is required", section, i)
			}
			if prior, duplicate := seen[entry.id]; duplicate {
				return fmt.Errorf("%s[%d].id %q duplicates %s", section, i, entry.id, prior)
			}
			seen[entry.id] = fmt.Sprintf("%s[%d]", section, i)
			if strings.TrimSpace(entry.content) == "" {
				return fmt.Errorf("%s[%d].%s is required", section, i, entry.contentField)
			}
			if strings.TrimSpace(entry.hash) == "" {
				return fmt.Errorf("%s[%d].content_hash is required", section, i)
			}
			if err := validateSpecIDList(fmt.Sprintf("%s[%d].evidence_ids", section, i), entry.evidenceIDs); err != nil {
				return err
			}
		}
		return nil
	}

	sections := []struct {
		name    string
		entries []specStableEntry
	}{
		{name: "outcomes", entries: outcomesAsStableEntries(r.Outcomes)},
		{name: "included_behaviors", entries: includedBehaviorsAsStableEntries(r.IncludedBehaviors)},
		{name: "exclusions", entries: exclusionsAsStableEntries(r.Exclusions)},
		{name: "binding_decisions", entries: bindingDecisionsAsStableEntries(r.BindingDecisions)},
		{name: "requirements", entries: requirementsAsStableEntries(r.Requirements)},
		{name: "acceptance_checks", entries: acceptanceChecksAsStableEntries(r.AcceptanceChecks)},
		{name: "negative_expectations", entries: negativeExpectationsAsStableEntries(r.NegativeExpectations)},
		{name: "recovery_expectations", entries: recoveryExpectationsAsStableEntries(r.RecoveryExpectations)},
		{name: "affected_public_paths", entries: publicPathsAsStableEntries(r.AffectedPublicPaths)},
	}
	for _, section := range sections {
		if err := validate(section.name, section.entries); err != nil {
			return err
		}
	}
	return nil
}

type specStableEntry struct {
	id           string
	content      string
	contentField string
	hash         string
	evidenceIDs  []string
}

func outcomesAsStableEntries(values []SpecOutcome) []specStableEntry {
	result := make([]specStableEntry, len(values))
	for i, value := range values {
		result[i] = specStableEntry{id: value.ID, content: value.Description, contentField: "description", hash: value.ContentHash, evidenceIDs: value.EvidenceIDs}
	}
	return result
}

func includedBehaviorsAsStableEntries(values []SpecIncludedBehavior) []specStableEntry {
	result := make([]specStableEntry, len(values))
	for i, value := range values {
		result[i] = specStableEntry{id: value.ID, content: value.Description, contentField: "description", hash: value.ContentHash, evidenceIDs: value.EvidenceIDs}
	}
	return result
}

func exclusionsAsStableEntries(values []SpecExclusion) []specStableEntry {
	result := make([]specStableEntry, len(values))
	for i, value := range values {
		result[i] = specStableEntry{id: value.ID, content: value.Description, contentField: "description", hash: value.ContentHash, evidenceIDs: value.EvidenceIDs}
	}
	return result
}

func bindingDecisionsAsStableEntries(values []SpecBindingDecision) []specStableEntry {
	result := make([]specStableEntry, len(values))
	for i, value := range values {
		result[i] = specStableEntry{id: value.ID, content: value.Description, contentField: "description", hash: value.ContentHash, evidenceIDs: value.EvidenceIDs}
	}
	return result
}

func requirementsAsStableEntries(values []SpecRequirement) []specStableEntry {
	result := make([]specStableEntry, len(values))
	for i, value := range values {
		result[i] = specStableEntry{id: value.ID, content: value.Description, contentField: "description", hash: value.ContentHash, evidenceIDs: value.EvidenceIDs}
	}
	return result
}

func acceptanceChecksAsStableEntries(values []SpecAcceptanceCheck) []specStableEntry {
	result := make([]specStableEntry, len(values))
	for i, value := range values {
		result[i] = specStableEntry{id: value.ID, content: value.Description, contentField: "description", hash: value.ContentHash, evidenceIDs: value.EvidenceIDs}
	}
	return result
}

func negativeExpectationsAsStableEntries(values []SpecNegativeExpectation) []specStableEntry {
	result := make([]specStableEntry, len(values))
	for i, value := range values {
		result[i] = specStableEntry{id: value.ID, content: value.Description, contentField: "description", hash: value.ContentHash, evidenceIDs: value.EvidenceIDs}
	}
	return result
}

func recoveryExpectationsAsStableEntries(values []SpecRecoveryExpectation) []specStableEntry {
	result := make([]specStableEntry, len(values))
	for i, value := range values {
		result[i] = specStableEntry{id: value.ID, content: value.Description, contentField: "description", hash: value.ContentHash, evidenceIDs: value.EvidenceIDs}
	}
	return result
}

func publicPathsAsStableEntries(values []SpecPublicPath) []specStableEntry {
	result := make([]specStableEntry, len(values))
	for i, value := range values {
		result[i] = specStableEntry{id: value.ID, content: value.Description, contentField: "description", hash: value.ContentHash, evidenceIDs: value.EvidenceIDs}
	}
	return result
}

func validateSpecIDList(field string, ids []string) error {
	seen := make(map[string]struct{}, len(ids))
	for i, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			return fmt.Errorf("%s[%d] is required", field, i)
		}
		if _, duplicate := seen[id]; duplicate {
			return fmt.Errorf("%s contains duplicate ID %q", field, id)
		}
		seen[id] = struct{}{}
	}
	return nil
}

// Compile-time guard that specification enums retain explicit JSON behavior.
var (
	_ json.Marshaler   = SpecRevisionStatus("")
	_ json.Unmarshaler = (*SpecRevisionStatus)(nil)
)
