package cmd

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/calcosmic/Aether/pkg/colony"
)

// planningEvidenceRefusalCode is a closed machine-readable reason why a
// source could not become a trustworthy evidence reference. A refusal never
// returns a partially populated reference.
type planningEvidenceRefusalCode string

const (
	planningEvidenceRefusalUnsupportedKind          planningEvidenceRefusalCode = "unsupported_kind"
	planningEvidenceRefusalPathOutsideApprovedRoots planningEvidenceRefusalCode = "path_outside_approved_roots"
	planningEvidenceRefusalMissingContent           planningEvidenceRefusalCode = "missing_content"
	planningEvidenceRefusalHashMismatch             planningEvidenceRefusalCode = "hash_mismatch"
	planningEvidenceRefusalInvalidMetadata          planningEvidenceRefusalCode = "invalid_metadata"
	planningEvidenceRefusalInvalidDimension         planningEvidenceRefusalCode = "invalid_dimension"
	planningEvidenceRefusalReadFailed               planningEvidenceRefusalCode = "read_failed"
)

// planningEvidenceRefusal is deliberately safe to surface. Detail describes
// the failed contract without including source bodies or secret-shaped text.
type planningEvidenceRefusal struct {
	Code   planningEvidenceRefusalCode
	Kind   colony.PlanningEvidenceKind
	Origin string
	Detail string
}

func (r *planningEvidenceRefusal) Error() string {
	if r == nil {
		return "planning evidence refused"
	}
	detail := strings.TrimSpace(r.Detail)
	if detail == "" {
		detail = "source did not satisfy the evidence contract"
	}
	if origin := strings.TrimSpace(r.Origin); origin != "" {
		return fmt.Sprintf("planning evidence %s refused (%s): %s", origin, r.Code, detail)
	}
	return fmt.Sprintf("planning evidence refused (%s): %s", r.Code, detail)
}

// planningEvidenceScope binds evidence to the exact planning authority it may
// inform. Observing the same bytes under a different scope produces a distinct
// reference rather than universal permission.
type planningEvidenceScope struct {
	GoalID                  string
	SessionID               string
	SpecificationRevisionID string
	PlanRevisionID          string
}

// planningEvidenceSourceState carries current trust state separately from
// content identity. Revoking or quarantining a Hive entry must immediately
// make the same content address inadmissible without pretending its bytes
// changed.
type planningEvidenceSourceState string

const (
	planningEvidenceSourceCurrent     planningEvidenceSourceState = "current"
	planningEvidenceSourceSuperseded  planningEvidenceSourceState = "superseded"
	planningEvidenceSourceRevoked     planningEvidenceSourceState = "revoked"
	planningEvidenceSourceQuarantined planningEvidenceSourceState = "quarantined"
	planningEvidenceSourceDormant     planningEvidenceSourceState = "dormant"
)

func (s planningEvidenceSourceState) valid() bool {
	switch s {
	case planningEvidenceSourceCurrent,
		planningEvidenceSourceSuperseded,
		planningEvidenceSourceRevoked,
		planningEvidenceSourceQuarantined,
		planningEvidenceSourceDormant:
		return true
	default:
		return false
	}
}

type planningEvidencePolicyCode string

const (
	planningEvidencePolicyAllowed               planningEvidencePolicyCode = "allowed"
	planningEvidencePolicyInvalidReference      planningEvidencePolicyCode = "invalid_reference"
	planningEvidencePolicyInvalidDimension      planningEvidencePolicyCode = "invalid_dimension"
	planningEvidencePolicyInadmissible          planningEvidencePolicyCode = "inadmissible"
	planningEvidencePolicyAlreadyCited          planningEvidencePolicyCode = "already_cited"
	planningEvidencePolicyNotMarkedFresh        planningEvidencePolicyCode = "not_marked_fresh"
	planningEvidencePolicyCurrentSourceUnknown  planningEvidencePolicyCode = "current_source_unknown"
	planningEvidencePolicyGoalMismatch          planningEvidencePolicyCode = "goal_mismatch"
	planningEvidencePolicySessionMismatch       planningEvidencePolicyCode = "session_mismatch"
	planningEvidencePolicySpecificationMismatch planningEvidencePolicyCode = "specification_revision_mismatch"
	planningEvidencePolicyPlanMismatch          planningEvidencePolicyCode = "plan_revision_mismatch"
	planningEvidencePolicySourceSuperseded      planningEvidencePolicyCode = "source_superseded"
	planningEvidencePolicySourceRevoked         planningEvidencePolicyCode = "source_revoked"
	planningEvidencePolicySourceQuarantined     planningEvidencePolicyCode = "source_quarantined"
	planningEvidencePolicySourceDormant         planningEvidencePolicyCode = "source_dormant"
	planningEvidencePolicySourceStateUnknown    planningEvidencePolicyCode = "source_state_unknown"
	planningEvidencePolicyOwnerAnswerChanged    planningEvidencePolicyCode = "owner_answer_changed"
	planningEvidencePolicyDimensionNotClaimed   planningEvidencePolicyCode = "dimension_not_claimed"
	planningEvidencePolicyKindNotApplicable     planningEvidencePolicyCode = "kind_not_applicable"
)

// planningEvidencePolicyResult keeps a refusal reason beside the boolean so a
// renderer can explain why a score did not move instead of silently dropping
// the evidence.
type planningEvidencePolicyResult struct {
	Allowed bool
	Code    planningEvidencePolicyCode
	Reason  string
}

type planningEvidenceFrontier struct {
	Scope                     planningEvidenceScope
	CitedEvidenceIDs          []string
	CurrentSourceHashes       map[string]string
	SourceStates              map[string]planningEvidenceSourceState
	OwnerAnswerEquivalenceKey string
}

// planningOwnerAnswerEquivalence contains every authority-sensitive input
// named by D-08. Any change produces a different key and requires the owner to
// revalidate the prior answer.
type planningOwnerAnswerEquivalence struct {
	GoalID                  string
	SessionID               string
	SpecificationRevisionID string
	BasePlanRevisionID      string
	MeaningHash             string
	BehaviorHash            string
	ImpactHash              string
	RiskHash                string
	AcceptanceHash          string
}

// planningEvidenceSource is the transient input to the catalogue. Repository
// sources use RepositoryPath and are read only after all paths pass preflight;
// logical sources use Content directly. Raw Content is never retained.
type planningEvidenceSource struct {
	Kind                 colony.PlanningEvidenceKind
	Origin               string
	RepositoryPath       string
	Content              []byte
	ExpectedContentHash  string
	Scope                planningEvidenceScope
	SourceRevision       string
	ObservedAt           time.Time
	ApplicableDimensions []colony.PlanningDimension
	State                planningEvidenceSourceState
}

// planningEvidenceLocator gives callers enough information to reopen the
// source through its owner without copying an unrestricted body into planning
// state or prompts.
type planningEvidenceLocator struct {
	Kind           colony.PlanningEvidenceKind `json:"kind"`
	Origin         string                      `json:"origin"`
	RepositoryPath string                      `json:"repository_path,omitempty"`
	SourceRevision string                      `json:"source_revision"`
}

// planningEvidenceRecord is the safe catalogue projection: a typed reference,
// a bounded/redacted summary, and locator metadata only.
type planningEvidenceRecord struct {
	Reference colony.PlanningEvidenceRef `json:"reference"`
	Summary   string                     `json:"summary"`
	Locator   planningEvidenceLocator    `json:"locator"`
}

type planningEvidenceCollectionRequest struct {
	RepositoryRoot string
	ApprovedRoots  []string
	Existing       []planningEvidenceRecord
	Sources        []planningEvidenceSource
	ReadFile       func(string) ([]byte, error)
}

type preparedPlanningEvidenceSource struct {
	source   planningEvidenceSource
	fullPath string
}

// planningEvidenceApplicabilityMatrix is intentionally conservative. A
// source kind first has to be capable of supporting a dimension, and the
// individual reference must then name that dimension explicitly.
var planningEvidenceApplicabilityMatrix = map[colony.PlanningEvidenceKind]map[colony.PlanningDimension]struct{}{
	colony.PlanningEvidenceSpecification: planningEvidenceDimensionSet(
		colony.PlanningDimensionKnowledge,
		colony.PlanningDimensionRequirements,
		colony.PlanningDimensionRisks,
		colony.PlanningDimensionDependencies,
		colony.PlanningDimensionEffort,
	),
	colony.PlanningEvidenceSurvey: planningEvidenceDimensionSet(
		colony.PlanningDimensionKnowledge,
		colony.PlanningDimensionRisks,
		colony.PlanningDimensionDependencies,
		colony.PlanningDimensionEffort,
	),
	colony.PlanningEvidenceCharter: planningEvidenceDimensionSet(
		colony.PlanningDimensionKnowledge,
		colony.PlanningDimensionRequirements,
		colony.PlanningDimensionRisks,
	),
	colony.PlanningEvidenceDecision: planningEvidenceDimensionSet(
		colony.PlanningDimensionKnowledge,
		colony.PlanningDimensionRequirements,
		colony.PlanningDimensionRisks,
		colony.PlanningDimensionDependencies,
		colony.PlanningDimensionEffort,
	),
	colony.PlanningEvidenceContext: planningEvidenceDimensionSet(
		colony.PlanningDimensionKnowledge,
		colony.PlanningDimensionRisks,
		colony.PlanningDimensionDependencies,
		colony.PlanningDimensionEffort,
	),
	colony.PlanningEvidenceResearch: planningEvidenceDimensionSet(
		colony.PlanningDimensionKnowledge,
		colony.PlanningDimensionRequirements,
		colony.PlanningDimensionRisks,
		colony.PlanningDimensionDependencies,
		colony.PlanningDimensionEffort,
	),
	colony.PlanningEvidenceHive: planningEvidenceDimensionSet(
		colony.PlanningDimensionKnowledge,
		colony.PlanningDimensionRisks,
		colony.PlanningDimensionDependencies,
		colony.PlanningDimensionEffort,
	),
	colony.PlanningEvidenceOutcome: planningEvidenceDimensionSet(
		colony.PlanningDimensionKnowledge,
		colony.PlanningDimensionRequirements,
		colony.PlanningDimensionRisks,
		colony.PlanningDimensionDependencies,
		colony.PlanningDimensionEffort,
	),
}

// collectPlanningEvidence builds one deterministic catalogue. It validates
// every repository path before the first read, preserves an existing exact
// reference (including its original observation/freshness), deduplicates by
// content address, and sorts by stable ID.
func collectPlanningEvidence(request planningEvidenceCollectionRequest) ([]planningEvidenceRecord, error) {
	root, err := canonicalPlanningEvidenceRoot(request.RepositoryRoot)
	if err != nil {
		return nil, err
	}

	prepared := make([]preparedPlanningEvidenceSource, 0, len(request.Sources))
	for _, source := range request.Sources {
		item := preparedPlanningEvidenceSource{source: source}
		if strings.TrimSpace(source.RepositoryPath) != "" {
			rel, fullPath, pathErr := resolvePlanningEvidencePath(root, request.ApprovedRoots, source.RepositoryPath)
			if pathErr != nil {
				return nil, pathErr
			}
			item.source.RepositoryPath = rel
			item.source.Origin = rel
			item.fullPath = fullPath
		}
		prepared = append(prepared, item)
	}

	readFile := request.ReadFile
	if readFile == nil {
		readFile = os.ReadFile
	}

	byID := make(map[string]planningEvidenceRecord, len(request.Existing)+len(prepared))
	existing := make(map[string]struct{}, len(request.Existing))
	for _, record := range request.Existing {
		if err := validateExistingPlanningEvidenceRecord(record); err != nil {
			return nil, err
		}
		if _, duplicate := byID[record.Reference.ID]; duplicate {
			continue
		}
		byID[record.Reference.ID] = record
		existing[record.Reference.ID] = struct{}{}
	}

	for _, item := range prepared {
		source := item.source
		if item.fullPath != "" {
			if source.Content != nil {
				return nil, planningEvidenceRefusalFor(source, planningEvidenceRefusalInvalidMetadata, "repository evidence must be read from its validated path, not supplied twice")
			}
			content, readErr := readFile(item.fullPath)
			if readErr != nil {
				return nil, planningEvidenceRefusalFor(source, planningEvidenceRefusalReadFailed, "could not read the validated repository source")
			}
			source.Content = content
		}
		record, normalizeErr := normalizePlanningEvidence(source)
		if normalizeErr != nil {
			return nil, normalizeErr
		}
		if _, wasExisting := existing[record.Reference.ID]; wasExisting {
			continue
		}
		if prior, duplicate := byID[record.Reference.ID]; duplicate {
			// Earliest observation is the original for exact duplicates. Reread
			// time is intentionally excluded from the evidence address.
			if record.Reference.ObservedAt.Before(prior.Reference.ObservedAt) {
				byID[record.Reference.ID] = record
			}
			continue
		}
		byID[record.Reference.ID] = record
	}

	records := make([]planningEvidenceRecord, 0, len(byID))
	for _, record := range byID {
		records = append(records, record)
	}
	sort.Slice(records, func(i, j int) bool {
		return records[i].Reference.ID < records[j].Reference.ID
	})
	return records, nil
}

// isFreshPlanningEvidence evaluates one immutable reference against the exact
// current frontier. Wall-clock rereading is intentionally absent: freshness
// comes from new content under unchanged authority and from not having cited
// that exact reference in an earlier card.
func isFreshPlanningEvidence(ref colony.PlanningEvidenceRef, frontier planningEvidenceFrontier) planningEvidencePolicyResult {
	if err := ref.Validate(); err != nil || !planningSHA256Pattern.MatchString(ref.ContentHash) || !strings.HasSuffix(ref.ID, "-"+safePlanningEvidenceHashPrefix(ref.ContentHash)) {
		return planningEvidenceDenied(planningEvidencePolicyInvalidReference, "evidence reference is structurally invalid or not content-addressed")
	}
	if !ref.Admissible {
		reason := strings.TrimSpace(ref.AdmissibilityReason)
		if reason == "" {
			reason = "evidence is marked inadmissible"
		}
		return planningEvidenceDenied(planningEvidencePolicyInadmissible, reason)
	}

	sourceKey := planningEvidenceSourceKey(ref)
	state, hasState := frontier.SourceStates[sourceKey]
	if ref.Kind == colony.PlanningEvidenceHive && !hasState {
		return planningEvidenceDenied(planningEvidencePolicySourceStateUnknown, "Hive evidence requires an explicit current, revoked, quarantined, dormant, or superseded state")
	}
	if hasState {
		if !state.valid() {
			return planningEvidenceDenied(planningEvidencePolicySourceStateUnknown, fmt.Sprintf("source state %q is not recognized", state))
		}
		switch state {
		case planningEvidenceSourceSuperseded:
			return planningEvidenceDenied(planningEvidencePolicySourceSuperseded, "source is superseded")
		case planningEvidenceSourceRevoked:
			return planningEvidenceDenied(planningEvidencePolicySourceRevoked, "source is revoked")
		case planningEvidenceSourceQuarantined:
			return planningEvidenceDenied(planningEvidencePolicySourceQuarantined, "source is quarantined")
		case planningEvidenceSourceDormant:
			return planningEvidenceDenied(planningEvidencePolicySourceDormant, "source is dormant")
		}
	}

	if ref.GoalID != strings.TrimSpace(frontier.Scope.GoalID) {
		return planningEvidenceDenied(planningEvidencePolicyGoalMismatch, "evidence belongs to a different planning goal")
	}
	if ref.SessionID != strings.TrimSpace(frontier.Scope.SessionID) {
		return planningEvidenceDenied(planningEvidencePolicySessionMismatch, "evidence belongs to a different planning session")
	}
	if ref.SpecificationRevisionID != strings.TrimSpace(frontier.Scope.SpecificationRevisionID) {
		return planningEvidenceDenied(planningEvidencePolicySpecificationMismatch, "evidence belongs to a different approved specification revision")
	}
	if ref.PlanRevisionID != strings.TrimSpace(frontier.Scope.PlanRevisionID) {
		return planningEvidenceDenied(planningEvidencePolicyPlanMismatch, "evidence belongs to a different base plan revision")
	}

	if ref.Kind == colony.PlanningEvidenceDecision && strings.TrimSpace(frontier.OwnerAnswerEquivalenceKey) != "" && ref.SourceRevision != strings.TrimSpace(frontier.OwnerAnswerEquivalenceKey) {
		return planningEvidenceDenied(planningEvidencePolicyOwnerAnswerChanged, "the prior owner answer requires revalidation because its equivalence key changed")
	}

	currentHash, ok := frontier.CurrentSourceHashes[sourceKey]
	if !ok || strings.TrimSpace(currentHash) == "" {
		return planningEvidenceDenied(planningEvidencePolicyCurrentSourceUnknown, "current source content hash is required to prove freshness")
	}
	if currentHash != ref.ContentHash {
		return planningEvidenceDenied(planningEvidencePolicySourceSuperseded, "source is superseded by changed content")
	}
	for _, citedID := range frontier.CitedEvidenceIDs {
		if strings.TrimSpace(citedID) == ref.ID {
			return planningEvidenceDenied(planningEvidencePolicyAlreadyCited, "evidence was already cited at the prior-card frontier")
		}
	}
	if !ref.Fresh {
		return planningEvidenceDenied(planningEvidencePolicyNotMarkedFresh, "evidence was previously classified as not fresh")
	}
	return planningEvidenceAllowed("evidence is current, scope-matched, and new at this planning frontier")
}

// evidenceAppliesToDimension requires two independent facts: the evidence kind
// can support the dimension, and this exact reference explicitly claims it.
func evidenceAppliesToDimension(ref colony.PlanningEvidenceRef, dimension colony.PlanningDimension) planningEvidencePolicyResult {
	if !dimension.Valid() {
		return planningEvidenceDenied(planningEvidencePolicyInvalidDimension, fmt.Sprintf("planning dimension %q is not supported", dimension))
	}
	if !ref.Kind.Valid() {
		return planningEvidenceDenied(planningEvidencePolicyInvalidReference, fmt.Sprintf("evidence kind %q is not supported", ref.Kind))
	}
	if !ref.Admissible {
		reason := strings.TrimSpace(ref.AdmissibilityReason)
		if reason == "" {
			reason = "evidence is marked inadmissible"
		}
		return planningEvidenceDenied(planningEvidencePolicyInadmissible, reason)
	}
	if !planningEvidenceKindAllowsDimension(ref.Kind, dimension) {
		return planningEvidenceDenied(planningEvidencePolicyKindNotApplicable, fmt.Sprintf("%s evidence cannot support the %s dimension", ref.Kind, dimension))
	}
	for _, claim := range ref.ApplicableDimensions {
		if claim == dimension {
			return planningEvidenceAllowed(fmt.Sprintf("evidence explicitly claims the %s dimension", dimension))
		}
	}
	return planningEvidenceDenied(planningEvidencePolicyDimensionNotClaimed, fmt.Sprintf("evidence does not explicitly claim the %s dimension", dimension))
}

func planningOwnerAnswerEquivalenceKey(value planningOwnerAnswerEquivalence) (string, error) {
	value.GoalID = strings.TrimSpace(value.GoalID)
	value.SessionID = strings.TrimSpace(value.SessionID)
	value.SpecificationRevisionID = strings.TrimSpace(value.SpecificationRevisionID)
	value.BasePlanRevisionID = strings.TrimSpace(value.BasePlanRevisionID)
	for _, required := range []struct {
		name  string
		value string
	}{
		{name: "goal_id", value: value.GoalID},
		{name: "session_id", value: value.SessionID},
		{name: "specification_revision_id", value: value.SpecificationRevisionID},
		{name: "base_plan_revision_id", value: value.BasePlanRevisionID},
	} {
		if required.value == "" {
			return "", fmt.Errorf("owner-answer equivalence %s is required", required.name)
		}
	}
	for _, required := range []struct {
		name  string
		value string
	}{
		{name: "meaning_hash", value: value.MeaningHash},
		{name: "behavior_hash", value: value.BehaviorHash},
		{name: "impact_hash", value: value.ImpactHash},
		{name: "risk_hash", value: value.RiskHash},
		{name: "acceptance_hash", value: value.AcceptanceHash},
	} {
		if !planningSHA256Pattern.MatchString(strings.TrimSpace(required.value)) {
			return "", fmt.Errorf("owner-answer equivalence %s must be a lowercase 64-character SHA-256 digest", required.name)
		}
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("encode owner-answer equivalence: %w", err)
	}
	return "decision-equivalence-" + planningEvidenceSHA256(encoded), nil
}

func planningEvidenceSourceKey(ref colony.PlanningEvidenceRef) string {
	return string(ref.Kind) + "\x00" + strings.TrimSpace(ref.Origin)
}

func planningEvidenceAllowed(reason string) planningEvidencePolicyResult {
	return planningEvidencePolicyResult{Allowed: true, Code: planningEvidencePolicyAllowed, Reason: reason}
}

func planningEvidenceDenied(code planningEvidencePolicyCode, reason string) planningEvidencePolicyResult {
	return planningEvidencePolicyResult{Allowed: false, Code: code, Reason: reason}
}

func safePlanningEvidenceHashPrefix(hash string) string {
	if len(hash) < 12 {
		return ""
	}
	return hash[:12]
}

// normalizePlanningEvidence turns an already loaded source into a stable
// content address. Observation time is metadata, not identity, so rereading
// unchanged bytes cannot manufacture freshness.
func normalizePlanningEvidence(source planningEvidenceSource) (planningEvidenceRecord, error) {
	if !source.Kind.Valid() {
		return planningEvidenceRecord{}, planningEvidenceRefusalFor(source, planningEvidenceRefusalUnsupportedKind, fmt.Sprintf("kind %q is not supported", source.Kind))
	}

	origin := strings.TrimSpace(source.Origin)
	repositoryPath := strings.TrimSpace(source.RepositoryPath)
	if repositoryPath != "" {
		clean, err := canonicalRepositoryRelativePath(repositoryPath)
		if err != nil {
			return planningEvidenceRecord{}, planningEvidenceRefusalFor(source, planningEvidenceRefusalPathOutsideApprovedRoots, err.Error())
		}
		repositoryPath = clean
		origin = clean
	}
	if origin == "" || strings.ContainsRune(origin, '\x00') {
		return planningEvidenceRecord{}, planningEvidenceRefusalFor(source, planningEvidenceRefusalInvalidMetadata, "origin is required and cannot contain NUL")
	}
	sourceRevision := strings.TrimSpace(source.SourceRevision)
	if sourceRevision == "" {
		return planningEvidenceRecord{}, planningEvidenceRefusalFor(source, planningEvidenceRefusalInvalidMetadata, "source revision is required")
	}
	if source.ObservedAt.IsZero() {
		return planningEvidenceRecord{}, planningEvidenceRefusalFor(source, planningEvidenceRefusalInvalidMetadata, "observation time is required")
	}

	scope := planningEvidenceScope{
		GoalID:                  strings.TrimSpace(source.Scope.GoalID),
		SessionID:               strings.TrimSpace(source.Scope.SessionID),
		SpecificationRevisionID: strings.TrimSpace(source.Scope.SpecificationRevisionID),
		PlanRevisionID:          strings.TrimSpace(source.Scope.PlanRevisionID),
	}
	if scope.GoalID == "" || scope.SessionID == "" || scope.SpecificationRevisionID == "" || scope.PlanRevisionID == "" {
		return planningEvidenceRecord{}, planningEvidenceRefusalFor(source, planningEvidenceRefusalInvalidMetadata, "goal, session, specification revision, and plan revision scope are required")
	}

	dimensions, err := normalizePlanningEvidenceDimensions(source.Kind, source.ApplicableDimensions)
	if err != nil {
		return planningEvidenceRecord{}, planningEvidenceRefusalFor(source, planningEvidenceRefusalInvalidDimension, err.Error())
	}
	sourceState := source.State
	if sourceState == "" {
		sourceState = planningEvidenceSourceCurrent
	}
	if !sourceState.valid() {
		return planningEvidenceRecord{}, planningEvidenceRefusalFor(source, planningEvidenceRefusalInvalidMetadata, fmt.Sprintf("source state %q is not supported", sourceState))
	}
	content, err := normalizePlanningEvidenceContent(source.Content)
	if err != nil {
		return planningEvidenceRecord{}, planningEvidenceRefusalFor(source, planningEvidenceRefusalMissingContent, err.Error())
	}

	identity := struct {
		SchemaVersion           string                      `json:"schema_version"`
		Kind                    colony.PlanningEvidenceKind `json:"kind"`
		Origin                  string                      `json:"origin"`
		RepositoryPath          string                      `json:"repository_path,omitempty"`
		GoalID                  string                      `json:"goal_id"`
		SessionID               string                      `json:"session_id"`
		SpecificationRevisionID string                      `json:"specification_revision_id"`
		PlanRevisionID          string                      `json:"plan_revision_id"`
		SourceRevision          string                      `json:"source_revision"`
		ApplicableDimensions    []colony.PlanningDimension  `json:"applicable_dimensions"`
		Content                 string                      `json:"content"`
	}{
		SchemaVersion:           colony.PlanningEvidenceSchemaVersion,
		Kind:                    source.Kind,
		Origin:                  origin,
		RepositoryPath:          repositoryPath,
		GoalID:                  scope.GoalID,
		SessionID:               scope.SessionID,
		SpecificationRevisionID: scope.SpecificationRevisionID,
		PlanRevisionID:          scope.PlanRevisionID,
		SourceRevision:          sourceRevision,
		ApplicableDimensions:    dimensions,
		Content:                 content,
	}
	encoded, err := json.Marshal(identity)
	if err != nil {
		return planningEvidenceRecord{}, planningEvidenceRefusalFor(source, planningEvidenceRefusalInvalidMetadata, "could not encode stable source identity")
	}
	contentHash := planningEvidenceSHA256(encoded)
	if expected := strings.TrimSpace(source.ExpectedContentHash); expected != "" && expected != contentHash {
		return planningEvidenceRecord{}, planningEvidenceRefusalFor(source, planningEvidenceRefusalHashMismatch, "claimed content hash does not match normalized source content and identity")
	}

	summary := summarizePlanningEvidence(source.Kind, origin, content, contentHash)
	admissible := sourceState == planningEvidenceSourceCurrent
	admissibilityReason := "current scoped source collected"
	if !admissible {
		admissibilityReason = fmt.Sprintf("source is %s", sourceState)
	}
	reference := colony.PlanningEvidenceRef{
		SchemaVersion:           colony.PlanningEvidenceSchemaVersion,
		ID:                      fmt.Sprintf("evidence-%s-%s", source.Kind, contentHash[:12]),
		ContentHash:             contentHash,
		Kind:                    source.Kind,
		Origin:                  origin,
		RepositoryPath:          repositoryPath,
		GoalID:                  scope.GoalID,
		SessionID:               scope.SessionID,
		SpecificationRevisionID: scope.SpecificationRevisionID,
		PlanRevisionID:          scope.PlanRevisionID,
		SourceRevision:          sourceRevision,
		ObservedAt:              source.ObservedAt.UTC(),
		ExcerptDigest:           planningEvidenceSHA256([]byte(summary)),
		ApplicableDimensions:    dimensions,
		Fresh:                   admissible,
		Admissible:              admissible,
		AdmissibilityReason:     admissibilityReason,
	}
	if err := reference.Validate(); err != nil {
		return planningEvidenceRecord{}, planningEvidenceRefusalFor(source, planningEvidenceRefusalInvalidMetadata, err.Error())
	}
	return planningEvidenceRecord{
		Reference: reference,
		Summary:   summary,
		Locator: planningEvidenceLocator{
			Kind:           source.Kind,
			Origin:         origin,
			RepositoryPath: repositoryPath,
			SourceRevision: sourceRevision,
		},
	}, nil
}

func canonicalPlanningEvidenceRoot(raw string) (string, error) {
	root := strings.TrimSpace(raw)
	if root == "" {
		return "", &planningEvidenceRefusal{Code: planningEvidenceRefusalInvalidMetadata, Detail: "repository root is required"}
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return "", &planningEvidenceRefusal{Code: planningEvidenceRefusalInvalidMetadata, Detail: "repository root could not be resolved"}
	}
	real, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return "", &planningEvidenceRefusal{Code: planningEvidenceRefusalInvalidMetadata, Detail: "repository root could not be resolved"}
	}
	info, err := os.Stat(real)
	if err != nil || !info.IsDir() {
		return "", &planningEvidenceRefusal{Code: planningEvidenceRefusalInvalidMetadata, Detail: "repository root must be an existing directory"}
	}
	return filepath.Clean(real), nil
}

// resolvePlanningEvidencePath performs lexical, approved-root, symlink, and
// regular-file checks before the collector invokes ReadFile.
func resolvePlanningEvidencePath(root string, approvedRoots []string, raw string) (string, string, error) {
	rel, err := canonicalRepositoryRelativePath(raw)
	if err != nil {
		return "", "", &planningEvidenceRefusal{Code: planningEvidenceRefusalPathOutsideApprovedRoots, Origin: strings.TrimSpace(raw), Detail: err.Error()}
	}
	if len(approvedRoots) == 0 {
		return "", "", &planningEvidenceRefusal{Code: planningEvidenceRefusalPathOutsideApprovedRoots, Origin: rel, Detail: "repository source has no approved root"}
	}

	fullPath := filepath.Join(root, filepath.FromSlash(rel))
	approved := false
	for _, rawApprovedRoot := range approvedRoots {
		approvedRel, approvedErr := canonicalApprovedEvidenceRoot(rawApprovedRoot)
		if approvedErr != nil {
			return "", "", approvedErr
		}
		approvedPath := filepath.Join(root, filepath.FromSlash(approvedRel))
		if pathContainedBy(approvedPath, fullPath) {
			approved = true
			break
		}
	}
	if !approved || !pathContainedBy(root, fullPath) {
		return "", "", &planningEvidenceRefusal{Code: planningEvidenceRefusalPathOutsideApprovedRoots, Origin: rel, Detail: "repository path is outside the approved evidence roots"}
	}

	info, err := os.Lstat(fullPath)
	if err != nil {
		return "", "", &planningEvidenceRefusal{Code: planningEvidenceRefusalReadFailed, Origin: rel, Detail: "repository evidence path cannot be inspected"}
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return "", "", &planningEvidenceRefusal{Code: planningEvidenceRefusalPathOutsideApprovedRoots, Origin: rel, Detail: "repository evidence must be a regular non-symlink file"}
	}
	realPath, err := filepath.EvalSymlinks(fullPath)
	if err != nil || !pathContainedBy(root, realPath) {
		return "", "", &planningEvidenceRefusal{Code: planningEvidenceRefusalPathOutsideApprovedRoots, Origin: rel, Detail: "repository evidence resolves outside the repository"}
	}
	return rel, realPath, nil
}

func canonicalRepositoryRelativePath(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("repository path is required")
	}
	if filepath.IsAbs(raw) {
		return "", fmt.Errorf("repository path must be relative")
	}
	clean := filepath.Clean(filepath.FromSlash(raw))
	if clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("repository path escapes its root")
	}
	return filepath.ToSlash(clean), nil
}

func canonicalApprovedEvidenceRoot(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "." {
		return ".", nil
	}
	clean, err := canonicalRepositoryRelativePath(raw)
	if err != nil {
		return "", &planningEvidenceRefusal{Code: planningEvidenceRefusalPathOutsideApprovedRoots, Origin: raw, Detail: "approved root must be repository-relative"}
	}
	return clean, nil
}

func normalizePlanningEvidenceContent(raw []byte) (string, error) {
	if len(raw) == 0 {
		return "", fmt.Errorf("source content is required")
	}
	if !utf8.Valid(raw) {
		return "", fmt.Errorf("source content must be UTF-8 text")
	}
	content := strings.TrimPrefix(string(raw), "\ufeff")
	content = strings.ReplaceAll(content, "\r\n", "\n")
	content = strings.ReplaceAll(content, "\r", "\n")
	lines := strings.Split(content, "\n")
	for i := range lines {
		lines[i] = strings.TrimRight(lines[i], " \t")
	}
	content = strings.TrimSpace(strings.Join(lines, "\n"))
	if content == "" {
		return "", fmt.Errorf("source content is required")
	}
	return content, nil
}

func planningEvidenceDimensionSet(values ...colony.PlanningDimension) map[colony.PlanningDimension]struct{} {
	result := make(map[colony.PlanningDimension]struct{}, len(values))
	for _, value := range values {
		result[value] = struct{}{}
	}
	return result
}

func planningEvidenceKindAllowsDimension(kind colony.PlanningEvidenceKind, dimension colony.PlanningDimension) bool {
	dimensions, ok := planningEvidenceApplicabilityMatrix[kind]
	if !ok {
		return false
	}
	_, ok = dimensions[dimension]
	return ok
}

func normalizePlanningEvidenceDimensions(kind colony.PlanningEvidenceKind, values []colony.PlanningDimension) ([]colony.PlanningDimension, error) {
	if len(values) == 0 {
		return nil, fmt.Errorf("at least one applicable dimension claim is required")
	}
	claimed := make(map[colony.PlanningDimension]struct{}, len(values))
	for _, value := range values {
		if !value.Valid() {
			return nil, fmt.Errorf("invalid planning dimension %q", value)
		}
		if !planningEvidenceKindAllowsDimension(kind, value) {
			return nil, fmt.Errorf("evidence kind %q cannot claim planning dimension %q", kind, value)
		}
		claimed[value] = struct{}{}
	}
	result := make([]colony.PlanningDimension, 0, len(claimed))
	for _, value := range colony.PlanningDimensions() {
		if _, ok := claimed[value]; ok {
			result = append(result, value)
		}
	}
	return result, nil
}

func summarizePlanningEvidence(kind colony.PlanningEvidenceKind, origin, content, contentHash string) string {
	fallback := fmt.Sprintf("%s evidence at %s (content redacted; sha256 %s)", kind, origin, contentHash[:12])
	scan := privacyScan(content)
	if scan.Blocked {
		return fallback
	}
	clean := strings.Join(strings.Fields(scan.Clean), " ")
	const maxSummaryRunes = 240
	runes := []rune(clean)
	if len(runes) > maxSummaryRunes {
		clean = string(runes[:maxSummaryRunes]) + "…"
	}
	sanitized, err := colony.SanitizeSignalContent(clean)
	if err != nil || strings.TrimSpace(sanitized) == "" {
		return fallback
	}
	return sanitized
}

func validateExistingPlanningEvidenceRecord(record planningEvidenceRecord) error {
	if err := record.Reference.Validate(); err != nil {
		return &planningEvidenceRefusal{Code: planningEvidenceRefusalInvalidMetadata, Origin: record.Reference.Origin, Kind: record.Reference.Kind, Detail: "existing evidence reference is invalid"}
	}
	if !planningSHA256Pattern.MatchString(record.Reference.ContentHash) || !strings.HasSuffix(record.Reference.ID, "-"+record.Reference.ContentHash[:12]) {
		return &planningEvidenceRefusal{Code: planningEvidenceRefusalHashMismatch, Origin: record.Reference.Origin, Kind: record.Reference.Kind, Detail: "existing evidence reference is not content-addressed"}
	}
	if strings.TrimSpace(record.Summary) == "" || strings.TrimSpace(record.Locator.Origin) == "" {
		return &planningEvidenceRefusal{Code: planningEvidenceRefusalInvalidMetadata, Origin: record.Reference.Origin, Kind: record.Reference.Kind, Detail: "existing evidence record lacks safe summary or locator"}
	}
	return nil
}

func planningEvidenceRefusalFor(source planningEvidenceSource, code planningEvidenceRefusalCode, detail string) error {
	origin := strings.TrimSpace(source.Origin)
	if strings.TrimSpace(source.RepositoryPath) != "" {
		origin = strings.TrimSpace(source.RepositoryPath)
	}
	return &planningEvidenceRefusal{Code: code, Kind: source.Kind, Origin: origin, Detail: detail}
}

func planningEvidenceSHA256(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
