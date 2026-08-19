package cmd

import (
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/memory"
	"github.com/calcosmic/Aether/pkg/storage"
)

type memoryHealthSummary struct {
	WisdomTotal         int
	PendingPromotions   int
	ActiveInstincts     int
	ArchivedInstincts   int
	AppliedInstincts    int
	ReviewCandidates    int
	RereadCandidates    int
	RecentFailures      int
	LastLearning        string
	LastFailure         string
	LastInstinctTouched string
}

func loadMemoryHealthSummary(s *storage.Store) memoryHealthSummary {
	summary := memoryHealthSummary{}
	if s == nil {
		return summary
	}

	var instincts colony.InstinctsFile
	_ = s.LoadJSON("instincts.json", &instincts)
	promotedSources := make(map[string]struct{}, len(instincts.Instincts))
	now := time.Now().UTC()
	for _, inst := range instincts.Instincts {
		if inst.Archived {
			summary.ArchivedInstincts++
			continue
		}
		summary.ActiveInstincts++
		apps := memory.SummarizeInstinctApplications(inst)
		if apps.Applications > 0 {
			summary.AppliedInstincts++
		}
		if memory.InstinctNeedsReview(inst, now) {
			summary.ReviewCandidates++
		}
		if memory.InstinctNeedsReread(inst, now) {
			summary.RereadCandidates++
		}
		if inst.Provenance.Source != "" {
			promotedSources[inst.Provenance.Source] = struct{}{}
		}
		summary.LastInstinctTouched = latestMemoryHealthTimestamp(summary.LastInstinctTouched, memory.InstinctReferenceTimestamp(inst))
	}

	var learnings colony.LearningFile
	if err := s.LoadJSON("learning-observations.json", &learnings); err == nil {
		summary.WisdomTotal = len(learnings.Observations)
		for _, obs := range learnings.Observations {
			summary.LastLearning = latestMemoryHealthTimestamp(summary.LastLearning, obs.LastSeen)
			if _, promoted := promotedSources[obs.ContentHash]; promoted {
				continue
			}
			eligible, _ := memory.CheckPromotion(obs)
			if !eligible {
				continue
			}
			summary.PendingPromotions++
			if memoryHealthDaysSince(obs.LastSeen) >= 30 {
				summary.RereadCandidates++
			}
		}
	}

	if midden, err := loadMiddenFile(s); err == nil {
		summary.RecentFailures = len(midden.Entries)
		for _, entry := range midden.Entries {
			summary.LastFailure = latestMemoryHealthTimestamp(summary.LastFailure, entry.Timestamp)
		}
	}

	return summary
}

func latestMemoryHealthTimestamp(current, candidate string) string {
	if candidate == "" {
		return current
	}
	if current == "" {
		return candidate
	}
	currentParsed, err := time.Parse(time.RFC3339, current)
	if err != nil {
		return candidate
	}
	candidateParsed, err := time.Parse(time.RFC3339, candidate)
	if err != nil {
		return current
	}
	if candidateParsed.After(currentParsed) {
		return candidate
	}
	return current
}

func memoryHealthDaysSince(timestamp string) int {
	if timestamp == "" {
		return 0
	}
	parsed, err := time.Parse(time.RFC3339, timestamp)
	if err != nil {
		return 0
	}
	days := int(time.Since(parsed).Hours() / 24)
	if days < 0 {
		return 0
	}
	return days
}
