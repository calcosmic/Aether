package cmd

import (
	"errors"
	"fmt"
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
)

type discussQuestion struct {
	ID             string   `json:"id,omitempty"`
	Category       string   `json:"category"`
	Question       string   `json:"question"`
	Options        []string `json:"options"`
	Reasoning      string   `json:"reasoning"`
	HardConstraint bool     `json:"hard_constraint,omitempty"`
	Status         string   `json:"status,omitempty"`
	Source         string   `json:"source,omitempty"`
}

type clarifiedIntentEntry struct {
	ID         string `json:"id"`
	Question   string `json:"question"`
	Resolution string `json:"resolution"`
	Source     string `json:"source,omitempty"`
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
	rootCmd.AddCommand(discussCmd)
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
	pending := loadPendingDecisionFile()
	activeSignals := activeSignalTexts()

	questions, createdCount, existingCount, err := materializeDiscussQuestions(goal, survey, pending, activeSignals, maxQuestions, dryRun)
	if err != nil {
		return nil, err
	}
	resolved := resolvedClarifiedIntentEntries(pending)

	next := "Run `aether plan` once the critical clarifications are resolved."
	if len(questions) > 0 {
		next = "Resolve a question with `aether discuss --resolve <id> --answer \"...\"`, then run `aether plan`."
	}

	return map[string]interface{}{
		"goal":              goal,
		"question_count":    len(questions),
		"created_count":     createdCount,
		"existing_count":    existingCount,
		"dry_run":           dryRun,
		"survey_docs":       survey.SurveyDocs,
		"questions":         questions,
		"resolved":          resolved,
		"resolved_count":    len(resolved),
		"pending_count":     countPendingClarifications(pending),
		"signal_count":      len(activeSignals),
		"survey_available":  len(survey.SurveyDocs) > 0 || len(survey.Frameworks) > 0 || len(survey.Directories) > 0,
		"next":              next,
		"discussion_status": discussionStatus(len(questions), createdCount, existingCount),
	}, nil
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
	found := -1
	for i := range file.Decisions {
		if file.Decisions[i].ID == id {
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

	remaining := countPendingClarifications(file)
	next := "Run `aether discuss` to review remaining questions before planning."
	if remaining == 0 {
		next = "Run `aether plan` to generate phases with the clarified intent."
	}

	return map[string]interface{}{
		"resolved":         true,
		"id":               id,
		"answer":           answer,
		"redirect_emitted": redirectEmitted,
		"redirect_text":    redirectText,
		"remaining":        remaining,
		"next":             next,
	}, nil
}

func materializeDiscussQuestions(goal string, survey codexSurveyContext, pending PendingDecisionFile, activeSignals []string, maxQuestions int, dryRun bool) ([]discussQuestion, int, int, error) {
	existingBySource := clarificationDecisionIndex(pending)
	candidates := generateDiscussCandidates(goal, survey)
	questions := make([]discussQuestion, 0, maxQuestions)
	createdCount := 0
	existingCount := 0
	dirty := false

	for _, candidate := range candidates {
		if len(questions) >= maxQuestions {
			break
		}
		if clarificationSuppressedBySignals(candidate.Category, activeSignals) {
			continue
		}
		if existing, ok := existingBySource[candidate.Source]; ok {
			if existing.Resolved {
				continue
			}
			candidate.ID = existing.ID
			candidate.Question, candidate.Options = parseClarificationDescription(existing.Description)
			candidate.Status = "pending"
			questions = append(questions, candidate)
			existingCount++
			continue
		}

		candidate.Status = "new"
		if !dryRun {
			decision := PendingDecision{
				ID:          fmt.Sprintf("pd_%d", time.Now().UnixNano()+int64(createdCount)),
				Type:        clarificationDecisionType,
				Description: formatClarificationDescription(candidate.Question, candidate.Options),
				Source:      candidate.Source,
				Resolved:    false,
				CreatedAt:   time.Now().UTC().Format(time.RFC3339),
			}
			pending.Decisions = append(pending.Decisions, decision)
			candidate.ID = decision.ID
			dirty = true
		}
		questions = append(questions, candidate)
		createdCount++
	}

	if dirty {
		if err := store.SaveJSON(pendingDecisionsFile, pending); err != nil {
			return nil, 0, 0, fmt.Errorf("failed to save clarification decisions: %w", err)
		}
	}
	return questions, createdCount, existingCount, nil
}

func generateDiscussCandidates(goal string, survey codexSurveyContext) []discussQuestion {
	goalLower := strings.ToLower(goal)
	candidates := []discussQuestion{
		buildDiscussSurfaceQuestion(survey),
		buildDiscussIntegrationQuestion(goalLower, survey),
		buildDiscussScopeQuestion(goalLower),
		buildDiscussVerificationQuestion(survey),
	}

	filtered := make([]discussQuestion, 0, len(candidates))
	for _, candidate := range candidates {
		if strings.TrimSpace(candidate.Question) == "" || strings.TrimSpace(candidate.Source) == "" {
			continue
		}
		filtered = append(filtered, candidate)
	}
	return filtered
}

func buildDiscussSurfaceQuestion(survey codexSurveyContext) discussQuestion {
	options := uniqueSortedStrings(append(append([]string{}, limitStrings(survey.Frameworks, 3)...), limitStrings(survey.Directories, 3)...))
	options = limitStrings(options, 3)
	reasoning := "The goal can be built in more than one place unless you pin down which existing surface should own the first slice."
	if len(options) == 0 {
		options = []string{
			"keep it in the current primary stack",
			"create a new isolated module",
			"research the best surface before deciding",
		}
		reasoning = "The survey did not expose a single obvious surface, so planning needs an explicit ownership choice before it guesses."
	} else if len(options) == 1 {
		options = append(options,
			"create a new isolated module",
			"research the best surface before deciding",
		)
		reasoning = fmt.Sprintf("The survey only highlighted %s as an obvious surface, but it is still worth confirming whether you want to stay there or carve out a separate module.", options[0])
	} else {
		options = append(options, "follow the dominant existing pattern")
		reasoning = fmt.Sprintf("The survey surfaced multiple plausible implementation surfaces (%s), so the plan should not guess which one owns the work.", strings.Join(limitStrings(options, 3), ", "))
	}
	return discussQuestion{
		Category:       "surface",
		Question:       "Which existing surface should own the first implementation slice?",
		Options:        limitStrings(options, 3),
		Reasoning:      reasoning,
		HardConstraint: true,
		Source:         discussSource("surface", true),
	}
}

func buildDiscussIntegrationQuestion(goalLower string, survey codexSurveyContext) discussQuestion {
	if !goalTouchesIntegration(goalLower) && len(survey.EntryPoints) == 0 && len(survey.Dependencies) == 0 {
		return discussQuestion{}
	}

	options := []string{
		"reuse existing contracts where possible",
		"add a thin adapter around current contracts",
		"allow a new contract if the current one blocks the goal",
	}
	reasoning := "Planning needs to know how aggressively it should reuse current APIs, data flows, or integration boundaries."
	if len(survey.EntryPoints) > 0 || len(survey.Dependencies) > 0 {
		reasoning = fmt.Sprintf("The survey found live entry points (%s) and dependencies (%s), so the plan should know whether to reuse them or introduce a new boundary.", renderCSV(limitStrings(survey.EntryPoints, 3), "none detected"), renderCSV(limitStrings(survey.Dependencies, 3), "none detected"))
	}

	return discussQuestion{
		Category:       "integration",
		Question:       "How tightly should this work reuse existing contracts and integrations?",
		Options:        options,
		Reasoning:      reasoning,
		HardConstraint: true,
		Source:         discussSource("integration", true),
	}
}

func buildDiscussScopeQuestion(goalLower string) discussQuestion {
	options := []string{
		"smallest end-to-end slice first",
		"broader feature coverage first",
		"architecture groundwork first",
	}
	if goalTouchesUI(goalLower) {
		options = []string{
			"smallest working slice first",
			"balanced function and polish",
			"polish-heavy first pass",
		}
	}
	return discussQuestion{
		Category:  "scope",
		Question:  "What should planning optimize for on the first pass?",
		Options:   options,
		Reasoning: "This keeps the plan from guessing the wrong tradeoff between speed, breadth, and cleanup.",
		Source:    discussSource("scope", false),
	}
}

func buildDiscussVerificationQuestion(survey codexSurveyContext) discussQuestion {
	options := []string{
		"focused regression tests only",
		"happy path and failure path coverage",
		"prototype first, tighten tests after feedback",
	}
	reasoning := "Workers routinely guess the verification bar; making it explicit prevents overbuilding or under-testing."
	if len(survey.TestFiles) == 0 {
		options = []string{
			"add one meaningful validation path now",
			"prototype first without tests",
			"research the test harness before committing",
		}
		reasoning = "The survey did not find obvious tests, so the plan needs an explicit answer about how much verification to front-load."
	} else {
		reasoning = fmt.Sprintf("The repo already contains tests such as %s, so the colony should know whether to keep that bar, raise it, or intentionally relax it for the first slice.", renderCSV(limitStrings(survey.TestFiles, 3), "existing tests"))
	}
	return discussQuestion{
		Category:  "verification",
		Question:  "What verification bar do you want on the first pass?",
		Options:   options,
		Reasoning: reasoning,
		Source:    discussSource("verification", false),
	}
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
	b.WriteString(visualDivider)

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
		b.WriteString(renderNextUp(stringValue(result["next"])))
		return b.String()
	}

	b.WriteString("Goal: ")
	b.WriteString(stringValue(result["goal"]))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("Questions: %d (%d new, %d existing)\n", intValue(result["question_count"]), intValue(result["created_count"]), intValue(result["existing_count"])))
	if intValue(result["resolved_count"]) > 0 {
		b.WriteString(fmt.Sprintf("Resolved clarifications already on file: %d\n", intValue(result["resolved_count"])))
	}
	b.WriteString("\n")

	if questions, ok := result["questions"].([]discussQuestion); ok && len(questions) > 0 {
		for idx, question := range questions {
			b.WriteString(fmt.Sprintf("%d. [%s] %s\n", idx+1, emptyFallback(question.ID, "pending"), question.Question))
			if len(question.Options) > 0 {
				b.WriteString("   Options: ")
				b.WriteString(strings.Join(question.Options, " | "))
				b.WriteString("\n")
			}
			if strings.TrimSpace(question.Reasoning) != "" {
				b.WriteString("   Why: ")
				b.WriteString(question.Reasoning)
				b.WriteString("\n")
			}
			if question.HardConstraint {
				b.WriteString("   This answer becomes a hard constraint.\n")
			}
		}
		b.WriteString("\n")
	} else {
		b.WriteString("No new clarification questions are outstanding.\n\n")
	}

	b.WriteString(renderNextUp(stringValue(result["next"])))
	return b.String()
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
		"surface":      {"react", "vue", "svelte", "stack", "surface", "module", "backend", "frontend"},
		"integration":  {"api", "contract", "integration", "endpoint", "data", "adapter"},
		"scope":        {"scope", "slice", "prototype", "polish", "cleanup", "breadth"},
		"verification": {"test", "coverage", "qa", "verify", "validation"},
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
	file := loadPendingDecisionFile()
	entries := resolvedClarifiedIntentEntries(file)
	lines := make([]string, 0, len(entries))
	for _, entry := range entries {
		question := strings.TrimSpace(entry.Question)
		answer := strings.TrimSpace(entry.Resolution)
		if question == "" || answer == "" {
			continue
		}
		lines = append(lines, fmt.Sprintf("- %s => %s", question, answer))
	}
	return lines
}

func clarificationIsHardConstraint(decision PendingDecision) bool {
	return strings.HasSuffix(strings.TrimSpace(decision.Source), ":hard")
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
