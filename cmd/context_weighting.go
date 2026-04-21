package cmd

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

func readHiveWisdomEntries(hubDir string, limit int, fallbacks *[]string) []hiveWisdomEntry {
	wisdomPath := filepath.Join(hubDir, "hive", "wisdom.json")
	data, err := os.ReadFile(wisdomPath)
	if err == nil {
		var wf hiveWisdomData
		if json.Unmarshal(data, &wf) == nil && len(wf.Entries) > 0 {
			if limit > 0 && len(wf.Entries) > limit {
				return append([]hiveWisdomEntry(nil), wf.Entries[:limit]...)
			}
			return append([]hiveWisdomEntry(nil), wf.Entries...)
		}
	}

	eternalPath := filepath.Join(hubDir, "eternal", "memory.json")
	eternalData, eternalErr := os.ReadFile(eternalPath)
	if eternalErr == nil {
		var entries []struct {
			Text string `json:"text"`
		}
		if json.Unmarshal(eternalData, &entries) == nil && len(entries) > 0 {
			count := len(entries)
			if limit > 0 && count > limit {
				count = limit
			}
			results := make([]hiveWisdomEntry, 0, count)
			for i := 0; i < count; i++ {
				results = append(results, hiveWisdomEntry{
					ID:         "eternal_fallback",
					Text:       entries[i].Text,
					SourceRepo: "eternal",
					Confidence: 0.70,
				})
			}
			return results
		}
	}

	*fallbacks = append(*fallbacks, "hive_wisdom: no hive or eternal data")
	return nil
}

func freshnessScoreFromTimestamp(ts string, now time.Time, fallback float64) float64 {
	ts = strings.TrimSpace(ts)
	if ts == "" {
		return clampScoreUnit(fallback)
	}
	parsed, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		return clampScoreUnit(fallback)
	}
	if parsed.After(now) {
		return 1
	}
	days := now.Sub(parsed).Hours() / 24
	if days < 0 {
		days = 0
	}
	return clampScoreUnit(math.Pow(0.5, days/60.0))
}

func latestFreshnessScore(now time.Time, fallback float64, timestamps ...string) float64 {
	best := 0.0
	found := false
	for _, ts := range timestamps {
		if strings.TrimSpace(ts) == "" {
			continue
		}
		score := freshnessScoreFromTimestamp(ts, now, fallback)
		if !found || score > best {
			best = score
			found = true
		}
	}
	if found {
		return best
	}
	return clampScoreUnit(fallback)
}

func clampScoreUnit(value float64) float64 {
	if value < 0 {
		return 0
	}
	if value > 1 {
		return 1
	}
	return value
}

func sectionRelevanceScore(name string) float64 {
	switch name {
	case "blockers", "risks":
		return 1.0
	case "pheromones", "signals":
		return 0.95
	case "clarified_intent":
		return 0.90
	case "state":
		return 0.85
	case "user_preferences":
		return 0.80
	case "instincts":
		return 0.80
	case "decisions":
		return 0.55
	case "hive_wisdom":
		return 0.25
	case "learnings":
		return 0.25
	case "recent_narrative":
		return 0.10
	default:
		return 0.25
	}
}

func protectedSectionPolicy(name string) (bool, string) {
	switch name {
	case "state":
		return true, "authoritative runtime state"
	case "pheromones", "signals":
		return true, "active user steering and constraints"
	case "blockers", "risks":
		return true, "active blockers must survive trimming"
	case "user_preferences":
		return true, "explicit user preferences"
	case "clarified_intent":
		return true, "clarified user intent"
	default:
		return false, ""
	}
}

func confidenceScoreFromLearnings(learnings []colony.Learning) float64 {
	if len(learnings) == 0 {
		return 0.4
	}
	total := 0.0
	for _, learning := range learnings {
		score := 0.20
		switch strings.ToLower(strings.TrimSpace(learning.Status)) {
		case "validated", "confirmed":
			score += 0.30
		case "active", "candidate":
			score += 0.15
		}
		if learning.Tested {
			score += 0.20
		}
		if strings.TrimSpace(learning.Evidence) != "" {
			score += 0.10
		}
		total += clampScoreUnit(score)
	}
	return clampScoreUnit(total / float64(len(learnings)))
}

func confidenceScoreFromDecisions(decisions []colony.Decision, currentPhase int) float64 {
	if len(decisions) == 0 {
		return 0.4
	}
	score := 0.55
	for _, decision := range decisions {
		if decision.Phase == currentPhase {
			score += 0.15
			break
		}
	}
	return clampScoreUnit(score)
}

func confidenceScoreFromInstinctEntries(entries []colony.InstinctEntry) float64 {
	if len(entries) == 0 {
		return 0.4
	}
	best := 0.0
	for _, entry := range entries {
		score := entry.TrustScore
		if entry.Confidence > score {
			score = entry.Confidence
		}
		if entry.Provenance.ApplicationCount >= 3 {
			score += 0.10
		}
		if score > best {
			best = score
		}
	}
	return clampScoreUnit(best)
}

func confidenceScoreFromLegacyInstincts(entries []struct {
	trigger    string
	action     string
	confidence float64
}) float64 {
	if len(entries) == 0 {
		return 0.4
	}
	best := 0.0
	for _, entry := range entries {
		if entry.confidence > best {
			best = entry.confidence
		}
	}
	return clampScoreUnit(best)
}

func confidenceScoreFromSignals(signals []colony.PheromoneSignal, now time.Time) float64 {
	if len(signals) == 0 {
		return 0.4
	}
	best := 0.0
	for _, sig := range signals {
		score := computeEffectiveStrength(sig, now)
		if score > best {
			best = score
		}
	}
	return clampScoreUnit(best)
}

func confidenceScoreFromHive(entries []hiveWisdomEntry) float64 {
	if len(entries) == 0 {
		return 0.4
	}
	best := 0.0
	for _, entry := range entries {
		if entry.Confidence > best {
			best = entry.Confidence
		}
	}
	return clampScoreUnit(best)
}

func buildHiveWisdomLines(entries []hiveWisdomEntry) []string {
	if len(entries) == 0 {
		return nil
	}
	lines := make([]string, 0, len(entries))
	for _, entry := range entries {
		lines = append(lines, truncateString(entry.Text, 200))
	}
	return lines
}

func phaseScopedRelevance(base float64, currentPhase int, phases ...int) float64 {
	if currentPhase <= 0 {
		return clampScoreUnit(base)
	}
	for _, phase := range phases {
		if phase == currentPhase {
			return clampScoreUnit(base + 0.10)
		}
	}
	return clampScoreUnit(base)
}

func latestSummaryFreshness(path string, now time.Time, fallback float64) float64 {
	data, err := os.ReadFile(path)
	if err != nil {
		return clampScoreUnit(fallback)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		fields := strings.SplitN(strings.TrimSpace(lines[i]), "|", 2)
		if len(fields) == 0 {
			continue
		}
		if ts := strings.TrimSpace(fields[0]); ts != "" {
			if _, err := time.Parse(time.RFC3339, ts); err == nil {
				return freshnessScoreFromTimestamp(ts, now, fallback)
			}
		}
	}
	return clampScoreUnit(fallback)
}

func latestDecisionFreshness(now time.Time, decisions []colony.Decision) float64 {
	timestamps := make([]string, 0, len(decisions))
	for _, decision := range decisions {
		timestamps = append(timestamps, decision.Timestamp)
	}
	return latestFreshnessScore(now, 0.75, timestamps...)
}

func decisionPhases(decisions []colony.Decision) []int {
	phases := make([]int, 0, len(decisions))
	for _, decision := range decisions {
		phases = append(phases, decision.Phase)
	}
	return phases
}

func latestPhaseLearningFreshness(now time.Time, phaseLearnings []colony.PhaseLearning) float64 {
	timestamps := make([]string, 0, len(phaseLearnings))
	for _, phaseLearning := range phaseLearnings {
		timestamps = append(timestamps, phaseLearning.Timestamp)
	}
	return latestFreshnessScore(now, 0.70, timestamps...)
}

func phaseLearningPhases(phaseLearnings []colony.PhaseLearning) []int {
	phases := make([]int, 0, len(phaseLearnings))
	for _, phaseLearning := range phaseLearnings {
		phases = append(phases, phaseLearning.Phase)
	}
	return phases
}

func phaseLearningConfidenceScore(phaseLearnings []colony.PhaseLearning) float64 {
	if len(phaseLearnings) == 0 {
		return 0.4
	}
	total := 0.0
	for _, phaseLearning := range phaseLearnings {
		total += confidenceScoreFromLearnings(phaseLearning.Learnings)
	}
	return clampScoreUnit(total / float64(len(phaseLearnings)))
}

func hiveFreshnessScore(now time.Time, entries []hiveWisdomEntry) float64 {
	timestamps := make([]string, 0, len(entries)*2)
	for _, entry := range entries {
		timestamps = append(timestamps, entry.AccessedAt, entry.CreatedAt)
	}
	return latestFreshnessScore(now, 0.50, timestamps...)
}

func latestInstinctFreshness(now time.Time, entries []colony.InstinctEntry, legacy []struct {
	trigger    string
	action     string
	confidence float64
}) float64 {
	timestamps := make([]string, 0, len(entries)*2)
	for _, entry := range entries {
		timestamps = append(timestamps, entry.Provenance.CreatedAt)
		if entry.Provenance.LastApplied != nil {
			timestamps = append(timestamps, *entry.Provenance.LastApplied)
		}
	}
	if len(timestamps) > 0 {
		return latestFreshnessScore(now, 0.70, timestamps...)
	}
	if len(legacy) > 0 {
		return 0.70
	}
	return 0.40
}

func instinctConfidenceScore(entries []colony.InstinctEntry, legacy []struct {
	trigger    string
	action     string
	confidence float64
}) float64 {
	if len(entries) > 0 {
		return confidenceScoreFromInstinctEntries(entries)
	}
	return confidenceScoreFromLegacyInstincts(legacy)
}
