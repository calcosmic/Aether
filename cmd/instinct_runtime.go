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

// rankedInstinctEntries ranks active instinct entries by
// memory.InstinctUsefulnessScore, highest first. This is the ONE ranking
// rule display code may use to pick a top-N instinct list (198.2 plan 03,
// TestStrongestHabitsHaveOneRankingRule) -- loadStrongestRuntimeInstincts is
// the only caller. Ties (equal score, which happens whenever two entries
// share confidence, trust and freshness) fall through to a strict, fully
// deterministic order: most-recently-referenced first, then instinct ID
// ascending as the final tiebreaker. Because IDs are unique, this comparator
// is a total order, so two calls over unchanged data always return the same
// sequence -- there is nothing left for a run of the process to vary.
//
// A prior version of this file also carried recentInstinctEntries, a
// recency-ordered second selector never wired to any renderer. Dead code
// that already had the shape of "select a top-N instinct list" was itself
// the landmine this test guards against, so it was removed rather than kept
// unused (198.2 plan 03).
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
