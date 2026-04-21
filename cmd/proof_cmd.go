package cmd

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/spf13/cobra"
)

type proofOutput struct {
	Goal      string          `json:"goal,omitempty"`
	State     string          `json:"state,omitempty"`
	Phase     int             `json:"phase,omitempty"`
	PhaseName string          `json:"phase_name,omitempty"`
	Context   proofContext    `json:"context"`
	Skills    proofSkillProof `json:"skills"`
	Summary   proofSummary    `json:"summary"`
	Next      string          `json:"next,omitempty"`
}

type proofContext struct {
	Surface       string                 `json:"surface"`
	BudgetMetric  string                 `json:"budget_metric,omitempty"`
	Budget        int                    `json:"budget,omitempty"`
	Used          int                    `json:"used,omitempty"`
	PromptSection string                 `json:"prompt_section,omitempty"`
	Warnings      []string               `json:"warnings,omitempty"`
	Included      []proofContextDecision `json:"included"`
	Trimmed       []proofContextDecision `json:"trimmed"`
	Preserved     []proofContextDecision `json:"preserved"`
	Blocked       []proofContextDecision `json:"blocked"`
}

type proofContextDecision struct {
	Name           string                          `json:"name"`
	Title          string                          `json:"title"`
	Source         string                          `json:"source"`
	BaseTrustClass colony.PromptTrustClass         `json:"base_trust_class,omitempty"`
	TrustClass     colony.PromptTrustClass         `json:"trust_class,omitempty"`
	Action         colony.PromptIntegrityAction    `json:"action,omitempty"`
	Blocked        bool                            `json:"blocked,omitempty"`
	Findings       []colony.PromptIntegrityFinding `json:"findings,omitempty"`
	Score          colony.ContextScoreBreakdown    `json:"score_breakdown,omitempty"`
	Preserved      bool                            `json:"preserved,omitempty"`
	PreserveReason string                          `json:"preserve_reason,omitempty"`
	TrimReason     string                          `json:"trim_reason,omitempty"`
	Decision       string                          `json:"decision,omitempty"`
}

type proofSkillProof struct {
	Source            string               `json:"source,omitempty"`
	Manifest          string               `json:"manifest,omitempty"`
	Phase             int                  `json:"phase,omitempty"`
	PhaseName         string               `json:"phase_name,omitempty"`
	DispatchCount     int                  `json:"dispatch_count"`
	MatchedSkillCount int                  `json:"matched_skill_count"`
	Dispatches        []proofSkillDispatch `json:"dispatches"`
}

type proofSkillDispatch struct {
	Name   string           `json:"name"`
	Caste  string           `json:"caste"`
	Task   string           `json:"task"`
	Status string           `json:"status,omitempty"`
	Wave   int              `json:"wave,omitempty"`
	TaskID string           `json:"task_id,omitempty"`
	Match  skillMatchResult `json:"match"`
}

type proofSummary struct {
	ContextSurface    string `json:"context_surface"`
	ContextIncluded   int    `json:"context_included"`
	ContextTrimmed    int    `json:"context_trimmed"`
	ContextPreserved  int    `json:"context_preserved"`
	ContextBlocked    int    `json:"context_blocked"`
	SkillSource       string `json:"skill_source,omitempty"`
	SkillDispatches   int    `json:"skill_dispatches"`
	SkillMatchedTotal int    `json:"skill_matched_total"`
}

var proofCmd = &cobra.Command{
	Use:   "proof",
	Short: "Inspect runtime context proof and skill proof for the active colony",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		state, err := loadActiveColonyState()
		if err != nil {
			outputError(1, colonyStateLoadMessage(err), nil)
			return nil
		}

		result := buildProofOutput(skillWorkspaceRoot(), state)
		outputWorkflow(result, renderProofVisual(state, result))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(proofCmd)
}

func buildProofOutput(root string, state colony.ColonyState) proofOutput {
	contextProof := buildProofContext()
	skillProof := buildProofSkillProof(root, state)
	primary, _ := workflowSuggestionsForState(state)

	goal := ""
	if state.Goal != nil {
		goal = strings.TrimSpace(*state.Goal)
	}

	phase := recoveryPhase(&state)
	phaseID := 0
	phaseName := ""
	if phase != nil {
		phaseID = phase.ID
		phaseName = phase.Name
	}

	return proofOutput{
		Goal:      goal,
		State:     string(state.State),
		Phase:     phaseID,
		PhaseName: phaseName,
		Context:   contextProof,
		Skills:    skillProof,
		Summary: proofSummary{
			ContextSurface:    contextProof.Surface,
			ContextIncluded:   len(contextProof.Included),
			ContextTrimmed:    len(contextProof.Trimmed),
			ContextPreserved:  len(contextProof.Preserved),
			ContextBlocked:    len(contextProof.Blocked),
			SkillSource:       skillProof.Source,
			SkillDispatches:   skillProof.DispatchCount,
			SkillMatchedTotal: skillProof.MatchedSkillCount,
		},
		Next: primary,
	}
}

func buildProofContext() proofContext {
	prime := buildColonyPrimeOutput(true)
	if strings.TrimSpace(prime.PromptSection) != "" || len(prime.Ledger.Included) > 0 || len(prime.Ledger.Blocked) > 0 {
		return proofContext{
			Surface:       "colony-prime",
			BudgetMetric:  "chars",
			Budget:        prime.Budget,
			Used:          prime.Used,
			PromptSection: prime.PromptSection,
			Warnings:      append([]string(nil), prime.Warnings...),
			Included:      convertColonyPrimeLedger(prime.Ledger.Included),
			Trimmed:       convertColonyPrimeLedger(prime.Ledger.Trimmed),
			Preserved:     convertColonyPrimeLedger(prime.Ledger.Preserved),
			Blocked:       convertColonyPrimeLedger(prime.Ledger.Blocked),
		}
	}

	capsule := buildContextCapsuleOutput(true, 8, 3, 2, 220)
	return proofContext{
		Surface:       "context-capsule",
		BudgetMetric:  "words",
		Budget:        220,
		Used:          capsule.WordCount,
		PromptSection: capsule.PromptSection,
		Warnings:      append([]string(nil), capsule.Warnings...),
		Included:      convertCapsuleDecisions(capsule.IncludedSections),
		Trimmed:       convertCapsuleDecisions(capsule.TrimmedSections),
		Preserved:     convertCapsuleDecisions(capsule.PreservedSections),
		Blocked:       convertCapsuleBlocked(capsule.Integrity),
	}
}

func buildProofSkillProof(root string, state colony.ColonyState) proofSkillProof {
	phase := recoveryPhase(&state)
	if phase == nil {
		return proofSkillProof{Dispatches: []proofSkillDispatch{}}
	}

	dispatches, source, manifestPath := proofDispatchesForState(state, phase)
	if len(dispatches) == 0 {
		return proofSkillProof{
			Source:     source,
			Manifest:   manifestPath,
			Phase:      phase.ID,
			PhaseName:  phase.Name,
			Dispatches: []proofSkillDispatch{},
		}
	}

	dispatchProofs := make([]proofSkillDispatch, 0, len(dispatches))
	totalMatched := 0
	hub := resolveHubPath()
	for _, dispatch := range dispatches {
		match := resolveSkillMatchesForRoot(hub, root, dispatch.Caste, dispatch.Task)
		totalMatched += match.Count
		dispatchProofs = append(dispatchProofs, proofSkillDispatch{
			Name:   dispatch.Name,
			Caste:  dispatch.Caste,
			Task:   dispatch.Task,
			Status: dispatch.Status,
			Wave:   dispatch.Wave,
			TaskID: dispatch.TaskID,
			Match:  match,
		})
	}

	return proofSkillProof{
		Source:            source,
		Manifest:          manifestPath,
		Phase:             phase.ID,
		PhaseName:         phase.Name,
		DispatchCount:     len(dispatchProofs),
		MatchedSkillCount: totalMatched,
		Dispatches:        dispatchProofs,
	}
}

func proofDispatchesForState(state colony.ColonyState, phase *colony.Phase) ([]codexBuildDispatch, string, string) {
	if phase == nil {
		return nil, "", ""
	}

	manifest := loadCodexContinueManifest(phase.ID)
	if manifest.Present && len(manifest.Data.Dispatches) > 0 {
		return manifest.Data.Dispatches, "build_manifest", displayDataPath(manifest.Path)
	}

	return plannedBuildDispatches(*phase, state.ColonyDepth), "phase_plan", ""
}

func convertColonyPrimeLedger(items []colonyPrimeLedgerItem) []proofContextDecision {
	result := make([]proofContextDecision, 0, len(items))
	for _, item := range items {
		result = append(result, proofContextDecision{
			Name:           item.Name,
			Title:          item.Title,
			Source:         item.Source,
			BaseTrustClass: item.BaseTrustClass,
			TrustClass:     item.TrustClass,
			Action:         item.Action,
			Blocked:        item.Blocked,
			Findings:       append([]colony.PromptIntegrityFinding(nil), item.Findings...),
			Score:          item.Score,
			Preserved:      item.Preserved,
			PreserveReason: item.PreserveReason,
			TrimReason:     item.TrimReason,
			Decision:       item.Decision,
		})
	}
	return result
}

func convertCapsuleDecisions(items []ContextCapsuleSectionDecision) []proofContextDecision {
	result := make([]proofContextDecision, 0, len(items))
	for _, item := range items {
		result = append(result, proofContextDecision{
			Name:           item.Name,
			Title:          item.Title,
			Source:         filepath.ToSlash(strings.TrimSpace(item.Source)),
			BaseTrustClass: item.BaseTrustClass,
			TrustClass:     item.TrustClass,
			Action:         item.Action,
			Score:          item.Score,
			Preserved:      item.Preserved,
			PreserveReason: item.PreserveReason,
			TrimReason:     item.TrimReason,
			Decision:       item.Decision,
		})
	}
	return result
}

func convertCapsuleBlocked(records []colony.PromptIntegrityRecord) []proofContextDecision {
	blocked := make([]proofContextDecision, 0)
	for _, record := range records {
		if record.Action != colony.PromptIntegrityActionBlock {
			continue
		}
		blocked = append(blocked, proofContextDecision{
			Name:           record.Name,
			Title:          record.Title,
			Source:         filepath.ToSlash(strings.TrimSpace(record.Source)),
			BaseTrustClass: record.BaseTrustClass,
			TrustClass:     record.TrustClass,
			Action:         record.Action,
			Blocked:        record.Blocked,
			Findings:       append([]colony.PromptIntegrityFinding(nil), record.Findings...),
		})
	}
	return blocked
}

func renderProofVisual(state colony.ColonyState, result proofOutput) string {
	var b strings.Builder
	b.WriteString(renderBanner("🔎", "Proof"))
	b.WriteString(visualDivider)
	if strings.TrimSpace(result.Goal) != "" {
		fmt.Fprintf(&b, "Goal: %s\n", result.Goal)
	}
	if result.Phase > 0 {
		fmt.Fprintf(&b, "Phase: %d", result.Phase)
		if strings.TrimSpace(result.PhaseName) != "" {
			fmt.Fprintf(&b, " (%s)", result.PhaseName)
		}
		b.WriteString("\n")
	}
	if strings.TrimSpace(result.State) != "" {
		fmt.Fprintf(&b, "State: %s\n", result.State)
	}

	b.WriteString("\n")
	b.WriteString(renderStageMarker("Context Proof"))
	fmt.Fprintf(&b, "Surface: %s\n", result.Context.Surface)
	if result.Context.Budget > 0 {
		fmt.Fprintf(&b, "Budget: %d %s | Used: %d\n", result.Context.Budget, result.Context.BudgetMetric, result.Context.Used)
	}
	fmt.Fprintf(&b, "Included: %d | Preserved: %d | Trimmed: %d | Blocked: %d\n",
		len(result.Context.Included), len(result.Context.Preserved), len(result.Context.Trimmed), len(result.Context.Blocked))
	renderProofDecisionSection(&b, "Blocked", result.Context.Blocked)
	renderProofDecisionSection(&b, "Trimmed", result.Context.Trimmed)
	renderProofDecisionSection(&b, "Preserved", result.Context.Preserved)
	renderProofDecisionSection(&b, "Included", result.Context.Included)

	b.WriteString("\n")
	b.WriteString(renderStageMarker("Skill Proof"))
	if result.Skills.DispatchCount == 0 {
		b.WriteString("No phase-aware skill proof is available yet.\n")
	} else {
		fmt.Fprintf(&b, "Source: %s\n", result.Skills.Source)
		if strings.TrimSpace(result.Skills.Manifest) != "" {
			fmt.Fprintf(&b, "Manifest: %s\n", result.Skills.Manifest)
		}
		fmt.Fprintf(&b, "Dispatches: %d | Matched skills: %d\n", result.Skills.DispatchCount, result.Skills.MatchedSkillCount)
		for _, dispatch := range result.Skills.Dispatches {
			b.WriteString("\n")
			fmt.Fprintf(&b, "%s %s\n", strings.TrimSpace(dispatch.Caste), strings.TrimSpace(dispatch.Name))
			fmt.Fprintf(&b, "Task: %s\n", strings.TrimSpace(dispatch.Task))
			if strings.TrimSpace(dispatch.Status) != "" {
				fmt.Fprintf(&b, "Status: %s\n", strings.TrimSpace(dispatch.Status))
			}
			if dispatch.Match.Count == 0 {
				b.WriteString("Matched: none\n")
				continue
			}
			fmt.Fprintf(&b, "Matched: %s\n", strings.Join(dispatch.Match.Matched, ", "))
			renderProofSkillEntries(&b, append(dispatch.Match.ColonySkills, dispatch.Match.DomainSkills...))
		}
	}

	primary, alternatives := workflowSuggestionsForState(state)
	alternatives = append(alternatives, "Run `aether status` to return to the colony dashboard.")
	b.WriteString(renderNextUp(primary, alternatives...))
	return b.String()
}

func renderProofDecisionSection(b *strings.Builder, title string, items []proofContextDecision) {
	if len(items) == 0 {
		return
	}
	fmt.Fprintf(b, "\n%s\n", title)
	for _, item := range items {
		fmt.Fprintf(b, "  - %s", proofDecisionLabel(item))
		if reason := proofDecisionReason(item); reason != "" {
			fmt.Fprintf(b, " — %s", reason)
		}
		if item.Score.Total > 0 {
			fmt.Fprintf(b, " (score %.3f)", item.Score.Total)
		}
		b.WriteString("\n")
	}
}

func renderProofSkillEntries(b *strings.Builder, entries []skillResolvedEntry) {
	for _, entry := range entries {
		fmt.Fprintf(b, "  - %s [%d]", entry.Name, entry.Score)
		reasonText := renderProofReasons(entry.Reasons)
		if reasonText != "" {
			fmt.Fprintf(b, " — %s", reasonText)
		}
		b.WriteString("\n")
	}
}

func renderProofReasons(reasons []skillMatchReason) string {
	if len(reasons) == 0 {
		return ""
	}
	parts := make([]string, 0, len(reasons))
	for _, reason := range reasons {
		label := strings.TrimSpace(reason.Code)
		evidence := strings.Join(reason.Evidence, ", ")
		if evidence != "" {
			parts = append(parts, fmt.Sprintf("%s=%s", label, evidence))
			continue
		}
		if label != "" {
			parts = append(parts, label)
		}
	}
	return strings.Join(parts, "; ")
}

func proofDecisionLabel(item proofContextDecision) string {
	label := strings.TrimSpace(item.Title)
	if label == "" {
		label = strings.TrimSpace(item.Name)
	}
	if label == "" {
		label = "unnamed section"
	}
	if strings.TrimSpace(item.Source) == "" {
		return label
	}
	return fmt.Sprintf("%s [%s]", label, item.Source)
}

func proofDecisionReason(item proofContextDecision) string {
	if item.Blocked && len(item.Findings) > 0 {
		first := item.Findings[0]
		if strings.TrimSpace(first.Evidence) != "" {
			return fmt.Sprintf("%s (%s)", first.Message, first.Evidence)
		}
		return first.Message
	}
	if strings.TrimSpace(item.TrimReason) != "" {
		return item.TrimReason
	}
	if strings.TrimSpace(item.PreserveReason) != "" {
		return item.PreserveReason
	}
	if strings.TrimSpace(item.Decision) != "" {
		return item.Decision
	}
	return ""
}
