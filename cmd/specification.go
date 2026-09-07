package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/calcosmic/Aether/pkg/colony"
)

// specificationBodySection is deliberately closed over the nine owner-readable
// categories in the canonical specification contract. Keeping this typed here
// prevents command callers from smuggling content into a generic items bag.
type specificationBodySection string

const (
	specificationSectionOutcomes             specificationBodySection = "outcomes"
	specificationSectionIncludedBehaviors    specificationBodySection = "included_behaviors"
	specificationSectionExclusions           specificationBodySection = "exclusions"
	specificationSectionBindingDecisions     specificationBodySection = "binding_decisions"
	specificationSectionRequirements         specificationBodySection = "requirements"
	specificationSectionAcceptanceChecks     specificationBodySection = "acceptance_checks"
	specificationSectionNegativeExpectations specificationBodySection = "negative_expectations"
	specificationSectionRecoveryExpectations specificationBodySection = "recovery_expectations"
	specificationSectionAffectedPublicPaths  specificationBodySection = "affected_public_paths"
)

var specificationBodyOrder = []specificationBodySection{
	specificationSectionOutcomes,
	specificationSectionIncludedBehaviors,
	specificationSectionExclusions,
	specificationSectionBindingDecisions,
	specificationSectionRequirements,
	specificationSectionAcceptanceChecks,
	specificationSectionNegativeExpectations,
	specificationSectionRecoveryExpectations,
	specificationSectionAffectedPublicPaths,
}

func (section specificationBodySection) valid() bool {
	for _, known := range specificationBodyOrder {
		if section == known {
			return true
		}
	}
	return false
}

func (section specificationBodySection) idPrefix() string {
	switch section {
	case specificationSectionOutcomes:
		return "outcome"
	case specificationSectionIncludedBehaviors:
		return "behavior"
	case specificationSectionExclusions:
		return "exclusion"
	case specificationSectionBindingDecisions:
		return "decision"
	case specificationSectionRequirements:
		return "requirement"
	case specificationSectionAcceptanceChecks:
		return "acceptance"
	case specificationSectionNegativeExpectations:
		return "negative"
	case specificationSectionRecoveryExpectations:
		return "recovery"
	case specificationSectionAffectedPublicPaths:
		return "path"
	default:
		return ""
	}
}

// specificationItemInput carries semantic lineage plus section-specific body
// fields. Description is used by every section, Verification only by owner
// acceptance, and Path only by affected public paths.
type specificationItemInput struct {
	Lineage      string
	Description  string
	Verification string
	Path         string
	EvidenceIDs  []string
}

type specificationDraftRequest struct {
	Scope                colony.SpecScope
	Outcomes             []specificationItemInput
	IncludedBehaviors    []specificationItemInput
	Exclusions           []specificationItemInput
	BindingDecisions     []specificationItemInput
	Requirements         []specificationItemInput
	AcceptanceChecks     []specificationItemInput
	NegativeExpectations []specificationItemInput
	RecoveryExpectations []specificationItemInput
	AffectedPublicPaths  []specificationItemInput
	CreatedAt            time.Time
}

type specificationChangeOperation string

const (
	specificationChangeAdd    specificationChangeOperation = "add"
	specificationChangeModify specificationChangeOperation = "modify"
	specificationChangeRemove specificationChangeOperation = "remove"
)

func (operation specificationChangeOperation) valid() bool {
	return operation == specificationChangeAdd || operation == specificationChangeModify || operation == specificationChangeRemove
}

type specificationRevisionChange struct {
	Operation specificationChangeOperation
	Section   specificationBodySection
	TargetID  string
	Item      specificationItemInput
}

type specificationRevisionRequest struct {
	PredecessorRevisionID  string
	PredecessorContentHash string
	Scope                  colony.SpecScope
	Changes                []specificationRevisionChange
	DecisionResolution     *planningDecisionResolution
	CreatedAt              time.Time
}

type specificationMutationOptions struct {
	Fault  lifecycleTransactionFaultHook
	Rename func(oldPath, newPath string) error
}

// specificationAffectedScope is the immediate, non-transitive impact seed.
// Plan 200-17 expands this through dependency edges; this engine deliberately
// does not rewrite or activate a plan while creating the successor contract.
type specificationAffectedScope struct {
	SpecItemIDs    []string `json:"spec_item_ids"`
	RequirementIDs []string `json:"requirement_ids"`
	TaskIDs        []string `json:"task_ids"`
	ProofLinkIDs   []string `json:"proof_link_ids"`
}

type specificationMutationResult struct {
	Specification colony.Specification
	Revision      colony.SpecRevision
	AffectedScope specificationAffectedScope
	Receipt       colony.LifecycleReceipt
	Replayed      bool
}

type specificationCanonicalItem struct {
	ID           string
	Description  string
	Verification string
	Path         string
	ContentHash  string
	EvidenceIDs  []string
}

type specificationBodySnapshot map[specificationBodySection][]specificationCanonicalItem

type specificationTransactionTarget struct {
	Root    lifecycleTransactionRootKind
	Path    string
	Content []byte
}

// buildSpecificationDraft is pure: callers can validate and present the exact
// draft before createSpecificationDraft performs the authoritative write.
func buildSpecificationDraft(request specificationDraftRequest) (colony.Specification, colony.SpecRevision, error) {
	scope, err := canonicalSpecificationScope(request.Scope)
	if err != nil {
		return colony.Specification{}, colony.SpecRevision{}, fmt.Errorf("specification draft scope: %w", err)
	}
	if request.CreatedAt.IsZero() {
		return colony.Specification{}, colony.SpecRevision{}, fmt.Errorf("specification draft created_at is required")
	}

	inputs := map[specificationBodySection][]specificationItemInput{
		specificationSectionOutcomes:             request.Outcomes,
		specificationSectionIncludedBehaviors:    request.IncludedBehaviors,
		specificationSectionExclusions:           request.Exclusions,
		specificationSectionBindingDecisions:     request.BindingDecisions,
		specificationSectionRequirements:         request.Requirements,
		specificationSectionAcceptanceChecks:     request.AcceptanceChecks,
		specificationSectionNegativeExpectations: request.NegativeExpectations,
		specificationSectionRecoveryExpectations: request.RecoveryExpectations,
		specificationSectionAffectedPublicPaths:  request.AffectedPublicPaths,
	}
	body := make(specificationBodySnapshot, len(specificationBodyOrder))
	for _, section := range specificationBodyOrder {
		items, buildErr := canonicalSpecificationInputItems(section, inputs[section])
		if buildErr != nil {
			return colony.Specification{}, colony.SpecRevision{}, buildErr
		}
		body[section] = items
	}
	if err := validateSpecificationBodyIDs(body); err != nil {
		return colony.Specification{}, colony.SpecRevision{}, err
	}
	if err := validateSpecificationScopeBindings(scope, nil, body); err != nil {
		return colony.Specification{}, colony.SpecRevision{}, err
	}

	specificationID, err := specificationLineageID(scope.GoalID)
	if err != nil {
		return colony.Specification{}, colony.SpecRevision{}, err
	}
	revision := colony.SpecRevision{
		SchemaVersion:   colony.SpecificationSchemaVersion,
		SpecificationID: specificationID,
		CreatedAt:       request.CreatedAt.UTC(),
		Scope:           scope,
		Status:          colony.SpecStatusDraft,
		Delta:           initialSpecificationDelta(body),
	}
	applySpecificationBody(&revision, body)
	if err := addressSpecificationRevision(&revision); err != nil {
		return colony.Specification{}, colony.SpecRevision{}, err
	}
	specification := colony.Specification{
		SchemaVersion:     colony.SpecificationSchemaVersion,
		ID:                specificationID,
		GoalID:            scope.GoalID,
		CurrentRevisionID: revision.ID,
		Revisions:         []colony.SpecRevision{revision},
	}
	if err := validateSpecificationState(specification); err != nil {
		return colony.Specification{}, colony.SpecRevision{}, fmt.Errorf("validate specification draft: %w", err)
	}
	return specification, revision, nil
}

// createSpecificationDraft commits the canonical state snapshot through the
// shared lifecycle transaction. Exact settled-input replay returns the same
// revision and durable receipt; an existing different lineage is never forked.
func createSpecificationDraft(root string, request specificationDraftRequest, opts specificationMutationOptions) (specificationMutationResult, error) {
	empty := specificationMutationResult{}
	repositoryRoot, err := canonicalSpecificationRoot(root)
	if err != nil {
		return empty, err
	}
	state, err := loadSpecificationColonyState(repositoryRoot)
	if err != nil {
		return empty, err
	}
	built, revision, err := buildSpecificationDraft(request)
	if err != nil {
		return empty, err
	}

	if state.Specification != nil {
		current, ok := currentSpecificationRevision(*state.Specification)
		if !ok {
			return empty, fmt.Errorf("existing specification has no current revision")
		}
		if state.Specification.ID != built.ID || current.ID != revision.ID || current.ContentHash != revision.ContentHash || len(state.Specification.Revisions) != 1 {
			return empty, fmt.Errorf("divergent specification draft replay refused; revise the current lineage explicitly")
		}
		if current.Status != colony.SpecStatusDraft || current.Approval != nil {
			return empty, fmt.Errorf("current specification revision %q is %s; create an explicit successor instead of replacing it", current.ID, current.Status)
		}
		stateBytes, marshalErr := marshalSpecificationState(state)
		if marshalErr != nil {
			return empty, marshalErr
		}
		transactionID := specificationTransactionID("draft", revision.ContentHash)
		receipt, commitErr := commitSpecificationTargets(repositoryRoot, transactionID, "specification-draft", []specificationTransactionTarget{{
			Root: lifecycleTransactionRootData, Path: "COLONY_STATE.json", Content: stateBytes,
		}}, opts)
		if commitErr != nil {
			return empty, fmt.Errorf("replay specification draft: %w", commitErr)
		}
		return specificationMutationResult{
			Specification: *state.Specification,
			Revision:      current,
			Receipt:       receipt,
			Replayed:      true,
		}, nil
	}

	updated, err := cloneColonyState(state)
	if err != nil {
		return empty, fmt.Errorf("clone state for specification draft: %w", err)
	}
	updated.Specification = &built
	if err := validatePlanningState(updated); err != nil {
		return empty, fmt.Errorf("validate state with specification draft: %w", err)
	}
	stateBytes, err := marshalSpecificationState(updated)
	if err != nil {
		return empty, err
	}
	transactionID := specificationTransactionID("draft", revision.ContentHash)
	receipt, err := commitSpecificationTargets(repositoryRoot, transactionID, "specification-draft", []specificationTransactionTarget{{
		Root: lifecycleTransactionRootData, Path: "COLONY_STATE.json", Content: stateBytes,
	}}, opts)
	if err != nil {
		return empty, fmt.Errorf("commit specification draft: %w", err)
	}
	return specificationMutationResult{Specification: built, Revision: revision, Receipt: receipt}, nil
}

// buildSpecificationSuccessor is the pure add/modify/remove reducer. The
// predecessor body is cloned, only explicit operations are applied, and the
// classified delta plus immediate plan-link impact are derived from snapshots.
func buildSpecificationSuccessor(specification colony.Specification, plan colony.Plan, request specificationRevisionRequest) (colony.Specification, colony.SpecRevision, specificationAffectedScope, error) {
	emptyScope := specificationAffectedScope{}
	if err := validateSpecificationState(specification); err != nil {
		return colony.Specification{}, colony.SpecRevision{}, emptyScope, fmt.Errorf("validate predecessor specification: %w", err)
	}
	if len(request.Changes) == 0 {
		return colony.Specification{}, colony.SpecRevision{}, emptyScope, fmt.Errorf("specification revision requires at least one classified change")
	}
	if request.CreatedAt.IsZero() {
		return colony.Specification{}, colony.SpecRevision{}, emptyScope, fmt.Errorf("specification revision created_at is required")
	}
	current, ok := currentSpecificationRevision(specification)
	if !ok {
		return colony.Specification{}, colony.SpecRevision{}, emptyScope, fmt.Errorf("specification has no current revision")
	}
	if current.ID != strings.TrimSpace(request.PredecessorRevisionID) || current.ContentHash != strings.TrimSpace(request.PredecessorContentHash) {
		return colony.Specification{}, colony.SpecRevision{}, emptyScope, fmt.Errorf("stale predecessor: current revision is %s (%s)", current.ID, current.ContentHash)
	}
	scope, err := canonicalSpecificationScope(request.Scope)
	if err != nil {
		return colony.Specification{}, colony.SpecRevision{}, emptyScope, fmt.Errorf("specification revision scope: %w", err)
	}
	if scope.GoalID != specification.GoalID {
		return colony.Specification{}, colony.SpecRevision{}, emptyScope, fmt.Errorf("specification revision crosses goal lineage")
	}
	if request.DecisionResolution != nil {
		if err := validateSpecificationDecisionResolution(*request.DecisionResolution); err != nil {
			return colony.Specification{}, colony.SpecRevision{}, emptyScope, err
		}
	}

	predecessorBody := specificationBodyFromRevision(current)
	successorBody := cloneSpecificationBody(predecessorBody)
	seenOperations := make(map[string]struct{}, len(request.Changes))
	for index, change := range request.Changes {
		if !change.Operation.valid() {
			return colony.Specification{}, colony.SpecRevision{}, emptyScope, fmt.Errorf("changes[%d] has invalid operation %q", index, change.Operation)
		}
		if !change.Section.valid() {
			return colony.Specification{}, colony.SpecRevision{}, emptyScope, fmt.Errorf("changes[%d] has invalid section %q", index, change.Section)
		}
		operationKey, keyErr := specificationRevisionOperationKey(change)
		if keyErr != nil {
			return colony.Specification{}, colony.SpecRevision{}, emptyScope, fmt.Errorf("changes[%d]: %w", index, keyErr)
		}
		if _, duplicate := seenOperations[operationKey]; duplicate {
			return colony.Specification{}, colony.SpecRevision{}, emptyScope, fmt.Errorf("ambiguous change: stable ID %q is targeted more than once", operationKey)
		}
		seenOperations[operationKey] = struct{}{}
		if _, applyErr := applySpecificationRevisionChange(successorBody, change); applyErr != nil {
			return colony.Specification{}, colony.SpecRevision{}, emptyScope, fmt.Errorf("changes[%d]: %w", index, applyErr)
		}
	}
	if err := validateSpecificationBodyIDs(successorBody); err != nil {
		return colony.Specification{}, colony.SpecRevision{}, emptyScope, err
	}
	if err := validateSpecificationScopeBindings(scope, predecessorBody, successorBody); err != nil {
		return colony.Specification{}, colony.SpecRevision{}, emptyScope, err
	}

	delta := compareSpecificationBodies(current.ID, predecessorBody, successorBody)
	if err := validateSpecificationFeatureDelta(scope, delta); err != nil {
		return colony.Specification{}, colony.SpecRevision{}, emptyScope, err
	}
	affected := affectedSpecificationScope(delta, plan)
	if request.DecisionResolution != nil {
		if err := validateSpecificationDecisionAffectedIDs(*request.DecisionResolution, affected.SpecItemIDs); err != nil {
			return colony.Specification{}, colony.SpecRevision{}, emptyScope, err
		}
	}

	successor := colony.SpecRevision{
		SchemaVersion:   colony.SpecificationSchemaVersion,
		SpecificationID: specification.ID,
		PredecessorID:   current.ID,
		CreatedAt:       request.CreatedAt.UTC(),
		Scope:           scope,
		Status:          colony.SpecStatusDraft,
		Delta:           delta,
	}
	applySpecificationBody(&successor, successorBody)
	if err := addressSpecificationRevision(&successor); err != nil {
		return colony.Specification{}, colony.SpecRevision{}, emptyScope, err
	}

	result, err := cloneSpecification(specification)
	if err != nil {
		return colony.Specification{}, colony.SpecRevision{}, emptyScope, err
	}
	result.Revisions[len(result.Revisions)-1].Status = colony.SpecStatusSuperseded
	result.Revisions = append(result.Revisions, successor)
	result.CurrentRevisionID = successor.ID
	if err := validateSpecificationState(result); err != nil {
		return colony.Specification{}, colony.SpecRevision{}, emptyScope, fmt.Errorf("validate specification successor: %w", err)
	}
	return result, successor, affected, nil
}

// reviseSpecification writes one immutable successor. A retry names the same
// predecessor and operations; if that exact successor is already current its
// original lifecycle receipt is returned without appending another revision.
func reviseSpecification(root string, request specificationRevisionRequest, opts specificationMutationOptions) (specificationMutationResult, error) {
	empty := specificationMutationResult{}
	repositoryRoot, err := canonicalSpecificationRoot(root)
	if err != nil {
		return empty, err
	}
	state, err := loadSpecificationColonyState(repositoryRoot)
	if err != nil {
		return empty, err
	}
	if state.Specification == nil {
		return empty, fmt.Errorf("no specification exists; create a draft before revising")
	}

	predecessorIndex := specificationRevisionIndex(*state.Specification, strings.TrimSpace(request.PredecessorRevisionID))
	if predecessorIndex < 0 {
		return empty, fmt.Errorf("stale predecessor: revision %q does not exist", request.PredecessorRevisionID)
	}
	predecessor := state.Specification.Revisions[predecessorIndex]
	if predecessor.ContentHash != strings.TrimSpace(request.PredecessorContentHash) {
		return empty, fmt.Errorf("stale predecessor: revision %q content hash changed", predecessor.ID)
	}

	base, err := specificationAtPredecessor(*state.Specification, predecessorIndex)
	if err != nil {
		return empty, err
	}
	built, successor, affected, err := buildSpecificationSuccessor(base, state.Plan, request)
	if err != nil {
		return empty, err
	}
	current, _ := currentSpecificationRevision(*state.Specification)
	if current.ID != predecessor.ID {
		if current.ID != successor.ID || current.ContentHash != successor.ContentHash || predecessorIndex != len(state.Specification.Revisions)-2 {
			return empty, fmt.Errorf("divergent specification revision replay refused; current revision is %q", current.ID)
		}
		if current.Status != colony.SpecStatusDraft || current.Approval != nil {
			return empty, fmt.Errorf("stale predecessor: exact successor %q has since changed authority", current.ID)
		}
		stateBytes, marshalErr := marshalSpecificationState(state)
		if marshalErr != nil {
			return empty, marshalErr
		}
		transactionID := specificationTransactionID("revise", successor.ContentHash)
		receipt, commitErr := commitSpecificationTargets(repositoryRoot, transactionID, "specification-revise", []specificationTransactionTarget{{
			Root: lifecycleTransactionRootData, Path: "COLONY_STATE.json", Content: stateBytes,
		}}, opts)
		if commitErr != nil {
			return empty, fmt.Errorf("replay specification revision: %w", commitErr)
		}
		return specificationMutationResult{
			Specification: *state.Specification,
			Revision:      current,
			AffectedScope: affected,
			Receipt:       receipt,
			Replayed:      true,
		}, nil
	}

	updated, err := cloneColonyState(state)
	if err != nil {
		return empty, fmt.Errorf("clone state for specification revision: %w", err)
	}
	updated.Specification = &built
	if err := validatePlanningState(updated); err != nil {
		return empty, fmt.Errorf("validate state with specification successor: %w", err)
	}
	stateBytes, err := marshalSpecificationState(updated)
	if err != nil {
		return empty, err
	}
	transactionID := specificationTransactionID("revise", successor.ContentHash)
	receipt, err := commitSpecificationTargets(repositoryRoot, transactionID, "specification-revise", []specificationTransactionTarget{{
		Root: lifecycleTransactionRootData, Path: "COLONY_STATE.json", Content: stateBytes,
	}}, opts)
	if err != nil {
		return empty, fmt.Errorf("commit specification successor: %w", err)
	}
	return specificationMutationResult{Specification: built, Revision: successor, AffectedScope: affected, Receipt: receipt}, nil
}

func canonicalSpecificationInputItems(section specificationBodySection, inputs []specificationItemInput) ([]specificationCanonicalItem, error) {
	if len(inputs) == 0 {
		return nil, fmt.Errorf("%s must contain typed stable-ID content", section)
	}
	items := make([]specificationCanonicalItem, 0, len(inputs))
	seen := make(map[string]struct{}, len(inputs))
	for index, input := range inputs {
		id, err := specificationStableID(section, input.Lineage)
		if err != nil {
			return nil, fmt.Errorf("%s[%d]: %w", section, index, err)
		}
		if _, duplicate := seen[id]; duplicate {
			return nil, fmt.Errorf("%s contains duplicate ID %q", section, id)
		}
		seen[id] = struct{}{}
		item, err := canonicalSpecificationItem(section, id, input)
		if err != nil {
			return nil, fmt.Errorf("%s[%d]: %w", section, index, err)
		}
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	return items, nil
}

func canonicalSpecificationItem(section specificationBodySection, id string, input specificationItemInput) (specificationCanonicalItem, error) {
	if !section.valid() {
		return specificationCanonicalItem{}, fmt.Errorf("invalid specification section %q", section)
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return specificationCanonicalItem{}, fmt.Errorf("stable ID is required")
	}
	if strings.TrimSpace(input.Lineage) != "" {
		derived, err := specificationStableID(section, input.Lineage)
		if err != nil {
			return specificationCanonicalItem{}, err
		}
		if derived != id {
			return specificationCanonicalItem{}, fmt.Errorf("semantic lineage resolves to %q, not target %q", derived, id)
		}
	}
	description := strings.TrimSpace(input.Description)
	verification := strings.TrimSpace(input.Verification)
	publicPath := strings.TrimSpace(input.Path)
	if description == "" {
		return specificationCanonicalItem{}, fmt.Errorf("description is required")
	}
	switch section {
	case specificationSectionAcceptanceChecks:
		if verification == "" {
			return specificationCanonicalItem{}, fmt.Errorf("verification is required for acceptance_checks")
		}
		if publicPath != "" {
			return specificationCanonicalItem{}, fmt.Errorf("path is only valid for affected_public_paths")
		}
	case specificationSectionAffectedPublicPaths:
		if publicPath == "" {
			return specificationCanonicalItem{}, fmt.Errorf("path is required for affected_public_paths")
		}
		if verification != "" {
			return specificationCanonicalItem{}, fmt.Errorf("verification is only valid for acceptance_checks")
		}
	default:
		if verification != "" || publicPath != "" {
			return specificationCanonicalItem{}, fmt.Errorf("section %s accepts description and evidence only", section)
		}
	}
	evidence, err := canonicalSpecificationIDs("evidence_ids", input.EvidenceIDs, true)
	if err != nil {
		return specificationCanonicalItem{}, err
	}
	hash, err := jsonSHA256(struct {
		Section      specificationBodySection `json:"section"`
		Description  string                   `json:"description"`
		Verification string                   `json:"verification,omitempty"`
		Path         string                   `json:"path,omitempty"`
		EvidenceIDs  []string                 `json:"evidence_ids"`
	}{section, description, verification, publicPath, evidence})
	if err != nil {
		return specificationCanonicalItem{}, fmt.Errorf("hash specification item %q: %w", id, err)
	}
	return specificationCanonicalItem{
		ID: id, Description: description, Verification: verification, Path: publicPath,
		ContentHash: hash, EvidenceIDs: evidence,
	}, nil
}

func specificationStableID(section specificationBodySection, lineage string) (string, error) {
	if !section.valid() {
		return "", fmt.Errorf("invalid specification section %q", section)
	}
	canonical := canonicalSpecificationLineage(lineage)
	if canonical == "" {
		return "", fmt.Errorf("semantic lineage is required")
	}
	digest, err := jsonSHA256(struct {
		Section specificationBodySection `json:"section"`
		Lineage string                   `json:"lineage"`
	}{section, canonical})
	if err != nil {
		return "", fmt.Errorf("hash semantic lineage: %w", err)
	}
	slug := canonical
	if len(slug) > 40 {
		slug = strings.Trim(slug[:40], "-")
	}
	return fmt.Sprintf("%s-%s-%s", section.idPrefix(), slug, digest[:8]), nil
}

func canonicalSpecificationLineage(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var builder strings.Builder
	separator := false
	for _, r := range value {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			if separator && builder.Len() > 0 {
				builder.WriteByte('-')
			}
			separator = false
			builder.WriteRune(r)
			continue
		}
		separator = true
	}
	return strings.Trim(builder.String(), "-")
}

func specificationLineageID(goalID string) (string, error) {
	goalID = strings.TrimSpace(goalID)
	if goalID == "" {
		return "", fmt.Errorf("specification goal ID is required")
	}
	hash, err := jsonSHA256(struct {
		GoalID string `json:"goal_id"`
	}{goalID})
	if err != nil {
		return "", err
	}
	return "specification-" + hash[:12], nil
}

func canonicalSpecificationScope(scope colony.SpecScope) (colony.SpecScope, error) {
	scope.GoalID = strings.TrimSpace(scope.GoalID)
	scope.SessionID = strings.TrimSpace(scope.SessionID)
	scope.FeatureID = strings.TrimSpace(scope.FeatureID)
	var err error
	scope.RequirementIDs, err = canonicalSpecificationIDs("requirement_ids", scope.RequirementIDs, false)
	if err != nil {
		return colony.SpecScope{}, err
	}
	scope.AcceptanceCheckIDs, err = canonicalSpecificationIDs("acceptance_check_ids", scope.AcceptanceCheckIDs, false)
	if err != nil {
		return colony.SpecScope{}, err
	}
	if err := scope.Validate(); err != nil {
		return colony.SpecScope{}, err
	}
	return scope, nil
}

func canonicalSpecificationIDs(field string, values []string, requireOne bool) ([]string, error) {
	if requireOne && len(values) == 0 {
		return nil, fmt.Errorf("%s requires at least one evidence reference", field)
	}
	result := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for index, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			return nil, fmt.Errorf("%s[%d] is required", field, index)
		}
		if _, duplicate := seen[value]; duplicate {
			return nil, fmt.Errorf("%s contains duplicate ID %q", field, value)
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	sort.Strings(result)
	if result == nil {
		result = []string{}
	}
	return result, nil
}

func validateSpecificationBodyIDs(body specificationBodySnapshot) error {
	seen := make(map[string]specificationBodySection)
	for _, section := range specificationBodyOrder {
		items := body[section]
		if len(items) == 0 {
			return fmt.Errorf("%s must contain typed stable-ID content", section)
		}
		for _, item := range items {
			if prior, duplicate := seen[item.ID]; duplicate {
				return fmt.Errorf("duplicate ID %q appears in %s and %s", item.ID, prior, section)
			}
			seen[item.ID] = section
		}
	}
	return nil
}

func validateSpecificationScopeBindings(scope colony.SpecScope, predecessor, successor specificationBodySnapshot) error {
	if scope.Kind != colony.SpecScopeFeature {
		return nil
	}
	knownRequirements := specificationUnionIDs(predecessor[specificationSectionRequirements], successor[specificationSectionRequirements])
	knownAcceptance := specificationUnionIDs(predecessor[specificationSectionAcceptanceChecks], successor[specificationSectionAcceptanceChecks])
	for _, id := range scope.RequirementIDs {
		if _, ok := knownRequirements[id]; !ok {
			return fmt.Errorf("feature scope requirement ID %q is absent from predecessor and successor", id)
		}
	}
	for _, id := range scope.AcceptanceCheckIDs {
		if _, ok := knownAcceptance[id]; !ok {
			return fmt.Errorf("feature scope acceptance ID %q is absent from predecessor and successor", id)
		}
	}
	return nil
}

func validateSpecificationFeatureDelta(scope colony.SpecScope, delta colony.SpecRevisionDelta) error {
	if scope.Kind != colony.SpecScopeFeature {
		return nil
	}
	requirements := specificationStringSet(scope.RequirementIDs)
	for _, id := range specificationChangedIDs(delta.Requirements) {
		if _, ok := requirements[id]; !ok {
			return fmt.Errorf("requirement %q is outside feature scope", id)
		}
	}
	acceptance := specificationStringSet(scope.AcceptanceCheckIDs)
	for _, id := range specificationChangedIDs(delta.AcceptanceChecks) {
		if _, ok := acceptance[id]; !ok {
			return fmt.Errorf("acceptance check %q is outside feature scope", id)
		}
	}
	return nil
}

func specificationUnionIDs(left, right []specificationCanonicalItem) map[string]struct{} {
	result := make(map[string]struct{}, len(left)+len(right))
	for _, item := range left {
		result[item.ID] = struct{}{}
	}
	for _, item := range right {
		result[item.ID] = struct{}{}
	}
	return result
}

func initialSpecificationDelta(body specificationBodySnapshot) colony.SpecRevisionDelta {
	delta := emptySpecificationDelta("")
	delta.Outcomes.AddedIDs = specificationItemIDs(body[specificationSectionOutcomes])
	delta.IncludedBehaviors.AddedIDs = specificationItemIDs(body[specificationSectionIncludedBehaviors])
	delta.Exclusions.AddedIDs = specificationItemIDs(body[specificationSectionExclusions])
	delta.BindingDecisions.AddedIDs = specificationItemIDs(body[specificationSectionBindingDecisions])
	delta.Requirements.AddedIDs = specificationItemIDs(body[specificationSectionRequirements])
	delta.AcceptanceChecks.AddedIDs = specificationItemIDs(body[specificationSectionAcceptanceChecks])
	delta.NegativeExpectations.AddedIDs = specificationItemIDs(body[specificationSectionNegativeExpectations])
	delta.RecoveryExpectations.AddedIDs = specificationItemIDs(body[specificationSectionRecoveryExpectations])
	delta.AffectedPublicPaths.AddedIDs = specificationItemIDs(body[specificationSectionAffectedPublicPaths])
	return delta
}

func emptySpecificationDelta(predecessorID string) colony.SpecRevisionDelta {
	empty := func() colony.SpecItemDelta {
		return colony.SpecItemDelta{AddedIDs: []string{}, ModifiedIDs: []string{}, RemovedIDs: []string{}, UnchangedIDs: []string{}}
	}
	return colony.SpecRevisionDelta{
		PredecessorRevisionID: predecessorID,
		Outcomes:              empty(), IncludedBehaviors: empty(), Exclusions: empty(), BindingDecisions: empty(),
		Requirements: empty(), AcceptanceChecks: empty(), NegativeExpectations: empty(),
		RecoveryExpectations: empty(), AffectedPublicPaths: empty(),
	}
}

func applySpecificationBody(revision *colony.SpecRevision, body specificationBodySnapshot) {
	revision.Outcomes = make([]colony.SpecOutcome, len(body[specificationSectionOutcomes]))
	for i, item := range body[specificationSectionOutcomes] {
		revision.Outcomes[i] = colony.SpecOutcome{ID: item.ID, Description: item.Description, ContentHash: item.ContentHash, EvidenceIDs: append([]string(nil), item.EvidenceIDs...)}
	}
	revision.IncludedBehaviors = make([]colony.SpecIncludedBehavior, len(body[specificationSectionIncludedBehaviors]))
	for i, item := range body[specificationSectionIncludedBehaviors] {
		revision.IncludedBehaviors[i] = colony.SpecIncludedBehavior{ID: item.ID, Description: item.Description, ContentHash: item.ContentHash, EvidenceIDs: append([]string(nil), item.EvidenceIDs...)}
	}
	revision.Exclusions = make([]colony.SpecExclusion, len(body[specificationSectionExclusions]))
	for i, item := range body[specificationSectionExclusions] {
		revision.Exclusions[i] = colony.SpecExclusion{ID: item.ID, Description: item.Description, ContentHash: item.ContentHash, EvidenceIDs: append([]string(nil), item.EvidenceIDs...)}
	}
	revision.BindingDecisions = make([]colony.SpecBindingDecision, len(body[specificationSectionBindingDecisions]))
	for i, item := range body[specificationSectionBindingDecisions] {
		revision.BindingDecisions[i] = colony.SpecBindingDecision{ID: item.ID, Description: item.Description, ContentHash: item.ContentHash, EvidenceIDs: append([]string(nil), item.EvidenceIDs...)}
	}
	revision.Requirements = make([]colony.SpecRequirement, len(body[specificationSectionRequirements]))
	for i, item := range body[specificationSectionRequirements] {
		revision.Requirements[i] = colony.SpecRequirement{ID: item.ID, Description: item.Description, ContentHash: item.ContentHash, EvidenceIDs: append([]string(nil), item.EvidenceIDs...)}
	}
	revision.AcceptanceChecks = make([]colony.SpecAcceptanceCheck, len(body[specificationSectionAcceptanceChecks]))
	for i, item := range body[specificationSectionAcceptanceChecks] {
		revision.AcceptanceChecks[i] = colony.SpecAcceptanceCheck{ID: item.ID, Description: item.Description, Verification: item.Verification, ContentHash: item.ContentHash, EvidenceIDs: append([]string(nil), item.EvidenceIDs...)}
	}
	revision.NegativeExpectations = make([]colony.SpecNegativeExpectation, len(body[specificationSectionNegativeExpectations]))
	for i, item := range body[specificationSectionNegativeExpectations] {
		revision.NegativeExpectations[i] = colony.SpecNegativeExpectation{ID: item.ID, Description: item.Description, ContentHash: item.ContentHash, EvidenceIDs: append([]string(nil), item.EvidenceIDs...)}
	}
	revision.RecoveryExpectations = make([]colony.SpecRecoveryExpectation, len(body[specificationSectionRecoveryExpectations]))
	for i, item := range body[specificationSectionRecoveryExpectations] {
		revision.RecoveryExpectations[i] = colony.SpecRecoveryExpectation{ID: item.ID, Description: item.Description, ContentHash: item.ContentHash, EvidenceIDs: append([]string(nil), item.EvidenceIDs...)}
	}
	revision.AffectedPublicPaths = make([]colony.SpecPublicPath, len(body[specificationSectionAffectedPublicPaths]))
	for i, item := range body[specificationSectionAffectedPublicPaths] {
		revision.AffectedPublicPaths[i] = colony.SpecPublicPath{ID: item.ID, Path: item.Path, Description: item.Description, ContentHash: item.ContentHash, EvidenceIDs: append([]string(nil), item.EvidenceIDs...)}
	}
}

func specificationBodyFromRevision(revision colony.SpecRevision) specificationBodySnapshot {
	body := make(specificationBodySnapshot, len(specificationBodyOrder))
	for _, value := range revision.Outcomes {
		body[specificationSectionOutcomes] = append(body[specificationSectionOutcomes], specificationCanonicalItem{ID: value.ID, Description: value.Description, ContentHash: value.ContentHash, EvidenceIDs: append([]string(nil), value.EvidenceIDs...)})
	}
	for _, value := range revision.IncludedBehaviors {
		body[specificationSectionIncludedBehaviors] = append(body[specificationSectionIncludedBehaviors], specificationCanonicalItem{ID: value.ID, Description: value.Description, ContentHash: value.ContentHash, EvidenceIDs: append([]string(nil), value.EvidenceIDs...)})
	}
	for _, value := range revision.Exclusions {
		body[specificationSectionExclusions] = append(body[specificationSectionExclusions], specificationCanonicalItem{ID: value.ID, Description: value.Description, ContentHash: value.ContentHash, EvidenceIDs: append([]string(nil), value.EvidenceIDs...)})
	}
	for _, value := range revision.BindingDecisions {
		body[specificationSectionBindingDecisions] = append(body[specificationSectionBindingDecisions], specificationCanonicalItem{ID: value.ID, Description: value.Description, ContentHash: value.ContentHash, EvidenceIDs: append([]string(nil), value.EvidenceIDs...)})
	}
	for _, value := range revision.Requirements {
		body[specificationSectionRequirements] = append(body[specificationSectionRequirements], specificationCanonicalItem{ID: value.ID, Description: value.Description, ContentHash: value.ContentHash, EvidenceIDs: append([]string(nil), value.EvidenceIDs...)})
	}
	for _, value := range revision.AcceptanceChecks {
		body[specificationSectionAcceptanceChecks] = append(body[specificationSectionAcceptanceChecks], specificationCanonicalItem{ID: value.ID, Description: value.Description, Verification: value.Verification, ContentHash: value.ContentHash, EvidenceIDs: append([]string(nil), value.EvidenceIDs...)})
	}
	for _, value := range revision.NegativeExpectations {
		body[specificationSectionNegativeExpectations] = append(body[specificationSectionNegativeExpectations], specificationCanonicalItem{ID: value.ID, Description: value.Description, ContentHash: value.ContentHash, EvidenceIDs: append([]string(nil), value.EvidenceIDs...)})
	}
	for _, value := range revision.RecoveryExpectations {
		body[specificationSectionRecoveryExpectations] = append(body[specificationSectionRecoveryExpectations], specificationCanonicalItem{ID: value.ID, Description: value.Description, ContentHash: value.ContentHash, EvidenceIDs: append([]string(nil), value.EvidenceIDs...)})
	}
	for _, value := range revision.AffectedPublicPaths {
		body[specificationSectionAffectedPublicPaths] = append(body[specificationSectionAffectedPublicPaths], specificationCanonicalItem{ID: value.ID, Description: value.Description, Path: value.Path, ContentHash: value.ContentHash, EvidenceIDs: append([]string(nil), value.EvidenceIDs...)})
	}
	return body
}

func cloneSpecificationBody(body specificationBodySnapshot) specificationBodySnapshot {
	cloned := make(specificationBodySnapshot, len(body))
	for _, section := range specificationBodyOrder {
		cloned[section] = make([]specificationCanonicalItem, len(body[section]))
		for index, item := range body[section] {
			cloned[section][index] = item
			cloned[section][index].EvidenceIDs = append([]string(nil), item.EvidenceIDs...)
		}
	}
	return cloned
}

func applySpecificationRevisionChange(body specificationBodySnapshot, change specificationRevisionChange) (string, error) {
	items := body[change.Section]
	switch change.Operation {
	case specificationChangeAdd:
		if strings.TrimSpace(change.TargetID) != "" {
			return "", fmt.Errorf("add derives its stable ID from semantic lineage; target ID must be empty")
		}
		id, err := specificationStableID(change.Section, change.Item.Lineage)
		if err != nil {
			return "", err
		}
		if specificationItemIndex(items, id) >= 0 || specificationBodyContainsID(body, id) {
			return "", fmt.Errorf("duplicate ID %q", id)
		}
		item, err := canonicalSpecificationItem(change.Section, id, change.Item)
		if err != nil {
			return "", err
		}
		body[change.Section] = append(items, item)
		sort.Slice(body[change.Section], func(i, j int) bool { return body[change.Section][i].ID < body[change.Section][j].ID })
		return id, nil
	case specificationChangeModify:
		id := strings.TrimSpace(change.TargetID)
		index := specificationItemIndex(items, id)
		if index < 0 {
			return "", fmt.Errorf("modify target %q is absent from %s", id, change.Section)
		}
		item, err := canonicalSpecificationItem(change.Section, id, change.Item)
		if err != nil {
			return "", err
		}
		if item.ContentHash == items[index].ContentHash {
			return "", fmt.Errorf("modify target %q has unchanged content", id)
		}
		body[change.Section][index] = item
		return id, nil
	case specificationChangeRemove:
		id := strings.TrimSpace(change.TargetID)
		if strings.TrimSpace(change.Item.Lineage) != "" || strings.TrimSpace(change.Item.Description) != "" || strings.TrimSpace(change.Item.Verification) != "" || strings.TrimSpace(change.Item.Path) != "" || len(change.Item.EvidenceIDs) != 0 {
			return "", fmt.Errorf("remove target %q cannot carry replacement content", id)
		}
		index := specificationItemIndex(items, id)
		if index < 0 {
			return "", fmt.Errorf("remove target %q is absent from %s", id, change.Section)
		}
		body[change.Section] = append(items[:index:index], items[index+1:]...)
		return id, nil
	default:
		return "", fmt.Errorf("invalid change operation %q", change.Operation)
	}
}

func specificationRevisionOperationKey(change specificationRevisionChange) (string, error) {
	if change.Operation == specificationChangeAdd {
		return specificationStableID(change.Section, change.Item.Lineage)
	}
	id := strings.TrimSpace(change.TargetID)
	if id == "" {
		return "", fmt.Errorf("%s target ID is required", change.Operation)
	}
	return id, nil
}

func specificationItemIndex(items []specificationCanonicalItem, id string) int {
	for index := range items {
		if items[index].ID == id {
			return index
		}
	}
	return -1
}

func specificationBodyContainsID(body specificationBodySnapshot, id string) bool {
	for _, section := range specificationBodyOrder {
		if specificationItemIndex(body[section], id) >= 0 {
			return true
		}
	}
	return false
}

func compareSpecificationBodies(predecessorID string, before, after specificationBodySnapshot) colony.SpecRevisionDelta {
	delta := emptySpecificationDelta(predecessorID)
	for _, section := range specificationBodyOrder {
		sectionDelta := compareSpecificationItems(before[section], after[section])
		switch section {
		case specificationSectionOutcomes:
			delta.Outcomes = sectionDelta
		case specificationSectionIncludedBehaviors:
			delta.IncludedBehaviors = sectionDelta
		case specificationSectionExclusions:
			delta.Exclusions = sectionDelta
		case specificationSectionBindingDecisions:
			delta.BindingDecisions = sectionDelta
		case specificationSectionRequirements:
			delta.Requirements = sectionDelta
		case specificationSectionAcceptanceChecks:
			delta.AcceptanceChecks = sectionDelta
		case specificationSectionNegativeExpectations:
			delta.NegativeExpectations = sectionDelta
		case specificationSectionRecoveryExpectations:
			delta.RecoveryExpectations = sectionDelta
		case specificationSectionAffectedPublicPaths:
			delta.AffectedPublicPaths = sectionDelta
		}
	}
	return delta
}

func compareSpecificationItems(before, after []specificationCanonicalItem) colony.SpecItemDelta {
	delta := colony.SpecItemDelta{AddedIDs: []string{}, ModifiedIDs: []string{}, RemovedIDs: []string{}, UnchangedIDs: []string{}}
	beforeByID := make(map[string]specificationCanonicalItem, len(before))
	afterByID := make(map[string]specificationCanonicalItem, len(after))
	for _, item := range before {
		beforeByID[item.ID] = item
	}
	for _, item := range after {
		afterByID[item.ID] = item
	}
	for id, item := range beforeByID {
		next, exists := afterByID[id]
		if !exists {
			delta.RemovedIDs = append(delta.RemovedIDs, id)
		} else if item.ContentHash != next.ContentHash {
			delta.ModifiedIDs = append(delta.ModifiedIDs, id)
		} else {
			delta.UnchangedIDs = append(delta.UnchangedIDs, id)
		}
	}
	for id := range afterByID {
		if _, exists := beforeByID[id]; !exists {
			delta.AddedIDs = append(delta.AddedIDs, id)
		}
	}
	sort.Strings(delta.AddedIDs)
	sort.Strings(delta.ModifiedIDs)
	sort.Strings(delta.RemovedIDs)
	sort.Strings(delta.UnchangedIDs)
	return delta
}

func addressSpecificationRevision(revision *colony.SpecRevision) error {
	revision.ID = ""
	revision.ContentHash = ""
	hash, err := specificationRevisionContentHash(*revision)
	if err != nil {
		return err
	}
	revision.ContentHash = hash
	revision.ID = "spec-revision-" + hash[:12]
	return nil
}

// specificationRevisionContentHash excludes observation time and authority
// status. Approval and supersession therefore cannot masquerade as a material
// specification edit, while scope, typed content, evidence, and delta do.
func specificationRevisionContentHash(revision colony.SpecRevision) (string, error) {
	return jsonSHA256(struct {
		SchemaVersion        string                           `json:"schema_version"`
		SpecificationID      string                           `json:"specification_id"`
		PredecessorID        string                           `json:"predecessor_id"`
		Scope                colony.SpecScope                 `json:"scope"`
		Outcomes             []colony.SpecOutcome             `json:"outcomes"`
		IncludedBehaviors    []colony.SpecIncludedBehavior    `json:"included_behaviors"`
		Exclusions           []colony.SpecExclusion           `json:"exclusions"`
		BindingDecisions     []colony.SpecBindingDecision     `json:"binding_decisions"`
		Requirements         []colony.SpecRequirement         `json:"requirements"`
		AcceptanceChecks     []colony.SpecAcceptanceCheck     `json:"acceptance_checks"`
		NegativeExpectations []colony.SpecNegativeExpectation `json:"negative_expectations"`
		RecoveryExpectations []colony.SpecRecoveryExpectation `json:"recovery_expectations"`
		AffectedPublicPaths  []colony.SpecPublicPath          `json:"affected_public_paths"`
		Delta                colony.SpecRevisionDelta         `json:"delta"`
	}{
		revision.SchemaVersion, revision.SpecificationID, revision.PredecessorID, revision.Scope,
		revision.Outcomes, revision.IncludedBehaviors, revision.Exclusions, revision.BindingDecisions,
		revision.Requirements, revision.AcceptanceChecks, revision.NegativeExpectations,
		revision.RecoveryExpectations, revision.AffectedPublicPaths, revision.Delta,
	})
}

func affectedSpecificationScope(delta colony.SpecRevisionDelta, plan colony.Plan) specificationAffectedScope {
	allChanged := specificationDeltaChangedIDs(delta)
	requirementIDs := specificationChangedIDs(delta.Requirements)
	proofCandidates := append([]string(nil), requirementIDs...)
	proofCandidates = append(proofCandidates, specificationChangedIDs(delta.AcceptanceChecks)...)
	proofCandidates = append(proofCandidates, specificationChangedIDs(delta.NegativeExpectations)...)
	proofCandidates = append(proofCandidates, specificationChangedIDs(delta.RecoveryExpectations)...)
	proofCandidates = append(proofCandidates, specificationChangedIDs(delta.AffectedPublicPaths)...)
	proofSet := specificationStringSet(proofCandidates)
	changedSet := specificationStringSet(allChanged)

	proofLinks := make(map[string]struct{})
	taskIDs := make(map[string]struct{})
	for _, phase := range plan.Phases {
		phaseLinked := specificationCollectIntersectingLinks(proofLinks, proofSet,
			phase.RequirementProofLinks, phase.AcceptanceProofLinks, phase.NegativeProofLinks,
			phase.RecoveryProofLinks, phase.PublicPathProofLinks, phase.AffectedSemanticIDs)
		for _, task := range phase.Tasks {
			taskLinked := specificationCollectIntersectingLinks(proofLinks, proofSet,
				task.RequirementProofLinks, task.AcceptanceProofLinks, task.NegativeProofLinks,
				task.RecoveryProofLinks, task.PublicPathProofLinks)
			for _, id := range task.AffectedSemanticIDs {
				if _, affected := changedSet[id]; affected {
					taskLinked = true
				}
			}
			if phaseLinked || taskLinked {
				id := strings.TrimSpace(task.SemanticID)
				if id == "" && task.ID != nil {
					id = strings.TrimSpace(*task.ID)
				}
				if id != "" {
					taskIDs[id] = struct{}{}
				}
			}
		}
	}
	return specificationAffectedScope{
		SpecItemIDs:    sortedSpecificationSet(specificationStringSet(allChanged)),
		RequirementIDs: sortedSpecificationSet(specificationStringSet(requirementIDs)),
		TaskIDs:        sortedSpecificationSet(taskIDs),
		ProofLinkIDs:   sortedSpecificationSet(proofLinks),
	}
}

func specificationCollectIntersectingLinks(destination, wanted map[string]struct{}, groups ...[]string) bool {
	found := false
	for _, group := range groups {
		for _, id := range group {
			if _, ok := wanted[id]; ok {
				destination[id] = struct{}{}
				found = true
			}
		}
	}
	return found
}

func specificationDeltaChangedIDs(delta colony.SpecRevisionDelta) []string {
	result := []string{}
	for _, section := range []colony.SpecItemDelta{
		delta.Outcomes, delta.IncludedBehaviors, delta.Exclusions, delta.BindingDecisions,
		delta.Requirements, delta.AcceptanceChecks, delta.NegativeExpectations,
		delta.RecoveryExpectations, delta.AffectedPublicPaths,
	} {
		result = append(result, specificationChangedIDs(section)...)
	}
	return result
}

func specificationChangedIDs(delta colony.SpecItemDelta) []string {
	result := append([]string(nil), delta.AddedIDs...)
	result = append(result, delta.ModifiedIDs...)
	result = append(result, delta.RemovedIDs...)
	sort.Strings(result)
	return result
}

func specificationStringSet(values []string) map[string]struct{} {
	result := make(map[string]struct{}, len(values))
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			result[value] = struct{}{}
		}
	}
	return result
}

func sortedSpecificationSet(values map[string]struct{}) []string {
	result := make([]string, 0, len(values))
	for value := range values {
		result = append(result, value)
	}
	sort.Strings(result)
	if result == nil {
		return []string{}
	}
	return result
}

func validateSpecificationDecisionResolution(resolution planningDecisionResolution) error {
	if resolution.Disposition != planningDecisionDispositionSuccessorSpecRequired {
		return fmt.Errorf("specification revision decision disposition must be successor_spec_required")
	}
	if len(resolution.AffectedSemanticIDs) == 0 || len(resolution.RevisionEvidence) == 0 {
		return fmt.Errorf("successor_spec_required requires affected IDs and revision evidence")
	}
	if _, err := canonicalSpecificationIDs("affected_semantic_ids", resolution.AffectedSemanticIDs, true); err != nil {
		return err
	}
	for index, evidence := range resolution.RevisionEvidence {
		if strings.TrimSpace(evidence.Dimension) == "" {
			return fmt.Errorf("revision_evidence[%d] requires dimension", index)
		}
		if strings.TrimSpace(evidence.SelectedValue) == "" && len(evidence.AffectedSemanticIDs) == 0 {
			return fmt.Errorf("revision_evidence[%d] requires selected value or affected IDs", index)
		}
	}
	return nil
}

func validateSpecificationDecisionAffectedIDs(resolution planningDecisionResolution, changed []string) error {
	changedSet := specificationStringSet(changed)
	for _, id := range resolution.AffectedSemanticIDs {
		if _, ok := changedSet[id]; !ok {
			return fmt.Errorf("successor_spec_required affected ID %q is not changed by this revision", id)
		}
	}
	return nil
}

func specificationItemIDs(items []specificationCanonicalItem) []string {
	result := make([]string, len(items))
	for index := range items {
		result[index] = items[index].ID
	}
	sort.Strings(result)
	return result
}

func specificationRevisionIndex(specification colony.Specification, id string) int {
	for index := range specification.Revisions {
		if specification.Revisions[index].ID == id {
			return index
		}
	}
	return -1
}

func specificationAtPredecessor(specification colony.Specification, index int) (colony.Specification, error) {
	if index < 0 || index >= len(specification.Revisions) {
		return colony.Specification{}, fmt.Errorf("specification predecessor index is invalid")
	}
	result, err := cloneSpecification(specification)
	if err != nil {
		return colony.Specification{}, err
	}
	result.Revisions = append([]colony.SpecRevision(nil), result.Revisions[:index+1]...)
	result.CurrentRevisionID = result.Revisions[index].ID
	if result.Revisions[index].Approval != nil {
		result.Revisions[index].Status = colony.SpecStatusApproved
	} else {
		result.Revisions[index].Status = colony.SpecStatusDraft
	}
	if err := validateSpecificationState(result); err != nil {
		return colony.Specification{}, fmt.Errorf("reconstruct predecessor specification: %w", err)
	}
	return result, nil
}

func cloneSpecification(specification colony.Specification) (colony.Specification, error) {
	content, err := json.Marshal(specification)
	if err != nil {
		return colony.Specification{}, fmt.Errorf("clone specification: %w", err)
	}
	var cloned colony.Specification
	if err := json.Unmarshal(content, &cloned); err != nil {
		return colony.Specification{}, fmt.Errorf("clone specification: %w", err)
	}
	return cloned, nil
}

func canonicalSpecificationRoot(root string) (string, error) {
	if strings.TrimSpace(root) == "" {
		return "", fmt.Errorf("specification repository root is required")
	}
	absolute, err := filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("resolve specification repository root: %w", err)
	}
	absolute = filepath.Clean(absolute)
	info, err := os.Lstat(absolute)
	if err != nil {
		return "", fmt.Errorf("inspect specification repository root: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return "", fmt.Errorf("specification repository root must be a real directory")
	}
	dataRoot := filepath.Join(absolute, ".aether", "data")
	dataInfo, err := os.Lstat(dataRoot)
	if err != nil {
		return "", fmt.Errorf("inspect specification lifecycle data root: %w", err)
	}
	if dataInfo.Mode()&os.ModeSymlink != 0 || !dataInfo.IsDir() {
		return "", fmt.Errorf("specification lifecycle data root must be a real directory")
	}
	return absolute, nil
}

func loadSpecificationColonyState(root string) (colony.ColonyState, error) {
	path := filepath.Join(root, ".aether", "data", "COLONY_STATE.json")
	state, _, err := loadColonyStateWithCompatibilityRepairReadOnlyFromPath(path)
	if err != nil {
		return colony.ColonyState{}, fmt.Errorf("load specification state: %w", err)
	}
	state = normalizeLegacyColonyState(state)
	migration, err := migratePlanningState(root, state)
	if err != nil {
		return colony.ColonyState{}, fmt.Errorf("load specification planning state: %w", err)
	}
	return migration.State, nil
}

func marshalSpecificationState(state colony.ColonyState) ([]byte, error) {
	content, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal specification state: %w", err)
	}
	return append(content, '\n'), nil
}

func specificationTransactionID(operation, contentHash string) string {
	return "spec-" + operation + "-" + contentHash[:16]
}

func commitSpecificationTargets(root, transactionID, command string, targets []specificationTransactionTarget, opts specificationMutationOptions) (colony.LifecycleReceipt, error) {
	config := lifecycleTransactionConfig{
		TransactionID: transactionID,
		Command:       command,
		Allowlist: lifecycleTransactionAllowlist{
			RepositoryRoot:    root,
			LifecycleDataRoot: filepath.Join(root, ".aether", "data"),
		},
		Fault:  opts.Fault,
		Rename: opts.Rename,
	}
	pending, err := specificationTransactionHasIntent(config)
	if err != nil {
		return colony.LifecycleReceipt{}, err
	}
	if pending {
		matches, matchErr := specificationPendingIntentMatches(config, targets)
		if matchErr != nil {
			return colony.LifecycleReceipt{}, matchErr
		}
		if !matches {
			return colony.LifecycleReceipt{}, fmt.Errorf("specification transaction %q has divergent staged content", transactionID)
		}
		return resumeLifecycleTransaction(config)
	}
	tx, err := beginLifecycleTransaction(config)
	if err != nil {
		return colony.LifecycleReceipt{}, err
	}
	for _, target := range targets {
		if err := tx.DeclareWrite(target.Root, target.Path, target.Content); err != nil {
			return colony.LifecycleReceipt{}, err
		}
	}
	if err := tx.Validate(); err != nil {
		return colony.LifecycleReceipt{}, err
	}
	return tx.Commit()
}

func specificationTransactionHasIntent(config lifecycleTransactionConfig) (bool, error) {
	tx, err := beginLifecycleTransaction(config)
	if err != nil {
		return false, err
	}
	_, err = os.Lstat(filepath.Join(tx.journalPath(), "intent.json"))
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("inspect specification transaction intent: %w", err)
	}
	return true, nil
}

func specificationPendingIntentMatches(config lifecycleTransactionConfig, expected []specificationTransactionTarget) (bool, error) {
	tx, err := beginLifecycleTransaction(config)
	if err != nil {
		return false, err
	}
	intent, progress, err := tx.loadJournal()
	if err != nil {
		return false, err
	}
	tx.intent, tx.progress = intent, progress
	manifests, err := tx.validateRecoveryEvidence()
	if err != nil {
		return false, err
	}
	targets := flattenLifecycleManifestTargets(intent, manifests)
	if len(targets) != len(expected) {
		return false, nil
	}
	wanted := make(map[string]specificationTransactionTarget, len(expected))
	for _, target := range expected {
		root, ok := tx.roots[target.Root]
		if !ok {
			return false, fmt.Errorf("specification transaction expected unconfigured root %q", target.Root)
		}
		absolute := filepath.Join(root.Path, filepath.Clean(target.Path))
		if _, duplicate := wanted[absolute]; duplicate {
			return false, fmt.Errorf("specification transaction expected duplicate target %q", absolute)
		}
		wanted[absolute] = target
	}
	for _, staged := range targets {
		expectedTarget, ok := wanted[staged.TargetPath]
		if !ok || staged.Action != lifecycleTransactionWrite || filepath.Clean(expectedTarget.Path) != expectedTarget.Path || staged.AfterDigest != lifecycleDigest(expectedTarget.Content) {
			return false, nil
		}
		delete(wanted, staged.TargetPath)
	}
	return len(wanted) == 0, nil
}
