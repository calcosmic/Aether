package colony

import (
	"math"
	"path/filepath"
	"sort"
	"strings"
)

type ContextScoreBreakdown struct {
	Trust        float64 `json:"trust"`
	Freshness    float64 `json:"freshness"`
	Confirmation float64 `json:"confirmation"`
	Relevance    float64 `json:"relevance"`
	Total        float64 `json:"total"`
}

type ContextCandidate struct {
	Name              string                `json:"name"`
	Title             string                `json:"title"`
	Source            string                `json:"source"`
	Content           string                `json:"content"`
	Cost              int                   `json:"cost,omitempty"`
	BudgetMetric      string                `json:"budget_metric,omitempty"`
	PriorityHint      int                   `json:"priority_hint,omitempty"`
	BaseTrustClass    PromptTrustClass      `json:"base_trust_class,omitempty"`
	TrustClass        PromptTrustClass      `json:"trust_class,omitempty"`
	Action            PromptIntegrityAction `json:"action,omitempty"`
	FreshnessScore    float64               `json:"freshness_score,omitempty"`
	ConfirmationScore float64               `json:"confirmation_score,omitempty"`
	RelevanceScore    float64               `json:"relevance_score,omitempty"`
	Protected         bool                  `json:"protected,omitempty"`
	PreserveReason    string                `json:"preserve_reason,omitempty"`
}

type RankedContextCandidate struct {
	ContextCandidate
	Score      ContextScoreBreakdown `json:"score_breakdown"`
	Preserved  bool                  `json:"preserved,omitempty"`
	TrimReason string                `json:"trim_reason,omitempty"`
	Decision   string                `json:"decision,omitempty"`
}

type ContextRankingResult struct {
	Included  []RankedContextCandidate `json:"included"`
	Trimmed   []RankedContextCandidate `json:"trimmed"`
	Preserved []RankedContextCandidate `json:"preserved"`
	Used      int                      `json:"used"`
	Budget    int                      `json:"budget"`
}

func RankContextCandidates(candidates []ContextCandidate, budget int) ContextRankingResult {
	if budget < 0 {
		budget = 0
	}

	ranked := make([]RankedContextCandidate, 0, len(candidates))
	for _, candidate := range candidates {
		ranked = append(ranked, RankedContextCandidate{
			ContextCandidate: normalizeContextCandidate(candidate),
			Score:            scoreContextCandidate(candidate),
		})
	}

	protected := make([]RankedContextCandidate, 0, len(ranked))
	regular := make([]RankedContextCandidate, 0, len(ranked))
	for _, item := range ranked {
		if item.Protected {
			protected = append(protected, item)
			continue
		}
		regular = append(regular, item)
	}

	sortRankedContextCandidates(protected)
	sortRankedContextCandidates(regular)

	result := ContextRankingResult{
		Included:  make([]RankedContextCandidate, 0, len(ranked)),
		Trimmed:   make([]RankedContextCandidate, 0, len(ranked)),
		Preserved: make([]RankedContextCandidate, 0, len(protected)),
		Budget:    budget,
	}

	currentLen := 0
	for _, item := range protected {
		item.Preserved = true
		item.Decision = "preserved"
		if strings.TrimSpace(item.PreserveReason) == "" {
			item.PreserveReason = "protected context must survive trimming"
		}
		result.Preserved = append(result.Preserved, item)
		result.Included = append(result.Included, item)
		currentLen += contextCandidateCost(item.ContextCandidate)
	}

	for _, item := range regular {
		itemCost := contextCandidateCost(item.ContextCandidate)
		if currentLen+itemCost > budget {
			remaining := budget - currentLen
			if trimmedCandidate, ok := trimContextCandidateToBudget(item.ContextCandidate, remaining); ok {
				item.ContextCandidate = trimmedCandidate
				item.Decision = "included"
				result.Included = append(result.Included, item)
				currentLen += contextCandidateCost(item.ContextCandidate)
				continue
			}
			item.Decision = "trimmed"
			item.TrimReason = "trimmed to preserve higher-weighted context within budget"
			result.Trimmed = append(result.Trimmed, item)
			continue
		}
		item.Decision = "included"
		result.Included = append(result.Included, item)
		currentLen += contextCandidateCost(item.ContextCandidate)
	}

	result.Used = currentLen
	return result
}

func scoreContextCandidate(candidate ContextCandidate) ContextScoreBreakdown {
	trust := clamp01(contextTrustClassScore(candidate.TrustClass))
	freshness := clamp01(candidate.FreshnessScore)
	confirmation := clamp01(candidate.ConfirmationScore)
	relevance := clamp01(candidate.RelevanceScore)

	total := 0.20*trust + 0.20*freshness + 0.20*confirmation + 0.40*relevance
	return ContextScoreBreakdown{
		Trust:        roundContextScore(trust),
		Freshness:    roundContextScore(freshness),
		Confirmation: roundContextScore(confirmation),
		Relevance:    roundContextScore(relevance),
		Total:        roundContextScore(total),
	}
}

func normalizeContextCandidate(candidate ContextCandidate) ContextCandidate {
	candidate.Name = strings.TrimSpace(candidate.Name)
	candidate.Title = strings.TrimSpace(candidate.Title)
	candidate.Source = filepath.ToSlash(strings.TrimSpace(candidate.Source))
	if candidate.Action == "" {
		candidate.Action = PromptIntegrityActionAllow
	}
	if candidate.BudgetMetric == "" {
		candidate.BudgetMetric = "chars"
	}
	if candidate.TrustClass == "" {
		candidate.TrustClass = candidate.BaseTrustClass
	}
	if candidate.Cost <= 0 {
		candidate.Cost = contentBudgetCost(candidate.Content, candidate.BudgetMetric)
	}
	return candidate
}

func contextTrustClassScore(class PromptTrustClass) float64 {
	switch class {
	case PromptTrustAuthorized:
		return 1.0
	case PromptTrustTrusted:
		return 0.9
	case PromptTrustUnknown:
		return 0.65
	case PromptTrustSuspicious:
		return 0.0
	default:
		return 0.5
	}
}

func sortRankedContextCandidates(items []RankedContextCandidate) {
	sort.SliceStable(items, func(i, j int) bool {
		left := items[i]
		right := items[j]

		if left.Score.Total != right.Score.Total {
			return left.Score.Total > right.Score.Total
		}
		if left.Score.Relevance != right.Score.Relevance {
			return left.Score.Relevance > right.Score.Relevance
		}
		if left.Score.Trust != right.Score.Trust {
			return left.Score.Trust > right.Score.Trust
		}
		if left.Score.Freshness != right.Score.Freshness {
			return left.Score.Freshness > right.Score.Freshness
		}
		if left.Score.Confirmation != right.Score.Confirmation {
			return left.Score.Confirmation > right.Score.Confirmation
		}
		if left.PriorityHint != right.PriorityHint {
			return left.PriorityHint > right.PriorityHint
		}
		if left.Name != right.Name {
			return left.Name < right.Name
		}
		return left.Source < right.Source
	})
}

func clamp01(value float64) float64 {
	if value < 0 {
		return 0
	}
	if value > 1 {
		return 1
	}
	return value
}

func roundContextScore(value float64) float64 {
	return math.Round(value*1e6) / 1e6
}

func contextCandidateCost(candidate ContextCandidate) int {
	if candidate.Cost > 0 {
		return candidate.Cost
	}
	return contentBudgetCost(candidate.Content, candidate.BudgetMetric)
}

func trimContextCandidateToBudget(candidate ContextCandidate, budget int) (ContextCandidate, bool) {
	candidate = normalizeContextCandidate(candidate)
	if budget <= 0 || contextCandidateCost(candidate) <= budget {
		return candidate, budget > 0
	}

	lines := strings.Split(candidate.Content, "\n")
	kept := make([]string, 0, len(lines))
	for _, line := range lines {
		full := strings.Join(append(kept, line), "\n")
		if contentBudgetCost(full, candidate.BudgetMetric) <= budget {
			kept = append(kept, line)
			continue
		}

		remaining := budget - contentBudgetCost(strings.Join(kept, "\n"), candidate.BudgetMetric)
		if remaining > 0 {
			truncated := truncateContentLineToBudget(line, remaining, candidate.BudgetMetric)
			if strings.TrimSpace(truncated) != "" {
				withTruncated := strings.Join(append(kept, truncated), "\n")
				if contentBudgetCost(withTruncated, candidate.BudgetMetric) <= budget {
					kept = append(kept, truncated)
				}
			}
		}
		break
	}

	trimmedContent := strings.TrimRight(strings.Join(kept, "\n"), "\n")
	if !trimmedCandidateHasPayload(candidate.Content, trimmedContent) {
		return candidate, false
	}

	candidate.Content = trimmedContent
	candidate.Cost = contentBudgetCost(trimmedContent, candidate.BudgetMetric)
	return normalizeContextCandidate(candidate), candidate.Cost > 0 && candidate.Cost <= budget
}

func trimmedCandidateHasPayload(original, trimmed string) bool {
	if strings.TrimSpace(trimmed) == "" {
		return false
	}
	if countNonEmptyLines(original) >= 2 && countNonEmptyLines(trimmed) < 2 {
		return false
	}
	return true
}

func countNonEmptyLines(content string) int {
	count := 0
	for _, line := range strings.Split(content, "\n") {
		if strings.TrimSpace(line) != "" {
			count++
		}
	}
	return count
}

func truncateContentLineToBudget(line string, budget int, metric string) string {
	if budget <= 0 {
		return ""
	}
	switch metric {
	case "words":
		words := strings.Fields(line)
		if len(words) == 0 {
			return ""
		}
		kept := make([]string, 0, len(words))
		for _, word := range words {
			candidate := word
			if len(kept) > 0 {
				candidate = strings.Join(append(append([]string{}, kept...), word), " ")
			}
			if contentBudgetCost(candidate, metric) > budget {
				break
			}
			kept = append(kept, word)
		}
		return strings.Join(kept, " ")
	default:
		if len(line) <= budget {
			return line
		}
		return strings.TrimSpace(line[:budget])
	}
}

func contentBudgetCost(content, metric string) int {
	switch metric {
	case "words":
		return len(strings.Fields(content))
	default:
		return len(content)
	}
}
