package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/spf13/cobra"
)

const specCommandSchemaVersion = "spec-command/v1"

type specCommandOperation string

const (
	specCommandOperationInspect          specCommandOperation = "inspect"
	specCommandOperationAdd              specCommandOperation = "add"
	specCommandOperationModify           specCommandOperation = "modify"
	specCommandOperationRemove           specCommandOperation = "remove"
	specCommandOperationApprove          specCommandOperation = "approve"
	specCommandOperationProjectionRepair specCommandOperation = "projection_repair"
)

type specCommandResultKind string

const (
	specCommandResultDraftCreation    specCommandResultKind = "draft_creation"
	specCommandResultInspection       specCommandResultKind = "inspection"
	specCommandResultRevisionCreation specCommandResultKind = "revision_creation"
	specCommandResultApproval         specCommandResultKind = "approval"
	specCommandResultProjectionRepair specCommandResultKind = "projection_repair"
)

// specCommandProjection is the stable machine view of the readable projection.
// The Markdown file is never parsed back into any authority decision.
type specCommandProjection struct {
	Path           string `json:"path"`
	RevisionID     string `json:"revision_id"`
	ExpectedDigest string `json:"expected_digest"`
	ActualDigest   string `json:"actual_digest,omitempty"`
	Missing        bool   `json:"missing"`
	Drifted        bool   `json:"drifted"`
}

// specCommandResult is shared by visual and JSON rendering. The nine body
// categories remain separate typed fields on purpose: a generic items field
// would erase which owner contract each stable ID belongs to.
type specCommandResult struct {
	SchemaVersion        string                           `json:"schema_version"`
	Command              string                           `json:"command"`
	Operation            specCommandOperation             `json:"operation"`
	ResultKind           specCommandResultKind            `json:"result_kind"`
	OutcomeKind          colony.OutcomeKind               `json:"outcome_kind"`
	StateEffect          colony.LifecycleStateEffect      `json:"state_effect"`
	SpecificationID      string                           `json:"specification_id"`
	BeforeRevisionID     string                           `json:"before_revision_id"`
	AfterRevisionID      string                           `json:"after_revision_id"`
	RevisionNumber       int                              `json:"revision_number"`
	ContentHash          string                           `json:"content_hash"`
	Status               colony.SpecRevisionStatus        `json:"status"`
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
	ClassifiedDelta      colony.SpecRevisionDelta         `json:"classified_delta"`
	AffectedScope        specificationAffectedScope       `json:"affected_scope"`
	Approval             *colony.SpecApprovalReceipt      `json:"approval,omitempty"`
	Receipt              *colony.LifecycleReceipt         `json:"receipt,omitempty"`
	Replayed             bool                             `json:"replayed"`
	Projection           specCommandProjection            `json:"projection"`
	ProjectionRepaired   bool                             `json:"projection_repaired"`
	NextAction           string                           `json:"next_action"`
}

type specCommandOptions struct {
	Inspect          bool
	Add              bool
	Modify           bool
	Remove           bool
	Approve          bool
	RepairProjection bool

	Section      string
	ItemID       string
	Lineage      string
	Text         string
	File         string
	Verification string
	PublicPath   string
	EvidenceIDs  []string

	PredecessorRevision string
	PredecessorHash     string
	ScopeKind           string
	SessionID           string
	FeatureID           string
	RequirementIDs      []string
	AcceptanceIDs       []string

	RevisionID    string
	RevisionHash  string
	ApprovalToken string
	ApprovedBy    string
}

var specCmd = newSpecCommand()

func newSpecCommand() *cobra.Command {
	options := &specCommandOptions{}
	command := &cobra.Command{
		Use:   "spec",
		Short: "Inspect, revise, approve, or repair the owner-readable specification",
		Long: "Inspect the canonical specification or create an immutable add, modify, or remove revision. " +
			"Approval requires --approve with the exact --revision-id, --revision-hash, and --approval-token shown by inspection. " +
			"Specification approval never accepts or activates a plan.",
		Args: func(cmd *cobra.Command, args []string) error {
			if err := cobra.NoArgs(cmd, args); err != nil {
				return err
			}
			_, err := validateSpecCommandOptions(*options)
			return err
		},
		Annotations: map[string]string{
			// The command owns its repository transaction directly and does not
			// need PersistentPreRunE to create or open the shared Store. This also
			// lets Args reject contradictory operations before any state access.
			"aether.io/store-free": "true",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			operation, err := validateSpecCommandOptions(*options)
			if err != nil {
				return err
			}
			result, err := runSpecCommand(resolveAetherRootPath(), operation, *options)
			if err != nil {
				outputError(1, err.Error(), nil)
				return nil
			}
			outputWorkflow(result, renderSpecCommandVisual(result))
			return nil
		},
	}

	flags := command.Flags()
	flags.BoolVar(&options.Inspect, "inspect", false, "Inspect the current canonical specification without changing state (also the default action)")
	flags.BoolVar(&options.Add, "add", false, "Add one typed item by creating an immutable successor revision")
	flags.BoolVar(&options.Modify, "modify", false, "Modify one stable typed item by creating an immutable successor revision")
	flags.BoolVar(&options.Remove, "remove", false, "Remove one stable typed item from the current body while retaining its history")
	flags.BoolVar(&options.Approve, "approve", false, "Approve exactly the revision named by --revision-id and --revision-hash using --approval-token")
	flags.BoolVar(&options.RepairProjection, "repair-projection", false, "Regenerate .aether/SPEC.md from canonical state without changing specification authority")

	flags.StringVar(&options.Section, "section", "", "Typed body section for --add, --modify, or --remove")
	flags.StringVar(&options.ItemID, "item-id", "", "Stable item ID required by --modify or --remove")
	flags.StringVar(&options.Lineage, "lineage", "", "Stable semantic lineage required by --add")
	flags.StringVar(&options.Text, "text", "", "Owner-readable source text for --add or --modify (mutually exclusive with --file)")
	flags.StringVar(&options.File, "file", "", "File containing owner-readable source text for --add or --modify (mutually exclusive with --text)")
	flags.StringVar(&options.Verification, "verification", "", "Owner-checkable verification text for an acceptance_checks item")
	flags.StringVar(&options.PublicPath, "path", "", "Affected public path for an affected_public_paths item")
	flags.StringArrayVar(&options.EvidenceIDs, "evidence", nil, "Stable evidence ID supporting an added or modified item (repeatable)")

	flags.StringVar(&options.PredecessorRevision, "predecessor-revision", "", "Exact predecessor revision ID required by --add, --modify, and --remove")
	flags.StringVar(&options.PredecessorHash, "predecessor-hash", "", "Optional exact predecessor content hash; a mismatch is refused as stale")
	flags.StringVar(&options.ScopeKind, "scope", "", "Successor scope: whole_goal or feature (default: predecessor scope)")
	flags.StringVar(&options.SessionID, "session-id", "", "Session identity for the successor scope (default: current state or predecessor session)")
	flags.StringVar(&options.FeatureID, "feature-id", "", "Feature identity required by --scope feature")
	flags.StringArrayVar(&options.RequirementIDs, "scope-requirement", nil, "Requirement ID inside a feature revision scope (repeatable)")
	flags.StringArrayVar(&options.AcceptanceIDs, "scope-acceptance", nil, "Acceptance-check ID inside a feature revision scope (repeatable)")

	flags.StringVar(&options.RevisionID, "revision-id", "", "Exact current draft revision ID required by --approve")
	flags.StringVar(&options.RevisionHash, "revision-hash", "", "Exact current draft content hash required by --approve")
	flags.StringVar(&options.ApprovalToken, "approval-token", "", "Explicit action token binding approval to the exact specification, revision, and hash")
	flags.StringVar(&options.ApprovedBy, "approved-by", "", "Owner identity recorded in the approval receipt (default: owner)")

	return command
}

func validateSpecCommandOptions(options specCommandOptions) (specCommandOperation, error) {
	selected := []struct {
		enabled   bool
		operation specCommandOperation
	}{
		{options.Inspect, specCommandOperationInspect},
		{options.Add, specCommandOperationAdd},
		{options.Modify, specCommandOperationModify},
		{options.Remove, specCommandOperationRemove},
		{options.Approve, specCommandOperationApprove},
		{options.RepairProjection, specCommandOperationProjectionRepair},
	}
	operation := specCommandOperationInspect
	count := 0
	for _, candidate := range selected {
		if candidate.enabled {
			count++
			operation = candidate.operation
		}
	}
	if count > 1 {
		return "", fmt.Errorf("--inspect, --add, --modify, --remove, --approve, and --repair-projection are mutually exclusive")
	}

	section := specificationBodySection(strings.TrimSpace(options.Section))
	hasSource := strings.TrimSpace(options.Text) != ""
	hasFile := strings.TrimSpace(options.File) != ""
	mutation := operation == specCommandOperationAdd || operation == specCommandOperationModify || operation == specCommandOperationRemove

	switch operation {
	case specCommandOperationInspect:
		if specCommandHasRevisionInputs(options) {
			return "", fmt.Errorf("inspection accepts no mutation, approval, or projection-repair inputs")
		}
	case specCommandOperationAdd, specCommandOperationModify:
		if !section.valid() {
			return "", fmt.Errorf("--section must be one of %s", specCommandSectionList())
		}
		if strings.TrimSpace(options.PredecessorRevision) == "" {
			return "", fmt.Errorf("--predecessor-revision is required for --%s", operation)
		}
		if hasSource == hasFile {
			return "", fmt.Errorf("--%s requires exactly one of --text or --file", operation)
		}
		if len(normalizeCLIStringList(options.EvidenceIDs)) == 0 {
			return "", fmt.Errorf("--%s requires at least one --evidence ID", operation)
		}
		if operation == specCommandOperationAdd {
			if strings.TrimSpace(options.Lineage) == "" {
				return "", fmt.Errorf("--lineage is required for --add")
			}
			if strings.TrimSpace(options.ItemID) != "" {
				return "", fmt.Errorf("--add derives its stable ID from --lineage and cannot accept --item-id")
			}
		} else if strings.TrimSpace(options.ItemID) == "" {
			return "", fmt.Errorf("--item-id is required for --modify")
		}
		if err := validateSpecCommandSectionFields(section, options); err != nil {
			return "", err
		}
	case specCommandOperationRemove:
		if !section.valid() {
			return "", fmt.Errorf("--section must be one of %s", specCommandSectionList())
		}
		if strings.TrimSpace(options.ItemID) == "" {
			return "", fmt.Errorf("--item-id is required for --remove")
		}
		if strings.TrimSpace(options.PredecessorRevision) == "" {
			return "", fmt.Errorf("--predecessor-revision is required for --remove")
		}
		if hasSource || hasFile || strings.TrimSpace(options.Lineage) != "" || strings.TrimSpace(options.Verification) != "" || strings.TrimSpace(options.PublicPath) != "" || len(options.EvidenceIDs) > 0 {
			return "", fmt.Errorf("--remove accepts only --section, --item-id, predecessor, and scope inputs")
		}
	case specCommandOperationApprove:
		if strings.TrimSpace(options.RevisionID) == "" || strings.TrimSpace(options.RevisionHash) == "" || strings.TrimSpace(options.ApprovalToken) == "" {
			return "", fmt.Errorf("--approve requires exact --revision-id, --revision-hash, and --approval-token inputs")
		}
		if specCommandHasMutationInputs(options) {
			return "", fmt.Errorf("--approve cannot be combined with revision or projection-repair inputs")
		}
	case specCommandOperationProjectionRepair:
		if specCommandHasMutationInputs(options) || strings.TrimSpace(options.RevisionID) != "" || strings.TrimSpace(options.RevisionHash) != "" || strings.TrimSpace(options.ApprovalToken) != "" || strings.TrimSpace(options.ApprovedBy) != "" {
			return "", fmt.Errorf("--repair-projection cannot be combined with revision or approval inputs")
		}
	}

	if mutation {
		if err := validateSpecCommandScopeShape(options); err != nil {
			return "", err
		}
	}
	return operation, nil
}

func specCommandHasRevisionInputs(options specCommandOptions) bool {
	return specCommandHasMutationInputs(options) || strings.TrimSpace(options.RevisionID) != "" ||
		strings.TrimSpace(options.RevisionHash) != "" || strings.TrimSpace(options.ApprovalToken) != "" ||
		strings.TrimSpace(options.ApprovedBy) != ""
}

func specCommandHasMutationInputs(options specCommandOptions) bool {
	return strings.TrimSpace(options.Section) != "" || strings.TrimSpace(options.ItemID) != "" ||
		strings.TrimSpace(options.Lineage) != "" || strings.TrimSpace(options.Text) != "" || strings.TrimSpace(options.File) != "" ||
		strings.TrimSpace(options.Verification) != "" || strings.TrimSpace(options.PublicPath) != "" || len(options.EvidenceIDs) > 0 ||
		strings.TrimSpace(options.PredecessorRevision) != "" || strings.TrimSpace(options.PredecessorHash) != "" ||
		strings.TrimSpace(options.ScopeKind) != "" || strings.TrimSpace(options.SessionID) != "" || strings.TrimSpace(options.FeatureID) != "" ||
		len(options.RequirementIDs) > 0 || len(options.AcceptanceIDs) > 0
}

func validateSpecCommandSectionFields(section specificationBodySection, options specCommandOptions) error {
	verification := strings.TrimSpace(options.Verification)
	publicPath := strings.TrimSpace(options.PublicPath)
	switch section {
	case specificationSectionAcceptanceChecks:
		if verification == "" {
			return fmt.Errorf("--verification is required for acceptance_checks")
		}
		if publicPath != "" {
			return fmt.Errorf("--path is only valid for affected_public_paths")
		}
	case specificationSectionAffectedPublicPaths:
		if publicPath == "" {
			return fmt.Errorf("--path is required for affected_public_paths")
		}
		if verification != "" {
			return fmt.Errorf("--verification is only valid for acceptance_checks")
		}
	default:
		if verification != "" || publicPath != "" {
			return fmt.Errorf("section %s accepts source text and evidence only", section)
		}
	}
	return nil
}

func validateSpecCommandScopeShape(options specCommandOptions) error {
	kind := strings.TrimSpace(options.ScopeKind)
	if kind == "" {
		if strings.TrimSpace(options.FeatureID) != "" || len(options.RequirementIDs) > 0 || len(options.AcceptanceIDs) > 0 {
			return fmt.Errorf("--feature-id and feature scope IDs require --scope feature")
		}
		return nil
	}
	if kind != string(colony.SpecScopeWholeGoal) && kind != string(colony.SpecScopeFeature) {
		return fmt.Errorf("--scope must be whole_goal or feature")
	}
	if kind == string(colony.SpecScopeWholeGoal) {
		if strings.TrimSpace(options.FeatureID) != "" || len(options.RequirementIDs) > 0 || len(options.AcceptanceIDs) > 0 {
			return fmt.Errorf("--scope whole_goal cannot contain feature identifiers")
		}
		return nil
	}
	if strings.TrimSpace(options.FeatureID) == "" {
		return fmt.Errorf("--feature-id is required for --scope feature")
	}
	if len(normalizeCLIStringList(options.RequirementIDs)) == 0 && len(normalizeCLIStringList(options.AcceptanceIDs)) == 0 {
		return fmt.Errorf("--scope feature requires --scope-requirement or --scope-acceptance")
	}
	return nil
}

func specCommandSectionList() string {
	values := make([]string, len(specificationBodyOrder))
	for index, section := range specificationBodyOrder {
		values[index] = string(section)
	}
	return strings.Join(values, ", ")
}

func runSpecCommand(root string, operation specCommandOperation, options specCommandOptions) (specCommandResult, error) {
	switch operation {
	case specCommandOperationInspect:
		return inspectSpecCommand(root)
	case specCommandOperationAdd, specCommandOperationModify, specCommandOperationRemove:
		return reviseSpecCommand(root, operation, options)
	case specCommandOperationApprove:
		return approveSpecCommand(root, options)
	case specCommandOperationProjectionRepair:
		return repairSpecCommandProjection(root)
	default:
		return specCommandResult{}, fmt.Errorf("unsupported specification operation %q", operation)
	}
}

func inspectSpecCommand(root string) (specCommandResult, error) {
	repositoryRoot, state, specification, current, err := loadSpecCommandState(root)
	if err != nil {
		return specCommandResult{}, err
	}
	projection, err := inspectSpecificationProjection(repositoryRoot)
	if err != nil {
		return specCommandResult{}, err
	}
	return buildSpecCommandResult(
		specCommandOperationInspect,
		specCommandResultInspection,
		colony.OutcomeKindNoChange,
		colony.LifecycleStateEffectNone,
		specification,
		current.ID,
		current,
		affectedSpecificationScope(current.Delta, state.Plan),
		nil,
		false,
		projection,
		false,
	), nil
}

func reviseSpecCommand(root string, operation specCommandOperation, options specCommandOptions) (specCommandResult, error) {
	repositoryRoot, state, specification, before, err := loadSpecCommandState(root)
	if err != nil {
		return specCommandResult{}, err
	}
	predecessor, ok := specificationRevisionByID(specification, strings.TrimSpace(options.PredecessorRevision))
	if !ok {
		return specCommandResult{}, fmt.Errorf("stale predecessor: revision %q does not exist", strings.TrimSpace(options.PredecessorRevision))
	}
	if suppliedHash := strings.TrimSpace(options.PredecessorHash); suppliedHash != "" && suppliedHash != predecessor.ContentHash {
		return specCommandResult{}, fmt.Errorf("stale predecessor: revision %q content hash changed", predecessor.ID)
	}
	scope, err := specCommandSuccessorScope(state, specification, predecessor, options)
	if err != nil {
		return specCommandResult{}, err
	}
	change := specificationRevisionChange{
		Operation: specificationChangeOperation(operation),
		Section:   specificationBodySection(strings.TrimSpace(options.Section)),
		TargetID:  strings.TrimSpace(options.ItemID),
	}
	if operation != specCommandOperationRemove {
		description, sourceErr := specCommandSourceText(repositoryRoot, options.Text, options.File)
		if sourceErr != nil {
			return specCommandResult{}, sourceErr
		}
		change.Item = specificationItemInput{
			Lineage:      strings.TrimSpace(options.Lineage),
			Description:  description,
			Verification: strings.TrimSpace(options.Verification),
			Path:         strings.TrimSpace(options.PublicPath),
			EvidenceIDs:  normalizeCLIStringList(options.EvidenceIDs),
		}
	}
	mutation, err := reviseSpecification(repositoryRoot, specificationRevisionRequest{
		PredecessorRevisionID:  predecessor.ID,
		PredecessorContentHash: predecessor.ContentHash,
		Scope:                  scope,
		Changes:                []specificationRevisionChange{change},
		CreatedAt:              time.Now().UTC(),
	}, specificationMutationOptions{})
	if err != nil {
		return specCommandResult{}, err
	}
	projection, err := inspectSpecificationProjection(repositoryRoot)
	if err != nil {
		return specCommandResult{}, err
	}
	return buildSpecCommandResult(
		operation,
		specCommandResultRevisionCreation,
		colony.OutcomeKindCompleted,
		mutation.Receipt.StateEffect,
		mutation.Specification,
		before.ID,
		mutation.Revision,
		mutation.AffectedScope,
		&mutation.Receipt,
		mutation.Replayed,
		projection,
		false,
	), nil
}

func approveSpecCommand(root string, options specCommandOptions) (specCommandResult, error) {
	repositoryRoot, _, _, before, err := loadSpecCommandState(root)
	if err != nil {
		return specCommandResult{}, err
	}
	approvedBy := strings.TrimSpace(options.ApprovedBy)
	if approvedBy == "" {
		approvedBy = "owner"
	}
	mutation, err := approveSpecification(repositoryRoot, specificationApprovalRequest{
		RevisionID:          strings.TrimSpace(options.RevisionID),
		RevisionContentHash: strings.TrimSpace(options.RevisionHash),
		ApprovalToken:       strings.TrimSpace(options.ApprovalToken),
		ApprovedBy:          approvedBy,
		ApprovedAt:          time.Now().UTC(),
	}, specificationMutationOptions{})
	if err != nil {
		return specCommandResult{}, err
	}
	projection, err := inspectSpecificationProjection(repositoryRoot)
	if err != nil {
		return specCommandResult{}, err
	}
	return buildSpecCommandResult(
		specCommandOperationApprove,
		specCommandResultApproval,
		colony.OutcomeKindCompleted,
		mutation.Receipt.StateEffect,
		mutation.Specification,
		before.ID,
		mutation.Revision,
		mutation.AffectedScope,
		&mutation.Receipt,
		mutation.Replayed,
		projection,
		false,
	), nil
}

func repairSpecCommandProjection(root string) (specCommandResult, error) {
	repositoryRoot, state, specification, current, err := loadSpecCommandState(root)
	if err != nil {
		return specCommandResult{}, err
	}
	repair, err := repairSpecificationProjection(repositoryRoot, specificationMutationOptions{})
	if err != nil {
		return specCommandResult{}, err
	}
	outcome := colony.OutcomeKindNoChange
	effect := colony.LifecycleStateEffectNone
	var receipt *colony.LifecycleReceipt
	if repair.Repaired {
		outcome = colony.OutcomeKindCompleted
		effect = repair.Receipt.StateEffect
		receipt = &repair.Receipt
	}
	return buildSpecCommandResult(
		specCommandOperationProjectionRepair,
		specCommandResultProjectionRepair,
		outcome,
		effect,
		specification,
		current.ID,
		current,
		affectedSpecificationScope(current.Delta, state.Plan),
		receipt,
		repair.Replayed,
		repair.Inspection,
		repair.Repaired,
	), nil
}

func loadSpecCommandState(root string) (string, colony.ColonyState, colony.Specification, colony.SpecRevision, error) {
	repositoryRoot, err := canonicalSpecificationRoot(root)
	if err != nil {
		return "", colony.ColonyState{}, colony.Specification{}, colony.SpecRevision{}, err
	}
	state, err := loadSpecificationColonyState(repositoryRoot)
	if err != nil {
		return "", colony.ColonyState{}, colony.Specification{}, colony.SpecRevision{}, err
	}
	if state.Specification == nil {
		return "", colony.ColonyState{}, colony.Specification{}, colony.SpecRevision{}, fmt.Errorf("no specification exists; resolve current intent before running aether spec")
	}
	current, ok := currentSpecificationRevision(*state.Specification)
	if !ok {
		return "", colony.ColonyState{}, colony.Specification{}, colony.SpecRevision{}, fmt.Errorf("specification has no current revision")
	}
	return repositoryRoot, state, *state.Specification, current, nil
}

func specCommandSuccessorScope(state colony.ColonyState, specification colony.Specification, predecessor colony.SpecRevision, options specCommandOptions) (colony.SpecScope, error) {
	scope := predecessor.Scope
	if sessionID := strings.TrimSpace(options.SessionID); sessionID != "" {
		scope.SessionID = sessionID
	} else if state.SessionID != nil && strings.TrimSpace(*state.SessionID) != "" {
		scope.SessionID = strings.TrimSpace(*state.SessionID)
	}
	if kind := strings.TrimSpace(options.ScopeKind); kind != "" {
		scope = colony.SpecScope{
			Kind:      colony.SpecScopeKind(kind),
			GoalID:    specification.GoalID,
			SessionID: scope.SessionID,
		}
		if scope.Kind == colony.SpecScopeFeature {
			scope.FeatureID = strings.TrimSpace(options.FeatureID)
			scope.RequirementIDs = normalizeCLIStringList(options.RequirementIDs)
			scope.AcceptanceCheckIDs = normalizeCLIStringList(options.AcceptanceIDs)
		}
	}
	canonical, err := canonicalSpecificationScope(scope)
	if err != nil {
		return colony.SpecScope{}, fmt.Errorf("specification revision scope: %w", err)
	}
	return canonical, nil
}

func specCommandSourceText(root, text, file string) (string, error) {
	if value := strings.TrimSpace(text); value != "" {
		return value, nil
	}
	path := strings.TrimSpace(file)
	if !filepath.IsAbs(path) {
		path = filepath.Join(root, filepath.FromSlash(path))
	}
	path = filepath.Clean(path)
	realRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return "", fmt.Errorf("resolve specification repository root: %w", err)
	}
	realPath, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", fmt.Errorf("read --file source: %w", err)
	}
	relative, err := filepath.Rel(realRoot, realPath)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) || filepath.IsAbs(relative) {
		return "", fmt.Errorf("--file source must resolve inside the specification repository")
	}
	info, err := os.Stat(realPath)
	if err != nil {
		return "", fmt.Errorf("inspect --file source: %w", err)
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("--file source must be a regular file")
	}
	content, err := os.ReadFile(realPath)
	if err != nil {
		return "", fmt.Errorf("read --file source: %w", err)
	}
	value := strings.TrimSpace(string(content))
	if value == "" {
		return "", fmt.Errorf("--file source contains no owner-readable text")
	}
	return value, nil
}

func buildSpecCommandResult(
	operation specCommandOperation,
	resultKind specCommandResultKind,
	outcome colony.OutcomeKind,
	effect colony.LifecycleStateEffect,
	specification colony.Specification,
	beforeRevisionID string,
	revision colony.SpecRevision,
	affected specificationAffectedScope,
	receipt *colony.LifecycleReceipt,
	replayed bool,
	projection specificationProjectionInspection,
	projectionRepaired bool,
) specCommandResult {
	return specCommandResult{
		SchemaVersion:        specCommandSchemaVersion,
		Command:              "spec",
		Operation:            operation,
		ResultKind:           resultKind,
		OutcomeKind:          outcome,
		StateEffect:          effect,
		SpecificationID:      specification.ID,
		BeforeRevisionID:     beforeRevisionID,
		AfterRevisionID:      revision.ID,
		RevisionNumber:       specificationRevisionIndex(specification, revision.ID) + 1,
		ContentHash:          revision.ContentHash,
		Status:               revision.Status,
		Scope:                revision.Scope,
		Outcomes:             revision.Outcomes,
		IncludedBehaviors:    revision.IncludedBehaviors,
		Exclusions:           revision.Exclusions,
		BindingDecisions:     revision.BindingDecisions,
		Requirements:         revision.Requirements,
		AcceptanceChecks:     revision.AcceptanceChecks,
		NegativeExpectations: revision.NegativeExpectations,
		RecoveryExpectations: revision.RecoveryExpectations,
		AffectedPublicPaths:  revision.AffectedPublicPaths,
		ClassifiedDelta:      revision.Delta,
		AffectedScope:        affected,
		Approval:             revision.Approval,
		Receipt:              receipt,
		Replayed:             replayed,
		Projection: specCommandProjection{
			Path:           projection.Path,
			RevisionID:     projection.RevisionID,
			ExpectedDigest: projection.ExpectedDigest,
			ActualDigest:   projection.ActualDigest,
			Missing:        projection.Missing,
			Drifted:        projection.Drifted,
		},
		ProjectionRepaired: projectionRepaired,
		NextAction:         specCommandNextAction(specification.ID, revision, projection),
	}
}

func specCommandNextAction(specificationID string, revision colony.SpecRevision, projection specificationProjectionInspection) string {
	if projection.Drifted {
		return availableCandidateCommand(candidateSpecRepair)
	}
	if revision.Status == colony.SpecStatusDraft {
		return availableCandidateCommand(
			candidateSpecApprove,
			revision.ID,
			revision.ContentHash,
			specificationApprovalToken(specificationID, revision.ID, revision.ContentHash),
		)
	}
	return availableCandidateCommand(candidatePlan)
}

type specCommandVisualItem struct {
	ID          string
	Description string
	Detail      string
}

func renderSpecCommandVisual(result specCommandResult) string {
	var builder strings.Builder
	builder.WriteString(renderBanner("📜", "Specification"))
	fmt.Fprintf(&builder, "Operation: %s\n", result.Operation)
	fmt.Fprintf(&builder, "SPEC: %s revision %d\n", result.SpecificationID, result.RevisionNumber)
	fmt.Fprintf(&builder, "Before: %s\n", result.BeforeRevisionID)
	fmt.Fprintf(&builder, "After: %s\n", result.AfterRevisionID)
	fmt.Fprintf(&builder, "Status: %s\n", strings.ToUpper(string(result.Status)))
	fmt.Fprintf(&builder, "Scope: %s\n", result.Scope.Kind)
	if result.Scope.Kind == colony.SpecScopeFeature {
		fmt.Fprintf(&builder, "Feature: %s\n", result.Scope.FeatureID)
	}
	if result.Replayed {
		builder.WriteString("Already recorded; the exact revision and receipt were retained.\n")
	}

	renderSpecCommandVisualSection(&builder, "What this goal delivers", specCommandOutcomeVisualItems(result.Outcomes))
	renderSpecCommandVisualSection(&builder, "Included", specCommandIncludedVisualItems(result.IncludedBehaviors))
	renderSpecCommandVisualSection(&builder, "Explicitly excluded", specCommandExclusionVisualItems(result.Exclusions))
	renderSpecCommandVisualSection(&builder, "Binding decisions", specCommandDecisionVisualItems(result.BindingDecisions))
	renderSpecCommandVisualSection(&builder, "Requirements", specCommandRequirementVisualItems(result.Requirements))
	renderSpecCommandVisualSection(&builder, "Owner-checkable acceptance", specCommandAcceptanceVisualItems(result.AcceptanceChecks))
	renderSpecCommandVisualSection(&builder, "Negative expectations", specCommandNegativeVisualItems(result.NegativeExpectations))
	renderSpecCommandVisualSection(&builder, "Recovery expectations", specCommandRecoveryVisualItems(result.RecoveryExpectations))
	renderSpecCommandVisualSection(&builder, "Affected public paths", specCommandPublicPathVisualItems(result.AffectedPublicPaths))

	builder.WriteString(renderStageMarker("Revision Impact"))
	renderSpecCommandVisualDelta(&builder, "Outcome", result.ClassifiedDelta.Outcomes)
	renderSpecCommandVisualDelta(&builder, "Included", result.ClassifiedDelta.IncludedBehaviors)
	renderSpecCommandVisualDelta(&builder, "Exclusions", result.ClassifiedDelta.Exclusions)
	renderSpecCommandVisualDelta(&builder, "Binding decisions", result.ClassifiedDelta.BindingDecisions)
	renderSpecCommandVisualDelta(&builder, "Requirements", result.ClassifiedDelta.Requirements)
	renderSpecCommandVisualDelta(&builder, "Acceptance", result.ClassifiedDelta.AcceptanceChecks)
	renderSpecCommandVisualDelta(&builder, "Negative", result.ClassifiedDelta.NegativeExpectations)
	renderSpecCommandVisualDelta(&builder, "Recovery", result.ClassifiedDelta.RecoveryExpectations)
	renderSpecCommandVisualDelta(&builder, "Public paths", result.ClassifiedDelta.AffectedPublicPaths)
	fmt.Fprintf(&builder, "Affected specification IDs: %s\n", specCommandIDSummary(result.AffectedScope.SpecItemIDs))
	fmt.Fprintf(&builder, "Affected task IDs: %s\n", specCommandIDSummary(result.AffectedScope.TaskIDs))
	fmt.Fprintf(&builder, "Affected proof IDs: %s\n", specCommandIDSummary(result.AffectedScope.ProofLinkIDs))
	if result.ProjectionRepaired {
		builder.WriteString("Projection: repaired from canonical state; specification authority was unchanged.\n")
	} else if result.Projection.Drifted {
		builder.WriteString("Projection: DRIFTED; canonical state was not changed.\n")
	} else {
		builder.WriteString("Projection: synchronized with canonical state.\n")
	}
	if result.Approval != nil {
		fmt.Fprintf(&builder, "Approval receipt: %s\n", result.Approval.ID)
		builder.WriteString("Specification approval does not accept or activate a plan.\n")
	}
	if result.Receipt != nil {
		fmt.Fprintf(&builder, "Transaction receipt: %s\n", result.Receipt.ReceiptID)
	}
	fmt.Fprintf(&builder, "State effect: %s\n", result.StateEffect)
	builder.WriteString(renderNextUp(result.NextAction))
	return builder.String()
}

func renderSpecCommandVisualSection(builder *strings.Builder, title string, items []specCommandVisualItem) {
	builder.WriteString(renderStageMarker(title))
	for _, item := range items {
		fmt.Fprintf(builder, "%s  %s\n", item.ID, item.Description)
		if item.Detail != "" {
			fmt.Fprintf(builder, "   %s\n", item.Detail)
		}
	}
}

func renderSpecCommandVisualDelta(builder *strings.Builder, title string, delta colony.SpecItemDelta) {
	fmt.Fprintf(builder, "%s: +%s ~%s -%s =%s\n", title,
		specCommandIDSummary(delta.AddedIDs),
		specCommandIDSummary(delta.ModifiedIDs),
		specCommandIDSummary(delta.RemovedIDs),
		specCommandIDSummary(delta.UnchangedIDs),
	)
}

func specCommandIDSummary(values []string) string {
	if len(values) == 0 {
		return "none"
	}
	return strings.Join(values, ", ")
}

func specCommandOutcomeVisualItems(values []colony.SpecOutcome) []specCommandVisualItem {
	items := make([]specCommandVisualItem, len(values))
	for index, value := range values {
		items[index] = specCommandVisualItem{ID: value.ID, Description: value.Description}
	}
	return items
}

func specCommandIncludedVisualItems(values []colony.SpecIncludedBehavior) []specCommandVisualItem {
	items := make([]specCommandVisualItem, len(values))
	for index, value := range values {
		items[index] = specCommandVisualItem{ID: value.ID, Description: value.Description}
	}
	return items
}

func specCommandExclusionVisualItems(values []colony.SpecExclusion) []specCommandVisualItem {
	items := make([]specCommandVisualItem, len(values))
	for index, value := range values {
		items[index] = specCommandVisualItem{ID: value.ID, Description: value.Description}
	}
	return items
}

func specCommandDecisionVisualItems(values []colony.SpecBindingDecision) []specCommandVisualItem {
	items := make([]specCommandVisualItem, len(values))
	for index, value := range values {
		items[index] = specCommandVisualItem{ID: value.ID, Description: value.Description}
	}
	return items
}

func specCommandRequirementVisualItems(values []colony.SpecRequirement) []specCommandVisualItem {
	items := make([]specCommandVisualItem, len(values))
	for index, value := range values {
		items[index] = specCommandVisualItem{ID: value.ID, Description: value.Description}
	}
	return items
}

func specCommandAcceptanceVisualItems(values []colony.SpecAcceptanceCheck) []specCommandVisualItem {
	items := make([]specCommandVisualItem, len(values))
	for index, value := range values {
		items[index] = specCommandVisualItem{ID: value.ID, Description: value.Description, Detail: "Verification: " + value.Verification}
	}
	return items
}

func specCommandNegativeVisualItems(values []colony.SpecNegativeExpectation) []specCommandVisualItem {
	items := make([]specCommandVisualItem, len(values))
	for index, value := range values {
		items[index] = specCommandVisualItem{ID: value.ID, Description: value.Description}
	}
	return items
}

func specCommandRecoveryVisualItems(values []colony.SpecRecoveryExpectation) []specCommandVisualItem {
	items := make([]specCommandVisualItem, len(values))
	for index, value := range values {
		items[index] = specCommandVisualItem{ID: value.ID, Description: value.Description}
	}
	return items
}

func specCommandPublicPathVisualItems(values []colony.SpecPublicPath) []specCommandVisualItem {
	items := make([]specCommandVisualItem, len(values))
	for index, value := range values {
		items[index] = specCommandVisualItem{ID: value.ID, Description: value.Description, Detail: "Path: " + value.Path}
	}
	return items
}
