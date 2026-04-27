package cmd

import (
	"fmt"
	"sort"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/storage"
	"github.com/spf13/cobra"
)

var shelfDetectCmd = &cobra.Command{
	Use:   "shelf-detect",
	Short: "Detect shelf candidates from colony state",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		var state colony.ColonyState
		if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
			state = colony.ColonyState{}
		}

		candidates, err := detectShelfCandidates(state, store)
		if err != nil {
			outputError(1, fmt.Sprintf("detection failed: %v", err), nil)
			return nil
		}

		outputOK(map[string]interface{}{
			"candidates": candidates,
			"count":      len(candidates),
			"summary":    shelfCandidateSummary(candidates),
		})
		return nil
	},
}

func detectShelfCandidates(state colony.ColonyState, s *storage.Store) ([]colony.ShelfEntry, error) {
	var all []colony.ShelfEntry

	all = append(all, detectExpiredFocusPheromones(s)...)
	all = append(all, detectLowConfidenceInstincts(state)...)
	all = append(all, detectUnresolvedFlags(s)...)
	all = append(all, detectRecurringRedirects(s)...)

	// Deduplicate by text content
	seen := make(map[string]bool)
	var deduped []colony.ShelfEntry
	for _, e := range all {
		if seen[e.Text] {
			continue
		}
		seen[e.Text] = true
		deduped = append(deduped, e)
	}

	// Sort by CreatedAt descending
	sort.Slice(deduped, func(i, j int) bool {
		return deduped[i].CreatedAt > deduped[j].CreatedAt
	})

	return deduped, nil
}

func detectExpiredFocusPheromones(s *storage.Store) []colony.ShelfEntry {
	var pf colony.PheromoneFile
	if err := s.LoadJSON("pheromones.json", &pf); err != nil {
		return nil
	}

	var entries []colony.ShelfEntry
	for _, sig := range pf.Signals {
		if sig.Type != "FOCUS" || sig.Active {
			continue
		}
		if sig.ArchivedAt == nil || *sig.ArchivedAt == "" {
			continue
		}
		// Check no phase completion after signal creation
		var state colony.ColonyState
		if err := s.LoadJSON("COLONY_STATE.json", &state); err == nil {
			if phaseCompletionsSince(&state, sig.CreatedAt) > 0 {
				continue
			}
		}

		text := extractText(sig.Content)
		entries = append(entries, colony.ShelfEntry{
			ID:           generateShelfID(),
			Text:         text,
			Source:       "colony",
			CreatedAt:    sig.CreatedAt,
			Category:     colony.ShelfCategoryPheromone,
			AutoDetected: true,
			Confidence:   0.6,
			Tags:         []string{"expired-focus", "pheromone"},
			Status:       colony.ShelfShelved,
		})
	}
	return entries
}

func detectLowConfidenceInstincts(state colony.ColonyState) []colony.ShelfEntry {
	var entries []colony.ShelfEntry
	for _, inst := range state.Memory.Instincts {
		if inst.Confidence < 0.5 || inst.Confidence >= 0.8 {
			continue
		}
		entries = append(entries, colony.ShelfEntry{
			ID:           generateShelfID(),
			Text:         fmt.Sprintf("%s: %s", inst.Trigger, inst.Action),
			Source:       "colony",
			CreatedAt:    inst.CreatedAt,
			Category:     colony.ShelfCategoryInstinct,
			AutoDetected: true,
			Confidence:   inst.Confidence,
			Tags:         []string{"near-miss", "instinct"},
			Status:       colony.ShelfShelved,
		})
	}
	return entries
}

func detectUnresolvedFlags(s *storage.Store) []colony.ShelfEntry {
	var ff colony.FlagsFile
	if err := s.LoadJSON("pending-decisions.json", &ff); err != nil {
		if err2 := s.LoadJSON("flags.json", &ff); err2 != nil {
			return nil
		}
	}

	var entries []colony.ShelfEntry
	for _, f := range ff.Decisions {
		if f.Resolved {
			continue
		}
		entries = append(entries, colony.ShelfEntry{
			ID:           generateShelfID(),
			Text:         f.Description,
			Source:       "user",
			CreatedAt:    f.CreatedAt,
			Category:     colony.ShelfCategoryUserNote,
			AutoDetected: true,
			Confidence:   0.55,
			Tags:         []string{"unresolved", "flag"},
			Status:       colony.ShelfShelved,
		})
	}
	return entries
}

func detectRecurringRedirects(s *storage.Store) []colony.ShelfEntry {
	var pf colony.PheromoneFile
	if err := s.LoadJSON("pheromones.json", &pf); err != nil {
		return nil
	}

	// Group REDIRECT signals by ContentHash
	byHash := make(map[string][]colony.PheromoneSignal)
	for _, sig := range pf.Signals {
		if sig.Type != "REDIRECT" || sig.ContentHash == nil || *sig.ContentHash == "" {
			continue
		}
		byHash[*sig.ContentHash] = append(byHash[*sig.ContentHash], sig)
	}

	var entries []colony.ShelfEntry
	for hash, signals := range byHash {
		if len(signals) < 2 {
			continue
		}
		// Check if they span different phases
		phases := make(map[int]bool)
		for _, sig := range signals {
			if sig.SourcePhase != nil {
				phases[*sig.SourcePhase] = true
			}
		}
		if len(phases) < 2 {
			continue
		}

		// Use most recent signal's text
		var latest colony.PheromoneSignal
		for _, sig := range signals {
			if sig.CreatedAt > latest.CreatedAt {
				latest = sig
			}
		}

		entries = append(entries, colony.ShelfEntry{
			ID:           generateShelfID(),
			Text:         extractText(latest.Content),
			Source:       "colony",
			CreatedAt:    latest.CreatedAt,
			Category:     colony.ShelfCategoryRedirect,
			AutoDetected: true,
			Confidence:   0.9,
			Tags:         []string{"recurring", "redirect", "permanent-guidance"},
			Status:       colony.ShelfShelved,
		})
		_ = hash // hash used as map key
	}
	return entries
}

func shelfCandidateSummary(candidates []colony.ShelfEntry) string {
	counts := make(map[colony.ShelfCategory]int)
	for _, c := range candidates {
		counts[c.Category]++
	}
	return fmt.Sprintf("%d shelf candidates: %d instincts, %d pheromones, %d flags, %d redirects",
		len(candidates),
		counts[colony.ShelfCategoryInstinct],
		counts[colony.ShelfCategoryPheromone],
		counts[colony.ShelfCategoryUserNote],
		counts[colony.ShelfCategoryRedirect])
}

func init() {
	rootCmd.AddCommand(shelfDetectCmd)
}
