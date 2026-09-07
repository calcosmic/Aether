package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/calcosmic/Aether/pkg/colony"
)

const specificationProjectionRelativePath = ".aether/SPEC.md"

type specificationProjectionInspection struct {
	Path           string
	RevisionID     string
	ExpectedDigest string
	ActualDigest   string
	Missing        bool
	Drifted        bool
}

type specificationProjectionRepairResult struct {
	Inspection specificationProjectionInspection
	Receipt    colony.LifecycleReceipt
	Repaired   bool
	Replayed   bool
}

type specificationProjectionEntry struct {
	ID          string
	Description string
	DetailName  string
	Detail      string
	EvidenceIDs []string
}

// renderSpecificationProjection is a one-way view over canonical state. It
// validates and reads the aggregate but never parses Markdown back into
// authority, so changing SPEC.md cannot approve or revise anything.
func renderSpecificationProjection(specification colony.Specification, plan colony.Plan) ([]byte, error) {
	if err := validateSpecificationState(specification); err != nil {
		return nil, fmt.Errorf("render specification projection: %w", err)
	}
	revision, ok := currentSpecificationRevision(specification)
	if !ok {
		return nil, fmt.Errorf("render specification projection: current revision is missing")
	}
	revisionNumber := specificationRevisionIndex(specification, revision.ID) + 1
	affected := affectedSpecificationScope(revision.Delta, plan)

	var builder strings.Builder
	builder.WriteString("# Aether Specification\n\n")
	renderSpecificationProjectionSection(&builder, "Outcome", specificationProjectionOutcomeEntries(revision.Outcomes))
	renderSpecificationProjectionSection(&builder, "Included Behavior", specificationProjectionIncludedEntries(revision.IncludedBehaviors))
	renderSpecificationProjectionSection(&builder, "Explicit Exclusions", specificationProjectionExclusionEntries(revision.Exclusions))
	renderSpecificationProjectionSection(&builder, "Binding Decisions", specificationProjectionDecisionEntries(revision.BindingDecisions))
	renderSpecificationProjectionSection(&builder, "Stable Requirements", specificationProjectionRequirementEntries(revision.Requirements))
	renderSpecificationProjectionSection(&builder, "Owner-Checkable Acceptance", specificationProjectionAcceptanceEntries(revision.AcceptanceChecks))
	renderSpecificationProjectionSection(&builder, "Negative Expectations", specificationProjectionNegativeEntries(revision.NegativeExpectations))
	renderSpecificationProjectionSection(&builder, "Recovery Expectations", specificationProjectionRecoveryEntries(revision.RecoveryExpectations))
	renderSpecificationProjectionSection(&builder, "Affected Public Paths", specificationProjectionPublicPathEntries(revision.AffectedPublicPaths))

	builder.WriteString("## Revision and Status\n\n")
	fmt.Fprintf(&builder, "- Goal: `%s`\n", specificationProjectionCode(specification.GoalID))
	fmt.Fprintf(&builder, "- Specification: `%s`\n", specificationProjectionCode(specification.ID))
	fmt.Fprintf(&builder, "- Revision: `%s` (revision %d)\n", specificationProjectionCode(revision.ID), revisionNumber)
	fmt.Fprintf(&builder, "- Content hash: `%s`\n", specificationProjectionCode(revision.ContentHash))
	fmt.Fprintf(&builder, "- Status: `%s`\n", strings.ToUpper(string(revision.Status)))
	if revision.PredecessorID == "" {
		builder.WriteString("- Predecessor: none\n")
	} else {
		fmt.Fprintf(&builder, "- Predecessor: `%s`\n", specificationProjectionCode(revision.PredecessorID))
	}
	if revision.Approval == nil {
		builder.WriteString("- Approval receipt: none\n\n")
	} else {
		fmt.Fprintf(&builder, "- Approval receipt: `%s` by %s\n\n", specificationProjectionCode(revision.Approval.ID), specificationProjectionText(revision.Approval.ApprovedBy))
	}

	builder.WriteString("## Scope\n\n")
	fmt.Fprintf(&builder, "- Kind: `%s`\n", specificationProjectionCode(string(revision.Scope.Kind)))
	fmt.Fprintf(&builder, "- Session: `%s`\n", specificationProjectionCode(revision.Scope.SessionID))
	if revision.Scope.Kind == colony.SpecScopeFeature {
		fmt.Fprintf(&builder, "- Feature: `%s`\n", specificationProjectionCode(revision.Scope.FeatureID))
		builder.WriteString("- Requirement IDs: ")
		renderSpecificationProjectionInlineIDs(&builder, revision.Scope.RequirementIDs)
		builder.WriteString("- Acceptance check IDs: ")
		renderSpecificationProjectionInlineIDs(&builder, revision.Scope.AcceptanceCheckIDs)
	}
	builder.WriteByte('\n')

	builder.WriteString("## Evidence Summary\n\n")
	renderSpecificationProjectionEvidence(&builder, revision)
	builder.WriteByte('\n')

	builder.WriteString("## Classified Delta\n\n")
	if revision.Delta.PredecessorRevisionID == "" {
		builder.WriteString("- Predecessor revision: none\n\n")
	} else {
		fmt.Fprintf(&builder, "- Predecessor revision: `%s`\n\n", specificationProjectionCode(revision.Delta.PredecessorRevisionID))
	}
	renderSpecificationProjectionDelta(&builder, "Outcome", revision.Delta.Outcomes)
	renderSpecificationProjectionDelta(&builder, "Included Behavior", revision.Delta.IncludedBehaviors)
	renderSpecificationProjectionDelta(&builder, "Explicit Exclusions", revision.Delta.Exclusions)
	renderSpecificationProjectionDelta(&builder, "Binding Decisions", revision.Delta.BindingDecisions)
	renderSpecificationProjectionDelta(&builder, "Stable Requirements", revision.Delta.Requirements)
	renderSpecificationProjectionDelta(&builder, "Owner-Checkable Acceptance", revision.Delta.AcceptanceChecks)
	renderSpecificationProjectionDelta(&builder, "Negative Expectations", revision.Delta.NegativeExpectations)
	renderSpecificationProjectionDelta(&builder, "Recovery Expectations", revision.Delta.RecoveryExpectations)
	renderSpecificationProjectionDelta(&builder, "Affected Public Paths", revision.Delta.AffectedPublicPaths)

	builder.WriteString("## Affected Downstream Scope\n\n")
	renderSpecificationProjectionNamedIDs(&builder, "Specification items", affected.SpecItemIDs)
	renderSpecificationProjectionNamedIDs(&builder, "Requirements", affected.RequirementIDs)
	renderSpecificationProjectionNamedIDs(&builder, "Tasks", affected.TaskIDs)
	renderSpecificationProjectionNamedIDs(&builder, "Proof links", affected.ProofLinkIDs)
	builder.WriteString("- Unaffected work: retained and still valid\n\n")

	builder.WriteString("## Exact Next Command\n\n")
	switch revision.Status {
	case colony.SpecStatusDraft:
		builder.WriteString("This draft does not authorize planning until the owner approves this exact revision.\n\n")
		token := specificationApprovalToken(specification.ID, revision.ID, revision.ContentHash)
		fmt.Fprintf(&builder, "`aether spec --approve --revision-id %s --revision-hash %s --approval-token '%s'`\n", revision.ID, revision.ContentHash, token)
	case colony.SpecStatusApproved:
		builder.WriteString("This exact specification revision is approved. Specification approval does not accept a plan.\n\n")
		builder.WriteString("`aether plan`\n")
	default:
		return nil, fmt.Errorf("render specification projection: current revision has unsupported status %q", revision.Status)
	}

	return []byte(builder.String()), nil
}

func renderSpecificationProjectionSection(builder *strings.Builder, heading string, entries []specificationProjectionEntry) {
	fmt.Fprintf(builder, "## %s\n\n", heading)
	if len(entries) == 0 {
		renderSpecificationProjectionList(builder, nil)
		builder.WriteByte('\n')
		return
	}
	for _, entry := range entries {
		fmt.Fprintf(builder, "- `%s` — %s\n", specificationProjectionCode(entry.ID), specificationProjectionText(entry.Description))
		if entry.DetailName != "" {
			fmt.Fprintf(builder, "  - %s: %s\n", entry.DetailName, specificationProjectionText(entry.Detail))
		}
		builder.WriteString("  - Evidence: ")
		renderSpecificationProjectionInlineIDs(builder, entry.EvidenceIDs)
	}
	builder.WriteByte('\n')
}

func renderSpecificationProjectionList(builder *strings.Builder, values []string) {
	if len(values) == 0 {
		builder.WriteString("- none\n")
		return
	}
	for _, value := range values {
		fmt.Fprintf(builder, "- `%s`\n", specificationProjectionCode(value))
	}
}

func renderSpecificationProjectionInlineIDs(builder *strings.Builder, values []string) {
	if len(values) == 0 {
		builder.WriteString("none\n")
		return
	}
	for index, value := range values {
		if index > 0 {
			builder.WriteString(", ")
		}
		fmt.Fprintf(builder, "`%s`", specificationProjectionCode(value))
	}
	builder.WriteByte('\n')
}

func renderSpecificationProjectionNamedIDs(builder *strings.Builder, name string, values []string) {
	fmt.Fprintf(builder, "- %s: ", name)
	renderSpecificationProjectionInlineIDs(builder, values)
}

func renderSpecificationProjectionDelta(builder *strings.Builder, heading string, delta colony.SpecItemDelta) {
	fmt.Fprintf(builder, "### %s\n\n", heading)
	renderSpecificationProjectionNamedIDs(builder, "Added", delta.AddedIDs)
	renderSpecificationProjectionNamedIDs(builder, "Modified", delta.ModifiedIDs)
	renderSpecificationProjectionNamedIDs(builder, "Removed (retained in history)", delta.RemovedIDs)
	renderSpecificationProjectionNamedIDs(builder, "Unchanged", delta.UnchangedIDs)
	builder.WriteByte('\n')
}

func renderSpecificationProjectionEvidence(builder *strings.Builder, revision colony.SpecRevision) {
	owners := make(map[string][]string)
	add := func(id string, evidenceIDs []string) {
		for _, evidenceID := range evidenceIDs {
			owners[evidenceID] = append(owners[evidenceID], id)
		}
	}
	for _, entry := range specificationProjectionAllEntries(revision) {
		add(entry.ID, entry.EvidenceIDs)
	}
	ids := make([]string, 0, len(owners))
	for id := range owners {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	if len(ids) == 0 {
		renderSpecificationProjectionList(builder, nil)
		return
	}
	for _, id := range ids {
		linked := canonicalProjectionIDs(owners[id])
		fmt.Fprintf(builder, "- `%s` supports ", specificationProjectionCode(id))
		renderSpecificationProjectionInlineIDs(builder, linked)
	}
}

func specificationProjectionAllEntries(revision colony.SpecRevision) []specificationProjectionEntry {
	result := []specificationProjectionEntry{}
	result = append(result, specificationProjectionOutcomeEntries(revision.Outcomes)...)
	result = append(result, specificationProjectionIncludedEntries(revision.IncludedBehaviors)...)
	result = append(result, specificationProjectionExclusionEntries(revision.Exclusions)...)
	result = append(result, specificationProjectionDecisionEntries(revision.BindingDecisions)...)
	result = append(result, specificationProjectionRequirementEntries(revision.Requirements)...)
	result = append(result, specificationProjectionAcceptanceEntries(revision.AcceptanceChecks)...)
	result = append(result, specificationProjectionNegativeEntries(revision.NegativeExpectations)...)
	result = append(result, specificationProjectionRecoveryEntries(revision.RecoveryExpectations)...)
	result = append(result, specificationProjectionPublicPathEntries(revision.AffectedPublicPaths)...)
	return result
}

func specificationProjectionOutcomeEntries(values []colony.SpecOutcome) []specificationProjectionEntry {
	result := make([]specificationProjectionEntry, len(values))
	for index, value := range values {
		result[index] = specificationProjectionEntry{ID: value.ID, Description: value.Description, EvidenceIDs: value.EvidenceIDs}
	}
	return result
}

func specificationProjectionIncludedEntries(values []colony.SpecIncludedBehavior) []specificationProjectionEntry {
	result := make([]specificationProjectionEntry, len(values))
	for index, value := range values {
		result[index] = specificationProjectionEntry{ID: value.ID, Description: value.Description, EvidenceIDs: value.EvidenceIDs}
	}
	return result
}

func specificationProjectionExclusionEntries(values []colony.SpecExclusion) []specificationProjectionEntry {
	result := make([]specificationProjectionEntry, len(values))
	for index, value := range values {
		result[index] = specificationProjectionEntry{ID: value.ID, Description: value.Description, EvidenceIDs: value.EvidenceIDs}
	}
	return result
}

func specificationProjectionDecisionEntries(values []colony.SpecBindingDecision) []specificationProjectionEntry {
	result := make([]specificationProjectionEntry, len(values))
	for index, value := range values {
		result[index] = specificationProjectionEntry{ID: value.ID, Description: value.Description, EvidenceIDs: value.EvidenceIDs}
	}
	return result
}

func specificationProjectionRequirementEntries(values []colony.SpecRequirement) []specificationProjectionEntry {
	result := make([]specificationProjectionEntry, len(values))
	for index, value := range values {
		result[index] = specificationProjectionEntry{ID: value.ID, Description: value.Description, EvidenceIDs: value.EvidenceIDs}
	}
	return result
}

func specificationProjectionAcceptanceEntries(values []colony.SpecAcceptanceCheck) []specificationProjectionEntry {
	result := make([]specificationProjectionEntry, len(values))
	for index, value := range values {
		result[index] = specificationProjectionEntry{ID: value.ID, Description: value.Description, DetailName: "Verification", Detail: value.Verification, EvidenceIDs: value.EvidenceIDs}
	}
	return result
}

func specificationProjectionNegativeEntries(values []colony.SpecNegativeExpectation) []specificationProjectionEntry {
	result := make([]specificationProjectionEntry, len(values))
	for index, value := range values {
		result[index] = specificationProjectionEntry{ID: value.ID, Description: value.Description, EvidenceIDs: value.EvidenceIDs}
	}
	return result
}

func specificationProjectionRecoveryEntries(values []colony.SpecRecoveryExpectation) []specificationProjectionEntry {
	result := make([]specificationProjectionEntry, len(values))
	for index, value := range values {
		result[index] = specificationProjectionEntry{ID: value.ID, Description: value.Description, EvidenceIDs: value.EvidenceIDs}
	}
	return result
}

func specificationProjectionPublicPathEntries(values []colony.SpecPublicPath) []specificationProjectionEntry {
	result := make([]specificationProjectionEntry, len(values))
	for index, value := range values {
		result[index] = specificationProjectionEntry{ID: value.ID, Description: value.Description, DetailName: "Path", Detail: value.Path, EvidenceIDs: value.EvidenceIDs}
	}
	return result
}

func canonicalProjectionIDs(values []string) []string {
	set := make(map[string]struct{}, len(values))
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			set[value] = struct{}{}
		}
	}
	result := make([]string, 0, len(set))
	for value := range set {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func specificationProjectionText(value string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
}

func specificationProjectionCode(value string) string {
	return strings.ReplaceAll(specificationProjectionText(value), "`", "&#96;")
}

func specificationStateProjectionTargets(state colony.ColonyState) ([]specificationTransactionTarget, error) {
	if state.Specification == nil {
		return nil, fmt.Errorf("canonical state has no specification to project")
	}
	stateBytes, err := marshalSpecificationState(state)
	if err != nil {
		return nil, err
	}
	projection, err := renderSpecificationProjection(*state.Specification, state.Plan)
	if err != nil {
		return nil, err
	}
	return []specificationTransactionTarget{
		{Root: lifecycleTransactionRootData, Path: "COLONY_STATE.json", Content: stateBytes},
		{Root: lifecycleTransactionRootRepository, Path: specificationProjectionRelativePath, Content: projection},
	}, nil
}

func inspectSpecificationProjection(root string) (specificationProjectionInspection, error) {
	repositoryRoot, err := canonicalSpecificationRoot(root)
	if err != nil {
		return specificationProjectionInspection{}, err
	}
	state, err := loadSpecificationColonyState(repositoryRoot)
	if err != nil {
		return specificationProjectionInspection{}, err
	}
	if state.Specification == nil {
		return specificationProjectionInspection{}, fmt.Errorf("no specification exists to inspect")
	}
	expected, err := renderSpecificationProjection(*state.Specification, state.Plan)
	if err != nil {
		return specificationProjectionInspection{}, err
	}
	return inspectSpecificationProjectionBytes(repositoryRoot, *state.Specification, expected)
}

func inspectSpecificationProjectionBytes(root string, specification colony.Specification, expected []byte) (specificationProjectionInspection, error) {
	revision, ok := currentSpecificationRevision(specification)
	if !ok {
		return specificationProjectionInspection{}, fmt.Errorf("specification has no current revision")
	}
	path := filepath.Join(root, specificationProjectionRelativePath)
	actual, err := readLifecycleFileState(path)
	if err != nil {
		return specificationProjectionInspection{}, fmt.Errorf("inspect specification projection: %w", err)
	}
	expectedDigest := lifecycleDigest(expected)
	return specificationProjectionInspection{
		Path:           path,
		RevisionID:     revision.ID,
		ExpectedDigest: expectedDigest,
		ActualDigest:   actual.Digest,
		Missing:        !actual.Exists,
		Drifted:        !actual.Exists || actual.Digest != expectedDigest,
	}, nil
}

func repairSpecificationProjection(root string, opts specificationMutationOptions) (specificationProjectionRepairResult, error) {
	repositoryRoot, err := canonicalSpecificationRoot(root)
	if err != nil {
		return specificationProjectionRepairResult{}, err
	}
	state, err := loadSpecificationColonyState(repositoryRoot)
	if err != nil {
		return specificationProjectionRepairResult{}, err
	}
	if state.Specification == nil {
		return specificationProjectionRepairResult{}, fmt.Errorf("no specification exists to repair")
	}
	projection, err := renderSpecificationProjection(*state.Specification, state.Plan)
	if err != nil {
		return specificationProjectionRepairResult{}, err
	}
	targets := []specificationTransactionTarget{{Root: lifecycleTransactionRootRepository, Path: specificationProjectionRelativePath, Content: projection}}
	transactionID, resume, err := specificationProjectionRepairTransaction(repositoryRoot, state.Specification.CurrentRevisionID, targets)
	if err != nil {
		return specificationProjectionRepairResult{}, err
	}
	if resume {
		receipt, commitErr := commitSpecificationTargets(repositoryRoot, transactionID, "specification-projection-repair", targets, opts)
		if commitErr != nil {
			return specificationProjectionRepairResult{}, fmt.Errorf("resume specification projection repair: %w", commitErr)
		}
		inspection, inspectErr := inspectSpecificationProjectionBytes(repositoryRoot, *state.Specification, projection)
		if inspectErr != nil {
			return specificationProjectionRepairResult{}, inspectErr
		}
		if inspection.Drifted {
			return specificationProjectionRepairResult{}, fmt.Errorf("specification projection repair replay did not restore canonical bytes")
		}
		return specificationProjectionRepairResult{Inspection: inspection, Receipt: receipt, Repaired: true, Replayed: true}, nil
	}

	inspection, err := inspectSpecificationProjectionBytes(repositoryRoot, *state.Specification, projection)
	if err != nil {
		return specificationProjectionRepairResult{}, err
	}
	if !inspection.Drifted {
		return specificationProjectionRepairResult{Inspection: inspection}, nil
	}
	receipt, err := commitSpecificationTargets(repositoryRoot, transactionID, "specification-projection-repair", targets, opts)
	if err != nil {
		return specificationProjectionRepairResult{}, fmt.Errorf("repair specification projection: %w", err)
	}
	inspection, err = inspectSpecificationProjectionBytes(repositoryRoot, *state.Specification, projection)
	if err != nil {
		return specificationProjectionRepairResult{}, err
	}
	if inspection.Drifted {
		return specificationProjectionRepairResult{}, fmt.Errorf("specification projection repair did not restore canonical bytes")
	}
	return specificationProjectionRepairResult{Inspection: inspection, Receipt: receipt, Repaired: true}, nil
}

// specificationProjectionRepairTransaction reuses only an unfinished intent.
// Completed repairs get a fresh ordinal so the same manual tamper can be fixed
// more than once without colliding with a durable historical receipt.
func specificationProjectionRepairTransaction(root, revisionID string, targets []specificationTransactionTarget) (string, bool, error) {
	digest := lifecycleDigest([]byte(strings.TrimSpace(revisionID)))
	prefix := "spec-projection-repair-" + strings.TrimPrefix(digest, "sha256:")[:16] + "-"
	transactionsRoot := filepath.Join(root, ".aether", "data", "transactions")
	if info, statErr := os.Lstat(transactionsRoot); statErr == nil {
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return "", false, fmt.Errorf("specification projection repair journal root is not a real directory")
		}
	} else if !os.IsNotExist(statErr) {
		return "", false, fmt.Errorf("inspect specification projection repair journal root: %w", statErr)
	}
	entries, err := os.ReadDir(transactionsRoot)
	if os.IsNotExist(err) {
		return prefix + "0001", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("list specification projection repair transactions: %w", err)
	}
	maximum := 0
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasPrefix(name, prefix) {
			continue
		}
		ordinal, parseErr := strconv.Atoi(strings.TrimPrefix(name, prefix))
		if parseErr != nil || ordinal < 1 {
			continue
		}
		if ordinal > maximum {
			maximum = ordinal
		}
		info, infoErr := entry.Info()
		if infoErr != nil || info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return "", false, fmt.Errorf("unsafe specification projection repair journal %q", filepath.Join(transactionsRoot, name))
		}
		journalPath := filepath.Join(transactionsRoot, name)
		if _, receiptErr := os.Lstat(filepath.Join(journalPath, "receipt.json")); receiptErr == nil {
			continue
		} else if !os.IsNotExist(receiptErr) {
			return "", false, fmt.Errorf("inspect specification projection repair receipt: %w", receiptErr)
		}
		progressBytes, progressErr := os.ReadFile(filepath.Join(journalPath, "progress.json"))
		if progressErr == nil {
			var progress lifecycleTransactionProgress
			if decodeErr := decodeLifecycleJSON(progressBytes, &progress); decodeErr != nil {
				return "", false, fmt.Errorf("decode specification projection repair progress: %w", decodeErr)
			}
			if progress.Stage == colony.TransactionStageRolledBack || progress.Stage == colony.TransactionStageRecoveryRequired {
				continue
			}
		} else if !os.IsNotExist(progressErr) {
			return "", false, fmt.Errorf("read specification projection repair progress: %w", progressErr)
		}
		config := lifecycleTransactionConfig{
			TransactionID: name,
			Command:       "specification-projection-repair",
			Allowlist: lifecycleTransactionAllowlist{
				RepositoryRoot:    root,
				LifecycleDataRoot: filepath.Join(root, ".aether", "data"),
			},
		}
		pending, pendingErr := specificationTransactionHasIntent(config)
		if pendingErr != nil {
			return "", false, pendingErr
		}
		if !pending {
			continue
		}
		matches, matchErr := specificationPendingIntentMatches(config, targets)
		if matchErr != nil {
			return "", false, matchErr
		}
		if !matches {
			return "", false, fmt.Errorf("unfinished specification projection repair %q has divergent staged content", name)
		}
		return name, true, nil
	}
	return fmt.Sprintf("%s%04d", prefix, maximum+1), false, nil
}
