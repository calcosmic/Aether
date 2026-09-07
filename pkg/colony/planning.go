package colony

import (
	"fmt"
	"strings"
	"time"
)

const (
	PlanningSchemaVersion          = "planning/v1"
	PlanningEvidenceSchemaVersion  = "planning-evidence/v1"
	PlanningIterationSchemaVersion = "planning-iteration/v2"
	PlanningTimelineSchemaVersion  = "planning-timeline/v1"
	PlanCandidateSchemaVersion     = "plan-candidate/v1"
	PlanAcceptanceSchemaVersion    = "plan-acceptance/v1"
)

// PlanningEvidenceKind identifies the authority-neutral source of a planning
// fact. Evidence can support a proposal, but no evidence kind grants approval
// or plan acceptance by itself.
type PlanningEvidenceKind string

const (
	PlanningEvidenceSpecification PlanningEvidenceKind = "specification"
	PlanningEvidenceSurvey        PlanningEvidenceKind = "survey"
	PlanningEvidenceCharter       PlanningEvidenceKind = "charter"
	PlanningEvidenceDecision      PlanningEvidenceKind = "decision"
	PlanningEvidenceContext       PlanningEvidenceKind = "context"
	PlanningEvidenceResearch      PlanningEvidenceKind = "research"
	PlanningEvidenceHive          PlanningEvidenceKind = "hive"
	PlanningEvidenceOutcome       PlanningEvidenceKind = "outcome"
)

func (v PlanningEvidenceKind) Valid() bool {
	switch v {
	case PlanningEvidenceSpecification,
		PlanningEvidenceSurvey,
		PlanningEvidenceCharter,
		PlanningEvidenceDecision,
		PlanningEvidenceContext,
		PlanningEvidenceResearch,
		PlanningEvidenceHive,
		PlanningEvidenceOutcome:
		return true
	default:
		return false
	}
}

func (v PlanningEvidenceKind) MarshalJSON() ([]byte, error) {
	return marshalLifecycleEnum("planning evidence kind", string(v), v.Valid())
}

func (v *PlanningEvidenceKind) UnmarshalJSON(data []byte) error {
	raw, err := unmarshalLifecycleEnum(data, "planning evidence kind", func(raw string) bool {
		return PlanningEvidenceKind(raw).Valid()
	})
	if err != nil {
		return err
	}
	*v = PlanningEvidenceKind(raw)
	return nil
}

// PlanningDimension is one of the five historical planning-readiness axes.
// There is deliberately no Overall dimension: Go derives the weighted overall
// from these five proposals later in the planning pipeline.
type PlanningDimension string

const (
	PlanningDimensionKnowledge    PlanningDimension = "knowledge"
	PlanningDimensionRequirements PlanningDimension = "requirements"
	PlanningDimensionRisks        PlanningDimension = "risks"
	PlanningDimensionDependencies PlanningDimension = "dependencies"
	PlanningDimensionEffort       PlanningDimension = "effort"
)

func (v PlanningDimension) Valid() bool {
	switch v {
	case PlanningDimensionKnowledge,
		PlanningDimensionRequirements,
		PlanningDimensionRisks,
		PlanningDimensionDependencies,
		PlanningDimensionEffort:
		return true
	default:
		return false
	}
}

func (v PlanningDimension) MarshalJSON() ([]byte, error) {
	return marshalLifecycleEnum("planning dimension", string(v), v.Valid())
}

func (v *PlanningDimension) UnmarshalJSON(data []byte) error {
	raw, err := unmarshalLifecycleEnum(data, "planning dimension", func(raw string) bool {
		return PlanningDimension(raw).Valid()
	})
	if err != nil {
		return err
	}
	*v = PlanningDimension(raw)
	return nil
}

// PlanningDimensions returns the canonical stable order used by validators,
// renderers, and the later weighted-overall calculation.
func PlanningDimensions() []PlanningDimension {
	return []PlanningDimension{
		PlanningDimensionKnowledge,
		PlanningDimensionRequirements,
		PlanningDimensionRisks,
		PlanningDimensionDependencies,
		PlanningDimensionEffort,
	}
}

// PlanningGapMateriality distinguishes an evidence gap the loop may disclose
// and stop with from one that requires an owner decision.
type PlanningGapMateriality string

const (
	PlanningGapNonMaterial PlanningGapMateriality = "non_material"
	PlanningGapMaterial    PlanningGapMateriality = "material"
)

func (v PlanningGapMateriality) Valid() bool {
	switch v {
	case PlanningGapNonMaterial, PlanningGapMaterial:
		return true
	default:
		return false
	}
}

func (v PlanningGapMateriality) MarshalJSON() ([]byte, error) {
	return marshalLifecycleEnum("planning gap materiality", string(v), v.Valid())
}

func (v *PlanningGapMateriality) UnmarshalJSON(data []byte) error {
	raw, err := unmarshalLifecycleEnum(data, "planning gap materiality", func(raw string) bool {
		return PlanningGapMateriality(raw).Valid()
	})
	if err != nil {
		return err
	}
	*v = PlanningGapMateriality(raw)
	return nil
}

// PlanningSemanticChangeKind is the explicit stable-ID comparison result for
// executable plan content.
type PlanningSemanticChangeKind string

const (
	PlanningSemanticChangeAdded     PlanningSemanticChangeKind = "added"
	PlanningSemanticChangeModified  PlanningSemanticChangeKind = "modified"
	PlanningSemanticChangeRemoved   PlanningSemanticChangeKind = "removed"
	PlanningSemanticChangePreserved PlanningSemanticChangeKind = "preserved"
)

func (v PlanningSemanticChangeKind) Valid() bool {
	switch v {
	case PlanningSemanticChangeAdded,
		PlanningSemanticChangeModified,
		PlanningSemanticChangeRemoved,
		PlanningSemanticChangePreserved:
		return true
	default:
		return false
	}
}

func (v PlanningSemanticChangeKind) MarshalJSON() ([]byte, error) {
	return marshalLifecycleEnum("planning semantic change kind", string(v), v.Valid())
}

func (v *PlanningSemanticChangeKind) UnmarshalJSON(data []byte) error {
	raw, err := unmarshalLifecycleEnum(data, "planning semantic change kind", func(raw string) bool {
		return PlanningSemanticChangeKind(raw).Valid()
	})
	if err != nil {
		return err
	}
	*v = PlanningSemanticChangeKind(raw)
	return nil
}

// PlanningAuthorityImpactKind names changes in authority separately from
// semantic plan changes so approvals cannot masquerade as plan improvement.
type PlanningAuthorityImpactKind string

const (
	PlanningAuthoritySpecApproval     PlanningAuthorityImpactKind = "specification_approval"
	PlanningAuthoritySpecSupersession PlanningAuthorityImpactKind = "specification_supersession"
	PlanningAuthorityOwnerDecision    PlanningAuthorityImpactKind = "owner_decision"
	PlanningAuthorityCandidateStatus  PlanningAuthorityImpactKind = "candidate_status"
	PlanningAuthorityPlanAcceptance   PlanningAuthorityImpactKind = "plan_acceptance"
)

func (v PlanningAuthorityImpactKind) Valid() bool {
	switch v {
	case PlanningAuthoritySpecApproval,
		PlanningAuthoritySpecSupersession,
		PlanningAuthorityOwnerDecision,
		PlanningAuthorityCandidateStatus,
		PlanningAuthorityPlanAcceptance:
		return true
	default:
		return false
	}
}

func (v PlanningAuthorityImpactKind) MarshalJSON() ([]byte, error) {
	return marshalLifecycleEnum("planning authority impact kind", string(v), v.Valid())
}

func (v *PlanningAuthorityImpactKind) UnmarshalJSON(data []byte) error {
	raw, err := unmarshalLifecycleEnum(data, "planning authority impact kind", func(raw string) bool {
		return PlanningAuthorityImpactKind(raw).Valid()
	})
	if err != nil {
		return err
	}
	*v = PlanningAuthorityImpactKind(raw)
	return nil
}

// PlanningStopReason is the complete deterministic continue/stop result
// vocabulary. OwnerDecision is a pause, not a candidate-eligible automatic stop.
type PlanningStopReason string

const (
	PlanningStopContinue           PlanningStopReason = "continue"
	PlanningStopTargetMet          PlanningStopReason = "target_met"
	PlanningStopDiminishingReturns PlanningStopReason = "diminishing_returns"
	PlanningStopStalledGap         PlanningStopReason = "stalled_gap"
	PlanningStopPassCap            PlanningStopReason = "pass_cap"
	PlanningStopOwnerDecision      PlanningStopReason = "owner_decision"
)

func (v PlanningStopReason) Valid() bool {
	switch v {
	case PlanningStopContinue,
		PlanningStopTargetMet,
		PlanningStopDiminishingReturns,
		PlanningStopStalledGap,
		PlanningStopPassCap,
		PlanningStopOwnerDecision:
		return true
	default:
		return false
	}
}

func (v PlanningStopReason) MarshalJSON() ([]byte, error) {
	return marshalLifecycleEnum("planning stop reason", string(v), v.Valid())
}

func (v *PlanningStopReason) UnmarshalJSON(data []byte) error {
	raw, err := unmarshalLifecycleEnum(data, "planning stop reason", func(raw string) bool {
		return PlanningStopReason(raw).Valid()
	})
	if err != nil {
		return err
	}
	*v = PlanningStopReason(raw)
	return nil
}

// PlanCandidateStatus describes a stopped proposal without changing the active
// plan revision. Only a separate PlanAcceptanceReceipt can make it accepted.
type PlanCandidateStatus string

const (
	PlanCandidatePendingReview PlanCandidateStatus = "pending_review"
	PlanCandidateAccepted      PlanCandidateStatus = "accepted"
	PlanCandidateRejected      PlanCandidateStatus = "rejected"
	PlanCandidateSuperseded    PlanCandidateStatus = "superseded"
	PlanCandidateExpired       PlanCandidateStatus = "expired"
	PlanCandidateInvalidated   PlanCandidateStatus = "invalidated"
)

func (v PlanCandidateStatus) Valid() bool {
	switch v {
	case PlanCandidatePendingReview,
		PlanCandidateAccepted,
		PlanCandidateRejected,
		PlanCandidateSuperseded,
		PlanCandidateExpired,
		PlanCandidateInvalidated:
		return true
	default:
		return false
	}
}

func (v PlanCandidateStatus) MarshalJSON() ([]byte, error) {
	return marshalLifecycleEnum("plan candidate status", string(v), v.Valid())
}

func (v *PlanCandidateStatus) UnmarshalJSON(data []byte) error {
	raw, err := unmarshalLifecycleEnum(data, "plan candidate status", func(raw string) bool {
		return PlanCandidateStatus(raw).Valid()
	})
	if err != nil {
		return err
	}
	*v = PlanCandidateStatus(raw)
	return nil
}

// PlanRecommendationDisposition is the Queen's advice about a candidate. It
// is intentionally narrower than candidate status and grants no authority.
type PlanRecommendationDisposition string

const (
	PlanRecommendationAccept PlanRecommendationDisposition = "accept"
	PlanRecommendationRevise PlanRecommendationDisposition = "revise"
)

func (v PlanRecommendationDisposition) Valid() bool {
	switch v {
	case PlanRecommendationAccept, PlanRecommendationRevise:
		return true
	default:
		return false
	}
}

func (v PlanRecommendationDisposition) MarshalJSON() ([]byte, error) {
	return marshalLifecycleEnum("plan recommendation disposition", string(v), v.Valid())
}

func (v *PlanRecommendationDisposition) UnmarshalJSON(data []byte) error {
	raw, err := unmarshalLifecycleEnum(data, "plan recommendation disposition", func(raw string) bool {
		return PlanRecommendationDisposition(raw).Valid()
	})
	if err != nil {
		return err
	}
	*v = PlanRecommendationDisposition(raw)
	return nil
}

// PlanRecommendationProducer is the closed authority class allowed to issue a
// QueenPlanRecommendation. ProducerID names the exact implementation instance.
type PlanRecommendationProducer string

const PlanRecommendationProducerQueen PlanRecommendationProducer = "queen"

func (v PlanRecommendationProducer) Valid() bool {
	return v == PlanRecommendationProducerQueen
}

func (v PlanRecommendationProducer) MarshalJSON() ([]byte, error) {
	return marshalLifecycleEnum("plan recommendation producer", string(v), v.Valid())
}

func (v *PlanRecommendationProducer) UnmarshalJSON(data []byte) error {
	raw, err := unmarshalLifecycleEnum(data, "plan recommendation producer", func(raw string) bool {
		return PlanRecommendationProducer(raw).Valid()
	})
	if err != nil {
		return err
	}
	*v = PlanRecommendationProducer(raw)
	return nil
}

// PlanAcceptancePolicy distinguishes legacy buildable plans from current plans
// that require an exact, explicit owner acceptance receipt.
type PlanAcceptancePolicy string

const (
	PlanAcceptanceLegacyUnbound PlanAcceptancePolicy = "legacy_unbound"
	PlanAcceptanceExplicitOwner PlanAcceptancePolicy = "explicit_owner"
)

func (v PlanAcceptancePolicy) Valid() bool {
	switch v {
	case PlanAcceptanceLegacyUnbound, PlanAcceptanceExplicitOwner:
		return true
	default:
		return false
	}
}

func (v PlanAcceptancePolicy) MarshalJSON() ([]byte, error) {
	return marshalLifecycleEnum("plan acceptance policy", string(v), v.Valid())
}

func (v *PlanAcceptancePolicy) UnmarshalJSON(data []byte) error {
	raw, err := unmarshalLifecycleEnum(data, "plan acceptance policy", func(raw string) bool {
		return PlanAcceptancePolicy(raw).Valid()
	})
	if err != nil {
		return err
	}
	*v = PlanAcceptancePolicy(raw)
	return nil
}

// PlanningEvidenceRef is one content-addressed, scoped input to a planning run.
type PlanningEvidenceRef struct {
	SchemaVersion           string               `json:"schema_version"`
	ID                      string               `json:"id"`
	ContentHash             string               `json:"content_hash"`
	Kind                    PlanningEvidenceKind `json:"kind"`
	Origin                  string               `json:"origin"`
	RepositoryPath          string               `json:"repository_path"`
	GoalID                  string               `json:"goal_id"`
	SessionID               string               `json:"session_id"`
	SpecificationRevisionID string               `json:"specification_revision_id"`
	PlanRevisionID          string               `json:"plan_revision_id"`
	SourceRevision          string               `json:"source_revision"`
	ObservedAt              time.Time            `json:"observed_at"`
	ExcerptDigest           string               `json:"excerpt_digest"`
	ApplicableDimensions    []PlanningDimension  `json:"applicable_dimensions"`
	Fresh                   bool                 `json:"fresh"`
	Admissible              bool                 `json:"admissible"`
	AdmissibilityReason     string               `json:"admissibility_reason"`
}

func (r PlanningEvidenceRef) Validate() error {
	if err := validatePlanningIdentity(r.SchemaVersion, PlanningEvidenceSchemaVersion, r.ID, r.ContentHash); err != nil {
		return err
	}
	if !r.Kind.Valid() {
		return fmt.Errorf("kind: invalid planning evidence kind %q", r.Kind)
	}
	for _, required := range []struct {
		name  string
		value string
	}{
		{name: "origin", value: r.Origin},
		{name: "goal_id", value: r.GoalID},
		{name: "session_id", value: r.SessionID},
		{name: "source_revision", value: r.SourceRevision},
		{name: "excerpt_digest", value: r.ExcerptDigest},
		{name: "admissibility_reason", value: r.AdmissibilityReason},
	} {
		if strings.TrimSpace(required.value) == "" {
			return fmt.Errorf("%s is required", required.name)
		}
	}
	if r.ObservedAt.IsZero() {
		return fmt.Errorf("observed_at is required")
	}
	if len(r.ApplicableDimensions) == 0 {
		return fmt.Errorf("applicable_dimensions are required")
	}
	seen := make(map[PlanningDimension]struct{}, len(r.ApplicableDimensions))
	for i, dimension := range r.ApplicableDimensions {
		if !dimension.Valid() {
			return fmt.Errorf("applicable_dimensions[%d]: invalid planning dimension %q", i, dimension)
		}
		if _, duplicate := seen[dimension]; duplicate {
			return fmt.Errorf("applicable_dimensions contains duplicate %q", dimension)
		}
		seen[dimension] = struct{}{}
	}
	return nil
}

// PlanningGap names the current uncertainty and the concrete evidence that
// would change the next continue/stop decision.
type PlanningGap struct {
	SchemaVersion           string                 `json:"schema_version"`
	ID                      string                 `json:"id"`
	ContentHash             string                 `json:"content_hash"`
	Dimension               PlanningDimension      `json:"dimension"`
	Materiality             PlanningGapMateriality `json:"materiality"`
	Severity                int                    `json:"severity"`
	Description             string                 `json:"description"`
	EvidenceIDs             []string               `json:"evidence_ids"`
	EvidenceThatWouldChange string                 `json:"evidence_that_would_change"`
}

func (g PlanningGap) Validate() error {
	if err := validatePlanningIdentity(g.SchemaVersion, PlanningSchemaVersion, g.ID, g.ContentHash); err != nil {
		return err
	}
	if !g.Dimension.Valid() {
		return fmt.Errorf("dimension: invalid planning dimension %q", g.Dimension)
	}
	if !g.Materiality.Valid() {
		return fmt.Errorf("materiality: invalid gap materiality %q", g.Materiality)
	}
	if g.Severity < 0 {
		return fmt.Errorf("severity cannot be negative")
	}
	if strings.TrimSpace(g.Description) == "" {
		return fmt.Errorf("description is required")
	}
	if err := validatePlanningIDList("evidence_ids", g.EvidenceIDs, false); err != nil {
		return err
	}
	if strings.TrimSpace(g.EvidenceThatWouldChange) == "" {
		return fmt.Errorf("evidence_that_would_change is required")
	}
	return nil
}

// PlanningDimensionAssessment is the Route-Setter's proposal for one axis.
// Before and After remain whole-number values; no overall score is accepted.
type PlanningDimensionAssessment struct {
	SchemaVersion     string            `json:"schema_version"`
	ID                string            `json:"id"`
	ContentHash       string            `json:"content_hash"`
	Dimension         PlanningDimension `json:"dimension"`
	Before            int               `json:"before"`
	After             int               `json:"after"`
	FreshEvidenceIDs  []string          `json:"fresh_evidence_ids"`
	ResolvedGapIDs    []string          `json:"resolved_gap_ids"`
	RemainingGap      PlanningGap       `json:"remaining_gap"`
	Rationale         string            `json:"rationale"`
	ProducerReceiptID string            `json:"producer_receipt_id"`
}

func (a PlanningDimensionAssessment) Validate() error {
	if err := validatePlanningIdentity(a.SchemaVersion, PlanningSchemaVersion, a.ID, a.ContentHash); err != nil {
		return err
	}
	if !a.Dimension.Valid() {
		return fmt.Errorf("dimension: invalid planning dimension %q", a.Dimension)
	}
	if a.Before < 0 || a.Before > 100 {
		return fmt.Errorf("before must be between 0 and 100")
	}
	if a.After < 0 || a.After > 100 {
		return fmt.Errorf("after must be between 0 and 100")
	}
	if err := validatePlanningIDList("fresh_evidence_ids", a.FreshEvidenceIDs, false); err != nil {
		return err
	}
	if err := validatePlanningIDList("resolved_gap_ids", a.ResolvedGapIDs, false); err != nil {
		return err
	}
	if err := a.RemainingGap.Validate(); err != nil {
		return fmt.Errorf("remaining_gap: %w", err)
	}
	if a.RemainingGap.Dimension != a.Dimension {
		return fmt.Errorf("remaining_gap.dimension must match dimension")
	}
	if strings.TrimSpace(a.Rationale) == "" {
		return fmt.Errorf("rationale is required")
	}
	if strings.TrimSpace(a.ProducerReceiptID) == "" {
		return fmt.Errorf("producer_receipt_id is required")
	}
	return nil
}

// PlanningSemanticChange is one stable semantic node comparison.
type PlanningSemanticChange struct {
	SemanticID  string                     `json:"semantic_id"`
	ContentHash string                     `json:"content_hash"`
	Kind        PlanningSemanticChangeKind `json:"kind"`
	BeforeHash  string                     `json:"before_hash"`
	AfterHash   string                     `json:"after_hash"`
	EvidenceIDs []string                   `json:"evidence_ids"`
}

func (c PlanningSemanticChange) Validate() error {
	if strings.TrimSpace(c.SemanticID) == "" {
		return fmt.Errorf("semantic_id is required")
	}
	if strings.TrimSpace(c.ContentHash) == "" {
		return fmt.Errorf("content_hash is required")
	}
	if !c.Kind.Valid() {
		return fmt.Errorf("kind: invalid semantic change kind %q", c.Kind)
	}
	switch c.Kind {
	case PlanningSemanticChangeAdded:
		if strings.TrimSpace(c.AfterHash) == "" {
			return fmt.Errorf("after_hash is required for added change")
		}
	case PlanningSemanticChangeModified:
		if strings.TrimSpace(c.BeforeHash) == "" || strings.TrimSpace(c.AfterHash) == "" {
			return fmt.Errorf("before_hash and after_hash are required for modified change")
		}
	case PlanningSemanticChangeRemoved:
		if strings.TrimSpace(c.BeforeHash) == "" {
			return fmt.Errorf("before_hash is required for removed change")
		}
	case PlanningSemanticChangePreserved:
		if strings.TrimSpace(c.BeforeHash) == "" || c.BeforeHash != c.AfterHash {
			return fmt.Errorf("preserved change requires equal before_hash and after_hash")
		}
	}
	return validatePlanningIDList("evidence_ids", c.EvidenceIDs, false)
}

// PlanningAuthorityImpact records an authority transition separately from the
// executable semantic delta.
type PlanningAuthorityImpact struct {
	ID                  string                      `json:"id"`
	ContentHash         string                      `json:"content_hash"`
	Kind                PlanningAuthorityImpactKind `json:"kind"`
	SourceID            string                      `json:"source_id"`
	AffectedSemanticIDs []string                    `json:"affected_semantic_ids"`
	Rationale           string                      `json:"rationale"`
}

func (i PlanningAuthorityImpact) Validate() error {
	if strings.TrimSpace(i.ID) == "" {
		return fmt.Errorf("id is required")
	}
	if strings.TrimSpace(i.ContentHash) == "" {
		return fmt.Errorf("content_hash is required")
	}
	if !i.Kind.Valid() {
		return fmt.Errorf("kind: invalid authority impact kind %q", i.Kind)
	}
	if strings.TrimSpace(i.SourceID) == "" {
		return fmt.Errorf("source_id is required")
	}
	if err := validatePlanningIDList("affected_semantic_ids", i.AffectedSemanticIDs, false); err != nil {
		return err
	}
	if strings.TrimSpace(i.Rationale) == "" {
		return fmt.Errorf("rationale is required")
	}
	return nil
}

// PlanningSemanticDelta keeps every executable category explicit and carries
// authority impacts in a separate collection.
type PlanningSemanticDelta struct {
	SchemaVersion        string                    `json:"schema_version"`
	ID                   string                    `json:"id"`
	ContentHash          string                    `json:"content_hash"`
	Phases               []PlanningSemanticChange  `json:"phases"`
	Tasks                []PlanningSemanticChange  `json:"tasks"`
	Dependencies         []PlanningSemanticChange  `json:"dependencies"`
	RequirementLinks     []PlanningSemanticChange  `json:"requirement_links"`
	AcceptanceChecks     []PlanningSemanticChange  `json:"acceptance_checks"`
	NegativeExpectations []PlanningSemanticChange  `json:"negative_expectations"`
	RecoveryExpectations []PlanningSemanticChange  `json:"recovery_expectations"`
	PublicPaths          []PlanningSemanticChange  `json:"public_paths"`
	AuthorityImpacts     []PlanningAuthorityImpact `json:"authority_impacts"`
}

func (d PlanningSemanticDelta) Validate() error {
	if err := validatePlanningIdentity(d.SchemaVersion, PlanningSchemaVersion, d.ID, d.ContentHash); err != nil {
		return err
	}
	sections := []struct {
		name    string
		changes []PlanningSemanticChange
	}{
		{name: "phases", changes: d.Phases},
		{name: "tasks", changes: d.Tasks},
		{name: "dependencies", changes: d.Dependencies},
		{name: "requirement_links", changes: d.RequirementLinks},
		{name: "acceptance_checks", changes: d.AcceptanceChecks},
		{name: "negative_expectations", changes: d.NegativeExpectations},
		{name: "recovery_expectations", changes: d.RecoveryExpectations},
		{name: "public_paths", changes: d.PublicPaths},
	}
	for _, section := range sections {
		for i := range section.changes {
			if err := section.changes[i].Validate(); err != nil {
				return fmt.Errorf("%s[%d]: %w", section.name, i, err)
			}
		}
	}
	for i := range d.AuthorityImpacts {
		if err := d.AuthorityImpacts[i].Validate(); err != nil {
			return fmt.Errorf("authority_impacts[%d]: %w", i, err)
		}
	}
	return nil
}

// PlanningStopDecision is the renderer-neutral result of deterministic stop
// policy validation. It never activates a plan.
type PlanningStopDecision struct {
	SchemaVersion           string             `json:"schema_version"`
	ID                      string             `json:"id"`
	ContentHash             string             `json:"content_hash"`
	Reason                  PlanningStopReason `json:"reason"`
	SelectedGapID           string             `json:"selected_gap_id"`
	ResidualGapIDs          []string           `json:"residual_gap_ids"`
	EvidenceIDs             []string           `json:"evidence_ids"`
	Rationale               string             `json:"rationale"`
	EvidenceThatWouldChange string             `json:"evidence_that_would_change"`
}

func (d PlanningStopDecision) Validate() error {
	if err := validatePlanningIdentity(d.SchemaVersion, PlanningSchemaVersion, d.ID, d.ContentHash); err != nil {
		return err
	}
	if !d.Reason.Valid() {
		return fmt.Errorf("reason: invalid planning stop reason %q", d.Reason)
	}
	if err := validatePlanningIDList("residual_gap_ids", d.ResidualGapIDs, false); err != nil {
		return err
	}
	if err := validatePlanningIDList("evidence_ids", d.EvidenceIDs, false); err != nil {
		return err
	}
	if strings.TrimSpace(d.Rationale) == "" {
		return fmt.Errorf("rationale is required")
	}
	if strings.TrimSpace(d.EvidenceThatWouldChange) == "" {
		return fmt.Errorf("evidence_that_would_change is required")
	}
	return nil
}

// PlanningIterationCard is the immutable completed Scout-to-Route-Setter pass
// shown by renderers and appended to the planning timeline.
type PlanningIterationCard struct {
	SchemaVersion           string                        `json:"schema_version"`
	ID                      string                        `json:"id"`
	ContentHash             string                        `json:"content_hash"`
	RunID                   string                        `json:"run_id"`
	Iteration               int                           `json:"iteration"`
	ScoutReceiptID          string                        `json:"scout_receipt_id"`
	ScoutReceiptHash        string                        `json:"scout_receipt_hash"`
	RouteSetterReceiptID    string                        `json:"route_setter_receipt_id"`
	RouteSetterReceiptHash  string                        `json:"route_setter_receipt_hash"`
	EvidenceIDs             []string                      `json:"evidence_ids"`
	DimensionAssessments    []PlanningDimensionAssessment `json:"dimension_assessments"`
	WeakestGap              PlanningGap                   `json:"weakest_gap"`
	SemanticDelta           PlanningSemanticDelta         `json:"semantic_delta"`
	Decision                PlanningStopDecision          `json:"decision"`
	EvidenceThatWouldChange string                        `json:"evidence_that_would_change"`
	CreatedAt               time.Time                     `json:"created_at"`
}

func (c PlanningIterationCard) Validate() error {
	if err := validatePlanningIdentity(c.SchemaVersion, PlanningIterationSchemaVersion, c.ID, c.ContentHash); err != nil {
		return err
	}
	for _, required := range []struct {
		name  string
		value string
	}{
		{name: "run_id", value: c.RunID},
		{name: "scout_receipt_id", value: c.ScoutReceiptID},
		{name: "scout_receipt_hash", value: c.ScoutReceiptHash},
		{name: "route_setter_receipt_id", value: c.RouteSetterReceiptID},
		{name: "route_setter_receipt_hash", value: c.RouteSetterReceiptHash},
	} {
		if strings.TrimSpace(required.value) == "" {
			return fmt.Errorf("%s is required", required.name)
		}
	}
	if c.Iteration <= 0 {
		return fmt.Errorf("iteration must be positive")
	}
	if err := validatePlanningIDList("evidence_ids", c.EvidenceIDs, true); err != nil {
		return err
	}
	if err := validatePlanningAssessments(c.DimensionAssessments); err != nil {
		return err
	}
	if err := c.WeakestGap.Validate(); err != nil {
		return fmt.Errorf("weakest_gap: %w", err)
	}
	if err := c.SemanticDelta.Validate(); err != nil {
		return fmt.Errorf("semantic_delta: %w", err)
	}
	if err := c.Decision.Validate(); err != nil {
		return fmt.Errorf("decision: %w", err)
	}
	if strings.TrimSpace(c.EvidenceThatWouldChange) == "" {
		return fmt.Errorf("evidence_that_would_change is required")
	}
	if c.CreatedAt.IsZero() {
		return fmt.Errorf("created_at is required")
	}
	return nil
}

// PlanningTimelineBinding identifies the complete immutable card chain used
// by a candidate and, after acceptance, its active PlanRevision.
type PlanningTimelineBinding struct {
	SchemaVersion  string   `json:"schema_version"`
	ID             string   `json:"id"`
	ContentHash    string   `json:"content_hash"`
	RunID          string   `json:"run_id"`
	CardIDs        []string `json:"card_ids"`
	FirstCardHash  string   `json:"first_card_hash"`
	LastCardHash   string   `json:"last_card_hash"`
	TimelineDigest string   `json:"timeline_digest"`
	Path           string   `json:"path"`
}

func (b PlanningTimelineBinding) Validate() error {
	if err := validatePlanningIdentity(b.SchemaVersion, PlanningTimelineSchemaVersion, b.ID, b.ContentHash); err != nil {
		return err
	}
	for _, required := range []struct {
		name  string
		value string
	}{
		{name: "run_id", value: b.RunID},
		{name: "first_card_hash", value: b.FirstCardHash},
		{name: "last_card_hash", value: b.LastCardHash},
		{name: "timeline_digest", value: b.TimelineDigest},
		{name: "path", value: b.Path},
	} {
		if strings.TrimSpace(required.value) == "" {
			return fmt.Errorf("%s is required", required.name)
		}
	}
	return validatePlanningIDList("card_ids", b.CardIDs, true)
}

// QueenPlanRecommendation is evidence-grounded advice. Its schema contains no
// candidate-status or acceptance mutation field.
type QueenPlanRecommendation struct {
	SchemaVersion string                        `json:"schema_version"`
	ID            string                        `json:"id"`
	ContentHash   string                        `json:"content_hash"`
	CandidateID   string                        `json:"candidate_id"`
	Disposition   PlanRecommendationDisposition `json:"disposition"`
	EvidenceIDs   []string                      `json:"evidence_ids"`
	Rationale     string                        `json:"rationale"`
	Producer      PlanRecommendationProducer    `json:"producer"`
	ProducerID    string                        `json:"producer_id"`
	CreatedAt     time.Time                     `json:"created_at"`
}

func (r QueenPlanRecommendation) Validate() error {
	if err := validatePlanningIdentity(r.SchemaVersion, PlanningSchemaVersion, r.ID, r.ContentHash); err != nil {
		return err
	}
	if strings.TrimSpace(r.CandidateID) == "" {
		return fmt.Errorf("candidate_id is required")
	}
	if !r.Disposition.Valid() {
		return fmt.Errorf("disposition: invalid recommendation disposition %q", r.Disposition)
	}
	if err := validatePlanningIDList("evidence_ids", r.EvidenceIDs, true); err != nil {
		return err
	}
	if strings.TrimSpace(r.Rationale) == "" {
		return fmt.Errorf("rationale is required")
	}
	if !r.Producer.Valid() {
		return fmt.Errorf("producer: invalid recommendation producer %q", r.Producer)
	}
	if strings.TrimSpace(r.ProducerID) == "" {
		return fmt.Errorf("producer_id is required")
	}
	if r.CreatedAt.IsZero() {
		return fmt.Errorf("created_at is required")
	}
	return nil
}

// PlanCandidate is an exact stopped proposal. It remains non-active until an
// independent PlanAcceptanceReceipt is atomically bound to it.
type PlanCandidate struct {
	SchemaVersion             string                        `json:"schema_version"`
	ID                        string                        `json:"id"`
	ContentHash               string                        `json:"content_hash"`
	Status                    PlanCandidateStatus           `json:"status"`
	CreatedAt                 time.Time                     `json:"created_at"`
	ExpiresAt                 time.Time                     `json:"expires_at"`
	Proposal                  PlanRevision                  `json:"proposal"`
	ProposalHash              string                        `json:"proposal_hash"`
	BasePlanRevisionID        string                        `json:"base_plan_revision_id"`
	BasePlanRevisionHash      string                        `json:"base_plan_revision_hash"`
	SpecificationRevisionID   string                        `json:"specification_revision_id"`
	SpecificationRevisionHash string                        `json:"specification_revision_hash"`
	Timeline                  PlanningTimelineBinding       `json:"timeline"`
	StopDecision              PlanningStopDecision          `json:"stop_decision"`
	DimensionAssessments      []PlanningDimensionAssessment `json:"dimension_assessments"`
	SemanticDelta             PlanningSemanticDelta         `json:"semantic_delta"`
	ResidualGaps              []PlanningGap                 `json:"residual_gaps"`
	EvidenceThatWouldChange   string                        `json:"evidence_that_would_change"`
	Recommendation            QueenPlanRecommendation       `json:"recommendation"`
	Acceptance                *PlanAcceptanceReceipt        `json:"acceptance"`
}

func (c PlanCandidate) Validate() error {
	if err := validatePlanningIdentity(c.SchemaVersion, PlanCandidateSchemaVersion, c.ID, c.ContentHash); err != nil {
		return err
	}
	if !c.Status.Valid() {
		return fmt.Errorf("status: invalid plan candidate status %q", c.Status)
	}
	if c.CreatedAt.IsZero() {
		return fmt.Errorf("created_at is required")
	}
	if c.ExpiresAt.IsZero() {
		return fmt.Errorf("expires_at is required")
	}
	for _, required := range []struct {
		name  string
		value string
	}{
		{name: "proposal.id", value: c.Proposal.ID},
		{name: "proposal_hash", value: c.ProposalHash},
		{name: "base_plan_revision_id", value: c.BasePlanRevisionID},
		{name: "base_plan_revision_hash", value: c.BasePlanRevisionHash},
		{name: "specification_revision_id", value: c.SpecificationRevisionID},
		{name: "specification_revision_hash", value: c.SpecificationRevisionHash},
	} {
		if strings.TrimSpace(required.value) == "" {
			return fmt.Errorf("%s is required", required.name)
		}
	}
	if c.Proposal.PlanHash != c.ProposalHash {
		return fmt.Errorf("proposal_hash must match proposal.plan_hash")
	}
	if err := c.Timeline.Validate(); err != nil {
		return fmt.Errorf("timeline: %w", err)
	}
	if err := c.StopDecision.Validate(); err != nil {
		return fmt.Errorf("stop_decision: %w", err)
	}
	if c.StopDecision.Reason == PlanningStopContinue || c.StopDecision.Reason == PlanningStopOwnerDecision {
		return fmt.Errorf("stop_decision.reason %q is not candidate eligible", c.StopDecision.Reason)
	}
	if err := validatePlanningAssessments(c.DimensionAssessments); err != nil {
		return err
	}
	if err := c.SemanticDelta.Validate(); err != nil {
		return fmt.Errorf("semantic_delta: %w", err)
	}
	for i := range c.ResidualGaps {
		if err := c.ResidualGaps[i].Validate(); err != nil {
			return fmt.Errorf("residual_gaps[%d].%w", i, err)
		}
	}
	if strings.TrimSpace(c.EvidenceThatWouldChange) == "" {
		return fmt.Errorf("evidence_that_would_change is required")
	}
	if err := c.Recommendation.Validate(); err != nil {
		return fmt.Errorf("recommendation: %w", err)
	}
	if c.Recommendation.CandidateID != c.ID {
		return fmt.Errorf("recommendation.candidate_id must match candidate id")
	}

	if c.Status == PlanCandidateAccepted {
		if c.Acceptance == nil {
			return fmt.Errorf("acceptance is required for accepted candidate")
		}
		if err := c.Acceptance.Validate(); err != nil {
			return fmt.Errorf("acceptance: %w", err)
		}
		if c.Acceptance.CandidateID != c.ID || c.Acceptance.CandidateContentHash != c.ContentHash {
			return fmt.Errorf("acceptance candidate binding must match candidate")
		}
	} else if c.Acceptance != nil {
		return fmt.Errorf("acceptance is permitted only for accepted candidate")
	}
	return nil
}

// PlanAcceptanceReceipt is the sole authority that turns one exact candidate
// into one exact active PlanRevision.
type PlanAcceptanceReceipt struct {
	SchemaVersion             string    `json:"schema_version"`
	ID                        string    `json:"id"`
	ContentHash               string    `json:"content_hash"`
	CandidateID               string    `json:"candidate_id"`
	CandidateContentHash      string    `json:"candidate_content_hash"`
	SpecificationRevisionID   string    `json:"specification_revision_id"`
	SpecificationRevisionHash string    `json:"specification_revision_hash"`
	BasePlanRevisionID        string    `json:"base_plan_revision_id"`
	BasePlanRevisionHash      string    `json:"base_plan_revision_hash"`
	TimelineID                string    `json:"timeline_id"`
	TimelineDigest            string    `json:"timeline_digest"`
	ProposalHash              string    `json:"proposal_hash"`
	AcceptanceTokenHash       string    `json:"acceptance_token_hash"`
	AcceptedBy                string    `json:"accepted_by"`
	AcceptedAt                time.Time `json:"accepted_at"`
	ActivatedPlanRevisionID   string    `json:"activated_plan_revision_id"`
	ActivatedPlanRevisionHash string    `json:"activated_plan_revision_hash"`
}

func (r PlanAcceptanceReceipt) Validate() error {
	if err := validatePlanningIdentity(r.SchemaVersion, PlanAcceptanceSchemaVersion, r.ID, r.ContentHash); err != nil {
		return err
	}
	for _, required := range []struct {
		name  string
		value string
	}{
		{name: "candidate_id", value: r.CandidateID},
		{name: "candidate_content_hash", value: r.CandidateContentHash},
		{name: "specification_revision_id", value: r.SpecificationRevisionID},
		{name: "specification_revision_hash", value: r.SpecificationRevisionHash},
		{name: "base_plan_revision_id", value: r.BasePlanRevisionID},
		{name: "base_plan_revision_hash", value: r.BasePlanRevisionHash},
		{name: "timeline_id", value: r.TimelineID},
		{name: "timeline_digest", value: r.TimelineDigest},
		{name: "proposal_hash", value: r.ProposalHash},
		{name: "acceptance_token_hash", value: r.AcceptanceTokenHash},
		{name: "accepted_by", value: r.AcceptedBy},
		{name: "activated_plan_revision_id", value: r.ActivatedPlanRevisionID},
		{name: "activated_plan_revision_hash", value: r.ActivatedPlanRevisionHash},
	} {
		if strings.TrimSpace(required.value) == "" {
			return fmt.Errorf("%s is required", required.name)
		}
	}
	if r.AcceptedAt.IsZero() {
		return fmt.Errorf("accepted_at is required")
	}
	return nil
}

func validatePlanningIdentity(schema, wantSchema, id, contentHash string) error {
	if schema != wantSchema {
		return fmt.Errorf("schema_version must be %q", wantSchema)
	}
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("id is required")
	}
	if strings.TrimSpace(contentHash) == "" {
		return fmt.Errorf("content_hash is required")
	}
	return nil
}

func validatePlanningIDList(field string, ids []string, required bool) error {
	if required && len(ids) == 0 {
		return fmt.Errorf("%s are required", field)
	}
	return validateSpecIDList(field, ids)
}

func validatePlanningAssessments(assessments []PlanningDimensionAssessment) error {
	dimensions := PlanningDimensions()
	if len(assessments) != len(dimensions) {
		return fmt.Errorf("dimension_assessments must contain exactly five dimensions")
	}
	seen := make(map[PlanningDimension]struct{}, len(assessments))
	for i := range assessments {
		if err := assessments[i].Validate(); err != nil {
			return fmt.Errorf("dimension_assessments[%d]: %w", i, err)
		}
		if _, duplicate := seen[assessments[i].Dimension]; duplicate {
			return fmt.Errorf("dimension_assessments contains duplicate %q", assessments[i].Dimension)
		}
		seen[assessments[i].Dimension] = struct{}{}
	}
	for _, dimension := range dimensions {
		if _, ok := seen[dimension]; !ok {
			return fmt.Errorf("dimension_assessments missing %q", dimension)
		}
	}
	return nil
}
