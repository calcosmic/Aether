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
	Lines    []string
	Blocked  []colonyPrimeLedgerItem
	Warnings []string
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
	discussCmd.Flags().Int("max-questions", 3, "Maximum number of clarification questions to surface")
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
// wrapper-composed questions — the same four the canned generator uses plus
// "analysis". Closed on purpose: a typed category is what downstream
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
// cite the scan fact or state datum it derives from, or it is the
// same-three-canned-questions problem wearing a new coat. Dedup is by
// source slug, exactly as canned questions dedup, so re-running the
// composition never doubles questions.
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
	closeLifecycleCommand(result, "discuss", "", "")
	return result, nil
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
	next := "Run `aether discuss` to review remaining questions before planning."
	// override is what THIS answer unblocked -- the run whose manifest was
	// waiting on it. Only the caller knows that, so it is fed to the one
	// decision as an input rather than written over its answer afterwards.
	override := ""
	if remaining == 0 {
		next = nextAfterClarificationResolution(file.Decisions[found])
		override = orchestratorBoundaryAfterDiscussCommand(file.Decisions[found].Source)
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
	closeLifecycleCommand(result, "discuss", override,
		"Your answer was the last thing this run was waiting on, so this picks it straight back up with what you decided.")
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
		b.WriteString("Clarification locked in.\n")
		b.WriteString("Decision: ")
		b.WriteString(stringValue(result["id"]))
		b.WriteString("\n")
		b.WriteString("Answer: ")
		b.WriteString(stringValue(result["answer"]))
		b.WriteString("\n")
		if emitted, _ := result["redirect_emitted"].(bool); emitted {
			b.WriteString("REDIRECT emitted: ")
			b.WriteString(stringValue(result["redirect_text"]))
			b.WriteString("\n")
		}
		b.WriteString(renderLifecycleClosing(result, "discuss"))
		return b.String()
	}

	b.WriteString("Goal: ")
	b.WriteString(stringValue(result["goal"]))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("Questions: %d (%d new, %d existing)\n", intValue(result["question_count"]), intValue(result["created_count"]), intValue(result["existing_count"])))
	if intValue(result["resolved_count"]) > 0 {
		b.WriteString(fmt.Sprintf("Resolved clarifications already on file: %d\n", intValue(result["resolved_count"])))
	}
	if notice := stringValue(result["stale_state_notice"]); notice != "" {
		b.WriteString(notice)
		b.WriteString("\n")
	}
	b.WriteString("\n")

	if questions, ok := result["questions"].([]discussQuestion); ok && len(questions) > 0 {
		b.WriteString(fmt.Sprintf("⚠ Owner decisions required — %d material choice(s)\n", len(questions)))
		b.WriteString("Planning is paused until every card has an exact owner answer.\n\n")
		for _, question := range questions {
			b.WriteString(fmt.Sprintf("Decision %s: %s\n", question.StableID, question.Question))
			b.WriteString("Why now: " + question.WhyNow + "\n")
			b.WriteString("Evidence: " + renderDiscussEvidenceRefs(question.Evidence) + "\n")
			b.WriteString("Queen recommends: " + question.QueenRecommendation + "\n")
			for _, choice := range question.Choices {
				b.WriteString(fmt.Sprintf("If %s: %s\n", choice.Label, choice.Consequence))
			}
			b.WriteString("Affected scope: " + strings.Join(question.AffectedSemanticIDs, ", ") + "\n")
			if strings.TrimSpace(question.PriorAnswer) == "" {
				b.WriteString("Prior answer: none\n")
			} else {
				b.WriteString("Prior answer: " + question.PriorAnswer + "\n")
			}
			b.WriteString("Revalidation: " + question.Revalidation + "\n")
			b.WriteString("Planning resumes: " + question.PlanningResumes + "\n")
			b.WriteString("Answer exactly: " + question.ExactAnswerSyntax + "\n")
			if question.HardConstraint {
				b.WriteString("This answer becomes a hard constraint.\n")
			}
			b.WriteString("\n")
		}
	} else {
		b.WriteString("No unresolved material owner questions remain; evidence answered the rest.\n\n")
	}

	b.WriteString(renderLifecycleClosing(result, "discuss"))
	return b.String()
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

func nextAfterClarificationResolution(decision PendingDecision) string {
	if command := orchestratorBoundaryAfterDiscussCommand(decision.Source); command != "" {
		return fmt.Sprintf("Run `%s` to request a fresh manifest with the clarified boundary.", command)
	}
	return "Run `aether plan` to generate phases with the clarified intent."
}

func orchestratorBoundaryAfterDiscussCommand(source string) string {
	parts := strings.Split(strings.TrimSpace(source), ":")
	if len(parts) < 2 || parts[0] != orchestratorBoundarySourcePrefix {
		return ""
	}
	workflow := normalizeOrchestratorBoundarySourcePart(parts[1], "")
	switch workflow {
	case "plan":
		return "aether plan"
	case "build":
		if len(parts) >= 4 && parts[2] == "phase" {
			if phase := strings.TrimSpace(parts[3]); phase != "" && phase != "0" {
				return "aether build " + phase
			}
		}
		return "aether build"
	case "continue":
		return "aether continue"
	case "seal":
		return "aether seal"
	default:
		return ""
	}
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
