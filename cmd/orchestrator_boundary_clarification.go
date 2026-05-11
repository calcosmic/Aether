package cmd

import (
	"fmt"
	"strings"
	"time"
)

const (
	orchestratorBoundarySourcePrefix      = "orchestrator"
	orchestratorBoundaryFallbackWorkflow  = "unknown"
	orchestratorBoundaryFallbackCategory  = "general"
	orchestratorBoundaryHardSourceSuffix  = ":hard"
	orchestratorBoundarySourcePartDivider = '-'
)

func orchestratorBoundaryClarificationSource(workflow string, phase int, category string, hard bool) string {
	workflow = normalizeOrchestratorBoundarySourcePart(workflow, orchestratorBoundaryFallbackWorkflow)
	category = normalizeOrchestratorBoundarySourcePart(category, orchestratorBoundaryFallbackCategory)
	source := fmt.Sprintf("%s:%s:phase:%d:%s", orchestratorBoundarySourcePrefix, workflow, normalizeOrchestratorBoundaryPhase(phase), category)
	if hard {
		return source + orchestratorBoundaryHardSourceSuffix
	}
	return source
}

func materializeOrchestratorBoundaryClarifications(workflow string, phase int, pending PendingDecisionFile, candidates []discussQuestion, maxQuestions int, dryRun bool) ([]discussQuestion, int, int, error) {
	if pending.Decisions == nil {
		pending.Decisions = []PendingDecision{}
	}
	if maxQuestions <= 0 || maxQuestions > len(candidates) {
		maxQuestions = len(candidates)
	}
	scope := loadCurrentPendingDecisionScope()

	existingBySource := clarificationDecisionIndex(pending)
	surfacedBySource := map[string]bool{}
	questions := make([]discussQuestion, 0, maxQuestions)
	createdCount := 0
	existingCount := 0
	dirty := false
	normalizedPhase := normalizeOrchestratorBoundaryPhase(phase)

	for _, candidate := range candidates {
		if len(questions) >= maxQuestions {
			break
		}
		if strings.TrimSpace(candidate.Question) == "" {
			continue
		}
		category := normalizeOrchestratorBoundarySourcePart(candidate.Category, orchestratorBoundaryFallbackCategory)
		source := orchestratorBoundaryClarificationSource(workflow, phase, category, candidate.HardConstraint)
		if surfacedBySource[source] {
			continue
		}

		candidate.Category = category
		candidate.Source = source
		if existing, ok := existingBySource[source]; ok {
			if existing.Resolved {
				continue
			}
			candidate.ID = existing.ID
			candidate.Question, candidate.Options = parseClarificationDescription(existing.Description)
			candidate.HardConstraint = clarificationIsHardConstraint(existing)
			candidate.Status = "pending"
			candidate.Source = existing.Source
			questions = append(questions, candidate)
			surfacedBySource[source] = true
			existingCount++
			continue
		}

		candidate.Status = "new"
		if !dryRun {
			decision := PendingDecision{
				ID:          fmt.Sprintf("pd_%d", time.Now().UnixNano()+int64(createdCount)),
				Type:        clarificationDecisionType,
				Description: formatClarificationDescription(candidate.Question, candidate.Options),
				Source:      source,
				Resolved:    false,
				CreatedAt:   time.Now().UTC().Format(time.RFC3339),
			}
			if normalizedPhase > 0 {
				phaseValue := normalizedPhase
				decision.Phase = &phaseValue
			}
			stampPendingDecisionScope(&decision, scope)
			pending.Decisions = append(pending.Decisions, decision)
			candidate.ID = decision.ID
			dirty = true
		}
		questions = append(questions, candidate)
		surfacedBySource[source] = true
		createdCount++
	}

	if dirty {
		if store == nil {
			return nil, 0, 0, fmt.Errorf("no store initialized")
		}
		if err := store.SaveJSON(pendingDecisionsFile, pending); err != nil {
			return nil, 0, 0, fmt.Errorf("failed to save boundary clarification decisions: %w", err)
		}
	}
	return questions, createdCount, existingCount, nil
}

func normalizeOrchestratorBoundaryPhase(phase int) int {
	if phase < 0 {
		return 0
	}
	return phase
}

func normalizeOrchestratorBoundarySourcePart(value, fallback string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var b strings.Builder
	lastDivider := false
	for _, r := range value {
		valid := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
		if valid {
			b.WriteRune(r)
			lastDivider = false
			continue
		}
		if b.Len() > 0 && !lastDivider {
			b.WriteRune(orchestratorBoundarySourcePartDivider)
			lastDivider = true
		}
	}
	normalized := strings.Trim(string(b.String()), string(orchestratorBoundarySourcePartDivider))
	if normalized == "" {
		return fallback
	}
	return normalized
}
