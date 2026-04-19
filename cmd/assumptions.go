package cmd

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/spf13/cobra"
)

const assumptionsFile = "assumptions.json"

var assumptionsAnalyzeCmd = &cobra.Command{
	Use:   "assumptions-analyze",
	Short: "Analyze plan assumptions, persist them, and emit steering signals",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		result, err := runAssumptionsAnalyze(skillWorkspaceRoot())
		if err != nil {
			outputError(1, err.Error(), nil)
			return nil
		}
		outputWorkflow(result, renderAssumptionsAnalyzeVisual(result))
		return nil
	},
}

var assumptionListCmd = &cobra.Command{
	Use:   "assumption-list",
	Short: "List persisted plan assumptions",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		phase, _ := cmd.Flags().GetInt("phase")
		unvalidatedOnly, _ := cmd.Flags().GetBool("unvalidated")
		result, err := runAssumptionList(phase, unvalidatedOnly)
		if err != nil {
			outputError(1, err.Error(), nil)
			return nil
		}
		outputWorkflow(result, renderAssumptionListVisual(result))
		return nil
	},
}

var assumptionValidateCmd = &cobra.Command{
	Use:   "assumption-validate",
	Short: "Mark one persisted assumption as validated",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		id := mustGetString(cmd, "id")
		if id == "" {
			return nil
		}
		note, _ := cmd.Flags().GetString("note")
		result, err := runAssumptionValidate(id, note)
		if err != nil {
			outputError(1, err.Error(), nil)
			return nil
		}
		outputWorkflow(result, renderAssumptionValidateVisual(result))
		return nil
	},
}

func init() {
	assumptionListCmd.Flags().Int("phase", 0, "Filter assumptions to a specific phase")
	assumptionListCmd.Flags().Bool("unvalidated", false, "Show only assumptions that still need validation")
	assumptionValidateCmd.Flags().String("id", "", "Assumption ID to validate")
	assumptionValidateCmd.Flags().String("note", "", "Optional validation note or evidence summary")

	rootCmd.AddCommand(assumptionsAnalyzeCmd)
	rootCmd.AddCommand(assumptionListCmd)
	rootCmd.AddCommand(assumptionValidateCmd)
}

func runAssumptionsAnalyze(root string) (map[string]interface{}, error) {
	state, err := loadActiveColonyState()
	if err != nil {
		return nil, fmt.Errorf("%s", colonyStateLoadMessage(err))
	}
	if state.Goal == nil || strings.TrimSpace(*state.Goal) == "" {
		return nil, fmt.Errorf("the colony goal is empty; run `aether init \"goal\"` again before analyzing assumptions")
	}
	if len(state.Plan.Phases) == 0 {
		return nil, fmt.Errorf("no project plan exists yet; run `aether plan` before analyzing assumptions")
	}

	survey, _ := loadCodexSurveyContext(root)
	existing := loadAssumptionsFile()
	assumptions := synthesizeAssumptions(strings.TrimSpace(*state.Goal), state.Plan.Phases, survey, existing)

	file := colony.AssumptionsFile{
		Version:     "1.0",
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Goal:        strings.TrimSpace(*state.Goal),
		Assumptions: assumptions,
	}
	if err := store.SaveJSON(assumptionsFile, file); err != nil {
		return nil, fmt.Errorf("failed to save assumptions: %w", err)
	}

	focusEmitted := 0
	feedbackEmitted := 0
	for _, assumption := range assumptions {
		if assumption.Validated {
			continue
		}
		switch assumption.Confidence {
		case colony.AssumptionConfidenceUnclear:
			if _, err := createPheromoneSignal("FOCUS", buildAssumptionFocus(assumption), "assumptions", "unclear assumption", "", 1.0, "normal"); err != nil {
				return nil, fmt.Errorf("failed to emit focus pheromone for assumption %s: %w", assumption.ID, err)
			}
			focusEmitted++
		case colony.AssumptionConfidenceConfident:
			if _, err := createPheromoneSignal("FEEDBACK", buildAssumptionFeedback(assumption), "assumptions", "high-confidence assumption", "", 1.0, "low"); err != nil {
				return nil, fmt.Errorf("failed to emit feedback pheromone for assumption %s: %w", assumption.ID, err)
			}
			feedbackEmitted++
		}
	}

	researchFlagged := 0
	for _, assumption := range assumptions {
		if assumption.ResearchFlag {
			researchFlagged++
		}
	}

	return map[string]interface{}{
		"analyzed":         true,
		"goal":             strings.TrimSpace(*state.Goal),
		"path":             filepath.Join(store.BasePath(), assumptionsFile),
		"assumption_count": len(assumptions),
		"research_flagged": researchFlagged,
		"focus_emitted":    focusEmitted,
		"feedback_emitted": feedbackEmitted,
		"assumptions":      assumptions,
		"next":             "Run `aether assumption-list` to review them, then `aether assumption-validate --id <id> --note \"...\"` as you confirm or override them.",
	}, nil
}

func runAssumptionList(phase int, unvalidatedOnly bool) (map[string]interface{}, error) {
	file := loadAssumptionsFile()
	filtered := make([]colony.Assumption, 0, len(file.Assumptions))
	for _, assumption := range file.Assumptions {
		if phase > 0 && assumption.Phase != phase {
			continue
		}
		if unvalidatedOnly && assumption.Validated {
			continue
		}
		filtered = append(filtered, assumption)
	}

	sort.SliceStable(filtered, func(i, j int) bool {
		if filtered[i].Phase != filtered[j].Phase {
			return filtered[i].Phase < filtered[j].Phase
		}
		return filtered[i].ID < filtered[j].ID
	})

	unvalidated := 0
	for _, assumption := range file.Assumptions {
		if !assumption.Validated {
			unvalidated++
		}
	}

	return map[string]interface{}{
		"count":        len(filtered),
		"total":        len(file.Assumptions),
		"unvalidated":  unvalidated,
		"goal":         file.Goal,
		"generated_at": file.GeneratedAt,
		"assumptions":  filtered,
		"next":         "Run `aether assumption-validate --id <id> --note \"...\"` to confirm one, or rerun `aether assumptions-analyze` after the plan changes.",
	}, nil
}

func runAssumptionValidate(id, note string) (map[string]interface{}, error) {
	file := loadAssumptionsFile()
	found := -1
	for i := range file.Assumptions {
		if file.Assumptions[i].ID == id {
			found = i
			break
		}
	}
	if found == -1 {
		return nil, fmt.Errorf("assumption %q not found", id)
	}

	file.Assumptions[found].Validated = true
	file.Assumptions[found].ValidatedAt = time.Now().UTC().Format(time.RFC3339)
	file.Assumptions[found].ValidationNote = strings.TrimSpace(note)

	if err := store.SaveJSON(assumptionsFile, file); err != nil {
		return nil, fmt.Errorf("failed to save assumptions: %w", err)
	}

	remaining := 0
	for _, assumption := range file.Assumptions {
		if !assumption.Validated {
			remaining++
		}
	}

	return map[string]interface{}{
		"validated": true,
		"id":        id,
		"note":      strings.TrimSpace(note),
		"remaining": remaining,
		"next":      "Run `aether assumption-list --unvalidated` to review what still needs confirmation.",
	}, nil
}

func loadAssumptionsFile() colony.AssumptionsFile {
	file := colony.AssumptionsFile{
		Version:     "1.0",
		Assumptions: []colony.Assumption{},
	}
	if store == nil {
		return file
	}
	if err := store.LoadJSON(assumptionsFile, &file); err != nil {
		return file
	}
	if file.Assumptions == nil {
		file.Assumptions = []colony.Assumption{}
	}
	return file
}

func synthesizeAssumptions(goal string, phases []colony.Phase, survey codexSurveyContext, existing colony.AssumptionsFile) []colony.Assumption {
	now := time.Now().UTC().Format(time.RFC3339)
	generated := []colony.Assumption{}

	firstPhase := 1
	lastPhase := 1
	if len(phases) > 0 {
		firstPhase = phases[0].ID
		lastPhase = phases[len(phases)-1].ID
	}

	if assumption := buildSurfaceAssumption(firstPhase, survey, now); assumption != nil {
		generated = append(generated, *assumption)
	}
	if assumption := buildIntegrationAssumption(firstPhase, goal, survey, now); assumption != nil {
		generated = append(generated, *assumption)
	}
	if assumption := buildVerificationAssumption(lastPhase, survey, now); assumption != nil {
		generated = append(generated, *assumption)
	}
	if assumption := buildScopeAssumption(firstPhase, goal, phases, now); assumption != nil {
		generated = append(generated, *assumption)
	}

	byID := map[string]colony.Assumption{}
	for _, assumption := range existing.Assumptions {
		byID[assumption.ID] = assumption
	}
	for i := range generated {
		if prior, ok := byID[generated[i].ID]; ok {
			generated[i].Validated = prior.Validated
			generated[i].ValidationNote = prior.ValidationNote
			generated[i].ValidatedAt = prior.ValidatedAt
		}
	}

	sort.SliceStable(generated, func(i, j int) bool {
		if generated[i].Phase != generated[j].Phase {
			return generated[i].Phase < generated[j].Phase
		}
		return generated[i].ID < generated[j].ID
	})
	return generated
}

func buildSurfaceAssumption(phase int, survey codexSurveyContext, createdAt string) *colony.Assumption {
	frameworks := limitStrings(uniqueSortedStrings(survey.Frameworks), 3)
	directories := limitStrings(uniqueSortedStrings(survey.Directories), 3)
	evidence := append([]string{}, frameworks...)
	evidence = append(evidence, directories...)

	confidence := colony.AssumptionConfidenceLikely
	text := "The first implementation slice should extend an existing product surface instead of introducing a parallel stack immediately."
	if len(frameworks) == 1 {
		confidence = colony.AssumptionConfidenceConfident
		text = fmt.Sprintf("The first implementation slice should stay in the existing %s surface.", frameworks[0])
	} else if len(frameworks) > 1 || len(directories) > 1 {
		confidence = colony.AssumptionConfidenceUnclear
	}

	return &colony.Assumption{
		ID:             fmt.Sprintf("asm_phase%d_surface", phase),
		Phase:          phase,
		Category:       "surface",
		AssumptionText: text,
		Evidence:       evidence,
		FilePath:       firstPathLike(evidence),
		Confidence:     confidence,
		IfWrong:        "The phase may start in the wrong layer and need to be re-planned once ownership is clarified.",
		ResearchFlag:   confidence == colony.AssumptionConfidenceUnclear,
		CreatedAt:      createdAt,
	}
}

func buildIntegrationAssumption(phase int, goal string, survey codexSurveyContext, createdAt string) *colony.Assumption {
	if !goalTouchesIntegration(strings.ToLower(goal)) && len(survey.EntryPoints) == 0 && len(survey.Dependencies) == 0 {
		return nil
	}

	entryPoints := limitStrings(uniqueSortedStrings(survey.EntryPoints), 3)
	deps := limitStrings(uniqueSortedStrings(survey.Dependencies), 3)
	evidence := append([]string{}, entryPoints...)
	evidence = append(evidence, deps...)

	confidence := colony.AssumptionConfidenceLikely
	if len(entryPoints) > 1 || len(deps) > 2 {
		confidence = colony.AssumptionConfidenceUnclear
	}
	if len(entryPoints) == 1 && len(deps) <= 1 {
		confidence = colony.AssumptionConfidenceConfident
	}

	return &colony.Assumption{
		ID:             fmt.Sprintf("asm_phase%d_integration", phase),
		Phase:          phase,
		Category:       "integration",
		AssumptionText: "The phase can reuse existing contracts and adapters before introducing a new integration boundary.",
		Evidence:       evidence,
		FilePath:       firstPathLike(evidence),
		Confidence:     confidence,
		IfWrong:        "Implementation could lock into the wrong API or data boundary and force a rewrite mid-phase.",
		ResearchFlag:   confidence == colony.AssumptionConfidenceUnclear,
		CreatedAt:      createdAt,
	}
}

func buildVerificationAssumption(phase int, survey codexSurveyContext, createdAt string) *colony.Assumption {
	testFiles := limitStrings(uniqueSortedStrings(survey.TestFiles), 3)
	confidence := colony.AssumptionConfidenceUnclear
	text := "A lightweight manual or prototype validation path is acceptable until the test harness is mapped."
	if len(testFiles) > 0 {
		confidence = colony.AssumptionConfidenceConfident
		text = fmt.Sprintf("Existing tests such as %s can anchor the first verification pass.", strings.Join(testFiles, ", "))
	}

	return &colony.Assumption{
		ID:             fmt.Sprintf("asm_phase%d_verification", phase),
		Phase:          phase,
		Category:       "verification",
		AssumptionText: text,
		Evidence:       testFiles,
		FilePath:       firstPathLike(testFiles),
		Confidence:     confidence,
		IfWrong:        "The phase may look finished without enough evidence that the critical path still works.",
		ResearchFlag:   confidence == colony.AssumptionConfidenceUnclear,
		CreatedAt:      createdAt,
	}
}

func buildScopeAssumption(phase int, goal string, phases []colony.Phase, createdAt string) *colony.Assumption {
	if len(phases) == 0 {
		return nil
	}

	goalLower := strings.ToLower(goal)
	text := "The first pass should optimize for the smallest end-to-end slice before expanding breadth or polish."
	confidence := colony.AssumptionConfidenceLikely
	if goalTouchesUI(goalLower) {
		text = "The first pass should bias toward a working slice before spending extra time on UI polish."
	}

	taskEvidence := []string{}
	for _, task := range phases[0].Tasks {
		if strings.TrimSpace(task.Goal) != "" {
			taskEvidence = append(taskEvidence, task.Goal)
		}
	}

	return &colony.Assumption{
		ID:             fmt.Sprintf("asm_phase%d_scope", phase),
		Phase:          phase,
		Category:       "scope",
		AssumptionText: text,
		Evidence:       limitStrings(taskEvidence, 3),
		Confidence:     confidence,
		IfWrong:        "The plan could over-invest in breadth or polish before the critical path is proven.",
		CreatedAt:      createdAt,
	}
}

func buildAssumptionFocus(assumption colony.Assumption) string {
	return fmt.Sprintf("Investigate phase %d %s assumption: %s", assumption.Phase, assumption.Category, assumption.AssumptionText)
}

func buildAssumptionFeedback(assumption colony.Assumption) string {
	return fmt.Sprintf("Proceed assuming phase %d %s: %s", assumption.Phase, assumption.Category, assumption.AssumptionText)
}

func firstPathLike(values []string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if strings.Contains(value, "/") || strings.Contains(value, ".") {
			return value
		}
	}
	return ""
}

func renderAssumptionsAnalyzeVisual(result map[string]interface{}) string {
	var b strings.Builder
	b.WriteString(renderBanner("🧠", "Assumptions"))
	b.WriteString(visualDivider)
	b.WriteString("Goal: ")
	b.WriteString(stringValue(result["goal"]))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("Assumptions: %d\n", intValue(result["assumption_count"])))
	b.WriteString(fmt.Sprintf("Signals: %d FOCUS, %d FEEDBACK\n", intValue(result["focus_emitted"]), intValue(result["feedback_emitted"])))
	if flagged := intValue(result["research_flagged"]); flagged > 0 {
		b.WriteString(fmt.Sprintf("Needs validation: %d unclear assumptions\n", flagged))
	}
	b.WriteString("\n")
	if assumptions, ok := result["assumptions"].([]colony.Assumption); ok {
		for _, assumption := range assumptions {
			b.WriteString(fmt.Sprintf("- [%s] Phase %d %s: %s\n", assumption.Confidence, assumption.Phase, assumption.Category, assumption.AssumptionText))
		}
		b.WriteString("\n")
	}
	b.WriteString(renderNextUp(stringValue(result["next"])))
	return b.String()
}

func renderAssumptionListVisual(result map[string]interface{}) string {
	var b strings.Builder
	b.WriteString(renderBanner("🧠", "Assumption List"))
	b.WriteString(visualDivider)
	b.WriteString("Goal: ")
	b.WriteString(emptyFallback(stringValue(result["goal"]), "unknown"))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("Showing %d of %d assumptions (%d still unvalidated)\n\n", intValue(result["count"]), intValue(result["total"]), intValue(result["unvalidated"])))
	if assumptions, ok := result["assumptions"].([]colony.Assumption); ok && len(assumptions) > 0 {
		for _, assumption := range assumptions {
			status := "pending"
			if assumption.Validated {
				status = "validated"
			}
			b.WriteString(fmt.Sprintf("- [%s] %s (%s)\n", assumption.ID, assumption.AssumptionText, status))
		}
	} else {
		b.WriteString("No assumptions recorded.\n")
	}
	b.WriteString("\n")
	b.WriteString(renderNextUp(stringValue(result["next"])))
	return b.String()
}

func renderAssumptionValidateVisual(result map[string]interface{}) string {
	var b strings.Builder
	b.WriteString(renderBanner("🧠", "Assumption Validate"))
	b.WriteString(visualDivider)
	b.WriteString("Validated: ")
	b.WriteString(stringValue(result["id"]))
	b.WriteString("\n")
	if note := strings.TrimSpace(stringValue(result["note"])); note != "" {
		b.WriteString("Note: ")
		b.WriteString(note)
		b.WriteString("\n")
	}
	b.WriteString(fmt.Sprintf("Remaining: %d\n", intValue(result["remaining"])))
	b.WriteString(renderNextUp(stringValue(result["next"])))
	return b.String()
}
