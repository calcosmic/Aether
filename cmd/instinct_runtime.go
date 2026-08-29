package cmd

import (
	"fmt"
	"sort"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/memory"
	"github.com/calcosmic/Aether/pkg/storage"
)

func loadActiveInstinctEntriesFromStore(s *storage.Store) ([]colony.InstinctEntry, error) {
	if s == nil {
		return nil, fmt.Errorf("no store initialized")
	}

	var file colony.InstinctsFile
	if err := s.LoadJSON("instincts.json", &file); err != nil {
		return nil, err
	}

	active := make([]colony.InstinctEntry, 0, len(file.Instincts))
	for _, inst := range file.Instincts {
		if inst.Archived {
			continue
		}
		active = append(active, inst)
	}
	return active, nil
}

func loadRuntimeInstincts(s *storage.Store, state *colony.ColonyState) []colony.Instinct {
	if entries, err := loadActiveInstinctEntriesFromStore(s); err == nil {
		instincts := make([]colony.Instinct, 0, len(entries))
		for _, entry := range entries {
			instincts = append(instincts, instinctEntryToLegacy(entry))
		}
		return instincts
	}

	if state == nil || state.Memory.Instincts == nil {
		return []colony.Instinct{}
	}

	instincts := make([]colony.Instinct, len(state.Memory.Instincts))
	copy(instincts, state.Memory.Instincts)
	return instincts
}

func instinctEntryToLegacy(entry colony.InstinctEntry) colony.Instinct {
	evidence := []string{}
	if entry.Provenance.Evidence != "" {
		evidence = []string{entry.Provenance.Evidence}
	}

	applications, successes, failures := instinctApplicationStats(entry)
	if applications < entry.Provenance.ApplicationCount {
		applications = entry.Provenance.ApplicationCount
	}
	lastApplied := entry.Provenance.LastApplied
	if lastApplied == nil {
		if ts := memory.SummarizeInstinctApplications(entry).LastApplied; ts != "" {
			lastApplied = &ts
		}
	}

	status := "active"
	if entry.Archived {
		status = "archived"
	}

	return colony.Instinct{
		ID:           entry.ID,
		Trigger:      entry.Trigger,
		Action:       entry.Action,
		Confidence:   entry.Confidence,
		Status:       status,
		Domain:       entry.Domain,
		Source:       entry.Provenance.Source,
		Evidence:     evidence,
		Tested:       applications > 0,
		CreatedAt:    entry.Provenance.CreatedAt,
		LastApplied:  lastApplied,
		Applications: applications,
		Successes:    successes,
		Failures:     failures,
	}
}

func instinctApplicationStats(entry colony.InstinctEntry) (applications, successes, failures int) {
	summary := memory.SummarizeInstinctApplications(entry)
	return summary.Applications, summary.Successes, summary.Failures
}

func loadInstinctFileOrEmpty(s *storage.Store) colony.InstinctsFile {
	file := colony.InstinctsFile{Version: "1.0", Instincts: []colony.InstinctEntry{}}
	if s == nil {
		return file
	}
	if err := s.LoadJSON("instincts.json", &file); err != nil {
		return file
	}
	if file.Version == "" {
		file.Version = "1.0"
	}
	if file.Instincts == nil {
		file.Instincts = []colony.InstinctEntry{}
	}
	return file
}

func activeInstinctCount(s *storage.Store, state *colony.ColonyState) int {
	return len(loadRuntimeInstincts(s, state))
}

func sortedActiveInstinctEntries(file colony.InstinctsFile) []colony.InstinctEntry {
	active := make([]colony.InstinctEntry, 0, len(file.Instincts))
	for _, inst := range file.Instincts {
		if inst.Archived {
			continue
		}
		active = append(active, inst)
	}

	sort.Slice(active, func(i, j int) bool {
		ti, _ := time.Parse(time.RFC3339, active[i].Provenance.CreatedAt)
		tj, _ := time.Parse(time.RFC3339, active[j].Provenance.CreatedAt)
		if ti.Equal(tj) {
			return active[i].ID < active[j].ID
		}
		return ti.Before(tj)
	})
	return active
}

// loadStrongestRuntimeInstincts returns the standalone instincts.json
// instincts ranked by memory.InstinctUsefulnessScore (trust/confidence,
// freshness, applications, success rate) -- strength, not recency.
//
// The one exception is the branch below: when instincts.json holds nothing
// (a colony still on the legacy in-state store), it falls back to
// state.Memory.Instincts sorted by CreatedAt descending. That branch is
// genuinely recency-ordered -- it is the only recency-ordered path this
// function has -- because the legacy Instinct type carries no trust score,
// confidence, or application history to rank on.
func loadStrongestRuntimeInstincts(s *storage.Store, state *colony.ColonyState, limit int) []colony.Instinct {
	if limit <= 0 {
		return []colony.Instinct{}
	}

	file := loadInstinctFileOrEmpty(s)
	if ranked := rankedInstinctEntries(file, time.Now().UTC(), limit); len(ranked) > 0 {
		out := make([]colony.Instinct, 0, len(ranked))
		for _, entry := range ranked {
			out = append(out, instinctEntryToLegacy(entry))
		}
		return out
	}

	if state == nil || len(state.Memory.Instincts) == 0 {
		return []colony.Instinct{}
	}

	// Legacy fallback -- recency-ordered, the only such path in this
	// function (see doc comment above).
	sorted := make([]colony.Instinct, len(state.Memory.Instincts))
	copy(sorted, state.Memory.Instincts)
	sort.Slice(sorted, func(i, j int) bool {
		ti, _ := time.Parse(time.RFC3339, sorted[i].CreatedAt)
		tj, _ := time.Parse(time.RFC3339, sorted[j].CreatedAt)
		if ti.Equal(tj) {
			return sorted[i].ID < sorted[j].ID
		}
		return ti.After(tj)
	})
	if limit > len(sorted) {
		limit = len(sorted)
	}
	return sorted[:limit]
}

func recentInstinctEntries(file colony.InstinctsFile, limit int) []colony.InstinctEntry {
	active := sortedActiveInstinctEntries(file)
	if len(active) == 0 {
		return []colony.InstinctEntry{}
	}
	if limit > len(active) {
		limit = len(active)
	}
	recent := make([]colony.InstinctEntry, 0, limit)
	for i := len(active) - 1; i >= 0 && len(recent) < limit; i-- {
		recent = append(recent, active[i])
	}
	return recent
}

func rankedInstinctEntries(file colony.InstinctsFile, now time.Time, limit int) []colony.InstinctEntry {
	active := make([]colony.InstinctEntry, 0, len(file.Instincts))
	for _, inst := range file.Instincts {
		if inst.Archived {
			continue
		}
		active = append(active, inst)
	}
	sort.Slice(active, func(i, j int) bool {
		leftScore := memory.InstinctUsefulnessScore(active[i], now)
		rightScore := memory.InstinctUsefulnessScore(active[j], now)
		if leftScore != rightScore {
			return leftScore > rightScore
		}
		leftTime := parseInstinctTimestamp(memory.InstinctReferenceTimestamp(active[i]))
		rightTime := parseInstinctTimestamp(memory.InstinctReferenceTimestamp(active[j]))
		if !leftTime.Equal(rightTime) {
			return leftTime.After(rightTime)
		}
		return active[i].ID < active[j].ID
	})
	if limit > len(active) {
		limit = len(active)
	}
	return active[:limit]
}

func parseInstinctTimestamp(ts string) time.Time {
	parsed, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		return time.Time{}
	}
	return parsed
}
