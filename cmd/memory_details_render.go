package cmd

import (
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/memory"
	"github.com/calcosmic/Aether/pkg/storage"
)

// memoryDetails is the assembled content behind the `memory-details` drill-down:
// the actual wisdom sentences, the lessons waiting on (or set aside from)
// promotion, the recent failures, and the standing instinct counts -- not the
// bare-count summary memory-metrics returns.
type memoryDetails struct {
	Wisdom         []queenCategoryEntries
	Pending        []pendingPromotion
	Deferred       []pendingPromotion
	Failures       []colony.MiddenEntry
	Instincts      memoryHealthSummary
	QueenMDUpdated string
	QueenPath      string
}

// queenCategoryEntries groups QUEEN.md bullet entries under the section
// heading they were written into.
type queenCategoryEntries struct {
	Category string
	Entries  []string
}

// pendingPromotion is one learning-observations.json entry that has not yet
// become an instinct, carrying its own sentence and how many times it has
// been observed.
type pendingPromotion struct {
	Content          string
	ObservationCount int
	LastSeen         string
	TrustScore       float64
}

// readQueenEntriesByCategory walks a QUEEN.md file line by line, tracking the
// current "## " heading and collecting every "- " bullet beneath it.
//
// This is deliberately a SECOND, category-preserving reader rather than a
// change to readQUEENMd (cmd/context.go), which flattens the whole file into
// a single map[string]string and loses which section a sentence came from.
// Five other callers depend on that flattened shape; the drill-down needs the
// section a sentence lives under, so it gets its own reader instead of
// widening readQUEENMd's contract underneath its existing callers.
func readQueenEntriesByCategory(path string) []queenCategoryEntries {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}

	var result []queenCategoryEntries
	var current *queenCategoryEntries

	for _, line := range strings.Split(string(data), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "## ") {
			name := mapLegacyQueenSection(strings.TrimPrefix(trimmed, "## "))
			result = append(result, queenCategoryEntries{Category: name})
			current = &result[len(result)-1]
			continue
		}
		if current == nil || !strings.HasPrefix(trimmed, "- ") {
			continue
		}
		entry := strings.TrimSpace(strings.TrimPrefix(trimmed, "- "))
		if entry != "" {
			current.Entries = append(current.Entries, entry)
		}
	}

	filtered := make([]queenCategoryEntries, 0, len(result))
	for _, cat := range result {
		if len(cat.Entries) > 0 {
			filtered = append(filtered, cat)
		}
	}
	return filtered
}

// queenLastEvolvedPattern extracts the last_evolved value from the JSON
// payload inside QUEEN.md's "<!-- METADATA ... -->" comment block.
var queenLastEvolvedPattern = regexp.MustCompile(`"last_evolved"\s*:\s*"([^"]+)"`)

// queenMarkdownUpdatedAt returns the Queen file's recorded last-evolved
// timestamp when the file carries one, otherwise the file's own modification
// time (RFC3339 UTC), otherwise the empty string.
func queenMarkdownUpdatedAt(path string) string {
	if data, err := os.ReadFile(path); err == nil {
		if m := queenLastEvolvedPattern.FindStringSubmatch(string(data)); len(m) == 2 {
			if ts := strings.TrimSpace(m[1]); ts != "" {
				return ts
			}
		}
	}
	if info, err := os.Stat(path); err == nil {
		return info.ModTime().UTC().Format(time.RFC3339)
	}
	return ""
}

// buildMemoryDetails assembles the drill-down's real content: QUEEN.md
// wisdom by section, pending and deferred learning-observations.json
// entries, the five most recent unacknowledged midden failures, and the
// standing instinct counts loadMemoryHealthSummary already computes.
func buildMemoryDetails(s *storage.Store) memoryDetails {
	d := memoryDetails{}
	if s == nil {
		return d
	}

	d.Instincts = loadMemoryHealthSummary(s)

	if qPath := localQueenPath(); qPath != "" {
		d.QueenPath = qPath
		d.Wisdom = readQueenEntriesByCategory(qPath)
		d.QueenMDUpdated = queenMarkdownUpdatedAt(qPath)
	}

	var instincts colony.InstinctsFile
	_ = s.LoadJSON("instincts.json", &instincts)
	promotedSources := make(map[string]struct{}, len(instincts.Instincts))
	for _, inst := range instincts.Instincts {
		if inst.Provenance.Source != "" {
			promotedSources[inst.Provenance.Source] = struct{}{}
		}
	}

	var learnings colony.LearningFile
	if err := s.LoadJSON("learning-observations.json", &learnings); err == nil {
		for _, obs := range learnings.Observations {
			entry := pendingPromotionFromObservation(obs)
			if obs.SourceType == "deferred" {
				d.Deferred = append(d.Deferred, entry)
				continue
			}
			if _, promoted := promotedSources[obs.ContentHash]; promoted {
				continue
			}
			eligible, _ := memory.CheckPromotion(obs)
			if !eligible {
				continue
			}
			d.Pending = append(d.Pending, entry)
		}
	}

	if midden, err := loadMiddenFile(s); err == nil {
		// Same unacknowledged-only, newest-first, cap-5 shape the
		// colony-prime capsule and composeBuildManifestBrief's own
		// "Recent Failures" section already use (cmd/colony_prime_context.go,
		// cmd/memory_feed.go) -- one convention for "recent failures", not a
		// third divergent reading of midden.json.
		unacked := make([]colony.MiddenEntry, 0, len(midden.Entries))
		for _, entry := range midden.Entries {
			if entry.Acknowledged != nil && *entry.Acknowledged {
				continue
			}
			unacked = append(unacked, entry)
		}
		sort.SliceStable(unacked, func(i, j int) bool {
			return unacked[i].Timestamp > unacked[j].Timestamp
		})
		const recentFailuresLimit = 5
		if len(unacked) > recentFailuresLimit {
			unacked = unacked[:recentFailuresLimit]
		}
		d.Failures = unacked
	}

	return d
}

func pendingPromotionFromObservation(obs colony.Observation) pendingPromotion {
	trust := 0.0
	if obs.TrustScore != nil {
		trust = *obs.TrustScore
	}
	return pendingPromotion{
		Content:          obs.Content,
		ObservationCount: obs.ObservationCount,
		LastSeen:         obs.LastSeen,
		TrustScore:       trust,
	}
}

// renderMemoryDetailsVisual renders the drill-down for a human. Heading
// wording is owner-facing plain English (CLAUDE.md Communication Style) --
// no repo jargon ("wisdom", "midden", "colony") stands alone in a heading.
// An empty block renders one plain sentence, never a blank heading.
func renderMemoryDetailsVisual(d memoryDetails) string {
	var b strings.Builder
	b.WriteString(renderBanner(commandEmoji("memory-details"), "What The Colony Has Learned"))
	b.WriteString(visualDividerStr())

	b.WriteString("What the colony has learned\n")
	if len(d.Wisdom) == 0 {
		b.WriteString("  Nothing recorded yet.\n")
	} else {
		for _, cat := range d.Wisdom {
			fmt.Fprintf(&b, "  %s\n", cat.Category)
			for _, entry := range cat.Entries {
				fmt.Fprintf(&b, "    - %s\n", entry)
			}
		}
	}

	b.WriteString("\nLessons waiting to be promoted\n")
	if len(d.Pending) == 0 {
		b.WriteString("  Nothing waiting right now.\n")
	} else {
		for _, p := range d.Pending {
			fmt.Fprintf(&b, "  - %s (seen %d times)\n", p.Content, p.ObservationCount)
		}
	}

	b.WriteString("\nLessons put aside\n")
	if len(d.Deferred) == 0 {
		b.WriteString("  Nothing put aside right now.\n")
	} else {
		for _, p := range d.Deferred {
			fmt.Fprintf(&b, "  - %s (seen %d times)\n", p.Content, p.ObservationCount)
		}
	}

	b.WriteString("\nRecent failures\n")
	if len(d.Failures) == 0 {
		b.WriteString("  No recent failures.\n")
	} else {
		for _, f := range d.Failures {
			fmt.Fprintf(&b, "  - [%s] %s: %s\n", f.Timestamp, f.Source, f.Message)
		}
	}

	b.WriteString("\nMemory at a glance\n")
	fmt.Fprintf(&b, "  Wisdom entries: %d\n", d.Instincts.WisdomTotal)
	fmt.Fprintf(&b, "  Active instincts: %d\n", d.Instincts.ActiveInstincts)
	queenUpdated := d.QueenMDUpdated
	if queenUpdated == "" {
		queenUpdated = "unknown"
	}
	fmt.Fprintf(&b, "  Wisdom file last changed: %s\n", queenUpdated)

	return b.String()
}
