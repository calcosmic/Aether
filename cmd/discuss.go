package cmd

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/spf13/cobra"
)

const (
	clarificationDecisionType = "clarification"
	discussOptionDelimiter    = " Options: "
	discussSourcePrefix       = "discuss:"

	clarifiedIntentMaxEntries       = 6
	clarifiedIntentMaxQuestionChars = 180
	clarifiedIntentMaxAnswerChars   = 500
	clarifiedIntentMaxEntryChars    = 720
	clarifiedIntentMaxSectionChars  = 1600
)

type discussQuestion struct {
	ID                  string                       `json:"id,omitempty"`
	StableID            string                       `json:"stable_id,omitempty"`
	Category            string                       `json:"category"`
	Domain              planningDecisionDomain       `json:"domain,omitempty"`
	Question            string                       `json:"question"`
	Decision            string                       `json:"decision,omitempty"`
	Options             []string                     `json:"options"`
	Reasoning           string                       `json:"reasoning"`
	WhyNow              string                       `json:"why_now,omitempty"`
	Evidence            []colony.PlanningEvidenceRef `json:"evidence,omitempty"`
	QueenRecommendation string                       `json:"queen_recommendation,omitempty"`
	Choices             []planningDecisionChoice     `json:"choices,omitempty"`
	AffectedSemanticIDs []string                     `json:"affected_semantic_ids,omitempty"`
	PriorAnswer         string                       `json:"prior_answer,omitempty"`
	Revalidation        string                       `json:"revalidation,omitempty"`
	PlanningResumes     string                       `json:"planning_resumes,omitempty"`
	ExactAnswerSyntax   string                       `json:"exact_answer_syntax,omitempty"`
	HardConstraint      bool                         `json:"hard_constraint,omitempty"`
	Status              string                       `json:"status,omitempty"`
	Source              string                       `json:"source,omitempty"`
}

type discussMaterialBatch struct {
	ID          string            `json:"id"`
	ContentHash string            `json:"content_hash"`
	Cards       []discussQuestion `json:"cards"`
}

// discussSpecificationBody keeps the nine owner-readable specification
// categories separate in the discuss closeout. The specification engine owns
// their canonical forms; this is only the structured presentation of the
// exact revision it committed or retained.
type discussSpecificationBody struct {
	Outcomes             []colony.SpecOutcome             `json:"outcomes"`
	IncludedBehaviors    []colony.SpecIncludedBehavior    `json:"included_behaviors"`
	Exclusions           []colony.SpecExclusion           `json:"exclusions"`
	BindingDecisions     []colony.SpecBindingDecision     `json:"binding_decisions"`
	Requirements         []colony.SpecRequirement         `json:"requirements"`
	AcceptanceChecks     []colony.SpecAcceptanceCheck     `json:"acceptance_checks"`
	NegativeExpectations []colony.SpecNegativeExpectation `json:"negative_expectations"`
	RecoveryExpectations []colony.SpecRecoveryExpectation `json:"recovery_expectations"`
	AffectedPublicPaths  []colony.SpecPublicPath          `json:"affected_public_paths"`
}

type discussSpecificationCloseout struct {
	SpecificationID       string                    `json:"specification_id"`
	RevisionID            string                    `json:"revision_id"`
	RevisionNumber        int                       `json:"revision_number"`
	ContentHash           string                    `json:"content_hash"`
	Status                colony.SpecRevisionStatus `json:"status"`
	Scope                 colony.SpecScope          `json:"scope"`
	SectionCounts         map[string]int            `json:"section_counts"`
	Body                  discussSpecificationBody  `json:"body"`
	UnresolvedCount       int                       `json:"unresolved_count"`
	ProjectionPath        string                    `json:"projection_path"`
	ProjectionDigest      string                    `json:"projection_digest"`
	ProjectionRepaired    bool                      `json:"projection_repaired"`
	ExactNextCommand      string                    `json:"exact_next_command"`
	ApprovalCommand       string                    `json:"approval_command,omitempty"`
	RevisionGuidance      string                    `json:"revision_guidance,omitempty"`
	Replayed              bool                      `json:"replayed"`
	WouldCreate           bool                      `json:"would_create,omitempty"`
	ApprovedSpecPreserved bool                      `json:"approved_spec_preserved,omitempty"`
	Receipt               *colony.LifecycleReceipt  `json:"receipt,omitempty"`
}

// discussEvidenceItem retains source text only while deciding whether current
// evidence already answers a question. Only the safe record projection leaves
// this file in a command result.
type discussEvidenceItem struct {
	Record  planningEvidenceRecord
	Content string
}

type discussEvidenceFrontier struct {
	Scope planningEvidenceScope
	Items []discussEvidenceItem
}

type clarifiedIntentEntry struct {
	ID         string `json:"id"`
	Question   string `json:"question"`
	Resolution string `json:"resolution"`
	Source     string `json:"source,omitempty"`
}

type clarifiedIntentRenderResult struct {
	// DecisionIDs correspond one-for-one with admitted Lines, after integrity and budget checks.
	DecisionIDs []string
	Lines       []string
	Blocked     []colonyPrimeLedgerItem
	Warnings    []string
}

type pendingDecisionScope struct {
	GoalHash      string
	SessionID     string
	InitializedAt *time.Time
}

var discussCmd = &cobra.Command{
	Use:   "discuss",
	Short: "Surface intent clarifications before planning and record resolved answers",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		resolveID, _ := cmd.Flags().GetString("resolve")
		answer, _ := cmd.Flags().GetString("answer")
		if strings.TrimSpace(resolveID) != "" {
			result, err := resolveDiscussQuestion(resolveID, answer)
			if err != nil {
				outputError(1, err.Error(), nil)
				return nil
			}
			outputWorkflow(result, renderDiscussVisual(result))
			return nil
		}

		// Typed intake for Queen-composed questions: the wrapper's model
		// composes questions from THIS goal and THIS codebase, and this
		// path materializes them into the same pending-decisions pipeline
		// the canned generator uses — same resolution, same REDIRECT
		// emission, same clarified-intent injection into worker context.
		if question, _ := cmd.Flags().GetString("add-question"); strings.TrimSpace(question) != "" {
			optionsRaw, _ := cmd.Flags().GetString("options")
			category, _ := cmd.Flags().GetString("category")
			grounding, _ := cmd.Flags().GetString("grounding")
			hard, _ := cmd.Flags().GetBool("hard")
			source, _ := cmd.Flags().GetString("source")
			result, err := addComposedDiscussQuestion(question, optionsRaw, category, grounding, source, hard)
			if err != nil {
				outputError(1, err.Error(), nil)
				return nil
			}
			outputWorkflow(result, renderComposedQuestionVisual(result))
			return nil
		}

		maxQuestions, _ := cmd.Flags().GetInt("max-questions")
		if maxQuestions <= 0 {
			maxQuestions = 3
		}
		dryRun, _ := cmd.Flags().GetBool("dry-run")

		result, err := runDiscuss(skillWorkspaceRoot(), maxQuestions, dryRun)
		if err != nil {
			outputError(1, err.Error(), nil)
			return nil
		}
		outputWorkflow(result, renderDiscussVisual(result))
		return nil
	},
}

func init() {
	discussCmd.Flags().Int("max-questions", 3, "Legacy compatibility flag; every material clarification is returned in one batch")
	discussCmd.Flags().Bool("dry-run", false, "Analyze and preview questions without writing pending decisions")
	discussCmd.Flags().String("resolve", "", "Clarification decision ID to resolve")
	discussCmd.Flags().String("answer", "", "Resolution text for --resolve")
	discussCmd.Flags().String("add-question", "", "Materialize a composed clarification question (Queen-composed intake)")
	discussCmd.Flags().String("options", "", "Pipe-separated answer options for --add-question (optional; empty = freeform)")
	discussCmd.Flags().String("category", "", "Category for --add-question: surface, integration, scope, verification, or analysis")
	discussCmd.Flags().String("grounding", "", "What the composed question is based on — REQUIRED with --add-question")
	discussCmd.Flags().Bool("hard", false, "Mark the composed question's answer as a hard constraint (REDIRECT on resolve)")
	discussCmd.Flags().String("source", "", "Stable dedup slug for --add-question (default derived from the question)")
	rootCmd.AddCommand(discussCmd)
}

// composedQuestionCategories is the closed category vocabulary for
// wrapper-composed questions. Closed on purpose: a typed category is what downstream
// suppression (clarificationSuppressedBySignals) and rendering key on.
var composedQuestionCategories = map[string]bool{
	"behavior": true, "authority": true, "risk_tolerance": true, "scope": true, "acceptance_meaning": true,
	// Legacy wrapper vocabulary remains accepted so already-integrated hosts
	// do not lose their exact intake path during the migration.
	"surface": true, "integration": true, "verification": true, "analysis": true,
}

// addComposedDiscussQuestion materializes ONE wrapper-composed question into
// pending-decisions.json — the typed intake behind the Queen-composed
// discuss flow. Grounding is refused when empty: a composed question must
// cite the scan fact or state datum it derives from. Dedup is by source
// slug, so re-running the composition never doubles questions.
func addComposedDiscussQuestion(question, optionsRaw, category, grounding, source string, hard bool) (map[string]interface{}, error) {
	state, err := loadActiveColonyState()
	if err != nil {
		return nil, fmt.Errorf("%s", colonyStateLoadMessage(err))
	}
	goal := strings.TrimSpace(derefGoal(state.Goal))
	if goal == "" {
		return nil, fmt.Errorf("the colony goal is empty; run `aether init \"goal\"` again before composing questions")
	}

	question = strings.TrimSpace(question)
	grounding = strings.TrimSpace(grounding)
	if grounding == "" {
		return nil, fmt.Errorf("--add-question requires --grounding: state what this question is based on (a scan fact, a survey finding, the goal's wording) — ungrounded questions are the canned-question problem again")
	}
	category = strings.ToLower(strings.TrimSpace(category))
	if category == "" {
		category = string(planningDecisionDomainBehavior)
	}
	if !composedQuestionCategories[category] {
		return nil, fmt.Errorf("unknown --category %q: must be one of behavior, authority, risk_tolerance, scope, acceptance_meaning, surface, integration, verification, or analysis", category)
	}
	domain := planningDecisionDomainForCategory(category)

	options := []string{}
	for _, opt := range strings.Split(optionsRaw, "|") {
		if opt = strings.TrimSpace(opt); opt != "" {
			options = append(options, opt)
		}
	}

	source = composedDiscussSource(domain, source, question)
	sourceBase := logicalDiscussSource(source)
	description := formatClarificationDescription(question, options)

	scope := pendingDecisionScopeFromState(state)
	pending := loadPendingDecisionFile()
	activePending, _ := filterPendingDecisionFileForScope(pending, scope)
	var priorResolved *PendingDecision
	for index := range activePending.Decisions {
		existing := activePending.Decisions[index]
		if existing.Type != clarificationDecisionType || logicalDiscussSource(existing.Source) != sourceBase {
			continue
		}
		if existing.Resolved {
			candidate := existing
			if priorResolved == nil || clarificationSortKey(candidate).After(clarificationSortKey(*priorResolved)) {
				priorResolved = &candidate
			}
		}
		if equivalentComposedDiscussDecision(existing, source, description, grounding, hard, domain) {
			status := "pending"
			if existing.Resolved {
				status = "already_resolved"
			}
			return map[string]interface{}{
				"created":               false,
				"id":                    existing.ID,
				"status":                status,
				"source":                existing.Source,
				"category":              string(domain),
				"question":              question,
				"grounding":             grounding,
				"revalidation_required": false,
			}, nil
		}
	}

	decision := PendingDecision{
		ID:             fmt.Sprintf("pd_%d", time.Now().UnixNano()),
		Type:           clarificationDecisionType,
		Description:    description,
		Source:         source,
		HardConstraint: hard,
		Grounding:      grounding,
		Resolved:       false,
		CreatedAt:      time.Now().UTC().Format(time.RFC3339),
	}
	stampPendingDecisionScope(&decision, scope)
	pending.Decisions = append(pending.Decisions, decision)
	if err := store.SaveJSON(pendingDecisionsFile, pending); err != nil {
		return nil, fmt.Errorf("failed to save composed question: %w", err)
	}

	return map[string]interface{}{
		"created":               true,
		"id":                    decision.ID,
		"status":                "new",
		"source":                source,
		"category":              string(domain),
		"question":              question,
		"options":               options,
		"grounding":             grounding,
		"hard":                  hard,
		"revalidation_required": priorResolved != nil,
		"prior_answer":          resolvedDecisionAnswer(priorResolved),
		"next":                  fmt.Sprintf("Resolve with `aether discuss --resolve %s --answer \"...\"` after the user picks.", decision.ID),
	}, nil
}

func planningDecisionDomainForCategory(category string) planningDecisionDomain {
	switch strings.ToLower(strings.TrimSpace(category)) {
	case "authority", "integration":
		return planningDecisionDomainAuthority
	case "risk_tolerance":
		return planningDecisionDomainRiskTolerance
	case "scope", "surface":
		return planningDecisionDomainScope
	case "acceptance_meaning", "verification":
		return planningDecisionDomainAcceptanceMeaning
	default:
		return planningDecisionDomainBehavior
	}
}

func composedDiscussSource(domain planningDecisionDomain, raw, question string) string {
	base := strings.TrimSpace(raw)
	base = strings.TrimPrefix(base, "wrapper:")
	parts := strings.Split(base, ":")
	if len(parts) > 1 && planningDecisionDomain(parts[0]).isMaterialDomain() {
		base = strings.Join(parts[1:], ":")
	}
	if base == "" {
		digest := sha256.Sum256([]byte(strings.ToLower(strings.Join(strings.Fields(question), " "))))
		base = "q-" + hex.EncodeToString(digest[:4])
	}
	return "wrapper:" + string(domain) + ":" + base
}

func logicalDiscussSource(source string) string {
	source = strings.TrimSpace(source)
	parts := strings.Split(source, ":")
	if len(parts) >= 3 && parts[0] == "wrapper" && planningDecisionDomain(parts[1]).isMaterialDomain() {
		return "wrapper:" + strings.Join(parts[2:], ":")
	}
	return source
}

func equivalentComposedDiscussDecision(existing PendingDecision, expectedSource, description, grounding string, hard bool, domain planningDecisionDomain) bool {
	if normalizePlanningDecisionText(existing.Description) != normalizePlanningDecisionText(description) ||
		normalizePlanningDecisionText(existing.Grounding) != normalizePlanningDecisionText(grounding) ||
		clarificationIsHardConstraint(existing) != hard {
		return false
	}
	if existingDomain := planningDecisionDomainForClarification(existing); existingDomain != domain {
		return false
	}
	return existing.Source == expectedSource || logicalDiscussSource(existing.Source) == logicalDiscussSource(expectedSource)
}

func resolvedDecisionAnswer(decision *PendingDecision) string {
	if decision == nil {
		return ""
	}
	return normalizePlanningDecisionText(decision.Resolution)
}

func renderComposedQuestionVisual(result map[string]interface{}) string {
	var b strings.Builder
	b.WriteString(renderBanner(commandEmoji("discuss"), "Question Composed"))
	b.WriteString(visualDividerStr())
	if !boolValue(result["created"]) {
		fmt.Fprintf(&b, "Already tracked (%s): %s\n", stringValue(result["status"]), stringValue(result["question"]))
		fmt.Fprintf(&b, "   └── id %s\n", stringValue(result["id"]))
		return b.String()
	}
	fmt.Fprintf(&b, "❓ %s\n", stringValue(result["question"]))
	if opts := stringSliceValue(result["options"]); len(opts) > 0 {
		fmt.Fprintf(&b, "   └── options: %s\n", strings.Join(opts, " | "))
	}
	fmt.Fprintf(&b, "   └── based on: %s\n", stringValue(result["grounding"]))
	if boolValue(result["hard"]) {
		b.WriteString("   └── hard constraint: the answer becomes a REDIRECT signal\n")
	}
	fmt.Fprintf(&b, "   └── id %s\n", stringValue(result["id"]))
	b.WriteString(renderNextUp(stringValue(result["next"])))
	return b.String()
}

func runDiscuss(root string, maxQuestions int, dryRun bool) (map[string]interface{}, error) {
	state, err := loadActiveColonyState()
	if err != nil {
		if errors.Is(err, errNoColonyInitialized) && store != nil {
			var raw colony.ColonyState
			if loadErr := store.LoadJSON("COLONY_STATE.json", &raw); loadErr == nil && raw.Goal != nil && strings.TrimSpace(*raw.Goal) == "" {
				return nil, fmt.Errorf("the colony goal is empty; run `aether init \"goal\"` again before discussing scope")
			}
		}
		return nil, fmt.Errorf("%s", colonyStateLoadMessage(err))
	}

	goal := strings.TrimSpace(derefGoal(state.Goal))
	if goal == "" {
		return nil, fmt.Errorf("the colony goal is empty; run `aether init \"goal\"` again before discussing scope")
	}

	survey, _ := loadCodexSurveyContext(root)
	scope := pendingDecisionScopeFromState(state)
	pending := loadPendingDecisionFile()
	activePending, stalePending := filterPendingDecisionFileForScope(pending, scope)
	staleClarificationCount := countClarifications(stalePending)
	activeSignals := activeSignalTexts()

	analyze := runDiscussAnalyze(root, goal)
	frontier, err := buildDiscussEvidenceFrontier(state, survey, analyze, activePending, activeSignals)
	if err != nil {
		return nil, err
	}
	questions, materialBatch, suppressedCount, err := materializeDiscussQuestions(activePending, &frontier, activeSignals)
	if err != nil {
		return nil, err
	}

	resolved := resolvedClarifiedIntentEntries(activePending)

	next := "Review the settled intent before creating its draft specification."
	if len(questions) > 0 {
		next = "Resolve every material card with its exact `aether discuss --resolve <id> --answer \"...\"` command."
	}
	staleNotice := pendingDecisionStaleNotice(staleClarificationCount)
	evidenceRecords := make([]planningEvidenceRecord, 0, len(frontier.Items))
	for _, item := range frontier.Items {
		evidenceRecords = append(evidenceRecords, item.Record)
	}

	var specificationCloseout *discussSpecificationCloseout
	if len(questions) == 0 {
		closeout, settleErr := settleDiscussSpecification(root, state, survey, analyze, activePending, frontier, activeSignals, dryRun)
		if settleErr != nil {
			return nil, settleErr
		}
		specificationCloseout = &closeout
		if closeout.WouldCreate {
			next = "Rerun `aether discuss` without --dry-run to create this exact draft, then run `aether spec` to review it."
		} else if closeout.ApprovedSpecPreserved {
			next = closeout.RevisionGuidance
		} else {
			next = "Run `aether spec` to review, revise, or explicitly approve this exact draft."
		}
	}

	result := map[string]interface{}{
		"goal":                      goal,
		"question_count":            len(questions),
		"created_count":             0,
		"existing_count":            len(questions),
		"dry_run":                   dryRun,
		"survey_docs":               survey.SurveyDocs,
		"evidence_count":            len(evidenceRecords),
		"evidence_frontier":         evidenceRecords,
		"evidence_suppressed_count": suppressedCount,
		"questions":                 questions,
		"material_batch":            materialBatch,
		"resolved":                  resolved,
		"resolved_count":            len(resolved),
		"pending_count":             len(questions),
		"stored_pending_count":      countPendingClarifications(activePending),
		"ignored_stale_count":       staleClarificationCount,
		"quarantined_stale_count":   staleClarificationCount,
		"stale_state_notice":        staleNotice,
		"signal_count":              len(activeSignals),
		"survey_available":          len(survey.SurveyDocs) > 0 || len(survey.Frameworks) > 0 || len(survey.Directories) > 0,
		"next":                      next,
		"discussion_status":         discussionStatus(len(questions), 0, len(questions)),
	}
	closeOverride := ""
	closeWhy := ""
	if specificationCloseout != nil {
		result["specification"] = *specificationCloseout
		result["specification_status"] = specificationCloseout.Status
		result["unresolved_count"] = specificationCloseout.UnresolvedCount
		result["projection_path"] = specificationCloseout.ProjectionPath
		result["exact_next_command"] = specificationCloseout.ExactNextCommand
		if specificationCloseout.ApprovedSpecPreserved {
			result["approved_spec"] = *specificationCloseout
		} else {
			result["draft_spec"] = *specificationCloseout
		}
		if specificationCloseout.WouldCreate {
			closeOverride = "aether discuss"
			closeWhy = "This was a dry run. Create the reviewed draft before opening the specification lifecycle."
		} else {
			closeOverride = specificationCloseout.ExactNextCommand
			closeWhy = "Settled intent now belongs to the specification lifecycle; planning remains unauthorized until the exact contract is approved."
		}
	}
	closeLifecycleCommand(result, "discuss", closeOverride, closeWhy)
	return result, nil
}

func settleDiscussSpecification(
	root string,
	state colony.ColonyState,
	survey codexSurveyContext,
	analyze analyzeScanData,
	pending PendingDecisionFile,
	frontier discussEvidenceFrontier,
	activeSignals []string,
	dryRun bool,
) (discussSpecificationCloseout, error) {
	repositoryRoot, err := canonicalSpecificationRoot(root)
	if err != nil {
		return discussSpecificationCloseout{}, fmt.Errorf("settle discuss specification: %w", err)
	}

	if state.Specification != nil {
		current, ok := currentSpecificationRevision(*state.Specification)
		if !ok {
			return discussSpecificationCloseout{}, fmt.Errorf("settle discuss specification: existing specification has no current revision")
		}
		inspection, inspectErr := inspectSpecificationProjection(repositoryRoot)
		if inspectErr != nil {
			return discussSpecificationCloseout{}, fmt.Errorf("inspect settled specification projection: %w", inspectErr)
		}
		projectionRepaired := false
		if current.Status == colony.SpecStatusDraft && inspection.Drifted && !dryRun {
			repair, repairErr := repairSpecificationProjection(repositoryRoot, specificationMutationOptions{})
			if repairErr != nil {
				return discussSpecificationCloseout{}, fmt.Errorf("render settled specification projection: %w", repairErr)
			}
			inspection = repair.Inspection
			projectionRepaired = repair.Repaired
		}
		approvedPreserved := current.Status == colony.SpecStatusApproved
		return newDiscussSpecificationCloseout(
			*state.Specification,
			current,
			nil,
			current.Status == colony.SpecStatusDraft,
			approvedPreserved,
			false,
			inspection,
			projectionRepaired,
		), nil
	}

	request, err := buildSettledDiscussDraftRequest(state, survey, analyze, pending, frontier, activeSignals, time.Now().UTC())
	if err != nil {
		return discussSpecificationCloseout{}, err
	}
	if dryRun {
		specification, revision, buildErr := buildSpecificationDraft(request)
		if buildErr != nil {
			return discussSpecificationCloseout{}, fmt.Errorf("preview settled specification draft: %w", buildErr)
		}
		projection, renderErr := renderSpecificationProjection(specification, state.Plan)
		if renderErr != nil {
			return discussSpecificationCloseout{}, fmt.Errorf("preview settled specification projection: %w", renderErr)
		}
		inspection := specificationProjectionInspection{
			Path:           filepath.Join(repositoryRoot, specificationProjectionRelativePath),
			RevisionID:     revision.ID,
			ExpectedDigest: lifecycleDigest(projection),
			Missing:        true,
			Drifted:        true,
		}
		return newDiscussSpecificationCloseout(specification, revision, nil, false, false, true, inspection, false), nil
	}

	mutation, err := createSpecificationDraft(repositoryRoot, request, specificationMutationOptions{})
	if err != nil {
		return discussSpecificationCloseout{}, fmt.Errorf("create settled specification draft: %w", err)
	}
	inspection, err := inspectSpecificationProjection(repositoryRoot)
	if err != nil {
		return discussSpecificationCloseout{}, fmt.Errorf("inspect created specification projection: %w", err)
	}
	receipt := mutation.Receipt
	return newDiscussSpecificationCloseout(
		mutation.Specification,
		mutation.Revision,
		&receipt,
		mutation.Replayed,
		false,
		false,
		inspection,
		false,
	), nil
}

func buildSettledDiscussDraftRequest(
	state colony.ColonyState,
	survey codexSurveyContext,
	analyze analyzeScanData,
	pending PendingDecisionFile,
	frontier discussEvidenceFrontier,
	activeSignals []string,
	createdAt time.Time,
) (specificationDraftRequest, error) {
	goal := strings.TrimSpace(derefGoal(state.Goal))
	if state.AcceptedCharter != nil && strings.TrimSpace(state.AcceptedCharter.Goal) != "" {
		goal = strings.TrimSpace(state.AcceptedCharter.Goal)
	}
	if goal == "" {
		return specificationDraftRequest{}, fmt.Errorf("settled discussion cannot create a specification without an accepted goal")
	}
	allEvidence := discussSpecificationEvidenceIDs(frontier, "", "")
	if len(allEvidence) == 0 {
		return specificationDraftRequest{}, fmt.Errorf("settled discussion cannot create a specification without admissible evidence")
	}
	charterEvidence := discussSpecificationEvidenceIDs(frontier, "", "state:")
	if len(charterEvidence) == 0 {
		charterEvidence = discussSpecificationEvidenceIDs(frontier, string(colony.PlanningEvidenceCharter), "")
	}
	charterEvidence = discussSpecificationEvidenceFallback(charterEvidence, allEvidence)
	contextEvidence := discussSpecificationEvidenceFallback(
		discussSpecificationEvidenceIDs(frontier, string(colony.PlanningEvidenceContext), ""),
		allEvidence,
	)
	surveyEvidence := discussSpecificationEvidenceFallback(
		discussSpecificationEvidenceIDs(frontier, string(colony.PlanningEvidenceSurvey), ""),
		contextEvidence,
	)

	request := specificationDraftRequest{
		Scope: colony.SpecScope{
			Kind:      colony.SpecScopeWholeGoal,
			GoalID:    frontier.Scope.GoalID,
			SessionID: frontier.Scope.SessionID,
		},
		CreatedAt: createdAt,
	}
	if strings.TrimSpace(request.Scope.GoalID) == "" || strings.TrimSpace(request.Scope.SessionID) == "" {
		return specificationDraftRequest{}, fmt.Errorf("settled discussion lacks current goal/session identity")
	}

	charter := state.Charter
	if state.AcceptedCharter != nil && state.AcceptedCharter.Charter != nil {
		charter = state.AcceptedCharter.Charter
	}
	add := func(destination *[]specificationItemInput, lineage, description, verification, publicPath string, evidence []string) {
		description = specificationProjectionText(description)
		if description == "" {
			return
		}
		for _, existing := range *destination {
			if specificationProjectionText(existing.Description) == description {
				return
			}
		}
		*destination = append(*destination, specificationItemInput{
			Lineage:      lineage,
			Description:  description,
			Verification: specificationProjectionText(verification),
			Path:         filepath.ToSlash(strings.TrimSpace(publicPath)),
			EvidenceIDs:  append([]string(nil), evidence...),
		})
	}

	add(&request.Outcomes, "accepted-goal", "Deliver the accepted goal: "+goal, "", "", charterEvidence)
	if charter != nil {
		add(&request.Outcomes, "charter-vision", charter.Vision, "", "", charterEvidence)
		included := charter.Intent
		if strings.TrimSpace(included) == "" {
			included = charter.Goals
		}
		add(&request.IncludedBehaviors, "charter-included-behavior", included, "", "", charterEvidence)
		add(&request.IncludedBehaviors, "charter-goals", charter.Goals, "", "", charterEvidence)
		if strings.TrimSpace(charter.TechStack) != "" {
			add(&request.BindingDecisions, "charter-technology-context", "Use the accepted implementation context: "+charter.TechStack, "", "", charterEvidence)
		}
		add(&request.BindingDecisions, "charter-governance", charter.Governance, "", "", charterEvidence)
		if strings.TrimSpace(charter.Constraints) != "" {
			add(&request.Exclusions, "charter-explicit-boundary", "Work outside these accepted constraints is excluded: "+charter.Constraints, "", "", charterEvidence)
			add(&request.NegativeExpectations, "charter-constraint-negative", "The result must not violate these accepted constraints: "+charter.Constraints, "", "", charterEvidence)
		}
		if strings.TrimSpace(charter.KeyRisks) != "" {
			add(&request.NegativeExpectations, "charter-known-risk", "The result must not silently realize this known risk: "+charter.KeyRisks, "", "", charterEvidence)
			add(&request.RecoveryExpectations, "charter-risk-recovery", "If the known risk occurs, preserve the last valid state and report it before continuing: "+charter.KeyRisks, "", "", charterEvidence)
		}
	}
	if len(request.IncludedBehaviors) == 0 {
		add(&request.IncludedBehaviors, "accepted-goal-behavior", "Implement only behavior needed to satisfy the accepted goal: "+goal, "", "", charterEvidence)
	}
	if len(request.Exclusions) == 0 {
		add(&request.Exclusions, "outside-accepted-goal", "Behavior outside the accepted goal is excluded unless the owner revises this specification.", "", "", charterEvidence)
	}

	resolved := append([]PendingDecision(nil), pending.Decisions...)
	sort.SliceStable(resolved, func(left, right int) bool { return resolved[left].ID < resolved[right].ID })
	for _, decision := range resolved {
		if decision.Type != clarificationDecisionType || !decision.Resolved || strings.TrimSpace(decision.Resolution) == "" {
			continue
		}
		question, _ := parseClarificationDescription(decision.Description)
		answer := specificationProjectionText(decision.Resolution)
		decisionText := specificationProjectionText(question + " — Owner decision: " + answer)
		lineage := discussSpecificationDecisionLineage(decision)
		decisionEvidence := discussSpecificationEvidenceFallback(
			discussSpecificationEvidenceIDs(frontier, string(colony.PlanningEvidenceDecision), "decision:"+decision.ID),
			allEvidence,
		)
		add(&request.BindingDecisions, lineage, decisionText, "", "", decisionEvidence)
		switch planningDecisionDomainForClarification(decision) {
		case planningDecisionDomainBehavior:
			add(&request.IncludedBehaviors, lineage, "Settled behavior: "+decisionText, "", "", decisionEvidence)
		case planningDecisionDomainScope:
			add(&request.Exclusions, lineage, "Settled scope boundary: "+decisionText, "", "", decisionEvidence)
		case planningDecisionDomainAcceptanceMeaning:
			add(&request.AcceptanceChecks, lineage, "Owner-checkable acceptance: "+decisionText, "Review the delivered behavior and its evidence against this exact accepted answer.", "", decisionEvidence)
		case planningDecisionDomainRiskTolerance:
			add(&request.NegativeExpectations, lineage, "Settled risk boundary: "+decisionText, "", "", decisionEvidence)
			add(&request.RecoveryExpectations, lineage, "If the settled risk boundary is crossed, preserve the last valid state and return to the owner: "+answer, "", "", decisionEvidence)
		}
		if clarificationIsHardConstraint(decision) {
			add(&request.NegativeExpectations, lineage+"-hard", "The result must not violate this explicit owner constraint: "+decisionText, "", "", decisionEvidence)
		}
	}

	for index, signal := range uniqueSortedStrings(activeSignals) {
		if index >= 8 {
			break
		}
		add(&request.BindingDecisions, fmt.Sprintf("active-owner-constraint-%02d", index+1), "Active owner constraint: "+signal, "", "", contextEvidence)
		add(&request.NegativeExpectations, fmt.Sprintf("active-owner-constraint-%02d", index+1), "The result must not violate this active owner constraint: "+signal, "", "", contextEvidence)
	}

	requirementText := goal
	if charter != nil && strings.TrimSpace(charter.Goals) != "" {
		requirementText = charter.Goals
	}
	add(&request.Requirements, "accepted-goal-requirement", "Required result: "+requirementText, "", "", charterEvidence)
	if len(request.BindingDecisions) == 0 {
		add(&request.BindingDecisions, "explicit-specification-authority", "The accepted goal and current evidence define this draft; only the owner may approve or revise it.", "", "", allEvidence)
	}
	if len(request.AcceptanceChecks) == 0 {
		add(&request.AcceptanceChecks, "owner-verifies-accepted-goal", "The owner can verify that the delivered behavior satisfies the accepted goal: "+goal, "Review the delivered behavior and its verification evidence against the accepted goal.", "", charterEvidence)
	}
	if len(request.NegativeExpectations) == 0 {
		add(&request.NegativeExpectations, "no-unapproved-scope", "The result must not add behavior outside the owner-approved specification.", "", "", allEvidence)
	}
	if len(request.RecoveryExpectations) == 0 {
		add(&request.RecoveryExpectations, "preserve-last-valid-state", "If the accepted result cannot be delivered safely, preserve the last valid state and report the blocker before continuing.", "", "", allEvidence)
	}

	paths := uniqueSortedStrings(survey.EntryPoints)
	if len(paths) == 0 {
		paths = []string{specificationProjectionRelativePath}
	}
	for index, path := range paths {
		if index >= 12 {
			break
		}
		evidence := surveyEvidence
		if path == specificationProjectionRelativePath {
			evidence = contextEvidence
		}
		add(&request.AffectedPublicPaths, "known-public-path-"+path, "Known owner-visible or public path affected by this contract: "+path, "", path, evidence)
	}
	if len(request.AffectedPublicPaths) == 0 {
		add(&request.AffectedPublicPaths, "specification-projection", "The owner-readable specification projection created by settled discussion.", "", specificationProjectionRelativePath, contextEvidence)
	}

	_ = analyze // The inventory is already bound into current-context evidence.
	return request, nil
}

// discussSpecificationDecisionLineage preserves established short decision
// identities while bounding generated IDs for both the base specification item
// and its optional "-hard" negative-expectation companion. The hash input is
// domain-separated so the compact identity cannot be confused with another
// digest use elsewhere in the planning lifecycle.
func discussSpecificationDecisionLineage(decision PendingDecision) string {
	identity := emptyFallback(strings.TrimSpace(decision.ID), logicalDiscussSource(decision.Source))
	lineage := "owner-decision-" + identity
	if _, err := colony.CanonicalSpecItemID(colony.SpecSectionBindingDecisions, lineage); err == nil {
		if _, hardErr := colony.CanonicalSpecItemID(colony.SpecSectionNegativeExpectations, lineage+"-hard"); hardErr == nil {
			return lineage
		}
	}

	digest := sha256.Sum256([]byte("aether/discuss/specification-decision-lineage/v1\x00" + identity))
	return "owner-decision-" + hex.EncodeToString(digest[:8])
}

func discussSpecificationEvidenceIDs(frontier discussEvidenceFrontier, kind, originPrefix string) []string {
	ids := []string{}
	for _, item := range frontier.Items {
		reference := item.Record.Reference
		if kind != "" && string(reference.Kind) != kind {
			continue
		}
		if originPrefix != "" && !strings.HasPrefix(reference.Origin, originPrefix) {
			continue
		}
		if strings.TrimSpace(reference.ID) != "" {
			ids = append(ids, reference.ID)
		}
	}
	return uniqueSortedStrings(ids)
}

func discussSpecificationEvidenceFallback(preferred, fallback []string) []string {
	if len(preferred) > 0 {
		return append([]string(nil), preferred...)
	}
	return append([]string(nil), fallback...)
}

func newDiscussSpecificationCloseout(
	specification colony.Specification,
	revision colony.SpecRevision,
	receipt *colony.LifecycleReceipt,
	replayed bool,
	approvedPreserved bool,
	wouldCreate bool,
	projection specificationProjectionInspection,
	projectionRepaired bool,
) discussSpecificationCloseout {
	body := discussSpecificationBody{
		Outcomes:             revision.Outcomes,
		IncludedBehaviors:    revision.IncludedBehaviors,
		Exclusions:           revision.Exclusions,
		BindingDecisions:     revision.BindingDecisions,
		Requirements:         revision.Requirements,
		AcceptanceChecks:     revision.AcceptanceChecks,
		NegativeExpectations: revision.NegativeExpectations,
		RecoveryExpectations: revision.RecoveryExpectations,
		AffectedPublicPaths:  revision.AffectedPublicPaths,
	}
	closeout := discussSpecificationCloseout{
		SpecificationID: specification.ID,
		RevisionID:      revision.ID,
		RevisionNumber:  specificationRevisionIndex(specification, revision.ID) + 1,
		ContentHash:     revision.ContentHash,
		Status:          revision.Status,
		Scope:           revision.Scope,
		SectionCounts: map[string]int{
			"outcomes":              len(revision.Outcomes),
			"included_behaviors":    len(revision.IncludedBehaviors),
			"exclusions":            len(revision.Exclusions),
			"binding_decisions":     len(revision.BindingDecisions),
			"requirements":          len(revision.Requirements),
			"acceptance_checks":     len(revision.AcceptanceChecks),
			"negative_expectations": len(revision.NegativeExpectations),
			"recovery_expectations": len(revision.RecoveryExpectations),
			"affected_public_paths": len(revision.AffectedPublicPaths),
		},
		Body:                  body,
		UnresolvedCount:       0,
		ProjectionPath:        specificationProjectionRelativePath,
		ProjectionDigest:      projection.ExpectedDigest,
		ProjectionRepaired:    projectionRepaired,
		ExactNextCommand:      "aether spec",
		Replayed:              replayed,
		WouldCreate:           wouldCreate,
		ApprovedSpecPreserved: approvedPreserved,
		Receipt:               receipt,
	}
	if revision.Status == colony.SpecStatusDraft {
		closeout.ApprovalCommand = fmt.Sprintf(
			"aether spec --approve --revision-id %s --revision-hash %s --approval-token '%s'",
			revision.ID,
			revision.ContentHash,
			specificationApprovalToken(specification.ID, revision.ID, revision.ContentHash),
		)
	}
	if approvedPreserved {
		closeout.RevisionGuidance = fmt.Sprintf(
			"Run `aether spec` to inspect approved revision %s (%s). Any material change requires an explicit scoped successor through `aether spec --add`, `aether spec --modify`, or `aether spec --remove`; `aether discuss` will not replace it.",
			revision.ID,
			revision.ContentHash,
		)
	}
	return closeout
}

func resolveDiscussQuestion(id, answer string) (map[string]interface{}, error) {
	if strings.TrimSpace(id) == "" {
		return nil, fmt.Errorf("--resolve requires a clarification decision ID")
	}
	answer = strings.TrimSpace(answer)
	if answer == "" {
		return nil, fmt.Errorf("--answer is required when resolving a clarification")
	}

	file := loadPendingDecisionFile()
	scope := loadCurrentPendingDecisionScope()
	found := -1
	for i := range file.Decisions {
		if file.Decisions[i].ID == id {
			if !pendingDecisionMatchesScope(file.Decisions[i], scope) {
				return nil, fmt.Errorf("clarification %q is stale for the current goal/session; run `aether discuss` to create or resolve current-goal clarifications", id)
			}
			found = i
			break
		}
	}
	if found == -1 {
		return nil, fmt.Errorf("clarification %q not found", id)
	}
	if file.Decisions[found].Type != clarificationDecisionType {
		return nil, fmt.Errorf("decision %q is not a clarification", id)
	}

	now := time.Now().UTC().Format(time.RFC3339)
	file.Decisions[found].Resolved = true
	file.Decisions[found].Resolution = answer
	file.Decisions[found].ResolvedAt = now
	stampPendingDecisionScope(&file.Decisions[found], scope)
	if err := store.SaveJSON(pendingDecisionsFile, file); err != nil {
		return nil, fmt.Errorf("failed to save clarification resolution: %w", err)
	}

	redirectEmitted := false
	redirectText := ""
	if clarificationIsHardConstraint(file.Decisions[found]) {
		redirectText = buildClarificationRedirect(file.Decisions[found], answer)
		if _, err := createPheromoneSignal("REDIRECT", redirectText, "discuss", "resolved clarification", "", 1.0, "high"); err != nil {
			return nil, fmt.Errorf("resolved clarification but failed to emit redirect: %w", err)
		}
		redirectEmitted = true
	}

	// Detect contradictory resolved decisions and emit FEEDBACK pheromones.
	// Non-blocking: errors from pheromone creation are silently ignored.
	conflicts := detectDecisionConflicts(file.Decisions)
	for _, conflict := range conflicts {
		_, _ = createPheromoneSignal("FEEDBACK", conflict, "discuss", "decision conflict detection", "", 0.5, "low")
	}

	if tracer != nil {
		var state colony.ColonyState
		if loadErr := store.LoadJSON("COLONY_STATE.json", &state); loadErr == nil && state.RunID != nil {
			_ = tracer.LogIntervention(*state.RunID, "discuss.resolved", "discuss", map[string]interface{}{
				"decisions":        1,
				"redirect_emitted": redirectEmitted,
			})
		}
	}

	activeFile, _ := filterPendingDecisionFileForScope(file, scope)
	remaining := countPendingClarifications(activeFile)
	next := "Run `aether discuss` to review the remaining material questions."
	override := ""
	var settled map[string]interface{}
	if remaining == 0 {
		if _, stateErr := loadActiveColonyState(); stateErr != nil {
			// Legacy pending-decision files can outlive the colony state that
			// created them. Preserve their exact resolution behavior, but do not
			// pretend a specification can be created without a current goal.
			next = "Run `aether init \"goal\"` before creating a draft specification for this resolved answer."
		} else {
			root := repoRootFromStore(store)
			settledResult, settleErr := runDiscuss(root, 3, false)
			if settleErr != nil {
				return nil, fmt.Errorf("clarification resolved, but draft specification handoff failed: %w", settleErr)
			}
			settled = settledResult
			next = stringValue(settled["next"])
			override = stringValue(settled["exact_next_command"])
			if override == "" {
				override = "aether spec"
			}
		}
	}

	result := map[string]interface{}{
		"resolved":         true,
		"id":               id,
		"answer":           answer,
		"redirect_emitted": redirectEmitted,
		"redirect_text":    redirectText,
		"remaining":        remaining,
		"next":             next,
	}
	if settled != nil {
		for _, key := range []string{
			"discussion_status", "specification", "specification_status", "draft_spec", "approved_spec",
			"unresolved_count", "projection_path", "exact_next_command",
		} {
			if value, ok := settled[key]; ok {
				result[key] = value
			}
		}
	}
	closeLifecycleCommand(result, "discuss", override,
		"The last material answer is now captured in a draft specification. Review that exact contract before planning can begin.")
	return result, nil
}

func buildDiscussEvidenceFrontier(state colony.ColonyState, survey codexSurveyContext, analyze analyzeScanData, pending PendingDecisionFile, activeSignals []string) (discussEvidenceFrontier, error) {
	goalID := pendingDecisionGoalHash(derefGoal(state.Goal))
	if goalID == "" {
		return discussEvidenceFrontier{}, fmt.Errorf("cannot build discuss evidence without a goal identity")
	}
	sessionID := "session-" + goalID[:16]
	if state.SessionID != nil && strings.TrimSpace(*state.SessionID) != "" {
		sessionID = strings.TrimSpace(*state.SessionID)
	}
	specificationRevisionID := "specification-unsettled"
	if state.Specification != nil && strings.TrimSpace(state.Specification.CurrentRevisionID) != "" {
		specificationRevisionID = strings.TrimSpace(state.Specification.CurrentRevisionID)
	}
	planRevisionID := strings.TrimSpace(state.Plan.ActiveRevisionID)
	if planRevisionID == "" {
		planRevisionID = "plan-unbound"
	}
	frontier := discussEvidenceFrontier{Scope: planningEvidenceScope{
		GoalID:                  goalID,
		SessionID:               sessionID,
		SpecificationRevisionID: specificationRevisionID,
		PlanRevisionID:          planRevisionID,
	}}
	observedAt := discussEvidenceObservedAt(state)
	add := func(kind colony.PlanningEvidenceKind, origin string, content []byte, observed time.Time, dimensions []colony.PlanningDimension) error {
		if len(strings.TrimSpace(string(content))) == 0 {
			return nil
		}
		record, err := normalizePlanningEvidence(planningEvidenceSource{
			Kind:                 kind,
			Origin:               origin,
			Content:              content,
			Scope:                frontier.Scope,
			SourceRevision:       discussEvidenceSourceRevision(origin, content),
			ObservedAt:           observed,
			ApplicableDimensions: dimensions,
			State:                planningEvidenceSourceCurrent,
		})
		if err != nil {
			return fmt.Errorf("build discuss evidence %s: %w", origin, err)
		}
		frontier.Items = append(frontier.Items, discussEvidenceItem{Record: record, Content: string(content)})
		return nil
	}

	if state.AcceptedCharter != nil {
		content, err := json.Marshal(state.AcceptedCharter)
		if err != nil {
			return discussEvidenceFrontier{}, fmt.Errorf("encode accepted charter evidence: %w", err)
		}
		charterObservedAt := state.AcceptedCharter.AcceptedAt
		if charterObservedAt.IsZero() {
			charterObservedAt = observedAt
		}
		if err := add(colony.PlanningEvidenceCharter, "state:accepted-charter", content, charterObservedAt, []colony.PlanningDimension{
			colony.PlanningDimensionKnowledge,
			colony.PlanningDimensionRequirements,
			colony.PlanningDimensionRisks,
		}); err != nil {
			return discussEvidenceFrontier{}, err
		}
	} else if state.Charter != nil {
		content, err := json.Marshal(state.Charter)
		if err != nil {
			return discussEvidenceFrontier{}, fmt.Errorf("encode charter evidence: %w", err)
		}
		if err := add(colony.PlanningEvidenceCharter, "state:charter", content, observedAt, []colony.PlanningDimension{
			colony.PlanningDimensionKnowledge,
			colony.PlanningDimensionRequirements,
			colony.PlanningDimensionRisks,
		}); err != nil {
			return discussEvidenceFrontier{}, err
		}
	}

	if discussSurveyHasEvidence(survey) {
		content, err := json.Marshal(survey)
		if err != nil {
			return discussEvidenceFrontier{}, fmt.Errorf("encode survey evidence: %w", err)
		}
		if err := add(colony.PlanningEvidenceSurvey, "survey:current-workspace", content, observedAt, []colony.PlanningDimension{
			colony.PlanningDimensionKnowledge,
			colony.PlanningDimensionRisks,
			colony.PlanningDimensionDependencies,
			colony.PlanningDimensionEffort,
		}); err != nil {
			return discussEvidenceFrontier{}, err
		}
	}

	contextContent, err := json.Marshal(struct {
		Goal          string   `json:"goal"`
		Signals       []string `json:"owner_constraints,omitempty"`
		DetectedType  string   `json:"detected_type,omitempty"`
		Languages     []string `json:"languages,omitempty"`
		Frameworks    []string `json:"frameworks,omitempty"`
		TopLevelDirs  []string `json:"top_level_dirs,omitempty"`
		TestFramework []string `json:"test_frameworks,omitempty"`
	}{
		Goal:          strings.TrimSpace(derefGoal(state.Goal)),
		Signals:       append([]string(nil), activeSignals...),
		DetectedType:  analyze.DetectedType,
		Languages:     append([]string(nil), analyze.Languages...),
		Frameworks:    append([]string(nil), analyze.Frameworks...),
		TopLevelDirs:  append([]string(nil), analyze.TopLevelDirs...),
		TestFramework: append([]string(nil), analyze.Governance.TestFrameworks...),
	})
	if err != nil {
		return discussEvidenceFrontier{}, fmt.Errorf("encode current context evidence: %w", err)
	}
	if err := add(colony.PlanningEvidenceContext, "context:current-goal", contextContent, observedAt, []colony.PlanningDimension{
		colony.PlanningDimensionKnowledge,
		colony.PlanningDimensionRisks,
		colony.PlanningDimensionDependencies,
		colony.PlanningDimensionEffort,
	}); err != nil {
		return discussEvidenceFrontier{}, err
	}

	for _, decision := range pending.Decisions {
		if decision.Type != clarificationDecisionType || !decision.Resolved || strings.TrimSpace(decision.Resolution) == "" {
			continue
		}
		content, err := json.Marshal(struct {
			Question   string `json:"question"`
			Answer     string `json:"answer"`
			Grounding  string `json:"grounding,omitempty"`
			Source     string `json:"source"`
			ResolvedAt string `json:"resolved_at,omitempty"`
		}{
			Question:   decision.Description,
			Answer:     decision.Resolution,
			Grounding:  decision.Grounding,
			Source:     decision.Source,
			ResolvedAt: decision.ResolvedAt,
		})
		if err != nil {
			return discussEvidenceFrontier{}, fmt.Errorf("encode prior decision evidence: %w", err)
		}
		answerObservedAt := observedAt
		if parsed, parseErr := time.Parse(time.RFC3339, strings.TrimSpace(decision.ResolvedAt)); parseErr == nil {
			answerObservedAt = parsed
		}
		if err := add(colony.PlanningEvidenceDecision, "decision:"+decision.ID, content, answerObservedAt, []colony.PlanningDimension{
			colony.PlanningDimensionKnowledge,
			colony.PlanningDimensionRequirements,
			colony.PlanningDimensionRisks,
			colony.PlanningDimensionDependencies,
			colony.PlanningDimensionEffort,
		}); err != nil {
			return discussEvidenceFrontier{}, err
		}
	}

	sort.Slice(frontier.Items, func(left, right int) bool {
		return frontier.Items[left].Record.Reference.ID < frontier.Items[right].Record.Reference.ID
	})
	return frontier, nil
}

func discussEvidenceObservedAt(state colony.ColonyState) time.Time {
	if state.InitializedAt != nil && !state.InitializedAt.IsZero() {
		return state.InitializedAt.UTC()
	}
	return time.Date(1970, time.January, 1, 0, 0, 0, 0, time.UTC)
}

func discussEvidenceSourceRevision(origin string, content []byte) string {
	sum := sha256.Sum256(append([]byte(strings.TrimSpace(origin)+"\x00"), content...))
	return "discuss-" + hex.EncodeToString(sum[:8])
}

func discussSurveyHasEvidence(survey codexSurveyContext) bool {
	return len(survey.SurveyDocs) > 0 || len(survey.Languages) > 0 || len(survey.Frameworks) > 0 ||
		len(survey.Directories) > 0 || len(survey.EntryPoints) > 0 || len(survey.Dependencies) > 0 ||
		len(survey.TestFiles) > 0 || len(survey.Issues) > 0 || len(survey.SecurityPatterns) > 0 || len(survey.SourceAnchors) > 0
}

func materializeDiscussQuestions(activePending PendingDecisionFile, frontier *discussEvidenceFrontier, activeSignals []string) ([]discussQuestion, *discussMaterialBatch, int, error) {
	latest := make(map[string]PendingDecision)
	for _, decision := range activePending.Decisions {
		if decision.Type != clarificationDecisionType || decision.Resolved || strings.TrimSpace(decision.Description) == "" {
			continue
		}
		key := logicalDiscussSource(decision.Source)
		if key == "" {
			key = decision.ID
		}
		if existing, ok := latest[key]; !ok || clarificationSortKey(decision).After(clarificationSortKey(existing)) {
			latest[key] = decision
		}
	}

	questions := make([]discussQuestion, 0, len(latest))
	suppressed := 0
	for _, decision := range latest {
		candidate, evidenceItems, err := planningDecisionCandidateForClarification(decision, frontier)
		if err != nil {
			return nil, nil, suppressed, err
		}
		category := string(candidate.Domain)
		if clarificationSuppressedBySignals(category, activeSignals) {
			suppressed++
			continue
		}
		candidate.AnswerableEvidenceIDs = discussAnswerableEvidenceIDs(decision, evidenceItems)
		classification, err := classifyPlanningDecision(candidate)
		if err != nil {
			return nil, nil, suppressed, fmt.Errorf("classify discuss decision %s: %w", decision.ID, err)
		}
		if !classification.RequiresOwner {
			suppressed++
			continue
		}

		prior := priorPlanningDecisionAnswer(activePending, decision, candidate, frontier.Scope)
		card, err := projectPlanningDecisionCard(planningDecisionCardRequest{
			Candidate: candidate,
			Scope: planningDecisionEquivalenceScope{
				GoalID:                          frontier.Scope.GoalID,
				SessionID:                       frontier.Scope.SessionID,
				ApprovedSpecificationRevisionID: frontier.Scope.SpecificationRevisionID,
				BasePlanRevisionID:              frontier.Scope.PlanRevisionID,
			},
			Prior: prior,
		})
		if err != nil {
			return nil, nil, suppressed, fmt.Errorf("project discuss decision %s: %w", decision.ID, err)
		}
		question, options := parseClarificationDescription(decision.Description)
		questions = append(questions, discussQuestion{
			ID:                  decision.ID,
			StableID:            card.DecisionID,
			Category:            category,
			Domain:              candidate.Domain,
			Question:            question,
			Decision:            question,
			Options:             options,
			Reasoning:           card.WhyNow,
			WhyNow:              card.WhyNow,
			Evidence:            card.Evidence,
			QueenRecommendation: card.QueenRecommendation,
			Choices:             card.Choices,
			AffectedSemanticIDs: card.AffectedSemanticIDs,
			PriorAnswer:         card.PriorAnswer,
			Revalidation:        card.Revalidation,
			PlanningResumes:     card.PlanningResumes,
			ExactAnswerSyntax:   fmt.Sprintf("aether discuss --resolve %s --answer \"<answer>\"", decision.ID),
			HardConstraint:      clarificationIsHardConstraint(decision),
			Status:              "pending",
			Source:              decision.Source,
		})
	}
	sort.Slice(questions, func(left, right int) bool {
		leftRank := planningDecisionDomainRank(questions[left].Domain)
		rightRank := planningDecisionDomainRank(questions[right].Domain)
		if leftRank != rightRank {
			return leftRank < rightRank
		}
		return questions[left].StableID < questions[right].StableID
	})
	if len(questions) == 0 {
		return questions, nil, suppressed, nil
	}
	contentHash, err := jsonSHA256(struct {
		GoalID    string            `json:"goal_id"`
		SessionID string            `json:"session_id"`
		Cards     []discussQuestion `json:"cards"`
	}{GoalID: frontier.Scope.GoalID, SessionID: frontier.Scope.SessionID, Cards: questions})
	if err != nil {
		return nil, nil, suppressed, fmt.Errorf("hash discuss material batch: %w", err)
	}
	return questions, &discussMaterialBatch{
		ID:          "discuss-material-batch-" + contentHash[:16],
		ContentHash: contentHash,
		Cards:       questions,
	}, suppressed, nil
}

func planningDecisionCandidateForClarification(decision PendingDecision, frontier *discussEvidenceFrontier) (planningDecisionCandidate, []discussEvidenceItem, error) {
	domain := planningDecisionDomainForClarification(decision)
	question, options := parseClarificationDescription(decision.Description)
	if strings.TrimSpace(question) == "" {
		return planningDecisionCandidate{}, nil, fmt.Errorf("clarification %s has no question text", decision.ID)
	}
	baseEvidence := append([]discussEvidenceItem(nil), frontier.Items...)
	grounding := strings.TrimSpace(decision.Grounding)
	if grounding == "" {
		grounding = "This current-goal clarification was already pending and remains an unresolved owner boundary."
	}
	groundingRecord, err := normalizePlanningEvidence(planningEvidenceSource{
		Kind:                 colony.PlanningEvidenceContext,
		Origin:               "context:clarification:" + emptyFallback(decision.ID, logicalDiscussSource(decision.Source)),
		Content:              []byte(grounding),
		Scope:                frontier.Scope,
		SourceRevision:       discussEvidenceSourceRevision(decision.Source, []byte(grounding)),
		ObservedAt:           discussDecisionObservedAt(decision),
		ApplicableDimensions: []colony.PlanningDimension{colony.PlanningDimensionKnowledge, colony.PlanningDimensionRisks},
		State:                planningEvidenceSourceCurrent,
	})
	if err != nil {
		return planningDecisionCandidate{}, nil, fmt.Errorf("build grounding for clarification %s: %w", decision.ID, err)
	}
	frontier.Items = append(frontier.Items, discussEvidenceItem{Record: groundingRecord, Content: grounding})
	evidence := []colony.PlanningEvidenceRef{groundingRecord.Reference}
	for _, item := range baseEvidence {
		if len(evidence) >= 4 {
			break
		}
		evidence = append(evidence, item.Record.Reference)
	}
	impact := planningDecisionImpactForDomain(domain, question)
	affectedID := discussAffectedSemanticID(domain, logicalDiscussSource(decision.Source), question)
	choices := discussPlanningDecisionChoices(domain, affectedID, options)
	recommendation := "Preserve the current contract until the owner explicitly chooses a different consequence."
	if len(options) > 0 {
		recommendation = fmt.Sprintf("Choose %q because it is the first viable outcome recorded at this evidence boundary.", options[0])
	}
	stableSource := logicalDiscussSource(decision.Source)
	if stableSource == "" {
		stableSource = decision.ID
	}
	stableHash := sha256.Sum256([]byte(stableSource))
	return planningDecisionCandidate{
		StableID:            "decision-" + string(domain) + "-" + hex.EncodeToString(stableHash[:6]),
		Domain:              domain,
		Decision:            question,
		WhyNow:              grounding,
		Evidence:            evidence,
		QueenRecommendation: recommendation,
		Impact:              impact,
		AffectedSemanticIDs: []string{affectedID},
		Choices:             choices,
		ResumeInstruction:   fmt.Sprintf("Resolve this card with aether discuss --resolve %s, then rerun aether discuss.", decision.ID),
	}, baseEvidence, nil
}

func planningDecisionDomainForClarification(decision PendingDecision) planningDecisionDomain {
	source := strings.TrimSpace(decision.Source)
	parts := strings.Split(source, ":")
	if len(parts) >= 3 && parts[0] == "wrapper" {
		if domain := planningDecisionDomain(parts[1]); domain.isMaterialDomain() {
			return domain
		}
	}
	if strings.HasPrefix(source, discussSourcePrefix) {
		category := strings.TrimSuffix(strings.TrimPrefix(source, discussSourcePrefix), ":hard")
		return planningDecisionDomainForCategory(category)
	}
	question := strings.ToLower(decision.Description)
	switch {
	case strings.Contains(question, "accept") || strings.Contains(question, "verify") || strings.Contains(question, "test"):
		return planningDecisionDomainAcceptanceMeaning
	case strings.Contains(question, "risk") || strings.Contains(question, "safe") || strings.Contains(question, "failure"):
		return planningDecisionDomainRiskTolerance
	case strings.Contains(question, "scope") || strings.Contains(question, "surface") || strings.Contains(question, "boundary"):
		return planningDecisionDomainScope
	case strings.Contains(question, "authority") || strings.Contains(question, "approve") || clarificationIsHardConstraint(decision):
		return planningDecisionDomainAuthority
	default:
		return planningDecisionDomainBehavior
	}
}

func planningDecisionImpactForDomain(domain planningDecisionDomain, question string) planningDecisionContractImpact {
	value := "owner answer changes the " + strings.ReplaceAll(string(domain), "_", " ") + " contract for: " + normalizePlanningDecisionText(question)
	impact := planningDecisionContractImpact{}
	switch domain {
	case planningDecisionDomainAuthority:
		impact.Authority = value
	case planningDecisionDomainRiskTolerance:
		impact.Risk = value
	case planningDecisionDomainScope:
		impact.Scope = value
	case planningDecisionDomainAcceptanceMeaning:
		impact.Acceptance = value
	default:
		impact.Behavior = value
	}
	return impact
}

func discussAffectedSemanticID(domain planningDecisionDomain, source, question string) string {
	seed := strings.TrimSpace(source)
	if seed == "" {
		seed = normalizePlanningDecisionText(question)
	}
	sum := sha256.Sum256([]byte(string(domain) + "\x00" + seed))
	return "intent:" + string(domain) + ":" + hex.EncodeToString(sum[:6])
}

func discussPlanningDecisionChoices(domain planningDecisionDomain, affectedID string, options []string) []planningDecisionChoice {
	if len(options) == 0 {
		options = []string{"Provide the bounded owner answer"}
	}
	choices := make([]planningDecisionChoice, 0, len(options))
	for index, option := range options {
		option = normalizePlanningDecisionText(option)
		if option == "" {
			continue
		}
		sum := sha256.Sum256([]byte(option))
		choices = append(choices, planningDecisionChoice{
			ID:                  fmt.Sprintf("choice-%02d-%s", index+1, hex.EncodeToString(sum[:3])),
			Label:               option,
			Consequence:         fmt.Sprintf("Choosing %s sets the %s contract for %s.", option, strings.ReplaceAll(string(domain), "_", " "), affectedID),
			Impact:              planningDecisionImpactForDomain(domain, option),
			AffectedSemanticIDs: []string{affectedID},
		})
	}
	return choices
}

func discussAnswerableEvidenceIDs(decision PendingDecision, evidence []discussEvidenceItem) []string {
	_, options := parseClarificationDescription(decision.Description)
	if len(options) == 0 {
		return nil
	}
	matchedIDs := []string{}
	matchedOptions := 0
	for _, option := range options {
		normalizedOption := normalizeDecisionText(option)
		if normalizedOption == "" {
			continue
		}
		matchedThisOption := false
		for _, item := range evidence {
			if strings.Contains(normalizeDecisionText(item.Content), normalizedOption) {
				matchedIDs = append(matchedIDs, item.Record.Reference.ID)
				matchedThisOption = true
			}
		}
		if matchedThisOption {
			matchedOptions++
		}
	}
	if matchedOptions != 1 {
		return nil
	}
	return nonEmptyPlanningDecisionIDs(matchedIDs)
}

func priorPlanningDecisionAnswer(file PendingDecisionFile, current PendingDecision, candidate planningDecisionCandidate, scope planningEvidenceScope) *planningDecisionAnswerRecord {
	var prior *PendingDecision
	currentSource := logicalDiscussSource(current.Source)
	for index := range file.Decisions {
		decision := file.Decisions[index]
		if !decision.Resolved || decision.Type != clarificationDecisionType || strings.TrimSpace(decision.Resolution) == "" || logicalDiscussSource(decision.Source) != currentSource {
			continue
		}
		copyDecision := decision
		if prior == nil || clarificationSortKey(copyDecision).After(clarificationSortKey(*prior)) {
			prior = &copyDecision
		}
	}
	if prior == nil {
		return nil
	}
	priorCandidate := candidate
	priorQuestion, priorOptions := parseClarificationDescription(prior.Description)
	priorCandidate.Domain = planningDecisionDomainForClarification(*prior)
	priorCandidate.Decision = priorQuestion
	priorCandidate.Impact = planningDecisionImpactForDomain(priorCandidate.Domain, priorQuestion)
	priorCandidate.AffectedSemanticIDs = []string{discussAffectedSemanticID(priorCandidate.Domain, currentSource, priorQuestion)}
	priorCandidate.Choices = discussPlanningDecisionChoices(priorCandidate.Domain, priorCandidate.AffectedSemanticIDs[0], priorOptions)
	key, err := buildPlanningDecisionEquivalenceKey(planningDecisionEquivalenceScope{
		GoalID:                          scope.GoalID,
		SessionID:                       scope.SessionID,
		ApprovedSpecificationRevisionID: scope.SpecificationRevisionID,
		BasePlanRevisionID:              scope.PlanRevisionID,
	}, priorCandidate)
	if err != nil {
		return nil
	}
	return &planningDecisionAnswerRecord{
		DecisionID: priorCandidate.StableID,
		ChoiceID:   discussChoiceIDForAnswer(prior.Resolution, priorCandidate.Choices),
		Answer:     prior.Resolution,
		Key:        key,
	}
}

func discussChoiceIDForAnswer(answer string, choices []planningDecisionChoice) string {
	normalized := normalizeDecisionText(answer)
	for _, choice := range choices {
		if normalizeDecisionText(choice.Label) == normalized {
			return choice.ID
		}
	}
	return "owner-answer"
}

func discussDecisionObservedAt(decision PendingDecision) time.Time {
	for _, candidate := range []string{decision.CreatedAt, decision.ResolvedAt} {
		if parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(candidate)); err == nil {
			return parsed
		}
	}
	return time.Date(1970, time.January, 1, 0, 0, 0, 0, time.UTC)
}

func discussionStatus(questionCount, createdCount, existingCount int) string {
	switch {
	case questionCount == 0:
		return "settled"
	case createdCount > 0:
		return "new_questions"
	case existingCount > 0:
		return "pending_questions"
	default:
		return "questions_ready"
	}
}

func renderDiscussVisual(result map[string]interface{}) string {
	var b strings.Builder
	b.WriteString(renderBanner("🧭", "Discuss"))
	b.WriteString(visualDividerStr())

	if resolved, _ := result["resolved"].(bool); resolved {
		b.WriteString(voiceLine("decision", "Clarification locked in.") + "\n")
		b.WriteString(voiceLine("decision", "Decision: "+stringValue(result["id"])) + "\n")
		b.WriteString(voiceLine("decision", "Answer: "+stringValue(result["answer"])) + "\n")
		if emitted, _ := result["redirect_emitted"].(bool); emitted {
			b.WriteString(signalTypeGlyph("REDIRECT") + " REDIRECT emitted: " + stringValue(result["redirect_text"]) + "\n")
		}
		if closeout, ok := discussSpecificationCloseoutFromResult(result); ok {
			b.WriteString(renderDiscussSpecificationCloseout(closeout))
		}
		b.WriteString(renderLifecycleClosing(result, "discuss"))
		return b.String()
	}

	b.WriteString(voiceLine("goal", "Goal: "+stringValue(result["goal"])) + "\n")
	b.WriteString(voiceLine("question", fmt.Sprintf("Questions: %d (%d new, %d existing)", intValue(result["question_count"]), intValue(result["created_count"]), intValue(result["existing_count"]))) + "\n")
	if intValue(result["resolved_count"]) > 0 {
		b.WriteString(voiceLine("question", fmt.Sprintf("Resolved clarifications already on file: %d", intValue(result["resolved_count"]))) + "\n")
	}
	if notice := stringValue(result["stale_state_notice"]); notice != "" {
		b.WriteString(voiceLine("warning", notice))
		b.WriteString("\n")
	}
	b.WriteString("\n")

	if questions, ok := result["questions"].([]discussQuestion); ok && len(questions) > 0 {
		b.WriteString(voiceLine("warning", fmt.Sprintf("Owner decisions required — %d material choice(s)", len(questions))) + "\n")
		b.WriteString(voiceLine("blocked", "Planning is paused until every card has an exact owner answer.") + "\n\n")
		for _, question := range questions {
			b.WriteString(voiceLine("question", fmt.Sprintf("Decision %s: %s", question.StableID, question.Question)) + "\n")
			b.WriteString(voiceLine("status", "Why now: "+question.WhyNow) + "\n")
			b.WriteString(voiceLine("evidence", "Evidence: "+renderDiscussEvidenceRefs(question.Evidence)) + "\n")
			b.WriteString(voiceLine("decision", "Queen recommends: "+question.QueenRecommendation+" (Queen is this project's coordinator)") + "\n")
			for _, choice := range question.Choices {
				b.WriteString(voiceLine("alternative", fmt.Sprintf("If %s: %s", choice.Label, choice.Consequence)) + "\n")
			}
			b.WriteString(voiceLine("artifact", "Affected scope: "+strings.Join(question.AffectedSemanticIDs, ", ")) + "\n")
			if strings.TrimSpace(question.PriorAnswer) == "" {
				b.WriteString(voiceLine("history", "Prior answer: none") + "\n")
			} else {
				b.WriteString(voiceLine("history", "Prior answer: "+question.PriorAnswer) + "\n")
			}
			b.WriteString(voiceLine("checkpoint", "Revalidation: "+question.Revalidation) + "\n")
			b.WriteString(voiceLine("next", "Planning resumes: "+question.PlanningResumes) + "\n")
			b.WriteString(voiceLine("next", "Answer exactly: "+question.ExactAnswerSyntax) + "\n")
			if question.HardConstraint {
				b.WriteString(voiceLine("avoid", "This answer becomes a hard constraint.") + "\n")
			}
			b.WriteString("\n")
		}
	} else {
		b.WriteString(voiceLine("done", "No unresolved material owner questions remain; evidence answered the rest.") + "\n\n")
		if closeout, ok := discussSpecificationCloseoutFromResult(result); ok {
			b.WriteString(renderDiscussSpecificationCloseout(closeout))
		}
	}

	b.WriteString(renderLifecycleClosing(result, "discuss"))
	return b.String()
}

func discussSpecificationCloseoutFromResult(result map[string]interface{}) (discussSpecificationCloseout, bool) {
	for _, key := range []string{"draft_spec", "approved_spec", "specification"} {
		switch value := result[key].(type) {
		case discussSpecificationCloseout:
			return value, true
		case *discussSpecificationCloseout:
			if value != nil {
				return *value, true
			}
		}
	}
	return discussSpecificationCloseout{}, false
}

func renderDiscussSpecificationCloseout(closeout discussSpecificationCloseout) string {
	var builder strings.Builder
	builder.WriteString("✓ Intent resolved\n")
	if closeout.Status == colony.SpecStatusDraft {
		builder.WriteString(renderStageMarker("Draft Specification"))
	} else {
		builder.WriteString(renderStageMarker("Existing Specification"))
	}
	fmt.Fprintf(&builder, "Draft SPEC: %s revision %d [%s]\n", closeout.SpecificationID, closeout.RevisionNumber, strings.ToUpper(string(closeout.Status)))
	fmt.Fprintf(&builder, "Revision ID: %s\n", closeout.RevisionID)
	fmt.Fprintf(&builder, "Content hash: %s\n", closeout.ContentHash)
	fmt.Fprintf(&builder, "Scope: %s\n", strings.ReplaceAll(string(closeout.Scope.Kind), "_", " "))
	if closeout.Scope.Kind == colony.SpecScopeFeature {
		fmt.Fprintf(&builder, "Feature: %s\n", closeout.Scope.FeatureID)
		fmt.Fprintf(&builder, "Requirement IDs: %s\n", specCommandIDSummary(closeout.Scope.RequirementIDs))
		fmt.Fprintf(&builder, "Acceptance IDs: %s\n", specCommandIDSummary(closeout.Scope.AcceptanceCheckIDs))
	}
	if closeout.Replayed && closeout.Status == colony.SpecStatusDraft {
		builder.WriteString("Draft already exists; the same revision was retained.\n")
	}
	if closeout.ProjectionRepaired {
		builder.WriteString("The readable projection was restored from canonical state.\n")
	}

	renderSpecCommandVisualSection(&builder, "goal", "Outcome", specCommandOutcomeVisualItems(closeout.Body.Outcomes))
	renderSpecCommandVisualSection(&builder, "done", "Included behavior", specCommandIncludedVisualItems(closeout.Body.IncludedBehaviors))
	renderSpecCommandVisualSection(&builder, "avoid", "Explicit exclusions", specCommandExclusionVisualItems(closeout.Body.Exclusions))
	renderSpecCommandVisualSection(&builder, "decision", "Binding decisions", specCommandDecisionVisualItems(closeout.Body.BindingDecisions))
	renderSpecCommandVisualSection(&builder, "requirement", "Requirements", specCommandRequirementVisualItems(closeout.Body.Requirements))
	renderSpecCommandVisualSection(&builder, "evidence", "Owner-checkable acceptance", specCommandAcceptanceVisualItems(closeout.Body.AcceptanceChecks))
	renderSpecCommandVisualSection(&builder, "avoid", "Negative expectations", specCommandNegativeVisualItems(closeout.Body.NegativeExpectations))
	renderSpecCommandVisualSection(&builder, "checkpoint", "Recovery expectations", specCommandRecoveryVisualItems(closeout.Body.RecoveryExpectations))
	renderSpecCommandVisualSection(&builder, "files", "Affected public paths", specCommandPublicPathVisualItems(closeout.Body.AffectedPublicPaths))

	if closeout.WouldCreate {
		builder.WriteString("Dry run only: this exact draft has not been committed.\n")
	} else if closeout.ApprovedSpecPreserved {
		builder.WriteString("The approved specification was left unchanged.\n")
		builder.WriteString(closeout.RevisionGuidance + "\n")
	} else {
		builder.WriteString("This draft does not authorize planning until the owner approves this exact revision.\n")
		fmt.Fprintf(&builder, "Review: `%s`\n", closeout.ExactNextCommand)
		if closeout.ApprovalCommand != "" {
			fmt.Fprintf(&builder, "Approve this exact revision after review: `%s`\n", closeout.ApprovalCommand)
		}
	}
	builder.WriteByte('\n')
	return builder.String()
}

func renderDiscussEvidenceRefs(evidence []colony.PlanningEvidenceRef) string {
	if len(evidence) == 0 {
		return "none"
	}
	values := make([]string, 0, len(evidence))
	for _, reference := range evidence {
		values = append(values, fmt.Sprintf("%s [%s] %s", reference.ID, reference.Kind, reference.Origin))
	}
	return strings.Join(values, "; ")
}

func loadPendingDecisionFile() PendingDecisionFile {
	var file PendingDecisionFile
	if store == nil {
		return PendingDecisionFile{Decisions: []PendingDecision{}}
	}
	if err := store.LoadJSON(pendingDecisionsFile, &file); err != nil {
		return PendingDecisionFile{Decisions: []PendingDecision{}}
	}
	if file.Decisions == nil {
		file.Decisions = []PendingDecision{}
	}
	return file
}

func loadCurrentPendingDecisionScope() pendingDecisionScope {
	if store == nil {
		return pendingDecisionScope{}
	}
	var state colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
		return pendingDecisionScope{}
	}
	return pendingDecisionScopeFromState(state)
}

func pendingDecisionScopeFromState(state colony.ColonyState) pendingDecisionScope {
	scope := pendingDecisionScope{GoalHash: pendingDecisionGoalHash(derefGoal(state.Goal))}
	if state.SessionID != nil {
		scope.SessionID = strings.TrimSpace(*state.SessionID)
	}
	if state.InitializedAt != nil {
		initializedAt := state.InitializedAt.UTC()
		scope.InitializedAt = &initializedAt
	}
	return scope
}

func pendingDecisionGoalHash(goal string) string {
	normalized := strings.Join(strings.Fields(strings.ToLower(strings.TrimSpace(goal))), " ")
	if normalized == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(normalized))
	return hex.EncodeToString(sum[:])
}

func stampPendingDecisionScope(decision *PendingDecision, scope pendingDecisionScope) {
	if decision == nil {
		return
	}
	if strings.TrimSpace(decision.GoalHash) == "" && strings.TrimSpace(scope.GoalHash) != "" {
		decision.GoalHash = scope.GoalHash
	}
	if strings.TrimSpace(decision.SessionID) == "" && strings.TrimSpace(scope.SessionID) != "" {
		decision.SessionID = scope.SessionID
	}
}

func filterPendingDecisionFileForScope(file PendingDecisionFile, scope pendingDecisionScope) (PendingDecisionFile, PendingDecisionFile) {
	active := PendingDecisionFile{Decisions: []PendingDecision{}}
	stale := PendingDecisionFile{Decisions: []PendingDecision{}}
	for _, decision := range file.Decisions {
		if pendingDecisionMatchesScope(decision, scope) {
			active.Decisions = append(active.Decisions, decision)
			continue
		}
		stale.Decisions = append(stale.Decisions, decision)
	}
	return active, stale
}

func pendingDecisionMatchesScope(decision PendingDecision, scope pendingDecisionScope) bool {
	// Prefer session ID over goal hash: two colonies with the same goal but
	// different sessions should not share pending decisions. Session ID is the
	// stronger scope boundary.
	scopeSession := strings.TrimSpace(scope.SessionID)
	decisionSession := strings.TrimSpace(decision.SessionID)
	scopeGoal := strings.TrimSpace(scope.GoalHash)
	decisionGoal := strings.TrimSpace(decision.GoalHash)
	if scopeSession != "" && decisionSession != "" {
		return scopeSession == decisionSession
	}
	if scopeSession != "" && decisionSession == "" && scope.InitializedAt != nil {
		createdAt, err := time.Parse(time.RFC3339, strings.TrimSpace(decision.CreatedAt))
		if err != nil || createdAt.Before(*scope.InitializedAt) {
			return false
		}
		if scopeGoal != "" && decisionGoal != "" {
			return scopeGoal == decisionGoal
		}
		return true
	}

	if scopeGoal != "" && decisionGoal != "" {
		return scopeGoal == decisionGoal
	}

	if scope.InitializedAt != nil {
		if createdAt, err := time.Parse(time.RFC3339, strings.TrimSpace(decision.CreatedAt)); err == nil {
			return !createdAt.Before(*scope.InitializedAt)
		}
		return false
	}
	return true
}

func pendingDecisionStaleNotice(count int) string {
	if count <= 0 {
		return ""
	}
	label := "clarification"
	if count != 1 {
		label = "clarifications"
	}
	return fmt.Sprintf("Ignored %d stale %s from a prior goal/session. They remain in pending-decisions.json for audit; run `aether discuss` to create or resolve current-goal clarifications.", count, label)
}

func clarificationDecisionIndex(file PendingDecisionFile) map[string]PendingDecision {
	index := map[string]PendingDecision{}
	for _, decision := range file.Decisions {
		if decision.Type != clarificationDecisionType {
			continue
		}
		if strings.TrimSpace(decision.Source) == "" {
			continue
		}
		if existing, ok := index[decision.Source]; ok {
			if clarificationSortKey(decision).After(clarificationSortKey(existing)) {
				index[decision.Source] = decision
			}
			continue
		}
		index[decision.Source] = decision
	}
	return index
}

func clarificationSortKey(decision PendingDecision) time.Time {
	for _, candidate := range []string{decision.ResolvedAt, decision.CreatedAt} {
		if ts, err := time.Parse(time.RFC3339, candidate); err == nil {
			return ts
		}
	}
	return time.Time{}
}

func resolvedClarifiedIntentEntries(file PendingDecisionFile) []clarifiedIntentEntry {
	entries := []clarifiedIntentEntry{}
	for _, decision := range file.Decisions {
		if decision.Type != clarificationDecisionType || !decision.Resolved || strings.TrimSpace(decision.Resolution) == "" {
			continue
		}
		question, _ := parseClarificationDescription(decision.Description)
		entries = append(entries, clarifiedIntentEntry{
			ID:         decision.ID,
			Question:   question,
			Resolution: decision.Resolution,
			Source:     decision.Source,
		})
	}
	sort.SliceStable(entries, func(i, j int) bool {
		return entries[i].ID < entries[j].ID
	})
	return entries
}

func countPendingClarifications(file PendingDecisionFile) int {
	total := 0
	for _, decision := range file.Decisions {
		if decision.Type == clarificationDecisionType && !decision.Resolved {
			total++
		}
	}
	return total
}

func countClarifications(file PendingDecisionFile) int {
	total := 0
	for _, decision := range file.Decisions {
		if decision.Type == clarificationDecisionType {
			total++
		}
	}
	return total
}

func activeSignalTexts() []string {
	pf := loadPheromones()
	if pf == nil || len(pf.Signals) == 0 {
		return nil
	}
	now := time.Now().UTC()
	active := filterSignalsForPrompt(pf.Signals, now)
	texts := make([]string, 0, len(active))
	for _, sig := range active {
		text := strings.ToLower(extractText(sig.Content))
		if strings.TrimSpace(text) != "" {
			texts = append(texts, text)
		}
	}
	return texts
}

func clarificationSuppressedBySignals(category string, activeSignals []string) bool {
	keywords := map[string][]string{
		"behavior":               {"behavior", "contract", "must", "should", "never"},
		"authority":              {"authority", "approve", "permission", "contract", "owner"},
		"risk_tolerance":         {"risk", "safe", "failure", "rollback", "recovery"},
		"acceptance_meaning":     {"accept", "test", "coverage", "verify", "validation"},
		"surface":                {"react", "vue", "svelte", "stack", "surface", "module", "backend", "frontend"},
		"integration":            {"api", "contract", "integration", "endpoint", "data", "adapter"},
		"scope":                  {"scope", "slice", "prototype", "polish", "cleanup", "breadth"},
		"verification":           {"test", "coverage", "qa", "verify", "validation"},
		"architecture":           {"architecture", "monolith", "module", "service", "boundary", "stack"},
		"dependencies":           {"dependency", "dependencies", "package", "library", "contract", "integration"},
		"testing_infrastructure": {"test", "coverage", "qa", "verify", "validation", "regression"},
		"deployment":             {"deploy", "deployment", "docker", "kubernetes", "release", "production"},
		"performance":            {"performance", "speed", "latency", "throughput", "scale"},
	}
	for _, signal := range activeSignals {
		for _, keyword := range keywords[category] {
			if strings.Contains(signal, keyword) {
				return true
			}
		}
	}
	return false
}

func formatClarificationDescription(question string, options []string) string {
	question = strings.TrimSpace(question)
	options = limitStrings(uniqueSortedStrings(options), 3)
	if len(options) == 0 {
		return question
	}
	return question + discussOptionDelimiter + strings.Join(options, " | ")
}

func parseClarificationDescription(description string) (string, []string) {
	parts := strings.SplitN(description, discussOptionDelimiter, 2)
	question := strings.TrimSpace(parts[0])
	if len(parts) == 1 {
		return question, nil
	}
	rawOptions := strings.Split(parts[1], "|")
	options := make([]string, 0, len(rawOptions))
	for _, option := range rawOptions {
		option = strings.TrimSpace(option)
		if option != "" {
			options = append(options, option)
		}
	}
	return question, options
}

func clarifiedIntentPromptEntries() []string {
	file, _ := loadScopedPendingDecisionFile(loadCurrentPendingDecisionScope())
	entries := resolvedClarifiedIntentEntries(file)
	return renderClarifiedIntentPromptEntries(entries)
}

func clarifiedIntentPromptRenderResult() clarifiedIntentRenderResult {
	return clarifiedIntentPromptRenderResultForScope(loadCurrentPendingDecisionScope())
}

func clarifiedIntentPromptRenderResultForScope(scope pendingDecisionScope) clarifiedIntentRenderResult {
	file, _ := loadScopedPendingDecisionFile(scope)
	entries := resolvedClarifiedIntentEntries(file)
	source := pendingDecisionsFile
	if store != nil {
		source = filepath.Join(store.BasePath(), pendingDecisionsFile)
	}
	return renderClarifiedIntentPromptEntriesWithIntegrity(entries, source)
}

func loadScopedPendingDecisionFile(scope pendingDecisionScope) (PendingDecisionFile, PendingDecisionFile) {
	file := loadPendingDecisionFile()
	return filterPendingDecisionFileForScope(file, scope)
}

func renderClarifiedIntentPromptEntries(entries []clarifiedIntentEntry) []string {
	rendered := renderBoundedClarifiedIntentPromptLines(entries)
	lines := make([]string, 0, len(rendered))
	for _, item := range rendered {
		lines = append(lines, item.line)
	}
	return lines
}

type clarifiedIntentPromptLine struct {
	entry clarifiedIntentEntry
	line  string
}

func renderBoundedClarifiedIntentPromptLines(entries []clarifiedIntentEntry) []clarifiedIntentPromptLine {
	rendered := make([]clarifiedIntentPromptLine, 0, clarifiedIntentMaxEntries)
	sectionChars := 0
	for _, entry := range entries {
		if len(rendered) >= clarifiedIntentMaxEntries {
			break
		}
		item, ok := boundedClarifiedIntentPromptLine(entry)
		if !ok {
			continue
		}
		lineChars := len(item.line) + 1
		if sectionChars+lineChars > clarifiedIntentMaxSectionChars {
			break
		}
		rendered = append(rendered, item)
		sectionChars += lineChars
	}
	return rendered
}

func boundedClarifiedIntentPromptLine(entry clarifiedIntentEntry) (clarifiedIntentPromptLine, bool) {
	question := truncateString(strings.TrimSpace(entry.Question), clarifiedIntentMaxQuestionChars)
	answer := truncateString(strings.TrimSpace(entry.Resolution), clarifiedIntentMaxAnswerChars)
	if question == "" || answer == "" {
		return clarifiedIntentPromptLine{}, false
	}
	line := truncateString(fmt.Sprintf("- %s => %s", question, answer), clarifiedIntentMaxEntryChars)
	return clarifiedIntentPromptLine{entry: entry, line: line}, true
}

func renderClarifiedIntentPromptEntriesWithIntegrity(entries []clarifiedIntentEntry, source string) clarifiedIntentRenderResult {
	result := clarifiedIntentRenderResult{
		Lines:    make([]string, 0, clarifiedIntentMaxEntries),
		Blocked:  []colonyPrimeLedgerItem{},
		Warnings: []string{},
	}
	source = strings.TrimSpace(source)
	if source == "" {
		source = pendingDecisionsFile
	}
	sectionChars := 0
	for idx, entry := range entries {
		if len(result.Lines) >= clarifiedIntentMaxEntries {
			break
		}
		item, ok := boundedClarifiedIntentPromptLine(entry)
		if !ok {
			continue
		}
		assessment := colony.AssessPromptSource(source, item.line)
		if assessment.Action == colony.PromptIntegrityActionBlock {
			entrySource := clarifiedIntentEntrySource(source, item.entry, idx)
			result.Warnings = append(result.Warnings, assessment.Warning("clarified_intent", entrySource))
			result.Blocked = append(result.Blocked, colonyPrimeLedgerItem{
				Name:           "clarified_intent",
				Title:          "Clarified Intent",
				Source:         filepath.ToSlash(entrySource),
				Priority:       8,
				Chars:          len(item.line),
				BaseTrustClass: assessment.BaseTrustClass,
				TrustClass:     assessment.TrustClass,
				Action:         assessment.Action,
				Blocked:        true,
				Findings:       append([]colony.PromptIntegrityFinding(nil), assessment.Findings...),
			})
			continue
		}
		lineChars := len(item.line) + 1
		if sectionChars+lineChars > clarifiedIntentMaxSectionChars {
			break
		}
		result.Lines = append(result.Lines, item.line)
		result.DecisionIDs = append(result.DecisionIDs, item.entry.ID)
		sectionChars += lineChars
	}
	return result
}

func clarifiedIntentEntrySource(source string, entry clarifiedIntentEntry, index int) string {
	id := strings.TrimSpace(entry.ID)
	if id == "" {
		id = fmt.Sprintf("entry_%d", index+1)
	}
	return source + "#" + id
}

func clarificationIsHardConstraint(decision PendingDecision) bool {
	// The typed field decides; the legacy ":hard" source suffix remains a
	// fallback so existing pending decisions keep their meaning.
	return decision.HardConstraint || strings.HasSuffix(strings.TrimSpace(decision.Source), ":hard")
}

func buildClarificationRedirect(decision PendingDecision, answer string) string {
	question, _ := parseClarificationDescription(decision.Description)
	question = strings.TrimSuffix(strings.TrimSpace(question), "?")
	if question == "" {
		return answer
	}
	return fmt.Sprintf("%s: %s", question, answer)
}

func discussSource(category string, hard bool) string {
	if hard {
		return discussSourcePrefix + category + ":hard"
	}
	return discussSourcePrefix + category
}

func goalTouchesUI(goal string) bool {
	for _, token := range []string{"dashboard", "ui", "page", "screen", "component", "frontend", "admin", "design"} {
		if strings.Contains(goal, token) {
			return true
		}
	}
	return false
}

func goalTouchesIntegration(goal string) bool {
	for _, token := range []string{"api", "data", "backend", "service", "auth", "integration", "sync", "dashboard"} {
		if strings.Contains(goal, token) {
			return true
		}
	}
	return false
}

func derefGoal(goal *string) string {
	if goal == nil {
		return ""
	}
	return *goal
}

// contradictionPair defines a pair of terms that contradict each other when
// both appear across resolved decisions. Positive and negative are lowercased
// keywords; category is a human-readable label for the conflict message.
type contradictionPair struct {
	positive string
	negative string
	category string
}

// contradictionPairs is a conservative list of genuinely contradictory term
// pairs. Keep the list small -- only add pairs where seeing both terms in
// resolved decisions clearly signals divergent intent.
var contradictionPairs = []contradictionPair{
	{"postgresql", "serverless", "database"},
	{"mysql", "serverless", "database"},
	{"sqlite", "serverless", "database"},
	{"monolith", "microservice", "architecture"},
	{"react", "vue", "frontend"},
	{"react", "svelte", "frontend"},
	{"rest", "graphql", "api"},
	{"docker", "no docker", "deployment"},
	{"kubernetes", "no kubernetes", "deployment"},
}

// detectDecisionConflicts examines resolved decisions for contradictory
// keyword pairs. For each contradiction pair where one resolved decision
// contains the positive keyword and another contains the negative keyword,
// it appends a formatted conflict string. Returns nil if no conflicts are
// found. Unresolved decisions and decisions with empty Resolution text are
// ignored.
func detectDecisionConflicts(decisions []PendingDecision) []string {
	if len(decisions) < 2 {
		return nil
	}

	var resolutions []string
	for _, d := range decisions {
		if !d.Resolved || strings.TrimSpace(d.Resolution) == "" {
			continue
		}
		resolutions = append(resolutions, strings.ToLower(d.Resolution))
	}

	if len(resolutions) < 2 {
		return nil
	}

	var conflicts []string
	for _, pair := range contradictionPairs {
		hasPositive := false
		hasNegative := false
		for _, r := range resolutions {
			if strings.Contains(r, pair.positive) {
				hasPositive = true
			}
			if strings.Contains(r, pair.negative) {
				hasNegative = true
			}
		}
		if hasPositive && hasNegative {
			conflicts = append(conflicts, fmt.Sprintf("Possible %s conflict: decisions reference both '%s' and '%s'", pair.category, pair.positive, pair.negative))
		}
	}

	if len(conflicts) == 0 {
		return nil
	}
	return conflicts
}
