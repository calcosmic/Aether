package cmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/storage"
	"github.com/spf13/cobra"
)

// shelfTodoPrefix is the prefix every shelf-derived todo carries in
// session.json's active_todos, so downstream code can distinguish shelf
// entries from phase-derived todos without a separate field (Phase 165 gap
// CR-01).
const shelfTodoPrefix = "[shelf:"

var shelfPromoteBatchCmd = &cobra.Command{
	Use:   "shelf-promote-batch",
	Short: "Promote multiple shelf entries by ID",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		idsRaw, _ := cmd.Flags().GetString("ids")
		colonyGoal, _ := cmd.Flags().GetString("colony")
		if idsRaw == "" {
			outputError(1, "flag --ids is required", nil)
			return nil
		}
		if colonyGoal == "" {
			outputError(1, "flag --colony is required", nil)
			return nil
		}

		ids := splitShelfIDs(idsRaw)
		var promoted []string
		var failed []string
		for _, id := range ids {
			if err := promoteShelfEntry(store, id, colonyGoal); err != nil {
				failed = append(failed, id)
			} else {
				promoted = append(promoted, id)
			}
		}

		todos := promotedShelfTodos(store, colonyGoal)

		outputOK(map[string]interface{}{
			"promoted": promoted,
			"failed":   failed,
			"count":    len(promoted),
			"todos":    todos,
		})
		return nil
	},
}

var shelfDismissBatchCmd = &cobra.Command{
	Use:   "shelf-dismiss-batch",
	Short: "Dismiss multiple shelf entries by ID",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		idsRaw, _ := cmd.Flags().GetString("ids")
		if idsRaw == "" {
			outputError(1, "flag --ids is required", nil)
			return nil
		}

		ids := splitShelfIDs(idsRaw)
		var dismissed []string
		var failed []string
		for _, id := range ids {
			if err := dismissShelfEntry(store, id); err != nil {
				failed = append(failed, id)
			} else {
				dismissed = append(dismissed, id)
			}
		}

		outputOK(map[string]interface{}{
			"dismissed": dismissed,
			"failed":    failed,
			"count":     len(dismissed),
		})
		return nil
	},
}

// splitShelfIDs splits a comma-separated raw ID list, trims each element, and
// drops empties. It always returns a non-nil slice, so callers can range over
// the result unconditionally. This is the single definition for ID parsing
// shared by shelf-promote-batch, shelf-dismiss-batch, and `aether init`'s
// --promote-shelf / --dismiss-shelf flags (Phase 165 gap CR-01).
func splitShelfIDs(raw string) []string {
	ids := []string{}
	for _, id := range strings.Split(raw, ",") {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		ids = append(ids, id)
	}
	return ids
}

// applyInitShelfSelections promotes and dismisses shelf entries by ID as part
// of the `aether init` transaction, under colonyGoal -- the same goal string
// the colony is created with. It never returns an error: a bad shelf ID must
// never be able to fail colony creation, the same principle promotedShelfTodos
// already applies to an unreadable shelf. Call it strictly after
// COLONY_STATE.json has been saved and strictly before the session is built,
// so every refusal branch in `aether init` returns before this ever runs
// (Phase 165 gap CR-01, threat T-165-09-01).
func applyInitShelfSelections(s *storage.Store, promoteRaw, dismissRaw, colonyGoal string) (promoted []string, dismissed []string, failed []string) {
	promoted = []string{}
	dismissed = []string{}
	failed = []string{}
	if s == nil {
		return promoted, dismissed, failed
	}
	if strings.TrimSpace(promoteRaw) == "" && strings.TrimSpace(dismissRaw) == "" {
		return promoted, dismissed, failed
	}
	for _, id := range splitShelfIDs(promoteRaw) {
		if err := promoteShelfEntry(s, id, colonyGoal); err != nil {
			failed = append(failed, id)
		} else {
			promoted = append(promoted, id)
		}
	}
	for _, id := range splitShelfIDs(dismissRaw) {
		if err := dismissShelfEntry(s, id); err != nil {
			failed = append(failed, id)
		} else {
			dismissed = append(dismissed, id)
		}
	}
	return promoted, dismissed, failed
}

func loadActiveShelf(s *storage.Store) ([]colony.ShelfEntry, error) {
	sf, err := readShelfFile(s)
	if err != nil {
		return nil, err
	}
	var active []colony.ShelfEntry
	for _, e := range sf.Entries {
		if e.Status == colony.ShelfShelved {
			active = append(active, e)
		}
	}
	sort.Slice(active, func(i, j int) bool {
		return active[i].CreatedAt > active[j].CreatedAt
	})
	return active, nil
}

func promoteShelfEntry(s *storage.Store, id string, colonyGoal string) error {
	sf, err := readShelfFile(s)
	if err != nil {
		return err
	}
	found := false
	for i := range sf.Entries {
		if sf.Entries[i].ID == id {
			sf.Entries[i].Status = colony.ShelfPromoted
			// Trim to match the query-side trim in promotedShelfTodos
			// (review WR-01): both sides of the e.PromotedTo == goal
			// comparison must be normalized the same way.
			sf.Entries[i].PromotedTo = strings.TrimSpace(colonyGoal)
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("entry %q not found", id)
	}
	return writeShelfFile(s, sf)
}

func dismissShelfEntry(s *storage.Store, id string) error {
	sf, err := readShelfFile(s)
	if err != nil {
		return err
	}
	found := false
	for i := range sf.Entries {
		if sf.Entries[i].ID == id {
			sf.Entries[i].Status = colony.ShelfDismissed
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("entry %q not found", id)
	}
	return writeShelfFile(s, sf)
}

func shelfEntryToTodo(entry colony.ShelfEntry) string {
	return fmt.Sprintf("%s%s] %s", shelfTodoPrefix, entry.Category, entry.Text)
}

// promotedShelfTodos returns the shelf-derived todo strings for every entry
// promoted to colonyGoal, newest-first. It never returns nil and never fails
// colony creation: a missing or unreadable shelf yields an empty slice
// (Phase 165 gap CR-01, threat T-165-07-05).
func promotedShelfTodos(s *storage.Store, colonyGoal string) []string {
	todos := []string{}
	if s == nil {
		return todos
	}
	sf, err := readShelfFile(s)
	if err != nil {
		return todos
	}
	goal := strings.TrimSpace(colonyGoal)
	var promoted []colony.ShelfEntry
	for _, e := range sf.Entries {
		if e.Status == colony.ShelfPromoted && e.PromotedTo == goal {
			promoted = append(promoted, e)
		}
	}
	sort.Slice(promoted, func(i, j int) bool {
		return promoted[i].CreatedAt > promoted[j].CreatedAt
	})
	for _, e := range promoted {
		todos = append(todos, shelfEntryToTodo(e))
	}
	return todos
}

// mergeShelfTodos combines the shelf-prefixed entries already present in a
// session's active_todos with a freshly derived todo list, so a session
// refresh (which recomputes phase-derived todos from colony state) never
// erases a shelf-seeded todo. Shelf entries from existing come first (deduped,
// original order), then any derived entries not already present. Never
// returns nil (Phase 165 gap CR-01).
func mergeShelfTodos(existing, derived []string) []string {
	merged := []string{}
	seen := make(map[string]bool)
	for _, e := range existing {
		if !strings.HasPrefix(e, shelfTodoPrefix) {
			continue
		}
		if seen[e] {
			continue
		}
		seen[e] = true
		merged = append(merged, e)
	}
	for _, d := range derived {
		if seen[d] {
			continue
		}
		seen[d] = true
		merged = append(merged, d)
	}
	return merged
}

func formatShelfForInit(entries []colony.ShelfEntry) string {
	if len(entries) == 0 {
		return "No shelf entries."
	}

	// Group by category
	byCat := make(map[colony.ShelfCategory][]colony.ShelfEntry)
	for _, e := range entries {
		byCat[e.Category] = append(byCat[e.Category], e)
	}

	var b strings.Builder
	n := 1
	for _, cat := range []colony.ShelfCategory{
		colony.ShelfCategoryRedirect,
		colony.ShelfCategoryInstinct,
		colony.ShelfCategoryPheromone,
		colony.ShelfCategoryUserNote,
	} {
		group := byCat[cat]
		if len(group) == 0 {
			continue
		}
		for _, e := range group {
			phase := e.SourcePhase
			if phase == 0 {
				phaseStr := "unknown"
				_ = phaseStr
				b.WriteString(fmt.Sprintf("%d. [%s] %s\n", n, cat, e.Text))
			} else {
				b.WriteString(fmt.Sprintf("%d. [%s] %s (from phase %d)\n", n, cat, e.Text, phase))
			}
			n++
		}
	}
	return b.String()
}

func init() {
	shelfPromoteBatchCmd.Flags().String("ids", "", "Comma-separated entry IDs (required)")
	shelfPromoteBatchCmd.Flags().String("colony", "", "Colony goal for promotion (required)")
	shelfDismissBatchCmd.Flags().String("ids", "", "Comma-separated entry IDs (required)")

	rootCmd.AddCommand(shelfPromoteBatchCmd)
	rootCmd.AddCommand(shelfDismissBatchCmd)
}
